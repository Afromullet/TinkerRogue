package raid

import (
	"fmt"
	"sort"

	"game_main/common"
	"game_main/mind/evaluation"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// SetDeployment validates and sets which player squads are deployed vs. reserved.
func SetDeployment(manager *common.EntityManager, deployedIDs, reserveIDs []ecs.EntityID) error {
	maxDeployed := MaxDeployedPerEncounter()

	if len(deployedIDs) == 0 {
		return fmt.Errorf("at least 1 squad must be deployed")
	}

	if len(deployedIDs) > maxDeployed {
		return fmt.Errorf("cannot deploy more than %d squads (got %d)", maxDeployed, len(deployedIDs))
	}

	// Validate no destroyed squads are deployed
	for _, id := range deployedIDs {
		if squads.IsSquadDestroyed(id, manager) {
			return fmt.Errorf("cannot deploy destroyed squad %d", id)
		}
	}

	// Find or create deployment entity
	var deployEntity *ecs.Entity
	for _, result := range manager.World.Query(DeploymentTag) {
		deployEntity = result.Entity
		break
	}

	if deployEntity == nil {
		deployEntity = manager.World.NewEntity()
	}

	deployEntity.AddComponent(DeploymentComponent, &DeploymentData{
		DeployedSquadIDs: append([]ecs.EntityID{}, deployedIDs...),
		ReserveSquadIDs:  append([]ecs.EntityID{}, reserveIDs...),
	})

	return nil
}

// AutoDeploy automatically selects the strongest available squads up to the deploy limit.
// Squads are ranked by power score from evaluation.CalculateSquadPower.
func AutoDeploy(manager *common.EntityManager) (*DeploymentData, error) {
	raidState := GetRaidState(manager)
	if raidState == nil {
		return nil, fmt.Errorf("no active raid")
	}

	maxDeployed := MaxDeployedPerEncounter()

	// Collect living squads and sort by power (descending)
	type squadScore struct {
		ID    ecs.EntityID
		Power float64
	}

	config := evaluation.GetPowerConfigByProfile("balanced")

	var scored []squadScore
	for _, squadID := range raidState.PlayerSquadIDs {
		if squads.IsSquadDestroyed(squadID, manager) {
			continue
		}
		power := evaluation.CalculateSquadPower(squadID, manager, config)
		scored = append(scored, squadScore{ID: squadID, Power: power})
	}

	// Sort by highest power first
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Power > scored[j].Power
	})

	var deployed, reserve []ecs.EntityID
	for i, s := range scored {
		if i < maxDeployed {
			deployed = append(deployed, s.ID)
		} else {
			reserve = append(reserve, s.ID)
		}
	}

	if len(deployed) == 0 {
		return nil, fmt.Errorf("no living squads available")
	}

	if err := SetDeployment(manager, deployed, reserve); err != nil {
		return nil, err
	}

	return GetDeployment(manager), nil
}
