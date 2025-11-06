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
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatMode provides focused UI for turn-based squad combat
type CombatMode struct {
	ui          *ebitenui.UI
	context     *UIContext
	layout      *LayoutConfig
	modeManager *UIModeManager

	rootContainer    *widget.Container
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

	// Panel builders for UI composition
	panelBuilders *PanelBuilders
}

func NewCombatMode(modeManager *UIModeManager) *CombatMode {
	return &CombatMode{
		modeManager: modeManager,
		combatLog:   make([]string, 0, 100),
	}
}

func (cm *CombatMode) Initialize(ctx *UIContext) error {
	cm.context = ctx
	cm.layout = NewLayoutConfig(ctx)
	cm.panelBuilders = NewPanelBuilders(cm.layout, cm.modeManager)

	// Initialize combat managers
	cm.turnManager = combat.NewTurnManager(ctx.ECSManager)
	cm.factionManager = combat.NewFactionManager(ctx.ECSManager)
	cm.movementSystem = combat.NewMovementSystem(ctx.ECSManager, common.GlobalPositionSystem)

	cm.ui = &ebitenui.UI{}
	cm.rootContainer = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	cm.ui.Container = cm.rootContainer

	// Build combat UI layout
	cm.buildTurnOrderPanel()
	cm.buildFactionInfoPanel()
	cm.buildSquadListPanel()
	cm.buildSquadDetailPanel()
	cm.buildCombatLog()
	cm.buildActionButtons()

	return nil
}

func (cm *CombatMode) buildTurnOrderPanel() {
	// Use panel builder for top-center panel
	cm.turnOrderPanel = cm.panelBuilders.BuildTopCenterPanel(0.4, 0.08, 0.01)

	// Dynamic turn order display (updated in Update())
	cm.turnOrderLabel = widget.NewText(
		widget.TextOpts.Text("Initializing combat...", LargeFace, color.White),
	)
	cm.turnOrderPanel.AddChild(cm.turnOrderLabel)

	cm.rootContainer.AddChild(cm.turnOrderPanel)
}

func (cm *CombatMode) buildFactionInfoPanel() {
	// Use panel builder for top-left panel
	cm.factionInfoPanel = cm.panelBuilders.BuildTopLeftPanel(0.15, 0.12, 0.01, 0.01)

	// Dynamic faction info (updated in Update())
	cm.factionInfoText = widget.NewText(
		widget.TextOpts.Text("Faction Info", SmallFace, color.White),
	)
	cm.factionInfoPanel.AddChild(cm.factionInfoText)

	cm.rootContainer.AddChild(cm.factionInfoPanel)
}

func (cm *CombatMode) buildSquadListPanel() {
	// Use panel builder for left-side panel
	cm.squadListPanel = cm.panelBuilders.BuildLeftSidePanel(0.15, 0.5, 0.01, widget.AnchorLayoutPositionCenter)

	// Squad list will be populated dynamically in updateSquadList()
	listLabel := widget.NewText(
		widget.TextOpts.Text("Your Squads:", SmallFace, color.White),
	)
	cm.squadListPanel.AddChild(listLabel)

	cm.rootContainer.AddChild(cm.squadListPanel)
}

func (cm *CombatMode) buildSquadDetailPanel() {
	// Use panel builder for left-bottom panel
	cm.squadDetailPanel = cm.panelBuilders.BuildLeftBottomPanel(0.15, 0.25, 0.01, 0.15)

	// Squad details will be updated dynamically
	cm.squadDetailText = widget.NewText(
		widget.TextOpts.Text("Select a squad\nto view details", SmallFace, color.White),
	)
	cm.squadDetailPanel.AddChild(cm.squadDetailText)

	cm.rootContainer.AddChild(cm.squadDetailPanel)
}

