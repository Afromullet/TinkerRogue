# Artifact System Architecture

**Last Updated:** 2026-04-19

---

## Overview

Artifacts are equippable items in TinkerRogue that augment squad capabilities during combat. They represent powerful relics, tools, and banners that a player collects, stockpiles in an inventory, and assigns to squads before entering battle.

The system is split along a clean architectural boundary: **minor artifacts** are passive and apply permanent stat modifiers at battle start, while **major artifacts** carry active behaviors that fire in response to combat events or direct player input.

Artifacts and perks share the `tactical/powers/powercore` foundation — a common `PowerContext`, `PowerLogger`, and `PowerPipeline` — so there is one logger, one context shape, and one ordered dispatch pipeline for both systems.

---

## The Two Categories

### Minor Artifacts

Minor artifacts are purely passive. Each minor artifact definition carries a list of `statModifiers` — positive or negative changes to unit stats such as armor, weapon power, movement speed, attack range, strength, or dexterity. When combat begins, `ApplyArtifactStatEffects` in `artifacts/system.go` iterates every squad in every faction and applies each equipped artifact's modifiers to all units in that squad as permanent `ActiveEffect` entries (duration `-1`). From that point forward, the effects simply exist on the units for the duration of the battle; there is nothing to trigger and no state to track.

Minor artifacts have no `behavior` field in their definition. Their tier is `"minor"`.

### Major Artifacts

Major artifacts are active and behavioral. Instead of static modifiers, a major artifact carries a `behavior` key that ties the artifact's definition to a registered `ArtifactBehavior` implementation. The behavior governs:

- Whether it requires player activation (`IsPlayerActivated()`) or fires automatically from hooks
- What target it requires (friendly squad, enemy squad, or none) — via `TargetType()`
- What game-state mutations it performs when triggered (`Activate`)
- Which lifecycle hooks it reacts to (`OnPostReset`, `OnAttackComplete`, `OnTurnEnd`)
- Whether its charge is once-per-battle or refreshes each round

Major artifact definitions may optionally include `statModifiers` as well, meaning a single artifact can combine passive stat changes with active effects.

Balance tuning values for major artifact behaviors are externalized in `assets/gamedata/artifactbalanceconfig.json` and loaded into the global `ArtifactBalance` config at startup. See **Balance Config** below.

---

## Data Model

### JSON Definitions

Artifact data lives in two JSON files under `assets/gamedata/`:

- `minor_artifacts.json` — all minor artifact blueprints
- `major_artifacts.json` — all major artifact blueprints

Both files share the same schema. Each entry has:

```
id          — unique string key used throughout the system
name        — human-readable display name
description — player-facing text
tier        — "minor" or "major"
statModifiers (optional) — array of { stat, modifier } objects
```

Major artifact entries do not carry a `behavior` field in the JSON files. Instead, during loading the system automatically sets `behavior = id`, so the artifact's unique ID serves double duty as the behavior key. This means the JSON ID and the Go behavior constant must match exactly.

### ArtifactDefinition (Go)

The `templates.ArtifactDefinition` struct is the Go representation of a single artifact blueprint loaded from JSON. It lives in `templates/artifactdefinitions.go`. The global `templates.ArtifactRegistry` map (keyed by artifact ID) is the single source of truth for all artifact definitions at runtime.

### ArtifactInventoryData and ArtifactInstance (ECS)

The player entity carries an `ArtifactInventoryComponent` containing `ArtifactInventoryData`. This tracks every artifact the player owns as a map from artifact ID to a slice of `ArtifactInstance` structs. Supporting multiple copies of the same artifact is a first-class feature: each copy is its own `ArtifactInstance` entry in that slice.

An `ArtifactInstance` has a single field: `EquippedOn`, which holds the `EntityID` of the squad the copy is assigned to, or `0` if the copy is in reserve. This design means ownership and equipment state are always derivable from a single component without cross-referencing separate data structures.

### EquipmentData (ECS)

