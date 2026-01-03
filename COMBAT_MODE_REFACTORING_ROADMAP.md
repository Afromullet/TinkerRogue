# Combat Mode Refactoring Roadmap

**File:** `gui/guicombat/combatmode.go`
**Current Size:** 1,059 lines
**Total Package Size:** 2,477 lines across 6 files
**Analysis Date:** 2026-01-02

---

## Executive Summary

The `combatmode.go` file has grown to **1,059 lines** and violates the **Single Responsibility Principle**. It currently handles:

1. UI panel construction (7 build methods)
2. Turn management and AI orchestration
3. Combat lifecycle (initialization, encounter setup, cleanup)
4. Visualization systems (danger/layer visualizers)
5. UI update coordination
6. Input routing
7. Battle recording and victory conditions
8. Entity lifecycle management

**Recommendation:** Split into **7 focused files** organized by responsibility, reducing `combatmode.go` to ~200-300 lines as a coordinator/orchestrator.

**Estimated Impact:**
- **Maintainability:** High - Each file has single, clear purpose
- **Testability:** High - Isolated concerns are easier to test
- **Cognitive Load:** Significant reduction - developers can focus on one aspect at a time
- **Line Count Reduction:** 766 lines moved from combatmode.go → specialized files (73% reduction)

---

## Current Responsibility Analysis

### 1. **UI Panel Construction** (Lines 165-270, ~105 lines)
**Methods:**
- `ensureUIFactoryInitialized()`
- `buildTurnOrderPanel()`
- `buildFactionInfoPanel()`
- `buildSquadListPanel()`
- `buildSquadDetailPanel()`
- `buildLogPanel()`
- `buildActionButtons()`
- `buildLayerStatusPanel()`

**Problem:** UI construction logic mixed with combat logic. Panel building is a separate concern from combat orchestration.

### 2. **UI Update Components** (Lines 307-397, ~90 lines)
**Methods:**
- `initializeUpdateComponents()`
- `makeCurrentFactionSquadFilter()`
- `updateLayerStatusWidget()`

**Problem:** Component initialization and update logic scattered through the file.

### 3. **Turn Management & AI Orchestration** (Lines 408-610, ~202 lines)
**Methods:**
- `handleEndTurn()`
- `executeAITurnIfNeeded()`
- `playAIAttackAnimations()`
- `playNextAIAttack()`
- `advanceAfterAITurn()`

**Problem:** Complex AI turn logic with animation chaining creates high cyclomatic complexity. This is a distinct subsystem.

### 4. **Combat Lifecycle Management** (Lines 612-691, ~79 lines)
**Methods:**
- `SetupEncounter()`
- `Enter()`
- `Exit()`
- `initialzieCombatFactions()` [sic - typo]

**Problem:** Initialization and cleanup logic is critical but scattered across multiple methods.

### 5. **Entity Cleanup** (Lines 807-920, ~113 lines)
**Methods:**
- `markEncounterDefeatedIfVictorious()`
- `cleanupCombatEntities()`

**Problem:** Complex cleanup logic with 7 steps should be isolated. Critical for preventing memory leaks.

### 6. **Visualization Systems** (Lines 127-163 in Initialize, 759-776 in Update, ~35 lines)
**Code:**
- Danger visualizer setup and updates
- Layer visualizer setup and updates
- Threat manager integration

**Problem:** Visualization is a cross-cutting concern that appears in multiple methods.

### 7. **Input Handling** (Lines 964-1058, ~94 lines)
**Method:**
- `HandleInput()` - Danger visualizer toggling (H key), Layer visualizer (L key), input delegation

**Problem:** Specialized input handling for visualization tools mixed with general input routing.

### 8. **Button Click Handlers** (Lines 291-305, ~14 lines)
**Methods:**
- `handleAttackClick()`
- `handleMoveClick()`
- `handleUndoMove()`
- `handleRedoMove()`
- `handleFlee()`

**Problem:** Simple delegation methods add noise; should be inline or grouped.

---

## Proposed Architecture

### File Structure Overview

```
gui/guicombat/
├── combatmode.go                    (~200-300 lines) - Main coordinator
├── combat_ui_builder.go             (~150 lines)     - UI panel construction
├── combat_lifecycle_manager.go      (~200 lines)     - Initialization & cleanup
├── combat_turn_orchestrator.go      (~250 lines)     - Turn management & AI
├── combat_visualization_manager.go  (~120 lines)     - Danger/layer visualizers
├── combat_ui_updater.go             (~100 lines)     - UI component updates
├── combat_input_router.go           (~100 lines)     - Specialized input routing
│
├── combat_action_handler.go         (existing)       - Action execution
├── combat_input_handler.go          (existing)       - Basic input processing
├── combat_log_manager.go            (existing)       - Log management
├── combat_animation_mode.go         (existing)       - Animation system
└── squad_combat_renderer.go         (existing)       - Squad rendering
```

**Total New Files:** 6
**Lines Moved:** ~920 lines
**Remaining in combatmode.go:** ~200-300 lines (orchestration only)

---

## Detailed Refactoring Plan

### Phase 1: UI Construction Separation

**New File:** `combat_ui_builder.go`

**Purpose:** Handle all UI panel construction and factory initialization.

**Responsibilities:**
- Panel creation (7 build methods)
- UI factory initialization
- Layout configuration

