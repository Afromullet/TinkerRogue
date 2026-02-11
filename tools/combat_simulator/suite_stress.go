package main

import (
	"game_main/tactical/squads"
)

// GenerateStressSuite generates edge case scenarios testing specific combat mechanics.
func GenerateStressSuite(pool *UnitPool) []Scenario {
	var scenarios []Scenario
	scenarios = append(scenarios, coverStress(pool)...)
	scenarios = append(scenarios, numericalAdvantage(pool)...)
	scenarios = append(scenarios, sizeMismatch(pool)...)
	scenarios = append(scenarios, speedDifferential(pool)...)
	scenarios = append(scenarios, dexterityExtremes(pool)...)
	scenarios = append(scenarios, magicSplash(pool)...)
	return scenarios
}

// coverStress tests high-cover squads against zero-cover glass cannons.
func coverStress(pool *UnitPool) []Scenario {
	pos5 := [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 1}, {2, 1}}
	pos3 := [][2]int{{0, 1}, {1, 1}, {2, 1}}

	return []Scenario{
		// Max cover vs zero cover glass cannons
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: Max Cover vs Glass Cannons",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"Max Cover", squads.FormationDefensive,
				[]string{"Knight", "Paladin", "Spearman", "Cleric", "Priest"},
				pos5,
			)},
			SideB: []SquadBlueprint{makeSquadBP(
				"Glass Cannons", squads.FormationOffensive,
				[]string{"Assassin", "Marksman", "Swordsman", "Warlock", "Wizard"},
				pos5,
			)},
		}),
		// Medium cover vs no cover
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: Medium Cover vs No Cover",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"Medium Cover", squads.FormationBalanced,
				[]string{"Fighter", "Battle Mage", "Mage"},
				pos3,
			)},
			SideB: []SquadBlueprint{makeSquadBP(
				"No Cover", squads.FormationBalanced,
				[]string{"Warrior", "Goblin Raider", "Archer"},
				pos3,
			)},
		}),
		// All cover vs wide-splash magic
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: Cover Wall vs Magic Splash",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"Cover Wall", squads.FormationDefensive,
				[]string{"Knight", "Paladin", "Priest"},
				pos3,
			)},
			SideB: []SquadBlueprint{makeSquadBP(
				"Splash Casters", squads.FormationBalanced,
				[]string{"Sorcerer", "Wizard", "Skeleton Archer"},
				pos3,
			)},
		}),
	}
}

// numericalAdvantage tests lopsided squad counts.
func numericalAdvantage(pool *UnitPool) []Scenario {
	pos5 := [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 1}, {2, 1}}
	pos3 := [][2]int{{0, 1}, {1, 1}, {2, 1}}

	return []Scenario{
		// 1 full squad vs 2 small squads
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: 1 Full vs 2 Small Squads",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"Full Squad", squads.FormationBalanced,
				[]string{"Knight", "Swordsman", "Archer", "Wizard", "Paladin"},
				pos5,
			)},
			SideB: []SquadBlueprint{
				makeSquadBP("Small 1", squads.FormationBalanced,
					[]string{"Warrior", "Crossbowman", "Mage"}, pos3),
				makeSquadBP("Small 2", squads.FormationBalanced,
					[]string{"Fighter", "Marksman", "Cleric"}, pos3),
			},
		}),
		// 2 full squads vs 4 small squads
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: 2 Full vs 4 Small Squads",
			Suite: "stress",
			SideA: []SquadBlueprint{
				makeSquadBP("Full A1", squads.FormationBalanced,
					[]string{"Knight", "Swordsman", "Archer", "Wizard", "Paladin"}, pos5),
				makeSquadBP("Full A2", squads.FormationBalanced,
					[]string{"Fighter", "Assassin", "Marksman", "Sorcerer", "Cleric"}, pos5),
			},
			SideB: []SquadBlueprint{
				makeSquadBP("Small B1", squads.FormationBalanced,
					[]string{"Warrior", "Crossbowman", "Mage"}, pos3),
				makeSquadBP("Small B2", squads.FormationBalanced,
					[]string{"Spearman", "Archer", "Priest"}, pos3),
				makeSquadBP("Small B3", squads.FormationOffensive,
					[]string{"Goblin Raider", "Rogue", "Scout"}, pos3),
				makeSquadBP("Small B4", squads.FormationBalanced,
					[]string{"Battle Mage", "Ranger", "Warlock"}, pos3),
			},
		}),
		// 1 elite squad vs 3 weak squads
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: 1 Elite vs 3 Weak Squads",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"Elite", squads.FormationBalanced,
				[]string{"Knight", "Assassin", "Marksman", "Wizard", "Paladin"},
				pos5,
			)},
			SideB: []SquadBlueprint{
				makeSquadBP("Weak 1", squads.FormationBalanced,
					[]string{"Goblin Raider", "Skeleton Archer", "Rogue"}, pos3),
				makeSquadBP("Weak 2", squads.FormationBalanced,
					[]string{"Goblin Raider", "Skeleton Archer", "Rogue"}, pos3),
				makeSquadBP("Weak 3", squads.FormationBalanced,
					[]string{"Goblin Raider", "Skeleton Archer", "Rogue"}, pos3),
			},
		}),
	}
}

