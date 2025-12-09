# EbitenUI Widget Rendering Optimization Plan
**Date:** 2025-12-08
**Target:** Reduce EbitenUI rendering overhead from 27.67s (25.8%) to <10s (<9%)
**Expected Savings:** ~17 seconds (15-16% of total runtime)

---

## Problem Analysis

### Current Performance Profile

```
ebitenui.(*UI).Draw()                                    27.67s (25.8%)
├─ ebitenui/widget.(*Container).Render                   26.44s (24.7%)
│  ├─ ebitenui/widget.(*Text).Render                     21.08s (19.7%)
│  │  └─ ebitenui/widget.(*Text).draw                    21.06s (19.6%)
│  │     └─ golang.org/x/image/font.MeasureString        16.98s (15.8%)
│  │        └─ freetype/truetype.(*GlyphBuf).Load        16.97s (15.8%)
│  ├─ ebitenui/widget.(*ScrollContainer).Render          20.74s (19.3%)
│  │  └─ ebitenui/widget.(*ScrollContainer).renderContent 20.25s (18.9%)
│  │     └─ ebitenui/image.(*MaskedRenderBuffer).Draw    20.36s (19.0%)
│  └─ ebitenui/widget.(*TextArea).Render                 19.12s (17.8%)
└─ Layout Calculations
   ├─ ebitenui/widget.(*GridLayout).Layout               848ms (0.79%)
   ├─ ebitenui/widget.(*RowLayout).layout                1.54s (1.44%)
   └─ ebitenui/widget.(*AnchorLayout).Layout             922ms (0.86%)
```

### Root Causes

1. **Text Widget Overhead** (21.08s)
   - TrueType font glyph loading: 16.97s
   - BBCode parsing: 740ms
   - String measurement for layout: 16.98s
   - Font face lookups and caching misses

2. **ScrollContainer Complexity** (20.74s)
   - Masked rendering (clipping): 20.36s
   - Nested container recursion
   - Content measurement and scrollbar positioning

3. **TextArea Widgets** (19.12s)
   - Combines text rendering + scroll container overhead
   - Used heavily in detail panels, log displays, squad info

4. **Layout Recalculation** (3.3s total)
   - GridLayout, RowLayout, AnchorLayout run every frame
   - Widget tree traversal for size calculations
   - No dirty-flag optimization

---

## Optimization Strategy

### Phase 1: Quick Wins (Low-Hanging Fruit)
**Timeline:** Week 1
**Expected Savings:** 5-7 seconds
**Difficulty:** Easy

### Phase 2: Caching Layer
**Timeline:** Week 2-3
**Expected Savings:** 8-10 seconds
**Difficulty:** Medium

### Phase 3: Architectural Changes
**Timeline:** Week 4+
**Expected Savings:** 2-5 seconds
**Difficulty:** Hard

---

## Phase 1: Quick Wins (Week 1)

### 1.1: Reduce BBCode Usage (Save ~700ms)

**Problem:** BBCode parsing adds 740ms overhead for color/formatting tags.

**Current Usage Pattern:**
```go
// gui/guicomponents/guicomponents.go
func DefaultSquadFormatter(data interface{}) string {
    info := data.(*SquadInfo)
    return fmt.Sprintf(`[color=yellow]%s[/color]
Units: %d/%d
HP: [color=green]%d[/color]/[color=red]%d[/color]
...`, info.Name, ...)
}
```

**Solution:** Use plain text where possible, apply color via widget options:

```go
// Simple text - no BBCode parsing
func DefaultSquadFormatterPlain(data interface{}) string {
    info := data.(*SquadInfo)
    return fmt.Sprintf(`%s
Units: %d/%d
HP: %d/%d
...`, info.Name, ...)
}

// Apply color via widget configuration
textWidget := widget.NewText(
    widget.TextOpts.Text(..., face, color.White),  // Base color
)
```

**Impact:**
- Eliminate BBCode parsing: ~740ms saved
- Simpler string formatting
- Trade-off: Less colorful UI (acceptable for performance)

---

### 1.2: Update Text Only When Changed (Save ~5s)

**Problem:** Text widgets re-render every frame even when text hasn't changed.

