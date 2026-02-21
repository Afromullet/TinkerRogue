package raid

import (
	"game_main/common"
)

// GetRaidState returns the singleton RaidStateData, or nil if no raid is active.
func GetRaidState(manager *common.EntityManager) *RaidStateData {
	for _, result := range manager.World.Query(RaidStateTag) {
		return common.GetComponentType[*RaidStateData](result.Entity, RaidStateComponent)
	}
	return nil
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
