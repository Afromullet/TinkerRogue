# Perk System

**Last Updated:** 2026-04-09
**Package:** `tactical/powers/perks`
**Related:** `tactical/combat/combattypes/perk_callbacks.go`, `tactical/combat/combatservices/combat_power_dispatch.go`

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
| Scope | Squad-level passive modifiers | Per-unit item effects | Targeted abilities with costs |
| Persistence | Survives between combats | Survives between combats | Consumed or on cooldown |
| State | `PerkRoundState` (per-combat) | Artifact charge counts | Spell cooldown/cost |
| Definition | `perkdata.json` | `artifactdata.json` | Spell templates |

### The Interface-Based Architecture

The system uses two interfaces:

- **`PerkBehavior`** — defines the contract for individual perks. Each perk is a struct embedding `BasePerkBehavior` (which provides no-op defaults) and overriding only the methods it needs. Registered via `RegisterPerkBehavior()`. This follows the same pattern as `ArtifactBehavior` in the artifact system.

- **`PerkDispatcher`** — defines how perks are dispatched into the combat damage pipeline. `SquadPerkDispatcher` implements this by iterating equipped `PerkBehavior` implementations for each squad. Injected into the combat system via `SetPerkDispatcher()`.

This approach means:

- Adding a new perk never requires touching existing combat code — implement the interface, register in `init()`.
- Each perk is a self-contained struct — all hooks for a perk are methods on one type.
- The combat engine works identically with zero perks or many perks; the dispatcher handles empty iterations as no-ops.

---

## 2. Architecture

### Package Layout

```
tactical/powers/perks/
├── components.go                -- ECS component structs: PerkSlotData, PerkRoundState, per-perk state structs
├── hooks.go                     -- PerkBehavior interface, BasePerkBehavior, behaviorRegistry, PerkLogger infrastructure
├── dispatcher.go                -- SquadPerkDispatcher (implements combattypes.PerkDispatcher)
├── behaviors_stateless.go       -- Stateless perk behaviors (pure functions of HookContext)
├── behaviors_stateful_round.go  -- Per-round stateful perk behaviors (use PerkState map)
├── behaviors_stateful_battle.go -- Per-battle stateful perk behaviors (use PerkBattleState map)
├── queries.go                   -- HasPerk, GetRoundState, forEachPerkBehavior, context builders
├── registry.go                  -- PerkDefinition, PerkRegistry, LoadPerkDefinitions, validateHookCoverage
├── balanceconfig.go             -- PerkBalanceConfig, per-perk balance structs, loaded from JSON
├── system.go                    -- EquipPerk, UnequipPerk, InitializeRoundState, CleanupRoundState,
│                                   ResetPerkRoundStateTurn, ResetPerkRoundStateRound, RunTurnStartHooks
├── perks_test.go                -- Unit tests (state accessors, reset logic, equip/unequip, perk lifecycles)
└── init.go                      -- ECS subsystem registration (PerkSlotComponent, PerkRoundStateComponent)

tactical/combat/combattypes/
└── perk_callbacks.go -- PerkDispatcher interface (no perks import)

tactical/combat/combatservices/
└── combat_power_dispatch.go  -- Wires both artifact behaviors and perk dispatcher into combat pipeline

assets/gamedata/
├── perkdata.json           -- Static definitions: id, name, description, tier, category, roles
└── perkbalanceconfig.json  -- Per-perk balance tuning values
```

### The Circular Import Problem and the Interface Bridge

The combat engine (`combatcore`) needs to call perk logic during damage calculation. The perk logic (`perks`) needs to call combat helpers like `GetSquadFaction` and `GetActiveSquadsForFaction` that live in `combatcore`. If both packages import each other directly, the Go compiler refuses to build.

The solution is a three-layer bridge using an interface:

```
combattypes           (defines PerkDispatcher interface — no perks import)
    ↑
combatcore            (imports combattypes, calls dispatcher.Method() in damage pipeline)
    ↑
perks                 (imports combatcore for DamageModifiers, CoverBreakdown types;
                       provides SquadPerkDispatcher implementing PerkDispatcher)
    ↑
combatservices        (imports combattypes, combatcore, and perks — wires them together)
```

