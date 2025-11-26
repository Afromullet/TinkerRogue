# Refactoring Analysis: GUI Presentation/Logic Separation
Generated: 2025-11-25
Target: gui/ folder - Separating presentation/drawing logic from gameplay/action logic

## EXECUTIVE SUMMARY

### Target Analysis
- **Scope**: All gui/ packages (core, guicombat, guisquads, guimodes, guicomponents, widgets)
- **Current State**: Tight coupling between UI rendering code and game logic execution. UI modes own game systems (TurnManager, FactionManager), action handlers execute combat logic AND update UI in same methods, Squad Builder directly manipulates ECS from event callbacks.
- **Primary Issues**:
  1. **System Ownership Violation**: CombatMode owns game systems (TurnManager lines 42-44, FactionManager line 43, MovementSystem line 44 in combatmode.go)
  2. **Mixed Responsibilities**: CombatActionHandler executes combat logic (ExecuteAttackAction line 190) AND updates UI (addLog line 194) in combat_action_handler.go
  3. **Direct ECS Manipulation**: SquadBuilderMode manipulates ECS directly from UI callbacks (lines 161-215, 330-378 in squadbuilder.go)
  4. **Untestable Business Logic**: No separate business logic layer - all game mechanics embedded in GUI classes
  5. **State Confusion**: BattleMapState mixes UI selection state with computed game state (ValidMoveTiles line 53 in contextstate.go)

- **Recommended Direction**: Create a Command/Event-driven architecture with a dedicated Game State Layer that owns game systems, processes game commands, and emits events for UI updates. This provides clear separation while maintaining the existing ECS patterns and game flow.

### Quick Wins vs Strategic Refactoring
- **Immediate Improvements** (1-3 days):
  - Extract combat execution logic from CombatActionHandler into combat system functions
  - Create GameStateManager to own TurnManager/FactionManager instead of CombatMode
  - Separate BattleMapState into UISelectionState and ComputedGameState

- **Medium-Term Goals** (1-2 weeks):
  - Implement Command pattern for all game actions (AttackCommand, MoveCommand, etc.)
  - Create EventBus for game state changes (TurnChanged, SquadMoved, CombatEnded)
  - Refactor SquadBuilder to use commands instead of direct ECS manipulation

- **Long-Term Architecture** (2-4 weeks):
  - Full Game State Layer with testable business logic
  - Complete event-driven UI updates
  - Replay/undo capability through command history

### Consensus Findings
- **Agreement Across All Perspectives**:
  - Current architecture violates separation of concerns
  - Game logic must be extracted from UI classes
  - Need a middle layer between GUI and ECS
  - Testing business logic is currently impossible

- **Divergent Perspectives**:
  - **Pragmatic view**: Incremental extraction with minimal refactoring
  - **Architectural view**: Full Command/Event pattern with proper layering
  - **Game-specific view**: Optimize for turn-based gameplay with game loop integration

