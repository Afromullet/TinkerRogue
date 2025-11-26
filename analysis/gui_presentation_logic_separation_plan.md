# GUI Presentation/Logic Separation Plan

**Date:** 2025-11-25
**Project:** TinkerRogue
**Purpose:** Separate presentation/drawing logic from gameplay/action logic in GUI system

---

## Executive Summary

The TinkerRogue GUI system shows **good architectural foundations** with clear mode management and component patterns, but suffers from **blurred boundaries** between presentation concerns (rendering, UI state, widget updates) and gameplay logic (combat actions, squad management, game state changes). This plan provides a comprehensive refactoring strategy to establish clean separation of concerns while maintaining ECS best practices.

**Key Finding:** The current architecture is approximately **70% correctly separated**, with the main violations occurring in action handlers that mix UI feedback with game logic execution, and rendering systems embedded within UI modes.

---

## Current Architecture Analysis

### Package Structure

```
gui/
├── core/                      # ✅ Well-designed core abstractions
│   ├── modemanager.go        # Mode lifecycle management
│   ├── uimode.go             # Mode interface definition
│   ├── contextstate.go       # UI state separation (BattleMapState, OverworldState)
│   └── gamemodecoordinator.go # Context switching (Overworld/BattleMap)
│
├── guicomponents/            # ✅ Excellent query/component pattern
│   ├── guicomponents.go      # Reusable UI components
│   └── guiqueries.go         # ECS query service (GUIQueries)
│
├── guimodes/                 # ⚠️ Rendering logic embedded in mode package
│   ├── explorationmode.go    # Exploration UI mode
│   ├── inventorymode.go      # Inventory UI mode
│   ├── infomode.go          # Inspection UI mode
│   └── guirenderers.go      # ⚠️ Rendering systems (should be separate)
│
├── guicombat/               # ⚠️ Mixed concerns - action + presentation
│   ├── combatmode.go        # ✅ Combat UI mode
│   ├── combat_action_handler.go  # ⚠️ Mixes game logic execution + UI feedback
│   ├── combat_input_handler.go   # ✅ Good input delegation
│   ├── combat_ui_factory.go     # ✅ Good widget creation
│   └── combat_log_manager.go    # ✅ Good UI utility
│
├── guisquads/               # ⚠️ Mixed concerns - action + presentation
│   ├── squadbuilder.go      # ⚠️ Mixes squad creation + UI logic
│   ├── squadmanagementmode.go    # ✅ Good mode structure
│   ├── formationeditormode.go    # ⚠️ Mixes formation logic + UI
│   ├── squaddeploymentmode.go    # ✅ Good mode structure
│   ├── squad_builder_grid_manager.go  # ⚠️ Mixes grid state + squad creation
│   └── squad_builder_ui_factory.go    # ✅ Good widget creation
│
├── widgets/                 # ✅ Excellent widget abstraction
│   ├── createwidgets.go     # Widget creation utilities
│   ├── button_factory.go    # Button creation utilities
│   ├── panel_factory.go     # Panel creation utilities
│   ├── layout.go           # Layout configuration
│   └── panelconfig.go      # Panel specifications
│
├── guiresources/           # ✅ Good resource management
│   └── guiresources.go     # Font/color resources
│
├── basemode.go             # ✅ Excellent base mode infrastructure
└── modehelpers.go          # ✅ Good utility functions
```

### Architectural Strengths

1. **Mode Management (core/)** - Clean separation of mode lifecycle and context switching
2. **Query Service (guicomponents/GUIQueries)** - Centralized ECS queries avoid duplication
3. **Widget Factories (widgets/)** - Proper separation of widget creation from business logic
4. **UI Components (guicomponents/)** - Reusable components with clean update patterns
5. **Context State (core/contextstate.go)** - Clear UI state vs. game state separation

### Architectural Violations

#### 1. **Action Handlers Mix UI Feedback with Game Logic**

**File:** `gui/guicombat/combat_action_handler.go`

**Problem:** The `CombatActionHandler` executes game logic (attacks, movement) AND manages UI feedback (logging, mode flags).

```go
// ❌ CURRENT: Mixes game logic execution + UI feedback
func (cah *CombatActionHandler) executeAttack() {
    // UI state manipulation
    selectedSquad := cah.battleMapState.SelectedSquadID
    selectedTarget := cah.battleMapState.SelectedTargetID

    // Game logic execution
    combatSys := combat.NewCombatActionSystem(cah.entityManager)
    err := combatSys.ExecuteAttackAction(selectedSquad, selectedTarget)

    // UI feedback
    if err != nil {
        cah.addLog(fmt.Sprintf("Attack failed: %v", err))
    } else {
        cah.addLog(fmt.Sprintf("%s attacked %s!", attackerName, targetName))
    }

    // UI state reset
    cah.battleMapState.InAttackMode = false
}
```

**Why This is Wrong:**
- Action handler **executes game logic** (creates combat system, calls ExecuteAttackAction)
- Action handler **manages UI state** (battleMapState flags)
- Action handler **provides UI feedback** (combat log)
- Violates Single Responsibility Principle
- Makes testing difficult (must mock UI components to test game logic)

#### 2. **Rendering Systems Embedded in UI Mode Package**

**File:** `gui/guimodes/guirenderers.go`

**Problem:** Rendering systems are in the `guimodes` package instead of a dedicated rendering package.

