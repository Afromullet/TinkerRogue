# TinkerRogue Technical Documentation

**Version:** 3.4 (2026 Edition)
**Last Updated:** 2026-02-18
**Project Type:** Turn-based tactical roguelike with squad combat
**Language:** Go 1.x
**ECS Library:** bytearena/ecs
**UI Framework:** Ebitengine v2 + ebitenui

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Core Systems](#core-systems)
4. [Data Flow Patterns](#data-flow-patterns)
5. [Development Patterns](#development-patterns)
6. [Performance Considerations](#performance-considerations)

**External Documentation:**
- [Entity Component System (ECS)](ECS_BEST_PRACTICES.md)
- [Squad & Combat System](SQUAD_COMBAT_SYSTEM.md)
- [World Generation](WORLD_GENERATION.md)
- [GUI Architecture](GUI_DOCUMENTATION.md)
- [Overworld Architecture](OVERWORLD_ARCHITECTURE.md)
- [Artifact & Inventory System](ARTIFACT_SYSTEM.md)
- [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md)
- [Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md)
- [Encounter System](ENCOUNTER_SYSTEM.md)
- [Caching Overview](CACHING_OVERVIEW.md)
- [Config Tuning Guide](CONFIG_TUNING_GUIDE.md)
- [Game Data Overview](GAMEDATA_OVERVIEW.md)

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
2. Read [Entity Component System (ECS)](ECS_BEST_PRACTICES.md)
3. Study [Squad & Combat System](SQUAD_COMBAT_SYSTEM.md) as reference implementation

**For Gameplay Programmers:**
1. [Squad & Combat System](SQUAD_COMBAT_SYSTEM.md) - Formation-based combat, turn management
2. [Artifact & Inventory System](docs/project_documentation/ARTIFACT_SYSTEM.md) - Item handling

**For UI Developers:**
1. [GUI Architecture](docs/gui_documentation/GUI_DOCUMENTATION.md) - Mode management
2. [Data Flow Patterns](#data-flow-patterns) - UI to ECS integration

**For Systems Programmers:**
1. [Core Systems](#core-systems) - Coordinate manager, position system
2. [Performance Considerations](#performance-considerations) - Optimization strategies
3. [Entity Component System (ECS)](ECS_BEST_PRACTICES.md) - Component access helpers

**For Overworld/Strategic Developers:**
1. [Overworld Architecture](OVERWORLD_ARCHITECTURE.md) - Tick system, factions, influence
2. [Encounter System](ENCOUNTER_SYSTEM.md) - Triggers, setup, resolution
3. [Config Tuning Guide](CONFIG_TUNING_GUIDE.md) - Balancing overworld parameters

**For AI Developers:**
1. [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) - Utility AI, action evaluation
2. [Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md) - Threat painting, danger levels
3. [Caching Overview](CACHING_OVERVIEW.md) - Evaluation caching strategies

**For Data/Config Developers:**
1. [Game Data Overview](GAMEDATA_OVERVIEW.md) - JSON data files, templates
2. [Config Tuning Guide](CONFIG_TUNING_GUIDE.md) - Tunable parameters
3. [Artifact & Inventory System](ARTIFACT_SYSTEM.md) - Item data formats

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

### Package Structure

```
TinkerRogue/
├── common/              # Core ECS utilities, shared components
│   ├── ecsutil.go      # Type-safe component access helpers
│   ├── commoncomponents.go  # Position, Attributes, Name
│   ├── positionsystem.go    # O(1) spatial grid (GlobalPositionSystem)
│   └── playerdata.go   # Player state
│
├── config/              # Centralized game constants
│   └── config.go            # Map dimensions, debug flags, profiling
│
├── world/               # World systems (renamed from coords/)
│   ├── coords/         # Coordinate management (CRITICAL)
│   │   ├── cordmanager.go  # Global CoordManager singleton
│   │   └── position.go     # LogicalPosition, PixelPosition types
│   └── worldmap/       # Procedural generation
│       ├── generator.go         # Generator registry
│       ├── gen_rooms_corridors.go
│       ├── gen_tactical_biome.go
│       └── gen_overworld.go
│
├── tactical/            # Tactical gameplay systems
│   ├── squads/         # Squad system (REFERENCE IMPLEMENTATION)
│   │   ├── squadcomponents.go   # 8 pure data components
│   │   ├── squadqueries.go      # Query functions
│   │   ├── squadcombat.go       # Combat logic
│   │   ├── squadabilities.go    # Leader abilities
│   │   └── squadmanager.go      # Initialization
│   ├── combat/         # Turn-based combat management
│   │   ├── turnmanager.go       # Turn order, round tracking
│   │   ├── combatfactionmanager.go  # Faction system
│   │   └── battlelog/           # Battle recording and export
│   ├── combatservices/ # Combat service layer
│   ├── squadservices/  # Squad service layer
│   ├── squadcommands/  # Squad command pattern (undo/redo)
│   ├── commander/      # Commander entities (NEW)
│   │   ├── components.go       # CommanderData, ActionState, Roster
│   │   ├── movement.go         # Overworld movement
│   │   ├── queries.go          # Commander lookups
│   │   ├── roster.go           # Squad roster management
│   │   ├── turnstate.go        # Overworld turn tracking
│   │   └── system.go           # Commander system logic
│   ├── spells/         # Spell system (NEW)
│   │   ├── components.go       # ManaData, SpellBookData
│   │   ├── system.go           # Spell casting logic
│   │   └── queries.go          # Spell lookups
│   └── effects/        # Active effects system (NEW)
│       ├── components.go       # ActiveEffectsData, StatType
│       ├── system.go           # Effect application/tick-down
│       └── queries.go          # Effect lookups
│
├── mind/                # AI and encounter systems (NEW)
│   ├── ai/             # Utility AI decision-making
│   │   ├── ai_controller.go    # AI turn controller
│   │   └── action_evaluator.go # Action scoring
│   ├── behavior/       # Threat analysis
│   │   ├── threat_layers.go    # Multi-layer threat maps
│   │   ├── threat_painting.go  # Threat projection
│   │   └── dangerlevel.go      # Danger level queries
│   ├── evaluation/     # Power and role evaluation
│   │   ├── power.go            # Squad power scoring
│   │   ├── roles.go            # Role-based multipliers
│   │   └── cache.go            # Evaluation caching
│   └── encounter/      # Encounter system
│       ├── encounter_service.go   # Core service (start/resolve)
│       ├── encounter_trigger.go   # Trigger conditions
│       ├── encounter_setup.go     # Enemy squad generation
│       ├── encounter_resolution.go # Post-combat cleanup
│       ├── encounter_generator.go # Config-driven generation
│       └── rewards.go             # XP and loot
│
├── overworld/           # Strategic overworld layer (NEW)
│   ├── core/           # Shared components and types
│   │   ├── components.go      # All overworld ECS components
│   │   ├── types.go           # FactionType, NodeCategory, enums
│   │   ├── node_registry.go   # Node type definitions (JSON)
│   │   ├── walkability.go     # Overworld walkable grid
│   │   ├── events.go          # Event types for tick results
│   │   └── resources.go       # Global overworld context
│   ├── tick/           # Strategic turn clock
│   │   └── tickmanager.go     # AdvanceTick, tick processing
│   ├── faction/        # NPC faction AI
│   │   ├── system.go          # Faction tick processing
│   │   ├── archetype.go       # Faction behavior archetypes
│   │   └── scoring.go         # Intent scoring functions
│   ├── threat/         # Threat node growth
│   │   ├── system.go          # Threat growth logic
│   │   └── queries.go         # Threat lookups
│   ├── influence/      # Influence radius system
│   │   ├── system.go          # Influence recalculation
│   │   ├── effects.go         # Interaction effects
│   │   └── queries.go         # Influence lookups
│   ├── node/           # Unified node management
│   │   ├── system.go          # Node creation
│   │   ├── queries.go         # Node lookups
│   │   └── validation.go      # Placement validation
│   ├── garrison/       # Garrison defense
│   │   ├── system.go          # Garrison logic, raid detection
│   │   └── queries.go         # Garrison lookups
│   ├── victory/        # Win/loss conditions
│   │   ├── system.go          # Victory checks
│   │   └── queries.go         # Victory state lookups
│   └── overworldlog/   # Event recording
│       ├── overworld_recorder.go  # Event capture
│       ├── overworld_summary.go   # Summary generation
│       └── overworld_export.go    # File export
│
├── gear/                # Inventory and items
│   ├── Inventory.go         # Pure ECS inventory (REFERENCE)
│   ├── items.go             # Item components
│   └── inventory_service.go # Service layer
│
├── gui/                 # User interface
│   ├── framework/      # Core mode infrastructure
│   │   ├── uimode.go          # UIMode interface, UIContext
│   │   ├── basemode.go        # Common mode infrastructure
│   │   ├── modemanager.go     # Mode lifecycle & transitions
│   │   ├── coordinator.go     # GameModeCoordinator (two-context system)
│   │   ├── contextstate.go    # TacticalState, OverworldState
│   │   ├── modebuilder.go     # Declarative mode configuration
│   │   ├── panelregistry.go   # Global panel type registry
│   │   ├── guiqueries.go      # ECS query abstraction
│   │   └── commandhistory.go  # Undo/redo system
│   ├── builders/       # UI construction helpers
│   │   ├── panels.go          # Panel building with functional options
│   │   ├── layout.go          # Layout calculations
│   │   ├── dialogs.go         # Modal dialog builders
│   │   └── panelspecs.go      # Standard panel specifications
│   ├── widgets/        # Widget wrappers & utilities
│   │   ├── cached_list.go     # Cached list (90% CPU reduction)
│   │   ├── cached_textarea.go # Cached text area
│   │   └── textdisplay.go     # Text display widget
│   ├── specs/          # Layout specifications
│   │   └── layout.go          # Responsive layout configuration
│   ├── widgetresources/ # Shared UI resources (renamed from guiresources/)
│   │   ├── guiresources.go     # UI resource loading
│   │   └── cachedbackground.go # Cached background rendering
│   ├── guicombat/      # Combat mode implementation
│   ├── guisquads/      # Squad management modes (editor, purchase, deployment, artifacts)
│   ├── guioverworld/   # Overworld mode (NEW)
│   ├── guinodeplacement/ # Node placement mode (NEW)
│   ├── guiexploration/ # Exploration mode (NEW)
│   ├── guispells/      # Spell casting UI (NEW)
│   ├── guiartifacts/   # Artifact management UI (NEW)
│   └── guistartmenu/   # Start menu (Overworld vs Roguelike) (NEW)
│
├── visual/              # Rendering systems
│   ├── graphics/       # Graphics utilities
│   └── rendering/      # Batch rendering, sprite management
│
├── input/               # Input handling
│   └── cameracontroller.go  # WASD movement, diagonals, map scroll toggle
│
├── templates/           # JSON-based entity creation (renamed)
│   ├── templatelib.go       # Template registry
│   ├── creators.go          # Factory functions
│   └── readdata.go          # JSON loading
│
├── testing/             # Test data and bootstrapping (NEW)
│   ├── testingdata.go       # Test item creation
│   ├── fixtures.go          # Test fixtures
│   └── bootstrap/           # Initial game entity seeding
│       ├── initial_squads.go     # Starting squads
│       ├── initial_commanders.go # Starting commanders
│       ├── initial_factions.go   # Starting factions
│       └── initial_artifacts.go  # Starting artifacts
│
├── tools/               # Development tools
│   ├── combat_simulator/    # Combat simulation suites
│   ├── combat_visualizer/   # Battle visualization
│   └── report_compressor/   # Report compression
│
└── game_main/           # Entry point and initialization
    ├── main.go              # Game struct, loop, start menu
    ├── gameinit.go          # ECS initialization
    ├── componentinit.go     # Component registration
    ├── setup_shared.go      # Shared bootstrap (data, ECS, player)
    ├── setup_overworld.go   # Overworld mode setup
    └── setup_roguelike.go   # Roguelike mode setup
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

**New dependency chains:**
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

---

## Core Systems

### Component Registration Flow

Understanding component initialization is critical. Components must be registered before use or you'll get nil pointer panics.

```go
// 1. Component Declaration (squads/squadcomponents.go)
package squads

var (
    SquadComponent *ecs.Component  // nil until initialized
    SquadTag ecs.Tag               // nil until initialized
)

type SquadData struct {
    SquadID ecs.EntityID
    Name    string
    // ... fields only, no methods
}

// 2. Component Registration (squads/squadmanager.go)
func InitSquadComponents(manager *common.EntityManager) {
    SquadComponent = manager.World.NewComponent()  // Register with ECS
    SquadMemberComponent = manager.World.NewComponent()
    // ... register all components
}

func InitSquadTags(manager *common.EntityManager) {
    SquadTag = ecs.BuildTag(SquadComponent)  // Build tag after component
    manager.WorldTags["squad"] = SquadTag    // Optional: name-based lookup
}

func InitializeSquadData(manager *common.EntityManager) error {
    InitSquadComponents(manager)  // MUST be called first
    InitSquadTags(manager)        // MUST be called second
    // Additional initialization...
    return nil
}

// 3. Game Initialization (game_main/gameinit.go)
func InitializeGame() {
    manager := common.NewEntityManager()

    // Initialize all systems in dependency order
    common.InitCommonComponents(manager)     // Position, Attributes, Name
    squads.InitializeSquadData(manager)      // Squad components
    combat.InitializeCombatComponents(manager)
    gear.InitializeGearComponents(manager)
    // ... more systems
}
```

**Critical Rule**: Never use a component before calling its `Init*Components()` function. This is the #1 source of nil pointer panics.

### EntityManager: The ECS Hub

```go
// common/ecsutil.go
type EntityManager struct {
    World     *ecs.Manager           // Underlying bytearena/ecs
    WorldTags map[string]ecs.Tag     // Named tag registry
}

// Global tags and systems
var (
    AllEntitiesTag ecs.Tag             // Query all entities
    GlobalPositionSystem *systems.PositionSystem  // O(1) spatial queries
)
```

**EntityManager Responsibilities:**

1. **World**: The actual ECS manager from bytearena/ecs library
2. **WorldTags**: String-based tag lookup (e.g., `manager.WorldTags["squad"]`)
3. **Component Access Helpers**: Type-safe wrappers around raw ECS API
4. **Entity Lifecycle**: Create, query, dispose entities

**Usage Pattern:**

```go
// Always pass EntityManager as parameter (never global)
func ProcessSquad(squadID ecs.EntityID, manager *common.EntityManager) {
    // Query using tags
    for _, result := range manager.World.Query(SquadTag) {
        entity := result.Entity
        data := common.GetComponentType[*SquadData](entity, SquadComponent)
        // Process data...
    }
}
```

### Position System: O(1) Spatial Queries

The `GlobalPositionSystem` is one of the most important performance optimizations in the codebase.

```go
// systems/positionsystem.go
type PositionSystem struct {
    manager     *ecs.Manager
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // VALUE keys!
}

// O(1) lookup instead of O(n) search
entityIDs := common.GlobalPositionSystem.GetEntitiesAtPosition(logicalPos)
```

**Before (O(n) - linear search):**
```go
// Had to iterate all entities with position component
for _, result := range manager.World.Query(PositionTag) {
    pos := common.GetPosition(result.Entity)
    if pos.X == targetX && pos.Y == targetY {
        // Found it after checking hundreds of entities!
    }
}
```

**After (O(1) - hash lookup):**
```go
// Direct hash map lookup
entityIDs := common.GlobalPositionSystem.GetEntitiesAtPosition(logicalPos)
if len(entityIDs) > 0 {
    // Found immediately
}
```

**Performance Impact**: 50x faster with 50+ entities, scales linearly vs. quadratically.

**Critical Implementation Detail**: Uses **value-based keys** (`coords.LogicalPosition`) instead of pointer keys (`*coords.LogicalPosition`). Pointer keys require creating temporary pointers for lookups which is 50x slower.

**Entity Lifecycle Integration:**

```go
// When creating entity with position
entity := manager.World.CreateEntity()
entity.AddComponent(common.PositionComponent, logicalPos)
common.GlobalPositionSystem.AddEntity(entity.GetID(), *logicalPos)

// When moving entity
common.GlobalPositionSystem.MoveEntity(entityID, oldPos, newPos)

// When destroying entity
common.GlobalPositionSystem.RemoveEntity(entityID, logicalPos)
manager.World.DisposeEntities(entity)
```

### Coordinate Manager: Unified Conversion

The `coords.CoordManager` is a global singleton that handles all coordinate conversions. This prevents index calculation bugs that were common in earlier versions.

```go
// coords/cordmanager.go
var CoordManager *CoordinateManager  // Global instance

type CoordinateManager struct {
    dungeonWidth  int  // 100
    dungeonHeight int  // 80
    tileSize      int  // 32 pixels
    scaleFactor   int  // 3x
    // ... more fields
}
```

**Coordinate Spaces:**

1. **Logical Coordinates**: Grid positions (0-99, 0-79) - game logic uses this
2. **Index**: Flat array index for tile storage
3. **Pixel Coordinates**: Rendering positions before scaling
4. **Screen Coordinates**: Final on-screen positions after scaling/viewport

**Conversion Functions:**

```go
// Logical <-> Index (CRITICAL FOR TILE ARRAYS)
tileIdx := coords.CoordManager.LogicalToIndex(logicalPos)  // ALWAYS USE THIS
logicalPos := coords.CoordManager.IndexToLogical(tileIdx)

// Logical <-> Pixel (for rendering)
pixelPos := coords.CoordManager.LogicalToPixel(logicalPos)
logicalPos := coords.CoordManager.PixelToLogical(pixelPos)

// Validation
if coords.CoordManager.IsValidLogical(pos) {
    // Position is within dungeon bounds
}
```

**Critical Warning**: **ALWAYS** use `CoordManager.LogicalToIndex()` when accessing tile arrays. Manual calculation (`y*width + x`) causes index out of bounds panics because the width might not match `dungeonWidth`.

```go
// ✅ CORRECT
tileIdx := coords.CoordManager.LogicalToIndex(logicalPos)
result.Tiles[tileIdx] = &tile

// ❌ WRONG - CAUSES PANICS
idx := y*width + x  // width might differ from CoordManager.dungeonWidth!
result.Tiles[idx] = &tile
```

---

> **Entity Component System (ECS)** has been moved to its own document. See [`ECS_BEST_PRACTICES.md`](ECS_BEST_PRACTICES.md) for full details on core ECS principles, component access patterns, entity lifecycle, and file organization.

---

> **Squad & Combat System** has been moved to its own document. See [`SQUAD_COMBAT_SYSTEM.md`](SQUAD_COMBAT_SYSTEM.md) for full details on squad formation, combat turn management, faction system, AI, and combat services.

---

> **GUI Architecture** has been moved to its own document. See [`docs/gui_documentation/GUI_DOCUMENTATION.md`](../gui_documentation/GUI_DOCUMENTATION.md) for full details.

> **Inventory & Gear** has been moved to its own document. See [`ARTIFACT_SYSTEM.md`](ARTIFACT_SYSTEM.md) for full details.

---

> **World Generation** has been moved to its own document. See [`WORLD_GENERATION.md`](WORLD_GENERATION.md) for full details on generator architecture, registry pattern, tile system, biome system, and A* pathfinding.

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

## Development Patterns

### Adding a New Component

```go
// 1. Declare in package components.go
package mysystem

var (
    MyComponent *ecs.Component
    MyTag ecs.Tag
)

type MyData struct {
    // ONLY data fields, NO methods
    SomeField string
    Value     int
}

// 2. Create initialization function
func InitMyComponents(manager *common.EntityManager) {
    MyComponent = manager.World.NewComponent()
}

func InitMyTags(manager *common.EntityManager) {
    MyTag = ecs.BuildTag(MyComponent)
    manager.WorldTags["mytag"] = MyTag
}

// 3. Call from game initialization
// game_main/gameinit.go
func InitializeGame() {
    manager := common.NewEntityManager()
    // ... other initializations
    mysystem.InitMyComponents(manager)
    mysystem.InitMyTags(manager)
}

// 4. Use component in entities
entity := manager.World.CreateEntity()
entity.AddComponent(mysystem.MyComponent, &mysystem.MyData{
    SomeField: "test",
    Value:     42,
})
entity.AddTag(mysystem.MyTag)
```

### Adding a New System Function

```go
// mysystem/mysystem.go

// System function that modifies state
func ProcessMyData(entityID ecs.EntityID, manager *common.EntityManager) error {
    // Get component
    myData := common.GetComponentTypeByIDWithTag[*MyData](
        manager, entityID, MyTag, MyComponent)
    if myData == nil {
        return fmt.Errorf("entity does not have MyComponent")
    }

    // Modify component
    myData.Value += 10

    return nil
}

// Query function that reads state
func GetEntitiesWithValue(minValue int, manager *common.EntityManager) []ecs.EntityID {
    var results []ecs.EntityID
    for _, result := range manager.World.Query(MyTag) {
        myData := common.GetComponentType[*MyData](result.Entity, MyComponent)
        if myData.Value >= minValue {
            results = append(results, result.Entity.GetID())
        }
    }
    return results
}
```

### Adding a New UI Mode

> See [`docs/gui_documentation/GUI_DOCUMENTATION.md`](../gui_documentation/GUI_DOCUMENTATION.md) for the full guide on adding UI modes.

### Adding a World Generator

```go
// worldmap/gen_myalgorithm.go

type MyGenerator struct {
    config GeneratorConfig
}

func init() {
    RegisterGenerator(&MyGenerator{
        config: DefaultConfig(),
    })
}

func (g *MyGenerator) Name() string {
    return "my_algorithm"
}

func (g *MyGenerator) Description() string {
    return "My custom map generation algorithm"
}

func (g *MyGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
    // Initialize tiles
    tiles := make([]*Tile, width*height)
    for i := range tiles {
        logicalPos := coords.CoordManager.IndexToLogical(i)
        tiles[i] = &Tile{
            TileType:   TileWall,
            LogicalPos: logicalPos,
            Image:      images.Wall,
            IsWalkable: false,
        }
    }

    // Generate rooms
    rooms := []Rect{}
    // ... algorithm logic

    // Carve floors
    for _, room := range rooms {
        for y := room.Y; y < room.Y+room.Height; y++ {
            for x := room.X; x < room.X+room.Width; x++ {
                idx := coords.CoordManager.LogicalToIndex(coords.LogicalPosition{X: x, Y: y})
                tiles[idx].TileType = TileFloor
                tiles[idx].Image = images.Floor
                tiles[idx].IsWalkable = true
            }
        }
    }

    // Collect valid positions
    validPositions := []coords.LogicalPosition{}
    for _, tile := range tiles {
        if tile.IsWalkable {
            validPositions = append(validPositions, tile.LogicalPos)
        }
    }

    return GenerationResult{
        Tiles:          tiles,
        Rooms:          rooms,
        ValidPositions: validPositions,
    }
}
```

**Critical**: Generators MUST register in `init()` or they won't be discovered.

---

## Performance Considerations

### 2026 Performance Enhancements

TinkerRogue has undergone significant performance optimization, particularly in rendering and UI systems.

> **GUI Performance:** See [`docs/gui_documentation/GUI_DOCUMENTATION.md`](../gui_documentation/GUI_DOCUMENTATION.md) for GUI-specific performance optimizations (90% CPU reduction via cached widgets).

**Rendering Performance:**

1. **Batched Tile Rendering** - Draw all tiles in single batch call
2. **Batched Sprite Rendering** - Batch entity sprites together
3. **Static Panel Backgrounds** - Pre-render panel backgrounds to reduce nineslice overhead
4. **Cached Background Images** - Reuse background textures
5. **DrawImageOptions Reuse** - Avoid allocating new options every frame

**Before:**
```go
// Created new image every frame (SLOW)
for _, tile := range tiles {
    overlay := ebiten.NewImage(tileSize, tileSize)
    overlay.Fill(color)
    screen.DrawImage(overlay, opts)
}
```

**After:**
```go
// Reuse single overlay image (FAST)
overlay := ebiten.NewImage(tileSize, tileSize)
for _, tile := range tiles {
    overlay.Fill(color)
    screen.DrawImage(overlay, opts)
    overlay.Clear()
}
```

### Component Access Performance

**Function Selection**:
- `GetComponentTypeByIDWithTag` (10-100x faster for known tags)
- `GetComponentTypeByID` (searches all entities)
- `GetComponentType` (fastest, when entity already available)

```go
// SLOW - Searches all 1000+ entities
data := common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)

// FAST - Searches only 10-50 squad entities
data := common.GetComponentTypeByIDWithTag[*SquadData](
    manager, squadID, SquadTag, SquadComponent)
```

### Spatial Query Performance

**Before O(n)**:
```go
// Searched every entity with position
for _, result := range manager.World.Query(PositionTag) {
    pos := common.GetPosition(result.Entity)
    if pos.X == targetX && pos.Y == targetY {
        // Found after checking 500+ entities
    }
}
```

**After O(1)**:
```go
// Direct hash map lookup
entityIDs := common.GlobalPositionSystem.GetEntitiesAtPosition(logicalPos)
// Returns immediately
```

**Impact**: 50x faster with 50+ entities, scales linearly.

### Map Key Performance

**ALWAYS use value types as map keys**:

```go
// SLOW - Pointer keys require temporary pointer creation
grid map[*coords.LogicalPosition][]ecs.EntityID

// To query:
tempPos := &coords.LogicalPosition{X: 10, Y: 20}  // Allocation!
entities := grid[tempPos]  // Won't work - different pointer

// FAST - Value keys use struct equality
grid map[coords.LogicalPosition][]ecs.EntityID

// To query:
entities := grid[coords.LogicalPosition{X: 10, Y: 20}]  // Works!
```

**Measured**: 50x performance difference in PositionSystem refactor.

### Query Optimization

**When to Cache**:
- Tight inner loops (Update/Render every frame)
- Large entity sets (1000+)
- Profile first to confirm bottleneck

**When NOT to Cache**:
- One-time queries
- Small entity sets (<100)
- Infrequent operations

**Example of Justified Caching**:
```go
// Combat resolution queries hundreds of units per attack
// Profile showed 30% of frame time in queries
// Solution: Cache unit positions at start of combat resolution

type CombatCache struct {
    unitPositions map[ecs.EntityID]coords.LogicalPosition
}

func buildCombatCache(squadID ecs.EntityID, manager *common.EntityManager) CombatCache {
    cache := CombatCache{unitPositions: make(map[ecs.EntityID]coords.LogicalPosition)}

    unitIDs := GetUnitIDsInSquad(squadID, manager)
    for _, unitID := range unitIDs {
        pos := common.GetPositionByIDWithTag(manager, unitID, SquadMemberTag)
        cache.unitPositions[unitID] = *pos
    }

    return cache
}

// Now use cache instead of repeated queries
func resolveCombat(cache CombatCache) {
    for unitID := range cache.unitPositions {
        pos := cache.unitPositions[unitID]  // O(1) map lookup
        // ... combat logic
    }
}
```

### Entity Lifecycle Optimization

**Batch Entity Creation**:
```go
// Instead of creating entities one at a time in a loop:
for i := 0; i < 100; i++ {
    entity := manager.World.CreateEntity()
    entity.AddComponent(/* ... */)
    entity.AddTag(/* ... */)
}

// Create all entities first, then add components:
entities := make([]*ecs.Entity, 100)
for i := 0; i < 100; i++ {
    entities[i] = manager.World.CreateEntity()
}

for i, entity := range entities {
    entity.AddComponent(/* ... */)
    entity.AddTag(/* ... */)
}
```

**Batch Disposal**:
```go
// Collect entities to dispose
toDispose := []*ecs.Entity{}
for _, result := range manager.World.Query(DeadTag) {
    toDispose = append(toDispose, result.Entity)
}

// Dispose all at once
manager.World.DisposeEntities(toDispose...)
```

---

**Document Version**: 3.4 (2026 Edition)
**Last Updated**: 2026-02-18
**Total Sections**: 6
**Major Changes**: Removed Coordinate System (merged into Core Systems), Component Catalog, Input System, and Appendices (A-F) to focus on core architecture and patterns. See external documentation links for detailed subsystem docs.
