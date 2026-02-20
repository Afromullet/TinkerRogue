package raid

import (
	"fmt"

	"game_main/common"
	"game_main/world/worldmap"
)

// buildFloorGraph creates RoomData entities from a DAG. Called by generateFloor
// to create room entities before garrison squads are assigned.
func buildFloorGraph(manager *common.EntityManager, dag *worldmap.FloorDAG, floorNumber int) {
	for _, node := range dag.Nodes {
		isEntry := node.ID == dag.EntryNodeID

		roomEntity := manager.World.NewEntity()
		roomEntity.AddComponent(RoomDataComponent, &RoomData{
			NodeID:         node.ID,
			RoomType:       node.RoomType,
			FloorNumber:    floorNumber,
			IsCleared:      false,
			IsAccessible:   isEntry,
			ChildNodeIDs:   append([]int{}, node.Children...),
			ParentNodeIDs:  append([]int{}, node.Parents...),
			OnCriticalPath: node.OnCriticalPath,
		})
	}
}

// GetAccessibleRooms returns node IDs of rooms the player can currently enter on a floor.
// A room is accessible if all its parent rooms are cleared, or if it's the entry room.
func GetAccessibleRooms(manager *common.EntityManager, floorNumber int) []int {
	var accessible []int
	rooms := GetAllRoomsForFloor(manager, floorNumber)

	for _, room := range rooms {
		if room.IsAccessible && !room.IsCleared {
			accessible = append(accessible, room.NodeID)
		}
	}
	return accessible
}

// MarkRoomCleared marks a room as cleared and recalculates accessibility for child rooms.
func MarkRoomCleared(manager *common.EntityManager, nodeID, floorNumber int) {
	room := GetRoomData(manager, nodeID, floorNumber)
	if room == nil {
		fmt.Printf("WARNING: MarkRoomCleared: room %d on floor %d not found\n", nodeID, floorNumber)
		return
	}

	room.IsCleared = true
	room.IsAccessible = true // Cleared rooms are always "accessible" (for display)

	// Update floor state
	floorState := GetFloorState(manager, floorNumber)
	if floorState != nil {
		floorState.RoomsCleared++
	}

	// Recalculate accessibility for child rooms
	for _, childNodeID := range room.ChildNodeIDs {
		updateRoomAccessibility(manager, childNodeID, floorNumber)
	}

	fmt.Printf("Room %d on floor %d marked cleared. Updating child accessibility.\n", nodeID, floorNumber)
}

// updateRoomAccessibility checks if all parent rooms of a room are cleared.
// If so, the room becomes accessible.
func updateRoomAccessibility(manager *common.EntityManager, nodeID, floorNumber int) {
	room := GetRoomData(manager, nodeID, floorNumber)
	if room == nil || room.IsAccessible {
		return
	}

	// A room is accessible if ALL parent rooms are cleared
	allParentsCleared := true
	for _, parentID := range room.ParentNodeIDs {
		parentRoom := GetRoomData(manager, parentID, floorNumber)
		if parentRoom == nil || !parentRoom.IsCleared {
			allParentsCleared = false
			break
		}
	}

	if allParentsCleared {
		room.IsAccessible = true
	}
}

// IsFloorComplete returns true if the stairs room is cleared or accessible
// (meaning the player has fought through to the exit).
func IsFloorComplete(manager *common.EntityManager, floorNumber int) bool {
	rooms := GetAllRoomsForFloor(manager, floorNumber)
	for _, room := range rooms {
		if room.RoomType == worldmap.GarrisonRoomStairs && room.IsCleared {
			return true
		}
	}
	return false
}