**Code to Move:**
```go
// FROM combatmode.go (lines 165-270)

type CombatUIBuilder struct {
    uiFactory  *gui.UIComponentFactory
    queries    *guicomponents.GUIQueries
    builders   gui.PanelBuilders
    layout     gui.ScreenLayout
}

func NewCombatUIBuilder(queries *guicomponents.GUIQueries, builders gui.PanelBuilders, layout gui.ScreenLayout) *CombatUIBuilder

// All build* methods moved here:
func (cub *CombatUIBuilder) BuildTurnOrderPanel() (*widget.Container, *widget.Text)
func (cub *CombatUIBuilder) BuildFactionInfoPanel() (*widget.Container, *widget.Text)
func (cub *CombatUIBuilder) BuildSquadListPanel() *widget.Container
func (cub *CombatUIBuilder) BuildSquadDetailPanel() (*widget.Container, *widget.Text)
func (cub *CombatUIBuilder) BuildLogPanel() (*widget.Container, *widgets.CachedTextAreaWrapper)
func (cub *CombatUIBuilder) BuildActionButtons(callbacks ActionButtonCallbacks) *widget.Container
func (cub *CombatUIBuilder) BuildLayerStatusPanel() (*widget.Container, *widget.Text)
```

**Integration in combatmode.go:**
```go
type CombatMode struct {
    // ... existing fields ...
    uiBuilder *CombatUIBuilder
}

// In Initialize():
cm.uiBuilder = NewCombatUIBuilder(cm.Queries, cm.PanelBuilders, cm.Layout)

// Simplified panel building:
err := gui.NewModeBuilder(&cm.BaseMode, gui.ModeConfig{
    ModeName:   "combat",
    ReturnMode: "exploration",
    Panels: []gui.PanelSpec{
        {CustomBuild: func() *widget.Container {
            panel, label := cm.uiBuilder.BuildTurnOrderPanel()
            cm.turnOrderLabel = label
            cm.turnOrderPanel = panel
            return panel
        }},
        // ... similar for other panels ...
    },
}).Build(ctx)
```

**Value Proposition:**
- **Line Reduction:** ~105 lines moved from combatmode.go
- **Maintainability:** UI construction logic isolated and testable
- **Extensibility:** Easy to add new panels without touching core combat logic
- **Single Responsibility:** Builder only concerned with UI construction

**Implementation Risk:** Low - Pure extraction, minimal dependencies

**Effort Estimate:** 2-3 hours

---

### Phase 2: UI Update Coordination

**New File:** `combat_ui_updater.go`

**Purpose:** Coordinate UI component updates and state synchronization.

**Responsibilities:**
- Initialize UI update components
- Manage UI refresh triggers
- Component lifecycle (create, update, refresh)
- Filter factories for squad lists

**Code to Move:**
```go
// FROM combatmode.go (lines 307-397)

type CombatUIUpdater struct {
    // Component references
    turnOrderComponent   *guicomponents.TextDisplayComponent
    factionInfoComponent *guicomponents.DetailPanelComponent
    squadListComponent   *guicomponents.SquadListComponent
    squadDetailComponent *guicomponents.DetailPanelComponent

    // Widget references
    turnOrderLabel  *widget.Text
    factionInfoText *widget.Text
    squadListPanel  *widget.Container
    squadDetailText *widget.Text
    layerStatusText *widget.Text
    layerStatusPanel *widget.Container

    // Dependencies
    queries       *guicomponents.GUIQueries
    combatService *combatservices.CombatService
    actionHandler *CombatActionHandler
    layerVisualizer *behavior.LayerVisualizer
}

func NewCombatUIUpdater(
    widgets CombatUIWidgets,
    queries *guicomponents.GUIQueries,
    combatService *combatservices.CombatService,
    actionHandler *CombatActionHandler,
) *CombatUIUpdater

// Initialization
func (cuu *CombatUIUpdater) InitializeComponents()

// Update methods
func (cuu *CombatUIUpdater) RefreshTurnOrder()
func (cuu *CombatUIUpdater) RefreshFactionInfo(factionID ecs.EntityID)
func (cuu *CombatUIUpdater) RefreshSquadList()
func (cuu *CombatUIUpdater) RefreshSquadDetail(squadID ecs.EntityID)
func (cuu *CombatUIUpdater) UpdateLayerStatus(visualizer *behavior.LayerVisualizer)

// State-driven updates (called from Update loop)
func (cuu *CombatUIUpdater) UpdateIfFactionChanged(currentFactionID, lastFactionID ecs.EntityID) bool
func (cuu *CombatUIUpdater) UpdateIfSquadChanged(currentSquadID, lastSquadID ecs.EntityID) bool

// Filter factory
func (cuu *CombatUIUpdater) MakeCurrentFactionSquadFilter() guicomponents.SquadFilter
```

**Integration in combatmode.go:**
```go
type CombatMode struct {
    // ... existing fields ...
    uiUpdater *CombatUIUpdater

    // State tracking (stays here - orchestration concern)
    lastFactionID     ecs.EntityID
    lastSelectedSquad ecs.EntityID
}

// In Initialize() - after UI is built:
cm.uiUpdater = NewCombatUIUpdater(
    CombatUIWidgets{
        TurnOrderLabel:  cm.turnOrderLabel,
        FactionInfoText: cm.factionInfoText,
        // ... etc
    },
    cm.Queries,
    cm.combatService,
    cm.actionHandler,
)
cm.uiUpdater.InitializeComponents()

// In Update():
func (cm *CombatMode) Update(deltaTime float64) error {
    currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
    if cm.uiUpdater.UpdateIfFactionChanged(currentFactionID, cm.lastFactionID) {
        cm.lastFactionID = currentFactionID
    }

    battleState := cm.Context.ModeCoordinator.GetBattleMapState()
    if cm.uiUpdater.UpdateIfSquadChanged(battleState.SelectedSquadID, cm.lastSelectedSquad) {
        cm.lastSelectedSquad = battleState.SelectedSquadID
    }

    // ... visualization updates ...
    return nil
}
```

