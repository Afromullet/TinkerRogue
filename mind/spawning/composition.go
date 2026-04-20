package spawning

import (
	"game_main/common"
	"game_main/campaign/overworld/core"
)

// getSquadComposition returns squad type distribution based on encounter type.
// Falls back to a random balanced mix if the encounter type has no preferences configured.
func getSquadComposition(encounterData *core.OverworldEncounterData, count int) []string {
	if encounterData == nil || encounterData.EncounterType == "" {
		return generateRandomComposition(count)
	}

	preferences := getSquadPreferences(encounterData.EncounterType)
	if len(preferences) == 0 {
		return generateRandomComposition(count)
	}

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = preferences[i%len(preferences)]
	}
	return result
}

// generateRandomComposition creates a random mix of squad types.
func generateRandomComposition(count int) []string {
	types := []string{SquadTypeMelee, SquadTypeRanged, SquadTypeMagic}
	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = types[common.RandomInt(len(types))]
	}
	return result
}

// getSquadPreferences retrieves preferred squad composition for an encounter type.
// Returns nil if the encounter type is not found, allowing random-composition fallback.
func getSquadPreferences(encounterTypeID string) []string {
	enc := core.GetNodeRegistry().GetEncounterByTypeID(encounterTypeID)
	if enc != nil && enc.ID != "default" {
		prefs := make([]string, len(enc.SquadPreferences))
		copy(prefs, enc.SquadPreferences)
		return prefs
	}
	return nil
}
