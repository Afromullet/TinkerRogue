package faction

import (
	"game_main/overworld/core"
	"game_main/templates"
)

// FactionArchetype defines strategic archetype and aggression level
type FactionArchetype struct {
	Strategy   string
	Aggression float64
}

// FactionBonuses defines behavior bonuses derived from archetype
type FactionBonuses struct {
	ExpansionBonus     float64
	FortificationBonus float64
	RaidingBonus       float64
	RetreatPenalty     float64
}

// strategyBonuses maps archetype strategies to behavior bonuses
var strategyBonuses = map[string]FactionBonuses{
	"Expansionist": {ExpansionBonus: 3.0, FortificationBonus: 0.0, RaidingBonus: 1.0, RetreatPenalty: 0.0},
	"Aggressor":    {ExpansionBonus: 2.0, FortificationBonus: 0.0, RaidingBonus: 4.0, RetreatPenalty: 0.0},
	"Raider":       {ExpansionBonus: 0.0, FortificationBonus: 0.0, RaidingBonus: 5.0, RetreatPenalty: -2.0},
	"Defensive":    {ExpansionBonus: 0.0, FortificationBonus: 2.0, RaidingBonus: 0.0, RetreatPenalty: 2.0},
	"Territorial":  {ExpansionBonus: -1.0, FortificationBonus: 1.0, RaidingBonus: 0.0, RetreatPenalty: -3.0},
}

// GetFactionArchetype returns archetype config for a faction type.
func GetFactionArchetype(factionType core.FactionType) FactionArchetype {
	factionName := factionType.String()
	if a, ok := templates.FactionArchetypeTemplates[factionName]; ok {
		return FactionArchetype{
			Strategy:   a.Strategy,
			Aggression: a.Aggression,
		}
	}
	// Default: neutral archetype
	return FactionArchetype{Strategy: "Defensive", Aggression: 0.5}
}

// GetFactionBonuses returns behavior bonuses for a faction type based on its archetype.
func GetFactionBonuses(factionType core.FactionType) FactionBonuses {
	archetype := GetFactionArchetype(factionType)
	if bonuses, ok := strategyBonuses[archetype.Strategy]; ok {
		return bonuses
	}
	return FactionBonuses{}
}

// GetFactionAggression returns the aggression level for a faction type.
func GetFactionAggression(factionType core.FactionType) float64 {
	return GetFactionArchetype(factionType).Aggression
}
