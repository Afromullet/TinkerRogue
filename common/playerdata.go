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
	SelectedThrowable  *ecs.Entity
	ThrowingAOEShape   interface{} // graphics.TileBasedShape - stored as interface{} to avoid circular dependency
	ThrowableItemIndex int
	// Note: ThrowableItem stores the entity reference. Use gear.GetItem() to extract the Item component.
	ThrowableItemEntity *ecs.Entity
}

// GetThrowableItemIndex returns the inventory index of the currently selected throwable
func (pl *PlayerThrowable) GetThrowableItemIndex() int {
	return pl.ThrowableItemIndex
}

// GetThrowableItemEntity returns the entity reference for the throwable item
func (pl *PlayerThrowable) GetThrowableItemEntity() *ecs.Entity {
	return pl.ThrowableItemEntity
}

// PlayerData holds all player information that needs to be easily accessible.
// This may work as a singleton.
type PlayerData struct {
	Throwables   PlayerThrowable
	InputStates  PlayerInputStates
	PlayerEntity *ecs.Entity
	Pos          *coords.LogicalPosition
	Inventory    interface{} // Stores *gear.Inventory - stored as interface{} to avoid circular dependency
}

// PlayerAttributes retrieves the attributes component from the player entity.
func (pl *PlayerData) PlayerAttributes() *Attributes {
	attr := &Attributes{}
	if data, ok := pl.PlayerEntity.GetComponentData(AttributeComponent); ok {
		attr = data.(*Attributes)
	}
	return attr
}
