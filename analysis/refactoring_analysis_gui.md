# Refactoring Analysis: GUI Package
Generated: 2025-10-13
Target: gui/ package (1365 LOC across 10 files)

## EXECUTIVE SUMMARY

### Target Analysis
- **Scope**: Complete GUI package refactoring - button factories, window creation, container patterns, and widget configuration
- **Current State**: 1365 LOC with significant code duplication across window/container creation, TextArea construction (3 nearly identical implementations), and Window creation (3 similar patterns)
- **Primary Issues**:
  1. **TextArea Duplication**: `CreateStatsTextArea()`, `CreateMsgTextArea()`, and `CreateTextArea()` share 90%+ identical code (different only in sizing)
  2. **Window Creation Duplication**: `InfoOptionsWindow`, `DisplayInfoWindow`, and `RootWindow` creation have 85% duplicate configuration
  3. **Container Pattern Repetition**: 11 instances of `widget.NewContainer()` with similar layout configurations
  4. **Button Factory Incomplete**: Only `CreateButton()` exists; no configuration-based factory pattern
  5. **Missing Abstractions**: No common GUI component interfaces or builder patterns

- **Recommended Direction**: Configuration-based factory pattern with composable builders, following the successful Entity Template System pattern (EntityConfig → generic factory)

### Quick Wins vs Strategic Refactoring
- **Immediate Improvements (2-3 hours)**:
  - Consolidate 3 TextArea creation functions into 1 configuration-based factory (-50 LOC)
  - Extract Window creation into configurable factory (-40 LOC)

- **Medium-Term Goals (4-6 hours)**:
  - Implement ButtonConfig/WindowConfig/ContainerConfig pattern
  - Create WidgetFactory with builder methods
  - Establish GUIComponent interface for uniform initialization

- **Long-Term Architecture (8-10 hours)**:
  - Implement declarative UI composition system
  - Add theme/style management system
  - Create reusable panel/dialog templates

### Consensus Findings
- **Agreement Across Perspectives**:
  - TextArea duplication is highest-impact target (3 functions → 1 factory = immediate 50 LOC reduction)
  - Configuration-based pattern proven successful in this codebase (Entity Template System, Graphics Shape System)
  - Ebiten UI library's verbosity requires abstraction layer for maintainability

- **Divergent Perspectives**:
  - **Architectural view**: Favor interface-based composition (GUIComponent, Renderer, InputHandler)
  - **Simplification view**: Favor concrete structs with configuration (WidgetConfig, WindowConfig)
  - **Critical assessment**: Both approaches valid - hybrid recommended (configs for creation, interfaces for behavior)

- **Critical Concerns**:
  - Over-abstraction risk: Ebiten UI already has its own abstractions - avoid double-wrapping
  - Type safety: Configuration structs must maintain compile-time checks
  - Migration cost: 1365 LOC with many call sites - need backward compatibility

---

## FINAL SYNTHESIZED APPROACHES

### Approach 1: Configuration-Based Widget Factory (Incremental Refactoring)

**Strategic Focus**: Gradual consolidation with zero breaking changes, proven factory pattern

**Problem Statement**:
Duplicate widget creation code creates maintenance burden and inconsistency. When modifying TextArea styling (e.g., changing font size or colors), developers must update 3 separate functions. Same issue for Windows (3 creation sites) and Containers (11 creation sites). This violates DRY and makes UI changes error-prone.

**Solution Overview**:
Apply the successful EntityConfig pattern from entitytemplates/creators.go to GUI widgets. Create configuration structs for each widget type and implement generic factory functions. This matches the proven pattern: specialized creation functions → configuration-based factory → backward-compatible wrappers.

**Code Example**:

*Before (TextArea duplication - messagesUI.go:44-98):*
```go
// THREE NEARLY IDENTICAL FUNCTIONS (statsui.go, messagesUI.go, createwidgets.go)
func (msgUI *PlayerMessageUI) CreateMsgTextArea() *widget.TextArea {
    xSize := graphics.StatsUIOffset
    ySize := graphics.ScreenInfo.LevelHeight / 4
    return widget.NewTextArea(
        widget.TextAreaOpts.ContainerOpts(
            widget.ContainerOpts.WidgetOpts(
                widget.WidgetOpts.MinSize(xSize, ySize),
            ),
        ),
        widget.TextAreaOpts.ControlWidgetSpacing(2),
        widget.TextAreaOpts.ProcessBBCode(true),
        widget.TextAreaOpts.FontColor(color.White),
        widget.TextAreaOpts.FontFace(TextAreaRes.face),
        widget.TextAreaOpts.TextPadding(TextAreaRes.entryPadding),
        widget.TextAreaOpts.ScrollContainerOpts(
            widget.ScrollContainerOpts.Image(ListRes.image)),
        widget.TextAreaOpts.SliderOpts(
            widget.SliderOpts.Images(ListRes.track, ListRes.handle),
            widget.SliderOpts.MinHandleSize(ListRes.handleSize),
            widget.SliderOpts.TrackPadding(ListRes.trackPadding),
        ),
    )
}

// statsui.go:18-63 - IDENTICAL except sizing
// createwidgets.go:12-50 - IDENTICAL except sizing
```

*After (Unified TextArea factory - createwidgets.go):*
```go
// Configuration struct
type TextAreaConfig struct {
    MinWidth         int
    MinHeight        int
    FontColor        color.Color
    ProcessBBCode    bool
    ShowScrollbar    bool
    ControlSpacing   int
    // Optional overrides (nil = use defaults from TextAreaRes)
    CustomFace       font.Face
    CustomPadding    *widget.Insets
    CustomImage      *widget.ScrollContainerImage
    CustomTrack      *widget.SliderTrackImage
    CustomHandle     *widget.ButtonImage
}

// Default configuration factory
func DefaultTextAreaConfig(width, height int) TextAreaConfig {
    return TextAreaConfig{
        MinWidth:       width,
        MinHeight:      height,
        FontColor:      color.White,
        ProcessBBCode:  true,
        ShowScrollbar:  false,
        ControlSpacing: 2,
    }
}

// Generic factory
func CreateTextAreaWithConfig(config TextAreaConfig) *widget.TextArea {
    // Use custom values or defaults
    face := config.CustomFace
    if face == nil {
        face = TextAreaRes.face
    }

    padding := config.CustomPadding
    if padding == nil {
        padding = &TextAreaRes.entryPadding
    }

    image := config.CustomImage
    if image == nil {
        image = ListRes.image
    }

    opts := []widget.TextAreaOpt{
        widget.TextAreaOpts.ContainerOpts(
            widget.ContainerOpts.WidgetOpts(
                widget.WidgetOpts.MinSize(config.MinWidth, config.MinHeight),
            ),
        ),
        widget.TextAreaOpts.ControlWidgetSpacing(config.ControlSpacing),
        widget.TextAreaOpts.ProcessBBCode(config.ProcessBBCode),
        widget.TextAreaOpts.FontColor(config.FontColor),
        widget.TextAreaOpts.FontFace(face),
        widget.TextAreaOpts.TextPadding(*padding),
        widget.TextAreaOpts.ScrollContainerOpts(
            widget.ScrollContainerOpts.Image(image)),
    }

    // Add slider opts if configured
    if config.ShowScrollbar || config.CustomTrack != nil {
        track := config.CustomTrack
        if track == nil {
            track = ListRes.track
        }
        handle := config.CustomHandle
        if handle == nil {
            handle = ListRes.handle
        }
        opts = append(opts, widget.TextAreaOpts.SliderOpts(
            widget.SliderOpts.Images(track, handle),
            widget.SliderOpts.MinHandleSize(ListRes.handleSize),
            widget.SliderOpts.TrackPadding(ListRes.trackPadding),
        ))
    }

    return widget.NewTextArea(opts...)
}

// Backward-compatible wrapper (keeps existing API)
func CreateTextArea(minSizeX, minSizeY int) *widget.TextArea {
    return CreateTextAreaWithConfig(DefaultTextAreaConfig(minSizeX, minSizeY))
}

// Usage in statsui.go (simplified):
func (statsUI *PlayerStatsUI) CreateStatsTextArea() *widget.TextArea {
    xSize := graphics.StatsUIOffset
    ySize := graphics.ScreenInfo.LevelHeight / 4
    return CreateTextAreaWithConfig(DefaultTextAreaConfig(xSize, ySize))
}

// Usage in messagesUI.go (simplified):
func (msgUI *PlayerMessageUI) CreateMsgTextArea() *widget.TextArea {
    xSize := graphics.StatsUIOffset
    ySize := graphics.ScreenInfo.LevelHeight / 4
    return CreateTextAreaWithConfig(DefaultTextAreaConfig(xSize, ySize))
}
```