Each squad entity can carry an `EquipmentComponent` containing `EquipmentData`. This is a simple list of artifact IDs (`[]string`) representing every artifact currently equipped on that squad. The list serves as the squad-side reference; the player's `ArtifactInventoryData` holds the instance-level state. Both must be kept consistent whenever an artifact is equipped or unequipped.

---

## Loading and Registration

### Startup Sequence

At application startup, `templates.ReadGameData()` calls `templates.LoadArtifactDefinitions()`. This reads both JSON files, constructs `ArtifactDefinition` structs, and stores them in `templates.ArtifactRegistry`. Major artifacts receive their `Behavior` field set to match their ID at this point.

### Behavior Registry

Separately from the definition registry, each major artifact has a corresponding Go behavior implementation. Implementations register themselves via `artifacts.RegisterBehavior()` during Go's `init()` phase, populating `artifacts.behaviorRegistry` — a global `map[string]ArtifactBehavior`.

The `init()` function in `artifacts/behaviors.go` calls `RegisterBehavior` for every behavior (both player-activated and automatic); the file no longer splits activated vs. passive into separate files. This follows the same self-registration pattern used by worldmap generators and ECS subsystems throughout the codebase.

### Validation

At startup, after both artifact definitions and behavior registrations are loaded, `artifacts.ValidateBehaviorCoverage()` cross-checks the two registries. It returns `[]error` describing any mismatches:

- A major artifact definition references a `behavior` key with no registered implementation.
- A registered behavior has no corresponding artifact definition in `templates.ArtifactRegistry`.

Callers decide whether mismatches are fatal (e.g., fail startup in debug builds). This mirrors `perks.ValidateHookCoverage()`.

`artifacts.IsRegisteredBehavior(key)` is also exposed so the combat logger can route `[GEAR]` vs `[PERK]` prefixes by asking whether a log `source` is a known artifact behavior.

### Balance Config

Balance tuning values for artifact behaviors are externalized in `assets/gamedata/artifactbalanceconfig.json` and loaded into the global `artifacts.ArtifactBalance` config by `artifacts.LoadArtifactBalanceConfig()`. Each behavior with tunable parameters has a corresponding typed struct (e.g., `SaboteursHourglassBalance` with `MovementReduction int`). Behaviors reference `ArtifactBalance.SaboteursHourglass.MovementReduction` instead of hardcoded values.

The loader validates all fields at startup (e.g., movement reduction must be positive). New tunables are added by extending the appropriate balance struct and JSON file — no behavior code changes are needed for rebalancing.

### ECS Component Registration

The `artifacts` package's own `init()` in `artifacts/init.go` uses `common.RegisterSubsystem` to create the `EquipmentComponent` and `ArtifactInventoryComponent` ECS component types when the entity manager is initialized. This follows the standard subsystem self-registration pattern from `CLAUDE.md`.

---

## Attaching Artifacts to Entities

### Player Inventory

The player entity is created with an `ArtifactInventoryComponent` during `InitializePlayerData` in `game_main/gameinit.go`. The inventory's `MaxArtifacts` is set from `templates.GameConfig.Player.Limits.DefaultPlayerMaxArtifacts`. Artifacts are added to this inventory via `artifacts.AddArtifactToInventory`, which appends an `ArtifactInstance{EquippedOn: 0}` entry. `artifactinventory.go` holds the inventory helpers (`NewArtifactInventory`, `GetPlayerArtifactInventory`, `totalInstanceCount`).

### Equipping to a Squad

Equipping an artifact calls `artifacts.EquipArtifact(playerID, squadID, artifactID, manager)`. This function:

1. Looks up the definition in `ArtifactRegistry` to verify it exists.
2. Confirms the player owns at least one unequipped copy via `IsArtifactAvailable` on the inventory component.
3. Gets or lazily creates the squad's `EquipmentComponent`.
4. Checks that the squad has not exceeded `templates.GameConfig.Player.Limits.MaxArtifactsPerCommander`.
5. Appends the artifact ID to `EquipmentData.EquippedArtifacts`.
6. Marks the specific inventory instance as equipped via `MarkArtifactEquipped` (sets `EquippedOn = squadID`).