**Current Pattern:**
```go
// Called EVERY frame in Update()
func (dpc *DetailPanelComponent) ShowSquad(squadID ecs.EntityID) {
    squadInfo := dpc.queries.GetSquadInfo(squadID)
    dpc.textWidget.Label = DefaultSquadFormatter(squadInfo)  // Always updates!
}
```

**Solution:** Track previous text and only update when changed:

```go
type DetailPanelComponent struct {
    textWidget    *widget.Text
    queries       *GUIQueries
    formatter     func(interface{}) string

    // Add tracking fields
    lastSquadID   ecs.EntityID
    lastText      string
    dirty         bool
}

func (dpc *DetailPanelComponent) ShowSquad(squadID ecs.EntityID) {
    if dpc.textWidget == nil {
        return
    }

    // Check if squad changed
    if squadID == dpc.lastSquadID && !dpc.dirty {
        return  // No update needed!
    }

    squadInfo := dpc.queries.GetSquadInfo(squadID)
    if squadInfo == nil {
        newText := "Squad not found"
        if newText != dpc.lastText {
            dpc.textWidget.Label = newText
            dpc.lastText = newText
        }
        return
    }

    newText := dpc.formatter(squadInfo)
    if newText != dpc.lastText {
        dpc.textWidget.Label = newText
        dpc.lastText = newText
        dpc.lastSquadID = squadID
        dpc.dirty = false
    }
}

func (dpc *DetailPanelComponent) MarkDirty() {
    dpc.dirty = true
}
```

**When to Mark Dirty:**
- Combat actions (attack, move)
- Turn changes
- HP changes
- Status effect changes

**Impact:**
- Most frames skip text updates entirely
- Text re-rendering only when data actually changes
- Estimated savings: ~5s (assuming 10% update rate vs 100%)

---

### 1.3: Reduce TextArea Usage (Save ~2s)

**Problem:** TextArea widgets (19.12s) are expensive due to ScrollContainer + Text combo.

**Current Usage:**
- Detail panels (squad info, faction info)
- Log display
- Info displays

**Solution A:** Use simple Text widgets where scrolling isn't needed:

```go
// BEFORE: TextArea with scroll container overhead
textArea := widgets.CreateTextAreaWithConfig(widgets.TextAreaConfig{
    MinWidth:  panelWidth - 20,
    MinHeight: panelHeight - 20,
    FontColor: color.White,
})

// AFTER: Simple Text widget (no scroll overhead)
text := widget.NewText(
    widget.TextOpts.Text("", face, color.White),
    widget.TextOpts.Position(widget.TextPositionStart, widget.TextPositionCenter),
    widget.TextOpts.WidgetOpts(
        widget.WidgetOpts.MinSize(panelWidth-20, panelHeight-20),
    ),
)
```

**Solution B:** Pre-truncate text content:

```go
func TruncateSquadInfo(info *SquadInfo, maxLines int) string {
    lines := []string{
        info.Name,
        fmt.Sprintf("Units: %d/%d", info.AliveUnits, info.TotalUnits),
        fmt.Sprintf("HP: %d/%d", info.CurrentHP, info.MaxHP),
        // ... up to maxLines
    }
    if len(lines) > maxLines {
        lines = lines[:maxLines]
    }
    return strings.Join(lines, "\n")
}
```

**Apply to:**
- Squad detail panel (right side) - rarely needs scrolling
- Faction info panel - static size
- Status displays - fixed content

**Keep TextArea only for:**
- Combat log (actually needs scrolling)
- Long help text
- Dynamic content that exceeds panel size

**Impact:** Reduce TextArea overhead by ~50% (~10s → ~5s)

---

### 1.4: Disable Layout Recalculation When Inactive (Save ~1-2s)

**Problem:** Layout calculations (GridLayout, RowLayout, etc.) run every frame even when UI is static.

**Current Behavior:**
```go
// ebitenui automatically recalculates layout every frame
container.Render(screen)  // Triggers layout recalc
```

**Solution:** Use ebitenui's built-in dirty flag system (if available) or implement custom:

```go
type CachedContainer struct {
    *widget.Container
    layoutDirty   bool
    lastRenderTime time.Time
}

func (cc *CachedContainer) MarkDirty() {
    cc.layoutDirty = true
}

// Override Render to skip layout when clean
func (cc *CachedContainer) Render(screen *ebiten.Image) {
    if cc.layoutDirty || time.Since(cc.lastRenderTime) > time.Second {
        // Only recalc layout when dirty or periodically
        cc.Container.Render(screen)
        cc.layoutDirty = false
        cc.lastRenderTime = time.Now()
    } else {
        // Just render without layout recalc
        cc.renderWithoutLayout(screen)
    }
}
```

