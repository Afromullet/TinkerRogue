# Technical Debt Analysis: `setup/gamesetup/` Package

**Date:** 2026-05-16
**Scope:** 10 files, 1,186 LOC. Orchestrates ECS init, world creation, player/commander/squad/faction creation, mode registration, and save-load restoration.
**Test coverage:** **0%** — no `_test.go` files in the package.

---

## Files in Scope

| File | LOC | Purpose |
|---|---|---|
| `initial_squads.go` | 349 | 5 squad-archetype creators + roster registration |
| `bootstrap.go` | 168 | `GameBootstrap` phases (data load, ECS, world, player, gameplay) |
| `mapgenconfig.go` | 153 | JSON → map-generator config overrides |
| `savehelpers.go` | 108 | Save/load restoration (player, renderables, map chunk) |
| `moderegistry.go` | 106 | Tactical/overworld/roguelike UI mode registration |
| `playerinit.go` | 75 | Player entity creation |
| `initial_commanders.go` | 75 | Commander roster construction |
| `ecsinit.go` | 56 | Core ECS components + tag registration |
| `initial_factions.go` | 54 | NPC faction creation on overworld |
| `helpers.go` | 42 | Walkable grid + pprof setup |

---

## 1. Debt Inventory

### Code Debt — Duplication (Critical)

#### D1. Squad creator copy-paste — `initial_squads.go:148-349`

Five functions (`createBalancedSquad`, `createRangedSquad`, `createMagicSquad`, `createMixedSquad`, `createCavalrySquad`) share ~85% of their body. Each independently:

- Allocates `unitsToCreate := []unitdefs.UnitTemplate{}`
- Defines a hardcoded `[][2]int{}` grid positions table
- Loops to pick random units from a pool (`unit.GridRow/GridCol` assigned by index)
- Picks `leaderIndex := common.RandomInt(unitCount)`, sets `unit.IsLeader = true`, then `unit.Attributes.Leadership = 20` (literal `20` repeated 5×)
- Calls `squadcore.CreateSquadFromTemplate(...)` with the same 5-arg pattern

Net: ~200 lines that should be ~40 + a data table. The `Leadership = 20` literal alone is a magic number copied 5 times.

#### D2. Player image loaded 4× from disk

`ebitenutil.NewImageFromFile(config.PlayerImagePath)` is called in:

- `playerinit.go:27` (fresh game, player)
- `initial_commanders.go:26` (fresh game, every commander)
- `savehelpers.go:55` (load path, player)
- `savehelpers.go:68` (load path, every commander)
- Plus `gui/guioverworld/overworld_action_handler.go:335`

No cache. Every commander on a saved game does a fresh disk read for the same PNG.

#### D3. Two divergent string→enum tables for squad/faction types

- `squadCreators` map (`initial_squads.go:21-27`) and `validSquadTypeIDs` in `templates/initialsetup.go:50-56` — comment says *"Keep keys in sync"*. They aren't structurally linked.
- `factionTypeFromString` switch (`initial_factions.go:42-54`) duplicates the keys of `validFactionTypeIDs` in templates with the same warning.

Adding a new squad type silently breaks if either side isn't updated.

#### D4. Mode registration duplication — `moderegistry.go:27-92`

`RegisterTacticalModes` and `RegisterRoguelikeTacticalModes` build overlapping mode slices. Adding a new tactical mode requires editing both. The roguelike variant also re-runs `newCombatServiceFactory()` rather than passing the one already constructed.

### Code Debt — Structure / Magic values

#### D5. Magic numbers throughout `initial_squads.go`

- Grid position tables hardcoded inside each creator instead of `formations.json`
- Squad sizes (`maxUnits := 5` at line 160, `unitCount := 3` at line 233, `GetRandomBetween(3,5)` / `(4,5)`) baked in
- `Attributes.Leadership = 20` (×5)
- Filter thresholds: `FilterByAttackRange(3)`, `FilterByMinMovementSpeed(6)` — no rationale; not configurable

