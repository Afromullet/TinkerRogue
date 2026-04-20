package encounter

import (
	"fmt"
	"math"

	"game_main/campaign/overworld/core"
	"game_main/core/common"
	"game_main/mind/combatlifecycle"
	"game_main/mind/spawning"
	"game_main/campaign/overworld/garrison"
	rstr "game_main/tactical/squads/roster"
	"game_main/tactical/squads/squadcore"
	"game_main/core/coords"

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
		enemyPositions = spawning.GeneratePositionsAroundPoint(playerStartPos, len(enemyIDs), 0, 2*math.Pi, spawning.EnemySpacingDistance, spawning.EnemySpacingDistance)
	} else {
		enemyIDs, enemyPositions, err = spawning.GenerateAttackerSquads(manager, playerStartPos, deployedSquads, encounterData)
		if err != nil {
			return nil, fmt.Errorf("failed to generate enemies: %w", err)
		}
	}

	playerPositions := spawning.GeneratePlayerSquadPositions(playerStartPos, len(deployedSquads))

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
