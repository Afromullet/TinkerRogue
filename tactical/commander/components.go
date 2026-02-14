package commander

import (
	"github.com/bytearena/ecs"
)

// Component and tag variables
var (
	CommanderComponent            *ecs.Component
	CommanderActionStateComponent *ecs.Component
	OverworldTurnStateComponent   *ecs.Component
	CommanderRosterComponent      *ecs.Component

	CommanderTag       ecs.Tag
	CommanderActionTag ecs.Tag
	OverworldTurnTag   ecs.Tag
)

// Package-level View for zero-allocation commander queries.
var CommanderView *ecs.View

// CommanderData - core data on each commander entity
type CommanderData struct {
	CommanderID ecs.EntityID
	Name        string
	IsActive    bool
}

// CommanderActionStateData - per-commander turn tracking (modeled after combat.ActionStateData)
type CommanderActionStateData struct {
	CommanderID       ecs.EntityID
	HasMoved          bool
	HasActed          bool
	MovementRemaining int
}

// OverworldTurnStateData - singleton tracking the overworld turn
type OverworldTurnStateData struct {
	CurrentTurn int
	TurnActive  bool
}

// CommanderRosterData - on the player entity, tracks all commanders
type CommanderRosterData struct {
	CommanderIDs  []ecs.EntityID
	MaxCommanders int
}