**Value Proposition:**
- **Line Reduction:** ~90 lines moved from combatmode.go
- **Testability:** UI update logic can be unit tested independently
- **Clarity:** Clear separation between "what to update" (orchestrator) and "how to update" (updater)
- **Performance:** Update optimization logic centralized

**Implementation Risk:** Low - Clear boundaries, existing component abstractions

**Effort Estimate:** 2-3 hours

---

### Phase 3: Turn Management & AI Orchestration

**New File:** `combat_turn_orchestrator.go`

**Purpose:** Manage turn flow, AI execution, and animation sequencing.

**Responsibilities:**
- Turn advancement logic
- AI turn detection and execution
- Attack animation sequencing
- Turn state updates and logging

**Code to Move:**
```go
// FROM combatmode.go (lines 408-610)

type CombatTurnOrchestrator struct {
    combatService *combatservices.CombatService
    logManager    *CombatLogManager
    queries       *guicomponents.GUIQueries
    modeManager   *core.UIModeManager
    combatLogArea *widgets.CachedTextAreaWrapper
    threatManager *behavior.FactionThreatLevelManager
    threatEvaluator *behavior.CompositeThreatEvaluator
}

func NewCombatTurnOrchestrator(
    combatService *combatservices.CombatService,
    logManager *CombatLogManager,
    queries *guicomponents.GUIQueries,
    modeManager *core.UIModeManager,
    combatLogArea *widgets.CachedTextAreaWrapper,
    threatManager *behavior.FactionThreatLevelManager,
    threatEvaluator *behavior.CompositeThreatEvaluator,
) *CombatTurnOrchestrator

// Turn management
func (cto *CombatTurnOrchestrator) EndCurrentTurn() error
func (cto *CombatTurnOrchestrator) ExecuteAITurnIfNeeded()
func (cto *CombatTurnOrchestrator) AdvanceAfterAITurn()

// AI animation sequencing
func (cto *CombatTurnOrchestrator) PlayAIAttackAnimations(aiController *combatservices.AIController)
func (cto *CombatTurnOrchestrator) PlayNextAIAttack(attacks []combatservices.QueuedAttack, index int, aiController *combatservices.AIController)

// Callbacks for UI updates
type TurnUpdateCallbacks struct {
    OnTurnChanged    func(factionID ecs.EntityID, round int)
    OnMarkSquadsDirty func()
}

func (cto *CombatTurnOrchestrator) SetUpdateCallbacks(callbacks TurnUpdateCallbacks)
```

**Integration in combatmode.go:**
```go
type CombatMode struct {
    // ... existing fields ...
    turnOrchestrator *CombatTurnOrchestrator
}

// In Initialize():
cm.turnOrchestrator = NewCombatTurnOrchestrator(
    cm.combatService,
    cm.logManager,
    cm.Queries,
    cm.ModeManager,
    cm.combatLogArea,
    cm.threatManager,
    cm.threatEvaluator,
)
cm.turnOrchestrator.SetUpdateCallbacks(TurnUpdateCallbacks{
    OnTurnChanged: func(factionID ecs.EntityID, round int) {
        cm.uiUpdater.RefreshTurnOrder()
        cm.uiUpdater.RefreshFactionInfo(factionID)
        cm.uiUpdater.RefreshSquadList()
        cm.squadDetailComponent.SetText("Select a squad\nto view details")
    },
    OnMarkSquadsDirty: cm.Queries.MarkAllSquadsDirty,
})

// Simplified handler:
func (cm *CombatMode) handleEndTurn() {
    cm.actionHandler.ClearMoveHistory()
    cm.turnOrchestrator.EndCurrentTurn()
}
```

**Value Proposition:**
- **Line Reduction:** ~202 lines moved from combatmode.go
- **Complexity Reduction:** High - isolates complex AI animation chaining
- **Testability:** Turn logic can be tested with mock callbacks
- **Single Responsibility:** Orchestrator only manages turn flow, not UI

**Implementation Risk:** Medium - Complex animation sequencing requires careful extraction

**Effort Estimate:** 4-5 hours

---

### Phase 4: Combat Lifecycle Management

**New File:** `combat_lifecycle_manager.go`

**Purpose:** Handle combat initialization, encounter setup, and entity cleanup.

**Responsibilities:**
- Encounter setup and spawning
- Combat faction initialization
- Entity cleanup on combat exit
- Victory condition checking and encounter marking

