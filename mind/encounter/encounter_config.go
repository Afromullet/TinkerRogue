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
				PowerMultiplier: template.PowerMultiplier,
				SquadCount:      template.SquadCount,
			}
		}
	}

	// Fallback to level 3 (fair fight) if not found
	for _, template := range templates.EncounterDifficultyTemplates {
		if template.Level == 3 {
			return EncounterDifficultyModifier{
				PowerMultiplier: template.PowerMultiplier,
				SquadCount:      template.SquadCount,
			}
		}
	}

	// Last resort fallback (should never happen if JSON loads correctly)
	return EncounterDifficultyModifier{
		PowerMultiplier: 1.0,
		SquadCount:      4,
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

// GetEncounterTypeByID retrieves encounter type info from the threat registry.
// Returns nil if not found.
func GetEncounterTypeByID(id string) *EncounterTypeInfo {
	def := core.GetThreatRegistry().GetByEncounterTypeID(id)
	if def.ID == "default" {
		return nil
	}
	return &EncounterTypeInfo{
		ID:                def.EncounterTypeID,
		Name:              def.EncounterTypeName,
		SquadPreferences:  def.SquadPreferences,
		DefaultDifficulty: def.DefaultDifficulty,
		Tags:              def.Tags,
	}
}

// EncounterTypeInfo provides encounter type information from the threat registry.
// This replaces the removed JSONEncounterType struct for external consumers.
type EncounterTypeInfo struct {
	ID                string
	Name              string
	SquadPreferences  []string
	DefaultDifficulty int
	Tags              []string
}
