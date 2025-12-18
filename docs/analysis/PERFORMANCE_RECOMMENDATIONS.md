# TinkerRogue Performance Optimization Recommendations

**Generated**: 2025-12-17
**Based On**: `docs/benchmarking/newest_benchmark/newest_benchmark.pb.gz`
**Profile Duration**: 120.10s (101.714s CPU time)

---

## EXECUTIVE SUMMARY

### Current Performance Status: GOOD

**Key Findings:**
- **75% of CPU time** is in external libraries (Ebiten, EbitenUI, font rendering) - **Cannot optimize**
- **25% of CPU time** is in game logic - **Partially optimizable**
- **Target optimizations**: ~800ms of overhead can be reduced to ~25ms (32x improvement)

**Top Bottlenecks:**
1. **Text/Font Rendering**: 19.94s (19.6%) - External library, but usage can be optimized
2. **UI Widget Rendering**: 26.49s (26.0%) - EbitenUI overhead, minimal control
3. **Memory Allocations**: 34.65s (34.1%) - Mix of external + internal, some optimizable
4. **Tile Rendering**: 12.76s (12.5%) - Acceptable for tactical grid
5. **ECS Component Access**: 3.52s (3.5%) - Library limitation, but can reduce calls

---

## PRIORITY 1: CRITICAL - IMMEDIATE WINS (4-6 hours)

### 1. Cache TurnManager.GetCurrentFaction (1 hour)

**Impact**: Eliminates 585ms (0.58% CPU time)

**Location**: `combat/turnmanager.go:93`

**Problem**:
```go
func (tm *TurnManager) GetCurrentFaction() ecs.EntityID {
    // O(n) query every call - 580ms wasted
    turnEntity := findTurnStateEntity(tm.manager)
    // ...
}
```

**Solution**:
```go
type TurnManager struct {
    manager         *common.EntityManager
    combatCache     *CombatQueryCache
    turnStateEntity *ecs.Entity  // NEW: Cache the entity
}

func (tm *TurnManager) InitializeCombat(factionIDs []ecs.EntityID) error {
    // ... existing code ...
    turnEntity := tm.manager.World.NewEntity()
    turnEntity.AddComponent(TurnStateComponent, &TurnStateData{...})

    tm.turnStateEntity = turnEntity  // Cache it
    return nil
}

func (tm *TurnManager) GetCurrentFaction() ecs.EntityID {
    if tm.turnStateEntity == nil {
        return 0  // No combat active
    }

    turnState := common.GetComponentType[*TurnStateData](
        tm.turnStateEntity, TurnStateComponent)
    if turnState == nil ||
       turnState.CurrentTurnIndex >= len(turnState.TurnOrder) {
        return 0
    }
    return turnState.TurnOrder[turnState.CurrentTurnIndex]
}

func (tm *TurnManager) EndCombat() error {
    // ... existing code ...
    tm.turnStateEntity = nil  // Invalidate cache
    return nil
}
```

**Testing**:
```bash
# Verify no regressions
go test ./combat/... -v
```

---

### 2. Remove Defer/Recover Overhead (2 hours)

**Impact**: Eliminates 25ms + improves debugging

**Location**: `common/ecsutil.go:95,117,142`

**Problem**: Empty defer/recover in every component access
```go
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {
    defer func() {
        if r := recover(); r != nil {
            // Empty - no error handling!
        }
    }()
    // ...
}
```

**Solution**:
```go
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {
    c, ok := entity.GetComponentData(component)
    if !ok {
        var nilValue T
        return nilValue
    }

    // Safe type assertion
    if typed, ok := c.(T); ok {
        return typed
    }

    // Type mismatch - log in debug mode
    var nilValue T
    return nilValue
}

func GetComponentTypeByID[T any](manager *EntityManager, entityID ecs.EntityID,
                                  component *ecs.Component) T {
    queryResult := manager.World.GetEntityByID(entityID)
    if queryResult == nil {
        var nilValue T
        return nilValue
    }

    return GetComponentType[T](queryResult.Entity, component)
}

func GetAttributesByIDWithTag(manager *EntityManager, entityID ecs.EntityID,
                               tag ecs.Tag) *Attributes {
    queryResult := manager.World.GetEntityByID(entityID, tag)
    if queryResult == nil {
        return nil
    }

    return GetComponentType[*Attributes](queryResult.Entity, AttributeComponent)
}
```

**Testing**:
```bash
# Run full test suite
go test ./... -v

# Check for nil panics
go test ./common/... -race
```

---

### 3. Pre-allocate Slice Capacities (1 hour)

**Impact**: Eliminates 30-49ms in growslice calls

**Locations**: `squads/squadqueries.go`, `combat/combatqueries.go`

**Problem**: Slices grow from 0 → 1 → 2 → 4 → 8, causing multiple reallocations

