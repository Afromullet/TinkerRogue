# GUI Refactoring Recommendations

**Last Updated:** 2025-12-19
**Scope:** gui/ package and all subpackages
**Focus:** Maintainability, DRY principles, and design patterns

---

## Executive Summary

The GUI codebase shows good architectural foundations with `BaseMode`, widget factories, and standardized panels. However, there are significant opportunities to reduce repetition, improve maintainability, and strengthen design patterns. This document prioritizes recommendations by impact.

**Key Metrics:**
- 44 Go files across 6 packages
- 10+ mode implementations with similar patterns
- Estimated 30-40% code reduction possible through consolidation
- High code duplication in mode initialization (60-80% similar across modes)

---

## Priority 1: Critical - High Impact on Maintainability


### 1.3 Standardize Enter/Exit/Update Patterns

**Problem:** Inconsistent lifecycle implementations across modes:

```go
// Some modes do real work in Enter
func (sm *SquadManagementMode) Enter(fromMode core.UIMode) error {
    sm.allSquadIDs = sm.Queries.SquadCache.FindAllSquads()
    sm.currentSquadIndex = 0
    sm.refreshCurrentSquad()
    sm.updateNavigationButtons()
    return nil
}

// Some modes do nothing
func (em *ExplorationMode) Enter(fromMode core.UIMode) error {
    fmt.Println("Entering Exploration Mode")
    return nil
}

// Some modes have complex refresh logic
func (cm *CombatMode) Enter(fromMode core.UIMode) error {
    // Complex multi-step refresh
}
```

**Impact:**
- **Unclear contracts:** What SHOULD happen in Enter vs Update vs Render?
- **Bugs:** Easy to forget refresh logic, leading to stale UI
- **Performance:** Some modes refresh unnecessarily, some don't refresh when needed

**Recommendation:** Define clear lifecycle contracts and templates

```go
// gui/core/lifecycle.go - Document lifecycle contracts
/*
UIMode Lifecycle Contracts:

Initialize(ctx):
  - Called ONCE when mode is first registered
  - Build all UI components (panels, buttons, labels)
  - Set up event handlers
  - Create component managers (list components, detail components)
  - DO NOT query ECS data (it may not exist yet)
  - MUST call InitializeBase(ctx) first

Enter(fromMode):
  - Called EVERY TIME mode becomes active
  - Refresh data from ECS (query for current entities)
  - Update UI to reflect current game state
  - Reset transient state (selections, filters)
  - Consider WHAT CHANGED since last Exit (optimization)

Exit(toMode):
  - Called when leaving mode
  - Clean up selections/highlights
  - Save any pending changes
  - DO NOT destroy UI (mode may be re-entered)

Update(deltaTime):
  - Called every frame while active
  - Poll for real-time changes (combat turn progress, animations)
  - Update dynamic displays (timers, counters)
  - MINIMIZE work - keep logic in Enter/Exit when possible

HandleInput(inputState):
  - Process user input
  - Return true if input consumed (stops propagation)
  - Call HandleCommonInput first (ESC, hotkeys)

Render(screen):
  - Custom drawing ONLY (overlays, highlights, tile rendering)
  - ebitenui handles widget rendering automatically
  - Most modes should leave this empty
*/
```

**Standard Enter Pattern Template:**
```go
// gui/enterpattern.go
type EnterRefresher interface {
    RefreshData()    // Query fresh data from ECS
    RefreshUI()      // Update widgets with fresh data
    ResetState()     // Clear transient state (selections, etc.)
}

// Standard implementation
func StandardEnter(mode EnterRefresher, fromMode core.UIMode) error {
    mode.RefreshData()
    mode.RefreshUI()
    mode.ResetState()
    return nil
}

// Usage in modes:
func (sm *SquadManagementMode) Enter(fromMode core.UIMode) error {
    return gui.StandardEnter(sm, fromMode)
}

func (sm *SquadManagementMode) RefreshData() {
    sm.allSquadIDs = sm.Queries.SquadCache.FindAllSquads()
}

func (sm *SquadManagementMode) RefreshUI() {
    sm.refreshCurrentSquad()
    sm.updateNavigationButtons()
}

func (sm *SquadManagementMode) ResetState() {
    sm.currentSquadIndex = 0
}
```

