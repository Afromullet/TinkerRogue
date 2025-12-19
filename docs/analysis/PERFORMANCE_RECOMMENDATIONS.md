# Performance Optimization Recommendations

**Generated:** 2025-12-19
**Benchmark Source:** benchmark_baseline.pb.gz
**Total Profile Duration:** 60.14s (55.67s samples)

---

## Executive Summary

Performance analysis reveals several optimization opportunities beyond text rendering. Key areas for improvement include memory allocation patterns, tile batching, ECS component access, and rendering workflows.

**Priority Legend:**
- **P0** (Critical): >1s cumulative impact, implement immediately
- **P1** (High): 500ms-1s cumulative impact, implement soon
- **P2** (Medium): 100-500ms cumulative impact, implement when convenient
- **P3** (Low): <100ms cumulative impact, consider for polish phase

---

## P0 - Critical Optimizations (>1s Impact)

### 1. Pre-allocate TileBatch Pool (1.5s+ total impact)

**Current Cost:**
- `NewTileBatch` allocation: 326ms
- `TileBatch.AddTile` append operations: 399ms (245ms vertices + 154ms indices)
- Total: **~725ms per render cycle** when viewport changes

**Problem:**
```go
// Creating new batches every viewport change
if r.batches[tile.image] == nil {
    r.batches[tile.image] = NewTileBatch(tile.image)  // 326ms
}

// Appending to slices with reallocation
tb.vertices = append(tb.vertices, /* 4 vertices */)  // 245ms
tb.indices = append(tb.indices, /* 6 indices */)     // 154ms
```

**Solution:**
Pre-allocate a fixed pool of TileBatch objects and reuse them.

```go
type TileRenderer struct {
    tiles   []*Tile
    batches map[*ebiten.Image]*TileBatch

    // Add batch pool
    batchPool     []*TileBatch
    batchPoolSize int
}

func NewTileRenderer(tiles []*Tile) *TileRenderer {
    const maxUniqueTileImages = 50  // Adjust based on your tileset

    // Pre-allocate batch pool
    pool := make([]*TileBatch, 0, maxUniqueTileImages)
    for i := 0; i < maxUniqueTileImages; i++ {
        pool = append(pool, &TileBatch{
            vertices: make([]ebiten.Vertex, 0, 1024),  // ~256 tiles
            indices:  make([]uint16, 0, 1536),         // ~256 tiles
        })
    }

    return &TileRenderer{
        tiles:         tiles,
        batches:       make(map[*ebiten.Image]*TileBatch, maxUniqueTileImages),
        batchPool:     pool,
        batchPoolSize: maxUniqueTileImages,
    }
}

func (r *TileRenderer) getBatchForImage(img *ebiten.Image) *TileBatch {
    if batch, exists := r.batches[img]; exists {
        return batch
    }

    // Get batch from pool instead of allocating
    if len(r.batchPool) > 0 {
        batch := r.batchPool[len(r.batchPool)-1]
        r.batchPool = r.batchPool[:len(r.batchPool)-1]
        batch.image = img
        r.batches[img] = batch
        return batch
    }

    // Fallback: create new batch if pool exhausted
    return NewTileBatch(img)
}

func (r *TileRenderer) InvalidateCache() {
    // Return batches to pool
    for img, batch := range r.batches {
        batch.Reset()
        batch.image = nil
        if len(r.batchPool) < r.batchPoolSize {
            r.batchPool = append(r.batchPool, batch)
        }
        delete(r.batches, img)
    }
    r.batchesBuilt = false
}
```

**Expected Impact:** 600-700ms reduction per viewport change
**Implementation Time:** 30 minutes
**Files:** `worldmap/tilerenderer.go`, `worldmap/tilebatch.go`

---

### 2. Cache Viewport Position Calculations (229ms impact)

**Current Cost:** `calculateViewportPosition`: 229ms cumulative

**Problem:**
```go
// Called for every tile, every frame when viewport changes
screenX, screenY = r.calculateViewportPosition(tile, opts.CenterOn, bounds)

// Inside calculateViewportPosition:
tileLogicalPos := coords.LogicalPosition{
    X: tile.PixelX / graphics.ScreenInfo.TileSize,  // Division every tile
    Y: tile.PixelY / graphics.ScreenInfo.TileSize,
}
screenX, screenY := coords.CoordManager.LogicalToScreen(logicalPos, playerPos)
```

**Solution:**
Cache the center offset calculation and reuse it for all tiles.