- **Critical Concerns**:
  - Risk of over-engineering with too many abstraction layers
  - Performance impact of event dispatching in game loop
  - Migration path complexity - must not break existing gameplay
  - Need to preserve ECS patterns (don't cache state, use queries)

---

## FINAL SYNTHESIZED APPROACHES

### Approach 1: Incremental Service Layer Extraction

**Strategic Focus**: Gradual extraction with minimal architectural disruption - pragmatic path to testability

**Problem Statement**:
Current code embeds game logic directly in UI handlers, making it impossible to test combat mechanics, movement validation, or squad operations without instantiating the entire GUI stack. Example: CombatActionHandler.executeAttack() (combat_action_handler.go lines 167-204) mixes combat system creation, validation checks, attack execution, destruction checking, AND UI logging in a single method that's untestable in isolation.

**Solution Overview**:
Create dedicated service classes (CombatService, SquadService, MovementService) that encapsulate game logic and are injected into UI handlers. Services own game systems and expose clean APIs. UI handlers become thin coordinators that call services and update UI based on results. This provides immediate testability without requiring full architectural restructure.

**Code Example**:

*Before (combat_action_handler.go lines 167-204):*
```go
func (cah *CombatActionHandler) executeAttack() {
    selectedSquad := cah.battleMapState.SelectedSquadID
    selectedTarget := cah.battleMapState.SelectedTargetID

    if selectedSquad == 0 || selectedTarget == 0 {
        return
    }

    // Create combat action system (BUSINESS LOGIC IN UI)
    combatSys := combat.NewCombatActionSystem(cah.entityManager)

    // Check if attack is valid (BUSINESS LOGIC IN UI)
    reason, canAttack := combatSys.CanSquadAttackWithReason(selectedSquad, selectedTarget)
    if !canAttack {
        cah.addLog(fmt.Sprintf("Cannot attack: %s", reason))
        cah.battleMapState.InAttackMode = false
        return
    }

    // Execute attack (BUSINESS LOGIC IN UI)
    attackerName := cah.queries.GetSquadName(selectedSquad)
    targetName := cah.queries.GetSquadName(selectedTarget)

    err := combatSys.ExecuteAttackAction(selectedSquad, selectedTarget)
    if err != nil {
        cah.addLog(fmt.Sprintf("Attack failed: %v", err))
    } else {
        cah.addLog(fmt.Sprintf("%s attacked %s!", attackerName, targetName))

        // Check if target destroyed (BUSINESS LOGIC IN UI)
        if squads.IsSquadDestroyed(selectedTarget, cah.entityManager) {
            cah.addLog(fmt.Sprintf("%s was destroyed!", targetName))
        }
    }

    // Reset attack mode (UI STATE UPDATE)
    cah.battleMapState.InAttackMode = false
}
```

*After - Service Layer (combat/combat_service.go):*
```go
// CombatService encapsulates all combat game logic
type CombatService struct {
    entityManager  *common.EntityManager
    turnManager    *TurnManager
    factionManager *FactionManager
}

func NewCombatService(manager *common.EntityManager) *CombatService {
    return &CombatService{
        entityManager:  manager,
        turnManager:    NewTurnManager(manager),
        factionManager: NewFactionManager(manager),
    }
}

// AttackResult contains all information about an attack
type AttackResult struct {
    Success         bool
    ErrorReason     string
    AttackerName    string
    TargetName      string
    TargetDestroyed bool
    DamageDealt     int
}

// ExecuteSquadAttack performs a squad attack and returns result
func (cs *CombatService) ExecuteSquadAttack(attackerID, targetID ecs.EntityID) *AttackResult {
    result := &AttackResult{}

    // Create combat action system
    combatSys := NewCombatActionSystem(cs.entityManager)

    // Validate attack
    reason, canAttack := combatSys.CanSquadAttackWithReason(attackerID, targetID)
    if !canAttack {
        result.Success = false
        result.ErrorReason = reason
        return result
    }

    // Get names for result
    result.AttackerName = getSquadNameByID(attackerID, cs.entityManager)
    result.TargetName = getSquadNameByID(targetID, cs.entityManager)

    // Execute attack
    err := combatSys.ExecuteAttackAction(attackerID, targetID)
    if err != nil {
        result.Success = false
        result.ErrorReason = err.Error()
        return result
    }

    result.Success = true
    result.TargetDestroyed = squads.IsSquadDestroyed(targetID, cs.entityManager)

    return result
}

// GetTurnManager exposes turn manager for UI queries
func (cs *CombatService) GetTurnManager() *TurnManager {
    return cs.turnManager
}

// GetFactionManager exposes faction manager for UI queries
func (cs *CombatService) GetFactionManager() *FactionManager {
    return cs.factionManager
}
```

*After - UI Handler (combat_action_handler.go):*
```go
type CombatActionHandler struct {
    battleMapState *core.BattleMapState
    logManager     *CombatLogManager
    queries        *guicomponents.GUIQueries
    combatService  *combat.CombatService  // Service injection
    combatLogArea  *widget.TextArea
}

func NewCombatActionHandler(
    battleMapState *core.BattleMapState,
    logManager *CombatLogManager,
    queries *guicomponents.GUIQueries,
    combatService *combat.CombatService,  // Inject service
    combatLogArea *widget.TextArea,
) *CombatActionHandler {
    return &CombatActionHandler{
        battleMapState: battleMapState,
        logManager:     logManager,
        queries:        queries,
        combatService:  combatService,
        combatLogArea:  combatLogArea,
    }
}

func (cah *CombatActionHandler) executeAttack() {
    selectedSquad := cah.battleMapState.SelectedSquadID
    selectedTarget := cah.battleMapState.SelectedTargetID

    if selectedSquad == 0 || selectedTarget == 0 {
        return
    }

    // Call service for all game logic
    result := cah.combatService.ExecuteSquadAttack(selectedSquad, selectedTarget)

    // Handle result - UI ONLY
    if !result.Success {
        cah.addLog(fmt.Sprintf("Cannot attack: %s", result.ErrorReason))
    } else {
        cah.addLog(fmt.Sprintf("%s attacked %s!", result.AttackerName, result.TargetName))
        if result.TargetDestroyed {
            cah.addLog(fmt.Sprintf("%s was destroyed!", result.TargetName))
        }
    }

    // Reset UI state
    cah.battleMapState.InAttackMode = false
}
```

**Key Changes**:
- Created CombatService that owns TurnManager, FactionManager, CombatActionSystem
- Extracted ExecuteSquadAttack() method that returns rich AttackResult struct
- UI handler becomes thin - calls service, interprets result, updates UI
- Business logic is now 100% testable without GUI dependencies
- Service exposes TurnManager/FactionManager for UI queries (read-only access)

**Value Proposition**:
- **Maintainability**: Game logic centralized in services, not scattered across UI handlers
- **Readability**: Clear separation - services = logic, handlers = coordination + UI
- **Extensibility**: Easy to add new game actions (just add service methods)
- **Complexity Impact**:
  - Reduced cyclomatic complexity in UI handlers (from 15+ to <5)
  - New service layer adds ~500 lines but removes ~800 from UI
  - Net -300 lines with better organization

**Implementation Strategy**:
1. **Phase 1 - Create Services** (2 days):
   - Create combat/combat_service.go with CombatService struct
   - Move TurnManager/FactionManager/MovementSystem ownership from CombatMode to CombatService
   - Create squads/squad_service.go with SquadService struct
   - Implement 5-10 core service methods (ExecuteSquadAttack, MoveSquad, CreateSquad, etc.)

2. **Phase 2 - Refactor CombatMode** (1 day):
   - Inject CombatService into CombatMode instead of creating managers directly
   - Update CombatActionHandler to use CombatService
   - Update UI code to call service methods instead of direct system calls
   - Test combat flow end-to-end

3. **Phase 3 - Refactor SquadBuilder** (1 day):
   - Create SquadBuilderService with PlaceUnit, RemoveUnit, CreateSquad methods
   - Inject service into SquadBuilderMode
   - Remove direct ECS manipulation from UI callbacks
   - Test squad builder flow

4. **Phase 4 - Add Tests** (1 day):
   - Write unit tests for CombatService.ExecuteSquadAttack()
   - Write tests for SquadService.CreateSquad() with capacity validation
   - Write tests for MovementService.ValidateMove()
   - Achieve >80% coverage of business logic

**Advantages**:
- **Immediate testability**: Can write unit tests for combat logic TODAY (just test service methods)
- **Incremental migration**: Refactor one UI handler at a time, no big-bang rewrite
- **Low risk**: Services wrap existing ECS code, minimal changes to combat/squads packages
- **Familiar pattern**: Service layer is well-understood, team can adapt quickly
- **Performance neutral**: No event dispatching overhead, direct method calls

**Drawbacks & Risks**:
- **Not complete separation**: Services still directly manipulate ECS, not a true "model" layer
  - *Mitigation*: This is acceptable for Phase 1. Services ARE the model layer for now.
- **Potential for fat services**: Services could grow large with many methods
  - *Mitigation*: Split into domain services (CombatService, SquadService, MovementService)
- **Still tightly coupled to ECS**: Services depend on EntityManager
  - *Mitigation*: This is by design. ECS IS our data model. Services orchestrate ECS systems.
- **Doesn't enable undo/replay**: Service methods mutate state immediately
  - *Mitigation*: Add command layer later if needed. Don't over-engineer now.

**Effort Estimate**:
- **Time**: 5 days (1 week with testing)
- **Complexity**: Low-Medium
- **Risk**: Low (wrapping existing code, not rewriting)
- **Files Impacted**:
  - New: 3 files (combat_service.go, squad_service.go, movement_service.go)
  - Modified: 8 files (combatmode.go, combat_action_handler.go, squadbuilder.go, squad_builder_grid_manager.go, formationeditormode.go, squaddeploymentmode.go, squadmanagementmode.go, gamemodecoordinator.go)

**Critical Assessment** (from refactoring-critic):
This is the **most pragmatic approach**. It provides immediate value (testability) without architectural astronautics. The service layer is a well-understood pattern that team members can grasp in 5 minutes. Risk is low because we're wrapping existing code, not rewriting it. The only danger is treating this as "done" when it's really Phase 1 - services are not a complete separation, but they're a huge step forward. Recommended as **first step** before considering more complex patterns.

---

### Approach 2: Command/Event Architecture with Game State Manager

**Strategic Focus**: Full separation with command/event patterns - enables undo, replay, networking

**Problem Statement**:
Current architecture makes it impossible to implement gameplay features like undo/redo, replay system, or networked multiplayer because game logic is executed immediately in UI handlers with no record of what happened. There's also no way to test "what if" scenarios (e.g., "what if squad A attacks squad B?") without actually modifying game state. The UI is directly coupled to ECS mutations, creating a God Object problem where CombatMode (lines 19-55 in combatmode.go) owns everything.

**Solution Overview**:
Implement a Command pattern where all game actions become command objects (AttackCommand, MoveCommand, EndTurnCommand). Create a GameStateManager that owns all game systems and processes commands. Commands return events (AttackExecuted, SquadMoved, TurnEnded) that UI handlers subscribe to for updates. This creates a unidirectional data flow: UI -> Commands -> GameStateManager -> Events -> UI Updates.

**Code Example**:

*Before (combatmode.go lines 42-44, 71-73, combat_action_handler.go lines 167-204):*
```go
// CombatMode owns game systems directly
type CombatMode struct {
    // ... other fields ...
    turnManager    *combat.TurnManager      // UI OWNS GAME LOGIC
    factionManager *combat.FactionManager   // UI OWNS GAME LOGIC
    movementSystem *combat.MovementSystem   // UI OWNS GAME LOGIC
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    // UI mode creates game systems
    cm.turnManager = combat.NewTurnManager(ctx.ECSManager)
    cm.factionManager = combat.NewFactionManager(ctx.ECSManager)
    cm.movementSystem = combat.NewMovementSystem(ctx.ECSManager, common.GlobalPositionSystem)
    // ...
}

// UI handler executes game logic directly
func (cah *CombatActionHandler) executeAttack() {
    combatSys := combat.NewCombatActionSystem(cah.entityManager)
    err := combatSys.ExecuteAttackAction(selectedSquad, selectedTarget)
    // ... logging ...
}
```

*After - Command Pattern (game/commands.go):*
```go
package game

// GameCommand interface for all game actions
type GameCommand interface {
    Execute(state *GameState) GameEvent
    Validate(state *GameState) error
}

// AttackCommand represents a squad attack action
type AttackCommand struct {
    AttackerID ecs.EntityID
    TargetID   ecs.EntityID
}

func (cmd *AttackCommand) Validate(state *GameState) error {
    combatSys := combat.NewCombatActionSystem(state.EntityManager)
    reason, canAttack := combatSys.CanSquadAttackWithReason(cmd.AttackerID, cmd.TargetID)
    if !canAttack {
        return fmt.Errorf("cannot attack: %s", reason)
    }
    return nil
}

func (cmd *AttackCommand) Execute(state *GameState) GameEvent {
    combatSys := combat.NewCombatActionSystem(state.EntityManager)

    attackerName := getSquadName(cmd.AttackerID, state.EntityManager)
    targetName := getSquadName(cmd.TargetID, state.EntityManager)

    err := combatSys.ExecuteAttackAction(cmd.AttackerID, cmd.TargetID)
    if err != nil {
        return &AttackFailedEvent{
            AttackerID: cmd.AttackerID,
            TargetID:   cmd.TargetID,
            Reason:     err.Error(),
        }
    }

    destroyed := squads.IsSquadDestroyed(cmd.TargetID, state.EntityManager)

    return &AttackExecutedEvent{
        AttackerID:      cmd.AttackerID,
        AttackerName:    attackerName,
        TargetID:        cmd.TargetID,
        TargetName:      targetName,
        TargetDestroyed: destroyed,
    }
}

// MoveCommand represents squad movement
type MoveCommand struct {
    SquadID     ecs.EntityID
    Destination coords.LogicalPosition
}

func (cmd *MoveCommand) Validate(state *GameState) error {
    validTiles := state.MovementSystem.GetValidMovementTiles(cmd.SquadID)
    for _, tile := range validTiles {
        if tile == cmd.Destination {
            return nil // Valid move
        }
    }
    return fmt.Errorf("invalid movement destination")
}

func (cmd *MoveCommand) Execute(state *GameState) GameEvent {
    err := state.MovementSystem.MoveSquad(cmd.SquadID, cmd.Destination)
    if err != nil {
        return &MoveFailedEvent{SquadID: cmd.SquadID, Reason: err.Error()}
    }

    squadName := getSquadName(cmd.SquadID, state.EntityManager)
    return &SquadMovedEvent{
        SquadID:     cmd.SquadID,
        SquadName:   squadName,
        NewPosition: cmd.Destination,
    }
}

// EndTurnCommand ends the current faction's turn
type EndTurnCommand struct{}

func (cmd *EndTurnCommand) Validate(state *GameState) error {
    if state.TurnManager.GetCurrentFaction() == 0 {
        return fmt.Errorf("no active combat")
    }
    return nil
}

func (cmd *EndTurnCommand) Execute(state *GameState) GameEvent {
    state.TurnManager.EndTurn()

    newFaction := state.TurnManager.GetCurrentFaction()
    newRound := state.TurnManager.GetCurrentRound()

    return &TurnEndedEvent{
        NewFactionID: newFaction,
        NewRound:     newRound,
    }
}
```

*After - Event System (game/events.go):*
```go
package game

// GameEvent interface for all game state changes
type GameEvent interface {
    EventType() string
}

type AttackExecutedEvent struct {
    AttackerID      ecs.EntityID
    AttackerName    string
    TargetID        ecs.EntityID
    TargetName      string
    TargetDestroyed bool
}

func (e *AttackExecutedEvent) EventType() string { return "AttackExecuted" }

type AttackFailedEvent struct {
    AttackerID ecs.EntityID
    TargetID   ecs.EntityID
    Reason     string
}

func (e *AttackFailedEvent) EventType() string { return "AttackFailed" }

type SquadMovedEvent struct {
    SquadID     ecs.EntityID
    SquadName   string
    NewPosition coords.LogicalPosition
}

func (e *SquadMovedEvent) EventType() string { return "SquadMoved" }

type TurnEndedEvent struct {
    NewFactionID ecs.EntityID
    NewRound     int
}

func (e *TurnEndedEvent) EventType() string { return "TurnEnded" }

// EventBus for distributing events to subscribers
type EventBus struct {
    subscribers map[string][]func(GameEvent)
}

func NewEventBus() *EventBus {
    return &EventBus{
        subscribers: make(map[string][]func(GameEvent)),
    }
}

func (eb *EventBus) Subscribe(eventType string, handler func(GameEvent)) {
    eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

func (eb *EventBus) Publish(event GameEvent) {
    if handlers, exists := eb.subscribers[event.EventType()]; exists {
        for _, handler := range handlers {
            handler(event)
        }
    }
}
```

*After - Game State Manager (game/game_state_manager.go):*
```go
package game

// GameState owns all game systems and ECS manager
type GameState struct {
    EntityManager  *common.EntityManager
    TurnManager    *combat.TurnManager
    FactionManager *combat.FactionManager
    MovementSystem *combat.MovementSystem
    EventBus       *EventBus

    commandHistory []GameCommand  // For undo/replay
}

func NewGameState(entityManager *common.EntityManager) *GameState {
    return &GameState{
        EntityManager:  entityManager,
        TurnManager:    combat.NewTurnManager(entityManager),
        FactionManager: combat.NewFactionManager(entityManager),
        MovementSystem: combat.NewMovementSystem(entityManager, common.GlobalPositionSystem),
        EventBus:       NewEventBus(),
        commandHistory: make([]GameCommand, 0),
    }
}

// ExecuteCommand validates and executes a command, publishes resulting event
func (gs *GameState) ExecuteCommand(cmd GameCommand) error {
    // Validate
    if err := cmd.Validate(gs); err != nil {
        return err
    }

    // Execute
    event := cmd.Execute(gs)

    // Record in history (for replay/undo)
    gs.commandHistory = append(gs.commandHistory, cmd)

    // Publish event
    gs.EventBus.Publish(event)

    return nil
}

// Query methods for UI (read-only)
func (gs *GameState) GetCurrentFaction() ecs.EntityID {
    return gs.TurnManager.GetCurrentFaction()
}

func (gs *GameState) GetCurrentRound() int {
    return gs.TurnManager.GetCurrentRound()
}

func (gs *GameState) GetValidMoveTiles(squadID ecs.EntityID) []coords.LogicalPosition {
    return gs.MovementSystem.GetValidMovementTiles(squadID)
}
```

*After - UI Handler (combat_action_handler.go):*
```go
type CombatActionHandler struct {
    battleMapState *core.BattleMapState
    logManager     *CombatLogManager
    queries        *guicomponents.GUIQueries
    gameState      *game.GameState  // Reference to game state
    combatLogArea  *widget.TextArea
}

func NewCombatActionHandler(
    battleMapState *core.BattleMapState,
    logManager *CombatLogManager,
    queries *guicomponents.GUIQueries,
    gameState *game.GameState,
    combatLogArea *widget.TextArea,
) *CombatActionHandler {
    handler := &CombatActionHandler{
        battleMapState: battleMapState,
        logManager:     logManager,
        queries:        queries,
        gameState:      gameState,
        combatLogArea:  combatLogArea,
    }

    // Subscribe to events
    handler.subscribeToEvents()

    return handler
}

func (cah *CombatActionHandler) subscribeToEvents() {
    // Subscribe to attack events
    cah.gameState.EventBus.Subscribe("AttackExecuted", func(event game.GameEvent) {
        e := event.(*game.AttackExecutedEvent)
        cah.addLog(fmt.Sprintf("%s attacked %s!", e.AttackerName, e.TargetName))
        if e.TargetDestroyed {
            cah.addLog(fmt.Sprintf("%s was destroyed!", e.TargetName))
        }
        cah.battleMapState.InAttackMode = false
    })

    cah.gameState.EventBus.Subscribe("AttackFailed", func(event game.GameEvent) {
        e := event.(*game.AttackFailedEvent)
        cah.addLog(fmt.Sprintf("Cannot attack: %s", e.Reason))
        cah.battleMapState.InAttackMode = false
    })

    cah.gameState.EventBus.Subscribe("SquadMoved", func(event game.GameEvent) {
        e := event.(*game.SquadMovedEvent)
        cah.addLog(fmt.Sprintf("%s moved to (%d, %d)", e.SquadName, e.NewPosition.X, e.NewPosition.Y))
        cah.battleMapState.InMoveMode = false
        cah.battleMapState.ValidMoveTiles = []coords.LogicalPosition{}
    })
}

func (cah *CombatActionHandler) executeAttack() {
    selectedSquad := cah.battleMapState.SelectedSquadID
    selectedTarget := cah.battleMapState.SelectedTargetID

    if selectedSquad == 0 || selectedTarget == 0 {
        return
    }

    // Create and execute command - that's it!
    cmd := &game.AttackCommand{
        AttackerID: selectedSquad,
        TargetID:   selectedTarget,
    }

    // Game state handles validation, execution, and event publishing
    // UI handler just listens for events
    cah.gameState.ExecuteCommand(cmd)
}

func (cah *CombatActionHandler) MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) error {
    cmd := &game.MoveCommand{
        SquadID:     squadID,
        Destination: newPos,
    }
    return cah.gameState.ExecuteCommand(cmd)
}
```

**Key Changes**:
- Created GameCommand interface for all game actions
- Created GameEvent interface for all state changes
- Created EventBus for publish/subscribe pattern
- Created GameState that owns all game systems (TurnManager, FactionManager, etc.)
- UI handlers submit commands and subscribe to events
- Unidirectional data flow: UI -> Commands -> GameState -> Events -> UI

**Value Proposition**:
- **Maintainability**: Clear separation - game logic in commands, UI logic in event handlers
- **Readability**: UI code is declarative - "execute this command" not imperative "do these 10 steps"
- **Extensibility**: Adding new actions is trivial - create command, create event, done
- **Complexity Impact**:
  - Adds ~800 lines of new code (command/event infrastructure)
  - Removes ~500 lines from UI handlers (logic moved to commands)
  - Net +300 lines but massively better organized
- **New Capabilities**:
  - Undo/redo: Store command history, implement inverse commands
  - Replay: Save command stream, replay to recreate game state
  - Networking: Serialize commands, send to server/clients
  - AI testing: Create command sequences programmatically
  - "What if" scenarios: Execute commands on cloned state

**Implementation Strategy**:
1. **Phase 1 - Infrastructure** (2 days):
   - Create game/ package with GameCommand, GameEvent interfaces
   - Implement EventBus with Subscribe/Publish methods
   - Create GameState that owns all game systems
   - Move TurnManager/FactionManager/MovementSystem ownership from CombatMode to GameState

2. **Phase 2 - Core Commands** (2 days):
   - Implement AttackCommand, MoveCommand, EndTurnCommand
   - Implement corresponding events (AttackExecuted, SquadMoved, TurnEnded)
   - Test command execution in isolation

3. **Phase 3 - Combat UI Integration** (2 days):
   - Inject GameState into CombatMode/CombatActionHandler
   - Replace direct system calls with ExecuteCommand calls
   - Implement event subscribers in UI handlers
   - Update UI based on events instead of direct result inspection

4. **Phase 4 - Squad Builder Integration** (2 days):
   - Implement CreateSquadCommand, PlaceUnitCommand, RemoveUnitCommand
   - Update SquadBuilderMode to use commands
   - Subscribe to squad events for UI updates

5. **Phase 5 - Testing & Polish** (2 days):
   - Write command unit tests (validate and execute in isolation)
   - Test event flow end-to-end
   - Add command history tracking
   - Implement basic undo (stretch goal)

**Advantages**:
- **Complete separation**: UI has zero game logic, just command creation and event handling
- **Testability**: Commands are pure functions (given state, execute, return event)
- **Replay capability**: Save command stream, replay entire game session
- **Undo/redo**: Store commands, implement inverse operations
- **Network-ready**: Commands are serializable, can send over network
- **AI-testable**: Can create command sequences programmatically for AI testing
- **Debugging**: Command history shows exact player actions that led to current state

**Drawbacks & Risks**:
- **Higher complexity**: More moving parts (commands, events, event bus, state manager)
  - *Mitigation*: Start with 5-10 core commands, expand gradually. Don't create command for every action on day 1.
- **Event ordering issues**: Multiple events from one command, subscribers may execute in wrong order
  - *Mitigation*: Design events to be self-contained. Each event should have all data needed to update UI.
- **Performance**: Event dispatching adds overhead vs direct method calls
  - *Mitigation*: Measure first. Event dispatch is O(n) over subscribers, typically <10. Profile if >1000 events/sec.
- **Learning curve**: Team needs to understand command/event patterns
  - *Mitigation*: Provide reference implementation with detailed comments. Pair programming for first few commands.
- **Over-engineering risk**: Might be building for features we don't need (undo, replay, networking)
  - *Mitigation*: Treat these as "options" unlocked by architecture. Don't implement undo unless needed.

**Effort Estimate**:
- **Time**: 10 days (2 weeks)
- **Complexity**: High
- **Risk**: Medium (new architecture, team learning curve)
- **Files Impacted**:
  - New: 10+ files (game/commands.go, game/events.go, game/event_bus.go, game/game_state_manager.go, game/*_command.go for each command type)
  - Modified: 15+ files (all combat UI files, all squad UI files, gamemodecoordinator.go, contextstate.go)

**Critical Assessment** (from refactoring-critic):
This is **architecturally excellent but potentially premature**. The command/event pattern provides beautiful separation and unlocks powerful features (undo, replay, networking). However, we must ask: **do we actually need those features?** If the answer is "not now, maybe later," this is over-engineering. The added complexity (commands, events, event bus) is only justified if we're building towards specific features that require it.

**Recommendation**: Only choose this approach if:
1. You plan to implement undo/replay within next 6 months
2. You're considering multiplayer/networking
3. You need to test complex AI scenarios programmatically

Otherwise, start with Approach 1 (Service Layer) and **migrate to this** when you actually need the capabilities. Don't build it "just in case."

---

### Approach 3: Game Loop State Machine with Turn Processor

**Strategic Focus**: Optimize for turn-based gameplay with explicit turn phases and testable turn logic

**Problem Statement**:
Current architecture treats combat as a sequence of UI-triggered actions with no clear turn structure. Combat state is implicit (scattered across TurnManager, ActionStateData, BattleMapState) rather than explicit. There's no way to validate "is this action legal given current turn phase?" or enforce turn flow (e.g., movement before attack, can't move twice). The game loop just renders whatever state exists, with no concept of "turn processing phase" vs "waiting for input phase". This makes it hard to add complex turn-based features like:
- Simultaneous resolution (both sides plan, then resolve together)
- Action point systems (each action costs points)
- Time-limited turns (30 second chess clock)
- Turn validation (can't attack if you've already moved, based on unit type)

**Solution Overview**:
Create a Turn State Machine that explicitly models turn phases (Planning, Execution, Resolution, NextTurn). Create a TurnProcessor that processes queued actions during Execution phase. UI handlers queue actions during Planning phase, then trigger Execution. This separates "what player wants to do" from "executing game mechanics," enabling complex turn logic, simultaneous resolution, and full testability of turn flow.

**Code Example**:

*Before (combat/turnmanager.go, combat_action_handler.go):*
```go
// TurnManager just tracks whose turn it is
type TurnManager struct {
    manager *common.EntityManager
}

func (tm *TurnManager) GetCurrentFaction() ecs.EntityID { /* ... */ }
func (tm *TurnManager) EndTurn() error { /* ... */ }

// Actions execute immediately when player clicks
func (cah *CombatActionHandler) executeAttack() {
    combatSys := combat.NewCombatActionSystem(cah.entityManager)
    err := combatSys.ExecuteAttackAction(selectedSquad, selectedTarget)
    // Executes RIGHT NOW - no turn phase concept
}
```

*After - Turn State Machine (game/turn_state_machine.go):*
```go
package game

// TurnPhase represents current phase of a turn
type TurnPhase int

const (
    PhaseWaitingForInput TurnPhase = iota  // Waiting for player to queue actions
    PhasePlanning                           // Player planning actions (can still cancel/modify)
    PhaseValidating                         // Validating queued actions
    PhaseExecuting                          // Executing actions in order
    PhaseResolving                          // Resolving side effects (ability triggers, etc.)
    PhaseNextTurn                           // Moving to next faction's turn
)

// TurnStateMachine manages turn phases and transitions
type TurnStateMachine struct {
    currentPhase   TurnPhase
    currentFaction ecs.EntityID
    currentRound   int

    queuedActions  []TurnAction       // Actions queued this turn
    validationErrs []error            // Errors from validation phase
    executionLog   []TurnActionResult // Results from execution
}

func NewTurnStateMachine() *TurnStateMachine {
    return &TurnStateMachine{
        currentPhase:   PhaseWaitingForInput,
        queuedActions:  make([]TurnAction, 0),
        validationErrs: make([]error, 0),
        executionLog:   make([]TurnActionResult, 0),
    }
}

// QueueAction adds an action to be executed this turn
func (tsm *TurnStateMachine) QueueAction(action TurnAction) error {
    if tsm.currentPhase != PhaseWaitingForInput && tsm.currentPhase != PhasePlanning {
        return fmt.Errorf("cannot queue action during %v phase", tsm.currentPhase)
    }

    tsm.queuedActions = append(tsm.queuedActions, action)
    tsm.currentPhase = PhasePlanning
    return nil
}

// ConfirmActions validates and executes all queued actions
func (tsm *TurnStateMachine) ConfirmActions(gameState *GameState) error {
    if tsm.currentPhase != PhasePlanning {
        return fmt.Errorf("cannot confirm actions during %v phase", tsm.currentPhase)
    }

    // Transition to validation phase
    tsm.currentPhase = PhaseValidating
    tsm.validationErrs = make([]error, 0)

    // Validate all actions
    for _, action := range tsm.queuedActions {
        if err := action.Validate(gameState); err != nil {
            tsm.validationErrs = append(tsm.validationErrs, err)
        }
    }

    // If any validation errors, stay in planning phase
    if len(tsm.validationErrs) > 0 {
        tsm.currentPhase = PhasePlanning
        return fmt.Errorf("validation failed: %d errors", len(tsm.validationErrs))
    }

    // Transition to execution phase
    tsm.currentPhase = PhaseExecuting
    tsm.executionLog = make([]TurnActionResult, 0)

    // Execute all actions in order
    for _, action := range tsm.queuedActions {
        result := action.Execute(gameState)
        tsm.executionLog = append(tsm.executionLog, result)
    }

    // Transition to resolution phase (trigger abilities, etc.)
    tsm.currentPhase = PhaseResolving
    tsm.resolveEffects(gameState)

    // Clear queued actions
    tsm.queuedActions = make([]TurnAction, 0)

    // Ready for next turn
    tsm.currentPhase = PhaseNextTurn

    return nil
}

// resolveEffects handles side effects (ability triggers, status effects)
func (tsm *TurnStateMachine) resolveEffects(gameState *GameState) {
    // Trigger "on turn end" abilities
    for _, result := range tsm.executionLog {
        if result.TriggerAbilities {
            squads.CheckAndTriggerAbilities(result.ActorID, gameState.EntityManager)
        }
    }
}

// NextTurn advances to next faction's turn
func (tsm *TurnStateMachine) NextTurn(gameState *GameState) error {
    if tsm.currentPhase != PhaseNextTurn {
        return fmt.Errorf("cannot advance turn during %v phase", tsm.currentPhase)
    }

    // Advance turn in turn manager
    gameState.TurnManager.EndTurn()

    // Get new faction
    tsm.currentFaction = gameState.TurnManager.GetCurrentFaction()
    tsm.currentRound = gameState.TurnManager.GetCurrentRound()

    // Reset to waiting for input
    tsm.currentPhase = PhaseWaitingForInput
    tsm.executionLog = make([]TurnActionResult, 0)

    return nil
}

// CancelActions clears queued actions and returns to input phase
func (tsm *TurnStateMachine) CancelActions() {
    tsm.queuedActions = make([]TurnAction, 0)
    tsm.currentPhase = PhaseWaitingForInput
}

// GetCurrentPhase returns current turn phase
func (tsm *TurnStateMachine) GetCurrentPhase() TurnPhase {
    return tsm.currentPhase
}

// GetQueuedActions returns actions queued for this turn
func (tsm *TurnStateMachine) GetQueuedActions() []TurnAction {
    return tsm.queuedActions
}
```

*After - Turn Action Interface (game/turn_actions.go):*
```go
package game

// TurnAction represents an action that can be performed during a turn
type TurnAction interface {
    Validate(gameState *GameState) error
    Execute(gameState *GameState) TurnActionResult
    GetActorID() ecs.EntityID
    GetActionType() string
}

// TurnActionResult contains result of executing an action
type TurnActionResult struct {
    Success          bool
    ActorID          ecs.EntityID
    ActionType       string
    Description      string
    TriggerAbilities bool  // Should abilities be checked after this action?
}

// AttackAction represents a squad attacking another squad
type AttackAction struct {
    AttackerID ecs.EntityID
    TargetID   ecs.EntityID
}

func (a *AttackAction) GetActorID() ecs.EntityID {
    return a.AttackerID
}

func (a *AttackAction) GetActionType() string {
    return "Attack"
}

func (a *AttackAction) Validate(gameState *GameState) error {
    combatSys := combat.NewCombatActionSystem(gameState.EntityManager)
    reason, canAttack := combatSys.CanSquadAttackWithReason(a.AttackerID, a.TargetID)
    if !canAttack {
        return fmt.Errorf("cannot attack: %s", reason)
    }

    // Check if squad has already acted this turn
    actionState := findActionStateEntity(a.AttackerID, gameState.EntityManager)
    if actionState != nil {
        actionData := common.GetComponentType[*combat.ActionStateData](actionState, combat.ActionStateComponent)
        if actionData.HasActed {
            return fmt.Errorf("squad has already acted this turn")
        }
    }

    return nil
}

func (a *AttackAction) Execute(gameState *GameState) TurnActionResult {
    combatSys := combat.NewCombatActionSystem(gameState.EntityManager)

    attackerName := getSquadName(a.AttackerID, gameState.EntityManager)
    targetName := getSquadName(a.TargetID, gameState.EntityManager)

    err := combatSys.ExecuteAttackAction(a.AttackerID, a.TargetID)
    if err != nil {
        return TurnActionResult{
            Success:     false,
            ActorID:     a.AttackerID,
            ActionType:  "Attack",
            Description: fmt.Sprintf("Attack failed: %v", err),
        }
    }

    destroyed := squads.IsSquadDestroyed(a.TargetID, gameState.EntityManager)

    desc := fmt.Sprintf("%s attacked %s!", attackerName, targetName)
    if destroyed {
        desc += fmt.Sprintf(" %s was destroyed!", targetName)
    }

    return TurnActionResult{
        Success:          true,
        ActorID:          a.AttackerID,
        ActionType:       "Attack",
        Description:      desc,
        TriggerAbilities: true,  // Check for combat abilities
    }
}

// MoveAction represents squad movement
type MoveAction struct {
    SquadID     ecs.EntityID
    Destination coords.LogicalPosition
}

func (m *MoveAction) GetActorID() ecs.EntityID {
    return m.SquadID
}

func (m *MoveAction) GetActionType() string {
    return "Move"
}

func (m *MoveAction) Validate(gameState *GameState) error {
    validTiles := gameState.MovementSystem.GetValidMovementTiles(m.SquadID)
    for _, tile := range validTiles {
        if tile == m.Destination {
            return nil
        }
    }
    return fmt.Errorf("invalid movement destination")
}

func (m *MoveAction) Execute(gameState *GameState) TurnActionResult {
    err := gameState.MovementSystem.MoveSquad(m.SquadID, m.Destination)
    if err != nil {
        return TurnActionResult{
            Success:     false,
            ActorID:     m.SquadID,
            ActionType:  "Move",
            Description: fmt.Sprintf("Movement failed: %v", err),
        }
    }

    // Update unit positions to match squad
    updateUnitPositions(m.SquadID, m.Destination, gameState.EntityManager)

    squadName := getSquadName(m.SquadID, gameState.EntityManager)

    return TurnActionResult{
        Success:     true,
        ActorID:     m.SquadID,
        ActionType:  "Move",
        Description: fmt.Sprintf("%s moved to (%d, %d)", squadName, m.Destination.X, m.Destination.Y),
    }
}
```

*After - UI Integration (combat_action_handler.go):*
```go
type CombatActionHandler struct {
    battleMapState *core.BattleMapState
    logManager     *CombatLogManager
    queries        *guicomponents.GUIQueries
    turnStateMachine *game.TurnStateMachine
    gameState      *game.GameState
    combatLogArea  *widget.TextArea
}

func (cah *CombatActionHandler) executeAttack() {
    selectedSquad := cah.battleMapState.SelectedSquadID
    selectedTarget := cah.battleMapState.SelectedTargetID

    if selectedSquad == 0 || selectedTarget == 0 {
        return
    }

    // Queue attack action instead of executing immediately
    action := &game.AttackAction{
        AttackerID: selectedSquad,
        TargetID:   selectedTarget,
    }

    err := cah.turnStateMachine.QueueAction(action)
    if err != nil {
        cah.addLog(fmt.Sprintf("Cannot queue attack: %v", err))
        return
    }

    cah.addLog(fmt.Sprintf("Attack queued! Press SPACE to execute turn."))
    cah.battleMapState.InAttackMode = false
}

func (cah *CombatActionHandler) MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) error {
    // Queue move action
    action := &game.MoveAction{
        SquadID:     squadID,
        Destination: newPos,
    }

    err := cah.turnStateMachine.QueueAction(action)
    if err != nil {
        cah.addLog(fmt.Sprintf("Cannot queue move: %v", err))
        return err
    }

    cah.addLog(fmt.Sprintf("Movement queued! Press SPACE to execute turn."))
    cah.battleMapState.InMoveMode = false
    cah.battleMapState.ValidMoveTiles = []coords.LogicalPosition{}

    return nil
}

