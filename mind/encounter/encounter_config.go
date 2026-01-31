package encounter

import (
	"game_main/templates"
)

// GetDifficultyModifier retrieves difficulty settings for a given encounter level.
// Falls back to level 3 (fair fight) if level is invalid.
func GetDifficultyModifier(level int) EncounterDifficultyModifier {
	// Search for matching difficulty level in templates
	for _, template := range templates.EncounterDifficultyTemplates {
		if template.Level == level {
			return EncounterDifficultyModifier{
				PowerMultiplier: template.PowerMultiplier,
				MinSquads:       template.MinSquads,
				MaxSquads:       template.MaxSquads,
			}
		}
	}

	// Fallback to level 3 (fair fight) if not found
	for _, template := range templates.EncounterDifficultyTemplates {
		if template.Level == 3 {
			return EncounterDifficultyModifier{
				PowerMultiplier: template.PowerMultiplier,
				MinSquads:       template.MinSquads,
				MaxSquads:       template.MaxSquads,
			}
		}
	}

	// Last resort fallback (should never happen if JSON loads correctly)
	return EncounterDifficultyModifier{
		PowerMultiplier: 1.0,
		MinSquads:       3,
		MaxSquads:       5,
	}
}

// GetSquadPreferences retrieves preferred squad composition for an encounter type.
// Returns nil if encounter type not found (allows random composition fallback).
func GetSquadPreferences(encounterType string) []string {
	for _, template := range templates.EncounterTypeTemplates {
		if template.ID == encounterType {
			// Return copy to prevent external modification
			prefs := make([]string, len(template.SquadPreferences))
			copy(prefs, template.SquadPreferences)
			return prefs
		}
	}
	return nil // Not found - allows caller to handle random composition
}

// GetEncounterTypeByID retrieves full encounter type template.
// Returns nil if not found.
func GetEncounterTypeByID(id string) *templates.JSONEncounterType {
	for i := range templates.EncounterTypeTemplates {
		if templates.EncounterTypeTemplates[i].ID == id {
			return &templates.EncounterTypeTemplates[i]
		}
	}
	return nil
}
