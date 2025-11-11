package gui

import (
	"fmt"
	"game_main/combat"
	"game_main/common"
	"game_main/coords"
	"game_main/graphics"
	"game_main/squads"
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatMode provides focused UI for turn-based squad combat
type CombatMode struct {
	BaseMode // Embed common mode infrastructure

	turnOrderPanel   *widget.Container
	factionInfoPanel *widget.Container
	combatLogArea    *widget.TextArea
	actionButtons    *widget.Container

	combatLog              []string // Store combat messages
	messageCountSinceTrim int      // Track messages for periodic trimming

	// Combat system managers
	turnManager    *combat.TurnManager
	factionManager *combat.FactionManager
	movementSystem *combat.MovementSystem

	// UI state
	turnOrderLabel    *widget.Text
	factionInfoText   *widget.Text
	squadListPanel    *widget.Container
	squadDetailPanel  *widget.Container
	squadDetailText   *widget.Text
	attackButton      *widget.Button
	moveButton        *widget.Button

	// Combat state
	selectedSquadID  ecs.EntityID
	selectedTargetID ecs.EntityID
	inAttackMode     bool
	inMoveMode       bool
	validMoveTiles   []coords.LogicalPosition

	// UI update components
	squadListComponent  *SquadListComponent
	squadDetailComponent *DetailPanelComponent
	factionInfoComponent *DetailPanelComponent
	turnOrderComponent   *TextDisplayComponent

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
		combatLog: make([]string, 0, 100),
	}
}

func (cm *CombatMode) Initialize(ctx *UIContext) error {
	// Initialize common mode infrastructure
	cm.InitializeBase(ctx)

	// Initialize combat managers
	cm.turnManager = combat.NewTurnManager(ctx.ECSManager)
	cm.factionManager = combat.NewFactionManager(ctx.ECSManager)
	cm.movementSystem = combat.NewMovementSystem(ctx.ECSManager, common.GlobalPositionSystem)

	// Build turn order panel (top-center)
	cm.turnOrderPanel = cm.panelBuilders.BuildPanel(
		TopCenter(),
		Size(0.4, 0.08),
		Padding(0.01),
		HorizontalRowLayout(),
	)
	cm.turnOrderLabel = CreateTextWithConfig(TextConfig{
		Text:     "Initializing combat...",
		FontFace: LargeFace,
		Color:    color.White,
	})
	cm.turnOrderPanel.AddChild(cm.turnOrderLabel)
	cm.rootContainer.AddChild(cm.turnOrderPanel)

	// Build faction info panel (top-left)
	cm.factionInfoPanel = cm.panelBuilders.BuildPanel(
		TopLeft(),
		Size(0.15, 0.12),
		Padding(0.01),
		RowLayout(),
	)
	cm.factionInfoText = CreateTextWithConfig(TextConfig{
		Text:     "Faction Info",
		FontFace: SmallFace,
		Color:    color.White,
	})
	cm.factionInfoPanel.AddChild(cm.factionInfoText)
	cm.rootContainer.AddChild(cm.factionInfoPanel)

	// Build squad list panel (left-side)
	cm.squadListPanel = cm.panelBuilders.BuildPanel(
		LeftCenter(),
		Size(0.15, 0.5),
		Padding(0.01),
		RowLayout(),
	)
	listLabel := CreateTextWithConfig(TextConfig{
		Text:     "Your Squads:",
		FontFace: SmallFace,
		Color:    color.White,
	})
	cm.squadListPanel.AddChild(listLabel)
	cm.rootContainer.AddChild(cm.squadListPanel)

	// Build squad detail panel (left-bottom)
	cm.squadDetailPanel = cm.panelBuilders.BuildPanel(
		LeftBottom(),
		Size(0.15, 0.25),
		CustomPadding(widget.Insets{
			Left:   int(float64(cm.layout.ScreenWidth) * 0.01),
			Bottom: int(float64(cm.layout.ScreenHeight) * 0.15),
		}),
		RowLayout(),
	)
	cm.squadDetailText = CreateTextWithConfig(TextConfig{
		Text:     "Select a squad\nto view details",
		FontFace: SmallFace,
		Color:    color.White,
	})
	cm.squadDetailPanel.AddChild(cm.squadDetailText)
	cm.rootContainer.AddChild(cm.squadDetailPanel)

	// Build combat log (right-side) using BuildPanel
	logContainer := cm.panelBuilders.BuildPanel(
		RightCenter(),
		Size(0.2, 0.85),
		Padding(0.01),
		AnchorLayout(),
	)

	// Create combat log text area
	logWidth := int(float64(cm.layout.ScreenWidth) * 0.2)
	logHeight := cm.layout.ScreenHeight - int(float64(cm.layout.ScreenHeight)*0.15)
	cm.combatLogArea = CreateTextAreaWithConfig(TextAreaConfig{
		MinWidth:  logWidth - 20,
		MinHeight: logHeight - 20,
		FontColor: color.White,
	})
	cm.combatLogArea.SetText("Combat started!\n")
	logContainer.AddChild(cm.combatLogArea)
	cm.rootContainer.AddChild(logContainer)

	// Build combat UI layout
	cm.buildActionButtons()

	// Initialize UI update components
	cm.initializeUpdateComponents()

	// Initialize rendering systems
	cm.movementRenderer = NewMovementTileRenderer()
	cm.highlightRenderer = NewSquadHighlightRenderer(cm.queries)

	return nil
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
			cm.selectSquad(squadID)
		},
	)
}

