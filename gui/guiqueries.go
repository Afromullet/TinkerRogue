// Package gui provides UI and mode system for the game
package gui

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
	ecsManager *common.EntityManager
}

// NewGUIQueries creates a new query service
func NewGUIQueries(ecsManager *common.EntityManager) *GUIQueries {
	return &GUIQueries{ecsManager: ecsManager}
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
	factionData := combat.FindFactionDataByID(factionID, gq.ecsManager)
	if factionData == nil {
		return nil
	}

	// Get faction manager for additional data
	factionManager := combat.NewFactionManager(gq.ecsManager)
	currentMana, maxMana := factionManager.GetFactionMana(factionID)
	squadIDs := factionManager.GetFactionSquads(factionID)

	// Count alive squads
	aliveCount := 0
	for _, squadID := range squadIDs {
		if !squads.IsSquadDestroyed(squadID, gq.ecsManager) {
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

// GetFactionName returns just the faction name (optimized lightweight query)
// Queries ONLY faction entities, avoiding expensive squad enumeration
func (gq *GUIQueries) GetFactionName(factionID ecs.EntityID) string {
	factionData := combat.FindFactionDataByID(factionID, gq.ecsManager)
	if factionData == nil {
		return "Unknown Faction"
	}
	return factionData.Name
}

// IsPlayerFaction checks if faction is player-controlled (optimized lightweight query)
// Queries ONLY faction entities, avoiding expensive squad enumeration
func (gq *GUIQueries) IsPlayerFaction(factionID ecs.EntityID) bool {
	factionData := combat.FindFactionDataByID(factionID, gq.ecsManager)
	if factionData == nil {
		return false
	}
	return factionData.IsPlayerControlled
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
	name := gq.GetSquadName(squadID)

	// Get unit IDs
	unitIDs := squads.GetUnitIDsInSquad(squadID, gq.ecsManager)

	// Calculate HP and alive units
	aliveUnits := 0
	totalHP := 0
	maxHP := 0
	for _, unitID := range unitIDs {
		for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["squadmember"]) {
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
	mapPos := combat.FindMapPositionBySquadID(squadID, gq.ecsManager)
	if mapPos != nil {
		pos := mapPos.Position
		position = &pos
		factionID = mapPos.FactionID
	}

	// Get action state using consolidated query function
	hasActed := false
	hasMoved := false
	movementRemaining := 0
	actionState := combat.FindActionStateBySquadID(squadID, gq.ecsManager)
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
		IsDestroyed:       squads.IsSquadDestroyed(squadID, gq.ecsManager),
		HasActed:          hasActed,
		HasMoved:          hasMoved,
		MovementRemaining: movementRemaining,
	}
}

// GetSquadName returns the squad name
// Returns "Unknown Squad" if squad not found
func (gq *GUIQueries) GetSquadName(squadID ecs.EntityID) string {
	for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["squad"]) {
		squadData := common.GetComponentType[*squads.SquadData](
			result.Entity, squads.SquadComponent)
		if squadData.SquadID == squadID {
			return squadData.Name
		}
	}
	return "Unknown Squad"
}

// FindAllSquads returns all squad entity IDs in the game
// Uses efficient ECS query pattern with SquadComponent tag
func (gq *GUIQueries) FindAllSquads() []ecs.EntityID {
	allSquads := make([]ecs.EntityID, 0)

	// Iterate through all entities
	entityIDs := gq.ecsManager.GetAllEntities()
	for _, entityID := range entityIDs {
		// Check if entity has SquadData component
		if gq.ecsManager.HasComponent(entityID, squads.SquadComponent) {
			allSquads = append(allSquads, entityID)
		}
	}

	return allSquads
}

// GetSquadAtPosition returns the squad entity ID at the given position
// Returns 0 if no squad at position or squad is destroyed
func (gq *GUIQueries) GetSquadAtPosition(pos coords.LogicalPosition) ecs.EntityID {
	// Query all MapPosition entities to find matching position
	for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["mapposition"]) {
		mapPos := common.GetComponentType[*combat.MapPositionData](
			result.Entity, combat.MapPositionComponent)

		if mapPos.Position.X == pos.X && mapPos.Position.Y == pos.Y {
			if !squads.IsSquadDestroyed(mapPos.SquadID, gq.ecsManager) {
				return mapPos.SquadID
			}
		}
	}
	return 0
}

// FindSquadsByFaction returns all squad IDs belonging to a faction
// Returns empty slice if no squads found for faction
// Filters out destroyed squads
func (gq *GUIQueries) FindSquadsByFaction(factionID ecs.EntityID) []ecs.EntityID {
	result := make([]ecs.EntityID, 0)

	// Use consolidated query function to get all positions for faction
	mapPositions := combat.FindMapPositionByFactionID(factionID, gq.ecsManager)
	for _, mapPos := range mapPositions {
		if !squads.IsSquadDestroyed(mapPos.SquadID, gq.ecsManager) {
			result = append(result, mapPos.SquadID)
		}
	}

	return result
}

// ===== COMBAT QUERIES =====

// GetEnemySquads returns all squads not in the given faction
func (gq *GUIQueries) GetEnemySquads(currentFactionID ecs.EntityID) []ecs.EntityID {
	enemySquads := []ecs.EntityID{}

	// Get all factions except current
	allFactions := gq.GetAllFactions()
	for _, factionID := range allFactions {
		if factionID != currentFactionID {
			squadIDs := gq.FindSquadsByFaction(factionID)
			enemySquads = append(enemySquads, squadIDs...)
		}
	}

	return enemySquads
}

// GetAllFactions returns all faction IDs
func (gq *GUIQueries) GetAllFactions() []ecs.EntityID {
	factionIDs := []ecs.EntityID{}
	for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["faction"]) {
		factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
		factionIDs = append(factionIDs, factionData.FactionID)
	}
	return factionIDs
}
