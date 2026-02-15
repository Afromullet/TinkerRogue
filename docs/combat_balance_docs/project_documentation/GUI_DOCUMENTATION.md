# TinkerRogue GUI Documentation

**Version:** 5.0
**Last Updated:** 2026-02-10

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Core Concepts](#core-concepts)
3. [Package Structure](#package-structure)
4. [Panel Registry System](#panel-registry-system)
5. [GUIQueries](#guiqueries)
6. [Panel Building System](#panel-building-system)
7. [Widget System](#widget-system)
8. [Context State](#context-state)
9. [Mode Catalog](#mode-catalog)
10. [Performance Optimization](#performance-optimization)
11. [Adding New Modes](#adding-new-modes)
12. [Best Practices](#best-practices)
13. [Common Patterns](#common-patterns)
14. [Troubleshooting](#troubleshooting)
15. [Glossary](#glossary)

---

## Architecture Overview

TinkerRogue's GUI system is built on a mode-based architecture where each screen (exploration, combat, inventory, squad management, overworld) is a self-contained mode with its own UI panels, input handlers, and state. The system separates UI state from game state through ECS queries and supports complex multi-context workflows.

### High-Level Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                       GameModeCoordinator                       │
│                     (Two-Context System)                        │
├────────────────────────────┬────────────────────────────────────┤
│    Overworld Context       │      BattleMap Context             │
│  (Strategic Layer)         │   (Tactical Layer)                 │
├────────────────────────────┼────────────────────────────────────┤
│ • Overworld                │ • Exploration                      │
│ • Node Placement           │ • Combat                           │
│ • Squad Management         │ • Combat Animation                 │
│ • Squad Builder            │ • Inventory                        │
│ • Squad Editor             │ • Squad Deployment                 │
│ • Unit Purchase            │                                    │
│ • Squad Deployment         │                                    │
└────────────────────────────┴────────────────────────────────────┘
         │                              │
         └──────────────┬───────────────┘
                        │
         ┌──────────────▼──────────────┐
         │      UIModeManager          │
         │   (Mode Lifecycle)          │
         └──────────────┬──────────────┘
                        │
         ┌──────────────▼──────────────┐
         │         UIMode              │
         │    (Interface)              │
         ├─────────────────────────────┤
         │ • Initialize(*UIContext)    │
         │ • Enter(fromMode)          │
         │ • Exit(toMode)             │
         │ • Update(deltaTime)        │
         │ • Render(*ebiten.Image)    │
         │ • HandleInput(*InputState) │
         │ • GetEbitenUI()            │
         │ • GetModeName()            │
         └─────────────────────────────┘
                        │
         ┌──────────────▼──────────────┐
         │       BaseMode              │
         │  (Common Infrastructure)    │
         ├─────────────────────────────┤
         │ • Panel Registry            │
         │ • Input Bindings            │
         │ • Command History           │
         │ • GUIQueries                │
         │ • PanelBuilders             │
         └─────────────────────────────┘
```

### Core Principles

1. **Pure UI State**: UI modes only store UI-specific state (selection, scroll position, panel visibility). Game state lives in ECS components.

2. **Query-Based Access**: Modes access game state through `GUIQueries`, never directly querying the ECS. This provides caching, validation, and a clean abstraction.

3. **Panel-Based Layout**: Each mode registers panels (UI regions) with declarative specifications. The panel registry handles creation, positioning, and lifecycle.

4. **Context Separation**: Two distinct UI contexts (Overworld and BattleMap) maintain independent state and mode histories, enabling seamless context switching.

5. **Responsive Design**: All layouts use proportional sizing (screen percentages) with centralized constants for consistent spacing across resolutions.

6. **Input Abstraction**: Hotkeys are declaratively bound to actions. InputState provides unified mouse/keyboard access.

7. **Performance First**: Cached rendering, background pooling, and dirty-tracking minimize redundant draws. 90% CPU reduction in list rendering.

---

## Core Concepts

### UIMode Interface

Every screen implements this interface:

```go
type UIMode interface {
    Initialize(ctx *UIContext) error       // One-time setup
    Enter(fromMode UIMode) error           // Called when mode becomes active
    Exit(toMode UIMode) error              // Called when leaving mode
    Update(deltaTime float64) error        // Per-frame updates
    Render(screen *ebiten.Image)           // Custom rendering (overlays)
    HandleInput(inputState *InputState) bool // Input processing
    GetEbitenUI() *ebitenui.UI             // Returns ebitenui root
    GetModeName() string                   // Mode identifier
}
```

**File:** `gui/framework/uimode.go`

**Initialize**: Called once when mode is first registered. Sets up panels, hotkeys, and dependencies.

**Enter**: Called when switching TO this mode. Receives the mode we're coming from (nil on first entry). Use to refresh UI.

**Exit**: Called when switching FROM this mode. Receives the mode we're going to. Use for cleanup.

**Update**: Called every frame while active. Receives delta time in seconds.

**Render**: Custom rendering for overlays (map tiles, highlights). ebitenui handles standard widget rendering automatically.

**HandleInput**: Processes keyboard/mouse input. Returns true if input was consumed (prevents propagation).

**GetEbitenUI**: Returns the ebitenui root UI for the mode.

**GetModeName**: Returns a string identifier for debugging/logging.

### UIContext

Shared context passed to all modes, providing access to game systems:

```go
type UIContext struct {
    ECSManager      *common.EntityManager
    PlayerData      *common.PlayerData
    GameMap         *worldmap.GameMap
    ScreenWidth     int
    ScreenHeight    int
    TileSize        int
    ModeCoordinator *GameModeCoordinator
    Queries         *GUIQueries
}
```

**ECSManager**: Direct ECS access (avoid when possible; prefer Queries).

**PlayerData**: Player-specific data (gold, unlocks, etc.).

**GameMap**: Current map reference for rendering.

**ScreenWidth/ScreenHeight/TileSize**: Rendering parameters.

**ModeCoordinator**: Access to context switching and mode management.

**Queries**: Typed query interface for all game state access.

### InputState

Unified input snapshot passed to Update:

```go
type InputState struct {
    MouseX            int
    MouseY            int
    MousePressed      bool
    MouseButton       ebiten.MouseButton
    KeysPressed       map[ebiten.Key]bool
    KeysJustPressed   map[ebiten.Key]bool
    PlayerInputStates *common.PlayerInputStates
}
```

**MousePressed**: True while mouse button is held.

**MouseButton**: Which button is pressed (Left/Right/Middle).

**KeysPressed**: Keys currently held down.

**KeysJustPressed**: Keys pressed this frame (for single-fire actions).

**PlayerInputStates**: Extended input state for game-specific controls.

### ModeTransition

Return value from Update to signal mode changes:

```go
type ModeTransition struct {
    ToMode UIMode
    Reason string
}
```

**ToMode**: The mode to switch to (nil = stay in current mode).

**Reason**: Debug string describing why the transition occurred.

Example:
```go
if input.KeysJustPressed[ebiten.KeyEscape] {
    return &ModeTransition{
        ToMode: previousMode,
        Reason: "User pressed ESC",
    }
}
return nil // Stay in current mode
```

### BaseMode

Common infrastructure inherited by all modes:

```go
type BaseMode struct {
    ui             *ebitenui.UI
    Context        *UIContext
    Layout         *specs.LayoutConfig
    ModeManager    *UIModeManager
    RootContainer  *widget.Container
    PanelBuilders  *builders.PanelBuilders
    Queries        *GUIQueries
    StatusLabel    *widget.Text
    CommandHistory *CommandHistory
    Panels         *PanelRegistry

    modeName   string
    returnMode string
    hotkeys    map[ebiten.Key]InputBinding
    self       UIMode
}
```

**ui**: The ebitenui root UI.

**Context**: Shared game context.

**Layout**: Responsive layout calculator.

**ModeManager**: Mode lifecycle manager.

**RootContainer**: Root widget container for the UI.

**PanelBuilders**: Fluent API for building panels.

**Queries**: ECS query interface.

**StatusLabel**: Bottom-left status text widget.

**CommandHistory**: Undo/redo system.

**Panels**: Registry of all panels in this mode.

**modeName**: String identifier for this mode.

**returnMode**: Mode to return to on ESC (if applicable).

**hotkeys**: Map of key bindings to actions.

**self**: Reference to the implementing mode (for method dispatch).

### InputBinding

Declarative hotkey specification for mode transitions:

```go
type InputBinding struct {
    Key        ebiten.Key
    TargetMode string
    Reason     string
}
```

Hotkeys are registered via `ModeBuilder` or manually:

```go
// Via ModeBuilder (preferred)
framework.NewModeBuilder(&m.BaseMode, framework.ModeConfig{
    Hotkeys: []framework.HotkeySpec{
        {Key: ebiten.KeyI, TargetMode: "inventory"},
        {Key: ebiten.KeyC, TargetMode: "combat"},
    },
}).Build(ctx)

// Manually
mode.RegisterHotkey(ebiten.KeyE, "squad_management")
```

When a registered hotkey is pressed, `HandleCommonInput()` automatically transitions to the target mode.

---

## Package Structure

The GUI system is organized into specialized packages:

```
gui/
├── framework/              # Core mode infrastructure
│   ├── uimode.go          # UIMode interface, UIContext, InputState, ModeTransition
│   ├── basemode.go        # Common mode infrastructure
│   ├── modemanager.go     # Mode lifecycle & transitions
│   ├── coordinator.go     # Two-context system (Overworld/BattleMap)
│   ├── contextstate.go    # Context-specific UI state (BattleMapState, OverworldState)
│   ├── modebuilder.go     # Declarative mode configuration
│   ├── panelregistry.go   # Global panel type registry + GetPanelWidget[T] generic
│   ├── guiqueries.go      # ECS query abstraction (SquadInfo, FactionInfo, SquadFilter)
│   ├── guiqueries_rendering.go  # Bridges GUIQueries to rendering interfaces
│   ├── commandhistory.go  # Undo/redo system
│   └── squadinfo_cache.go # Event-driven squad info cache
│
├── builders/              # UI construction helpers
│   ├── panels.go          # Panel building with functional options + TypedPanel + GridEditor + CreateStaticPanel
│   ├── layout.go          # Layout calculations, anchor helpers (AnchorStartStart, AnchorCenterStart, etc.)
│   ├── layout_test.go     # Layout tests
│   ├── dialogs.go         # Modal dialog builders (Confirmation, TextInput, Message, SelectionDialog)
│   ├── lists.go           # List widget builders
│   ├── widgets.go         # Widget creation helpers (ButtonGroup, TextInput, Container configs)
│   └── panelspecs.go      # Standard panel specifications (PanelLayoutSpec)
│
├── widgets/               # Widget wrappers & utilities
│   ├── cached_list.go     # Cached list rendering (90% CPU reduction) + SelectedEntry/SetSelectedEntry
│   ├── cached_textarea.go # Cached text area rendering + SetText/AppendText auto-dirty
│   ├── createwidgets.go   # Widget creation helpers
│   └── textdisplay.go     # Auto-updating text components
│
├── specs/                 # Layout specifications
│   └── layout.go          # LayoutConfig + ALL sizing constants (moved from widgets/)
│
├── guiresources/          # Shared UI resources
│   ├── guiresources.go    # Font faces, button graphics
│   └── cachedbackground.go # CachedBackground, CachedBackgroundPool, global cache accessors
│
├── guicombat/             # Combat mode implementation
│   ├── combatmode.go              # Main combat mode
│   ├── combatdeps.go              # Dependency injection
│   ├── combatvisualization.go     # Visual rendering (threat heatmaps, health bars, layers)
│   ├── combat_action_handler.go   # Attack/move execution
│   ├── combat_input_handler.go    # Map click + hotkey handling
│   ├── combat_log_manager.go      # Combat log formatting
│   ├── combat_panels_registry.go  # Combat panel registration
│   ├── combat_turn_flow.go        # Turn management logic
│   ├── combat_animation_mode.go   # Animation playback mode
│   ├── combat_animation_panels_registry.go  # Animation panel registrations
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
│   ├── squadeditor_movesquad.go       # Squad movement logic
│   ├── squad_builder_grid_manager.go  # Grid state management
│   ├── squadmanagement_panels_registry.go
│   ├── squadbuilder_panels_registry.go
│   ├── squadeditor_panels_registry.go
│   ├── squaddeployment_panels_registry.go
│   └── unitpurchase_panels_registry.go
│
├── guiexploration/        # Exploration & inventory modes
│   ├── explorationmode.go         # Default exploration mode
│   ├── inventorymode.go           # Full-screen inventory
│   ├── itemlistcomponent.go       # Inventory list component
│   ├── exploration_panels_registry.go
│   └── inventory_panels_registry.go
│
├── guioverworld/          # Overworld strategic mode
│   ├── overworldmode.go           # Main overworld mode (tick controls, threat engagement, auto-travel)
│   ├── overworld_renderer.go      # Overworld visualization renderer
│   └── overworld_panels_registry.go # Overworld panel registrations
│
└── guinodeplacement/      # Node placement mode
    ├── nodeplacementmode.go           # Player node placement on overworld map
    ├── nodeplacement_renderer.go      # Placement preview rendering
    └── nodeplacement_panels_registry.go # Node placement panel registrations
```

### Key Package Responsibilities

**framework/**: Core mode infrastructure. All modes import this. Contains UIMode interface, BaseMode, ModeManager, Coordinator, GUIQueries, and panel registry.

**builders/**: UI construction utilities. Provides fluent APIs for creating panels, widgets, dialogs, and layouts. All layout constants and panel specifications live here.

**widgets/**: Reusable widget wrappers with performance optimizations. CachedList and CachedTextArea reduce CPU usage by 90% through dirty-tracking.

**specs/**: Layout configuration and responsive sizing constants. Centralized location for all panel dimensions, padding, and positioning values.

**guiresources/**: Shared resources (fonts, button images, cached backgrounds). Global background pool provides pre-rendered NineSlice images for panels and scroll containers.

**guicombat/**: Combat mode implementation. Handles turn-based combat UI, threat visualization, action execution, animation playback, and combat logging.

**guisquads/**: Squad management modes. Five modes for creating, editing, purchasing units, and deploying squads.

**guiexploration/**: Exploration and inventory modes. The default gameplay mode and full-screen item management.

**guioverworld/**: Strategic overworld layer. Manages tick advancement, threat visualization, travel initiation, and encounter engagement.

**guinodeplacement/**: Node placement mode. Allows players to place strategic nodes on the overworld map with validation and preview.

---

## Panel Registry System

The panel registry provides a declarative system for defining, creating, and accessing UI panels. Each panel is identified by a `PanelType` constant and registered with a `PanelDescriptor` that specifies its layout, content, and behavior.

### PanelType

A string-based type for panel identification:

```go
type PanelType string

const (
    // Combat mode panels
    CombatPanelTurnOrder   PanelType = "combat_turn_order"
    CombatPanelSquadList   PanelType = "combat_squad_list"
    CombatPanelSquadDetail PanelType = "combat_squad_detail"
    // ... etc
)
```

### PanelContentType

Specifies how a panel's content is populated:

```go
const (
    ContentEmpty  PanelContentType = iota // Just container
    ContentText                           // Text label
    ContentCustom                         // Custom widget tree via callback
)
```

**ContentEmpty**: An empty container. Use for panels that will be populated later.

**ContentText**: A simple text label. Specify text in OnCreate.

**ContentCustom**: Full custom widget tree. OnCreate callback builds the entire panel.

### PanelDescriptor

Declarative panel specification:

```go
type PanelDescriptor struct {
    SpecName string
    Content  PanelContentType
    Position func(*specs.LayoutConfig) builders.PanelOption
    Width    float64
    Height   float64
    OnCreate func(*PanelResult, UIMode) error
}
```

**SpecName**: Name of a PanelLayoutSpec (from panelspecs.go). If empty, uses Position/Width/Height.

**Content**: Type of content (Empty, Text, Custom).

**Position**: Function returning position option (e.g., `func(l) { return builders.TopLeft() }`). Only used if SpecName is empty.

**Width/Height**: Panel dimensions as screen fractions (0.0-1.0). Only used if SpecName is empty.

**OnCreate**: Callback invoked after panel creation. Receives PanelResult and the mode instance. Use this to:
  - Store references to specific widgets
  - Set initial text/state
  - Add additional widgets
  - Register event handlers

### PanelResult

Returned by `BuildRegisteredPanel`, contains the built panel and widget references:

```go
type PanelResult struct {
    Container *widget.Container          // Root panel container
    Type      PanelType                  // Panel type identifier
    TextLabel *widget.Text               // Text widget (for ContentText)
    Custom    map[string]interface{}      // Custom widget storage
}
```

**Container**: The root panel container.

**Type**: Identifies this panel for later retrieval.

**TextLabel**: Text widget (auto-created for `ContentText`, or manually set in `OnCreate`).

**Custom**: Arbitrary widget storage. Use `result.Custom["textArea"] = textArea` to store any widget for later retrieval.

### Registering Panels

Panels are registered globally in `init()` functions, typically in `*_panels_registry.go` files:

```go
// File: gui/guicombat/combat_panels_registry.go

const (
    CombatPanelTurnOrder   framework.PanelType = "combat_turn_order"
    CombatPanelFactionInfo framework.PanelType = "combat_faction_info"
)

func init() {
    framework.RegisterPanel(CombatPanelTurnOrder, framework.PanelDescriptor{
        Content: framework.ContentText,
        OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
            bm := mode.(*CombatMode)
            layout := bm.Layout

            result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
                MinWidth:  int(float64(layout.ScreenWidth) * specs.CombatTurnOrderWidth),
                MinHeight: int(float64(layout.ScreenHeight) * specs.CombatTurnOrderHeight),
            })

            result.TextLabel = builders.CreateLargeLabel("Initializing combat...")
            result.Container.AddChild(result.TextLabel)
            return nil
        },
    })
}
```

### Custom Positioning Without SpecName

For one-off panels, omit SpecName and use the `OnCreate` callback to handle everything:

```go
framework.RegisterPanel(MyDynamicPanel, framework.PanelDescriptor{
    Content: framework.ContentText,
    OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
        bm := mode.(*MyMode)
        result.Container = bm.PanelBuilders.BuildPanel(
            builders.TopCenter(),
            builders.Size(0.3, 0.15),
            builders.Padding(specs.PaddingTight),
            builders.RowLayout(),
        )
        result.TextLabel = builders.CreateSmallLabel("Dynamic Panel")
        result.Container.AddChild(result.TextLabel)
        return nil
    },
})
```

### Accessing Panels

**Get Panel Result**:

```go
result := mode.Panels.Get(CombatPanelTurnOrder)
if result != nil {
    result.TextLabel.Label = "New turn order text"
}
```

**Get Typed Widget** (Generic):

```go
// Returns *CachedTextAreaWrapper or zero value if not found
textArea := framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](
    mode.Panels,
    CombatPanelCombatLog,
    "textArea",
)
if textArea != nil {
    textArea.SetText("Combat started!")
}
```

**Direct Custom Map Access**:

```go
result := mode.Panels.Get(CombatPanelCombatLog)
if result != nil {
    if ta, ok := result.Custom["textArea"].(*widgets.CachedTextAreaWrapper); ok {
        ta.SetText("Updated log")
    }
}
```

### Complete Example

```go
// Panel type constants (in mymode_panels_registry.go)
const (
    ExamplePanelHeader framework.PanelType = "example_header"
    ExamplePanelLog    framework.PanelType = "example_log"
)

// Registration in init() - uses global registry
func init() {
    framework.RegisterPanel(ExamplePanelHeader, framework.PanelDescriptor{
        Content: framework.ContentText,
        OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
            bm := mode.(*ExampleMode)
            layout := bm.Layout

            panelWidth := int(float64(layout.ScreenWidth) * specs.PanelWidthStandard)
            panelHeight := int(float64(layout.ScreenHeight) * specs.PanelHeightSmall)

            result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
                MinWidth:   panelWidth,
                MinHeight:  panelHeight,
                Background: guiresources.PanelRes.Image,
                Layout: widget.NewRowLayout(
                    widget.RowLayoutOpts.Direction(widget.DirectionVertical),
                ),
            })
            result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(10)

            result.TextLabel = builders.CreateLargeLabel("Example Mode")
            result.Container.AddChild(result.TextLabel)
            return nil
        },
    })

    framework.RegisterPanel(ExamplePanelLog, framework.PanelDescriptor{
        Content: framework.ContentCustom,
        OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
            bm := mode.(*ExampleMode)
            layout := bm.Layout

            panelWidth := int(float64(layout.ScreenWidth) * specs.PanelWidthStandard)
            panelHeight := int(float64(layout.ScreenHeight) * specs.PanelHeightTall)

            result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
                MinWidth:  panelWidth,
                MinHeight: panelHeight,
                Layout:    widget.NewAnchorLayout(),
            })
            result.Container.GetWidget().LayoutData = builders.AnchorEndEnd(10, 10)

            textArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
                MinWidth:  panelWidth - 20,
                MinHeight: panelHeight - 20,
            })
            result.Container.AddChild(textArea)
            result.Custom["textArea"] = textArea

            return nil
        },
    })
}

// Usage in mode
func (m *ExampleMode) updateHeader(text string) {
    result := m.Panels.Get(ExamplePanelHeader)
    if result != nil {
        result.TextLabel.Label = text
    }
}

func (m *ExampleMode) getLogTextArea() *widgets.CachedTextAreaWrapper {
    return framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](
        m.Panels,
        ExamplePanelLog,
        "textArea",
    )
}
```

---

## GUIQueries

GUIQueries provides a typed, cached query interface for accessing game state. Modes should never directly query ECS; instead, use GUIQueries methods.

### GUIQueries Structure

```go
type GUIQueries struct {
    ECSManager     *common.EntityManager
    factionManager *combat.CombatFactionManager
    SquadCache     *squads.SquadQueryCache
    CombatCache    *combat.CombatQueryCache
    squadInfoCache *SquadInfoCache
}
```

**ECSManager**: Direct ECS access (internal use).

**factionManager**: Combat faction system (internal use).

**SquadCache**: ECS-level squad query cache.

**CombatCache**: ECS-level combat query cache.

**squadInfoCache**: UI-specific squad info cache with invalidation.

### SquadInfo

Aggregated squad information for UI display:

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

**ID**: Squad entity ID.

**Name**: Squad display name.

**UnitIDs**: All unit entity IDs (alive and dead).

**AliveUnits**: Count of alive units.

**TotalUnits**: Total unit count.

**CurrentHP/MaxHP**: Aggregated health across all units.

**Position**: Squad's logical position.

**FactionID**: Owning faction entity ID.

**IsDestroyed**: True if squad is eliminated.

**HasActed**: True if squad has acted this turn (combat).

**HasMoved**: True if squad has moved this turn (combat).

**MovementRemaining**: Remaining movement points this turn.

### FactionInfo

Aggregated faction information for UI display:

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

**ID**: Faction entity ID.

**Name**: Faction display name.

**IsPlayerControlled**: True if this is the player faction.

**CurrentMana/MaxMana**: Faction mana resources.

**SquadIDs**: All squad entity IDs in this faction.

**AliveSquadCount**: Count of non-destroyed squads.

### Squad Queries

**GetSquadInfo**:

```go
squadInfo := queries.GetSquadInfo(squadID)
if squadInfo != nil {
    fmt.Printf("Squad %s: %d/%d HP\n", squadInfo.Name, squadInfo.CurrentHP, squadInfo.MaxHP)
}
```

**GetAllSquadInfo**:

```go
allSquads := queries.GetAllSquadInfo()
for _, squad := range allSquads {
    fmt.Println(squad.Name)
}
```

**GetAllSquadIDs**:

```go
squadIDs := queries.GetAllSquadIDs()
```

**GetSquadsByFaction**:

```go
enemySquadIDs := queries.GetSquadsByFaction(enemyFactionID)
```

**GetEnemySquadsForEncounter**:

```go
encounterSquadIDs := queries.GetEnemySquadsForEncounter(playerFactionID, encounterID)
```

### Faction Queries

**GetFactionInfo**:

```go
factionInfo := queries.GetFactionInfo(factionID)
if factionInfo != nil {
    fmt.Printf("Faction %s: %d mana\n", factionInfo.Name, factionInfo.CurrentMana)
}
```

**GetAllFactionInfo**:

```go
allFactions := queries.GetAllFactionInfo()
```

**GetFactionsForEncounter**:

```go
factionIDs := queries.GetFactionsForEncounter(encounterID)
```

**IsPlayerFaction**:

```go
if queries.IsPlayerFaction(factionID) {
    // Handle player faction
}
```

**GetPlayerFaction**:

```go
playerFactionID := queries.GetPlayerFaction()
```

### Squad Filtering

**SquadFilter** type:

```go
type SquadFilter func(squadInfo *SquadInfo) bool
```

**FilterSquadsAlive**:

```go
aliveFilter := queries.FilterSquadsAlive()
aliveSquads := queries.ApplyFilterToSquads(allSquadIDs, aliveFilter)
```

**ApplyFilterToSquads**:

```go
customFilter := func(squad *SquadInfo) bool {
    return squad.CurrentHP > squad.MaxHP/2
}
healthySquads := queries.ApplyFilterToSquads(squadIDs, customFilter)
```

**Chaining Filters**:

```go
aliveAndHealthy := func(squad *SquadInfo) bool {
    return !squad.IsDestroyed && squad.CurrentHP > squad.MaxHP/2
}
filteredSquads := queries.ApplyFilterToSquads(allSquadIDs, aliveAndHealthy)
```

### Cache Invalidation

**MarkSquadDirty**:

```go
queries.MarkSquadDirty(squadID) // Next GetSquadInfo will recompute
```

**MarkAllSquadsDirty**:

```go
queries.MarkAllSquadsDirty() // Invalidate entire cache
```

**InvalidateSquad**:

```go
queries.InvalidateSquad(squadID) // Remove from cache entirely
```

Cache invalidation is typically handled automatically by systems that modify squads, but modes can manually invalidate when needed.

### Rendering Bridge

**guiqueries_rendering.go** provides adapter methods to satisfy rendering interfaces:

```go
// Implements rendering.SquadInfoProvider
func (gq *GUIQueries) GetAllSquadIDs() []ecs.EntityID
func (gq *GUIQueries) GetSquadRenderInfo(squadID ecs.EntityID) rendering.SquadRenderInfo