func (cm *CombatMode) buildActionButtons() {
	// Create action buttons
	cm.attackButton = CreateButtonWithConfig(ButtonConfig{
		Text: "Attack (A)",
		OnClick: func() {
			cm.toggleAttackMode()
		},
	})

	cm.moveButton = CreateButtonWithConfig(ButtonConfig{
		Text: "Move (M)",
		OnClick: func() {
			cm.toggleMoveMode()
		},
	})

	endTurnBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "End Turn (Space)",
		OnClick: func() {
			cm.handleEndTurn()
		},
	})

	fleeBtn := CreateButtonWithConfig(ButtonConfig{
		Text: "Flee (ESC)",
		OnClick: func() {
			if exploreMode, exists := cm.modeManager.GetMode("exploration"); exists {
				cm.modeManager.RequestTransition(exploreMode, "Fled from combat")
			}
		},
	})

	// Build action buttons container using BuildPanel
	cm.actionButtons = cm.panelBuilders.BuildPanel(
		BottomCenter(),
		HorizontalRowLayout(),
		CustomPadding(widget.Insets{
			Bottom: int(float64(cm.layout.ScreenHeight) * 0.08),
		}),
	)

	cm.actionButtons.AddChild(cm.attackButton)
	cm.actionButtons.AddChild(cm.moveButton)
	cm.actionButtons.AddChild(endTurnBtn)
	cm.actionButtons.AddChild(fleeBtn)
	cm.rootContainer.AddChild(cm.actionButtons)
}

func (cm *CombatMode) addCombatLog(message string) {
	cm.combatLog = append(cm.combatLog, message)
	cm.messageCountSinceTrim++

	// Use AppendText for O(1) performance - only add the new message
	cm.combatLogArea.AppendText(message + "\n")

	// Every 100 messages, trim old entries to prevent unbounded growth
	if cm.messageCountSinceTrim >= 100 {
		cm.trimCombatLog()
	}
}

// trimCombatLog keeps only the last 300 messages and rebuilds the display
func (cm *CombatMode) trimCombatLog() {
	const maxMessages = 300

	if len(cm.combatLog) > maxMessages {
		// Remove oldest messages, keep most recent ones
		removed := len(cm.combatLog) - maxMessages
		cm.combatLog = cm.combatLog[removed:]

		// Rebuild the text area display with trimmed content
		fullText := ""
		for _, msg := range cm.combatLog {
			fullText += msg + "\n"
		}
		cm.combatLogArea.SetText(fullText)
	}

	cm.messageCountSinceTrim = 0
}

func (cm *CombatMode) handleEndTurn() {
	// End current faction's turn
	if err := cm.turnManager.EndTurn(); err != nil {
		cm.addCombatLog(fmt.Sprintf("Error ending turn: %v", err))
		return
	}

	// Get new faction info
	currentFactionID := cm.turnManager.GetCurrentFaction()
	round := cm.turnManager.GetCurrentRound()

	// Get faction name
	factionName := cm.queries.GetFactionName(currentFactionID)

	cm.addCombatLog(fmt.Sprintf("=== Round %d: %s's Turn ===", round, factionName))

	// Clear selection when turn changes
	cm.selectedSquadID = 0
	cm.selectedTargetID = 0
	cm.inAttackMode = false
	cm.inMoveMode = false
	cm.validMoveTiles = nil

	// Update UI displays using components
	cm.turnOrderComponent.Refresh()
	cm.factionInfoComponent.ShowFaction(currentFactionID)
	cm.squadListComponent.Refresh()
	cm.squadDetailComponent.SetText("Select a squad\nto view details")
}