```go
// ⚠️ CURRENT: Rendering logic in UI mode package
type MovementTileRenderer struct {
    fillColor color.Color
}

func (mtr *MovementTileRenderer) Render(screen *ebiten.Image, centerPos coords.LogicalPosition, validTiles []coords.LogicalPosition) {
    vr := NewViewportRenderer(screen, centerPos)
    for _, pos := range validTiles {
        vr.DrawTileOverlay(screen, pos, mtr.fillColor)
    }
}
```

**Why This is Wrong:**
- Rendering logic is presentation concern, not mode logic
- Renderers are reusable across modes but tied to `guimodes` package
- Violates package cohesion (modes should coordinate, not render)

#### 3. **Squad Builder Mixes Squad Creation with UI Logic**

**File:** `gui/guisquads/squadbuilder.go`

**Problem:** Squad builder mode directly calls ECS squad creation functions.

```go
// ❌ CURRENT: Mode directly creates squads
func (sbm *SquadBuilderMode) onCreateSquad() {
    // UI validation
    if sbm.currentSquadID == 0 {
        fmt.Println("No squad to create - grid is empty")
        return
    }

    // Game logic execution
    unitIDs := squads.GetUnitIDsInSquad(sbm.currentSquadID, sbm.Context.ECSManager)
    leaderID := sbm.gridManager.GetLeader()

    // ECS mutation
    leaderEntity := common.FindEntityByIDWithTag(sbm.Context.ECSManager, leaderID, squads.SquadMemberTag)
    leaderEntity.AddComponent(squads.LeaderComponent, &squads.LeaderData{})

    // UI feedback
    fmt.Printf("Squad created: %s with %d units\n", sbm.currentSquadName, len(unitIDs))
}
```

**Why This is Wrong:**
- Mode directly mutates ECS state (adds LeaderComponent)
- Squad creation logic scattered between mode and grid manager
- Difficult to test squad creation without UI infrastructure

#### 4. **Grid Manager Mixes Grid State with Squad Creation**

**File:** `gui/guisquads/squad_builder_grid_manager.go`

**Problem:** Grid manager manages both UI grid state AND executes squad creation logic.

```go
// ⚠️ Grid manager should only manage UI grid state, not create squads
type GridEditorManager struct {
    gridCells  [3][3]*GridCellButton
    squadID    ecs.EntityID  // ✅ UI state
    leaderID   ecs.EntityID  // ✅ UI state
    manager    *common.EntityManager  // ❌ Shouldn't execute game logic
}

func (gem *GridEditorManager) PlaceRosterUnitInCell(row, col int, template *squads.UnitTemplate, squadID ecs.EntityID, unitEntityID ecs.EntityID) error {
    // ❌ Executes game logic (squad membership)
    err := squads.AddUnitToSquad(unitEntityID, squadID, gem.manager)
    // ✅ Updates UI grid state
    gem.gridCells[row][col].unitID = unitEntityID
}
```

#### 5. **Input Handlers Sometimes Execute Game Logic**

**File:** `gui/guicombat/combat_input_handler.go`

**Current State:** Input handler is **mostly good** - it delegates to action handler. But the action handler then mixes concerns.

```go
// ✅ GOOD: Input handler delegates to action handler
func (cih *CombatInputHandler) HandleInput(inputState *core.InputState) bool {
    if inputState.KeysJustPressed[ebiten.KeyA] {
        cih.actionHandler.ToggleAttackMode()  // ✅ Delegates
        return true
    }
}
```

---

## Proposed Refactoring Strategy

### Phase 1: Establish Clear Package Boundaries

#### 1.1 Create `gui/rendering/` Package

**Purpose:** Centralize all rendering logic separate from UI modes.

**New Package Structure:**
```
gui/rendering/
├── viewport_renderer.go       # Viewport-based rendering utilities
├── movement_renderer.go        # Movement tile rendering
├── squad_highlight_renderer.go # Squad highlight rendering
├── ability_effect_renderer.go  # Ability visual effects
└── renderer_interface.go       # Common renderer interface
```

**Migration:**
- Move `ViewportRenderer`, `MovementTileRenderer`, `SquadHighlightRenderer` from `gui/guimodes/guirenderers.go` to `gui/rendering/`
- Modes import rendering package and use renderers as services

#### 1.2 Create `gui/actions/` Package

**Purpose:** Define UI action interfaces that modes use to trigger gameplay changes.

**New Package Structure:**
```
gui/actions/
├── combat_actions.go     # Combat action interface
├── squad_actions.go      # Squad management action interface
├── movement_actions.go   # Movement action interface
└── action_result.go      # Standardized action result types
```

**Pattern:**
```go
// Action interface (presentation layer)
type CombatActions interface {
    ExecuteAttack(attackerID, targetID ecs.EntityID) ActionResult
    ToggleMoveMode(squadID ecs.EntityID) ActionResult
    EndTurn() ActionResult
}

// Action result (carries both game outcome and UI feedback)
type ActionResult struct {
    Success      bool
    ErrorMessage string
    LogMessages  []string
    StateChanges UIStateChanges  // What UI should update
}

type UIStateChanges struct {
    ClearSelection    bool
    ExitMoveMode      bool
    RefreshSquadList  bool
    UpdateTurnDisplay bool
}
```

#### 1.3 Create `game_logic/` Package (Outside GUI)

**Purpose:** Pure game logic systems that execute gameplay actions.

**New Package Structure:**
```
game_logic/
├── combat_controller.go     # Combat gameplay controller
├── squad_controller.go      # Squad management controller
├── movement_controller.go   # Movement gameplay controller
└── action_validator.go      # Validation logic
```

