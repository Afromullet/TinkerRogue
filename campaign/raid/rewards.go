package raid

import (
	"game_main/common"
	"game_main/mind/combatlifecycle"
	"game_main/world/garrisongen"
)

// calculateRoomReward returns the reward and target for a cleared room.
// Does NOT grant rewards — the pipeline does that via ExecuteResolution.
func calculateRoomReward(manager *common.EntityManager, raidState *RaidStateData, roomType string) (combatlifecycle.Reward, combatlifecycle.GrantTarget) {
	if RaidConfig == nil || raidState == nil {
		return combatlifecycle.Reward{}, combatlifecycle.GrantTarget{}
	}

	// Base target for all combat rooms (gold + XP + mana)
	target := combatlifecycle.GrantTarget{
		PlayerEntityID: raidState.PlayerEntityID,
		SquadIDs:       raidState.PlayerSquadIDs,
	}

	// Floor scaling: 1.0 + (floor-1) * scalePercent/100
	scale := 1.0 + float64(raidState.CurrentFloor-1)*float64(RaidConfig.Rewards.FloorScalePercent)/100.0
	gold := int(float64(RaidConfig.Rewards.BaseGoldPerRoom) * scale)
	xp := int(float64(RaidConfig.Rewards.BaseXPPerRoom) * scale)
	arcana := int(float64(raidArcanaBase()) * scale)
	skill := int(float64(raidSkillBase()) * scale)

	reward := combatlifecycle.Reward{
		Gold:       gold,
		Experience: xp,
		ArcanaPts:  arcana,
		SkillPts:   skill,
	}

	// Command posts also restore mana
	if roomType == garrisongen.GarrisonRoomCommandPost {
		reward.Mana = RaidConfig.Rewards.CommandPostManaRestore
	}

	return reward, target
}

// raidArcanaBase returns the configured base Arcana per room, defaulting to 1.
func raidArcanaBase() int {
	if RaidConfig != nil && RaidConfig.Rewards.BaseArcanaPerRoom > 0 {
		return RaidConfig.Rewards.BaseArcanaPerRoom
	}
	return 1
}

// raidSkillBase returns the configured base Skill per room, defaulting to 1.
func raidSkillBase() int {
	if RaidConfig != nil && RaidConfig.Rewards.BaseSkillPerRoom > 0 {
		return RaidConfig.Rewards.BaseSkillPerRoom
	}
	return 1
}
