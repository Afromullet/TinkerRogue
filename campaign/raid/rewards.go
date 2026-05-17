package raid

import (
	"game_main/core/common"
	"game_main/mind/combatlifecycle"
	"game_main/tactical/commander"
	"game_main/world/garrisongen"

	"github.com/bytearena/ecs"
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
		CommanderIDs:   commandersForSquads(manager, raidState.PlayerSquadIDs),
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

// commandersForSquads returns the unique commander IDs that own the given
// squads, suitable for combatlifecycle.GrantTarget.CommanderIDs. Squads
// without a resolvable owner are skipped, and a single commander leading
// multiple squads appears only once — both are required by Grant's contract.
func commandersForSquads(manager *common.EntityManager, squadIDs []ecs.EntityID) []ecs.EntityID {
	if len(squadIDs) == 0 {
		return nil
	}
	commanderIDs := make([]ecs.EntityID, 0, len(squadIDs))
	seen := make(map[ecs.EntityID]struct{}, len(squadIDs))
	for _, squadID := range squadIDs {
		cid := commander.FindCommanderForSquad(squadID, manager)
		if cid == 0 {
			continue
		}
		if _, dup := seen[cid]; dup {
			continue
		}
		seen[cid] = struct{}{}
		commanderIDs = append(commanderIDs, cid)
	}
	return commanderIDs
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
