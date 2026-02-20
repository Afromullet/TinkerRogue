// Package framework provides UI framework infrastructure including query services
package framework

import (
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

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
		attrs := common.GetComponentTypeByID[*common.Attributes](manager, unitID, common.AttributeComponent)
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
	combatFaction := common.GetComponentType[*combat.CombatFactionData](squadEntity, combat.FactionMembershipComponent)
	if combatFaction != nil {
		factionID = combatFaction.FactionID
	}

	// Get action state if in combat
	hasActed := false
	hasMoved := false
	movementRemaining := 0
	actionState := sc.queries.CombatCache.FindActionStateBySquadID(squadID)
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
		IsDestroyed:       aliveUnits == 0,
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

// ClearAll resets the entire cache, removing all entries and dirty flags.
// Call when: a new combat starts to prevent stale data from previous combats.
func (sc *SquadInfoCache) ClearAll() {
	sc.cache = make(map[ecs.EntityID]*SquadInfo)
	sc.dirtySquads = make(map[ecs.EntityID]bool)
}

