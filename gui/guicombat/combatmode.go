package guicombat

import (
	"fmt"
	"game_main/common"
	"game_main/config"
	"game_main/gui"
	"game_main/gui/builders"
	"game_main/gui/core"
	"game_main/gui/guicomponents"
	"game_main/gui/guimodes"
	"game_main/gui/guiresources"
	"game_main/gui/widgets"
	"game_main/tactical/behavior"
	"game_main/tactical/combat"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combatservices"
	"game_main/tactical/squads"
	"game_main/world/coords"
	"game_main/world/encounter"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatMode provides focused UI for turn-based squad combat
type CombatMode struct {
	gui.BaseMode // Embed common mode infrastructure

	// Managers
	logManager    *CombatLogManager
	actionHandler *CombatActionHandler
	inputHandler  *CombatInputHandler
	uiFactory     *gui.UIComponentFactory
	combatService *combatservices.CombatService

	// UI panels and widgets
	turnOrderPanel   *widget.Container
	factionInfoPanel *widget.Container
	squadListPanel   *widget.Container
	squadDetailPanel *widget.Container
	combatLogArea    *widgets.CachedTextAreaWrapper // Cached for performance
	actionButtons    *widget.Container
	layerStatusPanel *widget.Container

	// UI text labels
	turnOrderLabel  *widget.Text
	factionInfoText *widget.Text
	squadDetailText *widget.Text
	layerStatusText *widget.Text

	// UI update components
	squadListComponent   *guicomponents.SquadListComponent
	squadDetailComponent *guicomponents.DetailPanelComponent
	factionInfoComponent *guicomponents.DetailPanelComponent
	turnOrderComponent   *guicomponents.TextDisplayComponent

	// Rendering systems
	movementRenderer  *guimodes.MovementTileRenderer
	highlightRenderer *guimodes.SquadHighlightRenderer
	dangerVisualizer  *behavior.DangerVisualizer
	layerVisualizer   *behavior.LayerVisualizer
	threatManager     *behavior.FactionThreatLevelManager
	threatEvaluator   *behavior.CompositeThreatEvaluator

	// State tracking for UI updates (GUI_PERFORMANCE_ANALYSIS.md)
	lastFactionID     ecs.EntityID
	lastSelectedSquad ecs.EntityID

	// Encounter tracking
	currentEncounterID ecs.EntityID // Tracks which encounter triggered this combat
}

func NewCombatMode(modeManager *core.UIModeManager) *CombatMode {
	cm := &CombatMode{
		logManager: NewCombatLogManager(),
	}
	cm.SetModeName("combat")
	cm.SetReturnMode("exploration") // ESC returns to exploration
	cm.ModeManager = modeManager
	return cm
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
	// Create combat service before ModeBuilder
	cm.combatService = combatservices.NewCombatService(ctx.ECSManager)

	// Build UI using ModeBuilder
	err := gui.NewModeBuilder(&cm.BaseMode, gui.ModeConfig{
		ModeName:   "combat",
		ReturnMode: "exploration",

		Panels: []gui.PanelSpec{
			{CustomBuild: cm.buildTurnOrderPanel},
			{CustomBuild: cm.buildFactionInfoPanel},
			{CustomBuild: cm.buildSquadListPanel},
			{CustomBuild: cm.buildSquadDetailPanel},
			{CustomBuild: cm.buildLogPanel},
			{CustomBuild: cm.buildActionButtons},
			{CustomBuild: cm.buildLayerStatusPanel},
		},
	}).Build(ctx)
	if err != nil {
		return err
	}

	// Post-UI initialization (after panels are built)
	cm.actionHandler = NewCombatActionHandler(
		ctx.ModeCoordinator.GetBattleMapState(),
		cm.logManager,
		cm.Queries,
		cm.combatService,
		cm.combatLogArea,
		cm.ModeManager,
	)

	cm.inputHandler = NewCombatInputHandler(
		cm.actionHandler,
		ctx.ModeCoordinator.GetBattleMapState(),
		cm.Queries,
	)

	cm.initializeUpdateComponents()

	cm.movementRenderer = guimodes.NewMovementTileRenderer()
	cm.highlightRenderer = guimodes.NewSquadHighlightRenderer(cm.Queries)

	// Cast GameMap from interface{} to *worldmap.GameMap
	//TODO, remove in future. Here for gamevisualizer
	gameMap := ctx.GameMap.(*worldmap.GameMap)

	//Create the initial Faction Threat Level Manager and add all factions.
	cm.threatManager = behavior.NewFactionThreatLevelManager(cm.Context.ECSManager, cm.Queries.CombatCache)
	for _, IDs := range cm.Queries.GetAllFactions() {

		cm.threatManager.AddFaction(IDs)

	}

	cm.dangerVisualizer = behavior.NewDangerVisualizer(ctx.ECSManager, gameMap, cm.threatManager)

	// Create threat evaluators for layer visualization
	allFactions := cm.Queries.GetAllFactions()
	if len(allFactions) > 0 {
		// Use player faction (first faction) for threat evaluation
		playerFactionID := allFactions[0]
		cm.threatEvaluator = behavior.NewCompositeThreatEvaluator(
			playerFactionID,
			ctx.ECSManager,
			cm.Queries.CombatCache,
			cm.threatManager,
		)
		cm.layerVisualizer = behavior.NewLayerVisualizer(
			ctx.ECSManager,
			gameMap,
			cm.threatEvaluator,
		)
	}

	return nil
}

func (cm *CombatMode) ensureUIFactoryInitialized() {
	if cm.uiFactory == nil {
		cm.uiFactory = gui.NewUIComponentFactory(cm.Queries, cm.PanelBuilders, cm.Layout)
	}
}

func (cm *CombatMode) buildTurnOrderPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	cm.turnOrderPanel = cm.uiFactory.CreateCombatTurnOrderPanel()
	cm.turnOrderLabel = builders.CreateLargeLabel("Initializing combat...")
	cm.turnOrderPanel.AddChild(cm.turnOrderLabel)

	return cm.turnOrderPanel
}

