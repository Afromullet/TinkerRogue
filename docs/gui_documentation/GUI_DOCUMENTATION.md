# TinkerRogue GUI System Documentation

**Version:** 4.0
**Last Updated:** 2026-01-11

Comprehensive technical documentation for TinkerRogue's GUI system. This guide covers the current architecture, mode-based system, panel registry, mode builder pattern, and implementation details.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Core Concepts](#core-concepts)
3. [Package Structure](#package-structure)
4. [Mode Infrastructure](#mode-infrastructure)
5. [Panel Building System](#panel-building-system)
6. [Widget System](#widget-system)
7. [Input Handling](#input-handling)
8. [Context Switching](#context-switching)
9. [Service Integration](#service-integration)
10. [Performance Optimization](#performance-optimization)
11. [Mode Catalog](#mode-catalog)
12. [Adding New Modes](#adding-new-modes)
13. [Best Practices](#best-practices)
14. [Common Patterns](#common-patterns)
15. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

### System Philosophy

TinkerRogue's GUI system follows a **mode-based architecture** where the game's UI is organized into distinct, self-contained modes. Each mode represents a specific player context (exploration, combat, squad management, etc.) with its own UI layout, input handling, and lifecycle.

**Key Architectural Principles:**

1. **Mode-Based Organization**: UI is divided into cohesive modes rather than monolithic state machines
2. **Declarative Configuration**: Modes use builders and registries instead of imperative UI construction
3. **Separation of Concerns**: UI state (modes) is separate from game state (ECS)
4. **Query Abstraction**: UI code accesses game data through a query layer (GUIQueries), not direct ECS
5. **Component Reuse**: Common UI update patterns are encapsulated in reusable components
6. **Panel Registry System**: Panels are registered once and built declaratively
7. **Performance First**: Caching, batching, and smart rendering reduce CPU/memory overhead

### High-Level Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                     GameModeCoordinator                          │
│   (Manages two independent contexts with state preservation)    │
└────────────────┬────────────────────────────┬───────────────────┘
                 │                            │
     ┌───────────▼────────────┐   ┌──────────▼────────────┐
     │  Overworld Context     │   │  BattleMap Context    │
     │  (UIModeManager)       │   │  (UIModeManager)      │
     └────────────────────────┘   └───────────────────────┘
                 │                            │
     ┌───────────▼────────────┐   ┌──────────▼────────────┐
     │ Strategic Modes        │   │ Tactical Modes        │
     │ • Squad Management     │   │ • Exploration         │
     │ • Squad Builder        │   │ • Combat              │
     │ • Squad Editor         │   │ • Inventory           │
     │ • Unit Purchase        │   │ • Squad Deployment    │
     │ • Squad Deployment     │   │                       │
     └────────────────────────┘   └───────────────────────┘
```

### Technology Stack

- **UI Framework**: [ebitenui](https://github.com/ebitenui/ebitenui) - Widget-based UI library
- **Rendering**: [Ebiten](https://ebiten.org/) - 2D game engine
- **ECS**: Custom entity-component-system for game state
- **Layout**: Anchor layout (absolute positioning) + Row layout (stacking)

---

## Core Concepts

### UIMode Interface

Every UI mode implements the `UIMode` interface, which defines the mode lifecycle:

```go
type UIMode interface {
    Initialize(ctx *UIContext) error  // One-time setup
    Enter(fromMode UIMode) error      // Called when mode becomes active
    Exit(toMode UIMode) error         // Called when leaving mode
    Update(deltaTime float64) error   // Per-frame updates
    Render(screen *ebiten.Image)      // Custom rendering (overlays)
    HandleInput(inputState *InputState) bool  // Input processing
    GetEbitenUI() *ebitenui.UI       // Returns ebitenui root
    GetModeName() string              // Mode identifier
}
```

**File:** `gui/framework/uimode.go`

### UIContext

Shared game state provided to all modes:

```go
type UIContext struct {
    ECSManager      *common.EntityManager
    PlayerData      *common.PlayerData
    ScreenWidth     int
    ScreenHeight    int
    TileSize        int
    ModeCoordinator *GameModeCoordinator
    Queries         interface{} // *GUIQueries
}
```

### BaseMode

Common mode infrastructure that all modes embed:

```go
type BaseMode struct {
    // UI Infrastructure
    ui            *ebitenui.UI
    RootContainer *widget.Container
    Layout        *specs.LayoutConfig
    PanelBuilders *builders.PanelBuilders

    // Services
    Context        *UIContext
    Queries        *GUIQueries
    ModeManager    *UIModeManager

    // Optional Features
    StatusLabel    *widget.Text
    CommandHistory *CommandHistory
    Panels         *PanelRegistry

    // Internal
    modeName       string
    returnMode     string
    hotkeys        map[ebiten.Key]InputBinding
    self           UIMode  // Reference to concrete mode
}
```

**File:** `gui/framework/basemode.go`

### Mode Lifecycle

```
┌──────────────┐
│ Initialize() │  ← Called once when mode is registered
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Enter()    │  ← Called when mode becomes active
└──────┬───────┘
       │
       ▼  (repeats while active)
┌──────────────┐
│   Update()   │  ← Called every frame
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Render()    │  ← Called to draw overlays
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Exit()     │  ← Called when leaving mode
└──────────────┘
```

---

## Package Structure

The GUI system is organized into specialized packages:

```
gui/
├── framework/              # Core mode infrastructure
│   ├── uimode.go          # UIMode interface, UIContext
│   ├── basemode.go        # Common mode infrastructure
│   ├── modemanager.go     # Mode lifecycle & transitions
│   ├── coordinator.go     # Two-context system (Overworld/BattleMap)
│   ├── contextstate.go    # Context-specific UI state
│   ├── modebuilder.go     # Declarative mode configuration
│   ├── panelregistry.go   # Global panel type registry
│   ├── guiqueries.go      # ECS query abstraction
│   ├── commandhistory.go  # Undo/redo system
│   ├── helpers.go         # Shared utility functions
│   ├── debuglogger.go     # Debug logging for modes
│   └── squadinfo_cache.go # Performance caching for squad data
│
├── builders/              # UI construction helpers
│   ├── panels.go          # Panel building with functional options
│   ├── layout.go          # Layout calculations & positioning
│   ├── dialogs.go         # Modal dialog builders
│   ├── lists.go           # List widget builders
│   └── panelspecs.go      # Standard panel specifications
│
├── widgets/               # Widget wrappers & utilities
│   ├── cached_list.go     # Cached list rendering (90% CPU reduction)
│   ├── cached_textarea.go # Cached text area rendering
│   ├── createwidgets.go   # Widget creation helpers
│   ├── layout_constants.go # Responsive sizing constants
│   └── textdisplay.go     # Auto-updating text components
│
├── specs/                 # Layout specifications
│   └── layout.go          # Responsive layout configuration
│
├── guiresources/          # Shared UI resources
│   ├── guiresources.go    # Font faces, button graphics
│   └── cachedbackground.go # Cached background rendering
│
├── guicombat/             # Combat mode implementation
│   ├── combatmode.go              # Main combat mode
│   ├── combatdeps.go              # Dependency injection
│   ├── combatlifecycle.go         # Combat initialization/cleanup
│   ├── combatvisualization.go     # Visual rendering
│   ├── combat_action_handler.go   # Attack/move execution
│   ├── combat_input_handler.go    # Map click handling
│   ├── combat_log_manager.go      # Combat log formatting
│   ├── combat_panels_registry.go  # Combat panel registration
│   ├── combat_animation_mode.go   # Animation playback mode
│   └── squad_combat_renderer.go   # Squad rendering utilities
│
├── guisquads/             # Squad management modes
│   ├── squadmanagementmode.go         # Squad overview/navigation
│   ├── squadeditormode.go             # Formation editor
│   ├── squadbuilder.go                # Create new squads
│   ├── squaddeploymentmode.go         # Place squads on map
│   ├── unitpurchasemode.go            # Buy units
│   ├── squadcomponents.go             # Shared squad UI components
│   ├── squadeditor_grid.go            # 3x3 grid editor
│   ├── squadeditor_roster.go          # Unit roster display
│   ├── squadeditor_refresh.go         # UI refresh logic
│   ├── squad_builder_grid_manager.go  # Grid state management
│   └── *_panels_registry.go           # Panel registrations
│
└── guimodes/              # Other game modes
    ├── explorationmode.go         # Default exploration mode
    ├── inventorymode.go           # Full-screen inventory
    ├── guirenderers.go            # Custom rendering utilities
    ├── itemlistcomponent.go       # Inventory list component
    └── *_panels_registry.go       # Panel registrations
```

### Key Package Responsibilities

**framework/**: Core abstractions (UIMode, BaseMode, ModeManager, Coordinator)
**builders/**: Panel and widget construction helpers
**widgets/**: Performance-optimized widget wrappers
**specs/**: Layout specifications and responsive sizing
**guiresources/**: Shared fonts, images, backgrounds
**guicombat/**: All combat-related UI
**guisquads/**: All squad management UI
**guimodes/**: Exploration, inventory, and other gameplay modes

---

## Mode Infrastructure

### ModeBuilder Pattern

**File:** `gui/framework/modebuilder.go`

The ModeBuilder provides declarative configuration for mode initialization, eliminating boilerplate code.

**Usage Example:**

```go
func (m *MyMode) Initialize(ctx *framework.UIContext) error {
    err := framework.NewModeBuilder(&m.BaseMode, framework.ModeConfig{
        ModeName:   "my_mode",
        ReturnMode: "exploration",

        Hotkeys: []framework.HotkeySpec{
            {Key: ebiten.KeyI, TargetMode: "inventory"},
            {Key: ebiten.KeyC, TargetMode: "combat"},
        },

        StatusLabel: true,  // Create status label
        Commands:    true,  // Enable undo/redo
        OnRefresh:   m.refreshUI,
    }).Build(ctx)

    if err != nil {
        return err
    }

    // Build panels from registry
    return m.BuildPanels(MyPanelType1, MyPanelType2)
}
```

**What ModeBuilder Does:**

1. Sets mode name and return mode
2. Initializes BaseMode (creates UI, queries, panel builders)
3. Registers hotkeys for mode transitions
4. Creates status label (if configured)
5. Initializes command history (if configured)

**Benefits:**

- Reduces mode `Initialize()` from ~50 lines to ~10 lines
- Consistent initialization pattern across all modes
- Declarative configuration reveals intent
- No missed initialization steps

### Panel Registry System

**File:** `gui/framework/panelregistry.go`

The Panel Registry provides **type-safe, declarative panel building**. Panels are registered globally (typically in `init()` functions) and built on-demand by modes.

**Architecture:**

```
┌─────────────────────────────────────────────────────────────────┐
│                     Global Panel Registry                        │
│  (Initialized at startup via init() functions)                  │
└─────────────────────────────────────────────────────────────────┘
                               │
                    ┌──────────┴──────────┐
                    │ RegisterPanel()     │
                    │ (Called in init())  │
                    └──────────┬──────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────┐
│  Panel Type → Panel Descriptor → Build Function                 │
│  • CombatPanelTurnOrder   → Spec + Content + OnCreate          │
│  • CombatPanelSquadList   → Spec + Content + OnCreate          │
│  • SquadEditorPanelGrid   → Spec + Content + OnCreate          │
└─────────────────────────────────────────────────────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │ BaseMode.BuildPanels()│
                    │ (Called in Initialize)│
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │  PanelRegistry      │
                    │  (Mode-local store) │
                    └─────────────────────┘
```

**Registration Example:**

```go
// File: gui/guicombat/combat_panels_registry.go

func init() {
    // Register turn order panel
    framework.RegisterPanel(CombatPanelTurnOrder, framework.PanelDescriptor{
        SpecName: "turn_order",  // Uses StandardPanels spec
        Content:  framework.ContentText,
        OnCreate: func(pr *framework.PanelResult, mode framework.UIMode) error {
            pr.TextLabel = builders.CreateLargeLabel("Turn Order")
            pr.Container.AddChild(pr.TextLabel)
            return nil
        },
    })

    // Register squad list panel
    framework.RegisterPanel(CombatPanelSquadList, framework.PanelDescriptor{
        SpecName: "squad_list",
        Content:  framework.ContentList,
        OnCreate: func(pr *framework.PanelResult, mode framework.UIMode) error {
            pr.List = builders.CreateListWithConfig(...)
            pr.Container.AddChild(pr.List)
            return nil
        },
    })
}
```

**Building Panels in Mode:**

```go
func (cm *CombatMode) Initialize(ctx *framework.UIContext) error {
    // ... ModeBuilder setup ...

    // Build panels from registry
    panels := []framework.PanelType{
        CombatPanelTurnOrder,
        CombatPanelSquadList,
        CombatPanelSquadDetail,
        CombatPanelCombatLog,
    }

    return cm.BuildPanels(panels...)
}
```

**Accessing Built Panels:**

```go
// Type-safe accessor pattern
func GetTurnOrderLabel(panels *framework.PanelRegistry) *widget.Text {
    if panel, exists := panels.Get(CombatPanelTurnOrder); exists {
        return panel.TextLabel
    }
    return nil
}

// Usage in mode
func (cm *CombatMode) Enter(fromMode framework.UIMode) error {
    label := GetTurnOrderLabel(cm.Panels)
    if label != nil {
        label.Label = "Round 1 - Player Turn"
    }
    return nil
}
```

**Benefits:**

- **Type Safety**: Panel types are strongly typed constants
- **Centralized Definition**: Panel specs defined once, used everywhere
- **Separation of Concerns**: Panel structure separate from mode logic
- **Easy Refactoring**: Change panel layout in one place
- **Conditional Building**: Modes can selectively build panels

### GUIQueries - Query Abstraction

**File:** `gui/framework/guiqueries.go`

GUIQueries provides a **unified query interface** for UI code to access ECS data without direct ECS coupling.

**Architecture:**

```go
type GUIQueries struct {
    ECSManager     *common.EntityManager
    // Additional services as needed
}

// Data Transfer Objects (DTOs)
type SquadInfo struct {
    ID                ecs.EntityID
    Name              string
    UnitIDs           []ecs.EntityID
    AliveUnits        int
    CurrentHP         int
    MaxHP             int
    Position          *coords.LogicalPosition
    IsDestroyed       bool
    HasActed          bool
    MovementRemaining int
}
```

**Common Query Methods:**

```go
// Squad queries
squadInfo := queries.GetSquadInfo(squadID)
allSquads := queries.GetAllSquads()
enemySquads := queries.GetEnemySquads(currentFactionID)

// Faction queries
factionInfo := queries.GetFactionInfo(factionID)
allFactions := queries.GetAllFactions()

// Position queries
creatureInfo := queries.GetCreatureAtPosition(logicalPos)
tileInfo := queries.GetTileInfo(logicalPos)
```

**Benefits:**

- **Encapsulation**: UI doesn't know about ECS implementation
- **Stability**: Query interface stays stable even if ECS changes
- **Testability**: Easy to mock for UI tests
- **Performance**: Can add caching at query layer
- **Type Safety**: Returns DTOs instead of raw component data

---

## Panel Building System

### Three Levels of Abstraction

**Level 1: Panel Registry (Highest)**

Use for standard panels defined once and reused:

```go
// Registration (in init())
framework.RegisterPanel(MyPanelType, framework.PanelDescriptor{
    SpecName: "standard_panel",
    Content:  framework.ContentText,
})

// Building (in mode)
cm.BuildPanels(MyPanelType)
```

**Level 2: Standard Panel Specs**

Use for common layouts without custom widgets:

```go
panel := builders.CreateStandardPanel(
    panelBuilders,
    "faction_info",  // StandardPanels entry
)
```

**File:** `gui/builders/panelspecs.go` defines `StandardPanels` map.

**Level 3: Functional Options (Custom)**

Use for unique layouts not in registry or specs:

```go
panel := panelBuilders.BuildPanel(
    builders.TopRight(),
    builders.Size(0.2, 0.15),
    builders.Padding(0.015),
    builders.RowLayout(),
)
```

**File:** `gui/builders/panels.go`

### Functional Options Reference

**Position Options:**

```go
builders.TopLeft()
builders.TopCenter()
builders.TopRight()
builders.LeftCenter()
builders.Center()
builders.RightCenter()
builders.BottomLeft()
builders.BottomCenter()
builders.BottomRight()
```

**Size Options:**

```go
builders.Size(widthPercent, heightPercent float64)
```

**Padding Options:**

```go
builders.Padding(percent float64)           // Uniform padding
builders.CustomPadding(insets widget.Insets) // Custom insets
```

**Layout Options:**

```go
builders.RowLayout()           // Vertical stacking
builders.HorizontalRowLayout() // Horizontal stacking
builders.AnchorLayout()        // Absolute positioning
```

### Responsive Sizing Constants

**File:** `gui/widgets/layout_constants.go`

All dimensions use percentage-based values:

```go
// Panel widths (% of screen width)
PanelWidthNarrow     = 0.15  // 15%
PanelWidthStandard   = 0.2   // 20%
PanelWidthMedium     = 0.3   // 30%
PanelWidthWide       = 0.4   // 40%

// Panel heights (% of screen height)
PanelHeightTiny      = 0.08  // 8%
PanelHeightSmall     = 0.12  // 12%
PanelHeightQuarter   = 0.25  // 25%
PanelHeightHalf      = 0.5   // 50%
PanelHeightTall      = 0.75  // 75%

// Padding (% of screen width)
PaddingExtraSmall    = 0.0125
PaddingTight         = 0.015
PaddingStandard      = 0.02
PaddingLoose         = 0.03
```

**Usage:**

```go
width := int(float64(screenWidth) * widgets.PanelWidthStandard)
padding := int(float64(screenWidth) * widgets.PaddingTight)
```

---

## Widget System

### Declarative Widget Creation

**File:** `gui/builders/widgets.go`

All widgets use **config structs** for declarative creation:

**Text Widgets:**

```go
// Simple labels
titleLabel := builders.CreateLargeLabel("Squad Management")
statusLabel := builders.CreateSmallLabel("Ready")

// Custom text
customText := builders.CreateTextWithConfig(builders.TextConfig{
    Text:     "Custom Message",
    FontFace: guiresources.LargeFace,
    Color:    color.White,
})
```

**Button Widgets:**

```go
saveBtn := builders.CreateButtonWithConfig(builders.ButtonConfig{
    Text: "Save Squad",
    OnClick: func() {
        // Handle click
    },
})
```

**List Widgets:**

```go
itemList := builders.CreateListWithConfig(builders.ListConfig{
    Entries: []interface{}{"Item 1", "Item 2"},
    EntryLabelFunc: func(e interface{}) string {
        return e.(string)
    },
    OnEntrySelected: func(e interface{}) {
        handleSelection(e)
    },
    MinWidth:  200,
    MinHeight: 300,
})
```

**Text Area Widgets:**

```go
detailArea := builders.CreateTextAreaWithConfig(builders.TextAreaConfig{
    MinWidth:  400,
    MinHeight: 300,
    FontColor: color.White,
})
```

### Performance-Optimized Widgets

**Cached List Wrapper**

**File:** `gui/widgets/cached_list.go`

Reduces CPU usage by ~90% for lists that don't update every frame:

```go
list := builders.CreateListWithConfig(...)
cachedList := widgets.NewCachedListWrapper(list)

// IMPORTANT: Mark dirty when list changes
cachedList.MarkDirty()
```

**When to use:**

- Squad lists (update when selection changes)
- Inventory lists (update when filter changes)
- Unit rosters (update when units added/removed)

**When NOT to use:**

- Lists that update every frame
- Lists with animated content

**Cached Text Area**

**File:** `gui/widgets/cached_textarea.go`

Similar caching for text areas:

```go
textArea := builders.CreateTextAreaWithConfig(...)
cachedTextArea := widgets.NewCachedTextAreaWrapper(textArea)

// Mark dirty when text changes
cachedTextArea.MarkDirty()
```

### Modal Dialogs

**File:** `gui/builders/dialogs.go`

**Confirmation Dialog:**

```go
confirmDialog := builders.CreateConfirmationDialog(builders.DialogConfig{
    Title:   "Disband Squad",
    Message: "Are you sure?",
    OnConfirm: func() {
        // Execute action
    },
    OnCancel: func() {
        // Handle cancel
    },
})
ui.AddWindow(confirmDialog)
```

**Text Input Dialog:**

```go
inputDialog := builders.CreateTextInputDialog(builders.TextInputDialogConfig{
    Title:       "Rename Squad",
    Message:     "Enter new name:",
    Placeholder: "Squad name",
    InitialText: currentName,
    OnConfirm: func(text string) {
        // Handle input
    },
})
ui.AddWindow(inputDialog)
```

**Message Dialog:**

```go
messageDialog := builders.CreateMessageDialog(builders.MessageDialogConfig{
    Title:   "Success",
    Message: "Squad created!",
    OnClose: func() {
        // Dialog closed
    },
})
ui.AddWindow(messageDialog)
```

---

## Input Handling

### InputState Structure

**File:** `gui/framework/uimode.go`

```go
type InputState struct {
    // Mouse
    MouseX            int
    MouseY            int
    MousePressed      bool
    MouseReleased     bool
    MouseButton       ebiten.MouseButton

    // Keyboard
    KeysPressed       map[ebiten.Key]bool   // Held this frame
    KeysJustPressed   map[ebiten.Key]bool   // Pressed THIS frame

    // Bridge to existing system
    PlayerInputStates *common.PlayerInputStates
}
```

### Hotkey Registration

Declarative hotkey registration via ModeBuilder or manually:

```go
// Via ModeBuilder
framework.NewModeBuilder(&m.BaseMode, framework.ModeConfig{
    Hotkeys: []framework.HotkeySpec{
        {Key: ebiten.KeyI, TargetMode: "inventory"},
        {Key: ebiten.KeyC, TargetMode: "combat"},
    },
}).Build(ctx)

// Manually
mode.RegisterHotkey(ebiten.KeyE, "squad_management")
```

### Input Handling Pattern

```go
func (m *MyMode) HandleInput(inputState *framework.InputState) bool {
    // 1. Handle common input (ESC, registered hotkeys)
    if m.HandleCommonInput(inputState) {
        return true  // Input consumed
    }

    // 2. Handle command history (Ctrl+Z, Ctrl+Y)
    if m.CommandHistory != nil && m.CommandHistory.HandleInput(inputState) {
        return true
    }

    // 3. Handle mode-specific input
    if inputState.KeysJustPressed[ebiten.KeySpace] {
        m.handleAction()
        return true
    }

    // 4. Handle mouse input
    if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
        m.handleClick(inputState.MouseX, inputState.MouseY)
        return true
    }

    return false  // Input not consumed
}
```

**IMPORTANT:** Always use `KeysJustPressed` for single-press actions, not `KeysPressed`.

### Modifier Keys

```go
// Ctrl+Z for undo
if inputState.KeysJustPressed[ebiten.KeyZ] &&
   inputState.KeysPressed[ebiten.KeyControl] {
    m.CommandHistory.Undo()
    return true
}

// Ctrl+Y for redo
if inputState.KeysJustPressed[ebiten.KeyY] &&
   inputState.KeysPressed[ebiten.KeyControl] {
    m.CommandHistory.Redo()
    return true
}
```

---

## Context Switching

**File:** `gui/framework/coordinator.go`

TinkerRogue uses a **two-context system** to separate strategic (Overworld) and tactical (BattleMap) gameplay.

### Game Contexts

```go
type GameContext int

const (
    ContextOverworld GameContext = iota  // Squad management, world map
    ContextBattleMap                     // Dungeon exploration, combat
)
```

**Overworld Context:**
- Squad Management
- Squad Builder
- Squad Editor
- Unit Purchase
- Squad Deployment (pre-combat)

**BattleMap Context:**
- Exploration
- Combat
- Inventory
- Squad Deployment (in-dungeon)

### GameModeCoordinator

Manages two independent `UIModeManager` instances:

```go
type GameModeCoordinator struct {
    overworldManager *UIModeManager   // Overworld modes
    battleMapManager *UIModeManager   // BattleMap modes
    activeManager    *UIModeManager   // Currently active
    currentContext   GameContext
    overworldState   *OverworldState  // Persistent UI state
    battleMapState   *BattleMapState  // Persistent UI state
}
```

### Context Switching Operations

**Switch to Overworld:**

```go
if context.ModeCoordinator != nil {
    context.ModeCoordinator.ReturnToOverworld("squad_management")
}
```

**Switch to BattleMap:**

```go
if context.ModeCoordinator != nil {
    context.ModeCoordinator.EnterBattleMap("exploration")
}
```

**Toggle Context (Ctrl+Tab):**

Handled automatically in `GameModeCoordinator.Update()`.

### State Persistence

**File:** `gui/framework/contextstate.go`

Each context has persistent state that survives context switches:

```go
// Overworld state
type OverworldState struct {
    SelectedSquadID    ecs.EntityID
    SquadListScroll    int
    SquadIDs           []ecs.EntityID
    EditingSquadID     ecs.EntityID
}

// BattleMap state
type BattleMapState struct {
    SelectedSquadID  ecs.EntityID
    InAttackMode     bool
    InMoveMode       bool
    ValidMoveTiles   []coords.LogicalPosition
}
```

**Access in Modes:**

```go
battleState := ctx.ModeCoordinator.GetBattleMapState()
battleState.InMoveMode = true
battleState.ValidMoveTiles = calculatedTiles
```

---

## Service Integration

### Command Pattern (Undo/Redo)

**File:** `gui/framework/commandhistory.go`

Provides undo/redo support for user actions:

**Setup (via ModeBuilder):**

```go
framework.NewModeBuilder(&m.BaseMode, framework.ModeConfig{
    Commands:  true,
    OnRefresh: m.refreshUI,
}).Build(ctx)
```

**Executing Commands:**

```go
cmd := squadcommands.NewRenameSquadCommand(squadID, newName, ecsManager)
success := mode.CommandHistory.Execute(cmd)
// Shows status: "✓ Renamed squad to Alpha" or "✗ Error"
```

**Undo/Redo:**

```go
mode.CommandHistory.Undo()  // Shows: "⟲ Undone: Rename squad"
mode.CommandHistory.Redo()  // Shows: "⟳ Redone: Rename squad"
```

**Creating Custom Commands:**

```go
type MyCommand struct {
    // Command data
}

func (c *MyCommand) Execute() error {
    // Perform action
    return nil
}

func (c *MyCommand) Undo() error {
    // Reverse action
    return nil
}

func (c *MyCommand) Description() string {
    return "Custom action"
}
```

### Rendering Utilities

**File:** `gui/guimodes/guirenderers.go`

**ViewportRenderer:**

Base renderer for viewport-centered drawing:

```go
renderer := guimodes.NewViewportRenderer(screen, playerPos)

// Draw tile overlay
renderer.DrawTileOverlay(screen, tilePos, color.RGBA{R: 0, G: 255, B: 0, A: 80})

// Draw tile border
renderer.DrawTileBorder(screen, tilePos, color.White, 3)
```

**MovementTileRenderer:**

Renders valid movement tiles:

```go
movementRenderer := guimodes.NewMovementTileRenderer()
movementRenderer.Render(screen, playerPos, validTiles)
```

**SquadHighlightRenderer:**

Renders squad highlights:

```go
highlightRenderer := guimodes.NewSquadHighlightRenderer(queries)
highlightRenderer.Render(screen, playerPos, currentFactionID, selectedSquadID)
```

---

## Performance Optimization

### Caching Strategies

**File:** `gui/framework/squadinfo_cache.go`

**SquadInfo Caching:**

```go
type SquadInfoCache struct {
    cache      map[ecs.EntityID]*SquadInfo
    lastUpdate time.Time
    ttl        time.Duration
}

// Usage in modes
squadInfo := cache.Get(squadID, queries)
cache.Invalidate(squadID)  // When squad changes
```

**Background Caching:**

**File:** `gui/guiresources/cachedbackground.go`

Pre-renders panel backgrounds:

```go
cachedBg := guiresources.NewCachedBackground(width, height, panelImage)
cachedBg.Draw(screen, x, y)
```

### Widget Performance

**Cached List (90% CPU reduction):**

```go
cachedList := widgets.NewCachedListWrapper(list)
cachedList.MarkDirty()  // When entries change
```

**Cached TextArea:**

```go
cachedTextArea := widgets.NewCachedTextAreaWrapper(textArea)
cachedTextArea.MarkDirty()  // When text changes
```

### Best Practices

1. **Cache Static Data**: Squad lists, rosters, unit templates
2. **Batch Updates**: Update UI once per game state change, not per frame
3. **Lazy Refresh**: Only refresh UI components when mode enters or data changes
4. **Selective Rendering**: Use `Render()` only for overlays, let ebitenui handle widgets
5. **Avoid Polling**: Use event-driven updates instead of checking state every frame

---

## Mode Catalog

### BattleMap Context Modes

#### Exploration Mode

**File:** `gui/guimodes/explorationmode.go`
**Mode Name:** `"exploration"`

Default mode during dungeon exploration.

**Panels:**
- Stats panel (top-right)
- Message log (bottom-right)
- Quick action buttons (bottom-center)

**Hotkeys:**
- `I` → Inventory Mode
- `C` → Combat Mode
- `D` → Squad Deployment Mode
- `E` → Overworld (Squad Management)
- `Right-Click` → Info Inspection

---

#### Combat Mode

**File:** `gui/guicombat/combatmode.go`
**Mode Name:** `"combat"`

Turn-based tactical squad combat.

**Panels:**
- Turn order (top-center)
- Faction info (top-left)
- Squad list (left-center)
- Squad detail (left-bottom)
- Combat log (bottom-right)
- Action buttons (bottom-center)
- Layer status (indicates AI processing)

**Combat Flow:**
1. Initialize factions
2. Player selects squad
3. Player selects action (attack/move)
4. Click map to execute
5. End turn → next faction

**Hotkeys:**
- `Space` → End Turn
- `ESC` → Flee Combat

**Services:**
- `CombatService` - Turn manager, movement
- `CombatActionHandler` - Attack/move execution
- `CombatInputHandler` - Map click handling

---

#### Inventory Mode

**File:** `gui/guimodes/inventorymode.go`
**Mode Name:** `"inventory"`

Full-screen inventory browsing.

**Panels:**
- Filter buttons (top-left)
- Item list (left)
- Item detail (right)
- Close button (bottom-center)

**Hotkeys:**
- `I` → Close
- `ESC` → Close

---

### Overworld Context Modes

#### Squad Management Mode

**File:** `gui/guisquads/squadmanagementmode.go`
**Mode Name:** `"squad_management"`

Squad overview and management.

**Panels:**
- Action buttons (bottom-center)

**Hotkeys:**
- `B` → Squad Builder
- `P` → Unit Purchase
- `E` → Squad Editor
- `ESC` → Return to BattleMap

**Features:**
- Undo/redo support
- Command pattern for squad operations

---

#### Squad Editor Mode

**File:** `gui/guisquads/squadeditormode.go`
**Mode Name:** `"squad_editor"`

Edit 3x3 squad formations.

**Panels:**
- Squad selector (left)
- 3x3 grid editor (center)
- Unit roster (right)
- Action buttons (bottom-center)

**Hotkeys:**
- `ESC` → Return to Squad Management
- `Ctrl+Z` → Undo
- `Ctrl+Y` → Redo

---

#### Squad Builder Mode

**File:** `gui/guisquads/squadbuilder.go`
**Mode Name:** `"squad_builder"`

Create new squads from roster units.

**Panels:**
- Roster panel (left)
- Grid editor (center)
- Squad info (right)
- Action buttons (bottom-center)

**Features:**
- Undo/redo support
- Create new squad or add to existing

---

#### Unit Purchase Mode

**File:** `gui/guisquads/unitpurchasemode.go`
**Mode Name:** `"unit_purchase"`

Buy new units.

**Panels:**
- Unit template list (left)
- Unit detail (right)
- Resource display (top-center)
- Action buttons (bottom-center)

**Hotkeys:**
- `ESC` → Return to Squad Management
- `Ctrl+Z` → Undo
- `Ctrl+Y` → Redo

---

## Adding New Modes

### Step-by-Step Guide

#### 1. Create Mode File

```go
package mypackage

import (
    "game_main/gui/framework"
    "github.com/hajimehoshi/ebiten/v2"
)

type MyNewMode struct {
    framework.BaseMode

    // Mode-specific fields
}

func NewMyNewMode(modeManager *framework.UIModeManager) *MyNewMode {
    mode := &MyNewMode{}
    mode.SetModeName("my_new_mode")
    mode.SetReturnMode("exploration")
    mode.SetSelf(mode)  // Required for panel registry
    mode.ModeManager = modeManager
    return mode
}
```

#### 2. Implement Initialize()

```go
func (m *MyNewMode) Initialize(ctx *framework.UIContext) error {
    // Use ModeBuilder for common setup
    err := framework.NewModeBuilder(&m.BaseMode, framework.ModeConfig{
        ModeName:   "my_new_mode",
        ReturnMode: "exploration",

        Hotkeys: []framework.HotkeySpec{
            {Key: ebiten.KeyM, TargetMode: "other_mode"},
        },

        StatusLabel: true,
        Commands:    false,
    }).Build(ctx)

    if err != nil {
        return err
    }

    // Build panels from registry
    return m.BuildPanels(MyPanelType1, MyPanelType2)
}
```

#### 3. Implement Lifecycle Methods

```go
func (m *MyNewMode) Enter(fromMode framework.UIMode) error {
    // Refresh UI when entering
    return nil
}

func (m *MyNewMode) Exit(toMode framework.UIMode) error {
    // Clean up when leaving
    return nil
}

func (m *MyNewMode) Update(deltaTime float64) error {
    // Per-frame updates (usually minimal)
    return nil
}

func (m *MyNewMode) Render(screen *ebiten.Image) {
    // Custom rendering (overlays only)
}

func (m *MyNewMode) HandleInput(inputState *framework.InputState) bool {
    // Handle common input first
    if m.HandleCommonInput(inputState) {
        return true
    }

    // Mode-specific input
    if inputState.KeysJustPressed[ebiten.KeySpace] {
        m.handleAction()
        return true
    }

    return false
}
```

#### 4. Register Panels (Optional)

```go
// File: mypackage/my_panels_registry.go

const (
    MyPanelType1 framework.PanelType = "my_panel_1"
    MyPanelType2 framework.PanelType = "my_panel_2"
)

func init() {
    framework.RegisterPanel(MyPanelType1, framework.PanelDescriptor{
        SpecName: "standard_panel",
        Content:  framework.ContentText,
        OnCreate: func(pr *framework.PanelResult, mode framework.UIMode) error {
            pr.TextLabel = builders.CreateLargeLabel("My Panel")
            pr.Container.AddChild(pr.TextLabel)
            return nil
        },
    })
}
```

#### 5. Register Mode

```go
// In main.go or mode registration code
coordinator := framework.NewGameModeCoordinator(ctx)

// For BattleMap context
newMode := mypackage.NewMyNewMode(coordinator.GetBattleMapManager())
coordinator.RegisterBattleMapMode(newMode)

// For Overworld context
newMode := mypackage.NewMyNewMode(coordinator.GetOverworldManager())
coordinator.RegisterOverworldMode(newMode)
```

### Implementation Checklist

**Mode Setup:**
- [ ] Create mode struct embedding `framework.BaseMode`
- [ ] Implement constructor calling `SetModeName()`, `SetReturnMode()`, `SetSelf()`
- [ ] Set `ModeManager` in constructor

**Initialization:**
- [ ] Use `ModeBuilder` for common setup
- [ ] Register hotkeys via `ModeConfig`
- [ ] Enable StatusLabel/Commands if needed
- [ ] Build panels from registry

**Lifecycle:**
- [ ] Implement `Enter()` to refresh UI
- [ ] Implement `Exit()` to clean up (if needed)
- [ ] Implement `Update()` for periodic updates (if needed)
- [ ] Implement `Render()` for custom overlays (if needed)

**Input:**
- [ ] Implement `HandleInput()`
- [ ] Call `HandleCommonInput()` first
- [ ] Handle command history input (if enabled)
- [ ] Use `KeysJustPressed` for single-press actions
- [ ] Return `true` when input consumed

**Integration:**
- [ ] Use `Queries` for ECS access (never direct)
- [ ] Use service layers for game logic
- [ ] Use `CommandHistory` for undoable actions (if enabled)

**Testing:**
- [ ] Test mode initialization
- [ ] Test input handling
- [ ] Test panel building
- [ ] Test mode transitions

---

## Best Practices

### UI Development Guidelines

#### 1. Always Use ModeBuilder

```go
// ✅ GOOD - Declarative configuration
framework.NewModeBuilder(&m.BaseMode, framework.ModeConfig{
    ModeName:   "my_mode",
    ReturnMode: "exploration",
    Hotkeys:    []framework.HotkeySpec{...},
}).Build(ctx)

// ❌ BAD - Manual initialization
m.InitializeBase(ctx)
m.SetModeName("my_mode")
m.RegisterHotkey(ebiten.KeyI, "inventory")
// ... many more lines ...
```

#### 2. Use Panel Registry

```go
// ✅ GOOD - Declarative panel building
m.BuildPanels(CombatPanelTurnOrder, CombatPanelSquadList)

// ❌ BAD - Manual panel creation
turnOrderPanel := m.PanelBuilders.BuildPanel(
    builders.TopCenter(),
    builders.Size(0.4, 0.08),
    // ... many options ...
)
m.RootContainer.AddChild(turnOrderPanel)
```

#### 3. Query Abstraction

```go
// ✅ GOOD - Use GUIQueries
squadInfo := m.Queries.GetSquadInfo(squadID)

// ❌ BAD - Direct ECS access
squadData := common.GetComponentType[*squads.SquadData](entity, squads.SquadComponent)
```

#### 4. Cache Performance-Critical Widgets

```go
// ✅ GOOD - Cached list
cachedList := widgets.NewCachedListWrapper(list)
cachedList.MarkDirty()  // When entries change

// ❌ BAD - Raw list (re-renders every frame)
container.AddChild(list)
```

#### 5. Separate UI State from Game State

```go
// ✅ GOOD - UI state in BattleMapState
battleState.InMoveMode = true
battleState.ValidMoveTiles = tiles

// ❌ BAD - UI state in ECS components
squadData.UIInMoveMode = true  // Don't pollute ECS with UI concerns
```

### Common Pitfalls

**Pitfall 1: Forgetting to Call SetSelf()**

```go
// ✅ CORRECT
func NewMyMode(modeManager *framework.UIModeManager) *MyMode {
    mode := &MyMode{}
    mode.SetSelf(mode)  // Required for panel registry
    return mode
}

// ❌ WRONG - Panel building will fail
func NewMyMode(modeManager *framework.UIModeManager) *MyMode {
    mode := &MyMode{}
    // Missing SetSelf()
    return mode
}
```

**Pitfall 2: Using KeysPressed Instead of KeysJustPressed**

```go
// ✅ CORRECT - Single action on press
if inputState.KeysJustPressed[ebiten.KeySpace] {
    m.handleAction()
}

// ❌ WRONG - Action repeats every frame while held
if inputState.KeysPressed[ebiten.KeySpace] {
    m.handleAction()  // Fires 60 times per second!
}
```

**Pitfall 3: Not Marking Cached Widgets Dirty**

```go
// ✅ CORRECT
list.SetEntries(newEntries)
cachedList.MarkDirty()  // Force re-render

// ❌ WRONG - UI shows stale data
list.SetEntries(newEntries)
// Forgot MarkDirty() - cache still shows old entries
```

**Pitfall 4: Modifying ECS Directly from UI**

```go
// ✅ CORRECT - Use service layer
result := combatService.EndTurn()

// ❌ WRONG - UI modifies ECS directly
actionState.HasActed = true  // Bypasses game logic
```

---

## Common Patterns

### Pattern: Refreshing UI on Enter

```go
func (m *MyMode) Enter(fromMode framework.UIMode) error {
    // Refresh all UI components
    m.refreshSquadList()
    m.refreshDetailsPanel()
    m.refreshStatusLabel()
    return nil
}
```

### Pattern: Type-Safe Panel Access

```go
// Define accessor in panel registry file
func GetTurnOrderLabel(panels *framework.PanelRegistry) *widget.Text {
    if panel, exists := panels.Get(CombatPanelTurnOrder); exists {
        return panel.TextLabel
    }
    return nil
}

// Use in mode
label := GetTurnOrderLabel(m.Panels)
if label != nil {
    label.Label = "Round 1"
}
```

### Pattern: Conditional Panel Building

```go
panels := []framework.PanelType{
    CombatPanelTurnOrder,
    CombatPanelSquadList,
}

// Add log only if enabled
if config.ENABLE_COMBAT_LOG {
    panels = append(panels, CombatPanelCombatLog)
}

return m.BuildPanels(panels...)
```

### Pattern: Dependency Injection

```go
// Consolidate dependencies in struct
type CombatModeDeps struct {
    battleState   *framework.BattleMapState
    combatService *combatservices.CombatService
    queries       *framework.GUIQueries
    logArea       *widget.TextArea
    logManager    *CombatLogManager
    modeManager   *framework.UIModeManager
}

// Pass to handlers
actionHandler := NewCombatActionHandler(deps)
inputHandler := NewCombatInputHandler(actionHandler, deps)
```

---

## Troubleshooting

### Problem: Panels Don't Appear

**Solution:** Ensure `SetSelf()` is called in constructor:

```go
func NewMyMode(...) *MyMode {
    mode := &MyMode{}
    mode.SetSelf(mode)  // Required!
    return mode
}
```

### Problem: Input Not Working

**Solution:** Check `HandleInput()` returns `true` when consuming input:

```go
func (m *MyMode) HandleInput(inputState *framework.InputState) bool {
    if inputState.KeysJustPressed[ebiten.KeySpace] {
        m.handleAction()
        return true  // IMPORTANT: Consume input
    }
    return false
}
```

### Problem: UI Shows Stale Data

**Solution:** Refresh UI in `Enter()`:

```go
func (m *MyMode) Enter(fromMode framework.UIMode) error {
    m.refreshAllComponents()  // Refresh on enter
    return nil
}
```

### Problem: Cached List Not Updating

**Solution:** Call `MarkDirty()` when list changes:

```go
list.SetEntries(newEntries)
cachedList.MarkDirty()  // Force re-render
```

### Problem: Mode Transition Fails

**Solution:** Ensure mode is registered:

```go
coordinator.RegisterBattleMapMode(myMode)
```

---

## Glossary

**BaseMode** - Common mode infrastructure embedded in all modes.

**Context** - Game layer (Overworld or BattleMap) with separate mode managers.

**DTO** - Data Transfer Object returned by GUIQueries.

**ModeBuilder** - Declarative configuration builder for mode initialization.

**Panel Registry** - Global registry mapping panel types to build functions.

**PanelResult** - Container holding built panel and widget references.

**UIContext** - Shared game state passed to all modes.

**UIMode** - Interface defining mode lifecycle (Initialize, Enter, Exit, Update, Render, HandleInput).

**UIModeManager** - Manages mode lifecycle and transitions within a context.

---

**End of Documentation**
