# Technical Debt Analysis: `tactical/powers/artifacts/`, `tactical/powers/perks/`, `tactical/powers/powercore/`

**Date:** 2026-06-11
**Scope:** 30 files across three packages — the artifact behavior/inventory system (13 source + 5 test files), the perk behavior/hook system (14 source + 1 test file), and the shared power foundation (3 source files, no tests).
**Test coverage:** artifacts **68.9%**, perks **50.5%**, powercore **0.0%**.

Overall verdict up front: these are among the *healthier* packages in the codebase — pure-data components, EntityID-only references, registry + JSON definition split, documented invariants, and real tests. The debt here is mostly **consistency drift between the two sibling packages**, **silent-failure paths**, and **shotgun surgery when adding a new perk**. There is one latent correctness bug (A3) and one release-build failure mode (K1) that deserve attention before more content is added.

---

## Files in Scope

| Package | Source files | Tests |
|---|---|---|
| `artifacts` | `behavior.go`, `behaviors.go`, `dispatcher.go`, `registry.go`, `context.go`, `artifactcharges.go`, `pending_effects.go`, `artifactinventory.go`, `system.go`, `queries.go`, `components.go`, `balanceconfig.go`, `init.go` | `gear_test.go`, `artifactbehavior_test.go`, `artifactcharges_test.go`, `artifactinventory_test.go`, `dispatcher_test.go` |
| `perks` | `hooks.go`, `dispatcher.go`, `system.go`, `registry.go`, `perkids.go`, `components.go`, `queries.go`, `unithelpers.go`, `format.go`, `balanceconfig.go`, `behaviors_stateless.go`, `behaviors_per_round.go`, `behaviors_per_battle.go`, `init.go` | `perks_test.go` |
| `powercore` | `context.go`, `logger.go`, `pipeline.go` | *(none)* |

---

## 1. Debt Inventory

### Correctness / Robustness Risks

#### K1. `LoadPerkBalanceConfig` fails soft in release builds — `perks/balanceconfig.go:115-136` ✅ RESOLVED 2026-06-12

> **Resolution:** `LoadPerkBalanceConfig` now returns `error` and `bootstrap.go` does `log.Fatalf`, matching the artifact loader. The DEBUG-only panic and WARNING-and-continue paths are gone.

If `perkbalanceconfig.json` is missing or unparseable, the function logs a WARNING and returns. `PerkBalance` stays zero-valued, so in a **release build** every perk silently misbehaves: `IsolatedPredator.DamageMult = 0` zeroes outgoing damage, `FieldMedic.HealDivisor = 0` would divide by zero in `TurnStart` (`behaviors_stateless.go:135`), `Fortify.MaxStationaryTurns = 0` never arms. In DEBUG it `panic`s instead — the opposite extreme, and against the project convention that loaders return errors and `gamesetup` decides fatality.

Contrast with its sibling: `artifacts.LoadArtifactBalanceConfig` returns `error` and `bootstrap.go:42-44` does `log.Fatalf` on failure. The two packages were written to the same design and have already drifted.

#### A3. Pending-effect consumption ignores the target's faction — `artifacts/behaviors.go:81-114`, `artifacts/dispatcher.go:54-79`

`pending_effects.go`'s package doc promises Phase 2 happens "in the next DispatchPostReset **for the target's faction**." The implementation doesn't check faction: `DispatchPostReset` fires `OnPostReset` for *any* behavior with pending effects, and `applyPendingEffects` **drains the entire queue via `Consume`** before filtering targets against the squads being reset. A targeted effect whose squad is *not* in `squadIDs` is consumed and silently dropped.

This is safe today only because of an unwritten invariant: with exactly two alternating factions, the post-reset that follows a player activation is always the enemy faction's. The moment a third faction exists (or reset ordering changes), Deadlock Shackles consumes its charge and does nothing — with no log and no test to catch it. Either re-queue non-matching effects instead of dropping them, or assert/document the two-faction invariant where it lives.

#### K2. `dispatcher.run` silently disables *stateless* perks when round state is missing — `perks/dispatcher.go:46-56`