func (cm *CombatMode) buildFactionInfoPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	cm.factionInfoPanel = cm.uiFactory.CreateCombatFactionInfoPanel()
	cm.factionInfoText = builders.CreateSmallLabel("Faction Info")
	cm.factionInfoPanel.AddChild(cm.factionInfoText)

	return cm.factionInfoPanel
}

func (cm *CombatMode) buildSquadListPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	cm.squadListPanel = cm.uiFactory.CreateCombatSquadListPanel()

	return cm.squadListPanel
}

func (cm *CombatMode) buildSquadDetailPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	cm.squadDetailPanel = cm.uiFactory.CreateCombatSquadDetailPanel()
	cm.squadDetailText = builders.CreateSmallLabel("Select a squad\nto view details")
	cm.squadDetailPanel.AddChild(cm.squadDetailText)

	return cm.squadDetailPanel
}

func (cm *CombatMode) buildLogPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	// Create log panel only if combat log is enabled
	if config.ENABLE_COMBAT_LOG {
		logContainer, logArea := cm.uiFactory.CreateCombatLogPanel()
		cm.combatLogArea = logArea
		return logContainer
	}

	// Return empty container if log is disabled
	return widget.NewContainer()
}

func (cm *CombatMode) buildActionButtons() *widget.Container {
	cm.ensureUIFactoryInitialized()

	// Create action buttons
	// TODO, get rid of handleAttackClick,handleMovClick,handleUndoMove,and handleRedoMove.
	// CombatActionSystem throws a null error when I call the function they are a wrapper around
	cm.actionButtons = cm.uiFactory.CreateCombatActionButtons(
		cm.handleAttackClick,
		cm.handleMoveClick,
		cm.handleUndoMove,
		cm.handleRedoMove,
		cm.handleEndTurn,
		cm.handleFlee,
	)

	return cm.actionButtons
}

func (cm *CombatMode) buildLayerStatusPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	// Create a small panel for layer status display
	panelWidth := int(float64(cm.Layout.ScreenWidth) * 0.15)   // 15% of screen width
	panelHeight := int(float64(cm.Layout.ScreenHeight) * 0.08) // 8% of screen height

	cm.layerStatusPanel = builders.CreatePanelWithConfig(builders.PanelConfig{
		MinWidth:   panelWidth,
		MinHeight:  panelHeight,
		Background: guiresources.PanelRes.Image,
		Layout: widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(3),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(5)),
		),
	})

	// Position in top-right corner
	rightPad := int(float64(cm.Layout.ScreenWidth) * 0.01)
	topPad := int(float64(cm.Layout.ScreenHeight) * 0.01)
	cm.layerStatusPanel.GetWidget().LayoutData = gui.AnchorEndStart(rightPad, topPad)

	// Create status text (initially hidden)
	cm.layerStatusText = builders.CreateSmallLabel("")
	cm.layerStatusPanel.AddChild(cm.layerStatusText)

	// Hide panel initially (shown when visualizer is active)
	cm.layerStatusPanel.GetWidget().Visibility = widget.Visibility_Hide

	return cm.layerStatusPanel
}

