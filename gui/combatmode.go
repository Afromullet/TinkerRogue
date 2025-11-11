package gui

import (
	"fmt"
	"game_main/combat"
	"game_main/common"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatMode provides focused UI for turn-based squad combat
type CombatMode struct {
	BaseMode // Embed common mode infrastructure

	// Managers
	logManager      *CombatLogManager
	stateManager    *CombatStateManager
	actionHandler   *CombatActionHandler
	inputHandler    *CombatInputHandler
	uiFactory       *CombatUIFactory

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

	// Combat system managers
	turnManager    *combat.TurnManager
	factionManager *combat.FactionManager
	movementSystem *combat.MovementSystem

	// UI update components
	squadListComponent    *SquadListComponent
	squadDetailComponent  *DetailPanelComponent
	factionInfoComponent  *DetailPanelComponent
	turnOrderComponent    *TextDisplayComponent

	// Rendering systems
	movementRenderer  *MovementTileRenderer
	highlightRenderer *SquadHighlightRenderer
}

func NewCombatMode(modeManager *UIModeManager) *CombatMode {
	return &CombatMode{
		BaseMode: BaseMode{
			modeManager: modeManager,
			modeName:    "combat",
			returnMode:  "exploration",
		},
		logManager:   NewCombatLogManager(),
		stateManager: NewCombatStateManager(),
	}
}

func (cm *CombatMode) Initialize(ctx *UIContext) error {
	// Initialize common mode infrastructure
	cm.InitializeBase(ctx)

	// Initialize combat system managers
	cm.turnManager = combat.NewTurnManager(ctx.ECSManager)
	cm.factionManager = combat.NewFactionManager(ctx.ECSManager)
	cm.movementSystem = combat.NewMovementSystem(ctx.ECSManager, common.GlobalPositionSystem)

	// Create UI factory
	cm.uiFactory = NewCombatUIFactory(cm.queries, cm.panelBuilders, cm.layout)

	// Build UI panels using factory
	cm.buildUILayout()

	// Create combat action handler
	cm.actionHandler = NewCombatActionHandler(
		cm.stateManager,
		cm.logManager,
		cm.queries,
		ctx.ECSManager,
		cm.turnManager,
		cm.factionManager,
		cm.movementSystem,
		cm.combatLogArea,
	)

	// Create combat input handler
	cm.inputHandler = NewCombatInputHandler(
		cm.actionHandler,
		cm.stateManager,
		cm.queries,
	)

	// Initialize UI update components
	cm.initializeUpdateComponents()

	// Initialize rendering systems
	cm.movementRenderer = NewMovementTileRenderer()
	cm.highlightRenderer = NewSquadHighlightRenderer(cm.queries)

	return nil
}

func (cm *CombatMode) buildUILayout() {
	// Build UI panels using factory
	cm.turnOrderPanel = cm.uiFactory.CreateTurnOrderPanel()
	cm.turnOrderLabel = cm.uiFactory.CreateTurnOrderLabel("Initializing combat...")
	cm.turnOrderPanel.AddChild(cm.turnOrderLabel)
	cm.rootContainer.AddChild(cm.turnOrderPanel)

	cm.factionInfoPanel = cm.uiFactory.CreateFactionInfoPanel()
	cm.factionInfoText = cm.uiFactory.CreateFactionInfoText("Faction Info")
	cm.factionInfoPanel.AddChild(cm.factionInfoText)
	cm.rootContainer.AddChild(cm.factionInfoPanel)

	cm.squadListPanel = cm.uiFactory.CreateSquadListPanel()
	cm.rootContainer.AddChild(cm.squadListPanel)

	cm.squadDetailPanel = cm.uiFactory.CreateSquadDetailPanel()
	cm.squadDetailText = cm.uiFactory.CreateSquadDetailText("Select a squad\nto view details")
	cm.squadDetailPanel.AddChild(cm.squadDetailText)
	cm.rootContainer.AddChild(cm.squadDetailPanel)

	// Create log panel
	logContainer, logArea := cm.uiFactory.CreateLogPanel()
	cm.combatLogArea = logArea
	cm.rootContainer.AddChild(logContainer)

	// Create action buttons
	cm.actionButtons = cm.uiFactory.CreateActionButtons(
		cm.handleAttackClick,
		cm.handleMoveClick,
		cm.handleEndTurn,
		cm.handleFlee,
	)
	cm.rootContainer.AddChild(cm.actionButtons)
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
	cm.turnOrderComponent = NewTextDisplayComponent(
		cm.turnOrderLabel,
		func() string {
			currentFactionID := cm.turnManager.GetCurrentFaction()
			if currentFactionID == 0 {
				return "No active combat"
			}

			round := cm.turnManager.GetCurrentRound()
			factionName := cm.queries.GetFactionName(currentFactionID)

			// Add indicator if player's turn
			playerIndicator := ""
			if cm.queries.IsPlayerFaction(currentFactionID) {
				playerIndicator = " >>> YOUR TURN <<<"
			}

			return fmt.Sprintf("Round %d | %s%s", round, factionName, playerIndicator)
		},
	)

	// Faction info component - displays squad count and mana
	cm.factionInfoComponent = NewDetailPanelComponent(
		cm.factionInfoText,
		cm.queries,
		func(data interface{}) string {
			factionInfo := data.(*FactionInfo)
			infoText := fmt.Sprintf("%s\n", factionInfo.Name)
			infoText += fmt.Sprintf("Squads: %d/%d\n", factionInfo.AliveSquadCount, len(factionInfo.SquadIDs))
			infoText += fmt.Sprintf("Mana: %d/%d", factionInfo.CurrentMana, factionInfo.MaxMana)
			return infoText
		},
	)

	// Squad detail component - displays selected squad details
	cm.squadDetailComponent = NewDetailPanelComponent(
		cm.squadDetailText,
		cm.queries,
		nil, // Use default formatter
	)

	// Squad list component - filter for player faction squads
	cm.squadListComponent = NewSquadListComponent(
		cm.squadListPanel,
		cm.queries,
		func(info *SquadInfo) bool {
			currentFactionID := cm.turnManager.GetCurrentFaction()
			if currentFactionID == 0 {
				return false
			}
			// Only show squads if it's player's turn
			if !cm.queries.IsPlayerFaction(currentFactionID) {
				return false
			}
			return !info.IsDestroyed && info.FactionID == currentFactionID
		},
		func(squadID ecs.EntityID) {
			cm.actionHandler.SelectSquad(squadID)
			cm.squadDetailComponent.ShowSquad(squadID)
		},
	)
}

func (cm *CombatMode) handleFlee() {
	if exploreMode, exists := cm.modeManager.GetMode("exploration"); exists {
		cm.modeManager.RequestTransition(exploreMode, "Fled from combat")
	}
}

func (cm *CombatMode) handleEndTurn() {
	// End current faction's turn
	if err := cm.turnManager.EndTurn(); err != nil {
		cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Error ending turn: %v", err))
		return
	}

	// Get new faction info
	currentFactionID := cm.turnManager.GetCurrentFaction()
	round := cm.turnManager.GetCurrentRound()

	// Get faction name
	factionName := cm.queries.GetFactionName(currentFactionID)

	cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("=== Round %d: %s's Turn ===", round, factionName))

	// Clear selection when turn changes using state manager
	cm.stateManager.Reset()

	// Update UI displays using components
	cm.turnOrderComponent.Refresh()
	cm.factionInfoComponent.ShowFaction(currentFactionID)
	cm.squadListComponent.Refresh()
	cm.squadDetailComponent.SetText("Select a squad\nto view details")
}

