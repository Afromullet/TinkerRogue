# Technical Debt Report: `tactical/combat/`

**Scope:** 6 packages, 27 files, ~6,566 LOC (~3,557 production + ~3,009 test)
**Date:** 2026-05-15

---

## 1. Debt Inventory

### Package LOC & Test Coverage

| Package | Prod LOC | Test LOC | Coverage | Risk |
|---|---|---|---|---|
| `combatcore` | 1,154 | 2,454 | **48.5%** | Medium |
| `combatservices` | 616 | 85 | **4.0%** | **Critical** |
| `combatstate` | 584 | 0 | **0.0%** | High |
| `combatmath` | 704 | 0 | **0.0%** | **Critical** |
| `battlelog` | 757 | 0 | **0.0%** | High |
| `combattypes` | 212 | 0 | n/a (types) | Low |

`combatmath` containing damage calculation, hit/dodge/crit rolls, and target selection at **0% coverage** is the largest quality risk in the entire combat folder.

### Code Debt (by file/severity)

| # | Location | Type | Detail |
|---|---|---|---|
| C-1 | `combatservices/combat_service.go` (523 LOC) | God class | 11 fields, 25 methods, owns lifecycle + power dispatch + threat + AI + GUI callbacks + faction safety net + cleanup. Touches 9 sibling packages. |
| C-2 | `combatcore/combatprocessing.go:24-133` `processAttack` (109 LOC) | Long function + nested nil checks | 6 separate `if dispatcher != nil` branches gate the perk pipeline. Replace with a non-nil default ("null object") dispatcher; passing `nil` is the only reason these checks exist. |
| C-3 | `combatcore/combatabilities.go:65-94, 113-126` | Switch-on-enum dispatch | `evaluateTrigger` and `executeAbility` are parallel switches on `TriggerType` / `AbilityType`. Four `apply*Effect` helpers (Rally / Heal / BattleCry / Fireball) share the same shape but no abstraction. Adding a new ability requires edits in 3 places. |
| C-4 | `combatcore/combatexecution_test.go` (1,641 LOC, 40 tests) | Bloated test file | Single file accounts for 25% of combat LOC. Should split into per-feature files (targeting / counterattack / damage / abilities). |
| C-5 | `combatservices/combat_service.go:489-523` | Near-duplicate dispose loops | `disposeEntitiesByTag`, `disposeEnemySquads`, `disposeEnemyUnits` are three near-identical query+dispose loops with different filter predicates. |
| C-6 | 20× `fmt.Printf` calls across 6 files | Print-debugging in prod | `[ABILITY]`, `[GEAR]`, `WARNING:`, `=== Combat Teardown ===` strings printed to stdout. No log level, no toggle. The PowerLogger framework already exists — ability/teardown code bypasses it. |
| C-7 | `combatmath/combattargeting.go` | Magic constants | Hardcoded `3` (grid size) and `2` (max rows) repeated in 6 places: `selectMeleeRowTargets`, `selectMeleeColumnTargets`, `getUnitsInLine`, `selectLowestArmorTarget`. Already a constant `squadcore.GRID_SIZE` exists elsewhere — not used here. |
| C-8 | `combatcore/combatabilities.go:40,58` | Magic constants | Hardcoded `4` (ability slot count) in two loops. Should be `len(abilityData.Slots)`. |
| C-9 | `combatcore/combatcalculation.go:128` | Magic constant | `int(float64(baseDamage) * 1.5)` — crit multiplier hardcoded while counterattack multiplier is configurable via `combatbalanceconfig.json`. Asymmetric: half the tuning levers exist, half are baked in. |
| C-10 | `battlelog/battle_summary.go:232-238` `getTargetName` | Stub / known-broken | Returns `fmt.Sprintf("Unit_%d", id)` placeholder instead of looking up the real name. Documented as "In a real implementation". |
| C-11 | `combatmath/combattargeting.go:113-132` vs `:219-245` | Code duplication | `selectMagicTargets` and `selectHealTargets` are ~85% identical: iterate `targetCells`, dedupe via `seen` map, gather from `GetUnitIDsAtGridPosition`. Differ only in alive/HP filter. |
| C-12 | `combatservices/combat_service.go:214-237` | "Safety net for a starter bug" | `assignDeployedSquadsToPlayerFaction` logs warnings and patches state because the starter "should have enrolled it." The comment admits this masks a bug — it should fix the bug, not paper over it. |
| C-13 | `combatcore/turnmanager.go:62-63` | Stale TODO | "Abilities are not yet implemented, but I foresee not needing this." Yet `CheckAndTriggerAbilities` is called on line 70 — so abilities **are** wired. Comment is misleading. |

