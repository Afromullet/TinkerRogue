package raid

import (
	"fmt"

	"game_main/core/common"
	"game_main/mind/combatlifecycle"
	"game_main/tactical/squads/squadcore"
)

// RaidEncounterResolver dispatches a raid combat outcome to the appropriate
// per-reason resolver. Built eagerly by RaidCombatStarter.Prepare and run by
// the standard ExecuteResolution pipeline at exit time.
type RaidEncounterResolver struct {
	RaidState  *RaidStateData
	RoomNodeID int
}

func (r *RaidEncounterResolver) Resolve(manager *common.EntityManager, ctx combatlifecycle.ResolutionContext) *combatlifecycle.ResolutionPlan {
	if ctx.PlayerVictory {
		return (&RaidRoomResolver{RaidState: r.RaidState, RoomNodeID: r.RoomNodeID}).Resolve(manager, ctx)
	}
	return (&RaidDefeatResolver{}).Resolve(manager, ctx)
}

// RaidRoomResolver resolves a successful raid room encounter.
type RaidRoomResolver struct {
	RaidState  *RaidStateData
	RoomNodeID int
}

func (r *RaidRoomResolver) Resolve(manager *common.EntityManager, _ combatlifecycle.ResolutionContext) *combatlifecycle.ResolutionPlan {
	if r.RaidState == nil {
		return nil
	}

	floorNumber := r.RaidState.CurrentFloor

	room := GetRoomData(manager, r.RoomNodeID, floorNumber)
	if room == nil {
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

	// Check floor completion
	if IsFloorComplete(manager, floorNumber) {
		floorState := GetFloorState(manager, floorNumber)
		if floorState != nil {
			floorState.IsComplete = true
		}
	}

	// Calculate room reward (pipeline grants it)
	reward, target := calculateRoomReward(manager, r.RaidState, room.RoomType)

	return &combatlifecycle.ResolutionPlan{
		Rewards:     reward,
		Target:      target,
		Description: fmt.Sprintf("Room %d (%s) cleared", room.NodeID, room.RoomType),
	}
}

// RaidDefeatResolver resolves a raid defeat (combat loss or flee).
type RaidDefeatResolver struct{}

func (r *RaidDefeatResolver) Resolve(manager *common.EntityManager, _ combatlifecycle.ResolutionContext) *combatlifecycle.ResolutionPlan {
	raidState := GetRaidState(manager)
	if raidState != nil {
		raidState.Status = RaidDefeat
	}
	return &combatlifecycle.ResolutionPlan{
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
		if !squadcore.IsSquadDestroyed(squadID, manager) {
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