// sizeMismatch tests large units (2x2, 2x1) against swarms of small units.
func sizeMismatch(pool *UnitPool) []Scenario {
	pos3 := [][2]int{{0, 1}, {1, 1}, {2, 1}}

	// Ogre (2x2) at position 0,0
	ogrePlacement := []UnitPlacement{
		{Name: "Ogre", GridRow: 0, GridCol: 0, IsLeader: true},
	}

	// Orc Warriors (2x1) at col 0, one per row
	orcWarriorPlacements := []UnitPlacement{
		{Name: "Orc Warrior", GridRow: 0, GridCol: 0, IsLeader: true},
		{Name: "Orc Warrior", GridRow: 1, GridCol: 0},
		{Name: "Orc Warrior", GridRow: 2, GridCol: 0},
	}

	return []Scenario{
		// Ogre (2x2) vs 3 fast 1x1 units
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: Ogre vs Assassin Swarm",
			Suite: "stress",
			SideA: []SquadBlueprint{{
				Name:      "Ogre",
				Formation: squads.FormationDefensive,
				Units:     ogrePlacement,
			}},
			SideB: []SquadBlueprint{makeSquadBP(
				"Assassin Swarm", squads.FormationOffensive,
				[]string{"Assassin", "Rogue", "Swordsman"},
				pos3,
			)},
		}),
		// 3 Orc Warriors (2x1) vs 3 Swordsmen (1x1)
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: Orc Warriors vs Swordsmen",
			Suite: "stress",
			SideA: []SquadBlueprint{{
				Name:      "Orc Warriors",
				Formation: squads.FormationOffensive,
				Units:     orcWarriorPlacements,
			}},
			SideB: []SquadBlueprint{makeSquadBP(
				"Swordsmen", squads.FormationOffensive,
				[]string{"Swordsman", "Swordsman", "Swordsman"},
				pos3,
			)},
		}),
	}
}

// speedDifferential tests squads of fast units against slow units.
func speedDifferential(pool *UnitPool) []Scenario {
	pos3 := [][2]int{{0, 1}, {1, 1}, {2, 1}}

	return []Scenario{
		// Max speed (5) vs min speed (2)
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: Speed 5 vs Speed 2",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"Fast (Spd 5)", squads.FormationOffensive,
				[]string{"Rogue", "Assassin", "Scout"},
				pos3,
			)},
			SideB: []SquadBlueprint{makeSquadBP(
				"Slow (Spd 2)", squads.FormationDefensive,
				[]string{"Knight", "Wizard", "Sorcerer"},
				pos3,
			)},
		}),
		// All speed 4+ vs all speed 2-3
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: Fast Squad vs Slow Squad",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"Fast Squad", squads.FormationOffensive,
				[]string{"Assassin", "Swordsman", "Goblin Raider"},
				pos3,
			)},
			SideB: []SquadBlueprint{makeSquadBP(
				"Slow Squad", squads.FormationDefensive,
				[]string{"Knight", "Paladin", "Wizard"},
				pos3,
			)},
		}),
	}
}

// dexterityExtremes tests high-dex dodge squads against high-strength brute squads.
func dexterityExtremes(pool *UnitPool) []Scenario {
	pos3 := [][2]int{{0, 1}, {1, 1}, {2, 1}}

	return []Scenario{
		// Max dex (Assassin 60, Rogue 55, Marksman 52) vs armored brutes
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: High Dex vs High Armor",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"High Dex", squads.FormationOffensive,
				[]string{"Assassin", "Rogue", "Marksman"},
				pos3,
			)},
			SideB: []SquadBlueprint{makeSquadBP(
				"High Armor", squads.FormationDefensive,
				[]string{"Knight", "Fighter", "Spearman"},
				pos3,
			)},
		}),
		// Dodge-heavy vs strength-heavy
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: Dodge Squad vs Brute Squad",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"Dodge Squad", squads.FormationBalanced,
				[]string{"Assassin", "Swordsman", "Scout"},
				pos3,
			)},
			SideB: []SquadBlueprint{makeSquadBP(
				"Brute Squad", squads.FormationBalanced,
				[]string{"Warrior", "Goblin Raider", "Fighter"},
				pos3,
			)},
		}),
	}
}

// magicSplash tests wide-pattern casters against narrow-pattern or physical units.
func magicSplash(pool *UnitPool) []Scenario {
	pos3 := [][2]int{{0, 1}, {1, 1}, {2, 1}}

	return []Scenario{
		// Sorcerer (9-cell) vs Warlock (4-cell corners)
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: Wide Splash vs Narrow Splash",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"Wide Splash", squads.FormationBalanced,
				[]string{"Sorcerer", "Sorcerer", "Sorcerer"},
				pos3,
			)},
			SideB: []SquadBlueprint{makeSquadBP(
				"Narrow Splash", squads.FormationBalanced,
				[]string{"Warlock", "Warlock", "Warlock"},
				pos3,
			)},
		}),
		// Wizard (6-cell top rows) vs physical melee
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: Wizard Barrage vs Melee Wall",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"Wizard Barrage", squads.FormationRanged,
				[]string{"Wizard", "Wizard", "Wizard"},
				pos3,
			)},
			SideB: []SquadBlueprint{makeSquadBP(
				"Melee Wall", squads.FormationDefensive,
				[]string{"Knight", "Fighter", "Warrior"},
				pos3,
			)},
		}),
		// All casters vs all melee
		blueprintToScenario(pool, ScenarioBlueprint{
			Name:  "Stress: Full Casters vs Full Melee",
			Suite: "stress",
			SideA: []SquadBlueprint{makeSquadBP(
				"Full Casters", squads.FormationRanged,
				[]string{"Sorcerer", "Wizard", "Skeleton Archer"},
				pos3,
			)},
			SideB: []SquadBlueprint{makeSquadBP(
				"Full Melee", squads.FormationOffensive,
				[]string{"Swordsman", "Assassin", "Warrior"},
				pos3,
			)},
		}),
	}
}