**Pattern:**
```go
// Pure game logic (no UI dependencies)
type CombatController struct {
    manager        *common.EntityManager
    combatSystem   *combat.CombatActionSystem
    turnManager    *combat.TurnManager
}

func (cc *CombatController) ExecuteAttack(attackerID, targetID ecs.EntityID) error {
    // 1. Validate action
    if err := cc.validateAttack(attackerID, targetID); err != nil {
        return err
    }

    // 2. Execute game logic
    if err := cc.combatSystem.ExecuteAttackAction(attackerID, targetID); err != nil {
        return err
    }

    // 3. Update game state
    combat.MarkSquadAsActed(attackerID, cc.manager)

    return nil
}
```

### Phase 2: Refactor Action Handlers

#### 2.1 Convert CombatActionHandler to CombatActionBridge

**Before:** `gui/guicombat/combat_action_handler.go` (mixed concerns)

**After:** `gui/guicombat/combat_action_bridge.go` (pure presentation bridge)

```go
// ✅ NEW APPROACH: Action bridge delegates to game logic controller
type CombatActionBridge struct {
    // Game logic controller (no UI dependencies)
    combatController *game_logic.CombatController

    // UI state (presentation only)
    battleMapState   *core.BattleMapState
    logManager       *CombatLogManager
    queries          *guicomponents.GUIQueries
}

func (cab *CombatActionBridge) ExecuteAttack(attackerID, targetID ecs.EntityID) ActionResult {
    // 1. Delegate to game logic controller
    err := cab.combatController.ExecuteAttack(attackerID, targetID)

    // 2. Convert game result to UI action result
    result := ActionResult{Success: err == nil}

    if err != nil {
        result.ErrorMessage = err.Error()
        result.LogMessages = []string{fmt.Sprintf("Attack failed: %v", err)}
    } else {
        attackerName := cab.queries.GetSquadName(attackerID)
        targetName := cab.queries.GetSquadName(targetID)
        result.LogMessages = []string{
            fmt.Sprintf("%s attacked %s!", attackerName, targetName),
        }

        // Check if target destroyed
        if squads.IsSquadDestroyed(targetID, cab.combatController.EntityManager()) {
            result.LogMessages = append(result.LogMessages,
                fmt.Sprintf("%s was destroyed!", targetName))
        }
    }

    // 3. Specify UI state changes
    result.StateChanges.ClearSelection = true
    result.StateChanges.ExitAttackMode = true

    return result
}
```

**Key Benefits:**
- Game logic in `game_logic.CombatController` (testable without UI)
- UI feedback in `CombatActionBridge` (delegates to controller)
- Clear separation of concerns
- Action results are standardized and predictable

#### 2.2 Refactor Squad Builder

**Before:** `squadbuilder.go` directly mutates ECS

**After:** Delegate to `game_logic.SquadController`

```go
// ✅ NEW: Squad controller handles all squad creation logic
type SquadController struct {
    manager *common.EntityManager
}

func (sc *SquadController) CreateSquad(squadID ecs.EntityID, leaderID ecs.EntityID) error {
    // 1. Validate squad
    unitIDs := squads.GetUnitIDsInSquad(squadID, sc.manager)
    if len(unitIDs) == 0 {
        return fmt.Errorf("cannot create empty squad")
    }

    if leaderID == 0 {
        return fmt.Errorf("squad must have a leader")
    }

    // 2. Assign leader component
    leaderEntity := common.FindEntityByIDWithTag(sc.manager, leaderID, squads.SquadMemberTag)
    if leaderEntity == nil {
        return fmt.Errorf("leader entity not found")
    }
    leaderEntity.AddComponent(squads.LeaderComponent, &squads.LeaderData{})

    return nil
}

// ✅ NEW: Squad builder mode delegates to controller
func (sbm *SquadBuilderMode) onCreateSquad() {
    result := sbm.squadController.CreateSquad(sbm.currentSquadID, sbm.gridManager.GetLeader())

    if result != nil {
        fmt.Printf("Squad creation failed: %v\n", result)
        return
    }

    // UI feedback
    unitCount := len(squads.GetUnitIDsInSquad(sbm.currentSquadID, sbm.Context.ECSManager))
    fmt.Printf("Squad created: %s with %d units\n", sbm.currentSquadName, unitCount)

    // UI state update
    sbm.onClearGrid()
}
```

### Phase 3: Implement Action Result Pattern

#### 3.1 Define ActionResult Types

```go
// gui/actions/action_result.go
package actions

// ActionResult represents the outcome of a UI-triggered action
type ActionResult struct {
    Success      bool
    ErrorMessage string
    InfoMessages []string
    LogMessages  []string
    StateChanges UIStateChanges
}

// UIStateChanges describes what UI components should update
type UIStateChanges struct {
    // Selection state
    ClearSelection    bool
    SelectedSquadID   ecs.EntityID
    SelectedTargetID  ecs.EntityID

    // Mode flags
    ExitAttackMode    bool
    ExitMoveMode      bool

    // Display updates
    RefreshSquadList  bool
    RefreshSquadDetail bool
    UpdateTurnDisplay bool
    RefreshFactionInfo bool

    // Movement state
    ClearMovementTiles bool
}

// Apply applies the state changes to a BattleMapState
func (sc *UIStateChanges) Apply(state *core.BattleMapState) {
    if sc.ClearSelection {
        state.SelectedSquadID = 0
        state.SelectedTargetID = 0
    }
    if sc.SelectedSquadID != 0 {
        state.SelectedSquadID = sc.SelectedSquadID
    }
    if sc.ExitAttackMode {
        state.InAttackMode = false
    }
    if sc.ExitMoveMode {
        state.InMoveMode = false
    }
    if sc.ClearMovementTiles {
        state.ValidMoveTiles = []coords.LogicalPosition{}
    }
}
```

