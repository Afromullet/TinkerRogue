# Refactoring Analysis: TinkerRogue Input System
Generated: 2025-12-26
Target: Input handling architecture across `input/` and `gui/` packages

## EXECUTIVE SUMMARY

### Target Analysis
- **Scope**: Complete input handling system spanning two parallel architectures
- **Current State**: Two independent input systems running simultaneously with unclear integration
- **Primary Issues**:
  1. Dual input pipelines (old controller-based + new mode-based) with no coordination
  2. Duplicated logic (Ctrl+Z/Y, mouse-to-tile conversion, hotkey patterns)
  3. Scattered state across 4+ structures (SharedInputState, InputState, PlayerInputStates, BattleMapState)
  4. Mixed responsibilities (GUI modes embedding input handling)
  5. Difficult to test (logic embedded in UI modes)

- **Recommended Direction**: **Unified Input Service Layer** - Transform old InputCoordinator into a centralized input service that captures raw input once, provides shared utilities (mouse-to-tile, hotkey registry), and routes to mode-specific handlers through a clean interface.

### Quick Wins vs Strategic Refactoring

**Immediate Improvements (Days):**
- Extract mouse-to-tile conversion to shared utility (eliminate duplication across 5+ locations)
- Centralize Ctrl+Z/Y handling in single location (remove from combat_input_handler.go)
- Document current input flow with sequence diagram (improves team understanding)

**Medium-Term Goals (1-2 Weeks):**
- Consolidate input capture in single location (eliminate parallel raw input polling)
- Unify InputState structures (merge SharedInputState + InputState + PlayerInputStates)
- Extract mode input handlers to separate structs (separate from UI mode implementations)

**Long-Term Architecture (2-4 Weeks):**
- Implement Input Service Layer pattern (transform InputCoordinator)
- Create Input Command abstraction for testability
- Establish clear input pipeline (Capture → Route → Handle → Execute)

### Consensus Findings

**Agreement Across Perspectives:**
- Current dual system creates confusion and maintenance burden
- Input state consolidation is essential (too many overlapping structures)
- Mouse-to-tile conversion must be centralized (DRY principle)
- Input logic should be testable independently of UI

**Divergent Perspectives:**
- **Architectural purity vs pragmatism**: Some approaches favor strict separation layers, others prefer embedded handlers with better organization
- **Migration risk**: Ranges from "incremental, mode-by-mode" to "unified refactor with feature flags"
- **Abstraction level**: Command pattern vs direct handler methods

**Critical Concerns:**
- Migration must not break existing gameplay during transition
- Over-abstraction could make simple input harder to implement
- Testing infrastructure shouldn't add significant complexity
- Must maintain flexibility for mode-specific input (combat vs exploration vs squad management)

---

## FINAL SYNTHESIZED APPROACHES

### Approach 1: Input Service Layer (Incremental Consolidation)

**Strategic Focus**: Transform old InputCoordinator into centralized input service, migrate mode-by-mode

**Problem Statement**:
The current system polls raw Ebiten input in two separate locations (old InputCoordinator AND new UIModeManager.updateInputState), creating race conditions, duplication, and unclear ownership. Mouse-to-tile conversion is repeated 5+ times across different mode handlers. Ctrl+Z/Y handling exists in both CommandHistory and combat_input_handler.

**Solution Overview**:
Create a **unified InputService** that captures raw Ebiten input once per frame, provides shared utilities (mouse-to-tile, hotkey matching), and exposes a clean InputContext to mode handlers. Old InputCoordinator becomes this service. Migrate modes incrementally to use the service while maintaining backward compatibility.

**Code Example**:

*Before (Current - Duplicated Input Capture):*
```go
// gui/core/modemanager.go (NEW SYSTEM)
func (umm *UIModeManager) updateInputState() {
    // Mouse position
    umm.inputState.MouseX, umm.inputState.MouseY = ebiten.CursorPosition()

    // Mouse buttons
    if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
        umm.inputState.MousePressed = true
        // ... more logic
    }

    // Keyboard
    for _, key := range keysToTrack {
        isPressed := ebiten.IsKeyPressed(key)
        umm.inputState.KeysPressed[key] = isPressed
        // ... more logic
    }
}

// input/movementcontroller.go (OLD SYSTEM)
func (mc *MovementController) HandleInput() bool {
    if inpututil.IsKeyJustReleased(ebiten.KeyW) {
        mc.movePlayer(0, -1)
        // ... more logic
    }
    // ... more key checks
}
```

*After (Unified Input Service):*
```go
// input/inputservice.go (TRANSFORMED FROM InputCoordinator)
package input

import (
    "game_main/world/coords"
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

// InputService captures raw input once per frame and provides shared utilities
type InputService struct {
    currentFrame  *InputFrame
    previousFrame *InputFrame
    coordManager  *coords.CoordinateManager
}

// InputFrame represents all input state for a single frame
type InputFrame struct {
    MouseX      int
    MouseY      int
    MouseLeft   bool
    MouseRight  bool
    KeysPressed map[ebiten.Key]bool

    // Derived state (computed once)
    MouseLogicalPos coords.LogicalPosition
}

func NewInputService(coordManager *coords.CoordinateManager) *InputService {
    return &InputService{
        currentFrame:  newInputFrame(),
        previousFrame: newInputFrame(),
        coordManager:  coordManager,
    }
}

// CaptureInput polls Ebiten once per frame - called from main Update()
func (is *InputService) CaptureInput(playerPos coords.LogicalPosition) {
    // Swap frames
    is.previousFrame, is.currentFrame = is.currentFrame, is.previousFrame

    // Capture raw input
    is.currentFrame.MouseX, is.currentFrame.MouseY = ebiten.CursorPosition()
    is.currentFrame.MouseLeft = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
    is.currentFrame.MouseRight = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)

    // Track all keys (expanded as needed)
    for key := ebiten.Key0; key <= ebiten.KeyMax; key++ {
        is.currentFrame.KeysPressed[key] = ebiten.IsKeyPressed(key)
    }

    // Compute derived state ONCE (mouse-to-tile conversion)
    is.currentFrame.MouseLogicalPos = graphics.MouseToLogicalPosition(
        is.currentFrame.MouseX,
        is.currentFrame.MouseY,
        playerPos,
    )
}

// Query methods for input state
func (is *InputService) IsKeyPressed(key ebiten.Key) bool {
    return is.currentFrame.KeysPressed[key]
}

func (is *InputService) IsKeyJustPressed(key ebiten.Key) bool {
    return is.currentFrame.KeysPressed[key] && !is.previousFrame.KeysPressed[key]
}

func (is *InputService) GetMouseLogicalPosition() coords.LogicalPosition {
    return is.currentFrame.MouseLogicalPos // Already computed!
}

func (is *InputService) IsMouseButtonJustPressed(button ebiten.MouseButton) bool {
    current := button == ebiten.MouseButtonLeft && is.currentFrame.MouseLeft ||
               button == ebiten.MouseButtonRight && is.currentFrame.MouseRight
    previous := button == ebiten.MouseButtonLeft && is.previousFrame.MouseLeft ||
                button == ebiten.MouseButtonRight && is.previousFrame.MouseRight
    return current && !previous
}

// GetInputContext creates read-only context for mode handlers
func (is *InputService) GetInputContext() *InputContext {
    return &InputContext{
        frame:   is.currentFrame,
        service: is,
    }
}

// InputContext is passed to mode handlers (read-only view)
type InputContext struct {
    frame   *InputFrame
    service *InputService
}

func (ic *InputContext) IsKeyJustPressed(key ebiten.Key) bool {
    return ic.service.IsKeyJustPressed(key)
}

func (ic *InputContext) GetMouseLogicalPos() coords.LogicalPosition {
    return ic.frame.MouseLogicalPos
}

func (ic *InputContext) IsMouseLeftJustPressed() bool {
    return ic.service.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
}
```

*Mode Handler Example (Using Input Service):*
```go
// gui/guicombat/combat_input_handler.go (UPDATED)
func (cih *CombatInputHandler) HandleInput(inputCtx *input.InputContext) bool {
    // Mouse clicks - no more manual conversion!
    if inputCtx.IsMouseLeftJustPressed() {
        clickedPos := inputCtx.GetMouseLogicalPos()

        if cih.battleMapState.InMoveMode {
            cih.handleMovementClick(clickedPos)
        } else {
            cih.handleSquadClick(clickedPos)
        }
        return true
    }

    // Keyboard - cleaner API
    if inputCtx.IsKeyJustPressed(ebiten.KeyA) {
        cih.actionHandler.ToggleAttackMode()
        return true
    }

    // ... more input handling
    return false
}

// ELIMINATED: Manual mouse-to-tile conversion (was 8+ lines repeated 5 times)
```

*Main Game Loop Integration:*
```go
// game_main/main.go (or wherever Update() is)
type Game struct {
    inputService     *input.InputService
    modeCoordinator  *core.GameModeCoordinator
    // ... other fields
}

func (g *Game) Update() error {
    // 1. Capture input ONCE per frame
    g.inputService.CaptureInput(*g.playerData.Pos)

    // 2. Get input context for this frame
    inputCtx := g.inputService.GetInputContext()

    // 3. Pass to mode coordinator (routes to active mode)
    g.modeCoordinator.HandleInput(inputCtx)

    // 4. Update game logic
    g.modeCoordinator.Update(deltaTime)

    return nil
}
```

