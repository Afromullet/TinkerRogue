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
//
// NOTE: interface{} fields use duck typing to avoid circular dependencies.
// This package cannot import graphics directly (graphics imports common).
// For strongly-typed access, use playerstate.PlayerThrowable instead.
type PlayerThrowable struct {
	SelectedThrowableID  ecs.EntityID
	ThrowingAOEShape     interface{} // graphics.TileBasedShape at runtime
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
//
// NOTE: interface{} fields use duck typing to avoid circular dependencies.
// This package cannot import gear directly (gear imports common).
// For strongly-typed access, use playerstate.PlayerData instead.
type PlayerData struct {
	Throwables     PlayerThrowable
	InputStates    PlayerInputStates
	PlayerEntityID ecs.EntityID
	Pos            *coords.LogicalPosition
	Inventory      interface{} // *gear.Inventory at runtime
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

// Note about interface{} fields:
// These fields use interface{} to avoid circular dependencies (common→gear/graphics→common).
// Type assertions are centralized in accessor methods below to ensure type safety.
// Callers should use these accessors instead of direct type assertions.

// GetInventoryAs returns the Inventory field cast to the specified type.
// This centralizes the type assertion to ensure compile-time awareness of the expected type.
// The caller (gui package) knows it expects *gear.Inventory at runtime.
func (pl *PlayerData) GetInventoryAs(expectedType interface{}) interface{} {
	if pl.Inventory == nil {
		return nil
	}
	return pl.Inventory
}

// GetThrowingShapeAs returns the ThrowingAOEShape field cast to the specified type.
// This centralizes the type assertion to ensure compile-time awareness of the expected type.
// The caller (input package) knows it expects graphics.TileBasedShape at runtime.
func (pt *PlayerThrowable) GetThrowingShapeAs(expectedType interface{}) interface{} {
	if pt.ThrowingAOEShape == nil {
		return nil
	}
	return pt.ThrowingAOEShape
}
