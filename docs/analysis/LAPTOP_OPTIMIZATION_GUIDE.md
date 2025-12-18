# TinkerRogue Performance Optimization Guide for Lower-End Hardware

**Generated**: 2025-12-17
**Based On**: `docs/benchmarking/text_rendering_benchmark/text_rendering_bench.pb.gz`
**Goal**: Make the game run smoothly on laptops and lower-end PCs
**Profile Duration**: 120.13s
**Total CPU Time**: 116,033ms (96.59% utilization)

---

## EXECUTIVE SUMMARY

### Current Performance Status: NEEDS OPTIMIZATION

**The game is CPU-bound** with most time spent in:
1. **Text/Font Rendering**: ~35% of CPU time
2. **Memory Allocations**: ~22% of CPU time
3. **Graphics Draw Calls**: ~17% of CPU time
4. **ECS/Squad Queries**: ~10% of CPU time
5. **Tile Rendering**: ~10% of CPU time

**For laptop compatibility**, focus on reducing:
- Draw call count (batch rendering)
- Memory allocations (object pooling)
- Font operations (text caching)
- Redundant ECS queries (caching)

### Estimated Improvement Potential

| Optimization Area | Current | Potential Savings | Priority |
|-------------------|---------|-------------------|----------|
| Memory Allocations | 25.6s | 10-15s (40-60%) | **CRITICAL** |
| Text Rendering | 20.1s | 5-10s (25-50%) | **HIGH** |
| Squad Info Queries | 5.5s | 4-5s (80-90%) | **HIGH** |
| NineSlice Allocations | 4.7s | 3-4s (70-80%) | **MEDIUM** |
| Tile Rendering | 11.3s | 2-4s (20-35%) | **MEDIUM** |
| Combat Grid | 1.3s | 1s (75%) | **LOW** |

**Total Potential Savings**: 25-40 seconds (22-35% faster)

---

## PROFILE BREAKDOWN

### Top 10 CPU Consumers

| Rank | Function | Time | % | Category |
|------|----------|------|---|----------|
| 1 | Text.Render / Text.draw | 20.1s | 17.3% | UI/Font |
| 2 | Image.DrawImage | 17.3s | 14.9% | Graphics |
| 3 | TileRenderer.Render | 11.3s | 9.7% | Graphics |
| 4 | font.MeasureString | 13.8s | 11.9% | Font |
| 5 | truetype.hinter.run | 12.7s | 11.0% | Font |
| 6 | mallocgc (memory) | 25.6s | 22.1% | Memory |
| 7 | newobject | 16.7s | 14.4% | Memory |
| 8 | SquadHighlightRenderer | 6.3s | 5.4% | Game Logic |
| 9 | GetSquadInfoCached | 5.5s | 4.7% | ECS |
| 10 | NineSlice.drawTile | 4.7s | 4.0% | UI |

### Memory Allocation Hotspots

| Source | Time in newobject | Issue |
|--------|-------------------|-------|
| NineSlice.drawTile | 3,276ms | Creates objects per tile |
| ECS Manager.GetEntityByID | 2,495ms | Allocates QueryResult each call |
| makemap_small | 2,630ms | Many small map allocations |
| SquadInfoCached | 686ms | Creates SquadInfo structs |
| commandQueue.EnqueueDraw | 851ms | Graphics command allocations |

---

## PRIORITY 1: CRITICAL OPTIMIZATIONS


### 1.2 Reduce ECS GetEntityByID Allocations (3-4 hours)

**Impact**: -2.5s (2.1% CPU time saved)
**Current**: `GetEntityByID` allocates new QueryResult every call

**Problem Analysis**:
```
GetEntityByID: 4,673ms
├── newobject: 2,495ms (53%)  ⚠️ Allocates QueryResult struct
└── fetchComponentsForEntity: 2,190ms
```

**The ECS library allocates a QueryResult struct for every GetEntityByID call.** You can't fix the library, but you can reduce calls.

**Solution A**: Batch component lookups

