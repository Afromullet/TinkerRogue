package raid

import (
	"fmt"

	"game_main/common"
	"game_main/mind/resolution"
	"game_main/world/worldmap"
)

// GrantRoomReward grants room-specific rewards when a room is cleared.
// Returns a description of the reward granted, or "" if none.
func GrantRoomReward(manager *common.EntityManager, raidState *RaidStateData, roomType string) string {
	if RaidConfig == nil || raidState == nil {
		return ""
	}

	switch roomType {
	case worldmap.GarrisonRoomCommandPost:
		return grantCommandPostReward(manager, raidState)
	}

	return ""
}

// grantCommandPostReward restores mana to the commander via the reward package.
// Returns a description of the resolution.
func grantCommandPostReward(manager *common.EntityManager, raidState *RaidStateData) string {
	manaRestore := RaidConfig.Rewards.CommandPostManaRestore
	if manaRestore <= 0 {
		return ""
	}

	r := resolution.Reward{Mana: manaRestore}
	target := resolution.GrantTarget{CommanderID: raidState.CommanderID}
	desc := resolution.Grant(manager, r, target)

	if desc != "" {
		fmt.Printf("Reward: Command post cleared — %s\n", desc)
	}
	return desc
}
