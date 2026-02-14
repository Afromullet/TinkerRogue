package behavior

import (
	"game_main/common"
	"game_main/mind/evaluation"
	"game_main/tactical/combat"

	"github.com/bytearena/ecs"
)

// FactionThreatLevelManager keeps track of each faction's threat levels.
type FactionThreatLevelManager struct {
	manager  *common.EntityManager
	cache    *combat.CombatQueryCache
	factions map[ecs.EntityID]*FactionThreatLevel
}

func NewFactionThreatLevelManager(manager *common.EntityManager, cache *combat.CombatQueryCache) *FactionThreatLevelManager {
	return &FactionThreatLevelManager{
		manager:  manager,
		cache:    cache,
		factions: make(map[ecs.EntityID]*FactionThreatLevel),
	}
}

func (ftlm *FactionThreatLevelManager) AddFaction(factionID ecs.EntityID) {

	if _, exists := ftlm.factions[factionID]; !exists {
		ftlm.factions[factionID] = NewFactionThreatLevel(factionID, ftlm.manager, ftlm.cache)
	}

	ftlm.factions[factionID].UpdateThreatRatings()
}

func (ftlm *FactionThreatLevelManager) UpdateFaction(factionID ecs.EntityID) {
	if faction, exists := ftlm.factions[factionID]; exists {
		faction.UpdateThreatRatings()
	}
}

func (ftlm *FactionThreatLevelManager) UpdateAllFactions() {
	for _, faction := range ftlm.factions {
		faction.UpdateThreatRatings()
	}
}

type FactionThreatLevel struct {
	manager           *common.EntityManager
	cache             *combat.CombatQueryCache
	factionID         ecs.EntityID
	squadThreatLevels map[ecs.EntityID]*SquadThreatLevel //Key is the squad ID. Value is the danger level
}

func NewFactionThreatLevel(factionID ecs.EntityID, manager *common.EntityManager, cache *combat.CombatQueryCache) *FactionThreatLevel {

	squadIDs := combat.GetSquadsForFaction(factionID, manager)

	ftl := &FactionThreatLevel{

		factionID:         factionID,
		squadThreatLevels: make(map[ecs.EntityID]*SquadThreatLevel, len(squadIDs)),
		manager:           manager,
		cache:             cache,
	}

	for _, ID := range squadIDs {
		ftl.squadThreatLevels[ID] = NewSquadThreatLevel(ftl.manager, ftl.cache, ID)
	}

	return ftl
}

func (ftr *FactionThreatLevel) UpdateThreatRatings() {

	squadIDs := combat.GetSquadsForFaction(ftr.factionID, ftr.manager)

	for _, squadID := range squadIDs {
		// Create threat level entry if squad wasn't tracked at creation time
		if _, exists := ftr.squadThreatLevels[squadID]; !exists {
			ftr.squadThreatLevels[squadID] = NewSquadThreatLevel(ftr.manager, ftr.cache, squadID)
		}
		// Update threat calculations
		ftr.squadThreatLevels[squadID].CalculateThreatLevels()
	}

}

type SquadThreatLevel struct {
	manager       *common.EntityManager
	cache         *combat.CombatQueryCache
	squadID       ecs.EntityID
	ThreatByRange map[int]float64 //Key is the range. Value is the danger level. How dangerous the squad is at each range
}

func NewSquadThreatLevel(manager *common.EntityManager, cache *combat.CombatQueryCache, squadID ecs.EntityID) *SquadThreatLevel {

	return &SquadThreatLevel{
		manager: manager,
		cache:   cache,
		squadID: squadID,
	}
}

// CalculateThreatLevels computes threat ratings for the squad.
// Uses shared power calculation from evaluation package for ThreatByRange.
func (stl *SquadThreatLevel) CalculateThreatLevels() {
	// Use shared power calculation for ThreatByRange
	config := evaluation.GetPowerConfigByProfile("Balanced")
	stl.ThreatByRange = evaluation.CalculateSquadPowerByRange(stl.squadID, stl.manager, config)
}