// Implements rendering.UnitInfoProvider
func (gq *GUIQueries) GetUnitIDsInSquad(squadID ecs.EntityID) []ecs.EntityID
func (gq *GUIQueries) GetUnitRenderInfo(unitID ecs.EntityID) rendering.UnitRenderInfo
```

This allows GUIQueries to be passed directly to rendering functions:

```go
rendering.DrawSquads(screen, queries, cameraX, cameraY)
```

### Best Practices

1. **Always use GUIQueries**: Never directly query ECS in UI code.
2. **Cache invalidation**: Call MarkSquadDirty after modifying squads.
3. **Filter before display**: Use SquadFilter to clean data before showing in UI.
4. **Check nil**: GetSquadInfo/GetFactionInfo return nil if entity doesn't exist.
5. **Prefer batch queries**: GetAllSquadInfo is more efficient than looping GetSquadInfo.

---

## Panel Building System

The panel building system provides three levels of abstraction for creating UI panels, from low-level widget construction to high-level declarative specs.

### Level 0: Raw ebitenui

Direct widget construction (rarely used):

```go
container := widget.NewContainer(
    widget.ContainerOpts.BackgroundImage(guiresources.PanelRes.Image),
    widget.ContainerOpts.Layout(widget.NewRowLayout(...)),
)
```

Use this only for highly specialized widgets not covered by builders.

### Level 1: PanelBuilders Fluent API

Functional options pattern for flexible panel construction:

```go
panel := mode.PanelBuilders.BuildPanel(
    builders.TopLeft(),
    builders.Size(0.3, 0.5),
    builders.Padding(specs.PaddingStandard),
    builders.RowLayout(),
)
```

**Common Options**:

- Position: `builders.TopLeft()`, `builders.TopRight()`, `builders.TopCenter()`, `builders.BottomLeft()`, `builders.BottomRight()`, `builders.BottomCenter()`, `builders.Center()`, `builders.LeftCenter()`, `builders.RightCenter()`
- `builders.Size(widthFraction, heightFraction)` - Size as fraction of screen
- `builders.Padding(paddingFraction)` / `builders.CustomPadding(insets)`
- `builders.RowLayout()` (vertical), `builders.HorizontalRowLayout()`, `builders.AnchorLayout()`

**Anchor Helpers** (for LayoutData positioning, takes pixel values):

```go
builders.AnchorStartStart(leftPadding, topPadding int)  // Top-left anchor
builders.AnchorCenterStart(topPadding int)              // Top-center anchor
builders.AnchorEndCenter(rightPadding int)              // Right-center anchor
builders.AnchorEndEnd(rightPadding, bottomPadding int)  // Bottom-right anchor
builders.AnchorEndStart(rightPadding, topPadding int)   // Top-right anchor
builders.AnchorCenterEnd(bottomPadding int)             // Bottom-center anchor
```

Example:
```go
topPad := int(float64(layout.ScreenHeight) * specs.PaddingTight)
panel := mode.PanelBuilders.BuildPanel(
    builders.Size(0.4, 0.08),
)
panel.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)
```

### Level 1.5: Typed Panel Building

Simplified builders for common panel patterns:

```go
// TypedPanelConfig
result := mode.PanelBuilders.BuildTypedPanel(builders.TypedPanelConfig{
    Type:       builders.PanelTypeDetail,
    SpecName:   "threat_info",
    DetailText: "Select a threat to view details",
})
textArea := result.TextArea
```

**PanelType Constants**:

- `PanelTypeSimple`: Empty container with standard styling
- `PanelTypeDetail`: Panel with a text area for displaying details
- `PanelTypeList`: Panel with a cached list widget
- `PanelTypeGrid`: Panel with a grid layout for buttons/cells

**TypedPanelConfig**:

```go
type TypedPanelConfig struct {
    Type         PanelType
    SpecName     string
    DetailText   string            // For PanelTypeDetail
    ListEntries  []string          // For PanelTypeList
    OnListSelect func(int, string) // For PanelTypeList
    GridRows     int               // For PanelTypeGrid
    GridCols     int               // For PanelTypeGrid
}
```

**TypedPanelResult**:

```go
type TypedPanelResult struct {
    Container *widget.Container
    TextArea  *widgets.CachedTextAreaWrapper // PanelTypeDetail
    List      *widgets.CachedListWrapper     // PanelTypeList
    GridCells [][]*widget.Button             // PanelTypeGrid
}
```

Example:
```go
listResult := mode.PanelBuilders.BuildTypedPanel(builders.TypedPanelConfig{
    Type:         builders.PanelTypeList,
    SpecName:     "squad_list",
    ListEntries:  []string{"Alpha Squad", "Bravo Squad"},
    OnListSelect: func(idx int, name string) {
        mode.selectSquad(idx)
    },
})
mode.squadListWidget = listResult.List
```

### Grid Editor Building

Specialized builder for interactive grid editors (used in squad formation editing):

```go
gridContainer, gridCells := mode.PanelBuilders.BuildGridEditor(builders.GridEditorConfig{
    CellTextFormat: func(row, col int) string {
        return fmt.Sprintf("[%d,%d]", row, col)
    },
    OnCellClick: func(row, col int) {
        mode.handleGridClick(row, col)
    },
})
```

**GridEditorConfig**:

```go
type GridEditorConfig struct {
    CellTextFormat func(row, col int) string
    OnCellClick    func(row, col int)
    Rows           int // Default 3
    Cols           int // Default 3
}
```

Returns the grid container and a 2D slice of button widgets for direct manipulation.

### Static Panel Creation

For panels that never change after creation, use CreateStaticPanel to enable caching:

```go
panel := builders.CreateStaticPanel(builders.ContainerConfig{
    MinWidth:      300,
    MinHeight:     200,
    Background:    guiresources.PanelRes.Image,
    Layout:        widget.NewRowLayout(...),
    EnableCaching: true, // Enable background caching
})
```

This pre-renders the panel background and significantly reduces draw calls.

### Level 2: PanelLayoutSpec

Declarative panel specifications (highest level):

```go
type PanelLayoutSpec struct {
    Name     string
    Position PanelOption
    Width    float64
    Height   float64
    Padding  float64
    Layout   PanelOption
    Custom   *widget.Insets
}
```

**Example Specification**:

```go
specs := map[string]*builders.PanelLayoutSpec{
    "combat_turn_order": {
        Name:     "combat_turn_order",
        Position: builders.TopLeft(),
        Width:    specs.CombatTurnOrderWidth,
        Height:   specs.CombatTurnOrderHeight,
        Padding:  specs.PaddingStandard,
        Layout:   builders.RowLayout(),
    },
    "combat_squad_list": {
        Name:     "combat_squad_list",
        Position: builders.TopRight(),
        Width:    specs.CombatSquadListWidth,
        Height:   specs.CombatSquadListHeight,
        Padding:  specs.PaddingTight,
        Layout:   builders.RowLayout(),
    },
}
```

**Building from Spec**:

```go
spec := specs["combat_turn_order"]
panel := mode.PanelBuilders.BuildPanelFromSpec(spec)
```

PanelLayoutSpec is the recommended approach for mode-wide panel definitions. Define all specs in one place, then reference by name in PanelDescriptors.

### Responsive Sizing Constants

All sizing constants are defined in `gui/specs/layout.go`. Use these constants instead of hardcoded values for consistent, resolution-independent layouts.

**Panel Widths** (as fractions of screen width):

```go
const (
    PanelWidthNarrow     = 0.15
    PanelWidthStandard   = 0.2
    PanelWidthMedium     = 0.3
    PanelWidthWide       = 0.4
    PanelWidthExtraWide  = 0.45
)
```

**Panel Heights** (as fractions of screen height):

```go
const (
    PanelHeightTiny      = 0.08
    PanelHeightSmall     = 0.12
    PanelHeightQuarter   = 0.25
    PanelHeightThird     = 0.33
    PanelHeightHalf      = 0.5
    PanelHeightExtraTall = 0.6
    PanelHeightTall      = 0.75
    PanelHeightFull      = 0.85
)
```

**Padding** (as fractions of screen dimension):

```go
const (
    PaddingExtraSmall    = 0.0125
    PaddingTight         = 0.015
    PaddingStandard      = 0.02
    PaddingLoose         = 0.03
    PaddingStackedWidget = 0.08
)
```

**Layout Offsets**:

```go
const (
    BottomButtonOffset = 0.08
)
```

**Combat Mode Layout**:

```go
const (
    CombatTurnOrderWidth      = 0.4
    CombatFactionInfoWidth    = 0.18
    CombatSquadListWidth      = 0.18
    CombatSquadDetailWidth    = 0.18
    CombatLogWidth            = 0.22
    CombatActionButtonWidth   = 0.35

    CombatTurnOrderHeight     = 0.08
    CombatFactionInfoHeight   = 0.10
    CombatSquadListHeight     = 0.35
    CombatSquadDetailHeight   = 0.25
    CombatLogHeight           = 0.15
    CombatActionButtonHeight  = 0.08
)
```

**Inventory Mode Layout**:

```go
const (
    InventoryListWidth  = 0.5
    InventoryListHeight = PanelHeightTall
)
```

**Squad Management Layout**:

```go
const (
    SquadMgmtPanelWidth  = 0.6
    SquadMgmtPanelHeight = 0.5
    SquadMgmtNavWidth    = 0.5
    SquadMgmtNavHeight   = 0.08
    // ... additional squad management constants
)
```

**Squad Builder Layout**:

```go
const (
    SquadBuilderGridWidth     = 0.5
    SquadBuilderUnitListWidth = 0.25
    // ... additional squad builder constants
)
```

**Squad Editor Layout**:

```go
const (
    SquadEditorNavHeight      = 0.08
    SquadEditorSquadListWidth = 0.25
    // ... additional squad editor constants
)
```

**Unit Purchase Layout**:

```go
const (
    UnitPurchaseListWidth  = 0.35
    UnitPurchaseListHeight = 0.7
    // ... additional unit purchase constants
)
```

**Squad Deployment Layout**:

```go
const (
    SquadDeployListWidth  = 0.3
    SquadDeployListHeight = 0.7
    // ... additional squad deployment constants
)
```

**Formation Editor Layout**, **Info Mode Layout**: Additional mode-specific constants for specialized layouts.

**Usage Example**:

```go
panel := mode.PanelBuilders.BuildPanel(
    builders.TopLeft(),
    builders.Size(specs.CombatSquadListWidth, specs.CombatSquadListHeight),
    builders.Padding(specs.PaddingStandard),
)
```

---

## Widget System

High-level widget creation helpers and wrappers for common UI patterns.

### Button Group

Create horizontal or vertical button groups with consistent styling:

```go
buttonContainer := builders.CreateButtonGroup(builders.ButtonGroupConfig{
    Buttons: []builders.ButtonSpec{
        {Text: "Attack (A)", OnClick: mode.handleAttack},
        {Text: "Move (M)", OnClick: mode.handleMove},
        {Text: "End Turn (Space)", OnClick: mode.handleEndTurn},
    },
    Direction:  widget.DirectionHorizontal,
    Spacing:    spacing,
    Padding:    builders.NewResponsiveHorizontalPadding(layout, specs.PaddingExtraSmall),
    LayoutData: &anchorLayout,
})
```

**ButtonGroupConfig**:

```go
type ButtonGroupConfig struct {
    Buttons    []ButtonSpec
    Direction  widget.Direction
    Spacing    int
    Padding    widget.Insets
    LayoutData interface{}
}
```

**ButtonSpec**:

```go
type ButtonSpec struct {
    Text    string
    OnClick func()
}
```

### Text Input

Create text input fields with validation and callbacks:

```go
nameInput := builders.CreateTextInputWithConfig(builders.TextInputConfig{
    MinWidth:    300,
    Placeholder: "Enter squad name...",
    OnChanged: func(text string) {
        mode.squadName = text
    },
})
```

**TextInputConfig**:

```go
type TextInputConfig struct {
    MinWidth    int
    Placeholder string
    OnChanged   func(string)
    OnSubmit    func(string)
}
```

### Container Configuration

Create panels with consistent configuration:

```go
panel := builders.CreatePanelWithConfig(builders.ContainerConfig{
    MinWidth:      300,
    MinHeight:     200,
    Background:    guiresources.PanelRes.Image,
    Layout:        widget.NewRowLayout(...),
    EnableCaching: true,
})
```

**ContainerConfig**:

```go
type ContainerConfig struct {
    MinWidth      int
    MinHeight     int
    Background    *image.NineSlice
    Layout        widget.Layouter
    EnableCaching bool
}
```

EnableCaching should be true for static panels that don't change after creation.

### Cached Text Area

Text areas with automatic dirty-tracking:

```go
cachedTextArea := builders.CreateCachedTextArea(builders.TextAreaConfig{
    MinWidth:  400,
    MinHeight: 300,
    FontColor: color.White,
})