// updateLayerStatusWidget updates the layer status panel visibility and text
func (cm *CombatMode) updateLayerStatusWidget() {
	if cm.layerStatusPanel == nil || cm.layerStatusText == nil || cm.layerVisualizer == nil {
		return
	}

	if cm.layerVisualizer.IsActive() {
		// Show panel and update text with current mode info
		modeInfo := cm.layerVisualizer.GetCurrentModeInfo()
		statusText := fmt.Sprintf("LAYER VIEW\n%s\n%s", modeInfo.Name, modeInfo.ColorKey)
		cm.layerStatusText.Label = statusText
		cm.layerStatusPanel.GetWidget().Visibility = widget.Visibility_Show
	} else {
		// Hide panel when visualizer is inactive
		cm.layerStatusPanel.GetWidget().Visibility = widget.Visibility_Hide
	}
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
	// Turn order component - displays current faction and round
	cm.turnOrderComponent = guicomponents.NewTextDisplayComponent(
		cm.turnOrderLabel,
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

				// Show player-specific turn indicator
				if factionData.PlayerID > 0 {
					// Human player's turn - show player name
					turnIndicator = fmt.Sprintf(" >>> %s's TURN <<<", factionData.PlayerName)
				} else {
					// AI's turn
					turnIndicator = " [AI TURN]"
				}
			}

			return fmt.Sprintf("Round %d | %s%s", round, factionName, turnIndicator)
		},
	)

	// Faction info component - displays squad count and mana
	cm.factionInfoComponent = guicomponents.NewDetailPanelComponent(
		cm.factionInfoText,
		cm.Queries,
		func(data interface{}) string {
			factionInfo := data.(*guicomponents.FactionInfo)

			// Get full faction data to access PlayerName
			factionData := cm.Queries.CombatCache.FindFactionDataByID(factionInfo.ID, cm.Queries.ECSManager)

			infoText := fmt.Sprintf("%s\n", factionInfo.Name)

			// Add player identification
			if factionData != nil && factionData.PlayerID > 0 {
				infoText += fmt.Sprintf("[%s]\n", factionData.PlayerName)
			}

			infoText += fmt.Sprintf("Squads: %d/%d\n", factionInfo.AliveSquadCount, len(factionInfo.SquadIDs))
			infoText += fmt.Sprintf("Mana: %d/%d", factionInfo.CurrentMana, factionInfo.MaxMana)
			return infoText
		},
	)

	// Squad detail component - displays selected squad details
	cm.squadDetailComponent = guicomponents.NewDetailPanelComponent(
		cm.squadDetailText,
		cm.Queries,
		nil, // Use default formatter
	)

	// Squad list component - filter for current faction squads (during player's turn only)
	// Extracted filter logic to separate method to eliminate inline duplication
	cm.squadListComponent = guicomponents.NewSquadListComponent(
		cm.squadListPanel,
		cm.Queries,
		cm.makeCurrentFactionSquadFilter(),
		func(squadID ecs.EntityID) {
			cm.actionHandler.SelectSquad(squadID)
			cm.squadDetailComponent.ShowSquad(squadID)
		},
	)
}

// makeCurrentFactionSquadFilter creates a filter for squads from the current faction
// Only shows squads during the player's faction's turn
func (cm *CombatMode) makeCurrentFactionSquadFilter() guicomponents.SquadFilter {
	return func(info *guicomponents.SquadInfo) bool {
		currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
		if currentFactionID == 0 {
			return false
		}
		// Only show squads if it's player's turn
		factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID, cm.Queries.ECSManager)
		if factionData == nil || !factionData.IsPlayerControlled {
			return false
		}
		return !info.IsDestroyed && info.FactionID == currentFactionID
	}
}

func (cm *CombatMode) handleFlee() {
	cm.logManager.UpdateTextArea(cm.combatLogArea, "Fleeing from combat...")

	// Return to exploration mode (stays in BattleMap context)
	if exploreMode, exists := cm.ModeManager.GetMode("exploration"); exists {
		cm.ModeManager.RequestTransition(exploreMode, "Fled from combat")
	}
}

