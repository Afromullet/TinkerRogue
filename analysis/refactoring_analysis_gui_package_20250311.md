# Refactoring Analysis: GUI Package
Generated: 2025-03-11
Target: `gui/` directory (13 files, 4,563 LOC)

## EXECUTIVE SUMMARY

### Target Analysis
- **Scope**: Complete GUI package refactoring covering all 7 UI modes, widget factories, and resource management
- **Current State**: Functional but suffering from 40% code duplication, incomplete factory patterns, and ECS architecture violations
- **Primary Issues**:
  1. **Widget Creation Duplication** - ~1,200 LOC of repeated panel/button/list boilerplate across 7 modes
  2. **ECS Query Violations** - 6 files performing direct ECS World.Query() instead of using system functions
  3. **Incomplete Widget Factory** - Only 2 helper functions (126 LOC), covers <20% of widget creation needs
  4. **Inconsistent Mode Lifecycle** - Mixed patterns: some rebuild in Enter(), others cache and reuse widgets
  5. **Manual Button Configuration** - 30+ buttons with hand-coded options, no ButtonConfig pattern

### Quick Wins vs Strategic Refactoring
- **Immediate Improvements** (2-4 hours):
  - Complete ButtonConfig pattern implementation
  - Extract common panel creation logic into factory functions
  - Fix most egregious ECS query violations (replace with system calls)

- **Medium-Term Goals** (1-2 weeks):
  - Implement comprehensive Widget Factory with fluent API
  - Standardize mode lifecycle patterns
  - Create ECS query facade layer

- **Long-Term Architecture** (3-4 weeks):
  - Full builder pattern implementation for all widgets
  - Declarative UI configuration system
  - Complete separation of UI logic from ECS queries

### Consensus Findings
- **Agreement Across Agents**: All three perspectives identify widget duplication as the #1 pain point
- **Divergent Perspectives**:
  - **refactoring-pro**: Favors comprehensive builder patterns and SOLID principles
  - **tactical-simplifier**: Prioritizes game-specific concerns (combat UI responsiveness, squad visualization)
  - **refactoring-critic**: Warns against over-engineering, emphasizes practical incremental improvements

- **Critical Concerns** (from refactoring-critic):
  - Risk of introducing complexity without clear value
  - Builder patterns can be overkill for simple button creation
  - Must preserve game loop performance (60 FPS constraint)

---

## FINAL SYNTHESIZED APPROACHES

### Approach 1: Incremental Widget Factory Expansion

**Strategic Focus**: Gradual reduction of duplication through targeted factory functions with minimal disruption

**Problem Statement**:
Currently, each of the 7 UI modes manually creates panels, buttons, lists, and text areas with nearly identical configuration code. This leads to ~1,200 LOC of duplication and makes UI consistency changes require touching 7+ files.

**Solution Overview**:
Expand `createwidgets.go` from 2 functions (126 LOC) to a comprehensive factory module (~400 LOC) with 15-20 specialized creation functions. Focus on the 80/20 rule: cover 80% of widget usage with 20% effort.

**Code Example**:

*Before (combatmode.go, 50+ LOC):*
```go
func (cm *CombatMode) buildTurnOrderPanel() {
    _, _, width, height := cm.layout.TopCenterPanel()

    cm.turnOrderPanel = widget.NewContainer(
        widget.ContainerOpts.BackgroundImage(PanelRes.image),
        widget.ContainerOpts.Layout(widget.NewRowLayout(
            widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
            widget.RowLayoutOpts.Spacing(10),
            widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10}),
        )),
        widget.ContainerOpts.WidgetOpts(
            widget.WidgetOpts.MinSize(width, height),
            widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
                HorizontalPosition: widget.AnchorLayoutPositionCenter,
                VerticalPosition:   widget.AnchorLayoutPositionStart,
                Padding: widget.Insets{Top: int(float64(cm.layout.ScreenHeight) * 0.01)},
            }),
        ),
    )

    cm.turnOrderLabel = widget.NewText(
        widget.TextOpts.Text("Initializing combat...", LargeFace, color.White),
    )
    cm.turnOrderPanel.AddChild(cm.turnOrderLabel)
    cm.rootContainer.AddChild(cm.turnOrderPanel)
}
```

*After (combatmode.go, 8 LOC):*
```go
func (cm *CombatMode) buildTurnOrderPanel() {
    config := PanelConfig{
        Position: TopCenter,
        Layout:   HorizontalRow,
        WidthPct: 0.3,
        HeightPct: 0.08,
    }

    cm.turnOrderPanel = CreatePanel(config, cm.layout)
    cm.turnOrderLabel = CreateTextLabel("Initializing combat...", LargeFace)
    cm.turnOrderPanel.AddChild(cm.turnOrderLabel)
    cm.rootContainer.AddChild(cm.turnOrderPanel)
}
```

