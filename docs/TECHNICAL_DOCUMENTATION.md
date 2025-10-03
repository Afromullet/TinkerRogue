# TinkerRogue: Comprehensive Technical Documentation

**Version:** 1.0
**Last Updated:** 2025-10-03
**Engine:** Ebiten v2 (Go 2D Game Engine)
**Architecture:** Entity Component System (ECS)

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Core Design Patterns](#core-design-patterns)
4. [Entity Component System (ECS)](#entity-component-system-ecs)
5. [Core Systems](#core-systems)
6. [Package Architecture](#package-architecture)
7. [Data Flow and Integration](#data-flow-and-integration)
8. [Visual Effects System](#visual-effects-system)
9. [Coordinate System](#coordinate-system)
10. [Template and Factory System](#template-and-factory-system)
11. [Status Effects vs Item Actions](#status-effects-vs-item-actions)
12. [Game Loop and Rendering](#game-loop-and-rendering)
13. [Development Roadmap](#development-roadmap)
14. [Appendices](#appendices)

---

## 1. Executive Summary

TinkerRogue is a tactical roguelike game built in Go using the Ebiten 2D game engine. The codebase implements a clean Entity Component System (ECS) architecture with clear separation of concerns across specialized packages. Recent refactoring efforts have consolidated scattered logic into unified systems, reducing technical debt and improving maintainability.

### Key Achievements
- **80% completion** of major simplification roadmap
- Unified coordinate management system (eliminated 73+ scattered calls)
- Consolidated input handling through InputCoordinator pattern
- Unified graphics shape system (8+ shape types → 1 BaseShape with variants)
- Generic entity template factory with type-safe creation

### Technology Stack
- **Language:** Go
- **Game Engine:** Ebiten v2
- **ECS Library:** bytearena/ecs
- **FOV System:** go-fov for field of view calculations

---

## 2. Architecture Overview

### 2.1 System Design Philosophy

The game follows a **component-based architecture** where entities are composed of reusable components rather than using inheritance hierarchies. This design enables:

1. **Flexible Entity Composition:** Entities are created by combining components
2. **Data-Oriented Design:** Components are pure data, systems operate on component queries
3. **Decoupled Systems:** Each package has a single responsibility
4. **Type Safety:** Go's strong typing with generic helpers for component access

### 2.2 High-Level Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                       Game Loop (main.go)                    │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │   Update()  │  │    Draw()    │  │  HandleInput()   │   │
│  └─────────────┘  └──────────────┘  └──────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
┌───────▼────────┐  ┌──────▼──────┐  ┌─────────▼─────────┐
│ InputCoordinator│  │  Rendering  │  │  Visual Effects   │
│   - Movement    │  │   System    │  │     Handler       │
│   - Combat      │  │             │  │                   │
│   - UI          │  └─────────────┘  └───────────────────┘
└────────┬────────┘
         │
┌────────▼────────────────────────────────────────────────┐
│             Entity Component System (ECS)                │
│  ┌──────────┐  ┌───────────┐  ┌──────────────────────┐ │
│  │ Entities │  │Components │  │   Systems/Queries    │ │
│  └──────────┘  └───────────┘  └──────────────────────┘ │
└──────────────────────────────────────────────────────────┘
         │
┌────────▼────────────────────────────────────────────────┐
│                    Game State                            │
│  ┌──────────┐  ┌───────────┐  ┌──────────────────────┐ │
│  │ GameMap  │  │PlayerData │  │   EntityManager      │ │
│  └──────────┘  └───────────┘  └──────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

### 2.3 Core Components Interaction

```
Player Input → InputCoordinator
                    ↓
              [Route to Appropriate Controller]
                    ↓
        ┌──────────┬──────────┬──────────┐
        │          │          │          │
   Movement    Combat       UI       Debug
   Controller  Controller   Controller  Input
        │          │          │          │
        └──────────┴──────────┴──────────┘
                    ↓
              [Modify ECS Entities]
                    ↓
              EntityManager
                    ↓
        [Update Attributes, Positions, etc.]
                    ↓
              Rendering System
                    ↓
              Screen Output
```

---

## 3. Core Design Patterns

### 3.1 Entity Component System (ECS) Pattern

**Purpose:** Composition over inheritance for game entities

**Implementation:**
```go
// Entity creation example
entity := manager.World.NewEntity()
entity.AddComponent(PositionComponent, &coords.LogicalPosition{X: 10, Y: 10})
entity.AddComponent(AttributeComponent, &common.Attributes{...})
entity.AddComponent(RenderableComponent, &rendering.Renderable{...})
```

**Benefits:**
- Entities are lightweight IDs
- Components are pure data structures
- Systems query for entities with specific component combinations
- Easy to add/remove capabilities at runtime

### 3.2 Coordinator Pattern (Input System)

**File:** `input/inputcoordinator.go`

**Purpose:** Centralized input handling with priority-based routing

**Architecture:**
```go
type InputCoordinator struct {
    movementController *MovementController
    combatController   *CombatController
    uiController       *UIController
    sharedState        *SharedInputState
}
```

**Priority Hierarchy:**
1. **UI Controller** (highest priority) - Inventory, menus
2. **Combat Controller** - Throwing, shooting
3. **Movement Controller** (lowest priority) - Player movement

**Benefits:**
- Eliminates scattered global input state
- Clear priority ordering prevents input conflicts
- Shared state managed explicitly
- Each controller is independently testable

### 3.3 Factory Pattern (Entity Templates)

**File:** `entitytemplates/creators.go`

**Purpose:** Type-safe entity creation from templates

**Evolution:**
```go
// OLD: 4 separate functions with duplicate code
CreateMonsterFromTemplate()
CreateMeleeWeaponFromTemplate()
CreateRangedWeaponFromTemplate()
CreateConsumableFromTemplate()

// NEW: Unified generic factory
CreateEntityFromTemplate(manager, config EntityConfig, data any) *ecs.Entity

// Type-safe enum for entity types
type EntityType int
const (
    EntityMeleeWeapon EntityType = iota
    EntityRangedWeapon
    EntityConsumable
    EntityCreature
)
```

**Usage Example:**
```go
config := EntityConfig{
    Type:      EntityCreature,
    Name:      "Goblin",
    ImagePath: "creatures/goblin.png",
    AssetDir:  "../assets/",
    Visible:   true,
    Position:  &coords.LogicalPosition{X: 10, Y: 10},
    GameMap:   gameMap,
}
entity := CreateEntityFromTemplate(manager, config, monsterData)
```

### 3.4 Component Composition Pattern

**File:** `entitytemplates/creators.go`

**Purpose:** Build entities through composable component adders

```go
type ComponentAdder func(entity *ecs.Entity)

func addMeleeWeaponComponents(w JSONMeleeWeapon) ComponentAdder {
    return func(entity *ecs.Entity) {
        entity.AddComponent(gear.ItemComponent, &gear.Item{Count: 1})
        entity.AddComponent(gear.MeleeWeaponComponent, &gear.MeleeWeapon{
            MinDamage:   w.MinDamage,
            MaxDamage:   w.MaxDamage,
            AttackSpeed: w.AttackSpeed,
        })
    }
}

// Compose entities with varying component sets
createFromTemplate(manager, name, imagePath, assetDir, visible, pos, adders...)
```

### 3.5 Strategy Pattern (Visual Effects)

**File:** `graphics/vx.go`

**Purpose:** Separate animation logic from rendering logic

**Architecture:**
```go
// Animator defines HOW properties change over time
type Animator interface {
    Update(effect *BaseEffect, elapsed float64) AnimationState
    Reset()
}

// Renderer defines HOW to draw the effect
type Renderer interface {
    Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState)
}

// BaseEffect combines both
type BaseEffect struct {
    animator Animator
    renderer Renderer
    // ... lifecycle fields
}
```

**Implementations:**
- **Animators:** FlickerAnimator, PulseAnimator, MotionAnimator, WaveAnimator
- **Renderers:** ImageRenderer, ProjectileRenderer, CloudRenderer, ElectricArcRenderer

**Example:**
```go
// Fire effect = Flicker animation + Image rendering
func NewFireEffect(x, y int, duration int) VisualEffect {
    return &BaseEffect{
        animator: &FlickerAnimator{
            scaleRange:   [2]float64{0.95, 1.05},
            opacityRange: [2]float64{0.7, 1.0},
        },
        renderer: &ImageRenderer{},
        // ... other fields
    }
}
```

### 3.6 Type-Safe Component Access Pattern

**File:** `common/ecsutil.go`

**Purpose:** Eliminate unsafe type assertions throughout codebase

```go
// Generic type-safe component getter with panic recovery
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {
    defer func() {
        if r := recover(); r != nil {
            // ERROR HANDLING IN FUTURE
        }
    }()

    if c, ok := entity.GetComponentData(component); ok {
        return c.(T)
    } else {
        var nilValue T
        return nilValue
    }
}

// Convenience wrappers for frequently accessed components
func GetAttributes(e *ecs.Entity) *Attributes {
    return GetComponentType[*Attributes](e, AttributeComponent)
}

func GetPosition(e *ecs.Entity) *coords.LogicalPosition {
    return GetComponentType[*coords.LogicalPosition](e, PositionComponent)
}
```

### 3.7 Unified Shape System Pattern

**File:** `graphics/drawableshapes.go`

**Purpose:** Consolidate 8+ shape types into unified system

**Before (Old System):**
```
Circle, Circle2, Circle3, Circle4
Square, Rectangle
Line, Cone
(Each with separate implementations)
```

**After (New System):**
```go
type BasicShapeType int
const (
    Circular    // radius-based
    Rectangular // width/height-based
    Linear      // length/direction-based
)

type BaseShape struct {
    Type       BasicShapeType
    Size       int              // Primary dimension
    Width, Height int           // For rectangles
    Direction  *ShapeDirection  // For directional shapes
    Quality    common.QualityType
}

// Unified interface
type TileBasedShape interface {
    GetIndices() []int
    UpdatePosition(pixelX, pixelY int)
    StartPositionPixels() (int, int)
    GetDirection() ShapeDirection
    CanRotate() bool
}
```

**Quality-Based Factories:**
```go
func NewCircle(pixelX, pixelY int, quality common.QualityType) *BaseShape {
    var radius int
    switch quality {
    case common.LowQuality:    radius = rand.Intn(3)  // 0-2
    case common.NormalQuality: radius = rand.Intn(4)  // 0-3
    case common.HighQuality:   radius = rand.Intn(9)  // 0-8
    }
    return &BaseShape{Type: Circular, Size: radius, Quality: quality}
}
```

---

## 4. Entity Component System (ECS)

### 4.1 ECS Fundamentals

**Library:** `github.com/bytearena/ecs`

The ECS architecture separates:
- **Entities:** Unique IDs with no behavior
- **Components:** Pure data structures
- **Systems:** Logic that operates on entities with specific components

### 4.2 Core Components

**Location:** `common/commoncomponents.go`

```go
// Position in game world (logical coordinates)
type Position struct {
    X, Y int
}

// Entity display name
type Name struct {
    NameStr string
}

// Creature/player stats
type Attributes struct {
    MaxHealth          int
    CurrentHealth      int
    AttackBonus        int
    BaseArmorClass     int
    BaseProtection     int
    BaseDodgeChance    float32
    BaseMovementSpeed  int
    // ... computed totals
    TotalArmorClass    int
    TotalProtection    int
    TotalDodgeChance   float32
    TotalMovementSpeed int
    TotalAttackSpeed   int
    CanAct             bool
}

// User messages (attacks, status effects)
type UserMessage struct {
    AttackMessage       string
    GameStateMessage    string
    StatusEffectMessage string
}
```

### 4.3 Specialized Components

**Gear Components** (`gear/equipmentcomponents.go`):
```go
// Base item component
type Item struct {
    Count int
}

// Weapon components
type MeleeWeapon struct {
    MinDamage, MaxDamage int
    AttackSpeed          int
}

type RangedWeapon struct {
    MinDamage, MaxDamage int
    ShootingRange        int
    AttackSpeed          int
    TargetArea          TileBasedShape
}

// Armor component
type Armor struct {
    ArmorClass  int
    Protection  int
    DodgeChance float32
}

// Inventory component
type Inventory struct {
    InventoryContent []*ecs.Entity
}
```

**Rendering Components** (`rendering/rendering.go`):
```go
type Renderable struct {
    Image   *ebiten.Image
    Visible bool
}
```

**Creature Components** (`monsters/creatures.go`):
```go
type Creature struct {
    Path []coords.LogicalPosition  // AI pathfinding
}
```

### 4.4 Component Registration

**File:** `game_main/componentinit.go`

Components must be registered with the ECS manager:

```go
func InitializeECS(em *common.EntityManager) {
    em.World = ecs.NewManager()
    em.WorldTags = make(map[string]ecs.Tag)

    // Register components
    common.PositionComponent = em.World.NewComponent()
    common.NameComponent = em.World.NewComponent()
    common.AttributeComponent = em.World.NewComponent()
    rendering.RenderableComponent = em.World.NewComponent()

    // Create tags for queries
    renderables := ecs.BuildTag(rendering.RenderableComponent, common.PositionComponent)
    em.WorldTags["renderables"] = renderables

    monsters := ecs.BuildTag(monsters.CreatureComponent, common.PositionComponent)
    em.WorldTags["monsters"] = monsters
}
```

### 4.5 Entity Manager Wrapper

**File:** `common/ecsutil.go`

```go
type EntityManager struct {
    World     *ecs.Manager           // ECS library manager
    WorldTags map[string]ecs.Tag     // Named queries
}

// Query entities with specific components
results := manager.World.Query(manager.WorldTags["monsters"])
for _, result := range results {
    entity := result.Entity
    pos := result.Components[PositionComponent].(*coords.LogicalPosition)
    // ... process entity
}
```

### 4.6 Entity Lifecycle

**Creation:**
```go
entity := manager.World.NewEntity()
entity.AddComponent(ComponentType, &ComponentData{...})
```

**Modification:**
```go
// Get component
attr := GetComponentType[*Attributes](entity, AttributeComponent)
// Modify
attr.CurrentHealth -= damage
```

**Deletion:**
```go
// Mark for deletion (removed in cleanup phase)
manager.World.RemoveEntity(entity)
```

**Cleanup** (`resourcemanager/cleanup.go`):
```go
func RemoveDeadEntities(em *EntityManager, gm *GameMap) {
    for _, c := range em.World.Query(em.WorldTags["monsters"]) {
        attr := GetAttributes(c.Entity)
        if attr.CurrentHealth <= 0 {
            RemoveEntity(em.World, gm, c.Entity)
        }
    }
}
```

---

## 5. Core Systems

### 5.1 Input System

**Architecture:** Priority-based coordinator pattern

**Files:**
- `input/inputcoordinator.go` - Main coordinator
- `input/movementcontroller.go` - Player movement
- `input/combatcontroller.go` - Ranged/throwing attacks
- `input/uicontroller.go` - UI interactions

**Input Flow:**
```
User Input (Ebiten)
      ↓
InputCoordinator.HandleInput()
      ↓
[Check priority order]
      ↓
1. UIController.CanHandle() → UIController.HandleInput()
   (Inventory open? Menu active?)
      ↓
2. CombatController.CanHandle() → CombatController.HandleInput()
   (Throwing mode? Shooting mode?)
      ↓
3. MovementController.CanHandle() → MovementController.HandleInput()
   (Arrow keys? WASD?)
```

**Shared State:**
```go
type SharedInputState struct {
    PrevCursor         coords.PixelPosition  // Last cursor position
    PrevThrowInds      []int                 // Previous throw AOE tiles
    PrevRangedAttInds  []int                 // Previous ranged attack tiles
    TurnTaken          bool                  // Player action this turn
}
```

**Controller Interface:**
```go
type InputController interface {
    HandleInput() bool      // Process input, return true if handled
    CanHandle() bool        // Can this controller handle input now?
    OnActivate()           // Called when controller becomes active
    OnDeactivate()         // Called when controller becomes inactive
}
```

### 5.2 Combat System

**File:** `combat/attackingsystem.go`

**Melee Attack Flow:**
```
MeleeAttackSystem(attacker_pos, defender_pos)
      ↓
[Identify attacker and defender entities]
      ↓
[Get weapon component]
      ↓
damage = weapon.CalculateDamage()
      ↓
PerformAttack(attacker, defender, damage)
      ↓
Attack Roll: d20 + AttackBonus vs ArmorClass
      ↓
[If hit] Dodge Roll: d100 vs DodgeChance
      ↓
[If not dodged] Apply: damage - Protection
      ↓
UpdateAttackMessage(result)
```

**Attack Resolution:**
```go
func PerformAttack(em *EntityManager, pl *PlayerData, gm *GameMap,
                   damage int, attacker, defender *ecs.Entity,
                   isPlayerAttacking bool) bool {
    attAttr := GetAttributes(attacker)
    defAttr := GetAttributes(defender)

    // d20 + attack bonus
    attackRoll := GetDiceRoll(20) + attAttr.AttackBonus

    if attackRoll >= defAttr.TotalArmorClass {
        dodgeRoll := GetRandomBetween(0, 100)

        if dodgeRoll >= int(defAttr.TotalDodgeChance) {
            totalDamage := damage - defAttr.TotalProtection
            if totalDamage < 0 { totalDamage = 1 }

            defAttr.CurrentHealth -= totalDamage
            return true  // Hit successful
        }
    }
    return false  // Miss or dodge
}
```

**Ranged Attack System:**
```go
func RangedAttackSystem(ecsmanager *EntityManager, pl *PlayerData,
                        gm *GameMap, attackerPos *LogicalPosition) {
    // Get weapon's target area (AOE)
    targets := weapon.GetTargets(ecsmanager)

    // Apply damage to all targets in range
    for _, target := range targets {
        if attackerPos.InRange(targetPos, weapon.ShootingRange) {
            damage := weapon.CalculateDamage()
            PerformAttack(..., damage, attacker, target, ...)

            // Display visual effect
            weapon.DisplayShootingVX(attackerPos, defenderPos)
        }
    }
}
```

### 5.3 Graphics and Rendering System

**Primary Files:**
- `graphics/drawableshapes.go` - Shape system
- `graphics/vx.go` - Visual effects
- `rendering/rendering.go` - Entity rendering

**Rendering Pipeline:**
```
Game.Draw(screen)
      ↓
[Render map tiles]
gameMap.DrawLevel(screen)
      ↓
[Render entities]
ProcessRenderables(ecsManager, gameMap, screen)
      ↓
[Query all renderable entities]
for entity in World.Query("renderables"):
    if entity.Visible and PlayerVisible.IsVisible(pos):
        DrawImage(entity.Image, position)
      ↓
[Render visual effects]
VXHandler.DrawVisualEffects(screen)
      ↓
[Render UI]
playerUI.Draw(screen)
```

**FOV-Based Rendering:**
```go
func ProcessRenderables(ecsmanager *EntityManager, gameMap GameMap,
                        screen *ebiten.Image, debugMode bool) {
    for _, result := range ecsmanager.World.Query(ecsmanager.WorldTags["renderables"]) {
        pos := result.Components[PositionComponent].(*LogicalPosition)
        renderable := result.Components[RenderableComponent].(*Renderable)

        if !renderable.Visible { continue }

        // Only render if in player's field of view (or debug mode)
        if debugMode || gameMap.PlayerVisible.IsVisible(pos.X, pos.Y) {
            tile := gameMap.Tiles[CoordManager.LogicalToIndex(*pos)]
            op := &ebiten.DrawImageOptions{}
            op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
            screen.DrawImage(renderable.Image, op)
        }
    }
}
```

**Scrolling Map Rendering:**
```go
func ProcessRenderablesInSquare(ecsmanager *EntityManager, gameMap GameMap,
                                 screen *ebiten.Image, playerPos *LogicalPosition,
                                 squareSize int, debugMode bool) {
    sq := coords.NewDrawableSection(playerPos.X, playerPos.Y, squareSize)

    // Calculate center offset for camera
    scaledCenterOffsetX := screenWidth/2 - playerPos.X*scaledTileSize
    scaledCenterOffsetY := screenHeight/2 - playerPos.Y*scaledTileSize

    for _, result := range ecsmanager.World.Query(...) {
        pos := result.Components[PositionComponent].(*LogicalPosition)

        // Only render entities in viewable square
        if pos.X >= sq.StartX && pos.X <= sq.EndX &&
           pos.Y >= sq.StartY && pos.Y <= sq.EndY {
            // Apply scaling and centering
            op.GeoM.Scale(scaleFactor, scaleFactor)
            op.GeoM.Translate(tilePixelX*scaleFactor + scaledCenterOffsetX,
                            tilePixelY*scaleFactor + scaledCenterOffsetY)
            screen.DrawImage(img, op)
        }
    }
}
```

### 5.4 Coordinate System

**File:** `coords/cordmanager.go`

**Problem Solved:** Eliminated 73+ scattered `CoordTransformer` calls

**Core Conversions:**
```
Logical Coordinates (X, Y)  ←→  Flat Array Index
        ↕                              ↕
Pixel Coordinates (pX, pY)  ←→  Screen Coordinates
```

**CoordinateManager API:**
```go
type CoordinateManager struct {
    dungeonWidth, dungeonHeight int
    tileSize                    int
    scaleFactor                 int
}

// Core conversions
LogicalToIndex(pos LogicalPosition) int
IndexToLogical(index int) LogicalPosition
LogicalToPixel(pos LogicalPosition) PixelPosition
PixelToLogical(pos PixelPosition) LogicalPosition
IndexToPixel(index int) PixelPosition

// Validation
IsValidLogical(pos LogicalPosition) bool

// Batch operations
GetTilePositions(indices []int) []LogicalPosition
```

**Global Instance:**
```go
var CoordManager *CoordinateManager  // Initialized in init()

// Usage throughout codebase
index := coords.CoordManager.LogicalToIndex(logicalPos)
pixelPos := coords.CoordManager.LogicalToPixel(logicalPos)
```

**Viewport System:**
```go
type Viewport struct {
    centerX, centerY int  // Logical coordinates of camera center
    manager          *CoordinateManager
}

// Convert logical to screen with camera offset
func (v *Viewport) LogicalToScreen(pos LogicalPosition) (float64, float64) {
    offsetX := screenWidth/2 - centerX*tileSize*scaleFactor
    offsetY := screenHeight/2 - centerY*tileSize*scaleFactor

    scaledX := pos.X*tileSize*scaleFactor
    scaledY := pos.Y*tileSize*scaleFactor

    return scaledX + offsetX, scaledY + offsetY
}
```

### 5.5 Template and Data Loading System

**Files:**
- `entitytemplates/creators.go` - Entity factories
- `entitytemplates/readdata.go` - JSON loading
- `entitytemplates/jsonstructs.go` - Data structures
- `entitytemplates/templatelib.go` - Template management

**Data Flow:**
```
JSON Template Files
      ↓
ReadGameData() → Load into memory structures
      ↓
Template Library (maps name → data)
      ↓
CreateEntityFromTemplate(config, data)
      ↓
Entity with components
```

**JSON Structure Example:**
```json
{
  "name": "Goblin",
  "image_path": "creatures/goblin.png",
  "attributes": {
    "max_health": 20,
    "attack_bonus": 3,
    "base_armor_class": 12,
    "base_protection": 2,
    "base_dodge_chance": 0.1,
    "base_movement_speed": 4
  },
  "melee_weapon": {
    "min_damage": 2,
    "max_damage": 6,
    "attack_speed": 1
  }
}
```

**Template Usage:**
```go
// 1. Read templates from JSON
entitytemplates.ReadGameData()

// 2. Create entity from template
config := EntityConfig{
    Type:      EntityCreature,
    Name:      template.Name,
    ImagePath: template.ImagePath,
    AssetDir:  "../assets/",
    Visible:   true,
    Position:  spawnPos,
    GameMap:   gameMap,
}
entity := CreateEntityFromTemplate(manager, config, templateData)

// 3. Add to game world
gameMap.AddEntityToTile(entity, spawnPos)
```

---

## 6. Package Architecture

### 6.1 Package Dependency Graph

```
main (game_main/)
  ├── common/          [Foundation - No dependencies]
  ├── coords/          [Depends on: none]
  ├── graphics/        [Depends on: common, coords]
  ├── rendering/       [Depends on: common, coords, graphics]
  ├── worldmap/        [Depends on: common, coords, graphics]
  ├── combat/          [Depends on: common, coords, graphics, worldmap]
  ├── gear/            [Depends on: common, coords, graphics]
  ├── monsters/        [Depends on: common, coords]
  ├── avatar/          [Depends on: common, coords, gear]
  ├── input/           [Depends on: common, coords, avatar, worldmap, gui]
  ├── entitytemplates/ [Depends on: common, coords, gear, monsters, rendering]
  ├── spawning/        [Depends on: common, coords, gear, entitytemplates]
  └── gui/             [Depends on: common, coords, gear, avatar]
```

### 6.2 Package Descriptions

#### `common/` - Foundation Package
**Purpose:** Core ECS utilities and shared types
**Files:**
- `commontypes.go` - Attributes, Name, UserMessage, Quality interfaces
- `commoncomponents.go` - Component definitions
- `ecsutil.go` - EntityManager, type-safe component access

**Key Exports:**
```go
type EntityManager struct {
    World     *ecs.Manager
    WorldTags map[string]ecs.Tag
}

func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T
func GetAttributes(e *ecs.Entity) *Attributes
func GetPosition(e *ecs.Entity) *coords.LogicalPosition
```

#### `coords/` - Coordinate Management
**Purpose:** Unified coordinate system
**Files:**
- `cordmanager.go` - CoordinateManager, Viewport
- `position.go` - LogicalPosition, PixelPosition types

**Key Functions:**
- Logical ↔ Index conversion
- Logical ↔ Pixel conversion
- Viewport/camera transformations

#### `graphics/` - Graphics Engine
**Purpose:** Shapes, visual effects, rendering utilities
**Files:**
- `drawableshapes.go` - Unified shape system
- `vx.go` - Visual effects system
- `colormatrix.go` - Color transformations
- `graphictypes.go` - Shared graphics types

**Shape System:**
```go
type BaseShape struct {
    Type      BasicShapeType  // Circular, Rectangular, Linear
    Size      int
    Direction *ShapeDirection
    Quality   common.QualityType
}

// Quality-based factories
NewCircle(x, y int, quality QualityType) *BaseShape
NewSquare(x, y int, quality QualityType) *BaseShape
NewLine(x, y int, dir ShapeDirection, quality QualityType) *BaseShape
```

#### `rendering/` - Entity Rendering
**Purpose:** Draw entities to screen
**Files:**
- `rendering.go` - ProcessRenderables, Renderable component

**Key Systems:**
- FOV-based rendering
- Scrolling map support
- Debug mode rendering

#### `worldmap/` - Game World
**Purpose:** Map generation, tile management, spatial operations
**Files:**
- `dungeongen.go` - Procedural dungeon generation
- `dungeontile.go` - Tile types and properties
- `GameMapUtil.go` - Map utilities

**Key Features:**
- Room-based dungeon generation
- Corridor connections
- Tile-based entity storage
- Field of view integration

#### `combat/` - Combat System
**Purpose:** Attack resolution and damage calculation
**Files:**
- `attackingsystem.go` - Melee and ranged attack systems

**Attack Flow:**
1. Attack roll (d20 + bonus) vs Armor Class
2. Dodge roll (d100) vs Dodge Chance
3. Damage calculation (damage - protection)
4. Message updates

#### `gear/` - Equipment and Items
**Purpose:** Items, inventory, equipment, status effects
**Files:**
- `items.go` - Base item types
- `equipmentcomponents.go` - Weapon, armor components
- `Inventory.go` - Inventory management
- `stateffect.go` - Status effects (Burning, Freezing, Sticky)
- `itemactions.go` - Item actions (Throwable)
- `consumables.go` - Consumable items
- `itemquality.go` - Quality system

**Status Effects:**
```go
type StatusEffects interface {
    StatusEffectComponent() *ecs.Component
    StatusEffectName() string
    Duration() int
    ApplyToCreature(c *ecs.QueryResult)
    StackEffect(eff any)
    Copy() StatusEffects
    common.Quality
}
```

#### `input/` - Input System
**Purpose:** Centralized input handling
**Files:**
- `inputcoordinator.go` - Main coordinator
- `movementcontroller.go` - Movement input
- `combatcontroller.go` - Combat input
- `uicontroller.go` - UI input
- `debuginput.go` - Debug commands

**Controller Pattern:**
```go
type InputController interface {
    HandleInput() bool
    CanHandle() bool
    OnActivate()
    OnDeactivate()
}
```

#### `entitytemplates/` - Entity Creation
**Purpose:** Template-based entity factories
**Files:**
- `creators.go` - Factory functions
- `readdata.go` - JSON loading
- `jsonstructs.go` - Template data structures
- `templatelib.go` - Template library

**Generic Factory:**
```go
func CreateEntityFromTemplate(
    manager EntityManager,
    config EntityConfig,
    data any
) *ecs.Entity
```

#### `spawning/` - Entity Spawning
**Purpose:** Procedural entity spawning
**Files:**
- `spawnmonsters.go` - Monster spawning
- `spawnloot.go` - Loot spawning
- `spawnthrowable.go` - Throwable item spawning
- `probtables.go` - Probability tables
- `loottables.go` - Loot tables

#### `gui/` - User Interface
**Purpose:** UI components and windows
**Files:**
- `playerUI.go` - Main player interface
- `itemui.go` - Item display
- `statsui.go` - Stats display
- `messagesUI.go` - Message log
- `equipmentUI.go` - Equipment panel
- `consumableUI.go` - Consumable panel
- `throwingUI.go` - Throwing interface

#### `avatar/` - Player Data
**Purpose:** Player-specific data and components
**Files:**
- `playerdata.go` - PlayerData struct, input states, equipment

**Player Data:**
```go
type PlayerData struct {
    PlayerEntity *ecs.Entity
    Pos          *coords.LogicalPosition
    Equipment    EquippedGear
    Inventory    *gear.Inventory
    InputStates  PlayerInputState
}
```

---

## 7. Data Flow and Integration

### 7.1 Game Initialization Sequence

```
main()
      ↓
BenchmarkSetup() [Optional profiling]
      ↓
NewGame()
  │
  ├── NewGameMap() → Generate dungeon
  ├── ReadGameData() → Load templates
  ├── InitializeECS() → Setup components
  ├── InitializePlayerData() → Create player
  ├── SpawnStartingCreatures() → Place enemies
  ├── SpawnStartingEquipment() → Give player gear
  └── AddCreaturesToTracker() → Register monsters
      ↓
CreateMainInterface() → Build UI
      ↓
NewInputCoordinator() → Setup input
      ↓
ebiten.RunGame(g) → Start game loop
```

### 7.2 Game Loop Data Flow

```
┌─────────────────── GAME LOOP ───────────────────┐
│                                                  │
│  Update()                                        │
│    ├── UI.Update()                              │
│    ├── VXHandler.UpdateVisualEffects()          │
│    ├── DebugInput()                             │
│    └── HandleInput()                            │
│         ├── UpdateAttributes()                  │
│         ├── InputCoordinator.HandleInput()      │
│         │    ├── UIController                   │
│         │    ├── CombatController               │
│         │    └── MovementController             │
│         ├── RunEffectTracker() [If turn taken]  │
│         └── RemoveDeadEntities()                │
│                                                  │
│  Draw(screen)                                    │
│    ├── DrawLevel() / DrawLevelCenteredSquare()  │
│    ├── ProcessRenderables()                     │
│    ├── ProcessUserLog()                         │
│    ├── VXHandler.DrawVisualEffects()            │
│    └── UI.Draw()                                │
│                                                  │
└──────────────────────────────────────────────────┘
```

### 7.3 Entity State Changes

**Damage Flow:**
```
Attack Input → CombatController
      ↓
MeleeAttackSystem() / RangedAttackSystem()
      ↓
PerformAttack()
      ├── Get attacker attributes
      ├── Get defender attributes
      ├── Roll attack (d20 + bonus)
      ├── Check armor class
      ├── Roll dodge (d100)
      ├── Calculate damage (dmg - prot)
      └── Modify defender.CurrentHealth
      ↓
UpdateAttackMessage()
      ↓
[Next update cycle]
RemoveDeadEntities()
      ├── Query all monsters
      ├── Check CurrentHealth <= 0
      └── RemoveEntity()
```

**Status Effect Flow:**
```
Throw Item Input → CombatController
      ↓
ThrowableAction.Execute()
      ├── Get AOE shape indices
      ├── Find monsters in AOE
      ├── Apply status effects
      └── Return applied effects
      ↓
[Effect Tracking]
RunEffectTracker()
      ├── Query entities with status effects
      ├── For each effect:
      │    ├── ApplyToCreature()
      │    ├── Decrement duration
      │    └── Remove if duration == 0
      └── Update attributes
```

### 7.4 Map-Entity Integration

**Tile-Based Entity Storage:**
```go
type Tile struct {
    TileType     TileType
    Blocked      bool
    tileContents TileContents
}

type TileContents struct {
    entities []*ecs.Entity  // All entities on this tile
}

// Add entity to tile
gameMap.AddEntityToTile(entity, position)

// Remove entity from tile
entity, err := gameMap.RemoveItemFromTile(index, position)
```

**Spatial Queries:**
```go
// Get creature at position
creature := GetCreatureAtPosition(ecsManager, position)

// Get all items on tile
tile := gameMap.Tile(position)
items := tile.tileContents.entities
```

---

## 8. Visual Effects System

### 8.1 Architecture Overview

**File:** `graphics/vx.go`

The visual effects system uses a **Strategy Pattern** to separate animation logic from rendering logic, enabling flexible composition of effects.

### 8.2 Core Components

```
VisualEffect (Interface)
      ↓
BaseEffect (Implementation)
  ├── Animator (Strategy)
  │    ├── FlickerAnimator
  │    ├── PulseAnimator
  │    ├── MotionAnimator
  │    └── WaveAnimator
  │
  └── Renderer (Strategy)
       ├── ImageRenderer
       ├── ProjectileRenderer
       ├── CloudRenderer
       └── ElectricArcRenderer
```

### 8.3 Effect Creation Examples

**Fire Effect:**
```go
func NewFireEffect(x, y int, duration int) VisualEffect {
    return &BaseEffect{
        startX:   float64(x),
        startY:   float64(y),
        duration: duration,
        img:      fireImage,
        animator: &FlickerAnimator{
            scaleRange:   [2]float64{0.95, 1.05},
            opacityRange: [2]float64{0.7, 1.0},
            jitterPos:    true,
        },
        renderer: &ImageRenderer{},
    }
}
```

**Electric Arc:**
```go
func NewElectricArc(startX, startY, endX, endY int, duration int) VisualEffect {
    return &BaseEffect{
        startX:   float64(startX),
        startY:   float64(startY),
        duration: duration,
        animator: nil,  // Stateless rendering
        renderer: NewElectricArcRenderer(startX, startY, endX, endY),
    }
}
```

### 8.4 Animator Implementations

**FlickerAnimator** (Fire, Electricity):
```go
type FlickerAnimator struct {
    flickerTimer int
    scaleRange   [2]float64  // min, max scale
    opacityRange [2]float64  // min, max opacity
    jitterPos    bool        // position jitter
}

func (a *FlickerAnimator) Update(effect *BaseEffect, elapsed float64) AnimationState {
    a.flickerTimer++
    return AnimationState{
        Scale:   randomBetween(scaleRange[0], scaleRange[1]),
        Opacity: randomBetween(opacityRange[0], opacityRange[1]),
        OffsetX: jitterPos ? random() : 0,
        OffsetY: jitterPos ? random() : 0,
    }
}
```

**MotionAnimator** (Projectiles):
```go
type MotionAnimator struct {
    endX, endY         float64
    currentX, currentY float64
    speed              float64
}

func (a *MotionAnimator) Update(effect *BaseEffect, elapsed float64) AnimationState {
    // Move toward target
    direction := normalize(endX - currentX, endY - currentY)
    currentX += direction.X * speed
    currentY += direction.Y * speed

    // Check if arrived
    if distance(current, end) < speed {
        effect.completed = true
    }

    return AnimationState{
        OffsetX: currentX - effect.startX,
        OffsetY: currentY - effect.startY,
    }
}
```

### 8.5 Renderer Implementations

**ImageRenderer** (Standard image-based effects):
```go
func (r *ImageRenderer) Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState) {
    opts := &ebiten.DrawImageOptions{}

    // Apply transformations
    opts.GeoM.Scale(state.Scale, state.Scale)
    opts.GeoM.Translate(effect.startX + state.OffsetX, effect.startY + state.OffsetY)

    // Apply color/opacity
    opts.ColorM.Scale(state.Brightness, state.Brightness, state.Brightness, state.Opacity)

    screen.DrawImage(effect.img, opts)
}
```

**ElectricArcRenderer** (Procedural line-based):
```go
func (r *ElectricArcRenderer) Draw(screen *ebiten.Image, effect *BaseEffect, state AnimationState) {
    // Generate jagged segments
    segments := generateElectricPath(effect.startX, effect.startY, r.endX, r.endY)

    // Draw segments with varying thickness
    for i := 0; i < len(segments)-1; i++ {
        vector.StrokeLine(screen,
            segments[i].x, segments[i].y,
            segments[i+1].x, segments[i+1].y,
            randomThickness(), electricColor, false)
    }
}
```

### 8.6 Effect Handler

**Global Handler:**
```go
var VXHandler VisualEffectHandler

type VisualEffectHandler struct {
    vx     []VisualEffect      // Single-point effects
    vxArea []VisualEffectArea  // Area-of-effect
}

func (vis *VisualEffectHandler) UpdateVisualEffects() {
    vis.clearVisualEffects()  // Remove completed effects

    for _, v := range vis.vx {
        v.UpdateVisualEffect()
    }
    for _, v := range vis.vxArea {
        v.UpdateVisualEffect()
    }
}

func (vis *VisualEffectHandler) DrawVisualEffects(screen *ebiten.Image) {
    for _, v := range vis.vx {
        v.DrawVisualEffect(screen)
    }
    for _, v := range vis.vxArea {
        v.DrawVisualEffect(screen)
    }
}
```

**Area Effects:**
```go
type VisualEffectArea struct {
    shape      TileBasedShape   // AOE shape
    visEffects []VisualEffect   // Effect per tile
}

func NewVisualEffectArea(centerX, centerY int, shape TileBasedShape, vx VisualEffect) VisualEffectArea {
    indices := shape.GetIndices()
    visEffects := make([]VisualEffect, 0)

    for _, ind := range indices {
        pos := coords.CoordManager.IndexToPixel(ind)
        vxCopy := vx.Copy()
        vxCopy.SetVXCommon(pos.X, pos.Y, vx.VXImg())
        visEffects = append(visEffects, vxCopy)
    }

    return VisualEffectArea{shape: shape, visEffects: visEffects}
}
```

### 8.7 Effect Usage in Game

```go
// Throwing item with fire effect
throwable.Execute(targetPos, sourcePos, world, tags)
  ↓
if throwable.VX != nil {
    throwable.VX.ResetVX()
    vxArea := NewVisualEffectArea(sourcePos.X, sourcePos.Y, throwable.Shape, throwable.VX)
    graphics.AddVXArea(vxArea)
}
  ↓
[Game Loop - Update]
VXHandler.UpdateVisualEffects()
  ↓
[Game Loop - Draw]
VXHandler.DrawVisualEffects(screen)
```

---

## 9. Coordinate System

### 9.1 Coordinate Spaces

The game uses **three coordinate systems**:

1. **Logical Coordinates** - Grid-based tile positions (X, Y)
2. **Pixel Coordinates** - Rendering positions (pX, pY)
3. **Screen Coordinates** - Final display with camera offset

### 9.2 Coordinate Manager

**File:** `coords/cordmanager.go`

**Problem Addressed:**
- Before: 73+ scattered `CoordTransformer` calls
- After: Centralized `CoordManager` instance

**Core API:**
```go
type CoordinateManager struct {
    dungeonWidth  int
    dungeonHeight int
    tileSize      int
    scaleFactor   int
}

// Logical ↔ Index
LogicalToIndex(pos LogicalPosition) int
IndexToLogical(index int) LogicalPosition

// Logical ↔ Pixel
LogicalToPixel(pos LogicalPosition) PixelPosition
PixelToLogical(pos PixelPosition) LogicalPosition

// Combined operations
IndexToPixel(index int) PixelPosition
```

**Conversion Examples:**
```go
// Logical (10, 5) → Index
// Formula: Y * dungeonWidth + X
index := coords.CoordManager.LogicalToIndex(LogicalPosition{X: 10, Y: 5})
// Result: 5 * 100 + 10 = 510

// Index → Logical
// Formula: X = index % width, Y = index / width
pos := coords.CoordManager.IndexToLogical(510)
// Result: LogicalPosition{X: 10, Y: 5}

// Logical → Pixel (32px tiles)
// Formula: pixelX = logicalX * tileSize
pixelPos := coords.CoordManager.LogicalToPixel(LogicalPosition{X: 10, Y: 5})
// Result: PixelPosition{X: 320, Y: 160}
```

### 9.3 Position Types

**File:** `coords/position.go`

```go
type LogicalPosition struct {
    X, Y int
}

func (p *LogicalPosition) IsEqual(other *LogicalPosition) bool {
    return p.X == other.X && p.Y == other.Y
}

func (p *LogicalPosition) InRange(other *LogicalPosition, rang int) bool {
    return p.ChebyshevDistance(other) <= rang
}

func (p *LogicalPosition) ChebyshevDistance(other *LogicalPosition) int {
    dx := abs(p.X - other.X)
    dy := abs(p.Y - other.Y)
    return max(dx, dy)
}

type PixelPosition struct {
    X, Y int
}
```

### 9.4 Viewport System

**Camera-Based Rendering:**
```go
type Viewport struct {
    centerX, centerY int  // Logical coords of camera center
    manager          *CoordinateManager
}

func (v *Viewport) LogicalToScreen(pos LogicalPosition) (float64, float64) {
    // Calculate offset to center viewport on centerX, centerY
    offsetX := screenWidth/2 - centerX*tileSize*scaleFactor
    offsetY := screenHeight/2 - centerY*tileSize*scaleFactor

    // Convert logical to scaled pixel coordinates
    scaledX := pos.X * tileSize * scaleFactor
    scaledY := pos.Y * tileSize * scaleFactor

    // Apply viewport offset
    return scaledX + offsetX, scaledY + offsetY
}

func (v *Viewport) ScreenToLogical(screenX, screenY int) LogicalPosition {
    // Reverse viewport transformation
    offsetX := screenWidth/2 - centerX*tileSize*scaleFactor
    offsetY := screenHeight/2 - centerY*tileSize*scaleFactor

    uncenteredX := screenX - offsetX
    uncenteredY := screenY - offsetY

    // Reverse scaling
    pixelX := uncenteredX / scaleFactor
    pixelY := uncenteredY / scaleFactor

    // Convert to logical
    return PixelToLogical(PixelPosition{X: pixelX, Y: pixelY})
}
```

### 9.5 Coordinate Usage Patterns

**Map Tile Access:**
```go
// Get tile at logical position
tile := gameMap.Tile(&LogicalPosition{X: 10, Y: 5})

// Internal implementation
func (gm *GameMap) Tile(pos *LogicalPosition) *Tile {
    index := coords.CoordManager.LogicalToIndex(*pos)
    return gm.Tiles[index]
}
```

**Entity Positioning:**
```go
// Place entity at logical position
entity.AddComponent(PositionComponent, &LogicalPosition{X: 10, Y: 5})

// Render entity at correct pixel position
pos := GetPosition(entity)
pixelPos := coords.CoordManager.LogicalToPixel(*pos)
op.GeoM.Translate(float64(pixelPos.X), float64(pixelPos.Y))
```

**Shape Calculations:**
```go
// Shape operates in pixel space for position
shape := graphics.NewCircle(pixelX, pixelY, quality)

// GetIndices converts to logical for tile queries
func (s *BaseShape) GetIndices() []int {
    logical := coords.CoordManager.PixelToLogical(s.Position)
    return s.calculateCircle(logical.X, logical.Y)
}
```

---

## 10. Template and Factory System

### 10.1 Evolution of Entity Creation

**Before (Multiple Functions):**
```go
CreateMonsterFromTemplate(mgr, name, img, pos, data, gm)
CreateMeleeWeaponFromTemplate(mgr, name, img, pos, data)
CreateRangedWeaponFromTemplate(mgr, name, img, pos, data)
CreateConsumableFromTemplate(mgr, name, img, pos, data)
```

**After (Unified Factory):**
```go
CreateEntityFromTemplate(manager EntityManager, config EntityConfig, data any) *ecs.Entity
```

### 10.2 EntityConfig System

**Configuration Structure:**
```go
type EntityType int
const (
    EntityMeleeWeapon EntityType = iota
    EntityRangedWeapon
    EntityConsumable
    EntityCreature
)

type EntityConfig struct {
    Type      EntityType
    Name      string
    ImagePath string
    AssetDir  string
    Visible   bool
    Position  *coords.LogicalPosition
    GameMap   *worldmap.GameMap  // Only for creatures (tile blocking)
}
```

### 10.3 Factory Implementation

**Generic Factory:**
```go
func CreateEntityFromTemplate(manager EntityManager, config EntityConfig, data any) *ecs.Entity {
    var adders []ComponentAdder

    switch config.Type {
    case EntityMeleeWeapon:
        w := data.(JSONMeleeWeapon)  // Type assertion
        adders = []ComponentAdder{addMeleeWeaponComponents(w)}

    case EntityRangedWeapon:
        w := data.(JSONRangedWeapon)
        adders = []ComponentAdder{addRangedWeaponComponents(w)}

    case EntityConsumable:
        c := data.(JSONAttributeModifier)
        adders = []ComponentAdder{addConsumableComponents(c)}

    case EntityCreature:
        m := data.(JSONMonster)
        adders = []ComponentAdder{addCreatureComponents(m)}

        // Creature-specific: block map tile
        if config.GameMap != nil && config.Position != nil {
            ind := coords.CoordManager.LogicalToIndex(*config.Position)
            config.GameMap.Tiles[ind].Blocked = true
        }
    }

    return createFromTemplate(manager, config.Name, config.ImagePath,
                            config.AssetDir, config.Visible, config.Position, adders...)
}
```

### 10.4 Component Adder Pattern

**Pattern Definition:**
```go
type ComponentAdder func(entity *ecs.Entity)
```

**Example Implementations:**
```go
func addMeleeWeaponComponents(w JSONMeleeWeapon) ComponentAdder {
    return func(entity *ecs.Entity) {
        entity.AddComponent(gear.ItemComponent, &gear.Item{Count: 1})
        entity.AddComponent(gear.MeleeWeaponComponent, &gear.MeleeWeapon{
            MinDamage:   w.MinDamage,
            MaxDamage:   w.MaxDamage,
            AttackSpeed: w.AttackSpeed,
        })
    }
}

func addCreatureComponents(m JSONMonster) ComponentAdder {
    return func(entity *ecs.Entity) {
        entity.AddComponent(monsters.CreatureComponent, &monsters.Creature{...})
        entity.AddComponent(common.UserMsgComponent, &common.UserMessage{})
        entity.AddComponent(common.AttributeComponent, &common.Attributes{
            MaxHealth:     m.Attributes.MaxHealth,
            CurrentHealth: m.Attributes.MaxHealth,
            AttackBonus:   m.Attributes.AttackBonus,
            // ... other attributes
        })

        // Optional components
        if m.Armor != nil {
            entity.AddComponent(gear.ArmorComponent, &gear.Armor{...})
        }
        if m.MeleeWeapon != nil {
            entity.AddComponent(gear.MeleeWeaponComponent, &gear.MeleeWeapon{...})
        }
    }
}
```

### 10.5 JSON Template Loading

**File:** `entitytemplates/readdata.go`

**JSON Structure:**
```go
type JSONMonster struct {
    Name        string               `json:"name"`
    ImagePath   string               `json:"image_path"`
    Attributes  JSONAttributes       `json:"attributes"`
    Armor       *JSONArmor           `json:"armor,omitempty"`
    MeleeWeapon *JSONMeleeWeapon     `json:"melee_weapon,omitempty"`
    RangedWeapon *JSONRangedWeapon   `json:"ranged_weapon,omitempty"`
}

type JSONAttributes struct {
    MaxHealth         int     `json:"max_health"`
    AttackBonus       int     `json:"attack_bonus"`
    BaseArmorClass    int     `json:"base_armor_class"`
    BaseProtection    int     `json:"base_protection"`
    BaseDodgeChance   float32 `json:"base_dodge_chance"`
    BaseMovementSpeed int     `json:"base_movement_speed"`
}
```

**Loading Process:**
```go
func ReadGameData() {
    // Read monster templates
    monsterData, _ := ioutil.ReadFile("templates/monsters.json")
    json.Unmarshal(monsterData, &MonsterTemplates)

    // Read weapon templates
    weaponData, _ := ioutil.ReadFile("templates/weapons.json")
    json.Unmarshal(weaponData, &WeaponTemplates)

    // Store in template library
    for _, template := range MonsterTemplates {
        TemplateLib[template.Name] = template
    }
}
```

### 10.6 Usage Examples

**Creating from Template:**
```go
// 1. Load templates
entitytemplates.ReadGameData()

// 2. Get template data
template := entitytemplates.TemplateLib["Goblin"]

// 3. Create entity
config := EntityConfig{
    Type:      EntityCreature,
    Name:      template.Name,
    ImagePath: template.ImagePath,
    AssetDir:  "../assets/",
    Visible:   true,
    Position:  &LogicalPosition{X: 10, Y: 5},
    GameMap:   gameMap,
}
goblin := CreateEntityFromTemplate(manager, config, template)

// 4. Add to world
gameMap.AddEntityToTile(goblin, config.Position)
```

**Backward Compatibility:**
```go
// Deprecated wrappers maintained for existing code
func CreateMonsterFromTemplate(mgr EntityManager, name, img string, pos LogicalPosition,
                               data JSONMonster, gm *GameMap) *ecs.Entity {
    config := EntityConfig{
        Type:      EntityCreature,
        Name:      name,
        ImagePath: img,
        Position:  &pos,
        GameMap:   gm,
    }
    return CreateEntityFromTemplate(mgr, config, data)
}
```

---

## 11. Status Effects vs Item Actions

### 11.1 Conceptual Separation

**Problem:** Throwables were forced into StatusEffect interface when they are **actions**, not **effects**

**Solution:** Separate ItemAction interface for item-triggered actions

### 11.2 ItemAction Interface

**File:** `gear/itemactions.go`

```go
// ItemAction represents actions that can be performed with items
// This is separate from StatusEffects which are ongoing conditions
type ItemAction interface {
    ActionName() string
    ActionComponent() *ecs.Component

    // Execute performs the action and returns status effects to apply
    Execute(targetPos, sourcePos *LogicalPosition,
            world *ecs.Manager, worldTags map[string]ecs.Tag) []StatusEffects

    // CanExecute checks if action can be performed
    CanExecute(targetPos, sourcePos *LogicalPosition) bool

    GetVisualEffect() graphics.VisualEffect
    GetAOEShape() graphics.TileBasedShape
    GetRange() int
    Copy() ItemAction

    common.Quality
}
```

### 11.3 ThrowableAction Implementation

**Composition Pattern:**
```go
type ThrowableAction struct {
    MainProps         CommonItemProperties
    ThrowingRange     int
    Damage            int
    Shape             graphics.TileBasedShape
    VX                graphics.VisualEffect
    EffectsToApply    []StatusEffects  // Throwables CONTAIN effects
}
```

**Key Difference:**
- **Old:** Throwable "is a" StatusEffect (forced inheritance)
- **New:** Throwable "has" StatusEffects (proper composition)

**Execution Flow:**
```go
func (t *ThrowableAction) Execute(targetPos, sourcePos *LogicalPosition,
                                   world *ecs.Manager, worldTags map[string]ecs.Tag) []StatusEffects {
    // Start visual effect
    if t.VX != nil {
        t.VX.ResetVX()
        graphics.AddVXArea(NewVisualEffectArea(sourcePos.X, sourcePos.Y, t.Shape, t.VX))
    }

    // Get affected positions
    affectedPositions := coords.CoordManager.GetTilePositionsAsCommon(t.Shape.GetIndices())
    appliedEffects := make([]StatusEffects, 0)

    // Apply effects to monsters in AOE
    for _, c := range world.Query(worldTags["monsters"]) {
        monsterPos := c.Components[PositionComponent].(*LogicalPosition)

        for _, pos := range affectedPositions {
            if monsterPos.IsEqual(&pos) && monsterPos.InRange(sourcePos, t.ThrowingRange) {
                // Apply all effects
                for _, effect := range t.EffectsToApply {
                    appliedEffects = append(appliedEffects, effect.Copy())
                }
            }
        }
    }

    return appliedEffects
}
```

### 11.4 StatusEffects Interface

**File:** `gear/stateffect.go`

```go
type StatusEffects interface {
    StatusEffectComponent() *ecs.Component
    StatusEffectName() string
    Duration() int

    // Apply effect to creature
    ApplyToCreature(c *ecs.QueryResult)

    // Display effect information
    DisplayString() string

    // Stack multiple instances of same effect
    StackEffect(eff any)

    Copy() StatusEffects
    common.Quality
}
```

**Status Effect Implementations:**

**Burning (Damage over Time):**
```go
type Burning struct {
    MainProps   CommonItemProperties
    Temperature int
}

func (b *Burning) ApplyToCreature(c *ecs.QueryResult) {
    b.MainProps.Duration -= 1

    attr := GetComponentType[*Attributes](c.Entity, AttributeComponent)
    attr.CurrentHealth -= b.Temperature
}

func (b *Burning) StackEffect(eff any) {
    e := eff.(*Burning)
    b.MainProps.AddDuration(e.MainProps)  // Add duration
    b.Temperature += e.Temperature        // Increase temperature
}
```

**Freezing (Disable Action):**
```go
type Freezing struct {
    MainProps CommonItemProperties
    Thickness int
}

func (f *Freezing) ApplyToCreature(c *ecs.QueryResult) {
    attr := GetComponentType[*Attributes](c.Entity, AttributeComponent)

    if f.MainProps.Duration > 0 {
        attr.CanAct = false  // Frozen
    } else {
        attr.CanAct = true   // Thawed
    }

    f.MainProps.Duration -= 1
}
```

**Sticky (Movement Debuff):**
```go
type Sticky struct {
    MainProps CommonItemProperties
    Spread    int
}

func (s *Sticky) ApplyToCreature(c *ecs.QueryResult) {
    var originalMovementSpeed int
    var initialized bool

    attr := GetComponentType[*Attributes](c.Entity, AttributeComponent)

    if !initialized {
        originalMovementSpeed = attr.TotalMovementSpeed
        initialized = true
    }

    attr.TotalMovementSpeed -= 5
    if attr.TotalMovementSpeed <= 0 {
        attr.TotalMovementSpeed = 1
    }

    s.MainProps.Duration--

    if s.MainProps.Duration == 0 {
        attr.TotalMovementSpeed = originalMovementSpeed
    }
}
```

### 11.5 Effect Application Flow

```
Throwable Item Used
      ↓
ThrowableAction.Execute()
      ├── Display VX on AOE tiles
      ├── Find creatures in AOE
      └── Return effects to apply
      ↓
[Effects stored in creature components]
      ↓
[Each turn]
RunEffectTracker()
      ↓
For each creature with effects:
  ├── Get all status effects
  ├── For each effect:
  │    ├── ApplyToCreature()
  │    ├── Decrement duration
  │    └── Remove if expired
  └── Update creature attributes
```

### 11.6 Quality System Integration

Both ItemActions and StatusEffects implement `common.Quality`:

```go
type Quality interface {
    CreateWithQuality(q QualityType)
}

type QualityType int
const (
    LowQuality = iota
    NormalQuality
    HighQuality
)
```

**Quality Application:**
```go
// ThrowableAction quality affects range, damage, and shape
func (t *ThrowableAction) CreateWithQuality(q QualityType) {
    t.MainProps = t.MainProps.CreateWithQuality(q)

    if q == LowQuality {
        t.ThrowingRange = 3 + rand.Intn(2)  // 3-4
        t.Damage = 1 + rand.Intn(2)         // 1-2
    } else if q == NormalQuality {
        t.ThrowingRange = 5 + rand.Intn(3)  // 5-7
        t.Damage = 2 + rand.Intn(3)         // 2-4
    } else if q == HighQuality {
        t.ThrowingRange = 8 + rand.Intn(4)  // 8-11
        t.Damage = 4 + rand.Intn(4)         // 4-7
    }

    // Shape quality is set when shape is created
}
```

---

## 12. Game Loop and Rendering

### 12.1 Main Game Loop

**File:** `game_main/main.go`

**Ebiten Game Interface:**
```go
type Game struct {
    em               common.EntityManager
    gameUI           gui.PlayerUI
    playerData       avatar.PlayerData
    gameMap          worldmap.GameMap
    inputCoordinator *input.InputCoordinator
}

// Ebiten interface methods
func (g *Game) Update() error
func (g *Game) Draw(screen *ebiten.Image)
func (g *Game) Layout(w, h int) (int, int)
```

### 12.2 Update Cycle

**Update Flow:**
```go
func (g *Game) Update() error {
    // 1. Update UI state
    g.gameUI.MainPlayerInterface.Update()

    // 2. Position UI elements
    gui.SetContainerLocation(g.gameUI.StatsUI.StatUIContainer,
                           g.gameMap.RightEdgeX, 0)

    // 3. Update visual effects
    graphics.VXHandler.UpdateVisualEffects()

    // 4. Handle debug input
    input.PlayerDebugActions(&g.playerData)

    // 5. Handle main game input
    HandleInput(g)

    return nil
}
```

**Input Handling:**
```go
func HandleInput(g *Game) {
    // Update player stats from equipment
    gear.UpdateEntityAttributes(g.playerData.PlayerEntity)
    g.gameUI.StatsUI.StatsTextArea.SetText(g.playerData.PlayerAttributes().DisplayString())

    // Process input through coordinator
    g.inputCoordinator.HandleInput()

    // If player took action, run effects
    if g.playerData.InputStates.HasKeyInput {
        gear.RunEffectTracker(g.playerData.PlayerEntity)
        g.gameUI.StatsUI.StatsTextArea.SetText(g.playerData.PlayerAttributes().DisplayString())
        g.playerData.InputStates.HasKeyInput = false
    }

    // Clean up dead entities
    resmanager.RemoveDeadEntities(&g.em, &g.gameMap)
}
```

### 12.3 Draw Cycle

**Rendering Order:**
```go
func (g *Game) Draw(screen *ebiten.Image) {
    // 1. Get screen dimensions
    graphics.ScreenInfo.ScreenWidth = screen.Bounds().Dx()
    graphics.ScreenInfo.ScreenHeight = screen.Bounds().Dy()

    // 2. Draw map tiles (either full map or centered square)
    if graphics.MAP_SCROLLING_ENABLED {
        g.gameMap.DrawLevelCenteredSquare(screen, g.playerData.Pos,
                                        graphics.ViewableSquareSize, DEBUG_MODE)
        rendering.ProcessRenderablesInSquare(&g.em, g.gameMap, screen,
                                           g.playerData.Pos, graphics.ViewableSquareSize, DEBUG_MODE)
    } else {
        g.gameMap.DrawLevel(screen, DEBUG_MODE)
        rendering.ProcessRenderables(&g.em, g.gameMap, screen, DEBUG_MODE)
    }

    // 3. Draw user messages
    gui.ProcessUserLog(g.em, screen, &g.gameUI.MsgUI)

    // 4. Draw visual effects
    graphics.VXHandler.DrawVisualEffects(screen)

    // 5. Draw UI
    g.gameUI.MainPlayerInterface.Draw(screen)
}
```

### 12.4 Map Rendering

**Full Map Rendering:**
```go
func (gameMap *GameMap) DrawLevel(screen *ebiten.Image, revealAllTiles bool) {
    for x := 0; x < ScreenInfo.DungeonWidth; x++ {
        for y := 0; y < ScreenInfo.DungeonHeight; y++ {
            logicalPos := LogicalPosition{X: x, Y: y}
            idx := CoordManager.LogicalToIndex(logicalPos)
            tile := gameMap.Tiles[idx]

            isVis := gameMap.PlayerVisible.IsVisible(x, y)
            if revealAllTiles { isVis = true }

            op := &ebiten.DrawImageOptions{}

            if isVis {
                op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
                tile.IsRevealed = true
            } else if tile.IsRevealed {
                op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
                // Darken out-of-FOV tiles
                op.ColorScale.ScaleWithColor(color.RGBA{1, 1, 1, 1})
            }

            // Apply color matrix (for AOE highlighting, etc.)
            if !tile.cm.IsEmpty() {
                cs.SetR(tile.cm.R)
                cs.SetG(tile.cm.G)
                cs.SetB(tile.cm.B)
                cs.SetA(tile.cm.A)
                op.ColorScale.ScaleWithColorScale(cs)
            }

            screen.DrawImage(tile.image, op)
        }
    }
}
```

**Scrolling Map (Centered on Player):**
```go
func (gameMap *GameMap) DrawLevelCenteredSquare(screen *ebiten.Image,
                                                 playerPos *LogicalPosition,
                                                 size int, revealAllTiles bool) {
    sq := coords.NewDrawableSection(playerPos.X, playerPos.Y, size)

    // Track right edge for UI positioning
    gameMap.RightEdgeX = 0

    for x := sq.StartX; x <= sq.EndX; x++ {
        for y := sq.StartY; y <= sq.EndY; y++ {
            logicalPos := LogicalPosition{X: x, Y: y}
            idx := CoordManager.LogicalToIndex(logicalPos)
            tile := gameMap.Tiles[idx]

            isVis := gameMap.PlayerVisible.IsVisible(x, y)
            if revealAllTiles { isVis = true }

            if isVis || tile.IsRevealed {
                op := &ebiten.DrawImageOptions{}

                // Apply scaling
                op.GeoM.Scale(float64(ScreenInfo.ScaleFactor),
                            float64(ScreenInfo.ScaleFactor))

                // Center on player
                offsetX, offsetY := graphics.OffsetFromCenter(playerPos.X, playerPos.Y,
                                                            tile.PixelX, tile.PixelY,
                                                            ScreenInfo)
                op.GeoM.Translate(offsetX, offsetY)

                // Track right edge
                tileRightEdge := int(offsetX + float64(tile.image.Bounds().Dx()*ScreenInfo.ScaleFactor))
                if tileRightEdge > gameMap.RightEdgeX {
                    gameMap.RightEdgeX = tileRightEdge
                }

                screen.DrawImage(tile.image, op)
            }
        }
    }
}
```

### 12.5 Entity Rendering

**File:** `rendering/rendering.go`

```go
func ProcessRenderables(ecsmanager *EntityManager, gameMap GameMap,
                        screen *ebiten.Image, debugMode bool) {
    for _, result := range ecsmanager.World.Query(ecsmanager.WorldTags["renderables"]) {
        pos := result.Components[PositionComponent].(*LogicalPosition)
        renderable := result.Components[RenderableComponent].(*Renderable)

        if !renderable.Visible { continue }

        // FOV check (or debug mode override)
        if debugMode || gameMap.PlayerVisible.IsVisible(pos.X, pos.Y) {
            // Get tile pixel position
            index := CoordManager.LogicalToIndex(*pos)
            tile := gameMap.Tiles[index]

            op := &ebiten.DrawImageOptions{}
            op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
            screen.DrawImage(renderable.Image, op)
        }
    }
}
```

### 12.6 Field of View (FOV) System

**Integration:** Uses `github.com/norendren/go-fov/fov`

**FOV Update:**
```go
type GameMap struct {
    PlayerVisible *fov.View
    // ... other fields
}

// Initialize
dungeonMap.PlayerVisible = fov.New()

// Update FOV when player moves
dungeonMap.PlayerVisible.Compute(playerPos.X, playerPos.Y, viewRadius)

// Check visibility
if gameMap.PlayerVisible.IsVisible(x, y) {
    // Tile is visible
}
```

**Tile Revelation:**
```go
type Tile struct {
    IsRevealed bool  // Has player seen this tile?
    // ... other fields
}

// When tile becomes visible
if isVis {
    tile.IsRevealed = true
}

// Draw revealed but out-of-FOV tiles (darkened)
if tile.IsRevealed && !isVis {
    op.ColorScale.ScaleWithColor(color.RGBA{1, 1, 1, 1})
    screen.DrawImage(tile.image, op)
}
```

### 12.7 Performance Considerations

**Optimization Techniques:**

1. **Spatial Partitioning:**
```go
// Only render entities in viewable square
if pos.X >= sq.StartX && pos.X <= sq.EndX &&
   pos.Y >= sq.StartY && pos.Y <= sq.EndY {
    // Render
}
```

2. **FOV Culling:**
```go
// Skip entities outside player's view
if !gameMap.PlayerVisible.IsVisible(pos.X, pos.Y) {
    continue
}
```

3. **Component Queries:**
```go
// Query only entities with required components
renderables := ecs.BuildTag(RenderableComponent, PositionComponent)
for _, result := range manager.World.Query(renderables) {
    // Process
}
```

4. **Visual Effect Cleanup:**
```go
func (vis *VisualEffectHandler) clearVisualEffects() {
    remainingEffects := make([]VisualEffect, 0)

    for _, v := range vis.vx {
        if !v.IsCompleted() {
            remainingEffects = append(remainingEffects, v)
        }
    }

    vis.vx = remainingEffects
}
```

---

## 13. Development Roadmap

### 13.1 Completed Simplifications

#### ✅ Input System Consolidation (100%)
**Achievement:** Eliminated scattered global state
- **Before:** Input handling spread across multiple files
- **After:** Unified InputCoordinator with priority-based controllers
- **Impact:** Clear separation of concerns, testable controllers

#### ✅ Coordinate System Standardization (100%)
**Achievement:** Unified CoordinateManager replaces 73+ scattered calls
- **Before:** `CoordTransformer` calls throughout codebase
- **After:** Global `CoordManager` with type-safe positions
- **Impact:** Single source of truth for coordinate conversions

#### ✅ Graphics Shape System (95%)
**Achievement:** 8+ shape types → 1 BaseShape with variants
- **Before:** Circle, Circle2, Circle3, Square, Rectangle, Line, Cone (all separate)
- **After:** `BaseShape` with `BasicShapeType` (Circular, Rectangular, Linear)
- **Impact:** Massive code deduplication, quality system integration
- **Remaining:** Extract direction system to separate file (5%)

#### ✅ Entity Template System (100%)
**Achievement:** 4 specialized functions → 1 generic factory
- **Before:** `CreateMonsterFromTemplate()`, `CreateMeleeWeaponFromTemplate()`, etc.
- **After:** `CreateEntityFromTemplate()` with `EntityType` enum
- **Impact:** Type-safe entity creation, spawning system unblocked

#### 🟡 Status Effects vs Item Behaviors (85%)
**Achievement:** Conceptual separation of effects from actions
- **Before:** Throwables forced into StatusEffect interface
- **After:** `ItemAction` interface with composition pattern
- **Impact:** Throwables contain effects (not "are" effects)
- **Remaining:** Extract quality interface for true separation (15%)

### 13.2 In Progress

#### ❌ GUI Button Factory (10%)
**Target:** 3 duplicate functions → 1 factory
- **Current:** Basic `CreateButton()` exists
- **Goal:** `CreateMenuButton(config)` with `ButtonConfig` struct
- **Impact:** -35 LOC (23% reduction in playerUI.go)

### 13.3 Future Roadmap

#### 🔄 Squad Combat Foundation (0%)
**Critical Path Item** - Blocks AI system, balance, spawning quality
- **Problem:** Current 1v1 combat incompatible with squad-based gameplay
- **Goal:** "Command several squads" (todos.txt:25)
- **Approach:** Incremental phases
  - Phase 1: PlayerSquad wrapper (backward compatible)
  - Phase 2: Multi-squad support, formations
  - Phase 3: Squad-aware spawning and AI integration
- **Estimated Effort:** 12-40 hours

### 13.4 Overall Progress

**Roadmap Completion:** 80% (weighted average)
- **Fully Complete:** 4 of 6 items
- **In Progress:** 1 of 6 items (Status Effects 85%)
- **Minimal Progress:** 1 of 6 items (GUI Buttons 10%)

### 13.5 Design Principles Established

1. **Composition Over Inheritance:** ECS architecture, ComponentAdder pattern
2. **Strategy Pattern for Variability:** Visual effects (Animator + Renderer)
3. **Factory Pattern for Creation:** Unified entity template factory
4. **Coordinator Pattern for Input:** Priority-based input routing
5. **Type Safety:** Generic helpers, enum-based configuration
6. **Single Responsibility:** Each package has clear purpose
7. **Centralized Management:** CoordManager, VXHandler, InputCoordinator

---

## 14. Appendices

### A. File Reference

**Core Game Loop:**
- `game_main/main.go` - Main entry point, game loop
- `game_main/gameinit.go` - Player and entity initialization
- `game_main/componentinit.go` - ECS component registration

**ECS Foundation:**
- `common/commontypes.go` - Core types (Attributes, Name, UserMessage)
- `common/commoncomponents.go` - Component definitions
- `common/ecsutil.go` - EntityManager, type-safe access

**Coordinate System:**
- `coords/cordmanager.go` - CoordinateManager, Viewport
- `coords/position.go` - LogicalPosition, PixelPosition

**Graphics:**
- `graphics/drawableshapes.go` - Unified shape system (BaseShape)
- `graphics/vx.go` - Visual effects (Animator + Renderer pattern)
- `graphics/colormatrix.go` - Color transformations
- `graphics/graphictypes.go` - Shared graphics types

**Rendering:**
- `rendering/rendering.go` - Entity rendering, FOV integration

**World Management:**
- `worldmap/dungeongen.go` - Procedural dungeon generation
- `worldmap/dungeontile.go` - Tile types and properties
- `worldmap/GameMapUtil.go` - Map utilities

**Combat:**
- `combat/attackingsystem.go` - Melee and ranged attacks

**Equipment and Items:**
- `gear/items.go` - Base item types
- `gear/equipmentcomponents.go` - Weapon, armor components
- `gear/Inventory.go` - Inventory management
- `gear/stateffect.go` - Status effects (Burning, Freezing, Sticky)
- `gear/itemactions.go` - Item actions (ThrowableAction)
- `gear/consumables.go` - Consumable items
- `gear/itemquality.go` - Quality system

**Input:**
- `input/inputcoordinator.go` - Main coordinator
- `input/movementcontroller.go` - Movement handling
- `input/combatcontroller.go` - Combat handling
- `input/uicontroller.go` - UI handling
- `input/debuginput.go` - Debug commands

**Entity Creation:**
- `entitytemplates/creators.go` - Factory functions
- `entitytemplates/readdata.go` - JSON loading
- `entitytemplates/jsonstructs.go` - Template structures
- `entitytemplates/templatelib.go` - Template library

**Spawning:**
- `spawning/spawnmonsters.go` - Monster spawning
- `spawning/spawnloot.go` - Loot spawning
- `spawning/spawnthrowable.go` - Throwable spawning
- `spawning/probtables.go` - Probability tables
- `spawning/loottables.go` - Loot tables

**UI:**
- `gui/playerUI.go` - Main player interface
- `gui/itemui.go` - Item display
- `gui/statsui.go` - Stats display
- `gui/messagesUI.go` - Message log
- `gui/equipmentUI.go` - Equipment panel
- `gui/consumableUI.go` - Consumable panel
- `gui/throwingUI.go` - Throwing interface

**Player:**
- `avatar/playerdata.go` - Player data, input states, equipment

**Monsters:**
- `monsters/creatures.go` - Creature component

**Utilities:**
- `randgen/randnumgen.go` - Random number generation
- `pathfinding/astar.go` - A* pathfinding
- `resourcemanager/cleanup.go` - Entity cleanup
- `trackers/creaturetracker.go` - Creature tracking
- `trackers/statustracker.go` - Status effect tracking
- `timesystem/initiative.go` - Turn/initiative system
- `testing/testingdata.go` - Test utilities

### B. Key Design Decisions

**Why ECS?**
- Flexible entity composition without inheritance hierarchies
- Data-oriented design for performance
- Easy to add/remove capabilities at runtime
- Natural fit for component-based game entities

**Why InputCoordinator?**
- Eliminated scattered global input state
- Clear priority ordering prevents conflicts
- Each controller independently testable
- Shared state managed explicitly

**Why Unified Coordinate System?**
- Before: 73+ scattered CoordTransformer calls
- After: Single global CoordManager
- Type-safe position wrappers prevent logical/pixel confusion
- Viewport system for camera/scrolling

**Why Separate ItemAction from StatusEffect?**
- Throwables are actions that trigger effects, not effects themselves
- Composition: Throwable "has" effects, not "is" effect
- Cleaner conceptual model
- Easier to extend with new action types

**Why Strategy Pattern for Visual Effects?**
- Separates "what changes" (Animator) from "how to draw" (Renderer)
- Easy to create new effects by combining existing strategies
- Stateless renderers can be shared
- Animation state calculated per-frame from elapsed time

### C. Common Patterns

**Type-Safe Component Access:**
```go
attr := common.GetComponentType[*Attributes](entity, AttributeComponent)
pos := common.GetPosition(entity)  // Convenience wrapper
```

**Entity Queries:**
```go
for _, result := range manager.World.Query(manager.WorldTags["monsters"]) {
    entity := result.Entity
    pos := result.Components[PositionComponent].(*LogicalPosition)
}
```

**Coordinate Conversions:**
```go
index := coords.CoordManager.LogicalToIndex(logicalPos)
pixelPos := coords.CoordManager.LogicalToPixel(logicalPos)
logicalPos := coords.CoordManager.PixelToLogical(pixelPos)
```

**Entity Creation from Template:**
```go
config := EntityConfig{
    Type: EntityCreature,
    Name: template.Name,
    // ... other config
}
entity := CreateEntityFromTemplate(manager, config, templateData)
```

**Visual Effect Creation:**
```go
vx := graphics.NewFireEffect(x, y, duration)
graphics.AddVX(vx)  // Add to global handler

// Or area effect
vxArea := graphics.NewVisualEffectArea(centerX, centerY, shape, vx)
graphics.AddVXArea(vxArea)
```

### D. Debugging Tips

**Debug Mode:**
```go
var DEBUG_MODE = false  // Set to true for debugging

// Reveals all tiles (no FOV)
gameMap.DrawLevel(screen, DEBUG_MODE)

// Renders all entities (no FOV)
rendering.ProcessRenderables(ecsManager, gameMap, screen, DEBUG_MODE)
```

**Benchmarking:**
```go
var ENABLE_BENCHMARKING = false  // Enable profiling

func BenchmarkSetup() {
    if ENABLE_BENCHMARKING {
        go func() {
            http.ListenAndServe("localhost:6060", nil)
        }()
        runtime.SetCPUProfileRate(1000)
        runtime.MemProfileRate = 1
    }
}

// Access pprof at http://localhost:6060/debug/pprof/
```

**Common Issues:**

1. **Coordinate Confusion:** Always use CoordManager conversions
2. **Missing Components:** Use type-safe GetComponentType with panic recovery
3. **Dead Entity Cleanup:** RemoveDeadEntities() called in HandleInput()
4. **FOV Not Updating:** Ensure PlayerVisible.Compute() called after player moves
5. **Visual Effects Lingering:** VXHandler.clearVisualEffects() removes completed effects

### E. Performance Metrics

**Target Performance:**
- **60 FPS** with 30+ entities on screen
- **Map Size:** 100x80 tiles (8000 tiles)
- **FOV Radius:** 10-15 tiles
- **Visual Effects:** 10+ concurrent effects

**Optimization Points:**
- Spatial culling (only render viewable square)
- FOV culling (skip out-of-view entities)
- Component queries (only fetch needed entities)
- Effect cleanup (remove completed effects)
- Coordinate caching (avoid redundant conversions)

### F. Extension Points

**Adding New Entity Types:**
1. Define JSON structure in `entitytemplates/jsonstructs.go`
2. Add `EntityType` enum value
3. Create `ComponentAdder` function
4. Add case to `CreateEntityFromTemplate()` switch

**Adding New Status Effects:**
1. Implement `StatusEffects` interface
2. Add component to `InitializeItemComponents()`
3. Add to `AllItemEffects` slice
4. Create visual effect in `GetVisualEffect()`

**Adding New Input Controllers:**
1. Implement `InputController` interface
2. Add to `InputCoordinator` struct
3. Add to `HandleInput()` priority chain
4. Share state via `SharedInputState`

**Adding New Visual Effects:**
1. Create new `Animator` implementation
2. Create new `Renderer` implementation (if needed)
3. Combine in factory function (e.g., `NewMyEffect()`)
4. Use existing `BaseEffect` for lifecycle

---

## Conclusion

TinkerRogue represents a well-architected tactical roguelike built on solid design principles:

- **Clean ECS Architecture:** Composition-based entities with clear component/system separation
- **Unified Systems:** CoordinateManager, InputCoordinator, BaseShape, and VisualEffect handlers eliminate scattered logic
- **Type Safety:** Generic helpers and enum-based configuration prevent runtime errors
- **Extensibility:** Strategy patterns and factory methods make adding content straightforward
- **Performance:** Spatial culling, FOV integration, and efficient queries maintain 60 FPS

The codebase has achieved **80% completion** of its simplification roadmap, with robust systems in place for input handling, coordinate management, graphics rendering, and entity creation. The remaining 20% focuses on higher-level features like squad combat and UI refinement.

This documentation serves as a comprehensive reference for understanding the codebase architecture, design patterns, and implementation details necessary for onboarding new developers or extending the game's functionality.

---

**For More Information:**
- Project Roadmap: `CLAUDE.md`
- Refactoring Priorities: `analysis/REFACTORING_PRIORITIES.md`
- Combat Architecture: `analysis/combat_refactoring.md`
- Development Workflows: `DEVELOPMENT_WORKFLOWS.md`
