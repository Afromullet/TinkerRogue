package raid

import (
	"fmt"

	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// ApplyPostEncounterRecovery applies differentiated HP recovery after an encounter:
// deployed squads get DeployedHPPercent, reserve squads get ReserveHPPercent.
func ApplyPostEncounterRecovery(manager *common.EntityManager, raidState *RaidStateData) {
	if RaidConfig == nil || raidState == nil {
		return
	}

	deployment := GetDeployment(manager)

	for _, squadID := range raidState.PlayerSquadIDs {
		hpPercent := RaidConfig.Recovery.DeployedHPPercent

		// Reserve squads get better recovery
		if deployment != nil && isInReserve(deployment, squadID) {
			hpPercent = RaidConfig.Recovery.ReserveHPPercent
		}

		applyHPRecovery(manager, squadID, hpPercent)
	}

	fmt.Printf("Recovery: Post-encounter recovery applied (deployed: %d%%, reserve: %d%%)\n",
		RaidConfig.Recovery.DeployedHPPercent, RaidConfig.Recovery.ReserveHPPercent)
}

// ApplyBetweenFloorRecovery applies HP recovery and morale bonus when advancing floors.
// All squads get DeployedHPPercent HP and BetweenFloorMoraleBonus morale.
func ApplyBetweenFloorRecovery(manager *common.EntityManager, raidState *RaidStateData) {
	if RaidConfig == nil || raidState == nil {
		return
	}

	for _, squadID := range raidState.PlayerSquadIDs {
		applyHPRecovery(manager, squadID, RaidConfig.Recovery.DeployedHPPercent)
		applyMoraleBonus(manager, squadID, RaidConfig.Recovery.BetweenFloorMoraleBonus)
	}

	fmt.Printf("Recovery: Between-floor recovery applied to %d squads\n", len(raidState.PlayerSquadIDs))
}

// isInReserve checks if a squad ID is in the reserve list of the current deployment.
func isInReserve(deployment *DeploymentData, squadID ecs.EntityID) bool {
	for _, id := range deployment.ReserveSquadIDs {
		if id == squadID {
			return true
		}
	}
	return false
}

// applyHPRecovery restores a percentage of max HP to all living units in a squad.
func applyHPRecovery(manager *common.EntityManager, squadID ecs.EntityID, hpPercent int) {
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	for _, unitID := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, unitID, common.AttributeComponent)
		if attr != nil && attr.CurrentHealth > 0 {
			heal := attr.MaxHealth * hpPercent / 100
			attr.CurrentHealth += heal
			if attr.CurrentHealth > attr.MaxHealth {
				attr.CurrentHealth = attr.MaxHealth
			}
		}
	}
}

// applyMoraleBonus adds morale to a squad, capped at maxMorale.
func applyMoraleBonus(manager *common.EntityManager, squadID ecs.EntityID, bonus int) {
	squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
	if squadData != nil {
		squadData.Morale += bonus
		if squadData.Morale > maxMorale {
			squadData.Morale = maxMorale
		}
	}
}

// applyMoralePenalty reduces squad morale, floored at 0.
func applyMoralePenalty(manager *common.EntityManager, squadID ecs.EntityID, penalty int) {
	squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
	if squadData != nil {
		squadData.Morale -= penalty
		if squadData.Morale < 0 {
			squadData.Morale = 0
		}
	}
}
