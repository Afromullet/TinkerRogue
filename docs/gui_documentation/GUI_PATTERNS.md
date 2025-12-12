# TinkerRogue GUI Patterns & Architecture

**Version:** 2.1
**Last Updated:** 2025-12-12
**Related Documentation:** See [DOCUMENTATION.md](../DOCUMENTATION.md) Section 9: "GUI Architecture" for architectural overview.

Comprehensive technical documentation for TinkerRogue's GUI system. This guide covers architectural patterns, common implementations, and best practices for building UI features.

> **Note:** This document provides deep-dive patterns for GUI implementation. For high-level architecture, component catalog, and data flow patterns, see the main DOCUMENTATION.md.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Widget Creation Patterns](#widget-creation-patterns)
3. [Layout Patterns](#layout-patterns)
4. [Panel Building System](#panel-building-system)
5. [Component Architecture](#component-architecture)
6. [Mode Infrastructure](#mode-infrastructure)
7. [Input Handling](#input-handling)
8. [Service Integration](#service-integration)
9. [Command Pattern](#command-pattern)
10. [Query Abstraction](#query-abstraction)
11. [Rendering Patterns](#rendering-patterns)
12. [Context Switching](#context-switching)
13. [Best Practices](#best-practices)
14. [Cross-Reference to Main Documentation](#cross-reference-to-main-documentation)

---

## Architecture Overview

### System Structure

```
gui/
├── core/                    # Core mode infrastructure
│   ├── uimode.go           # UIMode interface
│   ├── modemanager.go      # Mode lifecycle & transitions
│   ├── contextstate.go     # Context-specific UI state
│   └── gamemodecoordinator.go  # Context switching coordination
├── basemode.go             # Common mode infrastructure
├── commandhistory.go       # Undo/redo system
├── modehelpers.go          # Shared helper functions
├── widgets/                # Widget creation & configuration
│   ├── createwidgets.go    # Declarative widget builders
│   ├── layout.go           # Layout calculations
│   ├── panelconfig.go      # Functional options for panels
│   ├── panels.go           # Panel builder abstraction
│   ├── panel_factory.go    # Standard panel specifications
│   ├── button_factory.go   # Button group helpers
│   ├── list_helpers.go     # List widget helpers
│   └── dialogs.go          # Modal dialog builders
├── guicomponents/          # Reusable UI components
│   ├── guicomponents.go    # Component implementations
│   └── guiqueries.go       # ECS query abstraction
├── guimodes/               # Game mode implementations
│   ├── explorationmode.go
│   ├── inventorymode.go
│   ├── infomode.go
│   └── guirenderers.go     # Custom rendering utilities
├── guicombat/              # Combat-specific modes
│   ├── combatmode.go
│   ├── combat_ui_factory.go
│   ├── combat_input_handler.go
│   └── combat_action_handler.go
└── guisquads/              # Squad management modes
    ├── squadmanagementmode.go
    ├── formationeditormode.go
    ├── squadbuilder.go
    └── squad_builder_ui_factory.go
```

### Key Design Principles

1. **Separation of Concerns**: UI state (mode classes) vs. game state (ECS components)
2. **Composition Over Inheritance**: BaseMode provides shared infrastructure via embedding
3. **Declarative Configuration**: Widget builders use config structs, not imperative code
4. **Query Abstraction**: GUIQueries hides ECS implementation details from modes
5. **Functional Options**: Panel building uses functional options pattern for flexibility
6. **Component Reuse**: Encapsulated UI components handle common patterns (lists, detail panels)

---

## Widget Creation Patterns

### Declarative Widget Configuration

All widgets follow a **config struct + builder function** pattern for consistency and maintainability.

**File:** `gui/widgets/createwidgets.go`

#### Text Widgets

```go
// Simple labels
titleLabel := widgets.CreateLargeLabel("Squad Management")
statusLabel := widgets.CreateSmallLabel("Ready")

// Custom text configuration
customText := widgets.CreateTextWithConfig(widgets.TextConfig{
    Text:       "Custom Message",
    FontFace:   guiresources.LargeFace,
    Color:      color.White,
    LayoutData: widget.AnchorLayoutData{
        HorizontalPosition: widget.AnchorLayoutPositionCenter,
    },
})
```

#### Button Widgets

```go
// Simple button
saveBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text: "Save Squad",
    OnClick: func() {
        // Handle click
    },
})

// Custom button with sizing and layout
customBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text:      "Attack (A)",
    MinWidth:  150,
    MinHeight: 60,
    FontFace:  guiresources.SmallFace,
    OnClick:   handleAttack,
    LayoutData: widget.AnchorLayoutData{
        HorizontalPosition: widget.AnchorLayoutPositionCenter,
    },
})
```

#### List Widgets

```go
// Basic list
itemList := widgets.CreateListWithConfig(widgets.ListConfig{
    Entries: []interface{}{"Item 1", "Item 2", "Item 3"},
    EntryLabelFunc: func(e interface{}) string {
        return e.(string)
    },
    OnEntrySelected: func(e interface{}) {
        fmt.Println("Selected:", e)
    },
    MinWidth:  200,
    MinHeight: 300,
})

// Squad list (specialized helper)
squadList := widgets.CreateSquadList(widgets.SquadListConfig{
    SquadIDs:      allSquadIDs,
    Manager:       ecsManager,
    OnSelect:      handleSquadSelection,
    ScreenWidth:   screenWidth,
    ScreenHeight:  screenHeight,
    WidthPercent:  0.3,
    HeightPercent: 0.5,
})
```

**File:** `gui/widgets/list_helpers.go`

Additional specialized list helpers:
- `CreateSquadList` - Squad selection with automatic name formatting
- `CreateUnitList` - Unit display with health bars
- `CreateSimpleStringList` - Basic string selection
- `CreateInventoryList` - Flexible inventory display

#### Text Input Widgets

```go
nameInput := widgets.CreateTextInputWithConfig(widgets.TextInputConfig{
    MinWidth:    300,
    MinHeight:   50,
    Placeholder: "Enter squad name...",
    OnChanged: func(text string) {
        fmt.Println("Text changed:", text)
    },
})
```

#### Text Area Widgets

```go
detailArea := widgets.CreateTextAreaWithConfig(widgets.TextAreaConfig{
    MinWidth:  400,
    MinHeight: 300,
    FontColor: color.White,
})

detailArea.SetText("Squad details here...")
```

#### Panel Widgets

```go
// Basic panel
infoPanel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
    Title:      "Squad Info",
    MinWidth:   300,
    MinHeight:  200,
    Background: guiresources.PanelRes.Image,
    Padding:    widget.Insets{Left: 15, Right: 15, Top: 15, Bottom: 15},
    Layout:     widget.NewRowLayout(
        widget.RowLayoutOpts.Direction(widget.DirectionVertical),
        widget.RowLayoutOpts.Spacing(10),
    ),
})
```

### Modal Dialogs

**File:** `gui/widgets/dialogs.go`

```go
// Confirmation dialog
confirmDialog := widgets.CreateConfirmationDialog(widgets.DialogConfig{
    Title:   "Disband Squad",
    Message: "Are you sure you want to disband this squad?",
    OnConfirm: func() {
        // Handle confirmation
    },
    OnCancel: func() {
        // Handle cancellation
    },
})
ui.AddWindow(confirmDialog)

// Text input dialog
inputDialog := widgets.CreateTextInputDialog(widgets.TextInputDialogConfig{
    Title:       "Rename Squad",
    Message:     "Enter new squad name:",
    Placeholder: "Squad name",
    InitialText: "Alpha Squad",
    OnConfirm: func(text string) {
        // Handle input
    },
})
ui.AddWindow(inputDialog)

// Message dialog
messageDialog := widgets.CreateMessageDialog(widgets.MessageDialogConfig{
    Title:   "Success",
    Message: "Squad created successfully!",
    OnClose: func() {
        // Dialog closed
    },
})
ui.AddWindow(messageDialog)
```

---

## Layout Patterns

### Layout Types

TinkerRogue uses three primary layout strategies from ebitenui:

**File:** `gui/widgets/layout.go`, `gui/widgets/panelconfig.go`

#### 1. Anchor Layout (Absolute Positioning)

Used for panels positioned at screen edges or center.

```go
// Create container with anchor layout
container := widget.NewContainer(
    widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
)

// Position widgets using AnchorLayoutData
topRightPanel := panelBuilders.BuildPanel(
    widgets.TopRight(),              // Position
    widgets.Size(0.15, 0.2),        // 15% width, 20% height
    widgets.Padding(0.01),          // 1% padding
    widgets.RowLayout(),            // Interior layout
)
```

**Positioning Options:**
- `TopLeft()`, `TopCenter()`, `TopRight()`
- `LeftCenter()`, `Center()`, `RightCenter()`
- `BottomLeft()`, `BottomCenter()`, `BottomRight()`

#### 2. Row Layout (Stacking)

Used for vertically or horizontally stacked widgets.

```go
// Vertical stacking
buttonContainer := widget.NewContainer(
    widget.ContainerOpts.Layout(widget.NewRowLayout(
        widget.RowLayoutOpts.Direction(widget.DirectionVertical),
        widget.RowLayoutOpts.Spacing(10),
        widget.RowLayoutOpts.Padding(widget.Insets{
            Left: 15, Right: 15, Top: 15, Bottom: 15,
        }),
    )),
)

// Horizontal stacking
panel := panelBuilders.BuildPanel(
    widgets.BottomCenter(),
    widgets.HorizontalRowLayout(),  // Convenience wrapper
    widgets.Size(0.5, 0.08),
)
```

#### 3. Grid Layout (3x3 Grid)

Used for formation editors and spatial layouts.

**File:** `gui/widgets/panels.go`

```go
gridContainer, gridCells := panelBuilders.BuildGridEditor(widgets.GridEditorConfig{
    CellTextFormat: func(row, col int) string {
        return fmt.Sprintf("[%d,%d]", row, col)
    },
    OnCellClick: func(row, col int) {
        fmt.Printf("Clicked cell (%d, %d)\n", row, col)
    },
    Padding: widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10},
})

// Access individual cells
gridCells[1][1].Text = "Leader"
```

### Responsive Sizing

All dimensions use **percentage-based values** relative to screen size for responsiveness.

**File:** `gui/widgets/layout_constants.go`

```go
// Standard panel widths
const (
    PanelWidthNarrow     = 0.15  // 15% screen width
    PanelWidthStandard   = 0.2   // 20% screen width
    PanelWidthMedium     = 0.3   // 30% screen width
    PanelWidthWide       = 0.4   // 40% screen width
    PanelWidthExtraWide  = 0.45  // 45% screen width
)

// Standard panel heights
const (
    PanelHeightTiny      = 0.08  // 8% screen height
    PanelHeightSmall     = 0.12  // 12% screen height
    PanelHeightQuarter   = 0.25  // 25% screen height
    PanelHeightHalf      = 0.5   // 50% screen height
    PanelHeightTall      = 0.75  // 75% screen height
)

// Standard padding
const (
    PaddingExtraSmall    = 0.0125  // ~10px at 800px screen
    PaddingTight         = 0.015   // ~12px at 800px screen
    PaddingStandard      = 0.02    // ~16px at 800px screen
    PaddingLoose         = 0.03    // ~24px at 800px screen
)
```

**Usage:**

```go
// Calculate pixel dimensions
width := int(float64(screenWidth) * widgets.PanelWidthStandard)   // 20% of screen
height := int(float64(screenHeight) * widgets.PanelHeightSmall)   // 12% of screen
padding := int(float64(screenWidth) * widgets.PaddingTight)       // ~12px
```

---

## Panel Building System

The panel building system provides **three levels of abstraction** for creating UI panels:

### Level 1: Standard Panels (Highest Abstraction)

Pre-configured panel specifications for common use cases.

**File:** `gui/widgets/panel_factory.go`

```go
// Use pre-defined panel specification
turnOrderPanel := widgets.CreateStandardPanel(panelBuilders, "turn_order")
squadListPanel := widgets.CreateStandardPanel(panelBuilders, "squad_list")
statsPanel := widgets.CreateStandardPanel(panelBuilders, "stats_panel")

// Standard panel with additional options
customPanel := widgets.CreateStandardPanelWithOptions(
    panelBuilders,
    "faction_info",
    widgets.CustomPadding(widget.Insets{Bottom: 50}),  // Override padding
)
```

**Available Standard Panels:**

```go
var StandardPanels = map[string]PanelSpec{
    // Combat UI
    "turn_order":      {Position: TopCenter(), Width: 0.4, Height: 0.08, ...},
    "faction_info":    {Position: TopLeft(), Width: 0.15, Height: 0.12, ...},
    "squad_list":      {Position: LeftCenter(), Width: 0.15, Height: 0.5, ...},
    "squad_detail":    {Position: LeftBottom(), Width: 0.15, Height: 0.25, ...},
    "action_buttons":  {Position: BottomCenter(), Width: 0.5, Height: 0.08, ...},
    "combat_log":      {Position: BottomRight(), Width: 0.24, Height: 0.2, ...},

    // Exploration UI
    "stats_panel":     {Position: TopRight(), Width: 0.15, Height: 0.12, ...},
    "message_log":     {Position: BottomRight(), Width: 0.15, Height: 0.15, ...},
    "quick_inventory": {Position: BottomCenter(), Width: 0.5, Height: 0.08, ...},

    // Info UI
    "options_list":    {Position: Center(), Width: 0.25, Height: 0.5, ...},
    "info_detail":     {Position: RightCenter(), Width: 0.4, Height: 0.6, ...},
    "inventory_detail":{Position: RightCenter(), Width: 0.45, Height: 0.75, ...},
}
```

**Adding Custom Specifications:**

```go
widgets.AddPanelSpec("custom_panel", widgets.PanelSpec{
    Name:     "Custom Panel",
    Position: widgets.TopCenter(),
    Width:    widgets.PanelWidthWide,
    Height:   widgets.PanelHeightSmall,
    Padding:  widgets.PaddingTight,
    Layout:   widgets.RowLayout(),
})
```

### Level 2: BuildPanel with Functional Options

Composable panel configuration using functional options.

**File:** `gui/widgets/panelconfig.go`

```go
// Basic panel with position and size
panel := panelBuilders.BuildPanel(
    widgets.TopRight(),
    widgets.Size(0.15, 0.2),
    widgets.Padding(0.01),
    widgets.RowLayout(),
)

// Complex panel with custom options
detailPanel := panelBuilders.BuildPanel(
    widgets.Center(),
    widgets.Size(0.5, 0.6),
    widgets.CustomPadding(widget.Insets{Left: 20, Right: 20, Top: 10, Bottom: 10}),
    widgets.HorizontalRowLayout(),
    widgets.WithBackground(customBackground),
)
```

**Available Functional Options:**

```go
// Position
TopLeft(), TopCenter(), TopRight()
LeftCenter(), Center(), RightCenter()
BottomLeft(), BottomCenter(), BottomRight()

// Size
Size(widthPercent, heightPercent float64)

// Padding
Padding(percent float64)                    // Uniform padding
CustomPadding(insets widget.Insets)         // Custom padding

// Layout
RowLayout()                                 // Vertical stacking
HorizontalRowLayout()                       // Horizontal stacking
AnchorLayout()                             // Absolute positioning

// Content
WithTitle(title string)                     // Title text
WithBackground(background *e_image.NineSlice) // Custom background
```

### Level 3: Direct CreatePanelWithConfig

Low-level control for complex custom panels.

**File:** `gui/widgets/createwidgets.go`

```go
customPanel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
    Title:      "Custom Panel",
    MinWidth:   500,
    MinHeight:  300,
    Background: guiresources.PanelRes.Image,
    Padding:    widget.Insets{Left: 20, Right: 20, Top: 15, Bottom: 15},
    Layout: widget.NewGridLayout(
        widget.GridLayoutOpts.Columns(2),
        widget.GridLayoutOpts.Spacing(10, 10),
    ),
    LayoutData: widget.AnchorLayoutData{
        HorizontalPosition: widget.AnchorLayoutPositionCenter,
        VerticalPosition:   widget.AnchorLayoutPositionCenter,
    },
})
```

### Helper Functions for Common Patterns

**File:** `gui/modehelpers.go`

```go
// Standard detail panel (panel + textarea)
detailPanel, textArea := gui.CreateStandardDetailPanel(
    panelBuilders,
    layout,
    "info_detail",     // Spec name
    "Initial text",
)

// Bottom-center button container
buttonContainer := gui.CreateBottomCenterButtonContainer(panelBuilders)

// Action button group
buttonContainer := gui.CreateActionButtonGroup(
    panelBuilders,
    widgets.BottomCenter(),
    []widgets.ButtonSpec{
        {Text: "Save", OnClick: handleSave},
        {Text: "Cancel", OnClick: handleCancel},
    },
)

// Filter button container
filterContainer := gui.CreateFilterButtonContainer(panelBuilders, widgets.TopLeft())
```

---

## Component Architecture

Reusable UI components encapsulate common update patterns and reduce code duplication.

**File:** `gui/guicomponents/guicomponents.go`

### Component Pattern

Components follow this structure:
1. **Struct** holding widget references and dependencies
2. **Constructor** (New*Component)
3. **Refresh/Update** method(s)
4. **Optional formatter** functions for customization

### SquadListComponent

Manages squad list containers with filtering and selection.

```go
// Create component
squadList := guicomponents.NewSquadListComponent(
    squadListContainer,    // Container to populate
    queries,              // GUIQueries instance
    func(info *guicomponents.SquadInfo) bool {
        // Filter function - show only alive squads
        return !info.IsDestroyed
    },
    func(squadID ecs.EntityID) {
        // Selection callback
        handleSquadSelected(squadID)
    },
)

// Refresh the list (call when squads change)
squadList.Refresh()
```

**Implementation Details:**
- Automatically removes old buttons and creates new ones
- Applies filter to determine which squads to show
- Handles "AI Turn" message when no squads match filter
- Captures squad IDs in closures for button callbacks

**Usage Examples:**
- `gui/guicombat/combatmode.go:206-213` - Combat mode squad selection
- Squad management modes for filtering player/enemy squads

### DetailPanelComponent

Manages text widgets displaying entity details.

```go
// Create component with custom formatter
detailPanel := guicomponents.NewDetailPanelComponent(
    squadDetailText,  // widget.Text widget
    queries,
    func(data interface{}) string {
        info := data.(*guicomponents.SquadInfo)
        return fmt.Sprintf("Squad: %s\nHP: %d/%d", info.Name, info.CurrentHP, info.MaxHP)
    },
)

// Display squad details
detailPanel.ShowSquad(squadID)

// Display faction details
detailPanel.ShowFaction(factionID)

// Display arbitrary text
detailPanel.SetText("No selection")
```

**Default Formatters:**
- `DefaultSquadFormatter` - Name, units, HP, movement, status
- `DefaultFactionFormatter` - Name, squads, mana

**Usage Examples:**
- `gui/guicombat/combatmode.go:183-201` - Combat faction info panel
- `gui/guicombat/combatmode.go:196-201` - Combat squad detail panel

### TextDisplayComponent

Manages text widgets with periodic updates via formatter function.

```go
// Create component
turnOrderDisplay := guicomponents.NewTextDisplayComponent(
    turnOrderLabel,  // widget.Text widget
    func() string {
        round := combatService.GetCurrentRound()
        faction := combatService.GetCurrentFaction()
        return fmt.Sprintf("Round %d | Faction %d", round, faction)
    },
)

// Update display (typically called in Update())
turnOrderDisplay.Refresh()
```

**Usage Examples:**
- `gui/guicombat/combatmode.go:157-181` - Turn order display in combat

### ItemListComponent

Manages inventory list widgets with filtering.

```go
// Create component
itemList := guicomponents.NewItemListComponent(
    listWidget,       // widget.List widget
    queries,
    ecsManager,
    playerEntityID,
)

// Change filter and refresh
itemList.SetFilter("Throwables")  // Shows only throwable items
itemList.SetFilter("All")         // Shows all items

// Manual refresh
itemList.Refresh()
```

**Supported Filters:**
- "All" - All inventory items
- "Throwables" - Only throwable items
- Extensible for custom filters

**Usage Examples:**
- `gui/guimodes/inventorymode.go` - Inventory filtering

### StatsDisplayComponent

Manages text areas displaying player statistics.

```go
// Create component with custom formatter
statsDisplay := guicomponents.NewStatsDisplayComponent(
    statsTextArea,  // widget.TextArea widget
    func(playerData *common.PlayerData, ecsManager *common.EntityManager) string {
        return fmt.Sprintf("HP: %d\nMana: %d", playerData.HP, playerData.Mana)
    },
)

// Refresh stats (uses default formatter if none provided)
statsDisplay.RefreshStats(playerData, ecsManager)

// Set arbitrary text
statsDisplay.SetText("Stats unavailable")
```

**Usage Examples:**
- `gui/guimodes/explorationmode.go:56-59` - Player stats panel

---

## Mode Infrastructure

### UIMode Interface

All modes must implement this interface.

**File:** `gui/core/uimode.go`

```go
type UIMode interface {
    // Lifecycle
    Initialize(ctx *UIContext) error
    Enter(fromMode UIMode) error
    Exit(toMode UIMode) error

    // Per-frame updates
    Update(deltaTime float64) error
    Render(screen *ebiten.Image)

    // Input handling
    HandleInput(inputState *InputState) bool

    // Accessors
    GetEbitenUI() *ebitenui.UI
    GetModeName() string
}
```

### BaseMode Embedding

All modes embed `gui.BaseMode` to inherit common infrastructure.

**File:** `gui/basemode.go`

```go
type ExplorationMode struct {
    gui.BaseMode  // Embed common infrastructure

    // Mode-specific fields
    statsPanel *widget.Container
    messageLog *widget.TextArea
}

func NewExplorationMode(modeManager *core.UIModeManager) *ExplorationMode {
    mode := &ExplorationMode{}
    mode.SetModeName("exploration")
    mode.ModeManager = modeManager  // Required
    return mode
}
```

### BaseMode Fields (Available to All Modes)

```go
type BaseMode struct {
    // UI Infrastructure
    ui            *ebitenui.UI          // Root UI (call GetEbitenUI())
    RootContainer *widget.Container     // Root container
    Layout        *LayoutConfig         // Responsive layout calculations
    PanelBuilders *PanelBuilders        // Panel building helper

    // Services
    Context       *core.UIContext       // Game context (ECS, player data, screen info)
    Queries       *guicomponents.GUIQueries  // ECS query service
    ModeManager   *core.UIModeManager   // Mode transitions

    // Optional Features
    StatusLabel   *widget.Text          // Status message display
    CommandHistory *CommandHistory      // Undo/redo support

    // Internal
    hotkeys       map[ebiten.Key]InputBinding  // Registered hotkeys
    modeName      string
    returnMode    string
}
```

### Mode Initialization Pattern

**File:** `gui/guimodes/explorationmode.go:37-74`

```go
func (em *ExplorationMode) Initialize(ctx *core.UIContext) error {
    // 1. Initialize common infrastructure (REQUIRED)
    em.InitializeBase(ctx)

    // 2. Register hotkeys for mode transitions
    em.RegisterHotkey(ebiten.KeyI, "inventory")
    em.RegisterHotkey(ebiten.KeyC, "combat")

    // 3. Optional: Initialize command history
    em.InitializeCommandHistory(func() {
        // Refresh callback after undo/redo
        em.refreshUI()
    })

    // 4. Build UI layout using panel builders
    em.statsPanel = em.PanelBuilders.BuildPanel(
        widgets.TopRight(),
        widgets.Size(0.15, 0.12),
        widgets.Padding(0.015),
        widgets.RowLayout(),
    )
    em.RootContainer.AddChild(em.statsPanel)

    // 5. Create UI update components
    em.statsComponent = guicomponents.NewStatsDisplayComponent(
        em.statsTextArea,
        nil,  // Use default formatter
    )

    return nil
}
```

### Mode Lifecycle Hooks

```go
// Enter - Called when switching TO this mode
func (em *ExplorationMode) Enter(fromMode core.UIMode) error {
    fmt.Println("Entering Exploration Mode")

    // Refresh UI components
    em.statsComponent.RefreshStats(em.Context.PlayerData, em.Queries.ECSManager)

    return nil
}

// Exit - Called when switching FROM this mode
func (em *ExplorationMode) Exit(toMode core.UIMode) error {
    fmt.Println("Exiting Exploration Mode")

    // Clean up resources (if needed)
    // Save state (if needed)

    return nil
}

// Update - Called every frame while active
func (em *ExplorationMode) Update(deltaTime float64) error {
    // Periodic updates (e.g., refresh displays)
    // Most modes have minimal Update logic - UI updates happen in Enter()
    return nil
}

// Render - Called to draw custom rendering (overlays, highlights)
func (em *ExplorationMode) Render(screen *ebiten.Image) {
    // Custom rendering (optional)
    // ebitenui automatically renders widgets - only add overlays here
}
```

### Input Handling Pattern

**File:** `gui/guimodes/explorationmode.go:174-202`

```go
func (em *ExplorationMode) HandleInput(inputState *core.InputState) bool {
    // 1. Handle common input first (ESC key, registered hotkeys)
    if em.HandleCommonInput(inputState) {
        return true  // Input consumed
    }

    // 2. Handle mode-specific input
    if inputState.MouseButton == ebiten.MouseButton2 && inputState.MousePressed {
        // Right-click to inspect
        clickedPos := em.convertMouseToLogical(inputState.MouseX, inputState.MouseY)

        if infoMode, exists := em.ModeManager.GetMode("info_inspect"); exists {
            em.ModeManager.RequestTransition(infoMode, "Right-click inspection")
            return true
        }
    }

    return false  // Input not consumed
}
```

### Mode Transitions

```go
// Within same context (BattleMap or Overworld)
if targetMode, exists := em.ModeManager.GetMode("inventory"); exists {
    em.ModeManager.RequestTransition(targetMode, "I key pressed")
}

// Between contexts (BattleMap <-> Overworld)
if em.Context.ModeCoordinator != nil {
    // BattleMap -> Overworld
    em.Context.ModeCoordinator.ReturnToOverworld("squad_management")

    // Overworld -> BattleMap
    em.Context.ModeCoordinator.EnterBattleMap("exploration")
}
```

---

## Input Handling

### InputState Structure

**File:** `gui/core/uimode.go:54-63`

```go
type InputState struct {
    // Mouse
    MouseX            int
    MouseY            int
    MousePressed      bool
    MouseReleased     bool
    MouseButton       ebiten.MouseButton

    // Keyboard
    KeysPressed       map[ebiten.Key]bool   // Currently held
    KeysJustPressed   map[ebiten.Key]bool   // Pressed this frame

    // Legacy bridge
    PlayerInputStates *common.PlayerInputStates
}
```

### Hotkey Registration

**File:** `gui/basemode.go:74-95`

```go
// In Initialize()
mode.RegisterHotkey(ebiten.KeyI, "inventory")
mode.RegisterHotkey(ebiten.KeyC, "combat")
mode.RegisterHotkey(ebiten.KeyE, "squad_management")

// Hotkeys are automatically handled by HandleCommonInput()
```

### Common Input Handler

**File:** `gui/basemode.go:119-144`

```go
func (bm *BaseMode) HandleCommonInput(inputState *core.InputState) bool {
    // Check registered hotkeys
    for key, binding := range bm.hotkeys {
        if inputState.KeysJustPressed[key] {
            if targetMode, exists := bm.ModeManager.GetMode(binding.TargetMode); exists {
                bm.ModeManager.RequestTransition(targetMode, binding.Reason)
                return true
            }
        }
    }

    // ESC key - return to designated return mode
    if inputState.KeysJustPressed[ebiten.KeyEscape] {
        if returnMode, exists := bm.ModeManager.GetMode(bm.returnMode); exists {
            bm.ModeManager.RequestTransition(returnMode, "ESC pressed")
            return true
        }
    }

    return false
}
```

### Mouse Input Patterns

```go
// Left-click
if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
    // Handle left-click
}

// Right-click
if inputState.MouseButton == ebiten.MouseButtonRight && inputState.MousePressed {
    // Handle right-click
}

// Convert mouse to logical position
playerPos := *context.PlayerData.Pos
manager := coords.NewCoordinateManager(graphics.ScreenInfo)
viewport := coords.NewViewport(manager, playerPos)
clickedPos := viewport.ScreenToLogical(inputState.MouseX, inputState.MouseY)
```

### Keyboard Input Patterns

```go
// Single key press
if inputState.KeysJustPressed[ebiten.KeySpace] {
    // Handle spacebar press
}

// Modifier keys
if inputState.KeysJustPressed[ebiten.KeyZ] && inputState.KeysPressed[ebiten.KeyControl] {
    // Ctrl+Z for undo
}

if inputState.KeysJustPressed[ebiten.KeyY] && inputState.KeysPressed[ebiten.KeyControl] {
    // Ctrl+Y for redo
}
```

### Combat Input Handler Example

**File:** `gui/guicombat/combat_input_handler.go`

```go
type CombatInputHandler struct {
    actionHandler *CombatActionHandler
    battleState   *core.BattleMapState
    queries       *guicomponents.GUIQueries
    playerPos     *coords.LogicalPosition
    currentFactionID ecs.EntityID
}

func (cih *CombatInputHandler) HandleInput(inputState *core.InputState) bool {
    // Handle attack mode clicks
    if cih.battleState.InAttackMode && inputState.MousePressed {
        clickedPos := cih.convertMouseToLogical(inputState.MouseX, inputState.MouseY)

        // Check if clicked on enemy squad
        enemySquadID := cih.findSquadAtPosition(clickedPos)
        if enemySquadID != 0 {
            cih.actionHandler.ExecuteAttack(enemySquadID)
            return true
        }
    }

    // Handle movement mode clicks
    if cih.battleState.InMoveMode && inputState.MousePressed {
        clickedPos := cih.convertMouseToLogical(inputState.MouseX, inputState.MouseY)

        // Check if clicked on valid move tile
        if cih.isValidMoveTile(clickedPos) {
            cih.actionHandler.ExecuteMove(clickedPos)
            return true
        }
    }

    return false
}
```

---

## Service Integration

Modes interact with game logic through **service layers** that encapsulate ECS operations.

### GUIQueries (Query Service)

Centralized ECS query functions for UI modes.

**File:** `gui/guicomponents/guiqueries.go`

```go
type GUIQueries struct {
    ECSManager     *common.EntityManager
    factionManager *combat.FactionManager
}

// Create in BaseMode.InitializeBase()
queries := guicomponents.NewGUIQueries(ctx.ECSManager)
```

#### Faction Queries

```go
// Get complete faction info
factionInfo := queries.GetFactionInfo(factionID)
// Returns: FactionInfo{ID, Name, IsPlayerControlled, CurrentMana, MaxMana, SquadIDs, AliveSquadCount}

// Get all factions
allFactions := queries.GetAllFactions()  // []ecs.EntityID
```

#### Squad Queries

```go
// Get complete squad info
squadInfo := queries.GetSquadInfo(squadID)
// Returns: SquadInfo{ID, Name, UnitIDs, AliveUnits, TotalUnits, CurrentHP, MaxHP,
//                    Position, FactionID, IsDestroyed, HasActed, HasMoved, MovementRemaining}

// Get enemy squads
enemySquads := queries.GetEnemySquads(currentFactionID)  // []ecs.EntityID

// Filter squads
aliveFilter := queries.FilterSquadsAlive()
aliveSquads := queries.ApplyFilterToSquads(allSquadIDs, aliveFilter)
```

#### Creature Queries

```go
// Get creature at position
creatureInfo := queries.GetCreatureAtPosition(logicalPos)
// Returns: CreatureInfo{ID, Name, CurrentHP, MaxHP, Strength, Dexterity, Magic,
//                       Leadership, Armor, Weapon, IsMonster, IsPlayer}

if creatureInfo != nil {
    fmt.Printf("%s: %d/%d HP\n", creatureInfo.Name, creatureInfo.CurrentHP, creatureInfo.MaxHP)
}
```

#### Tile Queries

```go
// Get tile info
tileInfo := queries.GetTileInfo(logicalPos)
// Returns: TileInfo{Position, TileType, MovementCost, IsWalkable, HasEntity, EntityID}
```

### CombatService

Encapsulates combat turn management, movement, and faction operations.

**File:** Combat service integration in `gui/guicombat/combatmode.go:68`

```go
type CombatService struct {
    turnManager     *combat.TurnManager
    factionManager  *combat.FactionManager
    movementSystem  *combatservices.MovementSystem
    ecsManager      *common.EntityManager
}

// Create in combat mode
combatService := combatservices.NewCombatService(ctx.ECSManager)
```

#### Combat Operations

```go
// Initialize combat
factionIDs := queries.GetAllFactions()
err := combatService.InitializeCombat(factionIDs)

// Get current state
currentFaction := combatService.GetCurrentFaction()  // ecs.EntityID
currentRound := combatService.GetCurrentRound()      // int

// End turn
result := combatService.EndTurn()
if result.Success {
    newFaction := result.NewFaction
    newRound := result.NewRound
}

// Calculate movement
validTiles := combatService.CalculateValidMovement(squadID, currentPos, movementPoints)

// Execute move
moveResult := combatService.MoveSquad(squadID, targetPos)
```

### Movement Service Example

```go
// Calculate reachable tiles
movementSystem := combatservices.NewMovementSystem(ecsManager)
validTiles := movementSystem.GetValidMovementTiles(squadPos, movementRange)

// Check if tile is reachable
canMove := movementSystem.IsValidMove(squadPos, targetPos, movementRange)
```

---

## Command Pattern

Undo/redo support for UI actions.

**File:** `gui/commandhistory.go`, `squads/squadcommands/`

### CommandHistory Setup

```go
// In mode's Initialize()
mode.InitializeCommandHistory(func() {
    // Refresh callback - called after successful undo/redo
    mode.refreshCurrentSquad()
})
```

### Executing Commands

```go
// Create command
cmd := squadcommands.NewRenameSquadCommand(squadID, newName, ecsManager)

// Execute through CommandHistory
success := mode.CommandHistory.Execute(cmd)
// Automatically shows status message: "✓ Renamed squad to Alpha" or "✗ Error message"
```

### Undo/Redo

```go
// Manual undo/redo
mode.CommandHistory.Undo()  // Shows: "⟲ Undone: Rename squad"
mode.CommandHistory.Redo()  // Shows: "⟳ Redone: Rename squad"

// Keyboard shortcuts (automatically handled if CommandHistory initialized)
// Ctrl+Z for Undo
// Ctrl+Y for Redo

// Create undo/redo buttons
undoBtn := mode.CommandHistory.CreateUndoButton()   // "Undo (Ctrl+Z)"
redoBtn := mode.CommandHistory.CreateRedoButton()   // "Redo (Ctrl+Y)"
```

### Creating Custom Commands

```go
// Implement SquadCommand interface
type CustomCommand struct {
    // Command data
}

func (c *CustomCommand) Execute() error {
    // Perform action
    return nil
}

func (c *CustomCommand) Undo() error {
    // Reverse action
    return nil
}

func (c *CustomCommand) Description() string {
    return "Custom action performed"
}

// Usage
cmd := &CustomCommand{}
mode.CommandHistory.Execute(cmd)
```

---

## Query Abstraction

GUIQueries abstracts ECS implementation details from UI code.

**File:** `gui/guicomponents/guiqueries.go`

### Data Transfer Objects

```go
// FactionInfo - UI representation of faction
type FactionInfo struct {
    ID                 ecs.EntityID
    Name               string
    IsPlayerControlled bool
    CurrentMana        int
    MaxMana            int
    SquadIDs           []ecs.EntityID
    AliveSquadCount    int
}

// SquadInfo - UI representation of squad
type SquadInfo struct {
    ID                ecs.EntityID
    Name              string
    UnitIDs           []ecs.EntityID
    AliveUnits        int
    TotalUnits        int
    CurrentHP         int
    MaxHP             int
    Position          *coords.LogicalPosition
    FactionID         ecs.EntityID
    IsDestroyed       bool
    HasActed          bool
    HasMoved          bool
    MovementRemaining int
}

// CreatureInfo - UI representation of creature
type CreatureInfo struct {
    ID            ecs.EntityID
    Name          string
    CurrentHP     int
    MaxHP         int
    Strength      int
    Dexterity     int
    Magic         int
    Leadership    int
    Armor         int
    Weapon        int
    IsMonster     bool
    IsPlayer      bool
}
```

### Filter Functions

```go
// SquadFilter determines which squads to show
type SquadFilter func(squadInfo *SquadInfo) bool

// Built-in filter
aliveFilter := queries.FilterSquadsAlive()

// Custom filter
playerSquadsFilter := func(info *SquadInfo) bool {
    return !info.IsDestroyed && info.FactionID == playerFactionID
}

// Apply filter
filteredSquads := queries.ApplyFilterToSquads(allSquadIDs, playerSquadsFilter)
```

### Query Usage in Components

```go
// SquadListComponent uses filters
squadList := guicomponents.NewSquadListComponent(
    container,
    queries,
    func(info *guicomponents.SquadInfo) bool {
        // Only show alive squads from current faction
        return !info.IsDestroyed && info.FactionID == currentFactionID
    },
    handleSquadSelection,
)

// DetailPanelComponent uses DTO formatters
detailPanel := guicomponents.NewDetailPanelComponent(
    textWidget,
    queries,
    func(data interface{}) string {
        info := data.(*guicomponents.SquadInfo)
        return fmt.Sprintf("%s\nHP: %d/%d\nUnits: %d/%d",
            info.Name, info.CurrentHP, info.MaxHP, info.AliveUnits, info.TotalUnits)
    },
)
detailPanel.ShowSquad(squadID)  // Queries ECS internally
```

---

## Rendering Patterns

Custom rendering for overlays, highlights, and visual feedback.

**File:** `gui/guimodes/guirenderers.go`

### ViewportRenderer

Base renderer for viewport-centered drawing.

```go
type ViewportRenderer struct {
    screenData coords.ScreenData
    viewport   *coords.Viewport
}

// Create renderer
playerPos := *context.PlayerData.Pos
renderer := guimodes.NewViewportRenderer(screen, playerPos)

// Draw tile overlay
renderer.DrawTileOverlay(screen, tilePos, color.RGBA{R: 0, G: 255, B: 0, A: 80})

// Draw tile border
renderer.DrawTileBorder(screen, tilePos, color.White, 3)  // 3px border
```

### MovementTileRenderer

Renders valid movement tiles.

```go
// Create renderer
movementRenderer := guimodes.NewMovementTileRenderer()

// Render valid movement tiles
validTiles := []coords.LogicalPosition{
    {X: 5, Y: 3},
    {X: 6, Y: 3},
    {X: 5, Y: 4},
}
playerPos := *context.PlayerData.Pos
movementRenderer.Render(screen, playerPos, validTiles)
// Draws semi-transparent green overlay on each tile
```

### SquadHighlightRenderer

Renders squad position highlights.

```go
// Create renderer
highlightRenderer := guimodes.NewSquadHighlightRenderer(queries)

// Render all squad highlights
currentFactionID := combatService.GetCurrentFaction()
selectedSquadID := battleState.SelectedSquadID
playerPos := *context.PlayerData.Pos

highlightRenderer.Render(screen, playerPos, currentFactionID, selectedSquadID)
// Draws:
// - White border on selected squad
// - Blue border on friendly squads
// - Red border on enemy squads
```

### Custom Rendering in Modes

**File:** `gui/guicombat/combatmode.go:331-347`

```go
func (cm *CombatMode) Render(screen *ebiten.Image) {
    playerPos := *cm.Context.PlayerData.Pos
    currentFactionID := cm.combatService.GetCurrentFaction()
    battleState := cm.Context.ModeCoordinator.GetBattleMapState()

    // Render squad highlights (always shown)
    cm.highlightRenderer.Render(screen, playerPos, currentFactionID, battleState.SelectedSquadID)

    // Render valid movement tiles (only in move mode)
    if battleState.InMoveMode {
        if len(battleState.ValidMoveTiles) > 0 {
            cm.movementRenderer.Render(screen, playerPos, battleState.ValidMoveTiles)
        }
    }

    // Note: ebitenui automatically renders widgets - only add overlays here
}
```

### Creating Custom Renderers

```go
type CustomTileRenderer struct {
    fillColor color.Color
}

func NewCustomTileRenderer() *CustomTileRenderer {
    return &CustomTileRenderer{
        fillColor: color.RGBA{R: 255, G: 255, B: 0, A: 100},  // Yellow
    }
}

func (ctr *CustomTileRenderer) Render(screen *ebiten.Image, centerPos coords.LogicalPosition, tiles []coords.LogicalPosition) {
    vr := guimodes.NewViewportRenderer(screen, centerPos)

    for _, pos := range tiles {
        vr.DrawTileOverlay(screen, pos, ctr.fillColor)
    }
}
```

---

## Context Switching

TinkerRogue separates UI into two contexts: **Overworld** (strategic) and **BattleMap** (tactical).

**File:** `gui/core/gamemodecoordinator.go`

### Context Structure

```go
type GameContext int

const (
    ContextOverworld GameContext = iota  // Squad management, world map
    ContextBattleMap                     // Dungeon exploration, combat
)
```

### GameModeCoordinator

Manages two independent UIModeManagers - one per context.

```go
type GameModeCoordinator struct {
    overworldManager *UIModeManager   // Manages overworld modes
    battleMapManager *UIModeManager   // Manages battle map modes
    activeManager    *UIModeManager   // Currently active manager
    currentContext   GameContext
    overworldState   *OverworldState  // Persistent overworld UI state
    battleMapState   *BattleMapState  // Persistent battle map UI state
}
```

### Registering Modes

```go
coordinator := core.NewGameModeCoordinator(ctx)

// Register overworld modes
coordinator.RegisterOverworldMode(NewSquadManagementMode(coordinator.GetOverworldManager()))
coordinator.RegisterOverworldMode(NewFormationEditorMode(coordinator.GetOverworldManager()))
coordinator.RegisterOverworldMode(NewSquadBuilderMode(coordinator.GetOverworldManager()))

// Register battle map modes
coordinator.RegisterBattleMapMode(NewExplorationMode(coordinator.GetBattleMapManager()))
coordinator.RegisterBattleMapMode(NewCombatMode(coordinator.GetBattleMapManager()))
coordinator.RegisterBattleMapMode(NewInventoryMode(coordinator.GetBattleMapManager()))
```

### Context Switching

```go
// From BattleMap to Overworld
if context.ModeCoordinator != nil {
    context.ModeCoordinator.ReturnToOverworld("squad_management")
}

// From Overworld to BattleMap
if context.ModeCoordinator != nil {
    context.ModeCoordinator.EnterBattleMap("exploration")
}

// Toggle context (Ctrl+Tab)
context.ModeCoordinator.ToggleContext()  // Automatically handled in Update()
```

### Context State Management

**File:** `gui/core/contextstate.go`

```go
// OverworldState - persistent between context switches
type OverworldState struct {
    SelectedSquadID    ecs.EntityID
    SquadListScroll    int
    SquadIDs           []ecs.EntityID
    EditingSquadID     ecs.EntityID
    FormationDirty     bool
    // ... more state
}

// BattleMapState - transient UI selection/mode state
type BattleMapState struct {
    SelectedSquadID  ecs.EntityID
    SelectedTargetID ecs.EntityID
    InAttackMode     bool
    InMoveMode       bool
    ValidMoveTiles   []coords.LogicalPosition
}

// Access state in modes
battleState := context.ModeCoordinator.GetBattleMapState()
battleState.SelectedSquadID = squadID
battleState.InMoveMode = true
```

### State Persistence

```go
// Save state when leaving context
func (gmc *GameModeCoordinator) saveOverworldState() {
    // Save current selections, UI state, etc.
}

// Restore state when returning to context
func (gmc *GameModeCoordinator) restoreOverworldState() {
    // Restore selections, scroll positions, etc.
}
```

---

## Best Practices

### UI Development Guidelines

#### 1. Widget Creation
- **Always use declarative config structs** instead of imperative widget construction
- **Prefer standard panel specifications** over custom panel building
- **Use percentage-based sizing** from `layout_constants.go` for responsiveness
- **Leverage helper functions** (`CreateStandardDetailPanel`, `CreateActionButtonGroup`, etc.)

```go
// ✅ GOOD - Declarative config
button := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text: "Save",
    OnClick: handleSave,
})

// ❌ BAD - Imperative construction
button := widget.NewButton(
    widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(100, 100)),
    widget.ButtonOpts.Image(guiresources.ButtonImage),
    widget.ButtonOpts.Text("Save", guiresources.LargeFace, &widget.ButtonTextColor{Idle: color.White}),
    widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
        handleSave()
    }),
)
```

#### 2. Mode Structure
- **Embed BaseMode** for common infrastructure
- **Initialize with InitializeBase()** before building UI
- **Register hotkeys** instead of duplicating input handling
- **Use components** for common UI update patterns

```go
// ✅ GOOD - Clean mode structure
type CustomMode struct {
    gui.BaseMode

    // UI update components
    statsDisplay *guicomponents.StatsDisplayComponent
    squadList    *guicomponents.SquadListComponent
}

func (cm *CustomMode) Initialize(ctx *core.UIContext) error {
    cm.InitializeBase(ctx)  // Initialize common infrastructure
    cm.RegisterHotkey(ebiten.KeyE, "squad_management")

    // Build UI using panel builders
    statsPanel := cm.PanelBuilders.BuildPanel(widgets.TopRight(), ...)
    cm.RootContainer.AddChild(statsPanel)

    // Create components
    cm.statsDisplay = guicomponents.NewStatsDisplayComponent(...)

    return nil
}
```

#### 3. ECS Queries
- **Always use GUIQueries** instead of direct ECS access
- **Use DTOs** (FactionInfo, SquadInfo, etc.) for UI data
- **Apply filters** instead of manual filtering loops
- **Cache query results** only when necessary

```go
// ✅ GOOD - Query abstraction
squadInfo := queries.GetSquadInfo(squadID)
if squadInfo != nil {
    fmt.Printf("%s: %d HP\n", squadInfo.Name, squadInfo.CurrentHP)
}

// ❌ BAD - Direct ECS access
squadData := common.GetComponentType[*squads.SquadData](entity, squads.SquadComponent)
units := squads.GetUnitIDsInSquad(squadID, ecsManager)
totalHP := 0
for _, unitID := range units {
    attrs := common.GetAttributesByID(ecsManager, unitID)
    totalHP += attrs.CurrentHealth
}
```

#### 4. Input Handling
- **Call HandleCommonInput() first** to process ESC/hotkeys
- **Return true** when input is consumed to prevent propagation
- **Use KeysJustPressed** for single-press actions (not KeysPressed)
- **Check mode flags** before processing input

```go
func (m *CustomMode) HandleInput(inputState *core.InputState) bool {
    // Handle common input first
    if m.HandleCommonInput(inputState) {
        return true
    }

    // Mode-specific input
    if inputState.KeysJustPressed[ebiten.KeySpace] {  // ✅ JustPressed
        m.handleAction()
        return true
    }

    return false
}
```

#### 5. Component Usage
- **Create components in Initialize()**, refresh in Enter()/Update()
- **Use default formatters** unless custom display needed
- **Prefer components** over manual UI updates

```go
// ✅ GOOD - Component handles updates
cm.squadList = guicomponents.NewSquadListComponent(
    container, queries, filterFunc, selectionFunc,
)
cm.squadList.Refresh()  // Call when data changes

// ❌ BAD - Manual UI updates
for _, child := range container.Children() {
    container.RemoveChild(child)
}
for _, squadID := range allSquads {
    button := widgets.CreateButtonWithConfig(...)
    container.AddChild(button)
}
```

#### 6. Rendering
- **Only add overlays in Render()** - ebitenui handles widget rendering
- **Use viewport renderers** for tile-based drawing
- **Create reusable renderer objects** (MovementTileRenderer, SquadHighlightRenderer)
- **Avoid creating new images every frame** - cache where possible

```go
func (cm *CombatMode) Render(screen *ebiten.Image) {
    // ✅ GOOD - Reusable renderer objects
    cm.highlightRenderer.Render(screen, playerPos, currentFactionID, selectedSquadID)

    if battleState.InMoveMode {
        cm.movementRenderer.Render(screen, playerPos, battleState.ValidMoveTiles)
    }

    // Note: No need to render ebitenui widgets - handled automatically
}
```

#### 7. Service Integration
- **Use service layers** (CombatService, MovementSystem) for game logic
- **Don't modify ECS directly from UI** - use commands or services
- **Validate actions** in services, not UI code

```go
// ✅ GOOD - Service handles logic
result := combatService.EndTurn()
if result.Success {
    cm.logManager.UpdateTextArea(logArea, result.Message)
}

// ❌ BAD - UI modifies ECS directly
actionState := combat.FindActionStateBySquadID(squadID, ecsManager)
actionState.HasActed = true
combat.UpdateActionState(squadID, actionState, ecsManager)
```

### Common Pitfalls

#### Panel Overlap Issues
**Problem:** Panels overlap when positioned too close.

**Solution:** Use panel specifications from `panel_factory.go` which account for spacing.

```go
// ✅ GOOD - Standard panels avoid overlap
logPanel := widgets.CreateStandardPanel(panelBuilders, "combat_log")
actionPanel := widgets.CreateStandardPanel(panelBuilders, "action_buttons")

// ❌ BAD - Manual positioning risks overlap
logPanel := panelBuilders.BuildPanel(
    widgets.BottomRight(),
    widgets.Size(0.45, 0.2),  // May overlap with 50% bottom-center buttons
)
```

#### Stale UI Data
**Problem:** UI shows outdated information.

**Solution:** Call component Refresh() in Enter() and after state changes.

```go
func (cm *CombatMode) Enter(fromMode core.UIMode) error {
    // Refresh all components
    cm.turnOrderComponent.Refresh()
    cm.factionInfoComponent.ShowFaction(currentFactionID)
    cm.squadListComponent.Refresh()
    return nil
}
```

#### Memory Leaks from Event Handlers
**Problem:** Widget event handlers hold references to large objects.

**Solution:** Capture only necessary data in closures.

```go
// ✅ GOOD - Capture only ID
localSquadID := squadID
button := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    OnClick: func() {
        handleSquadClick(localSquadID)
    },
})

// ❌ BAD - Captures entire squad data
button := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    OnClick: func() {
        handleSquadClick(squadData)  // Holds reference to large struct
    },
})
```

### Testing Patterns

```go
// Test mode initialization
func TestExplorationModeInitialize(t *testing.T) {
    ctx := &core.UIContext{
        ECSManager: common.NewEntityManager(),
        PlayerData: &common.PlayerData{},
        ScreenWidth: 1024,
        ScreenHeight: 768,
    }

    modeManager := core.NewUIModeManager(ctx)
    mode := NewExplorationMode(modeManager)

    err := mode.Initialize(ctx)
    assert.NoError(t, err)
    assert.NotNil(t, mode.RootContainer)
    assert.NotNil(t, mode.Queries)
}

// Test component refresh
func TestSquadListComponentRefresh(t *testing.T) {
    manager := common.NewEntityManager()
    queries := guicomponents.NewGUIQueries(manager)
    container := widget.NewContainer()

    component := guicomponents.NewSquadListComponent(
        container, queries,
        func(info *guicomponents.SquadInfo) bool { return true },
        func(squadID ecs.EntityID) {},
    )

    component.Refresh()
    assert.Equal(t, 0, len(container.Children()))  // No squads yet
}
```

---

## Implementation Checklist

When creating a new GUI mode, use this checklist:

### Mode Setup
- [ ] Create mode struct embedding `gui.BaseMode`
- [ ] Implement constructor with `SetModeName()` and `ModeManager` assignment
- [ ] Implement `Initialize()` calling `InitializeBase(ctx)`
- [ ] Register hotkeys using `RegisterHotkey()`
- [ ] Set return mode using `SetReturnMode()` (if applicable)

### UI Layout
- [ ] Use `PanelBuilders.BuildPanel()` or `CreateStandardPanel()` for panels
- [ ] Use percentage-based sizing from `layout_constants.go`
- [ ] Add panels to `RootContainer`
- [ ] Create UI components (SquadListComponent, DetailPanelComponent, etc.)

### Lifecycle Hooks
- [ ] Implement `Enter()` to refresh UI components
- [ ] Implement `Exit()` to clean up resources (if needed)
- [ ] Implement `Update()` for periodic updates (if needed)
- [ ] Implement `Render()` for custom overlays (if needed)

### Input Handling
- [ ] Implement `HandleInput()` calling `HandleCommonInput()` first
- [ ] Handle mode-specific input with `KeysJustPressed`
- [ ] Return `true` when input is consumed

### Service Integration
- [ ] Use `Queries` for ECS data access (never direct ECS access)
- [ ] Use service layers for game logic (CombatService, MovementSystem, etc.)
- [ ] Use CommandHistory for undoable actions (if applicable)

### Testing
- [ ] Test mode initialization
- [ ] Test input handling
- [ ] Test component refresh
- [ ] Test mode transitions

---

## File Reference Index

### Core Infrastructure
- `gui/core/uimode.go` - UIMode interface
- `gui/core/modemanager.go` - Mode lifecycle management
- `gui/core/contextstate.go` - Context-specific UI state
- `gui/core/gamemodecoordinator.go` - Context switching
- `gui/basemode.go` - Common mode infrastructure

### Widgets & Layout
- `gui/widgets/createwidgets.go` - Declarative widget builders
- `gui/widgets/layout.go` - Layout calculations
- `gui/widgets/layout_constants.go` - Responsive sizing constants
- `gui/widgets/panelconfig.go` - Functional options for panels
- `gui/widgets/panels.go` - Panel builder abstraction
- `gui/widgets/panel_factory.go` - Standard panel specifications
- `gui/widgets/button_factory.go` - Button group helpers
- `gui/widgets/list_helpers.go` - List widget helpers
- `gui/widgets/dialogs.go` - Modal dialog builders

### Components & Services
- `gui/guicomponents/guicomponents.go` - Reusable UI components
- `gui/guicomponents/guiqueries.go` - ECS query abstraction
- `gui/commandhistory.go` - Undo/redo system
- `gui/modehelpers.go` - Shared helper functions

### Rendering
- `gui/guimodes/guirenderers.go` - Custom rendering utilities

### Example Implementations
- `gui/guimodes/explorationmode.go` - Basic mode with panels and components
- `gui/guicombat/combatmode.go` - Complex mode with services and rendering
- `gui/guisquads/squadmanagementmode.go` - Command pattern usage
- `gui/guisquads/formationeditormode.go` - Grid layout example

---

## Glossary

**BaseMode** - Common mode infrastructure that all modes embed for shared functionality.

**Component** - Reusable UI update logic (SquadListComponent, DetailPanelComponent, etc.).

**Context** - Game layer (Overworld or BattleMap) with separate mode managers.

**DTO** - Data Transfer Object (FactionInfo, SquadInfo) used by GUIQueries.

**Functional Options** - Pattern using option functions to configure panel building.

**GUIQueries** - Service layer abstracting ECS queries from UI code.

**Mode** - Distinct UI state (exploration, combat, inventory, etc.) implementing UIMode interface.

**ModeManager** - Manages mode lifecycle and transitions within a context.

**PanelBuilders** - Helper for creating panels with declarative configuration.

**Standard Panels** - Pre-configured panel specifications in `panel_factory.go`.

**UIContext** - Shared game state (ECS manager, player data, screen info) passed to modes.

**ViewportRenderer** - Base renderer for viewport-centered tile drawing.

---

## Cross-Reference to Main Documentation

This document complements the main **DOCUMENTATION.md** with deep-dive implementation patterns. Use this reference table to navigate between documents:

### Architecture & Design
| Topic | GUI_PATTERNS.md | DOCUMENTATION.md |
|-------|-----------------|------------------|
| **GUI Overview** | [Architecture Overview](#architecture-overview) | [Section 9: GUI Architecture](../DOCUMENTATION.md#gui-architecture) |
| **Mode Manager** | [Mode Infrastructure](#mode-infrastructure) | [Mode Manager Pattern](../DOCUMENTATION.md#mode-manager-pattern) |
| **Context Switching** | [Context Switching](#context-switching) | [GUI Architecture - Context](../DOCUMENTATION.md#mode-based-gui) |
| **BaseMode Pattern** | [BaseMode Embedding](#basemode-embedding) | [Mode Manager Pattern](../DOCUMENTATION.md#mode-manager-pattern) |

### Widgets & Layout
| Topic | GUI_PATTERNS.md | DOCUMENTATION.md |
|-------|-----------------|------------------|
| **Widget Creation** | [Widget Creation Patterns](#widget-creation-patterns) | [GUI Architecture - Modes](../DOCUMENTATION.md#available-modes) |
| **Layout System** | [Layout Patterns](#layout-patterns) | [Layout Patterns](../DOCUMENTATION.md#layout-patterns) |
| **Panel Building** | [Panel Building System](#panel-building-system) | [GUI Architecture - Context](../DOCUMENTATION.md#gui-context) |
| **Standard Panels** | [Level 1: Standard Panels](#level-1-standard-panels) | Not covered in main docs |

### Components & Services
| Topic | GUI_PATTERNS.md | DOCUMENTATION.md |
|-------|-----------------|------------------|
| **UI Components** | [Component Architecture](#component-architecture) | [GUI Architecture - Widgets](../DOCUMENTATION.md#widget-system) |
| **GUIQueries** | [Query Abstraction](#query-abstraction) | [Service Integration](#service-integration) |
| **Command Pattern** | [Command Pattern](#command-pattern) | Section 13: [Data Flow Patterns](../DOCUMENTATION.md#data-flow-patterns) |
| **Service Integration** | [Service Integration](#service-integration) | Section 8: [Combat System](../DOCUMENTATION.md#combat-service-layer) |

### Input & Rendering
| Topic | GUI_PATTERNS.md | DOCUMENTATION.md |
|-------|-----------------|------------------|
| **Input Handling** | [Input Handling](#input-handling) | Section 12: [Input System](../DOCUMENTATION.md#input-system) |
| **Rendering Patterns** | [Rendering Patterns](#rendering-patterns) | [Rendering Patterns](../DOCUMENTATION.md#rendering-patterns) |
| **Input Priority** | Not covered | Section 12: [Input Priority Chain](../DOCUMENTATION.md#input-priority-chain) |

---

## Document Purpose & Scope

**GUI_PATTERNS.md** provides:
- **Deep implementation details** for GUI patterns (not covered in main docs)
- **Code examples** showing how to use each pattern
- **File structure** and organizational conventions
- **Best practices** and common pitfalls
- **Checklist** for creating new GUI modes

**DOCUMENTATION.md** provides:
- **High-level architecture** overview
- **Component catalog** (complete list of all components)
- **Data flow** through the GUI system
- **Cross-system integration** (GUI + ECS + Combat)
- **Development patterns** for the entire project

### When to Use Each Document

**Start here (GUI_PATTERNS.md) if you're:**
- Implementing a new GUI mode
- Creating custom widgets
- Working on layout and positioning
- Debugging GUI input handling
- Learning GUI code patterns

**Start here (DOCUMENTATION.md) if you're:**
- New to TinkerRogue architecture
- Understanding the full system
- Integrating GUI with combat/inventory
- Looking for component specifications
- Understanding data flow patterns

---

**End of Documentation**