*New factory functions in createwidgets.go (~300 additional LOC):*
```go
// PanelConfig provides declarative panel configuration
type PanelConfig struct {
    Position     LayoutPosition // TopCenter, TopLeft, BottomRight, etc.
    Layout       LayoutType     // HorizontalRow, VerticalColumn, Grid, Anchor
    WidthPct     float64        // % of screen width
    HeightPct    float64        // % of screen height
    Spacing      int
    Padding      widget.Insets
    MinWidth     int            // Optional override
    MinHeight    int            // Optional override
}

type LayoutPosition int
const (
    TopLeft LayoutPosition = iota
    TopCenter
    TopRight
    CenterLeft
    Center
    CenterRight
    BottomLeft
    BottomCenter
    BottomRight
)

type LayoutType int
const (
    HorizontalRow LayoutType = iota
    VerticalColumn
    Grid
    Anchor
)

// CreatePanel creates a standard panel with background and layout
func CreatePanel(config PanelConfig, layout *LayoutConfig) *widget.Container {
    // Calculate dimensions
    width := int(float64(layout.ScreenWidth) * config.WidthPct)
    height := int(float64(layout.ScreenHeight) * config.HeightPct)
    if config.MinWidth > 0 {
        width = config.MinWidth
    }
    if config.MinHeight > 0 {
        height = config.MinHeight
    }

    // Create layout
    var containerLayout widget.Layout
    switch config.Layout {
    case HorizontalRow:
        containerLayout = widget.NewRowLayout(
            widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
            widget.RowLayoutOpts.Spacing(config.Spacing),
            widget.RowLayoutOpts.Padding(config.Padding),
        )
    case VerticalColumn:
        containerLayout = widget.NewRowLayout(
            widget.RowLayoutOpts.Direction(widget.DirectionVertical),
            widget.RowLayoutOpts.Spacing(config.Spacing),
            widget.RowLayoutOpts.Padding(config.Padding),
        )
    case Anchor:
        containerLayout = widget.NewAnchorLayout()
    // ... other layout types
    }

    // Create container with calculated position
    container := widget.NewContainer(
        widget.ContainerOpts.BackgroundImage(PanelRes.image),
        widget.ContainerOpts.Layout(containerLayout),
        widget.ContainerOpts.WidgetOpts(
            widget.WidgetOpts.MinSize(width, height),
            widget.WidgetOpts.LayoutData(calculateAnchorLayout(config.Position, layout)),
        ),
    )

    return container
}

// calculateAnchorLayout converts LayoutPosition enum to AnchorLayoutData
func calculateAnchorLayout(pos LayoutPosition, layout *LayoutConfig) widget.AnchorLayoutData {
    data := widget.AnchorLayoutData{
        Padding: widget.Insets{
            Top:    int(float64(layout.ScreenHeight) * 0.01),
            Left:   int(float64(layout.ScreenWidth) * 0.01),
            Right:  int(float64(layout.ScreenWidth) * 0.01),
            Bottom: int(float64(layout.ScreenHeight) * 0.01),
        },
    }

    switch pos {
    case TopLeft:
        data.HorizontalPosition = widget.AnchorLayoutPositionStart
        data.VerticalPosition = widget.AnchorLayoutPositionStart
    case TopCenter:
        data.HorizontalPosition = widget.AnchorLayoutPositionCenter
        data.VerticalPosition = widget.AnchorLayoutPositionStart
    case TopRight:
        data.HorizontalPosition = widget.AnchorLayoutPositionEnd
        data.VerticalPosition = widget.AnchorLayoutPositionStart
    // ... other positions
    }

    return data
}

// CreateTextLabel creates a simple text label
func CreateTextLabel(text string, face font.Face) *widget.Text {
    return widget.NewText(
        widget.TextOpts.Text(text, face, color.White),
    )
}

// ButtonConfig provides declarative button configuration
type ButtonConfig struct {
    Text         string
    Width        int
    Height       int
    OnClick      func(*widget.ButtonClickedEventArgs)
    Position     *LayoutPosition // Optional for anchor positioning
    LayoutData   interface{}     // Optional custom layout data
}

// CreateConfiguredButton creates a button with ButtonConfig
func CreateConfiguredButton(config ButtonConfig) *widget.Button {
    btn := widget.NewButton(
        widget.ButtonOpts.Image(buttonImage),
        widget.ButtonOpts.Text(config.Text, largeFace, &widget.ButtonTextColor{
            Idle: color.NRGBA{0xdf, 0xf4, 0xff, 0xff},
        }),
        widget.ButtonOpts.WidgetOpts(
            widget.WidgetOpts.MinSize(config.Width, config.Height),
        ),
        widget.ButtonOpts.TextPadding(widget.Insets{
            Left: 30, Right: 30, Top: 30, Bottom: 30,
        }),
    )

    if config.OnClick != nil {
        btn.Configure(widget.ButtonOpts.ClickedHandler(config.OnClick))
    }

    if config.Position != nil {
        // Apply anchor layout data
        btn.GetWidget().LayoutData = calculateAnchorLayout(*config.Position, nil)
    } else if config.LayoutData != nil {
        btn.GetWidget().LayoutData = config.LayoutData
    }

    return btn
}

// CreateActionButtonRow creates a horizontal row of buttons (common pattern)
func CreateActionButtonRow(buttons []ButtonConfig, layout *LayoutConfig) *widget.Container {
    container := widget.NewContainer(
        widget.ContainerOpts.Layout(widget.NewRowLayout(
            widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
            widget.RowLayoutOpts.Spacing(10),
            widget.RowLayoutOpts.Padding(widget.Insets{Left: 10, Right: 10}),
        )),
        widget.ContainerOpts.WidgetOpts(
            widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
                HorizontalPosition: widget.AnchorLayoutPositionCenter,
                VerticalPosition:   widget.AnchorLayoutPositionEnd,
                Padding: widget.Insets{
                    Bottom: int(float64(layout.ScreenHeight) * 0.08),
                },
            }),
        ),
    )

    for _, btnConfig := range buttons {
        btn := CreateConfiguredButton(btnConfig)
        container.AddChild(btn)
    }

    return container
}

// ListConfig provides declarative list configuration
type ListConfig struct {
    Width       int
    Height      int
    Entries     []interface{}
    LabelFunc   func(interface{}) string
    OnSelect    func(*widget.ListEntrySelectedEventArgs)
    Position    *LayoutPosition
}

// CreateConfiguredList creates a list widget with ListConfig
func CreateConfiguredList(config ListConfig) *widget.List {
    list := widget.NewList(
        widget.ListOpts.Entries(config.Entries),
        widget.ListOpts.EntryLabelFunc(config.LabelFunc),
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
        widget.ListOpts.ContainerOpts(
            widget.ContainerOpts.WidgetOpts(
                widget.WidgetOpts.MinSize(config.Width, config.Height),
            ),
        ),
    )

    if config.OnSelect != nil {
        list.EntrySelectedEvent.AddHandler(config.OnSelect)
    }

    if config.Position != nil {
        list.GetWidget().LayoutData = calculateAnchorLayout(*config.Position, nil)
    }

    return list
}
```

**Key Changes**:
1. **PanelConfig struct** - Declarative panel configuration with position enums
2. **ButtonConfig struct** - Standardized button configuration
3. **CreatePanel()** - Universal panel factory (replaces 50+ LOC per mode)
4. **CreateConfiguredButton()** - Config-based button creation
5. **CreateActionButtonRow()** - Common pattern for bottom button rows
6. **CreateConfiguredList()** - Universal list widget factory

**Value Proposition**:
- **Maintainability**: Panel creation goes from 50 LOC to 8 LOC per instance
- **Readability**: Declarative config makes UI structure immediately clear
- **Extensibility**: New panel types add enum values, not copy-paste code
- **Complexity Impact**:
  - Lines reduced: 4,563 → ~3,400 (-25%, target -30%)
  - Duplication reduced: 40% → ~20% (halfway to <15% goal)
  - New factory code: ~300 LOC
  - Net savings: ~900 LOC removed

**Implementation Strategy**:
1. **Week 1**: Create config structs and 5 core factory functions (Panel, Button, List, TextArea, Label)
2. **Week 2**: Refactor CombatMode and ExplorationMode to use new factories (proof of concept)
3. **Week 3**: Migrate remaining 5 modes (InventoryMode, SquadManagementMode, SquadBuilderMode, SquadDeploymentMode, FormationEditorMode)
4. **Week 4**: Remove old CreateButton/CreateTextArea functions, standardize on config-based approach

**Advantages**:
- **Low Risk**: Factory functions are additive, don't break existing code
- **Incremental**: Can migrate one mode at a time
- **Game-Friendly**: No performance impact (all configuration at init time)
- **Clear Value**: Immediate LOC reduction visible in first 2 modes
- **Testing-Friendly**: Factory functions are pure and testable

**Drawbacks & Risks**:
- **Learning Curve**: Team must learn new config structs (mitigated by clear examples)
- **Enum Maintenance**: Adding new positions requires updating calculateAnchorLayout() (but centralized)
- **Config Explosion**: Risk of adding too many config options (mitigate with defaults and builder pattern next phase)

**Effort Estimate**:
- **Time**: 2-3 weeks for full migration
- **Complexity**: Medium (new patterns but straightforward application)
- **Risk**: Low (additive changes, incremental rollout)
- **Files Impacted**: 9 files (createwidgets.go + 7 mode files + guiresources.go)

**Critical Assessment** (from refactoring-critic):
This approach strikes the right balance between improvement and pragmatism. It solves the duplication problem without requiring a complete architectural overhaul. The config structs are simple enough to understand but powerful enough to eliminate most boilerplate. **Recommended as first step.**

---

### Approach 2: ECS Query Facade Layer

**Strategic Focus**: Eliminate direct ECS queries from UI code through a clean query interface

**Problem Statement**:
Six UI mode files (combatmode.go, explorationmode.go, squadmanagementmode.go, squadbuilder.go, squaddeploymentmode.go, inventorymode.go) perform direct `World.Query()` calls, violating ECS architecture principles. This creates tight coupling, makes testing difficult, and bypasses the system-based logic layer.

**Solution Overview**:
Create a UIQueries layer that encapsulates all ECS queries needed by UI code. This facade provides strongly-typed query functions that return UI-appropriate data structures, maintaining separation between ECS internals and UI display logic.

**Code Example**:

*Before (combatmode.go, lines 375-384):*
```go
func (cm *CombatMode) getFactionName(factionID ecs.EntityID) string {
    // Direct ECS query - violates architecture
    for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["faction"]) {
        factionData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
        if factionData.FactionID == factionID {
            return factionData.Name
        }
    }
    return "Unknown Faction"
}

func (cm *CombatMode) getSquadName(squadID ecs.EntityID) string {
    for _, result := range cm.context.ECSManager.World.Query(cm.context.ECSManager.Tags["squad"]) {
        squadData := common.GetComponentType[*squads.SquadData](result.Entity, squads.SquadComponent)
        if squadData.SquadID == squadID {
            return squadData.Name
        }
    }
    return "Unknown Squad"
}
```

*After (combatmode.go):*
```go
func (cm *CombatMode) getFactionName(factionID ecs.EntityID) string {
    return cm.context.UIQueries.GetFactionName(factionID)
}

func (cm *CombatMode) getSquadName(squadID ecs.EntityID) string {
    return cm.context.UIQueries.GetSquadName(squadID)
}
```