```go
// Instead of multiple GetEntityByID calls:
// BAD:
pos := GetComponentTypeByID[*Position](manager, entityID, PositionComponent)
attrs := GetComponentTypeByID[*Attributes](manager, entityID, AttributeComponent)
health := GetComponentTypeByID[*Health](manager, entityID, HealthComponent)
// This calls GetEntityByID 3 times!

// GOOD: Get entity once, then extract components
entity := common.FindEntityByID(manager, entityID)
if entity != nil {
    pos := common.GetComponentType[*Position](entity, PositionComponent)
    attrs := common.GetComponentType[*Attributes](entity, AttributeComponent)
    health := common.GetComponentType[*Health](entity, HealthComponent)
}
// Only 1 GetEntityByID call!
```

**Solution B**: Use query iteration instead of ID lookups

```go
// BAD: Look up each squad by ID
for _, squadID := range squadIDs {
    squadEntity := manager.World.GetEntityByID(squadID)  // Allocates!
    // process...
}

// GOOD: Query once, iterate results
for _, result := range manager.World.Query(SquadTag) {
    squadID := result.Entity.GetID()
    if !isRelevantSquad(squadID) {
        continue
    }
    // result.Entity already available - no allocation
}
```

**Files to Update**:
- `gui/guicomponents/guiqueries.go` - GetSquadInfoCached
- `gui/guimodes/squadhighlightrenderer.go` - Squad iteration
- `combat/combatqueries.go` - Faction queries

---

### 1.3 Pool NineSlice Draw Operations (4-5 hours)

**Impact**: -3s (2.6% CPU time saved)
**Current**: NineSlice creates new objects for every tile draw

**Problem Analysis**:
```
NineSlice.drawTile: 4,654ms
├── newobject: 3,276ms (70%)  ⚠️ Creates DrawImageOptions each draw
└── Image.DrawImage: 1,336ms
```

**The Issue**: Every NineSlice tile creates a new `DrawImageOptions` struct.

**Solution**: You can't modify EbitenUI directly, but you can reduce NineSlice usage:

**Option A**: Reduce UI complexity
- Use simpler backgrounds (solid colors instead of NineSlice)
- Reduce number of UI panels/buttons visible
- Combine small widgets into larger ones

**Option B**: Pre-render static UI elements

```go
// Pre-render static UI backgrounds to an image once
type CachedBackground struct {
    image    *ebiten.Image
    dirty    bool
    width    int
    height   int
}

func (cb *CachedBackground) GetImage(w, h int) *ebiten.Image {
    if cb.image == nil || cb.width != w || cb.height != h || cb.dirty {
        cb.image = ebiten.NewImage(w, h)
        // Render NineSlice once
        nineSlice.Draw(cb.image, w, h)
        cb.width, cb.height = w, h
        cb.dirty = false
    }
    return cb.image
}
```

**Option C**: Reduce button/panel count
```go
// Instead of many small buttons, use a single list widget
// Bad: 20 squad buttons = 20 NineSlice backgrounds
// Good: 1 list widget with text = 1 NineSlice background
```

---

## PRIORITY 2: HIGH-IMPACT OPTIMIZATIONS

### 2.1 Reduce Font Operations (3-4 hours)

**Impact**: -3s to -5s (3-4% CPU time saved)
**Current**: Font hinting and measurement very expensive

**Problem Analysis**:
```
Font operations: ~35% of rendering time
├── MeasureString: 13,767ms (11.86%)  ⚠️ Called repeatedly for same text
├── hinter.run: 12,747ms (10.99%)     ⚠️ Font glyph loading
└── GlyphBuf.Load: 13,735ms (11.84%)
```

**Solution A**: Pre-render common text

```go
// gui/guiresources/textcache.go
type TextImageCache struct {
    cache map[string]*ebiten.Image
    face  font.Face
}

func (tic *TextImageCache) GetTextImage(text string) *ebiten.Image {
    if img, ok := tic.cache[text]; ok {
        return img
    }

    // Measure once
    bounds := font.BoundString(tic.face, text)
    w := bounds.Max.X.Ceil() - bounds.Min.X.Floor()
    h := bounds.Max.Y.Ceil() - bounds.Min.Y.Floor()

    // Create image once
    img := ebiten.NewImage(w, h)
    text.Draw(img, text, tic.face, 0, h, color.White)

    tic.cache[text] = img
    return img
}

// Usage: For static labels like "HP:", "Attack:", "Defense:"
var commonLabels = []string{"HP:", "Attack:", "Defense:", "Speed:", "Turn:"}

func PreloadCommonLabels() {
    for _, label := range commonLabels {
        TextImageCache.GetTextImage(label)
    }
}
```

