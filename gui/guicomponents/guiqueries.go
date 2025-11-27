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
	ECSManager *common.EntityManager
}

// NewGUIQueries creates a new query service
func NewGUIQueries(ecsManager *common.EntityManager) *GUIQueries {
	return &GUIQueries{ECSManager: ecsManager}
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

	// Get faction manager for additional data
	factionManager := combat.NewFactionManager(gq.ECSManager)
	currentMana, maxMana := factionManager.GetFactionMana(factionID)
	squadIDs := factionManager.GetFactionSquads(factionID)

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

	// Calculate HP and alive units
	aliveUnits := 0
	totalHP := 0
	maxHP := 0
	for _, unitID := range unitIDs {
		for _, result := range gq.ECSManager.World.Query(squads.SquadMemberTag) {
			if result.Entity.GetID() == unitID {
				attrs := common.GetComponentType[*common.Attributes](result.Entity, common.AttributeComponent)
				if attrs.CanAct {
					aliveUnits++
				}
				totalHP += attrs.CurrentHealth
				maxHP += attrs.MaxHealth
			}
		}
	}

	// Get position and faction using consolidated query function
	var position *coords.LogicalPosition
	var factionID ecs.EntityID
	mapPos := combat.FindMapPositionBySquadID(squadID, gq.ECSManager)
	if mapPos != nil {
		pos := mapPos.Position
		position = &pos
		factionID = mapPos.FactionID
	}

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
			// Get all positions for faction and extract squad IDs
			mapPositions := combat.FindMapPositionByFactionID(factionID, gq.ECSManager)
			for _, mapPos := range mapPositions {
				if !squads.IsSquadDestroyed(mapPos.SquadID, gq.ECSManager) {
					enemySquads = append(enemySquads, mapPos.SquadID)
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

// FilterSquadsByPlayer returns a filter that matches player faction squads
func (gq *GUIQueries) FilterSquadsByPlayer() SquadFilter {
	return func(info *SquadInfo) bool {
		factionData := combat.FindFactionDataByID(info.FactionID, gq.ECSManager)
		return !info.IsDestroyed && factionData != nil && factionData.IsPlayerControlled
	}
}

// FilterSquadsByFaction returns a filter that matches squads in a specific faction
func (gq *GUIQueries) FilterSquadsByFaction(factionID ecs.EntityID) SquadFilter {
	return func(info *SquadInfo) bool {
		return !info.IsDestroyed && info.FactionID == factionID
	}
}

// ApplyFilterToSquads applies a filter to a slice of squad IDs
// Returns filtered squad IDs as a new slice
// If filter is nil, returns all squads unchanged
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

// GetPlayerSquads returns all squads belonging to the player faction (alive only)
func (gq *GUIQueries) GetPlayerSquads() []ecs.EntityID {
	allSquads := squads.FindAllSquads(gq.ECSManager)
	return gq.ApplyFilterToSquads(allSquads, gq.FilterSquadsByPlayer())
}

// GetAliveSquads returns all squads that have not been destroyed
func (gq *GUIQueries) GetAliveSquads() []ecs.EntityID {
	allSquads := squads.FindAllSquads(gq.ECSManager)
	return gq.ApplyFilterToSquads(allSquads, gq.FilterSquadsAlive())
}

// GetSquadVisualization returns ASCII grid of squad formation
func (gq *GUIQueries) GetSquadVisualization(squadID ecs.EntityID) string {
	return squads.VisualizeSquad(squadID, gq.ECSManager)
}

// GetSquadUnitIDs returns all unit entity IDs in squad
func (gq *GUIQueries) GetSquadUnitIDs(squadID ecs.EntityID) []ecs.EntityID {
	return squads.GetUnitIDsInSquad(squadID, gq.ECSManager)
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