// Convenience methods automatically mark dirty
cachedTextArea.SetText("New content")
cachedTextArea.AppendText("\nAdditional line")
```

**TextAreaConfig**:

```go
type TextAreaConfig struct {
    MinWidth  int
    MinHeight int
    FontColor color.Color
}
```

**CachedTextAreaWrapper Methods**:

- `SetText(text string)`: Replace all text (auto-marks dirty)
- `AppendText(text string)`: Add to existing text (auto-marks dirty)
- `GetWidget() *widget.Container`: Get underlying widget
- `MarkDirty()`: Manual dirty flag (rarely needed)

### Cached List

Lists with 90% CPU reduction through dirty-tracking:

```go
// Create list using builders, then wrap for caching
list := builders.CreateListWithConfig(builders.ListConfig{
    Entries: entries,
    EntrySelected: func(args *widget.ListEntrySelectedEventArgs) {
        mode.handleSelection(args)
    },
})
cachedList := widgets.NewCachedListWrapper(list)

// Mark dirty when entries change
cachedList.MarkDirty()

// Selection methods
selectedEntry := cachedList.SelectedEntry()       // Returns interface{}
cachedList.SetSelectedEntry(someEntry)             // Set by value
```

**Key Methods**:

- `GetWidget() *widget.Widget`: Get underlying widget for UI hierarchy
- `MarkDirty()`: Force re-render on next frame
- `SelectedEntry() interface{}`: Get current selection
- `SetSelectedEntry(entry interface{})`: Set selection programmatically

**CachedListWrapper Methods**:

- `SetEntries(entries []string)`: Update list content (auto-marks dirty)
- `SelectedEntry() string`: Get currently selected entry (convenience over GetSelectedEntry)
- `SetSelectedEntry(entry string)`: Select entry by value (convenience)
- `GetSelectedEntry() (string, bool)`: Get selected entry with existence check
- `GetSelectedIndex() int`: Get selected index
- `SetSelectedIndex(index int)`: Set selection by index
- `GetWidget() *widget.Container`: Get underlying widget
- `MarkDirty()`: Manual dirty flag (rarely needed)

### Selection Dialog

Modal dialog for choosing from a list of options:

```go
dialog := builders.CreateSelectionDialog(builders.SelectionDialogConfig{
    Title:            "Select Squad to Edit",
    SelectionEntries: []string{"Alpha Squad", "Bravo Squad", "Charlie Squad"},
    OnSelect: func(selected string) {
        mode.editSquad(selected)
    },
    OnCancel: func() {
        // User cancelled
    },
})