**Benefits:**
- Clear contracts prevent bugs
- Template reduces boilerplate
- Easier to understand mode lifecycle
- Performance optimization opportunities (skip refresh when data unchanged)

---

## Priority 2: Important - Reduce Code Duplication

### 2.1 Extract Common Button Patterns

**Problem:** Button creation repeated 50+ times with minor variations:

```go
// Pattern 1: Mode transition buttons (appears ~20 times)
btn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text: "Squad Management (E)",
    OnClick: func() {
        if mode, exists := modeManager.GetMode("squad_management"); exists {
            modeManager.RequestTransition(mode, "Button clicked")
        }
    },
})

// Pattern 2: Context switch buttons (appears ~10 times)
btn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text: "Battle Map (ESC)",
    OnClick: func() {
        if ctx.ModeCoordinator != nil {
            ctx.ModeCoordinator.EnterBattleMap("exploration")
        }
    },
})

// Pattern 3: Dialog confirmation buttons (appears ~15 times)
confirmBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text: "Confirm",
    OnClick: func() {
        // Execute command
        cmd := squadcommands.NewXXXCommand(...)
        commandHistory.Execute(cmd)
        window.Close()
    },
})
```

**Recommendation:** Create button builders for common patterns

```go
// gui/buttonbuilders.go
package gui

// ModeTransitionButton creates a button that transitions to another mode
func ModeTransitionButton(modeManager *core.UIModeManager, text, targetMode string) *widget.Button {
    return widgets.CreateButtonWithConfig(widgets.ButtonConfig{
        Text: text,
        OnClick: func() {
            if mode, exists := modeManager.GetMode(targetMode); exists {
                modeManager.RequestTransition(mode, fmt.Sprintf("%s clicked", text))
            }
        },
    })
}

// ContextSwitchButton creates a button that switches game contexts
func ContextSwitchButton(coordinator *core.GameModeCoordinator, text, targetContext, targetMode string) *widget.Button {
    return widgets.CreateButtonWithConfig(widgets.ButtonConfig{
        Text: text,
        OnClick: func() {
            if coordinator != nil {
                switch targetContext {
                case "battlemap":
                    coordinator.EnterBattleMap(targetMode)
                case "overworld":
                    coordinator.ReturnToOverworld(targetMode)
                }
            }
        },
    })
}

// CommandButton creates a button that executes a command with confirmation
type CommandButtonConfig struct {
    Text            string
    ConfirmTitle    string
    ConfirmMessage  string
    CreateCommand   func() squadcommands.Command
    CommandHistory  *CommandHistory
    OnCancel        func()
    CloseWindow     *widget.Window // Optional
}

func CommandButton(config CommandButtonConfig) *widget.Button {
    return widgets.CreateButtonWithConfig(widgets.ButtonConfig{
        Text: config.Text,
        OnClick: func() {
            ShowConfirmationDialog(ConfirmationConfig{
                Title:   config.ConfirmTitle,
                Message: config.ConfirmMessage,
                OnConfirm: func() {
                    cmd := config.CreateCommand()
                    config.CommandHistory.Execute(cmd)
                    if config.CloseWindow != nil {
                        config.CloseWindow.Close()
                    }
                },
                OnCancel: config.OnCancel,
            })
        },
    })
}
```

**Usage:**
```go
// Before: 8 lines
squadMgmtBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
    Text: "Squad Management (E)",
    OnClick: func() {
        if mode, exists := modeManager.GetMode("squad_management"); exists {
            modeManager.RequestTransition(mode, "Button clicked")
        }
    },
})

// After: 1 line
squadMgmtBtn := gui.ModeTransitionButton(modeManager, "Squad Management (E)", "squad_management")
```

**Benefits:**
- Reduces button creation from 8 lines to 1-2 lines
- Eliminates ~200 lines of duplicated button code
- Consistent error handling and logging
- Easier to change button behavior globally

---

### 2.2 Consolidate Panel Building Patterns

**Problem:** Three different ways to build panels create confusion:

```go
// Method 1: CreateStandardPanel
panel := widgets.CreateStandardPanel(panelBuilders, "turn_order")

// Method 2: CreateStandardDetailPanel
panel, textarea := gui.CreateStandardDetailPanel(panelBuilders, layout, "inventory_detail", "")

// Method 3: CreatePanelWithConfig (custom)
panel := widgets.CreatePanelWithConfig(widgets.PanelConfig{
    MinWidth:  panelWidth,
    MinHeight: panelHeight,
    Layout:    widget.NewRowLayout(...),
})

// Method 4: PanelBuilders.BuildPanel with options
panel := panelBuilders.BuildPanel(
    widgets.TopCenter(),
    widgets.Size(0.5, 0.3),
    widgets.RowLayout(),
)
```

**Recommendation:** Standardize on builder pattern with clear use cases

```go
// widgets/panelbuilders.go - Enhanced with clear patterns

type PanelType int
const (
    PanelTypeSimple    PanelType = iota // Container only, no children
    PanelTypeDetail                      // Container + TextArea (for detail views)
    PanelTypeList                        // Container + List widget
    PanelTypeGrid                        // Container + Grid layout
    PanelTypeCustom                      // Full custom configuration
)

type PanelBuilderConfig struct {
    Type      PanelType
    SpecName  string // From StandardPanels (if empty, use custom config)

    // Detail panel options
    DetailText string // Initial text for detail panels

    // List panel options
    ListEntries []interface{}
    OnListSelect func(interface{})

    // Grid panel options
    GridRows int
    GridCols int
    OnCellClick func(row, col int)

    // Custom options (when SpecName is empty)
    Position    widgets.PanelOption
    Size        widgets.PanelOption
    Layout      widgets.PanelOption
    Padding     widgets.PanelOption
}

func (pb *PanelBuilders) BuildTypedPanel(config PanelBuilderConfig) (*widget.Container, interface{}) {
    var panel *widget.Container
    var content interface{} // Could be *widget.TextArea, *widget.List, etc.

    // Build base panel
    if config.SpecName != "" {
        panel = CreateStandardPanel(pb, config.SpecName)
    } else {
        panel = pb.BuildPanel(config.Position, config.Size, config.Layout, config.Padding)
    }

    // Add type-specific content
    switch config.Type {
    case PanelTypeDetail:
        textarea := createTextAreaForPanel(panel, pb.Layout, config.DetailText)
        panel.AddChild(textarea)
        content = textarea

    case PanelTypeList:
        list := createListForPanel(panel, config.ListEntries, config.OnListSelect)
        panel.AddChild(list)
        content = list

    case PanelTypeGrid:
        grid, cells := createGridForPanel(panel, config.GridRows, config.GridCols, config.OnCellClick)
        panel.AddChild(grid)
        content = cells

    case PanelTypeSimple, PanelTypeCustom:
        // No content, just return panel
    }

    return panel, content
}
```

**Usage:**
```go
// Before: Multiple different methods
panel1 := widgets.CreateStandardPanel(pb, "turn_order")
panel2, textarea := gui.CreateStandardDetailPanel(pb, layout, "inventory_detail", "")
panel3 := widgets.CreatePanelWithConfig(widgets.PanelConfig{...})

// After: Single consistent method
panel1, _ := pb.BuildTypedPanel(PanelBuilderConfig{
    Type: PanelTypeSimple,
    SpecName: "turn_order",
})

panel2, textarea := pb.BuildTypedPanel(PanelBuilderConfig{
    Type: PanelTypeDetail,
    SpecName: "inventory_detail",
    DetailText: "Select an item",
})

panel3, grid := pb.BuildTypedPanel(PanelBuilderConfig{
    Type: PanelTypeGrid,
    SpecName: "formation_grid",
    GridRows: 3,
    GridCols: 3,
    OnCellClick: handleCellClick,
})
```

**Benefits:**
- Single method for all panel creation
- Type-safe return values (know what content type you get)
- Reduces cognitive load (one pattern to remember)
- Eliminates helper function proliferation

---

### 2.3 Extract Dialog Creation Patterns

**Problem:** Confirmation dialogs manually built in 6+ places with near-identical code (60-80 lines each):