`combattypes/perk_callbacks.go` defines the `PerkDispatcher` interface with 9 methods matching the combat pipeline integration points. `combatcore` imports `combattypes` and calls dispatcher methods during damage calculation. `perks` provides `SquadPerkDispatcher` implementing the interface. `combatservices/combat_power_dispatch.go` imports all three and performs the wiring:

```go
cs.CombatActSystem.SetPerkDispatcher(&perks.SquadPerkDispatcher{})
```

### Dispatch Wiring

`setupPowerDispatch` in `combatservices/combat_power_dispatch.go` is called once when a `CombatService` is created. This function handles both artifact behavior dispatch and perk dispatch, with artifacts registering first at each event point. The perk portion performs:

1. **PerkDispatcher injection** — `SquadPerkDispatcher` injected into `CombatActionSystem`. The dispatcher's 9 methods are called at specific points in the damage pipeline (see Section 3).
2. **Post-reset hook** — fires when a faction's turn begins (after artifact `OnPostReset`); calls `ResetPerkRoundStateTurn()` then `RunTurnStartHooks` for every squad in that faction.
3. **Turn-end hook** — fires when a round advances (after artifact `OnTurnEnd`); calls `ResetPerkRoundStateRound()` for every squad with a `PerkSlotTag`.
4. **Attack-complete hook** — fires after each successful attack exchange; updates `AttackedThisTurn` on the attacker and `WasAttackedThisTurn` on the defender.
5. **Move-complete hook** — fires when a squad finishes moving; sets `MovedThisTurn = true` and resets `TurnsStationary = 0`.

### HookContext

All `PerkBehavior` methods receive a `*HookContext` value that bundles parameters the behavior might need:

```go
type HookContext struct {
    AttackerID      ecs.EntityID
    DefenderID      ecs.EntityID
    AttackerSquadID ecs.EntityID
    DefenderSquadID ecs.EntityID
    SquadID         ecs.EntityID      // The squad that owns the perk (used by TurnStart, DeathOverride)
    UnitID          ecs.EntityID      // Specific unit (used by DeathOverride, DamageRedirect)
    RoundNumber     int               // Current round (used by TurnStart)
    DamageAmount    int               // Incoming damage (used by DamageRedirect)
    RoundState      *PerkRoundState   // always the squad that owns the perk
    Manager         *common.EntityManager
}
```

Not all fields are populated for every method — some are zero-valued depending on context. For example, `TurnStart` has no attacker/defender, and `DeathOverride` uses `SquadID`/`UnitID` instead. Combat-oriented methods (DamageMod, PostDamage, CoverMod) populate the attacker/defender fields via a shared `buildCombatContext` helper.

The `RoundState` pointer inside `HookContext` always belongs to the squad that owns the perk, not the opposing squad.

---

## 3. Hook Types Reference

All hook types are methods on the `PerkBehavior` interface. `BasePerkBehavior` provides no-op defaults for all methods — perks override only the ones they need.

### 3.1 AttackerDamageMod / DefenderDamageMod

```go
AttackerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers)
DefenderDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers)
```

**When it fires:** Inside `processAttack`, after hit and dodge rolls pass, before final damage is computed. `AttackerDamageMod` iterates the attacker's perks; `DefenderDamageMod` iterates the defender's perks.