#### D6. Magic strings — no constants

`"players"`, `"renderables"`, `"all"`, `"map"`, `"raidstate"`, `"exploration"` are sprinkled across `playerinit.go`, `ecsinit.go`, `savehelpers.go`, plus `game_main/setup.go`. A typo wouldn't be caught until runtime.

#### D7. Dead code — `mapgenconfig.go:105-113`

`buildGarrisonRaidGenerator` only ever returns `nil`. The function exists solely to occupy a case in the switch and ends with `return nil // Use registry default`. Either delete the case or actually use it.

### Architecture Debt

#### A1. `GameBootstrap` struct has zero state — `bootstrap.go:27`

```go
type GameBootstrap struct{}
```

Every method is a pure function. The struct adds ceremony (`gamesetup.NewGameBootstrap()` called in `setup.go` at lines 28, 87, 114) with no benefit — no field, no interface, no DI, no lifecycle. Methods would be clearer as package functions.

#### A2. Phase ordering is documented in comments, not enforced — `bootstrap.go:35, 67, 82, 91, 127`

Each method says *"Phase N: Depends on X."* Nothing in the type system prevents calling `CreatePlayer` before `InitializeCoreECS`. Worse: `setup.go` actually invokes them in *different orders* per mode (`SetupOverworldMode` vs `SetupRoguelikeMode`), making the "phases" misleading.

#### A3. Save/load path is a parallel universe — `setup.go:148-207`

`SetupRoguelikeFromSave` duplicates `SetupRoguelikeMode` logic but bypasses `CreatePlayer`, so it manually:

- Re-loads unit templates (lines 164-166, comment admits the divergence)
- Re-rebuilds renderables (`RestoreRenderables`)
- Re-rebuilds walkable grid
- Re-rebuilds the `"players"` tag (in `PlayerChunk.Load`, not here)

Every new bootstrap step needs to be added in two places. The doc in `SAVE_SYSTEM.md:259-260` acknowledges this is a known sharp edge.

#### A4. `log.Fatalf` peppered through init — bootstrap.go:59, 97, 102, 105, 143; moderegistry.go:36, 60, 87

Failures kill the process immediately with no chance to save, log to a structured sink, or return to the start menu. Couples `gamesetup` permanently to process exit. The game has a `pendingLoad` recovery path in `main.go:94` — the same recovery should be possible for init failures.

#### A5. `gamesetup` is becoming a god package

Imports span 30+ packages: `campaign/overworld/{core,ids,node,tick,faction}`, `tactical/{combat/combatmath,commander,powers/{artifacts,perks,spells},squads/{roster,squadcore,unitdefs,squadcommands}}`, `mind/{ai,combatlifecycle,encounter}`, `gui/{framework,guicombat,guicombat/combatanimation,guiexploration,guinodeplacement,guioverworld,guiprogression,guiraid,guisquads,guiunitview,widgetresources}`, `world/{worldgen,worldmapcore,garrisongen}`, `templates`, `testing/bootstrap`, etc. The `moderegistry.go` comment explicitly admits one reason: *"This keeps the mind/ai import in gamesetup (bootstrapping layer) rather than in gui/guicombat."* Wiring concerns are leaking into a single hub.

#### A6. Mode mix-up — methods vs package funcs

`CreateWorld`, `CreatePlayer`, `InitializeGameplay`, `ConvertPOIsToNodes` are methods on `GameBootstrap`. `InitWalkableGridFromMap`, `CreateInitialFactions`, `CreateInitialCommanders`, `CreateSquadsForCommander`, `RegisterTacticalModes`, `ConfigureMapChunk`, `RestorePlayerData`, `RestoreRenderables` are package functions. No convention — caller has to remember which is which.

#### A7. Mutable package globals via `worldgen.ConfigOverride` — `mapgenconfig.go:19`

