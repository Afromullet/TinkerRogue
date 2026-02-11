package core

import (
	"game_main/common"
	"game_main/templates"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// GetCurrentTick returns the current tick from the tick state singleton.
// Returns 0 if no tick state exists.
func GetCurrentTick(manager *common.EntityManager) int64 {
	if tickState := GetTickState(manager); tickState != nil {
		return tickState.CurrentTick
	}
	return 0
}

// GetTickState retrieves the singleton tick state
func GetTickState(manager *common.EntityManager) *TickStateData {
	for _, result := range manager.World.Query(TickStateTag) {
		return common.GetComponentType[*TickStateData](result.Entity, TickStateComponent)
	}
	return nil
}

// MapFactionToThreatType converts faction type to corresponding threat type.
// Uses NodeRegistry for data-driven lookup.
// Used when factions spawn new threat nodes.
func MapFactionToThreatType(factionType FactionType) ThreatType {
	return GetNodeRegistry().GetThreatTypeForFaction(factionType)
}

// CalculateBaseMagnitude derives the influence base magnitude from threat intensity.
// Single source of truth for this formula — used at creation, evolution, and post-combat.
func CalculateBaseMagnitude(intensity int) float64 {
	return float64(intensity) * templates.InfluenceConfigTemplate.BaseMagnitudeMultiplier
}

// GetCardinalNeighbors returns the 4 adjacent positions (up, down, left, right)
// Use this instead of manually constructing adjacent position arrays
func GetCardinalNeighbors(pos coords.LogicalPosition) []coords.LogicalPosition {
	return []coords.LogicalPosition{
		{X: pos.X + 1, Y: pos.Y},
		{X: pos.X - 1, Y: pos.Y},
		{X: pos.X, Y: pos.Y + 1},
		{X: pos.X, Y: pos.Y - 1},
	}
}

// GetRandomTileFromSlice returns a random tile from a slice, or nil if empty
func GetRandomTileFromSlice(tiles []coords.LogicalPosition) *coords.LogicalPosition {
	if len(tiles) == 0 {
		return nil
	}
	tile := tiles[common.RandomInt(len(tiles))]
	return &tile
}

// IsThreatAtPosition checks if any threat node exists at the given position.
// Uses unified OverworldNodeComponent, filters by hostile owner.
func IsThreatAtPosition(manager *common.EntityManager, pos coords.LogicalPosition) bool {
	return GetThreatNodeAt(manager, pos) != 0
}

// IsAnyNodeAtPosition checks if any overworld node exists at the given position.
// Uses unified OverworldNodeComponent.
func IsAnyNodeAtPosition(manager *common.EntityManager, pos coords.LogicalPosition) bool {
	entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(pos)
	for _, entityID := range entityIDs {
		if manager.HasComponent(entityID, OverworldNodeComponent) {
			return true
		}
	}
	return false
}

// GetThreatNodeAt returns the EntityID of a threat node at a specific position.
// Returns 0 if no threat exists at the position.
// Uses unified OverworldNodeComponent, filters by hostile owner.
func GetThreatNodeAt(manager *common.EntityManager, pos coords.LogicalPosition) ecs.EntityID {
	entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(pos)
	for _, entityID := range entityIDs {
		if !manager.HasComponent(entityID, OverworldNodeComponent) {
			continue
		}
		data := common.GetComponentTypeByID[*OverworldNodeData](manager, entityID, OverworldNodeComponent)
		if data != nil && IsHostileOwner(data.OwnerID) {
			return entityID
		}
	}
	return 0
}