**Key Changes**:
1. **Single input capture point** - Raw Ebiten polling happens once in InputService.CaptureInput()
2. **Mouse-to-tile computed once** - Stored in InputFrame.MouseLogicalPos, no repeated conversions
3. **Clean query API** - IsKeyJustPressed(), GetMouseLogicalPos() instead of manual checks
4. **Read-only context** - InputContext passed to handlers prevents accidental state mutation
5. **Backward compatible** - Old controllers can migrate incrementally to use InputService

**Value Proposition**:
- **Maintainability**: Single source of truth for input state, DRY mouse-to-tile conversion
- **Readability**: Clear input pipeline (Capture → Route → Handle), no scattered polling
- **Extensibility**: Easy to add new input utilities (gamepad support, input recording, etc.)
- **Complexity Impact**:
  - Eliminates ~50 lines of duplicated mouse-to-tile conversion
  - Reduces frame-to-frame key tracking from 3 implementations to 1
  - Centralizes input capture from 2 locations to 1

**Implementation Strategy**:
1. **Week 1**: Create InputService, migrate mouse-to-tile conversion
   - Extract InputService from InputCoordinator
   - Add GetMouseLogicalPos() method with cached conversion
   - Update combat_input_handler.go to use it (prove concept)

2. **Week 2**: Consolidate input capture
   - Move raw Ebiten polling to InputService.CaptureInput()
   - Update UIModeManager to use InputContext instead of raw polling
   - Remove duplicate input capture from modemanager.go

3. **Week 3**: Migrate mode handlers incrementally
   - Update each mode's HandleInput() to accept InputContext
   - Remove individual raw input calls (ebiten.IsKeyPressed, etc.)
   - Test each mode thoroughly after migration

4. **Week 4**: Deprecate old structures
   - Remove SharedInputState (merge into InputFrame if needed)
   - Consolidate PlayerInputStates into InputService
   - Update all controllers to use InputService

**Advantages**:
- **Incremental migration**: Can migrate mode-by-mode without breaking existing functionality
- **No performance overhead**: Single input capture is actually faster than current dual polling
- **Clear ownership**: InputService owns input capture, modes own input handling logic
- **Easy to test**: InputContext can be mocked for unit tests of mode handlers
- **Foundation for future features**: Input recording/playback, gamepad support, input buffering

**Drawbacks & Risks**:
- **Migration effort**: Requires touching all mode handlers (~10 files) - *Mitigation: Do incrementally, one mode per day*
- **Learning curve**: Team must understand InputService API - *Mitigation: Good documentation + example mode*
- **Potential bugs during transition**: Mixed old/new input during migration - *Mitigation: Feature flag to switch between old/new per mode*
- **Over-centralization risk**: InputService could become god object - *Mitigation: Keep it focused on capture + utilities only, not routing logic*

**Effort Estimate**:
- **Time**: 3-4 weeks (1 week design + 2-3 weeks incremental migration)
- **Complexity**: Medium (architectural change but clear incremental path)
- **Risk**: Low-Medium (incremental migration reduces risk, good rollback at each step)
- **Files Impacted**:
  - Create: `input/inputservice.go` (new, ~200 lines)
  - Modify: `gui/core/modemanager.go`, `gui/guicombat/combat_input_handler.go`, all mode HandleInput() methods (~10 files)
  - Deprecate: `input/inputcoordinator.go` (transforms into InputService), eventually remove SharedInputState

**Critical Assessment**:
This approach provides the **best balance of pragmatism and architectural improvement**. It solves the immediate pain points (duplication, scattered polling) while establishing a foundation for future improvements (testability, input commands). The incremental migration path keeps risk low - you can validate each mode works before moving to the next. The InputService doesn't over-abstract (no Command pattern yet) but gives clear benefits (DRY, single capture point). This is the **recommended starting point** for teams that want clear improvement without big-bang rewrites.

---

### Approach 2: Mode-Centric Input Handlers (Separation + Organization)

**Strategic Focus**: Keep input handling in modes but extract to dedicated handler structs, improve organization

**Problem Statement**:
Current modes embed HandleInput() directly in UIMode implementation (combatmode.go has HandleInput at line 516), mixing UI rendering concerns with input logic. This makes it hard to test input behavior independently and creates large mode files (combatmode.go is 579 lines). Additionally, common input patterns (Ctrl+Z/Y, ESC, hotkeys) are scattered across BaseMode.HandleCommonInput() and individual mode implementations.

**Solution Overview**:
Extract input handling from each mode into dedicated **ModeInputHandler** structs that implement a common interface. Each mode creates its handler during initialization and delegates to it. Enhance BaseMode with a **CommonInputHandler** that centralizes Ctrl+Z/Y, ESC, and hotkey processing. Modes remain responsible for their input logic but with better organization and testability.

**Code Example**:

*Before (Current - Input Embedded in Mode):*
```go
// gui/guicombat/combatmode.go
type CombatMode struct {
    gui.BaseMode
    // ... UI components (40+ fields)
}

func (cm *CombatMode) HandleInput(inputState *core.InputState) bool {
    // 60+ lines of input handling mixed with UI mode
    if cm.HandleCommonInput(inputState) {
        return true
    }

    cm.inputHandler.SetPlayerPosition(cm.Context.PlayerData.Pos)
    cm.inputHandler.SetCurrentFactionID(cm.combatService.TurnManager.GetCurrentFaction())

    if cm.inputHandler.HandleInput(inputState) {
        return true
    }

    if inputState.KeysJustPressed[ebiten.KeySpace] {
        cm.handleEndTurn()
        return true
    }

    // ... 30+ more lines for H key (danger visualization), Shift+H, etc.
    return false
}
```

*After (Separated Input Handler):*
```go
// gui/guicombat/combat_input_handler.go (ENHANCED)
package guicombat

import (
    "game_main/gui/core"
    "game_main/world/coords"
    "github.com/hajimehoshi/ebiten/v2"
)

// CombatModeInputHandler handles all combat mode input (separated from UI rendering)
type CombatModeInputHandler struct {
    // Dependencies (not UI components)
    actionHandler    *CombatActionHandler
    combatService    *combatservices.CombatService
    battleMapState   *core.BattleMapState
    dangerVisualizer *behavior.DangerVisualizer
    logManager       *CombatLogManager

    // Callbacks for actions that affect UI
    onEndTurn        func()
    onVisualizationToggle func(status string)
}

func NewCombatModeInputHandler(
    actionHandler *CombatActionHandler,
    combatService *combatservices.CombatService,
    battleMapState *core.BattleMapState,
    dangerVisualizer *behavior.DangerVisualizer,
    logManager *CombatLogManager,
) *CombatModeInputHandler {
    return &CombatModeInputHandler{
        actionHandler:    actionHandler,
        combatService:    combatService,
        battleMapState:   battleMapState,
        dangerVisualizer: dangerVisualizer,
        logManager:       logManager,
    }
}

// SetCallbacks configures UI update callbacks (set by mode during initialization)
func (cmih *CombatModeInputHandler) SetCallbacks(onEndTurn func(), onVisualizationToggle func(string)) {
    cmih.onEndTurn = onEndTurn
    cmih.onVisualizationToggle = onVisualizationToggle
}

// HandleInput processes all combat-specific input
func (cmih *CombatModeInputHandler) HandleInput(inputState *core.InputState, playerPos coords.LogicalPosition) bool {
    // Mouse input
    if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
        clickedPos := graphics.MouseToLogicalPosition(inputState.MouseX, inputState.MouseY, playerPos)

        if cmih.battleMapState.InMoveMode {
            return cmih.handleMovementClick(clickedPos)
        } else {
            return cmih.handleSquadClick(clickedPos)
        }
    }

    // Keyboard input - organized by category

    // Combat actions
    if inputState.KeysJustPressed[ebiten.KeyA] {
        cmih.actionHandler.ToggleAttackMode()
        return true
    }

    if inputState.KeysJustPressed[ebiten.KeyM] {
        cmih.actionHandler.ToggleMoveMode()
        return true
    }

    if inputState.KeysJustPressed[ebiten.KeyTab] {
        cmih.actionHandler.CycleSquadSelection()
        return true
    }

    // Turn management
    if inputState.KeysJustPressed[ebiten.KeySpace] {
        if cmih.onEndTurn != nil {
            cmih.onEndTurn()
        }
        return true
    }

    // Danger visualization (H key + modifiers)
    if inputState.KeysJustPressed[ebiten.KeyH] {
        return cmih.handleDangerVisualization(inputState)
    }

    // Target selection (1-3 keys in attack mode)
    if cmih.battleMapState.InAttackMode {
        if inputState.KeysJustPressed[ebiten.Key1] {
            cmih.actionHandler.SelectEnemyTarget(0)
            return true
        }
        if inputState.KeysJustPressed[ebiten.Key2] {
            cmih.actionHandler.SelectEnemyTarget(1)
            return true
        }
        if inputState.KeysJustPressed[ebiten.Key3] {
            cmih.actionHandler.SelectEnemyTarget(2)
            return true
        }
    }

    return false
}

// handleDangerVisualization centralizes H key logic (Shift+H, H alone, Left Ctrl)
func (cmih *CombatModeInputHandler) handleDangerVisualization(inputState *core.InputState) bool {
    shiftPressed := inputState.KeysPressed[ebiten.KeyShift] ||
                    inputState.KeysPressed[ebiten.KeyShiftLeft] ||
                    inputState.KeysPressed[ebiten.KeyShiftRight]

    if shiftPressed {
        // Shift+H: Switch between enemy/player threat view
        cmih.dangerVisualizer.SwitchView()
        viewName := "Enemy Threats"
        if cmih.dangerVisualizer.GetViewMode() == behavior.ViewPlayerThreats {
            viewName = "Player Threats"
        }
        if cmih.onVisualizationToggle != nil {
            cmih.onVisualizationToggle(fmt.Sprintf("Switched to %s view", viewName))
        }
    } else {
        // H alone: Toggle visualization on/off
        cmih.dangerVisualizer.Toggle()
        status := "enabled"
        if !cmih.dangerVisualizer.IsActive() {
            status = "disabled"
        }
        if cmih.onVisualizationToggle != nil {
            cmih.onVisualizationToggle(fmt.Sprintf("Danger visualization %s", status))
        }
    }
    return true
}

// ... handleMovementClick, handleSquadClick (existing methods)
```

