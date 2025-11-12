// Package gui provides UI and mode system for the game
package gui

import (
	"github.com/bytearena/ecs"
)

// FilterHelper provides reusable squad filtering utilities
// Eliminates duplicated filtering logic across modes
type FilterHelper struct {
	queries *GUIQueries
}

// NewFilterHelper creates a new filter helper instance
func NewFilterHelper(queries *GUIQueries) *FilterHelper {
	return &FilterHelper{
		queries: queries,
	}
}

// FilterPlayerFactionSquads returns only squads from the player faction
// Filters out destroyed squads and enemy squads
func (fh *FilterHelper) FilterPlayerFactionSquads(allSquads []ecs.EntityID) []ecs.EntityID {
	filtered := make([]ecs.EntityID, 0, len(allSquads))

	for _, squadID := range allSquads {
		info := fh.queries.GetSquadInfo(squadID)
		if info == nil || info.IsDestroyed {
			continue
		}

		// Only include player faction squads
		if fh.queries.IsPlayerFaction(info.FactionID) {
			filtered = append(filtered, squadID)
		}
	}

	return filtered
}

// FilterAliveSquads returns only squads that have not been destroyed
func (fh *FilterHelper) FilterAliveSquads(allSquads []ecs.EntityID) []ecs.EntityID {
	filtered := make([]ecs.EntityID, 0, len(allSquads))

	for _, squadID := range allSquads {
		info := fh.queries.GetSquadInfo(squadID)
		if info != nil && !info.IsDestroyed {
			filtered = append(filtered, squadID)
		}
	}

	return filtered
}

// FilterFactionSquads returns only squads from a specific faction
func (fh *FilterHelper) FilterFactionSquads(allSquads []ecs.EntityID, factionID ecs.EntityID) []ecs.EntityID {
	filtered := make([]ecs.EntityID, 0, len(allSquads))

	for _, squadID := range allSquads {
		info := fh.queries.GetSquadInfo(squadID)
		if info != nil && !info.IsDestroyed && info.FactionID == factionID {
			filtered = append(filtered, squadID)
		}
	}

	return filtered
}

// Note: Use GUIQueries.FindSquadsByFaction() instead of GetSquadIDsForFaction() - more efficient.
// Note: Use GUIQueries.FindAllSquads() + filter instead of GetPlayerFactionSquadIDs() for consistency.