mode.ui.AddWindow(dialog)
```

**SelectionDialogConfig**:

```go
type SelectionDialogConfig struct {
    Title            string
    SelectionEntries []string
    OnSelect         func(string)
    OnCancel         func()
}
```

The dialog automatically closes after selection or cancellation.

### Other Dialog Types

**Confirmation Dialog**:

```go
dialog := builders.CreateConfirmationDialog(
    "Delete Squad",
    "Are you sure you want to delete this squad?",
    func() { mode.deleteSquad() },
    func() { /* Cancelled */ },
)
```

**Text Input Dialog**:

```go
dialog := builders.CreateTextInputDialog(
    "Enter Squad Name",
    "Name:",
    func(text string) { mode.createSquad(text) },
    func() { /* Cancelled */ },
)
```

**Message Dialog**:

```go
dialog := builders.CreateMessageDialog(
    "Error",
    "Invalid formation: must have at least one unit",
)
```

---

## Context State

The two-context system (Overworld and BattleMap) maintains separate UI state for each context. This enables seamless transitions between strategic and tactical gameplay.

### BattleMapState

UI state for tactical gameplay (exploration, combat, inventory):

```go
type BattleMapState struct {
    SelectedSquadID      ecs.EntityID
    SelectedTargetID     ecs.EntityID
    InAttackMode         bool
    InMoveMode           bool
    ShowHealthBars       bool
    TriggeredEncounterID ecs.EntityID
}
```

**SelectedSquadID**: Currently selected player squad (for movement, combat actions).

**SelectedTargetID**: Currently targeted enemy squad (in attack mode).

**InAttackMode**: True when player is selecting an attack target.

**InMoveMode**: True when player is selecting a movement destination.

**ShowHealthBars**: True to render health bars over units.

**TriggeredEncounterID**: Encounter that triggered combat transition (0 if none).

**Accessing BattleMapState**:

```go
state := mode.Context.ModeCoordinator.GetBattleMapState()
state.InAttackMode = true
```

### OverworldState

UI state for strategic gameplay (overworld map, threats, tick management):

```go
type OverworldState struct {
    CameraX          int
    CameraY          int
    SelectedThreatID ecs.EntityID
    HoveredPosition  *coords.LogicalPosition
    ShowInfluence    bool
    IsAutoTraveling  bool
}
```

**CameraX/CameraY**: Camera position for overworld map scrolling.

**SelectedThreatID**: Currently selected threat entity (for engagement, info display).

**HoveredPosition**: Mouse-hovered tile position (for placement preview, tooltips).

**ShowInfluence**: True to render influence zone visualization.

**IsAutoTraveling**: True when auto-travel mode is active (automated tick advancement).

**Accessing OverworldState**:

```go
state := mode.Context.ModeCoordinator.GetOverworldState()
state.ShowInfluence = true
```

### Context Switching

The GameModeCoordinator manages transitions between contexts:

```go
// Switch to BattleMap context with a specific initial mode
mode.Context.ModeCoordinator.EnterBattleMap("exploration")

