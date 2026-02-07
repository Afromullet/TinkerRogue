package main

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"
	"game_main/templates"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// createSimSquad is the headless equivalent of squads.CreateSquadFromTemplate.
// It creates a squad entity with units using templates.CreateUnit (no images/rendering).
// All component-adding logic mirrors squadcreation.go:200-355.
func createSimSquad(
	manager *common.EntityManager,
	squadName string,
	formation squads.FormationType,
	worldPos coords.LogicalPosition,
	unitTemplates []squads.UnitTemplate,
) ecs.EntityID {
	squadEntity := manager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(squads.SquadComponent, &squads.SquadData{
		SquadID:     squadID,
		Name:        squadName,
		Formation:   formation,
		Morale:      100,
		TurnCount:   0,
		MaxUnits:   9,
		IsDeployed: false,
	})
	squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{
		X: worldPos.X,
		Y: worldPos.Y,
	})

	common.GlobalPositionSystem.AddEntity(squadID, worldPos)

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
			tmpl.Name,
			tmpl.Attributes,
			&coords.LogicalPosition{X: worldPos.X, Y: worldPos.Y},
		)

		// All component additions below mirror squadcreation.go:272-348
		unitEntity.AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
			SquadID: squadID,
		})

		unitEntity.AddComponent(squads.GridPositionComponent, &squads.GridPositionData{
			AnchorRow: tmpl.GridRow,
			AnchorCol: tmpl.GridCol,
			Width:     width,
			Height:    height,
		})

		unitEntity.AddComponent(squads.UnitRoleComponent, &squads.UnitRoleData{
			Role: tmpl.Role,
		})

		unitEntity.AddComponent(squads.TargetRowComponent, &squads.TargetRowData{
			AttackType:  tmpl.AttackType,
			TargetCells: tmpl.TargetCells,
		})

		if tmpl.CoverValue > 0.0 {
			unitEntity.AddComponent(squads.CoverComponent, &squads.CoverData{
				CoverValue:     tmpl.CoverValue,
				CoverRange:     tmpl.CoverRange,
				RequiresActive: tmpl.RequiresActive,
			})
		}

		unitEntity.AddComponent(squads.AttackRangeComponent, &squads.AttackRangeData{
			Range: tmpl.AttackRange,
		})

		unitEntity.AddComponent(squads.MovementSpeedComponent, &squads.MovementSpeedData{
			Speed: tmpl.MovementSpeed,
		})

		unitEntity.AddComponent(squads.ExperienceComponent, &squads.ExperienceData{
			Level:         1,
			CurrentXP:     0,
			XPToNextLevel: 100,
		})

		unitEntity.AddComponent(squads.StatGrowthComponent, &squads.StatGrowthData{
			Strength:   tmpl.StatGrowths.Strength,
			Dexterity:  tmpl.StatGrowths.Dexterity,
			Magic:      tmpl.StatGrowths.Magic,
			Leadership: tmpl.StatGrowths.Leadership,
			Armor:      tmpl.StatGrowths.Armor,
			Weapon:     tmpl.StatGrowths.Weapon,
		})

		if tmpl.IsLeader {
			unitEntity.AddComponent(squads.LeaderComponent, &squads.LeaderData{
				Leadership: 10,
				Experience: 0,
			})
			unitEntity.AddComponent(squads.AbilitySlotComponent, &squads.AbilitySlotData{
				Slots: [4]squads.AbilitySlot{},
			})
			unitEntity.AddComponent(squads.CooldownTrackerComponent, &squads.CooldownTrackerData{
				Cooldowns:    [4]int{0, 0, 0, 0},
				MaxCooldowns: [4]int{0, 0, 0, 0},
			})
		}

		for _, cell := range cellsToOccupy {
			key := fmt.Sprintf("%d,%d", cell[0], cell[1])
			occupied[key] = true
		}
	}

	return squadID
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
	positions := [][2]int{
		{0, 0}, {0, 1}, {0, 2},
		{1, 1},
		{2, 1},
	}

	maxUnits := 5
	if len(squads.Units) < maxUnits {
		maxUnits = len(squads.Units)
	}

	leaderIndex := common.RandomInt(maxUnits)

	var unitsToCreate []squads.UnitTemplate
	for i := 0; i < maxUnits && i < len(positions); i++ {
		unit := squads.Units[i%len(squads.Units)]
		unit.GridRow = positions[i][0]
		unit.GridCol = positions[i][1]

		if i == leaderIndex {
			unit.IsLeader = true
			unit.Attributes.Leadership = 20
		}

		unitsToCreate = append(unitsToCreate, unit)
	}

	return createSimSquad(manager, squadName, squads.FormationBalanced, worldPos, unitsToCreate)
}

// createRangedSquad mirrors initialplayersquads.go:createRangedSquad.
// Filters units by AttackRange >= 3, random selection, 3-5 units.
func createRangedSquad(manager *common.EntityManager, squadName string, worldPos coords.LogicalPosition) ecs.EntityID {
	rangedUnits := filterUnitsByAttackRange(3)
	if len(rangedUnits) == 0 {
		// Fallback to balanced if no ranged units
		return createBalancedSquad(manager, squadName, worldPos)
	}

	unitCount := common.GetRandomBetween(3, 5)

	gridPositions := [][2]int{
		{0, 0}, {1, 1}, {2, 2},
		{0, 2}, {1, 0},
	}

	var unitsToCreate []squads.UnitTemplate
	for i := 0; i < unitCount; i++ {
		randomIdx := common.RandomInt(len(rangedUnits))
		unit := rangedUnits[randomIdx]
		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]
		unit.IsLeader = false
		unitsToCreate = append(unitsToCreate, unit)
	}

	leaderIndex := common.RandomInt(unitCount)
	unitsToCreate[leaderIndex].IsLeader = true
	unitsToCreate[leaderIndex].Attributes.Leadership = 20

	return createSimSquad(manager, squadName, squads.FormationRanged, worldPos, unitsToCreate)
}

