package simulator

import (
	"fmt"
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// createSimSquad is the headless equivalent of squads.CreateSquadFromTemplate.
// It creates a squad entity with units using templates.CreateUnit (no images/rendering).
// All component-adding logic mirrors squadcreation.go:200-355.
func createSimSquad(
	manager *common.EntityManager,
	squadName string,
	formation squadcore.FormationType,
	worldPos coords.LogicalPosition,
	unitTemplates []unitdefs.UnitTemplate,
) ecs.EntityID {
	squadEntity := manager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(squadcore.SquadComponent, &squadcore.SquadData{
		SquadID:    squadID,
		Name:       squadName,
		Formation:  formation,
		Morale:     100,
		TurnCount:  0,
		MaxUnits:   9,
		IsDeployed: false,
	})
	// Atomically add position component and register with position system
	manager.RegisterEntityPosition(squadEntity, worldPos)

	occupied := make(map[string]bool)

	for _, tmpl := range unitTemplates {
		width := tmpl.GridWidth
		if width == 0 {
			width = 1
		}
		height := tmpl.GridHeight
		if height == 0 {
			height = 1
		}

		if tmpl.GridRow < 0 || tmpl.GridCol < 0 {
			continue
		}
		if tmpl.GridRow+height > 3 || tmpl.GridCol+width > 3 {
			continue
		}

		canPlace := true
		var cellsToOccupy [][2]int
		for r := tmpl.GridRow; r < tmpl.GridRow+height; r++ {
			for c := tmpl.GridCol; c < tmpl.GridCol+width; c++ {
				key := fmt.Sprintf("%d,%d", r, c)
				if occupied[key] {
					canPlace = false
					break
				}
				cellsToOccupy = append(cellsToOccupy, [2]int{r, c})
			}
			if !canPlace {
				break
			}
		}
		if !canPlace {
			continue
		}

		// Headless unit creation (no images) - same base as templates.CreateUnit
		unitEntity := templates.CreateUnit(
			*manager,
			tmpl.UnitType,
			tmpl.Attributes,
			&coords.LogicalPosition{X: worldPos.X, Y: worldPos.Y},
		)

		// All component additions below mirror squadcreation.go:272-348
		unitEntity.AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{
			SquadID: squadID,
		})

		// Add all squad-specific components from template
		squadcore.ApplyUnitComponents(unitEntity, tmpl, tmpl.GridRow, tmpl.GridCol)

		if tmpl.IsLeader {
			squadcore.AddLeaderComponents(unitEntity)
		}

		for _, cell := range cellsToOccupy {
			key := fmt.Sprintf("%d,%d", cell[0], cell[1])
			occupied[key] = true
		}
	}

	return squadID
}

// ========================================
// SQUAD BUILDER
// Shared scaffolding for the squad factory functions below. Each factory
// supplies a formation, position list, unit count, and a per-slot selector;
// buildSquad assembles the unit templates, picks a leader, and delegates to
// createSimSquad.
// ========================================

type squadConfig struct {
	formation  squadcore.FormationType
	positions  [][2]int
	unitCount  int
	selectUnit func(i int) unitdefs.UnitTemplate
}

func buildSquad(manager *common.EntityManager, name string, pos coords.LogicalPosition, cfg squadConfig) ecs.EntityID {
	leaderIdx := common.RandomInt(cfg.unitCount)
	var units []unitdefs.UnitTemplate
	for i := 0; i < cfg.unitCount && i < len(cfg.positions); i++ {
		unit := cfg.selectUnit(i)
		unit.GridRow = cfg.positions[i][0]
		unit.GridCol = cfg.positions[i][1]
		unit.IsLeader = (i == leaderIdx)
		if unit.IsLeader {
			unit.Attributes.Leadership = 20
		}
		units = append(units, unit)
	}
	return createSimSquad(manager, name, cfg.formation, pos, units)
}

// filterUnits returns all unit templates from unitdefs.Units that satisfy the predicate.
// Replaces the prior per-attribute filterUnitsByAttackRange/filterUnitsByAttackType/filterUnitsByRole helpers.
func filterUnits(predicate func(unitdefs.UnitTemplate) bool) []unitdefs.UnitTemplate {
	var filtered []unitdefs.UnitTemplate
	for _, unit := range unitdefs.Units {
		if predicate(unit) {
			filtered = append(filtered, unit)
		}
	}
	return filtered
}

// ========================================
// SQUAD FACTORY FUNCTIONS
// Headless mirrors of initialplayersquads.go factory functions.
// Same unit selection logic, same grid positions, same leader assignment.
// Only difference: calls createSimSquad instead of CreateSquadFromTemplate.
// ========================================

