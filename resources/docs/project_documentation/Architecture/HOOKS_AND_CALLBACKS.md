# Hooks and Callbacks in TinkerRogue

**Last Updated:** 2026-04-21

This document is a reference for every hook, callback, and event-driven pattern in the codebase. It covers the perk hook system, the shared `PowerPipeline` used by `CombatService` to fan combat lifecycle events out to artifact and perk dispatchers, the `PerkDispatcher` interface that bridges `combatcore` and `perks`, `TurnManager` single-subscriber hooks, artifact behavior dispatch, the encounter post-combat callback, ECS subsystem self-registration, map generator registration, GUI callback patterns, and the overworld event log.

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [System Overview](#2-system-overview)
3. [Perk Hook System](#3-perk-hook-system)
4. [Power Pipeline and Combat Service Callbacks](#4-power-pipeline-and-combat-service-callbacks)
5. [Perk Dispatch Bridge](#5-perk-dispatch-bridge)
6. [TurnManager Callbacks](#6-turnmanager-callbacks)
7. [Artifact Behavior System](#7-artifact-behavior-system)
8. [Encounter Post-Combat Callback](#8-encounter-post-combat-callback)
9. [ECS Subsystem Self-Registration](#9-ecs-subsystem-self-registration)
10. [Map Generator Registry](#10-map-generator-registry)
11. [GUI Callback Patterns](#11-gui-callback-patterns)
12. [Overworld Event System](#12-overworld-event-system)
13. [Combat Round Hook Execution Order](#13-combat-round-hook-execution-order)
14. [Quick Reference Table](#14-quick-reference-table)

---

## 1. Introduction

TinkerRogue uses four distinct callback philosophies depending on the subsystem:

**Hook tables with a unified context struct** — The perk system defines a single `HookContext` that all ten hook slots receive. `HookContext` embeds `powercore.PowerContext` so the shared runtime dependencies (entity manager, query cache, round number, logger) live in exactly one place. This eliminates parameter drift between hook types and avoids making each behavior check whether it is the correct side of a combat exchange (the dispatcher methods handle side selection before invoking hooks).

**Ordered pipeline of typed subscribers** — `CombatService` owns a single `powercore.PowerPipeline` with four event families (post-reset, attack-complete, turn-end, move-complete). Subscribers register once at construction time in a declared order: **artifact dispatcher → perk dispatcher → GUI callback**. When a combat subsystem fires an event, the pipeline invokes every registered handler in order. This replaces the older pattern of four `[]func` slices on `CombatService`.

**Single-subscriber function fields** — `TurnManager`, `CombatActionSystem`, and `CombatMovementSystem` each store a single callback field (e.g., `onTurnEnd`, `onAttackComplete`). Only `CombatService` writes these, and it wires them to `pipeline.Fire*` so the combat core never imports perks, artifacts, or the GUI.

**Interface + registry dispatch** — Artifacts implement `ArtifactBehavior`. The registry holds all registered behaviors. `ArtifactDispatcher` owns the iteration logic; it calls every registered behavior in deterministic key order. Concrete behaviors embed `BaseBehavior` to opt out of events they do not need. Perks use an analogous pattern: `PerkBehavior` implementations live in `tactical/powers/perks/behaviors.go`; `SquadPerkDispatcher` implements the `combattypes.PerkDispatcher` interface and iterates per-squad. Activation logging for both systems uses the unified `powercore.PowerLogger` (source prefix `[GEAR]` or `[PERK]` is chosen by `artifacts.IsRegisteredBehavior(source)`).

The guiding principle throughout: game logic never lives in GUI state, and the combat core (`tactical/combat/combatcore`) never imports `perks` or `artifacts` directly. The dispatch layer in `combatservices` owns the wiring between those packages via the `combattypes.PerkDispatcher` interface and the `PowerPipeline`.

---

## 2. System Overview

```
game_main
    |
    +-- common.InitializeSubsystems()
    |       |
    |       +-- 14 RegisterSubsystem() calls (init functions in each package)
    |
    +-- NewCombatService()
            |
            +-- CombatActionSystem    ← SetPerkDispatcher(interface)
            |                           SetOnAttackComplete(pipeline.FireAttackComplete)
            |
            +-- TurnManager           ← SetOnTurnEnd(pipeline.FireTurnEnd)
            |                           SetPostResetHook(pipeline.FirePostReset)
            |
            +-- CombatMovementSystem  ← SetOnMoveComplete(pipeline.FireMoveComplete)
            |
            +-- powerPipeline *powercore.PowerPipeline
                 ├─ OnPostReset:      artifactDispatcher.DispatchPostReset
                 │                    → perkDispatcher.DispatchTurnStart
                 ├─ OnAttackComplete: artifactDispatcher.DispatchOnAttackComplete
                 │                    → perkDispatcher.DispatchAttackTracking
                 │                    → onAttackCompleteGUI (optional)
                 ├─ OnTurnEnd:        artifactDispatcher.DispatchOnTurnEnd
                 │                    → perkDispatcher.DispatchRoundEnd
                 │                    → onTurnEndGUI (optional)
                 └─ OnMoveComplete:   perkDispatcher.DispatchMoveTracking
                                      → onMoveCompleteGUI (optional)


perks.perkBehaviorImpls (map[PerkID]PerkBehavior)
    |
    +-- RegisterPerkBehavior() called from tactical/powers/perks/behaviors.go init()
    |
    +-- SquadPerkDispatcher implements combattypes.PerkDispatcher
    |      (injected into CombatActionSystem via SetPerkDispatcher)
    |
    +-- powercore.PowerLogger shared with artifacts


artifacts.behaviorRegistry (map[string]ArtifactBehavior)
    |
    +-- RegisterBehavior() called from tactical/powers/artifacts/behaviors.go init()
    |
    +-- ArtifactDispatcher dispatches via AllBehaviors() / GetEquippedBehaviors()
    |
    +-- Logger set via dispatcher.SetLogger(powercore.PowerLogger)
    |
    +-- ValidateBehaviorCoverage() cross-checks vs. templates.ArtifactRegistry at startup
    |
    +-- ArtifactBalance loaded from artifactbalanceconfig.json at startup


world/worldgen.generators (map[string]worldmapcore.MapGenerator)
    |
    +-- RegisterGenerator() called from gen_*.go and garrisongen/generator.go init()
    |
    +-- ConfigOverride func hook for JSON-driven config injection


campaign/overworld/core.EventLog
    |
    +-- LogEvent() → EventLog.AddEvent() → auto-records to OverworldRecorder
```

---

## 3. Perk Hook System

**Location:** `tactical/powers/perks/`

### 3.1 HookContext

All ten hook types receive a `*HookContext`. The struct **embeds** `powercore.PowerContext` by value, so `Manager`, `Cache`, `RoundNumber`, and `Logger` come from a single shared definition shared with artifacts. Perk-specific fields stay on the embedder:

```go
// tactical/powers/perks/hooks.go
type HookContext struct {
    powercore.PowerContext

    AttackerID      ecs.EntityID
    DefenderID      ecs.EntityID
    AttackerSquadID ecs.EntityID
    DefenderSquadID ecs.EntityID
    SquadID         ecs.EntityID // Squad that owns the perk (TurnStart, DeathOverride)
    UnitID          ecs.EntityID // Specific unit (DeathOverride, DamageRedirect)
    DamageAmount    int          // Incoming damage (DamageRedirect)
    RoundState      *PerkRoundState
}
```

`buildHookContext` in `queries.go` is the canonical constructor. It returns `nil` if the owner squad has no `PerkRoundState` component, which causes dispatcher methods to exit early without calling any hooks. `ctx.Log(source, squadID, message)` is nil-safe and routes through the shared `PowerLogger`.

### 3.2 PerkBehavior Interface

Each perk implements the `PerkBehavior` interface. Concrete behaviors embed `BasePerkBehavior` (which provides no-op defaults) and override only the hooks they need.

```go
// tactical/powers/perks/hooks.go
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

**Hook method signatures:**

| Method | Signature |
|---|---|
| `AttackerDamageMod` | `(ctx *HookContext, modifiers *combattypes.DamageModifiers)` |
| `DefenderDamageMod` | `(ctx *HookContext, modifiers *combattypes.DamageModifiers)` |
| `DefenderCoverMod` | `(ctx *HookContext, coverBreakdown *combattypes.CoverBreakdown)` |
| `TargetOverride` | `(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID` |
| `CounterMod` | `(ctx *HookContext, modifiers *combattypes.DamageModifiers) (skipCounter bool)` |
| `AttackerPostDamage` | `(ctx *HookContext, damageDealt int, wasKill bool)` |
| `DefenderPostDamage` | `(ctx *HookContext, damageDealt int, wasKill bool)` |
| `TurnStart` | `(ctx *HookContext)` |
| `DamageRedirect` | `(ctx *HookContext) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)` |
| `DeathOverride` | `(ctx *HookContext) (preventDeath bool)` |

### 3.3 Registry Pattern

Perk IDs use the typed `PerkID` string type (`type PerkID string`) defined in `perkids.go`. All 21 perk ID constants are typed, preventing raw strings from being passed where a perk ID is expected.

```go
// tactical/powers/perks/registry.go
var perkBehaviorImpls = map[PerkID]PerkBehavior{}

func RegisterPerkBehavior(b PerkBehavior)
func GetPerkBehavior(perkID PerkID) PerkBehavior
```

All `RegisterPerkBehavior` calls happen inside `init()` in a single file: `tactical/powers/perks/behaviors.go`. The file is internally organised by state requirement via section comments — **Stateless** (11 perks), **Per-round stateful** (7 perks), **Per-battle stateful** (3 perks) — for 21 total perks across Tier 1 (Combat Conditioning) and Tier 2 (Combat Specialization). Each perk implements only the hook methods it needs; the rest are inherited as no-ops from `BasePerkBehavior`.

### 3.4 SquadPerkDispatcher

`SquadPerkDispatcher` (in `tactical/powers/perks/dispatcher.go`) implements `combattypes.PerkDispatcher`. It owns the iteration logic that was previously spread across runner functions. Per-attack dispatcher methods:

| Method | Iterates perks of | Key behavior |
|---|---|---|
| `AttackerDamageMod` | attacker squad | invokes each `PerkBehavior.AttackerDamageMod` |
| `DefenderDamageMod` | defender squad | invokes each `DefenderDamageMod` |
| `TargetOverride` | attacker squad | chains target list through each `TargetOverride` |
| `CounterMod` | defender squad | returns `true` on first `skipCounter` |
| `AttackerPostDamage` | attacker squad | calls `AttackerPostDamage` |
| `DefenderPostDamage` | defender squad | calls `DefenderPostDamage` |
| `CoverMod` | defender squad | calls `DefenderCoverMod` |
| `DeathOverride` | specified squad | returns `true` on first `preventDeath` |
| `DamageRedirect` | defender squad | returns on first non-zero redirect target |

Plus four **lifecycle dispatch** methods wired as `PowerPipeline` subscribers:

| Method | Fires when | Effect |
|---|---|---|
| `DispatchTurnStart(squadIDs, round, manager)` | PostReset | `ResetPerkRoundStateTurn` per squad, then invoke each behavior's `TurnStart` hook |
| `DispatchRoundEnd(manager)` | TurnEnd | Clears per-round `PerkState` map across all squads with `PerkSlotTag` |
| `DispatchAttackTracking(attackerID, defenderID, manager)` | AttackComplete | Sets `AttackedThisTurn` on attacker, `WasAttackedThisTurn` on defender |
| `DispatchMoveTracking(squadID, manager)` | MoveComplete | Sets `MovedThisTurn = true`, resets `TurnsStationary` on moved squad |

`RunTurnStartHooks` (the one free-function runner remaining in `system.go`) is the internal helper `DispatchTurnStart` calls for each squad. All other iteration lives on the dispatcher.

### 3.5 PerkRoundState Lifecycle

`PerkRoundState` is an ECS component attached to each squad that has the `PerkSlotComponent`. It stores two kinds of state:

- **Shared tracking fields** — live directly on the struct (e.g., `MovedThisTurn`, `AttackedThisTurn`, `WasAttackedThisTurn`). Set by the dispatch layer, read by multiple perks, reset by `ResetPerkRoundStateTurn`.
- **Per-perk isolated state** — stored in two `map[PerkID]any` maps:
  - `PerkState` — per-round state, cleared by `ResetPerkRoundStateRound`. Each perk defines its own state struct (e.g., `*RecklessAssaultState`, `*BloodlustState`) and accesses it via generic helpers `GetPerkState[T]` / `SetPerkState`.
  - `PerkBattleState` — per-battle state, persists the entire combat, cleaned up by `CleanupRoundState`. Accessed via `GetBattleState[T]` / `SetBattleState`.

`ResetPerkRoundStateTurn` snapshots the current turn's state into the `WasAttackedLastTurn`, `DidNotAttackLastTurn`, and `WasIdleLastTurn` fields before clearing, so `counterpunch` and `deadshots_patience` can read previous-turn information.

The component declaration:

```go
// tactical/powers/perks/components.go
var (
    PerkSlotComponent       *ecs.Component
    PerkRoundStateComponent *ecs.Component
    PerkSlotTag             ecs.Tag
)
```

Initialization occurs in `tactical/powers/perks/init.go` via the subsystem registration pattern.

---

## 4. Power Pipeline and Combat Service Callbacks

**Location:** `tactical/powers/powercore/pipeline.go`, `tactical/combat/combatservices/combat_service.go`

`CombatService` does not own individual `[]func` slices per event. It owns a single `*powercore.PowerPipeline`, and every subscriber registers on it during `NewCombatService` in a declared execution order.

### 4.1 PowerPipeline

```go
// tactical/powers/powercore/pipeline.go
type (
    PostResetHandler      func(factionID ecs.EntityID, squadIDs []ecs.EntityID)
    AttackCompleteHandler func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)
    TurnEndHandler        func(round int)
    MoveCompleteHandler   func(squadID ecs.EntityID)
)

type PowerPipeline struct {
    postReset      []PostResetHandler
    attackComplete []AttackCompleteHandler
    turnEnd        []TurnEndHandler
    moveComplete   []MoveCompleteHandler
}

// Registration (append to ordered list)
func (p *PowerPipeline) OnPostReset(h PostResetHandler)
func (p *PowerPipeline) OnAttackComplete(h AttackCompleteHandler)
func (p *PowerPipeline) OnTurnEnd(h TurnEndHandler)
func (p *PowerPipeline) OnMoveComplete(h MoveCompleteHandler)

// Firing (iterates all registered handlers in order)
func (p *PowerPipeline) FirePostReset(factionID ecs.EntityID, squadIDs []ecs.EntityID)
func (p *PowerPipeline) FireAttackComplete(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)
func (p *PowerPipeline) FireTurnEnd(round int)
func (p *PowerPipeline) FireMoveComplete(squadID ecs.EntityID)
```

### 4.2 Wiring in NewCombatService

Registration is declarative in `NewCombatService`. The order below is the order handlers will fire:

```go
// Ordering (declared in NewCombatService)
//   1. Artifact behaviors (e.g. Deadlock Shackles must lock before perk TurnStart)
//   2. Perk hooks (TurnStart, state tracking)
//   3. GUI callbacks (cache invalidation, visuals) — nil-safe, last

cs.powerPipeline.OnPostReset(cs.artifactDispatcher.DispatchPostReset)
cs.powerPipeline.OnPostReset(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
    cs.perkDispatcher.DispatchTurnStart(squadIDs, cs.TurnManager.GetCurrentRound(), cs.EntityManager)
})

cs.powerPipeline.OnAttackComplete(cs.artifactDispatcher.DispatchOnAttackComplete)
cs.powerPipeline.OnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
    cs.perkDispatcher.DispatchAttackTracking(attackerID, defenderID, cs.EntityManager)
})
cs.powerPipeline.OnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
    if cs.onAttackCompleteGUI != nil { cs.onAttackCompleteGUI(attackerID, defenderID, result) }
})

cs.powerPipeline.OnTurnEnd(cs.artifactDispatcher.DispatchOnTurnEnd)
cs.powerPipeline.OnTurnEnd(func(round int) { cs.perkDispatcher.DispatchRoundEnd(cs.EntityManager) })
cs.powerPipeline.OnTurnEnd(func(round int) { if cs.onTurnEndGUI != nil { cs.onTurnEndGUI(round) } })

cs.powerPipeline.OnMoveComplete(func(squadID ecs.EntityID) { cs.perkDispatcher.DispatchMoveTracking(squadID, cs.EntityManager) })
cs.powerPipeline.OnMoveComplete(func(squadID ecs.EntityID) { if cs.onMoveCompleteGUI != nil { cs.onMoveCompleteGUI(squadID) } })
```

Then the combat subsystems' single-subscriber hooks are wired to the pipeline's `Fire*` methods so the subsystem never needs to know about any specific subscriber:

```go
combatActSystem.SetOnAttackComplete(cs.powerPipeline.FireAttackComplete)
movementSystem.SetOnMoveComplete(cs.powerPipeline.FireMoveComplete)
turnManager.SetOnTurnEnd(cs.powerPipeline.FireTurnEnd)
turnManager.SetPostResetHook(cs.powerPipeline.FirePostReset)
```

### 4.3 GUI Callback Registration

`CombatMode` installs its GUI callbacks via dedicated setters that replace the single GUI slot for each event:

```go
// tactical/combat/combatservices/combat_service.go
func (cs *CombatService) SetOnAttackCompleteGUI(fn func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult))
func (cs *CombatService) SetOnMoveCompleteGUI(fn func(squadID ecs.EntityID))
func (cs *CombatService) SetOnTurnEndGUI(fn func(round int))
```

`CombatMode.Enter` re-installs these each battle because the GUI closures capture widget pointers that are invalid after the combat UI is torn down.

### 4.4 When Each Event Fires

| Event | Fires when |
|---|---|
| `PostReset` | After `TurnManager.ResetSquadActions` completes for a faction (at start of that faction's turn). Also fires once during `InitializeCombat` for the first faction. |
| `AttackComplete` | After `CombatActionSystem.ExecuteAttackAction` succeeds and all damage/healing has been applied |
| `TurnEnd` | After `TurnManager.EndTurn()` advances the round counter and resets the new faction's actions |
| `MoveComplete` | After `CombatMovementSystem` successfully moves a squad |

### 4.5 Charge Tracker Lifecycle

`ArtifactChargeTracker` is created once in `NewCombatService` and **kept for the service's lifetime**. Per-battle reset happens via `tracker.Reset()` inside `InitializeCombat`, so the `ArtifactDispatcher` bindings on `PowerPipeline` stay valid across battles without re-subscribing.

---

## 5. Perk Dispatch Bridge

**Location:** `tactical/combat/combattypes/perk_callbacks.go` (the interface),
`tactical/powers/perks/dispatcher.go` (the implementation),
`tactical/combat/combatservices/combat_power_dispatch.go` (the wiring)

### Why it exists

`combatcore` (the package containing `CombatActionSystem`) must not import `perks`. Doing so would create a circular import because `perks` already imports `combattypes` for `DamageModifiers` and `CoverBreakdown`. The dispatch layer in `combatservices` sits above both and is allowed to import both.

### PerkDispatcher interface

```go
// tactical/combat/combattypes/perk_callbacks.go
type PerkDispatcher interface {
    AttackerDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
        modifiers *DamageModifiers, manager *common.EntityManager)
    DefenderDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
        modifiers *DamageModifiers, manager *common.EntityManager)
    CoverMod(attackerID, defenderID ecs.EntityID, cover *CoverBreakdown,
        manager *common.EntityManager)
    TargetOverride(attackerID, defenderSquadID ecs.EntityID, targets []ecs.EntityID,
        manager *common.EntityManager) []ecs.EntityID
    CounterMod(defenderSquadID, attackerID ecs.EntityID, modifiers *DamageModifiers,
        manager *common.EntityManager) bool
    AttackerPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
        damage int, wasKill bool, manager *common.EntityManager)
    DefenderPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
        damage int, wasKill bool, manager *common.EntityManager)
    DeathOverride(unitID, squadID ecs.EntityID, manager *common.EntityManager) bool
    DamageRedirect(defenderID, defenderSquadID ecs.EntityID, damageAmount int,
        manager *common.EntityManager) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)
}
```

Implementation: `perks.SquadPerkDispatcher`.

### Injection into combat core

```go
// tactical/combat/combatservices/combat_power_dispatch.go
func setupPowerDispatch(cs *CombatService, manager *common.EntityManager, cache *combatstate.CombatQueryCache) {
    // Shared PowerLogger used by both dispatchers
    logger := powercore.LoggerFunc(func(source string, squadID ecs.EntityID, message string) {
        prefix := "[PERK]"
        if artifacts.IsRegisteredBehavior(source) {
            prefix = "[GEAR]"
        }
        fmt.Printf("%s %s: %s (squad %d)\n", prefix, source, message, squadID)
    })
    cs.artifactDispatcher.SetLogger(logger)

    perkDispatcher := &perks.SquadPerkDispatcher{}
    perkDispatcher.SetLogger(logger)
    cs.perkDispatcher = perkDispatcher
    cs.CombatActSystem.SetPerkDispatcher(cs.perkDispatcher)
}
```

`CombatActionSystem` calls `cs.perkDispatcher.AttackerDamageMod(...)`, `...DefenderDamageMod(...)`, etc. during an attack. The four *lifecycle* dispatcher methods (`DispatchTurnStart`, `DispatchRoundEnd`, `DispatchAttackTracking`, `DispatchMoveTracking`) are reached through `PowerPipeline` subscribers, not through the interface call site.

---

## 6. TurnManager Callbacks

**Location:** `tactical/combat/combatcore/turnmanager.go`

`TurnManager` holds two single-subscriber function fields:

```go
type TurnManager struct {
    // ...
    onTurnEnd     func(round int)
    postResetHook func(factionID ecs.EntityID, squadIDs []ecs.EntityID)
}

func (tm *TurnManager) SetOnTurnEnd(fn func(int))
func (tm *TurnManager) SetPostResetHook(fn func(factionID ecs.EntityID, squadIDs []ecs.EntityID))
```

Single-subscriber: the last `Set*` call wins. `CombatService` is the only writer. In `NewCombatService`, both are wired directly to the pipeline:

```go
turnManager.SetOnTurnEnd(cs.powerPipeline.FireTurnEnd)
turnManager.SetPostResetHook(cs.powerPipeline.FirePostReset)
```

**When each fires:**

- `onTurnEnd` — fires at the end of `TurnManager.EndTurn()`, after the round counter has been incremented and the new faction's actions have been reset.
- `postResetHook` — fires at the end of `TurnManager.ResetSquadActions()`, once per faction turn start. Also fires once during `InitializeCombat` for the first faction.

`CombatActionSystem` uses a similar single-subscriber field (`onAttackComplete`) wired via `SetOnAttackComplete(cs.powerPipeline.FireAttackComplete)`. `CombatMovementSystem` does the same with `SetOnMoveComplete`.

---

## 7. Artifact Behavior System

**Location:** `tactical/powers/artifacts/`

### ArtifactBehavior interface

```go
// tactical/powers/artifacts/behavior.go
type ArtifactBehavior interface {
    BehaviorKey() string
    TargetType() BehaviorTargetType
    OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID)
    OnAttackComplete(ctx *BehaviorContext, attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)
    OnTurnEnd(ctx *BehaviorContext, round int)
    IsPlayerActivated() bool
    Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error
}
```

`TargetType()` returns a typed `BehaviorTargetType` enum (`TargetNone`, `TargetFriendly`, `TargetEnemy`) with a `String()` method for display labels. The GUI (`gui/guiartifacts`) imports these constants directly rather than maintaining duplicate types.

`BaseBehavior` provides no-op implementations of all methods except `BehaviorKey`. Concrete behaviors embed `BaseBehavior` and override only the methods they need.

**File split:**
- `behavior.go` — interface, constants (`BehaviorEngagementChains`, etc.), `BehaviorTargetType` enum, `BaseBehavior`.
- `behaviors.go` — all six concrete behavior structs and their `init()` registrations.

### BehaviorContext

```go
// tactical/powers/artifacts/context.go
type BehaviorContext struct {
    powercore.PowerContext              // embedded: Manager, Cache, RoundNumber, Logger
    ChargeTracker *ArtifactChargeTracker
}

