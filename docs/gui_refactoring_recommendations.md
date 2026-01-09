# GUI/Gameplay Decoupling Recommendations

**Document Version:** 1.0
**Date:** 2026-01-09
**Status:** Analysis Complete

---

## Executive Summary

The `gui/` package currently has significant coupling to gameplay logic:

| Issue | Severity | Impact |
|-------|----------|--------|
| Game lifecycle controlled by UI mode transitions | CRITICAL | Cannot test/run combat without UI |
| `BattleMapState` mixes UI selection with game rules | CRITICAL | Attack/move validation tied to UI flags |
| Services created and owned by UI modes | HIGH | Cannot reuse services outside UI |
| Direct `EntityManager` access from UI | HIGH | UI can modify any ECS state |
| Entity creation/disposal in UI callbacks | HIGH | Lifecycle scattered across UI code |

**Goal:** Separate "what the player sees" (UI) from "what the game does" (logic) so that:
- Game logic can run independently (headless, testing, replays)
- UI becomes a thin presentation layer
- Changes to UI don't break game rules
- Changes to game rules don't require UI modifications

---

## Table of Contents

1. [Architectural Patterns](#1-architectural-patterns)
2. [Proposed Package Boundaries](#2-proposed-package-boundaries)
3. [Communication Mechanisms](#3-communication-mechanisms)
4. [Incremental Refactoring Steps](#4-incremental-refactoring-steps)
5. [Trade-offs Analysis](#5-trade-offs-analysis)
6. [Reference Implementations](#6-reference-implementations)

---

## 1. Architectural Patterns

### 1.1 Game Session Manager (Recommended Primary Pattern)

Create a central `GameSession` that owns all game state and services, with UI modes receiving read-only views.

```
┌─────────────────────────────────────────────────────────────┐
│                      GameSession                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐    │
│  │CombatManager│ │SquadManager │ │InventoryManager     │    │
│  └─────────────┘ └─────────────┘ └─────────────────────┘    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                    EventBus                          │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              │
                    Publishes Events / Exposes Read Views
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        GUI Layer                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐    │
│  │ CombatMode  │ │ SquadMode   │ │  InventoryMode      │    │
│  │ (presenter) │ │ (presenter) │ │  (presenter)        │    │
│  └─────────────┘ └─────────────┘ └─────────────────────┘    │
│                                                              │
│  UI subscribes to events, sends commands, reads views        │
└─────────────────────────────────────────────────────────────┘
```

**Key Principles:**
- GameSession owns services (CombatService, SquadBuilderService, etc.)
- UI modes receive interfaces, not concrete managers
- UI sends commands; GameSession executes them
- GameSession publishes events; UI subscribes and updates display

### 1.2 Command Pattern for Actions

All game-mutating actions go through commands:

```go
// game/commands/commands.go
type Command interface {
    Execute() error
    Undo() error
    Description() string
}

type CommandDispatcher struct {
    handlers map[reflect.Type]CommandHandler
}

// Example commands
type AttackCommand struct {
    AttackerID ecs.EntityID
    TargetID   ecs.EntityID
}

type MoveCommand struct {
    UnitID      ecs.EntityID
    Destination coords.LogicalPosition
}

type EndTurnCommand struct {
    FactionID ecs.EntityID
}
```

**Benefits:**
- Undo/redo built-in (you already have `squadcommands.CommandExecutor`)
- Commands can be logged, replayed, serialized
- UI only constructs commands, never executes logic directly

### 1.3 Event Bus for Notifications

Game state changes emit events; UI subscribes:

```go
// game/events/events.go
type EventType string

const (
    EventCombatStarted    EventType = "combat.started"
    EventCombatEnded      EventType = "combat.ended"
    EventTurnChanged      EventType = "combat.turn.changed"
    EventUnitMoved        EventType = "unit.moved"
    EventUnitAttacked     EventType = "unit.attacked"
    EventUnitDied         EventType = "unit.died"
    EventSquadCreated     EventType = "squad.created"
    EventSquadModified    EventType = "squad.modified"
)

type Event struct {
    Type    EventType
    Payload interface{}
}

type EventBus struct {
    subscribers map[EventType][]func(Event)
    mu          sync.RWMutex
}

func (eb *EventBus) Subscribe(eventType EventType, handler func(Event)) {}
func (eb *EventBus) Publish(event Event) {}
```

### 1.4 Read-Only Views (Query Interfaces)

UI receives interfaces that only expose read operations:

```go
// game/views/combatview.go
type CombatView interface {
    GetCurrentFaction() ecs.EntityID
    GetFactionSquads(factionID ecs.EntityID) []SquadInfo
    GetValidMoveTargets(unitID ecs.EntityID) []coords.LogicalPosition
    GetValidAttackTargets(unitID ecs.EntityID) []ecs.EntityID
    IsPlayerTurn() bool
    GetCombatLog() []string
}

// game/views/squadview.go
type SquadView interface {
    GetSquad(squadID ecs.EntityID) *SquadInfo
    GetSquadMembers(squadID ecs.EntityID) []UnitInfo
    GetPlayerSquads() []ecs.EntityID
}
```

**Key:** Views are computed from current game state. UI cannot cache stale views.

---

## 2. Proposed Package Boundaries

### 2.1 New Package Structure

```
game_main/
├── game/                          # NEW: Core game logic layer
│   ├── session/                   # GameSession, lifecycle management
│   │   ├── session.go            # Main GameSession struct
│   │   ├── combatmanager.go      # Combat lifecycle (extracted from UI)
│   │   └── sessionconfig.go      # Configuration
│   │
│   ├── commands/                  # Command pattern implementation
│   │   ├── command.go            # Command interface
│   │   ├── dispatcher.go         # Command dispatcher
│   │   ├── combat_commands.go    # Attack, Move, EndTurn
│   │   └── squad_commands.go     # Create, Modify, Deploy
│   │
│   ├── events/                    # Event bus and event types
│   │   ├── bus.go                # EventBus implementation
│   │   └── types.go              # Event type definitions
│   │
│   └── views/                     # Read-only view interfaces
│       ├── combatview.go         # Combat state queries
│       ├── squadview.go          # Squad state queries
│       └── worldview.go          # World/map state queries
│
├── gui/                           # MODIFIED: Presentation only
│   ├── framework/                 # Keep: Mode management, context
│   │   ├── coordinator.go        # Mode switching
│   │   ├── uimode.go            # MODIFIED: Receives views, not managers
│   │   └── uistate.go           # NEW: Pure UI state (selections only)
│   │
│   ├── guicombat/                # MODIFIED: Uses views and commands
│   │   ├── combatmode.go        # Subscribes to events, sends commands
│   │   ├── combatpresenter.go   # NEW: Transforms views to UI widgets
│   │   └── combatanimation.go   # Keep: Animation logic
│   │
│   ├── guisquads/                # MODIFIED: Uses views and commands
│   │   ├── squadbuilder.go      # Sends commands, reads views
│   │   └── squadpresenter.go    # NEW: View -> widget transformation
│   │
│   ├── builders/                 # Keep: EbitenUI widget builders
│   ├── widgets/                  # Keep: Custom widgets
│   └── specs/                    # Keep: Layout constants
│
├── tactical/                      # EXISTING: Game mechanics (keep)
│   ├── combat/                   # Combat calculations
│   ├── combatservices/           # Combat service layer
│   ├── squads/                   # Squad ECS components
│   ├── squadservices/            # Squad services
│   └── squadcommands/            # MOVE to game/commands/
```

### 2.2 Import Direction Rules

```
ALLOWED IMPORTS:
─────────────────
gui/         → game/views (read-only interfaces)
gui/         → game/commands (to construct commands)
gui/         → game/events (to subscribe)
game/        → tactical/ (business logic)
game/        → common/ (ECS utilities)
tactical/    → common/ (ECS utilities)

FORBIDDEN IMPORTS:
──────────────────
game/        → gui/ (NO: game must not know about UI)
tactical/    → gui/ (NO: business logic must not know about UI)
gui/         → common/EntityManager (NO: use views instead)
gui/         → tactical/ directly (NO: go through game/ layer)
```

### 2.3 Dependency Injection

```go
// game/session/session.go
type GameSession struct {
    ecsManager      *common.EntityManager  // Owned by session
    combatManager   *CombatManager
    squadManager    *SquadManager
    eventBus        *events.EventBus
    commandDispatcher *commands.Dispatcher
}

func NewGameSession(config SessionConfig) *GameSession {
    session := &GameSession{
        ecsManager: common.NewEntityManager(),
        eventBus:   events.NewEventBus(),
    }
    session.combatManager = NewCombatManager(session.ecsManager, session.eventBus)
    session.squadManager = NewSquadManager(session.ecsManager, session.eventBus)
    session.commandDispatcher = commands.NewDispatcher(session)
    return session
}

// Views are created on demand, not stored
func (gs *GameSession) CombatView() views.CombatView {
    return gs.combatManager.CreateView()
}

func (gs *GameSession) SquadView() views.SquadView {
    return gs.squadManager.CreateView()
}

func (gs *GameSession) EventBus() *events.EventBus {
    return gs.eventBus
}

func (gs *GameSession) Dispatch(cmd commands.Command) error {
    return gs.commandDispatcher.Dispatch(cmd)
}
```

---

## 3. Communication Mechanisms

### 3.1 UI → Game: Commands

**Before (current - problematic):**
```go
// gui/guicombat/combat_action_handler.go
func (cah *CombatActionHandler) ExecuteAttack() {
    // UI directly calls combat service
    result, err := cah.combatService.ExecuteAttack(
        cah.battleMapState.SelectedSquadID,
        cah.battleMapState.SelectedTargetID,
    )
    // UI directly updates state
    cah.battleMapState.InAttackMode = false
}
```

**After (recommended):**
```go
// gui/guicombat/combatmode.go
func (cm *CombatMode) handleAttackClick() {
    // UI constructs command from UI state
    cmd := commands.NewAttackCommand(
        cm.uiState.SelectedSquadID,    // UI selection
        cm.uiState.SelectedTargetID,   // UI selection
    )

    // Dispatch to game session - UI doesn't know how it's executed
    if err := cm.gameSession.Dispatch(cmd); err != nil {
        cm.showError(err)
    }

    // Clear UI selection state (UI-only concern)
    cm.uiState.ClearAttackSelection()
}
```

### 3.2 Game → UI: Events

**Before (current - problematic):**
```go
// gui/guicombat/combatmode.go
func (cm *CombatMode) executePlayerAttack() {
    result := cm.combatService.ExecuteAttack(...)

    // UI directly handles result
    cm.logManager.LogAttackResult(result)
    cm.animation.PlayAttackAnimation(result)

    // UI checks if combat should end
    if cm.combatService.IsCombatOver() {
        cm.handleCombatEnd()
    }
}
```

**After (recommended):**
```go
// game/session/combatmanager.go
func (cm *CombatManager) ExecuteAttack(cmd *commands.AttackCommand) error {
    result := cm.combatService.ExecuteAttack(cmd.AttackerID, cmd.TargetID)

    // Game publishes event - doesn't know about UI
    cm.eventBus.Publish(events.Event{
        Type: events.EventUnitAttacked,
        Payload: &events.AttackPayload{
            AttackerID: cmd.AttackerID,
            TargetID:   cmd.TargetID,
            Damage:     result.Damage,
            IsCritical: result.IsCritical,
        },
    })

    if cm.combatService.IsCombatOver() {
        cm.eventBus.Publish(events.Event{
            Type: events.EventCombatEnded,
            Payload: &events.CombatEndPayload{
                Victor: cm.getVictor(),
            },
        })
    }
    return nil
}

// gui/guicombat/combatmode.go
func (cm *CombatMode) Initialize(ctx *framework.UIContext) error {
    // Subscribe to events
    cm.gameSession.EventBus().Subscribe(events.EventUnitAttacked, cm.onUnitAttacked)
    cm.gameSession.EventBus().Subscribe(events.EventCombatEnded, cm.onCombatEnded)
}

func (cm *CombatMode) onUnitAttacked(event events.Event) {
    payload := event.Payload.(*events.AttackPayload)

    // UI updates display
    cm.logArea.AppendLine(formatAttackMessage(payload))
    cm.animation.QueueAttackAnimation(payload)
}

func (cm *CombatMode) onCombatEnded(event events.Event) {
    payload := event.Payload.(*events.CombatEndPayload)
    cm.showVictoryScreen(payload.Victor)
}
```

### 3.3 UI Reads State: Views

**Before (current - problematic):**
```go
// gui/guicombat/combatmode.go
func (cm *CombatMode) getValidMoveTiles() []coords.LogicalPosition {
    battleState := cm.Context.ModeCoordinator.GetBattleMapState()

    // UI checks game rule flag stored in UI state
    if !battleState.InMoveMode {
        return []coords.LogicalPosition{}
    }

    // UI directly accesses movement system
    return cm.combatService.MovementSystem.GetValidMovementTiles(
        cm.combatService.CombatStateManager,
        battleState.SelectedSquadID,
    )
}
```

**After (recommended):**
```go
// game/views/combatview.go (interface)
type CombatView interface {
    GetValidMoveTargets(unitID ecs.EntityID) []coords.LogicalPosition
    CanUnitMove(unitID ecs.EntityID) bool
    // ... other read-only queries
}

// game/session/combatview_impl.go (implementation)
type combatViewImpl struct {
    combatManager *CombatManager
}

func (v *combatViewImpl) GetValidMoveTargets(unitID ecs.EntityID) []coords.LogicalPosition {
    if !v.combatManager.CanUnitMove(unitID) {
        return nil
    }
    return v.combatManager.movementSystem.GetValidMovementTiles(unitID)
}

func (v *combatViewImpl) CanUnitMove(unitID ecs.EntityID) bool {
    return v.combatManager.combatState.CanUnitAct(unitID)
}

// gui/guicombat/combatmode.go
func (cm *CombatMode) renderMovementOverlay(screen *ebiten.Image) {
    // UI asks view if move is valid (game rule check in game layer)
    if !cm.combatView.CanUnitMove(cm.uiState.SelectedUnitID) {
        return
    }

    // UI gets valid tiles from view
    validTiles := cm.combatView.GetValidMoveTargets(cm.uiState.SelectedUnitID)
    cm.movementRenderer.Render(screen, validTiles)
}
```

### 3.4 UI State vs Game State Separation

**New UI State structure (pure presentation):**
```go
// gui/framework/uistate.go
type UIState struct {
    // Selection state (what the player is pointing at)
    HoveredTile       *coords.LogicalPosition
    SelectedSquadID   ecs.EntityID
    SelectedTargetID  ecs.EntityID

    // UI mode flags (what UI mode is active)
    ShowingMoveOverlay   bool
    ShowingAttackOverlay bool
    ShowingInventory     bool

    // Animation state
    AnimationInProgress bool

    // Transient display state
    TooltipText string
    ErrorMessage string
}

// Methods only affect UI, never game state
func (us *UIState) SelectSquad(squadID ecs.EntityID) {
    us.SelectedSquadID = squadID
    us.SelectedTargetID = 0
}

func (us *UIState) EnterMoveMode() {
    us.ShowingMoveOverlay = true
    us.ShowingAttackOverlay = false
}

func (us *UIState) ClearSelection() {
    us.SelectedSquadID = 0
    us.SelectedTargetID = 0
    us.ShowingMoveOverlay = false
    us.ShowingAttackOverlay = false
}
```

**Contrast with game state (in game layer):**
```go
// game/session/combatstate.go
type CombatState struct {
    // Turn tracking (game rule)
    CurrentFactionID ecs.EntityID
    TurnNumber       int

    // Unit action tracking (game rule)
    UnitsActedThisTurn map[ecs.EntityID]bool

    // Combat status (game rule)
    IsActive bool
    Victor   ecs.EntityID
}
```

---

## 4. Incremental Refactoring Steps

### Phase 1: Extract Event Bus (Low Risk)

**Goal:** Introduce event-based communication without changing structure.

**Steps:**
1. Create `game/events/bus.go` with simple publish/subscribe
2. Add EventBus to existing UIContext
3. Modify one high-frequency operation to emit events (e.g., unit movement)
4. Have CombatMode subscribe and update display from event
5. Keep existing direct calls as fallback

**Files to modify:**
- New: `game/events/bus.go`, `game/events/types.go`
- Modify: `gui/framework/uimode.go` (add EventBus to context)
- Modify: `gui/guicombat/combatmode.go` (subscribe to events)
- Modify: `tactical/combatservices/movementsystem.go` (emit events)

**Validation:** Game still works identically; events are bonus notifications.

### Phase 2: Extract UI State from BattleMapState (Medium Risk)

**Goal:** Separate pure UI selection state from game rule flags.

**Steps:**
1. Create `gui/framework/uistate.go` with selection-only state
2. Move `InAttackMode`, `InMoveMode` logic to CombatView queries
3. Update CombatMode to use new UIState for selections
4. Update CombatMode to query CombatView for "can attack/move"
5. Deprecate `BattleMapState.InAttackMode/InMoveMode`

**Before:**
```go
if battleState.InAttackMode {
    ExecuteAttack()
}
```

**After:**
```go
if uiState.ShowingAttackOverlay && combatView.CanUnitAttack(uiState.SelectedSquadID) {
    session.Dispatch(commands.NewAttackCommand(...))
}
```

**Files to modify:**
- New: `gui/framework/uistate.go`
- Modify: `gui/framework/contextstate.go` (deprecate mixed fields)
- Modify: `gui/guicombat/combatmode.go`
- Modify: `gui/guicombat/combat_action_handler.go`

### Phase 3: Extract Combat Lifecycle to Game Layer (High Impact)

**Goal:** Move combat setup/teardown out of UI mode transitions.

**Steps:**
1. Create `game/session/combatmanager.go`
2. Move `combatlifecycle.go` logic to CombatManager
3. Create `game/views/combatview.go` interface
4. Modify CombatMode.Enter() to request combat start via command
5. Modify CombatMode.Exit() to request combat end via command
6. CombatManager emits events; CombatMode subscribes

**Key extraction:**
```go
// FROM: gui/guicombat/combatlifecycle.go
// TO: game/session/combatmanager.go

type CombatManager struct {
    ecsManager     *common.EntityManager
    combatService  *combatservices.CombatService
    eventBus       *events.EventBus
}

func (cm *CombatManager) StartCombat(encounterID ecs.EntityID) error {
    // All the logic currently in CombatLifecycleManager.SetupEncounter
    // ...
    cm.eventBus.Publish(events.Event{Type: events.EventCombatStarted})
    return nil
}

func (cm *CombatManager) EndCombat() error {
    // All the logic currently in CombatLifecycleManager.CleanupCombatEntities
    // ...
    cm.eventBus.Publish(events.Event{Type: events.EventCombatEnded})
    return nil
}
```

**Files to modify:**
- New: `game/session/combatmanager.go`
- New: `game/views/combatview.go`
- Modify: `gui/guicombat/combatmode.go` (use commands, subscribe to events)
- Delete: `gui/guicombat/combatlifecycle.go` (after migration)

### Phase 4: Introduce Command Dispatcher (Medium Impact)

**Goal:** All game-mutating operations go through commands.

**Steps:**
1. Create `game/commands/dispatcher.go`
2. Migrate existing `squadcommands` package to new location
3. Add combat commands (AttackCommand, MoveCommand, EndTurnCommand)
4. Modify UI handlers to construct and dispatch commands
5. Commands emit events on completion

**Files to modify:**
- New: `game/commands/dispatcher.go`
- New: `game/commands/combat_commands.go`
- Move: `tactical/squadcommands/` → `game/commands/squad_commands.go`
- Modify: `gui/guicombat/combat_action_handler.go`
- Modify: `gui/guisquads/squadbuilder.go`

### Phase 5: Create GameSession Facade (Final Integration)

**Goal:** Single entry point for all game operations.

**Steps:**
1. Create `game/session/session.go`
2. GameSession owns all managers (CombatManager, SquadManager)
3. GameSession owns EventBus and CommandDispatcher
4. Modify game startup to create GameSession first
5. Pass GameSession to UI layer
6. Remove direct EntityManager access from UIContext

**Final UIContext:**
```go
type UIContext struct {
    GameSession     *session.GameSession  // Replaces ECSManager
    PlayerData      *common.PlayerData
    ScreenWidth     int
    ScreenHeight    int
    TileSize        int
    ModeCoordinator *GameModeCoordinator
}
```

---

## 5. Trade-offs Analysis

### 5.1 Command Pattern

| Pros | Cons |
|------|------|
| Natural undo/redo support | More boilerplate per action |
| Actions can be logged/replayed | Indirection adds complexity |
| Clear separation of "what" from "how" | Need to serialize for network play |
| Testable without UI | Learning curve for team |

**When to use:** All state-mutating operations (attacks, moves, squad edits).

**When NOT to use:** Pure queries, UI-only state changes.

### 5.2 Event Bus

| Pros | Cons |
|------|------|
| Loose coupling between game and UI | Event ordering can be tricky |
| Multiple subscribers without coupling | Harder to debug (who handled what?) |
| Easy to add new UI features | Memory leaks if unsubscribe forgotten |
| Supports animation queuing | Payload type safety requires care |

**When to use:** Notifications that UI might want to animate/display.

**When NOT to use:** Request/response patterns (use commands + views instead).

### 5.3 View Interfaces

| Pros | Cons |
|------|------|
| Read-only guarantees | More interfaces to maintain |
| Clear API boundary | View creation has some overhead |
| Easy to mock for testing | Queries computed on-demand |
| UI can't corrupt game state | Need to design good view APIs |

**When to use:** All UI reads of game state.

**When NOT to use:** Internal game system communication (use direct access).

### 5.4 Full Refactor vs Incremental

| Approach | Pros | Cons |
|----------|------|------|
| Full rewrite | Clean architecture from start | High risk, long timeline |
| Incremental | Lower risk, ship features during | Temporary mixed patterns |

**Recommendation:** Incremental approach with Phases 1-5 over multiple development cycles.

### 5.5 Complexity Budget

The proposed architecture adds:
- 1 new package (`game/`)
- ~5 new files for events/commands/views
- ~2-3 interfaces per domain (Combat, Squad, Inventory)

This is a moderate complexity increase, justified by:
- Testability of game logic
- Future multiplayer/AI possibilities
- Reduced debugging time for UI/game issues

---

## 6. Reference Implementations

### 6.1 Minimal Event Bus

```go
// game/events/bus.go
package events

import "sync"

type EventType string

type Event struct {
    Type    EventType
    Payload interface{}
}

type Handler func(Event)

type EventBus struct {
    mu          sync.RWMutex
    subscribers map[EventType][]Handler
}

func NewEventBus() *EventBus {
    return &EventBus{
        subscribers: make(map[EventType][]Handler),
    }
}

func (eb *EventBus) Subscribe(eventType EventType, handler Handler) {
    eb.mu.Lock()
    defer eb.mu.Unlock()
    eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

func (eb *EventBus) Publish(event Event) {
    eb.mu.RLock()
    handlers := eb.subscribers[event.Type]
    eb.mu.RUnlock()

    for _, handler := range handlers {
        handler(event)
    }
}
```

### 6.2 Minimal Command Dispatcher

```go
// game/commands/dispatcher.go
package commands

import (
    "fmt"
    "reflect"
)

type Command interface {
    Execute() error
}

type UndoableCommand interface {
    Command
    Undo() error
}

type Dispatcher struct {
    handlers map[reflect.Type]func(Command) error
    history  []UndoableCommand
}

func NewDispatcher() *Dispatcher {
    return &Dispatcher{
        handlers: make(map[reflect.Type]func(Command) error),
    }
}

func (d *Dispatcher) Register(cmdType reflect.Type, handler func(Command) error) {
    d.handlers[cmdType] = handler
}

func (d *Dispatcher) Dispatch(cmd Command) error {
    cmdType := reflect.TypeOf(cmd)
    handler, ok := d.handlers[cmdType]
    if !ok {
        return fmt.Errorf("no handler for command type: %v", cmdType)
    }

    if err := handler(cmd); err != nil {
        return err
    }

    if undoable, ok := cmd.(UndoableCommand); ok {
        d.history = append(d.history, undoable)
    }
    return nil
}

func (d *Dispatcher) Undo() error {
    if len(d.history) == 0 {
        return fmt.Errorf("nothing to undo")
    }
    cmd := d.history[len(d.history)-1]
    d.history = d.history[:len(d.history)-1]
    return cmd.Undo()
}
```

### 6.3 Example Combat View

```go
// game/views/combatview.go
package views

import (
    "game_main/common"
    "game_main/world/coords"
    ecs "github.com/marioolofo/go-gameengine-ecs"
)

type CombatView interface {
    // Turn state
    GetCurrentFactionID() ecs.EntityID
    IsPlayerTurn() bool
    GetTurnNumber() int

    // Unit queries
    CanUnitMove(unitID ecs.EntityID) bool
    CanUnitAttack(unitID ecs.EntityID) bool
    GetValidMoveTargets(unitID ecs.EntityID) []coords.LogicalPosition
    GetValidAttackTargets(unitID ecs.EntityID) []ecs.EntityID

    // Squad queries
    GetFactionSquads(factionID ecs.EntityID) []SquadInfo
    GetSquadUnits(squadID ecs.EntityID) []UnitInfo

    // Combat state
    IsCombatActive() bool
    GetVictor() ecs.EntityID
}

type SquadInfo struct {
    ID       ecs.EntityID
    Name     string
    Position coords.LogicalPosition
    Health   int
    MaxHealth int
}

type UnitInfo struct {
    ID       ecs.EntityID
    Name     string
    Health   int
    MaxHealth int
    HasActed bool
}
```

---

## Appendix: Quick Reference

### A. Import Cheat Sheet

```go
// UI Mode file imports
import (
    "game_main/game/commands"   // To create commands
    "game_main/game/events"     // To subscribe to events
    "game_main/game/views"      // To read game state
    "game_main/gui/framework"   // UI framework
    "game_main/gui/builders"    // Widget builders
)

// NEVER import in UI:
// "game_main/common"          // Direct ECS access
// "game_main/tactical/..."    // Direct service access
```

### B. Pattern Quick Reference

| Want to... | Use |
|------------|-----|
| Modify game state | Dispatch a Command |
| Read game state | Query a View |
| React to game changes | Subscribe to Events |
| Track UI selection | Modify UIState |
| Create widgets | Use builders package |

### C. File Location Guide

| Logic Type | Location |
|------------|----------|
| Game lifecycle | `game/session/` |
| Game commands | `game/commands/` |
| Game events | `game/events/` |
| Game queries | `game/views/` |
| UI modes | `gui/guicombat/`, `gui/guisquads/` |
| UI state | `gui/framework/uistate.go` |
| Widget creation | `gui/builders/` |
| ECS components | `tactical/*/components.go` |
| ECS systems | `tactical/*/system.go` |

---

## Next Steps

1. **Review this document** with the team
2. **Decide on Phase 1** scope (Event Bus introduction)
3. **Create `game/events/` package** as proof of concept
4. **Migrate one event flow** (e.g., unit movement) end-to-end
5. **Measure impact** on code clarity and testability
6. **Proceed to Phase 2** based on learnings
