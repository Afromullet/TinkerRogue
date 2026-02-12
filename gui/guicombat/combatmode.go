package guicombat

import (
	"fmt"
	"game_main/config"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/guisquads"
	"game_main/gui/specs"
	"game_main/gui/widgets"
	"game_main/mind/encounter"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combatservices"

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

	// Visualization systems
	visualization *CombatVisualizationManager

	// Turn lifecycle management
	turnFlow *CombatTurnFlow

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
		ctx.ModeCoordinator.GetBattleMapState(),
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
	// Build standard panels
	panels := []framework.PanelType{
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
			{Text: "Attack (A)", OnClick: cm.handleAttackClick},
			{Text: "Move (M)", OnClick: cm.handleMoveClick},
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
			factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID, cm.Queries.ECSManager)
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

	// Faction info component - displays squad count and mana
	cm.factionInfoComponent = guisquads.NewDetailPanelComponent(
		factionInfoText,
		cm.Queries,
		func(data interface{}) string {
			factionInfo := data.(*framework.FactionInfo)
			factionData := cm.Queries.CombatCache.FindFactionDataByID(factionInfo.ID, cm.Queries.ECSManager)

			infoText := fmt.Sprintf("%s\n", factionInfo.Name)

			if factionData != nil && factionData.PlayerID > 0 {
				infoText += fmt.Sprintf("[%s]\n", factionData.PlayerName)
			}

			infoText += fmt.Sprintf("Squads: %d/%d\n", factionInfo.AliveSquadCount, len(factionInfo.SquadIDs))
			infoText += fmt.Sprintf("Mana: %d/%d", factionInfo.CurrentMana, factionInfo.MaxMana)
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
			factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID, cm.Queries.ECSManager)
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

	battleState := cm.Context.ModeCoordinator.GetBattleMapState()
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

	return nil
}

func (cm *CombatMode) Render(screen *ebiten.Image) {
	playerPos := *cm.Context.PlayerData.Pos
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	battleState := cm.Context.ModeCoordinator.GetBattleMapState()
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
