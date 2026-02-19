package worldmap

import (
	"game_main/common"
	"game_main/world/coords"
)

// GarrisonRoomMeta holds metadata for a single placed garrison room.
type GarrisonRoomMeta struct {
	RoomIndex      int
	RoomType       string
	Rect           Rect
	PlayerSpawns   []coords.LogicalPosition
	DefenderSpawns []coords.LogicalPosition
	OnCriticalPath bool
	ConnectedRooms []int // DAG edge targets (room indices by node ID)
}

// GarrisonFloorData holds per-floor metadata for the garrison raid.
type GarrisonFloorData struct {
	Rooms           []GarrisonRoomMeta
	EntryRoomIndex  int
	StairsRoomIndex int
	FloorNumber     int
}

// Spawn count targets per room type: {minPlayer, maxPlayer, minDefender, maxDefender}
var garrisonSpawnCounts = map[string][4]int{
	GarrisonRoomGuardPost:   {2, 3, 1, 2},
	GarrisonRoomBarracks:    {3, 4, 2, 3},
	GarrisonRoomArmory:      {2, 3, 1, 2},
	GarrisonRoomCommandPost: {2, 3, 1, 2},
	GarrisonRoomPatrolRoute: {2, 3, 2, 3},
	GarrisonRoomMageTower:   {2, 3, 1, 2},
	GarrisonRoomRestRoom:    {2, 3, 0, 0},
	GarrisonRoomStairs:      {2, 2, 0, 0},
}

// SetGarrisonSpawnCounts replaces the spawn count table with values from external config.
func SetGarrisonSpawnCounts(counts map[string][4]int) {
	garrisonSpawnCounts = counts
}

// buildGarrisonFloorData constructs floor metadata from the DAG and placed rooms.
func buildGarrisonFloorData(dag *FloorDAG, placedRooms map[int]Rect, floorNumber int, result *GenerationResult, width int) *GarrisonFloorData {
	data := &GarrisonFloorData{
		Rooms:           make([]GarrisonRoomMeta, 0, len(dag.Nodes)),
		EntryRoomIndex:  dag.EntryNodeID,
		StairsRoomIndex: dag.StairsNodeID,
		FloorNumber:     floorNumber,
	}

	for _, node := range dag.Nodes {
		room := placedRooms[node.ID]
		meta := GarrisonRoomMeta{
			RoomIndex:      node.ID,
			RoomType:       node.RoomType,
			Rect:           room,
			OnCriticalPath: node.OnCriticalPath,
			ConnectedRooms: append([]int{}, node.Children...),
		}

		// Compute spawn positions (terrain-aware zone-based search)
		meta.PlayerSpawns = computePlayerSpawns(room, node, dag, placedRooms, result, width)
		meta.DefenderSpawns = computeDefenderSpawns(room, node, result, width)

		data.Rooms = append(data.Rooms, meta)
	}

	return data
}

// computePlayerSpawns finds spawn points in the entry zone of the room using zone-based search.
func computePlayerSpawns(room Rect, node *FloorNode, dag *FloorDAG, placedRooms map[int]Rect, result *GenerationResult, width int) []coords.LogicalPosition {
	counts := garrisonSpawnCounts[node.RoomType]
	targetCount := common.GetRandomBetween(int(counts[0]), int(counts[1]))

	// Determine entry direction
	entryFromLeft := true
	if len(node.Parents) > 0 {
		parentRoom := placedRooms[node.Parents[0]]
		parentCX, _ := parentRoom.Center()
		roomCX, _ := room.Center()
		entryFromLeft = parentCX < roomCX
	}

	// Define the player entry zone: a 4-tile-deep strip at the entry edge
	var zoneX1, zoneX2, zoneY1, zoneY2 int
	entryDepth := 4
	if entryFromLeft {
		zoneX1 = room.X1 + 2
		zoneX2 = min(room.X1+2+entryDepth, room.X2-2)
	} else {
		zoneX2 = room.X2 - 2
		zoneX1 = max(room.X2-2-entryDepth, room.X1+2)
	}
	zoneY1 = room.Y1 + 2
	zoneY2 = room.Y2 - 2

	return findSpawnsInZone(zoneX1, zoneY1, zoneX2, zoneY2, targetCount, 2, result, width)
}