func (cm *CombatMode) handleEndTurn() {
	// Clear movement history when ending turn (can't undo moves from previous turns)
	cm.actionHandler.ClearMoveHistory()

	// End current faction's turn using turn manager
	err := cm.combatService.TurnManager.EndTurn()
	if err != nil {
		cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Error ending turn: %s", err.Error()))
		return
	}

	// Invalidate all squad caches since turn changed (all action states reset)
	cm.Queries.MarkAllSquadsDirty()

	// Get new faction info
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	round := cm.combatService.TurnManager.GetCurrentRound()

	// Update battle recorder with current round
	if cm.combatService.BattleRecorder != nil && cm.combatService.BattleRecorder.IsEnabled() {
		cm.combatService.BattleRecorder.SetCurrentRound(round)
	}

	// Get faction data for player name
	factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID, cm.Queries.ECSManager)
	factionName := "Unknown"
	turnMessage := ""

	if factionData != nil {
		factionName = factionData.Name
		if factionData.PlayerID > 0 {
			turnMessage = fmt.Sprintf("=== Round %d: %s (%s) ===", round, factionName, factionData.PlayerName)
		} else {
			turnMessage = fmt.Sprintf("=== Round %d: %s (AI) ===", round, factionName)
		}
	} else {
		turnMessage = fmt.Sprintf("=== Round %d: %s's Turn ===", round, factionName)
	}

	cm.logManager.UpdateTextArea(cm.combatLogArea, turnMessage)

	// Clear selection when turn changes
	cm.Context.ModeCoordinator.GetBattleMapState().Reset()

	// Update UI displays using components
	cm.turnOrderComponent.Refresh()
	cm.factionInfoComponent.ShowFaction(currentFactionID)
	cm.squadListComponent.Refresh()
	cm.squadDetailComponent.SetText("Select a squad\nto view details")

	cm.threatManager.UpdateAllFactions()

	// Update threat evaluator for layer visualization
	if cm.threatEvaluator != nil {
		cm.threatEvaluator.Update(round)
	}

	// Execute AI turn if current faction is AI-controlled
	cm.executeAITurnIfNeeded()

}

// executeAITurnIfNeeded checks if current faction is AI-controlled and executes its turn
func (cm *CombatMode) executeAITurnIfNeeded() {
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return
	}

	// Check if faction is AI-controlled
	factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID, cm.Queries.ECSManager)
	if factionData == nil || factionData.IsPlayerControlled {
		return // Player-controlled faction, wait for player input
	}

	// Create AI controller and execute AI turn
	aiController := cm.combatService.GetAIController()

	// Execute AI actions until faction has no more actions
	aiExecutedActions := aiController.DecideFactionTurn(currentFactionID)

	if aiExecutedActions {
		cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("%s (AI) executed actions", factionData.Name))
	} else {
		cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("%s (AI) has no valid actions", factionData.Name))
	}

	// Check if there are attacks to animate
	if aiController.HasQueuedAttacks() {
		cm.playAIAttackAnimations(aiController)
		return // Animation completion callback will advance the turn
	}

	// No animations - advance turn immediately
	cm.advanceAfterAITurn()
}

// playAIAttackAnimations plays all queued AI attack animations sequentially
func (cm *CombatMode) playAIAttackAnimations(aiController *combatservices.AIController) {
	attacks := aiController.GetQueuedAttacks()

	if len(attacks) == 0 {
		cm.advanceAfterAITurn()
		return
	}

	// Start playing first attack (will chain to next via callbacks)
	cm.playNextAIAttack(attacks, 0, aiController)
}

// playNextAIAttack plays a single AI attack animation and chains to the next
func (cm *CombatMode) playNextAIAttack(attacks []combatservices.QueuedAttack, index int, aiController *combatservices.AIController) {
	// All attacks played - return to combat mode and advance turn
	if index >= len(attacks) {
		aiController.ClearAttackQueue()

		// Return to combat mode
		if combatMode, exists := cm.ModeManager.GetMode("combat"); exists {
			cm.ModeManager.RequestTransition(combatMode, "AI attacks complete")
		}

		cm.advanceAfterAITurn()
		return
	}

	attack := attacks[index]
	isFirstAttack := (index == 0)

	// Get animation mode
	if animMode, exists := cm.ModeManager.GetMode("combat_animation"); exists {
		if caMode, ok := animMode.(*CombatAnimationMode); ok {
			// Configure for AI attack
			caMode.SetCombatants(attack.AttackerID, attack.DefenderID)
			caMode.SetAutoPlay(true) // Enable auto-play for AI attacks

			// Set callback to play next attack
			caMode.SetOnComplete(func() {
				// Reset animation state for next attack
				caMode.ResetForNextAttack()

				// Play next attack (stays in animation mode)
				cm.playNextAIAttack(attacks, index+1, aiController)
			})

			// Only transition on first attack (subsequent attacks stay in animation mode)
			if isFirstAttack {
				cm.ModeManager.RequestTransition(animMode, "AI Attack Animation")
			}
		} else {
			// Animation mode wrong type - skip animations and advance
			aiController.ClearAttackQueue()
			cm.advanceAfterAITurn()
		}
	} else {
		// No animation mode - skip animations and advance
		aiController.ClearAttackQueue()
		cm.advanceAfterAITurn()
	}
}

