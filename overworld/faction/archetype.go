package faction

import (
	"game_main/overworld/core"
	"game_main/templates"
)

// FactionArchetype defines strategic archetype for a faction
type FactionArchetype struct {
	Strategy string
}

// FactionBonuses defines behavior bonuses derived from archetype
type FactionBonuses struct {
	ExpansionBonus     float64
	FortificationBonus float64
	RaidingBonus       float64
	RetreatPenalty     float64
}

// GetFactionArchetype returns archetype config for a faction type.
func GetFactionArchetype(factionType core.FactionType) FactionArchetype {
	factionName := factionType.String()
	if a, ok := templates.FactionArchetypeTemplates[factionName]; ok {
		return FactionArchetype{
			Strategy: a.Strategy,
		}
	}
	// Default: neutral archetype
	return FactionArchetype{Strategy: "Defensive"}
}

// GetFactionBonuses returns behavior bonuses for a faction type based on its archetype.
func GetFactionBonuses(factionType core.FactionType) FactionBonuses {
	archetype := GetFactionArchetype(factionType)
	bonuses := templates.OverworldConfigTemplate.StrategyBonuses
	if cfg, ok := bonuses[archetype.Strategy]; ok {
		return FactionBonuses{
			ExpansionBonus:     cfg.ExpansionBonus,
			FortificationBonus: cfg.FortificationBonus,
			RaidingBonus:       cfg.RaidingBonus,
			RetreatPenalty:     cfg.RetreatPenalty,
		}
	}
	return FactionBonuses{}
}

