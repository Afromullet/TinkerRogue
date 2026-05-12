# Technical Debt Analysis: `visual/{graphics, maprender, rendering, vfx}`

**Date:** 2026-05-12
**Scope:** 14 files, 2087 LOC (production only — **zero test files in any of the four packages**)
**Status:** Functional and performance-aware (batching, caching), but with significant dead code in `vfx`, broken design contracts in `graphics.ColorMatrix`, and zero test coverage across the rendering hot path.

---

## Verdict at a Glance

The four packages divide cleanly:
- **`graphics`** (592 LOC) — value types: `ColorMatrix`, `BaseShape`, color factories, screen/coord re-exports.
- **`rendering`** (400 LOC) — low-level `QuadBatch`, `RenderingCache`, `ViewportRenderer`. Solid.
- **`maprender`** (277 LOC) — tile batching with viewport caching. Clean.
- **`vfx`** (818 LOC) — visual effect pipeline (`BaseEffect` + `Animator` + `Renderer`). **~50% dead code.**

The strongest part is `rendering` (well-structured batching + cache invalidation). The weakest is `vfx`: a clean strategy-pattern refactor was done, but the legacy effect functions and their backing renderers/animators were never deleted, even though only `NewFireEffect`, `NewIceEffect`, `NewElectricityEffect`, and `NewCloudEffect` remain reachable from outside the package.

The single most worrying finding is **`ColorMatrix.ApplyMatrix` is set everywhere but read nowhere** — the gate flag the type advertises does not exist in code, only in docs.

---

## 1. Debt Inventory

### A. Dead Code (vfx) — HIGH priority

A 2024-era refactor split monolithic effects into composable `Animator` + `Renderer` strategies. The new generic `NewEffect(...)` factory works, but **eight types and five factories have no callers outside the vfx package**.

Verified via `Grep` against the entire repo (only `vfx.NewFireEffect`/`NewIceEffect`/`NewElectricityEffect`/`NewCloudEffect`/`CreateVisualEffectByType`/`VXHandler`/`AddVX`/`AddVXArea`/`NewVisualEffectArea` appear in non-vfx code):

| Symbol | File | Status |
|---|---|---|
| `NewIceEffect2` | `vxfactory.go:66` | Unused |
| `NewProjectile` | `vxfactory.go:119` | Unused |
| `NewElectricityEffectNoImage` | `vxfactory.go:128` | Unused |
| `NewElectricArc` | `vxfactory.go:152` | Unused |
| `NewStickyGroundEffect` | `vxfactory.go:108` | Unused |
| `SineShimmerAnimator` | `animators.go:69` | Only referenced by `NewIceEffect2` |
| `MotionAnimator` + `NewMotionAnimator` | `animators.go:117` | Only referenced by `NewProjectile` |
| `WaveAnimator` | `animators.go:171` | Only referenced by `NewStickyGroundEffect` |
| `ProjectileRenderer` | `renderers.go:42` | Only referenced by `NewProjectile` |
| `LineSegmentRenderer` + `NewLineSegmentRenderer` | `renderers.go:108` | Only referenced by `NewElectricityEffectNoImage` |
| `ElectricArcRenderer` + `NewElectricArcRenderer` | `renderers.go:160` | Only referenced by `NewElectricArc` |
| `ProceduralRenderer` + `NewProceduralRenderer` | `renderers.go:211` | Only referenced by `NewStickyGroundEffect` |

Combined: **~410 LOC of code that goes nowhere**. The vfx package would shrink from 818 LOC to ~408 LOC (~50% reduction) by deletion.

The refactor's reusable scaffolding (`BaseEffect`, `RandomAnimator`, `PulseAnimator`, `ImageRenderer`, `CloudRenderer`) is what production actually uses; the rest is preserved-just-in-case. There is no pending work order or design doc that depends on these.

### B. Broken Contract — `ColorMatrix.ApplyMatrix` (HIGH risk, low blast)

`graphics/colormatrix.go:10`:
```go
type ColorMatrix struct {
    R, G, B, A  float32
    ApplyMatrix bool
}
```