**Code to Move:**
```go
// FROM combatmode.go (lines 612-651, 778-836, 839-920)

type CombatLifecycleManager struct {
    ecsManager       *common.EntityManager
    combatService    *combatservices.CombatService
    logManager       *CombatLogManager
    queries          *guicomponents.GUIQueries
    combatLogArea    *widgets.CachedTextAreaWrapper

    // Encounter tracking
    currentEncounterID ecs.EntityID
}

func NewCombatLifecycleManager(
    ecsManager *common.EntityManager,
    combatService *combatservices.CombatService,
    logManager *CombatLogManager,
    queries *guicomponents.GUIQueries,
    combatLogArea *widgets.CachedTextAreaWrapper,
) *CombatLifecycleManager

// Initialization
func (clm *CombatLifecycleManager) SetupEncounter(encounterID ecs.EntityID, playerStartPos coords.LogicalPosition) error
func (clm *CombatLifecycleManager) InitializeCombatFactions() error
func (clm *CombatLifecycleManager) StartBattle(fromMode core.UIMode, battleRecorder *battlelog.BattleRecorder) error

// Cleanup
func (clm *CombatLifecycleManager) CleanupCombatEntities()
func (clm *CombatLifecycleManager) MarkEncounterDefeatedIfVictorious()

// Victory
func (clm *CombatLifecycleManager) FinalizeBattle(battleRecorder *battlelog.BattleRecorder) error

// Getters
func (clm *CombatLifecycleManager) GetCurrentEncounterID() ecs.EntityID
```

**Integration in combatmode.go:**
```go
type CombatMode struct {
    // ... existing fields ...
    lifecycleManager *CombatLifecycleManager
}

// In Initialize():
cm.lifecycleManager = NewCombatLifecycleManager(
    ctx.ECSManager,
    cm.combatService,
    cm.logManager,
    cm.Queries,
    cm.combatLogArea,
)

// Simplified Enter():
func (cm *CombatMode) Enter(fromMode core.UIMode) error {
    fmt.Println("Entering Combat Mode")

    isComingFromAnimation := fromMode != nil && fromMode.GetModeName() == "combat_animation"

    if !isComingFromAnimation {
        // Get encounter from battle state
        battleMapState := cm.Context.ModeCoordinator.GetBattleMapState()
        encounterID := battleMapState.TriggeredEncounterID
        playerStartPos := *cm.Context.PlayerData.Pos

        // Start battle using lifecycle manager
        if err := cm.lifecycleManager.StartBattle(fromMode, cm.combatService.BattleRecorder); err != nil {
            return err
        }
    }

    // Refresh UI
    currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
    if currentFactionID != 0 {
        cm.uiUpdater.RefreshTurnOrder()
        cm.uiUpdater.RefreshFactionInfo(currentFactionID)
        cm.uiUpdater.RefreshSquadList()
    }

    return nil
}

// Simplified Exit():
func (cm *CombatMode) Exit(toMode core.UIMode) error {
    fmt.Println("Exiting Combat Mode")

    isToAnimation := toMode != nil && toMode.GetModeName() == "combat_animation"

    if !isToAnimation {
        cm.lifecycleManager.MarkEncounterDefeatedIfVictorious()
        cm.lifecycleManager.FinalizeBattle(cm.combatService.BattleRecorder)
        cm.lifecycleManager.CleanupCombatEntities()
    }

    // Clear visualizations
    if cm.visualizationManager != nil {
        cm.visualizationManager.ClearAll()
    }

    cm.logManager.Clear()
    return nil
}
```

**Value Proposition:**
- **Line Reduction:** ~292 lines moved from combatmode.go
- **Critical Logic Isolation:** Complex cleanup logic separated and testable
- **Memory Safety:** Entity lifecycle management in one place reduces leak risk
- **Clarity:** Clear separation of "when" (Enter/Exit) vs "how" (lifecycle manager)

**Implementation Risk:** Medium - Entity cleanup is critical; requires thorough testing

**Effort Estimate:** 5-6 hours (includes comprehensive testing)

---

### Phase 5: Visualization System Management

**New File:** `combat_visualization_manager.go`

**Purpose:** Coordinate danger/layer visualization systems.

**Responsibilities:**
- Visualization system initialization
- Update coordination during combat
- Toggle and mode switching
- Status widget updates

**Code to Move:**
```go
// FROM combatmode.go (scattered: lines 127-163, 273-288, 759-776 in Update, 986-1055 in HandleInput)

type CombatVisualizationManager struct {
    dangerVisualizer  *behavior.DangerVisualizer
    layerVisualizer   *behavior.LayerVisualizer
    threatManager     *behavior.FactionThreatLevelManager
    threatEvaluator   *behavior.CompositeThreatEvaluator

    // UI elements for status display
    layerStatusPanel *widget.Container
    layerStatusText  *widget.Text

    // Dependencies
    ecsManager  *common.EntityManager
    gameMap     *worldmap.GameMap
    combatCache *combat.CombatCache
    logManager  *CombatLogManager
    combatLogArea *widgets.CachedTextAreaWrapper
}

func NewCombatVisualizationManager(
    ecsManager *common.EntityManager,
    gameMap *worldmap.GameMap,
    combatCache *combat.CombatCache,
    logManager *CombatLogManager,
    combatLogArea *widgets.CachedTextAreaWrapper,
) *CombatVisualizationManager

// Initialization
func (cvm *CombatVisualizationManager) Initialize(allFactions []ecs.EntityID, playerFactionID ecs.EntityID)
func (cvm *CombatVisualizationManager) SetStatusWidgets(panel *widget.Container, text *widget.Text)

// Update
func (cvm *CombatVisualizationManager) Update(
    currentFactionID ecs.EntityID,
    currentRound int,
    playerPos coords.LogicalPosition,
    viewportSize int,
)
func (cvm *CombatVisualizationManager) UpdateLayerStatusWidget()
func (cvm *CombatVisualizationManager) UpdateThreatManager()

// Input handling
func (cvm *CombatVisualizationManager) HandleDangerToggle(shiftPressed bool)
func (cvm *CombatVisualizationManager) HandleDangerMetricCycle()
func (cvm *CombatVisualizationManager) HandleLayerToggle(shiftPressed bool)

// Cleanup
func (cvm *CombatVisualizationManager) ClearAll()

// Getters
func (cvm *CombatVisualizationManager) GetThreatManager() *behavior.FactionThreatLevelManager
func (cvm *CombatVisualizationManager) GetThreatEvaluator() *behavior.CompositeThreatEvaluator
```

