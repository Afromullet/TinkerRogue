# Refactoring Analysis: GUI-Game State Coupling
Generated: 2026-01-03
Target: GUI subsystems (gui/guicombat, gui/guisquads, gui/core)

## EXECUTIVE SUMMARY

### Target Analysis
- **Scope**: Complete decoupling of GUI layer from tactical game logic across combat and squad management systems
- **Current State**: 15 GUI files directly import tactical packages, violating separation of concerns and creating tight coupling
- **Primary Issues**:
  1. Business logic embedded in GUI action handlers
  2. Game state mixed with UI state
  3. GUI modes owning service lifecycle
  4. Direct ECS queries from presentation layer
  5. Lack of abstraction between layers

### Quick Wins vs Strategic Refactoring
- **Immediate Improvements** (1-2 days):
  - Extract TriggeredEncounterID from BattleMapState to game service
  - Move ExecuteAttack/MoveSquad logic from CombatActionHandler to CombatService
  - Create facade interface for GUI to access game systems

- **Medium-Term Goals** (1-2 weeks):
  - Implement full service layer abstraction
  - Refactor GUI to use commands/events pattern
  - Externalize service lifecycle from GUI modes

- **Long-Term Architecture** (2-4 weeks):
  - Full clean architecture with dependency inversion
  - Event-driven GUI updates
  - Complete testability and modularity

### Consensus Findings
- **Agreement Across Perspectives**: All approaches agree that:
  - Service layer is needed but currently insufficient as abstraction boundary
  - GUI should communicate through interfaces, not concrete tactical types
  - Service lifecycle belongs outside GUI
  - Game state must be cleanly separated from UI state

- **Divergent Perspectives**:
  - **Refactoring-Pro**: Favors full clean architecture with strict layering
  - **Tactical-Simplifier**: Prefers pragmatic facade pattern with minimal disruption
  - **Synthesis**: Hybrid approach - strong interfaces with incremental migration

- **Critical Concerns**:
  - Risk of over-engineering with too many abstraction layers
  - Need to preserve game development performance characteristics
  - Must avoid breaking existing functionality during migration
  - Testing strategy must validate behavior preservation

---

## DETAILED PROBLEM ANALYSIS

### Violation 1: Direct Tactical Package Imports (15 files)

