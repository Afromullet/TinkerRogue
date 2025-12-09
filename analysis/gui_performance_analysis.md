# GUI Performance Analysis - view_benchmark.pb.gz
**Date:** 2025-12-08
**Total Runtime:** 107.21 seconds
**Total Samples:** 107,213ms

## Executive Summary

The GUI subsystem has **critical performance issues** consuming significant CPU time during rendering. The top bottlenecks are:

1. **EbitenUI rendering (27.67s / 25.8%)** - Widget drawing dominates frame time
2. **Mode rendering (9.34s / 8.7%)** - Custom rendering in modes
3. **Mode updates (5.12s / 4.8%)** - Game logic per frame
4. **ECS queries for squad info (4.5s+ / 4.2%)** - Repeated lookups despite caching

---

## Top GUI Bottlenecks (Ranked by Impact)

### 🔥 CRITICAL #1: EbitenUI Widget Rendering (27.67s / 25.8%)

**Location:** `gui/core/modemanager.go:120`

```go
umm.currentMode.GetEbitenUI().Draw(screen)  // 27.67s
```

**Analysis:**
- **27.67 seconds** spent drawing ebitenui widgets
- This is the single largest performance cost in the entire GUI
- Includes `ebitenui/widget.(*Container).Render` (26.44s)
- Text rendering via `ebitenui/widget.(*Text).Render` (21.08s)
- ScrollContainer rendering (20.74s)

**Root Causes:**
1. **Text widget overhead** - BBCode parsing, font rendering, glyph loading
2. **ScrollContainer complexity** - Nested containers with masked rendering
3. **Layout recalculation** - GridLayout, RowLayout, AnchorLayout every frame
4. **Widget tree depth** - Deep nesting causes recursive rendering costs

**Impact:** Every frame spends ~26% of time just drawing widgets

---

### 🔥 CRITICAL #2: Squad Info Queries (4.5s+ / 4.2%)

**Locations:**
- `guicomponents.(*GUIQueries).GetSquadInfo` - 2.46s (2.29%)
- `guicomponents.(*GUIQueries).GetFactionInfo` - 2.37s (2.21%)
- `guicomponents.(*GUIQueries).GetSquadInfoCached` - 1.48s (1.38%)

**Analysis:**

Even with "cached" queries, these functions are still expensive:

#### GetSquadInfo Breakdown (2.46s total):
```
810ms - squads.GetUnitIDsInSquad()       // ECS query per squad
855ms - combat.FindActionStateBySquadID() // ECS query per squad
258ms - combat.GetSquadFaction()          // ECS query per squad
240ms - squads.GetSquadName()             // ECS query per squad
224ms - squads.IsSquadDestroyed()         // ECS query per squad
```

**Problem:** These functions run **multiple ECS queries per squad** despite being called frequently.

#### GetFactionInfo Breakdown (2.37s total):
```
2.04s - squads.IsSquadDestroyed() loop   // Called for EVERY squad in faction!
255ms - factionManager.GetFactionSquads()
```

**Problem:** `GetFactionInfo()` checks `IsSquadDestroyed()` for **every squad in the faction** in a loop. This is O(n²) behavior.

#### GetSquadInfoCached Issues (1.48s total):
```
980ms - common.GetAttributesByIDWithTag() loop  // Still queries ECS per unit
242ms - common.GetComponentTypeByID()           // Position lookup
```

**Problem:** Even the "cached" version still performs ECS queries **per unit** for attributes.

---

### 🔴 HIGH #3: Mode Update Logic (5.12s / 4.8%)

**Location:** `gui/core/modemanager.go:102`

```go
umm.currentMode.Update(deltaTime)  // 5.12s
```

**Analysis:**
This includes all mode-specific update logic. The profile doesn't break this down further, but this likely includes:
- Input processing
- State updates
- Squad selection logic
- Combat system updates
- Animation state machines

**Impact:** Over 5 seconds per 107-second run means modes are doing too much work per frame.

---

### 🔴 HIGH #4: Mode Rendering (9.34s / 8.7%)

**Location:** `gui/core/modemanager.go:117`

```go
umm.currentMode.Render(screen)  // 9.34s
```

**Top Contributors:**
- `worldmap.(*TileRenderer).renderTile` - 139ms
- `worldmap.(*TileRenderer).applyViewportTransformWithBounds` - 128ms
- `rendering.ProcessRenderablesInSquare` - 45ms
- `coords.(*CoordinateManager).LogicalToScreen` - 40ms (14.4s cumulative!)

**Analysis:** While individual calls seem small, the **cumulative cost** shows:
- `LogicalToScreen` is called SO frequently it accumulates **14.4 seconds** total
- Tile rendering with viewport transforms is expensive
- Renderable processing happens every frame

---