**Solution**:
```go
// squads/squadqueries.go
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    unitIDs := make([]ecs.EntityID, 0, 12)  // Typical squad: 6-10 units

    for _, result := range manager.World.Query(SquadMemberTag) {
        memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
        if memberData.SquadID == squadID {
            unitIDs = append(unitIDs, result.Entity.GetID())
        }
    }
    return unitIDs
}

func GetUnitIDsInRow(squadID ecs.EntityID, row int, manager *common.EntityManager) []ecs.EntityID {
    unitIDs := make([]ecs.EntityID, 0, 3)  // Max 3 per row
    seen := make(map[ecs.EntityID]bool, 3)
    // ...
}

func FindAllSquads(manager *common.EntityManager) []ecs.EntityID {
    allSquads := make([]ecs.EntityID, 0, 20)  // Typical: 10-20 squads
    // ...
}

// combat/combatqueries.go
func GetSquadsForFaction(factionID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    squadIDs := make([]ecs.EntityID, 0, 10)  // Typical faction: 5-10 squads
    // ...
}
```

**Files to Update**:
- `squads/squadqueries.go` (lines 42, 76, 137, 165)
- `combat/combatqueries.go` (lines 67, 98)
- `gui/guicomponents/guiqueries.go` (lines 156, 178)

---

## PRIORITY 2: HIGH - TEXT RENDERING OPTIMIZATION (3-4 hours)

### 4. Reduce Text Rendering Overhead (3 hours)

**Impact**: Could save 1-3 seconds (1-3% CPU time)

**Problem**: Text rendering consumes 19.94s (19.6% CPU time)
```
widget.(*Text).Render:  19,994ms (19.66%)
widget.(*Text).draw:    19,948ms (19.61%)
truetype.(*hinter).run: 12,830ms (12.61%)
font.MeasureString:     13,873ms (13.64%)
```

**Root Causes**:
1. Text measurement happens on every render
2. Same strings measured repeatedly (squad names, stats)
3. Font glyph loading not cached

**Solution A: Cache Text Measurements**

```go
// gui/guicomponents/textcache.go (NEW FILE)
package guicomponents

import (
    "sync"
    "golang.org/x/image/font"
)

type TextMeasurement struct {
    Width  int
    Height int
}

type TextMeasureCache struct {
    cache map[string]TextMeasurement
    mu    sync.RWMutex
    face  font.Face
}

func NewTextMeasureCache(face font.Face) *TextMeasureCache {
    return &TextMeasureCache{
        cache: make(map[string]TextMeasurement, 100),
        face:  face,
    }
}

func (tmc *TextMeasureCache) MeasureString(s string) TextMeasurement {
    tmc.mu.RLock()
    if measurement, ok := tmc.cache[s]; ok {
        tmc.mu.RUnlock()
        return measurement
    }
    tmc.mu.RUnlock()

    // Measure (expensive)
    advance := font.MeasureString(tmc.face, s)
    bounds, _ := tmc.face.GlyphBounds('M')

    measurement := TextMeasurement{
        Width:  advance.Ceil(),
        Height: (bounds.Max.Y - bounds.Min.Y).Ceil(),
    }

    tmc.mu.Lock()
    tmc.cache[s] = measurement
    tmc.mu.Unlock()

    return measurement
}

// Clear when font changes
func (tmc *TextMeasureCache) Clear() {
    tmc.mu.Lock()
    tmc.cache = make(map[string]TextMeasurement, 100)
    tmc.mu.Unlock()
}
```

**Solution B: Reduce Text Widget Count**

Check if you're creating excessive text widgets:
```bash
# Find text widget creation patterns
grep -r "widget.Text{" gui/ --include="*.go" | wc -l
grep -r "NewText" gui/ --include="*.go" | wc -l
```

