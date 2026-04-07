# Perks & Hook System: Combined Analysis

**Date:** 2026-04-07
**Sources:** PERKS_ECS_ANALYSIS.md (ECS Reviewer, Tactical Simplifier, Refactoring Pro), HOOK_CALLBACK_ANALYSIS.md (trpg-creator, tactical-simplifier, refactoring-pro, ecs-reviewer, game-dev)
**Supersedes:** `PERKS_ECS_ANALYSIS.md`, `HOOK_CALLBACK_ANALYSIS.md`, `PERK_SYSTEM_ISSUES.md`

---

## Executive Summary

Eight independent agent reviews converge on the same conclusion: **the perks package is well-designed and should not be rewritten.** It scores 9/10 on ECS compliance, uses the same hybrid patterns as artifacts and spells, and its complexity is justified by the problem domain (passive modifiers hooking into a multi-stage combat pipeline with three state lifetimes).

The actionable path forward is:
1. Apply a small set of ECS compliance fixes (move methods off a component, reorganize files)
2. Add tests (zero exist today -- highest-risk gap)
3. Populate mutual exclusion data for meaningful equip-time trade-offs
4. Optionally merge the two dispatch files and add missing hook points to open new design space

There is no architectural restructuring needed.

---

## 1. Architecture Overview

### The Four-Layer Power System

| Layer | What It Modifies | How | Player Agency |
|-------|-----------------|-----|---------------|
| Minor Artifact | Base stats | Permanent `ActiveEffect` at battle start | Equip choice only |
| Perk | Combat calculations | Hook into damage pipeline mid-calculation | Equip choice only |
| Major Artifact | Action economy | Charge-gated activation or event reaction | Timing decisions |
| Spell | Stats or damage | Mana-gated activation | Timing + target |

Each layer turns a distinct knob. Perks answer "how does this squad fight?" while artifacts answer "what special orders can the commander issue?" This separation maps to genre conventions (Fire Emblem skills vs. XCOM commander abilities vs. basic equipment).

### Perk System

**Location:** `tactical/powers/perks/`

- **Components:** `PerkSlotData` (equipped perk IDs, max 3), `PerkRoundState` (combat tracking, 3-tier state)
- **Tag:** `PerkSlotTag` for efficient squad querying
- **Hook Model:** `PerkHooks` struct with 10 optional function pointer fields (8 hook types, attacker/defender variants)
- **Dispatch:** 10 `Run*` runner functions, wired to combat via `perk_dispatch.go`
- **Content:** 21 perks (11 stateless, 8 per-round stateful, 3 per-battle stateful)

### Artifact System

**Location:** `tactical/powers/artifacts/`

- **Hook Model:** `ArtifactBehavior` interface (7 methods) with `BaseBehavior` no-op defaults
- **Dispatch:** `behavior_dispatch.go` registers 3 event hooks
- **State:** `ArtifactChargeTracker` with per-battle/per-round charge maps
- **Content:** 6 behaviors (3 player-activated, 3 passive)

### How They Wire Into Combat

Both systems register callbacks on the same `CombatService` event lists. Execution order is determined by registration order in `NewCombatService`: artifact dispatch registers first, then perk dispatch.

| Event | Artifacts | Perks |
|-------|-----------|-------|
| `RegisterPostResetHook` | `OnPostReset` broadcast | TurnStart hooks + state reset |
| `RegisterOnAttackComplete` | Squad-scoped `OnAttackComplete` | State tracking (AttackedThisTurn, WasAttackedThisTurn) |
| `RegisterOnTurnEnd` | Charge refresh + `OnTurnEnd` broadcast | Per-round state reset |
| `RegisterOnMoveComplete` | -- | Movement tracking |

The perk damage pipeline hooks fire in `combatprocessing.go` at: TargetOverride -> DamageMod (attacker) -> DamageMod (defender) -> CoverMod -> DamageRedirect -> DeathOverride -> PostDamage. This order is semantically correct -- each stage depends on the previous one.

### Perk Lifecycle on Squad Entities

