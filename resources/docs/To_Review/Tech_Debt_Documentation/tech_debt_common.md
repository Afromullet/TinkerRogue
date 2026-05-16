# Technical Debt Analysis: `core/common`

**Date:** 2026-05-16
**Scope:** `core/common/` package — 7 files, 821 LOC of production code + 217 LOC of tests

| File | LOC |
|------|-----|
| `commoncomponents.go` | 172 |
| `ecsutil.go` | 267 |
| `playerdata.go` | 42 |
| `positionsystem.go` | 210 |
| `randnumgen.go` | 50 |
| `resources.go` | 80 |
| `positionsystem_test.go` | 217 |

The `core/common` package is foundational. Cross-codebase usage:

- Component-access helpers (`GetComponentType`, `GetComponentTypeByID`, `manager.GetComponent`, `manager.HasComponent`): **515 occurrences across 108 files**
- `PlayerData` / `PlayerInputStates`: **39 files**
- `GlobalPositionSystem`: **40 call sites across 14 files**
- `ResourceStockpile` API: **6 files**
- RNG helpers: **30 files**

Defects in this package have wide blast radius and disproportionate cost.

---

## 1. Debt Inventory

### Critical — Correctness Bugs / Dead Code

| # | Item | File:Line | Severity |
|---|------|-----------|----------|
| 1 | `PositionSystem.GetEntityAt` is dead code with an O(n) ECS query inside what should be O(1) | `positionsystem.go:43-57` | **Critical** |
| 2 | `EntityManager.MoveEntity` silently returns `nil` when entity has no position — caller cannot detect no-op | `ecsutil.go:115-118` | **High** |
| 3 | `PositionSystem.AddEntity` declares `error` return but always returns `nil` — API lie | `positionsystem.go:75-88` | Medium |
| 4 | `MoveSquadAndMembers` drops `MoveEntity` errors for member units | `ecsutil.go:151-159` | **High** |
| 5 | Two functions named `MoveEntity` (`EntityManager` and `PositionSystem`) with different semantics; the `EntityManager` one re-implements Add/Remove instead of delegating | `ecsutil.go:108`, `positionsystem.go:118` | **High** |

**Bug detail (item 1):**
`GetEntityAt` body comments claim "bytearena/ecs doesn't have GetEntityByID, so we search" — but `EntityManager.FindEntityByID` (line 93) and `GetComponent` (line 52) already use `em.World.GetEntityByID(entityID)`. The comment is false. The function is **never called** anywhere outside its own file. 18 lines of dead, slow, lying code.

### High — Architecture / API Inconsistency

| # | Item | File:Line | Severity |
|---|------|-----------|----------|
| 6 | `PlayerAttributes()` returns a freshly-allocated zero `*Attributes` on miss — caller cannot distinguish missing-entity from real zero stats; allocates garbage | `playerdata.go:34-42` | **High** |
| 7 | `GetEntityName` and `PlayerAttributes` use the verbose `manager.GetComponent` + type-assert pattern, ignoring the typed helpers (`GetComponentTypeByID[T]`) the same package defines | `ecsutil.go:260-267`, `playerdata.go:34-42` | Medium |
| 8 | `ResourceStockpile` API is procedural (`CanAffordGold(s, n)`) while `Attributes` is method-based (`a.GetPhysicalDamage()`) — asymmetric in the same package | `resources.go:30-74` | Medium |
| 9 | Default `MovementSpeed: 3` / `AttackRange: 1` hard-coded in `NewAttributes` but `GetMovementSpeed`/`GetAttackRange` fall back to `config.Default*` — two sources of truth | `commoncomponents.go:51-52` vs `154-167` | Medium |
| 10 | `positionsystem.go:1` package doc says `// Package systems provides…` but file is `package common` | `positionsystem.go:1-3` | Low |

### Medium — Architecture

| # | Item | File:Line | Severity |
|---|------|-----------|----------|
| 11 | `GlobalPositionSystem` is a global mutable singleton — tests cannot run in parallel, init order matters. Acknowledged by CLAUDE.md but not documented in the file itself as a deliberate trade-off | `ecsutil.go:21` | Medium |
| 12 | Subsystem registrar pattern relies on undocumented `init()` ordering between packages. If two subsystems ever gain a dependency on each other's init, this will break silently | `ecsutil.go:230-257` | Medium |
| 13 | `PositionSystem.GetEntitiesInRadius` scans the bounding box (O((2r+1)²)) instead of iterating `spatialGrid` and filtering. Fine for small radius; pathological for AOE > 8 | `positionsystem.go:194-210` | Low |

### Test Coverage Gaps

| File | Production LOC | Test LOC | Coverage |
|------|----------------|----------|----------|
| `positionsystem.go` | 210 | 217 | Good |
| `ecsutil.go` | 267 | 0 | **0%** |
| `commoncomponents.go` | 172 | 0 | **0%** |
| `resources.go` | 80 | 0 | **0%** |
| `randnumgen.go` | 50 | 0 | **0%** |
| `playerdata.go` | 42 | 0 | **0%** |

**Untested critical paths:**

- `MoveEntity`, `MoveSquadAndMembers`, `RegisterEntityPosition`, `UnregisterEntityPosition`, `CleanDisposeEntity` — all 5 functions enforce the position-sync invariant the entire codebase depends on. Zero tests.
- All 16 `Attributes` derived-stat formulas (damage, hit rate, dodge, capacity, etc.) — silent off-by-N here ships as a combat-balance bug.
- `SpendGold` / `SpendMaterials` — money-burning functions used by purchase flows. No tests.
- `GetDiceRoll` / `GetRandomBetween` — every randomness consumer depends on the bound semantics (inclusive vs exclusive). No tests.

