# TinkerRogue — Codebase-Wide Technical Debt Report

**Date:** 2026-07-08
**Scope:** Entire repository — 403 non-test `.go` files (~58.7k LOC), 38 test files (~9.7k LOC), 86 packages.
**Method:** Two analysis rounds, six parallel passes total — (1) code-level debt, (2) testing/documentation debt, (3) architecture/dependency debt, (4) error-handling/concurrency debt, (5) per-frame performance debt, (6) gamedata/save-format/determinism/tooling debt — with file-level verification.
**Companion docs:** `tech_debt_gamesetup.md` (2026-05-16), `tech_debt_powers.md` (2026-06-11) — package-scoped reports this one complements.

---

## Executive Summary

**Overall verdict: this is a notably clean, disciplined codebase.** The ECS conventions in CLAUDE.md are followed almost everywhere — zero `*ecs.Entity` struct-field violations, zero manual `y*width+x` tile-index math, only **2 TODO comments in the entire repo**, no FIXME/HACK markers, no dead-code files, and no cross-layer import violations. The GUI/game-state separation rule is honored with no leakage.

The real debt concentrates in **eight areas** (1–4 from the first analysis round, 5–8 from the second):

| # | Debt area | Severity | One-line summary |
|---|-----------|----------|------------------|
| 1 | GUI panel-registry god-functions | High | 11 `*_panels_registry.go` files, ~2,381 lines of copy-pasted `init()` boilerplate; worst single `init()` is 410 lines |
| 2 | Test coverage gaps | High | 65 of 86 packages (76%) untested; `campaign/raid` and `setup/savesystem/chunks` are the riskiest zero-test areas |
| 3 | Unmaintained core dependency | Medium (long-fuse) | `bytearena/ecs` v1.0.0 (last release ~2019) underpins everything, with no insulation layer |
| 4 | Global-singleton sprawl | Medium | ~15 mutable registries in `templates/`, 4 balance-config globals, 43 `init()` registration functions with implicit boot ordering |
| 5 | Save-format versioning & determinism | High | Per-chunk `ChunkVersion()` is dead scaffolding; schema changes silently zero-fill; XP RNG is wall-clock-seeded, making combat-analysis output non-reproducible |
| 6 | Runtime crash paths | High | `log.Fatalf`/`panic` reachable from Load Game, New Game, and entity creation — including one that defeats the save-load fallback and one inside the by-rule panic-free `templates/` package |
| 7 | Per-frame allocations | Medium | Always-on double slice alloc in the VFX handler every frame; per-squad/per-unit struct allocs and a flood-fill re-run every frame during combat |
| 8 | Missing dev tooling | Medium | No CI, no lint config, no Makefile, no git hooks; 4.25 MB generated font blob committed |

Plus one **repo-hygiene issue distorting all reviews**: 202 of 304 currently-modified files are CRLF-only phantom churn from an `autocrlf`/`.gitattributes` misconfiguration — only 102 files have real content diffs (511 insertions / 395 deletions).

---

## 1. Debt Inventory

### 1.1 Code Debt

#### GUI panel-registry `init()` god-functions — the single biggest code-debt item

11 `*_panels_registry.go` files totaling **2,381 lines** with **48 `framework.RegisterPanel(...)` calls**. Nine of the fifteen longest functions in the repo are these package-level `init()` closures:

| Lines | Location |
|---|---|
| 410 | `gui/guisquads/squadeditor_panels_registry.go:49` — `func init()` |
| 259 | `gui/guicombat/combat_panels_registry.go:14` — `func init()` |
| 191 | `gui/guioverworld/overworld_panels_registry.go:13` — `func init()` |
| 182 | `gui/guisquads/unitpurchase_panels_registry.go:74` — `func init()` |
| 181 | `gui/guiraid/raid_panels_registry.go:23` — `func init()` |
| 179 | `gui/guisquads/artifact_panels_registry.go:55` — `func init()` |
| 143 | `gui/guisquads/squaddeployment_panels_registry.go:44` — `func init()` |
| 127 | `gui/guiexploration/exploration_panels_registry.go:23` — `func init()` |

Every registration follows the identical copy-pasted shape:

```go
framework.RegisterPanel(PanelX, framework.PanelDescriptor{
    Content: framework.ContentCustom,
    OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
        m := mode.(*SomeMode)      // repeated type assertion in every closure
        result.Container = ...
        return nil
    },
})
```

The `mode.(*XxxMode)` assertion + `result.Container =` assignment + sub-menu construction is duplicated dozens of times. `createSquadEditorSubMenu` even carries a comment ("Follows the same pattern as combat mode's createCombatSubMenu") — an explicit in-code acknowledgment that the same helper is reimplemented per mode. These functions are impossible to unit-test and painful to diff.

#### Other long functions / god-object candidates

| Lines | Location | Issue |
|---|---|---|
| 194 | `world/garrisongen/generator.go:220` — `GarrisonRaidGenerator.Generate` | Mixes orchestration with low-level carving |
| 167 | `gui/guicombat/combatinput/handler.go:51` — `CombatInputHandler.HandleInput` | Monolithic input dispatch |
| 139 | `gui/guicombat/combat_mode_init.go:79` — `CombatMode.Initialize` | |
| 120 | `world/garrisongen/dag.go` — `BuildGarrisonDAG` | |
| 107/104 | `gui/guiraid/floormap_renderer.go` — `ComputeLayout`, `renderCard` | |
| 102 | `gui/builders/panels.go:51` — `PanelBuilders.BuildPanel` | Giant option-struct dispatcher |

