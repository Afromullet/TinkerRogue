# Refactoring Analysis: GUI Package
Generated: 2025-11-05
Target: GUI Package (4563 LOC across 13 files)

## EXECUTIVE SUMMARY

### Target Analysis
- **Scope**: Complete GUI package for Go-based roguelike game
- **Current State**: 4563 LOC across 13 files, 8 UI modes, heavy duplication
- **Primary Issues**:
  1. ~60% code duplication across 8 mode implementations
  2. Missing button factory pattern (mentioned in CLAUDE.md as 10% complete)
  3. Complex widget construction ceremonies (10+ option calls per widget)
  4. Global mutable state in resource management
  5. CombatMode complexity (1119 LOC single file)
- **Recommended Direction**: Config-driven widget system + mode composition

### Quick Wins vs Strategic Refactoring
- **Immediate Improvements** (1-2 days):
  - ButtonConfig factory pattern
  - TextAreaConfig expansion to all common widgets
  - Extract shared panel builders to composition functions

- **Medium-Term Goals** (3-5 days):
  - Widget builder pattern for lists, containers, panels
  - Resource dependency injection
  - Mode base struct for shared logic

- **Long-Term Architecture** (1-2 weeks):
  - Full config-driven UI system
  - CombatMode decomposition
  - Layout DSL or declarative system

### Consensus Findings
- **High-Impact Quick Win**: ButtonConfig pattern (already started with TextAreaConfig)
- **Biggest Pain Point**: Repeated buildXXXPanel() methods across all 8 modes
- **Root Cause**: No abstraction layer between ebitenui and game code
- **Strategic Direction**: Incremental config-driven approach (proven by TextAreaConfig success)

---

## FINAL SYNTHESIZED APPROACHES

### Approach 1: Incremental Config-Driven Widget System

**Strategic Focus**: Build on existing TextAreaConfig pattern to eliminate widget construction duplication

**Problem Statement**:
Every mode builds UI widgets independently using verbose ebitenui option chains. Creating a button requires 5-10 lines of widget.ButtonOpts calls. Creating a list requires 10+ widget.ListOpts calls. This is repeated 40+ times across 8 modes with minimal variation.

The TextAreaConfig pattern (already present in createwidgets.go lines 98-125) proves this approach works.

**Solution Overview**:
Extend the config pattern to all common widgets (Button, List, Container, Panel) and create builder functions that accept configs. This reduces widget creation from 10+ lines to 1-3 lines while maintaining full customization.

**Code Example**:

*Before (from explorationmode.go lines 131-146):*
```go
throwableBtn := CreateButton("Throwables")
throwableBtn.Configure(
    widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
        if invMode, exists := em.modeManager.GetMode("inventory"); exists {
            if inventoryMode, ok := invMode.(*InventoryMode); ok {
                inventoryMode.SetInitialFilter("Throwables")
            }
            em.modeManager.RequestTransition(invMode, "Open Throwables")
        }
    }),
)
em.quickInventory.AddChild(throwableBtn)
```

*After:*
```go
throwableBtn := CreateButtonWithConfig(ButtonConfig{
    Text: "Throwables",
    OnClick: func() {
        if invMode, exists := em.modeManager.GetMode("inventory"); exists {
            if inventoryMode, ok := invMode.(*InventoryMode); ok {
                inventoryMode.SetInitialFilter("Throwables")
            }
            em.modeManager.RequestTransition(invMode, "Open Throwables")
        }
    },
})
em.quickInventory.AddChild(throwableBtn)
```

**Full Implementation (add to createwidgets.go):**