---

## 2. Impact Assessment

This is a solo-developed game, so impact is rated by **bugs-per-quarter likelihood** and **debug-cost-per-incident**, not dollar figures.

| Debt Item | Likelihood of biting | Debug cost if it does | Priority |
|-----------|---------------------|----------------------|----------|
| #2 `MoveEntity` silent skip | High — easy to call on entity that lost its position | High — position desync surfaces as "entity not at click target", hours to trace | **P0** |
| #4 dropped errors in `MoveSquadAndMembers` | Medium — only happens for malformed unit lists | High — half-moved squad is a nightmare to debug | **P0** |
| Test gaps on `Attributes` formulas | High — formulas get tweaked frequently for balance | High — wrong numbers ship and corrupt save data | **P0** |
| Test gaps on movement helpers | Medium | Critical — position invariants underlie everything | **P0** |
| #6 `PlayerAttributes` zero-on-miss | Low — player rarely missing | Medium — silent stat corruption | P1 |
| #9 duplicate movement/range defaults | Certain to bite when balancing | Low — quick to fix once spotted | P1 |
| #1 dead `GetEntityAt` | None (unused) — but invites someone to start using it | Medium — perf regression | P1 (delete) |
| #11 global `GlobalPositionSystem` | Low (working as designed) | n/a | P3 (document) |

---

## 3. Quick Wins (1–2 hours total)

1. **Delete `PositionSystem.GetEntityAt`** (`positionsystem.go:40-57`). Zero callers. Removes 18 lines of misleading code. ~5 min.
2. **Fix package doc comment** at `positionsystem.go:1-3` — change "Package systems" → "Package common". 1 min.
3. **Change `EntityManager.MoveEntity` to return an explicit error** when no position component, OR rename to `TryMoveEntity` and return `(bool, error)`. Update the 1 caller (`MoveSquadAndMembers`) to handle accordingly. ~30 min.
4. **Drop `error` return from `PositionSystem.AddEntity`** — it never fails. Update ~10 call sites. ~20 min.
5. **Handle errors in `MoveSquadAndMembers`** — collect into a multi-error or log+continue, but don't silently swallow. ~15 min.
6. **Consolidate `Attributes` defaults** — delete the magic numbers in `NewAttributes`, set fields to 0 and let the getters fill from `config.Default*`. ~10 min.
7. **Rewrite `GetEntityName` and `PlayerAttributes` to use `GetComponentTypeByID`** — leads by example. ~10 min.
8. **Remove the stray blank line in the `var` block** at `playerdata.go:10-13`. 30 sec.

**Total effort: ~2 hours. Eliminates 5 of 10 medium+ items and 2 of the bugs.**

---

## 4. Medium-Term (1 day each)

1. **Test suite for `Attributes` derived stats** — table-driven test covering all 16 formulas at min/typical/max input. Captures balance intent and prevents silent regressions. ~3 hrs.
2. **Test suite for ECS position invariants** — `MoveEntity`, `MoveSquadAndMembers`, `Register/UnregisterEntityPosition`, `CleanDisposeEntity`. Use `ValidatePositionSync` as the oracle after each operation. ~4 hrs.
3. **Refactor `EntityManager.MoveEntity` to delegate to `PositionSystem.MoveEntity`** instead of re-implementing Remove+Add. Removes duplication, single source of truth for position-system updates. ~30 min + tests.
4. **Convert `ResourceStockpile` to method receivers** — `(s *ResourceStockpile) CanAffordGold(n) bool` etc. 6 call sites to update. ~1 hr.
5. **Tests for `randnumgen.go`** — verify inclusive/exclusive bounds, seed determinism. Cheap insurance for a system every other system depends on. ~1 hr.

---

## 5. Long-Term (only if pain materializes)

- **Replace `GlobalPositionSystem` singleton with injection** — currently tolerable for a solo project; only worth doing if parallel tests or snapshot/replay support are needed.
- **Optimize `GetEntitiesInRadius`** — iterate occupied positions instead of bounding box. Worth doing only if profiler shows it.
- **Document subsystem init-order invariant** — add a comment at `RegisterSubsystem` explaining that subsystems must not depend on each other's components at registration time.

---

## 6. Prevention

For a solo project, lightweight gates make sense:

- Add `go test ./core/common/...` to the pre-commit script. The package is small enough to run fast.
- When adding an exported function to `core/common`, write at least one test in the same change. The package is touched by 108 files — undertested helpers here propagate bugs widely.
- Treat any function that returns `error` and "always returns nil" as a code smell — either it can fail and should, or it shouldn't return `error`.
- The `common` package should set the example for the patterns CLAUDE.md prescribes. Right now `GetEntityName` and `PlayerAttributes` violate the rules CLAUDE.md gives to everyone else.

---

## 7. TL;DR — Do This First

1. Delete `PositionSystem.GetEntityAt` (dead, broken, lying comment).
2. Make `MoveEntity` either return a real error or be explicit it's a no-op.
3. Stop dropping errors in `MoveSquadAndMembers`.
4. Write a table-driven test for `Attributes` derived stats — a balance regression here ships silently.
5. Fix the wrong `Package systems` comment in `positionsystem.go`.

Items 1, 3, 5 are 10-minute fixes with no downside. Item 4 is the single highest-value test you can add to this package.