### 🟡 MEDIUM #5: Input State Updates (68ms / 0.06%)

**Location:** `gui/core/modemanager.go:86`

```go
umm.updateInputState()  // 68ms
```

**Breakdown:**
```
56ms - Map copying: for k, v := range umm.inputState.KeysPressed
5ms  - Key polling: ebiten.IsKeyPressed()
```

**Problem:** Copying the `prevPressed` map every frame is inefficient:

```go
prevPressed := make(map[ebiten.Key]bool)  // Allocates every frame
for k, v := range umm.inputState.KeysPressed {
    prevPressed[k] = v  // 56ms copying!
}
```

---

### 🟡 MEDIUM #6: Mode Transitions (787ms / 0.73%)

**Location:** `gui/core/modemanager.go:74`

```go
toMode.Enter(umm.currentMode)  // 786ms
```

**Analysis:**
Mode transitions trigger expensive operations:
- UI widget reconstruction
- Cache rebuilding (squad lists, faction info)
- Container layout recalculation

This happens during `transitionToMode()` calls, which occur frequently when switching between combat/exploration/squad modes.

---

## Detailed Function Analysis

### Squad Query Functions

#### `GetSquadInfo` - 2.46s (Called VERY frequently)

```go
// gui/guicomponents/guiqueries.go:101
func (gq *GUIQueries) GetSquadInfo(squadID ecs.EntityID) *SquadInfo {
    name := squads.GetSquadName(squadID, gq.ECSManager)              // 240ms - Query
    unitIDs := squads.GetUnitIDsInSquad(squadID, gq.ECSManager)      // 810ms - Query

    // Loop over units
    for _, unitID := range unitIDs {
        attrs := common.GetAttributesByIDWithTag(...)                 // 53ms - Query per unit
    }

    squadPos := common.GetComponentTypeByID[*coords.LogicalPosition](...) // 5ms
    factionID = combat.GetSquadFaction(squadID, gq.ECSManager)       // 258ms - Query
    actionState := combat.FindActionStateBySquadID(squadID, gq.ECSManager) // 855ms - Query
    isDestroyed := squads.IsSquadDestroyed(squadID, gq.ECSManager)  // 224ms - Query
}
```

**Every call** performs **6+ ECS queries**. This is called from:
- `DetailPanelComponent.ShowSquad` (2.08s)
- `ApplyFilterToSquads` (266ms)
- Squad selection/display logic

---

#### `GetFactionInfo` - 2.37s (Nested loop disaster)

```go
// gui/guicomponents/guiqueries.go:52
func (gq *GUIQueries) GetFactionInfo(factionID ecs.EntityID) *FactionInfo {
    factionData := combat.FindFactionDataByID(factionID, gq.ECSManager)  // 41ms
    currentMana, maxMana := gq.factionManager.GetFactionMana(factionID)  // 33ms
    squadIDs := gq.factionManager.GetFactionSquads(factionID)            // 255ms

    // THIS IS THE KILLER:
    aliveCount := 0
    for _, squadID := range squadIDs {  // For EACH squad in faction
        if !squads.IsSquadDestroyed(squadID, gq.ECSManager) {  // 2.04s TOTAL!
            aliveCount++
        }
    }
}
```

**Problem:** If a faction has 10 squads, this calls `IsSquadDestroyed` **10 times**, and each call queries the ECS. This is called from `DetailPanelComponent.ShowFaction` which runs **every frame**.

---

#### `GetSquadInfoCached` - 1.48s (Not cached enough!)

```go
// gui/guicomponents/guiqueries.go:426
func (gq *GUIQueries) GetSquadInfoCached(squadID ecs.EntityID, cache *SquadInfoCache) *SquadInfo {
    // These are O(1) map lookups - GOOD
    name := cache.squadNames[squadID]           // 3ms
    unitIDs := cache.squadMembers[squadID]      // 2ms
    factionID := cache.squadFactions[squadID]   // 1ms

    // THIS is still expensive:
    for _, unitID := range unitIDs {
        attrs := common.GetAttributesByIDWithTag(...)  // 980ms - STILL QUERYING ECS!
    }

    squadPos := common.GetComponentTypeByID[*coords.LogicalPosition](...) // 242ms
}
```

**Problem:** The cache stores squad-level data but **still queries** unit attributes every time.

---

### Widget/UI Functions

#### Container Rendering - 26.44s

```
ebitenui/widget.(*Container).Render      - 26.44s
  └─ ebitenui/widget.(*Text).Render      - 21.08s
      └─ ebitenui/widget.(*Text).draw    - 21.06s
          └─ golang.org/x/image/font.MeasureString - 16.98s
              └─ github.com/golang/freetype/truetype.(*GlyphBuf).Load - 16.97s
```