**Key Changes**:
- **TextAreaConfig struct**: Centralizes all configuration parameters with sensible defaults
- **DefaultTextAreaConfig() factory**: Provides common configuration baseline
- **CreateTextAreaWithConfig()**: Single source of truth for TextArea creation
- **Backward compatibility**: Existing `CreateTextArea()` becomes thin wrapper
- **Type safety**: Compile-time checking for all configuration fields

**Value Proposition**:
- **Maintainability**: Change TextArea styling once instead of three places
- **Readability**: Configuration intent clear at call site (`DefaultTextAreaConfig(x, y)`)
- **Extensibility**: Adding scrollbar support or custom styling requires only config field addition
- **Complexity Impact**:
  - -150 LOC (3 duplicate functions eliminated)
  - +80 LOC (configuration struct + factory)
  - **Net: -70 LOC (46% reduction in TextArea code)**

**Implementation Strategy**:
1. **Phase 1 (1h)**: Create TextAreaConfig struct and CreateTextAreaWithConfig() factory in createwidgets.go
2. **Phase 2 (30m)**: Replace statsui.go and messagesUI.go implementations with factory calls
3. **Phase 3 (30m)**: Update existing CreateTextArea() to use new factory (maintains backward compatibility)
4. **Phase 4 (30m)**: Run full UI test pass to verify no visual regressions

**Advantages**:
- **Proven pattern**: Mirrors successful Entity Template System refactoring (EntityConfig → CreateEntityFromTemplate)
- **Zero breaking changes**: Backward-compatible wrappers preserve existing API
- **Incremental migration**: Can refactor one widget type at a time (TextArea → Window → Container → Button)
- **Clear wins**: 46% LOC reduction for TextAreas, 35% expected for Windows, 30% for Containers

**Drawbacks & Risks**:
- **Configuration bloat**: Config structs can become large if every Ebiten option is exposed
  - *Mitigation*: Only expose commonly-changed options, provide escape hatch for advanced usage
- **Default management**: Multiple sources of truth (TextAreaRes globals + config defaults)
  - *Mitigation*: Config always takes precedence, globals as fallback only
- **Testing surface**: Configuration combinations multiply test cases
  - *Mitigation*: Focus tests on commonly-used configurations, document advanced usage

**Effort Estimate**:
- **Time**: 6-8 hours (2h TextArea + 2h Window + 2h Container + 1-2h testing)
- **Complexity**: Low (proven pattern already in codebase)
- **Risk**: Low (backward compatibility maintained)
- **Files Impacted**: 7 files (createwidgets.go, statsui.go, messagesUI.go, infoUI.go, itemdisplaytype.go, throwingUI.go, itemui.go)

**Critical Assessment**:
This approach has highest practical value - it delivers immediate LOC reduction (70-100 lines) with minimal risk. The configuration pattern already proven in Entity Template System and Graphics Shape System means no architectural experimentation. Backward compatibility ensures migration can be gradual. Only concern is configuration struct maintenance, but this is manageable with clear documentation and default factories.

---

### Approach 2: Builder Pattern with Fluent API (Progressive Enhancement)

**Strategic Focus**: Developer experience optimization through chainable configuration

**Problem Statement**:
Even with configuration structs, widget creation requires constructing config objects with verbose field assignments. For complex widgets like Windows (with title bars, modals, drag handlers), configuration initialization becomes unwieldy. Developers need a more ergonomic API that guides them through configuration while maintaining type safety.

**Solution Overview**:
Implement builder pattern with method chaining, similar to Ebiten UI's native API but with higher-level abstractions. Builders encapsulate common configuration patterns and provide fluent API for customization. This reduces cognitive load by hiding Ebiten UI verbosity while maintaining compile-time safety.

**Code Example**:

*Before (Window creation duplication - infoUI.go:90-109 and :124-143):*
```go
// TWO NEARLY IDENTICAL WINDOW CREATIONS in infoUI.go
infoUI.InfoOptionsWindow = widget.NewWindow(
    widget.WindowOpts.Contents(infoUI.InfoOptionsContainer),
    widget.WindowOpts.Modal(),
    widget.WindowOpts.CloseMode(widget.NONE),
    widget.WindowOpts.Draggable(),
    widget.WindowOpts.Resizeable(),
    widget.WindowOpts.MinSize(500, 500),
    widget.WindowOpts.MoveHandler(func(args *widget.WindowChangedEventArgs) {
        // Window moved
    }),
    widget.WindowOpts.ResizeHandler(func(args *widget.WindowChangedEventArgs) {
        // Window resized
    }),
)

// 34 lines later - SAME CONFIGURATION repeated
infoUI.DisplayInfoWindow = widget.NewWindow(
    widget.WindowOpts.Contents(infoUI.DisplayInfoContainer),
    // ... IDENTICAL options ...
)

// itemdisplaytype.go:87-106 - SIMILAR pattern with minor variations
```

*After (Window builder with fluent API):*
```go
// Builder pattern in createwidgets.go
type WindowBuilder struct {
    contents       *widget.Container
    title          string
    titleContainer *widget.Container
    modal          bool
    closeMode      widget.WindowCloseMode
    draggable      bool
    resizeable     bool
    minWidth       int
    minHeight      int
    moveHandler    func(*widget.WindowChangedEventArgs)
    resizeHandler  func(*widget.WindowChangedEventArgs)
}

// Start building with required content
func NewWindowBuilder(contents *widget.Container) *WindowBuilder {
    return &WindowBuilder{
        contents:   contents,
        modal:      true,  // Sensible default for game UI
        closeMode:  widget.CLICK_OUT,
        draggable:  true,
        resizeable: true,
        minWidth:   500,
        minHeight:  500,
    }
}

// Fluent configuration methods
func (wb *WindowBuilder) WithTitle(title string) *WindowBuilder {
    wb.title = title
    return wb
}

func (wb *WindowBuilder) WithTitleContainer(container *widget.Container) *WindowBuilder {
    wb.titleContainer = container
    return wb
}

func (wb *WindowBuilder) WithCloseMode(mode widget.WindowCloseMode) *WindowBuilder {
    wb.closeMode = mode
    return wb
}

func (wb *WindowBuilder) WithSize(width, height int) *WindowBuilder {
    wb.minWidth = width
    wb.minHeight = height
    return wb
}

func (wb *WindowBuilder) NonModal() *WindowBuilder {
    wb.modal = false
    return wb
}

func (wb *WindowBuilder) NonDraggable() *WindowBuilder {
    wb.draggable = false
    return wb
}

func (wb *WindowBuilder) NonResizeable() *WindowBuilder {
    wb.resizeable = false
    return wb
}

func (wb *WindowBuilder) WithMoveHandler(handler func(*widget.WindowChangedEventArgs)) *WindowBuilder {
    wb.moveHandler = handler
    return wb
}

func (wb *WindowBuilder) WithResizeHandler(handler func(*widget.WindowChangedEventArgs)) *WindowBuilder {
    wb.resizeHandler = handler
    return wb
}

// Terminal method - builds the window
func (wb *WindowBuilder) Build() *widget.Window {
    opts := []widget.WindowOpt{
        widget.WindowOpts.Contents(wb.contents),
        widget.WindowOpts.MinSize(wb.minWidth, wb.minHeight),
    }

    if wb.title != "" && wb.titleContainer == nil {
        // Auto-create title container
        wb.titleContainer = createDefaultTitleContainer(wb.title)
    }

    if wb.titleContainer != nil {
        opts = append(opts, widget.WindowOpts.TitleBar(wb.titleContainer, 25))
    }

    if wb.modal {
        opts = append(opts, widget.WindowOpts.Modal())
    }

    opts = append(opts, widget.WindowOpts.CloseMode(wb.closeMode))

    if wb.draggable {
        opts = append(opts, widget.WindowOpts.Draggable())
    }

    if wb.resizeable {
        opts = append(opts, widget.WindowOpts.Resizeable())
    }

    if wb.moveHandler != nil {
        opts = append(opts, widget.WindowOpts.MoveHandler(wb.moveHandler))
    }

    if wb.resizeHandler != nil {
        opts = append(opts, widget.WindowOpts.ResizeHandler(wb.resizeHandler))
    }

    return widget.NewWindow(opts...)
}

// Helper for common pattern
func createDefaultTitleContainer(title string) *widget.Container {
    titleFace, _ := loadFont(12)
    titleContainer := widget.NewContainer(
        widget.ContainerOpts.BackgroundImage(PanelRes.titleBar),
        widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
    )
    titleContainer.AddChild(widget.NewText(
        widget.TextOpts.Text(title, titleFace, color.NRGBA{254, 255, 255, 255}),
        widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
            HorizontalPosition: widget.AnchorLayoutPositionCenter,
            VerticalPosition:   widget.AnchorLayoutPositionCenter,
        })),
    ))
    return titleContainer
}

// Usage in infoUI.go (DRAMATICALLY simplified):
infoUI.InfoOptionsWindow = NewWindowBuilder(infoUI.InfoOptionsContainer).
    WithCloseMode(widget.NONE).
    Build()

infoUI.DisplayInfoWindow = NewWindowBuilder(infoUI.DisplayInfoContainer).
    WithCloseMode(widget.NONE).
    Build()

// itemdisplaytype.go (with title):
itemDisplay.RootWindow = NewWindowBuilder(itemDisplay.RootContainer).
    WithTitle("Throwable Window").
    WithSize(graphics.ScreenInfo.GetCanvasWidth(), 500).
    Build()
```

