# Hooks and Callbacks in TinkerRogue

**Last Updated:** 2026-03-30

This document is a reference for every hook, callback, and event-driven pattern in the codebase. It covers the perk hook system, combat service callbacks, the dispatch bridge that prevents circular imports, TurnManager single-subscriber hooks, artifact behavior dispatch, the encounter post-combat callback, ECS subsystem self-registration, map generator registration, GUI callback patterns, and the overworld event log.

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [System Overview](#2-system-overview)
3. [Perk Hook System](#3-perk-hook-system)
4. [Combat Service Callbacks](#4-combat-service-callbacks)
5. [Perk Dispatch Layer](#5-perk-dispatch-layer)
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

**Hook tables with a unified context struct** — The perk system defines a single `HookContext` that all nine hook slots receive. This eliminates parameter drift between hook types and avoids making each behavior check whether it is the correct side of a combat exchange (the runner functions handle side selection before dispatching).

**Multi-subscriber slices with explicit registration** — `CombatService` maintains four `[]func` slices for attack, move, turn-end, and post-reset events. Multiple subscribers can listen to the same event. Callbacks are cleared on combat teardown.

**Single-subscriber function fields** — `TurnManager` stores one `onTurnEnd` and one `postResetHook` directly on the struct. Only `CombatService` writes these fields; they fan out to the slices above. The pattern allows the core package to remain unaware of subscribers.

**Interface + registry dispatch** — Artifacts implement `ArtifactBehavior`. The registry holds all registered behaviors. During each event, `CombatService` calls every registered behavior in deterministic key order. Individual behaviors use an embedded `BaseBehavior` no-op to opt out of events they do not need.

The guiding principle throughout: game logic never lives in GUI state, and the `combatcore` package never imports `perks`. The dispatch layer in `combatservices` owns the wiring between those two packages.

---

## 2. System Overview

```
game_main
    |
    +-- common.InitializeSubsystems()
    |       |
    |       +-- 11 RegisterSubsystem() calls (init functions in each package)
    |
    +-- NewCombatService()
            |
            +-- CombatActionSystem  ← perkCallbacks (PerkCallbacks struct)
            |                         onAttackComplete (single func field)
            |
            +-- TurnManager         ← onTurnEnd (single func field)
            |                         postResetHook (single func field)
            |
            +-- CombatMovementSystem ← onMoveComplete (single func field)
            |
            +-- CombatService       ← onAttackComplete []func  ─────────────┐
                                      onMoveComplete   []func               |
                                      onTurnEnd        []func   <── GUI, perk dispatch,
                                      postResetHooks   []func       artifact dispatch


perks.hookRegistry (map[string]*PerkHooks)
    |
    +-- RegisterPerkHooks() called from perks/behaviors.go init()
    |
    +-- Accessed via PerkCallbacks struct injected into CombatActionSystem


artifacts.behaviorRegistry (map[string]ArtifactBehavior)
    |
    +-- RegisterBehavior() called from artifactbehaviors_*.go init()
    |
    +-- Polled via AllBehaviors() in CombatService event handlers


worldmap.generators (map[string]MapGenerator)
    |
    +-- RegisterGenerator() called from gen_*.go init()
    |
    +-- ConfigOverride func hook for JSON-driven config injection


overworld/core.EventLog
    |
    +-- LogEvent() → EventLog.AddEvent() → auto-records to OverworldRecorder
```

---

## 3. Perk Hook System

**Location:** `tactical/perks/`

### 3.1 HookContext

All nine hook types receive a `*HookContext`. Fields that are zero-valued for a given hook type are documented in the source comments.

```go
// tactical/perks/hooks.go
type HookContext struct {
    AttackerID      ecs.EntityID
    DefenderID      ecs.EntityID
    AttackerSquadID ecs.EntityID
    DefenderSquadID ecs.EntityID
    SquadID         ecs.EntityID // Squad that owns the perk (TurnStart, DeathOverride)
    UnitID          ecs.EntityID // Specific unit (DeathOverride, DamageRedirect)
    RoundNumber     int          // Current round (TurnStart)
    DamageAmount    int          // Incoming damage (DamageRedirect)
    RoundState      *PerkRoundState
    Manager         *common.EntityManager
}
```

`buildHookContext` in `queries.go` is the canonical constructor. It returns `nil` if the owner squad has no `PerkRoundState` component, which causes all runner functions to exit early without calling any hooks.

### 3.2 PerkHooks Slots

`PerkHooks` is the struct attached to each perk ID in the registry. All fields are optional; nil fields are skipped by runner functions.

```go
// tactical/perks/hooks.go
type PerkHooks struct {
    AttackerDamageMod DamageModHook      // fires only when owning squad is the attacker
    DefenderDamageMod DamageModHook      // fires only when owning squad is the defender
    DefenderCoverMod  CoverModHook       // fires only when owning squad is the defender
    TargetOverride    TargetOverrideHook
    CounterMod        CounterModHook
    PostDamage        PostDamageHook
    TurnStart         TurnStartHook
    DamageRedirect    DamageRedirectHook
    DeathOverride     DeathOverrideHook
}
```

**Type signatures for each slot:**

| Slot | Signature |
|---|---|
| `AttackerDamageMod` | `func(ctx *HookContext, modifiers *combatcore.DamageModifiers)` |
| `DefenderDamageMod` | `func(ctx *HookContext, modifiers *combatcore.DamageModifiers)` |
| `DefenderCoverMod` | `func(ctx *HookContext, coverBreakdown *combatcore.CoverBreakdown)` |
| `TargetOverride` | `func(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID` |
| `CounterMod` | `func(ctx *HookContext, modifiers *combatcore.DamageModifiers) (skipCounter bool)` |
| `PostDamage` | `func(ctx *HookContext, damageDealt int, wasKill bool)` |
| `TurnStart` | `func(ctx *HookContext)` |
| `DamageRedirect` | `func(ctx *HookContext) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)` |
| `DeathOverride` | `func(ctx *HookContext) (preventDeath bool)` |

### 3.3 Registry Pattern

```go
// tactical/perks/hooks.go
var hookRegistry = map[string]*PerkHooks{}

func RegisterPerkHooks(perkID string, hooks *PerkHooks) {
    hookRegistry[perkID] = hooks
}

func GetPerkHooks(perkID string) *PerkHooks {
    return hookRegistry[perkID]
}
```

All `RegisterPerkHooks` calls happen inside an `init()` function in `tactical/perks/behaviors.go`. There are 24 perks registered across Tier 1 (Combat Conditioning) and Tier 2 (Combat Specialization). Several perks register multiple slots — for example, `bloodlust` registers both `PostDamage` and `AttackerDamageMod` so it can both track kills and apply the resulting bonus.

### 3.4 Runner Functions

Runner functions live in `tactical/perks/queries.go`. They are the public API consumed by `combatservices/perk_dispatch.go` via the `PerkCallbacks` bridge. There are 10 runners:

| Runner | Iterates perks of | Key behavior |
|---|---|---|
| `RunAttackerDamageModHooks` | attacker squad | calls `AttackerDamageMod` |
| `RunDefenderDamageModHooks` | defender squad | calls `DefenderDamageMod` |
| `RunTargetOverrideHooks` | attacker squad | chains target list through each `TargetOverride` |
| `RunCounterModHooks` | defender squad | returns `true` on first `skipCounter` return |
| `RunAttackerPostDamageHooks` | attacker squad | calls `PostDamage` |
| `RunDefenderPostDamageHooks` | defender squad | calls `PostDamage` |
| `RunTurnStartHooks` | specified squad | calls `TurnStart`; does not use `buildHookContext` (round state passed directly) |
| `RunCoverModHooks` | defender squad | calls `DefenderCoverMod` |
| `RunDeathOverrideHooks` | specified squad | returns `true` on first `preventDeath` return |
| `RunDamageRedirectHooks` | defender squad | returns on first non-zero redirect target |

All runners iterate `getActivePerkIDs(squadID, manager)`, which reads the `PerkSlotData.PerkIDs` slice on the squad entity. Perks without a registered `*PerkHooks` entry or with a nil slot are silently skipped.

### 3.5 PerkRoundState Lifecycle

`PerkRoundState` is an ECS component attached to each squad that has the `PerkSlotComponent`. It tracks three scopes of state:

- **Per-turn fields** — reset by `ResetPerTurn()` before `TurnStartHooks` fire. Examples: `MovedThisTurn`, `AttackedThisTurn`, `RecklessVulnerable`.
- **Per-round fields** — reset by `ResetPerRound()` when a new round begins (`OnTurnEnd` fires). Examples: `AttackedBy`, `KillsThisRound`, `DisruptionTargets`.
- **Per-battle fields** — never reset during combat. Examples: `HasAttackedThisCombat`, `ResoluteUsed`, `GrudgeStacks`.

`ResetPerTurn` snapshots the current turn's state into the `WasAttackedLastTurn`, `DidNotAttackLastTurn`, and `WasIdleLastTurn` fields before clearing, so `counterpunch` and `deadshots_patience` can read previous-turn information.

The component declaration:

```go
// tactical/perks/components.go
var (
    PerkSlotComponent       *ecs.Component
    PerkRoundStateComponent *ecs.Component
    PerkSlotTag             ecs.Tag
)
```

Initialization occurs in `tactical/perks/init.go` via the subsystem registration pattern.

---

## 4. Combat Service Callbacks

**Location:** `tactical/combat/combatservices/combat_events.go`

`CombatService` exposes four multi-subscriber callback lists. Each is a `[]func` slice — any number of subscribers may register; all are called in registration order.

```go
// tactical/combat/combatservices/combat_events.go
type OnAttackCompleteFunc func(attackerID, defenderID ecs.EntityID, result *combatcore.CombatResult)
type OnMoveCompleteFunc   func(squadID ecs.EntityID)
type OnTurnEndFunc        func(round int)
type PostResetHookFunc    func(factionID ecs.EntityID, squadIDs []ecs.EntityID)
```

**Registration methods:**

```go
func (cs *CombatService) RegisterOnAttackComplete(fn OnAttackCompleteFunc)
func (cs *CombatService) RegisterOnMoveComplete(fn OnMoveCompleteFunc)
func (cs *CombatService) RegisterOnTurnEnd(fn OnTurnEndFunc)
func (cs *CombatService) RegisterPostResetHook(fn PostResetHookFunc)
```

**Clearing:** `CombatService.ClearCallbacks()` sets all four slices to `nil`. This is called at the start of `CleanupCombat()` to prevent callbacks from firing against torn-down GUI state.

**When each callback fires:**

| Callback | Fires when |
|---|---|
| `onAttackComplete` | After `CombatActionSystem.ExecuteAttackAction` succeeds and all damage/healing has been applied |
| `onMoveComplete` | After `CombatMovementSystem` successfully moves a squad |
| `onTurnEnd` | After `TurnManager.EndTurn()` advances the round counter and resets the new faction's actions |
| `postResetHooks` | After `TurnManager.ResetSquadActions` completes for a faction (at start of that faction's turn) |

**Current subscribers registered in `NewCombatService`:**

- `setupBehaviorDispatch` registers three of the four: `PostResetHook`, `OnAttackComplete`, `OnTurnEnd`.
- `setupPerkDispatch` registers all four.
- The GUI layer (e.g., `guicombat`) registers its own `OnAttackComplete` and `OnMoveComplete` handlers after constructing `CombatService`.

---

## 5. Perk Dispatch Layer

**Location:** `tactical/combat/combatservices/perk_dispatch.go`

### Why it exists

`combatcore` (the package containing `CombatActionSystem`) must not import `perks`. Doing so would create a circular import because `perks` already imports `combatcore` for `DamageModifiers` and `CoverBreakdown`. The dispatch layer in `combatservices` sits above both and is allowed to import both.

### PerkCallbacks bridge struct

```go
// tactical/combat/combatcore/perk_callbacks.go
type PerkCallbacks struct {
    AttackerDamageMod  DamageHookRunner
    DefenderDamageMod  DamageHookRunner
    CoverMod           CoverHookRunner
    TargetOverride     TargetHookRunner
    PostDamage         PostDamageRunner
    DefenderPostDamage PostDamageRunner
    DeathOverride      DeathOverrideRunner
    CounterMod         CounterModRunner
    DamageRedirect     DamageRedirectRunner
}
```

All runner types in `perk_callbacks.go` are defined in `combatcore` with signatures that match the `perks.Run*` functions exactly. `combatservices` assigns the perk runner functions directly (no closure wrappers needed):

```go
// tactical/combat/combatservices/perk_dispatch.go
callbacks := &combatcore.PerkCallbacks{
    AttackerDamageMod:  perks.RunAttackerDamageModHooks,
    DefenderDamageMod:  perks.RunDefenderDamageModHooks,
    CoverMod:           perks.RunCoverModHooks,
    TargetOverride:     perks.RunTargetOverrideHooks,
    PostDamage:         perks.RunAttackerPostDamageHooks,
    DefenderPostDamage: perks.RunDefenderPostDamageHooks,
    DeathOverride:      perks.RunDeathOverrideHooks,
    CounterMod:         perks.RunCounterModHooks,
    DamageRedirect:     perks.RunDamageRedirectHooks,
}
cs.CombatActSystem.SetPerkCallbacks(callbacks)
```

### Additional wiring in setupPerkDispatch

Beyond injecting `PerkCallbacks`, `setupPerkDispatch` registers three `CombatService` callbacks:

1. **`RegisterPostResetHook`** — calls `roundState.ResetPerTurn()` then `RunTurnStartHooks` for each squad in the faction whose turn is starting.
2. **`RegisterOnTurnEnd`** — calls `roundState.ResetPerRound()` for every squad that has a `PerkSlotTag`. This resets per-round fields (kill counts, disruption maps, etc.) when a new round begins.
3. **`RegisterOnAttackComplete`** — marks `AttackedThisTurn = true` on the attacker's round state and `WasAttackedLastTurn = true` on the defender's round state. These flags feed the `counterpunch` and `deadshots_patience` perks on the following turn.
4. **`RegisterOnMoveComplete`** — sets `MovedThisTurn = true` and resets `TurnsStationary = 0` on the moved squad's round state. Used by `stalwart` and `fortify`.

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

These are single-subscriber: the last `Set*` call wins and the previous function is discarded. `CombatService` is the only writer. Its `NewCombatService` constructor sets both fields to closures that fan out to the corresponding `[]func` slices:

```go
turnManager.SetOnTurnEnd(func(round int) {
    for _, fn := range cs.onTurnEnd {
        fn(round)
    }
})
turnManager.SetPostResetHook(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
    for _, fn := range cs.postResetHooks {
        fn(factionID, squadIDs)
    }
})
```

**When each fires:**

- `onTurnEnd` — fires at the end of `TurnManager.EndTurn()`, after the round counter has been incremented and the new faction's actions have been reset.
- `postResetHook` — fires at the end of `TurnManager.ResetSquadActions()`, once per faction turn start. Also fires once during `InitializeCombat` for the first faction.

`CombatActionSystem` uses a similar single-subscriber field (`onAttackComplete`) set via `SetOnAttackComplete`. The same fan-out pattern applies.

---

## 7. Artifact Behavior System

**Location:** `tactical/powers/artifacts/artifactbehavior.go`

### ArtifactBehavior interface

```go
// tactical/powers/artifacts/artifactbehavior.go
type ArtifactBehavior interface {
    BehaviorKey() string
    TargetType() int
    OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID)
    OnAttackComplete(ctx *BehaviorContext, attackerID, defenderID ecs.EntityID, result *combatcore.CombatResult)
    OnTurnEnd(ctx *BehaviorContext, round int)
    IsPlayerActivated() bool
    Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error
}
```

`BaseBehavior` provides no-op implementations of all methods except `BehaviorKey`. Concrete behaviors embed `BaseBehavior` and override only the methods they need.

### BehaviorContext

```go
type BehaviorContext struct {
    Manager       *common.EntityManager
    cache         *combatcore.CombatQueryCache  // unexported; accessed via helper methods
    ChargeTracker *ArtifactChargeTracker
}
```

Helper methods on `BehaviorContext` expose the cache indirectly: `GetActionState`, `SetSquadLocked`, `ResetSquadActions`, `GetSquadFaction`, `GetFactionSquads`, `GetSquadSpeed`.

### Registry

```go
var behaviorRegistry = map[string]ArtifactBehavior{}

func RegisterBehavior(b ArtifactBehavior)
func GetBehavior(key string) ArtifactBehavior
func AllBehaviors() []ArtifactBehavior  // returns sorted by BehaviorKey for determinism
```

### Concrete behaviors

**Passive (event-driven)** — registered in `artifactbehaviors_passive.go`:

| Key | Behavior |
|---|---|
| `engagement_chains` | Grants a full move action after the attacker destroys a squad |
| `saboteurs_hourglass` | Player-activated; locks target squad on next `OnPostReset` |
| `twin_strike` | Player-activated; grants a bonus attack action to a squad |

**Activated** — registered in `artifactbehaviors_activated.go`:

| Key | Behavior |
|---|---|
| `deadlock_shackles` | Player-activated; fully locks target squad (no move, no act) |
| `chain_of_command` | Player-activated; resets a friendly squad's actions |
| `echo_drums` | Player-activated; grants bonus movement to all friendly squads |

### Passive vs. activated

`IsPlayerActivated()` returns `false` for purely passive behaviors and `true` for those that require an explicit player trigger. `ActivateArtifact(behavior, targetSquadID, ctx)` gates on this flag and returns an error if called on a non-activated behavior. `CanActivateArtifact` checks charge availability before activation.

### Dispatch in CombatService

`setupBehaviorDispatch` in `combat_service.go` registers three `CombatService` callbacks that iterate `AllBehaviors()` in sorted order:

```go
cs.RegisterPostResetHook(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
    ctx := makeBehaviorContext()
    for _, b := range artifacts.AllBehaviors() {
        b.OnPostReset(ctx, factionID, squadIDs)
    }
})
cs.RegisterOnAttackComplete(func(...) {
    ctx := makeBehaviorContext()
    for _, b := range artifacts.AllBehaviors() {
        b.OnAttackComplete(ctx, attackerID, defenderID, result)
    }
})
cs.RegisterOnTurnEnd(func(round int) {
    if cs.chargeTracker != nil {
        cs.chargeTracker.RefreshRoundCharges()
    }
    ctx := makeBehaviorContext()
    for _, b := range artifacts.AllBehaviors() {
        b.OnTurnEnd(ctx, round)
    }
})
```

`makeBehaviorContext` captures `cs.chargeTracker` by reference so that `InitializeCombat` can swap in a fresh tracker for each battle without re-registering callbacks.

---

## 8. Encounter Post-Combat Callback

**Location:** `mind/encounter/encounter_service.go`

```go
// mind/encounter/encounter_service.go
type EncounterService struct {
    // ...
    postCombatCallback func(combatcore.CombatExitReason, *combatcore.EncounterOutcome)
}

func (es *EncounterService) SetPostCombatCallback(fn func(combatcore.CombatExitReason, *combatcore.EncounterOutcome))
func (es *EncounterService) ClearPostCombatCallback()
```

This is a single-subscriber field. The last `SetPostCombatCallback` call wins. It fires at the end of `ExitCombat`, after all cleanup steps (outcome resolution, history recording, entity disposal) have completed. The only current subscriber is `RaidRunner`, which registers before each raid combat encounter and clears after receiving the result.

`ExitCombat` is the single exit point for all combat endings (victory, defeat, flee). Any system that needs to respond to combat completion should use this callback rather than hooking directly into `CombatService`.

---

## 9. ECS Subsystem Self-Registration

**Location:** `common/ecsutil.go`

```go
// common/ecsutil.go
var subsystemRegistrars []func(*EntityManager)

func RegisterSubsystem(registrar func(*EntityManager))
func InitializeSubsystems(em *EntityManager)
```

Each subsystem package calls `common.RegisterSubsystem` in its `init()` function. `InitializeSubsystems` is called once in game startup after `NewEntityManager()` returns. Registrars execute in Go's `init()` order (package import order).

**Registered packages** (12 subsystems as of this writing):

| Package | Location |
|---|---|
| `common` (base components) | `common/ecsutil.go` — PositionComponent, NameComponent, etc. are initialized before `InitializeSubsystems` |
| `squadcore` | `tactical/squads/squadcore/squadmanager.go` |
| `unitprogression` | `tactical/squads/unitprogression/components.go` |
| `roster` | `tactical/squads/roster/init.go` |
| `effects` | `tactical/powers/effects/init.go` |
| `spells` | `tactical/powers/spells/init.go` |
| `artifacts` | `tactical/powers/artifacts/init.go` |
| `perks` | `tactical/perks/init.go` |
| `combatcore` | `tactical/combat/combatcore/combatcomponents.go` |
| `commander` | `tactical/commander/init.go` |
| `overworld/core` | `overworld/core/init.go` |
| `mind/raid` | `mind/raid/init.go` |

Each registrar calls `em.World.NewComponent()` for its components and `ecs.BuildTag()` for its tags. Some also create ECS Views (`OverworldNodeView`, `OverworldFactionView` in `overworld/core`).

---

## 10. Map Generator Registry

**Location:** `world/worldmap/generator.go`

```go
// world/worldmap/generator.go
type MapGenerator interface {
    Generate(width, height int, images TileImageSet) GenerationResult
    Name() string
    Description() string
}

var generators = make(map[string]MapGenerator)

func RegisterGenerator(gen MapGenerator)
func GetGeneratorOrDefault(name string) MapGenerator  // falls back to "rooms_corridors"
```

Each generator registers itself in its own `init()` function. The five registered generators:

| Name | File | Generator type |
|---|---|---|
| `rooms_corridors` | `gen_rooms_corridors.go` | `RoomsAndCorridorsGenerator` |
| (strategic overworld) | `gen_overworld.go` | `StrategicOverworldGenerator` |
| (military base) | `gen_military_base.go` | `MilitaryBaseGenerator` |
| (garrison raid) | `gen_garrison.go` | `GarrisonRaidGenerator` |
| (cavern) | `gen_cavern.go` | `CavernGenerator` |

### ConfigOverride hook

```go
// world/worldmap/generator.go
var ConfigOverride func(name string) MapGenerator
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

**Location:** `overworld/core/events.go`

### Types

```go
// overworld/core/events.go
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
│       │       ├── 3. RunTargetOverrideHooks (attacker squad perks)
│       │       │       → TargetOverride slots on attacker's perks
│       │       │
│       │       ├── 4. ProcessAttackOnTargets [for each target unit]
│       │       │       │
│       │       │       ├── 5. RunCoverModHooks (defender perks)
│       │       │       │       → DefenderCoverMod slots on defender's perks
│       │       │       │
│       │       │       ├── 6. RunAttackerDamageModHooks (attacker squad perks)
│       │       │       │       → AttackerDamageMod slots on attacker's perks
│       │       │       │
│       │       │       ├── 7. RunDefenderDamageModHooks (defender squad perks)
│       │       │       │       → DefenderDamageMod slots on defender's perks
│       │       │       │
│       │       │       ├── 8. RunDamageRedirectHooks (defender squad perks)
│       │       │       │       → DamageRedirect slots on defender's perks
│       │       │       │       → redirected damage recorded to redirect target
│       │       │       │
│       │       │       ├── 9. RunDeathOverrideHooks (if damage would be lethal)
│       │       │       │       → DeathOverride slots on defender's perks
│       │       │       │
│       │       │       ├── 10. RunAttackerPostDamageHooks
│       │       │       │       → PostDamage slots on attacker's perks
│       │       │       │
│       │       │       └── 11. RunDefenderPostDamageHooks
│       │       │               → PostDamage slots on defender's perks
│       │       │
│       │       └── (counterattack — same pipeline, steps 4–11, with
│       │                CounterMod hooks checked first via perkCallbacks.CounterMod)
│       │
│       ├── 12. ApplyRecordedDamage / ApplyRecordedHealing (state mutation)
│       │
│       └── 13. cas.onAttackComplete fires (single func field)
│               → fans out to cs.onAttackComplete []func slice
│                       ├── artifact dispatch: AllBehaviors().OnAttackComplete
│                       └── perk tracking: AttackedThisTurn, WasAttackedLastTurn
│
│   (on end of player turn — player clicks End Turn)
│
├── 14. TurnManager.EndTurn
│       ├── increments round/turn index
│       ├── TurnManager.ResetSquadActions (new faction)
│       │       ├── effects.TickEffectsForUnits (decrements durations)
│       │       └── tm.postResetHook fires
│       │               → fans out to cs.postResetHooks []func
│       │                       ├── artifact dispatch: AllBehaviors().OnPostReset
│       │                       └── perk dispatch:
│       │                               ├── roundState.ResetPerTurn()
│       │                               └── RunTurnStartHooks (TurnStart slot on each perk)
│       │
│       └── tm.onTurnEnd fires
│               → fans out to cs.onTurnEnd []func
│                       ├── artifact dispatch: chargeTracker.RefreshRoundCharges,
│                       │                     AllBehaviors().OnTurnEnd
│                       └── perk dispatch: roundState.ResetPerRound()
```

Note: `ResetSquadActions` fires before `onTurnEnd`. This means `TurnStart` perk hooks run before `ResetPerRound` runs. If a perk needs data set during the previous round, it should read from the round state before `ResetPerRound` clears it. The ordering is intentional: `ResetPerTurn` runs at the start of a faction's turn (before hooks), `ResetPerRound` runs after the last faction's turn ends (new round).

---

## 14. Quick Reference Table

| System | Location | Pattern type | Subscriber model | When it fires |
|---|---|---|---|---|
| Perk hook registry | `tactical/perks/hooks.go` | Function-field struct keyed by perk ID | One `*PerkHooks` per perk ID | Called by runner functions during attack pipeline |
| Perk runner functions | `tactical/perks/queries.go` | Named functions | Iterate all active perks on a squad | Called by `PerkCallbacks` inside `combatcore` |
| PerkCallbacks bridge | `tactical/combat/combatcore/perk_callbacks.go` | Function-pointer struct | Single injected set | Inside `ProcessAttackOnTargets`, `ProcessCounterattackOnTargets` |
| Perk dispatch wiring | `tactical/combat/combatservices/perk_dispatch.go` | Wires perks to CombatService callbacks | N/A (wiring only) | At `NewCombatService` construction |
| CombatService onAttackComplete | `tactical/combat/combatservices/combat_events.go` | `[]func` slice | Multi-subscriber | After `ExecuteAttackAction` succeeds |
| CombatService onMoveComplete | `tactical/combat/combatservices/combat_events.go` | `[]func` slice | Multi-subscriber | After `CombatMovementSystem` moves a squad |
| CombatService onTurnEnd | `tactical/combat/combatservices/combat_events.go` | `[]func` slice | Multi-subscriber | After `TurnManager.EndTurn` advances round |
| CombatService postResetHooks | `tactical/combat/combatservices/combat_events.go` | `[]func` slice | Multi-subscriber | After `TurnManager.ResetSquadActions` for a faction |
| TurnManager onTurnEnd | `tactical/combat/combatcore/turnmanager.go` | Single func field | Single subscriber | Inside `TurnManager.EndTurn` |
| TurnManager postResetHook | `tactical/combat/combatcore/turnmanager.go` | Single func field | Single subscriber | Inside `TurnManager.ResetSquadActions` |
| CombatActionSystem onAttackComplete | `tactical/combat/combatcore/combatactionsystem.go` | Single func field | Single subscriber | At end of `ExecuteAttackAction` |
| ArtifactBehavior interface | `tactical/powers/artifacts/artifactbehavior.go` | Interface + registry | All registered behaviors polled | Via CombatService callbacks |
| Encounter postCombatCallback | `mind/encounter/encounter_service.go` | Single func field | Single subscriber | At end of `EncounterService.ExitCombat` |
| ECS subsystem registration | `common/ecsutil.go` | `[]func(*EntityManager)` slice | All registered subsystems | Once at startup via `InitializeSubsystems` |
| Map generator registry | `world/worldmap/generator.go` | `map[string]MapGenerator` | One generator per name | On `GetGeneratorOrDefault` call |
| Map generator ConfigOverride | `world/worldmap/generator.go` | Package-level func variable | Single subscriber | Before registry lookup in `GetGeneratorOrDefault` |
| Panel OnCreate | `gui/framework/panelregistry.go` | Field on `PanelDescriptor` | One per panel type | Once when `BuildRegisteredPanel` is called |
| ButtonSpec OnClick | `gui/builders/widgets.go` | Field on `ButtonSpec` | One per button | On user button press (ebitenui event) |
| Animation SetOnComplete | `gui/guicombat/combat_animation_mode.go` | Single func field | Single subscriber | When animation reaches `PhaseComplete` |
| CommandHistory onRefresh | `gui/framework/commandhistory.go` | Constructor param | Single subscriber | After successful `Execute`, `Undo`, or `Redo` |
| Overworld EventLog AddEvent | `overworld/core/events.go` | Append + auto-record | N/A (push model) | On every `core.LogEvent` call |
