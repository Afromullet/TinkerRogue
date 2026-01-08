package guicombat

import (
	"fmt"
	"game_main/config"
	"game_main/gui"
	"game_main/gui/builders"
	"game_main/gui/core"
	"game_main/gui/guicomponents"
	"game_main/gui/guiresources"
	"game_main/gui/widgets"
	"game_main/tactical/behavior"
	"game_main/tactical/combatservices"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatModeUI groups all UI widgets, components, and panels for CombatMode
type CombatModeUI struct {
	// Panels and widgets
	turnOrderPanel   *widget.Container
	factionInfoPanel *widget.Container
	squadListPanel   *widget.Container
	squadDetailPanel *widget.Container
	combatLogArea    *widgets.CachedTextAreaWrapper // Cached for performance
	actionButtons    *widget.Container
	layerStatusPanel *widget.Container

	// Text labels
	turnOrderLabel  *widget.Text
	factionInfoText *widget.Text
	squadDetailText *widget.Text
	layerStatusText *widget.Text

	// Update components
	squadListComponent   *guicomponents.SquadListComponent
	squadDetailComponent *guicomponents.DetailPanelComponent
	factionInfoComponent *guicomponents.DetailPanelComponent
	turnOrderComponent   *guicomponents.TextDisplayComponent
}

// CombatMode provides focused UI for turn-based squad combat
type CombatMode struct {
	gui.BaseMode // Embed common mode infrastructure

	// Managers
	logManager       *CombatLogManager
	actionHandler    *CombatActionHandler
	inputHandler     *CombatInputHandler
	uiFactory        *gui.UIComponentFactory
	combatService    *combatservices.CombatService
	lifecycleManager *CombatLifecycleManager

	// UI state (grouped for clarity)
	ui *CombatModeUI

	// Visualization systems (grouped for clarity)
	visualization *CombatVisualizationManager

	// State tracking for UI updates (GUI_PERFORMANCE_ANALYSIS.md)
	lastFactionID     ecs.EntityID
	lastSelectedSquad ecs.EntityID

	// Encounter tracking
	currentEncounterID ecs.EntityID // Tracks which encounter triggered this combat
}

func NewCombatMode(modeManager *core.UIModeManager) *CombatMode {
	cm := &CombatMode{
		logManager: NewCombatLogManager(),
		ui:         &CombatModeUI{}, // Initialize UI struct
	}
	cm.SetModeName("combat")
	cm.SetReturnMode("exploration") // ESC returns to exploration
	cm.ModeManager = modeManager
	return cm
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
	// Create combat service before ModeBuilder
	cm.combatService = combatservices.NewCombatService(ctx.ECSManager)

	// Create lifecycle manager (will be fully initialized after panels are built)
	cm.lifecycleManager = NewCombatLifecycleManager(
		ctx.ECSManager,
		nil, // Queries set after ModeBuilder
		cm.combatService,
		cm.logManager,
		nil, // combatLogArea set after panels are built
	)

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
	// Update lifecycle manager with Queries and combatLogArea now that they're available
	cm.lifecycleManager.queries = cm.Queries
	cm.lifecycleManager.combatLogArea = cm.ui.combatLogArea
	cm.lifecycleManager.SetBattleRecorder(cm.combatService.BattleRecorder)

	cm.actionHandler = NewCombatActionHandler(
		ctx.ModeCoordinator.GetBattleMapState(),
		cm.logManager,
		cm.Queries,
		cm.combatService,
		cm.ui.combatLogArea,
		cm.ModeManager,
	)

	cm.inputHandler = NewCombatInputHandler(
		cm.actionHandler,
		ctx.ModeCoordinator.GetBattleMapState(),
		cm.Queries,
	)

	cm.initializeUpdateComponents()

	// Initialize all visualization systems (renderers, threat managers, visualizers)
	// Cast GameMap from interface{} to *worldmap.GameMap
	gameMap := ctx.GameMap.(*worldmap.GameMap)
	cm.visualization = NewCombatVisualizationManager(ctx, cm.Queries, gameMap)

	return nil
}

func (cm *CombatMode) ensureUIFactoryInitialized() {
	if cm.uiFactory == nil {
		cm.uiFactory = gui.NewUIComponentFactory(cm.Queries, cm.PanelBuilders, cm.Layout)
	}
}

func (cm *CombatMode) buildTurnOrderPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	cm.ui.turnOrderPanel = cm.uiFactory.CreateCombatTurnOrderPanel()
	cm.ui.turnOrderLabel = builders.CreateLargeLabel("Initializing combat...")
	cm.ui.turnOrderPanel.AddChild(cm.ui.turnOrderLabel)

	return cm.ui.turnOrderPanel
}

