# Action System Design Document

**Created:** 2025-12-24
**Status:** Design Document
**Purpose:** Define architecture for queued action system supporting both player and AI actions

---

## Table of Contents

1. [Overview](#overview)
2. [Action System Architecture](#action-system-architecture)
3. [Package Structure](#package-structure)
4. [Component Design](#component-design)
5. [Action Types & Interface](#action-types--interface)
6. [Action Queue Pattern](#action-queue-pattern)
7. [Execution Pipeline](#execution-pipeline)
8. [AI vs Player Actions](#ai-vs-player-actions)
9. [Extensibility](#extensibility)
10. [Implementation Considerations](#implementation-considerations)
11. [Integration Points](#integration-points)

---

## Overview

### What is the Action System?

A unified system for representing, validating, queuing, and executing discrete tactical actions (move, attack, ability use) for both player-controlled and AI-controlled units.

### Design Goals

1. **Unified Execution**: Same action types used by player and AI
2. **Queueable**: Actions can be queued and processed in sequence
3. **Extensible**: New action types added without modifying core system
4. **Validatable**: Actions checked before execution (range, resources, state)
5. **Interruptible**: Actions can fail or be modified mid-execution
6. **ECS-Compliant**: Pure data components, query-based, system functions

### TRPG Inspiration

- **Fire Emblem**: Action → Preview → Confirm → Execute
- **FFT**: Charge Time → Action Queue → Resolution
- **XCOM**: Action Points → Multiple Actions → Turn Resolution
- **Into the Breach**: Preview Outcomes → Execute All → Resolve

---

## Action System Architecture

### High-Level Flow

```
Decision Making → Action Creation → Validation → Queue → Execution → Resolution
     (AI/Player)      (Factory)      (System)    (Component)  (System)    (Effects)
```

### Separation of Concerns

| Layer | Responsibility | Examples |
|-------|---------------|----------|
| **Decision** | Choose what to do | Player input, AI pathfinding |
| **Action** | Data describing intent | MoveAction{From, To}, AttackAction{Attacker, Target} |
| **Validation** | Can action execute? | Range check, resource check, state check |
| **Queue** | Store pending actions | ActionQueue component |
| **Execution** | Apply effects | Move unit, deal damage, apply status |
| **Resolution** | Handle outcomes | Death, counter-attack, chain reactions |

### Key Insight: Shared Execution, Separate Decision

```
Player Input → MoveAction ─┐
                          ├→ ValidateAction() → QueueAction() → ExecuteAction()
AI Pathfinding → MoveAction ─┘
```

**Same action data, same execution, different creation.**

---

## Package Structure

### Recommended Organization

```
actions/
├── components.go          # ActionQueue, ActionState components
├── actiontypes.go         # Action interface, base types
├── actions_move.go        # MoveAction implementation
├── actions_attack.go      # AttackAction implementation
├── actions_ability.go     # AbilityAction implementation
├── queries.go            # GetPendingActions, HasQueuedActions, etc.
├── validation.go         # ValidateAction, CanExecute checks
├── execution.go          # ExecuteAction, ProcessQueue
├── factory.go            # CreateMoveAction, CreateAttackAction helpers
└── actions_test.go       # Tests

ai/
├── components.go          # AIControllerData (separate from actions)
├── decision.go           # AI decision-making logic
├── targeting.go          # AI target selection
├── pathfinding.go        # AI movement planning
└── ai_test.go

combat/
├── attackingsystem.go    # PerformAttack (called by AttackAction)
├── combatstate.go        # Combat resolution state
└── turnmanager.go        # Turn order, action point tracking
```

### Why Separate AI and Actions?

- **Actions**: Pure data describing *what* to do (shared by all)
- **AI**: Decision-making logic for *choosing* what to do (AI-specific)
- **Combat**: Game rules for *resolving* actions (shared by all)

---

## Component Design

### ActionQueueComponent

Stores pending actions for an entity.

```go
// components.go

// ActionQueueData holds a sequence of actions to execute
type ActionQueueData struct {
    Actions []Action          // Ordered list of pending actions
    CurrentIndex int          // Which action is currently executing
    Paused bool              // Queue processing paused (e.g., waiting for animation)
}

var ActionQueueComponent = ecs.NewComponentType[*ActionQueueData]()

// Tag for entities with action queues (query optimization)
var HasActionsTag = ecs.NewTag()
```

### ActionStateComponent

Tracks execution state of current action.

```go
// ActionStateData represents the state of an action being executed
type ActionStateData struct {
    CurrentAction Action       // The action being executed
    State ActionExecutionState // Current execution state
    StartTime int64           // When action started (for timing)
    Progress float64          // 0.0 to 1.0 (for animations)
    Error string              // If action failed, why?
}

type ActionExecutionState int

const (
    ActionNotStarted ActionExecutionState = iota
    ActionValidating
    ActionExecuting
    ActionCompleted
    ActionFailed
    ActionCancelled
)

var ActionStateComponent = ecs.NewComponentType[*ActionStateData]()
```

### ECS Compliance

- **Pure Data**: No methods, only fields
- **No Entity Pointers**: Actions store `ecs.EntityID`, not `*ecs.Entity`
- **Tag for Queries**: `HasActionsTag` for efficient filtering
- **Separate State**: Queue (persistent) vs State (current execution)

---

## Action Types & Interface

### Action Interface

```go
// actiontypes.go

// Action represents a discrete game action that can be queued and executed
type Action interface {
    // Execute performs the action's effects
    // Returns error if action fails during execution
    Execute(manager *common.EntityManager) error

    // Validate checks if action can be executed
    // Returns error describing why action is invalid
    Validate(manager *common.EntityManager) error

    // ActorID returns the entity performing the action
    ActorID() ecs.EntityID

    // Type returns the action type (for debugging, logging, UI)
    Type() ActionType

    // Description returns human-readable description
    Description() string

    // Clone creates a copy of the action (for previewing)
    Clone() Action
}

type ActionType int

const (
    ActionTypeMove ActionType = iota
    ActionTypeAttack
    ActionTypeAbility
    ActionTypeWait
    ActionTypeItem
    // Extensible - add new types here
)

func (at ActionType) String() string {
    switch at {
    case ActionTypeMove: return "Move"
    case ActionTypeAttack: return "Attack"
    case ActionTypeAbility: return "Ability"
    case ActionTypeWait: return "Wait"
    case ActionTypeItem: return "Item"
    default: return "Unknown"
    }
}
```

### Base Action (Optional)

Reduce boilerplate for common fields.

```go
// BaseAction provides common fields for all actions
type BaseAction struct {
    Actor ecs.EntityID  // Entity performing action
}

func (ba *BaseAction) ActorID() ecs.EntityID {
    return ba.Actor
}
```

---

## Action Queue Pattern

### Queueing Actions

```go
// execution.go

// QueueAction adds an action to entity's action queue
// Creates ActionQueue component if it doesn't exist
func QueueAction(manager *common.EntityManager, actorID ecs.EntityID, action Action) error {
    // Validate before queueing
    if err := action.Validate(manager); err != nil {
        return fmt.Errorf("invalid action: %w", err)
    }

    actor := manager.GetEntityByID(actorID)
    if actor == nil {
        return fmt.Errorf("actor entity not found: %d", actorID)
    }

    // Get or create action queue
    queueData := common.GetComponentType[*ActionQueueData](actor, ActionQueueComponent)
    if queueData == nil {
        queueData = &ActionQueueData{
            Actions: make([]Action, 0),
            CurrentIndex: 0,
            Paused: false,
        }
        actor.AddComponent(ActionQueueComponent, queueData)
        actor.AddTag(HasActionsTag)
    }

    // Add to queue
    queueData.Actions = append(queueData.Actions, action)

    return nil
}

// ClearActionQueue removes all pending actions
func ClearActionQueue(manager *common.EntityManager, actorID ecs.EntityID) {
    actor := manager.GetEntityByID(actorID)
    if actor == nil {
        return
    }

    queueData := common.GetComponentType[*ActionQueueData](actor, ActionQueueComponent)
    if queueData != nil {
        queueData.Actions = nil
        queueData.CurrentIndex = 0
        actor.RemoveTag(HasActionsTag)
    }
}

// RemoveCurrentAction removes the action being executed
func RemoveCurrentAction(manager *common.EntityManager, actorID ecs.EntityID) {
    actor := manager.GetEntityByID(actorID)
    if actor == nil {
        return
    }

    queueData := common.GetComponentType[*ActionQueueData](actor, ActionQueueComponent)
    if queueData != nil && len(queueData.Actions) > 0 {
        queueData.Actions = queueData.Actions[1:]
        queueData.CurrentIndex = 0

        if len(queueData.Actions) == 0 {
            actor.RemoveTag(HasActionsTag)
        }
    }
}
```

### Processing Queue

```go
// ProcessActionQueues executes one action per entity with pending actions
// Call this each turn/tick to progress actions
func ProcessActionQueues(manager *common.EntityManager) {
    for _, result := range manager.World.Query(HasActionsTag) {
        entity := result.Entity

        queueData := common.GetComponentType[*ActionQueueData](entity, ActionQueueComponent)
        if queueData == nil || queueData.Paused || len(queueData.Actions) == 0 {
            continue
        }

        // Get current action
        currentAction := queueData.Actions[0]

        // Execute
        if err := executeActionWithState(manager, entity, currentAction); err != nil {
            log.Printf("Action execution failed for entity %d: %v", entity.ID, err)
            // Remove failed action
            RemoveCurrentAction(manager, entity.ID)
        }
    }
}

// executeActionWithState wraps execution with state tracking
func executeActionWithState(manager *common.EntityManager, entity *ecs.Entity, action Action) error {
    // Create or get action state
    stateData := common.GetComponentType[*ActionStateData](entity, ActionStateComponent)
    if stateData == nil {
        stateData = &ActionStateData{
            CurrentAction: action,
            State: ActionNotStarted,
            StartTime: 0, // TODO: Get game time
            Progress: 0.0,
        }
        entity.AddComponent(ActionStateComponent, stateData)
    }

    // Validate
    stateData.State = ActionValidating
    if err := action.Validate(manager); err != nil {
        stateData.State = ActionFailed
        stateData.Error = err.Error()
        return err
    }

    // Execute
    stateData.State = ActionExecuting
    if err := action.Execute(manager); err != nil {
        stateData.State = ActionFailed
        stateData.Error = err.Error()
        return err
    }

    // Complete
    stateData.State = ActionCompleted
    stateData.Progress = 1.0

    // Remove completed action from queue
    RemoveCurrentAction(manager, entity.ID)

    // Clear state
    entity.RemoveComponent(ActionStateComponent)

    return nil
}
```

---

## Execution Pipeline

### Execution Stages

```
1. Validate    → Check preconditions (range, resources, state)
2. Pre-Execute → Deduct resources (AP, MP), trigger "before" effects
3. Execute     → Apply core effect (move position, deal damage)
4. Post-Execute → Trigger reactions (counter-attack, overwatch)
5. Resolve     → Handle consequences (death, status effects)
```

### Validation Example

```go
// validation.go

// ValidateAction checks if action can be executed
func ValidateAction(manager *common.EntityManager, action Action) error {
    // Common validations
    actor := manager.GetEntityByID(action.ActorID())
    if actor == nil {
        return fmt.Errorf("actor does not exist")
    }

    // Check if actor is alive
    if !isAlive(actor) {
        return fmt.Errorf("actor is dead")
    }

    // Check if actor can act this turn
    if !canAct(actor) {
        return fmt.Errorf("actor cannot act this turn")
    }

    // Delegate to action-specific validation
    return action.Validate(manager)
}

// isAlive checks if entity is alive (has health > 0)
func isAlive(entity *ecs.Entity) bool {
    // TODO: Check health component
    return true
}

// canAct checks if entity can perform actions (not stunned, has AP, etc.)
func canAct(entity *ecs.Entity) bool {
    // TODO: Check action points, status effects
    return true
}
```

### Interruption Handling

```go
// InterruptAction cancels current action (e.g., stunned during move)
func InterruptAction(manager *common.EntityManager, actorID ecs.EntityID, reason string) {
    actor := manager.GetEntityByID(actorID)
    if actor == nil {
        return
    }

    stateData := common.GetComponentType[*ActionStateData](actor, ActionStateComponent)
    if stateData != nil {
        stateData.State = ActionCancelled
        stateData.Error = reason

        // Optionally: partial effect based on progress
        // e.g., if 50% through move, stop at midpoint
    }

    // Clear current action
    RemoveCurrentAction(manager, actorID)
}
```

---

## AI vs Player Actions

### Shared Action Types

Both player and AI create the same action structs:

```go
// actions_move.go

// MoveAction represents movement from one position to another
type MoveAction struct {
    BaseAction
    From coords.LogicalPosition
    To   coords.LogicalPosition
    Path []coords.LogicalPosition // Optional: planned path
}

func (ma *MoveAction) Type() ActionType {
    return ActionTypeMove
}

func (ma *MoveAction) Description() string {
    return fmt.Sprintf("Move from %v to %v", ma.From, ma.To)
}

func (ma *MoveAction) Validate(manager *common.EntityManager) error {
    // Check distance
    distance := manhattanDistance(ma.From, ma.To)

    // Check movement range
    actor := manager.GetEntityByID(ma.Actor)
    moveRange := getMovementRange(actor) // TODO: Get from stats

    if distance > moveRange {
        return fmt.Errorf("destination too far: %d > %d", distance, moveRange)
    }

    // Check destination is empty
    entitiesAtDest := common.GlobalPositionSystem.GetEntitiesAtPosition(ma.To)
    if len(entitiesAtDest) > 0 {
        return fmt.Errorf("destination occupied")
    }

    // TODO: Check path validity (pathfinding)

    return nil
}

func (ma *MoveAction) Execute(manager *common.EntityManager) error {
    // Update position component
    actor := manager.GetEntityByID(ma.Actor)
    if actor == nil {
        return fmt.Errorf("actor not found")
    }

    posData := common.GetComponentType[*common.PositionData](actor, common.PositionComponent)
    if posData == nil {
        return fmt.Errorf("actor has no position component")
    }

    // Update position system
    common.GlobalPositionSystem.RemoveEntity(ma.Actor, posData.LogicalPosition)
    posData.LogicalPosition = ma.To
    common.GlobalPositionSystem.AddEntity(ma.Actor, ma.To)

    // Update pixel position (for rendering)
    posData.PixelPosition = coords.CoordManager.LogicalToPixel(ma.To)

    return nil
}

func (ma *MoveAction) Clone() Action {
    clone := *ma
    if ma.Path != nil {
        clone.Path = make([]coords.LogicalPosition, len(ma.Path))
        copy(clone.Path, ma.Path)
    }
    return &clone
}
```

### Player Action Creation

```go
// From input controller

func (gc *GameController) HandleClick(pixelPos coords.PixelPosition) {
    logicalPos := coords.CoordManager.PixelToLogical(pixelPos)

    // If unit selected and clicking empty tile, create move action
    if gc.selectedUnit != 0 {
        action := &actions.MoveAction{
            BaseAction: actions.BaseAction{Actor: gc.selectedUnit},
            From: gc.selectedUnitPos,
            To: logicalPos,
        }

        // Queue action (will be validated)
        if err := actions.QueueAction(gc.manager, gc.selectedUnit, action); err != nil {
            log.Printf("Cannot move: %v", err)
            return
        }

        gc.selectedUnit = 0
    }
}
```

### AI Action Creation

```go
// From ai/decision.go

func (ai *AIController) DecideAction(manager *common.EntityManager, actorID ecs.EntityID) {
    // AI uses pathfinding to choose destination
    actor := manager.GetEntityByID(actorID)
    currentPos := getCurrentPosition(actor)

    // AI logic: find nearest enemy
    target := ai.findNearestEnemy(manager, actorID)
    if target == nil {
        return // No targets, wait
    }

    targetPos := getCurrentPosition(manager.GetEntityByID(target.ID))

    // Can we attack from here?
    if manhattanDistance(currentPos, targetPos) <= 1 {
        // Attack action
        action := &actions.AttackAction{
            BaseAction: actions.BaseAction{Actor: actorID},
            Target: target.ID,
        }
        actions.QueueAction(manager, actorID, action)
    } else {
        // Move closer
        path := ai.findPath(currentPos, targetPos)
        if len(path) > 1 {
            action := &actions.MoveAction{
                BaseAction: actions.BaseAction{Actor: actorID},
                From: currentPos,
                To: path[1], // Move one step
                Path: path,
            }
            actions.QueueAction(manager, actorID, action)
        }
    }
}
```

### Key Insight

**Same `MoveAction` struct, same `Execute()`, different creation:**

- **Player**: Created from mouse click
- **AI**: Created from pathfinding algorithm

**Execution pipeline doesn't care who created the action.**

---

## Extensibility

### Adding New Action Type

Example: Adding `AbilityAction` for special abilities.

**Step 1: Define Action**

```go
// actions_ability.go

// AbilityAction represents using a special ability
type AbilityAction struct {
    BaseAction
    AbilityID string              // Which ability to use
    Targets   []ecs.EntityID      // Targets (may be multiple for AOE)
    TargetPos *coords.LogicalPosition // Optional position target
}

func (aa *AbilityAction) Type() ActionType {
    return ActionTypeAbility
}

func (aa *AbilityAction) Description() string {
    return fmt.Sprintf("Use ability %s", aa.AbilityID)
}

func (aa *AbilityAction) Validate(manager *common.EntityManager) error {
    actor := manager.GetEntityByID(aa.Actor)

    // Check if actor has the ability
    if !hasAbility(actor, aa.AbilityID) {
        return fmt.Errorf("actor does not have ability %s", aa.AbilityID)
    }

    // Check ability cooldown
    if isOnCooldown(actor, aa.AbilityID) {
        return fmt.Errorf("ability on cooldown")
    }

    // Check resource cost (mana, stamina, etc.)
    if !canAfford(actor, aa.AbilityID) {
        return fmt.Errorf("insufficient resources")
    }

    // Validate targets (range, validity)
    return validateAbilityTargets(manager, aa)
}

func (aa *AbilityAction) Execute(manager *common.EntityManager) error {
    // Deduct resources
    deductAbilityCost(manager, aa.Actor, aa.AbilityID)

    // Apply ability effects
    return executeAbilityEffects(manager, aa)
}

func (aa *AbilityAction) Clone() Action {
    clone := *aa
    if aa.Targets != nil {
        clone.Targets = make([]ecs.EntityID, len(aa.Targets))
        copy(clone.Targets, aa.Targets)
    }
    return &clone
}
```

**Step 2: Add to ActionType Enum**

```go
// actiontypes.go

const (
    ActionTypeMove ActionType = iota
    ActionTypeAttack
    ActionTypeAbility  // ← Added
    ActionTypeWait
    ActionTypeItem
)
```

**Step 3: Use in Player/AI**

```go
// Player input
action := &actions.AbilityAction{
    BaseAction: actions.BaseAction{Actor: selectedUnit},
    AbilityID: "fireball",
    Targets: []ecs.EntityID{targetEnemy},
}
actions.QueueAction(manager, selectedUnit, action)

// AI decision
action := &actions.AbilityAction{
    BaseAction: actions.BaseAction{Actor: aiUnit},
    AbilityID: "heal",
    Targets: []ecs.EntityID{woundedAlly},
}
actions.QueueAction(manager, aiUnit, action)
```

**No changes needed to:**
- `ProcessActionQueues()` (generic over `Action` interface)
- `QueueAction()` (works with any `Action`)
- `ValidateAction()` (delegates to action's `Validate()`)

### Extensibility Benefits

1. **Open/Closed Principle**: Open for extension (new actions), closed for modification (core system unchanged)
2. **Interface-Based**: Any type implementing `Action` works
3. **No Switch Statements**: Polymorphism via interface methods
4. **Testable**: Each action type tested independently

---

## Implementation Considerations

### 1. Action Validation and Failure States

**Validation Timing:**
- **Queue Time**: Validate when action is queued (fail fast)
- **Execute Time**: Re-validate before execution (state may have changed)

**Example: State Change Between Queue and Execute**

```go
// Turn 1: AI queues "Move to (5,5)"
actions.QueueAction(manager, aiUnit, moveAction)  // Valid: (5,5) is empty

// Turn 2: Player moves to (5,5) first
// Player's move executes, occupies (5,5)

// Turn 3: AI's move executes
moveAction.Validate(manager)  // Now invalid: (5,5) occupied!
```

**Solution: Re-validate before execute**

```go
func executeActionWithState(...) error {
    // Always re-validate
    if err := action.Validate(manager); err != nil {
        stateData.State = ActionFailed
        stateData.Error = "Conditions changed: " + err.Error()
        return err
    }

    // Then execute
    action.Execute(manager)
}
```

**Failure Handling Options:**
1. **Skip Action**: Remove from queue, unit loses turn
2. **Retry Decision**: AI picks new action
3. **Fallback Action**: Default to "Wait" action

---

### 2. Animation and Async Execution

**Problem:** Actions may take time to animate (unit walks, projectile flies).

**Solution: Pause Queue During Animation**

```go
// Pause queue while action animates
func StartActionAnimation(manager *common.EntityManager, actorID ecs.EntityID) {
    actor := manager.GetEntityByID(actorID)
    queueData := common.GetComponentType[*ActionQueueData](actor, ActionQueueComponent)
    if queueData != nil {
        queueData.Paused = true
    }
}

// Resume when animation completes
func OnAnimationComplete(manager *common.EntityManager, actorID ecs.EntityID) {
    actor := manager.GetEntityByID(actorID)
    queueData := common.GetComponentType[*ActionQueueData](actor, ActionQueueComponent)
    if queueData != nil {
        queueData.Paused = false
    }
}
```

**Alternative: Action Progress Tracking**

```go
// ActionStateData has Progress field
stateData.Progress = animationTime / totalAnimationTime

// Execute in stages
if stateData.Progress < 1.0 {
    // Still animating, update progress
    stateData.Progress += deltaTime / actionDuration
    return nil // Not complete yet
}

// Animation complete, finalize action
finalizeAction(manager, action)
```

---

### 3. Interrupting Actions

**Interrupt Scenarios:**
- Unit takes damage mid-movement (knocked back)
- Unit stunned during attack wind-up
- Target dies before attack resolves

**Partial Execution:**

```go
func (ma *MoveAction) Execute(manager *common.EntityManager) error {
    // If interrupted mid-path, stop at current position
    stateData := getActionState(manager, ma.Actor)

    if stateData.State == ActionCancelled {
        // Partial move: stop at progress point
        currentStep := int(stateData.Progress * float64(len(ma.Path)))
        if currentStep < len(ma.Path) {
            ma.To = ma.Path[currentStep]
        }
    }

    // Apply final position
    updatePosition(manager, ma.Actor, ma.To)
    return nil
}
```

**Clean Interruption:**

```go
// On stun effect applied
func ApplyStun(manager *common.EntityManager, targetID ecs.EntityID) {
    // Cancel current action
    actions.InterruptAction(manager, targetID, "stunned")

    // Clear action queue (cannot act while stunned)
    actions.ClearActionQueue(manager, targetID)
}
```

---

### 4. Turn Order Integration

**Turn Manager Integration:**

```go
// combat/turnmanager.go

func (tm *TurnManager) ProcessTurn(manager *common.EntityManager) {
    currentUnit := tm.getCurrentUnit()

    // Check if unit has queued actions
    actor := manager.GetEntityByID(currentUnit)
    queueData := common.GetComponentType[*actions.ActionQueueData](actor, actions.ActionQueueComponent)

    if queueData != nil && len(queueData.Actions) > 0 {
        // Execute one action
        actions.ProcessActionQueues(manager)

        // Check if action consumed turn
        if queueData == nil || len(queueData.Actions) == 0 {
            // Turn complete, advance to next unit
            tm.nextTurn()
        }
    } else {
        // No actions queued, prompt for decision
        if isAIControlled(actor) {
            ai.DecideAction(manager, currentUnit)
        } else {
            // Wait for player input
        }
    }
}
```

**Action Point Economy (Optional):**

```go
type ActionPointData struct {
    Current int
    Maximum int
}

var ActionPointComponent = ecs.NewComponentType[*ActionPointData]()

// Each action has a cost
type Action interface {
    // ... existing methods
    APCost() int  // New method
}

// Validate checks AP
func (ma *MoveAction) Validate(manager *common.EntityManager) error {
    actor := manager.GetEntityByID(ma.Actor)
    apData := common.GetComponentType[*ActionPointData](actor, ActionPointComponent)

    if apData.Current < ma.APCost() {
        return fmt.Errorf("insufficient action points: %d < %d", apData.Current, ma.APCost())
    }

    // ... other validations
}

// Execute deducts AP
func (ma *MoveAction) Execute(manager *common.EntityManager) error {
    actor := manager.GetEntityByID(ma.Actor)
    apData := common.GetComponentType[*ActionPointData](actor, ActionPointComponent)

    apData.Current -= ma.APCost()

    // ... execute move
}
```

---

## Integration Points

### Existing Systems Modified

| System | File | Changes |
|--------|------|---------|
| **Combat** | `combat/attackingsystem.go` | Called by `AttackAction.Execute()` |
| **Position** | `systems/positionsystem.go` | Updated by `MoveAction.Execute()` |
| **Turn Manager** | `combat/turnmanager.go` | Calls `ProcessActionQueues()` each turn |
| **Input** | `input/inputcoordinator.go` | Creates actions from player input |
| **AI** | `ai/decision.go` | Creates actions from AI logic |

### New Systems Created

| System | File | Purpose |
|--------|------|---------|
| **Action Queue** | `actions/execution.go` | Queue management, execution pipeline |
| **Validation** | `actions/validation.go` | Pre-execution checks |
| **Action Types** | `actions/actions_*.go` | Concrete action implementations |

### Data Flow Example

```
Player Click
    ↓
InputCoordinator.HandleClick()
    ↓
CreateMoveAction()
    ↓
QueueAction() → Validate() → Add to ActionQueueComponent
    ↓
TurnManager.ProcessTurn()
    ↓
ProcessActionQueues()
    ↓
MoveAction.Execute() → GlobalPositionSystem.Update()
    ↓
RemoveCurrentAction()
    ↓
TurnManager.NextTurn()
```

---

## Summary

### Core Principles

1. **Action = Pure Data**: Struct describing intent, not execution logic
2. **Interface-Based**: Common `Action` interface for polymorphism
3. **Queue Pattern**: Actions queued, processed sequentially
4. **Validate-Execute Split**: Check before doing
5. **Shared Execution**: Same pipeline for player and AI
6. **ECS Compliant**: Components for queue/state, systems for logic

### Implementation Checklist

- [ ] Define `Action` interface
- [ ] Create `ActionQueueComponent` and `ActionStateComponent`
- [ ] Implement `MoveAction`, `AttackAction` concrete types
- [ ] Add `QueueAction()`, `ProcessActionQueues()` systems
- [ ] Integrate with `TurnManager`
- [ ] Hook up player input to create actions
- [ ] Implement AI decision-making to create actions
- [ ] Add validation logic
- [ ] Handle interruptions and failures
- [ ] Test with multiple queued actions

### Extensibility Examples

- **New Action Type**: Implement `Action` interface, add to `ActionType` enum
- **Custom Validation**: Override `Validate()` method
- **Multi-Stage Actions**: Use `ActionStateData.Progress` for phased execution
- **Conditional Actions**: Add precondition checks in `Validate()`
- **Reaction Actions**: Queue from post-execute hooks (counter-attack)

### Next Steps

1. Implement core `Action` interface and components
2. Add `MoveAction` and `AttackAction`
3. Integrate with existing `combat/attackingsystem.go`
4. Hook up `TurnManager` to call `ProcessActionQueues()`
5. Test with AI units performing queued moves
6. Extend with `AbilityAction` and action point economy

---

**Document Status:** Ready for Review
**Implementation Estimate:** 8-12 hours for core system + basic actions
**Risk Level:** Medium (affects combat, turn order, AI)
**Complexity:** Medium (new system, multiple integration points)