```go
// ButtonConfig provides declarative button configuration
type ButtonConfig struct {
    Text      string
    MinWidth  int
    MinHeight int
    FontFace  font.Face
    TextColor *widget.ButtonTextColor
    Image     *widget.ButtonImage
    Padding   widget.Insets
    OnClick   func() // Simplified callback - no args needed in most cases
    LayoutData interface{} // For positioning
}

// CreateButtonWithConfig creates a button from config
func CreateButtonWithConfig(config ButtonConfig) *widget.Button {
    // Apply defaults
    if config.MinWidth == 0 {
        config.MinWidth = 100
    }
    if config.MinHeight == 0 {
        config.MinHeight = 100
    }
    if config.FontFace == nil {
        config.FontFace = largeFace
    }
    if config.TextColor == nil {
        config.TextColor = &widget.ButtonTextColor{
            Idle: color.NRGBA{0xdf, 0xf4, 0xff, 0xff},
        }
    }
    if config.Image == nil {
        config.Image = buttonImage
    }
    if config.Padding.Left == 0 {
        config.Padding = widget.Insets{Left: 30, Right: 30, Top: 30, Bottom: 30}
    }

    opts := []widget.ButtonOpt{
        widget.ButtonOpts.WidgetOpts(
            widget.WidgetOpts.MinSize(config.MinWidth, config.MinHeight),
        ),
        widget.ButtonOpts.Image(config.Image),
        widget.ButtonOpts.Text(config.Text, config.FontFace, config.TextColor),
        widget.ButtonOpts.TextPadding(config.Padding),
    }

    // Add layout data if provided
    if config.LayoutData != nil {
        opts = append(opts, widget.ButtonOpts.WidgetOpts(
            widget.WidgetOpts.LayoutData(config.LayoutData),
        ))
    }

    // Add click handler if provided
    if config.OnClick != nil {
        opts = append(opts, widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
            config.OnClick()
        }))
    }

    return widget.NewButton(opts...)
}

// ListConfig provides declarative list configuration
type ListConfig struct {
    Entries         []interface{}
    EntryLabelFunc  func(interface{}) string
    OnEntrySelected func(interface{}) // Simplified callback
    MinWidth        int
    MinHeight       int
    LayoutData      interface{}
}

// CreateListWithConfig creates a list from config
func CreateListWithConfig(config ListConfig) *widget.List {
    // Apply defaults
    if config.MinWidth == 0 {
        config.MinWidth = 150
    }
    if config.MinHeight == 0 {
        config.MinHeight = 300
    }
    if config.EntryLabelFunc == nil {
        config.EntryLabelFunc = func(e interface{}) string {
            return fmt.Sprintf("%v", e)
        }
    }

    opts := []widget.ListOpt{
        widget.ListOpts.Entries(config.Entries),
        widget.ListOpts.EntryLabelFunc(config.EntryLabelFunc),
        widget.ListOpts.ContainerOpts(
            widget.ContainerOpts.WidgetOpts(
                widget.WidgetOpts.MinSize(config.MinWidth, config.MinHeight),
            ),
        ),
        widget.ListOpts.ScrollContainerOpts(
            widget.ScrollContainerOpts.Image(ListRes.image),
        ),
        widget.ListOpts.SliderOpts(
            widget.SliderOpts.Images(ListRes.track, ListRes.handle),
            widget.SliderOpts.MinHandleSize(ListRes.handleSize),
            widget.SliderOpts.TrackPadding(ListRes.trackPadding),
        ),
        widget.ListOpts.EntryColor(ListRes.entry),
        widget.ListOpts.EntryFontFace(ListRes.face),
    }

    // Add layout data if provided
    if config.LayoutData != nil {
        opts = append(opts, widget.ListOpts.ContainerOpts(
            widget.ContainerOpts.WidgetOpts(
                widget.WidgetOpts.LayoutData(config.LayoutData),
            ),
        ))
    }

    list := widget.NewList(opts...)

    // Add selection handler if provided
    if config.OnEntrySelected != nil {
        list.EntrySelectedEvent.AddHandler(func(args interface{}) {
            a := args.(*widget.ListEntrySelectedEventArgs)
            config.OnEntrySelected(a.Entry)
        })
    }

    return list
}

// PanelConfig provides declarative panel configuration
type PanelConfig struct {
    Title      string
    MinWidth   int
    MinHeight  int
    Background *image.NineSlice
    Padding    widget.Insets
    Layout     widget.Layout // Row, Grid, Anchor, etc.
    LayoutData interface{}
}

// CreatePanelWithConfig creates a container panel from config
func CreatePanelWithConfig(config PanelConfig) *widget.Container {
    // Apply defaults
    if config.Background == nil {
        config.Background = PanelRes.image
    }
    if config.Padding.Left == 0 {
        config.Padding = widget.Insets{Left: 15, Right: 15, Top: 15, Bottom: 15}
    }
    if config.Layout == nil {
        config.Layout = widget.NewAnchorLayout()
    }

    opts := []widget.ContainerOpt{
        widget.ContainerOpts.BackgroundImage(config.Background),
        widget.ContainerOpts.Layout(config.Layout),
    }

    if config.MinWidth > 0 || config.MinHeight > 0 {
        opts = append(opts, widget.ContainerOpts.WidgetOpts(
            widget.WidgetOpts.MinSize(config.MinWidth, config.MinHeight),
        ))
    }

    if config.LayoutData != nil {
        opts = append(opts, widget.ContainerOpts.WidgetOpts(
            widget.WidgetOpts.LayoutData(config.LayoutData),
        ))
    }

    container := widget.NewContainer(opts...)

    // Add title if provided
    if config.Title != "" {
        titleLabel := widget.NewText(
            widget.TextOpts.Text(config.Title, LargeFace, color.White),
        )
        container.AddChild(titleLabel)
    }

    return container
}
```

**Key Changes**:
- Add `ButtonConfig`, `ListConfig`, `PanelConfig` structs
- Add builder functions: `CreateButtonWithConfig`, `CreateListWithConfig`, `CreatePanelWithConfig`
- Simplify callbacks to `func()` instead of `func(*WidgetEventArgs)` where args not needed
- Automatic defaults for common values (font faces, colors, images, sizes)
- All existing functionality preserved through optional fields

**Implementation Strategy**:
1. **Phase 1 (2-3 hours)**: Add config structs and builders to `createwidgets.go`
2. **Phase 2 (4-6 hours)**: Refactor ExplorationMode and InventoryMode to use new configs
3. **Phase 3 (6-8 hours)**: Refactor remaining modes (Combat, Squad Management, Squad Builder, etc.)
4. **Phase 4 (2 hours)**: Remove old CreateButton function once all modes migrated
5. **Testing**: Verify each mode still functions identically after migration

**Advantages**:
- **Massive LOC reduction**: ~600-800 lines removed (estimated 15-20% of GUI package)
- **Improved readability**: Widget creation intent clear at a glance
- **Easier testing**: Configs can be validated without full UI setup
- **Consistent defaults**: Font faces, colors, sizes standardized across all modes
- **Incremental adoption**: Can migrate mode-by-mode without breaking changes
- **Proven pattern**: TextAreaConfig already exists and works well

**Drawbacks & Risks**:
- **Migration effort**: Must update ~40+ button creations, ~15+ list creations across 8 modes
  - *Mitigation*: Migrate one mode at a time, test thoroughly before next mode