`run()` returns early if `GetRoundState` is nil. That's correct for stateful perks, but stateless ones (Brace for Impact, Riposte, Cleave, Precision Strike, Vigilance…) need no round state at all, yet are also skipped. `PerkRoundState` is attached only by `InitializePerkRoundStatesForFaction` at combat init for squads that had perks *at that moment*. Any gap — a squad gaining perks mid-combat, an init path that forgets the call, reinforcements — silently turns off **all** perks for that squad with zero log output. A "perk equipped but round state missing" condition should at minimum log loudly.

#### A1. Charge type declared twice and can disagree — `artifacts/behavior.go:56`, `artifacts/behaviors.go:190,274,309`

Every behavior declares `ChargeType()` on the interface (Chain of Command and Echo Drums override it to `ChargeOncePerRound`), but the actual `UseCharge` call sites pass the charge type **again as a literal**: `ctx.ChargeTracker.UseCharge(BehaviorTwinStrike, ChargeOncePerBattle)`. The helpers `activateWithPending`/`activateBroadcastPending` likewise take a `chargeType` parameter the behavior already knows. Changing a behavior's `ChargeType()` method does nothing unless the call-site literal is also changed — and `CanActivateArtifact`/GUI availability reads would then disagree with consumption. The literal should be `b.ChargeType()` (or the helpers should accept the behavior, not the raw values).

#### A2. GUI activation path bypasses the dispatcher's logger — `gui/guiartifacts/artifact_handler.go:150-163`

`executeArtifact` hand-builds a `BehaviorContext` with `logger = nil` and `round = 0`. Every `ctx.Log(..., "activated")` call inside `Activate` methods is a nil-safe no-op, so **player-activated artifacts never appear in the combat log** — only the dispatcher-driven hooks (OnPostReset etc.) log. Activation *errors* are reported via a bare `fmt.Printf("[ARTIFACT] ...")`, sidestepping the `classifyLogPrefix` routing in `combat_power_dispatch.go`. The dispatcher already owns manager, cache, charge tracker, and logger; exposing an `Activate(behaviorKey, target)` method on `ArtifactDispatcher` (or `PowerOrchestrator`) removes the duplicate context construction and fixes the lost logs in one move.

### Code Debt — Duplication / Shotgun Surgery

#### K3. Adding one perk touches six places — `perks/` package-wide ✅ RESOLVED 2026-06-12 (validator portion)

> **Resolution:** `validatePerkBalance` (and `validateArtifactBalance`) replaced by `powercore.ValidateBalanceRanges` — balance fields declare their range via `balance:"fraction|mult|count|bonus"` struct tags, and untagged numeric fields fail loading, so the hand-written validator step is gone (recipe is now 5 touches). See `powercore/balancevalidate.go` and PERK_SYSTEM.md §7 step 6.

The current recipe for a new stateful perk:

1. Constant in `perkids.go`
2. Behavior struct + `RegisterPerkBehavior` in an `init()` in one of three `behaviors_*.go` files
3. State struct in `components.go` (if stateful)
4. Balance struct + field on `PerkBalanceConfig` in `balanceconfig.go`
5. A hand-written range-check block in `validatePerkBalance` (already **80 lines** of copy-paste for 21 perks; `complexity_report.txt` flags `LoadPerkDefinitions` at complexity 26)
6. Entries in `perkdata.json` **and** `perkbalanceconfig.json`

Steps 4–5 are the worst offenders: `validatePerkBalance` repeats the same three range patterns (`(0,1)` fraction, `[0.1,10]` multiplier, `(0,100]` count) twenty-seven times. A declarative table — `{field *float64, kind rangeKind, name string}` — or struct tags would collapse it to one loop and make step 5 disappear. The artifact package will grow the same problem (`validateArtifactBalance` is one field today).

#### K4. `EquipPerk` takes a `maxSlots` parameter nobody varies — `perks/system.go:17`

The only production caller (`gui/guisquads/squadeditor_perks.go:165`) passes the package's own `perks.MaxPerkSlots` back to it. Two sources of truth for one cap; a future caller passing a different number would silently create squads with more slots than the UI displays. Drop the parameter (or make the constant the documented default and delete one of the two).

