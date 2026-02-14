package guicombat

import (
	"fmt"
	"game_main/config"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/guispells"
	"game_main/gui/guisquads"
	"game_main/gui/specs"
	"game_main/gui/widgets"
	"game_main/mind/encounter"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combatservices"
	"game_main/tactical/spells"
	"game_main/tactical/squads"
	"game_main/templates"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatMode provides focused UI for turn-based squad combat.
// Uses Panel Registry for declarative panel building and type-safe widget access.
type CombatMode struct {
	framework.BaseMode // Embed common mode infrastructure

	// Dependencies (consolidated for all handlers)
	deps *CombatModeDeps

	// Managers
	logManager       *CombatLogManager
	actionHandler    *CombatActionHandler
	inputHandler     *CombatInputHandler
	combatService    *combatservices.CombatService
	encounterService *encounter.EncounterService

	// Update components (stored for refresh calls)
	squadDetailComponent *guisquads.DetailPanelComponent
	factionInfoComponent *guisquads.DetailPanelComponent
	turnOrderComponent   *widgets.TextDisplayComponent

	// Spell casting
	spellHandler *guispells.SpellCastingHandler

	// Visualization systems
	visualization *CombatVisualizationManager

	// Sub-menu controller (manages debug sub-menu visibility)
	subMenus *combatSubMenuController

	// Turn lifecycle management
	turnFlow *CombatTurnFlow

	// Spell selection panel widgets
	spellList        *widgets.CachedListWrapper
	spellDetailArea  *widgets.CachedTextAreaWrapper
	spellManaLabel   *widget.Text
	spellCastButton  *widget.Button
	selectedSpellDef *templates.SpellDefinition

	// State tracking for UI updates (GUI_PERFORMANCE_ANALYSIS.md)
	lastFactionID     ecs.EntityID
	lastSelectedSquad ecs.EntityID

}

func NewCombatMode(modeManager *framework.UIModeManager, encounterService *encounter.EncounterService) *CombatMode {
	cm := &CombatMode{
		logManager:       NewCombatLogManager(),
		encounterService: encounterService,
	}
	cm.SetModeName("combat")
	cm.SetReturnMode("exploration")
	cm.SetSelf(cm) // Enable panel registry building
	cm.ModeManager = modeManager
	return cm
}

func (cm *CombatMode) Initialize(ctx *framework.UIContext) error {
	// Create combat service before ModeBuilder
	cm.combatService = combatservices.NewCombatService(ctx.ECSManager)

	// Build UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&cm.BaseMode, framework.ModeConfig{
		ModeName:   "combat",
		ReturnMode: "exploration",
	}).Build(ctx)
	if err != nil {
		return err
	}

	// Initialize sub-menu controller before building panels (panels register with it)
	cm.subMenus = newCombatSubMenuController()

	// Build panels using registry
	if err := cm.buildPanelsFromRegistry(); err != nil {
		return err
	}

	// Build action buttons (needs callbacks, so done separately)
	cm.buildActionButtons()

	// Post-UI initialization
	combatLogArea := GetCombatLogTextArea(cm.Panels)

	// Create consolidated dependencies for handlers
	cm.deps = NewCombatModeDeps(
		ctx.ModeCoordinator.GetTacticalState(),
		cm.combatService,
		cm.encounterService,
		cm.Queries,
		combatLogArea,
		cm.logManager,
		cm.ModeManager,
	)

	// Create handlers with deps
	cm.actionHandler = NewCombatActionHandler(cm.deps)
	cm.inputHandler = NewCombatInputHandler(cm.actionHandler, cm.deps)

	// Create spell handler with its own focused deps
	spellDeps := &guispells.SpellCastingDeps{
		BattleState:      cm.deps.BattleState,
		ECSManager:       cm.deps.Queries.ECSManager,
		EncounterService: cm.deps.EncounterService,
		GameMap:          ctx.GameMap,
		PlayerPos:        ctx.PlayerData.Pos,
		AddCombatLog:     cm.deps.AddCombatLog,
		Queries:          cm.deps.Queries,
	}
	cm.spellHandler = guispells.NewSpellCastingHandler(spellDeps)
	cm.inputHandler.SetSpellHandler(cm.spellHandler)

	// Extract spell panel widget references and wire input callbacks
	cm.initializeSpellWidgetReferences()
	cm.inputHandler.SetSpellPanelCallbacks(cm.showSpellPanel, cm.hideSpellPanel)

	// Register cache invalidation callbacks (automatic, fires for both GUI and AI actions)
	cm.combatService.RegisterOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
		cm.Queries.MarkSquadDirty(attackerID)
		cm.Queries.MarkSquadDirty(defenderID)
		if result.AttackerDestroyed {
			cm.Queries.InvalidateSquad(attackerID)
		}
		if result.TargetDestroyed {
			cm.Queries.InvalidateSquad(defenderID)
		}
	})

	cm.combatService.RegisterOnMoveComplete(func(squadID ecs.EntityID) {
		cm.Queries.MarkSquadDirty(squadID)
	})

	cm.combatService.RegisterOnTurnEnd(func(round int) {
		cm.Queries.MarkAllSquadsDirty()
		cm.visualization.UpdateThreatManagers()
		cm.visualization.UpdateThreatEvaluator(round)

		// Reset spell cast flag for the new turn
		cm.deps.BattleState.HasCastSpell = false
	})

	// Initialize visualization systems
	cm.visualization = NewCombatVisualizationManager(ctx, cm.Queries, ctx.GameMap)

	// Wire visualization input support into input handler
	cm.inputHandler.SetVisualization(cm.visualization, cm.Panels, cm.logManager)

	// Initialize turn flow manager (before initializeUpdateComponents which sets UI refs on it)
	cm.turnFlow = NewCombatTurnFlow(
		cm.combatService,
		cm.visualization,
		cm.logManager,
		cm.actionHandler,
		cm.Queries,
		cm.ModeManager,
		cm.Panels,
		ctx,
	)

	cm.initializeUpdateComponents()

	return nil
}

