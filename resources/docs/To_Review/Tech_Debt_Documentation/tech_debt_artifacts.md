# Artifacts Package — Technical Debt Report

**Scope:** `tactical/powers/artifacts/` (18 files, 2,540 lines, ~1,540 prod / ~1,000 test)
**Reviewed:** components.go, artifactcharges.go, artifactinventory.go, balanceconfig.go, behavior.go, behaviors.go, context.go, dispatcher.go, init.go, pending_effects.go, queries.go, registry.go, system.go + 5 test files
**Cross-referenced callers:** `combatservices/`, `guiartifacts/`, `guisquads/`, `setup/savesystem/chunks/gear_chunk.go`, `setup/gamesetup/`, `testing/bootstrap/`

---

## 1. Debt Inventory

### Bugs / Real Defects

| # | Item | Location | Severity |
|---|---|---|---|
| **B1** | **`TesthasSpecificArtifactInFaction`** — lowercase 'h' after `Test` makes Go's test discovery skip this function. The test never runs. | `gear_test.go:315` | High |
| **B2** | `BehaviorSaboteurWsHourglass` constant identifier — typo: `Ws` instead of apostrophe. The string value `"saboteurs_hourglass"` is correct, but the Go identifier shows up in 6 files and tests. | `behavior.go:14`, used 12+ places | Low (cosmetic, but persistent) |
| **B3** | `LoadArtifactBalanceConfig` silently swallows missing/invalid config. If `artifactbalanceconfig.json` is missing in production, `MovementReduction` stays at 0 and Saboteur's Hourglass becomes a no-op artifact with no error surfaced. | `balanceconfig.go:25-39` | Medium |

### Test Coverage Gaps

| # | Item | Impact |
|---|---|---|
| **T1** | **`EngagementChainsBehavior.OnAttackComplete` is untested.** No test exercises the "full move action after a kill" path — one of two reactive behaviors and the only one not covered. | Reactive logic regressions ship silently |
| **T2** | **`EchoDrumsBehavior.Activate` is untested.** `gear_test.go` covers TwinStrike, Saboteur, Deadlock, ChainOfCommand — but the EchoDrums activation path (move/attack precondition, bonus movement, charge consumption) has no test. | Round-charge regressions ship silently |
| **T3** | No save/load round-trip test for `GearChunk`. Save logic at `gear_chunk.go:53-94` is untested in this package. | Save corruption only detected at runtime |
| **T4** | No test for `ValidateBehaviorCoverage` in `registry.go:70-93`. | Drift between JSON and code goes undetected until startup logs are read |
| **T5** | No test asserting `RefreshRoundCharges` runs in `DispatchOnTurnEnd` with multiple round-charge behaviors held simultaneously (only ChainOfCommand tested in isolation). | Order regressions for round vs battle charges go undetected |

### Code Duplication

| # | Item | Location | Lines |
|---|---|---|---|
| **D1** | **Test combat-context boilerplate** — `cache := NewCombatQueryCache; fm := NewCombatFactionManager; CreateCombatFaction; createTestSquadWithUnits; AddSquadToFaction; turnMgr := NewTurnManager; turnMgr.InitializeCombat; charges := NewArtifactChargeTracker; ctx := NewBehaviorContext`. `setupCombatContext` exists at `gear_test.go:501` but is **only used in the file, never adopted by 4 tests that pre-date it** (TestActivateTwinStrike, TestActivateTwinStrike_NotYetAttacked, TestDeadlockShackles_SkipsActivation, TestChainOfCommand_PassFullAction). Same pattern repeated across `dispatcher_test.go`. | gear_test.go, dispatcher_test.go | ~15 lines × 6 = ~90 |
| **D2** | Manual rollback pattern in `EquipArtifact`/`UnequipArtifact`: `data.EquippedArtifacts = append(data.EquippedArtifacts, artifactID)` to undo a slice mutation, twice in 60 lines. | `system.go:122-125`, `system.go:154-160` | Minor |
| **D3** | Inventory query helpers iterate `inv.OwnedArtifacts[id]` with a near-identical loop: `OwnsArtifact`, `IsArtifactAvailable`, `GetInstanceCount`, `getFirstSquadWithArtifact` (test). Could share an `eachInstance(inv, id, predicate)` helper, though current readability is acceptable. | `queries.go:75-93`, `system.go:31-41` | Minor |

### Architecture / Design Debt