// createBalancedSquad mirrors initialplayersquads.go:createBalancedSquad.
// Picks from Units[i%len(Units)] with balanced formation positions.
func createBalancedSquad(manager *common.EntityManager, squadName string, worldPos coords.LogicalPosition) ecs.EntityID {
	pool := unitdefs.Units
	maxUnits := 5
	if len(pool) < maxUnits {
		maxUnits = len(pool)
	}
	return buildSquad(manager, squadName, worldPos, squadConfig{
		formation: squadcore.FormationBalanced,
		positions: [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 1}, {2, 1}},
		unitCount: maxUnits,
		selectUnit: func(i int) unitdefs.UnitTemplate { return pool[i%len(pool)] },
	})
}

// createRangedSquad mirrors initialplayersquads.go:createRangedSquad.
// Filters units by AttackRange >= 3, random selection, 3-5 units.
func createRangedSquad(manager *common.EntityManager, squadName string, worldPos coords.LogicalPosition) ecs.EntityID {
	pool := filterUnits(func(u unitdefs.UnitTemplate) bool { return u.AttackRange >= 3 })
	if len(pool) == 0 {
		return createBalancedSquad(manager, squadName, worldPos)
	}
	count := common.GetRandomBetween(3, 5)
	return buildSquad(manager, squadName, worldPos, squadConfig{
		formation: squadcore.FormationRanged,
		positions: [][2]int{{0, 0}, {1, 1}, {2, 2}, {0, 2}, {1, 0}},
		unitCount: count,
		selectUnit: func(i int) unitdefs.UnitTemplate { return pool[common.RandomInt(len(pool))] },
	})
}

// createMagicSquad mirrors initialplayersquads.go:createMagicSquad.
// Filters units by AttackType == Magic, exactly 3 units.
func createMagicSquad(manager *common.EntityManager, squadName string, worldPos coords.LogicalPosition) ecs.EntityID {
	pool := filterUnits(func(u unitdefs.UnitTemplate) bool { return u.AttackType == unitdefs.AttackTypeMagic })
	if len(pool) == 0 {
		return createBalancedSquad(manager, squadName, worldPos)
	}
	return buildSquad(manager, squadName, worldPos, squadConfig{
		formation: squadcore.FormationBalanced,
		positions: [][2]int{{0, 1}, {1, 0}, {2, 1}},
		unitCount: 3,
		selectUnit: func(i int) unitdefs.UnitTemplate { return pool[common.RandomInt(len(pool))] },
	})
}

// createMixedSquad mirrors initialplayersquads.go:createMixedSquad.
// Alternates between ranged and magic units, 4-5 units.
func createMixedSquad(manager *common.EntityManager, squadName string, worldPos coords.LogicalPosition) ecs.EntityID {
	ranged := filterUnits(func(u unitdefs.UnitTemplate) bool { return u.AttackRange >= 3 })
	magic := filterUnits(func(u unitdefs.UnitTemplate) bool { return u.AttackType == unitdefs.AttackTypeMagic })
	if len(ranged) == 0 || len(magic) == 0 {
		return createBalancedSquad(manager, squadName, worldPos)
	}
	count := common.GetRandomBetween(4, 5)
	return buildSquad(manager, squadName, worldPos, squadConfig{
		formation: squadcore.FormationBalanced,
		positions: [][2]int{{0, 0}, {1, 1}, {2, 2}, {1, 2}, {2, 0}},
		unitCount: count,
		selectUnit: func(i int) unitdefs.UnitTemplate {
			if i%2 == 0 {
				return ranged[common.RandomInt(len(ranged))]
			}
			return magic[common.RandomInt(len(magic))]
		},
	})
}

// createMeleeSquad creates an all-melee squad using MeleeRow attack type units.
func createMeleeSquad(manager *common.EntityManager, squadName string, worldPos coords.LogicalPosition) ecs.EntityID {
	pool := filterUnits(func(u unitdefs.UnitTemplate) bool { return u.AttackType == unitdefs.AttackTypeMeleeRow })
	if len(pool) == 0 {
		return createBalancedSquad(manager, squadName, worldPos)
	}
	count := 5
	if len(pool) < count {
		count = len(pool)
	}
	return buildSquad(manager, squadName, worldPos, squadConfig{
		formation: squadcore.FormationOffensive,
		positions: [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 1}, {2, 1}},
		unitCount: count,
		selectUnit: func(i int) unitdefs.UnitTemplate { return pool[common.RandomInt(len(pool))] },
	})
}
