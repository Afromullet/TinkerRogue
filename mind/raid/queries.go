package raid

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// GetRaidState returns the singleton RaidStateData, or nil if no raid is active.
func GetRaidState(manager *common.EntityManager) *RaidStateData {
	for _, result := range manager.World.Query(RaidStateTag) {
		return common.GetComponentType[*RaidStateData](result.Entity, RaidStateComponent)
	}
	return nil
}

// GetRaidStateEntity returns the raid state entity ID, or 0 if none exists.
func GetRaidStateEntity(manager *common.EntityManager) ecs.EntityID {
	for _, result := range manager.World.Query(RaidStateTag) {
		return result.Entity.GetID()
	}
	return 0
}

// GetFloorState returns the FloorStateData for a given floor number, or nil if not found.
func GetFloorState(manager *common.EntityManager, floorNumber int) *FloorStateData {
	for _, result := range manager.World.Query(FloorStateTag) {
		data := common.GetComponentType[*FloorStateData](result.Entity, FloorStateComponent)
		if data != nil && data.FloorNumber == floorNumber {
			return data
		}
	}
	return nil
}

// GetRoomData returns the RoomData for a specific room on a floor, or nil if not found.
func GetRoomData(manager *common.EntityManager, nodeID, floorNumber int) *RoomData {
	for _, result := range manager.World.Query(RoomDataTag) {
		data := common.GetComponentType[*RoomData](result.Entity, RoomDataComponent)
		if data != nil && data.NodeID == nodeID && data.FloorNumber == floorNumber {
			return data
		}
	}
	return nil
}

// GetAlertData returns the AlertData for a given floor, or nil if not found.
func GetAlertData(manager *common.EntityManager, floorNumber int) *AlertData {
	for _, result := range manager.World.Query(AlertDataTag) {
		data := common.GetComponentType[*AlertData](result.Entity, AlertDataComponent)
		if data != nil && data.FloorNumber == floorNumber {
			return data
		}
	}
	return nil
}

// GetGarrisonSquadsForFloor returns all garrison squad entity IDs assigned to a floor.
func GetGarrisonSquadsForFloor(manager *common.EntityManager, floorNumber int) []ecs.EntityID {
	var ids []ecs.EntityID
	for _, result := range manager.World.Query(GarrisonSquadTag) {
		data := common.GetComponentType[*GarrisonSquadData](result.Entity, GarrisonSquadComponent)
		if data != nil && data.FloorNumber == floorNumber {
			ids = append(ids, result.Entity.GetID())
		}
	}
	return ids
}

// IsRoomCleared returns true if the room has been cleared by the player.
func IsRoomCleared(manager *common.EntityManager, nodeID, floorNumber int) bool {
	room := GetRoomData(manager, nodeID, floorNumber)
	if room == nil {
		return false
	}
	return room.IsCleared
}

// GetAllRoomsForFloor returns all RoomData entities for a given floor.
func GetAllRoomsForFloor(manager *common.EntityManager, floorNumber int) []*RoomData {
	var rooms []*RoomData
	for _, result := range manager.World.Query(RoomDataTag) {
		data := common.GetComponentType[*RoomData](result.Entity, RoomDataComponent)
		if data != nil && data.FloorNumber == floorNumber {
			rooms = append(rooms, data)
		}
	}
	return rooms
}

// GetDeployment returns the current DeploymentData, or nil if none exists.
func GetDeployment(manager *common.EntityManager) *DeploymentData {
	for _, result := range manager.World.Query(DeploymentTag) {
		return common.GetComponentType[*DeploymentData](result.Entity, DeploymentComponent)
	}
	return nil
}
