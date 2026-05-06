# Technical Debt Analysis: `artifacts` Package

**Date:** 2026-04-30 (verified against codebase 2026-05-06)
**Scope:** `tactical/powers/artifacts/` (17 files) + `gui/guiartifacts/` (3 files)

---

## Verification Notes

The original document was accurate. All items below were confirmed against the live code. Two items have partial mitigations that reduce their actual impact; those are noted inline.

---

## Debt Inventory

### 1. Silent Error Swallowing in GUI Handler — HIGH

`gui/guiartifacts/artifact_handler.go:186`:

```go
artifacts.ActivateArtifact(behaviorKey, targetSquadID, ctx)

// Clear artifact state regardless of success
h.deps.BattleState.InArtifactMode = false
```

The error return from `ActivateArtifact` is discarded. If activation fails (charge already spent, invalid target, nil action state), the panel closes as if it worked and the player gets no feedback. Every behavior added to this system is at risk of silent failure by default.

**Fix (15 min):** Capture the error. At minimum log it before clearing state. Optionally surface it in the UI.

---

### 2. Non-Deterministic `GetAllInstances` Order — MEDIUM (downgraded from HIGH)

`queries.go:110-122` iterates `map[string][]*ArtifactInstance` without sorting.

The original assessment was correct about the root cause. However, `gui/guisquads/artifact_refresh.go:75-81` already calls `sort.Slice` on the returned slice before building the display list. That is the only current consumer of `GetAllInstances`. The UI list is therefore stable in practice.

The function itself is still wrong: it makes an implicit ordering guarantee it does not provide. If a second caller is added without remembering to sort, the bug reappears.

**Fix (10 min):** Sort by `defID + InstanceIndex` inside `GetAllInstances` itself, eliminate the sort at the call site in `artifact_refresh.go`. One canonical behavior instead of a caller-must-sort contract.

---

### 3. `ArtifactDispatcher` Has No Direct Tests — HIGH

The original report said "dispatcher.go has no tests." That is still true for the `ArtifactDispatcher` struct itself. `gear_test.go` does test individual behaviors through their hook methods directly (e.g., `TestDeadlockShackles_SkipsActivation` calls `b.OnPostReset` directly). But the dispatcher's own logic — deduplication in `DispatchPostReset`, the cross-faction pending-effects second loop, and the `RefreshRoundCharges`-before-fire ordering in `DispatchOnTurnEnd` — has no coverage.

These are the specific untested paths:

- `DispatchPostReset`: the `fired map[string]bool` deduplication when the same behavior is equipped on two different squads. If this is wrong, a behavior fires twice or not at all per reset.
- `DispatchPostReset`: the second loop that fires behaviors with pending effects not present in the current faction's equipment (the Deadlock Shackles cross-faction path). This is the subtlest path in the package.
- `DispatchOnTurnEnd`: `RefreshRoundCharges()` is called before behavior fires. If the order swaps, round-charge behaviors get a free use per turn. The current order is correct but nothing will catch a future regression.

**Fix (2-3 hrs):** Add `dispatcher_test.go`. `setupCombatContext` in `gear_test.go` is the model. Cover: same behavior on two squads fires exactly once, Deadlock Shackles pending effect fires on enemy faction reset, round charges are refreshed before turn-end behaviors fire.

---

### 4. ChargeType Not Queryable from the Interface — MEDIUM

`ArtifactBehavior` has no `ChargeType()` method. Behaviors encode their charge type only inside their `Activate` implementations:

```go
// TwinStrikeBehavior
ctx.ChargeTracker.UseCharge(BehaviorTwinStrike, ChargeOncePerBattle)

// EchoDrumsBehavior / ChainOfCommandBehavior
ctx.ChargeTracker.UseCharge(BehaviorEchoDrums, ChargeOncePerRound)
```

`artifact_panel.go:66` hardcodes the label:

```go
detail += "\n\n[Charge spent this battle]"
```

