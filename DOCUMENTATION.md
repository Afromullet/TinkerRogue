# TinkerRogue: Technical Documentation

**Game Type:** Turn-based Tactical Roguelike
**Engine:** Ebiten v2 (2D Game Engine for Go)
**Architecture:** Entity Component System (ECS) using bytearena/ecs
**Language:** Go 1.x
**Last Updated:** 2025-10-08

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Core Game Systems](#core-game-systems)
4. [Package Documentation](#package-documentation)
5. [Game-Specific Patterns](#game-specific-patterns)
6. [Asset Pipeline](#asset-pipeline)
7. [Data Structures & Formats](#data-structures--formats)

---

## Executive Summary

TinkerRogue is a turn-based tactical roguelike built on the Ebiten 2D game engine, implementing a data-driven Entity Component System architecture. The game features procedurally generated dungeons, grid-based tactical combat, equipment systems with status effects, and an emerging squad-based tactical combat system.

### Key Technical Features

- **Pure ECS Architecture**: Entities are data containers with component-based composition
- **Data-Driven Design**: JSON-based entity templates for monsters, weapons, consumables
- **Coordinate System**: Unified coordinate management with logical/pixel/index transformations
- **Visual Effects System**: Tile-based shapes (circles, lines, cones) for combat feedback
- **Input Coordination**: Modular input handling with prioritized controllers (UI > Combat > Movement)
- **Template Factory System**: Generic entity creation with type-safe configuration patterns
- **Squad Combat System**: 3x3 grid tactical formations with role-based combat (in development)

### Game Loop Summary

```
Initialize ECS → Load Templates → Generate Dungeon → Spawn Entities → Game Loop:
  ├─ Update()
  │  ├─ UI Updates (Ebiten widgets)
  │  ├─ Visual Effects Updates
  │  ├─ Input Handling (InputCoordinator)
  │  ├─ Player Actions
  │  └─ Cleanup Dead Entities
  └─ Draw()
     ├─ Map Rendering
     ├─ Entity Rendering (renderables)
     ├─ UI Rendering
     └─ Visual Effects Rendering
```

---

## Architecture Overview

### High-Level Architecture

TinkerRogue uses a **component-based Entity Component System (ECS)** where:

1. **Entities** are opaque handles (`*ecs.Entity`) with no intrinsic behavior
2. **Components** are pure data structures registered with the ECS manager
3. **Systems** are functions that query entities by component tags and execute logic
4. **Tags** are component filters used for efficient entity queries

```
┌─────────────────────────────────────────────────────────────┐
│                      Game Loop (main.go)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │   Update()   │  │    Draw()    │  │   Layout()      │   │
│  └──────┬───────┘  └──────┬───────┘  └─────────────────┘   │
└─────────┼──────────────────┼────────────────────────────────┘
          │                  │
          ▼                  ▼
┌─────────────────┐  ┌─────────────────────────────────────┐
│ InputCoordinator│  │      Rendering System               │
│  ├─ Movement    │  │  ├─ Map Rendering                   │
│  ├─ Combat      │  │  ├─ Entity Rendering (Query)        │
│  └─ UI          │  │  ├─ UI Rendering                    │
└────────┬────────┘  │  └─ Visual Effects Rendering        │
         │           └─────────────────────────────────────┘
         ▼
┌─────────────────────────────────────────────────────────────┐
│                  ECS Manager (ecs.Manager)                   │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Entities: []Entity (opaque handles)                  │  │
│  │  Components: map[ComponentType]ComponentData          │  │
│  │  Tags: map[string]Tag (query filters)                 │  │
│  └───────────────────────────────────────────────────────┘  │
│  Query API: World.Query(tag) → []QueryResult              │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│              Component Layer (Data Structures)               │
│  ├─ Position, Attributes, Name (common/)                    │
│  ├─ Renderable (rendering/)                                 │
│  ├─ Creature, Item, Weapon, Armor (gear/, monsters/)        │
│  ├─ Squad Components (squads/)                              │
│  └─ UI Components (gui/)                                    │
└─────────────────────────────────────────────────────────────┘
```

### ECS Implementation

The game uses [bytearena/ecs](https://github.com/bytearena/ecs) library with the following structure:

```go
// EntityManager wraps the ECS library
type EntityManager struct {
    World     *ecs.Manager          // ECS world manager
    WorldTags map[string]ecs.Tag     // Query tags (e.g., "monsters", "renderables")
}

// Component registration (componentinit.go)
common.PositionComponent = manager.NewComponent()
rendering.RenderableComponent = manager.NewComponent()
gear.ItemComponent = manager.NewComponent()
// ... etc

// Tag creation for efficient queries
renderables := ecs.BuildTag(rendering.RenderableComponent, common.PositionComponent)
monsters := ecs.BuildTag(monsters.CreatureComponent, common.PositionComponent, common.AttributeComponent)

// Querying entities
for _, result := range ecsManager.World.Query(ecsManager.WorldTags["monsters"]) {
    pos := result.Components[common.PositionComponent].(*coords.LogicalPosition)
    attr := result.Components[common.AttributeComponent].(*common.Attributes)
    // ... process monster
}
```

### Key Architectural Patterns

#### 1. **Component Composition Pattern**

Entities are built by composing components:

```go
entity := manager.NewEntity().
    AddComponent(common.NameComponent, &common.Name{NameStr: "Goblin"}).
    AddComponent(common.PositionComponent, &coords.LogicalPosition{X: 10, Y: 10}).
    AddComponent(common.AttributeComponent, &common.Attributes{MaxHealth: 15}).
    AddComponent(monsters.CreatureComponent, &monsters.Creature{}).
    AddComponent(rendering.RenderableComponent, &rendering.Renderable{Image: img})
```

#### 2. **Template Factory Pattern**

Generic entity creation using configuration structs (entitytemplates/creators.go):

```go
type EntityConfig struct {
    Type      EntityType              // Melee, Ranged, Consumable, Creature
    Name      string
    ImagePath string
    AssetDir  string
    Visible   bool
    Position  *coords.LogicalPosition
    GameMap   *worldmap.GameMap       // Optional for creatures
}

entity := CreateEntityFromTemplate(manager, config, jsonData)
```

#### 3. **Coordinate Manager Pattern**

Unified coordinate transformation service (coords/cordmanager.go):

```go
// Global singleton
var CoordManager *CoordinateManager

// Conversions
logicalPos := LogicalPosition{X: 10, Y: 5}
pixelPos := CoordManager.LogicalToPixel(logicalPos)
index := CoordManager.LogicalToIndex(logicalPos)
```

#### 4. **Input Coordinator Pattern**

Prioritized input handling with controller chain (input/inputcoordinator.go):

```go
type InputCoordinator struct {
    movementController *MovementController
    combatController   *CombatController
    uiController       *UIController
}

// Priority: UI > Combat > Movement
func (ic *InputCoordinator) HandleInput() bool {
    if ic.uiController.CanHandle() {
        return ic.uiController.HandleInput()
    }
    if ic.combatController.CanHandle() {
        return ic.combatController.HandleInput()
    }
    return ic.movementController.HandleInput()
}
```

---

## Core Game Systems

### 1. Game Loop & State Management

**Location:** `game_main/main.go`

The main game loop follows Ebiten's `Game` interface pattern:

```go
type Game struct {
    em               common.EntityManager      // ECS manager
    gameUI           gui.PlayerUI              // UI state
    playerData       avatar.PlayerData         // Player state
    gameMap          worldmap.GameMap          // Level data
    inputCoordinator *input.InputCoordinator   // Input handler
}

// Ebiten interface methods
func (g *Game) Update() error           // Called every frame (~60 FPS)
func (g *Game) Draw(screen *ebiten.Image)  // Rendering
func (g *Game) Layout(w, h int) (int, int) // Screen dimensions
```

**Initialization Flow:**

1. **ECS Setup** (`InitializeECS`): Register all components and tags
2. **Data Loading** (`entitytemplates.ReadGameData`): Load JSON templates
3. **Map Generation** (`worldmap.NewGameMap`): Create dungeon
4. **Entity Spawning** (`spawning.SpawnStartingCreatures`): Populate map
5. **UI Creation** (`gui.CreateMainInterface`): Build UI widgets
6. **Input Setup** (`input.NewInputCoordinator`): Initialize controllers

**Update Loop:**

```go
func (g *Game) Update() error {
    // 1. Update UI widgets
    g.gameUI.MainPlayerInterface.Update()

    // 2. Position UI elements
    gui.SetContainerLocation(g.gameUI.StatsUI.StatUIContainer, ...)

    // 3. Update visual effects
    graphics.VXHandler.UpdateVisualEffects()

    // 4. Debug input
    input.PlayerDebugActions(&g.playerData)

    // 5. Main input handling
    HandleInput(g)

    return nil
}
```

**Draw Loop:**

```go
func (g *Game) Draw(screen *ebiten.Image) {
    // 1. Update screen dimensions
    graphics.ScreenInfo.ScreenWidth = screen.Bounds().Dx()
    graphics.ScreenInfo.ScreenHeight = screen.Bounds().Dy()

    // 2. Map rendering (two modes)
    if graphics.MAP_SCROLLING_ENABLED {
        g.gameMap.DrawLevelCenteredSquare(screen, playerPos, viewSize, DEBUG_MODE)
        rendering.ProcessRenderablesInSquare(...)
    } else {
        g.gameMap.DrawLevel(screen, DEBUG_MODE)
        rendering.ProcessRenderables(...)
    }

    // 3. UI rendering
    gui.ProcessUserLog(...)

    // 4. Visual effects
    graphics.VXHandler.DrawVisualEffects(screen)

    // 5. Widget rendering
    g.gameUI.MainPlayerInterface.Draw(screen)
}
```

---

### 2. Entity/GameObject System (ECS)

**Location:** `common/ecsutil.go`, `common/commoncomponents.go`, `game_main/componentinit.go`

#### Core Components

**Position Component:**
```go
type LogicalPosition struct {
    X int
    Y int
}

func (p *LogicalPosition) IsEqual(other *LogicalPosition) bool
func (p *LogicalPosition) ChebyshevDistance(other *LogicalPosition) int
func (p *LogicalPosition) InRange(other *LogicalPosition, maxRange int) bool
```

**Attributes Component:**
```go
type Attributes struct {
    // Base stats
    MaxHealth         int
    CurrentHealth     int
    AttackBonus       int
    BaseArmorClass    int
    BaseProtection    int
    BaseMovementSpeed int
    BaseDodgeChance   float32
    DamageBonus       int

    // Calculated totals (from equipment)
    TotalArmorClass    int
    TotalProtection    int
    TotalDodgeChance   float32
    TotalMovementSpeed int
    TotalAttackSpeed   int

    CanAct bool
}

func NewBaseAttributes(maxHealth, attackBonus, baseAC, baseProt, baseMovSpeed int,
                       dodge float32, damageBonus int) Attributes
func (a Attributes) DisplayString() string  // UI formatting
```

**Name Component:**
```go
type Name struct {
    NameStr string
}
```

**User Message Component:**
```go
type UserMessage struct {
    AttackMessage       string
    GameStateMessage    string
    StatusEffectMessage string
}
```

#### Component Registration

All components are registered in `game_main/componentinit.go`:

```go
func InitializeECS(ecsmanager *common.EntityManager) {
    manager := ecs.NewManager()
    tags := make(map[string]ecs.Tag)

    // Register components
    common.PositionComponent = manager.NewComponent()
    rendering.RenderableComponent = manager.NewComponent()
    common.NameComponent = manager.NewComponent()
    common.AttributeComponent = manager.NewComponent()
    gear.InventoryComponent = manager.NewComponent()
    gear.MeleeWeaponComponent = manager.NewComponent()
    gear.RangedWeaponComponent = manager.NewComponent()
    gear.ArmorComponent = manager.NewComponent()
    monsters.CreatureComponent = manager.NewComponent()

    // Create query tags
    renderables := ecs.BuildTag(rendering.RenderableComponent, common.PositionComponent)
    tags["renderables"] = renderables

    creatures := ecs.BuildTag(monsters.CreatureComponent, common.PositionComponent,
                               common.AttributeComponent)
    tags["monsters"] = creatures

    // Initialize subsystems
    gear.InitializeItemComponents(manager, tags)

    ecsmanager.World = manager
    ecsmanager.WorldTags = tags
}
```

#### Helper Functions

**Type-Safe Component Access:**

```go
// Generic component getter with panic recovery
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {
    defer func() {
        if r := recover(); r != nil {
            // Error handling
        }
    }()

    if c, ok := entity.GetComponentData(component); ok {
        return c.(T)
    }
    var nilValue T
    return nilValue
}

// Convenience wrappers
func GetAttributes(e *ecs.Entity) *Attributes
func GetPosition(e *ecs.Entity) *coords.LogicalPosition
```

**Spatial Queries:**

```go
// Find creature at position (O(n) - needs optimization)
func GetCreatureAtPosition(ecsmanager *EntityManager, pos *coords.LogicalPosition) *ecs.Entity {
    for _, c := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {
        curPos := GetPosition(c.Entity)
        if pos.IsEqual(curPos) {
            return c.Entity
        }
    }
    return nil
}

// Calculate distance between entities
func DistanceBetween(e1 *ecs.Entity, e2 *ecs.Entity) int {
    pos1 := GetPosition(e1)
    pos2 := GetPosition(e2)
    return pos1.ChebyshevDistance(pos2)
}
```

---

### 3. Input System

**Location:** `input/inputcoordinator.go`, `input/movementcontroller.go`, `input/combatcontroller.go`, `input/uicontroller.go`

The input system uses a **coordinator pattern** with three specialized controllers:

```
InputCoordinator
├─ UIController       (Priority 1: Highest)
│  ├─ Inventory menus
│  ├─ Equipment screens
│  └─ Message logs
├─ CombatController   (Priority 2: Medium)
│  ├─ Throwing items
│  └─ Ranged attacks
└─ MovementController (Priority 3: Lowest)
   ├─ Player movement
   └─ Melee attacks
```

#### InputCoordinator

**Responsibilities:**
- Route input to appropriate controller based on priority
- Manage shared state between controllers
- Coordinate turn-based action flow

```go
type InputCoordinator struct {
    movementController *MovementController
    combatController   *CombatController
    uiController       *UIController
    sharedState        *SharedInputState

    ecsManager *common.EntityManager
    playerData *avatar.PlayerData
    gameMap    *worldmap.GameMap
    playerUI   *gui.PlayerUI
}

type SharedInputState struct {
    PrevCursor         coords.PixelPosition
    PrevThrowInds      []int
    PrevRangedAttInds  []int
    PrevTargetLineInds []int
    TurnTaken          bool
}

func (ic *InputCoordinator) HandleInput() bool {
    // UI has highest priority
    if ic.uiController.CanHandle() {
        return ic.uiController.HandleInput()
    }

    // Combat (throwing/shooting) second
    if ic.combatController.CanHandle() {
        return ic.combatController.HandleInput()
    }

    // Movement/melee last
    if ic.movementController.CanHandle() {
        return ic.movementController.HandleInput()
    }

    return false
}
```

#### Controller Interface

All controllers implement a common interface:

```go
type InputController interface {
    HandleInput() bool      // Process input, return true if handled
    CanHandle() bool        // Check if controller should be active
    OnActivate()           // Called when controller becomes active
    OnDeactivate()         // Called when controller becomes inactive
}
```

#### Movement Controller

**Responsibilities:**
- Arrow key movement
- WASD movement
- Diagonal movement
- Melee attacks on collision
- Pathfinding integration

**Key Functions:**
```go
func (mc *MovementController) HandleInput() bool {
    if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
        mc.tryMove(0, -1)
    }
    // ... other directions
}

func (mc *MovementController) tryMove(dx, dy int) {
    newPos := coords.LogicalPosition{
        X: mc.playerData.Pos.X + dx,
        Y: mc.playerData.Pos.Y + dy,
    }

    if mc.gameMap.IsBlocked(newPos.X, newPos.Y) {
        // Try melee attack if monster present
        monster := common.GetCreatureAtPosition(mc.ecsManager, &newPos)
        if monster != nil {
            combat.MeleeAttackSystem(mc.ecsManager, mc.playerData, mc.gameMap,
                                     mc.playerData.Pos, &newPos)
        }
    } else {
        // Move player
        mc.playerData.Pos = &newPos
        mc.playerData.InputStates.HasKeyInput = true
    }
}
```

#### Combat Controller

**Responsibilities:**
- Throwable item targeting
- Ranged weapon targeting
- Visual feedback (trajectory lines, AOE indicators)
- Rotation of directional shapes

**Key Features:**
- Mouse cursor tracking
- Visual effect rendering for target areas
- Rotation controls (Q/E keys)
- Execution on mouse click

```go
func (cc *CombatController) HandleInput() bool {
    if cc.playerData.InputStates.ThrowingItem {
        cc.handleThrowingInput()
    }
    if cc.playerData.InputStates.ShootingRanged {
        cc.handleShootingInput()
    }
    return cc.CanHandle()
}

func (cc *CombatController) handleThrowingInput() {
    cursorX, cursorY := ebiten.CursorPosition()

    // Get throwable action from selected item
    throwable := item.GetThrowableAction()

    // Update visual feedback
    throwable.UpdateTargetArea(cursorX, cursorY)

    // Rotation controls
    if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
        throwable.RotateLeft()
    }
    if inpututil.IsKeyJustPressed(ebiten.KeyE) {
        throwable.RotateRight()
    }

    // Execute on click
    if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
        throwable.Execute(/* targets */)
        cc.playerData.InputStates.ThrowingItem = false
    }
}
```

#### UI Controller

**Responsibilities:**
- Inventory menu toggle (I key)
- Equipment menu toggle (E key)
- Consumable menu toggle (C key)
- Menu interaction handling

```go
func (uc *UIController) HandleInput() bool {
    if inpututil.IsKeyJustPressed(ebiten.KeyI) {
        uc.toggleInventoryMenu()
    }
    if inpututil.IsKeyJustPressed(ebiten.KeyE) {
        uc.toggleEquipmentMenu()
    }
    if inpututil.IsKeyJustPressed(ebiten.KeyC) {
        uc.toggleConsumableMenu()
    }
    return false
}

func (uc *UIController) CanHandle() bool {
    return uc.playerUI.ItemMenuOpen ||
           uc.playerUI.EquipmentMenuOpen ||
           uc.playerUI.ConsumableMenuOpen
}
```

---

### 4. Rendering/Graphics System

**Location:** `rendering/rendering.go`, `graphics/drawableshapes.go`, `graphics/vx.go`, `graphics/colormatrix.go`

#### Rendering Pipeline

**Entity Rendering:**

```go
// Render all entities with renderable component
func ProcessRenderables(ecsmanager *common.EntityManager, gameMap worldmap.GameMap,
                        screen *ebiten.Image, debugMode bool) {
    for _, result := range ecsmanager.World.Query(ecsmanager.WorldTags["renderables"]) {
        pos := result.Components[common.PositionComponent].(*coords.LogicalPosition)
        renderable := result.Components[RenderableComponent].(*Renderable)

        if !renderable.Visible {
            continue
        }

        // Visibility check (fog of war)
        if debugMode || gameMap.PlayerVisible.IsVisible(pos.X, pos.Y) {
            // Get tile pixel coordinates
            index := coords.CoordManager.LogicalToIndex(*pos)
            tile := gameMap.Tiles[index]

            // Draw sprite
            op := &ebiten.DrawImageOptions{}
            op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
            screen.DrawImage(renderable.Image, op)
        }
    }
}
```

**Viewport Rendering (Scrolling Map):**

```go
// Render only entities in viewport
func ProcessRenderablesInSquare(ecsmanager *common.EntityManager, gameMap worldmap.GameMap,
                                 screen *ebiten.Image, playerPos *coords.LogicalPosition,
                                 squareSize int, debugMode bool) {
    // Calculate viewport bounds
    sq := coords.NewDrawableSection(playerPos.X, playerPos.Y, squareSize)

    // Center viewport on player
    scaledCenterOffsetX := float64(screenWidth)/2 - float64(playerPos.X*scaledTileSize)
    scaledCenterOffsetY := float64(screenHeight)/2 - float64(playerPos.Y*scaledTileSize)

    for _, result := range ecsmanager.World.Query(ecsmanager.WorldTags["renderables"]) {
        pos := result.Components[common.PositionComponent].(*coords.LogicalPosition)

        // Cull entities outside viewport
        if pos.X >= sq.StartX && pos.X <= sq.EndX &&
           pos.Y >= sq.StartY && pos.Y <= sq.EndY {
            // Apply scaling and centering transforms
            op := &ebiten.DrawImageOptions{}
            op.GeoM.Scale(scaleFactor, scaleFactor)
            op.GeoM.Translate(scaledX + offsetX, scaledY + offsetY)
            screen.DrawImage(img, op)
        }
    }
}
```

**Renderable Component:**

```go
type Renderable struct {
    Image   *ebiten.Image
    Visible bool
}
```

#### Visual Effects System

**Location:** `graphics/vx.go`

**VXHandler (Global Singleton):**

```go
var VXHandler VisualEffectHandler

type VisualEffectHandler struct {
    ActiveVXs []VisualEffect
}

type VisualEffect struct {
    Image    *ebiten.Image
    PixelPos coords.PixelPosition
    Duration int        // Frames remaining
    Rotation float64    // Radians
}

func (vxh *VisualEffectHandler) UpdateVisualEffects() {
    // Decrement duration, remove expired effects
    for i := len(vxh.ActiveVXs) - 1; i >= 0; i-- {
        vxh.ActiveVXs[i].Duration--
        if vxh.ActiveVXs[i].Duration <= 0 {
            vxh.ActiveVXs = append(vxh.ActiveVXs[:i], vxh.ActiveVXs[i+1:]...)
        }
    }
}

func (vxh *VisualEffectHandler) DrawVisualEffects(screen *ebiten.Image) {
    for _, vx := range vxh.ActiveVXs {
        op := &ebiten.DrawImageOptions{}
        op.GeoM.Translate(float64(vx.PixelPos.X), float64(vx.PixelPos.Y))
        if vx.Rotation != 0 {
            op.GeoM.Rotate(vx.Rotation)
        }
        screen.DrawImage(vx.Image, op)
    }
}
```

**Usage Example (Shooting VX):**

```go
func (w *RangedWeapon) DisplayShootingVX(start, end *coords.LogicalPosition) {
    vx := VisualEffect{
        Image:    w.ShootingVXImage,
        PixelPos: coords.CoordManager.LogicalToPixel(*start),
        Duration: 15,  // 15 frames (~0.25 seconds at 60 FPS)
        Rotation: calculateAngle(start, end),
    }
    graphics.VXHandler.AddVisualEffect(vx)
}
```

#### Drawable Shapes System

**Location:** `graphics/drawableshapes.go`

**Unified Shape System:**

The graphics package provides a unified shape system for AOE targeting, throwables, and visual effects:

```go
type BasicShapeType int

const (
    Circular    BasicShapeType = iota  // Radius-based (circles)
    Rectangular                        // Width/height (squares/rectangles)
    Linear                             // Length/direction (lines/cones)
)

type BaseShape struct {
    Position   coords.PixelPosition
    Type       BasicShapeType
    Size       int                  // Primary dimension
    Width      int                  // For rectangles
    Height     int                  // For rectangles
    Direction  *ShapeDirection      // For directional shapes
    Quality    common.QualityType   // Affects size scaling
}
```

**Factory Functions:**

```go
// Create shapes with quality-based size scaling
func NewCircle(pixelX, pixelY int, quality common.QualityType) *BaseShape
func NewSquare(pixelX, pixelY int, quality common.QualityType) *BaseShape
func NewRectangle(pixelX, pixelY int, quality common.QualityType) *BaseShape
func NewLine(pixelX, pixelY int, direction ShapeDirection, quality common.QualityType) *BaseShape
func NewCone(pixelX, pixelY int, direction ShapeDirection, quality common.QualityType) *BaseShape
```

**Shape Quality Scaling:**

```go
// Example: Circle radius based on quality
switch quality {
case common.LowQuality:
    radius = rand.Intn(3)      // 0-2
case common.NormalQuality:
    radius = rand.Intn(4)      // 0-3
case common.HighQuality:
    radius = rand.Intn(9)      // 0-8
}
```

**Shape Interface:**

```go
type TileBasedShape interface {
    GetIndices() []int                      // Get affected tile indices
    UpdatePosition(pixelX, pixelY int)     // Move shape
    StartPositionPixels() (int, int)       // Get position
    GetDirection() ShapeDirection          // Get direction
    CanRotate() bool                       // Check if rotatable
}

// Usage example
shape := NewCone(pixelX, pixelY, LineUp, HighQuality)
affectedTiles := shape.GetIndices()  // Returns []int of map indices
shape.Rotate()                       // Rotate clockwise
```

**Direction System:**

```go
type ShapeDirection int

const (
    LineUp = iota
    LineDown
    LineRight
    LineLeft
    LineDiagonalUpRight
    LineDiagonalDownRight
    LineDiagonalUpLeft
    LinedDiagonalDownLeft
    NoDirection
)

func RotateRight(dir ShapeDirection) ShapeDirection
func RotateLeft(dir ShapeDirection) ShapeDirection
```

#### Color Matrix System

**Location:** `graphics/colormatrix.go`

**Purpose:** Apply color tinting to sprites (damage flashes, status effects)

```go
type ColorMatrix struct {
    R, G, B, A float64
}

// Red flash for damage
func GetDamageFlash() ebiten.ColorM {
    cm := ColorMatrix{R: 1.5, G: 0.5, B: 0.5, A: 1.0}
    return cm.ToEbitenColorM()
}

// Usage in rendering
op := &ebiten.DrawImageOptions{}
op.ColorM = GetDamageFlash()
screen.DrawImage(sprite, op)
```

---

### 5. Collision/Combat System

**Location:** `combat/attackingsystem.go`, `pathfinding/astar.go`

#### Combat Flow

**Melee Combat:**

```go
func MeleeAttackSystem(ecsmanager *common.EntityManager, pl *avatar.PlayerData,
                       gm *worldmap.GameMap, attackerPos, defenderPos *coords.LogicalPosition) {
    // 1. Determine attacker/defender and weapon
    var attacker, defender *ecs.Entity
    var weapon *gear.MeleeWeapon
    playerAttacking := false

    if pl.Pos.IsEqual(attackerPos) {
        playerAttacking = true
        attacker = pl.PlayerEntity
        defender = common.GetCreatureAtPosition(ecsmanager, defenderPos)
        weapon = pl.Equipment.MeleeWeapon()
    } else {
        attacker = common.GetCreatureAtPosition(ecsmanager, attackerPos)
        defender = pl.PlayerEntity
        weapon = common.GetComponentType[*gear.MeleeWeapon](attacker, gear.MeleeWeaponComponent)
    }

    // 2. Calculate damage and perform attack
    if weapon != nil {
        damage := weapon.CalculateDamage()
        attackSuccess := PerformAttack(ecsmanager, pl, gm, damage, attacker, defender, playerAttacking)
        UpdateAttackMessage(attacker, attackSuccess, playerAttacking, damage)
    }
}
```

**Attack Resolution:**

```go
func PerformAttack(em *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap,
                   damage int, attacker, defender *ecs.Entity, isPlayerAttacking bool) bool {
    attAttr := common.GetAttributes(attacker)
    defAttr := common.GetAttributes(defender)

    // 1. Attack roll: 1d20 + AttackBonus vs ArmorClass
    attackRoll := randgen.GetDiceRoll(20) + attAttr.AttackBonus

    if attackRoll >= defAttr.TotalArmorClass {
        // 2. Dodge roll: 1d100 vs DodgeChance
        dodgeRoll := randgen.GetRandomBetween(0, 100)

        if dodgeRoll >= int(defAttr.TotalDodgeChance) {
            // 3. Apply damage minus protection
            totalDamage := damage - defAttr.TotalProtection
            if totalDamage < 0 {
                totalDamage = 1  // Minimum damage
            }

            defAttr.CurrentHealth -= totalDamage
            return true
        }
    }

    // 4. Check for entity death
    if isPlayerAttacking {
        resmanager.RemoveEntity(em.World, gm, defender)
    }

    return false
}
```

**Ranged Combat:**

```go
func RangedAttackSystem(ecsmanager *common.EntityManager, pl *avatar.PlayerData,
                        gm *worldmap.GameMap, attackerPos *coords.LogicalPosition) {
    var attacker *ecs.Entity
    var weapon *gear.RangedWeapon
    var targets []*ecs.Entity

    if pl.Pos.IsEqual(attackerPos) {
        // Player attacking
        attacker = pl.PlayerEntity
        weapon = pl.Equipment.RangedWeapon()
        if weapon != nil {
            targets = weapon.GetTargets(ecsmanager)  // AOE targeting
        }
    } else {
        // Monster attacking (simple single-target)
        attacker = common.GetCreatureAtPosition(ecsmanager, attackerPos)
        weapon = common.GetComponentType[*gear.RangedWeapon](attacker, gear.RangedWeaponComponent)
        targets = append(targets, pl.PlayerEntity)
    }

    if weapon != nil {
        for _, target := range targets {
            defenderPos := common.GetPosition(target)

            // Range check
            if attackerPos.InRange(defenderPos, weapon.ShootingRange) {
                damage := weapon.CalculateDamage()
                attackSuccess := PerformAttack(ecsmanager, pl, gm, damage, attacker, target, playerAttacking)

                // Display visual effect
                weapon.DisplayShootingVX(attackerPos, defenderPos)
                UpdateAttackMessage(attacker, attackSuccess, playerAttacking, damage)
            }
        }
    }
}
```

#### Pathfinding

**Location:** `pathfinding/astar.go`

**A\* Implementation:**

```go
func AStarPath(start, goal coords.LogicalPosition, gameMap *worldmap.GameMap) []coords.LogicalPosition {
    openSet := []Node{{Pos: start, G: 0, H: heuristic(start, goal)}}
    closedSet := make(map[coords.LogicalPosition]bool)
    cameFrom := make(map[coords.LogicalPosition]coords.LogicalPosition)
    gScore := make(map[coords.LogicalPosition]int)
    gScore[start] = 0

    for len(openSet) > 0 {
        // Get node with lowest F score
        current := getLowestFScore(openSet)

        if current.Pos.IsEqual(&goal) {
            return reconstructPath(cameFrom, current.Pos)
        }

        // Process neighbors
        neighbors := getNeighbors(current.Pos, gameMap)
        for _, neighbor := range neighbors {
            if closedSet[neighbor] {
                continue
            }

            tentativeG := gScore[current.Pos] + 1
            if tentativeG < gScore[neighbor] {
                cameFrom[neighbor] = current.Pos
                gScore[neighbor] = tentativeG
                openSet = append(openSet, Node{
                    Pos: neighbor,
                    G: tentativeG,
                    H: heuristic(neighbor, goal),
                })
            }
        }

        closedSet[current.Pos] = true
    }

    return nil  // No path found
}

func heuristic(a, b coords.LogicalPosition) int {
    return a.ChebyshevDistance(&b)  // Chebyshev distance for 8-directional movement
}
```

---

### 6. UI System

**Location:** `gui/playerUI.go`, `gui/itemui.go`, `gui/statsui.go`, `gui/messagesUI.go`

**UI Framework:** [ebitenui](https://github.com/ebitenui/ebitenui) - widget-based UI library

#### UI Structure

```go
type PlayerUI struct {
    MainPlayerInterface *ebitenui.UI
    ItemMenuOpen        bool
    EquipmentMenuOpen   bool
    ConsumableMenuOpen  bool
    ThrowingMenuOpen    bool

    StatsUI      StatsUIData
    MsgUI        MessagesUIData
    ItemMenuUI   ItemMenuUIData
    EquipmentUI  EquipmentUIData
    ConsumableUI ConsumableUIData
    ThrowingUI   ThrowingUIData
}
```

**UI Components:**

1. **Stats UI** (right panel)
   - Health, AC, Protection, Dodge
   - Movement/Attack Speed
   - Updated every frame

2. **Message Log** (bottom panel)
   - Combat messages
   - Status effect notifications
   - Scrollable text area

3. **Inventory Menu** (I key)
   - List of all items
   - Equipped items marked
   - Click to equip/use

4. **Equipment Menu** (E key)
   - Weapon slots
   - Armor slots
   - Stats comparison

5. **Consumable Menu** (C key)
   - Potions, scrolls
   - Use/Throw options

6. **Throwing UI** (T key on item)
   - Target area visualization
   - Rotation controls
   - Execution confirmation

#### UI Creation Pattern

**Button Factory:**

```go
func CreateButton(text string, onClick func()) *widget.Button {
    btn := widget.NewButton(
        widget.ButtonOpts.WidgetOpts(
            widget.WidgetOpts.LayoutData(widget.RowLayoutData{
                Stretch: true,
            }),
        ),
        widget.ButtonOpts.Image(buttonImages),
        widget.ButtonOpts.Text(text, buttonFace, buttonTextColor),
        widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
            onClick()
        }),
    )
    return btn
}
```

**Container Layout:**

```go
func CreateContainer(width, height int, x, y int) *widget.Container {
    container := widget.NewContainer(
        widget.ContainerOpts.Layout(widget.NewRowLayout(
            widget.RowLayoutOpts.Direction(widget.DirectionVertical),
            widget.RowLayoutOpts.Spacing(10),
        )),
        widget.ContainerOpts.WidgetOpts(
            widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
                HorizontalPosition: widget.AnchorLayoutPositionStart,
                VerticalPosition:   widget.AnchorLayoutPositionStart,
            }),
        ),
    )
    return container
}
```

#### Message System

**Location:** `gui/messagesUI.go`, `gui/usermessages.go`

**Message Flow:**

```go
// 1. Messages written to entity UserMessage component
msg := common.GetComponentType[*common.UserMessage](entity, common.UserMsgComponent)
msg.AttackMessage = "You hit for 10 damage"

// 2. ProcessUserLog queries all messengers and displays
func ProcessUserLog(em common.EntityManager, screen *ebiten.Image, msgUI *MessagesUIData) {
    messages := []string{}

    for _, result := range em.World.Query(em.WorldTags["messengers"]) {
        msg := result.Components[common.UserMsgComponent].(*common.UserMessage)
        if msg.AttackMessage != "" {
            messages = append(messages, msg.AttackMessage)
            msg.AttackMessage = ""  // Clear after reading
        }
        // ... other message types
    }

    // Update text area
    msgUI.MessageTextArea.SetText(strings.Join(messages, "\n"))
}
```

---

### 7. Gameplay Mechanics

#### Item System

**Location:** `gear/items.go`, `gear/itemactions.go`, `gear/stateffect.go`

**Item Structure:**

```go
type Item struct {
    Properties *ecs.Entity      // Status effects entity
    Actions    []ItemAction     // Actions (throwable, consumable)
    Count      int              // Stack count
}
```

**Item Actions (Interface Pattern):**

```go
type ItemAction interface {
    ActionName() string
    Execute(targets []*ecs.Entity) error
    Copy() ItemAction
}

type ThrowableAction struct {
    TargetArea TileBasedShape  // AOE shape
    Effects    []StatusEffects // Status effects to apply
    Damage     int             // Direct damage
}

func (t *ThrowableAction) Execute(targets []*ecs.Entity) error {
    for _, target := range targets {
        // Apply damage
        attr := common.GetAttributes(target)
        attr.CurrentHealth -= t.Damage

        // Apply status effects
        for _, effect := range t.Effects {
            ApplyStatusEffect(target, effect)
        }
    }
    return nil
}
```

**Status Effects:**

```go
type StatusEffects interface {
    StatusEffectName() string
    StatusEffectComponent() *ecs.Component
    ApplyEffect(entity *ecs.Entity)
    RemoveEffect(entity *ecs.Entity)
    Duration() int
}

type Burning struct {
    DamagePerTurn int
    TurnsLeft     int
}

func (b *Burning) ApplyEffect(entity *ecs.Entity) {
    attr := common.GetAttributes(entity)
    attr.CurrentHealth -= b.DamagePerTurn
    b.TurnsLeft--
}
```

#### Equipment System

**Location:** `gear/equipmentcomponents.go`, `gear/Inventory.go`

**Equipment Structure:**

```go
type Equipment struct {
    WeaponSlot *ecs.Entity    // Melee or Ranged weapon
    ArmorSlot  *ecs.Entity    // Body armor
}

type Inventory struct {
    Items     []*ecs.Entity   // All items in inventory
    MaxSlots  int
    Equipment Equipment
}

// Type-safe equipment getters
func (e *Equipment) MeleeWeapon() *gear.MeleeWeapon {
    if e.WeaponSlot != nil && e.WeaponSlot.HasComponent(gear.MeleeWeaponComponent) {
        return common.GetComponentType[*gear.MeleeWeapon](e.WeaponSlot, gear.MeleeWeaponComponent)
    }
    return nil
}

func (e *Equipment) RangedWeapon() *gear.RangedWeapon {
    if e.WeaponSlot != nil && e.WeaponSlot.HasComponent(gear.RangedWeaponComponent) {
        return common.GetComponentType[*gear.RangedWeapon](e.WeaponSlot, gear.RangedWeaponComponent)
    }
    return nil
}
```

**Attribute Calculation:**

```go
// Update total attributes from base + equipment
func UpdateEntityAttributes(entity *ecs.Entity) {
    attr := common.GetAttributes(entity)
    inventory := common.GetComponentType[*gear.Inventory](entity, gear.InventoryComponent)

    // Start with base stats
    attr.TotalArmorClass = attr.BaseArmorClass
    attr.TotalProtection = attr.BaseProtection
    attr.TotalDodgeChance = attr.BaseDodgeChance

    // Add armor bonuses
    if armor := inventory.Equipment.ArmorSlot; armor != nil {
        armorComp := common.GetComponentType[*gear.Armor](armor, gear.ArmorComponent)
        attr.TotalArmorClass += armorComp.ArmorClass
        attr.TotalProtection += armorComp.Protection
        attr.TotalDodgeChance += armorComp.DodgeChance
    }

    // Add weapon bonuses
    // ... similar logic for weapons
}
```

**Weapon Components:**

```go
type MeleeWeapon struct {
    MinDamage   int
    MaxDamage   int
    AttackSpeed int
}

func (w *MeleeWeapon) CalculateDamage() int {
    return randgen.GetRandomBetween(w.MinDamage, w.MaxDamage)
}

type RangedWeapon struct {
    MinDamage      int
    MaxDamage      int
    ShootingRange  int
    AttackSpeed    int
    TargetArea     TileBasedShape  // AOE shape
    ShootingVXImage *ebiten.Image   // Visual effect sprite
}

func (w *RangedWeapon) GetTargets(ecsmanager *common.EntityManager) []*ecs.Entity {
    affectedIndices := w.TargetArea.GetIndices()
    targets := []*ecs.Entity{}

    for _, index := range affectedIndices {
        pos := coords.CoordManager.IndexToLogical(index)
        if creature := common.GetCreatureAtPosition(ecsmanager, &pos); creature != nil {
            targets = append(targets, creature)
        }
    }

    return targets
}
```

#### Consumable System

**Location:** `gear/consumables.go`, `gear/consumablequality.go`

**Consumable Structure:**

```go
type Consumable struct {
    Name         string
    AttrModifier common.Attributes  // Stat changes
    Duration     int                // Turns effect lasts (0 = instant)
}

func (c *Consumable) Apply(entity *ecs.Entity) {
    attr := common.GetAttributes(entity)

    if c.Duration == 0 {
        // Instant effect (healing potion)
        attr.CurrentHealth += c.AttrModifier.CurrentHealth
        if attr.CurrentHealth > attr.MaxHealth {
            attr.CurrentHealth = attr.MaxHealth
        }
    } else {
        // Temporary buff/debuff
        // Add to status effect tracker
        gear.AddTemporaryEffect(entity, c)
    }
}
```

**Quality System:**

```go
type ConsumableQuality int

const (
    Minor ConsumableQuality = iota
    Normal
    Greater
    Superior
)

func GetHealingPotionByQuality(quality ConsumableQuality) Consumable {
    switch quality {
    case Minor:
        return Consumable{Name: "Minor Healing Potion", AttrModifier: common.Attributes{CurrentHealth: 10}}
    case Normal:
        return Consumable{Name: "Healing Potion", AttrModifier: common.Attributes{CurrentHealth: 25}}
    case Greater:
        return Consumable{Name: "Greater Healing Potion", AttrModifier: common.Attributes{CurrentHealth: 50}}
    case Superior:
        return Consumable{Name: "Superior Healing Potion", AttrModifier: common.Attributes{CurrentHealth: 100}}
    }
}
```

---

### 8. Resource Management

**Location:** `resourcemanager/cleanup.go`

**Entity Removal System:**

```go
// Remove dead entities and clean up map references
func RemoveDeadEntities(em *common.EntityManager, gm *worldmap.GameMap) {
    for _, result := range em.World.Query(em.WorldTags["monsters"]) {
        attr := common.GetAttributes(result.Entity)

        if attr.CurrentHealth <= 0 {
            RemoveEntity(em.World, gm, result.Entity)
        }
    }
}

func RemoveEntity(manager *ecs.Manager, gm *worldmap.GameMap, entity *ecs.Entity) {
    // 1. Get position
    pos := common.GetPosition(entity)

    // 2. Unblock map tile
    index := coords.CoordManager.LogicalToIndex(*pos)
    gm.Tiles[index].Blocked = false

    // 3. Remove from entity tracker
    if tracker, ok := gm.EntityTracker[pos]; ok {
        delete(gm.EntityTracker, pos)
    }

    // 4. Destroy ECS entity
    manager.DisposeEntity(entity)
}
```

---

### 9. Coordinate System

**Location:** `coords/cordmanager.go`, `coords/position.go`

**Coordinate Types:**

```go
// Logical coordinates (grid position)
type LogicalPosition struct {
    X int
    Y int
}

// Pixel coordinates (screen rendering)
type PixelPosition struct {
    X int
    Y int
}
```

**CoordinateManager (Global Singleton):**

```go
var CoordManager *CoordinateManager

type CoordinateManager struct {
    dungeonWidth  int
    dungeonHeight int
    tileSize      int
    scaleFactor   int
    screenWidth   int
    screenHeight  int
}

// Core transformations
func (cm *CoordinateManager) LogicalToIndex(pos LogicalPosition) int {
    return (pos.Y * cm.dungeonWidth) + pos.X
}

func (cm *CoordinateManager) IndexToLogical(index int) LogicalPosition {
    x := index % cm.dungeonWidth
    y := index / cm.dungeonWidth
    return LogicalPosition{X: x, Y: y}
}

func (cm *CoordinateManager) LogicalToPixel(pos LogicalPosition) PixelPosition {
    return PixelPosition{
        X: pos.X * cm.tileSize,
        Y: pos.Y * cm.tileSize,
    }
}

func (cm *CoordinateManager) PixelToLogical(pos PixelPosition) LogicalPosition {
    return LogicalPosition{
        X: pos.X / cm.tileSize,
        Y: pos.Y / cm.tileSize,
    }
}
```

**Viewport System:**

```go
type Viewport struct {
    centerX, centerY int
    manager          *CoordinateManager
}

// Convert logical coordinates to screen coordinates with camera centering
func (v *Viewport) LogicalToScreen(pos LogicalPosition) (float64, float64) {
    offsetX := float64(v.manager.screenWidth)/2 -
               float64(v.centerX*v.manager.tileSize)*float64(v.manager.scaleFactor)
    offsetY := float64(v.manager.screenHeight)/2 -
               float64(v.centerY*v.manager.tileSize)*float64(v.manager.scaleFactor)

    scaledX := float64(pos.X*v.manager.tileSize) * float64(v.manager.scaleFactor)
    scaledY := float64(pos.Y*v.manager.tileSize) * float64(v.manager.scaleFactor)

    return scaledX + offsetX, scaledY + offsetY
}
```

**Position Utilities:**

```go
func (p *LogicalPosition) IsEqual(other *LogicalPosition) bool {
    return p.X == other.X && p.Y == other.Y
}

func (p *LogicalPosition) ChebyshevDistance(other *LogicalPosition) int {
    dx := abs(p.X - other.X)
    dy := abs(p.Y - other.Y)
    return max(dx, dy)
}

func (p *LogicalPosition) InRange(other *LogicalPosition, maxRange int) bool {
    return p.ChebyshevDistance(other) <= maxRange
}
```

---

### 10. Squad System (In Development)

**Location:** `squads/components.go`, `squads/squadmanager.go`, `squads/units.go`, `squads/squadcreation.go`

**Status:** 35% complete (621 LOC implemented)

**Architecture:** Pure ECS design demonstrating best practices for the codebase

#### Squad Components

**Squad Entity:**

```go
type SquadData struct {
    SquadID    ecs.EntityID      // Native entity ID (not pointer)
    Formation  FormationType     // Balanced, Defensive, Offensive, Ranged
    Name       string
    Morale     int               // 0-100
    SquadLevel int               // For spawning
    TurnCount  int
    MaxUnits   int               // Typically 9 (3x3 grid)
}

type FormationType int

const (
    FormationBalanced FormationType = iota
    FormationDefensive
    FormationOffensive
    FormationRanged
)
```

**Unit Entity Components:**

```go
// Links unit to parent squad (native ID, not pointer)
type SquadMemberData struct {
    SquadID ecs.EntityID
}

// Position within 3x3 grid (supports multi-cell units)
type GridPositionData struct {
    AnchorRow int  // Top-left row (0-2)
    AnchorCol int  // Top-left col (0-2)
    Width     int  // Columns occupied (1-3)
    Height    int  // Rows occupied (1-3)
}

func (g *GridPositionData) GetOccupiedCells() [][2]int {
    var cells [][2]int
    for r := g.AnchorRow; r < g.AnchorRow+g.Height && r < 3; r++ {
        for c := g.AnchorCol; c < g.AnchorCol+g.Width && c < 3; c++ {
            cells = append(cells, [2]int{r, c})
        }
    }
    return cells
}

// Combat role
type UnitRoleData struct {
    Role UnitRole  // Tank, DPS, Support
}

type UnitRole int

const (
    RoleTank UnitRole = iota
    RoleDPS
    RoleSupport
)
```

**Targeting Components:**

```go
type TargetMode int

const (
    TargetModeRowBased  TargetMode = iota  // Target entire row(s)
    TargetModeCellBased                    // Target specific cells
)

type TargetRowData struct {
    Mode TargetMode

    // Row-based targeting
    TargetRows    []int  // e.g., [0] for front row, [0,1,2] for all
    IsMultiTarget bool
    MaxTargets    int    // 0 = unlimited

    // Cell-based targeting (complex patterns)
    TargetCells [][2]int  // e.g., [[0,0], [0,1]] for 1x2 pattern
}
```

**Leader Abilities:**

```go
type LeaderData struct {
    Leadership int
    Experience int
}

type AbilitySlotData struct {
    Slots [4]AbilitySlot  // FFT-style 4 ability slots
}

type AbilitySlot struct {
    AbilityType  AbilityType  // Rally, Heal, BattleCry, Fireball
    TriggerType  TriggerType  // When to activate
    Threshold    float64      // Condition threshold
    HasTriggered bool         // Once-per-combat flag
    IsEquipped   bool
}

type AbilityType int

const (
    AbilityNone AbilityType = iota
    AbilityRally       // Damage boost to squad
    AbilityHeal        // HP restoration
    AbilityBattleCry   // Morale + damage boost
    AbilityFireball    // Direct AOE damage
)

type TriggerType int

const (
    TriggerNone TriggerType = iota
    TriggerSquadHPBelow    // Squad average HP < threshold
    TriggerTurnCount       // Specific turn number
    TriggerEnemyCount      // Number of enemy squads
    TriggerMoraleBelow     // Squad morale < threshold
    TriggerCombatStart     // First turn
)
```

#### Squad Query System

**Implemented Functions:**

```go
// Get all units at specific grid position
func GetUnitIDsAtGridPosition(squadID ecs.EntityID, row, col int,
                               manager *ecs.Manager) []ecs.EntityID {
    var unitIDs []ecs.EntityID

    for _, result := range manager.Query(SquadMemberTag) {
        memberData := result.Components[SquadMemberComponent].(*SquadMemberData)
        if memberData.SquadID != squadID {
            continue
        }

        gridPos := result.Components[GridPositionComponent].(*GridPositionData)
        if gridPos.OccupiesCell(row, col) {
            unitIDs = append(unitIDs, result.Entity.ID())
        }
    }

    return unitIDs
}

// Find unit entity by ID
func FindUnitByID(unitID ecs.EntityID, manager *ecs.Manager) *ecs.Entity {
    for _, result := range manager.Query(SquadMemberTag) {
        if result.Entity.ID() == unitID {
            return result.Entity
        }
    }
    return nil
}
```

**Planned Functions (Documented but not implemented):**

```go
// Get all units in squad
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *ecs.Manager) []ecs.EntityID

// Get units in specific row
func GetUnitIDsInRow(squadID ecs.EntityID, row int, manager *ecs.Manager) []ecs.EntityID

// Find squad leader
func GetLeaderID(squadID ecs.EntityID, manager *ecs.Manager) ecs.EntityID

// Check if squad is destroyed
func IsSquadDestroyed(squadID ecs.EntityID, manager *ecs.Manager) bool
```

#### Squad Combat (Planned)

**Combat Flow:**

```go
// NOT YET IMPLEMENTED
type CombatResult struct {
    TotalDamage    int
    UnitsHit       int
    CriticalHits   int
    AbilitiesTriggered []AbilityType
}

func ExecuteSquadAttack(attackerID, defenderID ecs.EntityID,
                        manager *ecs.Manager) *CombatResult {
    // 1. Get attacker's target data
    // 2. Determine which defender units are in target area
    // 3. Apply role-based damage modifiers
    // 4. Check for leader ability triggers
    // 5. Update morale based on casualties
    // 6. Return combat result
}
```

**Multi-Cell Unit Examples:**

- **1x1 Standard**: Goblins, archers (most units)
- **2x1 Cavalry**: Fast flanking units
- **1x2 Giants**: Front-line behemoths
- **2x2 Boss**: Large enemy commanders
- **3x3 Dragon**: Raid boss units

**Targeting Pattern Examples:**

- **1x1 Single Target**: `[[0,0]]` (front-left cell)
- **1x2 Cleave**: `[[0,0], [0,1]]` (horizontal slash)
- **2x2 Blast**: `[[0,0], [0,1], [1,0], [1,1]]` (area attack)
- **3x3 AOE**: All cells (full squad attack)
- **Row Target**: `TargetRows: [0]` (front row only)
- **All Rows**: `TargetRows: [0,1,2]` (piercing attack)

#### ECS Best Practices Demonstrated

The squad system exemplifies proper ECS architecture:

1. **Pure Data Components**: Zero logic methods (except pure helpers)
2. **Native EntityID Usage**: No custom entity pointers or registries
3. **Query-Based Relationships**: Discover relationships via ECS queries, not stored pointers
4. **System-Based Logic**: All combat/ability logic in systems, not components
5. **Value-Based Keys**: Future position system will use `map[LogicalPosition][]EntityID` (O(1) lookups)

---

## Package Documentation

### avatar/

**Purpose:** Player-specific data and state management

**Key Files:**
- `playerdata.go` - Player entity, position, input states, equipment

**Public API:**
```go
type PlayerData struct {
    PlayerEntity  *ecs.Entity
    Pos           *coords.LogicalPosition
    Equipment     gear.Equipment
    Inventory     gear.Inventory
    InputStates   InputStates
}

type InputStates struct {
    HasKeyInput    bool
    ThrowingItem   bool
    ShootingRanged bool
    MenuOpen       bool
}

func (p *PlayerData) PlayerAttributes() *common.Attributes
```

**Dependencies:** common, coords, gear

---

### combat/

**Purpose:** Combat system implementation (attack resolution, damage calculation)

**Key Files:**
- `attackingsystem.go` - Melee/ranged attack functions

**Public API:**
```go
func MeleeAttackSystem(ecsmanager *common.EntityManager, pl *avatar.PlayerData,
                       gm *worldmap.GameMap, attackerPos, defenderPos *coords.LogicalPosition)

func RangedAttackSystem(ecsmanager *common.EntityManager, pl *avatar.PlayerData,
                        gm *worldmap.GameMap, attackerPos *coords.LogicalPosition)

func PerformAttack(em *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap,
                   damage int, attacker, defender *ecs.Entity, isPlayerAttacking bool) bool
```

**Dependencies:** common, coords, gear, graphics, randgen, resourcemanager, worldmap

**Combat Formula:**
1. Attack Roll: 1d20 + AttackBonus >= ArmorClass
2. Dodge Roll: 1d100 >= DodgeChance
3. Damage: WeaponDamage - Protection (min 1)

---

### common/

**Purpose:** Core ECS utilities, shared components, helper functions

**Key Files:**
- `ecsutil.go` - EntityManager, component accessors
- `commoncomponents.go` - Position, Attributes, Name, UserMessage
- `commontypes.go` - QualityType enum

**Public API:**
```go
type EntityManager struct {
    World     *ecs.Manager
    WorldTags map[string]ecs.Tag
}

func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T
func GetAttributes(e *ecs.Entity) *Attributes
func GetPosition(e *ecs.Entity) *coords.LogicalPosition
func GetCreatureAtPosition(ecsmanager *EntityManager, pos *coords.LogicalPosition) *ecs.Entity
func DistanceBetween(e1 *ecs.Entity, e2 *ecs.Entity) int
```

**Components:**
- `PositionComponent` - Logical grid position
- `AttributeComponent` - Health, AC, stats
- `NameComponent` - Display name
- `UserMsgComponent` - UI messages

**Dependencies:** coords, bytearena/ecs

---

### coords/

**Purpose:** Coordinate system management and transformations

**Key Files:**
- `cordmanager.go` - CoordinateManager, Viewport
- `position.go` - LogicalPosition, PixelPosition

**Public API:**
```go
var CoordManager *CoordinateManager

type CoordinateManager struct { ... }

func (cm *CoordinateManager) LogicalToIndex(pos LogicalPosition) int
func (cm *CoordinateManager) IndexToLogical(index int) LogicalPosition
func (cm *CoordinateManager) LogicalToPixel(pos LogicalPosition) PixelPosition
func (cm *CoordinateManager) PixelToLogical(pos PixelPosition) LogicalPosition
func (cm *CoordinateManager) IsValidLogical(pos LogicalPosition) bool

type Viewport struct { ... }
func (v *Viewport) LogicalToScreen(pos LogicalPosition) (float64, float64)
func (v *Viewport) ScreenToLogical(screenX, screenY int) LogicalPosition
```

**Dependencies:** None (pure coordinate math)

---

### entitytemplates/

**Purpose:** JSON-based entity template loading and factory system

**Key Files:**
- `creators.go` - Generic entity factory, component adders
- `readdata.go` - JSON loading functions
- `jsonstructs.go` - JSON data structures
- `templatelib.go` - Template library management

**Public API:**
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
    GameMap   *worldmap.GameMap
}

func CreateEntityFromTemplate(manager common.EntityManager, config EntityConfig, data any) *ecs.Entity

// Legacy functions (deprecated but maintained for compatibility)
func CreateMeleeWeaponFromTemplate(manager common.EntityManager, w JSONMeleeWeapon, ...) *ecs.Entity
func CreateRangedWeaponFromTemplate(manager common.EntityManager, w JSONRangedWeapon, ...) *ecs.Entity
func CreateConsumableFromTemplate(manager common.EntityManager, c JSONAttributeModifier, ...) *ecs.Entity
func CreateCreatureFromTemplate(manager common.EntityManager, m JSONMonster, ...) *ecs.Entity

// Data loading
func ReadGameData()
func GetMonsterTemplate(name string) JSONMonster
func GetWeaponTemplate(name string) JSONMeleeWeapon
func GetConsumableTemplate(name string) JSONAttributeModifier
```

**JSON Structures:**
```go
type JSONMonster struct {
    Name         string
    Imgname      string
    Attributes   JSONAttributes
    Armor        *JSONArmor
    MeleeWeapon  *JSONMeleeWeapon
    RangedWeapon *JSONRangedWeapon
    Width        int      // Multi-cell unit support (squad system)
    Height       int
    Role         string   // "Tank", "DPS", "Support"
    TargetMode   string   // "row" or "cell"
    TargetRows   []int
    TargetCells  [][2]int
}

type JSONMeleeWeapon struct {
    MinDamage   int
    MaxDamage   int
    AttackSpeed int
}

type JSONRangedWeapon struct {
    ShootingVX   string
    MinDamage    int
    MaxDamage    int
    AttackSpeed  int
    ShootingRange int
    TargetArea   JSONTargetArea
}

type JSONAttributeModifier struct {  // Consumables
    Name       string
    Duration   int
    Attributes JSONAttributes
}
```

**Dependencies:** common, coords, gear, monsters, rendering, worldmap

---

### gear/

**Purpose:** Equipment, items, weapons, armor, consumables, status effects

**Key Files:**
- `items.go` - Item struct, item management
- `equipmentcomponents.go` - Weapon/Armor components
- `itemactions.go` - ThrowableAction, ConsumableAction
- `stateffect.go` - Status effect interface
- `consumables.go` - Consumable logic
- `consumablequality.go` - Quality tiers
- `itemquality.go` - Item quality enum
- `Inventory.go` - Inventory management
- `gearutil.go` - Utility functions

**Public API:**
```go
type Item struct {
    Properties *ecs.Entity      // Status effects
    Actions    []ItemAction     // Throwable, consumable, etc.
    Count      int
}

type ItemAction interface {
    ActionName() string
    Execute(targets []*ecs.Entity) error
    Copy() ItemAction
}

type ThrowableAction struct {
    TargetArea TileBasedShape
    Effects    []StatusEffects
    Damage     int
}

type MeleeWeapon struct {
    MinDamage   int
    MaxDamage   int
    AttackSpeed int
}

type RangedWeapon struct {
    MinDamage      int
    MaxDamage      int
    ShootingRange  int
    AttackSpeed    int
    TargetArea     TileBasedShape
    ShootingVXImage *ebiten.Image
}

type Armor struct {
    ArmorClass  int
    Protection  int
    DodgeChance float32
}

type Equipment struct {
    WeaponSlot *ecs.Entity
    ArmorSlot  *ecs.Entity
}

type Inventory struct {
    Items     []*ecs.Entity
    MaxSlots  int
    Equipment Equipment
}

func InitializeItemComponents(manager *ecs.Manager, tags map[string]ecs.Tag)
func UpdateEntityAttributes(entity *ecs.Entity)
func CreateItem(manager *ecs.Manager, name, imagePath string, pos coords.LogicalPosition,
                effects ...StatusEffects) *ecs.Entity
```

**Components:**
- `ItemComponent` - Item data
- `MeleeWeaponComponent` - Melee weapon stats
- `RangedWeaponComponent` - Ranged weapon stats
- `ArmorComponent` - Armor stats
- `ConsumableComponent` - Consumable effects
- `InventoryComponent` - Inventory data

**Dependencies:** common, coords, graphics, rendering

---

### graphics/

**Purpose:** Visual rendering, shapes, visual effects, coordinate transformations

**Key Files:**
- `drawableshapes.go` - Unified shape system (circles, lines, cones, etc.)
- `vx.go` - Visual effects handler
- `colormatrix.go` - Color tinting
- `graphictypes.go` - Graphics constants

**Public API:**
```go
// Global singletons
var VXHandler VisualEffectHandler
var ScreenInfo ScreenData

// Shape system
type BasicShapeType int
const (
    Circular BasicShapeType = iota
    Rectangular
    Linear
)

type BaseShape struct {
    Position  coords.PixelPosition
    Type      BasicShapeType
    Size      int
    Width     int
    Height    int
    Direction *ShapeDirection
    Quality   common.QualityType
}

func NewCircle(pixelX, pixelY int, quality common.QualityType) *BaseShape
func NewSquare(pixelX, pixelY int, quality common.QualityType) *BaseShape
func NewRectangle(pixelX, pixelY int, quality common.QualityType) *BaseShape
func NewLine(pixelX, pixelY int, direction ShapeDirection, quality common.QualityType) *BaseShape
func NewCone(pixelX, pixelY int, direction ShapeDirection, quality common.QualityType) *BaseShape

type TileBasedShape interface {
    GetIndices() []int
    UpdatePosition(pixelX, pixelY int)
    StartPositionPixels() (int, int)
    GetDirection() ShapeDirection
    CanRotate() bool
}

// Direction system
type ShapeDirection int
const (
    LineUp = iota
    LineDown
    LineRight
    LineLeft
    LineDiagonalUpRight
    LineDiagonalDownRight
    LineDiagonalUpLeft
    LinedDiagonalDownLeft
    NoDirection
)

func RotateRight(dir ShapeDirection) ShapeDirection
func RotateLeft(dir ShapeDirection) ShapeDirection

// Visual effects
type VisualEffectHandler struct {
    ActiveVXs []VisualEffect
}

type VisualEffect struct {
    Image    *ebiten.Image
    PixelPos coords.PixelPosition
    Duration int
    Rotation float64
}

func (vxh *VisualEffectHandler) AddVisualEffect(vx VisualEffect)
func (vxh *VisualEffectHandler) UpdateVisualEffects()
func (vxh *VisualEffectHandler) DrawVisualEffects(screen *ebiten.Image)

// Constants
const (
    MAP_SCROLLING_ENABLED = true
    ViewableSquareSize    = 50
    StatsUIOffset         = 500
)

type ScreenData struct {
    ScreenWidth   int
    ScreenHeight  int
    TileSize      int
    DungeonWidth  int
    DungeonHeight int
    ScaleFactor   int
    LevelWidth    int
    LevelHeight   int
    PaddingRight  int
}
```

**Dependencies:** common, coords

---

### gui/

**Purpose:** UI system (menus, stats, messages)

**Key Files:**
- `playerUI.go` - Main UI structure, button factory
- `itemui.go` - Inventory menu
- `equipmentUI.go` - Equipment menu
- `consumableUI.go` - Consumable menu
- `throwingUI.go` - Throwing target UI
- `statsui.go` - Stats panel
- `messagesUI.go` - Message log
- `usermessages.go` - Message processing
- `guiresources.go` - UI assets loading
- `createwidgets.go` - Widget factory functions
- `infoUI.go` - Info panel
- `itemdisplaytype.go` - Item display enums

**Public API:**
```go
type PlayerUI struct {
    MainPlayerInterface *ebitenui.UI
    ItemMenuOpen        bool
    EquipmentMenuOpen   bool
    ConsumableMenuOpen  bool
    ThrowingMenuOpen    bool

    StatsUI      StatsUIData
    MsgUI        MessagesUIData
    ItemMenuUI   ItemMenuUIData
    EquipmentUI  EquipmentUIData
    ConsumableUI ConsumableUIData
    ThrowingUI   ThrowingUIData
}

func (p *PlayerUI) CreateMainInterface(playerData *avatar.PlayerData, em *common.EntityManager)
func CreateButton(text string, onClick func()) *widget.Button
func CreateContainer(width, height, x, y int) *widget.Container
func ProcessUserLog(em common.EntityManager, screen *ebiten.Image, msgUI *MessagesUIData)
func SetContainerLocation(container *widget.Container, x, y int)
```

**Dependencies:** common, avatar, gear, ebitenui

---

### input/

**Purpose:** Input handling coordination and controllers

**Key Files:**
- `inputcoordinator.go` - Input coordinator, shared state
- `movementcontroller.go` - Movement and melee
- `combatcontroller.go` - Throwing and ranged
- `uicontroller.go` - Menu interactions
- `debuginput.go` - Debug commands

**Public API:**
```go
type InputCoordinator struct {
    movementController *MovementController
    combatController   *CombatController
    uiController       *UIController
    sharedState        *SharedInputState
}

func NewInputCoordinator(ecsManager *common.EntityManager, playerData *avatar.PlayerData,
                         gameMap *worldmap.GameMap, playerUI *gui.PlayerUI) *InputCoordinator

func (ic *InputCoordinator) HandleInput() bool

type InputController interface {
    HandleInput() bool
    CanHandle() bool
    OnActivate()
    OnDeactivate()
}

func PlayerDebugActions(playerData *avatar.PlayerData)
```

**Dependencies:** avatar, combat, common, coords, gear, gui, worldmap

---

### monsters/

**Purpose:** Creature components and behavior

**Key Files:**
- `creatures.go` - Creature component

**Public API:**
```go
type Creature struct {
    Path []coords.LogicalPosition  // Pathfinding cache
}

var CreatureComponent *ecs.Component
```

**Dependencies:** coords, ecs

---

### pathfinding/

**Purpose:** A\* pathfinding algorithm

**Key Files:**
- `astar.go` - A\* implementation

**Public API:**
```go
func AStarPath(start, goal coords.LogicalPosition, gameMap *worldmap.GameMap) []coords.LogicalPosition
```

**Algorithm:**
- Heuristic: Chebyshev distance (8-directional movement)
- Cost: Uniform (1 per tile)
- Obstacle handling: Blocked tiles excluded

**Dependencies:** coords, worldmap

---

### randgen/

**Purpose:** Random number generation utilities

**Key Files:**
- `randnumgen.go` - RNG functions

**Public API:**
```go
func GetRandomBetween(min, max int) int
func GetDiceRoll(sides int) int        // 1dN
func GetMultipleDiceRoll(count, sides int) int  // NdM
```

**Dependencies:** None

---

### rendering/

**Purpose:** Entity rendering system

**Key Files:**
- `rendering.go` - Renderable component, rendering functions

**Public API:**
```go
type Renderable struct {
    Image   *ebiten.Image
    Visible bool
}

var RenderableComponent *ecs.Component

func ProcessRenderables(ecsmanager *common.EntityManager, gameMap worldmap.GameMap,
                        screen *ebiten.Image, debugMode bool)

func ProcessRenderablesInSquare(ecsmanager *common.EntityManager, gameMap worldmap.GameMap,
                                 screen *ebiten.Image, playerPos *coords.LogicalPosition,
                                 squareSize int, debugMode bool)
```

**Dependencies:** common, coords, graphics, worldmap, ecs, ebiten

---

### resourcemanager/

**Purpose:** Entity lifecycle management (cleanup, removal)

**Key Files:**
- `cleanup.go` - Entity removal system

**Public API:**
```go
func RemoveDeadEntities(em *common.EntityManager, gm *worldmap.GameMap)
func RemoveEntity(manager *ecs.Manager, gm *worldmap.GameMap, entity *ecs.Entity)
```

**Dependencies:** common, coords, worldmap, ecs

---

### spawning/

**Purpose:** Entity spawning, loot tables, probability tables

**Key Files:**
- `spawnmonsters.go` - Monster spawning
- `spawnloot.go` - Equipment spawning
- `spawnthrowable.go` - Throwable item spawning
- `loottables.go` - Loot probability tables
- `probtables.go` - Generic probability table utilities

**Public API:**
```go
func SpawnStartingCreatures(level int, em *common.EntityManager, gm *worldmap.GameMap,
                             pd *avatar.PlayerData)

func SpawnStartingEquipment(em *common.EntityManager, gm *worldmap.GameMap,
                             pd *avatar.PlayerData)

func SpawnThrowableItem(manager *ecs.Manager, x, y int) *ecs.Entity

type LootTable struct {
    Entries []LootEntry
}

type LootEntry struct {
    TemplateName string
    Weight       int
}

func InitLootSpawnTables()
func SelectFromLootTable(table LootTable) string
```

**Dependencies:** common, coords, entitytemplates, gear, randgen, worldmap

---

### squads/

**Purpose:** Squad-based tactical combat system (in development)

**Key Files:**
- `components.go` - Squad/Unit components, abilities
- `squadmanager.go` - ECS registration
- `units.go` - Unit creation, multi-cell support
- `squadcreation.go` - Squad factory functions
- `squadqueries.go` - Query functions (partial)
- `squadcombat.go` - Combat system (not implemented)
- `squads_test.go` - Unit tests

**Public API (Implemented):**
```go
func InitializeSquadData() error
func CreateEmptySquad(manager *ecs.Manager, squadName string) *ecs.Entity
func CreateUnitEntity(manager *ecs.Manager, template entitytemplates.JSONMonster) *ecs.Entity
func GetUnitIDsAtGridPosition(squadID ecs.EntityID, row, col int, manager *ecs.Manager) []ecs.EntityID
func FindUnitByID(unitID ecs.EntityID, manager *ecs.Manager) *ecs.Entity

// Components
type SquadData struct { ... }
type SquadMemberData struct { ... }
type GridPositionData struct { ... }
type UnitRoleData struct { ... }
type TargetRowData struct { ... }
type LeaderData struct { ... }
type AbilitySlotData struct { ... }
```

**Public API (Planned):**
```go
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *ecs.Manager) []ecs.EntityID
func GetUnitIDsInRow(squadID ecs.EntityID, row int, manager *ecs.Manager) []ecs.EntityID
func GetLeaderID(squadID ecs.EntityID, manager *ecs.Manager) ecs.EntityID
func IsSquadDestroyed(squadID ecs.EntityID, manager *ecs.Manager) bool
func ExecuteSquadAttack(attackerID, defenderID ecs.EntityID, manager *ecs.Manager) *CombatResult
func CheckAndTriggerAbilities(squadID ecs.EntityID, manager *ecs.Manager)
func CreateSquadFromTemplate(manager *ecs.Manager, formation FormationType, units []JSONMonster)
```

**Dependencies:** common, coords, entitytemplates, ecs

**Status:** 35% complete (621 LOC), demonstrates perfect ECS patterns

---

### testing/

**Purpose:** Test utilities and test data creation

**Key Files:**
- `testingdata.go` - Test entity creation

**Public API:**
```go
func CreateTestItems(manager *ecs.Manager, tags map[string]ecs.Tag, gm *worldmap.GameMap)
func UpdateContentsForTest(em *common.EntityManager, gm *worldmap.GameMap)
func InitTestActionManager(em *common.EntityManager, pd *avatar.PlayerData)
```

**Dependencies:** common, coords, entitytemplates, gear, worldmap

---

### timesystem/

**Purpose:** Turn-based timing and initiative (minimal implementation)

**Key Files:**
- `initiative.go` - Initiative system

**Public API:**
```go
// Minimal/stub implementation
```

**Dependencies:** None

---

### trackers/

**Purpose:** Entity tracking and status effect management

**Key Files:**
- `creaturetracker.go` - Position-based entity tracking
- `statustracker.go` - Status effect tracking

**Public API:**
```go
type PositionTracker struct {
    PosTracker map[*coords.LogicalPosition]*ecs.Entity
}

func (pt *PositionTracker) AddCreature(pos *coords.LogicalPosition, entity *ecs.Entity)
func (pt *PositionTracker) RemoveCreature(pos *coords.LogicalPosition)
func (pt *PositionTracker) GetCreatureAt(pos *coords.LogicalPosition) *ecs.Entity

// Status effect tracking
func AddStatusEffect(entity *ecs.Entity, effect StatusEffect)
func RemoveStatusEffect(entity *ecs.Entity, effectName string)
func UpdateStatusEffects(entity *ecs.Entity)
```

**Note:** Position tracking currently uses O(n) map lookups with pointer keys. Planned refactoring to O(1) hash-based spatial grid using value keys.

**Dependencies:** coords, ecs

---

### worldmap/

**Purpose:** Dungeon generation, map data structures, FOV, tile rendering

**Key Files:**
- `dungeongen.go` - BSP dungeon generation
- `dungeontile.go` - Tile data structure
- `GameMapUtil.go` - Map utilities, tile loading

**Public API:**
```go
type GameMap struct {
    Tiles          []*Tile
    Width          int
    Height         int
    Rooms          []Rect
    PlayerVisible  *FOV
    EntityTracker  map[*coords.LogicalPosition]*ecs.Entity
    RightEdgeX     int
}

type Tile struct {
    PixelX     int
    PixelY     int
    TileType   int
    Blocked    bool
    Image      *ebiten.Image
    BlockSight bool
}

type Rect struct {
    X1, Y1, X2, Y2 int
}

func NewGameMap() GameMap
func (gm *GameMap) DrawLevel(screen *ebiten.Image, debugMode bool)
func (gm *GameMap) DrawLevelCenteredSquare(screen *ebiten.Image, playerPos *coords.LogicalPosition,
                                            squareSize int, debugMode bool)
func (gm *GameMap) IsBlocked(x, y int) bool
func (gm *GameMap) AddEntityToTile(entity *ecs.Entity, pos *coords.LogicalPosition)
func (gm *GameMap) RemoveEntityFromTile(pos *coords.LogicalPosition)
func (gm *GameMap) UnblockedLogicalCoords(centerX, centerY, count int) []coords.LogicalPosition

type FOV struct {
    VisibleTiles   map[int]bool
    ExploredTiles  map[int]bool
}

func (fov *FOV) IsVisible(x, y int) bool
func (fov *FOV) IsExplored(x, y int) bool
func (fov *FOV) UpdateFOV(playerPos *coords.LogicalPosition, viewRange int, gm *GameMap)
```

**Dungeon Generation:**
- Algorithm: Binary Space Partitioning (BSP)
- Creates rooms connected by corridors
- Procedurally places walls and floors

**FOV Algorithm:**
- Ray-casting from player position
- Blocks line of sight on wall tiles
- Explored tiles remain visible (grayed out)

**Dependencies:** coords, randgen, ebiten

---

## Game-Specific Patterns

### Pattern 1: ECS Query-Based Systems

**Problem:** Need to process entities with specific component combinations

**Solution:** Use ECS tags to query entities efficiently

```go
// Define tag during initialization
renderables := ecs.BuildTag(rendering.RenderableComponent, common.PositionComponent)
tags["renderables"] = renderables

// Query entities with tag
for _, result := range ecsManager.World.Query(ecsManager.WorldTags["renderables"]) {
    pos := result.Components[common.PositionComponent].(*coords.LogicalPosition)
    renderable := result.Components[RenderableComponent].(*Renderable)
    // Process entity...
}
```

**Benefits:**
- O(1) component lookup via component index
- Type-safe component access
- Efficient filtering

**Used in:** rendering, combat, UI, cleanup systems

---

### Pattern 2: Component Composition with Factory Functions

**Problem:** Creating entities with varying component sets

**Solution:** Use ComponentAdder functions and template pattern

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

func createFromTemplate(manager common.EntityManager, name, imagePath, assetDir string,
                        visible bool, pos *coords.LogicalPosition, adders ...ComponentAdder) *ecs.Entity {
    entity := createBaseEntity(manager, name, imagePath, assetDir, visible, pos)

    for _, adder := range adders {
        adder(entity)
    }

    return entity
}
```

**Benefits:**
- Flexible entity composition
- Reusable component sets
- Type-safe entity construction

**Used in:** entitytemplates package

---

### Pattern 3: Unified Coordinate Management

**Problem:** Multiple coordinate systems (logical, pixel, index) causing bugs

**Solution:** Global CoordinateManager singleton with type-safe position types

```go
// Type-safe position types
type LogicalPosition struct { X, Y int }
type PixelPosition struct { X, Y int }

// Global manager
var CoordManager *CoordinateManager

// Explicit conversions
logical := LogicalPosition{X: 10, Y: 5}
pixel := CoordManager.LogicalToPixel(logical)
index := CoordManager.LogicalToIndex(logical)
```

**Benefits:**
- Compile-time type safety (can't mix position types)
- Single source of truth for conversions
- Eliminates coordinate bugs

**Used in:** All packages requiring coordinate transforms

---

### Pattern 4: Prioritized Input Controllers

**Problem:** Conflicting input handlers (UI vs movement vs combat)

**Solution:** Chain of Responsibility pattern with priority ordering

```go
type InputController interface {
    HandleInput() bool
    CanHandle() bool
    OnActivate()
    OnDeactivate()
}

func (ic *InputCoordinator) HandleInput() bool {
    // Priority 1: UI (menus block other input)
    if ic.uiController.CanHandle() {
        return ic.uiController.HandleInput()
    }

    // Priority 2: Combat (throwing/shooting)
    if ic.combatController.CanHandle() {
        return ic.combatController.HandleInput()
    }

    // Priority 3: Movement (default)
    if ic.movementController.CanHandle() {
        return ic.movementController.HandleInput()
    }

    return false
}
```

**Benefits:**
- Clear input priority
- Prevents conflicting actions
- Easy to add new controllers

**Used in:** input package

---

### Pattern 5: Quality-Based Procedural Generation

**Problem:** Need varied item/shape sizes based on quality tiers

**Solution:** Quality enum with factory functions using switch statements

```go
type QualityType int

const (
    LowQuality QualityType = iota
    NormalQuality
    HighQuality
)

func NewCircle(pixelX, pixelY int, quality QualityType) *BaseShape {
    var radius int
    switch quality {
    case LowQuality:
        radius = rand.Intn(3)      // 0-2
    case NormalQuality:
        radius = rand.Intn(4)      // 0-3
    case HighQuality:
        radius = rand.Intn(9)      // 0-8
    }

    return &BaseShape{
        Position: coords.PixelPosition{X: pixelX, Y: pixelY},
        Type:     Circular,
        Size:     radius,
        Quality:  quality,
    }
}
```

**Benefits:**
- Consistent quality scaling
- Easy to balance
- Clear progression

**Used in:** graphics (shapes), spawning (loot)

---

### Pattern 6: JSON-Driven Entity Templates

**Problem:** Hardcoding entity stats makes balancing difficult

**Solution:** JSON template files with generic factory

```go
// JSON file: assets/gamedata/monsterdata.json
{
  "monsters": [
    {
      "name": "GoblinWarrior",
      "imgname": "goblin.png",
      "attributes": {
        "maxHealth": 15,
        "attackBonus": 1,
        "baseArmorClass": 10
      },
      "meleeWeapon": {
        "minDamage": 2,
        "maxDamage": 3,
        "attackSpeed": 5
      }
    }
  ]
}

// Generic factory
config := EntityConfig{
    Type:      EntityCreature,
    Name:      template.Name,
    ImagePath: template.Imgname,
    Position:  &spawnPos,
    GameMap:   gameMap,
}
creature := CreateEntityFromTemplate(manager, config, template)
```

**Benefits:**
- Easy balancing without recompilation
- Designers can edit stats
- Versioning and backups

**Used in:** entitytemplates package

---

### Pattern 7: Visual Effect Handler (Singleton)

**Problem:** Need to render temporary effects (projectiles, explosions) outside ECS

**Solution:** Global visual effect handler with frame-based duration

```go
var VXHandler VisualEffectHandler

type VisualEffect struct {
    Image    *ebiten.Image
    PixelPos coords.PixelPosition
    Duration int        // Frames remaining
    Rotation float64
}

// Add effect
graphics.VXHandler.AddVisualEffect(VisualEffect{
    Image:    projectileSprite,
    PixelPos: startPos,
    Duration: 15,  // ~0.25 seconds at 60 FPS
})

// Update every frame
graphics.VXHandler.UpdateVisualEffects()  // Decrement durations, remove expired
graphics.VXHandler.DrawVisualEffects(screen)  // Render active effects
```

**Benefits:**
- Non-entity visual feedback
- Automatic cleanup
- Simple API

**Used in:** combat (projectiles), throwables (AOE indicators)

---

### Pattern 8: Type-Safe Component Access

**Problem:** Component access requires type assertions and error handling

**Solution:** Generic helper functions with panic recovery

```go
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {
    defer func() {
        if r := recover(); r != nil {
            // Error handling
        }
    }()

    if c, ok := entity.GetComponentData(component); ok {
        return c.(T)
    }

    var nilValue T
    return nilValue
}

// Usage
attr := GetComponentType[*common.Attributes](entity, common.AttributeComponent)
weapon := GetComponentType[*gear.MeleeWeapon](entity, gear.MeleeWeaponComponent)
```

**Benefits:**
- Type-safe access
- Graceful failure handling
- Clean calling code

**Used in:** common package, all systems

---

### Pattern 9: Drawable Shapes with Unified Interface

**Problem:** Multiple shape types (circle, line, cone) with duplicate code

**Solution:** Unified BaseShape with type variants and factory pattern

```go
type BasicShapeType int

const (
    Circular BasicShapeType = iota
    Rectangular
    Linear
)

type BaseShape struct {
    Position  coords.PixelPosition
    Type      BasicShapeType
    Size      int
    Direction *ShapeDirection
}

func (s *BaseShape) GetIndices() []int {
    switch s.Type {
    case Circular:
        return s.calculateCircle()
    case Rectangular:
        return s.calculateRectangle()
    case Linear:
        return s.calculateLine()
    }
}

// Factory functions
func NewCircle(x, y int, quality QualityType) *BaseShape
func NewLine(x, y int, dir ShapeDirection, quality QualityType) *BaseShape
```

**Benefits:**
- Code reuse (1 type instead of 8+)
- Consistent interface
- Easy to extend

**Used in:** graphics package, throwables, ranged weapons

---

### Pattern 10: Entity Relationship via EntityID (Squad System Pattern)

**Problem:** Storing entity pointers creates tight coupling and GC issues

**Solution:** Use native `ecs.EntityID` for relationships, discover via queries

```go
// ❌ BAD: Stored entity pointer
type Squad struct {
    Leader *ecs.Entity  // Creates coupling, prevents serialization
}

// ✅ GOOD: EntityID reference, discovered via query
type SquadMemberData struct {
    SquadID ecs.EntityID  // Native type, just an integer
}

func GetLeaderID(squadID ecs.EntityID, manager *ecs.Manager) ecs.EntityID {
    for _, result := range manager.Query(LeaderTag) {
        memberData := result.Components[SquadMemberComponent].(*SquadMemberData)
        if memberData.SquadID == squadID {
            return result.Entity.ID()
        }
    }
    return 0
}
```

**Benefits:**
- Serializable relationships
- No dangling pointers
- Decoupled architecture

**Used in:** squads package (exemplar pattern for future refactoring)

---

## Asset Pipeline

### Asset Directory Structure

```
assets/
├── gamedata/               # JSON entity templates
│   ├── monsterdata.json    # Monster definitions
│   ├── weapondata.json     # Weapon stats
│   ├── consumabledata.json # Consumable items
│   ├── monsterprefixes.json # Name generation
│   ├── creaturemodifiers.json # Variant modifiers
│   └── backup/             # Template backups
├── tiles/                  # Map tile images
│   ├── floors/
│   │   └── limestone/      # Floor tile variants
│   └── walls/
│       └── marble/         # Wall tile variants
├── monsters/               # Monster sprites
│   ├── goblin.png
│   ├── orc.png
│   ├── skeleton_naga.png
│   └── ... (other monsters)
├── items/                  # Item sprites
│   └── ... (weapons, potions, etc.)
├── vx/                     # Visual effects
│   ├── arrow.png
│   ├── bolt.png
│   ├── dark_bolt.png
│   └── ... (projectiles)
└── fonts/                  # UI fonts
    └── mplus1pregular.go   # Embedded font
```

### Asset Loading

**Tile Loading (worldmap/GameMapUtil.go):**

```go
func loadTileImages() {
    // Floor tiles
    dir := "../assets/tiles/floors/limestone"
    files, _ := os.ReadDir(dir)

    for _, file := range files {
        if !file.IsDir() {
            floor, _, _ := ebitenutil.NewImageFromFile(dir + "/" + file.Name())
            floorImgs = append(floorImgs, floor)
        }
    }

    // Wall tiles
    dir = "../assets/tiles/walls/marble"
    files, _ = os.ReadDir(dir)

    for _, file := range files {
        if !file.IsDir() {
            wall, _, _ := ebitenutil.NewImageFromFile(dir + "/" + file.Name())
            wallImgs = append(wallImgs, wall)
        }
    }
}

// Random tile selection
tile.Image = floorImgs[rand.Intn(len(floorImgs))]
```

**Entity Sprite Loading (entitytemplates/creators.go):**

```go
func createBaseEntity(manager common.EntityManager, name, imagePath, assetDir string,
                      visible bool, pos *coords.LogicalPosition) *ecs.Entity {
    fpath := filepath.Join(assetDir, imagePath)
    img, _, err := ebitenutil.NewImageFromFile(fpath)
    if err != nil {
        log.Fatal(err)
    }

    entity := manager.World.NewEntity()
    entity.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
        Image: img, Visible: visible,
    })
    // ... other components
}
```

**Visual Effect Loading (gear/equipmentcomponents.go):**

```go
func (w *RangedWeapon) LoadShootingVX(vxPath string) {
    img, _, err := ebitenutil.NewImageFromFile(vxPath)
    if err != nil {
        log.Fatal(err)
    }
    w.ShootingVXImage = img
}
```

### JSON Template Loading

**Location:** `entitytemplates/readdata.go`

```go
// Global template storage
var (
    monsterTemplates    []JSONMonster
    weaponTemplates     []JSONMeleeWeapon
    consumableTemplates []JSONAttributeModifier
)

func ReadGameData() {
    readMonsterData()
    readWeaponData()
    readConsumableData()
}

func readMonsterData() {
    data, err := os.ReadFile("../assets/gamedata/monsterdata.json")
    if err != nil {
        log.Fatal(err)
    }

    var result struct {
        Monsters []JSONMonster `json:"monsters"`
    }

    json.Unmarshal(data, &result)
    monsterTemplates = result.Monsters
}

func GetMonsterTemplate(name string) JSONMonster {
    for _, monster := range monsterTemplates {
        if monster.Name == name {
            return monster
        }
    }
    log.Fatalf("Monster template not found: %s", name)
    return JSONMonster{}
}
```

**Usage:**

```go
// In main.go
entitytemplates.ReadGameData()

// In spawning code
template := entitytemplates.GetMonsterTemplate("GoblinWarrior")
config := entitytemplates.EntityConfig{
    Type:      entitytemplates.EntityCreature,
    Name:      template.Name,
    ImagePath: "../assets/monsters/" + template.Imgname,
    Position:  &spawnPos,
    GameMap:   gameMap,
}
creature := entitytemplates.CreateEntityFromTemplate(manager, config, template)
```

### Configuration Files

**Monster Data Format (monsterdata.json):**

```json
{
  "monsters": [
    {
      "name": "GoblinWarrior",
      "imgname": "goblin.png",
      "attributes": {
        "maxHealth": 15,
        "attackBonus": 1,
        "baseArmorClass": 10,
        "baseProtection": 1,
        "baseDodgeChance": 0.1,
        "baseMovementSpeed": 3,
        "damagebonus": 0
      },
      "armor": {
        "armorClass": 2,
        "protection": 1,
        "dodgeChance": 0.05
      },
      "meleeWeapon": {
        "minDamage": 2,
        "maxDamage": 3,
        "attackSpeed": 5
      },
      "rangedWeapon": null,
      "width": 1,
      "height": 1,
      "role": "Tank",
      "targetMode": "row",
      "targetRows": [0],
      "isMultiTarget": false,
      "maxTargets": 1
    }
  ]
}
```

**Weapon Data Format (weapondata.json):**

```json
{
  "weapons": [
    {
      "name": "IronSword",
      "imgname": "iron_sword.png",
      "minDamage": 3,
      "maxDamage": 7,
      "attackSpeed": 5
    }
  ]
}
```

**Consumable Data Format (consumabledata.json):**

```json
{
  "consumables": [
    {
      "name": "HealthPotion",
      "imgname": "potion_red.png",
      "duration": 0,
      "attributes": {
        "maxHealth": 0,
        "currentHealth": 25,
        "attackBonus": 0
      }
    }
  ]
}
```

---

## Data Structures & Formats

### Monster Data Schema

```go
type JSONMonster struct {
    Name         string           `json:"name"`
    Imgname      string           `json:"imgname"`
    Attributes   JSONAttributes   `json:"attributes"`
    Armor        *JSONArmor       `json:"armor"`        // Nullable
    MeleeWeapon  *JSONMeleeWeapon `json:"meleeWeapon"`  // Nullable
    RangedWeapon *JSONRangedWeapon `json:"rangedWeapon"` // Nullable

    // Squad system fields
    Width        int              `json:"width"`        // Multi-cell width (1-3)
    Height       int              `json:"height"`       // Multi-cell height (1-3)
    Role         string           `json:"role"`         // "Tank", "DPS", "Support"
    TargetMode   string           `json:"targetMode"`   // "row" or "cell"
    TargetRows   []int            `json:"targetRows"`   // Row-based targeting
    TargetCells  [][2]int         `json:"targetCells"`  // Cell-based targeting
    IsMultiTarget bool            `json:"isMultiTarget"`
    MaxTargets   int              `json:"maxTargets"`
}

type JSONAttributes struct {
    MaxHealth         int     `json:"maxHealth"`
    AttackBonus       int     `json:"attackBonus"`
    BaseArmorClass    int     `json:"baseArmorClass"`
    BaseProtection    int     `json:"baseProtection"`
    BaseDodgeChance   float32 `json:"baseDodgeChance"`
    BaseMovementSpeed int     `json:"baseMovementSpeed"`
    DamageBonus       int     `json:"damagebonus"`
}

type JSONArmor struct {
    ArmorClass  int     `json:"armorClass"`
    Protection  int     `json:"protection"`
    DodgeChance float32 `json:"dodgeChance"`
}
```

**Multi-Cell Unit Examples:**

| Unit Type | Width | Height | Description |
|-----------|-------|--------|-------------|
| GoblinWarrior | 1 | 1 | Standard 1x1 unit |
| TrollChampion | 2 | 1 | Wide 2x1 unit (cavalry) |
| DragonWhelp | 1 | 2 | Tall 1x2 unit |
| Boss (future) | 2 | 2 | Large 2x2 unit |

**Targeting Pattern Examples:**

| Pattern | TargetMode | TargetCells/Rows | Description |
|---------|-----------|------------------|-------------|
| Single Front | "row" | `[0]` | Front row only |
| All Rows | "row" | `[0,1,2]` | Piercing attack |
| 1x2 Cleave | "cell" | `[[0,0], [0,1]]` | Horizontal slash |
| 2x2 Blast | "cell" | `[[0,0], [0,1], [1,0], [1,1]]` | Area attack |
| Back Row Snipe | "row" | `[2]` | Archer targeting |

### Item Data Schema

```go
type JSONMeleeWeapon struct {
    MinDamage   int `json:"minDamage"`
    MaxDamage   int `json:"maxDamage"`
    AttackSpeed int `json:"attackSpeed"`
}

type JSONRangedWeapon struct {
    ShootingVX    string          `json:"shootingVX"`    // VX sprite path
    MinDamage     int             `json:"minDamage"`
    MaxDamage     int             `json:"maxDamage"`
    AttackSpeed   int             `json:"attackSpeed"`
    ShootingRange int             `json:"shootingRange"`
    TargetArea    JSONTargetArea  `json:"targetArea"`
}

type JSONTargetArea struct {
    Type   string `json:"type"`    // "Circle", "Line", "Cone", "Square"
    Radius int    `json:"radius"`  // For Circle
    Length int    `json:"length"`  // For Line/Cone
    Size   int    `json:"size"`    // For Square
}
```

**Target Area Examples:**

```json
// Circle AOE (radius 2)
"targetArea": {
  "type": "Circle",
  "radius": 2
}

// Line projectile (length 5)
"targetArea": {
  "type": "Line",
  "length": 5
}

// Cone spread (length 3)
"targetArea": {
  "type": "Cone",
  "length": 3
}

// Square blast (size 1 = 3x3)
"targetArea": {
  "type": "Square",
  "size": 1
}
```

### Consumable Data Schema

```go
type JSONAttributeModifier struct {
    Name       string         `json:"name"`
    Imgname    string         `json:"imgname"`
    Duration   int            `json:"duration"`    // 0 = instant
    Attributes JSONAttributes `json:"attributes"`
}
```

**Consumable Types:**

- **Duration = 0**: Instant effect (healing potions)
- **Duration > 0**: Temporary buff/debuff (strength potions, poison)

**Examples:**

```json
// Instant healing
{
  "name": "HealthPotion",
  "duration": 0,
  "attributes": {
    "currentHealth": 25
  }
}

// Temporary strength buff (5 turns)
{
  "name": "StrengthPotion",
  "duration": 5,
  "attributes": {
    "attackBonus": 3,
    "damageBonus": 2
  }
}
```

### Map Data Structure

```go
type GameMap struct {
    Tiles          []*Tile                                   // Flat array (Width * Height)
    Width          int
    Height         int
    Rooms          []Rect                                    // BSP-generated rooms
    PlayerVisible  *FOV                                      // Visibility tracker
    EntityTracker  map[*coords.LogicalPosition]*ecs.Entity  // Position → Entity
    RightEdgeX     int
}

type Tile struct {
    PixelX     int
    PixelY     int
    TileType   int           // Floor, Wall, etc.
    Blocked    bool
    Image      *ebiten.Image
    BlockSight bool
}

type Rect struct {
    X1, Y1 int  // Top-left
    X2, Y2 int  // Bottom-right
}

func (r Rect) Center() (int, int)
func (r Rect) Intersects(other Rect) bool
```

**FOV Data:**

```go
type FOV struct {
    VisibleTiles   map[int]bool  // Currently visible tile indices
    ExploredTiles  map[int]bool  // Ever-seen tile indices (for fog of war)
}
```

### Save Data (Not Implemented)

**Planned Structure:**

```go
type SaveData struct {
    Version      string
    Player       PlayerSaveData
    Map          MapSaveData
    Entities     []EntitySaveData
    Timestamp    time.Time
}

type PlayerSaveData struct {
    Position   coords.LogicalPosition
    Attributes common.Attributes
    Inventory  []ItemSaveData
    Equipment  EquipmentSaveData
}

type EntitySaveData struct {
    EntityID   string
    Template   string  // Template name for recreation
    Position   coords.LogicalPosition
    Components map[string]interface{}  // Serialized components
}
```

---

## Notable Architectural Patterns Discovered

### 1. **Pure ECS with Query-Based Systems**

The game demonstrates proper ECS architecture:
- Entities have no behavior (pure data containers)
- Components are plain structs
- Systems query entities by tags and execute logic

**Example:** Rendering system queries all entities with `RenderableComponent` + `PositionComponent`

### 2. **Data-Driven Design with JSON Templates**

Entity definitions live in JSON files, enabling:
- Balance changes without recompilation
- Designer-friendly stat editing
- Version control for game balance

### 3. **Type-Safe Coordinate System**

Three distinct position types prevent bugs:
- `LogicalPosition` (grid coordinates)
- `PixelPosition` (screen coordinates)
- Index (flat array position)

All conversions go through `CoordinateManager`.

### 4. **Quality-Based Procedural Generation**

Items and shapes scale based on quality tiers:
- `LowQuality`: Small AOE, low healing
- `NormalQuality`: Medium values
- `HighQuality`: Large AOE, high healing

### 5. **Component Composition via Factory Pattern**

Entity creation uses compositional factories:
- Base entity (name, position, sprite)
- + Weapon components
- + Armor components
- + Creature components
- = Complete entity

### 6. **Viewport-Centered Rendering**

Two rendering modes:
- Full map rendering (small maps)
- Viewport rendering with centering (large maps, scrolling)

Viewport system handles scaling and centering transforms.

### 7. **Visual Effect System (Non-ECS)**

Temporary visual effects (projectiles, explosions) bypass ECS:
- Global handler manages effect lifecycle
- Frame-based duration
- Automatic cleanup

### 8. **Input Priority Chain**

Input handling uses Chain of Responsibility:
1. UI (menus block lower-priority input)
2. Combat (throwing/shooting)
3. Movement (default action)

### 9. **Squad System ECS Best Practices**

The emerging squad system demonstrates production-quality ECS:
- Native `ecs.EntityID` for relationships (not entity pointers)
- Query-based relationship discovery
- Pure data components with zero logic
- System-based combat resolution

This pattern should be applied to legacy code (position tracking, weapon logic, item system).

### 10. **Unified Shape System**

8+ specialized shape types consolidated into single `BaseShape`:
- Type variants (Circular, Rectangular, Linear)
- Quality-based sizing
- Rotation support
- Factory pattern for creation

---

## Performance Characteristics

### Known Bottlenecks

1. **Position Tracking (O(n))**
   - Current: `map[*LogicalPosition]*Entity` uses pointer keys
   - Result: O(n) linear scan for every position lookup
   - Impact: 50x slower with 30+ entities
   - Fix: Migrate to `map[LogicalPosition][]EntityID` (value keys = O(1) hash)

2. **Creature Queries (O(n))**
   - `GetCreatureAtPosition()` queries all monsters every call
   - Impact: Slow combat/movement checks
   - Fix: Use spatial grid (planned with Position System refactoring)

3. **Renderable Queries (O(n))**
   - Queries all renderables every frame (~60 FPS)
   - Acceptable for <100 entities
   - May need spatial culling for >500 entities

4. **FOV Calculation (O(r²))**
   - Ray-casting FOV recalculates every movement
   - Impact: Minor on 100x80 maps
   - Optimization: Cache FOV when player doesn't move

### Optimization Opportunities

1. **Entity Pooling**: Reuse disposed entities instead of allocating new
2. **Spatial Partitioning**: Grid-based culling for rendering/collision
3. **Component Caching**: Cache frequently-accessed components
4. **FOV Dirty Checking**: Only recalculate on movement
5. **Asset Preloading**: Load all assets at startup (currently on-demand)

---

## Future Architecture Notes

### Squad System Integration

The squad system (35% complete) will require:

1. **Map Integration** (Phase 2)
   - Squad positioning on GameMap
   - Multi-cell unit collision handling
   - Formation rendering

2. **Input Integration** (Phase 2)
   - Squad selection UI
   - Formation control
   - Ability activation

3. **Combat Integration** (Phase 1.2-1.3)
   - Row-based targeting
   - Leader abilities
   - Multi-cell attack patterns

4. **Spawning Integration** (Phase 2.4)
   - Enemy squad spawning
   - Level-based difficulty scaling
   - Formation templates

### Legacy System Refactoring

Following squad system's ECS patterns:

1. **Position System** (Phase 0 - CRITICAL)
   - O(1) spatial grid
   - Native EntityID usage
   - 50x performance improvement

2. **Weapon System** (Phase 3.2)
   - Extract logic to WeaponSystem
   - Pure data components
   - Query-based targeting

3. **Item System** (Phase 3.3)
   - Flatten nested entity structure
   - Remove circular references
   - Simplify quality management

4. **Movement System** (Phase 3.4)
   - Extract to MovementSystem
   - Pathfinding integration
   - Speed-based movement cost

---

## Appendices

### A. Glossary

- **ECS**: Entity Component System
- **AOE**: Area of Effect
- **FOV**: Field of View
- **BSP**: Binary Space Partitioning
- **LOC**: Lines of Code
- **VX**: Visual Effects
- **UI**: User Interface
- **AC**: Armor Class
- **HP**: Hit Points
- **DPS**: Damage Per Second (or Damage role)

### B. File Statistics

**Total Go Files:** 67
**Total JSON Files:** 8
**Package Count:** 20
**Estimated LOC:** ~15,000 (including squad system)

**Largest Packages:**
1. graphics (~1,500 LOC)
2. gear (~1,200 LOC)
3. worldmap (~1,000 LOC)
4. gui (~900 LOC)
5. squads (~621 LOC)

### C. External Dependencies

```go
// Core dependencies
github.com/bytearena/ecs           // ECS library
github.com/hajimehoshi/ebiten/v2   // Game engine
github.com/ebitenui/ebitenui       // UI widgets

// Standard library
encoding/json     // JSON parsing
image            // Image handling
image/png        // PNG loading
log              // Logging
math             // Math utilities
math/rand        // Random generation
os               // File I/O
path/filepath    // Path utilities
runtime          // Performance profiling
net/http         // HTTP server (pprof)
net/http/pprof   // Profiling tools
```

### D. Build & Run Commands

```bash
# Build
go build -o game_main/game_main.exe game_main/*.go

# Run
go run game_main/*.go

# Test
go test ./...

# Clean
go clean

# Install dependencies
go mod tidy

# Enable profiling
# Navigate to http://localhost:6060/debug/pprof/ when ENABLE_BENCHMARKING = true
```

### E. Configuration Constants

**Location:** `graphics/graphictypes.go`, `coords/cordmanager.go`

```go
// Graphics
const MAP_SCROLLING_ENABLED = true
const ViewableSquareSize = 50
const StatsUIOffset = 500

// Map
DungeonWidth  = 100
DungeonHeight = 80
TileSize      = 32
ScaleFactor   = 3

// Debug
DEBUG_MODE = false
ENABLE_BENCHMARKING = false
```

### F. Known Issues & TODOs

**From Code Comments:**

1. Position tracking needs O(1) optimization (trackers/creaturetracker.go)
2. GetCreatureAtPosition does full monster query (common/ecsutil.go)
3. Coordinate conversion called every frame, should cache (coords/cordmanager.go)
4. Fist attack for unarmed combat not implemented (combat/attackingsystem.go)
5. Entity removal on player death not handled (combat/attackingsystem.go)
6. Squad combat system 65% incomplete (squads/)
7. Status effect tracker needs implementation (trackers/statustracker.go)
8. Save/load system not implemented

---

## Document Metadata

**Generated:** 2025-10-08
**Codebase Version:** main branch, commit 22c3dca
**Total Sections:** 7 main sections
**Total Packages Documented:** 20
**Word Count:** ~18,000
**Analysis Methodology:**
- Static code analysis via file reading
- JSON schema extraction
- Pattern identification via cross-file analysis
- Component dependency mapping
- ECS query flow tracing

**Files Analyzed:**
- 67 Go source files
- 8 JSON data files
- 1 project configuration file (CLAUDE.md)

---

**End of Documentation**
