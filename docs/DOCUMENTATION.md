# TinkerRogue Technical Documentation

**Version:** 5.0 (2026 Edition)
**Last Updated:** 2026-03-12
**Project Type:** Turn-based tactical roguelike with squad combat
**Language:** Go 1.x
**ECS Library:** bytearena/ecs
**UI Framework:** Ebitengine v2 + ebitenui

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

---

## Reading Paths

**For New Developers:**
1. Start with [Architecture Overview](#architecture-overview)
2. Read [Entity Component System (ECS)](Other/ECS_BEST_PRACTICES.md)
3. Study [Squad & Combat System](Systems/SQUAD_COMBAT_SYSTEM.md) as reference implementation

**For Gameplay Programmers:**
1. [Squad & Combat System](Systems/SQUAD_COMBAT_SYSTEM.md) - Formation-based combat, turn management
2. [Artifact & Inventory System](Systems/ARTIFACT_SYSTEM.md) - Item handling

**For UI Developers:**
1. [GUI Architecture](Other/GUI_DOCUMENTATION.md) - Mode management
2. [Data Flow Patterns](Other/DATA_FLOW_PATTERNS.md) - UI to ECS integration

**For Systems Programmers:**
1. [Project Layout](PROJECT_LAYOUT.md) - Full package tree
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

### Package Structure

For the full annotated package tree with file-level descriptions, see [PROJECT_LAYOUT.md](PROJECT_LAYOUT.md).

---

## Documentation Index

### Systems
- [Squad & Combat System](Systems/SQUAD_COMBAT_SYSTEM.md) — Formation-based combat, turn management, damage resolution
- [Overworld Architecture](Systems/OVERWORLD_ARCHITECTURE.md) — Tick system, factions, influence, strategic layer
- [Artifact & Inventory System](Systems/ARTIFACT_SYSTEM.md) — Items, equipment, consumables, throwables
- [World Generation](Systems/WORLD_GENERATION.md) — Generator registry, map algorithms, tile/biome system
- [Raids](Systems/RAIDS.md) — Garrison defense encounters
- [Save System](Systems/savesystem.md) — Chunk-based serialization

### AI
- [AI Algorithm Architecture](AI/AI_ALGORITHM_ARCHITECTURE.md) — Utility AI, action evaluation
- [AI Controller](AI/AI_CONTROLLER.md) — AI entry point and orchestration
- [AI Configuration](AI/AI_CONFIGURATION.md) — JSON config files for AI behavior
- [Behavior & Threat Layers](AI/BEHAVIOR_THREAT_LAYERS.md) — Threat painting, danger maps
- [Power Evaluation](AI/POWER_EVALUATION.md) — Power scoring, role evaluation
- [Encounter System](AI/ENCOUNTER_SYSTEM.md) — Encounter triggers, setup, resolution
- [Config Tuning Guide](AI/CONFIG_TUNING_GUIDE.md) — Tunable parameters and balancing

### Infrastructure
- [Entity Component System (ECS)](Other/ECS_BEST_PRACTICES.md) — Core ECS patterns and component access
- [GUI Architecture](Other/GUI_DOCUMENTATION.md) — Mode management, panel building
- [Data Flow Patterns](Other/DATA_FLOW_PATTERNS.md) — How data moves through the system
- [Caching Overview](Other/CACHING_OVERVIEW.md) — Evaluation caching strategies
- [Performance Guide](Other/PERFORMANCE_GUIDE.md) — Optimization strategies
- [Game Data Overview](Other/GAMEDATA_OVERVIEW.md) — JSON data files, templates
- [Project Layout](PROJECT_LAYOUT.md) — Full annotated package tree
- [Input Reference](INPUT_REFERENCE.md) — Input system and key bindings
- [Entity Reference](ENTITY_REFERENCE.md) — Entity types and components

---

**Document Version**: 5.0 (2026 Edition)
**Last Updated**: 2026-03-12
**Purpose**: Architecture overview and navigation hub. For critical developer warnings, see CLAUDE.md. For detailed subsystem documentation, follow the links above.
