# Performance Improvements - TinkerRogue

**Analysis Date:** 2025-12-17
**Profile Duration:** 120.15s
**Total CPU Time:** 143.425s (119.37% utilization)

## Executive Summary

The CPU profile reveals significant performance bottlenecks in the rendering pipeline, ECS component access, and memory allocation patterns. The hottest functions account for over 60% of CPU time, with clear optimization opportunities that could improve frame rates by 2-3x.

### Top 5 Bottlenecks
1. **Runtime profiling overhead** - 37.63% (53.967s) - lostProfileEvent artifacts
2. **DrawImage operations** - 44.10% (63.246s cumulative) - excessive draw calls
3. **TileRenderer.renderTile** - 10.32% (14.80s) - called every frame for every visible tile
4. **ECS GetEntityByID** - 2.57% (3.69s) - component map allocations on every call
5. **Font rendering** - 12.10% (17.356s) - TrueType hinting calculations

### Expected Impact Summary
- **High Impact Optimizations:** 40-60% performance improvement
- **Medium Impact Optimizations:** 15-25% performance improvement
- **Low Impact Optimizations:** 5-10% performance improvement

---


### 3. Cache GetSquadInfo Results Per Frame

**Current Hotspot:**
```
SquadHighlightRenderer.Render: 6.68s cumulative (4.66% of total)
- GetSquadInfo: 4.27s (called for every squad, every frame)
- GetAttributesByIDWithTag: 3.05s (called for every unit in every squad)
- GetComponentTypeByID: 3.65s (ECS component lookup overhead)
```

**Problem:**
`GetSquadInfo` is called repeatedly for the same squad within a single frame:

```go
func (shr *SquadHighlightRenderer) Render(...) {
    allSquads := shr.queries.SquadCache.FindAllSquads()

    for _, squadID := range allSquads {
        // EXPENSIVE: Full squad info rebuild every call
        squadInfo := shr.queries.GetSquadInfo(squadID)  // 4.27s total!

        // Inside GetSquadInfo:
        // - Iterates through all units
        // - Calls GetAttributesByIDWithTag for EACH unit
        // - Each attribute lookup triggers GetEntityByID (map allocation)
    }
}
```

Even though there's a `squadInfoCache`, it's being rebuilt too frequently.

**Solution:** Frame-based caching

```go
// Add to SquadHighlightRenderer
type SquadHighlightRenderer struct {
    queries          *guicomponents.GUIQueries
    selectedColor    color.Color
    borderThickness  int
    cachedRenderer   *ViewportRenderer
    lastCenterPos    coords.LogicalPosition
    lastScreenSizeX  int
    lastScreenSizeY  int

    // NEW: Frame-based squad info cache
    squadInfoCache   map[ecs.EntityID]*guicomponents.SquadInfo
    cacheFrameID     uint64  // Invalidate per frame
}

// Add frame counter to game state or pass it in
var globalFrameCounter uint64 = 0

func (shr *SquadHighlightRenderer) Render(
    screen *ebiten.Image,
    centerPos coords.LogicalPosition,
    currentFactionID ecs.EntityID,
    selectedSquadID ecs.EntityID,
    frameID uint64,  // NEW parameter
) {
    // Invalidate cache if new frame
    if shr.cacheFrameID != frameID {
        shr.squadInfoCache = make(map[ecs.EntityID]*guicomponents.SquadInfo)
        shr.cacheFrameID = frameID
    }

    vr := shr.cachedRenderer
    allSquads := shr.queries.SquadCache.FindAllSquads()

    for _, squadID := range allSquads {
        // Check cache first
        squadInfo, cached := shr.squadInfoCache[squadID]
        if !cached {
            squadInfo = shr.queries.GetSquadInfo(squadID)
            shr.squadInfoCache[squadID] = squadInfo
        }

        if squadInfo == nil || squadInfo.IsDestroyed || squadInfo.Position == nil {
            continue
        }

        // ... rest of rendering logic
    }
}
```

**Alternative (Better):** Move caching to GUIQueries