// advanceAfterAITurn advances to next turn after AI completes (with or without animations)
func (cm *CombatMode) advanceAfterAITurn() {
	// Process any queued commands

	// End AI turn
	err := cm.combatService.TurnManager.EndTurn()
	if err != nil {
		cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Error ending AI turn: %s", err.Error()))
		return
	}

	// Invalidate caches
	cm.Queries.MarkAllSquadsDirty()

	// Update UI
	newFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	round := cm.combatService.TurnManager.GetCurrentRound()

	// Update battle recorder with current round
	if cm.combatService.BattleRecorder != nil && cm.combatService.BattleRecorder.IsEnabled() {
		cm.combatService.BattleRecorder.SetCurrentRound(round)
	}

	newFactionData := cm.Queries.CombatCache.FindFactionDataByID(newFactionID, cm.Queries.ECSManager)
	if newFactionData != nil {
		turnMessage := ""
		if newFactionData.PlayerID > 0 {
			turnMessage = fmt.Sprintf("=== Round %d: %s (%s) ===", round, newFactionData.Name, newFactionData.PlayerName)
		} else {
			turnMessage = fmt.Sprintf("=== Round %d: %s (AI) ===", round, newFactionData.Name)
		}
		cm.logManager.UpdateTextArea(cm.combatLogArea, turnMessage)
	}

	cm.turnOrderComponent.Refresh()
	cm.factionInfoComponent.ShowFaction(newFactionID)
	cm.squadListComponent.Refresh()

	cm.threatManager.UpdateAllFactions()

	// Recursively execute next AI turn if needed
	cm.executeAITurnIfNeeded()
}

func (cm *CombatMode) SetupEncounter(fromMode core.UIMode) error {

	cm.logManager.UpdateTextArea(cm.combatLogArea, "Fresh combat encounter - spawning entities")

	// Get encounter ID from BattleMapState
	if cm.Context.ModeCoordinator != nil {
		battleMapState := cm.Context.ModeCoordinator.GetBattleMapState()
		cm.currentEncounterID = battleMapState.TriggeredEncounterID

		// Log encounter info if available
		if cm.currentEncounterID != 0 {
			entity := cm.Context.ECSManager.FindEntityByID(cm.currentEncounterID)
			if entity != nil {
				encounterData := common.GetComponentType[*encounter.OverworldEncounterData](
					entity,
					encounter.OverworldEncounterComponent,
				)
				if encounterData != nil {
					cm.logManager.UpdateTextArea(cm.combatLogArea,
						fmt.Sprintf("Encounter: %s (Level %d)", encounterData.Name, encounterData.Level))
				}
			}
		}
	}

	// Get player start position for squad spawning
	playerStartPos := coords.LogicalPosition{X: 50, Y: 40}
	if cm.Context.PlayerData != nil && cm.Context.PlayerData.Pos != nil {
		playerStartPos = *cm.Context.PlayerData.Pos
	}

	// Call SetupGameplayFactions to create combat entities
	if err := combat.SetupGameplayFactions(cm.Context.ECSManager, playerStartPos); err != nil {
		cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Error spawning combat entities: %v", err))
		return fmt.Errorf("failed to setup gameplay factions: %w", err)
	}

	return nil

}

