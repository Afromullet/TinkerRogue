package main

import (
	"game_main/common"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// UnitPlacement specifies a unit and its exact grid position in a squad.
type UnitPlacement struct {
	Name     string
	GridRow  int
	GridCol  int
	IsLeader bool
}

// SquadBlueprint defines the composition and layout of one squad.
type SquadBlueprint struct {
	Name      string
	Formation squads.FormationType
	Units     []UnitPlacement
}

// ScenarioBlueprint defines a complete scenario from blueprints.
type ScenarioBlueprint struct {
	Name  string
	Suite string
	SideA []SquadBlueprint
	SideB []SquadBlueprint
}

// createSquadFromBlueprint builds a squad entity from a blueprint using the unit pool.
func createSquadFromBlueprint(pool *UnitPool, manager *common.EntityManager, bp SquadBlueprint, worldPos coords.LogicalPosition) ecs.EntityID {
	var units []squads.UnitTemplate
	for _, placement := range bp.Units {
		tmpl := pool.Get(placement.Name)
		tmpl.GridRow = placement.GridRow
		tmpl.GridCol = placement.GridCol
		tmpl.IsLeader = placement.IsLeader
		if placement.IsLeader {
			tmpl.Attributes.Leadership = 20
		}
		units = append(units, tmpl)
	}
	return createSimSquad(manager, bp.Name, bp.Formation, worldPos, units)
}

// blueprintToScenario converts a ScenarioBlueprint into a Scenario using closures.
func blueprintToScenario(pool *UnitPool, bp ScenarioBlueprint) Scenario {
	var sideA []SquadSpec
	for _, sbp := range bp.SideA {
		captured := sbp
		sideA = append(sideA, SquadSpec{
			Name: captured.Name,
			CreateFn: func(manager *common.EntityManager, name string, pos coords.LogicalPosition) ecs.EntityID {
				return createSquadFromBlueprint(pool, manager, captured, pos)
			},
		})
	}

	var sideB []SquadSpec
	for _, sbp := range bp.SideB {
		captured := sbp
		sideB = append(sideB, SquadSpec{
			Name: captured.Name,
			CreateFn: func(manager *common.EntityManager, name string, pos coords.LogicalPosition) ecs.EntityID {
				return createSquadFromBlueprint(pool, manager, captured, pos)
			},
		})
	}

	return Scenario{
		Name:  bp.Name,
		SideA: sideA,
		SideB: sideB,
	}
}

// makeSquadBP creates a SquadBlueprint from unit names and positions.
// The first unit is always the leader.
func makeSquadBP(name string, formation squads.FormationType, unitNames []string, positions [][2]int) SquadBlueprint {
	var placements []UnitPlacement
	for i, uname := range unitNames {
		if i >= len(positions) {
			break
		}
		placements = append(placements, UnitPlacement{
			Name:     uname,
			GridRow:  positions[i][0],
			GridCol:  positions[i][1],
			IsLeader: i == 0,
		})
	}
	return SquadBlueprint{
		Name:      name,
		Formation: formation,
		Units:     placements,
	}
}
