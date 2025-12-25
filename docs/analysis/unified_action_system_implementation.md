# Unified Action System Implementation

**Created:** 2025-12-25
**Status:** Implementation Plan
**Purpose:** Integrate queued action system with existing combat, squads, and squadcommands packages

---

## Executive Summary

This document provides a comprehensive implementation plan for integrating the proposed **action queue system** with the existing **command pattern**, preserving the best of both worlds:

**From action_system_design.md:**
- ✅ Action queues for multi-turn movement
- ✅ AI can plan move → attack sequences
- ✅ Shared action types for player and AI
- ✅ Extensible action interface

**From existing squadcommands:**
- ✅ Undo/redo support (critical for tactical games)
- ✅ Command history tracking
- ✅ Integration with combat systems

**Key Architecture Decision:**
- **Actions** = Queueable operations (move, attack, ability)
- **Commands** = Undoable wrappers around action sequences
- Commands can execute single actions OR action sequences
- Both player and AI use the same action types

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Component Design](#component-design)
3. [Action System](#action-system)
4. [Command System](#command-system)
5. [Queue Integration](#queue-integration)
6. [Multi-Turn Movement](#multi-turn-movement)
7. [AI Planning](#ai-planning)
8. [Implementation Phases](#implementation-phases)
9. [Code Examples](#code-examples)
10. [Testing Strategy](#testing-strategy)

---

## Architecture Overview

### Unified System Design

```
┌──────────────────────────────────────────────────────────────┐
│                     Unified Action/Command System             │
└──────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┴─────────────────────┐
         │                                          │
┌────────▼────────┐                      ┌─────────▼────────┐
│  Action Layer   │                      │  Command Layer   │
│  (Queueable)    │                      │  (Undoable)      │
└─────────────────┘                      └──────────────────┘
│                                                            │
├─ MoveAction                             ├─ MoveCommand (single action)
├─ AttackAction                           ├─ AttackCommand (single action)
├─ AbilityAction                          ├─ MoveAttackCommand (sequence)
│                                         └─ AITurnCommand (AI sequence)
│                                                            │
└─────────────┬──────────────────────────────────────────────┘
              │
    ┌─────────▼──────────┐
    │  Execution Layer   │
    │   (Delegates to)   │
    └────────────────────┘
              │
    ┌─────────┼─────────┐
    │         │         │
CombatAction  Combat    Squads
System        Movement  Package
              System
```

### Key Concepts

**Action (Queueable Unit)**
- Pure data describing a single operation
- Can be queued in ActionQueueComponent
- Validates and executes via delegation to systems
- No undo capability (handled by commands)

**Command (Undoable Wrapper)**
- Wraps one or more actions
- Captures state for undo
- Tracked in CommandExecutor history
- Player and AI both use commands

**Queue (Multi-Turn Execution)**
- Squads can have queued actions
- Processed each turn via ProcessActionQueues()
- Enables multi-turn movement and AI planning

---

## Component Design

### 1. Rename Existing ActionStateComponent

**Problem:** combat.ActionStateComponent conflicts with proposed actions.ActionStateComponent

**Solution:**

```go
// tactical/combat/combatcomponents.go

// TurnStateData tracks per-turn action economy for squads
// RENAMED from ActionStateData to avoid conflict
type TurnStateData struct {
    SquadID           ecs.EntityID // Squad this belongs to
    HasActed          bool         // Squad performed major action (attack/ability)
    ActionPoints      int          // Movement points remaining this turn
    MaxActionPoints   int          // Maximum action points per turn
}

var TurnStateComponent = ecs.NewComponentType[*TurnStateData]()
var TurnStateTag ecs.Tag
```

**Rationale:**
- Renamed to clarify it tracks turn economy, not action execution
- Replaces `MovementRemaining` with `ActionPoints` for flexibility
- Enables XCOM-style multiple actions per turn if desired

### 2. Add ActionQueueComponent

**New component** for queueing actions:

```go
// tactical/actions/components.go

package actions

import (
    "github.com/bytearena/ecs"
)

// ActionQueueData holds pending actions for a squad
// Actions execute in order, one per ProcessActionQueues() call
type ActionQueueData struct {
    Actions      []Action  // Ordered list of pending actions
    CurrentIndex int       // Index of action being executed (always 0)
    Paused       bool      // If true, queue processing paused (animations)
}

var ActionQueueComponent = ecs.NewComponentType[*ActionQueueData]()

// Tag for efficient querying
var HasActionsQueuedTag ecs.Tag

// ExecutionStateData tracks progress of current action (OPTIONAL - for animations)
// Most turn-based games don't need this
type ExecutionStateData struct {
    CurrentAction Action            // Action being executed
    State         ExecutionState    // NotStarted/Validating/Executing/Completed/Failed
    Progress      float64          // 0.0 to 1.0 (for animations/progress bars)
    Error         string           // Error message if failed
}

type ExecutionState int

const (
    ExecutionNotStarted ExecutionState = iota
    ExecutionValidating
    ExecutionExecuting
    ExecutionCompleted
    ExecutionFailed
    ExecutionCancelled
)

var ExecutionStateComponent = ecs.NewComponentType[*ExecutionStateData]()
```

**Files to Update:**
- Create `tactical/actions/components.go` - NEW
- Update `tactical/combat/combatcomponents.go` - Rename ActionStateData
- Update all combat/* files referencing ActionStateComponent

---

## Action System

### Action Interface

```go
// tactical/actions/action_interface.go

package actions

import (
    "game_main/common"
    "github.com/bytearena/ecs"
)

// Action represents a discrete tactical operation that can be queued
// Actions are pure data + validation + execution delegation
// Actions do NOT handle undo - that's the Command layer's responsibility
type Action interface {
    // Execute performs the action's effects by delegating to systems
    // Returns error if action fails during execution
    Execute(manager *common.EntityManager) error

    // Validate checks if action can be executed NOW
    // Re-validated before execution (state may have changed since queuing)
    Validate(manager *common.EntityManager) error

    // ActorID returns the squad performing this action
    ActorID() ecs.EntityID

    // Type returns the action type (for UI, logging, filtering)
    Type() ActionType

    // Description returns human-readable description
    Description() string

    // Clone creates a deep copy (for preview/"what-if" scenarios)
    Clone() Action

    // ActionPointCost returns AP cost (for turn economy)
    // Returns 0 if action doesn't consume AP (like Wait)
    ActionPointCost() int
}

// ActionType categorizes actions
type ActionType int

const (
    ActionTypeMove ActionType = iota
    ActionTypeAttack
    ActionTypeAbility
    ActionTypeWait
    ActionTypeChangeFormation
    // Extensible - add new types here
)

func (at ActionType) String() string {
    switch at {
    case ActionTypeMove: return "Move"
    case ActionTypeAttack: return "Attack"
    case ActionTypeAbility: return "Ability"
    case ActionTypeWait: return "Wait"
    case ActionTypeChangeFormation: return "Formation"
    default: return "Unknown"
    }
}

// BaseAction provides common fields for all actions
type BaseAction struct {
    Actor ecs.EntityID  // Squad performing action
}

func (ba *BaseAction) ActorID() ecs.EntityID {
    return ba.Actor
}
```

### MoveAction Implementation

```go
// tactical/actions/move_action.go

package actions

import (
    "fmt"
    "game_main/common"
    "game_main/tactical/combat"
    "game_main/world/coords"

    "github.com/bytearena/ecs"
)

// MoveAction represents tactical movement on the battlefield
// Can represent single-step or multi-step movement
type MoveAction struct {
    BaseAction
    From     coords.LogicalPosition   // Starting position
    To       coords.LogicalPosition   // Destination
    Path     []coords.LogicalPosition // Full path (for multi-step moves)
    StepCost int                      // AP cost per step
}

func NewMoveAction(actorID ecs.EntityID, from, to coords.LogicalPosition) *MoveAction {
    return &MoveAction{
        BaseAction: BaseAction{Actor: actorID},
        From:       from,
        To:         to,
        StepCost:   1, // Default: 1 AP per tile
    }
}

func (ma *MoveAction) Type() ActionType {
    return ActionTypeMove
}

func (ma *MoveAction) Description() string {
    return fmt.Sprintf("Move from (%d,%d) to (%d,%d)",
        ma.From.X, ma.From.Y, ma.To.X, ma.To.Y)
}

func (ma *MoveAction) Validate(manager *common.EntityManager) error {
    // Get movement system to validate
    moveSys := combat.NewMovementSystem(manager, common.GlobalPositionSystem)

    // Check if squad exists
    squadEntity := manager.FindEntityByID(ma.Actor)
    if squadEntity == nil {
        return fmt.Errorf("squad %d not found", ma.Actor)
    }

    // Check current position matches expected
    currentPos, err := moveSys.GetSquadPosition(ma.Actor)
    if err != nil {
        return fmt.Errorf("cannot get squad position: %w", err)
    }

    if currentPos != ma.From {
        return fmt.Errorf("squad not at expected position: at (%d,%d), expected (%d,%d)",
            currentPos.X, currentPos.Y, ma.From.X, ma.From.Y)
    }

    // Check destination is valid
    if !moveSys.CanMoveTo(ma.Actor, ma.To) {
        return fmt.Errorf("cannot move to (%d,%d): tile occupied or invalid",
            ma.To.X, ma.To.Y)
    }

    // Check action points (NEW: using TurnStateData instead of ActionStateData)
    turnState := combat.FindTurnState(ma.Actor, manager)
    if turnState == nil {
        return fmt.Errorf("no turn state for squad")
    }

    distance := ma.From.ChebyshevDistance(&ma.To)
    apCost := distance * ma.StepCost

    if turnState.ActionPoints < apCost {
        return fmt.Errorf("insufficient action points: need %d, have %d",
            apCost, turnState.ActionPoints)
    }

    return nil
}

func (ma *MoveAction) Execute(manager *common.EntityManager) error {
    // Delegate to CombatMovementSystem
    moveSys := combat.NewMovementSystem(manager, common.GlobalPositionSystem)

    err := moveSys.MoveSquad(ma.Actor, ma.To)
    if err != nil {
        return fmt.Errorf("move execution failed: %w", err)
    }

    return nil
}

func (ma *MoveAction) ActionPointCost() int {
    distance := ma.From.ChebyshevDistance(&ma.To)
    return distance * ma.StepCost
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

### AttackAction Implementation

```go
// tactical/actions/attack_action.go

package actions

import (
    "fmt"
    "game_main/common"
    "game_main/tactical/combat"

    "github.com/bytearena/ecs"
)

// AttackAction represents an attack between two squads
type AttackAction struct {
    BaseAction
    TargetID ecs.EntityID  // Squad being attacked
}

func NewAttackAction(attackerID, targetID ecs.EntityID) *AttackAction {
    return &AttackAction{
        BaseAction: BaseAction{Actor: attackerID},
        TargetID:   targetID,
    }
}

func (aa *AttackAction) Type() ActionType {
    return ActionTypeAttack
}

func (aa *AttackAction) Description() string {
    return fmt.Sprintf("Attack squad %d", aa.TargetID)
}

func (aa *AttackAction) Validate(manager *common.EntityManager) error {
    combatSys := combat.NewCombatActionSystem(manager)

    // Delegate validation to CombatActionSystem
    reason, canAttack := combatSys.CanSquadAttackWithReason(aa.Actor, aa.TargetID)
    if !canAttack {
        return fmt.Errorf(reason)
    }

    // Check action points
    turnState := combat.FindTurnState(aa.Actor, manager)
    if turnState == nil {
        return fmt.Errorf("no turn state for squad")
    }

    if turnState.HasActed {
        return fmt.Errorf("squad has already acted this turn")
    }

    return nil
}

func (aa *AttackAction) Execute(manager *common.EntityManager) error {
    combatSys := combat.NewCombatActionSystem(manager)

    // Delegate to CombatActionSystem
    result := combatSys.ExecuteAttackAction(aa.Actor, aa.TargetID)
    if !result.Success {
        return fmt.Errorf("attack failed: %s", result.ErrorReason)
    }

    return nil
}

func (aa *AttackAction) ActionPointCost() int {
    // Attacks consume the major action, not AP
    return 0
}

func (aa *AttackAction) Clone() Action {
    clone := *aa
    return &clone
}
```

### Action Queue Functions

```go
// tactical/actions/queue.go

package actions

import (
    "fmt"
    "game_main/common"

    "github.com/bytearena/ecs"
)

// QueueAction adds an action to a squad's action queue
// Validates before adding (fail-fast)
// Creates ActionQueueComponent if it doesn't exist
func QueueAction(manager *common.EntityManager, actorID ecs.EntityID, action Action) error {
    // Validate before queueing
    if err := action.Validate(manager); err != nil {
        return fmt.Errorf("action validation failed: %w", err)
    }

    // Get squad entity
    actor := manager.FindEntityByID(actorID)
    if actor == nil {
        return fmt.Errorf("actor entity not found: %d", actorID)
    }

    // Get or create action queue
    queueData := common.GetComponentType[*ActionQueueData](actor, ActionQueueComponent)
    if queueData == nil {
        queueData = &ActionQueueData{
            Actions:      make([]Action, 0),
            CurrentIndex: 0,
            Paused:       false,
        }
        actor.AddComponent(ActionQueueComponent, queueData)
        actor.AddTag(HasActionsQueuedTag)
    }

    // Add to queue
    queueData.Actions = append(queueData.Actions, action)

    return nil
}

// ProcessActionQueues executes one action per squad with queued actions
// Call this each game update/tick to progress action queues
func ProcessActionQueues(manager *common.EntityManager) {
    // Query all squads with queued actions
    for _, result := range manager.World.Query(HasActionsQueuedTag) {
        entity := result.Entity

        queueData := common.GetComponentType[*ActionQueueData](entity, ActionQueueComponent)
        if queueData == nil || queueData.Paused || len(queueData.Actions) == 0 {
            continue
        }

        // Get current action (always first in queue)
        currentAction := queueData.Actions[0]

        // Execute with state tracking
        err := executeActionWithTracking(manager, entity, currentAction)
        if err != nil {
            // Action failed - remove from queue
            removeCurrentAction(manager, entity.ID)
        }
    }
}

// executeActionWithTracking wraps execution with optional state tracking
func executeActionWithTracking(manager *common.EntityManager, entity *ecs.Entity, action Action) error {
    // Optional: Track execution state (for animations)
    // Most turn-based games don't need this

    // Re-validate (state may have changed since queuing)
    if err := action.Validate(manager); err != nil {
        return fmt.Errorf("action no longer valid: %w", err)
    }

    // Execute
    if err := action.Execute(manager); err != nil {
        return fmt.Errorf("action execution failed: %w", err)
    }

    // Success - remove from queue
    removeCurrentAction(manager, entity.ID)

    return nil
}

// removeCurrentAction removes the first action from queue
func removeCurrentAction(manager *common.EntityManager, actorID ecs.EntityID) {
    actor := manager.FindEntityByID(actorID)
    if actor == nil {
        return
    }

    queueData := common.GetComponentType[*ActionQueueData](actor, ActionQueueComponent)
    if queueData != nil && len(queueData.Actions) > 0 {
        // Remove first action
        queueData.Actions = queueData.Actions[1:]

        // If queue now empty, remove tag
        if len(queueData.Actions) == 0 {
            actor.RemoveTag(HasActionsQueuedTag)
        }
    }
}

// ClearActionQueue removes all queued actions
func ClearActionQueue(manager *common.EntityManager, actorID ecs.EntityID) {
    actor := manager.FindEntityByID(actorID)
    if actor == nil {
        return
    }

    queueData := common.GetComponentType[*ActionQueueData](actor, ActionQueueComponent)
    if queueData != nil {
        queueData.Actions = nil
        actor.RemoveTag(HasActionsQueuedTag)
    }
}

// GetQueuedActions returns copy of queued actions (for UI display)
func GetQueuedActions(manager *common.EntityManager, actorID ecs.EntityID) []Action {
    actor := manager.FindEntityByID(actorID)
    if actor == nil {
        return nil
    }

    queueData := common.GetComponentType[*ActionQueueData](actor, ActionQueueComponent)
    if queueData == nil {
        return nil
    }

    // Return copy to prevent external modification
    actionsCopy := make([]Action, len(queueData.Actions))
    copy(actionsCopy, queueData.Actions)
    return actionsCopy
}

// HasQueuedActions checks if squad has pending actions
func HasQueuedActions(manager *common.EntityManager, actorID ecs.EntityID) bool {
    actor := manager.FindEntityByID(actorID)
    if actor == nil {
        return false
    }

    return actor.HasTag(HasActionsQueuedTag)
}
```

---

## Command System

### Extend SquadCommand Interface

```go
// tactical/squadcommands/command.go

package squadcommands

import (
    "game_main/tactical/actions"
    "github.com/bytearena/ecs"
)

// SquadCommand represents an undoable tactical operation
// Commands wrap Actions to provide undo/redo capability
// Both player and AI use commands
type SquadCommand interface {
    // Validate checks if the command can be executed
    Validate() error

    // Execute performs the command's operation
    // May queue one or more actions
    // Captures state needed for undo
    Execute() error

    // Undo reverses the command's operation
    // Restores captured state
    Undo() error

    // Description returns human-readable description
    Description() string

    // Type returns command type for filtering/UI
    Type() CommandType

    // ActorID returns the squad performing this command
    ActorID() ecs.EntityID

    // Clone creates a copy for preview (optional)
    Clone() SquadCommand

    // GetActions returns the actions this command will execute
    // Used for preview and validation
    GetActions() []actions.Action
}

// CommandType categorizes commands
type CommandType int

const (
    CommandTypeMove CommandType = iota
    CommandTypeAttack
    CommandTypeAbility
    CommandTypeMoveAttack    // Composite: move then attack
    CommandTypeAITurn        // AI-planned action sequence
    CommandTypeWait
    CommandTypeFormation
    CommandTypeMerge
    CommandTypeDisband
    CommandTypeRename
)

func (ct CommandType) String() string {
    switch ct {
    case CommandTypeMove: return "Move"
    case CommandTypeAttack: return "Attack"
    case CommandTypeAbility: return "Ability"
    case CommandTypeMoveAttack: return "Move + Attack"
    case CommandTypeAITurn: return "AI Turn"
    case CommandTypeWait: return "Wait"
    case CommandTypeFormation: return "Formation"
    case CommandTypeMerge: return "Merge"
    case CommandTypeDisband: return "Disband"
    case CommandTypeRename: return "Rename"
    default: return "Unknown"
    }
}
```

### MoveCommand (Updated)

```go
// tactical/squadcommands/move_command.go

package squadcommands

import (
    "fmt"
    "game_main/common"
    "game_main/tactical/actions"
    "game_main/tactical/combat"
    "game_main/world/coords"

    "github.com/bytearena/ecs"
)

// MoveCommand wraps one or more MoveActions for undo/redo
// Can handle single-step or multi-step movement
type MoveCommand struct {
    entityManager *common.EntityManager
    squadID       ecs.EntityID
    moveActions   []*actions.MoveAction  // May be multiple for multi-turn moves

    // Undo state
    oldPosition    coords.LogicalPosition
    oldTurnState   combat.TurnStateData
    squadName      string
}

// NewMoveCommand creates a move command (single destination)
func NewMoveCommand(
    manager *common.EntityManager,
    squadID ecs.EntityID,
    destination coords.LogicalPosition,
) *MoveCommand {
    return &MoveCommand{
        entityManager: manager,
        squadID:       squadID,
        moveActions:   createMoveActionsForDestination(manager, squadID, destination),
    }
}

func (cmd *MoveCommand) Type() CommandType {
    return CommandTypeMove
}

func (cmd *MoveCommand) ActorID() ecs.EntityID {
    return cmd.squadID
}

func (cmd *MoveCommand) GetActions() []actions.Action {
    result := make([]actions.Action, len(cmd.moveActions))
    for i, ma := range cmd.moveActions {
        result[i] = ma
    }
    return result
}

func (cmd *MoveCommand) Validate() error {
    // Validate first action (others will be validated when queued)
    if len(cmd.moveActions) == 0 {
        return fmt.Errorf("no move actions")
    }

    return cmd.moveActions[0].Validate(cmd.entityManager)
}

func (cmd *MoveCommand) Execute() error {
    // Capture state for undo
    cmd.captureState()

    // Queue all move actions
    for _, moveAction := range cmd.moveActions {
        err := actions.QueueAction(cmd.entityManager, cmd.squadID, moveAction)
        if err != nil {
            return fmt.Errorf("failed to queue move action: %w", err)
        }
    }

    return nil
}

func (cmd *MoveCommand) Undo() error {
    // Clear any remaining queued actions
    actions.ClearActionQueue(cmd.entityManager, cmd.squadID)

    // Restore position
    squadEntity := cmd.entityManager.FindEntityByID(cmd.squadID)
    if squadEntity == nil {
        return fmt.Errorf("squad entity not found")
    }

    currentPos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
    if currentPos == nil {
        return fmt.Errorf("squad has no position")
    }

    // Move back to old position
    unitIDs := squads.GetUnitIDsInSquad(cmd.squadID, cmd.entityManager)
    err := cmd.entityManager.MoveSquadAndMembers(
        cmd.squadID,
        squadEntity,
        unitIDs,
        *currentPos,
        cmd.oldPosition,
    )
    if err != nil {
        return fmt.Errorf("failed to restore position: %w", err)
    }

    // Restore turn state
    cmd.restoreTurnState()

    return nil
}

func (cmd *MoveCommand) Description() string {
    if len(cmd.moveActions) == 0 {
        return "Move (no actions)"
    }

    if len(cmd.moveActions) == 1 {
        return fmt.Sprintf("%s moves to (%d,%d)",
            cmd.squadName,
            cmd.moveActions[0].To.X,
            cmd.moveActions[0].To.Y)
    }

    return fmt.Sprintf("%s moves %d steps to (%d,%d)",
        cmd.squadName,
        len(cmd.moveActions),
        cmd.moveActions[len(cmd.moveActions)-1].To.X,
        cmd.moveActions[len(cmd.moveActions)-1].To.Y)
}

func (cmd *MoveCommand) Clone() SquadCommand {
    clonedActions := make([]*actions.MoveAction, len(cmd.moveActions))
    for i, ma := range cmd.moveActions {
        clonedActions[i] = ma.Clone().(*actions.MoveAction)
    }

    return &MoveCommand{
        entityManager: cmd.entityManager,
        squadID:       cmd.squadID,
        moveActions:   clonedActions,
    }
}

func (cmd *MoveCommand) captureState() {
    squadEntity := cmd.entityManager.FindEntityByID(cmd.squadID)
    if squadEntity != nil {
        // Capture position
        posPtr := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
        if posPtr != nil {
            cmd.oldPosition = *posPtr
        }

        // Capture name
        squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
        if squadData != nil {
            cmd.squadName = squadData.Name
        }
    }

    // Capture turn state
    turnState := combat.FindTurnState(cmd.squadID, cmd.entityManager)
    if turnState != nil {
        cmd.oldTurnState = *turnState
    }
}

func (cmd *MoveCommand) restoreTurnState() {
    turnStateEntity := combat.FindTurnStateEntity(cmd.squadID, cmd.entityManager)
    if turnStateEntity != nil {
        turnState := common.GetComponentType[*combat.TurnStateData](turnStateEntity, combat.TurnStateComponent)
        if turnState != nil {
            *turnState = cmd.oldTurnState
        }
    }
}

// Helper: Create move actions for reaching a destination
// May create multiple actions if distance > movement range
func createMoveActionsForDestination(
    manager *common.EntityManager,
    squadID ecs.EntityID,
    destination coords.LogicalPosition,
) []*actions.MoveAction {
    // Get current position
    squadEntity := manager.FindEntityByID(squadID)
    if squadEntity == nil {
        return nil
    }

    currentPos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
    if currentPos == nil {
        return nil
    }

    // Get movement range
    turnState := combat.FindTurnState(squadID, manager)
    if turnState == nil {
        return nil
    }

    movementRange := turnState.ActionPoints

    // Calculate distance
    distance := currentPos.ChebyshevDistance(&destination)

    // If within range, single action
    if distance <= movementRange {
        return []*actions.MoveAction{
            actions.NewMoveAction(squadID, *currentPos, destination),
        }
    }

    // Multi-turn move - create path
    // TODO: Implement pathfinding for obstacles
    // For now, simple straight-line movement

    path := createStraightPath(*currentPos, destination, movementRange)

    moveActions := make([]*actions.MoveAction, 0)
    from := *currentPos

    for _, to := range path {
        moveActions = append(moveActions, actions.NewMoveAction(squadID, from, to))
        from = to
    }

    return moveActions
}

// Helper: Create straight-line path broken into movement-range chunks
func createStraightPath(from, to coords.LogicalPosition, maxStepSize int) []coords.LogicalPosition {
    path := []coords.LogicalPosition{}

    current := from
    for current != to {
        // Calculate next step toward destination
        dx := to.X - current.X
        dy := to.Y - current.Y

        // Clamp to maxStepSize
        if abs(dx) > maxStepSize {
            if dx > 0 {
                dx = maxStepSize
            } else {
                dx = -maxStepSize
            }
        }

        if abs(dy) > maxStepSize {
            if dy > 0 {
                dy = maxStepSize
            } else {
                dy = -maxStepSize
            }
        }

        next := coords.LogicalPosition{
            X: current.X + dx,
            Y: current.Y + dy,
        }

        path = append(path, next)
        current = next
    }

    return path
}

func abs(x int) int {
    if x < 0 {
        return -x
    }
    return x
}
```

### AttackCommand (Updated)

```go
// tactical/squadcommands/attack_command.go

package squadcommands

import (
    "fmt"
    "game_main/common"
    "game_main/tactical/actions"
    "game_main/tactical/combat"
    "game_main/tactical/squads"

    "github.com/bytearena/ecs"
)

// AttackCommand wraps AttackAction for undo/redo
type AttackCommand struct {
    entityManager *common.EntityManager
    combatSystem  *combat.CombatActionSystem
    attackAction  *actions.AttackAction

    // Undo state
    attackerName       string
    defenderName       string
    oldTurnState       combat.TurnStateData
    oldDefenderHealth  map[ecs.EntityID]int
    unitsKilled        []ecs.EntityID
    wasDestroyed       bool
    defenderPosition   coords.LogicalPosition
}

func NewAttackCommand(
    manager *common.EntityManager,
    combatSys *combat.CombatActionSystem,
    attackerID ecs.EntityID,
    defenderID ecs.EntityID,
) *AttackCommand {
    return &AttackCommand{
        entityManager: manager,
        combatSystem:  combatSys,
        attackAction:  actions.NewAttackAction(attackerID, defenderID),
    }
}

func (cmd *AttackCommand) Type() CommandType {
    return CommandTypeAttack
}

func (cmd *AttackCommand) ActorID() ecs.EntityID {
    return cmd.attackAction.ActorID()
}

func (cmd *AttackCommand) GetActions() []actions.Action {
    return []actions.Action{cmd.attackAction}
}

func (cmd *AttackCommand) Validate() error {
    return cmd.attackAction.Validate(cmd.entityManager)
}

func (cmd *AttackCommand) Execute() error {
    // Capture state for undo
    cmd.captureState()

    // Queue action (will execute immediately via ProcessActionQueues)
    err := actions.QueueAction(cmd.entityManager, cmd.attackAction.ActorID(), cmd.attackAction)
    if err != nil {
        return fmt.Errorf("failed to queue attack: %w", err)
    }

    // Process queue immediately (attacks are immediate, not multi-turn)
    actions.ProcessActionQueues(cmd.entityManager)

    // Store results for undo
    cmd.captureResults()

    return nil
}

func (cmd *AttackCommand) Undo() error {
    // Restore turn state
    cmd.restoreTurnState()

    // Restore defender health
    for unitID, oldHP := range cmd.oldDefenderHealth {
        attr := common.GetComponentTypeByID[*common.Attributes](cmd.entityManager, unitID, common.AttributeComponent)
        if attr != nil {
            attr.CurrentHealth = oldHP
        }
    }

    // Restore squad destroyed status
    if cmd.wasDestroyed {
        squads.UpdateSquadDestroyedStatus(cmd.attackAction.TargetID, cmd.entityManager)

        // Re-add to position system
        defenderSquad := cmd.entityManager.FindEntityByID(cmd.attackAction.TargetID)
        if defenderSquad != nil {
            squadData := common.GetComponentType[*squads.SquadData](defenderSquad, squads.SquadComponent)
            if squadData != nil && !squadData.IsDestroyed {
                common.GlobalPositionSystem.AddEntity(cmd.attackAction.TargetID, cmd.defenderPosition)
            }
        }
    }

    return nil
}

func (cmd *AttackCommand) Description() string {
    return fmt.Sprintf("%s attacks %s", cmd.attackerName, cmd.defenderName)
}

func (cmd *AttackCommand) Clone() SquadCommand {
    return &AttackCommand{
        entityManager: cmd.entityManager,
        combatSystem:  cmd.combatSystem,
        attackAction:  cmd.attackAction.Clone().(*actions.AttackAction),
    }
}

func (cmd *AttackCommand) captureState() {
    // Capture names
    cmd.attackerName = getSquadName(cmd.attackAction.ActorID(), cmd.entityManager)
    cmd.defenderName = getSquadName(cmd.attackAction.TargetID, cmd.entityManager)

    // Capture turn state
    turnState := combat.FindTurnState(cmd.attackAction.ActorID(), cmd.entityManager)
    if turnState != nil {
        cmd.oldTurnState = *turnState
    }

    // Capture defender health
    cmd.oldDefenderHealth = make(map[ecs.EntityID]int)
    unitIDs := squads.GetUnitIDsInSquad(cmd.attackAction.TargetID, cmd.entityManager)
    for _, unitID := range unitIDs {
        attr := common.GetComponentTypeByID[*common.Attributes](cmd.entityManager, unitID, common.AttributeComponent)
        if attr != nil {
            cmd.oldDefenderHealth[unitID] = attr.CurrentHealth
        }
    }

    // Capture defender position
    defenderSquad := cmd.entityManager.FindEntityByID(cmd.attackAction.TargetID)
    if defenderSquad != nil {
        posPtr := common.GetComponentType[*coords.LogicalPosition](defenderSquad, common.PositionComponent)
        if posPtr != nil {
            cmd.defenderPosition = *posPtr
        }
    }
}

func (cmd *AttackCommand) captureResults() {
    // Check if defender was destroyed
    cmd.wasDestroyed = squads.IsSquadDestroyed(cmd.attackAction.TargetID, cmd.entityManager)

    // Check for killed units
    cmd.unitsKilled = []ecs.EntityID{}
    for unitID, oldHP := range cmd.oldDefenderHealth {
        attr := common.GetComponentTypeByID[*common.Attributes](cmd.entityManager, unitID, common.AttributeComponent)
        if attr != nil && attr.CurrentHealth <= 0 && oldHP > 0 {
            cmd.unitsKilled = append(cmd.unitsKilled, unitID)
        }
    }
}

func (cmd *AttackCommand) restoreTurnState() {
    turnStateEntity := combat.FindTurnStateEntity(cmd.attackAction.ActorID(), cmd.entityManager)
    if turnStateEntity != nil {
        turnState := common.GetComponentType[*combat.TurnStateData](turnStateEntity, combat.TurnStateComponent)
        if turnState != nil {
            *turnState = cmd.oldTurnState
        }
    }
}

func getSquadName(squadID ecs.EntityID, manager *common.EntityManager) string {
    squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
    if squadData != nil {
        return squadData.Name
    }
    return "Unknown"
}
```

### MoveAttackCommand (Composite)

```go
// tactical/squadcommands/move_attack_command.go

package squadcommands

import (
    "fmt"
    "game_main/common"
    "game_main/tactical/actions"
    "game_main/tactical/combat"
    "game_main/world/coords"

    "github.com/bytearena/ecs"
)

// MoveAttackCommand combines movement and attack in one undoable command
// Useful for player "move then attack" and AI planning
type MoveAttackCommand struct {
    entityManager *common.EntityManager
    combatSystem  *combat.CombatActionSystem

    moveCommand   *MoveCommand
    attackCommand *AttackCommand

    squadID       ecs.EntityID
    squadName     string
}

func NewMoveAttackCommand(
    manager *common.EntityManager,
    combatSys *combat.CombatActionSystem,
    squadID ecs.EntityID,
    moveDestination coords.LogicalPosition,
    attackTargetID ecs.EntityID,
) *MoveAttackCommand {
    return &MoveAttackCommand{
        entityManager: manager,
        combatSystem:  combatSys,
        moveCommand:   NewMoveCommand(manager, squadID, moveDestination),
        attackCommand: NewAttackCommand(manager, combatSys, squadID, attackTargetID),
        squadID:       squadID,
    }
}

func (cmd *MoveAttackCommand) Type() CommandType {
    return CommandTypeMoveAttack
}

func (cmd *MoveAttackCommand) ActorID() ecs.EntityID {
    return cmd.squadID
}

func (cmd *MoveAttackCommand) GetActions() []actions.Action {
    // Combine actions from both commands
    moveActions := cmd.moveCommand.GetActions()
    attackActions := cmd.attackCommand.GetActions()

    combined := make([]actions.Action, 0, len(moveActions)+len(attackActions))
    combined = append(combined, moveActions...)
    combined = append(combined, attackActions...)

    return combined
}

func (cmd *MoveAttackCommand) Validate() error {
    // Validate move
    if err := cmd.moveCommand.Validate(); err != nil {
        return fmt.Errorf("move validation failed: %w", err)
    }

    // Validate attack (after hypothetical move)
    // This is tricky - we need to validate attack assuming move succeeds
    // For now, just validate attack from current position
    if err := cmd.attackCommand.Validate(); err != nil {
        return fmt.Errorf("attack validation failed: %w", err)
    }

    return nil
}

func (cmd *MoveAttackCommand) Execute() error {
    // Capture squad name
    squadData := common.GetComponentTypeByID[*squads.SquadData](cmd.entityManager, cmd.squadID, squads.SquadComponent)
    if squadData != nil {
        cmd.squadName = squadData.Name
    }

    // Execute move (queues actions)
    if err := cmd.moveCommand.Execute(); err != nil {
        return fmt.Errorf("move execution failed: %w", err)
    }

    // Queue attack action (will execute after move completes)
    attackAction := cmd.attackCommand.attackAction
    if err := actions.QueueAction(cmd.entityManager, cmd.squadID, attackAction); err != nil {
        return fmt.Errorf("failed to queue attack: %w", err)
    }

    return nil
}

func (cmd *MoveAttackCommand) Undo() error {
    // Undo in reverse order: attack first, then move
    if err := cmd.attackCommand.Undo(); err != nil {
        return fmt.Errorf("attack undo failed: %w", err)
    }

    if err := cmd.moveCommand.Undo(); err != nil {
        return fmt.Errorf("move undo failed: %w", err)
    }

    return nil
}

func (cmd *MoveAttackCommand) Description() string {
    return fmt.Sprintf("%s moves and attacks", cmd.squadName)
}

func (cmd *MoveAttackCommand) Clone() SquadCommand {
    return &MoveAttackCommand{
        entityManager: cmd.entityManager,
        combatSystem:  cmd.combatSystem,
        moveCommand:   cmd.moveCommand.Clone().(*MoveCommand),
        attackCommand: cmd.attackCommand.Clone().(*AttackCommand),
        squadID:       cmd.squadID,
    }
}
```

---

## Multi-Turn Movement

### How It Works

**Scenario:** Squad wants to move 10 tiles, but only has 3 movement points per turn

**Old System (boolean flags):**
- Can only move max 3 tiles per turn
- Cannot queue future movement
- Player must manually continue movement each turn

**New System (action queues):**
- MoveCommand creates 4 MoveActions: [Move 3 tiles, Move 3 tiles, Move 3 tiles, Move 1 tile]
- All 4 actions queued in ActionQueueComponent
- Each turn, ProcessActionQueues() executes one MoveAction
- Movement completes over 4 turns automatically

### Usage Example

```go
// Player clicks destination 10 tiles away
destination := coords.LogicalPosition{X: 10, Y: 0}

// Create move command (automatically splits into multi-turn if needed)
cmd := squadcommands.NewMoveCommand(manager, squadID, destination)

// Execute (queues all move actions)
result := commandExecutor.Execute(cmd)

// Each turn, TurnManager calls:
actions.ProcessActionQueues(manager)  // Executes one move action per turn

// After 4 turns, squad reaches destination
// Player can undo entire 10-tile move with one undo command
```

---

## AI Planning

### AIDecisionSystem with Action Queues

```go
// ai/decision_system.go

package ai

import (
    "game_main/common"
    "game_main/tactical/actions"
    "game_main/tactical/combat"
    "game_main/tactical/squadcommands"
    "game_main/tactical/squads"
    "game_main/world/coords"

    "github.com/bytearena/ecs"
)

type AIDecisionSystem struct {
    entityManager  *common.EntityManager
    combatSystem   *combat.CombatActionSystem
    movementSystem *combat.CombatMovementSystem
}

func NewAIDecisionSystem(
    manager *common.EntityManager,
    combatSys *combat.CombatActionSystem,
    moveSys *combat.CombatMovementSystem,
) *AIDecisionSystem {
    return &AIDecisionSystem{
        entityManager:  manager,
        combatSystem:   combatSys,
        movementSystem: moveSys,
    }
}

// PlanTurn creates a command for the AI squad's turn
// Returns a command that may queue multiple actions
func (ai *AIDecisionSystem) PlanTurn(squadID ecs.EntityID) squadcommands.SquadCommand {
    // 1. Find best target
    target := ai.findBestTarget(squadID)
    if target == 0 {
        // No targets - wait
        return squadcommands.NewWaitCommand(ai.entityManager, squadID)
    }

    // 2. Check if can attack from current position
    attackRange := ai.combatSystem.GetSquadAttackRange(squadID)
    distance := squads.GetSquadDistance(squadID, target, ai.entityManager)

    if distance >= 0 && distance <= attackRange {
        // Can attack now - create attack command
        return squadcommands.NewAttackCommand(
            ai.entityManager,
            ai.combatSystem,
            squadID,
            target,
        )
    }

    // 3. Need to move closer - plan movement
    currentPos, _ := combat.GetSquadMapPosition(squadID, ai.entityManager)
    targetPos, _ := combat.GetSquadMapPosition(target, ai.entityManager)

    // Find best attack position (within attack range of target)
    attackPos := ai.findBestAttackPosition(squadID, currentPos, targetPos, attackRange)

    // 4. Check if we can reach attack position this turn
    turnState := combat.FindTurnState(squadID, ai.entityManager)
    moveRange := turnState.ActionPoints

    distanceToAttackPos := currentPos.ChebyshevDistance(&attackPos)

    if distanceToAttackPos <= moveRange {
        // Can reach attack position and attack this turn
        // Create move+attack composite command
        return squadcommands.NewMoveAttackCommand(
            ai.entityManager,
            ai.combatSystem,
            squadID,
            attackPos,
            target,
        )
    }

    // 5. Can't reach and attack this turn - just move closer
    // This will queue multi-turn movement if destination is far
    return squadcommands.NewMoveCommand(
        ai.entityManager,
        squadID,
        attackPos,
    )
}

// findBestAttackPosition finds optimal tile to attack target from
func (ai *AIDecisionSystem) findBestAttackPosition(
    squadID ecs.EntityID,
    currentPos coords.LogicalPosition,
    targetPos coords.LogicalPosition,
    attackRange int,
) coords.LogicalPosition {
    // Find all tiles within attack range of target
    candidates := []coords.LogicalPosition{}

    for x := targetPos.X - attackRange; x <= targetPos.X+attackRange; x++ {
        for y := targetPos.Y - attackRange; y <= targetPos.Y+attackRange; y++ {
            testPos := coords.LogicalPosition{X: x, Y: y}

            // Check if within attack range
            if testPos.ChebyshevDistance(&targetPos) > attackRange {
                continue
            }

            // Check if tile is valid (not occupied, in bounds)
            if ai.movementSystem.CanMoveTo(squadID, testPos) {
                candidates = append(candidates, testPos)
            }
        }
    }

    if len(candidates) == 0 {
        // No valid attack positions - just move toward target
        return ai.moveToward(currentPos, targetPos, 1)
    }

    // Find closest candidate to current position
    bestPos := candidates[0]
    bestDistance := currentPos.ChebyshevDistance(&bestPos)

    for _, candidate := range candidates[1:] {
        dist := currentPos.ChebyshevDistance(&candidate)
        if dist < bestDistance {
            bestDistance = dist
            bestPos = candidate
        }
    }

    return bestPos
}

// moveToward returns position one step closer to target
func (ai *AIDecisionSystem) moveToward(from, to coords.LogicalPosition, steps int) coords.LogicalPosition {
    dx := to.X - from.X
    dy := to.Y - from.Y

    // Normalize to step size
    if abs(dx) > steps {
        if dx > 0 {
            dx = steps
        } else {
            dx = -steps
        }
    }

    if abs(dy) > steps {
        if dy > 0 {
            dy = steps
        } else {
            dy = -steps
        }
    }

    return coords.LogicalPosition{
        X: from.X + dx,
        Y: from.Y + dy,
    }
}

// findBestTarget selects highest priority enemy squad
func (ai *AIDecisionSystem) findBestTarget(squadID ecs.EntityID) ecs.EntityID {
    myFaction := combat.GetFactionOwner(squadID, ai.entityManager)
    if myFaction == 0 {
        return 0
    }

    var bestTarget ecs.EntityID
    bestScore := -1.0

    for _, result := range ai.entityManager.World.Query(squads.SquadTag) {
        enemySquadID := result.Entity.ID
        enemyFaction := combat.GetFactionOwner(enemySquadID, ai.entityManager)

        // Skip allies and self
        if enemyFaction == myFaction || enemySquadID == squadID {
            continue
        }

        // Skip destroyed squads
        if squads.IsSquadDestroyed(enemySquadID, ai.entityManager) {
            continue
        }

        // Calculate threat score
        score := ai.calculateThreatScore(squadID, enemySquadID)

        if score > bestScore {
            bestScore = score
            bestTarget = enemySquadID
        }
    }

    return bestTarget
}

// calculateThreatScore rates target priority
func (ai *AIDecisionSystem) calculateThreatScore(attackerID, targetID ecs.EntityID) float64 {
    score := 100.0

    // Prefer closer targets
    distance := squads.GetSquadDistance(attackerID, targetID, ai.entityManager)
    if distance < 0 {
        return 0.0
    }
    score -= float64(distance) * 5.0

    // Prefer damaged targets
    avgHP := squads.GetSquadAverageHP(targetID, ai.entityManager)
    score += (100.0 - float64(avgHP)) * 0.5

    return score
}

func abs(x int) int {
    if x < 0 {
        return -x
    }
    return x
}
```

### AI Turn Example

```go
// Combat mode AI turn processing

func (tm *TurnManager) ProcessAITurn(squadID ecs.EntityID) {
    // AI plans turn (may create composite command with multiple actions)
    cmd := tm.aiSystem.PlanTurn(squadID)

    // Execute command (queues actions)
    result := tm.commandExecutor.Execute(cmd)

    if !result.Success {
        log.Printf("AI command failed: %v", result.Error)
        // Fallback to wait
        waitCmd := squadcommands.NewWaitCommand(tm.manager, squadID)
        tm.commandExecutor.Execute(waitCmd)
    }

    // Actions will execute over multiple turns via ProcessActionQueues()
}
```

---

## Implementation Phases

### Phase 1: Component Refactoring (3-4 hours)

**Rename combat.ActionStateComponent:**
- Rename `ActionStateData` → `TurnStateData`
- Add `ActionPoints` field (replace MovementRemaining)
- Update all references in combat package
- Update TurnManager, CombatMovementSystem, etc.

**Create ActionQueueComponent:**
- Create `tactical/actions/components.go`
- Define `ActionQueueData` and `ExecutionStateData`
- Register components in init()

**Files:**
- `tactical/combat/combatcomponents.go` - Rename
- `tactical/actions/components.go` - NEW
- `tactical/combat/TurnManager.go` - Update references
- `tactical/combat/CombatMovementSystem.go` - Update references
- `tactical/squadcommands/move_squad_command.go` - Update references

**Testing:**
```bash
go test ./tactical/combat/...
go test ./tactical/squadcommands/...
```

---

### Phase 2: Action System (6-8 hours)

**Create Action Interface and Base Types:**
- `tactical/actions/action_interface.go` - Action interface, ActionType enum
- `tactical/actions/move_action.go` - MoveAction implementation
- `tactical/actions/attack_action.go` - AttackAction implementation
- `tactical/actions/queue.go` - Queue management functions

**Testing:**
```bash
go test ./tactical/actions/ -v
```

---

### Phase 3: Update Commands (4-5 hours)

**Extend SquadCommand Interface:**
- Add `GetActions()` method
- Add `Clone()` method
- Update existing commands

**Update MoveCommand:**
- Support multi-turn movement
- Queue multiple MoveActions if needed

**Update Existing Commands:**
- Add GetActions() and Clone() to all commands
- Update Type() to return CommandType

**Create New Commands:**
- `AttackCommand` - Wraps AttackAction
- `MoveAttackCommand` - Composite command

**Files:**
- `tactical/squadcommands/command.go` - Update interface
- `tactical/squadcommands/move_command.go` - Rewrite for action queues
- `tactical/squadcommands/attack_command.go` - NEW
- `tactical/squadcommands/move_attack_command.go` - NEW

**Testing:**
```bash
go test ./tactical/squadcommands/ -v
```

---

### Phase 4: AI Integration (5-6 hours)

**Create AI Decision System:**
- `ai/decision_system.go` - AIDecisionSystem with PlanTurn()
- `ai/targeting.go` - Target selection logic
- `ai/positioning.go` - Attack position finding

**Integrate with TurnManager:**
- Add AIDecisionSystem field
- Implement ProcessAITurn()
- Call ProcessActionQueues() each update

**Files:**
- `ai/decision_system.go` - NEW
- `ai/targeting.go` - NEW
- `ai/positioning.go` - NEW
- `tactical/combat/TurnManager.go` - Add AI integration

**Testing:**
```bash
go test ./ai/ -v
```

---

### Phase 5: TurnManager Integration (3-4 hours)

**Update TurnManager:**
- Add CommandExecutor field
- Add AIDecisionSystem field
- Implement ProcessActionQueues() call in update loop
- Support both immediate and queued execution

**Queue Processing:**
- Call ProcessActionQueues() each game update
- Handle action completion
- Advance turns when queue empty

**Files:**
- `tactical/combat/TurnManager.go` - Major updates

---

### Phase 6: GUI Integration (4-5 hours)

**Update Combat Mode:**
- Replace direct system calls with commands
- Show queued actions in UI
- Add undo button
- Implement action preview (optional)

**Files:**
- `gui/guicombat/combatmode.go` - Update input handling
- `gui/guicombat/combat_ui.go` - Add action queue display
- `gui/guicombat/action_preview.go` - NEW (optional)

---

### Phase 7: Testing & Polish (4-5 hours)

**Unit Tests:**
- Test all action types
- Test command undo/redo
- Test multi-turn movement
- Test AI decision-making

**Integration Tests:**
- Test queue processing
- Test move+attack sequences
- Test AI vs player combat

**Manual Testing:**
- Gameplay testing
- Undo/redo testing
- Performance testing

---

## Testing Strategy

### Unit Tests

```go
// tactical/actions/move_action_test.go

func TestMoveAction_Validate(t *testing.T) {
    manager := setupTestManager()
    squad := createTestSquad(manager, coords.LogicalPosition{X: 0, Y: 0})

    // Create move action
    moveAction := actions.NewMoveAction(squad,
        coords.LogicalPosition{X: 0, Y: 0},
        coords.LogicalPosition{X: 2, Y: 2})

    // Should validate successfully
    err := moveAction.Validate(manager)
    if err != nil {
        t.Errorf("Validation failed: %v", err)
    }
}

func TestMoveAction_Execute(t *testing.T) {
    manager := setupTestManager()
    squad := createTestSquad(manager, coords.LogicalPosition{X: 0, Y: 0})

    moveAction := actions.NewMoveAction(squad,
        coords.LogicalPosition{X: 0, Y: 0},
        coords.LogicalPosition{X: 1, Y: 1})

    // Execute move
    err := moveAction.Execute(manager)
    if err != nil {
        t.Fatalf("Execute failed: %v", err)
    }

    // Verify position updated
    newPos, _ := combat.GetSquadMapPosition(squad, manager)
    if newPos.X != 1 || newPos.Y != 1 {
        t.Errorf("Position not updated correctly: got (%d,%d), want (1,1)",
            newPos.X, newPos.Y)
    }
}

// tactical/actions/queue_test.go

func TestActionQueue_MultiTurnMovement(t *testing.T) {
    manager := setupTestManager()
    squad := createTestSquad(manager, coords.LogicalPosition{X: 0, Y: 0})

    // Queue 3 moves
    for i := 1; i <= 3; i++ {
        moveAction := actions.NewMoveAction(squad,
            coords.LogicalPosition{X: i-1, Y: 0},
            coords.LogicalPosition{X: i, Y: 0})

        err := actions.QueueAction(manager, squad, moveAction)
        if err != nil {
            t.Fatalf("Failed to queue action %d: %v", i, err)
        }
    }

    // Verify queue has 3 actions
    queuedActions := actions.GetQueuedActions(manager, squad)
    if len(queuedActions) != 3 {
        t.Errorf("Expected 3 queued actions, got %d", len(queuedActions))
    }

    // Process queue 3 times
    for turn := 1; turn <= 3; turn++ {
        actions.ProcessActionQueues(manager)

        // Verify position after each turn
        pos, _ := combat.GetSquadMapPosition(squad, manager)
        if pos.X != turn {
            t.Errorf("Turn %d: position X should be %d, got %d", turn, turn, pos.X)
        }
    }

    // Queue should be empty
    if actions.HasQueuedActions(manager, squad) {
        t.Error("Queue should be empty after processing all actions")
    }
}

// tactical/squadcommands/move_command_test.go

func TestMoveCommand_MultiTurnMovement(t *testing.T) {
    manager := setupTestManager()
    squad := createTestSquad(manager, coords.LogicalPosition{X: 0, Y: 0})

    // Set movement range to 3
    setSquadMovementRange(squad, 3, manager)

    // Create command to move 10 tiles (requires 4 turns)
    cmd := squadcommands.NewMoveCommand(manager, squad, coords.LogicalPosition{X: 10, Y: 0})

    // Verify command creates 4 actions
    actions := cmd.GetActions()
    if len(actions) != 4 {
        t.Errorf("Expected 4 actions for 10-tile move with 3 AP, got %d", len(actions))
    }

    // Execute command (queues all actions)
    err := cmd.Execute()
    if err != nil {
        t.Fatalf("Execute failed: %v", err)
    }

    // Simulate 4 turns
    for turn := 0; turn < 4; turn++ {
        actions.ProcessActionQueues(manager)
    }

    // Verify final position
    finalPos, _ := combat.GetSquadMapPosition(squad, manager)
    if finalPos.X != 10 {
        t.Errorf("Final position should be (10,0), got (%d,%d)", finalPos.X, finalPos.Y)
    }

    // Test undo
    err = cmd.Undo()
    if err != nil {
        t.Fatalf("Undo failed: %v", err)
    }

    // Verify position restored
    restoredPos, _ := combat.GetSquadMapPosition(squad, manager)
    if restoredPos.X != 0 {
        t.Errorf("Position should be restored to (0,0), got (%d,%d)",
            restoredPos.X, restoredPos.Y)
    }
}

// ai/decision_system_test.go

func TestAIDecisionSystem_MoveAndAttack(t *testing.T) {
    manager := setupTestManager()
    combatSys := combat.NewCombatActionSystem(manager)
    moveSys := combat.NewMovementSystem(manager, common.GlobalPositionSystem)
    aiSys := ai.NewAIDecisionSystem(manager, combatSys, moveSys)

    // Create AI squad far from enemy (distance = 5)
    aiSquad := createTestSquad(manager, coords.LogicalPosition{X: 0, Y: 0})
    enemySquad := createTestSquad(manager, coords.LogicalPosition{X: 5, Y: 0})

    setSquadFaction(aiSquad, 1, manager)
    setSquadFaction(enemySquad, 2, manager)

    // Set AI squad movement range to 3 and attack range to 1
    setSquadMovementRange(aiSquad, 3, manager)
    setSquadAttackRange(aiSquad, 1, manager)

    // AI plans turn
    cmd := aiSys.PlanTurn(aiSquad)

    // Should create move+attack command
    if cmd.Type() != squadcommands.CommandTypeMoveAttack {
        t.Errorf("Expected MoveAttack command, got %s", cmd.Type())
    }

    // Verify command has move and attack actions
    actions := cmd.GetActions()
    if len(actions) < 2 {
        t.Errorf("Expected at least 2 actions (move + attack), got %d", len(actions))
    }

    // Verify last action is attack
    lastAction := actions[len(actions)-1]
    if lastAction.Type() != actions.ActionTypeAttack {
        t.Error("Last action should be attack")
    }
}
```

---

## Migration Checklist

### Pre-Implementation
- [ ] Backup codebase
- [ ] Create feature branch: `git checkout -b feature/unified-action-queue-system`

### Phase 1: Component Refactoring
- [ ] Rename `ActionStateData` → `TurnStateData` in combat/combatcomponents.go
- [ ] Add `ActionPoints` field to TurnStateData
- [ ] Create `tactical/actions/components.go`
- [ ] Define ActionQueueData and ExecutionStateData
- [ ] Update all combat/* references
- [ ] Run tests: `go test ./tactical/combat/...`
- [ ] Commit: `git commit -m "Refactor: Rename ActionStateData to TurnStateData, add ActionQueueComponent"`

### Phase 2: Action System
- [ ] Create `tactical/actions/action_interface.go`
- [ ] Create `tactical/actions/move_action.go`
- [ ] Create `tactical/actions/attack_action.go`
- [ ] Create `tactical/actions/queue.go`
- [ ] Write action tests
- [ ] Run tests: `go test ./tactical/actions/...`
- [ ] Commit: `git commit -m "Add Action system with Move and Attack actions"`

### Phase 3: Update Commands
- [ ] Update SquadCommand interface with GetActions() and Clone()
- [ ] Rewrite MoveCommand for action queues
- [ ] Create AttackCommand
- [ ] Create MoveAttackCommand
- [ ] Update all existing commands
- [ ] Write command tests
- [ ] Run tests: `go test ./tactical/squadcommands/...`
- [ ] Commit: `git commit -m "Update commands to use action queues"`

### Phase 4: AI Integration
- [ ] Create `ai/decision_system.go`
- [ ] Implement PlanTurn() with multi-action support
- [ ] Create targeting and positioning logic
- [ ] Write AI tests
- [ ] Run tests: `go test ./ai/...`
- [ ] Commit: `git commit -m "Add AI decision system with action planning"`

### Phase 5: TurnManager Integration
- [ ] Add CommandExecutor to TurnManager
- [ ] Add AIDecisionSystem to TurnManager
- [ ] Implement ProcessActionQueues() call
- [ ] Update ProcessAITurn() and ProcessPlayerCommand()
- [ ] Test integration
- [ ] Commit: `git commit -m "Integrate action queues with TurnManager"`

### Phase 6: GUI Integration
- [ ] Update combat mode to use commands
- [ ] Add action queue display to UI
- [ ] Add undo button
- [ ] Test manual gameplay
- [ ] Commit: `git commit -m "Integrate action queues with combat GUI"`

### Phase 7: Testing & Polish
- [ ] Run full test suite: `go test ./...`
- [ ] Manual gameplay testing
- [ ] Undo/redo testing
- [ ] Performance testing
- [ ] Fix bugs
- [ ] Update DOCUMENTATION.md
- [ ] Commit: `git commit -m "Final testing and documentation"`

### Merge
- [ ] Code review
- [ ] Final testing
- [ ] Merge to main

---

## Conclusion

This implementation plan provides:

**Action Queues (from action_system_design.md):**
- ✅ Multi-turn movement
- ✅ AI can plan move → attack sequences
- ✅ Actions queued and processed sequentially
- ✅ Extensible action types

**Undo/Redo (from squadcommands):**
- ✅ Full undo/redo for all commands
- ✅ Command history tracking
- ✅ State capture and restoration

**Integration:**
- ✅ Actions delegate to existing combat systems
- ✅ Commands wrap actions for undo capability
- ✅ Both player and AI use same command interface
- ✅ Minimal refactoring of working code

**Total Estimated Time:** 29-37 hours

**Key Benefits:**
- Multi-turn actions (long-distance movement)
- AI planning (move+attack sequences)
- Undo/redo support (tactical necessity)
- Unified player/AI interface
- Extensible for new action types

**Result:** A production-ready unified action system that combines the queueing capability from action_system_design.md with the undo/redo capability from the existing squadcommands pattern.