Steps 5 and 6 are transactional: if step 6 fails, the change from step 5 is rolled back.

Unequipping reverses this process via `artifacts.UnequipArtifact`, again with rollback on failure.

### Squad Management UI

The `gui/guisquads` package provides the `ArtifactMode` UI screen through which the player browses their inventory and assigns artifacts to squads before deployment. The equipment and inventory tabs both call into the functions above and refresh their lists after each operation.

---

## How Major Artifact Behaviors Work

### The ArtifactBehavior Interface

The `artifacts.ArtifactBehavior` interface, defined in `artifacts/behavior.go`, is the contract every major artifact behavior must satisfy:

```go
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

`TargetType()` returns a typed `BehaviorTargetType` enum (not a bare `int`). The type has a `String()` method returning `"No Target"`, `"Friendly Squad"`, or `"Enemy Squad"` for display purposes. The GUI imports these constants directly from the `artifacts` package rather than maintaining its own duplicate type.

The `artifacts.BaseBehavior` struct provides no-op default implementations of every method, so concrete behaviors only override the hooks they actually use. The default `Activate` returns `"not player-activated"`; the default `IsPlayerActivated` returns `false`.

### The BehaviorContext

Every hook receives a `*BehaviorContext`, which embeds `powercore.PowerContext` (for `Manager`, `Cache`, `RoundNumber`, `Logger`) and adds the artifact-specific `ChargeTracker`:

```go
type BehaviorContext struct {
    powercore.PowerContext
    ChargeTracker *ArtifactChargeTracker
}
```

`BehaviorContext` has two convenience methods that encapsulate common mutations so behaviors don't reach into `ActionStateData` directly:

- `SetSquadLocked(squadID)` — marks a squad as fully spent this turn (`HasActed`, `HasMoved`, `MovementRemaining = 0`).
- `ResetSquadActions(squadID, speed)` — fully refreshes a squad's action state.

Logging flows through the embedded `PowerContext.Log` (nil-safe). Behaviors call `ctx.Log(BehaviorKey, squadID, message)` — the log key is the artifact's behavior string constant, routed to a `[GEAR]` prefix by the shared logger.

### The Three Hooks

**OnPostReset** fires immediately after the `TurnManager.ResetSquadActions` call completes for a faction. This is the start-of-turn moment when squads have their `HasMoved`, `HasActed`, and `MovementRemaining` flags freshly initialized. Behaviors that want to modify a faction's initial action budget for a turn — for example, reducing enemy movement (Saboteur's Hourglass) or locking an enemy squad out before it acts (Deadlock Shackles) — do so here.

**OnAttackComplete** fires after every attack resolves. The hook receives the attacker's ID, the defender's ID, and a `CombatResult`. Behaviors that react to the outcome of an attack — such as Engagement Chains granting a full-move action after a kill — implement this hook. Only behaviors equipped on the *attacker* are dispatched (see below).

**OnTurnEnd** fires after the turn index advances and a full round completes. Per-round charges are refreshed at this moment (before the hook fires), so behaviors can assume round charges have been cleared. `AllBehaviors()` is dispatched unconditionally for this hook (broadcast).

### Charge Tracking

The `artifacts.ArtifactChargeTracker` tracks which behaviors have been used. Two granularities are supported via the `ChargeType` enum:

- **`ChargeOncePerBattle`** — the behavior cannot be used again once spent for the entire battle.
- **`ChargeOncePerRound`** — the behavior's charge refreshes when `OnTurnEnd` fires, so it can be used once per full round.

`CanActivateArtifact(behaviorKey, tracker)` gates all player-triggered activation by checking `ArtifactChargeTracker.IsAvailable`.

The tracker exposes a `Pending *PendingEffectQueue` field directly (no forwarding methods). Callers reach the queue via `tracker.Pending.Add(...)`, `tracker.Pending.Consume(...)`, etc.

`tracker.Reset()` clears battle charges, round charges, and pending effects. It is called at the start of each battle rather than replacing the tracker instance — more on this in the **Lifecycle** section.

### The PendingEffectQueue (Deferred Effects)

Some behaviors — Deadlock Shackles, Saboteur's Hourglass — queue a deferred effect on activation rather than applying it immediately, then consume those queued effects during the target faction's next `OnPostReset`. This two-phase mechanism lets an artifact activated during the player's turn apply its effect at the start of the enemy's next turn.

`PendingEffectQueue` (in `pending_effects.go`) is a small value-typed queue of `PendingArtifactEffect{Behavior, TargetSquadID}` entries:

```go
Phase 1 — Queue (during Activate):
    ctx.ChargeTracker.Pending.Add(BehaviorKey, targetSquadID)
    ctx.ChargeTracker.UseCharge(BehaviorKey, ChargeOncePerBattle)