**Integration in combatmode.go:**
```go
type CombatMode struct {
    // ... existing fields ...
    visualizationManager *CombatVisualizationManager
}

// In Initialize():
gameMap := ctx.GameMap.(*worldmap.GameMap)
cm.visualizationManager = NewCombatVisualizationManager(
    ctx.ECSManager,
    gameMap,
    cm.Queries.CombatCache,
    cm.logManager,
    cm.combatLogArea,
)

allFactions := cm.Queries.GetAllFactions()
if len(allFactions) > 0 {
    cm.visualizationManager.Initialize(allFactions, allFactions[0])
}

// After UI is built:
cm.visualizationManager.SetStatusWidgets(cm.layerStatusPanel, cm.layerStatusText)

// In Update():
func (cm *CombatMode) Update(deltaTime float64) error {
    // ... state change updates ...

    currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
    currentRound := cm.combatService.TurnManager.GetCurrentRound()
    playerPos := *cm.Context.PlayerData.Pos

    cm.visualizationManager.Update(currentFactionID, currentRound, playerPos, 30)

    return nil
}

// In HandleInput():
func (cm *CombatMode) HandleInput(inputState *core.InputState) bool {
    // ... existing input handling ...

    // H key - danger visualization
    if inputState.KeysJustPressed[ebiten.KeyH] {
        shiftPressed := inputState.KeysPressed[ebiten.KeyShift] ||
            inputState.KeysPressed[ebiten.KeyShiftLeft] ||
            inputState.KeysPressed[ebiten.KeyShiftRight]
        cm.visualizationManager.HandleDangerToggle(shiftPressed)
        return true
    }

    // Ctrl - danger metric cycle
    if inputState.KeysJustPressed[ebiten.KeyControlLeft] {
        cm.visualizationManager.HandleDangerMetricCycle()
        return true
    }

    // L key - layer visualization
    if inputState.KeysJustPressed[ebiten.KeyL] {
        shiftPressed := inputState.KeysPressed[ebiten.KeyShift] ||
            inputState.KeysPressed[ebiten.KeyShiftLeft] ||
            inputState.KeysPressed[ebiten.KeyShiftRight]
        cm.visualizationManager.HandleLayerToggle(shiftPressed)
        return true
    }

    return false
}
```

**Value Proposition:**
- **Line Reduction:** ~120 lines moved from combatmode.go
- **Consolidation:** Scattered visualization code unified
- **Encapsulation:** Visualization systems hidden behind clean interface
- **Input Handling:** Specialized input separated from general input routing

**Implementation Risk:** Low-Medium - Scattered code requires careful extraction

**Effort Estimate:** 3-4 hours

---

### Phase 6: Specialized Input Routing

**New File:** `combat_input_router.go`

**Purpose:** Route specialized input (visualization toggles) separate from basic combat input.

**Responsibilities:**
- Visualization hotkey handling (H, L, Ctrl keys)
- Delegation to visualization manager
- Common input handling (ESC)

**Code to Move:**
```go
// FROM combatmode.go (lines 964-1058)

type CombatInputRouter struct {
    visualizationManager *CombatVisualizationManager
    inputHandler         *CombatInputHandler
    combatService        *combatservices.CombatService
    playerData           *core.PlayerData
}

func NewCombatInputRouter(
    visualizationManager *CombatVisualizationManager,
    inputHandler *CombatInputHandler,
    combatService *combatservices.CombatService,
    playerData *core.PlayerData,
) *CombatInputRouter

func (cir *CombatInputRouter) HandleInput(inputState *core.InputState, onEndTurn func()) bool
func (cir *CombatInputRouter) handleVisualizationInput(inputState *core.InputState) bool
```

**Integration in combatmode.go:**
```go
type CombatMode struct {
    // ... existing fields ...
    inputRouter *CombatInputRouter
}

// In Initialize() - after all managers created:
cm.inputRouter = NewCombatInputRouter(
    cm.visualizationManager,
    cm.inputHandler,
    cm.combatService,
    cm.Context.PlayerData,
)

// Simplified HandleInput():
func (cm *CombatMode) HandleInput(inputState *core.InputState) bool {
    // Handle common input (ESC)
    if cm.HandleCommonInput(inputState) {
        return true
    }

    // Delegate to input router
    return cm.inputRouter.HandleInput(inputState, cm.handleEndTurn)
}
```

**Value Proposition:**
- **Line Reduction:** ~94 lines moved from combatmode.go
- **Separation of Concerns:** Visualization input isolated
- **Maintainability:** Easy to add new hotkeys without touching core mode
- **Clarity:** Clear delegation hierarchy

