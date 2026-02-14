package faction

import (
	"game_main/overworld/core"
	"game_main/templates"
	"testing"
)

// init sets up the overworld config template for tests.
// We populate the template directly rather than loading from JSON
// since tests run from different directories.
func init() {
	// Initialize difficulty manager for tests (Medium defaults)
	templates.GlobalDifficulty = templates.NewDefaultDifficultyManager()

	// Set up faction archetypes (now stored in FactionArchetypeTemplates)
	templates.FactionArchetypeTemplates = map[string]templates.FactionArchetypeConfig{
		"Cultists":     {Strategy: "Expansionist"},
		"Orcs":         {Strategy: "Aggressor"},
		"Bandits":      {Strategy: "Raider"},
		"Necromancers": {Strategy: "Defensive"},
		"Beasts":       {Strategy: "Territorial"},
	}

	// Set up strength thresholds
	templates.OverworldConfigTemplate.StrengthThresholds = templates.StrengthThresholdsConfig{
		Weak:     3,
		Strong:   7,
		Critical: 2,
	}

	// Set up faction AI config
	templates.OverworldConfigTemplate.FactionAI = templates.FactionAIConfig{
		DefaultIntentTickDuration: 10,
		MaxTerritorySize:          30,
	}

	// Set up strategy bonuses (previously hardcoded in archetype.go)
	templates.OverworldConfigTemplate.StrategyBonuses = map[string]templates.StrategyBonusConfig{
		"Expansionist": {ExpansionBonus: 3.0, FortificationBonus: 0.0, RaidingBonus: 1.0, RetreatPenalty: 0.0},
		"Aggressor":    {ExpansionBonus: 2.0, FortificationBonus: 0.0, RaidingBonus: 4.0, RetreatPenalty: 0.0},
		"Raider":       {ExpansionBonus: 0.0, FortificationBonus: 0.0, RaidingBonus: 5.0, RetreatPenalty: -2.0},
		"Defensive":    {ExpansionBonus: 0.0, FortificationBonus: 2.0, RaidingBonus: 0.0, RetreatPenalty: 2.0},
		"Territorial":  {ExpansionBonus: -1.0, FortificationBonus: 1.0, RaidingBonus: 0.0, RetreatPenalty: -3.0},
	}

	// Set up faction scoring config
	templates.OverworldConfigTemplate.FactionScoring = templates.FactionScoringConfig{
		Expansion: templates.ExpansionScoringConfig{
			StrongBonus:         5.0,
			SmallTerritoryBonus: 3.0,
			MaxTerritoryPenalty: -10.0,
		},
		Fortification: templates.FortificationScoringConfig{
			WeakBonus: 6.0,
			BaseValue: 2.0,
		},
		Raiding: templates.RaidingScoringConfig{
			StrongBonus:      3.0,
			VeryStrongOffset: 3,
		},
		Retreat: templates.RetreatScoringConfig{
			CriticalWeakBonus:     8.0,
			SmallTerritoryPenalty: -5.0,
			MinTerritorySize:      1,
		},
	}
}

// TestScoreExpansion_FactionDifferentiation verifies that different factions
// get different expansion scores based on their archetypes.
func TestScoreExpansion_FactionDifferentiation(t *testing.T) {
	// Create strong faction data (above strong threshold)
	strongFaction := &core.OverworldFactionData{
		Strength:      10, // Above strong threshold (7)
		TerritorySize: 5,  // Below expansion limit (20)
	}

	// Test Cultists (Expansionist archetype, aggression 0.7)
	strongFaction.FactionType = core.FactionCultists
	cultistScore := ScoreExpansion(strongFaction)

	// Test Necromancers (Defensive archetype, aggression 0.3)
	strongFaction.FactionType = core.FactionNecromancers
	necromancerScore := ScoreExpansion(strongFaction)

	// Test Bandits (Raider archetype, aggression 0.8)
	strongFaction.FactionType = core.FactionBandits
	banditScore := ScoreExpansion(strongFaction)

	// Expansionist Cultists should score higher than Defensive Necromancers
	if cultistScore <= necromancerScore {
		t.Errorf("Cultists (Expansionist) should score higher on expansion than Necromancers (Defensive): cultists=%.2f, necromancers=%.2f",
			cultistScore, necromancerScore)
	}

	// All scores should be positive for strong factions with small territory
	if cultistScore <= 0 || necromancerScore <= 0 || banditScore <= 0 {
		t.Errorf("All factions should have positive expansion scores when strong: cultists=%.2f, necromancers=%.2f, bandits=%.2f",
			cultistScore, necromancerScore, banditScore)
	}
}

