package guicombat

import (
	"game_main/gui"
	"game_main/gui/guicomponents"
	"game_main/gui/guimodes"
	"game_main/gui/widgets"
	"game_main/gui/core"
	"fmt"
	"game_main/combat"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatMode provides focused UI for turn-based squad combat
type CombatMode struct {
	gui.BaseMode // Embed common mode infrastructure

	// Managers
	logManager     *CombatLogManager
	actionHandler  *CombatActionHandler
	inputHandler   *CombatInputHandler
	uiFactory      *CombatUIFactory
	combatService  *combat.CombatService

	// UI panels and widgets
	turnOrderPanel   *widget.Container
	factionInfoPanel *widget.Container
	squadListPanel   *widget.Container
	squadDetailPanel *widget.Container
	combatLogArea    *widget.TextArea
	actionButtons    *widget.Container

	// UI text labels
	turnOrderLabel   *widget.Text
	factionInfoText  *widget.Text
	squadDetailText  *widget.Text

	// UI update components
	squadListComponent    *guicomponents.SquadListComponent
	squadDetailComponent  *guicomponents.DetailPanelComponent
	factionInfoComponent  *guicomponents.DetailPanelComponent
	turnOrderComponent    *guicomponents.TextDisplayComponent

	// Rendering systems
	movementRenderer  *guimodes.MovementTileRenderer
	highlightRenderer *guimodes.SquadHighlightRenderer
}

func NewCombatMode(modeManager *core.UIModeManager) *CombatMode {
	cm := &CombatMode{
		logManager: NewCombatLogManager(),
	}
	cm.SetModeName("combat")
	cm.ModeManager = modeManager
	return cm
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
	// Initialize common mode infrastructure
	cm.InitializeBase(ctx)

	// Create combat service (owns TurnManager, FactionManager, MovementSystem)
	cm.combatService = combat.NewCombatService(ctx.ECSManager)

	// Create UI factory
	cm.uiFactory = NewCombatUIFactory(cm.Queries, cm.PanelBuilders, cm.Layout)

	// Build UI panels using factory
	cm.buildUILayout()

	// Create combat action handler with service
	cm.actionHandler = NewCombatActionHandler(
		ctx.ModeCoordinator.GetBattleMapState(),
		cm.logManager,
		cm.Queries,
		cm.combatService,
		cm.combatLogArea,
	)

	// Create combat input handler
	cm.inputHandler = NewCombatInputHandler(
		cm.actionHandler,
		ctx.ModeCoordinator.GetBattleMapState(),
		cm.Queries,
	)

	// Initialize UI update components
	cm.initializeUpdateComponents()

	// Initialize rendering systems
	cm.movementRenderer = guimodes.NewMovementTileRenderer()
	cm.highlightRenderer = guimodes.NewSquadHighlightRenderer(cm.Queries)

	return nil
}

func (cm *CombatMode) buildUILayout() {
	// Build UI panels using factory
	cm.turnOrderPanel = cm.uiFactory.CreateTurnOrderPanel()
	cm.turnOrderLabel = widgets.CreateLargeLabel("Initializing combat...")
	cm.turnOrderPanel.AddChild(cm.turnOrderLabel)
	cm.RootContainer.AddChild(cm.turnOrderPanel)

	cm.factionInfoPanel = cm.uiFactory.CreateFactionInfoPanel()
	cm.factionInfoText = widgets.CreateSmallLabel("Faction Info")
	cm.factionInfoPanel.AddChild(cm.factionInfoText)
	cm.RootContainer.AddChild(cm.factionInfoPanel)

	cm.squadListPanel = cm.uiFactory.CreateSquadListPanel()
	cm.RootContainer.AddChild(cm.squadListPanel)

	cm.squadDetailPanel = cm.uiFactory.CreateSquadDetailPanel()
	cm.squadDetailText = widgets.CreateSmallLabel("Select a squad\nto view details")
	cm.squadDetailPanel.AddChild(cm.squadDetailText)
	cm.RootContainer.AddChild(cm.squadDetailPanel)

	// Create log panel
	logContainer, logArea := cm.uiFactory.CreateLogPanel()
	cm.combatLogArea = logArea
	cm.RootContainer.AddChild(logContainer)

	// Create action buttons
	cm.actionButtons = cm.uiFactory.CreateActionButtons(
		cm.handleAttackClick,
		cm.handleMoveClick,
		cm.handleEndTurn,
		cm.handleFlee,
	)
	cm.RootContainer.AddChild(cm.actionButtons)
}

// Button click handlers that delegate to action handler
func (cm *CombatMode) handleAttackClick() {
	cm.actionHandler.ToggleAttackMode()
}

func (cm *CombatMode) handleMoveClick() {
	cm.actionHandler.ToggleMoveMode()
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

			round := cm.combatService.GetTurnManager().GetCurrentRound()
			factionName := cm.Queries.GetFactionName(currentFactionID)

			// Add indicator if player's turn
			playerIndicator := ""
			if cm.Queries.IsPlayerFaction(currentFactionID) {
				playerIndicator = " >>> YOUR TURN <<<"
			}

			return fmt.Sprintf("Round %d | %s%s", round, factionName, playerIndicator)
		},
	)

	// Faction info component - displays squad count and mana
	cm.factionInfoComponent = guicomponents.NewDetailPanelComponent(
		cm.factionInfoText,
		cm.Queries,
		func(data interface{}) string {
			factionInfo := data.(*guicomponents.FactionInfo)
			infoText := fmt.Sprintf("%s\n", factionInfo.Name)
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
		if !cm.Queries.IsPlayerFaction(currentFactionID) {
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
	// End current faction's turn
	if err := cm.combatService.GetTurnManager().EndTurn(); err != nil {
		cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Error ending turn: %v", err))
		return
	}

	// Get new faction info
	currentFactionID := cm.combatService.GetCurrentFaction()
	round := cm.combatService.GetTurnManager().GetCurrentRound()

	// Get faction name
	factionName := cm.Queries.GetFactionName(currentFactionID)

	cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("=== Round %d: %s's Turn ===", round, factionName))

	// Clear selection when turn changes
	cm.Context.ModeCoordinator.GetBattleMapState().Reset()

	// Update UI displays using components
	cm.turnOrderComponent.Refresh()
	cm.factionInfoComponent.ShowFaction(currentFactionID)
	cm.squadListComponent.Refresh()
	cm.squadDetailComponent.SetText("Select a squad\nto view details")
}

func (cm *CombatMode) Enter(fromMode core.UIMode) error {
	fmt.Println("Entering Combat Mode")
	cm.logManager.UpdateTextArea(cm.combatLogArea, "=== COMBAT STARTED ===")

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
		factionName := cm.Queries.GetFactionName(currentFactionID)
		cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Round 1: %s goes first!", factionName))

		// Update displays using components
		cm.turnOrderComponent.Refresh()
		cm.factionInfoComponent.ShowFaction(currentFactionID)
		cm.squadListComponent.Refresh()
	} else {
		cm.logManager.UpdateTextArea(cm.combatLogArea, "No factions found - combat cannot start")
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
	// Update UI displays each frame using components
	cm.turnOrderComponent.Refresh()

	currentFactionID := cm.combatService.GetCurrentFaction()
	if currentFactionID != 0 {
		cm.factionInfoComponent.ShowFaction(currentFactionID)
	}

	selectedSquad := cm.Context.ModeCoordinator.GetBattleMapState().SelectedSquadID
	if selectedSquad != 0 {
		cm.squadDetailComponent.ShowSquad(selectedSquad)
	}

	return nil
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
		validTiles := battleState.ValidMoveTiles
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
