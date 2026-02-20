package main

import (
	"fmt"
	"game_main/tactical/squads"
)

// GenerateDuelSuite generates all pairwise monotype duel scenarios.
// Each unit type faces every other type in a deterministic squad.
// 1x1 units: 3v3, 2x1 units: 3v3, 2x2 units: 1v1.
func GenerateDuelSuite(pool *UnitPool) []Scenario {
	var scenarios []Scenario

	for i := 0; i < len(pool.All); i++ {
		for j := i + 1; j < len(pool.All); j++ {
			unitA := pool.All[i]
			unitB := pool.All[j]

			placementsA := duelPlacements(unitA)
			placementsB := duelPlacements(unitB)

			name := fmt.Sprintf("Duel: %s vs %s", unitA.UnitType, unitB.UnitType)

			bp := ScenarioBlueprint{
				Name:  name,
				Suite: "duels",
				SideA: []SquadBlueprint{{
					Name:      fmt.Sprintf("%s Squad", unitA.UnitType),
					Formation: squads.FormationBalanced,
					Units:     placementsA,
				}},
				SideB: []SquadBlueprint{{
					Name:      fmt.Sprintf("%s Squad", unitB.UnitType),
					Formation: squads.FormationBalanced,
					Units:     placementsB,
				}},
			}

			scenarios = append(scenarios, blueprintToScenario(pool, bp))
		}
	}

	return scenarios
}

// duelPlacements returns deterministic unit placements for a monotype duel squad.
// Grid is 3x3 (rows 0-2, cols 0-2).
//   - 1x1: 3 units at (0,1), (1,1), (2,1) - center column
//   - 2x1: 3 units at (0,0), (1,0), (2,0) - spanning cols 0-1 per row
//   - 2x2: 1 unit at (0,0) - spanning rows 0-1, cols 0-1
func duelPlacements(tmpl squads.UnitTemplate) []UnitPlacement {
	w := tmpl.GridWidth
	if w == 0 {
		w = 1
	}
	h := tmpl.GridHeight
	if h == 0 {
		h = 1
	}

	switch {
	case w <= 1 && h <= 1:
		return []UnitPlacement{
			{Name: tmpl.UnitType, GridRow: 0, GridCol: 1, IsLeader: true},
			{Name: tmpl.UnitType, GridRow: 1, GridCol: 1},
			{Name: tmpl.UnitType, GridRow: 2, GridCol: 1},
		}
	case w == 2 && h == 1:
		return []UnitPlacement{
			{Name: tmpl.UnitType, GridRow: 0, GridCol: 0, IsLeader: true},
			{Name: tmpl.UnitType, GridRow: 1, GridCol: 0},
			{Name: tmpl.UnitType, GridRow: 2, GridCol: 0},
		}
	case w == 2 && h == 2:
		return []UnitPlacement{
			{Name: tmpl.UnitType, GridRow: 0, GridCol: 0, IsLeader: true},
		}
	default:
		return []UnitPlacement{
			{Name: tmpl.UnitType, GridRow: 0, GridCol: 0, IsLeader: true},
		}
	}
}
