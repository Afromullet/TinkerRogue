package worldmap

import (
	"game_main/common"
	"game_main/coords"
	"game_main/graphics"
)

// RoomsAndCorridorsGenerator implements the classic roguelike generation
type RoomsAndCorridorsGenerator struct {
	config GeneratorConfig
}

// NewRoomsAndCorridorsGenerator creates a new rooms-and-corridors generator
func NewRoomsAndCorridorsGenerator(config GeneratorConfig) *RoomsAndCorridorsGenerator {
	return &RoomsAndCorridorsGenerator{config: config}
}

func (g *RoomsAndCorridorsGenerator) Name() string {
	return "rooms_corridors"
}

func (g *RoomsAndCorridorsGenerator) Description() string {
	return "Classic roguelike: rectangular rooms connected by L-shaped corridors"
}

func (g *RoomsAndCorridorsGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	result := GenerationResult{
		Tiles:          g.createEmptyTiles(width, height, images),
		Rooms:          make([]Rect, 0, g.config.MaxRooms),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	// Generate rooms with collision detection
	for idx := 0; idx < g.config.MaxRooms; idx++ {
		room := g.generateRandomRoom(width, height)

		if g.canPlaceRoom(room, result.Rooms) {
			g.carveRoom(&result, room, images)

			// Connect to previous room if not the first
			if len(result.Rooms) > 0 {
				prevRoom := result.Rooms[len(result.Rooms)-1]
				g.connectRooms(&result, prevRoom, room, images)
			}

			result.Rooms = append(result.Rooms, room)
		}
	}

	return result
}

// createEmptyTiles initializes all tiles as walls
func (g *RoomsAndCorridorsGenerator) createEmptyTiles(width, height int, images TileImageSet) []*Tile {
	tiles := make([]*Tile, width*height)

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)

			wallImg := images.WallImages[common.GetRandomBetween(0, len(images.WallImages)-1)]
			tile := NewTile(
				x*graphics.ScreenInfo.TileSize,
				y*graphics.ScreenInfo.TileSize,
				logicalPos, true, wallImg, WALL, false,
			)
			tiles[index] = &tile
		}
	}

	return tiles
}

// generateRandomRoom creates a room with random size and position
func (g *RoomsAndCorridorsGenerator) generateRandomRoom(mapWidth, mapHeight int) Rect {
	w := common.GetRandomBetween(g.config.MinRoomSize, g.config.MaxRoomSize)
	h := common.GetRandomBetween(g.config.MinRoomSize, g.config.MaxRoomSize)
	x := common.GetDiceRoll(mapWidth - w - 1)
	y := common.GetDiceRoll(mapHeight - h - 1)
	return NewRect(x, y, w, h)
}

// canPlaceRoom checks if room overlaps with any existing rooms
func (g *RoomsAndCorridorsGenerator) canPlaceRoom(room Rect, existing []Rect) bool {
	for _, other := range existing {
		if room.Intersect(other) {
			return false
		}
	}
	return true
}

// carveRoom converts wall tiles to floor tiles within room bounds
func (g *RoomsAndCorridorsGenerator) carveRoom(result *GenerationResult, room Rect, images TileImageSet) {
	for y := room.Y1 + 1; y < room.Y2; y++ {
		for x := room.X1 + 1; x < room.X2; x++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)

			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			result.Tiles[index].image = images.FloorImages[common.GetRandomBetween(0, len(images.FloorImages)-1)]

			// Add to valid positions
			result.ValidPositions = append(result.ValidPositions, logicalPos)
		}
	}
}

// connectRooms creates L-shaped corridor between two rooms
func (g *RoomsAndCorridorsGenerator) connectRooms(result *GenerationResult, room1, room2 Rect, images TileImageSet) {
	x1, y1 := room1.Center()
	x2, y2 := room2.Center()

	// Randomly choose L-shape orientation
	if common.GetDiceRoll(2) == 2 {
		g.carveHorizontalTunnel(result, x1, x2, y1, images)
		g.carveVerticalTunnel(result, y1, y2, x2, images)
	} else {
		g.carveVerticalTunnel(result, y1, y2, x1, images)
		g.carveHorizontalTunnel(result, x1, x2, y2, images)
	}
}

// carveHorizontalTunnel creates horizontal corridor
func (g *RoomsAndCorridorsGenerator) carveHorizontalTunnel(result *GenerationResult, x1, x2, y int, images TileImageSet) {
	for x := min(x1, x2); x <= max(x1, x2); x++ {
		logicalPos := coords.LogicalPosition{X: x, Y: y}
		index := coords.CoordManager.LogicalToIndex(logicalPos)

		if index >= 0 && index < len(result.Tiles) {
			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			result.Tiles[index].image = images.FloorImages[common.GetRandomBetween(0, len(images.FloorImages)-1)]
			result.ValidPositions = append(result.ValidPositions, logicalPos)
		}
	}
}

// carveVerticalTunnel creates vertical corridor
func (g *RoomsAndCorridorsGenerator) carveVerticalTunnel(result *GenerationResult, y1, y2, x int, images TileImageSet) {
	for y := min(y1, y2); y <= max(y1, y2); y++ {
		logicalPos := coords.LogicalPosition{X: x, Y: y}
		index := coords.CoordManager.LogicalToIndex(logicalPos)

		if index >= 0 && index < len(result.Tiles) {
			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			result.Tiles[index].image = images.FloorImages[common.GetRandomBetween(0, len(images.FloorImages)-1)]
			result.ValidPositions = append(result.ValidPositions, logicalPos)
		}
	}
}

// Register this generator on package initialization
func init() {
	RegisterGenerator(NewRoomsAndCorridorsGenerator(DefaultConfig()))
}