```go
// squadmanagementmode.go - Disband dialog
dialog := widgets.CreateConfirmationDialog(widgets.DialogConfig{
    Title: "Confirm Disband",
    Message: "Disband squad?",
    OnConfirm: func() {
        cmd := squadcommands.NewDisbandSquadCommand(...)
        commandHistory.Execute(cmd)
    },
    OnCancel: func() {
        setStatus("Cancelled")
    },
})
ui.AddWindow(dialog)

// squadmanagementmode.go - Merge dialog (90 lines!)
contentContainer := widget.NewContainer(...)
titleLabel := widgets.CreateLargeLabel("Merge Squads")
contentContainer.AddChild(titleLabel)
// ... 60 more lines of dialog building
window := widget.NewWindow(...)
ui.AddWindow(window)

// formationeditormode.go - Apply formation dialog (similar pattern)
// combatmode.go - Flee confirmation dialog (similar pattern)
```

**Recommendation:** Create dialog builder system

```go
// gui/dialogs/dialogbuilder.go
package dialogs

type DialogType int
const (
    DialogTypeConfirmation DialogType = iota
    DialogTypeSelection
    DialogTypeInput
    DialogTypeInfo
)

type DialogConfig struct {
    Type    DialogType
    Title   string
    Message string

    // Confirmation
    OnConfirm func()
    OnCancel  func()

    // Selection
    SelectionEntries []string
    OnSelect         func(string)

    // Input
    InputPlaceholder string
    OnInput          func(string)
}

type DialogBuilder struct {
    ui     *ebitenui.UI
    layout *widgets.LayoutConfig
}

func NewDialogBuilder(ui *ebitenui.UI, layout *widgets.LayoutConfig) *DialogBuilder {
    return &DialogBuilder{ui, layout}
}

func (db *DialogBuilder) Show(config DialogConfig) {
    var window *widget.Window

    switch config.Type {
    case DialogTypeConfirmation:
        window = db.buildConfirmationDialog(config)
    case DialogTypeSelection:
        window = db.buildSelectionDialog(config)
    case DialogTypeInput:
        window = db.buildInputDialog(config)
    case DialogTypeInfo:
        window = db.buildInfoDialog(config)
    }

    db.ui.AddWindow(window)
}

func (db *DialogBuilder) buildConfirmationDialog(config DialogConfig) *widget.Window {
    return widgets.CreateConfirmationDialog(widgets.DialogConfig{
        Title:     config.Title,
        Message:   config.Message,
        OnConfirm: config.OnConfirm,
        OnCancel:  config.OnCancel,
    })
}

func (db *DialogBuilder) buildSelectionDialog(config DialogConfig) *widget.Window {
    // Build selection dialog with list
    container := widget.NewContainer(...)

    titleLabel := widgets.CreateLargeLabel(config.Title)
    container.AddChild(titleLabel)

    if config.Message != "" {
        messageLabel := widgets.CreateSmallLabel(config.Message)
        container.AddChild(messageLabel)
    }

    list := widgets.CreateSimpleStringList(widgets.SimpleStringListConfig{
        Entries: config.SelectionEntries,
        // ...
    })
    container.AddChild(list)

    buttonContainer := widget.NewContainer(...)
    selectBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
        Text: "Select",
        OnClick: func() {
            if selected := list.SelectedEntry(); selected != nil {
                config.OnSelect(selected.(string))
            }
        },
    })
    buttonContainer.AddChild(selectBtn)

    cancelBtn := widgets.CreateButtonWithConfig(widgets.ButtonConfig{
        Text: "Cancel",
        OnClick: func() {
            if config.OnCancel != nil {
                config.OnCancel()
            }
        },
    })
    buttonContainer.AddChild(cancelBtn)

    container.AddChild(buttonContainer)

    return widget.NewWindow(
        widget.WindowOpts.Contents(container),
        widget.WindowOpts.Modal(),
    )
}
```

**Usage:**
```go
// Before: 90 lines for merge squad dialog
contentContainer := widget.NewContainer(...)
// ... 80 lines of manual dialog construction
window := widget.NewWindow(...)
ui.AddWindow(window)

// After: 10 lines using builder
dialogBuilder := dialogs.NewDialogBuilder(ui, layout)
dialogBuilder.Show(dialogs.DialogConfig{
    Type:    dialogs.DialogTypeSelection,
    Title:   "Merge Squads",
    Message: fmt.Sprintf("Select target squad to merge into:"),
    SelectionEntries: otherSquadNames,
    OnSelect: func(selected string) {
        // Handle merge
    },
    OnCancel: func() { setStatus("Cancelled") },
})
```