// Switch to Overworld context with a specific initial mode
mode.Context.ModeCoordinator.ReturnToOverworld("squad_management")

// Within a context, transition between modes using the mode manager
if targetMode, exists := mode.ModeManager.GetMode("combat"); exists {
    mode.ModeManager.RequestTransition(targetMode, "enter combat")
}
```

Each context maintains its own mode manager. Switching contexts preserves the previous mode stack, so returning to a context restores the last active mode.

**Example Flow**:

1. Player is in Overworld mode (Overworld context)
2. Selects threat and engages (EnterBattleMap → Exploration mode)
3. Enters combat (RequestTransition → Combat mode)
4. Combat ends (RequestTransition → Exploration mode)
5. Returns to overworld (ReturnToOverworld → Overworld mode)

State is preserved across context switches. OverworldState retains SelectedThreatID even when in BattleMap context.

---

## Mode Catalog

Comprehensive reference for all UI modes in TinkerRogue.

### BattleMap Context Modes

#### Exploration Mode

**File**: `gui/guiexploration/explorationmode.go`

**Mode Name**: `"exploration"`

**Purpose**: Default gameplay mode for exploring dungeons, moving squads, and initiating encounters.

**Panels**:
- `ExplorationPanelMessageLog`: Message log display
- `ExplorationPanelQuickInventory`: Quick access inventory panel

**Features**:
- Squad movement via mouse click
- Encounter triggering
- Quick inventory access
- Message log display

**Hotkeys**:
- `I`: Enter Inventory mode
- `C`: Enter Combat mode (debug)
- `D`: Enter Squad Deployment mode
- `ESC`: Return to Overworld

**Update Logic**:
- Handles mouse clicks for squad movement
- Updates message log from game events
- Processes hotkey input for mode transitions

**Draw Logic**:
- Renders game map with squads and entities
- Draws message log overlay

---

#### Combat Mode

**File**: `gui/guicombat/combatmode.go`

**Mode Name**: `"combat"`

**Purpose**: Turn-based tactical combat interface with threat visualization, action execution, and animation playback.

**Panels**:
- `CombatPanelTurnOrder`: Current turn order display
- `CombatPanelFactionInfo`: Faction stats (mana, squad count)
- `CombatPanelSquadList`: List of squads in combat
- `CombatPanelSquadDetail`: Selected squad details
- `CombatPanelCombatLog`: Combat event log
- `CombatPanelLayerStatus`: Layer visualization status (when active)

**Features**:
- Turn-based combat flow
- Attack mode with target selection
- Move mode with valid destination highlighting
- Threat heatmap visualization (gradient or threshold view)
- Layer visualization (armor, evasion, threat)
- Health bar rendering
- Combat log with damage/action reporting
- Undo/redo for movement
- Animation playback for attacks
- Command history

**Hotkeys**:
- `A`: Toggle attack mode
- `M`: Toggle move mode
- `TAB`: Cycle squad selection
- `Space`: End turn
- `H`: Toggle threat heatmap
- `Shift+H`: Switch threat view (gradient/threshold)
- `L`: Toggle layer visualization
- `Shift+L`: Cycle layer mode (armor/evasion/threat)
- `Ctrl+Right`: Toggle health bars
- `1`, `2`, `3`: Select enemy target (in attack mode)
- `Ctrl+Z`: Undo last move
- `Ctrl+Y`: Redo move
- `Ctrl+K`: Kill all enemies (debug)
- `ESC`: Return to exploration

**Update Logic**:
- Manages turn flow (player turn, enemy turn, turn end)
- Handles attack/move action execution
- Processes map clicks for targeting/movement
- Updates squad selection and details
- Formats combat log messages
- Triggers combat animation mode for attacks

**Draw Logic**:
- Renders game map with combat entities
- Draws threat heatmap overlay (when enabled)
- Draws layer visualization overlay (when enabled)
- Renders health bars (when enabled)
- Highlights valid movement tiles
- Shows attack range indicators

**Key Files**:
- `combatmode.go`: Main mode implementation
- `combatdeps.go`: Dependency injection
- `combat_action_handler.go`: Attack/move execution
- `combat_input_handler.go`: Map click and hotkey handling
- `combat_log_manager.go`: Combat log formatting
- `combat_turn_flow.go`: Turn lifecycle management
- `combatvisualization.go`: Threat heatmaps, health bars, layer visualization

---

#### Combat Animation Mode

**File**: `gui/guicombat/combat_animation_mode.go`

**Mode Name**: `"combat_animation"`

**Purpose**: Full-screen battle scene showing both squads side-by-side during attack animations. Provides visual feedback for combat actions with color-coded targeting.

**Panels**:
- `CombatAnimationPanelPrompt`: User prompt ("Press any key to continue" or "Press Space to replay")

**Animation Phases**:
1. **Idle**: Initial state before animation starts
2. **Attacking**: Animation playing (auto-advances)
3. **Waiting**: Animation complete, waiting for user input
4. **Complete**: User acknowledged, ready to return to combat

**Features**:
- Side-by-side squad rendering
- Color-coded unit highlighting (attackers/defenders)
- Auto-play mode for AI attacks
- Replay support
- Smooth phase transitions

**Hotkeys**:
- `Space`: Replay animation (in waiting phase)
- `ESC`: Skip animation immediately
- Any other key/mouse: Continue (in waiting phase)

**Update Logic**:
- Auto-advances through animation phases
- Handles user input for replay/continue
- Returns to combat mode on completion

**Draw Logic**:
- Renders attacking squad on left side
- Renders defending squad on right side
- Highlights attacking units (color-coded)
- Shows damage indicators
- Displays user prompt

**Key Files**:
- `combat_animation_mode.go`: Main animation mode
- `combat_animation_panels_registry.go`: Panel registrations

---

#### Inventory Mode

**File**: `gui/guiexploration/inventorymode.go`

**Mode Name**: `"inventory"`

**Purpose**: Full-screen item management interface.

**Panels**:
- Full-screen inventory list with item details

**Features**:
- Item browsing
- Item details display
- Equipment management (future)

**Hotkeys**:
- `ESC`: Return to exploration

**Update Logic**:
- Handles item selection
- Updates item details panel

**Draw Logic**:
- Renders inventory list
- Shows item details on selection

---

### Overworld Context Modes

#### Overworld Mode

**File**: `gui/guioverworld/overworldmode.go`

**Mode Name**: `"overworld"`

**Purpose**: Strategic layer where the player manages overworld ticks, views threats, initiates travel, and engages encounters. The overworld is a large-scale map with threats, nodes, and influence zones.

**Panels**:
- `OverworldPanelTickControls`: Tick advancement controls (Advance Tick, Auto-Travel, Toggle Influence)
- `OverworldPanelThreatInfo`: Selected threat details (name, position, strength)
- `OverworldPanelTickStatus`: Current tick count and travel status
- `OverworldPanelEventLog`: Recent overworld events (threat spawned, node captured, etc.)
- `OverworldPanelThreatStats`: Aggregated threat statistics (total threats, strongest threat, etc.)

**Features**:
- Tick advancement (manual and auto)
- Threat selection and engagement
- Influence zone visualization
- Auto-travel mode (automated tick advancement until destination)
- Travel initiation and cancellation
- Node placement mode entry
- Overworld event logging (with export support)
- Camera scrolling

**Hotkeys**:
- `Space`: Advance tick (manual)
- `A`: Toggle auto-travel
- `I`: Toggle influence zone visualization
- `E`: Engage selected threat (initiate travel)
- `C`: Cancel travel
- `N`: Enter node placement mode
- `W/S/D/Q/Z`: Movement keys (advance time)
- `ESC`: Return to BattleMap context
- Mouse click: Select/deselect threat

**Update Logic**:
- Handles tick advancement (manual and auto)
- Processes threat selection from mouse clicks
- Manages travel state (initiation, cancellation, completion)
- Updates event log from overworld events
- Handles camera scrolling

**Draw Logic**:
- Renders overworld map (threats, nodes, influence zones)
- Draws influence zone overlay (when enabled)
- Highlights selected threat
- Shows travel path (when traveling)
- Renders camera-adjusted entities

**Key Files**:
- `overworldmode.go`: Main overworld mode
- `overworld_renderer.go`: Overworld visualization
- `overworld_panels_registry.go`: Panel registrations

---

#### Node Placement Mode

**File**: `gui/guinodeplacement/nodeplacementmode.go`

**Mode Name**: `"node_placement"`

**Purpose**: Allows the player to place strategic nodes on the overworld map. Nodes provide control points, resource generation, or defensive positions.

**Panels**:
- `NodePlacementPanelNodeList`: Available node types
- `NodePlacementPanelInfo`: Node type details and placement instructions
- `NodePlacementPanelControls`: Node type cycling and placement controls

**Features**:
- Node type selection (cycle or direct)
- Placement validation (check if position is valid)
- Cursor preview (shows node at hovered position)
- Node placement on map click

**Hotkeys**:
- `TAB`: Cycle node type
- `1`, `2`, `3`, `4`: Select node type directly
- `ESC`: Return to overworld
- Mouse click: Place node at cursor position

**Update Logic**:
- Handles node type selection
- Validates placement positions
- Processes map clicks for node placement
- Updates cursor preview position

**Draw Logic**:
- Renders overworld map
- Draws placement cursor preview
- Highlights valid placement positions

**Key Files**:
- `nodeplacementmode.go`: Main node placement mode
- `nodeplacement_renderer.go`: Placement preview rendering
- `nodeplacement_panels_registry.go`: Panel registrations

---

#### Squad Management Mode

**File**: `gui/guisquads/squadmanagementmode.go`

**Mode Name**: `"squad_management"`

**Purpose**: Overview of all squads with navigation to editing, purchasing, and deployment.

**Panels**:
- Squad list panel
- Navigation panel (Edit, Purchase Units, Deploy, New Squad)

**Features**:
- Squad list display with stats
- Navigation to other squad modes
- Squad selection for editing

**Hotkeys**:
- `ESC`: Return to previous mode

**Update Logic**:
- Updates squad list from GUIQueries
- Handles squad selection
- Processes navigation button clicks

**Draw Logic**:
- Minimal (mostly ebitenui panels)

---

#### Squad Builder Mode

**File**: `gui/guisquads/squadbuilder.go`

**Mode Name**: `"squad_builder"`

**Purpose**: Create new squads with name and initial formation.

**Panels**:
- Name input panel
- 3x3 grid editor
- Unit roster (available units for placement)
- Create/Cancel buttons

**Features**:
- Squad naming
- Unit placement in 3x3 grid
- Formation validation
- Squad creation

**Hotkeys**:
- `ESC`: Cancel and return to squad management

**Update Logic**:
- Handles name input
- Processes grid cell clicks for unit placement
- Validates formation (at least one unit)
- Creates squad entity on confirmation

**Draw Logic**:
- Renders grid editor
- Shows unit roster

---

#### Squad Editor Mode

**File**: `gui/guisquads/squadeditormode.go`

**Mode Name**: `"squad_editor"`

**Purpose**: Edit existing squad formations (3x3 grid).

**Panels**:
- Squad list (select squad to edit)
- 3x3 grid editor
- Unit roster (units in squad)
- Save/Cancel buttons

**Features**:
- Squad selection
- Formation editing (drag-and-drop or click-to-place)
- Formation validation
- Save changes

**Hotkeys**:
- `ESC`: Cancel and return to squad management

**Update Logic**:
- Handles squad selection
- Processes grid cell clicks for unit movement
- Validates formation
- Saves changes to squad entity

**Draw Logic**:
- Renders grid editor
- Shows unit roster

**Key Files**:
- `squadeditormode.go`: Main editor mode
- `squadeditor_grid.go`: Grid editor component
- `squadeditor_roster.go`: Unit roster component
- `squadeditor_refresh.go`: UI refresh logic
- `squadeditor_movesquad.go`: Squad movement logic

---

#### Unit Purchase Mode

**File**: `gui/guisquads/unitpurchasemode.go`

**Mode Name**: `"unit_purchase"`

**Purpose**: Purchase units and add them to squads.

**Panels**:
- Available units list (with costs)
- Player gold display
- Squad selection (which squad to add unit to)
- Purchase/Cancel buttons

**Features**:
- Unit browsing
- Cost display
- Gold validation
- Unit purchase and assignment

**Hotkeys**:
- `ESC`: Cancel and return to squad management

**Update Logic**:
- Handles unit selection
- Validates gold cost
- Processes purchase transaction
- Adds unit to selected squad

**Draw Logic**:
- Renders unit list
- Shows gold balance

---

#### Squad Deployment Mode

**File**: `gui/guisquads/squaddeploymentmode.go`

**Mode Name**: `"squad_deployment"`

**Purpose**: Place squads on the game map.

**Panels**:
- Squad list (undeployed squads)
- Placement instructions
- Cancel button

**Features**:
- Squad selection
- Map click for placement
- Placement validation (walkable tiles, no overlap)

**Hotkeys**:
- `ESC`: Cancel and return to previous mode
- Mouse click: Place selected squad

**Update Logic**:
- Handles squad selection
- Processes map clicks for placement
- Validates placement positions
- Updates squad position entity

**Draw Logic**:
- Renders game map
- Highlights valid placement positions
- Shows cursor preview

---

## Performance Optimization

TinkerRogue's GUI system employs multiple optimization strategies to minimize CPU usage and maintain 60 FPS.

### Cached Rendering

**CachedListWrapper** reduces list rendering CPU by 90% through dirty-tracking:

```go
type CachedListWrapper struct {
    widget   *widget.Container
    list     *widget.List
    entries  []string
    isDirty  bool
}
```

**How it works**:
1. SetEntries marks isDirty = true
2. ebitenui calls PreferredSize on every frame
3. If isDirty, recalculate size and rebuild list
4. If clean, return cached size (no rebuild)

**CachedTextAreaWrapper** uses the same pattern for text areas:

```go
cachedTextArea.SetText("New text")  // Marks dirty
cachedTextArea.AppendText("\nMore") // Marks dirty
// Next frame: rebuilds once, then caches
```

### CachedBackgroundPool

Global pool of pre-rendered NineSlice backgrounds for panels and scroll containers.

**Structure**:

```go
type CachedBackgroundPool struct {
    source *image.NineSlice
    cache  map[cacheKey]*CachedBackground
}

