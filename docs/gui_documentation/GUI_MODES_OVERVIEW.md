# GUI Modes Overview

**Last Updated:** 2025-11-29

This document provides a comprehensive overview of TinkerRogue's GUI mode system, including the architecture, available modes, and how they work together to provide a seamless user experience.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Two-Context System](#two-context-system)
3. [Mode Catalog](#mode-catalog)
4. [Mode Transitions](#mode-transitions)
5. [Implementation Patterns](#implementation-patterns)
6. [Adding New Modes](#adding-new-modes)

---

## Architecture Overview

### Core Components

TinkerRogue's GUI system is built on a **mode-based architecture** that separates different UI contexts into distinct, self-contained modes. This architecture consists of three primary components:

#### 1. UIMode Interface (`gui/core/uimode.go`)

The `UIMode` interface defines the contract that all UI modes must implement:

```go
type UIMode interface {
    Initialize(ctx *UIContext) error  // Called once when mode is created
    Enter(fromMode UIMode) error      // Called when switching TO this mode
    Exit(toMode UIMode) error         // Called when switching FROM this mode
    Update(deltaTime float64) error   // Called every frame
    Render(screen *ebiten.Image)      // Called to draw mode-specific visuals
    HandleInput(inputState *InputState) bool  // Processes input events
    GetEbitenUI() *ebitenui.UI       // Returns the ebitenui root container
    GetModeName() string              // Returns mode identifier
}
```

**Key Lifecycle:**
- `Initialize()`: One-time setup (build UI widgets, create components)
- `Enter()`: Refresh data, restore state when becoming active
- `Update()`: Per-frame logic updates
- `HandleInput()`: Process keyboard/mouse input
- `Render()`: Custom rendering (overlays, highlights, etc.)
- `Exit()`: Clean up, save state when leaving mode

#### 2. UIModeManager (`gui/core/modemanager.go`)

Manages a collection of modes and handles transitions between them:

```go
type UIModeManager struct {
    currentMode       UIMode
    modes             map[string]UIMode
    context           *UIContext
    pendingTransition *ModeTransition
    inputState        *InputState
}
```

**Responsibilities:**
- Register modes by name
- Switch between modes (immediate or deferred)
- Maintain input state (keyboard, mouse)
- Update and render the current mode
- Process pending transitions at frame boundaries

#### 3. GameModeCoordinator (`gui/core/gamemodecoordinator.go`)

Coordinates **two separate UIModeManagers** for different game contexts:

```go
type GameModeCoordinator struct {
    overworldManager  *UIModeManager  // Strategic layer modes
    battleMapManager  *UIModeManager  // Tactical layer modes
    activeManager     *UIModeManager  // Currently active manager
    currentContext    GameContext     // Overworld or BattleMap
    overworldState    *OverworldState // Persistent overworld data
    battleMapState    *BattleMapState // Persistent battle data
}
```

**Key Feature:** Enables **context switching** between strategic (Overworld) and tactical (BattleMap) gameplay layers while preserving state in each context.

---

## Two-Context System

TinkerRogue separates gameplay into two distinct contexts, each with its own set of modes:

### Context: Overworld (Strategic Layer)

**Purpose:** Squad management, unit purchasing, formation editing, world map navigation

**Modes:**
- Squad Management
- Squad Builder
- Formation Editor
- Unit Purchase

**Access:** Press **Ctrl+Tab** or use "Squads (E)" button from Battle Map

**Characteristics:**
- Menu-driven interfaces
- Turn-based strategic decisions
- No real-time map interaction
- Persistent squad modifications

### Context: Battle Map (Tactical Layer)

**Purpose:** Dungeon exploration, combat, inventory management, tile inspection

**Modes:**
- Exploration Mode
- Combat Mode
- Inventory Mode
- Squad Deployment Mode
- Info Inspection Mode

**Access:** Press **Ctrl+Tab** or use "Battle Map (ESC)" button from Overworld

**Characteristics:**
- Map-centric gameplay
- Real-time exploration
- Tactical combat
- Position-based interactions

### Context Switching Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Game Mode Coordinator                     │
│  (Manages two independent UIModeManagers + state)           │
└─────────────────────────────────────────────────────────────┘
                  │                           │
         ┌────────┴─────────┐        ┌───────┴────────┐
         │  Overworld       │        │  BattleMap     │
         │  UIModeManager   │◄──────►│  UIModeManager │
         └──────────────────┘        └────────────────┘
                  │                           │
         ┌────────┴─────────┐        ┌───────┴────────┐
         │ Strategic Modes  │        │ Tactical Modes │
         │ - Squad Mgmt     │        │ - Exploration  │
         │ - Formation      │        │ - Combat       │
         │ - Unit Purchase  │        │ - Inventory    │
         │ - Squad Builder  │        │ - Deployment   │
         └──────────────────┘        └────────────────┘
```

---

## Mode Catalog

### Battle Map Context Modes

#### Exploration Mode
**File:** `gui/guimodes/explorationmode.go`
**Mode Name:** `"exploration"`
**Purpose:** Default mode during dungeon exploration

**Features:**
- Stats panel (top-right) - displays player attributes
- Message log (bottom-right) - game messages
- Quick access buttons (bottom-center):
  - Throwables
  - Squads (E) - switches to Overworld context
  - Inventory (I)
  - Deploy (D)
  - Combat (C)

**Hotkeys:**
- `I` → Inventory Mode
- `C` → Combat Mode
- `D` → Squad Deployment Mode
- `E` → Overworld/Squad Management (context switch)
- `Right-Click` → Info Inspection Mode (on clicked tile)

**Input Handling:**
- Delegates to existing input coordinator for movement
- Right-click inspection opens Info Mode with clicked position

---

#### Combat Mode
**File:** `gui/guicombat/combatmode.go`
**Mode Name:** `"combat"`
**Purpose:** Turn-based tactical squad combat

**Features:**
- Turn order panel (top-center) - shows current faction and round
- Faction info panel (top-right) - squad count, mana
- Squad list panel (left) - current faction's squads (player turn only)
- Squad detail panel (right) - selected squad details
- Combat log (bottom-right) - combat events and messages
- Action buttons (bottom-center):
  - Attack
  - Move
  - Undo Move
  - Redo Move
  - End Turn
  - Flee

**Combat Flow:**
1. Combat initialized with all factions (from ECS queries)
2. Turn order determined by TurnManager (via CombatService)
3. Player selects squad from list (only during player's turn)
4. Player selects action (attack/move)
5. Click on map to execute action
6. End turn → next faction's turn
7. Victory/defeat detection

**Rendering:**
- Squad highlights (colored by faction)
- Valid movement tiles (blue overlay in move mode)
- Attack ranges

**Hotkeys:**
- `Space` → End Turn
- `ESC` → Flee combat (returns to Exploration)

**Services:**
- `CombatService` - owns TurnManager, FactionManager, MovementSystem
- `CombatActionHandler` - processes attack/move actions
- `CombatInputHandler` - handles map clicks and squad selection

---

#### Inventory Mode
**File:** `gui/guimodes/inventorymode.go`
**Mode Name:** `"inventory"`
**Purpose:** Full-screen inventory browsing and item management

**Features:**
- Filter buttons (top-left): All, Throwables, Equipment, Consumables
- Item list (left) - filterable inventory
- Detail panel (right) - selected item details
- Close button (bottom-center)

**Item Selection:**
- **Throwables filter:** Selecting item prepares it for throwing, returns to Exploration
- **Other filters:** Shows item details only

**Hotkeys:**
- `I` → Close Inventory (return to Exploration)
- `ESC` → Close Inventory

**Services:**
- `InventoryService` - validates throwable selection, retrieves item data

---

#### Squad Deployment Mode
**File:** `gui/guisquads/squaddeploymentmode.go`
**Mode Name:** `"squad_deployment"`
**Purpose:** Place squads on the map before combat

**Features:**
- Squad list panel (left) - all alive squads
- Instruction text (top-center) - current action
- Action buttons (bottom-center):
  - Clear All - removes all squad positions
  - Start Combat - transitions to Combat Mode
  - Close (ESC)

**Deployment Flow:**
1. Select squad from list
2. Click on map to place squad
3. Repeat for all squads
4. Click "Start Combat" to begin

**Rendering:**
- Squad highlights (all placed squads)
- Currently selected squad highlight (brighter)

**Hotkeys:**
- `ESC` → Return to Exploration

**Services:**
- `SquadDeploymentService` - manages squad position assignments

---

#### Info Inspection Mode
**File:** `gui/guimodes/infomode.go`
**Mode Name:** `"info_inspect"`
**Purpose:** Display detailed information about clicked tiles/entities

**Features:**
- Options list (left) - inspection options:
  - Look at Creature
  - Look at Tile
- Detail panel (right) - inspection results

**Inspection Data:**
- **Creature:** Name, Type (Player/Monster), HP, Attributes (STR/DEX/MAG/LDR/ARM/WPN)
- **Tile:** Position, Type, Movement Cost, Walkable status, Entity presence

**Access:**
- Right-click on any tile in Exploration Mode

**Hotkeys:**
- `ESC` → Return to Exploration

**Services:**
- `GUIQueries` - abstracts ECS queries for creature/tile data

---

### Overworld Context Modes

#### Squad Management Mode
**File:** `gui/guisquads/squadmanagementmode.go`
**Mode Name:** `"squad_management"`
**Purpose:** View and manage individual squads

**Features:**
- Single squad panel (center) - displays one squad at a time:
  - Squad name
  - 3x3 grid visualization
  - Squad stats (units, HP, morale)
  - Unit list (scrollable)
- Navigation controls (center):
  - Previous button
  - Squad counter ("Squad 1 of 3")
  - Next button
- Command buttons (center):
  - Rename Squad
  - Disband Squad
  - Merge Squads
  - Undo
  - Redo
- Action buttons (bottom-center):
  - Battle Map (ESC)
  - Squad Builder (B)
  - Formation (F)
  - Buy Units (P)

**Commands (Undo/Redo Enabled):**
- **Rename:** Text input dialog → `RenameSquadCommand`
- **Disband:** Confirmation dialog → `DisbandSquadCommand` (returns units to roster)
- **Merge:** Squad selection dialog → confirmation → `MergeSquadsCommand`

**Hotkeys:**
- `B` → Squad Builder Mode
- `F` → Formation Editor Mode
- `P` → Unit Purchase Mode
- `ESC` → Return to Battle Map/Exploration
- `Ctrl+Z` → Undo
- `Ctrl+Y` → Redo

**Services:**
- `CommandHistory` - undo/redo stack for squad operations

---

#### Squad Builder Mode
**File:** `gui/guisquads/squadbuilder.go`
**Mode Name:** `"squad_builder"`
**Purpose:** Create new squads from available roster units

**Features:**
- Roster panel (left) - available units not in squads
- Grid editor (center) - 3x3 squad formation grid
- Selected squad info (right) - current squad being built
- Action buttons:
  - Create Squad
  - Add to Existing Squad
  - Clear Grid

**Build Flow:**
1. Select unit from roster
2. Click on grid cell to place unit
3. Repeat to fill formation
4. Create new squad or add to existing

**Commands (Undo/Redo Enabled):**
- **Create Squad:** `CreateSquadCommand`
- **Add to Squad:** `AddUnitsToSquadCommand`

---

#### Formation Editor Mode
**File:** `gui/guisquads/formationeditormode.go`
**Mode Name:** `"formation_editor"`
**Purpose:** Edit the 3x3 formation of existing squads

**Features:**
- Squad selector (left) - choose which squad to edit
- 3x3 grid editor (center) - drag/drop units to positions
- Unit palette (left) - unit types for placement
- Action buttons (bottom-center):
  - Apply Formation
  - Close (ESC)
  - Undo
  - Redo

**Editing Flow:**
1. Select squad from list
2. Current formation loads into grid
3. Select unit type from palette
4. Click grid cells to place/remove units
5. Apply Formation to commit changes

**Commands (Undo/Redo Enabled):**
- **Apply Formation:** Confirmation dialog → `ChangeFormationCommand`

**Hotkeys:**
- `ESC` → Return to Squad Management
- `Ctrl+Z` → Undo
- `Ctrl+Y` → Redo

---

#### Unit Purchase Mode
**File:** `gui/guisquads/unitpurchasemode.go`
**Mode Name:** `"unit_purchase"`
**Purpose:** Buy new units and add them to the roster

**Features:**
- Unit template list (left) - all available unit types with owned counts
- Detail panel (right):
  - Unit info (name, cost, role, size)
  - View Stats button
  - Stats display (HP, attributes, combat stats)
- Resource display (top-center):
  - Gold amount
  - Roster count/capacity
- Action buttons (bottom-center):
  - Buy Unit
  - Undo
  - Redo
  - Back (ESC)

**Purchase Flow:**
1. Select unit template from list
2. View unit details and stats
3. Click "Buy Unit" (if affordable and roster has space)
4. Unit added to roster (available for squad building)

**Validation:**
- Checks player gold
- Checks roster capacity
- Buy button disabled if unable to purchase

**Commands (Undo/Redo Enabled):**
- **Purchase Unit:** `PurchaseUnitCommand`

**Hotkeys:**
- `ESC` → Return to Squad Management
- `Ctrl+Z` → Undo
- `Ctrl+Y` → Redo

**Services:**
- `UnitPurchaseService` - validates purchases, manages gold/roster

---

## Mode Transitions

### Transition Types

#### 1. Immediate Transition
```go
modeManager.SetMode("mode_name")
```
Transitions immediately within current update cycle.

#### 2. Deferred Transition
```go
modeManager.RequestTransition(toMode, "reason")
```
Transition queued, executed at end of frame (after widget events processed).

### Common Transition Patterns

#### Battle Map Modes
```
Exploration ←→ Combat
Exploration ←→ Inventory
Exploration ←→ Squad Deployment
Exploration ←→ Info Inspection
```

#### Overworld Modes
```
Squad Management ←→ Squad Builder
Squad Management ←→ Formation Editor
Squad Management ←→ Unit Purchase
```

#### Context Switches
```
Exploration (Battle Map) ←→ Squad Management (Overworld)
```

### Transition Flow Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                      Mode Transition                          │
└──────────────────────────────────────────────────────────────┘
                            │
                ┌───────────┴───────────┐
                │                       │
         ┌──────▼──────┐         ┌─────▼─────┐
         │ Immediate   │         │ Deferred  │
         │ SetMode()   │         │ Request() │
         └──────┬──────┘         └─────┬─────┘
                │                      │
                │         ┌────────────┘
                │         │
         ┌──────▼─────────▼──────┐
         │ transitionToMode()    │
         └───────────────────────┘
                │
      ┌─────────┴─────────┐
      │                   │
  ┌───▼────┐        ┌─────▼─────┐
  │ Exit() │        │ Enter()   │
  │ (old)  │        │ (new)     │
  └────────┘        └───────────┘
```

### Hotkey Summary

#### Battle Map Context
| Key | Mode | Action |
|-----|------|--------|
| `I` | Exploration | Open Inventory |
| `C` | Exploration | Enter Combat |
| `D` | Exploration | Squad Deployment |
| `E` | Exploration | Switch to Overworld/Squad Management |
| `Space` | Combat | End Turn |
| `ESC` | Any | Return to Exploration (or previous mode) |
| `Right-Click` | Exploration | Info Inspection |

#### Overworld Context
| Key | Mode | Action |
|-----|------|--------|
| `B` | Squad Management | Squad Builder |
| `F` | Squad Management | Formation Editor |
| `P` | Squad Management | Unit Purchase |
| `ESC` | Any | Return to Squad Management or Battle Map |
| `Ctrl+Z` | Any with CommandHistory | Undo |
| `Ctrl+Y` | Any with CommandHistory | Redo |

#### Global
| Key | Context | Action |
|-----|---------|--------|
| `Ctrl+Tab` | Any | Toggle between Overworld and Battle Map |

---

## Implementation Patterns

### BaseMode Infrastructure

All modes inherit from `gui.BaseMode` which provides:

```go
type BaseMode struct {
    ModeName       string
    ReturnMode     string          // Mode to return to on ESC
    Context        *core.UIContext
    RootContainer  *widget.Container
    EbitenUI       *ebitenui.UI
    ModeManager    *core.UIModeManager

    // Common systems
    Queries        *guicomponents.GUIQueries
    PanelBuilders  *widgets.PanelBuilders
    Layout         *widgets.LayoutInfo
    CommandHistory *gui.CommandHistory
    StatusLabel    *widget.Text

    // Hotkey registry
    hotkeys        map[ebiten.Key]string
}
```

**Key Methods:**
- `InitializeBase(ctx)` - Sets up common infrastructure
- `HandleCommonInput(inputState)` - Processes ESC and registered hotkeys
- `RegisterHotkey(key, modeName)` - Maps key to mode transition
- `InitializeCommandHistory(callback)` - Sets up undo/redo
- `SetStatus(message)` - Updates status label

### UI Component Patterns

#### Stats Display Component
```go
statsComponent := guicomponents.NewStatsDisplayComponent(
    textArea,
    formatterFunc,
)
statsComponent.RefreshStats(playerData, manager)
```

#### Squad List Component
```go
squadListComponent := guicomponents.NewSquadListComponent(
    container,
    queries,
    filterFunc,
    onSelectFunc,
)
squadListComponent.Refresh()
squadListComponent.SetFilter("All")
```

#### Detail Panel Component
```go
detailComponent := guicomponents.NewDetailPanelComponent(
    textWidget,
    queries,
    formatterFunc,
)
detailComponent.ShowSquad(squadID)
detailComponent.ShowFaction(factionID)
```

### Service Layer Integration

Modes delegate business logic to service classes:

```go
// Combat Mode
combatService := combatservices.NewCombatService(manager)
combatService.InitializeCombat(factionIDs)
result := combatService.EndTurn()

// Inventory Mode
inventoryService := gear.NewInventoryService(manager)
result := inventoryService.SelectThrowable(playerID, itemIndex)

// Squad Deployment
deploymentService := squadservices.NewSquadDeploymentService(manager)
result := deploymentService.PlaceSquadAtPosition(squadID, pos)

// Unit Purchase
purchaseService := squadservices.NewUnitPurchaseService(manager)
validation := purchaseService.CanPurchaseUnit(playerID, template)
```

**Benefits:**
- UI code focuses on presentation
- Business logic testable independently
- Consistent validation across modes
- Service methods return `ServiceResult` with success/error

### Command Pattern (Undo/Redo)

Modes with `CommandHistory` execute commands for undoable actions:

```go
// Create command
cmd := squadcommands.NewRenameSquadCommand(
    manager,
    squadID,
    newName,
)

// Execute via history (enables undo/redo)
commandHistory.Execute(cmd)

// Undo/Redo
commandHistory.Undo()
commandHistory.Redo()
```

**Command Interface:**
```go
type Command interface {
    Execute() error
    Undo() error
    Description() string
}
```

**Available Commands:**
- `RenameSquadCommand`
- `DisbandSquadCommand`
- `MergeSquadsCommand`
- `CreateSquadCommand`
- `AddUnitsToSquadCommand`
- `ChangeFormationCommand`
- `PurchaseUnitCommand`

---

## Adding New Modes

### Step-by-Step Guide

#### 1. Create Mode File

```go
package mypackage

import (
    "game_main/gui"
    "game_main/gui/core"
    "github.com/hajimehoshi/ebiten/v2"
)

type MyNewMode struct {
    gui.BaseMode  // Embed common infrastructure

    // Mode-specific fields
    myPanel *widget.Container
}

func NewMyNewMode(modeManager *core.UIModeManager) *MyNewMode {
    mode := &MyNewMode{}
    mode.SetModeName("my_new_mode")
    mode.SetReturnMode("exploration")  // ESC returns here
    mode.ModeManager = modeManager
    return mode
}
```

#### 2. Implement UIMode Interface

```go
func (m *MyNewMode) Initialize(ctx *core.UIContext) error {
    // Initialize base infrastructure
    m.InitializeBase(ctx)

    // Register hotkeys if needed
    m.RegisterHotkey(ebiten.KeyM, "other_mode")

    // Build UI widgets
    m.buildUI()

    return nil
}

func (m *MyNewMode) Enter(fromMode core.UIMode) error {
    // Refresh data when entering mode
    fmt.Println("Entering My New Mode")
    return nil
}

func (m *MyNewMode) Exit(toMode core.UIMode) error {
    // Clean up when leaving mode
    fmt.Println("Exiting My New Mode")
    return nil
}

func (m *MyNewMode) Update(deltaTime float64) error {
    // Per-frame updates
    return nil
}

func (m *MyNewMode) Render(screen *ebiten.Image) {
    // Custom rendering (overlays, highlights)
}

func (m *MyNewMode) HandleInput(inputState *core.InputState) bool {
    // Handle common input first (ESC, hotkeys)
    if m.HandleCommonInput(inputState) {
        return true
    }

    // Handle mode-specific input
    return false
}
```

#### 3. Register Mode

In `game_main/main.go` or appropriate initialization code:

```go
// For Battle Map context
newMode := mypackage.NewMyNewMode(coordinator.GetBattleMapManager())
coordinator.RegisterBattleMapMode(newMode)

// For Overworld context
newMode := mypackage.NewMyNewMode(coordinator.GetOverworldManager())
coordinator.RegisterOverworldMode(newMode)
```

#### 4. Add Transitions

From other modes, add buttons or hotkeys:

```go
// Button
btn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text: "Open My Mode",
    OnClick: func() {
        if myMode, exists := modeManager.GetMode("my_new_mode"); exists {
            modeManager.RequestTransition(myMode, "User clicked button")
        }
    },
})

// Hotkey (in Initialize)
mode.RegisterHotkey(ebiten.KeyM, "my_new_mode")
```

### Best Practices

1. **Use BaseMode:** Always embed `gui.BaseMode` for common infrastructure
2. **Defer Transitions:** Use `RequestTransition()` for UI-triggered mode changes
3. **Delegate Logic:** Create service classes for business logic
4. **Component-Based UI:** Use `guicomponents` for reusable display logic
5. **Query Abstraction:** Use `GUIQueries` instead of direct ECS access
6. **Command Pattern:** Implement undo/redo for user actions
7. **State Separation:** UI state in mode, game state in ECS components
8. **Return Modes:** Set `ReturnMode` for ESC key behavior
9. **Context Awareness:** Register modes to appropriate context (Overworld vs BattleMap)

---

## State Management

### UI State vs Game State

**UI State (Transient):**
- Current selection (selected squad, item, etc.)
- UI mode flags (in attack mode, in move mode, etc.)
- Scroll positions, expanded panels
- **Storage:** Mode fields, BattleMapState, OverworldState

**Game State (Persistent):**
- Squad positions, formations, members
- Combat state (turn order, faction data)
- Inventory contents
- Unit attributes
- **Storage:** ECS components

### BattleMapState

Persistent state for Battle Map context:

```go
type BattleMapState struct {
    SelectedSquadID  ecs.EntityID
    InAttackMode     bool
    InMoveMode       bool
    ValidMoveTiles   []coords.LogicalPosition
    // ... other UI state
}
```

**Access:** `ctx.ModeCoordinator.GetBattleMapState()`

### OverworldState

Persistent state for Overworld context:

```go
type OverworldState struct {
    // TODO: Implement overworld state tracking
}
```

### Context Switch Persistence

When switching contexts (Ctrl+Tab):
1. Current context saves state to `*State` structure
2. New context activated
3. New context restores state from `*State` structure
4. Modes in new context can access preserved state

---

## References

### Core Files

- `gui/core/uimode.go` - UIMode interface
- `gui/core/modemanager.go` - UIModeManager
- `gui/core/gamemodecoordinator.go` - GameModeCoordinator
- `gui/basemode.go` - BaseMode infrastructure

### Mode Files

**Battle Map:**
- `gui/guimodes/explorationmode.go`
- `gui/guicombat/combatmode.go`
- `gui/guimodes/inventorymode.go`
- `gui/guisquads/squaddeploymentmode.go`
- `gui/guimodes/infomode.go`

**Overworld:**
- `gui/guisquads/squadmanagementmode.go`
- `gui/guisquads/squadbuilder.go`
- `gui/guisquads/formationeditormode.go`
- `gui/guisquads/unitpurchasemode.go`

### Supporting Systems

- `gui/guicomponents/` - Reusable display components
- `gui/widgets/` - Widget creation helpers
- `squads/squadcommands/` - Undoable squad commands
- `squads/squadservices/` - Squad business logic
- `combat/combatservices/` - Combat business logic
- `gear/inventoryservice.go` - Inventory business logic

---

## Conclusion

TinkerRogue's mode-based GUI architecture provides:

✅ **Clear Separation:** Each mode is self-contained with distinct purpose
✅ **Context Switching:** Seamless transition between strategic and tactical layers
✅ **State Management:** Persistent state across context switches
✅ **Reusable Components:** Shared UI components and patterns
✅ **Undo/Redo:** Command pattern for reversible actions
✅ **Service Layer:** Business logic separated from presentation
✅ **Extensibility:** Easy to add new modes following established patterns

This architecture enables rapid UI development while maintaining clean separation between presentation (modes), business logic (services), and data (ECS components).