**Benefits:**
- Reduces dialog creation from 60-90 lines to 5-15 lines
- Eliminates ~300 lines of duplicated dialog code
- Consistent dialog styling and behavior
- Easier to add new dialog types

---

## Priority 3: Design Improvements

### 3.1 Separate UI State from Game State

**Problem:** Some modes blur the line between UI state and game state:

```go
// BattleMapState in gui/core/contextstate.go
type BattleMapState struct {
    SelectedSquadID  ecs.EntityID // UI state - which squad is selected in UI
    IsAttackMode     bool          // UI state - UI interaction mode
    IsMoveMode       bool          // UI state - UI interaction mode
    // ... mixed with game state?
}
```

**Recommendation:** Clear separation with naming conventions

```go
// gui/core/uistate.go - ONLY UI-specific state
type ModeUIState struct {
    // Selection state (what's highlighted/selected in UI)
    SelectedEntityID ecs.EntityID
    HoveredEntityID  ecs.EntityID

    // Interaction state (what mode the UI is in)
    InteractionMode InteractionMode // enum: Browse, Attack, Move, Target

    // Display state (what's visible/filtered)
    CurrentFilter string
    CurrentPage   int

    // Transient state (undo/redo, temporary highlights)
    UndoStack []UIAction
    RedoStack []UIAction
}

// Game state stays in ECS (combat/combatstate.go, squads/squadstate.go, etc.)
// UI queries game state but doesn't duplicate it
```

**Benefits:**
- Clear ownership of state
- Easier testing (UI state vs game state)
- Reduces bugs from state synchronization

---

### 3.2 Introduce Component Registration System

**Problem:** Modes manually create and manage many update components:

```go
// combatmode.go
type CombatMode struct {
    // ... many component fields
    squadListComponent   *guicomponents.SquadListComponent
    squadDetailComponent *guicomponents.DetailPanelComponent
    factionInfoComponent *guicomponents.DetailPanelComponent
    turnOrderComponent   *guicomponents.TextDisplayComponent
}

func (cm *CombatMode) Update(deltaTime float64) error {
    // Manually update each component
    cm.squadListComponent.Update()
    cm.squadDetailComponent.Update()
    // ... etc
}
```

**Recommendation:** Component registry pattern

```go
// gui/core/componentregistry.go
type UIComponent interface {
    Update(deltaTime float64)
    Refresh() // Force refresh from ECS
}

type ComponentRegistry struct {
    components []UIComponent
}

func (cr *ComponentRegistry) Register(comp UIComponent) {
    cr.components = append(cr.components, comp)
}

func (cr *ComponentRegistry) UpdateAll(deltaTime float64) {
    for _, comp := range cr.components {
        comp.Update(deltaTime)
    }
}

func (cr *ComponentRegistry) RefreshAll() {
    for _, comp := range cr.components {
        comp.Refresh()
    }
}

// Add to BaseMode
type BaseMode struct {
    // ... existing fields
    Components *ComponentRegistry
}

// Usage in modes
func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    cm.InitializeBase(ctx)

    // Create components
    squadList := guicomponents.NewSquadListComponent(...)
    squadDetail := guicomponents.NewDetailPanelComponent(...)

    // Register for automatic updates
    cm.Components.Register(squadList)
    cm.Components.Register(squadDetail)

    return nil
}

func (cm *CombatMode) Update(deltaTime float64) error {
    cm.Components.UpdateAll(deltaTime) // One line instead of many
    return nil
}

func (cm *CombatMode) Enter(fromMode core.UIMode) error {
    cm.Components.RefreshAll() // Refresh all components
    return nil
}
```

**Benefits:**
- Reduces Update() boilerplate
- Easier to add new components
- Centralized lifecycle management
- Consistent refresh behavior

---

### 3.3 Improve Error Handling Patterns

**Problem:** Inconsistent error handling in mode transitions and initialization:

```go
// Some modes check exists
if mode, exists := modeManager.GetMode("target"); exists {
    modeManager.RequestTransition(mode, "reason")
}

// Some modes don't check
mode, _ := modeManager.GetMode("target")
modeManager.RequestTransition(mode, "reason") // Panic if not found!

// Some modes log errors
if err := mode.Initialize(ctx); err != nil {
    fmt.Printf("Error: %v\n", err)
    return err
}

// Some modes ignore errors
mode.Initialize(ctx)
```

**Recommendation:** Consistent error handling utilities

```go
// gui/core/errors.go
type ModeError struct {
    ModeName string
    Op       string // "initialize", "enter", "exit", "transition"
    Err      error
}

func (e *ModeError) Error() string {
    return fmt.Sprintf("mode %s: %s failed: %v", e.ModeName, e.Op, e.Err)
}

// Safe mode transition that logs but doesn't panic
func SafeTransition(mm *UIModeManager, targetModeName, reason string) error {
    mode, exists := mm.GetMode(targetModeName)
    if !exists {
        err := &ModeError{
            ModeName: targetModeName,
            Op:       "transition",
            Err:      fmt.Errorf("mode not registered"),
        }
        fmt.Printf("ERROR: %v\n", err)
        return err
    }

    mm.RequestTransition(mode, reason)
    return nil
}

// Usage
if err := core.SafeTransition(modeManager, "inventory", "I key pressed"); err != nil {
    // Error logged, continue gracefully
}
```

**Benefits:**
- Prevents crashes from unregistered modes
- Consistent error logging
- Easier debugging

---

## Priority 4: Code Organization

### 4.1 Package Structure Cleanup

**Current Structure:**
```
gui/
├── basemode.go               # Base mode implementation
├── commandhistory.go         # Command pattern
├── modehelpers.go           # Mode utilities
├── ui_helpers.go            # UI utilities
├── core/                    # Core abstractions
│   ├── contextstate.go
│   ├── gamemodecoordinator.go
│   ├── modemanager.go
│   └── uimode.go
├── guicombat/              # Combat-specific UI (7 files)
├── guicomponents/          # Reusable components (4 files)
├── guimodes/               # Standard modes (4 files)
├── guiresources/           # Resources (2 files)
├── guisquads/              # Squad-specific UI (7 files)
└── widgets/                # Widget factories (11 files)
```

**Problems:**
- `gui/` root has mix of base classes and helpers (4 files)
- `widgets/` has 11 files with overlapping concerns
- Unclear where new code should go

**Recommendation:** Reorganize for clarity

```
gui/
├── core/                    # Core framework (no changes)
│   ├── lifecycle.go         # NEW: Lifecycle contracts
│   ├── componentregistry.go # NEW: Component management
│   ├── errors.go           # NEW: Error handling
│   └── ... existing files
│
├── foundation/             # NEW: Base classes and patterns
│   ├── basemode.go         # MOVE from gui/
│   ├── modebuilder.go      # NEW: Mode builder pattern
│   ├── commandhistory.go   # MOVE from gui/
│   └── enterpattern.go     # NEW: Standard Enter pattern
│
├── builders/               # NEW: All creation/building code
│   ├── buttonbuilders.go   # NEW: Button builders
│   ├── panelbuilders.go    # MOVE from widgets/
│   ├── widgetbuilders.go   # CONSOLIDATE from widgets/createwidgets.go
│   └── listbuilders.go     # CONSOLIDATE from widgets/list_helpers.go
│
├── dialogs/                # NEW: Dialog system
│   └── dialogbuilder.go    # NEW: Dialog builder
│
├── helpers/                # Utility functions
│   ├── anchor.go          # MOVE from ui_helpers.go (anchor helpers)
│   ├── padding.go         # MOVE from ui_helpers.go (padding helpers)
│   └── positioning.go     # MOVE from modehelpers.go
│
├── components/             # RENAME from guicomponents/
│   └── ... existing files
│
├── resources/              # RENAME from guiresources/
│   └── ... existing files
│
├── modes/                  # CONSOLIDATE all mode packages
│   ├── combat/            # RENAME from guicombat/
│   ├── exploration/       # SPLIT from guimodes/
│   ├── info/              # SPLIT from guimodes/
│   ├── inventory/         # SPLIT from guimodes/
│   └── squads/            # RENAME from guisquads/
│
└── specs/                  # Configuration specifications
    ├── panelspecs.go      # MOVE from widgets/panelconfig.go
    └── layoutspecs.go     # MOVE from widgets/layout_constants.go
```