`InitMapGenConfigOverride` reaches into another package and assigns a function variable: `worldgen.ConfigOverride = func(name string) ...`. Tests, parallel maps, or multi-config scenarios cannot coexist. Same for `garrisongen.SetGarrisonRoomSizes/FloorScaling/SpawnCounts` (lines 127, 142, 151) which mutate package-level tables.

#### A8. `ConvertPOIsToNodes` lives on `GameBootstrap` but is called only from `InitializeGameplay`

It's a method (`bootstrap.go:152`) only used internally. The struct already has no state; the method placement is arbitrary.

### Testing Debt

#### T1. Zero test coverage in `setup/gamesetup/`

- No `_test.go` files
- Squad creators (~200 LOC of random selection + position assignment) are untested
- Mode registration, faction creation, save-load restoration are untested
- `RestoreRenderables` walks four entity classes — any regression only surfaces visually

#### T2. No regression test for the "fresh vs load" parity that `setup.go:163` warns about

A divergence (loaded game missing a piece of init done in fresh path) would not be caught until the player hit a specific code path in a loaded game.

### Documentation Debt

#### Doc1. Stale comments

- `playerinit.go:25`: *"PlayerComponent already registered in componentinit.go"* — file doesn't exist. Registration is in `ecsinit.go:41`.
- `bootstrap.go:72`: *"Phase 0 - MASTER_ROADMAP"* — historical reference to a long-completed roadmap.
- `bootstrap.go:35-127`: Phase numbering implies a strict pipeline that doesn't exist (see A2).

#### Doc2. `JSONSquadSetup` doesn't document the contract with `squadCreators`

The JSON config struct (`templates/initialsetup.go:22`) has `TypePool []string` with no comment pointing at the registry of valid IDs.

### Infrastructure / Robustness

#### I1. Silent no-op on missing config

- `mapgenconfig.go:15-17`: `if cfg == nil { return }` — no log
- `savehelpers.go:24-29`: `if chunk := savesystem.GetChunk("map"); chunk != nil { ... }` — silent if chunk missing; save will fail later
- `savehelpers.go:33-40`: `RestorePlayerData` returns silently if `"players"` tag missing or query empty — load proceeds with broken state

#### I2. `reportCoverage` uses `log.Fatalf` after looping — `bootstrap.go:51-65`

Logs every error then fatals. Reasonable, but the format `"FATAL %s coverage: %v"` uses `log.Printf` with the word "FATAL" then `log.Fatalf` afterwards — confusing for log aggregators expecting actual fatal severity.

---

## 2. Impact Assessment

| ID | Debt | Velocity Impact | Risk |
|----|------|-----------------|------|
| D1 | Squad creator duplication | Each new squad type = ~40 LOC of boilerplate; every formation tweak hits 5 files | **High** — actively blocks tactical content |
| D2 | Image reloaded 4× | Negligible startup time; high if asset list grows; one cache miss per commander on load | Medium |
| D3 / D6 | String key sync between data and code | New types silently fail at runtime; ~1hr per "I forgot to update the validator" | Medium |
| D4 | Mode registration dup | Each new mode = edits in 2+ functions | Medium |
| A1 / A6 | Stateless struct + mixed conventions | Onboarding confusion; ~30min/PR friction reviewing setup changes | Low |
| A2 | Phase ordering only in comments | One reordering bug = game won't boot; hard to diagnose | **High** |
| A3 | Save vs fresh divergence | Every new init step risks a load-only bug; SAVE_SYSTEM.md already calls this out | **High** |
| A4 | `log.Fatalf` everywhere | No graceful error UI; one bad JSON = process death | Medium |
| A5 | God-package imports | Compile-time coupling — every gameplay change drags `gamesetup` rebuild | Medium |
| A7 | Global mutable config | Can't run two map generators with different configs; hostile to tests | Medium |
| T1 / T2 | No tests | All regressions are runtime-only | **High** |