#### Kitchen-sink query file

`tactical/squads/squadcore/squadqueries.go` — **537 lines, ~26 exported query functions** spanning capacity math, movement speed, health %, pattern computation, roles, and distance. Faithful to the "queries in one file" convention but has outgrown it; natural split: `capacity_queries.go`, `movement_queries.go`, `pattern_queries.go`.

#### Duplication clusters

- **`gui/builders` option plumbing:** `panels.go` (556) + `widgets.go` (470) + `dialogs.go` (394) + `layout.go` (275) contain many parallel `CreateXWithConfig(ContainerConfig{...})` constructors with overlapping option-handling code.
- **Overworld `Execute*` scaffolding:** `campaign/overworld/{faction,threat,node,garrison,influence,victory}/system.go` each repeat the manager-lookup → mutate-data → re-register pattern in per-action `Execute*` bodies (e.g. `faction/system.go:140/162/214/251/310`, `threat/system.go:119/144/193/216`). Consistent by design, but the shared skeleton is re-typed six times.
- **Tools reporting:** `tools/combat_analysis/combat_visualizer/summary_renderer.go` has three ~100-line sibling renderers; the `combat_balance` and `report_compressor` aggregators duplicate engagement-processing logic.

#### Largest files (top 10)

| Lines | File |
|---|---|
| 713 | `world/worldgen/gen_cavern.go` |
| 563 | `world/garrisongen/terrain.go` |
| 556 | `gui/builders/panels.go` |
| 537 | `tactical/squads/squadcore/squadqueries.go` |
| 517 | `world/worldgen/gen_overworld.go` |
| 510 | `gui/guiraid/floormap_renderer.go` |
| 493 | `tactical/squads/squadcore/squadcreation.go` |
| 470 | `gui/builders/widgets.go` |
| 468 | `templates/validation.go` |
| 458 | `gui/guisquads/squadeditor_panels_registry.go` |

Nothing egregious (no file over ~750 lines), but the GUI layer dominates the list.

#### TODO markers (complete list — there are only 2)

- `campaign/overworld/faction/system.go:137` — `// TODO, consider using an interface for intent`
- `mind/ai/ai_controller.go:84` — `// TODO: AI spell casting - enemy commanders don't cast spells yet.` ← **a real gameplay feature gap**, not just a code note

### 1.2 Testing Debt

#### Coverage landscape

**65 of 86 packages (~76%) have no tests.** Test effort is heavily concentrated in `tactical/`:

| Directory | Non-test LOC | Test LOC | Test:Source ratio |
|---|---|---|---|
| `tactical/` | 12,662 | 7,280 | **57.5%** (best) |
| `world/` | 3,871 | 636 | 16.4% |
| `core/` | 1,360 | 214 | 15.7% |
| `templates/` | 2,382 | 303 | 12.7% |
| `campaign/` | 5,296 | 536 | 10.1% |
| `setup/` | 3,216 | 262 | 8.1% |
| `mind/` | 4,647 | 349 | 7.5% |
| `gui/` | **18,711** | 175 | **0.9%** (worst; largest code body) |
| `visual/` | 2,022 | 0 | **0%** |

#### Highest-risk untested areas (ranked)

1. **`setup/savesystem/chunks` — 8 chunk serializers (commander, gear, map, player, progression, raid, squad), zero tests.** Save/load data-integrity bugs are the most player-hostile failure class; round-trip serialization tests are cheap to write.
2. **`campaign/raid` — 16 files, zero tests.** The entire raid runner / encounter / deployment / rewards / recovery pipeline is unverified.
3. **`tactical/combat/combatmath` and `combatstate` — untested** despite `combatcore` being the best-tested package in the repo. The math that `combatcore` depends on has no direct coverage.
4. **`campaign/overworld` — 7 of 9 subpackages untested** (`core`, `garrison`, `influence`, `node`, `threat`, `tick`, `victory`).
5. **`gui/` — 20 of 21 subpackages untested** (only `gui/builders/layout_test.go` exists). GUI is legitimately hard to test, but 18.7k LOC at 0.9% is the largest untested surface in the project.
6. **`visual/` — all 5 packages, literally zero tests.**
7. Also untested: `mind/ai`, `mind/spawning`, `mind/encounter`, `tactical/commander`, `tactical/powers/spells`, `core/coords`, `core/config`, `input`, `world/garrisongen`, `setup/gamesetup` (already documented in `tech_debt_gamesetup.md`, still true).

#### Test quality signals (mostly healthy)

- **Zero `t.Skip()` calls**, zero flaky-test markers.
- Assertions are hand-rolled (`t.Errorf` ×380, `t.Error` ×211, `t.Fatalf` ×115) — no testify dependency; consistent and fine.
- **One flakiness smell:** `campaign/overworld/overworldlog/overworld_recorder_test.go:273` uses `time.Sleep(10ms)` to force a distinct timestamp — an injectable clock would remove it.
- **Maintenance concentration point:** `tactical/powers/perks/perks_test.go` is 1,187 lines / 51 test functions — by far the largest test file (next: `squads_test.go` 767, `gear_test.go` 686, `combat_test.go` 643).