- **Flexibility loss**: Some complex widgets might need custom options beyond configs
  - *Mitigation*: Keep option variadic parameters for advanced customization
- **Learning curve**: Developers must learn new config structs
  - *Mitigation*: Config structs are self-documenting with clear field names

**Effort Estimate**:
- **Time**: 14-19 hours (2-3 days with testing)
- **Complexity**: Low-Medium
- **Risk**: Low
- **Files Impacted**: 10 files (createwidgets.go + all 8 modes + infoUI.go)

**Critical Assessment**:
This is the highest-value refactoring. It eliminates the most duplication (widget construction ceremonies repeated 40+ times) with the least risk. The TextAreaConfig pattern proves this approach works. Migration can be incremental, reducing risk further. **Recommended as primary refactoring.**

---

### Approach 2: Mode Composition with Shared Panel Builders

**Strategic Focus**: Extract repeated buildXXXPanel() methods into reusable composition functions

**Problem Statement**:
Every mode has 3-6 buildXXXPanel() methods (buildStatsPanel, buildMessageLog, buildCloseButton, etc.) with 80-90% identical code. Only minor variations in positioning, sizing, and content. This results in ~2000 lines of duplicated panel building logic across 8 modes.

Example: Every mode has a "Close Button" at bottom-center with identical positioning logic, just different transition targets.

**Solution Overview**:
Create a set of high-level panel builder functions that encapsulate common UI patterns. Modes call these builders instead of repeating the full construction logic. This follows the composition-over-inheritance principle.

**Code Example**:

*Before (repeated in 6 modes):*
```go
func (em *ExplorationMode) buildCloseButton() {
    buttonContainer := widget.NewContainer(
        widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
    )

    closeBtn := CreateButton("Close (ESC)")
    closeBtn.Configure(
        widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
            if exploreMode, exists := em.modeManager.GetMode("exploration"); exists {
                em.modeManager.RequestTransition(exploreMode, "Close")
            }
        }),
    )

    buttonContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
        HorizontalPosition: widget.AnchorLayoutPositionCenter,
        VerticalPosition:   widget.AnchorLayoutPositionEnd,
        Padding: widget.Insets{
            Bottom: int(float64(em.layout.ScreenHeight) * 0.08),
        },
    }

    buttonContainer.AddChild(closeBtn)
    em.rootContainer.AddChild(buttonContainer)
}
```

*After (add to new file: gui/panels.go):*
```go
// PanelBuilders provides high-level UI composition functions
type PanelBuilders struct {
    layout      *LayoutConfig
    modeManager *UIModeManager
}

// BuildCloseButton creates a bottom-center close button that transitions to exploration mode
func (pb *PanelBuilders) BuildCloseButton(targetModeName string, buttonText string) *widget.Container {
    if buttonText == "" {
        buttonText = "Close (ESC)"
    }

    buttonContainer := widget.NewContainer(
        widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
    )

    closeBtn := CreateButton(buttonText)
    closeBtn.Configure(
        widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
            if targetMode, exists := pb.modeManager.GetMode(targetModeName); exists {
                pb.modeManager.RequestTransition(targetMode, "Close button pressed")
            }
        }),
    )

    buttonContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
        HorizontalPosition: widget.AnchorLayoutPositionCenter,
        VerticalPosition:   widget.AnchorLayoutPositionEnd,
        Padding: widget.Insets{
            Bottom: int(float64(pb.layout.ScreenHeight) * 0.08),
        },
    }

    buttonContainer.AddChild(closeBtn)
    return buttonContainer
}

// BuildStatsPanel creates a top-right stats panel
func (pb *PanelBuilders) BuildStatsPanel(content string) (*widget.Container, *widget.TextArea) {
    x, y, width, height := pb.layout.TopRightPanel()

    panel := widget.NewContainer(
        widget.ContainerOpts.BackgroundImage(PanelRes.image),
        widget.ContainerOpts.Layout(widget.NewAnchorLayout(
            widget.AnchorLayoutOpts.Padding(widget.Insets{
                Left: 10, Right: 10, Top: 10, Bottom: 10,
            }),
        )),
        widget.ContainerOpts.WidgetOpts(
            widget.WidgetOpts.MinSize(width, height),
        ),
    )

    textArea := CreateTextAreaWithConfig(TextAreaConfig{
        MinWidth:  width - 20,
        MinHeight: height - 20,
        FontColor: color.White,
    })
    textArea.SetText(content)

    panel.AddChild(textArea)
    SetContainerLocation(panel, x, y)

    return panel, textArea
}

// BuildMessageLog creates a bottom-right message log panel
func (pb *PanelBuilders) BuildMessageLog() (*widget.Container, *widget.TextArea) {
    x, y, width, height := pb.layout.BottomRightPanel()

    logConfig := TextAreaConfig{
        MinWidth:  width - 20,
        MinHeight: height - 20,
        FontColor: color.White,
    }
    messageLog := CreateTextAreaWithConfig(logConfig)

    logContainer := widget.NewContainer(
        widget.ContainerOpts.BackgroundImage(PanelRes.image),
        widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
        widget.ContainerOpts.WidgetOpts(
            widget.WidgetOpts.MinSize(width, height),
        ),
    )
    logContainer.AddChild(messageLog)

    SetContainerLocation(logContainer, x, y)

    return logContainer, messageLog
}

// BuildActionButtons creates a bottom-center button row
func (pb *PanelBuilders) BuildActionButtons(buttons []*widget.Button) *widget.Container {
    x, y := pb.layout.BottomCenterButtons()

    buttonContainer := widget.NewContainer(
        widget.ContainerOpts.Layout(widget.NewRowLayout(
            widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
            widget.RowLayoutOpts.Spacing(10),
            widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10}),
        )),
    )

    for _, btn := range buttons {
        buttonContainer.AddChild(btn)
    }

    SetContainerLocation(buttonContainer, x, y)
    return buttonContainer
}

// BuildSquadListPanel creates a left-side squad list with selection
func (pb *PanelBuilders) BuildSquadListPanel(squadNames []string, onSelect func(string)) (*widget.Container, *widget.List) {
    width := int(float64(pb.layout.ScreenWidth) * 0.15)
    height := int(float64(pb.layout.ScreenHeight) * 0.5)

    panel := widget.NewContainer(
        widget.ContainerOpts.BackgroundImage(PanelRes.image),
        widget.ContainerOpts.Layout(widget.NewRowLayout(
            widget.RowLayoutOpts.Direction(widget.DirectionVertical),
            widget.RowLayoutOpts.Spacing(5),
            widget.RowLayoutOpts.Padding(widget.Insets{Left: 5, Right: 5, Top: 10, Bottom: 10}),
        )),
        widget.ContainerOpts.WidgetOpts(
            widget.WidgetOpts.MinSize(width, height),
            widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
                HorizontalPosition: widget.AnchorLayoutPositionStart,
                VerticalPosition:   widget.AnchorLayoutPositionCenter,
                Padding: widget.Insets{
                    Left: int(float64(pb.layout.ScreenWidth) * 0.01),
                },
            }),
        ),
    )

    listLabel := widget.NewText(
        widget.TextOpts.Text("Squads:", SmallFace, color.White),
    )
    panel.AddChild(listLabel)

    // Convert squad names to entries
    entries := make([]interface{}, len(squadNames))
    for i, name := range squadNames {
        entries[i] = name
    }

    squadList := widget.NewList(
        widget.ListOpts.Entries(entries),
        widget.ListOpts.EntryLabelFunc(func(e interface{}) string {
            return e.(string)
        }),
        widget.ListOpts.ScrollContainerOpts(
            widget.ScrollContainerOpts.Image(ListRes.image),
        ),
        widget.ListOpts.SliderOpts(
            widget.SliderOpts.Images(ListRes.track, ListRes.handle),
        ),
        widget.ListOpts.EntryColor(ListRes.entry),
        widget.ListOpts.EntryFontFace(ListRes.face),
    )

    if onSelect != nil {
        squadList.EntrySelectedEvent.AddHandler(func(args interface{}) {
            a := args.(*widget.ListEntrySelectedEventArgs)
            if name, ok := a.Entry.(string); ok {
                onSelect(name)
            }
        })
    }

    panel.AddChild(squadList)
    return panel, squadList
}
```

