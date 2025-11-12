package gui

import (
	"game_main/coords"
	"game_main/graphics"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatInputHandler manages all input processing for combat mode
type CombatInputHandler struct {
	actionHandler      *CombatActionHandler
	stateManager       *CombatStateManager
	queries            *GUIQueries
	playerPos          *coords.LogicalPosition
	currentFactionID   ecs.EntityID
}

// NewCombatInputHandler creates a new combat input handler
func NewCombatInputHandler(
	actionHandler *CombatActionHandler,
	stateManager *CombatStateManager,
	queries *GUIQueries,
) *CombatInputHandler {
	return &CombatInputHandler{
		actionHandler: actionHandler,
		stateManager:  stateManager,
		queries:       queries,
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

// HandleInput processes input and returns true if input was consumed
func (cih *CombatInputHandler) HandleInput(inputState *InputState) bool {
	// Handle mouse clicks
	if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
		if cih.stateManager.IsMoveMode() {
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

	// Number keys 1-3 to select enemy targets in attack mode
	if cih.stateManager.IsAttackMode() {
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

	return false
}

// handleMovementClick processes clicks when in move mode
func (cih *CombatInputHandler) handleMovementClick(mouseX, mouseY int) {
	// Convert mouse coordinates to tile coordinates using viewport system
	if cih.playerPos == nil {
		return
	}

	// Create viewport centered on player
	manager := coords.NewCoordinateManager(graphics.ScreenInfo)
	viewport := coords.NewViewport(manager, *cih.playerPos)

	// Convert screen coordinates to logical coordinates
	clickedPos := viewport.ScreenToLogical(mouseX, mouseY)

	// Check if clicked position is in valid movement tiles
	if !cih.stateManager.IsValidMoveTile(clickedPos) {
		return // Invalid tile, do nothing
	}

	selectedSquad := cih.stateManager.GetSelectedSquad()
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

	manager := coords.NewCoordinateManager(graphics.ScreenInfo)
	viewport := coords.NewViewport(manager, *cih.playerPos)
	clickedPos := viewport.ScreenToLogical(mouseX, mouseY)

	// Find if a squad is at the clicked position
	clickedSquadID := cih.queries.GetSquadAtPosition(clickedPos)

	// If no squad was clicked, do nothing
	if clickedSquadID == 0 {
		return
	}

	// Get faction info for the clicked squad
	squadInfo := cih.queries.GetSquadInfo(clickedSquadID)
	if squadInfo == nil {
		return
	}
	clickedFactionID := squadInfo.FactionID

	// If no faction currently active, do nothing
	if cih.currentFactionID == 0 {
		return
	}

	// If it's the player's turn
	if cih.queries.IsPlayerFaction(cih.currentFactionID) {
		// If clicking an allied squad: select it
		if clickedFactionID == cih.currentFactionID {
			cih.actionHandler.SelectSquad(clickedSquadID)
			return
		}

		// If clicking an enemy squad and we have a selected squad: attack immediately
		selectedSquad := cih.stateManager.GetSelectedSquad()
		if selectedSquad != 0 && clickedFactionID != cih.currentFactionID {
			cih.stateManager.SetSelectedTarget(clickedSquadID)
			cih.actionHandler.ExecuteAttack()
		}
	}
}