### 1.3 Architecture & Dependency Debt

#### Dependency health (`go.mod`, Go 1.22.5, no replace directives or forks)

| Module | Version | Risk |
|---|---|---|
| `github.com/bytearena/ecs` | **v1.0.0** | **Highest-risk dependency.** The core ECS library, effectively unmaintained (last release ~2019). Everything in the game is built on it, with no fork or wrapper insulation. |
| `github.com/golang/freetype` | 2017 pseudo-version | Deprecated upstream in favor of `golang.org/x/image/font`. |
| `github.com/ebitenui/ebitenui` | v0.5.8 | Pre-1.0, API-unstable by nature; 52 files depend on it via `gui/framework`. |
| `github.com/hajimehoshi/ebiten/v2` | v2.7.8 | One minor line behind 2.8.x — trailing, not dangerous. |
| `golang.org/x/image` | v0.18.0 | Slightly behind, low risk. |
| `github.com/ojrac/opensimplex-go` | v1.0.2 | Fine. |

#### Global-singleton sprawl — the most pervasive architectural theme

- **Coordinate/position core:** `coords.CoordManager` (**53 refs across 20 files**), `common.GlobalPositionSystem` (**43 refs across 18 files**), plus `coords.ScreenInfo`, `coords.MAP_SCROLLING_ENABLED`.
- **ECS registration globals:** dozens of `var XComponent *ecs.Component` package globals populated via **43 `init()` functions / 56 `RegisterSubsystem` call sites**. Intentional per CLAUDE.md, but it is implicit global mutable state with hidden boot-order coupling.
- **Mutable data registries in `templates/`:** `SpellRegistry`, `MonsterTemplates`, `ArtifactRegistry`, `UnitSpellRegistry`, `GameConfig`, `GlobalDifficulty`, and ~15 more in `templates/registry.go`.
- **Balance singletons:** `combatmath.CombatBalance`, `perks.PerkBalance`, `artifacts.ArtifactBalance`, `raid.RaidConfig`.
- **Misc:** `vfx.VXHandler`, `overworld/core.WalkableGrid`, `raid.GarrisonArchetypes`, `gui/widgetresources` font/image globals.

#### Coupling profile

- Fan-in hubs: `core/common` imported by **214 files**, `core/coords` by 120, `templates` by 57, `gui/framework` by 52, `core/config` by 31. `core/common` is a genuine hub, **not** a dumping ground (8 files, largest is `ecsutil.go` at 309 lines).
- Fan-out is concentrated in composition roots (`setup/gamesetup/bootstrap.go` and `game_main/main.go`, 16 internal imports each — expected). Watch-list: `setup/gamesetup/moderegistry.go` (15), `gui/guicombat/combatmode.go` (14).
- **One boundary smell:** `templates/schema_overworld.go` imports `game_main/campaign/overworld/ids` — a low→high inversion from the data layer, mitigated by the target being a leaf ID-constants package. Benign but worth keeping leaf-only.

### 1.4 Config & Tooling Debt

- **Shipped debug flags:** `core/config/config.go:13-16` has `DEBUG_MODE = true` and `ENABLE_BENCHMARKING = true` as compile-time `const`s (the latter binds a pprof server on `localhost:6060`), referenced from 15 non-config sites. A release build currently ships with both on.
- **Balance config not centralized:** four parallel balance-config files live outside `core/config` as package globals — `campaign/raid/config.go`, `tactical/combat/combatmath/balanceconfig.go`, `tactical/powers/artifacts/balanceconfig.go`, `tactical/powers/perks/balanceconfig.go`.
- **Inline magic numbers in spawning:** `mind/spawning/util.go` hardcodes `MinTargetPower: 50.0` / `MaxTargetPower: 2000.0`; `mind/spawning/squadscreation.go` uses raw power thresholds.
- **CRLF phantom churn:** of 304 modified files in the working tree, **202 are line-ending-only changes** (`LF will be replaced by CRLF` warnings on every diff). Only 102 files carry real diffs (511+/395−). This obscures every review and diff until `.gitattributes`/`core.autocrlf` is fixed.

### 1.5 Documentation Debt (minor)

- **Doc comments are strong overall** — sampled coverage: `core/common` 100%, `campaign/raid` 94%, `gui/framework` 93%, `squadcore` 92%. The outlier: **`tactical/combat/combatcore` at ~50%** (12 of 24 exported functions documented) — ironically the best-tested package has the worst-documented API.
- `complexity_report.txt` generated 2026-05-09 (~2 months stale).
- `DOCUMENTATION.md` v5.1, last updated 2026-04-22 (~11 weeks old).
- `PROJECT_LAYOUT.md` header says "Last Updated: 2026-07-02" but its last commit is 2026-06-12 — the header date overstates freshness. Content-wise both it and `PACKAGE_DEPENDENCIES.md` are structurally accurate (the "86 packages" claim was verified exact).
- Leftover empty directories: `gui/modesetup/`, `cmd/` (no Go files).

### 1.6 Error-Handling & Crash-Path Debt

Overall discipline here is unusually strong — save/load and JSON paths check every error, `%w` wrapping appears 160 times across 58 files, sentinel errors use `errors.Is`/`errors.Join` correctly, and no unchecked nil-derefs were found at the hot nil-returning call sites (`GetSquadEntity`, `FindEntityByID`, `GetComponentType`). The debt is a short, specific list of hard-crash and swallowed-error sites:

#### Runtime crash paths (`log.Fatal`/`panic` reachable outside boot)

| Location | Problem |
|---|---|
| `game_main/setup.go:202` | **The worst one.** `SetupRoguelikeFromSave` returns wrapped errors for every failure so `main.go:83-96` can fall back gracefully to a new game — except the final `EnterTactical` failure, which `log.Fatalf`s and defeats the entire fallback. Triggered by the runtime Load Game action. |
| `templates/entity_factory.go:37` | `log.Fatal(err)` on creature-image load failure — inside the by-rule **panic-free `templates/` package** (CLAUDE.md hard rule), and reachable at entity-creation time, not just boot. Should return an error. |
| `world/worldmapcore/GameMapUtil.go:52,68,74` | Missing-asset panics ("no floor tiles loaded" etc.) reachable during map creation on New Game — a hard crash instead of a surfaced error. |
| `game_main/setup.go:103,141` | `log.Fatalf` on mode-entry failure in `SetupOverworldMode`/`SetupRoguelikeMode`. |
| `setup/gamesetup/playerinit.go:29` | `log.Fatal` on player-image load — outside the sanctioned single boot-time fatal point (`LoadGameData`). |
| `gui/widgetresources/guiresources.go:303` | `HexToColor` panics on an unparseable hex string — a config-data parse that crashes. |

The `templates/` validators themselves were verified genuinely panic-free — the `entity_factory.go` fatal is the single exception in that package.

#### Swallowed errors (~6 genuine discards + ~5 silent asset skips)

- `gui/widgetresources/guiresources.go:18-20` — package-init `var smallFace, _ = loadFont(30)` (×3 for fonts + button image): a failed load yields a **nil face used later**, deferring the failure to an obscure nil-deref instead of a clear boot error.
- `guiresources.go:49-57` — `newPanelResources` returns nil **silently** (no log) on a missing panel PNG.
- `visual/vfx/vxfactory.go:22` — `img, _, _ = ebitenutil.NewImageFromFile(...)`: VFX image errors fully dropped.
- `world/worldmapcore/GameMapUtil.go:41,45,57,61,72` — per-file tile/stairs asset-load errors silently skipped; partial asset-directory failures load without complaint (only the fully-empty case is caught — by the panics above).
- `templates/difficulty.go:90` — latent nil-deref: `dm.current.Load().Name` on an `atomic.Pointer` that is nil until the first `SetDifficulty`; worth confirming an init-time `Store` exists.

#### Concurrency: effectively debt-free

Exactly **one** goroutine in production code — the pprof HTTP server (`setup/gamesetup/helpers.go:33`, gated by `ENABLE_BENCHMARKING`) — which touches no game state. No channels, no mutexes, one defensive `atomic.Pointer` in `templates/difficulty.go`. The many package globals are safe by virtue of the single-threaded ebiten loop.

### 1.7 Performance Debt (per-frame hot paths)

The rendering architecture is fundamentally sound: **no raw `World.Query` calls in the Update/Draw trees** (cached `ecs.View`s are used correctly), every `ebiten.NewImage` call is guarded by a size-change cache, the position system is used as a true O(1) grid, and extensive cache layers already exist (`RenderingCache`/QuadBatch batching, `TileRenderer` rebuild throttling, `SquadInfoCache` with dirty-flag invalidation, cached widgets/backgrounds, reusable input key buffers). The debt is a handful of specific allocation/recompute sites:

| Impact | Location | Problem |
|---|---|---|
| **High — every frame, every mode** | `visual/vfx/vxhandler.go:86-104` | `clearVisualEffects` allocates **two fresh slices every frame unconditionally** (called from `UpdateVisualEffects` via `main.go`), even with zero active effects. Trivial fix: early-return when both lists are empty, or compact in place. |
| Med — every combat frame | `gui/framework/guiqueries_rendering.go:19-32` | `GetSquadRenderInfo` allocates a new `&SquadRenderInfo{}` per squad per frame, called by both the highlight renderer and health-bar renderer. The underlying `SquadInfo` is cached; this wrapper is not. |
| Med — every combat frame | `tactical/squads/squadcore/squadqueries.go:143-152` | `FindAllSquads` allocates a result slice per call; invoked by two renderers → 2 allocs/frame. |
| Med — every frame in move mode | `tactical/combat/combatcore/combatmovementsystem.go:112-138` | `GetValidMovementTiles` re-runs a full flood-fill every frame while `InMoveMode`, re-deriving an unchanged tile set until the player moves. Cache keyed on (squadID, position, movementRemaining). |
| Med — combat-animation frames | `visual/combatrender/squad_renderer.go:126` | Per-unit-per-frame `&ebiten.DrawImageOptions{}` alloc (the viewport/border renderers already reuse theirs — the pattern exists in-repo). Also `guiqueries_rendering.go:42-69` allocates per-unit `UnitRenderInfo` and runs an O(all-members) `GetUnitIDsInSquad` scan per squad per frame. |
| Med — raid floor map visible | `gui/guiraid/floormap_renderer.go:209,355,385-390` | `renderEdges` rebuilds the `cardByNode` map every frame (only changes on `ComputeLayout`); `fmt.Sprintf` per card / per hover-line per frame. |
| Low — overworld frames | `gui/guioverworld/overworld_renderer.go:99,108,170` | Per-node registry lookup and per-commander roster re-resolve + linear scan every frame; both resolvable once per render pass. |

