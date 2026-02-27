package main

import (
	"fmt"
	"game_main/tactical/squads"
)

// GenerateCompositionSuite generates scenarios testing role, attack type,
// composition ratio, and formation matchups.
func GenerateCompositionSuite(pool *UnitPool) []Scenario {
	var scenarios []Scenario
	scenarios = append(scenarios, roleMatchups(pool)...)
	scenarios = append(scenarios, attackTypeMatchups(pool)...)
	scenarios = append(scenarios, compositionRatios(pool)...)
	scenarios = append(scenarios, formationTests(pool)...)
	scenarios = append(scenarios, healMatchups(pool)...)
	return scenarios
}

// roleMatchups tests pure role compositions against each other.
func roleMatchups(pool *UnitPool) []Scenario {
	// Representative 1x1 units per role
	tanks := []string{"Knight", "Fighter", "Spearman"}
	dps := []string{"Swordsman", "Assassin", "Archer"}
	support := []string{"Paladin", "Cleric", "Mage"}

	type matchup struct {
		name  string
		sideA []string
		sideB []string
	}

	matchups := []matchup{
		{"Role: All-Tank vs All-DPS", tanks, dps},
		{"Role: All-Tank vs All-Support", tanks, support},
		{"Role: All-DPS vs All-Support", dps, support},
		{"Role: Tank Mirror", tanks, tanks},
		{"Role: DPS Mirror", dps, dps},
		{"Role: Support Mirror", support, support},
	}

	positions := [][2]int{{0, 1}, {1, 1}, {2, 1}}
	var scenarios []Scenario

	for _, m := range matchups {
		sideA := makeSquadBP("Side A", squads.FormationBalanced, m.sideA, positions)
		sideB := makeSquadBP("Side B", squads.FormationBalanced, m.sideB, positions)

		bp := ScenarioBlueprint{
			Name:  m.name,
			Suite: "compositions",
			SideA: []SquadBlueprint{sideA},
			SideB: []SquadBlueprint{sideB},
		}
		scenarios = append(scenarios, blueprintToScenario(pool, bp))
	}

	return scenarios
}

// attackTypeMatchups tests pure attack type compositions against each other.
func attackTypeMatchups(pool *UnitPool) []Scenario {
	meleeRow := []string{"Knight", "Warrior", "Goblin Raider"}
	meleeCol := []string{"Fighter", "Swordsman", "Assassin"}
	ranged := []string{"Archer", "Crossbowman", "Marksman"}
	magic := []string{"Wizard", "Sorcerer", "Warlock"}

	type matchup struct {
		name  string
		sideA []string
		sideB []string
	}

	matchups := []matchup{
		{"Attack: MeleeRow vs Ranged", meleeRow, ranged},
		{"Attack: MeleeColumn vs Magic", meleeCol, magic},
		{"Attack: MeleeRow vs MeleeColumn", meleeRow, meleeCol},
		{"Attack: Ranged vs Magic", ranged, magic},
		{"Attack: MeleeRow vs Magic", meleeRow, magic},
		{"Attack: MeleeColumn vs Ranged", meleeCol, ranged},
	}

	positions := [][2]int{{0, 1}, {1, 1}, {2, 1}}
	var scenarios []Scenario

	for _, m := range matchups {
		sideA := makeSquadBP("Side A", squads.FormationBalanced, m.sideA, positions)
		sideB := makeSquadBP("Side B", squads.FormationBalanced, m.sideB, positions)

		bp := ScenarioBlueprint{
			Name:  m.name,
			Suite: "compositions",
			SideA: []SquadBlueprint{sideA},
			SideB: []SquadBlueprint{sideB},
		}
		scenarios = append(scenarios, blueprintToScenario(pool, bp))
	}

	return scenarios
}