// buildPanelsFromRegistry builds all combat panels using the Panel Registry
func (cm *CombatMode) buildPanelsFromRegistry() error {
	// Build sub-menu panels first (they register with subMenus controller)
	// Then build standard panels
	panels := []framework.PanelType{
		CombatPanelDebugMenu,
		CombatPanelSpellSelection,
		CombatPanelTurnOrder,
		CombatPanelFactionInfo,
		CombatPanelSquadDetail,
		CombatPanelLayerStatus,
	}

	// Build combat log only if enabled
	if config.ENABLE_COMBAT_LOG {
		panels = append(panels, CombatPanelCombatLog)
	}

	return cm.BuildPanels(panels...)
}

// buildActionButtons creates the action button panel (needs callbacks)
func (cm *CombatMode) buildActionButtons() {
	spacing := int(float64(cm.Layout.ScreenWidth) * specs.PaddingTight)
	bottomPad := int(float64(cm.Layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := builders.AnchorCenterEnd(bottomPad)

	buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Debug", OnClick: cm.subMenus.Toggle("debug")},
			{Text: "Attack (A)", OnClick: cm.handleAttackClick},
			{Text: "Move (M)", OnClick: cm.handleMoveClick},
			{Text: "Cast Spell (S)", OnClick: cm.handleSpellClick},
			{Text: "Undo (Ctrl+Z)", OnClick: cm.handleUndoMove},
			{Text: "Redo (Ctrl+Y)", OnClick: cm.handleRedoMove},
			{Text: "End Turn (Space)", OnClick: cm.handleEndTurnClick},
			{Text: "Flee (ESC)", OnClick: cm.handleFleeClick},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(cm.Layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})

	cm.RootContainer.AddChild(buttonContainer)
}

// Button click handlers that delegate to action handler
func (cm *CombatMode) handleAttackClick() {
	cm.actionHandler.ToggleAttackMode()
}

func (cm *CombatMode) handleMoveClick() {
	cm.actionHandler.ToggleMoveMode()
}

func (cm *CombatMode) handleSpellClick() {
	if cm.spellHandler.IsInSpellMode() {
		cm.spellHandler.CancelSpellMode()
		cm.hideSpellPanel()
		return
	}
	cm.showSpellPanel()
}

func (cm *CombatMode) handleUndoMove() {
	cm.actionHandler.UndoLastMove()
}

func (cm *CombatMode) handleRedoMove() {
	cm.actionHandler.RedoLastMove()
}

func (cm *CombatMode) handleEndTurnClick() {
	cm.turnFlow.HandleEndTurn()
}

func (cm *CombatMode) handleFleeClick() {
	cm.turnFlow.HandleFlee()
}

// --- Spell panel methods ---

// initializeSpellWidgetReferences extracts widget references from the panel registry.
func (cm *CombatMode) initializeSpellWidgetReferences() {
	cm.spellList = framework.GetPanelWidget[*widgets.CachedListWrapper](cm.Panels, CombatPanelSpellSelection, "spellList")
	cm.spellDetailArea = framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](cm.Panels, CombatPanelSpellSelection, "detailArea")
	cm.spellManaLabel = framework.GetPanelWidget[*widget.Text](cm.Panels, CombatPanelSpellSelection, "manaLabel")
	cm.spellCastButton = framework.GetPanelWidget[*widget.Button](cm.Panels, CombatPanelSpellSelection, "castButton")
}