type cacheKey struct {
    width  int
    height int
}

type CachedBackground struct {
    image  *ebiten.Image
    width  int
    height int
}
```

**Global Accessor Functions**:

```go
// Get cached panel background (creates if not cached)
bg := guiresources.GetPanelBackground(width, height)

// Get cached scroll container backgrounds
idleBg := guiresources.GetScrollContainerIdleBackground(width, height)
disabledBg := guiresources.GetScrollContainerDisabledBackground(width, height)
maskBg := guiresources.GetScrollContainerMaskBackground(width, height)
```

**Pre-Caching**:

Pre-rendering common sizes at startup significantly reduces first-frame stuttering:

```go
// Pre-cache at common 1920x1080 sizes
guiresources.PreCacheScrollContainerBackgrounds()

// Pre-cache at runtime screen dimensions
guiresources.PreCacheScrollContainerSizes(screenWidth, screenHeight)
```

Call `PreCacheScrollContainerBackgrounds()` at game startup and `PreCacheScrollContainerSizes()` after window resize.

**How it works**:
1. First request for a size: NineSlice renders to image, caches result
2. Subsequent requests: Returns cached image (no render)
3. Same size used across multiple widgets: Single cached image shared

**Impact**: Reduces draw calls by 70-80% for panel-heavy modes. Essential for modes with many scroll containers (Squad Management, Inventory).

### Static Panel Caching

Panels that never change after creation can enable caching to pre-render their backgrounds:

```go
panel := builders.CreateStaticPanel(builders.ContainerConfig{
    MinWidth:      300,
    MinHeight:     200,
    Background:    guiresources.PanelRes.Image,
    Layout:        widget.NewRowLayout(...),
    EnableCaching: true, // Pre-render background
})
```

This is automatically used by `BuildPanelFromSpec` for spec-based panels.

### Query Caching

**GUIQueries** maintains two levels of caching:

1. **ECS-level caching** (SquadCache, CombatCache): Caches ECS queries for common lookups (GetSquadByID, GetSquadMembers).

2. **UI-level caching** (SquadInfoCache): Caches aggregated SquadInfo/FactionInfo structs. Only recomputes when invalidated.

**Invalidation**:

```go
// Invalidate specific squad (next GetSquadInfo will recompute)
queries.MarkSquadDirty(squadID)

// Invalidate all squads (after major state change)
queries.MarkAllSquadsDirty()
```

Systems that modify squads automatically invalidate caches. Modes rarely need manual invalidation.

**Impact**: 95% reduction in ECS query overhead for squad-heavy UI (Combat, Squad Management).

### Best Practices

1. **Use CachedListWrapper/CachedTextAreaWrapper** for all lists/text areas. Never use raw ebitenui widgets.

2. **Pre-cache backgrounds** at startup and after resize. Call `PreCacheScrollContainerSizes()` in window resize handler.

3. **Enable static panel caching** for panels that don't change. Use `EnableCaching: true` in ContainerConfig.

4. **Invalidate caches surgically**. Call `MarkSquadDirty(squadID)` for individual changes, not `MarkAllSquadsDirty()`.

5. **Batch UI updates**. Update all panel widgets in a single frame, not incrementally.

6. **Avoid redundant queries**. Cache GUIQueries results in mode state if used multiple times per frame.

---

## Adding New Modes

Step-by-step guide for creating a new UI mode.

### Step 1: Define Panel Types

Create panel type constants:

```go
// mymode_panels_registry.go
const (
    MyModePanelHeader PanelType = "mymode_header"
    MyModePanelList   PanelType = "mymode_list"
    MyModePanelDetail PanelType = "mymode_detail"
)
```

### Step 2: Create Panel Specifications

Define panel layout specs:

```go
// mymode_panels_registry.go
var myModePanelSpecs = map[string]*builders.PanelLayoutSpec{
    "mymode_header": {
        Name:     "mymode_header",
        Position: builders.TopLeft(),
        Width:    specs.PanelWidthWide,
        Height:   specs.PanelHeightSmall,
        Padding:  specs.PaddingStandard,
        Layout:   builders.RowLayout(),
    },
    "mymode_list": {
        Name:     "mymode_list",
        Position: builders.TopRight(),
        Width:    specs.PanelWidthStandard,
        Height:   specs.PanelHeightTall,
        Padding:  specs.PaddingTight,
        Layout:   builders.RowLayout(),
    },
}
```

### Step 3: Register Panels

Create registration in `init()` using the global registry:

```go
// mymode_panels_registry.go
func init() {
    framework.RegisterPanel(MyModePanelHeader, framework.PanelDescriptor{
        Content: framework.ContentText,
        OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
            bm := mode.(*MyMode)
            layout := bm.Layout

            panelWidth := int(float64(layout.ScreenWidth) * specs.PanelWidthStandard)
            panelHeight := int(float64(layout.ScreenHeight) * specs.PanelHeightSmall)

            result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
                MinWidth:   panelWidth,
                MinHeight:  panelHeight,
                Background: guiresources.PanelRes.Image,
                Layout: widget.NewRowLayout(
                    widget.RowLayoutOpts.Direction(widget.DirectionVertical),
                ),
            })
            result.Container.GetWidget().LayoutData = builders.AnchorCenterStart(10)

            result.TextLabel = builders.CreateSmallLabel("My Mode")
            result.Container.AddChild(result.TextLabel)
            return nil
        },
    })

    framework.RegisterPanel(MyModePanelList, framework.PanelDescriptor{
        Content: framework.ContentEmpty, // Container only, populated by component
        OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
            bm := mode.(*MyMode)
            layout := bm.Layout

            panelWidth := int(float64(layout.ScreenWidth) * specs.PanelWidthStandard)
            panelHeight := int(float64(layout.ScreenHeight) * specs.PanelHeightTall)

            result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
                MinWidth:   panelWidth,
                MinHeight:  panelHeight,
                Background: guiresources.PanelRes.Image,
                Layout: widget.NewRowLayout(
                    widget.RowLayoutOpts.Direction(widget.DirectionVertical),
                    widget.RowLayoutOpts.Spacing(5),
                ),
            })
            leftPad := int(float64(layout.ScreenWidth) * specs.PaddingTight)
            result.Container.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, 10)

            return nil
        },
    })
}
```

### Step 4: Define Mode Struct

Create mode struct embedding BaseMode:

```go
// mymode.go
type MyMode struct {
    framework.BaseMode

    // Mode-specific widgets
    listWidget *widgets.CachedListWrapper

    // Mode-specific state
    selectedIndex int
}
```

### Step 5: Implement Constructor and Initialize

Create the constructor and Initialize method:

```go
// mymode.go
func NewMyMode(modeManager *framework.UIModeManager) *MyMode {
    m := &MyMode{}
    m.SetModeName("mymode")
    m.SetReturnMode("previous_mode")  // ESC returns here
    m.ModeManager = modeManager
    m.SetSelf(m)  // Required for panel registry building
    return m
}