func (cah *CombatActionHandler) ExecuteTurn() {
    // Confirm all queued actions - this executes them
    err := cah.turnStateMachine.ConfirmActions(cah.gameState)
    if err != nil {
        cah.addLog(fmt.Sprintf("Turn execution failed: %v", err))
        return
    }

    // Log results
    for _, result := range cah.turnStateMachine.executionLog {
        cah.addLog(result.Description)
    }

    cah.addLog("Turn executed! Press E to end turn.")
}

func (cah *CombatActionHandler) CancelActions() {
    cah.turnStateMachine.CancelActions()
    cah.addLog("All queued actions cancelled.")
}
```

**Key Changes**:
- Created TurnStateMachine with explicit phases (WaitingForInput, Planning, Validating, Executing, Resolving, NextTurn)
- Actions are queued during Planning phase, not executed immediately
- ConfirmActions() validates all actions, then executes in sequence
- Resolution phase handles side effects (ability triggers, status effects)
- UI shows queued actions, player can cancel before confirming
- Turn flow is explicit and testable

**Value Proposition**:
- **Maintainability**: Turn logic is centralized in state machine, not scattered across UI
- **Readability**: Turn phases are explicit - clear what can happen when
- **Extensibility**: Easy to add new turn features (simultaneous resolution, action points, time limits)
- **Complexity Impact**:
  - Adds ~600 lines (state machine, turn actions)
  - Simplifies UI handlers (just queue actions, no immediate execution)
  - Net ~+200 lines but massively better turn structure
- **Game-Specific Benefits**:
  - Can implement "plan then execute" gameplay (XCOM-style)
  - Can add action point costs (each action reduces pool)
  - Can enforce action order rules (must move before attack for cavalry)
  - Can implement simultaneous turns (both sides plan, then resolve)

**Implementation Strategy**:
1. **Phase 1 - State Machine** (2 days):
   - Create TurnStateMachine with phase enum and transitions
   - Implement QueueAction, ConfirmActions, CancelActions, NextTurn methods
   - Test phase transitions in isolation

2. **Phase 2 - Turn Actions** (2 days):
   - Implement TurnAction interface
   - Create AttackAction, MoveAction, EndTurnAction
   - Test validation and execution for each action

3. **Phase 3 - Game State Integration** (1 day):
   - Create GameState that owns TurnStateMachine and game systems
   - Move system ownership from CombatMode to GameState
   - Wire up turn manager to state machine

4. **Phase 4 - UI Integration** (2 days):
   - Update CombatActionHandler to queue actions instead of execute
   - Add "Execute Turn" button to UI
   - Show queued actions in UI panel
   - Add "Cancel Actions" button

5. **Phase 5 - Polish** (1 day):
   - Add visual feedback for queued actions (highlight in list)
   - Show turn phase in UI (Planning, Executing, etc.)
   - Add hotkeys (Space = execute turn, ESC = cancel actions)
   - Test full turn flow end-to-end

**Advantages**:
- **Explicit turn structure**: No ambiguity about what phase you're in
- **Plan before execute**: Player can queue multiple actions, see them, then confirm
- **Turn validation**: Can enforce complex rules (action costs, ordering, etc.)
- **Testable turn logic**: Can create action sequences, execute turn, validate state
- **Enables advanced features**:
  - Simultaneous resolution (both sides plan, resolve together)
  - Action point system (each action costs points from pool)
  - Time-limited turns (30 second timer during Planning phase)
  - Undo during planning (cancel queued actions before execute)
  - Turn replay (save action sequence, replay later)

**Drawbacks & Risks**:
- **Gameplay change**: Queueing actions feels different from immediate execution
  - *Mitigation*: Make it optional - "quick mode" executes immediately, "tactical mode" queues actions
- **More complex UI**: Need to show queued actions, execute button, cancel button
  - *Mitigation*: Design clear UI panel for action queue with visual feedback
- **Potential confusion**: Players may not understand queue vs execute concept
  - *Mitigation*: Clear tutorial, visual feedback, tooltips explaining phases
- **Overkill for simple turns**: If most turns are "move then attack," queueing adds friction
  - *Mitigation*: Auto-execute if only one action queued (skip confirmation step)

**Effort Estimate**:
- **Time**: 8 days (1.5 weeks)
- **Complexity**: Medium-High
- **Risk**: Medium (changes gameplay feel, UI needs redesign)
- **Files Impacted**:
  - New: 5 files (turn_state_machine.go, turn_actions.go, game_state_manager.go)
  - Modified: 10 files (combatmode.go, combat_action_handler.go, combat_input_handler.go, combatui panels for action queue)

**Critical Assessment** (from refactoring-critic):
This is **excellent for turn-based strategy games** but requires careful consideration of gameplay impact. The state machine provides clear structure and enables powerful features (simultaneous resolution, action planning). However, it **changes how the game feels** - actions are no longer immediate. This could be good (more strategic, plan ahead) or bad (feels sluggish, extra clicks).

**Recommendation**: Only choose this if:
1. You want "plan then execute" gameplay (XCOM, Into the Breach style)
2. You're implementing simultaneous turns or action points
3. You need turn replay or complex turn validation

If you want immediate action execution (click attack, it happens now), this adds unnecessary friction. For immediate execution, choose Approach 1 (Service Layer) instead.

---

## COMPARATIVE ANALYSIS OF FINAL APPROACHES

### Effort vs Impact Matrix
| Approach | Effort | Impact | Risk | Recommended Priority |
|----------|--------|--------|------|---------------------|
| Approach 1: Service Layer | Low (1 week) | High (immediate testability) | Low (wrapping existing code) | **1** (Start here) |
| Approach 2: Command/Event | High (2 weeks) | Very High (complete separation, undo/replay) | Medium (new patterns, learning curve) | **3** (Only if needed) |
| Approach 3: Turn State Machine | Medium (1.5 weeks) | High (turn structure, advanced features) | Medium (gameplay feel change) | **2** (If turn-based features needed) |

### Decision Guidance

**Choose Approach 1 (Service Layer) if:**
- You need testable business logic RIGHT NOW
- You want lowest risk, incremental migration path
- You don't need undo/replay/networking capabilities yet
- You want to preserve current gameplay feel (immediate action execution)
- Your team is new to refactoring and needs familiar patterns
- **Recommendation**: **Start here for 95% of cases**

**Choose Approach 2 (Command/Event) if:**
- You're planning to implement undo/redo within 6 months
- You're considering multiplayer or networking features
- You need to test complex AI scenarios programmatically
- You want replay capability for debugging or showcasing
- You're comfortable with higher complexity for future-proofing
- **Recommendation**: Migrate TO this from Approach 1 when you actually need the capabilities

**Choose Approach 3 (Turn State Machine) if:**
- You want "plan then execute" gameplay (XCOM, Into the Breach style)
- You're implementing action point system or simultaneous turns
- You need complex turn validation (action ordering, costs, etc.)
- You want players to queue multiple actions before executing
- You're building a tactics-heavy game where planning is core mechanic
- **Recommendation**: Only if this aligns with your game design vision

### Combination Opportunities

**Best Migration Path (Recommended)**:
1. **Start with Approach 1** (Service Layer) - 1 week
   - Get immediate testability
   - Extract game logic from UI
   - Low risk, fast value

2. **Add Approach 3** (Turn State Machine) if needed - 1 week
   - Keep services, add state machine on top
   - Services become action executors for state machine
   - Enables turn-based features without losing testability

3. **Evolve to Approach 2** (Command/Event) if needed - 1 week
   - Turn state machine actions already look like commands
   - Add event bus for UI updates
   - Enables undo/replay/networking

**Why this path works**:
- Approach 1 services are not wasted - they become the execution layer
- Approach 3 state machine builds on services (calls service methods)
- Approach 2 command pattern wraps state machine (commands execute turn actions)
- Each step adds value independently, not throwaway work

**Combined Architecture Diagram**:
```
UI Layer (gui/)
    ↓ (calls methods)
