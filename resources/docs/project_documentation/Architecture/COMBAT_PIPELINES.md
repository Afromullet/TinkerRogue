# Combat Start and End Pipelines

**Last Updated:** 2026-04-21

This document is a comprehensive technical reference for every combat entry and exit pathway in TinkerRogue. It covers the five known combat entry points, the shared infrastructure that all of them converge on, and the type-specific teardown logic that runs when each combat concludes.

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Core Interfaces and Contracts](#core-interfaces-and-contracts)
4. [Shared Combat Entry Infrastructure](#shared-combat-entry-infrastructure)
5. [Combat Entry Pathways](#combat-entry-pathways)
   - [Pathway 1: Overworld Threat Encounter](#pathway-1-overworld-threat-encounter)
   - [Pathway 2: Garrison Defense](#pathway-2-garrison-defense)
   - [Pathway 3: Raid Room Encounter](#pathway-3-raid-room-encounter)
   - [Pathway 4: Debug "Start Raid" (Roguelike Mode)](#pathway-4-debug-start-raid-roguelike-mode)
   - [Pathway 5: Debug "Start Random Encounter" (Overworld Mode)](#pathway-5-debug-start-random-encounter-overworld-mode)
6. [The Shared Entry Funnel](#the-shared-entry-funnel)
7. [Combat Mode Initialization](#combat-mode-initialization)
8. [Combat End Pathways](#combat-end-pathways)
   - [Victory](#victory)
   - [Defeat](#defeat)
   - [Flee / Retreat](#flee--retreat)
9. [Type-Specific Resolution](#type-specific-resolution)
   - [Overworld Combat Resolution](#overworld-combat-resolution)
   - [Garrison Defense Resolution](#garrison-defense-resolution)
   - [Raid Room Resolution](#raid-room-resolution)
   - [Debug / No-Op Resolution](#debug--no-op-resolution)
10. [Combat Cleanup](#combat-cleanup)
    - [Enemy Cleanup (Dispose)](#enemy-cleanup-dispose)
    - [Player Squad Cleanup (Strip and Return)](#player-squad-cleanup-strip-and-return)
    - [Garrison Defense Special Case](#garrison-defense-special-case)
11. [Post-Combat Return Routing](#post-combat-return-routing)
12. [Dependency Graph](#dependency-graph)
13. [Data Flow Diagrams](#data-flow-diagrams)
14. [Additional Pathways and Edge Cases](#additional-pathways-and-edge-cases)
    - [Save/Load Raid Resume](#saveload-raid-resume)
    - [Raid Retreat and Resume](#raid-retreat-and-resume)
    - [Non-Combat Raid Rooms](#non-combat-raid-rooms)
    - [Post-Combat Listener Cleanup](#post-combat-listener-cleanup)
    - [ResolutionResult vs ResolutionPlan](#resolutionresult-vs-resolutionplan)
15. [Key File Index](#key-file-index)

---

## Executive Summary

TinkerRogue has five distinct combat entry points, but they all funnel into one shared pipeline. The pattern is:

1. A **trigger** (GUI button, turn-end event, or debug menu) creates a type-specific **CombatStarter** struct.
2. `combatlifecycle.ExecuteCombatStart` calls `starter.Prepare()` (which sets up ECS factions and squad positions), then calls `encounterService.TransitionToCombat()` (which saves player state, moves the camera, and switches the UI to combat mode).
3. `CombatMode.Enter()` initializes the turn manager and begins the battle.
4. When the battle ends, `CombatMode.Exit()` calls `encounterService.ExitCombat()` with the outcome.
5. `ExitCombat` dispatches to a type-specific **CombatResolver** (via `resolveEncounterOutcome` → `combatlifecycle.ExecuteResolution`), records history, calls `CombatService.CleanupCombat()` for entity disposal (which returns the player squad IDs for a follow-up `StripCombatComponents` call), and fires the post-combat listener for any registered systems (e.g., `RaidRunner`).

The key design decision is that all type-specific behavior is encoded in small, stateless structs (`CombatStarter` for entry, `CombatResolver` for exit), while the shared infrastructure (`ExecuteCombatStart`, `EncounterService.ExitCombat`, `CombatService.CleanupCombat`) is invariant across all combat types.

---

## Architecture Overview

```
Game Contexts
├── Tactical Context (UIModeManager)
│   ├── ExplorationMode         ← default mode
│   ├── CombatMode              ← battle UI
│   ├── CombatAnimationMode     ← attack animation sub-mode
│   ├── SquadDeploymentMode
│   ├── ProgressionMode         ← perk/spell/artifact unlocks
│   └── [RaidMode]              ← roguelike only
└── Overworld Context (UIModeManager)
    ├── OverworldMode           ← threat map + tick controls
    ├── SquadEditorMode
    ├── NodePlacementMode
    └── ...

Services (created once at startup, shared)
├── EncounterService            ← orchestrates combat lifecycle + history
└── RaidRunner                  ← orchestrates raid loop

ECS Tags queried during combat
├── FactionTag                  ← combat factions
├── TurnStateTag                ← turn manager state
├── ActionStateTag              ← per-squad action budgets
└── CombatFactionTag (FactionMembershipComponent) ← squad → faction links
```

The `GameModeCoordinator` (`gui/framework/coordinator.go`) owns both `UIModeManager` instances and provides the `ModeCoordinator` interface that `EncounterService` uses to switch the active mode to `"combat"` without importing any GUI package directly.

---

## Core Interfaces and Contracts

All combat-lifecycle contracts live in `mind/combatlifecycle/`:
- **`contracts.go`** — entry-side types (`CombatStarter`, `CombatSetup`, `CombatType`, `CombatTransitioner`, `CombatStartRollback`, `EncounterCallbacks`, `CombatCleaner`, `CombatExitReason`, the `PostCombatReturnRaid`/`PostCombatReturnDefault` constants).
- **`pipeline.go`** — resolution types (`CombatResolver`, `ResolutionPlan`, `ResolutionResult`, `ExecuteResolution`).
- **`reward.go`** — generic reward primitives (`Reward`, `GrantTarget`, `Grant`).

### CombatStarter

```go
type CombatStarter interface {
    Prepare(manager *common.EntityManager) (*CombatSetup, error)
}
```

Implemented by:
- `encounter.OverworldCombatStarter` (overworld threats + random debug encounters)
- `encounter.GarrisonDefenseStarter` (garrison defense)
- `raid.RaidCombatStarter` (raid room encounters) — in `campaign/raid/starters.go`

`Prepare` is responsible for creating ECS faction entities, assigning squads to factions, positioning squads on the tactical map, creating `ActionStateData` for each squad, and returning a `CombatSetup` that describes everything the shared pipeline needs.

### CombatSetup

```go
type CombatType int

const (
    CombatTypeOverworld       CombatType = iota // Standard overworld threat encounter
    CombatTypeGarrisonDefense                   // Defending a garrisoned node
    CombatTypeRaid                              // Raid room encounter
    CombatTypeDebug                             // Debug/test encounters
)

type CombatSetup struct {
    PlayerFactionID      ecs.EntityID
    EnemyFactionID       ecs.EntityID
    EnemySquadIDs        []ecs.EntityID
    CombatPosition       coords.LogicalPosition
    EncounterID          ecs.EntityID
    ThreatID             ecs.EntityID
    ThreatName           string
    RosterOwnerID        ecs.EntityID // 0 for garrison defense
    Type                 CombatType
    DefendedNodeID       ecs.EntityID
    PostCombatReturnMode string // PostCombatReturnRaid = "raid"; PostCombatReturnDefault = ""
}
```

`CombatSetup` is the universal handoff packet from type-specific setup to the shared transition. The `CombatType` enum replaces earlier `IsGarrisonDefense`/`IsRaidCombat` bool flags, preventing invalid states (both true) and enabling clean `switch` dispatch. `PostCombatReturnMode` allows the shared infrastructure to route the player to the correct post-combat mode. Typed constants (`PostCombatReturnRaid`, `PostCombatReturnDefault`) are defined in `contracts.go` for compile-time safety.

### CombatTransitioner

```go
type CombatTransitioner interface {
    TransitionToCombat(setup *CombatSetup) error
}
```

Satisfied by `EncounterService` via Go structural typing. Called after `Prepare` succeeds. Records `ActiveEncounter` state and triggers the GUI mode switch.

### CombatStartRollback (optional)

```go
type CombatStartRollback interface {
    Rollback()
}
```

Only `OverworldCombatStarter` implements this. If `TransitionToCombat` fails after `Prepare` has already hidden the encounter sprite, `Rollback` restores the sprite's visibility.

### EncounterCallbacks

```go
type EncounterCallbacks interface {
    ExitCombat(reason CombatExitReason, result *EncounterOutcome, cleaner CombatCleaner)
    GetRosterOwnerID() ecs.EntityID
    GetCurrentEncounterID() ecs.EntityID
}
```

The GUI's narrow view of `EncounterService`. `CombatMode` holds this as an interface (not a concrete type), keeping `gui/guicombat` from importing `mind/encounter`.

### CombatCleaner

```go
type CombatCleaner interface {
    CleanupCombat(enemySquadIDs []ecs.EntityID) []ecs.EntityID
}
```

Satisfied by `CombatService` via structural typing. Called inside `ExitCombat` to dispose enemy entities. The method **returns the player squad IDs** so the caller (`EncounterService`) can strip their combat components separately — this return value exists to avoid an import cycle where `CombatService` would otherwise need to know about `combatlifecycle.StripCombatComponents`.

### CombatResolver

```go
type CombatResolver interface {
    Resolve(manager *common.EntityManager) *ResolutionPlan
}
```

Implemented by:
- `encounter.OverworldCombatResolver` (threat node damage + rewards)
- `encounter.GarrisonDefenseResolver` (node capture or defense success)
- `encounter.FleeResolver` (logs the retreat event)
- `raid.RaidRoomResolver` (marks room cleared, grants rewards) — in `campaign/raid/resolvers.go`
- `raid.RaidDefeatResolver` (sets raid status to defeat) — in `campaign/raid/resolvers.go`

---

## Shared Combat Entry Infrastructure

### ExecuteCombatStart

**File:** `mind/combatlifecycle/starter.go`

```
func ExecuteCombatStart(
    transitioner combatlifecycle.CombatTransitioner,
    manager *common.EntityManager,
    starter combatlifecycle.CombatStarter,
) error
```

Returns only `error`. The same data previously returned by a result struct is already available in `ActiveEncounter` via `CombatSetup`.

This is the single entry point for all combat. Its three-step process:

1. Call `starter.Prepare(manager)` to get a `CombatSetup`.
2. Call `transitioner.TransitionToCombat(setup)` (which is `EncounterService.TransitionToCombat`).
3. If step 2 fails, check if `starter` implements `CombatStartRollback` and call `Rollback()`.

The `transitioner` is always the `EncounterService` instance created at startup (passed as a `combatlifecycle.CombatTransitioner` to avoid import cycles).

### EncounterService.TransitionToCombat

**File:** `mind/encounter/encounter_service.go`

```
func (es *EncounterService) TransitionToCombat(setup *CombatSetup) error
```

Checks that no encounter is already active and that `modeCoordinator` is not nil, then performs these steps inline:

1. Saves the player's current overworld position to `OriginalPlayerPosition`.
2. Calls `modeCoordinator.SetTriggeredEncounterID(encounterID)` and `ResetTacticalState()`.
3. Sets `PostCombatReturnMode` on `TacticalState` if specified — e.g., `combatlifecycle.PostCombatReturnRaid`.
4. Moves the player camera to `setup.CombatPosition`.
5. Calls `modeCoordinator.EnterCombatMode()` → `coordinator.EnterTactical("combat")`.
6. Creates and stores the `ActiveEncounter` record from the setup data.

### CombatFactionManager

**File:** `tactical/combat/combatstate/combatfactionmanager.go`

Used by all starters to create faction entities and assign squads. The key method is `AddSquadToFaction`, which:
- Adds a `CombatFactionData` component to the squad entity.
- Atomically registers or moves the squad's `LogicalPosition` in both the ECS component and the `GlobalPositionSystem`.

### CreateFactionPair

**File:** `mind/combatlifecycle/enrollment.go`

Creates a `CombatQueryCache`, `CombatFactionManager`, and two standard factions in one call:

```go
func CreateFactionPair(manager, playerName, enemyName, encounterID) (*CombatFactionManager, playerFactionID, enemyFactionID)
```

This 3-line sequence appeared in 4 places (overworld setup, garrison encounter, garrison defense starter, raid factions) and is now a single helper.

### EnrollSquadInFaction

**File:** `mind/combatlifecycle/enrollment.go`

The unified 4-step squad enrollment helper used by all starters:

1. `fm.AddSquadToFaction(factionID, squadID, pos)` — faction membership + position
2. `EnsureUnitPositions(manager, squadID, pos)` — all units get positions at squad location
3. `combatstate.CreateActionStateForSquad(manager, squadID)` — combat action tracking
4. Optionally marks squad as deployed (`squadData.IsDeployed = true`)

### EnrollSquadsAtPositions

**File:** `mind/combatlifecycle/enrollment.go`

Batch helper that calls `EnrollSquadInFaction` for each squad/position pair:

```go
func EnrollSquadsAtPositions(fm, manager, factionID, squadIDs, positions, markDeployed) error
```

Replaces the repeated `for i, squadID := range squadIDs { EnrollSquadInFaction(...) }` loops across all setup paths.

### EnsureUnitPositions

**File:** `mind/combatlifecycle/enrollment.go`

Called by `EnrollSquadInFaction` to give every unit in a squad a `LogicalPosition`. Units that already have positions are moved; units without positions have one registered. This is required before combat so that the movement system can find units on the map.

### CreateStandardFactions

**File:** `tactical/combat/combatstate/combatfactionmanager.go`

Factory method that creates the standard player + enemy faction pair:

```go
func (fm *CombatFactionManager) CreateStandardFactions(
    playerFactionName, enemyFactionName string, encounterID ecs.EntityID,
) (playerFactionID, enemyFactionID ecs.EntityID)
```

All starters now call this indirectly via `combatlifecycle.CreateFactionPair`, which wraps cache creation + faction manager creation + `CreateStandardFactions` into a single helper.

---

## Combat Entry Pathways

### Pathway 1: Overworld Threat Encounter

**Trigger:** Player clicks "Engage (E)" in `OverworldMode` while a selected commander is on the same tile as a threat node.

**File chain:**

```
gui/guioverworld/overworld_panels_registry.go            (Engage button)
  → gui/guioverworld/overworld_action_handler.go           EngageThreat()
    → mind/encounter/encounter_trigger.go                  TriggerCombatFromThreat()
      → mind/encounter/encounter_trigger.go                translateThreatToEncounter()
      → mind/encounter/encounter_trigger.go                createOverworldEncounter()
    → encounter.OverworldCombatStarter{...}
    → setup/gamesetup/moderegistry.go                      startCombat closure
      → mind/combatlifecycle/starter.go                    ExecuteCombatStart()
```

**Step-by-step:**

1. `EngageThreat(nodeID)` validates that the commander exists, has a position, and is co-located with the threat node.
2. `TriggerCombatFromThreat` reads the threat node's `OverworldNodeData`, looks up the encounter definition in `core.GetNodeRegistry()`, and creates an `OverworldEncounterData` entity with `ThreatNodeID` set to the threat's entity ID. This `ThreatNodeID` link is critical — it is later used by `OverworldCombatResolver.Resolve` to find the threat node and apply damage to it.
3. `OverworldCombatStarter.Prepare` validates the encounter entity via `encounter.ValidateEncounterEntity`, hides the encounter entity's sprite (stored for rollback), then calls `SpawnCombatEntities`.
4. `SpawnCombatEntities` (returns `*SpawnResult` with `EnemySquadIDs`, `PlayerFactionID`, `EnemyFactionID`) checks whether the threat node has an NPC garrison. If it does, those existing garrison squads become the enemies. If not, it generates enemies from a power budget via `generateAttackerSquads` — which delegates to `mind/spawning/squadscreation.go` for the actual squad composition. Faction creation and squad enrollment run through the shared `assembleCombatFactions` helper in both branches.
5. Power budget generation uses `evaluation.CalculateSquadPower` to measure the player's deployed squads, applies a difficulty multiplier from the encounter's level, and iteratively adds units from a type-filtered pool until the target power is reached.

**CombatSetup produced:**
- `Type = CombatTypeOverworld` (zero value / default)
- `PostCombatReturnMode = ""`  (returns to exploration)
- `RosterOwnerID = commanderID`

### Pathway 2: Garrison Defense

**Trigger:** An NPC faction's tick simulation raids a player-garrisoned node. `commander.EndTurn` returns a `PendingRaid` struct, which the overworld action handler picks up.

**File chain:**

```
gui/guioverworld/overworld_action_handler.go       EndTurn()
  → tickResult.PendingRaid != nil
  → gui/guioverworld/overworld_action_handler.go   HandleRaid()
    → mind/encounter/encounter_trigger.go          TriggerGarrisonDefense()
    → encounter.GarrisonDefenseStarter{...}
    → startCombat closure → ExecuteCombatStart()
```

**Step-by-step:**

1. `TriggerGarrisonDefense` creates an `OverworldEncounterData` entity with `IsGarrisonDefense = true` and `AttackingFactionType` set.
2. `GarrisonDefenseStarter.Prepare`:
   - Validates the encounter entity via `encounter.ValidateEncounterEntity`.
   - Reads the garrison's squad IDs from `garrison.GetGarrisonAtNode` (in `campaign/overworld/garrison`).
   - Creates two factions via `combatlifecycle.CreateFactionPair`; garrison squads join the player faction via `combatlifecycle.EnrollSquadsAtPositions` (they are the defenders), and a fresh set of generated enemy squads joins the enemy faction.
   - Enemy power is calculated from the average garrison squad power, clamped via `encounter.clampPowerTarget`, then multiplied by a difficulty modifier derived from the attacking faction's strength. This ensures the defense is appropriately challenging regardless of the player's current roster.
   - `RosterOwnerID = 0` because there is no commander directing this battle — the garrison defends autonomously.
3. The node's `LogicalPosition` is used as `CombatPosition`.

**CombatSetup produced:**
- `Type = CombatTypeGarrisonDefense`
- `DefendedNodeID = targetNodeID`
- `RosterOwnerID = 0`
- `PostCombatReturnMode = ""` (returns to exploration or overworld depending on active context)

### Pathway 3: Raid Room Encounter

**Trigger:** Player selects a combat room in `RaidMode` and confirms deployment.

**File chain:**

```
gui/guiraid/raidmode.go                     OnDeployConfirmed()
  → gui/guiraid/raidmode.go                 raidRunner.TriggerRaidEncounter(nodeID)
    → campaign/raid/raidrunner.go           TriggerRaidEncounter()
      → raid.RaidCombatStarter{...}
      → mind/combatlifecycle/starter.go     ExecuteCombatStart()
        → encounterService (as transitioner)
```

**Step-by-step:**

1. `TriggerRaidEncounter` snapshots alive unit counts per player squad before starting (stored in `preCombatAliveCounts` for the post-combat summary).
2. It resolves which squads are deployed: if a `DeploymentData` entity exists with `DeployedSquadIDs`, those are used; otherwise all player squad IDs from `RaidStateData` are used.
3. `RaidCombatStarter.Prepare` calls `SetupRaidFactions` (in `campaign/raid/raidencounter.go`) which places player squads at fixed offsets to the left (`playerOffsetX = -3`, `playerOffsetY = -2`) and garrison squads to the right (`enemyOffsetX = 3`, `enemyOffsetY = 2`) of `CombatPosition()` from config. Multiple squads are spread horizontally with `squadSpreadX = 2`.
4. Unlike overworld encounters, raid encounters do not generate new enemy squads. The garrison squads pre-created during `GenerateGarrison` are used directly.

**CombatSetup produced:**
- `Type = CombatTypeRaid`
- `PostCombatReturnMode = combatlifecycle.PostCombatReturnRaid` (returns to raid mode, not exploration)
- `RosterOwnerID = commanderID`
- `EncounterID = raidEntityID` (the raid entity, not an OverworldEncounterData entity)

`RaidCombatStarter` sets `SkipServiceResolution = true` on its `CombatSetup`, which flows through to `ActiveEncounter`. When `EncounterService.ExitCombat` sees that flag it skips overworld resolution and defeat-marking — raid resolution is handled separately by `RaidRunner.ResolveEncounter` via the post-combat listener callback. EncounterService does not check `CombatType` by name for this, so any future combat type with its own resolver can opt into the same behavior just by setting the flag.

### Pathway 4: Debug "Start Raid" (Roguelike Mode)

**Trigger:** Player opens the Debug sub-menu in `ExplorationMode` (roguelike context only) and clicks "Start Raid".

**File chain:**

```
gui/guiexploration/exploration_panels_registry.go       "Start Raid" button OnClick
  → em.ModeManager.RequestTransition(raidMode, "Debug: Start Raid")
    → gui/guiraid/raidmode.go                           Enter(fromMode)
      → raidRunner.IsActive() == false
      → gui/guiraid/raidmode.go                         autoStartRaid()
        → raidRunner.StartRaid(commanderID, playerID, raidSquads, floorCount)
          → campaign/raid/raidrunner.go                 StartRaid()
            → raid.GenerateGarrison(...)                (creates all floors/rooms/garrison squads)
        → raidRunner.EnterFloor(1)
```

This pathway does not immediately start combat. It transitions to `RaidMode`, which auto-generates the garrison and displays the floor map. The player then selects a room and confirms deployment (Pathway 3) to enter actual combat.

**The "Start Raid" button is only reachable in roguelike mode** because the "Debug" button that opens the sub-menu is conditionally shown. `ExplorationPanelActionButtons` checks whether the `squad_editor` tactical mode is registered — it only appears when `squad_editor` is in the tactical context (i.e., roguelike mode). The "Start Raid" button itself also checks `em.ModeManager.GetMode("raid")` to verify that a raid mode is registered before triggering the transition.

The `RaidRunner` registers as a post-combat callback via `encounterService.SetPostCombatCallback(...)` at construction time, so it receives the combat result automatically after each raid room battle. The callback includes a guard: it only calls `ResolveEncounter` when `raidEntityID != 0` AND `raidState.Status == RaidActive`. This prevents cross-contamination if the player retreats from a raid and then triggers an overworld encounter.

### Pathway 5: Debug "Start Random Encounter" (Overworld Mode)

**Trigger:** Player opens the Debug sub-menu in `OverworldMode` and clicks "Start Random Encounter".

**File chain:**

```
gui/guioverworld/overworld_panels_registry.go         "Start Random Encounter" button OnClick
  → gui/guioverworld/overworld_action_handler.go      StartRandomEncounter()
    → mind/encounter/encounter_trigger.go             TriggerRandomEncounter(difficulty=1)
    → encounter.OverworldCombatStarter{ThreatID: 0, ...}
    → startCombat closure → ExecuteCombatStart()
```

**Step-by-step:**

1. `TriggerRandomEncounter` creates an `OverworldEncounterData` entity with `ThreatNodeID = 0`. This zero value is the key distinction: when `ExitCombat` dispatches via `resolveEncounterOutcome`, the `CombatTypeOverworld` case checks `encounterData.ThreatNodeID != 0`, which is false, so no overworld resolution (threat damage, rewards) occurs.
2. `OverworldCombatStarter` is constructed with `ThreatID = 0` and `ThreatName = "Random Encounter"`. The `RosterOwnerID` is the currently selected commander.
3. `SpawnCombatEntities` detects that there is no `ThreatNodeID`, so it skips the garrison check and goes directly to power-budget enemy generation via `mind/spawning/`. With `EncounterType = ""`, the composition falls back to `generateRandomComposition`.

**CombatSetup produced:**
- `Type = CombatTypeOverworld` (zero value / default)
- `ThreatID = 0` (no threat node — combat has no overworld consequences)
- `PostCombatReturnMode = ""`

This pathway is safe to use repeatedly without side effects. Because `ThreatNodeID = 0`, the overworld resolver is skipped for both victory and defeat.

---

## The Shared Entry Funnel

All five pathways converge here:

```
                    Pathway 1 (Overworld Threat)
                    Pathway 2 (Garrison Defense)    ┐
                    Pathway 3 (Raid Room)            ├─ CombatStarter.Prepare()
                    Pathway 4 (Debug Raid)  ──────── ┘         │
                    Pathway 5 (Debug Random)                    ▼
                                               ┌─────────────────────────────┐
                                               │  combatlifecycle.           │
                                               │  ExecuteCombatStart()       │
                                               │  starter.go                 │
                                               └──────────┬──────────────────┘
                                                          │
                                                          ▼
                                               ┌─────────────────────────────┐
                                               │  EncounterService.          │
                                               │  TransitionToCombat()       │
                                               │  encounter_service.go       │
                                               └──────────┬──────────────────┘
                                                          │
                                                          ▼
                                               GameModeCoordinator.
                                               EnterCombatMode()
                                               → EnterTactical("combat")
                                               → UIModeManager.SetMode("combat")
                                               → CombatMode.Enter()
```

---

## Combat Mode Initialization

`CombatMode.Enter()` in `gui/guicombat/combatmode.go` runs the following each time combat starts (when not returning from the animation sub-mode):

1. **Clear stale caches**: `Queries.ClearSquadCache()` purges any cached squad data from the previous battle.
2. **Re-register callbacks**: `registerCombatCallbacks()` attaches fresh GUI hooks for attack-complete, move-complete, and turn-end events on `CombatService` via `SetOnAttackCompleteGUI`/`SetOnMoveCompleteGUI`/`SetOnTurnEndGUI`. These must be re-registered at the start of each battle because the prior combat's teardown cleared them.
3. **Refresh faction visualization**: `visualization.RefreshFactions(Queries)` updates the threat manager with the newly created faction entities.
4. **Start battle recording** (if `ENABLE_COMBAT_LOG_EXPORT` is true): Enables the `BattleRecorder` (`tactical/combat/battlelog/battle_recorder.go`).
5. **Initialize combat factions**: `combatService.InitializeCombat(factionIDs)` which:
   - Resets the `ArtifactChargeTracker` on the `ArtifactDispatcher`.
   - Applies minor artifact stat effects (permanent buffs from gear).
   - Calls `TurnManager.InitializeCombat` which randomizes faction turn order and sets `CombatActive = true`.

Internally, `CombatService` wires the combat subsystems' single-subscriber callbacks to the shared `powercore.PowerPipeline` once, in `NewCombatService`:

```
combatActSystem.SetOnAttackComplete(pipeline.FireAttackComplete)
movementSystem.SetOnMoveComplete(pipeline.FireMoveComplete)
turnManager.SetOnTurnEnd(pipeline.FireTurnEnd)
turnManager.SetPostResetHook(pipeline.FirePostReset)
```

The pipeline subscribers (artifact dispatcher, perk dispatcher, then GUI handlers) are registered in that order and fan out to each registered handler when combat fires an event. See [HOOKS_AND_CALLBACKS.md](HOOKS_AND_CALLBACKS.md) for the full dispatch architecture.

---

## Combat End Pathways

Combat ends in one of three ways. All three routes pass through `CombatMode.Exit()` in `gui/guicombat/combatmode.go`.

### Victory

1. All enemy squads are destroyed.
2. `CombatTurnFlow.CheckAndHandleVictory()` detects `BattleOver = true` from `CheckVictoryCondition()`.
3. `modeManager.RequestTransition(returnMode, ...)` is called with either `"exploration"` or `"raid"` depending on `TacticalState.PostCombatReturnMode`.
4. The mode manager calls `CombatMode.Exit(toMode)`.
5. In `Exit`, the victory result is retrieved from `combatService.GetExitResult()`, the reason is computed via `combatlifecycle.DetermineExitReason(combatService.IsFleeRequested(), victor.IsPlayerVictory)` — returning `ExitVictory` — and `encounterCallbacks.ExitCombat(ExitVictory, outcome, combatService)` is called.

### Defeat

1. All player squads are destroyed.
2. Same flow as Victory, but `result.IsPlayerVictory = false` and `reason = ExitDefeat`.

### Flee / Retreat

1. Player clicks the "Flee" button (wired to `turnFlow.HandleFlee()`).
2. `HandleFlee()` calls `combatService.MarkFleeRequested()` + `combatService.CacheVictoryResult(...)` (a synthetic retreat result) and then `modeManager.RequestTransition(returnMode, "Fled from combat")`.
3. In `CombatMode.Exit`, `combatlifecycle.DetermineExitReason(combatService.IsFleeRequested(), ...)` returns `ExitFlee`.
4. `encounterCallbacks.ExitCombat(ExitFlee, ...)` is called, then `combatService.ClearExitState()` resets the flags for the next battle.

---

## Type-Specific Resolution

`EncounterService.ExitCombat` is the single unified exit point. It makes a value copy of the entire `ActiveEncounter` struct (`enc := *es.activeEncounter`) so that all cleanup steps reference a stable snapshot even after `RecordEncounterCompletion` clears `activeEncounter`. It then dispatches resolution via `resolveEncounterOutcome()` using a `switch` on `CombatType`:

```go
// Simplified from ExitCombat
switch reason {
case ExitVictory, ExitDefeat:
    if combatType != CombatTypeRaid {
        es.resolveEncounterOutcome(encounter, result.IsPlayerVictory)
        // → switch encounter.Type:
        //     CombatTypeGarrisonDefense → GarrisonDefenseResolver
        //     CombatTypeOverworld       → OverworldCombatResolver (if ThreatNodeID != 0)
        //     CombatTypeDebug           → no-op
    }
    if result.IsPlayerVictory && combatType != CombatTypeRaid {
        es.markEncounterDefeated(encounterID)
    }
case ExitFlee:
    es.restoreEncounterSprite(encounterID)
    // dispatches FleeResolver ONLY if ThreatNodeID != 0
}
```

Raid resolution is NOT dispatched here. Instead, the post-combat callback (set by `RaidRunner` via `SetPostCombatCallback`) is fired in the final step of `ExitCombat`, and `RaidRunner.ResolveEncounter` handles the raid-specific dispatch.

### Overworld Combat Resolution

**File:** `mind/encounter/resolvers.go` — `OverworldCombatResolver.Resolve()`

Activated when: `Type = CombatTypeOverworld` and `ThreatNodeID != 0`.

**On player victory:**
- Counts dead enemy units via `combatlifecycle.CountDeadUnits`.
- Converts enemy casualties to intensity damage: every 5 enemies killed = 1 intensity point.
- Reduces `nodeData.Intensity` by the damage amount.
- If intensity reaches 0 or below: calls `threat.DestroyThreatNode` (in `campaign/overworld/threat`) and grants full rewards via `encounter.CalculateIntensityReward(oldIntensity)`.
- If intensity is still positive: grants partial rewards (`rewards.Scale(0.5)`) and resets `nodeData.GrowthProgress = 0.0`.

**On player defeat:**
- Increments `nodeData.Intensity` by 1 (the threat grows stronger after repelling the player).
- Updates the influence radius based on new intensity.
- No rewards granted.

**On flee:**
- Handled by `FleeResolver`: logs an event, no state changes to the threat node.

### Garrison Defense Resolution

**File:** `mind/encounter/resolvers.go` — `GarrisonDefenseResolver.Resolve()`

Activated when: `Type = CombatTypeGarrisonDefense`.

**On player victory (successful defense):**
- Logs a `EventGarrisonDefended` event.
- No rewards are granted (garrison defense is unpaid duty).

**On player defeat (garrison falls):**
- Calls `garrison.TransferNodeOwnership(manager, DefendedNodeID, newOwner)` where `newOwner` is the attacking faction type as a string.
- The node now belongs to the enemy faction.

**Special cleanup:** When `ExitCombat` detects `combatType == CombatTypeGarrisonDefense && result.IsPlayerVictory`, it calls `returnGarrisonSquadsToNode(defendedNodeID)` before `CleanupCombat`. This calls `combatlifecycle.StripCombatComponents` on the garrison squads, removing their `FactionMembershipComponent` and `PositionComponent`, resetting `IsDeployed = false`, but NOT disposing the squad entities. The garrison squads survive and remain in the `garrison.GarrisonData`.

### Raid Room Resolution

**File:** `campaign/raid/resolvers.go`

Activated by `RaidRunner.ResolveEncounter` via the post-combat listener, not by `EncounterService.ExitCombat`'s internal resolution.

**On victory (`RaidRoomResolver.Resolve`):**
- Marks all garrison squad entities in the room as destroyed (`GarrisonSquadData.IsDestroyed = true`).
- Calls `MarkRoomCleared`.
- Checks if the floor is complete via `IsFloorComplete`; sets `FloorStateData.IsComplete = true` if so.
- Calculates room rewards using `calculateRoomReward` and returns a `ResolutionPlan`.

**On defeat or flee (`RaidDefeatResolver.Resolve`):**
- Sets `RaidStateData.Status = RaidDefeat`.
- Returns a `ResolutionPlan` with no rewards.

After resolution, `RaidRunner.PostEncounterProcessing` runs:
1. Applies post-encounter HP recovery (deployed squads recover more than reserve squads).
2. Increments the floor's alert level via `IncrementAlert`.
3. Checks end conditions via `CheckRaidEndConditions` (all player squads destroyed → `RaidDefeat`; final floor complete → `RaidVictory`).

### Debug / No-Op Resolution

Activated when: `Type = CombatTypeOverworld` and `ThreatNodeID = 0`, or `Type = CombatTypeDebug`.

In `resolveEncounterOutcome`, the `CombatTypeOverworld` case checks `encounterData.ThreatNodeID != 0` before creating a resolver. When `ThreatNodeID = 0` (debug random encounter), no resolver is created and no resolution occurs. `CombatTypeDebug` is handled as a no-op in the switch. This means debug encounters have zero side effects on the game world.

---

## Combat Cleanup

`EncounterService.ExitCombat` calls `combatCleaner.CleanupCombat(enemySquadIDs)` where the cleaner is `CombatService`. The method **returns the player squad IDs**, which `EncounterService` then passes to `combatlifecycle.StripCombatComponents` (calling across the package boundary on the way back to avoid a circular import).

### CombatService.CleanupCombat

**File:** `tactical/combat/combatservices/combat_service.go`

Executed in this order:

1. **Cleanup effects**: `cleanupEffects()` removes all active spell/artifact effects from all units tagged `SquadMemberTag`. Prevents buffs and debuffs from persisting into the next battle.

2. **Collect player squad IDs**: `collectPlayerSquadIDs()` walks the player faction's squads and records their IDs. This list is returned to the caller; it is NOT stripped here (see the import-cycle note above).

3. **Build enemy set**: Enemy IDs are converted to a set for fast lookup during unit disposal.

4. **Dispose faction entities**: All entities with `FactionTag` are disposed.

5. **Dispose action state entities**: All entities with `ActionStateTag` are disposed.

6. **Dispose turn state entities**: All entities with `TurnStateTag` are disposed.

7. **Dispose enemy squads**: For each ID in `enemySquadIDs`, calls `manager.CleanDisposeEntity(entity, pos)` which unregisters from `GlobalPositionSystem` before disposal.

8. **Dispose enemy units**: Iterates all `SquadMemberTag` entities, checks if their `SquadMemberData.SquadID` is in the enemy set, and disposes them.

9. **Return**: the player squad ID slice.

After `CleanupCombat` returns, `EncounterService.ExitCombat` calls `combatlifecycle.StripCombatComponents` on those IDs, completing the player-squad teardown (position removal, faction membership removal, `IsDeployed = false`).

GUI callback clearing (attack-complete / move-complete / turn-end) happens separately on combat-mode exit, ensuring stale GUI closures cannot fire against torn-down widgets.

### Enemy Cleanup (Dispose)

Enemy squads and their units are permanently disposed from the ECS world. They were created specifically for this combat encounter (by the spawning package or taken from a pre-generated raid garrison). After disposal, they cease to exist.

**Exception:** For garrison defense victories, the garrison squads are NOT in `enemySquadIDs` at cleanup time because `returnGarrisonSquadsToNode` ran first and stripped their combat components. Since `CleanupCombat` only disposes squads in the provided `enemySquadIDs` list, the garrison squads survive.

For raid defeats, garrison squads in the room being fought are pre-created by `GenerateGarrison` and stored in `RoomData.GarrisonSquadIDs`. These ARE disposed by `CleanupCombat` after a defeat or flee (the room's garrison is treated as an enemy squad list). On victory, `RaidRoomResolver` marks them as `IsDestroyed = true` but they remain in ECS until `CleanupCombat` disposes them.

### Player Squad Cleanup (Strip and Return)

`combatlifecycle.StripCombatComponents` in `mind/combatlifecycle/cleanup.go`:

```go
func StripCombatComponents(manager *common.EntityManager, squadIDs []ecs.EntityID)
```

For each squad:
1. Removes `FactionMembershipComponent` from the squad entity.
2. Calls `manager.UnregisterEntityPosition(entity)` on the squad, which atomically removes the `PositionComponent` and removes the entity from `GlobalPositionSystem`.
3. Does the same for each unit in the squad via `squadcore.GetUnitIDsInSquad`.
4. Resets `SquadData.IsDeployed = false`.

Player squad entities are NOT disposed. They return to their pre-combat state: no position, no faction membership, not deployed. They remain in the player's roster and can be deployed in subsequent battles.

### Garrison Defense Special Case

For garrison defense victories specifically, `ExitCombat` calls `returnGarrisonSquadsToNode(defendedNodeID)` **before** passing `enemySquadIDs` to `CleanupCombat`. In this pathway:

- In `GarrisonDefenseStarter.Prepare`, the garrison squads join the **player faction** and the generated attackers join the **enemy faction**. So `CombatSetup.EnemySquadIDs` contains the generated attacker squads. The garrison squads are in the player faction.
- `CleanupCombat(enemySquadIDs)` disposes the generated attacker squads (the enemies in this context) and returns the player faction's squad IDs (which includes the garrison squads).
- But garrison squads must survive for future use. So before `CleanupCombat`, `returnGarrisonSquadsToNode` calls `StripCombatComponents` on the garrison squads, removing their position and faction membership. When `EncounterService` later calls `StripCombatComponents` on the returned player squad IDs, the garrison squads have no `FactionMembershipComponent` left to strip and are safely skipped.

The garrison squads end up alive in ECS with no position and `IsDeployed = false`, which is exactly the state needed for them to remain in `garrison.GarrisonData.SquadIDs` for future defense.

---

## Post-Combat Return Routing

After `ExitCombat` completes, the UI has already transitioned to the post-combat mode (this transition was requested by `CheckAndHandleVictory` or `HandleFlee` before `Exit` was called by the mode manager). The destination mode is determined by `CombatTurnFlow.getPostCombatReturnMode`:

```go
func (tf *CombatTurnFlow) getPostCombatReturnMode() string {
    tacticalState := tf.context.ModeCoordinator.GetTacticalState()
    if tacticalState.PostCombatReturnMode != "" {
        return tacticalState.PostCombatReturnMode
    }
    return "exploration"
}
```

`TacticalState.PostCombatReturnMode` is set during `TransitionToCombat` if `CombatSetup.PostCombatReturnMode` is non-empty. Currently only raid combat sets this to `combatlifecycle.PostCombatReturnRaid` (`"raid"`).

On entering `RaidMode` from `CombatMode`, `RaidMode.Enter(fromMode)` checks `fromMode.GetModeName() == "combat"` and `raidRunner.LastEncounterResult != nil` to trigger the summary panel display.

---

## Dependency Graph

The following shows import relationships relevant to the combat pipeline. Arrows mean "imports":

```
gui/guicombat
  → tactical/combat/combattypes       (CombatResult, DamageModifiers)
  → tactical/combat/combatservices    (CombatService)
  → mind/combatlifecycle              (CombatExitReason, EncounterCallbacks)
  → mind/ai                           (SetupCombatAI — injected at init time only)

gui/guioverworld
  → mind/encounter                    (EncounterService, OverworldCombatStarter)
  → mind/combatlifecycle              (CombatStarter interface)

gui/guiraid
  → campaign/raid                     (RaidRunner, GetRaidState, etc.)

mind/combatlifecycle
  → tactical/combat/combatstate       (CombatFactionManager, tags/components)
  → tactical/combat/combattypes       (DamageModifiers, CombatResult)
  → tactical/squads/squadcore         (SquadData, GetUnitIDsInSquad)
  → core/common                       (EntityManager)

mind/encounter
  → mind/combatlifecycle              (ExecuteResolution, StripCombatComponents,
                                        EnrollSquadInFaction, CreateFactionPair,
                                        EnrollSquadsAtPositions)
  → tactical/combat/combatstate       (FactionMembershipComponent)
  → tactical/combat/combattypes       (CombatSetup types used indirectly)
  → campaign/overworld/core           (OverworldEncounterData, OverworldNodeData)
  → campaign/overworld/garrison       (GarrisonData, TransferNodeOwnership)
  → campaign/overworld/threat         (DestroyThreatNode)
  → mind/spawning                     (squad composition for power-budget enemies)

campaign/raid
  → mind/combatlifecycle              (ExecuteCombatStart, ExecuteResolution,
                                        ApplyHPRecovery, EnrollSquadInFaction,
                                        CreateFactionPair)
  → mind/encounter                    (EncounterService)
  → tactical/combat/combatstate       (CombatQueryCache, CombatFactionManager)

tactical/combat/combatservices
  → tactical/combat/combatcore        (TurnManager, CombatActionSystem,
                                        CombatMovementSystem)
  → tactical/combat/combatstate       (FactionManager, query cache)
  → tactical/combat/combattypes       (PerkDispatcher interface, CombatResult)
  → tactical/combat/battlelog         (BattleRecorder)
  → tactical/powers/artifacts         (ArtifactDispatcher, ChargeTracker)
  → tactical/powers/effects           (effect application/teardown)
  → tactical/powers/perks             (SquadPerkDispatcher)
  → tactical/powers/powercore         (PowerPipeline, PowerLogger)
  → mind/combatlifecycle              (StripCombatComponents — via return-value protocol)
```

The critical boundary is that `gui/guicombat` never imports `mind/encounter`. The `EncounterCallbacks` interface in `mind/combatlifecycle/contracts.go` is the bridge, satisfied by `EncounterService` via structural typing.

---

## Data Flow Diagrams

### Combat Start: Full Flow

```
[GUI Trigger]
     │
     ▼
Create EncounterID entity
(encounter_trigger.go)
     │
     ▼
Construct CombatStarter
(type-specific struct)
     │
     ▼
ExecuteCombatStart()                          mind/combatlifecycle/starter.go
     │
     ├─→ starter.Prepare(manager)
     │       │
     │       ├─→ [Overworld] SpawnCombatEntities()
     │       │      ├─→ ensurePlayerSquadsDeployed()
     │       │      ├─→ [if garrison present] use garrisonData.SquadIDs as enemies
     │       │      ├─→ [else] generateAttackerSquads()
     │       │      │      └─→ mind/spawning/squadscreation.go
     │       │      │              createSquadForPowerBudget()
     │       │      └─→ assembleCombatFactions() (create pair + enroll both sides)
     │       │
     │       ├─→ [Garrison Defense] GarrisonDefenseStarter.Prepare()
     │       │      ├─→ generateAttackerSquads() (power derived from garrison)
     │       │      └─→ assembleCombatFactions() (garrison as defenders)
     │       │
     │       └─→ [Raid] SetupRaidFactions() (uses pre-created garrison squads)
     │
     └─→ EncounterService.TransitionToCombat(setup)
             │
             ├─→ Save OriginalPlayerPosition
             ├─→ Set PostCombatReturnMode on TacticalState
             ├─→ Move player camera to CombatPosition
             ├─→ GameModeCoordinator.EnterCombatMode()
             │      └─→ UIModeManager.SetMode("combat")
             │             └─→ CombatMode.Enter()
             │                    ├─→ ClearSquadCache()
             │                    ├─→ registerCombatCallbacks()
             │                    ├─→ RefreshFactions()
             │                    └─→ CombatService.InitializeCombat(factionIDs)
             │                           ├─→ ArtifactChargeTracker reset
             │                           ├─→ ApplyArtifactStatEffects
             │                           └─→ TurnManager.InitializeCombat()
             │                                  ├─→ Randomize turn order
             │                                  └─→ CombatActive = true
             └─→ Store ActiveEncounter record
```

### Combat End: Full Flow

```
[Victory/Defeat detected by CombatTurnFlow]
     OR
[Player clicks Flee]
     │
     ▼
CombatTurnFlow: RequestTransition(returnMode)
     │
     ▼
UIModeManager calls CombatMode.Exit(toMode)
     │
     ▼
Determine reason (ExitVictory / ExitDefeat / ExitFlee)
     │
     ▼
EncounterService.ExitCombat(reason, outcome, combatService)
     │
     ├─── Step 1: Resolve outcome (resolveEncounterOutcome)
     │       │   switch encounter.Type:
     │       │
     │       ├─→ [Not raid, not flee] resolveEncounterOutcome()
     │       │      ├─→ [CombatTypeGarrisonDefense] GarrisonDefenseResolver.Resolve()
     │       │      │      └─→ Victory: log event
     │       │      │      └─→ Defeat: TransferNodeOwnership()
     │       │      ├─→ [CombatTypeOverworld] OverworldCombatResolver.Resolve()
     │       │      │      └─→ Victory: DestroyThreatNode() or weaken + rewards
     │       │      │      └─→ Defeat: grow threat intensity
     │       │      └─→ [CombatTypeDebug] no-op
     │       │
     │       └─→ [Flee] restoreEncounterSprite() + FleeResolver.Resolve()
     │
     ├─── Step 2: Mark encounter defeated (victory, non-raid)
     │       └─→ markEncounterDefeated(): IsDefeated=true, hide sprite
     │
     ├─── Step 3: Restore player to OriginalPlayerPosition
     │
     ├─── Step 4: RecordEncounterCompletion()
     │       └─→ Clear activeEncounter
     │
     ├─── Step 5: playerSquadIDs := combatService.CleanupCombat(enemySquadIDs)
     │       ├─→ cleanupEffects()
     │       ├─→ collectPlayerSquadIDs() — returned to caller
     │       ├─→ disposeEntitiesByTag(FactionTag)
     │       ├─→ disposeEntitiesByTag(ActionStateTag)
     │       ├─→ disposeEntitiesByTag(TurnStateTag)
     │       ├─→ disposeEnemySquads(enemySquadIDs)
     │       └─→ disposeEnemyUnits(enemySquadSet)
     │
     ├─── Step 6: combatlifecycle.StripCombatComponents(playerSquadIDs)
     │       (removes position + faction membership, resets IsDeployed)
     │
     └─── Step 7: postCombatCallback(reason, result)
             └─→ [Raid only] RaidRunner.ResolveEncounter()
                    ├─→ Victory: RaidRoomResolver.Resolve()
                    │      └─→ MarkRoomCleared()
                    │      └─→ Grant rewards
                    ├─→ Defeat/Flee: RaidDefeatResolver.Resolve()
                    └─→ PostEncounterProcessing()
                           ├─→ ApplyPostEncounterRecovery()
                           ├─→ IncrementAlert()
                           └─→ CheckRaidEndConditions()
```

---

## Additional Pathways and Edge Cases

### Save/Load Raid Resume

When a player saves and loads mid-raid, the combat pipeline is re-wired without re-generating the garrison.

**File:** `game_main/setup.go` — `SetupRoguelikeFromSave()`

1. `SetupRoguelikeFromSave` calls `setupUICore` to create a fresh `EncounterService` and `GameModeCoordinator`.
2. Creates a new `RaidRunner` via `raid.NewRaidRunner`, which registers the post-combat listener.
3. Checks if a `RaidStateData` entity exists in the loaded ECS world.
4. If found, calls `raidRunner.RestoreFromSave(raidEntityID)`, which sets `rr.raidEntityID` so that `IsActive()` returns true.
5. When `RaidMode.Enter` runs, it sees `raidRunner.IsActive() == true` and skips `autoStartRaid()`, avoiding duplicate entity creation.

This path does NOT trigger combat directly — it restores the raid state so that the player can continue selecting rooms and entering combat via Pathway 3.

### Raid Retreat and Resume

The player can retreat from a raid without ending it. The raid state is preserved for later resumption.

**File:** `campaign/raid/raidrunner.go` — `Retreat()`

1. `Retreat()` sets `RaidStateData.Status = RaidRetreated` and returns.
2. The `RaidRunner` remains active (`raidEntityID != 0`), and `finishRaid` is NOT called — state is preserved. The post-combat callback's guard (`raidState.Status == RaidActive`) prevents any subsequent overworld combat from being processed as a raid encounter.
3. When `RaidMode.Enter` is called again, it detects `raidState.Status == RaidRetreated` and resets it to `RaidActive`, allowing the player to resume selecting rooms.

This is distinct from `RaidDefeatResolver` (which calls `finishRaid` and clears all state) and from the flee-from-combat path (which ends a single room encounter but leaves the overall raid intact).

### Non-Combat Raid Rooms

Not all raid rooms trigger combat. `OnRoomSelected` in `gui/guiraid/raidmode.go` branches based on `room.RoomType`:

- **`GarrisonRoomRestRoom`**: Calls `raidRunner.SelectRoom(nodeID)` directly, which applies HP recovery to deployed squads without entering combat.
- **`GarrisonRoomStairs`**: Also calls `raidRunner.SelectRoom(nodeID)`, which advances to the next floor.
- **Combat rooms**: Show the deployment panel and follow Pathway 3 when confirmed.

### Post-Combat Callback Cleanup

When a raid ends (any outcome — victory, defeat, or all end conditions met), `finishRaid` (`campaign/raid/raidrunner.go`) clears the callback:

```go
rr.encounterService.ClearPostCombatCallback()
rr.raidEntityID = 0
```

This means any combat triggered after the raid ends (e.g., an overworld encounter) will NOT invoke `RaidRunner.ResolveEncounter`. The callback is re-set when a new `RaidRunner` is created (at the next `NewRaidRunner` call or save-load) via `SetPostCombatCallback`.

Additionally, the callback itself includes a guard: it only calls `ResolveEncounter` when the raid state is `RaidActive`. This prevents cross-contamination if `Retreat()` sets the status to `RaidRetreated` but `finishRaid` hasn't been called yet (retreat preserves state for resume).

### ResolutionResult vs ResolutionPlan

The resolution pipeline has two distinct output types in `mind/combatlifecycle/pipeline.go`:

- **`ResolutionPlan`**: Returned by `CombatResolver.Resolve()`. Contains raw `Rewards`, `GrantTarget`, and a `Description`. This is what each resolver produces.
- **`ResolutionResult`**: Returned by `ExecuteResolution()`. Contains the same `Rewards` and `Description`, plus a `RewardText` string produced by calling `Grant()` on the plan's rewards. This is what consumers (like `RaidRunner.ResolveEncounter`) read.

`ExecuteResolution` bridges the two: it calls `resolver.Resolve()` to get a `ResolutionPlan`, calls `Grant()` to distribute rewards and generate human-readable text, and wraps both into a `ResolutionResult`.

---

## Key File Index

| File | Purpose |
|------|---------|
| `mind/combatlifecycle/contracts.go` | All shared interfaces: `CombatStarter`, `CombatSetup`, `CombatType`, `CombatTransitioner`, `EncounterCallbacks`, `CombatCleaner`, `CombatExitReason`, `PostCombatReturnRaid`/`PostCombatReturnDefault` constants |
| `mind/combatlifecycle/starter.go` | `ExecuteCombatStart`: the single entry point for all combat initiation |
| `mind/combatlifecycle/pipeline.go` | `CombatResolver`, `ResolutionPlan`, `ResolutionResult`, `ExecuteResolution`: the single entry point for all combat resolution |
| `mind/combatlifecycle/enrollment.go` | `CreateFactionPair`, `EnrollSquadInFaction`, `EnrollSquadsAtPositions`, `EnsureUnitPositions`: faction creation and squad enrollment helpers |
| `mind/combatlifecycle/cleanup.go` | `StripCombatComponents`: strips combat state from player squads without disposing them |
| `mind/combatlifecycle/reward.go` | `Reward`, `Grant`, `GrantTarget`: generic reward calculation and distribution primitives |
| `mind/combatlifecycle/casualties.go` | `GetLivingUnitIDs`, `CountDeadUnits`: casualty counting helpers |
| `mind/encounter/encounter_service.go` | `EncounterService`: tracks `ActiveEncounter`, implements `TransitionToCombat`, `ExitCombat`, `SetPostCombatCallback` |
| `mind/encounter/encounter_config.go` | `clampPowerTarget`: encounter power-clamping helper (encounter-only; raid uses archetypes) |
| `mind/encounter/validators.go` | `ValidateEncounterEntity`: validates encounter entity + OverworldEncounterData |
| `mind/encounter/rewards.go` | `CalculateIntensityReward`: threat-intensity-based reward calculation |
| `mind/encounter/starters.go` | `OverworldCombatStarter`, `GarrisonDefenseStarter`: two of the three `CombatStarter` implementations |
| `mind/encounter/encounter_trigger.go` | `TriggerCombatFromThreat`, `TriggerRandomEncounter`, `TriggerGarrisonDefense`: creates encounter entities |
| `mind/encounter/encounter_setup.go` | `SpawnCombatEntities` (returns `*SpawnResult`), `generateAttackerSquads`, `assembleCombatFactions`: combat entity creation |
| `mind/encounter/resolvers.go` | `OverworldCombatResolver`, `GarrisonDefenseResolver`, `FleeResolver` |
| `mind/encounter/types.go` | `ActiveEncounter`, `CompletedEncounter`, `SpawnResult`, `ModeCoordinator` interface |
| `mind/spawning/squadscreation.go` | `createSquadForPowerBudget`: power-budget squad generation used by encounter setup |
| `mind/spawning/composition.go` | `generateRandomComposition` and faction-typed composition helpers |
| `campaign/raid/starters.go` | `RaidCombatStarter`: raid-specific `CombatStarter` implementation |
| `campaign/raid/raidencounter.go` | `SetupRaidFactions`: positions squads for raid combat |
| `campaign/raid/raidrunner.go` | `RaidRunner`: orchestrates the full raid loop, registered as post-combat listener |
| `campaign/raid/resolvers.go` | `RaidRoomResolver`, `RaidDefeatResolver` |
| `tactical/combat/combatservices/combat_service.go` | `CombatService.CleanupCombat` (returns `[]ecs.EntityID`), `InitializeCombat`, `CheckVictoryCondition` |
| `tactical/combat/combatservices/combat_power_dispatch.go` | Logger wiring and perk dispatcher injection into `CombatActionSystem` |
| `tactical/combat/combatstate/combatfactionmanager.go` | `CombatFactionManager.CreateStandardFactions`, `CreateFactionWithPlayer`, `AddSquadToFaction` |
| `tactical/combat/combatstate/combatqueries.go` | `CreateActionStateForSquad`, `RemoveSquadFromMap`, `GetAllFactions`, `GetSquadsForFaction` |
| `tactical/combat/combatstate/combatcomponents.go` | `FactionMembershipComponent`, `ActionStateComponent`, tags |
| `tactical/combat/combatcore/turnmanager.go` | `TurnManager`: turn order, `InitializeCombat`, `EndTurn`, `ResetSquadActions` |
| `tactical/combat/combatcore/combatactionsystem.go` | `CombatActionSystem.ExecuteAttackAction`: the per-attack damage pipeline |
| `tactical/combat/combatcore/combatmovementsystem.go` | `CombatMovementSystem`: per-squad movement during combat |
| `tactical/combat/combattypes/combattypes.go` | `CombatResult`, `DamageModifiers`, `CoverBreakdown`, `EncounterOutcome` |
| `tactical/combat/combattypes/perk_callbacks.go` | `PerkDispatcher` interface (implemented by `perks.SquadPerkDispatcher`) |
| `tactical/combat/battlelog/battle_recorder.go` | `BattleRecorder` for `ENABLE_COMBAT_LOG_EXPORT` |
| `tactical/powers/powercore/pipeline.go` | `PowerPipeline` (shared subscriber lists used by `CombatService` to fan out events) |
| `tactical/powers/artifacts/dispatcher.go` | `ArtifactDispatcher`: DispatchPostReset / DispatchOnAttackComplete / DispatchOnTurnEnd |
| `tactical/powers/perks/dispatcher.go` | `SquadPerkDispatcher`: implements `combattypes.PerkDispatcher` + lifecycle `Dispatch*` methods |
| `gui/guicombat/combatmode.go` | `CombatMode.Enter` (init), `CombatMode.Exit` (calls ExitCombat) |
| `gui/guicombat/combat_turn_flow.go` | `CheckAndHandleVictory`, `HandleFlee`, `completeTurn`, `getPostCombatReturnMode` |
| `gui/guicombat/combatdeps.go` | `CombatModeDeps`: dependency container, holds `EncounterCallbacks` interface |
| `gui/guioverworld/overworld_action_handler.go` | `EngageThreat`, `HandleRaid`, `StartRandomEncounter` |
| `gui/guioverworld/overworld_panels_registry.go` | Debug panel with "Start Random Encounter" button |
| `gui/guiexploration/exploration_panels_registry.go` | Debug panel with "Start Raid" button (roguelike only) |
| `gui/guiraid/raidmode.go` | `OnDeployConfirmed`, `OnCombatComplete`, `autoStartRaid` |
| `gui/framework/coordinator.go` | `GameModeCoordinator`: implements `ModeCoordinator`, manages context switching |
| `setup/gamesetup/moderegistry.go` | Wires `EncounterService` to `startCombat` closure (returns `error`) for overworld modes; registers raid mode |
| `game_main/setup.go` | `SetupRoguelikeMode`, `SetupOverworldMode`, `SetupRoguelikeFromSave`: top-level wiring of services and modes |
