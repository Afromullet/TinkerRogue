# GUI Performance Analysis - Critical Bottlenecks

**Profile Date:** 2025-12-09
**Total Profile Duration:** 120.13 seconds
**Total Samples:** 91,378ms (76.07% CPU utilization)
**Analysis Type:** CPU profiling of GUI rendering and update loops

---

## Executive Summary

The GUI has **5 critical performance bottlenecks** consuming over 90% of CPU time:

1. **Memory Allocation Storm** - 56.7s (62.07%) - Consequence of excessive object creation
2. **ECS Query Overhead** - 30.7s (33.66%) - Expensive map allocations per query
3. **Text Rendering** - 16.8s (18.36%) - Font measurement on every draw
4. **Tile Rendering** - 16.9s (18.50%) - Viewport allocation per tile
5. **Viewport Allocation** - 12.0s (13.15%) - Creating new Viewport for every coordinate conversion

**Impact:** Game running at ~76% CPU just for rendering, causing potential frame drops and high power consumption.

---

## Critical Bottleneck #1: Viewport Allocation Storm

### The Problem

**Location:** `coords/cordmanager.go:278` in `LogicalToScreen()`

```go
func (cm *CoordinateManager) LogicalToScreen(pos LogicalPosition, centerPos *LogicalPosition) (float64, float64) {
    // ...
    viewport := NewViewport(cm, *centerPos)  // LINE 278 - 12.01s allocating!
    return viewport.LogicalToScreen(pos)
}
```

**Cost:** 12.0s (13.15% of total CPU time)

**Root Cause:**
Every call to `LogicalToScreen()` creates a **new Viewport struct**. This function is called:
- Once per tile rendered (hundreds of tiles per frame at 60 FPS)
- From `TileRenderer.renderTile()` line 86 (11.72s consumed here alone)
- From `ViewportRenderer.LogicalToScreen()` for overlays and borders

**Scale of Problem:**
If rendering 500 tiles per frame at 60 FPS = **30,000 Viewport allocations per second**

### Solution

**Option A: Cache Single Viewport (Recommended)**

Create one viewport per frame and reuse it:

```go
type CoordinateManager struct {
    // ... existing fields
    cachedViewport *Viewport
    lastCenterPos  LogicalPosition
}

func (cm *CoordinateManager) LogicalToScreen(pos LogicalPosition, centerPos *LogicalPosition) (float64, float64) {
    // ... scale factor logic
    pixelPos := cm.LogicalToPixel(pos)

    if centerPos == nil || !MAP_SCROLLING_ENABLED {
        return float64(pixelPos.X) * float64(scaleFactor),
               float64(pixelPos.Y) * float64(scaleFactor)
    }

    // CACHE VIEWPORT - only recreate if center moved
    if cm.cachedViewport == nil || cm.lastCenterPos != *centerPos {
        cm.cachedViewport = NewViewport(cm, *centerPos)
        cm.lastCenterPos = *centerPos
    }

    return cm.cachedViewport.LogicalToScreen(pos)
}
```

**Expected Savings:** ~11-12 seconds per 120s profile = **9-10% total CPU reduction**

**Option B: Inline Calculation**

Eliminate Viewport entirely and inline the math:

```go
func (cm *CoordinateManager) LogicalToScreen(pos LogicalPosition, centerPos *LogicalPosition) (float64, float64) {
    scaleFactor := 1
    if MAP_SCROLLING_ENABLED {
        scaleFactor = cm.scaleFactor
    }

    pixelX := float64(pos.X * cm.tileSize)
    pixelY := float64(pos.Y * cm.tileSize)

    if centerPos == nil || !MAP_SCROLLING_ENABLED {
        return pixelX * float64(scaleFactor), pixelY * float64(scaleFactor)
    }

    // Inline viewport calculation (no allocation)
    centerPixelX := float64(centerPos.X * cm.tileSize)
    centerPixelY := float64(centerPos.Y * cm.tileSize)

    offsetX := float64(cm.screenWidth)/2 - centerPixelX*float64(scaleFactor)
    offsetY := float64(cm.screenHeight)/2 - centerPixelY*float64(scaleFactor)

    return pixelX*float64(scaleFactor) + offsetX, pixelY*float64(scaleFactor) + offsetY
}
```

