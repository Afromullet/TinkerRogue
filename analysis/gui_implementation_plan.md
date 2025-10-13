# Implementation Plan: GUI Architecture Redesign
Generated: 2025-10-13 13:18:32
Feature: Comprehensive GUI system redesign for tactical roguelike
Coordinated By: implementation-synth

---

## EXECUTIVE SUMMARY

### Feature Overview
- **What**: Complete architectural redesign of GUI system (1365 LOC) to support tactical squad-based gameplay with future squad management UI, tactical overlays, and formation editors
- **Why**: Current monolithic PlayerUI design doesn't scale for complex tactical interfaces. Need architecture that supports 5+ squad panels, threat range overlays, formation editors, and rich tactical information density
- **Inspired By**: Fire Emblem (context-sensitive panels), XCOM (tactical overlay system), Jagged Alliance (squad management), Dear ImGui (immediate-mode patterns)
- **Complexity**: Architectural - Major system redesign with 3 fundamentally different paradigms

### Quick Assessment
- **Recommended Approach**: Plan 2 (ECS-Integrated Component System) for perfect alignment with existing squad system architecture
- **Implementation Time**: 24-32 hours (3-4 workdays)
- **Risk Level**: High (architectural change touches all UI code)
- **Blockers**: None - can implement immediately

### Consensus Findings
- **Agreement Across Agents**:
  - Current monolithic PlayerUI doesn't support squad UI complexity (5+ panels, dynamic information)
  - Configuration-based refactoring (existing analysis) is insufficient for tactical gameplay expansion
  - Need component-based architecture for reusability and testability
  - Performance not a concern (GUI is not hot path, immediate-mode is efficient)
  - ECS alignment beneficial (squad system already perfect ECS, UI should follow)

- **Divergent Perspectives**:
  - **Tactical view**: Prioritizes information density, context-driven panels, spatial overlays
  - **Technical view**: Prioritizes Go idioms, ECS integration, performance patterns
  - **Critical assessment**: Both views compatible - ECS components support tactical needs

- **Key Tradeoffs**:
  - **Gameplay vs performance**: Tactical information density requires rendering complexity, but Ebiten handles this efficiently
  - **Complexity vs features**: Component architecture adds LOC upfront but enables squad UI without exponential complexity growth
  - **Migration cost vs long-term value**: 24-32 hour investment prevents 100+ hour technical debt when adding squad UI

---

## FINAL SYNTHESIZED IMPLEMENTATION PLANS

### Plan 1: Context-Driven Modal UI System (Tactical Gameplay-First)

**Strategic Focus**: Different UI modes for different gameplay contexts (exploration, combat, squad management, formation editing)

**Gameplay Value**:
Players experience streamlined interfaces optimized for specific gameplay moments. In exploration mode, minimal UI shows stats and messages. In combat mode, threat ranges and action options dominate. In squad management mode, unit positions and formations become primary focus. This reduces cognitive load by showing only relevant information.

**Go Standards Compliance**:
Uses state machine pattern with interface-based modal controllers. Each mode is a controller implementing UIMode interface. Clean separation of concerns via interfaces, explicit state transitions, no global mutable state.

**Architecture Overview**:
```
┌─────────────────────────────────────────────────────────┐
│                    UIModeManager                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │ Exploration  │  │   Combat     │  │ SquadManage  │ │
│  │    Mode      │  │    Mode      │  │    Mode      │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────┘
           │                  │                  │
           ▼                  ▼                  ▼
     ┌──────────┐      ┌──────────┐      ┌──────────┐
     │ Minimal  │      │ Tactical │      │  Squad   │
     │ Panels   │      │ Overlay  │      │  Panels  │
     └──────────┘      └──────────┘      └──────────┘
```

**Code Example**:

*Core Structure:*
```go
package gui

import (
    "game_main/common"
    "github.com/hajimehoshi/ebiten/v2"
)

// UIMode - Interface for different UI modal states
type UIMode interface {
    Enter(ctx *ModeContext) error      // Called when transitioning to this mode
    Exit(ctx *ModeContext) error       // Called when leaving this mode
    Update(ctx *UpdateContext) error   // Called every frame
    Render(screen *ebiten.Image) error // Called every frame after Update
    HandleInput(input *InputEvent) (ModeTransition, error) // Returns next mode if transitioning
}

// ModeTransition - Defines mode change
type ModeTransition struct {
    NextMode UIMode
    ShouldTransition bool
}

// ModeContext - Data available during mode transitions
type ModeContext struct {
    PreviousMode UIMode
    Reason       string // Why transition occurred
    GameState    *common.GameState
}

// UpdateContext - Data available during update
type UpdateContext struct {
    PlayerData   *avatar.PlayerData
    ECSManager   *ecs.Manager
    DeltaTime    float64
    GameState    *common.GameState
}

// InputEvent - Unified input representation
type InputEvent struct {
    Type      InputType  // KeyPress, MouseClick, etc.
    Key       ebiten.Key
    MouseX    int
    MouseY    int
    Consumed  bool
}

type InputType int
const (
    InputKeyPress InputType = iota
    InputMouseClick
    InputMouseMove
    InputMouseWheel
)

// UIModeManager - Coordinates mode transitions
type UIModeManager struct {
    currentMode UIMode
    modes       map[string]UIMode // Named modes for easy lookup
    gameState   *common.GameState
}

func NewUIModeManager(gameState *common.GameState) *UIModeManager {
    mgr := &UIModeManager{
        modes:     make(map[string]UIMode),
        gameState: gameState,
    }

    // Register all modes
    mgr.modes["exploration"] = NewExplorationMode()
    mgr.modes["combat"] = NewCombatMode()
    mgr.modes["squad_management"] = NewSquadManagementMode()
    mgr.modes["formation_editor"] = NewFormationEditorMode()

    // Start in exploration mode
    mgr.currentMode = mgr.modes["exploration"]
    mgr.currentMode.Enter(&ModeContext{GameState: gameState})

    return mgr
}

func (mgr *UIModeManager) Update(ctx *UpdateContext) error {
    return mgr.currentMode.Update(ctx)
}

func (mgr *UIModeManager) Render(screen *ebiten.Image) error {
    return mgr.currentMode.Render(screen)
}

func (mgr *UIModeManager) HandleInput(input *InputEvent) error {
    transition, err := mgr.currentMode.HandleInput(input)
    if err != nil {
        return err
    }

    if transition.ShouldTransition {
        return mgr.TransitionTo(transition.NextMode, "player_action")
    }

    return nil
}

func (mgr *UIModeManager) TransitionTo(nextMode UIMode, reason string) error {
    ctx := &ModeContext{
        PreviousMode: mgr.currentMode,
        Reason:       reason,
        GameState:    mgr.gameState,
    }

    // Exit current mode
    if err := mgr.currentMode.Exit(ctx); err != nil {
        return fmt.Errorf("failed to exit mode: %w", err)
    }

    // Enter next mode
    if err := nextMode.Enter(ctx); err != nil {
        return fmt.Errorf("failed to enter mode: %w", err)
    }

    mgr.currentMode = nextMode
    return nil
}

// ExplorationMode - Minimal UI for exploration
type ExplorationMode struct {
    statsPanel    *StatsPanel
    messagePanel  *MessagePanel
    inventoryBtn  *Button
}

func NewExplorationMode() *ExplorationMode {
    return &ExplorationMode{
        statsPanel:   NewStatsPanel(),
        messagePanel: NewMessagePanel(),
        inventoryBtn: NewButton("Inventory"),
    }
}

func (em *ExplorationMode) Enter(ctx *ModeContext) error {
    // Initialize minimal UI elements
    em.statsPanel.Show()
    em.messagePanel.Show()
    return nil
}

func (em *ExplorationMode) Exit(ctx *ModeContext) error {
    // Clean up if needed
    return nil
}

func (em *ExplorationMode) Update(ctx *UpdateContext) error {
    em.statsPanel.Update(ctx)
    em.messagePanel.Update(ctx)
    return nil
}

func (em *ExplorationMode) Render(screen *ebiten.Image) error {
    em.statsPanel.Render(screen)
    em.messagePanel.Render(screen)
    em.inventoryBtn.Render(screen)
    return nil
}

func (em *ExplorationMode) HandleInput(input *InputEvent) (ModeTransition, error) {
    // Check for mode transition inputs
    if input.Type == InputKeyPress && input.Key == ebiten.KeyS {
        // Transition to squad management
        return ModeTransition{
            NextMode:         NewSquadManagementMode(),
            ShouldTransition: true,
        }, nil
    }

    // Handle regular input
    em.inventoryBtn.HandleClick(input)

    return ModeTransition{ShouldTransition: false}, nil
}

// CombatMode - Rich tactical UI for combat
type CombatMode struct {
    statsPanel      *StatsPanel
    threatOverlay   *ThreatRangeOverlay
    actionMenu      *ActionMenu
    targetReticle   *TargetReticle
    coverIndicator  *CoverIndicator
}

func NewCombatMode() *CombatMode {
    return &CombatMode{
        statsPanel:     NewStatsPanel(),
        threatOverlay:  NewThreatRangeOverlay(),
        actionMenu:     NewActionMenu(),
        targetReticle:  NewTargetReticle(),
        coverIndicator: NewCoverIndicator(),
    }
}

func (cm *CombatMode) Enter(ctx *ModeContext) error {
    // Initialize combat UI
    cm.threatOverlay.Calculate(ctx.GameState)
    cm.actionMenu.BuildActions(ctx.GameState.CurrentUnit)
    return nil
}

func (cm *CombatMode) Exit(ctx *ModeContext) error {
    cm.threatOverlay.Clear()
    return nil
}

func (cm *CombatMode) Update(ctx *UpdateContext) error {
    cm.statsPanel.Update(ctx)
    cm.threatOverlay.Update(ctx)
    cm.actionMenu.Update(ctx)
    cm.targetReticle.Update(ctx)
    cm.coverIndicator.Update(ctx)
    return nil
}

func (cm *CombatMode) Render(screen *ebiten.Image) error {
    // Render in layers: overlays → panels → menus → reticle
    cm.threatOverlay.Render(screen)
    cm.coverIndicator.Render(screen)
    cm.statsPanel.Render(screen)
    cm.actionMenu.Render(screen)
    cm.targetReticle.Render(screen)
    return nil
}

func (cm *CombatMode) HandleInput(input *InputEvent) (ModeTransition, error) {
    // Check for mode transition inputs
    if input.Type == InputKeyPress && input.Key == ebiten.KeyEscape {
        // Transition back to exploration
        return ModeTransition{
            NextMode:         NewExplorationMode(),
            ShouldTransition: true,
        }, nil
    }

    // Handle combat input
    cm.actionMenu.HandleClick(input)
    cm.targetReticle.HandleMove(input)

    return ModeTransition{ShouldTransition: false}, nil
}

// SquadManagementMode - Full squad UI with formations
type SquadManagementMode struct {
    squadPanels     []*SquadPanel      // One panel per squad
    formationEditor *FormationEditor
    unitList        *UnitList
    abilityPanel    *AbilityPanel
}

func NewSquadManagementMode() *SquadManagementMode {
    return &SquadManagementMode{
        squadPanels:     make([]*SquadPanel, 0),
        formationEditor: NewFormationEditor(),
        unitList:        NewUnitList(),
        abilityPanel:    NewAbilityPanel(),
    }
}

func (sm *SquadManagementMode) Enter(ctx *ModeContext) error {
    // Query ECS for all squads
    squads := sm.loadSquadsFromECS(ctx.GameState.ECSManager)

    // Create panel for each squad
    for _, squad := range squads {
        panel := NewSquadPanel(squad)
        sm.squadPanels = append(sm.squadPanels, panel)
    }

    return nil
}

func (sm *SquadManagementMode) Exit(ctx *ModeContext) error {
    // Cleanup squad panels
    sm.squadPanels = nil
    return nil
}

func (sm *SquadManagementMode) Update(ctx *UpdateContext) error {
    for _, panel := range sm.squadPanels {
        panel.Update(ctx)
    }
    sm.formationEditor.Update(ctx)
    sm.unitList.Update(ctx)
    sm.abilityPanel.Update(ctx)
    return nil
}

func (sm *SquadManagementMode) Render(screen *ebiten.Image) error {
    // Render squad panels in grid layout
    for i, panel := range sm.squadPanels {
        panel.RenderAt(screen, i*300, 0) // Offset each panel
    }
    sm.formationEditor.Render(screen)
    sm.unitList.Render(screen)
    sm.abilityPanel.Render(screen)
    return nil
}

func (sm *SquadManagementMode) HandleInput(input *InputEvent) (ModeTransition, error) {
    // Check for mode transition inputs
    if input.Type == InputKeyPress && input.Key == ebiten.KeyEscape {
        return ModeTransition{
            NextMode:         NewExplorationMode(),
            ShouldTransition: true,
        }, nil
    }

    // Handle squad management input
    for _, panel := range sm.squadPanels {
        panel.HandleClick(input)
    }
    sm.formationEditor.HandleInput(input)

    return ModeTransition{ShouldTransition: false}, nil
}

func (sm *SquadManagementMode) loadSquadsFromECS(manager *ecs.Manager) []SquadData {
    // Query ECS for squad entities using existing squad system
    // This integrates with squads/squadqueries.go functions
    // Implementation details connect to existing squad system
    return []SquadData{} // Placeholder
}
```

**Implementation Steps**:

1. **Core Mode System (8h)**
   - What: Implement UIMode interface, UIModeManager, ModeContext, UpdateContext
   - Files: `gui/modes.go` (new), `gui/mode_manager.go` (new)
   - Code:
     ```go
     // Define UIMode interface with Enter/Exit/Update/Render/HandleInput
     // Implement UIModeManager with mode registry and transitions
     // Create InputEvent and ModeTransition types
     ```

