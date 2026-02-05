package encounter

import (
	"fmt"
	"game_main/common"
	"game_main/mind/evaluation"
	"game_main/overworld/core"
	"game_main/tactical/squads"
	"game_main/world/coords"

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
	encounterData *core.OverworldEncounterData,
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
	config := evaluation.GetPowerConfigByProfile(DefaultPowerProfile)
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
	level := 3 // Default level
	if encounterData != nil {
		level = encounterData.Level
	}
	difficultyMod := GetDifficultyModifier(level)
	targetEnemySquadPower := avgPlayerSquadPower * difficultyMod.PowerMultiplier

	// Handle edge cases using difficulty-specific power bounds
	if avgPlayerSquadPower <= 0.0 {
		targetEnemySquadPower = difficultyMod.MinTargetPower
		difficultyMod.SquadCount = 1
	}
	if targetEnemySquadPower > difficultyMod.MaxTargetPower {
		targetEnemySquadPower = difficultyMod.MaxTargetPower
	}

	fmt.Printf("Generating encounter spec: Avg Power %.2f, Target Power %.2f\n",
		avgPlayerSquadPower, targetEnemySquadPower)

	// Generate enemy squad specifications using shared function
	enemySquadSpecs := generateEnemySquadsByPower(
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

// Note: generateEnemySquadsByPower and generateEnemyPosition are defined in encounter_setup.go
// and shared by both GenerateEncounterSpec and SetupBalancedEncounter