**Expected Savings:** ~12 seconds = **13% total CPU reduction** (eliminates allocation entirely)

---

## Critical Bottleneck #2: ECS Query Map Allocations

### The Problem

**Location:** `ecs.go:370,375,379` in `(*Manager).Query()`

```go
func (manager *Manager) Query(tag Tag) queryResultCollection {
    matches := make(queryResultCollection, 0)

    for _, entity := range manager.entities {
        if entity.tag&tag == tag {
            componentMap := make(map[*Component]interface{})  // LINE 370 - 9.89s!

            for _, component := range manager.components {
                if component.tag&tag == component.tag {
                    data, _ := entity.GetComponentData(component)
                    componentMap[component] = data  // LINE 375 - 5.92s!
                }
            }

            matches = append(matches, &QueryResult{  // LINE 379 - 14.34s!
                Entity:     entity,
                Components: componentMap,
            })
        }
    }
    return matches
}
```

**Cost:** 30.7s (33.66% of total CPU time)

**Root Cause:**
The ECS library creates a **new map for every matching entity** on every query. With hundreds of entities and frequent queries, this causes massive allocations.

**Current Usage:**
- `guicomponents/guiqueries.go:191` - `GetAllFactions()`
- `guicomponents/guiqueries.go:318` - `isMonster()`
- Called indirectly through various GUI update loops

### Solution

**Option A: Use ECS Views (Already Partially Implemented)**

The codebase already has View infrastructure but isn't using it everywhere:

```go
// In GUIQueries (guicomponents/guiqueries.go)
type GUIQueries struct {
    ECSManager     *common.EntityManager

    // Already exist - USE THESE!
    squadView       *ecs.View // All SquadTag entities
    squadMemberView *ecs.View // All SquadMemberTag entities
    actionStateView *ecs.View // All ActionStateTag entities
}
```

**Current Good Usage (line 391):**
```go
// Uses cached View - fast!
for _, result := range gq.squadView.Get() {
    entity := result.Entity
    // ...
}
```

**Current Bad Usage (line 191):**
```go
// Creates fresh query - SLOW!
for _, result := range gq.ECSManager.World.Query(combat.FactionTag) {
    factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
    factionIDs = append(factionIDs, factionData.FactionID)
}
```

**Fix for GetAllFactions():**

```go
// In NewGUIQueries(), add:
factionView: ecsManager.World.CreateView(combat.FactionTag),

// In GetAllFactions(), change to:
func (gq *GUIQueries) GetAllFactions() []ecs.EntityID {
    factionIDs := []ecs.EntityID{}
    for _, result := range gq.factionView.Get() {  // Use View instead of Query
        factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
        factionIDs = append(factionIDs, factionData.FactionID)
    }
    return factionIDs
}
```

**Expected Savings:** ~25-30 seconds = **27-33% total CPU reduction**

**Option B: Cache Query Results**

For queries that don't change frequently:

```go
type GUIQueries struct {
    // ... existing fields

    // Cached query results (invalidate when entities created/destroyed)
    cachedFactions     []ecs.EntityID
    factionCacheValid  bool
}

func (gq *GUIQueries) GetAllFactions() []ecs.EntityID {
    if gq.factionCacheValid {
        return gq.cachedFactions
    }

    // Rebuild cache
    gq.cachedFactions = []ecs.EntityID{}
    for _, result := range gq.factionView.Get() {
        factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
        gq.cachedFactions = append(gq.cachedFactions, factionData.FactionID)
    }
    gq.factionCacheValid = true

    return gq.cachedFactions
}

// Call when factions change
func (gq *GUIQueries) InvalidateFactionCache() {
    gq.factionCacheValid = false
}
```

---

## Critical Bottleneck #3: Text Rendering

### The Problem

**Location:** `ebitenui/widget/text.go:243` in `(*Text).draw()`

```go
func (t *Text) draw(screen *ebiten.Image) {
    // ...
    for _, word := range line {
        pieces, updatedColor := t.handleBBCodeColor(word)
        for _, piece := range pieces {
            text.Draw(screen, piece.text, t.Face, lx, ly, piece.color)
            wordWidth := font.MeasureString(t.Face, piece.text)  // LINE 243 - 12.70s!
            lx += wordWidth.Round()
        }
    }
}
```

**Cost:** 16.8s (18.36% of total CPU time)
- Font measurement: 12.7s
- Text drawing: 1.58s

