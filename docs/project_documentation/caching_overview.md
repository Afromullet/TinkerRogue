# TinkerRogue Caching Mechanisms

**Last Updated:** 2026-02-11

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

**File:** `gui/guiresources/cachedbackground.go`

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
  Package-level Views (OverworldNodeView, etc.)
    ↓
    Used by: GUIQueries, threat evaluators, rendering pipeline

Event-Driven:
  SquadInfoCache
    ↓ Depends on
    SquadQueryCache + CombatQueryCache
    ↓ Invalidated by
    Game events (damage, movement, turn start/end)

Threat Evaluation:
  CompositeThreatEvaluator
    ↓ Contains
    CombatThreatLayer, SupportValueLayer, PositionalRiskLayer
    ↓ Each embeds
    DirtyCache (round-based invalidation)
    ↓ Depends on
    CombatQueryCache (for squad positions, action states)

Rendering:
  TileRenderer (viewport position cache)
  RenderingCache.spriteBatches (image-based batching)
  CachedBackground (NineSlice pre-rendering)
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
3. EntityManager.MoveSquadAndMembers(squadID, oldPos, newPos)
   ↓
4. Updates PositionComponent on all squad members
   ↓
5. GlobalPositionSystem.MoveEntity() for each member (spatial grid update)
   ↓
6. ECS library auto-updates all Views (SquadQueryCache, CombatQueryCache, RenderingCache)
   ↓
7. GUI code calls guiQueries.MarkSquadDirty(squadID)
   ↓
8. SquadInfoCache invalidates squad's entry
   ↓
9. AI controller calls threatEvaluator.MarkDirty()
   ↓
10. All threat layers marked dirty
   ↓
11. Next AI turn: threatEvaluator.Update(currentRound) recomputes layers
```

**Key Insight:** Most caches update automatically (ECS Views), but event-driven caches require manual invalidation calls.

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

**Rendering Caches:** High
- Cached images: width × height × 4 bytes per pixel
- 1920×1080 screen buffer = 8MB
- Multiple UI panels can accumulate significant memory

**Recommendation:** Monitor texture memory usage if caching many large widgets.

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

**Event-Driven Caches:** Explicit invalidation at event sites
```go
// After damage
guiQueries.MarkSquadDirty(squadID)

// After turn end
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

**Threat Layers:** Invalidate on game state changes
```go
// After any squad movement or destruction
threatEvaluator.MarkDirty()

// At start of AI turn
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
4. **Threat Layers** - Round-based AI (expensive computations)
5. **Rendering Caches** - Frame-level drawing (GPU bottleneck mitigation)

The key insight is matching cache invalidation strategy to data access patterns:
- **Frame-level:** Clear every frame (sprite batches)
- **Event-driven:** Invalidate on specific game events (SquadInfoCache)
- **Round-based:** Recompute per turn (threat layers)
- **Viewport-based:** Rebuild on camera movement (tile renderer)
- **Automatic:** ECS Views (no manual management)

When in doubt, start without caching and profile. Premature caching adds complexity without guaranteed benefit. Let the profiler guide optimization efforts.
