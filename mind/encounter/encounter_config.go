package encounter

import (
	"game_main/overworld/core"
	"game_main/templates"
)

// getDifficultyModifier retrieves difficulty settings for a given encounter level.
// Falls back to level 3 (fair fight) if level is invalid.
// Applies global difficulty overlay (power scale, squad/unit count offsets).
func getDifficultyModifier(level int) templates.JSONEncounterDifficulty {
	var result templates.JSONEncounterDifficulty

	// Search for matching difficulty level in templates
	found := false
	for _, t := range templates.EncounterDifficultyTemplates {
		if t.Level == level {
			result = t
			found = true
			break
		}
	}

	if !found {
		// Fallback to level 3 (fair fight) if not found
		for _, t := range templates.EncounterDifficultyTemplates {
			if t.Level == 3 {
				result = t
				found = true
				break
			}
		}
	}

	if !found {
		// Last resort fallback (should never happen if JSON loads correctly)
		result = templates.JSONEncounterDifficulty{
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

	// Apply global difficulty overlay
	diff := templates.GlobalDifficulty.Encounter()
	result.PowerMultiplier *= diff.PowerMultiplierScale

	result.SquadCount += diff.SquadCountOffset
	if result.SquadCount < 1 {
		result.SquadCount = 1
	}

	result.MinUnitsPerSquad += diff.MinUnitsPerSquadOffset
	if result.MinUnitsPerSquad < 1 {
		result.MinUnitsPerSquad = 1
	}

	result.MaxUnitsPerSquad += diff.MaxUnitsPerSquadOffset
	if result.MaxUnitsPerSquad < result.MinUnitsPerSquad {
		result.MaxUnitsPerSquad = result.MinUnitsPerSquad
	}

	return result
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
