# Worldmap Strategy Pattern Refactoring Plan
Generated: 2025-11-08
Status: Implementation Ready

## Executive Summary

**Goal**: Refactor the worldmap package to easily support multiple map generation algorithms while maintaining code quality and avoiding over-complexity.

**Current Problem**: Map generation is hardcoded in `GameMap.GenerateLevelTiles()` (dungeongen.go:360-405). Adding new algorithms requires modifying GameMap, which violates Open/Closed Principle and makes testing difficult.

**Solution**: Strategy Pattern with simple registry - extract generation into a `MapGenerator` interface with concrete implementations for each algorithm.

**Effort**: **14-18 hours** (realistic estimate with testing and global state cleanup)

**Benefits**:
- Add new algorithms with zero changes to existing code
- Test each algorithm in isolation
- Reduce GameMap complexity by ~180 LOC
- Clear separation of concerns

---

## Table of Contents
1. [Current State Analysis](#current-state-analysis)
2. [Target Architecture](#target-architecture)
3. [Critical Issues to Address](#critical-issues-to-address)
4. [Implementation Phases](#implementation-phases)
5. [Code Examples](#code-examples)
6. [Testing Strategy](#testing-strategy)
7. [Migration Path](#migration-path)
8. [Second Algorithm Example](#second-algorithm-example)

---

## Current State Analysis

### Files in Scope
- `worldmap/dungeongen.go` (550 LOC) - Contains GameMap and generation logic
- `worldmap/dungeontile.go` (68 LOC) - Tile definitions
- `worldmap/GameMapUtil.go` - Image loading utilities

### Problem Areas

#### 1. Hardcoded Generation (dungeongen.go:360-405)
```go
func (gameMap *GameMap) GenerateLevelTiles() {
    MIN_SIZE := 6
    MAX_SIZE := 10
    MAX_ROOMS := 30

    // 46 lines of rooms-and-corridors logic embedded in GameMap
    // Cannot add BSP, cellular automata, etc. without modifying this method
}
```

**Issues**:
- Cannot swap algorithms without changing GameMap
- Parameters (MIN_SIZE, MAX_SIZE, MAX_ROOMS) are hardcoded constants
- Testing requires creating full GameMap instance
- No way to experiment with different generation styles

#### 2. Global State (GameMapUtil.go:10-12)
```go
var floorImgs = make([]*ebiten.Image, 0)
var wallImgs = make([]*ebiten.Image, 0)
var stairs_down *ebiten.Image
```

**Issues**:
- Global image arrays accessed by generation code
- `ValidPos` global variable mutated during generation (dungeongen.go:20)
- Generators must coordinate on shared state
- Hard to test with different image sets

#### 3. ECS Violation (dungeontile.go:23-25)
```go
type TileContents struct {
    entities []*ecs.Entity  // ANTI-PATTERN: Should be []ecs.EntityID
}
```

**Issues**:
- Uses entity pointers instead of EntityIDs
- Violates ECS best practices (see CLAUDE.md squad/inventory patterns)
- Should be fixed during refactoring

#### 4. Tight Coupling
- `createRoom()`, `createHorizontalTunnel()`, `createVerticalTunnel()` are GameMap methods
- Generation code directly mutates GameMap.Tiles and GameMap.Rooms
- Hard to extract without breaking dependencies

---

## Target Architecture

### Core Design Principles

1. **Separation of Concerns**: GameMap stores tiles, generators create tiles
2. **Open/Closed**: New algorithms don't modify existing code
3. **Simple Interface**: One method (`Generate()`), minimal complexity
4. **Testability**: Each generator independently testable
5. **Go Idiomatic**: Use interfaces, avoid over-engineering

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    GameMap (Data)                        │
│  - Tiles []*Tile                                        │
│  - Rooms []Rect                                         │
│  - PlayerVisible *fov.View                              │
│  - FOV, Pathfinding, Rendering methods                 │
└─────────────────────────────────────────────────────────┘
                            │
                            │ receives GenerationResult
                            ▼
┌─────────────────────────────────────────────────────────┐
│            MapGenerator Interface                        │
│  + Generate(width, height int) GenerationResult         │
│  + Name() string                                        │
│  + Description() string                                 │
└─────────────────────────────────────────────────────────┘
                            ▲
                            │ implements
        ┌───────────────────┼────────────────────┐
        │                   │                    │
┌───────────────┐  ┌────────────────┐  ┌─────────────────┐
│   Rooms &     │  │   BSP Tree     │  │    Cellular     │
│  Corridors    │  │   Dungeon      │  │    Automata     │
│  Generator    │  │   Generator    │  │    Caves        │
└───────────────┘  └────────────────┘  └─────────────────┘

┌─────────────────────────────────────────────────────────┐
│               Generator Registry                         │
│  map[string]MapGenerator                                │
│  - RegisterGenerator(gen)                               │
│  - GetGenerator(name) MapGenerator                      │
└─────────────────────────────────────────────────────────┘
```

### Package Structure

```
worldmap/
├── dungeongen.go          # GameMap, NewGameMap(), map operations
├── dungeontile.go         # Tile, TileType definitions
├── GameMapUtil.go         # Utility functions, image loading
├── generator.go           # NEW: Interface, registry, GenerationResult
├── gen_rooms_corridors.go # NEW: Current algorithm extracted
├── gen_bsp.go             # FUTURE: BSP tree algorithm
└── gen_cellular.go        # FUTURE: Cellular automata caves
```

**Naming Convention**: `gen_*.go` for all generator implementations

---

## Critical Issues to Address

### Issue 1: Global State Cleanup

**Problem**: `ValidPos`, `wallImgs`, `floorImgs` are global variables accessed during generation.

**Solution**: Two options:

**Option A: Pass as Context (Simpler)**
```go
type GenerationContext struct {
    WallImages  []*ebiten.Image
    FloorImages []*ebiten.Image
    ValidPositions *ValidPositions  // Mutable, cleared before generation
}

// Interface changes
type MapGenerator interface {
    Generate(width, height int, ctx *GenerationContext) GenerationResult
    Name() string
    Description() string
}
```

**Option B: Include in Result (Cleaner)**
```go
type GenerationResult struct {
    Tiles          []*Tile
    Rooms          []Rect
    ValidPositions []coords.LogicalPosition  // Returned by generator
}

// Generators load images themselves or receive them as parameters
```

**Recommendation**: Option B - cleaner separation, generators are more self-contained.

**Implementation Time**: +3 hours for global state cleanup

### Issue 2: Image Loading

**Current**: `loadTileImages()` called in `NewGameMap()`, sets global variables.

**Solution**:
```go
// In GameMapUtil.go
type TileImageSet struct {
    WallImages  []*ebiten.Image
    FloorImages []*ebiten.Image
    StairsDown  *ebiten.Image
}

func LoadTileImages() TileImageSet {
    // Load images, return struct (not global)
    return TileImageSet{...}
}

// Generators receive images as needed
func (g *RoomsAndCorridorsGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
    // Use images.WallImages, images.FloorImages
}
```

**Implementation Time**: +2 hours for image loading refactor

### Issue 3: ECS Compliance for TileContents

**Current Violation**:
```go
type TileContents struct {
    entities []*ecs.Entity  // Pointers to entities
}
```

**Fixed Version**:
```go
type TileContents struct {
    entityIDs []ecs.EntityID  // Entity IDs only
}
```

**Required Changes**:
- `dungeongen.go:178-188` - `AddEntityToTile()` stores EntityID
- `dungeongen.go:195-214` - `RemoveItemFromTile()` returns EntityID
- Any code that accesses tile.tileContents.entities

**Implementation Time**: +2 hours for ECS compliance fix

**Total Extra Time for Critical Issues**: 7 hours
**Combined with Base Refactoring (8-12h)**: **14-18 hours total**

---

## Implementation Phases

### Phase 0: Pre-Refactoring Cleanup (3 hours)

**Goal**: Fix global state before extraction

**Tasks**:
1. Create `TileImageSet` struct in GameMapUtil.go
2. Refactor `loadTileImages()` to return struct, not set globals
3. Fix ECS violation in TileContents (entities → entityIDs)
4. Update all references to tile.tileContents
5. Run tests to ensure no regressions

**Validation**: All tests pass, no behavior changes

**Git Checkpoint**: `git commit -m "Pre-refactoring: Fix global state and ECS violations"`

### Phase 1: Create Generator Interface and Registry (3 hours)

**Goal**: Establish core architecture

**Tasks**:
1. Create `worldmap/generator.go`
2. Define `MapGenerator` interface
3. Define `GenerationResult` struct
4. Create generator registry (RegisterGenerator, GetGenerator, ListGenerators)
5. Define `GeneratorConfig` for common parameters
6. Write basic tests for registry

**Deliverables**:
- `generator.go` (~100 LOC)
- Registry tests

**Git Checkpoint**: `git commit -m "Add MapGenerator interface and registry"`

### Phase 2: Extract Current Algorithm (4 hours)

**Goal**: Extract rooms-and-corridors to standalone generator

**Tasks**:
1. Create `worldmap/gen_rooms_corridors.go`
2. Define `RoomsAndCorridorsGenerator` struct
3. Extract logic from `GenerateLevelTiles()` into `Generate()` method
4. Extract `createRoom()`, `createHorizontalTunnel()`, `createVerticalTunnel()` as private methods
5. Register generator in `init()`
6. Write unit tests for generator

**Deliverables**:
- `gen_rooms_corridors.go` (~220 LOC)
- Unit tests for RoomsAndCorridorsGenerator

**Git Checkpoint**: `git commit -m "Extract rooms-and-corridors generator"`

### Phase 3: Modify GameMap to Use Generator (2 hours)

**Goal**: Update NewGameMap() to use generator pattern

**Tasks**:
1. Modify `NewGameMap()` to accept generator name parameter
2. Add `NewGameMapDefault()` wrapper for backward compatibility
3. Remove `GenerateLevelTiles()` method (no longer needed)
4. Remove `createRoom()`, `createHorizontalTunnel()`, `createVerticalTunnel()` methods
5. Update all callsites in game_main/main.go
6. Test thoroughly with existing gameplay

**Deliverables**:
- Modified `dungeongen.go` (-180 LOC generation code, +30 LOC integration code)
- Backward-compatible NewGameMap()

**Git Checkpoint**: `git commit -m "Integrate generator pattern into NewGameMap"`

### Phase 4: Add Second Algorithm (3 hours)

**Goal**: Validate design with real second algorithm

**Tasks**:
1. Create `gen_bsp.go` OR `gen_cellular.go`
2. Implement BSP tree or cellular automata generation
3. Register new generator
4. Write tests for new generator
5. Test in-game level generation variety

**Deliverables**:
- Second generator implementation (~180 LOC)
- Tests validating both algorithms work

**Git Checkpoint**: `git commit -m "Add [BSP/Cellular] generator, validate multi-algorithm support"`

### Phase 5: Documentation and Cleanup (1 hour)

**Goal**: Finalize refactoring

**Tasks**:
1. Update CLAUDE.md with worldmap refactoring completion
2. Document how to add new generators (README in worldmap/)
3. Add algorithm selection guide (when to use each algorithm)
4. Clean up any TODOs or temporary code

**Deliverables**:
- Updated documentation
- Clean, production-ready code

**Total Time**: **14-16 hours** (realistic estimate)

---

## Code Examples

### 1. Generator Interface (generator.go)

```go
package worldmap

import (
	"game_main/coords"
	"github.com/hajimehoshi/ebiten/v2"
)

// GenerationResult contains the output of a map generation algorithm
type GenerationResult struct {
	Tiles          []*Tile
	Rooms          []Rect
	ValidPositions []coords.LogicalPosition
}

// MapGenerator defines the interface for all map generation algorithms
type MapGenerator interface {
	// Generate creates a new map layout
	Generate(width, height int, images TileImageSet) GenerationResult

	// Name returns the algorithm name for selection
	Name() string

	// Description returns a human-readable description
	Description() string
}

// GeneratorConfig holds common parameters for generators
type GeneratorConfig struct {
	MinRoomSize int
	MaxRoomSize int
	MaxRooms    int
	Seed        int64 // 0 = use time-based seed
}

// DefaultConfig returns sensible defaults for dungeon generation
func DefaultConfig() GeneratorConfig {
	return GeneratorConfig{
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
// Returns nil if not found
func GetGenerator(name string) MapGenerator {
	return generators[name]
}

// GetGeneratorOrDefault retrieves algorithm by name, falls back to default
func GetGeneratorOrDefault(name string) MapGenerator {
	gen := generators[name]
	if gen == nil {
		gen = generators["rooms_corridors"] // Default fallback
	}
	return gen
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

### 2. Rooms & Corridors Generator (gen_rooms_corridors.go)

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
```

### 3. Modified GameMap (dungeongen.go)

```go
// NewGameMap creates a new game map using the specified generator algorithm
func NewGameMap(generatorName string) GameMap {
	images := LoadTileImages() // Returns TileImageSet, not global vars

	dungeonMap := GameMap{
		PlayerVisible: fov.New(),
	}

	// Get generator or fall back to default
	gen := GetGeneratorOrDefault(generatorName)

	// Generate the map
	result := gen.Generate(
		graphics.ScreenInfo.DungeonWidth,
		graphics.ScreenInfo.DungeonHeight,
		images,
	)

	dungeonMap.Tiles = result.Tiles
	dungeonMap.Rooms = result.Rooms
	dungeonMap.NumTiles = len(dungeonMap.Tiles)

	// Update global ValidPos for backward compatibility (temporary)
	ValidPos = ValidPositions{Pos: result.ValidPositions}

	dungeonMap.PlaceStairs()

	return dungeonMap
}

// NewGameMapDefault creates a game map with the default generator
// Provides backward compatibility for existing code
func NewGameMapDefault() GameMap {
	return NewGameMap("rooms_corridors")
}

// REMOVED: GenerateLevelTiles() - no longer needed
// REMOVED: createRoom() - moved to generator
// REMOVED: createHorizontalTunnel() - moved to generator
// REMOVED: createVerticalTunnel() - moved to generator
```

### 4. Updated TileImageSet (GameMapUtil.go)

```go
package worldmap

import (
	"os"
	"path/filepath"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// TileImageSet holds all images needed for tile rendering
type TileImageSet struct {
	WallImages  []*ebiten.Image
	FloorImages []*ebiten.Image
	StairsDown  *ebiten.Image
}

// LoadTileImages loads all tile images from disk
// Returns a TileImageSet instead of setting global variables
func LoadTileImages() TileImageSet {
	images := TileImageSet{
		WallImages:  make([]*ebiten.Image, 0),
		FloorImages: make([]*ebiten.Image, 0),
	}

	// Load floor tiles
	dir := "../assets/tiles/floors/limestone"
	files, _ := os.ReadDir(dir)
	for _, file := range files {
		if !file.IsDir() {
			path := filepath.Join(dir, file.Name())
			img, _, _ := ebitenutil.NewImageFromFile(path)
			images.FloorImages = append(images.FloorImages, img)
		}
	}

	// Load wall tiles
	dir = "../assets/tiles/walls/limestone"
	files, _ = os.ReadDir(dir)
	for _, file := range files {
		if !file.IsDir() {
			path := filepath.Join(dir, file.Name())
			img, _, _ := ebitenutil.NewImageFromFile(path)
			images.WallImages = append(images.WallImages, img)
		}
	}

	// Load stairs
	images.StairsDown, _, _ = ebitenutil.NewImageFromFile("../assets/tiles/stairs_down.png")

	return images
}
```

---

## Testing Strategy

### Unit Tests for Generators

```go
// worldmap/gen_rooms_corridors_test.go
package worldmap

import (
	"testing"
	"game_main/graphics"
)

func TestRoomsAndCorridorsGeneration(t *testing.T) {
	config := GeneratorConfig{
		MinRoomSize: 6,
		MaxRoomSize: 10,
		MaxRooms:    30,
	}

	gen := NewRoomsAndCorridorsGenerator(config)
	images := LoadTileImages()

	result := gen.Generate(80, 50, images)

	// Verify result structure
	if len(result.Tiles) != 80*50 {
		t.Errorf("Expected 4000 tiles, got %d", len(result.Tiles))
	}

	if len(result.Rooms) == 0 {
		t.Error("Expected at least one room")
	}

	// Verify at least some floors exist
	floorCount := 0
	for _, tile := range result.Tiles {
		if tile.TileType == FLOOR {
			floorCount++
		}
	}

	if floorCount == 0 {
		t.Error("No floor tiles generated")
	}

	t.Logf("Generated %d rooms with %d floor tiles", len(result.Rooms), floorCount)
}

func TestRoomsDoNotOverlap(t *testing.T) {
	config := DefaultConfig()
	gen := NewRoomsAndCorridorsGenerator(config)
	images := LoadTileImages()

	result := gen.Generate(80, 50, images)

	// Check all room pairs for overlap
	for i := 0; i < len(result.Rooms); i++ {
		for j := i + 1; j < len(result.Rooms); j++ {
			if result.Rooms[i].Intersect(result.Rooms[j]) {
				t.Errorf("Room %d overlaps with room %d", i, j)
			}
		}
	}
}

func TestValidPositionsMatchFloorTiles(t *testing.T) {
	config := DefaultConfig()
	gen := NewRoomsAndCorridorsGenerator(config)
	images := LoadTileImages()

	result := gen.Generate(80, 50, images)

	// Count floor tiles
	floorCount := 0
	for _, tile := range result.Tiles {
		if tile.TileType == FLOOR {
			floorCount++
		}
	}

	// ValidPositions should match floor count
	if len(result.ValidPositions) != floorCount {
		t.Errorf("ValidPositions count (%d) doesn't match floor tiles (%d)",
			len(result.ValidPositions), floorCount)
	}
}

func BenchmarkRoomsAndCorridors(b *testing.B) {
	config := DefaultConfig()
	gen := NewRoomsAndCorridorsGenerator(config)
	images := LoadTileImages()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate(80, 50, images)
	}
}
```

### Integration Tests

```go
// worldmap/integration_test.go
package worldmap

import (
	"testing"
)

func TestNewGameMapWithGenerator(t *testing.T) {
	gm := NewGameMap("rooms_corridors")

	if len(gm.Tiles) == 0 {
		t.Error("GameMap has no tiles")
	}

	if len(gm.Rooms) == 0 {
		t.Error("GameMap has no rooms")
	}

	// Verify stairs were placed
	hasStairs := false
	for _, tile := range gm.Tiles {
		if tile.TileType == STAIRS_DOWN {
			hasStairs = true
			break
		}
	}

	if !hasStairs {
		t.Error("No stairs placed on map")
	}
}

func TestNewGameMapDefaultBackwardCompatibility(t *testing.T) {
	gm := NewGameMapDefault()

	// Should work exactly like old NewGameMap()
	if len(gm.Tiles) == 0 {
		t.Error("Default GameMap has no tiles")
	}
}

func TestGeneratorRegistry(t *testing.T) {
	generators := ListGenerators()

	if len(generators) == 0 {
		t.Error("No generators registered")
	}

	// Should have at least rooms_corridors
	found := false
	for _, name := range generators {
		if name == "rooms_corridors" {
			found = true
			break
		}
	}

	if !found {
		t.Error("rooms_corridors generator not registered")
	}
}
```

### Testing Checklist

- [ ] Each generator produces valid tiles (correct count)
- [ ] Rooms don't overlap
- [ ] ValidPositions matches floor tiles
- [ ] Maps are navigable (connectivity test with pathfinding)
- [ ] Performance < 50ms per generation
- [ ] Registry works (register, get, list)
- [ ] Backward compatibility (NewGameMapDefault works)
- [ ] ECS compliance (TileContents uses EntityIDs)

---

## Migration Path

### Step-by-Step Migration

#### 1. Find All NewGameMap() Callsites

```bash
# Search for all usages
git grep "NewGameMap()" --name-only
```

Expected locations:
- `game_main/main.go` (initial map creation)
- `worldmap/dungeongen.go` (GoDownStairs function)
- Any test files

#### 2. Update Callsites

**Before**:
```go
dungeonMap := worldmap.NewGameMap()
```

**After (Option A - Use default)**:
```go
dungeonMap := worldmap.NewGameMapDefault()
```

**After (Option B - Specify algorithm)**:
```go
dungeonMap := worldmap.NewGameMap("rooms_corridors")
```

#### 3. Gradual Algorithm Adoption

Once refactoring is complete, you can start using multiple algorithms:

```go
// Level-based algorithm selection
func createMapForLevel(level int) worldmap.GameMap {
	var algorithm string

	switch {
	case level <= 3:
		algorithm = "rooms_corridors"  // Early levels: classic dungeons
	case level <= 6:
		algorithm = "bsp"              // Mid levels: structured layouts
	default:
		algorithm = "cellular"         // Deep levels: cave systems
	}

	return worldmap.NewGameMap(algorithm)
}

// Random algorithm for variety
func createRandomMap() worldmap.GameMap {
	algorithms := worldmap.ListGenerators()
	chosen := algorithms[rand.Intn(len(algorithms))]
	return worldmap.NewGameMap(chosen)
}
```

### Rollback Plan

If issues arise during implementation:

```bash
# Checkpoint before each phase
git tag phase-0-cleanup
git tag phase-1-interface
git tag phase-2-extraction
git tag phase-3-integration

# Rollback if needed
git reset --hard phase-1-interface
```

### Feature Flag Approach

For extra safety, implement feature flag:

```go
// In dungeongen.go
const USE_GENERATOR_PATTERN = true

func NewGameMap(generatorName string) GameMap {
	if !USE_GENERATOR_PATTERN {
		// Old implementation (keep temporarily)
		return oldNewGameMap()
	}

	// New implementation
	// ...
}
```

Remove feature flag after 1-2 release cycles of successful operation.

---

## Second Algorithm Example

### BSP Tree Dungeon Generator

To validate the design, here's a complete second algorithm implementation:

```go
// worldmap/gen_bsp.go
package worldmap

import (
	"game_main/common"
	"game_main/coords"
	"game_main/graphics"
)

// BSPGenerator implements Binary Space Partitioning dungeon generation
// Creates more structured, architectural layouts compared to rooms-and-corridors
type BSPGenerator struct {
	config GeneratorConfig
	minSplitSize int
	maxSplitDepth int
}

// NewBSPGenerator creates a new BSP tree generator
func NewBSPGenerator(config GeneratorConfig) *BSPGenerator {
	return &BSPGenerator{
		config: config,
		minSplitSize: 15,  // Minimum area size before stopping splits
		maxSplitDepth: 4,  // Maximum recursion depth
	}
}

func (g *BSPGenerator) Name() string {
	return "bsp"
}

func (g *BSPGenerator) Description() string {
	return "Binary Space Partitioning: structured architectural layouts with large rooms"
}

type BSPNode struct {
	x, y, w, h int
	left, right *BSPNode
	room *Rect
}

func (g *BSPGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	result := GenerationResult{
		Tiles:          g.createEmptyTiles(width, height, images),
		Rooms:          make([]Rect, 0),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	// Create root node spanning entire map
	root := &BSPNode{x: 1, y: 1, w: width-2, h: height-2}

	// Recursively split space
	g.splitNode(root, 0)

	// Create rooms in leaf nodes
	g.createRoomsInTree(root, &result, images)

	// Connect adjacent rooms
	g.connectRoomsInTree(root, &result, images)

	return result
}

func (g *BSPGenerator) splitNode(node *BSPNode, depth int) {
	// Stop if too deep or area too small
	if depth >= g.maxSplitDepth || node.w < g.minSplitSize || node.h < g.minSplitSize {
		return
	}

	// Decide split orientation based on area shape
	splitHorizontally := false
	if node.w > node.h && node.w / node.h >= 1.25 {
		splitHorizontally = false
	} else if node.h > node.w && node.h / node.w >= 1.25 {
		splitHorizontally = true
	} else {
		splitHorizontally = common.GetDiceRoll(2) == 1
	}

	// Calculate split position
	if splitHorizontally {
		split := common.GetRandomBetween(g.minSplitSize, node.h - g.minSplitSize)
		node.left = &BSPNode{x: node.x, y: node.y, w: node.w, h: split}
		node.right = &BSPNode{x: node.x, y: node.y + split, w: node.w, h: node.h - split}
	} else {
		split := common.GetRandomBetween(g.minSplitSize, node.w - g.minSplitSize)
		node.left = &BSPNode{x: node.x, y: node.y, w: split, h: node.h}
		node.right = &BSPNode{x: node.x + split, y: node.y, w: node.w - split, h: node.h}
	}

	// Recursively split children
	g.splitNode(node.left, depth+1)
	g.splitNode(node.right, depth+1)
}

func (g *BSPGenerator) createRoomsInTree(node *BSPNode, result *GenerationResult, images TileImageSet) {
	if node.left != nil {
		g.createRoomsInTree(node.left, result, images)
	}
	if node.right != nil {
		g.createRoomsInTree(node.right, result, images)
	}

	// Leaf node - create room
	if node.left == nil && node.right == nil {
		roomW := common.GetRandomBetween(g.config.MinRoomSize, min(g.config.MaxRoomSize, node.w-2))
		roomH := common.GetRandomBetween(g.config.MinRoomSize, min(g.config.MaxRoomSize, node.h-2))
		roomX := node.x + common.GetRandomBetween(1, node.w - roomW - 1)
		roomY := node.y + common.GetRandomBetween(1, node.h - roomH - 1)

		room := NewRect(roomX, roomY, roomW, roomH)
		node.room = &room

		g.carveRoom(result, room, images)
		result.Rooms = append(result.Rooms, room)
	}
}

func (g *BSPGenerator) connectRoomsInTree(node *BSPNode, result *GenerationResult, images TileImageSet) {
	if node.left == nil || node.right == nil {
		return
	}

	// Recursively connect children first
	g.connectRoomsInTree(node.left, result, images)
	g.connectRoomsInTree(node.right, result, images)

	// Connect left and right subtrees
	leftRoom := g.getRandomLeafRoom(node.left)
	rightRoom := g.getRandomLeafRoom(node.right)

	if leftRoom != nil && rightRoom != nil {
		x1, y1 := leftRoom.Center()
		x2, y2 := rightRoom.Center()

		// Create corridor
		g.carveHorizontalTunnel(result, x1, x2, y1, images)
		g.carveVerticalTunnel(result, y1, y2, x2, images)
	}
}

func (g *BSPGenerator) getRandomLeafRoom(node *BSPNode) *Rect {
	if node == nil {
		return nil
	}

	if node.room != nil {
		return node.room
	}

	// Randomly choose left or right subtree
	if common.GetDiceRoll(2) == 1 {
		room := g.getRandomLeafRoom(node.left)
		if room != nil {
			return room
		}
		return g.getRandomLeafRoom(node.right)
	} else {
		room := g.getRandomLeafRoom(node.right)
		if room != nil {
			return room
		}
		return g.getRandomLeafRoom(node.left)
	}
}

func (g *BSPGenerator) createEmptyTiles(width, height int, images TileImageSet) []*Tile {
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

func (g *BSPGenerator) carveRoom(result *GenerationResult, room Rect, images TileImageSet) {
	for y := room.Y1 + 1; y < room.Y2; y++ {
		for x := room.X1 + 1; x < room.X2; x++ {
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)

			result.Tiles[index].Blocked = false
			result.Tiles[index].TileType = FLOOR
			result.Tiles[index].image = images.FloorImages[common.GetRandomBetween(0, len(images.FloorImages)-1)]

			result.ValidPositions = append(result.ValidPositions, logicalPos)
		}
	}
}

func (g *BSPGenerator) carveHorizontalTunnel(result *GenerationResult, x1, x2, y int, images TileImageSet) {
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

func (g *BSPGenerator) carveVerticalTunnel(result *GenerationResult, y1, y2, x int, images TileImageSet) {
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

// Register BSP generator
func init() {
	RegisterGenerator(NewBSPGenerator(DefaultConfig()))
}
```

**Usage**:
```go
// Now you can generate BSP dungeons
bspMap := worldmap.NewGameMap("bsp")

// List all available generators
generators := worldmap.ListGenerators()
// Returns: ["rooms_corridors", "bsp"]
```

---

## Benefits Achieved

### 1. Easy Extensibility ✅

**Adding New Algorithm**:
1. Create `gen_<algorithm>.go` file (~150-200 LOC)
2. Implement `MapGenerator` interface (3 methods)
3. Register in `init()` function
4. **Zero changes to existing code**

**Example**: BSP generator above is completely self-contained. No modifications to GameMap, other generators, or registry code.

### 2. Maintainability ✅

**Before Refactoring**:
- GameMap: 550 LOC (generation + data + rendering + FOV)
- Monolithic GenerateLevelTiles(): 46 LOC
- Testing requires full GameMap instance

**After Refactoring**:
- GameMap: ~370 LOC (data + rendering + FOV only)
- Generator files: ~220 LOC each (focused, single responsibility)
- Testing: Each generator testable in isolation

**Code Quality Improvements**:
- Cyclomatic complexity: GameMap reduced from ~15 to ~8
- Clear separation of concerns
- Single responsibility principle applied
- Easier to debug (smaller, focused files)

### 3. Avoid Over-Complexity ✅

**Kept Simple**:
- Interface: 3 methods only (Generate, Name, Description)
- Registry: Simple map[string]Generator
- No builder patterns, no complex composition
- No metadata systems, no capability flags
- No template systems (can add later if needed)

**Complexity Budget**:
- New concepts: 1 (MapGenerator interface)
- New files: 2-3 (generator.go, gen_*.go)
- Learning curve: Low (understand interface, implement 3 methods)

---

## Frequently Asked Questions

### Q: What if I want to customize generation parameters?

**A**: Use GeneratorConfig when creating generators:

```go
config := worldmap.GeneratorConfig{
	MinRoomSize: 8,   // Larger rooms
	MaxRoomSize: 15,
	MaxRooms:    20,  // Fewer but larger
}

customGen := worldmap.NewRoomsAndCorridorsGenerator(config)
worldmap.RegisterGenerator(customGen)
```

### Q: Can I use different algorithms on different levels?

**A**: Yes, pass algorithm name to NewGameMap():

```go
func generateLevel(depth int) worldmap.GameMap {
	algorithm := "rooms_corridors"
	if depth > 5 {
		algorithm = "bsp"
	}
	return worldmap.NewGameMap(algorithm)
}
```

### Q: What about the global ValidPos variable?

**A**: It's populated from GenerationResult.ValidPositions for backward compatibility. Eventually, refactor code to use local positions instead of global.

### Q: How do I test that maps are navigable?

**A**: Use pathfinding integration test:

```go
func TestMapNavigability(t *testing.T) {
	gm := worldmap.NewGameMap("rooms_corridors")
	start := gm.Rooms[0].Center()
	end := gm.Rooms[len(gm.Rooms)-1].Center()

	path := BuildPath(&gm, start, end)
	if path == nil {
		t.Error("Map is not fully connected")
	}
}
```

### Q: What if a generator fails?

**A**: Current implementation doesn't handle errors. You could extend the interface:

```go
type MapGenerator interface {
	Generate(width, height int, images TileImageSet) (GenerationResult, error)
	// ...
}
```

But for initial implementation, generators are expected to always succeed.

### Q: Can I combine multiple algorithms?

**A**: Not directly with this approach. For hybrid generation, see Approach 2 (Functional Composition) in the full analysis. Current design prioritizes simplicity over composition.

---

## Next Steps

### Immediate Actions

1. **Review this document** with team
2. **Estimate effort** based on team velocity (suggested: 14-18 hours)
3. **Create git branch**: `feature/worldmap-generator-pattern`
4. **Start Phase 0**: Global state cleanup

### Success Criteria

- [ ] All existing tests pass
- [ ] New generator tests achieve >80% coverage
- [ ] Map generation performance < 50ms
- [ ] No crashes in 1000 generated maps
- [ ] Second algorithm (BSP) successfully implemented
- [ ] Code review approved
- [ ] Documentation updated (CLAUDE.md)

### Long-Term Vision

After completing this refactoring:
1. Add 2-3 more algorithms (cellular automata, drunkard's walk, maze)
2. Implement level progression system (different algorithms per depth)
3. Add prefab system (special rooms, vaults, boss arenas)
4. Consider data-driven config files for algorithm parameters
5. Add generation metrics (connectivity, openness, difficulty)

---

## Conclusion

This refactoring applies the Strategy Pattern pragmatically:
- **Simple interface** (3 methods)
- **Easy to extend** (new file, zero existing code changes)
- **Maintainable** (clear separation, isolated testing)
- **Not over-engineered** (no unnecessary abstraction)

**Realistic effort**: 14-18 hours including:
- 3h: Global state cleanup
- 3h: Interface and registry
- 4h: Extract current algorithm
- 2h: Integrate with GameMap
- 3h: Add second algorithm
- 1h: Documentation

This sets up the worldmap package for future algorithm variety while keeping the codebase clean and maintainable.

Ready to implement? Start with Phase 0: Pre-Refactoring Cleanup.
