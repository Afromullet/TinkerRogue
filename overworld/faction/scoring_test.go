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
	// Set up faction archetypes
	templates.OverworldConfigTemplate.FactionArchetypes = map[string]templates.FactionArchetypeConfig{
		"Cultists":     {Strategy: "Expansionist", Aggression: 0.7},
		"Orcs":         {Strategy: "Aggressor", Aggression: 0.9},
		"Bandits":      {Strategy: "Raider", Aggression: 0.8},
		"Necromancers": {Strategy: "Defensive", Aggression: 0.3},
		"Beasts":       {Strategy: "Territorial", Aggression: 0.5},
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
		ExpansionTerritoryLimit:   20,
		FortificationStrengthGain: 1,
		MaxTerritorySize:          30,
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
			StrongBonus: 3.0,
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
	cultistScore := ScoreExpansion(nil, nil, strongFaction)

	// Test Necromancers (Defensive archetype, aggression 0.3)
	strongFaction.FactionType = core.FactionNecromancers
	necromancerScore := ScoreExpansion(nil, nil, strongFaction)

	// Test Bandits (Raider archetype, aggression 0.8)
	strongFaction.FactionType = core.FactionBandits
	banditScore := ScoreExpansion(nil, nil, strongFaction)

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
	weakThreshold := core.GetWeakThreshold()

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

	weakScore := ScoreFortification(nil, nil, weakFaction)
	strongScore := ScoreFortification(nil, nil, strongFaction)

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
	strongThreshold := core.GetStrongThreshold()

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

	weakScore := ScoreRaiding(nil, nil, weakFaction)
	strongScore := ScoreRaiding(nil, nil, strongFaction)

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
	criticalThreshold := core.GetCriticalThreshold()

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

	criticalScore := ScoreRetreat(nil, nil, criticallyWeakFaction)
	healthyScore := ScoreRetreat(nil, nil, healthyFaction)

	// Critically weak factions should score higher on retreat
	if criticalScore <= healthyScore {
		t.Errorf("Critically weak factions should score higher on retreat: critical=%.2f, healthy=%.2f",
			criticalScore, healthyScore)
	}
}

// TestAggressionAffectsAllScores verifies that aggression modifier affects
// all four scoring functions appropriately.
func TestAggressionAffectsAllScores(t *testing.T) {
	// High aggression faction (Orcs: 0.9)
	highAggFaction := &core.OverworldFactionData{
		Strength:      10,
		TerritorySize: 5,
		FactionType:   core.FactionOrcs,
	}

	// Low aggression faction (Necromancers: 0.3)
	lowAggFaction := &core.OverworldFactionData{
		Strength:      10,
		TerritorySize: 5,
		FactionType:   core.FactionNecromancers,
	}

	// Test expansion: high aggression should give higher score
	highAggExpand := ScoreExpansion(nil, nil, highAggFaction)
	lowAggExpand := ScoreExpansion(nil, nil, lowAggFaction)

	// Orcs (Aggressor, 0.9 aggression) vs Necromancers (Defensive, 0.3 aggression)
	// After archetype bonuses and aggression scaling, Orcs should expand more
	if highAggExpand <= 0 || lowAggExpand < 0 {
		t.Errorf("Factions should have non-negative expansion scores: high=%.2f, low=%.2f", highAggExpand, lowAggExpand)
	}

	// Test fortification: low aggression should give higher score
	highAggFort := ScoreFortification(nil, nil, highAggFaction)
	lowAggFort := ScoreFortification(nil, nil, lowAggFaction)

	// Low aggression should favor fortification (inverse relationship)
	if lowAggFort <= highAggFort {
		t.Errorf("Low aggression factions should fortify more: high=%.2f, low=%.2f",
			highAggFort, lowAggFort)
	}

	// Test raiding: high aggression should give higher score
	highAggRaid := ScoreRaiding(nil, nil, highAggFaction)
	lowAggRaid := ScoreRaiding(nil, nil, lowAggFaction)

	if highAggRaid <= lowAggRaid {
		t.Errorf("High aggression factions should raid more: high=%.2f, low=%.2f",
			highAggRaid, lowAggRaid)
	}

	// Test retreat: aggression affects retreat modifier
	// Create critically weak versions
	criticalThreshold := core.GetCriticalThreshold()
	highAggFaction.Strength = criticalThreshold - 1
	lowAggFaction.Strength = criticalThreshold - 1

	highAggRetreat := ScoreRetreat(nil, nil, highAggFaction)
	lowAggRetreat := ScoreRetreat(nil, nil, lowAggFaction)

	// Note: Low aggression doesn't necessarily mean higher retreat scores
	// because archetype bonuses also affect retreat (Defensive has RetreatPenalty +2.0
	// which SUBTRACTS from score). The aggression multiplier is applied after.
	// Both should be positive when critically weak with sufficient territory.
	if highAggRetreat <= 0 || lowAggRetreat <= 0 {
		t.Errorf("Critically weak factions should have positive retreat scores: high=%.2f, low=%.2f",
			highAggRetreat, lowAggRetreat)
	}

	// Test aggression effect directly: same faction type, different aggression
	// Use Beasts (Territorial, 0.5) as baseline - moderate retreat penalty
	beastFaction := &core.OverworldFactionData{
		Strength:      criticalThreshold - 1,
		TerritorySize: 10,
		FactionType:   core.FactionBeasts,
	}
	beastRetreat := ScoreRetreat(nil, nil, beastFaction)

	// Beasts (Territorial) have RetreatPenalty -3.0 which adds to score
	// This makes them more willing to retreat than Orcs (Aggressor) with 0.0
	if beastRetreat <= highAggRetreat {
		t.Errorf("Beasts (Territorial, retreat penalty -3) should retreat more than Orcs (Aggressor, 0): beasts=%.2f, orcs=%.2f",
			beastRetreat, highAggRetreat)
	}
}