**Watch-list (event-driven today, O(n²) if they ever reach the frame path):** `GetActiveSquadsForFaction` (`combatqueries.go:97-106`, squads × members) and `GetEnemySquads` (`guiqueries.go:135-147`, factions × squads × members).

**Measurement gap:** only `combatcore` has benchmarks (`benchmarks_test.go`); none of the render/VFX/GUI hot paths above are measured. And `ENABLE_BENCHMARKING = true` (§1.4) means `runtime.SetCPUProfileRate` + the pprof listener impose profiling overhead on every ordinary play session.

### 1.8 Gamedata, Save-Format & Determinism Debt

#### Save-format versioning: scaffolding exists but is dead

- `SaveEnvelope` (`setup/savesystem/savesystem.go:37-51`) carries `Version`, `Timestamp`, `Checksum` — good bones. But the version check (`savesystem.go:162-165`) only rejects *newer* saves; there is no migration path (the comment says `// Version check (future: migration logic)`).
- **Per-chunk `ChunkVersion()` is dead code.** The interface declares it, all 7 chunks implement it (`progression_chunk.go:29` even returns 2 — evidence of a real schema change), but grep finds **zero call sites** — chunk versions are never serialized into the envelope, so a chunk schema change is undetectable at load time.
- **Schema change ⇒ silent partial corruption**, not an error: chunks decode via plain `json.Unmarshal` with no `DisallowUnknownFields` (`savesystem.go:183-192`) — renamed/removed fields silently become zero values; a chunk missing from the file is silently skipped. The SHA-256 checksum is `omitempty` — deleting it from a save file skips verification entirely (`savesystem.go:168`).
- Positives: atomic `.tmp` + rename write, best-effort `.bak` backup, clean 3-phase load (create → `RemapIDs` → `Validate`).

This compounds the §1.2 finding that the chunk serializers have zero tests: untested serializers **and** no schema-change detection.

#### RNG / determinism: three parallel RNG systems, one wall-clock-seeded

1. `core/common/randnumgen.go:8` — seeded PCG global (`rand.NewPCG(1, 2)`), reseedable via `SetRNGSeed`. Combat hit rolls use it; worldgen seeds it properly. Good.
2. `mind/combatlifecycle/reward.go:21` — separate `xpRNG = rand.New(rand.NewSource(time.Now().UnixNano()))`. **XP/level-up stat growth is non-reproducible by default**; `SetXPRNG` exists but is documented "tests only."
3. `gui/guiunitview/unitviewmode.go:74` — a third ad-hoc RNG in the UI.

**The kicker: `tools/combat_analysis` never calls `SetRNGSeed` or `SetXPRNG`.** The Monte-Carlo balance simulator shares the process-global RNG stream across all simulated battles and has wall-clock-seeded XP rolls — its output is non-reproducible run-to-run, which defeats the purpose of a balance-analysis pipeline.

#### Gamedata validation: four loading subsystems with inconsistent rigor

24 JSON files in `resources/assets/gamedata/` load through four different mechanisms:

| Subsystem | Failure behavior | Gaps |
|---|---|---|
| `templates.Loader[T]` (most files) | Fatal at boot — correct | `monsterDataLoader` (`readdata.go:32-35`) and `unitSpellLoader` (`unitspelldefinitions.go:18-21`) have **no validators** at all |
| Balance configs (`combatmath/balanceconfig.go:31-45`, perks, artifacts) | **Fail soft**: prints `WARNING`, leaves the global as zero values — silent gameplay degradation | `validateCombatBalance` only prints, never fails |
| `campaign/raid` loaders (`game_main/setup.go:31-36`) | Non-fatal "using defaults" warnings | Inconsistent with the fatal templates policy |
| Perk registry (`perks/registry.go:73`) | — | Loader ordering (spells before unitspells, nodes before encounters) enforced only by array position in `readdata.go:53-85`, documented as debt in `registry.go:5-18` |

- Unit→spell dangling references are **warnings, not errors** (`unitspelldefinitions.go:34-39`): a unit referencing a deleted spell boots fine and fails silently at cast time. (Node↔encounter↔faction links, by contrast, are validated fatally — the good pattern exists.)
- **Orphaned gamedata:** `consumabledata.json` and `creaturemodifiers.json` have zero Go references — dead files in the asset tree.
- Stringly-typing is mostly avoided (`SpellID`/`PerkID`/`ArtifactID` are named types); the one hotspot is stat names via `ParseStatType`'s 8-case string switch (`tactical/powers/effects/components.go:60-81`).

### 1.9 Build/Dev Tooling & Repo Hygiene Debt