**Key Changes**:
- **WindowBuilder**: Encapsulates all window configuration with sensible defaults
- **Fluent API**: Method chaining guides developers through configuration options
- **Smart defaults**: Modal, draggable, resizeable by default (matches 90% of use cases)
- **Negation methods**: `NonModal()`, `NonDraggable()` for opt-out rather than opt-in
- **Auto-title generation**: `WithTitle()` automatically creates title container
- **Type safety**: Compile-time checking, IDE autocomplete support

**Value Proposition**:
- **Maintainability**: Common window patterns defined once in builder defaults
- **Readability**: Configuration intent clear from method chain (`WithCloseMode(widget.NONE)`)
- **Extensibility**: Adding new window options requires only new builder method
- **Complexity Impact**:
  - -120 LOC (3 window creation sites simplified)
  - +150 LOC (WindowBuilder implementation)
  - **Net: +30 LOC but massive readability improvement**

**Implementation Strategy**:
1. **Phase 1 (2h)**: Implement WindowBuilder with all configuration methods in createwidgets.go
2. **Phase 2 (1h)**: Implement TextAreaBuilder and ContainerBuilder following same pattern
3. **Phase 3 (2h)**: Migrate all Window creation sites to use builder (infoUI.go, itemdisplaytype.go)
4. **Phase 4 (1h)**: Migrate TextArea and Container creation sites
5. **Phase 5 (1h)**: Add ButtonBuilder to complete widget coverage

**Advantages**:
- **Developer ergonomics**: Chainable API more pleasant to use than config struct initialization
- **IDE support**: Method chaining provides excellent autocomplete experience
- **Self-documenting**: Builder method names describe configuration intent
- **Flexible defaults**: Can change default behavior without breaking existing code
- **Extensibility**: Adding new options non-breaking (existing chains unchanged)

**Drawbacks & Risks**:
- **LOC increase**: Builder implementation adds ~30 LOC per widget type (offset by usage simplification)
  - *Mitigation*: Accept LOC increase as investment in maintainability
- **Pattern complexity**: Developers must learn builder pattern vs simple config structs
  - *Mitigation*: Provide examples and documentation, builders are intuitive
- **Partial configuration**: Method chains can be incomplete (missing required fields)
  - *Mitigation*: Constructor takes required fields, optional fields in methods
- **Testing**: Builder combinations multiply test cases
  - *Mitigation*: Test common paths, rely on Ebiten UI's internal testing for edge cases

**Effort Estimate**:
- **Time**: 10-12 hours (2h WindowBuilder + 2h TextAreaBuilder + 2h ContainerBuilder + 2h ButtonBuilder + 2h migration + 2h testing)
- **Complexity**: Medium (builder pattern requires careful API design)
- **Risk**: Medium (more invasive than config approach, requires API design decisions)
- **Files Impacted**: 10+ files (all GUI files + createwidgets.go expansion)

**Critical Assessment**:
This approach prioritizes developer experience over LOC reduction. The builder pattern is more ergonomic than configuration structs but comes with higher implementation cost (+30 LOC per widget type). Best suited for projects where GUI code will continue to expand significantly. For current codebase (1365 LOC, stable feature set), the simpler configuration approach (Approach 1) may provide better ROI. However, if adding squad UI, tactical overlay, or complex menu systems, builder pattern pays dividends through reduced cognitive load.

---

### Approach 3: Component Composition System (Architectural Restructuring)

**Strategic Focus**: Long-term scalability through interface-based composition, inspired by ECS patterns

**Problem Statement**:
Current GUI code mixes widget construction, layout management, event handling, and state management in monolithic structs (PlayerUI, InfoUI, etc.). This tight coupling makes it difficult to reuse UI patterns, test components in isolation, or compose new interfaces from existing pieces. As the game adds squad management UI, tactical overlays, and more complex menus, this monolithic approach won't scale.

**Solution Overview**:
Apply ECS-inspired composition to GUI architecture. Define small, focused interfaces for UI concerns (Renderable, Interactive, Layoutable, Themed). Implement concrete components that compose these interfaces. Use a GUIManager to coordinate component lifecycle, similar to how ECS Manager coordinates entities. This enables maximum reusability and testability while maintaining clear separation of concerns.

**Code Example**:

*Before (Monolithic PlayerUI - playerUI.go:15-45):*
```go
// Monolithic struct mixing concerns
type PlayerUI struct {
    ItemsUI             PlayerItemsUI          // Item display (construction + state)
    StatsUI             PlayerStatsUI          // Stats display (construction + state)
    MsgUI               PlayerMessageUI        // Messages (construction + state)
    InformationUI       InfoUI                 // Info windows (construction + state)
    MainPlayerInterface *ebitenui.UI           // Root container
}

// Creation tightly coupled to specific implementations
func (playerUI *PlayerUI) CreateMainInterface(playerData *avatar.PlayerData, ecsmanager *common.EntityManager) {
    playerUI.MainPlayerInterface = CreatePlayerUI(playerUI, playerData.Inventory, playerData, ecsmanager)
}

// CreatePlayerUI is 85 lines mixing construction, layout, positioning, initialization
func CreatePlayerUI(playerUI *PlayerUI, inv *gear.Inventory, pl *avatar.PlayerData, ecsmanager *common.EntityManager) *ebitenui.UI {
    ui := ebitenui.UI{}
    rootContainer := widget.NewContainer()

    // Construction mixed with layout
    itemDisplayOptionsContainer := CreateInventorySelectionContainer(playerUI, inv, pl, &ui)
    playerUI.StatsUI.CreateStatsUI()
    playerUI.StatsUI.StatsTextArea.SetText(pl.PlayerAttributes().DisplayString())
    playerUI.MsgUI.CreatMsgUI()

    // Layout mixed with construction
    rootContainer.AddChild(itemDisplayOptionsContainer)
    rootContainer.AddChild(playerUI.StatsUI.StatUIContainer)

    // Positioning hardcoded
    SetContainerLocation(itemDisplayOptionsContainer, graphics.ScreenInfo.GetCanvasWidth()/2, 0)
    SetContainerLocation(playerUI.StatsUI.StatUIContainer, graphics.ScreenInfo.GetCanvasWidth(), 0)

    // ... more mixed concerns ...
}
```

