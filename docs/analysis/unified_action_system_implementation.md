# Unified Action System Implementation (REVISED v2)

**Created:** 2025-12-25
**Revised:** 2025-12-26 (v2)
**Status:** Implementation Plan
**Purpose:** Add command queue system to existing combat, squads, and squadcommands packages

---

## Executive Summary

This document provides an **accurate** implementation plan for adding a **command queue system** to the existing TinkerRogue codebase. After thorough code review and architecture discussion, we're using the existing `SquadCommand` interface for queueing - **no separate Action interface needed**.

### What Already Exists ✅

**tactical/squadcommands/**
- `SquadCommand` interface - Validate(), Execute(), Undo(), Description()
- `MoveSquadCommand` - Full movement with undo/redo (tactical/squadcommands/move_squad_command.go)
- `CommandExecutor` - Undo/redo stacks, history tracking (tactical/squadcommands/command_executor.go)
- Multiple commands: RenameSquadCommand, DisbandSquadCommand, ChangeLeaderCommand, etc.

**tactical/combat/**
- `ActionStateData` - Tracks HasMoved, HasActed, MovementRemaining (tactical/combat/combatcomponents.go:39-44)
- `TurnStateData` - Tracks CurrentRound, TurnOrder, CurrentTurnIndex (tactical/combat/combatcomponents.go:32-37)
- `CombatActionSystem` - ExecuteAttackAction(), CanSquadAttackWithReason() (tactical/combat/combatactionsystem.go)
- `CombatMovementSystem` - MoveSquad(), CanMoveTo(), GetValidMovementTiles() (tactical/combat/CombatMovementSystem.go)
- `FactionData` - Faction ownership and player control (tactical/combat/combatcomponents.go:22-30)

**tactical/squads/**
- `ExecuteSquadAttack` - Complete attack execution (tactical/squads/squadcombat.go:19)
- `GetSquadDistance` - Chebyshev distance calculation (tactical/squads/squadqueries.go:229)
- `IsSquadDestroyed` - Squad status checking (tactical/squads/squadqueries.go:123)
- `CanUnitAttack` - Attack validation (tactical/squads/squadcombat.go:139)

### What Needs to Be Added 🆕

1. **Command Queue System** - Enable multi-turn command sequences
2. **MoveStepCommand** - Single-step movement command (for multi-turn paths)
3. **AttackCommand** - Undoable attack wrapper (currently attacks aren't undoable)
4. **AI Decision System** - AI planning with command queues
5. **Integration** - Connect command queues to existing TurnManager

### Key Architecture Decision

**Single Interface - No Duplication:**
- `SquadCommand` interface does EVERYTHING (queueable AND undoable)
- Queue holds `[]SquadCommand` directly
- Multi-turn movement = queue multiple small `MoveStepCommand`s
- CommandExecutor tracks top-level commands (player actions)
- Queue execution bypasses CommandExecutor (micro-steps, no history)

**Flow Example:**
```
Player clicks 10 tiles away
  ↓
CommandExecutor.Execute(MoveSquadCommand) → adds to undo history
  ↓
MoveSquadCommand.Execute() → creates 4 MoveStepCommands, queues them
  ↓
Turn 1: ProcessCommandQueues() → executes MoveStepCommand #1 (no history)
Turn 2: ProcessCommandQueues() → executes MoveStepCommand #2 (no history)
Turn 3: ProcessCommandQueues() → executes MoveStepCommand #3 (no history)
Turn 4: ProcessCommandQueues() → executes MoveStepCommand #4 (no history)
  ↓
Player hits undo → MoveSquadCommand.Undo() → clears queue, restores position
```

---

## Table of Contents

1. [Current System Analysis](#current-system-analysis)
2. [Command Queue Design](#command-queue-design)
3. [MoveStepCommand](#movestepcommand)
4. [AttackCommand Addition](#attackcommand-addition)
5. [Multi-Turn MoveSquadCommand](#multi-turn-movesquadcommand)
6. [AI Integration](#ai-integration)
7. [Implementation Phases](#implementation-phases)
8. [Migration Guide](#migration-guide)

---



## Command Queue Design

### New Components



---


---

## AttackCommand Addition

**Problem:** Attacks currently can't be undone. We need an AttackCommand to match MoveSquadCommand.

```go
// tactical/squadcommands/attack_command.go (NEW FILE)

package squadcommands

import (
    "fmt"
    "game_main/common"
    "game_main/tactical/combat"
    "game_main/tactical/squads"
    "game_main/world/coords"
    "github.com/bytearena/ecs"
)

// AttackCommand wraps attack execution for undo/redo capability
type AttackCommand struct {
    entityManager *common.EntityManager
    combatSystem  *combat.CombatActionSystem
    attackerID    ecs.EntityID
    defenderID    ecs.EntityID

    // Undo state
    attackerName       string
    defenderName       string
    oldActionState     combat.ActionStateData
    oldDefenderHealth  map[ecs.EntityID]int  // Unit HP before attack
    wasDestroyed       bool
    defenderPosition   coords.LogicalPosition
}

func NewAttackCommand(
    manager *common.EntityManager,
    combatSys *combat.CombatActionSystem,
    attackerID, defenderID ecs.EntityID,
) *AttackCommand {
    return &AttackCommand{
        entityManager: manager,
        combatSystem:  combatSys,
        attackerID:    attackerID,
        defenderID:    defenderID,
    }
}

func (cmd *AttackCommand) Validate() error {
    reason, canAttack := cmd.combatSystem.CanSquadAttackWithReason(cmd.attackerID, cmd.defenderID)
    if !canAttack {
        return fmt.Errorf(reason)
    }

    return nil
}

func (cmd *AttackCommand) Execute() error {
    // Capture state for undo
    cmd.captureState()

    // Delegate to CombatActionSystem (REUSE EXISTING)
    result := cmd.combatSystem.ExecuteAttackAction(cmd.attackerID, cmd.defenderID)
    if !result.Success {
        return fmt.Errorf("attack failed: %s", result.ErrorReason)
    }

    // Capture results for undo
    cmd.captureResults()

    return nil
}

func (cmd *AttackCommand) Undo() error {
    // Restore action state
    cmd.restoreActionState()

    // Restore defender health
    for unitID, oldHP := range cmd.oldDefenderHealth {
        entity := cmd.entityManager.FindEntityByID(unitID)
        if entity == nil {
            continue
        }

        attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
        if attr != nil {
            attr.CurrentHealth = oldHP
        }
    }

    // Restore squad destroyed status
    if cmd.wasDestroyed {
        squads.UpdateSquadDestroyedStatus(cmd.defenderID, cmd.entityManager)

        // Re-add to position system if no longer destroyed
        defenderSquad := cmd.entityManager.FindEntityByID(cmd.defenderID)
        if defenderSquad != nil {
            squadData := common.GetComponentType[*squads.SquadData](defenderSquad, squads.SquadComponent)
            if squadData != nil && !squadData.IsDestroyed {
                common.GlobalPositionSystem.AddEntity(cmd.defenderID, cmd.defenderPosition)
            }
        }
    }

    return nil
}

func (cmd *AttackCommand) Description() string {
    return fmt.Sprintf("%s attacks %s", cmd.attackerName, cmd.defenderName)
}

func (cmd *AttackCommand) captureState() {
    // Get names
    cmd.attackerName = getSquadName(cmd.attackerID, cmd.entityManager)
    cmd.defenderName = getSquadName(cmd.defenderID, cmd.entityManager)

    // Capture action state
    actionState := combat.FindActionState(cmd.attackerID, cmd.entityManager)
    if actionState != nil {
        cmd.oldActionState = *actionState
    }

    // Capture defender health
    cmd.oldDefenderHealth = make(map[ecs.EntityID]int)
    unitIDs := squads.GetUnitIDsInSquad(cmd.defenderID, cmd.entityManager)
    for _, unitID := range unitIDs {
        entity := cmd.entityManager.FindEntityByID(unitID)
        if entity == nil {
            continue
        }

        attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
        if attr != nil {
            cmd.oldDefenderHealth[unitID] = attr.CurrentHealth
        }
    }

    // Capture defender position
    defenderSquad := cmd.entityManager.FindEntityByID(cmd.defenderID)
    if defenderSquad != nil {
        posPtr := common.GetComponentType[*coords.LogicalPosition](defenderSquad, common.PositionComponent)
        if posPtr != nil {
            cmd.defenderPosition = *posPtr
        }
    }
}

func (cmd *AttackCommand) captureResults() {
    cmd.wasDestroyed = squads.IsSquadDestroyed(cmd.defenderID, cmd.entityManager)
}

func (cmd *AttackCommand) restoreActionState() {
    actionStateEntity := combat.FindActionStateEntity(cmd.attackerID, cmd.entityManager)
    if actionStateEntity != nil {
        actionState := common.GetComponentType[*combat.ActionStateData](actionStateEntity, combat.ActionStateComponent)
        if actionState != nil {
            *actionState = cmd.oldActionState
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

---

## Multi-Turn MoveSquadCommand

### Update MoveSquadCommand for Multi-Turn Movement

```go
// tactical/squadcommands/move_squad_command.go (MODIFY EXISTING)

// Update Execute() to queue step commands for multi-turn movement
func (cmd *MoveSquadCommand) Execute() error {
    // Capture state for undo (EXISTING CODE)
    squadEntity, err := getSquadOrError(cmd.squadID, cmd.entityManager)
    if err != nil {
        return err
    }

    squadData, err := getSquadDataOrError(squadEntity)
    if err == nil {
        cmd.squadName = squadData.Name
    }

    posPtr := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
    if posPtr == nil {
        return fmt.Errorf("squad has no position component")
    }
    cmd.oldPosition = *posPtr

    actionStateEntity := combat.FindActionStateEntity(cmd.squadID, cmd.entityManager)
    if actionStateEntity != nil {
        actionState := common.GetComponentType[*combat.ActionStateData](actionStateEntity, combat.ActionStateComponent)
        if actionState != nil {
            cmd.oldMovementRemaining = actionState.MovementRemaining
            cmd.oldHasMoved = actionState.HasMoved
        }
    }

    // NEW: Check if multi-turn movement needed
    distance := cmd.oldPosition.ChebyshevDistance(&cmd.newPosition)

    if distance <= cmd.oldMovementRemaining {
        // Single-turn movement - execute immediately (EXISTING BEHAVIOR)
        err = cmd.movementSystem.MoveSquad(cmd.squadID, cmd.newPosition)
        if err != nil {
            return fmt.Errorf("movement system failed: %w", err)
        }
    } else {
        // Multi-turn movement - queue step commands
        stepCommands := createMoveStepCommands(
            cmd.entityManager,
            cmd.movementSystem,
            cmd.squadID,
            cmd.oldPosition,
            cmd.newPosition,
            cmd.oldMovementRemaining,
        )

        for _, stepCmd := range stepCommands {
            err := QueueCommand(cmd.entityManager, cmd.squadID, stepCmd)
            if err != nil {
                return fmt.Errorf("failed to queue move step: %w", err)
            }
        }
    }

    return nil
}

// Update Undo to clear queued commands
func (cmd *MoveSquadCommand) Undo() error {
    // NEW: Clear any remaining queued commands
    ClearCommandQueue(cmd.entityManager, cmd.squadID)

    // EXISTING: Restore position and state
    squadEntity, err := getSquadOrError(cmd.squadID, cmd.entityManager)
    if err != nil {
        return err
    }

    posPtr := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
    if posPtr == nil {
        return fmt.Errorf("squad has no position component")
    }
    currentPos := *posPtr

    unitIDs := squads.GetUnitIDsInSquad(cmd.squadID, cmd.entityManager)
    if err := cmd.entityManager.MoveSquadAndMembers(
        cmd.squadID,
        squadEntity,
        unitIDs,
        currentPos,
        cmd.oldPosition,
    ); err != nil {
        return fmt.Errorf("failed to undo squad move: %w", err)
    }

    actionStateEntity := combat.FindActionStateEntity(cmd.squadID, cmd.entityManager)
    if actionStateEntity != nil {
        actionState := common.GetComponentType[*combat.ActionStateData](actionStateEntity, combat.ActionStateComponent)
        if actionState != nil {
            actionState.MovementRemaining = cmd.oldMovementRemaining
            actionState.HasMoved = cmd.oldHasMoved
        }
    }

    return nil
}

// Helper: Create step commands for multi-turn movement
func createMoveStepCommands(
    manager *common.EntityManager,
    moveSys *combat.CombatMovementSystem,
    squadID ecs.EntityID,
    start, end coords.LogicalPosition,
    movementPerTurn int,
) []SquadCommand {
    path := createStraightPath(start, end, movementPerTurn)

    stepCommands := make([]SquadCommand, len(path))
    for i, destination := range path {
        stepCommands[i] = NewMoveStepCommand(manager, moveSys, squadID, destination)
    }

    return stepCommands
}

func createStraightPath(from, to coords.LogicalPosition, maxStepSize int) []coords.LogicalPosition {
    path := []coords.LogicalPosition{}
    current := from

    for current != to {
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

        next := coords.LogicalPosition{X: current.X + dx, Y: current.Y + dy}
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

---

## AI Integration

### AI Decision System

```go
// tactical/ai/decision_system.go (NEW FILE)

package ai

import (
    "game_main/common"
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
// Returns SquadCommand directly (no Action layer)
func (ai *AIDecisionSystem) PlanTurn(squadID ecs.EntityID) squadcommands.SquadCommand {
    // Find best target
    target := ai.findBestTarget(squadID)
    if target == 0 {
        // No targets - skip turn
        return nil // TODO: Implement WaitCommand if needed
    }

    // Check if can attack from current position
    attackRange := ai.combatSystem.GetSquadAttackRange(squadID)
    distance := squads.GetSquadDistance(squadID, target, ai.entityManager)

    if distance >= 0 && distance <= attackRange {
        // Can attack now
        return squadcommands.NewAttackCommand(ai.entityManager, ai.combatSystem, squadID, target)
    }

    // Need to move closer
    currentPos, _ := ai.movementSystem.GetSquadPosition(squadID)
    targetPos, _ := ai.movementSystem.GetSquadPosition(target)

    // Find best attack position
    attackPos := ai.findBestAttackPosition(squadID, currentPos, targetPos, attackRange)

    // Create move command (will automatically queue steps if multi-turn)
    return squadcommands.NewMoveSquadCommand(ai.entityManager, ai.movementSystem, squadID, attackPos)
}

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

        if enemyFaction == myFaction || enemySquadID == squadID {
            continue
        }

        if squads.IsSquadDestroyed(enemySquadID, ai.entityManager) {
            continue
        }

        score := ai.calculateThreatScore(squadID, enemySquadID)
        if score > bestScore {
            bestScore = score
            bestTarget = enemySquadID
        }
    }

    return bestTarget
}

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

func (ai *AIDecisionSystem) findBestAttackPosition(
    squadID ecs.EntityID,
    currentPos, targetPos coords.LogicalPosition,
    attackRange int,
) coords.LogicalPosition {
    candidates := []coords.LogicalPosition{}

    // Find all tiles within attack range of target
    for x := targetPos.X - attackRange; x <= targetPos.X+attackRange; x++ {
        for y := targetPos.Y - attackRange; y <= targetPos.Y+attackRange; y++ {
            testPos := coords.LogicalPosition{X: x, Y: y}

            if testPos.ChebyshevDistance(&targetPos) > attackRange {
                continue
            }

            if ai.movementSystem.CanMoveTo(squadID, testPos) {
                candidates = append(candidates, testPos)
            }
        }
    }

    if len(candidates) == 0 {
        // No valid positions - move toward target
        return moveToward(currentPos, targetPos, 1)
    }

    // Find closest candidate
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

func moveToward(from, to coords.LogicalPosition, steps int) coords.LogicalPosition {
    dx := to.X - from.X
    dy := to.Y - from.Y

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

    return coords.LogicalPosition{X: from.X + dx, Y: from.Y + dy}
}

func abs(x int) int {
    if x < 0 {
        return -x
    }
    return x
}
```

---

## Implementation Phases

### Phase 1: Command Queue Foundation (3-4 hours)

**Create in tactical/squadcommands/:**
```
queue_components.go      # CommandQueueData, HasCommandsQueuedTag
command_queue.go         # QueueCommand(), ProcessCommandQueues()
move_step_command.go     # MoveStepCommand (single-step movement)
```

**Testing:**
```bash
go test ./tactical/squadcommands/ -v
```

---

### Phase 2: Attack Command (2-3 hours)

**Create:**
- `tactical/squadcommands/attack_command.go` - Undoable attack wrapper

**Update:**
- Add combat.FindActionState() helper if missing
- Add combat.FindActionStateEntity() helper if missing

**Testing:**
```bash
go test ./tactical/squadcommands/ -v
```

---

### Phase 3: Update MoveSquadCommand (3-4 hours)

**Modify:**
- `tactical/squadcommands/move_squad_command.go`:
  - Update Execute() to detect multi-turn movement
  - Queue MoveStepCommands when needed
  - Update Undo() to clear queue

**Add helpers:**
- createMoveStepCommands()
- createStraightPath()

**Testing:**
- Test single-step movement (should work as before)
- Test multi-turn movement (new behavior)
- Test undo during multi-turn movement

---

### Phase 4: AI Decision System (4-5 hours)

**Create:**
```
tactical/ai/
├── decision_system.go   # PlanTurn(), target selection
└── helpers.go          # findBestAttackPosition(), etc.
```

**Testing:**
```bash
go test ./tactical/ai/ -v
```

---

### Phase 5: TurnManager Integration (3-4 hours)

**Update:**
- `tactical/combat/TurnManager.go`:
  - Add ProcessCommandQueues() call in update loop
  - Integrate AIDecisionSystem
  - Update AI turn processing

**Files affected:**
- tactical/combat/TurnManager.go

---

### Phase 6: GUI Integration (3-4 hours)

**Update:**
- Combat mode to use AttackCommand
- Show queued commands in UI (optional)
- Add undo button functionality

**Files affected:**
- gui/guicombat/combatmode.go

---

## Migration Guide

### Required Helper Functions

Add these to `tactical/combat/combatqueries.go`:

```go
// FindActionState returns ActionStateData for a squad
func FindActionState(squadID ecs.EntityID, manager *common.EntityManager) *ActionStateData {
    entity := FindActionStateEntity(squadID, manager)
    if entity == nil {
        return nil
    }

    return common.GetComponentType[*ActionStateData](entity, ActionStateComponent)
}

// FindActionStateEntity returns the entity with ActionStateData for a squad
func FindActionStateEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
    for _, result := range manager.World.Query(ActionStateTag) {
        entity := result.Entity
        actionState := common.GetComponentType[*ActionStateData](entity, ActionStateComponent)

        if actionState != nil && actionState.SquadID == squadID {
            return entity
        }
    }

    return nil
}
```

Add to `tactical/squads/squadqueries.go`:

```go
// GetSquadAverageHP returns average HP percentage of units in squad
func GetSquadAverageHP(squadID ecs.EntityID, manager *common.EntityManager) float64 {
    unitIDs := GetUnitIDsInSquad(squadID, manager)
    if len(unitIDs) == 0 {
        return 0.0
    }

    totalHPPercent := 0.0
    count := 0

    for _, unitID := range unitIDs {
        entity := manager.FindEntityByID(unitID)
        if entity == nil {
            continue
        }

        attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
        if attr != nil && attr.MaxHealth > 0 {
            hpPercent := float64(attr.CurrentHealth) / float64(attr.MaxHealth) * 100.0
            totalHPPercent += hpPercent
            count++
        }
    }

    if count == 0 {
        return 0.0
    }

    return totalHPPercent / float64(count)
}
```

Add to `tactical/combat/combatqueries.go`:

```go
// GetFactionOwner returns the faction ID that owns a squad
func GetFactionOwner(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
    squadEntity := manager.FindEntityByID(squadID)
    if squadEntity == nil {
        return 0
    }

    factionData := common.GetComponentType[*CombatFactionData](squadEntity, CombatFactionComponent)
    if factionData == nil {
        return 0
    }

    return factionData.FactionID
}
```

---

## Summary

### What This Adds

1. **Command Queue System** - Multi-turn command execution
2. **MoveStepCommand** - Single-step movement (for queuing)
3. **AttackCommand** - Undoable attacks (fills gap)
4. **AI Planning** - Multi-turn AI sequences
5. **Multi-Turn Movement** - Automatic step command creation

### What This Preserves

1. **Single Interface** - SquadCommand does everything (no duplication)
2. **Undo/Redo** - CommandExecutor unchanged
3. **Combat Systems** - CombatActionSystem, CombatMovementSystem unchanged
4. **ActionStateData** - Turn economy tracking unchanged

### Architecture Benefits

- ✅ **No duplication** - Single SquadCommand interface
- ✅ **Clear separation** - CommandExecutor (player history) vs Queue (micro-steps)
- ✅ **Composable** - Commands can queue sub-commands
- ✅ **Reusable** - Existing systems unchanged

### Total Estimated Time

- Phase 1: 3-4 hours (Command queue)
- Phase 2: 2-3 hours (AttackCommand)
- Phase 3: 3-4 hours (Update MoveSquadCommand)
- Phase 4: 4-5 hours (AI system)
- Phase 5: 3-4 hours (TurnManager)
- Phase 6: 3-4 hours (GUI)

**Total: 18-24 hours**

### Files to Create

```
tactical/squadcommands/queue_components.go
tactical/squadcommands/command_queue.go
tactical/squadcommands/move_step_command.go
tactical/squadcommands/attack_command.go
tactical/ai/decision_system.go
tactical/ai/helpers.go
```

### Files to Modify

```
tactical/squadcommands/move_squad_command.go
tactical/combat/TurnManager.go
tactical/combat/combatqueries.go (add helpers)
tactical/squads/squadqueries.go (add GetSquadAverageHP)
gui/guicombat/combatmode.go
```

---

**Result:** A production-ready command queue system using the existing SquadCommand interface - no duplication, clean separation of concerns, preserves all undo/redo functionality.