func (cm *CombatMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Combat Mode")

	// Check if we're starting fresh combat or returning mid-combat
	// Fresh combat: coming from exploration, squad deployment, or worldmap modes
	// Mid-combat: returning from animation mode
	isComingFromAnimation := fromMode != nil && fromMode.GetModeName() == "combat_animation"
	shouldInitialize := !isComingFromAnimation

	if shouldInitialize {
		// Fresh combat start - initialize everything
		cm.logManager.UpdateTextArea(cm.combatLogArea, "=== COMBAT STARTED ===")

		cm.SetupEncounter(fromMode)
		// Start battle recording if enabled
		if config.ENABLE_COMBAT_LOG_EXPORT && cm.combatService.BattleRecorder != nil {
			cm.combatService.BattleRecorder.SetEnabled(true)
			cm.combatService.BattleRecorder.Start()
			cm.combatService.BattleRecorder.SetCurrentRound(1)
		}

		// Enter new mode
		if cm.initialzieCombatFactions() != nil {
			return fmt.Errorf("Error initializing combat factions")
		}

	}

	// Always refresh UI displays (whether fresh or returning from animation)
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	if currentFactionID != 0 {
		cm.turnOrderComponent.Refresh()
		cm.factionInfoComponent.ShowFaction(currentFactionID)
		cm.squadListComponent.Refresh()
	}

	return nil
}

func (cm *CombatMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Combat Mode")

	// Check if returning to exploration (not going to animation mode)
	isToAnimation := toMode != nil && toMode.GetModeName() == "combat_animation"

	// Mark encounter as defeated if player won
	if !isToAnimation {
		cm.markEncounterDefeatedIfVictorious()
	}

	// Clean up all combat entities when returning to exploration
	if !isToAnimation {
		cm.cleanupCombatEntities()
	}

	// Export battle log when leaving combat (not to animation mode)
	if !isToAnimation && config.ENABLE_COMBAT_LOG_EXPORT && cm.combatService.BattleRecorder != nil && cm.combatService.BattleRecorder.IsEnabled() {
		victor := cm.combatService.CheckVictoryCondition()
		victoryInfo := &battlelog.VictoryInfo{
			RoundsCompleted: victor.RoundsCompleted,
			VictorFaction:   victor.VictorFaction,
			VictorName:      victor.VictorName,
		}
		record := cm.combatService.BattleRecorder.Finalize(victoryInfo)
		if err := battlelog.ExportBattleJSON(record, config.COMBAT_LOG_EXPORT_DIR); err != nil {
			fmt.Printf("Failed to export combat log: %v\n", err)
		}
		cm.combatService.BattleRecorder.Clear()
		cm.combatService.BattleRecorder.SetEnabled(false)
	}

	// Clear danger visualization
	if cm.dangerVisualizer != nil {
		cm.dangerVisualizer.ClearVisualization()
	}

	// Clear layer visualization
	if cm.layerVisualizer != nil {
		cm.layerVisualizer.ClearVisualization()
	}

	// Clear combat log for next battle
	cm.logManager.Clear()
	return nil
}

func (cm *CombatMode) Update(deltaTime float64) error {
	// Only update UI displays when state changes (GUI_PERFORMANCE_ANALYSIS.md)
	// This avoids expensive text measurement on every frame (~10-15s CPU savings)
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

	// Update danger visualization if active
	if cm.dangerVisualizer.IsActive() {
		currentRound := cm.combatService.TurnManager.GetCurrentRound()
		playerPos := *cm.Context.PlayerData.Pos
		viewportSize := 30 // Process 30x30 tile area around player
		cm.dangerVisualizer.Update(currentFactionID, currentRound, playerPos, viewportSize)
	}

	// Update layer visualization if active
	if cm.layerVisualizer != nil && cm.layerVisualizer.IsActive() {
		currentRound := cm.combatService.TurnManager.GetCurrentRound()
		playerPos := *cm.Context.PlayerData.Pos
		viewportSize := 30
		cm.layerVisualizer.Update(currentFactionID, currentRound, playerPos, viewportSize)
	}

	return nil
}

func (cm *CombatMode) initialzieCombatFactions() error {

	// Collect all factions using query service
	factionIDs := cm.Queries.GetAllFactions()

	// Initialize combat with all factions
	if len(factionIDs) > 0 {
		if err := cm.combatService.InitializeCombat(factionIDs); err != nil {
			cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Error initializing combat: %v", err))
			return err
		}

		// Log initial faction
		currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
		factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID, cm.Queries.ECSManager)
		factionName := "Unknown"
		if factionData != nil {
			factionName = factionData.Name
		}
		cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Round 1: %s goes first!", factionName))
	} else {
		cm.logManager.UpdateTextArea(cm.combatLogArea, "No factions found - combat cannot start")
	}

	return nil

}