2. **Exploration Mode (4h)**
   - What: Minimal UI mode with stats, messages, basic buttons
   - Files: `gui/exploration_mode.go` (new)
   - Code:
     ```go
     // ExplorationMode implements UIMode
     // Reuse existing StatsPanel and MessagePanel (from refactoring)
     // Simple inventory button triggers throwables window
     ```

3. **Combat Mode (8h)**
   - What: Tactical UI with threat overlays, action menus, targeting
   - Files: `gui/combat_mode.go` (new), `gui/overlays.go` (new)
   - Code:
     ```go
     // CombatMode implements UIMode
     // ThreatRangeOverlay renders grid cells with threat levels
     // ActionMenu shows context-sensitive combat actions
     // TargetReticle shows selected target with hit chances
     ```

4. **Squad Management Mode (10h)**
   - What: Multi-squad UI with formation editor, unit lists, ability panels
   - Files: `gui/squad_management_mode.go` (new), `gui/formation_editor.go` (new)
   - Code:
     ```go
     // SquadManagementMode implements UIMode
     // Query ECS for all squads using squads/squadqueries.go
     // SquadPanel shows 3x3 grid for each squad (reuse squads/visualization.go)
     // FormationEditor allows drag-and-drop unit positioning
     ```

5. **Integration with Game Loop (2h)**
   - What: Replace PlayerUI with UIModeManager in main game loop
   - Files: `game_main/main.go`, `game_main/game.go`
   - Code:
     ```go
     // Replace: playerUI := gui.CreatePlayerUI(...)
     // With: uiManager := gui.NewUIModeManager(gameState)
     // Call: uiManager.Update(ctx) and uiManager.Render(screen)
     ```

**Tactical Design Analysis**:
- **Tactical Depth**: Each mode optimized for specific gameplay moment. Combat mode shows threat ranges/cover/actions without clutter from exploration UI. Squad management mode dedicates full screen to formation editing and unit management.
- **Genre Alignment**: Mirrors Fire Emblem's context-sensitive UI (map mode vs combat mode vs unit management). Matches XCOM's tactical overlay system.
- **Balance Impact**: Reduced cognitive load improves tactical decision-making. Players see only relevant information, reducing analysis paralysis.
- **Counter-play**: Threat overlay shows enemy movement/attack ranges, enabling defensive positioning. Cover indicator shows safe vs dangerous cells.

**Go Standards Analysis**:
- **Idiomatic Patterns**: State machine via interface-based modes. Explicit state transitions (no hidden state). Composition over inheritance (modes compose panels, not inherit).
- **Performance**: Immediate-mode rendering (Render() called every frame). Minimal allocations (modes allocated once during NewUIModeManager). Hot path optimized (Update/Render methods are simple delegations).
- **Error Handling**: Explicit error returns from Enter/Exit/Update/Render. Mode transitions can fail gracefully.
- **Testing Strategy**: Each mode testable in isolation (mock ModeContext and UpdateContext). Transition logic unit testable.

**Key Benefits**:
- **Gameplay**: Context-appropriate UI reduces cognitive load. Players see only relevant tactical information for current gameplay moment.
- **Code Quality**: Clear separation between modes. Each mode is self-contained and testable. No global state or mode confusion.
- **Performance**: Immediate-mode rendering efficient (only active mode renders). No hidden allocations or state management overhead.

**Drawbacks & Risks**:
- **Gameplay**: Mode transitions might feel jarring if not smoothed with animations. Risk of disorienting players during rapid mode changes.
  - *Mitigation*: Add fade transitions between modes. Clear visual indicators of current mode. Allow quick toggle keys.
- **Technical**: Potential code duplication if modes share panels (e.g., StatsPanel in multiple modes). Risk of mode proliferation (too many modes = complexity).
  - *Mitigation*: Share panel implementations across modes (composition). Limit to 4-5 core modes.
- **Performance**: Mode transitions allocate new mode objects. Risk of GC pressure if transitioning rapidly.
  - *Mitigation*: Pool mode objects or reuse them. Mode transitions infrequent (seconds, not frames).

**Effort Estimate**:
- **Time**: 32 hours (4 workdays)
- **Complexity**: Medium-High (state machine pattern requires careful design)
- **Risk**: Medium (mode transitions must be bulletproof, testing critical)
- **Files Impacted**: 3 existing (main.go, game.go, playerUI.go)
- **New Files**: 8 (modes.go, mode_manager.go, exploration_mode.go, combat_mode.go, squad_management_mode.go, formation_editor.go, overlays.go, input_events.go)

**Integration Points**:
- **Existing Squad System**: SquadManagementMode queries ECS using `squads/squadqueries.go` functions. Visualizes squads using `squads/visualization.go`.
- **Input System**: InputCoordinator routes input to UIModeManager.HandleInput(). Current input controllers (MovementController, CombatController) integrate with Combat mode.
- **ECS Manager**: Each mode receives ECSManager via UpdateContext. Modes query entities as needed.
- **Graphics System**: Overlays (threat ranges, cover) use existing `graphics/drawableshapes.go` BaseShape system.

**Critical Assessment** (from implementation-critic):
This approach has strong tactical gameplay value - context-driven UI directly addresses squad management complexity. Modal design is sound (state machine with interface-based transitions). However, potential for code duplication across modes (multiple modes may need similar panels). Risk of over-abstraction if modes become too granular (mode explosion). Best suited if game has distinct gameplay states that benefit from dedicated UI modes. Consider if tactical gameplay truly requires mode separation or if dynamic panel visibility suffices.

---

### Plan 2: ECS-Integrated Component System (Balanced Architecture)

**Strategic Focus**: UI elements as ECS components, unified with existing squad system architecture

**Gameplay Value**:
Players experience consistent data-driven UI that responds dynamically to game state changes. Squad panels automatically update when unit positions change. Tactical overlays recalculate when enemy units move. This creates seamless reactive UI that feels like natural extension of game world, not separate interface layer.

**Go Standards Compliance**:
Perfect ECS compliance following proven squad system patterns. Pure data components (UIComponent, PanelComponent, OverlayComponent), system-based rendering (GUIRenderSystem), query-based relationships (no stored entity pointers). Native ecs.EntityID usage throughout. Matches existing squad system architecture exactly.

**Architecture Overview**:
```
┌──────────────────────────────────────────────────────┐
│              ECS Manager (Unified)                   │
│  ┌───────────────────────────────────────────────┐  │
│  │  Entities: Squads, Units, UI Panels, Overlays │  │
│  └───────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
  ┌──────────┐  ┌──────────┐  ┌──────────┐
  │  Squad   │  │    UI    │  │ Overlay  │
  │ System   │  │  System  │  │ System   │
  └──────────┘  └──────────┘  └──────────┘
        │             │             │
        ▼             ▼             ▼
  Squad Data    Panel Data    Overlay Data
  (position)    (bounds)      (threat range)
```

**Code Example**:

*Core Structure:*
```go
package gui

import (
    "game_main/common"
    "game_main/ecs"
    "github.com/hajimehoshi/ebiten/v2"
)

// ========================================
// UI COMPONENTS (Pure Data, Zero Logic)
// ========================================

// UIComponentData - Base UI component (attached to all UI entities)
type UIComponentData struct {
    Visible    bool
    ZIndex     int     // Rendering layer (0=background, 100=foreground)
    Interactive bool   // Does this respond to input?
}

// BoundsComponentData - Position and size (pixel coordinates)
type BoundsComponentData struct {
    X      int
    Y      int
    Width  int
    Height int
}

// AnchorComponentData - Declarative positioning relative to screen
type AnchorComponentData struct {
    AnchorX AnchorType  // Start, Center, End
    AnchorY AnchorType
    OffsetX int
    OffsetY int
}

type AnchorType int
const (
    AnchorStart AnchorType = iota
    AnchorCenter
    AnchorEnd
)

// PanelComponentData - Panel-specific configuration
type PanelComponentData struct {
    PanelType  PanelType
    DataSource ecs.EntityID // Entity this panel displays (e.g., squad entity)
    AutoUpdate bool         // Recalculate content every frame?
}

type PanelType int
const (
    PanelStats PanelType = iota
    PanelMessages
    PanelSquad
    PanelAbilities
    PanelInventory
)

// OverlayComponentData - Overlay-specific configuration
type OverlayComponentData struct {
    OverlayType   OverlayType
    SourceEntity  ecs.EntityID // Entity generating overlay (e.g., unit entity)
    GridPositions []coords.LogicalPosition // Grid cells to highlight
    Color         color.RGBA
    Alpha         uint8
}

type OverlayType int
const (
    OverlayThreatRange OverlayType = iota
    OverlayMovementRange
    OverlayCoverZones
    OverlayTargetingReticle
)

// WidgetComponentData - Ebiten UI widget reference
type WidgetComponentData struct {
    Widget     widget.PreferredSizeLocateableWidget // Ebiten UI widget
    NeedsUpdate bool // Flag for lazy updates
}

// TextComponentData - Text display
type TextComponentData struct {
    Text     string
    FontFace font.Face
    Color    color.RGBA
}

// ButtonComponentData - Button-specific data
type ButtonComponentData struct {
    Text         string
    ClickHandler func(*ButtonClickContext) // Called when button clicked
    Enabled      bool
}

type ButtonClickContext struct {
    ButtonEntity ecs.EntityID
    GameState    *common.GameState
}

// ========================================
// UI SYSTEM (Rendering and Logic)
// ========================================

// GUIRenderSystem - Renders all UI entities
type GUIRenderSystem struct {
    manager        *ecs.Manager
    screen         *ebiten.Image
    widgetCache    map[ecs.EntityID]widget.PreferredSizeLocateableWidget
    ebitenUI       *ebitenui.UI
}

func NewGUIRenderSystem(manager *ecs.Manager) *GUIRenderSystem {
    return &GUIRenderSystem{
        manager:     manager,
        widgetCache: make(map[ecs.EntityID]widget.PreferredSizeLocateableWidget),
        ebitenUI:    &ebitenui.UI{},
    }
}

// Initialize - Register components with ECS
func (grs *GUIRenderSystem) Initialize() error {
    // Register UI components
    grs.manager.RegisterComponent(&UIComponentData{})
    grs.manager.RegisterComponent(&BoundsComponentData{})
    grs.manager.RegisterComponent(&AnchorComponentData{})
    grs.manager.RegisterComponent(&PanelComponentData{})
    grs.manager.RegisterComponent(&OverlayComponentData{})
    grs.manager.RegisterComponent(&WidgetComponentData{})
    grs.manager.RegisterComponent(&TextComponentData{})
    grs.manager.RegisterComponent(&ButtonComponentData{})

    return nil
}

// Update - Update UI entities (called every frame)
func (grs *GUIRenderSystem) Update(deltaTime float64, gameState *common.GameState) error {
    // Query all UI entities
    uiEntities := grs.queryUIEntities()

    for _, entityID := range uiEntities {
        entity := grs.manager.GetEntity(entityID)
        if entity == nil {
            continue
        }

        uiComp := entity.GetComponent(&UIComponentData{}).(*UIComponentData)
        if !uiComp.Visible {
            continue
        }

        // Update anchored positions
        if anchorComp := entity.GetComponent(&AnchorComponentData{}); anchorComp != nil {
            grs.updateAnchoredPosition(entity)
        }

        // Update panels that auto-refresh
        if panelComp := entity.GetComponent(&PanelComponentData{}); panelComp != nil {
            if panelComp.AutoUpdate {
                grs.updatePanelContent(entity, gameState)
            }
        }

        // Update overlays based on source entity
        if overlayComp := entity.GetComponent(&OverlayComponentData{}); overlayComp != nil {
            grs.updateOverlayData(entity, gameState)
        }
    }

    return nil
}

// Render - Render all UI entities (called every frame after Update)
func (grs *GUIRenderSystem) Render(screen *ebiten.Image) error {
    grs.screen = screen

    // Query UI entities sorted by ZIndex
    uiEntities := grs.queryUIEntitiesSorted()

    for _, entityID := range uiEntities {
        entity := grs.manager.GetEntity(entityID)
        if entity == nil {
            continue
        }

        uiComp := entity.GetComponent(&UIComponentData{}).(*UIComponentData)
        if !uiComp.Visible {
            continue
        }

        // Render based on component types
        if overlayComp := entity.GetComponent(&OverlayComponentData{}); overlayComp != nil {
            grs.renderOverlay(entity, screen)
        }

        if widgetComp := entity.GetComponent(&WidgetComponentData{}); widgetComp != nil {
            grs.renderWidget(entity, screen)
        }

        if textComp := entity.GetComponent(&TextComponentData{}); textComp != nil {
            grs.renderText(entity, screen)
        }
    }

    return nil
}

// HandleInput - Route input to interactive UI entities
func (grs *GUIRenderSystem) HandleInput(input *InputEvent) error {
    // Query interactive UI entities
    interactiveEntities := grs.queryInteractiveEntities()

    for _, entityID := range interactiveEntities {
        entity := grs.manager.GetEntity(entityID)
        if entity == nil {
            continue
        }

        // Check if input is within bounds
        boundsComp := entity.GetComponent(&BoundsComponentData{}).(*BoundsComponentData)
        if !grs.isInputInBounds(input, boundsComp) {
            continue
        }

        // Handle button clicks
        if buttonComp := entity.GetComponent(&ButtonComponentData{}); buttonComp != nil {
            if buttonComp.ClickHandler != nil && input.Type == InputMouseClick {
                ctx := &ButtonClickContext{
                    ButtonEntity: entityID,
                    GameState:    grs.gameState,
                }
                buttonComp.ClickHandler(ctx)
                input.Consumed = true
                return nil
            }
        }
    }

    return nil
}

// ========================================
// QUERY FUNCTIONS (ECS Pattern)
// ========================================

func (grs *GUIRenderSystem) queryUIEntities() []ecs.EntityID {
    // Query all entities with UIComponentData
    results := make([]ecs.EntityID, 0)
    for _, entity := range grs.manager.Entities {
        if entity.HasComponent(&UIComponentData{}) {
            results = append(results, entity.ID)
        }
    }
    return results
}

func (grs *GUIRenderSystem) queryUIEntitiesSorted() []ecs.EntityID {
    entities := grs.queryUIEntities()

    // Sort by ZIndex (low to high = back to front)
    sort.Slice(entities, func(i, j int) bool {
        entityI := grs.manager.GetEntity(entities[i])
        entityJ := grs.manager.GetEntity(entities[j])

        uiCompI := entityI.GetComponent(&UIComponentData{}).(*UIComponentData)
        uiCompJ := entityJ.GetComponent(&UIComponentData{}).(*UIComponentData)

        return uiCompI.ZIndex < uiCompJ.ZIndex
    })

    return entities
}

func (grs *GUIRenderSystem) queryInteractiveEntities() []ecs.EntityID {
    results := make([]ecs.EntityID, 0)
    for _, entity := range grs.manager.Entities {
        if uiComp := entity.GetComponent(&UIComponentData{}); uiComp != nil {
            if uiComp.(*UIComponentData).Interactive {
                results = append(results, entity.ID)
            }
        }
    }
    return results
}

// ========================================
// HELPER FUNCTIONS (Rendering Logic)
// ========================================

func (grs *GUIRenderSystem) updateAnchoredPosition(entity *ecs.Entity) {
    anchorComp := entity.GetComponent(&AnchorComponentData{}).(*AnchorComponentData)
    boundsComp := entity.GetComponent(&BoundsComponentData{}).(*BoundsComponentData)

    // Calculate position based on anchor
    screenWidth := graphics.ScreenInfo.GetCanvasWidth()
    screenHeight := graphics.ScreenInfo.GetCanvasHeight()

    var x, y int

    switch anchorComp.AnchorX {
    case AnchorStart:
        x = anchorComp.OffsetX
    case AnchorCenter:
        x = screenWidth/2 - boundsComp.Width/2 + anchorComp.OffsetX
    case AnchorEnd:
        x = screenWidth - boundsComp.Width + anchorComp.OffsetX
    }

    switch anchorComp.AnchorY {
    case AnchorStart:
        y = anchorComp.OffsetY
    case AnchorCenter:
        y = screenHeight/2 - boundsComp.Height/2 + anchorComp.OffsetY
    case AnchorEnd:
        y = screenHeight - boundsComp.Height + anchorComp.OffsetY
    }

    boundsComp.X = x
    boundsComp.Y = y
}

func (grs *GUIRenderSystem) updatePanelContent(entity *ecs.Entity, gameState *common.GameState) {
    panelComp := entity.GetComponent(&PanelComponentData{}).(*PanelComponentData)

    switch panelComp.PanelType {
    case PanelStats:
        // Update stats panel text from player data
        textComp := entity.GetComponent(&TextComponentData{}).(*TextComponentData)
        textComp.Text = gameState.PlayerData.PlayerAttributes().DisplayString()

    case PanelSquad:
        // Update squad panel from squad entity
        if panelComp.DataSource != 0 {
            squadEntity := grs.manager.GetEntity(panelComp.DataSource)
            if squadEntity != nil {
                // Use existing squad visualization
                visualization := squads.VisualizeSquad(panelComp.DataSource, grs.manager)
                textComp := entity.GetComponent(&TextComponentData{}).(*TextComponentData)
                textComp.Text = visualization
            }
        }

    case PanelMessages:
        // Update message log from game state
        textComp := entity.GetComponent(&TextComponentData{}).(*TextComponentData)
        textComp.Text = gameState.MessageLog.GetRecentMessages(10)
    }

    // Mark widget for update
    if widgetComp := entity.GetComponent(&WidgetComponentData{}); widgetComp != nil {
        widgetComp.(*WidgetComponentData).NeedsUpdate = true
    }
}

func (grs *GUIRenderSystem) updateOverlayData(entity *ecs.Entity, gameState *common.GameState) {
    overlayComp := entity.GetComponent(&OverlayComponentData{}).(*OverlayComponentData)

    switch overlayComp.OverlayType {
    case OverlayThreatRange:
        // Calculate threat range from source unit
        if overlayComp.SourceEntity != 0 {
            threatCells := grs.calculateThreatRange(overlayComp.SourceEntity, gameState)
            overlayComp.GridPositions = threatCells
        }

    case OverlayMovementRange:
        // Calculate movement range from source unit
        if overlayComp.SourceEntity != 0 {
            movementCells := grs.calculateMovementRange(overlayComp.SourceEntity, gameState)
            overlayComp.GridPositions = movementCells
        }

    case OverlayCoverZones:
        // Calculate cover zones from all units
        coverCells := grs.calculateCoverZones(gameState)
        overlayComp.GridPositions = coverCells
    }
}

func (grs *GUIRenderSystem) renderOverlay(entity *ecs.Entity, screen *ebiten.Image) {
    overlayComp := entity.GetComponent(&OverlayComponentData{}).(*OverlayComponentData)

    // Render each grid cell with overlay color
    for _, gridPos := range overlayComp.GridPositions {
        pixelPos := coords.GlobalCoordManager.GridToPixel(gridPos)

        // Use existing BaseShape system from graphics/drawableshapes.go
        shape := graphics.NewRectangularShape(
            pixelPos,
            graphics.ScreenInfo.TileSize,
            graphics.ScreenInfo.TileSize,
            overlayComp.Color,
        )
        shape.SetAlpha(overlayComp.Alpha)
        shape.Draw(screen)
    }
}

func (grs *GUIRenderSystem) renderWidget(entity *ecs.Entity, screen *ebiten.Image) {
    widgetComp := entity.GetComponent(&WidgetComponentData{}).(*WidgetComponentData)
    boundsComp := entity.GetComponent(&BoundsComponentData{}).(*BoundsComponentData)

    // Update widget position if needed
    if widgetComp.NeedsUpdate {
        r := image.Rect(boundsComp.X, boundsComp.Y,
            boundsComp.X+boundsComp.Width,
            boundsComp.Y+boundsComp.Height)
        widgetComp.Widget.SetLocation(r)
        widgetComp.NeedsUpdate = false
    }

    // Ebiten UI handles actual widget rendering
    // (widgets added to ebitenUI.Container during entity creation)
}

func (grs *GUIRenderSystem) renderText(entity *ecs.Entity, screen *ebiten.Image) {
    textComp := entity.GetComponent(&TextComponentData{}).(*TextComponentData)
    boundsComp := entity.GetComponent(&BoundsComponentData{}).(*BoundsComponentData)

    // Use ebitenutil.DebugPrintAt or text.Draw for rendering
    text.Draw(screen, textComp.Text, textComp.FontFace,
        boundsComp.X, boundsComp.Y, textComp.Color)
}

// ========================================
// ENTITY CREATION FACTORIES
// ========================================

// CreateStatsPanel - Create stats panel entity
func CreateStatsPanel(manager *ecs.Manager, playerData *avatar.PlayerData) ecs.EntityID {
    entity := manager.NewEntity()

    // Add UI components
    entity.AddComponent(&UIComponentData{
        Visible:     true,
        ZIndex:      10,
        Interactive: false,
    })

    entity.AddComponent(&AnchorComponentData{
        AnchorX: AnchorEnd,
        AnchorY: AnchorStart,
        OffsetX: -10,
        OffsetY: 10,
    })

    entity.AddComponent(&BoundsComponentData{
        Width:  graphics.StatsUIOffset,
        Height: graphics.ScreenInfo.LevelHeight / 4,
    })

    entity.AddComponent(&PanelComponentData{
        PanelType:  PanelStats,
        AutoUpdate: true, // Update every frame
    })

    entity.AddComponent(&TextComponentData{
        Text:     playerData.PlayerAttributes().DisplayString(),
        FontFace: smallFace,
        Color:    color.RGBA{255, 255, 255, 255},
    })

    // Create Ebiten UI widget
    config := DefaultTextAreaConfig(graphics.StatsUIOffset, graphics.ScreenInfo.LevelHeight/4)
    widget := CreateTextAreaWithConfig(config)
    widget.SetText(playerData.PlayerAttributes().DisplayString())

    entity.AddComponent(&WidgetComponentData{
        Widget:      widget,
        NeedsUpdate: false,
    })

    return entity.ID
}

// CreateSquadPanel - Create squad panel entity
func CreateSquadPanel(manager *ecs.Manager, squadID ecs.EntityID, offsetX int) ecs.EntityID {
    entity := manager.NewEntity()

    entity.AddComponent(&UIComponentData{
        Visible:     true,
        ZIndex:      10,
        Interactive: true,
    })

    entity.AddComponent(&BoundsComponentData{
        X:      offsetX,
        Y:      100,
        Width:  300,
        Height: 400,
    })

    entity.AddComponent(&PanelComponentData{
        PanelType:  PanelSquad,
        DataSource: squadID, // Link to squad entity
        AutoUpdate: true,
    })

    // Use squad visualization from squads/visualization.go
    visualization := squads.VisualizeSquad(squadID, manager)

    entity.AddComponent(&TextComponentData{
        Text:     visualization,
        FontFace: smallFace,
        Color:    color.RGBA{255, 255, 255, 255},
    })

    // Create widget
    config := DefaultTextAreaConfig(300, 400)
    widget := CreateTextAreaWithConfig(config)
    widget.SetText(visualization)

    entity.AddComponent(&WidgetComponentData{
        Widget:      widget,
        NeedsUpdate: false,
    })

    return entity.ID
}

// CreateThreatRangeOverlay - Create threat range overlay entity
func CreateThreatRangeOverlay(manager *ecs.Manager, unitEntityID ecs.EntityID) ecs.EntityID {
    entity := manager.NewEntity()

    entity.AddComponent(&UIComponentData{
        Visible:     true,
        ZIndex:      5, // Below panels, above game world
        Interactive: false,
    })

    entity.AddComponent(&OverlayComponentData{
        OverlayType:   OverlayThreatRange,
        SourceEntity:  unitEntityID,
        GridPositions: []coords.LogicalPosition{}, // Calculated during Update
        Color:         color.RGBA{255, 0, 0, 128}, // Semi-transparent red
        Alpha:         128,
    })

    return entity.ID
}

// CreateInventoryButton - Create inventory button entity
func CreateInventoryButton(manager *ecs.Manager, playerUI *PlayerUI, inventory *gear.Inventory) ecs.EntityID {
    entity := manager.NewEntity()

    entity.AddComponent(&UIComponentData{
        Visible:     true,
        ZIndex:      20, // Foreground
        Interactive: true,
    })

    entity.AddComponent(&AnchorComponentData{
        AnchorX: AnchorCenter,
        AnchorY: AnchorStart,
        OffsetX: 0,
        OffsetY: 10,
    })

    entity.AddComponent(&BoundsComponentData{
        Width:  200,
        Height: 60,
    })

    entity.AddComponent(&ButtonComponentData{
        Text:    "Throwables",
        Enabled: true,
        ClickHandler: func(ctx *ButtonClickContext) {
            // Open throwables window
            playerUI.ItemsUI.ThrowableItemDisplay.DisplayInventory()
        },
    })

    // Create Ebiten UI button widget
    button := CreateButton("Throwables")
    button.Configure(
        widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
            playerUI.ItemsUI.ThrowableItemDisplay.DisplayInventory()
        }),
    )

    entity.AddComponent(&WidgetComponentData{
        Widget:      button,
        NeedsUpdate: false,
    })

    return entity.ID
}
```

**Implementation Steps**:

1. **Component Definitions (4h)**
   - What: Define all UI component types (pure data structures)
   - Files: `gui/components.go` (new)
   - Code:
     ```go
     // Define UIComponentData, BoundsComponentData, AnchorComponentData,
     // PanelComponentData, OverlayComponentData, WidgetComponentData,
     // TextComponentData, ButtonComponentData (all pure data)
     ```

2. **GUIRenderSystem Core (6h)**
   - What: Implement rendering system with Update/Render/HandleInput
   - Files: `gui/gui_render_system.go` (new)
   - Code:
     ```go
     // GUIRenderSystem struct with manager, screen, widgetCache
     // Initialize() registers components
     // Update() updates anchored positions, panels, overlays
     // Render() renders in ZIndex order
     // HandleInput() routes to interactive entities
     ```

3. **Query Functions (2h)**
   - What: ECS query functions for UI entities
   - Files: `gui/gui_queries.go` (new)
   - Code:
     ```go
     // queryUIEntities() - all entities with UIComponentData
     // queryUIEntitiesSorted() - sorted by ZIndex
     // queryInteractiveEntities() - entities with Interactive=true
     // queryPanelsByType() - panels of specific type
     ```

4. **Entity Factories (6h)**
   - What: Factory functions for common UI entities
   - Files: `gui/ui_factories.go` (new)
   - Code:
     ```go
     // CreateStatsPanel() - stats panel entity
     // CreateSquadPanel() - squad panel entity
     // CreateThreatRangeOverlay() - threat overlay entity
     // CreateInventoryButton() - inventory button entity
     // CreateMessagePanel() - message log panel entity
     ```

5. **Overlay Rendering (4h)**
   - What: Implement overlay rendering and calculations
   - Files: `gui/overlays.go` (new)
   - Code:
     ```go
     // calculateThreatRange() - calculate threat cells from unit
     // calculateMovementRange() - calculate movement cells
     // calculateCoverZones() - calculate cover cells
     // renderOverlay() - render grid cell overlays
     ```

