# Tech Debt Analysis: `world/worldmapcore` & `world/worldgen`

**Date:** 2026-05-12
**Scope:** `world/worldmapcore` (6 files, 502 LOC) and `world/worldgen` (5 files, 1,524 LOC)
**Total:** 2,026 LOC, **0 test files**, 30 external caller files (177 symbol references)

These two packages sit at the bottom of the dependency stack — defects here amplify upward into rendering, save system, gamesetup, GUI exploration/overworld, and templates.

---

## 1. Debt Inventory

### 1.1 CRITICAL — Zero test coverage on procedural generation

**Item:** Neither package has a single `_test.go` file. All three generators (rooms, cavern, overworld) plus the registry, helpers, and `GameMap` lifecycle are uncovered.

**Why it matters:** Procedural generation is the *worst* code to leave untested because it's stochastic — bugs hide for many runs, only to surface as panics deep in QA. The connectivity, MST, and CA passes in `gen_cavern.go` are pure functions over `[]bool`, i.e. *trivial* to unit test.

**Quantify:** 2,026 LOC at 0% coverage; ~15 deterministic-input/seedable code paths that would be cheap to cover.

---

### 1.2 CRITICAL — Hidden global state via `graphics.ScreenInfo`

**Locations:**
- `world/worldmapcore/dungeongen.go:66-67, 100-101, 215`
- `world/worldgen/gen_helpers.go:67-68, 271-272`

**Problem:** `NewGameMap` ignores the `width/height` produced by the generator and re-reads them from `graphics.ScreenInfo`. `CreateEmptyTiles` accepts `width, height` parameters but then computes pixel coords from `graphics.ScreenInfo.TileSize` — *unrelated* to the input dimensions. `InBounds(x,y)` ignores the receiver entirely and asks the global.

This is the exact failure mode CLAUDE.md warns about ("`CoordinateManager.dungeonWidth` may not match function parameters"), reproduced in another guise.

**Impact:**
- Can't generate a map at a non-screen size (e.g., for save-file migration, headless tests, mini-maps).
- Tests would need to mutate package globals before each run.
- Garrison generator is forced to share the global even though its floors are conceptually independent.

---

### 1.3 HIGH — Dead code in `GameMap` (`tileContents`, `AddEntityToTile`, `RemoveItemFromTile`)

**Location:** `world/worldmapcore/dungeongen.go:121-157` and the `tileContents TileContents` field at `dungeontile.go:34`.

**Evidence:** A grep across the entire repo shows zero external callers. Only the doc file references them.

The comment claims *"Currently only used for holding items"* — but nothing puts items on tiles via this API; items are tracked by ECS position + `GlobalPositionSystem` (per CLAUDE.md). This is ~40 LOC of "ECS-violating-looking" code that gets surface-read as a guide but has no behavior.

**Cost:** Confuses readers (especially when paired with the `// Fixed ECS violation` comment that papered over rather than removed it) and bloats the public surface that callers might latch onto.

---

### 1.4 HIGH — Inconsistent grid representation: `[]bool` vs `[][]float64` vs `[]*Tile`

`gen_cavern.go` and `gen_helpers.go` use `terrainMap []bool` (true=walkable) indexed via `PositionToIndex`. `gen_overworld.go` uses `[][]float64` (row-major nested) for elevation/moisture but `[]bool` for terrain. The final tile slice is `[]*Tile` indexed via `coords.CoordManager.LogicalToIndex`.

**Cost:**
- `gen_overworld.go:130-170` allocates a `[][]float64` (one slice header per row + sub-slice headers), violating the very allocation-reduction pattern `CreateEmptyTiles` adopted at `gen_helpers.go:50-78`.
- Three indexing schemes coexist: `idx := y*width + x` (only inside `CarveCorridorBetween:171-172` to *decode* indices back to coords), `PositionToIndex(x,y)`, and direct `noiseMap[y][x]`.
- `CarveCorridorBetween:171-172` decodes via `idx%width, idx/width` — which is broken if `CoordManager.dungeonWidth != width`. Same root-cause as 1.2.

**Quantify:** ~6 helper functions in `gen_helpers.go` would unify cleanly behind a small `Grid[T]` wrapper or even just `(x,y,w,h) → idx` consistently applied.

---

### 1.5 HIGH — Long generator methods with mixed responsibilities