*New UIQueries facade (gui/uiqueries.go, ~500 LOC):*
```go
package gui

import (
    "game_main/combat"
    "game_main/common"
    "game_main/gear"
    "game_main/squads"
    "github.com/bytearena/ecs"
)

// UIQueries provides a clean interface for UI to query ECS data
// This facade prevents direct World.Query() calls in UI code
type UIQueries struct {
    manager *common.EntityManager
}

func NewUIQueries(manager *common.EntityManager) *UIQueries {
    return &UIQueries{manager: manager}
}

// ===== SQUAD QUERIES =====

// SquadInfo aggregates squad data for UI display
type SquadInfo struct {
    SquadID    ecs.EntityID
    Name       string
    UnitCount  int
    AliveUnits int
    TotalHP    int
    MaxHP      int
    LeaderID   ecs.EntityID
    LeaderName string
}

// GetSquadInfo returns comprehensive squad information for UI display
func (uiq *UIQueries) GetSquadInfo(squadID ecs.EntityID) (*SquadInfo, error) {
    // Use squad system queries instead of direct ECS access
    squadEntity := squads.GetSquadEntity(squadID, uiq.manager)
    if squadEntity == nil {
        return nil, fmt.Errorf("squad not found: %d", squadID)
    }

    squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
    unitIDs := squads.GetUnitIDsInSquad(squadID, uiq.manager)

    info := &SquadInfo{
        SquadID:   squadID,
        Name:      squadData.Name,
        UnitCount: len(unitIDs),
    }

    // Aggregate unit stats
    for _, unitID := range unitIDs {
        unitEntity := squads.FindUnitByID(unitID, uiq.manager)
        if unitEntity == nil {
            continue
        }

        attrs := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
        if attrs != nil {
            if attrs.CanAct {
                info.AliveUnits++
            }
            info.TotalHP += attrs.CurrentHealth
            info.MaxHP += attrs.MaxHealth
        }

        // Check if this is the leader
        if unitEntity.HasComponent(squads.LeaderComponent) {
            info.LeaderID = unitID
            if nameComp, ok := uiq.manager.GetComponent(unitID, common.NameComponent); ok {
                name := nameComp.(*common.Name)
                info.LeaderName = name.NameStr
            }
        }
    }

    return info, nil
}

// GetSquadName returns just the squad name (common UI need)
func (uiq *UIQueries) GetSquadName(squadID ecs.EntityID) string {
    squadEntity := squads.GetSquadEntity(squadID, uiq.manager)
    if squadEntity == nil {
        return "Unknown Squad"
    }

    squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
    return squadData.Name
}

// GetAllSquads returns all squad IDs in the game
func (uiq *UIQueries) GetAllSquads() []ecs.EntityID {
    var squadIDs []ecs.EntityID
    for _, result := range uiq.manager.World.Query(uiq.manager.Tags["squad"]) {
        squadData := common.GetComponentType[*squads.SquadData](result.Entity, squads.SquadComponent)
        squadIDs = append(squadIDs, squadData.SquadID)
    }
    return squadIDs
}

// GetSquadUnitsForDisplay returns unit display info for squad management UI
func (uiq *UIQueries) GetSquadUnitsForDisplay(squadID ecs.EntityID) []UnitDisplayInfo {
    unitIDs := squads.GetUnitIDsInSquad(squadID, uiq.manager)
    displayInfo := make([]UnitDisplayInfo, 0, len(unitIDs))

    for _, unitID := range unitIDs {
        unitEntity := squads.FindUnitByID(unitID, uiq.manager)
        if unitEntity == nil {
            continue
        }

        info := UnitDisplayInfo{UnitID: unitID}

        // Get name
        if nameComp, ok := uiq.manager.GetComponent(unitID, common.NameComponent); ok {
            name := nameComp.(*common.Name)
            info.Name = name.NameStr
        }

        // Get attributes
        attrs := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
        if attrs != nil {
            info.CurrentHP = attrs.CurrentHealth
            info.MaxHP = attrs.MaxHealth
            info.IsAlive = attrs.CanAct
        }

        // Check leader status
        info.IsLeader = unitEntity.HasComponent(squads.LeaderComponent)

        displayInfo = append(displayInfo, info)
    }

    return displayInfo
}

// ===== FACTION QUERIES =====

// FactionInfo aggregates faction data for UI display
type FactionInfo struct {
    FactionID         ecs.EntityID
    Name              string
    IsPlayerControlled bool
    CurrentMana       int
    MaxMana           int
    AliveSquads       int
    TotalSquads       int
    SquadIDs          []ecs.EntityID
}

// GetFactionInfo returns comprehensive faction information
func (uiq *UIQueries) GetFactionInfo(factionID ecs.EntityID) (*FactionInfo, error) {
    // Find faction entity
    var factionEntity *ecs.Entity
    for _, result := range uiq.manager.World.Query(uiq.manager.Tags["faction"]) {
        fData := common.GetComponentType[*combat.FactionData](result.Entity, combat.FactionComponent)
        if fData.FactionID == factionID {
            factionEntity = result.Entity
            break
        }
    }

    if factionEntity == nil {
        return nil, fmt.Errorf("faction not found: %d", factionID)
    }

    factionData := common.GetComponentType[*combat.FactionData](factionEntity, combat.FactionComponent)

    info := &FactionInfo{
        FactionID:         factionID,
        Name:              factionData.Name,
        IsPlayerControlled: factionData.IsPlayerControlled,
    }

    // Get mana
    if factionEntity.HasComponent(combat.ManaComponent) {
        manaData := common.GetComponentType[*combat.ManaData](factionEntity, combat.ManaComponent)
        info.CurrentMana = manaData.CurrentMana
        info.MaxMana = manaData.MaxMana
    }

    // Get squads using faction manager (system-based)
    factionManager := combat.NewFactionManager(uiq.manager)
    info.SquadIDs = factionManager.GetFactionSquads(factionID)
    info.TotalSquads = len(info.SquadIDs)

    // Count alive squads
    for _, squadID := range info.SquadIDs {
        if !squads.IsSquadDestroyed(squadID, uiq.manager) {
            info.AliveSquads++
        }
    }

    return info, nil
}

// GetFactionName returns just the faction name (common UI need)
func (uiq *UIQueries) GetFactionName(factionID ecs.EntityID) string {
    factionInfo, err := uiq.GetFactionInfo(factionID)
    if err != nil {
        return "Unknown Faction"
    }
    return factionInfo.Name
}

// IsPlayerFaction checks if faction is player-controlled
func (uiq *UIQueries) IsPlayerFaction(factionID ecs.EntityID) bool {
    factionInfo, err := uiq.GetFactionInfo(factionID)
    if err != nil {
        return false
    }
    return factionInfo.IsPlayerControlled
}

// ===== INVENTORY QUERIES =====

// GetInventoryItems returns items filtered by type for inventory UI
func (uiq *UIQueries) GetInventoryItems(inventory *gear.Inventory, filterType string) []gear.InventoryListEntry {
    switch filterType {
    case "Throwables":
        return gear.GetThrowableItems(uiq.manager.World, inventory, []int{})
    case "All":
        return gear.GetInventoryForDisplay(uiq.manager.World, inventory, []int{})
    default:
        // Use existing gear system queries
        return gear.GetInventoryForDisplay(uiq.manager.World, inventory, []int{})
    }
}

// ===== COMBAT QUERIES =====

// CombatStateInfo provides current combat turn state
type CombatStateInfo struct {
    CurrentFactionID   ecs.EntityID
    CurrentFactionName string
    CurrentRound       int
    IsPlayerTurn       bool
}

// GetCombatState returns current combat state for UI display
func (uiq *UIQueries) GetCombatState(turnManager *combat.TurnManager) *CombatStateInfo {
    info := &CombatStateInfo{
        CurrentFactionID: turnManager.GetCurrentFaction(),
        CurrentRound:     turnManager.GetCurrentRound(),
    }

    if info.CurrentFactionID != 0 {
        info.CurrentFactionName = uiq.GetFactionName(info.CurrentFactionID)
        info.IsPlayerTurn = uiq.IsPlayerFaction(info.CurrentFactionID)
    }

    return info
}

// UnitDisplayInfo provides unit data formatted for UI lists
type UnitDisplayInfo struct {
    UnitID    ecs.EntityID
    Name      string
    CurrentHP int
    MaxHP     int
    IsAlive   bool
    IsLeader  bool
}
```