Services (combat_service, squad_service) ← Start here (Approach 1)
    ↓ (calls ECS systems)
ECS Systems (combat/, squads/)
    ↓ (manipulates)
ECS Components (pure data)

--- If turn-based features needed, add: ---
UI Layer
    ↓ (queues actions)
Turn State Machine (Approach 3)
    ↓ (executes via)
Services (Approach 1)
    ↓ (calls)
ECS Systems

--- If undo/replay/networking needed, add: ---
UI Layer
    ↓ (creates commands)
Command Layer (Approach 2)
    ↓ (executes via)
Turn State Machine (Approach 3)
    ↓ (calls)
Services (Approach 1)
    ↓ (calls)
ECS Systems
```

---

## APPENDIX: INITIAL APPROACHES FROM ALL AGENTS

### A. Refactoring-Pro Approaches

#### Refactoring-Pro Approach 1: Service Layer Extraction
**Focus**: Pragmatic separation with minimal architectural change

**Problem**: CombatMode owns game systems (TurnManager line 42, FactionManager line 43, MovementSystem line 44 in combatmode.go). CombatActionHandler mixes business logic with UI updates (executeAttack lines 167-204 in combat_action_handler.go). Cannot test combat logic without UI.

**Solution**: Create CombatService, SquadService, MovementService that own game systems. Inject services into UI handlers. Extract execute methods to service methods that return result objects. UI interprets results and updates display.

**Code Example**: (See Final Approach 1 above - this was synthesized into final approach)

**Metrics**:
- Lines removed from UI: ~800
- Lines added to services: ~500
- Net: -300 lines
- Cyclomatic complexity reduction: UI handlers from 15+ to <5
- Test coverage enabled: 0% → 80% for business logic

**Assessment**:
- **Pros**: Immediate testability, low risk, familiar pattern, incremental migration
- **Cons**: Services still tightly coupled to ECS, no undo/replay capability
- **Effort**: 5 days (1 week)

---

#### Refactoring-Pro Approach 2: Facade Pattern with Game Session Manager
**Focus**: Single access point for all game operations

**Problem**: UI code calls multiple game systems directly (TurnManager, FactionManager, MovementSystem, CombatActionSystem). Creates tight coupling - UI knows too much about game internals. Hard to change game architecture without breaking UI.

**Solution**: Create GameSessionManager facade that provides unified interface to all game operations. UI calls facade methods like `session.AttackSquad(attackerID, targetID)` instead of creating combat systems, checking validation, executing, etc. Facade coordinates multiple systems behind simple API.

**Code Example**:
```go
// game/game_session_manager.go
type GameSessionManager struct {
    entityManager  *common.EntityManager
    turnManager    *combat.TurnManager
    factionManager *combat.FactionManager
    movementSystem *combat.MovementSystem
}

