package guicombat

import (
	"fmt"
	"game_main/config"
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/guisquads"
	"game_main/gui/specs"
	"game_main/gui/widgets"
	"game_main/mind/behavior"
	"game_main/mind/encounter"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combatservices"
	"game_main/world/coords"
	"game_main/world/worldmap"

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
	squadListComponent   *guisquads.SquadListComponent
	squadDetailComponent *guisquads.DetailPanelComponent
	factionInfoComponent *guisquads.DetailPanelComponent
	turnOrderComponent   *widgets.TextDisplayComponent

	// Visualization systems
	visualization *CombatVisualizationManager

	// State tracking for UI updates (GUI_PERFORMANCE_ANALYSIS.md)
	lastFactionID     ecs.EntityID
	lastSelectedSquad ecs.EntityID

	// Victory tracking (cached to avoid redundant checks)
	lastVictoryResult *combatservices.VictoryCheckResult

	// Flee tracking
	fleeRequested bool

	// Debug support
	debugLogger *framework.DebugLogger
}

func NewCombatMode(modeManager *framework.UIModeManager, encounterService *encounter.EncounterService) *CombatMode {
	cm := &CombatMode{
		logManager:       NewCombatLogManager(),
		debugLogger:      framework.NewDebugLogger("combat"),
		encounterService: encounterService,
	}
	cm.SetModeName("combat")
	cm.SetReturnMode("exploration")
	cm.SetSelf(cm) // Enable panel registry building
	cm.ModeManager = modeManager
	return cm
}

func (cm *CombatMode) Initialize(ctx *framework.UIContext) error {
	cm.debugLogger.Log("Initialize starting")

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

	cm.initializeUpdateComponents()

	// Initialize visualization systems
	gameMap := ctx.GameMap.(*worldmap.GameMap)
	cm.visualization = NewCombatVisualizationManager(ctx, cm.Queries, gameMap)

	cm.debugLogger.Log("Initialize complete")
	return nil
}

