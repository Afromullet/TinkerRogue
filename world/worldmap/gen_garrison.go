package worldmap

import (
	"game_main/common"
	"game_main/world/coords"
)

// GarrisonFloorConfig holds parameters for garrison floor generation.
type GarrisonFloorConfig struct {
	FloorNumber int
	Seed        int64
}

// DefaultGarrisonConfig returns a default config for floor 1.
func DefaultGarrisonConfig() GarrisonFloorConfig {
	return GarrisonFloorConfig{
		FloorNumber: 1,
		Seed:        0,
	}
}

// GarrisonRaidGenerator produces DAG-based garrison raid floors.
type GarrisonRaidGenerator struct {
	config GarrisonFloorConfig
}

func (g *GarrisonRaidGenerator) Name() string        { return "garrison_raid" }
func (g *GarrisonRaidGenerator) Description() string  { return "DAG-based garrison raid floors" }

func (g *GarrisonRaidGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	if g.config.Seed != 0 {
		common.SetRNGSeed(uint64(g.config.Seed), uint64(g.config.Seed))
	}

	result := GenerationResult{
		Tiles:          createEmptyTiles(width, height, images),
		Rooms:          make([]Rect, 0),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	floorNum := g.config.FloorNumber
	if floorNum < 1 {
		floorNum = 1
	}

	// Build abstract DAG
	dag := buildGarrisonDAG(floorNum)

	// Place rooms on the grid in topological order
	sorted := topologicalSort(dag)
	depths := dagDepth(dag)

	maxDepth := 0
	for _, d := range depths {
		if d > maxDepth {
			maxDepth = d
		}
	}
	if maxDepth == 0 {
		maxDepth = 1
	}

	placedRooms := make(map[int]Rect)
	roomPadding := 2

	for _, node := range sorted {
		roomW := common.GetRandomBetween(node.MinWidth, node.MaxWidth)
		roomH := common.GetRandomBetween(node.MinHeight, node.MaxHeight)

		// Target X band based on DAG depth (left-to-right progression)
		depth := depths[node.ID]
		usableW := width - roomW - 4
		if usableW < 1 {
			usableW = 1
		}
		// Use maxDepth+1 to distribute bands evenly across all depth levels
		xBandStart := 2 + usableW*depth/(maxDepth+1)
		xBandWidth := usableW / (maxDepth + 1)
		xBandEnd := xBandStart + xBandWidth
		if xBandEnd > width-roomW-2 {
			xBandEnd = width - roomW - 2
		}
		if xBandStart < 2 {
			xBandStart = 2
		}
		if xBandEnd <= xBandStart {
			xBandEnd = xBandStart + 1
		}

		// Target Y: center of map, offset for branch rooms
		targetY := height/2 - roomH/2
		if !node.OnCriticalPath && len(node.Parents) > 0 {
			parentRect, ok := placedRooms[node.Parents[0]]
			if ok {
				parentCY := (parentRect.Y1 + parentRect.Y2) / 2
				parentH := parentRect.Y2 - parentRect.Y1
				if common.GetDiceRoll(2) == 1 {
					targetY = parentCY - roomH - roomPadding*2
				} else {
					targetY = parentCY + parentH + roomPadding*2
				}
			}
		}

		placed := false
		var bestRect Rect

		// Attempt placement with increasing Y jitter
		for attempt := 0; attempt < 60; attempt++ {
			x := common.GetRandomBetween(xBandStart, xBandEnd)
			y := targetY + common.GetRandomBetween(-attempt, attempt)

			// Clamp to map bounds (leave 1-tile border)
			x = max(1, min(x, width-roomW-1))
			y = max(1, min(y, height-roomH-1))

			candidate := NewRect(x, y, roomW, roomH)
			if !garrisonOverlapsAny(candidate, placedRooms, roomPadding) {
				bestRect = candidate
				placed = true
				break
			}
		}

		// Fallback: progressively shrink room and retry
		if !placed {
			for shrinkStep := 1; shrinkStep <= 3 && !placed; shrinkStep++ {
				shrunkW := max(roomW-shrinkStep*2, 6)
				shrunkH := max(roomH-shrinkStep*2, 6)
				for attempt := 0; attempt < 60; attempt++ {
					x := common.GetRandomBetween(2, width-shrunkW-2)
					y := common.GetRandomBetween(2, height-shrunkH-2)
					candidate := NewRect(x, y, shrunkW, shrunkH)
					if !garrisonOverlapsAny(candidate, placedRooms, roomPadding) {
						bestRect = candidate
						placed = true
						break
					}
				}
			}
		}

		// Last resort: minimum size with reduced padding
		if !placed {
			minW := max(roomW-6, 6)
			minH := max(roomH-6, 6)
			for attempt := 0; attempt < 100; attempt++ {
				x := common.GetRandomBetween(2, max(3, width-minW-2))
				y := common.GetRandomBetween(2, max(3, height-minH-2))
				candidate := NewRect(x, y, minW, minH)
				if !garrisonOverlapsAny(candidate, placedRooms, 1) {
					bestRect = candidate
					placed = true
					break
				}
			}
		}

		// Absolute last resort: force-place at minimum size
		if !placed {
			x := common.GetRandomBetween(2, max(3, width-6-2))
			y := common.GetRandomBetween(2, max(3, height-6-2))
			bestRect = NewRect(x, y, 6, 6)
		}

		placedRooms[node.ID] = bestRect
		carveRoom(&result, bestRect, width, images)
		result.Rooms = append(result.Rooms, bestRect)
	}

	// Carve corridors along DAG edges with variable width and shape
	for _, node := range dag.Nodes {
		parentRoom := placedRooms[node.ID]
		for _, childID := range node.Children {
			childRoom := placedRooms[childID]
			childNode := dag.Nodes[childID]

			// Determine corridor width based on connected room types
			corridorWidth := garrisonCorridorWidth(node.RoomType, childNode.RoomType)

			// Use edge-based connection points instead of room centers
			fromX, fromY := garrisonEdgePoint(parentRoom, childRoom)
			toX, toY := garrisonEdgePoint(childRoom, parentRoom)

			// Choose corridor shape: Z-shape for branches, L-shape for critical path
			if !node.OnCriticalPath || !childNode.OnCriticalPath {
				carveZShapeCorridor(&result, fromX, fromY, toX, toY, corridorWidth, width, height, images)
			} else {
				carveLShapeCorridor(&result, fromX, fromY, toX, toY, corridorWidth, width, height, images)
			}

			// Place doorway markers (narrow to 1 tile at room boundaries)
			placeDoorway(&result, parentRoom, fromX, fromY, width, height, images)
			placeDoorway(&result, childRoom, toX, toY, width, height, images)
		}
	}

	// Ensure all rooms are connected via flood fill + corridor patching
	terrainMap := make([]bool, width*height)
	for i, t := range result.Tiles {
		if t != nil && !t.Blocked {
			terrainMap[i] = true
		}
	}
	ensureTerrainConnectivity(terrainMap, width, height)
	for i := 0; i < len(terrainMap); i++ {
		if terrainMap[i] && result.Tiles[i].Blocked {
			x := i % width
			y := i / width
			result.Tiles[i].Blocked = false
			result.Tiles[i].TileType = FLOOR
			result.Tiles[i].Image = selectRandomImage(images.FloorImages)
			result.ValidPositions = append(result.ValidPositions, coords.LogicalPosition{X: x, Y: y})
		}
	}

	// Inject per-room tactical terrain (pillars, barricades, alcoves)
	for _, node := range dag.Nodes {
		room := placedRooms[node.ID]
		injectGarrisonTerrain(node.RoomType, room, width, &result, images)
	}

	// Build floor metadata (spawn positions, room types)
	result.GarrisonData = buildGarrisonFloorData(dag, placedRooms, floorNum, &result, width)

	return result
}

// garrisonCorridorWidth determines corridor width based on connected room types.
func garrisonCorridorWidth(fromType, toType string) int {
	// Guard posts create chokepoints
	if fromType == GarrisonRoomGuardPost || toType == GarrisonRoomGuardPost {
		return 1
	}
	// Barracks and patrol routes use wide approaches
	if fromType == GarrisonRoomBarracks || toType == GarrisonRoomBarracks ||
		fromType == GarrisonRoomPatrolRoute || toType == GarrisonRoomPatrolRoute {
		return 3
	}
	// Default transitional width
	return 2
}

// garrisonEdgePoint finds the connection point on srcRoom's edge closest to dstRoom.
func garrisonEdgePoint(srcRoom, dstRoom Rect) (int, int) {
	srcCX, srcCY := srcRoom.Center()
	dstCX, dstCY := dstRoom.Center()

	dx := dstCX - srcCX
	dy := dstCY - srcCY

	// Determine which edge to connect from
	absDx := dx
	if absDx < 0 {
		absDx = -absDx
	}
	absDy := dy
	if absDy < 0 {
		absDy = -absDy
	}

	if absDx >= absDy {
		// Connect from left or right edge
		edgeY := srcCY + common.GetRandomBetween(-1, 1)
		edgeY = max(srcRoom.Y1+2, min(edgeY, srcRoom.Y2-2))
		if dx > 0 {
			return srcRoom.X2, edgeY // Right edge
		}
		return srcRoom.X1, edgeY // Left edge
	}

	// Connect from top or bottom edge
	edgeX := srcCX + common.GetRandomBetween(-1, 1)
	edgeX = max(srcRoom.X1+2, min(edgeX, srcRoom.X2-2))
	if dy > 0 {
		return edgeX, srcRoom.Y2 // Bottom edge
	}
	return edgeX, srcRoom.Y1 // Top edge
}

// carveLShapeCorridor carves an L-shaped corridor with variable width.
func carveLShapeCorridor(result *GenerationResult, fromX, fromY, toX, toY, corridorWidth, mapWidth, mapHeight int, images TileImageSet) {
	halfW := corridorWidth / 2

	// Horizontal segment at fromY
	for offset := -halfW; offset <= halfW; offset++ {
		y := fromY + offset
		if y >= 0 && y < mapHeight {
			carveHorizontalTunnel(result, fromX, toX, y, mapWidth, images)
		}
	}
	// Vertical segment at toX
	for offset := -halfW; offset <= halfW; offset++ {
		x := toX + offset
		if x >= 0 && x < mapWidth {
			carveVerticalTunnel(result, fromY, toY, x, mapWidth, images)
		}
	}
}

// carveZShapeCorridor carves a Z-shaped (dogleg) corridor that breaks line of sight.
func carveZShapeCorridor(result *GenerationResult, fromX, fromY, toX, toY, corridorWidth, mapWidth, mapHeight int, images TileImageSet) {
	halfW := corridorWidth / 2

	// Midpoint for the jog
	midX := (fromX + toX) / 2

	// First horizontal segment: fromX to midX at fromY
	for offset := -halfW; offset <= halfW; offset++ {
		y := fromY + offset
		if y >= 0 && y < mapHeight {
			carveHorizontalTunnel(result, fromX, midX, y, mapWidth, images)
		}
	}
	// Vertical jog: fromY to toY at midX
	for offset := -halfW; offset <= halfW; offset++ {
		x := midX + offset
		if x >= 0 && x < mapWidth {
			carveVerticalTunnel(result, fromY, toY, x, mapWidth, images)
		}
	}
	// Second horizontal segment: midX to toX at toY
	for offset := -halfW; offset <= halfW; offset++ {
		y := toY + offset
		if y >= 0 && y < mapHeight {
			carveHorizontalTunnel(result, midX, toX, y, mapWidth, images)
		}
	}

	// Carve the intersection corners to ensure smooth passage
	for dy := -halfW; dy <= halfW; dy++ {
		for dx := -halfW; dx <= halfW; dx++ {
			// At first corner (midX, fromY)
			cx, cy := midX+dx, fromY+dy
			if cx >= 0 && cx < mapWidth && cy >= 0 && cy < mapHeight {
				idx := positionToIndex(cx, cy, mapWidth)
				if idx >= 0 && idx < len(result.Tiles) && result.Tiles[idx].Blocked {
					result.Tiles[idx].Blocked = false
					result.Tiles[idx].TileType = FLOOR
					result.Tiles[idx].Image = selectRandomImage(images.FloorImages)
					result.ValidPositions = append(result.ValidPositions, coords.LogicalPosition{X: cx, Y: cy})
				}
			}
			// At second corner (midX, toY)
			cx2, cy2 := midX+dx, toY+dy
			if cx2 >= 0 && cx2 < mapWidth && cy2 >= 0 && cy2 < mapHeight {
				idx := positionToIndex(cx2, cy2, mapWidth)
				if idx >= 0 && idx < len(result.Tiles) && result.Tiles[idx].Blocked {
					result.Tiles[idx].Blocked = false
					result.Tiles[idx].TileType = FLOOR
					result.Tiles[idx].Image = selectRandomImage(images.FloorImages)
					result.ValidPositions = append(result.ValidPositions, coords.LogicalPosition{X: cx2, Y: cy2})
				}
			}
		}
	}
}

// placeDoorway creates a 1-tile-wide doorway where a corridor meets a room by
// placing wall tiles on either side of the connection point at the room boundary.
func placeDoorway(result *GenerationResult, room Rect, connX, connY, mapWidth, mapHeight int, images TileImageSet) {
	// Determine which edge the connection is on
	onLeft := connX == room.X1
	onRight := connX == room.X2
	onTop := connY == room.Y1
	onBottom := connY == room.Y2

	if onLeft || onRight {
		// Vertical edge: place walls above and below the connection point
		for dy := -2; dy <= 2; dy++ {
			if dy >= -1 && dy <= 1 {
				continue // Leave the doorway open (3 tiles centered on connY)
			}
			wy := connY + dy
			if wy > room.Y1 && wy < room.Y2 && wy >= 0 && wy < mapHeight {
				// Only wall-ify if it's currently a floor tile in the corridor area
				idx := positionToIndex(connX, wy, mapWidth)
				if idx >= 0 && idx < len(result.Tiles) && !result.Tiles[idx].Blocked {
					setTileWall(result, connX, wy, mapWidth, images)
				}
			}
		}
	} else if onTop || onBottom {
		// Horizontal edge: place walls left and right of the connection point
		for dx := -2; dx <= 2; dx++ {
			if dx >= -1 && dx <= 1 {
				continue // Leave the doorway open
			}
			wx := connX + dx
			if wx > room.X1 && wx < room.X2 && wx >= 0 && wx < mapWidth {
				idx := positionToIndex(wx, connY, mapWidth)
				if idx >= 0 && idx < len(result.Tiles) && !result.Tiles[idx].Blocked {
					setTileWall(result, wx, connY, mapWidth, images)
				}
			}
		}
	}
}

// garrisonOverlapsAny checks if candidate rect overlaps any placed room
// including padding for corridor space.
func garrisonOverlapsAny(candidate Rect, placed map[int]Rect, padding int) bool {
	for _, r := range placed {
		if !(candidate.X2+padding < r.X1 || r.X2+padding < candidate.X1 ||
			candidate.Y2+padding < r.Y1 || r.Y2+padding < candidate.Y1) {
			return true
		}
	}
	return false
}

func init() {
	RegisterGenerator(&GarrisonRaidGenerator{config: DefaultGarrisonConfig()})
}