*Usage in modes (e.g., ExplorationMode):*
```go1
func (em *ExplorationMode) Initialize(ctx *UIContext) error {
    em.context = ctx
    em.layout = NewLayoutConfig(ctx)
    em.panelBuilders = &PanelBuilders{layout: em.layout, modeManager: em.modeManager}

    em.ui = &ebitenui.UI{}
    em.rootContainer = widget.NewContainer()
    em.ui.Container = em.rootContainer

    // Build panels using composition
    statsPanel, em.statsTextArea := em.panelBuilders.BuildStatsPanel(
        em.context.PlayerData.PlayerAttributes().DisplayString(),
    )
    em.rootContainer.AddChild(statsPanel)

    logPanel, em.messageLog := em.panelBuilders.BuildMessageLog()
    em.rootContainer.AddChild(logPanel)

    closeBtn := em.panelBuilders.BuildCloseButton("exploration", "")
    em.rootContainer.AddChild(closeBtn)

    return nil
}
```

**Key Changes**:
- Create `PanelBuilders` struct with layout and mode manager dependencies
- Extract 6-8 common panel patterns: close button, stats panel, message log, action buttons, squad list, etc.
- Each builder encapsulates full construction logic including positioning
- Modes call builders instead of repeating code
- Return both container and child widgets for further customization if needed

**Implementation Strategy**:
1. **Phase 1 (3-4 hours)**: Create `panels.go` with 6-8 common panel builders
2. **Phase 2 (2 hours)**: Refactor ExplorationMode to use panel builders
3. **Phase 3 (3-4 hours)**: Refactor CombatMode to use panel builders
4. **Phase 4 (4-6 hours)**: Refactor remaining modes
5. **Phase 5 (2 hours)**: Remove duplicated buildXXX methods from individual modes

**Advantages**:
- **Massive LOC reduction**: ~800-1000 lines removed from mode files
- **Centralized panel logic**: Changes to panel appearance affect all modes
- **Consistent UI patterns**: All modes use same panel positioning and styling
- **Easier testing**: Panel builders can be unit tested independently
- **Separation of concerns**: Modes focus on mode-specific logic, not UI construction

**Drawbacks & Risks**:
- **Flexibility reduction**: Modes with unique panel requirements may not fit builders
  - *Mitigation*: Builders return containers for further customization; can still manually build custom panels
- **Refactoring effort**: Must update all 8 modes to use new builders
  - *Mitigation*: Migrate one mode at a time; old methods can coexist during transition
- **Abstraction overhead**: One more layer of indirection
  - *Mitigation*: Builders are simple, well-named functions; easy to understand