// onSpellSelected is the list click callback — updates detail area and cast button.
func (cm *CombatMode) onSpellSelected(spell *templates.SpellDefinition) {
	cm.selectedSpellDef = spell
	cm.updateSpellDetailPanel()
}

// updateSpellDetailPanel shows spell details and checks mana affordability.
func (cm *CombatMode) updateSpellDetailPanel() {
	spell := cm.selectedSpellDef
	if spell == nil || cm.spellDetailArea == nil {
		return
	}

	currentMana, _ := cm.spellHandler.GetCommanderMana()
	canAfford := currentMana >= spell.ManaCost

	targetType := "Single Target"
	if spell.IsAoE() {
		targetType = "AoE"
	}

	detail := fmt.Sprintf("=== %s ===\nCost: %d MP\nDamage: %d\nTarget: %s\n\n%s",
		spell.Name, spell.ManaCost, spell.Damage, targetType, spell.Description)

	if !canAfford {
		detail += "\n\n[color=ff4444]Not enough mana![/color]"
	}

	cm.spellDetailArea.SetText(detail)

	if cm.spellCastButton != nil {
		cm.spellCastButton.GetWidget().Disabled = !canAfford
	}
}

// refreshSpellPanel populates the list from the spell handler, updates mana label, clears selection.
func (cm *CombatMode) refreshSpellPanel() {
	allSpells := cm.spellHandler.GetAllSpells()
	currentMana, maxMana := cm.spellHandler.GetCommanderMana()

	// Update mana label
	if cm.spellManaLabel != nil {
		cm.spellManaLabel.Label = fmt.Sprintf("Mana: %d/%d", currentMana, maxMana)
	}

	// Populate spell list
	if cm.spellList != nil {
		entries := make([]interface{}, len(allSpells))
		for i, spell := range allSpells {
			entries[i] = spell
		}
		cm.spellList.GetList().SetEntries(entries)
		cm.spellList.MarkDirty()
	}

	// Clear selection
	cm.selectedSpellDef = nil
	if cm.spellDetailArea != nil {
		cm.spellDetailArea.SetText("Select a spell to view details")
	}
	if cm.spellCastButton != nil {
		cm.spellCastButton.GetWidget().Disabled = true
	}
}

// showSpellPanel validates preconditions, refreshes data, and shows the panel.
func (cm *CombatMode) showSpellPanel() {
	// Validate: already cast this turn?
	if cm.deps.BattleState.HasCastSpell {
		cm.deps.AddCombatLog("Already cast a spell this turn")
		return
	}

	// Validate: has spells?
	allSpells := cm.spellHandler.GetAllSpells()
	if len(allSpells) == 0 {
		cm.deps.AddCombatLog("No spells available")
		return
	}

	cm.deps.BattleState.InSpellMode = true
	cm.refreshSpellPanel()
	cm.subMenus.Show("spell")
}

// hideSpellPanel hides the panel and clears selection.
func (cm *CombatMode) hideSpellPanel() {
	cm.selectedSpellDef = nil
	cm.subMenus.CloseAll()
}

// onCastButtonClicked selects the spell for targeting and hides the panel.
func (cm *CombatMode) onCastButtonClicked() {
	if cm.selectedSpellDef == nil {
		return
	}
	cm.spellHandler.SelectSpell(cm.selectedSpellDef.ID)
	cm.hideSpellPanel()
}

// onSpellCancelClicked cancels spell mode and hides the panel.
func (cm *CombatMode) onSpellCancelClicked() {
	cm.spellHandler.CancelSpellMode()
	cm.hideSpellPanel()
}