**Root Cause:**
Every text widget measures font metrics **on every frame** even when text hasn't changed. This involves complex TrueType hinting (14% CPU in `truetype.(*hinter).run`).

### Solution

**Option A: Cache Text Measurements (Ebitenui Library Fix)**

This would require modifying the ebitenui library - **not recommended** as you'd have to maintain a fork.

**Option B: Reduce Text Updates**

Only update text widgets when data actually changes:

```go
// BEFORE (in CombatMode.Update() line 318-330)
func (cm *CombatMode) Update(deltaTime float64) error {
    // Updates EVERY FRAME even if nothing changed!
    cm.turnOrderComponent.Refresh()

    currentFactionID := cm.combatService.GetCurrentFaction()
    if currentFactionID != 0 {
        cm.factionInfoComponent.ShowFaction(currentFactionID)
    }

    selectedSquad := cm.Context.ModeCoordinator.GetBattleMapState().SelectedSquadID
    if selectedSquad != 0 {
        cm.squadDetailComponent.ShowSquad(selectedSquad)
    }

    return nil
}

// AFTER - Only update when state changes
func (cm *CombatMode) Update(deltaTime float64) error {
    // Track previous state
    if cm.lastFactionID != cm.combatService.GetCurrentFaction() {
        cm.turnOrderComponent.Refresh()
        cm.lastFactionID = cm.combatService.GetCurrentFaction()
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

**Option C: Use Simpler Fonts**

If using BBCode color parsing (line 237-248), consider:
- Disabling BBCode if not needed (`processBBCode = false`)
- Using pre-colored separate Text widgets instead of inline color changes
- Using bitmap fonts instead of TrueType (no hinting required)

**Expected Savings:** ~10-15 seconds = **11-16% total CPU reduction**

---

## Critical Bottleneck #4: Tile Rendering

### The Problem

**Location:** `worldmap/tilerenderer.go:63-95` in `renderTile()`

```go
func (r *TileRenderer) renderTile(x, y int, opts RenderOptions, bounds *RenderedBounds) {
    // ... FOV checks

    drawOpts := &ebiten.DrawImageOptions{}  // Allocation

    if opts.CenterOn != nil {
        r.applyViewportTransformWithBounds(drawOpts, tile, opts.CenterOn, bounds)  // LINE 86 - 11.72s!
    }

    r.applyColorMatrix(drawOpts, tile)
    opts.Screen.DrawImage(tile.image, drawOpts)  // LINE 94 - 4.90s!
}
```

**Cost:** 16.9s (18.50% of total CPU time)

**Root Cause:**
- Calls `LogicalToScreen()` which allocates Viewport (see Bottleneck #1)
- Creates new `DrawImageOptions` for every tile
- Calls `DrawImage()` hundreds of times per frame (no batching)

### Solution

**Option A: Fix Viewport Allocation (Primary)**

Implementing Bottleneck #1 solutions will eliminate the 11.72s cost in `applyViewportTransformWithBounds`.

**Option B: Reuse DrawImageOptions**

```go
type TileRenderer struct {
    tiles      []*Tile
    fov        *fov.View
    colorScale ebiten.ColorScale
    // Add reusable draw options
    drawOpts   *ebiten.DrawImageOptions
}