### Architecture Debt

| # | Location | Type | Detail |
|---|---|---|---|
| A-1 | `combatstate/combatcomponents.go:26-28` + `combatqueriescache.go` | Two parallel caching mechanisms | Package-level `ecs.View`s (`factionView`, `combatSquadView`) AND `CombatQueryCache` struct with its own `ActionStateView`/`FactionView`. `FactionView` is duplicated in both. New code is unclear which to use. |
| A-2 | `combatstate/combatqueries.go:199-218` `RemoveSquadFromMap` | Mixed responsibility | Lives in `combatstate` (a state/query package) but performs entity disposal (lifecycle). Disposes via `squadcore.DisposeSquadAndUnits` — crosses package layers. |
| A-3 | `combatservices/combat_service.go:78-150` `NewCombatService` (73 LOC) | Hidden coupling in constructor | Constructor knows about pipeline subscriber ordering for artifacts→perks→GUI across 4 events. Adding a new power system requires deep knowledge of this method. |
| A-4 | `combatservices/ai_interfaces.go` + `SetAIController`/`SetThreatProvider`/`SetThreatEvaluatorFactory` | Setter injection working around import cycle | Three separate injection points to avoid `ai → combatservices` cycle. Indicates the `combatservices` package is on the wrong side of the dependency graph for AI types. |
| A-5 | `battlelog/combatlogging.go:5` | Layering violation | `battlelog` (logging concern) imports `combatmath` (calc concern) for `CanUnitAttack`. Should call through `combatstate` or expose a query in `squadcore`. |
| A-6 | `combatabilities.go:96-111` `countEnemySquads` | O(n) full-world scan | Iterates all squads and calls `GetSquadFaction` on each — when a cached faction-aware view exists in the same package (`combatSquadView` in `combatqueries.go`). Called per-ability-check per-turn. |
| A-7 | `combatcore/combatabilities.go:194-204` `applyFireballEffect` | Wrong target selection (BUG) | "Pick first different squad" — does not consider faction. A Fireball cast by enemy A targets enemy B if iterated first. Bug, not just debt. |
| A-8 | `combatservices/combat_service.go:443-457` `cleanupEffects` | Over-broad cleanup | Iterates all `SquadMemberTag` entities (including enemy ones already being disposed seconds later). Should filter to player squads via `collectPlayerSquadIDs`. |
| A-9 | `combatcore/combatactionsystem.go` + `combatprocessing.go` | Damage-pipeline coupling | `processAttack` mutates a `*combattypes.CombatResult` and a `*combattypes.CombatLog` while running 9 perk hooks and a damage-redirect side effect. Hard to reason about, hard to test in isolation, hard to add features without regressions. |
| A-10 | `combattypes/combattypes.go:174-189` `CombatResult` | God struct | One struct mixes orchestration status (`Success`, `ErrorReason`, `*Destroyed` flags) with execution data (damage maps, kill list) with display data (`CombatLog` for names). |

### Testing Debt

| # | Issue | Detail |
|---|---|---|
| T-1 | **Zero coverage** on damage math | `combatmath` has 0 tests. Hit/dodge/crit/resistance/cover/heal all untested. This is the most game-affecting code in the folder. |
| T-2 | **Zero coverage** on state management | `combatstate` has 0 tests. Faction membership, action state, ZoC, victory checks — all untested at the unit level (only exercised transitively through `combatcore` tests). |
| T-3 | **Zero coverage** on battle log | `battlelog/battle_summary.go` (409 LOC) — the most complex non-math file in the folder — has no tests. `getTargetName` ships broken (C-10) precisely because nothing exercises it. |
| T-4 | Test cross-package coupling | `combatcore/combat_test_helpers_test.go` builds fixtures the math/state packages need but can't share without an extracted `testfx` package. Currently each package would have to re-implement. |
| T-5 | Vanity tests | `TestCombatResult_Structure` (combat_service_test.go:55-85) only sets fields and reads them back — tests Go's struct semantics, not your code. Misleading coverage signal. |
| T-6 | `combatexecution_test.go` reimplements `ExecuteAttackAction` | `executeTestAttack` (lines 22-54) duplicates the real pipeline minus counter/post-combat phases. Tests pass on a parallel implementation — production divergence won't be caught. |

### Documentation Debt

The combat folder is **better-documented than typical** — most public functions and complex internal flows have prose comments. Remaining gaps:

- **D-1**: No package-level `doc.go` in any of the 6 packages. New contributors must read code to learn what each package owns.
- **D-2**: `COMBAT_PIPELINES.md` exists but doesn't reflect the recent `powerPipeline` refactor where four `Fire*` methods collapsed into declarative subscribers.

---

## 2. Impact Assessment

**Velocity-impacting items** (ordered by hours/month):

| Item | Symptom | Est. hrs lost / month |
|---|---|---|
| T-1, T-2, T-3 (no tests on math/state/log) | Every change to damage/cover/ZoC requires manual verification; regressions land silently | 12–20 |
| C-2 (nil dispatcher checks) | Every perk-pipeline edit must thread `if dispatcher != nil` through 6 sites; tests must remember to pass nil OR real dispatcher | 3–5 |
| C-3 (ability switch dispatch) | Each new leader ability touches 3 switches in `combatabilities.go` | 2–4 per ability |
| A-3, A-4 (CombatService god class) | Any new system (status effects, terrain) needs surgery on `NewCombatService` + an injection setter | 4–8 |
| C-1 (combat_service.go length) | Hard to navigate; reviewers miss interactions across the 11 fields | 2–3 |

**Risk-bearing items** (ordered by blast radius):

- **A-7 (Fireball wrong target)**: Active bug. Cross-faction or 3-faction encounters target the wrong squad.
- **C-10 (`getTargetName` stub)**: Exported battle JSON contains `Unit_42` placeholders — breaks downstream `tools/combat_analysis/` consumers.
- **C-12 (faction safety net)**: A starter bug is being silently patched with a warning, meaning the actual bug recurs with no test catching it.
- **C-6 (stdout debug prints)**: 20 unconditional `fmt.Printf`s ship in release builds. Performance is fine, but the output is unsilenceable.

---

## 3. Prioritized Remediation Plan

### Quick Wins (1–2 days, immediate payoff)

1. **Fix Fireball target selection** (A-7) — _~30 min_. The "pick first different squad" needs `combatstate.GetSquadFaction` comparison. **Ship a test alongside this fix.**
2. **Replace nil-dispatcher checks with null-object** (C-2) — _~2 hr_. Introduce `combattypes.NoopPerkDispatcher{}` and pass it instead of `nil`. Removes 7 branches across 2 files. Tests that pass `nil` get a one-line fixture change.
3. **Promote crit multiplier to balance config** (C-9) — _~30 min_. Move `1.5` into `combatbalanceconfig.json` next to counterattack tuning.
4. **Delete vanity test** (T-5) — _~5 min_. `TestCombatResult_Structure` provides no signal.
5. **Replace hardcoded `3` and `4`** (C-7, C-8) — _~1 hr_. Use existing constants/`len()`.
6. **Fix `getTargetName`** (C-10) — _~30 min_. Take a `*common.EntityManager` and look up `common.NameComponent` like `snapshotUnits` already does.
7. **Delete stale TODO** (C-13) — _~5 min_. The "abilities I foresee not needing" comment contradicts the code.

**Estimated effort:** ~6 hours total → unblocks 3–5 hours/month of future churn.

### Medium-Term (1–3 weeks)

8. **Add `combatmath` test suite** (T-1) — _~12 hr_. Unit tests for `CalculateDamage`, `CalculateHealing`, `selectMeleeRowTargets`, etc. These are pure functions and easy to test. Target 70% coverage.
9. **Add `combatstate` test suite** (T-2) — _~8 hr_. Faction membership lifecycle, action state mutations, ZoC adjacency, victory-check enumeration.
10. **Extract testing fixtures into `tactical/combat/combattestfx`** (T-4) — _~4 hr_. `combatcore/combat_test_helpers_test.go` already has `CreateTestCombatManager`, `CreateTestSquad`. Lift to a non-test package so all combat sub-packages can share.
11. **Polymorphic ability dispatch** (C-3) — _~6 hr_. Replace dual switches with a `LeaderAbility` interface (`Evaluate(...) bool; Apply(...)`) and a registry. Each ability becomes one ~30 line file.
12. **Consolidate dispose helpers** (C-5) — _~2 hr_. One `disposeMatching(tag, predicate, cleanup)` covers all three.
13. **Resolve faction-safety-net** (C-12) — _~3 hr_. Find which starter path misses enrollment, fix it, then delete `assignDeployedSquadsToPlayerFaction`.
14. **Replace `fmt.Printf` with PowerLogger** (C-6) — _~3 hr_. Threading logger into `combatabilities.go` and `combat_service.go` teardown is straightforward; `PowerLogger` already exists in `powercore`.