6. **Integration with Game Loop (2h)**
   - What: Replace PlayerUI with GUIRenderSystem
   - Files: `game_main/main.go`, `game_main/game.go`
   - Code:
     ```go
     // Initialize: guiSystem := gui.NewGUIRenderSystem(ecsManager)
     // Update: guiSystem.Update(deltaTime, gameState)
     // Render: guiSystem.Render(screen)
     // Input: guiSystem.HandleInput(inputEvent)
     ```

**Tactical Design Analysis**:
- **Tactical Depth**: Dynamic overlays show threat ranges, movement zones, cover automatically. Players see real-time tactical information as game state changes. Squad panels update instantly when units move or take damage.
- **Genre Alignment**: Matches Fire Emblem's reactive UI (enemy phase shows all enemy ranges simultaneously). Mirrors XCOM's overlay system (threat rings, cover indicators).
- **Balance Impact**: Transparent overlays reduce guesswork in tactical planning. Players make informed decisions based on visible threat data.
- **Counter-play**: Threat overlays show enemy capabilities, enabling defensive positioning. Movement overlays show escape routes.

**Go Standards Analysis**:
- **Idiomatic Patterns**: Perfect ECS compliance - pure data components, system-based logic, query-based relationships. Mirrors proven squad system architecture (squads/components.go, squads/squadqueries.go).
- **Performance**: Query-based rendering (only visible entities rendered). Widget caching prevents re-creation. ZIndex sorting happens once per frame. Minimal allocations (components allocated during entity creation).
- **Error Handling**: Explicit error returns from Update/Render. Nil checks for entity queries. Graceful degradation if components missing.
- **Testing Strategy**: Components testable in isolation (pure data). GUIRenderSystem testable with mock ECSManager. Entity factories testable independently.

**Key Benefits**:
- **Gameplay**: Reactive UI responds automatically to game state changes. No manual synchronization between UI and game world. Squad panels update when squad data changes (no polling).
- **Code Quality**: Perfect alignment with existing squad system architecture. UI follows same ECS patterns as combat, movement, abilities. Consistent component model across entire codebase.
- **Performance**: ECS query performance (O(n) iteration over entities). Widget caching prevents re-creation overhead. Immediate-mode rendering efficient.

**Drawbacks & Risks**:
- **Gameplay**: Overlay complexity might cause visual clutter if too many overlays active simultaneously. Risk of information overload.
  - *Mitigation*: Limit active overlays (e.g., only show threat range for selected unit). Use transparency/color coding to distinguish overlays.
- **Technical**: ECS query overhead if UI entities numerous (100+ entities). Risk of over-querying in Update loop. Widget lifecycle management complexity (when to create/destroy widgets?).
  - *Mitigation*: Cache query results. Use dirty flags (NeedsUpdate) to skip unnecessary updates. Widget pooling if needed.
- **Performance**: ZIndex sorting every frame (O(n log n)). If 100+ UI entities, sorting cost increases. Widget rendering overhead if Ebiten UI not optimized.
  - *Mitigation*: Only sort when ZIndex changes (track dirty flag). Limit UI entities (most games have 10-30 UI elements). Benchmark widget rendering.

**Effort Estimate**:
- **Time**: 24 hours (3 workdays)
- **Complexity**: Medium (ECS pattern proven in squad system, direct application)
- **Risk**: Low-Medium (ECS architecture well-understood, but UI lifecycle complexity)
- **Files Impacted**: 3 existing (main.go, game.go, playerUI.go)
- **New Files**: 6 (components.go, gui_render_system.go, gui_queries.go, ui_factories.go, overlays.go, input_handler.go)

**Integration Points**:
- **Existing Squad System**: Squad panels query squad entities using `squads/squadqueries.go`. Visualize using `squads/visualization.go`. Perfect integration - squad data flows naturally to UI entities.
- **ECS Manager**: UI components registered alongside squad components. Single unified ECS Manager for entire game. All systems query same entity pool.
- **Position System**: Overlays use `common.GlobalPositionSystem` for spatial queries. Threat range calculations use O(1) position lookups.
- **Graphics System**: Overlays render using `graphics/drawableshapes.go` BaseShape system. Consistent rendering across UI and game world.

**Critical Assessment** (from implementation-critic):
This approach has highest architectural value - perfect ECS alignment with squad system eliminates conceptual mismatch. Data-driven UI naturally supports dynamic tactical information (threat overlays, squad updates). Query-based relationships prevent tight coupling. Code reuse high (squad visualization, position queries, shape rendering). Performance acceptable (ECS queries efficient at 10-30 UI entities). Risk is lifecycle management (when to create/destroy UI entities?) and potential over-querying. Best approach if prioritizing architectural consistency and future scalability. Recommended for this codebase given proven squad system success.

---

### Plan 3: Functional Renderer Pattern (Go Minimalist)

**Strategic Focus**: Pure functions generate UI from game state each frame, minimal abstraction over Ebiten

**Gameplay Value**:
Players experience zero-latency UI that always reflects exact game state. No synchronization bugs (UI always matches game world because UI IS game world representation). Simple mental model: game state → render functions → screen. Easy to reason about what UI shows at any moment.

**Go Standards Compliance**:
Extreme Go simplicity - pure functions, minimal interfaces, explicit over implicit. No ECS coupling, no complex systems. Just data structures and functions. Matches Go philosophy: "Clear is better than clever." Easy to test (pure functions), easy to understand (no hidden state), easy to debug (linear execution).

**Architecture Overview**:
```
┌───────────────────────────────────────────┐
│           Game State                      │
│  (PlayerData, Squads, Entities, Map)     │
└───────────────────────────────────────────┘
                  │
                  ▼
┌───────────────────────────────────────────┐
│      RenderUI(state, screen)              │
│  ┌─────────────────────────────────────┐ │
│  │  renderStats(state.Player, screen)  │ │
│  │  renderSquads(state.Squads, screen) │ │
│  │  renderOverlays(state.Combat, ...)  │ │
│  └─────────────────────────────────────┘ │
└───────────────────────────────────────────┘
                  │
                  ▼
            ┌──────────┐
            │  Screen  │
            └──────────┘
```

**Code Example**:

*Core Structure:*
```go
package gui

import (
    "game_main/avatar"
    "game_main/common"
    "game_main/squads"
    "github.com/hajimehoshi/ebiten/v2"
    "image/color"
)

// ========================================
// DATA STRUCTURES (Pure Data)
// ========================================

// GameUIState - All data needed to render UI
type GameUIState struct {
    Player      *avatar.PlayerData
    Squads      []SquadDisplayData
    Combat      *CombatUIState
    Messages    []string
    ScreenWidth int
    ScreenHeight int
}

// SquadDisplayData - Squad data formatted for UI
type SquadDisplayData struct {
    SquadID     ecs.EntityID
    Name        string
    Units       []UnitDisplayData
    Formation   [3][3]ecs.EntityID // 3x3 grid positions
    OffsetX     int // Rendering position
    OffsetY     int
}

// UnitDisplayData - Unit data formatted for UI
type UnitDisplayData struct {
    UnitID   ecs.EntityID
    Name     string
    HP       int
    MaxHP    int
    Row      int
    Col      int
    IsLeader bool
}

// CombatUIState - Combat-specific UI state
type CombatUIState struct {
    InCombat         bool
    ActiveUnit       ecs.EntityID
    ThreatRanges     []ThreatRangeData
    MovementRange    []coords.LogicalPosition
    SelectedTarget   ecs.EntityID
    AvailableActions []ActionData
}

// ThreatRangeData - Threat range display data
type ThreatRangeData struct {
    SourceUnit  ecs.EntityID
    Cells       []coords.LogicalPosition
    Color       color.RGBA
    Intensity   uint8 // 0-255 for fade effect
}

// ActionData - Combat action display data
type ActionData struct {
    Name        string
    Icon        *ebiten.Image
    Enabled     bool
    HotKey      ebiten.Key
    Description string
}

// ========================================
// MAIN RENDER FUNCTION (Entry Point)
// ========================================

// RenderUI - Main UI rendering function (called every frame)
func RenderUI(state *GameUIState, screen *ebiten.Image) error {
    // Render in layers (back to front)

    // Layer 0: Overlays (rendered on game world)
    if state.Combat != nil && state.Combat.InCombat {
        if err := renderCombatOverlays(state.Combat, screen); err != nil {
            return err
        }
    }

    // Layer 1: Background panels
    if err := renderStatsPanel(state.Player, screen); err != nil {
        return err
    }

    if err := renderMessagesPanel(state.Messages, screen); err != nil {
        return err
    }

    // Layer 2: Squad panels (if any squads exist)
    if len(state.Squads) > 0 {
        if err := renderSquadPanels(state.Squads, screen); err != nil {
            return err
        }
    }

    // Layer 3: Foreground (menus, dialogs)
    if state.Combat != nil && state.Combat.InCombat {
        if err := renderCombatActions(state.Combat, screen); err != nil {
            return err
        }
    }

    return nil
}

// ========================================
// PANEL RENDERING FUNCTIONS (Pure Functions)
// ========================================

// renderStatsPanel - Render player stats panel
func renderStatsPanel(player *avatar.PlayerData, screen *ebiten.Image) error {
    // Panel bounds
    x := graphics.ScreenInfo.GetCanvasWidth() - graphics.StatsUIOffset
    y := 10
    width := graphics.StatsUIOffset - 20
    height := graphics.ScreenInfo.LevelHeight / 4

    // Background
    drawPanelBackground(screen, x, y, width, height, color.RGBA{19, 26, 34, 255})

    // Stats text
    statsText := player.PlayerAttributes().DisplayString()
    drawText(screen, statsText, x+10, y+10, color.White)

    return nil
}

// renderMessagesPanel - Render message log panel
func renderMessagesPanel(messages []string, screen *ebiten.Image) error {
    // Panel bounds
    x := graphics.ScreenInfo.GetCanvasWidth() - graphics.StatsUIOffset
    y := graphics.ScreenInfo.LevelHeight/4 + 20
    width := graphics.StatsUIOffset - 20
    height := graphics.ScreenInfo.LevelHeight / 4

    // Background
    drawPanelBackground(screen, x, y, width, height, color.RGBA{19, 26, 34, 255})

    // Messages text (last 10 messages)
    messageText := strings.Join(messages[max(0, len(messages)-10):], "\n")
    drawText(screen, messageText, x+10, y+10, color.White)

    return nil
}

// renderSquadPanels - Render all squad panels
func renderSquadPanels(squads []SquadDisplayData, screen *ebiten.Image) error {
    for i, squad := range squads {
        if err := renderSquadPanel(squad, screen); err != nil {
            return err
        }
    }
    return nil
}

// renderSquadPanel - Render single squad panel
func renderSquadPanel(squad SquadDisplayData, screen *ebiten.Image) error {
    x := squad.OffsetX
    y := squad.OffsetY
    width := 300
    height := 400

    // Background
    drawPanelBackground(screen, x, y, width, height, color.RGBA{19, 26, 34, 255})

    // Squad name
    drawText(screen, squad.Name, x+10, y+10, color.White)

    // 3x3 grid visualization
    gridX := x + 20
    gridY := y + 40
    cellSize := 80

    for row := 0; row < 3; row++ {
        for col := 0; col < 3; col++ {
            unitID := squad.Formation[row][col]

            cellX := gridX + col*cellSize
            cellY := gridY + row*cellSize

            if unitID == 0 {
                // Empty cell
                drawGridCell(screen, cellX, cellY, cellSize, color.RGBA{50, 50, 50, 128}, "")
            } else {
                // Find unit data
                unit := findUnitByID(squad.Units, unitID)
                if unit != nil {
                    // Occupied cell
                    cellColor := color.RGBA{100, 150, 200, 255}
                    if unit.IsLeader {
                        cellColor = color.RGBA{200, 150, 100, 255} // Gold for leader
                    }
                    drawGridCell(screen, cellX, cellY, cellSize, cellColor, unit.Name)

                    // HP bar
                    hpPercent := float64(unit.HP) / float64(unit.MaxHP)
                    drawHPBar(screen, cellX, cellY+cellSize-10, cellSize, 5, hpPercent)
                }
            }
        }
    }

    return nil
}

// ========================================
// OVERLAY RENDERING FUNCTIONS (Pure Functions)
// ========================================

// renderCombatOverlays - Render all combat overlays
func renderCombatOverlays(combat *CombatUIState, screen *ebiten.Image) error {
    // Render threat ranges
    for _, threatRange := range combat.ThreatRanges {
        if err := renderThreatRange(threatRange, screen); err != nil {
            return err
        }
    }

    // Render movement range
    if len(combat.MovementRange) > 0 {
        if err := renderMovementRange(combat.MovementRange, screen); err != nil {
            return err
        }
    }

    // Render target reticle
    if combat.SelectedTarget != 0 {
        if err := renderTargetReticle(combat.SelectedTarget, screen); err != nil {
            return err
        }
    }

    return nil
}

// renderThreatRange - Render single threat range overlay
func renderThreatRange(threatRange ThreatRangeData, screen *ebiten.Image) error {
    for _, gridPos := range threatRange.Cells {
        pixelPos := coords.GlobalCoordManager.GridToPixel(gridPos)

        // Semi-transparent colored rectangle
        drawOverlayCell(screen, pixelPos.X, pixelPos.Y,
            graphics.ScreenInfo.TileSize, graphics.ScreenInfo.TileSize,
            threatRange.Color, threatRange.Intensity)
    }
    return nil
}

// renderMovementRange - Render movement range overlay
func renderMovementRange(cells []coords.LogicalPosition, screen *ebiten.Image) error {
    for _, gridPos := range cells {
        pixelPos := coords.GlobalCoordManager.GridToPixel(gridPos)

        // Blue semi-transparent overlay
        drawOverlayCell(screen, pixelPos.X, pixelPos.Y,
            graphics.ScreenInfo.TileSize, graphics.ScreenInfo.TileSize,
            color.RGBA{0, 100, 255, 128}, 128)
    }
    return nil
}

// renderTargetReticle - Render targeting reticle
func renderTargetReticle(targetID ecs.EntityID, screen *ebiten.Image) error {
    // Get target position
    targetEntity := common.ECSManager.GetEntity(targetID)
    if targetEntity == nil {
        return nil
    }

    posComp := targetEntity.GetComponent(&common.PositionData{})
    if posComp == nil {
        return nil
    }

    pos := posComp.(*common.PositionData).LogicalPosition
    pixelPos := coords.GlobalCoordManager.GridToPixel(pos)

    // Draw reticle (square outline)
    drawReticle(screen, pixelPos.X, pixelPos.Y,
        graphics.ScreenInfo.TileSize, graphics.ScreenInfo.TileSize,
        color.RGBA{255, 255, 0, 255}, 3) // Yellow, 3px thick

    return nil
}

// renderCombatActions - Render combat action menu
func renderCombatActions(combat *CombatUIState, screen *ebiten.Image) error {
    // Action menu position (bottom center)
    menuX := graphics.ScreenInfo.GetCanvasWidth()/2 - 200
    menuY := graphics.ScreenInfo.GetCanvasHeight() - 100
    menuWidth := 400
    menuHeight := 80

    // Background
    drawPanelBackground(screen, menuX, menuY, menuWidth, menuHeight,
        color.RGBA{19, 26, 34, 240})

    // Render actions in horizontal row
    actionSpacing := menuWidth / len(combat.AvailableActions)
    for i, action := range combat.AvailableActions {
        actionX := menuX + i*actionSpacing + 10
        actionY := menuY + 10

        // Action button
        buttonColor := color.RGBA{100, 100, 100, 255}
        if action.Enabled {
            buttonColor = color.RGBA{150, 150, 200, 255}
        }

        drawActionButton(screen, actionX, actionY, 60, 60, buttonColor,
            action.Name, action.Icon)

        // Hotkey indicator
        if action.HotKey != 0 {
            hotkeyText := fmt.Sprintf("[%s]", action.HotKey.String())
            drawText(screen, hotkeyText, actionX, actionY+70, color.Gray{Y: 128})
        }
    }

    return nil
}

// ========================================
// DATA PREPARATION FUNCTIONS (State → DisplayData)
// ========================================

// PrepareGameUIState - Convert game state to UI state
func PrepareGameUIState(gameState *common.GameState) *GameUIState {
    uiState := &GameUIState{
        Player:       gameState.PlayerData,
        ScreenWidth:  graphics.ScreenInfo.GetCanvasWidth(),
        ScreenHeight: graphics.ScreenInfo.GetCanvasHeight(),
    }

    // Prepare squad display data
    uiState.Squads = prepareSquadDisplayData(gameState.ECSManager)

    // Prepare combat UI state (if in combat)
    if gameState.InCombat {
        uiState.Combat = prepareCombatUIState(gameState)
    }

    // Prepare message log
    uiState.Messages = gameState.MessageLog.GetAllMessages()

    return uiState
}

// prepareSquadDisplayData - Convert ECS squad entities to display data
func prepareSquadDisplayData(manager *ecs.Manager) []SquadDisplayData {
    displayData := make([]SquadDisplayData, 0)

    // Query all squads from ECS using existing squad system
    // This uses squads/squadqueries.go functions
    for _, entity := range manager.Entities {
        if !entity.HasComponent(&squads.SquadData{}) {
            continue
        }

        squadComp := entity.GetComponent(&squads.SquadData{}).(*squads.SquadData)

        // Get all units in squad
        unitIDs := squads.GetUnitIDsInSquad(entity.ID, manager)

        // Build unit display data
        units := make([]UnitDisplayData, 0)
        formation := [3][3]ecs.EntityID{}

        for _, unitID := range unitIDs {
            unitEntity := manager.GetEntity(unitID)
            if unitEntity == nil {
                continue
            }

            // Get unit components
            memberComp := unitEntity.GetComponent(&squads.SquadMemberData{}).(*squads.SquadMemberData)
            statsComp := unitEntity.GetComponent(&squads.UnitStatsData{}).(*squads.UnitStatsData)
            gridComp := unitEntity.GetComponent(&squads.GridPositionData{}).(*squads.GridPositionData)

            // Build unit display data
            unitDisplay := UnitDisplayData{
                UnitID:   unitID,
                Name:     memberComp.UnitName,
                HP:       statsComp.CurrentHP,
                MaxHP:    statsComp.MaxHP,
                Row:      gridComp.Row,
                Col:      gridComp.Col,
                IsLeader: unitEntity.HasComponent(&squads.LeaderTag{}),
            }

            units = append(units, unitDisplay)

            // Place in formation grid
            formation[gridComp.Row][gridComp.Col] = unitID
        }

        // Build squad display data
        displayData = append(displayData, SquadDisplayData{
            SquadID:   entity.ID,
            Name:      squadComp.Name,
            Units:     units,
            Formation: formation,
            OffsetX:   len(displayData) * 320, // Offset each squad panel
            OffsetY:   50,
        })
    }

    return displayData
}

// prepareCombatUIState - Prepare combat UI state
func prepareCombatUIState(gameState *common.GameState) *CombatUIState {
    combat := &CombatUIState{
        InCombat:         true,
        ActiveUnit:       gameState.ActiveUnit,
        ThreatRanges:     []ThreatRangeData{},
        MovementRange:    []coords.LogicalPosition{},
        SelectedTarget:   gameState.SelectedTarget,
        AvailableActions: []ActionData{},
    }

    // Calculate threat ranges for all enemy units
    for _, entity := range gameState.ECSManager.Entities {
        if entity.HasComponent(&common.EnemyTag{}) {
            threatRange := calculateUnitThreatRange(entity.ID, gameState)
            combat.ThreatRanges = append(combat.ThreatRanges, threatRange)
        }
    }

    // Calculate movement range for active unit
    if combat.ActiveUnit != 0 {
        combat.MovementRange = calculateUnitMovementRange(combat.ActiveUnit, gameState)
    }

    // Build available actions
    combat.AvailableActions = buildAvailableActions(combat.ActiveUnit, gameState)

    return combat
}

// ========================================
// LOW-LEVEL DRAWING PRIMITIVES (Helpers)
// ========================================

func drawPanelBackground(screen *ebiten.Image, x, y, width, height int, bgColor color.RGBA) {
    // Use existing shape system
    shape := graphics.NewRectangularShape(
        coords.PixelPosition{X: x, Y: y},
        width, height, bgColor,
    )
    shape.Draw(screen)
}

func drawText(screen *ebiten.Image, text string, x, y int, textColor color.Color) {
    // Use ebitenutil.DebugPrintAt or text.Draw
    ebitenutil.DebugPrintAt(screen, text, x, y)
}

func drawGridCell(screen *ebiten.Image, x, y, size int, cellColor color.RGBA, label string) {
    // Cell background
    shape := graphics.NewRectangularShape(
        coords.PixelPosition{X: x, Y: y},
        size, size, cellColor,
    )
    shape.Draw(screen)

    // Cell border
    drawRectangleOutline(screen, x, y, size, size, color.White, 2)

    // Label
    if label != "" {
        drawText(screen, label, x+5, y+5, color.White)
    }
}

func drawHPBar(screen *ebiten.Image, x, y, width, height int, hpPercent float64) {
    // Background (red)
    bgShape := graphics.NewRectangularShape(
        coords.PixelPosition{X: x, Y: y},
        width, height, color.RGBA{255, 0, 0, 255},
    )
    bgShape.Draw(screen)

    // Foreground (green, proportional to HP)
    fgWidth := int(float64(width) * hpPercent)
    fgShape := graphics.NewRectangularShape(
        coords.PixelPosition{X: x, Y: y},
        fgWidth, height, color.RGBA{0, 255, 0, 255},
    )
    fgShape.Draw(screen)
}

func drawOverlayCell(screen *ebiten.Image, x, y, width, height int, overlayColor color.RGBA, alpha uint8) {
    shape := graphics.NewRectangularShape(
        coords.PixelPosition{X: x, Y: y},
        width, height, overlayColor,
    )
    shape.SetAlpha(alpha)
    shape.Draw(screen)
}

func drawReticle(screen *ebiten.Image, x, y, width, height int, lineColor color.RGBA, thickness int) {
    // Draw four corner brackets
    // Top-left corner
    drawRectangle(screen, x, y, 10, thickness, lineColor) // Horizontal
    drawRectangle(screen, x, y, thickness, 10, lineColor) // Vertical

    // Top-right corner
    drawRectangle(screen, x+width-10, y, 10, thickness, lineColor)
    drawRectangle(screen, x+width-thickness, y, thickness, 10, lineColor)

    // Bottom-left corner
    drawRectangle(screen, x, y+height-thickness, 10, thickness, lineColor)
    drawRectangle(screen, x, y+height-10, thickness, 10, lineColor)

    // Bottom-right corner
    drawRectangle(screen, x+width-10, y+height-thickness, 10, thickness, lineColor)
    drawRectangle(screen, x+width-thickness, y+height-10, thickness, 10, lineColor)
}

func drawActionButton(screen *ebiten.Image, x, y, width, height int, btnColor color.RGBA, label string, icon *ebiten.Image) {
    // Button background
    shape := graphics.NewRectangularShape(
        coords.PixelPosition{X: x, Y: y},
        width, height, btnColor,
    )
    shape.Draw(screen)

    // Icon (if provided)
    if icon != nil {
        op := &ebiten.DrawImageOptions{}
        op.GeoM.Translate(float64(x+10), float64(y+10))
        screen.DrawImage(icon, op)
    }

    // Label
    drawText(screen, label, x+5, y+height-15, color.White)
}

func drawRectangle(screen *ebiten.Image, x, y, width, height int, rectColor color.RGBA) {
    shape := graphics.NewRectangularShape(
        coords.PixelPosition{X: x, Y: y},
        width, height, rectColor,
    )
    shape.Draw(screen)
}

func drawRectangleOutline(screen *ebiten.Image, x, y, width, height int, lineColor color.Color, thickness int) {
    // Top
    drawRectangle(screen, x, y, width, thickness, lineColor.(color.RGBA))
    // Bottom
    drawRectangle(screen, x, y+height-thickness, width, thickness, lineColor.(color.RGBA))
    // Left
    drawRectangle(screen, x, y, thickness, height, lineColor.(color.RGBA))
    // Right
    drawRectangle(screen, x+width-thickness, y, thickness, height, lineColor.(color.RGBA))
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func findUnitByID(units []UnitDisplayData, unitID ecs.EntityID) *UnitDisplayData {
    for i := range units {
        if units[i].UnitID == unitID {
            return &units[i]
        }
    }
    return nil
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
```

**Implementation Steps**:

1. **Data Structures (2h)**
   - What: Define all UI display data structures (pure data)
   - Files: `gui/ui_data.go` (new)
   - Code:
     ```go
     // Define GameUIState, SquadDisplayData, UnitDisplayData,
     // CombatUIState, ThreatRangeData, ActionData (all pure data)
     ```

2. **Data Preparation Functions (4h)**
   - What: Convert ECS/game state to UI display data
   - Files: `gui/ui_preparation.go` (new)
   - Code:
     ```go
     // PrepareGameUIState() - main conversion function
     // prepareSquadDisplayData() - query squads from ECS
     // prepareCombatUIState() - build combat UI state
     // buildAvailableActions() - build action menu
     ```

3. **Panel Rendering Functions (4h)**
   - What: Implement panel rendering (stats, messages, squads)
   - Files: `gui/panel_rendering.go` (new)
   - Code:
     ```go
     // renderStatsPanel() - render player stats
     // renderMessagesPanel() - render message log
     // renderSquadPanels() - render all squad panels
     // renderSquadPanel() - render single squad panel
     ```

4. **Overlay Rendering Functions (4h)**
   - What: Implement overlay rendering (threat, movement, targeting)
   - Files: `gui/overlay_rendering.go` (new)
   - Code:
     ```go
     // renderCombatOverlays() - render all overlays
     // renderThreatRange() - render threat range cells
     // renderMovementRange() - render movement range cells
     // renderTargetReticle() - render targeting reticle
     ```

5. **Drawing Primitives (2h)**
   - What: Low-level drawing helper functions
   - Files: `gui/drawing_primitives.go` (new)
   - Code:
     ```go
     // drawPanelBackground(), drawText(), drawGridCell(),
     // drawHPBar(), drawOverlayCell(), drawReticle(),
     // drawActionButton(), drawRectangle(), drawRectangleOutline()
     ```

6. **Integration with Game Loop (2h)**
   - What: Replace PlayerUI with functional rendering
   - Files: `game_main/main.go`, `game_main/game.go`
   - Code:
     ```go
     // In Update(): uiState := gui.PrepareGameUIState(gameState)
     // In Draw(): gui.RenderUI(uiState, screen)
     // No UI state stored between frames
     ```

**Tactical Design Analysis**:
- **Tactical Depth**: Threat overlays calculated fresh every frame from current game state. Players always see up-to-date tactical information. No synchronization lag between game state changes and UI updates.
- **Genre Alignment**: Matches Fire Emblem's immediate feedback (enemy phase threat ranges appear instantly). Mirrors XCOM's reactive overlays (cover zones recalculate as units move).
- **Balance Impact**: Zero-latency UI ensures players make decisions on accurate information. No "stale UI" bugs where displayed HP doesn't match actual HP.
- **Counter-play**: Threat overlays recalculate when enemies move, showing new dangerous zones. Movement range updates when terrain changes (doors open/close).