func (cm *CombatMode) buildCombatLog() {
	// Use panel builder for right-side panel
	var logContainer *widget.Container
	logContainer, cm.combatLogArea = cm.panelBuilders.BuildRightSidePanel("Combat started!\n")
	cm.rootContainer.AddChild(logContainer)
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

	// Use panel builder for action button container
	buttons := []*widget.Button{cm.attackButton, cm.moveButton, endTurnBtn, fleeBtn}
	cm.actionButtons = cm.panelBuilders.BuildActionButtons(buttons)

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
	factionName := cm.getFactionName(currentFactionID)

	cm.addCombatLog(fmt.Sprintf("=== Round %d: %s's Turn ===", round, factionName))

	// Clear selection when turn changes
	cm.selectedSquadID = 0
	cm.selectedTargetID = 0
	cm.inAttackMode = false
	cm.inMoveMode = false
	cm.validMoveTiles = nil

	// Update UI displays
	cm.updateTurnDisplay()
	cm.updateFactionDisplay()
	cm.updateSquadList()
	cm.updateSquadDetail()
}

func (cm *CombatMode) getFactionName(factionID ecs.EntityID) string {
	// Query faction entity to get name
	for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["faction"]) {
		factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
		if factionData.FactionID == factionID {
			return factionData.Name
		}
	}
	return "Unknown Faction"
}

func (cm *CombatMode) isPlayerFaction(factionID ecs.EntityID) bool {
	// Query faction entity to check if player controlled
	for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["faction"]) {
		factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
		if factionData.FactionID == factionID {
			return factionData.IsPlayerControlled
		}
	}
	return false
}

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

	// Get all enemy squads
	enemySquads := []ecs.EntityID{}
	for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["mapposition"]) {
		mapPos := common.GetComponentType[*combat.MapPositionData](result.Entity, combat.MapPositionComponent)
		if mapPos.FactionID != currentFactionID {
			if !squads.IsSquadDestroyed(mapPos.SquadID, cm.context.ECSManager) {
				enemySquads = append(enemySquads, mapPos.SquadID)
			}
		}
	}

	if len(enemySquads) == 0 {
		cm.addCombatLog("No enemy targets available!")
		return
	}

	// Show up to 3 targets
	for i := 0; i < len(enemySquads) && i < 3; i++ {
		targetName := cm.getSquadName(enemySquads[i])
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
	squadName := cm.getSquadName(squadID)
	cm.addCombatLog(fmt.Sprintf("Selected: %s", squadName))

	cm.updateSquadDetail()
}

func (cm *CombatMode) getSquadName(squadID ecs.EntityID) string {
	for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["squad"]) {
		squadData := common.GetComponentType[*squads.SquadData](result.Entity, squads.SquadComponent)
		if squadData.SquadID == squadID {
			return squadData.Name
		}
	}
	return "Unknown Squad"
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
	attackerName := cm.getSquadName(cm.selectedSquadID)
	targetName := cm.getSquadName(cm.selectedTargetID)

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

func (cm *CombatMode) updateSquadList() {
	// Clear existing buttons (keep label)
	children := cm.squadListPanel.Children()
	for len(children) > 1 {
		cm.squadListPanel.RemoveChild(children[len(children)-1])
		children = cm.squadListPanel.Children()
	}

	currentFactionID := cm.turnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		return
	}

	// Only show squads if it's player's turn
	if !cm.isPlayerFaction(currentFactionID) {
		noSquadsText := widget.NewText(
			widget.TextOpts.Text("AI Turn", SmallFace, color.Gray{Y: 128}),
		)
		cm.squadListPanel.AddChild(noSquadsText)
		return
	}

	squadIDs := cm.factionManager.GetFactionSquads(currentFactionID)

	for _, squadID := range squadIDs {
		// Skip destroyed squads
		if squads.IsSquadDestroyed(squadID, cm.context.ECSManager) {
			continue
		}

		squadName := cm.getSquadName(squadID)

		// Create button for each squad
		localSquadID := squadID // Capture for closure
		squadButton := CreateButtonWithConfig(ButtonConfig{
			Text: squadName,
			OnClick: func() {
				cm.selectSquad(localSquadID)
			},
		})

		cm.squadListPanel.AddChild(squadButton)
	}
}