**Location**: gui/guicombat/* and gui/guisquads/*

**Current Implementation**:
```go
// gui/guicombat/combatmode.go
import (
    "game_main/tactical/behavior"
    "game_main/tactical/combat"
    "game_main/tactical/combatservices"
    "game_main/tactical/squads"
)

// gui/guisquads/squaddeploymentmode.go
import (
    "game_main/tactical/squads"
    "game_main/tactical/squadservices"
)
```

**Problem**:
- GUI has compile-time dependency on tactical layer
- Changes to tactical types require GUI recompilation
- Cannot test GUI without tactical implementations
- Violates Dependency Inversion Principle (DIP)
- Makes parallel development difficult

**Impact**:
- High coupling score (15 direct dependencies)
- Difficult to mock for testing
- Circular dependency risk
- Poor modularity

---

### Violation 2: Game State in UI State Structures

**Location**: gui/core/contextstate.go:56

**Current Implementation**:
```go
type BattleMapState struct {
    // UI Selection State
    SelectedSquadID  ecs.EntityID
    SelectedTargetID ecs.EntityID

    // UI Mode Flags
    InAttackMode bool
    InMoveMode   bool

    // VIOLATION: This is game state, not UI state
    TriggeredEncounterID ecs.EntityID // Encounter that triggered this combat
}
```

**Problem**:
- Mixes game state with UI state in same structure
- Violates Single Responsibility Principle (SRP)
- CLAUDE.md explicitly forbids: "BattleMapState = ONLY UI state (selection, mode flags)"
- Encounter management is game logic, not presentation logic

**Impact**:
- Confusion about ownership and responsibilities
- Harder to reason about state changes
- Violates documented architecture principles

**Correct Approach**:
```go
// UI state only
type BattleMapState struct {
    SelectedSquadID  ecs.EntityID
    SelectedTargetID ecs.EntityID
    InAttackMode     bool
    InMoveMode       bool
}

// Game state managed elsewhere (combat service or session)
// GUI references it but doesn't own it
```

---

### Violation 3: Business Logic in GUI Action Handlers

**Location**: gui/guicombat/combat_action_handler.go

**Current Implementation**:
```go
func (cah *CombatActionHandler) ExecuteAttack() {
    selectedSquad := cah.battleMapState.SelectedSquadID
    selectedTarget := cah.battleMapState.SelectedTargetID

    // VIOLATION: Direct call to combat system
    result := cah.combatService.CombatActSystem.ExecuteAttackAction(
        selectedSquad, selectedTarget)

    // VIOLATION: Direct cache invalidation logic
    cah.queries.MarkSquadDirty(selectedSquad)
    cah.queries.MarkSquadDirty(selectedTarget)

    // VIOLATION: UI state manipulation interleaved with game logic
    cah.battleMapState.InAttackMode = false

    // VIOLATION: Business logic decisions in GUI
    if !result.Success {
        cah.addLog(fmt.Sprintf("Cannot attack: %s", result.ErrorReason))
        return
    }
    // ... animation triggering logic
}
```

**Problem**:
- Action handler contains business logic (attack validation, execution)
- GUI component directly manipulates game systems
- Interleaves UI concerns with game logic
- Makes testing difficult (requires full game setup)
- Violates SLAP (Single Level of Abstraction Principle)

**Impact**:
- ~200 lines of business logic in GUI layer
- Cannot test attack logic without GUI
- Difficult to reuse logic in other contexts (e.g., AI, networking)

---

### Violation 4: GUI Modes Creating Services

**Location**: gui/guicombat/combatmode.go:88

**Current Implementation**:
```go
func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    // VIOLATION: GUI mode owns service lifecycle
    cm.combatService = combatservices.NewCombatService(ctx.ECSManager)

    // ... rest of initialization
}
```

**Location**: gui/guisquads/squaddeploymentmode.go:54

**Current Implementation**:
```go
func (sdm *SquadDeploymentMode) Initialize(ctx *core.UIContext) error {
    // VIOLATION: GUI mode owns service lifecycle
    sdm.deploymentService = squadservices.NewSquadDeploymentService(ctx.ECSManager)

    // ... rest of initialization
}
```

**Problem**:
- Service lifecycle coupled to GUI mode lifecycle
- Cannot share services across modes
- Each mode creates its own service instance
- Service state management unclear
- Violates Inversion of Control (IoC) principle

**Impact**:
- Potential service duplication
- State synchronization issues if services should be singletons
- Difficult to manage service dependencies
- Cannot inject mocks for testing

---

### Violation 5: Game Decision Logic in Input Handlers

**Location**: gui/guicombat/combat_input_handler.go:183

**Current Implementation**:
```go
func (cih *CombatInputHandler) handleSquadClick(mouseX, mouseY int) {
    // ... coordinate conversion ...

    clickedSquadID := combat.GetSquadAtPosition(clickedPos, cih.queries.ECSManager)

    // VIOLATION: Faction logic in input handler
    squadInfo := cih.queries.GetSquadInfo(clickedSquadID)
    clickedFactionID := squadInfo.FactionID

    // VIOLATION: Game rules in GUI
    factionData := cih.queries.CombatCache.FindFactionDataByID(
        cih.currentFactionID, cih.queries.ECSManager)

    if factionData != nil && factionData.IsPlayerControlled {
        // VIOLATION: Turn logic in input handler
        if clickedFactionID == cih.currentFactionID {
            cih.actionHandler.SelectSquad(clickedSquadID)
            return
        }

        // VIOLATION: Enemy detection logic in GUI
        if selectedSquad != 0 && clickedFactionID != cih.currentFactionID {
            cih.battleMapState.SelectedTargetID = clickedSquadID
            cih.actionHandler.ExecuteAttack()
        }
    }
}
```

**Problem**:
- Input handler contains faction allegiance logic
- Turn validation logic in presentation layer
- Enemy detection is game logic, not UI logic
- Direct ECS queries from input handler

**Impact**:
- Game rules scattered across GUI code
- Difficult to change faction logic without touching GUI
- Cannot reuse logic for AI or other controllers
- Violates Open/Closed Principle (OCP)

---

## ARCHITECTURAL ANALYSIS

### Current Architecture (Problematic)

```
┌─────────────────────────────────────────┐
│           GUI Layer (Presentation)      │
│  ┌──────────────┐    ┌───────────────┐ │
│  │ CombatMode   │    │ SquadModes    │ │
│  │              │    │               │ │
│  │ - Creates    │    │ - Creates     │ │
│  │   Services   │    │   Services    │ │
│  │ - Calls ECS  │    │ - Calls ECS   │ │
│  │   directly   │    │   directly    │ │
│  └───────┬──────┘    └───────┬───────┘ │
│          │                   │         │
│          └───────┬───────────┘         │
└──────────────────┼─────────────────────┘
                   │ Direct Dependencies
                   ▼
┌─────────────────────────────────────────┐
│        Tactical Layer (Business)        │
│  ┌──────────────┐    ┌───────────────┐ │
│  │ CombatSrvc   │    │ SquadServices │ │
│  └──────────────┘    └───────────────┘ │
│         │                    │          │
│         └────────┬───────────┘          │
│                  ▼                      │
│  ┌─────────────────────────────────┐   │
│  │   ECS Systems & Components      │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘

Problems:
- GUI depends directly on tactical types
- No abstraction boundary
- Services created by GUI
- Business logic in GUI handlers
```

### Target Architecture (Clean)

```
┌─────────────────────────────────────────┐
│           GUI Layer (Presentation)      │
│  ┌──────────────┐    ┌───────────────┐ │
│  │ CombatMode   │    │ SquadModes    │ │
│  │              │    │               │ │
│  │ - Uses       │    │ - Uses        │ │
│  │   Interfaces │    │   Interfaces  │ │
│  │ - Receives   │    │ - Receives    │ │
│  │   Services   │    │   Services    │ │
│  └───────┬──────┘    └───────┬───────┘ │
│          │                   │         │
│          └───────┬───────────┘         │
└──────────────────┼─────────────────────┘
                   │ Interface Dependency
                   ▼
┌─────────────────────────────────────────┐
│      Abstraction Layer (Interfaces)     │
│  ┌──────────────────────────────────┐   │
│  │   ICombatController              │   │
│  │   ISquadManager                  │   │
│  │   IGameSessionFacade             │   │
│  └──────────────────────────────────┘   │
└──────────────────┬──────────────────────┘
                   │ Implementation
                   ▼
┌─────────────────────────────────────────┐
│        Tactical Layer (Business)        │
│  ┌──────────────┐    ┌───────────────┐ │
│  │ CombatSrvc   │    │ SquadServices │ │
│  │ (implements  │    │ (implements   │ │
│  │  interface)  │    │  interface)   │ │
│  └──────────────┘    └───────────────┘ │
│         │                    │          │
│  ┌─────────────────────────────────┐   │
│  │   ECS Systems & Components      │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘

Benefits:
- GUI depends on interfaces only
- Tactical layer hidden behind abstraction
- Services injected into GUI
- Business logic in tactical layer
- Testable with mocks
```

---

## REFACTORING APPROACHES

### APPROACH 1: Service Facade with Interface Abstraction (RECOMMENDED)

**Strategic Focus**: "Pragmatic Clean Architecture - Interface-based decoupling with minimal disruption"

**Problem Statement**:
GUI layer has 15 direct dependencies on tactical packages, making the codebase tightly coupled, difficult to test, and violating the Dependency Inversion Principle. The current `combatservices` and `squadservices` packages exist but are used concretely rather than through abstractions.

**Solution Overview**:
Create a thin interface layer that GUI depends on, with tactical services implementing these interfaces. Use dependency injection to provide services to GUI modes. This approach leverages existing service packages while adding the abstraction boundary needed for proper separation.

**Code Example - Phase 1: Define Interfaces**:

*Before* (gui/guicombat/combatmode.go):
```go
import (
    "game_main/tactical/combatservices"
    "game_main/tactical/combat"
    "game_main/tactical/behavior"
)

type CombatMode struct {
    combatService *combatservices.CombatService
    dangerVisualizer *behavior.DangerVisualizer
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    // GUI creates service
    cm.combatService = combatservices.NewCombatService(ctx.ECSManager)
}
```

*After* (gui/interfaces/game_interfaces.go - NEW FILE):
```go
package interfaces

import (
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// ICombatController provides all combat operations needed by GUI
type ICombatController interface {
    // Combat lifecycle
    InitializeCombat(factionIDs []ecs.EntityID) error
    CheckVictoryCondition() *VictoryInfo

    // Turn management
    GetCurrentFaction() ecs.EntityID
    GetCurrentRound() int
    EndTurn() error

    // Combat actions
    ExecuteAttack(attackerID, defenderID ecs.EntityID) *AttackResult
    GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition
    MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) *MoveResult

    // Squad queries
    GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID
    GetSquadInfo(squadID ecs.EntityID) *SquadInfo
    IsSquadPlayerControlled(squadID ecs.EntityID) bool

    // AI
    ExecuteAITurn(factionID ecs.EntityID) *AITurnResult
}

// IVisualizationController provides visualization data
type IVisualizationController interface {
    UpdateDangerVisualization(factionID ecs.EntityID, round int, centerPos coords.LogicalPosition, viewportSize int)
    GetDangerLevel(pos coords.LogicalPosition) float64
    ToggleDangerView()
    IsDangerVisualizationActive() bool
}

// Simple DTOs for interface (no tactical dependencies)
type AttackResult struct {
    Success         bool
    ErrorReason     string
    AttackerName    string
    TargetName      string
    Damage          int
    TargetDestroyed bool
}

type MoveResult struct {
    Success     bool
    Error       string
    Description string
}

type SquadInfo struct {
    ID           ecs.EntityID
    Name         string
    FactionID    ecs.EntityID
    Position     coords.LogicalPosition
    IsDestroyed  bool
    UnitCount    int
}

type VictoryInfo struct {
    BattleOver       bool
    VictorFaction    ecs.EntityID
    VictorName       string
    DefeatedFactions []ecs.EntityID
    RoundsCompleted  int
}

type AITurnResult struct {
    ActionsExecuted bool
    QueuedAttacks   []QueuedAttackInfo
}

type QueuedAttackInfo struct {
    AttackerID ecs.EntityID
    DefenderID ecs.EntityID
}
```

*After* (gui/guicombat/combatmode.go):
```go
import (
    "game_main/gui/interfaces" // Only GUI interfaces
    "game_main/gui/core"
)

type CombatMode struct {
    combatController interfaces.ICombatController
    vizController    interfaces.IVisualizationController
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    // Services injected via context
    cm.combatController = ctx.GameSession.CombatController()
    cm.vizController = ctx.GameSession.VisualizationController()
}
```

**Code Example - Phase 2: Implement Interfaces**:

*New File* (tactical/combatservices/combat_controller.go):
```go
package combatservices

import (
    "game_main/gui/interfaces"
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// Ensure CombatService implements ICombatController
var _ interfaces.ICombatController = (*CombatService)(nil)

// ExecuteAttack implements ICombatController
func (cs *CombatService) ExecuteAttack(attackerID, defenderID ecs.EntityID) *interfaces.AttackResult {
    // Use existing CombatActionSystem
    result := cs.CombatActSystem.ExecuteAttackAction(attackerID, defenderID)

    // Convert internal result to interface DTO
    return &interfaces.AttackResult{
        Success:         result.Success,
        ErrorReason:     result.ErrorReason,
        AttackerName:    result.AttackerName,
        TargetName:      result.TargetName,
        Damage:          result.Damage,
        TargetDestroyed: result.TargetDestroyed,
    }
}

// GetCurrentFaction implements ICombatController
func (cs *CombatService) GetCurrentFaction() ecs.EntityID {
    return cs.TurnManager.GetCurrentFaction()
}

// ... implement other interface methods
```

**Code Example - Phase 3: Service Injection**:

*Modified* (gui/core/uicontext.go):
```go
type UIContext struct {
    // ... existing fields ...
    GameSession IGameSession // NEW: injected game session
}

// IGameSession provides access to all game controllers
type IGameSession interface {
    CombatController() interfaces.ICombatController
    SquadManager() interfaces.ISquadManager
    VisualizationController() interfaces.IVisualizationController
}
```

*New File* (game_main/game_session.go):
```go
package main

import (
    "game_main/common"
    "game_main/gui/interfaces"
    "game_main/tactical/combatservices"
    "game_main/tactical/squadservices"
)

// GameSession owns all game services and implements IGameSession
type GameSession struct {
    entityManager     *common.EntityManager
    combatService     *combatservices.CombatService
    squadBuilder      *squadservices.SquadBuilderService
    deploymentService *squadservices.SquadDeploymentService
}

func NewGameSession(em *common.EntityManager) *GameSession {
    return &GameSession{
        entityManager:     em,
        combatService:     combatservices.NewCombatService(em),
        squadBuilder:      squadservices.NewSquadBuilderService(em),
        deploymentService: squadservices.NewSquadDeploymentService(em),
    }
}

func (gs *GameSession) CombatController() interfaces.ICombatController {
    return gs.combatService
}

// ... other controller accessors
```

**Key Changes**:
1. Create `gui/interfaces/game_interfaces.go` with all GUI-needed operations
2. Make existing services implement these interfaces
3. Create `GameSession` to own service lifecycle
4. Inject `GameSession` into `UIContext`
5. GUI modes receive services through context, not create them
6. Remove all tactical package imports from GUI

**Value Proposition**:
- **Maintainability**: Clear contract between GUI and game logic
- **Readability**: Interface documents what GUI actually needs
- **Extensibility**: Can add new implementations without changing GUI
- **Complexity Impact**:
  - +1 new package (gui/interfaces)
  - +1 new file (game_session.go)
  - -15 import dependencies removed from GUI
  - ~400 lines of interface definitions
  - ~200 lines of adapter code

**Implementation Strategy**:
1. **Step 1** (4 hours): Create `gui/interfaces/game_interfaces.go` with all interface definitions
2. **Step 2** (6 hours): Make `CombatService` and `SquadServices` implement interfaces
3. **Step 3** (4 hours): Create `GameSession` and wire up service ownership
4. **Step 4** (8 hours): Refactor GUI modes to use injected interfaces
5. **Step 5** (4 hours): Remove tactical imports from GUI, update all call sites
6. **Step 6** (2 hours): Validation testing - ensure all functionality preserved

**Advantages**:
- Leverages existing service layer - no complete rewrite needed
- Interfaces provide compile-time contract verification
- Enables mocking for GUI unit tests
- Services remain concrete internally, only abstracted at boundary
- Incremental migration - can refactor one mode at a time
- Aligns with Go idioms (small interfaces, duck typing)

**Drawbacks & Risks**:
- **Interface Maintenance**: Need to keep interfaces in sync with service capabilities
  - Mitigation: Use static assertion (var _ IInterface = (*Impl)(nil))
- **DTO Conversion Overhead**: Need to convert between internal types and interface DTOs
  - Mitigation: Keep DTOs simple, use composition where possible
- **Increased File Count**: Adds new package and session management code
  - Mitigation: Clear organization, well-documented purposes

**Effort Estimate**:
- **Time**: 28 hours (~3.5 days for one developer)
- **Complexity**: Medium
- **Risk**: Low (incremental changes, interfaces are compile-time checked)
- **Files Impacted**:
  - New: 2 files (interfaces, game_session)
  - Modified: 15 GUI files + 4 service files + uicontext.go = ~20 files

**Critical Assessment** (Refactoring-Critic Perspective):
This approach provides practical value with manageable risk. The interface layer is not over-engineering - it's a necessary abstraction for testability and decoupling. The existing service layer makes this refactoring straightforward since the logic is already organized. The key risk is maintaining the interface as the game evolves, but static type checking and Go's small interface philosophy mitigate this. This is a solid foundation for clean architecture without the overhead of full hexagonal/onion patterns.

---

### APPROACH 2: Command/Event Pattern with Mediator

**Strategic Focus**: "Event-Driven Decoupling - Message-based communication with complete UI/Logic separation"

**Problem Statement**:
GUI currently makes direct synchronous calls to game systems, creating tight coupling and making it difficult to track game state changes or add features like undo/redo, networking, or replays. Business logic is interleaved with presentation logic in event handlers.

**Solution Overview**:
Introduce a command pattern for GUI-to-game communication and an event pattern for game-to-GUI updates. GUI sends commands (e.g., "AttackCommand") to a mediator, which executes them and emits events (e.g., "AttackExecutedEvent") that GUI observes. This creates complete decoupling and enables advanced features.

**Code Example - Command Pattern**:

*New File* (game_main/commands/combat_commands.go):
```go
package commands

import (
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// Command represents any player action
type Command interface {
    Execute() error
    Undo() error
    Redo() error
    GetDescription() string
}

// AttackCommand represents an attack action
type AttackCommand struct {
    AttackerID ecs.EntityID
    DefenderID ecs.EntityID
    executor   *CombatExecutor // Internal, not exposed to GUI
}

func NewAttackCommand(attackerID, defenderID ecs.EntityID) *AttackCommand {
    return &AttackCommand{
        AttackerID: attackerID,
        DefenderID: defenderID,
    }
}

func (c *AttackCommand) Execute() error {
    // Delegate to combat executor (encapsulates tactical layer)
    return c.executor.ExecuteAttack(c.AttackerID, c.DefenderID)
}

func (c *AttackCommand) Undo() error {
    // Cannot undo attacks in this game, return error
    return fmt.Errorf("attacks cannot be undone")
}

func (c *AttackCommand) GetDescription() string {
    return fmt.Sprintf("Attack squad %d with squad %d", c.DefenderID, c.AttackerID)
}

// MoveSquadCommand represents a movement action
type MoveSquadCommand struct {
    SquadID     ecs.EntityID
    OldPosition coords.LogicalPosition
    NewPosition coords.LogicalPosition
    executor    *MovementExecutor
}

func (c *MoveSquadCommand) Execute() error {
    return c.executor.MoveSquad(c.SquadID, c.NewPosition)
}

func (c *MoveSquadCommand) Undo() error {
    // Movement can be undone
    return c.executor.MoveSquad(c.SquadID, c.OldPosition)
}
```

*New File* (game_main/commands/command_mediator.go):
```go
package commands

import (
    "fmt"
    "game_main/events"
)

// CommandMediator receives commands from GUI and executes them
type CommandMediator struct {
    eventBus    *events.EventBus
    history     []Command
    historyPos  int
}

func NewCommandMediator(eventBus *events.EventBus) *CommandMediator {
    return &CommandMediator{
        eventBus:   eventBus,
        history:    make([]Command, 0),
        historyPos: -1,
    }
}

// Execute runs a command and adds it to history
func (m *CommandMediator) Execute(cmd Command) error {
    err := cmd.Execute()
    if err != nil {
        return err
    }

    // Add to history for undo/redo
    m.history = append(m.history[:m.historyPos+1], cmd)
    m.historyPos++

    // Emit event for GUI to react
    m.eventBus.Publish(events.CommandExecutedEvent{
        Command: cmd,
    })

    return nil
}

func (m *CommandMediator) Undo() error {
    if m.historyPos < 0 {
        return fmt.Errorf("nothing to undo")
    }

    cmd := m.history[m.historyPos]
    err := cmd.Undo()
    if err != nil {
        return err
    }

    m.historyPos--
    m.eventBus.Publish(events.CommandUndoneEvent{Command: cmd})
    return nil
}
```

**Code Example - Event Pattern**:

*New File* (game_main/events/combat_events.go):
```go
package events

import (
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// Event represents any game state change
type Event interface {
    GetType() string
    GetTimestamp() time.Time
}

// AttackExecutedEvent is emitted when an attack completes
type AttackExecutedEvent struct {
    AttackerID      ecs.EntityID
    DefenderID      ecs.EntityID
    Damage          int
    DefenderDestroyed bool
    Timestamp       time.Time
}

func (e AttackExecutedEvent) GetType() string { return "AttackExecuted" }
func (e AttackExecutedEvent) GetTimestamp() time.Time { return e.Timestamp }

// SquadMovedEvent is emitted when a squad moves
type SquadMovedEvent struct {
    SquadID      ecs.EntityID
    OldPosition  coords.LogicalPosition
    NewPosition  coords.LogicalPosition
    Timestamp    time.Time
}

func (e SquadMovedEvent) GetType() string { return "SquadMoved" }
func (e SquadMovedEvent) GetTimestamp() time.Time { return e.Timestamp }

// TurnEndedEvent is emitted when a faction's turn ends
type TurnEndedEvent struct {
    OldFactionID ecs.EntityID
    NewFactionID ecs.EntityID
    Round        int
    Timestamp    time.Time
}
```

*New File* (game_main/events/event_bus.go):
```go
package events

import "sync"

// EventBus manages event subscriptions and publishing
type EventBus struct {
    subscribers map[string][]EventHandler
    mu          sync.RWMutex
}

type EventHandler func(Event)

func NewEventBus() *EventBus {
    return &EventBus{
        subscribers: make(map[string][]EventHandler),
    }
}

// Subscribe registers a handler for an event type
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
    eb.mu.Lock()
    defer eb.mu.Unlock()
    eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

// Publish sends an event to all subscribers
func (eb *EventBus) Publish(event Event) {
    eb.mu.RLock()
    handlers := eb.subscribers[event.GetType()]
    eb.mu.RUnlock()

    for _, handler := range handlers {
        handler(event)
    }
}
```

**Code Example - GUI Integration**:

*After* (gui/guicombat/combat_action_handler.go):
```go
package guicombat

import (
    "game_main/commands"
    "game_main/events"
)

type CombatActionHandler struct {
    mediator    *commands.CommandMediator
    eventBus    *events.EventBus
    // ... UI state only, no game logic
}

func NewCombatActionHandler(mediator *commands.CommandMediator, eventBus *events.EventBus) *CombatActionHandler {
    cah := &CombatActionHandler{
        mediator: mediator,
        eventBus: eventBus,
    }

    // Subscribe to game events
    eventBus.Subscribe("AttackExecuted", cah.onAttackExecuted)
    eventBus.Subscribe("SquadMoved", cah.onSquadMoved)

    return cah
}

// ExecuteAttack now just creates and sends a command
func (cah *CombatActionHandler) ExecuteAttack(attackerID, defenderID ecs.EntityID) {
    cmd := commands.NewAttackCommand(attackerID, defenderID)
    err := cah.mediator.Execute(cmd)
    if err != nil {
        cah.addLog(fmt.Sprintf("Attack failed: %s", err.Error()))
    }
}

// Event handler reacts to game state changes
func (cah *CombatActionHandler) onAttackExecuted(event events.Event) {
    attackEvent := event.(events.AttackExecutedEvent)

    // Update UI based on event
    cah.addLog(fmt.Sprintf("Attack dealt %d damage", attackEvent.Damage))
    if attackEvent.DefenderDestroyed {
        cah.addLog("Target destroyed!")
    }

    // Trigger animation if needed
    cah.triggerAttackAnimation(attackEvent.AttackerID, attackEvent.DefenderID)
}
```

**Key Changes**:
1. Create command objects for all player actions
2. Create event objects for all game state changes
3. Introduce CommandMediator to execute commands
4. Introduce EventBus for pub/sub communication
5. GUI sends commands, subscribes to events
6. Game logic executes commands, publishes events

**Value Proposition**:
- **Maintainability**: Complete separation of concerns - GUI only handles presentation
- **Readability**: Clear intent (commands) and clear reactions (event handlers)
- **Extensibility**: Easy to add new features (logging, networking, AI) by subscribing to events
- **Complexity Impact**:
  - +2 new packages (commands, events)
  - +~800 lines of command/event infrastructure
  - -~300 lines removed from GUI action handlers
  - Enables undo/redo, replay, networking

**Implementation Strategy**:
1. **Step 1** (8 hours): Create command infrastructure (Command interface, concrete commands, CommandMediator)
2. **Step 2** (6 hours): Create event infrastructure (Event interface, concrete events, EventBus)
3. **Step 3** (10 hours): Refactor CombatActionHandler to use commands/events
4. **Step 4** (8 hours): Refactor squad GUI handlers to use commands/events
5. **Step 5** (6 hours): Wire up event subscriptions in all GUI modes
6. **Step 6** (4 hours): Testing and validation

**Advantages**:
- **Complete Decoupling**: GUI has zero knowledge of game implementation
- **Testability**: Can test commands in isolation, mock event bus for GUI tests
- **Advanced Features**: Foundation for undo/redo, replay, networking, AI training
- **Temporal Decoupling**: Events can be queued, delayed, batched
- **Audit Trail**: Command history provides complete action log
- **Parallel Development**: GUI team and game logic team can work independently

**Drawbacks & Risks**:
- **Indirection Overhead**: More mental hops from action to result
  - Mitigation: Good naming, documentation, and IDE support for navigation
- **Event Synchronization**: Need to handle event ordering and race conditions
  - Mitigation: Single-threaded event bus, or careful mutex usage
- **Boilerplate**: Every action needs command + event + handler
  - Mitigation: Code generation or templates for common patterns
- **Debugging Difficulty**: Harder to trace execution flow through event bus
  - Mitigation: Event logging, debugging tools, good error messages
- **Potential Over-engineering**: May be overkill for a single-player local game
  - Mitigation: Only implement if planning multiplayer/networking features

**Effort Estimate**:
- **Time**: 42 hours (~5.5 days for one developer)
- **Complexity**: High
- **Risk**: Medium (new architectural pattern, requires careful event design)
- **Files Impacted**:
  - New: ~15 files (command definitions, event definitions, mediator, event bus)
  - Modified: All 15 GUI files + service layer = ~25 files

**Critical Assessment** (Refactoring-Critic Perspective):
This approach is theoretically excellent and provides powerful capabilities, but it may be over-engineering for TinkerRogue's current needs. The command/event pattern shines in distributed systems, multiplayer games, or when undo/redo is critical. For a single-player tactical game, the indirection overhead and increased complexity may outweigh benefits. However, if future plans include multiplayer, replays, or complex undo scenarios, this investment pays off. The risk is spending ~1 week on infrastructure that might not be needed. Recommend this approach only if you have concrete plans for features that require event-driven architecture.

---

### APPROACH 3: Incremental Extraction with Facade Pattern

**Strategic Focus**: "Minimal Disruption Migration - Gradual extraction using facade pattern for low-risk refactoring"

**Problem Statement**:
A complete architectural overhaul (Approach 1) or command/event system (Approach 2) carries significant risk and effort. Teams need a way to improve coupling incrementally while maintaining system stability and allowing ongoing feature development.

**Solution Overview**:
Introduce a single facade class (`GameFacade`) that wraps all tactical operations the GUI needs. GUI imports only this facade, not tactical packages. Incrementally move business logic from GUI handlers into the facade. This provides immediate decoupling benefits with minimal architectural changes.

**Code Example - Facade Creation**:

*New File* (gui/gamefacade/game_facade.go):
```go
package gamefacade

import (
    "game_main/common"
    "game_main/world/coords"
    "game_main/tactical/combatservices"
    "game_main/tactical/squadservices"
    "github.com/bytearena/ecs"
)

// GameFacade provides simplified access to all game operations needed by GUI
// This is the ONLY point of contact between GUI and tactical layer
type GameFacade struct {
    entityManager     *common.EntityManager
    combatService     *combatservices.CombatService
    squadBuilder      *squadservices.SquadBuilderService
    deploymentService *squadservices.SquadDeploymentService
}

func NewGameFacade(em *common.EntityManager) *GameFacade {
    return &GameFacade{
        entityManager:     em,
        combatService:     combatservices.NewCombatService(em),
        squadBuilder:      squadservices.NewSquadBuilderService(em),
        deploymentService: squadservices.NewSquadDeploymentService(em),
    }
}

// === Combat Operations ===

// CombatAttackRequest contains attack parameters
type CombatAttackRequest struct {
    AttackerID ecs.EntityID
    DefenderID ecs.EntityID
}

// CombatAttackResponse contains attack results
type CombatAttackResponse struct {
    Success         bool
    ErrorMessage    string
    AttackerName    string
    DefenderName    string
    Damage          int
    DefenderKilled  bool
}

// ExecuteCombatAttack handles all attack logic
func (gf *GameFacade) ExecuteCombatAttack(req CombatAttackRequest) CombatAttackResponse {
    // Business logic centralized here, not in GUI
    result := gf.combatService.CombatActSystem.ExecuteAttackAction(req.AttackerID, req.DefenderID)

    // Invalidate caches (GUI doesn't need to know about this)
    gf.invalidateSquadCache(req.AttackerID)
    gf.invalidateSquadCache(req.DefenderID)

    return CombatAttackResponse{
        Success:        result.Success,
        ErrorMessage:   result.ErrorReason,
        AttackerName:   result.AttackerName,
        DefenderName:   result.TargetName,
        Damage:         result.Damage,
        DefenderKilled: result.TargetDestroyed,
    }
}

// MoveSquadRequest contains movement parameters
type MoveSquadRequest struct {
    SquadID     ecs.EntityID
    NewPosition coords.LogicalPosition
}

// MoveSquadResponse contains movement results
type MoveSquadResponse struct {
    Success      bool
    ErrorMessage string
    Description  string
    OldPosition  coords.LogicalPosition
}

// MoveSquad handles all movement logic
func (gf *GameFacade) MoveSquad(req MoveSquadRequest) MoveSquadResponse {
    // Get current position
    squadEntity := gf.entityManager.FindEntityByID(req.SquadID)
    if squadEntity == nil {
        return MoveSquadResponse{
            Success:      false,
            ErrorMessage: "Squad not found",
        }
    }

    oldPos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
    if oldPos == nil {
        return MoveSquadResponse{
            Success:      false,
            ErrorMessage: "Squad has no position",
        }
    }

    // Execute movement
    err := gf.combatService.MovementSystem.MoveSquad(req.SquadID, req.NewPosition)
    if err != nil {
        return MoveSquadResponse{
            Success:      false,
            ErrorMessage: err.Error(),
        }
    }

    // Invalidate cache
    gf.invalidateSquadCache(req.SquadID)

    squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
    description := fmt.Sprintf("Moved %s from (%d,%d) to (%d,%d)",
        squadData.Name, oldPos.X, oldPos.Y, req.NewPosition.X, req.NewPosition.Y)

    return MoveSquadResponse{
        Success:     true,
        Description: description,
        OldPosition: *oldPos,
    }
}

// GetValidMovement tiles for a squad
func (gf *GameFacade) GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition {
    tiles := gf.combatService.MovementSystem.GetValidMovementTiles(squadID)
    if tiles == nil {
        return []coords.LogicalPosition{}
    }
    return tiles
}

// === Turn Management ===

type TurnInfo struct {
    CurrentFaction     ecs.EntityID
    CurrentRound       int
    FactionName        string
    IsPlayerControlled bool
    PlayerName         string
}

func (gf *GameFacade) GetCurrentTurnInfo() TurnInfo {
    factionID := gf.combatService.TurnManager.GetCurrentFaction()
    round := gf.combatService.TurnManager.GetCurrentRound()

    factionData := gf.combatService.CombatCache.FindFactionDataByID(factionID, gf.entityManager)

    info := TurnInfo{
        CurrentFaction: factionID,
        CurrentRound:   round,
        FactionName:    "Unknown",
    }

    if factionData != nil {
        info.FactionName = factionData.Name
        info.IsPlayerControlled = factionData.IsPlayerControlled
        info.PlayerName = factionData.PlayerName
    }

    return info
}

func (gf *GameFacade) EndTurn() error {
    return gf.combatService.TurnManager.EndTurn()
}

// === Squad Queries ===

type SquadSummary struct {
    ID          ecs.EntityID
    Name        string
    Position    coords.LogicalPosition
    FactionID   ecs.EntityID
    IsDestroyed bool
    UnitCount   int
}

func (gf *GameFacade) GetSquadSummary(squadID ecs.EntityID) *SquadSummary {
    squadEntity := gf.entityManager.FindEntityByID(squadID)
    if squadEntity == nil {
        return nil
    }

    squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
    posData := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
    factionData := common.GetComponentType[*combat.FactionData](squadEntity, combat.FactionComponent)

    if squadData == nil || posData == nil {
        return nil
    }

    summary := &SquadSummary{
        ID:          squadID,
        Name:        squadData.Name,
        Position:    *posData,
        IsDestroyed: squads.IsSquadDestroyed(squadID, gf.entityManager),
        UnitCount:   len(squadData.MemberIDs),
    }

    if factionData != nil {
        summary.FactionID = factionData.FactionID
    }

    return summary
}

// GetAliveSquadsInFaction returns alive squads for a faction
func (gf *GameFacade) GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID {
    return gf.combatService.GetAliveSquadsInFaction(factionID)
}

// === Helper Methods ===

func (gf *GameFacade) invalidateSquadCache(squadID ecs.EntityID) {
    // Cache invalidation logic encapsulated here
    // GUI doesn't need to know about caching
}
```

**Code Example - GUI Using Facade**:

*Before* (gui/guicombat/combat_action_handler.go):
```go
package guicombat

import (
    "game_main/tactical/combatservices"
    "game_main/tactical/combat"
    "game_main/gui/guicomponents"
)

type CombatActionHandler struct {
    combatService *combatservices.CombatService
    queries       *guicomponents.GUIQueries
}

func (cah *CombatActionHandler) ExecuteAttack() {
    selectedSquad := cah.battleMapState.SelectedSquadID
    selectedTarget := cah.battleMapState.SelectedTargetID

    // Direct call to tactical layer
    result := cah.combatService.CombatActSystem.ExecuteAttackAction(selectedSquad, selectedTarget)

    // Manual cache invalidation
    cah.queries.MarkSquadDirty(selectedSquad)
    cah.queries.MarkSquadDirty(selectedTarget)

    // UI logic mixed with game logic
    cah.battleMapState.InAttackMode = false

    if !result.Success {
        cah.addLog(fmt.Sprintf("Cannot attack: %s", result.ErrorReason))
        return
    }

    // ... more logic
}
```

*After* (gui/guicombat/combat_action_handler.go):
```go
package guicombat

import (
    "game_main/gui/gamefacade"
    "game_main/gui/core"
)

type CombatActionHandler struct {
    gameFacade     *gamefacade.GameFacade
    battleMapState *core.BattleMapState
}

func (cah *CombatActionHandler) ExecuteAttack() {
    selectedSquad := cah.battleMapState.SelectedSquadID
    selectedTarget := cah.battleMapState.SelectedTargetID

    // Single facade call - all business logic inside facade
    response := cah.gameFacade.ExecuteCombatAttack(gamefacade.CombatAttackRequest{
        AttackerID: selectedSquad,
        DefenderID: selectedTarget,
    })

    // UI logic only
    cah.battleMapState.InAttackMode = false

    if !response.Success {
        cah.addLog(fmt.Sprintf("Cannot attack: %s", response.ErrorMessage))
        return
    }

    cah.addLog(fmt.Sprintf("%s attacked %s for %d damage!",
        response.AttackerName, response.DefenderName, response.Damage))

    if response.DefenderKilled {
        cah.addLog(fmt.Sprintf("%s was destroyed!", response.DefenderName))
    }
}
```

**Key Changes**:
1. Create single `GameFacade` class in `gui/gamefacade` package
2. Facade contains request/response types for all operations
3. GUI imports only `gui/gamefacade`, not tactical packages
4. Move business logic from GUI handlers into facade methods
5. Facade internally owns/creates all tactical services
6. Cache invalidation, validation, etc. moved into facade

**Value Proposition**:
- **Maintainability**: Single point of contact reduces coupling
- **Readability**: Request/response types clarify operation contracts
- **Extensibility**: Easy to add new operations to facade
- **Complexity Impact**:
  - +1 new package (gui/gamefacade)
  - +~600 lines for facade implementation
  - -~200 lines removed from GUI handlers
  - -15 import statements removed from GUI

**Implementation Strategy**:
1. **Step 1** (4 hours): Create `GameFacade` skeleton with basic structure
2. **Step 2** (6 hours): Extract combat operations (attack, move, turn management)
3. **Step 3** (6 hours): Extract squad operations (queries, management)
4. **Step 4** (8 hours): Refactor CombatMode to use facade
5. **Step 5** (6 hours): Refactor SquadModes to use facade
6. **Step 6** (2 hours): Remove tactical imports from GUI, validation testing

**Advantages**:
- **Incremental Migration**: Can refactor one operation at a time
- **Low Risk**: Changes are localized, existing logic preserved
- **Simple Pattern**: Facade is well-understood, easy to explain
- **No New Concepts**: No commands, events, or complex abstractions
- **Quick Wins**: Immediate decoupling with minimal code changes
- **Backward Compatible**: Can keep old code paths during migration

**Drawbacks & Risks**:
- **Facade Bloat**: Risk of creating "god object" with too many responsibilities
  - Mitigation: Organize facade into logical sections, use composition
- **Not True Abstraction**: Still concrete dependency, just one layer removed
  - Mitigation: Facade can later implement interfaces (Approach 1 upgrade path)
- **Cache Knowledge**: GUI still needs to know about cache invalidation timing
  - Mitigation: Move all cache logic into facade
- **Temptation to Leak**: Easy to expose too much tactical detail through facade
  - Mitigation: Strict review, only expose what GUI actually needs

**Effort Estimate**:
- **Time**: 32 hours (~4 days for one developer)
- **Complexity**: Low-Medium
- **Risk**: Low (incremental, non-breaking changes)
- **Files Impacted**:
  - New: 1 file (game_facade.go)
  - Modified: 15 GUI files = ~16 files total

**Critical Assessment** (Refactoring-Critic Perspective):
This approach provides the best effort-to-value ratio for teams that want quick decoupling wins without architectural risk. The facade pattern is simple, well-understood, and incrementally adoptable. The main risk is creating a bloated facade, but this can be managed with good organization and code reviews. This approach is ideal if you want to improve architecture without blocking feature development. It's also a good stepping stone - once the facade is in place, you can refactor it to implement interfaces (Approach 1) or extract commands (Approach 2) later. Recommended for teams prioritizing pragmatism and steady progress over architectural purity.

---

## COMPARATIVE ANALYSIS OF APPROACHES

### Effort vs Impact Matrix

| Approach | Effort | Impact | Risk | Recommended Priority |
|----------|--------|--------|------|---------------------|
| **Approach 1: Service Facade with Interfaces** | Medium (3.5 days) | High | Low | **1 - RECOMMENDED** |
| **Approach 2: Command/Event Pattern** | High (5.5 days) | Very High | Medium | 3 - Only if planning multiplayer |
| **Approach 3: Incremental Facade** | Low-Medium (4 days) | Medium | Low | 2 - Good stepping stone |

### Decision Guidance

**Choose Approach 1 (Service Facade with Interfaces) if:**
- You want clean architecture with testability
- You value compile-time type safety
- You want to enable parallel development (GUI and game logic teams)
- You're comfortable with Go interfaces and dependency injection
- You want a solid foundation for future growth
- **This is the RECOMMENDED approach for most teams**

**Choose Approach 2 (Command/Event Pattern) if:**
- You're planning multiplayer or networking features
- You need full undo/redo capability
- You want replay functionality
- You're building a complex simulation requiring event sourcing
- You have time for significant architectural investment
- You have experience with event-driven architectures

**Choose Approach 3 (Incremental Facade) if:**
- You want the quickest path to basic decoupling
- You're actively developing features and can't pause for refactoring
- You want minimal risk and maximum backward compatibility
- You prefer simple patterns over sophisticated architectures
- You plan to upgrade to Approach 1 later
- You're new to clean architecture and want to learn incrementally

### Combination Opportunities

**Recommended Path: Approach 3 → Approach 1**
1. **Week 1**: Implement Approach 3 (Facade)
   - Get immediate decoupling benefits
   - Continue feature development with improved structure
2. **Week 2-3**: Upgrade facade to implement interfaces (Approach 1)
   - Extract interfaces from facade methods
   - Add dependency injection
   - Full clean architecture achieved incrementally

**Advanced Path: Approach 1 → Selective Approach 2**
1. **Weeks 1-2**: Implement Approach 1 (Interfaces)
   - Clean architecture foundation in place
2. **Later**: Add command/event pattern only where needed
   - Undo/redo for specific operations
   - Event bus for cross-cutting concerns (logging, analytics)
   - No need to refactor everything to commands

---

## DETAILED IMPLEMENTATION ROADMAP

### Phase 1: Foundation (Week 1)

**Goals**:
- Remove direct tactical imports from GUI
- Establish clear abstraction boundary
- Minimal disruption to ongoing development

**Tasks**:
1. **Create Interface Package** (Day 1, 4 hours)
   - Create `gui/interfaces/game_interfaces.go`
   - Define `ICombatController`, `ISquadManager`, `IVisualizationController`
   - Define DTOs (AttackResult, MoveResult, SquadInfo, etc.)
   - Document interface contracts

2. **Implement Interfaces in Services** (Day 1-2, 8 hours)
   - Make `CombatService` implement `ICombatController`
   - Make squad services implement `ISquadManager`
   - Add static type assertions (var _ IInterface = (*Impl)(nil))
   - Write adapter methods for DTO conversions

3. **Create GameSession** (Day 2, 4 hours)
   - Create `game_main/game_session.go`
   - Implement service ownership and lifecycle
   - Implement `IGameSession` interface
   - Wire up service factory methods

4. **Update UIContext** (Day 2, 2 hours)
   - Add `GameSession IGameSession` field to `UIContext`
   - Update context initialization in main.go
   - Inject game session into context

5. **Validation Testing** (Day 3, 6 hours)
   - Write interface compliance tests
   - Test DTO conversions
   - Verify service lifecycle
   - End-to-end smoke tests

### Phase 2: Combat GUI Migration (Week 2)

**Goals**:
- Refactor combat GUI to use injected interfaces
- Remove tactical imports from combat GUI files
- Preserve all existing functionality

**Tasks**:
1. **Refactor CombatMode** (Day 1, 6 hours)
   - Change service fields to interface types
   - Get services from `ctx.GameSession`
   - Update all method calls to use interface methods
   - Remove tactical package imports

2. **Refactor CombatActionHandler** (Day 2, 6 hours)
   - Update `ExecuteAttack()` to use `ICombatController.ExecuteAttack()`
   - Update `MoveSquad()` to use `ICombatController.MoveSquad()`
   - Replace direct `TurnManager` calls with interface methods
   - Remove tactical imports

3. **Refactor CombatInputHandler** (Day 2, 4 hours)
   - Update faction checking to use interface queries
   - Replace direct combat system calls
   - Remove tactical imports

4. **Testing and Validation** (Day 3, 6 hours)
   - Combat mode functionality testing
   - Attack/move/turn progression testing
   - Animation integration testing
   - Performance regression testing

### Phase 3: Squad GUI Migration (Week 3)

**Goals**:
- Refactor squad GUI to use injected interfaces
- Remove tactical imports from squad GUI files
- Complete GUI layer decoupling

**Tasks**:
1. **Refactor SquadDeploymentMode** (Day 1, 6 hours)
   - Update to use `ISquadManager` interface
   - Get service from `ctx.GameSession`
   - Remove tactical imports

2. **Refactor SquadEditorMode** (Day 2, 6 hours)
   - Update to use `ISquadManager` interface
   - Replace direct squad service calls
   - Remove tactical imports

3. **Refactor SquadBuilderMode** (Day 2, 4 hours)
   - Update to use `ISquadManager` interface
   - Update unit purchase logic
   - Remove tactical imports

4. **Final Validation** (Day 3, 6 hours)
   - Full regression testing
   - Performance testing
   - Code coverage analysis
   - Documentation updates

### Phase 4: Cleanup and Optimization (Week 4)

**Goals**:
- Remove dead code
- Optimize DTO conversions
- Add comprehensive tests
- Update documentation

**Tasks**:
1. **Code Cleanup** (Day 1, 4 hours)
   - Remove unused imports
   - Delete dead code paths
   - Refactor redundant logic
   - Apply Go fmt and vet

2. **Write Unit Tests** (Day 1-2, 8 hours)
   - GUI unit tests with mocked interfaces
   - Service layer tests with real implementations
   - Integration tests for critical paths
   - Achieve >80% code coverage

3. **Performance Optimization** (Day 3, 4 hours)
   - Profile DTO conversion overhead
   - Optimize hot paths
   - Cache frequently-used data
   - Benchmark critical operations

4. **Documentation** (Day 3-4, 8 hours)
   - Update CLAUDE.md with new patterns
   - Document interface contracts
   - Write migration guide for future developers
   - Create architecture diagrams

---

## TESTING STRATEGY

### Unit Testing Approach

**GUI Layer Tests** (with mocked interfaces):
```go
package guicombat_test

import (
    "testing"
    "game_main/gui/guicombat"
    "game_main/gui/interfaces"
    "github.com/bytearena/ecs"
)

// Mock combat controller for testing
type MockCombatController struct {
    executeAttackCalled bool
    attackResult        *interfaces.AttackResult
}

func (m *MockCombatController) ExecuteAttack(attackerID, defenderID ecs.EntityID) *interfaces.AttackResult {
    m.executeAttackCalled = true
    return m.attackResult
}

// ... implement other interface methods

func TestCombatActionHandler_ExecuteAttack_Success(t *testing.T) {
    // Setup mock
    mockController := &MockCombatController{
        attackResult: &interfaces.AttackResult{
            Success:      true,
            AttackerName: "Squad A",
            TargetName:   "Squad B",
            Damage:       10,
        },
    }

    // Create handler with mock
    handler := guicombat.NewCombatActionHandler(mockController)
    handler.ExecuteAttack(1, 2)

    // Verify
    if !mockController.executeAttackCalled {
        t.Error("ExecuteAttack not called on controller")
    }
}

func TestCombatActionHandler_ExecuteAttack_Failure(t *testing.T) {
    // Setup mock with failure
    mockController := &MockCombatController{
        attackResult: &interfaces.AttackResult{
            Success:     false,
            ErrorReason: "Out of range",
        },
    }

    handler := guicombat.NewCombatActionHandler(mockController)
    handler.ExecuteAttack(1, 2)

    // Verify error handling
    // (check log message, UI state, etc.)
}
```

**Service Layer Tests** (with real ECS):
```go
package combatservices_test

import (
    "testing"
    "game_main/common"
    "game_main/tactical/combatservices"
    "game_main/gui/interfaces"
)

func TestCombatService_ExecuteAttack_ImplementsInterface(t *testing.T) {
    // Verify service implements interface
    var _ interfaces.ICombatController = (*combatservices.CombatService)(nil)
}

func TestCombatService_ExecuteAttack_Success(t *testing.T) {
    // Setup ECS with test entities
    em := common.NewEntityManager()
    service := combatservices.NewCombatService(em)

    // Create test squads
    attackerID := createTestSquad(em, "Attacker", 100, 10)
    defenderID := createTestSquad(em, "Defender", 50, 5)

    // Execute attack
    result := service.ExecuteAttack(attackerID, defenderID)

    // Verify result
    if !result.Success {
        t.Errorf("Attack should succeed, got error: %s", result.ErrorReason)
    }
    if result.Damage <= 0 {
        t.Error("Attack should deal damage")
    }
}
```

### Integration Testing

**Full Combat Flow Test**:
```go
func TestCombatFlow_PlayerAttackAI_IntegrationTest(t *testing.T) {
    // Setup full game session
    em := common.NewEntityManager()
    gameSession := NewGameSession(em)

    // Create test factions and squads
    playerFaction := createTestFaction(em, "Player", true)
    aiFaction := createTestFaction(em, "AI", false)

    playerSquad := createTestSquadInFaction(em, playerFaction, "Player Squad")
    aiSquad := createTestSquadInFaction(em, aiFaction, "AI Squad")

    // Initialize combat
    controller := gameSession.CombatController()
    err := controller.InitializeCombat([]ecs.EntityID{playerFaction, aiFaction})
    if err != nil {
        t.Fatalf("Failed to initialize combat: %v", err)
    }

    // Verify it's player's turn
    turnInfo := controller.GetCurrentTurnInfo()
    if !turnInfo.IsPlayerControlled {
        t.Error("Player should go first")
    }

    // Execute attack
    result := controller.ExecuteAttack(playerSquad, aiSquad)
    if !result.Success {
        t.Errorf("Attack should succeed: %s", result.ErrorReason)
    }

    // End turn
    err = controller.EndTurn()
    if err != nil {
        t.Errorf("Failed to end turn: %v", err)
    }

    // Verify turn changed to AI
    turnInfo = controller.GetCurrentTurnInfo()
    if turnInfo.IsPlayerControlled {
        t.Error("Turn should have switched to AI")
    }
}
```

### Regression Testing

**Behavior Preservation Tests**:
- Record current behavior before refactoring
- Replay same inputs after refactoring
- Verify outputs match exactly
- Use golden file testing for complex scenarios

**Performance Regression Tests**:
- Benchmark critical paths before and after
- Ensure no >10% performance degradation
- Profile DTO conversion overhead
- Monitor memory allocations

---

## VIOLATION REMEDIATION CHECKLIST

### V1: Direct Tactical Package Imports (15 files)
- [ ] Create `gui/interfaces/game_interfaces.go` with all needed interfaces
- [ ] Make `CombatService` implement `ICombatController`
- [ ] Make squad services implement `ISquadManager`
- [ ] Create `GameSession` to own service lifecycle
- [ ] Inject `GameSession` into `UIContext`
- [ ] Refactor all 6 combat GUI files to use interfaces
- [ ] Refactor all 6 squad GUI files to use interfaces
- [ ] Remove tactical imports from all GUI files
- [ ] Add static type assertions to verify interface compliance
- [ ] Write tests with mocked interfaces

### V2: Game State in UI State Structures
- [ ] Remove `TriggeredEncounterID` from `BattleMapState`
- [ ] Add encounter tracking to `GameSession` or `CombatService`
- [ ] Add `GetCurrentEncounter()` method to `ICombatController`
- [ ] Update `CombatMode.Enter()` to query encounter from service
- [ ] Update `CombatMode.Exit()` to update encounter state through service
- [ ] Document that `BattleMapState` is UI state only
- [ ] Add validation tests to prevent game state in UI state

### V3: Business Logic in GUI Action Handlers
- [ ] Move attack validation logic to `CombatService.ExecuteAttack()`
- [ ] Move movement logic to `CombatService.MoveSquad()`
- [ ] Move cache invalidation to service layer
- [ ] Make `CombatActionHandler` purely presentational
- [ ] Extract business logic tests into service layer tests
- [ ] GUI tests should only mock interfaces, not test game logic

### V4: GUI Modes Creating Services
- [ ] Remove `NewCombatService()` call from `CombatMode.Initialize()`
- [ ] Remove `NewSquadDeploymentService()` call from `SquadDeploymentMode.Initialize()`
- [ ] Create services in `GameSession` constructor
- [ ] Inject services via `UIContext.GameSession`
- [ ] Update all mode Initialize() methods to receive services
- [ ] Add lifecycle tests to verify services persist across mode changes

### V5: Faction Logic in Input Handlers
- [ ] Move faction checking to `ICombatController.CanSelectSquad()`
- [ ] Move enemy detection to `ICombatController.IsEnemySquad()`
- [ ] Move turn validation to `ICombatController.IsPlayerTurn()`
- [ ] Update `CombatInputHandler` to call interface methods for decisions
- [ ] Extract game rules tests into service layer
- [ ] GUI tests should verify input handling, not game rules

---

## RISK MITIGATION

### Technical Risks

**Risk: Interface Changes Break GUI**
- **Likelihood**: Medium
- **Impact**: High
- **Mitigation**:
  - Static type assertions catch missing methods at compile time
  - Comprehensive interface tests
  - Version interface with deprecation warnings

**Risk: DTO Conversion Overhead**
- **Likelihood**: Low
- **Impact**: Medium
- **Mitigation**:
  - Keep DTOs simple (no deep copying)
  - Benchmark conversion in hot paths
  - Use pointers where appropriate

**Risk: Service State Synchronization**
- **Likelihood**: Low
- **Impact**: High
- **Mitigation**:
  - Services owned by single `GameSession`
  - Clear ownership documentation
  - Integration tests verify state consistency

### Process Risks

**Risk: Team Unfamiliar with Interfaces**
- **Likelihood**: Medium
- **Impact**: Medium
- **Mitigation**:
  - Pair programming during initial implementation
  - Code review with architecture discussion
  - Document patterns in CLAUDE.md

**Risk: Scope Creep During Refactoring**
- **Likelihood**: High
- **Impact**: Medium
- **Mitigation**:
  - Strict adherence to roadmap
  - Feature freeze during refactoring weeks
  - Regular check-ins on progress

**Risk: Regression Bugs**
- **Likelihood**: Medium
- **Impact**: High
- **Mitigation**:
  - Comprehensive regression test suite
  - Manual testing of all combat/squad scenarios
  - Phased rollout (one subsystem at a time)

---

## SUCCESS METRICS

### Code Quality Metrics
- **Coupling**: Reduce GUI→Tactical dependencies from 15 to 1 (interface package)
- **Cohesion**: All GUI files import only gui/* packages
- **Test Coverage**: Achieve >80% coverage in GUI layer
- **Lines of Code**: Net neutral or slight reduction (removing duplication)

### Architectural Metrics
- **Dependency Direction**: GUI depends on interfaces, tactical implements them
- **Abstraction Layers**: Clear separation (GUI → Interfaces → Tactical → ECS)
- **Service Ownership**: All services owned by GameSession, not GUI modes

### Team Velocity Metrics
- **Development Speed**: No degradation after refactoring complete
- **Bug Rate**: No increase in production bugs
- **Onboarding Time**: New developers understand architecture faster

---

## CONCLUSION

This refactoring analysis provides three distinct approaches to addressing GUI-Game State coupling in TinkerRogue:

1. **Approach 1 (RECOMMENDED)**: Service Facade with Interface Abstraction
   - Best balance of clean architecture and pragmatism
   - 3.5 days effort, high impact, low risk
   - Enables testability, parallel development, and future growth
   - Leverages existing service layer with minimal disruption

2. **Approach 2**: Command/Event Pattern with Mediator
   - Powerful but potentially over-engineered for current needs
   - 5.5 days effort, very high impact, medium risk
   - Recommended only if planning multiplayer or advanced features

3. **Approach 3**: Incremental Extraction with Facade Pattern
   - Quickest path to basic decoupling
   - 4 days effort, medium impact, low risk
   - Good stepping stone toward Approach 1
   - Ideal for teams learning clean architecture incrementally

**Recommended Action Plan**:
- **Immediate** (Week 1): Implement Approach 1 interfaces and GameSession
- **Short-term** (Week 2-3): Migrate combat and squad GUI to use interfaces
- **Medium-term** (Week 4): Cleanup, testing, documentation
- **Long-term**: Monitor for opportunities to selectively add command/event pattern

All approaches address the core violations identified in CLAUDE.md and provide a path to cleaner, more maintainable architecture while respecting the game development context and performance requirements.

---

**END OF ANALYSIS**
