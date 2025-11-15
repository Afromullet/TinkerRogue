// Package playerstate provides strongly-typed player state structures without circular dependencies.
// This package imports game_main/gear and game_main/graphics to avoid circular deps with common.
package playerstate

import (
	"game_main/common"
	"game_main/coords"
	"game_main/gear"
	"game_main/graphics"

	"github.com/bytearena/ecs"
)

// PlayerInputStates tracks input-related state for the player.
// Used in the input package to determine when button inputs are valid.
type PlayerInputStates struct {
	IsThrowing  bool
	HasKeyInput bool // Tells us whether the player pressed any key.
}

// PlayerThrowable tracks state and variables for throwing items.
// Uses strongly-typed graphics.TileBasedShape (no interface{} needed).
type PlayerThrowable struct {
	SelectedThrowableID  ecs.EntityID
	ThrowingAOEShape     graphics.TileBasedShape // Strongly typed - no type assertions needed
	ThrowableItemIndex   int
	ThrowableItemEntityID ecs.EntityID
}

// GetThrowableItemIndex returns the inventory index of the currently selected throwable.
func (pl *PlayerThrowable) GetThrowableItemIndex() int {
	return pl.ThrowableItemIndex
}

// PlayerData holds all player information that needs to be easily accessible.
// Uses strongly-typed *gear.Inventory (no interface{} needed).
type PlayerData struct {
	Throwables     PlayerThrowable
	InputStates    PlayerInputStates
	PlayerEntityID ecs.EntityID
	Pos            *coords.LogicalPosition
	Inventory      *gear.Inventory // Strongly typed - no type assertions needed
}

// PlayerAttributes retrieves the attributes component from the player entity.
func (pl *PlayerData) PlayerAttributes(ecsManager *common.EntityManager) *common.Attributes {
	attr := &common.Attributes{}
	if pl.PlayerEntityID != 0 {
		if data, ok := ecsManager.GetComponent(pl.PlayerEntityID, common.AttributeComponent); ok {
			attr = data.(*common.Attributes)
		}
	}
	return attr
}