ChainOfCommand and EchoDrums are per-round. When their charge refreshes mid-combat, the panel still shows "Charge spent this battle" until the next `Refresh()`. The label is actively wrong for two of the six behaviors.

**Fix (45 min):** Add `ChargeType() ChargeType` to `ArtifactBehavior` with a `ChargeOncePerBattle` default in `BaseBehavior`. Override in EchoDrums and ChainOfCommand. Update `UpdateDetailPanel` to branch on the value.

---

### 5. `ToggleArtifactMode` is Dead Code — MEDIUM (downgraded from MEDIUM, now confirmed dead)

`artifact_handler.go:41` defines `ToggleArtifactMode`. A codebase-wide search finds zero callers outside the file where it is defined. The panel uses `ArtifactPanelController.Toggle()`, which calls `CancelArtifactMode()` and `Show()` directly.

`ToggleArtifactMode` is unreachable production code that modifies `BattleState.InArtifactMode` without touching any widget. Anyone who wires it by mistake (e.g., binding it to a keyboard shortcut) will see state get set with no panel appearing.

**Fix (5 min):** Delete `ToggleArtifactMode` from `artifact_handler.go`.

---

### 6. `ArtifactBalance` is Global Mutable State — MEDIUM

`balanceconfig.go` exports:

```go
var ArtifactBalance ArtifactBalanceConfig
```

`SaboteursHourglassBehavior.OnPostReset` reads `ArtifactBalance.SaboteursHourglass.MovementReduction` directly. `LoadArtifactBalanceConfig` and `validateArtifactBalance` have no test coverage. A broken JSON silently produces zero values at startup; `validateArtifactBalance` prints a warning to stdout that is easy to miss.

As artifact count grows, more fields accumulate in the global and the gap between what JSON says and what is tested widens.

**Fix (1 hr):** Write tests that call `validateArtifactBalance` directly with zero and negative inputs to confirm warnings fire. Longer term: pass balance values through `BehaviorContext` or as constructor arguments so tests do not need file I/O.

---

### 7. Duplicate `EquippedArtifacts` Iteration — MEDIUM

Three functions in `queries.go` independently walk `data.EquippedArtifacts` and call `templates.GetArtifactDefinition(id)`:

| Function | Purpose |
|---|---|
| `GetArtifactDefinitions` | collects all defs |
| `GetEquippedBehaviors` | resolves behavior from `def.Behavior` |
| `HasArtifactBehavior` | checks one behavior key |

`GetEquippedBehaviors` and `HasArtifactBehavior` could both be derived from a single `GetArtifactDefinitions` call. Every new query over equipped artifacts means a new raw loop. `EquipArtifact` in `system.go` has a fourth independent duplicate-check loop over the same slice.

**Fix (30 min):** Rewrite `GetEquippedBehaviors` and `HasArtifactBehavior` to call `GetArtifactDefinitions` internally. The raw loop lives in one place.

---

### 8. `ArtifactPanelController` Two-Step Construction — LOW

```go
ctrl := NewArtifactPanelController(deps)
// ... build widgets ...
ctrl.SetWidgets(list, detail, activateButton)  // must not forget this
```

If `SetWidgets` is not called, all widget interactions silently no-op (nil guards in `UpdateDetailPanel` and `Refresh` prevent panics but the panel never updates). This is a wiring trap on any new combat screen.

**Fix (20 min):** Pass widget references to the constructor, or use a builder that returns a fully initialized controller.

---

### 9. `ValidateBehaviorCoverage` Inner Loop is O(n×m) — LOW

`registry.go:80-93` scans all artifact definitions for every registered behavior. With 6 behaviors and ~20 artifacts this is immaterial. The fix is trivial when artifact count reaches ~30.

**Fix (10 min):** Build a `map[string]bool` of all referenced behavior keys from `ArtifactRegistry` in one pass before the outer loop.

---

### 10. `fmt.Printf` Bypasses Structured Logging — LOW

