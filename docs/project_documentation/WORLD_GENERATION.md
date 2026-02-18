# World Generation

**Version:** 2.0
**Last Updated:** 2026-02-18
**Package:** `game_main/world/worldmap`

---

## Overview

The world generation system lives in `world/worldmap/` and provides procedural map generation through a plugin-style registry. Any generator implementing the `MapGenerator` interface can be registered by name and selected at runtime. The system currently ships four generators covering dungeon rooms, organic caves, tactical biome combat maps, and large-scale strategic overworlds.

Map generation feeds directly into the `GameMap` struct, which owns the tile array, room list, valid position cache, biome map, POI list, and faction start positions used by the rest of the game.

---

## Table of Contents

1. [Directory Structure](#1-directory-structure)
2. [Core Types](#2-core-types)
3. [Generator Interface and Registry](#3-generator-interface-and-registry)
4. [Available Generators](#4-available-generators)
5. [Shared Generation Helpers](#5-shared-generation-helpers)
6. [Tile System](#6-tile-system)
7. [Biome System](#7-biome-system)
8. [Image Loading and Asset Configuration](#8-image-loading-and-asset-configuration)
9. [Coordinate System](#9-coordinate-system)
10. [A* Pathfinding](#10-a-pathfinding)
11. [Game Map API](#11-game-map-api)
12. [Game Mode Integration](#12-game-mode-integration)
13. [Adding a New Generator](#13-adding-a-new-generator)

---

## 1. Directory Structure

```
world/worldmap/
    generator.go           # MapGenerator interface, GenerationResult, registry functions
    dungeongen.go          # GameMap struct, NewGameMap, tile helpers, PlaceStairs, A* bridge
    dungeontile.go         # Tile struct, TileType constants, NewTile
    biome.go               # Biome int enum (5 values), String() method
    tileconfig.go          # Asset paths, POI constants, allBiomes list
    GameMapUtil.go         # TileImageSet/BiomeTileSet structs, LoadTileImages()
    gen_rooms_corridors.go # RoomsAndCorridorsGenerator ("rooms_corridors")
    gen_cavern.go          # CavernGenerator ("cavern")
    gen_tactical_biome.go  # TacticalBiomeGenerator ("tactical_biome")
    gen_overworld.go       # StrategicOverworldGenerator ("overworld")
    gen_helpers.go         # Shared: createEmptyTiles, carveRoom, carveTunnel,
                           #         floodFillRegion, carveCorridorBetween,
                           #         ensureTerrainConnectivity
    astar.go               # AStar struct, GetPath, BuildPath

world/coords/
    cordmanager.go         # CoordinateManager, Viewport, ScreenData, LogicalToIndex
    position.go            # LogicalPosition, PixelPosition, distance helpers
```

---

## 2. Core Types

### GenerationResult

`GenerationResult` is the value every generator returns from `Generate()`. All fields are populated or left as their zero values depending on what the generator produces.

```go
// world/worldmap/generator.go
type GenerationResult struct {
    Tiles                 []*Tile
    Rooms                 []Rect
    ValidPositions        []coords.LogicalPosition
    POIs                  []POIData
    FactionStartPositions []FactionStartPosition
    BiomeMap              []Biome   // flat array, indexed by positionToIndex(x, y, width)
}
```

Field descriptions:

| Field | Type | Description |
|---|---|---|
| `Tiles` | `[]*Tile` | Flat tile array, indexed using `coords.CoordManager.LogicalToIndex()` |
| `Rooms` | `[]Rect` | Rectangular areas; index 0 is the player start room for room-based generators |
| `ValidPositions` | `[]coords.LogicalPosition` | All non-blocked positions; used for spawning and stair placement |
| `POIs` | `[]POIData` | Named points of interest with position and biome |
| `FactionStartPositions` | `[]FactionStartPosition` | Generator-chosen positions for faction spawning |
| `BiomeMap` | `[]Biome` | Per-tile biome value in the same flat index space as `Tiles` |

### POIData

```go
// world/worldmap/generator.go
type POIData struct {
    Position coords.LogicalPosition
    NodeID   string  // One of: "town", "temple", "guild_hall", "watchtower"
    Biome    Biome
}
```

### FactionStartPosition

```go
// world/worldmap/generator.go
type FactionStartPosition struct {
    Position coords.LogicalPosition
    Biome    Biome
    Sector   int  // Which map sector (0-4)
}
```

### Rect

`Rect` is the fundamental room/area type used by all room-based generators.

```go
// world/worldmap/dungeongen.go
type Rect struct {
    X1, X2, Y1, Y2 int
}

func NewRect(x, y, width, height int) Rect

func (r Rect) IsInRoom(x, y int) bool
func (r Rect) GetCoordinates() []coords.LogicalPosition
func (r Rect) GetCoordinatesWithoutCenter() []coords.LogicalPosition
func (r *Rect) Center() (int, int)
func (r *Rect) Intersect(other Rect) bool
```

`GetCoordinates()` excludes wall tiles (inner tiles only). `GetCoordinatesWithoutCenter()` additionally excludes the center point and is used for monster placement.

### GeneratorConfig

Shared configuration used by the rooms-and-corridors and tactical-biome generators.

```go
// world/worldmap/generator.go
type GeneratorConfig struct {
    MinRoomSize int
    MaxRoomSize int
    MaxRooms    int
    Seed        int64  // 0 = time-based seed
}

func DefaultConfig() GeneratorConfig {
    return GeneratorConfig{
        MinRoomSize: 6,
        MaxRoomSize: 10,
        MaxRooms:    30,
        Seed:        0,
    }
}
```

---

## 3. Generator Interface and Registry

### Interface

```go
// world/worldmap/generator.go
type MapGenerator interface {
    Generate(width, height int, images TileImageSet) GenerationResult
    Name() string
    Description() string
}
```

- `Name()` returns the string key used for registry lookup (for example, `"rooms_corridors"`).
- `Generate()` receives the map dimensions and the pre-loaded tile image set; it returns a fully-populated `GenerationResult`.

### Registry

```go
// world/worldmap/generator.go

// RegisterGenerator adds a generator to the registry.
// Key is gen.Name().
func RegisterGenerator(gen MapGenerator)

// GetGeneratorOrDefault looks up the generator by name.
// Falls back to "rooms_corridors" if the name is not found.
func GetGeneratorOrDefault(name string) MapGenerator
```

There is no `GetGenerator()` function. The only lookup function is `GetGeneratorOrDefault`.

### Self-Registration Pattern

Every generator registers itself in its own `init()` function. This means importing the `worldmap` package is sufficient to make all generators available; no manual registration calls are needed in application code.

```go
// Example from world/worldmap/gen_rooms_corridors.go
func init() {
    RegisterGenerator(NewRoomsAndCorridorsGenerator(DefaultConfig()))
}
```

The same pattern appears in `gen_cavern.go`, `gen_tactical_biome.go`, and `gen_overworld.go`.

---

## 4. Available Generators

### 4.1 rooms_corridors

**File:** `world/worldmap/gen_rooms_corridors.go`
**Key:** `"rooms_corridors"`
**Default:** Yes — `GetGeneratorOrDefault` falls back to this generator.

Classic roguelike dungeon generation. Produces rectangular rooms connected by single-tile-wide L-shaped corridors.

**Algorithm — collision rejection, not BSP:**

1. Attempt to place `MaxRooms` rooms (default 30).
2. For each attempt, generate a room with random size (6-10 tiles) at a random position.
3. Reject the room if it intersects any already-placed room (`Intersect` check).
4. If accepted, carve the room floor tiles and connect to the previous room with an L-shaped corridor.
5. The L-shape orientation (horizontal-first vs vertical-first) is chosen randomly.

```go
type RoomsAndCorridorsGenerator struct {
    config GeneratorConfig
}

func NewRoomsAndCorridorsGenerator(config GeneratorConfig) *RoomsAndCorridorsGenerator

func (g *RoomsAndCorridorsGenerator) Name() string        // "rooms_corridors"
func (g *RoomsAndCorridorsGenerator) Description() string // "Classic roguelike: rectangular rooms connected by L-shaped corridors"
func (g *RoomsAndCorridorsGenerator) Generate(width, height int, images TileImageSet) GenerationResult
```

**Output:** Populates `Tiles`, `Rooms`, and `ValidPositions`. `BiomeMap`, `POIs`, and `FactionStartPositions` are empty/nil.

**Player start:** `GameMap.StartingPosition()` returns the center of `Rooms[0]`.

---

### 4.2 cavern

**File:** `world/worldmap/gen_cavern.go`
**Key:** `"cavern"`

Organic cave layouts designed for squad-based tactical combat. Produces large open chambers connected by corridors wide enough for multi-unit movement (3-5 tiles) while maintaining chokepoints.

**Algorithm:**

1. Seed guaranteed circular chambers across a 3x2 sector grid (adaptive to map size).
2. Randomly fill non-chamber tiles using `FillDensity` (default 0.42).
3. Apply cellular automata smoothing (default 6 iterations). Rule: 5+ wall neighbors out of 8 neighbors makes a tile a wall.
4. Re-stamp chambers at radius minus one to preserve interiors eroded by CA.
5. Widen narrow passages: any walkable tile with fewer than 8 walkable tiles in its 5x5 neighborhood gets expanded by opening a 3x3 area around it (capped at 70% total walkable).
6. Enforce solid border walls of `BorderThickness` (default 2).
7. Run shared `ensureTerrainConnectivity` to join disconnected regions.
8. One more widening pass and border enforcement after connectivity.
9. Place 2x2 pillar cover clusters inside chambers (proportional to chamber area times `PillarDensity`).
10. Place faction start positions on opposite ends of the map.
11. Convert boolean terrain map to `Tile` objects with `BiomeMountain` biome throughout.

```go
type CavernGenerator struct {
    config CavernConfig
}

type CavernConfig struct {
    FillDensity      float64  // Initial random wall density (default 0.42)
    NumChambers      int      // Target chambers (default 5, actual 4-6 after optional drop)
    MinChamberRadius int      // Minimum radius in tiles (default 8)
    MaxChamberRadius int      // Maximum radius in tiles (default 12)
    CAIterations     int      // Cellular automata passes (default 6)
    MinPassageWidth  int      // Minimum passage width (default 3)
    PillarDensity    float64  // Cover pillar density per chamber (default 0.03)
    BorderThickness  int      // Solid wall border (default 2)
}

func DefaultCavernConfig() CavernConfig
```

**Output:** Populates `Tiles`, `Rooms` (chambers recorded as bounding rects), `ValidPositions`, `FactionStartPositions` (two positions, left and right zones), and `BiomeMap` (all `BiomeMountain`). `POIs` is empty.

**Faction start placement:** Scores walkable tiles in the left quarter and right quarter of the map by openness in a 6-tile scan radius, selects the best in each zone, and clears a 6-tile circular area around each selected position.

---

### 4.3 tactical_biome

**File:** `world/worldmap/gen_tactical_biome.go`
**Key:** `"tactical_biome"`

Generates single-biome tactical battle maps using cellular automata. The biome is selected randomly each generation. Each biome has a `BiomeTacticalProfile` that controls obstacle density and which tactical features are added.

**Algorithm:**

1. Randomly select a biome from `allBiomes`.
2. Look up the biome's `BiomeTacticalProfile`.
3. Initialize terrain randomly at `obstacleDensity` (fraction blocked).
4. Apply 5 iterations of cellular automata smoothing (same 5-neighbor rule as cavern generator).
5. Conditionally apply tactical features based on profile:
   - `hasCover`: scatter small 2x2 to 3x3 obstacle clusters (~1% of tile count).
   - `hasChokePoints`: impose 3-5 column or row barriers with 70% fill probability.
   - `hasOpenSpace`: carve 2-3 circular open areas with radius 5-7.
6. Run shared `ensureTerrainConnectivity`.
7. Convert to tiles using biome-specific images.
8. Clear a 5-tile spawn area at map center.

```go
type TacticalBiomeGenerator struct {
    config GeneratorConfig
}

type BiomeTacticalProfile struct {
    obstacleDensity float64
    hasCover        bool
    hasElevation    bool
    hasChokePoints  bool
    hasOpenSpace    bool
    hasRoughTerrain bool
    floorVariants   int
    wallVariants    int
}
```

Obstacle densities by biome:

| Biome | `obstacleDensity` | Key Characteristics |
|---|---|---|
| `BiomeGrassland` | 0.20 | Open space, scattered cover, gentle elevation |
| `BiomeForest` | 0.35 | Dense, choke points from tree clusters, rough terrain |
| `BiomeDesert` | 0.15 | Very open, minimal cover, rough terrain |
| `BiomeMountain` | 0.45 | Rocky, many choke points, high ground |
| `BiomeSwamp` | 0.30 | Islands of dry land, choke points, rough terrain |

**Output:** Populates `Tiles` and `ValidPositions`. `Rooms`, `BiomeMap`, `POIs`, and `FactionStartPositions` are empty.

---

### 4.4 overworld

**File:** `world/worldmap/gen_overworld.go`
**Key:** `"overworld"`

Large-scale strategic world map generation using multi-octave fractal Brownian motion noise. Produces continent-shaped landmasses with biome-classified terrain, terrain-aware POI placement, and four faction starting positions.

**Algorithm:**

1. Generate two independent fBm noise maps: elevation (4 octaves, scale 0.035) and moisture (3 octaves, scale 0.045). Both use OpenSimplex noise via `opensimplex-go`.
2. Apply continent shaping: radial distance falloff from map center reduces elevation by up to 60% at the edges, pushing map borders toward water biomes.
3. Classify each tile into a biome using the elevation/moisture thresholds described below.
4. Build a boolean walkability map and run `ensureTerrainConnectivity`; corridors carved by connectivity become `BiomeGrassland` floor tiles.
5. Place four faction starting positions in corner sectors (top-left, top-right, bottom-right, bottom-left), each chosen by scoring walkable tiles with the most walkable neighbors in a 5-tile radius. Only `BiomeGrassland` and `BiomeForest` tiles qualify.
6. Place typed POIs in order: towns (3), temples (2), watchtowers (3), guild halls (2). Guild halls must be within 20 tiles of a town.

```go
type StrategicOverworldGenerator struct {
    config StrategicOverworldConfig
}

type StrategicOverworldConfig struct {
    ElevationOctaves int      // Default 4
    ElevationScale   float64  // Default 0.035
    MoistureOctaves  int      // Default 3
    MoistureScale    float64  // Default 0.045
    Persistence      float64  // Amplitude decay per octave, default 0.5
    Lacunarity       float64  // Frequency multiplier per octave, default 2.0

    WaterThresh    float64  // Elevation below this = BiomeSwamp (water), default 0.28
    MountainThresh float64  // Elevation above this = BiomeMountain, default 0.72
    ForestMoisture float64  // Moisture above this = BiomeForest, default 0.55
    SwampMoisture  float64  // Moisture above this at low elevation = BiomeSwamp, default 0.70

    TownCount       int  // Default 3
    TempleCount     int  // Default 2
    GuildHallCount  int  // Default 2
    WatchtowerCount int  // Default 3
    POIMinDistance  int  // Minimum tile distance between POIs, default 12

    FactionCount      int  // Default 4
    FactionMinSpacing int  // Default 25

    Seed int64  // 0 = time.Now().UnixNano()
}

func DefaultStrategicOverworldConfig() StrategicOverworldConfig
func NewStrategicOverworldGenerator(config StrategicOverworldConfig) *StrategicOverworldGenerator
```

**Biome classification logic (`determineBiome`):**

```
elevation < WaterThresh (0.28)                          → BiomeSwamp (impassable water)
elevation > MountainThresh (0.72)                       → BiomeMountain (impassable rock)
moisture > SwampMoisture (0.70) AND elevation < 0.40    → BiomeSwamp
elevation > 0.60 AND moisture < 0.35                    → BiomeDesert
moisture > ForestMoisture (0.55)                        → BiomeForest
otherwise                                               → BiomeGrassland
```

`BiomeSwamp` and `BiomeMountain` produce blocked wall tiles. `BiomeDesert`, `BiomeForest`, and `BiomeGrassland` produce walkable floor tiles.

**POI terrain rules:**

| POI Type | Valid Terrain |
|---|---|
| `town` | `BiomeGrassland` or `BiomeForest` |
| `temple` | Elevation > 0.55, not swamp or mountain; or `BiomeDesert` |
| `watchtower` | Elevation > 0.50, not swamp or mountain |
| `guild_hall` | Any walkable biome; must be within 20 tiles of a town |

**Output:** Fully populates all `GenerationResult` fields: `Tiles`, `Rooms` (1x1 rects for each POI, used by `StartingPosition`/`PlaceStairs`), `ValidPositions`, `POIs`, `FactionStartPositions`, and `BiomeMap`.

---

## 5. Shared Generation Helpers

`gen_helpers.go` provides package-level helper functions shared by multiple generators.

### Tile Initialization

```go
// createEmptyTiles initializes a width*height tile array, all walls.
// Allocates all Tile values in one contiguous slice to reduce GC pressure,
// then stores pointers into that slice.
func createEmptyTiles(width, height int, images TileImageSet) []*Tile
```

The contiguous allocation strategy avoids thousands of individual heap allocations during generation.

### Room and Corridor Carving

```go
// carveRoom converts wall tiles to floor tiles inside room boundaries (excluding outer wall row/column).
func carveRoom(result *GenerationResult, room Rect, width int, images TileImageSet)

// carveHorizontalTunnel carves a single-tile-wide horizontal corridor from x1 to x2 at row y.
func carveHorizontalTunnel(result *GenerationResult, x1, x2, y, width int, images TileImageSet)

// carveVerticalTunnel carves a single-tile-wide vertical corridor from y1 to y2 at column x.
func carveVerticalTunnel(result *GenerationResult, y1, y2, x, width int, images TileImageSet)
```

All carving functions update `result.Tiles` in-place and append to `result.ValidPositions`.

### Index Conversion

```go
// positionToIndex converts x, y to a flat array index using row-major order.
// Note: generators use this internally. External callers should use
// coords.CoordManager.LogicalToIndex() to guarantee consistency.
func positionToIndex(x, y, width int) int {
    return y*width + x
}
```

### Image Selection

```go
// selectRandomImage returns a random element from the slice, or nil if empty.
func selectRandomImage(images []*ebiten.Image) *ebiten.Image

// getBiomeImages returns wall and floor image slices for a biome,
// falling back to the default wall/floor images if biome-specific images are absent.
func getBiomeImages(images TileImageSet, biome Biome) (wallImages, floorImages []*ebiten.Image)
```

### Connectivity

```go
// ensureTerrainConnectivity finds all disconnected walkable regions and connects
// each to the largest region using L-shaped corridors carved into the terrain map.
// If no walkable region exists, opens the central 50% of the map.
func ensureTerrainConnectivity(terrainMap []bool, width, height int)

// floodFillRegion returns flat indices of all walkable tiles reachable from (startX, startY).
func floodFillRegion(terrainMap []bool, visited []bool, startX, startY, width, height int) []int

// carveCorridorBetween creates an L-shaped corridor in the boolean terrain map
// between two flat indices (horizontal segment first, then vertical).
func carveCorridorBetween(terrainMap []bool, width, height, fromIdx, toIdx int)
```

`ensureTerrainConnectivity` is called by the cavern, tactical-biome, and overworld generators. It operates on the boolean terrain map, not directly on `GenerationResult.Tiles`; callers are responsible for synchronizing tile state after the call.

---

## 6. Tile System

### TileType

```go
// world/worldmap/dungeontile.go
type TileType int

const (
    WALL       TileType = iota  // 0 — blocked, opaque
    FLOOR                       // 1 — walkable
    STAIRS_DOWN                 // 2 — walkable, triggers level descent
)
```

There are exactly three tile types. `STAIRS_DOWN` is placed post-generation by `GameMap.PlaceStairs()`.

### Tile

```go
// world/worldmap/dungeontile.go
type Tile struct {
    PixelX       int
    PixelY       int
    TileCords    coords.LogicalPosition
    Blocked      bool
    Image        *ebiten.Image
    tileContents TileContents      // unexported: holds entity IDs on the tile
    TileType     TileType
    IsRevealed   bool
    cm           graphics.ColorMatrix  // unexported: color tint
    Biome        Biome
    POIType      string  // "town", "temple", etc., or "" if not a POI
}

func NewTile(pixelX, pixelY int, tileCords coords.LogicalPosition,
    blocked bool, img *ebiten.Image, tileType TileType, isRevealed bool) Tile
```

**Caution:** `NewTile` hardcodes `Blocked = true` and `TileType = WALL` regardless of parameters. Callers that need a non-wall tile must set those fields explicitly after construction. This is done in all generator `convertToTiles` methods.

### TileContents

```go
type TileContents struct {
    EntityIDs []ecs.EntityID
}
```

`TileContents` stores ECS entity IDs (not pointers) for items resting on the tile. Currently used only for item pickup.

---

## 7. Biome System

```go
// world/worldmap/biome.go
type Biome int

const (
    BiomeGrassland Biome = iota  // 0
    BiomeForest                  // 1
    BiomeDesert                  // 2
    BiomeMountain                // 3
    BiomeSwamp                   // 4
)

func (b Biome) String() string
// Returns: "grassland", "forest", "desert", "mountain", "swamp", or "unknown"
```

The canonical ordered list is defined in `tileconfig.go`:

```go
var allBiomes = []Biome{BiomeGrassland, BiomeForest, BiomeDesert, BiomeMountain, BiomeSwamp}
```

`allBiomes` is the single source of truth used by `LoadTileImages()` and `TacticalBiomeGenerator.selectBiome()`.

Biomes serve two roles:

1. **Overworld terrain classification** — determines whether a tile is walkable (desert, forest, grassland) or blocked (mountain, swamp acting as water).
2. **Visual variety** — each biome loads its own wall and floor image directories from `assets/tiles/floors/<biome>` and `assets/tiles/walls/<biome>`.

---

## 8. Image Loading and Asset Configuration

### TileImageSet and BiomeTileSet

```go
// world/worldmap/GameMapUtil.go
type TileImageSet struct {
    WallImages  []*ebiten.Image
    FloorImages []*ebiten.Image
    StairsDown  *ebiten.Image
    BiomeImages map[Biome]*BiomeTileSet
    POIImages   map[string]*ebiten.Image
}

type BiomeTileSet struct {
    WallImages  []*ebiten.Image
    FloorImages []*ebiten.Image
}
```

`LoadTileImages()` is called once at `NewGameMap` time and the resulting `TileImageSet` is passed to `Generate()`. Generators call `getBiomeImages(images, biome)` to obtain the correct images for each tile.

### Asset Paths

Defined in `tileconfig.go`:

```go
var tileAssetBase = filepath.Join("..", "assets", "tiles")
var defaultFloorDir = "limestone"
var defaultWallDir  = "marble"
var stairsFile      = "stairs1.png"
```

Path construction functions:

```go
func defaultFloorPath() string         // ../assets/tiles/floors/limestone
func defaultWallPath() string          // ../assets/tiles/walls/marble
func stairsPath() string               // ../assets/tiles/stairs1.png
func biomeFloorPath(biome Biome) string // ../assets/tiles/floors/<biome.String()>
func biomeWallPath(biome Biome) string  // ../assets/tiles/walls/<biome.String()>
func poiAssetPath(poiType string) string
```

### POI Constants

```go
// world/worldmap/tileconfig.go
const (
    POITown       = "town"
    POITemple     = "temple"
    POIGuildHall  = "guild_hall"
    POIWatchtower = "watchtower"
)
```

POI asset mapping:

```go
var poiAssetConfig = map[string]string{
    POITown:       filepath.Join("maptiles", "town",       "dithmenos2.png"),
    POITemple:     filepath.Join("maptiles", "temple",     "golden_statue_1.png"),
    POIGuildHall:  filepath.Join("maptiles", "guild_hall", "machine_tukima.png"),
    POIWatchtower: filepath.Join("maptiles", "watchtower", "crumbled_column_1.png"),
}
```

POI images are loaded into `TileImageSet.POIImages` keyed by the POI type string. When the overworld generator places a POI, it replaces the tile's `Image` with the POI-specific image and sets `Tile.POIType`.

---

## 9. Coordinate System

The world generation system depends on two coordinate types from `world/coords/`.

### LogicalPosition and PixelPosition

```go
// world/coords/position.go
type LogicalPosition struct{ X, Y int }
type PixelPosition   struct{ X, Y int }

func (p *LogicalPosition) IsEqual(other *LogicalPosition) bool
func (p *LogicalPosition) ManhattanDistance(other *LogicalPosition) int
func (p *LogicalPosition) ChebyshevDistance(other *LogicalPosition) int
func (p *LogicalPosition) InRange(other *LogicalPosition, distance int) bool
```

All map coordinates, entity positions, and room coordinates use `LogicalPosition`. `PixelPosition` is only used within rendering code.

### CoordinateManager

```go
// world/coords/cordmanager.go
var CoordManager *CoordinateManager  // global singleton, initialized in init()

// LogicalToIndex converts a LogicalPosition to the flat tile array index.
// This is the ONLY safe way to index into GameMap.Tiles from outside the worldmap package.
func (cm *CoordinateManager) LogicalToIndex(pos LogicalPosition) int {
    return (pos.Y * cm.dungeonWidth) + pos.X
}

// IndexToLogical is the inverse.
func (cm *CoordinateManager) IndexToLogical(index int) LogicalPosition

// LogicalToPixel and PixelToLogical convert between coordinate spaces.
func (cm *CoordinateManager) LogicalToPixel(pos LogicalPosition) PixelPosition
func (cm *CoordinateManager) PixelToLogical(pos PixelPosition) LogicalPosition
```

Default map dimensions (from `config/config.go` via `cordmanager.go`):

```
DungeonWidth  = 100
DungeonHeight = 80
TileSize      = 32 pixels
ScaleFactor   = 3
```

**Critical:** Always use `coords.CoordManager.LogicalToIndex()` when indexing `GameMap.Tiles` from outside generator code. The `positionToIndex(x, y, width)` helper in `gen_helpers.go` is for generator-internal use only, where the local `width` parameter is guaranteed to match.

---

## 10. A* Pathfinding

The A* implementation in `astar.go` operates on `GameMap` values (not pointers, noted as a TODO in the code).

```go
// world/worldmap/astar.go
type AStar struct{}

// GetPath finds the shortest path from start to end.
// ignoreWalls=false: path avoids WALL tiles.
// ignoreWalls=true:  path passes through all tile types.
// Returns nil if no path exists.
func (as AStar) GetPath(
    gameMap     GameMap,
    start       *coords.LogicalPosition,
    end         *coords.LogicalPosition,
    ignoreWalls bool,
) []coords.LogicalPosition

// BuildPath is a convenience wrapper that calls GetPath with ignoreWalls=false.
func BuildPath(gm *GameMap, start, other *coords.LogicalPosition) []coords.LogicalPosition
```

**Algorithm details:**
- Expands four cardinal neighbors (no diagonals).
- Cost function: `g` = steps from start, `h` = Chebyshev distance to goal, `f = g + h`.
- Open and closed lists are plain slices; membership checks are linear scans. Performance degrades on large open maps.
- Traversability is determined by `tile.TileType == WALL`, not `tile.Blocked`.

---

## 11. Game Map API

### Construction

```go
// world/worldmap/dungeongen.go

// NewGameMap creates a GameMap using the named generator.
// Falls back to "rooms_corridors" if the name is not registered.
func NewGameMap(generatorName string) GameMap

// NewGameMapDefault is a convenience wrapper for "rooms_corridors".
func NewGameMapDefault() GameMap
```

`NewGameMap` calls `LoadTileImages()`, runs the generator, copies all result fields to `GameMap`, and then calls `PlaceStairs`.

### GameMap

```go
type GameMap struct {
    Tiles                 []*Tile
    Rooms                 []Rect
    NumTiles              int
    RightEdgeX            int
    TopEdgeY              int
    ValidPositions        []coords.LogicalPosition
    BiomeMap              []Biome
    POIs                  []POIData
    FactionStartPositions []FactionStartPosition
    TileColorsDirty       bool
}
```

### Key Methods

```go
// Tile looks up a tile by logical position.
// Uses coords.CoordManager.LogicalToIndex internally.
func (gameMap *GameMap) Tile(pos *coords.LogicalPosition) *Tile

// StartingPosition returns the player start position.
// For room-based generators: center of Rooms[0].
// For non-room generators: map center if walkable, then first ValidPosition, then map center as fallback.
func (gameMap *GameMap) StartingPosition() coords.LogicalPosition

// PlaceStairs places STAIRS_DOWN at the center of a random room (index >= 1).
// Falls back to a random ValidPosition for non-room generators.
func (gm *GameMap) PlaceStairs(images TileImageSet)

// GetBiomeAt returns the biome at a position, defaulting to BiomeGrassland.
func (gm *GameMap) GetBiomeAt(pos coords.LogicalPosition) Biome

// IsOpaque returns true if the tile at (x, y) is a WALL (used by FOV).
func (gameMap GameMap) IsOpaque(x, y int) bool

// InBounds checks if (x, y) is within DungeonWidth/DungeonHeight.
func (gameMap GameMap) InBounds(x, y int) bool

// AddEntityToTile adds an entity ID to a tile's TileContents.
func (gameMap *GameMap) AddEntityToTile(entity *ecs.Entity, pos *coords.LogicalPosition)

// RemoveItemFromTile removes the entity at index from a tile's TileContents.
// Returns the EntityID (ECS-compliant, no entity pointer).
func (gameMap *GameMap) RemoveItemFromTile(index int, pos *coords.LogicalPosition) (ecs.EntityID, error)

// ApplyColorMatrix sets a ColorMatrix on tiles at the given indices.
// Sets TileColorsDirty = true to trigger a render pass.
func (gameMap *GameMap) ApplyColorMatrix(indices []int, m graphics.ColorMatrix)
func (gameMap *GameMap) ApplyColorMatrixToIndex(index int, m graphics.ColorMatrix)

// UnblockedIndices returns indices of non-blocked tiles in a square area centered at (pixelX, pixelY).
func (gameMap *GameMap) UnblockedIndices(pixelX, pixelY, size int) []int
func (gameMap *GameMap) UnblockedLogicalCoords(pixelX, pixelY, size int) []coords.LogicalPosition
```

---

## 12. Game Mode Integration

### Overworld Mode

The overworld generator is invoked from `game_main/setup_overworld.go`:

```go
// game_main/setup_overworld.go
func SetupOverworldMode(g *Game) {
    boot := NewGameBootstrap()
    boot.CreateWorld(&g.gameMap, "overworld")  // calls NewGameMap("overworld")
    // ...
}
```

After map creation, `InitializeGameplay` registers the walkable grid with the pathfinding overlay:

```go
core.InitWalkableGrid(config.DefaultMapWidth, config.DefaultMapHeight)
for _, pos := range gm.ValidPositions {
    core.SetTileWalkable(pos, true)
}
```

### POI-to-Node Conversion

POIs from `GenerationResult.POIs` become neutral overworld nodes at startup:

```go
// game_main/setup_overworld.go
func (gb *GameBootstrap) ConvertPOIsToNodes(em *common.EntityManager, gm *worldmap.GameMap) {
    for _, poi := range gm.POIs {
        node.CreateNode(em, node.CreateNodeParams{
            Position:   poi.Position,
            NodeTypeID: poi.NodeID,   // "town", "temple", "guild_hall", "watchtower"
            OwnerID:    core.OwnerNeutral,
            CurrentTick: currentTick,
        })
    }
}
```

This bridges world generation (tile and visual layer) to the overworld systems (influence, garrison, threat suppression).

### Faction Start Positions

`FactionStartPositions` from `GenerationResult` are used by `bootstrap.InitializeOverworldFactions` to place faction entities at generator-chosen positions rather than hardcoded coordinates.

---

## 13. Adding a New Generator

1. Create a new file in `world/worldmap/`, for example `gen_my_layout.go`.
2. Define a config struct if the generator has tunable parameters.
3. Implement the `MapGenerator` interface:

```go
type MyGenerator struct {
    config MyConfig
}

func (g *MyGenerator) Name() string        { return "my_layout" }
func (g *MyGenerator) Description() string { return "One-sentence description" }

func (g *MyGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
    result := GenerationResult{
        Tiles:          createEmptyTiles(width, height, images),
        Rooms:          make([]Rect, 0),
        ValidPositions: make([]coords.LogicalPosition, 0),
    }
    // ... generation logic ...
    return result
}
```

4. Register in `init()`:

```go
func init() {
    RegisterGenerator(&MyGenerator{config: defaultMyConfig()})
}
```

5. Use `GetGeneratorOrDefault("my_layout")` or pass `"my_layout"` to `NewGameMap("my_layout")` to activate it.

**Guidelines:**

- Use `createEmptyTiles` to initialize the tile array; it ensures a single contiguous allocation.
- Index into the tile array with `positionToIndex(x, y, width)` inside the generator, or `coords.CoordManager.LogicalToIndex(pos)` from outside.
- Call `ensureTerrainConnectivity(terrainMap, width, height)` before finalizing tiles if you use a boolean intermediate map.
- After `ensureTerrainConnectivity` modifies the terrain map, re-synchronize `result.Tiles` by re-scanning the terrain map (see `applyConnectivityFixes` in `gen_overworld.go` for the pattern).
- `NewTile` hardcodes `Blocked = true` and `TileType = WALL`; set floor tiles explicitly after construction.
- Populate `FactionStartPositions` if you want factions to spawn at generator-chosen positions rather than falling back to defaults.
- Populate `POIs` if you want overworld nodes to be created from generation output.