Optimize by:
1. **Reuse widgets** instead of recreating every frame
2. **Batch static text** (labels that don't change)
3. **Use TextArea sparingly** (15,780ms just for TextArea.Render)

**Solution C: Simplify BBCode Parsing**
```
widget.(*Text).handleBBCodeColor: 706ms (0.69%)
```

If you're using BBCode for colored text, consider:
1. Pre-parse BBCode once when text is set
2. Cache parsed segments
3. Or avoid BBCode for static/frequently updated text

---

### 5. Optimize Squad Info Rendering (1 hour)

**Impact**: Reduces redundant text rendering

**Problem**: Squad stats are rendered as text on every frame
```
GetSquadInfo:       3,797ms (3.73%)
GetSquadInfoCached: 3,796ms (3.73%)
BuildSquadInfoCache:  96ms (0.09%)
```

**Current Flow**:
```
Frame N:
  BuildSquadInfoCache() -> 96ms (good - builds data maps)
  GetSquadInfo(squadA) -> Creates Text widget -> Measures text -> Renders
  GetSquadInfo(squadB) -> Creates Text widget -> Measures text -> Renders
  ... (repeated for every visible squad)
```

**Optimization**:
```go
// gui/guicomponents/squadinforenderer.go (NEW)
type SquadInfoRenderer struct {
    textCache       *TextMeasureCache
    cachedWidgets   map[ecs.EntityID]*widget.Text  // Reuse widgets
    lastUpdateFrame int
}

func (sir *SquadInfoRenderer) GetOrCreateSquadText(squadID ecs.EntityID,
                                                    info SquadInfo) *widget.Text {
    // Reuse widget if data unchanged
    if w, exists := sir.cachedWidgets[squadID]; exists {
        // Only update text if squad info changed
        if w.Label != info.DisplayText {
            w.Label = info.DisplayText
        }
        return w
    }

    // Create new widget
    w := widget.NewText(widget.TextOpts.Text(info.DisplayText, ...))
    sir.cachedWidgets[squadID] = w
    return w
}
```

---

## PRIORITY 3: MEDIUM - ECS QUERY OPTIMIZATION (3-4 hours)

### 6. Add SquadView to CombatQueryCache (2 hours)

**Impact**: Saves ~100-150ms per profile

**Location**: `combat/combatqueriescache.go`

**Problem**: `GetSquadsForFaction` performs O(n) World.Query on every call

**Solution**:
```go
// combat/combatqueriescache.go
type CombatQueryCache struct {
    ActionStateView *ecs.View
    FactionView     *ecs.View
    SquadView       *ecs.View  // NEW
}

func NewCombatQueryCache(manager *common.EntityManager) *CombatQueryCache {
    return &CombatQueryCache{
        ActionStateView: manager.World.CreateView(ActionStateTag),
        FactionView:     manager.World.CreateView(FactionTag),
        SquadView:       manager.World.CreateView(squads.SquadTag),  // NEW
    }
}

// combat/combatqueries.go
func GetSquadsForFaction(factionID ecs.EntityID,
                         cache *CombatQueryCache) []ecs.EntityID {
    squadIDs := make([]ecs.EntityID, 0, 10)

    // Use cached View instead of World.Query
    for _, result := range cache.SquadView.Get() {
        combatFaction := common.GetComponentType[*CombatFactionData](
            result.Entity, CombatFactionComponent)
        if combatFaction != nil && combatFaction.FactionID == factionID {
            squadIDs = append(squadIDs, result.Entity.GetID())
        }
    }
    return squadIDs
}
```

**Update Callers**:
```go
// combat/turnmanager.go - Update to pass cache
func (tm *TurnManager) InitializeCombat(factionIDs []ecs.EntityID) error {
    // ...
    for _, factionID := range factionIDs {
        squadIDs := GetSquadsForFaction(factionID, tm.combatCache)  // Pass cache
        // ...
    }
}
```

---

### 7. Audit Remaining World.Query Calls (2 hours)

**Goal**: Find and cache remaining uncached queries

**Process**:
```bash
# Find all World.Query calls
grep -rn "World.Query" --include="*.go" | grep -v "CreateView"

# Check each call:
# 1. Is it in a hot path (called frequently)?
# 2. Can it use an existing View?
# 3. Should we add a new View?
```

**Common Patterns to Convert**:
```go
// BEFORE (uncached)
for _, result := range manager.World.Query(SomeTag) {
    // Process...
}

// AFTER (cached)
// 1. Add View to appropriate cache struct
type SomeCache struct {
    SomeView *ecs.View
}

// 2. Use View instead
for _, result := range cache.SomeView.Get() {
    // Process...
}
```

---

## PRIORITY 4: LOW - MICRO-OPTIMIZATIONS (2-3 hours)

### 8. Reduce GetEntityByID Calls (2 hours)

**Impact**: Minor, but good practice

**Problem**: GetEntityByID creates new maps on every call (3.52s total)
```
GetEntityByID: 3,525ms (3.47%)
  - newobject:                1,800ms (allocate QueryResult)
  - fetchComponentsForEntity: 1,677ms (build component map)
```

**You can't fix the library, but you can reduce calls**:

```bash
# Find GetEntityByID usage
grep -rn "GetEntityByID" --include="*.go" | wc -l

# Check if any can use entity from query result instead
```

**Pattern to Follow**:
```go
// PREFER: Use entity from query
for _, result := range manager.World.Query(SquadTag) {
    entity := result.Entity  // Already have entity
    data := common.GetComponentType[*Data](entity, Component)  // Direct access
}

// AVOID: Looking up by ID when you already have entity
for _, result := range manager.World.Query(SquadTag) {
    entityID := result.Entity.GetID()
    entity := manager.World.GetEntityByID(entityID)  // Wasteful!
    data := common.GetComponentType[*Data](entity, Component)
}
```

---

### 9. Optimize TileRenderer DrawImageOptions (1 hour)

**Impact**: Small, but you already did similar optimization

**Location**: `worldmap/tilerenderer.go`

**Current State**: Already optimized! (reuses DrawImageOptions)

**Double-check**:
```go
// Verify you're reusing DrawImageOptions
type TileRenderer struct {
    opts *ebiten.DrawImageOptions  // Reused
}

func (tr *TileRenderer) renderTile(...) {
    tr.opts.GeoM.Reset()  // Reset instead of new
    tr.opts.GeoM.Scale(...)
    img.DrawImage(tile, tr.opts)  // Reuse
}
```

**If not already done**, apply same pattern to:
- `gui/guicombat/combatanimationmode.go` (grid background)
- `gui/guimodes/movementtilerenderer.go` (movement tiles)

---

## PRIORITY 5: MONITORING & MEASUREMENT

### 10. Set Up Continuous Profiling

```go
// game_main/main.go
import (
    "net/http"
    _ "net/http/pprof"
    "log"
)

func main() {
    // Start profiling server in debug builds
    if debugMode {
        go func() {
            log.Println("Profiling: http://localhost:6060/debug/pprof/")
            log.Println(http.ListenAndServe("localhost:6060", nil))
        }()
    }

    // ... rest of main
}
```

**Usage**:
```bash
# CPU profile during gameplay
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Memory profile
curl http://localhost:6060/debug/pprof/heap > mem.prof
go tool pprof mem.prof

# Goroutine analysis
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof
```

---

## IMPLEMENTATION ROADMAP

### Week 1: Critical Wins (4-6 hours)
- [ ] Cache TurnState entity (1h)
- [ ] Remove defer/recover (2h)
- [ ] Pre-allocate slices (1h)
- [ ] Test and verify (1h)

**Expected Gain**: 640ms → 25ms (25x improvement, 0.6% CPU time)

### Week 2: Text Optimization (3-4 hours)
- [ ] Add text measurement cache (2h)
- [ ] Optimize squad info rendering (1h)
- [ ] Test text rendering (1h)

**Expected Gain**: 1-3s reduction (1-3% CPU time)

### Week 3: ECS Optimization (3-4 hours)
- [ ] Add SquadView to combat cache (2h)
- [ ] Audit World.Query calls (2h)

**Expected Gain**: 100-150ms reduction (0.1-0.15% CPU time)

### Week 4: Measurement
- [ ] Generate new benchmark (1h)
- [ ] Compare before/after (1h)
- [ ] Document improvements (1h)

---

## WHAT NOT TO OPTIMIZE

### Don't Optimize These (External Libraries):
1. **EbitenUI rendering** (26.49s, 26%) - Can't change
2. **Font glyph loading** (12.83s, 12.6%) - Can't change
3. **Ebiten DrawImage** (19.76s, 19.4%) - Can't change
4. **GC overhead** (13.65s, 13.4%) - Runtime management

### Don't Optimize These (Acceptable Cost):
1. **Tile rendering** (12.76s, 12.5%) - Reasonable for tactical grid
2. **Combat logic** (521ms, 0.5%) - Already efficient
3. **ECS library internals** (3.52s, 3.5%) - External library

---

## EXPECTED RESULTS

### Current Profile (101.7s CPU time):
- External libraries: 75.9s (75%)
- Game logic: 25.8s (25%)
  - Optimizable: 0.8s (0.8%)
  - Acceptable: 25.0s (24.2%)

### After Optimizations:
- External libraries: 75.9s (75%) - Unchanged
- Game logic: 24-25s (24-25%)
  - Optimizable: 0.025s (0.025%) ✅
  - Acceptable: 24-25s (24-25%)

### Overall Impact:
- **CPU time reduction**: 1-2% (visible in profiler, minor in gameplay)
- **Code quality**: Significantly improved (cleaner, faster, more maintainable)
- **Memory usage**: Reduced allocations
- **Debugging**: Easier without defer/recover wrapping

---

## CONCLUSION

### Performance Verdict: ALREADY WELL-OPTIMIZED

**Your code is excellent**. 75% of CPU time is in external libraries (expected for a graphics game). The remaining 25% is mostly acceptable overhead.

**Recommended Actions**:
1. ✅ **Implement Priority 1** (Critical wins) - 4-6 hours, worth it
2. ⚠️ **Consider Priority 2** (Text optimization) - 3-4 hours, good ROI if text-heavy UI
3. ⏸️ **Skip Priority 3-5** unless profiling shows new bottlenecks

**Total Recommended Effort**: 7-10 hours
**Expected Improvement**: 1.5-2% CPU time reduction + cleaner code

**The game is already performant enough for a turn-based tactical RPG.** Focus optimizations on code quality and maintainability rather than chasing marginal FPS gains.
