# Artifact System Architecture

**Last Updated:** 2026-02-17

---

## Overview

Artifacts are equippable items in TinkerRogue that augment squad capabilities during combat. They represent powerful relics, tools, and banners that a player collects, stockpiles in an inventory, and assigns to squads before entering battle.

The system is split along a clean architectural boundary: **minor artifacts** are passive and apply permanent stat modifiers at battle start, while **major artifacts** carry active behaviors that fire in response to combat events or direct player input.

---

## The Two Categories

### Minor Artifacts

Minor artifacts are purely passive. Each minor artifact definition carries a list of `statModifiers` — positive or negative changes to unit stats such as armor, weapon power, movement speed, attack range, strength, or dexterity. When combat begins, `ApplyArtifactStatEffects` in `gear/system.go` iterates every squad in every faction and applies each equipped artifact's modifiers to all units in that squad as permanent `ActiveEffect` entries (duration `-1`). From that point forward, the effects simply exist on the units for the duration of the battle; there is nothing to trigger and no state to track.

Minor artifacts have no `behavior` field in their definition. Their tier is `"minor"`.

### Major Artifacts

Major artifacts are active and behavioral. Instead of static modifiers, a major artifact carries a `behavior` key that ties the artifact's definition to a registered `ArtifactBehavior` implementation. The behavior governs:

- When the artifact fires (which combat hook)
- Whether it requires player activation or fires automatically
- What target it requires (friendly squad, enemy squad, or none)
- What game-state mutations it performs when triggered
- Whether its charge refreshes each round or is once-per-battle

Major artifact definitions may optionally include `statModifiers` as well, meaning a single artifact can combine passive stat changes with active effects.

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

Separately from the definition registry, each major artifact has a corresponding Go behavior implementation. Implementations register themselves via `gear.RegisterBehavior()` during Go's `init()` phase, populating `gear.behaviorRegistry` — a global map from behavior key to `ArtifactBehavior` interface.

The `init()` functions in `gear/artifactbehaviors_activated.go` and `gear/artifactbehaviors_passive.go` each call `RegisterBehavior` for the behaviors they implement. This follows the same self-registration pattern used by worldmap generators and ECS subsystems throughout the codebase.

### ECS Component Registration

The `gear` package's own `init()` in `gear/init.go` uses `common.RegisterSubsystem` to create the `EquipmentComponent` and `ArtifactInventoryComponent` ECS component types when the entity manager is initialized. This follows the standard subsystem self-registration pattern from `CLAUDE.md`.

---

## Attaching Artifacts to Entities

### Player Inventory

The player entity is created with an `ArtifactInventoryComponent` during `InitializePlayerData` in `game_main/gameinit.go`. The inventory's `MaxArtifacts` is set from `config.DefaultPlayerMaxArtifacts`. Artifacts are added to this inventory via `gear.AddArtifactToInventory`, which appends an `ArtifactInstance{EquippedOn: 0}` entry.

### Equipping to a Squad

Equipping an artifact calls `gear.EquipArtifact(playerID, squadID, artifactID, manager)`. This function:

1. Looks up the definition in `ArtifactRegistry` to verify it exists.
2. Confirms the player owns at least one unequipped copy via the inventory component.
3. Gets or lazily creates the squad's `EquipmentComponent`.
4. Checks that the squad has not exceeded `config.DefaultMaxArtifactsPerCommander`.
5. Appends the artifact ID to `EquipmentData.EquippedArtifacts`.
6. Marks the specific inventory instance as equipped (sets `EquippedOn = squadID`).

Steps 5 and 6 are transactional: if step 6 fails, the change from step 5 is rolled back.

Unequipping reverses this process via `gear.UnequipArtifact`, again with rollback on failure.

### Squad Management UI

The `gui/guisquads` package provides the `ArtifactMode` UI screen through which the player browses their inventory and assigns artifacts to squads before deployment. The equipment and inventory tabs both call into the functions above and refresh their lists after each operation.

---

## How Major Artifact Behaviors Work

### The ArtifactBehavior Interface

The `gear.ArtifactBehavior` interface, defined in `gear/artifactbehavior.go`, is the contract every major artifact behavior must satisfy:

```
BehaviorKey()       — returns the behavior's string identifier
TargetType()        — returns TargetNone, TargetFriendly, or TargetEnemy
IsPlayerActivated() — returns true if a player must explicitly trigger this
Activate()          — called when a player manually triggers the behavior
OnPostReset()       — hook fired after a faction's squads have their actions reset
OnAttackComplete()  — hook fired after every successful attack resolves
OnTurnEnd()         — hook fired at the end of each complete round
```

The `gear.BaseBehavior` struct provides no-op default implementations of every method so concrete behaviors only need to override the hooks they actually use.

### The Three Hooks

**OnPostReset** fires immediately after the `TurnManager.ResetSquadActions` call completes for a faction. This is the start-of-turn moment when squads have their `HasMoved`, `HasActed`, and `MovementRemaining` flags freshly initialized. Behaviors that want to modify a faction's initial action budget for a turn — for example, reducing enemy movement or locking an enemy squad out before it acts — do so here.

**OnAttackComplete** fires after every attack resolves, regardless of which faction is attacking. The hook receives the attacker's ID, the defender's ID, and a `CombatResult` describing what happened (whether the target was destroyed, damage dealt, etc.). Behaviors that react to the outcome of an attack — such as granting bonus movement after a kill — implement this hook.

**OnTurnEnd** fires after the turn index advances and a full round completes. Per-round charges are refreshed at this moment (before the hook fires), so behaviors can assume round charges have been cleared. This hook is available for any future behaviors that need to act at round boundaries.