`system.go:186` and `balanceconfig.go` use `fmt.Printf` for warnings. The dispatcher uses `powercore.PowerLogger`. This inconsistency means some artifact warnings go to stdout while activation events go through the structured combat log.

**Fix (30 min):** Thread the logger into `ApplyArtifactStatEffects` or use `powercore.DefaultLogger` where one exists.

---

## Prioritized Roadmap

### Do these now (total ~3.5 hrs)

**1. Delete `ToggleArtifactMode`** — 5 min
It is confirmed dead code. Leaving it creates a wiring trap for future work.

**2. Fix silent error drop in `executeArtifact`** — 15 min
Every future artifact benefits immediately. The error is already returned from `ActivateArtifact`; the caller just needs to use it.

**3. Fix `GetAllInstances` ordering** — 10 min
Move the sort inside `GetAllInstances`, remove the duplicate sort at the `artifact_refresh.go` call site. Correct behavior guaranteed by default rather than by caller discipline.

**4. Add `dispatcher_test.go`** — 2-3 hrs
The dispatcher's deduplication and cross-faction pending dispatch are the highest-risk untested paths in the package. Use `setupCombatContext` from `gear_test.go` as the model. Cover: deduplication when the same behavior is on two squads, cross-faction pending dispatch (Deadlock Shackles targeting enemy faction), round-charge refresh happens before behavior fires.

---

### Before adding artifact #7

**5. Add `ChargeType()` to the interface** — 45 min
Two existing behaviors (EchoDrums, ChainOfCommand) already show the wrong label in the panel. Fix this before adding another per-round behavior.

**6. Consolidate `EquippedArtifacts` iteration** — 30 min
Fewer places to update when the loop logic changes.

---

### Before balance tuning pass

**7. Balance config testability** — 1 hr
Write tests for `validateArtifactBalance` with zero and negative inputs. Catches broken JSON accepted silently at startup.

---

### Low priority (stable code — defer until nearby)

- `ValidateBehaviorCoverage` O(n×m): fix when artifact count exceeds ~30
- Two-step panel construction: fix if a second artifact panel screen is added
- `fmt.Printf` vs structured logging: fix during a logging pass across the powers layer

---

## Prevention

Two habits that prevent the highest-severity items from recurring:

1. **Never ignore an `error` return at a UI boundary.** At minimum: `if err != nil { log.Println("[ARTIFACT]", err) }`. The silent drop in `executeArtifact` exists because the error return was added to `ActivateArtifact` after the GUI was written.

2. **Every new behavior gets one test that fires it through `ArtifactDispatcher.DispatchPostReset` or `DispatchOnAttackComplete`, not just through the behavior's hook method directly.** The behaviors are tested in `gear_test.go`, but the dispatcher path — including deduplication — is what runs in production.

---

## Summary Table

| # | Item | File | Effort | Severity | Status |
|---|---|---|---|---|---|
| 1 | Silent error drop in `executeArtifact` | `artifact_handler.go:186` | 15 min | High | Open |
| 2 | `GetAllInstances` non-deterministic order | `queries.go:110` | 10 min | Medium | Partially mitigated by caller sort |
| 3 | Dispatcher has no direct tests | `dispatcher.go` | 2-3 hrs | High | Open |
| 4 | `ChargeType` not on interface | `behavior.go` | 45 min | Medium | Open |
| 5 | `ToggleArtifactMode` is dead code | `artifact_handler.go:41` | 5 min | Medium | Confirmed dead, delete it |
| 6 | Balance config untestable/global | `balanceconfig.go` | 1 hr | Medium | Open |
| 7 | Repeated equipped-artifact iteration | `queries.go`, `system.go` | 30 min | Medium | Open |
| 8 | Two-step panel construction | `artifact_panel.go` | 20 min | Low | Open |
| 9 | `ValidateBehaviorCoverage` O(n×m) | `registry.go:80` | 10 min | Low | Open |
| 10 | `fmt.Printf` vs structured logging | `system.go:186`, `balanceconfig.go` | 30 min | Low | Open |
