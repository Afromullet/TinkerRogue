package worldgen

import (
	"game_main/common"
	"game_main/world/coords"
	"game_main/world/worldmapcore"
)

// RoomsCorridorsConfig holds parameters for the rooms-and-corridors generator
type RoomsCorridorsConfig struct {
	MinRoomSize int
	MaxRoomSize int
	MaxRooms    int
	Seed        int64 // 0 = use time-based seed
}

// DefaultRoomsCorridorsConfig returns sensible defaults for rooms-and-corridors generation
func DefaultRoomsCorridorsConfig() RoomsCorridorsConfig {
	return RoomsCorridorsConfig{
		MinRoomSize: 6,
		MaxRoomSize: 10,
		MaxRooms:    30,
		Seed:        0,
	}
}

// RoomsAndCorridorsGenerator implements the classic roguelike generation
type RoomsAndCorridorsGenerator struct {
	config RoomsCorridorsConfig
}

// NewRoomsAndCorridorsGenerator creates a new rooms-and-corridors generator
func NewRoomsAndCorridorsGenerator(config RoomsCorridorsConfig) *RoomsAndCorridorsGenerator {
	return &RoomsAndCorridorsGenerator{config: config}
}

func (g *RoomsAndCorridorsGenerator) Name() string {
	return "rooms_corridors"
}

func (g *RoomsAndCorridorsGenerator) Description() string {
	return "Classic roguelike: rectangular rooms connected by L-shaped corridors"
}

func (g *RoomsAndCorridorsGenerator) Generate(width, height int, images worldmapcore.TileImageSet) worldmapcore.GenerationResult {
	if g.config.Seed != 0 {
		common.SetRNGSeed(uint64(g.config.Seed), uint64(g.config.Seed))
	}

	result := worldmapcore.GenerationResult{
		Tiles:          CreateEmptyTiles(width, height, images),
		Rooms:          make([]worldmapcore.Rect, 0, g.config.MaxRooms),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	// Generate rooms with collision detection
	for idx := 0; idx < g.config.MaxRooms; idx++ {
		room := g.generateRandomRoom(width, height)

		if g.canPlaceRoom(room, result.Rooms) {
			CarveRoom(&result, room, width, images)

			// Connect to previous room if not the first
			if len(result.Rooms) > 0 {
				prevRoom := result.Rooms[len(result.Rooms)-1]
				g.connectRooms(&result, prevRoom, room, width, images)
			}

			result.Rooms = append(result.Rooms, room)
		}
	}

	return result
}

// generateRandomRoom creates a room with random size and position
func (g *RoomsAndCorridorsGenerator) generateRandomRoom(mapWidth, mapHeight int) worldmapcore.Rect {
	w := common.GetRandomBetween(g.config.MinRoomSize, g.config.MaxRoomSize)
	h := common.GetRandomBetween(g.config.MinRoomSize, g.config.MaxRoomSize)
	x := common.GetDiceRoll(mapWidth - w - 1)
	y := common.GetDiceRoll(mapHeight - h - 1)
	return worldmapcore.NewRect(x, y, w, h)
}

// canPlaceRoom checks if room overlaps with any existing rooms
func (g *RoomsAndCorridorsGenerator) canPlaceRoom(room worldmapcore.Rect, existing []worldmapcore.Rect) bool {
	for _, other := range existing {
		if room.Intersect(other) {
			return false
		}
	}
	return true
}

// connectRooms creates L-shaped corridor between two rooms
func (g *RoomsAndCorridorsGenerator) connectRooms(result *worldmapcore.GenerationResult, room1, room2 worldmapcore.Rect, width int, images worldmapcore.TileImageSet) {
	x1, y1 := room1.Center()
	x2, y2 := room2.Center()

	// Randomly choose L-shape orientation
	if common.GetDiceRoll(2) == 2 {
		CarveHorizontalTunnel(result, x1, x2, y1, width, images)
		CarveVerticalTunnel(result, y1, y2, x2, width, images)
	} else {
		CarveVerticalTunnel(result, y1, y2, x1, width, images)
		CarveHorizontalTunnel(result, x1, x2, y2, width, images)
	}
}

// Register this generator on package initialization
func init() {
	RegisterGenerator(NewRoomsAndCorridorsGenerator(DefaultRoomsCorridorsConfig()))
}
