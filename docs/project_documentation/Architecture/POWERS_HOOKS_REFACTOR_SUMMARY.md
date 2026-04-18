# Powers Hooks Refactor — Session Summary

**Last Updated:** 2026-04-17
**Purpose:** Single entry point for reapproaching the artifacts/perks hooks refactor. Captures origin, work landed, design decisions and their rationale, and what was deliberately left undone. Read this before touching `tactical/powers/artifacts/`, `tactical/powers/perks/`, or `tactical/powers/powercore/`.

## Companion docs

- `POWERS_HOOKS_REVIEW.md` — **historical.** Original review that drove Phase 1 + Phase 2. Keep for context, don't re-execute.
- `POWERS_HOOKS_FOLLOWUP.md` — **historical.** Post-phase-2 audit; items #1 and #2 are done, #3 was skipped. Keep for context.
- `HOOKS_AND_CALLBACKS.md` — **stale in places** (see "Documentation drift" below). Useful for understanding adjacent systems (TurnManager callbacks, encounter callbacks, etc.) but its perk/artifact sections reference removed APIs.
- `docs/project_documentation/Systems/PERK_SYSTEM.md`, `ARTIFACT_SYSTEM.md` — **stale in places** (same issue).

---

## 1. Origin

The user asked for a review of the artifacts and perks packages: "make sure that the hooks and callbacks are cleanly designed, maintainable, have single sources of truth when applicable, and follow good software engineering and game development practices."

Starting state had two parallel hook systems (one for artifacts, one for perks) that had been built independently. They shared no types, duplicated logger globals, and the combined ordering policy was hardcoded across four near-duplicate dispatch methods in `CombatService`. The two systems were each internally coherent but their combined surface was harder to reason about than it needed to be.

## 2. Timeline

Broad strokes of what happened in this session, in order:

1. **Review** (produced `POWERS_HOOKS_REVIEW.md`). Catalogued the two systems, identified 4 high-leverage findings (H1–H4), 4 medium findings (M1–M4), 3 lower findings (L1–L3). Recommended a 2-phase approach.

2. **Phase 1 — Shared foundation.** Created `tactical/powers/powercore/`: `PowerContext`, `PowerLogger`, `PowerPipeline`. Both package contexts now embed `PowerContext` by value. Replaced two package-global loggers with a single injected `PowerLogger`. Replaced four hand-written `Fire*` dispatchers in `CombatService` with a declarative `PowerPipeline` wired once in `NewCombatService`.

3. **Phase 2 — Encapsulation + safety.**
   - H4: charge tracker required at `NewArtifactDispatcher`; `Reset()` in-place on battle start.
   - M1: added helpers for shared-tracking reads/writes (most later removed; `IncrementTurnsStationary` kept).
   - M2: extracted `PendingEffectQueue` with a package-doc explanation of the two-phase pattern.
   - M3: both `ValidateBehaviorCoverage` and `ValidateHookCoverage` return `[]error`; bootstrap fail-fast in debug builds.

4. **First wrapper cleanup.** User noticed several Phase 2 additions were pure forwarders. Removed six pure getters on `HookContext` (`MovedThisTurn`, `TurnsStationary`, `WasAttackedLastTurn`, `DidNotAttackLastTurn`, `WasIdleLastTurn`, `ResetTurnsStationary`) plus an unused `dispatcher.ChargeTracker()` accessor. Kept `IncrementTurnsStationary` (encapsulates monotonic-cap invariant) and `LogPerk` (typed→string conversion).

5. **Second wrapper cleanup.** Removed four `CombatService.Fire*` methods — one-line forwarders to `cs.powerPipeline.Fire*`. Wiring passes pipeline method values directly. Also removed `NewPowerPipeline()` constructor; `&PowerPipeline{}` inline.

6. **Follow-up audit** (produced `POWERS_HOOKS_FOLLOWUP.md`). Identified three "worth doing" items, all in `perks/dispatcher.go`. User approved #1 and #2.

7. **Follow-up items #1 + #2.** Collapsed four near-identical damage-pipeline hook bodies into a shared `d.run` primitive. Eliminated post-hoc `ctx.X = y` mutation in `TargetOverride`, `CounterMod`, `DeathOverride`, `DamageRedirect` by having callers declare all fields up front.

8. **Third wrapper cleanup.** User caught that I had introduced `ctxFields` — a struct that duplicated `HookContext`'s field list. Removed `ctxFields` and `buildContext` entirely. `d.run` now takes a `HookContext` template directly; every caller uses the single type. Tests migrated to inline `&HookContext{...}` construction.