**Analysis:** Font/glyph loading is extremely expensive. Every text widget triggers:
1. BBCode parsing (`handleBBCodeColor` - 740ms)
2. Font face lookups
3. Glyph buffer loading from TrueType fonts
4. String measurement for layout

---

## Recommendations (Prioritized)

### 🏆 PRIORITY 1: Cache Unit Attributes in SquadInfoCache

**Location:** `gui/guicomponents/guiqueries.go`

**Current:**
```go
// BuildSquadInfoCache doesn't cache unit attributes
cache := &SquadInfoCache{
    squadNames:      make(map[ecs.EntityID]string),
    squadMembers:    make(map[ecs.EntityID][]ecs.EntityID),
    // ...missing unit attributes!
}
```

**Proposed:**
```go
type SquadInfoCache struct {
    squadNames      map[ecs.EntityID]string
    squadMembers    map[ecs.EntityID][]ecs.EntityID

    // ADD THESE:
    unitAttributes  map[ecs.EntityID]*common.AttributesData  // Per-unit cache
    squadHP         map[ecs.EntityID]struct{ current, max int }
    squadAliveCount map[ecs.EntityID]int
}

func (gq *GUIQueries) BuildSquadInfoCache() *SquadInfoCache {
    cache := &SquadInfoCache{
        // ...existing maps...
        unitAttributes:  make(map[ecs.EntityID]*common.AttributesData),
        squadHP:         make(map[ecs.EntityID]struct{ current, max int }),
        squadAliveCount: make(map[ecs.EntityID]int),
    }

    // Cache unit attributes ONCE
    for _, result := range gq.squadMemberView.Get() {
        unitID := result.Entity.GetID()
        attrs := common.GetComponentType[*common.AttributesData](result.Entity, common.AttributesComponent)
        cache.unitAttributes[unitID] = attrs
    }

    // Pre-calculate squad HP/alive counts
    for squadID, unitIDs := range cache.squadMembers {
        alive, totalHP, maxHP := 0, 0, 0
        for _, unitID := range unitIDs {
            if attrs := cache.unitAttributes[unitID]; attrs != nil {
                if attrs.CanAct { alive++ }
                totalHP += attrs.CurrentHealth
                maxHP += attrs.MaxHealth
            }
        }
        cache.squadAliveCount[squadID] = alive
        cache.squadHP[squadID] = struct{ current, max int }{totalHP, maxHP}
    }

    return cache
}
```

**Expected Gain:** Eliminate **980ms** from `GetSquadInfoCached`, plus **53ms** from `GetSquadInfo` loops.

---

### 🏆 PRIORITY 2: Batch IsSquadDestroyed Checks in GetFactionInfo

**Location:** `gui/guicomponents/guiqueries.go:52`

**Current Problem:**
```go
for _, squadID := range squadIDs {  // 2.04s total
    if !squads.IsSquadDestroyed(squadID, gq.ECSManager) {
        aliveCount++
    }
}
```

**Solution:** Use the cache's `destroyedStatus` map:

```go
func (gq *GUIQueries) GetFactionInfo(factionID ecs.EntityID) *FactionInfo {
    // ... existing code ...

    // Use cache instead of queries
    cache := gq.BuildSquadInfoCache()  // Or pass cache from caller
    aliveCount := 0
    for _, squadID := range squadIDs {
        if !cache.destroyedStatus[squadID] {  // O(1) map lookup!
            aliveCount++
        }
    }
}
```

**Expected Gain:** Eliminate **2.04 seconds** from faction info queries.

---

### 🏆 PRIORITY 3: Reduce EbitenUI Text Widget Usage

**Location:** Across all GUI modes

**Problem:** Text widgets are **extremely expensive** (21s / 19.6% of total time).

**Solutions:**

#### A. Use Pre-rendered Text Images
For static/infrequent text, render once to an image and reuse:

```go
type CachedTextImage struct {
    text      string
    image     *ebiten.Image
    dirty     bool
}

func (cti *CachedTextImage) Update(newText string) {
    if newText != cti.text {
        cti.text = newText
        cti.dirty = true
    }
}

func (cti *CachedTextImage) Render(screen *ebiten.Image, x, y int) {
    if cti.dirty {
        cti.image = renderTextToImage(cti.text)  // Expensive, but rare
        cti.dirty = false
    }
    screen.DrawImage(cti.image, &ebiten.DrawImageOptions{...})  // Cheap!
}
```

#### B. Reduce BBCode Usage
BBCode parsing adds **740ms** overhead. Use simpler formatting where possible.

#### C. Batch Text Updates
Instead of updating text widgets every frame, only update when values actually change.

**Expected Gain:** Reduce text rendering from **21s to ~10s** (50% improvement).

---

### 🏆 PRIORITY 4: Optimize Input State Copying