**Benefits:**
- Clear package purpose (foundation, builders, helpers, modes)
- Easier to find code ("Where do buttons get created?" → builders/)
- Scalable (new modes go in modes/, new builders in builders/)
- Reduces cognitive load

---

### 4.2 Reduce widgets/ Package Complexity

**Current `widgets/` Package (11 files):**
- button_factory.go
- panel_factory.go
- createwidgets.go
- list_helpers.go
- dialogs.go
- layout.go
- layout_constants.go
- panelconfig.go
- panels.go
- cachedpanels.go

**Problem:** Hard to know which file has which function

**Recommendation:** Consolidate to 4 focused files

```
builders/ (NEW package location)
├── widgets.go           # CONSOLIDATE: createwidgets.go + button_factory.go
├── panels.go            # CONSOLIDATE: panel_factory.go + panels.go + cachedpanels.go
├── lists.go             # RENAME: list_helpers.go
└── dialogs.go           # KEEP as-is

specs/ (NEW package location)
├── panelspecs.go        # MOVE: panelconfig.go
└── layout.go            # CONSOLIDATE: layout.go + layout_constants.go
```

**Benefits:**
- Reduces 11 files to 6 files
- Clear naming: "I need a list" → lists.go
- Less package jumping

---

## Implementation Strategy

### Phase 1: Foundation (Weeks 1-2)
1. Create `ModeBuilder` pattern
2. Convert 2-3 simple modes to use ModeBuilder (inventory, exploration, info)
3. Validate pattern works
4. Document usage

### Phase 2: Consolidation (Weeks 3-4)
5. Consolidate factory patterns into UIComponentFactory
6. Extract button builders (ModeTransitionButton, etc.)
7. Create DialogBuilder system
8. Update all modes to use new patterns

### Phase 3: Cleanup (Week 5)
9. Reorganize package structure
10. Move files to new locations
11. Update imports
12. Remove deprecated files

### Phase 4: Polish (Week 6)
13. Add lifecycle documentation
14. Add error handling utilities
15. Add component registry
16. Final cleanup and testing

---

## Testing Strategy

Each refactoring should be tested:

1. **Visual Testing:** Launch game, navigate through modes, verify UI works
2. **Regression Testing:** Verify existing functionality unchanged
3. **Before/After Comparison:** Compare code line counts, file counts
4. **Developer Testing:** Have another developer try using new patterns

---

## Success Metrics

**Code Reduction:**
- Target: 30-40% reduction in GUI code (600-800 lines removed)
- Fewer files: 44 → ~35 files
- Smaller modes: 100-line Initialize → 20-line Initialize

**Maintainability:**
- New mode creation time: 2 hours → 30 minutes
- Common change propagation: 10 files → 1-2 files
- Cognitive load: "Which pattern?" → "Use ModeBuilder"

**Quality:**
- Fewer null pointer bugs (safe transitions)
- Consistent error handling
- Clearer package boundaries

---

## Additional Resources

- See `docs/ecs_best_practices.md` for ECS patterns
- See `CLAUDE.md` for general development guidelines
- Reference `gui/foundation/basemode.go` for mode lifecycle

---

## Questions & Discussion

**Q: Will this break existing code?**
A: Implement gradually. Keep old patterns working during transition. Deprecate after all modes converted.

**Q: How long will this take?**
A: 4-6 weeks for full implementation. But benefits appear immediately as each phase completes.

**Q: Should we do all of this?**
A: Priority 1 (Critical) recommendations should definitely be done. Priority 2-4 can be evaluated based on time/benefit.

**Q: What about backward compatibility?**
A: Old widget creators can be marked deprecated but kept for gradual migration. Remove after all usages updated.

---

## Conclusion

The GUI codebase has good bones but suffers from pattern proliferation and code duplication. The recommendations in this document focus on:

1. **Reducing Repetition:** ModeBuilder, button builders, dialog builders
2. **Improving Maintainability:** Clear lifecycle, consolidated factories, organized packages
3. **Following Principles:** DRY, Single Responsibility, Clear Abstractions

Implementing these recommendations will make the codebase easier to understand, extend, and maintain while reducing bug surface area and improving developer productivity.