*Enhanced BaseMode with Common Input Handler:*
```go
// gui/basemode.go (ENHANCED)
package gui

import (
    "game_main/gui/core"
    "github.com/hajimehoshi/ebiten/v2"
)

// CommonInputHandler centralizes input patterns shared across all modes
type CommonInputHandler struct {
    modeManager    *core.UIModeManager
    commandHistory *CommandHistory
    hotkeys        map[ebiten.Key]InputBinding
    returnMode     string
}

func NewCommonInputHandler(modeManager *core.UIModeManager, returnMode string) *CommonInputHandler {
    return &CommonInputHandler{
        modeManager: modeManager,
        hotkeys:     make(map[ebiten.Key]InputBinding),
        returnMode:  returnMode,
    }
}

// SetCommandHistory enables undo/redo support
func (cih *CommonInputHandler) SetCommandHistory(ch *CommandHistory) {
    cih.commandHistory = ch
}

// RegisterHotkey adds a mode transition hotkey
func (cih *CommonInputHandler) RegisterHotkey(key ebiten.Key, targetMode string) {
    cih.hotkeys[key] = InputBinding{
        Key:        key,
        TargetMode: targetMode,
        Reason:     fmt.Sprintf("%s key pressed", key.String()),
    }
}

// HandleInput processes common input (ESC, hotkeys, undo/redo)
// Returns true if input was consumed
func (cih *CommonInputHandler) HandleInput(inputState *core.InputState) bool {
    // Undo/Redo (Ctrl+Z, Ctrl+Y) - CENTRALIZED HERE
    if cih.commandHistory != nil {
        if cih.commandHistory.HandleInput(inputState) {
            return true
        }
    }

    // ESC key - return to designated mode
    if inputState.KeysJustPressed[ebiten.KeyEscape] {
        if cih.returnMode != "" {
            if returnMode, exists := cih.modeManager.GetMode(cih.returnMode); exists {
                cih.modeManager.RequestTransition(returnMode, "ESC pressed")
                return true
            }
        }
    }

    // Registered hotkeys for mode transitions
    for key, binding := range cih.hotkeys {
        if inputState.KeysJustPressed[key] {
            if targetMode, exists := cih.modeManager.GetMode(binding.TargetMode); exists {
                cih.modeManager.RequestTransition(targetMode, binding.Reason)
                return true
            }
        }
    }

    return false
}

// BaseMode updated to use CommonInputHandler
type BaseMode struct {
    // ... existing fields
    commonInputHandler *CommonInputHandler // NEW
}

func (bm *BaseMode) InitializeBase(ctx *core.UIContext) {
    // ... existing initialization

    // Create common input handler
    bm.commonInputHandler = NewCommonInputHandler(bm.ModeManager, bm.returnMode)
}

// InitializeCommandHistory now registers with common handler
func (bm *BaseMode) InitializeCommandHistory(onRefresh func()) {
    bm.CommandHistory = NewCommandHistory(bm.SetStatus, onRefresh)
    bm.commonInputHandler.SetCommandHistory(bm.CommandHistory) // REGISTER
}

// RegisterHotkey now delegates to common handler
func (bm *BaseMode) RegisterHotkey(key ebiten.Key, targetMode string) {
    bm.commonInputHandler.RegisterHotkey(key, targetMode)
}

// HandleCommonInput now delegates to common handler
func (bm *BaseMode) HandleCommonInput(inputState *core.InputState) bool {
    return bm.commonInputHandler.HandleInput(inputState)
}
```

*Updated CombatMode (Simplified):*
```go
// gui/guicombat/combatmode.go (UPDATED)
type CombatMode struct {
    gui.BaseMode

    // Separated input handler
    modeInputHandler *CombatModeInputHandler

    // ... UI components (rendering-focused)
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    // ... UI building

    // Create mode-specific input handler
    cm.modeInputHandler = NewCombatModeInputHandler(
        cm.actionHandler,
        cm.combatService,
        ctx.ModeCoordinator.GetBattleMapState(),
        cm.dangerVisualizer,
        cm.logManager,
    )

    // Set callbacks for UI updates
    cm.modeInputHandler.SetCallbacks(
        cm.handleEndTurn,
        func(msg string) { cm.logManager.UpdateTextArea(cm.combatLogArea, msg) },
    )

    return nil
}

func (cm *CombatMode) HandleInput(inputState *core.InputState) bool {
    // Common input (ESC, hotkeys, Ctrl+Z/Y) - CENTRALIZED
    if cm.HandleCommonInput(inputState) {
        return true
    }

    // Mode-specific input - DELEGATED TO HANDLER
    return cm.modeInputHandler.HandleInput(inputState, *cm.Context.PlayerData.Pos)
}

// CombatMode is now 200 lines instead of 579 (input logic moved to handler)
```

**Key Changes**:
1. **Extracted input handlers** - Each mode has dedicated ModeInputHandler struct (e.g., CombatModeInputHandler)
2. **CommonInputHandler** - Centralizes Ctrl+Z/Y, ESC, hotkey processing (no more duplication)
3. **Callback pattern** - Input handlers use callbacks for UI updates (maintains separation)
4. **Testable handlers** - Can unit test CombatModeInputHandler without creating full CombatMode
5. **Organized by concern** - Mode files focus on UI building, handler files focus on input logic

**Value Proposition**:
- **Maintainability**: Input logic separated from UI rendering, easier to find and modify
- **Readability**: Smaller mode files (combatmode.go: 579 → ~200 lines), input logic organized by category
- **Extensibility**: Easy to add new input without bloating mode files
- **Complexity Impact**:
  - Ctrl+Z/Y handling: 2 implementations → 1 (CommandHistory.HandleInput in CommonInputHandler)
  - Mode file sizes: Reduced by ~60% (input logic extracted)
  - New abstractions: +1 struct per mode (ModeInputHandler) but clearer separation

**Implementation Strategy**:
1. **Week 1**: Create CommonInputHandler and update BaseMode
   - Extract common input logic to CommonInputHandler
   - Update BaseMode.HandleCommonInput() to delegate
   - Test with existing modes (backward compatible)

2. **Week 2**: Extract combat mode input handler
   - Create CombatModeInputHandler with all combat input logic
   - Update CombatMode to use handler
   - Remove Ctrl+Z/Y from combat_input_handler.go (use CommonInputHandler)
   - Test combat mode thoroughly

3. **Week 3**: Extract handlers for other modes
   - Create ExplorationModeInputHandler, InventoryModeInputHandler, etc.
   - Update each mode to use its handler
   - Validate each mode works after extraction

4. **Week 4**: Add unit tests for input handlers
   - Test CombatModeInputHandler independently
   - Test CommonInputHandler with mock mode manager
   - Document input handling patterns for future modes

**Advantages**:
- **Pragmatic separation**: Input logic separated but still mode-owned (not over-abstracted)
- **Incremental migration**: Can extract handlers mode-by-mode
- **Easy testing**: Input handlers testable without full UI mode setup
- **Clear organization**: New developers know where to find input logic (ModeInputHandler files)
- **Eliminates duplication**: CommonInputHandler removes Ctrl+Z/Y, ESC, hotkey duplication

**Drawbacks & Risks**:
- **More files**: +1 handler file per mode (~10 new files) - *Mitigation: Clear naming convention, co-located with modes*
- **Callback complexity**: Handlers need callbacks to trigger UI updates - *Mitigation: Use function types, document patterns*
- **Doesn't fix dual input capture**: Still have InputCoordinator + UIModeManager polling separately - *Mitigation: Can combine with Approach 1 later*
- **Learning curve**: Team must understand handler extraction pattern - *Mitigation: Provide clear example (CombatModeInputHandler), document in CLAUDE.md*

**Effort Estimate**:
- **Time**: 3-4 weeks (1 week CommonInputHandler + 2-3 weeks mode-by-mode extraction)
- **Complexity**: Low-Medium (mostly refactoring, clear pattern to follow)
- **Risk**: Low (incremental extraction, modes still work during transition)
- **Files Impacted**:
  - Create: `gui/guicombat/combat_mode_input_handler.go`, `gui/guimodes/exploration_input_handler.go`, etc. (~10 new files)
  - Modify: `gui/basemode.go` (CommonInputHandler), all mode HandleInput() methods (~10 files)
  - Simplify: Mode files (combatmode.go etc.) become 50-60% smaller