func NewGameSessionManager(manager *common.EntityManager) *GameSessionManager {
    return &GameSessionManager{
        entityManager:  manager,
        turnManager:    combat.NewTurnManager(manager),
        factionManager: combat.NewFactionManager(manager),
        movementSystem: combat.NewMovementSystem(manager, common.GlobalPositionSystem),
    }
}

// Single method for attacking - hides all complexity
func (gsm *GameSessionManager) AttackSquad(attackerID, targetID ecs.EntityID) error {
    combatSys := combat.NewCombatActionSystem(gsm.entityManager)

    if reason, canAttack := combatSys.CanSquadAttackWithReason(attackerID, targetID); !canAttack {
        return fmt.Errorf("cannot attack: %s", reason)
    }

    if err := combatSys.ExecuteAttackAction(attackerID, targetID); err != nil {
        return err
    }

    return nil
}

// Single method for moving - hides complexity
func (gsm *GameSessionManager) MoveSquad(squadID ecs.EntityID, destination coords.LogicalPosition) error {
    if err := gsm.movementSystem.MoveSquad(squadID, destination); err != nil {
        return err
    }
    return nil
}

// Query methods
func (gsm *GameSessionManager) GetCurrentFaction() ecs.EntityID {
    return gsm.turnManager.GetCurrentFaction()
}

func (gsm *GameSessionManager) GetValidMoveTiles(squadID ecs.EntityID) []coords.LogicalPosition {
    return gsm.movementSystem.GetValidMovementTiles(squadID)
}
```

**Metrics**:
- UI coupling reduced: Calls 1 facade instead of 4-5 systems
- API surface area: ~20 methods vs 50+ across systems
- Lines in facade: ~400
- Lines removed from UI: ~300 (no system creation, coordination)

**Assessment**:
- **Pros**: Simple API for UI, hides game complexity, easy to understand
- **Cons**: Facade could become bloated "God Object," doesn't improve testability much
- **Effort**: 4 days

---

#### Refactoring-Pro Approach 3: Repository Pattern for Game State Access
**Focus**: Separate game state queries from commands

**Problem**: UI queries game state directly (GetSquadName, GetCurrentFaction, GetValidMoveTiles) AND executes commands (ExecuteAttack, MoveSquad). This violates Command Query Separation (CQS) - queries should not have side effects, commands should not return data. Makes it hard to cache queries or audit commands.

**Solution**: Create GameStateRepository for all queries (read-only) and GameCommandExecutor for all commands (write). UI uses repository for display updates, command executor for actions. This separates reads from writes, enables query caching, command logging.

**Code Example**:
```go
// game/game_state_repository.go - READ ONLY
type GameStateRepository struct {
    entityManager  *common.EntityManager
    turnManager    *combat.TurnManager
    factionManager *combat.FactionManager

    // Query caches (optional optimization)
    squadNamesCache map[ecs.EntityID]string
}