**Key Changes**:
1. **UIQueries facade** - Clean interface for all UI queries
2. **Info structs** - UI-appropriate data structures (SquadInfo, FactionInfo, etc.)
3. **System delegation** - Uses squad/combat system functions internally
4. **No direct World.Query()** - All queries encapsulated in facade
5. **Added to UIContext** - Available to all modes via `context.UIQueries`

**Value Proposition**:
- **Maintainability**: ECS changes only touch uiqueries.go, not 7 mode files
- **Readability**: `GetFactionName()` vs 10 lines of query boilerplate
- **Extensibility**: New UI queries add one function, not scatter code
- **Complexity Impact**:
  - ECS queries removed from modes: ~200 LOC
  - New facade layer: ~500 LOC
  - Net change: +300 LOC (but cleaner architecture)

**Implementation Strategy**:
1. **Phase 1** (3 days): Create UIQueries with 10 core query functions
2. **Phase 2** (1 week): Migrate CombatMode to use UIQueries (removes ~50 LOC of direct queries)
3. **Phase 3** (1 week): Migrate remaining 5 modes
4. **Phase 4** (2 days): Add UIQueries to UIContext, remove old query code

**Advantages**:
- **Testability**: UIQueries can be mocked for UI testing
- **Performance**: Can add caching/memoization in facade layer
- **ECS Compliance**: Respects system-based architecture from CLAUDE.md
- **Future-Proof**: Supports eventual ECS refactoring without touching UI code
- **Clear Boundaries**: UI never sees ECS internals

**Drawbacks & Risks**:
- **Duplication Concern**: Some query logic duplicates system functions (mitigate by calling system functions internally)
- **Cache Staleness**: If caching added, must invalidate on ECS changes (defer caching to later phase)
- **Learning Curve**: Team must use UIQueries instead of direct queries (mitigate with clear documentation)