**Critical Assessment**:
This approach focuses on **organization and separation** without changing the overall architecture. It's **lower risk** than Approach 1 because it doesn't touch input capture or consolidation - just extracts and organizes existing logic. Good choice if you want **quick maintainability wins** without architectural changes. However, it **doesn't solve the dual input system problem** (InputCoordinator + UIModeManager still poll separately) or eliminate mouse-to-tile duplication. Best used as a **first step** before tackling deeper consolidation, or as a **standalone improvement** if you're happy with the current dual-system architecture.

---

### Approach 3: Input Command Pattern (Testability + Replay)

**Strategic Focus**: Abstract input into Commands for testability, input recording, and AI/scripting support

**Problem Statement**:
Current input handling is procedural (direct method calls in HandleInput) making it impossible to test input sequences, record gameplay for replay, or script AI behavior. Input validation (can squad move? can attack target?) is scattered across handlers. There's no way to undo complex multi-step actions or replay a turn for debugging.

**Solution Overview**:
Implement the **Command Pattern** for input handling. Each input creates a Command object (MoveCommand, AttackCommand, EndTurnCommand) that encapsulates the action and its validation. Commands are executed through a central CommandExecutor that provides undo/redo, validation, and logging. This enables comprehensive testing (test commands directly), input recording (serialize command sequence), and future features (AI using same commands as player, replay system).

**Code Example**:

*Before (Current - Direct Execution):*
```go
// gui/guicombat/combat_input_handler.go
func (cih *CombatInputHandler) HandleInput(inputState *core.InputState) bool {
    if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
        clickedPos := graphics.MouseToLogicalPosition(inputState.MouseX, inputState.MouseY, *cih.playerPos)

        if cih.battleMapState.InMoveMode {
            // Direct execution - no validation abstraction, hard to test
            validTiles := cih.actionHandler.combatService.MovementSystem.GetValidMovementTiles(cih.battleMapState.SelectedSquadID)
            isValidTile := false
            for _, validPos := range validTiles {
                if validPos.X == clickedPos.X && validPos.Y == clickedPos.Y {
                    isValidTile = true
                    break
                }
            }
            if !isValidTile {
                return false
            }

            // Execute move - can't undo, can't test in isolation
            cih.actionHandler.MoveSquad(cih.battleMapState.SelectedSquadID, clickedPos)
        }
        return true
    }
    return false
}
```

*After (Command Pattern):*
```go
// tactical/inputcommands/commands.go (NEW)
package inputcommands

import (
    "fmt"
    "game_main/tactical/combatservices"
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// InputCommand represents a player action that can be validated, executed, and undone
type InputCommand interface {
    // Validate checks if command can be executed (without changing state)
    Validate() error

    // Execute performs the command (changes game state)
    Execute() error

    // Undo reverses the command (restores previous state)
    Undo() error

    // Description returns human-readable description for logging
    Description() string
}

// MoveSquadCommand moves a squad to a target position
type MoveSquadCommand struct {
    squadID       ecs.EntityID
    targetPos     coords.LogicalPosition
    combatService *combatservices.CombatService

    // State for undo
    previousPos coords.LogicalPosition
}

func NewMoveSquadCommand(squadID ecs.EntityID, targetPos coords.LogicalPosition, combatService *combatservices.CombatService) *MoveSquadCommand {
    return &MoveSquadCommand{
        squadID:       squadID,
        targetPos:     targetPos,
        combatService: combatService,
    }
}

func (cmd *MoveSquadCommand) Validate() error {
    // Check if squad exists
    squad := cmd.combatService.ECSManager.FindEntityByID(cmd.squadID)
    if squad == nil {
        return fmt.Errorf("squad %d not found", cmd.squadID)
    }

    // Check if target position is valid for movement
    validTiles := cmd.combatService.MovementSystem.GetValidMovementTiles(cmd.squadID)
    isValid := false
    for _, validPos := range validTiles {
        if validPos.X == cmd.targetPos.X && validPos.Y == cmd.targetPos.Y {
            isValid = true
            break
        }
    }

    if !isValid {
        return fmt.Errorf("position (%d, %d) is not a valid move target", cmd.targetPos.X, cmd.targetPos.Y)
    }

    return nil
}

func (cmd *MoveSquadCommand) Execute() error {
    // Get current position for undo
    currentPos := cmd.combatService.MovementSystem.GetSquadPosition(cmd.squadID)
    cmd.previousPos = currentPos

    // Execute move through combat service
    return cmd.combatService.MovementSystem.MoveSquad(cmd.squadID, cmd.targetPos)
}

func (cmd *MoveSquadCommand) Undo() error {
    // Move back to previous position
    return cmd.combatService.MovementSystem.MoveSquad(cmd.squadID, cmd.previousPos)
}

func (cmd *MoveSquadCommand) Description() string {
    return fmt.Sprintf("Move squad %d to (%d, %d)", cmd.squadID, cmd.targetPos.X, cmd.targetPos.Y)
}

// AttackSquadCommand executes an attack
type AttackSquadCommand struct {
    attackerID    ecs.EntityID
    defenderID    ecs.EntityID
    combatService *combatservices.CombatService

    // State for undo (captures defender health before attack)
    defenderPreviousHealth int
}

func NewAttackSquadCommand(attackerID, defenderID ecs.EntityID, combatService *combatservices.CombatService) *AttackSquadCommand {
    return &AttackSquadCommand{
        attackerID:    attackerID,
        defenderID:    defenderID,
        combatService: combatService,
    }
}

func (cmd *AttackSquadCommand) Validate() error {
    // Check both squads exist
    // Check attacker can attack (has action points, in range, etc.)
    // Check defender is valid target (alive, enemy faction, etc.)
    // ... validation logic
    return nil
}

func (cmd *AttackSquadCommand) Execute() error {
    // Capture defender state for undo
    defenderHealth := cmd.combatService.GetSquadHealth(cmd.defenderID)
    cmd.defenderPreviousHealth = defenderHealth

    // Execute attack
    return cmd.combatService.CombatSystem.ExecuteAttack(cmd.attackerID, cmd.defenderID)
}

func (cmd *AttackSquadCommand) Undo() error {
    // Restore defender health (simplified - real impl would restore full state)
    return cmd.combatService.SetSquadHealth(cmd.defenderID, cmd.defenderPreviousHealth)
}

func (cmd *AttackSquadCommand) Description() string {
    return fmt.Sprintf("Squad %d attacks squad %d", cmd.attackerID, cmd.defenderID)
}

// EndTurnCommand ends current faction's turn
type EndTurnCommand struct {
    combatService   *combatservices.CombatService
    previousFaction ecs.EntityID
}

func NewEndTurnCommand(combatService *combatservices.CombatService) *EndTurnCommand {
    return &EndTurnCommand{
        combatService: combatService,
    }
}

func (cmd *EndTurnCommand) Validate() error {
    // Check if there's an active turn to end
    currentFaction := cmd.combatService.TurnManager.GetCurrentFaction()
    if currentFaction == 0 {
        return fmt.Errorf("no active turn to end")
    }
    return nil
}

func (cmd *EndTurnCommand) Execute() error {
    cmd.previousFaction = cmd.combatService.TurnManager.GetCurrentFaction()
    return cmd.combatService.TurnManager.EndTurn()
}

func (cmd *EndTurnCommand) Undo() error {
    // Undo turn change (complex - may need to restore full turn state)
    return cmd.combatService.TurnManager.SetCurrentFaction(cmd.previousFaction)
}

func (cmd *EndTurnCommand) Description() string {
    return "End turn"
}
```

*Command Executor (Orchestrates Commands):*
```go
// tactical/inputcommands/executor.go (NEW)
package inputcommands

import (
    "fmt"
)

// CommandExecutor validates and executes commands with undo/redo support
type CommandExecutor struct {
    history    []InputCommand
    undoStack  []InputCommand
    maxHistory int
}

func NewCommandExecutor() *CommandExecutor {
    return &CommandExecutor{
        history:    make([]InputCommand, 0),
        undoStack:  make([]InputCommand, 0),
        maxHistory: 100, // Keep last 100 commands
    }
}

// Execute validates and runs a command
func (ce *CommandExecutor) Execute(cmd InputCommand) error {
    // Validate before execution
    if err := cmd.Validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    // Execute command
    if err := cmd.Execute(); err != nil {
        return fmt.Errorf("execution failed: %w", err)
    }

    // Add to history
    ce.history = append(ce.history, cmd)
    if len(ce.history) > ce.maxHistory {
        ce.history = ce.history[1:] // Remove oldest
    }

    // Clear redo stack (new command invalidates redo)
    ce.undoStack = nil

    return nil
}

// Undo reverses the last command
func (ce *CommandExecutor) Undo() error {
    if len(ce.history) == 0 {
        return fmt.Errorf("no commands to undo")
    }

    // Pop last command
    lastCmd := ce.history[len(ce.history)-1]
    ce.history = ce.history[:len(ce.history)-1]

    // Undo it
    if err := lastCmd.Undo(); err != nil {
        return fmt.Errorf("undo failed: %w", err)
    }

    // Add to redo stack
    ce.undoStack = append(ce.undoStack, lastCmd)

    return nil
}

// Redo re-executes the last undone command
func (ce *CommandExecutor) Redo() error {
    if len(ce.undoStack) == 0 {
        return fmt.Errorf("no commands to redo")
    }

    // Pop from redo stack
    redoCmd := ce.undoStack[len(ce.undoStack)-1]
    ce.undoStack = ce.undoStack[:len(ce.undoStack)-1]

    // Re-execute (no validation - already validated)
    if err := redoCmd.Execute(); err != nil {
        return fmt.Errorf("redo failed: %w", err)
    }

    // Add back to history
    ce.history = append(ce.history, redoCmd)

    return nil
}

// GetHistory returns command history (for replay, debugging)
func (ce *CommandExecutor) GetHistory() []InputCommand {
    return ce.history
}

// CanUndo returns whether undo is available
func (ce *CommandExecutor) CanUndo() bool {
    return len(ce.history) > 0
}

// CanRedo returns whether redo is available
func (ce *CommandExecutor) CanRedo() bool {
    return len(ce.undoStack) > 0
}
```

