# TinkerRogue Technical Documentation

**Version:** 4.0 (2026 Edition)
**Last Updated:** 2026-03-02
**Project Type:** Turn-based tactical roguelike with squad combat
**Language:** Go 1.x
**ECS Library:** bytearena/ecs
**UI Framework:** Ebitengine v2 + ebitenui

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Core Systems Directory](#core-systems-directory)
4. [Data Flow Patterns](#data-flow-patterns)
5. [Critical Warnings Quick Reference](#critical-warnings-quick-reference)

**External Documentation:**

*Systems:*
- [Squad & Combat System](Systems/SQUAD_COMBAT_SYSTEM.md)
- [Overworld Architecture](Systems/OVERWORLD_ARCHITECTURE.md)
- [Artifact & Inventory System](Systems/ARTIFACT_SYSTEM.md)
- [World Generation](Systems/WORLD_GENERATION.md)
- [Raids](Systems/RAIDS.md)
- [Save System](Systems/savesystem.md)

*AI:*
- [AI Algorithm Architecture](AI/AI_ALGORITHM_ARCHITECTURE.md)
- [AI Controller](AI/AI_CONTROLLER.md)
- [AI Configuration](AI/AI_CONFIGURATION.md)
- [Behavior & Threat Layers](AI/BEHAVIOR_THREAT_LAYERS.md)
- [Power Evaluation](AI/POWER_EVALUATION.md)
- [Encounter System](AI/ENCOUNTER_SYSTEM.md)
- [Config Tuning Guide](AI/CONFIG_TUNING_GUIDE.md)

*Other:*
- [Entity Component System (ECS)](Other/ECS_BEST_PRACTICES.md)
- [GUI Architecture](Other/GUI_DOCUMENTATION.md)
- [Caching Overview](Other/CACHING_OVERVIEW.md)
- [Performance Guide](Other/PERFORMANCE_GUIDE.md)
- [Game Data Overview](Other/GAMEDATA_OVERVIEW.md)
- [Project Layout](PROJECT_LAYOUT.md)

---

## Executive Summary

TinkerRogue is a turn-based tactical roguelike implemented in Go using a pure Entity Component System (ECS) architecture. The game offers two modes: **Overworld Mode**, a strategic layer where player-controlled commanders traverse a procedurally generated map, manage factions, garrison nodes, and trigger tactical encounters against growing threats; and **Roguelike Mode**, a pure tactical dungeon crawl on cavern maps. Both modes share the same squad-based tactical combat system where players control formations of units in a 3x3 grid layout, casting spells and using artifacts to defeat enemy squads on procedurally generated battle maps.

### Key Architectural Decisions

1. **Pure ECS Architecture**: All game logic follows strict ECS patterns with zero logic in components, EntityID-only relationships, and query-based data access.

2. **Global Coordinate Manager**: A single global `coords.CoordManager` instance handles all coordinate conversions, preventing index-out-of-bounds errors that plagued earlier versions.

3. **O(1) Spatial Queries**: The `GlobalPositionSystem` provides constant-time position lookups using value-based map keys, replacing O(n) linear searches.

4. **Mode-Based GUI**: UI uses a mode manager pattern with context isolation between different game states (overworld, combat, inventory, squad builder).

5. **Template-Based Entity Creation**: Entity spawning uses JSON templates loaded from files, separating data from code.

### Technical Metrics

- **Lines of Code**: ~30,000+ across 170+ files
- **Packages**: 30+ packages (common, coords, tactical, gui, gear, worldmap, visual, overworld, mind, config, etc.)
- **Components**: 50+ registered ECS components
- **Entity Types**: Units, Squads, Items, Tiles, Factions, Combat State, Commanders, Overworld Nodes, Encounters
- **Performance**:
  - Value-based map keys provide 50x improvement over pointer-based keys
  - Cached list widgets reduce CPU usage by ~90%
  - Batched rendering for tiles and sprites
  - O(1) spatial queries via GlobalPositionSystem

### Reading Paths

**For New Developers:**
1. Start with [Architecture Overview](#architecture-overview)
2. Read [Entity Component System (ECS)](Other/ECS_BEST_PRACTICES.md)
3. Study [Squad & Combat System](Systems/SQUAD_COMBAT_SYSTEM.md) as reference implementation

**For Gameplay Programmers:**
1. [Squad & Combat System](Systems/SQUAD_COMBAT_SYSTEM.md) - Formation-based combat, turn management
2. [Artifact & Inventory System](Systems/ARTIFACT_SYSTEM.md) - Item handling

**For UI Developers:**
1. [GUI Architecture](Other/GUI_DOCUMENTATION.md) - Mode management
2. [Data Flow Patterns](#data-flow-patterns) - UI to ECS integration

**For Systems Programmers:**
1. [Core Systems Directory](#core-systems-directory) - Coordinate manager, position system
2. [Performance Guide](Other/PERFORMANCE_GUIDE.md) - Optimization strategies
3. [Entity Component System (ECS)](Other/ECS_BEST_PRACTICES.md) - Component access helpers

**For Overworld/Strategic Developers:**
1. [Overworld Architecture](Systems/OVERWORLD_ARCHITECTURE.md) - Tick system, factions, influence
2. [Encounter System](AI/ENCOUNTER_SYSTEM.md) - Triggers, setup, resolution
3. [Config Tuning Guide](AI/CONFIG_TUNING_GUIDE.md) - Balancing overworld parameters

**For AI Developers:**
1. [AI Algorithm Architecture](AI/AI_ALGORITHM_ARCHITECTURE.md) - Utility AI, action evaluation
2. [Behavior & Threat Layers](AI/BEHAVIOR_THREAT_LAYERS.md) - Threat painting, danger levels
3. [Caching Overview](Other/CACHING_OVERVIEW.md) - Evaluation caching strategies

**For Data/Config Developers:**
1. [Game Data Overview](Other/GAMEDATA_OVERVIEW.md) - JSON data files, templates
2. [Config Tuning Guide](AI/CONFIG_TUNING_GUIDE.md) - Tunable parameters
3. [Artifact & Inventory System](Systems/ARTIFACT_SYSTEM.md) - Item data formats

---

## Architecture Overview

### System Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Start Menu                                    │
│  gui/guistartmenu/ → Overworld Mode OR Roguelike Mode                │
└───────────────┬──────────────────────────────────────────────────────┘
                │
┌───────────────▼──────────────────────────────────────────────────────┐
│                    Game Main Loop (game_main/)                        │
│  setup_shared.go (ECS, data) + setup_overworld/setup_roguelike       │
└───────────────┬──────────────────────────────────────────────────────┘
                │
                ├─ Global Systems (Initialized Once)
                │  ├─ coords.CoordManager (coordinate conversions)
                │  ├─ common.GlobalPositionSystem (O(1) spatial queries)
                │  └─ common.EntityManager (ECS world + tag registry)
                │
                ├─ Tactical Systems (Per-Entity State)
                │  ├─ tactical/squads/ (formation management, abilities)
                │  ├─ tactical/combat/ (turn management, damage resolution)
                │  ├─ tactical/commander/ (commander entities, mana, rosters)
                │  ├─ tactical/spells/ (spell casting, spellbooks)
                │  ├─ tactical/effects/ (active buffs/debuffs)
                │  └─ gear/ (inventory, items, equipment)
                │
                ├─ Overworld Systems (Strategic Layer)
                │  ├─ overworld/tick/ (strategic turn clock)
                │  ├─ overworld/faction/ (NPC faction AI and intents)
                │  ├─ overworld/threat/ (threat node growth)
                │  ├─ overworld/influence/ (influence radius interactions)
                │  ├─ overworld/node/ (unified node management)
                │  ├─ overworld/garrison/ (garrison defense)
                │  └─ overworld/victory/ (win/loss conditions)
                │
                ├─ Mind Systems (AI + Encounters)
                │  ├─ mind/ai/ (utility AI action evaluation)
                │  ├─ mind/behavior/ (threat layers, danger maps)
                │  ├─ mind/evaluation/ (power scoring, role evaluation)
                │  └─ mind/encounter/ (encounter triggers, setup, resolution)
                │
                ├─ GUI Layer (GameModeCoordinator)
                │  ├─ gui/framework/ (coordinator, two UIModeManagers)
                │  ├─ gui/guicombat/ (combat + animation modes)
                │  ├─ gui/guioverworld/ (overworld mode)
                │  ├─ gui/guiexploration/ (exploration mode)
                │  ├─ gui/guisquads/ (squad editor, purchase, deploy, artifacts)
                │  ├─ gui/guinodeplacement/ (node placement mode)
                │  └─ gui/guispells/, gui/guiartifacts/ (spell/artifact panels)
                │
                └─ Input Layer
                   └─ input/cameracontroller.go (WASD movement, map scroll)
```

### Dependency Flow

```
game_main (entry point)
    ↓
common (ECS utilities) ← ALL systems depend on this
    ↓
coords (coordinate conversions) ← Used by rendering, input, worldmap
    ↓
systems (spatial queries) ← Used by combat, movement
    ↓
squads, combat, gear (game logic) ← Independent of each other
    ↓
tactical/commander → overworld/core (commander movement uses walkable grid)
    ↓
overworld/ (tick, faction, threat, influence, node, garrison, victory)
    ↓
mind/ (ai, behavior, evaluation, encounter) → tactical/, overworld/
    ↓
gui (presentation layer) → Reads from ECS, writes commands
    ↓
input (camera controller) → Modifies player position via CameraController
```

**Key Principle**: Dependencies flow downward. GUI and input layers are at the top and depend on lower layers but not vice versa. Game logic systems (squads, combat, gear) are independent and communicate through ECS.

**Key dependency chains:**
- `overworld/` → `common/`, `world/coords/` (spatial data, ECS)
- `mind/encounter/` → `tactical/`, `overworld/`, `gui/framework/` (bridges overworld threats to tactical combat)
- `tactical/commander/` → `overworld/core/` (walkable grid for commander movement)
- `mind/ai/`, `mind/behavior/` → `tactical/squads/`, `tactical/combat/` (AI evaluates squad/combat state)

### Global Instances

TinkerRogue uses three global singleton instances for performance and convenience:

```go
// coords/cordmanager.go
var CoordManager *CoordinateManager  // Initialized in init()

// common/ecsutil.go
var GlobalPositionSystem *systems.PositionSystem  // Initialized in gameinit.go

// EntityManager is NOT global - passed as parameter
// Prevents tight coupling and enables testing
```

**Why These Globals Are Acceptable:**

1. **CoordManager**: Coordinate math is pure and stateless. Having one instance prevents manual width/height passing and eliminates index calculation bugs.

2. **GlobalPositionSystem**: Spatial queries are accessed from many unrelated systems. Global access prevents threading the system through every function call.

3. **EntityManager is NOT Global**: Game state must be testable and isolated. EntityManager is always passed as a parameter to enable multiple game instances and unit testing.

### Package Structure

For the full annotated package tree with file-level descriptions, see [PROJECT_LAYOUT.md](PROJECT_LAYOUT.md).

---

## Core Systems Directory

Brief summaries of each major system with links to detailed documentation.

### Entity Component System (ECS)
Pure data components with zero logic, EntityID-only relationships, query-based data access, and system functions for all behavior. The `common/` package provides type-safe component access helpers and the `EntityManager` hub.
**Details:** [ECS_BEST_PRACTICES.md](Other/ECS_BEST_PRACTICES.md)

### Position System
The `GlobalPositionSystem` provides O(1) spatial queries via a value-keyed hash map (`map[LogicalPosition][]EntityID`). Used by combat, movement, rendering, and AI systems. Entities must be added/removed from the position system alongside ECS lifecycle operations.
**Details:** [ECS_BEST_PRACTICES.md](Other/ECS_BEST_PRACTICES.md), [CACHING_OVERVIEW.md](Other/CACHING_OVERVIEW.md)

### Coordinate Manager
The global `coords.CoordManager` singleton handles all coordinate space conversions: logical grid positions, flat array indices, pixel positions, and screen positions. Critical for tile array indexing.
**Details:** [ECS_BEST_PRACTICES.md](Other/ECS_BEST_PRACTICES.md)

### Squads & Combat
Formation-based tactical combat with 3x3 grid layouts. Squads contain units with roles (tank, healer, DPS). Turn management uses a faction-based system with action points. Includes spell casting, active effects, and artifacts.
**Details:** [SQUAD_COMBAT_SYSTEM.md](Systems/SQUAD_COMBAT_SYSTEM.md), [ARTIFACT_SYSTEM.md](Systems/ARTIFACT_SYSTEM.md)

### Overworld
Strategic layer with tick-based progression. Commanders traverse a procedurally generated map, managing factions, garrisoning nodes, and encountering threats. Includes faction AI, influence radius interactions, threat growth, and victory conditions.
**Details:** [OVERWORLD_ARCHITECTURE.md](Systems/OVERWORLD_ARCHITECTURE.md)

### AI & Encounters
Utility AI evaluates actions by scoring attack value, positioning, and threat levels. Threat layers paint danger maps for tactical positioning. Encounters bridge overworld events to tactical combat, handling setup, resolution, and rewards.
**Details:** [AI_ALGORITHM_ARCHITECTURE.md](AI/AI_ALGORITHM_ARCHITECTURE.md), [ENCOUNTER_SYSTEM.md](AI/ENCOUNTER_SYSTEM.md), [BEHAVIOR_THREAT_LAYERS.md](AI/BEHAVIOR_THREAT_LAYERS.md)

### GUI
Mode-based UI with a `GameModeCoordinator` managing two `UIModeManager` contexts (tactical and overworld). Each game state (combat, exploration, squad editing, overworld) is an isolated `UIMode`. Panel building uses functional options pattern.
**Details:** [GUI_DOCUMENTATION.md](Other/GUI_DOCUMENTATION.md)

### World Generation
Generator registry pattern where algorithms self-register via `init()`. Supports multiple generator types: rooms & corridors, tactical biome, and overworld. Uses tile/biome system with A* pathfinding.
**Details:** [WORLD_GENERATION.md](Systems/WORLD_GENERATION.md)

### Artifacts & Inventory
Pure ECS inventory system with items as entities. Artifacts provide combat effects through the effects system. Equipment slots, consumables, and throwables supported.
**Details:** [ARTIFACT_SYSTEM.md](Systems/ARTIFACT_SYSTEM.md)

### Save System
Chunk-based save system that serializes specific portions of game state independently.
**Details:** [savesystem.md](Systems/savesystem.md)

### Raids
Garrison defense encounters triggered by faction AI during overworld ticks.
**Details:** [RAIDS.md](Systems/RAIDS.md)

---

## Data Flow Patterns

Understanding how data flows through the system is critical for debugging and extending functionality.

### Player Action Flow

```
User Input (Keyboard/Mouse)
    ↓
┌─────────────────────────────────────┐
│ GameModeCoordinator.Update()        │
│   → Active UIModeManager.Update()   │
│   → Active UIMode.Update()          │
│   (handles UI clicks, combat,       │
│    overworld commands, etc.)         │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│ CameraController.HandleInput()      │
│   (WASD movement, diagonals,        │
│    map scroll toggle)               │
└──────────────┬──────────────────────┘
               ↓
ECS components modified
    ↓
System functions process changes
    ↓
Rendering reads ECS state
    ↓
Display updated to screen
```

### Combat Action Flow

```
Player selects attack target (CombatMode input handler)
    ↓
CombatMode.handleAttackAction()
    ↓
combatservices/CombatService.ExecuteSquadAttack()
    ↓
squads/ExecuteSquadAttack() (ECS system function)
    ├─ Query attacker units
    ├─ Query defender units
    ├─ Calculate damage per unit
    ├─ Apply damage to components
    └─ Generate CombatResult
    ↓
GUI reads CombatResult
    ↓
Combat log updated
    ↓
Turn advancement check
```

### Inventory Action Flow

```
Player uses item
    ↓
InventoryMode.onUseItemClicked()
    ↓
gear/InventoryService.UseItem()
    ↓
gear/GetItemByID() (query item component)
    ↓
gear/executeItemAction() (apply effects)
    ├─ If consumable: Apply stat effects to player
    ├─ If throwable: Create projectile entity
    └─ If equipment: Equip to slot
    ↓
gear/RemoveItem() (decrement count)
    ↓
GUI refreshes inventory list
```

### Map Generation Flow

```
Game initialization
    ↓
worldmap/GetGenerator("rooms_corridors")
    ↓
Generator.Generate(width, height, images)
    ├─ Initialize tile array
    ├─ Create rooms (BSP)
    ├─ Connect corridors
    ├─ Place doors
    └─ Collect valid positions
    ↓
GameMap creation
    ↓
Spawn player at valid position
    ↓
GlobalPositionSystem.AddEntity(playerID, startPos)
    ↓
Spawn entities (monsters, items)
    ↓
Rendering displays map
```

### Entity Template Flow

```
Request entity creation (e.g., "goblin")
    ↓
entitytemplates/CreateMonster("goblin")
    ↓
Load template from MonsterTemplates (JSON data)
    ↓
Create entity with components:
    ├─ PositionComponent
    ├─ AttributeComponent (from template stats)
    ├─ NameComponent
    └─ MonsterTag
    ↓
GlobalPositionSystem.AddEntity(entityID, pos)
    ↓
Return entityID to caller
```

### Game Initialization Flow

```
1. main() starts
   ↓
2. SetupSharedSystems() (setup_shared.go)
   ├─ LoadGameData() → JSON templates
   ├─ InitializeCoreECS() → ECS manager, 50+ components, GlobalPositionSystem
   └─ Configure graphics
   ↓
3. Show StartMenu (Overworld vs Roguelike selection)
   ↓
4. Mode-specific setup (setup_overworld.go OR setup_roguelike.go)
   ├─ CreateWorld() → Generate map (overworld or cavern)
   ├─ CreatePlayer() → Player entity, initial commander, starting squads
   ├─ SetupDebugContent() → Test data (if DEBUG_MODE)
   ├─ [Overworld only] InitializeGameplay()
   │   ├─ tick.CreateTickStateEntity()
   │   ├─ commander.CreateOverworldTurnState()
   │   ├─ InitWalkableGrid()
   │   └─ bootstrap.InitializeOverworldFactions()
   ├─ setupUICore() → GameModeCoordinator + EncounterService
   ├─ Register modes (tactical + overworld)
   └─ SetupInputCoordinator() → CameraController
   ↓
5. coordinator.EnterTactical("exploration")
   ↓
6. Run Game Loop (ebiten.RunGame)
   ├─ Update() @ 60 FPS
   └─ Draw() @ 60 FPS
```

### Game Loop Flow

```
┌─────────────────────────────────────────────────┐
│          Update() - 60 FPS                      │
└─────────────────────┬───────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        │                           │
┌───────▼──────────────┐     ┌─────▼──────────┐
│ GameModeCoordinator  │     │ Visual Effects │
│ .Update()            │     │ Update         │
│                      │     └─────┬──────────┘
│ Active UIModeManager │           │
│   → UIMode.Update()  │     ┌─────▼──────────┐
│   (UI + game input)  │     │ CameraController│
└───────┬──────────────┘     │ .HandleInput() │
        │                    │ (WASD movement)│
        │                    └─────┬──────────┘
        └──────────┬───────────────┘
                   │
           ECS state updated
                   │
                   ▼

┌─────────────────────────────────────────────────┐
│          Draw() - 60 FPS                        │
└─────────────────────┬───────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        │                           │
┌───────▼────────┐          ┌──────▼─────────┐
│ Map Rendering  │          │ Entity         │
│ (Tiles)        │          │ Rendering      │
│ [Tactical only]│          │ [Tactical only]│
└───────┬────────┘          └──────┬─────────┘
        │                          │
        └──────────┬───────────────┘
                   │
           ┌───────▼────────┐
           │ Visual Effects │
           │ Rendering      │
           └────────┬───────┘
                    │
           ┌────────▼─────────────┐
           │ GameModeCoordinator  │
           │ .Render()            │
           │ (EbitenUI overlay)   │
           └──────────────────────┘
```

### Overworld Tick Flow

```
Player clicks "End Turn" (OverworldMode)
    ↓
tick.AdvanceTick(manager)
    ├─ Increment TickStateData.CurrentTick
    ├─ faction.ProcessFactionTick() (evaluate intents, execute actions)
    ├─ threat.ProcessThreatGrowth() (grow threat nodes by influence)
    ├─ influence.RecalculateInteractions() (synergy/competition/suppression)
    ├─ garrison.CheckRaids() (detect faction raids on garrisoned nodes)
    └─ victory.CheckVictoryConditions()
    ↓
TickResult (events, pending raids, game-over flag)
    ↓
GUI refreshes overworld panels
    ↓
If pending raid → trigger garrison defense encounter
If game over → show victory/defeat screen
```

### Encounter Flow

```
Commander approaches threat node (overworld movement)
    ↓
encounter.TriggerCombatFromThreat(threatNodeID)
    ↓
EncounterService.StartEncounter()
    ├─ Generate enemy squads from encounter config
    ├─ Generate tactical battle map
    ├─ Create combat state entities
    └─ coordinator.EnterTactical("squad_deployment")
    ↓
SquadDeploymentMode (player places squads on map)
    ↓
CombatMode (turn-based tactical combat)
    ↓
Combat ends (victory/defeat)
    ↓
EncounterService.ResolveEncounter()
    ├─ Award XP and rewards
    ├─ Mark threat node as defeated (or player retreat)
    └─ coordinator.ReturnToOverworld("overworld")
    ↓
OverworldMode resumes
```

---

## Critical Warnings Quick Reference

These are the most common sources of bugs. Each links to the detailed documentation.

### CoordManager Indexing

**ALWAYS** use `coords.CoordManager.LogicalToIndex()` when accessing tile arrays. Manual calculation (`y*width + x`) causes index out of bounds panics because the width might not match `dungeonWidth`.

```go
// CORRECT
tileIdx := coords.CoordManager.LogicalToIndex(logicalPos)

// WRONG - causes panics
idx := y*width + x
```

**Details:** [ECS_BEST_PRACTICES.md](Other/ECS_BEST_PRACTICES.md)

### Entity Lifecycle

When removing entities with positions, always clean up the position system first. Use `manager.CleanDisposeEntity(entity, pos)` for atomic cleanup, or manually call `GlobalPositionSystem.RemoveEntity()` before `DisposeEntities()`.

**Details:** [ECS_BEST_PRACTICES.md](Other/ECS_BEST_PRACTICES.md)

### GUI State Separation

`BattleMapState` / `OverworldState` hold ONLY UI state (selection, mode flags). Game state lives in ECS components. Never store game logic in UI state structures.

**Details:** [GUI_DOCUMENTATION.md](Other/GUI_DOCUMENTATION.md)

### Generator Registration

New worldmap generators must register in `init()` or they won't be discovered by the registry.

**Details:** [WORLD_GENERATION.md](Systems/WORLD_GENERATION.md)

---

**Document Version**: 4.0 (2026 Edition)
**Last Updated**: 2026-03-02
**Total Sections**: 5
**Purpose**: Architecture guide and navigation hub. For detailed subsystem documentation, follow the external links above.
