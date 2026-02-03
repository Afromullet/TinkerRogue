package faction

import (
	"game_main/common"
	"game_main/overworld/core"

	"github.com/bytearena/ecs"
)

// ScoreExpansion evaluates how good expansion is right now.
// Uses unified strength thresholds and faction archetype system.
func ScoreExpansion(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	strongBonus, smallTerritoryBonus, maxTerritoryPenalty := core.GetExpansionScoringParams()

	// Favor expansion when strong (use unified threshold)
	if factionData.Strength >= core.GetStrongThreshold() {
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

	// Apply faction archetype bonus
	score += core.GetFactionBonuses(factionData.FactionType).ExpansionBonus

	// Apply aggression modifier: high aggression = more territorial (70-100%)
	aggression := core.GetFactionAggression(factionData.FactionType)
	score *= (0.7 + aggression*0.3)

	return score
}

// ScoreFortification evaluates defensive posture.
// Uses unified strength thresholds and faction archetype system.
func ScoreFortification(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	weakBonus, baseValue := core.GetFortificationScoringParams()

	// Favor fortify when weak (use unified threshold)
	if factionData.Strength < core.GetWeakThreshold() {
		score += weakBonus
	}

	// Always some value to fortifying
	score += baseValue

	// Apply faction archetype bonus
	score += core.GetFactionBonuses(factionData.FactionType).FortificationBonus

	// Apply aggression modifier: low aggression = more defensive (100-130%)
	aggression := core.GetFactionAggression(factionData.FactionType)
	score *= (1.3 - aggression*0.3)

	return score
}

// ScoreRaiding evaluates attacking player/other factions.
// Uses unified strength thresholds and faction archetype system.
func ScoreRaiding(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	strongBonus := core.GetRaidingScoringParams()

	// Need minimum strength to raid (use unified strong threshold)
	if factionData.Strength < core.GetStrongThreshold() {
		return 0.0
	}

	// Apply faction archetype bonus
	bonuses := core.GetFactionBonuses(factionData.FactionType)
	score += bonuses.RaidingBonus

	// Apply aggression modifier to raiding intensity
	aggression := core.GetFactionAggression(factionData.FactionType)
	score *= (0.5 + aggression*0.5) // Scale raiding by aggression (0.5-1.0x)

	// Raid if very strong (use unified strong threshold + offset)
	if factionData.Strength > core.GetStrongThreshold()+3 {
		score += strongBonus
	}

	return score
}

// ScoreRetreat evaluates abandoning territory.
// Uses unified strength thresholds and faction archetype system.
func ScoreRetreat(manager *common.EntityManager, entity *ecs.Entity, factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	criticalWeakBonus, smallTerritoryPenalty, minTerritorySize := core.GetRetreatScoringParams()

	// Only retreat if critically weak (use unified threshold)
	if factionData.Strength < core.GetCriticalThreshold() {
		score += criticalWeakBonus
	}

	// Don't retreat if territory is small
	if factionData.TerritorySize <= minTerritorySize {
		score += smallTerritoryPenalty
	}

	// Apply faction archetype penalty
	score -= core.GetFactionBonuses(factionData.FactionType).RetreatPenalty

	// Apply aggression modifier: high aggression = less likely to retreat (80-120%)
	aggression := core.GetFactionAggression(factionData.FactionType)
	score *= (1.2 - aggression*0.4)

	return score
}
