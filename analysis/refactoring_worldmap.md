# Refactoring Analysis: Worldmap Generation System
Generated: 2025-11-08 06:46:52
Target: worldmap/ package (4 files, ~919 LOC)

## EXECUTIVE SUMMARY

### Target Analysis
- **Scope**: Worldmap package refactoring to support multiple map generation algorithms
- **Current State**: Single hardcoded rooms-and-corridors algorithm tightly coupled to GameMap struct
- **Primary Issues**:
  1. **Zero extensibility** - Adding new algorithms requires modifying GameMap
  2. **Tight coupling** - Generation logic embedded in data structure
  3. **Mixed concerns** - GameMap handles generation, storage, rendering, entity management, FOV
  4. **Monolithic function** - GenerateLevelTiles() (46 LOC) does everything
  5. **ECS violations** - TileContents uses entity pointers instead of EntityIDs
- **Recommended Direction**: Strategy Pattern with Simple Registry (Approach 1) for best balance of flexibility and pragmatism

### Quick Wins vs Strategic Refactoring
- **Immediate Improvements** (4-6 hours): Extract current algorithm to separate file, create config struct
- **Medium-Term Goals** (1-2 days): Implement generator interface, add second algorithm (BSP or cellular automata)
- **Long-Term Architecture** (3-5 days): Full separation of concerns, composable generation primitives, ECS compliance

### Consensus Findings
- **Agreement Across Perspectives**:
  - Current implementation prevents adding new algorithms without GameMap modifications
  - Generation should be separated from map data structure
  - Interface-based approach enables testing and flexibility
  - ECS violations in TileContents need fixing
- **Divergent Perspectives**:
  - Architectural view favors comprehensive strategy pattern
  - Game-specific view favors simpler registry with minimal abstraction
  - Pragmatic view favors incremental extraction with proof of concept first
- **Critical Concerns**:
  - Over-engineering risk if full strategy pattern implemented without proven need
  - Performance impact minimal (generation happens once per level)
  - Breaking changes to GameMap could affect existing systems (FOV, pathfinding, rendering)
  - Need to maintain backward compatibility during transition

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

---

### Approach 2: Functional Composition with Generation Primitives

**Strategic Focus**: "Go-idiomatic simplicity with composable building blocks"

