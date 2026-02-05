package encounter

import (
	"game_main/overworld/core"
	"game_main/templates"
)

// GetDifficultyModifier retrieves difficulty settings for a given encounter level.
// Falls back to level 3 (fair fight) if level is invalid.
func GetDifficultyModifier(level int) EncounterDifficultyModifier {
	// Search for matching difficulty level in templates
	for _, template := range templates.EncounterDifficultyTemplates {
		if template.Level == level {
			return EncounterDifficultyModifier{
				PowerMultiplier:  template.PowerMultiplier,
				SquadCount:       template.SquadCount,
				MinUnitsPerSquad: template.MinUnitsPerSquad,
				MaxUnitsPerSquad: template.MaxUnitsPerSquad,
				MinTargetPower:   template.MinTargetPower,
				MaxTargetPower:   template.MaxTargetPower,
			}
		}
	}

	// Fallback to level 3 (fair fight) if not found
	for _, template := range templates.EncounterDifficultyTemplates {
		if template.Level == 3 {
			return EncounterDifficultyModifier{
				PowerMultiplier:  template.PowerMultiplier,
				SquadCount:       template.SquadCount,
				MinUnitsPerSquad: template.MinUnitsPerSquad,
				MaxUnitsPerSquad: template.MaxUnitsPerSquad,
				MinTargetPower:   template.MinTargetPower,
				MaxTargetPower:   template.MaxTargetPower,
			}
		}
	}

	// Last resort fallback (should never happen if JSON loads correctly)
	return EncounterDifficultyModifier{
		PowerMultiplier:  1.0,
		SquadCount:       4,
		MinUnitsPerSquad: 3,
		MaxUnitsPerSquad: 5,
		MinTargetPower:   50.0,
		MaxTargetPower:   2000.0,
	}
}

// GetSquadPreferences retrieves preferred squad composition for an encounter type.
// Returns nil if encounter type not found (allows random composition fallback).
func GetSquadPreferences(encounterTypeID string) []string {
	def := core.GetThreatRegistry().GetByEncounterTypeID(encounterTypeID)
	if def.ID != "default" {
		// Return copy to prevent external modification
		prefs := make([]string, len(def.SquadPreferences))
		copy(prefs, def.SquadPreferences)
		return prefs
	}
	return nil // Not found - allows caller to handle random composition
}