func NewBehaviorContext(power *powercore.PowerContext, chargeTracker *ArtifactChargeTracker) *BehaviorContext
```

Helper methods on `BehaviorContext` perform common squad/action-state mutations: `SetSquadLocked`, `ResetSquadActions`, plus cache-backed reads via the embedded `PowerContext.Cache`.

### Registry

```go
var behaviorRegistry = map[string]ArtifactBehavior{}

func RegisterBehavior(b ArtifactBehavior)
func GetBehavior(key string) ArtifactBehavior
func AllBehaviors() []ArtifactBehavior  // returns sorted by BehaviorKey for determinism
func IsRegisteredBehavior(key string) bool
```

### Logging

Artifacts use the unified `powercore.PowerLogger` rather than a dedicated artifact logger. The logger is set on the dispatcher via `SetLogger(powercore.PowerLogger)`; behaviors invoke it through `ctx.Log(behaviorKey, squadID, message)` (inherited from the embedded `PowerContext`). `artifacts.IsRegisteredBehavior(source)` determines whether the emitted line gets a `[GEAR]` or `[PERK]` prefix.

### Validation

`ValidateBehaviorCoverage()` cross-checks the behavior registry against `templates.ArtifactRegistry` at startup. It warns if a major artifact definition references a behavior key with no implementation, or if a registered behavior has no corresponding artifact definition. Called during `GameBootstrap.LoadGameData()` after both registries are populated.

### Balance Config

Tunable values for artifact behaviors are externalised in `resources/assets/gamedata/artifactbalanceconfig.json` and loaded into `artifacts.ArtifactBalance` by `LoadArtifactBalanceConfig()`. Each behavior with tunables has a typed struct (e.g., `SaboteursHourglassBalance{MovementReduction int}`). The loader validates fields at startup.

### Concrete behaviors

| Key | Behavior |
|---|---|
| `engagement_chains` | Grants a full move action after the attacker destroys a squad (passive/event-driven) |
| `saboteurs_hourglass` | Player-activated; locks target squad on next `OnPostReset` |
| `twin_strike` | Player-activated; grants a bonus attack action to a squad |
| `deadlock_shackles` | Player-activated; fully locks target squad (no move, no act) |
| `chain_of_command` | Player-activated; resets a friendly squad's actions |
| `echo_drums` | Player-activated; grants bonus movement to all friendly squads |

### Passive vs. activated

`IsPlayerActivated()` returns `false` for purely passive behaviors and `true` for those that require an explicit player trigger. `ActivateArtifact(behavior, targetSquadID, ctx)` gates on this flag and returns an error if called on a non-activated behavior. `CanActivateArtifact` checks charge availability before activation.

### ArtifactDispatcher

**Location:** `tactical/powers/artifacts/dispatcher.go`

```go
type ArtifactDispatcher struct {
    manager       *common.EntityManager
    cache         *combatstate.CombatQueryCache
    chargeTracker *ArtifactChargeTracker
    logger        powercore.PowerLogger
}