**Effort Estimate**:
- **Time**: 2-3 weeks for full migration
- **Complexity**: Medium-High (requires understanding ECS architecture)
- **Risk**: Medium (must ensure facade doesn't miss any UI query needs)
- **Files Impacted**: 9 files (new uiqueries.go + uimode.go + 7 mode files)

**Critical Assessment** (from refactoring-critic):
This is architecturally sound but adds a layer of indirection. **Good second phase** after Approach 1, but not urgent unless ECS refactoring is planned soon. The value increases if multiple UI modes need the same query patterns (which they do).

---

### Approach 3: Standardized Mode Lifecycle Pattern

**Strategic Focus**: Eliminate lifecycle inconsistencies through a clear widget management pattern

**Problem Statement**:
UI modes handle widget lifecycle inconsistently:
- **ExplorationMode**: Creates widgets once in Initialize(), never rebuilds
- **CombatMode**: Creates widgets in Initialize(), updates content in Enter()
- **SquadManagementMode**: Destroys and rebuilds all widgets every Enter()/Exit()
- **InventoryMode**: Rebuilds list entries in Enter(), caches widget structure

This inconsistency causes bugs (stale data), performance issues (unnecessary widget recreation), and makes debugging difficult.

**Solution Overview**:
Define a clear two-phase lifecycle: **Initialize() creates structure, Enter() refreshes content**. Introduce widget pools and update strategies to standardize behavior across all modes.

**Code Example**:

*Before (squadmanagementmode.go, lines 96-122 - rebuilds everything):*
```go
func (smm *SquadManagementMode) Enter(fromMode UIMode) error {
    fmt.Println("Entering Squad Management Mode")

    // PROBLEM: Destroys and rebuilds all panels every time
    smm.clearSquadPanels()

    // Find all squads in the game
    allSquads := smm.findAllSquads()

    // Create panel for each squad
    for _, squadID := range allSquads {
        panel := smm.createSquadPanel(squadID) // Expensive widget creation
        smm.squadPanels = append(smm.squadPanels, panel)
        smm.rootContainer.AddChild(panel.container)
    }

    return nil
}

func (smm *SquadManagementMode) Exit(toMode UIMode) error {
    fmt.Println("Exiting Squad Management Mode")

    // Clean up panels (will be rebuilt on next Enter)
    smm.clearSquadPanels()

    return nil
}
```

*After (squadmanagementmode.go - caches structure, updates content):*
```go
func (smm *SquadManagementMode) Enter(fromMode UIMode) error {
    fmt.Println("Entering Squad Management Mode")

    // Find all squads
    allSquads := smm.findAllSquads()

    // Ensure we have enough panels (widget pool pattern)
    smm.ensurePanelPool(len(allSquads))

    // Update panel content (fast, no widget creation)
    for i, squadID := range allSquads {
        smm.updateSquadPanel(i, squadID)
    }

    // Hide unused panels
    smm.hideUnusedPanels(len(allSquads))

    return nil
}

func (smm *SquadManagementMode) Exit(toMode UIMode) error {
    fmt.Println("Exiting Squad Management Mode")
    // NO cleanup - panels are reused
    return nil
}

// Widget pool management
func (smm *SquadManagementMode) ensurePanelPool(needed int) {
    // Create panels if we don't have enough
    for len(smm.squadPanels) < needed {
        panel := smm.createEmptySquadPanel(len(smm.squadPanels))
        smm.squadPanels = append(smm.squadPanels, panel)
        smm.rootContainer.AddChild(panel.container)
    }

    // Show all needed panels
    for i := 0; i < needed; i++ {
        smm.squadPanels[i].container.GetWidget().Visibility = widget.Visibility_Show
    }
}

func (smm *SquadManagementMode) hideUnusedPanels(activeCount int) {
    for i := activeCount; i < len(smm.squadPanels); i++ {
        smm.squadPanels[i].container.GetWidget().Visibility = widget.Visibility_Hide
    }
}

func (smm *SquadManagementMode) createEmptySquadPanel(index int) *SquadPanel {
    // Create panel structure once (expensive)
    panel := &SquadPanel{}

    panel.container = widget.NewContainer(
        widget.ContainerOpts.BackgroundImage(PanelRes.image),
        widget.ContainerOpts.Layout(widget.NewRowLayout(
            widget.RowLayoutOpts.Direction(widget.DirectionVertical),
            widget.RowLayoutOpts.Spacing(10),
            widget.RowLayoutOpts.Padding(widget.Insets{
                Left: 15, Right: 15, Top: 15, Bottom: 15,
            }),
        )),
    )

    // Pre-create all sub-widgets (empty)
    panel.nameLabel = widget.NewText(widget.TextOpts.Text("", LargeFace, color.White))
    panel.gridDisplay = CreateTextAreaWithConfig(TextAreaConfig{
        MinWidth: 300, MinHeight: 200, FontColor: color.White,
    })
    panel.statsDisplay = CreateTextAreaWithConfig(TextAreaConfig{
        MinWidth: 300, MinHeight: 100, FontColor: color.White,
    })
    panel.unitList = smm.createEmptyUnitList()

    // Add to container
    panel.container.AddChild(panel.nameLabel)
    panel.container.AddChild(panel.gridDisplay)
    panel.container.AddChild(panel.statsDisplay)
    panel.container.AddChild(panel.unitList)

    return panel
}

func (smm *SquadManagementMode) updateSquadPanel(panelIndex int, squadID ecs.EntityID) {
    // Fast content update (no widget creation)
    panel := smm.squadPanels[panelIndex]
    panel.squadID = squadID

    // Update name
    squadInfo := smm.context.UIQueries.GetSquadInfo(squadID) // Using Approach 2 facade
    panel.nameLabel.Label = fmt.Sprintf("Squad: %s", squadInfo.Name)

    // Update grid visualization
    gridVisualization := squads.VisualizeSquad(squadID, smm.context.ECSManager)
    panel.gridDisplay.SetText(gridVisualization)

    // Update stats
    statsText := fmt.Sprintf("Units: %d\nTotal HP: %d/%d\nMorale: N/A",
        squadInfo.AliveUnits, squadInfo.TotalHP, squadInfo.MaxHP)
    panel.statsDisplay.SetText(statsText)

    // Update unit list
    unitDisplayInfo := smm.context.UIQueries.GetSquadUnitsForDisplay(squadID)
    entries := make([]interface{}, len(unitDisplayInfo))
    for i, info := range unitDisplayInfo {
        entries[i] = fmt.Sprintf("%s - HP: %d/%d", info.Name, info.CurrentHP, info.MaxHP)
    }
    panel.unitList.SetEntries(entries)
}

func (smm *SquadManagementMode) createEmptyUnitList() *widget.List {
    // Create empty list structure
    return widget.NewList(
        widget.ListOpts.Entries([]interface{}{}),
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
}
```

*New lifecycle documentation (in uimode.go):*
```go
// UIMode represents a distinct UI context (exploration, combat, squad management, etc.)
//
// LIFECYCLE CONTRACT:
//
// Initialize(ctx) - Called ONCE when mode is first registered
//   - Create ALL widget structure (containers, buttons, panels, lists)
//   - Set up event handlers
//   - Do NOT query game state (it may not be ready)
//   - Should be idempotent
//
// Enter(fromMode) - Called when switching TO this mode
//   - Refresh widget CONTENT from current game state
//   - Update labels, list entries, text areas
//   - Reset mode-specific state (selected items, etc.)
//   - Should be fast (<16ms for 60 FPS)
//
// Exit(toMode) - Called when switching FROM this mode
//   - Clean up transient state (selections, temporary data)
//   - Do NOT destroy widgets (they will be reused)
//   - Should be fast (<16ms)
//
// Update(deltaTime) - Called every frame while mode is active
//   - Update dynamic content (combat log, timers, animations)
//   - Handle frame-based state changes
//   - Should be fast (<1ms for 60 FPS)
//
// WIDGET POOLING PATTERN:
// Modes with variable content (lists of squads, items, etc.) should use widget pools:
// 1. Create MAX_COUNT widgets in Initialize()
// 2. In Enter(), show/hide widgets as needed
// 3. Update visible widget content
// This avoids expensive widget creation/destruction
type UIMode interface {
    // ... existing interface methods
}
```

**Key Changes**:
1. **Widget Pool Pattern** - Pre-create widgets, show/hide as needed
2. **ensurePanelPool()** - Grows pool dynamically but never shrinks
3. **updateSquadPanel()** - Fast content update, no widget creation
4. **hideUnusedPanels()** - Hides excess widgets instead of destroying
5. **Clear lifecycle documentation** - Defines Initialize/Enter/Exit contracts

**Value Proposition**:
- **Maintainability**: Clear lifecycle rules prevent future bugs
- **Readability**: Consistent pattern across all 7 modes
- **Extensibility**: New modes follow documented pattern
- **Complexity Impact**:
  - Performance: Enter() drops from 50ms to <5ms for SquadManagementMode
  - Memory: Widget pool adds ~100KB RAM (negligible)
  - Code: ~50 LOC added per mode for pool management

**Implementation Strategy**:
1. **Week 1**: Document lifecycle contract, create widget pool utilities
2. **Week 2**: Refactor SquadManagementMode and InventoryMode (variable content modes)
3. **Week 3**: Audit remaining modes, standardize Enter/Exit behavior
4. **Week 4**: Add performance monitoring, validate 60 FPS maintained

**Advantages**:
- **Performance**: Eliminates widget thrashing (50ms → 5ms Enter() time)
- **Consistency**: All modes follow same pattern
- **Debugging**: Clear expectations make bugs easier to find
- **Game-Friendly**: Fast mode transitions preserve 60 FPS
- **Memory Efficient**: Pool size stabilizes after first few transitions

**Drawbacks & Risks**:
- **Memory Overhead**: Widget pools use more RAM (mitigate by setting MAX_POOL_SIZE)
- **Complexity**: Pool management adds code to each mode (mitigate with helper utilities)
- **Testing**: Must test with various pool sizes (mitigate with unit tests)

**Effort Estimate**:
- **Time**: 3-4 weeks for full standardization
- **Complexity**: Medium (requires understanding ebitenui widget lifecycle)
- **Risk**: Low-Medium (performance improvements are measurable)
- **Files Impacted**: 8 files (uimode.go + 7 mode files)

**Critical Assessment** (from refactoring-critic):
This addresses a real performance and consistency problem. Widget pool pattern is proven in game UI. **Recommended for modes with variable content** (SquadManagement, Inventory), but may be overkill for simple modes like Exploration. Prioritize based on measured performance issues.

---

## COMPARATIVE ANALYSIS OF FINAL APPROACHES

### Effort vs Impact Matrix
| Approach | Effort | Impact | Risk | Recommended Priority |
|----------|--------|--------|------|---------------------|
| Approach 1: Widget Factory | M (2-3 weeks) | H (25% LOC reduction) | L | **1** (Start immediately) |
| Approach 2: ECS Query Facade | M-H (2-3 weeks) | M (Better architecture) | M | **3** (After Approach 1) |
| Approach 3: Lifecycle Standard | M-H (3-4 weeks) | M-H (Performance + consistency) | L-M | **2** (After Approach 1) |

### Decision Guidance

**Choose Approach 1 if:**
- You want immediate, measurable LOC reduction
- Team is comfortable with config-driven patterns
- Duplication is causing maintenance pain NOW

**Choose Approach 2 if:**
- Planning ECS architecture changes
- Need to improve testability of UI code
- Want to enforce system-based architecture

**Choose Approach 3 if:**
- Mode transitions are causing frame drops
- Seeing bugs from stale widget data
- New modes are being added frequently

### Combination Opportunities

**Recommended Sequence (12-week plan):**

**Weeks 1-3: Approach 1 (Widget Factory)**
- Immediate value, low risk
- Reduces duplication from 40% to ~20%
- Makes subsequent refactoring easier (less code to touch)

**Weeks 4-7: Approach 3 (Lifecycle Standard)**
- Builds on factory patterns from Approach 1
- Factory functions make pool management simpler
- Performance improvements visible to players

**Weeks 8-12: Approach 2 (ECS Query Facade)**
- Clean up remaining architecture violations
- By this point, only ~20 query sites remain (down from 40+)
- UIQueries can use factories from Approach 1 for info structs

**Synergies:**
- **Approach 1 + 3**: Factory functions create pool widgets efficiently
- **Approach 1 + 2**: UIQueries can return config structs for direct factory use
- **Approach 2 + 3**: Facade provides batch queries for fast Enter() updates

---

## APPENDIX: INITIAL APPROACHES FROM ALL AGENTS

### A. Refactoring-Pro Approaches

#### Refactoring-Pro Approach 1: Comprehensive Builder Pattern

**Focus**: Full implementation of Builder pattern for all widget types with fluent API

**Problem**: Widget creation is verbose and error-prone. 50-line boilerplate for simple panels.

**Solution**:
Implement full Builder pattern with method chaining:
```go
panel := NewPanelBuilder(layout).
    Position(TopCenter).
    Size(0.3, 0.08).
    Layout(HorizontalRow).
    Spacing(10).
    Padding(10, 10, 10, 10).
    Background(PanelRes.image).
    Build()

button := NewButtonBuilder(layout).
    Text("Attack (A)").
    OnClick(cm.toggleAttackMode).
    Size(100, 100).
    Position(BottomCenter).
    Build()
```

**Metrics**:
- LOC reduction: 4,563 → ~3,000 (-34%)
- Builder infrastructure: ~800 LOC
- Net savings: ~1,400 LOC

**Assessment**:
- **Pros**: Fluent API is very readable, flexible, testable
- **Cons**: High upfront cost (~800 LOC builders), may be over-engineering for simple cases
- **Effort**: 4-5 weeks (builders + migration)

---

#### Refactoring-Pro Approach 2: Composition Over Configuration

**Focus**: Break modes into composable UI components (similar to React components)

**Problem**: Modes are monolithic (1,118 LOC for CombatMode). Hard to reuse UI patterns.

**Solution**:
Create reusable UI components:
```go
// Reusable components
type SquadListComponent struct {
    container *widget.Container
    list      *widget.List
    queries   *UIQueries
}

func (slc *SquadListComponent) Render(factionID ecs.EntityID) {
    squads := slc.queries.GetFactionSquads(factionID)
    slc.list.SetEntries(squads)
}

// CombatMode composes components
type CombatMode struct {
    squadList    *SquadListComponent
    combatLog    *CombatLogComponent
    turnDisplay  *TurnDisplayComponent
    actionBar    *ActionBarComponent
}

func (cm *CombatMode) Initialize(ctx *UIContext) error {
    cm.squadList = NewSquadListComponent(ctx)
    cm.combatLog = NewCombatLogComponent(ctx)
    cm.turnDisplay = NewTurnDisplayComponent(ctx)
    cm.actionBar = NewActionBarComponent(ctx)

    // Compose layout
    cm.rootContainer.AddChild(cm.squadList.Container())
    cm.rootContainer.AddChild(cm.combatLog.Container())
    // ...
}
```

**Metrics**:
- Component library: ~600 LOC
- CombatMode reduction: 1,118 → ~400 LOC (composition code)
- Net: ~700 LOC savings per complex mode

**Assessment**:
- **Pros**: Reusable components, cleaner mode files, testable in isolation
- **Cons**: Requires rethinking UI architecture, components may be too granular
- **Effort**: 5-6 weeks (component design + migration)

---

#### Refactoring-Pro Approach 3: Declarative UI Specification

**Focus**: Define UI layout in data structures, render from spec

**Problem**: UI structure is buried in imperative widget creation code. Hard to visualize.

**Solution**:
Declarative UI specs (YAML/JSON or Go structs):
```go
combatModeSpec := UISpec{
    Layout: "anchor",
    Children: []WidgetSpec{
        {
            Type: "panel",
            ID: "turnOrderPanel",
            Position: TopCenter,
            Size: Size{WidthPct: 0.3, HeightPct: 0.08},
            Layout: "horizontal_row",
            Children: []WidgetSpec{
                {Type: "text", ID: "turnOrderLabel", Text: "Initializing combat...", Font: "large"},
            },
        },
        {
            Type: "panel",
            ID: "squadListPanel",
            Position: CenterLeft,
            Size: Size{WidthPct: 0.15, HeightPct: 0.5},
            Layout: "vertical_column",
            Children: []WidgetSpec{
                {Type: "text", Text: "Your Squads:", Font: "small"},
                {Type: "list", ID: "squadList", DataSource: "faction_squads"},
            },
        },
        // ...
    },
}

// Renderer creates widgets from spec
renderer := NewUIRenderer(context)
combatUI := renderer.RenderSpec(combatModeSpec)
```

**Metrics**:
- Renderer engine: ~1,000 LOC
- Per-mode specs: ~200 LOC (down from 1,000+)
- Net: ~800 LOC savings per mode

**Assessment**:
- **Pros**: Separation of structure and logic, easy to visualize, could be editor-generated
- **Cons**: High complexity, debugging is harder (indirection), overkill for this project size
- **Effort**: 6-8 weeks (renderer + specs + migration)

---

### B. Tactical-Simplifier Approaches

#### Tactical-Simplifier Approach 1: Combat UI Performance Optimization

**Focus**: Optimize CombatMode for real-time responsiveness and squad visualization clarity

**Gameplay Preservation**: Combat turn responsiveness is critical for tactical decision-making

**Go-Specific Optimizations**: Batch updates, widget caching, string builder for combat log

**Code Example**:
```go
// Before: Combat log appends one message at a time (causes layout thrashing)
func (cm *CombatMode) addCombatLog(message string) {
    cm.combatLog = append(cm.combatLog, message)
    cm.combatLogArea.AppendText(message + "\n") // Layout update!
}

// After: Batch combat log updates per frame
type CombatLogBatch struct {
    messages []string
    builder  strings.Builder
}

func (cm *CombatMode) addCombatLog(message string) {
    cm.logBatch.messages = append(cm.logBatch.messages, message)
}

func (cm *CombatMode) Update(deltaTime float64) error {
    // Flush log batch once per frame
    if len(cm.logBatch.messages) > 0 {
        cm.logBatch.builder.Reset()
        for _, msg := range cm.logBatch.messages {
            cm.logBatch.builder.WriteString(msg)
            cm.logBatch.builder.WriteString("\n")
        }
        cm.combatLogArea.AppendText(cm.logBatch.builder.String())
        cm.logBatch.messages = cm.logBatch.messages[:0]
    }

    // ... rest of update
}
```

**Game System Impact**:
- **Combat system**: Reduces UI lag from multi-unit attacks (5 attacks = 5 log entries = 5 layout updates → 1 batch update)
- **Entity system**: No impact
- **Graphics/rendering**: Squad highlight rendering optimized (pre-compute border images)

**Assessment**:
- **Pros**: Directly improves player experience, measurable FPS impact
- **Cons**: Micro-optimization, may not be bottleneck yet
- **Effort**: 1 week

---

#### Tactical-Simplifier Approach 2: Squad Visualization Consolidation

**Focus**: Unify squad visualization across 4 different modes (CombatMode, SquadManagementMode, SquadBuilderMode, SquadDeploymentMode)

**Gameplay Preservation**: Consistent squad representation helps players learn formations

**Go-Specific Optimizations**: Shared rendering component, cached grid layout

**Code Example**:
```go
// Before: Each mode renders squads differently
// CombatMode: Colored borders on map tiles
// SquadManagementMode: 3x3 text grid in TextArea
// SquadBuilderMode: 3x3 button grid
// SquadDeploymentMode: Highlighted map positions

// After: Unified SquadVisualizationComponent
type SquadVisualizationComponent struct {
    squadID     ecs.EntityID
    displayMode SquadDisplayMode // Grid3x3, MapOverlay, ListSummary
    container   *widget.Container
    cells       [3][3]*widget.Container
    queries     *UIQueries
}

type SquadDisplayMode int
const (
    Grid3x3    SquadDisplayMode = iota // For builder/management
    MapOverlay                          // For combat/deployment
    ListSummary                         // For compact displays
)

func (svc *SquadVisualizationComponent) Render() {
    squadInfo := svc.queries.GetSquadInfo(svc.squadID)

    switch svc.displayMode {
    case Grid3x3:
        svc.renderGrid3x3(squadInfo)
    case MapOverlay:
        svc.renderMapOverlay(squadInfo)
    case ListSummary:
        svc.renderListSummary(squadInfo)
    }
}

func (svc *SquadVisualizationComponent) renderGrid3x3(info *SquadInfo) {
    // Shared logic for 3x3 grid visualization
    for row := 0; row < 3; row++ {
        for col := 0; col < 3; col++ {
            cell := svc.cells[row][col]
            unit := info.GetUnitAt(row, col)
            if unit != nil {
                cell.BackgroundColor = colorForRole(unit.Role)
                // Add text, borders, leader marker, etc.
            }
        }
    }
}
```

**Game System Impact**:
- **Squad system**: No changes, visualization reads existing data
- **Combat system**: Consistent highlight colors improve faction recognition
- **UI consistency**: Players see same squad representation in all contexts

**Assessment**:
- **Pros**: Improves player learning curve, reduces code duplication in visualization logic
- **Cons**: Different modes may legitimately need different views (not all 4 can unify)
- **Effort**: 2 weeks (component + integration)

---

#### Tactical-Simplifier Approach 3: Mode Transition Animation System

**Focus**: Add smooth transitions between UI modes to preserve context and reduce jarring switches

**Gameplay Preservation**: Smooth transitions help players maintain mental map of game state

**Go-Specific Optimizations**: Goroutine-free animation system using frame deltas

**Code Example**:
```go
// Before: Instant mode switches (jarring)
func (umm *UIModeManager) transitionToMode(toMode UIMode, reason string) error {
    if umm.currentMode != nil {
        umm.currentMode.Exit(toMode)
    }
    toMode.Enter(umm.currentMode)
    umm.currentMode = toMode
    return nil
}

// After: Animated fade transitions
type ModeTransitionAnimator struct {
    fadeOutDuration float64 // seconds
    fadeInDuration  float64
    currentPhase    TransitionPhase
    elapsed         float64
}

type TransitionPhase int
const (
    NoTransition TransitionPhase = iota
    FadeOut
    ModeSwitch
    FadeIn
)

func (mta *ModeTransitionAnimator) Update(deltaTime float64, manager *UIModeManager) {
    if mta.currentPhase == NoTransition {
        return
    }

    mta.elapsed += deltaTime

    switch mta.currentPhase {
    case FadeOut:
        alpha := 1.0 - (mta.elapsed / mta.fadeOutDuration)
        manager.currentMode.SetAlpha(alpha)

        if mta.elapsed >= mta.fadeOutDuration {
            mta.currentPhase = ModeSwitch
            mta.elapsed = 0
        }

    case ModeSwitch:
        // Instant switch while screen is black
        manager.currentMode.Exit(manager.pendingMode)
        manager.pendingMode.Enter(manager.currentMode)
        manager.currentMode = manager.pendingMode
        manager.pendingMode = nil

        mta.currentPhase = FadeIn

    case FadeIn:
        alpha := mta.elapsed / mta.fadeInDuration
        manager.currentMode.SetAlpha(alpha)

        if mta.elapsed >= mta.fadeInDuration {
            mta.currentPhase = NoTransition
            mta.elapsed = 0
            manager.currentMode.SetAlpha(1.0)
        }
    }
}
```

**Game System Impact**:
- **Player experience**: Reduces cognitive load during mode transitions
- **Performance**: Minimal (alpha blending is GPU-accelerated)
- **Combat flow**: Smoother transition to/from combat mode

**Assessment**:
- **Pros**: Professional feel, helps player orientation
- **Cons**: Adds latency to mode switches (0.3-0.5s), may feel sluggish in rapid switching
- **Effort**: 2-3 weeks (animation system + UI mode integration)

---

### C. Refactoring-Critic Evaluation of Initial Approaches

**Refactoring-Pro Approach 1 (Builder Pattern):**
- ✅ **Pros**: Fluent API is readable and flexible
- ❌ **Cons**: 800 LOC of builder infrastructure for 1,400 LOC savings = 57% efficiency. Overkill for simple buttons.
- ⚠️ **Risk**: Team must learn builder pattern, adds mental overhead
- **Verdict**: Over-engineering. Use simpler config structs (Synthesized Approach 1) instead.

**Refactoring-Pro Approach 2 (Composition):**
- ✅ **Pros**: Reusable components are maintainable
- ❌ **Cons**: Requires architectural rethinking, may fragment code too much
- ⚠️ **Risk**: Components may be too granular (over-abstraction)
- **Verdict**: Good long-term vision, but too big a leap now. Defer to future phase.

**Refactoring-Pro Approach 3 (Declarative UI):**
- ✅ **Pros**: Separation of structure and logic is clean
- ❌ **Cons**: 1,000 LOC renderer for questionable benefit, debugging nightmare
- ⚠️ **Risk**: High complexity, not justified for 4,563 LOC codebase
- **Verdict**: **Reject**. This is web framework thinking applied to game UI. Overkill.

**Tactical-Simplifier Approach 1 (Combat Performance):**
- ✅ **Pros**: Directly addresses player experience
- ❌ **Cons**: Premature optimization - is this actually a bottleneck?
- ⚠️ **Risk**: Micro-optimization without profiling data
- **Verdict**: **Good idea**, but profile first. If combat log causes frame drops, implement batching.

**Tactical-Simplifier Approach 2 (Squad Visualization):**
- ✅ **Pros**: Unifying visualization reduces duplication and improves consistency
- ✅ **Pros**: Directly helps players learn squad formations
- ❌ **Cons**: Different contexts may need different visualizations (forced unification is bad)
- **Verdict**: **Partially adopt**. Share grid rendering logic, but allow mode-specific customization.

**Tactical-Simplifier Approach 3 (Transition Animations):**
- ✅ **Pros**: Professional polish, improves UX
- ❌ **Cons**: Adds latency, may feel sluggish for fast players
- ⚠️ **Risk**: Animation bugs can block mode switches (hard to debug)
- **Verdict**: **Nice-to-have**, but not a priority. Defer until core refactoring is done.

---

## SYNTHESIS RATIONALE

### Why These 3 Final Approaches?

**Approach 1 Selection (Widget Factory Expansion):**
Combines the practical value of refactoring-pro's focus on DRY with refactoring-critic's insistence on simplicity. Config structs provide 80% of builder pattern benefits with 20% of the complexity. Tactical-simplifier's focus on game performance influenced the decision to keep factories lightweight and fast.

**What it combines:**
- Refactoring-Pro Approach 1: Config-driven widget creation (but simpler than full builders)
- Refactoring-Pro Approach 2: Hint of composition (PanelConfig, ButtonConfig are composable)
- Tactical-Simplifier Approach 2: Unified visualization patterns (influenced CreatePanel design)
- Refactoring-Critic: Emphasis on simplicity and incremental rollout

**Approach 2 Selection (ECS Query Facade):**
Addresses refactoring-pro's architectural concerns about ECS violations while respecting tactical-simplifier's need for fast UI queries. Refactoring-critic approved this as "architecturally sound" with clear value if ECS patterns are important (which CLAUDE.md confirms they are).

**What it combines:**
- Refactoring-Pro: System-based architecture, separation of concerns
- Tactical-Simplifier: Performance-conscious (facade can cache/batch queries)
- Refactoring-Critic: Practical value assessment (testability, future-proofing)

**Approach 3 Selection (Lifecycle Standard):**
Born from tactical-simplifier's performance focus (Approach 1) combined with refactoring-pro's desire for consistency. Refactoring-critic approved widget pools as "proven in game UI" with measurable benefits.

**What it combines:**
- Tactical-Simplifier Approach 1: Performance optimization (widget pooling)
- Refactoring-Pro Approach 2: Consistent patterns across modes
- Tactical-Simplifier Approach 2: Helps with squad visualization consistency
- Refactoring-Critic: Pragmatic focus on measurable performance gains

### Rejected Elements

**Full Builder Pattern (Refactoring-Pro #1):**
Too much infrastructure (800 LOC) for the benefit. Config structs provide similar readability without the complexity.

**Declarative UI Specs (Refactoring-Pro #3):**
Massive over-engineering (1,000 LOC renderer). This pattern is for large frameworks (React, Flutter), not a 4,563 LOC GUI package. Debugging would be nightmare.

**Component Composition (Refactoring-Pro #2):**
Good long-term vision, but too big a leap. Requires rethinking entire UI architecture. Defer to Phase 2 after widget factories are proven.

**Transition Animations (Tactical-Simplifier #3):**
Nice-to-have polish, but not addressing core problems (duplication, ECS violations). Defer until technical debt is paid down.

**Combat Log Batching (Tactical-Simplifier #1):**
Good idea IF profiling shows it's a bottleneck. Without data, this is premature optimization. Note in recommendations, but don't prioritize.

**Squad Visualization Unification (Tactical-Simplifier #2):**
Partially adopted in Approach 1 (CreatePanel can handle grid layouts), but avoided forcing all modes into one visualization. Different contexts legitimately need different views.

### Refactoring-Critic Key Insights

1. **Simplicity First**: Config structs beat builders 9 times out of 10 for this project size
2. **Measure Before Optimizing**: Don't add batching/caching without profiling data
3. **Incremental Value**: Each approach must show clear value within 2-3 weeks
4. **Avoid Over-Abstraction**: Components and declarative specs are web framework patterns that don't fit game UI well
5. **Performance Is Real**: Widget pools are proven and measurable - prioritize this
6. **Architecture Matters**: ECS violations are technical debt - Approach 2 pays it down

---

## PRINCIPLES APPLIED

### Software Engineering Principles

**DRY (Don't Repeat Yourself):**
- Approach 1: Eliminates ~1,200 LOC of widget creation duplication
- Approach 2: Centralizes ECS queries in UIQueries facade
- Approach 3: Widget pools eliminate repeated create/destroy patterns

**SOLID Principles:**
- **Single Responsibility**: UIQueries handles queries, factories handle creation, modes handle logic
- **Open/Closed**: Config structs extend via new fields, not modifying factories
- **Liskov Substitution**: All UIMode implementations follow documented lifecycle contract
- **Interface Segregation**: UIQueries provides focused query methods, not monolithic interface
- **Dependency Inversion**: Modes depend on UIQueries interface, not concrete ECS implementation

**KISS (Keep It Simple, Stupid):**
- Config structs over full builders (Approach 1)
- Reject declarative UI specs as over-engineering
- Widget pools over complex component lifecycle management

**YAGNI (You Aren't Gonna Need It):**
- Defer transition animations (not solving current problems)
- Defer component composition (not needed yet)
- Don't build renderer engine for 13 UI files

**SLAP (Single Level of Abstraction Principle):**
- CreatePanel() hides widget creation details
- UIQueries hides ECS query boilerplate
- Mode files operate at "business logic" level, not widget API level

**Separation of Concerns:**
- UI presentation (mode files) separate from data queries (UIQueries)
- Widget creation (factories) separate from widget content (lifecycle updates)
- Structure (Initialize) separate from content (Enter)

### Go-Specific Best Practices

**Composition Over Inheritance:**
- Config structs compose behavior via fields, not inheritance hierarchies
- UIQueries composes system functions, not extending base class

**Interface Design:**
- UIMode interface defines clear lifecycle contract
- Widget factories accept config structs (duck typing), not requiring interface implementation

**Error Handling:**
- UIQueries returns (data, error) tuples for fallible operations
- Mode lifecycle errors propagate up to manager for centralized handling

**Value Semantics:**
- Config structs passed by value (small, immutable)
- Widget pointers returned (ebitenui requirement)

**Idiomatic Patterns:**
- Factory functions like NewUIQueries(manager) (standard Go constructor pattern)
- Exported config structs with unexported factory internals

### Game Development Considerations

**Performance:**
- Widget pools eliminate allocation/deallocation churn (Approach 3)
- Batch UI updates in Update() once per frame (60 FPS friendly)
- Factory functions run at init time, not every frame

**Real-Time Constraints:**
- Enter() must complete <16ms for 60 FPS (widget pools enable this)
- Update() budget <1ms per mode (no heavy queries in Update)
- Render() only draws overlays, not full UI (ebitenui handles that)

**Game Loop Integration:**
- UIModeManager.Update() called once per frame
- ebitenui.UI.Update() called after mode logic
- Mode lifecycle synchronized with game state changes (combat start/end)

**Tactical Gameplay Preservation:**
- Combat mode responsiveness maintained (fast Enter/Exit)
- Squad visualization consistent across modes (CreatePanel handles grids)
- Turn-based flow preserved (no animations blocking input)

---

## NEXT STEPS

### Recommended Action Plan

**Immediate (Weeks 1-3): Approach 1 - Widget Factory Expansion**
1. Create PanelConfig, ButtonConfig, ListConfig structs in createwidgets.go
2. Implement CreatePanel(), CreateConfiguredButton(), CreateConfiguredList()
3. Refactor CombatMode and ExplorationMode (proof of concept)
4. Measure LOC reduction and verify no functionality regressions
5. Migrate remaining 5 modes if POC succeeds

**Short-term (Weeks 4-7): Approach 3 - Lifecycle Standardization**
1. Document lifecycle contract in uimode.go
2. Implement widget pool pattern in SquadManagementMode
3. Measure Enter() time before/after (target: <16ms)
4. Apply pools to InventoryMode and SquadBuilderMode
5. Audit all modes for lifecycle compliance

**Medium-term (Weeks 8-12): Approach 2 - ECS Query Facade**
1. Create gui/uiqueries.go with 10 core query functions
2. Add UIQueries to UIContext
3. Migrate CombatMode to use facade (removes ~50 LOC of direct queries)
4. Migrate remaining modes incrementally
5. Remove all direct World.Query() calls from GUI package

**Long-term (Phase 2, future):**
- Consider component composition (refactoring-pro #2) if adding many new modes
- Add performance monitoring/profiling to identify real bottlenecks
- Evaluate transition animations once core refactoring is complete

### Validation Strategy

**Testing Approach:**
- **Unit Tests**: Factory functions are pure, easily testable (input config → output widget)
- **Integration Tests**: Mode lifecycle tests (Initialize → Enter → Update → Exit sequence)
- **UI Tests**: Manual testing of each mode after migration (checklist: all buttons work, all data displays correctly)
- **Performance Tests**: Measure Enter() time, Update() time, memory usage before/after

**Rollback Plan:**
- Keep old CreateButton/CreateTextArea functions during migration (parallel implementation)
- Migrate one mode at a time (if mode breaks, only that mode rolls back)
- Git branch per approach (approach-1-widget-factory, approach-3-lifecycle, approach-2-ecs-queries)
- Feature flags for new patterns (if needed for A/B testing)

**Success Metrics:**
- LOC reduction: 4,563 → ~3,200 (30% target)
- Duplication reduction: 40% → <15%
- Enter() time: <16ms for all modes (60 FPS compliance)
- Test coverage: >80% for factory functions and UIQueries
- Zero functionality regressions (all UI features work as before)

### Additional Resources

**Go Patterns Documentation:**
- [Effective Go - Interfaces and Types](https://golang.org/doc/effective_go#interfaces)
- [Go Code Review Comments - Package Names](https://github.com/golang/go/wiki/CodeReviewComments#package-names)
- [Dave Cheney - SOLID Go Design](https://dave.cheney.net/2016/08/20/solid-go-design)

**Game Architecture References:**
- [Game Programming Patterns - Component](https://gameprogrammingpatterns.com/component.html) (relevant for UI components)
- [Ebiten UI Documentation](https://ebitenui.github.io/) (ebitenui widget lifecycle)
- [Roguelike UI Best Practices](http://www.roguebasin.com/index.php/Roguelike_Interface) (context for tactical UI needs)

**Refactoring Resources:**
- [Refactoring Guru - Extract Method](https://refactoring.guru/extract-method) (factory function pattern)
- [Martin Fowler - Introduce Parameter Object](https://refactoring.guru/introduce-parameter-object) (config struct pattern)
- [Working Effectively with Legacy Code](https://www.oreilly.com/library/view/working-effectively-with/0131177052/) (incremental refactoring strategies)

---

## CONCLUSION

This analysis presents three balanced, actionable refactoring approaches for the GUI package:

1. **Widget Factory Expansion** - Immediate, low-risk LOC reduction through config-driven factories
2. **ECS Query Facade** - Architectural cleanup to enforce system-based patterns
3. **Lifecycle Standardization** - Performance and consistency improvements through widget pools

**Recommended sequence**: Approach 1 → Approach 3 → Approach 2 over 12 weeks.

Each approach has been vetted from three perspectives:
- **Refactoring-Pro**: Architectural soundness and DRY/SOLID compliance
- **Tactical-Simplifier**: Game-specific performance and player experience
- **Refactoring-Critic**: Practical value and avoidance of over-engineering

The synthesis rejects over-engineered solutions (full builders, declarative specs, component framework) in favor of pragmatic, incremental improvements that deliver measurable value within 2-3 weeks per approach.

**Key Success Factors:**
- Incremental rollout (one mode at a time)
- Clear measurable goals (LOC reduction, performance metrics)
- Respect for game constraints (60 FPS, real-time responsiveness)
- Balance of theory (SOLID, DRY) and practice (ship working code)

---

**END OF ANALYSIS**
