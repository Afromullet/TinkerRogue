package main

import (
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// squadCreateFn is the factory function signature used by scenarios.
// Mirrors the pattern from initialplayersquads.go squad configs.
type squadCreateFn func(manager *common.EntityManager, name string, pos coords.LogicalPosition) ecs.EntityID

// Scenario defines a combat matchup between two sides.
type Scenario struct {
	Name  string
	SideA []SquadSpec
	SideB []SquadSpec
}

// SquadSpec defines how to build one squad for a scenario.
type SquadSpec struct {
	Name     string
	CreateFn squadCreateFn
}

// AllScenarios returns all predefined combat scenarios.
func AllScenarios() []Scenario {
	return []Scenario{
		scenario1SmallBalanced(),
		scenario2LargeBalanced(),
		scenario3MeleeVsRanged(),
		scenario4TankVsDPS(),
		scenario5MagicVsMelee(),
		scenario6Outnumbered(),
		scenario7OneBigVsManySmall(),
		scenario8MixedVsMixed(),
	}
}

// Scenario 1: Small Balanced vs Small Balanced (2v2)
func scenario1SmallBalanced() Scenario {
	return Scenario{
		Name: "Small Balanced vs Small Balanced",
		SideA: []SquadSpec{
			{"Alpha Balanced 1", createBalancedSquad},
			{"Alpha Balanced 2", createBalancedSquad},
		},
		SideB: []SquadSpec{
			{"Bravo Balanced 1", createBalancedSquad},
			{"Bravo Balanced 2", createBalancedSquad},
		},
	}
}

// Scenario 2: Large vs Large (4v4)
func scenario2LargeBalanced() Scenario {
	return Scenario{
		Name: "Large vs Large",
		SideA: []SquadSpec{
			{"Alpha Balanced 1", createBalancedSquad},
			{"Alpha Balanced 2", createBalancedSquad},
			{"Alpha Ranged 1", createRangedSquad},
			{"Alpha Mixed 1", createMixedSquad},
		},
		SideB: []SquadSpec{
			{"Bravo Balanced 1", createBalancedSquad},
			{"Bravo Balanced 2", createBalancedSquad},
			{"Bravo Ranged 1", createRangedSquad},
			{"Bravo Mixed 1", createMixedSquad},
		},
	}
}

// Scenario 3: Melee vs Ranged (3v3)
func scenario3MeleeVsRanged() Scenario {
	return Scenario{
		Name: "Melee vs Ranged",
		SideA: []SquadSpec{
			{"Melee Squad 1", createMeleeSquad},
			{"Melee Squad 2", createMeleeSquad},
			{"Melee Squad 3", createMeleeSquad},
		},
		SideB: []SquadSpec{
			{"Ranged Squad 1", createRangedSquad},
			{"Ranged Squad 2", createRangedSquad},
			{"Ranged Squad 3", createRangedSquad},
		},
	}
}

// Scenario 4: Balanced vs Ranged (tank-heavy balanced vs ranged-focused)
func scenario4TankVsDPS() Scenario {
	return Scenario{
		Name: "Balanced vs Ranged",
		SideA: []SquadSpec{
			{"Balanced Squad 1", createBalancedSquad},
			{"Balanced Squad 2", createBalancedSquad},
			{"Balanced Squad 3", createBalancedSquad},
		},
		SideB: []SquadSpec{
			{"Ranged Squad 1", createRangedSquad},
			{"Ranged Squad 2", createRangedSquad},
			{"Ranged Squad 3", createRangedSquad},
		},
	}
}

// Scenario 5: Magic vs Melee (2v2)
func scenario5MagicVsMelee() Scenario {
	return Scenario{
		Name: "Magic vs Melee",
		SideA: []SquadSpec{
			{"Magic Squad 1", createMagicSquad},
			{"Magic Squad 2", createMagicSquad},
		},
		SideB: []SquadSpec{
			{"Melee Squad 1", createMeleeSquad},
			{"Melee Squad 2", createMeleeSquad},
		},
	}
}

// Scenario 6: Outnumbered (2 balanced vs 4 balanced)
func scenario6Outnumbered() Scenario {
	return Scenario{
		Name: "Outnumbered (2 vs 4)",
		SideA: []SquadSpec{
			{"Strong Squad 1", createBalancedSquad},
			{"Strong Squad 2", createBalancedSquad},
		},
		SideB: []SquadSpec{
			{"Militia 1", createBalancedSquad},
			{"Militia 2", createBalancedSquad},
			{"Militia 3", createBalancedSquad},
			{"Militia 4", createBalancedSquad},
		},
	}
}

// Scenario 7: One Big vs Many Small (1 balanced vs 3 magic)
func scenario7OneBigVsManySmall() Scenario {
	return Scenario{
		Name: "One Big vs Many Small",
		SideA: []SquadSpec{
			{"Full Squad", createBalancedSquad},
		},
		SideB: []SquadSpec{
			{"Small Magic 1", createMagicSquad},
			{"Small Magic 2", createMagicSquad},
			{"Small Magic 3", createMagicSquad},
		},
	}
}

// Scenario 8: Mixed vs Mixed (diverse compositions, 3v3)
func scenario8MixedVsMixed() Scenario {
	return Scenario{
		Name: "Mixed vs Mixed",
		SideA: []SquadSpec{
			{"Alpha Balanced", createBalancedSquad},
			{"Alpha Ranged", createRangedSquad},
			{"Alpha Magic", createMagicSquad},
		},
		SideB: []SquadSpec{
			{"Bravo Melee", createMeleeSquad},
			{"Bravo Mixed", createMixedSquad},
			{"Bravo Balanced", createBalancedSquad},
		},
	}
}

// createScenarioSquads builds all squads for a scenario and returns side A and B squad IDs.
// Side A at (10, 10), Side B at (11, 10) - Chebyshev distance = 1.
func createScenarioSquads(manager *common.EntityManager, scenario Scenario) ([]ecs.EntityID, []ecs.EntityID) {
	posA := coords.LogicalPosition{X: 10, Y: 10}
	posB := coords.LogicalPosition{X: 11, Y: 10}

	var sideA []ecs.EntityID
	for _, spec := range scenario.SideA {
		squadID := spec.CreateFn(manager, spec.Name, posA)
		sideA = append(sideA, squadID)
	}

	var sideB []ecs.EntityID
	for _, spec := range scenario.SideB {
		squadID := spec.CreateFn(manager, spec.Name, posB)
		sideB = append(sideB, squadID)
	}

	return sideA, sideB
}