```go
type TileRenderer struct {
    // ... existing fields ...

    // Cache viewport transformation
    viewportOffsetX float64
    viewportOffsetY float64
    viewportScale   float32
    lastCenter      *coords.LogicalPosition
}

func (r *TileRenderer) updateViewportTransform(center *coords.LogicalPosition) {
    if center == nil {
        r.viewportOffsetX, r.viewportOffsetY = 0, 0
        r.viewportScale = 1.0
        return
    }

    // Calculate offset once for the entire viewport
    centerScreen := coords.LogicalPosition{X: center.X, Y: center.Y}
    originX, originY := coords.CoordManager.LogicalToScreen(centerScreen, center)

    r.viewportOffsetX = -originX + float64(graphics.ScreenInfo.ScreenWidth/2)
    r.viewportOffsetY = -originY + float64(graphics.ScreenInfo.ScreenHeight/2)
    r.viewportScale = float32(graphics.ScreenInfo.ScaleFactor)
    r.lastCenter = center
}

func (r *TileRenderer) calculateViewportPositionFast(tile *Tile) (float32, float32) {
    // Simple offset application instead of full coordinate transform
    screenX := float64(tile.PixelX)*float64(r.viewportScale) + r.viewportOffsetX
    screenY := float64(tile.PixelY)*float64(r.viewportScale) + r.viewportOffsetY
    return float32(screenX), float32(screenY)
}

// In Render(), call once before tile loop:
if needsRebuild && opts.CenterOn != nil {
    r.updateViewportTransform(opts.CenterOn)
}

// Then in addTileToBatch:
if opts.CenterOn != nil {
    screenX, screenY = r.calculateViewportPositionFast(tile)
} else {
    screenX = float32(tile.PixelX)
    screenY = float32(tile.PixelY)
}
```

**Expected Impact:** 150-200ms reduction
**Implementation Time:** 20 minutes
**Files:** `worldmap/tilerenderer.go:152-179`

---

### 3. Optimize ProcessRenderablesInSquare Component Access (156ms impact)

**Current Cost:**
- `GetComponentType` calls: 156ms (96ms + 60ms from calls)
- `screen.DrawImage` calls: 362ms

**Problem:**
```go
for _, result := range cache.RenderablesView.Get() {
    pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
    renderable := common.GetComponentType[*Renderable](result.Entity, RenderableComponent)
    // ... bounds check ...
    screen.DrawImage(img, op)  // Individual draw call per entity
}
```

**Solution A - Component Access:**
Access components directly from query result to avoid map lookups.

```go
// In rendering/components.go, add a query result type:
type RenderableQueryResult struct {
    EntityID   ecs.EntityID
    Position   *coords.LogicalPosition
    Renderable *Renderable
}

// Modify RenderingCache to store component data directly
type RenderingCache struct {
    RenderablesView *ecs.View
    // Add cached query results
    cachedRenderables []RenderableQueryResult
    cacheValid        bool
}

func (rc *RenderingCache) RefreshCache(manager *common.EntityManager) {
    rc.cachedRenderables = rc.cachedRenderables[:0]  // Reuse slice

    for _, result := range rc.RenderablesView.Get() {
        pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
        renderable := common.GetComponentType[*Renderable](result.Entity, RenderableComponent)

        if renderable.Visible {
            rc.cachedRenderables = append(rc.cachedRenderables, RenderableQueryResult{
                EntityID:   result.Entity.GetID(),
                Position:   pos,
                Renderable: renderable,
            })
        }
    }
    rc.cacheValid = true
}

func ProcessRenderablesInSquare(...) {
    // Refresh cache only when entities change
    if !cache.cacheValid {
        cache.RefreshCache(ecsmanager)
    }

    sq := coords.NewDrawableSection(playerPos.X, playerPos.Y, squareSize)

    for _, data := range cache.cachedRenderables {
        if data.Position.X >= sq.StartX && data.Position.X <= sq.EndX &&
           data.Position.Y >= sq.StartY && data.Position.Y <= sq.EndY {

            logicalPos := coords.LogicalPosition{X: data.Position.X, Y: data.Position.Y}
            op := &ebiten.DrawImageOptions{}
            op.GeoM.Scale(float64(graphics.ScreenInfo.ScaleFactor), float64(graphics.ScreenInfo.ScaleFactor))
            screenX, screenY := coords.CoordManager.LogicalToScreen(logicalPos, playerPos)
            op.GeoM.Translate(screenX, screenY)
            screen.DrawImage(data.Renderable.Image, op)
        }
    }
}
```