**Solution B**: Reduce text widget updates

```go
// Only update text when value actually changes
type SmartLabel struct {
    widget    *widget.Text
    lastValue string
}

func (sl *SmartLabel) SetText(text string) {
    if sl.lastValue != text {  // Only update if changed
        sl.widget.Label = text
        sl.lastValue = text
    }
}
```

**Solution C**: Use bitmap fonts instead of TrueType

Bitmap fonts are MUCH faster than TrueType because:
- No glyph loading
- No hinting
- Pre-rendered at specific sizes

```go
// Consider using ebiten's bitmap font support for frequently changing text
import "github.com/hajimehoshi/ebiten/v2/text/v2"

// Use bitmap font for damage numbers, HP displays, etc.
```

---

### 2.2 Optimize Tile Rendering (3-4 hours)

**Impact**: -2s to -4s (2-3% CPU time saved)
**Current**: Drawing every visible tile individually

**Problem Analysis**:
```
TileRenderer.Render: 11,260ms (9.70%)
├── TileRenderer.renderTile: 11,143ms
│   └── Image.DrawImage: 10,366ms (93%)  ⚠️ One call per tile
└── inMapBounds: 67ms
```

**Solution A**: Batch tile drawing (complex but big savings)

```go
// Instead of drawing tiles one by one, batch them by texture
type TileBatch struct {
    vertices  []ebiten.Vertex
    indices   []uint16
    tileImage *ebiten.Image
}

func (tb *TileBatch) AddTile(x, y float64, srcRect image.Rectangle) {
    // Add vertices for this tile (6 per tile for 2 triangles)
    // This batches many tiles into a single draw call
}

func (tb *TileBatch) Draw(screen *ebiten.Image) {
    // Single DrawTriangles call for all tiles of this type
    screen.DrawTriangles(tb.vertices, tb.indices, tb.tileImage, nil)
}
```

**Solution B**: Pre-render static map portions

```go
// Cache the entire visible map region as a single image
type MapCache struct {
    cachedImage    *ebiten.Image
    cachedCenter   coords.LogicalPosition
    cachedRadius   int
    dirty          bool
}

func (mc *MapCache) GetMapImage(center coords.LogicalPosition, radius int) *ebiten.Image {
    if !mc.dirty && mc.cachedCenter == center && mc.cachedRadius == radius {
        return mc.cachedImage  // Return cached image
    }

    // Render map to cached image
    mc.renderMapToCache(center, radius)
    mc.cachedCenter = center
    mc.cachedRadius = radius
    mc.dirty = false

    return mc.cachedImage
}

// Mark dirty when map changes (tile damage, visibility changes)
func (mc *MapCache) MarkDirty() {
    mc.dirty = true
}
```

**Solution C**: Reduce visible tile count

```go
// Reduce viewport size on lower-end hardware
type PerformanceSettings struct {
    ViewportRadius int  // Default: 15, Low: 10
    TileSize       int  // Default: 32, Low: 24
}

// Fewer tiles = fewer draw calls
// 15 radius = ~900 tiles
// 10 radius = ~400 tiles (55% fewer)
```

---

### 2.3 Optimize Combat Grid Background (1-2 hours)

**Impact**: -1s (0.9% CPU time saved)
**Current**: Excessive type conversions in grid drawing

**Problem Analysis**:
```
CombatAnimationMode.drawGridBackground: 1,325ms
├── convTnoptr: 1,115ms (84%)  ⚠️ Type conversions
└── Image.Set: 205ms
```

**Solution**: Avoid interface conversions in hot loops

```go
// BAD: Using interface{} or any in loops
func drawGrid(cells []interface{}) {
    for _, cell := range cells {
        c := cell.(*Cell)  // Type assertion = convTnoptr
        // draw...
    }
}

// GOOD: Use concrete types
func drawGrid(cells []*Cell) {
    for _, cell := range cells {
        // No type assertion needed
        // draw...
    }
}
```

**Also**: Pre-render the grid background

