package guicombat

import (
	"fmt"
	"game_main/config"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/guiartifacts"
	"game_main/gui/guiinspect"
	"game_main/gui/guispells"
	"game_main/gui/guisquads"
	"game_main/gui/specs"
	"game_main/gui/widgets"
	"game_main/mind/encounter"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combatservices"
	"game_main/tactical/spells"
	"game_main/tactical/squads"
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
	actionHandler    *CombatActionHandler
	inputHandler     *CombatInputHandler
	combatService    *combatservices.CombatService
	encounterService *encounter.EncounterService

	// Update components (stored for refresh calls)
	squadDetailComponent *guisquads.DetailPanelComponent
	factionInfoComponent *guisquads.DetailPanelComponent
	turnOrderComponent   *widgets.TextDisplayComponent

	// Spell panel controller (owns spell selection UI + handler)
	spellPanel *guispells.SpellPanelController

	// Artifact panel controller (owns artifact activation UI + handler)
	artifactPanel *guiartifacts.ArtifactPanelController

	// Visualization systems
	visualization *CombatVisualizationManager

	// Sub-menu controller (manages debug sub-menu visibility)
	subMenus *combatSubMenuController

	// Turn lifecycle management
	turnFlow *CombatTurnFlow

	// State tracking for UI updates (GUI_PERFORMANCE_ANALYSIS.md)
	lastFactionID     ecs.EntityID
	lastSelectedSquad ecs.EntityID

}

func NewCombatMode(modeManager *framework.UIModeManager, encounterService *encounter.EncounterService) *CombatMode {
	cm := &CombatMode{
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

	// Create consolidated dependencies for handlers
	cm.deps = NewCombatModeDeps(
		ctx.ModeCoordinator.GetTacticalState(),
		cm.combatService,
		cm.encounterService,
		cm.Queries,
		cm.ModeManager,
	)

	// Create handlers with deps
	cm.actionHandler = NewCombatActionHandler(cm.deps)
	cm.inputHandler = NewCombatInputHandler(cm.actionHandler, cm.deps)

	// Create spell handler and panel controller
	spellDeps := &guispells.SpellCastingDeps{
		BattleState:      cm.deps.BattleState,
		ECSManager:       cm.deps.Queries.ECSManager,
		EncounterService: cm.deps.EncounterService,
		GameMap:          ctx.GameMap,
		PlayerPos:        ctx.PlayerData.Pos,
		Queries:          cm.deps.Queries,
	}
	spellHandler := guispells.NewSpellCastingHandler(spellDeps)
	cm.spellPanel = guispells.NewSpellPanelController(&guispells.SpellPanelDeps{
		Handler:      spellHandler,
		BattleState:  cm.deps.BattleState,
		ShowSubmenu:  func() { cm.subMenus.Show("spell") },
		CloseSubmenu: func() { cm.subMenus.CloseAll() },
	})

	// Extract spell panel widget references
	cm.spellPanel.SetWidgets(
		framework.GetPanelWidget[*widgets.CachedListWrapper](cm.Panels, CombatPanelSpellSelection, "spellList"),
		framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](cm.Panels, CombatPanelSpellSelection, "detailArea"),
		framework.GetPanelWidget[*widget.Text](cm.Panels, CombatPanelSpellSelection, "manaLabel"),
		framework.GetPanelWidget[*widget.Button](cm.Panels, CombatPanelSpellSelection, "castButton"),
	)

	// Wire spell panel into input handler
	cm.inputHandler.SetSpellPanel(cm.spellPanel)

	// Create artifact handler and panel controller
	artifactDeps := &guiartifacts.ArtifactActivationDeps{
		BattleState:      cm.deps.BattleState,
		CombatService:    cm.deps.CombatService,
		EncounterService: cm.deps.EncounterService,
		Queries:          cm.deps.Queries,
	}
	artifactHandler := guiartifacts.NewArtifactActivationHandler(artifactDeps)
	artifactHandler.SetPlayerPosition(ctx.PlayerData.Pos)
	cm.artifactPanel = guiartifacts.NewArtifactPanelController(&guiartifacts.ArtifactPanelDeps{
		Handler:      artifactHandler,
		BattleState:  cm.deps.BattleState,
		ShowSubmenu:  func() { cm.subMenus.Show("artifact") },
		CloseSubmenu: func() { cm.subMenus.CloseAll() },
	})

	// Extract artifact panel widget references
	cm.artifactPanel.SetWidgets(
		framework.GetPanelWidget[*widgets.CachedListWrapper](cm.Panels, CombatPanelArtifactSelection, "artifactList"),
		framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](cm.Panels, CombatPanelArtifactSelection, "detailArea"),
		framework.GetPanelWidget[*widget.Button](cm.Panels, CombatPanelArtifactSelection, "activateButton"),
	)

	// Wire artifact panel into input handler
	cm.inputHandler.SetArtifactPanel(cm.artifactPanel)

	// Create inspect panel controller and wire into input handler
	inspectController := guiinspect.NewInspectPanelController(cm.Queries)
	inspectResult := cm.Panels.Get(guiinspect.InspectPanelType)
	if inspectResult != nil {
		inspectController.SetWidgets(
			framework.GetPanelWidget[*widget.Text](cm.Panels, guiinspect.InspectPanelType, "squadNameLabel"),
			framework.GetPanelWidget[[3][3]*widget.Button](cm.Panels, guiinspect.InspectPanelType, "gridCells"),
			framework.GetPanelWidget[[3][3]*widget.Button](cm.Panels, guiinspect.InspectPanelType, "attackGridCells"),
			inspectResult.Container,
		)
	}
	cm.inputHandler.SetInspectPanel(inspectController)

	// Register cache invalidation callbacks (automatic, fires for both GUI and AI actions)
	cm.registerCombatCallbacks()

	// Initialize visualization systems
	cm.visualization = NewCombatVisualizationManager(ctx, cm.Queries, ctx.GameMap)

	// Wire visualization input support into input handler
	cm.inputHandler.SetVisualization(cm.visualization, cm.Panels)

	// Initialize turn flow manager (before initializeUpdateComponents which sets UI refs on it)
	cm.turnFlow = NewCombatTurnFlow(
		cm.combatService,
		cm.visualization,
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
		CombatPanelArtifactSelection,
		guiinspect.InspectPanelType,
		CombatPanelTurnOrder,
		CombatPanelFactionInfo,
		CombatPanelSquadDetail,
		CombatPanelLayerStatus,
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
			{Text: "Artifact (D)", OnClick: cm.handleArtifactClick},
			{Text: "Inspect (I)", OnClick: cm.handleInspectClick},
			{Text: "Undo (Ctrl+Z)", OnClick: cm.handleUndoMove},
			{Text: "End Turn (Space)", OnClick: cm.handleEndTurnClick},
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
	cm.spellPanel.Toggle()
}