*Input Handler Using Commands:*
```go
// gui/guicombat/combat_input_handler.go (UPDATED)
package guicombat

import (
    "game_main/tactical/inputcommands"
    "github.com/hajimehoshi/ebiten/v2"
)

type CombatInputHandler struct {
    commandExecutor *inputcommands.CommandExecutor
    combatService   *combatservices.CombatService
    battleMapState  *core.BattleMapState
}

func NewCombatInputHandler(combatService *combatservices.CombatService, battleMapState *core.BattleMapState) *CombatInputHandler {
    return &CombatInputHandler{
        commandExecutor: inputcommands.NewCommandExecutor(),
        combatService:   combatService,
        battleMapState:  battleMapState,
    }
}

func (cih *CombatInputHandler) HandleInput(inputState *core.InputState, playerPos coords.LogicalPosition) bool {
    // Mouse click - create MoveCommand
    if inputState.MouseButton == ebiten.MouseButtonLeft && inputState.MousePressed {
        clickedPos := graphics.MouseToLogicalPosition(inputState.MouseX, inputState.MouseY, playerPos)

        if cih.battleMapState.InMoveMode {
            // Create command
            cmd := inputcommands.NewMoveSquadCommand(
                cih.battleMapState.SelectedSquadID,
                clickedPos,
                cih.combatService,
            )

            // Execute through executor (validates + executes + adds to history)
            if err := cih.commandExecutor.Execute(cmd); err != nil {
                fmt.Printf("Move failed: %v\n", err) // Or show in UI
                return false
            }
            return true
        }
    }

    // Space key - create EndTurnCommand
    if inputState.KeysJustPressed[ebiten.KeySpace] {
        cmd := inputcommands.NewEndTurnCommand(cih.combatService)

        if err := cih.commandExecutor.Execute(cmd); err != nil {
            fmt.Printf("End turn failed: %v\n", err)
            return false
        }
        return true
    }

    // Ctrl+Z - undo last command
    if inputState.KeysJustPressed[ebiten.KeyZ] && inputState.KeysPressed[ebiten.KeyControl] {
        if err := cih.commandExecutor.Undo(); err != nil {
            fmt.Printf("Undo failed: %v\n", err)
        }
        return true
    }

    // Ctrl+Y - redo last command
    if inputState.KeysJustPressed[ebiten.KeyY] && inputState.KeysPressed[ebiten.KeyControl] {
        if err := cih.commandExecutor.Redo(); err != nil {
            fmt.Printf("Redo failed: %v\n", err)
        }
        return true
    }

    return false
}

// GetCommandHistory exposes history for replay, debugging
func (cih *CombatInputHandler) GetCommandHistory() []inputcommands.InputCommand {
    return cih.commandExecutor.GetHistory()
}
```

*Unit Test Example (Now Possible!):*
```go
// tactical/inputcommands/commands_test.go (NEW)
package inputcommands_test

import (
    "testing"
    "game_main/tactical/inputcommands"
    "game_main/world/coords"
)

func TestMoveSquadCommand_Validate(t *testing.T) {
    // Setup test combat service with known state
    combatService := setupTestCombatService()

    tests := []struct {
        name        string
        squadID     ecs.EntityID
        targetPos   coords.LogicalPosition
        expectError bool
    }{
        {
            name:        "valid move",
            squadID:     1,
            targetPos:   coords.LogicalPosition{X: 5, Y: 5},
            expectError: false,
        },
        {
            name:        "invalid squad",
            squadID:     999,
            targetPos:   coords.LogicalPosition{X: 5, Y: 5},
            expectError: true,
        },
        {
            name:        "out of range",
            squadID:     1,
            targetPos:   coords.LogicalPosition{X: 100, Y: 100},
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := inputcommands.NewMoveSquadCommand(tt.squadID, tt.targetPos, combatService)
            err := cmd.Validate()

            if tt.expectError && err == nil {
                t.Errorf("expected error but got nil")
            }
            if !tt.expectError && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}

func TestCommandExecutor_UndoRedo(t *testing.T) {
    combatService := setupTestCombatService()
    executor := inputcommands.NewCommandExecutor()

    // Execute move command
    cmd := inputcommands.NewMoveSquadCommand(1, coords.LogicalPosition{X: 5, Y: 5}, combatService)
    if err := executor.Execute(cmd); err != nil {
        t.Fatalf("execute failed: %v", err)
    }

    // Verify squad moved
    newPos := combatService.MovementSystem.GetSquadPosition(1)
    if newPos.X != 5 || newPos.Y != 5 {
        t.Errorf("squad did not move to expected position")
    }

    // Undo move
    if err := executor.Undo(); err != nil {
        t.Fatalf("undo failed: %v", err)
    }

    // Verify squad returned to original position
    undoPos := combatService.MovementSystem.GetSquadPosition(1)
    if undoPos.X == 5 || undoPos.Y == 5 {
        t.Errorf("undo did not restore original position")
    }

    // Redo move
    if err := executor.Redo(); err != nil {
        t.Fatalf("redo failed: %v", err)
    }

    // Verify squad moved again
    redoPos := combatService.MovementSystem.GetSquadPosition(1)
    if redoPos.X != 5 || redoPos.Y != 5 {
        t.Errorf("redo did not restore moved position")
    }
}
```

**Key Changes**:
1. **Command abstraction** - Each input action becomes a Command object (MoveSquadCommand, AttackSquadCommand)
2. **Validation separated** - Validate() method checks preconditions without execution
3. **Undo built-in** - Each command captures state for reversal
4. **CommandExecutor** - Centralizes execution, undo/redo, history
5. **Testable** - Commands can be tested in isolation with mock combat service

**Value Proposition**:
- **Maintainability**: Validation logic centralized in commands, easy to modify rules
- **Readability**: Input handlers become declarative (create command → execute), not procedural
- **Extensibility**: Add new commands without modifying executor or handlers
- **Complexity Impact**:
  - Input validation: Scattered across handlers → Centralized in Command.Validate()
  - Undo/redo: Mode-specific implementations → Unified in CommandExecutor
  - Testing: Impossible → Comprehensive (test commands, executor, integration)
  - New abstractions: +1 interface, +N command types (~10-15 commands)

**Implementation Strategy**:
1. **Week 1**: Design command architecture + core commands
   - Create InputCommand interface
   - Implement MoveSquadCommand, AttackSquadCommand, EndTurnCommand
   - Create CommandExecutor with undo/redo
   - Write unit tests for commands

2. **Week 2**: Integrate commands into combat mode
   - Update CombatInputHandler to use CommandExecutor
   - Replace direct combat service calls with commands
   - Remove old undo/redo implementation
   - Test combat mode thoroughly

3. **Week 3**: Expand command coverage
   - Create SelectSquadCommand, ToggleModeCommand, etc.
   - Update all input handlers to use commands
   - Add command logging/replay infrastructure

4. **Week 4**: Advanced features
   - Implement command serialization (save/load replays)
   - Add AI command generation (AI uses same commands as player)
   - Create input recording tool for debugging

**Advantages**:
- **Comprehensive testing**: Can unit test every player action in isolation
- **Input recording/replay**: Command history enables replay system for debugging
- **AI consistency**: AI can use same commands as player (no separate code paths)
- **Clear validation**: Validate() method makes preconditions explicit
- **Undo/redo anywhere**: Any command can be undone, not just combat moves
- **Future features**: Enables multiplayer (send commands), scripting, tutorials

**Drawbacks & Risks**:
- **High initial complexity**: +50-100% more code (command objects + executor) - *Mitigation: Start with 3-5 core commands, expand gradually*
- **Learning curve**: Team must understand Command pattern - *Mitigation: Comprehensive documentation, pair programming*
- **Undo state management**: Complex commands need careful state capture - *Mitigation: Start with simple stateless commands, add complexity gradually*
- **Over-engineering risk**: Not all input needs commands (UI navigation) - *Mitigation: Use commands for game actions only, not UI hotkeys*
- **Performance overhead**: Command object allocation per input - *Mitigation: Object pooling if needed, unlikely to be bottleneck in turn-based game*