**Implementation Risk:** Low - Simple extraction

**Effort Estimate:** 1-2 hours

---

### Phase 7: Final Cleanup - Inline Button Handlers

**Location:** combatmode.go (lines 291-305)

**Purpose:** Remove trivial delegation methods that add no value.

**Current Code:**
```go
func (cm *CombatMode) handleAttackClick() {
    cm.actionHandler.ToggleAttackMode()
}

func (cm *CombatMode) handleMoveClick() {
    cm.actionHandler.ToggleMoveMode()
}

func (cm *CombatMode) handleUndoMove() {
    cm.actionHandler.UndoLastMove()
}

func (cm *CombatMode) handleRedoMove() {
    cm.actionHandler.RedoLastMove()
}
```

**Refactored Code:**
```go
// In buildActionButtons() - now in combat_ui_builder.go
func (cub *CombatUIBuilder) BuildActionButtons(actionHandler *CombatActionHandler, onEndTurn, onFlee func()) *widget.Container {
    return cub.uiFactory.CreateCombatActionButtons(
        actionHandler.ToggleAttackMode,    // Direct method reference
        actionHandler.ToggleMoveMode,      // Direct method reference
        actionHandler.UndoLastMove,        // Direct method reference
        actionHandler.RedoLastMove,        // Direct method reference
        onEndTurn,                         // Still needs wrapper (has logic)
        onFlee,                            // Still needs wrapper (has logic)
    )
}

// In combatmode.go - keep only handlers with logic:
func (cm *CombatMode) handleEndTurn() {
    cm.actionHandler.ClearMoveHistory()
    cm.turnOrchestrator.EndCurrentTurn()
}

func (cm *CombatMode) handleFlee() {
    cm.logManager.UpdateTextArea(cm.combatLogArea, "Fleeing from combat...")
    if exploreMode, exists := cm.ModeManager.GetMode("exploration"); exists {
        cm.ModeManager.RequestTransition(exploreMode, "Fled from combat")
    }
}
```

**Value Proposition:**
- **Line Reduction:** 8 lines removed (trivial handlers eliminated)
- **Directness:** Button callbacks directly invoke action handler methods
- **Clarity:** Less indirection improves code readability

**Implementation Risk:** Very Low - Simple refactoring

**Effort Estimate:** 30 minutes

---

## Final Architecture: combatmode.go

After all refactoring phases, `combatmode.go` becomes a **lean orchestrator** (~200-300 lines):

```go
package guicombat

type CombatMode struct {
    gui.BaseMode

    // Specialized managers (each handles one concern)
    uiBuilder            *CombatUIBuilder
    uiUpdater            *CombatUIUpdater
    lifecycleManager     *CombatLifecycleManager
    turnOrchestrator     *CombatTurnOrchestrator
    visualizationManager *CombatVisualizationManager
    inputRouter          *CombatInputRouter

    // Existing handlers (already separated)
    logManager    *CombatLogManager
    actionHandler *CombatActionHandler
    inputHandler  *CombatInputHandler

    // Service layer
    combatService *combatservices.CombatService

    // Rendering
    movementRenderer  *guimodes.MovementTileRenderer
    highlightRenderer *guimodes.SquadHighlightRenderer

    // UI widgets (references for updates)
    turnOrderPanel   *widget.Container
    factionInfoPanel *widget.Container
    // ... other panels (kept for widget tree structure)

    // Minimal state tracking (orchestration concern)
    lastFactionID     ecs.EntityID
    lastSelectedSquad ecs.EntityID
}

func NewCombatMode(modeManager *core.UIModeManager) *CombatMode

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    // 1. Create combat service
    cm.combatService = combatservices.NewCombatService(ctx.ECSManager)

    // 2. Build UI
    cm.uiBuilder = NewCombatUIBuilder(...)
    cm.buildAllPanels(ctx)

    // 3. Initialize managers
    cm.lifecycleManager = NewCombatLifecycleManager(...)
    cm.turnOrchestrator = NewCombatTurnOrchestrator(...)
    cm.visualizationManager = NewCombatVisualizationManager(...)
    cm.uiUpdater = NewCombatUIUpdater(...)
    cm.inputRouter = NewCombatInputRouter(...)

    // 4. Initialize action/input handlers
    cm.actionHandler = NewCombatActionHandler(...)
    cm.inputHandler = NewCombatInputHandler(...)

    // 5. Initialize renderers
    cm.movementRenderer = guimodes.NewMovementTileRenderer()
    cm.highlightRenderer = guimodes.NewSquadHighlightRenderer(...)

    return nil
}

func (cm *CombatMode) Enter(fromMode core.UIMode) error {
    isComingFromAnimation := fromMode != nil && fromMode.GetModeName() == "combat_animation"

    if !isComingFromAnimation {
        if err := cm.lifecycleManager.StartBattle(fromMode, cm.combatService.BattleRecorder); err != nil {
            return err
        }
    }

    cm.uiUpdater.RefreshAll()
    return nil
}

func (cm *CombatMode) Exit(toMode core.UIMode) error {
    isToAnimation := toMode != nil && toMode.GetModeName() == "combat_animation"

    if !isToAnimation {
        cm.lifecycleManager.MarkEncounterDefeatedIfVictorious()
        cm.lifecycleManager.FinalizeBattle(cm.combatService.BattleRecorder)
        cm.lifecycleManager.CleanupCombatEntities()
        cm.visualizationManager.ClearAll()
        cm.logManager.Clear()
    }

    return nil
}

func (cm *CombatMode) Update(deltaTime float64) error {
    // UI updates (state-driven)
    currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
    if cm.uiUpdater.UpdateIfFactionChanged(currentFactionID, cm.lastFactionID) {
        cm.lastFactionID = currentFactionID
    }

    battleState := cm.Context.ModeCoordinator.GetBattleMapState()
    if cm.uiUpdater.UpdateIfSquadChanged(battleState.SelectedSquadID, cm.lastSelectedSquad) {
        cm.lastSelectedSquad = battleState.SelectedSquadID
    }

    // Visualization updates
    cm.visualizationManager.Update(
        currentFactionID,
        cm.combatService.TurnManager.GetCurrentRound(),
        *cm.Context.PlayerData.Pos,
        30,
    )

    return nil
}

func (cm *CombatMode) Render(screen *ebiten.Image) {
    playerPos := *cm.Context.PlayerData.Pos
    currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
    battleState := cm.Context.ModeCoordinator.GetBattleMapState()

    cm.highlightRenderer.Render(screen, playerPos, currentFactionID, battleState.SelectedSquadID)

    if battleState.InMoveMode {
        validTiles := cm.getValidMoveTiles()
        if len(validTiles) > 0 {
            cm.movementRenderer.Render(screen, playerPos, validTiles)
        }
    }
}

func (cm *CombatMode) HandleInput(inputState *core.InputState) bool {
    if cm.HandleCommonInput(inputState) {
        return true
    }
    return cm.inputRouter.HandleInput(inputState, cm.handleEndTurn)
}

// Private helpers (minimal orchestration logic)
func (cm *CombatMode) buildAllPanels(ctx *core.UIContext) error
func (cm *CombatMode) getValidMoveTiles() []coords.LogicalPosition
func (cm *CombatMode) handleEndTurn()
func (cm *CombatMode) handleFlee()
```

