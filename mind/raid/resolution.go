package raid

import (
	"fmt"

	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// ProcessVictory handles a successful encounter in the raid.
// Marks the room cleared, applies morale changes, priority target effects, and rewards.
// roomNodeID identifies the specific room that was fought in.
// preCombatAliveCounts maps squad ID → living unit count before the encounter (for death penalties).
func ProcessVictory(manager *common.EntityManager, raidState *RaidStateData, roomNodeID int, preCombatAliveCounts map[ecs.EntityID]int) string {
	if raidState == nil {
		return ""
	}

	floorNumber := raidState.CurrentFloor

	room := GetRoomData(manager, roomNodeID, floorNumber)
	if room == nil {
		fmt.Printf("ProcessVictory: Room %d not found on floor %d\n", roomNodeID, floorNumber)
		return ""
	}

	// Mark garrison squads as destroyed
	for _, gSquadID := range room.GarrisonSquadIDs {
		gData := common.GetComponentTypeByID[*GarrisonSquadData](manager, gSquadID, GarrisonSquadComponent)
		if gData != nil {
			gData.IsDestroyed = true
		}
		raidState.GarrisonKillCount++
	}

	MarkRoomCleared(manager, room.NodeID, floorNumber)

	// Apply victory morale bonus
	if RaidConfig != nil {
		for _, squadID := range raidState.PlayerSquadIDs {
			applyMoraleBonus(manager, squadID, RaidConfig.Recovery.VictoryMoraleBonus)
		}
	}

	// Grant room-specific rewards
	rewardText := GrantRoomReward(manager, raidState, room.RoomType)

	fmt.Printf("ProcessVictory: Room %d (%s) cleared on floor %d\n",
		room.NodeID, room.RoomType, floorNumber)

	// Check floor completion
	if IsFloorComplete(manager, floorNumber) {
		floorState := GetFloorState(manager, floorNumber)
		if floorState != nil {
			floorState.IsComplete = true
		}
		fmt.Printf("ProcessVictory: Floor %d complete!\n", floorNumber)
	}

	return rewardText
}

// ProcessDefeat handles a lost encounter.
func ProcessDefeat(manager *common.EntityManager) {
	raidState := GetRaidState(manager)
	if raidState == nil {
		return
	}

	// Apply defeat morale penalty
	if RaidConfig != nil {
		for _, squadID := range raidState.PlayerSquadIDs {
			applyMoralePenalty(manager, squadID, RaidConfig.Recovery.DefeatMoralePenalty)
		}
	}

	raidState.Status = RaidDefeat
	fmt.Println("ProcessDefeat: Raid ended in defeat")
}

// CheckRaidEndConditions evaluates whether the raid should end.
// Returns the appropriate status (RaidActive if still going).
func CheckRaidEndConditions(manager *common.EntityManager) RaidStatus {
	raidState := GetRaidState(manager)
	if raidState == nil {
		return RaidDefeat
	}

	// Check if all player squads are destroyed
	allDestroyed := true
	for _, squadID := range raidState.PlayerSquadIDs {
		if !squads.IsSquadDestroyed(squadID, manager) {
			allDestroyed = false
			break
		}
	}
	if allDestroyed {
		return RaidDefeat
	}

	// Check if final floor is complete
	if raidState.CurrentFloor == raidState.TotalFloors {
		if IsFloorComplete(manager, raidState.CurrentFloor) {
			return RaidVictory
		}
	}

	return RaidActive
}