func (cm *CombatMode) handleArtifactClick() {
	cm.artifactPanel.Toggle()
}

func (cm *CombatMode) handleInspectClick() {
	cm.inputHandler.toggleInspectMode()
}

func (cm *CombatMode) handleUndoMove() {
	cm.actionHandler.UndoLastMove()
}

func (cm *CombatMode) handleEndTurnClick() {
	cm.turnFlow.HandleEndTurn()
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


// registerCombatCallbacks registers cache invalidation callbacks on the combat service.
// Must be called on each combat start because CleanupCombat clears all callbacks.
func (cm *CombatMode) registerCombatCallbacks() {
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

		// Close any open sub-menus (inspect, spell, artifact panels)
		cm.subMenus.CloseAll()
	})
}

func (cm *CombatMode) Enter(fromMode framework.UIMode) error {
	isComingFromAnimation := fromMode != nil && fromMode.GetModeName() == "combat_animation"
	shouldInitialize := !isComingFromAnimation

	if shouldInitialize {
		// Reset stale caches from previous combat
		cm.Queries.ClearSquadCache()
		cm.visualization.ResetHighlightColors()

		// Re-register callbacks (cleared by previous CleanupCombat)
		cm.registerCombatCallbacks()

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
				return fmt.Errorf("error initializing combat factions: %w", err)
			}
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
	if cm.spellPanel != nil && cm.spellPanel.Handler().IsAoETargeting() {
		mouseX, mouseY := ebiten.CursorPosition()
		cm.spellPanel.Handler().HandleAoETargetingFrame(mouseX, mouseY)
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
