// Package guicomponents provides UI component and query utilities for the game
package guicomponents

import (
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// GUIQueries provides centralized ECS query functions for all UI modes.
// This eliminates query duplication and provides a consistent query interface.
type GUIQueries struct {
	ECSManager     *common.EntityManager
	factionManager *combat.FactionManager

	// Query caches (own Views that are automatically maintained by ECS library)
	SquadCache     *squads.SquadQueryCache
	CombatCache    *combat.CombatQueryCache
	squadInfoCache *SquadInfoCache // Event-driven cache for turn-based game

	// Cached ECS Views (automatically maintained by ECS library)
	monstersView *ecs.View // All MonsterComponent entities (GUI_PERFORMANCE_ANALYSIS.md)
}

// NewGUIQueries creates a new query service
func NewGUIQueries(ecsManager *common.EntityManager) *GUIQueries {
	gq := &GUIQueries{
		ECSManager:     ecsManager,
		factionManager: combat.NewFactionManager(ecsManager),

		// Initialize query caches (own Views that are automatically maintained by ECS library)
		SquadCache:  squads.NewSquadQueryCache(ecsManager),
		CombatCache: combat.NewCombatQueryCache(ecsManager),
	}

	// Initialize monstersView if monsters tag exists in WorldTags
	if monstersTag, ok := ecsManager.WorldTags["monsters"]; ok {
		gq.monstersView = ecsManager.World.CreateView(monstersTag)
	}

	// Initialize smart squad info cache (event-driven, not frame-level)
	gq.squadInfoCache = NewSquadInfoCache(gq)

	return gq
}

// ===== SQUAD INFO CACHE INVALIDATION =====
// These methods expose cache invalidation to other systems.
// Call these when game events occur to keep cache up-to-date.

// MarkSquadDirty marks a squad's cached info as stale.
// Call when: squad takes damage, moves, uses action, unit dies.
func (gq *GUIQueries) MarkSquadDirty(squadID ecs.EntityID) {
	gq.squadInfoCache.MarkSquadDirty(squadID)
}

// MarkAllSquadsDirty marks all cached squad info as stale.
// Call when: turn starts/ends, combat begins/ends.
func (gq *GUIQueries) MarkAllSquadsDirty() {
	gq.squadInfoCache.MarkAllDirty()
}

// InvalidateSquad completely removes a squad from cache.
// Call when: squad is destroyed or removed from game.
func (gq *GUIQueries) InvalidateSquad(squadID ecs.EntityID) {
	gq.squadInfoCache.InvalidateSquad(squadID)
}

// ===== FACTION QUERIES =====

// FactionInfo encapsulates all faction data needed by UI
type FactionInfo struct {
	ID                 ecs.EntityID
	Name               string
	IsPlayerControlled bool
	CurrentMana        int
	MaxMana            int
	SquadIDs           []ecs.EntityID
	AliveSquadCount    int
}

// GetFactionInfo returns complete faction information for UI display
func (gq *GUIQueries) GetFactionInfo(factionID ecs.EntityID) *FactionInfo {
	// Use cached query (100-500x faster than full World.Query)
	factionData := gq.CombatCache.FindFactionDataByID(factionID, gq.ECSManager)
	if factionData == nil {
		return nil
	}

	// Use stored faction manager for additional data
	currentMana, maxMana := gq.factionManager.GetFactionMana(factionID)
	squadIDs := gq.factionManager.GetFactionSquads(factionID)

	// Count alive squads
	aliveCount := 0
	for _, squadID := range squadIDs {
		if !squads.IsSquadDestroyed(squadID, gq.ECSManager) {
			aliveCount++
		}
	}

	return &FactionInfo{
		ID:                 factionID,
		Name:               factionData.Name,
		IsPlayerControlled: factionData.IsPlayerControlled,
		CurrentMana:        currentMana,
		MaxMana:            maxMana,
		SquadIDs:           squadIDs,
		AliveSquadCount:    aliveCount,
	}
}

// ===== SQUAD QUERIES =====

// SquadInfo encapsulates all squad data needed by UI
type SquadInfo struct {
	ID                ecs.EntityID
	Name              string
	UnitIDs           []ecs.EntityID
	AliveUnits        int
	TotalUnits        int
	CurrentHP         int
	MaxHP             int
	Position          *coords.LogicalPosition
	FactionID         ecs.EntityID
	IsDestroyed       bool
	HasActed          bool
	HasMoved          bool
	MovementRemaining int
}

// GetSquadInfo returns complete squad information for UI display.
// Uses event-driven cache that only rebuilds when game events invalidate data.
// This is optimal for turn-based games where data changes are discrete and infrequent.
// Performance: O(1) for cached data, only rebuilds on invalidation events.
func (gq *GUIQueries) GetSquadInfo(squadID ecs.EntityID) *SquadInfo {
	return gq.squadInfoCache.GetSquadInfo(squadID)
}

// ===== COMBAT QUERIES =====

// GetEnemySquads returns all squads not in the given faction
func (gq *GUIQueries) GetEnemySquads(currentFactionID ecs.EntityID) []ecs.EntityID {
	enemySquads := []ecs.EntityID{}

	// Get all factions except current
	allFactions := gq.GetAllFactions()
	for _, factionID := range allFactions {
		if factionID != currentFactionID {
			// Get all squads in this faction
			squadIDs := combat.GetSquadsForFaction(factionID, gq.ECSManager)
			for _, squadID := range squadIDs {
				if !squads.IsSquadDestroyed(squadID, gq.ECSManager) {
					enemySquads = append(enemySquads, squadID)
				}
			}
		}
	}

	return enemySquads
}

// GetAllFactions returns all faction IDs
func (gq *GUIQueries) GetAllFactions() []ecs.EntityID {
	factionIDs := []ecs.EntityID{}
	// Use cached View instead of Query (avoids 30,000+ map allocations per second)
	for _, result := range gq.CombatCache.FactionView.Get() {
		factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
		factionIDs = append(factionIDs, factionData.FactionID)
	}
	return factionIDs
}

// ===== UNIFIED SQUAD FILTERING =====

// FilterSquadsAlive returns a filter that matches non-destroyed squads
func (gq *GUIQueries) FilterSquadsAlive() SquadFilter {
	return func(info *SquadInfo) bool {
		return !info.IsDestroyed
	}
}

// ApplyFilterToSquads applies a filter to a slice of squad IDs
// Returns filtered squad IDs as a new slice
// If filter is nil, returns all squads unchanged
// Note: For performance-critical paths, use ApplyFilterToSquadsCached instead
func (gq *GUIQueries) ApplyFilterToSquads(squadIDs []ecs.EntityID, filter SquadFilter) []ecs.EntityID {
	if filter == nil {
		return squadIDs
	}

	filtered := make([]ecs.EntityID, 0, len(squadIDs))
	for _, squadID := range squadIDs {
		info := gq.GetSquadInfo(squadID)
		if info != nil && filter(info) {
			filtered = append(filtered, squadID)
		}
	}
	return filtered
}

// ===== CREATURE/ENTITY QUERIES =====

// CreatureInfo encapsulates all creature data needed by UI
type CreatureInfo struct {
	ID         ecs.EntityID
	Name       string
	CurrentHP  int
	MaxHP      int
	Strength   int
	Dexterity  int
	Magic      int
	Leadership int
	Armor      int
	Weapon     int
	IsMonster  bool
	IsPlayer   bool
}

// GetCreatureAtPosition returns creature information at a specific position
// Returns nil if no creature found at the position
// Handles both monsters and players
// Optimized: Batches all component lookups into single GetEntityByID call.
func (gq *GUIQueries) GetCreatureAtPosition(pos coords.LogicalPosition) *CreatureInfo {
	// First try the common helper (looks for monsters)
	creatureID := common.GetCreatureAtPosition(gq.ECSManager, &pos)

	// If no monster found, check if there's any entity at the position
	if creatureID == 0 && common.GlobalPositionSystem != nil {
		creatureID = common.GlobalPositionSystem.GetEntityIDAt(pos)
	}

	if creatureID == 0 {
		return nil
	}

	// This avoids multiple GetEntityByID allocations
	entity := gq.ECSManager.FindEntityByID(creatureID)
	if entity == nil {
		return nil
	}

	// Get creature name from entity
	name := "Unknown"
	nameComp := common.GetComponentType[*common.Name](entity, common.NameComponent)
	if nameComp != nil {
		name = nameComp.NameStr
	}

	// Get creature attributes from entity
	attrs := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attrs == nil {
		// Return basic info if no attributes
		return &CreatureInfo{
			ID:        creatureID,
			Name:      name,
			IsMonster: gq.isMonster(creatureID),
			IsPlayer:  gq.isPlayer(creatureID),
		}
	}

	// Return full creature info
	return &CreatureInfo{
		ID:         creatureID,
		Name:       name,
		CurrentHP:  attrs.CurrentHealth,
		MaxHP:      attrs.MaxHealth,
		Strength:   attrs.Strength,
		Dexterity:  attrs.Dexterity,
		Magic:      attrs.Magic,
		Leadership: attrs.Leadership,
		Armor:      attrs.Armor,
		Weapon:     attrs.Weapon,
		IsMonster:  gq.isMonster(creatureID),
		IsPlayer:   gq.isPlayer(creatureID),
	}
}

// isMonster checks if an entity is a monster
func (gq *GUIQueries) isMonster(entityID ecs.EntityID) bool {
	// Use cached View instead of Query (avoids 30,000+ map allocations per second)
	if gq.monstersView != nil {
		for _, result := range gq.monstersView.Get() {
			if result.Entity.GetID() == entityID {
				return true
			}
		}
	}
	return false
}

// isPlayer checks if an entity is the player
func (gq *GUIQueries) isPlayer(entityID ecs.EntityID) bool {
	return gq.ECSManager.HasComponent(entityID, common.PlayerComponent)
}

// ===== TILE QUERIES =====

// TileInfo encapsulates tile data needed by UI
type TileInfo struct {
	Position     coords.LogicalPosition
	TileType     string
	MovementCost int
	IsWalkable   bool
	HasEntity    bool
	EntityID     ecs.EntityID
}

// GetTileInfo returns information about a tile at a specific position
// This is a basic implementation - extend based on your tile system
func (gq *GUIQueries) GetTileInfo(pos coords.LogicalPosition) *TileInfo {
	info := &TileInfo{
		Position:     pos,
		TileType:     "Floor", // Default - extend with actual tile system
		MovementCost: 1,       // Default - extend with actual tile system
		IsWalkable:   true,    // Default - extend with actual tile system
	}

	// Check if there's an entity at this position
	if common.GlobalPositionSystem != nil {
		entityID := common.GlobalPositionSystem.GetEntityIDAt(pos)
		if entityID != 0 {
			info.HasEntity = true
			info.EntityID = entityID
		}
	}

	return info
}