---

## 3. Prioritized Remediation Plan

### Quick Wins (1–2 days)

1. **Extract squad creator helper** — replace `D1` with a single `buildSquadFromArchetype(arch SquadArchetype) (ecs.EntityID, error)` consuming a struct: `{name, unitFilter, gridPositions, count, leadership, formation}`. Move the 5 archetype tables into JSON (`squadarchetypes.json`) or a `var archetypes = map[string]SquadArchetype{...}`. ~200 LOC → ~80 LOC + data.
2. **Cache the player sprite** — load once at `LoadGameData` into a package-level `var playerSprite *ebiten.Image`, replace all 4 in-file loads. (`D2`)
3. **Define string constants** — `const TagPlayers = "players"` etc. in `common/tags.go` or `gamesetup/keys.go`. (`D6`)
4. **Delete dead `buildGarrisonRaidGenerator` case** or finish the implementation. (`D7`)
5. **Strip `GameBootstrap` struct** — convert methods to package functions; delete `NewGameBootstrap()` and its 3 call sites. (`A1`)
6. **Fix stale comments** in `playerinit.go:25` and `bootstrap.go:72`. (`Doc1`)
7. **Log instead of silent-return** in `mapgenconfig.go:15`, `savehelpers.go:24-29`, `savehelpers.go:33-40`. (`I1`)

### Medium-Term (1–3 weeks)

8. **Unify squad type registry** — single source of truth: `squadArchetypes := map[string]SquadArchetype{...}` exposed by `gamesetup` (or `templates`), used both by JSON validation and creation. Eliminates `D3` for squads. Same pattern for faction types.
9. **Merge `RegisterTacticalModes` variants** — one function taking `ModeSetVariant` enum, with a single base mode list and additive deltas. (`D4`)
10. **Add focused tests** — `initial_squads_test.go` verifying each archetype produces N units, has exactly one leader, fills correct grid cells; `mode_registry_test.go` verifying every advertised mode actually registers; `savehelpers_test.go` verifying `RestoreRenderables` populates expected entities. (`T1`)
11. **Return errors instead of `log.Fatalf`** at least from the orchestration layer, letting `main.go` decide whether to fall back to start menu. (`A4`)

### Long-Term (1–2 months)

12. **Unify fresh + load init pipeline** — define a `Pipeline []Phase` with each Phase declaring deps and a `RunFresh / RunFromSave` pair. `setup.go` becomes data, not control flow. Kills the divergence flagged by `A3` + the warning in `setup.go:163`. Pairs with parity tests (`T2`).
13. **Encapsulate map-gen overrides** — replace `worldgen.ConfigOverride` and `garrisongen.SetGarrison*` globals with a `WorldGenContext` value threaded through `CreateWorld`. (`A7`)
14. **Carve up `gamesetup`** — push GUI registration (`moderegistry.go`) into `gui/framework/registry.go`; push factions/commanders/squads into their own `*/setup.go` files in their owning packages, with `gamesetup` reduced to orchestration only. (`A5`)

---

## 4. Prevention

- **Lint rule / CI check**: forbid `log.Fatalf` outside `main.go` and `init()`.
- **Test gate**: require new files in `setup/gamesetup/` to have a corresponding `_test.go`.
- **Single-source registries**: any `var validXIDs = map[string]bool{...}` paired with a switch elsewhere is a smell — extract a single typed registry.
- **No new globals**: a `golangci-lint` `gochecknoglobals` exemption list with hard cap at current count.

---

## Summary

The biggest *current* productivity drag is **D1** (squad creator duplication) — high-frequency change area with a clean refactor path. The biggest *latent* risk is **A3 + T1** (fresh/load divergence with no tests). The biggest *structural* concern is **A5** (gamesetup growing into a god-package as new systems plug in). Quick wins 1–7 are low-risk and recover most of the day-to-day friction; the longer items address the parity/coupling problems before they get worse.