#### 3.2 Update Modes to Use ActionResult

```go
// CombatMode uses action result pattern
func (cm *CombatMode) handleAttackClick() {
    result := cm.actionBridge.ToggleAttackMode()

    // Apply state changes
    result.StateChanges.Apply(cm.Context.ModeCoordinator.GetBattleMapState())

    // Apply UI feedback
    for _, msg := range result.LogMessages {
        cm.logManager.UpdateTextArea(cm.combatLogArea, msg)
    }

    // Apply display updates
    if result.StateChanges.RefreshSquadList {
        cm.squadListComponent.Refresh()
    }
}
```

### Phase 4: Separate Rendering from Modes

#### 4.1 Create RenderingCoordinator

```go
// gui/rendering/coordinator.go
type RenderingCoordinator struct {
    movementRenderer  *MovementTileRenderer
    highlightRenderer *SquadHighlightRenderer
    effectRenderer    *AbilityEffectRenderer
}

func NewRenderingCoordinator(queries *guicomponents.GUIQueries) *RenderingCoordinator {
    return &RenderingCoordinator{
        movementRenderer:  NewMovementTileRenderer(),
        highlightRenderer: NewSquadHighlightRenderer(queries),
        effectRenderer:    NewAbilityEffectRenderer(),
    }
}

// RenderCombatScene renders all combat visual overlays
func (rc *RenderingCoordinator) RenderCombatScene(screen *ebiten.Image, scene *CombatSceneData) {
    // Render squad highlights
    rc.highlightRenderer.Render(
        screen,
        scene.CenterPos,
        scene.CurrentFactionID,
        scene.SelectedSquadID,
    )

    // Render movement tiles (if in move mode)
    if scene.InMoveMode && len(scene.ValidMoveTiles) > 0 {
        rc.movementRenderer.Render(screen, scene.CenterPos, scene.ValidMoveTiles)
    }

    // Render ability effects
    rc.effectRenderer.Render(screen, scene.CenterPos, scene.ActiveEffects)
}
```

#### 4.2 Update CombatMode to Use RenderingCoordinator

```go
// ✅ NEW: CombatMode delegates all rendering to coordinator
type CombatMode struct {
    gui.BaseMode

    // Rendering service
    renderingCoordinator *rendering.RenderingCoordinator

    // ... other fields
}

func (cm *CombatMode) Render(screen *ebiten.Image) {
    // Build scene data from UI state
    sceneData := &rendering.CombatSceneData{
        CenterPos:         *cm.Context.PlayerData.Pos,
        CurrentFactionID:  cm.turnManager.GetCurrentFaction(),
        SelectedSquadID:   cm.Context.ModeCoordinator.GetBattleMapState().SelectedSquadID,
        InMoveMode:        cm.Context.ModeCoordinator.GetBattleMapState().InMoveMode,
        ValidMoveTiles:    cm.Context.ModeCoordinator.GetBattleMapState().ValidMoveTiles,
    }

    // Delegate to rendering coordinator
    cm.renderingCoordinator.RenderCombatScene(screen, sceneData)
}
```

---

## Refactoring Roadmap

### Sprint 1: Foundation (Estimated 8-12 hours)

**Goal:** Establish new package structure and move rendering logic.

1. **Create new packages** (1 hour)
   - `gui/rendering/` - Rendering systems
   - `gui/actions/` - Action interfaces and results
   - `game_logic/` - Pure game logic controllers

2. **Move rendering systems** (2-3 hours)
   - Move `ViewportRenderer`, `MovementTileRenderer`, `SquadHighlightRenderer` to `gui/rendering/`
   - Create `RenderingCoordinator` in `gui/rendering/coordinator.go`
   - Update import statements in modes

3. **Define action result types** (2 hours)
   - Create `ActionResult` struct in `gui/actions/action_result.go`
   - Define `UIStateChanges` struct
   - Implement `Apply()` method

4. **Create game logic controllers** (3-4 hours)
   - Create `game_logic.CombatController`
   - Create `game_logic.SquadController`
   - Create `game_logic.MovementController`
   - Move validation logic to controllers

5. **Testing** (2 hours)
   - Test rendering still works after package move
   - Test controller creation and basic methods

### Sprint 2: Combat System (Estimated 10-14 hours)

**Goal:** Refactor combat action handler to use new pattern.

1. **Create CombatActionBridge** (3-4 hours)
   - Rename `combat_action_handler.go` to `combat_action_bridge.go`
   - Refactor to delegate to `CombatController`
   - Return `ActionResult` from all methods

2. **Update CombatMode** (3-4 hours)
   - Use `RenderingCoordinator` for rendering
   - Apply `ActionResult` state changes
   - Update button handlers to use action results

3. **Update CombatInputHandler** (2 hours)
   - Ensure input handler only delegates to action bridge
   - Apply action results from input handler

4. **Testing** (3-4 hours)
   - Test combat actions still work
   - Test UI state updates correctly
   - Test combat log messages appear
   - Test rendering overlays work

### Sprint 3: Squad Management (Estimated 10-12 hours)

**Goal:** Refactor squad builder and formation editor.