**Effort Estimate**:
- **Time**: 4-6 weeks (1 week design + 1 week core implementation + 2-3 weeks full integration + 1 week advanced features)
- **Complexity**: High (new pattern, significant abstraction, comprehensive testing infrastructure)
- **Risk**: Medium-High (complex refactor but incremental integration possible)
- **Files Impacted**:
  - Create: `tactical/inputcommands/commands.go`, `tactical/inputcommands/executor.go`, `tactical/inputcommands/*_test.go` (~500+ new lines)
  - Modify: All input handlers (~10 files), combat service integration
  - Remove: Old undo/redo implementations, scattered validation logic

**Critical Assessment**:
This approach provides the **most comprehensive solution** for testing, validation, and future features (replay, AI, multiplayer). However, it's **the most complex** and requires the **highest initial investment**. Best suited for teams that:
- Need comprehensive testing (e.g., complex combat rules, frequent bugs)
- Plan advanced features (AI scripting, replay system, multiplayer)
- Have time for significant refactoring (4-6 weeks)
- Are comfortable with design patterns (Command pattern understanding)

**Not recommended if:**
- You need quick wins (use Approach 1 or 2 instead)
- Team is unfamiliar with Command pattern (steep learning curve)
- Simple turn-based game without complex validation needs
- Limited development time (high effort estimate)

This is the **"future-proof"** approach - invest now for long-term flexibility, testability, and advanced features. Consider it for **Phase 2** after consolidating with Approach 1.

---

## COMPARATIVE ANALYSIS OF FINAL APPROACHES

### Effort vs Impact Matrix
| Approach | Effort | Impact | Risk | Recommended Priority |
|----------|--------|--------|------|---------------------|
| **Approach 1: Input Service Layer** | Medium (3-4 weeks) | High (eliminates duplication, consolidates capture) | Low-Medium (incremental migration) | **1** - Start here |
| **Approach 2: Mode-Centric Handlers** | Medium (3-4 weeks) | Medium (better organization, no arch change) | Low (refactoring only) | **2** - Or combine with #1 |
| **Approach 3: Input Command Pattern** | High (4-6 weeks) | High (testability, replay, validation) | Medium-High (complex abstraction) | **3** - Future phase |

### Decision Guidance

**Choose Approach 1 (Input Service Layer) if:**
- You want to **eliminate the dual input system** (InputCoordinator + UIModeManager polling separately)
- **DRY is a priority** (mouse-to-tile conversion repeated 5+ times is painful)
- You need a **foundation for future improvements** without over-engineering today
- Team is comfortable with **incremental migration** (mode-by-mode, validate each step)
- **Quick wins matter** (mouse-to-tile consolidation gives immediate benefit)

**Choose Approach 2 (Mode-Centric Handlers) if:**
- You're **happy with current dual-system architecture** (don't want to touch input capture)
- **Organization is main pain point** (combatmode.go is too big, input logic scattered)
- You want **low-risk refactoring** (just extracting existing logic, no architectural changes)
- Team prefers **clear file separation** (input logic in separate handler files)
- You plan to **use with Approach 1 later** (extract handlers first, then consolidate input service)

**Choose Approach 3 (Input Command Pattern) if:**
- **Testability is critical** (need comprehensive unit tests for input logic)
- You're building **advanced features** (replay system, AI scripting, multiplayer)
- Team is comfortable with **design patterns** (Command pattern familiarity)
- You have **4-6 weeks for refactoring** (significant investment required)
- **Long-term flexibility** is more important than quick wins
- You're willing to **iterate on abstraction** (command design evolves with usage)

### Combination Opportunities

**Recommended Combination: Approach 1 + Approach 2 (Phased)**

**Phase 1 (Weeks 1-4): Input Service Layer (Approach 1)**
- Consolidate input capture in InputService
- Eliminate mouse-to-tile duplication
- Provide clean InputContext API to modes

**Phase 2 (Weeks 5-8): Extract Mode Handlers (Approach 2)**
- Extract CombatModeInputHandler, etc. from mode files
- Use CommonInputHandler for shared patterns
- Modes now delegate to handlers that use InputService

**Result**: Clean architecture with:
- Single input capture point (InputService)
- DRY utilities (mouse-to-tile, hotkey matching)
- Organized input logic (separate handler files)
- Testable handlers (can mock InputContext)
- Clear separation (capture → route → handle)

**Future Enhancement: Add Commands Selectively (Approach 3)**

After Approach 1 + 2 stabilize, add Command pattern for **complex actions only**:
- Use commands for: MoveSquad, AttackSquad, EndTurn (need undo/validation)
- Keep direct methods for: UI navigation, hotkeys, simple toggles
- Best of both worlds: Commands where needed, simple handlers where sufficient

**Example Mixed Pattern:**
```go
// Simple input - direct handling
if inputCtx.IsKeyJustPressed(ebiten.KeyA) {
    cih.battleMapState.InAttackMode = !cih.battleMapState.InAttackMode
    return true
}

// Complex input - command pattern
if inputCtx.IsMouseLeftJustPressed() {
    clickedPos := inputCtx.GetMouseLogicalPos()
    cmd := inputcommands.NewMoveSquadCommand(squadID, clickedPos, combatService)
    return cih.commandExecutor.Execute(cmd) == nil
}
```

---

## APPENDIX: ADDITIONAL ANALYSIS

### A. Current Input Flow (Detailed)

```
┌─────────────────────────────────────────┐
│  Ebiten Game Loop (60 FPS)             │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│  Game.Update()                          │
│  - Polls input from TWO systems         │
└──────┬──────────────────────┬───────────┘
       │                      │
       │ NEW SYSTEM           │ OLD SYSTEM
       ▼                      ▼
┌──────────────────┐   ┌──────────────────┐
│ GameModeCoord.   │   │ InputCoordinator │
│ Update()         │   │ .HandleInput()   │
└────┬─────────────┘   └────┬─────────────┘
     │                      │
     ├─→ UIModeManager      ├─→ UIController (deprecated)
     │   .Update()          ├─→ CombatController (throwables)
     │   ├─→ updateInputState() - POLLS EBITEN
     │   ├─→ currentMode.HandleInput(inputState)
     │   │   └─→ BaseMode.HandleCommonInput()
     │   │       ├─→ ESC key
     │   │       ├─→ Hotkeys (E, I, C)
     │   │       └─→ CommandHistory.HandleInput()
     │   │           └─→ Ctrl+Z, Ctrl+Y (DUPLICATION)
     │   │
     │   ├─→ CombatMode.HandleInput()
     │   │   ├─→ HandleCommonInput() (above)
     │   │   ├─→ CombatInputHandler.HandleInput()
     │   │   │   ├─→ Mouse clicks (move/attack)
     │   │   │   ├─→ A/M/Tab keys
     │   │   │   └─→ Ctrl+Z/Y (DUPLICATION)
     │   │   ├─→ Space key (end turn)
     │   │   └─→ H key (danger viz)
     │   │
     │   └─→ ExplorationMode.HandleInput()
     │       └─→ HandleCommonInput() only
     │
     └─→ MovementController.HandleInput()
         ├─→ POLLS EBITEN (IsKeyJustReleased)
         ├─→ WASD movement
         ├─→ G pickup
         └─→ L/M debug keys

ISSUES:
1. Two separate raw input polls (updateInputState + IsKeyJustReleased)
2. Ctrl+Z/Y handled in CommandHistory AND combat_input_handler
3. Mouse-to-tile conversion in every mode handler (5+ locations)
4. Unclear priority (does old system run if new system consumes input?)
```

### B. Proposed Input Flow (Approach 1: Input Service Layer)

```
┌─────────────────────────────────────────┐
│  Ebiten Game Loop (60 FPS)             │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│  Game.Update()                          │
│  1. inputService.CaptureInput(playerPos)│
│     - Polls Ebiten ONCE                 │
│     - Computes mouse-to-tile ONCE       │
│  2. inputCtx = inputService.GetContext()│
│  3. modeCoordinator.HandleInput(inputCtx)│
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│  GameModeCoordinator.HandleInput(inputCtx)│
│  - Routes to active mode manager        │
└──────┬──────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│  UIModeManager.HandleInput(inputCtx)    │
│  - Passes InputContext to current mode  │
└──────┬──────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│  CombatMode.HandleInput(inputCtx)       │
│  1. commonInputHandler.Handle(inputCtx) │
│     ├─→ ESC key                         │
│     ├─→ Hotkeys                         │
│     └─→ Ctrl+Z/Y (SINGLE LOCATION)      │
│  2. modeInputHandler.Handle(inputCtx)   │
│     ├─→ Mouse (uses inputCtx.GetMouseLogicalPos())
│     ├─→ Keyboard (A/M/Tab/Space/H)      │
│     └─→ No raw Ebiten calls             │
└─────────────────────────────────────────┘

IMPROVEMENTS:
✅ Single input capture point
✅ Mouse-to-tile computed once, cached in InputFrame
✅ Ctrl+Z/Y in one location (CommonInputHandler)
✅ Clear input pipeline (Capture → Context → Route → Handle)
✅ No duplicate polling
✅ Easy to test (mock InputContext)
```

### C. State Consolidation Opportunity