`ApplyMatrix` is set by every constructor and re-exported in docs (`VISUAL_PACKAGES.md:253-257`: *"`ApplyMatrix` is a gate flag. When false, the tile renderer skips color application entirely"*). **No code reads it.** The renderer (`maprender/tilerenderer.go:135`) gates on `cm.IsEmpty()`, which only inspects RGBA == 0:

```go
if !tile.GetColorMatrix().IsEmpty() {
    cm := tile.GetColorMatrix()
    colorR, colorG, colorB, colorA = cm.R, cm.G, cm.B, cm.A
}
```

Consequences:
- `NewEmptyMatrix()` sets `ApplyMatrix: true` with RGBA=0 — `IsEmpty()` returns true, so the matrix is skipped. Works by accident.
- `Tile.NewTile` initializes `ApplyMatrix: false` — also irrelevant.
- A legitimate `{R:0, G:0, B:0, A:0.5}` (black 50% tint) would be skipped by `IsEmpty()` even though it's a real tint.

**Fix shape:** either honor `ApplyMatrix` in `IsEmpty()` (`return !c.ApplyMatrix || (c.R==0 && ...)`), or delete the field. Deleting is the right call — no consumer needs three states (off/empty/on); two suffice.

### C. Dead Helpers — `graphics`

| Symbol | File | Callers |
|---|---|---|
| `GetLineTo` | `drawableshapes.go:399` | None — doc claims "used for line-of-sight and line-based spell targeting" but `Grep` finds zero non-doc references |
| `NoDirection` enum value | `drawableshapes.go:32` | Only returned by `BaseShape.GetDirection` when `Direction==nil`; never consumed by any caller |
| `AllDirections` exported var | `drawableshapes.go:35` | Only used internally by `RotateRight`/`RotateLeft` — should be unexported |
| `LargeShape`, `MediumShape` enum values | `drawableshapes.go:14-15` | All factory call sites pass `SmallShape`. `Grep` confirms `Medium`/`LargeShape` have no callers outside the constructor switch statements |

`SmallShape` is the only `ShapeSize` ever used. The `case Medium/Large` branches in five constructors (`NewCircle`, `NewSquare`, `NewRectangle`, `NewLine`, `NewCone`) are unreachable.

### D. Re-exports That Add Indirection

`graphics/graphictypes.go:11-12`:
```go
var ScreenInfo = coords.NewScreenData()
var CoordManager = coords.CoordManager
```

These are aliases for `coords` package globals. `graphics.ScreenInfo` has 27 reads scattered across 9 files; `graphics.CoordManager` has 0 reads (callers go directly to `coords.CoordManager`). The alias serves no purpose other than letting some packages import only `graphics` instead of also importing `coords` — but the alias itself is the import dependency that makes that work, so the reduction is illusory.

`graphics.MouseToLogicalPosition` (`graphictypes.go:16-18`) is a 3-line wrapper around `coords.CoordManager.ScreenToLogical`. 7 callers depend on it; the wrapper is fine but should live in `coords` (no graphics-specific behavior).

**Net:** `graphictypes.go` is a 19-line file whose only real content is a function that belongs elsewhere. Move `MouseToLogicalPosition` to `coords` and delete `ScreenInfo`/`CoordManager` aliases — call sites update by adding one import.

### E. `BaseEffect.Copy()` — Shallow Copy Shares Mutable Animator State

`vfx/vx.go:104-108`:
```go
func (e *BaseEffect) Copy() VisualEffect {
    // Shallow copy - animators and renderers are stateless except for frame counters
    copy := *e
    return &copy
}
```

This is called from `vxhandler.go:31` to fan one effect across N tiles in a `VisualEffectArea`. The shallow copy copies the `animator Animator` interface header, **which contains a pointer to the same underlying animator struct**. So all N copies share:

- `RandomAnimator.flickerTimer` — incremented every Update → all tiles flicker in lock-step.
- `PulseAnimator.puffinessPhase` — same phase accumulation across all tiles.
- `SineShimmerAnimator.shimmerPhase` — same.
- (`MotionAnimator.currentX/Y/completed` would be catastrophic, but `MotionAnimator` is dead code.)

