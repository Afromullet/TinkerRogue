package raid

import (
	"game_main/common"
	"game_main/world/worldmap"
)

// AssignArchetypesToFloor determines which archetype should defend each room.
// Returns a map of nodeID → archetype name.
// Rest rooms and stairs get no assignment (not present in returned map).
func AssignArchetypesToFloor(dag *worldmap.FloorDAG, floorNumber int) map[int]string {
	assignments := make(map[int]string)

	for _, node := range dag.Nodes {
		// Skip non-combat rooms
		if node.RoomType == worldmap.GarrisonRoomRestRoom || node.RoomType == worldmap.GarrisonRoomStairs {
			continue
		}

		archName := pickArchetypeForRoom(node.RoomType, node.OnCriticalPath, floorNumber)
		if archName != "" {
			assignments[node.ID] = archName
		}
	}

	return assignments
}

// pickArchetypeForRoom selects the best archetype for a room type.
// Prefers archetypes whose PreferredRooms list matches the room type.
// Falls back to a generic combat archetype if no preferred match is found.
func pickArchetypeForRoom(roomType string, onCriticalPath bool, floorNumber int) string {
	// Collect archetypes that prefer this room type
	var preferred []string
	for _, arch := range GarrisonArchetypes {
		for _, pref := range arch.PreferredRooms {
			if pref == roomType {
				preferred = append(preferred, arch.Name)
				break
			}
		}
	}

	if len(preferred) > 0 {
		return preferred[common.RandomInt(len(preferred))]
	}

	// No preferred match — pick a generic combat archetype based on path position
	criticalArchetypes := CriticalPathArchetypes()
	branchArchetypes := BranchArchetypes()
	eliteArchetypes := EliteArchetypes()
	eliteFloorThreshold := EliteFloorThreshold()

	if onCriticalPath {
		return criticalArchetypes[common.RandomInt(len(criticalArchetypes))]
	}

	// Higher floors unlock more dangerous archetypes
	pool := branchArchetypes
	if floorNumber >= eliteFloorThreshold {
		pool = append(append([]string{}, pool...), eliteArchetypes...)
	}

	return pool[common.RandomInt(len(pool))]
}