Phase 2 — Consume (in OnPostReset):
    pendingEffects := ctx.ChargeTracker.Pending.Consume(BehaviorKey)
    // apply effect to each queued targetSquadID
```

Effects without a specific target (e.g., Saboteur's "slow ALL enemy squads") pass `targetSquadID = 0` and the behavior's `OnPostReset` broadcasts to every squad in the faction instead of filtering by target.

### Shared Behavior Helpers

`behaviors.go` factors three common patterns out of individual behavior implementations:

- **`requireCharge(ctx, behaviorKey) error`** — returns a standard error if the behavior's charge is already spent.
- **`activateWithPending(ctx, behaviorKey, chargeType, targetSquadID) error`** — the canonical queue-and-consume-charge pattern used by Deadlock Shackles and Saboteur's Hourglass.
- **`applyPendingEffects(ctx, behaviorKey, squadIDs, broadcast, applyFn)`** — consumes queued effects for `behaviorKey` and invokes `applyFn` for each affected squad. Setting `broadcast = true` ignores per-target filtering (AOE); `broadcast = false` restricts application to squads whose IDs appear as queued targets.

These helpers keep each individual behavior implementation to a few lines.

### Player-Activated vs. Automatic Behaviors

Behaviors where `IsPlayerActivated()` returns `true` appear in the in-combat artifact panel (`gui/guiartifacts/artifact_handler.go`). The player opens artifact mode, selects one of these behaviors, optionally clicks a target squad (if `TargetType()` is not `TargetNone`), and the handler calls `artifacts.ActivateArtifact(behaviorKey, targetSquadID, ctx)`. The charge gate is checked inside `Activate` via `requireCharge`.

Behaviors where `IsPlayerActivated()` returns `false` never appear in the activation UI. They fire automatically whenever a relevant hook fires on a squad that has them equipped.

### Hook Dispatch via ArtifactDispatcher

`ArtifactDispatcher` (`artifacts/dispatcher.go`) encapsulates all artifact lifecycle hook dispatch. It holds references to the entity manager, combat query cache, charge tracker, and logger:

```go
func NewArtifactDispatcher(
    manager *common.EntityManager,
    cache *combatstate.CombatQueryCache,
    chargeTracker *ArtifactChargeTracker,
) *ArtifactDispatcher
```

**`chargeTracker` is required at construction and a nil tracker causes an immediate panic.** A previous design exposed `SetChargeTracker`, which let the dispatcher silently no-op charge refreshes if wiring was forgotten; the constructor-required form surfaces wiring bugs at startup. Per-battle reset happens via `chargeTracker.Reset()` rather than swapping the tracker — the dispatcher's PowerPipeline subscriptions stay valid across battles.

The dispatcher exposes three dispatch methods matching the hook interface:

- **`DispatchPostReset(factionID, squadIDs)`** — fires `OnPostReset` for two disjoint sets of behaviors, deduplicated:
    1. Every behavior equipped on any squad in the faction (via `GetEquippedBehaviors`).
    2. Every behavior with pending effects queued from a previous activation (via `chargeTracker.Pending.Keys()`).
  
   The second set is essential because pending effects target squads the equipping squad doesn't own. Deadlock Shackles, for example, is equipped on the player's squad but its `OnPostReset` needs to run during the *enemy* faction's reset.

- **`DispatchOnAttackComplete(attackerID, defenderID, result)`** — fires `OnAttackComplete` only for behaviors equipped on the attacker.

- **`DispatchOnTurnEnd(round)`** — calls `chargeTracker.RefreshRoundCharges()` first, then fires `OnTurnEnd` on every behavior returned by `AllBehaviors()` (broadcast in deterministic alphabetical order).

### Pipeline Subscription

`combat_service.go` owns a `powercore.PowerPipeline` and subscribes the dispatcher's methods during `NewCombatService` in a declared execution order. `setupPowerDispatch` (in `combat_power_dispatch.go`) has a narrow job: construct the shared `PowerLogger` and inject it into both artifact and perk dispatchers.

```go
// Subscriber order (declared once in NewCombatService):
cs.powerPipeline.OnPostReset(cs.artifactDispatcher.DispatchPostReset)         // Phase 1
cs.powerPipeline.OnPostReset(func(...) { cs.perkDispatcher.DispatchTurnStart(...) }) // Phase 2