func NewGameStateRepository(manager *common.EntityManager, turnMgr *combat.TurnManager, factionMgr *combat.FactionManager) *GameStateRepository {
    return &GameStateRepository{
        entityManager:   manager,
        turnManager:     turnMgr,
        factionManager:  factionMgr,
        squadNamesCache: make(map[ecs.EntityID]string),
    }
}

// All query methods - no side effects
func (gsr *GameStateRepository) GetSquadName(squadID ecs.EntityID) string {
    if name, cached := gsr.squadNamesCache[squadID]; cached {
        return name
    }

    name := getSquadNameByID(squadID, gsr.entityManager)
    gsr.squadNamesCache[squadID] = name
    return name
}

func (gsr *GameStateRepository) GetCurrentFaction() ecs.EntityID {
    return gsr.turnManager.GetCurrentFaction()
}

func (gsr *GameStateRepository) GetValidMoveTiles(squadID ecs.EntityID) []coords.LogicalPosition {
    moveSys := combat.NewMovementSystem(gsr.entityManager, common.GlobalPositionSystem)
    return moveSys.GetValidMovementTiles(squadID)
}

func (gsr *GameStateRepository) GetEnemySquads(factionID ecs.EntityID) []ecs.EntityID {
    // Query implementation
}

// game/game_command_executor.go - WRITE ONLY
type GameCommandExecutor struct {
    entityManager  *common.EntityManager
    turnManager    *combat.TurnManager
    movementSystem *combat.MovementSystem

    commandLog []ExecutedCommand  // Audit log
}

type ExecutedCommand struct {
    CommandType string
    Timestamp   time.Time
    ActorID     ecs.EntityID
    Success     bool
    Error       error
}

func NewGameCommandExecutor(manager *common.EntityManager, turnMgr *combat.TurnManager) *GameCommandExecutor {
    return &GameCommandExecutor{
        entityManager:  manager,
        turnManager:    turnMgr,
        movementSystem: combat.NewMovementSystem(manager, common.GlobalPositionSystem),
        commandLog:     make([]ExecutedCommand, 0),
    }
}

// All command methods - have side effects, return errors only
func (gce *GameCommandExecutor) ExecuteAttack(attackerID, targetID ecs.EntityID) error {
    combatSys := combat.NewCombatActionSystem(gce.entityManager)

    reason, canAttack := combatSys.CanSquadAttackWithReason(attackerID, targetID)
    if !canAttack {
        gce.logCommand("Attack", attackerID, false, fmt.Errorf(reason))
        return fmt.Errorf("cannot attack: %s", reason)
    }

    err := combatSys.ExecuteAttackAction(attackerID, targetID)
    gce.logCommand("Attack", attackerID, err == nil, err)

    return err
}

func (gce *GameCommandExecutor) ExecuteMove(squadID ecs.EntityID, destination coords.LogicalPosition) error {
    err := gce.movementSystem.MoveSquad(squadID, destination)
    gce.logCommand("Move", squadID, err == nil, err)
    return err
}

func (gce *GameCommandExecutor) logCommand(cmdType string, actorID ecs.EntityID, success bool, err error) {
    gce.commandLog = append(gce.commandLog, ExecutedCommand{
        CommandType: cmdType,
        Timestamp:   time.Now(),
        ActorID:     actorID,
        Success:     success,
        Error:       err,
    })
}
```

**Metrics**:
- Separation: 100% queries in repository, 100% commands in executor
- Command logging: Automatic audit trail of all actions
- Query caching: Potential performance improvement for repeated queries
- Lines added: ~600 (repository + executor)

**Assessment**:
- **Pros**: Clear CQS separation, enables query caching, automatic command audit log
- **Cons**: More classes to manage, query cache invalidation complexity, UI needs both repository AND executor
- **Effort**: 6 days

---

### B. Tactical-Simplifier Approaches

#### Tactical-Simplifier Approach 1: Game Loop Integration with Tick-Based Processing
**Focus**: Align game logic with game loop (Update/Render), separate tick-based logic from UI events

**Problem**: Combat logic executes in UI event handlers (button clicks, key presses) which is outside the game loop. This creates timing issues - actions can happen at any time, not synchronized with game Update() cycle. Can't implement time-based features (cooldowns, status effects with durations) because there's no consistent "game tick."

**Solution**: Move game logic execution to GameState.Update() method called from main game loop. UI events queue intents (AttackIntent, MoveIntent), Update() processes intents and executes game logic. This synchronizes all game logic with frame updates, enabling time-based features.

**Code Example**:
```go
// game/game_state.go
type GameState struct {
    entityManager  *common.EntityManager
    turnManager    *combat.TurnManager

    pendingIntents []GameIntent  // Queued from UI events
    tickCount      int           // Game ticks elapsed
}

type GameIntent interface {
    GetType() string
    GetActorID() ecs.EntityID
}

type AttackIntent struct {
    AttackerID ecs.EntityID
    TargetID   ecs.EntityID
}

func (ai *AttackIntent) GetType() string { return "Attack" }
func (ai *AttackIntent) GetActorID() ecs.EntityID { return ai.AttackerID }

// Called from main game loop every frame
func (gs *GameState) Update(deltaTime float64) {
    gs.tickCount++

    // Process pending intents from UI
    for _, intent := range gs.pendingIntents {
        gs.processIntent(intent)
    }
    gs.pendingIntents = make([]GameIntent, 0)

    // Update status effects, cooldowns, etc.
    gs.updateStatusEffects(deltaTime)
    gs.updateCooldowns(deltaTime)
}

func (gs *GameState) processIntent(intent GameIntent) {
    switch intent.GetType() {
    case "Attack":
        attackIntent := intent.(*AttackIntent)
        gs.executeAttack(attackIntent.AttackerID, attackIntent.TargetID)
    case "Move":
        moveIntent := intent.(*MoveIntent)
        gs.executeMove(moveIntent.SquadID, moveIntent.Destination)
    }
}

// UI queues intents, doesn't execute directly
func (gs *GameState) QueueAttackIntent(attackerID, targetID ecs.EntityID) {
    gs.pendingIntents = append(gs.pendingIntents, &AttackIntent{
        AttackerID: attackerID,
        TargetID:   targetID,
    })
}
```

**Gameplay Preservation**: Turn-based gameplay intact - intents queue instantly, execute on next tick (within 16ms at 60fps, imperceptible to player).

**Go-Specific Optimizations**: Intent interface instead of polymorphism, type switch instead of virtual dispatch.

**Game System Impact**:
- **Combat**: Can add status effects with durations (2 turn stun = tickCount + 2*60)
- **Entity**: Can add cooldown components (ability on cooldown for N ticks)
- **Rendering**: No change, still renders current state

**Assessment**:
- **Pros**: Enables time-based features, synchronizes with game loop, clear intent queue
- **Cons**: Adds one frame latency (queue on frame N, execute frame N+1), intent queue complexity
- **Effort**: 5 days

---

#### Tactical-Simplifier Approach 2: Tactical State Manager with Formation Preservation
**Focus**: Preserve formation integrity and tactical positioning during refactoring

**Problem**: Squad formations are defined in squads/ package but UI directly manipulates positions during movement/deployment. This can break formation rules (units must stay in formation shape). When UI handler calls MovementSystem.MoveSquad(), it only moves squad anchor - UI is responsible for updating unit positions (see combat_action_handler.go lines 265-281). This is error-prone and violates encapsulation.

**Solution**: Create TacticalStateManager that ensures all tactical rules (formations, positioning, facings) are enforced when game logic executes. UI never manipulates positions directly - calls TacticalStateManager.MoveSquadInFormation() which handles squad + all units + formation preservation.

**Code Example**:
```go
// game/tactical_state_manager.go
type TacticalStateManager struct {
    entityManager  *common.EntityManager
    movementSystem *combat.MovementSystem
}

// MoveSquadInFormation moves squad AND all units, preserving formation
func (tsm *TacticalStateManager) MoveSquadInFormation(squadID ecs.EntityID, newPos coords.LogicalPosition) error {
    // Move squad anchor
    if err := tsm.movementSystem.MoveSquad(squadID, newPos); err != nil {
        return err
    }

    // Get formation data
    squadEntity := squads.GetSquadEntity(squadID, tsm.entityManager)
    squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)

    // Update all unit positions based on formation
    unitIDs := squads.GetUnitIDsInSquad(squadID, tsm.entityManager)
    for _, unitID := range unitIDs {
        // Get unit's grid position within formation
        gridPos := common.GetComponentTypeByIDWithTag[*squads.GridPositionData](
            tsm.entityManager, unitID, squads.SquadMemberTag, squads.GridPositionComponent)

        // Calculate world position from squad position + grid offset
        unitWorldPos := tsm.calculateFormationPosition(newPos, squadData.Formation, gridPos)

        // Update unit position
        unitEntity := common.FindEntityByIDWithTag(tsm.entityManager, unitID, squads.SquadMemberTag)
        if unitEntity.HasComponent(common.PositionComponent) {
            posPtr := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
            posPtr.X = unitWorldPos.X
            posPtr.Y = unitWorldPos.Y

            // Update spatial grid
            common.GlobalPositionSystem.RemoveEntity(unitID, *posPtr)
            common.GlobalPositionSystem.AddEntity(unitID, unitWorldPos)
        }
    }

    return nil
}

func (tsm *TacticalStateManager) calculateFormationPosition(
    squadPos coords.LogicalPosition,
    formation squads.FormationType,
    gridPos *squads.GridPositionData,
) coords.LogicalPosition {
    // Apply formation offsets based on type (line, column, box, etc.)
    // This encapsulates formation logic instead of spreading it across UI
    // ...
}