- **No CI** (`.github/` absent), **no Makefile/Taskfile**, **no lint config** (`.golangci.yml`/staticcheck), **no active git hooks**. The only build documentation is the command in CLAUDE.md.
- **4.25 MB generated font file committed:** `resources/assets/fonts/mplus1pregular.go` is a single embedded byte-slice literal tracked in git — bloats every clone and diff. (Go 1.16+ `//go:embed` on the `.ttf` would eliminate it.)
- **Build artifacts in the working tree:** `tools.exe` (12 MB) at repo root plus binaries under `tools/combat_analysis/*/`. `*.exe` is gitignored, but extensionless binaries (e.g. `combat_visualizer`) are not caught by the pattern.
- **Hardcoded key bindings:** `gui/framework/defaultbindings.go` (173 lines) — the `ActionMap` abstraction cleanly decouples actions from keys, but there is no JSON/config loader on top of it; every rebind requires a recompile. Key overloads across modes (`KeyM`, `KeyS`) are only safe due to mode isolation.
- Stray Clojure tooling dirs (`.clj-kondo/`, `.lsp/`) in a Go repo (gitignored, minor).
- `tools/` itself is healthy: a real subcommand CLI (`tools/main.go` — sim/balance/viz/compress) with maintained scripts, not abandoned one-offs.

---

## 2. Impact & Risk Assessment

| Item | Severity | Concrete risk |
|---|---|---|
| `setup/savesystem/chunks` untested **+ dead `ChunkVersion()` + no `DisallowUnknownFields`** | **Critical** | Silent save-data corruption; a serializer or schema change destroys player progress with no test, no version check, and no decode error to catch it — only discovered on load, possibly long after |
| Load-game `log.Fatalf` (`game_main/setup.go:202`) | **High** | The one unrecoverable step in an otherwise recoverable load path — a corrupted save hard-crashes the game instead of falling back to a new game as `main.go` intends |
| Non-reproducible RNG in combat-analysis (`reward.go:21` + unseeded simulator) | **High** | The balance pipeline's Monte-Carlo output varies run-to-run; balance conclusions drawn from it are unrepeatable |
| Balance configs fail soft to zero values (`combatmath/balanceconfig.go:31-45` et al.) | **High** | A malformed balance JSON silently zeroes combat math — the game runs but plays wrong, with only a console warning |
| `campaign/raid` untested | **High** | Entire raid loop (encounters, deployment, rewards, recovery) can regress unnoticed; it is a primary gameplay pillar |
| Panel-registry god-`init()`s | **High** | Every GUI change lands in an untestable 100–400 line closure; merge conflicts and copy-paste bugs multiply; slows all UI work |
| `bytearena/ecs` unmaintained | **High** (slow-burning) | Go-version or platform break upstream has no fix path; blocks future Go upgrades; migration cost grows with every new component |
| CRLF phantom churn | **Medium** | 2/3 of every `git status` is noise; real regressions hide inside 300-file diffs; blame history degrades |
| Global registries/singletons | **Medium** | Tests must mutate global state (ordering hazards); parallel test execution and future headless simulation are constrained |
| `DEBUG_MODE`/pprof shipped on | **Medium** | Debug hooks and a localhost profiling server in release builds; performance and surface-area concerns |
| `combatmath`/`combatstate` untested | **Medium** | Balance-formula regressions surface as "combat feels wrong" rather than a failing test |
| New Game asset panics (`GameMapUtil.go:52/68/74`) + `templates/entity_factory.go:37` Fatal | **Medium** | Missing or renamed assets hard-crash at map/entity creation instead of surfacing an error; the entity-factory case violates the templates panic-free rule |
| Per-frame allocations (VFX handler, combat render-info, move-mode flood-fill) | **Medium** | Steady GC pressure at 60fps; the VFX one runs in every mode unconditionally |
| No CI / lint / hooks | **Medium** | Nothing mechanically enforces `go vet`, tests, or formatting; regressions land silently |
| Unvalidated gamedata loaders (monsterdata, unitspells) + dangling spell refs as warnings | **Medium** | Bad JSON or a deleted spell ships and fails at runtime (cast time) instead of boot |
| `gui/` 0.9% coverage | **Medium** | Largest code body regresses silently; partially mitigated by the framework/spec split making logic extractable |
| Silent asset-load skips + init-time discarded font errors (`guiresources.go:18-20`) | **Low** | Missing assets surface as distant nil-derefs or invisible sprites rather than clear errors |
| Balance config in 4 scattered files | **Low** | Tuning requires knowing 4 locations; inconsistent override paths |
| Orphaned gamedata (`consumabledata.json`, `creaturemodifiers.json`) | **Low** | Dead files invite confusion about what's live |
| Hardcoded key bindings; 4.25 MB committed font blob | **Low** | Rebinds require recompile; repo bloat on every clone |
| combatcore doc comments ~50% | **Low** | Onboarding friction on the most-central combat API |
| Stale docs / empty dirs | **Low** | Misleading freshness claims; trivial cleanup |

---

## 3. Prioritized Remediation Roadmap

Effort estimates assume a solo developer familiar with the codebase.

### Phase 1 — Quick wins (each < 1 day, ~2–3 days total)

