// Package guicomponents provides UI component and query utilities for the game
package guicomponents

import (
	"game_main/combat"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// GUIQueries provides centralized ECS query functions for all UI modes.
// This eliminates query duplication and provides a consistent query interface.
type GUIQueries struct {
	ECSManager     *common.EntityManager
	factionManager *combat.FactionManager

	// Cached ECS Views (automatically maintained by ECS library)
	squadView       *ecs.View // All SquadTag entities
	squadMemberView *ecs.View // All SquadMemberTag entities
	actionStateView *ecs.View // All ActionStateTag entities
}

// NewGUIQueries creates a new query service
func NewGUIQueries(ecsManager *common.EntityManager) *GUIQueries {
	return &GUIQueries{
		ECSManager:     ecsManager,
		factionManager: combat.NewFactionManager(ecsManager),

		// Initialize Views for cached queries (one-time O(n) cost, then O(1) access)
		squadView:       ecsManager.World.CreateView(squads.SquadTag),
		squadMemberView: ecsManager.World.CreateView(squads.SquadMemberTag),
		actionStateView: ecsManager.World.CreateView(combat.ActionStateTag),
	}
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
	factionData := combat.FindFactionDataByID(factionID, gq.ECSManager)
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

// GetSquadInfo returns complete squad information for UI display
func (gq *GUIQueries) GetSquadInfo(squadID ecs.EntityID) *SquadInfo {
	// Get squad name
	name := squads.GetSquadName(squadID, gq.ECSManager)

	// Get unit IDs
	unitIDs := squads.GetUnitIDsInSquad(squadID, gq.ECSManager)

	// Calculate HP and alive units (direct lookup instead of full query per unit)
	aliveUnits := 0
	totalHP := 0
	maxHP := 0
	for _, unitID := range unitIDs {
		attrs := common.GetAttributesByIDWithTag(gq.ECSManager, unitID, squads.SquadMemberTag)
		if attrs != nil {
			if attrs.CanAct {
				aliveUnits++
			}
			totalHP += attrs.CurrentHealth
			maxHP += attrs.MaxHealth
		}
	}

	// Get position and faction directly from squad components
	var position *coords.LogicalPosition
	var factionID ecs.EntityID

	// Get position from PositionComponent
	squadPos := common.GetComponentTypeByID[*coords.LogicalPosition](gq.ECSManager, squadID, common.PositionComponent)
	if squadPos != nil {
		pos := *squadPos
		position = &pos
	}

	// Get faction from CombatFactionComponent
	factionID = combat.GetSquadFaction(squadID, gq.ECSManager)

	// Get action state using consolidated query function
	hasActed := false
	hasMoved := false
	movementRemaining := 0
	actionState := combat.FindActionStateBySquadID(squadID, gq.ECSManager)
	if actionState != nil {
		hasActed = actionState.HasActed
		hasMoved = actionState.HasMoved
		movementRemaining = actionState.MovementRemaining
	}

	return &SquadInfo{
		ID:                squadID,
		Name:              name,
		UnitIDs:           unitIDs,
		AliveUnits:        aliveUnits,
		TotalUnits:        len(unitIDs),
		CurrentHP:         totalHP,
		MaxHP:             maxHP,
		Position:          position,
		FactionID:         factionID,
		IsDestroyed:       squads.IsSquadDestroyed(squadID, gq.ECSManager),
		HasActed:          hasActed,
		HasMoved:          hasMoved,
		MovementRemaining: movementRemaining,
	}
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
	for _, result := range gq.ECSManager.World.Query(combat.FactionTag) {
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

// ApplyFilterToSquadsCached applies a filter using pre-built cache (performance-optimized version)
// Returns filtered squad IDs as a new slice
// If filter is nil, returns all squads unchanged
func (gq *GUIQueries) ApplyFilterToSquadsCached(squadIDs []ecs.EntityID, filter SquadFilter, cache *SquadInfoCache) []ecs.EntityID {
	if filter == nil {
		return squadIDs
	}

	filtered := make([]ecs.EntityID, 0, len(squadIDs))
	for _, squadID := range squadIDs {
		info := gq.GetSquadInfoCached(squadID, cache)
		if info != nil && filter(info) {
			filtered = append(filtered, squadID)
		}
	}
	return filtered
}

// ===== CREATURE/ENTITY QUERIES =====

// CreatureInfo encapsulates all creature data needed by UI
type CreatureInfo struct {
	ID            ecs.EntityID
	Name          string
	CurrentHP     int
	MaxHP         int
	Strength      int
	Dexterity     int
	Magic         int
	Leadership    int
	Armor         int
	Weapon        int
	IsMonster     bool
	IsPlayer      bool
}

// GetCreatureAtPosition returns creature information at a specific position
// Returns nil if no creature found at the position
// Handles both monsters and players
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

	// Get creature name
	name := "Unknown"
	if nameComp, ok := gq.ECSManager.GetComponent(creatureID, common.NameComponent); ok {
		if nameData, ok := nameComp.(*common.Name); ok {
			name = nameData.NameStr
		}
	}

	// Get creature attributes
	attrs := common.GetAttributesByID(gq.ECSManager, creatureID)
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
	if monstersTag, ok := gq.ECSManager.WorldTags["monsters"]; ok {
		for _, result := range gq.ECSManager.World.Query(monstersTag) {
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

// ===== PERFORMANCE-OPTIMIZED SQUAD INFO CACHING =====

// SquadInfoCache holds pre-built lookup maps for one render cycle.
// This eliminates O(n) query scans by building maps once per frame.
type SquadInfoCache struct {
	squadNames      map[ecs.EntityID]string
	squadMembers    map[ecs.EntityID][]ecs.EntityID
	actionStates    map[ecs.EntityID]*combat.ActionStateData
	squadFactions   map[ecs.EntityID]ecs.EntityID
	destroyedStatus map[ecs.EntityID]bool
}

// BuildSquadInfoCache creates lookup maps from Views (O(squads + units + states)).
// Call once per frame/render cycle, then reuse for all squad queries.
// This replaces multiple O(n) scans with a single O(n) pass and O(1) lookups.
func (gq *GUIQueries) BuildSquadInfoCache() *SquadInfoCache {
	cache := &SquadInfoCache{
		squadNames:      make(map[ecs.EntityID]string),
		squadMembers:    make(map[ecs.EntityID][]ecs.EntityID),
		actionStates:    make(map[ecs.EntityID]*combat.ActionStateData),
		squadFactions:   make(map[ecs.EntityID]ecs.EntityID),
		destroyedStatus: make(map[ecs.EntityID]bool),
	}

	// Single pass over all squads (uses cached View, not fresh query)
	for _, result := range gq.squadView.Get() {
		entity := result.Entity
		squadData := common.GetComponentType[*squads.SquadData](entity, squads.SquadComponent)
		squadID := squadData.SquadID

		cache.squadNames[squadID] = squadData.Name
		cache.destroyedStatus[squadID] = squadData.IsDestroyed

		// Get faction if squad is in combat
		combatFaction := common.GetComponentType[*combat.CombatFactionData](entity, combat.CombatFactionComponent)
		if combatFaction != nil {
			cache.squadFactions[squadID] = combatFaction.FactionID
		}
	}

	// Single pass over all squad members (uses cached View)
	for _, result := range gq.squadMemberView.Get() {
		memberData := common.GetComponentType[*squads.SquadMemberData](result.Entity, squads.SquadMemberComponent)
		squadID := memberData.SquadID
		unitID := result.Entity.GetID()
		cache.squadMembers[squadID] = append(cache.squadMembers[squadID], unitID)
	}

	// Single pass over all action states (uses cached View)
	for _, result := range gq.actionStateView.Get() {
		actionState := common.GetComponentType[*combat.ActionStateData](result.Entity, combat.ActionStateComponent)
		cache.actionStates[actionState.SquadID] = actionState
	}

	return cache
}

// GetSquadInfoCached returns squad info using pre-built cache.
// Replaces GetSquadInfo for performance-critical paths (O(units_in_squad) vs O(all_entities)).
// Requires BuildSquadInfoCache() to be called first to generate the cache.
func (gq *GUIQueries) GetSquadInfoCached(squadID ecs.EntityID, cache *SquadInfoCache) *SquadInfo {
	// All lookups are O(1) map access (no queries!)
	name := cache.squadNames[squadID]
	unitIDs := cache.squadMembers[squadID]
	factionID := cache.squadFactions[squadID]
	isDestroyed := cache.destroyedStatus[squadID]
	actionState := cache.actionStates[squadID]

	// Calculate HP and alive units (now uses O(1) GetComponentTypeByID from Phase 1)
	aliveUnits := 0
	totalHP := 0
	maxHP := 0
	for _, unitID := range unitIDs {
		attrs := common.GetAttributesByIDWithTag(gq.ECSManager, unitID, squads.SquadMemberTag)
		if attrs != nil {
			if attrs.CanAct {
				aliveUnits++
			}
			totalHP += attrs.CurrentHealth
			maxHP += attrs.MaxHealth
		}
	}

	// Position lookup (O(1) after Phase 1 optimization)
	var position *coords.LogicalPosition
	squadPos := common.GetComponentTypeByID[*coords.LogicalPosition](gq.ECSManager, squadID, common.PositionComponent)
	if squadPos != nil {
		pos := *squadPos
		position = &pos
	}

	// Extract action state fields
	hasActed := false
	hasMoved := false
	movementRemaining := 0
	if actionState != nil {
		hasActed = actionState.HasActed
		hasMoved = actionState.HasMoved
		movementRemaining = actionState.MovementRemaining
	}

	return &SquadInfo{
		ID:                squadID,
		Name:              name,
		UnitIDs:           unitIDs,
		AliveUnits:        aliveUnits,
		TotalUnits:        len(unitIDs),
		CurrentHP:         totalHP,
		MaxHP:             maxHP,
		Position:          position,
		FactionID:         factionID,
		IsDestroyed:       isDestroyed,
		HasActed:          hasActed,
		HasMoved:          hasMoved,
		MovementRemaining: movementRemaining,
	}
}
