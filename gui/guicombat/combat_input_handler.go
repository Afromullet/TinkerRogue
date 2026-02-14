package guicombat

import (
	"fmt"
	"game_main/common"
	"game_main/gui/framework"
	"game_main/mind/behavior"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/visual/graphics"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatInputHandler manages all input processing for combat mode.
// Uses CombatModeDeps for shared state and services.
type CombatInputHandler struct {
	actionHandler    *CombatActionHandler
	deps             *CombatModeDeps
	playerPos        *coords.LogicalPosition
	currentFactionID ecs.EntityID

	// Debug kill mode
	inDebugKillMode bool

	// Visualization input support
	visualization *CombatVisualizationManager
	panels        *framework.PanelRegistry
	logManager    *CombatLogManager
}

// NewCombatInputHandler creates a new combat input handler
func NewCombatInputHandler(actionHandler *CombatActionHandler, deps *CombatModeDeps) *CombatInputHandler {
	return &CombatInputHandler{
		actionHandler: actionHandler,
		deps:          deps,
	}
}

// SetPlayerPosition sets the player position for viewport calculations
func (cih *CombatInputHandler) SetPlayerPosition(playerPos *coords.LogicalPosition) {
	cih.playerPos = playerPos
}

// SetCurrentFactionID sets the current faction ID for turn checking
func (cih *CombatInputHandler) SetCurrentFactionID(factionID ecs.EntityID) {
	cih.currentFactionID = factionID
}

// SetVisualization sets the visualization manager and related dependencies for keybinding handling
func (cih *CombatInputHandler) SetVisualization(viz *CombatVisualizationManager, panels *framework.PanelRegistry, logManager *CombatLogManager) {
	cih.visualization = viz
	cih.panels = panels
	cih.logManager = logManager
}

// EnterDebugKillMode activates click-to-kill mode for debug purposes.
func (cih *CombatInputHandler) EnterDebugKillMode() {
	cih.inDebugKillMode = true
	cih.actionHandler.addLog("[DEBUG] Kill mode active - click a squad to remove it")
}

// handleDebugKillClick processes a click while in debug kill mode.
func (cih *CombatInputHandler) handleDebugKillClick(mouseX, mouseY int) {
	if cih.playerPos == nil {
		return
	}

	clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)
	clickedSquadID := combat.GetSquadAtPosition(clickedPos, cih.deps.Queries.ECSManager)

	if clickedSquadID == 0 {
		cih.actionHandler.addLog("[DEBUG] No squad at that position")
		return
	}

	cih.actionHandler.DebugKillSquad(clickedSquadID)
}