| # | Action | Effort | Payoff |
|---|---|---|---|
| 1 | Add `.gitattributes` with an explicit eol policy (e.g. `* text=auto eol=lf`), renormalize once (`git add --renormalize .`), commit | ~1–2 h | Kills the 202-file phantom churn permanently; every future diff is readable |
| 2 | Gate `DEBUG_MODE` / `ENABLE_BENCHMARKING` behind build tags (`//go:build debug`) or move to JSON config | ~2–3 h | Release builds stop shipping pprof + debug hooks |
| 3 | Add `staticcheck ./...` + `go vet ./...` to the standard workflow (pre-commit or CI) | ~1 h | Locks in the already-excellent dead-code hygiene; catches unused code (U1000) mechanically |
| 4 | Replace `time.Sleep` in `overworld_recorder_test.go:273` with an injectable clock or explicit timestamp | ~1 h | Removes the only flakiness smell in the suite |
| 5 | Delete empty `gui/modesetup/` and `cmd/` dirs; regenerate `complexity_report.txt`; fix `PROJECT_LAYOUT.md` header date | ~1 h | Doc accuracy |
| 6 | Fix `game_main/setup.go:202` — return a wrapped error instead of `log.Fatalf` so the load-game fallback in `main.go` works | ~30 min | A corrupted save falls back to a new game instead of crashing |
| 7 | `templates/entity_factory.go:37` — return an error instead of `log.Fatal` (restores the templates panic-free rule) | ~30 min | CLAUDE.md rule compliance; no entity-creation crash |
| 8 | Early-return in `vfx.clearVisualEffects` (`vxhandler.go:86`) when both effect lists are empty | ~15 min | Kills the only always-on per-frame allocation |
| 9 | Seed `xpRNG` from the common seeded RNG (or reseed via `SetXPRNG` at boot); call `SetRNGSeed`/`SetXPRNG` in the combat-analysis simulator | ~1–2 h | Balance-pipeline output becomes reproducible |
| 10 | Delete or wire up orphaned `consumabledata.json` / `creaturemodifiers.json`; extend `.gitignore` to catch extensionless tool binaries | ~30 min | Repo hygiene |

### Phase 2 — Short term (1–2 weeks)

| # | Action | Effort | Payoff |
|---|---|---|---|
| 1 | **Round-trip serialization tests for `setup/savesystem/chunks`** — for each of the 8 chunk serializers: build entities via `testing/` fixtures, save, load, assert equality | ~2–3 days | Eliminates the Critical-severity save-corruption blind spot; also documents the save format |
| 2 | **Generic typed panel-registration helper** — e.g. `RegisterModePanel[M framework.UIMode](id PanelID, build func(m M, result *PanelResult) error)` that performs the type assertion once. Migrate the two worst registries (`squadeditor`, `combat`) first, then the remaining nine mechanically | ~3–4 days | Collapses much of the ~2,381 lines of registry boilerplate; new panels become one small typed function |
| 3 | **Core-path tests for `campaign/raid`** — raid runner state transitions, deployment validation, reward calculation (the pure-logic parts, not rendering) | ~2–3 days | Covers the largest untested gameplay pillar |
| 4 | Split `squadcore/squadqueries.go` into cohesive files (`capacity_queries.go`, `movement_queries.go`, `pattern_queries.go`) — pure file move, no signature changes | ~2 h | Navigability; keeps the queries-file convention honest |
| 5 | **Wire `ChunkVersion()` into the save envelope** (serialize per-chunk versions, check on load) and decide the `DisallowUnknownFields` policy | ~1 day | Schema changes become detectable errors instead of silent zero-values; pairs with item 1's round-trip tests |
| 6 | Add validators for `monsterdata.json` and `unitspells.json`; promote dangling unit→spell references from warning to fatal (the fatal pattern already exists in `validation.go:132-166`) | ~1 day | Bad gamedata fails at boot, not at cast time |
| 7 | Cache the per-frame combat render info: reuse `SquadRenderInfo`/`FindAllSquads` results via the existing `SquadInfoCache` dirty mechanism; cache `GetValidMovementTiles` keyed on (squad, position, movement) | ~1–2 days | Removes the main steady-state combat GC pressure |

### Phase 3 — Medium term (1–2 months, interleaved with feature work)

| # | Action | Effort | Payoff |
|---|---|---|---|
| 1 | Tests for `tactical/combat/combatmath` and `combatstate` (pure math → table-driven tests are cheap) | ~2 days | Balance formulas regress loudly instead of silently |
| 2 | Consolidate `gui/builders` option plumbing — unify the parallel `CreateXWithConfig` constructors behind a smaller option set; shrink the 102-line `BuildPanel` dispatcher | ~3–4 days | Reduces the 1,700-line builders surface; fewer places to touch per widget change |
| 3 | Extract shared overworld `Execute*` scaffolding (manager-lookup → mutate → re-register) into one helper used by all 6 system.go files | ~1–2 days | Six copies become one; new overworld actions get the skeleton for free |
| 4 | Decide balance-config placement: either fold the 4 `balanceconfig.go` globals behind `core/config`'s JSON-override path, or document in CLAUDE.md why per-package balance files are the convention | ~1 day | One tuning story instead of four |
| 5 | Extract spawning magic numbers (`mind/spawning/util.go` power bounds) into config | ~2 h | Tunable difficulty without recompiling |
| 6 | Doc comments for the undocumented half of `tactical/combat/combatcore`'s exported API | ~2–3 h | Fixes the one doc-coverage outlier |
| 7 | Address `ai_controller.go:84` TODO — enemy commander spell casting — or convert it to a tracked design item | feature-sized | Closes a real gameplay asymmetry |
| 8 | Unify the three RNGs behind `core/common/randnumgen.go` (kill the wall-clock `xpRNG` and the UI-local RNG) | ~half day | One seedable stream; full-run determinism becomes possible |
| 9 | Unify the four gamedata loading subsystems onto `Loader[T]` fail-fast semantics (balance configs and raid loaders currently fail soft) | ~2 days | One loading story; malformed JSON can't silently zero combat math |
| 10 | Convert `GameMapUtil.go` asset panics and silent per-file skips into returned errors surfaced at New Game | ~half day | Missing assets produce a clear message instead of a crash or invisible tiles |
| 11 | Keybindings JSON loader on top of the existing `ActionMap` abstraction (`defaultbindings.go` becomes the fallback) | ~1 day | User-rebindable keys without recompiling |
| 12 | Minimal CI (GitHub Actions): build + `go vet` + `staticcheck` + `go test ./...`; replace the committed 4.25 MB font `.go` file with `//go:embed` of the `.ttf` | ~half day | Mechanical enforcement; smaller repo |

