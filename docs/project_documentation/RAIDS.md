# Raid System -- Technical Documentation

**Last Updated:** 2026-02-20

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Overview](#2-architecture-overview)
3. [Package Structure](#3-package-structure)
4. [ECS Components and Data Structures](#4-ecs-components-and-data-structures)
5. [Configuration System](#5-configuration-system)
6. [Garrison Generation](#6-garrison-generation)
7. [Floor Graph and Room Navigation](#7-floor-graph-and-room-navigation)
8. [Squad Archetypes and Assignment](#8-squad-archetypes-and-assignment)
9. [Deployment System](#9-deployment-system)
10. [Alert System](#10-alert-system)
11. [Combat Integration](#11-combat-integration)
12. [Resolution and Recovery](#12-resolution-and-recovery)
13. [Reward System](#13-reward-system)
14. [GUI Layer](#14-gui-layer)
15. [Raid Lifecycle -- End to End](#15-raid-lifecycle----end-to-end)
16. [Integration with Other Systems](#16-integration-with-other-systems)
17. [Map Generation for Raids](#17-map-generation-for-raids)
18. [Design Decisions and Rationale](#18-design-decisions-and-rationale)
19. [Current Limitations and TODOs](#19-current-limitations-and-todos)
20. [File Reference](#20-file-reference)

---

## 1. Executive Summary

The raid system implements a multi-floor dungeon assault mode where the player sends squads to attack a procedurally generated garrison. Each garrison consists of multiple floors, each floor being a directed acyclic graph (DAG) of rooms. The player navigates this room graph, choosing which rooms to enter. Combat rooms trigger tactical encounters against pre-composed garrison squads. Non-combat rooms (rest rooms, stairs) provide recovery or floor progression.

Key characteristics of the raid system:

- **Multi-floor progression** -- The player fights through 3+ floors of increasing difficulty.
- **DAG-based room navigation** -- Each floor is a graph of interconnected rooms. The player must clear parent rooms before child rooms become accessible.
- **Pre-composed garrison squads** -- Enemy squads are built from named archetypes (e.g., "Chokepoint Guard", "Shield Wall") rather than generated from power budgets.
- **Alert escalation** -- As the player clears rooms, the garrison becomes more alert, gaining stat buffs and activating reserve squads.
- **Morale and recovery** -- Player squads accumulate morale changes and receive partial HP recovery between encounters.
- **Deployment management** -- The player can choose which squads to deploy vs. hold in reserve for each encounter.

The raid system is currently accessible only through the debug menu in roguelike mode.

---

## 2. Architecture Overview

The raid system spans three architectural layers:

```
+-------------------------------------------------------------------+
|                        GUI Layer (guiraid/)                       |
|  RaidMode  |  FloorMapPanel  |  DeployPanel  |  SummaryPanel      |
|  RaidUIState  |  RaidPanelRegistry                                |
+-------------------------------------------------------------------+
        |                       |                        |
        v                       v                        v
+-------------------------------------------------------------------+
|                    Orchestration Layer (mind/raid/)                |
|  RaidRunner (controller / coordinator)                            |
|  - StartRaid, EnterFloor, SelectRoom, TriggerRaidEncounter        |
|  - ResolveEncounter, PostEncounterProcessing, AdvanceFloor        |
+-------------------------------------------------------------------+
        |           |            |             |           |
        v           v            v             v           v
+-------------------------------------------------------------------+
|                    ECS Data + System Layer (mind/raid/)            |
|  Components  |  Queries  |  Garrison Gen  |  Alert     | Recovery |
|  Archetypes  |  Assignment  |  Deployment  |  Resolution| Rewards |
|  FloorGraph  |  Config                                            |
+-------------------------------------------------------------------+
        |                                                    |
        v                                                    v
+-------------------------------------------------------------------+
|              External Systems                                     |
|  encounter/EncounterService  |  squads/  |  combat/  |  worldmap/ |
|  commander/  |  evaluation/  |  spells/  |  coords/  |  common/   |
+-------------------------------------------------------------------+
```

### Separation of Concerns

The architecture strictly separates three concerns:

1. **GUI state** (`guiraid/raidstate.go`) -- Tracks which panel is shown, which room is selected, and post-encounter summary data. Contains zero game logic.

2. **Orchestration** (`mind/raid/raidrunner.go`) -- The `RaidRunner` struct coordinates the raid lifecycle. It is NOT an ECS system; it is a service that reads and writes ECS state while managing flow control (floor transitions, encounter triggering, post-combat callbacks).

3. **ECS data and systems** -- Pure data components (`RaidStateData`, `FloorStateData`, `RoomData`, etc.) and stateless system functions (`GenerateGarrison`, `AssignArchetypesToFloor`, `IncrementAlert`, etc.) that operate on those components.

---

## 3. Package Structure

### `mind/raid/` -- Core Raid Package

```
mind/raid/
  init.go            -- ECS subsystem registration (components + tags)
  components.go      -- Pure data component structs and RaidStatus enum
  config.go          -- JSON config loading and accessor functions
  archetypes.go      -- Garrison squad archetype definitions
  garrison.go        -- Garrison generation (floors, rooms, squads)
  floorgraph.go      -- Room DAG navigation and accessibility
  assignment.go      -- Archetype-to-room assignment logic
  deployment.go      -- Player squad deployment management
  alert.go           -- Alert level escalation and reserve activation
  recovery.go        -- HP and morale recovery functions
  resolution.go      -- Post-combat victory/defeat processing
  rewards.go         -- Room-type-specific reward grants
  raidencounter.go   -- Combat faction setup for raid encounters
  raidrunner.go      -- Top-level raid coordinator (RaidRunner)
  queries.go         -- ECS query functions for raid entities
```

### `gui/guiraid/` -- Raid GUI Package

```
gui/guiraid/
  raidstate.go              -- UI-only state (RaidUIState)
  raidmode.go               -- Main UIMode implementation (RaidMode)
  raid_panels_registry.go   -- Panel widget construction via framework
  floormap_panel.go         -- Floor map display controller
  deploy_panel.go           -- Squad deployment controller
  summary_panel.go          -- Post-encounter summary controller
```

### Related External Files

| File | Role |
|------|------|
| `assets/gamedata/raidconfig.json` | All raid tuning parameters |
| `world/worldmap/gen_garrison_dag.go` | DAG construction and room types |
| `world/worldmap/gen_garrison.go` | Physical map generation from DAG |
| `world/worldmap/gen_garrison_meta.go` | Spawn positions and room metadata |
| `world/worldmap/gen_garrison_terrain.go` | Per-room tactical terrain injection |
| `game_main/setup_shared.go` | Raid config loading at startup |
| `game_main/setup_roguelike.go` | RaidRunner creation and mode registration |
| `mind/encounter/encounter_service.go` | `BeginRaidCombat()` and `ExitCombat()` integration |
| `mind/encounter/types.go` | `ActiveEncounter.IsRaidCombat` flag |
| `gui/specs/layout.go` | Raid panel size constants |
| `gui/guicombat/combat_turn_flow.go` | `PostCombatReturnMode` handling |

---

## 4. ECS Components and Data Structures

The raid system registers six ECS components via the standard subsystem registration pattern in `mind/raid/init.go`:

```go
func init() {
    common.RegisterSubsystem(func(em *common.EntityManager) {
        initRaidComponents(em)
        initRaidTags(em)
    })
}
```

### Component Table

| Component | Tag | Data Struct | Cardinality | Purpose |
|-----------|-----|-------------|-------------|---------|
| `RaidStateComponent` | `RaidStateTag` | `*RaidStateData` | Singleton | Overall raid progress |
| `FloorStateComponent` | `FloorStateTag` | `*FloorStateData` | One per floor | Per-floor progress |
| `RoomDataComponent` | `RoomDataTag` | `*RoomData` | One per room | Individual room state |
| `AlertDataComponent` | `AlertDataTag` | `*AlertData` | One per floor | Alert escalation |
| `GarrisonSquadComponent` | `GarrisonSquadTag` | `*GarrisonSquadData` | One per garrison squad | Marks squads as garrison defenders |
| `DeploymentComponent` | `DeploymentTag` | `*DeploymentData` | Singleton | Current deployment split |

### RaidStateData (Singleton)

```go
type RaidStateData struct {
    CurrentFloor       int            // Which floor the player is on (1-indexed)
    TotalFloors        int            // Total number of floors in the garrison
    Status             RaidStatus     // None / Active / Victory / Defeat / Retreated
    CommanderID        ecs.EntityID   // Player's commander for mana rewards
    PlayerSquadIDs     []ecs.EntityID // All player squads in the raid
    GarrisonKillCount  int            // Total garrison squads destroyed
}
```

The `RaidStatus` enum tracks the overall raid state:

| Value | Meaning |
|-------|---------|
| `RaidNone` (0) | No raid active (zero value) |
| `RaidActive` | Raid is in progress |
| `RaidVictory` | Player cleared all floors |
| `RaidDefeat` | All player squads destroyed or lost a battle |
| `RaidRetreated` | Player chose to retreat |

### FloorStateData (One Per Floor)

```go
type FloorStateData struct {
    FloorNumber      int
    RoomsCleared     int
    RoomsTotal       int
    GarrisonSquadIDs []ecs.EntityID  // Active garrison squads on this floor
    ReserveSquadIDs  []ecs.EntityID  // Reserve squads (activate on alert)
    IsComplete       bool            // True when stairs room is cleared
}
```

### RoomData (One Per Room)

```go
type RoomData struct {
    NodeID           int              // DAG node identifier
    RoomType         string           // Room type constant (e.g., "guard_post")
    FloorNumber      int
    IsCleared        bool             // Player has defeated this room
    IsAccessible     bool             // All parent rooms are cleared
    GarrisonSquadIDs []ecs.EntityID   // Garrison squads defending this room
    ChildNodeIDs     []int            // Downstream room IDs in DAG
    ParentNodeIDs    []int            // Upstream room IDs in DAG
    OnCriticalPath   bool             // Part of the mandatory path to stairs
}
```

### AlertData (One Per Floor)

```go
type AlertData struct {
    FloorNumber    int
    CurrentLevel   int   // 0-3 (Unaware -> Suspicious -> Alerted -> Lockdown)
    EncounterCount int   // Number of encounters on this floor
}
```

### GarrisonSquadData (One Per Garrison Squad)

```go
type GarrisonSquadData struct {
    ArchetypeName string   // e.g., "chokepoint_guard"
    FloorNumber   int
    RoomNodeID    int      // -1 for reserve squads
    IsReserve     bool     // True if not yet assigned to a room
    IsDestroyed   bool     // True after player defeats them
}
```

This component is added alongside the standard `SquadComponent` on the same entity. It marks a squad as part of the garrison rather than a player squad.

### DeploymentData (Singleton)

```go
type DeploymentData struct {
    DeployedSquadIDs []ecs.EntityID  // Squads that will fight next encounter
    ReserveSquadIDs  []ecs.EntityID  // Squads held back (get better recovery)
}
```

---

## 5. Configuration System

All raid tuning parameters are loaded from `assets/gamedata/raidconfig.json` at game startup via `raid.LoadRaidConfig()`, called in `game_main/setup_shared.go`:

```go
if err := raid.LoadRaidConfig("assets/gamedata/raidconfig.json"); err != nil {
    fmt.Printf("WARNING: Failed to load raid config: %v (using defaults)\n", err)
}
```

The global `raid.RaidConfig` variable holds the parsed configuration. Every accessor function in `config.go` provides a hardcoded default if the config is nil, making the system resilient to missing config files.

### Configuration Sections

#### `raid` -- Core Parameters

| Field | Default | Description |
|-------|---------|-------------|
| `maxPlayerSquads` | 4 | Maximum squads the player can bring |
| `maxDeployedPerEncounter` | 3 | Maximum squads that fight simultaneously |
| `defaultFloorCount` | 3 | Floors in a standard raid |
| `reservesPerFloor` | 1 | Base number of reserve garrison squads per floor |
| `extraReservesAfterFloor` | 3 | Floor number at which reserves increase by 1 |
| `combatPositionX/Y` | 50, 40 | World-space position where combat takes place |

#### `recovery` -- HP and Morale

| Field | Value | Description |
|-------|-------|-------------|
| `deployedHPPercent` | 12 | HP% restored to deployed squads after encounter |
| `reserveHPPercent` | 27 | HP% restored to reserve squads after encounter |
| `betweenFloorMoraleBonus` | 3 | Morale bonus when advancing floors |
| `victoryMoraleBonus` | 5 | Morale bonus after winning an encounter |
| `restRoomMoraleBonus` | 5 | Morale bonus from rest rooms |
| `restRoomHPPercent` | 20 | HP% restored from rest rooms |
| `unitDeathMoralePenalty` | 3 | Morale loss per unit death |
| `defeatMoralePenalty` | 10 | Morale loss on defeat |

The differentiated recovery (12% deployed vs. 27% reserve) creates a strategic incentive to rotate squads between deployed and reserve roles. Squads held in reserve recover significantly more HP.

#### `alert` -- Escalation Levels

```json
{
  "levels": [
    { "level": 0, "name": "Unaware",    "encounterThreshold": 0, "armorBonus": 0, "strengthBonus": 0, "weaponBonus": 0, "activatesReserves": false },
    { "level": 1, "name": "Suspicious",  "encounterThreshold": 2, "armorBonus": 1, "strengthBonus": 0, "weaponBonus": 0, "activatesReserves": false },
    { "level": 2, "name": "Alerted",     "encounterThreshold": 4, "armorBonus": 1, "strengthBonus": 1, "weaponBonus": 0, "activatesReserves": true },
    { "level": 3, "name": "Lockdown",    "encounterThreshold": 6, "armorBonus": 1, "strengthBonus": 2, "weaponBonus": 1, "activatesReserves": true }
  ]
}
```

Alert levels are per-floor and escalate based on the number of encounters. At levels 2 and 3, reserve garrison squads are activated and placed into accessible rooms.

#### `morale` -- Debuff Thresholds

| Morale Range | DEX Penalty | STR Penalty |
|-------------|-------------|-------------|
| 30-100 | 0 | 0 |
| 10-29 | -2 | 0 |
| 0-9 | -2 | -2 |

Low morale imposes stat penalties on squad units, incentivizing players to manage morale through victories and rest rooms.

#### `archetypeAssignment` -- Squad Placement Rules

| Field | Values | Description |
|-------|--------|-------------|
| `criticalPathArchetypes` | chokepoint_guard, shield_wall, orc_vanguard | Squads placed on the critical path |
| `branchArchetypes` | ranged_battery, fast_response, ambush_pack | Squads placed on branch rooms |
| `eliteArchetypes` | mage_tower, command_post | Squads that only appear on floor 3+ |
| `eliteFloorThreshold` | 3 | Floor where elites unlock |
| `reserveArchetypes` | fast_response, ambush_pack | Templates for reserve squads |

---

## 6. Garrison Generation

Garrison generation is the process of creating all ECS entities for a raid. It is triggered by `GenerateGarrison()` in `mind/raid/garrison.go`.

### Generation Flow

```
GenerateGarrison(manager, floorCount, commanderID, playerSquadIDs)
  |
  +-- Create RaidStateData entity (singleton)
  |
  +-- for each floor (1..floorCount):
        |
        +-- generateFloor(manager, floorNumber)
              |
              +-- worldmap.BuildGarrisonDAG(floorNumber)
              |     Build abstract DAG with room types, critical path, branches
              |
              +-- Create AlertData entity for this floor
              |
              +-- buildFloorGraph(manager, dag, floorNumber)
              |     Create RoomData entities from DAG nodes
              |
              +-- AssignArchetypesToFloor(dag, floorNumber)
              |     Map room node IDs to archetype names
              |
              +-- for each assigned room:
              |     InstantiateGarrisonSquad(manager, archetype, floor, nodeID, false)
              |       Create squad entity via squads.CreateSquadFromTemplate
              |       Add GarrisonSquadComponent
              |       Link squad ID to room's GarrisonSquadIDs
              |
              +-- for each reserve:
              |     InstantiateGarrisonSquad(manager, archetype, floor, -1, true)
              |
              +-- Create FloorStateData entity
```

### InstantiateGarrisonSquad

This function bridges the raid archetype system to the existing squad system. For each unit in the archetype definition, it:

1. Looks up the monster template by name via `squads.GetTemplateByName()`.
2. Clones the template and overrides grid position, size, and leader status from the archetype definition.
3. Calls `squads.CreateSquadFromTemplate()` with a dummy position (garrison squads are physically placed only when combat starts).
4. Adds `GarrisonSquadComponent` to mark the squad as a garrison defender.

```go
// From garrison.go:148
squadID := squads.CreateSquadFromTemplate(manager, displayName, squads.FormationBalanced, dummyPos, unitTemplates)
```

The display name includes the floor number for debugging: e.g., "Chokepoint Guard (F2)".

---

## 7. Floor Graph and Room Navigation

Each floor's room structure is a directed acyclic graph (DAG). The DAG determines:
- Which rooms the player must clear to progress (critical path).
- Which rooms are optional side branches (offering rewards or rest).
- The order in which rooms become accessible.

### Room Accessibility Rules

A room is **accessible** if and only if:
- It is the entry room (no parents), OR
- All of its parent rooms have been cleared.

This is implemented in `floorgraph.go`:

```go
// A room is accessible if ALL parent rooms are cleared
allParentsCleared := true
for _, parentID := range room.ParentNodeIDs {
    parentRoom := GetRoomData(manager, parentID, floorNumber)
    if parentRoom == nil || !parentRoom.IsCleared {
        allParentsCleared = false
        break
    }
}
```

When a room is cleared via `MarkRoomCleared()`, it triggers accessibility recalculation for all child rooms.

### Floor Completion

A floor is complete when the **stairs room** is cleared:

```go
func IsFloorComplete(manager *common.EntityManager, floorNumber int) bool {
    rooms := GetAllRoomsForFloor(manager, floorNumber)
    for _, room := range rooms {
        if room.RoomType == worldmap.GarrisonRoomStairs && room.IsCleared {
            return true
        }
    }
    return false
}
```

This means the player does not need to clear all rooms on a floor -- only rooms on the critical path leading to the stairs are mandatory. Branch rooms are optional.

### Room Types

| Constant | String Value | Behavior | Has Garrison | On Critical Path |
|----------|-------------|----------|--------------|-----------------|
| `GarrisonRoomBarracks` | "barracks" | Combat | Yes | Both |
| `GarrisonRoomGuardPost` | "guard_post" | Combat | Yes | Entry room always |
| `GarrisonRoomArmory` | "armory" | Combat | Yes | Both |
| `GarrisonRoomCommandPost` | "command_post" | Combat + Mana reward | Yes | Branch only |
| `GarrisonRoomPatrolRoute` | "patrol_route" | Combat | Yes | Both |
| `GarrisonRoomMageTower` | "mage_tower" | Combat | Yes | Branch only |
| `GarrisonRoomRestRoom` | "rest_room" | Recovery (no combat) | No | Branch only |
| `GarrisonRoomStairs` | "stairs" | Floor progression | No | End of critical path |

---

## 8. Squad Archetypes and Assignment

### Archetype Definitions

Garrison squad archetypes are defined as Go data in `mind/raid/archetypes.go`. Each archetype specifies:

- A name and display name.
- A list of units with monster template names and grid positions.
- A list of preferred room types.

There are currently **8 archetypes**:

| Name | Display Name | Units | Preferred Rooms | Category |
|------|-------------|-------|-----------------|----------|
| `chokepoint_guard` | Chokepoint Guard | Knight x2, Crossbowman x2, Priest | guard_post | Critical path |
| `shield_wall` | Shield Wall | Ogre (2x2), Archer x2, Cleric | barracks, armory | Critical path |
| `ranged_battery` | Ranged Battery | Spearman, Marksman x2, Archer, Mage | mage_tower | Branch |
| `fast_response` | Fast Response | Swordsman x2, Goblin Raider x2, Scout | patrol_route | Branch / Reserve |
| `mage_tower` | Mage Tower | Battle Mage, Wizard x2, Warlock, Sorcerer | mage_tower | Elite |
| `ambush_pack` | Ambush Pack | Assassin x2, Rogue x2, Ranger | patrol_route | Branch / Reserve |
| `command_post` | Command Post Guard | Knight, Paladin, Crossbowman, Cleric, Priest | command_post | Elite |
| `orc_vanguard` | Orc Vanguard | Orc Warrior (2x1), Ogre (2x2), Warrior x2 | barracks | Critical path |

### Assignment Algorithm

`AssignArchetypesToFloor()` in `assignment.go` maps each combat room to an archetype. The algorithm:

1. **Skip non-combat rooms** -- Rest rooms and stairs get no garrison.
2. **Check preferred rooms** -- If any archetype has the room type in its `PreferredRooms` list, randomly select one of those preferred archetypes.
3. **Fall back to path-based selection** -- If no preferred match exists:
   - Critical path rooms get archetypes from `criticalPathArchetypes` config.
   - Branch rooms get archetypes from `branchArchetypes` config.
   - On floors >= `eliteFloorThreshold` (default 3), branch rooms may also receive `eliteArchetypes`.

This design ensures thematic consistency (guard posts get "Chokepoint Guard", mage towers get "Ranged Battery") while providing variety through random selection within categories.

---

## 9. Deployment System

Before each combat encounter, the player can choose which squads to deploy (send into battle) and which to hold in reserve (better recovery afterward).

### Manual Deployment

`SetDeployment()` in `deployment.go` validates and stores the deployment split:

- At least 1 squad must be deployed.
- Cannot exceed `maxDeployedPerEncounter` (default 3).
- Cannot deploy destroyed squads.

### Auto-Deploy

`AutoDeploy()` automatically selects the strongest available squads:

1. Calculates power score for each living player squad using `evaluation.CalculateSquadPower()`.
2. Sorts by power (descending).
3. Deploys the top N squads (up to `maxDeployedPerEncounter`).
4. Places remaining squads in reserve.

This uses the "balanced" power profile from the evaluation system, ensuring consistent power measurement.

### Recovery Differentiation

The deployment split directly affects post-encounter recovery:
- **Deployed squads** receive `deployedHPPercent` (12%) HP recovery.
- **Reserve squads** receive `reserveHPPercent` (27%) HP recovery.

This creates a meaningful strategic decision: deploying all squads maximizes combat power but minimizes recovery, while holding squads in reserve preserves them for later encounters.

---

## 10. Alert System

The alert system is a per-floor escalation mechanic that makes the garrison progressively harder as the player fights more encounters.

### Alert Levels

Alert escalates through four levels, tracked in `AlertData.CurrentLevel`:

| Level | Name | Encounters Required | Armor Bonus | STR Bonus | Weapon Bonus | Activates Reserves |
|-------|------|-------------------|-------------|-----------|--------------|-------------------|
| 0 | Unaware | 0 | +0 | +0 | +0 | No |
| 1 | Suspicious | 2 | +1 | +0 | +0 | No |
| 2 | Alerted | 4 | +1 | +1 | +0 | Yes |
| 3 | Lockdown | 6 | +1 | +2 | +1 | Yes |

### Escalation Flow

`IncrementAlert()` is called after every encounter:

```
IncrementAlert(manager, floorNumber)
  |
  +-- Increment EncounterCount
  +-- Check config thresholds (highest qualifying level wins)
  +-- If level changed:
        +-- ActivateReserves(manager, floorNumber)
              |
              +-- Find eligible rooms (accessible, uncleared, combat type)
              +-- Move reserve squads into eligible rooms
              +-- Update GarrisonSquadData (IsReserve=false, RoomNodeID set)
              +-- Update room's GarrisonSquadIDs
```

### Reserve Activation

When `ActivatesReserves` is true for the new alert level, reserve squads are moved from the floor's reserve pool into accessible combat rooms. This adds new enemies to rooms the player has not yet cleared, increasing difficulty. Reserve squads are moved one-to-one into eligible rooms (accessible, uncleared, and not rest rooms or stairs).

---

## 11. Combat Integration

The raid system integrates with the existing combat system through two key connection points: `SetupRaidFactions()` and `EncounterService.BeginRaidCombat()`.

### Faction Setup

`SetupRaidFactions()` in `raidencounter.go` creates combat factions for a raid encounter:

1. Creates a "Raid Attackers" faction (player, controlled by Player 1).
2. Creates a "Garrison Defenders" faction (AI-controlled).
3. Positions player squads left of the combat position with `playerOffsetX = -3`, `playerOffsetY = -2`.
4. Positions garrison squads right of the combat position with `enemyOffsetX = 3`, `enemyOffsetY = 2`.
5. Calls `encounter.EnsureUnitPositions()` to place individual units.
6. Calls `combat.CreateActionStateForSquad()` to initialize combat state.

```
Player Squads                  Garrison Squads
    [-3, -2]      [CombatPos]      [+3, +2]
    [-1, -2]       (50, 40)        [+5, +2]
    [+1, -2]                       [+7, +2]
```

Multiple squads are spread horizontally using `squadSpreadX = 2`.

### Combat Mode Transition

`RaidRunner.TriggerRaidEncounter()` orchestrates the transition:

1. Snapshots living unit counts for morale penalty calculation.
2. Gets deployed squad IDs from the deployment entity.
3. Calls `SetupRaidFactions()` to create factions and position squads.
4. Calls `encounterService.BeginRaidCombat()` which:
   - Saves the player's original position.
   - Sets `TacticalState.PostCombatReturnMode = "raid"` (so combat returns to raid mode, not exploration).
   - Creates `ActiveEncounter` with `IsRaidCombat = true` (skips overworld resolution).
   - Transitions to "combat" mode via `modeCoordinator.EnterTactical("combat")`.

### Post-Combat Callback

When combat ends, the `EncounterService.ExitCombat()` method fires `PostCombatCallback`, which was registered by `RaidRunner` at construction time:

```go
encounterService.PostCombatCallback = func(reason encounter.CombatExitReason, result *encounter.CombatResult) {
    if rr.raidEntityID != 0 {
        rr.ResolveEncounter(reason, result)
    }
}
```

The combat mode then transitions to the "raid" mode (via `PostCombatReturnMode`), where `RaidMode.Enter()` detects the return from combat and shows the encounter summary panel.

### IsRaidCombat Flag

The `ActiveEncounter.IsRaidCombat` flag is critical. When true:
- `EncounterService.EndEncounter()` skips overworld resolution entirely (no threat marking, no sprite toggling).
- The raid system handles its own resolution through `RaidRunner.ResolveEncounter()`.

---

## 12. Resolution and Recovery

### Victory Resolution

`ProcessVictory()` in `resolution.go` handles a successful encounter:

1. Marks all garrison squads in the room as destroyed (`IsDestroyed = true`).
2. Increments `GarrisonKillCount` on the raid state.
3. Calls `MarkRoomCleared()` which:
   - Sets `IsCleared = true` on the room.
   - Increments `FloorState.RoomsCleared`.
   - Recalculates accessibility for all child rooms.
4. Applies victory morale bonus to all player squads.
5. Grants room-specific rewards.
6. Checks if the floor is now complete (stairs cleared).

### Defeat Resolution

`ProcessDefeat()` sets `RaidStatus = RaidDefeat` and applies morale penalties to all player squads. The raid ends immediately.

### End Condition Checks

`CheckRaidEndConditions()` evaluates whether the raid should end:
- **Defeat**: All player squads are destroyed.
- **Victory**: Current floor is the last floor AND it is complete.
- **Active**: Otherwise, the raid continues.

### Post-Encounter Processing

`RaidRunner.PostEncounterProcessing()` runs after every encounter resolution:

1. Apply differentiated HP recovery (deployed vs. reserve).
2. Increment alert level and potentially activate reserves.
3. Check end conditions.

### Between-Floor Recovery

When advancing to a new floor via `AdvanceFloor()`:
- All player squads receive `deployedHPPercent` HP recovery.
- All player squads receive `betweenFloorMoraleBonus` morale.

### HP Recovery Implementation

```go
func applyHPRecovery(manager *common.EntityManager, squadID ecs.EntityID, hpPercent int) {
    unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
    for _, unitID := range unitIDs {
        attr := common.GetComponentTypeByID[*common.Attributes](manager, unitID, common.AttributeComponent)
        if attr != nil && attr.CurrentHealth > 0 {
            heal := attr.MaxHealth * hpPercent / 100
            attr.CurrentHealth += heal
            if attr.CurrentHealth > attr.MaxHealth {
                attr.CurrentHealth = attr.MaxHealth
            }
        }
    }
}
```

Important: Dead units (CurrentHealth <= 0) do not receive healing. Only living units are restored.

---

## 13. Reward System

Room-specific rewards are granted through `GrantRoomReward()` in `rewards.go`. Currently, only one reward type is implemented:

### Command Post Reward

Clearing a command post room restores `commandPostManaRestore` (default 15) mana to the player's commander. This uses the `spells.ManaComponent` to find the commander's mana pool.

```go
manaData.CurrentMana += manaRestore
if manaData.CurrentMana > manaData.MaxMana {
    manaData.CurrentMana = manaData.MaxMana
}
```

The reward text is returned to the GUI for display in the encounter summary panel.

### Extension Point

Other room types could grant rewards by adding cases to the `GrantRoomReward()` switch statement. Potential future rewards might include:
- Armory rooms granting equipment upgrades.
- Barracks rooms granting recruitment tokens.
- Mage tower rooms granting spell components.

---

## 14. GUI Layer

The raid GUI consists of a single `UIMode` (`RaidMode`) that manages three sub-panels, switching between them based on the current phase of the raid interaction.

### Panel Architecture

```
RaidMode (framework.UIMode)
  |
  +-- RaidUIState (UI-only state)
  |     SelectedRoomID, CurrentPanel, ShowingSummary, SummaryData
  |
  +-- FloorMapPanel (main view)
  |     Shows room list, alert level, room selection buttons
  |
  +-- DeployPanel (pre-combat)
  |     Shows squad list with HP/morale, auto-deploy, start battle
  |
  +-- SummaryPanel (post-combat)
        Shows encounter outcome, units lost, rewards
```

### Panel Registry

Panels are constructed via the framework's panel registry system in `raid_panels_registry.go`. Each panel registers a `PanelDescriptor` with a creation callback that builds the widget tree. Custom widgets (labels, buttons) are stored in a `Custom` map for dynamic access by the controller classes.

Panel size constants are defined in `gui/specs/layout.go`:

| Panel | Width | Height |
|-------|-------|--------|
| Floor Map | 80% | 85% |
| Deploy | 70% | 75% |
| Summary | 60% | 65% |

### Panel Visibility

Only one panel is visible at a time. `showPanel()` toggles visibility using ebitenui's `widget.Visibility_Show` and `widget.Visibility_Hide_Blocking`:

```go
func (rm *RaidMode) showPanel(panel RaidPanel) {
    rm.state.CurrentPanel = panel
    floorContainer.GetWidget().Visibility = visibilityFor(panel == PanelFloorMap)
    deployContainer.GetWidget().Visibility = visibilityFor(panel == PanelDeploy)
    summaryContainer.GetWidget().Visibility = visibilityFor(panel == PanelSummary)
}
```

### FloorMapPanel

The floor map panel is the primary raid interface. It displays:

- **Title**: "Garrison Raid -- Floor X/Y"
- **Alert level**: Current level name and encounter count.
- **Room list**: All rooms on the current floor with status (Locked/Accessible/Cleared), garrison squad count, critical path marker.
- **Room buttons**: Clickable buttons for each accessible, uncleared room.
- **Retreat button**: Ends the raid with Retreated status.

Room buttons are dynamically rebuilt each time the panel refreshes. When a room is selected:
- Non-combat rooms (rest, stairs) are processed immediately by `RaidRunner.SelectRoom()`.
- Combat rooms transition to the deploy panel.

**Keyboard shortcuts**: Number keys 1-9 select accessible rooms by index.

### DeployPanel

The deploy panel shows before each combat encounter:

- **Title**: Room type, node ID, and garrison squad count.
- **Squad list**: Each player squad with name, HP%, morale, living/total units, and status (Ready/Destroyed).
- **Auto Deploy button**: Runs `AutoDeploy()` to select strongest squads.
- **Start Battle button**: Confirms deployment and triggers combat.
- **Back button**: Returns to floor map.

**Keyboard shortcuts**: Enter starts battle, Escape goes back.

### SummaryPanel

The summary panel appears after returning from combat:

- Room name and type.
- Units lost count.
- Current alert level.
- Reward text (if any).
- Continue button.

**Keyboard shortcuts**: Enter or Space dismisses the summary.

### GUI Flow Diagram

```
[Enter Raid Mode]
       |
       v
  [FloorMapPanel]  <----+
       |                 |
  (select room)          |
       |                 |
  combat room?           |
    /       \            |
  yes        no          |
   |          |          |
   v          v          |
[DeployPanel] (process)--+
   |                     |
 (confirm)               |
   |                     |
   v                     |
[Combat Mode]            |
   |                     |
 (combat ends)           |
   |                     |
   v                     |
[SummaryPanel]           |
   |                     |
 (continue)              |
   |                     |
   +---------------------+
```

### Auto-Start Behavior

When `RaidMode.Enter()` is called and no raid is active, the mode automatically starts a raid using `autoStartRaid()`. This:

1. Finds the player's first commander.
2. Gets the commander's squad roster.
3. Limits squads to `MaxPlayerSquads()`.
4. Calls `raidRunner.StartRaid()`.
5. Enters floor 1.

This auto-start behavior serves as the debug entry point. In a production implementation, the player would likely configure the raid (select squads, choose difficulty) before starting.

---

## 15. Raid Lifecycle -- End to End

This section traces the complete lifecycle of a raid from initiation to completion.

### Phase 1: Initialization

```
1. Player triggers raid (currently via debug menu -> raid mode)
2. RaidMode.Enter() -> autoStartRaid()
3. Find commander and squads
4. RaidRunner.StartRaid(commanderID, squadIDs, floorCount)
   -> GenerateGarrison() creates all ECS entities:
      - RaidStateData (singleton)
      - Per floor: FloorStateData, AlertData, RoomData entities, GarrisonSquad entities
5. RaidRunner.EnterFloor(1) sets CurrentFloor
6. FloorMapPanel.Refresh() displays floor 1 rooms
```

### Phase 2: Room Selection

```
7. Player sees room list with accessibility status
8. Player clicks room button (or presses number key)
9. RaidMode.OnRoomSelected(nodeID)
   -> If rest room: RaidRunner.SelectRoom() applies recovery, marks cleared
   -> If stairs: marks cleared, sets floor complete
   -> If combat room: show DeployPanel
```

### Phase 3: Deployment

```
10. DeployPanel.Refresh() shows squad list with HP/morale
11. Player optionally clicks "Auto Deploy"
    -> AutoDeploy() sorts by power, selects top N
12. Player clicks "Start Battle"
13. RaidMode.OnDeployConfirmed()
    -> If no deployment set, AutoDeploy() is called
    -> RaidRunner.TriggerRaidEncounter(nodeID)
```

### Phase 4: Combat Setup

```
14. Snapshot pre-combat alive counts
15. Get deployed squad IDs from DeploymentData
16. SetupRaidFactions() creates factions, positions squads
17. encounterService.BeginRaidCombat()
    -> Save player position
    -> Set PostCombatReturnMode = "raid"
    -> Create ActiveEncounter (IsRaidCombat = true)
    -> EnterTactical("combat") -> combat mode starts
```

### Phase 5: Combat

```
18. Standard tactical combat plays out (turns, attacks, movement)
19. Victory/defeat determined by CombatService
20. CombatTurnFlow calls encounterService.ExitCombat()
    -> EndEncounter() skips overworld resolution (IsRaidCombat)
    -> RecordEncounterCompletion() saves history
    -> PostCombatCallback fires -> RaidRunner.ResolveEncounter()
21. Combat mode transitions to "raid" (PostCombatReturnMode)
```

### Phase 6: Resolution

```
22. RaidRunner.ResolveEncounter(reason, result)
    -> Calculate units lost (pre vs. post alive counts)
    -> If victory: ProcessVictory() marks room cleared, morale bonus, rewards
    -> If defeat: ProcessDefeat() sets RaidDefeat status
    -> PostEncounterProcessing():
       a. ApplyPostEncounterRecovery() (deployed vs. reserve differentiation)
       b. IncrementAlert() (may activate reserves)
       c. CheckRaidEndConditions()
    -> Build RaidEncounterResult for GUI display
```

### Phase 7: Summary and Continuation

```
23. RaidMode.Enter() detects return from combat
24. OnCombatComplete() shows SummaryPanel with encounter results
25. Player clicks "Continue" or presses Enter
26. OnSummaryDismissed() returns to FloorMapPanel
    -> If floor complete and not last floor: AdvanceFloor()
       - ApplyBetweenFloorRecovery()
       - EnterFloor(nextFloor)
    -> If floor complete and last floor: "Raid complete!"
    -> Otherwise: continue on current floor
27. Loop back to Phase 2 (room selection)
```

### Phase 8: Termination

The raid ends when any of these conditions are met:

| Condition | Status | Trigger |
|-----------|--------|---------|
| All player squads destroyed | `RaidDefeat` | `CheckRaidEndConditions()` |
| Player loses a combat encounter | `RaidDefeat` | `ProcessDefeat()` |
| Final floor stairs cleared | `RaidVictory` | `AdvanceFloor()` or `CheckRaidEndConditions()` |
| Player clicks Retreat | `RaidRetreated` | `RaidRunner.Retreat()` |

When the raid finishes:
```go
func (rr *RaidRunner) finishRaid(status RaidStatus) {
    rr.encounterService.PostCombatCallback = nil  // Clear callback
    rr.raidEntityID = 0                            // Clear runner state
}
```

---

## 16. Integration with Other Systems

### Squads (`tactical/squads/`)

The raid system creates garrison squads using `squads.CreateSquadFromTemplate()`, ensuring they are proper squad entities with all standard components. Key integration points:

- `squads.GetTemplateByName()` -- Looks up monster templates for archetype units.
- `squads.GetUnitIDsInSquad()` -- Gets unit entity IDs for HP recovery and alive counting.
- `squads.IsSquadDestroyed()` -- Checks if a squad has no living units.
- `squads.GetSquadHealthPercent()` -- Used by the deploy panel to show squad HP%.
- `squads.SquadData.Morale` -- Read and written for morale bonuses/penalties.
- `squads.SquadData.IsDeployed` -- Set to true when entering combat.

### Combat (`tactical/combat/`)

- `combat.NewCombatQueryCache()` -- Required for faction manager creation.
- `combat.NewCombatFactionManager()` -- Creates factions for raid encounters.
- `combat.CreateActionStateForSquad()` -- Initializes combat action state for each squad.

### Encounter Service (`mind/encounter/`)

The encounter service acts as the bridge between the raid system and the combat mode:

- `BeginRaidCombat()` -- Initiates combat with raid-specific state.
- `ExitCombat()` -- Fires the `PostCombatCallback` that the raid runner listens on.
- `ActiveEncounter.IsRaidCombat` -- Flag that prevents overworld resolution.
- `PostCombatReturnMode` -- Directs the combat exit to return to "raid" mode instead of "exploration".

### Commander (`tactical/commander/`)

- `commander.GetPlayerCommanderRoster()` -- Used by `autoStartRaid()` to find the player's commander.
- `RaidStateData.CommanderID` -- Stored for mana reward distribution.

### Evaluation (`mind/evaluation/`)

- `evaluation.CalculateSquadPower()` -- Used by `AutoDeploy()` to rank squads by strength.
- `evaluation.GetPowerConfigByProfile("balanced")` -- Power calculation configuration.

### Spells (`tactical/spells/`)

- `spells.ManaData` -- Read and written by command post reward to restore commander mana.

### World Map (`world/worldmap/`)

The worldmap package provides the DAG generation infrastructure:

- `worldmap.BuildGarrisonDAG()` -- Constructs the abstract room graph.
- `worldmap.FloorDAG`, `FloorNode` -- DAG data structures.
- `worldmap.GarrisonRoom*` constants -- Room type identifiers.
- `GarrisonRaidGenerator` -- Registered map generator for physical map rendering.

### GUI Framework (`gui/framework/`)

- `framework.UIMode` interface -- `RaidMode` implements this.
- `framework.PanelType` / `framework.RegisterPanel()` -- Panel registration.
- `framework.UIModeManager` -- Mode switching.
- `framework.TacticalState.PostCombatReturnMode` -- Controls post-combat destination.

---

## 17. Map Generation for Raids

The worldmap package provides full physical map generation for garrison raids, though the current raid system primarily uses the abstract DAG. The physical map generation is available through the `GarrisonRaidGenerator` registered as `"garrison_raid"`.

### DAG Construction (`gen_garrison_dag.go`)

`BuildGarrisonDAG()` constructs the abstract floor graph:

1. **Build critical path**: A chain of combat rooms from entry (always guard_post) to stairs. Length varies by floor (see floor scaling).
2. **Attach branch chains**: Branch rooms are attached off critical-path nodes until the target room count is reached. At most one rest room per floor. Branches may optionally reconnect to downstream critical-path nodes, creating diamond/merge structures.
3. **Chain extension**: 40% chance to chain a second room off each branch.

### Floor Scaling

Room count and complexity scale by floor number:

| Floor | Critical Path | Total Rooms | Unlocked Room Types |
|-------|--------------|-------------|---------------------|
| 1 | 3-4 | 6-8 | guard_post, barracks, patrol_route, rest_room, armory |
| 2 | 3-4 | 7-9 | Same as floor 1 |
| 3 | 4-5 | 8-10 | + command_post, mage_tower |
| 4 | 3-4 | 7-9 | Same as floor 3 |
| 5 | 3-4 | 6-8 | Same as floor 3 |

### Physical Map Generation (`gen_garrison.go`)

The `GarrisonRaidGenerator.Generate()` method converts the DAG to a physical tile map:

1. **Topological sort** the DAG for placement order.
2. **Calculate depth** of each node for left-to-right positioning.
3. **Place rooms** as rectangles on the tile grid, using depth-based X bands and center-biased Y positions. Multiple fallback strategies handle room placement failures (shrinking, reduced padding, forced placement).
4. **Carve corridors** between connected rooms. L-shaped corridors for critical path connections, Z-shaped (dogleg) corridors for branches (breaks line of sight).
5. **Place doorways** -- 1-tile-wide openings where corridors meet rooms.
6. **Ensure connectivity** via flood fill and corridor patching.
7. **Inject tactical terrain** per room type.

### Per-Room Terrain (`gen_garrison_terrain.go`)

Each room type has 2-3 terrain layout variants randomly selected at generation time:

| Room Type | Variant Count | Terrain Features |
|-----------|--------------|-----------------|
| Guard Post | 3 | Double pillars, staggered walls, kill box (U-alcove) |
| Barracks | 3 | Scattered pillars, bunk rows, training yard (pillar ring) |
| Armory | 2 | Barricade line with gaps, weapon racks (parallel walls) |
| Command Post | 2 | U-shaped rear alcove, war table (3x3 central block) |
| Patrol Route | 2 | Sparse cover pillars, border pillar rows |
| Mage Tower | 2 | Staggered pillar grid, arcane corridor (winding walls) |

Rest rooms and stairs have no terrain injection.

### Spawn Position Computation (`gen_garrison_meta.go`)

The metadata system computes spawn positions for each room:

- **Player spawns**: Positioned in a 4-tile-deep strip at the entry edge (determined by parent room direction).
- **Defender spawns**: Positioned in room-type-specific zones:
  - Guard posts and command posts: Rear 40% (anchored behind terrain).
  - Barracks and armories: Rear half (distributed).
  - Patrol routes and mage towers: Rear third (spread wide).

All spawn positions are validated for 3x3 clear area and minimum spacing.

---

## 18. Design Decisions and Rationale

### Why Pre-Composed Archetypes Instead of Power-Budget Generation?

The standard encounter system generates enemy squads from a power budget (see `mind/encounter/`). The raid system instead uses pre-composed archetypes. This was chosen because:

- **Thematic consistency**: Each room type should feel distinct. A guard post should have Knights and Crossbowmen, not random creatures.
- **Predictability**: Players can learn what each room type defends with, enabling strategic planning about which rooms to tackle and in what order.
- **Balance control**: Pre-composed squads can be individually balanced and tested, rather than relying on power budget calculations.

### Why a DAG Instead of a Grid?

The room graph is a directed acyclic graph rather than a 2D grid. This enables:

- **Non-linear but ordered progression**: Players can choose between multiple paths without backtracking.
- **Critical path with optional branches**: The game guarantees a minimum path length while offering optional side content.
- **Diamond structures**: Branch rooms can reconnect to downstream critical-path nodes, creating interesting routing decisions.

### Why Differentiated Recovery?

Reserve squads receive more than twice the HP recovery of deployed squads (27% vs. 12%). This creates a meaningful deployment decision: the player must balance using all available force (deploy everything) against preserving squad health for later encounters (hold some in reserve).

### Why Per-Floor Alert Instead of Global?

Alert is tracked per-floor rather than persisting across floors. This ensures each floor starts at a baseline difficulty, preventing the first floor from making later floors impossibly hard. The increasing floor complexity (more rooms, elite archetypes) provides sufficient difficulty scaling.

### Why Is RaidRunner Not an ECS System?

The `RaidRunner` struct coordinates flow between multiple ECS components and external services (encounter service, combat mode). Making it a pure ECS system would require:
- Storing encounter service references in components (violating pure data).
- Creating complex state machines in ECS queries.
- Losing the clear sequential flow of encounter -> resolve -> recover -> next.

Instead, `RaidRunner` is a service that reads/writes ECS state while managing the procedural flow. This is the same pattern used by `EncounterService`.

---

## 19. Current Limitations and TODOs

### Known Limitations

1. **Debug-only entry point**: Raids are only accessible through the debug menu in roguelike mode. No in-game UI exists to initiate a raid from the overworld or a menu.

2. **No manual squad selection**: The `autoStartRaid()` function takes all available squads. There is no UI for the player to choose which squads to bring on a raid.

3. **Alert buffs not applied to units**: The `AlertLevelConfig` defines `ArmorBonus`, `StrengthBonus`, and `WeaponBonus`, but `IncrementAlert()` only activates reserves -- the stat bonuses are not applied to garrison units. This is noted as a config-driven feature that needs wiring.

4. **Morale thresholds not applied**: The `morale.thresholds` config defines `DexPenalty` and `StrPenalty` at low morale levels, but there is no system function that applies these penalties to squad units during combat.

5. **Unit death morale penalty unused**: The config defines `unitDeathMoralePenalty` but the `ResolveEncounter()` method only counts total units lost -- it does not apply per-death morale penalties. The pre-combat alive count snapshot is captured but only used for summary display.

6. **No visual floor map**: The `FloorMapPanel.Render()` method is a stub. Room connections are shown only as a text list, not as a visual graph with lines between nodes.

7. **No squad selection in deploy panel**: The deploy panel shows squad info but does not provide toggle buttons for individual squad deployment. The player can only use "Auto Deploy" or "Start Battle" (which triggers auto-deploy if none is set).

8. **GarrisonKillCount** display: The `RaidStateData.GarrisonKillCount` field has a TODO comment: "Wire to summary display or victory screen."

9. **Physical map generation unused by raid flow**: The `GarrisonRaidGenerator` is registered and functional, but the current raid system only uses the abstract DAG. Combat takes place at a fixed world position (`combatPositionX/Y`), not in the physical garrison rooms. The generated room terrain, spawn positions, and corridors are not used during raid combat.

### Suggested Improvements

- Wire alert stat bonuses to garrison units before combat.
- Implement per-death morale penalties using `preCombatAliveCounts`.
- Add visual DAG rendering in the floor map panel.
- Add toggle buttons for manual squad deployment selection.
- Create a proper raid initiation UI (squad selection, difficulty selection).
- Use generated physical maps for raid combat positioning.
- Add more room-type rewards (armory, mage tower).

---

## 20. File Reference

### Core Package (`mind/raid/`)

| File | Lines | Primary Contents |
|------|-------|-----------------|
| `init.go` | 39 | Subsystem registration, component/tag initialization |
| `components.go` | 107 | All ECS component structs, RaidStatus enum |
| `config.go` | 215 | JSON config loading, accessor functions with defaults |
| `archetypes.go` | 136 | 8 garrison squad archetype definitions |
| `garrison.go` | 164 | `GenerateGarrison()`, `InstantiateGarrisonSquad()` |
| `floorgraph.go` | 126 | Room DAG navigation, accessibility, `MarkRoomCleared()` |
| `assignment.go` | 66 | `AssignArchetypesToFloor()`, archetype selection logic |
| `deployment.go` | 103 | `SetDeployment()`, `AutoDeploy()` |
| `alert.go` | 125 | `IncrementAlert()`, `ActivateReserves()` |
| `recovery.go` | 97 | HP and morale recovery functions |
| `resolution.go` | 112 | `ProcessVictory()`, `ProcessDefeat()`, `CheckRaidEndConditions()` |
| `rewards.go` | 48 | `GrantRoomReward()`, command post mana reward |
| `raidencounter.go` | 99 | `SetupRaidFactions()`, combat faction creation |
| `raidrunner.go` | 358 | `RaidRunner` struct, full raid lifecycle orchestration |
| `queries.go` | 112 | ECS query functions for all raid entities |

### GUI Package (`gui/guiraid/`)

| File | Lines | Primary Contents |
|------|-------|-----------------|
| `raidstate.go` | 29 | `RaidUIState`, `RaidPanel` enum |
| `raidmode.go` | 314 | `RaidMode` UIMode, panel switching, event handlers |
| `raid_panels_registry.go` | 205 | Panel widget construction, framework registration |
| `floormap_panel.go` | 184 | Floor map display, room buttons, keyboard input |
| `deploy_panel.go` | 162 | Squad list, auto-deploy, battle confirmation |
| `summary_panel.go` | 92 | Post-encounter result display |

### World Map Package (`world/worldmap/`)

| File | Primary Contents |
|------|-----------------|
| `gen_garrison_dag.go` | DAG construction, room types, floor scaling |
| `gen_garrison.go` | Physical map generation, room placement, corridors |
| `gen_garrison_meta.go` | Spawn positions, room metadata |
| `gen_garrison_terrain.go` | Per-room tactical terrain injection |

### Configuration

| File | Description |
|------|-------------|
| `assets/gamedata/raidconfig.json` | All tunable raid parameters |

### Setup and Integration

| File | Relevant Code |
|------|---------------|
| `game_main/setup_shared.go` | `raid.LoadRaidConfig()` call at startup |
| `game_main/setup_roguelike.go` | `RaidRunner` creation, mode registration |
| `mind/encounter/encounter_service.go` | `BeginRaidCombat()`, `PostCombatCallback` |
| `mind/encounter/types.go` | `ActiveEncounter.IsRaidCombat` |
| `gui/guicombat/combat_turn_flow.go` | `PostCombatReturnMode` handling |
| `gui/specs/layout.go` | `RaidFloorMapWidth/Height`, `RaidDeployWidth/Height`, `RaidSummaryWidth/Height` |