### Charge Tracking

The `gear.ArtifactChargeTracker`, created fresh at the start of each battle in `CombatService.InitializeCombat`, tracks which behaviors have been used. Two granularities are supported:

- **ChargeOncePerBattle** — the behavior cannot be used again once spent for the entire battle.
- **ChargeOncePerRound** — the behavior's charge refreshes when `OnTurnEnd` fires, so it can be used once per full round.

`CanActivateArtifact` gates all player-triggered activation by checking `ArtifactChargeTracker.IsAvailable`. The charge tracker also supports **pending effects**: some behaviors queue a deferred effect on activation (via `AddPendingEffect`) rather than applying it immediately, then consume those queued effects later inside an `OnPostReset` hook. This is the mechanism that allows an artifact activated during the player's turn to apply its effect at the start of the enemy's next turn.

### Player-Activated vs. Automatic Behaviors

Behaviors where `IsPlayerActivated()` returns `true` appear in the in-combat artifact panel (`gui/guiartifacts/artifact_handler.go`). The player opens artifact mode, selects one of these behaviors, optionally clicks a target squad (if `TargetType()` is not `TargetNone`), and the handler calls `gear.ActivateArtifact`. The charge gate is checked before `Activate` is dispatched.

Behaviors where `IsPlayerActivated()` returns `false` never appear in the activation UI. They fire automatically whenever their hook condition is met — the system calls every registered behavior's hook after each relevant combat event regardless of whether the behavior is player-activated, and the behavior's own implementation decides whether to act based on its own preconditions (such as checking whether the attacking squad actually has this artifact equipped).

### Hook Dispatch Wiring

`combatservices.setupBehaviorDispatch` (in `tactical/combatservices/combat_service.go`) wires the hook calls at `CombatService` construction time. It registers three callbacks with the combat event system:

- A post-reset callback that iterates `gear.AllBehaviors()` and calls each behavior's `OnPostReset`.
- An attack-complete callback that iterates all behaviors and calls `OnAttackComplete`.
- A turn-end callback that first refreshes round charges on the tracker, then calls `OnTurnEnd` on all behaviors.

`gear.AllBehaviors()` returns all registered behaviors in deterministic alphabetical order. Each hook call is broadcast to all behaviors unconditionally; each behavior is responsible for checking whether it applies to the current situation (for example, by checking which squads carry the artifact or whether charges are available).

---

## Combat Lifecycle Integration

### Battle Start

When `CombatService.InitializeCombat` is called:

1. A fresh `ArtifactChargeTracker` is created and stored on the service.
2. `gear.ApplyArtifactStatEffects` is called for each faction's squads, applying all equipped artifact stat modifiers to each unit as permanent `ActiveEffect` entries. This covers both major and minor artifacts, since both may have `statModifiers`.
3. The `TurnManager` is initialized, action states are created for all squads, and `ResetSquadActions` is called for the first faction — triggering the first `OnPostReset` hook dispatch.

### Battle End

`CombatService.CleanupCombat` removes all active effects from all units (including those applied by artifacts), removes combat components, and disposes enemy entities. The charge tracker is implicitly discarded since it lives on the `CombatService` and is replaced at the next `InitializeCombat` call. Artifact equipment assignments on squads persist across battles; the `EquipmentComponent` is not touched during cleanup.

---

## Architecture Summary

```
assets/gamedata/
  major_artifacts.json     — static definitions (id, name, description, tier)
  minor_artifacts.json     — static definitions (id, name, description, tier, statModifiers)

templates/
  ArtifactRegistry         — global map[string]*ArtifactDefinition, populated at startup

gear/
  components.go            — EquipmentData, ArtifactInventoryData, ArtifactInstance (ECS data)
  init.go                  — registers ECS components via common.RegisterSubsystem
  artifactbehavior.go      — ArtifactBehavior interface, BaseBehavior, behavior registry
  artifactbehaviors_activated.go  — player-triggered behavior implementations (self-register)
  artifactbehaviors_passive.go    — event-driven behavior implementations (self-register)
  artifactcharges.go       — ArtifactChargeTracker, PendingArtifactEffect
  queries.go               — read-only accessors (GetEquipmentData, HasArtifactBehavior, etc.)
  system.go                — mutation functions (EquipArtifact, UnequipArtifact,
                             ApplyArtifactStatEffects, inventory management)

tactical/combatservices/
  combat_service.go        — owns ArtifactChargeTracker, calls setupBehaviorDispatch
  combat_events.go         — hook registration API for the CombatService

gui/guisquads/
  artifact_refresh.go      — ArtifactMode UI for equipping/unequipping artifacts on squads

gui/guiartifacts/
  artifact_handler.go      — in-combat artifact activation handler (selection, targeting, execution)
```

The data flow at combat time is:

```
CombatService.InitializeCombat
  └─ ApplyArtifactStatEffects        (minor + major stat modifiers → unit ActiveEffects)
  └─ TurnManager.ResetSquadActions
       └─ postResetHook → AllBehaviors().OnPostReset (each behavior checks its preconditions)

CombatActionSystem.ExecuteAttackAction
  └─ onAttackComplete → AllBehaviors().OnAttackComplete

TurnManager.EndTurn
  └─ onTurnEnd → ChargeTracker.RefreshRoundCharges
               → AllBehaviors().OnTurnEnd

Player activates artifact via UI
  └─ gear.CanActivateArtifact (charge check)
  └─ gear.ActivateArtifact → behavior.Activate(ctx, targetSquadID)
```