**Effort Estimate**:
- **Time**: 14-20 hours (2-3 days with testing)
- **Complexity**: Medium
- **Risk**: Low-Medium
- **Files Impacted**: 10 files (new panels.go + all 8 modes + modemanager.go)

**Critical Assessment**:
Strong complement to Approach 1. Eliminates mode-level duplication while Approach 1 eliminates widget-level duplication. Could combine both for maximum impact. **Recommended as follow-up to Approach 1.**

---

### Approach 3: CombatMode Decomposition with System Functions

**Strategic Focus**: Break down the massive CombatMode (1119 LOC) into focused subsystems

**Problem Statement**:
CombatMode is the largest single file in the GUI package at 1119 LOC with 40+ methods. It handles multiple responsibilities: combat state management, UI rendering, squad visualization, combat logging, input handling, turn management, and squad movement. This violates Single Responsibility Principle and makes the file difficult to understand, test, and modify.

Specific issues:
- Combat log management (lines 312-343) embedded in UI code
- Squad rendering with viewport calculations (lines 791-866) mixed with combat logic
- Squad detail updates (lines 588-642) duplicate ECS queries in multiple methods
- Turn management callbacks (lines 345-396) tightly coupled to UI updates

**Solution Overview**:
Decompose CombatMode into 4-5 focused subsystems using system functions (following ECS best practices from CLAUDE.md). Extract combat logging, squad rendering, squad info queries, and turn handling into separate system modules that CombatMode orchestrates.

**Code Example**:

*Before (combatmode.go lines 312-343 - combat log embedded in UI):*
```go
func (cm *CombatMode) addCombatLog(message string) {
    cm.combatLog = append(cm.combatLog, message)
    cm.messageCountSinceTrim++

    // Use AppendText for O(1) performance - only add the new message
    cm.combatLogArea.AppendText(message + "\n")

    // Every 100 messages, trim old entries to prevent unbounded growth
    if cm.messageCountSinceTrim >= 100 {
        cm.trimCombatLog()
    }
}

// trimCombatLog keeps only the last 300 messages and rebuilds the display
func (cm *CombatMode) trimCombatLog() {
    const maxMessages = 300

    if len(cm.combatLog) > maxMessages {
        // Remove oldest messages, keep most recent ones
        removed := len(cm.combatLog) - maxMessages
        cm.combatLog = cm.combatLog[removed:]

        // Rebuild the text area display with trimmed content
        fullText := ""
        for _, msg := range cm.combatLog {
            fullText += msg + "\n"
        }
        cm.combatLogArea.SetText(fullText)
    }

    cm.messageCountSinceTrim = 0
}
```

*After (new file: gui/combatlog.go - pure system with no UI coupling):*
```go
package gui

// CombatLogSystem manages combat message history with automatic trimming
type CombatLogSystem struct {
    messages           []string
    messageCount       int
    maxMessages        int  // Maximum messages before trim
    trimThreshold      int  // Trigger trim after this many additions
}

func NewCombatLogSystem() *CombatLogSystem {
    return &CombatLogSystem{
        messages:      make([]string, 0, 300),
        maxMessages:   300,
        trimThreshold: 100,
    }
}

// AddMessage appends a message and returns whether a trim occurred
func (cls *CombatLogSystem) AddMessage(message string) (trimmed bool) {
    cls.messages = append(cls.messages, message)
    cls.messageCount++

    // Check if trim needed
    if cls.messageCount >= cls.trimThreshold {
        cls.trim()
        return true
    }

    return false
}

// trim removes oldest messages to keep within maxMessages limit
func (cls *CombatLogSystem) trim() {
    if len(cls.messages) > cls.maxMessages {
        removed := len(cls.messages) - cls.maxMessages
        cls.messages = cls.messages[removed:]
    }
    cls.messageCount = 0
}

// GetAllMessages returns all current messages for display
func (cls *CombatLogSystem) GetAllMessages() []string {
    return cls.messages
}

// GetLatestMessage returns most recent message
func (cls *CombatLogSystem) GetLatestMessage() string {
    if len(cls.messages) == 0 {
        return ""
    }
    return cls.messages[len(cls.messages)-1]
}

// Clear resets the log
func (cls *CombatLogSystem) Clear() {
    cls.messages = cls.messages[:0]
    cls.messageCount = 0
}
```

*CombatMode usage:*
```go
type CombatMode struct {
    // ... existing fields ...
    combatLogSystem *CombatLogSystem
    combatLogArea   *widget.TextArea // UI widget
}

func (cm *CombatMode) Initialize(ctx *UIContext) error {
    // ... existing code ...
    cm.combatLogSystem = NewCombatLogSystem()
    // ... build UI ...
    return nil
}

func (cm *CombatMode) addCombatLog(message string) {
    trimmed := cm.combatLogSystem.AddMessage(message)

    if trimmed {
        // Rebuild entire display after trim
        fullText := ""
        for _, msg := range cm.combatLogSystem.GetAllMessages() {
            fullText += msg + "\n"
        }
        cm.combatLogArea.SetText(fullText)
    } else {
        // Append only new message
        cm.combatLogArea.AppendText(message + "\n")
    }
}

func (cm *CombatMode) Exit(toMode UIMode) error {
    cm.combatLogSystem.Clear()
    return nil
}
```

