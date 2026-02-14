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
	rosterOwnerID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
	encounterData *core.OverworldEncounterData,
) (*EncounterSpec, error) {
	// Validate player
	if rosterOwnerID == 0 {
		return nil, fmt.Errorf("invalid roster owner entity ID")
	}

	// Ensure roster owner has deployed squads
	if err := ensurePlayerSquadsDeployed(rosterOwnerID, manager); err != nil {
		return nil, fmt.Errorf("failed to deploy player squads: %w", err)
	}

	// Get player's roster
	config := evaluation.GetPowerConfigByProfile(DefaultPowerProfile)
	roster := squads.GetPlayerSquadRoster(rosterOwnerID, manager)
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
	difficultyMod := getDifficultyModifier(level)
	targetEnemySquadPower := avgPlayerSquadPower * difficultyMod.PowerMultiplier

	// Handle edge cases using difficulty-specific power bounds
	if avgPlayerSquadPower <= 0.0 {
		targetEnemySquadPower = difficultyMod.MinTargetPower
		difficultyMod.SquadCount = 1
	}
	if targetEnemySquadPower > difficultyMod.MaxTargetPower {
		targetEnemySquadPower = difficultyMod.MaxTargetPower
	}

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