// DeploySquad places squad on battlefield with formation
func (tsm *TacticalStateManager) DeploySquad(squadID ecs.EntityID, deployPos coords.LogicalPosition) error {
    // Validate deployment position
    if !tsm.isValidDeploymentPosition(deployPos) {
        return fmt.Errorf("invalid deployment position")
    }

    // Move squad to position
    return tsm.MoveSquadInFormation(squadID, deployPos)
}
```

**Gameplay Preservation**: Formations work correctly, units stay in formation during movement (this was buggy before when UI handled it).

**Go-Specific Optimizations**: Struct composition (TacticalStateManager composes MovementSystem), value-based position calculations.

**Game System Impact**:
- **Combat**: Formations preserved during movement, improves tactical positioning
- **Entity**: No changes to ECS, just better encapsulation
- **Rendering**: Formations render correctly after movement

**Assessment**:
- **Pros**: Fixes formation bugs, encapsulates tactical logic, UI simpler (one method call)
- **Cons**: Another manager class, formation logic complexity
- **Effort**: 4 days

---

#### Tactical-Simplifier Approach 3: Pure Functional Combat Resolver
**Focus**: Make combat resolution pure function (given state, return new state) for testability

**Problem**: Combat execution (ExecuteAttackAction in combat/combatactions.go) directly mutates ECS components. This makes it impossible to test "what would happen if X attacks Y" without actually modifying game state. Can't implement AI that evaluates multiple attack options.

**Solution**: Create pure functional combat resolver that takes immutable input (attacker stats, defender stats) and returns combat result (damage, status effects, etc.). Separate "compute combat outcome" from "apply outcome to ECS." This enables testing, AI scenario evaluation, and combat preview.

**Code Example**:
```go
// combat/combat_resolver.go - PURE FUNCTIONS
type CombatInput struct {
    AttackerStats Attributes
    DefenderStats Attributes
    AttackerFormation squads.FormationType
    DefenderFormation squads.FormationType
    // Immutable snapshot of relevant game state
}

type CombatOutcome struct {
    DamageToDefender  int
    DamageToAttacker  int // Counter-attack damage
    DefenderDestroyed bool
    AttackerDestroyed bool
    StatusEffects     []StatusEffect
}

type StatusEffect struct {
    TargetID  ecs.EntityID
    EffectType string
    Duration   int
}

// Pure function - no side effects, same input always produces same output
func ResolveCombat(input CombatInput) CombatOutcome {
    outcome := CombatOutcome{}

    // Calculate damage based on stats and formations
    baseDamage := calculateBaseDamage(input.AttackerStats, input.DefenderStats)
    formationBonus := getFormationBonus(input.AttackerFormation, input.DefenderFormation)
    finalDamage := baseDamage + formationBonus

    outcome.DamageToDefender = finalDamage

    // Calculate counter-attack if defender survives
    if input.DefenderStats.GetMaxHealth() > finalDamage {
        counterDamage := calculateCounterDamage(input.DefenderStats, input.AttackerStats)
        outcome.DamageToAttacker = counterDamage
    }

    // Determine destruction
    outcome.DefenderDestroyed = input.DefenderStats.GetMaxHealth() <= finalDamage
    outcome.AttackerDestroyed = input.AttackerStats.GetMaxHealth() <= outcome.DamageToAttacker

    return outcome
}

// Separate function to APPLY outcome to ECS (has side effects)
func ApplyCombatOutcome(outcome CombatOutcome, attackerID, defenderID ecs.EntityID, manager *common.EntityManager) error {
    // Apply damage to defender
    defenderAttrs := common.GetAttributesByID(manager, defenderID)
    defenderAttrs.CurrentHealth -= outcome.DamageToDefender

    // Apply counter-attack damage to attacker
    if outcome.DamageToAttacker > 0 {
        attackerAttrs := common.GetAttributesByID(manager, attackerID)
        attackerAttrs.CurrentHealth -= outcome.DamageToAttacker
    }

    // Destroy entities if needed
    if outcome.DefenderDestroyed {
        squads.DestroySquad(defenderID, manager)
    }
    if outcome.AttackerDestroyed {
        squads.DestroySquad(attackerID, manager)
    }

    return nil
}

// High-level function that combines resolve + apply
func ExecuteAttack(attackerID, defenderID ecs.EntityID, manager *common.EntityManager) (CombatOutcome, error) {
    // Gather input
    input := CombatInput{
        AttackerStats: *common.GetAttributesByID(manager, attackerID),
        DefenderStats: *common.GetAttributesByID(manager, defenderID),
        // ... get formations ...
    }

    // Resolve combat (pure)
    outcome := ResolveCombat(input)

    // Apply outcome (side effects)
    err := ApplyCombatOutcome(outcome, attackerID, defenderID, manager)

    return outcome, err
}
```

**Usage in UI**:
```go
// UI can preview combat without executing
input := buildCombatInput(attackerID, targetID, gameState)
outcome := combat.ResolveCombat(input)  // Pure, no side effects
displayCombatPreview(outcome)  // Show "This attack will deal X damage"

