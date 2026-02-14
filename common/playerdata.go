package common

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// Component markers
var (
	PlayerComponent *ecs.Component // Marks player entity

)

// Player is a component marker for player entities
type Player struct {
}

// PlayerInputStates tracks input-related state for the player.
// Used in the input package to determine when button inputs are valid.
type PlayerInputStates struct {
	HasKeyInput bool // Tells us whether the player pressed any key.
}

// PlayerData holds all player information that needs to be easily accessible.
// This may work as a singleton.
type PlayerData struct {
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