// HandleInput processes input and returns true if input was consumed
func (cih *CombatInputHandler) HandleInput(inputState *framework.InputState) bool {
	// ESC cancels debug kill mode
	if cih.inDebugKillMode && inputState.KeysJustPressed[ebiten.KeyEscape] {
		cih.inDebugKillMode = false
		cih.actionHandler.addLog("[DEBUG] Kill mode cancelled")
		return true
	}

	// Handle mouse clicks
	if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
		// Debug kill mode takes priority over all other click handling
		if cih.inDebugKillMode {
			defer func() { cih.inDebugKillMode = false }()
			cih.handleDebugKillClick(inputState.MouseX, inputState.MouseY)
			return true
		}

		if cih.deps.BattleState.InMoveMode {
			// In move mode: click to move squad
			cih.handleMovementClick(inputState.MouseX, inputState.MouseY)
		} else {
			// Not in move mode: click to select/attack squad
			cih.handleSquadClick(inputState.MouseX, inputState.MouseY)
		}
		return true
	}

	// Space to end turn
	if inputState.KeysJustPressed[ebiten.KeySpace] {
		// This is handled at a higher level
		return false
	}

	// A key to toggle attack mode
	if inputState.KeysJustPressed[ebiten.KeyA] {
		cih.actionHandler.ToggleAttackMode()
		return true
	}

	// M key to toggle move mode
	if inputState.KeysJustPressed[ebiten.KeyM] {
		cih.actionHandler.ToggleMoveMode()
		return true
	}

	// TAB to cycle through squads
	if inputState.KeysJustPressed[ebiten.KeyTab] {
		cih.actionHandler.CycleSquadSelection()
		return true
	}

	// Ctrl+Z to undo last move
	if inputState.KeysJustPressed[ebiten.KeyZ] && (inputState.KeysPressed[ebiten.KeyControl] || inputState.KeysPressed[ebiten.KeyMeta]) {
		cih.actionHandler.UndoLastMove()
		return true
	}

	// Ctrl+Y to redo last move
	if inputState.KeysJustPressed[ebiten.KeyY] && (inputState.KeysPressed[ebiten.KeyControl] || inputState.KeysPressed[ebiten.KeyMeta]) {
		cih.actionHandler.RedoLastMove()
		return true
	}

	// Number keys 1-3 to select enemy targets in attack mode
	if cih.deps.BattleState.InAttackMode {
		if inputState.KeysJustPressed[ebiten.Key1] {
			cih.actionHandler.SelectEnemyTarget(0)
			return true
		}
		if inputState.KeysJustPressed[ebiten.Key2] {
			cih.actionHandler.SelectEnemyTarget(1)
			return true
		}
		if inputState.KeysJustPressed[ebiten.Key3] {
			cih.actionHandler.SelectEnemyTarget(2)
			return true
		}
	}

	// Visualization keybindings
	if cih.handleThreatToggle(inputState) {
		return true
	}
	if cih.handleHealthBarToggle(inputState) {
		return true
	}
	if cih.handleLayerToggle(inputState) {
		return true
	}

	// Ctrl+K to kill all enemy squads (debug command)
	// Also accept just K without modifier for easier debugging
	if inputState.KeysJustPressed[ebiten.KeyK] {
		println("[DEBUG] K key pressed")
		if inputState.KeysPressed[ebiten.KeyControl] {
			println("[DEBUG] Control is held")
		}
		if inputState.KeysPressed[ebiten.KeyMeta] {
			println("[DEBUG] Meta is held")
		}

		if inputState.KeysPressed[ebiten.KeyControl] || inputState.KeysPressed[ebiten.KeyMeta] {
			println("[DEBUG] Calling killAllEnemySquads()")
			cih.killAllEnemySquads()
			return true
		}
	}

	return false
}

// handleMovementClick processes clicks when in move mode
func (cih *CombatInputHandler) handleMovementClick(mouseX, mouseY int) {
	// Convert mouse coordinates to tile coordinates
	if cih.playerPos == nil {
		return
	}

	// Convert screen coordinates to logical coordinates (handles both scrolling modes)
	clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)

	// Compute valid tiles on-demand from movement system
	validTiles := cih.deps.CombatService.MovementSystem.GetValidMovementTiles(cih.deps.BattleState.SelectedSquadID)
	if validTiles == nil {
		validTiles = []coords.LogicalPosition{}
	}

	// Check if clicked position is in valid movement tiles
	isValidTile := false
	for _, validPos := range validTiles {
		if validPos.X == clickedPos.X && validPos.Y == clickedPos.Y {
			isValidTile = true
			break
		}
	}
	if !isValidTile {
		return // Invalid tile, do nothing
	}

	selectedSquad := cih.deps.BattleState.SelectedSquadID
	if selectedSquad == 0 {
		return
	}

	// Execute movement through action handler
	cih.actionHandler.MoveSquad(selectedSquad, clickedPos)
}