// TestScoreFortification_WeakFactionsFortifyMore verifies that weak factions
// get higher fortification scores.
func TestScoreFortification_WeakFactionsFortifyMore(t *testing.T) {
	weakThreshold := templates.OverworldConfigTemplate.StrengthThresholds.Weak

	// Create weak faction (below weak threshold)
	weakFaction := &core.OverworldFactionData{
		Strength:      weakThreshold - 1,
		TerritorySize: 10,
		FactionType:   core.FactionNecromancers, // Defensive archetype
	}

	// Create strong faction (above weak threshold)
	strongFaction := &core.OverworldFactionData{
		Strength:      weakThreshold + 5,
		TerritorySize: 10,
		FactionType:   core.FactionNecromancers,
	}

	weakScore := ScoreFortification(weakFaction)
	strongScore := ScoreFortification(strongFaction)

	// Weak factions should score higher on fortification
	if weakScore <= strongScore {
		t.Errorf("Weak factions should score higher on fortification: weak=%.2f, strong=%.2f",
			weakScore, strongScore)
	}

	// Necromancers (Defensive) should have positive fortification score
	if weakScore <= 0 {
		t.Errorf("Defensive faction should have positive fortification score: %.2f", weakScore)
	}
}

// TestScoreRaiding_RequiresStrength verifies that raiding requires minimum strength.
func TestScoreRaiding_RequiresStrength(t *testing.T) {
	strongThreshold := templates.OverworldConfigTemplate.StrengthThresholds.Strong

	// Create weak faction (below strong threshold)
	weakFaction := &core.OverworldFactionData{
		Strength:      strongThreshold - 1,
		TerritorySize: 10,
		FactionType:   core.FactionBandits, // Raider archetype
	}

	// Create strong faction (above strong threshold)
	strongFaction := &core.OverworldFactionData{
		Strength:      strongThreshold + 5,
		TerritorySize: 10,
		FactionType:   core.FactionBandits,
	}

	weakScore := ScoreRaiding(weakFaction)
	strongScore := ScoreRaiding(strongFaction)

	// Weak factions should not be able to raid
	if weakScore != 0.0 {
		t.Errorf("Weak factions should have zero raiding score: got %.2f", weakScore)
	}

	// Strong Bandits (Raider archetype) should have positive raiding score
	if strongScore <= 0 {
		t.Errorf("Strong Bandits should have positive raiding score: got %.2f", strongScore)
	}
}

// TestScoreRetreat_CriticallyWeakRetreats verifies that critically weak factions
// get high retreat scores.
func TestScoreRetreat_CriticallyWeakRetreats(t *testing.T) {
	criticalThreshold := templates.OverworldConfigTemplate.StrengthThresholds.Critical

	// Create critically weak faction
	criticallyWeakFaction := &core.OverworldFactionData{
		Strength:      criticalThreshold - 1,
		TerritorySize: 10, // Not at minimum territory
		FactionType:   core.FactionBeasts,
	}

	// Create healthy faction
	healthyFaction := &core.OverworldFactionData{
		Strength:      criticalThreshold + 5,
		TerritorySize: 10,
		FactionType:   core.FactionBeasts,
	}

	criticalScore := ScoreRetreat(criticallyWeakFaction)
	healthyScore := ScoreRetreat(healthyFaction)

	// Critically weak factions should score higher on retreat
	if criticalScore <= healthyScore {
		t.Errorf("Critically weak factions should score higher on retreat: critical=%.2f, healthy=%.2f",
			criticalScore, healthyScore)
	}
}

// TestArchetypeDifferentiatesAllScores verifies that different archetypes
// produce different scoring outcomes across all four intent types.
func TestArchetypeDifferentiatesAllScores(t *testing.T) {
	// Orcs (Aggressor): raiding bonus +4.0, expansion bonus +2.0
	orcFaction := &core.OverworldFactionData{
		Strength:      10,
		TerritorySize: 5,
		FactionType:   core.FactionOrcs,
	}

	// Necromancers (Defensive): fortification bonus +2.0, retreat penalty +2.0
	necroFaction := &core.OverworldFactionData{
		Strength:      10,
		TerritorySize: 5,
		FactionType:   core.FactionNecromancers,
	}

	// Aggressor should raid more than Defensive
	orcRaid := ScoreRaiding(orcFaction)
	necroRaid := ScoreRaiding(necroFaction)
	if orcRaid <= necroRaid {
		t.Errorf("Orcs (Aggressor) should raid more than Necromancers (Defensive): orcs=%.2f, necro=%.2f",
			orcRaid, necroRaid)
	}

	// Defensive should fortify more than Aggressor
	orcFort := ScoreFortification(orcFaction)
	necroFort := ScoreFortification(necroFaction)
	if necroFort <= orcFort {
		t.Errorf("Necromancers (Defensive) should fortify more than Orcs (Aggressor): necro=%.2f, orcs=%.2f",
			necroFort, orcFort)
	}

	// Beasts (Territorial, retreat penalty -3.0) should retreat more than Orcs (retreat penalty 0.0)
	criticalThreshold := templates.OverworldConfigTemplate.StrengthThresholds.Critical
	orcFaction.Strength = criticalThreshold - 1
	beastFaction := &core.OverworldFactionData{
		Strength:      criticalThreshold - 1,
		TerritorySize: 10,
		FactionType:   core.FactionBeasts,
	}
	beastRetreat := ScoreRetreat(beastFaction)
	orcRetreat := ScoreRetreat(orcFaction)
	if beastRetreat <= orcRetreat {
		t.Errorf("Beasts (Territorial, retreat penalty -3) should retreat more than Orcs (Aggressor, 0): beasts=%.2f, orcs=%.2f",
			beastRetreat, orcRetreat)
	}
}
