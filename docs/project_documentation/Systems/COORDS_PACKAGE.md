# The `world/coords` Package

**Last Updated:** 2026-03-30

---

## Table of Contents

1. [Overview](#overview)
2. [Coordinate Spaces](#coordinate-spaces)
3. [LogicalPosition](#logicalposition)
4. [PixelPosition](#pixelposition)
5. [CoordinateManager](#coordinatemanager)
   - [Fields and Initialization](#fields-and-initialization)
   - [The Global Instance](#the-global-instance)
   - [ScreenData and Configuration](#screendata-and-configuration)
6. [Core Conversion Methods](#core-conversion-methods)
   - [LogicalToIndex and IndexToLogical](#logicaltoidex-and-indextological)
   - [LogicalToPixel and PixelToLogical](#logicaltopixel-and-pixeltological)
   - [LogicalToScreen and ScreenToLogical](#logicaltoscreen-and-screentological)
   - [IndexToScreen](#indextoscreen)
7. [The Viewport System](#the-viewport-system)
   - [MAP_SCROLLING_ENABLED](#map_scrolling_enabled)
   - [Viewport Struct](#viewport-struct)
   - [How Viewport Centering Works](#how-viewport-centering-works)
8. [Utility Types](#utility-types)
   - [DrawableSection](#drawablesection)
9. [Distance Metrics on LogicalPosition](#distance-metrics-on-logicalposition)
10. [Critical Usage Rules](#critical-usage-rules)
11. [Usage Patterns Across the Codebase](#usage-patterns-across-the-codebase)
12. [Architecture Rationale](#architecture-rationale)

---

## Overview

The `world/coords` package is the authoritative, centralized coordinate system for TinkerRogue. It defines the two fundamental coordinate spaces used by the game — logical tile coordinates and pixel screen coordinates — and provides the `CoordinateManager`, a single global instance that handles all conversions between them.

Every system in the game that touches position, movement, rendering, or spatial queries goes through this package. The design exists to prevent a class of bugs that arose before centralization: code that calculated tile array indices manually using locally-known width values. When those width values diverged from the CoordinateManager's internal `dungeonWidth`, silent index corruption resulted. By routing all index calculations through `CoordManager.LogicalToIndex()`, the game guarantees that every subsystem uses the same source of truth.

The package lives at `world/coords/` and consists of two files:

- `world/coords/cordmanager.go` — `CoordinateManager`, `ScreenData`, `Viewport`, `DrawableSection`, and the global `CoordManager` instance
- `world/coords/position.go` — `LogicalPosition`, `PixelPosition`, and distance/comparison methods

---

## Coordinate Spaces

The game uses three conceptually distinct coordinate spaces:

| Space | Type | Unit | Purpose |
|---|---|---|---|
| **Logical** | `LogicalPosition` | Tile | Game world position, ECS components, movement, AI |
| **Pixel** | `PixelPosition` | Pixel (1x scale) | Pre-viewport rendering, stored in `Tile.PixelX/PixelY` |
| **Screen** | `float64` pair | Pixel (scaled + offset) | Final draw position passed to Ebiten `DrawImage` calls |

Logical coordinates are the primary currency of game logic. An entity at logical position `{X: 5, Y: 3}` occupies the tile in column 5, row 3 of the dungeon grid. Pixel positions are an intermediate representation: the unscaled pixel location a tile would occupy at 1:1 scale. Screen coordinates are what ultimately get drawn and include viewport centering and scaling.

The distinction between pixel and screen is critical in viewport-scrolling mode. When the player moves, screen coordinates shift so the viewport stays centered on the player, but pixel coordinates (and logical coordinates) never change.

---

## LogicalPosition

**File:** `world/coords/position.go:10`

```go
type LogicalPosition struct {
    X, Y int
}
```

`LogicalPosition` is the fundamental position type for all game world entities. It is used in:

- ECS `PositionComponent` attached to every entity that has a map location
- The `GlobalPositionSystem` spatial grid (keyed by `LogicalPosition`)
- All movement commands and validity checks
- AI pathfinding and threat map calculations
- Map generation and tile array lookups

Every ECS entity with a position stores a `*LogicalPosition` as its `PositionComponent` data. The convention throughout the codebase is that game logic always operates in logical coordinates and only converts to pixel or screen coordinates at the final rendering step.

### Why Tile Coordinates?

Using tile indices as the coordinate unit keeps game logic simple and uniform. Movement costs, attack ranges, line-of-sight, and pathfinding all operate in integer tile steps. There is no subpixel game state; an entity is always exactly on a tile.

### Methods on LogicalPosition

`LogicalPosition` carries several spatial utility methods used heavily by AI and combat systems:

```go
func (p *LogicalPosition) IsEqual(other *LogicalPosition) bool
func (p *LogicalPosition) ManhattanDistance(other *LogicalPosition) int
func (p *LogicalPosition) ChebyshevDistance(other *LogicalPosition) int
func (p *LogicalPosition) InRange(other *LogicalPosition, distance int) bool
```

**ManhattanDistance** sums the absolute X and Y differences. It is used for range checks where diagonal movement is not allowed or where a city-block distance model is appropriate — for example, the threat visualizer calculating how far a threat is from a squad.

**ChebyshevDistance** returns the maximum of the absolute X and Y differences. This is the "king moves" metric on a chess board and accurately models movement cost in a game where diagonal movement costs the same as cardinal movement. It is the dominant distance function in this codebase: combat range checks, AI positioning, movement cost calculation, and perk proximity checks all use Chebyshev distance.

**InRange** is a convenience wrapper around `ManhattanDistance` for readability:

```go
// tactical/combat/combatcore/combatactionsystem.go
distance := attackerPos.ChebyshevDistance(&defenderPos)

// mind/behavior/threat_positional.go
distance := enemyPos.ChebyshevDistance(&pos)

// overworld/influence/queries.go
dist := nodes[i].Pos.ManhattanDistance(&nodes[j].Pos)
```

---

## PixelPosition

**File:** `world/coords/position.go:16`

```go
type PixelPosition struct {
    X, Y int
}
```

`PixelPosition` represents a screen position in unscaled pixels. Its primary use is as the intermediate type for `LogicalToPixel` and `PixelToLogical` conversions. It also appears as the position type in `BaseShape` (the spell AOE shape system in `visual/graphics/drawableshapes.go`), where shapes track their cursor position in pixel space and convert to logical coordinates when computing which tiles they cover.

Unlike `LogicalPosition`, `PixelPosition` has no methods and carries no distance utilities. It is purely a type-safe container for pixel coordinates to prevent accidental mixing with logical coordinates or screen coordinates.

---

## CoordinateManager

**File:** `world/coords/cordmanager.go:70`

`CoordinateManager` is the central authority for all coordinate transformations. It owns the canonical values for dungeon dimensions, tile size, scale factor, and screen dimensions, and exposes them through a unified API.

### Fields and Initialization

```go
type CoordinateManager struct {
    dungeonWidth  int
    dungeonHeight int
    tileSize      int
    scaleFactor   int
    screenWidth   int
    screenHeight  int
    viewport      *Viewport
}
```

All fields are private. The outside world reads them only through methods (`GetDungeonWidth()`, `GetDungeonHeight()`, `GetScaledTileSize()`). This encapsulation prevents callers from accidentally caching or shadowing the values.

The `viewport` field holds a single reusable `Viewport` instance, initialized in `NewCoordinateManager`. This avoids allocating a new viewport struct on every frame — an optimization specifically noted in the source (referencing `GUI_PERFORMANCE_ANALYSIS.md`).

### The Global Instance

```go
var CoordManager *CoordinateManager

func init() {
    screenData := NewScreenData()
    CoordManager = NewCoordinateManager(screenData)
}
```

`CoordManager` is package-level global, initialized by the `init()` function before any other code in the package runs. This means it is available immediately when any package that imports `world/coords` starts up, with no manual initialization call required.

The global pattern is intentional. Because tile-based coordinate conversions depend only on the dungeon grid dimensions and tile size — values that are fixed at game configuration time — there is no need to pass the manager as a dependency. Every system that converts coordinates uses the same dungeon geometry, so sharing a global instance is both correct and convenient.

> **Note:** `CLAUDE.md` lists `coords.CoordManager` as a global instance to be used for all tile indexing, alongside `common.GlobalPositionSystem` for spatial queries. The `EntityManager` by contrast is passed as a parameter because it carries mutable game state. `CoordManager` is configuration, not state.

### ScreenData and Configuration

`ScreenData` is a value type that carries the raw configuration numbers used to construct a `CoordinateManager`:

```go
type ScreenData struct {
    ScreenWidth   int
    ScreenHeight  int
    TileSize      int
    DungeonWidth  int
    DungeonHeight int
    ScaleFactor   int
    LevelWidth    int
    LevelHeight   int
    PaddingRight  int
}
```

`NewScreenData()` constructs the default `ScreenData` by reading from `setup/config`:

| Config Variable | Default Value | Description |
|---|---|---|
| `config.DefaultTilePixels` | 32 | Tile side length in pixels at 1x scale |
| `config.DefaultScaleFactor` | 3 | Render scale multiplier in scrolling mode |
| `config.DefaultRightPadding` | 500 | Extra canvas pixels added to the right for the UI panel |
| Hardcoded `DungeonWidth` | 100 | Map width in tiles |
| Hardcoded `DungeonHeight` | 80 | Map height in tiles |

These defaults can be overridden by loading `gameconfig.json` through `templates.ReadGameData()`, which calls `config.SetConfigFromJSON()`. However, `CoordManager` is initialized by `init()` before JSON loading occurs. This means any JSON override of tile pixels or scale factor does not retroactively update `CoordManager`. In practice the defaults match the JSON values; this is a known limitation flagged by a `TODO` comment in `NewScreenData()`.

`ScreenData` also exposes two computed-width helpers:

```go
func (s ScreenData) GetCanvasWidth() int  // TileSize * DungeonWidth + PaddingRight
func (s ScreenData) GetCanvasHeight() int // TileSize * DungeonHeight
```

These are used in game layout calculations to determine the total Ebiten canvas size.

---

## Core Conversion Methods

### LogicalToIndex and IndexToLogical

**File:** `world/coords/cordmanager.go:107`

```go
func (cm *CoordinateManager) LogicalToIndex(pos LogicalPosition) int {
    return (pos.Y * cm.dungeonWidth) + pos.X
}

func (cm *CoordinateManager) IndexToLogical(index int) LogicalPosition {
    x := index % cm.dungeonWidth
    y := index / cm.dungeonWidth
    return LogicalPosition{X: x, Y: y}
}
```

These two methods are the most critical in the entire package. The game map is stored as a flat `[]*Tile` slice, not a 2D array. Converting between a `(X, Y)` tile coordinate and the correct slice index requires knowing the row stride, which is `dungeonWidth`. `LogicalToIndex` encodes this formula; `IndexToLogical` inverts it.

**Why this is safety-critical:** Any code that computes the index manually — `y*width + x` — must pass in the same `width` value that was used to allocate the tile slice. If that local `width` variable ever diverges from `cm.dungeonWidth`, the index is wrong. Because the tile slice is allocated to length `dungeonWidth * dungeonHeight`, a wrong index either silently reads the wrong tile or causes an index-out-of-range panic at runtime. By making `LogicalToIndex` the only place this formula lives, there is a single value (`cm.dungeonWidth`) and a single formula, eliminating the divergence hazard.

This is important enough that `CLAUDE.md` calls it out explicitly in the Critical Warnings section:

> Always use `coords.CoordManager.LogicalToIndex()` for tile arrays. Width may differ from `CoordinateManager.dungeonWidth`!

**Usage across the codebase:**

`LogicalToIndex` is called in over 20 places spanning map generation, rendering, input handling, AI, the save system, and the overworld. A representative sample:

```go
// world/worldmap/dungeongen.go — reading a tile during generation
index := coords.CoordManager.LogicalToIndex(logicalPos)

// visual/rendering/rendering.go — full-map render mode
index := coords.CoordManager.LogicalToIndex(logicalPos)
tile := gameMap.Tiles[index]

// input/cameracontroller.go — collision check before player movement
index := coords.CoordManager.LogicalToIndex(nextLogicalPos)
nextTile := mc.gameMap.Tiles[index]

// overworld/core/walkability.go — overworld terrain grid
idx := coords.CoordManager.LogicalToIndex(pos)
WalkableGrid[idx] = walkable

// setup/savesystem/chunks/map_chunk.go — serializing/deserializing map data
idx := coords.CoordManager.LogicalToIndex(logicalPos)
```

`IndexToLogical` is used less frequently, primarily where index-based iteration needs to recover the `(X, Y)` coordinates — for example in `gui/guispells/spell_handler.go` where a flat index produced by a shape calculation is converted back to a position for rendering:

```go
logicalPos := coords.CoordManager.IndexToLogical(idx)
```

### LogicalToPixel and PixelToLogical

**File:** `world/coords/cordmanager.go:119`

```go
func (cm *CoordinateManager) LogicalToPixel(pos LogicalPosition) PixelPosition {
    return PixelPosition{
        X: pos.X * cm.tileSize,
        Y: pos.Y * cm.tileSize,
    }
}

func (cm *CoordinateManager) PixelToLogical(pos PixelPosition) LogicalPosition {
    return LogicalPosition{
        X: pos.X / cm.tileSize,
        Y: pos.Y / cm.tileSize,
    }
}
```

These conversions move between logical tile coordinates and 1x-scale pixel coordinates. The formula is simple: multiply by `tileSize` to get pixels, divide by `tileSize` to get tiles. At the default tile size of 32 pixels, logical position `{5, 3}` maps to pixel position `{160, 96}`.

These are primarily intermediate steps used inside `LogicalToScreen` and `ScreenToLogical`. They also appear directly in a few places:

- `gui/guispells/spell_handler.go` — converting a target logical position to pixel coordinates for spell effect placement
- `visual/graphics/drawableshapes.go` — converting pixel cursor position to logical tile for shape index calculation, and vice versa when computing line endpoints

**`PixelToLogical` uses integer division**, which truncates toward zero. This means any pixel within a tile maps to that tile's origin. For example, pixel `{170, 100}` (which lies within the tile at logical `{5, 3}` if tiles are 32px) maps to `{5, 3}`. This is the correct behavior for hit testing: any pixel click within a tile should identify that tile.

### LogicalToScreen and ScreenToLogical

**File:** `world/coords/cordmanager.go:223`

```go
func (cm *CoordinateManager) LogicalToScreen(
    pos LogicalPosition,
    centerPos *LogicalPosition,
) (float64, float64)

func (cm *CoordinateManager) ScreenToLogical(
    screenX, screenY int,
    centerPos *LogicalPosition,
) LogicalPosition
```

These are the high-level conversion functions used by rendering code. They handle both rendering modes transparently:

**When `centerPos` is `nil` or `MAP_SCROLLING_ENABLED` is `false`:**
The function applies only the scale factor (1x in non-scrolling mode) and returns a straightforward scaled pixel position. This is full-map mode, where the entire dungeon is visible on screen at once.

**When `centerPos` is non-nil and `MAP_SCROLLING_ENABLED` is `true`:**
The function delegates to the internal `Viewport.LogicalToScreen()`, which centers the view on `centerPos` and applies the configured scale factor (3x by default). The `centerPos` is typically the player's current position.

The `centerPos *LogicalPosition` parameter is a pointer so that `nil` can serve as a sentinel for "full map mode," avoiding the need for a separate function or a boolean flag parameter.

**Callers use this method whenever they need to know where to draw something:**

```go
// visual/rendering/rendering.go — viewport mode rendering
screenX, screenY := coords.CoordManager.LogicalToScreen(logicalPos, vp.centerPos)

// visual/rendering/tilerenderer.go — tile batch rendering
screenX, screenY := coords.CoordManager.LogicalToScreen(tileLogicalPos, center)

// gui/guispells/spell_handler.go — spell AOE overlay drawing
sx, sy := coords.CoordManager.LogicalToScreen(pos, playerPos)
```

`ScreenToLogical` is used for the reverse: converting a mouse click position (in screen pixels) back to the logical tile the user clicked on. This is the input side of every mouse-based interaction.

```go
// visual/graphics/graphictypes.go — mouse click to tile
return coords.CoordManager.ScreenToLogical(mouseX, mouseY, &centerPos)

// gui/guioverworld/overworld_renderer.go — overworld map click
logicalPos := r.ScreenToLogical(screenX, screenY)
```

### IndexToScreen

**File:** `world/coords/cordmanager.go:277`

```go
func (cm *CoordinateManager) IndexToScreen(
    index int,
    centerPos *LogicalPosition,
) (float64, float64) {
    return cm.LogicalToScreen(cm.IndexToLogical(index), centerPos)
}
```

A convenience combinator that chains `IndexToLogical` into `LogicalToScreen`. Used in `visual/graphics/vxhandler.go` for drawing visual effects (VFX) that are keyed by flat tile index:

```go
sx, sy := coords.CoordManager.IndexToScreen(ind, &centerPos)
```

---

## The Viewport System

### MAP_SCROLLING_ENABLED

**File:** `world/coords/cordmanager.go:12`

```go
var MAP_SCROLLING_ENABLED = true
```

This package-level variable acts as a global mode switch. When `true`, the game renders a scaled, player-centered viewport. When `false`, it renders the entire dungeon at 1x scale. It can be toggled at runtime with a key binding (handled in `input/cameracontroller.go`).

The variable lives in the `coords` package rather than `config` because it is read by the coordinate conversion functions themselves, and those functions cannot import `config` without creating an import cycle (config imports nothing; coords would need to import config, and config already imports nothing from coords, so this is actually safe — but the design choice keeps the mode flag colocated with the code that consults it).

### Viewport Struct

**File:** `world/coords/cordmanager.go:82`

```go
type Viewport struct {
    centerX, centerY int
    manager          *CoordinateManager
}
```

`Viewport` encapsulates the scrolling camera state: where the viewport is centered. It holds the logical coordinates of the center point and a reference to its parent `CoordinateManager` so it can access tile size and scale factor.

A single `Viewport` instance is held inside `CoordinateManager.viewport`. It is created once in `NewCoordinateManager` and reused across all frames. The center is updated via `SetCenter()` before any conversion call:

```go
func (v *Viewport) SetCenter(pos LogicalPosition) {
    v.centerX = pos.X
    v.centerY = pos.Y
}
```

This update-then-use pattern avoids allocation while keeping the viewport current.

### How Viewport Centering Works

`Viewport.LogicalToScreen()` applies three transformations in sequence:

1. **Convert logical to pixel:** multiply `pos.X` and `pos.Y` by `tileSize`
2. **Calculate centering offset:** subtract the center's pixel position scaled to screen scale, then add half the screen width/height
3. **Apply scale and offset:** multiply pixel position by `scaleFactor`, then add the centering offset

The formula in the source:

```go
centerPixelX := float64(v.centerX * v.manager.tileSize)
centerPixelY := float64(v.centerY * v.manager.tileSize)

offsetX := float64(v.manager.screenWidth)/2 - centerPixelX*float64(v.manager.scaleFactor)
offsetY := float64(v.manager.screenHeight)/2 - centerPixelY*float64(v.manager.scaleFactor)

scaledX := pixelX * float64(v.manager.scaleFactor)
scaledY := pixelY * float64(v.manager.scaleFactor)

return scaledX + offsetX, scaledY + offsetY
```

The offset calculation `screenWidth/2 - centerPixelX*scaleFactor` produces the translation that puts the center tile's top-left corner at the center of the screen minus half a scaled tile. When this offset is added to any other tile's scaled pixel position, tiles to the left of center have negative offsets (drawing left of center) and tiles to the right have positive offsets. The result is that `centerPos` always draws at the middle of the screen.

`Viewport.ScreenToLogical()` inverts this: it strips the offset, reverses the scale, and calls `PixelToLogical()` to recover the tile.

**Screen dimensions are updated every frame** via `UpdateScreenDimensions()`, called from the Ebiten `Draw()` callback. This ensures that if the window is resized, the offset calculation stays correct.

```go
// game_main/main.go — called every frame
coords.CoordManager.UpdateScreenDimensions(screen.Bounds().Dx(), screen.Bounds().Dy())
```

The `visual/rendering/viewport.go` `ViewportRenderer` also calls this in `NewViewportRenderer()`, providing a second update point for subsystems that create their own renderer per frame.

---

## Utility Types

### DrawableSection

**File:** `world/coords/cordmanager.go:196`

```go
type DrawableSection struct {
    StartX int
    StartY int
    EndX   int
    EndY   int
}

func NewDrawableSection(centerX, centerY, size int) DrawableSection {
    halfSize := size / 2
    return DrawableSection{
        StartX: centerX - halfSize,
        StartY: centerY - halfSize,
        EndX:   centerX + halfSize,
        EndY:   centerY + halfSize,
    }
}
```

`DrawableSection` defines a rectangular region of the logical map using inclusive logical coordinates. It is used by the rendering system to cull entities outside the visible viewport, avoiding drawing calls for tiles and sprites that are off-screen.

`NewDrawableSection` creates a square region of side length `size` centered on `(centerX, centerY)`. In practice `centerX`/`centerY` is the player's logical position, and `size` is `config.DefaultZoomNumberOfSquare` (typically 30), so only the 30x30 tile region around the player is considered for rendering.

```go
// visual/rendering/rendering.go
sq := coords.NewDrawableSection(playerPos.X, playerPos.Y, squareSize)

// visual/rendering/tilerenderer.go
sq := coords.NewDrawableSection(opts.CenterOn.X, opts.CenterOn.Y, opts.ViewportSize)
```

The `DrawableSection` is then used as a bounds filter in the rendering loop:

```go
if pos.X < vp.section.StartX || pos.X > vp.section.EndX ||
    pos.Y < vp.section.StartY || pos.Y > vp.section.EndY {
    continue
}
```

Note that `DrawableSection` does not clamp to map bounds. It can produce negative or out-of-bounds coordinates when the player is near a map edge. Calling code is responsible for range-checking before using a `DrawableSection` index.

---

## Distance Metrics on LogicalPosition

Two distance metrics are available on `LogicalPosition`:

**Manhattan Distance** (`|dx| + |dy|`): Appropriate when movement is strictly 4-directional (no diagonals). Used in:
- Threat visualization (distance from threat zone to squad)
- Overworld influence node spacing

**Chebyshev Distance** (`max(|dx|, |dy|)`): Appropriate when diagonal movement costs the same as cardinal movement. This is the standard in TinkerRogue's tactical system. Used in:
- Combat attack range validation
- Movement cost calculation
- AI unit positioning and threat evaluation
- Perk proximity checks

When choosing which metric to use, the rule of thumb in this codebase is: if it involves tactical combat or movement on the dungeon grid, use Chebyshev. If it involves overworld distances where diagonal shortcuts are not modeled, use Manhattan.

---

## Critical Usage Rules

### Rule 1: Always Use LogicalToIndex for Tile Array Access

Never compute a tile index as `y*width + x` using a locally-known width. Always use:

```go
// Correct
idx := coords.CoordManager.LogicalToIndex(pos)
tile := gameMap.Tiles[idx]

// Wrong — width might not match cm.dungeonWidth
idx := pos.Y*mapWidth + pos.X
tile := gameMap.Tiles[idx]  // silent wrong tile or panic
```

The `dungeonWidth` field in `CoordinateManager` is the authoritative map row stride. Any other width value you have in scope may come from a parameter, a struct field, or a local calculation that drifts from the canonical value over time.

### Rule 2: LogicalPosition is the ECS Position Type

The ECS `PositionComponent` stores `*coords.LogicalPosition`. When you retrieve a position from an entity, you have a logical coordinate. Convert to pixel or screen only at the rendering boundary. Do not store pixel or screen coordinates in ECS components or game state.

### Rule 3: Call UpdateScreenDimensions Each Draw Frame

The viewport centering offset depends on the current screen width and height. Because Ebiten windows can be resized, these must be refreshed. The main `Draw()` function in `game_main/main.go` does this, and `ViewportRenderer.NewViewportRenderer()` does it as well. If you write a new rendering path that bypasses both, call:

```go
coords.CoordManager.UpdateScreenDimensions(screen.Bounds().Dx(), screen.Bounds().Dy())
```

### Rule 4: Pass nil centerPos for Full-Map Mode

When writing code that supports both scrolling and full-map modes, pass `nil` as the `centerPos` argument to `LogicalToScreen` and `ScreenToLogical` to select full-map behavior:

```go
// Scrolling mode — centers on player
sx, sy := coords.CoordManager.LogicalToScreen(pos, &playerPos)

// Full map mode — no centering, no extra scaling
sx, sy := coords.CoordManager.LogicalToScreen(pos, nil)
```

---

## Usage Patterns Across the Codebase

### Pattern: Tile Lookup from Movement Input

The camera controller checks tile walkability before moving the player. It constructs a `LogicalPosition` for the destination, calls `LogicalToIndex`, and inspects the resulting tile:

```go
// input/cameracontroller.go
nextLogicalPos := coords.LogicalPosition{X: nextPosition.X, Y: nextPosition.Y}
index := coords.CoordManager.LogicalToIndex(nextLogicalPos)
nextTile := mc.gameMap.Tiles[index]

if !nextTile.Blocked {
    mc.playerData.Pos.X = nextPosition.X
    // ...
}
```

### Pattern: Map Generation

During dungeon generation, every tile is initialized by converting an `(x, y)` loop variable to a logical position, then to an index:

```go
// world/worldmap/gen_helpers.go
logicalPos := coords.LogicalPosition{X: x, Y: y}
index := positionToIndex(x, y)  // wraps CoordManager.LogicalToIndex
tileValues[index] = NewTile(x*tileSize, y*tileSize, logicalPos, ...)
tiles[index] = &tileValues[index]
```

The pixel position stored in the tile (`x*tileSize`, `y*tileSize`) is the 1x-scale pixel position and is used only in non-scrolling full-map mode rendering. In scrolling mode, the tile's logical position is reconverted to screen coordinates at draw time.

### Pattern: Rendering Entities in Viewport Mode

The rendering system computes where to draw each visible entity:

```go
// visual/rendering/rendering.go
logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}
screenX, screenY := coords.CoordManager.LogicalToScreen(logicalPos, vp.centerPos)
scale := float32(graphics.ScreenInfo.ScaleFactor)
dstX = float32(screenX)
dstY = float32(screenY)
dstW = srcW * scale
dstH = srcH * scale
```

### Pattern: ViewportRenderer Wrapper

`visual/rendering/viewport.go` provides `ViewportRenderer`, a thin wrapper that stores a persistent `centerPos` so callers don't need to pass it on every call:

```go
func (vr *ViewportRenderer) LogicalToScreen(pos coords.LogicalPosition) (float64, float64) {
    return coords.CoordManager.LogicalToScreen(pos, &vr.centerPos)
}
```

This is the preferred interface for subsystems (like `combatoverlays.go`) that render many things at once and want to avoid threading `centerPos` through every helper function.

### Pattern: Mouse Click to Tile

Input handlers convert raw mouse screen coordinates back to logical tiles:

```go
// visual/graphics/graphictypes.go
return coords.CoordManager.ScreenToLogical(mouseX, mouseY, &centerPos)
```

### Pattern: Overworld and Walkability Grids

Systems that maintain their own flat boolean arrays use `LogicalToIndex` consistently to ensure grid indices align with the tile slice layout:

```go
// overworld/core/walkability.go
idx := coords.CoordManager.LogicalToIndex(pos)
WalkableGrid[idx] = walkable
```

### Pattern: Spell AOE Shape Indices

The spell system's AOE shapes (`visual/graphics/drawableshapes.go`) track their center in pixel coordinates (matching mouse cursor position). When computing which tiles a shape covers, they convert pixel to logical, then logical to index for each tile in the shape area:

```go
func (s *BaseShape) GetIndices() []int {
    logical := coords.CoordManager.PixelToLogical(s.Position)
    // ... calculate shape tiles in logical space ...
    indices = append(indices, coords.CoordManager.LogicalToIndex(...))
}
```

---

## Architecture Rationale

### Why a Global Instance?

The coordinate system is configuration, not mutable game state. The dungeon is always `100 x 80` tiles with 32-pixel tiles (by default). These values do not change during a play session. Making `CoordManager` a global avoids threading an extra parameter through every function that touches position — a burden that would span dozens of packages.

The analogy in `CLAUDE.md` is instructive: `EntityManager` is passed as a parameter because it holds mutable ECS state that must be explicit. `CoordManager` is listed alongside it as a global because it holds only configuration.

### Why PixelPosition is Separate from Screen Coordinates

`PixelPosition` is typed as a distinct struct from a bare `(int, int)` pair. This gives the compiler the ability to catch mix-ups between pixel coordinates (unscaled, not yet offset) and screen coordinates (scaled and offset). Returning `(float64, float64)` for screen coordinates further distinguishes them from the `int`-based pixel type. The type distinction communicates intent and prevents certain categories of rendering bugs.

### Why LogicalToScreen Takes a *LogicalPosition Instead of LogicalPosition

Passing a pointer allows `nil` to represent "no viewport center, use full-map mode." A zero value `LogicalPosition{0, 0}` would be ambiguous — it could mean the viewport is centered on tile zero, or it could mean no centering is wanted. The pointer sentinel eliminates that ambiguity at the cost of a small indirection.

### The Viewport Reuse Optimization

Rather than creating a new `Viewport` struct on every `LogicalToScreen` call, `CoordinateManager` keeps a single `viewport` field and updates its center before use. At the rendering frame rates of a game, allocating small structs repeatedly contributes to GC pressure. The single-instance pattern keeps the hot rendering path allocation-free.
