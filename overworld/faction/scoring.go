package faction

import (
	"game_main/overworld/core"
	"game_main/templates"
)

// ScoreExpansion evaluates how good expansion is right now.
// Uses unified strength thresholds and faction archetype system.
func ScoreExpansion(factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	strongBonus, smallTerritoryBonus, maxTerritoryPenalty := core.GetExpansionScoringParams()

	// Favor expansion when strong (use unified threshold)
	if factionData.Strength >= templates.OverworldConfigTemplate.StrengthThresholds.Strong {
		score += strongBonus
	}

	// Favor expansion when territory is small (below 2/3 of max)
	if factionData.TerritorySize < templates.OverworldConfigTemplate.FactionAI.MaxTerritorySize*2/3 {
		score += smallTerritoryBonus
	}

	// Penalize if at territory limit
	if factionData.TerritorySize >= templates.OverworldConfigTemplate.FactionAI.MaxTerritorySize {
		score += maxTerritoryPenalty
	}

	// Apply faction archetype bonus
	score += GetFactionBonuses(factionData.FactionType).ExpansionBonus

	return score
}

// ScoreFortification evaluates defensive posture.
// Uses unified strength thresholds and faction archetype system.
func ScoreFortification(factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	weakBonus, baseValue := core.GetFortificationScoringParams()

	// Favor fortify when weak (use unified threshold)
	if factionData.Strength < templates.OverworldConfigTemplate.StrengthThresholds.Weak {
		score += weakBonus
	}

	// Always some value to fortifying
	score += baseValue

	// Apply faction archetype bonus
	score += GetFactionBonuses(factionData.FactionType).FortificationBonus

	return score
}

// ScoreRaiding evaluates attacking player/other factions.
// Uses unified strength thresholds and faction archetype system.
func ScoreRaiding(factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	strongBonus, veryStrongOffset := core.GetRaidingScoringParams()

	// Need minimum strength to raid (use unified strong threshold)
	if factionData.Strength < templates.OverworldConfigTemplate.StrengthThresholds.Strong {
		return 0.0
	}

	// Apply faction archetype bonus
	score += GetFactionBonuses(factionData.FactionType).RaidingBonus

	// Raid if very strong (use unified strong threshold + offset)
	if factionData.Strength > templates.OverworldConfigTemplate.StrengthThresholds.Strong+veryStrongOffset {
		score += strongBonus
	}

	return score
}

// ScoreRetreat evaluates abandoning territory.
// Uses unified strength thresholds and faction archetype system.
func ScoreRetreat(factionData *core.OverworldFactionData) float64 {
	score := 0.0

	// Get config parameters
	criticalWeakBonus, smallTerritoryPenalty, minTerritorySize := core.GetRetreatScoringParams()

	// Only retreat if critically weak (use unified threshold)
	if factionData.Strength < templates.OverworldConfigTemplate.StrengthThresholds.Critical {
		score += criticalWeakBonus
	}

	// Don't retreat if territory is small
	if factionData.TerritorySize <= minTerritorySize {
		score += smallTerritoryPenalty
	}

	// Apply faction archetype penalty
	score -= GetFactionBonuses(factionData.FactionType).RetreatPenalty

	return score
}
