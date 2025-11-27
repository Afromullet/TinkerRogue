# GUI Separation Analysis: TinkerRogue UI Architecture
Generated: 2025-11-27
Target: GUI folder and packages (entire UI layer)

---

## EXECUTIVE SUMMARY

### Current State Assessment

**Strengths:**
- **Excellent service layer pattern exists**: `CombatService`, `SquadBuilderService` demonstrate perfect separation
- **Clean UI state management**: `BattleMapState` and `OverworldState` contain ONLY UI state (selections, modes)
- **Query service abstraction**: `GUIQueries` provides read-only ECS access for UI
- **Proper delegation in combat**: `CombatActionHandler` correctly delegates all game logic to `CombatService`

**Current Separation Status:**
- **Well Separated** (70%): Combat UI, Squad Builder UI
- **Needs Improvement** (20%): Squad Management, Formation Editor
- **Minor Issues** (10%): Exploration Mode, Info Mode

**Primary Issues Identified:**
1. **Direct ECS manipulation in UI modes**: Some modes directly query/modify components instead of using services
2. **Missing service layer for squad operations**: Squad management and formation editing lack dedicated services
3. **Inconsistent pattern application**: Not all UI modes follow the CombatMode service delegation pattern
4. **Mixed responsibilities in some UI factories**: Some UI factories contain business logic validation

### Recommended Direction

**Phase 1 (High Priority)**: Extract remaining game logic to services following `CombatService` pattern
**Phase 2 (Medium Priority)**: Standardize all UI modes to use service delegation
**Phase 3 (Low Priority)**: Enhance query service and eliminate remaining direct ECS access

### Quick Wins vs Strategic Refactoring

**Immediate Improvements (1-2 days):**
- Create `SquadManagementService` for squad operations
- Move formation validation logic from UI to service layer
- Standardize service result pattern across all services

**Medium-Term Goals (3-5 days):**
- Refactor FormationEditor to use service delegation
- Create deployment service for squad deployment logic
- Implement command pattern for undoable UI operations

**Long-Term Architecture (1-2 weeks):**
- Full service layer for all game systems
- Event-driven UI updates instead of polling
- Separation of rendering logic from UI modes

### Consensus Findings

**Agreement:**
- Service layer pattern (`CombatService`, `SquadBuilderService`) is the right approach
- UI state (`BattleMapState`, `OverworldState`) correctly contains only presentation state
- `GUIQueries` provides appropriate read-only abstraction
- Current architecture is sound; needs consistent application

**Critical Concerns:**
- **Avoid over-engineering**: Don't create services for trivial operations (simple queries, display logic)
- **Maintain game context**: This is a tactical roguelike, not a web app - keep patterns game-appropriate
- **Incremental refactoring**: Don't break working code; extract services gradually
- **Test coverage**: Ensure services have tests before relying on them

---

## FINAL SYNTHESIZED APPROACHES

### Approach 1: Service Layer Completion (Incremental Pattern Extension)

**Strategic Focus**: Complete the service layer by following existing `CombatService` and `SquadBuilderService` patterns for remaining UI operations

**Problem Statement**:
Currently, some UI modes (SquadManagementMode, FormationEditor, SquadDeployment) directly access ECS components for operations that involve game logic. This creates tight coupling and makes testing difficult. The codebase already has excellent service examples (`CombatService`, `SquadBuilderService`) but they haven't been applied consistently across all UI interactions.

**Solution Overview**:
Extract game logic from remaining UI modes into focused service classes that own systems and encapsulate business rules. Follow the established pattern where services:
- Own their systems (TurnManager, FactionManager, etc.)
- Return structured result objects
- Contain all validation logic
- Are injected into UI modes via constructor

**Code Examples**:

*Before (SquadManagementMode - Direct ECS access):*
```go
// gui/guisquads/squadmanagementmode.go
func (smm *SquadManagementMode) getSquadStats(squadID ecs.EntityID) string {
    // Direct ECS query - mixing UI and data access
    squadInfo := smm.Queries.GetSquadInfo(squadID)
    if squadInfo == nil {
        return "Squad not found"
    }
    return fmt.Sprintf("Units: %d\nTotal HP: %d/%d\nMorale: N/A",
        squadInfo.TotalUnits, squadInfo.CurrentHP, squadInfo.MaxHP)
}

// No service for disbanding squads, renaming, etc.
```

*After (SquadManagementMode - Service delegation):*
```go
// squads/squadservices/squad_management_service.go
type SquadManagementService struct {
    entityManager *common.EntityManager
}

type SquadStatsResult struct {
    SquadID      ecs.EntityID
    Name         string
    UnitCount    int
    CurrentHP    int
    MaxHP        int
    Morale       int
    CanDisband   bool
    DisbandError string
}

func (sms *SquadManagementService) GetSquadStats(squadID ecs.EntityID) *SquadStatsResult {
    // Business logic: calculate stats, check state
    squadData := squads.GetSquadData(squadID, sms.entityManager)
    if squadData == nil {
        return nil
    }

    stats := calculateSquadStats(squadID, sms.entityManager)
    canDisband, reason := sms.validateDisbandment(squadID)

    return &SquadStatsResult{
        SquadID:      squadID,
        Name:         squadData.Name,
        UnitCount:    stats.TotalUnits,
        CurrentHP:    stats.CurrentHP,
        MaxHP:        stats.MaxHP,
        Morale:       stats.Morale,
        CanDisband:   canDisband,
        DisbandError: reason,
    }
}

type DisbandSquadResult struct {
    Success        bool
    Error          string
    UnitsReturned  int
    RosterUpdated  bool
}

func (sms *SquadManagementService) DisbandSquad(
    playerID ecs.EntityID,
    squadID ecs.EntityID,
) *DisbandSquadResult {
    // All game logic encapsulated
    result := &DisbandSquadResult{}

    // Validate
    if !sms.canDisbandSquad(squadID) {
        result.Success = false
        result.Error = "Squad cannot be disbanded (in combat or deployed)"
        return result
    }

    // Get units to return to roster
    unitIDs := squads.GetUnitIDsInSquad(squadID, sms.entityManager)

    // Return units to roster
    roster := squads.GetPlayerRoster(playerID, sms.entityManager)
    for _, unitID := range unitIDs {
        roster.ReturnUnitToRoster(unitID)
    }

    // Remove squad entity
    squads.DestroySquad(squadID, sms.entityManager)

    result.Success = true
    result.UnitsReturned = len(unitIDs)
    result.RosterUpdated = true
    return result
}

// gui/guisquads/squadmanagementmode.go
type SquadManagementMode struct {
    gui.BaseMode
    squadService *squadservices.SquadManagementService // Injected
    // ... other fields
}

func (smm *SquadManagementMode) getSquadStats(squadID ecs.EntityID) string {
    // UI code - just formats service result
    stats := smm.squadService.GetSquadStats(squadID)
    if stats == nil {
        return "Squad not found"
    }
    return fmt.Sprintf("Units: %d\nTotal HP: %d/%d\nMorale: %d",
        stats.UnitCount, stats.CurrentHP, stats.MaxHP, stats.Morale)
}

func (smm *SquadManagementMode) onDisbandSquad() {
    result := smm.squadService.DisbandSquad(
        smm.Context.PlayerData.PlayerEntityID,
        smm.currentSquadID,
    )

    if result.Success {
        smm.logManager.AddLog(fmt.Sprintf("Squad disbanded. %d units returned to roster.",
            result.UnitsReturned))
        smm.refreshSquadList()
    } else {
        smm.logManager.AddLog(fmt.Sprintf("Error: %s", result.Error))
    }
}
```

**Key Changes**:
- Created `SquadManagementService` following `CombatService` pattern
- Extracted validation logic (can disband, in combat check, etc.) to service
- Service returns structured result objects with all needed UI data
- UI mode only handles presentation and delegates business logic
- Service owns squad lifecycle operations (disband, rename, merge, etc.)

**Value Proposition**:

- **Maintainability**: Game logic centralized in testable services
- **Readability**: UI modes become thin presentation layers
- **Extensibility**: New squad operations added to service, not scattered in UI
- **Complexity Impact**:
  - Lines of code: ~+200 in services, ~-50 in UI modes (net +150)
  - Cyclomatic complexity: Reduced in UI modes, isolated in services
  - Testability: High (services can be unit tested without UI)

**Implementation Strategy**:

1. **Create SquadManagementService** (~2 hours)
   - File: `squads/squadservices/squad_management_service.go`
   - Methods: `GetSquadStats`, `DisbandSquad`, `RenameSquad`, `MergeSquads`
   - Follow `CombatService` result pattern
   - Add tests in `squad_management_service_test.go`

2. **Create FormationService** (~2 hours)
   - File: `squads/squadservices/formation_service.go`
   - Methods: `ValidateFormation`, `ApplyFormation`, `SaveFormationPreset`
   - Extract formation validation from UI
   - Add position conflict detection logic

3. **Create SquadDeploymentService** (~1.5 hours)
   - File: `squads/squadservices/squad_deployment_service.go`
   - Methods: `ValidateDeployment`, `DeploySquad`, `RecallSquad`
   - Handle deployment tile validation
   - Manage deployment state transitions

4. **Refactor UI modes to use services** (~2 hours)
   - Update SquadManagementMode to inject SquadManagementService
   - Update FormationEditor to inject FormationService
   - Update SquadDeploymentMode to inject SquadDeploymentService
   - Replace direct ECS access with service calls

5. **Add service tests** (~1 hour)
   - Test validation logic in isolation
   - Test error cases
   - Verify state changes

**Advantages**:
- **Follows existing patterns**: Developers already understand CombatService approach
- **Low risk**: Incremental extraction, can be done file-by-file
- **High testability**: Services are pure logic, easily unit tested
- **Clear boundaries**: Service = game logic, UI = presentation
- **Immediate value**: Each service extraction improves code quality

**Drawbacks & Risks**:
- **Service proliferation**: Could end up with too many small services (mitigation: combine related operations)
- **Result object verbosity**: Many result structs needed (mitigation: use common result patterns, generate where possible)
- **Over-abstraction for simple queries**: Don't create services for read-only display queries (mitigation: keep `GUIQueries` for read-only, services for mutations)
- **Integration testing still needed**: Services need ECS setup to test (mitigation: use existing test utilities)

**Effort Estimate**:
- **Time**: 8-10 hours (1-2 days)
- **Complexity**: Medium (pattern established, just need to apply consistently)
- **Risk**: Low (incremental, preserves existing functionality)
- **Files Impacted**: 9 files
  - New: 3 service files + 3 test files
  - Modified: SquadManagementMode, FormationEditor, SquadDeploymentMode