*Additional System: gui/squadqueryhelpers.go (extract squad info queries):*
```go
package gui

import (
    "game_main/common"
    "game_main/squads"
    "github.com/bytearena/ecs"
)

// SquadQueryHelpers provides reusable squad information queries for UI
type SquadQueryHelpers struct {
    ecsManager *common.EntityManager
}

func NewSquadQueryHelpers(ecsManager *common.EntityManager) *SquadQueryHelpers {
    return &SquadQueryHelpers{ecsManager: ecsManager}
}

// GetSquadName returns the name of a squad by ID
func (sqh *SquadQueryHelpers) GetSquadName(squadID ecs.EntityID) string {
    for _, result := range sqh.ecsManager.World.Query(sqh.ecsManager.Tags["squad"]) {
        squadData := common.GetComponentType[*squads.SquadData](result.Entity, squads.SquadComponent)
        if squadData.SquadID == squadID {
            return squadData.Name
        }
    }
    return "Unknown Squad"
}

// SquadHealthInfo contains health stats for a squad
type SquadHealthInfo struct {
    AliveUnits int
    TotalUnits int
    TotalHP    int
    MaxHP      int
}

// GetSquadHealthInfo returns health statistics for a squad
func (sqh *SquadQueryHelpers) GetSquadHealthInfo(squadID ecs.EntityID) SquadHealthInfo {
    unitIDs := squads.GetUnitIDsInSquad(squadID, sqh.ecsManager)

    info := SquadHealthInfo{
        TotalUnits: len(unitIDs),
    }

    for _, unitID := range unitIDs {
        for _, result := range sqh.ecsManager.World.Query(sqh.ecsManager.Tags["squadmember"]) {
            if result.Entity.GetID() == unitID {
                attrs := common.GetComponentType[*common.Attributes](result.Entity, common.AttributeComponent)
                if attrs.CanAct {
                    info.AliveUnits++
                }
                info.TotalHP += attrs.CurrentHealth
                info.MaxHP += attrs.MaxHealth
            }
        }
    }

    return info
}

// SquadActionInfo contains action state for a squad
type SquadActionInfo struct {
    HasActed          bool
    HasMoved          bool
    MovementRemaining int
}

// GetSquadActionInfo returns action state for a squad
func (sqh *SquadQueryHelpers) GetSquadActionInfo(squadID ecs.EntityID) SquadActionInfo {
    // Find action state entity
    for _, result := range sqh.ecsManager.World.Query(sqh.ecsManager.Tags["actionstate"]) {
        actionState := common.GetComponentType[*combat.ActionStateData](result.Entity, combat.ActionStateComponent)
        if actionState.SquadID == squadID {
            return SquadActionInfo{
                HasActed:          actionState.HasActed,
                HasMoved:          actionState.HasMoved,
                MovementRemaining: actionState.MovementRemaining,
            }
        }
    }
    return SquadActionInfo{}
}
```

*CombatMode simplified usage:*
```go
func (cm *CombatMode) updateSquadDetail() {
    if cm.selectedSquadID == 0 {
        cm.squadDetailText.Label = "Select a squad\nto view details"
        return
    }

    // Use helper systems instead of inline queries
    squadName := cm.squadQueryHelpers.GetSquadName(cm.selectedSquadID)
    healthInfo := cm.squadQueryHelpers.GetSquadHealthInfo(cm.selectedSquadID)
    actionInfo := cm.squadQueryHelpers.GetSquadActionInfo(cm.selectedSquadID)

    detailText := fmt.Sprintf("%s\n", squadName)
    detailText += fmt.Sprintf("Units: %d/%d\n", healthInfo.AliveUnits, healthInfo.TotalUnits)
    detailText += fmt.Sprintf("HP: %d/%d\n", healthInfo.TotalHP, healthInfo.MaxHP)
    detailText += fmt.Sprintf("Move: %d\n", actionInfo.MovementRemaining)

    if actionInfo.HasActed {
        detailText += "Status: Acted\n"
    } else if actionInfo.HasMoved {
        detailText += "Status: Moved\n"
    } else {
        detailText += "Status: Ready\n"
    }

    cm.squadDetailText.Label = detailText
}
```

**Full Decomposition Structure**:

```
gui/
├── combatmode.go (600 LOC, orchestration only)
├── combatlog.go (80 LOC, pure log system)
├── squadqueryhelpers.go (120 LOC, ECS query helpers)
├── squadrenderer.go (150 LOC, squad visualization)
├── combatturnhandler.go (100 LOC, turn callbacks)
```

**Key Changes**:
- Extract CombatLogSystem: pure message management, no UI coupling
- Extract SquadQueryHelpers: reusable ECS queries for squad info
- Extract SquadRenderer: viewport-based squad visualization
- Extract CombatTurnHandler: turn management callbacks
- CombatMode becomes thin orchestration layer: ~600 LOC (from 1119 LOC)

**Implementation Strategy**:
1. **Phase 1 (4 hours)**: Extract CombatLogSystem, update CombatMode to use it, test
2. **Phase 2 (3 hours)**: Extract SquadQueryHelpers, update all squad info methods
3. **Phase 3 (4 hours)**: Extract SquadRenderer for highlight/movement rendering
4. **Phase 4 (3 hours)**: Extract CombatTurnHandler for turn management
5. **Phase 5 (2 hours)**: Final cleanup, ensure CombatMode only orchestrates

**Advantages**:
- **Reduced complexity**: CombatMode drops from 1119 LOC to ~600 LOC
- **Testability**: Each system can be unit tested independently
- **Reusability**: SquadQueryHelpers useful in other modes (SquadManagement, SquadBuilder)
- **Separation of concerns**: Each system has single, clear responsibility
- **ECS alignment**: Follows system-based logic pattern from CLAUDE.md best practices
- **Maintainability**: Changes to combat logging don't affect squad rendering, etc.

