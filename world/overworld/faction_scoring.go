package overworld

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// ScoreExpansion evaluates how good expansion is right now
func ScoreExpansion(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
	score := 0.0

	// Favor expansion when strong
	if factionData.Strength >= ExpansionStrengthThreshold {
		score += 5.0
	}

	// Favor expansion when territory is small
	if factionData.TerritorySize < ExpansionTerritoryLimit {
		score += 3.0
	}

	// Penalize if at territory limit
	if factionData.TerritorySize >= MaxTerritorySize {
		score -= 10.0
	}

	// Faction type modifiers
	switch factionData.FactionType {
	case FactionCultists:
		score += 3.0 // Cultists love to expand
	case FactionOrcs:
		score += 2.0 // Orcs are territorial
	case FactionBeasts:
		score -= 1.0 // Beasts prefer smaller territories
	}

	return score
}

// ScoreFortification evaluates defensive posture
func ScoreFortification(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
	score := 0.0

	// Favor fortify when weak
	if factionData.Strength < FortificationWeakThreshold {
		score += 6.0
	}

	// Always some value to fortifying
	score += 2.0

	// Faction type modifiers
	switch factionData.FactionType {
	case FactionNecromancers:
		score += 2.0 // Necromancers prefer defense
	}

	return score
}

// ScoreRaiding evaluates attacking player/other factions
func ScoreRaiding(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
	score := 0.0

	// Need minimum strength to raid
	if factionData.Strength < RaidStrengthThreshold {
		return 0.0
	}

	// Aggressive faction types raid more
	switch factionData.FactionType {
	case FactionBandits:
		score += 5.0
	case FactionOrcs:
		score += 4.0
	}

	// Raid if strong
	if factionData.Strength > 10 {
		score += 3.0
	}

	return score
}

// ScoreRetreat evaluates abandoning territory
func ScoreRetreat(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
	score := 0.0

	// Only retreat if critically weak
	if factionData.Strength < RetreatCriticalStrength {
		score += 8.0
	}

	// Don't retreat if territory is small
	if factionData.TerritorySize <= 1 {
		score -= 5.0
	}

	return score
}
