# GUI Architecture and Patterns Reference

**Version:** 1.0
**Last Updated:** 2025-11-21
**Purpose:** Reference guide for implementing and modifying UI components in the TinkerRogue GUI system

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Core Concepts](#core-concepts)
3. [Mode System](#mode-system)
4. [Widget Patterns](#widget-patterns)
5. [Layout System](#layout-system)
6. [Component System](#component-system)
7. [Query System](#query-system)
8. [Standard Workflows](#standard-workflows)
9. [Best Practices](#best-practices)
10. [Common Pitfalls](#common-pitfalls)

---

## Architecture Overview

### Package Structure

```
gui/
├── core/                    # Core mode management system
│   ├── uimode.go           # UIMode interface
│   └── modemanager.go      # Mode lifecycle and transitions
├── widgets/                 # Reusable widget factories
│   ├── createwidgets.go    # Widget creation with config pattern
│   ├── panelconfig.go      # Functional options for panels
│   ├── panel_factory.go    # Standard panel specifications
│   ├── layout.go           # Layout calculations
│   └── layout_constants.go # Size/padding constants
├── guicomponents/          # High-level reusable components
│   ├── guicomponents.go    # Component types (SquadList, DetailPanel, etc.)
│   └── guiqueries.go       # Centralized ECS query service
├── guimodes/               # Concrete UI mode implementations
│   ├── explorationmode.go  # Dungeon exploration UI
│   ├── inventorymode.go    # Inventory management UI
│   └── infomode.go         # Entity inspection UI
├── guisquads/              # Squad-related UI modes
│   ├── squadmanagementmode.go  # Squad overview UI
│   ├── formationeditormode.go  # Formation editing UI
│   ├── squadbuilder.go         # Squad creation UI
│   └── squaddeploymentmode.go  # Deployment UI
├── guicombat/              # Combat-specific UI
│   ├── combatmode.go       # Combat UI mode
│   └── combat_ui_factory.go # Combat panel factories
├── guiresources/           # Shared GUI resources (fonts, images, colors)
│   └── guiresources.go
├── basemode.go             # Base mode with common infrastructure
└── modehelpers.go          # Helper functions for common patterns
```

### Design Philosophy

1. **Mode-Based Architecture**: UI is organized into distinct modes (exploration, combat, inventory, etc.)
2. **Composition Over Inheritance**: Modes embed `BaseMode` to inherit common behavior
3. **Declarative Configuration**: Widgets use config structs instead of variadic options
4. **Functional Options**: Panels use functional options for composability
5. **Component-Based Updates**: Reusable components manage data refresh logic
6. **Centralized Queries**: Single query service (`GUIQueries`) for all ECS data access
7. **Separation of Concerns**: UI logic separate from game logic

---

## Core Concepts

### The UIMode Interface

Every UI mode implements the `core.UIMode` interface:

```go
type UIMode interface {
    Initialize(ctx *UIContext) error  // Called once when mode created
    Enter(fromMode UIMode) error      // Called when switching TO this mode
    Exit(toMode UIMode) error         // Called when switching FROM this mode
    Update(deltaTime float64) error   // Called every frame
    Render(screen *ebiten.Image)      // Custom rendering (optional)
    HandleInput(inputState *InputState) bool // Process input
    GetEbitenUI() *ebitenui.UI       // Returns the widget tree
    GetModeName() string              // Returns mode identifier
}
```

### UIContext - Shared Game State

The `UIContext` provides modes with access to game state:

```go
type UIContext struct {
    ECSManager   *common.EntityManager  // ECS world
    PlayerData   *common.PlayerData     // Player state
    ScreenWidth  int                    // Screen dimensions
    ScreenHeight int
    TileSize     int                    // Tile size for positioning
}
```

### BaseMode - Common Infrastructure

All modes should embed `BaseMode` to inherit common functionality:

```go
type ExplorationMode struct {
    gui.BaseMode  // Embed common infrastructure

    // Mode-specific fields
    statsPanel *widget.Container
    messageLog *widget.TextArea
}
```

**BaseMode provides:**
- `Context` - UIContext reference
- `Layout` - LayoutConfig for responsive sizing
- `ModeManager` - Mode transition management
- `RootContainer` - Root ebitenui container
- `PanelBuilders` - Panel factory service
- `Queries` - Unified ECS query service
- `RegisterHotkey()` - Declarative hotkey registration
- `HandleCommonInput()` - Standard input handling (ESC, hotkeys)

---

## Mode System

### Creating a New Mode

**Step 1: Define the Mode Struct**

```go
package guimodes

import (
    "game_main/gui"
    "game_main/gui/core"
    "github.com/ebitenui/ebitenui/widget"
)

type MyNewMode struct {
    gui.BaseMode  // ALWAYS embed BaseMode

    // Mode-specific UI widgets
    mainPanel   *widget.Container
    detailText  *widget.TextArea
    actionButtons *widget.Container
}
```

**Step 2: Constructor with Mode Name**

```go
func NewMyNewMode(modeManager *core.UIModeManager) *MyNewMode {
    mode := &MyNewMode{}
    mode.SetModeName("my_new_mode")      // Set unique identifier
    mode.SetReturnMode("exploration")     // Set mode to return to on ESC
    mode.ModeManager = modeManager        // Required for transitions
    return mode
}
```

**Step 3: Initialize Method**

```go
func (m *MyNewMode) Initialize(ctx *core.UIContext) error {
    // ALWAYS call InitializeBase first
    m.InitializeBase(ctx)

    // Register hotkeys for quick mode transitions
    m.RegisterHotkey(ebiten.KeyI, "inventory")
    m.RegisterHotkey(ebiten.KeyE, "exploration")

    // Build UI panels
    m.buildMainPanel()
    m.buildActionButtons()

    return nil
}
```

**Step 4: Lifecycle Methods**

```go
// Enter is called when switching TO this mode
func (m *MyNewMode) Enter(fromMode core.UIMode) error {
    fmt.Println("Entering MyNewMode")

    // Refresh UI with current data
    m.refreshPanels()

    return nil
}

// Exit is called when switching FROM this mode
func (m *MyNewMode) Exit(toMode core.UIMode) error {
    fmt.Println("Exiting MyNewMode")

    // Clean up temporary data
    m.clearTemporaryState()

    return nil
}
```

**Step 5: Input Handling**

```go
func (m *MyNewMode) HandleInput(inputState *core.InputState) bool {
    // ALWAYS handle common input first (ESC, hotkeys)
    if m.HandleCommonInput(inputState) {
        return true  // Input consumed
    }

    // Mode-specific input handling
    if inputState.KeysJustPressed[ebiten.KeySpace] {
        m.performAction()
        return true
    }

    return false  // Input not consumed
}
```

**Step 6: Register with Mode Manager**

```go
// In your main game initialization
modeManager := core.NewUIModeManager(uiContext)

myMode := guimodes.NewMyNewMode(modeManager)
modeManager.RegisterMode(myMode)

// Set as starting mode if desired
modeManager.SetMode("my_new_mode")
```

### Mode Transitions

**Programmatic Transitions:**

```go
// Request transition (happens at end of frame)
if targetMode, exists := m.ModeManager.GetMode("inventory"); exists {
    m.ModeManager.RequestTransition(targetMode, "User opened inventory")
}

// Immediate transition
m.ModeManager.SetMode("combat")
```

**Declarative Hotkeys:**

```go
// In Initialize()
m.RegisterHotkey(ebiten.KeyI, "inventory")      // I key opens inventory
m.RegisterHotkey(ebiten.KeyC, "combat")         // C key opens combat
m.RegisterHotkey(ebiten.KeyE, "squad_management") // E key opens squads

// HandleCommonInput() automatically processes these
```

---

## Widget Patterns

### Configuration Structs

All widgets use config structs for declarative creation:

#### Button Configuration

```go
button := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text:      "Attack",
    MinWidth:  200,           // Optional, default 100
    MinHeight: 80,            // Optional, default 100
    FontFace:  guiresources.LargeFace,  // Optional, default LargeFace
    Padding:   widget.Insets{Left: 20, Right: 20, Top: 10, Bottom: 10},
    OnClick: func() {
        fmt.Println("Attack button clicked")
    },
})
```

**Config Fields:**
- `Text` (string) - Button label
- `MinWidth`, `MinHeight` (int) - Size constraints
- `FontFace` (font.Face) - Font (default: `guiresources.LargeFace`)
- `TextColor` (*widget.ButtonTextColor) - Text colors for states
- `Image` (*widget.ButtonImage) - Button background images
- `Padding` (widget.Insets) - Text padding
- `OnClick` (func()) - Click handler (simplified, no args)
- `LayoutData` (interface{}) - Positioning data

#### List Configuration

```go
list := widgets.CreateListWithConfig(widgets.ListConfig{
    Entries:    squadEntries,  // []interface{} of list items
    MinWidth:   400,
    MinHeight:  600,
    EntryLabelFunc: func(e interface{}) string {
        if squad, ok := e.(SquadData); ok {
            return fmt.Sprintf("%s - %d units", squad.Name, squad.UnitCount)
        }
        return fmt.Sprintf("%v", e)
    },
    OnEntrySelected: func(selectedEntry interface{}) {
        fmt.Printf("Selected: %v\n", selectedEntry)
    },
})
```

**Config Fields:**
- `Entries` ([]interface{}) - List data
- `EntryLabelFunc` (func(interface{}) string) - Format each entry
- `OnEntrySelected` (func(interface{})) - Selection handler
- `MinWidth`, `MinHeight` (int) - Size constraints
- `LayoutData` (interface{}) - Positioning data

#### TextArea Configuration

```go
textArea := widgets.CreateTextAreaWithConfig(widgets.TextAreaConfig{
    MinWidth:  500,
    MinHeight: 300,
    FontColor: color.White,
})
textArea.SetText("Initial content")
```

**Config Fields:**
- `MinWidth`, `MinHeight` (int) - Size constraints
- `FontColor` (color.Color) - Text color

#### Text Configuration

```go
label := widgets.CreateTextWithConfig(widgets.TextConfig{
    Text:     "Squad Status",
    FontFace: guiresources.LargeFace,
    Color:    color.White,
    LayoutData: widget.AnchorLayoutData{
        HorizontalPosition: widget.AnchorLayoutPositionCenter,
    },
})

// Convenience functions
largeLabel := widgets.CreateLargeLabel("Title")  // Uses LargeFace
smallLabel := widgets.CreateSmallLabel("Detail") // Uses SmallFace
```

#### Panel Configuration

```go
panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
    Title:      "Squad Details",
    MinWidth:   400,
    MinHeight:  300,
    Background: guiresources.PanelRes.Image,
    Layout:     widget.NewRowLayout(...),
    LayoutData: widget.AnchorLayoutData{...},
})
```

**Config Fields:**
- `Title` (string) - Panel title (displayed at top)
- `MinWidth`, `MinHeight` (int) - Size constraints
- `Background` (*e_image.NineSlice) - Panel background image
- `Padding` (widget.Insets) - Internal padding
- `Layout` (widget.Layouter) - Layout strategy
- `LayoutData` (interface{}) - Positioning data

---

## Layout System

### Functional Options for Panels

Panels use functional options for composability:

```go
panel := panelBuilders.BuildPanel(
    widgets.TopRight(),                    // Position
    widgets.Size(0.2, 0.3),               // 20% width, 30% height
    widgets.Padding(0.01),                // 1% padding
    widgets.RowLayout(),                  // Vertical layout
)
```

### Position Options

```go
widgets.TopLeft()       widgets.TopCenter()       widgets.TopRight()
widgets.LeftCenter()    widgets.Center()          widgets.RightCenter()
widgets.BottomLeft()    widgets.BottomCenter()    widgets.BottomRight()
```

### Size Options

```go
widgets.Size(widthPercent, heightPercent)  // Percentage of screen (0.0 - 1.0)
```

### Padding Options

```go
widgets.Padding(percent)  // Uniform padding percentage

widgets.CustomPadding(widget.Insets{
    Left:   20,
    Right:  20,
    Top:    10,
    Bottom: 50,
})
```

### Layout Options

```go
widgets.RowLayout()            // Vertical stacking
widgets.HorizontalRowLayout()  // Horizontal stacking
widgets.AnchorLayout()         // Absolute positioning
```

### Standard Panel Specifications

Use predefined panel specs for consistency:

```go
// Create panel from specification
panel := widgets.CreateStandardPanel(panelBuilders, "turn_order")

// Available standard specs:
// Combat UI:
// - "turn_order"     - Top center, wide, tiny height
// - "faction_info"   - Top left, narrow, small
// - "squad_list"     - Left center, narrow, half height
// - "squad_detail"   - Left bottom, narrow, quarter height
// - "action_buttons" - Bottom center, half width, tiny height

// Exploration UI:
// - "stats_panel"    - Top right, narrow, small
// - "message_log"    - Bottom right, narrow, small
// - "quick_inventory" - Bottom center, half width, tiny height

// Info/Inspection UI:
// - "options_list"   - Center, medium, half height
```

### Layout Constants

All constants are in `widgets/layout_constants.go`:

**Width Percentages:**
```go
PanelWidthNarrow     = 0.15  // Stats display, faction info
PanelWidthStandard   = 0.2   // Squad lists, inventory filters
PanelWidthMedium     = 0.3   // Filters, secondary content
PanelWidthWide       = 0.4   // Top bars, main content
PanelWidthExtraWide  = 0.45  // Detail views, full-width content
```

**Height Percentages:**
```go
PanelHeightTiny      = 0.08  // Top bars, button containers
PanelHeightSmall     = 0.12  // Faction info, header sections
PanelHeightQuarter   = 0.25  // Quarter screen
PanelHeightThird     = 0.33  // Third screen
PanelHeightHalf      = 0.5   // Half screen
PanelHeightExtraTall = 0.6   // Between half and full
PanelHeightTall      = 0.75  // Detail views, list containers
PanelHeightFull      = 0.85  // Nearly full screen
```

**Padding:**
```go
PaddingTight    = 0.01  // Tight padding
PaddingStandard = 0.02  // Standard padding
PaddingLoose    = 0.03  // Loose padding
```

### Responsive Sizing Examples

```go
// Calculate actual pixel dimensions from percentages
width := int(float64(layout.ScreenWidth) * widgets.PanelWidthNarrow)
height := int(float64(layout.ScreenHeight) * PanelHeightSmall)

// Use LayoutConfig for special calculations
x, y, width, height := layout.CenterWindow(0.5, 0.5)  // Centered 50% window
x, y, width, height := layout.GridLayoutArea()         // 2-column grid area
```

---

## Component System

### Reusable Components

Components manage data refresh logic for common UI patterns:

#### SquadListComponent - Dynamic Squad Buttons

```go
// Create component
squadList := guicomponents.NewSquadListComponent(
    container,   // Container to hold buttons
    queries,     // GUIQueries instance
    func(info *guicomponents.SquadInfo) bool {
        // Filter: show only alive player squads
        return !info.IsDestroyed && queries.IsPlayerFaction(info.FactionID)
    },
    func(squadID ecs.EntityID) {
        // Selection handler
        fmt.Printf("Selected squad: %d\n", squadID)
    },
)

// Refresh squad buttons (call when data changes)
squadList.Refresh()
```

#### DetailPanelComponent - Entity Details Display

```go
// Create component with custom formatter
detailPanel := guicomponents.NewDetailPanelComponent(
    textWidget,  // *widget.Text to display details
    queries,     // GUIQueries instance
    func(data interface{}) string {
        // Custom formatter
        info := data.(*guicomponents.SquadInfo)
        return fmt.Sprintf("Squad: %s\nHP: %d/%d",
            info.Name, info.CurrentHP, info.MaxHP)
    },
)

// Show squad details
detailPanel.ShowSquad(squadID)

// Show faction details
detailPanel.ShowFaction(factionID)

// Show custom text
detailPanel.SetText("No selection")
```

#### ItemListComponent - Inventory Lists

```go
// Create component
itemList := guicomponents.NewItemListComponent(
    listWidget,       // *widget.List
    queries,          // GUIQueries instance
    ecsManager,       // EntityManager
    playerEntityID,   // Player entity
)

// Set filter and refresh
itemList.SetFilter("Throwables")  // "All", "Throwables", "Equipment", etc.
itemList.Refresh()                // Explicit refresh
```

#### StatsDisplayComponent - Player Stats

```go
// Create component with default formatter
statsDisplay := guicomponents.NewStatsDisplayComponent(
    textAreaWidget,  // *widget.TextArea
    nil,             // nil = use default formatter (PlayerAttributes)
)

// Refresh stats display
statsDisplay.RefreshStats(playerData, ecsManager)

// Use custom formatter
statsDisplay := guicomponents.NewStatsDisplayComponent(
    textAreaWidget,
    func(pd *common.PlayerData, em *common.EntityManager) string {
        return fmt.Sprintf("HP: %d\nMana: %d",
            pd.CurrentHP, pd.CurrentMana)
    },
)
```

### Component Benefits

1. **Encapsulates refresh logic** - No need to duplicate update code
2. **Consistent formatting** - Default formatters ensure uniformity
3. **Separation of concerns** - Component manages data, mode manages layout
4. **Easy to test** - Component logic testable in isolation

---

## Query System

### GUIQueries - Centralized ECS Queries

**Never query ECS directly in modes.** Use `GUIQueries` service:

```go
// In your mode's Initialize():
queries := em.Queries  // Inherited from BaseMode

// Faction queries
factionInfo := queries.GetFactionInfo(factionID)  // Complete faction data
factionName := queries.GetFactionName(factionID)  // Just name (optimized)
isPlayer := queries.IsPlayerFaction(factionID)    // Player check (optimized)
allFactions := queries.GetAllFactions()           // All faction IDs

// Squad queries
squadInfo := queries.GetSquadInfo(squadID)        // Complete squad data
squadName := queries.GetSquadName(squadID)        // Just name
allSquads := queries.FindAllSquads()              // All squad IDs
squadAtPos := queries.GetSquadAtPosition(pos)     // Squad at position
factionSquads := queries.FindSquadsByFaction(factionID)

// Filtered queries
playerSquads := queries.GetPlayerSquads()         // Player's squads only
aliveSquads := queries.GetAliveSquads()           // Non-destroyed squads
enemySquads := queries.GetEnemySquads(currentFactionID)
```

### Query Data Structures

#### SquadInfo

```go
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
```

#### FactionInfo

```go
type FactionInfo struct {
    ID                 ecs.EntityID
    Name               string
    IsPlayerControlled bool
    CurrentMana        int
    MaxMana            int
    SquadIDs           []ecs.EntityID
    AliveSquadCount    int
}
```

### Custom Filters

```go
// Create custom filter function
alivePlayerFilter := func(info *guicomponents.SquadInfo) bool {
    return !info.IsDestroyed &&
           queries.IsPlayerFaction(info.FactionID) &&
           info.AliveUnits > 0
}

// Apply filter to squad list
allSquads := queries.FindAllSquads()
filtered := queries.ApplyFilterToSquads(allSquads, alivePlayerFilter)

// Use with SquadListComponent
squadList := guicomponents.NewSquadListComponent(
    container, queries, alivePlayerFilter, onSelect,
)
```

---

## Standard Workflows

### Creating a Simple Panel

```go
func (m *MyMode) buildInfoPanel() {
    // Use standard panel spec
    panel := widgets.CreateStandardPanel(m.PanelBuilders, "stats_panel")

    // Add label
    label := widgets.CreateLargeLabel("Squad Info")
    panel.AddChild(label)

    // Add text area
    textArea := widgets.CreateTextAreaWithConfig(widgets.TextAreaConfig{
        MinWidth:  200,
        MinHeight: 150,
        FontColor: color.White,
    })
    textArea.SetText("Details here")
    panel.AddChild(textArea)

    m.RootContainer.AddChild(panel)
}
```

### Creating a Custom Panel

```go
func (m *MyMode) buildCustomPanel() {
    panel := m.PanelBuilders.BuildPanel(
        widgets.BottomRight(),             // Position
        widgets.Size(0.25, 0.4),          // 25% width, 40% height
        widgets.Padding(0.02),            // 2% padding
        widgets.RowLayout(),              // Vertical layout
    )

    // Add content
    label := widgets.CreateSmallLabel("Actions")
    panel.AddChild(label)

    button := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
        Text: "Do Something",
        OnClick: func() {
            m.performAction()
        },
    })
    panel.AddChild(button)

    m.RootContainer.AddChild(panel)
}
```

### Creating Action Buttons

```go
func (m *MyMode) buildActionButtons() {
    // Use helper for standard bottom-center container
    buttonContainer := gui.CreateBottomCenterButtonContainer(m.PanelBuilders)

    // Use helper to add buttons
    gui.AddActionButton(buttonContainer, "Attack", func() {
        m.performAttack()
    })

    gui.AddActionButton(buttonContainer, "Defend", func() {
        m.performDefend()
    })

    gui.AddActionButton(buttonContainer, "Flee", func() {
        m.flee()
    })

    m.RootContainer.AddChild(buttonContainer)
}
```

### Creating a Detail Panel with TextArea

```go
func (m *MyMode) buildDetailPanel() {
    // Use helper for standard pattern
    panel, textArea := gui.CreateDetailPanel(
        m.PanelBuilders,
        m.Layout,
        widgets.RightCenter(),                // Position
        widgets.PanelWidthExtraWide,          // Width (0.45)
        widgets.PanelHeightTall,              // Height (0.75)
        widgets.PaddingStandard,              // Padding (0.02)
        "Select an item to view details",    // Default text
    )

    // Store references
    m.detailPanel = panel
    m.detailText = textArea

    m.RootContainer.AddChild(panel)
}
```

### Creating a 3x3 Grid Editor

```go
func (m *MyMode) buildGridEditor() {
    gridContainer, gridCells := m.PanelBuilders.BuildGridEditor(
        widgets.GridEditorConfig{
            CellTextFormat: func(row, col int) string {
                // Get unit at position
                unitID := m.getUnitAt(row, col)
                if unitID == 0 {
                    return "Empty"
                }
                return m.queries.GetSquadName(unitID)
            },
            OnCellClick: func(row, col int) {
                m.handleCellClick(row, col)
            },
            Padding: widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10},
        },
    )

    // Store grid cell references if needed
    m.gridCells = gridCells

    m.RootContainer.AddChild(gridContainer)
}
```

### Using UI Factories

For complex modes, create a factory:

```go
// In combat mode
factory := NewCombatUIFactory(m.Queries, m.PanelBuilders, m.Layout)

turnOrderPanel := factory.CreateTurnOrderPanel()
m.RootContainer.AddChild(turnOrderPanel)

factionPanel := factory.CreateFactionInfoPanel()
m.RootContainer.AddChild(factionPanel)

logPanel, logText := factory.CreateLogPanel()
m.combatLog = logText
m.RootContainer.AddChild(logPanel)

actionButtons := factory.CreateActionButtons(
    m.onAttack,
    m.onMove,
    m.onEndTurn,
    m.onFlee,
)
m.RootContainer.AddChild(actionButtons)
```

---

## Best Practices

### 1. Always Use BaseMode

```go
// ✅ CORRECT
type MyMode struct {
    gui.BaseMode
    // mode-specific fields
}

// ❌ WRONG - Don't implement UIMode directly
type MyMode struct {
    ecsManager *common.EntityManager
    // ... manual setup
}
```

### 2. Always Call InitializeBase

```go
// ✅ CORRECT
func (m *MyMode) Initialize(ctx *core.UIContext) error {
    m.InitializeBase(ctx)  // FIRST
    // ... build UI
}

// ❌ WRONG - Missing InitializeBase
func (m *MyMode) Initialize(ctx *core.UIContext) error {
    m.Context = ctx  // Manual setup
    // ...
}
```

### 3. Use Config Structs, Not Manual Widget Creation

```go
// ✅ CORRECT - Declarative config
button := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text: "Attack",
    OnClick: func() { m.attack() },
})

// ❌ WRONG - Manual widget construction
button := widget.NewButton(
    widget.ButtonOpts.WidgetOpts(
        widget.WidgetOpts.MinSize(100, 100),
    ),
    widget.ButtonOpts.Image(guiresources.ButtonImage),
    widget.ButtonOpts.Text("Attack", guiresources.LargeFace, &widget.ButtonTextColor{
        Idle: color.White,
    }),
    // ... many more options
)
```

### 4. Use Standard Panel Specs When Available

```go
// ✅ CORRECT - Reuse standard spec
panel := widgets.CreateStandardPanel(m.PanelBuilders, "turn_order")

// ❌ WRONG - Duplicate standard layout
panel := m.PanelBuilders.BuildPanel(
    widgets.TopCenter(),
    widgets.Size(0.4, 0.08),
    widgets.Padding(0.01),
    widgets.HorizontalRowLayout(),
)
```

### 5. Use GUIQueries, Never Direct ECS Queries

```go
// ✅ CORRECT - Use query service
squadName := m.Queries.GetSquadName(squadID)
squadInfo := m.Queries.GetSquadInfo(squadID)

// ❌ WRONG - Direct ECS query
for _, result := range m.Context.ECSManager.World.Query(squads.SquadTag) {
    // ... manual component access
}
```

### 6. Use Components for Dynamic Content

```go
// ✅ CORRECT - Component manages refresh
m.squadList = guicomponents.NewSquadListComponent(
    container, m.Queries, filter, onSelect,
)
// Later:
m.squadList.Refresh()

// ❌ WRONG - Manual button management
for _, squadID := range squads {
    btn := widgets.CreateButtonWithConfig(...)
    container.AddChild(btn)
}
// Later: Need to manually clear and rebuild
```

### 7. Use Helpers for Common Patterns

```go
// ✅ CORRECT - Use helper
closeBtn := gui.CreateCloseButton(m.ModeManager, "exploration", "Close")

// ❌ WRONG - Duplicate close button pattern
closeBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text: "Close",
    OnClick: func() {
        if mode, exists := m.ModeManager.GetMode("exploration"); exists {
            m.ModeManager.RequestTransition(mode, "Close")
        }
    },
})
```

### 8. Register Hotkeys Declaratively

```go
// ✅ CORRECT - Declarative registration
func (m *MyMode) Initialize(ctx *core.UIContext) error {
    m.InitializeBase(ctx)
    m.RegisterHotkey(ebiten.KeyI, "inventory")
    m.RegisterHotkey(ebiten.KeyE, "exploration")
}

// ❌ WRONG - Manual input handling
func (m *MyMode) HandleInput(inputState *core.InputState) bool {
    if inputState.KeysJustPressed[ebiten.KeyI] {
        if mode, exists := m.ModeManager.GetMode("inventory"); exists {
            m.ModeManager.RequestTransition(mode, "I key")
        }
    }
}
```

### 9. Use Layout Constants

```go
// ✅ CORRECT - Use named constants
panel := m.PanelBuilders.BuildPanel(
    widgets.TopRight(),
    widgets.Size(widgets.PanelWidthNarrow, widgets.PanelHeightSmall),
    widgets.Padding(widgets.PaddingTight),
)

// ❌ WRONG - Magic numbers
panel := m.PanelBuilders.BuildPanel(
    widgets.TopRight(),
    widgets.Size(0.15, 0.12),
    widgets.Padding(0.01),
)
```

### 10. Separate UI Building into Methods

```go
// ✅ CORRECT - Organized into methods
func (m *MyMode) Initialize(ctx *core.UIContext) error {
    m.InitializeBase(ctx)
    m.buildStatsPanel()
    m.buildActionButtons()
    m.buildDetailPanel()
    return nil
}

func (m *MyMode) buildStatsPanel() { /* ... */ }
func (m *MyMode) buildActionButtons() { /* ... */ }
func (m *MyMode) buildDetailPanel() { /* ... */ }

// ❌ WRONG - Everything in Initialize
func (m *MyMode) Initialize(ctx *core.UIContext) error {
    m.InitializeBase(ctx)
    // 200 lines of UI building...
}
```

---

## Common Pitfalls

### 1. Forgetting to Call InitializeBase

**Symptom:** `m.Queries` is nil, `m.PanelBuilders` is nil, hotkeys don't work

**Fix:**
```go
func (m *MyMode) Initialize(ctx *core.UIContext) error {
    m.InitializeBase(ctx)  // ADD THIS FIRST
    // ... rest of initialization
}
```

### 2. Not Setting Mode Name

**Symptom:** Mode transitions fail, GetModeName returns empty string

**Fix:**
```go
func NewMyMode(modeManager *core.UIModeManager) *MyMode {
    mode := &MyMode{}
    mode.SetModeName("my_mode")  // ADD THIS
    mode.ModeManager = modeManager
    return mode
}
```

### 3. Capturing Loop Variables in Closures

**Symptom:** All buttons call handler with same ID (last iteration value)

**Fix:**
```go
// ❌ WRONG
for _, squadID := range squadIDs {
    btn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
        OnClick: func() {
            m.selectSquad(squadID)  // BUG: Uses last squadID
        },
    })
}

// ✅ CORRECT
for _, squadID := range squadIDs {
    localID := squadID  // Capture in local variable
    btn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
        OnClick: func() {
            m.selectSquad(localID)  // Correct: Uses captured value
        },
    })
}
```

### 4. Direct ECS Queries Instead of GUIQueries

**Symptom:** Duplicate query logic, inconsistent data formatting

**Fix:**
```go
// ❌ WRONG
for _, result := range m.Context.ECSManager.World.Query(squads.SquadTag) {
    squadData := common.GetComponentType[*squads.SquadData](result.Entity, squads.SquadComponent)
    // ... manual processing
}

// ✅ CORRECT
squadInfo := m.Queries.GetSquadInfo(squadID)
```

### 5. Not Refreshing Components on Enter

**Symptom:** Stale data shown when returning to mode

**Fix:**
```go
func (m *MyMode) Enter(fromMode core.UIMode) error {
    // Refresh all components
    if m.squadList != nil {
        m.squadList.Refresh()
    }
    if m.statsComponent != nil {
        m.statsComponent.RefreshStats(m.Context.PlayerData, m.Context.ECSManager)
    }
    return nil
}
```

### 6. Creating Widgets Before InitializeBase

**Symptom:** Nil pointer errors accessing Layout, PanelBuilders

**Fix:**
```go
// ❌ WRONG
func (m *MyMode) Initialize(ctx *core.UIContext) error {
    m.buildPanels()      // Uses m.PanelBuilders (nil!)
    m.InitializeBase(ctx)
    return nil
}

// ✅ CORRECT
func (m *MyMode) Initialize(ctx *core.UIContext) error {
    m.InitializeBase(ctx)  // Initialize first
    m.buildPanels()        // Then build UI
    return nil
}
```

### 7. Not Cleaning Up in Exit

**Symptom:** Memory leaks, duplicate widgets on re-entry

**Fix:**
```go
func (m *MyMode) Exit(toMode core.UIMode) error {
    // Clean up components
    if m.squadList != nil {
        m.squadList.Clear()
    }

    // Clear temporary state
    m.selectedSquadID = 0
    m.temporaryData = nil

    return nil
}
```

### 8. Using Widget Pointers After Removal

**Symptom:** Widgets don't update, or panic on access

**Fix:**
```go
// Store references to containers, not individual widgets
type MyMode struct {
    gui.BaseMode
    squadListContainer *widget.Container  // Store container
    statsTextArea      *widget.TextArea   // Store mutable widgets
}

// Rebuild children, don't store button pointers
func (m *MyMode) refreshButtons() {
    // Clear old buttons
    children := m.squadListContainer.Children()
    for _, child := range children {
        m.squadListContainer.RemoveChild(child)
    }

    // Add new buttons (don't store pointers to them)
    for _, squad := range squads {
        btn := widgets.CreateButtonWithConfig(...)
        m.squadListContainer.AddChild(btn)
    }
}
```

### 9. Forgetting to Add Widgets to Container

**Symptom:** Widgets created but not visible

**Fix:**
```go
// ✅ CORRECT
panel := widgets.CreateStandardPanel(m.PanelBuilders, "stats_panel")
label := widgets.CreateLargeLabel("Stats")
panel.AddChild(label)  // Add to panel
m.RootContainer.AddChild(panel)  // Add to root

// ❌ WRONG
panel := widgets.CreateStandardPanel(m.PanelBuilders, "stats_panel")
label := widgets.CreateLargeLabel("Stats")
// Forgot to add label to panel!
m.RootContainer.AddChild(panel)
```

### 10. Hardcoding Screen Dimensions

**Symptom:** UI breaks on different resolutions

**Fix:**
```go
// ❌ WRONG
width := 1920 * 0.2  // Hardcoded resolution

// ✅ CORRECT
width := int(float64(m.Layout.ScreenWidth) * widgets.PanelWidthNarrow)
```

---

## Advanced Patterns

### Custom Panel Specifications

Add your own standard panel specs:

```go
// In your mode's Initialize():
widgets.AddPanelSpec("my_custom_panel", widgets.PanelSpec{
    Position: widgets.TopCenter(),
    Width:    0.35,
    Height:   0.15,
    Padding:  widgets.PaddingStandard,
    Layout:   widgets.RowLayout(),
})

// Use it:
panel := widgets.CreateStandardPanel(m.PanelBuilders, "my_custom_panel")
```

### Custom Formatters

```go
// Create specialized formatter
mySquadFormatter := func(data interface{}) string {
    info := data.(*guicomponents.SquadInfo)
    return fmt.Sprintf(
        "[%s]\n"+
        "Units: %d/%d\n"+
        "HP: %d/%d (%d%%)\n"+
        "Morale: %s",
        info.Name,
        info.AliveUnits, info.TotalUnits,
        info.CurrentHP, info.MaxHP,
        (info.CurrentHP * 100) / info.MaxHP,
        getMoraleStatus(info),
    )
}

// Use with component
detailPanel := guicomponents.NewDetailPanelComponent(
    textWidget, queries, mySquadFormatter,
)
```

### Combining Components

```go
// Squad list with linked detail panel
m.squadList = guicomponents.NewSquadListComponent(
    m.squadListContainer,
    m.Queries,
    m.Queries.FilterSquadsByPlayer(),  // Filter
    func(squadID ecs.EntityID) {
        // When squad selected, update detail panel
        m.detailPanel.ShowSquad(squadID)
        m.selectedSquadID = squadID
    },
)

// Refresh both together
func (m *MyMode) refreshUI() {
    m.squadList.Refresh()
    if m.selectedSquadID != 0 {
        m.detailPanel.ShowSquad(m.selectedSquadID)
    }
}
```

### Modal Dialogs

```go
func (m *MyMode) showConfirmDialog(message string, onConfirm func()) {
    // Create centered modal panel
    x, y, width, height := m.Layout.CenterWindow(0.4, 0.3)

    modalPanel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
        MinWidth:  width,
        MinHeight: height,
        Layout: widget.NewRowLayout(
            widget.RowLayoutOpts.Direction(widget.DirectionVertical),
            widget.RowLayoutOpts.Spacing(10),
        ),
        LayoutData: widget.AnchorLayoutData{
            HorizontalPosition: widget.AnchorLayoutPositionCenter,
            VerticalPosition:   widget.AnchorLayoutPositionCenter,
        },
    })

    // Message
    label := widgets.CreateLargeLabel(message)
    modalPanel.AddChild(label)

    // Buttons
    confirmBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
        Text: "Confirm",
        OnClick: func() {
            m.RootContainer.RemoveChild(modalPanel)
            if onConfirm != nil {
                onConfirm()
            }
        },
    })
    modalPanel.AddChild(confirmBtn)

    cancelBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
        Text: "Cancel",
        OnClick: func() {
            m.RootContainer.RemoveChild(modalPanel)
        },
    })
    modalPanel.AddChild(cancelBtn)

    m.RootContainer.AddChild(modalPanel)
}
```

---

## Quick Reference

### Essential Imports

```go
import (
    "game_main/gui"
    "game_main/gui/core"
    "game_main/gui/widgets"
    "game_main/gui/guicomponents"
    "game_main/gui/guiresources"

    "github.com/ebitenui/ebitenui/widget"
    "github.com/hajimehoshi/ebiten/v2"
)
```

### Mode Template

```go
type MyMode struct {
    gui.BaseMode
    // mode fields
}

func NewMyMode(mm *core.UIModeManager) *MyMode {
    mode := &MyMode{}
    mode.SetModeName("my_mode")
    mode.SetReturnMode("exploration")
    mode.ModeManager = mm
    return mode
}

func (m *MyMode) Initialize(ctx *core.UIContext) error {
    m.InitializeBase(ctx)
    m.RegisterHotkey(ebiten.KeyEscape, "exploration")
    m.buildUI()
    return nil
}

func (m *MyMode) Enter(fromMode core.UIMode) error {
    m.refreshUI()
    return nil
}

func (m *MyMode) Exit(toMode core.UIMode) error {
    m.cleanup()
    return nil
}

func (m *MyMode) HandleInput(inputState *core.InputState) bool {
    if m.HandleCommonInput(inputState) {
        return true
    }
    // mode-specific input
    return false
}

func (m *MyMode) buildUI() {
    // UI construction
}

func (m *MyMode) refreshUI() {
    // Data refresh
}

func (m *MyMode) cleanup() {
    // Cleanup
}
```

---

## Resources

- **ebitenui Documentation**: https://github.com/ebitenui/ebitenui
- **Ebiten Documentation**: https://ebiten.org/
- **Source Files:**
  - `gui/core/uimode.go` - UIMode interface
  - `gui/basemode.go` - BaseMode implementation
  - `gui/widgets/` - Widget factory patterns
  - `gui/guicomponents/` - Reusable components
  - `gui/modehelpers.go` - Common helper functions

---

**END OF DOCUMENTATION**
