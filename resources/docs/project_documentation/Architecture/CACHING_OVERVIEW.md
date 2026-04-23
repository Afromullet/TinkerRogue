# TinkerRogue Caching Mechanisms

**Last Updated:** 2026-04-22

This document provides a comprehensive overview of all caching mechanisms used throughout the TinkerRogue codebase. Caching is critical for performance in this ECS-based roguelike, reducing O(n) queries to O(1) or O(k) lookups where k is the number of matching entities.

---

## Table of Contents

1. [ECS View-Based Caches](#ecs-view-based-caches)
2. [Event-Driven Caches](#event-driven-caches)
3. [Spatial Grid Caches](#spatial-grid-caches)
4. [Rendering Caches](#rendering-caches)
5. [AI Threat Evaluation](#ai-threat-evaluation)
6. [Widget Render Caches](#widget-render-caches)
7. [Cache Relationships](#cache-relationships)
8. [Performance Characteristics](#performance-characteristics)
9. [PowerPipeline and Hook-Based Cache Invalidation](#powerpipeline-and-hook-based-cache-invalidation)

---

## 1. ECS View-Based Caches

ECS Views are automatically-maintained caches provided by the bytearena/ecs library. When components are added or removed from entities, the library automatically updates all relevant Views. This provides O(k) iteration where k = entities with the component, versus O(n) for World.Query() where n = all entities.

### 1.1 SquadQueryCache

**File:** `tactical/squads/squadcore/squadcache.go`

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

**File:** `tactical/combat/combatstate/combatqueriescache.go`

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
**File:** `campaign/overworld/core/init.go`

```go
var OverworldNodeView *ecs.View
var OverworldFactionView *ecs.View

// Initialized in init():
OverworldNodeView = em.World.CreateView(OverworldNodeTag)
OverworldFactionView = em.World.CreateView(OverworldFactionTag)
```

**Usage:** `campaign/overworld/influence/queries.go` uses `core.OverworldNodeView.Get()` for O(k) node queries

---

#### Squad Member View
**File:** `tactical/squads/squadcore/squadmanager.go`

```go
var squadMemberView *ecs.View

// Initialized in init():
squadMemberView = em.World.CreateView(SquadMemberTag)
```

**Usage:** Zero-allocation squad member queries within the squads package

---

#### Combat Views
**File:** `tactical/combat/combatstate/combatcomponents.go`

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

### 1.6 Power-System Dispatchers (Artifact / Perk)

The power system caches per-battle and per-round state that the combat pipeline
needs to query on every attack, move, and turn-end. These dispatchers are not
ECS Views, but they live in the same subscriber pool as the GUI cache callbacks
(see Section 9) and share the same "built once, fired many times" pattern.

#### 1.6.1 ArtifactChargeTracker

**File:** `tactical/powers/artifacts/artifactcharges.go`

**Purpose:** Tracks charges for active artifact behaviors (per-battle and
per-round budgets). Constructed once in `NewCombatService` and `Reset()` per
battle inside `InitializeCombat`. The instance identity is preserved across
battles so that the `ArtifactDispatcher`'s bindings on the `PowerPipeline`
stay valid.

**Key API:**
```go
UseCharge(behavior string, chargeType ChargeType)
IsAvailable(behavior string) bool
RefreshRoundCharges()     // called on turn boundaries via OnTurnEnd subscriber
Reset()                   // called on battle boundaries via InitializeCombat
```

#### 1.6.2 ArtifactDispatcher

**File:** `tactical/powers/artifacts/dispatcher.go`

Subscribes to `OnPostReset`, `OnAttackComplete`, and `OnTurnEnd` on the
pipeline. Consults the charge tracker before firing behaviors so that
per-round/per-battle budgets are honored without the rest of combat knowing
about charges.

#### 1.6.3 SquadPerkDispatcher

**File:** `tactical/powers/perks/dispatcher.go`

Subscribes to all four pipeline events. Dispatches the corresponding perk
lifecycle hooks (`DispatchTurnStart`, `DispatchAttackTracking`,
`DispatchMoveTracking`, `DispatchRoundEnd`) into each squad's per-squad perk
round-state components.

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

**File:** `core/common/positionsystem.go`

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

**File:** `visual/maprender/tilerenderer.go`

**Purpose:** Cache tile batches when viewport hasn't moved to avoid rebuilding sprite batches every frame.

**Structure:**
```go
type TileRenderer struct {
    tiles   []*worldmapcore.Tile
    batches map[*ebiten.Image]*rendering.QuadBatch

    // Cache state
    lastCenterX      int
    lastCenterY      int
    lastViewportSize int
    batchesBuilt     bool
}
```

The rebuild check also consults `RenderOptions.TileColorsDirty` so palette/tint
changes force a refresh even when the viewport has not moved.

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

### 4.5 CachedViewport

**File:** `visual/rendering/viewport.go`

**Purpose:** Share a single `ViewportRenderer` across multiple tile renderers and
avoid recreating it when only the camera center changes.

**Structure:**
```go
type CachedViewport struct {
    renderer    *ViewportRenderer
    lastCenter  coords.LogicalPosition
    lastScreenX int
    lastScreenY int
}
```

**Cache Strategy:** Recreate-on-dimension-change
- `GetRenderer(screen, centerPos)` rebuilds the underlying `ViewportRenderer`
  only when the screen width/height changes
- Center-position changes update the cached renderer in-place rather than
  allocating a fresh one

**Performance Impact:** Eliminates `ViewportRenderer` allocations during normal
camera panning. Combined with `BorderImageCache` and the overlay cache inside
`ViewportRenderer`, keeps tile-border/overlay rendering allocation-free on the
hot path.

---

## 5. AI Threat Evaluation

AI threat layers are recomputed each time `Update()` is called. An earlier
`DirtyCache` abstraction was removed once analysis showed that its round-based
skip almost never fired in practice — `Update()` is called at well-defined
lifecycle points (AI turn start, turn-end visualization refresh) rather than
opportunistically, so there is no read-heavy hot path to protect.

### 5.1 CompositeThreatEvaluator

**File:** `mind/behavior/threat_composite.go`

**Purpose:** Combines multiple threat layers under a single update entry point.

**Structure:**
```go
type CompositeThreatEvaluator struct {
    manager   *common.EntityManager
    cache     *combat.CombatQueryCache
    factionID ecs.EntityID

    // Individual layers (stateless across calls — Compute() clears and refills maps)
    combatThreat   *CombatThreatLayer
    supportValue   *SupportValueLayer
    positionalRisk *PositionalRiskLayer
}
```

**Update Pattern:**
```go
func (cte *CompositeThreatEvaluator) Update() {
    // Compute layers in dependency order
    cte.combatThreat.Compute()      // First (provides base data)
    cte.supportValue.Compute()      // Depends on combat
    cte.positionalRisk.Compute()    // Depends on combat
}
```

**When Update() is called:**
- Start of each AI faction turn (`AIController.DecideFactionTurn`)
- Turn-end visualization refresh (`CombatVisualizationManager.UpdateThreatEvaluator`)

**Performance:** Update runs at well-defined lifecycle points, not inside AI
per-position scoring, so recomputing unconditionally is cheap in practice. The
earlier round-based skip was removed because it rarely fired.

---

### 5.2 ThreatLayerBase

**File:** `mind/behavior/threat_layers.go`

**Purpose:** Base type shared by all threat layers, holding common dependencies.

**Structure:**
```go
type ThreatLayerBase struct {
    manager   *common.EntityManager
    cache     *combat.CombatQueryCache
    factionID ecs.EntityID
}
```

**Pattern:** Composition-based inheritance
- Concrete layers embed `ThreatLayerBase` for shared dependencies (manager, cache, faction)
- Each layer implements its own `Compute()` that fully clears and repopulates its maps

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
    PowerPipeline subscribers (attack / move / turn-end / post-reset)

PowerPipeline (tactical/powers/powercore/pipeline.go):
  CombatActionSystem   → FireAttackComplete(attacker, defender, result)
  CombatMovementSystem → FireMoveComplete(squadID)
  TurnManager          → FireTurnEnd(round), FirePostReset(factionID, squadIDs)
    ↓ Dispatches in registration order to:
    1. ArtifactDispatcher  (charge tracking, artifact reactions)
    2. SquadPerkDispatcher (perk attack/move tracking, TurnStart, RoundEnd)
    3. onAttackCompleteGUI / onMoveCompleteGUI / onTurnEndGUI (nil-safe)
       ↓ GUI callback invokes
       SquadInfoCache.MarkSquadDirty() / InvalidateSquad() / MarkAllSquadsDirty()
       Threat layer updates (UpdateThreatManagers, UpdateThreatEvaluator)

Threat Evaluation:
  CompositeThreatEvaluator
    ↓ Contains
    CombatThreatLayer, SupportValueLayer, PositionalRiskLayer
    ↓ Each layer embeds
    ThreatLayerBase (manager + CombatQueryCache + factionID)
    ↓ Depends on
    CombatQueryCache (for squad positions, action states)
    ↓ Recomputed by
    AIController.updateThreatLayers (AI turn start) +
    CombatVisualizationManager.UpdateThreatEvaluator (OnTurnEnd hook)

Rendering:
  TileRenderer (viewport position cache; visual/maprender/tilerenderer.go)
  RenderingCache.spriteBatches (image-based batching)
  CachedBackground (NineSlice pre-rendering)
  BorderImageCache (viewport border images)
  ViewportRenderer (overlay cache, reusable DrawImageOptions)
  CachedViewport (shares a single ViewportRenderer across tile renderers)
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
8. CombatMovementSystem fires its raw hook → powerPipeline.FireMoveComplete(squadID)
   ↓
9. PowerPipeline dispatches in registration order:
    a. SquadPerkDispatcher.DispatchMoveTracking (perk state update)
    b. onMoveCompleteGUI → cm.Queries.MarkSquadDirty(squadID)
   ↓
10. SquadInfoCache marks squad's entry dirty
   ↓
11. Next UI update: SquadInfoCache recomputes squad info from ECS
```

**Key Insight:** The pipeline eliminates manual cache invalidation calls in combat logic and keeps artifact, perk, and cache reactions to a single declared order.

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
| CachedViewport | 0 (reused) | O(1) on resize | O(1 renderer) | Sharing a ViewportRenderer across tile renderers |
| PowerPipeline dispatch | O(subs) iteration | 0 (append-only) | O(subs) | Ordered artifact/perk/GUI reactions |
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

## 9. PowerPipeline and Hook-Based Cache Invalidation

Combat cache invalidation is driven by a single dispatcher, `PowerPipeline`,
which sits between the raw subsystem hooks (attack / move / turn-end) and every
observer that needs to react — artifact behaviors, perk lifecycle tracking, and
GUI cache invalidation. Ordering is established once, at pipeline-registration
time in `NewCombatService`, and firing is a single pass down the ordered
subscriber list.

This replaces the older "CombatService maintains three Register* callback
slices" design. There is no `combat_events.go` file, no `Register*` method,
and no `ClearCallbacks` method — GUI callbacks are now single-valued fields
that are rebound on every combat mode initialization.

### 9.1 Architecture Overview

```
Combat System (Write Path)         PowerPipeline (Dispatcher)         Observers (Read Path)
──────────────────────────         ────────────────────────           ──────────────────────
CombatActionSystem.ExecuteAttack   ─►  FireAttackComplete      ─►    ArtifactDispatcher
CombatMovementSystem.MoveSquad     ─►  FireMoveComplete        ─►    SquadPerkDispatcher
TurnManager.EndTurn                ─►  FireTurnEnd             ─►    GUI (SquadInfoCache, visuals)
TurnManager (post-reset)           ─►  FirePostReset
```

**Key Design Principles:**
- **Single dispatcher:** `PowerPipeline` owns the ordered subscriber list for
  each event; subsystems fire directly into it.
- **Order established at registration:** `NewCombatService` registers
  subscribers in the exact firing order: **artifacts → perks → GUI**.
- **Orthogonal concerns:** Combat systems fire raw hooks; pipeline subscribers
  decide what to do (charge tracking, perk tracking, cache invalidation, visuals).
- **Single GUI binding:** GUI callbacks are single-valued fields
  (`onAttackCompleteGUI`, `onMoveCompleteGUI`, `onTurnEndGUI`) set via
  `SetOn*CompleteGUI` methods — assigning a new callback replaces the previous one.

**Files Involved:**
- `tactical/powers/powercore/pipeline.go` — `PowerPipeline` (ordered subscriber lists for PostReset, AttackComplete, MoveComplete, TurnEnd)
- `tactical/powers/artifacts/dispatcher.go` — `ArtifactDispatcher`
- `tactical/powers/artifacts/artifactcharges.go` — `ArtifactChargeTracker`
- `tactical/powers/perks/dispatcher.go` — `SquadPerkDispatcher`
- `tactical/combat/combatservices/combat_service.go` — Wiring: creates pipeline, registers subscribers, exposes `SetOn*CompleteGUI` methods
- `tactical/combat/combatcore/combatactionsystem.go` — Raw `SetOnAttackComplete` hook forwards to `powerPipeline.FireAttackComplete`
- `tactical/combat/combatcore/combatmovementsystem.go` — Raw `SetOnMoveComplete` hook
- `tactical/combat/combatcore/turnmanager.go` — Raw `SetOnTurnEnd` and `SetPostResetHook`
- `gui/guicombat/combatmode.go` — `registerCombatCallbacks` binds the three GUI callbacks

---

### 9.2 The Four Pipeline Events

`PowerPipeline` (defined in `tactical/powers/powercore/pipeline.go`) exposes
four lifecycle events. Each has an `On<Event>` method to register a subscriber
and a `Fire<Event>` method that the combat subsystems call when their action
finishes.

```go
type PowerPipeline struct {
    postReset      []PostResetHandler
    attackComplete []AttackCompleteHandler
    turnEnd        []TurnEndHandler
    moveComplete   []MoveCompleteHandler
}
```

| Event            | Handler signature                                                                 | Fired by                            |
|------------------|-----------------------------------------------------------------------------------|-------------------------------------|
| PostReset        | `func(factionID ecs.EntityID, squadIDs []ecs.EntityID)`                          | `TurnManager` (post-reset hook)     |
| AttackComplete   | `func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)`    | `CombatActionSystem.ExecuteAttackAction` |
| MoveComplete     | `func(squadID ecs.EntityID)`                                                     | `CombatMovementSystem.MoveSquad`    |
| TurnEnd          | `func(round int)`                                                                | `TurnManager.EndTurn`               |

#### 9.2.1 OnAttackComplete

**What runs (in registration order):**
1. `ArtifactDispatcher.DispatchOnAttackComplete` — artifact reactions, charge decrement
2. `SquadPerkDispatcher.DispatchAttackTracking` — perk bookkeeping (attacks landed, hits taken)
3. GUI callback (set by `registerCombatCallbacks`):
   ```go
   cm.Queries.MarkSquadDirty(attackerID)
   cm.Queries.MarkSquadDirty(defenderID)
   if result.AttackerDestroyed { cm.Queries.InvalidateSquad(attackerID) }
   if result.TargetDestroyed   { cm.Queries.InvalidateSquad(defenderID) }
   ```

**Fires after:** all unit attacks processed, counterattacks resolved, damage
applied, dead units disposed, and squad-destroyed flags computed.

---

#### 9.2.2 OnMoveComplete

**What runs (in registration order):**
1. `SquadPerkDispatcher.DispatchMoveTracking` — perk move-based triggers
2. GUI callback:
   ```go
   cm.Queries.MarkSquadDirty(squadID)
   ```

**Fires after:** `PositionComponent` updated for every squad member,
`GlobalPositionSystem` updated, movement cost deducted, and `HasMoved` flag set.

---

#### 9.2.3 OnTurnEnd

**What runs (in registration order):**
1. `ArtifactDispatcher.DispatchOnTurnEnd` — turn-end artifact effects and charge refresh
2. `SquadPerkDispatcher.DispatchRoundEnd` — perk round-end bookkeeping
3. GUI callback:
   ```go
   cm.Queries.MarkAllSquadsDirty()            // All action states reset
   cm.visualization.UpdateThreatManagers()    // Recalculate threat layers
   cm.visualization.UpdateThreatEvaluator()   // Update AI evaluation
   ```

**Fires after:** turn index incremented, round number potentially incremented,
next faction's action states reset.

---

#### 9.2.4 OnPostReset

**What runs (in registration order):**
1. `ArtifactDispatcher.DispatchPostReset` — artifact setup (e.g. Deadlock Shackles must lock before perk TurnStart sees the state)
2. `SquadPerkDispatcher.DispatchTurnStart` — per-round perk start-of-turn hooks

**Fires after:** action states have been reset for the faction whose turn is
about to begin, but before any AI or player input runs. No GUI callback is
registered on this event.

---

### 9.3 Pipeline Wiring Flow

**Step 1: `NewCombatService` registers subscribers** (`tactical/combat/combatservices/combat_service.go`)

All subscribers are appended to the pipeline in a single block. The order in
this block is exactly the order in which they fire at runtime.

```go
cs.powerPipeline = &powercore.PowerPipeline{}

// PostReset: artifacts first (e.g. Deadlock Shackles), then perk TurnStart
cs.powerPipeline.OnPostReset(cs.artifactDispatcher.DispatchPostReset)
cs.powerPipeline.OnPostReset(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
    if cs.perkDispatcher != nil {
        cs.perkDispatcher.DispatchTurnStart(squadIDs, cs.TurnManager.GetCurrentRound(), cs.EntityManager)
    }
})

// AttackComplete: artifacts → perk attack tracking → GUI
cs.powerPipeline.OnAttackComplete(cs.artifactDispatcher.DispatchOnAttackComplete)
cs.powerPipeline.OnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
    if cs.perkDispatcher != nil {
        cs.perkDispatcher.DispatchAttackTracking(attackerID, defenderID, cs.EntityManager)
    }
})
cs.powerPipeline.OnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
    if cs.onAttackCompleteGUI != nil {
        cs.onAttackCompleteGUI(attackerID, defenderID, result)
    }
})

// TurnEnd: artifacts → perk round-end → GUI
cs.powerPipeline.OnTurnEnd(cs.artifactDispatcher.DispatchOnTurnEnd)
cs.powerPipeline.OnTurnEnd(func(round int) {
    if cs.perkDispatcher != nil {
        cs.perkDispatcher.DispatchRoundEnd(cs.EntityManager)
    }
})
cs.powerPipeline.OnTurnEnd(func(round int) {
    if cs.onTurnEndGUI != nil { cs.onTurnEndGUI(round) }
})

// MoveComplete: perk move tracking → GUI
cs.powerPipeline.OnMoveComplete(func(squadID ecs.EntityID) {
    if cs.perkDispatcher != nil {
        cs.perkDispatcher.DispatchMoveTracking(squadID, cs.EntityManager)
    }
})
cs.powerPipeline.OnMoveComplete(func(squadID ecs.EntityID) {
    if cs.onMoveCompleteGUI != nil { cs.onMoveCompleteGUI(squadID) }
})

// Subsystems fire directly into the pipeline — no intermediate wrapper methods.
combatActSystem.SetOnAttackComplete(cs.powerPipeline.FireAttackComplete)
movementSystem.SetOnMoveComplete(cs.powerPipeline.FireMoveComplete)
turnManager.SetOnTurnEnd(cs.powerPipeline.FireTurnEnd)
turnManager.SetPostResetHook(cs.powerPipeline.FirePostReset)
```

**Step 2: GUI binds its three callbacks** (`gui/guicombat/combatmode.go::registerCombatCallbacks`, lines 374–395)

Combat mode calls `SetOn*CompleteGUI` to install the cache-invalidation
callbacks. Because these are single-valued fields, re-initializing combat
mode simply overwrites the previous closure:

```go
func (cm *CombatMode) registerCombatCallbacks() {
    cm.combatService.SetOnAttackCompleteGUI(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
        cm.Queries.MarkSquadDirty(attackerID)
        cm.Queries.MarkSquadDirty(defenderID)
        if result.AttackerDestroyed { cm.Queries.InvalidateSquad(attackerID) }
        if result.TargetDestroyed   { cm.Queries.InvalidateSquad(defenderID) }
    })

    cm.combatService.SetOnMoveCompleteGUI(func(squadID ecs.EntityID) {
        cm.Queries.MarkSquadDirty(squadID)
    })

    cm.combatService.SetOnTurnEndGUI(func(round int) {
        cm.Queries.MarkAllSquadsDirty()
        cm.visualization.UpdateThreatManagers()
        cm.visualization.UpdateThreatEvaluator()
    })
}
```

**Step 3: Firing during combat**

```
ExecuteAttackAction() applies damage
  → combatActSystem.onAttackComplete(attackerID, defenderID, result)
    → powerPipeline.FireAttackComplete iterates subscribers in order:
        1. ArtifactDispatcher.DispatchOnAttackComplete
        2. SquadPerkDispatcher.DispatchAttackTracking
        3. onAttackCompleteGUI → MarkSquadDirty / InvalidateSquad
  → SquadInfoCache recomputes on next GetSquadInfo call
```

---

### 9.4 Manual Invalidation Bypass Cases

Three actions skip the pipeline and invalidate caches directly because they
modify game state outside the normal combat flow (no subsystem hook fires):

#### 9.4.1 Undo Move

**Location:** `gui/guicombat/combat_action_handler.go:166–178`

Undo reverses a move whose `OnMoveComplete` already fired. Re-firing the hook
would double-count perk/move tracking, so the handler invalidates directly.

```go
func (cah *CombatActionHandler) UndoLastMove() {
    if !cah.CanUndoMove() { return }
    result := cah.commandExecutor.Undo()
    if result.Success {
        cah.deps.Queries.MarkAllSquadsDirty()
    }
}
```

---

#### 9.4.2 Redo Move

**Location:** `gui/guicombat/combat_action_handler.go:180–192`

```go
func (cah *CombatActionHandler) RedoLastMove() {
    if !cah.CanRedoMove() { return }
    result := cah.commandExecutor.Redo()
    if result.Success {
        cah.deps.Queries.MarkAllSquadsDirty()
    }
}
```

---

#### 9.4.3 Debug Kill Squad

**Location:** `gui/guicombat/combat_action_handler.go:250–263`

```go
func (cah *CombatActionHandler) DebugKillSquad(squadID ecs.EntityID) {
    if cah.deps.BattleState.SelectedSquadID == squadID {
        cah.deps.BattleState.SelectedSquadID = 0
    }
    if err := combatstate.RemoveSquadFromMap(squadID, cah.deps.Queries.ECSManager); err != nil {
        return
    }
    cah.deps.Queries.InvalidateSquad(squadID)
}
```

`InvalidateSquad` (full removal from the cache) rather than `MarkSquadDirty`
because the squad no longer exists.

---

### 9.5 Callback Lifecycle

There is no `ClearCallbacks` method and `TeardownCombat` does not touch the GUI
callback fields. GUI callbacks are single-valued fields:

```go
type CombatService struct {
    onAttackCompleteGUI func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)
    onMoveCompleteGUI   func(squadID ecs.EntityID)
    onTurnEndGUI        func(round int)
}
```

Each combat mode initialization calls `registerCombatCallbacks`, which
overwrites the previous closures. The closures capture `cm.Queries` and
`cm.visualization`, which are reconstructed together with the callbacks, so
there is no stale-reference hazard as long as pipeline firing is paused while
combat mode is being rebuilt (enforced by the combat mode lifecycle — the
pipeline is not fired outside an initialized combat mode).

If a future change introduces a long-lived CombatService that outlives the
GUI, add an explicit clear step here.

---

### 9.6 Read Path vs Write Path Separation

The pipeline maintains a clean separation between data access patterns:

#### Write Path (Combat Logic)
- `CombatActionSystem.ExecuteAttackAction()` — Modifies HP, action states
- `CombatMovementSystem.MoveSquad()` — Modifies positions, movement remaining
- `TurnManager.EndTurn()` / post-reset hook — Modifies turn state, resets actions
- These systems never query caches or invalidate caches
- They only call `powerPipeline.Fire<Event>` at completion boundaries

#### Read Path (GUI and AI)
- `SquadInfoCache.GetSquadInfo()` — Reads aggregate HP, action states
- `CombatQueryCache.FindActionStateBySquadID()` — Reads action availability
- `GUIQueries.GetSquadInfo()` — Reads comprehensive squad data
- These systems never modify game state
- They only react to pipeline events by marking dirty flags

#### PowerPipeline (Connector)
- Single ordered-dispatch layer between write and read paths
- Fires at discrete event boundaries (attack / move / turn-end / post-reset)
- Supports multiple observers per event (artifacts, perks, GUI)
- Registration order in `NewCombatService` is the firing order

**Benefit:** Combat systems can be tested without GUI, AI can be tested without
graphics, and new power-system subscribers (new artifact families, new perk
categories, analytics exports) can be added with a single `OnX` call instead
of editing four parallel `Fire*` method bodies.

---

### 9.7 Pipeline Benefits

**Automatic Invalidation:**
- No manual invalidation calls scattered in combat logic
- Pipeline fires for both player and AI actions (no special cases)
- Registration block in `NewCombatService` is the single place ordering is declared

**Correct Timing:**
- Subscribers fire AFTER state modifications complete
- Caches never see intermediate/invalid state
- Each `Fire<Event>` call iterates in deterministic registration order

**Multiple Observers:**
- `ArtifactDispatcher` reacts to attack/turn-end/post-reset
- `SquadPerkDispatcher` reacts to attack/move/turn-end/post-reset
- GUI callback marks caches dirty and refreshes visualization
- Adding a new observer is one `powerPipeline.OnX(handler)` call

**Lifecycle Safety:**
- GUI callbacks are single-valued fields; re-initialization overwrites them
- Artifact charge state lives in a long-lived `ArtifactChargeTracker` that is
  `Reset()` per battle, so the dispatcher's bindings on the pipeline stay valid
  across battles

**Performance:**
- Invalidation only at discrete events (not every frame)
- Lazy recomputation (caches rebuild on next access)
- Minimal overhead — a `FireX` call is a for-loop over a small subscriber slice

---

### 9.8 Common Patterns

#### Pattern 1: Marking Individual Squads Dirty

Use when squad state changes but the squad still exists. Wire inside the GUI
callback that `registerCombatCallbacks` installs:

```go
cm.combatService.SetOnAttackCompleteGUI(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
    cm.Queries.MarkSquadDirty(attackerID)
    cm.Queries.MarkSquadDirty(defenderID)
})
```

#### Pattern 2: Invalidating Destroyed Squads

```go
if result.TargetDestroyed {
    cm.Queries.InvalidateSquad(defenderID)
}
```

- `MarkSquadDirty()` — flags for recomputation, squad still exists
- `InvalidateSquad()` — complete removal, squad no longer queryable

#### Pattern 3: Marking All Squads Dirty

```go
cm.combatService.SetOnTurnEndGUI(func(round int) {
    cm.Queries.MarkAllSquadsDirty()
})
```

#### Pattern 4: Adding a New Pipeline Subscriber

To have a new system react to combat lifecycle events (e.g. analytics,
battle recorder, a future power category), append a subscriber in
`NewCombatService` in the correct order relative to artifacts / perks / GUI:

```go
cs.powerPipeline.OnAttackComplete(myAnalytics.RecordAttack)
```

No other code needs to change — subsystems already fire directly into
`powerPipeline.Fire*`.

---

### 9.9 Testing Cache Invalidation

#### Verifying Pipeline Firing

```go
func TestAttackFiresPipeline(t *testing.T) {
    fired := false
    var receivedAttacker ecs.EntityID

    combatService.SetOnAttackCompleteGUI(func(attacker, defender ecs.EntityID, result *combattypes.CombatResult) {
        fired = true
        receivedAttacker = attacker
    })

    combatService.CombatActSystem.ExecuteAttackAction(squadA, squadB)

    assert.True(t, fired, "GUI attack callback should fire")
    assert.Equal(t, squadA, receivedAttacker, "Callback should receive correct attacker ID")
}
```

#### Verifying Cache Invalidation

```go
func TestCacheInvalidationOnAttack(t *testing.T) {
    info1 := guiQueries.GetSquadInfo(squadA)

    combatService.CombatActSystem.ExecuteAttackAction(squadA, squadB)

    info2 := guiQueries.GetSquadInfo(squadA)

    assert.NotEqual(t, info1.HasActed, info2.HasActed, "Cache should reflect new action state")
}
```

#### Verifying Callback Rebinding

```go
func TestGUICallbackIsOverwritten(t *testing.T) {
    firstFired, secondFired := false, false

    combatService.SetOnAttackCompleteGUI(func(...) { firstFired = true })
    combatService.SetOnAttackCompleteGUI(func(...) { secondFired = true })

    combatService.CombatActSystem.ExecuteAttackAction(squadA, squadB)

    assert.False(t, firstFired, "Replaced callback must not fire")
    assert.True(t, secondFired, "Latest callback must fire")
}
```

---

### 9.10 Future Improvements

**Potential Enhancements:**
1. **Pipeline Metrics** — Track subscriber count and per-event firing frequency for profiling
2. **Conditional Subscribers** — Register with predicates (only fire for specific factions/squads)
3. **Deferred Invalidation** — Batch cache invalidations at frame boundaries instead of on every Fire call
4. **Pipeline Replay** — Record and replay event sequences for regression tests

**Current Limitations:**
- Subscribers fire in strict registration order; inserting a new one "between" artifacts and perks requires editing `NewCombatService`
- No panic recovery around individual subscribers — a panic aborts the whole Fire
- No way to unregister a single subscriber (the pipeline is rebuilt only by constructing a fresh `CombatService`)

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

**Pipeline-Based Invalidation (Combat Systems):** Bind GUI callbacks once, fire automatically
```go
// ✅ CORRECT — bind GUI callbacks during combat mode initialization
cm.combatService.SetOnAttackCompleteGUI(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
    cm.Queries.MarkSquadDirty(attackerID)
    cm.Queries.MarkSquadDirty(defenderID)
    if result.TargetDestroyed { cm.Queries.InvalidateSquad(defenderID) }
})