*After (Component composition system):*
```go
// Core interfaces (createwidgets.go or new gui/components.go)

// GUIComponent - minimal interface for all UI components
type GUIComponent interface {
    Initialize() error                  // Setup component (called once)
    Update(ctx *UpdateContext) error    // Update state (called every frame)
    GetWidget() widget.PreferredSizeLocateableWidget  // Get underlying Ebiten UI widget
}

// Renderable - components that need custom rendering
type Renderable interface {
    GUIComponent
    Render(screen *ebiten.Image) error
}

// Interactive - components that handle input
type Interactive interface {
    GUIComponent
    HandleClick(x, y int) bool
    HandleHover(x, y int) bool
}

// Themed - components that respond to theme changes
type Themed interface {
    GUIComponent
    ApplyTheme(theme *UITheme) error
}

// Layoutable - components that participate in layout
type Layoutable interface {
    GUIComponent
    GetLayoutConstraints() LayoutConstraints
    SetPosition(x, y int)
}

// UpdateContext - data passed to components during update
type UpdateContext struct {
    PlayerData   *avatar.PlayerData
    ECSManager   *common.EntityManager
    DeltaTime    float64
    ScreenWidth  int
    ScreenHeight int
}

// LayoutConstraints - declarative layout specification
type LayoutConstraints struct {
    AnchorX      AnchorPosition  // Start, Center, End
    AnchorY      AnchorPosition
    OffsetX      int
    OffsetY      int
    MinWidth     int
    MinHeight    int
    MaxWidth     int
    MaxHeight    int
}

type AnchorPosition int
const (
    AnchorStart AnchorPosition = iota
    AnchorCenter
    AnchorEnd
)

// Concrete component implementation example: StatsPanel
type StatsPanel struct {
    container    *widget.Container
    textArea     *widget.TextArea
    constraints  LayoutConstraints
    initialized  bool
}

func NewStatsPanel(constraints LayoutConstraints) *StatsPanel {
    return &StatsPanel{
        constraints: constraints,
    }
}

func (sp *StatsPanel) Initialize() error {
    if sp.initialized {
        return nil
    }

    // Use factory from Approach 1
    config := DefaultTextAreaConfig(
        sp.constraints.MinWidth,
        sp.constraints.MinHeight,
    )
    sp.textArea = CreateTextAreaWithConfig(config)

    sp.container = widget.NewContainer(
        widget.ContainerOpts.Layout(widget.NewAnchorLayout(
            widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(30)),
        )),
    )
    sp.container.AddChild(sp.textArea)

    sp.initialized = true
    return nil
}

func (sp *StatsPanel) Update(ctx *UpdateContext) error {
    if ctx.PlayerData != nil {
        sp.textArea.SetText(ctx.PlayerData.PlayerAttributes().DisplayString())
    }
    return nil
}

func (sp *StatsPanel) GetWidget() widget.PreferredSizeLocateableWidget {
    return sp.container
}

func (sp *StatsPanel) GetLayoutConstraints() LayoutConstraints {
    return sp.constraints
}

func (sp *StatsPanel) SetPosition(x, y int) {
    SetContainerLocation(sp.container, x, y)
}

// GUIManager - coordinates component lifecycle
type GUIManager struct {
    components []GUIComponent
    root       *ebitenui.UI
    theme      *UITheme
}

func NewGUIManager() *GUIManager {
    return &GUIManager{
        components: make([]GUIComponent, 0),
        root:       &ebitenui.UI{},
        theme:      DefaultTheme(),
    }
}

func (gm *GUIManager) RegisterComponent(component GUIComponent) error {
    if err := component.Initialize(); err != nil {
        return err
    }

    gm.components = append(gm.components, component)

    // Add to root container
    if gm.root.Container == nil {
        gm.root.Container = widget.NewContainer()
    }
    gm.root.Container.AddChild(component.GetWidget())

    // Handle layout if component is Layoutable
    if layoutable, ok := component.(Layoutable); ok {
        gm.applyLayout(layoutable)
    }

    // Apply theme if component is Themed
    if themed, ok := component.(Themed); ok {
        themed.ApplyTheme(gm.theme)
    }

    return nil
}

func (gm *GUIManager) Update(ctx *UpdateContext) error {
    for _, component := range gm.components {
        if err := component.Update(ctx); err != nil {
            return err
        }
    }
    return nil
}

func (gm *GUIManager) applyLayout(component Layoutable) {
    constraints := component.GetLayoutConstraints()

    // Calculate position based on constraints
    var x, y int

    switch constraints.AnchorX {
    case AnchorStart:
        x = constraints.OffsetX
    case AnchorCenter:
        x = graphics.ScreenInfo.GetCanvasWidth()/2 + constraints.OffsetX
    case AnchorEnd:
        x = graphics.ScreenInfo.GetCanvasWidth() + constraints.OffsetX
    }

    switch constraints.AnchorY {
    case AnchorStart:
        y = constraints.OffsetY
    case AnchorCenter:
        y = graphics.ScreenInfo.GetCanvasHeight()/2 + constraints.OffsetY
    case AnchorEnd:
        y = graphics.ScreenInfo.GetCanvasHeight() + constraints.OffsetY
    }

    component.SetPosition(x, y)
}

func (gm *GUIManager) GetRoot() *ebitenui.UI {
    return gm.root
}

// Usage in playerUI.go (DRAMATICALLY simplified):
type PlayerUI struct {
    manager *GUIManager
}

func (playerUI *PlayerUI) CreateMainInterface(playerData *avatar.PlayerData, ecsmanager *common.EntityManager) {
    playerUI.manager = NewGUIManager()

    // Declarative component registration
    statsPanel := NewStatsPanel(LayoutConstraints{
        AnchorX:   AnchorEnd,
        AnchorY:   AnchorStart,
        OffsetX:   0,
        OffsetY:   0,
        MinWidth:  graphics.StatsUIOffset,
        MinHeight: graphics.ScreenInfo.LevelHeight / 4,
    })
    playerUI.manager.RegisterComponent(statsPanel)

    messagesPanel := NewMessagesPanel(LayoutConstraints{
        AnchorX:   AnchorEnd,
        AnchorY:   AnchorStart,
        OffsetY:   graphics.ScreenInfo.LevelHeight/4 + graphics.ScreenInfo.TileSize,
        MinWidth:  graphics.StatsUIOffset,
        MinHeight: graphics.ScreenInfo.LevelHeight / 4,
    })
    playerUI.manager.RegisterComponent(messagesPanel)

    inventoryButtons := NewInventoryButtonPanel(LayoutConstraints{
        AnchorX:  AnchorCenter,
        AnchorY:  AnchorStart,
        MinWidth: 400,
    })
    playerUI.manager.RegisterComponent(inventoryButtons)

    // Get the composed UI
    playerUI.MainPlayerInterface = playerUI.manager.GetRoot()
}

// Update loop (called from main game loop)
func (playerUI *PlayerUI) Update(playerData *avatar.PlayerData, ecsmanager *common.EntityManager, dt float64) {
    ctx := &UpdateContext{
        PlayerData:   playerData,
        ECSManager:   ecsmanager,
        DeltaTime:    dt,
        ScreenWidth:  graphics.ScreenInfo.GetCanvasWidth(),
        ScreenHeight: graphics.ScreenInfo.GetCanvasHeight(),
    }
    playerUI.manager.Update(ctx)
}
```

**Key Changes**:
- **GUIComponent interface**: Minimal contract for all UI components (Initialize, Update, GetWidget)
- **Composition interfaces**: Renderable, Interactive, Themed, Layoutable for opt-in behaviors
- **StatsPanel component**: Self-contained implementation with clear lifecycle
- **GUIManager**: Coordinates component registration, initialization, layout, theming, updates
- **Declarative layout**: LayoutConstraints express positioning intent, manager handles calculation
- **Type-safe composition**: Interface checking at compile time, components opt into capabilities

**Value Proposition**:
- **Maintainability**: Components are isolated, changes don't ripple across codebase
- **Readability**: Intent clear from component registration (`NewStatsPanel(LayoutConstraints{...})`)
- **Extensibility**: New components implement interfaces, no changes to manager
- **Complexity Impact**:
  - -200 LOC (monolithic UI structs eliminated, positioning logic centralized)
  - +300 LOC (interfaces, GUIManager, component implementations)
  - **Net: +100 LOC but massive architectural improvement**

**Implementation Strategy**:
1. **Phase 1 (3h)**: Define core interfaces (GUIComponent, Renderable, Interactive, Themed, Layoutable) and UpdateContext
2. **Phase 2 (2h)**: Implement GUIManager with registration, initialization, layout, theming
3. **Phase 3 (4h)**: Migrate StatsUI and MessagesUI to component implementations (StatsPanel, MessagesPanel)
4. **Phase 4 (3h)**: Migrate ItemsUI and InfoUI to component implementations
5. **Phase 5 (2h)**: Refactor PlayerUI to use GUIManager, update game loop to call Update()
6. **Phase 6 (2h)**: Add theming system and apply to all components
7. **Phase 7 (2h)**: Testing and documentation

**Advantages**:
- **ECS alignment**: Mirrors successful ECS patterns from squad system (pure data components, system coordination)
- **Testability**: Components can be unit tested in isolation without full UI hierarchy
- **Reusability**: Components like StatsPanel can be reused in different contexts (player UI, squad UI, enemy UI)
- **Scalability**: Adding squad management UI becomes component registration, not monolithic refactoring
- **Theme support**: Centralized theming enables UI consistency and player customization
- **Flexible layout**: Declarative constraints decouple positioning from construction

**Drawbacks & Risks**:
- **Highest complexity**: Requires architectural understanding of composition patterns
  - *Mitigation*: Provide extensive documentation and examples, phase migration
- **LOC increase**: +100 LOC for infrastructure, though offset by long-term maintainability
  - *Mitigation*: Accept initial investment for long-term scalability
- **Interface proliferation**: 5+ interfaces may confuse developers unfamiliar with pattern
  - *Mitigation*: Most components only implement GUIComponent, advanced interfaces opt-in
- **Migration cost**: Most invasive refactoring, requires touching all GUI code
  - *Mitigation*: Migrate one UI section at a time (Stats → Messages → Items → Info)
- **Over-engineering risk**: Current 1365 LOC may not justify this level of abstraction
  - *Mitigation*: Only pursue if planning significant GUI expansion (squad UI, tactical overlay, etc.)