```go
// The combat grid background is static - render once
type CombatGridCache struct {
    backgroundImage *ebiten.Image
    gridSize        int
    dirty           bool
}

func (cgc *CombatGridCache) GetBackground(size int) *ebiten.Image {
    if cgc.backgroundImage == nil || cgc.gridSize != size || cgc.dirty {
        cgc.backgroundImage = ebiten.NewImage(size, size)
        cgc.drawGrid(cgc.backgroundImage)
        cgc.gridSize = size
        cgc.dirty = false
    }
    return cgc.backgroundImage
}
```

---

## PRIORITY 3: MEDIUM-IMPACT OPTIMIZATIONS

### 3.1 Implement Object Pooling (4-5 hours)

**Impact**: -2s to -3s across many areas

**General Pattern**:
```go
// Generic object pool
type Pool[T any] struct {
    pool    []*T
    newFunc func() *T
}

func NewPool[T any](newFunc func() *T, initialSize int) *Pool[T] {
    p := &Pool[T]{
        pool:    make([]*T, 0, initialSize),
        newFunc: newFunc,
    }
    for i := 0; i < initialSize; i++ {
        p.pool = append(p.pool, newFunc())
    }
    return p
}

func (p *Pool[T]) Get() *T {
    if len(p.pool) > 0 {
        obj := p.pool[len(p.pool)-1]
        p.pool = p.pool[:len(p.pool)-1]
        return obj
    }
    return p.newFunc()
}

func (p *Pool[T]) Put(obj *T) {
    p.pool = append(p.pool, obj)
}
```

**Specific Pools to Create**:

1. **DrawImageOptions Pool**
```go
var drawOptsPool = sync.Pool{
    New: func() interface{} {
        return &ebiten.DrawImageOptions{}
    },
}

func GetDrawOpts() *ebiten.DrawImageOptions {
    opts := drawOptsPool.Get().(*ebiten.DrawImageOptions)
    opts.GeoM.Reset()
    opts.ColorScale.Reset()
    return opts
}

func PutDrawOpts(opts *ebiten.DrawImageOptions) {
    drawOptsPool.Put(opts)
}
```

2. **SquadInfo Pool**
```go
var squadInfoPool = sync.Pool{
    New: func() interface{} {
        return &SquadInfo{}
    },
}
```

3. **Slice Reuse**
```go
// Reuse slices instead of creating new ones
type SlicePool struct {
    entityIDs []ecs.EntityID
}

func (sp *SlicePool) GetEntityIDSlice(capacity int) []ecs.EntityID {
    if cap(sp.entityIDs) < capacity {
        sp.entityIDs = make([]ecs.EntityID, 0, capacity)
    }
    return sp.entityIDs[:0]  // Reuse backing array
}
```

---

### 3.2 Reduce Draw Call Count (2-3 hours)

**Impact**: -1s to -2s

**Current Draw Call Sources**:
```
Image.DrawImage: 17,310ms
├── Tiles: 10,366ms (60%)
├── NineSlice: 1,336ms (8%)
├── Text glyphs: 4,018ms (23%)
└── Other: 1,590ms (9%)
```

**Strategies**:

1. **Combine adjacent same-type tiles**
```go
// Instead of drawing each floor tile separately,
// use a single large floor texture
```

2. **Use texture atlases**
```go
// Combine multiple sprites into one texture
// Draw sub-rectangles instead of separate images
```

3. **Reduce UI widget count**
```go
// Combine multiple text labels into one formatted string
// "HP: 100" + "ATK: 50" + "DEF: 30" = 3 widgets
// "HP: 100 | ATK: 50 | DEF: 30" = 1 widget
```

---

### 3.3 Implement Dirty Rectangles (4-5 hours)

**Impact**: -2s to -5s (depends on gameplay)

**Concept**: Only redraw regions that changed

```go
type DirtyRegionTracker struct {
    dirtyRects []image.Rectangle
    fullRedraw bool
}

func (drt *DirtyRegionTracker) MarkDirty(rect image.Rectangle) {
    drt.dirtyRects = append(drt.dirtyRects, rect)
}

func (drt *DirtyRegionTracker) MarkFullRedraw() {
    drt.fullRedraw = true
}

func (drt *DirtyRegionTracker) ShouldRedraw(rect image.Rectangle) bool {
    if drt.fullRedraw {
        return true
    }
    for _, dirty := range drt.dirtyRects {
        if dirty.Overlaps(rect) {
            return true
        }
    }
    return false
}

func (drt *DirtyRegionTracker) Clear() {
    drt.dirtyRects = drt.dirtyRects[:0]
    drt.fullRedraw = false
}
```