**Expected Impact:** 100-120ms reduction
**Implementation Time:** 45 minutes
**Files:** `rendering/rendering.go:50-79`, `rendering/components.go`

---

## P1 - High Priority Optimizations (500ms-1s Impact)

### 4. Batch Entity Rendering (362ms impact)

**Current Cost:** Individual `screen.DrawImage` calls: 362ms

**Problem:**
Each entity renders separately, causing many draw calls.

**Solution:**
Implement sprite batching similar to tile batching.

```go
type SpriteBatch struct {
    vertices []ebiten.Vertex
    indices  []uint16
    image    *ebiten.Image
}

type RenderingCache struct {
    RenderablesView *ecs.View
    // Add sprite batches
    spriteBatches map[*ebiten.Image]*SpriteBatch
}

func ProcessRenderablesInSquare(...) {
    // Clear existing batches
    for _, batch := range cache.spriteBatches {
        batch.vertices = batch.vertices[:0]
        batch.indices = batch.indices[:0]
    }

    sq := coords.NewDrawableSection(playerPos.X, playerPos.Y, squareSize)

    // Collect sprites into batches by image
    for _, result := range cache.RenderablesView.Get() {
        pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
        renderable := common.GetComponentType[*Renderable](result.Entity, RenderableComponent)

        if !renderable.Visible {
            continue
        }

        if pos.X >= sq.StartX && pos.X <= sq.EndX && pos.Y >= sq.StartY && pos.Y <= sq.EndY {
            logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}
            screenX, screenY := coords.CoordManager.LogicalToScreen(logicalPos, playerPos)

            // Add to batch instead of drawing individually
            addToSpriteBatch(cache, renderable.Image, screenX, screenY)
        }
    }

    // Draw all batches
    for _, batch := range cache.spriteBatches {
        if len(batch.vertices) > 0 {
            screen.DrawTriangles(batch.vertices, batch.indices, batch.image, nil)
        }
    }
}
```

**Expected Impact:** 250-300ms reduction
**Implementation Time:** 1 hour
**Files:** `rendering/rendering.go`, new `rendering/spritebatch.go`

---

### 5. Reduce TileBatch.Draw Overhead (1.04s impact)

**Current Cost:** `screen.DrawTriangles` in `TileBatch.Draw`: 1.04s

**Problem:**
While batching is working, we're still seeing high cost in DrawTriangles.

**Investigation Needed:**
1. Are we creating too many batches? (Check number of unique tile images)
2. Are batches too large? (Check vertex/index counts)
3. Can we merge batches further?

**Diagnostic Code:**
```go
func (r *TileRenderer) Render(opts RenderOptions) RenderedBounds {
    // ... existing code ...

    // Add diagnostic logging (remove in production)
    if opts.CenterOn != nil {
        totalBatches := 0
        totalVertices := 0
        for _, batch := range r.batches {
            if len(batch.vertices) > 0 {
                totalBatches++
                totalVertices += len(batch.vertices)
            }
        }
        fmt.Printf("Render stats: %d batches, %d total vertices\n", totalBatches, totalVertices)
    }

    // ... rest of render ...
}
```

**Expected Impact:** 300-400ms reduction (if batches can be optimized further)
**Implementation Time:** Investigation: 15 minutes, Fix: 30-60 minutes
**Files:** `worldmap/tilerenderer.go`, `worldmap/tilebatch.go`

---

## P2 - Medium Priority Optimizations (100-500ms Impact)

### 6. Optimize Bounds Checking in Tile Rendering (26ms impact)

**Current Cost:** `inMapBounds` checks: 26ms

**Problem:**
```go
for x := bounds.MinX; x <= bounds.MaxX; x++ {
    for y := bounds.MinY; y <= bounds.MaxY; y++ {
        if !r.inMapBounds(x, y) {  // Redundant if bounds are already clamped
            continue
        }
        r.addTileToBatch(x, y, opts, &bounds)
    }
}
```

**Solution:**
Clamp bounds in `calculateBounds` to avoid per-tile checks.

