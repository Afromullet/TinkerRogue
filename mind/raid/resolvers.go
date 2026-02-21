package raid

import (
	"fmt"

	"game_main/common"
	"game_main/mind/combatpipeline"
	"game_main/tactical/squads"
)

// RaidRoomResolver resolves a successful raid room encounter.
// Replaces ProcessVictory.
type RaidRoomResolver struct {
	RaidState  *RaidStateData
	RoomNodeID int
}

func (r *RaidRoomResolver) Resolve(manager *common.EntityManager) *combatpipeline.ResolutionPlan {
	if r.RaidState == nil {
		return nil
	}

	floorNumber := r.RaidState.CurrentFloor

	room := GetRoomData(manager, r.RoomNodeID, floorNumber)
	if room == nil {
		fmt.Printf("RaidRoomResolver: Room %d not found on floor %d\n", r.RoomNodeID, floorNumber)
		return nil
	}

	// Mark garrison squads as destroyed
	for _, gSquadID := range room.GarrisonSquadIDs {
		gData := common.GetComponentTypeByID[*GarrisonSquadData](manager, gSquadID, GarrisonSquadComponent)
		if gData != nil {
			gData.IsDestroyed = true
		}
	}

	MarkRoomCleared(manager, room.NodeID, floorNumber)

	fmt.Printf("RaidRoomResolver: Room %d (%s) cleared on floor %d\n",
		room.NodeID, room.RoomType, floorNumber)

	// Check floor completion
	if IsFloorComplete(manager, floorNumber) {
		floorState := GetFloorState(manager, floorNumber)
		if floorState != nil {
			floorState.IsComplete = true
		}
		fmt.Printf("RaidRoomResolver: Floor %d complete!\n", floorNumber)
	}

	// Calculate room reward (pipeline grants it)
	reward, target := calculateRoomReward(manager, r.RaidState, room.RoomType)

	return &combatpipeline.ResolutionPlan{
		Rewards:     reward,
		Target:      target,
		Description: fmt.Sprintf("Room %d (%s) cleared", room.NodeID, room.RoomType),
	}
}

// RaidDefeatResolver resolves a raid defeat (combat loss or flee).
// Replaces ProcessDefeat.
type RaidDefeatResolver struct{}

func (r *RaidDefeatResolver) Resolve(manager *common.EntityManager) *combatpipeline.ResolutionPlan {
	raidState := GetRaidState(manager)
	if raidState != nil {
		raidState.Status = RaidDefeat
	}
	fmt.Println("RaidDefeatResolver: Raid ended in defeat")

	return &combatpipeline.ResolutionPlan{
		Description: "Raid ended in defeat",
	}
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
