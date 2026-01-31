package overworld

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// ScoreExpansion evaluates how good expansion is right now
func ScoreExpansion(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	strongBonus, smallTerritoryBonus, maxTerritoryPenalty,
		cultistMod, orcMod, beastMod := GetExpansionScoringParams()

	// Favor expansion when strong
	if factionData.Strength >= GetExpansionStrengthThreshold() {
		score += strongBonus
	}

	// Favor expansion when territory is small
	if factionData.TerritorySize < GetExpansionTerritoryLimit() {
		score += smallTerritoryBonus
	}

	// Penalize if at territory limit
	if factionData.TerritorySize >= GetMaxTerritorySize() {
		score += maxTerritoryPenalty
	}

	// Faction type modifiers
	switch factionData.FactionType {
	case FactionCultists:
		score += cultistMod // Cultists love to expand
	case FactionOrcs:
		score += orcMod // Orcs are territorial
	case FactionBeasts:
		score += beastMod // Beasts prefer smaller territories
	}

	return score
}

// ScoreFortification evaluates defensive posture
func ScoreFortification(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	weakBonus, baseValue, necromancerMod := GetFortificationScoringParams()

	// Favor fortify when weak
	if factionData.Strength < GetFortificationWeakThreshold() {
		score += weakBonus
	}

	// Always some value to fortifying
	score += baseValue

	// Faction type modifiers
	switch factionData.FactionType {
	case FactionNecromancers:
		score += necromancerMod // Necromancers prefer defense
	}

	return score
}

// ScoreRaiding evaluates attacking player/other factions
func ScoreRaiding(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	banditMod, orcMod, strongBonus, strongThreshold := GetRaidingScoringParams()

	// Need minimum strength to raid
	if factionData.Strength < GetRaidStrengthThreshold() {
		return 0.0
	}

	// Aggressive faction types raid more
	switch factionData.FactionType {
	case FactionBandits:
		score += banditMod
	case FactionOrcs:
		score += orcMod
	}

	// Raid if strong
	if factionData.Strength > strongThreshold {
		score += strongBonus
	}

	return score
}

// ScoreRetreat evaluates abandoning territory
func ScoreRetreat(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	criticalWeakBonus, smallTerritoryPenalty, minTerritorySize := GetRetreatScoringParams()

	// Only retreat if critically weak
	if factionData.Strength < GetRetreatCriticalStrength() {
		score += criticalWeakBonus
	}

	// Don't retreat if territory is small
	if factionData.TerritorySize <= minTerritorySize {
		score += smallTerritoryPenalty
	}

	return score
}