## 3. What landed (current code state)

### `tactical/powers/powercore/` (new package)

- `PowerContext` — shared base struct: `Manager`, `Cache`, `RoundNumber`, `Logger`. Embedded by value in both `artifacts.BehaviorContext` and `perks.HookContext`.
- `PowerLogger` interface + `LoggerFunc` adapter. Nil-safe `ctx.Log(source, squadID, message)` via method on `*PowerContext`.
- `PowerPipeline` with typed subscriber lists for four lifecycle events (`PostReset`, `AttackComplete`, `TurnEnd`, `MoveComplete`). `On*` to register, `Fire*` to invoke.

### `tactical/powers/artifacts/`

- `BehaviorContext` embeds `powercore.PowerContext` by value. Only artifact-specific field: `ChargeTracker *ArtifactChargeTracker`.
- `ArtifactDispatcher` requires a non-nil `ArtifactChargeTracker` at construction (panics on nil). `Reset()` the tracker in place to start a new battle — do not swap the tracker out; that would invalidate pipeline subscriber bindings.
- `PendingEffectQueue` (in `pending_effects.go`) encapsulates the two-phase deferred-effect pattern (queue on Activate → consume on next OnPostReset). Package doc explains the phases. `ArtifactChargeTracker` delegates pending-effect methods to the queue.
- `ValidateBehaviorCoverage() []error` — returns errors instead of printing.
- `IsRegisteredBehavior(key)` — exposed so the shared logger can route `[GEAR]` vs `[PERK]` prefixes.
- Behavior activations log via `ctx.Log(BehaviorKey, squadID, msg)`. No package-global logger.
- `BehaviorContext` still has thin forwarders: `GetActionState`, `GetSquadFaction`, `GetFactionSquads`, `GetSquadSpeed`. **These are pre-existing; user deliberately left them alone.** `SetSquadLocked` and `ResetSquadActions` earn their keep (three-field atomic writes).

### `tactical/powers/perks/`

- `HookContext` embeds `powercore.PowerContext` by value. Perk-specific fields: attacker/defender/squad IDs, `SquadID`, `UnitID`, `DamageAmount`, `RoundState *PerkRoundState`.
- `HookContext` helpers: `LogPerk(perkID, squadID, msg)` (typed PerkID→string), `IncrementTurnsStationary(max)` (monotonic cap invariant). That's it — no pure getters.
- `SquadPerkDispatcher` has a `logger` field set via `SetLogger(PowerLogger)`. All nine `PerkDispatcher` interface methods delegate to one primitive:
  ```go
  func (d *SquadPerkDispatcher) run(ownerSquadID ecs.EntityID, manager *common.EntityManager, ctx HookContext, hook func(*HookContext, PerkBehavior) bool)
  ```
  Callers populate the `HookContext` fields they need; `run` overwrites `PowerContext` and `RoundState` (the dispatcher-owned fields) and iterates equipped perks. Returning `false` from hook terminates iteration early.
- **No `buildHookContext`, `buildCombatContext`, `ctxFields`, or `buildContext` free functions.** All context construction is either inline in `run` or inline in test files via `&HookContext{...}` struct literals.
- `ValidateHookCoverage() []error` — matches the artifact pattern.

### `tactical/combat/combatservices/`

- `NewCombatService` constructs a long-lived `ArtifactChargeTracker`, passes it to `NewArtifactDispatcher`, registers pipeline subscribers once in declared order (artifact → perk → GUI), and passes pipeline `Fire*` method values directly to the subsystem hooks. No `CombatService.Fire*` methods.
- `InitializeCombat` calls `cs.chargeTracker.Reset()` instead of swapping trackers.
- `setupPowerDispatch` constructs a single `powercore.LoggerFunc`, calls `SetLogger` on both dispatchers, and attaches the perk dispatcher to `CombatActSystem`. Routing decides `[GEAR]`/`[PERK]` prefix via `artifacts.IsRegisteredBehavior(source)`.

### `setup/gamesetup/bootstrap.go`

- `LoadGameData` calls `reportCoverage("artifact", artifacts.ValidateBehaviorCoverage())` and `reportCoverage("perk", perks.ValidateHookCoverage())`. `reportCoverage` fails fatally in `config.DEBUG_MODE`, warns otherwise.

## 4. Key design decisions + rationale