1. **Create SquadActionBridge** (3-4 hours)
   - Create bridge in `gui/guisquads/squad_action_bridge.go`
   - Delegate to `SquadController`
   - Return `ActionResult` for squad creation

2. **Refactor SquadBuilderMode** (4-5 hours)
   - Update `onCreateSquad()` to use action bridge
   - Apply action results to UI state
   - Update grid manager to only manage UI grid state

3. **Refactor FormationEditorMode** (2-3 hours)
   - Use action bridge for formation changes
   - Apply action results

4. **Testing** (2 hours)
   - Test squad creation still works
   - Test formation editor still works

### Sprint 4: Polish and Documentation (Estimated 4-6 hours)

**Goal:** Clean up, document, and verify all modes follow pattern.

1. **Update ExplorationMode** (1-2 hours)
   - Ensure rendering uses `RenderingCoordinator`
   - Verify no direct game logic execution

2. **Update InventoryMode** (1 hour)
   - Verify follows action result pattern
   - No direct ECS mutations

3. **Documentation** (2-3 hours)
   - Update CLAUDE.md with new patterns
   - Document action result pattern
   - Document rendering coordinator pattern
   - Add examples to docs/gui_best_practices.md

---

## File-by-File Recommendations

### gui/guicombat/combat_action_handler.go → combat_action_bridge.go

**Current State:** Mixes game logic execution with UI feedback
**Action Required:** Refactor to delegation pattern

**Changes:**
1. Rename to `combat_action_bridge.go`
2. Remove direct `combat.NewCombatActionSystem()` calls
3. Add `game_logic.CombatController` field
4. Return `ActionResult` from all methods
5. Remove direct `battleMapState` mutations (return state changes in result)

**Before (Lines 167-204):**
```go
func (cah *CombatActionHandler) executeAttack() {
    selectedSquad := cah.battleMapState.SelectedSquadID
    selectedTarget := cah.battleMapState.SelectedTargetID

    combatSys := combat.NewCombatActionSystem(cah.entityManager)
    err := combatSys.ExecuteAttackAction(selectedSquad, selectedTarget)

    if err != nil {
        cah.addLog(fmt.Sprintf("Attack failed: %v", err))
    }

    cah.battleMapState.InAttackMode = false
}
```

**After:**
```go
func (cab *CombatActionBridge) ExecuteAttack(attackerID, targetID ecs.EntityID) ActionResult {
    err := cab.combatController.ExecuteAttack(attackerID, targetID)

    result := ActionResult{Success: err == nil}
    if err != nil {
        result.LogMessages = []string{fmt.Sprintf("Attack failed: %v", err)}
    } else {
        attackerName := cab.queries.GetSquadName(attackerID)
        targetName := cab.queries.GetSquadName(targetID)
        result.LogMessages = []string{fmt.Sprintf("%s attacked %s!", attackerName, targetName)}
    }

    result.StateChanges.ExitAttackMode = true
    return result
}
```

### gui/guicombat/combatmode.go

**Current State:** Good mode structure, but embeds rendering systems
**Action Required:** Use `RenderingCoordinator`

**Changes:**
1. Replace `movementRenderer` and `highlightRenderer` fields with `renderingCoordinator`
2. Update `Render()` method to build scene data and delegate to coordinator
3. Update button handlers to apply `ActionResult` state changes

**Before (Lines 312-328):**
```go
func (cm *CombatMode) Render(screen *ebiten.Image) {
    playerPos := *cm.Context.PlayerData.Pos
    currentFactionID := cm.turnManager.GetCurrentFaction()
    battleState := cm.Context.ModeCoordinator.GetBattleMapState()
    selectedSquad := battleState.SelectedSquadID

    cm.highlightRenderer.Render(screen, playerPos, currentFactionID, selectedSquad)

    if battleState.InMoveMode {
        validTiles := battleState.ValidMoveTiles
        if len(validTiles) > 0 {
            cm.movementRenderer.Render(screen, playerPos, validTiles)
        }
    }
}
```

**After:**
```go
func (cm *CombatMode) Render(screen *ebiten.Image) {
    sceneData := &rendering.CombatSceneData{
        CenterPos:        *cm.Context.PlayerData.Pos,
        CurrentFactionID: cm.turnManager.GetCurrentFaction(),
        SelectedSquadID:  cm.Context.ModeCoordinator.GetBattleMapState().SelectedSquadID,
        InMoveMode:       cm.Context.ModeCoordinator.GetBattleMapState().InMoveMode,
        ValidMoveTiles:   cm.Context.ModeCoordinator.GetBattleMapState().ValidMoveTiles,
    }

    cm.renderingCoordinator.RenderCombatScene(screen, sceneData)
}
```

### gui/guimodes/guirenderers.go → gui/rendering/

**Current State:** Rendering logic in wrong package
**Action Required:** Move entire file to new package

**Changes:**
1. Move `ViewportRenderer`, `MovementTileRenderer`, `SquadHighlightRenderer` to `gui/rendering/`
2. Create `gui/rendering/coordinator.go` with `RenderingCoordinator`
3. Update imports in `guicombat/combatmode.go` and other modes

### gui/guisquads/squadbuilder.go

**Current State:** Directly mutates ECS to create squads
**Action Required:** Delegate to `game_logic.SquadController`

**Changes:**
1. Add `squadController *game_logic.SquadController` field
2. Update `onCreateSquad()` to call `squadController.CreateSquad()`
3. Apply action result to UI state
4. Remove direct ECS mutations

