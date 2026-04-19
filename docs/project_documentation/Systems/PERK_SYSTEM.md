# Perk System

**Last Updated:** 2026-04-19
**Package:** `tactical/powers/perks`
**Related:** `tactical/powers/powercore/`, `tactical/combat/combattypes/perk_callbacks.go`, `tactical/combat/combatservices/combat_service.go`, `tactical/combat/combatservices/combat_power_dispatch.go`

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Hook Types Reference](#3-hook-types-reference)
4. [PerkBehavior Interface — Attacker vs Defender Methods](#4-perkbehavior-interface--attacker-vs-defender-methods)
5. [PerkRoundState Lifecycle](#5-perkroundstate-lifecycle)
6. [All 21 Perks Reference Table](#6-all-21-perks-reference-table)
7. [How to Add a New Perk (Step-by-Step)](#7-how-to-add-a-new-perk-step-by-step)
8. [Common Patterns](#8-common-patterns)
9. [Perk Activation Feedback](#9-perk-activation-feedback)

---

## 1. Overview

Perks are squad-level passive abilities that modify combat behavior. A squad equips perks into a fixed number of slots, and those perks automatically fire during combat via an interface-based dispatch system — no polling, no manual checks scattered through attack code.

### How Perks Differ from Artifacts and Spells

| Dimension | Perks | Artifacts | Spells |
|---|---|---|---|
| Activation | Automatic — fires on hooks | Passive or charge-triggered | Explicit cast action |
| Scope | Squad-level passive modifiers | Per-squad equipment effects | Targeted abilities with costs |
| Persistence | Survives between combats | Survives between combats | Consumed or on cooldown |
| State | `PerkRoundState` (per-combat) | `ArtifactChargeTracker` | Spell cooldown/cost |
| Definition | `perkdata.json` | `major_artifacts.json` / `minor_artifacts.json` | Spell templates |

### The Interface-Based Architecture

The system uses two interfaces:

- **`PerkBehavior`** — defines the contract for individual perks. Each perk is a struct embedding `BasePerkBehavior` (which provides no-op defaults) and overriding only the methods it needs. Registered via `RegisterPerkBehavior()`. This mirrors the `ArtifactBehavior` pattern in the artifact system.

- **`PerkDispatcher`** (in `combattypes/perk_callbacks.go`) — defines how perks are dispatched into the combat damage pipeline. `SquadPerkDispatcher` implements this by iterating equipped `PerkBehavior` implementations for each squad. Injected into the combat system via `SetPerkDispatcher()`.

This approach means:

- Adding a new perk never requires touching existing combat code — implement the interface, register in `init()`.
- Each perk is a self-contained struct — all hooks for a perk are methods on one type.
- The combat engine works identically with zero perks or many perks; the dispatcher handles empty iterations as no-ops.

### Shared powercore Foundation

Both artifacts and perks share the `tactical/powers/powercore` package, which provides:

- **`PowerContext`** — holds `Manager`, `Cache`, `RoundNumber`, `Logger`. Embedded by both `HookContext` (perks) and `BehaviorContext` (artifacts) so those fields live in one place.
- **`PowerLogger`** / **`LoggerFunc`** — a single logging interface shared by both systems. Replaces the old per-package `SetPerkLogger` / `SetArtifactLogger` callbacks.
- **`PowerPipeline`** — an ordered event subscriber pipeline owned by `CombatService`. Power systems subscribe to `OnPostReset`, `OnAttackComplete`, `OnTurnEnd`, `OnMoveComplete`; fire order is registration order.

---

## 2. Architecture

### Package Layout

```
tactical/powers/perks/
├── components.go    -- ECS component structs: PerkSlotData, PerkRoundState, per-perk state structs,
│                       generic state accessors (GetPerkState, GetOrInitPerkState, SetPerkState,
│                       GetBattleState, GetOrInitBattleState, SetBattleState)
├── hooks.go         -- HookContext (embeds PowerContext), PerkBehavior interface, BasePerkBehavior,
│                       perkBehaviorImpls registry, RegisterPerkBehavior, HookContext.LogPerk,
│                       HookContext.IncrementTurnsStationary
├── dispatcher.go    -- SquadPerkDispatcher (implements combattypes.PerkDispatcher). Exposes damage
│                       pipeline hooks AND lifecycle dispatch methods (DispatchTurnStart,
│                       DispatchRoundEnd, DispatchAttackTracking, DispatchMoveTracking)
├── behaviors.go     -- All 21 perk behavior implementations, grouped by state lifecycle
│                       (stateless / per-round stateful / per-battle stateful) for readability
├── unithelpers.go   -- Shared unit-query helpers (FindLowestHPUnit, FindHighestDexUnitByRole,
│                       CountTanksInRow, HasWoundedUnit, FindFirstTankInSquad, GetUnitsInRow)
├── queries.go       -- HasPerk, GetEquippedPerkIDs, GetRoundState, ForEachFriendlySquad,
│                       forEachPerkBehavior, getSquadIDForUnit
├── registry.go      -- PerkDefinition, PerkID type, PerkTier/PerkCategory enums, PerkRegistry,
│                       LoadPerkDefinitions, ValidateHookCoverage
├── balanceconfig.go -- PerkBalanceConfig, per-perk balance structs, LoadPerkBalanceConfig,
│                       validatePerkBalance
├── system.go        -- EquipPerk, UnequipPerk, InitializeRoundState, CleanupRoundState,
│                       InitializePerkRoundStatesForFaction, HasAnyPerks,
│                       ResetPerkRoundStateTurn, ResetPerkRoundStateRound, RunTurnStartHooks
├── perkids.go       -- Typed PerkID constants (compile-time string safety)
├── init.go          -- ECS subsystem registration (PerkSlotComponent, PerkRoundStateComponent,
│                       PerkSlotTag)
└── perks_test.go    -- Unit tests

tactical/powers/powercore/
├── context.go  -- PowerContext (shared runtime context for artifacts + perks)
├── logger.go   -- PowerLogger interface, LoggerFunc adapter, nil-safe ctx.Log helper
└── pipeline.go -- PowerPipeline (ordered subscriber lists for PostReset, AttackComplete,
                  TurnEnd, MoveComplete events)

tactical/combat/combattypes/
└── perk_callbacks.go -- PerkDispatcher interface (no perks import; 9 methods)

tactical/combat/combatservices/
├── combat_service.go        -- Owns PowerPipeline; wires artifact + perk dispatchers as
│                                subscribers in NewCombatService
└── combat_power_dispatch.go -- setupPowerDispatch: constructs the shared PowerLogger and
                                 injects it into both dispatchers. Pipeline subscription lives
                                 in NewCombatService

assets/gamedata/
├── perkdata.json           -- Static definitions: id, name, description, tier, category, roles,
│                              exclusiveWith, unlockCost
└── perkbalanceconfig.json  -- Per-perk balance tuning values
```

### The Circular Import Problem and the Interface Bridge

The combat engine (`combatcore`) needs to call perk logic during damage calculation. The perk logic (`perks`) imports `combatcore` types indirectly (via `combattypes`) and also uses `combatstate` helpers like `GetSquadFaction` and `GetActiveSquadsForFaction`. If `combatcore` and `perks` imported each other directly, the build would fail.

The solution is an interface bridge:

```
combattypes    (defines PerkDispatcher interface — no perks import; DamageModifiers,
                CoverBreakdown types)
    ↑
combatcore     (imports combattypes, calls dispatcher.Method() in damage pipeline)
    ↑
perks          (imports combattypes + combatstate; provides SquadPerkDispatcher
                implementing PerkDispatcher)
    ↑
combatservices (imports combattypes, combatcore, perks, artifacts, powercore —
                wires them together through PowerPipeline)
```

`combattypes/perk_callbacks.go` defines `PerkDispatcher` with 9 methods matching the combat pipeline integration points. `combatcore` imports `combattypes` and calls dispatcher methods during damage calculation. `perks` provides `SquadPerkDispatcher` implementing the interface. `combatservices` imports all four and performs the wiring:

```go
perkDispatcher := &perks.SquadPerkDispatcher{}
perkDispatcher.SetLogger(logger)
cs.CombatActSystem.SetPerkDispatcher(perkDispatcher)
```

### Dispatch Wiring

`setupPowerDispatch` in `combatservices/combat_power_dispatch.go` is called once during `NewCombatService`. It has a narrow job:

1. Construct the shared `PowerLogger` (a `powercore.LoggerFunc`) that routes messages with a `[GEAR]` or `[PERK]` prefix based on whether the source is a registered artifact behavior (via `artifacts.IsRegisteredBehavior(source)`).
2. Inject the logger into `cs.artifactDispatcher` (already constructed in `NewCombatService`).
3. Construct `SquadPerkDispatcher`, inject the logger, store it on `cs.perkDispatcher`, and inject it into `cs.CombatActSystem`.

**Pipeline subscription happens in `NewCombatService`**, not in `setupPowerDispatch`. `CombatService` owns a `powercore.PowerPipeline` and registers subscribers once, in declared execution order:

```go
// PostReset
cs.powerPipeline.OnPostReset(cs.artifactDispatcher.DispatchPostReset)
cs.powerPipeline.OnPostReset(func(factionID, squadIDs) {
    cs.perkDispatcher.DispatchTurnStart(squadIDs, round, manager)
})

// AttackComplete
cs.powerPipeline.OnAttackComplete(cs.artifactDispatcher.DispatchOnAttackComplete)
cs.powerPipeline.OnAttackComplete(func(attackerID, defenderID, result) {
    cs.perkDispatcher.DispatchAttackTracking(attackerID, defenderID, manager)
})
cs.powerPipeline.OnAttackComplete(func(...) { cs.onAttackCompleteGUI(...) })

// TurnEnd
cs.powerPipeline.OnTurnEnd(cs.artifactDispatcher.DispatchOnTurnEnd)
cs.powerPipeline.OnTurnEnd(func(round) { cs.perkDispatcher.DispatchRoundEnd(manager) })
cs.powerPipeline.OnTurnEnd(func(...) { cs.onTurnEndGUI(...) })

// MoveComplete
cs.powerPipeline.OnMoveComplete(func(sid) { cs.perkDispatcher.DispatchMoveTracking(sid, manager) })
cs.powerPipeline.OnMoveComplete(func(...) { cs.onMoveCompleteGUI(...) })
```

Subsystem hooks forward directly into the pipeline:

```go
combatActSystem.SetOnAttackComplete(cs.powerPipeline.FireAttackComplete)
movementSystem.SetOnMoveComplete(cs.powerPipeline.FireMoveComplete)
turnManager.SetOnTurnEnd(cs.powerPipeline.FireTurnEnd)
turnManager.SetPostResetHook(cs.powerPipeline.FirePostReset)
```

The dispatcher lifecycle methods perform these jobs:

1. **`DispatchTurnStart(squadIDs, round, manager)`** — for each squad in the faction, calls `ResetPerkRoundStateTurn()` then `RunTurnStartHooks()`.
2. **`DispatchRoundEnd(manager)`** — queries every squad with `PerkSlotTag` and calls `ResetPerkRoundStateRound()`.
3. **`DispatchAttackTracking(attackerID, defenderID, manager)`** — sets `AttackedThisTurn = true` on the attacker's round state and `WasAttackedThisTurn = true` on the defender's.
4. **`DispatchMoveTracking(squadID, manager)`** — sets `MovedThisTurn = true` and resets `TurnsStationary = 0`.

### HookContext

All `PerkBehavior` methods receive a `*HookContext`. It embeds `powercore.PowerContext` (for `Manager`, `Cache`, `RoundNumber`, `Logger`) and adds perk-specific fields:

```go
type HookContext struct {
    powercore.PowerContext

    AttackerID      ecs.EntityID
    DefenderID      ecs.EntityID
    AttackerSquadID ecs.EntityID
    DefenderSquadID ecs.EntityID
    SquadID         ecs.EntityID   // Squad that owns the perk (TurnStart, DeathOverride)
    UnitID          ecs.EntityID   // Specific unit (DeathOverride, DamageRedirect)
    DamageAmount    int            // Incoming damage (DamageRedirect)
    RoundState      *PerkRoundState
}
```

Not all fields are populated for every method — some are zero-valued depending on context. For example, `TurnStart` has no attacker/defender; `DeathOverride` uses `SquadID`/`UnitID`. The dispatcher fills combat-oriented fields via `combatCtx(...)` and then calls `run()`, which populates the embedded `PowerContext` and `RoundState` once before iterating behaviors.

The `RoundState` pointer inside `HookContext` always belongs to the squad that owns the perk, never the opposing squad.

Two helper methods on `HookContext` cut boilerplate:

- `ctx.LogPerk(perkID, squadID, message)` — nil-safe log through the embedded `PowerLogger`. Converts the typed `PerkID` to the string `source` expected by `PowerLogger.Log`.
- `ctx.IncrementTurnsStationary(max)` — increments `RoundState.TurnsStationary` capped at `max`. Keeps the monotonic-up-to-max invariant in one place.

---

## 3. Hook Types Reference

All hook types are methods on the `PerkBehavior` interface. `BasePerkBehavior` provides no-op defaults for all methods — perks override only the ones they need.

### 3.1 AttackerDamageMod / DefenderDamageMod

```go
AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers)
DefenderDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers)
```

**When it fires:** Inside `processAttack`, after hit and dodge rolls pass, before final damage is computed. `AttackerDamageMod` iterates the attacker's perks; `DefenderDamageMod` iterates the defender's perks.

**What it modifies:** `combattypes.DamageModifiers` fields:

| Field | Type | Meaning |
|---|---|---|
| `DamageMultiplier` | `float64` | Multiplicative damage scalar (default 1.0) |
| `HitPenalty` | `int` | Subtracted from attacker's hit threshold (positive = worse) |
| `CritBonus` | `int` | Added to crit threshold (positive = more crits) |
| `SkipCrit` | `bool` | If true, crit roll is skipped entirely |
| `CoverBonus` | `float64` | Additional cover fraction added after perk cover hooks |
| `IsCounterattack` | `bool` | Read-only flag; behaviors can branch on this |

**Perks that use AttackerDamageMod:** `reckless_assault`, `executioners_instinct`, `isolated_predator`, `opening_salvo`, `last_line`, `cleave`, `bloodlust`, `grudge_bearer`, `counterpunch`, `deadshots_patience`

**Perks that use DefenderDamageMod:** `reckless_assault` (vulnerability), `shieldwall_discipline`, `vigilance`, `adaptive_armor`

**Example — Executioner's Instinct:**

```go
func (b *ExecutionersInstinctBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
    if HasWoundedUnit(ctx.DefenderSquadID, PerkBalance.ExecutionersInstinct.HPThreshold, ctx.Manager) {
        modifiers.CritBonus += PerkBalance.ExecutionersInstinct.CritBonus
        ctx.LogPerk(PerkExecutionersInstinct, ctx.AttackerSquadID, "crit bonus vs wounded target")
    }
}
```

The `HasWoundedUnit` helper lives in `unithelpers.go` — shared across any perk that inspects unit HP ratios.

---

### 3.2 TargetOverride

```go
TargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID
```

**When it fires:** At the start of `processAttack`, before any per-target loop.

**What it does:** Receives the default target list and returns a replacement list. `BasePerkBehavior` returns `defaultTargets` unchanged.

**Perks that use it:** `cleave` (adds rear-row targets), `precision_strike` (replaces target with lowest-HP enemy).

---

### 3.3 CounterMod

```go
CounterMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) (skipCounter bool)
```

**When it fires:** In `executeCounterattack`, after the main attack resolves, before the counterattack loop begins.

**What it does:** Can modify `counterModifiers` in place or return `true` to suppress the counterattack entirely. `BasePerkBehavior` returns `false`.

**Perks that use it:** `stalwart` (upgrades counter damage to 100%), `riposte` (removes counter hit penalty).

---

### 3.4 AttackerPostDamage / DefenderPostDamage

```go
AttackerPostDamage(ctx *HookContext, damageDealt int, wasKill bool)
DefenderPostDamage(ctx *HookContext, damageDealt int, wasKill bool)
```

**When it fires:** In `processAttack`, after damage is recorded and after the `DeathOverride` check. `AttackerPostDamage` iterates the attacker's perks; `DefenderPostDamage` iterates the defender's perks.

**Perks that use AttackerPostDamage:** `bloodlust` (increments kill counter)

**Perks that use DefenderPostDamage:** `grudge_bearer` (increments grudge stack)

---

### 3.5 TurnStart

```go
TurnStart(ctx *HookContext)
```

**When it fires:** Inside `SquadPerkDispatcher.DispatchTurnStart`, once per squad per faction turn, after `ResetPerkRoundStateTurn()` runs. Uses `ctx.SquadID`, `ctx.RoundNumber`, `ctx.RoundState`, `ctx.Manager`.

**Important ordering:** `ResetPerkRoundStateTurn()` runs before `TurnStart`. This means `TurnStart` sees the already-cleared per-turn fields, but the snapshot fields (`WasAttackedLastTurn`, `DidNotAttackLastTurn`, `WasIdleLastTurn`) have already been saved.

**Perks that use it:** `field_medic`, `reckless_assault`, `fortify`, `resolute`, `counterpunch`, `deadshots_patience`.

---

### 3.6 DefenderCoverMod

```go
DefenderCoverMod(ctx *HookContext, coverBreakdown *combattypes.CoverBreakdown)
```

**When it fires:** In `CalculateDamage`, after the base cover breakdown is computed, before cover reduction is applied to damage.

**What it modifies:** `CoverBreakdown.TotalReduction` — a float64 in `[0.0, 1.0]`. Behaviors must clamp to 1.0 after adding their bonus.

**Perks that use it:** `brace_for_impact` (flat +15%), `fortify` (up to +15% based on stationary turns).

---

### 3.7 DamageRedirect

```go
DamageRedirect(ctx *HookContext) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)
```

**When it fires:** In `processAttack`, after damage is calculated but before it is applied. Uses `ctx.UnitID`, `ctx.SquadID`, `ctx.DamageAmount`.

**What it does:** Returns `(0, 0, 0)` from `BasePerkBehavior` (no redirect). If `redirectTargetID != 0`, the combat engine splits the damage.

**Perks that use it:** `guardian_protocol`.

---

### 3.8 DeathOverride

```go
DeathOverride(ctx *HookContext) (preventDeath bool)
```

**When it fires:** In `processAttack`, after `WasKilled` is set to `true`, before post-damage hooks run. Uses `ctx.UnitID`, `ctx.SquadID`, `ctx.RoundState`.

**What it does:** If it returns `true`, the combat engine retroactively reduces recorded damage so the unit survives at 1 HP. `BasePerkBehavior` returns `false`.

**Perks that use it:** `resolute`.

---

## 4. PerkBehavior Interface — Attacker vs Defender Methods

### The Interface

```go
type PerkBehavior interface {
    PerkID() PerkID

    AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers)
    DefenderDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers)
    DefenderCoverMod(ctx *HookContext, coverBreakdown *combattypes.CoverBreakdown)
    TargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID
    CounterMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) (skipCounter bool)
    AttackerPostDamage(ctx *HookContext, damageDealt int, wasKill bool)
    DefenderPostDamage(ctx *HookContext, damageDealt int, wasKill bool)
    TurnStart(ctx *HookContext)
    DamageRedirect(ctx *HookContext) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)
    DeathOverride(ctx *HookContext) (preventDeath bool)
}
```

`PerkID()` returns the typed `perks.PerkID` string (not a bare `string`) — catches typos at compile time. `BasePerkBehavior` provides no-op defaults for all other methods. Perks embed it and override only the methods they need.

### Why the Attacker/Defender Split Exists

For `DamageModifiers` and `CoverBreakdown`, a perk's behavior is fundamentally different depending on whether the perk-owning squad is attacking or defending. Without the split, every method body would need a self-check like `if !HasPerk(ctx.AttackerSquadID, "my_perk", ctx.Manager) { return }`, which is redundant since the dispatcher already knows which squad owns the perk.

The split eliminates that boilerplate:

- `SquadPerkDispatcher.AttackerDamageMod` iterates the **attacker's** perk list and calls each behavior's `AttackerDamageMod`.
- `SquadPerkDispatcher.DefenderDamageMod` iterates the **defender's** perk list and calls each behavior's `DefenderDamageMod`.
- `SquadPerkDispatcher.CoverMod` always calls `DefenderCoverMod` because cover only applies to the defending unit.

### The Reckless Assault Multi-Method Pattern

`RecklessAssaultBehavior` overrides three methods on the same struct:

```go
type RecklessAssaultBehavior struct{ BasePerkBehavior }

func (b *RecklessAssaultBehavior) PerkID() PerkID                           { return PerkRecklessAssault }
func (b *RecklessAssaultBehavior) TurnStart(ctx *HookContext)                { /* resets vulnerability */ }
func (b *RecklessAssaultBehavior) AttackerDamageMod(ctx, modifiers)          { /* +30% dmg, sets vulnerable */ }
func (b *RecklessAssaultBehavior) DefenderDamageMod(ctx, modifiers)          { /* +20% incoming when vulnerable */ }
```

All three methods use `ctx.RoundState` belonging to the perk-owning squad. A perk can override any combination of methods, or none (inheriting no-ops from `BasePerkBehavior`).

### Dispatcher Flow

`SquadPerkDispatcher` implements `PerkDispatcher` by iterating equipped behaviors. Every damage-pipeline method uses a shared `run` primitive that builds the `HookContext` once, then invokes a per-method callback:

```go
func (d *SquadPerkDispatcher) run(ownerSquadID ecs.EntityID, manager *common.EntityManager,
    ctx HookContext, hook func(*HookContext, PerkBehavior) bool) {

    roundState := GetRoundState(ownerSquadID, manager)
    if roundState == nil {
        return
    }
    ctx.PowerContext = powercore.PowerContext{Manager: manager, Logger: d.logger}
    ctx.RoundState = roundState
    forEachPerkBehavior(ownerSquadID, manager, func(b PerkBehavior) bool {
        return hook(&ctx, b)
    })
}
```

Return `false` from the hook callback to terminate iteration early — used by `CounterMod`, `DeathOverride`, and `DamageRedirect`. No nil-checks on individual methods are needed; `BasePerkBehavior` provides no-ops and `forEachPerkBehavior` iterates zero items when the squad has no perks.

---

## 5. PerkRoundState Lifecycle

`PerkRoundState` is an ECS component attached to each squad entity at the start of combat (via `InitializePerkRoundStatesForFaction`) and removed when combat ends (via `CleanupRoundState`).

### State Fields by Lifetime

`PerkRoundState` has two layers: **shared tracking fields** that live directly on the struct (set by the dispatcher, read by multiple perks), and **per-perk isolated state** stored in maps keyed by `PerkID`.

#### Shared Tracking Fields (direct struct fields)

**Per-turn** (reset by `ResetPerkRoundStateTurn()` before each squad's turn):

| Field | Type | Writer | Reader |
|---|---|---|---|
| `MovedThisTurn` | `bool` | `SquadPerkDispatcher.DispatchMoveTracking` | Stalwart, Fortify |
| `AttackedThisTurn` | `bool` | `SquadPerkDispatcher.DispatchAttackTracking` | ResetPerkRoundStateTurn (snapshot) |
| `WasAttackedThisTurn` | `bool` | `SquadPerkDispatcher.DispatchAttackTracking` | ResetPerkRoundStateTurn (snapshot) |

**Cross-round but movement-dependent** (not reset by either method; modified explicitly):

| Field | Type | Writer | Reader |
|---|---|---|---|
| `TurnsStationary` | `int` | Fortify TurnStart + DispatchMoveTracking | Fortify DefenderCoverMod |

**Snapshot fields** (computed by `ResetPerkRoundStateTurn()` from prior-turn values, then read by `TurnStart` methods):

| Field | Type | Used By |
|---|---|---|
| `WasAttackedLastTurn` | `bool` | Counterpunch |
| `DidNotAttackLastTurn` | `bool` | Counterpunch |
| `WasIdleLastTurn` | `bool` | Deadshot's Patience |

#### Per-Perk Isolated State (PerkState / PerkBattleState maps)

Per-perk state lives in two `map[PerkID]any` maps. Each perk defines its own small state struct in `components.go` and accesses it via generic helpers:

```go
// Round state (cleared by ResetPerkRoundStateRound)
state := GetPerkState[*BloodlustState](ctx.RoundState, PerkBloodlust)
SetPerkState(ctx.RoundState, PerkBloodlust, &BloodlustState{KillsThisRound: 1})

// Battle state (persists entire combat, never reset)
state := GetBattleState[*ResoluteState](ctx.RoundState, PerkResolute)
state := GetOrInitBattleState(ctx.RoundState, PerkResolute, func() *ResoluteState { ... })
```

**Per-round state structs** (stored in `PerkState`, cleared by `ResetPerkRoundStateRound`):

| Struct | Fields | Used By |
|---|---|---|
| `RecklessAssaultState` | `Vulnerable bool` | Reckless Assault |
| `AdaptiveArmorState` | `AttackedBy map[EntityID]int` | Adaptive Armor |
| `BloodlustState` | `KillsThisRound int` | Bloodlust |
| `CounterpunchState` | `Ready bool` | Counterpunch |
| `DeadshotState` | `Ready bool` | Deadshot's Patience |

**Per-battle state structs** (stored in `PerkBattleState`, persist entire combat):

| Struct | Fields | Used By |
|---|---|---|
| `OpeningSalvoState` | `HasAttackedThisCombat bool` | Opening Salvo |
| `ResoluteState` | `Used map[EntityID]bool`, `RoundStartHP map[EntityID]int` | Resolute |
| `GrudgeBearerState` | `Stacks map[EntityID]int` | Grudge Bearer |

### State Flow Diagram

```
Combat Start
    │
    ▼
InitializePerkRoundStatesForFaction (system.go)
    Creates fresh PerkRoundState{} on each squad entity with perks
    │
    ▼
── ROUND LOOP ──
    │
    ▼
DispatchRoundEnd  (via powerPipeline.OnTurnEnd subscriber)
    For every squad with PerkSlotTag: ResetPerkRoundStateRound(state)
    → PerkState = nil (clears all per-perk round state maps)
    → PerkBattleState is preserved
    │
    ▼
── TURN LOOP (per faction) ──
    │
    ▼
DispatchTurnStart  (via powerPipeline.OnPostReset subscriber)
    For each squad in faction:
      ResetPerkRoundStateTurn(state)
        Saves snapshots:
          WasAttackedLastTurn  = WasAttackedThisTurn
          DidNotAttackLastTurn = !AttackedThisTurn
          WasIdleLastTurn      = !MovedThisTurn && !AttackedThisTurn
        Clears per-turn shared fields:
          MovedThisTurn = false
          AttackedThisTurn = false
          WasAttackedThisTurn = false
      RunTurnStartHooks(squadID, round, state, manager, logger)
        field_medic: heals lowest-HP unit
        fortify: increments or resets TurnsStationary
        resolute: snapshots current HP into ResoluteState.RoundStartHP
        counterpunch: evaluates snapshots → sets CounterpunchState.Ready
        deadshots_patience: evaluates WasIdleLastTurn → sets DeadshotState.Ready
    │
    ▼
── ATTACK EXCHANGES ──
    dispatcher.AttackerDamageMod / dispatcher.DefenderDamageMod
    dispatcher.CoverMod
    dispatcher.TargetOverride
    dispatcher.DamageRedirect
    dispatcher.DeathOverride
    dispatcher.AttackerPostDamage / dispatcher.DefenderPostDamage
    DispatchAttackTracking: sets AttackedThisTurn, WasAttackedThisTurn
    DispatchMoveTracking:   sets MovedThisTurn, resets TurnsStationary
    │
    ▼
    (next faction turn → back to DispatchTurnStart)
    │
    ▼
    (next round → back to DispatchRoundEnd)

Combat End
    │
    ▼
CleanupRoundState (system.go)
    Removes PerkRoundStateComponent from each squad entity
```

### Per-Perk State Access

Per-perk state is stored in `PerkState` (round) and `PerkBattleState` (battle) maps, accessed via generic helpers. Use `GetOrInitPerkState` / `GetOrInitBattleState` when the state needs to be created on first access:

```go
state := GetOrInitBattleState(ctx.RoundState, PerkResolute, func() *ResoluteState {
    return &ResoluteState{
        Used:         make(map[ecs.EntityID]bool),
        RoundStartHP: make(map[ecs.EntityID]int),
    }
})
```

The `PerkState` map is set to `nil` by `ResetPerkRoundStateRound()`, which releases all per-perk round state at once. `PerkBattleState` persists the entire combat and is cleaned up by `CleanupRoundState`.

---

## 6. All 21 Perks Reference Table

Tier values: 0 = Combat Conditioning, 1 = Combat Specialization.
Category values: 0 = Offense, 1 = Defense, 2 = Tactical, 3 = Reactive, 4 = Doctrine.

**Mutual exclusion pairs:** `cleave` ↔ `precision_strike`, `reckless_assault` ↔ `stalwart`, `bloodlust` ↔ `field_medic`.

| ID | Name | Tier | Category | Roles | Cost | Methods Overridden | Description |
|---|---|---|---|---|---|---|---|
| `brace_for_impact` | Brace for Impact | 0 | Defense | Tank | 2 | DefenderCoverMod | +15% cover bonus when defending |
| `reckless_assault` | Reckless Assault | 0 | Offense | DPS | 2 | TurnStart, AttackerDamageMod, DefenderDamageMod | +30% damage dealt; +20% damage received until next turn |
| `stalwart` | Stalwart | 0 | Defense | Tank, Support | 2 | CounterMod | If squad did not move, counterattacks deal 100% damage instead of 50% |
| `executioners_instinct` | Executioner's Instinct | 0 | Offense | DPS | 2 | AttackerDamageMod | +25% crit chance vs any squad with a unit below 30% HP |
| `shieldwall_discipline` | Shieldwall Discipline | 0 | Defense | Tank | 2 | DefenderDamageMod | Per Tank in front row: -5% incoming damage (max 15%) |
| `isolated_predator` | Isolated Predator | 0 | Offense | DPS | 2 | AttackerDamageMod | +25% damage when no friendly squads within 3 tiles |
| `vigilance` | Vigilance | 0 | Defense | Tank, Support | 2 | DefenderDamageMod | Critical hits against this squad become normal hits |
| `field_medic` | Field Medic | 0 | Reactive | Support | 2 | TurnStart | At round start, lowest-HP unit heals 10% of max HP |
| `opening_salvo` | Opening Salvo | 0 | Offense | DPS | 2 | AttackerDamageMod | +35% damage on squad's first attack of the entire combat |
| `last_line` | Last Line | 0 | Defense | Tank, Support | 2 | AttackerDamageMod | When last surviving friendly squad: +20% hit, dodge, and damage |
| `cleave` | Cleave | 1 | Tactical | DPS | 3 | TargetOverride, AttackerDamageMod | Melee attacks also hit one unit in the row behind the target; -30% damage to all targets |
| `riposte` | Riposte | 1 | Defense | Tank, DPS | 3 | CounterMod | Counterattacks have no hit penalty (normally -20) |
| `guardian_protocol` | Guardian Protocol | 1 | Defense | Tank | 4 | DamageRedirect | When adjacent friendly squad is attacked, one Tank absorbs 25% of damage |
| `adaptive_armor` | Adaptive Armor | 1 | Defense | Tank | 3 | DefenderDamageMod | -10% damage from same attacker per hit (stacks to 30%, resets each round) |
| `bloodlust` | Bloodlust | 1 | Offense | DPS | 3 | AttackerPostDamage, AttackerDamageMod | Each unit kill this round grants +15% damage on the next attack (stacks, resets per round) |
| `fortify` | Fortify | 1 | Defense | Tank, Support | 3 | TurnStart, DefenderCoverMod | +5% cover per consecutive stationary turn (max +15% after 3 turns; moving resets) |
| `precision_strike` | Precision Strike | 1 | Tactical | DPS | 3 | TargetOverride | Highest-dex DPS unit redirects to lowest-HP enemy instead of normal targeting |
| `resolute` | Resolute | 1 | Defense | Tank, DPS, Support | 4 | TurnStart, DeathOverride | A unit survives a killing blow at 1 HP if it had >50% HP at round start (once per unit per battle) |
| `grudge_bearer` | Grudge Bearer | 1 | Reactive | DPS, Tank | 3 | DefenderPostDamage, AttackerDamageMod | +20% damage vs squads that have damaged this squad (stacks to +40%) |
| `counterpunch` | Counterpunch | 1 | Reactive | DPS, Tank | 3 | TurnStart, AttackerDamageMod | If attacked last turn AND did not attack last turn, next attack deals +40% damage |
| `deadshots_patience` | Deadshot's Patience | 1 | Offense | DPS | 4 | TurnStart, AttackerDamageMod | If completely idle last turn (no move, no attack), next ranged or magic attack gains +50% damage and +20 accuracy |

---

## 7. How to Add a New Perk (Step-by-Step)

This section walks through adding a hypothetical new perk: **Iron Will** — a Tier 1 Defense perk for Tank and Support units that reduces all damage taken by 10% when any unit in the squad is below 50% HP.

### Step 1: Add the JSON Definition

Open `assets/gamedata/perkdata.json` and append to the `"perks"` array:

```json
{
  "id": "iron_will",
  "name": "Iron Will",
  "description": "When any squad unit falls below 50% HP, all units take 10% less damage.",
  "tier": 1,
  "category": 1,
  "roles": ["Tank", "Support"],
  "exclusiveWith": [],
  "unlockCost": 3
}
```

### Step 2: Add the Perk ID Constant

In `perkids.go`, add:

```go
const PerkIronWill PerkID = "iron_will"
```

The typed `PerkID` means typos in behavior code fail compilation rather than at runtime.

### Step 3: Create the Behavior Struct

Add it to `behaviors.go` in the section that matches its state lifetime:

- **Stateless** — no per-combat state reads/writes. Pure function of `HookContext` + ECS queries.
- **Per-round stateful** — reads or writes `PerkRoundState` maps. Cleared by `ResetPerkRoundStateRound`.
- **Per-battle stateful** — uses `PerkBattleState`. Persists entire combat.

Iron Will is stateless:

```go
type IronWillBehavior struct{ BasePerkBehavior }

func (b *IronWillBehavior) PerkID() PerkID { return PerkIronWill }

func (b *IronWillBehavior) DefenderDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) {
    if HasWoundedUnit(ctx.DefenderSquadID, 0.5, ctx.Manager) {
        modifiers.DamageMultiplier *= 0.90
        ctx.LogPerk(PerkIronWill, ctx.DefenderSquadID, "-10% damage (squad wounded)")
    }
}
```

`HasWoundedUnit` already lives in `unithelpers.go` — no need to reimplement unit-iteration logic.

### Step 4: Register in init()

In `behaviors.go`'s existing `init()` function:

```go
RegisterPerkBehavior(&IronWillBehavior{})
```

Place it under the section comment that matches its state lifetime (stateless / per-round / per-battle) for readability only — dispatch does not inspect which section a behavior lives in.

### Step 5: Add Per-Perk State (if needed)

Iron Will is stateless — no state struct needed. If your perk needs state, define a small struct in `components.go`:

```go
type IronWillState struct {
    TriggeredThisRound bool
}
// Access: GetPerkState[*IronWillState](ctx.RoundState, PerkIronWill)
// Store:  SetPerkState(ctx.RoundState, PerkIronWill, &IronWillState{...})
```

### Step 6: Verify

1. `go build ./...` — confirms no compile errors.
2. `go test ./tactical/powers/perks/...` — runs perk tests.
3. On startup, `LoadPerkDefinitions()` + `ValidateHookCoverage()` prints a warning for any ID that has a JSON entry but no registered behavior, or vice versa.

### Complete Worked Example Summary

| Step | File | What to Do |
|---|---|---|
| 1 | `assets/gamedata/perkdata.json` | Append JSON object |
| 2 | `tactical/powers/perks/perkids.go` | Add `PerkIronWill PerkID = "iron_will"` constant |
| 3 | `tactical/powers/perks/behaviors.go` | Create behavior struct, embed `BasePerkBehavior`, override methods |
| 4 | `tactical/powers/perks/behaviors.go` | Add `RegisterPerkBehavior(&IronWillBehavior{})` in `init()` |
| 5 | `tactical/powers/perks/components.go` | Add per-perk state struct if needed |
| 6 | Terminal | Build and test |

---

## 8. Common Patterns

### Reading All Units in a Squad

Prefer `unithelpers.go` functions (`FindLowestHPUnit`, `HasWoundedUnit`, `CountTanksInRow`, etc.) over open-coded loops — they reuse `squadcore.GetAliveUnitAttributes` so the alive-check lives in one place. Only hand-roll if you need a new filter:

```go
for _, uid := range squadcore.GetUnitIDsInSquad(squadID, ctx.Manager) {
    attr := squadcore.GetAliveUnitAttributes(uid, ctx.Manager)
    if attr == nil {
        continue // dead/missing — skip
    }
    // ... use attr
}
```

### Checking a Unit's Role

```go
entity := ctx.Manager.FindEntityByID(unitID)
if entity == nil {
    continue
}
roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
if roleData == nil {
    continue
}
if roleData.Role == unitdefs.RoleTank {
    // ...
}
```

### Iterating Friendly Squads

Use `ForEachFriendlySquad` in `queries.go`:

```go
ForEachFriendlySquad(ctx.AttackerSquadID, ctx.Manager, func(friendlyID ecs.EntityID) bool {
    // return false to stop iteration early
    return true
})
```

It handles the "no faction" edge case and skips `squadID` automatically.

### Both-Sides Perks (Two Methods, Same Struct)

When a perk needs to fire on both attacker and defender turns, override both methods on the same struct:

```go
type MyPerkBehavior struct{ BasePerkBehavior }

func (b *MyPerkBehavior) PerkID() PerkID { return PerkMyPerk }
func (b *MyPerkBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) { /* ... */ }
func (b *MyPerkBehavior) DefenderDamageMod(ctx *HookContext, modifiers *combattypes.DamageModifiers) { /* ... */ }
```

Both methods receive `ctx.RoundState` pointing to their own squad's state. If the attacker method sets a flag and the defender method needs to read it, both are reading the same squad's `PerkRoundState`.

### Lazy State Initialization

Per-perk state structs that contain maps should use `GetOrInitPerkState` / `GetOrInitBattleState`:

```go
state := GetOrInitPerkState(ctx.RoundState, PerkAdaptiveArmor, func() *AdaptiveArmorState {
    return &AdaptiveArmorState{AttackedBy: make(map[ecs.EntityID]int)}
})
state.AttackedBy[attackerID]++
```

### Checking HasPerk from Inside a Behavior

You should rarely need this — the attacker/defender method split removes most such needs. However, `guardian_protocol` legitimately needs to check whether a neighboring squad also has the perk:

```go
if !HasPerk(friendlyID, PerkGuardianProtocol, ctx.Manager) {
    continue
}
```

---

## 9. Perk Activation Feedback

When a perk has a meaningful effect during combat, it logs a perk-specific message via the shared `powercore.PowerLogger`.

### Infrastructure

Perk logging flows through `HookContext.LogPerk`, which is nil-safe and routes through the embedded `PowerContext.Log`:

```go
func (ctx *HookContext) LogPerk(perkID PerkID, squadID ecs.EntityID, message string) {
    ctx.Log(string(perkID), squadID, message)
}
```

`ctx.Log` is inherited from `powercore.PowerContext` and is nil-safe — if no logger is set (e.g., in tests) the call silently no-ops.

### Wiring

The logger is constructed once in `combatservices/combat_power_dispatch.go` and shared between artifacts and perks:

```go
logger := powercore.LoggerFunc(func(source string, squadID ecs.EntityID, message string) {
    prefix := "[PERK]"
    if artifacts.IsRegisteredBehavior(source) {
        prefix = "[GEAR]"
    }
    fmt.Printf("%s %s: %s (squad %d)\n", prefix, source, message, squadID)
})

cs.artifactDispatcher.SetLogger(logger)
perkDispatcher.SetLogger(logger)
```

Routing between `[GEAR]` and `[PERK]` is decided by asking `artifacts.IsRegisteredBehavior(source)` — the `source` string passes through unchanged from the call site (perk ID or artifact behavior key).

Per-combat wiring keeps the logger on the dispatcher instance (not a package-global), so each combat can theoretically wire its own without mutating shared state.

### Example Messages

| Perk | Message |
|------|---------|
| Stalwart | "full-damage counterattack" |
| Bloodlust | "+30% damage (2 kills)" |
| Resolute | "unit survives lethal damage at 1 HP" |
| Reckless Assault | "+30% damage, now vulnerable" |
| Fortify | "+10% cover (2 turns stationary)" |
| Guardian Protocol | "tank absorbs 12 damage" |

Logging only fires when the perk has an observable effect — no-op paths produce no output.

---

## Appendix: File Quick Reference

| File | Purpose |
|---|---|
| `tactical/powers/perks/components.go` | `PerkSlotData`, `PerkRoundState`, per-perk state structs, generic state accessors |
| `tactical/powers/perks/hooks.go` | `HookContext` (embeds `powercore.PowerContext`); `PerkBehavior` interface; `BasePerkBehavior`; `perkBehaviorImpls` registry; `RegisterPerkBehavior`; `HookContext.LogPerk`; `HookContext.IncrementTurnsStationary` |
| `tactical/powers/perks/dispatcher.go` | `SquadPerkDispatcher` — implements `combattypes.PerkDispatcher` (9 damage-pipeline methods) and exposes lifecycle methods (`DispatchTurnStart`, `DispatchRoundEnd`, `DispatchAttackTracking`, `DispatchMoveTracking`) |
| `tactical/powers/perks/behaviors.go` | All 21 perk behavior implementations, grouped by state lifecycle |
| `tactical/powers/perks/unithelpers.go` | Shared unit-query helpers (`FindLowestHPUnit`, `HasWoundedUnit`, `CountTanksInRow`, …) |
| `tactical/powers/perks/queries.go` | `HasPerk`, `GetEquippedPerkIDs`, `GetRoundState`, `ForEachFriendlySquad`, `forEachPerkBehavior`, `getSquadIDForUnit` |
| `tactical/powers/perks/registry.go` | `PerkID` type, `PerkDefinition`, `PerkTier`/`PerkCategory` enums, `PerkRegistry`, `LoadPerkDefinitions`, `ValidateHookCoverage` |
| `tactical/powers/perks/balanceconfig.go` | `PerkBalanceConfig`, per-perk balance structs, `LoadPerkBalanceConfig`, `validatePerkBalance` |
| `tactical/powers/perks/system.go` | `EquipPerk`, `UnequipPerk`, `InitializeRoundState`, `CleanupRoundState`, `InitializePerkRoundStatesForFaction`, `HasAnyPerks`, `ResetPerkRoundStateTurn`, `ResetPerkRoundStateRound`, `RunTurnStartHooks` |
| `tactical/powers/perks/perkids.go` | Typed `PerkID` constants |
| `tactical/powers/perks/init.go` | ECS subsystem `init()` — registers `PerkSlotComponent`, `PerkRoundStateComponent`, `PerkSlotTag` |
| `tactical/powers/powercore/context.go` | `PowerContext` (shared runtime context embedded by both HookContext and BehaviorContext) |
| `tactical/powers/powercore/logger.go` | `PowerLogger` interface, `LoggerFunc` adapter, nil-safe `ctx.Log` |
| `tactical/powers/powercore/pipeline.go` | `PowerPipeline` — ordered subscriber lists for combat lifecycle events |
| `tactical/combat/combattypes/perk_callbacks.go` | `PerkDispatcher` interface |
| `tactical/combat/combatservices/combat_service.go` | Owns `PowerPipeline`; registers pipeline subscribers in order |
| `tactical/combat/combatservices/combat_power_dispatch.go` | `setupPowerDispatch`: constructs shared `PowerLogger`, injects it into both dispatchers |
| `assets/gamedata/perkdata.json` | Static perk definitions loaded at startup |
| `assets/gamedata/perkbalanceconfig.json` | Per-perk balance tuning values |