// buildPanelsFromRegistry builds all combat panels using the Panel Registry
func (cm *CombatMode) buildPanelsFromRegistry() error {
	// Build standard panels
	panels := []framework.PanelType{
		CombatPanelTurnOrder,
		CombatPanelFactionInfo,
		CombatPanelSquadList,
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
			{Text: "End Turn (Space)", OnClick: cm.handleEndTurn},
			{Text: "Flee (ESC)", OnClick: cm.handleFlee},
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

func (cm *CombatMode) initializeUpdateComponents() {
	// Get widgets from registry
	turnOrderLabel := cm.GetTextLabel(CombatPanelTurnOrder)
	factionInfoText := cm.GetTextLabel(CombatPanelFactionInfo)
	squadDetailText := cm.GetTextLabel(CombatPanelSquadDetail)
	squadListPanel := cm.GetPanelContainer(CombatPanelSquadList)

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

	// Squad list component - filter for current faction squads
	cm.squadListComponent = guisquads.NewSquadListComponent(
		squadListPanel,
		cm.Queries,
		cm.makeCurrentFactionSquadFilter(),
		func(squadID ecs.EntityID) {
			cm.actionHandler.SelectSquad(squadID)
			cm.squadDetailComponent.ShowSquad(squadID)
		},
	)
}

// makeCurrentFactionSquadFilter creates a filter for squads from the current faction
func (cm *CombatMode) makeCurrentFactionSquadFilter() framework.SquadFilter {
	return func(info *framework.SquadInfo) bool {
		currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
		if currentFactionID == 0 {
			return false
		}
		factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID, cm.Queries.ECSManager)
		if factionData == nil || !factionData.IsPlayerControlled {
			return false
		}
		return !info.IsDestroyed && info.FactionID == currentFactionID
	}
}

// formatTurnMessage creates the turn transition message for combat log
func (cm *CombatMode) formatTurnMessage(factionID ecs.EntityID, round int) string {
	factionData := cm.Queries.CombatCache.FindFactionDataByID(factionID, cm.Queries.ECSManager)
	factionName := "Unknown"

	if factionData != nil {
		factionName = factionData.Name
		if factionData.PlayerID > 0 {
			return fmt.Sprintf("=== Round %d: %s (%s) ===", round, factionName, factionData.PlayerName)
		}
		return fmt.Sprintf("=== Round %d: %s (AI) ===", round, factionName)
	}
	return fmt.Sprintf("=== Round %d: %s's Turn ===", round, factionName)
}

func (cm *CombatMode) handleFlee() {
	combatLogArea := GetCombatLogTextArea(cm.Panels)
	cm.logManager.UpdateTextArea(combatLogArea, "Fleeing from combat...")

	rounds := cm.combatService.TurnManager.GetCurrentRound()
	cm.lastVictoryResult = &combatservices.VictoryCheckResult{
		BattleOver:      true,
		IsPlayerVictory: false,
		VictorName:      "Retreat",
		RoundsCompleted: rounds,
	}
	cm.fleeRequested = true

	if exploreMode, exists := cm.ModeManager.GetMode("exploration"); exists {
		cm.ModeManager.RequestTransition(exploreMode, "Fled from combat")
	}
}

// checkAndHandleVictory checks if combat has ended and handles the transition.
// Returns true if combat ended (victory or defeat), false if combat continues.
// Caches the victory result to avoid redundant checks during cleanup.
func (cm *CombatMode) checkAndHandleVictory() bool {
	result := cm.combatService.CheckVictoryCondition()

	if !result.BattleOver {
		return false
	}

	// Cache the result for use in Exit() to avoid redundant checks
	cm.lastVictoryResult = result

	combatLogArea := GetCombatLogTextArea(cm.Panels)

	// Display victory or defeat message (uses single source of truth from CombatService)
	if result.IsPlayerVictory {
		cm.logManager.UpdateTextArea(combatLogArea, "=== VICTORY! ===")
		cm.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("%s is victorious!", result.VictorName))
	} else {
		cm.logManager.UpdateTextArea(combatLogArea, "=== DEFEAT ===")
		if result.VictorFaction != 0 {
			cm.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("%s has won the battle.", result.VictorName))
		} else {
			cm.logManager.UpdateTextArea(combatLogArea, "All forces have been eliminated.")
		}
	}

	cm.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("Battle lasted %d rounds.", result.RoundsCompleted))

	// Transition to exploration mode
	if exploreMode, exists := cm.ModeManager.GetMode("exploration"); exists {
		cm.ModeManager.RequestTransition(exploreMode, "Combat ended - "+result.VictorName+" victorious")
	}

	return true
}

func (cm *CombatMode) handleEndTurn() {
	cm.actionHandler.ClearMoveHistory()

	err := cm.combatService.TurnManager.EndTurn()
	if err != nil {
		combatLogArea := GetCombatLogTextArea(cm.Panels)
		cm.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("Error ending turn: %s", err.Error()))
		return
	}

	cm.Queries.MarkAllSquadsDirty()

	// Check for victory after player ends turn
	if cm.checkAndHandleVictory() {
		return
	}

	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	round := cm.combatService.TurnManager.GetCurrentRound()

	if cm.combatService.BattleRecorder != nil && cm.combatService.BattleRecorder.IsEnabled() {
		cm.combatService.BattleRecorder.SetCurrentRound(round)
	}

	turnMessage := cm.formatTurnMessage(currentFactionID, round)
	combatLogArea := GetCombatLogTextArea(cm.Panels)
	cm.logManager.UpdateTextArea(combatLogArea, turnMessage)

	cm.Context.ModeCoordinator.GetBattleMapState().Reset()

	cm.turnOrderComponent.Refresh()
	cm.factionInfoComponent.ShowFaction(currentFactionID)
	cm.squadListComponent.Refresh()
	cm.squadDetailComponent.SetText("Select a squad\nto view details")

	cm.visualization.UpdateThreatManagers()
	cm.visualization.UpdateThreatEvaluator(round)

	cm.executeAITurnIfNeeded()
}