func (r *TileRenderer) renderTile(x, y int, opts RenderOptions, bounds *RenderedBounds) {
    // ... FOV checks

    // REUSE instead of allocate
    r.drawOpts.GeoM.Reset()
    r.drawOpts.ColorScale.Reset()

    if opts.CenterOn != nil {
        r.applyViewportTransformWithBounds(r.drawOpts, tile, opts.CenterOn, bounds)
    }

    r.applyColorMatrix(r.drawOpts, tile)
    opts.Screen.DrawImage(tile.image, r.drawOpts)
}
```

**Option C: Early Culling**

Add screen bounds check before expensive viewport transform:

```go
func (r *TileRenderer) renderTile(x, y int, opts RenderOptions, bounds *RenderedBounds) {
    logicalPos := coords.LogicalPosition{X: x, Y: y}

    // Quick screen bounds check BEFORE expensive operations
    if opts.CenterOn != nil {
        dx := abs(x - opts.CenterOn.X)
        dy := abs(y - opts.CenterOn.Y)
        if dx > opts.ViewportSize/2+1 || dy > opts.ViewportSize/2+1 {
            return  // Off-screen, skip entirely
        }
    }

    // ... rest of rendering
}
```

**Expected Savings (combined):** ~12-14 seconds = **13-15% total CPU reduction**

---

## Critical Bottleneck #5: Memory Allocation Overhead

### The Problem

**Location:** `runtime.mallocgc` (multiple call sites)

**Cost:** 56.7s (62.07% of total CPU time)

**Root Cause:**
This is the **consequence** of all the above bottlenecks:
- Viewport allocations (12s)
- ECS query map allocations (30s)
- DrawImageOptions allocations (per tile)
- Text widget allocations

**Garbage Collection Impact:**
High allocation rate triggers frequent GC pauses, adding additional overhead beyond the 56.7s measured.

### Solution

**No Direct Fix - Resolve Other Bottlenecks**

This overhead will automatically decrease by 40-50% when the other bottlenecks are fixed. The remaining allocations are:
- Legitimate temporary objects
- Framework overhead (ebitenui, ebiten)
- Unavoidable language overhead

**Expected Savings (after other fixes):** ~25-30 seconds = **27-33% total CPU reduction**

---

## Additional Optimization Opportunities

### 1. Border Image Caching (Already Implemented)

**Location:** `gui/guimodes/guirenderers.go:15-32`

✅ **Good:** BorderImageCache already caches border images to avoid GPU allocations.

**Potential Issue:** Still calling `Fill()` on cached images every frame (lines 84-87):

```go
// These Fill() calls may be redundant
topBorder.Fill(borderColor)
bottomBorder.Fill(borderColor)
leftBorder.Fill(borderColor)
rightBorder.Fill(borderColor)
```

**Fix:** Only fill if color changed:

```go
type BorderImageCache struct {
    top, bottom, left, right *ebiten.Image
    tileSize, thickness      int
    lastColor                color.Color  // Add this
}

func (cache *BorderImageCache) GetOrCreate(tileSize, thickness int, borderColor color.Color) (...) {
    if cache.top == nil || cache.tileSize != tileSize || cache.thickness != thickness {
        // Recreate images
        cache.top = ebiten.NewImage(tileSize, thickness)
        // ... others
        cache.lastColor = nil  // Force refill
    }

    // Only fill if color changed
    if cache.lastColor != borderColor {
        cache.top.Fill(borderColor)
        // ... others
        cache.lastColor = borderColor
    }

    return cache.top, cache.bottom, cache.left, cache.right
}
```

### 2. Squad Info Caching (Already Implemented)

**Location:** `gui/guicomponents/guiqueries.go:366-482`

✅ **Good:** `BuildSquadInfoCache()` and `GetSquadInfoCached()` already implement efficient caching.

**Usage Check:** Ensure all render paths use the cached version:

```go
// GOOD (guirenderers.go:171)
cache := shr.queries.BuildSquadInfoCache()
for _, squadID := range allSquads {
    squadInfo := shr.queries.GetSquadInfoCached(squadID, cache)  // Using cache!
}