**Impact:** Reduce layout overhead by 50-70% (~3.3s → ~1s)

---

## Phase 2: Caching Layer (Week 2-3)

### 2.1: Pre-rendered Text Cache (Save ~10s)

**Concept:** Render text to images once, reuse the image until text changes.

**Implementation:**

```go
// pkg: gui/rendering/textcache.go

type TextCache struct {
    cache map[string]*CachedText
    mu    sync.RWMutex
}

type CachedText struct {
    text      string
    image     *ebiten.Image
    width     int
    height    int
    timestamp time.Time
}

func NewTextCache() *TextCache {
    return &TextCache{
        cache: make(map[string]*CachedText),
    }
}

func (tc *TextCache) GetOrRender(text string, face font.Face, clr color.Color) *ebiten.Image {
    tc.mu.RLock()
    cached, exists := tc.cache[text]
    tc.mu.RUnlock()

    if exists && time.Since(cached.timestamp) < 5*time.Second {
        return cached.image  // Return cached image
    }

    // Cache miss - render to image
    img := tc.renderTextToImage(text, face, clr)

    tc.mu.Lock()
    tc.cache[text] = &CachedText{
        text:      text,
        image:     img,
        timestamp: time.Now(),
    }
    tc.mu.Unlock()

    return img
}

func (tc *TextCache) renderTextToImage(text string, face font.Face, clr color.Color) *ebiten.Image {
    // Measure text bounds
    bounds := text.BoundString(face, text)
    width := bounds.Max.X - bounds.Min.X
    height := bounds.Max.Y - bounds.Min.Y

    // Create image
    img := ebiten.NewImage(width, height)

    // Render text once
    ebitenutil.DrawText(img, text, face, clr)

    return img
}

func (tc *TextCache) Clear() {
    tc.mu.Lock()
    defer tc.mu.Unlock()
    tc.cache = make(map[string]*CachedText)
}

func (tc *TextCache) Invalidate(text string) {
    tc.mu.Lock()
    defer tc.mu.Unlock()
    delete(tc.cache, text)
}
```

**Usage in Components:**

```go
type DetailPanelComponent struct {
    textCache     *TextCache
    cachedImage   *ebiten.Image
    lastText      string
    position      image.Point
}

func (dpc *DetailPanelComponent) ShowSquad(squadID ecs.EntityID) {
    squadInfo := dpc.queries.GetSquadInfo(squadID)
    newText := dpc.formatter(squadInfo)

    if newText != dpc.lastText {
        dpc.cachedImage = dpc.textCache.GetOrRender(newText, guiresources.SmallFace, color.White)
        dpc.lastText = newText
    }
}

func (dpc *DetailPanelComponent) Render(screen *ebiten.Image) {
    if dpc.cachedImage != nil {
        opts := &ebiten.DrawImageOptions{}
        opts.GeoM.Translate(float64(dpc.position.X), float64(dpc.position.Y))
        screen.DrawImage(dpc.cachedImage, opts)
    }
}
```

**Impact:**
- First render: Expensive (same as before)
- Subsequent renders: Cheap image blit (~0.1ms vs ~200ms per text widget)
- Estimated savings: ~10s (50% of text rendering cost)

**Trade-offs:**
- Memory usage: ~10-20MB for cached text images
- Complexity: Need to manage cache invalidation
- Text updates slightly delayed (need explicit invalidation)

---

### 2.2: Widget Pooling (Save ~1-2s)

**Problem:** Creating/destroying widgets causes allocation overhead.

**Solution:** Reuse widget instances from a pool:

```go
type WidgetPool struct {
    buttons    []*widget.Button
    texts      []*widget.Text
    containers []*widget.Container
    inUse      map[widget.PreferredSizeLocateableWidget]bool
}

func (wp *WidgetPool) GetButton() *widget.Button {
    for _, btn := range wp.buttons {
        if !wp.inUse[btn] {
            wp.inUse[btn] = true
            return btn
        }
    }

    // Pool empty - create new
    btn := widget.NewButton(...)
    wp.buttons = append(wp.buttons, btn)
    wp.inUse[btn] = true
    return btn
}

func (wp *WidgetPool) Release(w widget.PreferredSizeLocateableWidget) {
    wp.inUse[w] = false
}
```