func NewArtifactDispatcher(manager, cache, chargeTracker) *ArtifactDispatcher
func (d *ArtifactDispatcher) SetLogger(l powercore.PowerLogger)
func (d *ArtifactDispatcher) DispatchPostReset(factionID, squadIDs)       // broadcast to all behaviors
func (d *ArtifactDispatcher) DispatchOnAttackComplete(attackerID, defenderID, result)  // scoped
func (d *ArtifactDispatcher) DispatchOnTurnEnd(round)                     // broadcast + charge refresh
```

`CombatService` owns an `ArtifactDispatcher` instance (created in `NewCombatService` with a persistent `ChargeTracker`). The three dispatch methods are wired as subscribers on `PowerPipeline` (see §4).

---

## 8. Encounter Post-Combat Callback

**Location:** `mind/encounter/encounter_service.go`

```go
type EncounterService struct {
    // ...
    postCombatCallback func(combatlifecycle.CombatExitReason, *combattypes.EncounterOutcome)
}

func (es *EncounterService) SetPostCombatCallback(fn func(combatlifecycle.CombatExitReason, *combattypes.EncounterOutcome))
func (es *EncounterService) ClearPostCombatCallback()
```

Single-subscriber field. The last `SetPostCombatCallback` call wins. It fires at the end of `ExitCombat`, after all cleanup steps (outcome resolution, history recording, entity disposal, player squad stripping) have completed. The only current subscriber is `RaidRunner`, which registers at construction time and clears on `finishRaid`.

`ExitCombat` is the single exit point for all combat endings (victory, defeat, flee). Any system that needs to respond to combat completion should use this callback rather than hooking directly into `CombatService`.

---

## 9. ECS Subsystem Self-Registration

**Location:** `core/common/ecsutil.go`

```go
var subsystemRegistrars []func(*EntityManager)

