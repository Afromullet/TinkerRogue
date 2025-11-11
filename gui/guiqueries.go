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
	for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["faction"]) {
		factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
		if factionData.FactionID == factionID {
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
	}
	return nil
}

// GetFactionName returns just the faction name (lightweight query)
func (gq *GUIQueries) GetFactionName(factionID ecs.EntityID) string {
	info := gq.GetFactionInfo(factionID)
	if info != nil {
		return info.Name
	}
	return "Unknown Faction"
}

// IsPlayerFaction checks if faction is player-controlled
func (gq *GUIQueries) IsPlayerFaction(factionID ecs.EntityID) bool {
	info := gq.GetFactionInfo(factionID)
	return info != nil && info.IsPlayerControlled
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

	// Get position and faction
	var position *coords.LogicalPosition
	var factionID ecs.EntityID
	for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["mapposition"]) {
		mapPos := common.GetComponentType[*combat.MapPositionData](result.Entity, combat.MapPositionComponent)
		if mapPos.SquadID == squadID {
			pos := mapPos.Position
			position = &pos
			factionID = mapPos.FactionID
			break
		}
	}

	// Get action state
	hasActed := false
	hasMoved := false
	movementRemaining := 0
	for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["actionstate"]) {
		actionState := common.GetComponentType[*combat.ActionStateData](result.Entity, combat.ActionStateComponent)
		if actionState.SquadID == squadID {
			hasActed = actionState.HasActed
			hasMoved = actionState.HasMoved
			movementRemaining = actionState.MovementRemaining
			break
		}
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

// GetSquadName returns the squad name (delegates to existing function)
func (gq *GUIQueries) GetSquadName(squadID ecs.EntityID) string {
	return GetSquadName(gq.ecsManager, squadID)
}

// FindAllSquads returns all squad IDs (delegates to existing function)
func (gq *GUIQueries) FindAllSquads() []ecs.EntityID {
	return FindAllSquads(gq.ecsManager)
}

// GetSquadAtPosition finds squad at given position (delegates to existing function)
func (gq *GUIQueries) GetSquadAtPosition(pos coords.LogicalPosition) ecs.EntityID {
	return GetSquadAtPosition(gq.ecsManager, pos)
}

// FindSquadsByFaction returns squads for a faction (delegates to existing function)
func (gq *GUIQueries) FindSquadsByFaction(factionID ecs.EntityID) []ecs.EntityID {
	return FindSquadsByFaction(gq.ecsManager, factionID)
}

// ===== COMBAT QUERIES =====

// GetEnemySquads returns all squads not in the given faction
func (gq *GUIQueries) GetEnemySquads(currentFactionID ecs.EntityID) []ecs.EntityID {
	enemySquads := []ecs.EntityID{}
	for _, result := range gq.ecsManager.World.Query(gq.ecsManager.Tags["mapposition"]) {
		mapPos := common.GetComponentType[*combat.MapPositionData](result.Entity, combat.MapPositionComponent)
		if mapPos.FactionID != currentFactionID {
			if !squads.IsSquadDestroyed(mapPos.SquadID, gq.ecsManager) {
				enemySquads = append(enemySquads, mapPos.SquadID)
			}
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