// createMagicSquad mirrors initialplayersquads.go:createMagicSquad.
// Filters units by AttackType == Magic, exactly 3 units.
func createMagicSquad(manager *common.EntityManager, squadName string, worldPos coords.LogicalPosition) ecs.EntityID {
	magicUnits := filterUnitsByAttackType(squads.AttackTypeMagic)
	if len(magicUnits) == 0 {
		return createBalancedSquad(manager, squadName, worldPos)
	}

	unitCount := 3

	gridPositions := [][2]int{
		{0, 1}, {1, 0}, {2, 1},
	}

	var unitsToCreate []squads.UnitTemplate
	for i := 0; i < unitCount; i++ {
		randomIdx := common.RandomInt(len(magicUnits))
		unit := magicUnits[randomIdx]
		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]
		unit.IsLeader = false
		unitsToCreate = append(unitsToCreate, unit)
	}

	leaderIndex := common.RandomInt(unitCount)
	unitsToCreate[leaderIndex].IsLeader = true
	unitsToCreate[leaderIndex].Attributes.Leadership = 20

	return createSimSquad(manager, squadName, squads.FormationBalanced, worldPos, unitsToCreate)
}

// createMixedSquad mirrors initialplayersquads.go:createMixedSquad.
// Alternates between ranged and magic units, 4-5 units.
func createMixedSquad(manager *common.EntityManager, squadName string, worldPos coords.LogicalPosition) ecs.EntityID {
	rangedUnits := filterUnitsByAttackRange(3)
	magicUnits := filterUnitsByAttackType(squads.AttackTypeMagic)

	if len(rangedUnits) == 0 || len(magicUnits) == 0 {
		return createBalancedSquad(manager, squadName, worldPos)
	}

	unitCount := common.GetRandomBetween(4, 5)

	gridPositions := [][2]int{
		{0, 0}, {1, 1}, {2, 2},
		{1, 2}, {2, 0},
	}

	var unitsToCreate []squads.UnitTemplate
	for i := 0; i < unitCount; i++ {
		var unit squads.UnitTemplate

		if i%2 == 0 && len(rangedUnits) > 0 {
			randomIdx := common.RandomInt(len(rangedUnits))
			unit = rangedUnits[randomIdx]
		} else if len(magicUnits) > 0 {
			randomIdx := common.RandomInt(len(magicUnits))
			unit = magicUnits[randomIdx]
		} else if len(rangedUnits) > 0 {
			randomIdx := common.RandomInt(len(rangedUnits))
			unit = rangedUnits[randomIdx]
		}

		unit.GridRow = gridPositions[i][0]
		unit.GridCol = gridPositions[i][1]
		unit.IsLeader = false
		unitsToCreate = append(unitsToCreate, unit)
	}

	leaderIndex := common.RandomInt(unitCount)
	unitsToCreate[leaderIndex].IsLeader = true
	unitsToCreate[leaderIndex].Attributes.Leadership = 20

	return createSimSquad(manager, squadName, squads.FormationBalanced, worldPos, unitsToCreate)
}

// createMeleeSquad creates an all-melee squad using MeleeRow attack type units.
func createMeleeSquad(manager *common.EntityManager, squadName string, worldPos coords.LogicalPosition) ecs.EntityID {
	meleeUnits := filterUnitsByAttackType(squads.AttackTypeMeleeRow)
	if len(meleeUnits) == 0 {
		return createBalancedSquad(manager, squadName, worldPos)
	}

	positions := [][2]int{
		{0, 0}, {0, 1}, {0, 2},
		{1, 1},
		{2, 1},
	}

	unitCount := 5
	if len(meleeUnits) < unitCount {
		unitCount = len(meleeUnits)
	}

	leaderIndex := common.RandomInt(unitCount)

	var unitsToCreate []squads.UnitTemplate
	for i := 0; i < unitCount && i < len(positions); i++ {
		randomIdx := common.RandomInt(len(meleeUnits))
		unit := meleeUnits[randomIdx]
		unit.GridRow = positions[i][0]
		unit.GridCol = positions[i][1]

		if i == leaderIndex {
			unit.IsLeader = true
			unit.Attributes.Leadership = 20
		}

		unitsToCreate = append(unitsToCreate, unit)
	}

	return createSimSquad(manager, squadName, squads.FormationOffensive, worldPos, unitsToCreate)
}

// ========================================
// FILTER HELPERS
// Mirrors of the unexported filter functions in initialplayersquads.go.
// ========================================

func filterUnitsByAttackRange(minRange int) []squads.UnitTemplate {
	var filtered []squads.UnitTemplate
	for _, unit := range squads.Units {
		if unit.AttackRange >= minRange {
			filtered = append(filtered, unit)
		}
	}
	return filtered
}

func filterUnitsByAttackType(attackType squads.AttackType) []squads.UnitTemplate {
	var filtered []squads.UnitTemplate
	for _, unit := range squads.Units {
		if unit.AttackType == attackType {
			filtered = append(filtered, unit)
		}
	}
	return filtered
}

func filterUnitsByRole(role squads.UnitRole) []squads.UnitTemplate {
	var filtered []squads.UnitTemplate
	for _, unit := range squads.Units {
		if unit.Role == role {
			filtered = append(filtered, unit)
		}
	}
	return filtered
}
