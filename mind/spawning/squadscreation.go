package spawning

import (
	"fmt"
	"math"

	"game_main/core/common"
	"game_main/mind/evaluation"
	"game_main/campaign/overworld/core"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/templates"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// generateEnemySquadsByPower creates enemy squads matching the per-squad target power.
// Returns enemySquadSpec entries with type, name, position, and assigned power.
func generateEnemySquadsByPower(
	manager *common.EntityManager,
	targetSquadPower float64,
	difficultyMod templates.JSONEncounterDifficulty,
	encounterData *core.OverworldEncounterData,
	playerPos coords.LogicalPosition,
	config *evaluation.PowerConfig,
) []enemySquadSpec {
	squadCount := difficultyMod.SquadCount

	enemySquads := []enemySquadSpec{}

	squadTypes := getSquadComposition(encounterData, squadCount)

	enemyPositions := GeneratePositionsAroundPoint(playerPos, squadCount, 0, 2*math.Pi, EnemySpacingDistance, EnemySpacingDistance)

	for i := 0; i < squadCount; i++ {
		pos := enemyPositions[i]
		squadName := fmt.Sprintf("Enemy Squad %d", i+1)

		squadID := createSquadForPowerBudget(
			manager,
			targetSquadPower,
			squadTypes[i],
			squadName,
			pos,
			config,
			difficultyMod,
		)

		if squadID != 0 {
			enemySquads = append(enemySquads, enemySquadSpec{
				SquadID:  squadID,
				Position: pos,
				Power:    targetSquadPower,
				Type:     squadTypes[i],
				Name:     squadName,
			})
		}
	}

	return enemySquads
}

// createSquadForPowerBudget creates a squad matching target power using the shared
// evaluation package for per-unit power estimation.
func createSquadForPowerBudget(
	manager *common.EntityManager,
	targetPower float64,
	squadType string,
	name string,
	position coords.LogicalPosition,
	config *evaluation.PowerConfig,
	difficultyMod templates.JSONEncounterDifficulty,
) ecs.EntityID {
	unitPool := filterUnitsBySquadType(squadType)
	if len(unitPool) == 0 {
		unitPool = unitdefs.Units
	}

	if len(unitPool) == 0 {
		return 0
	}

	unitsToCreate := []unitdefs.UnitTemplate{}
	currentPower := 0.0
	// Safe grid positions for 2-wide units (avoid rightmost column); extended for Boss difficulty.
	gridPositions := [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 0}, {2, 1}, {3, 0}, {3, 1}}

	maxUnits := difficultyMod.MaxUnitsPerSquad
	if maxUnits > len(gridPositions) {
		maxUnits = len(gridPositions)
	}

	for currentPower < targetPower && len(unitsToCreate) < maxUnits {
		unit := unitPool[common.RandomInt(len(unitPool))]

		unitPower := evaluation.EstimateUnitPowerFromTemplate(unit, config)

		unit.GridRow = gridPositions[len(unitsToCreate)][0]
		unit.GridCol = gridPositions[len(unitsToCreate)][1]
		unit.IsLeader = (len(unitsToCreate) == 0)

		unitsToCreate = append(unitsToCreate, unit)
		currentPower += unitPower

		if currentPower >= targetPower*PowerThreshold {
			break
		}
	}

	minUnits := difficultyMod.MinUnitsPerSquad
	if minUnits > len(gridPositions) {
		minUnits = len(gridPositions)
	}
	for len(unitsToCreate) < minUnits && len(unitPool) > 0 {
		unit := unitPool[common.RandomInt(len(unitPool))]
		unit.GridRow = gridPositions[len(unitsToCreate)][0]
		unit.GridCol = gridPositions[len(unitsToCreate)][1]
		unitsToCreate = append(unitsToCreate, unit)
	}

	if len(unitsToCreate) > 0 {
		unitsToCreate[0].Attributes.Leadership = LeadershipAttributeBase
	}

	return squadcore.CreateSquadFromTemplate(
		manager,
		name,
		squadcore.FormationBalanced,
		position,
		unitsToCreate,
	)
}

// filterUnitsBySquadType selects units matching a squad archetype.
func filterUnitsBySquadType(squadType string) []unitdefs.UnitTemplate {
	switch squadType {
	case SquadTypeMelee:
		return unitdefs.FilterByMaxAttackRange(2)
	case SquadTypeRanged:
		return unitdefs.FilterByAttackRange(3)
	case SquadTypeMagic:
		return unitdefs.FilterByAttackType(unitdefs.AttackTypeMagic)
	case SquadTypeSupport:
		return unitdefs.FilterByAttackType(unitdefs.AttackTypeHeal)
	default:
		return unitdefs.Units
	}
}

// GenerateAttackerSquads creates attacker squad entities using a power budget derived from
// sourceSquadIDs. centerPos is the spawn center for the attacker arc. Returns parallel
// slices of squad IDs and spawn positions.
func GenerateAttackerSquads(
	manager *common.EntityManager,
	centerPos coords.LogicalPosition,
	sourceSquadIDs []ecs.EntityID,
	encounterData *core.OverworldEncounterData,
) ([]ecs.EntityID, []coords.LogicalPosition, error) {
	if len(sourceSquadIDs) == 0 {
		return nil, nil, fmt.Errorf("no source squads to derive power budget")
	}

	config := evaluation.GetPowerConfigByProfile(DefaultPowerProfile)
	level := 3
	if encounterData != nil {
		level = encounterData.Level
	}
	difficultyMod := getDifficultyModifier(level)
	targetPower := calculateTargetPower(manager, sourceSquadIDs, config, difficultyMod)

	// Fall back to a single squad if the target power is below the difficulty floor.
	if targetPower <= difficultyMod.MinTargetPower {
		difficultyMod.SquadCount = 1
	}

	specs := generateEnemySquadsByPower(manager, targetPower, difficultyMod, encounterData, centerPos, config)

	ids := make([]ecs.EntityID, len(specs))
	positions := make([]coords.LogicalPosition, len(specs))
	for i, s := range specs {
		ids[i] = s.SquadID
		positions[i] = s.Position
	}
	return ids, positions, nil
}