| # | Item | Location |
|---|---|---|
| **A1** | **`EquipmentData` lacks `MaxSlots` field** — slot enforcement reads `templates.GameConfig.Player.Limits.MaxArtifactsPerCommander` at every `EquipArtifact` call. Inconsistent with `ArtifactInventoryData` which carries `MaxArtifacts`. Couples runtime equip logic to a global, makes per-squad slot variants impossible. | `system.go:108-110` |
| **A2** | **AOE marker overload of pending queue** — Saboteur's Hourglass is an AOE behavior but uses `pending.Add(key, 0)` to signal "broadcast", and `applyPendingEffects` carries a `broadcast bool` parameter to skip the per-target filter. The queue's `TargetSquadID = 0` has dual meaning (no-target vs all-targets). | `behaviors.go:60-93`, `behaviors.go:128-145` |
| **A3** | **`AllBehaviors()` iterated in `DispatchOnTurnEnd`** — fires `OnTurnEnd` on every registered behavior whether equipped or not. PostReset is correctly scoped to equipped + pending. Inconsistency; trivial cost today but invites dead-hook drift as more behaviors land. | `dispatcher.go:90-95` |
| **A4** | **Per-call slice allocation in hot paths** — `GetEquippedBehaviors` and `GetArtifactDefinitions` each allocate a fresh slice. Called from `DispatchOnAttackComplete` (per attack) and `DispatchPostReset` (per faction reset). Negligible at current scale (<10 artifacts/squad) but a query cache or sync.Pool is the eventual answer. | `queries.go:18-46` |
| **A5** | **`PendingEffectQueue.Consume` rebuilds `effects` slice O(n)** even when consuming a single key. Fine at game scale (<10 entries) but a `slices.DeleteFunc`-style in-place compact would match the surrounding code style. | `pending_effects.go:59-71` |
| **A6** | **Charge-then-mutate ordering footgun in TwinStrike, EchoDrums, ChainOfCommand** — mutate state first, call `UseCharge` last. Currently safe (no failures between mutation and charge), but adds a "Don't add a failable call between these lines" implicit invariant. | `behaviors.go:157-172, 211-261, 274-291` |
| **A7** | `ApplyArtifactStatEffects` has no inverse — cleanup happens at the call site via `effects.RemoveAllEffects` per unit. Implicit pairing; no test/contract enforces apply→remove. | `system.go:170-200` |
| **A8** | `fmt.Printf` for warnings in `system.go:186` (invalid stat modifier) and `balanceconfig.go` (3 sites). No structured logger / no severity levels. | system.go, balanceconfig.go |
| **A9** | `GetTargetType` lives in `guiartifacts/artifact_handler.go:151` but is a thin wrapper over `artifacts.GetBehavior(...).TargetType()`. Dead helper — callers can hit the package directly. | `gui/guiartifacts/artifact_handler.go` |

### Documentation Debt

- `behaviors.go` has good package-level comment explaining the immediate/reactive/deferred trichotomy. **Individual behaviors lack contract docs** (preconditions, side effects, charge type rationale). E.g. ChainOfCommand has 4 distinct precondition checks with no doc comment summarizing them.
- `BehaviorContext.SetSquadLocked` and `ResetSquadActions` are useful primitives but undocumented as the "preferred" mutation API for behaviors (vs direct actionState mutation in TwinStrike/EchoDrums). Inconsistent usage.

### Test Quality Issues

- **Global state mutation** — `setupTestArtifacts` reassigns `templates.ArtifactRegistry`, `setupDispatcherArtifacts` mutates `ArtifactBalance.SaboteursHourglass.MovementReduction`, `setupTestManager` mutates `templates.GameConfig`. Tests are **not safe under `t.Parallel()`** even though nothing currently uses it.
- Hand-built artifact registry fixtures (`setupTestArtifacts`) drift from production JSON. No fixture loaded from real JSON files.

---

## 2. Impact Assessment

| Item | Risk | Practical impact |
|---|---|---|
| **B1** (lowercase `TesthasSpecificArtifactInFaction`) | High | A test you believe is running isn't. Silent regression risk on `hasSpecificArtifactInFaction` helper logic. Five-character fix. |
| **T1, T2** (untested behaviors) | High | EngagementChains and EchoDrums are 2 of 6 major artifacts. If the test suite is the gate for changes, those 2 ship blind. ~33% of major artifact behaviors are untested. |
| **B3** (silent balance config failure) | Medium | A typo or missing file in `gamedata/artifactbalanceconfig.json` makes Saboteur's Hourglass a no-op with no error. Combat plays through "successfully" — masks data drift. |
| **A1** (global config in equip path) | Medium | Blocks per-squad slot count features (e.g. an artifact that grants +1 slot, or a commander class with different capacity). |
| **A2** (AOE-via-pending overload) | Medium | The next AOE behavior added will likely either reinvent the broadcast mechanism or copy the `targetSquadID = 0` convention without realizing it's a marker. |
| **D1** (test boilerplate) | Low | Wasted lines, but `setupCombatContext` already exists — half-finished refactor. |
| **A6** (charge-then-mutate ordering) | Low | No present bug, but the next contributor adding validation between mutation and charge will create a "consumed charge but mutation rolled back" bug. |
| **A8** (fmt.Printf warnings) | Low | Production logs are noisy and ungreppable; harmless functionally. |

---

## 3. Prioritized Roadmap

### Quick Wins (this sprint, ~4 hours total)