func (cm *CombatMode) updateSquadDetail() {
	if cm.selectedSquadID == 0 {
		cm.squadDetailText.Label = "Select a squad\nto view details"
		return
	}

	squadName := cm.getSquadName(cm.selectedSquadID)

	// Get unit count and HP
	unitIDs := squads.GetUnitIDsInSquad(cm.selectedSquadID, cm.context.ECSManager)
	aliveUnits := 0
	totalHP := 0
	maxHP := 0

	for _, unitID := range unitIDs {
		for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["squadmember"]) {
			if result.Entity.GetID() == unitID {
				attrs := common.GetComponentType[*common.Attributes](result.Entity, common.AttributeComponent)
				if attrs.CanAct {
					aliveUnits++
				}
				totalHP += attrs.CurrentHealth
				maxHP += attrs.MaxHealth
			}
		}
	}

	// Get action state
	actionEntity := cm.findActionStateEntity(cm.selectedSquadID)
	hasActed := false
	hasMoved := false
	movementRemaining := 0

	if actionEntity != nil {
		actionState := common.GetComponentType[*combat.ActionStateData](actionEntity, combat.ActionStateComponent)
		hasActed = actionState.HasActed
		hasMoved = actionState.HasMoved
		movementRemaining = actionState.MovementRemaining
	}

	detailText := fmt.Sprintf("%s\n", squadName)
	detailText += fmt.Sprintf("Units: %d/%d\n", aliveUnits, len(unitIDs))
	detailText += fmt.Sprintf("HP: %d/%d\n", totalHP, maxHP)
	detailText += fmt.Sprintf("Move: %d\n", movementRemaining)

	if hasActed {
		detailText += "Status: Acted\n"
	} else if hasMoved {
		detailText += "Status: Moved\n"
	} else {
		detailText += "Status: Ready\n"
	}

	cm.squadDetailText.Label = detailText
}

func (cm *CombatMode) findActionStateEntity(squadID ecs.EntityID) *ecs.Entity {
	for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["actionstate"]) {
		actionState := common.GetComponentType[*combat.ActionStateData](result.Entity, combat.ActionStateComponent)
		if actionState.SquadID == squadID {
			return result.Entity
		}
	}
	return nil
}

func (cm *CombatMode) Enter(fromMode UIMode) error {
	fmt.Println("Entering Combat Mode")
	cm.addCombatLog("=== COMBAT STARTED ===")

	// Collect all factions
	var factionIDs []ecs.EntityID
	for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["faction"]) {
		factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
		factionIDs = append(factionIDs, factionData.FactionID)
	}

	// Initialize combat with all factions
	if len(factionIDs) > 0 {
		if err := cm.turnManager.InitializeCombat(factionIDs); err != nil {
			cm.addCombatLog(fmt.Sprintf("Error initializing combat: %v", err))
			return err
		}

		// Log initial faction
		currentFactionID := cm.turnManager.GetCurrentFaction()
		factionName := cm.getFactionName(currentFactionID)
		cm.addCombatLog(fmt.Sprintf("Round 1: %s goes first!", factionName))

		// Update displays
		cm.updateTurnDisplay()
		cm.updateFactionDisplay()
		cm.updateSquadList()
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
	// Update UI displays each frame
	cm.updateTurnDisplay()
	cm.updateFactionDisplay()
	cm.updateSquadDetail() // Update detail panel if squad selected
	return nil
}

func (cm *CombatMode) updateTurnDisplay() {
	currentFactionID := cm.turnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		cm.turnOrderLabel.Label = "No active combat"
		return
	}

	round := cm.turnManager.GetCurrentRound()
	factionName := cm.getFactionName(currentFactionID)

	// Add indicator if player's turn
	playerIndicator := ""
	if cm.isPlayerFaction(currentFactionID) {
		playerIndicator = " >>> YOUR TURN <<<"
	}

	cm.turnOrderLabel.Label = fmt.Sprintf("Round %d | %s%s", round, factionName, playerIndicator)
}

