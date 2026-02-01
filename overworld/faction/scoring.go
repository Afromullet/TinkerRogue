package faction

import (
	"game_main/common"
	"game_main/overworld/core"

	"github.com/bytearena/ecs"
)

// ScoreExpansion evaluates how good expansion is right now
func ScoreExpansion(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	strongBonus, smallTerritoryBonus, maxTerritoryPenalty,
		cultistMod, orcMod, beastMod := core.GetExpansionScoringParams()

	// Favor expansion when strong
	if factionData.Strength >= core.GetExpansionStrengthThreshold() {
		score += strongBonus
	}

	// Favor expansion when territory is small
	if factionData.TerritorySize < core.GetExpansionTerritoryLimit() {
		score += smallTerritoryBonus
	}

	// Penalize if at territory limit
	if factionData.TerritorySize >= core.GetMaxTerritorySize() {
		score += maxTerritoryPenalty
	}

	// Faction type modifiers
	switch factionData.FactionType {
	case core.FactionCultists:
		score += cultistMod // Cultists love to expand
	case core.FactionOrcs:
		score += orcMod // Orcs are territorial
	case core.FactionBeasts:
		score += beastMod // Beasts prefer smaller territories
	}

	return score
}

// ScoreFortification evaluates defensive posture
func ScoreFortification(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	weakBonus, baseValue, necromancerMod := core.GetFortificationScoringParams()

	// Favor fortify when weak
	if factionData.Strength < core.GetFortificationWeakThreshold() {
		score += weakBonus
	}

	// Always some value to fortifying
	score += baseValue

	// Faction type modifiers
	switch factionData.FactionType {
	case core.FactionNecromancers:
		score += necromancerMod // Necromancers prefer defense
	}

	return score
}

// ScoreRaiding evaluates attacking player/other factions
func ScoreRaiding(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	banditMod, orcMod, strongBonus, strongThreshold := core.GetRaidingScoringParams()

	// Need minimum strength to raid
	if factionData.Strength < core.GetRaidStrengthThreshold() {
		return 0.0
	}

	// Aggressive faction types raid more
	switch factionData.FactionType {
	case core.FactionBandits:
		score += banditMod
	case core.FactionOrcs:
		score += orcMod
	}

	// Raid if strong
	if factionData.Strength > strongThreshold {
		score += strongBonus
	}

	return score
}

// ScoreRetreat evaluates abandoning territory
func ScoreRetreat(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	criticalWeakBonus, smallTerritoryPenalty, minTerritorySize := core.GetRetreatScoringParams()

	// Only retreat if critically weak
	if factionData.Strength < core.GetRetreatCriticalStrength() {
		score += criticalWeakBonus
	}

	// Don't retreat if territory is small
	if factionData.TerritorySize <= minTerritorySize {
		score += smallTerritoryPenalty
	}

	return score
}