// ✅ CORRECT — combat systems fire the pipeline; the pipeline fans out to artifacts/perks/GUI
combatActSystem.SetOnAttackComplete(cs.powerPipeline.FireAttackComplete)

// ❌ WRONG — combat logic should NOT touch caches directly
combatActSystem.ExecuteAttack(...)
guiQueries.MarkSquadDirty(squadID)  // Belongs in the GUI callback
```

**Manual Invalidation (Special Cases):** Only for actions that bypass the pipeline
```go
// Undo/Redo — original move already fired through the pipeline; bypass now
cah.commandExecutor.Undo()
cah.deps.Queries.MarkAllSquadsDirty()

// Debug actions — bypass normal combat flow entirely
combatstate.RemoveSquadFromMap(squadID, manager)
queries.InvalidateSquad(squadID)
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

**Threat Layers:** Recomputed at lifecycle points (no dirty flag)
```go
// Via the GUI OnTurnEnd callback (installed by registerCombatCallbacks)
cm.combatService.SetOnTurnEndGUI(func(round int) {
    cm.visualization.UpdateThreatManagers()
    cm.visualization.UpdateThreatEvaluator()
})

// At AI turn start
threatEvaluator.Update()
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

1. **ECS Views** — Automatic, always-correct, O(k) queries (foundation)
2. **Spatial Grids** — O(1) position lookups (critical for gameplay)
3. **Event-Driven Caches** — Turn-based aggregates (optimal for UI)
4. **PowerPipeline Dispatch** — Ordered artifact/perk/GUI reactions to combat events (clean separation)
5. **Threat Layers** — Per-turn AI recomputation at lifecycle points
6. **Rendering Caches** — Frame-level drawing (GPU bottleneck mitigation)

The key insight is matching cache invalidation strategy to data access patterns:
- **Frame-level:** Clear every frame (sprite batches)
- **Event-driven:** Invalidate on specific game events (SquadInfoCache)
- **Pipeline-driven:** Ordered dispatch to artifact/perk/GUI subscribers (combat caches)
- **Turn-based:** Recompute at lifecycle points (threat layers)
- **Viewport-based:** Rebuild on camera movement (tile renderer)
- **Size-based:** Rebuild on dimension changes (border cache, overlay cache, CachedViewport)
- **Automatic:** ECS Views (no manual management)

**PowerPipeline Benefits:**
- Eliminates manual cache invalidation in combat logic
- Works for both player and AI actions automatically
- Clean separation between combat systems (write path) and caches (read path)
- Single ordered-subscriber list per event, declared once in `NewCombatService`
- Easy to add new observers — artifacts, perks, analytics, or any future power category — with one `powerPipeline.OnX(handler)` call

When in doubt, start without caching and profile. Premature caching adds complexity without guaranteed benefit. Let the profiler guide optimization efforts.