cs.powerPipeline.OnAttackComplete(cs.artifactDispatcher.DispatchOnAttackComplete)    // Phase 1
cs.powerPipeline.OnAttackComplete(func(...) { cs.perkDispatcher.DispatchAttackTracking(...) })
cs.powerPipeline.OnAttackComplete(func(...) { cs.onAttackCompleteGUI(...) })         // GUI last

cs.powerPipeline.OnTurnEnd(cs.artifactDispatcher.DispatchOnTurnEnd)                  // Phase 1
cs.powerPipeline.OnTurnEnd(func(round) { cs.perkDispatcher.DispatchRoundEnd(...) })  // Phase 2
cs.powerPipeline.OnTurnEnd(func(...) { cs.onTurnEndGUI(...) })
```

Artifact handlers subscribe first at every event point. This ordering guarantees that artifact effects (e.g., Deadlock Shackles locking a squad) resolve before perk turn-start hooks. The subsystem hooks on `CombatActionSystem`, `MovementSystem`, and `TurnManager` forward directly into the pipeline (`cs.powerPipeline.FirePostReset`, `.FireAttackComplete`, `.FireTurnEnd`, `.FireMoveComplete`).

### Shared Logger

Artifacts and perks share a single `powercore.PowerLogger`. `combat_power_dispatch.go` constructs it once:

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

Behaviors call `ctx.Log(BehaviorKey, squadID, message)`; the logger routes to `[GEAR]` by looking up the `source` in the registered-behavior map. The previous `SetArtifactLogger` package-level callback has been removed.

### Registered Major Artifact Behaviors

| Behavior Key | Struct | Target | Activation | Hooks Overridden | Summary |
|---|---|---|---|---|---|
| `engagement_chains` | `EngagementChainsBehavior` | None | Automatic | `OnAttackComplete` | Grants attacker a full move action after a kill |
| `saboteurs_hourglass` | `SaboteursHourglassBehavior` | None | Player | `Activate`, `OnPostReset` | Queues -2 movement to all enemy squads on their next reset |
| `twin_strike` | `TwinStrikeBehavior` | Friendly | Player | `Activate` | Refunds `HasActed` for a friendly squad that has already attacked |
| `deadlock_shackles` | `DeadlockShacklesBehavior` | Enemy | Player | `Activate`, `OnPostReset` | Queues "fully lock" on a targeted enemy squad for its next turn |
| `chain_of_command` | `ChainOfCommandBehavior` | Friendly | Player | `Activate` | Passes a full action from the wearer squad to a friendly squad that has acted |
| `echo_drums` | `EchoDrumsBehavior` | Friendly | Player | `Activate` | Grants a bonus movement phase to a squad that has already moved + attacked |

---

## Combat Lifecycle Integration

### Battle Start

When `CombatService.InitializeCombat` is called:

1. `cs.chargeTracker.Reset()` clears battle charges, round charges, and pending effects in place. The tracker instance is shared with the `ArtifactDispatcher` and created once in `NewCombatService` — resetting in place preserves the pipeline subscriber bindings rather than swapping the tracker.
2. For each faction, `artifacts.ApplyArtifactStatEffects(factionSquads, manager)` applies all equipped artifact stat modifiers to each unit as permanent `ActiveEffect` entries. This covers both major and minor artifacts, since both may have `statModifiers`.
3. `perks.InitializePerkRoundStatesForFaction(factionSquads, manager)` attaches a fresh `PerkRoundState` to every squad in the faction that has perks.
4. `TurnManager.InitializeCombat(factionIDs)` completes initialization, creating action states and calling `ResetSquadActions` for the first faction. That ResetSquadActions call fires the post-reset hook chain: artifact `DispatchPostReset` runs first, followed by perk `DispatchTurnStart`.

### Battle End

`CombatService.CleanupCombat` removes all active effects from all units (including those applied by artifacts), disposes enemy entities, and returns the player squad IDs so the encounter service can strip cross-cutting combat components. The charge tracker is not disposed — it is reset at the start of the next battle. Artifact equipment assignments on squads persist across battles; the `EquipmentComponent` is not touched during cleanup.

---

## Architecture Summary

```
assets/gamedata/
  major_artifacts.json           — static definitions (id, name, description, tier)
  minor_artifacts.json           — static definitions (id, name, description, tier, statModifiers)
  artifactbalanceconfig.json     — balance tuning values per behavior