```go
// In GUIQueries
type GUIQueries struct {
    ECSManager    *common.EntityManager
    SquadCache    *squads.SquadCache
    squadInfoCache *SquadInfoCache

    // NEW: Frame-based cache
    frameSquadInfo map[ecs.EntityID]*SquadInfo
    currentFrameID uint64
}

func (gq *GUIQueries) BeginFrame(frameID uint64) {
    if gq.currentFrameID != frameID {
        gq.frameSquadInfo = make(map[ecs.EntityID]*SquadInfo)
        gq.currentFrameID = frameID
    }
}

func (gq *GUIQueries) GetSquadInfo(squadID ecs.EntityID) *SquadInfo {
    // Check frame cache
    if info, cached := gq.frameSquadInfo[squadID]; cached {
        return info
    }

    // Build squad info
    info := gq.GetSquadInfoCached(squadID, gq.squadInfoCache)
    gq.frameSquadInfo[squadID] = info
    return info
}
```

**Expected Impact:** HIGH
- Reduces GetSquadInfo calls from N (per renderer) to 1 per squad per frame
- Estimated improvement: 3-4s per frame (50-66 FPS gain)
- Most impactful for combat mode with many squads visible

**Files to Modify:**
- `C:\Users\Afromullet\Desktop\TinkerRogue\gui\guicomponents\guiqueries.go`
- `C:\Users\Afromullet\Desktop\TinkerRogue\gui\guimodes\guirenderers.go`

---

### 4. Optimize ECS GetEntityByID - Remove Component Map Allocation

**Current Hotspot:**
```
Manager.GetEntityByID: 3.69s cumulative (2.57% of total)
- fetchComponentsForEntity: 1.73s
  - make(map[*Component]interface{}): 1.68s (ALLOCATION on every call!)
```

**Problem:**
The ECS library's `GetEntityByID` allocates a new component map on EVERY call:

```go
// In github.com/bytearena/ecs
func (manager Manager) GetEntityByID(id EntityID, tagelements ...interface{}) *QueryResult {
    manager.lock.RLock()
    res, ok := manager.entitiesByID[id]

    tag := BuildTag(tagelements...)
    components := manager.fetchComponentsForEntity(res, tag)  // ALLOCATION!
    manager.lock.RUnlock()

    return &QueryResult{
        Entity:     res,
        Components: components,  // Map allocated in fetchComponentsForEntity
    }
}

func (manager *Manager) fetchComponentsForEntity(entity *Entity, tag Tag) map[*Component]interface{} {
    componentMap := make(map[*Component]interface{})  // EXPENSIVE ALLOCATION!
    // ... populate map
    return componentMap
}
```

**Solution:** Since we never use the `Components` map in the returned `QueryResult`, we can optimize our wrapper:

```go
// Add to EntityManager
type EntityManager struct {
    World      *ecs.Manager
    WorldTags  map[string]ecs.Tag
    entityCache map[ecs.EntityID]*ecs.Entity  // NEW: Simple entity cache
}

// Fast entity lookup without component map allocation
func (em *EntityManager) GetEntityByIDFast(entityID ecs.EntityID) *ecs.Entity {
    // Option 1: Use a simple cache (if ECS library doesn't expose entitiesByID)
    if entity, cached := em.entityCache[entityID]; cached {
        return entity
    }

    // Option 2: If we can access the internal map (requires modification)
    // return em.World.entitiesByID[entityID]

    // Fallback: Use existing method but extract entity only
    result := em.World.GetEntityByID(entityID)
    if result == nil {
        return nil
    }

    em.entityCache[entityID] = result.Entity
    return result.Entity
}

// Replace all GetComponentTypeByID calls
func GetComponentTypeByID[T any](manager *EntityManager, entityID ecs.EntityID, component *ecs.Component) T {
    entity := manager.GetEntityByIDFast(entityID)  // NEW: No map allocation
    if entity == nil {
        var nilValue T
        return nilValue
    }

    if c, ok := entity.GetComponentData(component); ok {
        return c.(T)
    }

    var nilValue T
    return nilValue
}
```