// getFactionName and isPlayerFaction methods removed - now using cm.queries service

func (cm *CombatMode) toggleAttackMode() {
	if cm.selectedSquadID == 0 {
		cm.addCombatLog("Select a squad first!")
		return
	}

	cm.inAttackMode = !cm.inAttackMode
	cm.inMoveMode = false // Disable move mode

	if cm.inAttackMode {
		cm.addCombatLog("Attack mode: Press 1-3 to target enemy")
		cm.showAvailableTargets()
	} else {
		cm.addCombatLog("Attack mode cancelled")
		cm.selectedTargetID = 0
	}
}

func (cm *CombatMode) showAvailableTargets() {
	currentFactionID := cm.turnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return
	}

	// Get all enemy squads using query service
	enemySquads := cm.queries.GetEnemySquads(currentFactionID)

	if len(enemySquads) == 0 {
		cm.addCombatLog("No enemy targets available!")
		return
	}

	// Show up to 3 targets
	for i := 0; i < len(enemySquads) && i < 3; i++ {
		targetName := cm.queries.GetSquadName(enemySquads[i])
		cm.addCombatLog(fmt.Sprintf("  [%d] %s", i+1, targetName))
	}
}

func (cm *CombatMode) toggleMoveMode() {
	if cm.selectedSquadID == 0 {
		cm.addCombatLog("Select a squad first!")
		return
	}

	cm.inMoveMode = !cm.inMoveMode
	cm.inAttackMode = false // Disable attack mode

	if cm.inMoveMode {
		// Get valid movement tiles
		cm.validMoveTiles = cm.movementSystem.GetValidMovementTiles(cm.selectedSquadID)

		if len(cm.validMoveTiles) == 0 {
			cm.addCombatLog("No movement remaining!")
			cm.inMoveMode = false
			return
		}

		cm.addCombatLog(fmt.Sprintf("Move mode: Click a tile (%d tiles available)", len(cm.validMoveTiles)))
		cm.addCombatLog("Click on the map to move, or press M to cancel")
	} else {
		cm.addCombatLog("Move mode cancelled")
		cm.validMoveTiles = nil
	}
}

func (cm *CombatMode) selectSquad(squadID ecs.EntityID) {
	cm.selectedSquadID = squadID
	cm.inAttackMode = false
	cm.inMoveMode = false
	cm.selectedTargetID = 0

	// Get squad name
	squadName := cm.queries.GetSquadName(squadID)
	cm.addCombatLog(fmt.Sprintf("Selected: %s", squadName))

	// Update detail panel using component
	cm.squadDetailComponent.ShowSquad(squadID)
}


func (cm *CombatMode) selectTarget(targetSquadID ecs.EntityID) {
	if !cm.inAttackMode {
		return
	}

	cm.selectedTargetID = targetSquadID

	// Execute attack immediately
	cm.executeAttack()
}

func (cm *CombatMode) executeAttack() {
	if cm.selectedSquadID == 0 || cm.selectedTargetID == 0 {
		return
	}

	// Create combat action system
	combatSys := combat.NewCombatActionSystem(cm.context.ECSManager)

	// Check if attack is valid with detailed reason
	reason, canAttack := combatSys.CanSquadAttackWithReason(cm.selectedSquadID, cm.selectedTargetID)
	if !canAttack {
		cm.addCombatLog(fmt.Sprintf("Cannot attack: %s", reason))
		cm.inAttackMode = false
		cm.selectedTargetID = 0
		return
	}

	// Execute attack
	attackerName := cm.queries.GetSquadName(cm.selectedSquadID)
	targetName := cm.queries.GetSquadName(cm.selectedTargetID)

	err := combatSys.ExecuteAttackAction(cm.selectedSquadID, cm.selectedTargetID)
	if err != nil {
		cm.addCombatLog(fmt.Sprintf("Attack failed: %v", err))
	} else {
		cm.addCombatLog(fmt.Sprintf("%s attacked %s!", attackerName, targetName))

		// Check if target destroyed
		if squads.IsSquadDestroyed(cm.selectedTargetID, cm.context.ECSManager) {
			cm.addCombatLog(fmt.Sprintf("%s was destroyed!", targetName))
		}
	}

	// Reset attack mode
	cm.inAttackMode = false
	cm.selectedTargetID = 0
}