**Problem Statement**:
Beyond just supporting different algorithms, the current monolithic approach makes it hard to:
- Reuse generation logic (room placement, corridor carving)
- Mix and match techniques (BSP room placement + drunkard's walk corridors)
- Test individual generation steps
- Create hybrid algorithms for variety

Roguelike generation often combines techniques (BSP for layout + cellular for caverns + prefabs for special rooms). The current structure prevents this composition.

**Solution Overview**:
Use pure functions for generation primitives (PlaceRooms, CarveCorridors, PlacePrefabs) that operate on map state. Generators become function compositions. This is more Go-idiomatic than OOP interfaces and enables powerful combinations.

**Code Example**:

*After - Primitive Functions (worldmap/genprimitives.go):*
```go
package worldmap

import (
	"game_main/common"
	"game_main/coords"
)

// MapState represents the mutable state during generation
type MapState struct {
	Tiles  []*Tile
	Rooms  []Rect
	Width  int
	Height int
}

// NewMapState creates initial empty map state
func NewMapState(width, height int) *MapState {
	state := &MapState{
		Tiles:  make([]*Tile, width*height),
		Rooms:  make([]Rect, 0),
		Width:  width,
		Height: height,
	}

	// Initialize all tiles as walls
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)
			tile := NewTile(x*16, y*16, logicalPos, true, wallImgs[0], WALL, false)
			state.Tiles[index] = &tile
		}
	}

	return state
}

// GeneratorFunc is a function that modifies map state
type GeneratorFunc func(*MapState)

// Compose combines multiple generator functions into one
func Compose(fns ...GeneratorFunc) GeneratorFunc {
	return func(state *MapState) {
		for _, fn := range fns {
			fn(state)
		}
	}
}

// PlaceRoomsRandomly adds non-overlapping rectangular rooms
func PlaceRoomsRandomly(minSize, maxSize, maxRooms int) GeneratorFunc {
	return func(state *MapState) {
		for i := 0; i < maxRooms; i++ {
			room := generateRandomRoom(state.Width, state.Height, minSize, maxSize)

			// Check collision
			canPlace := true
			for _, existing := range state.Rooms {
				if room.Intersect(existing) {
					canPlace = false
					break
				}
			}

			if canPlace {
				carveRoom(state, room)
				state.Rooms = append(state.Rooms, room)
			}
		}
	}
}

// ConnectRoomsWithCorridors creates L-shaped corridors between all rooms
func ConnectRoomsWithCorridors() GeneratorFunc {
	return func(state *MapState) {
		for i := 1; i < len(state.Rooms); i++ {
			x1, y1 := state.Rooms[i-1].Center()
			x2, y2 := state.Rooms[i].Center()

			if common.GetDiceRoll(2) == 1 {
				carveHorizontalCorridor(state, x1, x2, y1)
				carveVerticalCorridor(state, y1, y2, x2)
			} else {
				carveVerticalCorridor(state, y1, y2, x1)
				carveHorizontalCorridor(state, x1, x2, y2)
			}
		}
	}
}

// AddCellularCaverns uses cellular automata to add natural caves
func AddCellularCaverns(iterations int, threshold float64) GeneratorFunc {
	return func(state *MapState) {
		// Initial random noise
		for i := range state.Tiles {
			if common.GetRandomBetween(0, 100) < 45 && state.Tiles[i].TileType == WALL {
				state.Tiles[i].TileType = FLOOR
				state.Tiles[i].Blocked = false
			}
		}

		// Cellular automata iterations
		for iter := 0; iter < iterations; iter++ {
			newTiles := make([]TileType, len(state.Tiles))
			copy(newTiles, extractTypes(state.Tiles))

			for x := 1; x < state.Width-1; x++ {
				for y := 1; y < state.Height-1; y++ {
					wallCount := countNeighborWalls(state, x, y)
					idx := coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: x, Y: y})

					if wallCount >= 5 {
						newTiles[idx] = WALL
					} else {
						newTiles[idx] = FLOOR
					}
				}
			}

			applyTypes(state.Tiles, newTiles)
		}
	}
}

// PlacePrefab places a pre-designed room at a random location
func PlacePrefab(prefab [][]TileType) GeneratorFunc {
	return func(state *MapState) {
		// Find random valid placement
		x := common.GetRandomBetween(1, state.Width-len(prefab[0])-1)
		y := common.GetRandomBetween(1, state.Height-len(prefab)-1)

		// Stamp prefab onto map
		for py := 0; py < len(prefab); py++ {
			for px := 0; px < len(prefab[py]); px++ {
				logicalPos := coords.LogicalPosition{X: x + px, Y: y + py}
				idx := coords.CoordManager.LogicalToIndex(logicalPos)
				state.Tiles[idx].TileType = prefab[py][px]
				state.Tiles[idx].Blocked = (prefab[py][px] == WALL)
			}
		}
	}
}

// Helper functions
func generateRandomRoom(mapWidth, mapHeight, minSize, maxSize int) Rect {
	w := common.GetRandomBetween(minSize, maxSize)
	h := common.GetRandomBetween(minSize, maxSize)
	x := common.GetDiceRoll(mapWidth - w - 1)
	y := common.GetDiceRoll(mapHeight - h - 1)
	return NewRect(x, y, w, h)
}

func carveRoom(state *MapState, room Rect) {
	for y := room.Y1 + 1; y < room.Y2; y++ {
		for x := room.X1 + 1; x < room.X2; x++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)
			state.Tiles[index].TileType = FLOOR
			state.Tiles[index].Blocked = false
			state.Tiles[index].image = floorImgs[common.GetRandomBetween(0, len(floorImgs)-1)]
		}
	}
}

func carveHorizontalCorridor(state *MapState, x1, x2, y int) {
	for x := min(x1, x2); x <= max(x1, x2); x++ {
		idx := coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: x, Y: y})
		if idx >= 0 && idx < len(state.Tiles) {
			state.Tiles[idx].TileType = FLOOR
			state.Tiles[idx].Blocked = false
		}
	}
}

func carveVerticalCorridor(state *MapState, y1, y2, x int) {
	for y := min(y1, y2); y <= max(y1, y2); y++ {
		idx := coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: x, Y: y})
		if idx >= 0 && idx < len(state.Tiles) {
			state.Tiles[idx].TileType = FLOOR
			state.Tiles[idx].Blocked = false
		}
	}
}

func countNeighborWalls(state *MapState, x, y int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			idx := coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: x + dx, Y: y + dy})
			if state.Tiles[idx].TileType == WALL {
				count++
			}
		}
	}
	return count
}
```

*After - Composed Algorithms (worldmap/gencomposed.go):*
```go
package worldmap

// Classic rooms and corridors using composition
func GenerateRoomsAndCorridors(width, height int) GenerationResult {
	state := NewMapState(width, height)

	generator := Compose(
		PlaceRoomsRandomly(6, 10, 30),
		ConnectRoomsWithCorridors(),
	)

	generator(state)

	return GenerationResult{
		Tiles: state.Tiles,
		Rooms: state.Rooms,
	}
}

// Hybrid: BSP rooms + drunkard's walk corridors + cellular caverns
func GenerateHybridDungeon(width, height int) GenerationResult {
	state := NewMapState(width, height)

	generator := Compose(
		PlaceRoomsBSP(5, 15),           // BSP for structured layout
		AddCellularCaverns(4, 0.5),     // Natural caves
		ConnectRoomsWithCorridors(),     // Ensure connectivity
		PlacePrefab(treasureRoomPrefab), // Special room
	)

	generator(state)

	return GenerationResult{
		Tiles: state.Tiles,
		Rooms: state.Rooms,
	}
}

// Pure caves using cellular automata
func GenerateCaves(width, height int) GenerationResult {
	state := NewMapState(width, height)

	generator := Compose(
		AddCellularCaverns(5, 0.45),
		EnsureConnectivity(), // Make sure all areas are reachable
	)

	generator(state)

	return GenerationResult{
		Tiles: state.Tiles,
		Rooms: []Rect{}, // No formal rooms in caves
	}
}
```

**Key Changes**:
1. Generators are pure functions: `func(*MapState)`
2. Compose() combines primitives into complex algorithms
3. Each primitive is independently testable
4. Mix and match for variety (rooms + caves + prefabs)
5. No interfaces, just functions - very Go-idiomatic

**Value Proposition**:
- **Maintainability**: Small, focused functions (15-30 LOC each) vs monolithic 46 LOC
- **Readability**: Algorithm is visible as function composition
- **Extensibility**: New primitives = new functions, new algorithms = new compositions
- **Complexity Impact**:
  - Each primitive has cyclomatic complexity ~3-5 (very simple)
  - Algorithms self-document through composition
  - Testing: Each primitive testable in isolation

**Implementation Strategy**:
1. **Phase 1** (3 hours): Create genprimitives.go with core primitives (PlaceRooms, ConnectRooms)
2. **Phase 2** (2 hours): Implement Compose() and MapState
3. **Phase 3** (2 hours): Create gencomposed.go with 2-3 example algorithms
4. **Phase 4** (2 hours): Add advanced primitives (cellular, BSP, prefabs)
5. **Phase 5** (1 hour): Update NewGameMap to use composed generators
6. **Total**: ~10 hours for full implementation

**Advantages**:
- **Maximum flexibility**: Mix any primitives in any order
  ```go
  // Want rooms + natural caves + special vaults?
  generator := Compose(
      PlaceRoomsRandomly(6, 10, 20),
      AddCellularCaverns(3, 0.4),
      ConnectRoomsWithCorridors(),
      PlacePrefab(bossRoomPrefab),
  )
  ```
- **Easy testing**: Each primitive is 15-30 LOC, easily unit testable
- **No boilerplate**: No interface implementations, just write functions
- **Performance**: Function calls are cheap, no virtual dispatch overhead

**Drawbacks & Risks**:
- **State management**: MapState is mutable, must be passed carefully
  - *Mitigation*: Document that GeneratorFunc modifies state in-place
- **Composition order matters**: PlaceRooms before ConnectRooms, not after
  - *Mitigation*: Clear naming and documentation of primitives
- **Learning curve**: Developers must understand composition pattern
  - *Mitigation*: Provide 3-4 example composed algorithms as templates

**Effort Estimate**:
- **Time**: 10-14 hours (2 days with testing and examples)
- **Complexity**: Medium-High (functional approach, state management)
- **Risk**: Medium (new pattern for the codebase, requires careful state handling)
- **Files Impacted**:
  - New: genprimitives.go (~300 LOC), gencomposed.go (~150 LOC)
  - Modified: dungeongen.go (replace GenerateLevelTiles with composition)

**Critical Assessment** (Pragmatic Evaluation):
This approach is more powerful than Approach 1 but also more complex. It's ideal if you anticipate needing hybrid algorithms (rooms + caves, BSP + cellular). The functional style is very Go-idiomatic and enables powerful compositions. However, the mutable MapState and the importance of composition order introduce complexity. Best suited for teams comfortable with functional programming patterns. If you only need to swap complete algorithms (not mix primitives), Approach 1 is simpler. If you want maximum generation variety and are willing to invest in the pattern, this is superior.

---

### Approach 3: Incremental Extraction (Minimal Risk, Proof of Concept)

**Strategic Focus**: "Prove the concept first, refactor later"

**Problem Statement**:
Both Approach 1 and 2 require significant upfront design. What if the interface is wrong? What if we don't actually need multiple algorithms? The risk of over-engineering is real.

The immediate problem is: **we can't test generation in isolation or experiment with parameters without modifying GameMap**. We don't need full strategy pattern yet - we need separation as a first step.

**Solution Overview**:
Extract the current generation logic to a standalone function with a config struct, but keep everything in the same file initially. This proves the separation works, enables testing, and sets up future interface extraction with zero risk.

**Code Example**:

*After - Extracted with Config (dungeongen.go):*
```go
package worldmap

// GenerationConfig holds parameters for dungeon generation
type GenerationConfig struct {
	Width       int
	Height      int
	MinRoomSize int
	MaxRoomSize int
	MaxRooms    int
}

// DefaultGenerationConfig returns sensible defaults
func DefaultGenerationConfig() GenerationConfig {
	return GenerationConfig{
		Width:       graphics.ScreenInfo.DungeonWidth,
		Height:      graphics.ScreenInfo.DungeonHeight,
		MinRoomSize: 6,
		MaxRoomSize: 10,
		MaxRooms:    30,
	}
}

// GenerateTilesRoomsAndCorridors creates a classic roguelike dungeon
// Returns tiles and rooms separately from GameMap
func GenerateTilesRoomsAndCorridors(config GenerationConfig) ([]*Tile, []Rect) {
	tiles := createEmptyTiles(config.Width, config.Height)
	rooms := make([]Rect, 0, config.MaxRooms)

	for idx := 0; idx < config.MaxRooms; idx++ {
		w := common.GetRandomBetween(config.MinRoomSize, config.MaxRoomSize)
		h := common.GetRandomBetween(config.MinRoomSize, config.MaxRoomSize)
		x := common.GetDiceRoll(config.Width - w - 1)
		y := common.GetDiceRoll(config.Height - h - 1)
		newRoom := NewRect(x, y, w, h)

		// Check intersection
		okToAdd := true
		for _, otherRoom := range rooms {
			if newRoom.Intersect(otherRoom) {
				okToAdd = false
				break
			}
		}

		if okToAdd {
			carveRoomIntoTiles(tiles, newRoom)

			// Connect to previous room
			if len(rooms) > 0 {
				connectRoomsInTiles(tiles, rooms[len(rooms)-1], newRoom)
			}

			rooms = append(rooms, newRoom)
		}
	}

	return tiles, rooms
}

// Helper: Create empty tile array (extracted for reuse)
func createEmptyTiles(width, height int) []*Tile {
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

// Helper: Carve room into existing tiles
func carveRoomIntoTiles(tiles []*Tile, room Rect) {
	for y := room.Y1 + 1; y < room.Y2; y++ {
		for x := room.X1 + 1; x < room.X2; x++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)
			tiles[index].Blocked = false
			tiles[index].TileType = FLOOR
			tiles[index].image = floorImgs[common.GetRandomBetween(0, len(floorImgs)-1)]
			ValidPos.Add(x, y)
		}
	}
}

// Helper: Connect two rooms with corridors
func connectRoomsInTiles(tiles []*Tile, room1, room2 Rect) {
	x1, y1 := room1.Center()
	x2, y2 := room2.Center()

	if common.GetDiceRoll(2) == 2 {
		carveHorizontalTunnelInTiles(tiles, x1, x2, y1)
		carveVerticalTunnelInTiles(tiles, y1, y2, x2)
	} else {
		carveVerticalTunnelInTiles(tiles, y1, y2, x1)
		carveHorizontalTunnelInTiles(tiles, x1, x2, y2)
	}
}

func carveHorizontalTunnelInTiles(tiles []*Tile, x1, x2, y int) {
	for x := min(x1, x2); x <= max(x1, x2); x++ {
		logicalPos := coords.LogicalPosition{X: x, Y: y}
		index := coords.CoordManager.LogicalToIndex(logicalPos)
		if index >= 0 && index < len(tiles) {
			tiles[index].Blocked = false
			tiles[index].TileType = FLOOR
			tiles[index].image = floorImgs[common.GetRandomBetween(0, len(floorImgs)-1)]
			ValidPos.Add(x, y)
		}
	}
}

func carveVerticalTunnelInTiles(tiles []*Tile, y1, y2, x int) {
	for y := min(y1, y2); y <= max(y1, y2); y++ {
		logicalPos := coords.LogicalPosition{X: x, Y: y}
		index := coords.CoordManager.LogicalToIndex(logicalPos)
		if index >= 0 && index < len(tiles) {
			tiles[index].Blocked = false
			tiles[index].TileType = FLOOR
			tiles[index].image = floorImgs[common.GetRandomBetween(0, len(floorImgs)-1)]
			ValidPos.Add(x, y)
		}
	}
}

// Modified NewGameMap - now uses extracted function
func NewGameMap() GameMap {
	loadTileImages()
	ValidPos = ValidPositions{
		Pos: make([]coords.LogicalPosition, 0),
	}

	config := DefaultGenerationConfig()
	tiles, rooms := GenerateTilesRoomsAndCorridors(config)

	dungeonMap := GameMap{
		Tiles:         tiles,
		Rooms:         rooms,
		PlayerVisible: fov.New(),
		NumTiles:      len(tiles),
	}

	dungeonMap.PlaceStairs()
	return dungeonMap
}

// REMOVED: GenerateLevelTiles() method - replaced by standalone function
// REMOVED: createRoom(), createHorizontalTunnel(), createVerticalTunnel() - renamed and extracted
```

*Testing becomes possible (worldmap/generation_test.go):*
```go
package worldmap

import (
	"testing"
)

func TestGenerateTilesRoomsAndCorridors(t *testing.T) {
	config := GenerationConfig{
		Width:       80,
		Height:      50,
		MinRoomSize: 6,
		MaxRoomSize: 10,
		MaxRooms:    30,
	}

	tiles, rooms := GenerateTilesRoomsAndCorridors(config)

	// Verify output
	if len(tiles) != 80*50 {
		t.Errorf("Expected 4000 tiles, got %d", len(tiles))
	}

	if len(rooms) == 0 {
		t.Error("Expected at least one room")
	}

	// Verify at least some floors exist
	floorCount := 0
	for _, tile := range tiles {
		if tile.TileType == FLOOR {
			floorCount++
		}
	}

	if floorCount == 0 {
		t.Error("No floor tiles generated")
	}

	t.Logf("Generated %d rooms with %d floor tiles", len(rooms), floorCount)
}

func TestConfigurableRoomSizes(t *testing.T) {
	smallRooms := GenerationConfig{
		Width: 80, Height: 50,
		MinRoomSize: 3, MaxRoomSize: 5, MaxRooms: 50,
	}

	largeRooms := GenerationConfig{
		Width: 80, Height: 50,
		MinRoomSize: 10, MaxRoomSize: 15, MaxRooms: 10,
	}

	_, smallRoomsList := GenerateTilesRoomsAndCorridors(smallRooms)
	_, largeRoomsList := GenerateTilesRoomsAndCorridors(largeRooms)

	t.Logf("Small rooms config: %d rooms", len(smallRoomsList))
	t.Logf("Large rooms config: %d rooms", len(largeRoomsList))

	// Large rooms should generate fewer rooms (less space)
	if len(largeRoomsList) > len(smallRoomsList) {
		t.Error("Large rooms should fit fewer rooms than small rooms")
	}
}

func BenchmarkGeneration(b *testing.B) {
	config := DefaultGenerationConfig()

	for i := 0; i < b.N; i++ {
		GenerateTilesRoomsAndCorridors(config)
	}
}
```

**Key Changes**:
1. Extracted GenerateLevelTiles() → GenerateTilesRoomsAndCorridors(config)
2. Created GenerationConfig for parameters
3. Made helper functions standalone (carveRoomIntoTiles, etc.)
4. NewGameMap now calls standalone function
5. **No interface yet** - just extraction

**Value Proposition**:
- **Maintainability**: Same code, better structure (separation of concerns achieved)
- **Readability**: Generation is now a pure function with clear inputs/outputs
- **Extensibility**: Foundation for future interface extraction
- **Complexity Impact**:
  - Same algorithmic complexity
  - Easier to understand flow (no hidden GameMap mutations)
  - Testable generation for the first time

**Implementation Strategy**:
1. **Phase 1** (2 hours): Extract GenerateTilesRoomsAndCorridors with config
2. **Phase 2** (1 hour): Rename and extract helper functions
3. **Phase 3** (1 hour): Update NewGameMap to use extracted function
4. **Phase 4** (2 hours): Write comprehensive tests
5. **Future** (optional): Add interface on top if second algorithm is needed
6. **Total**: ~6 hours for extraction + testing

**Advantages**:
- **Zero risk**: Exact same algorithm, just reorganized
- **Immediate testability**: Can test generation without creating full GameMap
  ```go
  func TestMyNewParameters(t *testing.T) {
      config := GenerationConfig{MinRoomSize: 8, MaxRoomSize: 12, MaxRooms: 15}
      tiles, rooms := GenerateTilesRoomsAndCorridors(config)
      // Test specific parameters
  }
  ```
- **No breaking changes**: NewGameMap() signature unchanged
- **Foundation for future**: Easy to add interface later
- **Experiment friendly**: Tweak config, run tests, see results

**Drawbacks & Risks**:
- **Not extensible yet**: Still only one algorithm, just extracted
  - *Mitigation*: This is intentional - prove separation works first
- **Code duplication**: If adding second algorithm, will duplicate some logic
  - *Mitigation*: Extract common helpers when second algorithm is actually needed
- **Delayed full solution**: Need Phase 2 refactoring for true multi-algorithm support
  - *Mitigation*: Phase 2 is much lower risk after proving extraction works

**Effort Estimate**:
- **Time**: 6-8 hours (1 day with thorough testing)
- **Complexity**: Low (extraction refactoring, no new concepts)
- **Risk**: Very Low (no behavior changes, incremental)
- **Files Impacted**:
  - Modified: dungeongen.go (~40 LOC added for config/extraction, ~50 LOC refactored)
  - New: generation_test.go (~100 LOC tests)

**Critical Assessment** (Pragmatic Evaluation):
This is the most pragmatic approach. It solves the immediate problem (can't test generation, can't experiment with parameters) with minimal risk. Unlike Approach 1 and 2, there's no upfront interface design that might be wrong. You extract what exists, prove it works, then decide if you need interfaces. If you're unsure whether you'll actually add multiple algorithms, this is the right starting point. You can always evolve to Approach 1 later (extraction makes that trivial). The downside is you don't get multi-algorithm support immediately, but you avoid over-engineering if that feature is never needed. Best for teams that value incremental delivery and want to defer architectural decisions until requirements are clear.

---

## COMPARATIVE ANALYSIS OF FINAL APPROACHES

### Effort vs Impact Matrix
| Approach | Effort | Impact | Risk | Recommended Priority |
|----------|--------|--------|------|---------------------|
| Approach 1: Strategy Pattern | Medium (8-12h) | High | Low | **1** |
| Approach 2: Functional Composition | Medium-High (10-14h) | Very High | Medium | 2 |
| Approach 3: Incremental Extraction | Low (6-8h) | Medium | Very Low | 3 |

### Decision Guidance

**Choose Approach 1 (Strategy Pattern) if:**
- You know you need multiple complete algorithms (BSP, cellular, drunkard's walk)
- Team is comfortable with interface-based design
- You want production-ready architecture immediately
- Testing individual algorithms in isolation is important
- **Recommended for: Most teams - best balance of all factors**

**Choose Approach 2 (Functional Composition) if:**
- You want to mix and match generation techniques (hybrid algorithms)
- Team prefers functional programming patterns
- Maximum flexibility is worth the complexity
- You anticipate complex generation needs (rooms + caves + prefabs in one map)
- **Recommended for: Advanced use cases needing hybrid generation**

**Choose Approach 3 (Incremental Extraction) if:**
- You're uncertain whether multiple algorithms are actually needed
- Want to minimize upfront investment
- Prefer incremental refactoring over big-bang changes
- Immediate need is testing/parameter tuning, not new algorithms
- **Recommended for: Risk-averse teams or when requirements are unclear**

### Combination Opportunities

**Phase 1 + Phase 2 Hybrid**:
Start with Approach 3 (Incremental Extraction), then evolve to Approach 1 (Strategy Pattern) when second algorithm is actually needed:

1. **Week 1**: Extract current algorithm with config (6-8 hours)
2. **Week 1-2**: Experiment with parameters, write tests (ongoing)
3. **Week 3** (if needed): Add MapGenerator interface around proven extraction
4. **Week 4**: Implement second algorithm (BSP or cellular)

This minimizes risk while maintaining path to full strategy pattern.

**Approach 1 + Approach 2 Hybrid**:
Use Strategy Pattern for high-level algorithms, but implement each algorithm using functional primitives:

```go
// Interface for algorithm selection
type MapGenerator interface {
	Generate(width, height int) GenerationResult
	Name() string
}

// Implementation uses functional composition
type RoomsAndCorridorsGenerator struct{}

func (g *RoomsAndCorridorsGenerator) Generate(w, h int) GenerationResult {
	state := NewMapState(w, h)

	// Use composable primitives internally
	generator := Compose(
		PlaceRoomsRandomly(6, 10, 30),
		ConnectRoomsWithCorridors(),
	)
	generator(state)

	return GenerationResult{Tiles: state.Tiles, Rooms: state.Rooms}
}
```

This combines interface-based selection (Approach 1) with primitive reuse (Approach 2).

---

## APPENDIX: INITIAL APPROACHES FROM ALL PERSPECTIVES

### A. Refactoring-Pro Approaches (Architectural Focus)

#### Refactoring-Pro Approach 1: Strategy Pattern with Generator Interface
**Focus**: Clean separation through interface-based design

**Problem**: GameMap has generation hardcoded in GenerateLevelTiles() method, violating Open/Closed Principle

**Solution**:
- Create MapGenerator interface with Generate() method
- Extract current algorithm to RoomsAndCorridorsGenerator
- Use registry pattern for algorithm selection
- GameMap receives generated tiles, doesn't create them

**Code Sample**: (See Final Approach 1 above for full implementation)

**Metrics**:
- Reduces dungeongen.go from 550 LOC to ~370 LOC (33% reduction)
- Cyclomatic complexity: GameMap drops from ~15 to ~8
- Adds 2-3 new files (generator.go, gen_*.go)
- Each generator ~150-200 LOC

**Assessment**:
- **Pros**: Clean contracts, easy to add algorithms, testable, production-ready
- **Cons**: More files, slight indirection, migration effort
- **Effort**: 8-12 hours (interface + extraction + migration + testing)

---

#### Refactoring-Pro Approach 2: Functional Generator Pipeline
**Focus**: Go-idiomatic pure functions with composition

**Problem**: Beyond algorithm selection, need to reuse generation logic and create hybrid approaches

**Solution**:
- Generators are pure functions: `func(config) GenerationResult`
- Composable primitives (PlaceRooms, CarveCorridors, AddCaverns)
- Combine primitives to create algorithms
- No interfaces, just function composition

**Code Sample**: (See Final Approach 2 above for full implementation)

**Metrics**:
- genprimitives.go: ~300 LOC (reusable building blocks)
- gencomposed.go: ~150 LOC (example compositions)
- Each primitive: 15-30 LOC (very focused)
- dungeongen.go: modified to use composition

**Assessment**:
- **Pros**: Maximum flexibility, Go-idiomatic, easy testing, composable
- **Cons**: Mutable state, composition order matters, learning curve
- **Effort**: 10-14 hours (primitives + compositions + examples + testing)

---

#### Refactoring-Pro Approach 3: Builder Pattern with Fluent API
**Focus**: Incremental generation with readable chaining

**Problem**: Need flexible generation with clear step-by-step control

**Solution**:
- MapBuilder with fluent API (WithRooms(), WithCorridors(), WithFeatures())
- Each step is pluggable and optional
- Build() executes all steps and returns GameMap
- Clear order of operations

**Code Sample**:
```go
type MapBuilder struct {
	width, height int
	steps         []BuildStep
}

type BuildStep func(*MapState) error

func NewMapBuilder(width, height int) *MapBuilder {
	return &MapBuilder{width: width, height: height, steps: make([]BuildStep, 0)}
}

func (b *MapBuilder) WithRooms(minSize, maxSize, maxRooms int) *MapBuilder {
	step := func(state *MapState) error {
		// Place rooms logic
		return nil
	}
	b.steps = append(b.steps, step)
	return b
}

func (b *MapBuilder) WithCorridors() *MapBuilder {
	step := func(state *MapState) error {
		// Connect rooms logic
		return nil
	}
	b.steps = append(b.steps, step)
	return b
}

func (b *MapBuilder) WithCaverns(iterations int) *MapBuilder {
	step := func(state *MapState) error {
		// Cellular automata logic
		return nil
	}
	b.steps = append(b.steps, step)
	return b
}

func (b *MapBuilder) Build() (*GameMap, error) {
	state := NewMapState(b.width, b.height)

	for _, step := range b.steps {
		if err := step(state); err != nil {
			return nil, err
		}
	}

	return stateToGameMap(state), nil
}

// Usage:
gm, err := NewMapBuilder(80, 50).
	WithRooms(6, 10, 30).
	WithCorridors().
	WithCaverns(3).
	Build()
```

**Metrics**:
- MapBuilder: ~150 LOC
- Each WithX method: ~20-40 LOC
- Build logic: ~30 LOC
- Example usage: 3-4 lines

**Assessment**:
- **Pros**: Readable, flexible, incremental generation, self-documenting
- **Cons**: More methods, potential for misuse (wrong order), boilerplate
- **Effort**: 8-12 hours (builder + steps + error handling + testing)

**Why not final**: More complex than Strategy Pattern for same extensibility, fluent API adds ceremony without clear benefit over simple function calls. Over-engineered for current needs.

---

### B. Tactical-Simplifier Approaches (Game-Specific Focus)

#### Tactical-Simplifier Approach 1: Simple Algorithm Registry
**Focus**: Minimal code to add new algorithms

**Problem**: Adding BSP, cellular automata, or drunkard's walk requires modifying GameMap

**Gameplay Preservation**: Different algorithms provide level variety for replayability

**Go-Specific Optimizations**:
- Use map[string]GeneratorFunc for simple registry
- No complex interfaces, just function signatures
- One-line registration: `RegisterAlgorithm("bsp", GenerateBSP)`

**Code Sample**:
```go
// Simple function signature for generators
type GeneratorFunc func(width, height int) ([]*Tile, []Rect)

// Registry
var algorithms = make(map[string]GeneratorFunc)

func RegisterAlgorithm(name string, fn GeneratorFunc) {
	algorithms[name] = fn
}

func Generate(algorithmName string, width, height int) ([]*Tile, []Rect) {
	fn, ok := algorithms[algorithmName]
	if !ok {
		fn = algorithms["rooms_corridors"] // default
	}
	return fn(width, height)
}

// Adding new algorithm is trivial:
func GenerateBSP(width, height int) ([]*Tile, []Rect) {
	// BSP logic here
	return tiles, rooms
}

func init() {
	RegisterAlgorithm("rooms_corridors", GenerateRoomsAndCorridors)
	RegisterAlgorithm("bsp", GenerateBSP)
	RegisterAlgorithm("cellular", GenerateCellular)
}
```

**Game System Impact**:
- Level variety improves with multiple algorithms
- Easy A/B testing for level design
- Performance unchanged (generation once per level)

**Assessment**:
- **Pros**: Minimal code, easy to understand, fast to implement
- **Cons**: Less type safety, no algorithm metadata, runtime errors
- **Effort**: 4-6 hours (registry + extract current + one new algorithm)

---

#### Tactical-Simplifier Approach 2: Composable Generation Primitives
**Focus**: Reusable building blocks for roguelike generation

**Problem**: Roguelikes often need hybrid algorithms (BSP layout + cellular caves + prefab vaults)

**Gameplay Preservation**: Primitives enable unique level combinations for each playthrough

**Go-Specific Optimizations**:
- Small, focused functions (15-30 LOC each)
- No state, just transformations
- Easy to test each primitive independently

**Code Sample**: (See Final Approach 2 for full implementation)

**Game System Impact**:
- Can create themed levels: "dungeon" (rooms+corridors), "caves" (cellular), "hybrid" (both)
- Special room primitives (boss rooms, treasure vaults, secret areas)
- Easy to add game-specific features (squad spawn points, tactical cover placement)

**Assessment**:
- **Pros**: Maximum reusability, easy testing, supports hybrid algorithms
- **Cons**: Requires understanding composition, more files
- **Effort**: 10-12 hours (primitives + examples + testing)

---

#### Tactical-Simplifier Approach 3: Template-Based Generation with Variants
**Focus**: Designer-friendly parameter-driven generation

**Problem**: Game designers want to tweak generation without coding

**Gameplay Preservation**: Data-driven approach enables rapid iteration on level feel

**Go-Specific Optimizations**:
- Use structs for algorithm templates
- JSON/TOML config files for parameters
- Hot-reload configs without recompiling

**Code Sample**:
```go
// Algorithm template with variant parameters
type GenerationTemplate struct {
	Algorithm   string
	Parameters  map[string]interface{}
	Description string
}

// Load from JSON
func LoadTemplate(filename string) GenerationTemplate {
	// Parse JSON config
	return template
}

// Example config (dungeon_templates.json):
{
	"small_tight_dungeon": {
		"algorithm": "rooms_corridors",
		"parameters": {
			"min_room_size": 4,
			"max_room_size": 6,
			"max_rooms": 50,
			"corridor_width": 1
		}
	},
	"large_open_dungeon": {
		"algorithm": "rooms_corridors",
		"parameters": {
			"min_room_size": 10,
			"max_room_size": 20,
			"max_rooms": 15,
			"corridor_width": 3
		}
	},
	"cave_system": {
		"algorithm": "cellular_automata",
		"parameters": {
			"iterations": 5,
			"wall_threshold": 0.45,
			"smoothing": true
		}
	}
}

// Usage:
template := LoadTemplate("dungeon_templates.json").Get("small_tight_dungeon")
tiles, rooms := GenerateFromTemplate(template)
```

**Game System Impact**:
- Designers can A/B test level feel without programmer involvement
- Easy to create progression (level 1 = small rooms, level 10 = large caves)
- Parameters can be balanced separately from code

**Assessment**:
- **Pros**: Designer-friendly, data-driven, easy iteration
- **Cons**: Config explosion, limited to parameterizable algorithms, validation needed
- **Effort**: 8-10 hours (template system + configs + validation + examples)

**Why not final**: Good for parameter tuning but doesn't solve multi-algorithm extensibility. Better as addition to Approach 1 or 2 than standalone solution. Risk of config complexity outweighing benefits.

---

## SYNTHESIS RATIONALE

### Why These 3 Final Approaches?

**Approach 1 Selection (Strategy Pattern with Simple Registry)**:
Combined refactoring-pro's Strategy Pattern (clean interfaces, testability) with tactical-simplifier's Simple Registry (minimal ceremony, easy registration). This creates production-ready architecture without over-engineering. The interface is minimal (one method), the registry is straightforward (map[string]Generator), and the value is immediate (adding algorithms becomes single-file effort).

**Key combination**:
- Pro-1's interface design + Simp-1's registration simplicity
- Balances architectural cleanliness with practical Go idioms
- Most requested by consensus: "easy to add algorithms, testable, clear separation"

**Approach 2 Selection (Functional Composition with Primitives)**:
Combined refactoring-pro's Functional Pipeline (pure functions, composition) with tactical-simplifier's Composable Primitives (roguelike-specific building blocks). This enables hybrid algorithms (rooms + caves + prefabs) through function composition. Very Go-idiomatic, maximum flexibility.

**Key combination**:
- Pro-2's composition pattern + Simp-2's roguelike primitives
- Addresses advanced use case: hybrid generation
- Most powerful but requires functional programming comfort

**Approach 3 Selection (Incremental Extraction)**:
New synthesis from refactoring-critic's pragmatic evaluation. Neither original approach addressed the "prove it first" concern. This approach extracts current algorithm to standalone function with config, enabling testing/experimentation without committing to interfaces. Can evolve to Approach 1 later with minimal effort.

**Key insight**:
- Critic perspective: "Validate separation works before designing interfaces"
- Addresses uncertainty: "Do we actually need multiple algorithms?"
- Lowest risk path to testability and experimentation

### Rejected Elements

**Builder Pattern (Pro-3)**:
Fluent API adds ceremony without clear benefit. Strategy Pattern (Approach 1) provides same extensibility with less boilerplate. Builder is better for complex object construction; generation is more naturally expressed as functions or strategies.

**Template-Based Variants (Simp-3)**:
Data-driven configs are valuable but don't solve the core extensibility problem. Better as an addition to Approach 1 (interface + registry + config files) than standalone. Risk of config complexity. Can be added later if designers need parameter tuning.

**Complex Metadata Systems**:
Some perspectives suggested algorithm discovery, capability flags, parameter introspection. Rejected as over-engineering. Simple Name() and Description() methods (Approach 1) provide enough metadata for current needs. Can add more later if needed (YAGNI principle).

### Critical Evaluation Summary

**From Refactoring-Critic Perspective**:

**Approach 1 Assessment**:
Best balance of theory and practice. Strategy Pattern is appropriate here (not over-engineering) because:
1. Problem is real: Can't add algorithms without modifying GameMap
2. Solution is minimal: One-method interface, simple registry
3. Value is immediate: Second algorithm requires zero changes to existing code
4. Risk is low: Extraction maintains exact behavior

**Approach 2 Assessment**:
Powerful but complex. Appropriate if:
1. You actually need hybrid generation (rooms + caves in one map)
2. Team is comfortable with functional patterns
3. Willing to invest in primitives library

Not recommended if you just want to swap complete algorithms (use Approach 1 instead).

**Approach 3 Assessment**:
Most pragmatic if uncertain about requirements. Proves separation works before committing to interfaces. Can evolve to Approach 1 with 2-4 hours additional work. Risk of delaying full solution if multiple algorithms are definitely needed.

**Overall Recommendation**: **Approach 1 for most teams**. It's the sweet spot between Approach 3's caution and Approach 2's power. Delivers extensibility without over-engineering.

---

## PRINCIPLES APPLIED

### Software Engineering Principles

**DRY (Don't Repeat Yourself)**:
- All approaches extract generation logic from GameMap (was duplicated conceptually with responsibilities)
- Approach 2 maximizes DRY through primitives (room placement reused across algorithms)
- Approach 1 maintains DRY while keeping algorithms self-contained

**SOLID Principles**:
- **Single Responsibility**: GameMap currently does generation + storage + rendering + FOV + entity management. All approaches separate generation responsibility.
- **Open/Closed**: Approach 1 and 2 make system open to extension (new algorithms) but closed to modification (no GameMap changes)
- **Liskov Substitution**: Approach 1's MapGenerator interface ensures any algorithm can substitute for another
- **Interface Segregation**: Approach 1 uses minimal interface (1 method), not bloated with unnecessary methods
- **Dependency Inversion**: GameMap depends on abstraction (Generator interface) not concrete algorithms

**KISS (Keep It Simple, Stupid)**:
- Approach 3 is KISS champion: Extract, add config, test - no interfaces until proven needed
- Approach 1 balances simplicity with extensibility: Simple interface, simple registry
- Approach 2 sacrifices some simplicity for power (functional composition requires understanding)

**YAGNI (You Aren't Gonna Need It)**:
- Approach 3 embodies YAGNI: Don't add interfaces until second algorithm is actually needed
- Rejected builder pattern and template system as YAGNI violations (complex features without proven need)
- Approach 1 validated as NOT YAGNI: Extensibility is stated requirement, interface enables it

**SLAP (Single Level of Abstraction Principle)**:
- All approaches improve SLAP: Current GenerateLevelTiles mixes high-level (room placement) with low-level (pixel calculations)
- Approach 1: High-level (Generate) delegates to algorithm-specific implementation
- Approach 2: Each primitive operates at consistent abstraction level

**Separation of Concerns**:
- Core principle driving all approaches
- GameMap concerns: Store tiles, manage FOV, handle entity placement
- Generator concerns: Create tiles, place rooms, carve corridors
- Current code violates this; all approaches fix it

### Go-Specific Best Practices

**Idiomatic Go Patterns**:
- **Approach 1**: Interface-based polymorphism (Go standard: io.Reader, http.Handler)
- **Approach 2**: Function composition and pure functions (very Go-idiomatic)
- **Approach 3**: Simple functions with config structs (Go's preference for simplicity)

**Composition Over Inheritance**:
- No inheritance used (Go doesn't have it)
- Approach 1: Composition through interface implementation
- Approach 2: Composition through function chaining
- All approaches favor Go's embedding and composition model

**Interface Design**:
- Approach 1 follows Go interface best practices:
  - Small interfaces (1-2 methods)
  - Named by behavior (MapGenerator.Generate)
  - Defined by consumer (worldmap package), implemented by algorithms

**Error Handling**:
- All approaches use Go error returns (not exceptions)
- Approach 3's extraction enables better error handling (function can return error)
- Current code has no error handling for generation failures

**Package Organization**:
- Approach 1: Clear file naming (generator.go, gen_rooms_corridors.go, gen_bsp.go)
- Approach 2: Primitives package organization (genprimitives.go, gencomposed.go)
- Follows Go convention: flat package structure, descriptive file names

### Game Development Considerations

**Performance Implications**:
- Generation happens once per level (not in game loop)
- All approaches have negligible performance difference
- Function calls (Approach 2) vs virtual dispatch (Approach 1): Unmeasurable in this context
- Critical path is tile array allocation (~4000 tiles), not algorithm structure

**Real-Time System Constraints**:
- Generation is NOT real-time (happens during level transition)
- No frame budget concerns
- Can take 10-50ms for generation without impacting gameplay
- All approaches well within acceptable generation time

**Game Loop Integration**:
- Generation called from level transition, not update loop
- All approaches integrate identically: `gm := NewGameMap("rooms_corridors")`
- No impact on rendering, input, or combat systems

**Tactical Gameplay Preservation**:
- Different algorithms affect tactical depth:
  - Rooms+corridors: Chokepoints, ambush locations
  - Cellular caves: Open combat, less cover
  - BSP: Large structured spaces, predictable layout
- All approaches enable algorithm variety, enhancing tactical gameplay
- Must ensure all algorithms create navigable, balanced spaces

**Level Variety and Replayability**:
- Current single algorithm limits replayability
- Approach 1 and 2 enable algorithm rotation per level
- Approach 3 enables parameter variation (min step toward variety)
- Multiple algorithms directly improve roguelike core loop

**ECS Compliance**:
- TileContents uses entity pointers (violates ECS best practices)
- **BONUS FIX**: While refactoring generation, fix TileContents:
  ```go
  // Before (dungeontile.go):
  type TileContents struct {
  	entities []*ecs.Entity  // ANTI-PATTERN
  }

  // After:
  type TileContents struct {
  	entityIDs []ecs.EntityID  // ECS COMPLIANT
  }
  ```
- This brings worldmap package in line with squad system and inventory system patterns

---

## NEXT STEPS

### Recommended Action Plan

**Immediate (Choose Your Path)**:

**If choosing Approach 1 (Recommended):**
1. **Day 1 Morning** (3 hours): Create generator.go with MapGenerator interface and registry
2. **Day 1 Afternoon** (3 hours): Extract current algorithm to gen_rooms_corridors.go
3. **Day 2 Morning** (2 hours): Modify NewGameMap, test thoroughly, ensure backward compatibility
4. **Day 2 Afternoon** (3 hours): Implement second algorithm (BSP or cellular) to validate design
5. **Day 3** (2 hours): Write comprehensive tests, document interface, update CLAUDE.md

**If choosing Approach 3 (Cautious):**
1. **Session 1** (2 hours): Extract GenerateTilesRoomsAndCorridors with GenerationConfig
2. **Session 2** (2 hours): Write tests for extracted function, experiment with parameters
3. **Session 3** (2 hours): If second algorithm needed, evolve to Approach 1 (interface + registry)

**Short-Term (Week 2-3)**:
- Add 2-3 more algorithms: BSP, cellular automata, drunkard's walk
- Create algorithm selection system (random, level-based, player choice)
- Update level progression to use variety of algorithms
- Ensure all algorithms support squad deployment (navigable spaces, tactical cover)

**Medium-Term (Month 2)**:
- Fix ECS violation in TileContents (entity pointers → EntityIDs)
- Add prefab system (special rooms, vaults, boss arenas)
- Consider Approach 2 primitives if hybrid generation is needed
- Add generation metrics (connectivity, openness, difficulty)

**Long-Term (Ongoing)**:
- Build library of tested algorithms
- Add data-driven configs for algorithm parameters (Template approach from Simp-3)
- Create level theme system (dungeon, cave, ruins, each with appropriate algorithms)
- Profile generation performance if maps get very large (100x100+)

### Validation Strategy

**Testing Approach**:
1. **Unit Tests**: Each algorithm generates valid maps (tiles allocated, rooms exist, connectivity)
   ```go
   func TestGeneratorCreatesValidMap(t *testing.T) {
       gen := NewRoomsAndCorridorsGenerator(DefaultConfig())
       result := gen.Generate(80, 50)

       assert.Equal(t, 80*50, len(result.Tiles))
       assert.True(t, len(result.Rooms) > 0)
       assertMapIsConnected(t, result.Tiles)
       assertRoomsNonOverlapping(t, result.Rooms)
   }
   ```

2. **Integration Tests**: Generated maps work with pathfinding, FOV, entity placement
   ```go
   func TestGeneratedMapWithPathfinding(t *testing.T) {
       gm := NewGameMap("rooms_corridors")
       start := gm.Rooms[0].Center()
       end := gm.Rooms[len(gm.Rooms)-1].Center()

       path := BuildPath(&gm, &start, &end)
       assert.NotNil(t, path, "Should find path between rooms")
   }
   ```

3. **Visual Tests**: Render generated maps to image files, manually inspect variety
4. **Property-Based Tests**: Fuzzing with random seeds, ensure no crashes/invalid states
5. **Performance Tests**: Benchmark generation time, ensure < 50ms per map

**Rollback Plan**:
- Use feature flag: `USE_NEW_GENERATOR=false` falls back to old GenerateLevelTiles
- Keep old code commented out for one release cycle
- Git tag before refactoring: `git tag pre-generation-refactor`
- Can revert single commit if issues found

**Success Metrics**:
1. **Extensibility**: Adding new algorithm takes < 4 hours and requires 0 changes to existing files
2. **Testability**: Each algorithm has >80% test coverage
3. **Performance**: Generation time remains < 50ms per map
4. **Stability**: Zero crashes in 1000 generated maps with random seeds
5. **Variety**: Player can distinguish between different algorithms visually
6. **Backward Compatibility**: All existing GameMap usage continues to work

### Additional Resources

**Go Patterns Documentation**:
- [Effective Go: Interfaces](https://go.dev/doc/effective_go#interfaces)
- [Go by Example: Interfaces](https://gobyexample.com/interfaces)
- [Strategy Pattern in Go](https://refactoring.guru/design-patterns/strategy/go/example)

**Game Architecture References**:
- [Procedural Content Generation in Games (textbook)](http://pcgbook.com/)
- [RogueBasin: Dungeon Generation](http://www.roguebasin.com/index.php?title=Category:Articles_with_code)
- [BSP Dungeon Generation](http://www.roguebasin.com/index.php?title=Basic_BSP_Dungeon_generation)
- [Cellular Automata for Cave Generation](http://www.roguebasin.com/index.php?title=Cellular_Automata_Method_for_Generating_Random_Cave-Like_Levels)

**Refactoring Resources**:
- [Refactoring: Improving the Design of Existing Code (Martin Fowler)](https://refactoring.com/)
- [Working Effectively with Legacy Code (Michael Feathers)](https://www.oreilly.com/library/view/working-effectively-with/0131177052/)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

**ECS Best Practices** (from CLAUDE.md):
- Reference: `squads/*.go` - Perfect ECS with pure data components
- Reference: `gear/Inventory.go` - System functions, not component methods
- Reference: `gear/items.go` - EntityID-based relationships
- Apply these patterns when fixing TileContents entity pointer issue

---

END OF ANALYSIS
