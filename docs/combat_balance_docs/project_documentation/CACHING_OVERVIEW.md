# TinkerRogue Caching Mechanisms

**Last Updated:** 2026-02-14

This document provides a comprehensive overview of all caching mechanisms used throughout the TinkerRogue codebase. Caching is critical for performance in this ECS-based roguelike, reducing O(n) queries to O(1) or O(k) lookups where k is the number of matching entities.

---

## Table of Contents

1. [ECS View-Based Caches](#ecs-view-based-caches)
2. [Event-Driven Caches](#event-driven-caches)
3. [Spatial Grid Caches](#spatial-grid-caches)
4. [Rendering Caches](#rendering-caches)
5. [AI Threat Evaluation Caches](#ai-threat-evaluation-caches)
6. [Widget Render Caches](#widget-render-caches)
7. [Cache Relationships](#cache-relationships)
8. [Performance Characteristics](#performance-characteristics)
9. [Hook-Based Cache Invalidation System](#hook-based-cache-invalidation-system)

---

## 1. ECS View-Based Caches

ECS Views are automatically-maintained caches provided by the bytearena/ecs library. When components are added or removed from entities, the library automatically updates all relevant Views. This provides O(k) iteration where k = entities with the component, versus O(n) for World.Query() where n = all entities.

### 1.1 SquadQueryCache

**File:** `tactical/squads/squadcache.go`

**Purpose:** Provides cached access to squad-related queries for GUI hot paths.

**Structure:**
```go
type SquadQueryCache struct {
    SquadView       *ecs.View // All SquadTag entities
    SquadMemberView *ecs.View // All SquadMemberTag entities
    LeaderView      *ecs.View // All LeaderTag entities
}
```

**Initialization:**
```go
// Created during GUI initialization
cache := squads.NewSquadQueryCache(manager)
// Views are created with:
manager.World.CreateView(SquadTag)
```

**Usage Pattern:**
- GUI code uses `cache.GetSquadEntity(squadID)` instead of `squads.GetSquadEntity(squadID, manager)`
- Iteration over `cache.SquadView.Get()` instead of `manager.World.Query(SquadTag)`

**Invalidation:** Automatic - ECS library updates Views when components change

**Performance:** 100-500x faster than full World.Query() for large entity counts

**Methods:**
- `GetSquadEntity(squadID)` - Find squad by ID
- `GetUnitIDsInSquad(squadID)` - Get all units in a squad
- `GetLeaderID(squadID)` - Find squad leader
- `GetSquadName(squadID)` - Get squad name
- `FindAllSquads()` - List all squad IDs

**Reference Implementation:** This is the canonical example of proper ECS View usage in TinkerRogue.

---

### 1.2 CombatQueryCache

**File:** `tactical/combat/combatqueriescache.go`

**Purpose:** Cached access to combat-related queries (action states, factions).

**Structure:**
```go
type CombatQueryCache struct {
    ActionStateView *ecs.View // All ActionStateTag entities
    FactionView     *ecs.View // All FactionTag entities
}
```

**Initialization:**
```go
cache := combat.NewCombatQueryCache(manager)
```

**Usage:**
- `FindActionStateBySquadID(squadID)` - Get squad's combat action state
- `FindFactionByID(factionID)` - Get faction entity
- `FindFactionDataByID(factionID)` - Get faction data component

**Invalidation:** Automatic via ECS library

**Performance Impact:** Critical for turn-based combat where action states are queried frequently during AI decision-making.

---

### 1.3 RenderingCache

**File:** `visual/rendering/renderingcache.go`

**Purpose:** Cached access to renderable entities for the rendering pipeline.

**Structure:**
```go
type RenderingCache struct {
    RenderablesView *ecs.View                  // All RenderablesTag entities
    spriteBatches   map[*ebiten.Image]*QuadBatch // Sprite batching by image
}
```

**Initialization:**
```go
cache := rendering.NewRenderingCache(manager)
```

**View Usage:** Iterate `cache.RenderablesView.Get()` to find all entities with sprites

**Sprite Batch Cache:**
- Groups sprites by image for batch rendering
- Pre-allocates 20 batch slots (typical games have 5-20 unique sprite images)
- `GetOrCreateSpriteBatch(image)` - Lazy creation of batches
- `ClearSpriteBatches()` - Reset all batches for next frame
- `DrawSpriteBatches(screen)` - Render all collected sprites

**Invalidation:**
- View: Automatic via ECS
- Sprite batches: Manual clear each frame via `ClearSpriteBatches()`
- Special case: `RefreshRenderablesView()` must be called after batch entity disposal to prevent stale entity rendering

**Performance:** Batch rendering reduces draw calls by 10-50x

---

### 1.4 Package-Level ECS Views

Several packages maintain package-level Views initialized during subsystem registration:

#### Overworld Core Views
**File:** `overworld/core/init.go`

```go
var OverworldNodeView *ecs.View
var OverworldFactionView *ecs.View

// Initialized in init():
OverworldNodeView = em.World.CreateView(OverworldNodeTag)
OverworldFactionView = em.World.CreateView(OverworldFactionTag)
```

**Usage:** `overworld/influence/queries.go` uses `core.OverworldNodeView.Get()` for O(k) node queries

---

#### Squad Member View
**File:** `tactical/squads/squadmanager.go`

```go
var squadMemberView *ecs.View

// Initialized in init():
squadMemberView = em.World.CreateView(SquadMemberTag)
```

**Usage:** Zero-allocation squad member queries within the squads package

---

#### Combat Views
**File:** `tactical/combat/combatcomponents.go`

```go
var factionView *ecs.View
var combatSquadView *ecs.View

// Initialized in init():
factionView = em.World.CreateView(FactionTag)
combatSquadView = em.World.CreateView(squads.SquadTag)
```

**Usage:** Internal combat system queries

---

#### Commander View
**File:** `tactical/commander/components.go`, `tactical/commander/init.go`

```go
var CommanderView *ecs.View

// Initialized in init():
CommanderView = em.World.CreateView(CommanderTag)
```

**Usage:** Zero-allocation commander queries for the overworld commander system
**Purpose:** Supports the commander system where each commander controls their own squad roster

---

### 1.5 View Best Practices

**When to Use Views:**
- GUI hot paths (queries every frame)
- System update loops (iteration over specific entity types)
- Scenarios where the same query runs multiple times

**When NOT to Use Views:**
- One-off queries in game logic
- Tests (use canonical query functions from `*queries.go`)
- When you don't own the EntityManager lifecycle

**Memory Trade-off:** Each View maintains a list of matching entities, but this is O(k) space where k = matching entities, which is acceptable for most cases.

**Critical Note:** Views are automatically maintained by the ECS library. You never need to manually update them - they reflect component changes immediately.

---

## 2. Event-Driven Caches

Event-driven caches are optimal for turn-based games where data changes are discrete and infrequent. Unlike frame-level caches that expire every frame, event-driven caches remain valid until explicitly invalidated by game events.

### 2.1 SquadInfoCache

**File:** `gui/framework/squadinfo_cache.go`

**Purpose:** Caches computed SquadInfo aggregates (HP totals, alive units, action states) to avoid recomputation every frame.

**Structure:**
```go
type SquadInfoCache struct {
    cache       map[ecs.EntityID]*SquadInfo  // Cached computed results
    dirtySquads map[ecs.EntityID]bool        // Squads needing recomputation
    queries     *GUIQueries
}
```

**Cache Strategy:** Lazy evaluation with dirty flag tracking
- Data cached indefinitely until marked dirty
- Recomputation only occurs when cache is missing or dirty
- O(1) lookups for cached data

**Invalidation Methods:**
```go
MarkSquadDirty(squadID)    // Invalidate single squad
MarkAllDirty()             // Invalidate all squads (turn start/end)
InvalidateSquad(squadID)   // Complete removal (squad destroyed)
```

**When to Invalidate:**
- Squad takes damage
- Squad moves
- Squad uses action
- Unit dies
- Turn starts/ends
- Combat begins/ends

**Performance:** Reduces expensive aggregate calculations from every frame (60 FPS) to only when game state changes (turn-based events).

**Usage Example:**
```go
// GUI code - O(1) for cached data
info := guiQueries.GetSquadInfo(squadID)

// After squad takes damage
guiQueries.MarkSquadDirty(squadID)

// After turn ends
guiQueries.MarkAllSquadsDirty()
```

**Cached Data:**
- Squad name, ID
- Unit IDs and count
- Alive units vs total units
- Current HP / Max HP totals
- Position
- Faction membership
- Combat action state (HasActed, HasMoved, MovementRemaining)

**Dependencies:** Uses `SquadQueryCache` and `CombatQueryCache` for data collection during rebuild.

---

### 2.2 GUIQueries Cache Coordination

**File:** `gui/framework/guiqueries.go`

**Purpose:** Centralized query service that coordinates multiple caches for UI.

**Structure:**
```go
type GUIQueries struct {
    ECSManager     *common.EntityManager
    factionManager *combat.CombatFactionManager

    SquadCache     *squads.SquadQueryCache      // ECS View cache
    CombatCache    *combat.CombatQueryCache     // ECS View cache
    squadInfoCache *SquadInfoCache              // Event-driven cache
}
```

**Integration Pattern:**
- Owns multiple cache types (View-based + event-driven)
- Exposes unified query interface to GUI code
- Provides cache invalidation methods that propagate to all relevant caches

**Benefits:**
- Single injection point for GUI dependencies
- Consistent cache management
- Eliminates query duplication across UI modes

---

## 3. Spatial Grid Caches

Spatial grids provide O(1) position-based entity lookup, replacing O(n) linear searches through all entities.

### 3.1 GlobalPositionSystem

**File:** `common/positionsystem.go`

**Purpose:** O(1) spatial queries for entity positions using a hash map grid.

**Structure:**
```go
type PositionSystem struct {
    manager     *ecs.Manager
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID // Value keys for O(1) hash
}
```

**Cache Strategy:** Manual maintenance - game code must add/remove entities
- Uses value-type keys (not pointers) for 50x faster hash lookups
- Multiple entities can occupy same position (slice per position)

**API Methods:**
```go
// Queries
GetEntityIDAt(pos)           // O(1) - First entity at position
GetEntityAt(pos)             // O(1) - First entity (as *ecs.Entity)
GetAllEntityIDsAt(pos)       // O(1) - All entities at position
GetEntitiesInRadius(center, radius) // O(r²) where r=radius

// Maintenance
AddEntity(entityID, pos)     // Register entity
RemoveEntity(entityID, pos)  // Unregister entity
MoveEntity(entityID, old, new) // Atomic move (remove + add)

// Utility
GetEntityCount()             // Total tracked entities
GetOccupiedPositions()       // All non-empty positions
Clear()                      // Reset grid
```

**Invalidation:** Manual
- Call `AddEntity()` when entity spawns
- Call `RemoveEntity()` when entity despawns
- Call `MoveEntity()` when entity moves
- `EntityManager.MoveEntity()` and `MoveSquadAndMembers()` handle this automatically

**Performance:** 50x faster than linear search with 50+ entities

**Global Access:** `common.GlobalPositionSystem` is a global singleton initialized during game setup

**Critical Pattern:** Always use `EntityManager.MoveEntity()` or `MoveSquadAndMembers()` which atomically update both the PositionComponent and the PositionSystem.

**Value Keys vs Pointer Keys:**
```go
// ✅ CORRECT (50x faster)
spatialGrid map[coords.LogicalPosition][]ecs.EntityID

// ❌ WRONG (slow hash, GC pressure)
spatialGrid map[*coords.LogicalPosition][]ecs.EntityID
```

---

## 4. Rendering Caches

Rendering caches reduce CPU usage by avoiding redundant rendering operations.

### 4.1 TileRenderer Batch Cache

**File:** `visual/rendering/tilerenderer.go`

**Purpose:** Cache tile batches when viewport hasn't moved to avoid rebuilding sprite batches every frame.

**Structure:**
```go
type TileRenderer struct {
    tiles   []*worldmap.Tile
    batches map[*ebiten.Image]*QuadBatch

    // Cache state
    lastCenterX      int
    lastCenterY      int
    lastViewportSize int
    batchesBuilt     bool
}
```

**Cache Strategy:** Invalidate-on-change
- Batches rebuilt only when viewport position or size changes
- Full map rendering builds batches once and reuses indefinitely
- Batches grouped by image for efficient DrawTriangles() calls

**Invalidation:**
```go
// Automatic invalidation when:
if r.lastCenterX != opts.CenterOn.X ||
   r.lastCenterY != opts.CenterOn.Y ||
   r.lastViewportSize != opts.ViewportSize {
    needsRebuild = true
}
```

**Performance:** Eliminates 90%+ of batch rebuilds during static camera scenarios (inventory, menus, stationary player)

**Batch Management:**
```go
// Reuse batches to avoid allocations
for _, batch := range r.batches {
    batch.Reset() // Clear vertices but keep allocation
}

// Add tile to appropriate batch
if r.batches[tile.Image] == nil {
    r.batches[tile.Image] = NewQuadBatch(tile.Image, ...)
}
batch := r.batches[tile.Image]
batch.AddQuad(...) // Accumulate vertices
```

---

### 4.2 CachedBackground (NineSlice Cache)

**File:** `gui/widgetresources/cachedbackground.go`

**Purpose:** Pre-render NineSlice backgrounds to reduce UI rendering overhead.

**Structure:**
```go
type CachedBackground struct {
    source      *image.NineSlice
    cachedImage *ebiten.Image
    dirty       bool
    width       int
    height      int
}
```

**Cache Strategy:** Render-once until size changes or marked dirty

**Invalidation:**
```go
needsRender := cachedImage == nil ||
               width != w ||
               height != h ||
               dirty
```

**Performance Impact:** Reduces NineSlice.drawTile() allocations by ~70% for static panels

**Pool Pattern:**
```go
type CachedBackgroundPool struct {
    source *image.NineSlice
    cache  map[cacheKey]*CachedBackground
}
```

Caches backgrounds at multiple sizes (e.g., different button sizes) for reuse.

**Global Pools:**
- `panelBackgroundPool`
- `buttonBackgroundPool`
- `scrollContainerIdlePool`
- `scrollContainerDisabledPool`
- `scrollContainerMaskPool`

**Pre-Caching:** `PreCacheScrollContainerBackgrounds()` renders common sizes at startup to avoid first-frame hitches

**Common Sizes (1920x1080):**
- 672x756 - Unit purchase list
- 576x756 - Squad deployment list
- 384x540 - Squad editor unit list
- 480x648 - Formation editor list

---

### 4.3 BorderImageCache

**File:** `visual/rendering/viewport.go`

**Purpose:** Cache border images for tile highlighting to avoid GPU allocations in the render loop.

**Structure:**
```go
type BorderImageCache struct {
    top, bottom, left, right *ebiten.Image
    tileSize, thickness      int
}
```

**Cache Strategy:** Create-once per size configuration
- Border images (top, bottom, left, right) are created and filled with white
- Images are recreated only when tile size or border thickness changes (e.g., window resize)
- Tinted using ColorScale at draw time for different border colors

**Invalidation:**
```go
// Automatic invalidation when dimensions change:
if cache.top == nil || cache.tileSize != tileSize || cache.thickness != thickness {
    // Recreate border images
}
```

**Performance Impact:** Eliminates per-frame image allocations for tile borders, reducing GC pressure

**Usage:**
```go
top, bottom, left, right := cache.GetOrCreate(tileSize, thickness)
// Draw with ColorScale for different colors
```

---

### 4.4 ViewportRenderer Overlay Cache

**File:** `visual/rendering/viewport.go`

**Purpose:** Reusable image and DrawImageOptions to avoid allocations when drawing tile overlays and borders.

**Structure:**
```go
type ViewportRenderer struct {
    centerPos       coords.LogicalPosition
    borderImages    BorderImageCache
    overlayCache    *ebiten.Image              // Reusable overlay image
    overlayTileSize int                        // Track size for invalidation
    borderDrawOpts  [4]ebiten.DrawImageOptions // Reusable for borders [top, bottom, left, right]
    overlayDrawOpts ebiten.DrawImageOptions    // Reusable for overlays
}
```

**Cache Strategy:** Object pooling pattern
- `overlayCache` - Single reusable image for tile overlays, recreated only when tile size changes
- `borderDrawOpts` and `overlayDrawOpts` - Reused DrawImageOptions to avoid allocations

**Invalidation:**
```go
// Overlay cache invalidated when tile size changes:
if vr.overlayCache == nil || vr.overlayTileSize != tileSize {
    vr.overlayCache = ebiten.NewImage(tileSize, tileSize)
    vr.overlayTileSize = tileSize
}
```

**Performance Impact:**
- Reduces per-frame allocations for DrawImageOptions
- Avoids creating new overlay images every draw call
- Critical for performance when highlighting many tiles (movement range, ability targets, etc.)

**Usage:**
```go
// DrawImageOptions are reset and reused:
vr.overlayDrawOpts.GeoM.Reset()
vr.overlayDrawOpts.GeoM.Translate(screenX, screenY)
screen.DrawImage(vr.overlayCache, &vr.overlayDrawOpts)
```

---

## 5. AI Threat Evaluation Caches

AI threat layers use round-based caching to avoid recomputing expensive threat maps every decision.

### 5.1 DirtyCache (Generic Cache Invalidation)

**File:** `mind/evaluation/cache.go`

**Purpose:** Generic dirty flag system for round-based lazy evaluation.

**Structure:**
```go
type DirtyCache struct {
    lastUpdateRound int
    isDirty         bool
    isInitialized   bool
}
```

**Usage Pattern:**
```go
// Check validity
if cache.IsValid(currentRound) {
    return cachedValue
}

// Recompute
value = computeExpensiveValue()
cache.MarkClean(currentRound)

// Invalidate on game event
cache.MarkDirty()
```

**Embedded in Threat Layers:**
- `ThreatLayerBase` embeds `DirtyCache`
- All threat layers (Combat, Support, Positional) inherit dirty tracking
- Round-based invalidation ensures cache validity per turn

---

### 5.2 CompositeThreatEvaluator

**File:** `mind/behavior/threat_composite.go`

**Purpose:** Combines multiple threat layers with coordinated cache invalidation.

**Structure:**
```go
type CompositeThreatEvaluator struct {
    manager   *common.EntityManager
    cache     *combat.CombatQueryCache
    factionID ecs.EntityID

    // Individual layers (each has embedded DirtyCache)
    combatThreat   *CombatThreatLayer
    supportValue   *SupportValueLayer
    positionalRisk *PositionalRiskLayer

    // Composite cache state
    lastUpdateRound int
    isDirty         bool
}
```

**Cache Strategy:** Coordinated round-based updates
- All layers update together when round changes
- Layers can be individually marked dirty on game events
- Skip recomputation if already up-to-date for current round

**Update Pattern:**
```go
func (cte *CompositeThreatEvaluator) Update(currentRound int) {
    // Skip if already up-to-date
    if !cte.isDirty && cte.lastUpdateRound == currentRound {
        return
    }

    // Compute layers in dependency order
    cte.combatThreat.Compute()      // First (provides base data)
    cte.supportValue.Compute()       // Depends on combat
    cte.positionalRisk.Compute()     // Depends on combat

    // Mark clean
    cte.lastUpdateRound = currentRound
    cte.isDirty = false
}
```

**Invalidation:**
```go
MarkDirty() // Propagates to all layers
```

**When to Invalidate:**
- Squad moves
- Squad destroyed
- Combat state changes
- Unit dies

**Performance:** Avoids recomputing threat maps for every AI action evaluation (hundreds of positions per turn).

---

### 5.3 ThreatLayerBase

**File:** `mind/behavior/threat_layers.go`

**Purpose:** Base class for all threat layers providing common cache functionality.

**Structure:**
```go
type ThreatLayerBase struct {
    *evaluation.DirtyCache  // Embedded cache
    manager                *common.EntityManager
    cache                  *combat.CombatQueryCache
    factionID              ecs.EntityID
}
```

**Pattern:** Composition-based inheritance
- Concrete layers embed `ThreatLayerBase`
- Common cache management inherited
- Each layer implements its own `Compute()` method that calls `markClean(currentRound)` after computation

**Concrete Layers:**
- `CombatThreatLayer` - Melee and ranged threat
- `SupportValueLayer` - Healing/support opportunities
- `PositionalRiskLayer` - Flanking and exposure risk

---

## 6. Widget Render Caches

GUI widgets can cache their rendered output to avoid redundant draws.

### 6.1 CachedListWrapper

**File:** `gui/widgets/cached_list.go`

**Purpose:** Cache rendered List widget output to avoid redrawing static lists every frame.

**Structure:**
```go
type CachedListWrapper struct {
    list         *widget.List
    cachedImage  *ebiten.Image
    dirty        bool
    lastWidth    int
    lastHeight   int
    renderCount  int  // For profiling
}
```

**Cache Strategy:** Render-on-change
- Only re-renders when marked dirty or size changes
- Trades memory (cached image) for CPU (skip rendering)

**Invalidation:**
```go
// Manual invalidation - caller must mark dirty when:
MarkDirty()  // Content changes, entries added/removed, selection changes
```

**Performance:** Reduces CPU usage by ~90% for static lists

**Usage Pattern:**
```go
list := builders.CreateListWithConfig(...)
cachedList := NewCachedListWrapper(list)

// Whenever list entries change:
cachedList.MarkDirty()
```

**Critical Requirement:** Caller MUST call `MarkDirty()` when:
- List entries added/removed
- Entry content changes
- Selection changes (if showing selection)

---

### 6.2 CachedTextAreaWrapper

**File:** `gui/widgets/cached_textarea.go`

**Purpose:** Cache rendered TextArea output for read-only text displays.

**Structure:**
```go
type CachedTextAreaWrapper struct {
    textarea     *widget.TextArea
    cachedImage  *ebiten.Image
    dirty        bool
    lastWidth    int
    lastHeight   int
    renderCount  int
}
```

**Same Pattern as CachedListWrapper:**
- Render-on-change with dirty flag
- Manual `MarkDirty()` required
- ~90% CPU reduction for static text

**Convenience Methods:**
```go
SetText(text)      // Sets text AND marks dirty
AppendText(text)   // Appends text AND marks dirty
```

**When NOT to Use:**
- Frequently updating text (combat logs that update every action)
- User-editable text areas (input fields)
- Text that scrolls automatically

**Ideal for:**
- Static help text
- Read-only information panels
- Mission briefings
- Character descriptions

---

## 7. Cache Relationships

Understanding how caches depend on each other is critical for proper invalidation.

### Dependency Graph

```
Global Singleton:
  GlobalPositionSystem (spatial grid)
    ↓
    Used by: EntityManager.MoveEntity(), squad movement, combat positioning

ECS Views (auto-maintained):
  SquadQueryCache
  CombatQueryCache
  RenderingCache.RenderablesView
  Package-level Views (OverworldNodeView, CommanderView, etc.)
    ↓
    Used by: GUIQueries, threat evaluators, rendering pipeline

Event-Driven:
  SquadInfoCache
    ↓ Depends on
    SquadQueryCache + CombatQueryCache
    ↓ Invalidated by
    Hook-based invalidation system (OnAttackComplete, OnMoveComplete, OnTurnEnd)

Hook-Based Invalidation:
  CombatActionSystem → fires OnAttackComplete hook
  CombatMovementSystem → fires OnMoveComplete hook
  TurnManager → fires OnTurnEnd hook
    ↓ Registered at
    CombatService (forwards to GUI callbacks)
    ↓ GUI callbacks invoke
    SquadInfoCache.MarkSquadDirty() / InvalidateSquad() / MarkAllSquadsDirty()
    ↓ Also triggers
    Threat layer updates (UpdateThreatManagers, UpdateThreatEvaluator)

Threat Evaluation:
  CompositeThreatEvaluator
    ↓ Contains
    CombatThreatLayer, SupportValueLayer, PositionalRiskLayer
    ↓ Each embeds
    DirtyCache (round-based invalidation)
    ↓ Depends on
    CombatQueryCache (for squad positions, action states)
    ↓ Invalidated by
    OnTurnEnd hook (round-based updates)

Rendering:
  TileRenderer (viewport position cache)
  RenderingCache.spriteBatches (image-based batching)
  CachedBackground (NineSlice pre-rendering)
  BorderImageCache (viewport border images)
  ViewportRenderer (overlay cache, reusable DrawImageOptions)
  CachedListWrapper / CachedTextAreaWrapper (widget output)
```

---

### Invalidation Cascade Example

**Scenario:** Player moves a squad during combat

```
1. User initiates move command
   ↓
2. MoveSquadCommand.Execute()
   ↓
3. CombatMovementSystem.MoveSquad()
   ↓
4. EntityManager.MoveSquadAndMembers(squadID, oldPos, newPos)
   ↓
5. Updates PositionComponent on all squad members
   ↓
6. GlobalPositionSystem.MoveEntity() for each member (spatial grid update)
   ↓
7. ECS library auto-updates all Views (SquadQueryCache, CombatQueryCache, RenderingCache)
   ↓
8. CombatMovementSystem fires OnMoveComplete hook
   ↓
9. CombatService forwards hook to registered GUI callbacks
   ↓
10. GUI callback: cm.Queries.MarkSquadDirty(squadID)
   ↓
11. SquadInfoCache marks squad's entry dirty
   ↓
12. Next UI update: SquadInfoCache recomputes squad info from ECS
```

**Key Insight:** Hook-based invalidation eliminates manual cache invalidation calls in combat logic. Caches update automatically via registered callbacks.

---

## 8. Performance Characteristics

### Cache Performance Summary

| Cache Type | Lookup Speed | Update Cost | Memory Overhead | Best For |
|------------|--------------|-------------|-----------------|----------|
| ECS View | O(k) iteration | O(1) automatic | O(k) space | Frequent queries of specific entity types |
| PositionSystem | O(1) lookup | O(1) manual | O(occupied tiles) | Position-based queries |
| SquadInfoCache | O(1) cached | O(units in squad) | O(cached squads) | Turn-based UI updates |
| ThreatLayers | O(1) per position | O(map size × units) | O(map size) | AI decision-making |
| TileRenderer | 0 (skipped) | O(visible tiles) | O(unique images) | Static camera scenarios |
| CachedBackground | 0 (skipped) | O(width × height) | O(num sizes) | Static UI panels |
| BorderImageCache | 0 (skipped) | O(1) | O(4 images) | Tile border rendering |
| ViewportRenderer | 0 (reused) | 0 | O(1 image + opts) | Tile overlay/border drawing |
| Widget Caches | 0 (skipped) | O(widget complexity) | O(widget size) | Static UI content |

**k** = entities with matching component
**n** = total entities in world

---

### Measured Performance Improvements

**ECS Views:**
- 100-500x faster than World.Query() for large entity counts
- Squad queries: ~0.01ms (View) vs ~5ms (Query) with 500 entities

**PositionSystem:**
- 50x faster than linear search with 50+ entities
- Essential for pathfinding, FOV, and collision detection

**SquadInfoCache:**
- Eliminates 60 FPS × (squad count) unnecessary recalculations
- Example: 10 squads = 600 saved computations per second in static UI

**TileRenderer:**
- 90%+ batch rebuild elimination during static camera
- Reduces frame time by 2-5ms in dungeon exploration

**CachedBackground:**
- 70% reduction in NineSlice allocations
- Measured impact: ~1ms saved per large ScrollContainer per frame

**BorderImageCache:**
- Eliminates per-frame GPU image allocations for tile borders
- Critical for reducing GC pressure when highlighting many tiles

**ViewportRenderer:**
- Eliminates DrawImageOptions allocations (previously ~100+ per frame for movement highlighting)
- Reduces per-frame allocation overhead by reusing single overlay image

**Widget Caches:**
- 90% CPU reduction for static lists/textareas
- Combat log with 100 entries: ~3ms (uncached) vs ~0.3ms (cached)

---

### Memory Trade-offs

**ECS Views:** Minimal
- O(k) entity references per View
- Views share memory with World.Query() results

**PositionSystem:** Moderate
- O(occupied tiles × entities per tile)
- Typical dungeon: ~1000 tiles × 1-2 entities = 2KB

**SquadInfoCache:** Low
- O(cached squads) × sizeof(SquadInfo)
- 10 squads × 200 bytes = 2KB

**ThreatLayers:** High
- O(map tiles) × 4 floats per layer
- 100×100 map × 3 layers × 4 bytes = 120KB per faction

**Rendering Caches:** Moderate to High
- Cached images: width × height × 4 bytes per pixel
- BorderImageCache: 4 small images per tile size configuration (~1KB total)
- ViewportRenderer: 1 overlay image per tile size (~256KB for 256x256 tile)
- TileRenderer batches: Pre-allocated vertex/index buffers per image
- Widget caches: Can accumulate significant memory with many large widgets
- 1920×1080 full-screen buffer = 8MB

**Recommendation:** Monitor texture memory usage if caching many large widgets.

---

## 9. Hook-Based Cache Invalidation System

The combat cache invalidation system uses a hook-based architecture to automatically invalidate caches when game state changes occur. This ensures cache coherence without requiring manual invalidation calls scattered throughout combat logic.

### 9.1 Architecture Overview

The hook system creates a clean separation between combat logic (write path) and cache management (invalidation path):

```
Combat System (Write Path)          Hook System (Connector)          Cache Layer (Read Path)
────────────────────────            ───────────────────────          ──────────────────────
CombatActionSystem.ExecuteAttack    →  OnAttackComplete hook   →    SquadInfoCache.MarkDirty()
CombatMovementSystem.MoveSquad      →  OnMoveComplete hook     →    SquadInfoCache.MarkDirty()
TurnManager.EndTurn                 →  OnTurnEnd hook          →    SquadInfoCache.MarkAllDirty()
```

**Key Design Principles:**
- **Orthogonal Concerns:** Combat logic focuses on game rules; cache invalidation is handled separately via hooks
- **Single Registration Point:** All cache invalidation callbacks registered in one place (combatmode.go)
- **Automatic Propagation:** Hooks fire for both player actions and AI actions (no special cases)
- **Lifecycle Management:** Callbacks cleared on combat exit to prevent stale references

**Files Involved:**
- `tactical/combatservices/combat_events.go` - Hook callback type definitions
- `tactical/combatservices/combat_service.go` - Hook registration and callback storage
- `tactical/combat/combatactionsystem.go` - Fires OnAttackComplete
- `tactical/combat/combatmovementsystem.go` - Fires OnMoveComplete
- `tactical/combat/turnmanager.go` - Fires OnTurnEnd
- `gui/guicombat/combatmode.go` - Registers cache invalidation callbacks

---

### 9.2 The Three Combat Hooks

#### 9.2.1 OnAttackComplete Hook

**Purpose:** Invalidate squad caches after combat damage is applied.

**Fired By:** `CombatActionSystem.ExecuteAttackAction()` (combatactionsystem.go:168-170)
```go
// After all damage applied and squads potentially destroyed
if cas.onAttackComplete != nil {
    cas.onAttackComplete(attackerID, defenderID, result)
}
```

**Registered At:** `CombatService.NewCombatService()` (combat_service.go:71-75)
```go
combatActSystem.SetOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
    for _, fn := range cs.onAttackComplete {
        fn(attackerID, defenderID, result)
    }
})
```

**GUI Callback:** `combatmode.go:111-120`
```go
cm.combatService.RegisterOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
    cm.Queries.MarkSquadDirty(attackerID)  // Attacker HP/action state changed
    cm.Queries.MarkSquadDirty(defenderID)  // Defender HP changed
    if result.AttackerDestroyed {
        cm.Queries.InvalidateSquad(attackerID)  // Complete removal
    }
    if result.TargetDestroyed {
        cm.Queries.InvalidateSquad(defenderID)  // Complete removal
    }
})
```

**What Gets Invalidated:**
- Attacker squad cache (HP, action state)
- Defender squad cache (HP, potentially destroyed units)
- Complete removal for destroyed squads (InvalidateSquad vs MarkDirty)

**When It Fires:**
- After all unit attacks processed
- After counterattacks resolved
- After damage applied to ECS components
- After dead units disposed
- After squads potentially destroyed

---

#### 9.2.2 OnMoveComplete Hook

**Purpose:** Invalidate squad caches after movement completes.

**Fired By:** `CombatMovementSystem.MoveSquad()` (combatmovementsystem.go:109-112)
```go
// After squad and members moved atomically
if ms.onMoveComplete != nil {
    ms.onMoveComplete(squadID)
}
```

**Registered At:** `CombatService.NewCombatService()` (combat_service.go:77-81)
```go
movementSystem.SetOnMoveComplete(func(squadID ecs.EntityID) {
    for _, fn := range cs.onMoveComplete {
        fn(squadID)
    }
})
```

**GUI Callback:** `combatmode.go:122-124`
```go
cm.combatService.RegisterOnMoveComplete(func(squadID ecs.EntityID) {
    cm.Queries.MarkSquadDirty(squadID)  // Position and movement remaining changed
})
```

**What Gets Invalidated:**
- Squad position cache
- Movement remaining (action state)
- HasMoved flag

**When It Fires:**
- After PositionComponent updated
- After PositionSystem updated
- After movement cost deducted
- After HasMoved flag set

---

#### 9.2.3 OnTurnEnd Hook

**Purpose:** Invalidate all caches and update threat evaluations at turn boundaries.

**Fired By:** `TurnManager.EndTurn()` (turnmanager.go:145-148)
```go
// After turn index advanced and action states reset
if tm.onTurnEnd != nil {
    tm.onTurnEnd(turnState.CurrentRound)
}
```

**Registered At:** `CombatService.NewCombatService()` (combat_service.go:83-87)
```go
turnManager.SetOnTurnEnd(func(round int) {
    for _, fn := range cs.onTurnEnd {
        fn(round)
    }
})
```

**GUI Callback:** `combatmode.go:126-130`
```go
cm.combatService.RegisterOnTurnEnd(func(round int) {
    cm.Queries.MarkAllSquadsDirty()            // All action states reset
    cm.visualization.UpdateThreatManagers()     // Recalculate threat layers
    cm.visualization.UpdateThreatEvaluator(round)  // Update AI evaluation
})
```

**What Gets Invalidated:**
- All squad caches (action states reset for new faction)
- Threat evaluation layers (AI needs fresh data)
- Composite threat evaluators (position-based AI scoring)

**When It Fires:**
- After turn index incremented
- After round number potentially incremented
- After new faction's action states reset

---

### 9.3 Hook Registration Flow

**Step 1: Combat Service Construction** (`combat_service.go:48-89`)

The CombatService constructor creates all combat systems and wires their internal hooks to forward to registered GUI callbacks:

```go
func NewCombatService(manager *common.EntityManager) *CombatService {
    // ... create systems ...

    cs := &CombatService{
        // ... system fields ...
        onAttackComplete: []OnAttackCompleteFunc{},  // Empty callback slices
        onMoveComplete:   []OnMoveCompleteFunc{},
        onTurnEnd:        []OnTurnEndFunc{},
    }

    // Wire system hooks to forward to registered callbacks
    combatActSystem.SetOnAttackComplete(func(...) {
        for _, fn := range cs.onAttackComplete { fn(...) }
    })

    movementSystem.SetOnMoveComplete(func(...) {
        for _, fn := range cs.onMoveComplete { fn(...) }
    })

    turnManager.SetOnTurnEnd(func(...) {
        for _, fn := range cs.onTurnEnd { fn(...) }
    })

    return cs
}
```

**Step 2: GUI Registration** (`combatmode.go:110-130`)

During combat mode initialization, the GUI registers cache invalidation callbacks:

```go
func (cm *CombatMode) Initialize(ctx *framework.UIContext) error {
    // ... create combat service and UI ...

    // Register cache invalidation callbacks
    cm.combatService.RegisterOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
        cm.Queries.MarkSquadDirty(attackerID)
        cm.Queries.MarkSquadDirty(defenderID)
        if result.AttackerDestroyed { cm.Queries.InvalidateSquad(attackerID) }
        if result.TargetDestroyed { cm.Queries.InvalidateSquad(defenderID) }
    })

    cm.combatService.RegisterOnMoveComplete(func(squadID ecs.EntityID) {
        cm.Queries.MarkSquadDirty(squadID)
    })

    cm.combatService.RegisterOnTurnEnd(func(round int) {
        cm.Queries.MarkAllSquadsDirty()
        cm.visualization.UpdateThreatManagers()
        cm.visualization.UpdateThreatEvaluator(round)
    })

    return nil
}
```

**Step 3: Hook Firing During Combat**

When combat actions occur, the systems fire their hooks which propagate to all registered callbacks:

```go
// Example: Attack flow
ExecuteAttackAction() → applies damage → cas.onAttackComplete(attackerID, defenderID, result)
    → CombatService forwards to registered callbacks
    → GUI callback: cm.Queries.MarkSquadDirty(attackerID)
    → SquadInfoCache marks squad dirty
    → Next UI update recomputes squad info from ECS
```

---

### 9.4 Manual Invalidation Bypass Cases

Three special cases bypass the hook system and manually invalidate caches because they modify game state outside normal combat flow:

#### 9.4.1 Undo Move

**Location:** `combat_action_handler.go:199-214`

**Why Manual:** Undo reverses a move that already fired OnMoveComplete. Re-firing the hook would be incorrect.

```go
func (cah *CombatActionHandler) UndoLastMove() {
    result := cah.commandExecutor.Undo()

    if result.Success {
        // Manual invalidation - undo doesn't fire hooks
        cah.deps.Queries.MarkAllSquadsDirty()
        cah.addLog(fmt.Sprintf("⟲ Undid: %s", result.Description))
    }
}
```

**Invalidation Strategy:** MarkAllSquadsDirty() instead of per-squad (conservative, ensures correctness)

---

#### 9.4.2 Redo Move

**Location:** `combat_action_handler.go:217-232`

**Why Manual:** Redo re-applies a move that was undone. Hook already fired during original move.

```go
func (cah *CombatActionHandler) RedoLastMove() {
    result := cah.commandExecutor.Redo()

    if result.Success {
        // Manual invalidation - redo doesn't fire hooks
        cah.deps.Queries.MarkAllSquadsDirty()
        cah.addLog(fmt.Sprintf("⟳ Redid: %s", result.Description))
    }
}
```

**Invalidation Strategy:** MarkAllSquadsDirty() for safety

---

#### 9.4.3 Debug Kill Squad

**Location:** `combat_action_handler.go:283-302`

**Why Manual:** Debug action bypasses normal combat flow. No damage applied, squad just removed.

```go
func (cah *CombatActionHandler) DebugKillSquad(squadID ecs.EntityID) {
    // Use normal cleanup path
    combat.RemoveSquadFromMap(squadID, cah.deps.Queries.ECSManager)

    // Manual invalidation - debug actions don't fire hooks
    cah.deps.Queries.InvalidateSquad(squadID)

    cah.addLog(fmt.Sprintf("[DEBUG] Killed squad: %s", squadName))
}
```

**Invalidation Strategy:** InvalidateSquad() (complete removal, not MarkDirty)

---

### 9.5 Cleanup and Lifecycle Management

#### Clearing Callbacks on Combat Exit

**Location:** `combat_service.go:269` (called from `CleanupCombat()`)

```go
func (cs *CombatService) ClearCallbacks() {
    cs.onAttackComplete = nil
    cs.onMoveComplete = nil
    cs.onTurnEnd = nil
}
```

**Why This Matters:**
- GUI state is torn down when exiting combat mode
- Callbacks reference `cm.Queries` and `cm.visualization` which become invalid
- Leaving callbacks registered would cause nil pointer dereferences if systems fire hooks after mode exit
- Called during `CleanupCombat()` before entity disposal

**Call Site:** `combat_service.go:266-289`
```go
func (cs *CombatService) CleanupCombat(enemySquadIDs []ecs.EntityID) {
    // Clear registered callbacks (they reference GUI state being torn down)
    cs.ClearCallbacks()

    // ... rest of cleanup ...
}
```

---

### 9.6 Read Path vs Write Path Separation

The hook system maintains a clean separation between data access patterns:

#### Write Path (Combat Logic)
- `CombatActionSystem.ExecuteAttackAction()` - Modifies HP, action states
- `CombatMovementSystem.MoveSquad()` - Modifies positions, movement remaining
- `TurnManager.EndTurn()` - Modifies turn state, resets actions
- These systems NEVER query caches or invalidate caches
- They only fire hooks at completion boundaries

#### Read Path (GUI and AI)
- `SquadInfoCache.GetSquadInfo()` - Reads aggregate HP, action states
- `CombatQueryCache.FindActionStateBySquadID()` - Reads action availability
- `GUIQueries.GetSquadInfo()` - Reads comprehensive squad data
- These systems NEVER modify game state
- They only react to hook callbacks by marking dirty flags

#### Hook System (Connector)
- Bridges write path to read path
- Fires at discrete event boundaries (attack complete, move complete, turn end)
- Allows multiple observers (GUI cache, AI cache, visualization)
- Decouples combat logic from cache management

**Benefit:** Combat systems can be tested without GUI, AI can be tested without graphics, caches can be validated independently.

---

### 9.7 Hook System Benefits

**Automatic Invalidation:**
- No manual invalidation calls scattered in combat logic
- Hooks fire for both player and AI actions (no special cases)
- Single registration point makes invalidation logic auditable

**Correct Timing:**
- Hooks fire AFTER state modifications complete
- Caches never see intermediate/invalid state
- Hooks fire atomically (no interleaving with game logic)

**Multiple Observers:**
- GUI caches invalidate on hooks
- AI threat layers invalidate on hooks
- Visualization updates on hooks
- Easy to add new observers without modifying combat systems

**Lifecycle Safety:**
- Callbacks cleared on combat exit
- No stale references to torn-down GUI state
- Systems can be tested without GUI (hooks optional)

**Performance:**
- Invalidation only at discrete events (not every frame)
- Lazy recomputation (caches rebuild on next access, not immediately)
- Minimal overhead (simple function calls, no complex event systems)

---

### 9.8 Common Patterns

#### Pattern 1: Marking Individual Squads Dirty

Use when squad state changes but squad still exists:

```go
cm.combatService.RegisterOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
    cm.Queries.MarkSquadDirty(attackerID)   // HP or action state changed
    cm.Queries.MarkSquadDirty(defenderID)   // HP changed
})
```

#### Pattern 2: Invalidating Destroyed Squads

Use when squad is completely removed from combat:

```go
if result.TargetDestroyed {
    cm.Queries.InvalidateSquad(defenderID)  // Complete removal from cache
}
```

**Difference:**
- `MarkSquadDirty()` - Flags for recomputation, squad still exists
- `InvalidateSquad()` - Complete removal, squad no longer queryable

#### Pattern 3: Marking All Squads Dirty

Use when global state changes (turn boundaries, undo/redo):

```go
cm.combatService.RegisterOnTurnEnd(func(round int) {
    cm.Queries.MarkAllSquadsDirty()  // All action states reset
})
```

#### Pattern 4: Hook + Cache Update

Use when hooks trigger both cache invalidation and visualization updates:

```go
cm.combatService.RegisterOnTurnEnd(func(round int) {
    cm.Queries.MarkAllSquadsDirty()            // Cache invalidation
    cm.visualization.UpdateThreatManagers()     // Visualization update
})
```

---

### 9.9 Testing Cache Invalidation

#### Verifying Hook Firing

```go
func TestAttackFiresHook(t *testing.T) {
    hookFired := false
    attackerID := ecs.EntityID(0)

    combatService.RegisterOnAttackComplete(func(attacker, defender ecs.EntityID, result *squads.CombatResult) {
        hookFired = true
        attackerID = attacker
    })

    combatService.CombatActSystem.ExecuteAttackAction(squadA, squadB)

    assert.True(t, hookFired, "OnAttackComplete should fire")
    assert.Equal(t, squadA, attackerID, "Hook should receive correct attacker ID")
}
```

#### Verifying Cache Invalidation

```go
func TestCacheInvalidationOnAttack(t *testing.T) {
    // Get initial cached data
    info1 := guiQueries.GetSquadInfo(squadA)

    // Perform attack
    combatService.CombatActSystem.ExecuteAttackAction(squadA, squadB)

    // Get updated cached data
    info2 := guiQueries.GetSquadInfo(squadA)

    // Verify action state changed
    assert.NotEqual(t, info1.HasActed, info2.HasActed, "Cache should reflect new action state")
}
```

#### Verifying Callback Cleanup

```go
func TestCallbacksClearedOnExit(t *testing.T) {
    combatService.RegisterOnAttackComplete(func(...) { /* callback */ })

    combatService.CleanupCombat(enemySquads)

    // Verify no callbacks remain
    assert.Empty(t, combatService.onAttackComplete, "Callbacks should be cleared")
}
```

---

### 9.10 Future Improvements

**Potential Enhancements:**
1. **Hook Metrics** - Track hook firing frequency for performance analysis
2. **Conditional Callbacks** - Register callbacks with predicates (only fire for specific factions)
3. **Priority Ordering** - Allow callbacks to specify execution order
4. **Deferred Invalidation** - Batch invalidations at frame boundaries
5. **Hook Replay** - Record and replay hook sequences for testing

**Current Limitations:**
- No ordering guarantees for multiple registered callbacks
- No error handling if callback throws (would need panic recovery)
- No way to unregister specific callbacks (only ClearCallbacks() clears all)

---

## Best Practices

### When to Add a Cache

**Add a cache when:**
- Profiler shows query/computation as bottleneck
- Same data queried multiple times per frame
- Computation is expensive (O(n²), aggregation, pathfinding)
- Data changes infrequently relative to access frequency

**Don't add a cache when:**
- Query/computation is already fast (< 0.1ms)
- Data accessed once per frame or less
- Cache invalidation logic would be complex/error-prone
- Memory is constrained

---

### Cache Invalidation Guidelines

**ECS Views:** Automatic - no action required

**Spatial Grids:** Use helper methods
```go
// ✅ CORRECT - Automatic position system update
manager.MoveEntity(entityID, entity, oldPos, newPos)

// ❌ WRONG - Manual component update without position system
pos.X = newX
pos.Y = newY
```

**Hook-Based Invalidation (Combat Systems):** Register callbacks once, fire automatically
```go
// ✅ CORRECT - Register callbacks during mode initialization
cm.combatService.RegisterOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
    cm.Queries.MarkSquadDirty(attackerID)
    cm.Queries.MarkSquadDirty(defenderID)
    if result.TargetDestroyed { cm.Queries.InvalidateSquad(defenderID) }
})

// ✅ CORRECT - Combat systems fire hooks, don't invalidate caches directly
if cas.onAttackComplete != nil {
    cas.onAttackComplete(attackerID, defenderID, result)
}

// ❌ WRONG - Combat logic should NOT invalidate caches directly
combatActSystem.ExecuteAttack(...)
guiQueries.MarkSquadDirty(squadID)  // This should be in a hook callback
```

**Manual Invalidation (Special Cases):** Only for actions that bypass hooks
```go
// Undo/Redo - hooks already fired during original action
cah.commandExecutor.Undo()
cah.deps.Queries.MarkAllSquadsDirty()  // Manual invalidation required

// Debug actions - bypass normal combat flow
combat.RemoveSquadFromMap(squadID, manager)
queries.InvalidateSquad(squadID)  // Manual invalidation required
```

**Event-Driven Caches (Non-Combat):** Explicit invalidation at event sites
```go
// After damage (non-combat)
guiQueries.MarkSquadDirty(squadID)

// After turn end (via hook callback)
guiQueries.MarkAllSquadsDirty()

// After squad destroyed
guiQueries.InvalidateSquad(squadID)
```

**Rendering Caches:** Clear/mark dirty based on content changes
```go
// Every frame
renderingCache.ClearSpriteBatches()

// When entities added/removed
renderingCache.RefreshRenderablesView(manager)

// When widget content changes
cachedList.MarkDirty()
```

**Threat Layers:** Invalidate on game state changes (via hook callbacks)
```go
// Via OnTurnEnd hook (automatic)
cm.combatService.RegisterOnTurnEnd(func(round int) {
    cm.visualization.UpdateThreatManagers()
    cm.visualization.UpdateThreatEvaluator(round)
})

// Manual update at AI turn start
threatEvaluator.Update(currentRound)
```

---

### Testing Cache Correctness

**Verify cache hits:**
```go
// Widget caches expose render count
fmt.Printf("Renders: %d\n", cachedList.GetRenderCount())
```

**Test invalidation:**
```go
// Get cached value
info1 := cache.GetSquadInfo(squadID)

// Modify game state
DamageSquad(squadID, 10)

// Verify cache sees change
info2 := cache.GetSquadInfo(squadID)
assert.NotEqual(t, info1.CurrentHP, info2.CurrentHP)
```

**Compare cached vs canonical:**
```go
// Cached query
result1 := cache.GetSquadEntity(squadID)

// Canonical query
result2 := squads.GetSquadEntity(squadID, manager)

assert.Equal(t, result1, result2)
```

---

## Future Improvements

### Potential Optimizations

1. **Viewport Culling Cache** - Cache visibility calculations for large maps
2. **Path Cache** - Cache computed paths for repeated queries
3. **FOV Cache** - Cache field-of-view calculations when player/squads don't move
4. **Ability Target Cache** - Cache valid ability targets per position
5. **Formation Cache** - Cache valid formation positions for squad sizes

### Monitoring

Consider adding cache statistics:
- Hit rate (cached vs recomputed)
- Memory usage per cache type
- Invalidation frequency
- Query time distribution (cached vs uncached)

---

## Conclusion

TinkerRogue uses a layered caching strategy:

1. **ECS Views** - Automatic, always-correct, O(k) queries (foundation)
2. **Spatial Grids** - O(1) position lookups (critical for gameplay)
3. **Event-Driven Caches** - Turn-based aggregates (optimal for UI)
4. **Hook-Based Invalidation** - Automatic cache invalidation via combat events (clean separation)
5. **Threat Layers** - Round-based AI (expensive computations)
6. **Rendering Caches** - Frame-level drawing (GPU bottleneck mitigation)

The key insight is matching cache invalidation strategy to data access patterns:
- **Frame-level:** Clear every frame (sprite batches)
- **Event-driven:** Invalidate on specific game events (SquadInfoCache)
- **Hook-based:** Automatic invalidation via registered callbacks (combat caches)
- **Round-based:** Recompute per turn (threat layers)
- **Viewport-based:** Rebuild on camera movement (tile renderer)
- **Size-based:** Rebuild on dimension changes (border cache, overlay cache)
- **Automatic:** ECS Views (no manual management)

**Hook-Based Invalidation Benefits:**
- Eliminates manual cache invalidation in combat logic
- Works for both player and AI actions automatically
- Clean separation between combat systems (write path) and caches (read path)
- Single registration point makes invalidation auditable
- Easy to add new cache observers without modifying combat code

When in doubt, start without caching and profile. Premature caching adds complexity without guaranteed benefit. Let the profiler guide optimization efforts.