**Estimated effort:** ~38 hours → coverage from ~25% to ~60% in critical paths.

### Long-Term (1–2 quarters)

15. **Split `CombatService`** (C-1, A-3) into three:
    - `CombatService` — system ownership + lifecycle
    - `PowerOrchestrator` — pipeline + dispatcher wiring (currently lines 95–149)
    - `CombatExitController` — flee/victory state (currently lines 56–58 + 256–285)
    - _~16 hr_
16. **Move `RemoveSquadFromMap` out of `combatstate`** (A-2) — into a `combatlifecycle` helper or `squadcore`. _~3 hr_
17. **Resolve dependency-graph for AI** (A-4) — introduce a `tactical/combat/aicontract` package that both `combatservices` and `mind/ai` import, instead of three setter-injection points. _~6 hr_
18. **Split `combatexecution_test.go`** (C-4) into `targeting_test.go`, `counterattack_test.go`, `damage_pipeline_test.go`, `abilities_test.go`. _~4 hr_
19. **Reconcile two faction caches** (A-1) — pick one, delete the other, and document it. _~2 hr_
20. **Refactor `processAttack`** (A-9) — extract `runAttackerHooks`, `runDefenderHooks`, `applyDeathOverride` so the main loop is 25 lines instead of 95. _~6 hr_

**Estimated effort:** ~37 hours.

---

## 4. Implementation Strategy

**For null-object dispatcher (Quick Win 2):**

```go
// combattypes/perk_callbacks.go
type NoopPerkDispatcher struct{}

func (NoopPerkDispatcher) AttackerDamageMod(...)     {}
func (NoopPerkDispatcher) DefenderDamageMod(...)     {}
func (NoopPerkDispatcher) CoverMod(...)              {}
func (NoopPerkDispatcher) TargetOverride(_, _, t, _) []ecs.EntityID { return t }
// ...etc
```

Then in `combatactionsystem.go`:
```go
func NewCombatActionSystem(manager, cache) *CombatActionSystem {
    return &CombatActionSystem{
        manager: manager, combatCache: cache,
        perkDispatcher: combattypes.NoopPerkDispatcher{},  // ← default, no nil
    }
}
```

Drop the 7 `if dispatcher != nil` checks in `combatprocessing.go` — they become unreachable.

**For polymorphic abilities (Medium-Term 11):**

```go
// combatcore/abilities/ability.go
type LeaderAbility interface {
    ShouldTrigger(slot *squadcore.AbilitySlot, squadID ecs.EntityID, em *common.EntityManager) bool
    Apply(slot *squadcore.AbilitySlot, squadID ecs.EntityID, em *common.EntityManager)
}

var registry = map[squadcore.AbilityType]LeaderAbility{
    squadcore.AbilityRally:     RallyAbility{},
    squadcore.AbilityHeal:      HealAbility{},
    squadcore.AbilityBattleCry: BattleCryAbility{},
    squadcore.AbilityFireball:  FireballAbility{},
}
```

`CheckAndTriggerAbilities` shrinks to ~20 lines and adding an ability is one file.

---

## 5. Prevention

- **Coverage gate**: CI fails if `combatmath` or `combatstate` drops below 70% once seeded.
- **Layer check**: forbid `battlelog → combatmath` import in a `go vet`-equivalent or arch-test (already a stated layering principle in `CLAUDE.md`).
- **No-`fmt.Printf` rule** in `tactical/combat/**` (excluding tests). Use `powercore.PowerLogger`.
- **Magic-number lint**: `3` and `4` repeated in `combatmath`/`combatabilities` should resolve to named constants.

---

## 6. Summary

The combat folder is **structurally healthier than typical** — clean ECS patterns, no entity pointers, decent comments, a real pipeline abstraction in `PowerPipeline`. Its debt clusters in three places:

1. **Testing**: 4 of 6 packages have 0% coverage including `combatmath` (damage math) and `battlelog` (export to tooling). This is the single biggest risk.
2. **`CombatService` and `processAttack`**: both are doing too much; they soak up new-feature complexity instead of distributing it.
3. **Dispatcher nil-checks and ability switch statements**: small but pervasive friction every time the perk or ability system grows.

Quick wins listed above clear ~6 hours of fixes that immediately reduce future maintenance cost. Medium-term coverage work (~38 hr) is the highest-leverage investment — every other refactor on this list is risky to attempt without it.