#### C1. Twin registry/validation patterns drifting apart — `artifacts/registry.go` vs `perks/registry.go` + `hooks.go`

Both packages implement the same shape — string-keyed behavior registry, JSON definition registry, startup cross-check (`ValidateBehaviorCoverage` / `ValidateHookCoverage`) — with different naming, different error handling (see K1), and different load APIs (`error` vs `[]error` vs `void`). `powercore` was created exactly to host shared mechanics and could own a generic `Registry[K, V]` + coverage check. Low urgency, but every divergence (K1 being the proof) starts here.

### Architecture Debt

#### P1. `PowerContext.RoundNumber` is written but never read — `powercore/context.go:28`

No behavior in either package reads `ctx.RoundNumber`. Worse, the values written are mostly wrong: `ArtifactDispatcher` passes `0` for PostReset and AttackComplete (`dispatcher.go:55,83`), the GUI passes `0`, and `perks.SquadPerkDispatcher.run` leaves it zero — only `DispatchOnTurnEnd` and `RunTurnStartHooks` carry the real round. The field is a trap: the first behavior that reads it will get `0` in most hooks and the bug will be invisible. Either thread the round through every dispatch path or delete the field until a consumer exists.

Same story in miniature for `Cache`: perks' `run()` builds `PowerContext{Manager, Logger}` without `Cache` (`perks/dispatcher.go:51`), so the "shared fields live in one place" promise of powercore is only honored by artifacts. A perk hook that reaches for `ctx.Cache` will nil-panic.

#### A5. Equipped state stored in two places with manual rollback — `artifacts/system.go:83-177`

"What is equipped where" lives both in `EquipmentData.EquippedArtifacts` (on the squad) and `ArtifactInstance.EquippedOn` (in the player inventory). `EquipArtifact`/`UnequipArtifact` keep them in sync with append-then-rollback sequences (`system.go:134-137`, `166-173`). It works and is tested, but every new mutation path must re-implement the two-phase dance. Deriving one side from the other (inventory as the single source; `EquippedArtifacts` as a query or cache) would remove the rollback code entirely.

#### C2. Terminology: "turn" events trigger "round" resets — `power_orchestrator.go:107-112`, `perks/system.go:113-131`

`OnTurnEnd` → `DispatchRoundEnd` → `ResetPerkRoundStateRound`; meanwhile `DispatchTurnStart` is fired from the **PostReset** event. The per-turn/per-round/per-battle vocabulary is load-bearing (it names the three behavior files and two state maps), so the mismatch between event names and reset names costs real comprehension time for anyone tracing perk state lifecycle. A naming pass (or a comment block in `power_orchestrator.go` mapping event → reset) is cheap.

#### C3. `combatstate.SetGearLogger` package-global undercuts the powercore seam — `power_orchestrator.go:61-63`

powercore's `PowerLogger` was introduced to replace package-global logger callbacks, but `InstallLogger` still mutates a global in `combatstate` to route gear messages. One stray global keeps the old pattern alive as a template for the next person.

### Naming / Clarity

#### K5. `HookContext` field validity depends on the hook — `perks/hooks.go:18-29`

Eight identity fields where any subset may be zero depending on hook type, documented only in a comment. Call sites compensate by setting overlapping fields to the same value (`CounterMod` sets `DefenderSquadID` *and* `SquadID`; `DamageRedirect` sets five fields, with `DefenderID == UnitID`). Works, tested, but every new hook re-derives which fields to fill by reading other dispatch methods. Small constructor helpers per hook family (the existing `combatCtx` is the right idea, extended) would encode the contract.

#### K6. `GuardianProtocol.RedirectFraction` is a divisor, not a fraction — `perks/balanceconfig.go:65`, `behaviors_stateless.go:245`

`damageAmount / RedirectFraction` with a value of 4 means "redirect 25%". A balance tuner reading the JSON will type `0.25` and get a zero-division-adjacent surprise (int field, so `0.25` → unmarshal error at best, `0` at worst — which `validatePerkBalance` does catch). Rename to `RedirectDivisor` or store an actual fraction.