func (m *MyMode) Initialize(ctx *framework.UIContext) error {
    // Build base UI using ModeBuilder
    err := framework.NewModeBuilder(&m.BaseMode, framework.ModeConfig{
        ModeName:   "mymode",
        ReturnMode: "previous_mode",
        Hotkeys: []framework.HotkeySpec{
            {Key: ebiten.KeyR, TargetMode: "some_other_mode"},
        },
    }).Build(ctx)
    if err != nil {
        return err
    }

    // Build panels from global registry
    if err := m.BuildPanels(
        MyModePanelHeader,
        MyModePanelList,
    ); err != nil {
        return err
    }

    return nil
}
```

### Step 6: Implement UIMode Interface

Implement the required interface methods. BaseMode provides default implementations
for most methods, so you only need to override what you need:

```go
// mymode.go

func (m *MyMode) Enter(fromMode framework.UIMode) error {
    // Refresh UI on mode entry
    m.refreshList()
    return nil
}

func (m *MyMode) Exit(toMode framework.UIMode) error {
    // Cleanup on mode exit
    m.selectedIndex = -1
    return nil
}

func (m *MyMode) Update(deltaTime float64) error {
    // Per-frame updates (only if needed)
    // Most modes don't need this
    return nil
}

func (m *MyMode) Render(screen *ebiten.Image) {
    // Custom rendering (optional)
    // e.g., draw map overlay, custom graphics
    // ebitenui handles widget rendering automatically
}

func (m *MyMode) HandleInput(inputState *framework.InputState) bool {
    // Process common input first (registered hotkeys, ESC)
    if m.HandleCommonInput(inputState) {
        return true
    }

    // Mode-specific input handling
    if inputState.MousePressed {
        // Handle mouse clicks
    }

    if inputState.KeysJustPressed[ebiten.KeyR] {
        m.refreshList()
        return true
    }

    return false
}

// GetEbitenUI and GetModeName are inherited from BaseMode
```

### Step 7: Add Helper Methods

Implement mode-specific logic:

```go
// mymode.go

func (m *MyMode) refreshList() {
    // Query game state
    items := m.Queries.GetAllSquadInfo()

    // Update list panel content
    listPanel := m.GetPanelContainer(MyModePanelList)
    if listPanel != nil {
        // Rebuild list contents
        // (specific implementation depends on your list widget choice)
    }
}

func (m *MyMode) updateDetailPanel() {
    result := m.Panels.Get(MyModePanelDetail)
    if result == nil {
        return
    }
    if m.selectedIndex >= 0 {
        items := m.Queries.GetAllSquadInfo()
        result.TextLabel.Label = fmt.Sprintf("Selected: %s", items[m.selectedIndex].Name)
    } else {
        result.TextLabel.Label = "No selection"
    }
}
```

### Step 8: Integrate with Mode Manager

Register the mode with the UIModeManager:

```go
// In game initialization
myMode := NewMyMode(modeManager)
modeManager.RegisterMode("mymode", myMode)
```

The mode will be initialized when first activated via `modeManager.RequestTransition()`.

### Step 9: Test

Test all functionality:
- Panel rendering and positioning
- Hotkey bindings (HandleCommonInput)
- Custom input handling (HandleInput)
- Mode transitions (RequestTransition, context switching)
- Enter/Exit lifecycle
- Query integration

---

## Best Practices

### UI State vs Game State

**UI State** (lives in mode struct or context state):
- Selected index/entity ID
- Scroll position
- Panel visibility flags
- Cursor position
- UI mode flags (InAttackMode, ShowHealthBars)

**Game State** (lives in ECS components):
- Entity positions
- Health/stats
- Inventory contents
- Combat status
- Squad membership

**Rule**: If it affects gameplay logic or needs to persist across mode changes, it's game state. If it only affects UI rendering or interaction, it's UI state.

### Query First

Always access game state through GUIQueries, never directly through ECS:

```go
// GOOD
squadInfo := mode.Queries.GetSquadInfo(squadID)

// BAD
entity := mode.Context.ECSManager.FindEntityByID(squadID)
data := common.GetComponentType[*SquadData](entity, SquadComponent)
```

GUIQueries provides caching, validation, and a clean abstraction. Direct ECS access bypasses optimizations and couples UI to ECS implementation.

### Panel Registry Over Manual Construction

Use PanelRegistry for all panels, not manual widget construction:

```go
// GOOD - Register in init(), build via BuildPanels()
func init() {
    framework.RegisterPanel(MyPanelType, framework.PanelDescriptor{
        Content: framework.ContentCustom,
        OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
            // Build widgets, set result.Container, etc.
            return nil
        },
    })
}

// In mode Initialize():
m.BuildPanels(MyPanelType)

// BAD
myPanel := widget.NewContainer(...)
mode.RootContainer.AddChild(myPanel)
```

PanelRegistry provides lifecycle management, automatic positioning, and centralized panel access.

### Responsive Sizing

Use constants from `gui/specs/layout.go`, not hardcoded values:

```go
// GOOD - Use named constants
panelWidth := int(float64(layout.ScreenWidth) * specs.PanelWidthStandard)
panelHeight := int(float64(layout.ScreenHeight) * specs.PanelHeightTall)

// BAD - Hardcoded fractions
panelWidth := int(float64(layout.ScreenWidth) * 0.2)
panelHeight := int(float64(layout.ScreenHeight) * 0.75)
```

Constants ensure consistency and make global layout changes easy.

### Cached Widgets

Use CachedListWrapper and CachedTextAreaWrapper for all lists and text areas:

```go
// GOOD - Use cached wrappers with dirty-tracking
textArea := builders.CreateCachedTextArea(config)
textArea.SetText("Updated text")  // Auto-marks dirty

// BAD
list := widget.NewList(...) // No caching, rebuilds every frame
```

Cached widgets reduce CPU by 90%.

### Invalidate Surgically

Invalidate caches only when necessary, and prefer targeted invalidation:

```go
// GOOD (specific squad changed)
mode.Queries.MarkSquadDirty(squadID)

// ACCEPTABLE (major state change)
mode.Queries.MarkAllSquadsDirty()

// BAD (invalidating on every frame)
mode.Queries.MarkAllSquadsDirty() // In Update()
```

Cache invalidation is expensive. Only invalidate when game state changes.

### Hotkey Registration

There are two approaches for hotkeys:

**ModeBuilder hotkeys** for mode transitions (processed by HandleCommonInput):
```go
// GOOD - In ModeConfig for mode transitions
framework.NewModeBuilder(&m.BaseMode, framework.ModeConfig{
    Hotkeys: []framework.HotkeySpec{
        {Key: ebiten.KeyE, TargetMode: "squad_management"},
    },
}).Build(ctx)

// Or register individually
m.RegisterHotkey(ebiten.KeyE, "squad_management")
```

**HandleInput** for mode-specific actions:
```go
// GOOD - In HandleInput for custom actions
func (m *MyMode) HandleInput(inputState *framework.InputState) bool {
    if m.HandleCommonInput(inputState) { return true }  // Hotkeys + ESC
    if inputState.KeysJustPressed[ebiten.KeyR] {
        m.refresh()
        return true
    }
    return false
}
```

Registered hotkeys are automatically processed by HandleCommonInput.

### Enter Refresh

Always refresh UI in Enter, not in Update:

```go
func (m *MyMode) Enter(fromMode framework.UIMode) error {
    m.refreshList()        // Refresh on mode entry
    m.updateDetailPanel()
    return nil
}

func (m *MyMode) Update(deltaTime float64) error {
    // Don't refresh every frame - only update if state changed
    return nil
}
```

Enter ensures UI is up-to-date when entering the mode. Update should only refresh in response to user input or events.

### Separation of Concerns

Keep modes focused on UI, not game logic:

```go
// GOOD (call game system)
func (m *CombatMode) handleAttack() {
    combat.ExecuteAttack(attacker, target)
    m.refreshCombatLog()
}

// BAD (implement attack logic in mode)
func (m *CombatMode) handleAttack() {
    // Calculate damage
    // Apply damage
    // Check for death
    // Update ECS
    // ...
}
```

Modes orchestrate UI and call game systems. Game systems implement logic.

---

## Common Patterns

### Selecting from a List

Use the SquadListComponent pattern (as in CombatMode):

```go
// Create a squad list component with filter and selection callback
m.squadListComponent = guisquads.NewSquadListComponent(
    m.GetPanelContainer(MyPanelSquadList),
    m.Queries,
    func(info *framework.SquadInfo) bool {
        return !info.IsDestroyed  // Filter function
    },
    func(squadID ecs.EntityID) {
        m.handleSquadSelected(squadID)
    },
)