**Before (Lines 330-379):**
```go
func (sbm *SquadBuilderMode) onCreateSquad() {
    // ... validation ...

    // Direct ECS mutation
    leaderEntity := common.FindEntityByIDWithTag(...)
    leaderEntity.AddComponent(squads.LeaderComponent, &squads.LeaderData{})

    // ... more logic ...
}
```

**After:**
```go
func (sbm *SquadBuilderMode) onCreateSquad() {
    result := sbm.squadController.CreateSquad(sbm.currentSquadID, sbm.gridManager.GetLeader())

    if !result.Success {
        // Display error
        for _, msg := range result.LogMessages {
            fmt.Println(msg)
        }
        return
    }

    // UI feedback
    for _, msg := range result.InfoMessages {
        fmt.Println(msg)
    }

    // UI state update
    sbm.onClearGrid()
}
```

### gui/guisquads/squad_builder_grid_manager.go

**Current State:** Mixes UI grid state with squad creation logic
**Action Required:** Keep only UI grid state management

**Changes:**
1. Remove `PlaceRosterUnitInCell()` squad membership logic
2. Grid manager only tracks `unitID` and `rosterEntryID` in cells
3. Mode delegates squad membership changes to `SquadController`

**Before:**
```go
func (gem *GridEditorManager) PlaceRosterUnitInCell(...) error {
    // ❌ Executes game logic
    err := squads.AddUnitToSquad(unitEntityID, squadID, gem.manager)

    // ✅ Updates UI state
    gem.gridCells[row][col].unitID = unitEntityID
}
```

**After:**
```go
func (gem *GridEditorManager) SetCellUnit(row, col int, unitID ecs.EntityID, rosterEntryID ecs.EntityID) {
    // ✅ Only updates UI grid state
    gem.gridCells[row][col].unitID = unitID
    gem.gridCells[row][col].rosterEntryID = rosterEntryID
    gem.RefreshGridDisplay()
}

// Mode handles game logic separately:
func (sbm *SquadBuilderMode) placeRosterUnitInCell(...) {
    // 1. Game logic (delegate to controller)
    result := sbm.squadController.AddUnitToSquad(unitEntityID, squadID)

    // 2. UI state update (grid manager)
    if result.Success {
        sbm.gridManager.SetCellUnit(row, col, unitEntityID, rosterEntryID)
    }
}
```

---

## Testing Strategy

### Unit Tests

**Game Logic Controllers** (No UI dependencies):
```go
func TestCombatController_ExecuteAttack(t *testing.T) {
    manager := common.CreateTestEntityManager()
    controller := game_logic.NewCombatController(manager)

    attackerID := createTestSquad(manager, "Attacker")
    targetID := createTestSquad(manager, "Target")

    err := controller.ExecuteAttack(attackerID, targetID)
    assert.NoError(t, err)

    // Verify game state changed
    actionState := combat.FindActionStateBySquadID(attackerID, manager)
    assert.True(t, actionState.HasActed)
}
```

**Action Bridges** (Test delegation and result conversion):
```go
func TestCombatActionBridge_ExecuteAttack(t *testing.T) {
    manager := common.CreateTestEntityManager()
    controller := game_logic.NewCombatController(manager)
    queries := guicomponents.NewGUIQueries(manager)

    bridge := NewCombatActionBridge(controller, queries, ...)

    attackerID := createTestSquad(manager, "Attacker")
    targetID := createTestSquad(manager, "Target")

    result := bridge.ExecuteAttack(attackerID, targetID)

    assert.True(t, result.Success)
    assert.NotEmpty(t, result.LogMessages)
    assert.True(t, result.StateChanges.ExitAttackMode)
}
```

### Integration Tests

**Mode Behavior** (Test full UI flow):
```go
func TestCombatMode_HandleAttackClick(t *testing.T) {
    ctx := createTestUIContext()
    mode := NewCombatMode(nil)
    mode.Initialize(ctx)

    // Simulate button click
    mode.handleAttackClick()

    // Verify UI state changed
    battleState := ctx.ModeCoordinator.GetBattleMapState()
    assert.True(t, battleState.InAttackMode)
}
```

---

## Architectural Patterns Summary

### Pattern 1: Game Logic Controller

**Purpose:** Pure game logic with no UI dependencies

**Structure:**
```go
type CombatController struct {
    manager      *common.EntityManager
    combatSystem *combat.CombatActionSystem
    turnManager  *combat.TurnManager
}

func (cc *CombatController) ExecuteAttack(attackerID, targetID ecs.EntityID) error {
    // Pure game logic
    return cc.combatSystem.ExecuteAttackAction(attackerID, targetID)
}
```

**Benefits:**
- Testable without UI infrastructure
- Reusable across different UI modes (CLI, GUI, networked)
- Clear separation of concerns

### Pattern 2: Action Bridge

**Purpose:** Translates UI actions to game logic calls and converts results to UI feedback

**Structure:**
```go
type CombatActionBridge struct {
    controller     *game_logic.CombatController
    queries        *guicomponents.GUIQueries
    // NO game state fields (uses controller)
}

func (cab *CombatActionBridge) ExecuteAttack(attackerID, targetID ecs.EntityID) ActionResult {
    // 1. Delegate to controller
    err := cab.controller.ExecuteAttack(attackerID, targetID)

    // 2. Convert to UI result
    result := ActionResult{Success: err == nil}
    result.LogMessages = []string{/* ... */}
    result.StateChanges.ExitAttackMode = true

    return result
}
```

