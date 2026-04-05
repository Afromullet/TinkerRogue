# Perk System

**Last Updated:** 2026-04-05
**Package:** `tactical/powers/perks`
**Related:** `tactical/combat/combattypes/perk_callbacks.go`, `tactical/combat/combatservices/perk_dispatch.go`

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Hook Types Reference](#3-hook-types-reference)
4. [PerkHooks Struct — Attacker vs Defender Slots](#4-perkhooks-struct--attacker-vs-defender-slots)
5. [PerkRoundState Lifecycle](#5-perkroundstate-lifecycle)
6. [All 24 Perks Reference Table](#6-all-24-perks-reference-table)
7. [How to Add a New Perk (Step-by-Step)](#7-how-to-add-a-new-perk-step-by-step)
8. [Common Patterns](#8-common-patterns)

---

## 1. Overview

Perks are squad-level passive abilities that modify combat behavior. A squad equips perks into a fixed number of slots, and those perks automatically fire during combat via an event-hook system — no polling, no manual checks scattered through attack code.

### How Perks Differ from Artifacts and Spells

| Dimension | Perks | Artifacts | Spells |
|---|---|---|---|
| Activation | Automatic — fires on hooks | Passive or charge-triggered | Explicit cast action |
| Scope | Squad-level passive modifiers | Per-unit item effects | Targeted abilities with costs |
| Persistence | Survives between combats | Survives between combats | Consumed or on cooldown |
| State | `PerkRoundState` (per-combat) | Artifact charge counts | Spell cooldown/cost |
| Definition | `perkdata.json` | `artifactdata.json` | Spell templates |

### The Hook-Based Architecture

The core design decision is that perks express themselves as callbacks registered against named event slots. When the combat engine reaches a decision point (before damage calculation, after a kill, at turn start, etc.) it calls a dispatcher that iterates every perk equipped on the relevant squad and fires any matching hook functions.

This approach means:

- Adding a new perk never requires touching existing combat code.
- Each perk is isolated; a broken perk cannot crash the calculation for another perk.
- The combat engine can run with zero perks (all callback pointers are nil-checked) and with full perks; the call sites are identical.

---

## 2. Architecture

### Package Layout

```
tactical/powers/perks/
├── components.go                -- ECS component structs: PerkSlotData, PerkRoundState, per-perk state structs
├── hooks.go                     -- Hook function type definitions + PerkHooks struct + hookRegistry
├── behaviors_stateless.go       -- Stateless perk behaviors (pure functions of HookContext)
├── behaviors_stateful_round.go  -- Per-round stateful perk behaviors (use PerkState map)
├── behaviors_stateful_battle.go -- Per-battle stateful perk behaviors (use PerkBattleState map)
├── queries.go                   -- HasPerk, GetRoundState, all RunXxx hook runner functions
├── registry.go                  -- PerkDefinition, PerkRegistry, LoadPerkDefinitions, validateHookCoverage
├── balanceconfig.go             -- PerkBalanceConfig, per-perk balance structs, loaded from JSON
├── system.go                    -- EquipPerk, UnequipPerk, InitializeRoundState, CleanupRoundState
└── init.go                      -- ECS subsystem registration (PerkSlotComponent, PerkRoundStateComponent)

tactical/combat/combattypes/
└── perk_callbacks.go -- PerkCallbacks struct and runner type aliases (no perks import)

tactical/combat/combatservices/
└── perk_dispatch.go  -- Wires perks.Run* functions into combattypes.PerkCallbacks

assets/gamedata/
├── perkdata.json           -- Static definitions: id, name, description, tier, category, roles
└── perkbalanceconfig.json  -- Per-perk balance tuning values
```

### The Circular Import Problem and the Bridge Layer

The combat engine (`combatcore`) needs to call perk logic during damage calculation. The perk logic (`perks`) needs to call combat helpers like `GetSquadFaction` and `GetActiveSquadsForFaction` that live in `combatcore`. If both packages import each other directly, the Go compiler refuses to build.

The solution is a three-layer bridge:

```
combattypes           (defines PerkCallbacks struct and runner type aliases — no perks import)
    |
combatservices        (imports combattypes, combatcore, and perks — wires them together)
    |
perks                 (imports combatcore for DamageModifiers, CoverBreakdown types)
```

`combattypes/perk_callbacks.go` defines function type aliases such as `DamageHookRunner` and `CoverHookRunner` that match the signatures of the `perks.RunXxx` functions, but the file itself never imports the `perks` package. `combatcore` imports `combattypes` to use `PerkCallbacks` in its function signatures. `combatservices/perk_dispatch.go` imports all three and performs direct function assignment:

```go
callbacks := &combattypes.PerkCallbacks{
    AttackerDamageMod: perks.RunAttackerDamageModHooks,
    DefenderDamageMod: perks.RunDefenderDamageModHooks,
    // ...
}
cs.CombatActSystem.SetPerkCallbacks(callbacks)
```

Because Go function values are first-class, the assignment is type-safe at compile time even though `combattypes` never names the `perks` package.

### Dispatch Wiring

`setupPerkDispatch` in `combatservices/perk_dispatch.go` is called once when a `CombatService` is created. It performs four registrations:

1. **Inline combat callbacks** — `PerkCallbacks` struct injected into `CombatActionSystem`.
2. **Post-reset hook** — fires when a faction's turn begins; calls `ResetPerTurn()` then `RunTurnStartHooks` for every squad in that faction.
3. **Turn-end hook** — fires when a round advances; calls `ResetPerRound()` for every squad with a `PerkSlotTag`.
4. **Attack-complete hook** — fires after each successful attack exchange; updates `AttackedThisTurn` on the attacker and `WasAttackedLastTurn` on the defender.
5. **Move-complete hook** — fires when a squad finishes moving; sets `MovedThisTurn = true` and resets `TurnsStationary = 0`.

### HookContext

All hook types receive a `*HookContext` value that bundles parameters the behavior might need:

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

Not all fields are populated for every hook type — some are zero-valued depending on context. For example, `TurnStart` hooks have no attacker/defender, and `DeathOverride` hooks use `SquadID`/`UnitID` instead. Combat-oriented hooks (DamageMod, PostDamage, CoverMod) populate the attacker/defender fields via a shared `buildCombatContext` helper.

The `RoundState` pointer inside `HookContext` always belongs to the squad that owns the hook, not the opposing squad. For hooks that need the opposing squad's state (e.g., `disruption` writing to the defender's state), the behavior calls `GetRoundState(ctx.DefenderSquadID, ctx.Manager)` explicitly.

---

## 3. Hook Types Reference

### 3.1 DamageModHook

```go
type DamageModHook func(ctx *HookContext, modifiers *combatcore.DamageModifiers)
```

**When it fires:** Inside `calculateDamage`, after hit and dodge rolls pass, before final damage is computed.

**What it modifies:** `DamageModifiers` fields:

| Field | Type | Meaning |
|---|---|---|
| `DamageMultiplier` | `float64` | Multiplicative damage scalar (default 1.0) |
| `HitPenalty` | `int` | Subtracted from attacker's hit threshold (positive = worse) |
| `CritBonus` | `int` | Added to crit threshold (positive = more crits) |
| `SkipCrit` | `bool` | If true, crit roll is skipped entirely |
| `CoverBonus` | `float64` | Additional cover fraction added after perk cover hooks |
| `IsCounterattack` | `bool` | Read-only flag; behaviors can branch on this |

**Split into attacker and defender slots:** See Section 4.

**Perks that use it:**
- Attacker slot: `reckless_assault`, `executioners_instinct`, `isolated_predator`, `opening_salvo`, `last_line`, `cleave` (damage penalty), `adaptive_armor` (defender version), `bloodlust`, `grudge_bearer`, `counterpunch`, `marked_for_death`, `deadshots_patience`
- Defender slot: `reckless_assault` (vulnerability), `shieldwall_discipline`, `vigilance`, `adaptive_armor`

**Example — Executioner's Instinct:**

```go
func executionerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
    unitIDs := squadcore.GetUnitIDsInSquad(ctx.DefenderSquadID, ctx.Manager)
    for _, unitID := range unitIDs {
        attr := common.GetComponentTypeByID[*common.Attributes](ctx.Manager, unitID, common.AttributeComponent)
        if attr == nil || attr.CurrentHealth <= 0 {
            continue
        }
        maxHP := attr.GetMaxHealth()
        if maxHP > 0 && float64(attr.CurrentHealth)/float64(maxHP) < 0.3 {
            modifiers.CritBonus += 25
            return
        }
    }
}
```

The hook scans defender units for any below 30% HP. If found, it adds 25 to `CritBonus` and returns immediately. The `return` is an optimization — finding one qualifying unit is sufficient.

---

### 3.2 TargetOverrideHook

```go
type TargetOverrideHook func(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID
```

**When it fires:** At the start of `processAttack`, before any per-target loop.

**What it does:** Receives the default target list from `SelectTargetUnits` and returns a replacement list. The returned slice becomes the iteration set for damage calculation.

**Perks that use it:** `cleave` (adds rear-row targets), `precision_strike` (replaces target with lowest-HP enemy).

**Important constraint:** The hook must return a valid slice even when it decides not to modify targets. Returning `defaultTargets` unchanged is always safe.

**Example — Cleave:**

```go
func cleaveTargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID {
    // Only fires for melee-row attack type
    targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](
        ctx.Manager, ctx.AttackerID, squadcore.TargetRowComponent,
    )
    if targetData == nil || targetData.AttackType != unitdefs.AttackTypeMeleeRow {
        return defaultTargets
    }
    // ... appends units from the row behind the primary target
}
```

Note that `cleave` also registers an `AttackerDamageMod` to apply the -30% damage penalty. Both hooks fire; the target override widens the hit set, and the damage modifier reduces the per-hit power.

---

### 3.3 CounterModHook

```go
type CounterModHook func(ctx *HookContext, modifiers *combatcore.DamageModifiers) (skipCounter bool)
```

**When it fires:** In `ExecuteAttackAction`, after the main attack resolves, before the counterattack loop begins.

**What it does:** Can modify `counterModifiers` in place (changing the counterattack's damage or hit penalty) or return `true` to suppress the counterattack entirely.

**Perks that use it:** `stalwart` (upgrades counter damage to 100%), `riposte` (removes counter hit penalty).

**Example — Stalwart:**

```go
func stalwartCounterMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) bool {
    if !ctx.RoundState.MovedThisTurn {
        modifiers.DamageMultiplier = 1.0 // Override 0.5 default
    }
    return false // Never suppress the counter
}
```

`Stalwart` reads `ctx.RoundState.MovedThisTurn` to determine whether the squad is eligible. It assigns `1.0` directly rather than multiplying; this intentionally overrides whatever the base counterattack multiplier was set to (`0.5`).

---

### 3.4 PostDamageHook

```go
type PostDamageHook func(ctx *HookContext, damageDealt int, wasKill bool)
```

**When it fires:** In `processAttack`, after `recordDamageToUnit` returns and before the `DeathOverride` check.

**What it does:** Reacts to damage that was just recorded. Common uses are tracking kill counts (`bloodlust`), flagging that the defender was hit (`disruption`), and accumulating revenge stacks (`grudge_bearer`).

**Two dispatch calls per attack:** The runner in `queries.go` has both `RunAttackerPostDamageHooks` and `RunDefenderPostDamageHooks`. Both are called in `processAttack` as `PostDamage` and `DefenderPostDamage` respectively. A perk can register the same function in both slots if it needs to react regardless of role.

**Perks that use it:** `disruption` (writes to cross-squad state), `bloodlust` (increments kill counter), `grudge_bearer` (increments grudge stack on defender's state).

---

### 3.5 TurnStartHook

```go
type TurnStartHook func(ctx *HookContext)
```

**When it fires:** In the post-reset hook registered in `setupPerkDispatch`, once per squad per faction turn, after `ResetPerTurn()` runs. Uses `ctx.SquadID`, `ctx.RoundNumber`, `ctx.RoundState`, `ctx.Manager`.

**What it does:** Evaluates prior-turn state to set up current-turn bonuses, performs per-turn healing, or transitions multi-turn state machines.

**Important ordering:** `ResetPerTurn()` runs before `TurnStartHooks`. This means `TurnStartHooks` see the already-cleared per-turn fields, but the snapshot fields (`WasAttackedLastTurn`, `WasIdleLastTurn`) have already been saved from the previous values.

**Perks that use it:** `field_medic`, `overwatch`, `fortify`, `resolute`, `counterpunch`, `deadshots_patience`.

**Example — Counterpunch:**

```go
func counterpunchTurnStart(ctx *HookContext) {
    if ctx.RoundState.WasAttackedLastTurn && ctx.RoundState.DidNotAttackLastTurn {
        state := &CounterpunchState{Ready: true}
        SetPerkState(ctx.RoundState, "counterpunch", state)
    }
}
```

The hook reads the snapshot fields set by `ResetPerTurn()` and arms the counterpunch state. The companion `counterpunchDamageMod` then consumes that flag when the squad attacks.

---

### 3.6 CoverModHook

```go
type CoverModHook func(ctx *HookContext, coverBreakdown *combatcore.CoverBreakdown)
```

**When it fires:** In `calculateDamage`, after the base cover breakdown is computed by `CalculateCoverBreakdown`, before the cover reduction is applied to damage.

**What it modifies:** `CoverBreakdown.TotalReduction` — a float64 in `[0.0, 1.0]` representing the fraction of damage blocked by cover. Behaviors must clamp to 1.0 after adding their bonus.

**Only defender perks use this slot.** See Section 4 for the reason.

**Perks that use it:** `brace_for_impact` (flat +15%), `fortify` (up to +15% based on stationary turns).

---

### 3.7 DamageRedirectHook

```go
type DamageRedirectHook func(ctx *HookContext) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)
```

**When it fires:** This hook type is defined in `hooks.go` and appears in `PerkHooks`. Uses `ctx.UnitID` (defender unit), `ctx.SquadID` (defender squad), `ctx.DamageAmount`.

**What it does:** Intercepts damage before it is recorded. Returns three values: the reduced damage that should be applied to the original target, the ID of the redirect target (a tank in a guardian squad), and the redirected amount. Returning `(damageAmount, 0, 0)` means no redirect.

**Perks that use it:** `guardian_protocol`.

**Example — Guardian Protocol:**

`guardianDamageRedirect` finds adjacent friendly squads that also have `guardian_protocol` equipped. If it finds a living Tank in such a squad, it redirects 25% of the incoming damage to that tank:

```go
guardianDmg := damageAmount / 4
remainingDmg := damageAmount - guardianDmg
return remainingDmg, unitID, guardianDmg
```

The caller applies `remainingDmg` to the original target and `guardianDmg` to `unitID`.

---

### 3.8 DeathOverrideHook

```go
type DeathOverrideHook func(ctx *HookContext) (preventDeath bool)
```

**When it fires:** In `processAttack`, after `WasKilled` is set to `true` on an attack event, before the event is appended to the log. Uses `ctx.UnitID`, `ctx.SquadID`, `ctx.RoundState`, `ctx.Manager`.

**What it does:** If it returns `true`, the combat engine retroactively reduces the recorded damage so the unit survives at exactly 1 HP and removes the unit from `result.UnitsKilled`.

**Perks that use it:** `resolute`.

**Example — Resolute:**

```go
func resoluteDeathOverride(ctx *HookContext) bool {
    state := GetBattleState[*ResoluteState](ctx.RoundState, "resolute")
    if state == nil {
        return false
    }
    if state.Used[ctx.UnitID] {
        return false  // Already spent for this unit
    }
    roundStartHP := state.RoundStartHP[ctx.UnitID]
    attr := common.GetComponentTypeByID[*common.Attributes](ctx.Manager, ctx.UnitID, common.AttributeComponent)
    if attr == nil {
        return false
    }
    maxHP := attr.GetMaxHealth()
    if maxHP > 0 && float64(roundStartHP)/float64(maxHP) > PerkBalance.Resolute.HPThreshold {
        state.Used[ctx.UnitID] = true
        return true  // Prevent death
    }
    return false
}
```

`ResoluteState.Used` is stored in `PerkBattleState` (persists entire combat), so the save is truly once per unit per combat. `RoundStartHP` is snapshotted each round by the companion `TurnStartHook`, so the HP threshold check is against the unit's HP at the beginning of the current round, not at the time of the killing blow.

---

## 4. PerkHooks Struct — Attacker vs Defender Slots

`PerkHooks` is the struct registered in `hookRegistry` for each perk ID:

```go
type PerkHooks struct {
    State              StateRequirements  // Declares state dependencies (zero value = stateless)
    AttackerDamageMod  DamageModHook      // runs only when this squad is the attacker
    DefenderDamageMod  DamageModHook      // runs only when this squad is the defender
    DefenderCoverMod   CoverModHook       // runs only when this squad is the defender
    TargetOverride     TargetOverrideHook
    CounterMod         CounterModHook
    AttackerPostDamage PostDamageHook     // runs only when this squad is the attacker
    DefenderPostDamage PostDamageHook     // runs only when this squad is the defender
    TurnStart          TurnStartHook
    DamageRedirect     DamageRedirectHook
    DeathOverride      DeathOverrideHook
}
```

### Why the Attacker/Defender Split Exists

For `DamageModHook` and `CoverModHook`, a perk's behavior is fundamentally different depending on whether the perk-owning squad is attacking or defending. Without the split, every behavior function would need to open with a call like `if !perks.HasPerk(ctx.AttackerSquadID, "my_perk", ctx.Manager) { return }`, which is redundant since the dispatcher already knows which squad owns the hook.

The split eliminates that boilerplate:

- `RunAttackerDamageModHooks` iterates the **attacker's** perk list and calls `AttackerDamageMod`.
- `RunDefenderDamageModHooks` iterates the **defender's** perk list and calls `DefenderDamageMod`.
- `RunCoverModHooks` always calls `DefenderCoverMod` because cover only applies to the defending unit.

### The Reckless Assault Two-Slot Pattern

`reckless_assault` is the clearest demonstration of why both slots exist for the same perk:

```go
RegisterPerkHooks("reckless_assault", &PerkHooks{
    AttackerDamageMod: recklessAssaultAttackerMod,  // +30% damage when attacking
    DefenderDamageMod: recklessAssaultDefenderMod,  // +20% damage taken when defending
})
```

The attacker hook fires when the squad is attacking, boosting outgoing damage and setting `RecklessVulnerable = true` on the squad's own `RoundState`. The defender hook fires when the squad is being attacked, checking whether `RecklessVulnerable` is set and increasing incoming damage accordingly.

Both hooks belong to the same perk but fire at different times with different `RoundState` contexts. The attacker hook's `ctx.RoundState` is the attacker's state; the defender hook's `ctx.RoundState` is the defender's state.

### Dispatcher Pseudocode

Combat-oriented hooks (DamageMod, PostDamage, CoverMod) use a shared `buildCombatContext` helper that populates attacker/defender fields:

```
// In RunAttackerDamageModHooks:
ctx = buildCombatContext(attackerSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID)
    // internally: buildHookContext(ownerSquadID) + sets attacker/defender fields
for each perkID in attacker's PerkSlotData.PerkIDs:
    hooks = GetPerkHooks(perkID)
    if hooks.AttackerDamageMod != nil:
        hooks.AttackerDamageMod(ctx, modifiers)

// In RunDefenderDamageModHooks:
ctx = buildCombatContext(defenderSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID)
for each perkID in defender's PerkSlotData.PerkIDs:
    hooks = GetPerkHooks(perkID)
    if hooks.DefenderDamageMod != nil:
        hooks.DefenderDamageMod(ctx, modifiers)
```

A perk can populate only one slot, both slots, or neither (for a perk that only uses `TurnStart`). Nil slots are skipped cheaply.

---

## 5. PerkRoundState Lifecycle

`PerkRoundState` is an ECS component attached to each squad entity at the start of combat and removed when combat ends.

### State Fields by Lifetime

`PerkRoundState` has two layers: **shared tracking fields** that live directly on the struct (set by the dispatch layer, read by multiple perks), and **per-perk isolated state** stored in maps keyed by perk ID.

#### Shared Tracking Fields (direct struct fields)

**Per-turn** (reset by `ResetPerTurn()` before each squad's turn):

| Field | Type | Writer | Reader |
|---|---|---|---|
| `MovedThisTurn` | `bool` | `perk_dispatch.go` OnMoveComplete | Stalwart, Fortify |
| `AttackedThisTurn` | `bool` | `perk_dispatch.go` OnAttackComplete | ResetPerTurn (snapshot) |
| `RecklessVulnerable` | `bool` | Reckless Assault (AttackerDamageMod) | Reckless Assault (DefenderMod) |

**Cross-round but movement-dependent** (not reset by either method; modified explicitly):

| Field | Type | Writer | Reader |
|---|---|---|---|
| `TurnsStationary` | `int` | Fortify TurnStart + OnMoveComplete | Fortify CoverMod |

**Snapshot fields** (computed by `ResetPerTurn()` from prior-turn values, then read by `TurnStartHooks`):

| Field | Type | Used By |
|---|---|---|
| `WasAttackedLastTurn` | `bool` | Counterpunch |
| `DidNotAttackLastTurn` | `bool` | Counterpunch |
| `WasIdleLastTurn` | `bool` | Deadshot's Patience |

#### Per-Perk Isolated State (PerkState / PerkBattleState maps)

Per-perk state lives in two `map[string]interface{}` maps keyed by perk ID. Each perk defines its own small state struct in `components.go` and accesses it via generic helpers:

```go
// Round state (cleared by ResetPerRound)
state := GetPerkState[*BloodlustState](ctx.RoundState, "bloodlust")
SetPerkState(ctx.RoundState, "bloodlust", &BloodlustState{KillsThisRound: 1})

// Battle state (persists entire combat, never reset)
state := GetBattleState[*ResoluteState](ctx.RoundState, "resolute")
state := GetOrInitBattleState(ctx.RoundState, "resolute", func() *ResoluteState { ... })
```

**Per-round state structs** (stored in `PerkState`, cleared by `ResetPerRound`):

| Struct | Fields | Used By |
|---|---|---|
| `AdaptiveArmorState` | `AttackedBy map[EntityID]int` | Adaptive Armor |
| `BloodlustState` | `KillsThisRound int` | Bloodlust |
| `DisruptionState` | `Targets map[EntityID]bool` | Disruption |
| `CounterpunchState` | `Ready bool` | Counterpunch |
| `DeadshotState` | `Ready bool` | Deadshot's Patience |
| `OverwatchState` | `Active bool` | Overwatch (placeholder) |

**Per-battle state structs** (stored in `PerkBattleState`, persist entire combat):

| Struct | Fields | Used By |
|---|---|---|
| `OpeningSalvoState` | `HasAttackedThisCombat bool` | Opening Salvo |
| `ResoluteState` | `Used map[EntityID]bool`, `RoundStartHP map[EntityID]int` | Resolute |
| `GrudgeBearerState` | `Stacks map[EntityID]int` | Grudge Bearer |
| `MarkedForDeathState` | `MarkedSquad EntityID` | Marked for Death |

This design prevents `PerkRoundState` from growing a new field for every stateful perk. Adding a new stateful perk means adding a new state struct — no changes to reset methods or shared state.

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
ResetPerRound()
    PerkState = nil  (clears all per-perk round state maps)
    PerkBattleState is preserved
    |
    v
-- TURN LOOP (per faction) --
    |
    v
ResetPerTurn()
    Saves snapshots:
        WasAttackedLastTurn = RecklessVulnerable || WasAttackedLastTurn
        DidNotAttackLastTurn = !AttackedThisTurn
        WasIdleLastTurn = !MovedThisTurn && !AttackedThisTurn
    Clears per-turn shared fields:
        MovedThisTurn = false
        AttackedThisTurn = false
        RecklessVulnerable = false
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
    RunAttackerDamageModHooks
    RunDefenderDamageModHooks
    RunCoverModHooks
    RunTargetOverrideHooks
    RunAttackerPostDamageHooks / RunDefenderPostDamageHooks
    RunDeathOverrideHooks
    (attack-complete hook sets AttackedThisTurn, WasAttackedLastTurn)
    (move-complete hook sets MovedThisTurn, resets TurnsStationary)
    |
    v
    (next faction turn -> back to ResetPerTurn)
    |
    v
    (next round -> back to ResetPerRound)

Combat End
    |
    v
CleanupRoundState (system.go)
    Removes PerkRoundStateComponent from each squad entity
```

### Per-Perk State Access

Per-perk state is stored in `PerkState` (round) and `PerkBattleState` (battle) maps, accessed via generic helpers. Use `GetOrInitPerkState` / `GetOrInitBattleState` when the state needs to be created on first access:

```go
state := GetOrInitBattleState(ctx.RoundState, "resolute", func() *ResoluteState {
    return &ResoluteState{
        Used:         make(map[ecs.EntityID]bool),
        RoundStartHP: make(map[ecs.EntityID]int),
    }
})
```

The `PerkState` map is set to `nil` by `ResetPerRound()`, which releases all per-perk round state at once. `PerkBattleState` persists the entire combat and is cleaned up by `CleanupRoundState`.

---

## 6. All 24 Perks Reference Table

Tier values: 0 = Combat Conditioning, 1 = Combat Specialization.
Category values: 0 = Offense, 1 = Defense, 2 = Tactical, 3 = Reactive, 4 = Doctrine.

| ID | Name | Tier | Category | Roles | Cost | Hook Types | Description |
|---|---|---|---|---|---|---|---|
| `brace_for_impact` | Brace for Impact | 0 | Defense | Tank | 2 | DefenderCoverMod | +15% cover bonus when defending |
| `reckless_assault` | Reckless Assault | 0 | Offense | DPS | 2 | AttackerDamageMod, DefenderDamageMod | +30% damage dealt; +20% damage received until next turn |
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
| `disruption` | Disruption | 1 | Reactive | DPS, Support | 3 | PostDamage | Dealing damage marks target squad; their next attack is debuffed -15% this round |
| `guardian_protocol` | Guardian Protocol | 1 | Defense | Tank | 4 | DamageRedirect | When adjacent friendly squad is attacked, one Tank absorbs 25% of damage |
| `overwatch` | Overwatch | 1 | Tactical | DPS | 4 | TurnStart | Skip attack to auto-attack at 75% damage next enemy that moves in range (v1 placeholder) |
| `adaptive_armor` | Adaptive Armor | 1 | Defense | Tank | 3 | DefenderDamageMod | -10% damage from same attacker per hit (stacks to 30%, resets each round) |
| `bloodlust` | Bloodlust | 1 | Offense | DPS | 3 | PostDamage, AttackerDamageMod | Each unit kill this round grants +15% damage on the next attack (stacks, resets per round) |
| `fortify` | Fortify | 1 | Defense | Tank, Support | 3 | TurnStart, DefenderCoverMod | +5% cover per consecutive stationary turn (max +15% after 3 turns; moving resets) |
| `precision_strike` | Precision Strike | 1 | Tactical | DPS | 3 | TargetOverride | Highest-dex DPS unit redirects to lowest-HP enemy instead of normal targeting |
| `resolute` | Resolute | 1 | Defense | Tank, DPS, Support | 4 | TurnStart, DeathOverride | A unit survives a killing blow at 1 HP if it had >50% HP at round start (once per unit per battle) |
| `grudge_bearer` | Grudge Bearer | 1 | Reactive | DPS, Tank | 3 | PostDamage, AttackerDamageMod | +20% damage vs squads that have damaged this squad (stacks to +40%) |
| `counterpunch` | Counterpunch | 1 | Reactive | DPS, Tank | 3 | TurnStart, AttackerDamageMod | If attacked last turn AND did not attack last turn, next attack deals +40% damage |
| `marked_for_death` | Marked for Death | 1 | Tactical | DPS, Support | 3 | AttackerDamageMod | Spend attack to mark an enemy; marked enemy takes +25% from next friendly attack |
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

**Field reference:**
- `"tier"`: 0 for Combat Conditioning, 1 for Combat Specialization.
- `"category"`: 0 Offense, 1 Defense, 2 Tactical, 3 Reactive, 4 Doctrine.
- `"exclusiveWith"`: list of perk IDs that cannot be equipped alongside this one. Must be symmetric — if A excludes B, B must also exclude A (validated on load).

### Step 2: Write the Behavior Function

Choose the appropriate behavior file based on state requirements:
- `behaviors_stateless.go` — pure functions of HookContext (no state read/written)
- `behaviors_stateful_round.go` — uses `PerkState` (resets each round)
- `behaviors_stateful_battle.go` — uses `PerkBattleState` (persists entire combat)

Since Iron Will is stateless, add it to `behaviors_stateless.go`:

```go
// Iron Will: -10% damage taken when squad HP falls below 50%
func ironWillDefenderMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
    // Compute total current HP vs total max HP for the squad
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
        modifiers.DamageMultiplier *= 0.90 // -10% damage
    }
}
```

**Design notes:**
- The function uses `ctx.DefenderSquadID` to look up the squad's own units, because this is a `DefenderDamageMod`. The defender is the squad under attack.
- It uses `GetComponentTypeByID` (by ID) rather than `GetComponentType` (by entity pointer) because it only has unit IDs, not entity pointers. Either pattern is correct; use `GetComponentType` only when you already hold the entity.
- It skips dead units (`CurrentHealth <= 0`) so they do not inflate the max total.

### Step 3: Register the Hook in behaviors.go init()

In the `init()` function at the top of the chosen behavior file, add the registration:

```go
RegisterPerkHooks("iron_will", &PerkHooks{DefenderDamageMod: ironWillDefenderMod})
```

Each behavior file has its own `init()` that registers hooks for the perks in that file. The `init()` in behavior files is separate from the ECS subsystem `init()` in `init.go`.

### Step 4: Add PerkRoundState Fields (if the perk needs state)

`Iron Will` is stateless — it reads live HP and applies a modifier without remembering anything between hooks. No state fields are required.

If your perk needs state, define a state struct in `components.go` and choose the appropriate storage:

```go
// Per-round state (resets each round via ResetPerRound)
type IronWillState struct {
    TriggeredThisRound bool
}
// Access: GetPerkState[*IronWillState](ctx.RoundState, "iron_will")
// Store:  SetPerkState(ctx.RoundState, "iron_will", &IronWillState{...})

// Per-battle state (persists entire combat)
// Access: GetBattleState[*IronWillState](ctx.RoundState, "iron_will")
// Store:  SetBattleState(ctx.RoundState, "iron_will", &IronWillState{...})
```

Per-perk state is automatically cleaned up: `ResetPerRound()` sets `PerkState = nil`, and `CleanupRoundState` removes the entire component at combat end. No manual reset code needed.

For state that needs to survive the per-turn reset but tracks prior-turn values (like `WasAttackedLastTurn`), use the shared tracking fields on `PerkRoundState` directly and snapshot inside `ResetPerTurn()`.

### Step 5: Verify

1. Run `go build -o game_main/game_main.exe game_main/*.go` — confirms no compile errors.
2. Run `go test ./tactical/perks/...` to run any existing perk tests.
3. On startup, `LoadPerkDefinitions()` calls `validateHookCoverage()`, which prints a warning for any ID that has a JSON entry but no hook registration, or vice versa. Check the console for:
   ```
   Loaded 25 perk definitions
   ```
   with no WARNING lines about `iron_will`.

**Verifying mutual exclusivity (if used):** If you set `"exclusiveWith"` in the JSON, `LoadPerkDefinitions` validates symmetry and prints warnings for asymmetric pairs.

### Complete Worked Example Summary

| Step | File | What to Do |
|---|---|---|
| 1 | `assets/gamedata/perkdata.json` | Append JSON object with all required fields |
| 2 | `tactical/powers/perks/behaviors_*.go` | Write behavior function(s) in the appropriate file |
| 3 | `tactical/powers/perks/behaviors_*.go` | Add `RegisterPerkHooks("iron_will", &PerkHooks{...})` in `init()` |
| 4 | `tactical/powers/perks/components.go` | Add per-perk state struct if needed |
| 5 | Terminal | Build and run; watch for WARNING lines on load |

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

Use `GetComponentTypeByID` when you only have an `ecs.EntityID`. Use `GetComponentType` when you already hold `*ecs.Entity` (e.g., from a query result). Never store entity pointers between calls.

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

The valid role constants are `unitdefs.RoleTank`, `unitdefs.RoleDPS`, and `unitdefs.RoleSupport`.

### Checking a Unit's Grid Position (Front Row, etc.)

```go
gridPos := common.GetComponentType[*squadcore.GridPositionData](entity, squadcore.GridPositionComponent)
if gridPos != nil && gridPos.AnchorRow == 0 {
    // Unit is in the front row
}
```

`AnchorRow` is zero-indexed. Row 0 is the front row, row 2 is the back row.

### Checking Faction and Getting Faction Allies

```go
faction := combatcore.GetSquadFaction(squadID, ctx.Manager)
if faction == 0 {
    return // No faction — bail out
}
friendlySquads := combatcore.GetActiveSquadsForFaction(faction, ctx.Manager)
for _, friendlyID := range friendlySquads {
    if friendlyID == squadID {
        continue // Skip self
    }
    // ...
}
```

`GetSquadFaction` and `GetActiveSquadsForFaction` are in `combatcore`. The `perks` package can call them because `perks` imports `combatcore` (not the other direction).

### Getting Map Position for Spatial Checks

```go
pos := common.GetComponentTypeByID[*coords.LogicalPosition](
    ctx.Manager, squadID, common.PositionComponent,
)
if pos == nil {
    return
}
// Chebyshev distance to another position
dist := pos.ChebyshevDistance(otherPos)
if dist <= 3 {
    // Within range
}
```

### Reading PerkRoundState for Another Squad

A perk's `HookContext.RoundState` belongs to the hook-owning squad. To read another squad's state:

```go
otherState := perks.GetRoundState(otherSquadID, ctx.Manager)
if otherState != nil {
    // Read or write otherState fields
}
```

`GetRoundState` returns nil if the squad has no `PerkRoundStateComponent` (i.e., no perks). Always nil-check.

### Writing Cross-Squad State (the Disruption Pattern)

`disruption` writes into the defender's state from the attacker's `PostDamageHook`:

```go
func disruptionPostDamage(ctx *HookContext, damageDealt int, wasKill bool) {
    if damageDealt <= 0 {
        return
    }
    // Mark the defender in the attacker's own per-perk state
    state := GetOrInitPerkState(ctx.RoundState, "disruption", func() *DisruptionState {
        return &DisruptionState{Targets: make(map[ecs.EntityID]bool)}
    })
    state.Targets[ctx.DefenderSquadID] = true

    // Also mark in the defender's state so the defender's damage mod can read it
    defenderState := GetRoundState(ctx.DefenderSquadID, ctx.Manager)
    if defenderState != nil {
        dState := GetOrInitPerkState(defenderState, "disruption", func() *DisruptionState {
            return &DisruptionState{Targets: make(map[ecs.EntityID]bool)}
        })
        dState.Targets[ctx.AttackerSquadID] = true
    }
}
```

The pattern: write into `ctx.RoundState`'s per-perk state for attacker-side tracking, and call `GetRoundState` for the other squad when the effect needs to be detectable from the other side.

### Both-Sides Perks (Two Hooks, Same Perk)

When a perk needs to fire on both attacker and defender turns:

```go
RegisterPerkHooks("my_perk", &PerkHooks{
    AttackerDamageMod: myPerkAttackerFunc,  // fires when squad is attacker
    DefenderDamageMod: myPerkDefenderFunc,  // fires when squad is defender
})
```

Both functions will receive `ctx.RoundState` pointing to their own squad's state. If the attacker function sets a flag and the defender function needs to read it, both are reading the same squad's `PerkRoundState`, so no cross-squad lookup is needed.

### Lazy State Initialization Pattern

Per-perk state structs that contain maps should use `GetOrInitPerkState` / `GetOrInitBattleState` to lazily initialize on first access:

```go
state := GetOrInitPerkState(ctx.RoundState, "adaptive_armor", func() *AdaptiveArmorState {
    return &AdaptiveArmorState{AttackedBy: make(map[ecs.EntityID]int)}
})
state.AttackedBy[attackerID]++
```

`ResetPerRound()` sets `PerkState = nil`, which releases all per-perk round state at once. `PerkBattleState` is never reset during combat.

### Checking HasPerk from Inside a Behavior

You should rarely need this — the attacker/defender slot split removes most such needs. However, `guardian_protocol` legitimately needs to check whether a neighboring squad also has the perk:

```go
if !HasPerk(friendlyID, "guardian_protocol", manager) {
    continue
}
```

`HasPerk` is exported and lives in `queries.go`. Within behavior functions, `manager` comes from the raw parameters (for hooks that receive it directly) or from `ctx.Manager`.

---

## Appendix: File Quick Reference

| File | Purpose |
|---|---|
| `tactical/powers/perks/components.go` | `PerkSlotData`, `PerkRoundState`, per-perk state structs, generic state accessors |
| `tactical/powers/perks/hooks.go` | Hook function type definitions; `PerkHooks` struct; `hookRegistry` map; `RegisterPerkHooks` |
| `tactical/powers/perks/behaviors_stateless.go` | Stateless perk behaviors (pure functions of HookContext) |
| `tactical/powers/perks/behaviors_stateful_round.go` | Per-round stateful perk behaviors (use PerkState map) |
| `tactical/powers/perks/behaviors_stateful_battle.go` | Per-battle stateful perk behaviors (use PerkBattleState map) |
| `tactical/powers/perks/queries.go` | `HasPerk`, `GetRoundState`, all `RunXxx` dispatcher functions |
| `tactical/powers/perks/registry.go` | `PerkDefinition`, `PerkRegistry`, `LoadPerkDefinitions`, `validateHookCoverage` |
| `tactical/powers/perks/balanceconfig.go` | `PerkBalanceConfig`, per-perk balance structs, `LoadPerkBalanceConfig` |
| `tactical/powers/perks/system.go` | `EquipPerk`, `UnequipPerk`, `InitializeRoundState`, `CleanupRoundState` |
| `tactical/powers/perks/init.go` | ECS subsystem `init()` — registers `PerkSlotComponent`, `PerkRoundStateComponent` |
| `tactical/combat/combattypes/perk_callbacks.go` | `PerkCallbacks` struct and runner type aliases (no perks import) |
| `tactical/combat/combatservices/perk_dispatch.go` | Wires `perks.RunXxx` into `combattypes.PerkCallbacks`; registers lifecycle hooks |
| `assets/gamedata/perkdata.json` | Static perk definitions loaded at startup |
| `assets/gamedata/perkbalanceconfig.json` | Per-perk balance tuning values |