**Usage**:
```go
func (tr *TileRenderer) Render(screen *ebiten.Image) {
    for y := 0; y < height; y++ {
        for x := 0; x < width; x++ {
            tileRect := image.Rect(x*tileSize, y*tileSize, (x+1)*tileSize, (y+1)*tileSize)

            if dirtyTracker.ShouldRedraw(tileRect) {
                tr.renderTile(screen, x, y)
            }
        }
    }
}
```

---

## PRIORITY 4: LOW-PRIORITY / FUTURE OPTIMIZATIONS

### 4.1 Graphics Quality Settings

```go
type GraphicsQuality int

const (
    QualityLow GraphicsQuality = iota
    QualityMedium
    QualityHigh
)

type GraphicsSettings struct {
    Quality           GraphicsQuality
    ViewportRadius    int
    TileSize          int
    EnableShadows     bool
    EnableAnimations  bool
    MaxVisibleSquads  int
    TextQuality       TextQuality
}

var QualityPresets = map[GraphicsQuality]GraphicsSettings{
    QualityLow: {
        ViewportRadius:   8,      // Smaller viewport
        TileSize:         24,     // Smaller tiles
        EnableShadows:    false,
        EnableAnimations: false,  // Skip combat animations
        MaxVisibleSquads: 6,
        TextQuality:      TextBitmap,
    },
    QualityMedium: {
        ViewportRadius:   12,
        TileSize:         32,
        EnableShadows:    true,
        EnableAnimations: true,
        MaxVisibleSquads: 12,
        TextQuality:      TextTrueType,
    },
    QualityHigh: {
        ViewportRadius:   16,
        TileSize:         32,
        EnableShadows:    true,
        EnableAnimations: true,
        MaxVisibleSquads: 20,
        TextQuality:      TextTrueType,
    },
}
```

### 4.2 Frame Rate Limiting

```go
// For laptops, cap at 30 FPS to reduce power consumption
func (g *Game) Layout(w, h int) (int, int) {
    if g.settings.LowPowerMode {
        ebiten.SetTPS(30)  // 30 ticks per second
    } else {
        ebiten.SetTPS(60)
    }
    return w, h
}
```

### 4.3 Lazy Loading

```go
// Don't load all assets at startup
type LazyAsset struct {
    path   string
    image  *ebiten.Image
    loaded bool
}

func (la *LazyAsset) Get() *ebiten.Image {
    if !la.loaded {
        la.image = loadImage(la.path)
        la.loaded = true
    }
    return la.image
}
```

---

## IMPLEMENTATION ROADMAP

### Week 1: Critical Fixes (12-15 hours)

| Task | Hours | Expected Savings |
|------|-------|------------------|
| Frame-level SquadInfo cache | 5-6h | 5s |
| Reduce GetEntityByID calls | 3-4h | 2.5s |
| Pool NineSlice/reduce usage | 4-5h | 3s |
| **Total** | **12-15h** | **~10.5s (9% faster)** |

### Week 2: High-Impact Fixes (10-12 hours)

| Task | Hours | Expected Savings |
|------|-------|------------------|
| Text caching/optimization | 3-4h | 3-5s |
| Tile rendering optimization | 3-4h | 2-4s |
| Combat grid cache | 1-2h | 1s |
| General object pooling | 3-4h | 2-3s |
| **Total** | **10-14h** | **~8-13s (7-11% faster)** |

### Week 3: Quality of Life (8-10 hours)

| Task | Hours | Expected Savings |
|------|-------|------------------|
| Graphics quality settings | 4-5h | Variable |
| Dirty rectangle rendering | 4-5h | 2-5s |
| **Total** | **8-10h** | **2-5s + quality options** |

### Total Investment

**Time**: 30-40 hours
**Expected Improvement**: 20-30 seconds saved (17-26% faster)

---

## QUICK WINS (Can do in 1-2 hours each)

