# TinkerRogue Performance Guide

**Last Updated:** 2026-03-02

A consolidated reference for performance patterns and optimizations used throughout TinkerRogue. For cache-specific details see [CACHING_OVERVIEW.md](CACHING_OVERVIEW.md). For ECS-specific performance patterns see [ECS_BEST_PRACTICES.md](ECS_BEST_PRACTICES.md). For GUI performance see [GUI_DOCUMENTATION.md](GUI_DOCUMENTATION.md).

---

## Table of Contents

1. [Rendering Performance](#rendering-performance)
2. [Component Access Performance](#component-access-performance)
3. [Spatial Query Performance](#spatial-query-performance)
4. [Map Key Performance](#map-key-performance)
5. [Query Optimization](#query-optimization)
6. [Entity Lifecycle Optimization](#entity-lifecycle-optimization)
7. [State Change Tracking](#state-change-tracking)
8. [ECS View Usage](#ecs-view-usage)
9. [Ebiten / ebitenui Usage](#ebiten--ebitenui-usage)

---

## Rendering Performance

TinkerRogue has undergone significant rendering optimization in 2026.

### Key Optimizations

1. **Batched Tile Rendering** - Draw all tiles of the same biome in a single batch call (see `visual/rendering/tilebatch.go` and `tilerenderer.go`)
2. **Batched Sprite Rendering** - Batch entity sprites together
3. **Static Panel Backgrounds** - Pre-render panel backgrounds to reduce NineSlice overhead
4. **Cached Background Images** - Reuse background textures across frames
5. **DrawImageOptions Reuse** - Avoid allocating new options every frame

### Image Reuse Pattern

**Before (slow):**
```go
// Created new image every frame
for _, tile := range tiles {
    overlay := ebiten.NewImage(tileSize, tileSize)
    overlay.Fill(color)
    screen.DrawImage(overlay, opts)
}
```

**After (fast):**
```go
// Reuse single overlay image
overlay := ebiten.NewImage(tileSize, tileSize)
for _, tile := range tiles {
    overlay.Fill(color)
    screen.DrawImage(overlay, opts)
    overlay.Clear()
}
```

### Viewport Caching

Only recreate viewport renderers when screen size actually changes:

```go
func (mtr *MovementTileRenderer) Render(screen *ebiten.Image, centerPos coords.LogicalPosition, validTiles []coords.LogicalPosition) {
    screenX, screenY := screen.Bounds().Dx(), screen.Bounds().Dy()

    // Only recreate renderer if screen size changed or not yet created
    if mtr.cachedRenderer == nil || mtr.lastScreenSizeX != screenX || mtr.lastScreenSizeY != screenY {
        mtr.cachedRenderer = NewViewportRenderer(screen, centerPos)
        mtr.lastScreenSizeX = screenX
        mtr.lastScreenSizeY = screenY
    } else if mtr.lastCenterPos != centerPos {
        mtr.cachedRenderer.UpdateCenter(centerPos)
        mtr.lastCenterPos = centerPos
    }
}
```

### Overlay Image Caching

```go
// Create overlay image only once or when size changes
if vr.overlayCache == nil || vr.overlayTileSize != tileSize {
    vr.overlayCache = ebiten.NewImage(tileSize, tileSize)
    vr.overlayTileSize = tileSize
}
vr.overlayCache.Fill(fillColor)
```

---

## Component Access Performance

**Function Selection** (fastest to slowest):

| Function | Speed | Use When |
|----------|-------|----------|
| `GetComponentType` | Fastest | Entity already available |
| `GetComponentTypeByIDWithTag` | 10-100x faster than ByID | Known tag narrows search |
| `GetComponentTypeByID` | Slowest | Searches all entities |

```go
// SLOW - Searches all 1000+ entities
data := common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)

// FAST - Searches only 10-50 squad entities
data := common.GetComponentTypeByIDWithTag[*SquadData](
    manager, squadID, SquadTag, SquadComponent)
```

### Get Entity Once, Extract Multiple Components

When you need multiple components from the same entity, find the entity once instead of calling `GetComponentTypeByID` multiple times:

```go
// OPTIMIZATION: Get entity once, extract all components
entity := common.FindEntityByID(gem.entityManager, cell.unitID)
if entity == nil {
    continue
}

gridPosData := common.GetComponentType[*squads.GridPositionData](entity, squads.GridPositionComponent)
nameData := common.GetComponentType[*common.Name](entity, common.NameComponent)
roleData := common.GetComponentType[*squads.UnitRoleData](entity, squads.UnitRoleComponent)
```

This replaces 3 separate `GetComponentTypeByID` calls with 1 `FindEntityByID` + 3 direct component lookups.

---

## Spatial Query Performance

The `GlobalPositionSystem` provides O(1) position lookups using value-based map keys.

**Before O(n):**
```go
// Searched every entity with position
for _, result := range manager.World.Query(PositionTag) {
    pos := common.GetPosition(result.Entity)
    if pos.X == targetX && pos.Y == targetY {
        // Found after checking 500+ entities
    }
}
```

**After O(1):**
```go
// Direct hash map lookup
entityIDs := common.GlobalPositionSystem.GetEntitiesAtPosition(logicalPos)
// Returns immediately
```

**Impact**: 50x faster with 50+ entities, scales linearly vs. quadratically.

---

## Map Key Performance

**ALWAYS use value types as map keys:**

```go
// SLOW - Pointer keys require temporary pointer creation
grid map[*coords.LogicalPosition][]ecs.EntityID

// To query:
tempPos := &coords.LogicalPosition{X: 10, Y: 20}  // Allocation!
entities := grid[tempPos]  // Won't work - different pointer

// FAST - Value keys use struct equality
grid map[coords.LogicalPosition][]ecs.EntityID

// To query:
entities := grid[coords.LogicalPosition{X: 10, Y: 20}]  // Works!
```

**Measured**: 50x performance difference in PositionSystem refactor.

---

## Query Optimization

### When to Cache

- Tight inner loops (Update/Render every frame)
- Large entity sets (1000+)
- Profile first to confirm bottleneck

### When NOT to Cache

- One-time queries
- Small entity sets (<100)
- Infrequent operations

### Example: Combat Resolution Cache

```go
// Combat resolution queries hundreds of units per attack
// Profile showed 30% of frame time in queries
// Solution: Cache unit positions at start of combat resolution

type CombatCache struct {
    unitPositions map[ecs.EntityID]coords.LogicalPosition
}

func buildCombatCache(squadID ecs.EntityID, manager *common.EntityManager) CombatCache {
    cache := CombatCache{unitPositions: make(map[ecs.EntityID]coords.LogicalPosition)}

    unitIDs := GetUnitIDsInSquad(squadID, manager)
    for _, unitID := range unitIDs {
        pos := common.GetPositionByIDWithTag(manager, unitID, SquadMemberTag)
        cache.unitPositions[unitID] = *pos
    }

    return cache
}

// Now use cache instead of repeated queries
func resolveCombat(cache CombatCache) {
    for unitID := range cache.unitPositions {
        pos := cache.unitPositions[unitID]  // O(1) map lookup
        // ... combat logic
    }
}
```

---

## Entity Lifecycle Optimization

### Batch Entity Creation

```go
// Instead of creating entities one at a time in a loop:
for i := 0; i < 100; i++ {
    entity := manager.World.CreateEntity()
    entity.AddComponent(/* ... */)
    entity.AddTag(/* ... */)
}

// Create all entities first, then add components:
entities := make([]*ecs.Entity, 100)
for i := 0; i < 100; i++ {
    entities[i] = manager.World.CreateEntity()
}

for i, entity := range entities {
    entity.AddComponent(/* ... */)
    entity.AddTag(/* ... */)
}
```

### Batch Disposal

```go
// Collect entities to dispose
toDispose := []*ecs.Entity{}
for _, result := range manager.World.Query(DeadTag) {
    toDispose = append(toDispose, result.Entity)
}

// Dispose all at once
manager.World.DisposeEntities(toDispose...)
```

---

## State Change Tracking

A recurring optimization pattern: track whether state has changed, and only perform expensive updates when it has.

### GUI State Change Detection

```go
// Only update UI displays when state changes
// This avoids expensive text measurement on every frame (~10-15s CPU savings)
func (cm *CombatMode) Update(deltaTime float64) error {
    currentFactionID := cm.combatService.GetCurrentFaction()
    if cm.lastFactionID != currentFactionID {
        cm.turnOrderComponent.Refresh()
        cm.lastFactionID = currentFactionID
        if cm.lastFactionID != 0 {
            cm.factionInfoComponent.ShowFaction(cm.lastFactionID)
        }
    }

    battleState := cm.Context.ModeCoordinator.GetBattleMapState()
    if cm.lastSelectedSquad != battleState.SelectedSquadID {
        cm.lastSelectedSquad = battleState.SelectedSquadID
        if cm.lastSelectedSquad != 0 {
            cm.squadDetailComponent.ShowSquad(cm.lastSelectedSquad)
        }
    }

    return nil
}
```

### Struct Initialization Caching

Avoid repeated struct creation — initialize expensive types once:

```go
func NewCoordinateManager(screenData ScreenData) *CoordinateManager {
    cm := &CoordinateManager{
        dungeonWidth:  screenData.DungeonWidth,
        dungeonHeight: screenData.DungeonHeight,
        tileSize:      screenData.TileSize,
        scaleFactor:   screenData.ScaleFactor,
    }

    // Initialize viewport once with origin (will be updated via SetCenter)
    cm.viewport = NewViewport(cm, LogicalPosition{X: 0, Y: 0})

    return cm
}
```

---

## ECS View Usage

ECS Views are automatically maintained by the ECS library and provide faster iteration than repeated queries.

### Creating Views

```go
type GUIQueries struct {
    ECSManager      *common.EntityManager
    factionManager  *combat.FactionManager

    // Cached ECS Views (automatically maintained by ECS library)
    squadView       *ecs.View // All SquadTag entities
    squadMemberView *ecs.View // All SquadMemberTag entities
    actionStateView *ecs.View // All ActionStateTag entities
}

// Another example: rendering view
func NewRenderingCache(manager *common.EntityManager) *RenderingCache {
    return &RenderingCache{
        // Create View - one-time O(n) cost
        // View is automatically maintained when RenderableComponent added/removed
        RenderablesView: manager.World.CreateView(RenderablesTag),
    }
}
```

**When to use Views vs Queries:**
- **Views**: Frequently iterated entity sets (every frame), large entity counts
- **Queries**: One-off lookups, small entity sets, infrequent operations

---

## Ebiten / ebitenui Usage

### Minimizing Image Creation

`ebiten.NewImage` is expensive. Use geometric operations and color tinting instead:

```go
// Border images are filled with white and tinted using ColorScale.
// Geometric manipulations (scale, translate, etc.) are faster than creating new images.
func (cache *BorderImageCache) GetOrCreate(tileSize, thickness int) (*ebiten.Image, *ebiten.Image, *ebiten.Image, *ebiten.Image) {
    if cache.top == nil || cache.tileSize != tileSize || cache.thickness != thickness {
        // Recreate images only on size change
        cache.top = ebiten.NewImage(tileSize, thickness)
        cache.bottom = ebiten.NewImage(tileSize, thickness)
        cache.left = ebiten.NewImage(thickness, tileSize)
        cache.right = ebiten.NewImage(thickness, tileSize)

        white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
        cache.top.Fill(white)
        cache.bottom.Fill(white)
        cache.left.Fill(white)
        cache.right.Fill(white)

        cache.tileSize = tileSize
        cache.thickness = thickness
    }
    return cache.top, cache.bottom, cache.left, cache.right
}
```

### ebitenui Widget Caching

Create GUI widgets only once; update labels instead of recreating:

```go
func (slc *SquadListComponent) Refresh() {
    newFilteredSquads := slc.getFilteredSquads()

    // OPTIMIZATION: Check if squad list changed
    if !slc.squadListChanged(newFilteredSquads) {
        // FAST PATH: No change - just update button labels if needed
        slc.updateButtonLabels(newFilteredSquads)
        return
    }

    // SLOW PATH: Squad list changed - update widgets
    slc.updateButtonWidgets(newFilteredSquads)
    slc.filteredSquads = newFilteredSquads
}

func (slc *SquadListComponent) updateButtonLabels(squadIDs []ecs.EntityID) {
    for _, squadID := range squadIDs {
        button, exists := slc.buttons[squadID]
        if !exists {
            continue
        }
        squadInfo := slc.queries.GetSquadInfo(squadID)
        if squadInfo == nil {
            continue
        }
        // Update button text if it changed
        textWidget := button.Text()
        if textWidget != nil && textWidget.Label != squadInfo.Name {
            textWidget.Label = squadInfo.Name
        }
    }
}
```

### Pre-rendering Static UI Backgrounds (NineSlice Caching)

**Problem**: EbitenUI's `NineSlice.drawTile()` creates new `DrawImageOptions` for every tile draw, causing ~4.7s CPU time and 3,276ms in allocations.

**Solution**: Pre-render static UI backgrounds to cached images.

```go
type CachedBackground struct {
    source      *image.NineSlice
    cachedImage *ebiten.Image
    dirty       bool
    width, height int
}

// Only re-renders when dimensions change or marked dirty
func (cb *CachedBackground) GetImage(w, h int) *ebiten.Image {
    needsRender := cb.cachedImage == nil ||
        cb.width != w || cb.height != h || cb.dirty
    if needsRender {
        cb.render(w, h)
    }
    return cb.cachedImage
}
```

**Global Background Pools** reuse cached backgrounds for common panel sizes:

```go
func GetPanelBackground(w, h int) *ebiten.Image {
    if panelBackgroundPool == nil {
        panelBackgroundPool = NewCachedBackgroundPool(PanelRes.Image)
    }
    return panelBackgroundPool.GetImage(w, h)
}
```

All panels created via `BuildPanel()` have caching enabled by default. Disable for dynamic panels (tooltips, popups) with `WithDynamicBackground()`.

**Performance Impact:**
- **3s CPU time** reduction (2.6% of total)
- **~70% reduction** in NineSlice allocation overhead
- **3,276ms allocation savings** from DrawImageOptions pooling

**When to enable/disable caching:**
- **Enable** (default): Static panels with fixed dimensions, panels visible for multiple frames
- **Disable**: Tooltips (short-lived), popups (created/destroyed frequently), panels that resize every frame

See `gui/widgetresources/cachedbackground.go` and `gui/widgets/cachedpanels.go` for implementation details.

### Batching Drawing Operations

Tiles of the same biome use the same image, enabling batch drawing. See `visual/rendering/tilebatch.go` and `visual/rendering/tilerenderer.go` for implementation.

---

**See also:**
- [CACHING_OVERVIEW.md](CACHING_OVERVIEW.md) - Cache architecture and invalidation strategies
- [ECS_BEST_PRACTICES.md](ECS_BEST_PRACTICES.md) - ECS patterns and component organization
- [GUI_DOCUMENTATION.md](GUI_DOCUMENTATION.md) - GUI-specific optimization (90% CPU reduction via cached widgets)