Lock-step animation across an AoE may be intentional (or unnoticed). The comment says "stateless except for frame counters," but frame counters are the entire animation. **At minimum, the comment is wrong and should be a deliberate decision documented on the type.** If independence is desired, `Copy()` must deep-copy the animator (each strategy needs its own `Clone()` method).

### F. `BaseEffect.UpdateVisualEffect` Doesn't Update the Animator

`vfx/vx.go:53-63`:
```go
func (e *BaseEffect) UpdateVisualEffect() {
    if e.completed { return }
    elapsed := time.Since(e.startTime).Seconds()
    if int(elapsed) >= e.duration {
        e.completed = true
    }
}
```

The animator's `Update()` is only invoked from `DrawVisualEffect`. So if a frame skips drawing (off-screen, paused), the animator state stalls — but the `completed` clock keeps ticking against wall time. For `MotionAnimator` (dead) this would mean a projectile that skipped a draw frame never advances. For the live animators (`RandomAnimator`, `PulseAnimator`) this is mostly fine because state is per-call, not cumulative.

The Update/Draw split is broken: animators that need per-tick state (any kinematic animator) cannot work in this architecture. Document this constraint or move `animator.Update()` into `UpdateVisualEffect`.

### G. Architectural Coupling — `vxhandler` reaches into `graphics` only for `BaseShape`

`vfx/vxhandler.go:4` imports `game_main/visual/graphics` solely so `NewVisualEffectArea` can accept `*graphics.BaseShape`. `BaseShape` is a tile-targeting concept tied to the spell/spatial system — coupling vfx to it means vfx can't be reused for non-tile effects. Two cleaner options:

1. Have callers (`spell_handler.go`) compute `[]coords.LogicalPosition` from the shape and pass that — vfx loses the graphics dependency entirely.
2. Add a small `type IndexProvider interface { GetIndices() []int }` and accept that.

This matters because the only outside consumer of vfx is `gui/guispells/spell_handler.go`. Removing the `graphics` import would invert the dependency to where `vfx` becomes purely a draw library.

### H. Allocation Hotspots (currently dormant — only relevant if dead code revives)

These are buried in unused code; they become real if anyone reawakens these effects:

- `ProceduralRenderer.Draw` (`renderers.go:230`): `ebiten.NewImage(int(2*radius), int(2*radius))` allocated **5 times per frame** inside the wave loop. GPU allocation in render path.
- `ElectricArcRenderer.Draw` (`renderers.go:196`): `r.segments = make([][]float64, 0)` reallocated every frame, then 11 appends.
- `LineSegmentRenderer.Draw` (`renderers.go:142-156`): mutates `segments` in place but also calls `common.GetDiceRoll(55)` 6 times per frame for color jitter.

If you delete these per item A, the hotspots vanish. Worth flagging here so a future revival doesn't quietly land in the render loop.

### I. Renderer State Smells

`vfx/renderers.go`:
- `ProjectileRenderer` and `ElectricArcRenderer` store `endX/endY` on the renderer instead of in `BaseEffect`. This is *why* `Copy()` is unsafe — the renderer pointer is shared but its embedded endpoint is per-instance. `MotionAnimator` ALSO duplicates `endX/endY`.
- The `Renderer` interface ignores `state.OffsetX/OffsetY` in `LineSegmentRenderer` and `ElectricArcRenderer`. Those renderers don't honor the contract.
- `ImageRenderer.Draw` (`renderers.go:18-39`): the brightness-vs-colorshift-vs-default branch is order-sensitive (brightness wins, then shift, then plain alpha). Three mutually exclusive code paths means the `AnimationState` struct can't represent "brightness + shift" simultaneously — a constraint not documented anywhere.

### J. `NewViewportRenderer` Has a Construction-Time Side Effect

`rendering/viewport.go:53-55`:
```go
func NewViewportRenderer(screen *ebiten.Image, centerPos coords.LogicalPosition) *ViewportRenderer {
    coords.CoordManager.UpdateScreenDimensions(screen.Bounds().Dx(), screen.Bounds().Dy())
    ...
}
```