// Then execute if player confirms
if playerConfirms {
    combat.ExecuteAttack(attackerID, targetID, manager)
}
```

**Gameplay Preservation**: Combat mechanics unchanged, just separated computation from application.

**Go-Specific Optimizations**: Value-based input struct (pass by value, immutable), separate pure and impure functions.

**Game System Impact**:
- **Combat**: Can preview attacks, test outcomes, evaluate AI options
- **Entity**: No changes to ECS
- **Graphics**: Can show combat preview UI

**Assessment**:
- **Pros**: Testable combat (unit test ResolveCombat), enables preview, AI can evaluate options
- **Cons**: More verbose (separate resolve + apply), need to snapshot state for input
- **Effort**: 5 days

---

### C. Refactoring-Critic Evaluation

After reviewing all 6 approaches from refactoring-pro and tactical-simplifier perspectives, here's the critical synthesis:

**Strengths Across Approaches**:
- All recognize need to separate UI from business logic
- All enable better testing (current = 0% coverage)
- All respect ECS architecture (don't violate component patterns)
- All provide concrete code examples with real file references

**Weaknesses & Over-Engineering Risks**:
1. **Refactoring-Pro Approach 2** (Facade Pattern): Facade risks becoming God Object. Doesn't improve testability much over Approach 1 (Service Layer). Main benefit is simpler API, but that's achieved better with Service Layer that splits by domain.

2. **Refactoring-Pro Approach 3** (Repository/Command Executor): CQS separation is theoretically beautiful but adds complexity without clear value. Caching queries sounds good but ECS queries are already fast. Command logging could be added to Service Layer approach without separate class.

3. **Tactical-Simplifier Approach 1** (Tick-Based Processing): Adds one frame latency for all actions. For turn-based game, this is unnecessary - actions don't need tick synchronization. Better suited for real-time games with cooldowns/status effects.

4. **Tactical-Simplifier Approach 2** (Tactical State Manager): Good idea (encapsulate formation logic) but too narrow. This is really a subset of Service Layer approach - just make it SquadService.MoveSquadInFormation() instead of whole new manager.

5. **Tactical-Simplifier Approach 3** (Pure Functional Resolver): Excellent for testing but verbose. Separating resolve/apply is right idea but creates duplication. Could achieve same testability with well-designed service methods that return results before applying.

**Best Elements to Synthesize**:
- **Service Layer core** (Refactoring-Pro 1): Immediate testability, low risk, familiar
- **Turn State Machine** (Tactical-Simplifier implicit): Explicit turn phases enable advanced features
- **Pure combat resolution** (Tactical-Simplifier 3): Separate compute from apply for testability
- **Command pattern** (Refactoring-Pro 2 hinted): If we need undo/replay, add commands later
- **Event system** (Refactoring-Pro 2 hinted): If we need decoupled UI updates, add events later

**Final Recommendations**:
1. **Approach 1** (Service Layer): Start here - 90% of value, 10% of risk
2. **Approach 2** (Command/Event): Migrate to this ONLY if you need undo/replay/networking
3. **Approach 3** (Turn State Machine): Add if you want "plan then execute" gameplay

**Anti-Patterns to Avoid**:
- Don't create Facade that just delegates to services (use services directly)
- Don't separate Repository/CommandExecutor (Service Layer is sufficient)
- Don't add tick-based processing for turn-based game (unnecessary complexity)
- Don't create separate TacticalStateManager (fold into SquadService)
- Don't implement undo/replay/networking "just in case" (YAGNI violation)

---

## SYNTHESIS RATIONALE

### Why These 3 Final Approaches?

**Approach 1 Selection** (Service Layer):
Combined **Refactoring-Pro Approach 1** (Service Layer core) with **Tactical-Simplifier Approach 2** (formation preservation logic) and elements of **Tactical-Simplifier Approach 3** (result objects instead of direct mutation). This provides:
- Immediate testability (can test services in isolation)
- Familiar pattern (team can understand in minutes)
- Low risk (wrapping existing code)
- Practical value (fixes "can't test business logic" problem TODAY)

Chose this as **final Approach 1** because it's the pragmatic foundation all other approaches build on.

**Approach 2 Selection** (Command/Event):
Combined **Refactoring-Pro Approach 2** (unified game session access) with implied command/event pattern from multiple approaches. Kept best parts:
- Command pattern for actions (enables undo/replay)
- Event system for UI updates (decoupled)
- GameState owns all systems (not UI)
- Unidirectional data flow

Discarded **Refactoring-Pro Approach 3** (Repository/Command Executor) because CQS separation doesn't add value over this approach. Command pattern already separates reads (query methods) from writes (commands).

Chose this as **final Approach 2** because it's the natural evolution from Approach 1 when you need advanced features. Not for everyone, but architecturally sound when justified.

**Approach 3 Selection** (Turn State Machine):
Combined **Tactical-Simplifier implicit turn structure** from Approach 1 with explicit state machine pattern. Added turn phases, action queueing, validation/execution separation. Kept game-specific focus:
- Explicit turn phases (Planning, Executing, Resolving)
- Action queueing (plan before execute)
- Turn validation (enforce game rules)
- Side effect resolution (ability triggers)

Discarded **Tactical-Simplifier Approach 1** (tick-based) because turn-based game doesn't need frame-synchronized actions.

Chose this as **final Approach 3** because it's the best fit for tactics-heavy games with complex turn rules. Not appropriate for all games, but powerful when aligned with design.

### Rejected Elements
**What from initial 6 approaches was NOT included:**
1. **Facade Pattern** (Refactoring-Pro 2): Too likely to become God Object, doesn't improve testability
2. **Repository/Command Executor split** (Refactoring-Pro 3): CQS separation adds classes without clear value
3. **Tick-Based Processing** (Tactical-Simplifier 1): Adds latency without benefit for turn-based game
4. **Tactical State Manager** (Tactical-Simplifier 2): Good idea but too narrow, folded into Service Layer
5. **Query Caching**: Premature optimization, ECS queries are already fast enough

**Why rejected**: Each added complexity without proportional value. Following YAGNI principle - don't build features we don't need yet.

### Refactoring-Critic Key Insights

**Critical evaluation shaped final approaches by**:
1. **Prioritizing practical value over theoretical purity**: Service Layer isn't "perfect" architecture but provides immediate testability - that's what matters
2. **Highlighting over-engineering risks**: Command/Event pattern is excellent but only justified for specific use cases (undo, replay, networking)
3. **Emphasizing migration path**: Start simple (Services), evolve as needed (Commands/State Machine)
4. **Calling out YAGNI violations**: Don't build undo system "just in case," build it when you need it
5. **Balancing game-specific needs**: Turn State Machine makes sense for tactics game, not for all games

**Key insight**: **Incremental refactoring beats big-bang rewrites**. Service Layer provides 80% of value in 20% of time. Command/Event and State Machine are specialized tools for specific problems - use when problem exists, not preemptively.

---

## PRINCIPLES APPLIED

### Software Engineering Principles

**DRY (Don't Repeat Yourself)**:
- Service Layer extracts repeated combat logic (validation, execution, logging) from multiple UI handlers
- Command pattern eliminates duplicated validation logic across action handlers
- Result objects prevent repeated "get name, check destroyed, log message" code

**SOLID Principles**:
- **Single Responsibility**: Services own game logic, UI owns display. TurnManager owns turn flow, not UI.
- **Open/Closed**: Service methods are extension points (add new attack types), UI handlers closed for modification
- **Liskov Substitution**: Command interface allows any command to be executed/validated uniformly
- **Interface Segregation**: Narrow service APIs (CombatService vs SquadService) instead of one mega-facade
- **Dependency Inversion**: UI depends on service interfaces, not concrete ECS systems

**KISS (Keep It Simple)**:
- Service Layer is simplest solution that solves problem
- Command/Event only introduced when needed for specific features
- Avoided over-engineering (no unnecessary abstraction layers)

**YAGNI (You Aren't Gonna Need It)**:
- Don't build undo system until you need undo feature
- Don't add event bus until you need decoupled UI updates
- Don't implement replay until you need replay capability
- Start with Service Layer, evolve based on ACTUAL needs

**SLAP (Single Level of Abstraction Principle)**:
- UI handlers at "handle user action" abstraction
- Services at "execute game logic" abstraction
- ECS systems at "manipulate components" abstraction
- No mixing levels (UI doesn't manipulate components directly)

**Separation of Concerns**:
- UI: Handle input, display state, coordinate actions
- Services: Execute game logic, enforce rules, return results
- ECS: Store data, provide queries, manage entities

### Go-Specific Best Practices

**Idiomatic Go Patterns**:
- Struct composition over inheritance (Service composes TurnManager, FactionManager)
- Interfaces for behavior (GameCommand interface), structs for data (CombatInput)
- Error return values (service methods return error, not exceptions)
- Value receivers for immutable operations (ResolveCombat takes CombatInput by value)

**Composition Over Inheritance**:
- Services compose game systems instead of inheriting from base class
- TurnStateMachine composes TurnAction implementations
- No class hierarchies, just struct composition

**Interface Design**:
- Small, focused interfaces (GameCommand has 2 methods)
- Accept interfaces, return structs (Execute accepts GameState interface, returns concrete Result)
- Implicit interface satisfaction (no "implements" keyword)

**Error Handling**:
- Return errors, don't panic (ValidateAttack returns error, doesn't throw)
- Wrap errors with context (fmt.Errorf("cannot attack: %w", err))
- Check errors at boundaries (UI checks service errors, logs/displays to user)

### Game Development Considerations

**Performance Implications**:
- Service Layer: Minimal overhead (just method calls)
- Command/Event: O(n) event dispatch, typically <10 subscribers
- Turn State Machine: No overhead, just state tracking
- All approaches respect ECS performance (use queries, not iteration)

**Real-Time System Constraints**:
- 60 FPS target maintained (no long-running operations in Update/Render)
- Turn-based gameplay means no frame-critical logic
- UI updates happen after game logic (render current state)

**Game Loop Integration**:
- Service methods called from UI event handlers (button clicks)
- Commands executed synchronously (no async complexity)
- State machine phases align with turn flow, not frame updates

**Tactical Gameplay Preservation**:
- Formations preserved through movement (TacticalStateManager logic in services)
- Turn order maintained (TurnManager ownership moved but logic unchanged)
- Combat mechanics intact (ResolveCombat uses existing damage formulas)
- Squad builder workflow unchanged (just calls SquadService instead of direct ECS)

---

## NEXT STEPS

### Recommended Action Plan

**1. Immediate** (Start today - Day 1-2):
- Create game/ package (or combat_services/, squad_services/)
- Implement CombatService with ExecuteSquadAttack() method
- Write first unit test for combat logic (without GUI dependencies)
- **Success Metric**: One business logic test passing

**2. Short-term** (Next 3-5 days - Week 1):
- Complete CombatService, SquadService, MovementService
- Inject services into CombatMode, CombatActionHandler, SquadBuilderMode
- Refactor UI handlers to call services instead of direct systems
- Write 20+ unit tests covering core game mechanics
- **Success Metric**: 80% test coverage of business logic, combat mode working with services

**3. Medium-term** (Next 1-2 weeks):
- Evaluate: Do we need undo/replay/networking? If yes, add Command/Event pattern
- Evaluate: Do we want "plan then execute" gameplay? If yes, add Turn State Machine
- If neither needed, stop here (Service Layer is sufficient)
- If Command/Event needed: Implement command pattern, event bus, migrate UI handlers
- If State Machine needed: Implement turn phases, action queueing, update UI
- **Success Metric**: Advanced features working if implemented, or satisfaction with Service Layer

**4. Long-term** (Next month+):
- Use testable business logic to implement new game features faster
- Add complex combat mechanics (abilities, formations, status effects)
- Implement AI that uses services to evaluate options
- Consider multiplayer (if Command/Event pattern implemented)
- **Success Metric**: New features added in days instead of weeks, AI making tactical decisions

### Validation Strategy

**Testing Approach**:
1. **Unit Tests**: Test service methods in isolation (given input, assert output)
   - Example: `TestExecuteSquadAttack_DestroyedTarget`
   - Mock EntityManager if needed (or use test ECS world)

2. **Integration Tests**: Test UI handler + service integration
   - Example: `TestCombatActionHandler_ExecuteAttack_UpdatesUI`
   - Verify service called correctly, UI updated based on result

3. **End-to-End Tests**: Test full gameplay flow
   - Example: `TestCombatMode_FullTurn_AttackAndEndTurn`
   - Simulate player actions, verify game state changes correctly

**Rollback Plan**:
- Services are additive (wrap existing code), can remove without breaking game
- Keep original code in place during migration (comment out, don't delete)
- If service approach fails, revert to direct system calls
- Git branch for refactoring (`feature/service-layer`), merge only when working

**Success Metrics**:
- **Testability**: Can write unit test for combat logic without UI (0% → 80% coverage)
- **Maintainability**: Adding new combat action takes hours, not days
- **Complexity**: UI handler methods <50 lines, cyclomatic complexity <5
- **Bugs**: Fewer bugs from UI logic errors (separation prevents mistakes)
- **Velocity**: New features implemented faster (test-driven development)

### Additional Resources

**Go Patterns Documentation**:
- Effective Go: https://golang.org/doc/effective_go
- Go Code Review Comments: https://github.com/golang/go/wiki/CodeReviewComments
- Service Pattern in Go: https://www.gobeyond.dev/service-pattern/

**Game Architecture References**:
- Game Programming Patterns (Nystrom): Command pattern, State pattern
- ECS Best Practices: TinkerRogue docs/ecs_best_practices.md
- Turn-Based Game Architecture: https://www.gamedeveloper.com/programming/implementing-turn-based-combat

**Refactoring Resources**:
- Refactoring (Fowler): Extract Method, Extract Class patterns
- Working Effectively with Legacy Code (Feathers): Test harness techniques
- Clean Architecture (Martin): Separation of concerns, dependency rule

---

**END OF ANALYSIS**

---

## APPENDIX: IMPLEMENTATION CHECKLIST

Use this checklist when implementing Approach 1 (Service Layer):

### Phase 1: Create Services (Days 1-2)
- [ ] Create combat/combat_service.go file
- [ ] Define CombatService struct with entityManager, turnManager, factionManager fields
- [ ] Implement NewCombatService() constructor
- [ ] Implement ExecuteSquadAttack() with AttackResult return type
- [ ] Implement GetTurnManager(), GetFactionManager() query methods
- [ ] Create squads/squad_service.go file
- [ ] Define SquadService struct
- [ ] Implement CreateSquad(), GetSquadInfo(), DestroySquad() methods
- [ ] Write first unit test: TestCombatService_ExecuteSquadAttack_Success()
- [ ] Write second unit test: TestCombatService_ExecuteSquadAttack_InvalidTarget()

### Phase 2: Refactor CombatMode (Days 3-4)
- [ ] Update CombatMode.Initialize() to create CombatService
- [ ] Remove TurnManager, FactionManager, MovementSystem fields from CombatMode
- [ ] Add CombatService field to CombatMode
- [ ] Update CombatActionHandler constructor to accept CombatService
- [ ] Replace direct system calls with service.ExecuteSquadAttack() in executeAttack()
- [ ] Update UI logging to use AttackResult fields
- [ ] Test combat flow end-to-end (attack, destroy, end turn)
- [ ] Write integration test: TestCombatMode_AttackFlow()

### Phase 3: Refactor SquadBuilder (Day 5)
- [ ] Create SquadBuilderService (if not already in SquadService)
- [ ] Implement PlaceUnitInSquad(), RemoveUnitFromSquad() service methods
- [ ] Inject SquadService into SquadBuilderMode
- [ ] Update placeRosterUnitInCell() to call service.PlaceUnit()
- [ ] Update removeUnitFromCell() to call service.RemoveUnit()
- [ ] Update onCreateSquad() to call service.FinalizeSquad()
- [ ] Test squad builder flow (place units, create squad, verify ECS state)
- [ ] Write test: TestSquadBuilderService_CreateSquad_EnforcesCapacity()

### Phase 4: Add Tests (Days 6-7)
- [ ] Write TestCombatService_ExecuteSquadAttack_CounterAttack()
- [ ] Write TestCombatService_ExecuteSquadAttack_BothDestroyed()
- [ ] Write TestSquadService_CreateSquad_WithFormation()
- [ ] Write TestSquadService_AddUnit_ExceedsCapacity()
- [ ] Write TestMovementService_ValidateMove_OutOfRange()
- [ ] Measure test coverage (aim for >80% of service code)
- [ ] Add table-driven tests for combat damage calculations
- [ ] Document service APIs with godoc comments

### Validation Before Merging
- [ ] All existing gameplay works (combat, squad builder, deployment)
- [ ] No regressions (test every major feature manually)
- [ ] Unit tests cover core business logic (>80% coverage of services)
- [ ] Code review completed (check CLAUDE.md guidelines)
- [ ] Performance is acceptable (no noticeable lag vs before)
- [ ] Commit messages follow conventions
- [ ] Documentation updated (add services to CLAUDE.md)

---

**Congratulations!** If you've completed this checklist, you've successfully separated presentation from business logic, enabled testability, and set foundation for future refactoring.

**Next decision point**: After using Service Layer for 2-4 weeks, evaluate:
- Do we need undo/replay? → Implement Approach 2 (Command/Event)
- Do we want plan-then-execute gameplay? → Implement Approach 3 (Turn State Machine)
- Are services working well? → Keep current architecture, focus on features

---
