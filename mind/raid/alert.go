package raid

import (
	"game_main/common"
	"game_main/tactical/squads/squadcore"
	"game_main/world/worldmap/garrison"

	"github.com/bytearena/ecs"
)

// ActivateReserves moves reserve squads into accessible rooms when the alert level
// config has ActivatesReserves: true.
func ActivateReserves(manager *common.EntityManager, floorNumber int) {
	alertData := GetAlertData(manager, floorNumber)
	if alertData == nil {
		return
	}

	levelCfg := GetAlertLevel(alertData.CurrentLevel)
	if levelCfg == nil || !levelCfg.ActivatesReserves {
		return
	}

	floorState := GetFloorState(manager, floorNumber)
	if floorState == nil || len(floorState.ReserveSquadIDs) == 0 {
		return
	}

	// Find accessible, uncleared rooms that could receive reinforcements
	rooms := GetAllRoomsForFloor(manager, floorNumber)
	var eligibleRooms []*RoomData
	for _, room := range rooms {
		if room.IsAccessible && !room.IsCleared && room.RoomType != garrison.GarrisonRoomRestRoom && room.RoomType != garrison.GarrisonRoomStairs {
			eligibleRooms = append(eligibleRooms, room)
		}
	}

	if len(eligibleRooms) == 0 {
		return
	}

	// Move reserve squads into eligible rooms
	activated := 0
	var remainingReserves []ecs.EntityID
	for _, reserveID := range floorState.ReserveSquadIDs {
		gData := common.GetComponentTypeByID[*GarrisonSquadData](manager, reserveID, GarrisonSquadComponent)
		if gData == nil || squadcore.IsSquadDestroyed(reserveID, manager) {
			continue
		}

		if activated < len(eligibleRooms) {
			room := eligibleRooms[activated]
			gData.IsReserve = false
			gData.RoomNodeID = room.NodeID
			room.GarrisonSquadIDs = append(room.GarrisonSquadIDs, reserveID)
			floorState.GarrisonSquadIDs = append(floorState.GarrisonSquadIDs, reserveID)
			activated++

		} else {
			remainingReserves = append(remainingReserves, reserveID)
		}
	}

	floorState.ReserveSquadIDs = remainingReserves
}

// IncrementAlert updates the alert level based on encounter count and applies effects.
// Replaces the hardcoded alert logic from raidrunner.go.
func IncrementAlert(manager *common.EntityManager, floorNumber int) {
	alertData := GetAlertData(manager, floorNumber)
	if alertData == nil {
		return
	}

	alertData.EncounterCount++
	oldLevel := alertData.CurrentLevel

	// Determine new alert level from encounter count using config thresholds.
	// Iterate levels in descending order so the highest qualifying level wins.
	if RaidConfig != nil {
		for i := len(RaidConfig.Alert.Levels) - 1; i >= 0; i-- {
			lvl := RaidConfig.Alert.Levels[i]
			if lvl.EncounterThreshold > 0 && alertData.EncounterCount >= lvl.EncounterThreshold && alertData.CurrentLevel < lvl.Level {
				alertData.CurrentLevel = lvl.Level
				break
			}
		}
	}

	// Activate reserves if level changed
	if alertData.CurrentLevel != oldLevel {
		ActivateReserves(manager, floorNumber)
	}
}