func (cm *CombatMode) initializeUpdateComponents() {
	// Get widgets from registry
	turnOrderLabel := cm.GetTextLabel(CombatPanelTurnOrder)
	factionInfoText := cm.GetTextLabel(CombatPanelFactionInfo)
	squadDetailText := cm.GetTextLabel(CombatPanelSquadDetail)

	// Turn order component - displays current faction and round
	cm.turnOrderComponent = widgets.NewTextDisplayComponent(
		turnOrderLabel,
		func() string {
			currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
			if currentFactionID == 0 {
				return "No active combat"
			}

			round := cm.combatService.TurnManager.GetCurrentRound()
			factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID)
			factionName := "Unknown"
			turnIndicator := ""

			if factionData != nil {
				factionName = factionData.Name

				if factionData.PlayerID > 0 {
					turnIndicator = fmt.Sprintf(" >>> %s's TURN <<<", factionData.PlayerName)
				} else {
					turnIndicator = " [AI TURN]"
				}
			}

			return fmt.Sprintf("Round %d | %s%s", round, factionName, turnIndicator)
		},
	)

	// Faction info component - displays squad count and commander mana
	cm.factionInfoComponent = guisquads.NewDetailPanelComponent(
		factionInfoText,
		cm.Queries,
		func(data interface{}) string {
			factionInfo := data.(*framework.FactionInfo)
			factionData := cm.Queries.CombatCache.FindFactionDataByID(factionInfo.ID)

			infoText := fmt.Sprintf("%s\n", factionInfo.Name)

			if factionData != nil && factionData.PlayerID > 0 {
				infoText += fmt.Sprintf("[%s]\n", factionData.PlayerName)
			}

			infoText += fmt.Sprintf("Squads: %d/%d\n", factionInfo.AliveSquadCount, len(factionInfo.SquadIDs))
			infoText += fmt.Sprintf("Mana: %d/%d", factionInfo.CurrentMana, factionInfo.MaxMana)

			// Show commander mana from spell system if available
			if cm.encounterService != nil {
				commanderID := cm.encounterService.GetRosterOwnerID()
				if commanderID != 0 {
					manaData := spells.GetManaData(commanderID, cm.Queries.ECSManager)
					if manaData != nil {
						infoText += fmt.Sprintf("\nSpell Mana: %d/%d", manaData.CurrentMana, manaData.MaxMana)
					}
				}
			}
			return infoText
		},
	)

	// Squad detail component - displays selected squad details
	cm.squadDetailComponent = guisquads.NewDetailPanelComponent(
		squadDetailText,
		cm.Queries,
		nil, // Use default formatter
	)

	// Pass UI component refs to turn flow manager
	cm.turnFlow.SetUIComponents(
		cm.turnOrderComponent,
		cm.factionInfoComponent,
		cm.squadDetailComponent,
	)
}


func (cm *CombatMode) Enter(fromMode framework.UIMode) error {
	isComingFromAnimation := fromMode != nil && fromMode.GetModeName() == "combat_animation"
	shouldInitialize := !isComingFromAnimation

	combatLogArea := GetCombatLogTextArea(cm.Panels)

	if shouldInitialize {
		cm.logManager.UpdateTextArea(combatLogArea, "=== COMBAT STARTED ===")

		// Refresh threat manager with newly created factions (must be after SpawnCombatEntities)
		cm.visualization.RefreshFactions(cm.Queries)

		// Start battle recording if enabled
		if config.ENABLE_COMBAT_LOG_EXPORT && cm.combatService.BattleRecorder != nil {
			cm.combatService.BattleRecorder.SetEnabled(true)
			cm.combatService.BattleRecorder.Start()
			cm.combatService.BattleRecorder.SetCurrentRound(1)
		}

		// Initialize combat factions
		factionIDs := cm.Queries.GetAllFactions()
		if len(factionIDs) > 0 {
			if err := cm.combatService.InitializeCombat(factionIDs); err != nil {
				cm.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("Error initializing combat: %v", err))
				return fmt.Errorf("error initializing combat factions: %w", err)
			}

			// Log initial faction
			currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
			factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID)
			factionName := "Unknown"
			if factionData != nil {
				factionName = factionData.Name
			}
			cm.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("Round 1: %s goes first!", factionName))
		} else {
			cm.logManager.UpdateTextArea(combatLogArea, "No factions found - combat cannot start")
		}
	}

	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	if currentFactionID != 0 {
		cm.turnOrderComponent.Refresh()
		cm.factionInfoComponent.ShowFaction(currentFactionID)
	}

	return nil
}