```go
func (r *TileRenderer) calculateBounds(opts RenderOptions) RenderedBounds {
    var bounds RenderedBounds

    if opts.CenterOn != nil {
        sq := coords.NewDrawableSection(opts.CenterOn.X, opts.CenterOn.Y, opts.ViewportSize)
        bounds = RenderedBounds{
            MinX: sq.StartX,
            MaxX: sq.EndX,
            MinY: sq.StartY,
            MaxY: sq.EndY,
        }
    } else {
        bounds = RenderedBounds{
            MinX: 0,
            MaxX: graphics.ScreenInfo.DungeonWidth - 1,
            MinY: 0,
            MaxY: graphics.ScreenInfo.DungeonHeight - 1,
        }
    }

    // Clamp to map bounds once
    bounds.MinX = max(0, bounds.MinX)
    bounds.MaxX = min(graphics.ScreenInfo.DungeonWidth-1, bounds.MaxX)
    bounds.MinY = max(0, bounds.MinY)
    bounds.MaxY = min(graphics.ScreenInfo.DungeonHeight-1, bounds.MaxY)

    return bounds
}

// Remove inMapBounds check from render loop
for x := bounds.MinX; x <= bounds.MaxX; x++ {
    for y := bounds.MinY; y <= bounds.MaxY; y++ {
        r.addTileToBatch(x, y, opts, &bounds)  // No bounds check needed
    }
}
```

**Expected Impact:** 20-25ms reduction
**Implementation Time:** 10 minutes
**Files:** `worldmap/tilerenderer.go:252-275`

---

### 7. Optimize Tile Image Bounds Access (20ms impact)

**Current Cost:** `tile.image.Bounds()` calls: 46ms total (13ms + 31ms + 2ms)

**Problem:**
Calling `Bounds()` for every tile when all tiles likely have the same dimensions.

**Solution:**
Cache tile dimensions if all tiles are the same size.

```go
type TileRenderer struct {
    tiles   []*Tile
    batches map[*ebiten.Image]*TileBatch

    // Cache common tile dimensions
    cachedTileWidth  int
    cachedTileHeight int
    uniformTileSize  bool  // True if all tiles are same size
}

func NewTileRenderer(tiles []*Tile) *TileRenderer {
    r := &TileRenderer{
        tiles:   tiles,
        batches: make(map[*ebiten.Image]*TileBatch, 20),
    }

    // Check if tiles are uniform size (common case)
    if len(tiles) > 0 && tiles[0].image != nil {
        firstBounds := tiles[0].image.Bounds()
        r.cachedTileWidth = firstBounds.Dx()
        r.cachedTileHeight = firstBounds.Dy()
        r.uniformTileSize = true

        // Verify all tiles match
        for _, tile := range tiles {
            if tile.image != nil {
                bounds := tile.image.Bounds()
                if bounds.Dx() != r.cachedTileWidth || bounds.Dy() != r.cachedTileHeight {
                    r.uniformTileSize = false
                    break
                }
            }
        }
    }

    return r
}

func (r *TileRenderer) addTileToBatch(x, y int, opts RenderOptions, bounds *RenderedBounds) {
    // ... existing code ...

    // Get tile dimensions efficiently
    var tileW, tileH float32
    if r.uniformTileSize {
        tileW = float32(r.cachedTileWidth)
        tileH = float32(r.cachedTileHeight)
    } else {
        tileBounds := tile.image.Bounds()
        tileW = float32(tileBounds.Dx())
        tileH = float32(tileBounds.Dy())
    }

    // ... rest of function ...
}
```

**Expected Impact:** 35-40ms reduction
**Implementation Time:** 15 minutes
**Files:** `worldmap/tilerenderer.go`

---

### 8. Investigate GUI Widget Rendering (12.5s cumulative)

**Current Cost:**
- `ebitenui/widget.(*Container).Render`: 12.5s cumulative (22.46%)
- `ebitenui/widget.(*Text).draw`: 8.15s (14.64%)
- `ebitenui/widget.(*ScrollContainer).Render`: 7.5s (13.48%)

**Problem:**
GUI widgets consume significant time, but much of this is text rendering (excluded from optimization scope).

**Actionable Optimizations:**

1. **Reduce Layout Recalculations:**
```go
// In gui modes, cache layout results
type BaseMode struct {
    // ... existing fields ...
    layoutCache      map[string]widget.PreferredSize
    layoutCacheValid bool
}

func (bm *BaseMode) InvalidateLayout() {
    bm.layoutCacheValid = false
}
```

2. **Minimize Container Hierarchy:**
Audit container nesting in GUI modes. Each container level adds overhead.

3. **Lazy Widget Updates:**
Only update widgets that changed, not entire UI every frame.

**Expected Impact:** 200-400ms reduction
**Implementation Time:** 2-3 hours (requires UI audit)
**Files:** `gui/basemode.go`, various mode files

---