func RegisterSubsystem(registrar func(*EntityManager))
func InitializeSubsystems(em *EntityManager)
```

Each subsystem package calls `common.RegisterSubsystem` in its `init()` function. `InitializeSubsystems` is called once in game startup after `NewEntityManager()` returns. Registrars execute in Go's `init()` order (package import order).

**Registered packages** (14 subsystems):

| Package | Location |
|---|---|
| `common` (base components) | `core/common/ecsutil.go` — PositionComponent, NameComponent, etc. are initialized before `InitializeSubsystems` |
| `squadcore` | `tactical/squads/squadcore/squadmanager.go` |
| `unitprogression` | `tactical/squads/unitprogression/components.go` |
| `roster` | `tactical/squads/roster/init.go` |
| `effects` | `tactical/powers/effects/init.go` |
| `spells` | `tactical/powers/spells/init.go` |
| `artifacts` | `tactical/powers/artifacts/init.go` |
| `perks` | `tactical/powers/perks/init.go` |
| `progression` | `tactical/powers/progression/init.go` |
| `combatstate` | `tactical/combat/combatstate/combatcomponents.go` |
| `commander` | `tactical/commander/init.go` |
| `campaign/overworld/core` | `campaign/overworld/core/init.go` |
| `campaign/raid` | `campaign/raid/init.go` |

Each registrar calls `em.World.NewComponent()` for its components and `ecs.BuildTag()` for its tags. Some also create ECS Views (`OverworldNodeView`, `OverworldFactionView` in `campaign/overworld/core`).

---

## 10. Map Generator Registry

**Location:** `world/worldmapcore/generator.go` (interface), `world/worldgen/registry.go` (registry + overrides)

```go
// world/worldmapcore/generator.go
type MapGenerator interface {
    Generate(width, height int, images TileImageSet) GenerationResult
    Name() string
    Description() string
}