A constructor mutating a global. Hidden coupling: anyone calling `NewViewportRenderer` for any reason silently changes the resolution recorded in the global `CoordManager`. `CachedViewport.GetRenderer` (`viewport.go:148-162`) recreates the renderer on screen-size change — fine — but every recreation re-fires this side effect. Move the call to an explicit `coords.CoordManager.SyncToScreen(screen)` and have callers invoke it once in their main loop.

### K. `RenderingCache.RefreshRenderablesView` — Documented But Not Called

`rendering/renderingcache.go:59-63`:
```go
// RefreshRenderablesView recreates the RenderablesView to force it to update
// Call this after batch entity disposal to ensure stale entities don't render
func (rc *RenderingCache) RefreshRenderablesView(manager *common.EntityManager) { ... }
```

`Grep` shows zero callers in production code, only documentation references (`CACHING_OVERVIEW.md:1581`, `VISUAL_PACKAGES.md:415`). Either:
1. The contract is silently violated (stale entities can render → bug surface), or
2. The contract no longer matters (the underlying ECS handles disposal correctly now).

Pick one. If (2), delete the method and the doc warnings. If (1), find the disposal sites and add the call (probably in `combatlifecycle/`).

### L. Naming Inconsistencies

- `LinedDiagonalDownLeft` (`drawableshapes.go:31`) — typo for `LineDiagonalDownLeft`. Fix is one find-replace.
- `AddVisualEffecArea` (`vxhandler.go:81`) — missing `t`, should be `AddVisualEffectArea`.
- `BaseShape.UpdateSize` and `BaseShape.UpdateDimensions` overlap — `UpdateSize` mutates Width/Height only when `Width==Height` (the "is this a square" heuristic at `drawableshapes.go:295`). Fragile — once a square is non-square, `UpdateSize` silently stops working on Width/Height. Use explicit shape subtypes or document the heuristic loudly.

### M. Testing Debt (CRITICAL)

```
$ ls visual/{graphics,maprender,rendering,vfx}/*_test.go
(no files)
```

Zero unit tests across all four packages. The rendering hot path runs every frame. Untested behaviors include:
- `QuadBatch.Add` vertex math (a single +/- mistake corrupts every quad in the frame).
- `BaseShape.calculateCircle/Rectangle/Line` index generation — these are pure functions, trivially testable.
- `ColorMatrix.IsEmpty` semantics — already wrong (item B), no test would have caught it.
- `RotateRight/RotateLeft` on `ShapeDirection` — pure function.
- `tilesChanged` invalidation logic — fragile pointer-comparison heuristic.
- `CachedViewport.GetRenderer` recreation conditions.

Concrete impact: when item B is fixed, there's no regression net. When the dead vfx code is deleted, no test confirms remaining factories still work.

### N. Documentation Drift