// executeAITurnIfNeeded checks if current faction is AI-controlled and executes its turn
func (cm *CombatMode) executeAITurnIfNeeded() {
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return
	}

	factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID, cm.Queries.ECSManager)
	if factionData == nil || factionData.IsPlayerControlled {
		return
	}

	aiController := cm.combatService.GetAIController()
	aiExecutedActions := aiController.DecideFactionTurn(currentFactionID)

	combatLogArea := GetCombatLogTextArea(cm.Panels)
	if aiExecutedActions {
		cm.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("%s (AI) executed actions", factionData.Name))
	} else {
		cm.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("%s (AI) has no valid actions", factionData.Name))
	}

	if aiController.HasQueuedAttacks() {
		cm.playAIAttackAnimations(aiController)
		return
	}

	cm.advanceAfterAITurn()
}

// playAIAttackAnimations plays all queued AI attack animations sequentially
func (cm *CombatMode) playAIAttackAnimations(aiController *combatservices.AIController) {
	attacks := aiController.GetQueuedAttacks()

	if len(attacks) == 0 {
		cm.advanceAfterAITurn()
		return
	}

	cm.playNextAIAttack(attacks, 0, aiController)
}

// playNextAIAttack plays a single AI attack animation and chains to the next
func (cm *CombatMode) playNextAIAttack(attacks []combatservices.QueuedAttack, index int, aiController *combatservices.AIController) {
	if index >= len(attacks) {
		aiController.ClearAttackQueue()

		if combatMode, exists := cm.ModeManager.GetMode("combat"); exists {
			cm.ModeManager.RequestTransition(combatMode, "AI attacks complete")
		}

		cm.advanceAfterAITurn()
		return
	}

	attack := attacks[index]
	isFirstAttack := (index == 0)

	if animMode, exists := cm.ModeManager.GetMode("combat_animation"); exists {
		if caMode, ok := animMode.(*CombatAnimationMode); ok {
			caMode.SetCombatants(attack.AttackerID, attack.DefenderID)
			caMode.SetAutoPlay(true)

			caMode.SetOnComplete(func() {
				caMode.ResetForNextAttack()
				cm.playNextAIAttack(attacks, index+1, aiController)
			})

			if isFirstAttack {
				cm.ModeManager.RequestTransition(animMode, "AI Attack Animation")
			}
		} else {
			aiController.ClearAttackQueue()
			cm.advanceAfterAITurn()
		}
	} else {
		aiController.ClearAttackQueue()
		cm.advanceAfterAITurn()
	}
}

// advanceAfterAITurn advances to next turn after AI completes
func (cm *CombatMode) advanceAfterAITurn() {
	err := cm.combatService.TurnManager.EndTurn()
	if err != nil {
		combatLogArea := GetCombatLogTextArea(cm.Panels)
		cm.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("Error ending AI turn: %s", err.Error()))
		return
	}

	cm.Queries.MarkAllSquadsDirty()

	// Check for victory after AI turn
	if cm.checkAndHandleVictory() {
		return
	}

	newFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	round := cm.combatService.TurnManager.GetCurrentRound()

	if cm.combatService.BattleRecorder != nil && cm.combatService.BattleRecorder.IsEnabled() {
		cm.combatService.BattleRecorder.SetCurrentRound(round)
	}

	turnMessage := cm.formatTurnMessage(newFactionID, round)
	combatLogArea := GetCombatLogTextArea(cm.Panels)
	cm.logManager.UpdateTextArea(combatLogArea, turnMessage)

	cm.turnOrderComponent.Refresh()
	cm.factionInfoComponent.ShowFaction(newFactionID)
	cm.squadListComponent.Refresh()

	cm.visualization.UpdateThreatManagers()

	cm.executeAITurnIfNeeded()
}