**Benefits:**
- Clear boundary between game logic and UI feedback
- Standardized action results
- Easy to test bridge logic separately

### Pattern 3: Action Result

**Purpose:** Standardized way to communicate game outcomes and required UI updates

**Structure:**
```go
type ActionResult struct {
    Success      bool
    ErrorMessage string
    LogMessages  []string
    StateChanges UIStateChanges
}

type UIStateChanges struct {
    ExitAttackMode    bool
    RefreshSquadList  bool
    // ... more flags
}
```

**Benefits:**
- Explicit about what UI should update
- Decouples game logic from UI state management
- Makes testing predictable

### Pattern 4: Rendering Coordinator

**Purpose:** Centralize all rendering logic separate from UI modes

**Structure:**
```go
type RenderingCoordinator struct {
    movementRenderer  *MovementTileRenderer
    highlightRenderer *SquadHighlightRenderer
}

func (rc *RenderingCoordinator) RenderCombatScene(screen *ebiten.Image, scene *CombatSceneData) {
    // Render all visual overlays
}
```

**Benefits:**
- Rendering logic separate from mode coordination
- Reusable across modes
- Easy to add new visual effects

---

## Success Metrics

### Code Metrics

1. **Package Dependency Graph**
   - **Before:** `gui/guicombat` → `combat` (correct) AND `gui/guicombat` → ECS mutation (incorrect)
   - **After:** `gui/guicombat` → `game_logic` → `combat` (correct)

2. **Testability**
   - **Before:** Cannot test combat logic without UI infrastructure
   - **After:** `CombatController` testable with only `EntityManager`

3. **Lines of Code in UI Modes**
   - **Before:** `combatmode.go` ~355 lines (includes rendering + action handling)
   - **After:** `combatmode.go` ~250 lines (coordination only)

### Architectural Compliance

1. **Action Handlers**
   - ✅ All action handlers return `ActionResult`
   - ✅ No direct ECS mutations in action handlers
   - ✅ Game logic in `game_logic/` controllers

2. **Rendering**
   - ✅ All rendering logic in `gui/rendering/` package
   - ✅ Modes use `RenderingCoordinator`
   - ✅ No rendering logic in mode files

3. **UI Modes**
   - ✅ Modes coordinate components and handle input
   - ✅ Modes apply `ActionResult` state changes
   - ✅ No game logic execution in modes

---

## Code Examples

### Example 1: Combat Attack Flow

**Before (Mixed Concerns):**
```go
// gui/guicombat/combat_action_handler.go
func (cah *CombatActionHandler) executeAttack() {
    // UI state
    selectedSquad := cah.battleMapState.SelectedSquadID

    // Game logic execution
    combatSys := combat.NewCombatActionSystem(cah.entityManager)
    err := combatSys.ExecuteAttackAction(selectedSquad, selectedTarget)

    // UI feedback
    cah.addLog(fmt.Sprintf("Attack result: %v", err))

    // UI state mutation
    cah.battleMapState.InAttackMode = false
}
```

**After (Separated Concerns):**

**Step 1: Game Logic Controller**
```go
// game_logic/combat_controller.go
type CombatController struct {
    manager      *common.EntityManager
    combatSystem *combat.CombatActionSystem
}

func (cc *CombatController) ExecuteAttack(attackerID, targetID ecs.EntityID) error {
    return cc.combatSystem.ExecuteAttackAction(attackerID, targetID)
}
```

**Step 2: Action Bridge**
```go
// gui/guicombat/combat_action_bridge.go
type CombatActionBridge struct {
    controller *game_logic.CombatController
    queries    *guicomponents.GUIQueries
}

func (cab *CombatActionBridge) ExecuteAttack(attackerID, targetID ecs.EntityID) ActionResult {
    err := cab.controller.ExecuteAttack(attackerID, targetID)

    result := ActionResult{Success: err == nil}
    if err != nil {
        result.LogMessages = []string{fmt.Sprintf("Attack failed: %v", err)}
    } else {
        attackerName := cab.queries.GetSquadName(attackerID)
        targetName := cab.queries.GetSquadName(targetID)
        result.LogMessages = []string{fmt.Sprintf("%s attacked %s!", attackerName, targetName)}
    }

    result.StateChanges.ExitAttackMode = true
    return result
}
```

**Step 3: Mode Application**
```go
// gui/guicombat/combatmode.go
func (cm *CombatMode) handleAttackClick() {
    battleState := cm.Context.ModeCoordinator.GetBattleMapState()

    // Execute action through bridge
    result := cm.actionBridge.ExecuteAttack(
        battleState.SelectedSquadID,
        battleState.SelectedTargetID,
    )

    // Apply state changes
    result.StateChanges.Apply(battleState)

    // Apply UI feedback
    for _, msg := range result.LogMessages {
        cm.logManager.UpdateTextArea(cm.combatLogArea, msg)
    }

    // Update displays if needed
    if result.StateChanges.RefreshSquadList {
        cm.squadListComponent.Refresh()
    }
}
```

### Example 2: Squad Creation Flow

**Before (Mixed Concerns):**
```go
// gui/guisquads/squadbuilder.go
func (sbm *SquadBuilderMode) onCreateSquad() {
    // UI validation
    unitIDs := squads.GetUnitIDsInSquad(sbm.currentSquadID, sbm.Context.ECSManager)
    if len(unitIDs) == 0 {
        fmt.Println("Cannot create empty squad")
        return
    }

    // Direct ECS mutation
    leaderEntity := common.FindEntityByIDWithTag(sbm.Context.ECSManager, leaderID, squads.SquadMemberTag)
    leaderEntity.AddComponent(squads.LeaderComponent, &squads.LeaderData{})

    // UI feedback
    fmt.Printf("Squad created: %s\n", sbm.currentSquadName)

    // UI state
    sbm.onClearGrid()
}
```

