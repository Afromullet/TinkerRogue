package overworld

import (
	"fmt"
	"game_main/common"
	"game_main/world/coords"
)

// formatEventString is a helper for formatting event descriptions
// Centralizes string formatting for event logging
func formatEventString(format string, args ...interface{}) string {
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