### Phase 4 — Long term / strategic

| # | Action | Effort | Payoff |
|---|---|---|---|
| 1 | **`bytearena/ecs` insulation strategy.** Recommended: don't migrate — *wrap*. `common.EntityManager` already mediates most access; finish routing the remaining direct `ecs.*` uses (World.Query, NewComponent, NewTag, DisposeEntities) through it, then vendor or fork the library. A future engine swap becomes a bounded refactor instead of a rewrite | ~1–2 weeks spread out | Converts the highest structural risk into a contained one |
| 2 | Reduce direct global access where cheap: new code takes `CoordManager`/position-system via parameters or a context struct; existing 96 call sites migrate opportunistically, not big-bang | ongoing habit | Unlocks parallel tests and headless simulation over time |
| 3 | GUI logic extraction: as panels are touched, move decision logic out of `OnCreate`/`OnClick` closures into plain testable functions, letting `gui/` coverage grow organically past 0.9% | ongoing habit | Shrinks the largest untested surface without a dedicated testing project |

---

## 4. Prevention

- **Tooling gate:** `go vet ./...` + `staticcheck ./...` before commit (Phase 1 item — the codebase is clean enough today that turning this on is nearly free).
- **Line endings:** `.gitattributes` committed so `autocrlf` differences can never again generate phantom diffs.
- **Test expectation:** any new package under `tactical/`, `campaign/`, `setup/savesystem/`, or `world/` ships with at least one `_test.go`; new save-chunk serializers require a round-trip test.
- **Size tripwire:** treat any single function crossing ~100 lines (especially registry `init()`s) as a refactor trigger — the generic registration helper from Phase 2 makes staying small the path of least resistance.
- **Keep the existing CLAUDE.md doc-sync rule** (PROJECT_LAYOUT.md / PACKAGE_DEPENDENCIES.md updated in the same change as structural edits) — it is working; both files verified accurate.

---

## 5. What's Healthy (keep doing this)

Credit where due — these are the reasons this report is short on horror stories:

- **ECS discipline is essentially perfect.** Zero stored `*ecs.Entity` struct fields (56 occurrences are all legitimate params/returns/locals), zero manual tile-index math, components are pure data, queries/systems/components file split is followed everywhere.
- **Layering is clean.** No cross-layer import violations anywhere: `core/` imports nothing high-level, `visual/` doesn't touch `gui/`, `campaign/`/`tactical/`/`mind/` don't touch `gui/` or `visual/`. The single `templates → overworld/ids` wrinkle is a leaf-constants package.
- **UI-state/game-state separation holds.** `BattleMapState`/`OverworldState` appear only inside `gui/` and `game_main/` — no leakage into game logic.
- **Near-zero debt markers.** 2 TODOs, 0 FIXMEs, 0 commented-out code blocks, 0 backup/deprecated files.
- **Doc-comment coverage is 90%+** in most sampled packages; `PACKAGE_DEPENDENCIES.md`'s package count is exactly accurate.
- **`tactical/` at 57.5% test:source ratio** shows the team knows how to test this architecture — the gaps elsewhere are prioritization, not capability.
- **Recent git history is deliberate refactoring** (GameMap encapsulation series, cleanup passes), not abandoned WIP.
- **Error wrapping is exemplary where it matters.** The save/load path checks and `%w`-wraps every error (160 wrap sites across 58 files), uses sentinel errors with `errors.Is` and `errors.Join` correctly, and callers of nil-returning helpers consistently nil-check — no unchecked derefs were found.
- **Concurrency is a non-issue.** One stateless pprof goroutine; the single-threaded ebiten loop makes the package globals safe in practice.
- **The render path is architected for performance.** Per-frame code uses cached `ecs.View`s (never raw queries), all `ebiten.NewImage` calls are size-change-cached, and there's a real cache ecosystem (QuadBatch sprite batching, tile-batch rebuild throttling, dirty-flagged `SquadInfoCache`, cached widgets). The per-frame findings in §1.7 are gaps in an otherwise deliberate system.
- **The save system has good bones**: atomic tmp+rename writes, `.bak` backups, SHA-256 checksums, and a clean 3-phase load — the versioning gaps in §1.8 are unfinished scaffolding, not absent design.
- **Identifiers are properly typed** (`SpellID`, `PerkID`, `ArtifactID` as named types) — stringly-typed debt is nearly absent.

---

*Generated 2026-07-08 from a six-pass repository analysis in two rounds (code-level, testing/docs, architecture/dependencies; then error-handling/concurrency, per-frame performance, gamedata/save-format/determinism/tooling). File:line citations were spot-verified against the working tree.*