func (cm *CombatMode) Enter(fromMode framework.UIMode) error {
	fromModeName := "nil"
	if fromMode != nil {
		fromModeName = fromMode.GetModeName()
	}
	cm.debugLogger.LogModeTransition("ENTER", fromModeName)

	isComingFromAnimation := fromMode != nil && fromMode.GetModeName() == "combat_animation"
	shouldInitialize := !isComingFromAnimation

	combatLogArea := GetCombatLogTextArea(cm.Panels)

	if shouldInitialize {
		cm.debugLogger.Log("Fresh combat start - initializing")
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
		cm.squadListComponent.Refresh()
	}

	return nil
}

func (cm *CombatMode) Exit(toMode framework.UIMode) error {
	toModeName := "nil"
	if toMode != nil {
		toModeName = toMode.GetModeName()
	}
	cm.debugLogger.LogModeTransition("EXIT", toModeName)

	isToAnimation := toMode != nil && toMode.GetModeName() == "combat_animation"

	if !isToAnimation {
		cm.debugLogger.Log("Full cleanup - returning to exploration")

		// Get victory result (use cached if available, otherwise check now)
		victor := cm.lastVictoryResult
		if victor == nil {
			victor = cm.combatService.CheckVictoryCondition()
		}

		// Determine exit reason
		reason := encounter.ExitDefeat
		if cm.fleeRequested {
			reason = encounter.ExitFlee
			cm.fleeRequested = false
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
				cm.debugLogger.LogError("ExportBattleLog", err, nil)
			}
			cm.combatService.BattleRecorder.Clear()
			cm.combatService.BattleRecorder.SetEnabled(false)
		}

		// Clear cached victory result
		cm.lastVictoryResult = nil
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

// getValidMoveTiles computes valid movement tiles on-demand
func (cm *CombatMode) getValidMoveTiles() []coords.LogicalPosition {
	battleState := cm.Context.ModeCoordinator.GetBattleMapState()

	if battleState.SelectedSquadID == 0 {
		return []coords.LogicalPosition{}
	}

	if !battleState.InMoveMode {
		return []coords.LogicalPosition{}
	}

	tiles := cm.combatService.MovementSystem.GetValidMovementTiles(battleState.SelectedSquadID)
	if tiles == nil {
		return []coords.LogicalPosition{}
	}

	return tiles
}

func (cm *CombatMode) Render(screen *ebiten.Image) {
	playerPos := *cm.Context.PlayerData.Pos
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	battleState := cm.Context.ModeCoordinator.GetBattleMapState()
	selectedSquad := battleState.SelectedSquadID

	cm.visualization.GetHighlightRenderer().Render(screen, playerPos, currentFactionID, selectedSquad)

	if battleState.InMoveMode {
		validTiles := cm.getValidMoveTiles()
		if len(validTiles) > 0 {
			cm.visualization.GetMovementRenderer().Render(screen, playerPos, validTiles)
		}
	}

	// Render health bars when enabled
	if battleState.ShowHealthBars {
		cm.visualization.GetHealthBarRenderer().Render(screen, playerPos)
	}
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
		cm.handleEndTurn()
		return true
	}

	// H key to toggle threat heat map
	if inputState.KeysJustPressed[ebiten.KeyH] {
		threatViz := cm.visualization.GetThreatVisualizer()
		if threatViz == nil {
			return true
		}

		shiftPressed := inputState.KeysPressed[ebiten.KeyShift] ||
			inputState.KeysPressed[ebiten.KeyShiftLeft] ||
			inputState.KeysPressed[ebiten.KeyShiftRight]

		combatLogArea := GetCombatLogTextArea(cm.Panels)

		if shiftPressed {
			threatViz.SwitchThreatView()
			viewName := "Enemy Threats"
			if threatViz.GetThreatViewMode() == behavior.ViewPlayerThreats {
				viewName = "Player Threats"
			}
			cm.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("Switched to %s view", viewName))
		} else {
			// If not active or in different mode: activate in Threat mode
			// If already active in Threat mode: turn off
			if !threatViz.IsActive() || threatViz.GetMode() != behavior.VisualizerModeThreat {
				threatViz.SetMode(behavior.VisualizerModeThreat)
				if !threatViz.IsActive() {
					threatViz.Toggle()
				}
				cm.logManager.UpdateTextArea(combatLogArea, "Threat visualization enabled")
			} else {
				threatViz.Toggle()
				cm.logManager.UpdateTextArea(combatLogArea, "Threat visualization disabled")
			}
		}
		cm.updateLayerStatusWidget()
		return true
	}

	// Right Control key to toggle health bars
	if inputState.KeysJustPressed[ebiten.KeyControlRight] {
		battleState := cm.Context.ModeCoordinator.GetBattleMapState()
		battleState.ShowHealthBars = !battleState.ShowHealthBars
		status := "enabled"
		if !battleState.ShowHealthBars {
			status = "disabled"
		}
		combatLogArea := GetCombatLogTextArea(cm.Panels)
		cm.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("Health bars %s", status))
		return true
	}

	// L key to toggle layer visualizer
	if inputState.KeysJustPressed[ebiten.KeyL] {
		threatViz := cm.visualization.GetThreatVisualizer()
		if threatViz == nil {
			return true
		}

		shiftPressed := inputState.KeysPressed[ebiten.KeyShift] ||
			inputState.KeysPressed[ebiten.KeyShiftLeft] ||
			inputState.KeysPressed[ebiten.KeyShiftRight]

		combatLogArea := GetCombatLogTextArea(cm.Panels)

		if shiftPressed {
			threatViz.CycleLayerMode()
			modeInfo := threatViz.GetLayerModeInfo()
			cm.logManager.UpdateTextArea(combatLogArea,
				fmt.Sprintf("Layer: %s (%s)", modeInfo.Name, modeInfo.ColorKey))
		} else {
			// If not active or in different mode: activate in Layer mode
			// If already active in Layer mode: turn off
			if !threatViz.IsActive() || threatViz.GetMode() != behavior.VisualizerModeLayer {
				threatViz.SetMode(behavior.VisualizerModeLayer)
				if !threatViz.IsActive() {
					threatViz.Toggle()
				}
				modeInfo := threatViz.GetLayerModeInfo()
				cm.logManager.UpdateTextArea(combatLogArea,
					fmt.Sprintf("Layer visualization enabled: %s", modeInfo.Name))
			} else {
				threatViz.Toggle()
				cm.logManager.UpdateTextArea(combatLogArea, "Layer visualization disabled")
			}
		}
		cm.updateLayerStatusWidget()
		return true
	}

	return false
}

// updateLayerStatusWidget updates the layer status panel visibility and text
func (cm *CombatMode) updateLayerStatusWidget() {
	threatViz := cm.visualization.GetThreatVisualizer()
	layerStatusPanel := cm.GetPanelContainer(CombatPanelLayerStatus)
	layerStatusText := cm.GetTextLabel(CombatPanelLayerStatus)

	if layerStatusPanel == nil || layerStatusText == nil || threatViz == nil {
		return
	}

	// Show layer status only when in layer mode and active
	if threatViz.IsActive() && threatViz.GetMode() == behavior.VisualizerModeLayer {
		modeInfo := threatViz.GetLayerModeInfo()
		statusText := fmt.Sprintf("LAYER VIEW\n%s\n%s", modeInfo.Name, modeInfo.ColorKey)
		layerStatusText.Label = statusText
		layerStatusPanel.GetWidget().Visibility = widget.Visibility_Show
	} else {
		layerStatusPanel.GetWidget().Visibility = widget.Visibility_Hide
	}
}