**Embed `PowerContext` by value, not by pointer.**
Value embedding means zero-value contexts (bare `&HookContext{}` in tests) don't nil-panic when code accesses promoted fields like `ctx.Manager`. Copying a `PowerContext` is four pointer-sized fields — trivial.

**Keep artifacts and perks as separate packages with separate behavior interfaces.**
Artifacts have 3 event hooks (`OnPostReset`, `OnAttackComplete`, `OnTurnEnd`) + `Activate`. Perks have 10 damage-pipeline hooks. Unifying them would force both shapes through a lowest-common-denominator interface and harm both. The review (H1–H3) only extracts the *infrastructure* — context, logger, pipeline — not the hook surfaces.

**One `HookContext` type, not separate "input" and "output" structs.**
The third wrapper cleanup: `ctxFields` was a duplicate of `HookContext`'s field list. The dispatcher's `run` now takes a `HookContext` template and overwrites `PowerContext`/`RoundState` (dispatcher-owned). One type, one field list, same caller ergonomics.

**Charge tracker is long-lived, reset in place.**
The alternative was to reconstruct `ArtifactDispatcher` on battle start. That would break the `PowerPipeline` subscriber bindings (which captured method values on the old dispatcher). `tracker.Reset()` is cheaper and safer.

**Pipeline subscriber order declared once, in `NewCombatService`.**
Previously each of four `Fire*` methods enumerated the order by hand. Adding a new power system meant edits in four places plus hoping the comment at `combat_service.go:86-88` stayed accurate. Now subscribers are appended in order at construction, the ordering comment lives where the order is declared, and `Fire*` methods are one-liners (now inlined entirely).

**`IncrementTurnsStationary` is the only "setter-with-logic" helper on `HookContext`.**
Pure getters (`ctx.MovedThisTurn()` etc.) didn't pay for themselves — callers can write `ctx.RoundState.MovedThisTurn` and the nil check is paper-over defence against impossible states. Helpers only earn their place when they encapsulate an invariant (monotonic cap) or a conversion (typed ID → string).

**`d.run` takes a template `HookContext` value, not a `*HookContext`.**
Value parameters give callers the clean `HookContext{...}` literal syntax without a stray `&`. The function copies once (four-ish pointer-sized fields), then takes `&ctx` internally for the closure. `PowerContext` and `RoundState` get overwritten inside `run`, so caller-provided values in those fields are discarded — documented at the method.

## 5. Explicitly not done (+ why)

**L1 — Registration file splits differ between packages.**
Artifacts: `artifactbehaviors_activated.go` / `_passive.go`. Perks: `behaviors_stateless.go` / `_stateful_round.go` / `_stateful_battle.go`. Both groupings make sense locally; unifying would churn without benefit.

**L2 — Perk iteration order depends on `PerkSlotData.PerkIDs` slice order.**
Only matters if two equipped perks on the same squad conflict. No current conflict exists. File as a TODO if one arises.

**L3 — `AllBehaviors()` rebuilds a sorted slice on every `DispatchOnTurnEnd`.**
Minor allocation. Cache at registration time if profiling flags it.

**Follow-up #3 — Rename `DispatchAttackTracking` / `DispatchMoveTracking` / `DispatchRoundEnd`.**
These are state-only methods, not dispatchers. Names are slightly misleading. Left alone because the rename touches pipeline wiring in `combat_service.go` and the benefit is cosmetic. Bundle with a larger edit in the area if one comes up.

**Generic `powercore.Registry[K,V]`.**
Both packages have `map[K]V` registries with `Register`/`Get`. A generic would save ~15 lines but replace two concrete types with parametric indirection for trivial maps holding ~6 artifacts and ~21 perks. Locality > abstraction.

**Generic coverage validator / balance loader.**
Structurally similar signatures, but the bodies reach into package-specific registries or validate package-specific fields. A generic would need callback injection — more surface, not less.

**Unifying `ArtifactBehavior` and `PerkBehavior` interfaces.**
Addressed above. Not happening.

**Unifying equipment components.**
`EquipmentData.EquippedArtifacts` tracks multi-instance `ArtifactInstance` with `EquippedOn` across an inventory. `PerkSlotData.PerkIDs` is a simple `[]PerkID`. Different data requirements.