| Method | LOC | Cyclomatic-ish | Issue |
|---|---|---|---|
| `CavernGenerator.Generate` | 82 (84-166) | low | OK — orchestrator only |
| `CavernGenerator.carveDrunkardTunnel` | 95 (366-460) | ~14 | Movement + carving + radius toggle + clamping interleaved |
| `CavernGenerator.cellularAutomataStep` | 42 | ~10 | Two algorithms in one method, branched by `gentle bool` flag |
| `CavernGenerator.checkWalkableRatio` | 47 | ~10 | One branch does CA, the other branch reinvents erosion inline |
| `placeTypedPOIs` + `placePOIType` + `placeGuildHalls` | 110 combined | — | Town/temple/watchtower share `placePOIType`, but `placeGuildHalls` is a near-duplicate variant (see 1.6) |
| `StrategicOverworldGenerator.Generate` step pipeline | 38 (90-127) | low | OK, but uncommented "Step 4" mutates result + terrainMap with no docs on the invariant |

**Cost:** Each generator is one large `struct{ config }` with a method per pipeline stage. None is over the 500-LOC "god class" line individually (cavern is 619 LOC of file but the struct's logic is partitioned), but `carveDrunkardTunnel` and `checkWalkableRatio` need decomposition before they grow further.

---

### 1.6 HIGH — Duplication: POI placement loops

`placePOIType` (`gen_overworld.go:393-443`) and `placeGuildHalls` (`gen_overworld.go:446-501`) share:
- Retry loop with `maxAttempts := count * 50`
- `validIdx := common.GetRandomBetween(0, len(result.ValidPositions)-1)` pick
- `PositionToIndex` + bounds check
- Distance filter via `isTooCloseEuclidean`
- Identical `result.POIs = append(...)` + `result.Rooms = append(...)` + image assignment block

~40 lines of near-duplicate logic. Guild halls only differ by the "must be near a town" predicate. This is a textbook case for parameterizing a `placementRule` function:

```go
type poiRule struct {
    nodeID         string
    count, minDist int
    accept         func(pos coords.LogicalPosition, idx int) bool
}
```

Then `placeGuildHalls` collapses to one rule with `accept = withinDistance(townPositions, 20)`.

---

### 1.7 MEDIUM — Two unrelated `IsTooClose*` helpers (Chebyshev vs Euclidean)

- `IsTooCloseToAny` (`gen_helpers.go:347-362`) — Chebyshev distance, takes `[][2]int`.
- `isTooCloseEuclidean` (`gen_overworld.go:525-532`) — Euclidean, takes `[]coords.LogicalPosition`.

Different distance metrics may be intentional, but the *shape mismatch* (`[2]int` vs `LogicalPosition`) is gratuitous. One uses package-private lowercase; the other is exported. Consolidate to a single typed helper in `gen_helpers.go`.

---

### 1.8 MEDIUM — Leaky `MapGenerator.Generate` API

`MapGenerator.Generate(width, height, images)` returns a `GenerationResult` with `GarrisonData any` (`generator.go:29`). Using `any` here punctures the abstraction: it's a hidden out-of-band channel for one specific generator. Either:
- Lift garrison-specific data to a `GarrisonResult` returned by `garrisongen` only, OR
- Define `type GeneratorMetadata interface { ... }` so the type is at least named.

Also: `images TileImageSet` is a *visual* input to generation, which conflates layout with rendering. A generator should produce *abstract* tile data (type + biome), and a separate "paint" step should pick images. This is why `applyConnectivityFixes` has to re-call `GetBiomeImages` (`gen_overworld.go:295`) after the fact.

---

### 1.9 MEDIUM — Registry: stringly-typed global mutable map

`worldgen/registry.go`:
- Package-level `var generators = make(map[string]worldmapcore.MapGenerator)` mutated by `init()` and `RegisterGenerator`.
- `var ConfigOverride func(name string) worldmapcore.MapGenerator` — a second registration mechanism layered on top.
- `GetGenerator` silently falls back to `"rooms_corridors"` if the name isn't found (`registry.go:30-32`). No error, no log. A typo in a save file or config silently swaps the algorithm.

**Cost:** Hard to test (must clear globals between tests). The silent fallback is a debug-time landmine.

---

### 1.10 MEDIUM — `Rect` shape ambiguity

`worldmapcore.Rect` is `{X1, X2, Y1, Y2}`, but `NewRect(x, y, width, height)` sets `X2 = x + width, Y2 = y + height`. Then `CarveRoom` uses `y := room.Y1 + 1; y < room.Y2` (exclusive). So `X2/Y2` are simultaneously "right/bottom edge" *and* "x+width" — fine until callers (e.g., the room as a stand-in for a POI at `gen_overworld.go:438`) create `NewRect(pos.X, pos.Y, 1, 1)` which produces a 1×1 rect that `Center()` reports as `(pos.X+0, pos.Y+0)` — works only because `(X1+X2)/2 == X1` for width 1. The semantic is "half-open" but undocumented.

`Intersect` uses `<=` and `>=`, i.e. *inclusive* — inconsistent with the half-open carving convention. Two rooms sharing an edge "intersect" by this method but their carved interiors don't overlap. This is why room-placement is conservative but still occasionally wastes attempts.

---

### 1.11 LOW — Error-swallowing tile loaders

`LoadTileImages` (`GameMapUtil.go:29-79`) silently discards every `os.ReadDir` and `ebitenutil.NewImageFromFile` error. A misnamed biome directory yields an empty `BiomeImages` map silently, which downstream becomes `nil` images on tiles. The fallback chain in `selectBiomeFloorImage` masks the failure further.

**Quantify:** 5 error-swallow sites. At minimum, the *first* failure on the default floor/wall directory should panic (game is unrunnable) rather than producing a black map.

---

### 1.12 LOW — Misc smells

- `GameMap.IsOpaque` has `// TODO: Change this to check for WALL, not blocked` — already does (line 241). Stale TODO.
- `GameMap` receiver inconsistency: `Tile`, `StartingPosition`, `PlaceStairs`, `AddEntityToTile`, `RemoveItemFromTile`, `ApplyColorMatrix*` use `*GameMap`; `InBounds` and `IsOpaque` use value receivers. Inconsistent and causes one extra copy of a struct that holds slices.
- `gen_overworld.go:130-170` allocates `[][]float64` per generation (~2 maps × W×H float64). For an overworld 200×100, that's 320 KB churn × 2 per generate.
- `gen_overworld.go:273-301` (`applyConnectivityFixes`) recomputes biome/image for any tile carved by `EnsureTerrainConnectivity` — but `EnsureTerrainConnectivity` doesn't report what it changed; the function diffs `terrainMap` against `result.Tiles[idx].Blocked`. Fragile: depends on those two staying in lock-step.
- `erosionAccretionPass` (`gen_cavern.go:511-562`) builds a `[]struct{idx int; val bool}` anonymous struct slice each pass. Use a named type and reuse the buffer.

---

## 2. Impact Assessment

| Item | Risk | Velocity hit | Trigger to fix |
|---|---|---|---|
| 1.1 No tests | **Critical** | Every change in these packages is a coin flip | First seed-dependent panic in QA |
| 1.2 `graphics.ScreenInfo` global | **Critical** | Blocks testability & headless gen | Already biting garrison gen (forced screen size) |
| 1.3 Dead `tileContents` API | High | Misleads new contributors | Trivial — delete now |
| 1.4 Grid type inconsistency | High | Each new generator pays the tax | Next generator authored |
| 1.5/1.6 Cavern method size + POI dup | High | New POI types require copy-paste | When adding 5th POI type |
| 1.8 `GarrisonData any` | Medium | Will rot as garrison features grow | When second special-case generator appears |
| 1.9 Registry silent fallback | Medium | Debug-time landmine | When a save loads with old map name |
| 1.10 Rect half-open vs inclusive | Medium | Edge-case off-by-ones | Already (intersect generates churn) |
| 1.11 Silent image errors | Low | Black-map mystery bugs | Already plausible in CI |

---

## 3. Prioritized Roadmap

### Sprint 1 — Quick Wins (≤ 1 day each)

1. **Delete `tileContents` / `AddEntityToTile` / `RemoveItemFromTile`** and the `TileContents` struct. Verify with `go build ./...`. Update `WORLD_GENERATION.md` and `dungeontile.go:23-25` comment. **Effort: 30 min.**

2. **Remove stale `// TODO: Change this to check for WALL`** on `IsOpaque`. **Effort: 1 min.**

3. **Make `GameMap` receivers consistent** — convert `InBounds` and `IsOpaque` to `*GameMap`. **Effort: 5 min.**

4. **Fail loudly on missing default tiles** in `LoadTileImages` — keep biome dirs optional, but panic if `defaultFloorPath()` or `defaultWallPath()` yield zero images. **Effort: 15 min.**

5. **Log instead of silently-falling-back** in `GetGenerator` when the requested name isn't registered. **Effort: 10 min.**

### Sprint 2 — Test scaffolding (1 week)

6. **Plumb width/height through instead of `graphics.ScreenInfo`.** Pass a `GenContext{Width, Height, TileSize int}` to `Generate` and to `CreateEmptyTiles`. `NewGameMap` reads from `graphics.ScreenInfo` *at the call site* and passes the context in. **Effort: 3-4 hours.** Unlocks everything below.

7. **Write seed-locked golden tests** for each generator: fixed seed → assert tile count walls vs floors, room count, connectivity. **Effort: 1 day per generator (3 days total).**

8. **Unit-test pure helpers** in `gen_helpers.go`: `FloodFillRegion`, `EnsureTerrainConnectivity`, `IsTooCloseToAny`, `FindBestOpenPosition`, `ScoreTerrainOpenness`, `TryPlace2x2PillarOnTerrain`. These are pure functions on `[]bool`. **Effort: 1 day, ~80% coverage.**

### Sprint 3 — Targeted refactor (2 weeks)

9. **Extract `pointPlacement` helper** to collapse `placePOIType` and `placeGuildHalls`. **Effort: 3 hours.**

10. **Replace `[][]float64` with `[]float64`** in `gen_overworld.go`. Use `PositionToIndex(x,y)` everywhere. Removes ~3 KB of slice headers per generation and unifies indexing. **Effort: 2 hours.**

11. **Split `carveDrunkardTunnel`** into `stepDrunkard()`, `carveCircle()`, `clampToMap()`. Drop cyclomatic to ≤ 6 per fn. **Effort: 2 hours.**

12. **Split `cellularAutomataStep`** into `caAggressive` and `caGentle` (one wraps `countWallNeighbors`). Remove the `gentle bool` parameter. **Effort: 1 hour.**

### Sprint 4 — Architectural cleanup (only if a new generator is planned)

13. **Decouple generation from rendering.** Generators emit `(TileType, Biome, POIType)` per tile; rendering pass picks images. Removes `images` param from generator and the awkward post-hoc image assignment in `applyConnectivityFixes`. **Effort: 1 week.**

14. **Replace `GarrisonData any`** with a typed `GeneratorExtras` map keyed by string + typed accessors in `garrisongen`. **Effort: 4 hours.**

15. **Document `Rect`'s half-open convention** at the type definition; reconsider whether `Intersect` should use `<` to match. **Effort: 30 min + audit of all `Intersect` callers.**

---

## 4. Quick-Win Patch List (this PR can land)

```
[ ] world/worldmapcore/dungeongen.go   delete AddEntityToTile, RemoveItemFromTile (~37 LOC)
[ ] world/worldmapcore/dungeontile.go  delete TileContents struct + tileContents field (~5 LOC)
[ ] world/worldmapcore/dungeongen.go   remove stale TODO on IsOpaque (line 234)
[ ] world/worldmapcore/dungeongen.go   normalize receivers to *GameMap on InBounds/IsOpaque
[ ] world/worldmapcore/GameMapUtil.go  panic if default floor/wall load returns 0 images
[ ] world/worldgen/registry.go         log when GetGenerator falls back to default
```

Net: −45 LOC, −1 ECS pseudo-violation, +2 fail-fast guards. Risk: low (dead code + log statement).

---

## 5. Prevention Plan

- **Coverage gate:** Once Sprint 2 lands, require ≥ 60% line coverage for `world/...` in CI. Currently 0% → any test is improvement.
- **Lint rule:** add a custom check (or a code-review checklist item per CLAUDE.md §"Critical Warnings") forbidding new uses of `graphics.ScreenInfo.Dungeon*` inside `world/`.
- **Registry validation:** at startup, log the full set of registered generator names and validate that every name referenced in `mapgenconfig.go` is registered. Fails fast on typos.
- **Half-open convention:** document on `Rect` and add a `_ = Rect{}.contains(...)` comment showing the intent. Future PRs reference this.

---

## 6. ROI Snapshot

| Effort bucket | Hours | Primary payoff |
|---|---|---|
| Quick wins (Sprint 1) | ~1 hr | Removes misleading code; fail-loud on broken assets |
| Test scaffolding (Sprint 2) | ~32 hrs | Makes every later change safe; unlocks seedable gen |
| Targeted refactor (Sprint 3) | ~10 hrs | Cuts duplication; cleans hottest methods |
| Architectural (Sprint 4) | ~40 hrs | Only justified if a 4th generator is on the roadmap |

The **highest leverage** items are Sprint 1 #1 (delete dead code) and Sprint 2 #6 (parameterize `width/height`). Together they're ~5 hours and they unblock all of #7–#15.