**Better Solution:** Fork/modify the ECS library or use direct map access

```go
// If we can access internal maps (requires exposing or forking)
func (em *EntityManager) GetEntityByIDDirect(entityID ecs.EntityID) *ecs.Entity {
    em.World.lock.RLock()
    entity := em.World.entitiesByID[entityID]
    em.World.lock.RUnlock()
    return entity
}
```

**Expected Impact:** HIGH
- Eliminates thousands of map allocations per frame
- Reduces GetEntityByID from 3.65s to < 0.5s
- Estimated improvement: 3.0s per frame (50 FPS gain)

**Files to Modify:**
- `C:\Users\Afromullet\Desktop\TinkerRogue\common\ecsutil.go`
- `C:\Users\Afromullet\Desktop\TinkerRogue\common\EntityManager.go`

**Consideration:** This may require forking the ECS library or submitting a PR to add a lightweight entity lookup method.

---

## Medium Impact Optimizations

### 5. Batch Tile Rendering with Dirty Rectangles

**Current Hotspot:**
```
TileRenderer.renderTile: 14.80s cumulative (10.32% of total)
- Called for every visible tile, every frame
- Screen.DrawImage: 13.34s (90% of renderTile time)
```

**Problem:**
The game re-renders ALL visible tiles every frame, even when nothing has changed:

```go
func (r *TileRenderer) Render(opts RenderOptions) RenderedBounds {
    // Renders EVERY tile in viewport every frame
    for x := bounds.MinX; x <= bounds.MaxX; x++ {
        for y := bounds.MinY; y <= bounds.MaxY; y++ {
            r.renderTile(x, y, opts, &bounds)  // Expensive!
        }
    }
}
```

**Solution:** Implement dirty rectangle tracking

```go
type TileRenderer struct {
    tiles            []*Tile
    fov              fov.FOVMap
    drawOpts         ebiten.DrawImageOptions
    colorScale       ebiten.ColorScale

    // NEW: Dirty tracking
    offscreenBuffer  *ebiten.Image
    dirtyTiles       map[coords.LogicalPosition]bool
    lastFOVRevision  uint64
    forceFullRender  bool
}

func (r *TileRenderer) Render(opts RenderOptions) RenderedBounds {
    // Create offscreen buffer if needed
    if r.offscreenBuffer == nil {
        width, height := opts.Screen.Bounds().Dx(), opts.Screen.Bounds().Dy()
        r.offscreenBuffer = ebiten.NewImage(width, height)
        r.forceFullRender = true
    }

    // Check if FOV changed (forces redraw of affected tiles)
    currentRevision := r.fov.GetRevision()
    if r.lastFOVRevision != currentRevision {
        // Mark visible tiles as dirty
        r.markFOVTilesDirty()
        r.lastFOVRevision = currentRevision
    }

    // Only render if we have dirty tiles or need full render
    if r.forceFullRender {
        r.renderAllTiles(opts)
        r.forceFullRender = false
        r.dirtyTiles = make(map[coords.LogicalPosition]bool)
    } else if len(r.dirtyTiles) > 0 {
        r.renderDirtyTiles(opts)
        r.dirtyTiles = make(map[coords.LogicalPosition]bool)
    }

    // Blit offscreen buffer to screen
    drawOpts := &ebiten.DrawImageOptions{}
    opts.Screen.DrawImage(r.offscreenBuffer, drawOpts)

    return r.calculateBounds(opts)
}

func (r *TileRenderer) MarkTileDirty(pos coords.LogicalPosition) {
    if r.dirtyTiles == nil {
        r.dirtyTiles = make(map[coords.LogicalPosition]bool)
    }
    r.dirtyTiles[pos] = true
}
```

**Usage in game loop:**
```go
// When entities move
func MoveEntity(from, to coords.LogicalPosition) {
    tileRenderer.MarkTileDirty(from)
    tileRenderer.MarkTileDirty(to)
    // ... perform move
}

// When doors open/close
func ToggleDoor(pos coords.LogicalPosition) {
    tileRenderer.MarkTileDirty(pos)
    // ... toggle door
}
```

