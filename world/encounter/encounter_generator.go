package encounter

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/evaluation"
	"game_main/tactical/squads"
	"game_main/world/coords"
	"math"

	"github.com/bytearena/ecs"
)

// GenerateEncounterSpec creates an encounter specification without combat dependencies.
// This allows encounter generation to be tested independently of combat setup.
// Use SetupBalancedEncounter for the full setup that includes combat infrastructure.
//
// Returns an EncounterSpec that can be passed to combat.SetupCombatFromEncounter.
func GenerateEncounterSpec(
	manager *common.EntityManager,
	playerEntityID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
	encounterData *OverworldEncounterData,
) (*EncounterSpec, error) {
	// Validate player
	if playerEntityID == 0 {
		return nil, fmt.Errorf("invalid player entity ID")
	}

	// Ensure player has deployed squads
	if err := ensurePlayerSquadsDeployed(playerEntityID, manager); err != nil {
		return nil, fmt.Errorf("failed to deploy player squads: %w", err)
	}

	// Get player's roster
	config := evaluation.GetDefaultConfig()
	roster := squads.GetPlayerSquadRoster(playerEntityID, manager)
	if roster == nil {
		return nil, fmt.Errorf("player has no squad roster")
	}

	deployedSquads := roster.GetDeployedSquads(manager)
	if len(deployedSquads) == 0 {
		return nil, fmt.Errorf("no deployed squads")
	}

	// Calculate average power per squad
	totalPlayerPower := 0.0
	for _, squadID := range deployedSquads {
		squadPower := evaluation.CalculateSquadPower(squadID, manager, config)
		totalPlayerPower += squadPower
	}
	avgPlayerSquadPower := totalPlayerPower / float64(len(deployedSquads))

	// Determine difficulty
	difficultyMod := getEncounterDifficulty(encounterData)
	targetEnemySquadPower := avgPlayerSquadPower * difficultyMod.PowerMultiplier

	// Handle edge cases
	if avgPlayerSquadPower <= 0.0 {
		targetEnemySquadPower = 50.0
		difficultyMod.MinSquads = 1
		difficultyMod.MaxSquads = 1
	}
	if targetEnemySquadPower > 2000.0 {
		targetEnemySquadPower = 2000.0
	}

	fmt.Printf("Generating encounter spec: Avg Power %.2f, Target Power %.2f\n",
		avgPlayerSquadPower, targetEnemySquadPower)

	// Generate enemy squad specifications
	enemySquadSpecs := generateEnemySquadSpecs(
		manager,
		targetEnemySquadPower,
		difficultyMod,
		encounterData,
		playerStartPos,
		config,
	)

	encounterType := ""
	difficulty := 2
	if encounterData != nil {
		encounterType = encounterData.EncounterType
		difficulty = encounterData.Level
	}

	return &EncounterSpec{
		PlayerSquadIDs: deployedSquads,
		EnemySquads:    enemySquadSpecs,
		Difficulty:     difficulty,
		EncounterType:  encounterType,
		PlayerStartPos: playerStartPos,
	}, nil
}

// generateEnemySquadSpecs creates enemy squad specifications without combat dependencies.
func generateEnemySquadSpecs(
	manager *common.EntityManager,
	targetSquadPower float64,
	difficultyMod EncounterDifficultyModifier,
	encounterData *OverworldEncounterData,
	playerPos coords.LogicalPosition,
	config *evaluation.PowerConfig,
) []EnemySquadSpec {
	squadCount := common.GetRandomBetween(difficultyMod.MinSquads, difficultyMod.MaxSquads)
	squadTypes := getSquadComposition(encounterData, squadCount)

	specs := make([]EnemySquadSpec, 0, squadCount)

	for i := 0; i < squadCount; i++ {
		pos := generateEnemyPositionSpec(playerPos, i, squadCount)

		// Create the actual squad
		squadID := createSquadForPowerBudget(
			manager,
			targetSquadPower,
			squadTypes[i],
			fmt.Sprintf("Enemy Squad %d", i+1),
			pos,
			config,
		)

		if squadID != 0 {
			specs = append(specs, EnemySquadSpec{
				SquadID:  squadID,
				Position: pos,
				Power:    targetSquadPower,
				Type:     squadTypes[i],
				Name:     fmt.Sprintf("Enemy Squad %d", i+1),
			})
		}
	}

	return specs
}

// generateEnemyPositionSpec creates position for an enemy squad.
func generateEnemyPositionSpec(playerPos coords.LogicalPosition, index, total int) coords.LogicalPosition {
	angle := (float64(index) / float64(total)) * 2.0 * math.Pi
	distance := 10

	offsetX := int(math.Round(float64(distance) * math.Cos(angle)))
	offsetY := int(math.Round(float64(distance) * math.Sin(angle)))

	x := clampPosition(playerPos.X+offsetX, 0, 99)
	y := clampPosition(playerPos.Y+offsetY, 0, 79)

	return coords.LogicalPosition{X: x, Y: y}
}