- **At equip time:** `EquipPerk` adds perk ID to `PerkSlotData.PerkIDs` (persists between combats)
- **Combat start:** `InitializeRoundState(squadID, manager)` adds `PerkRoundStateComponent` to each squad with perks
- **During combat:** Components accessed via standard ECS API:
  ```go
  data := common.GetComponentTypeByID[*PerkSlotData](manager, squadID, PerkSlotComponent)
  state := common.GetComponentTypeByID[*PerkRoundState](manager, squadID, PerkRoundStateComponent)
  ```
- **Combat end:** `CleanupRoundState(squadID, manager)` removes `PerkRoundStateComponent`
- **Between combats:** Only `PerkSlotData` persists (which perks are equipped)

### Performance

With max 3 perks per squad and turn-based combat, the iteration cost (map lookups + function calls) is negligible. The `forEachPerkHook` iterator walks a 3-element slice per squad. This would only matter for Monte Carlo AI simulation, which is not planned.

---

## 2. ECS Assessment

### Compliance Scorecard

| Category | Score | Notes |
|---|---|---|
| Pure Data Components | 7/10 | `PerkSlotData` is clean; `PerkRoundState` has mutating methods |
| EntityID Usage | 10/10 | No `*ecs.Entity` stored anywhere |
| Query-Based Relationships | 10/10 | All state access goes through ECS manager |
| System-Based Logic | 8/10 | Hook runners in `queries.go` instead of `system.go` |
| Value Map Keys | 10/10 | All maps use value types (`string`, `ecs.EntityID`) |
| Subsystem Registration | 10/10 | Standard `init()` + `RegisterSubsystem` pattern |
| **Overall** | **9/10** | |

### Pattern Comparison: How Powers Systems Use ECS

All three powers subsystems follow the same hybrid pattern -- global registries for static data, ECS components for per-entity state:

| System | Static Registry (Non-ECS) | Per-Entity State (ECS) |
|---|---|---|
| **Perks** | `PerkRegistry`, `hookRegistry` | `PerkSlotData`, `PerkRoundState` on squad entities |
| **Artifacts** | `templates.GetArtifactDefinition()`, `behaviorRegistry` | `EquipmentData`, `ArtifactInventoryData` on squad entities |
| **Spells** | `templates.GetSpellDefinition()` | `ManaData`, `SpellBookData` on squad entities |

This is the correct pattern. Static, immutable template data belongs in global registries. Per-entity mutable state belongs in ECS components.

### Where Perks Differ from the Gold-Standard (Squads)

| Aspect | Squads | Perks | Justified? |
|---|---|---|---|
| Views (cached queries) | `SquadQueryCache` with `*ecs.View` | None | Yes -- perks are accessed by known squad ID, not iterated |
| Component count | 7+ | 2 | Yes -- simpler per-entity state needs |
| Entity creation | Creates squad and unit entities | Attaches to existing squad entities | Yes -- perks are modifiers, not entities |
| Tags | Multiple (SquadTag, SquadMemberTag, LeaderTag) | One (PerkSlotTag) | Yes -- only one query pattern needed |

### Why More ECS Would Hurt