**Drawbacks & Risks**:
- **Over-decomposition**: Could create too many small files
  - *Mitigation*: Each extracted system has >80 LOC and clear purpose
- **Refactoring complexity**: Untangling tightly coupled code requires careful analysis
  - *Mitigation*: Incremental extraction; test after each extraction
- **Performance**: More indirection could impact performance
  - *Mitigation*: Systems are thin wrappers; no measurable overhead expected

**Effort Estimate**:
- **Time**: 16-20 hours (3-4 days with testing)
- **Complexity**: Medium-High
- **Risk**: Medium
- **Files Impacted**: 6 files (combatmode.go split into 5 files)

**Critical Assessment**:
This addresses the single largest file complexity issue. However, it requires careful refactoring to avoid breaking combat functionality. **Recommended as medium-term refactoring after Approaches 1 and 2 stabilize the codebase.** CombatMode is complex but functional; don't rush decomposition.

---

## COMPARATIVE ANALYSIS OF FINAL APPROACHES

### Effort vs Impact Matrix
| Approach | Effort | Impact | Risk | Recommended Priority |
|----------|--------|--------|------|---------------------|
| Approach 1: Config-Driven Widgets | Medium | High | Low | 1 |
| Approach 2: Mode Composition | Medium | High | Low | 2 |
| Approach 3: CombatMode Decomposition | High | Medium | Medium | 3 |

### Decision Guidance

**Choose Approach 1 if:**
- You want the quickest LOC reduction with lowest risk
- You need to make adding new UI widgets easier immediately
- You want to follow the proven TextAreaConfig pattern
- Your team is comfortable with config structs and builders
- You want to improve consistency across the GUI

**Choose Approach 2 if:**
- You want to eliminate mode-level duplication
- You need centralized control over panel appearance
- You're okay with a slightly higher abstraction layer
- You want to make creating new modes faster
- You value consistent UI patterns across all modes

**Choose Approach 3 if:**
- CombatMode's complexity is actively blocking development
- You need better testability for combat logic
- You want to reuse squad query logic across multiple modes
- You're willing to invest more time for long-term maintainability
- You have thorough tests for combat functionality

### Combination Opportunities

**Recommended Sequence:**
1. **Start with Approach 1** (Config-Driven Widgets): Quick wins, low risk, high value
2. **Follow with Approach 2** (Mode Composition): Builds on Approach 1, eliminates mode duplication
3. **Consider Approach 3** (CombatMode Decomposition): Only if combat complexity is actively painful

**Synergies:**
- Approach 1 + Approach 2: Eliminates ~1400-1800 LOC (30-40% of GUI package)
- Approach 2 can use Approach 1's config builders in panel construction
- Approach 3 benefits from Approach 2's panel builders (combat UI becomes simpler)

**Maximum Impact Strategy:**
Implement Approaches 1 and 2 together over 1 week:
- Days 1-2: Config structs and builders (Approach 1)
- Days 3-4: Panel builders (Approach 2)
- Day 5: Refactor all modes to use both
- Result: ~1500 LOC removed, much cleaner architecture, low risk

---

## APPENDIX: ADDITIONAL CONSIDERATIONS

### A. Resource Management Refactoring (Bonus Approach)

**Problem**: Global mutable state in guiresources.go (lines 17-28)

```go
var smallFace, _ = loadFont(30)
var largeFace, _ = loadFont(50)
var buttonImage, _ = loadButtonImage()
var defaultWidgetColor = e_image.NewNineSliceColor(color.NRGBA{0x13, 0x1a, 0x22, 0xff})

// Exported fonts for use in UI modes
var SmallFace = smallFace
var LargeFace = largeFace

var PanelRes *panelResources = newPanelResources()
var ListRes *listResources = newListResources()
var TextAreaRes *textAreaResources = newTextAreaResources()
```

**Solution**: Dependency injection via UIContext

```go
// Add to UIContext (uimode.go)
type UIContext struct {
    ECSManager   *common.EntityManager
    PlayerData   *common.PlayerData
    ScreenWidth  int
    ScreenHeight int
    TileSize     int
    // NEW: Injected resources
    Resources    *GUIResources
}

// GUIResources encapsulates all GUI resources
type GUIResources struct {
    SmallFace font.Face
    LargeFace font.Face
    ButtonImage *widget.ButtonImage
    DefaultWidgetColor *e_image.NineSlice
    PanelRes *panelResources
    ListRes *listResources
    TextAreaRes *textAreaResources
}

// NewGUIResources initializes resources with error handling
func NewGUIResources() (*GUIResources, error) {
    smallFace, err := loadFont(30)
    if err != nil {
        return nil, fmt.Errorf("failed to load small font: %w", err)
    }

    largeFace, err := loadFont(50)
    if err != nil {
        return nil, fmt.Errorf("failed to load large font: %w", err)
    }

    buttonImage, err := loadButtonImage()
    if err != nil {
        return nil, fmt.Errorf("failed to load button image: %w", err)
    }

    return &GUIResources{
        SmallFace: smallFace,
        LargeFace: largeFace,
        ButtonImage: buttonImage,
        DefaultWidgetColor: e_image.NewNineSliceColor(color.NRGBA{0x13, 0x1a, 0x22, 0xff}),
        PanelRes: newPanelResources(),
        ListRes: newListResources(),
        TextAreaRes: newTextAreaResources(),
    }, nil
}
```