**Current State (Scattered):**
```go
// input/inputcoordinator.go
type SharedInputState struct {
    PrevCursor         coords.PixelPosition
    PrevThrowInds      []int
    PrevRangedAttInds  []int
    PrevTargetLineInds []int
    TurnTaken          bool
}

// gui/core/uimode.go
type InputState struct {
    MouseX            int
    MouseY            int
    MousePressed      bool
    MouseReleased     bool
    MouseButton       ebiten.MouseButton
    KeysPressed       map[ebiten.Key]bool
    KeysJustPressed   map[ebiten.Key]bool
    PlayerInputStates *common.PlayerInputStates
}

// common/playerdata.go
type PlayerInputStates struct {
    IsThrowing    bool
    HasKeyInput   bool
    // ... other flags
}

// gui/core/gamemodecoordinator.go
type BattleMapState struct {
    SelectedSquadID   ecs.EntityID
    SelectedTargetID  ecs.EntityID
    InMoveMode        bool
    InAttackMode      bool
    // ... UI state mixed with game state
}
```

**Proposed Consolidation (Approach 1):**
```go
// input/inputservice.go
type InputFrame struct {
    // Raw input (captured from Ebiten)
    MouseX      int
    MouseY      int
    MouseLeft   bool
    MouseRight  bool
    KeysPressed map[ebiten.Key]bool

    // Derived state (computed once per frame)
    MouseLogicalPos coords.LogicalPosition
    MousePixelPos   coords.PixelPosition
}

// Mode-specific state stays in modes (not input system)
// BattleMapState keeps UI state (SelectedSquadID, InMoveMode) - that's correct
// Game state lives in ECS components - that's correct
// Input state lives in InputService - centralized
```

### D. Testing Strategy Comparison

**Current (Hard to Test):**
```go
// Can't test combat input without creating full CombatMode + UI
func TestCombatInput(t *testing.T) {
    // Need: ECSManager, PlayerData, GameMap, CombatService,
    //       UIModeManager, BattleMapState, all UI widgets...
    // TOO COMPLEX - not feasible
}
```

**Approach 1 (Testable Handlers):**
```go
// Can test input handler with mock InputContext
func TestCombatInputHandler_AttackMode(t *testing.T) {
    // Setup
    handler := NewCombatInputHandler(mockCombatService, mockBattleMapState)
    mockInput := &input.InputContext{
        // Mock input state
    }

    // Test
    handled := handler.HandleInput(mockInput)

    // Verify
    assert.True(t, handled)
    assert.True(t, mockBattleMapState.InAttackMode)
}
```

**Approach 3 (Comprehensive Testing):**
```go
// Can test commands in complete isolation
func TestMoveSquadCommand_Validate(t *testing.T) {
    cmd := inputcommands.NewMoveSquadCommand(squadID, targetPos, mockCombatService)
    err := cmd.Validate()
    assert.NoError(t, err)
}

func TestMoveSquadCommand_Execute(t *testing.T) {
    cmd := inputcommands.NewMoveSquadCommand(squadID, targetPos, mockCombatService)
    err := cmd.Execute()
    assert.NoError(t, err)
    // Verify squad moved
}

func TestCommandExecutor_UndoRedo(t *testing.T) {
    executor := inputcommands.NewCommandExecutor()
    // Test undo/redo flow
}
```

### E. Mouse-to-Tile Duplication (Current Problem)

**Currently repeated in 5+ locations:**

1. `gui/guicombat/combat_input_handler.go:123`
```go
clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)
```

2. `gui/guicombat/combat_input_handler.go:159`
```go
clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *cih.playerPos)
```

