# Combat Start and End Pipelines

**Last Updated:** 2026-05-15

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
    - [ResolutionResult fields](#resolutionresult-fields)
15. [Key File Index](#key-file-index)

---

## Executive Summary

TinkerRogue has five distinct combat entry points, but they all funnel into one shared pipeline. The pattern is:

1. A **trigger** (GUI button, turn-end event, or debug menu) creates a type-specific **CombatStarter** struct.
2. `combatlifecycle.ExecuteCombatStart` calls `starter.Prepare()` (which sets up ECS factions and squad positions), then calls `encounterService.TransitionToCombat()` (which saves player state, moves the camera, and switches the UI to combat mode).
3. `CombatMode.Enter()` initializes the turn manager and begins the battle.
4. When the battle ends, `CombatMode.Exit()` calls `encounterService.ExitCombat()` with the outcome.
5. `ExitCombat` runs the setup's resolver via `combatlifecycle.ExecuteResolution`, records history, calls `CombatService.TeardownCombat()` for entity disposal (which internally strips combat-only state from player squads — faction membership, perk round state, positions, IsDeployed), and fires the post-combat listener for any registered systems (e.g., `RaidRunner`).

The key design decision is that all type-specific behavior is encoded in small, stateless structs (`CombatStarter` for entry, `CombatResolver` for exit), while the shared infrastructure (`ExecuteCombatStart`, `EncounterService.ExitCombat`, `CombatService.TeardownCombat`) is invariant across all combat types.

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

All combat-lifecycle contracts live in `mind/combatlifecycle/`, split by lifecycle phase:
- **`start.go`** — start-side types and orchestration (`CombatStarter`, `CombatSetup` with its derived `PostCombatReturnMode()` method, `CombatType`, `CombatTransitioner`, `ExecuteCombatStart`, the `PostCombatReturnRaid`/`PostCombatReturnDefault` constants).
- **`exit.go`** — exit-side types and orchestration (`CombatExitReason`, `EncounterOutcome`, `ResolutionContext`, `CombatResolver`, `ResolutionResult`, `ExitHooks`, `ExecuteResolution`, `ExecuteCombatExit`).
- **`teardown.go`** — `CombatTeardown` interface plus the `StripCombatComponents` and `ApplyHPRecovery` helpers.
- **`reward.go`** — generic reward primitives (`Reward`, `GrantTarget`, `Grant`).

`EncounterController` lives in `mind/encounter/types.go` — it is not a shared contract, only a GUI↔encounter port.

### CombatStarter

```go
type CombatStarter interface {
    // Returns the prepared CombatSetup and an optional rollback closure.
    // The closure (when non-nil) undoes Prepare's side effects if
    // TransitionToCombat later fails. Starters with nothing to undo return nil.
    Prepare(manager *common.EntityManager) (*CombatSetup, func(), error)
}
```

Implemented by:
- `encounter.OverworldCombatStarter` (overworld threats + random debug encounters) — returns a closure that restores the encounter sprite.
- `encounter.GarrisonDefenseStarter` (garrison defense) — no side effects; returns `nil`.
- `raid.RaidCombatStarter` (raid room encounters, in `campaign/raid/starters.go`) — no side effects; returns `nil`.

`Prepare` is responsible for creating ECS faction entities, assigning squads to factions, positioning squads on the tactical map, creating `ActionStateData` for each squad, and returning a `CombatSetup` that describes everything the shared pipeline needs. The rollback closure consolidates what was previously a separate `CombatStartRollback` interface — co-locating the undo handle with the resource that needs it, captured in a closure rather than tracked as a struct field.

### CombatSetup

```go
type CombatType int

const (
    CombatTypeOverworld       CombatType = iota // Standard overworld threat encounter
    CombatTypeGarrisonDefense                   // Defending a garrisoned node
    CombatTypeRaid                              // Raid room encounter
)

type CombatSetup struct {
    PlayerFactionID ecs.EntityID
    EnemyFactionID  ecs.EntityID
    EnemySquadIDs   []ecs.EntityID
    CombatPosition  coords.LogicalPosition
    EncounterID     ecs.EntityID
    ThreatID        ecs.EntityID
    ThreatName      string
    RosterOwnerID   ecs.EntityID // 0 for garrison defense
    Type            CombatType
    DefendedNodeID  ecs.EntityID
    Resolver        CombatResolver // built eagerly by Prepare; runtime state arrives via ResolutionContext
}

// PostCombatReturnMode is a method, not a field — derived from Type.
func (s *CombatSetup) PostCombatReturnMode() string { ... }
```

`CombatSetup` is the universal handoff packet from type-specific setup to the shared transition. The `CombatType` enum replaces earlier `IsGarrisonDefense`/`IsRaidCombat` bool flags, preventing invalid states (both true) and enabling clean `switch` dispatch. `PostCombatReturnMode()` is a method on `CombatSetup` that derives the return-mode key from `Type` (raid → `PostCombatReturnRaid`, all others → `PostCombatReturnDefault`); the typed constants are defined alongside the method in `start.go`. Storing the value as a derived method rather than a field prevents `Type` and the return mode from drifting out of sync.

### CombatTransitioner

```go
type CombatTransitioner interface {
    TransitionToCombat(setup *CombatSetup) error
}
```

Satisfied by `EncounterService` via Go structural typing. Called after `Prepare` succeeds. Records `ActiveEncounter` state and triggers the GUI mode switch.

### EncounterController

```go
type EncounterController interface {
    ExitCombat(reason CombatExitReason, result *EncounterOutcome, cleaner CombatTeardown)
    GetRosterOwnerID() ecs.EntityID
    GetCurrentEncounterID() ecs.EntityID
}
```

The GUI's narrow view of `EncounterService`. `CombatMode` holds this as an interface (not a concrete type), so the GUI depends only on the `EncounterController` surface area — not on `EncounterService`'s full implementation. Defined in `mind/encounter/types.go`.

### CombatTeardown

```go
type CombatTeardown interface {
    TeardownCombat(enemySquadIDs []ecs.EntityID)
}
```

Satisfied by `CombatService` via structural typing. Called inside `ExitCombat` to dispose enemy entities and strip combat-only state from player squads. The implementation calls `combatstate.RemoveCombatMembership`, `perks.RemovePerkRoundState`, and `squadcore.ResetSquadDeployment` for each player squad before disposing faction entities. Each owning package exports its own cleanup helper; `CombatService` invokes them directly — no upward import into `mind/combatlifecycle` is needed because `combat_service.go` already imports `combatstate`, `perks`, and `squadcore`.

### CombatResolver

```go
type CombatResolver interface {
    Resolve(manager *common.EntityManager, ctx ResolutionContext) *ResolutionResult
}

// ResolutionContext is the single value-struct carrying everything the exit
// pipeline needs. Resolvers, ExitHooks, and ExecuteCombatExit all receive it.
type ResolutionContext struct {
    Setup CombatSetup // starter-built snapshot; resolvers should leave this to orchestration

    Reason         CombatExitReason
    Outcome        *EncounterOutcome // non-nil; built by the GUI layer
    PlayerEntityID ecs.EntityID
    PlayerSquadIDs []ecs.EntityID    // snapshotted up-front to survive activeEncounter being cleared

    OriginalPlayerPosition coords.LogicalPosition // consumed by OnRestorePlayer
}
```

Resolvers are constructed eagerly by `CombatStarter.Prepare()` and attached to the `CombatSetup.Resolver` field. Runtime information (exit reason, outcome, player entity, player squads) arrives via `ResolutionContext` at exit time so resolvers don't need to capture it in closures. Each combat type has exactly one resolver — branches for flee/victory/defeat are inlined inside `Resolve` rather than dispatched to sub-resolvers.

`ResolutionContext` is role-phased rather than type-segregated. Resolvers read the runtime-state fields (`Reason`, `Outcome.IsPlayerVictory`, `PlayerEntityID`, `PlayerSquadIDs`); they should treat `Setup` and `OriginalPlayerPosition` as opaque (setup data belongs in resolver instance fields captured at `Prepare` time). Hooks read whatever they need — typically `Setup.Type`, `Setup.EncounterID`/`DefendedNodeID`, `Outcome`, and `OriginalPlayerPosition`.

Implemented by:
- `encounter.OverworldCombatResolver` — threat-node damage + rewards on victory, intensity growth on defeat, log-only flee branch inlined.
- `encounter.GarrisonDefenseResolver` — node capture on defeat, defense-success log on victory.
- `raid.RaidEncounterResolver` (`campaign/raid/resolvers.go`) — victory branch marks room cleared + grants rewards; defeat/flee branch sets `RaidStateData.Status = RaidDefeat`.

### ResolutionResult

```go
type ResolutionResult struct {
    Rewards     Reward
    Target      GrantTarget // input to ExecuteResolution; zero-valued after Grant runs
    RewardText  string      // output from ExecuteResolution; empty when read by the resolver
    Description string      // resolver-supplied summary
}
```

`ResolutionResult` is lifecycle-phased: the resolver fills `Rewards`/`Target`/`Description`, then `ExecuteResolution` calls `Grant(Rewards, Target)` and writes the human-readable summary into `RewardText` in place. Consumers (e.g., `RaidRunner.ResolveEncounter` via `postCombatCallback`) read `Rewards`, `RewardText`, and `Description`. `Target` lingers on the consumer-facing struct but is harmless — it is documented as "intermediate state, consumed by `ExecuteResolution`".

### ExecuteCombatExit and ExitHooks (symmetric exit pipeline)

`mind/combatlifecycle/exit.go` exposes a single exit-side entry point that mirrors `ExecuteCombatStart` on the start side:

```go
func ExecuteCombatExit(
    manager *common.EntityManager,
    ctx ResolutionContext,
    hooks ExitHooks,
    teardown CombatTeardown,
    onHistory func(*ResolutionResult),
) *ResolutionResult
```

The caller (`EncounterService.ExitCombat`) builds the `ResolutionContext` value once from its `activeEncounter` (capturing `PlayerSquadIDs` from the roster up front via `collectPlayerSquadIDs` so the resolver sees a stable list even after `activeEncounter` is cleared), then passes it by value through the entire orchestration. Pass-by-value means the orchestration holds no reference into the encounter package — no import cycle, and no race with `activeEncounter` being cleared mid-flow by `onHistory`.

Sequencing (any nil dependency is skipped):

```
EncounterService.ExitCombat
    └─▶ ExecuteCombatExit(manager, ctx, hooks, teardown, onHistory)
            1. ExecuteResolution(manager, ctx.Setup.Resolver, ctx) → *ResolutionResult
            2. hooks.OnAfterResolution(ctx)             (sprite restore on flee, mark-defeated on victory)
            3. hooks.OnRestorePlayer(ctx)               (camera back to pre-encounter position)
            4. onHistory(result)                         (history record, clear activeEncounter)
            5. hooks.OnBeforeTeardown(ctx)              (garrison-defense return-to-node)
            6. teardown.TeardownCombat(ctx.Setup.EnemySquadIDs) (entity disposal)
    └─▶ postCombatCallback(reason, outcome, result)     (caller fires after orchestration returns)
```

#### ExitHooks

```go
type ExitHooks interface {
    OnAfterResolution(ctx ResolutionContext)
    OnRestorePlayer(ctx ResolutionContext)
    OnBeforeTeardown(ctx ResolutionContext)
}
```

The encounter package supplies `EncounterExitHooks` (`mind/encounter/exit_hooks.go`). Raid does not need its own hooks — its post-combat work happens via the `postCombatCallback` that `EncounterService.ExitCombat` fires after `ExecuteCombatExit` returns.

`EncounterExitHooks` methods:

- **`OnAfterResolution`** — on flee, restore the encounter sprite; on overworld/garrison victory (not raid), mark the encounter defeated and hide the sprite permanently.
- **`OnRestorePlayer`** — write `state.OriginalPlayerPosition` back to the player's position pointer obtained from the `ModeCoordinator`.
- **`OnBeforeTeardown`** — on garrison-defense victory, look up the garrison and call `combatlifecycle.StripCombatComponents` on its squads so they survive `TeardownCombat`'s enemy disposal.

---

## Shared Combat Entry Infrastructure

### ExecuteCombatStart

**File:** `mind/combatlifecycle/start.go`

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
3. If step 2 fails, invoke the rollback closure returned by `Prepare` (when non-nil) to undo any side effects.

The `transitioner` is always the `EncounterService` instance created at startup (passed as a `combatlifecycle.CombatTransitioner` to avoid import cycles).

### EncounterService.TransitionToCombat

**File:** `mind/encounter/encounter_service.go`

```
func (es *EncounterService) TransitionToCombat(setup *CombatSetup) error
```

Checks that no encounter is already active and that `modeCoordinator` is not nil, then performs these steps inline:

1. Saves the player's current overworld position to `OriginalPlayerPosition`.
2. Calls `modeCoordinator.SetTriggeredEncounterID(encounterID)` and `ResetTacticalState()`.
3. Sets `TacticalState.PostCombatReturnMode` to `setup.PostCombatReturnMode()` when the method returns a non-empty string (currently only raid encounters do).
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

### EnrollSquadsAtPositions

**File:** `mind/combatlifecycle/enrollment.go`

The public batch enrollment helper used by all starters. For each squad/position pair, it performs the 3-step enrollment:

1. `fm.AddSquadToFaction(factionID, squadID, pos)` — faction membership + position
2. `MoveSquadUnitsToCombatPosition(manager, squadID, pos)` — all units get positions at squad location
3. `combatstate.CreateActionStateForSquad(manager, squadID)` — combat action tracking

```go
func EnrollSquadsAtPositions(fm, manager, factionID, squadIDs, positions) error
```

Deployment status (`SquadData.IsDeployed`) is the caller's policy — use `MarkSquadsDeployed(manager, squadIDs)` separately for squads that weren't already deployed (garrison defenders, raid player squads).

### MoveSquadUnitsToCombatPosition

**File:** `mind/combatlifecycle/enrollment.go`

Called during squad enrollment (see `EnrollSquadsAtPositions`) to give every unit in a squad a `LogicalPosition`. Units that already have positions are moved; units without positions have one registered. This is required before combat so that the movement system can find units on the map.

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
      → mind/combatlifecycle/start.go                    ExecuteCombatStart()
```

**Step-by-step:**

1. `EngageThreat(nodeID)` validates that the commander exists, has a position, and is co-located with the threat node.
2. `TriggerCombatFromThreat` reads the threat node's `OverworldNodeData`, looks up the encounter definition in `core.GetNodeRegistry()`, and creates an `OverworldEncounterData` entity with `ThreatNodeID` set to the threat's entity ID. This `ThreatNodeID` link is critical — it is later used by `OverworldCombatResolver.Resolve` to find the threat node and apply damage to it.
3. `OverworldCombatStarter.Prepare` validates the encounter entity via `encounter.ValidateEncounterEntity`, hides the encounter entity's sprite (stored for rollback), then calls `SpawnCombatEntities`.
4. `SpawnCombatEntities` (returns `*SpawnResult` with `EnemySquadIDs`, `PlayerFactionID`, `EnemyFactionID`) checks whether the threat node has an NPC garrison. If it does, those existing garrison squads become the enemies. If not, it generates enemies from a power budget via `generateAttackerSquads` — which delegates to `mind/spawning/squadscreation.go` for the actual squad composition. Faction creation and squad enrollment run through the shared `assembleCombatFactions` helper in both branches.
5. Power budget generation uses `evaluation.CalculateSquadPower` to measure the player's deployed squads, applies a difficulty multiplier from the encounter's level, and iteratively adds units from a type-filtered pool until the target power is reached.

**CombatSetup produced:**
- `Type = CombatTypeOverworld` (zero value / default)
- `PostCombatReturnMode() = ""`  (returns to exploration; derived from `Type`)
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

1. `TriggerGarrisonDefense` creates an `OverworldEncounterData` entity with `AttackingFactionType` set. (The combat path is discriminated by `CombatSetup.Type == CombatTypeGarrisonDefense`, set by `GarrisonDefenseStarter` rather than by a field on the encounter data.)
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
- `PostCombatReturnMode() = ""` (returns to exploration or overworld depending on active context)

### Pathway 3: Raid Room Encounter

**Trigger:** Player selects a combat room in `RaidMode` and confirms deployment.

**File chain:**

```
gui/guiraid/raidmode.go                     OnDeployConfirmed()
  → gui/guiraid/raidmode.go                 raidRunner.TriggerRaidEncounter(nodeID)
    → campaign/raid/raidrunner.go           TriggerRaidEncounter()
      → raid.RaidCombatStarter{...}
      → mind/combatlifecycle/start.go     ExecuteCombatStart()
        → encounterService (as transitioner)
```

**Step-by-step:**

1. `TriggerRaidEncounter` snapshots alive unit counts per player squad before starting (stored in `preCombatAliveCounts` for the post-combat summary).
2. It resolves which squads are deployed: if a `DeploymentData` entity exists with `DeployedSquadIDs`, those are used; otherwise all player squad IDs from `RaidStateData` are used.
3. `RaidCombatStarter.Prepare` calls `SetupRaidFactions` (in `campaign/raid/raidencounter.go`) which places player squads at fixed offsets to the left (`playerOffsetX = -3`, `playerOffsetY = -2`) and garrison squads to the right (`enemyOffsetX = 3`, `enemyOffsetY = 2`) of `CombatPosition()` from config. Multiple squads are spread horizontally with `squadSpreadX = 2`.
4. Unlike overworld encounters, raid encounters do not generate new enemy squads. The garrison squads pre-created during `GenerateGarrison` are used directly.

**CombatSetup produced:**
- `Type = CombatTypeRaid`
- `PostCombatReturnMode() = combatlifecycle.PostCombatReturnRaid` (returns to raid mode, not exploration)
- `RosterOwnerID = commanderID`
- `EncounterID = raidEntityID` (the raid entity, not an OverworldEncounterData entity)

`RaidCombatStarter` attaches a `RaidEncounterResolver` to `CombatSetup.Resolver`, just like every other starter. `EncounterService.ExitCombat` runs the resolver via the standard `ExecuteResolution` pipeline regardless of combat type. The post-combat callback (`SetPostCombatCallback`) is now a **notification hook only**: `RaidRunner.ResolveEncounter` receives the `*ResolutionResult` produced by the resolver and uses it to build the GUI summary (`RewardText`, units lost, alert level) and run raid-specific state updates (`PostEncounterProcessing`).

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

The `RaidRunner` registers as a post-combat listener via `encounterService.AddPostCombatListener(...)` at construction time, so it receives the combat result automatically after each raid room battle. The listener includes a guard: it only calls `ResolveEncounter` when `raidEntityID != 0` AND `raidState.Status == RaidActive`. This prevents cross-contamination if the player retreats from a raid and then triggers an overworld encounter.

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
- `PostCombatReturnMode() = ""`

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
                                               │  start.go                   │
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

`EncounterService.ExitCombat` is a thin shell over `combatlifecycle.ExecuteCombatExit`. It snapshots `*es.activeEncounter` into a `ResolutionContext` value (capturing `PlayerSquadIDs` from the roster up front via `collectPlayerSquadIDs` so the resolver sees a stable list), constructs an `EncounterExitHooks`, defines a history-recording closure, and delegates the orchestration:

```go
// Simplified from ExitCombat
enc := *es.activeEncounter
ctx := combatlifecycle.ResolutionContext{
    Setup:                  enc.Setup,
    Reason:                 reason,
    Outcome:                result,
    PlayerEntityID:         enc.PlayerEntityID,
    PlayerSquadIDs:         enc.PlayerSquadIDs, // snapshot taken at TransitionToCombat
    OriginalPlayerPosition: enc.OriginalPlayerPosition,
}
hooks := NewEncounterExitHooks(es.manager, es.modeCoordinator)
onHistory := func(resolution *combatlifecycle.ResolutionResult) {
    /* append CompletedEncounter, clear activeEncounter */
}
resolution := combatlifecycle.ExecuteCombatExit(es.manager, ctx, hooks, teardown, onHistory)
for _, entry := range es.postCombatListeners {
    entry.fn(reason, result, resolution)
}
```

The six-step sequence inside `ExecuteCombatExit` is documented above under [ExecuteCombatExit and ExitHooks](#executecombatexit-and-exithooks-symmetric-exit-pipeline). All sprite restoration, mark-defeated, position-restore, and garrison-return logic now lives in the `EncounterExitHooks` methods rather than inline in `ExitCombat`.

Resolvers themselves decide what to do based on the `ResolutionContext`. Each combat type has exactly one resolver — branches for flee/victory/defeat are inlined inside `Resolve` rather than dispatched to sub-resolvers. Post-combat listeners (used by `RaidRunner`) fire after `ExecuteCombatExit` returns and receive the `*ResolutionResult` for GUI summary use.

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
- Inline flee branch in `OverworldCombatResolver.Resolve`: logs a `EventCombatResolved` retreat event, no state changes to the threat node, no rewards.

### Garrison Defense Resolution

**File:** `mind/encounter/resolvers.go` — `GarrisonDefenseResolver.Resolve()`

Activated when: `Type = CombatTypeGarrisonDefense`.

**On player victory (successful defense):**
- Logs a `EventGarrisonDefended` event.
- No rewards are granted (garrison defense is unpaid duty).

**On player defeat (garrison falls):**
- Calls `garrison.TransferNodeOwnership(manager, DefendedNodeID, newOwner)` where `newOwner` is the attacking faction type as a string.
- The node now belongs to the enemy faction.

**Special cleanup:** `EncounterExitHooks.OnBeforeTeardown` fires during `ExecuteCombatExit` immediately before `TeardownCombat`. When `state.Setup.Type == CombatTypeGarrisonDefense && outcome.IsPlayerVictory`, it looks up the garrison and calls `combatlifecycle.StripCombatComponents` on its squads, removing their `FactionMembershipComponent` and `PositionComponent`, resetting `IsDeployed = false`, but NOT disposing the squad entities. The garrison squads survive and remain in `garrison.GarrisonData`.

### Raid Room Resolution

**File:** `campaign/raid/resolvers.go`

Activated by `RaidRunner.ResolveEncounter` via the post-combat listener, not by `EncounterService.ExitCombat`'s internal resolution.

**On victory (`RaidEncounterResolver.Resolve` — victory branch):**
- Marks all garrison squad entities in the room as destroyed (`GarrisonSquadData.IsDestroyed = true`).
- Calls `MarkRoomCleared`.
- Checks if the floor is complete via `IsFloorComplete`; sets `FloorStateData.IsComplete = true` if so.
- Calculates room rewards using `calculateRoomReward` and returns a `*ResolutionResult`.

**On defeat or flee (`RaidEncounterResolver.Resolve` — defeat branch):**
- Sets `RaidStateData.Status = RaidDefeat`.
- Returns a `*ResolutionResult` with no rewards.

After resolution, `RaidRunner.PostEncounterProcessing` runs:
1. Applies post-encounter HP recovery (deployed squads recover more than reserve squads).
2. Increments the floor's alert level via `IncrementAlert`.
3. Checks end conditions via `CheckRaidEndConditions` (all player squads destroyed → `RaidDefeat`; final floor complete → `RaidVictory`).

### Debug / No-Op Resolution

Activated when: `Type = CombatTypeOverworld` and `ThreatNodeID = 0`.

In `resolveEncounterOutcome`, the `CombatTypeOverworld` case checks `encounterData.ThreatNodeID != 0` before creating a resolver. When `ThreatNodeID = 0` (debug random encounter), no resolver is created and no resolution occurs. This means debug encounters have zero side effects on the game world.

---

## Combat Cleanup

`EncounterService.ExitCombat` calls `teardown.TeardownCombat(enemySquadIDs)` where `teardown` is the `CombatTeardown` interface satisfied by `CombatService`. The method strips combat-only state from player squads (via direct calls to `combatstate.RemoveCombatMembership`, `perks.RemovePerkRoundState`, and `squadcore.ResetSquadDeployment`) before disposing faction entities, so the caller does not need to follow up.

### CombatService.TeardownCombat

**File:** `tactical/combat/combatservices/combat_service.go`

Executed in this order:

1. **Cleanup effects**: `cleanupEffects()` removes all active spell/artifact effects from all units tagged `SquadMemberTag`. Prevents buffs and debuffs from persisting into the next battle.

2. **Strip player squad combat state**: `collectPlayerSquadIDs()` walks the player faction's squads, and for each one calls `combatstate.RemoveCombatMembership(entity)`, `perks.RemovePerkRoundState(entity)`, and `squadcore.ResetSquadDeployment(manager, entity)`. This removes `FactionMembershipComponent`, `PerkRoundStateComponent`, the squad and unit positions, and resets `IsDeployed = false`. Must happen before step 4 (which disposes the faction entities the membership component points to).

3. **Build enemy set**: Enemy IDs are converted to a set for fast lookup during unit disposal.

4. **Dispose faction entities**: All entities with `FactionTag` are disposed.

5. **Dispose action state entities**: All entities with `ActionStateTag` are disposed.

6. **Dispose turn state entities**: All entities with `TurnStateTag` are disposed.

7. **Dispose enemy squads**: For each ID in `enemySquadIDs`, calls `manager.CleanDisposeEntity(entity, pos)` which unregisters from `GlobalPositionSystem` before disposal.

8. **Dispose enemy units**: Iterates all `SquadMemberTag` entities, checks if their `SquadMemberData.SquadID` is in the enemy set, and disposes them.

GUI callback clearing (attack-complete / move-complete / turn-end) happens separately on combat-mode exit, ensuring stale GUI closures cannot fire against torn-down widgets.

### Enemy Cleanup (Dispose)

Enemy squads and their units are permanently disposed from the ECS world. They were created specifically for this combat encounter (by the spawning package or taken from a pre-generated raid garrison). After disposal, they cease to exist.

**Exception:** For garrison defense victories, the garrison squads are NOT in `enemySquadIDs` at cleanup time because `EncounterExitHooks.OnBeforeTeardown` (fired by `ExecuteCombatExit` immediately before `TeardownCombat`) ran first and stripped their combat components. Since `TeardownCombat` only disposes squads in the provided `enemySquadIDs` list, the garrison squads survive.

For raid defeats, garrison squads in the room being fought are pre-created by `GenerateGarrison` and stored in `RoomData.GarrisonSquadIDs`. These ARE disposed by `TeardownCombat` after a defeat or flee (the room's garrison is treated as an enemy squad list). On victory, the victory branch of `RaidEncounterResolver` marks them as `IsDestroyed = true` but they remain in ECS until `TeardownCombat` disposes them.

### Player Squad Cleanup

Combat-only state is stripped from player squads via direct calls to package-exported helpers. Each owning package exports its own cleanup helper:

- `tactical/combat/combatstate.RemoveCombatMembership(entity)` — removes `FactionMembershipComponent`.
- `tactical/powers/perks.RemovePerkRoundState(entity)` — removes `PerkRoundStateComponent`.
- `tactical/squads/squadcore.ResetSquadDeployment(manager, entity)` — unregisters squad and unit positions, resets `IsDeployed = false`.

`combatlifecycle.StripCombatComponents(manager, squadIDs)` calls all three in sequence for each squad ID. Both consumers — `CombatService.TeardownCombat` (for player squads at combat exit) and `EncounterExitHooks.OnBeforeTeardown` (for garrison defenders returning to their node) — do the same three calls. `CombatService` invokes them inline because it cannot import `mind/combatlifecycle` (upward layering); the three helpers are in packages it already imports.

Adding a new combat-only component to the cleanup set: export a `RemoveX` helper from the owning package, then add one call site in both `combatlifecycle/teardown.go::StripCombatComponents` and `combat_service.go::TeardownCombat`.

Player squad entities are NOT disposed. They return to their pre-combat state: no position, no faction membership, not deployed. They remain in the player's roster and can be deployed in subsequent battles.

### Garrison Defense Special Case

For garrison defense victories specifically, `EncounterExitHooks.OnBeforeTeardown` fires inside `ExecuteCombatExit` **before** `enemySquadIDs` is passed to `TeardownCombat`. In this pathway:

- In `GarrisonDefenseStarter.Prepare`, the garrison squads join the **player faction** and the generated attackers join the **enemy faction**. So `CombatSetup.EnemySquadIDs` contains the generated attacker squads. The garrison squads are in the player faction.
- `TeardownCombat(enemySquadIDs)` disposes the generated attacker squads (the enemies in this context) and runs the cleanup-hook registry against the player faction's squads.
- But garrison squads must survive for future use. `OnBeforeTeardown` runs first and calls `StripCombatComponents` on the garrison squads, stripping their combat state (position, faction membership, deployment flag) via the same hook registry. When `TeardownCombat` later runs the hooks again on the player faction's squads, the garrison squads have no `FactionMembershipComponent` left to strip and the hooks are safely idempotent.

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

`TacticalState.PostCombatReturnMode` is set during `TransitionToCombat` when `setup.PostCombatReturnMode()` returns a non-empty string. Currently only raid combat returns a non-empty value (`combatlifecycle.PostCombatReturnRaid`, `"raid"`), derived from `setup.Type == CombatTypeRaid`.

On entering `RaidMode` from `CombatMode`, `RaidMode.Enter(fromMode)` checks `fromMode.GetModeName() == "combat"` and `raidRunner.LastEncounterResult != nil` to trigger the summary panel display.

---

## Dependency Graph

The following shows import relationships relevant to the combat pipeline. Arrows mean "imports":

```
gui/guicombat
  → tactical/combat/combattypes       (CombatResult, DamageModifiers)
  → tactical/combat/combatservices    (CombatService)
  → mind/combatlifecycle              (CombatExitReason, EncounterOutcome, CombatTeardown)
  → mind/encounter                    (EncounterController interface only)
  → mind/ai                           (SetupCombatAI — injected at init time only)

gui/guioverworld
  → mind/encounter                    (EncounterService, OverworldCombatStarter)
  → mind/combatlifecycle              (CombatStarter interface)

gui/guiraid
  → campaign/raid                     (RaidRunner, GetRaidState, etc.)

mind/combatlifecycle
  → core/common                       (EntityManager)
  → core/coords                       (LogicalPosition)
  → core/config                       (DEBUG_MODE)
  → tactical/combat/combatstate       (CombatFactionManager, RemoveCombatMembership)
  → tactical/squads/squadcore         (SquadData, GetUnitIDsInSquad, ResetSquadDeployment)
  → tactical/squads/unitprogression   (AwardExperience)
  → tactical/commander                (FindCommanderForSquad)
  → tactical/powers/perks             (RemovePerkRoundState)
  → tactical/powers/progression       (AddArcanaPoints, AddSkillPoints)
  → tactical/powers/spells            (ManaData)

mind/encounter
  → mind/combatlifecycle              (ExecuteResolution, ExecuteCombatExit,
                                        ResolutionContext, ExitHooks, StripCombatComponents,
                                        CreateFactionPair, EnrollSquadsAtPositions,
                                        CombatStarter/CombatResolver interfaces)
  → mind/spawning                     (squad composition for power-budget enemies)
  → tactical/squads/roster            (GetPlayerSquadRoster — for PlayerSquadIDs snapshot)
  → tactical/squads/squadcore         (squad helpers used during setup)
  → campaign/overworld/core           (OverworldEncounterData, OverworldNodeData)
  → campaign/overworld/garrison       (GarrisonData, TransferNodeOwnership)
  → campaign/overworld/threat         (DestroyThreatNode)
  → campaign/overworld/ids            (EncounterTypeID, NodeTypeID)
  → templates                         (reward configuration)
  → core/common, core/coords          (EntityManager, LogicalPosition)

campaign/raid
  → mind/combatlifecycle              (ExecuteCombatStart, ExecuteResolution,
                                        ApplyHPRecovery, CreateFactionPair,
                                        EnrollSquadsAtPositions)
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

The critical boundary is that `gui/guicombat` only touches `mind/encounter` through the `EncounterController` interface in `mind/encounter/types.go` — satisfied by `EncounterService` via structural typing. It does not depend on the concrete service or its internals.

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
ExecuteCombatStart()                          mind/combatlifecycle/start.go
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
     ├─→ snapshot activeEncounter → ResolutionContext (Setup, Reason, Outcome,
     │                                                  PlayerEntityID,
     │                                                  PlayerSquadIDs (from roster),
     │                                                  OriginalPlayerPosition)
     ├─→ build EncounterExitHooks (sprite/defeated/position/garrison-return)
     ├─→ define onHistory closure (append CompletedEncounter, clear activeEncounter)
     │
     ▼
combatlifecycle.ExecuteCombatExit(manager, ctx, hooks, teardown, onHistory)
     │
     ├─── Step 1: ExecuteResolution(manager, ctx.Setup.Resolver, ctx) — if Resolver != nil
     │       │   resolver decides what to do based on ctx.Reason / ctx.Outcome.IsPlayerVictory:
     │       │
     │       ├─→ OverworldCombatResolver.Resolve
     │       │      ├─→ Victory: DestroyThreatNode() or weaken + rewards
     │       │      ├─→ Defeat: grow threat intensity
     │       │      └─→ Flee branch (inline): log retreat event, no state changes
     │       │
     │       ├─→ GarrisonDefenseResolver.Resolve
     │       │      ├─→ Victory: log event
     │       │      └─→ Defeat: TransferNodeOwnership()
     │       │
     │       ├─→ RaidEncounterResolver.Resolve
     │       │      ├─→ Victory branch (inline): MarkRoomCleared + rewards
     │       │      └─→ Defeat/Flee branch (inline): set raidState.Status = RaidDefeat
     │       │
     │       └─→ (debug encounter, nil resolver) returns empty *ResolutionResult
     │       └─→ Grant(Rewards, Target) fills RewardText in place
     │
     ├─── Step 2: hooks.OnAfterResolution
     │       ├─→ [Flee only] restoreEncounterSprite()
     │       └─→ [Victory + non-raid] markEncounterDefeated(): IsDefeated=true, hide sprite
     │
     ├─── Step 3: hooks.OnRestorePlayer
     │       └─→ write OriginalPlayerPosition back via ModeCoordinator
     │
     ├─── Step 4: onHistory(resolution)
     │       ├─→ append CompletedEncounter to history (ring-buffered)
     │       └─→ clear es.activeEncounter
     │
     ├─── Step 5: hooks.OnBeforeTeardown
     │       └─→ [Garrison defense victory only] StripCombatComponents(garrison squads)
     │
     └─── Step 6: combatService.TeardownCombat(enemySquadIDs)
             ├─→ cleanupEffects()
             ├─→ For each playerSquad in collectPlayerSquadIDs():
             │      ├─ combatstate.RemoveCombatMembership(entity)
             │      ├─ perks.RemovePerkRoundState(entity)
             │      └─ squadcore.ResetSquadDeployment(manager, entity)
             ├─→ disposeEntitiesByTag(FactionTag)
             ├─→ disposeEntitiesByTag(ActionStateTag)
             ├─→ disposeEntitiesByTag(TurnStateTag)
             ├─→ disposeEnemySquads(enemySquadIDs)
             └─→ disposeEnemyUnits(enemySquadSet)

ExecuteCombatExit returns *ResolutionResult
     │
     ▼
EncounterService.ExitCombat: postCombatCallback(reason, outcome, resolution)
     └─→ [Raid only] RaidRunner.ResolveEncounter()
            ├─→ Read RewardText from resolution (already granted in Step 1)
            ├─→ PostEncounterProcessing()
            │      ├─→ ApplyPostEncounterRecovery()
            │      ├─→ IncrementAlert()
            │      └─→ CheckRaidEndConditions()
            └─→ Build LastEncounterResult for GUI summary
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

This is distinct from `RaidEncounterResolver`'s defeat branch (which sets `RaidStatus = RaidDefeat` and ultimately leads to `finishRaid` clearing all state) and from the flee-from-combat path (which ends a single room encounter but leaves the overall raid intact).

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

### ResolutionResult fields

`ResolutionResult` (defined in `mind/combatlifecycle/exit.go`) is a single type that carries resolution state through both the resolver and `ExecuteResolution`, with lifecycle-phased fields:

- **`Rewards`** (`Reward`) — set by the resolver; consumed by `Grant` inside `ExecuteResolution`.
- **`Target`** (`GrantTarget`) — set by the resolver; consumed by `Grant` inside `ExecuteResolution`. Zero-valued for resolvers that grant nothing. After `Grant` runs, the field lingers on the consumer-facing struct but is no longer meaningful — documented as "intermediate state".
- **`RewardText`** (`string`) — empty when the resolver returns; filled by `ExecuteResolution` after `Grant` produces the human-readable summary ("150 gold, 75 XP"). Consumed by listeners (e.g., `RaidRunner.ResolveEncounter`).
- **`Description`** (`string`) — resolver-supplied summary ("Threat 42 destroyed") read by both the orchestration and downstream listeners.

`ExecuteResolution` mutates the resolver-returned pointer in place to fill `RewardText`; resolvers must return a freshly built struct literal (true today — they all do). Earlier revisions of this package had two separate types (`ResolutionPlan` for the resolver output and `ResolutionResult` for the post-`Grant` output); they were merged because `ResolutionPlan` had no consumers outside `ExecuteResolution` itself.

---

## Key File Index

| File | Purpose |
|------|---------|
| `mind/combatlifecycle/start.go` | Start-side contracts + orchestration: `CombatStarter` (returns optional rollback closure), `CombatSetup` (with `PostCombatReturnMode()` method), `CombatType`, `CombatTransitioner`, `ExecuteCombatStart`, `PostCombatReturnRaid`/`PostCombatReturnDefault` constants |
| `mind/combatlifecycle/exit.go` | Exit-side contracts + orchestration: `CombatExitReason`, `EncounterOutcome`, `ResolutionContext`, `CombatResolver`, `ResolutionResult`, `ExitHooks`, `ExecuteResolution`, `ExecuteCombatExit` |
| `mind/combatlifecycle/teardown.go` | `CombatTeardown` interface plus `StripCombatComponents` and `ApplyHPRecovery` helpers |
| `mind/combatlifecycle/enrollment.go` | `CreateFactionPair`, `EnrollSquadsAtPositions`, `MoveSquadUnitsToCombatPosition`: faction creation and squad enrollment helpers |
| `mind/combatlifecycle/reward.go` | `Reward`, `Grant`, `GrantTarget`: generic reward calculation and distribution primitives |
| `mind/combatlifecycle/casualties.go` | `GetLivingUnitIDs`, `CountDeadUnits`: casualty counting helpers |
| `mind/encounter/encounter_service.go` | `EncounterService`: tracks `ActiveEncounter`, implements `TransitionToCombat`, `ExitCombat`, `SetPostCombatCallback` |
| `mind/encounter/encounter_config.go` | `clampPowerTarget`: encounter power-clamping helper (encounter-only; raid uses archetypes) |
| `mind/encounter/validators.go` | `ValidateEncounterEntity`: validates encounter entity + OverworldEncounterData |
| `mind/encounter/rewards.go` | `CalculateIntensityReward`: threat-intensity-based reward calculation |
| `mind/encounter/starters.go` | `OverworldCombatStarter`, `GarrisonDefenseStarter`: two of the three `CombatStarter` implementations |
| `mind/encounter/encounter_trigger.go` | `TriggerCombatFromThreat`, `TriggerRandomEncounter`, `TriggerGarrisonDefense`: creates encounter entities |
| `mind/encounter/encounter_setup.go` | `SpawnCombatEntities` (returns `*SpawnResult`), `generateAttackerSquads`, `assembleCombatFactions`: combat entity creation |
| `mind/encounter/resolvers.go` | `OverworldCombatResolver` (flee branch inlined), `GarrisonDefenseResolver` |
| `mind/encounter/exit_hooks.go` | `EncounterExitHooks`: implements `combatlifecycle.ExitHooks` (sprite restore, mark-defeated, position restore, garrison return-to-node) |
| `mind/encounter/types.go` | `ActiveEncounter`, `CompletedEncounter`, `SpawnResult`, `ModeCoordinator`, `EncounterController` interfaces |
| `mind/spawning/squadscreation.go` | `createSquadForPowerBudget`: power-budget squad generation used by encounter setup |
| `mind/spawning/composition.go` | `generateRandomComposition` and faction-typed composition helpers |
| `campaign/raid/starters.go` | `RaidCombatStarter`: raid-specific `CombatStarter` implementation |
| `campaign/raid/raidencounter.go` | `SetupRaidFactions`: positions squads for raid combat |
| `campaign/raid/raidrunner.go` | `RaidRunner`: orchestrates the full raid loop, registered as post-combat listener |
| `campaign/raid/resolvers.go` | `RaidEncounterResolver` (victory + defeat branches inlined) |
| `tactical/combat/combatservices/combat_service.go` | `CombatService.TeardownCombat` (satisfies `combatlifecycle.CombatTeardown`; strips player squads via `combatstate.RemoveCombatMembership` + `perks.RemovePerkRoundState` + `squadcore.ResetSquadDeployment`), `InitializeCombat`, `CheckVictoryCondition` |
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
| `gui/guicombat/combatdeps.go` | `CombatModeDeps`: dependency container, holds `encounter.EncounterController` interface |
| `gui/guioverworld/overworld_action_handler.go` | `EngageThreat`, `HandleRaid`, `StartRandomEncounter` |
| `gui/guioverworld/overworld_panels_registry.go` | Debug panel with "Start Random Encounter" button |
| `gui/guiexploration/exploration_panels_registry.go` | Debug panel with "Start Raid" button (roguelike only) |
| `gui/guiraid/raidmode.go` | `OnDeployConfirmed`, `OnCombatComplete`, `autoStartRaid` |
| `gui/framework/coordinator.go` | `GameModeCoordinator`: implements `ModeCoordinator`, manages context switching |
| `setup/gamesetup/moderegistry.go` | Wires `EncounterService` to `startCombat` closure (returns `error`) for overworld modes; registers raid mode |
| `game_main/setup.go` | `SetupRoguelikeMode`, `SetupOverworldMode`, `SetupRoguelikeFromSave`: top-level wiring of services and modes |