**Key Characteristics:**
- **Orchestration Only:** Coordinates managers, doesn't implement logic
- **Clear Dependencies:** Each manager has focused responsibility
- **Minimal State:** Only tracks UI state deltas (last faction, last squad)
- **Readable Flow:** Initialize → Enter → Update → Render → HandleInput → Exit

---

## Implementation Strategy

### Recommended Order

1. **Phase 1: UI Builder** (2-3 hours)
   - Lowest risk, highest value
   - Immediately improves readability
   - No complex dependencies

2. **Phase 2: UI Updater** (2-3 hours)
   - Natural follow-up to UI Builder
   - Completes UI separation
   - Establishes component pattern

3. **Phase 5: Visualization Manager** (3-4 hours)
   - Consolidates scattered code
   - Independent from turn/lifecycle logic
   - Can be tested in isolation

4. **Phase 6: Input Router** (1-2 hours)
   - Quick win after visualization
   - Simplifies HandleInput significantly

5. **Phase 3: Turn Orchestrator** (4-5 hours)
   - Most complex extraction
   - Benefits from cleaner context (after UI separation)
   - Requires careful callback design

6. **Phase 4: Lifecycle Manager** (5-6 hours)
   - Critical entity cleanup logic
   - Needs comprehensive testing
   - Save for when other refactorings are stable

7. **Phase 7: Inline Handlers** (30 minutes)
   - Final polish
   - Quick cleanup win

**Total Estimated Time:** 18-24 hours

### Risk Mitigation

1. **Incremental Commits:** Each phase is a separate, tested commit
2. **Test Coverage:** Add unit tests for each new manager before refactoring
3. **Integration Tests:** Maintain existing combat mode integration tests
4. **Regression Testing:** Full combat playthrough after each phase
5. **Rollback Plan:** Each phase is independently reversible

---

## Testing Strategy

### Unit Tests (New)

**File:** `combat_ui_builder_test.go`
- Test each panel build method independently
- Verify widget structure and layout
- Mock UIComponentFactory

**File:** `combat_ui_updater_test.go`
- Test state change detection
- Verify component refresh calls
- Test filter factory logic

**File:** `combat_turn_orchestrator_test.go`
- Test turn advancement logic
- Mock AI controller and animation system
- Verify callback invocations

**File:** `combat_lifecycle_manager_test.go`
- Test encounter setup with mock entities
- Test faction initialization
- Test cleanup (verify all entities removed)
- Test victory condition marking

**File:** `combat_visualization_manager_test.go`
- Test initialization with multiple factions
- Test toggle behavior
- Test status widget updates

**File:** `combat_input_router_test.go`
- Test hotkey routing
- Verify delegation to correct handlers

### Integration Tests (Existing - Should Still Pass)

- Full combat flow (Enter → turns → Exit)
- AI turn execution
- Animation transitions
- Entity lifecycle (spawn → cleanup)

---

## Quantified Benefits

### Line Count Reduction
- **Before:** combatmode.go = 1,059 lines
- **After:** combatmode.go = ~250 lines (76% reduction)
- **New Files:** 6 files × ~150 lines avg = ~920 lines (organized)

### Cyclomatic Complexity Reduction
- **Before:** combatmode.go = ~40-50 (high)
- **After:**
  - combatmode.go = ~8-12 (low - orchestration only)
  - Each manager = ~10-15 (moderate - focused logic)