// compositionRatios tests different role distributions within squads.
func compositionRatios(pool *UnitPool) []Scenario {
	positions5 := [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 1}, {2, 1}}

	type comp struct {
		name  string
		units []string
	}

	ratios := [][2]comp{
		// Tank-heavy vs DPS-heavy
		{
			comp{"2T/2D/1S", []string{"Knight", "Spearman", "Swordsman", "Archer", "Paladin"}},
			comp{"1T/3D/1S", []string{"Fighter", "Assassin", "Marksman", "Wizard", "Cleric"}},
		},
		// Support-heavy vs Pure DPS
		{
			comp{"1T/1D/3S", []string{"Knight", "Swordsman", "Paladin", "Mage", "Cleric"}},
			comp{"1T/4D", []string{"Fighter", "Assassin", "Archer", "Sorcerer", "Warlock"}},
		},
		// Balanced vs Full DPS
		{
			comp{"Balanced 2T/2D/1S", []string{"Knight", "Fighter", "Warrior", "Archer", "Paladin"}},
			comp{"Full DPS", []string{"Swordsman", "Assassin", "Marksman", "Wizard", "Warlock"}},
		},
		// Melee-heavy vs Ranged-heavy
		{
			comp{"4Melee/1Support", []string{"Knight", "Fighter", "Warrior", "Swordsman", "Paladin"}},
			comp{"4Ranged/1Support", []string{"Archer", "Crossbowman", "Marksman", "Scout", "Mage"}},
		},
		// High-cover vs No-cover
		{
			comp{"High Cover", []string{"Knight", "Paladin", "Spearman", "Cleric", "Priest"}},
			comp{"No Cover", []string{"Swordsman", "Assassin", "Archer", "Marksman", "Warlock"}},
		},
		// Magic-focused vs Physical-focused
		{
			comp{"Magic Squad", []string{"Wizard", "Sorcerer", "Warlock", "Mage", "Priest"}},
			comp{"Physical Squad", []string{"Knight", "Fighter", "Swordsman", "Warrior", "Spearman"}},
		},
	}

	var scenarios []Scenario
	for _, r := range ratios {
		sideA := makeSquadBP(r[0].name, squads.FormationBalanced, r[0].units, positions5)
		sideB := makeSquadBP(r[1].name, squads.FormationBalanced, r[1].units, positions5)

		bp := ScenarioBlueprint{
			Name:  fmt.Sprintf("Comp: %s vs %s", r[0].name, r[1].name),
			Suite: "compositions",
			SideA: []SquadBlueprint{sideA},
			SideB: []SquadBlueprint{sideB},
		}
		scenarios = append(scenarios, blueprintToScenario(pool, bp))
	}

	return scenarios
}

// healMatchups tests compositions that include heal-type units (Cleric, Priest)
// alongside damage dealers. These verify that healing integrates properly
// into combat logging and that mixed compositions behave correctly.
func healMatchups(pool *UnitPool) []Scenario {
	type matchup struct {
		name  string
		sideA []string
		sideB []string
	}

	matchups := []matchup{
		{"Heal: Mixed+Healer vs Pure DPS",
			[]string{"Knight", "Swordsman", "Cleric"},
			[]string{"Assassin", "Archer", "Wizard"}},
		{"Heal: Double Healer vs Tanks",
			[]string{"Fighter", "Cleric", "Priest"},
			[]string{"Knight", "Spearman", "Paladin"}},
		{"Heal: Healer+Ranged vs Melee",
			[]string{"Archer", "Marksman", "Cleric"},
			[]string{"Knight", "Fighter", "Warrior"}},
		{"Heal: Mirror (both have healer)",
			[]string{"Knight", "Archer", "Cleric"},
			[]string{"Fighter", "Wizard", "Priest"}},
	}

	positions := [][2]int{{0, 1}, {1, 1}, {2, 1}}
	var scenarios []Scenario

	for _, m := range matchups {
		sideA := makeSquadBP("Side A", squads.FormationBalanced, m.sideA, positions)
		sideB := makeSquadBP("Side B", squads.FormationBalanced, m.sideB, positions)

		bp := ScenarioBlueprint{
			Name:  m.name,
			Suite: "compositions",
			SideA: []SquadBlueprint{sideA},
			SideB: []SquadBlueprint{sideB},
		}
		scenarios = append(scenarios, blueprintToScenario(pool, bp))
	}

	return scenarios
}

// formationTests tests the same composition under different formations.
func formationTests(pool *UnitPool) []Scenario {
	units := []string{"Knight", "Swordsman", "Archer", "Wizard", "Paladin"}
	positions5 := [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 1}, {2, 1}}

	type formPair struct {
		nameA string
		formA squads.FormationType
		nameB string
		formB squads.FormationType
	}

	pairs := []formPair{
		{"Balanced", squads.FormationBalanced, "Offensive", squads.FormationOffensive},
		{"Balanced", squads.FormationBalanced, "Defensive", squads.FormationDefensive},
		{"Balanced", squads.FormationBalanced, "Ranged", squads.FormationRanged},
		{"Offensive", squads.FormationOffensive, "Defensive", squads.FormationDefensive},
		{"Offensive", squads.FormationOffensive, "Ranged", squads.FormationRanged},
		{"Defensive", squads.FormationDefensive, "Ranged", squads.FormationRanged},
	}

	var scenarios []Scenario
	for _, f := range pairs {
		sideA := makeSquadBP(f.nameA+" Formation", f.formA, units, positions5)
		sideB := makeSquadBP(f.nameB+" Formation", f.formB, units, positions5)

		bp := ScenarioBlueprint{
			Name:  fmt.Sprintf("Formation: %s vs %s", f.nameA, f.nameB),
			Suite: "compositions",
			SideA: []SquadBlueprint{sideA},
			SideB: []SquadBlueprint{sideB},
		}
		scenarios = append(scenarios, blueprintToScenario(pool, bp))
	}

	return scenarios
}