**After (Separated Concerns):**

**Step 1: Game Logic Controller**
```go
// game_logic/squad_controller.go
type SquadController struct {
    manager *common.EntityManager
}

func (sc *SquadController) CreateSquad(squadID ecs.EntityID, leaderID ecs.EntityID) error {
    // Validation
    unitIDs := squads.GetUnitIDsInSquad(squadID, sc.manager)
    if len(unitIDs) == 0 {
        return fmt.Errorf("cannot create empty squad")
    }
    if leaderID == 0 {
        return fmt.Errorf("squad must have a leader")
    }

    // Game logic
    leaderEntity := common.FindEntityByIDWithTag(sc.manager, leaderID, squads.SquadMemberTag)
    if leaderEntity == nil {
        return fmt.Errorf("leader entity not found")
    }
    leaderEntity.AddComponent(squads.LeaderComponent, &squads.LeaderData{})

    return nil
}
```

**Step 2: Action Bridge**
```go
// gui/guisquads/squad_action_bridge.go
type SquadActionBridge struct {
    controller *game_logic.SquadController
    queries    *guicomponents.GUIQueries
}

func (sab *SquadActionBridge) CreateSquad(squadID ecs.EntityID, leaderID ecs.EntityID) ActionResult {
    err := sab.controller.CreateSquad(squadID, leaderID)

    result := ActionResult{Success: err == nil}
    if err != nil {
        result.ErrorMessage = err.Error()
        result.LogMessages = []string{fmt.Sprintf("Squad creation failed: %v", err)}
    } else {
        squadName := sab.queries.GetSquadName(squadID)
        unitCount := len(squads.GetUnitIDsInSquad(squadID, sab.controller.EntityManager()))
        result.InfoMessages = []string{
            fmt.Sprintf("Squad created: %s with %d units", squadName, unitCount),
        }
    }

    result.StateChanges.ClearGridState = true
    return result
}
```

**Step 3: Mode Application**
```go
// gui/guisquads/squadbuilder.go
func (sbm *SquadBuilderMode) onCreateSquad() {
    // Execute action through bridge
    result := sbm.actionBridge.CreateSquad(sbm.currentSquadID, sbm.gridManager.GetLeader())

    // Apply UI feedback
    if !result.Success {
        fmt.Println(result.ErrorMessage)
        return
    }

    for _, msg := range result.InfoMessages {
        fmt.Println(msg)
    }

    // Apply state changes
    if result.StateChanges.ClearGridState {
        sbm.onClearGrid()
    }
}
```

---

## Migration Checklist

### Phase 1: Foundation ✅

- [ ] Create `gui/rendering/` package
- [ ] Create `gui/actions/` package
- [ ] Create `game_logic/` package
- [ ] Move rendering systems to `gui/rendering/`
- [ ] Create `RenderingCoordinator`
- [ ] Define `ActionResult` types
- [ ] Create `CombatController` in `game_logic/`
- [ ] Create `SquadController` in `game_logic/`
- [ ] Test rendering still works

### Phase 2: Combat System ✅

- [ ] Refactor `combat_action_handler.go` → `combat_action_bridge.go`
- [ ] Update all methods to return `ActionResult`
- [ ] Update `CombatMode.Render()` to use `RenderingCoordinator`
- [ ] Update button handlers to apply `ActionResult`
- [ ] Update `CombatInputHandler` to use action results
- [ ] Test combat actions work correctly
- [ ] Test UI state updates correctly
- [ ] Test rendering works correctly

### Phase 3: Squad Management ✅

- [ ] Create `SquadActionBridge`
- [ ] Update `SquadBuilderMode.onCreateSquad()` to use bridge
- [ ] Update grid manager to only manage UI state
- [ ] Update `FormationEditorMode` to use bridge
- [ ] Test squad creation works
- [ ] Test formation editor works

### Phase 4: Polish ✅

- [ ] Review all modes for direct ECS mutations
- [ ] Update `ExplorationMode` to use `RenderingCoordinator`
- [ ] Update `InventoryMode` to follow pattern
- [ ] Update CLAUDE.md with new patterns
- [ ] Create `docs/gui_best_practices.md`
- [ ] Add code examples to documentation
- [ ] Final integration testing

---

## Conclusion

This refactoring plan establishes clear separation between:

1. **Presentation Layer (GUI)** - Widget management, rendering, UI state
2. **Action Bridge Layer** - Translates UI actions to game logic calls
3. **Game Logic Layer** - Pure gameplay systems with no UI dependencies

The proposed architecture maintains ECS best practices while adding clean separation of concerns. The `ActionResult` pattern provides a standardized way to communicate game outcomes and required UI updates, making the system more maintainable and testable.

**Estimated Total Effort:** 32-44 hours across 4 sprints

**Key Benefits:**
- Game logic testable without UI infrastructure
- Clear responsibility boundaries
- Standardized action result pattern
- Reusable rendering systems
- Better separation of concerns
- Easier to add new features

**Risk Mitigation:**
- Incremental refactoring (one system at a time)
- Comprehensive testing at each phase
- Parallel development possible (rendering vs. actions)
- Backward-compatible during migration