**Critical Assessment**:
This is the most practical approach because it extends an already-proven pattern in the codebase. `CombatService` and `SquadBuilderService` demonstrate that this architecture works well for this game. The key is avoiding over-abstraction - only create services for operations that involve business logic or state mutations, not for simple read-only queries (that's what `GUIQueries` is for).

---

### Approach 2: Command Pattern for UI Operations (Undoable Actions)

**Strategic Focus**: Implement command pattern for squad operations to enable undo/redo, better separation of concerns, and centralized validation

**Problem Statement**:
Current UI operations execute immediately without ability to undo, validate in isolation, or queue operations. This makes it difficult to implement features like "confirm before disbanding" or "undo last formation change". Additionally, validation logic is scattered between UI event handlers and service methods.

**Solution Overview**:
Introduce lightweight command objects that encapsulate squad operations. Commands contain validation logic, execution logic, and undo logic. UI modes create and execute commands through a CommandExecutor that handles validation, execution, and history management.

**Code Examples**:

*Before (Direct service calls):*
```go
// gui/guisquads/squadmanagementmode.go
func (smm *SquadManagementMode) onDisbandSquad() {
    // No confirmation, no undo, immediate execution
    if smm.currentSquadID == 0 {
        return
    }

    // Direct service call
    unitIDs := squads.GetUnitIDsInSquad(smm.currentSquadID, smm.Context.ECSManager)
    roster := squads.GetPlayerRoster(smm.Context.PlayerData.PlayerEntityID, smm.Context.ECSManager)
    for _, unitID := range unitIDs {
        roster.ReturnUnitToRoster(unitID)
    }
    squads.DestroySquad(smm.currentSquadID, smm.Context.ECSManager)

    smm.currentSquadID = 0
    smm.refreshSquadList()
}
```

*After (Command pattern):*
```go
// squads/squadservices/commands.go
type SquadCommand interface {
    Validate() error
    Execute() error
    Undo() error
    Description() string
}

type DisbandSquadCommand struct {
    entityManager  *common.EntityManager
    playerID       ecs.EntityID
    squadID        ecs.EntityID

    // Captured state for undo
    savedSquadData *squads.SquadData
    savedUnitIDs   []ecs.EntityID
    savedPositions map[ecs.EntityID]coords.LogicalPosition
}

func NewDisbandSquadCommand(
    manager *common.EntityManager,
    playerID ecs.EntityID,
    squadID ecs.EntityID,
) *DisbandSquadCommand {
    return &DisbandSquadCommand{
        entityManager: manager,
        playerID:      playerID,
        squadID:       squadID,
    }
}

func (cmd *DisbandSquadCommand) Validate() error {
    // All validation logic centralized
    if cmd.squadID == 0 {
        return fmt.Errorf("invalid squad ID")
    }

    // Check if squad exists
    squadData := squads.GetSquadData(cmd.squadID, cmd.entityManager)
    if squadData == nil {
        return fmt.Errorf("squad does not exist")
    }

    // Check if squad is in combat
    if combat.IsSquadInCombat(cmd.squadID, cmd.entityManager) {
        return fmt.Errorf("cannot disband squad in combat")
    }

    // Check if squad is deployed
    if combat.IsSquadDeployed(cmd.squadID, cmd.entityManager) {
        return fmt.Errorf("recall squad before disbanding")
    }

    return nil
}

func (cmd *DisbandSquadCommand) Execute() error {
    // Capture state for undo
    cmd.savedSquadData = squads.GetSquadData(cmd.squadID, cmd.entityManager)
    cmd.savedUnitIDs = squads.GetUnitIDsInSquad(cmd.squadID, cmd.entityManager)
    cmd.savedPositions = make(map[ecs.EntityID]coords.LogicalPosition)

    for _, unitID := range cmd.savedUnitIDs {
        pos := common.GetPosition(cmd.entityManager, unitID)
        if pos != nil {
            cmd.savedPositions[unitID] = *pos
        }
    }

    // Execute operation
    roster := squads.GetPlayerRoster(cmd.playerID, cmd.entityManager)
    for _, unitID := range cmd.savedUnitIDs {
        if err := roster.ReturnUnitToRoster(unitID); err != nil {
            return fmt.Errorf("failed to return unit to roster: %w", err)
        }
    }

    if err := squads.DestroySquad(cmd.squadID, cmd.entityManager); err != nil {
        return fmt.Errorf("failed to destroy squad: %w", err)
    }

    return nil
}

func (cmd *DisbandSquadCommand) Undo() error {
    // Restore squad from saved state
    newSquadID := squads.CreateSquadFromData(cmd.savedSquadData, cmd.entityManager)

    // Re-add units to squad
    roster := squads.GetPlayerRoster(cmd.playerID, cmd.entityManager)
    for _, unitID := range cmd.savedUnitIDs {
        roster.RemoveUnitFromRoster(unitID)
        squads.AddUnitToSquad(newSquadID, unitID, cmd.entityManager)

        // Restore position
        if pos, ok := cmd.savedPositions[unitID]; ok {
            common.SetPosition(cmd.entityManager, unitID, pos)
        }
    }

    return nil
}

func (cmd *DisbandSquadCommand) Description() string {
    return fmt.Sprintf("Disband squad %s", cmd.savedSquadData.Name)
}

// squads/squadservices/command_executor.go
type CommandExecutor struct {
    history    []SquadCommand
    redoStack  []SquadCommand
    maxHistory int
}

func NewCommandExecutor() *CommandExecutor {
    return &CommandExecutor{
        history:    make([]SquadCommand, 0),
        redoStack:  make([]SquadCommand, 0),
        maxHistory: 20,
    }
}

type CommandResult struct {
    Success     bool
    Error       string
    Description string
}

func (ce *CommandExecutor) Execute(cmd SquadCommand) *CommandResult {
    result := &CommandResult{
        Description: cmd.Description(),
    }

    // Validate
    if err := cmd.Validate(); err != nil {
        result.Success = false
        result.Error = fmt.Sprintf("Validation failed: %v", err)
        return result
    }

    // Execute
    if err := cmd.Execute(); err != nil {
        result.Success = false
        result.Error = fmt.Sprintf("Execution failed: %v", err)
        return result
    }

    // Add to history
    ce.history = append(ce.history, cmd)
    if len(ce.history) > ce.maxHistory {
        ce.history = ce.history[1:]
    }

    // Clear redo stack (new action invalidates redo)
    ce.redoStack = ce.redoStack[:0]

    result.Success = true
    return result
}

func (ce *CommandExecutor) Undo() *CommandResult {
    if len(ce.history) == 0 {
        return &CommandResult{
            Success: false,
            Error:   "Nothing to undo",
        }
    }

    // Pop last command
    lastCmd := ce.history[len(ce.history)-1]
    ce.history = ce.history[:len(ce.history)-1]

    // Execute undo
    if err := lastCmd.Undo(); err != nil {
        return &CommandResult{
            Success: false,
            Error:   fmt.Sprintf("Undo failed: %v", err),
        }
    }

    // Add to redo stack
    ce.redoStack = append(ce.redoStack, lastCmd)

    return &CommandResult{
        Success:     true,
        Description: fmt.Sprintf("Undid: %s", lastCmd.Description()),
    }
}

// gui/guisquads/squadmanagementmode.go
type SquadManagementMode struct {
    gui.BaseMode
    commandExecutor *squadservices.CommandExecutor
    // ... other fields
}

func (smm *SquadManagementMode) onDisbandSquad() {
    if smm.currentSquadID == 0 {
        return
    }

    // Create command
    cmd := squadservices.NewDisbandSquadCommand(
        smm.Context.ECSManager,
        smm.Context.PlayerData.PlayerEntityID,
        smm.currentSquadID,
    )

    // Execute via executor
    result := smm.commandExecutor.Execute(cmd)

    if result.Success {
        smm.addLog(fmt.Sprintf("✓ %s", result.Description))
        smm.currentSquadID = 0
        smm.refreshSquadList()
    } else {
        smm.addLog(fmt.Sprintf("✗ %s", result.Error))
    }
}

func (smm *SquadManagementMode) onUndo() {
    result := smm.commandExecutor.Undo()
    if result.Success {
        smm.addLog(fmt.Sprintf("⟲ %s", result.Description))
        smm.refreshSquadList()
    } else {
        smm.addLog(result.Error)
    }
}
```

**Key Changes**:
- Introduced `SquadCommand` interface with Validate/Execute/Undo methods
- Created command objects for each operation (Disband, Rename, Merge, etc.)
- Commands capture state for undo
- `CommandExecutor` manages command history and undo/redo stacks
- UI modes create and execute commands instead of calling services directly

**Value Proposition**:

- **Maintainability**: Validation and execution logic centralized in commands
- **Readability**: Each operation is a self-contained command class
- **Extensibility**: New operations = new command classes, no UI changes
- **Complexity Impact**:
  - Lines of code: ~+400 (command infrastructure + command classes)
  - Cyclomatic complexity: Commands are simple; executor handles complexity
  - User experience: Enables undo/redo, better error messages

**Implementation Strategy**:

1. **Create command infrastructure** (~3 hours)
   - Define `SquadCommand` interface
   - Implement `CommandExecutor` with history management
   - Add tests for executor

2. **Implement core commands** (~4 hours)
   - `DisbandSquadCommand`
   - `RenameSquadCommand`
   - `MergeSquadsCommand`
   - `ChangeFormationCommand`
   - Each with validation, execute, undo logic

3. **Integrate with UI** (~2 hours)
   - Inject `CommandExecutor` into UI modes
   - Replace direct operations with command execution
   - Add undo/redo buttons and hotkeys

4. **Add command tests** (~2 hours)
   - Test validation logic per command
   - Test execute/undo cycles
   - Test error cases

**Advantages**:
- **Undo/Redo support**: Major UX improvement for free
- **Centralized validation**: All validation in command's Validate() method
- **Atomic operations**: Commands are all-or-nothing transactions
- **Audit trail**: Command history provides operation log
- **Testability**: Commands are pure logic, easily tested

**Drawbacks & Risks**:
- **More boilerplate**: Each operation needs a command class (mitigation: use code generation for simple commands)
- **Memory overhead**: Storing undo state (mitigation: limit history size, only save essential state)
- **Undo complexity**: Some operations hard to undo (mitigation: mark certain commands as non-undoable)
- **Over-engineering for simple operations**: Not all operations need undo (mitigation: mix command pattern with simple service calls)
- **State consistency**: Undo might fail if game state changed (mitigation: validation in Undo() method)

**Effort Estimate**:
- **Time**: 11-13 hours (1.5-2 days)
- **Complexity**: High (new pattern for codebase, requires careful state management)
- **Risk**: Medium (undo logic can be tricky, needs thorough testing)
- **Files Impacted**: 12 files
  - New: commands.go, command_executor.go, 5 command implementations, 3 test files
  - Modified: SquadManagementMode, FormationEditor, SquadBuilder

**Critical Assessment**:
This approach provides significant UX value (undo/redo) but requires more upfront investment. It's a good fit for squad management operations where users might want to experiment and revert changes. However, it may be over-engineering for simpler operations like viewing squad stats. Best used selectively for high-value operations rather than applied universally. Consider combining with Approach 1 - use services for simple operations, commands for complex/undoable operations.

---

### Approach 3: Event-Driven UI Updates (Observer Pattern)

**Strategic Focus**: Replace polling-based UI updates with event-driven notifications to reduce coupling and improve performance

**Problem Statement**:
Currently, UI modes poll for state changes every frame in their `Update()` methods. This creates unnecessary coupling (UI needs to know what to check), wastes CPU cycles (checking unchanged state), and makes it hard to know when data actually changed. For example, `CombatMode.Update()` calls component refresh methods every frame even when nothing changed.

**Solution Overview**:
Introduce an event system where game logic publishes events when state changes, and UI components subscribe to relevant events. This inverts the dependency - game logic doesn't know about UI, and UI only updates when notified of actual changes.

**Code Examples**:

*Before (Polling every frame):*
```go
// gui/guicombat/combatmode.go
func (cm *CombatMode) Update(deltaTime float64) error {
    // Called every frame - wasteful if nothing changed
    cm.turnOrderComponent.Refresh()

    currentFactionID := cm.combatService.GetCurrentFaction()
    if currentFactionID != 0 {
        cm.factionInfoComponent.ShowFaction(currentFactionID)
    }

    selectedSquad := cm.Context.ModeCoordinator.GetBattleMapState().SelectedSquadID
    if selectedSquad != 0 {
        cm.squadDetailComponent.ShowSquad(selectedSquad)
    }

    return nil
}
```

*After (Event-driven updates):*
```go
// common/events.go
type EventType string

const (
    EventTurnChanged        EventType = "turn_changed"
    EventSquadSelected      EventType = "squad_selected"
    EventSquadHealthChanged EventType = "squad_health_changed"
    EventSquadDestroyed     EventType = "squad_destroyed"
    EventFactionEliminated  EventType = "faction_eliminated"
)

type Event struct {
    Type      EventType
    Data      interface{}
    Timestamp time.Time
}

type TurnChangedEvent struct {
    PreviousFaction ecs.EntityID
    NewFaction      ecs.EntityID
    NewRound        int
}

type SquadSelectedEvent struct {
    SquadID ecs.EntityID
}

type SquadHealthChangedEvent struct {
    SquadID    ecs.EntityID
    OldHP      int
    NewHP      int
    Percentage float64
}

// common/event_bus.go
type EventHandler func(event Event)

type EventBus struct {
    handlers map[EventType][]EventHandler
    mu       sync.RWMutex
}

func NewEventBus() *EventBus {
    return &EventBus{
        handlers: make(map[EventType][]EventHandler),
    }
}

func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
    eb.mu.Lock()
    defer eb.mu.Unlock()

    eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *EventBus) Publish(event Event) {
    eb.mu.RLock()
    handlers := eb.handlers[event.Type]
    eb.mu.RUnlock()

    for _, handler := range handlers {
        handler(event)
    }
}

// combat/combatservices/combat_service.go
type CombatService struct {
    entityManager  *common.EntityManager
    turnManager    *combat.TurnManager
    factionManager *combat.FactionManager
    movementSystem *combat.CombatMovementSystem
    eventBus       *common.EventBus // Added
}

func (cs *CombatService) EndTurn() *EndTurnResult {
    result := &EndTurnResult{
        PreviousFaction: cs.turnManager.GetCurrentFaction(),
    }

    err := cs.turnManager.EndTurn()
    if err != nil {
        result.Success = false
        result.Error = err.Error()
        return result
    }

    result.Success = true
    result.NewFaction = cs.turnManager.GetCurrentFaction()
    result.NewRound = cs.turnManager.GetCurrentRound()

    // Publish event - game logic doesn't know about UI
    cs.eventBus.Publish(common.Event{
        Type: common.EventTurnChanged,
        Data: &common.TurnChangedEvent{
            PreviousFaction: result.PreviousFaction,
            NewFaction:      result.NewFaction,
            NewRound:        result.NewRound,
        },
        Timestamp: time.Now(),
    })

    return result
}

func (cs *CombatService) ExecuteSquadAttack(attackerID, targetID ecs.EntityID) *AttackResult {
    // ... existing attack logic ...

    result.Success = true
    result.TargetDestroyed = squads.IsSquadDestroyed(targetID, cs.entityManager)

    // Publish events
    if result.TargetDestroyed {
        cs.eventBus.Publish(common.Event{
            Type: common.EventSquadDestroyed,
            Data: &common.SquadDestroyedEvent{
                SquadID: targetID,
            },
            Timestamp: time.Now(),
        })
    } else {
        // Get HP for health changed event
        squadInfo := // ... get squad HP ...
        cs.eventBus.Publish(common.Event{
            Type: common.EventSquadHealthChanged,
            Data: &common.SquadHealthChangedEvent{
                SquadID:    targetID,
                OldHP:      // ... old HP ...
                NewHP:      squadInfo.CurrentHP,
                Percentage: float64(squadInfo.CurrentHP) / float64(squadInfo.MaxHP),
            },
            Timestamp: time.Now(),
        })
    }

    return result
}

// gui/guicombat/combatmode.go
type CombatMode struct {
    gui.BaseMode

    combatService *combatservices.CombatService
    eventBus      *common.EventBus

    // ... other fields ...
}

func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    cm.InitializeBase(ctx)

    // Create event bus (or inject shared one)
    cm.eventBus = common.NewEventBus()

    // Create combat service with event bus
    cm.combatService = combatservices.NewCombatService(ctx.ECSManager, cm.eventBus)

    // Subscribe to events
    cm.subscribeToEvents()

    // ... rest of initialization ...

    return nil
}

func (cm *CombatMode) subscribeToEvents() {
    // Turn changed - update turn display and faction info
    cm.eventBus.Subscribe(common.EventTurnChanged, func(event common.Event) {
        turnEvent := event.Data.(*common.TurnChangedEvent)

        // Update turn order display
        cm.turnOrderComponent.Refresh()

        // Update faction info
        cm.factionInfoComponent.ShowFaction(turnEvent.NewFaction)

        // Log turn change
        factionName := cm.combatService.GetFactionName(turnEvent.NewFaction)
        cm.logManager.UpdateTextArea(cm.combatLogArea,
            fmt.Sprintf("=== Round %d: %s's Turn ===", turnEvent.NewRound, factionName))

        // Clear selection
        cm.Context.ModeCoordinator.GetBattleMapState().Reset()
        cm.squadListComponent.Refresh()
    })

    // Squad selected - update detail panel
    cm.eventBus.Subscribe(common.EventSquadSelected, func(event common.Event) {
        selectEvent := event.Data.(*common.SquadSelectedEvent)
        cm.squadDetailComponent.ShowSquad(selectEvent.SquadID)
    })

    // Squad health changed - update if currently selected
    cm.eventBus.Subscribe(common.EventSquadHealthChanged, func(event common.Event) {
        healthEvent := event.Data.(*common.SquadHealthChangedEvent)

        // Only refresh if this squad is currently selected
        battleState := cm.Context.ModeCoordinator.GetBattleMapState()
        if battleState.SelectedSquadID == healthEvent.SquadID {
            cm.squadDetailComponent.ShowSquad(healthEvent.SquadID)
        }

        // Update squad list (shows HP bars)
        cm.squadListComponent.RefreshSquad(healthEvent.SquadID)
    })

    // Squad destroyed - remove from UI
    cm.eventBus.Subscribe(common.EventSquadDestroyed, func(event common.Event) {
        destroyedEvent := event.Data.(*common.SquadDestroyedEvent)

        // Log destruction
        squadName := squads.GetSquadName(destroyedEvent.SquadID, cm.Queries.ECSManager)
        cm.logManager.UpdateTextArea(cm.combatLogArea,
            fmt.Sprintf("%s was destroyed!", squadName))

        // Clear selection if destroyed squad was selected
        battleState := cm.Context.ModeCoordinator.GetBattleMapState()
        if battleState.SelectedSquadID == destroyedEvent.SquadID {
            battleState.SelectedSquadID = 0
            cm.squadDetailComponent.SetText("Select a squad")
        }

        // Refresh squad list
        cm.squadListComponent.Refresh()

        // Check victory condition
        victoryResult := cm.combatService.CheckVictoryCondition()
        if victoryResult.BattleOver {
            // ... handle battle end ...
        }
    })
}

func (cm *CombatMode) Update(deltaTime float64) error {
    // No polling needed - events drive updates
    // Only update animations or time-based effects here
    return nil
}

// gui/guicombat/combat_action_handler.go
func (cah *CombatActionHandler) SelectSquad(squadID ecs.EntityID) {
    cah.battleMapState.SelectedSquadID = squadID

    // Publish event instead of direct UI update
    cah.eventBus.Publish(common.Event{
        Type: common.EventSquadSelected,
        Data: &common.SquadSelectedEvent{
            SquadID: squadID,
        },
        Timestamp: time.Now(),
    })

    // Get squad name and log
    squadName := squads.GetSquadName(squadID, cah.queries.ECSManager)
    cah.addLog(fmt.Sprintf("Selected: %s", squadName))
}
```

**Key Changes**:
- Created `EventBus` for publish/subscribe pattern
- Services publish events when state changes
- UI modes subscribe to relevant events in `Initialize()`
- `Update()` no longer polls - only handles time-based updates
- UI components update reactively when events fire

**Value Proposition**:

- **Maintainability**: Clear separation - services don't know about UI
- **Readability**: Event subscriptions document what triggers UI updates
- **Extensibility**: New UI components can subscribe without changing services
- **Complexity Impact**:
  - Performance: Updates only when state changes (not every frame)
  - CPU usage: Reduced from polling every frame to event-driven
  - Responsiveness: Immediate UI updates when events fire

**Implementation Strategy**:

1. **Create event infrastructure** (~2 hours)
   - Define `Event`, `EventType`, `EventBus`
   - Implement subscribe/publish mechanisms
   - Add event data structures for game events
   - Write tests for event bus

2. **Add events to services** (~3 hours)
   - Modify `CombatService` to publish events
   - Modify `SquadBuilderService` to publish events
   - Define event types for all state changes
   - Document which events each service publishes

3. **Refactor UI to event-driven** (~4 hours)
   - Remove polling logic from `Update()` methods
   - Add event subscriptions in `Initialize()`
   - Update UI components in event handlers
   - Test event-driven UI updates

4. **Add selective updates** (~1 hour)
   - Only refresh components affected by events
   - Add dirty flags for batch updates
   - Optimize event handler performance

**Advantages**:
- **Performance**: No wasted cycles polling unchanged state
- **Decoupling**: Services don't reference UI, only publish events
- **Flexibility**: Multiple UI components can react to same event
- **Debugging**: Event log shows exactly what changed and when
- **Testability**: Can test services and UI independently

**Drawbacks & Risks**:
- **Event proliferation**: May end up with many event types (mitigation: use hierarchical events, wildcards)
- **Debugging difficulty**: Event chains can be hard to trace (mitigation: add event logging, dev tools)
- **Memory leaks**: Forgetting to unsubscribe (mitigation: auto-cleanup when modes exit)
- **Threading issues**: Event handlers need thread safety (mitigation: ensure events fire on main thread only)
- **Over-engineering for simple cases**: Not all updates need events (mitigation: use for cross-component updates only)
- **Event ordering**: Multiple events may fire in unexpected order (mitigation: batch related events)

**Effort Estimate**:
- **Time**: 10-12 hours (1.5-2 days)
- **Complexity**: High (architectural change, affects many files)
- **Risk**: Medium (event-driven can introduce subtle bugs, needs thorough testing)
- **Files Impacted**: 15+ files
  - New: events.go, event_bus.go, event_bus_test.go
  - Modified: CombatService, SquadBuilderService, all UI modes, action handlers

**Critical Assessment**:
Event-driven architecture is powerful for complex UI with many interconnected components. However, for this tactical roguelike, it may be over-engineering. The current frame-by-frame update is simple and works well for a game loop. Event systems add complexity and potential for bugs (race conditions, forgotten subscriptions, event storms). This approach would be more valuable if the game had more complex UI interactions or real-time multiplayer. For a single-player tactical game, the simpler polling approach combined with Approach 1's service layer is likely sufficient. Consider this approach only if performance profiling shows update loops are a bottleneck, or if implementing features like spectator mode or AI debugging views.

---

## COMPARATIVE ANALYSIS OF FINAL APPROACHES

### Effort vs Impact Matrix
| Approach | Effort | Impact | Risk | Recommended Priority |
|----------|--------|--------|------|---------------------|
| Approach 1: Service Layer Completion | Medium | High | Low | **1** (Do First) |
| Approach 2: Command Pattern | High | Medium | Medium | **3** (Consider Later) |
| Approach 3: Event-Driven Updates | High | Low | Medium | **2** (Only If Needed) |

### Decision Guidance

**Choose Approach 1 (Service Layer Completion) if:**
- You want to follow established patterns already in the codebase
- You need quick wins with low risk
- You want better testability and separation of concerns
- You're planning to add more squad operations in the future
- **Recommended for: Immediate implementation**

**Choose Approach 2 (Command Pattern) if:**
- You want to add undo/redo functionality to squad management
- You need better validation and error handling
- You want to add confirmation dialogs for destructive operations
- You're building a complex editor (formation designer, scenario editor)
- **Recommended for: Later phase if UX demands it**

**Choose Approach 3 (Event-Driven Updates) if:**
- Performance profiling shows update loops are a bottleneck
- You're adding real-time multiplayer or spectator features
- You have many UI components reacting to same state changes
- You need audit trails or replay functionality
- **Recommended for: Only if clear performance need identified**

### Combination Opportunities

**Best Combination: Approach 1 + Selective Approach 2**
1. Start with Approach 1 (Service Layer Completion) for all UI operations
2. Add Approach 2 (Command Pattern) only for squad management operations where undo is valuable:
   - Disband squad (restore if accident)
   - Merge squads (undo if wrong squads selected)
   - Change formation (revert to previous formation)
3. Keep simple operations (view stats, list squads) as direct service calls
4. Skip Approach 3 unless profiling shows need

**Implementation Order:**
1. **Week 1**: Implement Approach 1 completely
   - Create SquadManagementService, FormationService, SquadDeploymentService
   - Refactor UI modes to use services
   - Add service tests
   - Verify existing functionality works

2. **Week 2** (if needed): Add selective Command Pattern
   - Implement CommandExecutor infrastructure
   - Create DisbandSquadCommand, MergeSquadsCommand
   - Add undo/redo buttons to SquadManagementMode
   - Mix commands with simple service calls

3. **Future** (only if needed): Consider event-driven updates
   - Profile current performance
   - Identify actual bottlenecks
   - Implement events only for proven performance issues

---

## APPENDIX: CURRENT ARCHITECTURE ASSESSMENT

### A. Well-Separated Examples (Follow These Patterns)

#### Example 1: CombatService (Excellent Separation)

**File**: `combat/combatservices/combat_service.go`

**Why It's Good:**
```go
// ✅ Service owns systems
type CombatService struct {
    entityManager  *common.EntityManager
    turnManager    *combat.TurnManager      // Owned by service
    factionManager *combat.FactionManager   // Owned by service
    movementSystem *combat.CombatMovementSystem // Owned by service
}

// ✅ Returns structured results (not raw ECS data)
type AttackResult struct {
    Success         bool
    ErrorReason     string
    AttackerName    string
    TargetName      string
    TargetDestroyed bool
    DamageDealt     int
}

// ✅ Contains all validation and game logic
func (cs *CombatService) ExecuteSquadAttack(attackerID, targetID ecs.EntityID) *AttackResult {
    // Validation
    reason, canAttack := combatSys.CanSquadAttackWithReason(attackerID, targetID)
    if !canAttack {
        result.Success = false
        result.ErrorReason = reason
        return result
    }

    // Execution
    err := combatSys.ExecuteAttackAction(attackerID, targetID)
    // ... handle result
}
```

**UI Usage** (Clean delegation):
```go
// gui/guicombat/combat_action_handler.go
func (cah *CombatActionHandler) ExecuteAttack() {
    // ✅ UI just delegates and formats result
    result := cah.combatService.ExecuteSquadAttack(selectedSquad, selectedTarget)

    if !result.Success {
        cah.addLog(fmt.Sprintf("Cannot attack: %s", result.ErrorReason))
    } else {
        cah.addLog(fmt.Sprintf("%s attacked %s!", result.AttackerName, result.TargetName))
    }

    // ✅ UI state update only
    cah.battleMapState.InAttackMode = false
}
```

**Lessons:**
- Services own their systems (TurnManager, FactionManager)
- Services return domain-specific result objects
- UI never touches ECS directly for mutations
- Clear separation: Service = game logic, UI = presentation

---

#### Example 2: SquadBuilderService (Excellent Separation)

**File**: `squads/squadservices/squad_builder_service.go`

**Why It's Good:**
```go
// ✅ Service handles complex multi-step operations atomically
func (sbs *SquadBuilderService) AssignRosterUnitToSquad(
    playerID ecs.EntityID,
    squadID ecs.EntityID,
    unitEntityID ecs.EntityID,
    unitTemplate squads.UnitTemplate,
    row, col int,
) *AssignUnitResult {
    // ✅ All validation logic in service
    roster := squads.GetPlayerRoster(playerID, sbs.entityManager)
    if roster == nil {
        return &AssignUnitResult{Success: false, Error: "Player has no roster"}
    }

    // ✅ Business logic: Mark unit as assigned in roster
    if err := roster.MarkUnitAsAssigned(unitEntityID); err != nil {
        return &AssignUnitResult{Success: false, Error: err.Error()}
    }

    // ✅ Business logic: Create squad member entity
    placedUnitID := squads.CreateSquadMember(squadID, unitTemplate, row, col, sbs.entityManager)

    // ✅ Atomic operation - both roster AND squad updated together
    return &AssignUnitResult{
        Success:       true,
        PlacedUnitID:  placedUnitID,
        RosterUnitID:  unitEntityID,
    }
}
```

**UI Usage** (Clean delegation):
```go
// gui/guisquads/squadbuilder.go
func (sbm *SquadBuilderMode) placeRosterUnitInCell(row, col int, rosterEntry *squads.UnitRosterEntry) {
    // ✅ UI collects parameters, delegates to service
    result := sbm.squadBuilderSvc.AssignRosterUnitToSquad(
        sbm.Context.PlayerData.PlayerEntityID,
        sbm.currentSquadID,
        unitEntityID,
        *unitTemplate,
        row, col,
    )

    // ✅ UI only handles success/failure presentation
    if !result.Success {
        fmt.Printf("Failed to place unit: %s\n", result.Error)
        return
    }

    // ✅ UI updates display based on result
    sbm.gridManager.UpdateDisplayForPlacedUnit(result.PlacedUnitID, unitTemplate, row, col, result.RosterUnitID)
}
```

**Lessons:**
- Services handle atomic operations (roster + squad updates together)
- Services contain validation and business rules
- UI provides parameters and handles presentation
- Clear result objects communicate success/failure with context

---

#### Example 3: GUIQueries (Excellent Read-Only Abstraction)

**File**: `gui/guicomponents/guiqueries.go`

**Why It's Good:**
```go
// ✅ Read-only query service - no mutations
type GUIQueries struct {
    ECSManager     *common.EntityManager
    factionManager *combat.FactionManager
}

// ✅ Returns UI-friendly data structures (not raw ECS entities)
type SquadInfo struct {
    ID                ecs.EntityID
    Name              string
    UnitIDs           []ecs.EntityID
    AliveUnits        int
    TotalUnits        int
    CurrentHP         int
    MaxHP             int
    Position          *coords.LogicalPosition
    FactionID         ecs.EntityID
    IsDestroyed       bool
    HasActed          bool
    HasMoved          bool
    MovementRemaining int
}

// ✅ Encapsulates complex queries
func (gq *GUIQueries) GetSquadInfo(squadID ecs.EntityID) *SquadInfo {
    // Multiple ECS queries combined into one UI-friendly result
    name := squads.GetSquadName(squadID, gq.ECSManager)
    unitIDs := squads.GetUnitIDsInSquad(squadID, gq.ECSManager)

    // Calculate aggregates
    aliveUnits, totalHP, maxHP := 0, 0, 0
    for _, unitID := range unitIDs {
        attrs := common.GetAttributesByIDWithTag(gq.ECSManager, unitID, squads.SquadMemberTag)
        if attrs != nil {
            if attrs.CanAct { aliveUnits++ }
            totalHP += attrs.CurrentHealth
            maxHP += attrs.MaxHealth
        }
    }

    // Return complete UI-friendly data
    return &SquadInfo{
        ID:         squadID,
        Name:       name,
        UnitIDs:    unitIDs,
        AliveUnits: aliveUnits,
        TotalUnits: len(unitIDs),
        CurrentHP:  totalHP,
        MaxHP:      maxHP,
        // ... other fields
    }
}
```

**Lessons:**
- Separate query service for read-only operations
- Returns aggregated data (no need for UI to query multiple times)
- UI-friendly data structures (not raw ECS components)
- Services handle mutations, GUIQueries handles reads

---

### B. Needs Improvement Examples (Refactor These)

#### Example 1: SquadManagementMode (Direct ECS Access)

**File**: `gui/guisquads/squadmanagementmode.go`

**Problem:**
```go
// ❌ UI mode directly queries ECS
func (smm *SquadManagementMode) createUnitList(squadID ecs.EntityID) *widget.List {
    unitIDs := squads.GetUnitIDsInSquad(squadID, smm.Queries.ECSManager)

    entries := make([]interface{}, 0, len(unitIDs))
    for _, unitID := range unitIDs {
        // ❌ UI code directly accessing ECS components
        if attrRaw, ok := smm.Context.ECSManager.GetComponent(unitID, common.AttributeComponent); ok {
            attr := attrRaw.(*common.Attributes)
            if nameRaw, ok := smm.Context.ECSManager.GetComponent(unitID, common.NameComponent); ok {
                name := nameRaw.(*common.Name)
                nameStr = name.NameStr
            }
            entries = append(entries, fmt.Sprintf("%s - HP: %d/%d", nameStr, attr.CurrentHealth, attr.MaxHealth))
        }
    }

    return widgets.CreateListWithConfig(/* ... */)
}

// ❌ No service for squad operations (disband, rename, etc.)
// Missing: DisbandSquad(), RenameSquad(), MergeSquads(), etc.
```

**Why It's Problematic:**
- UI knows about ECS component structure
- No service layer for squad management operations
- Can't test squad operations without UI
- No structured results for success/failure

**How to Fix (Apply Approach 1):**
```go
// ✅ Create SquadManagementService
type SquadManagementService struct {
    entityManager *common.EntityManager
}

type UnitDisplayInfo struct {
    UnitID      ecs.EntityID
    Name        string
    CurrentHP   int
    MaxHP       int
    DisplayText string
}

func (sms *SquadManagementService) GetSquadUnitList(squadID ecs.EntityID) []*UnitDisplayInfo {
    unitIDs := squads.GetUnitIDsInSquad(squadID, sms.entityManager)
    result := make([]*UnitDisplayInfo, 0, len(unitIDs))

    for _, unitID := range unitIDs {
        attrs := common.GetAttributesByID(sms.entityManager, unitID)
        name := common.GetName(sms.entityManager, unitID)

        info := &UnitDisplayInfo{
            UnitID:      unitID,
            Name:        name,
            CurrentHP:   attrs.CurrentHealth,
            MaxHP:       attrs.MaxHealth,
            DisplayText: fmt.Sprintf("%s - HP: %d/%d", name, attrs.CurrentHealth, attrs.MaxHealth),
        }
        result = append(result, info)
    }

    return result
}

// ✅ UI uses service
func (smm *SquadManagementMode) createUnitList(squadID ecs.EntityID) *widget.List {
    units := smm.squadManagementService.GetSquadUnitList(squadID)

    entries := make([]interface{}, len(units))
    for i, unit := range units {
        entries[i] = unit.DisplayText
    }

    return widgets.CreateListWithConfig(/* ... */)
}
```

---

#### Example 2: FormationEditor (Business Logic in UI)

**File**: `gui/guisquads/formationeditormode.go`

**Problem** (hypothetical - file not fully examined):
```go
// ❌ Formation validation in UI
func (fem *FormationEditorMode) onApplyFormation() {
    // ❌ Validation logic in UI code
    if fem.currentFormation == nil {
        fem.showError("No formation selected")
        return
    }

    // ❌ Direct ECS manipulation
    squadEntity := common.FindEntityByIDWithTag(fem.Context.ECSManager, fem.editingSquadID, squads.SquadTag)
    if squadEntity == nil {
        return
    }

    // ❌ Business logic: checking position conflicts
    positions := make(map[coords.LogicalPosition]bool)
    for _, assignment := range fem.currentFormation.Assignments {
        pos := coords.LogicalPosition{X: assignment.GridX, Y: assignment.GridY}
        if positions[pos] {
            fem.showError("Formation has position conflicts")
            return
        }
        positions[pos] = true
    }

    // ❌ Direct component updates
    for _, assignment := range fem.currentFormation.Assignments {
        // Update squad member positions...
    }
}
```

**Why It's Problematic:**
- Formation validation logic embedded in UI
- Direct ECS component manipulation
- No structured result for operation outcome
- Can't reuse formation logic elsewhere (AI, scenario editor)

**How to Fix (Apply Approach 1):**
```go
// ✅ Create FormationService
type FormationService struct {
    entityManager *common.EntityManager
}

type FormationValidationResult struct {
    Valid             bool
    Errors            []string
    PositionConflicts []coords.LogicalPosition
}

func (fs *FormationService) ValidateFormation(formation *squads.Formation) *FormationValidationResult {
    result := &FormationValidationResult{
        Valid:  true,
        Errors: make([]string, 0),
    }

    // ✅ Business logic in service
    positions := make(map[coords.LogicalPosition]bool)
    for _, assignment := range formation.Assignments {
        pos := coords.LogicalPosition{X: assignment.GridX, Y: assignment.GridY}
        if positions[pos] {
            result.Valid = false
            result.Errors = append(result.Errors, fmt.Sprintf("Position conflict at (%d, %d)", pos.X, pos.Y))
            result.PositionConflicts = append(result.PositionConflicts, pos)
        }
        positions[pos] = true
    }

    return result
}

type ApplyFormationResult struct {
    Success       bool
    Error         string
    UnitsRepositioned int
}

func (fs *FormationService) ApplyFormation(squadID ecs.EntityID, formation *squads.Formation) *ApplyFormationResult {
    // ✅ Validation
    validation := fs.ValidateFormation(formation)
    if !validation.Valid {
        return &ApplyFormationResult{
            Success: false,
            Error:   strings.Join(validation.Errors, "; "),
        }
    }

    // ✅ Execution
    unitsRepositioned := 0
    for _, assignment := range formation.Assignments {
        // Apply formation positions to squad members
        unitsRepositioned++
    }

    return &ApplyFormationResult{
        Success:           true,
        UnitsRepositioned: unitsRepositioned,
    }
}

// ✅ UI uses service
func (fem *FormationEditorMode) onApplyFormation() {
    result := fem.formationService.ApplyFormation(fem.editingSquadID, fem.currentFormation)

    if result.Success {
        fem.showSuccess(fmt.Sprintf("Formation applied (%d units repositioned)", result.UnitsRepositioned))
    } else {
        fem.showError(result.Error)
    }
}
```

---

### C. Minor Issues (Low Priority)

#### Example 1: ExplorationMode (Coordinate Conversion in UI)

**File**: `gui/guimodes/explorationmode.go`

**Minor Issue:**
```go
// ⚠️ Coordinate conversion logic in UI handler
func (em *ExplorationMode) HandleInput(inputState *core.InputState) bool {
    if inputState.MouseButton == ebiten.MouseButton2 && inputState.MousePressed {
        // ⚠️ UI creating coordinate manager and viewport
        playerPos := *em.Context.PlayerData.Pos
        manager := coords.NewCoordinateManager(graphics.ScreenInfo)
        viewport := coords.NewViewport(manager, playerPos)
        clickedPos := viewport.ScreenToLogical(inputState.MouseX, inputState.MouseY)

        // Transition to info mode
        if infoMode, exists := em.ModeManager.GetMode("info_inspect"); exists {
            if infoModeTyped, ok := infoMode.(*InfoMode); ok {
                infoModeTyped.SetInspectPosition(clickedPos)
                em.ModeManager.RequestTransition(infoMode, "Right-click inspection")
            }
        }
        return true
    }
    return false
}
```

**Why It's Minor:**
- Coordinate conversion is presentation logic (acceptable in UI)
- Not game state mutation
- Isolated to input handling

**If You Want to Clean It Up:**
```go
// ✅ Extract to helper if used in multiple places
func (em *ExplorationMode) screenToLogicalPosition(screenX, screenY int) coords.LogicalPosition {
    playerPos := *em.Context.PlayerData.Pos
    manager := coords.NewCoordinateManager(graphics.ScreenInfo)
    viewport := coords.NewViewport(manager, playerPos)
    return viewport.ScreenToLogical(screenX, screenY)
}

func (em *ExplorationMode) HandleInput(inputState *core.InputState) bool {
    if inputState.MouseButton == ebiten.MouseButton2 && inputState.MousePressed {
        clickedPos := em.screenToLogicalPosition(inputState.MouseX, inputState.MouseY)

        if infoMode, exists := em.ModeManager.GetMode("info_inspect"); exists {
            if infoModeTyped, ok := infoMode.(*InfoMode); ok {
                infoModeTyped.SetInspectPosition(clickedPos)
                em.ModeManager.RequestTransition(infoMode, "Right-click inspection")
            }
        }
        return true
    }
    return false
}
```

---

## PRINCIPLES APPLIED

### Software Engineering Principles

**DRY (Don't Repeat Yourself):**
- **Approach 1**: Services eliminate duplicate validation and query logic across UI modes
- **Approach 2**: Commands encapsulate reusable operations with single implementation
- **Approach 3**: Event handlers react to single event source instead of polling multiple times

**SOLID Principles:**
- **Single Responsibility**:
  - Services own game logic
  - UI modes own presentation
  - GUIQueries own read operations
  - Commands own single operations

- **Open/Closed**:
  - New squad operations extend services without modifying UI
  - New commands implement interface without changing executor
  - New event handlers subscribe without changing publishers

- **Liskov Substitution**:
  - All services return same result pattern
  - All commands implement same interface
  - All events follow same structure

- **Interface Segregation**:
  - GUIQueries for reads, Services for mutations
  - Command interface minimal (Validate/Execute/Undo)
  - Event handlers only get relevant data

- **Dependency Inversion**:
  - UI depends on service abstractions, not ECS
  - Services depend on EntityManager interface
  - Events decouple publishers from subscribers

**KISS (Keep It Simple, Stupid):**
- **Approach 1**: Simple service delegation pattern (already proven in codebase)
- **Approach 2**: Commands may be over-engineering for simple operations
- **Approach 3**: Event-driven adds complexity without clear benefit

**YAGNI (You Aren't Gonna Need It):**
- **Approach 1**: Only create services for operations that need them
- **Approach 2**: Don't implement undo for operations that don't need it
- **Approach 3**: Don't add events until polling becomes a problem

**SLAP (Single Level of Abstraction Principle):**
- Services operate at business logic level
- UI modes operate at presentation level
- Commands operate at operation level
- Events operate at notification level

**Separation of Concerns:**
- **Game Logic**: Services (CombatService, SquadBuilderService, etc.)
- **UI Presentation**: Modes (CombatMode, SquadManagementMode, etc.)
- **Data Queries**: GUIQueries (read-only ECS access)
- **UI State**: BattleMapState, OverworldState (selection, modes)

---

### Go-Specific Best Practices

**Composition Over Inheritance:**
```go
// ✅ BaseMode provides common functionality via embedding
type CombatMode struct {
    gui.BaseMode // Embed, don't inherit
    combatService *combatservices.CombatService
}
```

**Interface Design:**
```go
// ✅ Small, focused interfaces
type SquadCommand interface {
    Validate() error
    Execute() error
    Undo() error
    Description() string
}
```

**Error Handling:**
```go
// ✅ Services return structured results, not just errors
type AttackResult struct {
    Success     bool
    ErrorReason string
    // ... other data
}

// Not: func ExecuteAttack(...) error
// Yes: func ExecuteAttack(...) *AttackResult
```

**Value Receivers vs Pointer Receivers:**
```go
// ✅ Use pointer receivers for services (stateful)
func (cs *CombatService) ExecuteAttack(...) *AttackResult

// ✅ Use value receivers for result objects (immutable)
func (ar AttackResult) String() string
```

**Package Organization:**
```go
// ✅ Clear package boundaries
combat/combatservices/   // Game logic services
gui/guicombat/           // Combat UI
gui/guicomponents/       // Shared UI components
gui/core/                // UI infrastructure
```

---

### Game Development Considerations

**Performance:**
- **Frame Budget**: 60 FPS = 16.67ms per frame
- **Approach 1**: No performance impact (same logic, different location)
- **Approach 2**: Minimal overhead (command object allocation)
- **Approach 3**: Could improve performance (no polling) BUT adds complexity

**Real-Time Constraints:**
- **UI updates must be immediate**: Event-driven helps, but polling is already fast enough
- **Game loop integration**: Services fit naturally into game loop
- **Input latency**: Direct service calls have zero additional latency

**Tactical Gameplay Preservation:**
- **Turn-based mechanics**: Services preserve turn manager state correctly
- **Squad formations**: FormationService maintains tactical positioning
- **Combat resolution**: CombatService handles all combat rules consistently

**Memory Management:**
- **ECS entity lifecycle**: Services use EntityManager properly
- **Position system updates**: Services update GlobalPositionSystem
- **Component cleanup**: Services handle entity disposal

---

## NEXT STEPS

### Recommended Action Plan

**Immediate (This Week):**
1. **Create SquadManagementService** (2 hours)
   - File: `squads/squadservices/squad_management_service.go`
   - Methods: `GetSquadStats`, `DisbandSquad`, `RenameSquad`
   - Tests: `squad_management_service_test.go`

2. **Create FormationService** (2 hours)
   - File: `squads/squadservices/formation_service.go`
   - Methods: `ValidateFormation`, `ApplyFormation`, `SaveFormationPreset`
   - Tests: `formation_service_test.go`

3. **Refactor SquadManagementMode** (1.5 hours)
   - Inject SquadManagementService
   - Replace direct ECS access with service calls
   - Update to use structured results

4. **Refactor FormationEditor** (1.5 hours)
   - Inject FormationService
   - Remove validation logic from UI
   - Use service for formation operations

**Short-Term (Next Week):**
1. **Create SquadDeploymentService** (1.5 hours)
   - Handle deployment tile validation
   - Manage deployment state

2. **Review and standardize result patterns** (1 hour)
   - Ensure all services return consistent result objects
   - Document result patterns

3. **Add service integration tests** (2 hours)
   - Test service interactions with ECS
   - Verify state changes

**Medium-Term (Next Sprint):**
1. **Evaluate need for Command Pattern** (research task)
   - Gather user feedback on undo/redo desire
   - Profile squad operation performance
   - Decide if worth the investment

2. **Consider event-driven updates** (research task)
   - Profile UI update performance
   - Identify actual bottlenecks
   - Only implement if proven need

3. **Documentation and examples** (ongoing)
   - Document service creation pattern
   - Add examples for new developers
   - Update CLAUDE.md with service guidelines

**Long-Term (Future Considerations):**
1. **Service layer for all game systems**
   - InventoryService
   - ItemService
   - WorldGenerationService
   - etc.

2. **Enhanced testing infrastructure**
   - Service mocking utilities
   - Integration test helpers
   - UI test automation

3. **Developer tools**
   - Service call logging
   - Performance profiling
   - Debug UI for service state

---

### Validation Strategy

**Testing Approach:**

1. **Unit Tests for Services**
   ```go
   func TestSquadManagementService_DisbandSquad(t *testing.T) {
       // Setup ECS with test squad
       manager := setupTestECS()
       service := NewSquadManagementService(manager)

       // Create test squad
       squadID := createTestSquad(manager, "Test Squad", 3)
       playerID := createTestPlayer(manager)

       // Execute disband
       result := service.DisbandSquad(playerID, squadID)

       // Verify result
       if !result.Success {
           t.Fatalf("Expected success, got error: %s", result.Error)
       }
       if result.UnitsReturned != 3 {
           t.Errorf("Expected 3 units returned, got %d", result.UnitsReturned)
       }

       // Verify squad destroyed
       squadEntity := common.FindEntityByIDWithTag(manager, squadID, squads.SquadTag)
       if squadEntity != nil {
           t.Error("Expected squad to be destroyed")
       }
   }
   ```

2. **Integration Tests for UI-Service Interaction**
   ```go
   func TestSquadManagementMode_DisbandSquad(t *testing.T) {
       // Setup UI mode with service
       mode := setupTestSquadManagementMode()

       // Create test squad
       squadID := createTestSquad(mode.Context.ECSManager, "Test Squad", 3)
       mode.currentSquadID = squadID

       // Execute UI action
       mode.onDisbandSquad()

       // Verify UI state updated
       if mode.currentSquadID != 0 {
           t.Error("Expected currentSquadID to be cleared")
       }

       // Verify log message
       // Verify squad list refreshed
   }
   ```

3. **Manual Testing Checklist**
   - [ ] Create squad → verify roster updated
   - [ ] Disband squad → verify units returned
   - [ ] Rename squad → verify name changed
   - [ ] Change formation → verify positions updated
   - [ ] Deploy squad → verify deployment state
   - [ ] Recall squad → verify return to roster
   - [ ] Error cases → verify appropriate messages

**Rollback Plan:**

If refactoring causes issues:

1. **Git branch strategy**
   ```bash
   # Before starting
   git checkout -b feature/service-layer-completion
   git commit -m "Before refactoring: baseline"

   # After each service
   git commit -m "Added SquadManagementService"
   git commit -m "Refactored SquadManagementMode to use service"

   # If issues arise
   git revert HEAD  # Undo last commit
   # or
   git checkout main  # Return to working state
   ```

2. **Feature flags** (if needed)
   ```go
   const useSquadManagementService = true

   if useSquadManagementService {
       result := smm.squadService.DisbandSquad(...)
   } else {
       // Old direct ECS code
   }
   ```

3. **Incremental rollout**
   - Refactor one UI mode at a time
   - Test thoroughly before moving to next
   - Keep old code commented until confirmed working

**Success Metrics:**

1. **Code Metrics**
   - Lines of UI code reduced by 20-30%
   - Lines of service code increased (but testable)
   - Cyclomatic complexity reduced in UI modes

2. **Test Coverage**
   - Service layer: 80%+ coverage
   - UI modes: Focus on integration tests

3. **Developer Experience**
   - New features use service pattern consistently
   - Reduced ECS access in UI code
   - Easier to add squad operations

4. **User Experience**
   - No regressions in existing functionality
   - Better error messages from services
   - (If Command Pattern added) Undo/redo works reliably

---

### Additional Resources

**Go Patterns Documentation:**
- https://go.dev/doc/effective_go
- https://github.com/golang/go/wiki/CodeReviewComments
- https://dave.cheney.net/practical-go/presentations/gophercon-singapore-2019.html

**Game Architecture References:**
- "Game Programming Patterns" by Robert Nystrom (especially Service Locator, Command, Observer chapters)
- ECS best practices: https://github.com/SanderMertens/ecs-faq
- Tactical game architecture: https://www.gamedeveloper.com/design/designing-a-robust-input-handling-system-for-a-turn-based-game

**Refactoring Resources:**
- "Refactoring: Improving the Design of Existing Code" by Martin Fowler
- "Working Effectively with Legacy Code" by Michael Feathers
- Go refactoring tools: https://github.com/godoctor/godoctor

**Testing in Go:**
- https://go.dev/doc/tutorial/add-a-test
- https://quii.gitbook.io/learn-go-with-tests/
- Table-driven tests: https://dave.cheney.net/2019/05/07/prefer-table-driven-tests

---

## CONCLUSION

The TinkerRogue codebase already demonstrates excellent separation of concerns in some areas (`CombatService`, `SquadBuilderService`, `GUIQueries`). The primary recommendation is to **complete the service layer pattern** by applying it consistently across all UI interactions.

**Key Takeaways:**

1. **Follow existing patterns**: `CombatService` and `SquadBuilderService` are perfect examples - replicate this pattern for remaining operations

2. **Incremental approach**: Extract services one UI mode at a time, test thoroughly, then move on

3. **Avoid over-engineering**:
   - ✅ Services for game logic and mutations
   - ✅ GUIQueries for read-only display data
   - ❌ Don't create services for trivial operations
   - ❌ Don't add Command Pattern unless undo is truly needed
   - ❌ Don't add events unless polling is proven bottleneck

4. **Maintain game context**: This is a tactical roguelike, not a web app - keep patterns appropriate for game development

5. **Test as you go**: Each service should have tests before relying on it

**Next Concrete Steps:**

1. Create `SquadManagementService` (2 hours)
2. Create `FormationService` (2 hours)
3. Refactor SquadManagementMode (1.5 hours)
4. Refactor FormationEditor (1.5 hours)
5. Add tests for new services (1 hour)

**Total estimated time: 8 hours (1 day of focused work)**

This will bring the GUI separation to a consistently high standard across the entire codebase while maintaining simplicity and following established patterns.

---

END OF ANALYSIS
