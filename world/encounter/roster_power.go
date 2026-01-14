package encounter

import (
	"game_main/common"
	"game_main/tactical/squads"
	"github.com/bytearena/ecs"
)

// CalculateRosterPower computes the total power for a player's squad roster
// Weighs deployed squads higher than reserves
func CalculateRosterPower(
	playerID ecs.EntityID,
	manager *common.EntityManager,
	config *EvaluationConfigData,
) float64 {
	roster := squads.GetPlayerSquadRoster(playerID, manager)
	if roster == nil {
		return 0.0
	}

	totalPower := 0.0

	for _, squadID := range roster.OwnedSquads {
		squadPower := CalculateSquadPower(squadID, manager, config)

		// Apply deployment weight
		squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
		if squadData != nil {
			totalPower += applyDeploymentWeight(squadPower, squadData.IsDeployed, config)
		}
	}

	return totalPower
}

// CalculateDeployedSquadsPower computes power for only deployed squads
// Useful for combat encounter balancing
func CalculateDeployedSquadsPower(
	playerID ecs.EntityID,
	manager *common.EntityManager,
	config *EvaluationConfigData,
) float64 {
	roster := squads.GetPlayerSquadRoster(playerID, manager)
	if roster == nil {
		return 0.0
	}

	deployedSquads := roster.GetDeployedSquads(manager)

	totalPower := 0.0
	for _, squadID := range deployedSquads {
		squadPower := CalculateSquadPower(squadID, manager, config)
		totalPower += squadPower
	}

	return totalPower
}

// PowerBreakdown provides detailed power breakdown for analysis
type PowerBreakdown struct {
	TotalPower     float64                  // Combined roster power
	DeployedPower  float64                  // Power from deployed squads
	ReservePower   float64                  // Power from reserve squads
	SquadBreakdown map[ecs.EntityID]float64 // SquadID → power
}

// CalculateRosterPowerBreakdown returns detailed power analysis
func CalculateRosterPowerBreakdown(
	playerID ecs.EntityID,
	manager *common.EntityManager,
	config *EvaluationConfigData,
) PowerBreakdown {
	roster := squads.GetPlayerSquadRoster(playerID, manager)
	if roster == nil {
		return PowerBreakdown{}
	}

	breakdown := PowerBreakdown{
		SquadBreakdown: make(map[ecs.EntityID]float64),
	}

	for _, squadID := range roster.OwnedSquads {
		squadPower := CalculateSquadPower(squadID, manager, config)
		breakdown.SquadBreakdown[squadID] = squadPower

		squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
		if squadData != nil {
			weightedPower := applyDeploymentWeight(squadPower, squadData.IsDeployed, config)
			if squadData.IsDeployed {
				breakdown.DeployedPower += weightedPower
			} else {
				breakdown.ReservePower += weightedPower
			}
		}
	}

	breakdown.TotalPower = breakdown.DeployedPower + breakdown.ReservePower

	return breakdown
}
