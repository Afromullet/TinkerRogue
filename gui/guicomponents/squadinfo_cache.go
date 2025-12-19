// Package guicomponents provides UI component and query utilities for the game
package guicomponents

import (
	"game_main/combat"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// SquadInfoCache provides event-driven caching of SquadInfo.
type SquadInfoCache struct {
	cache       map[ecs.EntityID]*SquadInfo
	dirtySquads map[ecs.EntityID]bool
	queries     *GUIQueries
}

// NewSquadInfoCache creates a new event-driven squad info cache.
func NewSquadInfoCache(queries *GUIQueries) *SquadInfoCache {
	return &SquadInfoCache{
		cache:       make(map[ecs.EntityID]*SquadInfo),
		dirtySquads: make(map[ecs.EntityID]bool),
		queries:     queries,
	}
}

// GetSquadInfo returns cached squad info, rebuilding only if marked dirty.
// This is O(1) for cached data and only rebuilds when events invalidate the cache.
func (sc *SquadInfoCache) GetSquadInfo(squadID ecs.EntityID) *SquadInfo {
	// Check if we need to rebuild this squad's info
	if sc.dirtySquads[squadID] || sc.cache[squadID] == nil {
		sc.cache[squadID] = sc.buildSquadInfo(squadID)
		delete(sc.dirtySquads, squadID)
	}
	return sc.cache[squadID]
}

// buildSquadInfo constructs complete squad info from ECS components.
// This is called only when the cache is invalid or missing.
func (sc *SquadInfoCache) buildSquadInfo(squadID ecs.EntityID) *SquadInfo {
	manager := sc.queries.ECSManager

	// Get squad entity
	queryResult := manager.World.GetEntityByID(squadID)
	if queryResult == nil {
		return nil
	}
	squadEntity := queryResult.Entity

	// Get squad data
	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	if squadData == nil {
		return nil
	}

	// Get squad members
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)

	// Calculate HP and alive units
	aliveUnits := 0
	totalHP := 0
	maxHP := 0
	for _, unitID := range unitIDs {
		attrs := common.GetAttributesByID(manager, unitID)
		if attrs != nil {
			if attrs.CanAct {
				aliveUnits++
			}
			totalHP += attrs.CurrentHealth
			maxHP += attrs.MaxHealth
		}
	}

	// Get position
	var position *coords.LogicalPosition
	squadPos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
	if squadPos != nil {
		pos := *squadPos
		position = &pos
	}

	// Get faction if squad is in combat
	var factionID ecs.EntityID
	combatFaction := common.GetComponentType[*combat.CombatFactionData](squadEntity, combat.CombatFactionComponent)
	if combatFaction != nil {
		factionID = combatFaction.FactionID
	}

	// Get action state if in combat
	hasActed := false
	hasMoved := false
	movementRemaining := 0
	actionState := sc.queries.CombatCache.FindActionStateBySquadID(squadID, manager)
	if actionState != nil {
		hasActed = actionState.HasActed
		hasMoved = actionState.HasMoved
		movementRemaining = actionState.MovementRemaining
	}

	return &SquadInfo{
		ID:                squadID,
		Name:              squadData.Name,
		UnitIDs:           unitIDs,
		AliveUnits:        aliveUnits,
		TotalUnits:        len(unitIDs),
		CurrentHP:         totalHP,
		MaxHP:             maxHP,
		Position:          position,
		FactionID:         factionID,
		IsDestroyed:       squadData.IsDestroyed,
		HasActed:          hasActed,
		HasMoved:          hasMoved,
		MovementRemaining: movementRemaining,
	}
}

// ===== INVALIDATION METHODS =====
// Call these when game events occur to mark cached data as stale.

// MarkSquadDirty marks a single squad's cache as invalid.
// Call when: squad takes damage, moves, uses action, unit dies, etc.
func (sc *SquadInfoCache) MarkSquadDirty(squadID ecs.EntityID) {
	sc.dirtySquads[squadID] = true
}

// MarkSquadsDirty marks multiple squads as dirty at once.
// Useful for batch invalidations (e.g., all squads in a faction).
func (sc *SquadInfoCache) MarkSquadsDirty(squadIDs []ecs.EntityID) {
	for _, squadID := range squadIDs {
		sc.dirtySquads[squadID] = true
	}
}

// MarkAllDirty marks all cached squads as dirty.
// Call when: turn starts/ends, combat begins/ends, global state changes.
func (sc *SquadInfoCache) MarkAllDirty() {
	for squadID := range sc.cache {
		sc.dirtySquads[squadID] = true
	}
}

// InvalidateSquad completely removes a squad from cache.
// Call when: squad is destroyed or removed from game.
func (sc *SquadInfoCache) InvalidateSquad(squadID ecs.EntityID) {
	delete(sc.cache, squadID)
	delete(sc.dirtySquads, squadID)
}

// InvalidateAll completely clears the cache.
// Call when: major game state changes (scene transitions, combat end).
func (sc *SquadInfoCache) InvalidateAll() {
	sc.cache = make(map[ecs.EntityID]*SquadInfo)
	sc.dirtySquads = make(map[ecs.EntityID]bool)
}

// ===== UTILITY METHODS =====

// GetCacheStats returns cache statistics for debugging/profiling.
func (sc *SquadInfoCache) GetCacheStats() (cached int, dirty int) {
	return len(sc.cache), len(sc.dirtySquads)
}

// PreloadSquads pre-builds cache for a list of squads.
// Useful for pre-warming cache before expensive operations.
func (sc *SquadInfoCache) PreloadSquads(squadIDs []ecs.EntityID) {
	for _, squadID := range squadIDs {
		sc.GetSquadInfo(squadID) // Triggers build if not cached
	}
}
