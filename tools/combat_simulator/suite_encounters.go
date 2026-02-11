package main

import (
	"fmt"
	"game_main/tactical/squads"
	"game_main/templates"
)

// factionRoster maps each faction to its melee, ranged, and magic representative units.
var factionRoster = map[string]map[string]string{
	"Necromancers": {"melee": "Warrior", "ranged": "Skeleton Archer", "magic": "Priest"},
	"Bandits":      {"melee": "Fighter", "ranged": "Crossbowman", "magic": "Ranger"},
	"Cultists":     {"melee": "Battle Mage", "ranged": "Warlock", "magic": "Sorcerer"},
	"Beasts":       {"melee": "Goblin Raider", "ranged": "Scout", "magic": "Mage"},
	"Orcs":         {"melee": "Spearman", "ranged": "Marksman", "magic": "Wizard"},
}

// standardPlayerSquad is a balanced 5-unit player composition used in all encounter tests.
var standardPlayerUnits = []string{"Knight", "Swordsman", "Archer", "Wizard", "Paladin"}
var standardPlayerPositions = [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 1}, {2, 1}}

// GenerateEncounterSuite generates scenarios from encounterdata.json definitions.
// Each encounter uses faction-appropriate units based on squad preferences.
// The player side uses a standardized balanced composition.
func GenerateEncounterSuite(pool *UnitPool) []Scenario {
	var scenarios []Scenario

	for _, enc := range templates.EncounterDefinitionTemplates {
		roster, ok := factionRoster[enc.FactionID]
		if !ok {
			continue
		}

		// Build enemy squads from squad preferences
		var enemySquads []SquadBlueprint
		positions3 := [][2]int{{0, 1}, {1, 1}, {2, 1}}

		for sqIdx, pref := range enc.SquadPreferences {
			unitName, ok := roster[pref]
			if !ok {
				continue
			}

			squadName := fmt.Sprintf("%s %s %d", enc.FactionID, pref, sqIdx+1)
			enemySquads = append(enemySquads, makeSquadBP(
				squadName,
				squadFormationForPref(pref),
				[]string{unitName, unitName, unitName},
				positions3,
			))
		}

		if len(enemySquads) == 0 {
			continue
		}

		// Build player side
		playerSquad := makeSquadBP(
			"Player Squad",
			squads.FormationBalanced,
			standardPlayerUnits,
			standardPlayerPositions,
		)

		bp := ScenarioBlueprint{
			Name:  fmt.Sprintf("Encounter: %s (%s)", enc.EncounterTypeName, enc.FactionID),
			Suite: "encounters",
			SideA: []SquadBlueprint{playerSquad},
			SideB: enemySquads,
		}

		scenarios = append(scenarios, blueprintToScenario(pool, bp))
	}

	return scenarios
}

// squadFormationForPref returns an appropriate formation for a squad preference type.
func squadFormationForPref(pref string) squads.FormationType {
	switch pref {
	case "melee":
		return squads.FormationOffensive
	case "ranged":
		return squads.FormationRanged
	case "magic":
		return squads.FormationBalanced
	default:
		return squads.FormationBalanced
	}
}