// markEncounterDefeatedIfVictorious marks the encounter as defeated if player won
func (cm *CombatMode) markEncounterDefeatedIfVictorious() {
	// Only mark if we have a tracked encounter
	if cm.currentEncounterID == 0 {
		return
	}

	// Check victory condition
	victor := cm.combatService.CheckVictoryCondition()

	// Only mark as defeated if a player faction won
	if victor.VictorFaction != 0 {
		factionData := cm.Queries.CombatCache.FindFactionDataByID(victor.VictorFaction, cm.Queries.ECSManager)
		if factionData != nil && factionData.IsPlayerControlled {
			// Player won - mark encounter as defeated
			entity := cm.Context.ECSManager.FindEntityByID(cm.currentEncounterID)
			if entity != nil {
				encounterData := common.GetComponentType[*encounter.OverworldEncounterData](
					entity,
					encounter.OverworldEncounterComponent,
				)
				if encounterData != nil {
					encounterData.IsDefeated = true
					fmt.Printf("Marked encounter '%s' as defeated\n", encounterData.Name)
					cm.logManager.UpdateTextArea(cm.combatLogArea,
						fmt.Sprintf("Encounter '%s' defeated!", encounterData.Name))
				}
			}
		}
	}
}

// cleanupCombatEntities removes ALL combat entities when returning to exploration
func (cm *CombatMode) cleanupCombatEntities() {
	fmt.Println("Cleaning up combat entities")

	// Step 1: Collect IDs of combat squads (those being removed)
	combatSquadIDs := make(map[ecs.EntityID]bool)
	for _, result := range cm.Context.ECSManager.World.Query(squads.SquadTag) {
		entity := result.Entity
		// Combat squads have CombatFactionComponent
		if entity.HasComponent(combat.CombatFactionComponent) {
			combatSquadIDs[entity.GetID()] = true
		}
	}

	fmt.Printf("Found %d combat squads to remove\n", len(combatSquadIDs))

	// Step 2: Remove all faction entities
	for _, result := range cm.Context.ECSManager.World.Query(combat.FactionTag) {
		entity := result.Entity
		cm.Context.ECSManager.World.DisposeEntities(entity)
	}

	// Step 3: Remove ONLY combat squads (those with CombatFactionComponent)
	// This preserves exploration squads which don't have this component
	for _, result := range cm.Context.ECSManager.World.Query(squads.SquadTag) {
		entity := result.Entity

		// CRITICAL: Only remove squads that belong to factions (combat squads)
		// Exploration squads don't have CombatFactionComponent and should be preserved
		if !entity.HasComponent(combat.CombatFactionComponent) {
			fmt.Printf("Preserving exploration squad: %d\n", entity.GetID())
			continue // Skip exploration squads
		}

		fmt.Printf("Removing combat squad: %d\n", entity.GetID())

		// Remove from position system
		if entity.HasComponent(common.PositionComponent) {
			posData := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			if posData != nil {
				common.GlobalPositionSystem.RemoveEntity(entity.GetID(), *posData)
			}
		}

		// Dispose entity
		cm.Context.ECSManager.World.DisposeEntities(entity)
	}

	// Step 4: Remove ONLY unit entities that belong to combat squads
	// This preserves units in exploration squads
	for _, result := range cm.Context.ECSManager.World.Query(squads.SquadMemberTag) {
		entity := result.Entity

		// Get unit's squad ID
		memberData := common.GetComponentType[*squads.SquadMemberData](entity, squads.SquadMemberComponent)
		if memberData != nil {
			// Only remove if this unit belongs to a combat squad
			if combatSquadIDs[memberData.SquadID] {
				fmt.Printf("Removing combat unit from squad %d\n", memberData.SquadID)
				cm.Context.ECSManager.World.DisposeEntities(entity)
			} else {
				fmt.Printf("Preserving exploration unit from squad %d\n", memberData.SquadID)
			}
		}
	}

	// Step 5: Remove all action state entities
	for _, result := range cm.Context.ECSManager.World.Query(combat.ActionStateTag) {
		entity := result.Entity
		cm.Context.ECSManager.World.DisposeEntities(entity)
	}

	// Step 6: Remove turn state entity
	for _, result := range cm.Context.ECSManager.World.Query(combat.TurnStateTag) {
		entity := result.Entity
		cm.Context.ECSManager.World.DisposeEntities(entity)
	}

	// Step 7: Clear all caches
	cm.Queries.MarkAllSquadsDirty()

	fmt.Println("Combat entities cleanup complete")
}

