package encounter

import (
	"game_main/overworld/core"
	"game_main/templates"
)

// getDifficultyModifier retrieves difficulty settings for a given encounter level.
// Falls back to level 3 (fair fight) if level is invalid.
func getDifficultyModifier(level int) templates.JSONEncounterDifficulty {
	// Search for matching difficulty level in templates
	for _, t := range templates.EncounterDifficultyTemplates {
		if t.Level == level {
			return t
		}
	}

	// Fallback to level 3 (fair fight) if not found
	for _, t := range templates.EncounterDifficultyTemplates {
		if t.Level == 3 {
			return t
		}
	}

	// Last resort fallback (should never happen if JSON loads correctly)
	return templates.JSONEncounterDifficulty{
		Level:            3,
		Name:             "Fair Fight",
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
	enc := core.GetNodeRegistry().GetEncounterByTypeID(encounterTypeID)
	if enc != nil && enc.ID != "default" {
		// Return copy to prevent external modification
		prefs := make([]string, len(enc.SquadPreferences))
		copy(prefs, enc.SquadPreferences)
		return prefs
	}
	return nil // Not found - allows caller to handle random composition
}