#### A6. `gear_test.go` is named for a package that no longer exists

The artifacts package's main behavior test file carries the pre-rename `gear` name. (Related: `CLAUDE.md`'s "Reference Implementations" section still points at `gear/Inventory.go` and `systems/positionsystem.go`, both stale paths.)

### Documentation Debt

#### D1. Stale cross-references

- `perks/components.go:26-28` lifecycle table cites `combat_power_dispatch.go` as the writer of `MovedThisTurn`/`AttackedThisTurn`; the wiring moved to `power_orchestrator.go` + `perks/dispatcher.go`.
- `POWER_LAYERS.md:51` claims `ValidateHookCoverage` is "defined but currently uncalled — see `tech_debt_perks.md` D3". It *is* called (`bootstrap.go:46`) and `tech_debt_perks.md` doesn't exist.
- `ENTITY_REFERENCE.md:314` says perk "slot cap depends on squad progression"; `PROGRESSION.md:490` correctly says it's the constant 3. One of them is wrong.

### Testing Debt

#### T1. powercore: 0% coverage

Three small files, but `PowerPipeline` ordering ("artifacts before perks before GUI") is the package's entire reason to exist and has no test asserting registration-order invocation. The nil-safe `ctx.Log` and `NewBehaviorContext(nil, ...)` paths are likewise untested at their source (only indirectly via artifacts tests).

#### T2. perks at 50.5%, all in one file

`perks_test.go` is the only test file for 14 source files. Untested or thinly tested by inspection: `LoadPerkDefinitions` validation branches (duplicate IDs, invalid roles, asymmetric exclusivity), `validatePerkBalance`, `format.go`, and the dispatcher early-exit behavior from K2. The artifacts package shows the better pattern: five focused test files by concern.

#### T3. No test pins the A3 two-faction invariant

`dispatcher_test.go` covers pending-effect happy paths; nothing asserts what happens when a post-reset fires for a faction that doesn't contain the pending target. That's exactly the case that will regress silently.

---

## 2. Impact Assessment

| ID | Debt | Impact | Risk |
|----|------|--------|------|
| K1 | Perk balance fails soft in release | All perks no-op/zero-damage on a bad deploy; division by zero in FieldMedic | **High** |
| A3 | Pending effects drained on wrong faction's reset | Latent bug; activates the moment multi-faction combat exists | **High (latent)** |
| K2 | Stateless perks need round state | Whole-squad perk blackout with zero diagnostics on any init gap | Medium-High |
| A1 | Charge type literal duplication | One-line mismatch desyncs UI availability from actual consumption | Medium |
| A2 | GUI bypasses logger | Player-facing: activations missing from combat log today | Medium (live, cosmetic) |
| K3 | 6-touch perk addition, 80-line validator | ~30-60 min friction + review noise per perk; scales with content plans | **High (velocity)** |
| P1 | RoundNumber written, never read, mostly wrong | Trap for the first consumer; invisible bug | Medium |
| A5 | Dual equipped-state + rollbacks | Each new mutation path re-implements two-phase sync | Medium |
| K4/K6/A6/D1 | Param/naming/doc staleness | Onboarding confusion, wrong-doc decisions | Low |
| T1-T3 | Coverage gaps | Regressions in ordering/validation surface only in playtests | Medium |

---

## 3. Prioritized Remediation Plan

### Quick Wins (hours, low risk)