**Effort Estimate**:
- **Time**: 18-20 hours (3h interfaces + 2h manager + 7h component migration + 4h layout system + 2h theming + 2h testing)
- **Complexity**: High (requires architectural design, interface knowledge, careful migration)
- **Risk**: High (most invasive, potential for over-engineering)
- **Files Impacted**: All 10 GUI files + new components.go file

**Critical Assessment**:
This approach represents maximum architectural investment. It aligns perfectly with the ECS patterns proven successful in the squad system and provides best long-term scalability. However, for current codebase size (1365 LOC) and stable feature set, this may be over-engineering. **Recommended only if**:
1. Planning significant GUI expansion (squad management UI with 5+ new panels)
2. Need UI component reusability across multiple game modes
3. Want to enable modding/theming for player customization
4. Team has strong composition pattern experience

For most cases, Approach 1 (Configuration-Based Factory) provides 80% of benefits for 20% of effort. Approach 3 is the "future-proof" option but requires justification through planned feature expansion.

---

## COMPARATIVE ANALYSIS OF FINAL APPROACHES

### Effort vs Impact Matrix
| Approach | Effort | Impact | Risk | Recommended Priority |
|----------|--------|--------|------|---------------------|
| Approach 1: Configuration Factory | Low (6-8h) | High (immediate 70-100 LOC reduction) | Low | 1 |
| Approach 2: Builder Pattern | Medium (10-12h) | Medium (ergonomics, +30 LOC) | Medium | 3 |
| Approach 3: Component Composition | High (18-20h) | Very High (scalability, +100 LOC) | High | 2 |

### Decision Guidance

**Choose Approach 1 (Configuration Factory) if:**
- **Primary goal**: Eliminate code duplication quickly with minimal risk
- **Timeline constraint**: Need results within 1-2 days
- **Team familiarity**: Developers comfortable with struct-based configuration (already proven in Entity Template System)
- **Current priority**: Reduce maintenance burden on existing GUI code without major changes
- **Feature stability**: GUI feature set is stable, no major UI expansion planned

**Choose Approach 2 (Builder Pattern) if:**
- **Primary goal**: Improve developer experience when writing new GUI code
- **Timeline flexibility**: Can invest 2 weeks for ergonomic improvements
- **Team preference**: Developers prefer fluent APIs over configuration structs
- **Current priority**: Making GUI code more pleasant to write for ongoing feature development
- **Feature growth**: Moderate GUI expansion planned (3-5 new windows/panels)

**Choose Approach 3 (Component Composition) if:**
- **Primary goal**: Future-proof architecture for significant GUI expansion
- **Timeline flexibility**: Can invest 3-4 weeks for architectural refactoring
- **Team experience**: Developers familiar with composition patterns and interface-based design
- **Current priority**: Enable squad management UI, tactical overlays, and other complex interface systems
- **Feature explosion**: Major GUI expansion planned (10+ new components, multiple game modes)

### Combination Opportunities

**Pragmatic Hybrid: Approach 1 + Approach 2 (Recommended)**
- **Implementation**: Start with Configuration Factory (Approach 1), then add optional Builder wrappers (Approach 2)
- **Timeline**: 8-10 hours (6h factory + 2-4h builders for most-used widgets)
- **Benefits**:
  - Immediate LOC reduction from factory pattern
  - Ergonomic builders for complex widgets (Windows, Buttons)
  - Lower risk than pure builder approach (factory is foundation)
- **Migration path**:
  1. Week 1: Implement TextAreaConfig + factory → 70 LOC reduction
  2. Week 2: Implement WindowConfig + factory → 40 LOC reduction
  3. Week 3: Add WindowBuilder wrapper for ergonomics → 0 LOC change, improved DX
  4. Week 4: Add ButtonBuilder for future button creation → foundation for button factory

**Future-Proof Path: Approach 1 → Approach 3**
- **Implementation**: Start with Configuration Factory (Approach 1), migrate to Component Composition (Approach 3) when GUI expansion hits threshold
- **Threshold trigger**: When adding 5+ new UI panels or implementing squad management UI
- **Benefits**:
  - Immediate wins from factory pattern without blocking future scalability
  - Configuration structs become component initialization data
  - Factories become component factory methods
- **Migration path**:
  1. Now: Implement Approach 1 (6-8h) → immediate 70-100 LOC reduction
  2. When squad UI starts: Begin Approach 3 migration (18-20h) → existing configs fold into component constructors
  3. End state: Component-based architecture with proven configuration patterns

**Quick Win + Strategic Reserve:**
- **Implementation**: Approach 1 now, document Approach 3 design for future
- **Timeline**: 6-8 hours implementation + 2 hours design documentation
- **Benefits**:
  - Solve immediate duplication problem
  - Design document guides future expansion
  - Team learns Approach 1 patterns before tackling Approach 3
- **Documentation**: Create `gui/ARCHITECTURE.md` describing component composition vision, interface contracts, and migration strategy

---

## APPENDIX: DETAILED CODE ANALYSIS

### A. Current State Duplication Metrics

#### TextArea Duplication (Highest Priority)
**3 nearly identical implementations:**

| File | Function | LOC | Differences |
|------|----------|-----|-------------|
| createwidgets.go | CreateTextArea() | 38 | Generic sizing parameters |
| statsui.go | CreateStatsTextArea() | 45 | Stats-specific sizing (xSize, ySize calculated) |
| messagesUI.go | CreateMsgTextArea() | 53 | Messages-specific sizing (identical calculation to stats) |

**Duplicate code blocks:**
- Lines 14-48 in createwidgets.go
- Lines 18-63 in statsui.go (95% identical to createwidgets.go)
- Lines 44-98 in messagesUI.go (95% identical to createwidgets.go)

**Refactoring impact**: 3 functions (136 LOC) → 1 factory function + config struct (60 LOC) = **56% reduction (-76 LOC)**

---

#### Window Creation Duplication (Second Priority)
**3 similar implementations:**

| File | Window | LOC | Variations |
|------|--------|-----|-----------|
| infoUI.go | InfoOptionsWindow | 20 | CloseMode: widget.NONE, no title |
| infoUI.go | DisplayInfoWindow | 20 | CloseMode: widget.NONE, no title (IDENTICAL to InfoOptionsWindow) |
| itemdisplaytype.go | RootWindow | 20 | CloseMode: widget.CLICK_OUT, with title bar |

**Duplicate code blocks:**
- Lines 90-109 in infoUI.go (InfoOptionsWindow)
- Lines 124-143 in infoUI.go (DisplayInfoWindow) - **100% duplicate of InfoOptionsWindow**
- Lines 87-106 in itemdisplaytype.go (85% similar, adds title bar)

**Common configuration**:
- All use `widget.WindowOpts.Modal()`
- All use `widget.WindowOpts.Draggable()`
- All use `widget.WindowOpts.Resizeable()`
- All use `widget.WindowOpts.MinSize()` (different values)
- All have empty MoveHandler and ResizeHandler placeholders

**Refactoring impact**: 3 window creations (60 LOC) → 1 factory function with WindowConfig (25 LOC usage) = **58% reduction (-35 LOC)**

---

#### Container Creation Patterns (Third Priority)
**11 instances with repetitive patterns:**

| File | Container Purpose | Layout Type | LOC |
|------|-------------------|-------------|-----|
| playerUI.go:53 | Root container | None (empty) | 3 |
| itemui.go:23 | Inventory selection | GridLayout 4 columns | 15 |
| statsui.go:69 | Stats container | AnchorLayout | 8 |
| messagesUI.go:104 | Messages container | AnchorLayout | 8 |
| throwingUI.go:58 | Throwing root | GridLayout 2 columns | 14 |
| throwingUI.go:80 | Throwing inventory | AnchorLayout | 4 |
| infoUI.go:81 | Info options | AnchorLayout + RowLayout | 8 |
| infoUI.go:115 | Info display | AnchorLayout + RowLayout | 8 |
| itemdisplaytype.go:52 | Item selected | AnchorLayout | 4 |
| itemdisplaytype.go:61 | Item display | AnchorLayout | 4 |
| itemdisplaytype.go:75 | Title container | AnchorLayout | 3 |

**Common patterns identified:**
1. **AnchorLayout containers** (7 instances): Same padding (30px), same background (defaultWidgetColor)
2. **GridLayout containers** (2 instances): Different column counts, similar spacing/padding configuration
3. **Composite layouts** (2 instances): AnchorLayout + RowLayout/GridLayout

**Refactoring impact**: Extracting 3 factory functions:
- `CreateAnchorContainer(padding int) → -20 LOC`
- `CreateGridContainer(columns int, spacing int) → -15 LOC`
- `CreateCompositeContainer(config ContainerConfig) → -10 LOC`
- **Total: -45 LOC (40% reduction)**