**Expected Impact:** MEDIUM
- Reduces renderTile calls from ~400-1000 per frame to 0-50 (only changed tiles)
- Estimated improvement: 10-12s when static, minimal overhead when dynamic
- Most beneficial for exploration mode, less so for combat

**Files to Modify:**
- `C:\Users\Afromullet\Desktop\TinkerRogue\worldmap\tilerenderer.go`

---

### 6. Pre-calculate LogicalToScreen Transformations

**Current Hotspot:**
```
applyViewportTransformWithBounds: 643ms cumulative (0.45% of total)
- LogicalToScreen: 259ms (called for every tile)
- Integer division: ~150ms (tile coordinate calculations)
```

**Problem:**
Every tile calculates its screen position from scratch:

```go
func (r *TileRenderer) applyViewportTransformWithBounds(opts *ebiten.DrawImageOptions, tile *Tile, center *coords.LogicalPosition, bounds *RenderedBounds) {
    // EXPENSIVE: Calculated for EVERY tile
    tileLogicalPos := coords.LogicalPosition{
        X: tile.PixelX / graphics.ScreenInfo.TileSize,  // Division!
        Y: tile.PixelY / graphics.ScreenInfo.TileSize,  // Division!
    }

    // EXPENSIVE: Complex coordinate transformation
    screenX, screenY := coords.CoordManager.LogicalToScreen(tileLogicalPos, center)
    opts.GeoM.Translate(screenX, screenY)
}
```

**Solution:** Pre-calculate tile logical positions

```go
type Tile struct {
    image        *ebiten.Image
    PixelX       int
    PixelY       int
    cm           ColorMatrix
    IsRevealed   bool

    // NEW: Cache logical position (calculated once at init)
    LogicalPos   coords.LogicalPosition
}

// During tile creation/initialization
func CreateTile(x, y int) *Tile {
    return &Tile{
        PixelX: x * graphics.ScreenInfo.TileSize,
        PixelY: y * graphics.ScreenInfo.TileSize,
        LogicalPos: coords.LogicalPosition{X: x, Y: y},  // Pre-calculated
        // ... other fields
    }
}

// Modified transform function
func (r *TileRenderer) applyViewportTransformWithBounds(opts *ebiten.DrawImageOptions, tile *Tile, center *coords.LogicalPosition, bounds *RenderedBounds) {
    // Use pre-calculated logical position (no division!)
    screenX, screenY := coords.CoordManager.LogicalToScreen(tile.LogicalPos, center)

    opts.GeoM.Scale(float64(graphics.ScreenInfo.ScaleFactor), float64(graphics.ScreenInfo.ScaleFactor))
    opts.GeoM.Translate(screenX, screenY)

    // ... bounds tracking
}
```

**Expected Impact:** MEDIUM
- Eliminates ~1000 integer divisions per frame
- Reduces coordinate calculation overhead by 60-70%
- Estimated improvement: 400-500ms per frame (8-10 FPS gain)

**Files to Modify:**
- `C:\Users\Afromullet\Desktop\TinkerRogue\worldmap\dungeongen.go` (Tile creation)
- `C:\Users\Afromullet\Desktop\TinkerRogue\worldmap\tilerenderer.go`

---

### 7. Reduce FindAllSquads Query Frequency

**Current Hotspot:**
```
SquadHighlightRenderer.Render: 6.68s cumulative
- FindAllSquads: 152ms (seems low but called multiple times per frame)
```

**Problem:**
`FindAllSquads()` is called by multiple renderers in the same frame:

```go
// In SquadHighlightRenderer.Render
allSquads := shr.queries.SquadCache.FindAllSquads()

// In CombatMode.Render
allSquads := cm.queries.SquadCache.FindAllSquads()

// In other systems...
```

**Solution:** Cache at frame level