func (cm *CombatMode) Exit(toMode framework.UIMode) error {
	isToAnimation := toMode != nil && toMode.GetModeName() == "combat_animation"

	if !isToAnimation {
		// Get victory result (use cached if available, otherwise check now)
		victor := cm.turnFlow.GetVictoryResult()
		if victor == nil {
			victor = cm.combatService.CheckVictoryCondition()
		}

		// Determine exit reason
		reason := encounter.ExitDefeat
		if cm.turnFlow.IsFleeRequested() {
			reason = encounter.ExitFlee
		} else if victor.IsPlayerVictory {
			reason = encounter.ExitVictory
		}

		// Single call handles: overworld resolution, history recording, entity cleanup
		if cm.encounterService != nil {
			cm.encounterService.ExitCombat(reason,
				&encounter.CombatResult{
					IsPlayerVictory:  victor.IsPlayerVictory,
					VictorFaction:    victor.VictorFaction,
					VictorName:       victor.VictorName,
					RoundsCompleted:  victor.RoundsCompleted,
					DefeatedFactions: victor.DefeatedFactions,
				},
				cm.combatService)
		}

		// Export battle log if enabled (GUI-only concern, stays here)
		if config.ENABLE_COMBAT_LOG_EXPORT && cm.combatService.BattleRecorder != nil && cm.combatService.BattleRecorder.IsEnabled() {
			victoryInfo := &battlelog.VictoryInfo{
				RoundsCompleted: victor.RoundsCompleted,
				VictorFaction:   victor.VictorFaction,
				VictorName:      victor.VictorName,
			}
			record := cm.combatService.BattleRecorder.Finalize(victoryInfo)
			if err := battlelog.ExportBattleJSON(record, config.COMBAT_LOG_EXPORT_DIR); err != nil {
				fmt.Printf("Error exporting battle log: %v\n", err)
			}
			cm.combatService.BattleRecorder.Clear()
			cm.combatService.BattleRecorder.SetEnabled(false)
		}

		// Clear cached victory/flee state
		cm.turnFlow.ClearState()
	}

	cm.visualization.ClearAllVisualizations()
	cm.logManager.Clear()
	return nil
}

func (cm *CombatMode) Update(deltaTime float64) error {
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	if cm.lastFactionID != currentFactionID {
		cm.turnOrderComponent.Refresh()
		cm.lastFactionID = currentFactionID
		if cm.lastFactionID != 0 {
			cm.factionInfoComponent.ShowFaction(cm.lastFactionID)
		}
	}

	battleState := cm.Context.ModeCoordinator.GetTacticalState()
	if cm.lastSelectedSquad != battleState.SelectedSquadID {
		cm.lastSelectedSquad = battleState.SelectedSquadID
		if cm.lastSelectedSquad != 0 {
			cm.squadDetailComponent.ShowSquad(cm.lastSelectedSquad)
		}
	}

	currentRound := cm.combatService.TurnManager.GetCurrentRound()
	playerPos := *cm.Context.PlayerData.Pos
	viewportSize := 30

	cm.visualization.UpdateThreatVisualization(currentFactionID, currentRound, playerPos, viewportSize)

	// Update AoE spell targeting overlay each frame
	if cm.spellHandler != nil && cm.spellHandler.IsAoETargeting() {
		mouseX, mouseY := ebiten.CursorPosition()
		cm.spellHandler.HandleAoETargetingFrame(mouseX, mouseY)
	}

	return nil
}

func (cm *CombatMode) Render(screen *ebiten.Image) {
	playerPos := *cm.Context.PlayerData.Pos
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	battleState := cm.Context.ModeCoordinator.GetTacticalState()
	cm.visualization.RenderAll(screen, playerPos, currentFactionID, battleState, cm.combatService)
}

func (cm *CombatMode) HandleInput(inputState *framework.InputState) bool {
	if cm.HandleCommonInput(inputState) {
		return true
	}

	cm.inputHandler.SetPlayerPosition(cm.Context.PlayerData.Pos)
	cm.inputHandler.SetCurrentFactionID(cm.combatService.TurnManager.GetCurrentFaction())

	if cm.inputHandler.HandleInput(inputState) {
		return true
	}

	if inputState.KeysJustPressed[ebiten.KeySpace] {
		cm.turnFlow.HandleEndTurn()
		return true
	}

	return false
}