func (cm *CombatMode) updateFactionDisplay() {
	currentFactionID := cm.turnManager.GetCurrentFaction()
	if currentFactionID == 0 {
		cm.factionInfoText.Label = "No faction info"
		return
	}

	factionName := cm.getFactionName(currentFactionID)
	currentMana, maxMana := cm.factionManager.GetFactionMana(currentFactionID)

	// Get squad info
	squadIDs := cm.factionManager.GetFactionSquads(currentFactionID)
	aliveSquads := 0
	for _, squadID := range squadIDs {
		if !squads.IsSquadDestroyed(squadID, cm.context.ECSManager) {
			aliveSquads++
		}
	}

	infoText := fmt.Sprintf("%s\n", factionName)
	infoText += fmt.Sprintf("Squads: %d/%d\n", aliveSquads, len(squadIDs))
	infoText += fmt.Sprintf("Mana: %d/%d", currentMana, maxMana)

	cm.factionInfoText.Label = infoText
}

func (cm *CombatMode) Render(screen *ebiten.Image) {
	// Render all squad highlights (show player vs enemy squads with different colors)
	cm.renderAllSquadHighlights(screen)

	// Render valid movement tiles if in move mode
	if cm.inMoveMode && len(cm.validMoveTiles) > 0 {
		cm.renderMovementTiles(screen)
	}
}

func (cm *CombatMode) renderMovementTiles(screen *ebiten.Image) {
	// Get player position for viewport centering
	playerPos := *cm.context.PlayerData.Pos

	// Create viewport centered on player using the initialized ScreenInfo
	// Update screen dimensions from current screen buffer
	screenData := graphics.ScreenInfo
	screenData.ScreenWidth = screen.Bounds().Dx()
	screenData.ScreenHeight = screen.Bounds().Dy()

	manager := coords.NewCoordinateManager(screenData)
	viewport := coords.NewViewport(manager, playerPos)

	tileSize := screenData.TileSize
	scaleFactor := screenData.ScaleFactor

	// Create a semi-transparent green overlay for valid tiles
	for _, pos := range cm.validMoveTiles {
		// Convert logical position to screen position using viewport
		screenX, screenY := viewport.LogicalToScreen(pos)

		// Draw a scaled green rectangle
		scaledTileSize := tileSize * scaleFactor
		rect := ebiten.NewImage(scaledTileSize, scaledTileSize)
		rect.Fill(color.RGBA{R: 0, G: 255, B: 0, A: 80}) // Semi-transparent green

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenX, screenY)
		screen.DrawImage(rect, op)
	}
}

func (cm *CombatMode) renderAllSquadHighlights(screen *ebiten.Image) {
	// Get player position and viewport
	playerPos := *cm.context.PlayerData.Pos

	screenData := graphics.ScreenInfo
	screenData.ScreenWidth = screen.Bounds().Dx()
	screenData.ScreenHeight = screen.Bounds().Dy()

	manager := coords.NewCoordinateManager(screenData)
	viewport := coords.NewViewport(manager, playerPos)

	tileSize := screenData.TileSize
	scaleFactor := screenData.ScaleFactor
	borderThickness := 3
	scaledTileSize := tileSize * scaleFactor

	// Get current faction ID to determine player squads
	currentFactionID := cm.turnManager.GetCurrentFaction()

	// Query all squads on map
	for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["mapposition"]) {
		mapPosData := common.GetComponentType[*combat.MapPositionData](result.Entity, combat.MapPositionComponent)

		// Skip destroyed squads
		if squads.IsSquadDestroyed(mapPosData.SquadID, cm.context.ECSManager) {
			continue
		}

		// Convert logical position to screen position
		screenX, screenY := viewport.LogicalToScreen(mapPosData.Position)

		// Determine highlight color based on faction and selection status
		var highlightColor color.RGBA
		var borderOpacity uint8 = 150

		if mapPosData.SquadID == cm.selectedSquadID {
			// Selected squad gets bright white border
			highlightColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		} else if mapPosData.FactionID == currentFactionID {
			// Player faction squads get blue border
			highlightColor = color.RGBA{R: 0, G: 150, B: 255, A: borderOpacity}
		} else {
			// Enemy faction squads get red border
			highlightColor = color.RGBA{R: 255, G: 0, B: 0, A: borderOpacity}
		}

		// Draw border rectangles
		// Top border
		topBorder := ebiten.NewImage(scaledTileSize, borderThickness)
		topBorder.Fill(highlightColor)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenX, screenY)
		screen.DrawImage(topBorder, op)

		// Bottom border
		bottomBorder := ebiten.NewImage(scaledTileSize, borderThickness)
		bottomBorder.Fill(highlightColor)
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenX, screenY+float64(scaledTileSize-borderThickness))
		screen.DrawImage(bottomBorder, op)

		// Left border
		leftBorder := ebiten.NewImage(borderThickness, scaledTileSize)
		leftBorder.Fill(highlightColor)
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenX, screenY)
		screen.DrawImage(leftBorder, op)

		// Right border
		rightBorder := ebiten.NewImage(borderThickness, scaledTileSize)
		rightBorder.Fill(highlightColor)
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenX+float64(scaledTileSize-borderThickness), screenY)
		screen.DrawImage(rightBorder, op)
	}
}