// BAD (if exists anywhere)
squadInfo := shr.queries.GetSquadInfo(squadID)  // Don't use this in render!
```

### 3. ViewportRenderer Caching (Partially Implemented)

**Location:** `gui/guimodes/guirenderers.go:156-167`

✅ **Good:** SquadHighlightRenderer caches ViewportRenderer.

**Issue:** Still recreates on every Render() call if needed:

```go
// Only recreate renderer if viewport moved or not yet created
if shr.cachedRenderer == nil || shr.lastCenterPos != centerPos {
    shr.cachedRenderer = NewViewportRenderer(screen, centerPos)  // Allocation
    shr.lastCenterPos = centerPos
}
```

**Better:** Reuse and update existing renderer:

```go
if shr.cachedRenderer == nil {
    shr.cachedRenderer = NewViewportRenderer(screen, centerPos)
} else if shr.lastCenterPos != centerPos {
    // Update center without reallocation
    shr.cachedRenderer.centerPos = centerPos
}
shr.lastCenterPos = centerPos
```

---

## Implementation Priority

### Phase 1: Critical Fixes (60-70% CPU reduction)

**Week 1 - Viewport Allocation (13% gain)**
1. Implement viewport caching in `CoordinateManager.LogicalToScreen()`
2. OR inline viewport calculation to eliminate allocation entirely
3. Verify with profiler

**Week 2 - ECS Queries (27-33% gain)**
1. Add missing Views to GUIQueries (FactionView, etc.)
2. Replace all `World.Query()` calls with `View.Get()`
3. Profile to confirm reduction

**Week 3 - Text Update Optimization (11-16% gain)**
1. Add state tracking to mode Update() functions
2. Only call component Refresh() when state changes
3. Consider disabling BBCode if not used

### Phase 2: Secondary Optimizations (10-15% gain)

**Week 4**
1. Reuse DrawImageOptions in TileRenderer
2. Add early culling for off-screen tiles
3. Optimize border image cache to track color changes

### Phase 3: Validation

**Week 5**
1. Run full profiling session with all fixes
2. Compare before/after metrics
3. Document final performance characteristics

---

## Expected Results

### Before Optimization
- **Total CPU Time:** 91.38s / 120s profile = 76% CPU utilization
- **Frame Time Budget:** ~17ms at 60 FPS
- **Estimated Frame Drops:** Likely during complex scenes

### After Phase 1 (Conservative Estimate)
- **CPU Reduction:** 60-70% of bottleneck overhead
- **New CPU Time:** ~35-40s / 120s = 29-33% CPU utilization
- **Frame Time Budget:** ~6-8ms of CPU per frame
- **Frame Drops:** Eliminated for typical gameplay

### After All Phases
- **CPU Reduction:** 70-80% of bottleneck overhead
- **New CPU Time:** ~20-30s / 120s = 17-25% CPU utilization
- **Frame Time Budget:** ~3-5ms of CPU per frame
- **Headroom:** 2-3x performance margin for additional features

---

## Monitoring & Validation

### Before Each Fix
```bash
# Create baseline profile
go test -cpuprofile=cpu_before.pb.gz -bench=. ./...

# OR run game with profiling
go build -o game.exe game_main/*.go
# Add profiling flag to game
```

### After Each Fix
```bash
# Create comparison profile
go test -cpuprofile=cpu_after.pb.gz -bench=. ./...

# Compare
go tool pprof -top cpu_before.pb.gz > before.txt
go tool pprof -top cpu_after.pb.gz > after.txt
diff before.txt after.txt
```

### Key Metrics to Track
- `runtime.mallocgc` cumulative time (should decrease)
- `ecs.(*Manager).Query` cumulative time (should approach 0)
- `NewViewport` allocation count (should approach 0)
- `font.MeasureString` cumulative time (should decrease)
- Overall samples/second (should decrease)

---

## Risk Assessment

### Low Risk
- Viewport caching: Simple state tracking, easy to revert
- ECS Views: Already partially implemented, proven pattern
- DrawImageOptions reuse: Standard optimization

### Medium Risk
- Text update tracking: Need careful state management
- May miss edge cases where text should update

### High Risk
- Inlining viewport calculation: Changes coordinate system
- Requires thorough testing across all render paths

**Recommendation:** Start with Low Risk optimizations, measure results, then proceed to Medium/High Risk only if needed.

---

## File Reference

**Files Modified in Phase 1:**
- `coords/cordmanager.go` - Viewport caching
- `gui/guicomponents/guiqueries.go` - Add Views
- `gui/guicombat/combatmode.go` - State tracking
- `gui/guimodes/explorationmode.go` - State tracking

**Files to Profile:**
- `worldmap/tilerenderer.go` - Tile rendering
- `gui/guimodes/guirenderers.go` - Overlay rendering
- `gui/widgets/*.go` - Widget updates

---

## Conclusion

The GUI performance issues stem from **excessive object allocation** in hot paths:
1. Creating Viewports for every coordinate conversion (13% CPU)
2. Creating maps for every ECS query (34% CPU)
3. Measuring fonts on every text draw (19% CPU)

These three bottlenecks alone account for **66% of total CPU time**.

**Recommended Action:**
Implement Phase 1 fixes in priority order. Each fix is independent and can be validated separately. Expected total improvement: **60-70% reduction in CPU usage** with minimal code changes.

The good news: The codebase already has caching infrastructure (Views, BuildSquadInfoCache, BorderImageCache) - it just needs to be used consistently throughout the GUI systems.