1. **[B1, 5 min]** Rename `TesthasSpecificArtifactInFaction` → `TestHasSpecificArtifactInFaction` so it actually runs. `gear_test.go:315`.
2. **[T1, ~30 min]** Add `TestEngagementChains_FullMoveAfterKill` covering the OnAttackComplete path: assert `MovementRemaining = squadSpeed` and `HasMoved = false` after a kill, no-op when attacker also dies, no-op when target survives.
3. **[T2, ~30 min]** Add `TestEchoDrums_Activate` covering: requires HasMoved+HasActed, resets HasMoved, restores movement, consumes round charge, second activation fails.
4. **[B3, ~15 min]** Make `LoadArtifactBalanceConfig` return an error and have `setup/gamesetup/bootstrap.go` fail-fast in dev (matches `reportCoverage` pattern already used for `ValidateBehaviorCoverage`).
5. **[B2, ~15 min]** Rename `BehaviorSaboteurWsHourglass` → `BehaviorSaboteursHourglass` (the string value stays `"saboteurs_hourglass"`). Single-shot find-and-replace.
6. **[D1, ~45 min]** Refactor the 4 tests in `gear_test.go` that pre-date `setupCombatContext` to use it. Eliminates ~60 lines of duplicate setup.
7. **[A9, ~5 min]** Delete the `GetTargetType` wrapper in `guiartifacts/artifact_handler.go:151-157` and inline `b.TargetType()`.

### Medium-Term (1–2 sprints)

8. **[T3]** Save/load round-trip test for `GearChunk.Save` → `Load` → `RemapIDs`. Worth 1 day. Save corruption is the worst class of latent bug.
9. **[T4]** Coverage test asserting every JSON-defined `Behavior` has a registered implementation, including a synthetic test that injects a bad def and expects errors.
10. **[A2]** Replace AOE-via-zero-target convention with explicit `BroadcastEffect` flag on `PendingArtifactEffect`. Removes the `broadcast bool` parameter from `applyPendingEffects` and the dual meaning of `TargetSquadID = 0`.
11. **[A3]** Scope `DispatchOnTurnEnd` to equipped behaviors (mirror the `DispatchPostReset` pattern). Or: keep firing all behaviors but add a comment justifying the asymmetry.
12. **[A6]** Move charge consumption to **before** state mutation in TwinStrike/EchoDrums/ChainOfCommand. Reverse rollback risk by paying the charge first; if a future precondition fails between charge and mutation, you spend a charge but no state changed — a less destructive failure mode.

### Long-Term (next quarter, only if scope grows)

13. **[A1]** Add `MaxSlots` field to `EquipmentData` populated from config at squad creation; remove the global lookup in the equip path. Also enables per-commander/per-squad slot variants.
14. **[A4]** If artifact count per squad grows beyond ~5 routinely, cache `GetEquippedBehaviors` per squad with invalidation on equip/unequip.
15. Behavior contract docs — add 3-line preamble to each behavior describing preconditions and effects, mirroring the package-level docs.

---

## 4. Implementation Strategy — Recommended PR Sequence

```
PR 1 (Quick Wins): B1 + B2 + T1 + T2 + B3 + A9 — ~2 hours, all isolated
PR 2 (Test cleanup): D1 — adopt setupCombatContext everywhere
PR 3 (Save round-trip): T3 — separate because save tests need their own fixture work
PR 4 (Charge ordering): A6 — touches 3 behaviors, needs careful review
PR 5 (AOE refactor): A2 — touches PendingArtifactEffect public type, all callers
```

Each is independently revertible. PR 1 alone closes the highest-impact bugs and lifts coverage.

---

## 5. Prevention Plan

1. **Test discovery lint** — `go vet` doesn't catch `Testfoo` (lowercase second char). Add a `golangci-lint` check or a 3-line `pre-commit` script that greps for `^func Test[a-z]` in `_test.go` files. This **alone** would have caught B1.
2. **Behavior coverage test** — extend `TestBehaviorRegistry_AllRegistered` to also assert each behavior has an `Activate`/`OnAttackComplete`/`OnPostReset` test by name — fail CI when a behavior is added without a corresponding test function.
3. **Config load fail-fast in dev** — `bootstrap.go` already does this for `ValidateBehaviorCoverage`. Apply the same gate to `LoadArtifactBalanceConfig` (B3 fix institutionalizes this).
4. **Document the charge-mutation ordering rule** — once A6 is done, add a `behaviors.go` package comment: "Always consume charges before mutating state, so failed mid-flow operations don't roll back into a refunded charge."

---

## 6. Summary

The package is **well-organized and small** (1,540 prod lines, clear file separation, good package-level docs in `behaviors.go` and `pending_effects.go`). The biggest issues are **tactical**, not architectural:

- **2 bugs and 5 test gaps** ship in roughly 4 hours of work (PR 1).
- The package's design is sound — pure-data components, registered behaviors, dispatcher pattern, value-embedded context. No god classes, no circular deps, no inappropriate intimacy.
- The "deferred effect" two-phase queue is one of the cleaner patterns in the codebase. Don't refactor the core; clean up the warts.

**Recommended action: ship PR 1 this week.** It removes all High-severity items (B1, T1, T2) and one Medium (B3) for ~2 hours of effort. The rest is optional polish that scales with how much new artifact content you plan to add.
