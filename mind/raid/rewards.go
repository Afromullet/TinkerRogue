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

	switch roomType {
	case worldmap.GarrisonRoomCommandPost:
		return commandPostReward(raidState)
	}

	return combatpipeline.Reward{}, combatpipeline.GrantTarget{}
}

// commandPostReward returns the mana reward for clearing a command post.
func commandPostReward(raidState *RaidStateData) (combatpipeline.Reward, combatpipeline.GrantTarget) {
	manaRestore := RaidConfig.Rewards.CommandPostManaRestore
	if manaRestore <= 0 {
		return combatpipeline.Reward{}, combatpipeline.GrantTarget{}
	}

	return combatpipeline.Reward{Mana: manaRestore}, combatpipeline.GrantTarget{CommanderID: raidState.CommanderID}
}