---

### B. Button Factory Gap Analysis

**Current state:**
- `CreateButton(text string)` exists in createwidgets.go:52-82 (31 LOC)
- Only one button type supported (standard game button with fixed styling)

**Missing button types:**
1. **Menu button**: Larger size, different font, different color scheme
2. **Icon button**: No text, image-based
3. **Toggle button**: State-based appearance
4. **Disabled button**: Grayed-out appearance

**Usage sites:**
- `CreateOpenThrowablesButton()` in playerUI.go:89-109 (uses CreateButton + adds handler)
- Equipment button removed (commented out in itemui.go:39)

**Current button creation pattern problems:**
1. Click handlers added separately after button creation (verbose)
2. No support for button variants (size, color, style)
3. Hardcoded sizing (100x100 minimum, 30px padding)
4. No disabled state configuration

**Refactoring opportunity:**
```go
// Add ButtonConfig struct
type ButtonConfig struct {
    Text         string
    Width        int
    Height       int
    Padding      widget.Insets
    Image        *widget.ButtonImage  // Allow custom styling
    FontFace     font.Face
    TextColor    *widget.ButtonTextColor
    ClickHandler func(*widget.ButtonClickedEventArgs)
}

// Factory function
func CreateButtonWithConfig(config ButtonConfig) *widget.Button {
    // Implementation with all config options
}

// Convenience factories
func CreateMenuButton(text string, handler func(*widget.ButtonClickedEventArgs)) *widget.Button {
    return CreateButtonWithConfig(ButtonConfig{
        Text:         text,
        Width:        200,  // Larger than standard
        Height:       60,
        ClickHandler: handler,
        // ... menu-specific defaults
    })
}
```

**Impact**: Enables CreateOpenThrowablesButton simplification:
- Current: 21 lines (button creation + configuration + handler)
- After: 5 lines (single factory call with handler)

---

### C. Resource Management Analysis

**Global resources in guiresources.go:**
```go
var smallFace, _ = loadFont(30)
var largeFace, _ = loadFont(50)
var buttonImage, _ = loadButtonImage()
var defaultWidgetColor = e_image.NewNineSliceColor(color.NRGBA{0x13, 0x1a, 0x22, 0xff})

var PanelRes *panelResources = newPanelResources()
var ListRes *listResources = newListResources()
var TextAreaRes *textAreaResources = newTextAreaResources()
```

**Problem: Scattered resource structs**
- `panelResources` (38 LOC) - panel styling
- `listResources` (86 LOC) - list/scrollbar styling
- `textAreaResources` (70 LOC) - textarea styling (95% duplicate of listResources!)

**Duplication discovered**:
`newListResources()` and `newTextAreaResources()` are 95% identical:
- Same image loading (idle, disabled, mask, tracks, handles)
- Same slider configuration
- Different: listResources has `entry *widget.ListEntryColor`, textAreaResources omits this

**Refactoring opportunity**:
```go
// Unified scroll widget resources
type scrollWidgetResources struct {
    image        *widget.ScrollContainerImage
    track        *widget.SliderTrackImage
    trackPadding widget.Insets
    handle       *widget.ButtonImage
    handleSize   int
    face         font.Face
    entryPadding widget.Insets
}

func newScrollWidgetResources() *scrollWidgetResources {
    // Single implementation, shared by list and textarea
}

// List-specific additions
type listResources struct {
    *scrollWidgetResources  // Embed common resources
    entry *widget.ListEntryColor
}

func newListResources() *listResources {
    return &listResources{
        scrollWidgetResources: newScrollWidgetResources(),
        entry: &widget.ListEntryColor{/* ... */},
    }
}

// TextArea just uses scroll resources directly
var TextAreaRes *scrollWidgetResources = newScrollWidgetResources()
var ListRes *listResources = newListResources()
```

**Impact**: Eliminate 70 LOC duplicate resource initialization (-45% in resource code)

---

### D. Layout System Analysis

**Current positioning approach:**
- Hardcoded calculations in CreatePlayerUI() (playerUI.go:48-85)
- Manual positioning via SetContainerLocation() helper (createwidgets.go:84-91)

**Positioning patterns found:**
```go
// Pattern 1: Screen edge anchoring
SetContainerLocation(itemDisplayOptionsContainer,
    graphics.ScreenInfo.GetCanvasWidth()/2, 0)  // Center-top

SetContainerLocation(playerUI.StatsUI.StatUIContainer,
    graphics.ScreenInfo.GetCanvasWidth(), 0)    // Right-top

// Pattern 2: Relative positioning
SetContainerLocation(playerUI.MsgUI.msgUIContainer,
    graphics.ScreenInfo.GetCanvasWidth(),
    graphics.ScreenInfo.GetCanvasHeight()/4 + graphics.ScreenInfo.TileSize)  // Right, 1/4 down + 1 tile
```

**Problems identified:**
1. **Calculation duplication**: Screen edge calculations repeated at every positioning call
2. **Intent unclear**: Is `GetCanvasWidth()` meant to be right edge or is offset needed?
3. **No relative positioning**: Can't express "below stats panel" declaratively
4. **Hardcoded for screen size**: No responsive layout support

**Refactoring opportunity** (Approach 3 provides this):
```go
// Declarative layout with clear intent
statsPanel := NewStatsPanel(LayoutConstraints{
    AnchorX:   AnchorEnd,     // Right edge
    AnchorY:   AnchorStart,   // Top edge
    OffsetX:   0,
    OffsetY:   0,
    MinWidth:  graphics.StatsUIOffset,
    MinHeight: graphics.ScreenInfo.LevelHeight / 4,
})

messagesPanel := NewMessagesPanel(LayoutConstraints{
    AnchorX:   AnchorEnd,     // Right edge
    AnchorY:   AnchorStart,   // Top edge
    OffsetY:   graphics.ScreenInfo.LevelHeight/4 + graphics.ScreenInfo.TileSize,  // Below stats
    MinWidth:  graphics.StatsUIOffset,
    MinHeight: graphics.ScreenInfo.LevelHeight / 4,
})
```

**Impact**: Clear intent, centralized calculation, foundation for responsive layout

---

### E. Interface Opportunity Analysis

**Interfaces present in codebase:**
```go
// itemdisplaytype.go:28-33
type ItemDisplayer interface {
    CreateRootContainer()
    SetupContainers()
    CreateInventoryList(propFilters ...gear.StatusEffects)
    DisplayInventory()
}

// createwidgets.go:93-95
type StringDisplay interface {
    DisplayString()
}
```

**ItemDisplayer interface analysis:**
- Purpose: Standardize inventory display windows (Throwables, Equipment [removed], Consumables)
- Implementation: Only `ThrowingItemDisplay` currently implements (Equipment removed in squad system migration)
- Problem: Overly specific - forces all implementers to have same 4-method structure
- Usage: Not used polymorphically (no code accepts `ItemDisplayer` interface)

**Assessment**: Interface exists but provides limited value
- Not used for polymorphism
- Only 1 current implementation
- Methods too specific to inventory display

**Refactoring opportunity** (Approach 3):
Replace ItemDisplayer with more general GUIComponent:
```go
type GUIComponent interface {
    Initialize() error
    Update(ctx *UpdateContext) error
    GetWidget() widget.PreferredSizeLocateableWidget
}

// Inventory display becomes a specific component type
type InventoryDisplayComponent struct {
    displayType  InventoryDisplayType  // Throwables, Equipment, Consumables
    filterFunc   func(*gear.Inventory) []gear.InventoryListEntry
    selectHandler func(gear.InventoryListEntry)
    // ... component fields
}
```

**Impact**: More flexible component model, better polymorphism, clearer composition

---

## SYNTHESIS RATIONALE

### Why These 3 Final Approaches?

**Approach 1 Selection (Configuration Factory)**:
This approach was selected as the primary recommendation because:
1. **Proven pattern**: Entity Template System (recently completed) used identical configuration-based pattern with great success
2. **Immediate ROI**: 70-100 LOC reduction achievable in 6-8 hours with minimal risk
3. **Addresses primary pain point**: TextArea duplication (3 functions → 1 factory = 56% reduction)
4. **Backward compatible**: Existing code continues to work, migration is opt-in
5. **Foundation for future**: Configuration structs can evolve into builder methods or component initialization data

**Combination of analysis perspectives**:
- **Architectural view** identified pattern: DRY violation, opportunity for abstraction
- **Simplification view** validated: Concrete struct-based configuration is simpler than interfaces
- **Critical assessment** confirmed: This is practical, not over-engineering; proven in this codebase