func (cm *CombatMode) buildFactionInfoPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	cm.ui.factionInfoPanel = cm.uiFactory.CreateCombatFactionInfoPanel()
	cm.ui.factionInfoText = builders.CreateSmallLabel("Faction Info")
	cm.ui.factionInfoPanel.AddChild(cm.ui.factionInfoText)

	return cm.ui.factionInfoPanel
}

func (cm *CombatMode) buildSquadListPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	cm.ui.squadListPanel = cm.uiFactory.CreateCombatSquadListPanel()

	return cm.ui.squadListPanel
}

func (cm *CombatMode) buildSquadDetailPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	cm.ui.squadDetailPanel = cm.uiFactory.CreateCombatSquadDetailPanel()
	cm.ui.squadDetailText = builders.CreateSmallLabel("Select a squad\nto view details")
	cm.ui.squadDetailPanel.AddChild(cm.ui.squadDetailText)

	return cm.ui.squadDetailPanel
}

func (cm *CombatMode) buildLogPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	// Create log panel only if combat log is enabled
	if config.ENABLE_COMBAT_LOG {
		logContainer, logArea := cm.uiFactory.CreateCombatLogPanel()
		cm.ui.combatLogArea = logArea
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
	cm.ui.actionButtons = cm.uiFactory.CreateCombatActionButtons(
		cm.handleAttackClick,
		cm.handleMoveClick,
		cm.handleUndoMove,
		cm.handleRedoMove,
		cm.handleEndTurn,
		cm.handleFlee,
	)

	return cm.ui.actionButtons
}

