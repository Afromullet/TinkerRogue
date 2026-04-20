# Visual Packages Technical Reference

**Package paths:** `game_main/visual/graphics`, `game_main/visual/rendering`
**Last updated:** 2026-03-25

---

## Table of Contents

1. [Overview](#overview)
2. [Package: graphics](#package-graphics)
   - [Core Abstraction: VisualEffect](#core-abstraction-visualeffect)
   - [BaseEffect: Lifecycle and Timing](#baseeffect-lifecycle-and-timing)
   - [Animators](#animators)
   - [Renderers](#renderers)
   - [Effect Factory: vxfactory](#effect-factory-vxfactory)
   - [VisualEffectHandler and VisualEffectArea](#visualeffecthandler-and-visualeffectarea)
   - [Drawable Shapes](#drawable-shapes)
   - [ColorMatrix](#colormatrix)
   - [Global Aliases](#global-aliases)
3. [Package: rendering](#package-rendering)
   - [QuadBatch: The GPU Batching Primitive](#quadbatch-the-gpu-batching-primitive)
   - [TileRenderer: Map Tile Rendering](#tilerenderer-map-tile-rendering)
   - [Map Rendering Entry Points](#map-rendering-entry-points)
   - [RenderingCache: Sprite Batching for ECS Entities](#renderingcache-sprite-batching-for-ecs-entities)
   - [Renderable ECS Component](#renderable-ecs-component)
   - [Entity Rendering Entry Points](#entity-rendering-entry-points)
   - [ViewportRenderer and Coordinate Utilities](#viewportrenderer-and-coordinate-utilities)
   - [Combat Overlays: Movement and Health Bars](#combat-overlays-movement-and-health-bars)
   - [Squad Highlight Renderer](#squad-highlight-renderer)
   - [SquadCombatRenderer: The Combat Animation Screen](#squadcombatrenderer-the-combat-animation-screen)
   - [Render Data Interfaces](#render-data-interfaces)
4. [Cross-Package Design Patterns](#cross-package-design-patterns)
5. [Integration with the Game Loop](#integration-with-the-game-loop)
6. [Integration with the World Layer](#integration-with-the-world-layer)
7. [Integration with GUI Systems](#integration-with-gui-systems)
8. [Data Flow Diagram](#data-flow-diagram)
9. [Performance Considerations](#performance-considerations)
10. [Common Extension Points](#common-extension-points)

---

## Overview

The `visual/` directory contains the two packages responsible for everything drawn to the screen in TinkerRogue that is not part of the EbitenUI widget tree. Together they form a layered rendering subsystem sitting between the game's ECS data model and the Ebiten GPU API.

**`visual/graphics`** is the lower-level package. It defines abstractions for animated visual effects (spells, projectiles, explosions), the geometric shapes used to select tiles for area effects, and the `ColorMatrix` type used to tint map tiles. It also provides a small set of global aliases that re-export coordinate and screen state from `world/coords` for convenience.

**`visual/rendering`** is the higher-level package. It converts ECS component data and world-map tile data into draw calls, groups them into GPU-efficient batches via `QuadBatch`, and provides a collection of focused renderers for specific game elements: map tiles, entity sprites, squad highlights, health bars, movement overlays, and the combat animation screen.

Neither package reaches into game logic. They are consumers of data, not producers of it. The only ECS dependency in `rendering` is reading `common.Renderable` and `common.PositionComponent` from entities; it never writes to components or modifies game state.

---

## Package: graphics

**Import path:** `game_main/visual/graphics`
**Source files:** `animators.go`, `colormatrix.go`, `drawableshapes.go`, `graphictypes.go`, `renderers.go`, `vx.go`, `vxfactory.go`, `vxhandler.go`

### Core Abstraction: VisualEffect

`vx.go` defines the interface that all visual effects implement:

```go
type VisualEffect interface {
    UpdateVisualEffect()
    DrawVisualEffect(screen *ebiten.Image)
    SetVXCommon(x, y int, img *ebiten.Image)
    IsCompleted() bool
    VXImg() *ebiten.Image
    ResetVX()
    Copy() VisualEffect
}
```

The design separates the three concerns of a visual effect into dedicated types:

- **Lifecycle and position** (`BaseEffect`) — start time, duration, screen coordinates, completion flag.
- **Animation** (`Animator` interface) — transforms time elapsed into an `AnimationState` (scale, opacity, brightness, positional offset).
- **Drawing** (`Renderer` interface) — applies an `AnimationState` to the GPU using `ebiten.DrawImage` or vector primitives.

The `AnimationState` struct is the data-transfer object between these two concerns:

```go
type AnimationState struct {
    Scale      float64
    Opacity    float64
    Brightness float64
    ColorShift float64
    OffsetX    float64
    OffsetY    float64
}
```

Zero values are safe: an effect with no animator will pass `{Scale: 1.0, Opacity: 1.0}` to its renderer.

### BaseEffect: Lifecycle and Timing

`vx.go` also contains `BaseEffect`, the concrete struct that satisfies `VisualEffect` and holds all lifecycle state. It is not meant to be used directly; `NewEffect` in `vxfactory.go` constructs it.

Key lifecycle rules:

- `UpdateVisualEffect` marks `completed = true` when `time.Since(startTime) >= duration`. It returns early if already completed.
- `DrawVisualEffect` delegates to the animator first (to get `AnimationState`), then to the renderer. It also returns early if completed.
- `ResetVX` resets `startTime` to `time.Now()`, clears `completed`, and calls `animator.Reset()`. This allows effects to be replayed without allocation.
- `Copy()` performs a shallow copy of the entire struct. Because animators and renderers are either stateless or carry only frame-counter fields, this is safe for duplicating a prototype effect across multiple tile positions (used by `VisualEffectArea`).

### Animators

`animators.go` provides five concrete `Animator` implementations. All implement `Update(effect *BaseEffect, elapsed float64) AnimationState` and `Reset()`.

| Type | Behavior | Used by |
|---|---|---|
| `RandomAnimator` | Randomizes scale, opacity, brightness, color shift, and positional jitter each frame within configured ranges. Zero-value ranges are skipped. | Fire, ice, electricity effects |
| `SineShimmerAnimator` | Uses a `math.Sin` phase to produce smooth shimmering on scale and color shift channels. | `NewIceEffect2` |
| `PulseAnimator` | Two independent sine waves: one for scale, one for opacity, advancing at configurable speeds. | Cloud effects |
| `MotionAnimator` | Moves a position linearly from start to end at a fixed speed. Marks the parent `BaseEffect.completed` when within one step of the target. | Projectiles |
| `WaveAnimator` | Slow sine-wave modulation on opacity only. | Sticky ground effects |

`RandomAnimator` is the most general. Its zero-value handling means callers configure only the channels they need and omit the rest — no interface proliferation for each combination.

`MotionAnimator` requires a constructor (`NewMotionAnimator`) because it carries mutable current-position state:

```go
func NewMotionAnimator(startX, startY, endX, endY int, speed float64) *MotionAnimator
```

It is the only animator that can terminate an effect through a mechanism other than duration expiry.

### Renderers

`renderers.go` provides six concrete `Renderer` implementations. All implement `Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState)`.

| Type | Technique | Notes |
|---|---|---|
| `ImageRenderer` | `DrawImage` with `GeoM` scale/translate and `ColorM` adjustments for brightness and color shift | General purpose; used by most image-based effects |
| `ProjectileRenderer` | `DrawImage` with `Atan2`-based rotation and center-pivot translation | Computes angle from `effect.startY` to `endY`, rotating the sprite to face the direction of travel |
| `CloudRenderer` | `DrawImage` called three times: main layer plus two scaled-down layers at 30% opacity for "fluffiness" | Deliberately multi-draw; the layering produces a volumetric illusion |
| `LineSegmentRenderer` | `ebitenutil.DrawLine` for each pre-generated segment; jitters endpoints each frame | Procedural, no image asset; segments allocated at construction |
| `ElectricArcRenderer` | `vector.StrokeLine` with thickness for each segment; re-generates 10 segments from start to end each frame | Procedural arc from one point to another; randomizes color and thickness |
| `ProceduralRenderer` | `DrawImage` of freshly created circle images in a rotating formation | Used for sticky ground; least GPU-friendly due to per-frame image creation |

`NewLineSegmentRenderer` and `NewElectricArcRenderer` both require constructors because they carry geometry state.

### Effect Factory: vxfactory

`vxfactory.go` provides the public API for creating concrete visual effects. The central function is:

```go
func NewEffect(startX, startY, duration int, cfg EffectConfig) VisualEffect
```

`EffectConfig` is a plain struct:

```go
type EffectConfig struct {
    ImagePath string   // "" means procedural effect (no image loaded)
    Animator  Animator // nil means no animation (static image at full opacity)
    Renderer  Renderer // required
}
```

All named effect constructors delegate to `NewEffect` with a pre-configured `EffectConfig`. This means adding a new effect type requires only a new function — no changes to `BaseEffect`, animators, or renderers unless new behavior is needed.

Named constructors and their configurations:

| Function | Asset | Animator | Renderer |
|---|---|---|---|
| `NewFireEffect` | `effects/cloud_fire2.png` | `RandomAnimator` (scale, opacity, jitter) | `ImageRenderer` |
| `NewIceEffect` | `effects/frost0.png` | `RandomAnimator` (scale, opacity, color shift) | `ImageRenderer` |
| `NewIceEffect2` | `effects/frost0.png` | `SineShimmerAnimator` | `ImageRenderer` |
| `NewCloudEffect` | `effects/cloud_poison0.png` | `PulseAnimator` | `CloudRenderer` |
| `NewElectricityEffect` | `effects/zap0.png` | `RandomAnimator` (scale, brightness, jitter) | `ImageRenderer` |
| `NewStickyGroundEffect` | none | `WaveAnimator` | `ProceduralRenderer` |
| `NewProjectile` | `effects/arrow3.png` | `MotionAnimator` | `ProjectileRenderer` |
| `NewElectricityEffectNoImage` | none | none | `LineSegmentRenderer` |
| `NewElectricArc` | none | none | `ElectricArcRenderer` |

`CreateVisualEffectByType(vxType string, x, y, duration int) VisualEffect` is a string-dispatch wrapper for the four main spell effect types ("fire", "ice", "electricity", "cloud"). This decouples the spell system from knowing specific constructor names; the spell template specifies the effect type as a string.

### VisualEffectHandler and VisualEffectArea

`vxhandler.go` provides the runtime container for live effects.

**`VisualEffectHandler`** is the top-level manager. A single global instance, `VXHandler`, lives here. It holds two slices:

- `vx []VisualEffect` — single-point effects.
- `vxArea []VisualEffectArea` — area effects spanning multiple tile positions.

Package-level convenience functions wrap the global:

```go
func AddVX(a VisualEffect)
func AddVXArea(a VisualEffectArea)
```

The handler's game-loop integration methods are:

- `UpdateVisualEffects()` — calls `clearVisualEffects()` first (removes completed entries by building a new slice of survivors), then calls `UpdateVisualEffect()` on all remaining effects.
- `DrawVisualEffects(screen *ebiten.Image)` — calls `DrawVisualEffect` on every live effect.

These two methods are called directly from `game_main/main.go`'s `Update` and `Draw` functions respectively.

**`VisualEffectArea`** applies one effect to every tile covered by a `TileBasedShape`. Construction takes a prototype `VisualEffect`; it calls `Copy()` to stamp one instance per tile, each positioned at the correct screen coordinate:

```go
func NewVisualEffectArea(centerX, centerY int, shape TileBasedShape, vx VisualEffect) VisualEffectArea
```

Completion is determined by the first element of the effects slice — when that is done, the whole area is considered done. This works because all copies have the same duration and start time.

### Drawable Shapes

`drawableshapes.go` defines the geometry system that determines which map tiles are affected by area spells and how visual effects are positioned across them.

**`TileBasedShape`** interface:

```go
type TileBasedShape interface {
    GetIndices() []int
    UpdatePosition(pixelX, pixelY int)
    StartPositionPixels() (int, int)
    GetDirection() ShapeDirection
    CanRotate() bool
}
```

`GetIndices()` is the core method. It returns flat tile-array indices (using `coords.CoordManager.LogicalToIndex`) for all tiles covered by the shape at its current position. These indices are used both for visual effect placement (in `VisualEffectArea`) and for applying `ColorMatrix` overlays to map tiles (in the spell targeting handler).

**`BaseShape`** is the single concrete implementation, parameterized by a `BasicShapeType` enum:

| `BasicShapeType` | Fields used | Algorithm |
|---|---|---|
| `Circular` | `Size` (radius) | `x² + y² <= r²` Bresenham-style |
| `Rectangular` | `Width`, `Height` | Nested loop over `[-halfW, halfW] x [-halfH, halfH]` |
| `Linear` | `Size` (length), `Direction` | Step along a direction vector |

Factory functions create `BaseShape` values with size randomized within a `ShapeSize` category (Small, Medium, Large). This randomization happens at construction time, not per-frame, so the shape is stable once created.

**`ShapeDirection`** is an 8-way compass enum (up, down, left, right, four diagonals). `RotateRight` and `RotateLeft` advance the direction through a fixed ordered slice, implementing rotation as index arithmetic. `DirectionToCoords` maps a direction to `(deltaX, deltaY)` unit vectors.

**`ShapeConfig`** is a JSON-deserializable struct used by the spell template system to describe shapes declaratively. `CreateShapeFromConfig` converts it to a `TileBasedShape`.

`GetLineTo(start, end LogicalPosition) []int` is a standalone helper that uses pixel-space interpolation to compute a line of tile indices between two logical positions. It is used for line-of-sight and line-based spell targeting rather than for instancing `BaseShape`.

### ColorMatrix

`colormatrix.go` defines the `ColorMatrix` type used to tint individual map tiles:

```go
type ColorMatrix struct {
    R, G, B, A  float32
    ApplyMatrix bool
}
```

`ApplyMatrix` is a gate flag. When false, the tile renderer skips color application entirely (the zero value of the struct means "no tint"). When true, the RGBA values are passed directly to `QuadBatch.Add` as color modulation values.

The file provides nine gradient constructors, one for each semantic use in the tactical map overlay system:

| Function | Color | Semantic Use |
|---|---|---|
| `CreateRedGradient` | Red | Danger, high threat |
| `CreateBlueGradient` | Blue | Expected damage zones |
| `CreateGreenGradient` | Green | Allied units, safe zones |
| `CreateMagentaGradient` | Magenta | Mixed ally/enemy tiles |
| `CreateYellowGradient` | Yellow | Selected squad position |
| `CreateOrangeGradient` | Orange | Melee threat zones |
| `CreateCyanGradient` | Cyan | Ranged fire zones |
| `CreatePurpleGradient` | Purple | Isolation risk |
| `CreateRedOrangeGradient` | Red-orange | Engagement pressure |

Two pre-constructed constants, `GreenColorMatrix` and `RedColorMatrix`, are defined for use without an opacity parameter.

`NewEmptyMatrix()` creates a zero-value matrix with `ApplyMatrix: true`, used to clear a previously applied tint.

### Global Aliases

`graphictypes.go` re-exports two values from `world/coords` under the `graphics` package name:

```go
var ScreenInfo = coords.NewScreenData()
var CoordManager = coords.CoordManager
```

`MouseToLogicalPosition` provides a single call site for converting mouse screen coordinates to logical tile coordinates, delegating to `coords.CoordManager.ScreenToLogical`.

These aliases exist so that GUI and rendering code can import only `visual/graphics` rather than also importing `world/coords` for these common operations.

---

## Package: rendering

**Import path:** `game_main/visual/rendering`
**Source files:** `combatoverlays.go`, `maprendering.go`, `quadbatch.go`, `renderdata.go`, `rendering.go`, `renderingcache.go`, `squad_renderer.go`, `squadhighlights.go`, `tilerenderer.go`, `viewport.go`

### QuadBatch: The GPU Batching Primitive

`quadbatch.go` is the performance foundation of the rendering package. A `QuadBatch` accumulates an arbitrary number of textured rectangles (quads) that share the same source `*ebiten.Image`, then issues a single `DrawTriangles` call for all of them.

```go
type QuadBatch struct {
    vertices []ebiten.Vertex
    indices  []uint16
    image    *ebiten.Image
}
```

Each call to `Add` appends four vertices and six indices (two triangles forming a quad). `Draw` issues the single GPU call. `Reset` sets both slices to length zero without releasing the underlying memory, making per-frame reuse allocation-free.

The vertex layout is a standard triangle-strip pair. For a quad at `(dstX, dstY)` with destination size `(dstW, dstH)` and source rectangle `(srcX, srcY, srcW, srcH)`:

```
v0 (dstX,       dstY)        → (srcX,       srcY)
v1 (dstX+dstW,  dstY)        → (srcX+srcW,  srcY)
v2 (dstX,       dstY+dstH)   → (srcX,       srcY+srcH)
v3 (dstX+dstW,  dstY+dstH)   → (srcX+srcW,  srcY+srcH)
Triangles: [0,1,2] and [1,3,2]
```

The `r, g, b, a` color parameters modulate the texture color per-vertex. Passing `1.0, 1.0, 1.0, 1.0` renders at full color and opacity.

Pre-allocation constants defined in `rendering.go`:

```go
TileBatchDefaultNumImages = 20   // Expected unique tile images
TileVerticeBatchSize      = 800  // Pre-allocated vertices per tile batch
TileIndicesBatchSize      = 1200 // Pre-allocated indices per tile batch
SpriteVerticesBatchSize   = 256  // Pre-allocated vertices per sprite batch
SpriteIndicesBatchSize    = 384  // Pre-allocated indices per sprite batch
```

### TileRenderer: Map Tile Rendering

`tilerenderer.go` manages the batched rendering of all map tiles.

```go
type TileRenderer struct {
    tiles            []*worldmap.Tile
    batches          map[*ebiten.Image]*QuadBatch
    lastCenterX      int
    lastCenterY      int
    lastViewportSize int
    batchesBuilt     bool
}
```

`Render(opts RenderOptions) RenderedBounds` is the entry point. `RenderOptions` carries:

```go
type RenderOptions struct {
    RevealAll       bool
    CenterOn        *coords.LogicalPosition // nil for full-map mode
    ViewportSize    int
    Screen          *ebiten.Image
    TileColorsDirty bool
}
```

The renderer uses a lazy-rebuild strategy. Batches are rebuilt only when:
- It is the first render call (`batchesBuilt == false`).
- The viewport center or size has changed (detected by comparing against `lastCenterX/Y/Size`).
- `TileColorsDirty` is true (a tile's `ColorMatrix` was modified since the last draw).

After rebuilding, the same batches are re-drawn every frame without re-examining individual tiles. This is correct because tile images do not change per frame; only positions and colors change, and positions only change when the viewport moves.

When viewport mode is active (`CenterOn != nil`), screen positions are computed via `coords.CoordManager.LogicalToScreen(tileLogicalPos, center)` and the tile dimensions are scaled by `graphics.ScreenInfo.ScaleFactor`. In full-map mode, tiles use their stored `PixelX/PixelY` values directly with no scaling.

`RenderedBounds` captures the edges of what was drawn for use by the UI layout system:

```go
type RenderedBounds struct {
    MinX, MaxX int
    MinY, MaxY int
    RightEdgeX int // Rightmost pixel column rendered
    TopEdgeY   int // Topmost pixel row rendered
}
```

The `RightEdgeX` and `TopEdgeY` fields are populated during viewport rendering and stored back on `GameMap` by the game loop to allow UI panels to align themselves to the edge of the rendered map.

### Map Rendering Entry Points

`maprendering.go` provides the public API for map rendering. It manages two cached `TileRenderer` instances (one for full-map, one for viewport) and detects map replacement by comparing tile slice identity:

```go
func DrawMap(screen *ebiten.Image, gameMap *worldmap.GameMap, revealAll bool)

func DrawMapCentered(screen *ebiten.Image, gameMap *worldmap.GameMap,
    centerPos *coords.LogicalPosition, viewportSize int,
    revealAll bool) RenderedBounds
```

`tilesChanged` detects when the tile slice has been replaced by comparing both length and the address of the first element. When a change is detected, `ResetMapRenderers` is called to force the cached renderers to be rebuilt from scratch on the next draw. `ResetMapRenderers` is also exposed publicly for cases where the map changes outside the normal draw path.

Both functions set `gameMap.TileColorsDirty = false` after rendering to acknowledge the dirty flag.

### RenderingCache: Sprite Batching for ECS Entities

`renderingcache.go` provides the infrastructure for batching ECS entity sprites.

```go
type RenderingCache struct {
    RenderablesView *ecs.View
    spriteBatches   map[*ebiten.Image]*QuadBatch
}
```

`RenderablesView` is an ECS library view that automatically tracks all entities tagged with `common.RenderablesTag`. It is maintained by the ECS library without manual synchronization; adding or removing the tag from an entity automatically updates the view.

`spriteBatches` maps each unique `*ebiten.Image` to its own `QuadBatch`. Sprites are grouped by image to minimize draw calls in the same way tiles are.

`GetOrCreateSpriteBatch` lazily creates a batch for an image the first time it is needed. `ClearSpriteBatches` resets all batches at the start of each frame. `DrawSpriteBatches` issues one draw call per unique image.

`RefreshRenderablesView` recreates the view from scratch. This is necessary after batch entity disposal: the ECS library may not immediately remove disposed entities from existing views, so the view must be rebuilt to ensure stale entities do not appear in the next render pass.

### Renderable ECS Component

The `common.Renderable` struct is defined in `common/commoncomponents.go` and used by the rendering package:

```go
type Renderable struct {
    Image   *ebiten.Image
    Visible bool
}
```

An entity must have `common.RenderableComponent` (carrying a `*Renderable`) and `common.PositionComponent` (carrying a `*coords.LogicalPosition`) to be drawn by the sprite rendering pipeline. The `Visible` flag allows hiding entities without removing their component.

### Entity Rendering Entry Points

`rendering.go` provides the public API for entity sprite rendering. Both functions delegate to the internal `processRenderablesCore`:

```go
func ProcessRenderables(gameMap worldmap.GameMap, screen *ebiten.Image, cache *RenderingCache)

func ProcessRenderablesInSquare(gameMap worldmap.GameMap, screen *ebiten.Image,
    playerPos *coords.LogicalPosition, squareSize int, cache *RenderingCache)
```

`ProcessRenderables` renders all entities in full-map mode: no bounds check, sprites drawn at their tile's stored pixel coordinates without scaling.

`ProcessRenderablesInSquare` renders only entities whose logical position falls within a square region of `squareSize` tiles around `playerPos`. Entities outside that region are skipped. Positions within the region are converted via `coords.CoordManager.LogicalToScreen` and scaled by `graphics.ScreenInfo.ScaleFactor`.

The internal `viewportParams` struct carries the filtering parameters and selects which code path to use:

```go
type viewportParams struct {
    centerPos *coords.LogicalPosition
    section   coords.DrawableSection
}
```

When `vp` is nil, full-map mode is active. When non-nil, viewport mode applies.

### ViewportRenderer and Coordinate Utilities

`viewport.go` provides two utilities used by the overlay renderers (movement tiles, health bars, squad highlights).

**`ViewportRenderer`** wraps `coords.CoordManager` and caches expensive GPU objects:

```go
type ViewportRenderer struct {
    centerPos       coords.LogicalPosition
    borderImages    BorderImageCache
    overlayCache    *ebiten.Image
    overlayTileSize int
    borderDrawOpts  [4]ebiten.DrawImageOptions
    overlayDrawOpts ebiten.DrawImageOptions
}
```

Key methods:

- `LogicalToScreen(pos) (float64, float64)` — delegates to `CoordManager.LogicalToScreen`.
- `DrawTileOverlay(screen, pos, fillColor)` — draws a solid-filled tile-sized rectangle at `pos`. Reuses `overlayCache` across frames, recreating it only when tile size changes.
- `DrawTileBorder(screen, pos, borderColor, thickness)` — draws four border rectangles around a tile using `BorderImageCache`. Border images are filled white once and tinted with `ebiten.ColorScale` each call, avoiding per-frame image creation.

**`BorderImageCache`** holds four pre-created border images (top, bottom, left, right) keyed by `tileSize` and `thickness`. It recreates all four only when either dimension changes (e.g., on window resize).

**`CachedViewport`** wraps a `*ViewportRenderer` with smart invalidation. It recreates the renderer only when screen dimensions change and calls `UpdateCenter` when only the center position changes:

```go
func (cv *CachedViewport) GetRenderer(screen *ebiten.Image, centerPos coords.LogicalPosition) *ViewportRenderer
```

Every overlay renderer in the package embeds a `CachedViewport` rather than holding a `*ViewportRenderer` directly, so they all benefit from this caching without duplicating the logic.

### Combat Overlays: Movement and Health Bars

`combatoverlays.go` defines two focused renderers used during tactical combat.

**`MovementTileRenderer`** draws semi-transparent green overlays on all tiles in a supplied list:

```go
func (mtr *MovementTileRenderer) Render(screen *ebiten.Image, centerPos coords.LogicalPosition, validTiles []coords.LogicalPosition)
```

It calls `vr.DrawTileOverlay` for each position. The fill color is fixed at `RGBA{0, 255, 0, 80}` — a 31% opacity green. The caller supplies the list of valid movement destinations computed by the movement system.

**`HealthBarRenderer`** draws a proportional red health bar above every squad on the tactical map:

```go
func (hbr *HealthBarRenderer) Render(screen *ebiten.Image, centerPos coords.LogicalPosition)
```

It queries all squads from its `SquadInfoProvider`, skips destroyed squads and squads without position data, computes `healthRatio = currentHP / maxHP`, and draws a dark-gray background bar with a red fill bar of proportional width. The fill bar uses `SubImage` to clip the pre-created fill image to the correct width, avoiding per-frame image creation.

The bar is positioned 10 pixels above the tile (`yOffset = -10`) and centered horizontally within the tile at 80% of tile width. Background and fill images are created once and reused; they are only recreated when tile size changes.

### Squad Highlight Renderer

`squadhighlights.go` draws colored tile borders around all squads on the tactical map to make faction membership and selection state visually clear.

```go
type SquadHighlightRenderer struct {
    dataProvider    SquadInfoProvider
    selectedColor   color.Color
    factionColors   map[ecs.EntityID]color.Color
    defaultColor    color.Color
    borderThickness int
    viewport        CachedViewport
}
```

Color assignment follows a deterministic palette of eight colors. Each faction is assigned the next color in the palette on first encounter, and subsequent calls reuse the cached assignment. The selected squad is always shown with a white border regardless of faction.

`ResetFactionColors()` must be called when a new combat encounter starts to prevent color index drift from carrying over and assigning unexpected colors to factions in subsequent encounters.

`Render` iterates all squads from `SquadInfoProvider.GetAllSquadIDs()`, resolves the highlight color for each, and calls `vr.DrawTileBorder` with a thickness of 3 pixels.

### SquadCombatRenderer: The Combat Animation Screen

`squad_renderer.go` handles rendering squads on the combat animation screen — the close-up panel that plays when two squads engage in combat. This is distinct from the tactical map and does not use tile coordinates.

```go
type SquadCombatRenderer struct {
    unitProvider UnitInfoProvider
}
```

`RenderSquad(screen, squadID, baseX, baseY, cellSize, facingLeft)` draws all units in a squad on a 3x3 grid. `RenderSquadWithHighlight` adds per-unit color override capability for animation effects (e.g., flashing attacking units). `RenderUnitWithColor` is the per-unit primitive.

The rotation logic in `RenderUnitWithColor` applies a 90-degree transform to make squads face each other across the combat screen:

- Attacker (facing right): `displayCol = 2 - anchorRow`, `displayRow = anchorCol`.
- Defender (facing left): `displayCol = anchorRow`, `displayRow = 2 - anchorCol`.

The horizontal mirror for defenders is applied via `op.GeoM.Scale(-scale, scale)` followed by a translation to compensate for the negative origin. This is pure rendering math; the underlying unit formation data is not modified.

Sprites are scaled uniformly to fit within their cell: `scale = min(scaleX, scaleY)` preserves aspect ratio. The sprite is then centered within its cell by computing offset from the scaled dimensions versus the cell dimensions.

### Render Data Interfaces

`renderdata.go` defines the data-transfer types and provider interfaces used by the overlay renderers. This file is the boundary between the rendering package and the rest of the game.

```go
type SquadRenderInfo struct {
    ID          ecs.EntityID
    Position    *coords.LogicalPosition
    FactionID   ecs.EntityID
    IsDestroyed bool
    CurrentHP   int
    MaxHP       int
}

type UnitRenderInfo struct {
    AnchorRow int
    AnchorCol int
    Width     int
    Height    int
    Image     *ebiten.Image
    IsAlive   bool
}
```

The provider interfaces are satisfied by adapters in `gui/framework/guiqueries_rendering.go`:

```go
type SquadInfoProvider interface {
    GetAllSquadIDs() []ecs.EntityID
    GetSquadRenderInfo(squadID ecs.EntityID) *SquadRenderInfo
}

type UnitInfoProvider interface {
    GetUnitIDsInSquad(squadID ecs.EntityID) []ecs.EntityID
    GetUnitRenderInfo(unitID ecs.EntityID) *UnitRenderInfo
}
```

Using interfaces here is the key decoupling mechanism: the rendering package has no direct dependency on any ECS query functions, squad components, or the entity manager. Callers construct the renderers with an appropriate provider and the renderers remain portable.

---

## Cross-Package Design Patterns

### Strategy Pattern in Effect System

The `VisualEffect` / `Animator` / `Renderer` three-way split is an implementation of the Strategy pattern. `BaseEffect` is the context; `Animator` and `Renderer` are the strategies. Adding a new animation style or drawing technique requires only creating a new struct implementing the relevant interface, with no changes to the lifecycle machinery.

### Prototype via Copy for Area Effects

`VisualEffectArea` uses `vx.Copy()` to stamp multiple instances of a single prototype effect across all tiles in an area. This avoids the caller having to construct N identical effects and manages the fact that each effect needs its own screen-coordinate state (`startX`, `startY`) while sharing all other configuration.

### Lazy Batch Rebuilding

Both `TileRenderer` and `RenderingCache` use a dirty-flag or change-detection approach to avoid rebuilding geometry every frame. The key insight is that most frames are identical to the previous frame from a batch perspective: the map did not change, the entity positions did not change significantly. Rebuild only on detected change.

### Image-Keyed Batches

Both tile and sprite rendering group draw commands by `*ebiten.Image` pointer. Since the same image instance is shared across all tiles of a given type (and across all entities using the same sprite), this naturally produces small numbers of batches even for large maps with many entity types.

### Provider Interface Decoupling

The overlay renderers (`HealthBarRenderer`, `SquadHighlightRenderer`, `SquadCombatRenderer`) accept provider interfaces rather than ECS queries or entity manager references. This means the rendering package has zero dependency on the tactical or squad packages. The concrete implementations live in `gui/framework/guiqueries_rendering.go`, which has access to the ECS manager and knows how to extract the needed data.

---

## Integration with the Game Loop

The main game loop in `game_main/main.go` integrates both packages directly.

In `Update()`:
```go
graphics.VXHandler.UpdateVisualEffects()
```

In `Draw()`:
```go
// Update screen dimensions
graphics.ScreenInfo.ScreenWidth = screen.Bounds().Dx()
graphics.ScreenInfo.ScreenHeight = screen.Bounds().Dy()
coords.CoordManager.UpdateScreenDimensions(...)

// Render map (one of two modes)
if coords.MAP_SCROLLING_ENABLED {
    bounds := rendering.DrawMapCentered(screen, &g.gameMap, g.playerData.Pos,
        config.DefaultZoomNumberOfSquare, config.DEBUG_MODE)
    g.gameMap.RightEdgeX = bounds.RightEdgeX
    g.gameMap.TopEdgeY = bounds.TopEdgeY
    rendering.ProcessRenderablesInSquare(g.gameMap, screen, g.playerData.Pos,
        config.DefaultZoomNumberOfSquare, g.renderingCache)
} else {
    rendering.DrawMap(screen, &g.gameMap, config.DEBUG_MODE)
    rendering.ProcessRenderables(g.gameMap, screen, g.renderingCache)
}

// Draw visual effects on top of map
graphics.VXHandler.DrawVisualEffects(screen)

// EbitenUI renders on top of everything else
g.gameModeCoordinator.Render(screen)
```

The rendering order establishes the layer stack:
1. Map tiles (via `TileRenderer` batches)
2. ECS entity sprites (via `RenderingCache` batches)
3. Visual effects (via `VisualEffectHandler`)
4. EbitenUI overlay (modal panels, HUD)

The overlay renderers (`MovementTileRenderer`, `HealthBarRenderer`, `SquadHighlightRenderer`) are called from within EbitenUI context rendering inside `g.gameModeCoordinator.Render(screen)`, which places them above the map but below any modal dialogs.

---

## Integration with the World Layer

The `world/worldmap` package integrates with `visual/graphics` through the `ColorMatrix` system.

`worldmap.Tile` stores a `graphics.ColorMatrix` as a private field with `SetColorMatrix` and `GetColorMatrix` accessors. `worldmap.GameMap` exposes two methods for batch application:

```go
func (gameMap *GameMap) ApplyColorMatrix(indices []int, m graphics.ColorMatrix)
func (gameMap *GameMap) ApplyColorMatrixToIndex(index int, m graphics.ColorMatrix)
```

Both methods set `gameMap.TileColorsDirty = true` after modifying tiles. The `TileRenderer` checks this flag on each `Render` call and forces a batch rebuild when it is true. The flag is cleared by `DrawMap` and `DrawMapCentered` after rendering.

This mechanism is used for:
- **Spell targeting overlays** — the AoE shape handler applies a magenta `ColorMatrix` to tiles under the cursor, then clears them with `NewEmptyMatrix()` when the cursor moves.
- **Tactical analysis overlays** — the threat visualizer applies named gradient colors (from `colormatrix.go`) to tiles to show movement ranges, threat zones, and expected damage.

---

## Integration with GUI Systems

The GUI combat layer (`gui/guicombat`) constructs and owns the overlay renderers:

```go
// In combatvisualization.go
movementRenderer:  rendering.NewMovementTileRenderer()
highlightRenderer: rendering.NewSquadHighlightRenderer(queries)
healthBarRenderer: rendering.NewHealthBarRenderer(queries)
```

The `queries` parameter satisfies `SquadInfoProvider`. The concrete type is `*framework.GUIQueries`, which wraps the entity manager and translates ECS queries into the `SquadRenderInfo` / `UnitRenderInfo` data transfer objects.

The `SquadCombatRenderer` is constructed inside the combat animation mode:

```go
// In combat_animation_panels_registry.go
cam.squadRenderer = rendering.NewSquadCombatRenderer(cam.Queries)
```

The spell casting handler in `gui/guispells/spell_handler.go` uses the `graphics` package directly for two purposes: converting mouse position to logical tile coordinates via `graphics.MouseToLogicalPosition`, and manipulating `graphics.TileBasedShape` instances for AoE targeting. It calls `GameMap.ApplyColorMatrix` to paint the targeting overlay onto tiles.

---

## Data Flow Diagram

```
                          game_main/main.go
                          |
               Update()   |    Draw()
               |          |    |
               v          |    v
   graphics.VXHandler.    |   [Tactical context only]
   UpdateVisualEffects()  |    |
                          |    +---> rendering.DrawMapCentered / DrawMap
                          |    |         |
                          |    |         v
                          |    |    TileRenderer.Render()
                          |    |         |
                          |    |         v
                          |    |    QuadBatch.Add() (per tile)
                          |    |    QuadBatch.Draw() (per unique image)
                          |    |
                          |    +---> rendering.ProcessRenderablesInSquare / ProcessRenderables
                          |    |         |
                          |    |         v
                          |    |    RenderingCache.RenderablesView.Get()
                          |    |    QuadBatch.Add() (per entity)
                          |    |    QuadBatch.Draw() (per unique image)
                          |    |
                          |    +---> graphics.VXHandler.DrawVisualEffects()
                          |    |         |
                          |    |         v
                          |    |    BaseEffect.DrawVisualEffect()
                          |    |         |
                          |    |    Animator.Update() -> AnimationState
                          |    |    Renderer.Draw() -> ebiten draw calls
                          |    |
                          |    +---> gameModeCoordinator.Render()
                          |              |
                          |              +---> CombatVisualization
                          |              |     MovementTileRenderer.Render()
                          |              |     SquadHighlightRenderer.Render()
                          |              |     HealthBarRenderer.Render()
                          |              |         |
                          |              |         v
                          |              |     ViewportRenderer.DrawTileOverlay/Border()
                          |              |         |
                          |              |         v
                          |              |     QuadBatch equivalent (per-tile draws)
                          |              |
                          |              +---> CombatAnimationMode
                          |                    SquadCombatRenderer.RenderSquad()
                          |                        |
                          |                        v
                          |                    ebiten.DrawImage() per unit
```

---

## Performance Considerations

**Tile rendering** is the most significant. A full dungeon map has `dungeonWidth * dungeonHeight` tiles. Without batching, this is one draw call per tile. With `TileRenderer`, the batches are built once when the viewport changes and re-submitted every frame. A typical dungeon with 5-10 unique floor/wall images produces 5-10 draw calls regardless of map size.

**Sprite rendering** follows the same pattern. The number of draw calls equals the number of unique `*ebiten.Image` values among all `Renderable` entities, not the number of entities. Entities sharing the same sprite image (e.g., multiple goblins) are batched into one call.

**Tile color dirty flag** prevents quadratic behavior. Without the dirty flag, the tile renderer would need to inspect every tile's color matrix every frame. With it, batch rebuilds triggered by `ApplyColorMatrix` cost only one extra rebuild per frame, and subsequent frames are free.

**Image caching in overlay renderers** avoids GPU texture allocation in the draw loop. `BorderImageCache`, `overlayCache`, `bgImage`, and `fillImage` are all reused across frames and only recreated on dimension changes.

**`ProceduralRenderer`** (sticky ground effect) is the notable exception: it creates new `ebiten.Image` instances every frame for each circle it draws. This is intentional for a rarely-used prototype effect but would be a problem if used at scale.

**`VisualEffectHandler.clearVisualEffects`** runs every `Update` call and allocates new slices to hold survivors. For small numbers of effects (typical: 0-10 simultaneously), the allocation cost is negligible. The handler does not use index-swap-delete for simplicity.

---

## Common Extension Points

**Adding a new visual effect** — implement a new named constructor in `vxfactory.go` using `NewEffect` with the appropriate `EffectConfig`. Add a new animator in `animators.go` or a new renderer in `renderers.go` only if existing ones cannot be configured to achieve the desired behavior. Add the new type string to `CreateVisualEffectByType` if it should be accessible from spell templates.

**Adding a new area effect shape** — add a new factory function (following `NewCircle`, `NewSquare`, etc.) that returns a `*BaseShape` with the appropriate `BasicShapeType` and dimension fields. Add a new case to `CreateShapeFromConfig` if the shape should be configurable from JSON templates.

**Adding a new tactical overlay** — create a new renderer struct in `combatoverlays.go` that embeds `CachedViewport` and implements a `Render` method. Use `vr.DrawTileOverlay` or `vr.DrawTileBorder` from `ViewportRenderer`. Construct and call it from the appropriate GUI combat type.

**Adding a new tile color semantic** — add a new `Create*Gradient` function in `colormatrix.go` following the existing naming convention. Apply it via `GameMap.ApplyColorMatrix` from the appropriate analysis system.

**Changing the rendering mode** — `coords.MAP_SCROLLING_ENABLED` in `world/coords` switches between viewport-centered and full-map rendering modes throughout both packages without any code changes in `visual/`. The coordinate manager handles the transformation differences transparently.
