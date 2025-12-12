package guicombat

import (
	"game_main/coords"
	"game_main/graphics"
	"game_main/gui/core"
	"game_main/gui/guicomponents"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// CombatInputHandler manages all input processing for combat mode
type CombatInputHandler struct {
	actionHandler    *CombatActionHandler
	battleMapState   *core.BattleMapState
	queries          *guicomponents.GUIQueries
	playerPos        *coords.LogicalPosition
	currentFactionID ecs.EntityID
}

// NewCombatInputHandler creates a new combat input handler
func NewCombatInputHandler(
	actionHandler *CombatActionHandler,
	battleMapState *core.BattleMapState,
	queries *guicomponents.GUIQueries,
) *CombatInputHandler {
	return &CombatInputHandler{
		actionHandler:  actionHandler,
		battleMapState: battleMapState,
		queries:        queries,
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
func (cih *CombatInputHandler) HandleInput(inputState *core.InputState) bool {
	// Handle mouse clicks
	if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
		if cih.battleMapState.InMoveMode {
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
	if cih.battleMapState.InAttackMode {
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
	// Convert mouse coordinates to tile coordinates
	if cih.playerPos == nil {
		return
	}

	// Convert screen coordinates to logical coordinates (handles both scrolling modes)
	clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)

	// Check if clicked position is in valid movement tiles
	isValidTile := false
	for _, validPos := range cih.battleMapState.ValidMoveTiles {
		if validPos.X == clickedPos.X && validPos.Y == clickedPos.Y {
			isValidTile = true
			break
		}
	}
	if !isValidTile {
		return // Invalid tile, do nothing
	}

	selectedSquad := cih.battleMapState.SelectedSquadID
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
	clickedSquadID := cih.actionHandler.combatService.GetSquadAtPosition(clickedPos)

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
	factionData := cih.queries.CombatCache.FindFactionDataByID(cih.currentFactionID, cih.queries.ECSManager)
	if factionData != nil && factionData.IsPlayerControlled {
		// If clicking an allied squad: select it
		if clickedFactionID == cih.currentFactionID {
			cih.actionHandler.SelectSquad(clickedSquadID)
			return
		}

		// If clicking an enemy squad and we have a selected squad: attack immediately
		selectedSquad := cih.battleMapState.SelectedSquadID
		if selectedSquad != 0 && clickedFactionID != cih.currentFactionID {
			cih.battleMapState.SelectedTargetID = clickedSquadID
			cih.actionHandler.ExecuteAttack()
		}
	}
}
