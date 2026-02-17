# TinkerRogue Technical Documentation

**Version:** 3.0 (2026 Edition)
**Last Updated:** 2026-01-11
**Project Type:** Turn-based tactical roguelike with squad combat
**Language:** Go 1.x
**ECS Library:** bytearena/ecs
**UI Framework:** Ebitengine v2 + ebitenui

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Core Systems](#core-systems)
4. [Component Catalog](#component-catalog)
5. [Coordinate System](#coordinate-system)
6. [Entity Component System (ECS)](#entity-component-system-ecs)
7. [Squad System](#squad-system)
8. [Combat System](#combat-system)
9. [World Generation](#world-generation)
10. [Input System](#input-system)
11. [Data Flow Patterns](#data-flow-patterns)
12. [Development Patterns](#development-patterns)
13. [Performance Considerations](#performance-considerations)
14. [Appendices](#appendices)

**External Documentation:**
- [GUI Architecture](docs/gui_documentation/GUI_DOCUMENTATION.md)
- [Overworld System](docs/project_documentation/OVERWORLD_ARCHITECTURE.md)
- [Artifact & Inventory System](docs/project_documentation/ARTIFACT_SYSTEM.md)

---

## Executive Summary

TinkerRogue is a turn-based tactical roguelike implemented in Go using a pure Entity Component System (ECS) architecture. The game features squad-based tactical combat where players control formations of units in a 3x3 grid layout, engaging enemy squads on procedurally generated maps.

### Key Architectural Decisions

1. **Pure ECS Architecture**: All game logic follows strict ECS patterns with zero logic in components, EntityID-only relationships, and query-based data access.

2. **Global Coordinate Manager**: A single global `coords.CoordManager` instance handles all coordinate conversions, preventing index-out-of-bounds errors that plagued earlier versions.

3. **O(1) Spatial Queries**: The `GlobalPositionSystem` provides constant-time position lookups using value-based map keys, replacing O(n) linear searches.

4. **Mode-Based GUI**: UI uses a mode manager pattern with context isolation between different game states (overworld, combat, inventory, squad builder).

5. **Template-Based Entity Creation**: Entity spawning uses JSON templates loaded from files, separating data from code.

### Technical Metrics

- **Lines of Code**: ~20,000+ across 120+ files
- **Packages**: 15 major systems (common, coords, tactical, gui, gear, worldmap, visual, etc.)
- **Components**: 30+ registered ECS components
- **Entity Types**: Units, Squads, Items, Tiles, Factions, Combat State
- **Performance**:
  - Value-based map keys provide 50x improvement over pointer-based keys
  - Cached list widgets reduce CPU usage by ~90%
  - Batched rendering for tiles and sprites
  - O(1) spatial queries via GlobalPositionSystem

### Reading Paths

**For New Developers:**
1. Start with [Architecture Overview](#architecture-overview)
2. Read [Entity Component System](#entity-component-system-ecs)
3. Study [Squad System](#squad-system) as reference implementation
4. Review `docs/ecs_best_practices.md` for detailed patterns

**For Gameplay Programmers:**
1. [Squad System](#squad-system) - Formation-based combat
2. [Combat System](#combat-system) - Turn management and resolution
3. [Artifact & Inventory System](docs/project_documentation/ARTIFACT_SYSTEM.md) - Item handling

**For UI Developers:**
1. [GUI Architecture](docs/gui_documentation/GUI_DOCUMENTATION.md) - Mode management
2. [Input System](#input-system) - Controller pattern
3. [Data Flow Patterns](#data-flow-patterns) - UI to ECS integration

**For Systems Programmers:**
1. [Coordinate System](#coordinate-system) - Global manager pattern
2. [Performance Considerations](#performance-considerations) - Optimization strategies
3. [Entity Component System](#entity-component-system-ecs) - Component access helpers

---

## Architecture Overview

### System Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Game Main Loop                            │
│  (game_main/main.go, gameinit.go)                               │
└─────────────┬───────────────────────────────────────────────────┘
              │
              ├─ Global Systems (Initialized Once)
              │  ├─ coords.CoordManager (coordinate conversions)
              │  ├─ common.GlobalPositionSystem (O(1) spatial queries)
              │  └─ common.EntityManager (ECS world + tag registry)
              │
              ├─ Core Systems (Per-Entity State)
              │  ├─ squads/ (formation management, abilities)
              │  ├─ combat/ (turn management, damage resolution)
              │  ├─ gear/ (inventory, items, equipment)
              │  └─ worldmap/ (procedural generation)
              │
              ├─ GUI Layer (Mode-Based)
              │  ├─ gui/core/ (UIModeManager, context switching)
              │  ├─ gui/guimodes/ (OverworldMode, CombatMode, etc.)
              │  ├─ gui/widgets/ (reusable UI components)
              │  └─ gui/guicombat/ (combat-specific UI)
              │
              └─ Input Layer (Controller Pattern)
                 ├─ input/inputcoordinator.go (dispatches to controllers)
                 ├─ input/movementcontroller.go (player movement)
                 ├─ input/combatcontroller.go (combat actions)
                 └─ input/uicontroller.go (UI interactions)
```

### Package Structure

```
TinkerRogue/
├── common/              # Core ECS utilities, shared components
│   ├── ecsutil.go      # Type-safe component access helpers
│   ├── commoncomponents.go  # Position, Attributes, Name
│   └── playerdata.go   # Player state
│
├── world/               # World systems (renamed from coords/)
│   ├── coords/         # Coordinate management (CRITICAL)
│   │   ├── cordmanager.go  # Global CoordManager singleton
│   │   └── position.go     # LogicalPosition, PixelPosition types
│   ├── systems/        # ECS systems (position tracking, etc.)
│   │   └── positionsystem.go  # O(1) spatial grid (GlobalPositionSystem)
│   └── worldmap/       # Procedural generation
│       ├── generator.go         # Generator registry
│       ├── gen_rooms_corridors.go
│       ├── gen_tactical_biome.go
│       └── gen_overworld.go
│
├── tactical/            # Tactical gameplay systems (NEW)
│   ├── squads/         # Squad system (REFERENCE IMPLEMENTATION)
│   │   ├── squadcomponents.go   # 8 pure data components
│   │   ├── squadqueries.go      # Query functions
│   │   ├── squadcombat.go       # Combat logic
│   │   ├── squadabilities.go    # Leader abilities
│   │   └── squadmanager.go      # Initialization
│   ├── combat/         # Turn-based combat management
│   │   ├── turnmanager.go       # Turn order, round tracking
│   │   └── gameplayfactions.go  # Faction system
│   ├── combatservices/ # Combat service layer
│   ├── squadservices/  # Squad service layer
│   ├── squadcommands/  # Squad command pattern (undo/redo)
│   ├── ai/             # AI decision-making
│   └── behavior/       # Behavior trees and utility AI
│
├── gear/                # Inventory and items
│   ├── Inventory.go         # Pure ECS inventory (REFERENCE)
│   ├── items.go             # Item components
│   └── inventory_service.go # Service layer
│
├── gui/                 # User interface (MAJOR OVERHAUL)
│   ├── framework/      # Core mode infrastructure (NEW)
│   │   ├── uimode.go          # UIMode interface, UIContext
│   │   ├── basemode.go        # Common mode infrastructure
│   │   ├── modemanager.go     # Mode lifecycle & transitions
│   │   ├── coordinator.go     # Two-context system (Overworld/BattleMap)
│   │   ├── modebuilder.go     # Declarative mode configuration
│   │   ├── panelregistry.go   # Global panel type registry
│   │   ├── guiqueries.go      # ECS query abstraction
│   │   └── commandhistory.go  # Undo/redo system
│   ├── builders/       # UI construction helpers (NEW)
│   │   ├── panels.go          # Panel building with functional options
│   │   ├── layout.go          # Layout calculations
│   │   ├── dialogs.go         # Modal dialog builders
│   │   └── panelspecs.go      # Standard panel specifications
│   ├── widgets/        # Widget wrappers & utilities (NEW)
│   │   ├── cached_list.go     # Cached list (90% CPU reduction)
│   │   ├── cached_textarea.go # Cached text area
│   │   └── createwidgets.go   # Widget creation helpers
│   ├── specs/          # Layout specifications (NEW)
│   │   └── layout.go          # Responsive layout configuration
│   ├── guiresources/   # Shared UI resources
│   │   └── cachedbackground.go # Cached background rendering
│   ├── guicombat/      # Combat mode implementation
│   ├── guisquads/      # Squad management modes
│   └── guimodes/       # Other game modes (exploration, inventory)
│
├── visual/              # Rendering systems (NEW)
│   ├── graphics/       # Graphics utilities
│   └── rendering/      # Batch rendering, sprite management
│
├── input/               # Input handling
│   ├── inputcoordinator.go  # Central dispatcher
│   ├── movementcontroller.go
│   ├── combatcontroller.go
│   └── uicontroller.go
│
├── templates/           # JSON-based entity creation (renamed)
│   ├── templatelib.go       # Template registry
│   ├── creators.go          # Factory functions
│   └── readdata.go          # JSON loading
│
├── tools/               # Development tools (NEW)
│   └── combatsim/      # Combat simulation and analysis
│
└── game_main/           # Entry point and initialization
    ├── main.go              # Game loop
    ├── gameinit.go          # System initialization
    └── componentinit.go     # Component registration
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
gui (presentation layer) → Reads from ECS, writes commands
    ↓
input (user interaction) → Modifies ECS state via controllers
```

**Key Principle**: Dependencies flow downward. GUI and input layers are at the top and depend on lower layers but not vice versa. Game logic systems (squads, combat, gear) are independent and communicate through ECS.

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

## Entity Component System (ECS)

TinkerRogue uses the bytearena/ecs library with strict architectural patterns documented in `docs/ecs_best_practices.md`.

### Core ECS Principles

#### 1. Pure Data Components

Components contain **zero logic** - only data fields.

```go
// ✅ CORRECT - Pure data
type SquadData struct {
    SquadID       ecs.EntityID
    Formation     FormationType
    Name          string
    Morale        int
}

// ❌ WRONG - Has methods
type SquadData struct {
    // ... fields ...
}
func (s *SquadData) GetMorale() int { return s.Morale }  // NO!
```

#### 2. EntityID-Only Relationships

Store `ecs.EntityID` for relationships, never entity pointers.

```go
// ✅ CORRECT
type SquadMemberData struct {
    SquadID ecs.EntityID  // Safe, stable reference
}

// ❌ WRONG
type SquadMemberData struct {
    Squad *ecs.Entity  // Can become invalid!
}
```

**Why**: Entity pointers can become dangling when entities are disposed. EntityIDs are stable and safe.

#### 3. Query-Based Relationships

Discover relationships through queries instead of caching them.

```go
// ✅ CORRECT - Query when needed
func GetUnitsInSquad(squadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID
    for _, result := range manager.World.Query(SquadMemberTag) {
        memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
        if memberData.SquadID == squadID {
            unitIDs = append(unitIDs, result.Entity.GetID())
        }
    }
    return unitIDs
}

// ❌ WRONG - Cached relationship
type SquadData struct {
    UnitIDs []ecs.EntityID  // Requires manual sync!
}
```

**Why**: Cached relationships require manual synchronization. Queries are always up-to-date.

#### 4. System-Based Logic

All behavior belongs in system functions, not component methods.

```go
// ✅ CORRECT - System function
func ExecuteSquadAttack(attackerID, defenderID ecs.EntityID, manager *common.EntityManager) *CombatResult {
    // Logic here
}

// ❌ WRONG - Logic in component
func (s *SquadData) Attack(target *SquadData) {
    // NO! Put this in a system function
}
```

#### 5. Value-Based Map Keys

Use value types as map keys for O(1) performance.

```go
// ✅ CORRECT - Value key (fast)
spatialGrid map[coords.LogicalPosition][]ecs.EntityID

// ❌ WRONG - Pointer key (50x slower)
spatialGrid map[*coords.LogicalPosition][]ecs.EntityID
```

### Component Access Patterns

TinkerRogue provides type-safe helper functions in `common/ecsutil.go`.

#### GetComponentType - From Entity Pointer

Use when you already have an entity from a query:

```go
for _, result := range manager.World.Query(SquadTag) {
    entity := result.Entity
    squadData := common.GetComponentType[*SquadData](entity, SquadComponent)
    // Use squadData...
}
```

#### GetComponentTypeByID - From EntityID

Use when you only have an EntityID:

```go
func ProcessUnit(unitID ecs.EntityID, manager *common.EntityManager) {
    attributes := common.GetComponentTypeByID[*Attributes](manager, unitID, AttributeComponent)
    if attributes == nil {
        return  // Component not found
    }
    // Use attributes...
}
```

#### GetComponentTypeByIDWithTag - Optimized Query

Use when you know which tag the entity belongs to (10-100x faster):

```go
// Searches only entities with SquadTag (typically 10-50 entities)
squadData := common.GetComponentTypeByIDWithTag[*SquadData](
    manager, squadID, SquadTag, SquadComponent)

// Instead of searching AllEntitiesTag (potentially 1000+ entities)
```

### Entity Lifecycle

```go
// 1. Create Entity
entity := manager.World.CreateEntity()
entityID := entity.GetID()  // Or use manager.NextID() for custom IDs

// 2. Add Components
entity.AddComponent(SquadComponent, &SquadData{
    SquadID: entityID,
    Name:    "Alpha Squad",
})

// 3. Add Tags
entity.AddTag(SquadTag)

// 4. Register with Spatial System (if has position)
if hasPosition {
    common.GlobalPositionSystem.AddEntity(entityID, logicalPos)
}

// 5. Destroy Entity (proper cleanup)
if entity.HasComponent(common.PositionComponent) {
    pos := common.GetPosition(entity)
    common.GlobalPositionSystem.RemoveEntity(entityID, *pos)
}
manager.World.DisposeEntities(entity)
```

### File Organization

Each ECS package follows this structure:

```
squads/
├── squadcomponents.go   # Component data definitions ONLY
├── squadqueries.go      # Query functions (read-only)
├── squadcombat.go       # Combat system logic
├── squadabilities.go    # Ability system logic
└── squadmanager.go      # Initialization
```

**File Responsibilities:**

- `components.go`: Data structs, component variables, tags
- `*queries.go`: Functions that search/filter entities
- `*system.go`: Functions that modify components
- `*manager.go`: Initialization (InitComponents, InitTags)

---

## Squad System

The squad system (`tactical/squads/`) is the **reference implementation** for proper ECS architecture in TinkerRogue. Study this system to understand how to structure new features.

> **Package Location**: Squads moved to `tactical/squads/` as part of 2026 reorganization.

### Overview

Squads are formations of units arranged in a 3x3 grid. Each squad is an entity, and each unit in the squad is also an entity with a `SquadMemberData` component linking it to its parent squad.

```
Squad Entity (SquadData)
    ├─ Unit Entity (SquadMemberData, GridPositionData, UnitRoleData)
    ├─ Unit Entity (SquadMemberData, GridPositionData, UnitRoleData)
    └─ Unit Entity (Leader) (SquadMemberData, LeaderData, AbilitySlotData)
```

### Component Architecture

**8 Components** (all in `squadcomponents.go`):

1. **SquadData**: Squad-level properties (name, formation, morale)
2. **SquadMemberData**: Links unit to parent squad
3. **GridPositionData**: Unit's position in 3x3 grid
4. **UnitRoleData**: Combat role (Tank, DPS, Support)
5. **CoverData**: Defensive cover mechanics
6. **LeaderData**: Leader bonuses
7. **AbilitySlotData**: 4 ability slots (FFT-style)
8. **TargetRowData**: Which rows/cells to attack

**Tags**:
- `SquadTag`: Entities with SquadData
- `SquadMemberTag`: Entities with SquadMemberData
- `LeaderTag`: Multi-component tag (LeaderComponent + SquadMemberComponent)

### Squad Component Example

```go
// squadcomponents.go:40-50
type SquadData struct {
    SquadID       ecs.EntityID  // Native entity ID (not pointer!)
    Formation     FormationType
    Name          string
    Morale        int           // 0-100
    SquadLevel    int
    TurnCount     int
    MaxUnits      int           // Typically 9 (3x3 grid)
    UsedCapacity  float64
    TotalCapacity int
}
```

**Note**: Zero methods on the data struct. All logic is in system functions.

### Query Functions (squadqueries.go)

```go
// Get squad entity by ID
func GetSquadEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
    for _, result := range manager.World.Query(SquadTag) {
        squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
        if squadData.SquadID == squadID {
            return result.Entity
        }
    }
    return nil
}

// Get all units in a squad (query-based, not cached)
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID
    for _, result := range manager.World.Query(SquadMemberTag) {
        memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
        if memberData.SquadID == squadID {
            unitIDs = append(unitIDs, result.Entity.GetID())
        }
    }
    return unitIDs
}
```

### System Functions (squadcombat.go)

```go
// squadcombat.go:15-50
func ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID,
                       manager *common.EntityManager) *CombatResult {
    // 1. Get squad data
    attackerData := common.GetComponentTypeByIDWithTag[*SquadData](
        manager, attackerSquadID, SquadTag, SquadComponent)
    defenderData := common.GetComponentTypeByIDWithTag[*SquadData](
        manager, defenderSquadID, SquadTag, SquadComponent)

    // 2. Get units
    attackerUnits := GetUnitIDsInSquad(attackerSquadID, manager)
    defenderUnits := GetUnitIDsInSquad(defenderSquadID, manager)

    // 3. Execute combat logic
    result := &CombatResult{}
    for _, attackerID := range attackerUnits {
        // Get attacker stats, find targets, calculate damage...
    }

    return result
}
```

### Grid Position System

Units occupy cells in a 3x3 grid (row 0-2, col 0-2). Multi-cell units are supported.

```go
// squadcomponents.go:88-112
type GridPositionData struct {
    AnchorRow int  // Top-left row (0-2)
    AnchorCol int  // Top-left col (0-2)
    Width     int  // Number of columns (1-3)
    Height    int  // Number of rows (1-3)
}

// Get all cells this unit occupies
func (g *GridPositionData) GetOccupiedCells() [][2]int {
    var cells [][2]int
    for r := g.AnchorRow; r < g.AnchorRow+g.Height && r < 3; r++ {
        for c := g.AnchorCol; c < g.AnchorCol+g.Width && c < 3; c++ {
            cells = append(cells, [2]int{r, c})
        }
    }
    return cells
}
```

**Formation Layout Example:**
```
Front Row (0):  [Tank]  [Tank]  [DPS]
Middle Row (1): [DPS]   [Leader][DPS]
Back Row (2):   [Healer][Ranged][Ranged]
```

### Leader Abilities

Leaders have 4 ability slots (Final Fantasy Tactics style).

```go
// squadcomponents.go:230-240
type AbilitySlotData struct {
    Slots [4]AbilitySlot  // Can't have multiple components of same type
}

type AbilitySlot struct {
    AbilityType  AbilityType  // Rally, Heal, BattleCry, Fireball
    TriggerType  TriggerType  // When to activate
    Threshold    float64      // Condition threshold
    HasTriggered bool         // Once-per-combat flag
    IsEquipped   bool
}
```

**Ability Types**:
- `AbilityRally`: +5 Strength for 3 turns
- `AbilityHeal`: +10 HP healing
- `AbilityBattleCry`: +3 Strength, +10 Morale (once per combat)
- `AbilityFireball`: 15 direct damage

**Trigger Types**:
- `TriggerCombatStart`: First turn of combat
- `TriggerSquadHPBelow`: Squad average HP < threshold
- `TriggerTurnCount`: Specific turn number
- `TriggerMoraleBelow`: Squad morale < threshold

**Ability Execution** (`squadabilities.go:20-80`):

```go
func CheckAndTriggerAbilities(squadID ecs.EntityID, manager *common.EntityManager) {
    leaderID := GetSquadLeader(squadID, manager)
    if leaderID == 0 {
        return  // No leader
    }

    abilityData := common.GetComponentTypeByIDWithTag[*AbilitySlotData](
        manager, leaderID, LeaderTag, AbilitySlotComponent)

    for i, slot := range abilityData.Slots {
        if !slot.IsEquipped || slot.HasTriggered {
            continue
        }

        if shouldTrigger(slot, squadID, manager) {
            executeAbility(slot.AbilityType, squadID, manager)
            abilityData.Slots[i].HasTriggered = true
        }
    }
}
```

### Initialization

```go
// squadmanager.go:15-52
func InitSquadComponents(manager *common.EntityManager) {
    SquadComponent = manager.World.NewComponent()
    SquadMemberComponent = manager.World.NewComponent()
    GridPositionComponent = manager.World.NewComponent()
    UnitRoleComponent = manager.World.NewComponent()
    CoverComponent = manager.World.NewComponent()
    LeaderComponent = manager.World.NewComponent()
    TargetRowComponent = manager.World.NewComponent()
    AbilitySlotComponent = manager.World.NewComponent()
}

func InitSquadTags(manager *common.EntityManager) {
    SquadTag = ecs.BuildTag(SquadComponent)
    SquadMemberTag = ecs.BuildTag(SquadMemberComponent)
    LeaderTag = ecs.BuildTag(LeaderComponent, SquadMemberComponent)  // Multi-component

    manager.WorldTags["squad"] = SquadTag
    manager.WorldTags["squadmember"] = SquadMemberTag
    manager.WorldTags["leader"] = LeaderTag
}

func InitializeSquadData(manager *common.EntityManager) error {
    InitSquadComponents(manager)
    InitSquadTags(manager)
    return nil
}
```

**Critical**: Call `InitializeSquadData()` during game initialization before creating any squad entities.

---

## Combat System

The combat system (`tactical/combat/`, `tactical/combatservices/`) manages turn-based tactical encounters between factions.

> **Package Location**: Combat moved to `tactical/combat/` as part of 2026 reorganization.

### Recent Enhancements (2026)

1. **Multiple Faction Support** - Supports hot-seat multiplayer with multiple player-controlled factions
2. **Threat Map System** - AI uses threat maps for tactical decision-making (`tactical/ai/`)
3. **Combat Visualization** - File-based combat recording for testing and analysis (`tools/combatsim/`)
4. **Movement System** - Squads can move multiple tiles over multiple actions based on movement speed
5. **Combat Animation** - Visual feedback for attacks, movements, and abilities

### Architecture

```
Combat Flow:
1. InitializeCombat() - Setup turn order, create action states
2. ResetSquadActions() - Reset movement/action flags for active faction
3. ProcessActions() - Player/AI selects actions
4. ExecuteAttacks() - Resolve combat via squad system
5. CheckVictory() - Determine if combat ends
6. EndTurn() - Advance to next faction
7. Repeat steps 2-6 until victory
```

### Turn Manager

```go
// combat/turnmanager.go:11-18
type TurnManager struct {
    manager *common.EntityManager
}

func NewTurnManager(manager *common.EntityManager) *TurnManager {
    return &TurnManager{manager: manager}
}
```

### Combat State Components

```go
// combat/components.go (conceptual - actual location may vary)
type TurnStateData struct {
    CurrentRound     int
    TurnOrder        []ecs.EntityID  // Faction IDs
    CurrentTurnIndex int
    CombatActive     bool
}

type ActionStateData struct {
    SquadID           ecs.EntityID
    HasMoved          bool
    HasActed          bool
    MovementRemaining int
}
```

### Combat Initialization

```go
// turnmanager.go:21-50
func (tm *TurnManager) InitializeCombat(factionIDs []ecs.EntityID) error {
    // Randomize turn order
    turnOrder := make([]ecs.EntityID, len(factionIDs))
    copy(turnOrder, factionIDs)
    shuffleFactionOrder(turnOrder)

    // Create turn state entity
    turnEntity := tm.manager.World.NewEntity()
    turnEntity.AddComponent(TurnStateComponent, &TurnStateData{
        CurrentRound:     1,
        TurnOrder:        turnOrder,
        CurrentTurnIndex: 0,
        CombatActive:     true,
    })

    // Create action states for all squads
    for _, factionID := range factionIDs {
        factionSquads := GetSquadsForFaction(factionID, tm.manager)
        for _, squadID := range factionSquads {
            tm.createActionStateForSquad(squadID)
            squads.CheckAndTriggerAbilities(squadID, tm.manager)  // Combat-start abilities
        }
    }

    // Reset actions for first faction
    tm.ResetSquadActions(turnOrder[0])
    return nil
}
```

### Turn Advancement

```go
// turnmanager.go:109-132
func (tm *TurnManager) EndTurn() error {
    turnEntity := findTurnStateEntity(tm.manager)
    if turnEntity == nil {
        return fmt.Errorf("no active combat")
    }

    turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)
    turnState.CurrentTurnIndex++

    // Wraparound to new round
    if turnState.CurrentTurnIndex >= len(turnState.TurnOrder) {
        turnState.CurrentTurnIndex = 0
        turnState.CurrentRound++
    }

    // Reset action states for new faction
    newFactionID := turnState.TurnOrder[turnState.CurrentTurnIndex]
    return tm.ResetSquadActions(newFactionID)
}
```

### Faction System

Factions group squads together (player faction, enemy factions).

```go
// combat/gameplayfactions.go (conceptual)
func GetSquadsForFaction(factionID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    var squadIDs []ecs.EntityID
    for _, result := range manager.World.Query(SquadTag) {
        squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
        // Check if squad belongs to faction (via component or other means)
        squadIDs = append(squadIDs, squadData.SquadID)
    }
    return squadIDs
}
```

### Victory Conditions

```go
// combat/victory.go (conceptual)
func CheckVictoryCondition(manager *common.EntityManager) (bool, ecs.EntityID) {
    activeFactions := make(map[ecs.EntityID]bool)

    for _, result := range manager.World.Query(SquadTag) {
        squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
        if !IsSquadDestroyed(squadData.SquadID, manager) {
            factionID := GetSquadFaction(squadData.SquadID, manager)
            activeFactions[factionID] = true
        }
    }

    if len(activeFactions) == 1 {
        // One faction remains - victory
        for factionID := range activeFactions {
            return true, factionID
        }
    }

    return false, 0
}
```

### AI System (New in 2026)

**File:** `tactical/ai/`, `tactical/behavior/`

TinkerRogue features a utility AI system with threat map generation for tactical decision-making.

**Threat Map System:**

```go
// AI evaluates danger at each position on the battlefield
type ThreatMap struct {
    DangerLevels map[coords.LogicalPosition]DangerLevel
    FactionID    ecs.EntityID
}

// Danger levels guide movement and positioning
type DangerLevel int
const (
    DangerNone DangerLevel = iota
    DangerLow
    DangerMedium
    DangerHigh
    DangerExtreme
)
```

**AI Decision Making:**

1. **Threat Assessment** - Calculate danger at each tile based on enemy squads
2. **Utility Evaluation** - Score potential actions (attack, move, ability)
3. **Action Selection** - Choose highest-utility action
4. **Execution** - Perform action via combat service layer

**Visualization Tools:**

- `DangerVisualizer` - Renders threat map overlay for debugging
- Combat simulation recording for analysis

### Combat Service Layer

The combat service provides a higher-level API over the ECS components.

```go
// combat/combatservices/combat_service.go
type CombatService struct {
    manager     *common.EntityManager
    turnManager *combat.TurnManager
}

func (cs *CombatService) StartCombat(playerFaction, enemyFaction ecs.EntityID) error {
    factionIDs := []ecs.EntityID{playerFaction, enemyFaction}
    return cs.turnManager.InitializeCombat(factionIDs)
}

func (cs *CombatService) ExecuteSquadAttack(attackerID, defenderID ecs.EntityID) (*squads.CombatResult, error) {
    // Validation
    if !cs.CanSquadAct(attackerID) {
        return nil, fmt.Errorf("squad cannot act")
    }

    // Execute via squad system
    result := squads.ExecuteSquadAttack(attackerID, defenderID, cs.manager)

    // Update action state
    cs.MarkSquadActed(attackerID)

    return result, nil
}
```

---

> **GUI Architecture** has been moved to its own document. See [`docs/gui_documentation/GUI_DOCUMENTATION.md`](../gui_documentation/GUI_DOCUMENTATION.md) for full details.

> **Inventory & Gear** has been moved to its own document. See [`ARTIFACT_SYSTEM.md`](ARTIFACT_SYSTEM.md) for full details.

---

## World Generation

The worldmap system (`worldmap/`) provides procedural map generation with a plugin-style generator registry.

### Generator Architecture

```go
// worldmap/generator.go:14-24
type MapGenerator interface {
    Generate(width, height int, images TileImageSet) GenerationResult
    Name() string
    Description() string
}

type GenerationResult struct {
    Tiles          []*Tile
    Rooms          []Rect
    ValidPositions []coords.LogicalPosition
}
```

### Generator Registry

Generators register themselves in `init()` functions:

```go
// worldmap/gen_rooms_corridors.go (conceptual)
type RoomsCorridorsGenerator struct {
    config GeneratorConfig
}

func init() {
    RegisterGenerator(&RoomsCorridorsGenerator{
        config: DefaultConfig(),
    })
}

func (g *RoomsCorridorsGenerator) Name() string {
    return "rooms_corridors"
}

func (g *RoomsCorridorsGenerator) Description() string {
    return "Classic dungeon with rooms connected by corridors"
}
```

**Critical Warning**: New generators MUST register in `init()` or they won't be available.

### Available Generators

1. **rooms_corridors**: Classic dungeon (default)
   - Rectangular rooms connected by L-shaped corridors
   - Binary Space Partitioning (BSP) room placement
   - File: `worldmap/gen_rooms_corridors.go`

2. **tactical_biome**: Tactical combat maps
   - Environmental features (trees, rocks, water)
   - Cover mechanics integration
   - File: `worldmap/gen_tactical_biome.go`

3. **overworld**: World map generation
   - Large-scale terrain generation
   - Location placement (towns, dungeons)
   - File: `worldmap/gen_overworld.go`

### Using Generators

```go
// Get generator by name
gen := worldmap.GetGenerator("rooms_corridors")
if gen == nil {
    gen = worldmap.GetGeneratorOrDefault("rooms_corridors")  // Fallback
}

// Generate map
images := loadTileImages()
result := gen.Generate(100, 80, images)

// result.Tiles contains the generated map
// result.Rooms contains room rectangles
// result.ValidPositions contains spawn points
```

### Tile System

```go
// worldmap/dungeontile.go (conceptual)
type Tile struct {
    TileType      TileType
    LogicalPos    coords.LogicalPosition
    Image         *ebiten.Image
    IsWalkable    bool
    IsTransparent bool
    // ... more properties
}

type TileType int

const (
    TileFloor TileType = iota
    TileWall
    TileWater
    TileDoor
    TileStairsUp
    TileStairsDown
)
```

### Room-Based Generation Algorithm

```go
// worldmap/gen_rooms_corridors.go (conceptual)
func (g *RoomsCorridorsGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
    tiles := initializeTiles(width, height, TileWall, images.Wall)
    rooms := []Rect{}

    // Binary Space Partitioning
    partitions := bspPartition(Rect{0, 0, width, height}, g.config.MinRoomSize)

    // Create room in each partition
    for _, partition := range partitions {
        room := carveRoom(partition, tiles, images.Floor)
        rooms = append(rooms, room)
    }

    // Connect rooms with corridors
    for i := 0; i < len(rooms)-1; i++ {
        connectRooms(rooms[i], rooms[i+1], tiles, images.Floor)
    }

    // Place doors
    placeDoors(rooms, tiles, images.Door)

    // Collect valid positions
    validPositions := getFloorPositions(tiles)

    return GenerationResult{
        Tiles:          tiles,
        Rooms:          rooms,
        ValidPositions: validPositions,
    }
}
```

### Biome System

```go
// worldmap/biome.go (conceptual)
type Biome struct {
    Name            string
    FloorTileType   TileType
    WallTileType    TileType
    FeatureDensity  float64  // 0.0-1.0
    Features        []FeatureType
}

type FeatureType int

const (
    FeatureTree FeatureType = iota
    FeatureRock
    FeatureWater
    FeatureChasm
)

var Biomes = map[string]Biome{
    "dungeon": {
        Name:           "Stone Dungeon",
        FloorTileType:  TileFloor,
        WallTileType:   TileWall,
        FeatureDensity: 0.05,
        Features:       []FeatureType{FeatureRock},
    },
    "forest": {
        Name:           "Dense Forest",
        FloorTileType:  TileGrass,
        WallTileType:   TileTree,
        FeatureDensity: 0.15,
        Features:       []FeatureType{FeatureTree, FeatureRock},
    },
}
```

### A* Pathfinding

```go
// worldmap/astar.go (conceptual)
func FindPath(start, goal coords.LogicalPosition, gameMap *GameMap) []coords.LogicalPosition {
    // A* implementation
    openSet := []Node{{Pos: start, GScore: 0, FScore: heuristic(start, goal)}}
    cameFrom := make(map[coords.LogicalPosition]coords.LogicalPosition)

    for len(openSet) > 0 {
        current := popLowestFScore(openSet)

        if current.Pos == goal {
            return reconstructPath(cameFrom, current.Pos)
        }

        for _, neighbor := range getNeighbors(current.Pos, gameMap) {
            if !isWalkable(neighbor, gameMap) {
                continue
            }

            tentativeGScore := current.GScore + 1

            if tentativeGScore < getGScore(neighbor) {
                cameFrom[neighbor] = current.Pos
                updateScores(neighbor, tentativeGScore, heuristic(neighbor, goal))
                openSet = append(openSet, Node{Pos: neighbor, ...})
            }
        }
    }

    return nil  // No path found
}
```

---

## Input System

The input system (`input/`) uses a **coordinator pattern** to dispatch input to specialized controllers.

### Input Coordinator

```go
// input/inputcoordinator.go:37-46
type InputCoordinator struct {
    movementController *MovementController
    combatController   *CombatController
    uiController       *UIController
    sharedState        *SharedInputState

    ecsManager *common.EntityManager
    playerData *common.PlayerData
    gameMap    *worldmap.GameMap
}
```

### Input Priority Chain

```go
// inputcoordinator.go:64-83
func (ic *InputCoordinator) HandleInput() bool {
    inputHandled := false

    // Check UI input first (highest priority)
    if ic.uiController.CanHandle() {
        inputHandled = ic.uiController.HandleInput() || inputHandled
    }

    // Then combat input (throwing/shooting)
    if ic.combatController.CanHandle() {
        inputHandled = ic.combatController.HandleInput() || inputHandled
    }

    // Finally movement input (lowest priority)
    if ic.movementController.CanHandle() {
        inputHandled = ic.movementController.HandleInput() || inputHandled
    }

    return inputHandled
}
```

**Priority Order**:
1. UI Input (inventory, menus)
2. Combat Input (attack, throw, abilities)
3. Movement Input (WASD, arrow keys)

### Controller Interface

```go
// inputcoordinator.go:30-35
type InputController interface {
    HandleInput() bool
    CanHandle() bool      // Is this controller active?
    OnActivate()          // Called when controller becomes active
    OnDeactivate()        // Called when controller becomes inactive
}
```

### Movement Controller

```go
// input/movementcontroller.go (conceptual)
type MovementController struct {
    manager     *common.EntityManager
    playerData  *common.PlayerData
    gameMap     *worldmap.GameMap
    sharedState *SharedInputState
}

func (mc *MovementController) CanHandle() bool {
    // Can handle if not in combat mode
    return !mc.playerData.InCombat
}

func (mc *MovementController) HandleInput() bool {
    playerID := mc.playerData.PlayerSquadID

    // Get player position
    playerPos := common.GetPositionByID(mc.manager, playerID)
    if playerPos == nil {
        return false
    }

    newPos := *playerPos

    // Process directional input
    if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
        newPos.Y--
    } else if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
        newPos.Y++
    }

    if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
        newPos.X--
    } else if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
        newPos.X++
    }

    // Check if moved
    if newPos != *playerPos {
        if mc.isValidMove(newPos) {
            mc.movePlayer(playerID, *playerPos, newPos)
            return true
        }
    }

    return false
}

func (mc *MovementController) movePlayer(playerID ecs.EntityID,
                                         oldPos, newPos coords.LogicalPosition) {
    // Update position component
    playerPos := common.GetPositionByID(mc.manager, playerID)
    *playerPos = newPos

    // Update spatial system
    common.GlobalPositionSystem.MoveEntity(playerID, oldPos, newPos)

    // Check for encounters
    mc.checkForEncounter(newPos)
}
```

### Combat Controller

```go
// input/combatcontroller.go (conceptual)
type CombatController struct {
    manager     *common.EntityManager
    playerData  *common.PlayerData
    gameMap     *worldmap.GameMap
    sharedState *SharedInputState
}

func (cc *CombatController) CanHandle() bool {
    // Can handle if in combat mode
    return cc.playerData.InCombat
}

func (cc *CombatController) HandleInput() bool {
    // Handle attack targeting
    if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
        target := cc.getTargetAtMouse()
        if target != 0 {
            cc.executeAttack(cc.playerData.SelectedSquad, target)
            return true
        }
    }

    // Handle ability keys (1-4)
    if ebiten.IsKeyPressed(ebiten.Key1) {
        cc.useAbility(0)  // Slot 0
        return true
    }
    // ... more ability keys

    // Handle item throwing
    if cc.playerData.InputStates.ThrowingItem {
        if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
            target := cc.getMouseLogicalPosition()
            cc.throwItem(cc.playerData.SelectedItemIndex, target)
            return true
        }
    }

    return false
}
```

### UI Controller

```go
// input/uicontroller.go (conceptual)
type UIController struct {
    playerData  *common.PlayerData
    sharedState *SharedInputState
}

func (uc *UIController) CanHandle() bool {
    // Can handle if UI window is open
    return uc.playerData.InputStates.InventoryOpen ||
           uc.playerData.InputStates.SquadBuilderOpen
}

func (uc *UIController) HandleInput() bool {
    // Toggle inventory
    if justPressed(ebiten.KeyI) {
        uc.playerData.InputStates.InventoryOpen = !uc.playerData.InputStates.InventoryOpen
        return true
    }

    // Toggle squad builder
    if justPressed(ebiten.KeyB) {
        uc.playerData.InputStates.SquadBuilderOpen = !uc.playerData.InputStates.SquadBuilderOpen
        return true
    }

    // Close windows on Escape
    if justPressed(ebiten.KeyEscape) {
        uc.closeAllWindows()
        return true
    }

    return false
}
```

### Shared Input State

```go
// inputcoordinator.go:12-18
type SharedInputState struct {
    PrevCursor         coords.PixelPosition
    PrevThrowInds      []int
    PrevRangedAttInds  []int
    PrevTargetLineInds []int
    TurnTaken          bool
}
```

Shared state allows controllers to communicate state between frames (e.g., cursor highlighting, targeting indicators).

---

## Data Flow Patterns

Understanding how data flows through the system is critical for debugging and extending functionality.

### Player Action Flow

```
User Input (Keyboard/Mouse)
    ↓
InputCoordinator.HandleInput()
    ↓
[Priority Chain]
    ├─ UIController (if UI open)
    ├─ CombatController (if in combat)
    └─ MovementController (otherwise)
    ↓
Controller modifies ECS components
    ↓
System functions process changes
    ↓
Rendering reads ECS state
    ↓
Display updated to screen
```

### Combat Action Flow

```
Player selects attack target
    ↓
CombatController.executeAttack()
    ↓
combat/CombatService.ExecuteSquadAttack()
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
2. Create ECS Manager
   ↓
3. Register Components (componentinit.go)
   ├─ common.PositionComponent
   ├─ common.AttributeComponent
   ├─ squads.SquadComponent
   ├─ gear.InventoryComponent
   └─ ... 30+ components
   ↓
4. Create Tags
   ├─ "renderables" = Renderable + Position
   ├─ "monsters" = Attributes + Position
   └─ ... tags for queries
   ↓
5. Initialize Systems (gameinit.go)
   ├─ GlobalPositionSystem = NewPositionSystem(manager)
   ├─ InputCoordinator = NewInputCoordinator()
   └─ ... other systems
   ↓
6. Load Templates (ReadGameData)
   ├─ Read monsterdata.json → MonsterTemplates
   ├─ Read weapondata.json → WeaponTemplates
   └─ Read consumabledata.json → ConsumableTemplates
   ↓
7. Generate World Map
   ├─ generator := GetGenerator("rooms_corridors")
   ├─ result := generator.Generate(80, 50)
   └─ gameMap := NewGameMap(result)
   ↓
8. Create Player Entity
   ├─ player := manager.NewEntity()
   ├─ Add components (Attributes, Position, Renderable, Inventory)
   └─ GlobalPositionSystem.AddEntity(playerID, playerPos)
   ↓
9. Spawn Entities
   ├─ SpawnMonsters(manager, gameMap, 20)
   ├─ SpawnItems(manager, gameMap, 10)
   └─ Each spawn registers in PositionSystem
   ↓
10. Setup UI Modes (gamesetup.go)
    ├─ inventoryMode := NewInventoryMode()
    ├─ squadMode := NewSquadManagementMode()
    └─ modeManager.RegisterMode(...)
    ↓
11. Run Game Loop (ebiten.RunGame)
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
┌───────▼────────┐          ┌──────▼─────────┐
│ UI Update      │          │ Systems Update │
│ (EbitenUI)     │          │                │
└───────┬────────┘          └──────┬─────────┘
        │                          │
        │                    ┌─────▼──────┐
        │                    │Visual      │
        │                    │Effects     │
        │                    │Update      │
        │                    └─────┬──────┘
        │                          │
        │                    ┌─────▼──────┐
        │                    │Debug       │
        │                    │Input       │
        │                    └─────┬──────┘
        │                          │
        └──────────┬───────────────┘
                   │
           ┌───────▼────────┐
           │ Input          │
           │ Coordinator    │
           │                │
           │ (Priority-     │
           │  based         │
           │  routing)      │
           └────────┬───────┘
                    │
          ┌─────────┴──────────┐
          │                    │
    ┌─────▼────┐        ┌─────▼──────┐
    │ UI Input │        │ Game Input │
    │ (if UI   │        │ (if no UI) │
    │  active) │        │            │
    └──────────┘        └─────┬──────┘
                              │
                       ┌──────▼─────────┐
                       │ Movement       │
                       │ Combat Actions │
                       │ Item Use       │
                       └────────────────┘

┌─────────────────────────────────────────────────┐
│          Draw() - 60 FPS                        │
└─────────────────────┬───────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        │                           │
┌───────▼────────┐          ┌──────▼─────────┐
│ Map Rendering  │          │ Entity         │
│                │          │ Rendering      │
│ (Tiles)        │          │                │
└───────┬────────┘          └──────┬─────────┘
        │                          │
        └──────────┬───────────────┘
                   │
           ┌───────▼────────┐
           │ Visual Effects │
           │ Rendering      │
           └────────┬───────┘
                    │
           ┌────────▼────────┐
           │ UI Rendering    │
           │ (EbitenUI)      │
           └─────────────────┘
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

## Component Catalog

Complete reference of all components in TinkerRogue.

### Common Components (common/)

#### PositionComponent
```go
type LogicalPosition struct {
    X int
    Y int
}
```
**Usage**: World position of entities
**Systems**: GlobalPositionSystem, MovementController
**Tag**: AllEntitiesTag

#### AttributeComponent
```go
type Attributes struct {
    MaxHP     int
    HP        int
    Strength  int
    Dexterity int
    Vitality  int
    Wisdom    int
}
```
**Usage**: Character stats
**Systems**: Combat resolution, item effects
**Tag**: AllEntitiesTag

#### NameComponent
```go
type Name struct {
    NameStr string
}
```
**Usage**: Display name for entities
**Systems**: GUI, combat log
**Tag**: AllEntitiesTag

### Squad Components (squads/)

See [Squad System](#squad-system) for detailed documentation.

| Component | Purpose | Fields |
|-----------|---------|--------|
| SquadComponent | Squad properties | SquadID, Formation, Name, Morale |
| SquadMemberComponent | Unit-to-squad link | SquadID |
| GridPositionComponent | 3x3 grid position | AnchorRow, AnchorCol, Width, Height |
| UnitRoleComponent | Combat role | Role (Tank/DPS/Support) |
| CoverComponent | Defensive cover | CoverValue, CoverRange |
| LeaderComponent | Leader bonuses | Leadership, Experience |
| AbilitySlotComponent | Leader abilities | Slots[4] |
| TargetRowComponent | Attack targeting | TargetRows, TargetCells |

### Combat Components (combat/)

| Component | Purpose | Fields |
|-----------|---------|--------|
| TurnStateComponent | Turn tracking | CurrentRound, TurnOrder, CurrentTurnIndex |
| ActionStateComponent | Squad actions | SquadID, HasMoved, HasActed |
| FactionComponent | Faction membership | FactionID, FactionType |

### Gear Components (gear/)

| Component | Purpose | Fields |
|-----------|---------|--------|
| InventoryComponent | Item storage | ItemEntityIDs[]ecs.EntityID |
| ItemComponent | Item properties | ItemID, Count, ItemType, Effects, Actions |
| EquipmentComponent | Equipped gear | SlotType, BonusAttributes |

### Worldmap Components (worldmap/)

| Component | Purpose | Fields |
|-----------|---------|--------|
| TileComponent | Tile properties | TileType, IsWalkable, IsTransparent |
| LocationComponent | World locations | LocationType, Name, Level |

---

## Appendices

### Appendix A: File Reference

#### Critical Files (Updated 2026)

| File | Lines | Purpose |
|------|-------|---------|
| `common/ecsutil.go` | 302 | Type-safe component access helpers |
| `world/coords/cordmanager.go` | 235 | Global coordinate conversions |
| `world/systems/positionsystem.go` | 183 | O(1) spatial queries |
| `tactical/squads/squadcomponents.go` | 331 | Reference ECS component design |
| `tactical/squads/squadqueries.go` | ~200 | Reference query functions |
| `gear/Inventory.go` | 229 | Reference pure ECS system |
| `tactical/combat/turnmanager.go` | 155 | Turn-based combat |
| `tactical/ai/` | ~800 | Utility AI and threat maps |

#### Package Overviews (Updated 2026)

| Package | Files | LOC | Purpose |
|---------|-------|-----|---------|
| `common/` | 8 | ~800 | Core ECS utilities |
| `world/coords/` | 3 | ~400 | Coordinate management |
| `world/systems/` | 2 | ~250 | Position system |
| `world/worldmap/` | 10 | ~2000 | Map generation |
| `tactical/squads/` | 12 | ~4900 | Squad system |
| `tactical/combat/` | 8 | ~1200 | Combat management |
| `tactical/combatservices/` | 6 | ~800 | Combat service layer |
| `tactical/squadservices/` | 4 | ~500 | Squad service layer |
| `tactical/squadcommands/` | 8 | ~600 | Undo/redo commands |
| `tactical/ai/` | 6 | ~800 | AI and threat maps |
| `tactical/behavior/` | 3 | ~400 | Behavior trees |
| `visual/rendering/` | 5 | ~600 | Batch rendering |
| `gear/` | 7 | ~900 | Inventory & items |
| `input/` | 5 | ~600 | Input handling |
| `tools/combatsim/` | 4 | ~500 | Combat analysis tools |

### Appendix B: Glossary

**ECS**: Entity Component System - architectural pattern separating data (components) from logic (systems)

**EntityID**: Stable integer identifier for entities (preferred over pointers)

**Tag**: ECS query selector built from one or more components

**Component**: Pure data structure with zero methods

**System**: Function that operates on components (not part of component struct)

**Query**: ECS operation that finds entities with specific components

**LogicalPosition**: Grid coordinates (0-99, 0-79)

**PixelPosition**: Rendering coordinates before scaling

**ScreenPosition**: Final on-screen coordinates after viewport transform

**Formation**: Squad layout preset (Balanced, Defensive, Offensive, Ranged)

**Grid Position**: Unit position in 3x3 squad grid (row 0-2, col 0-2)

**Controller**: Input handler specialized for specific game state

**Generator**: Procedural map creation algorithm

**Biome**: Environmental theme for map generation

**Template**: JSON data file defining entity properties

**Service**: Higher-level API layer over ECS systems

**Faction**: Group of squads (player faction, enemy factions)

**Turn Manager**: Combat system that tracks turn order and rounds

**Spatial Grid**: O(1) position lookup data structure

**Threat Map**: AI tactical map showing danger levels at each battlefield position

**Utility AI**: AI decision-making system that scores actions by utility value

**Combat Visualization**: File-based recording system for combat testing and analysis

**Command Pattern**: Undo/redo support for user actions via reversible commands

### Appendix C: Common Mistakes

#### Using Components Before Initialization
```go
// ❌ WRONG - nil pointer panic
entity.AddComponent(SquadComponent, data)  // SquadComponent is nil!

// ✅ CORRECT - Initialize first
squads.InitSquadComponents(manager)
entity.AddComponent(SquadComponent, data)
```

#### Manual Index Calculation
```go
// ❌ WRONG - Index out of bounds
idx := y*width + x  // width might not match dungeonWidth!
result.Tiles[idx] = &tile

// ✅ CORRECT - Use CoordManager
idx := coords.CoordManager.LogicalToIndex(logicalPos)
result.Tiles[idx] = &tile
```

#### Storing Entity Pointers
```go
// ❌ WRONG - Can become dangling
type SquadData struct {
    Units []*ecs.Entity  // Pointers become invalid!
}

// ✅ CORRECT - Store EntityIDs
type SquadData struct {
    // Don't even cache this - query instead!
}

func GetUnitsInSquad(squadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    // Query when needed
}
```

#### Pointer Map Keys
```go
// ❌ WRONG - 50x slower
grid map[*coords.LogicalPosition][]ecs.EntityID

// ✅ CORRECT - Use values
grid map[coords.LogicalPosition][]ecs.EntityID
```

#### Logic in Components
```go
// ❌ WRONG - Component method
func (s *SquadData) Attack(target *SquadData) {
    // NO! This is a system function
}

// ✅ CORRECT - System function
func ExecuteAttack(attackerID, defenderID ecs.EntityID, manager *common.EntityManager) {
    // Logic here
}
```

#### Forgetting Generator Registration
```go
// ❌ WRONG - Generator not registered
type MyGenerator struct {}
func (g *MyGenerator) Generate(...) { /* ... */ }
// Forgot init()!

// ✅ CORRECT - Register in init
func init() {
    RegisterGenerator(&MyGenerator{})
}
```

#### GUI-Specific Mistakes

> See [`docs/gui_documentation/GUI_DOCUMENTATION.md`](../gui_documentation/GUI_DOCUMENTATION.md) for GUI-specific common mistakes and patterns.

### Appendix D: Testing Patterns

#### Component Tests
```go
func TestSquadData(t *testing.T) {
    squadData := &SquadData{
        SquadID:   1,
        Formation: FormationLine,
        Name:      "Test Squad",
        Morale:    100,
    }

    assert.Equal(t, 100, squadData.Morale)
    assert.Equal(t, "Test Squad", squadData.Name)
}
```

#### Query Function Tests
```go
func TestGetSquadEntity(t *testing.T) {
    manager := common.CreateTestEntityManager()
    InitSquadComponents(manager)
    InitSquadTags(manager)

    // Create test squad
    entity := manager.World.CreateEntity()
    squadID := ecs.EntityID(1)
    entity.AddComponent(SquadComponent, &SquadData{SquadID: squadID})
    entity.AddTag(SquadTag)

    // Test query
    result := GetSquadEntity(squadID, manager)
    assert.NotNil(t, result)

    // Test not found
    notFound := GetSquadEntity(ecs.EntityID(999), manager)
    assert.Nil(t, notFound)
}
```

#### System Function Tests
```go
func TestExecuteSquadAttack(t *testing.T) {
    manager := common.CreateTestEntityManager()
    InitSquadComponents(manager)
    InitSquadTags(manager)

    // Create attacker
    attackerID := CreateSquad("Attacker", FormationLine, manager)

    // Create defender
    defenderID := CreateSquad("Defender", FormationLine, manager)

    // Execute attack
    result := ExecuteSquadAttack(attackerID, defenderID, manager)

    // Verify
    assert.NotNil(t, result)
    assert.Greater(t, result.DamageDealt, 0)

    defenderData := common.GetComponentTypeByIDWithTag[*SquadData](
        manager, defenderID, SquadTag, SquadComponent)
    assert.Less(t, defenderData.Morale, 100)
}
```

### Appendix E: Detailed Package Guide

This appendix provides detailed API documentation for each major package in TinkerRogue, including file structure, responsibilities, dependencies, and public interfaces.

#### E.1 `game_main/` - Entry Point & Initialization

**Purpose:** Application entry point, game loop, system initialization.

**Key Files:**
- `main.go` (Game struct, Update/Draw loop)
- `componentinit.go` - Component registration
- `gameinit.go` - System initialization
- `gamesetup.go` - UI mode setup
- `config.go` - Configuration constants

**Responsibilities:**
- Initialize ECS manager and register all components
- Create global systems (Position System, etc.)
- Setup game state (player, map, UI modes)
- Run game loop at 60 FPS
- Handle graceful shutdown

**Dependencies:**
- All other packages (this is the root)

**Public API:**
```go
type Game struct {
    em               *common.EntityManager
    playerData       common.PlayerData
    gameMap          *worldmap.GameMap
    uiModeManager    *gui.UIModeManager
    // ... systems
}

func (g *Game) Update() error  // 60 FPS, turn-based state machine
func (g *Game) Draw(screen *ebiten.Image)
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int)
```

### Appendix F: 2026 Changes Summary

#### Major Architectural Changes

**Package Reorganization:**
- Created `tactical/` parent package for all combat-related systems
- Created `world/` parent package for spatial systems
- Created `visual/` package for rendering utilities
- Created `tools/` package for development tools
- Restructured `gui/` with `framework/`, `builders/`, `widgets/`, `specs/` subpackages

> **GUI System Overhaul:** See [`docs/gui_documentation/GUI_DOCUMENTATION.md`](../gui_documentation/GUI_DOCUMENTATION.md) for full details on the 2026 GUI changes.

**Combat Enhancements:**
1. **Threat Map System** - AI uses tactical danger maps
2. **Multiple Factions** - Hot-seat multiplayer support
3. **Movement System** - Multi-tile movement over multiple actions
4. **Combat Visualization** - File-based combat recording
5. **Combat Animation** - Visual feedback system

**Performance Improvements:**
- 90% CPU reduction in GUI rendering (cached lists/textareas)
- Batched tile and sprite rendering
- Static panel backgrounds
- DrawImageOptions reuse
- Optimized spatial queries remain O(1)

#### Migration Notes (For Developers)

**Old → New Package Paths:**
```
squads/ → tactical/squads/
combat/ → tactical/combat/
coords/ → world/coords/
systems/ → world/systems/
worldmap/ → world/worldmap/
entitytemplates/ → templates/
gui/core/ → gui/framework/
```

---

**Document Version**: 3.1 (2026 Edition)
**Last Updated**: 2026-02-17
**Total Sections**: 13 + Appendices (A-F)
**Major Changes**: Extracted GUI, Inventory/Gear, and Overworld sections to dedicated documents