func (cm *CombatMode) buildLayerStatusPanel() *widget.Container {
	cm.ensureUIFactoryInitialized()

	// Create a small panel for layer status display
	panelWidth := int(float64(cm.Layout.ScreenWidth) * 0.15)   // 15% of screen width
	panelHeight := int(float64(cm.Layout.ScreenHeight) * 0.08) // 8% of screen height

	cm.ui.layerStatusPanel = builders.CreatePanelWithConfig(builders.PanelConfig{
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
	cm.ui.layerStatusPanel.GetWidget().LayoutData = builders.AnchorEndStart(rightPad, topPad)

	// Create status text (initially hidden)
	cm.ui.layerStatusText = builders.CreateSmallLabel("")
	cm.ui.layerStatusPanel.AddChild(cm.ui.layerStatusText)

	// Hide panel initially (shown when visualizer is active)
	cm.ui.layerStatusPanel.GetWidget().Visibility = widget.Visibility_Hide

	return cm.ui.layerStatusPanel
}

// updateLayerStatusWidget updates the layer status panel visibility and text
func (cm *CombatMode) updateLayerStatusWidget() {
	layerViz := cm.visualization.GetLayerVisualizer()
	if cm.ui.layerStatusPanel == nil || cm.ui.layerStatusText == nil || layerViz == nil {
		return
	}

	if layerViz.IsActive() {
		// Show panel and update text with current mode info
		modeInfo := layerViz.GetCurrentModeInfo()
		statusText := fmt.Sprintf("LAYER VIEW\n%s\n%s", modeInfo.Name, modeInfo.ColorKey)
		cm.ui.layerStatusText.Label = statusText
		cm.ui.layerStatusPanel.GetWidget().Visibility = widget.Visibility_Show
	} else {
		// Hide panel when visualizer is inactive
		cm.ui.layerStatusPanel.GetWidget().Visibility = widget.Visibility_Hide
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
	cm.ui.turnOrderComponent = guicomponents.NewTextDisplayComponent(
		cm.ui.turnOrderLabel,
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
	cm.ui.factionInfoComponent = guicomponents.NewDetailPanelComponent(
		cm.ui.factionInfoText,
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
	cm.ui.squadDetailComponent = guicomponents.NewDetailPanelComponent(
		cm.ui.squadDetailText,
		cm.Queries,
		nil, // Use default formatter
	)

	// Squad list component - filter for current faction squads (during player's turn only)
	// Extracted filter logic to separate method to eliminate inline duplication
	cm.ui.squadListComponent = guicomponents.NewSquadListComponent(
		cm.ui.squadListPanel,
		cm.Queries,
		cm.makeCurrentFactionSquadFilter(),
		func(squadID ecs.EntityID) {
			cm.actionHandler.SelectSquad(squadID)
			cm.ui.squadDetailComponent.ShowSquad(squadID)
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
	cm.logManager.UpdateTextArea(cm.ui.combatLogArea, "Fleeing from combat...")

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
		cm.logManager.UpdateTextArea(cm.ui.combatLogArea, fmt.Sprintf("Error ending turn: %s", err.Error()))
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

	// Log turn change
	turnMessage := cm.formatTurnMessage(currentFactionID, round)
	cm.logManager.UpdateTextArea(cm.ui.combatLogArea, turnMessage)

	// Clear selection when turn changes
	cm.Context.ModeCoordinator.GetBattleMapState().Reset()

	// Update UI displays using components
	cm.ui.turnOrderComponent.Refresh()
	cm.ui.factionInfoComponent.ShowFaction(currentFactionID)
	cm.ui.squadListComponent.Refresh()
	cm.ui.squadDetailComponent.SetText("Select a squad\nto view details")

	cm.visualization.UpdateThreatManagers()

	// Update threat evaluator for layer visualization
	cm.visualization.UpdateThreatEvaluator(round)

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
		cm.logManager.UpdateTextArea(cm.ui.combatLogArea, fmt.Sprintf("%s (AI) executed actions", factionData.Name))
	} else {
		cm.logManager.UpdateTextArea(cm.ui.combatLogArea, fmt.Sprintf("%s (AI) has no valid actions", factionData.Name))
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
		cm.logManager.UpdateTextArea(cm.ui.combatLogArea, fmt.Sprintf("Error ending AI turn: %s", err.Error()))
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

	// Log turn change
	turnMessage := cm.formatTurnMessage(newFactionID, round)
	cm.logManager.UpdateTextArea(cm.ui.combatLogArea, turnMessage)

	cm.ui.turnOrderComponent.Refresh()
	cm.ui.factionInfoComponent.ShowFaction(newFactionID)
	cm.ui.squadListComponent.Refresh()

	cm.visualization.UpdateThreatManagers()

	// Recursively execute next AI turn if needed
	cm.executeAITurnIfNeeded()
}

func (cm *CombatMode) SetupEncounter(fromMode core.UIMode) error {
	// Get encounter ID from BattleMapState
	encounterID := ecs.EntityID(0)
	if cm.Context.ModeCoordinator != nil {
		battleMapState := cm.Context.ModeCoordinator.GetBattleMapState()
		encounterID = battleMapState.TriggeredEncounterID
	}

	// Get player start position for squad spawning
	playerStartPos := coords.LogicalPosition{X: 50, Y: 40}
	if cm.Context.PlayerData != nil && cm.Context.PlayerData.Pos != nil {
		playerStartPos = *cm.Context.PlayerData.Pos
	}

	// Delegate to lifecycle manager
	var err error
	cm.currentEncounterID, err = cm.lifecycleManager.SetupEncounter(encounterID, playerStartPos)
	return err
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
		cm.logManager.UpdateTextArea(cm.ui.combatLogArea, "=== COMBAT STARTED ===")

		cm.SetupEncounter(fromMode)

		// Start battle recording if enabled
		cm.lifecycleManager.StartBattleRecording(1)

		// Initialize combat factions
		if _, err := cm.lifecycleManager.InitializeCombatFactions(); err != nil {
			return fmt.Errorf("error initializing combat factions: %w", err)
		}

	}

	// Always refresh UI displays (whether fresh or returning from animation)
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	if currentFactionID != 0 {
		cm.ui.turnOrderComponent.Refresh()
		cm.ui.factionInfoComponent.ShowFaction(currentFactionID)
		cm.ui.squadListComponent.Refresh()
	}

	return nil
}

func (cm *CombatMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Combat Mode")

	// Check if returning to exploration (not going to animation mode)
	isToAnimation := toMode != nil && toMode.GetModeName() == "combat_animation"

	// Handle lifecycle cleanup when returning to exploration
	if !isToAnimation {
		// Mark encounter as defeated if player won
		cm.lifecycleManager.MarkEncounterDefeated(cm.currentEncounterID)

		// Clean up all combat entities
		cm.lifecycleManager.CleanupCombatEntities()

		// Export battle log if enabled
		if err := cm.lifecycleManager.ExportBattleLog(); err != nil {
			fmt.Printf("Failed to export combat log: %v\n", err)
		}
	}

	// Clear all visualizations
	cm.visualization.ClearAllVisualizations()

	// Clear combat log for next battle
	cm.logManager.Clear()
	return nil
}

func (cm *CombatMode) Update(deltaTime float64) error {
	// Only update UI displays when state changes (GUI_PERFORMANCE_ANALYSIS.md)
	// This avoids expensive text measurement on every frame (~10-15s CPU savings)
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
	if cm.lastFactionID != currentFactionID {
		cm.ui.turnOrderComponent.Refresh()
		cm.lastFactionID = currentFactionID
		if cm.lastFactionID != 0 {
			cm.ui.factionInfoComponent.ShowFaction(cm.lastFactionID)
		}
	}

	battleState := cm.Context.ModeCoordinator.GetBattleMapState()
	if cm.lastSelectedSquad != battleState.SelectedSquadID {
		cm.lastSelectedSquad = battleState.SelectedSquadID
		if cm.lastSelectedSquad != 0 {
			cm.ui.squadDetailComponent.ShowSquad(cm.lastSelectedSquad)
		}
	}

	// Update visualizations if active
	currentRound := cm.combatService.TurnManager.GetCurrentRound()
	playerPos := *cm.Context.PlayerData.Pos
	viewportSize := 30 // Process 30x30 tile area around player

	cm.visualization.UpdateDangerVisualization(currentFactionID, currentRound, playerPos, viewportSize)
	cm.visualization.UpdateLayerVisualization(currentFactionID, currentRound, playerPos, viewportSize)

	return nil
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
	cm.visualization.GetHighlightRenderer().Render(screen, playerPos, currentFactionID, selectedSquad)

	// Render valid movement tiles (only in move mode)
	if battleState.InMoveMode {
		validTiles := cm.getValidMoveTiles()
		if len(validTiles) > 0 {
			cm.visualization.GetMovementRenderer().Render(screen, playerPos, validTiles)
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

		dangerViz := cm.visualization.GetDangerVisualizer()
		if shiftPressed {
			// Shift+H: Switch between enemy/player threat view
			dangerViz.SwitchView()
			viewName := "Enemy Threats"
			if dangerViz.GetViewMode() == behavior.ViewPlayerThreats {
				viewName = "Player Threats"
			}
			cm.logManager.UpdateTextArea(cm.ui.combatLogArea, fmt.Sprintf("Switched to %s view", viewName))
		} else {
			// H alone: Toggle visualization on/off
			dangerViz.Toggle()
			status := "enabled"
			if !dangerViz.IsActive() {
				status = "disabled"
			}
			cm.logManager.UpdateTextArea(cm.ui.combatLogArea, fmt.Sprintf("Danger visualization %s", status))
		}
		return true
	}

	// Left Control key to cycle between danger/expected damage metrics (when visualizer active)
	if inputState.KeysJustPressed[ebiten.KeyControlLeft] {
		dangerViz := cm.visualization.GetDangerVisualizer()
		if dangerViz.IsActive() {
			dangerViz.CycleMetric()
			metricName := "Danger"
			if dangerViz.GetMetricMode() == behavior.MetricExpectedDamage {
				metricName = "Expected Damage"
			}
			cm.logManager.UpdateTextArea(cm.ui.combatLogArea, fmt.Sprintf("Switched to %s metric", metricName))
			return true
		}
	}

	// L key to toggle layer visualizer
	if inputState.KeysJustPressed[ebiten.KeyL] {
		layerViz := cm.visualization.GetLayerVisualizer()
		if layerViz == nil {
			return true
		}

		// Check if Shift key is pressed
		shiftPressed := inputState.KeysPressed[ebiten.KeyShift] ||
			inputState.KeysPressed[ebiten.KeyShiftLeft] ||
			inputState.KeysPressed[ebiten.KeyShiftRight]

		if shiftPressed {
			// Shift+L: Cycle through layer modes
			layerViz.CycleMode()
			modeInfo := layerViz.GetCurrentModeInfo()
			cm.logManager.UpdateTextArea(cm.ui.combatLogArea,
				fmt.Sprintf("Layer: %s (%s)", modeInfo.Name, modeInfo.ColorKey))
		} else {
			// L alone: Toggle visualization on/off
			layerViz.Toggle()
			status := "enabled"
			if !layerViz.IsActive() {
				status = "disabled"
			}
			cm.logManager.UpdateTextArea(cm.ui.combatLogArea,
				fmt.Sprintf("Layer visualization %s", status))
		}
		// Update the status widget to reflect current mode
		cm.updateLayerStatusWidget()
		return true
	}

	return false
}