Converting per-perk state from `map[string]interface{}` to individual ECS components would:
- Explode component count (8+ new registrations, growing with each perk)
- Complicate lifecycle (round reset is currently one line: `s.PerkState = nil`)
- Add no query benefit (per-perk state is always accessed for a specific squad's specific perk)
- Require more init boilerplate per stateful perk

Converting global registries (`hookRegistry`, `PerkRegistry`, `PerkBalance`) to ECS would add query overhead with zero benefit -- no system ever iterates "all perk definitions" through ECS. These are type catalogs, not runtime entities.

Converting function pointer dispatch to ECS tags/components would lose type safety and the ability to register hooks from `init()` without a manager reference.

Return-value hooks (`TargetOverrideHook`, `DamageRedirectHook`) are another category where ECS has no advantage. ECS systems write to components; they do not return values up the call stack. Adding intermediate "result components" to mediate between the damage pipeline and perk hooks would be pure overhead.

### ArtifactChargeTracker: Acceptable Non-ECS

`ArtifactChargeTracker` (`artifacts/artifactcharges.go`) has private fields and 9 methods, functioning as an encapsulated domain service rather than an ECS component. However, there is only ever one instance per combat, and its lifetime is managed by `CombatService`. Converting to exported fields + free functions provides minimal practical benefit. This is technically non-ECS but acceptable given its single-instance, combat-scoped lifetime.

### Behavior Dispatch: Two Correct Patterns

| System | Dispatch Mechanism | Why Correct |
|---|---|---|
| **Perks** | `hookRegistry` maps perk ID to `*PerkHooks` (struct of function values) | Most perks implement 1-2 of 10 hooks. Nil fields are natural no-ops. Lighter than interface dispatch. |
| **Artifacts** | `behaviorRegistry` maps artifact ID to `ArtifactBehavior` (interface + `BaseBehavior`) | Coarser-grained with fewer methods. Interface dispatch is idiomatic Go at this cardinality. |

Both are valid. Converting either to the other's pattern would add complexity without benefit.

---

## 3. Redundancy Assessment: Perks vs. Artifacts

### Genuinely Duplicated (Could Share Infrastructure)

| Pattern | Perks | Artifacts | Worth Unifying? |
|---------|-------|-----------|-----------------|
| Dispatch wiring | `perk_dispatch.go` | `behavior_dispatch.go` | **Yes** -- merge into one file with explicit phase ordering |
| Event registration | 4 `Register*` calls | 3 `Register*` calls | Yes (side effect of dispatch merge) |

### Looks Duplicated But Is Not

| Pattern | Why Different |
|---------|---------------|
| Registries | Different payload types (function pointer struct vs interface) |
| Context objects | `HookContext` has 12 combat-event fields; `BehaviorContext` has 3 fields + cache/charges -- almost no overlap |
| State | Different data, different lifecycles (3-tier map vs charge tracker) |
| Iteration | `forEachPerkHook` per squad (3-element max) vs `AllBehaviors()` broadcast or squad-scoped |

**Verdict:** Do NOT merge perks and artifacts into a single system. A unified abstraction would create a 13-method God Interface where most methods are no-ops. The structural parallels are appropriate given their semantic differences.

---

## 4. Cognitive Load

### Adding a New Perk

**Files to touch:** 4-6
1. `perkids.go` -- add string constant
2. `perkdata.json` -- add JSON definition
3. `perkbalanceconfig.json` -- add balance values
4. `balanceconfig.go` -- add balance struct + field + validation
5. `behaviors_*.go` -- write behavior + `RegisterPerkHooks` in `init()`
6. `components.go` -- add state struct (only if stateful)

**Concepts required:** 10+ (hook signatures, context fields, state tiers, dispatch lifecycle, balance config pattern, state requirements annotation)

### Adding a New Artifact Behavior

**Files to touch:** 3
1. `artifactbehavior.go` -- add behavior key constant
2. `artifactbehaviors_*.go` -- struct + method overrides + `RegisterBehavior` in `init()`
3. Artifact data JSON -- add definition

**Concepts required:** 5 (behavior interface, context helpers, charges, pending effects)

Adding artifacts is simpler because `BaseBehavior` embedding eliminates runner functions, nil checks, and hook-specific iteration. The perk system's higher complexity is justified by the harder problem (modifying mid-calculation combat math at 8 distinct pipeline stages vs. reacting to 3 events).

### Why the 10 Runner Functions Can't Be Unified

The 8 hook function signatures are genuinely different -- they modify different data types (`DamageModifiers`, `CoverBreakdown`, target lists) and have different return values. Attempting to unify them into a common `CombatEvent` payload would lose type safety and force runtime type switching. The 10 `Run*` functions follow an identical structure (build context, iterate via `forEachPerkHook`, nil-check, call), and the helper functions `buildCombatContext` and `forEachPerkHook` already extract the common core. The remaining repetition is the lesser evil in Go -- each runner is straightforward, easy to modify per-hook, and explicitly typed.

---

## 5. Game Design Assessment

### Missing Player Feedback

When perks trigger, there is no visible feedback. Fire Emblem shows "Sol activated!" popups. Without feedback, players cannot learn when perks are helping.

**Recommendation:** Add combat log messages at minimum (e.g., "[PERK] Stalwart: full-damage counterattack").

### Mutual Exclusion Not Populated

All `ExclusiveWith` arrays in `perkdata.json` are empty. This is a missed opportunity for meaningful equip-time trade-offs:
- `cleave` <-> `precision_strike` (AoE vs single-target; also fixes the TargetOverride conflict)
- `reckless_assault` <-> `stalwart` (aggressive vs defensive identity)
- `bloodlust` <-> `field_medic` (kill focus vs heal focus)

### Spell-Perk Non-Interaction

`ExecuteSpellCast` in `spells/system.go` performs its own damage calculation without calling perk hooks. A squad with Reckless Assault that casts a damage spell gets no bonus. This may be intentional (spells operate "above" perks) but should be an explicit design decision, not an accidental gap.

### Genre Comparison

| Feature | Fire Emblem | FFT | XCOM | TinkerRogue |
|---------|-------------|-----|------|-------------|
| Passive combat modifiers | Skills | Support abilities | Perks | Perks |
| Commander abilities | Dancer | -- | Commander abilities | Major Artifacts |
| Equipment stat boosts | Weapons/items | Equipment | Armor/weapons | Minor Artifacts |
| Movement modifiers | Movement skills | Movement abilities | Perks | **Missing** |
| Reaction/interrupt | Counter/Vantage | Reaction abilities | Overwatch | **Missing** (Overwatch defined but no hook) |
| Mutual exclusion | Some skills | -- | Perk tree choices | **Not populated** |
| Activation feedback | Skill popup | -- | -- | **Missing** |

The three main genre gaps: movement-modifying perks (requires `PreMovement` hook), reaction/interrupt (requires same hook for Overwatch), and perk activation feedback.

---

## 6. What Should NOT Change

These aspects are well-designed and should be preserved:

**Architecture:**
- Four-layer power system separation (minor artifacts / perks / major artifacts / spells)
- Perks and artifacts as separate systems with separate dispatch
- Hook-based architecture -- scales to 40+ perks
- Three-layer import bridge (`combattypes` -> `combatcore` -> `combatservices`)

**Perk Internals:**
- `PerkHooks` function pointer struct (correct for sparse 1-2 of 10 hook population)
- Attacker/defender hook split (eliminates self-checks in all behavior code)
- `forEachPerkHook` iterator (centralizes iteration, supports early exit)
- `buildCombatContext` / `buildHookContext` helpers
- Three-file behavior split (stateless / stateful-round / stateful-battle)
- Per-perk state maps with generics (`GetPerkState[T]`, `SetPerkState`)
- Balance config JSON separation
- Perk ID constants in `perkids.go`
- `validateHookCoverage` startup check

**Artifact Internals:**
- `ArtifactBehavior` interface with `BaseBehavior` embedding
- `ArtifactChargeTracker` encapsulation (single-instance, combat-scoped)

**Global Registries:**
- `hookRegistry`, `behaviorRegistry`, `PerkRegistry`, `PerkBalance` -- these are type catalogs and config, not ECS entities

---

## 7. Recommended Changes

### Tier 1: Quick Wins

#### R1. Move Methods Off PerkRoundState (ECS Compliance)

**Files:** `components.go`, `system.go`, `perk_dispatch.go`
**Priority:** High | **Risk:** None

`ResetPerTurn()` and `ResetPerRound()` are mutating methods on an ECS component. Per the project's ECS conventions, they belong in system functions. Move to `system.go` as standalone functions:

```go
func ResetPerkRoundStateTurn(s *PerkRoundState) { ... }
func ResetPerkRoundStateRound(s *PerkRoundState) { ... }
```

Update call sites in `perk_dispatch.go`.

#### R2. Move Hook Runners to system.go (File Organization)

**Files:** `queries.go`, `system.go`
**Priority:** Medium | **Risk:** None

The 10 `Run*Hooks` functions modify `DamageModifiers`, `CoverBreakdown`, apply heals, and set state. They are system functions, not queries. Move them to `system.go`. Keep only the true queries in `queries.go`: `GetEquippedPerkIDs`, `HasPerk`, `GetRoundState`, context builders, `forEachPerkHook`.

#### R3. Populate ExclusiveWith in perkdata.json (Game Design)

**Files:** `assets/gamedata/perkdata.json`
**Priority:** Medium | **Risk:** None

Add mutual exclusion pairs:
- `cleave` <-> `precision_strike` (fixes TargetOverride conflict + meaningful AoE vs single-target choice)
- `reckless_assault` <-> `stalwart` (aggressive vs defensive)
- `bloodlust` <-> `field_medic` (kill focus vs heal focus)

#### R4. Remove StateRequirements / StateCategory (Cleanup)

**Files:** `hooks.go`, all `behaviors_*.go` files
**Priority:** Low | **Risk:** None

`StateRequirements` is documentation-as-code that is never enforced. The `validateHookCoverage` check produces a non-fatal NOTICE only. Doc comments on behavior functions provide more precise state dependency info. Remove `StateRequirements`, `StateCategory`, the `State` field from `PerkHooks`, and the state check from validation.

#### R5. Replace interface{} with any (Style)

**Files:** `components.go`
**Priority:** Low | **Risk:** None

Go 1.18+ style. Find-replace `interface{}` with `any` in setter signatures. The generic parameters already use `T any`.

#### R6. Extract Shared State Accessor Helper (Dedup)

**Files:** `components.go`
**Priority:** Low | **Risk:** Very low

The `GetPerkState`/`GetBattleState` accessor families are structurally identical, differing only in which map they operate on. Extract a shared `getFromMap[T]` / `setInMap` / `getOrInitFromMap[T]` helper, then implement both families as thin wrappers.

### Tier 2: Targeted Improvements

#### R7. Add Tests (Highest Value)

**Files:** New `*_test.go` files in `tactical/powers/perks/`
**Priority:** High | **Risk:** None (additive)

Zero tests exist. Priority areas:
- Stateful perk lifecycles (arm, fire, reset for round-scoped and battle-scoped perks)
- Shared tracking field snapshots (`ResetPerTurn` logic)
- Multi-perk interactions (Bloodlust + Resolute, Cleave + Precision Strike)
- Balance config loading (missing/zero values, validation)

The pure-function structure of stateless behaviors makes unit testing straightforward. The `HookContext` struct is self-contained and easy to construct for tests.

#### R8. Merge Dispatch Files

**Files:** `perk_dispatch.go`, `behavior_dispatch.go` -> single `combat_power_dispatch.go`
**Priority:** Medium | **Risk:** Low

Combine into one file with explicit phase ordering. Currently, a developer must find both files and mentally merge their event handlers to understand "what happens when an attack completes." A single file with clear phase comments makes the execution order visible and intentional.

#### R9. Add Perk Activation Feedback

**Files:** Combat log/messaging system, perk `Run*` functions
**Priority:** Medium | **Risk:** Low

Add combat log messages when perks trigger. Players need to see when perks are helping to make informed equip decisions. At minimum: "[PERK] Stalwart: full-damage counterattack", "[PERK] Bloodlust: +15% damage (1 kill)".

### Tier 3: Future Design Space (Hook Points)

These are not fixes -- they open new perk design space when needed.

#### High Priority

| Hook | What It Enables |
|------|----------------|
| `OnHealComplete` | Support perk design space ("+20% healing done", "cleanse debuffs on heal") |
| `OnSquadDestroyed` | Morale mechanics, revenge bonuses, martyrdom perks |
| `OnEffectApplied` / `OnEffectExpired` | Debuff immunity, duration reduction |

#### Medium Priority

| Hook | What It Enables | Notes |
|------|----------------|-------|
| `PreMovement` | Overwatch, movement cost modifiers, threat zones | **Required for Overwatch perk** |
| `OnSpellCast` | Magic-focused squad perks, spell damage modifiers | Requires design decision: do perks affect spells? |
| `PreCounterattack` | Attacker-side counter suppression ("suppress counter when flanking") | Currently only defender perks affect counters |
| `OnRoundStart` | Simultaneous round-start effects for all squads | Currently only per-faction TurnStart exists |

#### Lower Priority (Niche But Genre-Standard)

| Hook | What It Enables |
|------|----------------|
| `OnFormationChange` | One-time formation-reactive effects |
| `OnItemUsed` | Consumable interaction perks |
| `AttackerCoverMod` | Cover-ignoring offense perks ("ignore 50% cover") |
| `OnDodge` | Reactive dodge perks |
| `OnBattleStart` | One-time battle setup buffs |

#### Conditional: Modifier Priority/Stacking Policy

**Files:** `perks/hooks.go`, `perks/queries.go`, `combatcore/combatprocessing.go`
**Priority:** Only if perk count exceeds ~40 or slot count exceeds 5

Add a `Priority int` field to `PerkHooks` and sort perks by priority before iteration. Add stacking policies (additive vs. multiplicative) to `DamageModifiers` contributions. This prevents multiplicative stacking from producing runaway numbers at scale. Not needed at the current 21-perk, 3-slot scale.

### Not Recommended

| Anti-Recommendation | Why |
|---------------------|-----|
| Merge perks and artifacts into one system | Creates 13-method God Interface; systems serve different purposes |
| Convert per-perk state to individual ECS components | Explodes component count, complicates lifecycle, adds no query benefit |
| Convert registries to ECS entities | Registries are type catalogs, not runtime entities |
| Replace `PerkHooks` struct with interface | Function pointer struct is correct for sparse hook population |
| Build a generic event bus | Loses type safety; named callbacks are more debuggable |
| Make perks fully data-driven (JSON behaviors) | Complex perks (Guardian Protocol, Cleave, Resolute) need code |
| Replace `PerkCallbacks` bridge with interface | Split opinion -- one analysis recommended it (eliminates 76-line bridge file, replaces 7 duplicated type signatures with 1 interface), others said low value. The bridge works fine as-is; an interface adapter would be cleaner but the improvement is marginal. Optional at best. |

---

## 8. Open Issues

### Cleave + Precision Strike TargetOverride Conflict

Both perks register `TargetOverride` hooks. `forEachPerkHook` iterates in equip order. If Precision Strike fires after Cleave, it overrides Cleave's expanded target list. **Fix:** R3 (mutual exclusion in JSON).

### Dispatch Execution Order Is Implicit

Artifact dispatch registers before perk dispatch in `NewCombatService`. If this order changes, artifacts' `OnPostReset` (e.g., Deadlock Shackles locking a squad) may fire after perks' `TurnStart` hooks instead of before. **Fix:** R7 (merged dispatch file with explicit ordering).

### Overwatch Placeholder

`PerkOverwatch` has a constant in `perkids.go` and a JSON definition but no hook registration, triggering a validation warning at startup. Either implement it (requires `PreMovement` hook point) or remove the placeholder.

### Spell-Perk Non-Interaction

`ExecuteSpellCast` bypasses perk hooks. Needs an explicit design decision: should perks affect spell damage?

---

## 9. Summary: Path Forward

```
Phase 1: ECS Compliance & Cleanup (Tier 1 items)
  R1. Move methods off PerkRoundState
  R2. Move hook runners to system.go
  R3. Populate ExclusiveWith
  R4. Remove StateRequirements
  R5. Replace interface{} with any
  R6. Extract shared state accessor helper

Phase 2: Testing & Polish (Tier 2 items)
  R7. Add unit tests (highest value)
  R8. Merge dispatch files
  R9. Add perk activation feedback

Phase 3: New Design Space (Tier 3, as needed)
  Add hook points (OnHealComplete, PreMovement, OnSquadDestroyed, and others)
  Implement Overwatch perk (requires PreMovement hook)
  Decide on spell-perk interaction
  Add modifier priority/stacking policy (only if perks exceed ~40)
```

No architectural restructuring. No ECS rewrite. The system is sound -- invest in tests, trade-off data, and player feedback.