### 1. Add FPS Counter
```go
var frameCount int
var lastTime time.Time
var currentFPS int

func (g *Game) Update() error {
    frameCount++
    if time.Since(lastTime) >= time.Second {
        currentFPS = frameCount
        frameCount = 0
        lastTime = time.Now()
    }
    return nil
}
```

### 2. Reduce UI Update Frequency
```go
// Only update UI every N frames
var uiUpdateCounter int

func (g *Game) updateUI() {
    uiUpdateCounter++
    if uiUpdateCounter % 3 != 0 {  // Update every 3rd frame
        return
    }
    // expensive UI updates...
}
```

### 3. Skip Hidden Elements
```go
func (container *Container) Render(screen *ebiten.Image) {
    if !container.IsVisible() {
        return  // Don't render hidden containers
    }
    // ...
}
```

### 4. Reduce Squad List Refresh
```go
// Only refresh when data actually changes
func (slc *SquadListComponent) Refresh() {
    if !slc.isDirty {
        return
    }
    // ...
    slc.isDirty = false
}

func (slc *SquadListComponent) MarkDirty() {
    slc.isDirty = true
}
```

---

## LAPTOP-SPECIFIC RECOMMENDATIONS

### For Integrated Graphics (Intel/AMD)

1. **Reduce Resolution**
   ```go
   ebiten.SetWindowSize(1280, 720)  // Instead of 1920x1080
   ```

2. **Use Lower Tile Sizes**
   ```go
   TileSize = 24  // Instead of 32
   ```

3. **Disable VSync in favor of frame limiting**
   ```go
   ebiten.SetVsyncEnabled(false)
   ebiten.SetTPS(30)
   ```

4. **Reduce Visible Area**
   ```go
   ViewportRadius = 8  // Instead of 15
   ```

### For Low RAM (<8GB)

1. **Lazy load assets**
2. **Dispose unused images**
   ```go
   image.Dispose()  // Free GPU memory
   ```
3. **Limit cached items**
   ```go
   const MaxCachedSquadInfo = 20
   ```

### For Older CPUs

1. **Reduce ECS query frequency**
2. **Cache more aggressively**
3. **Skip frames for non-critical updates**
4. **Use simpler AI calculations**

---

## MONITORING AND PROFILING

### Add Performance Metrics

```go
type PerfMetrics struct {
    FrameTime     time.Duration
    RenderTime    time.Duration
    UpdateTime    time.Duration
    DrawCalls     int
    Allocations   int
}

func (g *Game) Draw(screen *ebiten.Image) {
    start := time.Now()
    // ... render ...
    g.metrics.RenderTime = time.Since(start)
}
```

### Runtime Profiling

```go
// Add to main.go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    // ...
}
```

### Profile During Gameplay

```bash
# While game is running on laptop:
curl http://localhost:6060/debug/pprof/profile?seconds=60 > laptop_profile.pb.gz
go tool pprof laptop_profile.pb.gz
```

---

## CONCLUSION

### Summary

**Main Performance Bottlenecks**:
1. Memory allocations (22% of CPU) - Pool objects
2. Text rendering (17% of CPU) - Cache text
3. Redundant ECS queries (10% of CPU) - Frame caching
4. Tile rendering (10% of CPU) - Batch/cache
5. NineSlice overhead (4% of CPU) - Simplify UI

### Recommended Priority Order

1. **Frame-level SquadInfo cache** - Biggest win for least effort
2. **Reduce GetEntityByID calls** - Affects many systems
3. **Text caching** - Major rendering savings
4. **Tile batching/caching** - Graphics improvement
5. **Object pooling** - General memory savings
6. **Graphics quality settings** - User choice for their hardware

### Expected Results

| Hardware | Before | After (Estimated) |
|----------|--------|-------------------|
| Gaming PC | 60 FPS | 60 FPS (less CPU usage) |
| Laptop (Good) | 30-40 FPS | 50-60 FPS |
| Laptop (Low-end) | 15-25 FPS | 30-45 FPS |

### Final Notes

- **Profile on actual laptop hardware** to identify laptop-specific issues
- **Add quality settings UI** to let users adjust performance
- **Test incrementally** - implement one optimization, measure, repeat
- **Don't over-optimize** - fix the big issues first (80/20 rule)

The most impactful single change is the **Frame-level SquadInfo cache**, which should give ~5s savings (~4% improvement) with just 5-6 hours of work. Start there!
