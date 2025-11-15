package common

import (
	"game_main/coords"

	"github.com/bytearena/ecs"
)

// Player component marker
var PlayerComponent *ecs.Component

// Player is a component marker for player entities
type Player struct {
}

// PlayerInputStates tracks input-related state for the player.
// Used in the input package to determine when button inputs are valid.
type PlayerInputStates struct {
	IsThrowing  bool
	HasKeyInput bool // Tells us whether the player pressed any key.
}

// PlayerThrowable tracks state and variables for throwing items.
// Inventory, drawing, and input systems use this.
type PlayerThrowable struct {
	SelectedThrowableID  ecs.EntityID
	ThrowableItemIndex   int
	// Note: ThrowableItemEntityID stores the entity ID. Use GetComponentType to extract components.
	ThrowableItemEntityID ecs.EntityID
}

// GetThrowableItemIndex returns the inventory index of the currently selected throwable
func (pl *PlayerThrowable) GetThrowableItemIndex() int {
	return pl.ThrowableItemIndex
}

// PlayerData holds all player information that needs to be easily accessible.
// This may work as a singleton.
type PlayerData struct {
	Throwables     PlayerThrowable
	InputStates    PlayerInputStates
	PlayerEntityID ecs.EntityID
	Pos            *coords.LogicalPosition
}

// PlayerAttributes retrieves the attributes component from the player entity.
func (pl *PlayerData) PlayerAttributes(ecsManager *EntityManager) *Attributes {
	attr := &Attributes{}
	if pl.PlayerEntityID != 0 {
		if data, ok := ecsManager.GetComponent(pl.PlayerEntityID, AttributeComponent); ok {
			attr = data.(*Attributes)
		}
	}
	return attr
}