**Apply to:**
- Squad list buttons (created/destroyed frequently)
- Action buttons
- Temporary UI elements

**Impact:** Reduce allocation overhead by ~50% (~1-2s)

---

### 2.3: Lazy Rendering for Off-Screen Widgets (Save ~3s)

**Problem:** All widgets render even if off-screen or hidden.

**Solution:** Implement viewport culling:

```go
type SmartContainer struct {
    *widget.Container
    viewport image.Rectangle  // Screen bounds
}

func (sc *SmartContainer) Render(screen *ebiten.Image) {
    screenBounds := screen.Bounds()

    for _, child := range sc.Children() {
        childBounds := child.GetWidget().Rect

        // Check if widget is visible
        if childBounds.Overlaps(screenBounds) {
            child.Render(screen)  // Render visible widgets only
        }
    }
}
```

**Apply to:**
- ScrollContainer content (only render visible portion)
- Long lists (squad list, item list)
- Off-screen panels

**Impact:** Reduce rendering by 30-50% for scrollable content (~3s)

---

## Phase 3: Architectural Changes (Week 4+)

### 3.1: Replace TextArea with Custom Renderer

**Concept:** Implement lightweight text display without ebitenui overhead.

```go
type LightweightTextDisplay struct {
    text       string
    lines      []string
    face       font.Face
    color      color.Color
    position   image.Point
    bounds     image.Rectangle
    textImage  *ebiten.Image
    dirty      bool
}

func (ltd *LightweightTextDisplay) SetText(text string) {
    if text != ltd.text {
        ltd.text = text
        ltd.lines = strings.Split(text, "\n")
        ltd.dirty = true
    }
}

func (ltd *LightweightTextDisplay) Update() {
    if ltd.dirty {
        ltd.renderToImage()
        ltd.dirty = false
    }
}

func (ltd *LightweightTextDisplay) renderToImage() {
    // Calculate required size
    maxWidth := 0
    totalHeight := 0
    for _, line := range ltd.lines {
        bounds := text.BoundString(ltd.face, line)
        width := bounds.Max.X - bounds.Min.X
        if width > maxWidth {
            maxWidth = width
        }
        totalHeight += bounds.Max.Y - bounds.Min.Y
    }

    // Create image
    ltd.textImage = ebiten.NewImage(maxWidth, totalHeight)

    // Render lines
    y := 0
    for _, line := range ltd.lines {
        text.Draw(ltd.textImage, line, ltd.face, 0, y, ltd.color)
        y += ltd.face.Metrics().Height.Ceil()
    }
}

func (ltd *LightweightTextDisplay) Render(screen *ebiten.Image) {
    if ltd.textImage != nil {
        opts := &ebiten.DrawImageOptions{}
        opts.GeoM.Translate(float64(ltd.position.X), float64(ltd.position.Y))
        screen.DrawImage(ltd.textImage, opts)
    }
}
```

**Migration Path:**
1. Identify TextArea widgets that don't need scrolling
2. Replace with LightweightTextDisplay
3. Keep TextArea only where scrolling is essential

**Impact:** Reduce text rendering overhead by 70-80% (~15s → ~3-5s)

---

### 3.2: Consider Alternative UI Libraries

**If ebitenui remains too slow after all optimizations:**

**Option A: Hybrid Approach**
- Keep ebitenui for complex widgets (buttons, scroll containers)
- Use custom rendering for text-heavy displays

**Option B: Custom UI Framework**
- Implement minimal UI system tailored to game needs
- Full control over rendering pipeline
- Higher development cost but maximum performance

