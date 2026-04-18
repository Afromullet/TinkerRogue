package encounter

import (
	"fmt"
	"math"

	"game_main/common"
	"game_main/mind/combatlifecycle"
	"game_main/mind/evaluation"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	rstr "game_main/tactical/squads/roster"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/templates"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// getGarrisonForEncounter returns garrison data if this encounter targets a garrisoned node.
// Returns nil if there's no garrison or no threat node.
func getGarrisonForEncounter(manager *common.EntityManager, encounterData *core.OverworldEncounterData) *core.GarrisonData {
	if encounterData.ThreatNodeID == 0 {
		return nil
	}
	garrisonData := garrison.GetGarrisonAtNode(manager, encounterData.ThreatNodeID)
	if garrisonData == nil || len(garrisonData.SquadIDs) == 0 {
		return nil
	}
	return garrisonData
}

// SpawnCombatEntities creates player and enemy factions for an overworld encounter.
// If the target node has a garrison, those squads become the enemies; otherwise new
// enemies are generated via power budget against the player's deployed squads.
func SpawnCombatEntities(
	manager *common.EntityManager,
	rosterOwnerID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
	encounterData *core.OverworldEncounterData,
	encounterID ecs.EntityID,
) (*SpawnResult, error) {
	if rosterOwnerID == 0 {
		return nil, fmt.Errorf("invalid roster owner entity ID")
	}

	deployedSquads, err := ensurePlayerSquadsDeployed(rosterOwnerID, manager)
	if err != nil {
		return nil, err
	}

	enemyFactionName := "Enemy Forces"
	var enemyIDs []ecs.EntityID
	var enemyPositions []coords.LogicalPosition

	if garrisonData := getGarrisonForEncounter(manager, encounterData); garrisonData != nil {
		enemyFactionName = "Garrison Forces"
		enemyIDs = garrisonData.SquadIDs
		enemyPositions = generatePositionsAroundPoint(playerStartPos, len(enemyIDs), 0, 2*math.Pi, EnemySpacingDistance, EnemySpacingDistance)
	} else {
		enemyIDs, enemyPositions, err = generateAttackerSquads(manager, playerStartPos, deployedSquads, encounterData)
		if err != nil {
			return nil, fmt.Errorf("failed to generate enemies: %w", err)
		}
	}

	playerPositions := generatePlayerSquadPositions(playerStartPos, len(deployedSquads))

	return assembleCombatFactions(
		manager, encounterID,
		"Player Forces", enemyFactionName,
		deployedSquads, enemyIDs,
		playerPositions, enemyPositions,
		false,
	)
}

// ensurePlayerSquadsDeployed returns the player's deployed squad IDs, auto-deploying
// all owned squads if none are currently deployed.
func ensurePlayerSquadsDeployed(rosterOwnerID ecs.EntityID, manager *common.EntityManager) ([]ecs.EntityID, error) {
	roster := rstr.GetPlayerSquadRoster(rosterOwnerID, manager)
	if roster == nil {
		return nil, fmt.Errorf("player has no squad roster")
	}

	deployed := roster.GetDeployedSquads(manager)
	if len(deployed) == 0 {
		for _, squadID := range roster.OwnedSquads {
			squadData := common.GetComponentTypeByID[*squadcore.SquadData](manager, squadID, squadcore.SquadComponent)
			if squadData != nil {
				squadData.IsDeployed = true
			}
		}
		deployed = roster.GetDeployedSquads(manager)
	}

	if len(deployed) == 0 {
		return nil, fmt.Errorf("no deployed squads available")
	}
	return deployed, nil
}

// generateAttackerSquads creates attacker squad entities using a power budget derived from
// sourceSquadIDs. centerPos is the spawn center for the attacker arc.
func generateAttackerSquads(
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

// assembleCombatFactions creates the player+enemy faction pair and enrolls the provided
// squads at the given positions. markPlayerDeployed controls the IsDeployed flag on the
// player-side squads (true for garrison defenders, false for already-deployed rosters).
func assembleCombatFactions(
	manager *common.EntityManager,
	encounterID ecs.EntityID,
	playerFactionName, enemyFactionName string,
	playerSquadIDs, enemySquadIDs []ecs.EntityID,
	playerPositions, enemyPositions []coords.LogicalPosition,
	markPlayerDeployed bool,
) (*SpawnResult, error) {
	fm, playerFactionID, enemyFactionID := combatlifecycle.CreateFactionPair(manager, playerFactionName, enemyFactionName, encounterID)

	if err := combatlifecycle.EnrollSquadsAtPositions(fm, manager, playerFactionID, playerSquadIDs, playerPositions, markPlayerDeployed); err != nil {
		return nil, fmt.Errorf("failed to add player squads: %w", err)
	}
	if err := combatlifecycle.EnrollSquadsAtPositions(fm, manager, enemyFactionID, enemySquadIDs, enemyPositions, false); err != nil {
		return nil, fmt.Errorf("failed to add enemy squads: %w", err)
	}

	return &SpawnResult{
		EnemySquadIDs:   enemySquadIDs,
		PlayerFactionID: playerFactionID,
		EnemyFactionID:  enemyFactionID,
	}, nil
}

// generatePositionsAroundPoint creates positions distributed around a center point.
// arcStart/arcEnd define the angle range in radians (0 to 2*Pi for full circle).
// minDistance/maxDistance define the radius range from center.
func generatePositionsAroundPoint(
	center coords.LogicalPosition,
	count int,
	arcStart, arcEnd float64,
	minDistance, maxDistance int,
) []coords.LogicalPosition {
	positions := make([]coords.LogicalPosition, count)
	arcRange := arcEnd - arcStart
	mapWidth := coords.CoordManager.GetDungeonWidth()
	mapHeight := coords.CoordManager.GetDungeonHeight()

	for i := 0; i < count; i++ {
		angle := arcStart + (float64(i)/float64(count))*arcRange
		// Alternate between min and max distance for variety
		distance := minDistance + (i % (maxDistance - minDistance + 1))

		offsetX := int(math.Round(float64(distance) * math.Cos(angle)))
		offsetY := int(math.Round(float64(distance) * math.Sin(angle)))

		pos := coords.LogicalPosition{
			X: clampPosition(center.X+offsetX, 0, mapWidth-1),
			Y: clampPosition(center.Y+offsetY, 0, mapHeight-1),
		}
		positions[i] = pos
	}
	return positions
}

// generatePlayerSquadPositions creates positions for player squads around starting point
func generatePlayerSquadPositions(startPos coords.LogicalPosition, count int) []coords.LogicalPosition {
	// Player squads: arc from -Pi/2 to Pi/2 (facing forward), distance alternating 3-4
	return generatePositionsAroundPoint(startPos, count, -math.Pi/2, math.Pi/2, PlayerMinDistance, PlayerMaxDistance)
}

// generateEnemySquadsByPower creates enemy squads matching target squad power.
// targetPower is now the per-squad target (average player squad power * difficulty).
// Returns EnemySquadSpec which includes Type and Name for full encounter specification.
func generateEnemySquadsByPower(
	manager *common.EntityManager,
	targetSquadPower float64,
	difficultyMod templates.JSONEncounterDifficulty,
	encounterData *core.OverworldEncounterData,
	playerPos coords.LogicalPosition,
	config *evaluation.PowerConfig,
) []EnemySquadSpec {
	// Use fixed squad count from difficulty modifier
	squadCount := difficultyMod.SquadCount

	enemySquads := []EnemySquadSpec{}

	// Get squad composition preferences
	squadTypes := getSquadComposition(encounterData, squadCount)

	// Pre-compute all enemy positions once (avoids N² allocation)
	enemyPositions := generatePositionsAroundPoint(playerPos, squadCount, 0, 2*math.Pi, EnemySpacingDistance, EnemySpacingDistance)

	for i := 0; i < squadCount; i++ {
		pos := enemyPositions[i]
		squadName := fmt.Sprintf("Enemy Squad %d", i+1)

		// Each enemy squad targets the same power (average player squad power * difficulty)
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
			enemySquads = append(enemySquads, EnemySquadSpec{
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

// getSquadComposition returns squad type distribution based on encounter type
func getSquadComposition(encounterData *core.OverworldEncounterData, count int) []string {
	if encounterData == nil || encounterData.EncounterType == "" {
		// Random balanced composition
		return generateRandomComposition(count)
	}

	// Use encounter preferences from JSON configuration
	preferences := GetSquadPreferences(encounterData.EncounterType)
	if len(preferences) == 0 {
		return generateRandomComposition(count)
	}

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = preferences[i%len(preferences)]
	}

	return result
}

// generateRandomComposition creates a random mix of squad types
func generateRandomComposition(count int) []string {
	types := []string{SquadTypeMelee, SquadTypeRanged, SquadTypeMagic}
	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = types[common.RandomInt(len(types))]
	}
	return result
}

// createSquadForPowerBudget creates a squad matching target power.
// Uses the shared evaluation package for power estimation.
func createSquadForPowerBudget(
	manager *common.EntityManager,
	targetPower float64,
	squadType string,
	name string,
	position coords.LogicalPosition,
	config *evaluation.PowerConfig,
	difficultyMod templates.JSONEncounterDifficulty,
) ecs.EntityID {
	// Select unit pool based on squad type
	unitPool := filterUnitsBySquadType(squadType)
	if len(unitPool) == 0 {
		unitPool = unitdefs.Units // Fallback to all units
	}

	if len(unitPool) == 0 {
		return 0 // No units available
	}

	// Iteratively add units until power budget reached
	unitsToCreate := []unitdefs.UnitTemplate{}
	currentPower := 0.0
	// Use safe grid positions that work for 2-wide units (avoid rightmost column)
	// Extended pattern to support up to 8 units for Boss difficulty
	gridPositions := [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 0}, {2, 1}, {3, 0}, {3, 1}}

	maxUnits := difficultyMod.MaxUnitsPerSquad
	if maxUnits > len(gridPositions) {
		maxUnits = len(gridPositions)
	}

	for currentPower < targetPower && len(unitsToCreate) < maxUnits {
		// Pick random unit from pool
		unit := unitPool[common.RandomInt(len(unitPool))]

		// Estimate unit power contribution using shared function
		unitPower := evaluation.EstimateUnitPowerFromTemplate(unit, config)

		// Set grid position
		unit.GridRow = gridPositions[len(unitsToCreate)][0]
		unit.GridCol = gridPositions[len(unitsToCreate)][1]
		unit.IsLeader = (len(unitsToCreate) == 0) // First unit is leader

		unitsToCreate = append(unitsToCreate, unit)
		currentPower += unitPower

		// Stop if we've reached the power threshold
		if currentPower >= targetPower*PowerThreshold {
			break
		}
	}

	// Ensure minimum units based on difficulty
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

	// Set leader attributes
	if len(unitsToCreate) > 0 {
		unitsToCreate[0].Attributes.Leadership = LeadershipAttributeBase
	}

	// Create squad
	squadID := squadcore.CreateSquadFromTemplate(
		manager,
		name,
		squadcore.FormationBalanced,
		position,
		unitsToCreate,
	)

	return squadID
}

// filterUnitsBySquadType selects units matching squad archetype
func filterUnitsBySquadType(squadType string) []unitdefs.UnitTemplate {
	switch squadType {
	case SquadTypeMelee:
		return unitdefs.FilterByMaxAttackRange(2) // Melee: range <= 2
	case SquadTypeRanged:
		return unitdefs.FilterByAttackRange(3) // Ranged: range >= 3
	case SquadTypeMagic:
		return unitdefs.FilterByAttackType(unitdefs.AttackTypeMagic)
	case SquadTypeSupport:
		return unitdefs.FilterByAttackType(unitdefs.AttackTypeHeal)
	default:
		return unitdefs.Units
	}
}

// clampPosition ensures a position stays within bounds
func clampPosition(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
