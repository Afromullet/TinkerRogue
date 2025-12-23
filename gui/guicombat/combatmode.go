package guicombat

import (
	"fmt"
	"game_main/behavior"
	"game_main/combat/combatservices"
	"game_main/config"
	"game_main/coords"
	"game_main/gui"
	"game_main/gui/builders"
	"game_main/gui/core"
	"game_main/gui/guicomponents"
	"game_main/gui/guimodes"
	"game_main/gui/widgets"

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

	// UI text labels
	turnOrderLabel  *widget.Text
	factionInfoText *widget.Text
	squadDetailText *widget.Text

	// UI update components
	squadListComponent   *guicomponents.SquadListComponent
	squadDetailComponent *guicomponents.DetailPanelComponent
	factionInfoComponent *guicomponents.DetailPanelComponent
	turnOrderComponent   *guicomponents.TextDisplayComponent

	// Rendering systems
	movementRenderer  *guimodes.MovementTileRenderer
	highlightRenderer *guimodes.SquadHighlightRenderer

	// State tracking for UI updates (GUI_PERFORMANCE_ANALYSIS.md)
	lastFactionID     ecs.EntityID
	lastSelectedSquad ecs.EntityID
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
			currentFactionID := cm.combatService.GetCurrentFaction()
			if currentFactionID == 0 {
				return "No active combat"
			}

			round := cm.combatService.GetCurrentRound()
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
		currentFactionID := cm.combatService.GetCurrentFaction()
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
	if exploreMode, exists := cm.ModeManager.GetMode("exploration"); exists {
		cm.ModeManager.RequestTransition(exploreMode, "Fled from combat")
	}
}

func (cm *CombatMode) handleEndTurn() {
	// Clear movement history when ending turn (can't undo moves from previous turns)
	cm.actionHandler.ClearMoveHistory()

	// End current faction's turn using service
	result := cm.combatService.EndTurn()
	if !result.Success {
		cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Error ending turn: %s", result.Error))
		return
	}

	// Invalidate all squad caches since turn changed (all action states reset)
	cm.Queries.MarkAllSquadsDirty()

	// Get new faction info from result
	currentFactionID := result.NewFaction
	round := result.NewRound

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

	behavior.ThreatLevelManager.UpdateAllFactions()

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

		// Enter new mode
		if cm.initialzieCombatFactions() != nil {
			return fmt.Errorf("Error initializing combat factions")
		}

	}

	// Always refresh UI displays (whether fresh or returning from animation)
	currentFactionID := cm.combatService.GetCurrentFaction()
	if currentFactionID != 0 {
		cm.turnOrderComponent.Refresh()
		cm.factionInfoComponent.ShowFaction(currentFactionID)
		cm.squadListComponent.Refresh()
	}

	return nil
}

func (cm *CombatMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Combat Mode")
	// Clear combat log for next battle
	cm.logManager.Clear()
	return nil
}

func (cm *CombatMode) Update(deltaTime float64) error {
	// Only update UI displays when state changes (GUI_PERFORMANCE_ANALYSIS.md)
	// This avoids expensive text measurement on every frame (~10-15s CPU savings)
	currentFactionID := cm.combatService.GetCurrentFaction()
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
		currentFactionID := cm.combatService.GetCurrentFaction()
		factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID, cm.Queries.ECSManager)
		factionName := "Unknown"
		if factionData != nil {
			factionName = factionData.Name
		}
		cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Round 1: %s goes first!", factionName))
	} else {
		cm.logManager.UpdateTextArea(cm.combatLogArea, "No factions found - combat cannot start")
	}

	//Create the initial Faction Threat Level Manager and add all factions.
	behavior.ThreatLevelManager = behavior.NewFactionThreatLevelManager(cm.Context.ECSManager)
	for _, IDs := range factionIDs {

		behavior.ThreatLevelManager.AddFaction(IDs)

	}

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

	// Compute from ECS game state via combat service
	tiles := cm.combatService.GetValidMovementTiles(battleState.SelectedSquadID)
	if tiles == nil {
		return []coords.LogicalPosition{}
	}

	return tiles
}

func (cm *CombatMode) Render(screen *ebiten.Image) {
	playerPos := *cm.Context.PlayerData.Pos
	currentFactionID := cm.combatService.GetCurrentFaction()
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
	cm.inputHandler.SetCurrentFactionID(cm.combatService.GetCurrentFaction())

	// Handle combat-specific input through input handler
	if cm.inputHandler.HandleInput(inputState) {
		return true
	}

	// Space to end turn (handled separately here)
	if inputState.KeysJustPressed[ebiten.KeySpace] {
		cm.handleEndTurn()
		return true
	}

	return false
}