### Maintainability Improvements
1. **Single Responsibility:** Each file has one clear purpose
2. **Testability:** Managers can be unit tested with mocks
3. **Discoverability:** New developers know exactly where to look
4. **Change Isolation:** UI changes don't touch turn logic, etc.

### Readability Improvements
- **Cognitive Load:** Developer only needs to understand 1-2 files at a time
- **File Size:** All files under 300 lines (comfortable single-screen reading)
- **Clear Boundaries:** "What" (orchestrator) vs "How" (managers) separation

---

## Alignment with ECS Best Practices

### Current Violations
- **CombatMode is a "God Object":** Knows about UI, turn management, entities, visualization
- **Mixed Concerns:** GUI logic and game logic intertwined

### Post-Refactoring Compliance
✅ **Separation of Concerns:** Each manager handles one aspect
✅ **Query-Based Access:** Managers use GUIQueries, not cached entities
✅ **System Functions:** Managers are system-like (operate on ECS data)
✅ **Pure Coordination:** CombatMode orchestrates, doesn't implement

### ECS Pattern Alignment
- `CombatLifecycleManager` → Similar to system initialization/cleanup
- `CombatTurnOrchestrator` → Similar to turn processing system
- `CombatUIUpdater` → Similar to reactive UI system
- `CombatVisualizationManager` → Similar to rendering system

---

## Migration Checklist

### Per-Phase Checklist

- [ ] Create new file with appropriate package and imports
- [ ] Extract structs and methods from combatmode.go
- [ ] Add constructor (New* function)
- [ ] Update combatmode.go to use new manager
- [ ] Write unit tests for new manager
- [ ] Run integration tests
- [ ] Manual playtest (full combat cycle)
- [ ] Commit with descriptive message
- [ ] Update this roadmap with completion status

### Phase Completion Status

- [ ] **Phase 1:** UI Builder (combat_ui_builder.go)
- [ ] **Phase 2:** UI Updater (combat_ui_updater.go)
- [ ] **Phase 3:** Turn Orchestrator (combat_turn_orchestrator.go)
- [ ] **Phase 4:** Lifecycle Manager (combat_lifecycle_manager.go)
- [ ] **Phase 5:** Visualization Manager (combat_visualization_manager.go)
- [ ] **Phase 6:** Input Router (combat_input_router.go)
- [ ] **Phase 7:** Inline Handlers (cleanup)

---

## Potential Pitfalls

### 1. Circular Dependencies
**Risk:** Managers need references to each other
**Mitigation:** Use callback interfaces instead of direct references

### 2. Over-Extraction
**Risk:** Too many small files with trivial logic
**Mitigation:** Each manager must have >100 lines and clear responsibility

### 3. State Synchronization
**Risk:** Managers get out of sync with combat state
**Mitigation:** Single source of truth (combatService), managers query on demand

### 4. Breaking Animation Flow
**Risk:** AI animation sequencing is complex and fragile
**Mitigation:** Comprehensive testing, careful callback design in Phase 3

### 5. Entity Cleanup Bugs
**Risk:** Improper cleanup causes entity leaks or crashes
**Mitigation:** Extensive testing in Phase 4, clear ownership rules

---

## Alternative Approaches Considered

### Alternative 1: Keep Everything in CombatMode
**Reasoning:** "It works, don't fix it"
**Rejected Because:** 1,059 lines is unmaintainable; new features will make it worse

### Alternative 2: Split by UI vs Logic (2 files)
**Reasoning:** Simple split, only 2 new files
**Rejected Because:** Still too coarse-grained; each file would be >500 lines

### Alternative 3: Extract All Methods to Package Functions
**Reasoning:** Avoid struct overhead
**Rejected Because:** Loses cohesion; harder to test; no clear ownership

### Alternative 4: Use Embedded Structs (Mixins)
**Reasoning:** Go-idiomatic composition
**Rejected Because:** Still keeps all code in combatmode.go; doesn't solve file size

**Chosen Approach:** Manager-based separation with clear single responsibilities provides best balance of maintainability, testability, and Go idioms.

---

## Success Criteria

### Code Metrics
- ✅ combatmode.go reduced to <300 lines
- ✅ No file in guicombat exceeds 400 lines
- ✅ Cyclomatic complexity of combatmode.go methods <5
- ✅ Each manager has >90% test coverage

### Functional Requirements
- ✅ All existing combat features work identically
- ✅ No performance regression (measured via profiling)
- ✅ All existing tests pass
- ✅ No new bugs introduced (full regression test)

### Developer Experience
- ✅ New developers can understand combat mode in <30 minutes
- ✅ Bug fixes require editing 1-2 files (not 5+)
- ✅ Adding new features doesn't require combatmode.go changes

---

## Conclusion

This refactoring transforms `combatmode.go` from a **1,059-line monolith** into a **clean orchestrator** supported by 6 focused managers. Each phase is incremental, testable, and provides immediate value.

**Key Success Factors:**
1. **Single Responsibility:** Each file has one clear purpose
2. **Maintainability:** Complexity isolated into manageable units
3. **Testability:** Managers can be unit tested independently
4. **Extensibility:** New features integrate into appropriate manager

**Next Steps:**
1. Review and approve this roadmap
2. Begin Phase 1 (UI Builder) as proof of concept
3. Iterate through phases sequentially
4. Celebrate 73% line reduction in combatmode.go!

---

**Document Version:** 1.0
**Author:** TinkerRogue Refactoring Analysis
**Last Updated:** 2026-01-02