**Location:** `gui/core/modemanager.go:149`

**Current:**
```go
prevPressed := make(map[ebiten.Key]bool)  // Allocates every frame
for k, v := range umm.inputState.KeysPressed {
    prevPressed[k] = v  // 56ms copying
}
```

**Solution:** Use a field instead of local variable:

```go
type UIModeManager struct {
    inputState        InputState
    prevKeysPressed   map[ebiten.Key]bool  // Add field
    // ...
}

func (umm *UIModeManager) updateInputState() {
    // Swap instead of copy
    umm.prevKeysPressed, umm.inputState.KeysPressed = umm.inputState.KeysPressed, umm.prevKeysPressed

    // Clear the "new" map (which was previous frame's data)
    for k := range umm.inputState.KeysPressed {
        delete(umm.inputState.KeysPressed, k)
    }

    // Update keys
    for _, key := range keysToTrack {
        isPressed := ebiten.IsKeyPressed(key)
        umm.inputState.KeysPressed[key] = isPressed
        wasPressed := umm.prevKeysPressed[key]
        umm.inputState.KeysJustPressed[key] = isPressed && !wasPressed
    }
}
```

**Expected Gain:** Eliminate **56ms** allocation overhead.

---

### 🥉 PRIORITY 5: Cache LogicalToScreen Conversions

**Location:** `coords.(*CoordinateManager).LogicalToScreen` - **14.4s cumulative**

**Problem:** This function is called extremely frequently (accumulates 14.4s total time).

**Solution:** Cache recent conversions:

```go
type CoordinateManager struct {
    // ...existing fields...
    screenPosCache map[LogicalPosition]PixelPosition
    cacheViewportHash uint64  // Invalidate cache when viewport moves
}

func (cm *CoordinateManager) LogicalToScreen(logicalPos LogicalPosition) PixelPosition {
    viewportHash := hashViewport(cm.currentViewport)

    if viewportHash != cm.cacheViewportHash {
        cm.screenPosCache = make(map[LogicalPosition]PixelPosition)  // Clear cache
        cm.cacheViewportHash = viewportHash
    }

    if cached, exists := cm.screenPosCache[logicalPos]; exists {
        return cached
    }

    result := cm.logicalToScreenUncached(logicalPos)
    cm.screenPosCache[logicalPos] = result
    return result
}
```

**Expected Gain:** Reduce coordinate conversion overhead by **50-80%** (7-11 seconds).

---

## Summary Table

| Bottleneck | Current Cost | Fix Priority | Expected Gain | Difficulty |
|------------|--------------|--------------|---------------|------------|
| EbitenUI Text Rendering | 21.08s (19.7%) | 🏆 High | ~10s (50%) | Medium |
| GetFactionInfo IsDestroyed Loop | 2.37s (2.2%) | 🏆 Critical | ~2s (85%) | Easy |
| GetSquadInfo Unit Attributes | 2.46s (2.3%) | 🏆 Critical | ~1s (40%) | Easy |
| GetSquadInfoCached Attrs | 1.48s (1.4%) | 🏆 Critical | ~1s (65%) | Easy |
| LogicalToScreen Calls | 14.4s (13.4%) | 🥉 Medium | ~7-11s (50-80%) | Medium |
| Input State Map Copy | 56ms (0.05%) | 🥉 Low | 56ms (100%) | Easy |

**Total Potential Savings:** **21-25 seconds** (19-23% of total runtime)

---

## Implementation Order

1. **Week 1 - Easy Wins (Cache Improvements)**
   - Add unit attributes to SquadInfoCache
   - Fix GetFactionInfo to use destroyedStatus cache
   - Optimize input state copying

2. **Week 2 - Coordinate Caching**
   - Implement LogicalToScreen caching
   - Profile to verify gains

3. **Week 3 - Text Widget Optimization**
   - Implement cached text images
   - Reduce BBCode usage
   - Batch text updates

4. **Week 4 - Profile & Iterate**
   - Re-profile with new benchmarks
   - Identify remaining bottlenecks
   - Consider ebitenui alternatives if text rendering still too slow

---

## Files to Modify

### Critical Changes:
- `gui/guicomponents/guiqueries.go` - Cache unit attributes, fix faction queries
- `gui/core/modemanager.go` - Optimize input state handling
- `coords/CoordinateManager.go` - Add position caching

### Medium Changes:
- Various GUI modes - Implement cached text rendering
- `gui/guicomponents/guicomponents.go` - Optimize component refresh logic

---

## Notes

- The ECS query system is already using Views (cached queries), but component lookups by ID still have overhead
- The EntityID caching optimization mentioned in CLAUDE.md would help further reduce lookup costs
- Consider creating a "GUI frame cache" that's rebuilt once per frame and reused by all queries