// computeDefenderSpawns finds spawn points in the defender zone based on room type.
func computeDefenderSpawns(room Rect, node *FloorNode, result *GenerationResult, width int) []coords.LogicalPosition {
	counts := garrisonSpawnCounts[node.RoomType]
	targetCount := common.GetRandomBetween(int(counts[2]), int(counts[3]))
	if targetCount == 0 {
		return nil
	}

	roomW := room.X2 - room.X1
	roomH := room.Y2 - room.Y1

	var zoneX1, zoneX2, zoneY1, zoneY2 int

	switch node.RoomType {
	case GarrisonRoomGuardPost, GarrisonRoomCommandPost:
		// Anchored: behind primary terrain feature (rear 40%)
		zoneX1 = room.X1 + roomW*60/100
		zoneX2 = room.X2 - 2
		zoneY1 = room.Y1 + 2
		zoneY2 = room.Y2 - 2

	case GarrisonRoomBarracks, GarrisonRoomArmory:
		// Distributed: rear half, spread out
		zoneX1 = room.X1 + roomW/2
		zoneX2 = room.X2 - 2
		zoneY1 = room.Y1 + 2
		zoneY2 = room.Y2 - 2

	case GarrisonRoomPatrolRoute, GarrisonRoomMageTower:
		// Mobile: rear third, spread wide
		zoneX1 = room.X1 + roomW*2/3
		zoneX2 = room.X2 - 2
		zoneY1 = room.Y1 + max(2, roomH/6)
		zoneY2 = room.Y2 - max(2, roomH/6)

	default:
		// Generic center
		zoneX1 = room.X1 + roomW/3
		zoneX2 = room.X2 - roomW/3
		zoneY1 = room.Y1 + roomH/3
		zoneY2 = room.Y2 - roomH/3
	}

	return findSpawnsInZone(zoneX1, zoneY1, zoneX2, zoneY2, targetCount, 2, result, width)
}

// findSpawnsInZone searches a rectangular zone for valid spawn positions.
// Returns up to targetCount positions, each at least minSpacing tiles apart.
func findSpawnsInZone(x1, y1, x2, y2, targetCount, minSpacing int, result *GenerationResult, width int) []coords.LogicalPosition {
	spawns := make([]coords.LogicalPosition, 0, targetCount)

	// Collect all valid candidate positions in the zone
	candidates := make([]coords.LogicalPosition, 0)
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			pos := coords.LogicalPosition{X: x, Y: y}
			if isSpawnValid(pos, result, width) {
				candidates = append(candidates, pos)
			}
		}
	}

	if len(candidates) == 0 {
		return spawns
	}

	// Shuffle candidates for variety
	for i := len(candidates) - 1; i > 0; i-- {
		j := common.GetRandomBetween(0, i)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}

	// Greedily select positions that satisfy spacing constraints
	placedCoords := make([][2]int, 0, targetCount)
	for _, cand := range candidates {
		if len(spawns) >= targetCount {
			break
		}

		if !isTooCloseToAny(cand.X, cand.Y, placedCoords, minSpacing) {
			spawns = append(spawns, cand)
			placedCoords = append(placedCoords, [2]int{cand.X, cand.Y})
		}
	}

	return spawns
}

// isSpawnValid checks that a position is walkable and has a 3x3 clear area.
func isSpawnValid(pos coords.LogicalPosition, result *GenerationResult, width int) bool {
	numTiles := len(result.Tiles)
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			nx := pos.X + dx
			ny := pos.Y + dy
			idx := positionToIndex(nx, ny, width)
			if idx < 0 || idx >= numTiles {
				return false
			}
			if result.Tiles[idx].Blocked {
				return false
			}
		}
	}
	return true
}