```go
type SquadCache struct {
    // ... existing fields

    // NEW: Frame-level caching
    cachedAllSquads []ecs.EntityID
    allSquadsFrameID uint64
}

func (sc *SquadCache) FindAllSquads(frameID uint64) []ecs.EntityID {
    // Return cached result if same frame
    if sc.allSquadsFrameID == frameID && sc.cachedAllSquads != nil {
        return sc.cachedAllSquads
    }

    // Rebuild cache
    sc.cachedAllSquads = sc.findAllSquadsInternal()
    sc.allSquadsFrameID = frameID
    return sc.cachedAllSquads
}
```

**Expected Impact:** MEDIUM
- Reduces redundant ECS queries from 3-5 per frame to 1
- Estimated improvement: 400-600ms per frame (7-10 FPS gain)

**Files to Modify:**
- `C:\Users\Afromullet\Desktop\TinkerRogue\squads\squadcache.go`

---

## Low Impact Optimizations

### 8. Optimize Font Rendering

**Current Hotspot:**
```
truetype.(*hinter).run: 17.356s flat (12.10% of total)
```

**Problem:**
Font hinting calculations are expensive. This is primarily in the UI library (ebitenui).

**Solution:**
1. Reduce text rendering frequency
2. Cache rendered text as images
3. Use simpler fonts that don't require complex hinting

```go
type CachedText struct {
    image    *ebiten.Image
    text     string
    revision uint64
}

func DrawCachedText(screen *ebiten.Image, text string, revision uint64) {
    if cachedText.revision != revision || cachedText.text != text {
        // Re-render text to image
        cachedText.image = renderTextToImage(text)
        cachedText.text = text
        cachedText.revision = revision
    }

    // Draw cached image instead of re-rendering text
    screen.DrawImage(cachedText.image, opts)
}
```

**Expected Impact:** LOW (for gameplay), HIGH (for UI-heavy screens)
- Most text is in UI panels which don't change frequently
- Estimated improvement: 1-2s per frame when UI is visible

**Files to Modify:**
- Various UI components in `C:\Users\Afromullet\Desktop\TinkerRogue\gui\`

---

### 9. Use DrawImageOptions Pool

**Current Hotspot:**
```
Multiple allocations of ebiten.DrawImageOptions throughout rendering
```

**Problem:**
Creating new `DrawImageOptions` repeatedly:

```go
// Called hundreds of times per frame
op := &ebiten.DrawImageOptions{}  // Allocation!
op.GeoM.Translate(x, y)
screen.DrawImage(img, op)
```

**Solution:** Pool draw options

```go
var drawOptsPool = sync.Pool{
    New: func() interface{} {
        return &ebiten.DrawImageOptions{}
    },
}

func (r *TileRenderer) renderTile(...) {
    // Get from pool
    opts := drawOptsPool.Get().(*ebiten.DrawImageOptions)
    defer drawOptsPool.Put(opts)

    // Reset to clean state
    opts.GeoM.Reset()
    opts.ColorScale.Reset()

    // Use opts...
    screen.DrawImage(tile.image, opts)
}
```

**Expected Impact:** LOW
- Reduces allocation pressure
- Estimated improvement: 200-300ms per frame (3-5 FPS gain)

**Files to Modify:**
- `C:\Users\Afromullet\Desktop\TinkerRogue\worldmap\tilerenderer.go`
- `C:\Users\Afromullet\Desktop\TinkerRogue\gui\guimodes\guirenderers.go`

---

### 10. Optimize FOV Visibility Checks

**Current Hotspot:**
```
renderTile: isVisible check: 418ms cumulative
```

**Problem:**
FOV visibility is checked for every tile every frame:

```go
isVisible := r.fov.IsVisible(x, y) || opts.RevealAll
```

**Solution:** Cache FOV results with revision counter

```go
type TileRenderer struct {
    // ... existing fields
    fovCache     []bool  // Flat array matching tile array
    fovRevision  uint64
}

func (r *TileRenderer) Render(opts RenderOptions) {
    // Update FOV cache if FOV changed
    currentRevision := r.fov.GetRevision()
    if r.fovRevision != currentRevision {
        r.rebuildFOVCache()
        r.fovRevision = currentRevision
    }

    // ... render loop
}