**What it modifies:** `DamageModifiers` fields:

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
func (b *ExecutionersInstinctBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
    unitIDs := squadcore.GetUnitIDsInSquad(ctx.DefenderSquadID, ctx.Manager)
    for _, unitID := range unitIDs {
        attr := common.GetComponentTypeByID[*common.Attributes](ctx.Manager, unitID, common.AttributeComponent)
        if attr == nil || attr.CurrentHealth <= 0 {
            continue
        }
        maxHP := attr.GetMaxHealth()
        if maxHP > 0 && float64(attr.CurrentHealth)/float64(maxHP) < PerkBalance.ExecutionersInstinct.HPThreshold {
            modifiers.CritBonus += PerkBalance.ExecutionersInstinct.CritBonus
            logPerkActivation(PerkExecutionersInstinct, ctx.AttackerSquadID, "crit bonus vs wounded target")
            return
        }
    }
}
```

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
CounterMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) (skipCounter bool)
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

**When it fires:** In the post-reset hook, once per squad per faction turn, after `ResetPerkRoundStateTurn()` runs. Uses `ctx.SquadID`, `ctx.RoundNumber`, `ctx.RoundState`, `ctx.Manager`.

**Important ordering:** `ResetPerkRoundStateTurn()` runs before `TurnStart`. This means `TurnStart` sees the already-cleared per-turn fields, but the snapshot fields (`WasAttackedLastTurn`, `WasIdleLastTurn`) have already been saved.

**Perks that use it:** `field_medic`, `reckless_assault`, `fortify`, `resolute`, `counterpunch`, `deadshots_patience`.

---

### 3.6 DefenderCoverMod

```go
DefenderCoverMod(ctx *HookContext, coverBreakdown *combatcore.CoverBreakdown)
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
    PerkID() string
    AttackerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers)
    DefenderDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers)
    DefenderCoverMod(ctx *HookContext, coverBreakdown *combatcore.CoverBreakdown)
    TargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID
    CounterMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) (skipCounter bool)
    AttackerPostDamage(ctx *HookContext, damageDealt int, wasKill bool)
    DefenderPostDamage(ctx *HookContext, damageDealt int, wasKill bool)
    TurnStart(ctx *HookContext)
    DamageRedirect(ctx *HookContext) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)
    DeathOverride(ctx *HookContext) (preventDeath bool)
}
```

`BasePerkBehavior` provides no-op defaults for all methods. Perks embed it and override only the methods they need.

### Why the Attacker/Defender Split Exists

For `DamageModifiers` and `CoverBreakdown`, a perk's behavior is fundamentally different depending on whether the perk-owning squad is attacking or defending. Without the split, every method body would need a self-check like `if !HasPerk(ctx.AttackerSquadID, "my_perk", ctx.Manager) { return }`, which is redundant since the dispatcher already knows which squad owns the perk.

The split eliminates that boilerplate:

- `SquadPerkDispatcher.AttackerDamageMod` iterates the **attacker's** perk list and calls each behavior's `AttackerDamageMod`.
- `SquadPerkDispatcher.DefenderDamageMod` iterates the **defender's** perk list and calls each behavior's `DefenderDamageMod`.
- `SquadPerkDispatcher.CoverMod` always calls `DefenderCoverMod` because cover only applies to the defending unit.

### The Reckless Assault Two-Method Pattern

`RecklessAssaultBehavior` overrides three methods on the same struct:

```go
type RecklessAssaultBehavior struct{ BasePerkBehavior }