func (cm *CombatMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Combat Mode")
	cm.logManager.UpdateTextArea(cm.combatLogArea, "=== COMBAT STARTED ===")

	// Collect all factions using query service
	factionIDs := cm.queries.GetAllFactions()

	// Initialize combat with all factions
	if len(factionIDs) > 0 {
		if err := cm.turnManager.InitializeCombat(factionIDs); err != nil {
			cm.logManager.UpdateTextArea(cm.combatLogArea, fmt.Sprintf("Error initializing combat: %v", err))
			return err
		}

		// Log initial faction
		currentFactionID := cm.turnManager.GetCurrentFaction()
		factionName := cm.queries.GetFactionName(currentFactionID)
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

func (cm *CombatMode) Exit(toMode UIMode) error {
	fmt.Println("Exiting Combat Mode")
	// Clear combat log for next battle
	cm.logManager.Clear()
	return nil
}

func (cm *CombatMode) Update(deltaTime float64) error {
	// Update UI displays each frame using components
	cm.turnOrderComponent.Refresh()

	currentFactionID := cm.turnManager.GetCurrentFaction()
	if currentFactionID != 0 {
		cm.factionInfoComponent.ShowFaction(currentFactionID)
	}

	selectedSquad := cm.stateManager.GetSelectedSquad()
	if selectedSquad != 0 {
		cm.squadDetailComponent.ShowSquad(selectedSquad)
	}

	return nil
}

func (cm *CombatMode) Render(screen *ebiten.Image) {
	playerPos := *cm.context.PlayerData.Pos
	currentFactionID := cm.turnManager.GetCurrentFaction()
	selectedSquad := cm.stateManager.GetSelectedSquad()

	// Render squad highlights (always shown)
	cm.highlightRenderer.Render(screen, playerPos, currentFactionID, selectedSquad)

	// Render valid movement tiles (only in move mode)
	if cm.stateManager.IsMoveMode() {
		validTiles := cm.stateManager.GetValidMoveTiles()
		if len(validTiles) > 0 {
			cm.movementRenderer.Render(screen, playerPos, validTiles)
		}
	}
}

// renderMovementTiles and renderAllSquadHighlights removed - now using MovementTileRenderer and SquadHighlightRenderer

func (cm *CombatMode) HandleInput(inputState *InputState) bool {
	// Handle common input (ESC key to flee combat)
	if cm.HandleCommonInput(inputState) {
		return true
	}

	// Update input handler with player position and faction info
	cm.inputHandler.SetPlayerPosition(cm.context.PlayerData.Pos)
	cm.inputHandler.SetCurrentFactionID(cm.turnManager.GetCurrentFaction())

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