// world/worldgen/registry.go
var generators = make(map[string]worldmapcore.MapGenerator)

func RegisterGenerator(gen worldmapcore.MapGenerator)
func GetGeneratorOrDefault(name string) worldmapcore.MapGenerator  // falls back to "rooms_corridors"
```

Each generator registers itself in its own `init()` function. The four registered generators:

| Name | File | Generator type |
|---|---|---|
| `rooms_corridors` | `world/worldgen/gen_rooms_corridors.go` | `RoomsAndCorridorsGenerator` |
| `strategic_overworld` | `world/worldgen/gen_overworld.go` | `StrategicOverworldGenerator` |
| `cavern` | `world/worldgen/gen_cavern.go` | `CavernGenerator` |
| `garrison_raid` | `world/garrisongen/generator.go` | `GarrisonRaidGenerator` |

### ConfigOverride hook

```go
// world/worldgen/registry.go
var ConfigOverride func(name string) worldmapcore.MapGenerator
```

`ConfigOverride` is a package-level function variable. When non-nil, it is called by `GetGeneratorOrDefault` before falling back to the registry. `game_main` sets this after loading JSON config to inject generators built with runtime parameters (e.g., room count from the config file). If it returns `nil`, the registry entry is used unchanged.

---

## 11. GUI Callback Patterns

### Panel OnCreate callback

```go
// gui/framework/panelregistry.go
type PanelDescriptor struct {
    SpecName string
    Content  PanelContentType
    Position func(*specs.LayoutConfig) builders.PanelOption
    Width    float64
    Height   float64
    OnCreate func(*PanelResult, UIMode) error
}
```

`OnCreate` is called once after the panel container is built. It receives the `PanelResult` (with `Container` already set if a `SpecName` was provided) and the current `UIMode`. Panels with `ContentCustom` typically implement all widget creation inside `OnCreate` and store widget references in `PanelResult.Custom`. If `OnCreate` is non-nil, the `BuildRegisteredPanel` function skips its own text-label construction.

### ButtonSpec OnClick

```go
// gui/builders/widgets.go
type ButtonSpec struct {
    Text    string
    OnClick func()
}
```

`OnClick` is a bare `func()` stored per-button. Helper functions `CreateBottomActionBar`, `CreateLeftActionBar`, and `CreateRightActionBar` accept `[]ButtonSpec` slices and wire each `OnClick` into the underlying ebitenui button widget.

### Animation SetOnComplete

```go
// gui/guicombat/combat_animation_mode.go
func (cam *CombatAnimationMode) SetOnComplete(callback func())
```

`SetOnComplete` stores a single `func()` in `cam.onAnimationComplete`. It is called when the animation reaches `PhaseComplete`. The caller sets it immediately before starting each animation sequence and the callback typically advances the combat turn flow (e.g., checking victory, updating the UI state). There is no explicit clear mechanism; the field is overwritten on each use.

### CommandHistory onRefresh

```go
// gui/framework/commandhistory.go
func NewCommandHistory(onStatusChange func(string), onRefresh func()) *CommandHistory
```

`onRefresh` is called after every successful `Execute`, `Undo`, or `Redo` call. GUI modes pass a closure that re-renders the relevant panel (e.g., the squad editor list). `onStatusChange` is called unconditionally with a human-readable result string. Both are stored as private fields and are not replaceable after construction. Modes call `BaseMode.InitializeCommandHistory(onRefresh)` to set both callbacks together.

---

## 12. Overworld Event System

**Location:** `campaign/overworld/core/events.go`

### Types

```go
type OverworldEvent struct {
    Type        EventType
    Tick        int64
    EntityID    ecs.EntityID
    Description string
    Data        map[string]interface{}
}