`resources/docs/project_documentation/Systems/VISUAL_PACKAGES.md` (790 LOC) describes:
- `graphics.TileBasedShape` — type does not exist (it's `BaseShape`).
- `ApplyMatrix` semantics that don't match code (item B).
- `RefreshRenderablesView` use cases that have no callers (item K).
- `GetLineTo` use cases that have no callers (item C).

The doc describes an aspirational API; the code is the actual one. Reconcile or rewrite.

---

## 2. Impact Assessment

| Item | Severity | Velocity Impact | Risk |
|---|---|---|---|
| **A** Dead vfx code (~410 LOC) | **High** | Every grep hit, every file open, every "what does this do" question | Low (harmless until awakened); High if revived (item H) |
| **B** `ApplyMatrix` is a lie | **High** | Anyone reading the type believes a gate exists | Medium — black-tint edge case is silently dropped |
| C Dead graphics helpers | Medium | Same as A but smaller surface | Low |
| D `graphics` re-exports | Low | Slight import-graph confusion | Low |
| E `Copy()` shares animator state | Medium | Anyone adding a kinematic animator to AoE will have a "why is this broken" day | Medium |
| F Update/Draw split | Low | Documented constraint missing | Low (current animators tolerate it) |
| G vfx → graphics coupling | Low | Blocks reuse | Low |
| H Allocation hotspots in dead code | Low (dormant) | None today | High if A is reverted instead of accepted |
| I Renderer state smells | Low | Refactor friction | Low |
| J Constructor side effect | Low | Surprises future testers | Medium |
| **K** `RefreshRenderablesView` orphan | Medium | Either docs lie or code has a bug | Medium — could be a stale-entity render bug |
| L Naming (`Effec`, `Lined…`) | Trivial | Annoying greps | None |
| **M** Zero tests | **High** | Every change is a leap of faith | High — silent rendering bugs are hard to spot |
| N Doc drift | Medium | New contributors get the wrong mental model | Low |

**Highest-leverage single fix:** A (delete dead vfx code). Removes ~410 LOC, kills items H and most of I, and shrinks the surface that needs the M test backfill.

**Highest-risk-per-LOC item:** B (ApplyMatrix). Three-line struct field that misleads every reader.

---

## 3. Prioritized Roadmap

### Quick Wins (≤ 1 day total)

1. **Delete dead vfx symbols (item A)** — 30 min. Confirmed by grep; deletion order: factories → renderers → animators. Verify build with `go build ./...` and `go test ./...`. Resulting `vxfactory.go` shrinks from 156 to ~80 LOC; `renderers.go` from 242 to ~95 LOC; `animators.go` from 187 to ~70 LOC.
2. **Delete or honor `ColorMatrix.ApplyMatrix` (item B)** — 30 min. Recommend deletion: remove field from struct, drop `ApplyMatrix:` from all 11 constructors and `Tile.NewTile`. `IsEmpty()` keeps current semantics but rename to `IsTransparent()` to match what it actually checks (RGBA all zero).
3. **Delete `GetLineTo`, `NoDirection`, unexport `AllDirections` (item C)** — 15 min. If any future LOS code needs it, it'll come back with tests.
4. **Fix typos: `LinedDiagonalDownLeft` → `LineDiagonalDownLeft`, `AddVisualEffecArea` → `AddVisualEffectArea` (item L)** — 5 min find/replace each. Both are exported names; verify no external consumers (this is a single-binary repo, so safe).
5. **Verify `RefreshRenderablesView` is needed (item K)** — 30 min investigation. Read `combatlifecycle/` cleanup paths; confirm whether disposed entities can leak into views. Either delete the method+docs or add a single call site.
6. **Move `MouseToLogicalPosition` to `coords` (item D)** — 20 min. 7 import lines update; pure mechanical move.
7. **Update CLAUDE.md / VISUAL_PACKAGES.md** to reflect deletions (item N) — 30 min.

Total: ~3 hours for items 1–7. Removes ~450 LOC, fixes one silent bug, and aligns docs with code.

### Medium-Term (1–3 days)

8. **Decide and document `BaseEffect.Copy` semantics (item E)** — 2 hours. Either:
   - Confirm "lock-step animation across AoE is intentional" — fix the misleading comment, add a `// Animation state is shared across copies` warning.
   - Add `Animator.Clone()` to the interface and implement per-strategy. Then `Copy()` calls `e.animator.Clone()`. Test that two copies advance independently.
9. **Decouple `vfx` from `graphics.BaseShape` (item G)** — half day. `NewVisualEffectArea` accepts `[]coords.LogicalPosition`; `spell_handler.go` calls `shape.GetIndices()` and translates to logical positions before invoking. Drops the `vfx → graphics` import edge.
10. **Add unit tests for the cheap-to-test parts (item M, first pass)** — 1 day:
    - `BaseShape.calculateCircle/Rectangle/Line` against golden index sets.
    - `RotateRight/RotateLeft` round-trip and full-cycle.
    - `ColorMatrix.IsEmpty` (post-rename).
    - `QuadBatch.Add` vertex layout for a single quad (compare against hand-computed `ebiten.Vertex`).
    - `tilesChanged` with same/replaced/empty/grown slices.
11. **Move `NewViewportRenderer` side effect out of constructor (item J)** — 1 hour. Add `coords.CoordManager.SyncToScreen(screen)`; call once in `main.go` Draw before any viewport renderer is fetched.

### Long-Term (only if it pays off)

12. **Test coverage for `TileRenderer.Render` and `RenderingCache` batching** — 1–2 days. Requires a fake `ebiten.Image` or running headless. Worth it once item 10 is done and you trust the pure parts.
13. **Promote `graphics.BaseShape` out of `graphics`** — `BaseShape` is a tile-targeting concept, not a graphics concept. It belongs in `core/coords` or a new `tactical/targeting/` package. Not urgent — current location works — but it would let `graphics` become purely "color and primitives."
14. **Replace `ShapeSize` with explicit dimensions** — five constructors all switch on `Small/Medium/Large` and only `Small` is ever used. Either make the dimension a parameter (`NewCircle(x, y, radius)`) or ditch the categories. Combine with item 13.

---

## 4. Prevention

- **Pre-merge gate:** require `go test ./visual/...` to pass with the new test files from item 10. CI catches at least the pure-function regressions.
- **Lint for re-export sprawl:** add a TODO checklist item: "Does this file export aliases for another package's symbols? If yes, ask whether the original location should change instead."
- **Dead-code scan:** run `staticcheck` or `unused` periodically. The vfx accumulation (item A) would have shown up as `U1000` warnings during the original refactor.
- **Constructor side effects:** code-review rule — constructors return values, they don't mutate globals. Item J is a teaching example.

---

## 5. What I Did NOT Find (positive signals)

- **`rendering` package is solid.** `QuadBatch` (`quadbatch.go`) is 80 LOC of focused vertex assembly; `RenderingCache` (`renderingcache.go`) is a thin batch-by-image map. No god-objects, no leaky abstractions.
- **Batching is doing real work.** `TileRenderer` (`maprender/tilerenderer.go:73-90`) tracks `batchesBuilt`, viewport center, and `TileColorsDirty` to skip rebuilds — the heaviest path runs only when something actually changed.
- **Coordinate logic centralized.** Both `TileRenderer.calculateViewportPosition` and `processRenderablesCore` use `coords.CoordManager.LogicalToScreen`, so scrolling-mode and viewport-centering quirks live in one place.
- **No `*ecs.Entity` storage.** `RenderingCache` uses an `*ecs.View`, components are read fresh per frame.
- **No circular deps.** Import graph: `graphics → coords`, `rendering → graphics + coords`, `maprender → rendering + graphics + coords + worldmapcore`, `vfx → graphics + coords`. Strictly acyclic.
- **Cache invalidation is thought through.** `BorderImageCache.GetOrCreate` (`viewport.go:19-38`) recreates GPU images only on size change. `CachedViewport.GetRenderer` recreates on screen change, updates on center change, no-ops otherwise.

The packages are not in bad shape *structurally*. The debt is concentrated in `vfx` (post-refactor cruft) and `graphics` (a broken type contract), with zero test coverage as the multiplier.

---

## 6. File-by-File LOC Reference

```
visual/graphics/colormatrix.go              137  LOC
visual/graphics/drawableshapes.go           437  LOC  (~50 LOC dead in unused size enums + GetLineTo)
visual/graphics/graphictypes.go              18  LOC  (collapsible to MouseToLogicalPosition only)

visual/maprender/maprendering.go             74  LOC
visual/maprender/tilerenderer.go            203  LOC

visual/rendering/quadbatch.go                79  LOC
visual/rendering/rendering.go                96  LOC
visual/rendering/renderingcache.go           63  LOC  (RefreshRenderablesView orphan, item K)
visual/rendering/viewport.go                162  LOC

visual/vfx/animators.go                     187  LOC  (~120 LOC dead — items A, H)
visual/vfx/renderers.go                     242  LOC  (~150 LOC dead — items A, H, I)
visual/vfx/vx.go                            108  LOC  (Copy/Update issues — items E, F)
visual/vfx/vxfactory.go                     156  LOC  (~75 LOC dead — item A)
visual/vfx/vxhandler.go                     125  LOC

TOTAL                                      2087  LOC
TESTS                                         0  LOC  (item M)
ESTIMATED DEAD CODE                       ~450  LOC  (~22%)
```

**Headline:** removing items A + B + C + L (~3 hours of work) deletes ~22% of the code, fixes one silent bug, and aligns the docs — without changing a single piece of behavior the game currently relies on.