**Approach 2 Selection (Builder Pattern)**:
This approach was selected as the ergonomic enhancement because:
1. **Developer experience**: Fluent API more pleasant for complex widget configuration (Windows with 8+ options)
2. **Progressive enhancement**: Builds on Approach 1's configuration structs (builder wraps config)
3. **Industry standard**: Builder pattern widely understood, good IDE support
4. **Balances verbosity**: Ebiten UI is verbose - builder provides right abstraction level
5. **Optional adoption**: Can implement builders for most-used widgets (Window, Button) without full migration

**Combination of analysis perspectives**:
- **Architectural view** suggested: Fluent interfaces for better composition
- **Simplification view** warned: Don't add unnecessary complexity
- **Critical assessment** balanced: Builders justified for Window creation (85% duplication), optional for simpler widgets

**Approach 3 Selection (Component Composition)**:
This approach was selected as the future-proofing option because:
1. **Aligns with ECS**: Squad system demonstrated perfect ECS patterns - GUI should follow same philosophy
2. **Maximum scalability**: Adding squad management UI (5+ panels) requires composition, not monoliths
3. **Testability**: Component isolation enables unit testing without full UI hierarchy
4. **Reusability**: StatsPanel component reusable in player UI, squad UI, enemy UI, debug UI
5. **Long-term vision**: Positions codebase for tactical overlay, formation editor, complex menus

**Combination of analysis perspectives**:
- **Architectural view** championed: Interface-based composition for maximum flexibility
- **Simplification view** cautioned: 1365 LOC may not justify this complexity
- **Critical assessment** concluded: Only pursue if GUI expansion planned (threshold: 5+ new panels)

### Rejected Elements

**From initial analysis - NOT included in final approaches:**

1. **Theme/Style System** (mentioned in Approach 3 but not core):
   - Reason: 1365 LOC codebase likely doesn't need dynamic theming
   - Would add complexity without clear user benefit
   - If needed later, can add via Themed interface in Approach 3

2. **Declarative UI DSL** (considered but rejected):
   - Reason: Go doesn't have good DSL support, would require reflection or code generation
   - Over-engineering for current needs
   - Ebiten UI already provides declarative option functions

3. **Event Bus System** (considered for inter-component communication):
   - Reason: Current UI communication is simple (direct method calls)
   - Event bus adds indirection without clear benefit
   - Can use UpdateContext pattern from Approach 3 if needed

4. **Virtual DOM / Reactive UI** (modern web-inspired approach):
   - Reason: Ebiten UI is immediate-mode, not retained-mode - mismatch
   - Would require rewriting Ebiten UI wrapper - massive undertaking
   - Current widget model works fine for game UI

5. **Automated Layout Solver** (constraint-based layout):
   - Reason: Game UI has fixed layouts, not responsive like web
   - Complexity far exceeds benefit
   - Manual positioning works fine for 10-15 UI elements

6. **Widget Pooling / Recycling** (performance optimization):
   - Reason: No performance issues identified in current UI
   - Premature optimization
   - Ebiten UI handles widget lifecycle efficiently

### Critical Evaluation Insights

**Key insights from critical analysis perspective:**

1. **Configuration vs Builder trade-off**:
   - Configuration structs: Lower LOC, simpler pattern, proven in codebase
   - Builders: Higher LOC, better ergonomics, more complex pattern
   - **Decision**: Start with configs (Approach 1), add builders for complex widgets (Approach 2 optional)

2. **Abstraction level calibration**:
   - Too low: Ebiten UI verbosity leaks into game code (current state)
   - Just right: Configuration factories hide verbosity, expose intent (Approach 1)
   - Too high: Component composition, interfaces, managers (Approach 3)
   - **Decision**: Approach 1 is "just right" for current needs, Approach 3 for future expansion

3. **Migration risk assessment**:
   - Approach 1: Low risk (backward compatible, incremental)
   - Approach 2: Medium risk (API change, but optional)
   - Approach 3: High risk (full architectural change, touches all code)
   - **Decision**: Prioritize by risk/reward ratio - Approach 1 first

4. **LOC reduction reality check**:
   - Naive estimate: "Eliminate duplication = massive LOC reduction"
   - Reality: Factory infrastructure costs LOC upfront
   - Approach 1: -70 LOC net (infrastructure costs offset by duplication elimination)
   - Approach 2: +30 LOC net (ergonomics investment, not LOC reduction)
   - Approach 3: +100 LOC net (architectural investment, not LOC reduction)
   - **Decision**: Only Approach 1 delivers LOC reduction; others are strategic investments

5. **Real-world implementation considerations**:
   - Factory functions need comprehensive testing (config combinations multiply test cases)
   - Documentation critical (new developers must understand factory patterns)
   - Migration can't be big-bang (must be incremental to avoid breaking game)
   - **Decision**: Phase implementation - TextArea first (highest impact, lowest risk)

---

## PRINCIPLES APPLIED

### Software Engineering Principles