templates/
  ArtifactRegistry               — global map[string]*ArtifactDefinition, populated at startup

tactical/powers/powercore/
  context.go                     — PowerContext (shared Manager/Cache/RoundNumber/Logger,
                                   embedded by BehaviorContext and HookContext)
  logger.go                      — PowerLogger interface, LoggerFunc adapter, nil-safe ctx.Log
  pipeline.go                    — PowerPipeline (ordered subscriber lists for PostReset,
                                   AttackComplete, TurnEnd, MoveComplete events)

tactical/powers/artifacts/
  components.go                  — EquipmentData, ArtifactInventoryData, ArtifactInstance,
                                   ArtifactInstanceInfo, Tier constants, ECS component vars
  init.go                        — common.RegisterSubsystem for EquipmentComponent,
                                   ArtifactInventoryComponent
  behavior.go                    — ArtifactBehavior interface, BaseBehavior, BehaviorTargetType
                                   enum, behavior key constants
  behaviors.go                   — all major artifact behavior implementations in one file;
                                   shared helpers requireCharge / activateWithPending /
                                   applyPendingEffects
  context.go                     — BehaviorContext (embeds PowerContext + ChargeTracker);
                                   SetSquadLocked / ResetSquadActions helpers
  pending_effects.go             — PendingEffectQueue, PendingArtifactEffect
                                   (two-phase deferred-effect pattern)
  artifactcharges.go             — ChargeType enum, ArtifactChargeTracker (embeds
                                   PendingEffectQueue via Pending field)
  dispatcher.go                  — ArtifactDispatcher (DispatchPostReset,
                                   DispatchOnAttackComplete, DispatchOnTurnEnd; chargeTracker
                                   required at construction — nil panics)
  registry.go                    — behaviorRegistry, RegisterBehavior, GetBehavior,
                                   IsRegisteredBehavior, AllBehaviors, ActivateArtifact,
                                   CanActivateArtifact, ValidateBehaviorCoverage
  balanceconfig.go               — ArtifactBalanceConfig, LoadArtifactBalanceConfig, validation
  queries.go                     — GetEquipmentData, GetArtifactDefinitions,
                                   GetEquippedBehaviors, HasArtifactBehavior,
                                   GetFactionSquadWithBehavior; inventory query helpers
                                   (CanAddArtifact, OwnsArtifact, IsArtifactAvailable,
                                   GetArtifactCount, GetAllInstances, GetInstanceCount)
  artifactinventory.go           — NewArtifactInventory, GetPlayerArtifactInventory,
                                   totalInstanceCount
  system.go                      — EquipArtifact, UnequipArtifact, ApplyArtifactStatEffects,
                                   inventory mutation helpers (AddArtifactToInventory,
                                   RemoveArtifactFromInventory, MarkArtifactEquipped,
                                   MarkArtifactAvailable)