**Go Standards Analysis**:
- **Idiomatic Patterns**: Pure functions everywhere. No global mutable state. Data flows one direction (game state → display data → screen). Explicit over implicit. Clear is better than clever.
- **Performance**: Zero allocation in render path (display data prepared once). Immediate-mode rendering (no retained state). Simple function calls (no vtable lookups). Cache-friendly (linear data structures).
- **Error Handling**: Explicit error returns from render functions. Nil checks for entity lookups. Graceful degradation (missing data → skip rendering).
- **Testing Strategy**: Pure functions trivially testable. Mock game state → call render function → verify screen output. No ECS dependency in tests (just data structures).

**Key Benefits**:
- **Gameplay**: Zero synchronization bugs. UI always matches game state because UI IS game state representation. No hidden state, no staleness.
- **Code Quality**: Extreme simplicity. No abstractions, no interfaces, no systems. Just data and functions. Easy to understand, easy to debug, easy to maintain.
- **Performance**: Fastest rendering approach. No ECS queries, no component lookups, just function calls. Display data prepared once, rendering is trivial.

**Drawbacks & Risks**:
- **Gameplay**: No state persistence between frames. Modal dialogs/windows require explicit state management in game state. Risk of forgetting to add dialog state to GameUIState.
  - *Mitigation*: Add DialogState to GameUIState. Document that all UI state must live in game state.
- **Technical**: Data preparation overhead if game state large (converting ECS to display data every frame). Risk of duplicate code in render functions. No code reuse between panels.
  - *Mitigation*: Optimize preparation functions (cache queries). Extract common rendering patterns into helper functions.
- **Performance**: Preparation functions query ECS every frame. If 100+ squads, query cost increases. Display data allocation every frame (GC pressure).
  - *Mitigation*: Pool display data structures. Only query changed entities (dirty tracking). Benchmark preparation functions.

**Effort Estimate**:
- **Time**: 18 hours (2-3 workdays)
- **Complexity**: Low (pure functions, no abstractions, straightforward)
- **Risk**: Low (no complex systems, easy to understand and test)
- **Files Impacted**: 2 existing (main.go, game.go)
- **New Files**: 5 (ui_data.go, ui_preparation.go, panel_rendering.go, overlay_rendering.go, drawing_primitives.go)

**Integration Points**:
- **Existing Squad System**: Preparation functions query squad entities using `squads/squadqueries.go`. Format squad data for display. No tight coupling - squad system is just data source.
- **ECS Manager**: Preparation functions query ECS for entities. No component registration, no systems. ECS is read-only from UI perspective.
- **Position System**: Overlay rendering uses `common.GlobalPositionSystem` for spatial queries. Threat range calculations use O(1) position lookups.
- **Graphics System**: All rendering uses `graphics/drawableshapes.go` BaseShape system. Consistent rendering primitives.