- **DRY (Don't Repeat Yourself)**:
  - **Problem**: 3 TextArea creation functions share 95% code, 3 Window creation sites share 85% code
  - **Solution**: Consolidate into configuration-based factories (single source of truth)
  - **Applied in**: All 3 approaches (Approach 1 with configs, Approach 2 with builders, Approach 3 with components)

- **SOLID Principles**:
  - **Single Responsibility**:
    - Current violation: PlayerUI mixes construction, layout, state management
    - Applied in Approach 3: Components have single responsibility (StatsPanel only handles stats display)
  - **Open/Closed**:
    - Applied in Approach 1: Configuration structs extensible without modifying factory function
    - Applied in Approach 2: Builder methods additive, existing chains unchanged
  - **Liskov Substitution**:
    - Applied in Approach 3: GUIComponent implementations substitutable via interface
  - **Interface Segregation**:
    - Applied in Approach 3: Separate interfaces (Renderable, Interactive, Themed) instead of monolithic interface
  - **Dependency Inversion**:
    - Applied in Approach 3: GUIManager depends on GUIComponent interface, not concrete types

- **KISS (Keep It Simple, Stupid)**:
  - **Priority**: Simplicity over cleverness
  - **Applied in Approach 1**: Configuration structs are simplest solution to duplication
  - **Avoided**: Theme systems, event buses, reactive UI - unnecessary complexity

- **YAGNI (You Aren't Gonna Need It)**:
  - **Applied**: Rejected declarative UI DSL, virtual DOM, widget pooling - no current need
  - **Validated in critical assessment**: Only build abstractions needed now (Approach 1), defer complex systems (Approach 3) until justified

- **SLAP (Single Level of Abstraction Principle)**:
  - **Problem**: CreatePlayerUI() mixes high-level composition with low-level Ebiten UI calls
  - **Applied in Approach 1**: Factory functions hide low-level Ebiten UI details
  - **Applied in Approach 3**: Component methods operate at single abstraction level (Initialize/Update/GetWidget)

- **Separation of Concerns**:
  - **Problem**: Monolithic UI structs mix widget construction, layout, event handling, state
  - **Applied in Approach 1**: Separate construction (factories) from usage (UI code)
  - **Applied in Approach 3**: Separate concerns via interfaces (Layoutable for positioning, Interactive for input)

### Go-Specific Best Practices

- **Composition over inheritance**:
  - Go doesn't have inheritance - use struct embedding and interfaces
  - Applied in Approach 3: Components embed behavior via interface composition
  - Example: `scrollWidgetResources` embedded in `listResources` (Appendix C)

- **Interface design considerations**:
  - Keep interfaces small (Go proverb: "The bigger the interface, the weaker the abstraction")
  - Applied in Approach 3: GUIComponent has 3 methods, opt-in interfaces have 1-2 methods
  - Avoid: Monolithic ItemDisplayer interface (4 methods, no polymorphic usage)

- **Error handling approaches**:
  - Explicit error returns, no exceptions
  - Applied in all approaches: Factory functions return (widget, error) when loading resources
  - Applied in Approach 3: Initialize() and Update() return error for explicit error handling

- **Value vs pointer semantics**:
  - Use values for small, immutable data (configs)
  - Use pointers for large, mutable data (widgets, components)
  - Applied in Approach 1: TextAreaConfig passed by value (config is specification)
  - Applied in Approach 3: Components are pointers (stateful, mutable)

- **Idiomatic Go patterns used**:
  - **Functional options**: Ebiten UI uses this pattern extensively, approaches follow suit
  - **Builder pattern** (Approach 2): Go-idiomatic method chaining with pointer receivers
  - **Interface composition** (Approach 3): Small interfaces composed into larger capabilities

### Game Development Considerations

- **Performance implications**:
  - GUI is not performance-critical (10-15 widgets, rendered once per frame)
  - Approach 1/2: Negligible performance impact (factories called at initialization)
  - Approach 3: Minimal Update() overhead (10-15 components * simple state checks = microseconds)
  - **Validation**: No performance concerns identified

- **Real-time system constraints**:
  - GUI must not block game loop (Ebiten is 60 FPS)
  - Current code: Immediate-mode rendering, no blocking operations
  - All approaches: Maintain immediate-mode pattern, no async complications
  - **Validation**: No real-time concerns introduced

- **Game loop integration**:
  - Current: UI updated inline with game logic (scattered calls)
  - Approach 1/2: No change to update pattern
  - Approach 3: Centralized Update() call in game loop (UpdateContext pattern)
  - **Benefit**: Approach 3 makes UI lifecycle explicit

- **Tactical gameplay preservation**:
  - Not applicable: GUI refactoring doesn't affect tactical systems
  - Squad UI integration: Approach 3 enables clean integration of squad panels
  - **Future consideration**: Squad management UI will need component composition (Approach 3)

---

## NEXT STEPS

### Recommended Action Plan

**Immediate (This Week): Implement Approach 1 (6-8 hours)**
1. **Day 1 (3-4h)**: TextArea consolidation
   - Create `TextAreaConfig` struct in createwidgets.go
   - Implement `CreateTextAreaWithConfig()` factory
   - Implement `DefaultTextAreaConfig()` helper
   - Update `CreateTextArea()` to use factory (backward compatible)
   - Replace `CreateStatsTextArea()` in statsui.go with factory call
   - Replace `CreateMsgTextArea()` in messagesUI.go with factory call
   - Run full UI test pass

2. **Day 2 (2-3h)**: Window consolidation
   - Create `WindowConfig` struct in createwidgets.go
   - Implement `CreateWindowWithConfig()` factory
   - Replace InfoOptionsWindow creation in infoUI.go
   - Replace DisplayInfoWindow creation in infoUI.go
   - Replace RootWindow creation in itemdisplaytype.go
   - Run full UI test pass

3. **Day 3 (1h)**: Container simplification
   - Create `CreateAnchorContainer()` helper
   - Create `CreateGridContainer()` helper
   - Replace 3-5 most duplicated container creation sites
   - Run full UI test pass

**Short-term (Next 2 Weeks): Optional Approach 2 Enhancement (4-6 hours)**
Only pursue if GUI code actively being expanded (new windows/panels)

1. **Week 2 (2-3h)**: WindowBuilder implementation
   - Implement `WindowBuilder` struct in createwidgets.go
   - Add fluent configuration methods
   - Update 1-2 complex window creation sites to use builder
   - Document builder pattern for team

2. **Week 2 (1-2h)**: ButtonBuilder implementation
   - Implement `ButtonConfig` struct
   - Implement `CreateButtonWithConfig()` factory
   - Create `ButtonBuilder` wrapper (optional)
   - Simplify `CreateOpenThrowablesButton()` using factory

3. **Week 2 (1h)**: Documentation and examples
   - Add examples to createwidgets.go comments
   - Update CLAUDE.md with completed GUI refactoring
   - Remove "10% complete" status from roadmap

**Medium-term (When Squad UI Starts): Evaluate Approach 3 (Planning Phase)**
Trigger: When beginning squad management UI implementation

1. **Pre-implementation (2-3h)**: Design validation
   - Review Approach 3 component composition design
   - Validate interfaces match squad UI requirements
   - Prototype 1-2 components (StatsPanel, SquadPanel)
   - Decision point: Proceed with Approach 3 or extend Approach 1/2?

2. **If proceeding (18-20h)**: Approach 3 implementation
   - Follow implementation strategy from Approach 3 section
   - Migrate existing UI incrementally (Stats → Messages → Items → Info)
   - Implement squad management UI as components
   - Document component composition patterns

**Long-term (3+ Months): Architecture Review**
1. Assess whether Approach 3 is needed based on GUI growth
2. Review squad UI implementation complexity
3. Evaluate tactical overlay requirements
4. Make final decision: extend Approach 1/2 or migrate to Approach 3

### Validation Strategy

**Testing Approach**:
1. **Visual regression testing**:
   - Before/after screenshots of all UI panels
   - Verify pixel-perfect match (no unintended styling changes)
   - Test: Main menu, stats panel, messages panel, throwables window, info windows

2. **Functional testing**:
   - Verify all buttons clickable and handlers execute
   - Verify all windows draggable/resizeable (if configured)
   - Verify all text areas display correct content
   - Verify inventory displays filter correctly

3. **Integration testing**:
   - Run full game loop with refactored UI
   - Test level transitions (UI state resets correctly)
   - Test throwable selection → AOE display flow
   - Test right-click info window → creature lookup flow

4. **Code review checklist**:
   - [ ] No Ebiten UI verbosity leaked into game code
   - [ ] Configuration structs have sensible defaults
   - [ ] Factory functions handle nil/zero values gracefully
   - [ ] Backward compatibility maintained (existing code compiles)
   - [ ] Documentation added for new patterns

**Rollback Plan**:
1. **Git branching strategy**:
   - Create `feature/gui-refactor-approach1` branch
   - Commit TextArea refactor separately
   - Commit Window refactor separately
   - Commit Container refactor separately
   - Allows granular rollback if issues discovered

2. **Feature flag approach** (if needed):
   ```go
   const useNewGUIFactories = true  // Set to false to revert

   func CreateTextArea(x, y int) *widget.TextArea {
       if useNewGUIFactories {
           return CreateTextAreaWithConfig(DefaultTextAreaConfig(x, y))
       } else {
           // Old implementation (kept temporarily)
           return legacyCreateTextArea(x, y)
       }
   }
   ```

3. **Rollback triggers**:
   - Visual regressions discovered in UI panels
   - Performance degradation (> 1ms increase in UI initialization)
   - Functional bugs introduced (windows not responding to input)
   - Build failures on team machines (Go version incompatibility unlikely but check)

**Success Metrics**:
1. **LOC reduction**:
   - Target: -70 LOC (TextArea + Window consolidation)
   - Measure: `git diff --stat` before/after

2. **Duplication elimination**:
   - Target: 0 duplicate TextArea creation functions
   - Target: 0 duplicate Window creation blocks
   - Measure: Manual code review

3. **Maintainability improvement**:
   - Target: Change TextArea styling requires editing 1 location (not 3)
   - Measure: Modify font size in all TextAreas - should be 1-line change

4. **Developer velocity**:
   - Target: Creating new TextArea/Window 50% faster
   - Measure: Time to create new stats panel (before: 5 minutes, after: 2 minutes)

### Additional Resources

**Go GUI patterns documentation**:
- Ebiten UI documentation: https://github.com/ebitenui/ebitenui
- Go builder pattern: https://refactoring.guru/design-patterns/builder/go/example
- Effective Go - interfaces: https://go.dev/doc/effective_go#interfaces

**Game architecture references**:
- Game Programming Patterns (Nystrom) - Component pattern chapter
- Entity-Component-System patterns (matches squad system architecture)
- Immediate-mode GUI patterns (Ebiten is immediate-mode like Dear ImGui)

**Refactoring resources**:
- Refactoring: Improving the Design of Existing Code (Fowler)
  - Chapter: Extract Function (applies to factory extraction)
  - Chapter: Replace Constructor with Factory Function
  - Chapter: Introduce Parameter Object (TextAreaConfig pattern)

**Internal codebase patterns to reference**:
- `entitytemplates/creators.go`: EntityConfig → CreateEntityFromTemplate (proven pattern)
- `graphics/drawableshapes.go`: BaseShape consolidation (similar consolidation success)
- `squads/` package: ECS component patterns (if extending to Approach 3)

---

## CONCLUSION

**Summary**: GUI package has 1365 LOC with significant duplication (3 TextArea functions 95% identical, 3 Window creation blocks 85% similar). Three refactoring approaches analyzed:

1. **Approach 1 (Configuration Factory)**: Immediate 70-100 LOC reduction, low risk, proven pattern. **RECOMMENDED FOR IMMEDIATE IMPLEMENTATION.**

2. **Approach 2 (Builder Pattern)**: Ergonomic enhancement, optional addition to Approach 1. **RECOMMENDED IF ACTIVELY ADDING NEW GUI ELEMENTS.**

3. **Approach 3 (Component Composition)**: Architectural investment for major GUI expansion. **RECOMMENDED ONLY IF SQUAD UI REQUIRES 5+ NEW PANELS.**

**Recommended path**: Implement Approach 1 now (6-8 hours), document Approach 3 design for future, evaluate Approach 2 when GUI expansion begins.

**Next action**: Begin TextArea consolidation (highest impact, lowest risk) following Phase 1 implementation strategy.

---

END OF ANALYSIS