func (b *RecklessAssaultBehavior) PerkID() string { return PerkRecklessAssault }
func (b *RecklessAssaultBehavior) TurnStart(ctx *HookContext)         { /* resets vulnerability */ }
func (b *RecklessAssaultBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) { /* +30% damage, sets vulnerable */ }
func (b *RecklessAssaultBehavior) DefenderDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) { /* +20% incoming when vulnerable */ }
```

The `TurnStart` method resets vulnerability. The `AttackerDamageMod` fires when the squad attacks, boosting damage and setting `RecklessAssaultState{Vulnerable: true}`. The `DefenderDamageMod` fires when the squad is being attacked, increasing incoming damage if vulnerable.

All three methods use `ctx.RoundState` belonging to the perk-owning squad. A perk can override any combination of methods, or none (inheriting no-ops from `BasePerkBehavior`).

### Dispatcher Flow

`SquadPerkDispatcher` implements `PerkDispatcher` by iterating equipped behaviors:

```go
func (d *SquadPerkDispatcher) AttackerDamageMod(...) {
    ctx := buildCombatContext(attackerSquadID, ...)
    forEachPerkBehavior(attackerSquadID, manager, func(behavior PerkBehavior) bool {
        behavior.AttackerDamageMod(ctx, modifiers)
        return true
    })
}
```

No nil-checks on individual methods are needed — `BasePerkBehavior` provides no-ops. If a squad has no perks, `forEachPerkBehavior` iterates zero items and returns immediately.

---

## 5. PerkRoundState Lifecycle

`PerkRoundState` is an ECS component attached to each squad entity at the start of combat and removed when combat ends.

### State Fields by Lifetime

`PerkRoundState` has two layers: **shared tracking fields** that live directly on the struct (set by the dispatch layer, read by multiple perks), and **per-perk isolated state** stored in maps keyed by perk ID.

#### Shared Tracking Fields (direct struct fields)

**Per-turn** (reset by `ResetPerkRoundStateTurn()` before each squad's turn):

| Field | Type | Writer | Reader |
|---|---|---|---|
| `MovedThisTurn` | `bool` | `combat_power_dispatch.go` OnMoveComplete | Stalwart, Fortify |
| `AttackedThisTurn` | `bool` | `combat_power_dispatch.go` OnAttackComplete | ResetPerkRoundStateTurn (snapshot) |
| `WasAttackedThisTurn` | `bool` | `combat_power_dispatch.go` OnAttackComplete | ResetPerkRoundStateTurn (snapshot) |

**Cross-round but movement-dependent** (not reset by either method; modified explicitly):

| Field | Type | Writer | Reader |
|---|---|---|---|
| `TurnsStationary` | `int` | Fortify TurnStart + OnMoveComplete | Fortify DefenderCoverMod |

**Snapshot fields** (computed by `ResetPerkRoundStateTurn()` from prior-turn values, then read by `TurnStart` methods):

| Field | Type | Used By |
|---|---|---|
| `WasAttackedLastTurn` | `bool` | Counterpunch |
| `DidNotAttackLastTurn` | `bool` | Counterpunch |
| `WasIdleLastTurn` | `bool` | Deadshot's Patience |

#### Per-Perk Isolated State (PerkState / PerkBattleState maps)

Per-perk state lives in two `map[string]any` maps keyed by perk ID. Each perk defines its own small state struct in `components.go` and accesses it via generic helpers:

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
    |
    v
InitializeRoundState (system.go)
    Creates fresh PerkRoundState{} on each squad entity
    |
    v
-- ROUND LOOP --
    |
    v
ResetPerkRoundStateRound()
    PerkState = nil  (clears all per-perk round state maps)
    PerkBattleState is preserved
    |
    v
-- TURN LOOP (per faction) --
    |
    v
ResetPerkRoundStateTurn()
    Saves snapshots:
        WasAttackedLastTurn = WasAttackedThisTurn
        DidNotAttackLastTurn = !AttackedThisTurn
        WasIdleLastTurn = !MovedThisTurn && !AttackedThisTurn
    Clears per-turn shared fields:
        MovedThisTurn = false
        AttackedThisTurn = false
        WasAttackedThisTurn = false
    |
    v
RunTurnStartHooks (for each squad in faction)
    field_medic: heals lowest-HP unit
    fortify: increments or resets TurnsStationary
    resolute: snapshots current HP into ResoluteState.RoundStartHP
    counterpunch: evaluates snapshots -> sets CounterpunchState.Ready
    deadshots_patience: evaluates WasIdleLastTurn -> sets DeadshotState.Ready
    |
    v
-- ATTACK EXCHANGES --
    dispatcher.AttackerDamageMod / dispatcher.DefenderDamageMod
    dispatcher.CoverMod
    dispatcher.TargetOverride
    dispatcher.DamageRedirect
    dispatcher.DeathOverride
    dispatcher.AttackerPostDamage / dispatcher.DefenderPostDamage
    (attack-complete hook sets AttackedThisTurn, WasAttackedThisTurn)
    (move-complete hook sets MovedThisTurn, resets TurnsStationary)
    |
    v
    (next faction turn -> back to ResetPerkRoundStateTurn)
    |
    v
    (next round -> back to ResetPerkRoundStateRound)

Combat End
    |
    v
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

**Mutual exclusion pairs:** `cleave` <-> `precision_strike`, `reckless_assault` <-> `stalwart`, `bloodlust` <-> `field_medic`.

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

This section walks through adding a hypothetical new perk: **Iron Will** — a Tier 1 Defense perk for Tank and Support units that reduces all damage taken by 10% when the squad's current total HP is below 50% of its starting HP.

### Step 1: Add the JSON Definition

Open `assets/gamedata/perkdata.json` and append to the `"perks"` array:

```json
{
  "id": "iron_will",
  "name": "Iron Will",
  "description": "When the squad's total HP falls below 50% of starting HP, all units take 10% less damage.",
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
const PerkIronWill = "iron_will"
```

### Step 3: Create the Behavior Struct

Choose the appropriate behavior file:
- `behaviors_stateless.go` — pure functions of HookContext (no state)
- `behaviors_stateful_round.go` — uses `PerkState` (resets each round)
- `behaviors_stateful_battle.go` — uses `PerkBattleState` (persists entire combat)

Since Iron Will is stateless, add it to `behaviors_stateless.go`:

```go
type IronWillBehavior struct{ BasePerkBehavior }

func (b *IronWillBehavior) PerkID() string { return PerkIronWill }

func (b *IronWillBehavior) DefenderDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
    unitIDs := squadcore.GetUnitIDsInSquad(ctx.DefenderSquadID, ctx.Manager)
    totalCurrent := 0
    totalMax := 0
    for _, uid := range unitIDs {
        attr := common.GetComponentTypeByID[*common.Attributes](
            ctx.Manager, uid, common.AttributeComponent,
        )
        if attr == nil || attr.CurrentHealth <= 0 {
            continue
        }
        totalCurrent += attr.CurrentHealth
        totalMax += attr.GetMaxHealth()
    }
    if totalMax > 0 && float64(totalCurrent)/float64(totalMax) < 0.5 {
        modifiers.DamageMultiplier *= 0.90
        logPerkActivation(PerkIronWill, ctx.DefenderSquadID, "-10% damage (squad below 50% HP)")
    }
}
```

### Step 4: Register in init()

In the `init()` function at the top of the same behavior file:

```go
RegisterPerkBehavior(&IronWillBehavior{})
```

### Step 5: Add Per-Perk State (if needed)

Iron Will is stateless — no state struct needed. If your perk needs state, define a state struct in `components.go`:

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
3. On startup, `LoadPerkDefinitions()` calls `validateHookCoverage()`, which prints a warning for any ID that has a JSON entry but no registered behavior, or vice versa.

### Complete Worked Example Summary

| Step | File | What to Do |
|---|---|---|
| 1 | `assets/gamedata/perkdata.json` | Append JSON object |
| 2 | `tactical/powers/perks/perkids.go` | Add `PerkIronWill` constant |
| 3 | `tactical/powers/perks/behaviors_*.go` | Create behavior struct, embed `BasePerkBehavior`, override methods |
| 4 | `tactical/powers/perks/behaviors_*.go` | Add `RegisterPerkBehavior(&IronWillBehavior{})` in `init()` |
| 5 | `tactical/powers/perks/components.go` | Add per-perk state struct if needed |
| 6 | Terminal | Build and test |

---

## 8. Common Patterns

### Reading All Units in a Squad

```go
unitIDs := squadcore.GetUnitIDsInSquad(squadID, ctx.Manager)
for _, uid := range unitIDs {
    attr := common.GetComponentTypeByID[*common.Attributes](
        ctx.Manager, uid, common.AttributeComponent,
    )
    if attr == nil || attr.CurrentHealth <= 0 {
        continue // Skip dead units
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

### Checking Faction and Getting Faction Allies

```go
faction := combatcore.GetSquadFaction(squadID, ctx.Manager)
if faction == 0 {
    return
}
friendlySquads := combatcore.GetActiveSquadsForFaction(faction, ctx.Manager)
for _, friendlyID := range friendlySquads {
    if friendlyID == squadID {
        continue // Skip self
    }
    // ...
}
```

### Both-Sides Perks (Two Methods, Same Struct)

When a perk needs to fire on both attacker and defender turns, override both methods on the same struct:

```go
type MyPerkBehavior struct{ BasePerkBehavior }

func (b *MyPerkBehavior) PerkID() string { return PerkMyPerk }
func (b *MyPerkBehavior) AttackerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) { /* fires when squad is attacker */ }
func (b *MyPerkBehavior) DefenderDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) { /* fires when squad is defender */ }
```

Both methods receive `ctx.RoundState` pointing to their own squad's state. If the attacker method sets a flag and the defender method needs to read it, both are reading the same squad's `PerkRoundState`.

### Lazy State Initialization

Per-perk state structs that contain maps should use `GetOrInitPerkState` / `GetOrInitBattleState` to lazily initialize on first access:

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

When a perk has a meaningful effect during combat, it logs a perk-specific message via the `PerkLogger` system.

### Infrastructure (`hooks.go`)

```go
type PerkLogger func(perkID string, squadID ecs.EntityID, message string)

func SetPerkLogger(fn PerkLogger)
func logPerkActivation(perkID string, squadID ecs.EntityID, message string)
```

`SetPerkLogger` registers a callback. `logPerkActivation` is called from inside individual behavior methods (not at the dispatcher level) so that messages describe the actual outcome.

### Wiring

`combat_power_dispatch.go` registers a logger during combat setup:

```go
perks.SetPerkLogger(func(perkID string, squadID ecs.EntityID, message string) {
    fmt.Printf("[PERK] %s: %s (squad %d)\n", perkID, message, squadID)
})
```

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
| `tactical/powers/perks/hooks.go` | `PerkBehavior` interface; `BasePerkBehavior`; `behaviorRegistry`; `RegisterPerkBehavior`; `PerkLogger` infrastructure |
| `tactical/powers/perks/dispatcher.go` | `SquadPerkDispatcher` — implements `combattypes.PerkDispatcher` by iterating equipped behaviors |
| `tactical/powers/perks/behaviors_stateless.go` | Stateless perk behavior structs |
| `tactical/powers/perks/behaviors_stateful_round.go` | Per-round stateful perk behavior structs |
| `tactical/powers/perks/behaviors_stateful_battle.go` | Per-battle stateful perk behavior structs |
| `tactical/powers/perks/queries.go` | `HasPerk`, `GetRoundState`, `forEachPerkBehavior`, context builders |
| `tactical/powers/perks/registry.go` | `PerkDefinition`, `PerkRegistry`, `LoadPerkDefinitions`, `validateHookCoverage` |
| `tactical/powers/perks/balanceconfig.go` | `PerkBalanceConfig`, per-perk balance structs, `LoadPerkBalanceConfig` |
| `tactical/powers/perks/system.go` | `EquipPerk`, `UnequipPerk`, `InitializeRoundState`, `CleanupRoundState`, `ResetPerkRoundStateTurn`, `ResetPerkRoundStateRound`, `RunTurnStartHooks` |
| `tactical/powers/perks/init.go` | ECS subsystem `init()` — registers `PerkSlotComponent`, `PerkRoundStateComponent` |
| `tactical/combat/combattypes/perk_callbacks.go` | `PerkDispatcher` interface (no perks import) |
| `tactical/combat/combatservices/combat_power_dispatch.go` | Wires artifact behaviors and `SquadPerkDispatcher` into combat pipeline |
| `assets/gamedata/perkdata.json` | Static perk definitions loaded at startup |
| `assets/gamedata/perkbalanceconfig.json` | Per-perk balance tuning values |