tactical/combat/combatservices/
  combat_service.go              — owns ArtifactChargeTracker, ArtifactDispatcher,
                                   SquadPerkDispatcher, PowerPipeline; subscribes handlers
                                   in NewCombatService; InitializeCombat calls
                                   chargeTracker.Reset() per battle
  combat_power_dispatch.go       — setupPowerDispatch: constructs shared PowerLogger,
                                   injects it into both dispatchers

gui/guisquads/
  artifact_refresh.go            — ArtifactMode UI for equipping/unequipping artifacts

gui/guiartifacts/
  artifact_handler.go            — in-combat artifact activation handler; imports
                                   BehaviorTargetType from artifacts package
```

### Runtime Data Flow

```
CombatService.InitializeCombat
  └─ chargeTracker.Reset()              (in-place reset; dispatcher binding preserved)
  └─ ApplyArtifactStatEffects           (minor + major stat modifiers → unit ActiveEffects)
  └─ InitializePerkRoundStatesForFaction
  └─ TurnManager.InitializeCombat
       └─ ResetSquadActions (first faction)
            └─ powerPipeline.FirePostReset
                 ├─ artifactDispatcher.DispatchPostReset   (Phase 1, fires first)
                 │    ├─ equipped behaviors for this faction's squads
                 │    └─ behaviors with pending effects (cross-faction)
                 └─ perkDispatcher.DispatchTurnStart       (Phase 2)

CombatActionSystem.ExecuteAttackAction
  └─ powerPipeline.FireAttackComplete
       ├─ artifactDispatcher.DispatchOnAttackComplete      (attacker-equipped only)
       ├─ perkDispatcher.DispatchAttackTracking
       └─ onAttackCompleteGUI

TurnManager.EndTurn
  └─ powerPipeline.FireTurnEnd
       ├─ artifactDispatcher.DispatchOnTurnEnd  (refresh round charges, broadcast OnTurnEnd)
       ├─ perkDispatcher.DispatchRoundEnd
       └─ onTurnEndGUI

MovementSystem.ExecuteMove
  └─ powerPipeline.FireMoveComplete
       ├─ perkDispatcher.DispatchMoveTracking
       └─ onMoveCompleteGUI

Player activates artifact via UI
  └─ artifacts.CanActivateArtifact(behaviorKey, chargeTracker)
  └─ artifacts.ActivateArtifact(behaviorKey, targetSquadID, ctx)
       └─ behavior.Activate(ctx, targetSquadID)
             ├─ requireCharge (returns error if charge spent)
             ├─ [optional] ctx.ChargeTracker.Pending.Add(behaviorKey, targetSquadID)
             ├─ ctx.ChargeTracker.UseCharge(behaviorKey, ChargeType)
             └─ immediate state mutations (or defer via pending queue)
```

### Startup Loading Sequence

```
GameBootstrap.LoadGameData
  └─ templates.ReadGameData()                  (loads artifact JSON definitions)
  └─ perks.LoadPerkDefinitions()
  └─ perks.LoadPerkBalanceConfig()
  └─ artifacts.LoadArtifactBalanceConfig()     (loads artifactbalanceconfig.json)
  └─ artifacts.ValidateBehaviorCoverage()      (cross-checks definitions vs. behaviors)
  └─ perks.ValidateHookCoverage()
```
