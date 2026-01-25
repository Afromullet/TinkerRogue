package overworld

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// GetFactionByID retrieves faction data by entity ID
func GetFactionByID(manager *common.EntityManager, factionID ecs.EntityID) *OverworldFactionData {
	entity := manager.FindEntityByID(factionID)
	if entity == nil {
		return nil
	}
	return common.GetComponentType[*OverworldFactionData](entity, OverworldFactionComponent)
}

// CountFactions returns number of active factions
func CountFactions(manager *common.EntityManager) int {
	count := 0
	for range manager.World.Query(OverworldFactionTag) {
		count++
	}
	return count
}

// GetStrongestFaction returns the faction with highest strength
func GetStrongestFaction(manager *common.EntityManager) (ecs.EntityID, *OverworldFactionData) {
	var strongestID ecs.EntityID
	var strongestData *OverworldFactionData
	maxStrength := -1

	for _, result := range manager.World.Query(OverworldFactionTag) {
		factionData := common.GetComponentType[*OverworldFactionData](result.Entity, OverworldFactionComponent)
		if factionData != nil && factionData.Strength > maxStrength {
			maxStrength = factionData.Strength
			strongestID = result.Entity.GetID()
			strongestData = factionData
		}
	}

	return strongestID, strongestData
}

// GetWeakestFaction returns the faction with lowest strength
func GetWeakestFaction(manager *common.EntityManager) (ecs.EntityID, *OverworldFactionData) {
	var weakestID ecs.EntityID
	var weakestData *OverworldFactionData
	minStrength := 9999

	for _, result := range manager.World.Query(OverworldFactionTag) {
		factionData := common.GetComponentType[*OverworldFactionData](result.Entity, OverworldFactionComponent)
		if factionData != nil && factionData.Strength < minStrength {
			minStrength = factionData.Strength
			weakestID = result.Entity.GetID()
			weakestData = factionData
		}
	}

	return weakestID, weakestData
}

// GetFactionsByType returns all factions of a specific type
func GetFactionsByType(manager *common.EntityManager, factionType FactionType) []ecs.EntityID {
	var factions []ecs.EntityID

	for _, result := range manager.World.Query(OverworldFactionTag) {
		factionData := common.GetComponentType[*OverworldFactionData](result.Entity, OverworldFactionComponent)
		if factionData != nil && factionData.FactionType == factionType {
			factions = append(factions, result.Entity.GetID())
		}
	}

	return factions
}

// GetTotalTerritorySize returns sum of all faction territories
func GetTotalTerritorySize(manager *common.EntityManager) int {
	total := 0

	for _, result := range manager.World.Query(OverworldFactionTag) {
		factionData := common.GetComponentType[*OverworldFactionData](result.Entity, OverworldFactionComponent)
		if factionData != nil {
			total += factionData.TerritorySize
		}
	}

	return total
}

// GetAverageFactioStrength calculates average strength across all factions
func GetAverageFactionStrength(manager *common.EntityManager) float64 {
	count := 0
	totalStrength := 0

	for _, result := range manager.World.Query(OverworldFactionTag) {
		factionData := common.GetComponentType[*OverworldFactionData](result.Entity, OverworldFactionComponent)
		if factionData != nil {
			totalStrength += factionData.Strength
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return float64(totalStrength) / float64(count)
}

// GetFactionsByIntent returns factions with a specific intent
func GetFactionsByIntent(manager *common.EntityManager, intent FactionIntent) []ecs.EntityID {
	var factions []ecs.EntityID

	for _, result := range manager.World.Query(OverworldFactionTag) {
		intentData := common.GetComponentType[*StrategicIntentData](result.Entity, StrategicIntentComponent)
		if intentData != nil && intentData.Intent == intent {
			factions = append(factions, result.Entity.GetID())
		}
	}

	return factions
}

// CountFactionsByIntent returns number of factions with specific intent
func CountFactionsByIntent(manager *common.EntityManager, intent FactionIntent) int {
	count := 0

	for _, result := range manager.World.Query(OverworldFactionTag) {
		intentData := common.GetComponentType[*StrategicIntentData](result.Entity, StrategicIntentComponent)
		if intentData != nil && intentData.Intent == intent {
			count++
		}
	}

	return count
}