// Refresh when state changes
m.squadListComponent.Refresh()
```

### Updating Detail Panel on Selection

```go
func (m *MyMode) updateDetailPanel(squadID ecs.EntityID) {
    result := m.Panels.Get(MyModePanelDetail)
    if result == nil {
        return
    }

    if squadID == 0 {
        result.TextLabel.Label = "No selection"
        return
    }

    info := m.Queries.GetSquadInfo(squadID)
    if info != nil {
        result.TextLabel.Label = fmt.Sprintf("Name: %s\nUnits: %d", info.Name, info.TotalUnits)
    }
}
```

### Modal Confirmation

```go
func (m *MyMode) confirmDelete() {
    dialog := builders.CreateConfirmationDialog(
        "Confirm Delete",
        "Are you sure you want to delete this squad?",
        func() { m.executeDelete() },
        func() { /* Cancelled */ },
    )
    m.GetEbitenUI().AddWindow(dialog)
}

func (m *MyMode) executeDelete() {
    squads.DeleteSquad(m.selectedSquadID, m.Context.ECSManager)
    m.Queries.InvalidateSquad(m.selectedSquadID)
    m.refreshList()
}
```

### Text Input Dialog

```go
func (m *MyMode) promptForName() {
    dialog := builders.CreateTextInputDialog(
        "Enter Squad Name",
        "Name:",
        func(text string) { m.createSquad(text) },
        func() { /* Cancelled */ },
    )
    m.GetEbitenUI().AddWindow(dialog)
}
```

### Mode Transition

```go
// Simple mode transition via RequestTransition
func (m *SquadManagementMode) editSquad(squadID ecs.EntityID) {
    // Store data in shared context state before transitioning
    m.Context.ModeCoordinator.GetBattleMapState().SelectedSquadID = squadID

    if editorMode, exists := m.ModeManager.GetMode("squad_editor"); exists {
        m.ModeManager.RequestTransition(editorMode, "edit squad")
    }
}

// In SquadEditorMode - read shared state on entry
func (m *SquadEditorMode) Enter(fromMode framework.UIMode) error {
    battleState := m.Context.ModeCoordinator.GetBattleMapState()
    if battleState.SelectedSquadID != 0 {
        m.loadSquad(battleState.SelectedSquadID)
    }
    return nil
}
```

### Context Switching

```go
// From BattleMap to Overworld
func (m *ExplorationMode) returnToOverworld() {
    m.Context.ModeCoordinator.ReturnToOverworld("squad_management")
}

// From Overworld to BattleMap
func (m *OverworldMode) returnToBattleMap() {
    m.Context.ModeCoordinator.EnterBattleMap("exploration")
}
```

### Undo/Redo

```go
// Enable command history in ModeBuilder
framework.NewModeBuilder(&m.BaseMode, framework.ModeConfig{
    Commands:  true,
    OnRefresh: func() { m.refreshCurrentSquad() },
}).Build(ctx)

// Record a command
m.CommandHistory.Execute(&MoveCommand{
    squadID: m.selectedSquadID,
    from:    from,
    to:      to,
})

// Handle undo/redo in HandleInput
if inputState.KeysJustPressed[ebiten.KeyZ] && inputState.KeysPressed[ebiten.KeyControl] {
    m.CommandHistory.Undo()
    return true
}
```

### Filtering Query Results

```go
// Get all alive player squads
allSquads := mode.Queries.GetAllSquadInfo()
playerFactionID := mode.Queries.GetPlayerFaction()

alivePlayerSquads := []SquadInfo{}
for _, squad := range allSquads {
    if squad.FactionID == playerFactionID && !squad.IsDestroyed {
        alivePlayerSquads = append(alivePlayerSquads, squad)
    }
}

// Or use filter helper
aliveFilter := mode.Queries.FilterSquadsAlive()
playerSquadIDs := mode.Queries.GetSquadsByFaction(playerFactionID)
alivePlayerSquadIDs := mode.Queries.ApplyFilterToSquads(playerSquadIDs, aliveFilter)
```

### Dynamic Panel Positioning

```go
// In OnCreate, position relative to layout using anchor helpers
OnCreate: func(result *framework.PanelResult, mode framework.UIMode) error {
    bm := mode.(*MyMode)
    layout := bm.Layout

    // Calculate offset based on another panel's size
    topOffset := int(float64(layout.ScreenHeight) * (specs.PanelHeightSmall + specs.PaddingTight))

    result.Container = builders.CreatePanelWithConfig(builders.ContainerConfig{
        MinWidth:  int(float64(layout.ScreenWidth) * 0.5),
        MinHeight: int(float64(layout.ScreenHeight) * 0.7),
    })
    result.Container.GetWidget().LayoutData = builders.AnchorStartStart(10, topOffset)
    return nil
},
```

---

## Troubleshooting

### Panel Not Appearing

**Symptom**: Panel registered but not visible.

**Causes**:
1. Panel positioned off-screen (check Position option)
2. Panel size is zero (check Width/Height)
3. Panel created but not added to RootContainer (check OnCreate)
4. Panel obscured by another panel (check z-order)

**Solution**:
- Verify Position/Width/Height in OnCreate
- Check OnCreate sets result.Container and adds widgets
- Verify BuildPanels was called in Initialize

### List Not Updating

**Symptom**: SetEntries called but list doesn't refresh.

**Causes**:
1. Using raw widget.List instead of CachedListWrapper
2. Not calling SetEntries (calling direct list methods)
3. List widget reference is nil

**Solution**:
- Use CachedListWrapper.SetEntries (auto-marks dirty)
- Verify listWidget is stored in OnCreate
- Check GetPanelWidget returns non-nil

### Hotkey Not Working

**Symptom**: Pressing key doesn't trigger action.

**Causes**:
1. Hotkey not registered (for mode transitions) or not handled in HandleInput (for custom actions)
2. HandleCommonInput not called in HandleInput
3. Another mode/dialog consuming input
4. Using KeysPressed instead of KeysJustPressed

**Solution**:
- For mode transitions: Register with RegisterHotkey or ModeConfig.Hotkeys
- Call HandleCommonInput first in HandleInput
- For custom actions: Check in HandleInput after HandleCommonInput
- Check modal dialogs aren't blocking input
- Use KeysJustPressed for single-fire actions

### Cache Not Invalidating

**Symptom**: UI shows stale data after game state change.

**Causes**:
1. Forgot to call MarkSquadDirty after modification
2. Cache invalidation in wrong place
3. Using cached data directly without re-querying

**Solution**:
- Call MarkSquadDirty immediately after modifying squad
- Call MarkAllSquadsDirty after major state changes
- Always re-query after invalidation

### Mode Transition Not Working

**Symptom**: RequestTransition called but mode doesn't change.

**Causes**:
1. Mode not registered with UIModeManager
2. Wrong mode name passed to GetMode
3. Using context switching when should use mode transition (or vice versa)

**Solution**:
- Verify mode registered with modeManager.RegisterMode
- Use ModeManager.GetMode(name) to retrieve mode, then RequestTransition
- For same-context transitions: use RequestTransition
- For cross-context transitions: use ModeCoordinator.EnterBattleMap/ReturnToOverworld

### Panel Layout Broken After Resize

**Symptom**: Panels misaligned after window resize.

**Causes**:
1. Using absolute pixel values instead of fractions
2. Not recalculating layout on resize
3. Cached backgrounds not invalidated

**Solution**:
- Use responsive constants (specs.PanelWidthStandard, etc.)
- Call PreCacheScrollContainerSizes after resize
- Use anchor helpers for relative positioning

### Performance Degradation

**Symptom**: FPS drops in UI-heavy modes.

**Causes**:
1. Using raw widgets instead of cached wrappers
2. Not pre-caching backgrounds
3. Querying ECS every frame
4. Invalidating cache every frame

**Solution**:
- Replace widget.List with CachedListWrapper
- Call PreCacheScrollContainerBackgrounds at startup
- Cache query results in mode state
- Only invalidate cache when game state changes

---

## Glossary

**Anchor**: A reference point for positioning widgets (top-left, center, bottom-right, etc.).

**BaseMode**: Common infrastructure inherited by all modes. Provides panel registry, input handling, and mode lifecycle.

**BattleMap Context**: Tactical gameplay context (exploration, combat, inventory).

**Cached Widget**: Widget wrapper with dirty-tracking to avoid redundant rendering.

**Context State**: UI-specific state for a context (BattleMapState, OverworldState). Separate from game state.

**Coordinator**: GameModeCoordinator manages two-context system and context switching.

**Descriptor**: Declarative specification for a panel (PanelDescriptor) or layout (PanelLayoutSpec).

**ECS**: Entity-Component-System architecture. Game state lives in ECS components.

**Functional Options**: Pattern for configurable builders (e.g., `builders.Size(...)`, `builders.Padding(...)`).

**GUIQueries**: Typed, cached query interface for accessing game state from UI.

**Hotkey**: Keyboard shortcut bound to an action via InputBinding.

**InputState**: Unified snapshot of mouse/keyboard state passed to mode HandleInput.

**Mode**: A self-contained screen/interface (exploration, combat, inventory, etc.). Implements UIMode interface.

**ModeManager**: Manages mode lifecycle and transitions within a single context.

**ModeTransition**: Struct used internally by UIModeManager when processing mode changes.

**NineSlice**: Image rendering technique for resizable panels with fixed corners.

**OnCreate**: Callback invoked after panel creation for custom initialization.

**Overworld Context**: Strategic gameplay context (overworld map, threats, ticks).

**Panel**: A UI region/window. Identified by PanelType and registered with PanelRegistry.

**PanelBuilders**: Fluent API for constructing panels with functional options.

**PanelRegistry**: Global registry of all panels in a mode. Provides creation and access.

**PanelResult**: Returned by BuildRegisteredPanel, contains Container, Type, TextLabel, and Custom map.

**PanelType**: String constant identifying a panel (e.g., "combat_turn_order").

**Responsive Sizing**: Layout using screen-fraction constants instead of absolute pixels.

**SquadInfo**: Aggregated squad data for UI display (cached in GUIQueries).

**UIContext**: Shared context passed to all modes (ECS manager, player data, screen dimensions, etc.).

**UIMode**: Interface implemented by all modes (Initialize, Enter, Exit, Update, Render, HandleInput, GetEbitenUI, GetModeName).

**UIModeManager**: Manages mode lifecycle and transitions within a context.

**Widget**: ebitenui UI element (button, text, list, container, etc.).

---

**End of Documentation**
