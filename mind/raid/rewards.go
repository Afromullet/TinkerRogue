package raid

import (
	"game_main/common"
	"game_main/mind/combatpipeline"
	"game_main/world/worldmap"
)

// calculateRoomReward returns the reward and target for a cleared room.
// Does NOT grant rewards — the pipeline does that via ExecuteResolution.
func calculateRoomReward(manager *common.EntityManager, raidState *RaidStateData, roomType string) (combatpipeline.Reward, combatpipeline.GrantTarget) {
	if RaidConfig == nil || raidState == nil {
		return combatpipeline.Reward{}, combatpipeline.GrantTarget{}
	}

	// Base target for all combat rooms (gold + XP)
	target := combatpipeline.GrantTarget{
		PlayerEntityID: raidState.PlayerEntityID,
		SquadIDs:       raidState.PlayerSquadIDs,
		CommanderID:    raidState.CommanderID,
	}

	// Floor scaling: 1.0 + (floor-1) * scalePercent/100
	scale := 1.0 + float64(raidState.CurrentFloor-1)*float64(RaidConfig.Rewards.FloorScalePercent)/100.0
	gold := int(float64(RaidConfig.Rewards.BaseGoldPerRoom) * scale)
	xp := int(float64(RaidConfig.Rewards.BaseXPPerRoom) * scale)

	reward := combatpipeline.Reward{Gold: gold, Experience: xp}

	// Command posts also restore mana
	if roomType == worldmap.GarrisonRoomCommandPost {
		reward.Mana = RaidConfig.Rewards.CommandPostManaRestore
	}

	return reward, target
}