// handleSquadClick processes clicks when not in move mode
func (cih *CombatInputHandler) handleSquadClick(mouseX, mouseY int) {
	// Convert mouse coordinates to tile coordinates
	if cih.playerPos == nil {
		return
	}

	// Convert screen coordinates to logical coordinates (handles both scrolling modes)
	clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)

	// Find if a squad is at the clicked position
	clickedSquadID := combat.GetSquadAtPosition(clickedPos, cih.deps.Queries.ECSManager)

	// If no squad was clicked, do nothing
	if clickedSquadID == 0 {
		return
	}

	// Get faction info for the clicked squad
	squadInfo := cih.deps.Queries.GetSquadInfo(clickedSquadID)
	if squadInfo == nil {
		return
	}
	clickedFactionID := squadInfo.FactionID

	// If no faction currently active, do nothing
	if cih.currentFactionID == 0 {
		return
	}

	// If it's the player's turn
	factionData := cih.deps.Queries.CombatCache.FindFactionDataByID(cih.currentFactionID, cih.deps.Queries.ECSManager)
	if factionData != nil && factionData.IsPlayerControlled {
		// If clicking an allied squad: select it
		if clickedFactionID == cih.currentFactionID {
			cih.actionHandler.SelectSquad(clickedSquadID)
			return
		}

		// If clicking an enemy squad and we have a selected squad: attack immediately
		selectedSquad := cih.deps.BattleState.SelectedSquadID
		if selectedSquad != 0 && clickedFactionID != cih.currentFactionID {
			cih.deps.BattleState.SelectedTargetID = clickedSquadID
			cih.actionHandler.ExecuteAttack()
		}
	}
}

// killAllEnemySquads is a debug function that instantly kills all units in all enemy squads.
// This is useful for quickly testing victory conditions.
// Only affects squads in the current encounter (not all squads in the ECS).
func (cih *CombatInputHandler) killAllEnemySquads() {
	// Get current encounter ID to filter only this encounter's squads
	encounterID := cih.deps.EncounterService.GetCurrentEncounterID()
	if encounterID == 0 {
		println("[DEBUG] No current encounter, cannot kill enemies")
		return
	}

	// CRITICAL: Must use PLAYER faction ID, not current turn faction ID
	// If we use currentFactionID and it's the AI's turn, we'd kill player squads instead!
	playerFactionID := cih.getPlayerFactionID(encounterID)
	if playerFactionID == 0 {
		println("[DEBUG] No player faction found, cannot kill enemies")
		return
	}

	enemySquads := cih.deps.Queries.GetEnemySquadsForEncounter(playerFactionID, encounterID)
	println("[DEBUG] Ctrl+K pressed - Found", len(enemySquads), "enemy squads in encounter", encounterID, "to kill")

	if len(enemySquads) == 0 {
		return
	}

	// Kill all units in each enemy squad
	totalKilled := 0
	for _, squadID := range enemySquads {
		killed := cih.killAllUnitsInSquad(squadID)
		totalKilled += killed
		println("[DEBUG] Killed", killed, "units in squad", squadID)
	}

	println("[DEBUG] Total units killed:", totalKilled)

	// Note: Victory will be detected at next turn boundary (handleEndTurn or advanceAfterAITurn)
	// Debug cheat doesn't need immediate victory check since player must still end turn
}

// killAllUnitsInSquad sets all units in a squad to 0 health and returns count killed
func (cih *CombatInputHandler) killAllUnitsInSquad(squadID ecs.EntityID) int {
	// Get all units in the squad
	unitIDs := squads.GetUnitIDsInSquad(squadID, cih.deps.Queries.ECSManager)
	killed := 0

	// Set health to 0 for all units
	for _, unitID := range unitIDs {
		unitEntity := cih.deps.Queries.ECSManager.FindEntityByID(unitID)
		if unitEntity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
		if attr != nil {
			oldHealth := attr.CurrentHealth
			attr.CurrentHealth = 0
			killed++
			println("[DEBUG]   Unit", unitID, "health:", oldHealth, "-> 0")
		}
	}

	return killed
}

// getPlayerFactionID finds the player-controlled faction in the current encounter
func (cih *CombatInputHandler) getPlayerFactionID(encounterID ecs.EntityID) ecs.EntityID {
	encounterFactions := cih.deps.Queries.GetFactionsForEncounter(encounterID)
	for _, factionID := range encounterFactions {
		factionData := cih.deps.Queries.CombatCache.FindFactionDataByID(factionID, cih.deps.Queries.ECSManager)
		if factionData != nil && factionData.IsPlayerControlled {
			return factionID
		}
	}
	return 0
}