func (cm *CombatMode) HandleInput(inputState *InputState) bool {
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

	// ESC to flee combat
	if inputState.KeysJustPressed[ebiten.KeyEscape] {
		if exploreMode, exists := cm.modeManager.GetMode("exploration"); exists {
			cm.modeManager.RequestTransition(exploreMode, "ESC pressed - fled combat")
			return true
		}
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

	// Get all enemy squads (squads not in current faction)
	enemySquads := []ecs.EntityID{}
	for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["mapposition"]) {
		mapPos := common.GetComponentType[*combat.MapPositionData](result.Entity, combat.MapPositionComponent)
		if mapPos.FactionID != currentFactionID {
			// Check if squad is alive
			if !squads.IsSquadDestroyed(mapPos.SquadID, cm.context.ECSManager) {
				enemySquads = append(enemySquads, mapPos.SquadID)
			}
		}
	}

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

	squadName := cm.getSquadName(cm.selectedSquadID)
	cm.addCombatLog(fmt.Sprintf("%s moved to (%d, %d)", squadName, clickedPos.X, clickedPos.Y))

	// Exit move mode
	cm.inMoveMode = false
	cm.validMoveTiles = nil

	// Update squad detail to show new movement remaining
	cm.updateSquadDetail()
}

func (cm *CombatMode) handleSquadClick(mouseX, mouseY int) {
	// Convert mouse coordinates to tile coordinates
	playerPos := *cm.context.PlayerData.Pos
	manager := coords.NewCoordinateManager(graphics.ScreenInfo)
	viewport := coords.NewViewport(manager, playerPos)
	clickedPos := viewport.ScreenToLogical(mouseX, mouseY)

	// Find if a squad is at the clicked position
	var clickedSquadID ecs.EntityID
	var clickedFactionID ecs.EntityID

	for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["mapposition"]) {
		mapPos := common.GetComponentType[*combat.MapPositionData](result.Entity, combat.MapPositionComponent)

		// Check if squad is at clicked position
		if mapPos.Position.X == clickedPos.X && mapPos.Position.Y == clickedPos.Y {
			// Make sure squad is not destroyed
			if !squads.IsSquadDestroyed(mapPos.SquadID, cm.context.ECSManager) {
				clickedSquadID = mapPos.SquadID
				clickedFactionID = mapPos.FactionID
				break
			}
		}
	}

	// If no squad was clicked, do nothing
	if clickedSquadID == 0 {
		return
	}

	currentFactionID := cm.turnManager.GetCurrentFaction()

	// If it's the player's turn
	if cm.isPlayerFaction(currentFactionID) {
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
	if currentFactionID == 0 || !cm.isPlayerFaction(currentFactionID) {
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

func (cm *CombatMode) GetEbitenUI() *ebitenui.UI {
	return cm.ui
}

func (cm *CombatMode) GetModeName() string {
	return "combat"
}