## P3 - Low Priority Optimizations (<100ms Impact)

### 9. Reduce GetComponentType Overhead (154ms total)

**Current Cost:** `common.GetComponentType`: 154ms cumulative

**Problem:**
Frequent component access with defer overhead.

**Solution:**
Create inline accessor for hot paths.

```go
// Add to common/ecsutil.go
func GetComponentTypeFast[T any](entity *ecs.Entity, component *ecs.Component) T {
    // No defer, no error handling - use only in verified hot paths
    if c, ok := entity.GetComponentData(component); ok {
        return c.(T)
    }
    var nilValue T
    return nilValue
}
```

**Expected Impact:** 50-80ms reduction
**Implementation Time:** 30 minutes (requires identifying safe usage sites)
**Files:** `common/ecsutil.go`, call sites in hot paths

---

### 10. Optimize LogicalToIndex Calls (11ms impact)

**Current Cost:** `coords.CoordManager.LogicalToIndex`: 11ms

**Solution:**
Inline the calculation in hot paths.

```go
// Instead of:
idx := coords.CoordManager.LogicalToIndex(logicalPos)

// Use inline:
idx := logicalPos.Y * graphics.ScreenInfo.DungeonWidth + logicalPos.X
```

**Note:** Only use inline version when you're certain DungeonWidth is correct. The CoordManager exists to prevent index calculation bugs.

**Expected Impact:** 8-10ms reduction
**Implementation Time:** 15 minutes
**Files:** `worldmap/tilerenderer.go:102`

---

## Implementation Roadmap

### Week 1 - Critical Wins (P0)
1. **Day 1-2:** Implement TileBatch pooling (#1) - 600-700ms gain
2. **Day 3:** Cache viewport calculations (#2) - 150-200ms gain
3. **Day 4:** Optimize ProcessRenderablesInSquare (#3) - 100-120ms gain

**Expected Total Gain:** 850-1020ms (~15-18% performance improvement)

### Week 2 - High Priority (P1)
4. **Day 1-2:** Implement sprite batching (#4) - 250-300ms gain
5. **Day 3:** Investigate/optimize DrawTriangles (#5) - 300-400ms gain (potential)

**Expected Total Gain:** 550-700ms (~10-12% additional improvement)

### Week 3+ - Polish (P2 + P3)
6. Bounds checking optimization (#6) - 20-25ms
7. Tile bounds caching (#7) - 35-40ms
8. GUI widget optimization (#8) - 200-400ms
9. GetComponentType optimization (#9) - 50-80ms
10. LogicalToIndex inlining (#10) - 8-10ms

**Expected Total Gain:** 313-555ms (~5-9% additional improvement)

---

## Measurement & Validation

After each optimization:

1. **Run Benchmark:**
```bash
go test -bench=. -benchmem -cpuprofile=profile.pb.gz ./...
go tool pprof -top -cum profile.pb.gz
```

2. **Compare Results:**
```bash
# Before and after comparison
go tool pprof -base=benchmark_baseline.pb.gz -top profile.pb.gz
```

3. **Verify Correctness:**
   - Visual inspection (no rendering artifacts)
   - Run existing tests
   - Play through combat scenarios

---

## Long-Term Recommendations

### 1. Implement Frame Budget System
Track frame time and disable expensive features if budget exceeded:
```go
const targetFrameTime = 16.67 * time.Millisecond  // 60 FPS

type FrameBudget struct {
    startTime    time.Time
    allowBatching bool
}

func (fb *FrameBudget) Start() {
    fb.startTime = time.Now()
}

func (fb *FrameBudget) Remaining() time.Duration {
    return targetFrameTime - time.Since(fb.startTime)
}
```

### 2. Profile Different Game States
Current profile may be from a specific game state. Profile:
- Combat with many units
- Overworld with many renderables
- UI-heavy screens (squad management)
- Map generation

### 3. Consider Entity Count Limits
If entity count grows unbounded, implement culling:
- Spatial partitioning for renderables
- Only process entities in viewport + margin
- Lazy update for off-screen entities

---

## Notes

- Text rendering (14.64% + additional) excluded per requirements (Ebiten limitation)
- Runtime profiling overhead (14.78% lostProfileEvent) is not real game cost
- CGO calls (35.23%) are mostly Ebiten/DirectX internals - limited optimization opportunity
- Focus optimizations on game code (worldmap, rendering, ECS) where we have control

**Total Realistic Optimization Potential:** 1.7-2.3 seconds (~30-40% improvement)