func (cm *CombatMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Combat Mode")
	cm.addCombatLog("=== COMBAT STARTED ===")

	// Collect all factions using query service
	factionIDs := cm.queries.GetAllFactions()

	// Initialize combat with all factions
	if len(factionIDs) > 0 {
		if err := cm.turnManager.InitializeCombat(factionIDs); err != nil {
			cm.addCombatLog(fmt.Sprintf("Error initializing combat: %v", err))
			return err
		}

		// Log initial faction
		currentFactionID := cm.turnManager.GetCurrentFaction()
		factionName := cm.queries.GetFactionName(currentFactionID)
		cm.addCombatLog(fmt.Sprintf("Round 1: %s goes first!", factionName))

		// Update displays using components
		cm.turnOrderComponent.Refresh()
		cm.factionInfoComponent.ShowFaction(currentFactionID)
		cm.squadListComponent.Refresh()
	} else {
		cm.addCombatLog("No factions found - combat cannot start")
	}

	return nil
}

func (cm *CombatMode) Exit(toMode UIMode) error {
	fmt.Println("Exiting Combat Mode")
	// Clear combat log for next battle
	cm.combatLog = cm.combatLog[:0]
	cm.messageCountSinceTrim = 0
	return nil
}

func (cm *CombatMode) Update(deltaTime float64) error {
	// Update UI displays each frame using components
	cm.turnOrderComponent.Refresh()

	currentFactionID := cm.turnManager.GetCurrentFaction()
	if currentFactionID != 0 {
		cm.factionInfoComponent.ShowFaction(currentFactionID)
	}

	if cm.selectedSquadID != 0 {
		cm.squadDetailComponent.ShowSquad(cm.selectedSquadID)
	}

	return nil
}

func (cm *CombatMode) Render(screen *ebiten.Image) {
	playerPos := *cm.context.PlayerData.Pos
	currentFactionID := cm.turnManager.GetCurrentFaction()

	// Render squad highlights (always shown)
	cm.highlightRenderer.Render(screen, playerPos, currentFactionID, cm.selectedSquadID)

	// Render valid movement tiles (only in move mode)
	if cm.inMoveMode && len(cm.validMoveTiles) > 0 {
		cm.movementRenderer.Render(screen, playerPos, cm.validMoveTiles)
	}
}

// renderMovementTiles and renderAllSquadHighlights removed - now using MovementTileRenderer and SquadHighlightRenderer

func (cm *CombatMode) HandleInput(inputState *InputState) bool {
	// Handle common input (ESC key to flee combat)
	if cm.HandleCommonInput(inputState) {
		return true
	}

	// Handle left mouse clicks
	if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
		if cm.inMoveMode {
			// In move mode: click to move squad
			cm.handleMovementClick(inputState.MouseX, inputState.MouseY)
		} else {
			// Not in move mode: click to select/attack squad
			cm.handleSquadClick(inputState.MouseX, inputState.MouseY)
		}
		return true
	}

	// Space to end turn
	if inputState.KeysJustPressed[ebiten.KeySpace] {
		cm.handleEndTurn()
		return true
	}

	// A key to toggle attack mode
	if inputState.KeysJustPressed[ebiten.KeyA] {
		cm.toggleAttackMode()
		return true
	}

	// M key to toggle move mode
	if inputState.KeysJustPressed[ebiten.KeyM] {
		cm.toggleMoveMode()
		return true
	}

	// TAB to cycle through squads
	if inputState.KeysJustPressed[ebiten.KeyTab] {
		cm.cycleSquadSelection()
		return true
	}

	// Number keys 1-3 to select enemy targets in attack mode
	if cm.inAttackMode {
		if inputState.KeysJustPressed[ebiten.Key1] {
			cm.selectEnemyTarget(0)
			return true
		}
		if inputState.KeysJustPressed[ebiten.Key2] {
			cm.selectEnemyTarget(1)
			return true
		}
		if inputState.KeysJustPressed[ebiten.Key3] {
			cm.selectEnemyTarget(2)
			return true
		}
	}

	return false
}

func (cm *CombatMode) selectEnemyTarget(index int) {
	currentFactionID := cm.turnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return
	}

	// Get all enemy squads using unified query service
	enemySquads := cm.queries.GetEnemySquads(currentFactionID)

	if index < 0 || index >= len(enemySquads) {
		cm.addCombatLog(fmt.Sprintf("No enemy squad at index %d", index+1))
		return
	}

	cm.selectTarget(enemySquads[index])
}

