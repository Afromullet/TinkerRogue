package core

import (
	"fmt"
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// FormatEventString is a helper for formatting event descriptions
// Centralizes string formatting for event logging
func FormatEventString(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

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

// MapFactionToThreatType converts faction type to corresponding threat type
// Used when factions spawn new threat nodes
func MapFactionToThreatType(factionType FactionType) ThreatType {
	switch factionType {
	case FactionNecromancers:
		return ThreatNecromancer
	case FactionBandits:
		return ThreatBanditCamp
	case FactionOrcs:
		return ThreatOrcWarband
	case FactionBeasts:
		return ThreatBeastNest
	case FactionCultists:
		return ThreatCorruption
	default:
		return ThreatBanditCamp
	}
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

// IsThreatAtPosition checks if any threat node exists at the given position
func IsThreatAtPosition(manager *common.EntityManager, pos coords.LogicalPosition) bool {
	return GetThreatNodeAt(manager, pos) != 0
}

// GetThreatNodeAt returns the EntityID of a threat node at a specific position.
// Returns 0 if no threat exists at the position.
// Prefer using queries directly when iterating over multiple threats.
func GetThreatNodeAt(manager *common.EntityManager, pos coords.LogicalPosition) ecs.EntityID {
	entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(pos)
	for _, entityID := range entityIDs {
		if manager.HasComponent(entityID, ThreatNodeComponent) {
			return entityID
		}
	}
	return 0
}

// SquadChecker is an interface for checking squad status without circular dependency
// This interface allows the overworld package to query squad status without importing the squads package
type SquadChecker interface {
	// HasActiveSquads returns true if the player has any squads with living units
	HasActiveSquads(manager *common.EntityManager) bool
}

// squadChecker is the injected implementation for squad checking
// Set this in main package initialization via SetSquadChecker()
var squadChecker SquadChecker

// SetSquadChecker injects the squad checking implementation
// Call this from main package after squads package is initialized
func SetSquadChecker(checker SquadChecker) {
	squadChecker = checker
}

// GetSquadChecker returns the injected squad checker
func GetSquadChecker() SquadChecker {
	return squadChecker
}
