package raid

import (
	"fmt"

	"game_main/common"
	"game_main/mind/resolution"

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

		resolution.ApplyHPRecovery(manager, squadID, hpPercent)
	}

	fmt.Printf("Recovery: Post-encounter recovery applied (deployed: %d%%, reserve: %d%%)\n",
		RaidConfig.Recovery.DeployedHPPercent, RaidConfig.Recovery.ReserveHPPercent)
}

// ApplyBetweenFloorRecovery applies HP recovery when advancing floors.
// All squads get DeployedHPPercent HP.
func ApplyBetweenFloorRecovery(manager *common.EntityManager, raidState *RaidStateData) {
	if RaidConfig == nil || raidState == nil {
		return
	}

	for _, squadID := range raidState.PlayerSquadIDs {
		resolution.ApplyHPRecovery(manager, squadID, RaidConfig.Recovery.DeployedHPPercent)
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