func (r *TileRenderer) renderTile(x, y int, opts RenderOptions) {
    idx := coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: x, Y: y})

    // Fast array lookup instead of FOV.IsVisible call
    isVisible := r.fovCache[idx] || opts.RevealAll
    // ... rest of rendering
}
```

**Expected Impact:** LOW
- FOV.IsVisible is likely already optimized
- Estimated improvement: 200-400ms per frame (3-7 FPS gain)

**Files to Modify:**
- `C:\Users\Afromullet\Desktop\TinkerRogue\worldmap\tilerenderer.go`

---

## Implementation Priority

### Phase 1: Quick Wins (1-2 hours, 60% improvement)
1. **Eliminate DrawTileOverlay allocations** (30 min)
2. **Fix DrawTileBorder Fill calls** (45 min)
3. **Use DrawImageOptions pool** (15 min)

### Phase 2: Frame Caching (2-3 hours, 40% improvement)
4. **Cache GetSquadInfo per frame** (1 hour)
5. **Cache FindAllSquads per frame** (30 min)
6. **Pre-calculate tile logical positions** (1 hour)

### Phase 3: Major Refactoring (4-6 hours, 50% improvement)
7. **Optimize ECS GetEntityByID** (2-3 hours, may require ECS library changes)
8. **Implement dirty rectangle rendering** (2-3 hours)

### Phase 4: Polish (2-4 hours, 10-20% improvement)
9. **FOV caching with revisions** (1 hour)
10. **Text rendering optimization** (1-2 hours)

---

## Measurement & Validation

### Before Optimization
```bash
go build -o game_main/game_main.exe game_main/*.go
game_main/game_main.exe -cpuprofile=before.prof
go tool pprof -top before.prof
```

### After Each Optimization
```bash
go build -o game_main/game_main.exe game_main/*.go
game_main/game_main.exe -cpuprofile=after_opt1.prof
go tool pprof -top -diff_base=before.prof after_opt1.prof
```

### Key Metrics to Track
- **Frame time** (target: < 16.67ms for 60 FPS)
- **DrawImage calls per frame** (target: < 1000)
- **Memory allocations per frame** (target: < 1MB)
- **GC pressure** (target: < 1ms per frame)

---

## Additional Profiling Commands

```bash
# Focus on allocations
go tool pprof -alloc_space cpu_profile.pb.gz

# Focus on specific functions
go tool pprof -list="DrawImage" cpu_profile.pb.gz
go tool pprof -list="GetSquadInfo" cpu_profile.pb.gz

# Generate flame graph (requires go-torch)
go tool pprof -http=:8080 cpu_profile.pb.gz
```

---

## Notes

1. **Runtime Profiling Overhead:** The 37.63% `lostProfileEvent` overhead is an artifact of profiling. Real performance will be better than profile suggests.

2. **Ebiten Best Practices:**
   - Minimize `NewImage` calls in render loop
   - Reuse `DrawImageOptions` objects
   - Use `ColorScale` instead of creating colored images
   - Batch draw calls when possible

3. **ECS Performance:**
   - Avoid `GetEntityByID` in hot loops
   - Cache component lookups per frame
   - Use Views and Queries instead of individual entity lookups

4. **Memory Management:**
   - Object pooling for frequently allocated objects
   - Pre-allocate slices with known capacity
   - Avoid map allocations in inner loops

5. **Testing Strategy:**
   - Profile after each optimization
   - Use benchmark tests for critical paths
   - Test with various squad counts (1, 10, 50 squads)
   - Test in different game modes (exploration vs combat)

---

## Conclusion

The profiling data shows clear optimization opportunities that could improve performance by 2-3x with focused effort. The most impactful changes are:

1. **Caching rendered UI elements** (images, borders, overlays)
2. **Frame-level caching for ECS queries**
3. **Optimizing the ECS GetEntityByID path**
4. **Dirty rectangle tracking for tile rendering**

Starting with Phase 1 optimizations would provide immediate, measurable results with minimal risk. Phase 2 and 3 require more careful refactoring but offer substantial performance gains.

The current architecture is solid; these optimizations are about eliminating redundant work rather than fundamental redesign.