**Critical Assessment** (from implementation-critic):
This approach has highest simplicity value - pure functional rendering is easiest to understand and test. No abstractions means no over-engineering risk. Performance excellent (no ECS overhead, just function calls). However, lacks code reuse (panels can't share logic easily). Data preparation overhead every frame (converting ECS to display data). Modal dialogs require explicit state management in game state (more work for game logic). Best suited for small UI (10-15 panels) or if team prioritizes simplicity over reusability. Not recommended if planning 50+ UI elements (preparation overhead grows linearly). Consider hybrid: functional rendering for simple panels, component architecture for complex panels.

---

## COMPARATIVE ANALYSIS OF FINAL PLANS

### Effort vs Impact Matrix
| Plan | Tactical Depth | Go Quality | Performance | Risk | Time | Priority |
|------|---------------|------------|-------------|------|------|----------|
| Plan 1 (Modal UI) | M | M | M | M | 32h | 3 |
| Plan 2 (ECS-Integrated) | H | H | H | L-M | 24h | 1 |
| Plan 3 (Functional) | M | H | H | L | 18h | 2 |

### Decision Guidance

**Choose Plan 1 (Context-Driven Modal UI System) if:**
- **Primary goal**: Different gameplay contexts require dramatically different interfaces
- **Game design**: Clear mode separation (exploration mode vs combat mode vs squad management mode)
- **Tactical requirement**: Context-appropriate information density (show only relevant data per gameplay moment)
- **Team preference**: Developers comfortable with state machine patterns and mode transitions
- **Feature roadmap**: Planning multiple distinct gameplay modes with unique UI requirements

**Choose Plan 2 (ECS-Integrated Component System) if:**
- **Primary goal**: Architectural consistency with existing squad system (perfect ECS alignment)
- **Code consistency**: Want UI to follow same patterns as combat, movement, and squad systems
- **Tactical requirement**: Dynamic reactive UI that updates automatically when game state changes
- **Team experience**: Developers already mastered ECS architecture through squad system implementation
- **Feature roadmap**: Planning squad management UI with 5+ dynamic panels that query ECS data
- **RECOMMENDED**: This is the best fit for the current codebase given the proven squad system success

**Choose Plan 3 (Functional Renderer Pattern) if:**
- **Primary goal**: Maximum simplicity and minimal abstraction
- **Code philosophy**: "Clear is better than clever" - pure functions over complex systems
- **Tactical requirement**: Zero-latency UI that always reflects exact game state
- **Team preference**: Developers prefer explicit code over abstractions and systems
- **Feature roadmap**: UI scope limited (10-20 panels), no plans for massive expansion

### Combination Opportunities

**Hybrid: Plan 2 (Core) + Plan 3 (Overlays)**
- **Implementation**: Use ECS-integrated components for panels (stats, squads), functional rendering for overlays (threat ranges, movement zones)
- **Rationale**: Panels benefit from ECS reactivity, overlays benefit from functional simplicity
- **Migration**: Implement Plan 2 for all panels, then implement Plan 3 overlay rendering
- **Code example**:
  ```go
  // Panels: ECS entities with components
  statsPanel := gui.CreateStatsPanel(ecsManager, playerData)

  // Overlays: Pure functions
  gui.RenderThreatOverlays(combatState, screen)
  ```

**Phased: Plan 3 (Now) → Plan 2 (Later)**
- **Implementation**: Start with functional rendering (18h), migrate to ECS components when squad UI added
- **Rationale**: Get immediate simple solution, defer architectural investment until needed
- **Trigger**: When implementing squad management UI (5+ panels, complex relationships)
- **Migration path**: Functional render functions become component render methods, display data becomes components

**Hybrid: Plan 1 (Modes) + Plan 2 (Components)**
- **Implementation**: Use modal system for mode transitions, ECS components within each mode
- **Rationale**: Best of both - context-appropriate UI (modal system) with reactive components (ECS)
- **Complexity**: Highest (combines two architectural patterns)
- **Code example**:
  ```go
  // Modes coordinate high-level UI
  type CombatMode struct {
      // ECS components for rendering
      threatOverlayEntity ecs.EntityID
      actionMenuEntity    ecs.EntityID
  }
  ```

---

## APPENDIX: ARCHITECTURE ANALYSIS

### A. Current GUI System Pain Points

**Monolithic PlayerUI Struct (playerUI.go)**:
```go
type PlayerUI struct {
    ItemsUI             PlayerItemsUI
    StatsUI             PlayerStatsUI
    MsgUI               PlayerMessageUI
    InformationUI       InfoUI
    MainPlayerInterface *ebitenui.UI
}
```
**Problems**:
1. Tight coupling - changing one sub-UI requires understanding all sub-UIs
2. No clear ownership - who manages PlayerUI lifecycle?
3. Hard to test - must construct entire PlayerUI to test single panel
4. Hard to extend - adding squad UI requires modifying PlayerUI struct

**Manual Widget Creation (createwidgets.go, statsui.go, messagesUI.go)**:
```go
// 45 lines of duplicate Ebiten UI configuration
func (statsUI *PlayerStatsUI) CreateStatsTextArea() *widget.TextArea {
    return widget.NewTextArea(
        widget.TextAreaOpts.ContainerOpts(...),
        widget.TextAreaOpts.ControlWidgetSpacing(2),
        widget.TextAreaOpts.ProcessBBCode(true),
        // ... 40 more lines
    )
}
```
**Problems**:
1. Ebiten UI verbosity leaked into game code
2. No abstraction layer for common patterns
3. Code duplication (3 TextArea functions, 95% identical)
4. Hard to change styling (must update 3 locations)

**Hardcoded Positioning (playerUI.go:74-78)**:
```go
SetContainerLocation(itemDisplayOptionsContainer,
    graphics.ScreenInfo.GetCanvasWidth()/2, 0)

SetContainerLocation(playerUI.StatsUI.StatUIContainer,
    graphics.ScreenInfo.GetCanvasWidth(), 0)
```
**Problems**:
1. Magic numbers (what does "/2" mean? Center? Right edge?)
2. No responsive layout (breaks if screen size changes)
3. No relative positioning (can't say "below stats panel")
4. Hard to visualize layout (must run game to see positions)

**No Separation of Concerns (playerUI.go:48-85)**:
```go
func CreatePlayerUI(...) *ebitenui.UI {
    // Mixes: construction, layout, positioning, initialization
    rootContainer := widget.NewContainer()
    itemDisplayOptionsContainer := CreateInventorySelectionContainer(...)
    playerUI.StatsUI.CreateStatsUI()
    rootContainer.AddChild(itemDisplayOptionsContainer)
    SetContainerLocation(itemDisplayOptionsContainer, ...)
    // ... all in one 85-line function
}
```
**Problems**:
1. Single function does 5 things (construction, layout, positioning, initialization, wiring)
2. Can't test construction without testing positioning
3. Can't reuse layout logic for different UI hierarchies
4. Hard to understand control flow (linear 85-line function)

### B. Squad UI Requirements (Future Feature)

**Squad Management Interface Requirements**:
Based on squad system (2358 LOC, 85% complete) and TRPG genre conventions:

1. **Multiple Squad Panels** (3-5 squads, each needs dedicated panel):
   - 3x3 grid showing unit positions
   - Unit HP bars
   - Leader indicator
   - Formation name
   - Total squad stats (aggregate HP, morale, abilities)

2. **Formation Editor**:
   - Drag-and-drop unit positioning
   - Formation presets (Balanced, Defensive, Offensive, Ranged)
   - Validation (e.g., leader must be in front row)
   - Preview threat ranges with new formation

3. **Unit Detail Panel**:
   - Selected unit stats (HP, Attack, Defense, Dodge, Crit)
   - Equipped abilities
   - Status effects
   - Grid position (Row/Col)

4. **Tactical Overlay**:
   - Threat ranges for all enemy squads (semi-transparent red cells)
   - Movement ranges for all friendly squads (blue cells)
   - Cover zones (green cells for full cover, yellow for partial)
   - Line-of-sight indicators

5. **Squad Ability Bar**:
   - Leader abilities (Rally, Heal, Battle Cry, Fireball)
   - Trigger conditions (HP thresholds, turn counts, morale levels)
   - Ability cooldowns/charges
   - Hotkey indicators

**Complexity Analysis**:
- **5 new panel types** (squad, formation editor, unit detail, ability bar, tactical overlay)
- **Dynamic data** (squad panels update when units move/die, overlays recalculate each frame)
- **Complex interactions** (drag-and-drop, formation validation, ability triggering)
- **High information density** (3-5 squad panels + overlays + ability bar simultaneously)

**Why Current Architecture Insufficient**:
1. **Monolithic PlayerUI** can't scale to 5+ squad panels (adding SquadUI1, SquadUI2, SquadUI3 to struct is unmaintainable)
2. **Manual positioning** doesn't support dynamic panel count (can't hardcode positions for 1-5 squads)
3. **No component reusability** (each squad panel is nearly identical, but no shared implementation)
4. **No ECS integration** (squad panels need to query squad entities, current UI has no ECS awareness)

### C. ECS Integration Patterns

**Current ECS Usage (Squad System)**:
```go
// Perfect ECS compliance (squads/components.go)
type SquadData struct {
    Name   string
    Leader ecs.EntityID
}

type UnitStatsData struct {
    MaxHP     int
    CurrentHP int
    Attack    int
    Defense   int
}

type GridPositionData struct {
    Row    int
    Col    int
    Width  int
    Height int
}

// Query-based relationships (squads/squadqueries.go)
func GetUnitIDsInSquad(squadID ecs.EntityID, manager *ecs.Manager) []ecs.EntityID {
    // Query entities, no stored pointers
}

func GetLeaderID(squadID ecs.EntityID, manager *ecs.Manager) ecs.EntityID {
    // Query for LeaderTag, no stored reference
}
```

**ECS Integration for UI (Plan 2 Approach)**:
```go
// UI components follow same pattern
type PanelComponentData struct {
    PanelType  PanelType
    DataSource ecs.EntityID // Squad entity to display
    AutoUpdate bool
}

// Query-based UI relationships
func GetPanelsForSquad(squadID ecs.EntityID, manager *ecs.Manager) []ecs.EntityID {
    // Query UI entities displaying this squad
}

// System-based rendering
type GUIRenderSystem struct {
    manager *ecs.Manager
}

func (grs *GUIRenderSystem) Update(deltaTime float64) {
    // Query all UI entities, update based on data source entities
}
```

**Benefits of ECS Integration**:
1. **Unified data model**: Squad entities, unit entities, UI entities all in same ECS Manager
2. **Query-based relationships**: UI panels discover squad data via queries (no stored pointers)
3. **System coordination**: GUIRenderSystem coordinates with SquadSystem, CombatSystem, etc.
4. **Component reuse**: Squad panels are just entities with PanelComponentData + SquadDisplayData
5. **Testing**: UI components testable in isolation (mock ECS Manager, add test entities)

**ECS vs Non-ECS UI Comparison**:
| Aspect | ECS-Integrated (Plan 2) | Non-ECS (Plan 1, Plan 3) |
|--------|------------------------|--------------------------|
| Data model | UI entities = ECS entities | UI data separate from ECS |
| Relationships | Query-based (GetPanelsForSquad) | Manual references or separate tracking |
| Updates | System-based (GUIRenderSystem.Update) | Manual update calls or pure functions |
| Squad integration | Direct ECS queries | Convert ECS data to UI data |
| Code consistency | Matches squad system patterns | Different patterns from squad system |
| Testing | Mock ECS Manager | Mock game state or UI-specific mocks |
| Complexity | Medium (ECS knowledge required) | Low (Plan 3) to Medium (Plan 1) |

### D. Performance Considerations

**Ebiten Rendering Performance**:
- **Immediate-mode**: GUI rendered every frame (60 FPS)
- **Not performance-critical**: Typical tactical roguelike has 10-30 UI elements (not 1000s)
- **Bottlenecks**: Text rendering (if many TextAreas), image blitting (if many icons), not logic/queries

**Plan 1 (Modal UI) Performance**:
- **Update cost**: O(n) where n = UI elements in active mode (typically 5-10)
- **Render cost**: O(n) where n = UI elements in active mode
- **Mode transition cost**: O(1) allocation (new mode object), amortized over seconds
- **Hot path**: Update() and Render() per mode (5-10 element iterations = microseconds)
- **Verdict**: Performance excellent (only active mode rendered, minimal overhead)

**Plan 2 (ECS-Integrated) Performance**:
- **Update cost**: O(m + n) where m = UI entities, n = squad entities to query
- **Render cost**: O(m log m) where m = UI entities (ZIndex sort + render)
- **Query cost**: O(n) iteration over all entities to find UI entities
- **Hot path**: queryUIEntities() → sort by ZIndex → render each entity
- **Allocation**: Component access via GetComponent() (pointer lookups), ZIndex sort (minimal allocations)
- **Verdict**: Performance good at small scale (10-30 UI entities). Optimize with caching if 50+ UI entities.

**Plan 3 (Functional) Performance**:
- **Preparation cost**: O(n) where n = squads (query ECS, build display data)
- **Render cost**: O(m) where m = UI panels to render (pure function calls)
- **Allocation**: Display data allocated every frame (GameUIState, SquadDisplayData[], etc.)
- **Hot path**: PrepareGameUIState() → RenderUI() (linear execution)
- **GC pressure**: Display data allocated/discarded every frame (may trigger GC if large)
- **Verdict**: Performance excellent for rendering (pure functions), potential GC issue if many squads (pool display data)

**Optimization Strategies**:
1. **Widget caching**: Don't re-create Ebiten widgets every frame (store in entity/struct)
2. **Dirty tracking**: Only update UI when underlying data changes (dirty flags)
3. **Spatial culling**: Don't render UI elements outside viewport (not applicable for fixed panels)
4. **Z-index caching**: Only re-sort UI entities when ZIndex changes (track dirty flag)
5. **Query optimization**: Cache ECS query results, invalidate when entities added/removed
6. **Display data pooling**: Reuse SquadDisplayData structs instead of allocating every frame

**Performance Target**:
- **UI update budget**: < 1ms per frame (out of 16.6ms total at 60 FPS)
- **UI render budget**: < 2ms per frame (Ebiten handles widget rendering efficiently)
- **Total UI budget**: < 3ms per frame (18% of frame time)
- **Verdict**: All 3 plans meet performance target at expected scale (10-30 UI elements)

### E. Migration Path from Current GUI

**Migration Complexity**:
| Aspect | Plan 1 (Modal) | Plan 2 (ECS) | Plan 3 (Functional) |
|--------|----------------|--------------|---------------------|
| Files changed | 3 existing + 8 new | 3 existing + 6 new | 2 existing + 5 new |
| PlayerUI refactor | Replace with UIModeManager | Replace with GUIRenderSystem | Remove (render functions only) |
| Widget reuse | High (modes reuse panels) | Medium (components wrap widgets) | Low (widgets created per panel) |
| Incremental migration | Hard (modes are atomic) | Easy (migrate panel by panel) | Medium (migrate render function by function) |
| Rollback risk | High (mode system or nothing) | Low (gradual migration) | Medium (some functions migrated) |
| Testing during migration | Hard (need mode complete) | Easy (test each component) | Easy (test each function) |

**Recommended Migration Strategy (Plan 2)**:
1. **Phase 1: Component definitions + System skeleton (4h)**
   - Define UI component types
   - Implement GUIRenderSystem with empty Update/Render
   - No breaking changes (components unused)

2. **Phase 2: Migrate stats panel (2h)**
   - Create stats panel as ECS entity
   - Test rendering
   - Keep old PlayerUI.StatsUI for fallback

3. **Phase 3: Migrate messages panel (2h)**
   - Create messages panel as ECS entity
   - Test rendering
   - Keep old PlayerUI.MsgUI for fallback

4. **Phase 4: Migrate inventory buttons (2h)**
   - Create button entities
   - Test input handling
   - Remove old CreateInventorySelectionContainer

5. **Phase 5: Remove old PlayerUI (2h)**
   - Delete PlayerUI struct
   - Update main.go to use GUIRenderSystem
   - Full integration testing

6. **Phase 6: Add squad panels (6h)**
   - Implement squad panel entities
   - Integrate with squad system queries
   - Add formation editor panel

**Fallback Strategy**:
- Keep old PlayerUI code in separate file (playerUI_legacy.go)
- Feature flag: `const useNewGUISystem = true`
- If issues found, flip flag to revert to legacy UI
- Remove legacy code after 1 week of stable operation

---

## SYNTHESIS RATIONALE

### Why These 3 Final Plans?

**Plan 1 Selection (Context-Driven Modal UI System)**:
This approach addresses the fundamental question: "Do different gameplay moments need different interfaces?" Tactical roguelikes have distinct contexts (exploration, combat, squad management) that benefit from dedicated UI modes. Modal system provides clean separation between contexts, reducing cognitive load (players see only relevant information). Inspired by Fire Emblem's context-sensitive UI and XCOM's mode-based interfaces. Represents the "tactical gameplay-first" perspective.

**Plan 2 Selection (ECS-Integrated Component System)**:
This approach directly addresses architectural consistency. The squad system (2358 LOC, perfect ECS compliance) demonstrates the power of ECS architecture. Extending this pattern to UI creates unified data model (squads, units, UI panels all ECS entities), consistent query patterns, and reusable component design. Represents the "architectural harmony" perspective. Recommended as primary choice due to proven success of squad system ECS patterns.

**Plan 3 Selection (Functional Renderer Pattern)**:
This approach challenges the assumption that UI needs complex abstraction. Pure functional rendering (game state → render functions → screen) is simplest possible architecture. Zero hidden state, zero synchronization bugs, maximum debuggability. Represents the "Go minimalist" perspective - "clear is better than clever." Fastest implementation time (18h) and lowest complexity. Best choice for teams prioritizing simplicity over architectural sophistication.

### Elements Combined

**From Tactical Gameplay Perspective**:
- **Information density**: All plans address showing relevant tactical data (threat ranges, squad formations, unit stats)
- **Context awareness**: Plan 1 explicitly modal, Plan 2 uses visibility flags, Plan 3 conditionally renders based on game state
- **Squad management**: All plans include squad panel designs (3x3 grid visualization, unit HP, formations)
- **Overlay systems**: Plans 1, 2, 3 all include threat range, movement range, and cover overlays

**From Go Standards Perspective**:
- **Composition patterns**: Plan 1 uses interface-based modes, Plan 2 uses ECS components, Plan 3 uses function composition
- **Performance optimization**: All plans use immediate-mode rendering (no retained state), widget caching, and minimal allocations
- **ECS integration**: Plan 2 native ECS, Plans 1 and 3 query ECS as data source
- **Testing strategies**: All plans designed for testability (mocked contexts, isolated components, pure functions)

**Synthesis Insights**:
The three plans represent three valid architectural paradigms:
1. **State machine** (Plan 1): Object-oriented, mode-based, context-driven
2. **Data-driven** (Plan 2): ECS-native, component-based, query-driven
3. **Functional** (Plan 3): Pure functions, data transformation, render-only

All three can coexist in hybrid approaches (e.g., ECS components within modal states, or functional overlays with ECS panels).

### Elements Rejected

**From Initial Analysis - NOT included in final approaches:**

1. **Declarative UI DSL** (rejected):
   - Reason: Go lacks good DSL support without reflection/code generation
   - Would require significant tooling investment
   - Ebiten UI already provides option-based "declarative" pattern

2. **Virtual DOM / Reactive UI** (rejected):
   - Reason: Ebiten is immediate-mode, not retained-mode - architectural mismatch
   - Would require rewriting Ebiten wrapper entirely
   - No benefit over immediate-mode for game UI

3. **Event Bus System** (rejected):
   - Reason: Overkill for game UI (10-30 elements, simple interactions)
   - Adds indirection without clear benefit
   - Direct method calls or UpdateContext pattern sufficient

4. **Automated Layout Solver** (rejected):
   - Reason: Game UI has fixed layouts (not responsive like web)
   - Constraint-based layout complexity far exceeds benefit
   - Manual positioning (Plan 1, 3) or declarative anchors (Plan 2) sufficient

5. **Theme/Style System** (considered but deferred):
   - Included in Plan 2 (Themed interface) but not core
   - Reason: 1365 LOC codebase likely doesn't need dynamic theming yet
   - Can add later via Themed interface if needed

6. **UI State Machine for Dialogs** (partially included):
   - Plan 1 includes modal state machine
   - Plans 2 and 3 require explicit dialog state in game state
   - Reason: Modal dialogs don't fit pure functional approach perfectly

### Key Insights from Multi-Agent Analysis

**Tactical Insights (trpg-creator perspective)**:
- Squad management UI requires 5+ panels with complex interactions (formation editor, unit detail, ability bar)
- Information density critical: players need to process threat ranges, squad formations, unit stats simultaneously
- Context matters: exploration vs combat vs squad management have different information priorities
- TRPG conventions: 3x3 squad grid, threat range overlays, cover indicators, HP bars standard across genre

**Technical Insights (go-standards-reviewer perspective)**:
- ECS architecture proven successful in squad system (2358 LOC, perfect patterns) - should extend to UI
- Pure functional rendering simplest approach but lacks code reuse for complex UIs
- Ebiten immediate-mode rendering efficient (not performance concern)
- Go idioms favor composition over inheritance: interface-based modes (Plan 1), ECS components (Plan 2), function composition (Plan 3)

**Synthesis Insights (implementation-critic evaluation)**:
- **Plan 1**: Highest cognitive load reduction (context-appropriate UI) but highest migration risk (atomic mode system)
- **Plan 2**: Best architectural consistency (ECS alignment) and lowest migration risk (incremental), recommended for this codebase
- **Plan 3**: Fastest implementation (18h) and simplest code but lacks reusability for squad UI complexity

### Implementation-Critic Key Insights

**Code Quality Assessment**:
- **Plan 1**: Modal state machine well-designed (clean transitions, explicit state) but potential mode proliferation risk
- **Plan 2**: Perfect ECS compliance mirrors squad system, highest code consistency, proper separation of concerns
- **Plan 3**: Extreme simplicity is virtue but potential code duplication in render functions (no shared panel logic)

**Architectural Soundness**:
- **Plan 1**: Sound state machine architecture, proven pattern, but adds abstraction layer over Ebiten
- **Plan 2**: Soundest architecture for this codebase (matches existing squad system), proper component model
- **Plan 3**: Minimal architecture (just functions and data), lowest abstraction overhead

**Over-Engineering Warnings**:
- **Plan 1**: Risk of mode explosion (too many modes = complexity), mitigate by limiting to 4-5 core modes
- **Plan 2**: Risk of over-querying ECS (Update() every frame), mitigate with dirty tracking and caching
- **Plan 3**: Risk of data preparation overhead (build display data every frame), mitigate with pooling

**Practical Value Ranking**:
1. **Plan 2 (ECS-Integrated)**: Highest practical value for this codebase (architectural consistency, proven patterns, scalable)
2. **Plan 3 (Functional)**: High value for simple UI (fast implementation, easy to understand) but limited scalability
3. **Plan 1 (Modal)**: Medium value (good for distinct contexts) but highest migration risk and complexity

---

## PRINCIPLES APPLIED

### TRPG Design Principles
- **Tactical Depth**: All plans include threat overlays, squad formations, and tactical information density
- **Genre Conventions**: 3x3 squad grids, HP bars, threat ranges, cover indicators match Fire Emblem/XCOM/Jagged Alliance
- **Balance**: Information transparency enables tactical decision-making (no hidden mechanics)
- **Player Agency**: UI shows options clearly (available actions, movement ranges, threat zones)

### Go Programming Principles
- **Idiomatic Go**: Composition over inheritance (interfaces in Plan 1, components in Plan 2, function composition in Plan 3)
- **Performance**: Immediate-mode rendering (no retained state), minimal allocations, hot path optimized
- **Simplicity**: KISS - Plan 3 exemplifies "clear is better than clever"
- **Maintainability**: Clear separation of concerns (modes, components, or functions)

### ECS Architecture Principles (Plan 2)
- **Pure data components**: UIComponentData, BoundsComponentData, PanelComponentData have zero logic
- **System-based logic**: GUIRenderSystem contains all rendering/update logic
- **Query-based relationships**: queryUIEntities(), queryUIEntitiesSorted() discover entities dynamically
- **Native EntityID usage**: All entity references use ecs.EntityID (not pointers)

### Integration Principles
- **Existing Architecture**: Plan 2 aligns with squad system (ECS), Plans 1/3 integrate via queries
- **Position System**: All plans use GlobalPositionSystem for O(1) spatial lookups
- **Graphics System**: All plans use BaseShape system for consistent rendering
- **Input System**: All plans route through InputCoordinator for unified input handling

---

## BLOCKERS & DEPENDENCIES

### Prerequisites
- **Squad system 85% complete**: Combat system, query system, visualization complete. Abilities and formations in progress (12-16h remaining).
- **Position system complete**: O(1) spatial grid operational. No blockers for overlay rendering.
- **Entity template system complete**: Generic CreateEntityFromTemplate() factory available. No blockers for UI entity creation.

### Architectural Blockers
**None** - All three plans can be implemented immediately with existing infrastructure.

### Recommended Order
1. **Complete squad system abilities + formations** (12-16h): Finish squad system before building squad UI
2. **Implement GUI architecture** (18-32h depending on plan): Choose Plan 2 (ECS-Integrated) recommended
3. **Integrate squad UI** (8-12h): Add squad panels, formation editor, tactical overlays
4. **Polish and balance** (4-8h): Refine UI layout, visual effects, usability

### Deferral Options
If squad system not complete, can implement partial GUI architecture:
- **Plan 2**: Implement component system and GUIRenderSystem now, add squad panels when squad system ready
- **Plan 3**: Implement functional rendering for current panels (stats, messages), add squad rendering functions later
- **Plan 1**: Implement exploration and combat modes now, add squad management mode later

---

## TESTING STRATEGY

### Build Verification
```bash
go build -o game_main/game_main.exe game_main/*.go
go test ./...
go test ./gui/... -v
```

### Manual Testing Scenarios

**Scenario 1: Stats Panel Display**
- Setup: Launch game, enter main menu
- Expected: Stats panel visible in top-right corner, shows player HP/Attack/Defense
- Validates: Panel rendering, text display, positioning

**Scenario 2: Squad Panel Display**
- Setup: Load game with 3 squads created
- Expected: 3 squad panels visible, each showing 3x3 grid with unit positions
- Validates: Multiple panel rendering, ECS query integration, grid layout

**Scenario 3: Threat Range Overlay**
- Setup: Enter combat mode with enemy units present
- Expected: Red semi-transparent cells showing enemy attack ranges
- Validates: Overlay rendering, spatial calculations, transparency

**Scenario 4: Formation Editor Interaction**
- Setup: Open squad management mode, select squad, click formation editor
- Expected: Can drag units to new grid positions, formation updates
- Validates: Input handling, drag-and-drop, state management

**Scenario 5: Mode Transition (Plan 1 only)**
- Setup: In exploration mode, press 'C' to enter combat mode
- Expected: UI transitions to combat mode (threat overlays appear, action menu appears)
- Validates: Mode transitions, state cleanup, context switching

### Balance Testing
Not applicable to GUI architecture (no gameplay balance impact). UI layout and usability tested through manual playtesting.

### Performance Testing
```go
// Benchmark UI rendering (Plan 2 example)
func BenchmarkGUIRenderSystem_Update(b *testing.B) {
    manager := ecs.NewManager()
    guiSystem := gui.NewGUIRenderSystem(manager)

    // Create 30 UI entities (typical game UI)
    for i := 0; i < 30; i++ {
        gui.CreateStatsPanel(manager, mockPlayerData)
    }

    gameState := &common.GameState{PlayerData: mockPlayerData}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        guiSystem.Update(0.016, gameState)
    }
}

func BenchmarkGUIRenderSystem_Render(b *testing.B) {
    // ... similar setup ...
    screen := ebiten.NewImage(1920, 1080)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        guiSystem.Render(screen)
    }
}
```

**Performance Targets**:
- Update: < 500 microseconds per frame (30 UI entities)
- Render: < 1 millisecond per frame (30 UI entities)
- Total UI overhead: < 3ms per frame (18% of 16.6ms budget at 60 FPS)

---

## RISK ASSESSMENT

### Gameplay Risks
| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Information overload (too many overlays) | M | M | Limit simultaneous overlays, use transparency |
| Mode confusion (Plan 1) | M | L | Clear mode indicators, smooth transitions |
| UI staleness (data out of sync) | H | L (Plan 2/3), M (Plan 1) | Reactive updates (Plan 2), fresh rendering (Plan 3) |
| Squad UI too complex (5+ panels) | M | M | Progressive disclosure, collapsible panels |

### Technical Risks
| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| ECS query overhead (Plan 2) | M | L | Cache queries, dirty tracking |
| Mode proliferation (Plan 1) | M | M | Limit to 4-5 core modes |
| Display data allocation (Plan 3) | L | M | Pool display data structures |
| Widget lifecycle complexity (Plan 2) | M | M | Clear creation/destruction rules |
| Migration breaking existing UI | H | M (Plan 1), L (Plan 2/3) | Incremental migration, feature flags |

### Performance Risks
| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| ZIndex sorting every frame (Plan 2) | L | L | Only sort when ZIndex changes |
| Text rendering overhead | M | L | Cache rendered text, lazy updates |
| GC pressure from display data (Plan 3) | M | M | Pool display data, reduce allocations |
| Overlay rendering cost (many cells) | M | L | Spatial culling, limit overlay count |

---

## IMPLEMENTATION ROADMAP

### Recommended Approach: Plan 2 (ECS-Integrated Component System)

**Phase 1: Foundation** (Estimated: 8 hours)
1. **Component Definitions (4h)**
   - Files: `gui/components.go` (new)
   - Code: Define UIComponentData, BoundsComponentData, AnchorComponentData, PanelComponentData, OverlayComponentData, WidgetComponentData, TextComponentData, ButtonComponentData
   - Validates: Component structs compile, zero logic methods

2. **GUIRenderSystem Skeleton (4h)**
   - Files: `gui/gui_render_system.go` (new)
   - Code: GUIRenderSystem struct, Initialize(), empty Update/Render/HandleInput
   - Validates: System initializes, components register successfully

**Phase 2: Core Rendering** (Estimated: 8 hours)
1. **Query Functions (2h)**
   - Files: `gui/gui_queries.go` (new)
   - Code: queryUIEntities(), queryUIEntitiesSorted(), queryInteractiveEntities()
   - Validates: Queries return correct entities, sorting works

2. **Update Logic (3h)**
   - Files: `gui/gui_render_system.go`
   - Code: Implement Update() - anchored positions, panel content, overlay data
   - Validates: Panels update when data changes, overlays recalculate

3. **Render Logic (3h)**
   - Files: `gui/gui_render_system.go`
   - Code: Implement Render() - overlays, widgets, text rendering
   - Validates: UI renders correctly, ZIndex respected

**Phase 3: Panel Migration** (Estimated: 8 hours)
1. **Stats Panel Entity (2h)**
   - Files: `gui/ui_factories.go` (new)
   - Code: CreateStatsPanel() factory function
   - Validates: Stats panel renders, displays player stats correctly

2. **Messages Panel Entity (2h)**
   - Files: `gui/ui_factories.go`
   - Code: CreateMessagePanel() factory function
   - Validates: Messages panel renders, displays message log

3. **Inventory Button Entity (2h)**
   - Files: `gui/ui_factories.go`
   - Code: CreateInventoryButton() factory function
   - Validates: Button clickable, opens throwables window

4. **Remove Old PlayerUI (2h)**
   - Files: `game_main/main.go`, `game_main/game.go`, `gui/playerUI.go`
   - Code: Replace PlayerUI with GUIRenderSystem
   - Validates: Game runs with new UI system, no regressions

### Rollback Plan
1. **Feature flag**: `const useNewGUISystem = true` in `gui/config.go`
2. **Legacy code**: Keep old PlayerUI in `gui/playerUI_legacy.go`
3. **Rollback trigger**: If critical bugs found (UI not rendering, input broken, crashes)
4. **Rollback process**: Set flag to false, recompile, legacy UI takes over
5. **Timeline**: Remove legacy code after 1 week of stable operation

### Success Metrics
- [ ] Build compiles successfully with new GUI system
- [ ] All tests pass (go test ./gui/...)
- [ ] Stats panel renders correctly in top-right corner
- [ ] Messages panel renders correctly below stats panel
- [ ] Inventory button clickable and opens window
- [ ] Performance: UI update < 500 microseconds, render < 1ms
- [ ] No regressions in existing features (game playable)

---

## NEXT STEPS

### Immediate Actions
1. **Review Plans**: Choose which final plan to implement (Plan 2 recommended)
2. **Check Blockers**: Verify squad system status (abilities + formations remaining)
3. **Prepare Environment**: Ensure development setup ready (Go 1.19+, Ebiten v2)

### Implementation Decision

**After reviewing this document, you have 3 options:**

**Option A: Implement Yourself**
- Use Plan 2 (ECS-Integrated Component System) as implementation guide
- Reference code examples and step-by-step instructions
- Ask questions if any section needs clarification
- Follow incremental migration strategy for safety

**Option B: Have Agent Implement**
- Specify which plan to implement (Plan 2 recommended)
- Agent will execute step-by-step following chosen plan
- Agent will report results and deviations
- Estimated time: 24 hours (3 workdays) for Plan 2

**Option C: Modify Plan First**
- Request changes to any of the 3 plans
- Combine elements from multiple plans (e.g., Plan 2 core + Plan 3 overlays)
- Adjust scope or approach before implementation
- Re-evaluate blockers or dependencies

### Questions to Consider
- Which plan best fits current project priorities? (Architecture consistency → Plan 2, Simplicity → Plan 3, Context-driven → Plan 1)
- Are there any blockers that need addressing first? (Squad system abilities + formations recommended but not required)
- Should squad UI be included in initial implementation or added later? (Recommended: later, after GUI architecture stable)
- Is the scope appropriate for current timeline? (24h for Plan 2, 18h for Plan 3, 32h for Plan 1)

---

## ADDITIONAL RESOURCES

### TRPG Design Resources
- Fire Emblem UI analysis: Context-sensitive panels, threat range display, unit management
- XCOM UI analysis: Tactical overlay system, cover indicators, action menu design
- Jagged Alliance UI analysis: Squad management, formation editor, complex information density
- Tactical RPG design principles: Turn-based workflow, deliberate decision-making, spatial awareness

### Go Programming Resources
- [Effective Go](https://go.dev/doc/effective_go): Interfaces, composition, error handling
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments): Idiomatic Go patterns
- [Go by Example](https://gobyexample.com/): Pure functions, composition, simplicity

### Ebiten Resources
- [Ebiten Documentation](https://ebiten.org/): Immediate-mode rendering, performance tips
- [Ebiten UI Documentation](https://github.com/ebitenui/ebitenui): Widget library, layout system
- [Dear ImGui Concepts](https://github.com/ocornut/imgui): Immediate-mode GUI principles (applies to Ebiten)

### ECS Resources
- [ECS Architecture Patterns](https://github.com/SanderMertens/ecs-faq): Query-based relationships, component design
- **Internal codebase reference**: `squads/` package (2358 LOC, perfect ECS compliance) - use as template
- [Entity-Component-System on Wikipedia](https://en.wikipedia.org/wiki/Entity_component_system): Theory and benefits

### Refactoring Resources
- Refactoring: Improving the Design of Existing Code (Fowler)
  - Chapter: Replace Conditional with Polymorphism (applies to Plan 1 mode system)
  - Chapter: Extract Class (applies to component extraction)
  - Chapter: Introduce Parameter Object (applies to UpdateContext, ModeContext)

### Internal Codebase Patterns to Reference
- `squads/` package: Perfect ECS patterns (pure data components, query-based relationships, system coordination)
- `entitytemplates/creators.go`: Configuration-based factory pattern (EntityConfig → CreateEntityFromTemplate)
- `graphics/drawableshapes.go`: BaseShape system for consistent rendering primitives
- `coords/cordmanager.go`: CoordinateManager for spatial transformations (grid ↔ pixel)
- `systems/positionsystem.go`: O(1) spatial grid system (use for overlay calculations)

---

## CONCLUSION

**Summary**: GUI package (1365 LOC) requires architectural redesign to support tactical squad-based gameplay. Current monolithic PlayerUI design doesn't scale for squad management UI (5+ panels, dynamic data, complex interactions). Three comprehensive architectural approaches analyzed:

1. **Plan 1 (Context-Driven Modal UI System)**: Different UI modes for different gameplay contexts (exploration, combat, squad management). Reduces cognitive load by showing only relevant information. 32 hours implementation, medium risk. Best for games with distinct gameplay modes.

2. **Plan 2 (ECS-Integrated Component System)**: UI elements as ECS entities, perfect alignment with existing squad system (2358 LOC, proven patterns). Reactive UI updates automatically when game state changes. 24 hours implementation, low-medium risk. **RECOMMENDED** for this codebase due to architectural consistency and proven ECS success.

3. **Plan 3 (Functional Renderer Pattern)**: Pure functions generate UI from game state each frame. Minimal abstraction, maximum simplicity. 18 hours implementation, low risk. Best for teams prioritizing simplicity over architectural sophistication.

**Recommended path**: Implement Plan 2 (ECS-Integrated Component System) for perfect alignment with squad system architecture, lowest migration risk (incremental), and highest scalability for future squad UI expansion.

**Next action**: Choose implementation plan (Plan 2 recommended), verify no blockers (squad system 85% complete, can proceed), begin Phase 1 (Foundation - component definitions + system skeleton, 8 hours).

---

END OF IMPLEMENTATION PLAN