func (cm *CombatMode) handleMovementClick(mouseX, mouseY int) {
	// Convert mouse coordinates to tile coordinates using viewport system
	// This accounts for camera centering and scale factor

	// Get player position for viewport centering
	playerPos := *cm.context.PlayerData.Pos

	// Create viewport centered on player using the initialized ScreenInfo
	// graphics.ScreenInfo is updated every frame with current screen dimensions
	manager := coords.NewCoordinateManager(graphics.ScreenInfo)
	viewport := coords.NewViewport(manager, playerPos)

	// Convert screen coordinates to logical coordinates
	clickedPos := viewport.ScreenToLogical(mouseX, mouseY)

	// Check if clicked position is in valid movement tiles
	isValidTile := false
	for _, validPos := range cm.validMoveTiles {
		if validPos.X == clickedPos.X && validPos.Y == clickedPos.Y {
			isValidTile = true
			break
		}
	}

	if !isValidTile {
		cm.addCombatLog("Invalid movement destination!")
		return
	}

	// Execute movement
	err := cm.movementSystem.MoveSquad(cm.selectedSquadID, clickedPos)
	if err != nil {
		cm.addCombatLog(fmt.Sprintf("Movement failed: %v", err))
		return
	}

	// Update unit positions to match squad position
	cm.updateUnitPositions(cm.selectedSquadID, clickedPos)

	squadName := cm.queries.GetSquadName(cm.selectedSquadID)
	cm.addCombatLog(fmt.Sprintf("%s moved to (%d, %d)", squadName, clickedPos.X, clickedPos.Y))

	// Exit move mode
	cm.inMoveMode = false
	cm.validMoveTiles = nil

	// Update squad detail to show new movement remaining
	if cm.selectedSquadID != 0 {
		cm.squadDetailComponent.ShowSquad(cm.selectedSquadID)
	}
}

func (cm *CombatMode) handleSquadClick(mouseX, mouseY int) {
	// Convert mouse coordinates to tile coordinates
	playerPos := *cm.context.PlayerData.Pos
	manager := coords.NewCoordinateManager(graphics.ScreenInfo)
	viewport := coords.NewViewport(manager, playerPos)
	clickedPos := viewport.ScreenToLogical(mouseX, mouseY)

	// Find if a squad is at the clicked position using unified query service
	clickedSquadID := cm.queries.GetSquadAtPosition(clickedPos)

	// If no squad was clicked, do nothing
	if clickedSquadID == 0 {
		return
	}

	// Get faction info for the clicked squad
	squadInfo := cm.queries.GetSquadInfo(clickedSquadID)
	if squadInfo == nil {
		return
	}
	clickedFactionID := squadInfo.FactionID

	currentFactionID := cm.turnManager.GetCurrentFaction()

	// If it's the player's turn
	if cm.queries.IsPlayerFaction(currentFactionID) {
		// If clicking an allied squad: select it
		if clickedFactionID == currentFactionID {
			cm.selectSquad(clickedSquadID)
			return
		}

		// If clicking an enemy squad and we have a selected squad: attack immediately
		if cm.selectedSquadID != 0 && clickedFactionID != currentFactionID {
			cm.selectedTargetID = clickedSquadID
			cm.executeAttack()
		}
	}
}

func (cm *CombatMode) updateUnitPositions(squadID ecs.EntityID, newSquadPos coords.LogicalPosition) {
	// Get all units in the squad
	unitIDs := squads.GetUnitIDsInSquad(squadID, cm.context.ECSManager)

	// Update each unit's position to match the squad's new position
	for _, unitID := range unitIDs {
		for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["squadmember"]) {
			if result.Entity.GetID() == unitID {
				// Update PositionComponent
				if result.Entity.HasComponent(common.PositionComponent) {
					posPtr := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
					if posPtr != nil {
						posPtr.X = newSquadPos.X
						posPtr.Y = newSquadPos.Y
					}
				}
				break
			}
		}
	}
}

func (cm *CombatMode) cycleSquadSelection() {
	currentFactionID := cm.turnManager.GetCurrentFaction()
	if currentFactionID == 0 || !cm.queries.IsPlayerFaction(currentFactionID) {
		return
	}

	squadIDs := cm.factionManager.GetFactionSquads(currentFactionID)

	// Filter out destroyed squads
	aliveSquads := []ecs.EntityID{}
	for _, squadID := range squadIDs {
		if !squads.IsSquadDestroyed(squadID, cm.context.ECSManager) {
			aliveSquads = append(aliveSquads, squadID)
		}
	}

	if len(aliveSquads) == 0 {
		return
	}

	// Find current index
	currentIndex := -1
	for i, squadID := range aliveSquads {
		if squadID == cm.selectedSquadID {
			currentIndex = i
			break
		}
	}

	// Select next squad
	nextIndex := (currentIndex + 1) % len(aliveSquads)
	cm.selectSquad(aliveSquads[nextIndex])
}

