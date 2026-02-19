package raid

import (
	"fmt"

	"game_main/common"
	"game_main/tactical/spells"
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

// grantCommandPostReward restores mana to the commander.
// Returns a description of the reward.
func grantCommandPostReward(manager *common.EntityManager, raidState *RaidStateData) string {
	manaRestore := RaidConfig.Rewards.CommandPostManaRestore
	if manaRestore <= 0 {
		return ""
	}

	manaData := common.GetComponentTypeByID[*spells.ManaData](manager, raidState.CommanderID, spells.ManaComponent)
	if manaData == nil {
		return ""
	}

	manaData.CurrentMana += manaRestore
	if manaData.CurrentMana > manaData.MaxMana {
		manaData.CurrentMana = manaData.MaxMana
	}

	reward := fmt.Sprintf("Commander gained %d mana (%d/%d)", manaRestore, manaData.CurrentMana, manaData.MaxMana)
	fmt.Printf("Reward: Command post cleared — %s\n", reward)
	return reward
}