**Option C: Evaluate Alternatives**
- [ebitenui-extended](https://github.com/ebitenui/ebitenui/discussions) - Check for performance-focused forks
- Custom immediate-mode UI (like Dear ImGui pattern)

---

## Implementation Checklist

### Week 1: Quick Wins
- [ ] Audit all BBCode usage, remove where unnecessary
- [ ] Implement text change tracking in DetailPanelComponent
- [ ] Implement text change tracking in SquadListComponent
- [ ] Implement text change tracking in StatsDisplayComponent
- [ ] Replace TextArea with Text widget where scrolling not needed
- [ ] Add dirty flags to layout-heavy containers
- [ ] Benchmark: Measure savings (target: 5-7s reduction)

### Week 2: Text Cache Implementation
- [ ] Create TextCache package (`gui/rendering/textcache.go`)
- [ ] Implement GetOrRender with TTL-based invalidation
- [ ] Integrate TextCache into DetailPanelComponent
- [ ] Integrate TextCache into combat log
- [ ] Integrate TextCache into squad list
- [ ] Add cache invalidation on game state changes
- [ ] Benchmark: Measure savings (target: 8-10s additional reduction)

### Week 3: Advanced Caching
- [ ] Implement WidgetPool for buttons and text widgets
- [ ] Implement viewport culling for ScrollContainers
- [ ] Add lazy rendering to squad list
- [ ] Add lazy rendering to combat log
- [ ] Benchmark: Measure savings (target: 3-4s additional reduction)

### Week 4: Evaluate and Iterate
- [ ] Re-profile with new benchmark
- [ ] Measure total reduction (target: >15s saved)
- [ ] Identify remaining bottlenecks
- [ ] Decide on Phase 3 approaches based on results

---

## Testing Strategy

### Performance Testing

```go
// benchmark_test.go
func BenchmarkTextWidgetRendering(b *testing.B) {
    // Setup
    screen := ebiten.NewImage(800, 600)
    textWidget := widget.NewText(...)
    textWidget.Label = "Sample squad info..."

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        textWidget.Render(screen)
    }
}

func BenchmarkCachedTextRendering(b *testing.B) {
    screen := ebiten.NewImage(800, 600)
    cache := NewTextCache()
    text := "Sample squad info..."

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        img := cache.GetOrRender(text, guiresources.SmallFace, color.White)
        screen.DrawImage(img, &ebiten.DrawImageOptions{})
    }
}
```

### Regression Testing

- [ ] Verify all UI elements still display correctly
- [ ] Test text updates trigger properly
- [ ] Verify cache invalidation works on state changes
- [ ] Check memory usage doesn't exceed 50MB for cache
- [ ] Ensure no visual glitches or rendering artifacts

---

## Success Metrics

### Performance Goals
- **Target:** Reduce rendering from 27.67s to <10s
- **Minimum Acceptable:** <15s (>45% improvement)
- **Stretch Goal:** <7s (>75% improvement)

### Per-Phase Targets
| Phase | Target Reduction | Cumulative Total |
|-------|------------------|------------------|
| Phase 1 (Quick Wins) | 5-7s | 20-22s remaining |
| Phase 2 (Caching) | 10-12s | 8-12s remaining |
| Phase 3 (Architectural) | 3-5s | <7s remaining |

### Quality Metrics
- No visual regression
- Text updates within 1 frame
- Memory usage <100MB total (including cache)
- No UI lag or stuttering

---

## Rollback Plan

If optimizations cause issues:

1. **Phase 1 Rollback:**
   - Revert text change tracking (restore always-update behavior)
   - Re-enable BBCode if visual quality suffers
   - Restore TextArea widgets if text truncation causes issues

2. **Phase 2 Rollback:**
   - Disable TextCache (keep rendering per-frame)
   - Remove widget pooling
   - Restore normal rendering (no viewport culling)

3. **Phase 3 Rollback:**
   - Restore ebitenui TextArea widgets
   - Abandon custom UI framework

**Git Strategy:** Create feature branches for each phase, merge only after benchmarks confirm improvement.

---

## References

- **Profile Data:** `docs/benchmarking/benchmark_1/view_benchmark.pb.gz`
- **Analysis:** `docs/benchmarking/benchmark_1/gui_performance_analysis.md`
- **EbitenUI Docs:** https://github.com/ebitenui/ebitenui
- **Ebiten Performance Guide:** https://ebiten.org/documents/performancetips.html
- **TrueType Font Optimization:** Consider bitmap fonts for fixed-size text

---

## Notes

- Text rendering dominates because TrueType fonts require expensive glyph rasterization
- Bitmap fonts would be faster but less flexible (fixed size, aliasing)
- Pre-rendering to images trades CPU time for memory
- Most GUI text is semi-static (changes only on game events, not every frame)
- Combat log is the exception - highly dynamic, needs special handling
