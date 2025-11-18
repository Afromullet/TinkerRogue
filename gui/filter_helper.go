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
// Delegates to centralized GUIQueries filtering
func (fh *FilterHelper) FilterPlayerFactionSquads(allSquads []ecs.EntityID) []ecs.EntityID {
	return fh.queries.ApplyFilterToSquads(allSquads, fh.queries.FilterSquadsByPlayer())
}

// FilterAliveSquads returns only squads that have not been destroyed
// Delegates to centralized GUIQueries filtering
func (fh *FilterHelper) FilterAliveSquads(allSquads []ecs.EntityID) []ecs.EntityID {
	return fh.queries.ApplyFilterToSquads(allSquads, fh.queries.FilterSquadsAlive())
}

// FilterFactionSquads returns only squads from a specific faction
// Delegates to centralized GUIQueries filtering
func (fh *FilterHelper) FilterFactionSquads(allSquads []ecs.EntityID, factionID ecs.EntityID) []ecs.EntityID {
	return fh.queries.ApplyFilterToSquads(allSquads, fh.queries.FilterSquadsByFaction(factionID))
}

// Note: Use GUIQueries.FindSquadsByFaction() instead of GetSquadIDsForFaction() - more efficient.
// Note: Use GUIQueries.FindAllSquads() + filter instead of GetPlayerFactionSquadIDs() for consistency.