**Benefits**:
- Explicit initialization with error handling
- Testable (can inject mock resources)
- No package-level state
- Lifecycle management (can reload resources)

**Effort**: 3-4 hours to refactor all modes to use ctx.Resources instead of globals

### B. Layout DSL (Advanced Future Consideration)

**Problem**: Layout calculations scattered, magic percentages everywhere

**Solution**: Declarative layout DSL

```go
// Layout DSL concept
panel := layout.Panel{
    Position: layout.TopRight(),
    Size: layout.Percent{Width: 15, Height: 20},
    Margin: layout.Percent{All: 1},
    Content: []layout.Widget{
        layout.TextArea{
            ID: "statsText",
            Config: TextAreaConfig{...},
        },
    },
}

// Compile to concrete positions
rendered := panel.Render(screenWidth, screenHeight)
```

**Benefits**: Declarative, composable, readable

**Drawbacks**: Significant effort, may be over-engineering

**Recommendation**: Only consider if GUI complexity continues growing

### C. Testing Strategy

**Current State**: No GUI tests (UI testing is hard with ebitenui)

**Recommendations**:
1. **Config Validation**: Test that ButtonConfig/ListConfig produce correct widget options
2. **System Tests**: Test CombatLogSystem, SquadQueryHelpers independently
3. **Integration Tests**: Test mode transitions without rendering
4. **Visual Tests**: Screenshot-based regression testing (advanced)

**Quick Win**: Add table-driven tests for config builders

```go
func TestButtonConfigDefaults(t *testing.T) {
    tests := []struct {
        name   string
        config ButtonConfig
        want   ButtonConfig
    }{
        {
            name: "empty config applies defaults",
            config: ButtonConfig{Text: "Test"},
            want: ButtonConfig{
                Text: "Test",
                MinWidth: 100,
                MinHeight: 100,
                // ... all defaults
            },
        },
    }
    // ... test logic
}
```

---

## PRINCIPLES APPLIED

### Software Engineering Principles
- **DRY (Don't Repeat Yourself)**: Approaches 1 & 2 eliminate massive widget/panel duplication
- **SOLID Principles**:
  - Single Responsibility: Approach 3 decomposes CombatMode into focused systems
  - Open/Closed: Config structs allow extension without modification
  - Dependency Inversion: Resource injection reverses dependency on globals
- **KISS (Keep It Simple, Stupid)**: Config structs are simpler than 10+ option chains
- **YAGNI (You Aren't Gonna Need It)**: Avoided full layout DSL, focused on proven patterns
- **SLAP (Single Level of Abstraction Principle)**: Panel builders encapsulate low-level details
- **Separation of Concerns**: Systems separate combat logic from UI, queries from rendering

### Go-Specific Best Practices
- **Composition over inheritance**: PanelBuilders use composition, not base classes
- **Config structs**: Idiomatic Go pattern for optional parameters
- **Error handling**: Resource initialization returns errors, not panics
- **Dependency injection**: UIContext provides dependencies explicitly
- **System functions**: Stateless helpers that operate on data

### Game Development Considerations
- **Performance**: No significant overhead from abstractions (thin wrappers)
- **Real-time constraints**: Combat log trimming prevents unbounded memory growth
- **Viewport integration**: Squad rendering respects viewport system
- **ECS alignment**: Follows ECS best practices from CLAUDE.md (system-based logic)
- **Tactical gameplay**: Refactorings preserve all combat functionality

---

## NEXT STEPS

### Recommended Action Plan
1. **Immediate** (This week):
   - Implement Approach 1: Config-Driven Widgets (2-3 days)
   - Start with ButtonConfig, ListConfig, PanelConfig
   - Refactor 2-3 modes as proof-of-concept

2. **Short-term** (Next 1-2 weeks):
   - Complete Approach 1 migration across all modes
   - Implement Approach 2: Mode Composition (2-3 days)
   - Extract 6-8 common panel builders
   - Refactor all modes to use panel builders

3. **Medium-term** (Next month):
   - Assess if CombatMode complexity is still blocking
   - If yes, implement Approach 3: CombatMode Decomposition (3-4 days)
   - Extract combat log, squad queries, rendering systems

4. **Long-term** (Next quarter):
   - Consider resource dependency injection (Appendix A)
   - Add unit tests for config builders and systems
   - Evaluate if layout DSL needed (Appendix B)

### Validation Strategy
- **Testing Approach**:
  - Manual testing: Load each mode, verify no visual regressions
  - Functionality testing: Click all buttons, verify transitions work
  - Performance testing: Check combat log doesn't slow down after 1000 messages
  - Config testing: Unit test that configs produce expected options

- **Rollback Plan**:
  - Commit after each mode migration
  - Keep old methods until all modes migrated
  - Can revert individual mode migrations if issues found

- **Success Metrics**:
  - LOC reduction: Target 1500 LOC (30%+) with Approaches 1 & 2
  - Build time: Should not increase (no new dependencies)
  - Bug rate: Track GUI-related bugs before/after
  - Developer velocity: Time to add new UI mode should decrease

### Additional Resources
- **ebitenui documentation**: https://github.com/ebitenui/ebitenui
- **Go config pattern**: Functional Options vs Config Structs (prefer config structs for complex objects)
- **ECS patterns**: Reference squad system (squads/*.go) for system-based design
- **GUI testing**: Consider screenshot-based visual regression testing

---

END OF ANALYSIS
