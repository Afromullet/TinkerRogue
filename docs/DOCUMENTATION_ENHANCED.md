# TinkerRogue: Comprehensive Technical Documentation

**Game Type:** Turn-based Tactical Roguelike
**Engine:** Ebiten v2 (2D Game Engine for Go)
**Architecture:** Entity Component System (ECS) using bytearena/ecs
**Language:** Go 1.x
**Last Updated:** 2025-10-22 (Enhanced v4.0 - Complete Reference Edition)

---

## Quick Navigation

### For New Developers
- [Getting Started Guide](#getting-started-guide) - Setup and first run
- [Core Concepts](#core-concepts) - ECS, coordinate systems, data flow
- [Common Workflows](#developer-workflows) - Adding features, debugging, testing

### For System Understanding
- [Architecture Overview](#architecture-overview) - High-level system design
- [Component Reference](#complete-component-reference) - All components documented
- [System Functions](#system-functions-reference) - Query and manipulation APIs

### For Implementation
- [ECS Best Practices](#ecs-best-practices-gold-standard) - Squad/Inventory patterns
- [Code Examples](#practical-code-examples) - Common operations
- [Anti-Patterns](#anti-patterns-to-avoid) - What not to do

---

## Table of Contents

### 1. [Executive Summary](#executive-summary)
   - [What is TinkerRogue?](#what-is-tinkerrogue)
   - [Design Philosophy](#design-philosophy)
   - [Key Technical Features](#key-technical-features)
   - [Architectural Evolution](#architectural-evolution-v30)
   - [Current Status](#current-status)

### 2. [Getting Started Guide](#getting-started-guide)
   - [Prerequisites](#prerequisites)
   - [Installation](#installation)
   - [Building and Running](#building-and-running)
   - [Project Structure](#project-structure)
   - [First Steps](#first-steps)

### 3. [Core Concepts](#core-concepts)
   - [Entity Component System (ECS)](#entity-component-system-ecs)
   - [Coordinate System](#coordinate-system)
   - [Game Loop](#game-loop)
   - [Data Flow](#data-flow)

### 4. [Architecture Overview](#architecture-overview)
   - [High-Level Architecture](#high-level-architecture)
   - [ECS Implementation Details](#ecs-implementation-details)
   - [Package Organization](#package-organization)
   - [Key Architectural Patterns](#key-architectural-patterns)

### 5. [Complete Component Reference](#complete-component-reference)
   - [Common Components](#common-components)
   - [Squad System Components](#squad-system-components)
   - [Gear System Components](#gear-system-components)
   - [Graphics Components](#graphics-components)
   - [Component Registration](#component-registration)

### 6. [System Functions Reference](#system-functions-reference)
   - [Position System](#position-system)
   - [Squad Query Functions](#squad-query-functions)
   - [Inventory System Functions](#inventory-system-functions)
   - [Entity Management Functions](#entity-management-functions)

### 7. [Core Game Systems](#core-game-systems)
   - [Game Loop & State Management](#game-loop--state-management)
   - [Input System](#input-system)
   - [Rendering System](#rendering-system)
   - [Combat System](#combat-system)
   - [Squad System](#squad-system)
   - [Inventory System](#inventory-system)
   - [UI System](#ui-system)
   - [World Map & Generation](#world-map--generation)

### 8. [ECS Best Practices (Gold Standard)](#ecs-best-practices-gold-standard)
   - [Pure Data Components](#pure-data-components)
   - [Native EntityID Usage](#native-entityid-usage)
   - [Query-Based Relationships](#query-based-relationships)
   - [System-Based Logic](#system-based-logic)
   - [Value Map Keys](#value-map-keys)
   - [Reference Implementations](#reference-implementations)

### 9. [Practical Code Examples](#practical-code-examples)
   - [Creating Entities](#creating-entities)
   - [Querying Entities](#querying-entities)
   - [Adding Components](#adding-components)
   - [Position Lookups](#position-lookups)
   - [Inventory Operations](#inventory-operations)
   - [Squad Operations](#squad-operations)

### 10. [Developer Workflows](#developer-workflows)
   - [Adding a New Component](#adding-a-new-component)
   - [Creating a New Entity Type](#creating-a-new-entity-type)
   - [Implementing a System](#implementing-a-system)
   - [Adding a UI Mode](#adding-a-ui-mode)
   - [Debugging ECS Issues](#debugging-ecs-issues)

### 11. [Anti-Patterns to Avoid](#anti-patterns-to-avoid)
   - [Entity Pointers in Maps](#entity-pointers-in-maps)
   - [Logic in Components](#logic-in-components)
   - [Direct Entity References](#direct-entity-references)
   - [Pointer Map Keys](#pointer-map-keys)

### 12. [Package Documentation](#package-documentation)
   - [Package Index](#package-index)
   - [Detailed Package Descriptions](#detailed-package-descriptions)

### 13. [Asset Pipeline](#asset-pipeline)
   - [Directory Structure](#asset-directory-structure)
   - [Loading Systems](#asset-loading)
   - [JSON Templates](#json-template-loading)

### 14. [Troubleshooting](#troubleshooting)
   - [Common Issues](#common-issues)
   - [Performance Problems](#performance-problems)
   - [Debugging Techniques](#debugging-techniques)

### 15. [Appendices](#appendices)
   - [Glossary](#glossary)
   - [File Statistics](#file-statistics)
   - [External Dependencies](#external-dependencies)
   - [Configuration Reference](#configuration-reference)

---

## Executive Summary

### What is TinkerRogue?

TinkerRogue is a **turn-based tactical roguelike** that combines classic dungeon-crawling gameplay with emergent squad-based tactical combat. Built on the Ebiten 2D game engine and powered by a pure Entity Component System architecture, the game represents an evolution from traditional roguelike mechanics toward a tactical experience inspired by Final Fantasy Tactics.

**Core Gameplay Vision:**
- Command multiple squads arranged in 3x3 tactical formations
- Role-based unit specialization (Tanks, DPS, Support)
- Automated leader abilities triggered by battlefield conditions
- Procedurally generated dungeons with permadeath consequences

### Design Philosophy

**Entity Component System First:**
Every design decision flows from ECS principles. Entities are lightweight handles, components are pure data, and systems are functions. This architecture enables composition over inheritance and makes extending gameplay trivial.

**Data-Driven Design:**
Monsters, weapons, consumables, and squad units are defined in JSON files and loaded at runtime. Game designers can modify entity properties without touching source code.

**Performance Through Architecture:**
The Position System achieves O(1) lookups using value-based map keys (50x improvement over pointer-based approach). The Squad System demonstrates how proper ECS design enables complex tactical combat without sacrificing performance.

### Key Technical Features

1. **Pure ECS Architecture** - Entities are handles, components are data, systems are logic
2. **Data-Driven Templates** - JSON-based entity definitions with generic factory pattern
3. **Type-Safe Coordinates** - LogicalPosition/PixelPosition with CoordinateManager
4. **Visual Effects System** - Unified BaseShape system (Circular/Rectangular/Linear)
5. **Input Coordination** - Priority-based controller pattern (UI → Combat → Movement)
6. **Squad Combat** - 85% complete (2358 LOC), 8 components, tactical formations
7. **Inventory System** - 100% ECS compliant (2025-10-21 refactor)
8. **Position System** - O(1) spatial grid lookups (50x faster than legacy tracker)

### Architectural Evolution (v3.0)

**Removed Systems** (replaced with superior ECS implementations):
- Individual creature entities → Squad-based units
- Weapon components with logic → Attributes system
- Position tracker (pointer keys) → Position System (value keys)
- Individual combat resolution → Squad-level tactical combat

**Current Gold Standard:**
The squad and inventory systems demonstrate perfect ECS patterns:
- Pure data components (zero logic methods)
- Native `ecs.EntityID` usage (no pointers)
- Query-based relationships (discover via ECS)
- System-based logic (all behavior in functions)
- Value-based map keys (O(1) performance)

### Current Status

**Operational Systems** (100% Complete):
- ✅ ECS infrastructure with 30+ component types
- ✅ Position System (O(1) lookups, 399 LOC)
- ✅ Inventory System (ECS refactor, 533 LOC)
- ✅ Input Coordination (3 controllers, priority-based)
- ✅ Graphics System (BaseShape consolidation, 390 LOC)
- ✅ Entity Templates (generic factory, 283 LOC)
- ✅ Coordinate System (type-safe transformations)

**Squad System** (85% Complete, 2358 LOC):
- ✅ 8 ECS components (perfect data/logic separation)
- ✅ 7 query functions (GetUnitIDsInSquad, GetLeaderID, etc.)
- ✅ Combat system (ExecuteSquadAttack with hit/dodge/crit/cover)
- ✅ Visualization (3x3 grid rendering)
- ✅ Testing infrastructure (comprehensive test suite)
- ❌ Ability system (8-10h remaining)
- ⚠️ Formation presets (4-6h remaining)

**Remaining Work:** 12-16 hours for squad completion + map integration

---

## Getting Started Guide

### Prerequisites

**Required:**
- Go 1.18+ (for generics support)
- Git (for cloning repository)

**Recommended:**
- VS Code with Go extension
- Terminal with Git Bash (Windows)

### Installation

```bash
# Clone repository
git clone <repository-url>
cd TinkerRogue

# Install dependencies
go mod tidy

# Verify installation
go version  # Should show Go 1.18+
```

### Building and Running

```bash
# Build executable
go build -o game_main/game_main.exe game_main/*.go

# Run directly (no build step)
go run game_main/*.go

# Run tests
go test ./...

# Run specific package tests
go test ./squads -v
go test ./systems -v
```

**Build Output:**
`game_main/game_main.exe` (Windows)
Executable is placed in the `game_main` directory alongside source files.

### Project Structure

```
TinkerRogue/
├── assets/                  # Game resources
│   ├── creatures/          # Entity sprites (PNG)
│   ├── items/              # Item sprites (PNG)
│   ├── fonts/              # Font files
│   └── gamedata/           # JSON templates
│       ├── monsterdata.json
│       ├── weapondata.json
│       └── consumabledata.json
├── analysis/               # Design documents
│   ├── MASTER_ROADMAP.md  # Primary planning doc
│   └── squad_system_final.md
├── docs/                   # Documentation
│   └── DOCUMENTATION.md   # This file
├── common/                 # Core ECS utilities
├── coords/                 # Coordinate system
├── entitytemplates/        # Template factories
├── game_main/              # Main entry point
├── gear/                   # Inventory & items
├── graphics/               # Rendering & shapes
├── gui/                    # UI modes
├── input/                  # Input controllers
├── rendering/              # Sprite rendering
├── spawning/               # Entity spawning
├── squads/                 # Squad combat system
├── systems/                # ECS systems
├── testing/                # Test utilities
├── worldmap/               # Map generation
├── CLAUDE.md              # Project config
└── go.mod                 # Dependencies
```

### First Steps

**1. Read the Core Concepts:**
Understanding ECS, coordinate systems, and the game loop is essential before diving into code.

**2. Explore Reference Implementations:**
- `squads/components.go` - Perfect ECS component design
- `gear/Inventory.go` - System-based functions
- `systems/positionsystem.go` - O(1) spatial queries

**3. Run the Game:**
```bash
cd game_main
go run *.go
```

**4. Experiment with Debug Commands:**
- F1-F12: Various debug actions (see `input/debuginput.go`)
- Arrow Keys: Move player
- Mouse Click: Target position

**5. Modify a Template:**
Edit `assets/gamedata/monsterdata.json` to change monster stats, then reload the game to see changes.

---

## Core Concepts

### Entity Component System (ECS)

**Entities** are lightweight handles (`*ecs.Entity`) with unique IDs. An entity is nothing more than a container for components.

```go
// Creating an entity
entity := manager.NewEntity()
entityID := entity.GetID()  // Native EntityID (not a pointer)
```

**Components** are pure data structures with no behavior:

```go
// ✅ CORRECT: Pure data component
type Position struct {
    X, Y int
}

// ❌ WRONG: Component with logic method
type Position struct {
    X, Y int
}
func (p *Position) MoveTo(x, y int) {  // NO! Logic belongs in systems
    p.X, p.Y = x, y
}
```

**Systems** are functions that query entities and execute logic:

```go
// System function signature
func RenderSystem(manager *ecs.Manager) {
    for _, result := range manager.Query(renderableTag) {
        entity := result.Entity
        pos := GetComponent[*Position](entity, PositionComponent)
        renderable := GetComponent[*Renderable](entity, RenderableComponent)
        // Render entity at position
    }
}
```

**Why This Matters:**
Component composition replaces inheritance. Want a monster that can pick up items? Just add an Inventory component. No class hierarchy needed.

### Coordinate System

TinkerRogue uses **three coordinate spaces**:

1. **LogicalPosition** (tile-based):
```go
type LogicalPosition struct { X, Y int }
// Example: LogicalPosition{X: 10, Y: 5} = 10th tile right, 5th down
```

2. **PixelPosition** (screen rendering):
```go
type PixelPosition struct { X, Y int }
// Example: PixelPosition{X: 320, Y: 160} = screen location
```

3. **Index** (1D array mapping):
```go
index := coords.CoordManager.LogicalToIndex(logicalPos)
// Converts (x, y) to array index: y * mapWidth + x
```

**Transformations:**
```go
// LogicalPosition → PixelPosition (for rendering)
pixelPos := coords.CoordManager.LogicalToPixel(logicalPos)

// PixelPosition → LogicalPosition (for mouse input)
logicalPos := coords.CoordManager.PixelToLogical(pixelPos)

// LogicalPosition → Index (for map array access)
index := coords.CoordManager.LogicalToIndex(logicalPos)

// Index → LogicalPosition (for reverse lookup)
logicalPos := coords.CoordManager.IndexToLogical(index)
```

**Why Three Systems?**
Game logic operates in tiles, rendering needs pixels, maps store data in 1D arrays. The CoordinateManager singleton handles all conversions.

### Game Loop

The game runs at ~60 FPS with separated `Update()` and `Draw()` phases:

```go
// Update() - Process input and game logic
func (g *Game) Update() error {
    // 1. UI updates (EbitenUI widgets)
    g.uiModeManager.Update(deltaTime)

    // 2. Visual effects (animations)
    graphics.VXHandler.UpdateVisualEffects()

    // 3. Debug input (F1-F12)
    input.PlayerDebugActions(&g.playerData)

    // 4. Main input handling
    HandleInput(g)  // Only advances state when player acts

    return nil
}

// Draw() - Render current state
func (g *Game) Draw(screen *ebiten.Image) {
    // 1. Map rendering
    g.gameMap.DrawLevelCenteredSquare(screen, ...)

    // 2. Entity rendering
    rendering.ProcessRenderablesInSquare(...)

    // 3. Visual effects
    graphics.VXHandler.DrawVisualEffects(screen)

    // 4. UI overlay
    g.uiModeManager.Render(screen)
}
```

**Turn-Based Implementation:**
The game is turn-based but runs in real-time. `Update()` only advances game state when the player presses an action key. Between actions, animations and UI continue updating but no game logic executes.

### Data Flow

```
┌─────────────┐
│ JSON Files  │ (monsterdata.json, weapondata.json)
└──────┬──────┘
       │ ReadGameData()
       ▼
┌─────────────────┐
│ Template System │ (entitytemplates/)
└──────┬──────────┘
       │ CreateEntityFromTemplate()
       ▼
┌─────────────────┐
│ ECS Manager     │ (Entities + Components)
└──────┬──────────┘
       │
       ├─► Position System (O(1) lookups)
       ├─► Squad System (query functions)
       ├─► Inventory System (system functions)
       └─► Rendering System (draw entities)
```

**Flow Explanation:**
1. Game starts → Load JSON templates
2. Player enters dungeon → Spawn entities from templates
3. Every frame → Query entities, execute systems
4. Player acts → Update components, trigger systems
5. Draw phase → Render based on component data

---

## Complete Component Reference

### Common Components

#### Name
**File:** `common/commoncomponents.go:7-9`
**Purpose:** Display name for entities
```go
type Name struct {
    NameStr string
}
```
**Usage:** Combat logs, inventory display, UI labels
**Registration:** `common.NameComponent`

#### Attributes
**File:** `common/commoncomponents.go:17-38`
**Purpose:** Core stats and combat capabilities
```go
type Attributes struct {
    // Core Attributes
    Strength   int  // Physical Damage, Physical Resistance, Max HP
    Dexterity  int  // Hit Rate, Crit Chance, Dodge
    Magic      int  // Magic Damage, Healing Amount, Magic Defense
    Leadership int  // Unit Capacity (squad size)
    Armor      int  // Damage Reduction Modifier
    Weapon     int  // Damage Increase Modifier

    // Runtime State
    CurrentHealth int
    MaxHealth     int
    CanAct        bool
}
```
**Derived Stats:** (calculated methods, not stored)
- `GetPhysicalDamage()` = (Strength / 2) + (Weapon * 2)
- `GetPhysicalResistance()` = (Strength / 4) + (Armor * 2)
- `GetMaxHealth()` = 20 + (Strength * 2)
- `GetHitRate()` = 80 + (Dexterity * 2), capped at 100
- `GetCritChance()` = Dexterity / 2, capped at 50
- `GetDodgeChance()` = Dexterity / 3, capped at 40
- `GetMagicDamage()` = Magic * 3
- `GetHealingAmount()` = Magic * 2
- `GetMagicDefense()` = Magic / 2
- `GetUnitCapacity()` = 6 + (Leadership / 3), capped at 9
- `GetCapacityCost()` = (Strength + Weapon + Armor) / 5.0

**Registration:** `common.AttributeComponent`

#### UserMessage
**File:** `common/commoncomponents.go:11-15`
**Purpose:** Display messages to player
```go
type UserMessage struct {
    AttackMessage       string
    GameStateMessage    string
    StatusEffectMessage string
}
```
**Usage:** Combat feedback, status updates
**Registration:** `common.UserMsgComponent`
**Note:** Candidate for removal (legacy)

### Squad System Components

#### SquadData
**File:** `squads/components.go:38-50`
**Purpose:** Squad entity's core data
```go
type SquadData struct {
    SquadID       ecs.EntityID  // ✅ Native EntityID
    Formation     FormationType // Balanced/Defensive/Offensive/Ranged
    Name          string
    Morale        int           // 0-100
    SquadLevel    int
    TurnCount     int
    MaxUnits      int           // Typically 9
    UsedCapacity  float64       // Current capacity consumed
    TotalCapacity int           // From leader's Leadership
}
```
**Formation Types:**
- `FormationBalanced` = Mix of roles
- `FormationDefensive` = Tank-heavy
- `FormationOffensive` = DPS-focused
- `FormationRanged` = Back-line heavy

**Registration:** `squads.SquadComponent`

#### SquadMemberData
**File:** `squads/components.go:82-85`
**Purpose:** Links unit to parent squad
```go
type SquadMemberData struct {
    SquadID ecs.EntityID  // ✅ Native EntityID (not pointer)
}
```
**Registration:** `squads.SquadMemberComponent`

#### GridPositionData
**File:** `squads/components.go:88-113`
**Purpose:** Unit position in 3x3 tactical grid
```go
type GridPositionData struct {
    AnchorRow int  // Top-left row (0-2)
    AnchorCol int  // Top-left col (0-2)
    Width     int  // Columns occupied (1-3)
    Height    int  // Rows occupied (1-3)
}
```
**Methods:**
- `GetOccupiedCells() [][2]int` - All cells this unit occupies
- `OccupiesCell(row, col int) bool` - Check if unit is at cell
- `GetRows() []int` - All row indices occupied

**Supports Multi-Cell Units:**
- 1x1: Single cell (default)
- 2x2: Large creature (giant, dragon)
- 1x3: Cavalry formation (horizontal)
- 3x1: Wall formation (vertical)

**Registration:** `squads.GridPositionComponent`

#### UnitRoleData
**File:** `squads/components.go:123-149`
**Purpose:** Combat role specialization
```go
type UnitRoleData struct {
    Role UnitRole
}

type UnitRole int
const (
    RoleTank    UnitRole = iota  // Takes hits first, high defense
    RoleDPS                      // High damage output
    RoleSupport                  // Buffs, heals, utility
)
```
**Registration:** `squads.UnitRoleComponent`

#### CoverData
**File:** `squads/components.go:152-166`
**Purpose:** Defensive cover mechanics
```go
type CoverData struct {
    CoverValue     float64  // Damage reduction (0.0-1.0, e.g., 0.25 = 25%)
    CoverRange     int      // Rows behind (1 = immediate, 2 = two rows)
    RequiresActive bool     // Dead units don't provide cover
}
```
**Method:**
- `GetCoverBonus(isActive bool) float64` - Returns cover if active

**Registration:** `squads.CoverComponent`

#### AttackRangeData
**File:** `squads/components.go:168-173`
**Purpose:** World-based attack range
```go
type AttackRangeData struct {
    Range int  // World tiles (Melee=1, Ranged=3, Magic=4)
}
```
**Registration:** `squads.AttackRangeComponent`

#### MovementSpeedData
**File:** `squads/components.go:175-178`
**Purpose:** World map movement
```go
type MovementSpeedData struct {
    Speed int  // Tiles per turn (typically 1-5)
}
```
**Note:** Squad speed = minimum of all unit speeds

**Registration:** `squads.MovementSpeedComponent`

#### TargetRowData
**File:** `squads/components.go:188-214`
**Purpose:** Combat targeting configuration
```go
type TargetRowData struct {
    Mode TargetMode  // RowBased or CellBased

    // Row-based targeting
    TargetRows    []int
    IsMultiTarget bool
    MaxTargets    int

    // Cell-based targeting
    TargetCells [][2]int  // Specific grid cells [[row, col], ...]
}
```
**Targeting Modes:**
- `TargetModeRowBased`: Target entire row(s)
- `TargetModeCellBased`: Target specific cells (for AOE patterns)

**Example Patterns:**
```go
// Front row targeting
TargetRowData{Mode: RowBased, TargetRows: []int{0}}

// All rows (AOE)
TargetRowData{Mode: RowBased, TargetRows: []int{0, 1, 2}}

// 2x2 quad pattern
TargetRowData{Mode: CellBased, TargetCells: [][2]int{{0,0}, {0,1}, {1,0}, {1,1}}}
```

**Registration:** `squads.TargetRowComponent`

#### LeaderData
**File:** `squads/components.go:220-224`
**Purpose:** Squad leader marker
```go
type LeaderData struct {
    Leadership int  // Bonus to squad stats
    Experience int  // Leader progression (future)
}
```
**Registration:** `squads.LeaderComponent`

#### AbilitySlotData
**File:** `squads/components.go:226-239`
**Purpose:** Equipped abilities (FFT-style)
```go
type AbilitySlotData struct {
    Slots [4]AbilitySlot  // 4 ability slots
}

type AbilitySlot struct {
    AbilityType  AbilityType
    TriggerType  TriggerType
    Threshold    float64
    HasTriggered bool
    IsEquipped   bool
}
```
**Ability Types:**
- `AbilityRally` - Strength bonus to squad
- `AbilityHeal` - HP restoration
- `AbilityBattleCry` - Morale + strength boost
- `AbilityFireball` - Direct damage

**Trigger Types:**
- `TriggerSquadHPBelow` - When HP < threshold
- `TriggerTurnCount` - Specific turn number
- `TriggerEnemyCount` - Number of enemy squads
- `TriggerMoraleBelow` - Morale < threshold
- `TriggerCombatStart` - First turn of combat

**Registration:** `squads.AbilitySlotComponent`

#### CooldownTrackerData
**File:** `squads/components.go:279-284`
**Purpose:** Ability cooldown tracking
```go
type CooldownTrackerData struct {
    Cooldowns    [4]int  // Turns remaining for slots 0-3
    MaxCooldowns [4]int  // Base cooldown durations
}
```
**Registration:** `squads.CooldownTrackerComponent`

### Gear System Components

#### Inventory
**File:** `gear/Inventory.go:20-22`
**Purpose:** Pure data inventory component
```go
type Inventory struct {
    ItemEntityIDs []ecs.EntityID  // ✅ ECS best practice: EntityID, not pointers
}
```
**⚠️ DO NOT USE DIRECTLY:** Use system functions instead:
```go
// ✅ CORRECT: Use system functions
gear.AddItem(manager, inv, itemEntityID)
gear.RemoveItem(manager, inv, index)
items := gear.GetInventoryForDisplay(manager, inv, nil)

// ❌ WRONG: Direct manipulation
inv.ItemEntityIDs = append(inv.ItemEntityIDs, itemID)  // NO!
```
**Registration:** `gear.InventoryComponent`

#### Item
**File:** `gear/items.go:42-46`
**Purpose:** Item component
```go
type Item struct {
    Properties ecs.EntityID  // ✅ Status effects entity ID (not pointer)
    Actions    []ItemAction
    Count      int
}
```
**Methods:** (minimal, prefer system functions)
- `GetAction(actionName string) ItemAction`
- `HasAction(actionName string) bool`

**Registration:** `gear.ItemComponent`

#### ItemAction Interface
**File:** `gear/itemactions.go`
**Purpose:** Defines item behavior (throwable, consumable, etc.)
```go
type ItemAction interface {
    ActionName() string
    Copy() ItemAction
    Execute(...)  // Action-specific behavior
}
```
**Implementations:**
- `ThrowableAction` - Can be thrown (AOE pattern)
- `ConsumableAction` - Can be consumed (status effects)

### Graphics Components

#### Renderable
**File:** `rendering/rendering.go`
**Purpose:** Visual representation
```go
type Renderable struct {
    Image   *ebiten.Image
    Visible bool
}
```
**Registration:** `rendering.RenderableComponent`

#### BaseShape
**File:** `graphics/drawableshapes.go:79-87`
**Purpose:** Visual effect shapes
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
```
**Shape Types:**
- `Circular` - Radius-based (explosions, AOE)
- `Rectangular` - Width/height (walls, zones)
- `Linear` - Direction/length (cones, beams)

**Not a component:** Used for visual effects, not attached to entities

### Component Registration

**Location:** `game_main/componentinit.go`

All components must be registered at startup:

```go
func InitializeECS(manager *ecs.Manager, tags map[string]ecs.Tag) {
    // Common components
    common.PositionComponent = manager.NewComponent()
    common.NameComponent = manager.NewComponent()
    common.AttributeComponent = manager.NewComponent()

    // Squad components
    squads.SquadComponent = manager.NewComponent()
    squads.SquadMemberComponent = manager.NewComponent()
    squads.GridPositionComponent = manager.NewComponent()
    squads.UnitRoleComponent = manager.NewComponent()
    squads.CoverComponent = manager.NewComponent()
    squads.LeaderComponent = manager.NewComponent()
    squads.TargetRowComponent = manager.NewComponent()
    squads.AbilitySlotComponent = manager.NewComponent()
    squads.CooldownTrackerComponent = manager.NewComponent()
    squads.AttackRangeComponent = manager.NewComponent()
    squads.MovementSpeedComponent = manager.NewComponent()

    // Gear components
    gear.ItemComponent = manager.NewComponent()
    gear.InventoryComponent = manager.NewComponent()

    // Graphics components
    rendering.RenderableComponent = manager.NewComponent()

    // Create tags
    tags["renderables"] = ecs.BuildTag(
        rendering.RenderableComponent,
        common.PositionComponent,
    )
    tags["monsters"] = ecs.BuildTag(
        common.AttributeComponent,
        common.PositionComponent,
    )
    squads.SquadTag = ecs.BuildTag(squads.SquadComponent)
    squads.SquadMemberTag = ecs.BuildTag(squads.SquadMemberComponent)
    squads.LeaderTag = ecs.BuildTag(
        squads.LeaderComponent,
        squads.SquadMemberComponent,
    )
}
```

**Why Registration Matters:**
The ECS library needs to know about each component type for efficient storage and querying. Registration creates component type identifiers used throughout the codebase.

---

## System Functions Reference

### Position System

**File:** `systems/positionsystem.go` (399 LOC)
**Purpose:** O(1) position-based entity lookup

**Performance:** 50x faster than legacy O(n) linear search

#### Core Functions

```go
// NewPositionSystem creates the system
func NewPositionSystem(manager *ecs.Manager) *PositionSystem

// GetEntityIDAt returns first entity at position (O(1))
func (ps *PositionSystem) GetEntityIDAt(pos coords.LogicalPosition) ecs.EntityID

// GetEntityAt returns entity pointer at position
func (ps *PositionSystem) GetEntityAt(pos coords.LogicalPosition) *ecs.Entity

// GetAllEntityIDsAt returns all entities at position (for stacking)
func (ps *PositionSystem) GetAllEntityIDsAt(pos coords.LogicalPosition) []ecs.EntityID

// AddEntity registers entity at position
func (ps *PositionSystem) AddEntity(entityID ecs.EntityID, pos coords.LogicalPosition) error

// RemoveEntity unregisters entity from position
func (ps *PositionSystem) RemoveEntity(entityID ecs.EntityID, pos coords.LogicalPosition) error

// MoveEntity updates entity position (more efficient than Remove + Add)
func (ps *PositionSystem) MoveEntity(entityID ecs.EntityID, oldPos, newPos coords.LogicalPosition) error

// GetEntitiesInRadius returns all entities within radius (Chebyshev distance)
func (ps *PositionSystem) GetEntitiesInRadius(center coords.LogicalPosition, radius int) []ecs.EntityID
```

#### Usage Example

```go
// Access global position system
posSystem := common.GlobalPositionSystem

// Check for entity at position
entityID := posSystem.GetEntityIDAt(coords.LogicalPosition{X: 10, Y: 5})
if entityID != 0 {
    // Entity found
}

// Add entity to position tracking
posSystem.AddEntity(newEntity.GetID(), spawnPos)

// Move entity
posSystem.MoveEntity(entityID, oldPos, newPos)

// Find entities in AOE radius
targets := posSystem.GetEntitiesInRadius(centerPos, 3)
```

### Squad Query Functions

**File:** `squads/squadqueries.go` (286 LOC)
**Purpose:** Query-based squad and unit lookup

#### Unit Queries

```go
// FindUnitByID finds unit entity by ID
func FindUnitByID(unitID ecs.EntityID, squadmanager *common.EntityManager) *ecs.Entity

// GetUnitIDsAtGridPosition returns units occupying grid cell
func GetUnitIDsAtGridPosition(squadID ecs.EntityID, row, col int, squadmanager *common.EntityManager) []ecs.EntityID

// GetUnitIDsInSquad returns all units in squad
func GetUnitIDsInSquad(squadID ecs.EntityID, squadmanager *common.EntityManager) []ecs.EntityID

// GetUnitIDsInRow returns alive units in specific row
func GetUnitIDsInRow(squadID ecs.EntityID, row int, squadmanager *common.EntityManager) []ecs.EntityID

// GetLeaderID finds squad leader unit ID
func GetLeaderID(squadID ecs.EntityID, squadmanager *common.EntityManager) ecs.EntityID
```

#### Squad Queries

```go
// GetSquadEntity finds squad entity by squad ID
func GetSquadEntity(squadID ecs.EntityID, squadmanager *common.EntityManager) *ecs.Entity

// IsSquadDestroyed checks if all units are dead
func IsSquadDestroyed(squadID ecs.EntityID, squadmanager *common.EntityManager) bool
```

#### Capacity System

```go
// GetSquadUsedCapacity calculates total capacity consumed
func GetSquadUsedCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) float64

// GetSquadTotalCapacity returns capacity from leader's Leadership
func GetSquadTotalCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) int

// GetSquadRemainingCapacity returns available capacity
func GetSquadRemainingCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) float64

// CanAddUnitToSquad checks if unit fits in squad
func CanAddUnitToSquad(squadID ecs.EntityID, unitCapacityCost float64, squadmanager *common.EntityManager) bool

// UpdateSquadCapacity recalculates cached capacity values
func UpdateSquadCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager)
```

#### Distance & Range

```go
// GetSquadDistance calculates Manhattan distance between squads
func GetSquadDistance(squad1ID, squad2ID ecs.EntityID, squadmanager *common.EntityManager) int

// GetSquadMovementSpeed returns minimum speed of alive units
func GetSquadMovementSpeed(squadID ecs.EntityID, squadmanager *common.EntityManager) int
```

#### Usage Example

```go
// Find all units in a squad
unitIDs := squads.GetUnitIDsInSquad(squadID, ecsManager)

// Check each unit's health
for _, unitID := range unitIDs {
    unit := squads.FindUnitByID(unitID, ecsManager)
    attr := common.GetAttributes(unit)
    if attr.CurrentHealth <= 0 {
        // Unit is dead
    }
}

// Find units in front row
frontRowUnits := squads.GetUnitIDsInRow(squadID, 0, ecsManager)

// Check if squad can add more units
leaderID := squads.GetLeaderID(squadID, ecsManager)
leader := squads.FindUnitByID(leaderID, ecsManager)
leaderAttr := common.GetAttributes(leader)
newUnitCost := leaderAttr.GetCapacityCost()

if squads.CanAddUnitToSquad(squadID, newUnitCost, ecsManager) {
    // Add unit
}
```

### Inventory System Functions

**File:** `gear/Inventory.go` (241 LOC)
**Purpose:** System-based inventory management

**⚠️ IMPORTANT:** Never manipulate Inventory component directly. Always use system functions.

#### Core Operations

```go
// AddItem adds item to inventory (increments count if exists)
func AddItem(manager *ecs.Manager, inv *Inventory, itemEntityID ecs.EntityID)

// RemoveItem decrements count or removes item
func RemoveItem(manager *ecs.Manager, inv *Inventory, index int)

// GetItemEntityID returns entity ID at index
func GetItemEntityID(inv *Inventory, index int) (ecs.EntityID, error)
```

#### Display Functions

```go
// GetInventoryForDisplay builds list for UI rendering
func GetInventoryForDisplay(manager *ecs.Manager, inv *Inventory, indicesToSelect []int, itemPropertiesFilter ...StatusEffects) []any

// GetEffectNames returns status effect names for item
func GetEffectNames(manager *ecs.Manager, inv *Inventory, index int) ([]string, error)
```

#### Action Filtering

```go
// GetInventoryByAction filters items by action capability
func GetInventoryByAction(manager *ecs.Manager, inv *Inventory, indicesToSelect []int, actionName string) []any

// GetThrowableItems returns throwable items
func GetThrowableItems(manager *ecs.Manager, inv *Inventory, indicesToSelect []int) []any

// HasItemsWithAction checks if inventory has action
func HasItemsWithAction(manager *ecs.Manager, inv *Inventory, actionName string) bool

// HasThrowableItems checks for throwable items
func HasThrowableItems(manager *ecs.Manager, inv *Inventory) bool
```

#### Usage Example

```go
// Get player inventory component
playerEntity := ecsManager.World.Query(playerTag)[0].Entity
inv := common.GetComponentType[*gear.Inventory](playerEntity, gear.InventoryComponent)

// Add item
gear.AddItem(ecsManager.World, inv, itemEntityID)

// Display inventory
items := gear.GetInventoryForDisplay(ecsManager.World, inv, nil)
for _, item := range items {
    entry := item.(gear.InventoryListEntry)
    fmt.Printf("%s x%d\n", entry.Name, entry.Count)
}

// Find throwable items
throwables := gear.GetThrowableItems(ecsManager.World, inv, nil)

// Remove item at index 0
gear.RemoveItem(ecsManager.World, inv, 0)
```

### Entity Management Functions

**File:** `common/ecsutil.go` (149 LOC)
**Purpose:** Core entity utilities

#### Component Access

```go
// GetComponentType retrieves typed component (type-safe)
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T

// GetAttributes convenience function
func GetAttributes(e *ecs.Entity) *Attributes

// GetPosition convenience function
func GetPosition(e *ecs.Entity) *coords.LogicalPosition
```

#### Entity Queries

```go
// GetAllEntities returns all entity IDs
func (em *EntityManager) GetAllEntities() []ecs.EntityID

// HasComponent checks if entity has component
func (em *EntityManager) HasComponent(entityID ecs.EntityID, component *ecs.Component) bool

// GetComponent retrieves component by entity ID
func (em *EntityManager) GetComponent(entityID ecs.EntityID, component *ecs.Component) (interface{}, bool)

// GetCreatureAtPosition finds monster at position (uses O(1) PositionSystem)
func GetCreatureAtPosition(ecsmanager *EntityManager, pos *coords.LogicalPosition) *ecs.Entity
```

#### Distance Calculation

```go
// DistanceBetween calculates Chebyshev distance
func DistanceBetween(e1 *ecs.Entity, e2 *ecs.Entity) int
```

#### Usage Example

```go
// Type-safe component access
pos := common.GetPosition(entity)
attr := common.GetAttributes(entity)

// Check for component
if entity.HasComponent(gear.ItemComponent) {
    item := common.GetComponentType[*gear.Item](entity, gear.ItemComponent)
}

// Find creature at position
targetPos := coords.LogicalPosition{X: 10, Y: 5}
creature := common.GetCreatureAtPosition(ecsManager, &targetPos)
if creature != nil {
    // Attack creature
}

// Calculate distance
distance := common.DistanceBetween(playerEntity, enemyEntity)
if distance <= attackRange {
    // Can attack
}
```

---

## ECS Best Practices (Gold Standard)

The squad and inventory systems demonstrate **perfect ECS architecture**. Use these patterns for all new systems.

### Pure Data Components

**✅ CORRECT:** Components contain only data, zero logic methods.

```go
// ✅ Perfect example: SquadMemberData
type SquadMemberData struct {
    SquadID ecs.EntityID  // Just data
}

// ✅ Perfect example: Inventory
type Inventory struct {
    ItemEntityIDs []ecs.EntityID  // Just data
}
```

**❌ WRONG:** Components with logic methods.

```go
// ❌ Anti-pattern: Logic in component
type Inventory struct {
    Items []*ecs.Entity
}
func (inv *Inventory) AddItem(item *ecs.Entity) {  // NO! Logic belongs in systems
    inv.Items = append(inv.Items, item)
}
```

**Why This Matters:**
Logic in components prevents reuse, complicates testing, and violates ECS principles. Systems should operate on components, not the other way around.

### Native EntityID Usage

**✅ CORRECT:** Use `ecs.EntityID` everywhere, never entity pointers.

```go
// ✅ Perfect: EntityID relationships
type SquadMemberData struct {
    SquadID ecs.EntityID  // Native type
}

type Item struct {
    Properties ecs.EntityID  // Native type, not pointer
}

// ✅ Perfect: EntityID in maps
type PositionSystem struct {
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // Value keys
}
```

**❌ WRONG:** Entity pointers in components or maps.

```go
// ❌ Anti-pattern: Entity pointers
type SquadMemberData struct {
    Squad *ecs.Entity  // NO! Prevents serialization
}

type Item struct {
    Properties *ecs.Entity  // NO! Creates circular references
}

// ❌ Anti-pattern: Pointer map keys
spatialGrid map[*coords.LogicalPosition]*ecs.Entity  // NO! O(n) lookup
```

**Why This Matters:**
- EntityIDs enable serialization (pointers don't)
- EntityIDs avoid circular references
- Value types work correctly as map keys (O(1) lookup)
- Entity pointers can become stale if entity is destroyed

### Query-Based Relationships

**✅ CORRECT:** Discover relationships through ECS queries, don't store references.

```go
// ✅ Perfect: Query-based discovery
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

// ✅ Perfect: No stored unit list in SquadData
type SquadData struct {
    SquadID   ecs.EntityID
    Formation FormationType
    // Units discovered via query, not stored
}
```

**❌ WRONG:** Storing entity lists in components.

```go
// ❌ Anti-pattern: Stored relationships
type SquadData struct {
    SquadID ecs.EntityID
    Units   []ecs.EntityID  // NO! Requires manual sync on add/remove
}
```

**Why This Matters:**
Stored relationships require manual synchronization (add unit → update squad list). Queries are always correct because they reflect actual component state.

### System-Based Logic

**✅ CORRECT:** All logic in functions, components are pure data.

```go
// ✅ Perfect: System function
func AddItem(manager *ecs.Manager, inv *Inventory, itemEntityID ecs.EntityID) {
    itemEntity := FindItemEntityByID(manager, itemEntityID)
    // Logic here...
}

// ✅ Perfect: Pure data component
type Inventory struct {
    ItemEntityIDs []ecs.EntityID  // No methods
}
```

**❌ WRONG:** Logic methods on components.

```go
// ❌ Anti-pattern: Component methods
type Inventory struct {
    Items []ecs.EntityID
}
func (inv *Inventory) AddItem(itemID ecs.EntityID) {  // NO!
    inv.Items = append(inv.Items, itemID)
}
```

**Why This Matters:**
System functions can be tested independently, reused across systems, and composed. Component methods create tight coupling.

### Value Map Keys

**✅ CORRECT:** Use value types as map keys for O(1) performance.

```go
// ✅ Perfect: Value-based keys (O(1) hash lookup)
type PositionSystem struct {
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // Value type key
}

// ✅ Perfect: LogicalPosition is a value type
type LogicalPosition struct {
    X, Y int  // Value type (not pointer)
}
```

**❌ WRONG:** Pointer keys causing O(n) performance.

```go
// ❌ Anti-pattern: Pointer keys (O(n) lookup)
spatialGrid map[*coords.LogicalPosition]*ecs.Entity  // NO! Pointer key
```

**Why This Matters:**
Go's map implementation hashes value types efficiently. Pointer keys can't be hashed by value, requiring iteration (O(n)). This caused a **50x performance degradation** in the legacy tracker system.

### Reference Implementations

Use these files as templates for new systems:

**Squad System** (Perfect ECS):
- `squads/components.go` (331 LOC) - Pure data components
- `squads/squadqueries.go` (286 LOC) - Query-based discovery
- `squads/squadcombat.go` (406 LOC) - System-based logic
- `squads/visualization.go` (175 LOC) - Rendering system

**Inventory System** (Perfect ECS):
- `gear/Inventory.go` (241 LOC) - System functions
- `gear/items.go` (177 LOC) - EntityID-based relationships
- `gear/gearutil.go` (115 LOC) - Query-based entity lookup

**Position System** (Perfect ECS):
- `systems/positionsystem.go` (399 LOC) - O(1) spatial grid

**Entity Templates** (Perfect Factory Pattern):
- `entitytemplates/creators.go` (283 LOC) - Generic factory

---

## Practical Code Examples

### Creating Entities

#### Basic Entity Creation

```go
// Create entity with components
entity := manager.NewEntity().
    AddComponent(common.NameComponent, &common.Name{NameStr: "Goblin"}).
    AddComponent(common.PositionComponent, &coords.LogicalPosition{X: 10, Y: 5}).
    AddComponent(common.AttributeComponent, &common.Attributes{
        Strength:      10,
        Dexterity:     8,
        CurrentHealth: 20,
        MaxHealth:     20,
    })

entityID := entity.GetID()  // Get native EntityID
```

#### Template-Based Entity Creation

```go
// Load templates at startup
entitytemplates.ReadGameData()

// Create from template
config := entitytemplates.EntityConfig{
    Type:      entitytemplates.Creature,
    Name:      "Goblin",  // Template name from JSON
    ImagePath: "../assets/creatures/",
    Visible:   true,
    Position:  &coords.LogicalPosition{X: 10, Y: 5},
}

entity := entitytemplates.CreateEntityFromTemplate(manager, config, nil)
```

#### Creating Squad Unit

```go
// Create squad entity
squadEntity := manager.NewEntity().
    AddComponent(squads.SquadComponent, &squads.SquadData{
        SquadID:       squadEntity.GetID(),  // Self-reference
        Formation:     squads.FormationBalanced,
        Name:          "Alpha Squad",
        MaxUnits:      9,
        TotalCapacity: 6,
    })

// Create unit in squad
unitEntity := manager.NewEntity().
    AddComponent(common.AttributeComponent, &common.Attributes{
        Strength:  12,
        Dexterity: 10,
        MaxHealth: 30,
    }).
    AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
        SquadID: squadEntity.GetID(),  // Link to squad
    }).
    AddComponent(squads.GridPositionComponent, &squads.GridPositionData{
        AnchorRow: 0,
        AnchorCol: 1,
        Width:     1,
        Height:    1,
    }).
    AddComponent(squads.UnitRoleComponent, &squads.UnitRoleData{
        Role: squads.RoleTank,
    })
```

### Querying Entities

#### Basic Query

```go
// Query all monsters
for _, result := range ecsManager.World.Query(ecsManager.Tags["monsters"]) {
    entity := result.Entity
    pos := common.GetPosition(entity)
    attr := common.GetAttributes(entity)

    fmt.Printf("Monster at (%d, %d) - HP: %d/%d\n",
        pos.X, pos.Y, attr.CurrentHealth, attr.MaxHealth)
}
```

#### Query with Position Filter

```go
// Find all entities in radius
centerPos := coords.LogicalPosition{X: 10, Y: 10}
entityIDs := common.GlobalPositionSystem.GetEntitiesInRadius(centerPos, 3)

for _, entityID := range entityIDs {
    // Find entity
    for _, result := range ecsManager.World.Query(ecs.BuildTag()) {
        if result.Entity.GetID() == entityID {
            // Process entity
            break
        }
    }
}
```

#### Squad-Specific Query

```go
// Find all units in squad
squadID := squadEntity.GetID()
unitIDs := squads.GetUnitIDsInSquad(squadID, ecsManager)

// Process each unit
for _, unitID := range unitIDs {
    unit := squads.FindUnitByID(unitID, ecsManager)
    if unit == nil {
        continue
    }

    attr := common.GetAttributes(unit)
    role := common.GetComponentType[*squads.UnitRoleData](unit, squads.UnitRoleComponent)

    fmt.Printf("Unit: %s, HP: %d, Role: %s\n",
        attr.CurrentHealth, role.Role.String())
}
```

### Adding Components

#### Add Component to Existing Entity

```go
// Add Inventory component to player
player := getPlayerEntity(ecsManager)
inventory := &gear.Inventory{
    ItemEntityIDs: make([]ecs.EntityID, 0),
}
player.AddComponent(gear.InventoryComponent, inventory)
```

#### Add Component Conditionally

```go
// Add Leader component if unit has high Leadership
unit := getUnitEntity(ecsManager)
attr := common.GetAttributes(unit)

if attr.Leadership >= 10 {
    leader := &squads.LeaderData{
        Leadership: attr.Leadership,
        Experience: 0,
    }
    unit.AddComponent(squads.LeaderComponent, leader)

    // Also add ability slots
    abilitySlots := &squads.AbilitySlotData{
        Slots: [4]squads.AbilitySlot{},
    }
    unit.AddComponent(squads.AbilitySlotComponent, abilitySlots)
}
```

### Position Lookups

#### Check Position for Entity

```go
// Check if position is occupied
targetPos := coords.LogicalPosition{X: 12, Y: 8}
entityID := common.GlobalPositionSystem.GetEntityIDAt(targetPos)

if entityID != 0 {
    fmt.Println("Position occupied")
} else {
    fmt.Println("Position free")
}
```

#### Add Entity to Position System

```go
// Register entity in position system
newEntity := createMonster(manager)
spawnPos := coords.LogicalPosition{X: 10, Y: 5}

// Add to ECS
newEntity.AddComponent(common.PositionComponent, &spawnPos)

// Register in position system
common.GlobalPositionSystem.AddEntity(newEntity.GetID(), spawnPos)
```

#### Move Entity

```go
// Move entity from old to new position
oldPos := *common.GetPosition(entity)
newPos := coords.LogicalPosition{X: oldPos.X + 1, Y: oldPos.Y}

// Update position component
posComp := common.GetPosition(entity)
posComp.X = newPos.X
posComp.Y = newPos.Y

// Update position system
common.GlobalPositionSystem.MoveEntity(entity.GetID(), oldPos, newPos)
```

### Inventory Operations

#### Add Item to Inventory

```go
// Get player inventory
player := getPlayerEntity(ecsManager)
inv := common.GetComponentType[*gear.Inventory](player, gear.InventoryComponent)

// Create item
itemEntity := gear.CreateItem(manager, "Health Potion",
    coords.LogicalPosition{X: 0, Y: 0},
    "../assets/items/potion.png",
    gear.NewHealing(10))

// Add to inventory using system function
gear.AddItem(manager, inv, itemEntity.GetID())
```

#### Display Inventory

```go
// Get inventory display
items := gear.GetInventoryForDisplay(manager, inv, nil)

for _, item := range items {
    entry := item.(gear.InventoryListEntry)
    fmt.Printf("[%d] %s x%d\n", entry.Index, entry.Name, entry.Count)
}
```

#### Filter Throwable Items

```go
// Get only throwable items
throwables := gear.GetThrowableItems(manager, inv, nil)

if len(throwables) == 0 {
    fmt.Println("No throwable items")
    return
}

// Display throwable options
for i, item := range throwables {
    entry := item.(gear.InventoryListEntry)
    fmt.Printf("%d) %s\n", i, entry.Name)
}
```

### Squad Operations

#### Create Squad with Units

```go
// Create squad
squad := squads.CreateSquadEntity(manager, "Alpha Squad", squads.FormationBalanced)
squadID := squad.GetID()

// Create tank unit
tankAttrs := common.NewAttributes(15, 8, 0, 0, 3, 2)  // High Strength, Armor
tank := manager.NewEntity().
    AddComponent(common.AttributeComponent, &tankAttrs).
    AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
        SquadID: squadID,
    }).
    AddComponent(squads.GridPositionComponent, &squads.GridPositionData{
        AnchorRow: 0, AnchorCol: 1,  // Front center
        Width: 1, Height: 1,
    }).
    AddComponent(squads.UnitRoleComponent, &squads.UnitRoleData{
        Role: squads.RoleTank,
    }).
    AddComponent(squads.AttackRangeComponent, &squads.AttackRangeData{
        Range: 1,  // Melee
    })

// Create DPS unit
dpsAttrs := common.NewAttributes(12, 15, 0, 0, 1, 4)  // High Dexterity, Weapon
dps := manager.NewEntity().
    AddComponent(common.AttributeComponent, &dpsAttrs).
    AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
        SquadID: squadID,
    }).
    AddComponent(squads.GridPositionComponent, &squads.GridPositionData{
        AnchorRow: 1, AnchorCol: 1,  // Middle center
        Width: 1, Height: 1,
    }).
    AddComponent(squads.UnitRoleComponent, &squads.UnitRoleData{
        Role: squads.RoleDPS,
    }).
    AddComponent(squads.AttackRangeComponent, &squads.AttackRangeData{
        Range: 3,  // Ranged
    })
```

#### Execute Squad Attack

```go
// Execute attack between squads
result := squads.ExecuteSquadAttack(attackerSquadID, defenderSquadID, ecsManager)

fmt.Printf("Total Damage: %d\n", result.TotalDamage)
fmt.Printf("Units Killed: %d\n", len(result.UnitsKilled))

for unitID, damage := range result.DamageByUnit {
    fmt.Printf("Unit %d took %d damage\n", unitID, damage)
}
```

#### Check Squad Capacity

```go
// Check if squad can add more units
remaining := squads.GetSquadRemainingCapacity(squadID, ecsManager)

if remaining >= 1.0 {
    fmt.Printf("Can add unit (%.2f capacity remaining)\n", remaining)
} else {
    fmt.Println("Squad at capacity")
}

// Get detailed capacity info
used := squads.GetSquadUsedCapacity(squadID, ecsManager)
total := squads.GetSquadTotalCapacity(squadID, ecsManager)
fmt.Printf("Capacity: %.2f / %d\n", used, total)
```

---

## Developer Workflows

### Adding a New Component

**Step 1: Define component struct** (pure data, no methods)

```go
// File: common/newcomponent.go
package common

type MyComponent struct {
    Value     int
    IsActive  bool
    EntityRef ecs.EntityID  // ✅ Use EntityID, not pointer
}
```

**Step 2: Register component** (in `game_main/componentinit.go`)

```go
// Add to InitializeECS function
var MyComponent *ecs.Component

func InitializeECS(manager *ecs.Manager, tags map[string]ecs.Tag) {
    // ... existing components ...

    common.MyComponent = manager.NewComponent()

    // ... rest of initialization ...
}
```

**Step 3: Create tag (if needed)**

```go
// If component is frequently queried, create a tag
myTag := ecs.BuildTag(common.MyComponent, common.PositionComponent)
tags["myEntities"] = myTag
```

**Step 4: Add to entities**

```go
entity := manager.NewEntity().
    AddComponent(common.MyComponent, &common.MyComponent{
        Value:    10,
        IsActive: true,
    })
```

**Step 5: Create system functions**

```go
// File: common/mysystem.go
package common

// System function operates on component data
func ProcessMyComponent(manager *ecs.Manager, tags map[string]ecs.Tag) {
    for _, result := range manager.Query(tags["myEntities"]) {
        myComp := GetComponentType[*MyComponent](result.Entity, MyComponent)
        if myComp.IsActive {
            // Process component
        }
    }
}
```

### Creating a New Entity Type

**Step 1: Define JSON schema**

```json
// File: assets/gamedata/myentitydata.json
{
    "entities": [
        {
            "name": "ExampleEntity",
            "value": 42,
            "isActive": true,
            "imageFile": "example.png"
        }
    ]
}
```

**Step 2: Create JSON struct**

```go
// File: entitytemplates/jsonstructs.go
type JSONMyEntity struct {
    Name      string `json:"name"`
    Value     int    `json:"value"`
    IsActive  bool   `json:"isActive"`
    ImageFile string `json:"imageFile"`
}
```

**Step 3: Add loading function**

```go
// File: entitytemplates/readdata.go
var MyEntityTemplates []JSONMyEntity

func ReadMyEntityData() {
    data, err := os.ReadFile("../assets/gamedata/myentitydata.json")
    if err != nil {
        log.Fatal(err)
    }

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

**Step 4: Create factory function**

```go
// File: entitytemplates/creators.go
func CreateMyEntity(manager *ecs.Manager, name string, pos coords.LogicalPosition) *ecs.Entity {
    // Find template
    var template *JSONMyEntity
    for _, t := range MyEntityTemplates {
        if t.Name == name {
            template = &t
            break
        }
    }
    if template == nil {
        log.Printf("Template not found: %s", name)
        return nil
    }

    // Load image
    img, _, err := ebitenutil.NewImageFromFile("../assets/items/" + template.ImageFile)
    if err != nil {
        log.Fatal(err)
    }

    // Create entity
    entity := manager.NewEntity().
        AddComponent(common.NameComponent, &common.Name{NameStr: template.Name}).
        AddComponent(common.PositionComponent, &pos).
        AddComponent(common.MyComponent, &common.MyComponent{
            Value:    template.Value,
            IsActive: template.IsActive,
        }).
        AddComponent(rendering.RenderableComponent, &rendering.Renderable{
            Image:   img,
            Visible: true,
        })

    return entity
}
```

**Step 5: Use factory**

```go
// In game initialization
entity := entitytemplates.CreateMyEntity(manager, "ExampleEntity",
    coords.LogicalPosition{X: 10, Y: 5})
```

### Implementing a System

**Step 1: Create system struct** (if stateful)

```go
// File: systems/mysystem.go
package systems

type MySystem struct {
    manager *ecs.Manager
    cache   map[ecs.EntityID]int  // Optional: state
}

func NewMySystem(manager *ecs.Manager) *MySystem {
    return &MySystem{
        manager: manager,
        cache:   make(map[ecs.EntityID]int),
    }
}
```

**Step 2: Implement Update function**

```go
func (ms *MySystem) Update(deltaTime float64) {
    // Query entities
    for _, result := range ms.manager.Query(myTag) {
        entity := result.Entity
        myComp := common.GetComponentType[*MyComponent](entity, MyComponent)

        // System logic
        if myComp.IsActive {
            myComp.Value++
        }
    }
}
```

**Step 3: Register system in game loop**

```go
// File: game_main/gameinit.go
func SetupSystems(g *Game) {
    g.mySystem = systems.NewMySystem(g.em.World)
}

// File: game_main/main.go
func (g *Game) Update() error {
    // ... existing updates ...

    g.mySystem.Update(1.0 / 60.0)

    return nil
}
```

### Adding a UI Mode

**Step 1: Create mode struct**

```go
// File: gui/mymode.go
package gui

type MyMode struct {
    context *UIContext
    // Mode-specific state
}

func NewMyMode() *MyMode {
    return &MyMode{}
}
```

**Step 2: Implement UIMode interface**

```go
func (m *MyMode) GetModeName() string {
    return "MyMode"
}

func (m *MyMode) Initialize(ctx *UIContext) error {
    m.context = ctx
    // Build UI widgets here
    return nil
}

func (m *MyMode) Enter(fromMode UIMode) error {
    // Setup when entering mode
    return nil
}

func (m *MyMode) Exit(toMode UIMode) error {
    // Cleanup when exiting mode
    return nil
}

func (m *MyMode) HandleInput(input *InputState) {
    // Process keyboard/mouse input
    if input.KeysJustPressed[ebiten.KeyEscape] {
        // Request transition to another mode
        m.context.ModeManager.RequestTransition(explorationMode, "Back to exploration")
    }
}

func (m *MyMode) Update(deltaTime float64) error {
    // Update mode state
    return nil
}

func (m *MyMode) Render(screen *ebiten.Image) {
    // Render UI (EbitenUI handles this automatically)
}
```

**Step 3: Register mode**

```go
// File: game_main/gamesetup.go
func SetupUI(g *Game) {
    // ... existing mode setup ...

    myMode := gui.NewMyMode()
    g.uiModeManager.RegisterMode(myMode)
}
```

**Step 4: Trigger mode transition**

```go
// From another mode or input handler
g.uiModeManager.SetMode("MyMode")

// Or request transition (happens at end of frame)
g.uiModeManager.RequestTransition(myMode, "Opening MyMode")
```

### Debugging ECS Issues

#### Check Entity Has Component

```go
if entity.HasComponent(common.MyComponent) {
    fmt.Println("Entity has MyComponent")
} else {
    fmt.Println("Entity missing MyComponent")
}
```

#### Print All Components on Entity

```go
// Iterate through all component types
components := []struct{
    name string
    comp *ecs.Component
}{
    {"Position", common.PositionComponent},
    {"Attributes", common.AttributeComponent},
    {"Name", common.NameComponent},
    // ... add more ...
}

for _, c := range components {
    if entity.HasComponent(c.comp) {
        fmt.Printf("✓ %s\n", c.name)
    } else {
        fmt.Printf("✗ %s\n", c.name)
    }
}
```

#### Verify Tag Query

```go
// Check how many entities match tag
results := manager.Query(tags["monsters"])
fmt.Printf("Found %d monsters\n", len(results))

// Print first entity's components
if len(results) > 0 {
    entity := results[0].Entity
    pos := common.GetPosition(entity)
    attr := common.GetAttributes(entity)
    name := common.GetComponentType[*common.Name](entity, common.NameComponent)

    fmt.Printf("Monster: %s at (%d, %d), HP: %d/%d\n",
        name.NameStr, pos.X, pos.Y, attr.CurrentHealth, attr.MaxHealth)
}
```

#### Debug Position System

```go
// Check position system state
fmt.Printf("Entities tracked: %d\n", common.GlobalPositionSystem.GetEntityCount())

// Check specific position
pos := coords.LogicalPosition{X: 10, Y: 5}
entityID := common.GlobalPositionSystem.GetEntityIDAt(pos)
if entityID != 0 {
    fmt.Printf("Entity %d at position (%d, %d)\n", entityID, pos.X, pos.Y)
} else {
    fmt.Println("Position empty")
}

// List all occupied positions
occupied := common.GlobalPositionSystem.GetOccupiedPositions()
fmt.Printf("Occupied positions: %d\n", len(occupied))
for i, pos := range occupied {
    if i >= 10 {
        fmt.Println("...")
        break
    }
    fmt.Printf("  (%d, %d)\n", pos.X, pos.Y)
}
```

#### Trace Component Data Flow

```go
// Add logging to track component changes
func MoveEntity(entity *ecs.Entity, newPos coords.LogicalPosition) {
    oldPos := *common.GetPosition(entity)

    fmt.Printf("[MOVE] Entity %d: (%d, %d) → (%d, %d)\n",
        entity.GetID(), oldPos.X, oldPos.Y, newPos.X, newPos.Y)

    // Update component
    posComp := common.GetPosition(entity)
    posComp.X = newPos.X
    posComp.Y = newPos.Y

    // Update position system
    common.GlobalPositionSystem.MoveEntity(entity.GetID(), oldPos, newPos)
}
```

---

## Anti-Patterns to Avoid

### Entity Pointers in Maps

**❌ WRONG:**
```go
type PositionTracker struct {
    entities map[*coords.LogicalPosition]*ecs.Entity  // NO!
}
```

**Why It's Wrong:**
- Prevents serialization (can't save/load game)
- Pointer keys cause O(n) lookup (pointer comparison, not value hashing)
- Creates memory leaks if entities destroyed but pointers remain
- Map iteration order is undefined

**✅ CORRECT:**
```go
type PositionSystem struct {
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // YES!
}
```

**Benefits:**
- Value keys enable O(1) hash-based lookup (50x faster)
- EntityIDs remain valid even if entity recreated
- Enables serialization
- Clear ownership semantics

### Logic in Components

**❌ WRONG:**
```go
type Inventory struct {
    Items []ecs.EntityID
}

func (inv *Inventory) AddItem(itemID ecs.EntityID) {  // NO!
    inv.Items = append(inv.Items, itemID)
}
```

**Why It's Wrong:**
- Violates ECS principles (components are data, systems are logic)
- Can't test logic without creating component instances
- Can't reuse logic across different component types
- Tight coupling between data and behavior

**✅ CORRECT:**
```go
type Inventory struct {
    ItemEntityIDs []ecs.EntityID  // Pure data
}

// System function (separate from component)
func AddItem(manager *ecs.Manager, inv *Inventory, itemID ecs.EntityID) {
    // Logic here
}
```

**Benefits:**
- Clear separation of concerns
- Testable in isolation
- Reusable across systems
- Follows ECS architecture

### Direct Entity References

**❌ WRONG:**
```go
type Item struct {
    Properties *ecs.Entity  // NO!
}

type SquadMemberData struct {
    Squad *ecs.Entity  // NO!
}
```

**Why It's Wrong:**
- Creates circular references (hard to serialize)
- Pointers become stale if entity destroyed
- Prevents entity recycling
- Complicates garbage collection

**✅ CORRECT:**
```go
type Item struct {
    Properties ecs.EntityID  // YES!
}

type SquadMemberData struct {
    SquadID ecs.EntityID  // YES!
}

// Lookup via system function
func FindItemEntity(manager *ecs.Manager, itemID ecs.EntityID) *ecs.Entity {
    for _, result := range manager.Query(ecs.BuildTag()) {
        if result.Entity.GetID() == itemID {
            return result.Entity
        }
    }
    return nil
}
```

**Benefits:**
- EntityIDs are stable identifiers
- Easy to serialize (just store ID)
- Enables entity destruction/recreation
- Clear ownership

### Pointer Map Keys

**❌ WRONG:**
```go
// Legacy PositionTracker (removed in v3.0)
type PositionTracker struct {
    positions map[*coords.LogicalPosition]*ecs.Entity  // NO!
}

func (pt *PositionTracker) GetAt(pos *coords.LogicalPosition) *ecs.Entity {
    return pt.positions[pos]  // O(n) lookup!
}
```

**Why It's Wrong:**
- Pointer comparison, not value hashing
- O(n) performance (must iterate to find matching pointer)
- 50x slower than value-based keys (measured)
- Memory fragmentation from pointer storage

**✅ CORRECT:**
```go
type PositionSystem struct {
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // YES!
}

func (ps *PositionSystem) GetEntityIDAt(pos coords.LogicalPosition) ecs.EntityID {
    if ids, ok := ps.spatialGrid[pos]; ok && len(ids) > 0 {
        return ids[0]  // O(1) lookup!
    }
    return 0
}
```

**Benefits:**
- O(1) hash-based lookup
- Value semantics (position {10, 5} always equals {10, 5})
- 50x performance improvement
- Better cache locality

### Stored Relationships

**❌ WRONG:**
```go
type SquadData struct {
    SquadID ecs.EntityID
    Units   []ecs.EntityID  // NO! Requires manual sync
}

// Problem: Must update Units list when adding/removing
func AddUnitToSquad(squad *SquadData, unitID ecs.EntityID) {
    squad.Units = append(squad.Units, unitID)  // Manual sync
}
```

**Why It's Wrong:**
- Requires manual synchronization (easy to forget)
- Duplicate source of truth (components AND list)
- Becomes stale if units destroyed
- Complicates entity lifecycle

**✅ CORRECT:**
```go
type SquadData struct {
    SquadID   ecs.EntityID
    Formation FormationType
    // Units discovered via query, not stored
}

// Query-based discovery (always correct)
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

**Benefits:**
- Single source of truth (component data)
- Always reflects actual state
- No manual synchronization
- Handles entity destruction automatically

---

## Troubleshooting

### Common Issues

#### Entity Not Rendering

**Symptoms:** Entity exists in ECS but doesn't appear on screen

**Causes:**
1. Missing Renderable component
2. Missing Position component
3. Renderable.Visible = false
4. Entity outside viewport (if scrolling enabled)
5. Z-order issue (rendered behind something)

**Solutions:**
```go
// Check for required components
if !entity.HasComponent(rendering.RenderableComponent) {
    fmt.Println("Missing Renderable component")
}
if !entity.HasComponent(common.PositionComponent) {
    fmt.Println("Missing Position component")
}

// Check visibility
renderable := common.GetComponentType[*rendering.Renderable](entity, rendering.RenderableComponent)
if !renderable.Visible {
    fmt.Println("Entity not visible")
    renderable.Visible = true
}

// Check position
pos := common.GetPosition(entity)
fmt.Printf("Entity at (%d, %d)\n", pos.X, pos.Y)

// Check if in viewport
if graphics.MAP_SCROLLING_ENABLED {
    playerPos := g.playerData.Pos
    distance := pos.ChebyshevDistance(&playerPos)
    if distance > graphics.ViewableSquareSize / 2 {
        fmt.Println("Entity outside viewport")
    }
}
```

#### Component Not Found

**Symptoms:** `GetComponentData()` returns (nil, false)

**Causes:**
1. Component not registered in `componentinit.go`
2. Component not added to entity
3. Wrong component variable used

**Solutions:**
```go
// Verify component registration
if common.PositionComponent == nil {
    fmt.Println("PositionComponent not registered!")
}

// Check if entity has component
if !entity.HasComponent(common.PositionComponent) {
    fmt.Println("Entity missing Position component")
    // Add component
    entity.AddComponent(common.PositionComponent, &coords.LogicalPosition{X: 0, Y: 0})
}

// Use type-safe accessor
pos := common.GetPosition(entity)  // Returns nil if missing
if pos == nil {
    fmt.Println("Position is nil")
}
```

#### Query Returns No Results

**Symptoms:** `manager.Query(tag)` returns empty slice

**Causes:**
1. Tag not registered
2. No entities match tag requirements
3. Component not added to entities
4. Tag built with wrong components

**Solutions:**
```go
// Check tag exists
if tag, ok := ecsManager.Tags["monsters"]; ok {
    results := ecsManager.World.Query(tag)
    fmt.Printf("Found %d monsters\n", len(results))
} else {
    fmt.Println("monsters tag not registered!")
}

// Build temporary tag to debug
tempTag := ecs.BuildTag(common.AttributeComponent, common.PositionComponent)
results := ecsManager.World.Query(tempTag)
fmt.Printf("Entities with Attributes+Position: %d\n", len(results))

// Check entity has all tag components
entity := getTestEntity()
hasAttr := entity.HasComponent(common.AttributeComponent)
hasPos := entity.HasComponent(common.PositionComponent)
fmt.Printf("Has Attributes: %v, Has Position: %v\n", hasAttr, hasPos)
```

#### Position System Sync Issues

**Symptoms:** Entity at position according to component, but not in PositionSystem

**Causes:**
1. Entity not added to PositionSystem
2. Position changed but MoveEntity() not called
3. Entity destroyed but not removed from PositionSystem

**Solutions:**
```go
// Add entity to PositionSystem after creation
entity := createMonster(manager)
pos := common.GetPosition(entity)
common.GlobalPositionSystem.AddEntity(entity.GetID(), *pos)

// Always use MoveEntity when changing position
func MoveEntitySafely(entity *ecs.Entity, newPos coords.LogicalPosition) {
    oldPos := *common.GetPosition(entity)

    // Update component
    posComp := common.GetPosition(entity)
    *posComp = newPos

    // Update PositionSystem
    common.GlobalPositionSystem.MoveEntity(entity.GetID(), oldPos, newPos)
}

// Remove from PositionSystem when destroying
func DestroyEntity(entity *ecs.Entity, manager *ecs.Manager) {
    pos := common.GetPosition(entity)
    common.GlobalPositionSystem.RemoveEntity(entity.GetID(), *pos)
    manager.DeleteEntity(entity)
}
```

#### Inventory System Errors

**Symptoms:** Items not appearing, duplicate items, count errors

**Causes:**
1. Direct Inventory manipulation (bypassing system functions)
2. EntityID not found in manager
3. Item.Properties not set correctly

**Solutions:**
```go
// ✅ ALWAYS use system functions
gear.AddItem(manager, inv, itemEntityID)  // Correct
inv.ItemEntityIDs = append(...)           // WRONG!

// Verify item entity exists
itemEntity := gear.FindItemEntityByID(manager, itemEntityID)
if itemEntity == nil {
    fmt.Printf("Item entity %d not found\n", itemEntityID)
}

// Check item has required components
if !itemEntity.HasComponent(gear.ItemComponent) {
    fmt.Println("Entity missing Item component")
}
if !itemEntity.HasComponent(common.NameComponent) {
    fmt.Println("Entity missing Name component")
}

// Debug inventory contents
for i, itemID := range inv.ItemEntityIDs {
    itemEntity := gear.FindItemEntityByID(manager, itemID)
    if itemEntity == nil {
        fmt.Printf("[%d] Invalid item ID: %d\n", i, itemID)
        continue
    }
    name := common.GetComponentType[*common.Name](itemEntity, common.NameComponent)
    item := gear.GetItemByID(manager, itemID)
    fmt.Printf("[%d] %s x%d\n", i, name.NameStr, item.Count)
}
```

### Performance Problems

#### Slow Rendering

**Symptoms:** FPS drops, choppy animation

**Causes:**
1. Too many entities being rendered
2. Large images not scaled
3. Inefficient querying in draw loop
4. Debug drawing enabled

**Solutions:**
```go
// Enable viewport rendering (only draw visible area)
graphics.MAP_SCROLLING_ENABLED = true

// Check number of renderables
renderableTag := ecsManager.Tags["renderables"]
count := len(ecsManager.World.Query(renderableTag))
fmt.Printf("Rendering %d entities\n", count)

// Disable debug mode
DEBUG_MODE = false

// Profile rendering
start := time.Now()
rendering.ProcessRenderablesInSquare(...)
elapsed := time.Since(start)
if elapsed > time.Millisecond * 16 {
    fmt.Printf("Rendering slow: %v\n", elapsed)
}
```

#### Slow Position Lookups

**Symptoms:** Lag when moving, attacking, or spawning entities

**Causes:**
1. Not using PositionSystem (falling back to O(n) search)
2. PositionSystem not initialized
3. Querying all entities instead of using position system

**Solutions:**
```go
// Verify PositionSystem initialized
if common.GlobalPositionSystem == nil {
    fmt.Println("PositionSystem not initialized!")
    common.GlobalPositionSystem = systems.NewPositionSystem(ecsManager.World)
}

// Use PositionSystem for lookups
// ✅ FAST: O(1) lookup
entityID := common.GlobalPositionSystem.GetEntityIDAt(pos)

// ❌ SLOW: O(n) search
for _, result := range ecsManager.World.Query(monsterTag) {
    if pos.IsEqual(common.GetPosition(result.Entity)) {
        // Found entity (but slow!)
    }
}

// Check PositionSystem entity count
count := common.GlobalPositionSystem.GetEntityCount()
fmt.Printf("Position system tracking %d entities\n", count)
```

#### Memory Leaks

**Symptoms:** Memory usage grows over time

**Causes:**
1. Entities destroyed but not removed from PositionSystem
2. Event listeners not cleaned up
3. Component data not freed
4. Image resources duplicated

**Solutions:**
```go
// Proper entity destruction
func DestroyEntity(entity *ecs.Entity, manager *ecs.Manager) {
    // Remove from PositionSystem
    if entity.HasComponent(common.PositionComponent) {
        pos := common.GetPosition(entity)
        common.GlobalPositionSystem.RemoveEntity(entity.GetID(), *pos)
    }

    // Remove from manager (frees component data)
    manager.DeleteEntity(entity)
}

// Reuse images instead of loading multiple times
var imageCache = make(map[string]*ebiten.Image)

func LoadImage(path string) (*ebiten.Image, error) {
    if img, ok := imageCache[path]; ok {
        return img, nil  // Reuse cached image
    }

    img, _, err := ebitenutil.NewImageFromFile(path)
    if err != nil {
        return nil, err
    }
    imageCache[path] = img
    return img, nil
}
```

### Debugging Techniques

#### Enable Debug Mode

```go
// File: game_main/config.go
const DEBUG_MODE = true  // Shows debug overlays
```

#### Add Debug Logging

```go
// Add to Update() loop
func (g *Game) Update() error {
    if DEBUG_MODE {
        // Log entity counts
        fmt.Printf("Entities: %d\n", len(g.em.GetAllEntities()))

        // Log position system state
        fmt.Printf("Positions tracked: %d\n", common.GlobalPositionSystem.GetEntityCount())

        // Log player state
        fmt.Printf("Player: (%d, %d) HP: %d/%d\n",
            g.playerData.Pos.X, g.playerData.Pos.Y,
            g.playerData.Attributes.CurrentHealth,
            g.playerData.Attributes.MaxHealth)
    }

    return nil
}
```

#### Debug Input Commands

```go
// File: input/debuginput.go
// F1-F12 keys trigger debug actions
func PlayerDebugActions(playerData *common.PlayerData) {
    if ebiten.IsKeyPressed(ebiten.KeyF1) {
        // Print all entities
        for _, entityID := range ecsManager.GetAllEntities() {
            fmt.Printf("Entity %d\n", entityID)
        }
    }

    if ebiten.IsKeyPressed(ebiten.KeyF2) {
        // Print occupied positions
        positions := common.GlobalPositionSystem.GetOccupiedPositions()
        for _, pos := range positions {
            entityID := common.GlobalPositionSystem.GetEntityIDAt(pos)
            fmt.Printf("Position (%d, %d): Entity %d\n", pos.X, pos.Y, entityID)
        }
    }
}
```

#### Visual Debug Overlays

```go
// Draw entity bounding boxes
func DrawDebugOverlay(screen *ebiten.Image, entities []ecs.Entity) {
    for _, entity := range entities {
        pos := common.GetPosition(&entity)
        pixelPos := coords.CoordManager.LogicalToPixel(*pos)

        // Draw red rectangle around entity
        ebitenutil.DrawRect(screen,
            float64(pixelPos.X), float64(pixelPos.Y),
            float64(graphics.ScreenInfo.TileSize),
            float64(graphics.ScreenInfo.TileSize),
            color.RGBA{255, 0, 0, 128})
    }
}
```

---

## Appendices

### Glossary

**ECS (Entity Component System)**: Architectural pattern where entities are IDs, components are pure data, and systems are logic functions.

**Entity**: Lightweight handle (ID) that components attach to. Has no intrinsic behavior.

**Component**: Pure data structure registered with ECS manager. Describes what an entity is.

**System**: Function that queries entities and executes logic. Describes how entities behave.

**Tag**: Pre-built entity filter for efficient querying. Tags match entities with specific component combinations.

**EntityID**: Native identifier (`ecs.EntityID`) for entities. Use instead of entity pointers.

**LogicalPosition**: Tile-based coordinate (X, Y integers). Primary coordinate system for game logic.

**PixelPosition**: Screen pixel coordinate. Used for rendering.

**CoordinateManager**: Singleton that transforms between coordinate spaces.

**Squad**: Tactical unit composed of multiple members in 3x3 grid formation.

**Template**: JSON-defined entity configuration loaded at runtime.

**Factory**: Function that creates entities from templates.

**Query**: ECS operation that returns entities matching component criteria.

**System Function**: Stateless function operating on component data (ECS pattern).

**Position System**: O(1) spatial lookup system using value-based map keys.

### File Statistics

**Total Lines of Code:** ~15,000+ (estimated)

**Key Systems:**
- Squad System: 2358 LOC (squads/)
- Inventory System: 533 LOC (gear/)
- Position System: 399 LOC (systems/)
- Graphics: 390 LOC (graphics/)
- Entity Templates: 283 LOC (entitytemplates/)

**Package Breakdown:**
- common/ - Core ECS utilities
- coords/ - Coordinate system
- entitytemplates/ - Template factories
- game_main/ - Main entry point
- gear/ - Inventory & items
- graphics/ - Rendering & shapes
- gui/ - UI modes
- input/ - Input controllers
- rendering/ - Sprite rendering
- spawning/ - Entity spawning
- squads/ - Squad combat system
- systems/ - ECS systems
- worldmap/ - Map generation

### External Dependencies

```go
// Core dependencies
github.com/bytearena/ecs         // ECS framework
github.com/hajimehoshi/ebiten/v2 // Game engine
github.com/ebitenui/ebitenui     // UI toolkit
github.com/norendren/go-fov/fov  // Field of view

// Image handling
image
image/color
image/png

// Standard library
encoding/json  // Template loading
fmt
log
math
os
```

**Installation:**
```bash
go mod tidy  # Installs all dependencies
```

### Configuration Reference

**File:** `game_main/config.go`

```go
// Debug flags
const DEBUG_MODE = false              // Debug visualization
const ENABLE_BENCHMARKING = false     // CPU profiling

// Player starting attributes
const DefaultPlayerStrength = 15      // 50 HP (20 + 15*2)
const DefaultPlayerDexterity = 20     // 100% hit, 10% crit, 6% dodge
const DefaultPlayerMagic = 0          // No magic abilities
const DefaultPlayerLeadership = 0     // No squad leadership
const DefaultPlayerArmor = 2          // 4 physical resistance (2*2)
const DefaultPlayerWeapon = 3         // 6 bonus damage (3*2)

// Asset paths
const PlayerImagePath = "../assets/creatures/player1.png"
const AssetItemsDir = "../assets/items/"
```

**Graphics Configuration:**

```go
// File: graphics/graphictypes.go
const (
    DefaultTileSize     = 32
    DefaultDungeonWidth = 80
    DefaultDungeonHeight = 50
    ViewableSquareSize  = 25  // Viewport radius
    StatsUIOffset       = 200 // Stats panel width
)

var MAP_SCROLLING_ENABLED = true  // Viewport rendering
```

**Build Commands:**

```bash
# Build executable
go build -o game_main/game_main.exe game_main/*.go

# Run without building
go run game_main/*.go

# Run tests
go test ./...

# Run specific package tests
go test ./squads -v
go test ./systems -v

# Install dependencies
go mod tidy
```

---

## Document Metadata

**Version:** 4.0 Enhanced Edition
**Date:** 2025-10-22
**Author:** Claude Code (Anthropic)
**Codebase Version:** v3.0 (2025-10-21 Inventory Refactor)
**Lines:** 6000+ (enhanced from 6263 original)
**Purpose:** Comprehensive technical reference for TinkerRogue developers

**Enhancement Focus:**
- Complete component reference with all fields documented
- Comprehensive system function reference
- Practical code examples for all common operations
- Developer workflow guides (adding components, systems, entities)
- ECS best practices with squad/inventory patterns
- Troubleshooting section with solutions
- Quick navigation for different developer needs

**Changelog:**
- v4.0 (2025-10-22): Enhanced edition with component reference, workflows, troubleshooting
- v3.0 (2025-10-12): Codebase audit with actual LOC counts
- v2.0 (2025-10-21): Inventory system refactor documentation
- v1.0 (2025-10-10): Initial comprehensive documentation

**Related Documents:**
- CLAUDE.md - Project configuration and build commands
- analysis/MASTER_ROADMAP.md - Development roadmap
- analysis/squad_system_final.md - Squad system architecture
- docs/DEVELOPMENT_WORKFLOWS.md - Developer workflows
- docs/AGENT_REFERENCE.md - AI agent reference

---

**Navigation Quick Links:**

- [Back to Top](#tinkerrogue-comprehensive-technical-documentation)
- [Quick Navigation](#quick-navigation)
- [Getting Started](#getting-started-guide)
- [Core Concepts](#core-concepts)
- [Component Reference](#complete-component-reference)
- [System Functions](#system-functions-reference)
- [ECS Best Practices](#ecs-best-practices-gold-standard)
- [Code Examples](#practical-code-examples)
- [Developer Workflows](#developer-workflows)
- [Troubleshooting](#troubleshooting)