type EventLog struct {
    Events  []OverworldEvent
    MaxSize int
    Unread  int
}
```

`EventLog` keeps the most recent `MaxSize` events (default 100) in a sliding window. The `Unread` counter increments on every `AddEvent` and is used by the GUI to display a notification badge. Callers are responsible for resetting `Unread` when events are displayed.

### Auto-recording on AddEvent

`EventLog.AddEvent` checks `GetContext().Recorder` after appending. If the recorder is enabled, it calls `ctx.Recorder.RecordEvent(...)` synchronously before trimming the log. This means every event that enters the log is automatically written to the `OverworldRecorder` without the caller needing to know about recording.

### Session management

```go
func StartRecordingSession(currentTick int64)   // enables recorder, begins session
func FinalizeRecording(outcome, reason string) error  // exports to JSON
func ClearRecording()                            // resets for next session
```

Recording is gated on `config.ENABLE_OVERWORLD_LOG_EXPORT`. When disabled, `AddEvent` still appends to `EventLog.Events` for in-game display but skips the recorder call.

### Calling convention

```go
core.LogEvent(eventType, tick, entityID, description, nil)
```

`LogEvent` is the sole entry point for external callers. It constructs the `OverworldEvent`, calls `EventLog.AddEvent`, and also prints to stdout for debugging. Passing `nil` for `data` is safe — it is replaced with an empty map.

---

## 13. Combat Round Hook Execution Order

The following is the sequential execution order for one complete attack cycle, showing every hook and callback that fires:

```
Player selects attack (attackerID → defenderID)
│
├── 1. CombatActionSystem.ExecuteAttackAction
│       │
│       ├── 2. [for each attacking unit]
│       │       │
│       │       ├── 3. perkDispatcher.TargetOverride (attacker squad)
│       │       │       → TargetOverride on each attacker perk
│       │       │
│       │       ├── 4. ProcessAttackOnTargets [for each target unit]
│       │       │       │
│       │       │       ├── 5. perkDispatcher.AttackerDamageMod (attacker perks)
│       │       │       │
│       │       │       ├── 6. perkDispatcher.DefenderDamageMod (defender perks)
│       │       │       │
│       │       │       ├── 7. combatmath.CalculateDamage
│       │       │       │       └── perkDispatcher.CoverMod (inside applyResistanceAndCover)
│       │       │       │
│       │       │       ├── 8. perkDispatcher.DamageRedirect (defender perks)
│       │       │       │       → redirected damage recorded to redirect target
│       │       │       │
│       │       │       ├── 9. Apply damage to defender
│       │       │       │
│       │       │       ├── 10. perkDispatcher.DeathOverride (if damage would be lethal)
│       │       │       │       → may reverse the kill, adjust recorded damage
│       │       │       │
│       │       │       ├── 11. perkDispatcher.AttackerPostDamage
│       │       │       │
│       │       │       └── 12. perkDispatcher.DefenderPostDamage
│       │       │
│       │       └── (counterattack — same inner pipeline, steps 5–12, with
│       │                perkDispatcher.CounterMod checked first)
│       │
│       ├── 13. ApplyRecordedDamage / ApplyRecordedHealing (state mutation)
│       │
│       └── 14. combatActSystem.onAttackComplete fires
│               → routed to powerPipeline.FireAttackComplete
│               → pipeline subscribers in order:
│                       ├── artifactDispatcher.DispatchOnAttackComplete
│                       ├── perkDispatcher.DispatchAttackTracking
│                       │     (sets AttackedThisTurn, WasAttackedThisTurn)
│                       └── onAttackCompleteGUI (if set)
│
│   (on end of player turn — player clicks End Turn)
│
├── 15. TurnManager.EndTurn
│       ├── increments round/turn index
│       ├── TurnManager.ResetSquadActions (new faction)
│       │       ├── effects.TickEffectsForUnits (decrements durations)
│       │       └── turnManager.postResetHook fires
│       │               → routed to powerPipeline.FirePostReset
│       │               → pipeline subscribers in order:
│       │                       ├── artifactDispatcher.DispatchPostReset
│       │                       └── perkDispatcher.DispatchTurnStart
│       │                               ├── ResetPerkRoundStateTurn() per squad
│       │                               └── TurnStart on each perk behavior
│       │
│       └── turnManager.onTurnEnd fires
│               → routed to powerPipeline.FireTurnEnd
│               → pipeline subscribers in order:
│                       ├── artifactDispatcher.DispatchOnTurnEnd
│                       │     (refresh per-round charges, call OnTurnEnd on each behavior)
│                       ├── perkDispatcher.DispatchRoundEnd
│                       │     (clear per-round PerkState across all squads)
│                       └── onTurnEndGUI (if set)
```

Note: `ResetSquadActions` (→ `PostReset`) fires before `EndTurn` completes (→ `TurnEnd`). The first `PostReset` fires once during `InitializeCombat` for the opening faction, without a preceding `TurnEnd`. The ordering means `TurnStart` perk hooks run at the start of the new faction's turn, while `ResetPerkRoundStateRound` (inside `DispatchRoundEnd`) runs at the very end of the previous faction's turn. Perks that need previous-turn data must read from the round state *before* `ResetPerkRoundStateTurn` clears the per-turn tracking fields.

---

## 14. Quick Reference Table

| System | Location | Pattern type | Subscriber model | When it fires |
|---|---|---|---|---|
| Perk behavior registry | `tactical/powers/perks/registry.go` | Interface + `map[PerkID]PerkBehavior` | One `PerkBehavior` per `PerkID` | Called by `SquadPerkDispatcher` during attack pipeline |
| PerkDispatcher interface | `tactical/combat/combattypes/perk_callbacks.go` | Interface with 9 methods | Single injected implementation | Inside `CombatActionSystem.ProcessAttackOnTargets`, `ProcessCounterattackOnTargets` |
| SquadPerkDispatcher | `tactical/powers/perks/dispatcher.go` | Struct method receiver | Implements PerkDispatcher + lifecycle Dispatch* methods | Per-attack (via interface) + on PowerPipeline events (lifecycle) |
| Power dispatch wiring | `tactical/combat/combatservices/combat_power_dispatch.go` | Logger + perk dispatcher injection | N/A (wiring only) | At `NewCombatService` construction |
| PowerPipeline | `tactical/powers/powercore/pipeline.go` | Per-event `[]Handler` slices + `Fire*` | Multi-subscriber (registered by `CombatService`) | On `Fire*` calls from combat subsystem callbacks |
| CombatService GUI setters | `tactical/combat/combatservices/combat_service.go` | Single func fields (`onAttackCompleteGUI`, `onMoveCompleteGUI`, `onTurnEndGUI`) | Single subscriber per event | Invoked as last pipeline subscriber, nil-safe |
| TurnManager onTurnEnd | `tactical/combat/combatcore/turnmanager.go` | Single func field | Single subscriber (wired to `pipeline.FireTurnEnd`) | Inside `TurnManager.EndTurn` |
| TurnManager postResetHook | `tactical/combat/combatcore/turnmanager.go` | Single func field | Single subscriber (wired to `pipeline.FirePostReset`) | Inside `TurnManager.ResetSquadActions` |
| CombatActionSystem onAttackComplete | `tactical/combat/combatcore/combatactionsystem.go` | Single func field | Single subscriber (wired to `pipeline.FireAttackComplete`) | At end of `ExecuteAttackAction` |
| CombatMovementSystem onMoveComplete | `tactical/combat/combatcore/combatmovementsystem.go` | Single func field | Single subscriber (wired to `pipeline.FireMoveComplete`) | At end of successful squad move |
| ArtifactBehavior interface | `tactical/powers/artifacts/behavior.go` | Interface + registry in `registry.go` | Broadcast (PostReset, TurnEnd) or squad-scoped (AttackComplete) | Via `ArtifactDispatcher` subscribers on `PowerPipeline` |
| ArtifactDispatcher | `tactical/powers/artifacts/dispatcher.go` | Struct | Instance on `CombatService` | On PowerPipeline events |
| PowerLogger | `tactical/powers/powercore/logger.go` | Interface / func adapter | Single injected; shared by artifacts and perks | On artifact activation / perk hook emission |
| Artifact validation | `tactical/powers/artifacts/registry.go` | Cross-registry check | N/A (one-shot at startup) | During `GameBootstrap.LoadGameData()` |
| Artifact balance config | `tactical/powers/artifacts/balanceconfig.go` | JSON → global struct | N/A (loaded once) | During `GameBootstrap.LoadGameData()` |
| Encounter postCombatCallback | `mind/encounter/encounter_service.go` | Single func field | Single subscriber | At end of `EncounterService.ExitCombat` |
| ECS subsystem registration | `core/common/ecsutil.go` | `[]func(*EntityManager)` slice | All registered subsystems | Once at startup via `InitializeSubsystems` |
| Map generator registry | `world/worldgen/registry.go` | `map[string]worldmapcore.MapGenerator` | One generator per name | On `GetGeneratorOrDefault` call |
| Map generator ConfigOverride | `world/worldgen/registry.go` | Package-level func variable | Single subscriber | Before registry lookup in `GetGeneratorOrDefault` |
| Panel OnCreate | `gui/framework/panelregistry.go` | Field on `PanelDescriptor` | One per panel type | Once when `BuildRegisteredPanel` is called |
| ButtonSpec OnClick | `gui/builders/widgets.go` | Field on `ButtonSpec` | One per button | On user button press (ebitenui event) |
| Animation SetOnComplete | `gui/guicombat/combat_animation_mode.go` | Single func field | Single subscriber | When animation reaches `PhaseComplete` |
| CommandHistory onRefresh | `gui/framework/commandhistory.go` | Constructor param | Single subscriber | After successful `Execute`, `Undo`, or `Redo` |
| Overworld EventLog AddEvent | `campaign/overworld/core/events.go` | Append + auto-record | N/A (push model) | On every `core.LogEvent` call |