// handleThreatToggle handles H key to toggle threat heat map
func (cih *CombatInputHandler) handleThreatToggle(inputState *framework.InputState) bool {
	if !inputState.KeysJustPressed[ebiten.KeyH] {
		return false
	}

	threatViz := cih.visualization.GetThreatVisualizer()
	if threatViz == nil {
		return true
	}

	shiftPressed := inputState.KeysPressed[ebiten.KeyShift] ||
		inputState.KeysPressed[ebiten.KeyShiftLeft] ||
		inputState.KeysPressed[ebiten.KeyShiftRight]

	combatLogArea := GetCombatLogTextArea(cih.panels)

	if shiftPressed {
		threatViz.SwitchThreatView()
		viewName := "Enemy Threats"
		if threatViz.GetThreatViewMode() == behavior.ViewPlayerThreats {
			viewName = "Player Threats"
		}
		cih.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("Switched to %s view", viewName))
	} else {
		// If not active or in different mode: activate in Threat mode
		// If already active in Threat mode: turn off
		if !threatViz.IsActive() || threatViz.GetMode() != behavior.VisualizerModeThreat {
			threatViz.SetMode(behavior.VisualizerModeThreat)
			if !threatViz.IsActive() {
				threatViz.Toggle()
			}
			cih.logManager.UpdateTextArea(combatLogArea, "Threat visualization enabled")
		} else {
			threatViz.Toggle()
			cih.logManager.UpdateTextArea(combatLogArea, "Threat visualization disabled")
		}
	}
	cih.updateLayerStatusWidget()
	return true
}

// handleHealthBarToggle handles Ctrl+Right key to toggle health bars
func (cih *CombatInputHandler) handleHealthBarToggle(inputState *framework.InputState) bool {
	if !inputState.KeysJustPressed[ebiten.KeyControlRight] {
		return false
	}

	battleState := cih.deps.BattleState
	battleState.ShowHealthBars = !battleState.ShowHealthBars
	status := "enabled"
	if !battleState.ShowHealthBars {
		status = "disabled"
	}
	combatLogArea := GetCombatLogTextArea(cih.panels)
	cih.logManager.UpdateTextArea(combatLogArea, fmt.Sprintf("Health bars %s", status))
	return true
}

// handleLayerToggle handles L key to toggle layer visualizer
func (cih *CombatInputHandler) handleLayerToggle(inputState *framework.InputState) bool {
	if !inputState.KeysJustPressed[ebiten.KeyL] {
		return false
	}

	threatViz := cih.visualization.GetThreatVisualizer()
	if threatViz == nil {
		return true
	}

	shiftPressed := inputState.KeysPressed[ebiten.KeyShift] ||
		inputState.KeysPressed[ebiten.KeyShiftLeft] ||
		inputState.KeysPressed[ebiten.KeyShiftRight]

	combatLogArea := GetCombatLogTextArea(cih.panels)

	if shiftPressed {
		threatViz.CycleLayerMode()
		modeInfo := threatViz.GetLayerModeInfo()
		cih.logManager.UpdateTextArea(combatLogArea,
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
			cih.logManager.UpdateTextArea(combatLogArea,
				fmt.Sprintf("Layer visualization enabled: %s", modeInfo.Name))
		} else {
			threatViz.Toggle()
			cih.logManager.UpdateTextArea(combatLogArea, "Layer visualization disabled")
		}
	}
	cih.updateLayerStatusWidget()
	return true
}

// updateLayerStatusWidget updates the layer status panel visibility and text
func (cih *CombatInputHandler) updateLayerStatusWidget() {
	threatViz := cih.visualization.GetThreatVisualizer()

	result := cih.panels.Get(CombatPanelLayerStatus)
	if result == nil || threatViz == nil {
		return
	}
	layerStatusPanel := result.Container
	layerStatusText := result.TextLabel
	if layerStatusPanel == nil || layerStatusText == nil {
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
