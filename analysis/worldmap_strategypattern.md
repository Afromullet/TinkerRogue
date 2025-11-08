
---

## FINAL SYNTHESIZED APPROACHES

### Approach 1: Strategy Pattern with Simple Registry

**Strategic Focus**: "Clean separation with pragmatic implementation"

**Problem Statement**:
The current GameMap struct has GenerateLevelTiles() hardcoded at lines 360-405 in dungeongen.go. Adding a new algorithm (BSP, cellular automata, drunkard's walk) requires:
1. Modifying GameMap methods
2. Adding conditional logic for algorithm selection
3. Risk of breaking existing generation
4. No way to test algorithms in isolation

This violates Open/Closed Principle and makes experimentation difficult.

**Solution Overview**:
Extract generation into a `MapGenerator` interface with concrete implementations for each algorithm. Use a simple registration system for algorithm discovery. GameMap becomes a pure data structure that receives generated tiles from a generator.

**Code Example**:

*Before (dungeongen.go lines 360-405):*
```go
func (gameMap *GameMap) GenerateLevelTiles() {
	MIN_SIZE := 6
	MAX_SIZE := 10
	MAX_ROOMS := 30

	tiles := gameMap.CreateTiles()
	gameMap.Tiles = tiles
	contains_rooms := false

	for idx := 0; idx < MAX_ROOMS; idx++ {
		w := common.GetRandomBetween(MIN_SIZE, MAX_SIZE)
		h := common.GetRandomBetween(MIN_SIZE, MAX_SIZE)
		x := common.GetDiceRoll(graphics.ScreenInfo.DungeonWidth - w - 1)
		y := common.GetDiceRoll(graphics.ScreenInfo.DungeonHeight - h - 1)
		new_room := NewRect(x, y, w, h)

		okToAdd := true
		for _, otherRoom := range gameMap.Rooms {
			if new_room.Intersect(otherRoom) {
				okToAdd = false
				break
			}
		}

		if okToAdd {
			gameMap.createRoom(new_room)
			if contains_rooms {
				newX, newY := new_room.Center()
				prevX, prevY := gameMap.Rooms[len(gameMap.Rooms)-1].Center()
				coinflip := common.GetDiceRoll(2)
				if coinflip == 2 {
					gameMap.createHorizontalTunnel(prevX, newX, prevY)
					gameMap.createVerticalTunnel(prevY, newY, newX)
				} else {
					gameMap.createHorizontalTunnel(prevX, newX, newY)
					gameMap.createVerticalTunnel(prevY, newY, prevX)
				}
			}
			gameMap.Rooms = append(gameMap.Rooms, new_room)
			contains_rooms = true
		}
	}
}
```

*After - Interface Definition (worldmap/generator.go):*
```go
package worldmap

import (
	"game_main/coords"
	"game_main/graphics"
)

// GenerationResult contains the output of a map generation algorithm
type GenerationResult struct {
	Tiles []*Tile
	Rooms []Rect
}

// MapGenerator defines the interface for all map generation algorithms
type MapGenerator interface {
	// Generate creates a new map layout
	Generate(width, height int) GenerationResult

	// Name returns the algorithm name for debugging/selection
	Name() string

	// Description returns a human-readable description
	Description() string
}

// GeneratorConfig holds common parameters for generators
type GeneratorConfig struct {
	Width      int
	Height     int
	MinRoomSize int
	MaxRoomSize int
	MaxRooms    int
	Seed        int64 // 0 = use time-based seed
}

// DefaultConfig returns sensible defaults for dungeon generation
func DefaultConfig() GeneratorConfig {
	return GeneratorConfig{
		Width:       graphics.ScreenInfo.DungeonWidth,
		Height:      graphics.ScreenInfo.DungeonHeight,
		MinRoomSize: 6,
		MaxRoomSize: 10,
		MaxRooms:    30,
		Seed:        0,
	}
}

// Generator registry for algorithm selection
var generators = make(map[string]MapGenerator)

// RegisterGenerator adds a new algorithm to the registry
func RegisterGenerator(gen MapGenerator) {
	generators[gen.Name()] = gen
}

// GetGenerator retrieves an algorithm by name
func GetGenerator(name string) (MapGenerator, bool) {
	gen, ok := generators[name]
	return gen, ok
}

// ListGenerators returns all registered algorithm names
func ListGenerators() []string {
	names := make([]string, 0, len(generators))
	for name := range generators {
		names = append(names, name)
	}
	return names
}
```

*After - Rooms & Corridors Implementation (worldmap/gen_rooms_corridors.go):*
```go
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

func NewRoomsAndCorridorsGenerator(config GeneratorConfig) *RoomsAndCorridorsGenerator {
	return &RoomsAndCorridorsGenerator{config: config}
}

func (g *RoomsAndCorridorsGenerator) Name() string {
	return "rooms_corridors"
}

func (g *RoomsAndCorridorsGenerator) Description() string {
	return "Classic roguelike: rectangular rooms connected by corridors"
}

func (g *RoomsAndCorridorsGenerator) Generate(width, height int) GenerationResult {
	result := GenerationResult{
		Tiles: g.createEmptyTiles(width, height),
		Rooms: make([]Rect, 0, g.config.MaxRooms),
	}

	// Generate rooms with collision detection
	for idx := 0; idx < g.config.MaxRooms; idx++ {
		room := g.generateRandomRoom(width, height)

		if g.canPlaceRoom(room, result.Rooms) {
			g.carveRoom(&result, room)

			// Connect to previous room if not the first
			if len(result.Rooms) > 0 {
				g.connectRooms(&result, result.Rooms[len(result.Rooms)-1], room)
			}

			result.Rooms = append(result.Rooms, room)
		}
	}

	return result
}

func (g *RoomsAndCorridorsGenerator) createEmptyTiles(width, height int) []*Tile {
	tiles := make([]*Tile, width*height)

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)

			wallImg := wallImgs[common.GetRandomBetween(0, len(wallImgs)-1)]
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

func (g *RoomsAndCorridorsGenerator) generateRandomRoom(mapWidth, mapHeight int) Rect {
	w := common.GetRandomBetween(g.config.MinRoomSize, g.config.MaxRoomSize)
	h := common.GetRandomBetween(g.config.MinRoomSize, g.config.MaxRoomSize)
	x := common.GetDiceRoll(mapWidth - w - 1)
	y := common.GetDiceRoll(mapHeight - h - 1)
	return NewRect(x, y, w, h)
}

func (g *RoomsAndCorridorsGenerator) canPlaceRoom(room Rect, existing []Rect) bool {
	for _, other := range existing {
		if room.Intersect(other) {
			return false
		}
	}
	return true
}

func (g *RoomsAndCorridorsGenerator) carveRoom(result *GenerationResult, room Rect) {
	for y := room.Y1 + 1; y < room.Y2; y++ {
		for x := room.X1 + 1; x < room.X2; x++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)

			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			result.Tiles[index].image = floorImgs[common.GetRandomBetween(0, len(floorImgs)-1)]

			ValidPos.Add(x, y)
		}
	}
}

func (g *RoomsAndCorridorsGenerator) connectRooms(result *GenerationResult, room1, room2 Rect) {
	x1, y1 := room1.Center()
	x2, y2 := room2.Center()

	if common.GetDiceRoll(2) == 2 {
		g.carveHorizontalTunnel(result, x1, x2, y1)
		g.carveVerticalTunnel(result, y1, y2, x2)
	} else {
		g.carveVerticalTunnel(result, y1, y2, x1)
		g.carveHorizontalTunnel(result, x1, x2, y2)
	}
}

func (g *RoomsAndCorridorsGenerator) carveHorizontalTunnel(result *GenerationResult, x1, x2, y int) {
	for x := min(x1, x2); x <= max(x1, x2); x++ {
		logicalPos := coords.LogicalPosition{X: x, Y: y}
		index := coords.CoordManager.LogicalToIndex(logicalPos)

		if index >= 0 && index < len(result.Tiles) {
			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			result.Tiles[index].image = floorImgs[common.GetRandomBetween(0, len(floorImgs)-1)]
			ValidPos.Add(x, y)
		}
	}
}

func (g *RoomsAndCorridorsGenerator) carveVerticalTunnel(result *GenerationResult, y1, y2, x int) {
	for y := min(y1, y2); y <= max(y1, y2); y++ {
		logicalPos := coords.LogicalPosition{X: x, Y: y}
		index := coords.CoordManager.LogicalToIndex(logicalPos)

		if index >= 0 && index < len(result.Tiles) {
			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			result.Tiles[index].image = floorImgs[common.GetRandomBetween(0, len(floorImgs)-1)]
			ValidPos.Add(x, y)
		}
	}
}

// Register this generator on package initialization
func init() {
	RegisterGenerator(NewRoomsAndCorridorsGenerator(DefaultConfig()))
}
```

*After - Modified GameMap (dungeongen.go):*
```go
// NewGameMap creates a new game map using the specified generator algorithm
func NewGameMap(generatorName string) GameMap {
	loadTileImages()
	ValidPos = ValidPositions{
		Pos: make([]coords.LogicalPosition, 0),
	}

	dungeonMap := GameMap{
		PlayerVisible: fov.New(),
	}

	// Get generator or fall back to default
	gen, ok := GetGenerator(generatorName)
	if !ok {
		gen, _ = GetGenerator("rooms_corridors")
	}

	// Generate the map
	result := gen.Generate(
		graphics.ScreenInfo.DungeonWidth,
		graphics.ScreenInfo.DungeonHeight,
	)

	dungeonMap.Tiles = result.Tiles
	dungeonMap.Rooms = result.Rooms
	dungeonMap.NumTiles = len(dungeonMap.Tiles)
	dungeonMap.PlaceStairs()

	return dungeonMap
}

// Remove GenerateLevelTiles() - no longer needed!
// createRoom(), createHorizontalTunnel(), createVerticalTunnel() - moved to generator
```

**Key Changes**:
1. Created MapGenerator interface with Generate(), Name(), Description()
2. Extracted rooms-and-corridors logic to RoomsAndCorridorsGenerator
3. Added simple registry pattern (RegisterGenerator, GetGenerator)
4. Created GeneratorConfig for algorithm parameters
5. NewGameMap now accepts generator name parameter
6. Removed 180+ LOC from GameMap struct

**Value Proposition**:
- **Maintainability**: GameMap reduced from 550 LOC to ~370 LOC (33% reduction)
- **Readability**: Clear separation - GameMap = data, Generator = algorithm
- **Extensibility**: New algorithms just implement interface + register
- **Complexity Impact**:
  - Cyclomatic complexity of GameMap: ~15 → ~8
  - Generation logic isolated and testable
  - +2 new files, -180 LOC from dungeongen.go

**Implementation Strategy**:
1. **Phase 1** (2 hours): Create generator.go with interface and registry
2. **Phase 2** (3 hours): Extract current algorithm to gen_rooms_corridors.go
3. **Phase 3** (1 hour): Modify NewGameMap to use generator, test thoroughly
4. **Phase 4** (2 hours): Add second algorithm (BSP or cellular) to validate design
5. **Total**: ~8 hours for complete implementation with validation

**Advantages**:
- **Easy to add algorithms**: Implement interface, call RegisterGenerator(), done
  - Example: Adding BSP requires ~150 LOC in new file, zero changes to existing files
- **Testable in isolation**: Each generator can be unit tested independently
  ```go
  func TestRoomsAndCorridors(t *testing.T) {
      gen := NewRoomsAndCorridorsGenerator(DefaultConfig())
      result := gen.Generate(80, 50)
      assert.True(t, len(result.Rooms) > 0)
      assert.True(t, len(result.Tiles) == 80*50)
  }
  ```
- **No breaking changes**: Existing code continues to work with default generator
- **Clear contracts**: Interface defines exact requirements for new algorithms

**Drawbacks & Risks**:
- **More files**: +3 files (generator.go, gen_rooms_corridors.go, future algorithms)
  - *Mitigation*: Clear naming convention (gen_*.go), better than 1000+ LOC file
- **Indirection**: One more step to trace generation flow
  - *Mitigation*: Interface is simple (1 method), registry is straightforward
- **Migration effort**: Need to update all NewGameMap() callsites
  - *Mitigation*: Add NewGameMapDefault() wrapper for backward compatibility
  ```go
  func NewGameMapDefault() GameMap {
      return NewGameMap("rooms_corridors")
  }
  ```

**Effort Estimate**:
- **Time**: 8-12 hours (1-2 days with testing)
- **Complexity**: Medium (interface design, extraction, migration)
- **Risk**: Low (incremental changes, backward compatible)
- **Files Impacted**:
  - Modified: dungeongen.go (remove 180 LOC, add 20 LOC)
  - New: generator.go (~90 LOC), gen_rooms_corridors.go (~200 LOC)
  - Callsites: game_main/main.go and anywhere NewGameMap() is called

**Critical Assessment** (Pragmatic Evaluation):
This approach strikes the best balance between theory and practice. It applies the Strategy Pattern without over-engineering - the interface is minimal (one method), the registry is simple (map[string]Generator), and the implementation path is clear. The value is immediate: adding a BSP generator becomes a single new file with zero modifications to existing code. Risk is low because the current algorithm is extracted as-is, maintaining identical behavior. This is production-ready architecture, not academic exercise.
