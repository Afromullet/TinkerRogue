# TinkerRogue: Technical Documentation

**Last Updated:** 2025-11-21
**Version:** 5.0 - Architectural Edition

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Architecture Overview](#2-architecture-overview)
3. [Core Systems](#3-core-systems)
4. [Package Guide](#4-package-guide)
5. [Data Flow & Integration](#5-data-flow--integration)
6. [Development Guide](#6-development-guide)
7. [Reference](#7-reference)

---

## 1. Introduction

### What is TinkerRogue?

TinkerRogue is a **turn-based tactical roguelike** built in Go using the Ebiten game engine. It combines classic roguelike dungeon crawling with tactical squad-based combat inspired by games like Final Fantasy Tactics and Fire Emblem.

**Core Gameplay:**
- Explore procedurally generated dungeons
- Command squads arranged in 3x3 tactical formations
- Manage inventory, equipment, and consumables
- Engage in turn-based combat with emergent tactical depth

### Design Philosophy

The codebase is built on three foundational principles:

1. **Pure Entity Component System (ECS) Architecture**
   - Entities are lightweight IDs, not objects
   - Components are pure data with zero logic
   - Systems are functions that operate on components
   - This enables extreme flexibility and composability

2. **Data-Driven Design**
   - Game content (monsters, items, weapons) defined in JSON
   - Entity templates loaded at runtime
   - Designers can modify content without touching code

3. **Performance Through Architecture**
   - Type-safe coordinate systems prevent bugs
   - O(1) spatial queries using value-based map keys
   - Clean separation of concerns for maintainability

### Key Technical Features

- **ECS Framework:** bytearena/ecs library
- **Game Engine:** Ebiten v2 (2D, cross-platform)
- **UI Framework:** EbitenUI for complex UI modes
- **Spatial Queries:** Custom O(1) position system (50x faster than naive approaches)
- **Squad Combat:** 2675 LOC tactical combat system with abilities, formations, and multi-cell units
- **Inventory System:** 533 LOC ECS-compliant item management
- **Map Generation:** Strategy pattern with pluggable generators (rooms-and-corridors, tactical biome)

---

## 2. Architecture Overview

### 2.1 The Big Picture

TinkerRogue's architecture follows a strict ECS pattern with clear system boundaries:

```
┌─────────────────────────────────────────────────────────────────┐
│                         GAME LOOP                                │
│  (game_main/main.go - 60 FPS, turn-based state machine)         │
└────────────┬────────────────────────────────────────────────────┘
             │
        ┌────▼────┐
        │ UPDATE  │
        └────┬────┘
             │
    ┌────────┴────────┐
    │                 │
┌───▼───────┐  ┌─────▼──────┐
│   INPUT   │  │   SYSTEMS  │
│  SYSTEM   │  │  (ECS)     │
└───┬───────┘  └─────┬──────┘
    │                │
    │     ┌──────────┼──────────┐
    │     │          │          │
    │  ┌──▼───┐  ┌──▼────┐  ┌──▼─────┐
    │  │Squad │  │Combat │  │Position│
    │  │System│  │System │  │System  │
    │  └──┬───┘  └──┬────┘  └──┬─────┘
    │     │         │          │
    │     └─────────┼──────────┘
    │               │
    │     ┌─────────▼─────────┐
    │     │   ECS MANAGER     │
    │     │  (Entity Store)   │
    │     └─────────┬─────────┘
    │               │
    │         ┌─────▼──────┐
    │         │ COMPONENTS │
    │         │ (Pure Data)│
    │         └────────────┘
    │
┌───▼────┐
│  DRAW  │
└───┬────┘
    │
┌───▼────────┐
│ RENDERING  │
│  SYSTEM    │
└────┬───────┘
     │
┌────▼────┐
│ SCREEN  │
└─────────┘
```

### 2.2 Architectural Layers

The codebase is organized into distinct layers:

#### Layer 1: Core Infrastructure
- **ECS Manager** (`common/`) - Entity and component registration
- **Coordinate System** (`coords/`) - LogicalPosition ↔ PixelPosition conversions
- **Position System** (`systems/positionsystem.go`) - O(1) spatial lookups

#### Layer 2: Game Systems
- **Input System** (`input/`) - Priority-based input handling (UI → Combat → Movement)
- **Squad System** (`squads/`) - Tactical formations, combat, abilities
- **Inventory System** (`gear/`) - Item management, equipment
- **Combat System** - Attack resolution, damage calculation
- **World Map** (`worldmap/`) - Procedural generation with strategy pattern

#### Layer 3: Presentation
- **Rendering System** (`rendering/`) - Entity sprite rendering
- **Graphics System** (`graphics/`) - Visual effects, shapes, animations
- **GUI System** (`gui/`) - Modal UI interfaces (inventory, squad management, etc.)

#### Layer 4: Content
- **Entity Templates** (`entitytemplates/`) - JSON-based entity factories
- **Asset Management** - Image loading, data file parsing

### 2.3 Key Architectural Patterns

#### Pure ECS Architecture

The entire game follows strict ECS principles:

```go
// Entities are lightweight handles
entity := manager.NewEntity()
entityID := entity.GetID()  // Use EntityID, not pointers

// Components are pure data (no methods)
type Position struct {
    X, Y int  // Just data
}

// Systems are functions that operate on components
func MoveSystem(manager *ecs.Manager) {
    for _, result := range manager.Query(movableTag) {
        pos := GetComponent[*Position](result.Entity)
        pos.X += 1  // System logic
    }
}
```

**Why This Matters:**
- Components can be added/removed at runtime (extreme flexibility)
- No inheritance hierarchy to manage
- Systems are testable in isolation
- Easy to parallelize systems (future optimization)

#### Query-Based Relationships

Instead of storing references, we discover relationships through queries:

```go
// ✅ CORRECT: Query-based discovery
func GetUnitsInSquad(squadID EntityID) []EntityID {
    var units []EntityID
    for _, result := range manager.Query(squadMemberTag) {
        member := GetComponent[*SquadMemberData](result.Entity)
        if member.SquadID == squadID {
            units = append(units, result.Entity.GetID())
        }
    }
    return units
}

// ❌ WRONG: Stored references (manual synchronization)
type Squad struct {
    Units []EntityID  // Must manually update on add/remove
}
```

**Benefits:**
- Always reflects actual state
- No synchronization bugs
- Handles entity destruction automatically

#### Value-Based Map Keys

The Position System achieves O(1) lookups using value types as map keys:

```go
// ✅ CORRECT: Value-based keys (O(1) hash lookup)
type PositionSystem struct {
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID
}

// ❌ WRONG: Pointer keys (O(n) comparison)
spatialGrid map[*coords.LogicalPosition]*ecs.Entity
```

**Impact:** 50x performance improvement over pointer-based legacy system.

#### Strategy Pattern for Map Generation

World map generation uses the strategy pattern for pluggable algorithms:

```go
type MapGenerator interface {
    GetName() string
    GetDescription() string
    Generate(width, height int) *GenerationResult
}

// Register generators at init time
func init() {
    RegisterGenerator("rooms_corridors", &RoomsCorridorsGenerator{})
    RegisterGenerator("tactical_biome", &TacticalBiomeGenerator{})
}

// Use any registered generator
gameMap := worldmap.NewGameMap("tactical_biome")
```

**Benefits:**
- Add new generators without modifying existing code
- Each generator independently testable
- Clear contract via interface

### 2.4 Package Organization

```
TinkerRogue/
├── game_main/          # Entry point, game loop, initialization
├── common/             # Core ECS utilities, shared components
├── coords/             # Coordinate system and transformations
├── systems/            # ECS systems (position, movement, etc.)
├── squads/             # Squad system (components, queries, combat)
├── gear/               # Inventory and item system
├── combat/             # Combat resolution (damage, hit/miss)
├── input/              # Input handling (priority-based controllers)
├── rendering/          # Entity rendering system
├── graphics/           # Visual effects, shapes, screen utilities
├── gui/                # UI modes (inventory, squad management, etc.)
├── worldmap/           # Map generation (strategy pattern)
├── entitytemplates/    # JSON-based entity factories
├── spawning/           # Entity spawning logic
├── testing/            # Test utilities and helpers
└── assets/             # Game resources (images, JSON data)
```

### 2.5 Critical Design Decisions

#### Decision: Use EntityID Instead of Entity Pointers

**Rationale:**
- Entity pointers become stale if entity is destroyed/recreated
- Prevents serialization (can't save/load game state)
- Creates circular reference issues

**Implementation:**
All components store `ecs.EntityID` (uint64) instead of `*ecs.Entity`.

#### Decision: O(1) Position System

**Rationale:**
- Previous pointer-based tracker was O(n) for position lookups
- Caused 50x performance degradation in entity-dense scenes
- Value-based map keys enable proper hash lookups

**Implementation:**
`map[coords.LogicalPosition][]ecs.EntityID` using value types.

#### Decision: System Functions Instead of Component Methods

**Rationale:**
- Components with methods violate ECS principles
- Makes testing difficult (must create full component instances)
- Prevents logic reuse across different components

**Implementation:**
All logic in standalone functions (e.g., `gear.AddItem()` instead of `inv.AddItem()`).

#### Decision: Type-Safe Coordinate System

**Rationale:**
- Mixing logical (tile) and pixel coordinates caused constant bugs
- Explicit types prevent accidental misuse

**Implementation:**
`LogicalPosition` and `PixelPosition` as distinct types with conversion functions.

---

## 3. Core Systems

This section explains each major system, its purpose, how it works, and how it integrates with other systems.

### 3.1 Entity Component System (ECS)

#### Purpose

The ECS is the **foundation** of the entire game. Every game object (player, monster, item, squad) is an entity with components attached. All game logic operates through systems that query and manipulate these components.

#### How It Works

**Entities:**
```go
entity := manager.NewEntity()  // Creates lightweight ID container
entityID := entity.GetID()     // Get native EntityID (uint64)
```

**Components:**
```go
// Component definition (pure data, no methods)
type Attributes struct {
    Strength      int
    Dexterity     int
    CurrentHealth int
    MaxHealth     int
}

// Component registration (at startup)
common.AttributeComponent = manager.NewComponent()

// Adding component to entity
entity.AddComponent(common.AttributeComponent, &Attributes{
    Strength:      15,
    CurrentHealth: 50,
    MaxHealth:     50,
})
```

**Tags (for querying):**
```go
// Define tag (combination of components)
monsterTag := ecs.BuildTag(
    common.AttributeComponent,
    common.PositionComponent,
)

// Query entities with tag
for _, result := range manager.Query(monsterTag) {
    entity := result.Entity
    attr := GetComponent[*Attributes](entity)
    pos := GetComponent[*Position](entity)
    // Process entity
}
```

**Systems:**
```go
// System function operates on queried entities
func HealthRegenSystem(manager *ecs.Manager, monsterTag ecs.Tag) {
    for _, result := range manager.Query(monsterTag) {
        attr := GetComponent[*Attributes](result.Entity)
        if attr.CurrentHealth < attr.MaxHealth {
            attr.CurrentHealth += 1  // Regenerate health
        }
    }
}
```

#### Integration Points

- **Position System**: Queries entities with PositionComponent for spatial lookups
- **Squad System**: Queries entities with SquadMemberComponent to find squad units
- **Inventory System**: Queries entities with ItemComponent to access items
- **Rendering System**: Queries entities with RenderableComponent to draw sprites

#### Key Files

- `game_main/componentinit.go` - Component registration
- `common/ecsutil.go` - Utility functions for component access
- `common/commoncomponents.go` - Shared components (Name, Attributes, Position)

---

### 3.2 Position System

#### Purpose

Provides **O(1) spatial queries** for finding entities at specific positions or within a radius. This is critical for:
- Checking if a tile is occupied before moving
- Finding targets for combat/abilities
- Item pickup detection
- Line-of-sight calculations

#### How It Works

The Position System maintains a spatial grid using a map with value-based keys:

```go
type PositionSystem struct {
    manager     *ecs.Manager
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID
    entityToPos map[ecs.EntityID]coords.LogicalPosition
}
```

**Key Operations:**

1. **Add Entity:**
```go
posSystem.AddEntity(entityID, coords.LogicalPosition{X: 10, Y: 5})
// Registers entity at position in spatial grid
```

2. **Lookup by Position (O(1)):**
```go
entityID := posSystem.GetEntityIDAt(coords.LogicalPosition{X: 10, Y: 5})
// Returns first entity at position (0 if empty)
```

3. **Move Entity:**
```go
posSystem.MoveEntity(entityID, oldPos, newPos)
// More efficient than Remove + Add
```

4. **Radius Query:**
```go
entities := posSystem.GetEntitiesInRadius(centerPos, 3)
// Returns all entities within Chebyshev distance
```

#### Why O(1) Performance?

**The Key Insight:** Value-based map keys enable hash lookups.

```go
// LogicalPosition is a value type (not pointer)
type LogicalPosition struct {
    X, Y int
}

// Go can hash value types for O(1) map lookup
spatialGrid[LogicalPosition{X: 10, Y: 5}]  // O(1) hash lookup
```

**Legacy Approach (O(n) - removed):**
```go
// Pointer keys can't be hashed by value → O(n) iteration
spatialGrid[&LogicalPosition{X: 10, Y: 5}]  // O(n) comparison
```

**Performance Impact:** 50x speedup in entity-dense scenarios.

#### Integration Points

- **Movement System**: Checks if target position is occupied before moving
- **Combat System**: Finds targets at specific positions
- **Spawning System**: Finds empty positions for entity placement
- **GUI System**: Determines what entity the player clicked on

#### Key Files

- `systems/positionsystem.go` (399 LOC) - Complete implementation
- `common/globals.go` - GlobalPositionSystem singleton

---

### 3.3 Input System

#### Purpose

Handles all player input (keyboard, mouse) with **priority-based routing**. Higher-priority controllers (e.g., UI) can block input from reaching lower-priority controllers (e.g., movement).

#### Architecture

```
Input Event
    ↓
InputCoordinator
    ↓
┌───────────────────────────────┐
│ Priority 1: UI Controller     │ → Handles inventory/menu input
└────────────┬──────────────────┘
             │ (blocked if UI consumed input)
             ↓
┌───────────────────────────────┐
│ Priority 2: Combat Controller │ → Handles targeting/abilities
└────────────┬──────────────────┘
             │ (blocked if combat active)
             ↓
┌───────────────────────────────┐
│ Priority 3: Movement Controller│ → Handles movement/interaction
└───────────────────────────────┘
```

**Priority Rules:**
- UI mode active? UI controller handles input, others blocked
- Combat targeting mode? Combat controller handles input
- Normal exploration? Movement controller handles input

#### How It Works

**InputCoordinator:**
```go
type InputCoordinator struct {
    controllers []InputController  // Ordered by priority
}

func (ic *InputCoordinator) HandleInput() {
    for _, controller := range ic.controllers {
        if controller.CanHandle() {  // Check if controller is active
            if controller.HandleInput() {  // Handle input
                return  // Input consumed, stop propagation
            }
        }
    }
}
```

**Controller Interface:**
```go
type InputController interface {
    CanHandle() bool           // Is this controller currently active?
    HandleInput() bool         // Handle input, return true if consumed
    GetPriority() int          // Priority level (lower = higher priority)
}
```

**Example Controller:**
```go
type MovementController struct {
    playerData *common.PlayerData
    gameMap    *worldmap.GameMap
}

func (mc *MovementController) CanHandle() bool {
    // Can handle if no UI mode is active
    return !mc.uiModeActive
}

func (mc *MovementController) HandleInput() bool {
    // Check for movement keys
    if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
        newPos := mc.playerData.Pos.Translate(0, -1)
        if !mc.gameMap.IsBlocked(newPos) {
            mc.playerData.Pos = newPos
            return true  // Consumed input
        }
    }
    return false  // Input not handled
}
```

#### Integration Points

- **GUI System**: UI modes register as high-priority input controllers
- **Combat System**: Combat targeting registers as medium-priority controller
- **Movement System**: Movement registers as low-priority (fallback) controller
- **Debug System**: Debug commands bypass priority system

#### Key Files

- `input/inputcoordinator.go` - Priority-based input routing
- `input/movementcontroller.go` - Player movement input
- `input/combatcontroller.go` - Combat targeting input
- `input/debuginput.go` - Debug commands (F1-F12)

---

### 3.4 Squad System

#### Purpose

Implements **tactical squad-based combat** with:
- 3x3 grid formations (front/middle/back rows)
- Role-based units (Tank, DPS, Support)
- Multi-cell units (1x1, 2x2, 1x3, etc.)
- Leader abilities (Rally, Heal, Battle Cry, Fireball)
- Cover mechanics (units in front provide cover to units behind)
- Capacity-based squad building (Leadership stat determines squad size)

#### Architecture

The Squad System is built on 8 pure-data components:

```
Squad Entity
    ├─ SquadData (formation, morale, capacity)
    └─ (leader unit reference via query)
        ↓
Unit Entities (members)
    ├─ SquadMemberData (links to parent squad)
    ├─ GridPositionData (position in 3x3 grid)
    ├─ UnitRoleData (Tank/DPS/Support)
    ├─ Attributes (health, stats)
    ├─ AttackRangeData (melee/ranged)
    ├─ MovementSpeedData (world map speed)
    ├─ CoverData (cover bonus for units behind)
    ├─ TargetRowData (which rows this unit attacks)
    ├─ LeaderData (if unit is squad leader)
    ├─ AbilitySlotData (4 equipped abilities)
    └─ CooldownTrackerData (ability cooldowns)
```

#### How It Works

**Creating a Squad:**
```go
// 1. Create squad entity
squadEntity := manager.NewEntity().
    AddComponent(squads.SquadComponent, &squads.SquadData{
        SquadID:       squadEntity.GetID(),  // Self-reference
        Formation:     squads.FormationBalanced,
        Name:          "Alpha Squad",
        MaxUnits:      9,
        TotalCapacity: 6,  // From leader's Leadership stat
    })

// 2. Create unit entity in squad
tankUnit := manager.NewEntity().
    AddComponent(common.AttributeComponent, &common.Attributes{
        Strength:  15,  // High strength = tank
        Dexterity: 8,
    }).
    AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
        SquadID: squadEntity.GetID(),  // Link to squad
    }).
    AddComponent(squads.GridPositionComponent, &squads.GridPositionData{
        AnchorRow: 0,  // Front row
        AnchorCol: 1,  // Center column
        Width:     1,  // Single cell
        Height:    1,
    }).
    AddComponent(squads.UnitRoleComponent, &squads.UnitRoleData{
        Role: squads.RoleTank,
    }).
    AddComponent(squads.AttackRangeComponent, &squads.AttackRangeData{
        Range: 1,  // Melee range
    })
```

**Querying Squad Units:**
```go
// Query-based discovery (no stored unit list!)
func GetUnitIDsInSquad(squadID ecs.EntityID) []ecs.EntityID {
    var unitIDs []ecs.EntityID
    for _, result := range manager.Query(squadMemberTag) {
        member := GetComponent[*SquadMemberData](result.Entity)
        if member.SquadID == squadID {
            unitIDs = append(unitIDs, result.Entity.GetID())
        }
    }
    return unitIDs
}

// Find units in specific row
func GetUnitIDsInRow(squadID ecs.EntityID, row int) []ecs.EntityID {
    var unitIDs []ecs.EntityID
    for _, unitID := range GetUnitIDsInSquad(squadID) {
        unit := FindUnitByID(unitID)
        gridPos := GetComponent[*GridPositionData](unit)
        if gridPos.OccupiesRow(row) {
            attr := GetComponent[*Attributes](unit)
            if attr.CurrentHealth > 0 {  // Only alive units
                unitIDs = append(unitIDs, unitID)
            }
        }
    }
    return unitIDs
}
```

**Combat Execution:**
```go
// Execute squad vs squad combat
result := squads.ExecuteSquadAttack(attackerSquadID, defenderSquadID, manager)

// Result contains:
// - TotalDamage: int
// - DamageByUnit: map[EntityID]int
// - UnitsKilled: []EntityID
// - CriticalHits: int
// - Dodged: int
```

**Combat Flow:**
1. Find all alive units in attacker squad
2. For each attacker unit:
   - Determine which defender rows it can target (based on TargetRowData)
   - Find alive units in target rows
   - Select target (prioritize front row for melee)
   - Calculate hit chance (attacker Dexterity vs defender Dodge)
   - Roll hit/miss
   - If hit, calculate damage (Strength + Weapon - Armor)
   - Roll critical hit (based on Dexterity)
   - Apply cover reduction (if target is behind another unit)
   - Reduce target health
3. Return combat results

**Ability System:**

Leaders have 4 ability slots with trigger conditions:

```go
type AbilitySlot struct {
    AbilityType  AbilityType   // Rally, Heal, BattleCry, Fireball
    TriggerType  TriggerType   // HPBelow, TurnCount, EnemyCount, etc.
    Threshold    float64       // Trigger condition value
    HasTriggered bool          // Once-per-combat abilities
    IsEquipped   bool
}

// Abilities auto-trigger each turn
func ProcessAbilities(squadID ecs.EntityID, turnCount int) {
    leaderID := GetLeaderID(squadID)
    leader := FindUnitByID(leaderID)
    abilitySlots := GetComponent[*AbilitySlotData](leader)

    for i, slot := range abilitySlots.Slots {
        if !slot.IsEquipped || slot.HasTriggered {
            continue
        }

        // Check trigger condition
        if CheckTrigger(slot.TriggerType, slot.Threshold, squadID, turnCount) {
            ExecuteAbility(slot.AbilityType, squadID)
            slot.HasTriggered = true
        }
    }
}
```

**Capacity System:**

Squad size is limited by leader's Leadership stat:

```go
// Calculate unit cost
func GetCapacityCost(unit *ecs.Entity) float64 {
    attr := GetAttributes(unit)
    // Cost = (Strength + Weapon + Armor) / 5.0
    return float64(attr.Strength + attr.Weapon + attr.Armor) / 5.0
}

// Calculate squad capacity
func GetSquadTotalCapacity(squadID ecs.EntityID) int {
    leaderID := GetLeaderID(squadID)
    leader := FindUnitByID(leaderID)
    attr := GetAttributes(leader)
    // Capacity = 6 + (Leadership / 3), capped at 9
    return min(9, 6 + attr.Leadership/3)
}

// Check if unit can be added
func CanAddUnitToSquad(squadID ecs.EntityID, unitCost float64) bool {
    used := GetSquadUsedCapacity(squadID)
    total := GetSquadTotalCapacity(squadID)
    return (used + unitCost) <= float64(total)
}
```

#### Integration Points

- **Position System**: Squads occupy positions on world map
- **Combat System**: Squad combat replaces individual creature combat
- **GUI System**: Squad management UI for building/editing formations
- **World Map**: Squads move across map, trigger encounters

#### Key Files

- `squads/components.go` (331 LOC) - 8 pure-data components
- `squads/squadqueries.go` (140 LOC) - Query functions
- `squads/squadcombat.go` (387 LOC) - Combat execution
- `squads/squadabilities.go` (317 LOC) - Ability system
- `squads/visualization.go` (175 LOC) - 3x3 grid rendering
- `squads/squadformations.go` - Formation presets (in progress)

**Status:** 95% complete (formation presets remaining)

---

### 3.5 Inventory System

#### Purpose

Manages **item storage and manipulation** for entities (player, monsters, containers). Fully ECS-compliant with system functions instead of component methods.

#### Architecture

```
Entity (Player/Monster)
    ├─ Inventory Component (pure data)
    │      └─ ItemEntityIDs: []ecs.EntityID
    │
    └─ (Items are separate entities)
            ↓
Item Entities
    ├─ Name Component
    ├─ Item Component
    │      ├─ Properties: ecs.EntityID (status effects entity)
    │      ├─ Actions: []ItemAction (throwable, consumable)
    │      └─ Count: int (stack count)
    └─ Renderable Component (for ground rendering)
```

#### How It Works

**Creating Items:**
```go
// Create item entity from template
itemEntity := entitytemplates.CreateEntityFromTemplate(manager, EntityConfig{
    Type:      entitytemplates.Consumable,
    Name:      "Health Potion",
    ImagePath: "../assets/items/",
    Visible:   true,
    Position:  &coords.LogicalPosition{X: 0, Y: 0},
}, nil)

// Add item actions
item := GetComponent[*gear.Item](itemEntity)
item.Actions = append(item.Actions, gear.NewConsumableAction(healing))
```

**Adding to Inventory (System Function):**
```go
// ✅ CORRECT: Use system function
playerInv := GetComponent[*gear.Inventory](playerEntity)
gear.AddItem(manager, playerInv, itemEntity.GetID())

// ❌ WRONG: Direct manipulation
playerInv.ItemEntityIDs = append(playerInv.ItemEntityIDs, itemID)  // NO!
```

**Displaying Inventory:**
```go
// Get formatted list for UI rendering
items := gear.GetInventoryForDisplay(manager, playerInv, nil)

for _, item := range items {
    entry := item.(gear.InventoryListEntry)
    // entry.Index: int
    // entry.Name: string
    // entry.Count: int
    // entry.EntityID: ecs.EntityID
    fmt.Printf("[%d] %s x%d\n", entry.Index, entry.Name, entry.Count)
}
```

**Filtering by Action:**
```go
// Get only throwable items
throwables := gear.GetThrowableItems(manager, playerInv, nil)

// Get items with specific action
consumables := gear.GetInventoryByAction(manager, playerInv, nil, "consumable")
```

**Removing Items:**
```go
// Remove item at index (decrements count or removes)
gear.RemoveItem(manager, playerInv, itemIndex)
```

#### Item Actions

Items can have multiple actions (e.g., throwable AND consumable):

```go
type ItemAction interface {
    ActionName() string           // "throwable", "consumable", etc.
    Copy() ItemAction             // For item duplication
    Execute(...)                  // Action-specific logic
}

// Throwable action
type ThrowableAction struct {
    AOEPattern     [][2]int     // Cells affected by throw
    BaseDamage     int
    StatusEffectID ecs.EntityID
}

// Consumable action
type ConsumableAction struct {
    StatusEffectID ecs.EntityID
}
```

**Using Item Actions:**
```go
item := gear.GetItemByID(manager, itemEntityID)

if item.HasAction("throwable") {
    throwable := item.GetAction("throwable").(*gear.ThrowableAction)
    // Execute throw logic
    throwable.Execute(targetPos, manager)
}

if item.HasAction("consumable") {
    consumable := item.GetAction("consumable").(*gear.ConsumableAction)
    // Execute consume logic
    consumable.Execute(playerEntity, manager)
    gear.RemoveItem(manager, playerInv, itemIndex)
}
```

#### Integration Points

- **GUI System**: Inventory UI mode displays and manages items
- **Combat System**: Throwable items used in combat
- **Player System**: Player inventory for item management
- **Entity Templates**: Items created from JSON templates
- **Position System**: Items on ground have position for pickup

#### Key Files

- `gear/Inventory.go` (241 LOC) - Pure component + system functions
- `gear/items.go` (177 LOC) - Item component + actions
- `gear/itemactions.go` - ItemAction interface + implementations
- `gear/gearutil.go` (115 LOC) - Query-based entity lookup

**Status:** 100% complete (ECS refactor completed 2025-10-21)

---

### 3.6 Rendering System

#### Purpose

Draws all entities with RenderableComponent to the screen. Handles:
- Sprite rendering at correct pixel positions
- Viewport scrolling (only render visible area)
- Z-ordering (entities render in correct order)
- Visual effects (explosions, beams, etc.)

#### Architecture

```
Draw Phase
    ↓
┌──────────────────────┐
│ ProcessRenderables   │
│ (rendering/render.go)│
└──────────┬───────────┘
           │
           ├──> Query entities with RenderableComponent
           │
           ├──> Convert LogicalPosition → PixelPosition
           │
           ├──> Check if in viewport (if scrolling enabled)
           │
           └──> Draw sprite to screen
                    ↓
            ┌─────────────┐
            │ ebiten.Image│
            └─────────────┘
```

#### How It Works

**Entity Rendering:**
```go
func ProcessRenderablesInSquare(
    screen *ebiten.Image,
    manager *ecs.Manager,
    renderableTag ecs.Tag,
    centerPos coords.LogicalPosition,
) {
    for _, result := range manager.Query(renderableTag) {
        entity := result.Entity

        // Get components
        renderable := GetComponent[*Renderable](entity)
        pos := GetComponent[*coords.LogicalPosition](entity)

        // Skip if not visible
        if !renderable.Visible {
            continue
        }

        // Convert to pixel position
        pixelPos := coords.CoordManager.LogicalToPixel(*pos)

        // Check if in viewport (if scrolling enabled)
        if graphics.MAP_SCROLLING_ENABLED {
            distance := pos.ChebyshevDistance(&centerPos)
            if distance > graphics.ViewableSquareSize / 2 {
                continue  // Outside viewport
            }
        }

        // Draw sprite
        options := &ebiten.DrawImageOptions{}
        options.GeoM.Translate(float64(pixelPos.X), float64(pixelPos.Y))
        screen.DrawImage(renderable.Image, options)
    }
}
```

**Visual Effects:**

The Graphics system handles temporary visual effects (explosions, beams, etc.):

```go
type BaseShape struct {
    Position   coords.PixelPosition
    Type       BasicShapeType  // Circular, Rectangular, Linear
    Size       int
    Width      int
    Height     int
    Direction  *ShapeDirection
    Quality    common.QualityType
}

// Register visual effect
graphics.VXHandler.AddVisualEffect(&graphics.VisualEffect{
    Shape:       baseShape,
    Color:       color.RGBA{255, 0, 0, 255},
    Duration:    time.Millisecond * 500,
    FadeOut:     true,
})

// Effects update and render automatically each frame
graphics.VXHandler.UpdateVisualEffects()
graphics.VXHandler.DrawVisualEffects(screen)
```

#### Integration Points

- **ECS Manager**: Queries entities with RenderableComponent
- **Position System**: Uses entity positions for sprite placement
- **Coordinate System**: Converts logical → pixel positions
- **Graphics System**: Renders visual effects on top of entities
- **World Map**: Renders tiles before entities

#### Key Files

- `rendering/rendering.go` - Core rendering system
- `graphics/drawableshapes.go` (390 LOC) - Visual effects
- `graphics/graphictypes.go` - Screen configuration
- `graphics/visualeffects.go` - Visual effect manager

---

### 3.7 World Map System

#### Purpose

Generates procedural dungeon maps using pluggable generation algorithms. Supports multiple generators via strategy pattern.

#### Architecture

```
┌─────────────────────┐
│ MapGenerator        │ (interface)
│ ┌─────────────────┐ │
│ │ GetName()       │ │
│ │ GetDescription()│ │
│ │ Generate()      │ │
│ └─────────────────┘ │
└──────────┬──────────┘
           │
     ┌─────┴─────┐
     │           │
┌────▼────┐ ┌───▼─────┐
│ Rooms & │ │ Tactical│
│Corridors│ │ Biome   │
│Generator│ │Generator│
└─────────┘ └─────────┘
```

**Available Generators:**

1. **Rooms and Corridors** (default)
   - Classic roguelike generation
   - Rectangular rooms connected by corridors
   - Guaranteed connectivity

2. **Tactical Biome**
   - Cellular automata for natural caves
   - 5 biomes (forest, desert, swamp, mountain, ruins)
   - Designed for squad-based tactical combat

#### How It Works

**Using a Generator:**
```go
// Default generator (rooms and corridors)
gameMap := worldmap.NewGameMapDefault()

// Specify generator
gameMap := worldmap.NewGameMap("tactical_biome")

// List available generators
generators := worldmap.ListGenerators()
// Returns: ["rooms_corridors", "tactical_biome"]
```

**Implementing a Generator:**
```go
type MyGenerator struct{}

func (g *MyGenerator) GetName() string {
    return "my_generator"
}

func (g *MyGenerator) GetDescription() string {
    return "My custom map generation algorithm"
}

func (g *MyGenerator) Generate(width, height int) *worldmap.GenerationResult {
    result := &worldmap.GenerationResult{
        Tiles:      make([]*worldmap.Tile, width*height),
        PlayerPos:  coords.LogicalPosition{X: width/2, Y: height/2},
        RoomCenters: []coords.LogicalPosition{},
    }

    // Generate map...
    for y := 0; y < height; y++ {
        for x := 0; x < width; x++ {
            logicalPos := coords.LogicalPosition{X: x, Y: y}

            // ⚠️ CRITICAL: Use CoordinateManager for indexing
            tileIdx := coords.CoordManager.LogicalToIndex(logicalPos)

            result.Tiles[tileIdx] = &worldmap.Tile{
                Blocked: false,
                // ... tile properties
            }
        }
    }

    return result
}

// Register generator at init time
func init() {
    worldmap.RegisterGenerator("my_generator", &MyGenerator{})
}
```

**Critical: CoordinateManager Usage**

When accessing `result.Tiles`, **ALWAYS** use `CoordinateManager` for indexing:

```go
// ✅ CORRECT: Use CoordinateManager
logicalPos := coords.LogicalPosition{X: x, Y: y}
tileIdx := coords.CoordManager.LogicalToIndex(logicalPos)
result.Tiles[tileIdx] = &tile

// ❌ WRONG: Manual calculation causes panics
idx := y*width + x  // This width may differ from CoordManager.dungeonWidth!
result.Tiles[idx] = &tile  // PANIC: index out of range
```

**Why?** The CoordinateManager's internal `dungeonWidth` may differ from the generator's `width` parameter. Manual calculation creates wrong indices (can even be negative!).

#### Integration Points

- **Game Initialization**: Map generated at startup
- **Spawning System**: Uses room centers for entity placement
- **Player System**: Uses player start position
- **Rendering System**: Renders tiles before entities
- **Position System**: Checks tile.Blocked for movement validation

#### Key Files

- `worldmap/generator.go` - MapGenerator interface + registry
- `worldmap/gen_rooms_corridors.go` - Classic roguelike generator
- `worldmap/gen_tactical_biome.go` - Cellular automata + biomes
- `worldmap/gen_helpers.go` - Shared helper functions
- `worldmap/GameMapUtil.go` - TileImageSet (no globals)

**Status:** 100% complete (strategy pattern implemented 2025-11-08)

---

### 3.8 GUI System

#### Purpose

Manages **modal UI interfaces** for inventory, squad management, formation editing, etc. Uses EbitenUI for widget rendering and event handling.

#### Architecture

```
┌───────────────────┐
│ UIModeManager     │
│ ┌───────────────┐ │
│ │ RegisterMode()│ │
│ │ SetMode()     │ │
│ │ Update()      │ │
│ │ Render()      │ │
│ └───────────────┘ │
└─────────┬─────────┘
          │
    ┌─────┴─────┬─────────┬──────────┬─────────┐
    │           │         │          │         │
┌───▼─────┐ ┌──▼───┐ ┌──▼────┐ ┌───▼───┐ ┌──▼────┐
│Inventory│ │Squad │ │Formation│ │Deploy │ │ Info  │
│  Mode   │ │Mgmt  │ │ Editor  │ │ Mode  │ │ Mode  │
│         │ │Mode  │ │  Mode   │ │       │ │       │
└─────────┘ └──────┘ └─────────┘ └───────┘ └───────┘
```

**UI Modes:**

1. **Exploration Mode** - Normal gameplay (not a UI mode, but default state)
2. **Inventory Mode** - View/use items
3. **Squad Management Mode** - Build squads, assign units
4. **Formation Editor Mode** - Edit 3x3 grid formations
5. **Squad Deployment Mode** - Place squads on world map
6. **Info Mode** - View help/documentation

#### How It Works

**Creating a UI Mode:**
```go
type MyMode struct {
    context *UIContext
    // Mode-specific state
    selectedIndex int
}

// Implement UIMode interface
func (m *MyMode) GetModeName() string {
    return "MyMode"
}

func (m *MyMode) Initialize(ctx *UIContext) error {
    m.context = ctx

    // Build UI widgets using EbitenUI
    m.container = widget.NewContainer(
        widget.ContainerOpts.Layout(widget.NewGridLayout(...)),
    )

    // Add buttons, lists, text, etc.
    return nil
}

func (m *MyMode) Enter(fromMode UIMode) error {
    // Setup when entering mode (e.g., load data)
    return nil
}

func (m *MyMode) Exit(toMode UIMode) error {
    // Cleanup when exiting mode
    return nil
}

func (m *MyMode) HandleInput(input *InputState) {
    // Process keyboard/mouse input
    if input.KeysJustPressed[ebiten.KeyEscape] {
        m.context.ModeManager.RequestTransition(explorationMode, "Back")
    }
}

func (m *MyMode) Update(deltaTime float64) error {
    // Update mode state (animations, timers, etc.)
    return nil
}

func (m *MyMode) Render(screen *ebiten.Image) {
    // EbitenUI handles rendering automatically
}
```

**Mode Transitions:**
```go
// Register mode
modeManager.RegisterMode(myMode)

// Set mode immediately
modeManager.SetMode("MyMode")

// Request transition (deferred until end of frame)
modeManager.RequestTransition(myMode, "Opening MyMode")
```

**UI Context:**

Each mode receives a UIContext with access to game state:

```go
type UIContext struct {
    ModeManager   *UIModeManager
    ECSManager    *common.EntityManager
    PlayerData    *common.PlayerData
    GameMap       *worldmap.GameMap
    PositionSys   *systems.PositionSystem
    // ... other systems
}
```

#### Integration Points

- **Input System**: UI modes are high-priority input controllers
- **Inventory System**: Inventory mode displays and manipulates items
- **Squad System**: Squad management mode edits squads and formations
- **ECS Manager**: UI modes query entities for display
- **Rendering System**: UI renders on top of game world

#### Key Files

- `gui/uimodemanager.go` - Mode registration and transitions
- `gui/inventorymode.go` - Inventory UI
- `gui/squadmanagementmode.go` - Squad builder UI
- `gui/formationeditormode.go` - Formation grid editor
- `gui/squaddeploymentmode.go` - Squad placement on map
- `gui/infomode.go` - Help/documentation viewer

---

## 4. Package Guide

This section provides a detailed look at each package, its purpose, key files, and how it fits into the overall architecture.

### 4.1 `game_main/` - Entry Point & Initialization

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

---

### 4.2 `common/` - Core ECS Utilities

**Purpose:** Shared ECS utilities, common components, global state.

**Key Files:**
- `ecsutil.go` (149 LOC) - Component access helpers
- `commoncomponents.go` - Name, Attributes, UserMessage components
- `globals.go` - GlobalPositionSystem singleton
- `playertypes.go` - PlayerData struct
- `attributetypes.go` - Combat-related types

**Responsibilities:**
- Define common components used across systems
- Provide type-safe component access functions
- Maintain global singletons (Position System)
- Define core game types (Attributes, PlayerData)

**Dependencies:**
- `coords/` - For LogicalPosition
- `github.com/bytearena/ecs` - ECS framework

**Public API:**
```go
// Component access
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T
func GetAttributes(e *ecs.Entity) *Attributes
func GetPosition(e *ecs.Entity) *coords.LogicalPosition

// Entity queries
func GetCreatureAtPosition(manager *EntityManager, pos *coords.LogicalPosition) *ecs.Entity
func DistanceBetween(e1, e2 *ecs.Entity) int

// Global state
var GlobalPositionSystem *systems.PositionSystem
```

---

### 4.3 `coords/` - Coordinate System

**Purpose:** Type-safe coordinate transformations between logical, pixel, and index spaces.

**Key Files:**
- `coordinatemanager.go` - CoordinateManager singleton
- `logicalposition.go` - Tile-based coordinates
- `pixelposition.go` - Screen pixel coordinates

**Responsibilities:**
- Define LogicalPosition (tile-based) and PixelPosition (pixel-based) types
- Provide conversions between coordinate spaces
- Maintain dungeon dimensions for index calculations
- Distance calculations (Manhattan, Chebyshev)

**Dependencies:** None (pure math)

**Public API:**
```go
type LogicalPosition struct { X, Y int }
type PixelPosition struct { X, Y int }

type CoordinateManager struct {
    tileSize      int
    dungeonWidth  int
    dungeonHeight int
}

func (cm *CoordinateManager) LogicalToPixel(pos LogicalPosition) PixelPosition
func (cm *CoordinateManager) PixelToLogical(pos PixelPosition) LogicalPosition
func (cm *CoordinateManager) LogicalToIndex(pos LogicalPosition) int
func (cm *CoordinateManager) IndexToLogical(index int) LogicalPosition

func (lp LogicalPosition) ManhattanDistance(other *LogicalPosition) int
func (lp LogicalPosition) ChebyshevDistance(other *LogicalPosition) int

// Global singleton
var CoordManager *CoordinateManager
```

---

### 4.4 `systems/` - ECS Systems

**Purpose:** ECS system implementations (position, movement, etc.).

**Key Files:**
- `positionsystem.go` (399 LOC) - O(1) spatial lookups
- `movementsystem.go` - Entity movement logic
- `turnmanager.go` - Turn-based game state machine

**Responsibilities:**
- Implement core game systems
- Maintain spatial grid for position queries
- Handle turn-based state transitions
- Provide system APIs for other packages

**Dependencies:**
- `common/` - ECS utilities
- `coords/` - Position types
- `github.com/bytearena/ecs`

**Public API:**
```go
// Position System
type PositionSystem struct { /* ... */ }
func NewPositionSystem(manager *ecs.Manager) *PositionSystem
func (ps *PositionSystem) GetEntityIDAt(pos coords.LogicalPosition) ecs.EntityID
func (ps *PositionSystem) AddEntity(entityID ecs.EntityID, pos coords.LogicalPosition)
func (ps *PositionSystem) MoveEntity(entityID ecs.EntityID, old, new coords.LogicalPosition)
func (ps *PositionSystem) GetEntitiesInRadius(center coords.LogicalPosition, radius int) []ecs.EntityID
```

---

### 4.5 `squads/` - Squad Combat System

**Purpose:** Tactical squad-based combat with formations, abilities, and multi-cell units.

**Key Files:**
- `components.go` (331 LOC) - 8 pure-data components
- `squadqueries.go` (140 LOC) - Query functions
- `squadcombat.go` (387 LOC) - Combat execution
- `squadabilities.go` (317 LOC) - Ability system
- `visualization.go` (175 LOC) - 3x3 grid rendering
- `squadformations.go` - Formation presets (in progress)

**Responsibilities:**
- Define squad and unit components
- Provide query functions for finding units/squads
- Execute squad vs squad combat
- Process leader abilities
- Render 3x3 formation grids
- Calculate squad capacity and movement

**Dependencies:**
- `common/` - ECS utilities, Attributes
- `coords/` - Positions
- `github.com/bytearena/ecs`

**Public API:**
```go
// Query functions
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID
func GetUnitIDsInRow(squadID ecs.EntityID, row int, manager *common.EntityManager) []ecs.EntityID
func GetLeaderID(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID
func IsSquadDestroyed(squadID ecs.EntityID, manager *common.EntityManager) bool

// Combat
func ExecuteSquadAttack(attackerID, defenderID ecs.EntityID, manager *common.EntityManager) *CombatResult

// Capacity
func GetSquadUsedCapacity(squadID ecs.EntityID, manager *common.EntityManager) float64
func GetSquadTotalCapacity(squadID ecs.EntityID, manager *common.EntityManager) int
func CanAddUnitToSquad(squadID ecs.EntityID, unitCost float64, manager *common.EntityManager) bool
```

**Status:** 95% complete (formation presets remaining, ~4-6 hours)

---

### 4.6 `gear/` - Inventory & Items

**Purpose:** Item management with pure ECS design.

**Key Files:**
- `Inventory.go` (241 LOC) - Pure component + system functions
- `items.go` (177 LOC) - Item component + actions
- `itemactions.go` - ItemAction interface
- `gearutil.go` (115 LOC) - Query helpers

**Responsibilities:**
- Define Inventory and Item components (pure data)
- Provide system functions for inventory manipulation
- Implement item actions (throwable, consumable)
- Query-based item lookup

**Dependencies:**
- `common/` - ECS utilities
- `coords/` - Positions (for ground items)
- `github.com/bytearena/ecs`

**Public API:**
```go
// System functions
func AddItem(manager *ecs.Manager, inv *Inventory, itemEntityID ecs.EntityID)
func RemoveItem(manager *ecs.Manager, inv *Inventory, index int)
func GetItemEntityID(inv *Inventory, index int) (ecs.EntityID, error)
func GetInventoryForDisplay(manager *ecs.Manager, inv *Inventory, indicesToSelect []int) []any

// Filtering
func GetThrowableItems(manager *ecs.Manager, inv *Inventory, selected []int) []any
func GetInventoryByAction(manager *ecs.Manager, inv *Inventory, selected []int, action string) []any
func HasItemsWithAction(manager *ecs.Manager, inv *Inventory, action string) bool

// Item lookup
func FindItemEntityByID(manager *ecs.Manager, itemID ecs.EntityID) *ecs.Entity
func GetItemByID(manager *ecs.Manager, itemID ecs.EntityID) *Item
```

**Status:** 100% complete (ECS refactor completed 2025-10-21)

---

### 4.7 `input/` - Input Handling

**Purpose:** Priority-based input routing with specialized controllers.

**Key Files:**
- `inputcoordinator.go` - Priority-based routing
- `movementcontroller.go` - Player movement
- `combatcontroller.go` - Combat targeting
- `debuginput.go` - Debug commands (F1-F12)

**Responsibilities:**
- Route input to correct controller based on priority
- Implement movement, combat, and UI input handling
- Provide debug commands for development
- Block lower-priority input when higher-priority active

**Dependencies:**
- `common/` - PlayerData
- `worldmap/` - GameMap (for collision)
- `github.com/hajimehoshi/ebiten/v2`

**Public API:**
```go
type InputController interface {
    CanHandle() bool
    HandleInput() bool
    GetPriority() int
}

type InputCoordinator struct { /* ... */ }
func NewInputCoordinator() *InputCoordinator
func (ic *InputCoordinator) RegisterController(controller InputController)
func (ic *InputCoordinator) HandleInput()
```

---

### 4.8 `rendering/` - Sprite Rendering

**Purpose:** Render entities with RenderableComponent to screen.

**Key Files:**
- `rendering.go` - Core rendering system
- `renderabletypes.go` - Renderable component

**Responsibilities:**
- Query entities with RenderableComponent
- Convert logical positions to pixel positions
- Render sprites at correct screen locations
- Handle viewport scrolling (only render visible area)

**Dependencies:**
- `common/` - ECS utilities
- `coords/` - Position transformations
- `graphics/` - Screen info
- `github.com/hajimehoshi/ebiten/v2`

**Public API:**
```go
type Renderable struct {
    Image   *ebiten.Image
    Visible bool
}

func ProcessRenderablesInSquare(
    screen *ebiten.Image,
    manager *ecs.Manager,
    renderableTag ecs.Tag,
    centerPos coords.LogicalPosition,
)
```

---

### 4.9 `graphics/` - Visual Effects

**Purpose:** Temporary visual effects (explosions, beams, etc.) and screen utilities.

**Key Files:**
- `drawableshapes.go` (390 LOC) - BaseShape + 3 variants
- `visualeffects.go` - VisualEffectHandler
- `graphictypes.go` - Screen configuration

**Responsibilities:**
- Define shape types (Circular, Rectangular, Linear)
- Manage visual effect lifecycle (create, update, render, expire)
- Provide screen dimension constants
- Handle fade-out animations

**Dependencies:**
- `coords/` - PixelPosition
- `common/` - QualityType
- `github.com/hajimehoshi/ebiten/v2`

**Public API:**
```go
type BaseShape struct {
    Position  coords.PixelPosition
    Type      BasicShapeType
    Size      int
    Width     int
    Height    int
    Direction *ShapeDirection
    Quality   common.QualityType
}

type VisualEffectHandler struct { /* ... */ }
func (vx *VisualEffectHandler) AddVisualEffect(effect *VisualEffect)
func (vx *VisualEffectHandler) UpdateVisualEffects()
func (vx *VisualEffectHandler) DrawVisualEffects(screen *ebiten.Image)

// Global singleton
var VXHandler *VisualEffectHandler
```

---

### 4.10 `gui/` - UI Modes

**Purpose:** Modal UI interfaces using EbitenUI.

**Key Files:**
- `uimodemanager.go` - Mode management
- `inventorymode.go` - Inventory UI
- `squadmanagementmode.go` - Squad builder
- `formationeditormode.go` - 3x3 grid editor
- `squaddeploymentmode.go` - Squad placement
- `infomode.go` - Help viewer
- `explorationmode.go` - Default mode

**Responsibilities:**
- Register and manage UI modes
- Handle mode transitions
- Provide UIContext to modes
- Route input to active mode
- Render UI overlays

**Dependencies:**
- `common/` - ECS utilities, PlayerData
- `gear/` - Inventory system
- `squads/` - Squad system
- `github.com/ebitenui/ebitenui`

**Public API:**
```go
type UIMode interface {
    GetModeName() string
    Initialize(ctx *UIContext) error
    Enter(fromMode UIMode) error
    Exit(toMode UIMode) error
    HandleInput(input *InputState)
    Update(deltaTime float64) error
    Render(screen *ebiten.Image)
}

type UIModeManager struct { /* ... */ }
func NewUIModeManager(ctx *UIContext) *UIModeManager
func (umm *UIModeManager) RegisterMode(mode UIMode)
func (umm *UIModeManager) SetMode(modeName string)
func (umm *UIModeManager) RequestTransition(mode UIMode, reason string)
```

---

### 4.11 `worldmap/` - Map Generation

**Purpose:** Procedural dungeon generation with strategy pattern.

**Key Files:**
- `generator.go` - MapGenerator interface + registry
- `gen_rooms_corridors.go` - Classic roguelike
- `gen_tactical_biome.go` - Cellular automata + biomes
- `gen_helpers.go` - Shared helpers
- `GameMapUtil.go` - TileImageSet (no globals)
- `GameMap.go` - GameMap struct + methods

**Responsibilities:**
- Define MapGenerator interface
- Implement multiple generation algorithms
- Register and retrieve generators
- Provide map data structures (Tile, GameMap)
- Handle tile rendering

**Dependencies:**
- `coords/` - LogicalPosition, CoordinateManager
- `common/` - ECS utilities
- `github.com/hajimehoshi/ebiten/v2`

**Public API:**
```go
type MapGenerator interface {
    GetName() string
    GetDescription() string
    Generate(width, height int) *GenerationResult
}

func RegisterGenerator(name string, gen MapGenerator)
func GetGenerator(name string) MapGenerator
func ListGenerators() []string

func NewGameMap(generatorName string) *GameMap
func NewGameMapDefault() *GameMap

type GameMap struct { /* ... */ }
func (gm *GameMap) IsBlocked(pos coords.LogicalPosition) bool
func (gm *GameMap) DrawLevelCenteredSquare(screen *ebiten.Image, ...)
```

**Status:** 100% complete (strategy pattern implemented 2025-11-08)

---

### 4.12 `entitytemplates/` - Entity Factories

**Purpose:** JSON-based entity creation with generic factory pattern.

**Key Files:**
- `creators.go` (283 LOC) - Generic CreateEntityFromTemplate
- `readdata.go` - JSON loading functions
- `jsonstructs.go` - JSON schema definitions

**Responsibilities:**
- Define JSON schemas for monsters, items, weapons
- Load JSON data at startup
- Provide generic factory for entity creation
- Support multiple entity types via EntityType enum

**Dependencies:**
- `common/` - ECS utilities, components
- `coords/` - LogicalPosition
- `gear/` - Item, Inventory components
- `rendering/` - Renderable component
- `github.com/hajimehoshi/ebiten/v2`

**Public API:**
```go
type EntityType int
const (
    Creature EntityType = iota
    Weapon
    Consumable
    Equipment
)

type EntityConfig struct {
    Type      EntityType
    Name      string
    ImagePath string
    Visible   bool
    Position  *coords.LogicalPosition
}

func ReadGameData()  // Load all JSON templates
func CreateEntityFromTemplate(
    manager *ecs.Manager,
    config EntityConfig,
    statusEffects map[string]ecs.EntityID,
) *ecs.Entity
```

**Status:** 100% complete (generic factory pattern)

---

### 4.13 `spawning/` - Entity Spawning

**Purpose:** Spawn entities at appropriate locations on the map.

**Key Files:**
- `spawning.go` - Spawn functions

**Responsibilities:**
- Find valid spawn positions (using PositionSystem)
- Create entities from templates
- Register entities in PositionSystem
- Spawn monsters, items, and equipment

**Dependencies:**
- `common/` - ECS utilities, PositionSystem
- `coords/` - LogicalPosition
- `entitytemplates/` - Entity creation
- `worldmap/` - GameMap

**Public API:**
```go
func SpawnMonsters(manager *ecs.Manager, gameMap *worldmap.GameMap, count int)
func SpawnItems(manager *ecs.Manager, gameMap *worldmap.GameMap, count int)
func FindEmptyPosition(gameMap *worldmap.GameMap) coords.LogicalPosition
```

---

### 4.14 `testing/` - Test Utilities

**Purpose:** Helper functions and utilities for testing.

**Key Files:**
- `testutils.go` - Test entity creation helpers

**Responsibilities:**
- Provide utilities for creating test entities
- Setup test environments
- Mock ECS components
- Assert functions for tests

**Dependencies:**
- `common/` - ECS utilities
- `github.com/bytearena/ecs`

**Public API:**
```go
func CreateTestEntity(manager *ecs.Manager) *ecs.Entity
func CreateTestSquad(manager *ecs.Manager) (*ecs.Entity, []ecs.EntityID)
// ... test helpers
```

---

## 5. Data Flow & Integration

This section explains how data moves through the system and how different systems integrate with each other.

### 5.1 Game Initialization Flow

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

### 5.2 Game Loop Flow

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

### 5.3 Entity Creation Flow

```
JSON File
    ↓
ReadGameData()
    ↓
Template Arrays
    ├─ MonsterTemplates []JSONMonster
    ├─ WeaponTemplates []JSONWeapon
    └─ ConsumableTemplates []JSONConsumable
    ↓
CreateEntityFromTemplate(config)
    ↓
1. Find template by name
    ↓
2. Load image
    ↓
3. Create entity
   entity := manager.NewEntity()
    ↓
4. Add components based on EntityType
   ├─ Common: Name, Position, Renderable
   ├─ Creature: Attributes
   ├─ Item: Item, ItemAction
   └─ Weapon: Attributes, Item
    ↓
5. Return entity
    ↓
Caller registers in PositionSystem
   GlobalPositionSystem.AddEntity(entityID, pos)
```

**Example:**

```go
// 1. Load templates at startup
entitytemplates.ReadGameData()

// 2. Create entity from template
config := entitytemplates.EntityConfig{
    Type:      entitytemplates.Creature,
    Name:      "Goblin",  // Template name from JSON
    ImagePath: "../assets/creatures/",
    Visible:   true,
    Position:  &coords.LogicalPosition{X: 10, Y: 5},
}
goblin := entitytemplates.CreateEntityFromTemplate(manager, config, nil)

// 3. Register in position system
common.GlobalPositionSystem.AddEntity(goblin.GetID(), *config.Position)
```

### 5.4 Combat Flow

```
Player initiates attack
    ↓
1. Find target position
   targetPos := playerPos.Translate(direction)
    ↓
2. Query PositionSystem (O(1))
   targetID := GlobalPositionSystem.GetEntityIDAt(targetPos)
    ↓
3. Find target entity
   target := manager.FindEntity(targetID)
    ↓
4. Get combat stats
   attackerAttr := GetAttributes(playerEntity)
   defenderAttr := GetAttributes(target)
    ↓
5. Calculate hit chance
   hitRate := attackerAttr.GetHitRate()
   dodgeChance := defenderAttr.GetDodgeChance()
   finalHitChance := hitRate - dodgeChance
    ↓
6. Roll hit/miss
   if rand.Intn(100) < finalHitChance {
       // Hit!
    ↓
7. Calculate damage
   damage := attackerAttr.GetPhysicalDamage()
   resistance := defenderAttr.GetPhysicalResistance()
   finalDamage := max(1, damage - resistance)
    ↓
8. Roll critical hit
   if rand.Intn(100) < attackerAttr.GetCritChance() {
       finalDamage *= 2  // Critical!
   }
    ↓
9. Apply damage
   defenderAttr.CurrentHealth -= finalDamage
    ↓
10. Check death
    if defenderAttr.CurrentHealth <= 0 {
        // Entity died
        ├─ RemoveEntity from PositionSystem
        ├─ DeleteEntity from manager
        └─ Spawn death visual effect
    }
```

### 5.5 Squad Combat Flow

```
ExecuteSquadAttack(attackerSquadID, defenderSquadID)
    ↓
1. Query all units in attacker squad
   attackerUnits := GetUnitIDsInSquad(attackerSquadID)
    ↓
2. Filter to alive units
   for each unit:
       if attr.CurrentHealth > 0 {
           aliveUnits.append(unitID)
       }
    ↓
3. For each alive attacker unit:
    ↓
    3a. Get targeting data
        targetRowData := GetComponent[*TargetRowData](unit)
        targetRows := targetRowData.TargetRows  // e.g., [0] (front row)
    ↓
    3b. Find targets in defender squad
        targets := []
        for each targetRow in targetRows:
            unitsInRow := GetUnitIDsInRow(defenderSquadID, targetRow)
            targets.append(unitsInRow)
    ↓
    3c. Select specific target
        if len(targets) > 0 {
            target := targets[rand.Intn(len(targets))]
        }
    ↓
    3d. Calculate hit/damage (same as individual combat)
        ├─ Hit chance vs dodge
        ├─ Damage vs resistance
        ├─ Critical hit roll
        └─ Cover reduction (if target behind another unit)
    ↓
    3e. Apply damage to target
        targetAttr.CurrentHealth -= finalDamage
    ↓
    3f. Record results
        result.DamageByUnit[targetID] += finalDamage
        if targetAttr.CurrentHealth <= 0 {
            result.UnitsKilled.append(targetID)
        }
    ↓
4. Return combat result
   return &CombatResult{
       TotalDamage: totalDamage,
       DamageByUnit: damageMap,
       UnitsKilled: killedUnits,
       CriticalHits: critCount,
       Dodged: dodgeCount,
   }
```

### 5.6 Inventory Flow

```
Player opens inventory
    ↓
1. UI mode activated
   modeManager.SetMode("Inventory")
    ↓
2. Inventory mode queries player inventory
   playerEntity := GetPlayerEntity()
   inv := GetComponent[*gear.Inventory](playerEntity)
    ↓
3. Build display list (system function)
   items := gear.GetInventoryForDisplay(manager, inv, nil)
    ↓
4. For each item entity ID in inventory:
    ↓
    4a. Find item entity
        itemEntity := gear.FindItemEntityByID(manager, itemID)
    ↓
    4b. Get item data
        name := GetComponent[*Name](itemEntity)
        item := GetComponent[*Item](itemEntity)
    ↓
    4c. Build display entry
        entry := InventoryListEntry{
            Index:    i,
            Name:     name.NameStr,
            Count:    item.Count,
            EntityID: itemID,
        }
    ↓
    4d. Append to display list
        displayList.append(entry)
    ↓
5. Render UI list
   EbitenUI renders list widgets
    ↓
6. Player selects item → Use
    ↓
7. Get item actions
   item := gear.GetItemByID(manager, selectedItemID)
   if item.HasAction("consumable") {
       action := item.GetAction("consumable")
       action.Execute(playerEntity, manager)
       gear.RemoveItem(manager, inv, selectedIndex)
   }
```

### 5.7 Map Generation Flow

```
Game initialization
    ↓
1. Get generator
   generator := worldmap.GetGenerator("rooms_corridors")
    ↓
2. Execute generation
   result := generator.Generate(80, 50)
    ↓
3. Generator creates tiles
   for y := 0; y < height; y++ {
       for x := 0; x < width; x++ {
           logicalPos := LogicalPosition{X: x, Y: y}

           // ⚠️ Use CoordinateManager for indexing
           tileIdx := coords.CoordManager.LogicalToIndex(logicalPos)

           result.Tiles[tileIdx] = &Tile{
               Blocked: isWall(x, y),
               // ... tile properties
           }
       }
   }
    ↓
4. Return generation result
   return &GenerationResult{
       Tiles:       tiles,
       PlayerPos:   startPos,
       RoomCenters: roomCenters,
   }
    ↓
5. Create GameMap from result
   gameMap := NewGameMapFromResult(result)
    ↓
6. Spawn entities at room centers
   for _, center := range result.RoomCenters {
       SpawnMonsterAt(center)
   }
```

### 5.8 Position System Integration

The Position System is integrated throughout the codebase:

**Movement:**
```go
// Check if target position is occupied
newPos := playerPos.Translate(dx, dy)
occupantID := GlobalPositionSystem.GetEntityIDAt(newPos)
if occupantID == 0 && !gameMap.IsBlocked(newPos) {
    // Position is free, move player
    GlobalPositionSystem.MoveEntity(playerID, playerPos, newPos)
    playerPos = newPos
}
```

**Combat:**
```go
// Find target at position
targetPos := attackerPos.Translate(direction)
targetID := GlobalPositionSystem.GetEntityIDAt(targetPos)
if targetID != 0 {
    target := manager.FindEntity(targetID)
    // Execute combat
}
```

**Spawning:**
```go
// Find empty position for spawn
for attempts := 0; attempts < 100; attempts++ {
    pos := coords.LogicalPosition{
        X: rand.Intn(mapWidth),
        Y: rand.Intn(mapHeight),
    }

    // Check if position is free
    if GlobalPositionSystem.GetEntityIDAt(pos) == 0 && !gameMap.IsBlocked(pos) {
        // Spawn here
        monster := CreateEntityFromTemplate(...)
        GlobalPositionSystem.AddEntity(monster.GetID(), pos)
        break
    }
}
```

**AOE Abilities:**
```go
// Find all entities in radius
centerPos := abilityTargetPos
radius := 3
affectedEntities := GlobalPositionSystem.GetEntitiesInRadius(centerPos, radius)

for _, entityID := range affectedEntities {
    entity := manager.FindEntity(entityID)
    // Apply AOE effect
}
```

---

## 6. Development Guide

### 6.1 Setting Up Development Environment

**Prerequisites:**
- Go 1.18+ (for generics support)
- Git
- Code editor (VS Code recommended)

**Installation:**
```bash
# Clone repository
git clone <repository-url>
cd TinkerRogue

# Install dependencies
go mod tidy

# Verify installation
go version  # Should show Go 1.18+
```

**Building:**
```bash
# Build executable
go build -o game_main/game_main.exe game_main/*.go

# Run directly
go run game_main/*.go
```

**Testing:**
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./squads -v
go test ./systems -v

# Run with coverage
go test ./squads -cover
```

### 6.2 ECS Best Practices

**Rule 1: Components are Pure Data**

```go
// ✅ CORRECT
type MyComponent struct {
    Value int
    IsActive bool
}

// ❌ WRONG
type MyComponent struct {
    Value int
}
func (m *MyComponent) DoSomething() {  // NO! Logic in component
    m.Value++
}
```

**Rule 2: Use EntityID, Not Pointers**

```go
// ✅ CORRECT
type SquadMemberData struct {
    SquadID ecs.EntityID  // Native type
}

// ❌ WRONG
type SquadMemberData struct {
    Squad *ecs.Entity  // Pointer prevents serialization
}
```

**Rule 3: Query-Based Relationships**

```go
// ✅ CORRECT: Query-based discovery
func GetUnitsInSquad(squadID ecs.EntityID) []ecs.EntityID {
    var units []ecs.EntityID
    for _, result := range manager.Query(squadMemberTag) {
        member := GetComponent[*SquadMemberData](result.Entity)
        if member.SquadID == squadID {
            units = append(units, result.Entity.GetID())
        }
    }
    return units
}

// ❌ WRONG: Stored references
type SquadData struct {
    Units []ecs.EntityID  // Requires manual sync
}
```

**Rule 4: System Functions, Not Methods**

```go
// ✅ CORRECT
func AddItem(manager *ecs.Manager, inv *Inventory, itemID ecs.EntityID) {
    // System logic
}

// ❌ WRONG
func (inv *Inventory) AddItem(itemID ecs.EntityID) {  // NO!
    // Logic in component method
}
```

**Rule 5: Value-Based Map Keys**

```go
// ✅ CORRECT: Value keys (O(1) hash lookup)
spatialGrid := make(map[coords.LogicalPosition][]ecs.EntityID)

// ❌ WRONG: Pointer keys (O(n) comparison)
spatialGrid := make(map[*coords.LogicalPosition]*ecs.Entity)
```

### 6.3 Common Workflows

#### Adding a New Component

1. **Define component struct (pure data):**
```go
// File: common/newcomponent.go
type MyComponent struct {
    Value    int
    IsActive bool
}
```

2. **Register component (in componentinit.go):**
```go
var MyComponent *ecs.Component

func InitializeECS(manager *ecs.Manager, tags map[string]ecs.Tag) {
    common.MyComponent = manager.NewComponent()
}
```

3. **Create tag (if needed):**
```go
myTag := ecs.BuildTag(common.MyComponent, common.PositionComponent)
tags["myEntities"] = myTag
```

4. **Add to entities:**
```go
entity.AddComponent(common.MyComponent, &common.MyComponent{
    Value:    10,
    IsActive: true,
})
```

#### Creating a New Entity Type

1. **Define JSON schema (assets/gamedata/mydata.json):**
```json
{
    "entities": [
        {
            "name": "MyEntity",
            "value": 42,
            "imageFile": "myentity.png"
        }
    ]
}
```

2. **Create JSON struct (entitytemplates/jsonstructs.go):**
```go
type JSONMyEntity struct {
    Name      string `json:"name"`
    Value     int    `json:"value"`
    ImageFile string `json:"imageFile"`
}
```

3. **Add loading function (entitytemplates/readdata.go):**
```go
var MyEntityTemplates []JSONMyEntity

func ReadMyEntityData() {
    data, _ := os.ReadFile("../assets/gamedata/mydata.json")
    var wrapper struct {
        Entities []JSONMyEntity `json:"entities"`
    }
    json.Unmarshal(data, &wrapper)
    MyEntityTemplates = wrapper.Entities
}

// Add to ReadGameData()
func ReadGameData() {
    ReadMonsterData()
    ReadWeaponData()
    ReadMyEntityData()  // Add here
}
```

4. **Create factory (entitytemplates/creators.go):**
```go
func CreateMyEntity(manager *ecs.Manager, name string, pos coords.LogicalPosition) *ecs.Entity {
    // Find template
    var template *JSONMyEntity
    for _, t := range MyEntityTemplates {
        if t.Name == name {
            template = &t
            break
        }
    }

    // Load image
    img, _, _ := ebitenutil.NewImageFromFile("../assets/myentities/" + template.ImageFile)

    // Create entity
    entity := manager.NewEntity().
        AddComponent(common.NameComponent, &common.Name{NameStr: template.Name}).
        AddComponent(common.PositionComponent, &pos).
        AddComponent(common.MyComponent, &common.MyComponent{Value: template.Value}).
        AddComponent(rendering.RenderableComponent, &rendering.Renderable{Image: img, Visible: true})

    return entity
}
```

#### Implementing a System

1. **Create system struct:**
```go
// File: systems/mysystem.go
type MySystem struct {
    manager *ecs.Manager
    cache   map[ecs.EntityID]int
}

func NewMySystem(manager *ecs.Manager) *MySystem {
    return &MySystem{
        manager: manager,
        cache:   make(map[ecs.EntityID]int),
    }
}
```

2. **Implement Update function:**
```go
func (ms *MySystem) Update(deltaTime float64) {
    for _, result := range ms.manager.Query(myTag) {
        entity := result.Entity
        myComp := common.GetComponentType[*MyComponent](entity)

        if myComp.IsActive {
            myComp.Value++
        }
    }
}
```

3. **Register in game loop:**
```go
// File: game_main/gameinit.go
g.mySystem = systems.NewMySystem(g.em.World)

// File: game_main/main.go
func (g *Game) Update() error {
    g.mySystem.Update(1.0 / 60.0)
    return nil
}
```

### 6.4 Debugging Tips

**Enable Debug Mode:**
```go
// File: game_main/config.go
const DEBUG_MODE = true
```

**Check Entity Components:**
```go
if entity.HasComponent(common.MyComponent) {
    fmt.Println("Entity has MyComponent")
} else {
    fmt.Println("Entity missing MyComponent")
}
```

**Verify Tag Queries:**
```go
results := manager.Query(tags["monsters"])
fmt.Printf("Found %d monsters\n", len(results))

if len(results) > 0 {
    entity := results[0].Entity
    attr := common.GetAttributes(entity)
    fmt.Printf("HP: %d/%d\n", attr.CurrentHealth, attr.MaxHealth)
}
```

**Debug Position System:**
```go
fmt.Printf("Entities tracked: %d\n", common.GlobalPositionSystem.GetEntityCount())

entityID := common.GlobalPositionSystem.GetEntityIDAt(pos)
if entityID != 0 {
    fmt.Printf("Entity %d at position (%d, %d)\n", entityID, pos.X, pos.Y)
}
```

**Add Logging:**
```go
func MoveEntity(entity *ecs.Entity, newPos coords.LogicalPosition) {
    oldPos := *common.GetPosition(entity)
    fmt.Printf("[MOVE] Entity %d: (%d, %d) → (%d, %d)\n",
        entity.GetID(), oldPos.X, oldPos.Y, newPos.X, newPos.Y)

    // Move logic...
}
```

### 6.5 Performance Optimization

**Use Position System for Spatial Queries:**
```go
// ✅ FAST: O(1) lookup
entityID := common.GlobalPositionSystem.GetEntityIDAt(pos)

// ❌ SLOW: O(n) search
for _, result := range manager.Query(allEntitiesTag) {
    if pos.IsEqual(common.GetPosition(result.Entity)) {
        // Found (but slow!)
    }
}
```

**Enable Viewport Rendering:**
```go
// Only render visible entities
graphics.MAP_SCROLLING_ENABLED = true
```

**Cache Frequently Accessed Data:**
```go
type CombatSystem struct {
    attributeCache map[ecs.EntityID]*common.Attributes
}

func (cs *CombatSystem) GetAttributes(entityID ecs.EntityID) *common.Attributes {
    if attr, ok := cs.attributeCache[entityID]; ok {
        return attr
    }
    // Cache miss, fetch and cache
    entity := cs.manager.FindEntity(entityID)
    attr := common.GetAttributes(entity)
    cs.attributeCache[entityID] = attr
    return attr
}
```

---

## 7. Reference

### 7.1 Build Commands

```bash
# Build executable
go build -o game_main/game_main.exe game_main/*.go

# Run directly
go run game_main/*.go

# Run tests
go test ./...

# Run specific package tests
go test ./squads -v

# Run with coverage
go test ./squads -cover

# Install dependencies
go mod tidy
```

### 7.2 Configuration

**game_main/config.go:**
```go
// Debug
const DEBUG_MODE = false
const ENABLE_BENCHMARKING = false

// Player stats
const DefaultPlayerStrength = 15
const DefaultPlayerDexterity = 20
const DefaultPlayerMagic = 0
const DefaultPlayerLeadership = 0
const DefaultPlayerArmor = 2
const DefaultPlayerWeapon = 3

// Asset paths
const PlayerImagePath = "../assets/creatures/player1.png"
const AssetItemsDir = "../assets/items/"
```

**graphics/graphictypes.go:**
```go
const DefaultTileSize = 32
const DefaultDungeonWidth = 80
const DefaultDungeonHeight = 50
const ViewableSquareSize = 25
const StatsUIOffset = 200

var MAP_SCROLLING_ENABLED = true
```

### 7.3 External Dependencies

```go
// Core dependencies
github.com/bytearena/ecs         // ECS framework
github.com/hajimehoshi/ebiten/v2 // Game engine
github.com/ebitenui/ebitenui     // UI toolkit
github.com/norendren/go-fov/fov  // Field of view

// Standard library
encoding/json
image
image/color
image/png
fmt
log
math
os
```

### 7.4 Project Status

**Completed Systems (100%):**
- ✅ ECS infrastructure
- ✅ Position System (O(1) lookups)
- ✅ Inventory System (ECS refactor complete)
- ✅ Input Coordination (priority-based)
- ✅ Graphics System (BaseShape consolidation)
- ✅ Entity Templates (generic factory)
- ✅ Coordinate System (type-safe)
- ✅ World Map Generation (strategy pattern)

**Squad System (95%):**
- ✅ 8 ECS components
- ✅ Query functions
- ✅ Combat system
- ✅ Ability system (auto-triggering)
- ✅ Visualization
- ⚠️ Formation presets (4-6h remaining)

**Total Lines of Code:** ~15,000+

---

## Document History

**Version 5.0** (2025-11-21) - Architectural Edition
- Complete rewrite focusing on systems, architecture, and interactions
- Added comprehensive architecture overview with diagrams
- Detailed system descriptions with integration points
- Package guide with responsibilities and dependencies
- Data flow and integration examples
- Development guide with best practices and workflows
- Removed excessive code listings in favor of explanatory text

**Version 4.0** (2025-10-22) - Enhanced Edition
- Component reference with all fields documented
- System function reference
- Practical code examples
- Developer workflows
- Troubleshooting section

**Version 3.0** (2025-10-12) - Codebase Audit
- Actual LOC counts from file analysis
- Squad system status update

**Version 2.0** (2025-10-21) - Inventory Refactor
- Documented inventory system ECS refactor

**Version 1.0** (2025-10-10) - Initial Documentation
- Basic component listings
- Initial architecture notes

---

**For Additional Information:**
- See `CLAUDE.md` for project configuration and build commands
- See `analysis/MASTER_ROADMAP.md` for development roadmap
- See `analysis/squad_system_final.md` for squad system architecture details