3. Every other mode that handles mouse input
4. Old input system if it handled mouse (currently doesn't)

**After Approach 1 (Computed Once):**
```go
// In InputService.CaptureInput() - ONCE per frame
is.currentFrame.MouseLogicalPos = graphics.MouseToLogicalPosition(
    is.currentFrame.MouseX,
    is.currentFrame.MouseY,
    playerPos,
)

// In handlers - NO conversion needed
clickedPos := inputCtx.GetMouseLogicalPos() // Already computed!
```

**Savings:**
- ~8 lines of code eliminated per mouse handler
- ~5-10 mouse-to-tile conversions per frame → 1 conversion per frame
- Performance: Negligible but cleaner (conversion isn't expensive)
- Maintainability: Change conversion algorithm once, not 5+ times

---

## SYNTHESIS RATIONALE

### Why These 3 Final Approaches?

**Approach 1 Selection (Input Service Layer)**:
Chosen because it **directly addresses the dual input system problem** (InputCoordinator + UIModeManager polling separately), which is the root architectural issue. It provides:
- Clear consolidation path (transform InputCoordinator → InputService)
- Immediate value (DRY mouse-to-tile conversion)
- Foundation for future improvements (testability, commands)
- Incremental migration (low risk)
- Aligns with your preference for "transform old InputCoordinator" (Requirement #2)

Combines best elements from pragmatic refactoring (consolidate duplication) and architectural thinking (clear layers).

**Approach 2 Selection (Mode-Centric Handlers)**:
Chosen because it respects **pragmatic separation of concerns** (Requirement #3) without dogmatic layer enforcement. It provides:
- Better organization without architectural changes (low risk)
- Extracted handlers for testability (but not over-abstracted)
- CommonInputHandler eliminates Ctrl+Z/Y duplication
- Can be combined with Approach 1 (phased implementation)
- Aligns with your preference for "better organized but not dogmatic" (Requirement #3)

Addresses maintainability pain points (large mode files, scattered logic) without touching input capture architecture.

**Approach 3 Selection (Input Command Pattern)**:
Chosen because it represents **comprehensive testability solution** (Requirement #4 "testable but don't over-engineer"). It provides:
- Command pattern for validation + undo + replay
- Unit testable input logic (comprehensive coverage)
- Future features (AI scripting, multiplayer, replay)
- Clear validation abstractions (explicit preconditions)
- Higher complexity but significant long-term value

Represents "future-proof" investment for teams that need advanced features and comprehensive testing.

### Rejected Elements

**Not Included from Initial Analysis:**

1. **Big-bang rewrite** - Considered full rewrite of both systems from scratch
   - **Why rejected**: Too risky (breaks existing gameplay), doesn't align with "incremental transition" (Requirement #1)
   - **Alternative**: Approach 1 provides incremental consolidation instead

2. **Strict layer separation** - Considered input → routing → handling → execution as distinct layers
   - **Why rejected**: Over-engineering for current needs, violates "pragmatic embedding" (Requirement #3)
   - **Alternative**: Approach 2 provides organized handlers without strict layers

3. **Event-driven architecture** - Considered event bus for input events (Publish/Subscribe)
   - **Why rejected**: Adds complexity without clear benefit, harder to debug than direct calls
   - **Alternative**: Direct method calls with InputContext (Approach 1) is simpler

4. **Full Command pattern for ALL input** - Considered commands for UI navigation, hotkeys, etc.
   - **Why rejected**: Over-abstraction, simple input doesn't need commands (violates "don't over-engineer")
   - **Alternative**: Approach 3 uses commands selectively (complex actions only)

5. **Input recording without commands** - Considered serializing raw input frames
   - **Why rejected**: Low-level approach (frame-by-frame) is fragile for replay
   - **Alternative**: Approach 3 command serialization is semantic (action-based replay)

### Key Insights from Multi-Perspective Analysis

**From Architectural Perspective:**
- Dual input systems (InputCoordinator + UIModeManager) are the root problem
- Input capture should happen once per frame in single location
- Clear separation: Capture (InputService) → Route (ModeManager) → Handle (ModeHandlers)

**From Game Development Perspective:**
- Turn-based tactical game doesn't need frame-perfect input (unlike real-time action game)
- Undo/redo is critical for tactical gameplay (Command pattern natural fit)
- Mode-specific input patterns (combat vs exploration) justify mode-specific handlers

**From Pragmatic Perspective:**
- Incremental migration is essential (can't break existing gameplay)
- Quick wins matter (mouse-to-tile DRY gives immediate value)
- Don't over-engineer (not all input needs commands)
- Testability important but shouldn't add significant complexity

**Consensus Points:**
- All perspectives agree: Consolidate input capture (Approach 1)
- All perspectives agree: DRY mouse-to-tile conversion (Approach 1)
- All perspectives agree: Incremental migration over big-bang (Approach 1 + 2 phased)

**Divergence Points:**
- Commands vs direct methods (Approach 3 vs 1/2)
- When to separate handlers (Approach 2 immediate vs combine with Approach 1)
- Testing infrastructure investment (Approach 3 comprehensive vs Approach 1 basic)

---

## PRINCIPLES APPLIED

### Software Engineering Principles

**DRY (Don't Repeat Yourself)**:
- **Approach 1**: Mouse-to-tile conversion computed once in InputService, cached in InputFrame
- **Approach 2**: CommonInputHandler centralizes Ctrl+Z/Y, ESC, hotkey handling
- **Approach 3**: Validation logic centralized in Command.Validate() methods
- **Before**: Mouse-to-tile in 5+ locations, Ctrl+Z/Y in 2 locations

**SOLID Principles**:
- **Single Responsibility Principle (SRP)**:
  - Approach 1: InputService responsible for capture only, modes handle routing
  - Approach 2: ModeInputHandlers handle input, modes handle UI rendering
  - Approach 3: Commands handle single action (MoveSquad, Attack, etc.)

- **Open/Closed Principle (OCP)**:
  - Approach 1: InputService open to new query methods (GetGamepadInput), closed to modification of capture logic
  - Approach 3: CommandExecutor open to new command types, closed to modification of undo/redo logic

- **Dependency Inversion Principle (DIP)**:
  - Approach 1: Modes depend on InputContext interface, not concrete InputService
  - Approach 3: Handlers depend on InputCommand interface, not concrete command implementations

**KISS (Keep It Simple, Stupid)**:
- **Approach 1**: Simple InputService (capture + query methods), no over-abstraction
- **Approach 2**: Simple handler extraction, no complex patterns
- **Approach 3**: Commands only for complex actions (not UI hotkeys)

**YAGNI (You Aren't Gonna Need It)**:
- **Approach 1**: Doesn't include gamepad support yet (add when needed)
- **Approach 2**: Doesn't extract handlers for simple modes (ExplorationMode stays embedded)
- **Approach 3**: Doesn't implement command serialization until replay system needed

**SLAP (Single Level of Abstraction Principle)**:
- **Approach 1**: HandleInput() methods at same level (check key → call action), not mixed with Ebiten polling
- **Approach 2**: Handler methods organized by category (combat actions, turn management, visualization)
- **Approach 3**: Command.Execute() at game logic level, not mixed with input checking

**Separation of Concerns (SOC)**:
- **Approach 1**: Input capture (InputService) separate from input handling (modes)
- **Approach 2**: Input handling (handlers) separate from UI rendering (modes)
- **Approach 3**: Validation (Validate) separate from execution (Execute) separate from reversal (Undo)

### Go-Specific Best Practices

**Composition over inheritance**:
- **Approach 2**: BaseMode uses CommonInputHandler as component, not inheritance
- **Approach 3**: Commands implement interface, no base class

**Interface design**:
- **Approach 1**: InputContext provides minimal read-only interface to handlers
- **Approach 3**: InputCommand interface with Validate/Execute/Undo methods

**Error handling**:
- **Approach 1**: InputService methods return InputContext (no errors - capture always succeeds)
- **Approach 3**: Commands return explicit errors for validation and execution failures

**Value types where appropriate**:
- **Approach 1**: InputFrame is struct (not pointer), passed by value for safety
- **Current**: coords.LogicalPosition already value type (good)

**Clear package boundaries**:
- **Approach 1**: `input/` package owns capture, `gui/` package owns routing/handling
- **Approach 3**: `tactical/inputcommands/` package owns commands, `gui/` uses them

### Game Development Considerations

**Turn-based constraints**:
- **Approach 1**: No input buffering needed (unlike real-time), simple frame capture sufficient
- **Approach 3**: Commands natural fit for turn-based (discrete actions, undo/redo)

**Tactical gameplay**:
- **Approach 2**: Mode-specific handlers respect different input patterns (combat vs exploration)
- **Approach 3**: Validation in commands ensures tactical rules enforced (can squad move? can attack?)

**Player experience**:
- **All approaches**: Maintain undo/redo for tactical games (critical feature)
- **Approach 3**: Command history enables "show me what happened" debugging for players

**Performance** (not critical for turn-based but considered):
- **Approach 1**: Single input capture faster than dual polling (negligible difference but cleaner)
- **Approach 3**: Command allocation overhead (negligible for turn-based, could pool if needed)

---

## NEXT STEPS

### Recommended Action Plan

**Immediate (This Week)**:
1. **Document current input flow** (use diagram in Appendix A)
   - Share with team for discussion
   - Identify any additional pain points

2. **Choose approach** based on decision guidance above
   - Quick wins + foundation: **Approach 1**
   - Organization only: **Approach 2**
   - Future-proof: **Approach 3**
   - Recommended: **Approach 1 → Approach 2 (phased)**

3. **Create proof-of-concept**
   - Approach 1: Implement InputService, migrate combat mode only
   - Approach 2: Extract CombatModeInputHandler
   - Approach 3: Implement MoveSquadCommand + CommandExecutor

**Short-term (Next 2-4 Weeks)**:
1. **Implement chosen approach** following implementation strategy
2. **Migrate incrementally** mode-by-mode with thorough testing
3. **Document patterns** in CLAUDE.md for team reference
4. **Monitor for issues** during migration (rollback points at each mode)

**Medium-term (1-3 Months)**:
1. **Stabilize architecture** - all modes using new system
2. **Deprecate old code** - remove InputCoordinator or transform to InputService
3. **Add tests** - unit tests for handlers or commands
4. **Optimize** - profile if needed, address any performance issues

**Long-term (3-6 Months)**:
1. **Evaluate advanced features**
   - Approach 1 + 2: Add commands selectively (Approach 3 integration)
   - Replay system (if Approach 3 chosen)
   - AI command generation (if Approach 3 chosen)

2. **Refine based on usage**
   - Adjust InputService API based on mode needs
   - Simplify command structure if over-abstracted
   - Add utilities as patterns emerge

### Validation Strategy

**Testing Approach**:
- **Unit tests**: Commands (Approach 3), input handlers (Approach 1/2)
- **Integration tests**: Full input flow (mouse click → command execution → ECS update)
- **Manual testing**: Each mode after migration, all input combinations
- **Regression testing**: Save input sequences, replay to verify behavior unchanged

**Rollback Plan**:
- **Approach 1**: Feature flag per mode (old vs new input path), can revert individual modes
- **Approach 2**: Keep old HandleInput() methods during transition, delete after validation
- **Approach 3**: Commands optional, can fall back to direct execution if issues

**Success Metrics**:
- **Code quality**: Lines of duplicated code (mouse-to-tile, Ctrl+Z/Y) reduced to 0
- **Maintainability**: Time to add new input feature (should decrease by 30-50%)
- **Testability**: Code coverage for input handling (target 60-80% for critical paths)
- **Stability**: No input-related bugs in production for 2+ weeks after migration

### Additional Resources

**Go Patterns**:
- [Effective Go](https://go.dev/doc/effective_go) - Interfaces, composition
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) - Style guide

**Game Architecture**:
- [Game Programming Patterns](https://gameprogrammingpatterns.com/) - Command pattern, Update method
- [Entity Component System FAQ](https://github.com/SanderMertens/ecs-faq) - ECS best practices

**Input System Design**:
- [Input System Best Practices (Unity)](https://docs.unity3d.com/Packages/com.unity.inputsystem@1.0/manual/index.html) - Concepts applicable to any engine
- [Ebiten Input Examples](https://ebiten.org/en/examples/) - Ebiten-specific patterns

**Refactoring**:
- [Refactoring: Improving the Design of Existing Code](https://martinfowler.com/books/refactoring.html) - Incremental refactoring techniques
- [Working Effectively with Legacy Code](https://www.oreilly.com/library/view/working-effectively-with/0131177052/) - Migration strategies

---

## FINAL RECOMMENDATION

Based on your requirements (incremental migration, transform InputCoordinator, pragmatic separation, testability, maintainability focus), I recommend:

### **Phase 1: Approach 1 (Input Service Layer) - Weeks 1-4**

Start with Input Service Layer to consolidate input capture and eliminate duplication. This gives immediate value (DRY mouse-to-tile) while establishing foundation for further improvements.

**Why Start Here:**
- Solves biggest architectural issue (dual input systems)
- Quick wins (mouse-to-tile DRY) in Week 1
- Transforms InputCoordinator as desired
- Low risk (incremental mode-by-mode migration)
- Foundation for Phases 2-3

### **Phase 2: Approach 2 (Extract Handlers) - Weeks 5-8**

After InputService stabilizes, extract mode input handlers for better organization. This leverages InputService (handlers use InputContext) while improving code organization.

**Why Second:**
- Builds on InputService foundation
- Further improves maintainability
- Makes testing easier (handlers testable independently)
- Pragmatic separation (organized but not dogmatic)

### **Phase 3 (Optional): Approach 3 (Commands) - Months 3-6**

If comprehensive testing or advanced features (replay, AI) become priorities, selectively introduce commands for complex actions.

**Why Optional/Later:**
- Significant investment (4-6 weeks)
- Evaluate need after Phases 1-2 stabilize
- Can introduce gradually (MoveSquad command first, expand if valuable)
- Not needed for immediate pain points

### **Expected Outcome (After Phases 1-2)**:

✅ **Single input capture point** - InputService polls Ebiten once per frame
✅ **DRY utilities** - Mouse-to-tile computed once, Ctrl+Z/Y in one location
✅ **Clear pipeline** - Capture (InputService) → Route (UIModeManager) → Handle (ModeHandlers)
✅ **Better organization** - Input logic in dedicated handler files, modes focus on UI
✅ **Testable** - Can unit test handlers with mock InputContext
✅ **Maintainable** - Easy to add new input features, clear where to look for logic
✅ **Low risk** - Incremental migration with validation at each step

**Estimated Total Effort**: 6-8 weeks for Phases 1-2, with usable improvements every 1-2 weeks.

**First concrete step**: Create `input/inputservice.go` with InputService and InputFrame structs, implement CaptureInput() and basic query methods. Migrate combat mode to prove concept. (Week 1 goal)

---

END OF ANALYSIS
