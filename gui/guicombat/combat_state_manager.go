package guicombat

import (
	"game_main/coords"

	"github.com/bytearena/ecs"
)

// AttackMode represents the current attack state
type AttackMode int

const (
	AttackModeNone AttackMode = iota
	AttackModeActive
	AttackModeSelectingTarget
)

// CombatStateManager tracks combat state and transitions
type CombatStateManager struct {
	selectedSquadID  ecs.EntityID
	selectedTargetID ecs.EntityID
	inAttackMode     bool
	inMoveMode       bool
	validMoveTiles   []coords.LogicalPosition
}

// NewCombatStateManager creates a new combat state manager
func NewCombatStateManager() *CombatStateManager {
	return &CombatStateManager{
		selectedSquadID:  0,
		selectedTargetID: 0,
		inAttackMode:     false,
		inMoveMode:       false,
		validMoveTiles:   []coords.LogicalPosition{},
	}
}

// SetAttackMode enables or disables attack mode
func (csm *CombatStateManager) SetAttackMode(enable bool) {
	if enable && csm.selectedSquadID == 0 {
		return // Cannot enable attack mode without selected squad
	}
	csm.inAttackMode = enable
	if enable {
		csm.inMoveMode = false // Disable move mode
	} else {
		csm.selectedTargetID = 0 // Clear target when disabling
	}
}

// SetMoveMode enables or disables move mode
func (csm *CombatStateManager) SetMoveMode(enable bool, validTiles []coords.LogicalPosition) {
	if enable && csm.selectedSquadID == 0 {
		return // Cannot enable move mode without selected squad
	}
	csm.inMoveMode = enable
	if enable {
		csm.inAttackMode = false // Disable attack mode
		csm.validMoveTiles = validTiles
	} else {
		csm.validMoveTiles = []coords.LogicalPosition{} // Clear tiles
	}
}

// SetSelectedSquad sets the currently selected squad
func (csm *CombatStateManager) SetSelectedSquad(squadID ecs.EntityID) {
	csm.selectedSquadID = squadID
	csm.inAttackMode = false
	csm.inMoveMode = false
	csm.selectedTargetID = 0
	csm.validMoveTiles = []coords.LogicalPosition{}
}

// SetSelectedTarget sets the target squad for attack
func (csm *CombatStateManager) SetSelectedTarget(targetID ecs.EntityID) {
	csm.selectedTargetID = targetID
}

// GetSelectedSquad returns the currently selected squad
func (csm *CombatStateManager) GetSelectedSquad() ecs.EntityID {
	return csm.selectedSquadID
}

// GetSelectedTarget returns the current target squad
func (csm *CombatStateManager) GetSelectedTarget() ecs.EntityID {
	return csm.selectedTargetID
}

// IsAttackMode returns true if in attack mode
func (csm *CombatStateManager) IsAttackMode() bool {
	return csm.inAttackMode
}

// IsMoveMode returns true if in move mode
func (csm *CombatStateManager) IsMoveMode() bool {
	return csm.inMoveMode
}

// GetValidMoveTiles returns the list of valid movement tiles
func (csm *CombatStateManager) GetValidMoveTiles() []coords.LogicalPosition {
	tiles := make([]coords.LogicalPosition, len(csm.validMoveTiles))
	copy(tiles, csm.validMoveTiles)
	return tiles
}

// IsValidMoveTile checks if a position is in the list of valid moves
func (csm *CombatStateManager) IsValidMoveTile(pos coords.LogicalPosition) bool {
	for _, validPos := range csm.validMoveTiles {
		if validPos.X == pos.X && validPos.Y == pos.Y {
			return true
		}
	}
	return false
}

// Reset clears all state (used when turn changes)
func (csm *CombatStateManager) Reset() {
	csm.selectedSquadID = 0
	csm.selectedTargetID = 0
	csm.inAttackMode = false
	csm.inMoveMode = false
	csm.validMoveTiles = []coords.LogicalPosition{}
}

// GetState returns a snapshot of the current state
type CombatState struct {
	SelectedSquadID  ecs.EntityID
	SelectedTargetID ecs.EntityID
	InAttackMode     bool
	InMoveMode       bool
	ValidTileCount   int
}

func (csm *CombatStateManager) GetState() CombatState {
	return CombatState{
		SelectedSquadID:  csm.selectedSquadID,
		SelectedTargetID: csm.selectedTargetID,
		InAttackMode:     csm.inAttackMode,
		InMoveMode:       csm.inMoveMode,
		ValidTileCount:   len(csm.validMoveTiles),
	}
}