// getValidMoveTiles computes valid movement tiles on-demand from combat service
// This is computed game data, not UI state, so we don't cache it
func (cm *CombatMode) getValidMoveTiles() []coords.LogicalPosition {
	battleState := cm.Context.ModeCoordinator.GetBattleMapState()

	if battleState.SelectedSquadID == 0 {
		return []coords.LogicalPosition{}
	}

	if !battleState.InMoveMode {
		return []coords.LogicalPosition{}
	}

	// Compute from ECS game state via movement system
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

	// Render squad highlights (always shown)
	cm.highlightRenderer.Render(screen, playerPos, currentFactionID, selectedSquad)

	// Render valid movement tiles (only in move mode)
	if battleState.InMoveMode {
		validTiles := cm.getValidMoveTiles()
		if len(validTiles) > 0 {
			cm.movementRenderer.Render(screen, playerPos, validTiles)
		}
	}
}

// renderMovementTiles and renderAllSquadHighlights removed - now using MovementTileRenderer and SquadHighlightRenderer

func (cm *CombatMode) HandleInput(inputState *core.InputState) bool {
	// Handle common input (ESC key to flee combat)
	if cm.HandleCommonInput(inputState) {
		return true
	}

	// Update input handler with player position and faction info
	cm.inputHandler.SetPlayerPosition(cm.Context.PlayerData.Pos)
	cm.inputHandler.SetCurrentFactionID(cm.combatService.TurnManager.GetCurrentFaction())

	// Handle combat-specific input through input handler
	if cm.inputHandler.HandleInput(inputState) {
		return true
	}

	// Space to end turn (handled separately here)
	if inputState.KeysJustPressed[ebiten.KeySpace] {
		cm.handleEndTurn()
		return true
	}

	// H key to toggle danger heat map
	if inputState.KeysJustPressed[ebiten.KeyH] {
		// Check if Shift key is pressed
		shiftPressed := inputState.KeysPressed[ebiten.KeyShift] ||
			inputState.KeysPressed[ebiten.KeyShiftLeft] ||
			inputState.KeysPressed[ebiten.KeyShiftRight]

		if shiftPressed {
			// Shift+H: Switch between enemy/player threat view
			cm.dangerVisualizer.SwitchView()
			viewName := "Enemy Threats"
			if cm.dangerVisualizer.GetViewMode() == behavior.ViewPlayerThreats {
				viewName = "Player Threats"
			}
			cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Switched to %s view", viewName))
		} else {
			// H alone: Toggle visualization on/off
			cm.dangerVisualizer.Toggle()
			status := "enabled"
			if !cm.dangerVisualizer.IsActive() {
				status = "disabled"
			}
			cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Danger visualization %s", status))
		}
		return true
	}

	// Left Control key to cycle between danger/expected damage metrics (when visualizer active)
	if inputState.KeysJustPressed[ebiten.KeyControlLeft] {
		if cm.dangerVisualizer.IsActive() {
			cm.dangerVisualizer.CycleMetric()
			metricName := "Danger"
			if cm.dangerVisualizer.GetMetricMode() == behavior.MetricExpectedDamage {
				metricName = "Expected Damage"
			}
			cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Switched to %s metric", metricName))
			return true
		}
	}

	// L key to toggle layer visualizer
	if inputState.KeysJustPressed[ebiten.KeyL] {
		if cm.layerVisualizer == nil {
			return true
		}

		// Check if Shift key is pressed
		shiftPressed := inputState.KeysPressed[ebiten.KeyShift] ||
			inputState.KeysPressed[ebiten.KeyShiftLeft] ||
			inputState.KeysPressed[ebiten.KeyShiftRight]

		if shiftPressed {
			// Shift+L: Cycle through layer modes
			cm.layerVisualizer.CycleMode()
			modeInfo := cm.layerVisualizer.GetCurrentModeInfo()
			cm.logManager.UpdateTextArea(cm.combatLogArea,
				fmt.Sprintf("Layer: %s (%s)", modeInfo.Name, modeInfo.ColorKey))
		} else {
			// L alone: Toggle visualization on/off
			cm.layerVisualizer.Toggle()
			status := "enabled"
			if !cm.layerVisualizer.IsActive() {
				status = "disabled"
			}
			cm.logManager.UpdateTextArea(cm.combatLogArea,
				fmt.Sprintf("Layer visualization %s", status))
		}
		// Update the status widget to reflect current mode
		cm.updateLayerStatusWidget()
		return true
	}

	return false
}