1. ~~**Make `LoadPerkBalanceConfig` return `error`** and have `bootstrap.go` treat it exactly like the artifact loader (`log.Fatalf`). Delete the DEBUG-only panic. (K1)~~ ✅ Done 2026-06-12
2. **Replace charge-type literals with `b.ChargeType()`** in `behaviors.go` activations and the two `activate*Pending` helpers (have them take the behavior or call `ChargeType()` internally). (A1)
3. **Add `Activate` to `ArtifactDispatcher`** (it already has manager/cache/tracker/logger) and route `guiartifacts.executeArtifact` through it. Fixes the silent combat-log gap and deletes the hand-rolled context. (A2)
4. **Drop the `maxSlots` parameter from `EquipPerk`**; use `MaxPerkSlots` internally. (K4)
5. **Delete or correctly populate `PowerContext.RoundNumber`.** If kept, pass the real round from `TurnManager` through `DispatchPostReset`/`DispatchOnAttackComplete` and perks' `run`. (P1)
6. **Log loudly in `perks.run`** when a squad has equipped perks but no `PerkRoundState`. (K2 mitigation)
7. **Fix stale comments/docs**: `components.go` lifecycle table, `POWER_LAYERS.md:51`, `ENTITY_REFERENCE.md:314`; rename `gear_test.go` → `behaviors_test.go`. (D1, A6)
8. **Rename `RedirectFraction` → `RedirectDivisor`** (JSON key too — one-time data edit). (K6)

### Medium-Term (1–2 weeks)

9. ~~**Table-driven `validatePerkBalance`**: declare per-field `{ptr, kind}` entries; one loop validates all. Apply the same table to artifacts before it grows. Cuts the per-perk recipe from 6 touches to 5 and removes the worst copy-paste. (K3)~~ ✅ Done 2026-06-12 via `balance` struct tags + `powercore.ValidateBalanceRanges` (with tests)
10. **Resolve A3 explicitly**: either (a) make `applyPendingEffects` re-queue effects whose target is not in `squadIDs`, or (b) add a faction check in `DispatchPostReset`'s pending loop. Add the missing test for "post-reset fires for non-target faction" either way. (A3, T3)
11. **Split `run` requirements by hook statefulness**: stateless hooks dispatch without round state; only stateful ones early-exit. Alternatively attach `PerkRoundState` lazily in `run`. (K2)
12. **powercore tests**: pipeline registration-order invariant, nil-logger no-op, `NewBehaviorContext(nil, …)`. Small file, closes the 0%. (T1)
13. **Split `perks_test.go`** by concern (registry/loading, dispatcher, behaviors per taxonomy file) and add `LoadPerkDefinitions` validation-branch tests. (T2)

### Long-Term (as content scales)

14. **Single source of truth for equipped artifacts**: make `ArtifactInstance.EquippedOn` authoritative and derive `EquipmentData.EquippedArtifacts` (or vice versa), deleting the rollback choreography in `EquipArtifact`/`UnequipArtifact`. (A5)
15. **Generic registry + coverage validation in powercore** consumed by both packages, unifying `ValidateBehaviorCoverage`/`ValidateHookCoverage` naming and error contracts. (C1)
16. **Terminology pass on turn/round naming** across `power_orchestrator.go` and `perks/system.go` reset functions, or a single authoritative comment mapping events → resets. (C2)
17. **Retire `combatstate.SetGearLogger`** by passing the `PowerLogger` to whatever emits gear messages. (C3)

---

## 4. Prevention

- **Loader convention check**: every `Load*Config` in `tactical/powers/` returns `error`; `gamesetup` is the only fatality point. (Same rule the templates package already enforces.)
- **New-perk checklist** in `PERK_SYSTEM.md`: the 5 touch points, in order, with the validator table as step 4 — keeps step count honest until K3 lands.
- **Test gate**: a new behavior (perk or artifact) without a corresponding test case fails review; both packages already have the harness, so the marginal cost is minutes.
- **No new fields on shared contexts without a reader**: `RoundNumber` shows how a speculative field rots; powercore changes should name their first consumer.

---

## Summary

The three packages are structurally sound — the worst debts are *seams between them*. Fix the two asymmetric failure modes first (**K1**: soft-fail perk balance in release; **A3**: pending effects drained on the wrong faction's reset) since both fail silently. The biggest velocity drag is **K3** (six-touch perk addition with an 80-line copy-paste validator), which matters in direct proportion to how many more perks are planned. **A2** is the only player-visible defect today: artifact activations never reach the combat log. Quick wins 1–8 are mechanical and low-risk; items 9–13 pay for themselves within a few perks' worth of content work.