**Pre-existing `BehaviorContext.Get*` helpers.**
`GetActionState`, `GetSquadFaction`, `GetFactionSquads`, `GetSquadSpeed` are thin forwarders. They predate this refactor, they're used 13 times in artifact behaviors, and removing them inflates each call site by one `ctx.Manager` / `ctx.Cache` argument. User explicitly said "leave alone unless a separate pass." They remain.

## 6. Documentation drift (real, outstanding)

Four project docs reference symbols that no longer exist. Nothing is broken — the docs are stale, not wrong in principle.

- `docs/project_documentation/Architecture/HOOKS_AND_CALLBACKS.md` — mentions `buildHookContext`, `SetArtifactLogger`, `ArtifactLogger` type.
- `docs/project_documentation/Systems/PERK_SYSTEM.md` — mentions `buildCombatContext`, `SetPerkLogger`, `logPerkActivation`.
- `docs/project_documentation/Systems/ARTIFACT_SYSTEM.md` — mentions `logArtifactActivation`, `SetArtifactLogger`.
- `docs/project_documentation/Architecture/POWERS_HOOKS_FOLLOWUP.md` — "Worth doing" section describes items #1 and #2 as pending; they're done.

Fix is a doc-only pass. Not done in this session because the user wasn't specifically asking for it and the code work kept producing follow-up findings.

## 7. Guidance for future sessions

**If you're adding a new artifact behavior:**
- Implement `ArtifactBehavior`; embed `BaseBehavior` for no-op defaults.
- Register in an `init()` in one of the behavior files.
- Declare the behavior key string constant in `artifactbehavior.go` alongside the others.
- Add the artifact definition JSON (`assets/gamedata/artifactdata.json`) with a matching `behavior` field. `ValidateBehaviorCoverage` will complain at startup if the key doesn't match.
- Use `ctx.Log(BehaviorKey, squadID, msg)` for logging. Don't `fmt.Println` or `fmt.Printf`.

**If you're adding a new perk:**
- Implement `PerkBehavior`; embed `BasePerkBehavior` for no-op defaults.
- Declare the `PerkID` constant in `perkids.go`.
- Register in an `init()` in the appropriate `behaviors_stateless.go` / `behaviors_stateful_round.go` / `behaviors_stateful_battle.go` file (pick by state scope).
- Add JSON definition in `assets/gamedata/perkdata.json`. `ValidateHookCoverage` will complain if missing.
- Per-perk state: define a struct, use `GetPerkState[T]` / `SetPerkState` helpers on `ctx.RoundState`. Shared tracking fields (`MovedThisTurn`, etc.) — access through `ctx.RoundState.X` directly; only `IncrementTurnsStationary` has a helper method.
- Use `ctx.LogPerk(PerkID, squadID, msg)` for logging.

**If you're adding a new hook method to `PerkBehavior`:**
- Add to the interface in `hooks.go`.
- Add a no-op default to `BasePerkBehavior`.
- Add a dispatcher method in `perks/dispatcher.go` that calls `d.run(ownerSquadID, manager, HookContext{...}, func(ctx, b) bool { ... })`. Follow the existing shape.
- Add the invocation site in the combat pipeline (`combatcore/combatprocessing.go`, `combatmath.go`, or via `CombatService.powerPipeline` if it's a lifecycle event).
- If it needs early-termination semantics, return `false` from the callback to stop iteration.

**If you're adding a new lifecycle event:**
- Add a handler type + `On*`/`Fire*` pair to `powercore/pipeline.go`.
- Register artifact, perk, and GUI subscribers in `NewCombatService` in the appropriate order.
- Wire the invoking subsystem's callback to `cs.powerPipeline.Fire*`.

**If you're reading code in this area and find a discrepancy with the docs:**
Trust the code. The code is authoritative. Update the docs or flag the drift.

**If the user asks "are there more refactoring opportunities":**
Check this doc's section 5 first. Most pattern-level extractions (generic registry, generic validator, unified behavior interfaces) have been considered and rejected with reasons. Don't re-propose them unless you have a concrete use case that changes the calculus.

## 8. Test / verification invariants

- `go build ./...` must pass.
- `go test ./tactical/powers/... ./tactical/combat/...` must pass; full `go test ./...` is the smoke test.
- Manual smoke: start a battle, activate Deadlock Shackles on an enemy squad, confirm the lock fires on their next reset. Exercises pending effects + artifact→perk ordering.
- `config.DEBUG_MODE = true` will fail-fast at startup if behavior coverage validation fails. If the game doesn't boot in debug after adding a new perk/artifact, check `reportCoverage` output first.
