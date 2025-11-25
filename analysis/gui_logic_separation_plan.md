# GUI Logic Separation Plan

**Date:** 2025-11-24
**Purpose:** Analyze and separate game logic from user interface logic in the TinkerRogue GUI system

---

## Executive Summary

The TinkerRogue GUI architecture shows a mixed approach to separating concerns. While there are some good patterns in place (UIMode abstraction, GUIQueries service, component-based updates), **game logic and UI logic are still heavily intertwined**, particularly in:

1. **Action Handlers** executing game state mutations directly
2. **UI Modes** containing business logic for squad building, combat actions, and roster management
3. **Grid Managers** mixing UI state with ECS mutations
4. **BattleMapState/OverworldState** storing computed game state that should live in ECS

This document provides a **concrete plan** to separate these concerns using **Command Pattern**, **Service Layer**, and **Event-Driven Architecture**.

---

## Current Architecture Analysis

### What's Working Well ✓

#### 1. Mode Management System
- **UIMode Interface** provides clean abstraction for different screens
- **UIModeManager** handles mode lifecycle (Initialize, Enter, Exit, Update, Render)
- **GameModeCoordinator** separates Overworld and BattleMap contexts
- **BaseMode** reduces boilerplate with shared infrastructure

#### 2. Query Abstraction
- **GUIQueries** centralizes read-only ECS queries
- Provides value objects (`SquadInfo`, `FactionInfo`) instead of exposing raw ECS
- Eliminates query duplication across modes

#### 3. UI Components
- **SquadListComponent**, **DetailPanelComponent** manage widget updates
- Separate presentation logic from mode logic
- Use callbacks for selection events

#### 4. Widget Factories
- **PanelBuilders**, **ButtonFactory** encapsulate ebitenui widget creation
- Consistent styling and layout patterns

### Critical Issues ⚠️

#### Issue 1: Action Handlers Execute Game Logic Directly

**File:** `gui/guicombat/combat_action_handler.go`

**Problem:**
```go
func (cah *CombatActionHandler) executeAttack() {
    // UI handler directly creates combat systems and executes attacks
    combatSys := combat.NewCombatActionSystem(cah.entityManager)
    err := combatSys.ExecuteAttackAction(selectedSquad, selectedTarget)

    // UI handler updates unit positions
    cah.updateUnitPositions(squadID, newPos)

    // UI handler checks game state
    if squads.IsSquadDestroyed(selectedTarget, cah.entityManager) {
        // ...
    }
}
```

**Why This Is Bad:**
- UI code directly mutates game state
- Tight coupling between presentation and business logic
- No separation of concerns for testing
- Difficult to implement undo/redo, replays, or network sync
- Business rules scattered across UI handlers

**Impact:**
- Cannot test combat logic without UI
- Cannot run headless simulations
- Cannot implement AI that uses same logic without UI dependencies

---

#### Issue 2: Squad Builder Contains Business Logic

**File:** `gui/guisquads/squadbuilder.go`

**Problem:**
```go
func (sbm *SquadBuilderMode) placeRosterUnitInCell(row, col int, rosterEntry *squads.UnitRosterEntry) {
    // UI mode creates temporary squads
    if sbm.currentSquadID == 0 {
        sbm.createTemporarySquad()
    }

    // UI mode looks up unit templates
    var unitTemplate *squads.UnitTemplate
    for i := range squads.Units {
        if squads.Units[i].Name == rosterEntry.TemplateName {
            unitTemplate = &squads.Units[i]
            break
        }
    }

    // UI mode gets roster and allocates units
    roster := squads.GetPlayerRoster(sbm.Context.PlayerData.PlayerEntityID, sbm.Context.ECSManager)
    unitEntityID := roster.GetUnitEntityForTemplate(rosterEntry.TemplateName)

    // UI mode marks units as in squad
    roster.MarkUnitInSquad(unitEntityID, sbm.currentSquadID)
}
```

**Why This Is Bad:**
- Squad creation logic embedded in UI mode
- Roster allocation logic in presentation layer
- Template lookup in UI code
- Business rules for squad validity scattered across UI

**Impact:**
- Cannot create squads programmatically (e.g., for AI, loading save games, procedural generation)
- Squad validation logic duplicated or missing
- Difficult to test squad creation rules

---

#### Issue 3: Grid Manager Mutates ECS Directly

**File:** `gui/guisquads/squad_builder_grid_manager.go`

**Problem:**
```go
func (gem *GridEditorManager) PlaceRosterUnitInCell(...) error {
    // Grid manager (UI component) calls game logic directly
    err := squads.AddUnitToSquad(squadID, gem.entityManager, *unitTemplate, row, col)

    // Grid manager updates widget state AND ECS state in same method
    cell.unitID = unitID
    cell.button.Text().Label = cellText
}
```

**Why This Is Bad:**
- UI component has ECS mutation responsibilities
- Mixing widget updates with game state mutations
- No clear boundary between presentation and domain logic

---

#### Issue 4: UI State Stores Computed Game Data

**File:** `gui/core/contextstate.go`

**Problem:**
```go
type BattleMapState struct {
    ValidMoveTiles []coords.LogicalPosition // Cached from MovementSystem
}
```

**Why This Is Bad:**
- Movement tiles should be computed on-demand from ECS
- Caching creates synchronization issues
- If ECS state changes (e.g., squad destroyed), cached tiles become stale
- Violates single source of truth principle

---

## Recommended Architecture

### Overview: Three-Layer Separation

```
┌─────────────────────────────────────────────────────────────┐
│                    PRESENTATION LAYER                        │
│  (GUI Modes, Widgets, Renderers, Input Handlers)           │
│                                                              │
│  - Displays information to user                             │
│  - Captures user input                                      │
│  - Translates input to Commands                             │
│  - Subscribes to Events for updates                         │
└─────────────────────────────────────────────────────────────┘
                           ↓ Commands
                           ↑ Events
┌─────────────────────────────────────────────────────────────┐
│                     SERVICE LAYER                            │
│  (Game Services, Command Handlers, Event Bus)               │
│                                                              │
│  - Validates user actions                                   │
│  - Coordinates ECS systems                                  │
│  - Enforces business rules                                  │
│  - Publishes events for state changes                       │
└─────────────────────────────────────────────────────────────┘
                           ↓ Mutations
                           ↑ Queries
┌─────────────────────────────────────────────────────────────┐
│                      DOMAIN LAYER                            │
│  (ECS Components, Systems, Queries)                         │
│                                                              │
│  - Pure game state (ECS components)                         │
│  - Game logic (ECS systems)                                 │
│  - Query functions (read-only)                              │
│  - NO UI dependencies                                       │
└─────────────────────────────────────────────────────────────┘
```

---

## Pattern 1: Command Pattern for User Actions

### Concept

All user actions that modify game state become **Commands**. Commands are:
1. **Created by UI** based on user input
2. **Validated and executed by Services**
3. **Never executed directly by UI**

### Example: Combat Attack Command

**Before (Current - BAD):**
```go
// In CombatActionHandler
func (cah *CombatActionHandler) executeAttack() {
    // UI directly executes attack
    combatSys := combat.NewCombatActionSystem(cah.entityManager)
    err := combatSys.ExecuteAttackAction(selectedSquad, selectedTarget)
}
```

**After (Proposed - GOOD):**

**Step 1:** Define Command
```go
// File: services/commands/combat_commands.go
package commands

import "github.com/bytearena/ecs"

type AttackSquadCommand struct {
    AttackerID ecs.EntityID
    TargetID   ecs.EntityID
}

type MoveSquadCommand struct {
    SquadID  ecs.EntityID
    Position coords.LogicalPosition
}

type EndTurnCommand struct {
    FactionID ecs.EntityID
}
```

**Step 2:** Create Service Handler
```go
// File: services/combat_service.go
package services

type CombatService struct {
    entityManager  *common.EntityManager
    eventBus       *events.EventBus
    combatSystem   *combat.CombatActionSystem
    movementSystem *combat.MovementSystem
    turnManager    *combat.TurnManager
}

func NewCombatService(em *common.EntityManager, bus *events.EventBus) *CombatService {
    return &CombatService{
        entityManager:  em,
        eventBus:       bus,
        combatSystem:   combat.NewCombatActionSystem(em),
        movementSystem: combat.NewMovementSystem(em, common.GlobalPositionSystem),
        turnManager:    combat.NewTurnManager(em),
    }
}

// ExecuteAttack handles attack command with validation and events
func (cs *CombatService) ExecuteAttack(cmd *commands.AttackSquadCommand) error {
    // 1. Validate command
    reason, canAttack := cs.combatSystem.CanSquadAttackWithReason(cmd.AttackerID, cmd.TargetID)
    if !canAttack {
        return fmt.Errorf("cannot attack: %s", reason)
    }

    // 2. Execute game logic
    err := cs.combatSystem.ExecuteAttackAction(cmd.AttackerID, cmd.TargetID)
    if err != nil {
        return err
    }

    // 3. Publish events for UI updates
    cs.eventBus.Publish(&events.AttackExecutedEvent{
        AttackerID: cmd.AttackerID,
        TargetID:   cmd.TargetID,
    })

    // Check if target destroyed
    if squads.IsSquadDestroyed(cmd.TargetID, cs.entityManager) {
        cs.eventBus.Publish(&events.SquadDestroyedEvent{
            SquadID: cmd.TargetID,
        })
    }

    return nil
}

// MoveSquad handles movement command
func (cs *CombatService) MoveSquad(cmd *commands.MoveSquadCommand) error {
    // 1. Validate (movement system handles this)
    err := cs.movementSystem.MoveSquad(cmd.SquadID, cmd.Position)
    if err != nil {
        return err
    }

    // 2. Update all units in squad
    cs.updateUnitPositions(cmd.SquadID, cmd.Position)

    // 3. Publish event
    cs.eventBus.Publish(&events.SquadMovedEvent{
        SquadID:  cmd.SquadID,
        Position: cmd.Position,
    })

    return nil
}

// EndTurn handles turn end command
func (cs *CombatService) EndTurn(cmd *commands.EndTurnCommand) error {
    // 1. Execute turn manager logic
    err := cs.turnManager.EndTurn()
    if err != nil {
        return err
    }

    // 2. Get new turn info
    newFactionID := cs.turnManager.GetCurrentFaction()
    round := cs.turnManager.GetCurrentRound()

    // 3. Publish event
    cs.eventBus.Publish(&events.TurnChangedEvent{
        NewFactionID: newFactionID,
        Round:        round,
    })

    return nil
}
```

**Step 3:** UI Uses Service (No Direct ECS Access)
```go
// File: gui/guicombat/combat_action_handler.go (REFACTORED)
type CombatActionHandler struct {
    combatService  *services.CombatService
    battleMapState *core.BattleMapState
    logManager     *CombatLogManager
    combatLogArea  *widget.TextArea
}

func (cah *CombatActionHandler) executeAttack() {
    // UI creates command from UI state
    cmd := &commands.AttackSquadCommand{
        AttackerID: cah.battleMapState.SelectedSquadID,
        TargetID:   cah.battleMapState.SelectedTargetID,
    }

    // Service validates and executes (UI just waits for result)
    err := cah.combatService.ExecuteAttack(cmd)
    if err != nil {
        // Events will update UI, or we show error
        cah.addLog(fmt.Sprintf("Attack failed: %v", err))
        return
    }

    // Reset UI state
    cah.battleMapState.InAttackMode = false
}
```

**Benefits:**
- ✓ UI cannot execute invalid actions (service validates first)
- ✓ Combat logic testable without UI
- ✓ Can record/replay commands for undo or network play
- ✓ Single place for business rules
- ✓ Events allow multiple UI components to react

---

## Pattern 2: Event Bus for State Change Notifications

### Concept

When game state changes, **Services publish Events** that UI components subscribe to. This decouples "what changed" from "how UI updates".

### Example: Squad Destroyed Event

**Event Definition:**
```go
// File: services/events/combat_events.go
package events

type AttackExecutedEvent struct {
    AttackerID ecs.EntityID
    TargetID   ecs.EntityID
}

type SquadDestroyedEvent struct {
    SquadID ecs.EntityID
}

type SquadMovedEvent struct {
    SquadID  ecs.EntityID
    Position coords.LogicalPosition
}

type TurnChangedEvent struct {
    NewFactionID ecs.EntityID
    Round        int
}
```

**Event Bus Implementation:**
```go
// File: services/events/event_bus.go
package events

type Event interface{}

type EventHandler func(event Event)

type EventBus struct {
    handlers map[string][]EventHandler
}

func NewEventBus() *EventBus {
    return &EventBus{
        handlers: make(map[string][]EventHandler),
    }
}

// Subscribe registers a handler for an event type
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
    eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Publish sends an event to all subscribers
func (eb *EventBus) Publish(event Event) {
    eventType := fmt.Sprintf("%T", event)
    for _, handler := range eb.handlers[eventType] {
        handler(event)
    }
}
```

**UI Subscribes to Events:**
```go
// File: gui/guicombat/combatmode.go (REFACTORED)
func (cm *CombatMode) Initialize(ctx *core.UIContext) error {
    cm.InitializeBase(ctx)

    // Subscribe to combat events
    ctx.EventBus.Subscribe("*events.AttackExecutedEvent", func(event events.Event) {
        e := event.(*events.AttackExecutedEvent)
        attackerName := cm.Queries.GetSquadName(e.AttackerID)
        targetName := cm.Queries.GetSquadName(e.TargetID)
        cm.logManager.UpdateTextArea(cm.combatLogArea,
            fmt.Sprintf("%s attacked %s!", attackerName, targetName))
    })

    ctx.EventBus.Subscribe("*events.SquadDestroyedEvent", func(event events.Event) {
        e := event.(*events.SquadDestroyedEvent)
        squadName := cm.Queries.GetSquadName(e.SquadID)
        cm.logManager.UpdateTextArea(cm.combatLogArea,
            fmt.Sprintf("%s was destroyed!", squadName))
    })

    ctx.EventBus.Subscribe("*events.TurnChangedEvent", func(event events.Event) {
        e := event.(*events.TurnChangedEvent)
        cm.turnOrderComponent.Refresh()
        cm.factionInfoComponent.ShowFaction(e.NewFactionID)
        cm.squadListComponent.Refresh()
    })

    return nil
}
```

**Benefits:**
- ✓ Multiple UI components can react to same event
- ✓ Service doesn't know about UI widgets
- ✓ Easy to add logging, analytics, network sync
- ✓ Decouples notification from update logic

---

## Pattern 3: Service Layer for Squad Building

### Problem Recap

Squad building logic is currently in `squadbuilder.go` UI mode:
- Creates temporary squads
- Looks up unit templates
- Allocates roster units
- Validates capacity

### Solution: SquadBuilderService

**Service Definition:**
```go
// File: services/squad_builder_service.go
package services

type SquadBuilderService struct {
    entityManager *common.EntityManager
    eventBus      *events.EventBus
}

// CreateSquad creates a new squad with validation
func (sbs *SquadBuilderService) CreateSquad(playerID ecs.EntityID, squadName string) (ecs.EntityID, error) {
    // Business logic: validate squad name
    if squadName == "" {
        return 0, fmt.Errorf("squad name cannot be empty")
    }

    // Create squad entity
    squadID := squads.CreateEmptySquad(sbs.entityManager, squadName)

    // Publish event
    sbs.eventBus.Publish(&events.SquadCreatedEvent{
        SquadID:   squadID,
        SquadName: squadName,
    })

    return squadID, nil
}

// AddUnitToSquad adds a roster unit to a squad with validation
func (sbs *SquadBuilderService) AddUnitToSquad(cmd *commands.AddUnitToSquadCommand) error {
    // 1. Get roster
    roster := squads.GetPlayerRoster(cmd.PlayerID, sbs.entityManager)
    if roster == nil {
        return fmt.Errorf("player roster not found")
    }

    // 2. Get unit template
    unitTemplate := squads.GetUnitTemplateByName(cmd.TemplateName)
    if unitTemplate == nil {
        return fmt.Errorf("unit template not found: %s", cmd.TemplateName)
    }

    // 3. Check roster availability
    unitEntityID := roster.GetUnitEntityForTemplate(cmd.TemplateName)
    if unitEntityID == 0 {
        return fmt.Errorf("no available units of type %s", cmd.TemplateName)
    }

    // 4. Check capacity
    currentCapacity := squads.GetSquadUsedCapacity(cmd.SquadID, sbs.entityManager)
    totalCapacity := squads.GetSquadTotalCapacity(cmd.SquadID, sbs.entityManager)
    unitCost := unitTemplate.Attributes.GetCapacityCost()

    if currentCapacity + unitCost > float64(totalCapacity) {
        return fmt.Errorf("insufficient capacity: %.1f/%.1f (need %.1f)",
            currentCapacity, float64(totalCapacity), unitCost)
    }

    // 5. Add unit to squad
    err := squads.AddUnitToSquad(cmd.SquadID, sbs.entityManager, *unitTemplate, cmd.GridRow, cmd.GridCol)
    if err != nil {
        return err
    }

    // 6. Mark unit as allocated
    err = roster.MarkUnitInSquad(unitEntityID, cmd.SquadID)
    if err != nil {
        // Rollback squad addition if roster update fails
        // TODO: Implement rollback mechanism
        return err
    }

    // 7. Publish event
    sbs.eventBus.Publish(&events.UnitAddedToSquadEvent{
        SquadID:    cmd.SquadID,
        UnitID:     unitEntityID,
        GridRow:    cmd.GridRow,
        GridCol:    cmd.GridCol,
    })

    return nil
}

// RemoveUnitFromSquad removes a unit and returns it to roster
func (sbs *SquadBuilderService) RemoveUnitFromSquad(cmd *commands.RemoveUnitFromSquadCommand) error {
    // 1. Remove from squad (squad system handles this)
    err := squads.RemoveUnitFromSquad(cmd.SquadID, cmd.UnitID, sbs.entityManager)
    if err != nil {
        return err
    }

    // 2. Return to roster
    roster := squads.GetPlayerRoster(cmd.PlayerID, sbs.entityManager)
    if roster != nil {
        err = roster.MarkUnitAvailable(cmd.UnitID)
        if err != nil {
            return err
        }
    }

    // 3. Publish event
    sbs.eventBus.Publish(&events.UnitRemovedFromSquadEvent{
        SquadID: cmd.SquadID,
        UnitID:  cmd.UnitID,
    })

    return nil
}

// FinalizeSquad validates and completes squad creation
func (sbs *SquadBuilderService) FinalizeSquad(cmd *commands.FinalizeSquadCommand) error {
    // 1. Validate has units
    unitIDs := squads.GetUnitIDsInSquad(cmd.SquadID, sbs.entityManager)
    if len(unitIDs) == 0 {
        return fmt.Errorf("cannot finalize empty squad")
    }

    // 2. Validate has leader
    if cmd.LeaderID == 0 {
        return fmt.Errorf("squad must have a designated leader")
    }

    // 3. Assign leader component
    leaderEntity := common.FindEntityByIDWithTag(sbs.entityManager, cmd.LeaderID, squads.SquadMemberTag)
    if leaderEntity == nil {
        return fmt.Errorf("leader unit not found")
    }
    leaderEntity.AddComponent(squads.LeaderComponent, &squads.LeaderData{})

    // 4. Update squad name if changed
    squadEntity := squads.GetSquadEntity(cmd.SquadID, sbs.entityManager)
    if squadEntity != nil {
        squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
        squadData.Name = cmd.SquadName
    }

    // 5. Publish event
    sbs.eventBus.Publish(&events.SquadFinalizedEvent{
        SquadID:   cmd.SquadID,
        SquadName: cmd.SquadName,
    })

    return nil
}
```

**UI Uses Service:**
```go
// File: gui/guisquads/squadbuilder.go (REFACTORED)
func (sbm *SquadBuilderMode) onCellClicked(row, col int) {
    // UI just creates command and sends to service
    if sbm.selectedRosterEntry == nil {
        return
    }

    // Ensure squad exists
    if sbm.currentSquadID == 0 {
        squadID, err := sbm.squadBuilderService.CreateSquad(
            sbm.Context.PlayerData.PlayerEntityID,
            sbm.currentSquadName)
        if err != nil {
            fmt.Printf("Failed to create squad: %v\n", err)
            return
        }
        sbm.currentSquadID = squadID
    }

    // Create command
    cmd := &commands.AddUnitToSquadCommand{
        PlayerID:     sbm.Context.PlayerData.PlayerEntityID,
        SquadID:      sbm.currentSquadID,
        TemplateName: sbm.selectedRosterEntry.TemplateName,
        GridRow:      row,
        GridCol:      col,
    }

    // Service validates and executes
    err := sbm.squadBuilderService.AddUnitToSquad(cmd)
    if err != nil {
        fmt.Printf("Failed to add unit: %v\n", err)
        return
    }

    // UI updates happen via event subscriptions
}
```

**Benefits:**
- ✓ All squad creation logic in one place
- ✓ Easy to test squad validation rules
- ✓ Can create squads from code (AI, save loading, etc.)
- ✓ Centralized capacity and roster management

---

## Pattern 4: Computed Properties On-Demand

### Problem Recap

BattleMapState caches computed data:
```go
type BattleMapState struct {
    ValidMoveTiles []coords.LogicalPosition // Cached from MovementSystem
}
```

### Solution: Compute On-Demand via Service

**Remove cached state:**
```go
// File: gui/core/contextstate.go (REFACTORED)
type BattleMapState struct {
    // UI Selection State ONLY
    SelectedSquadID  ecs.EntityID
    SelectedTargetID ecs.EntityID

    // UI Mode Flags ONLY
    InAttackMode bool
    InMoveMode   bool

    // NO COMPUTED DATA - query on demand instead
}
```

**Query movement tiles when needed:**
```go
// File: services/combat_service.go
func (cs *CombatService) GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition {
    return cs.movementSystem.GetValidMovementTiles(squadID)
}

// File: gui/guicombat/combatmode.go (usage)
func (cm *CombatMode) Render(screen *ebiten.Image) {
    if battleState.InMoveMode && battleState.SelectedSquadID != 0 {
        // Query on-demand instead of using cached state
        validTiles := cm.combatService.GetValidMovementTiles(battleState.SelectedSquadID)
        cm.movementRenderer.Render(screen, playerPos, validTiles)
    }
}
```

**Benefits:**
- ✓ No stale data
- ✓ Single source of truth (ECS)
- ✓ Simpler state management

---

## Implementation Roadmap

### Phase 1: Foundation (1-2 days)

**Goal:** Set up service layer infrastructure

1. Create `services/` package structure:
   ```
   services/
   ├── commands/
   │   ├── combat_commands.go
   │   ├── squad_commands.go
   │   └── movement_commands.go
   ├── events/
   │   ├── event_bus.go
   │   ├── combat_events.go
   │   └── squad_events.go
   ├── combat_service.go
   ├── squad_builder_service.go
   └── movement_service.go
   ```

2. Implement EventBus (simple map-based version)
3. Add EventBus to UIContext
4. Write unit tests for EventBus

### Phase 2: Combat Service Migration (2-3 days)

**Goal:** Move combat logic from UI to service

1. Create CombatService with command handlers:
   - `ExecuteAttack(cmd)`
   - `MoveSquad(cmd)`
   - `EndTurn(cmd)`

2. Define events:
   - `AttackExecutedEvent`
   - `SquadDestroyedEvent`
   - `SquadMovedEvent`
   - `TurnChangedEvent`

3. Refactor CombatMode to use service:
   - Replace direct `combatSystem` calls with `combatService` calls
   - Subscribe to events for log updates
   - Remove direct ECS access from action handler

4. Test combat flow end-to-end

### Phase 3: Squad Builder Service Migration (2-3 days)

**Goal:** Move squad creation logic from UI to service

1. Create SquadBuilderService with command handlers:
   - `CreateSquad(cmd)`
   - `AddUnitToSquad(cmd)`
   - `RemoveUnitFromSquad(cmd)`
   - `SetSquadLeader(cmd)`
   - `FinalizeSquad(cmd)`

2. Define events:
   - `SquadCreatedEvent`
   - `UnitAddedToSquadEvent`
   - `UnitRemovedFromSquadEvent`
   - `SquadLeaderSetEvent`
   - `SquadFinalizedEvent`

3. Refactor SquadBuilderMode:
   - Replace direct ECS calls with service calls
   - Subscribe to events for grid updates
   - Remove GridEditorManager ECS access

4. Test squad building flow

### Phase 4: Remove Cached State (1 day)

**Goal:** Eliminate computed data caching in UI state

1. Remove `ValidMoveTiles` from BattleMapState
2. Add `GetValidMovementTiles()` to CombatService
3. Update CombatMode.Render() to query on-demand
4. Test movement highlighting still works

### Phase 5: Cleanup and Documentation (1 day)

**Goal:** Polish and document the new architecture

1. Add comprehensive service tests
2. Update CLAUDE.md with service layer patterns
3. Create migration guide for remaining modes
4. Remove unused cached state fields

---

## Testing Strategy

### Service Layer Tests

Services should be **fully testable without UI**:

```go
// File: services/combat_service_test.go
func TestCombatService_ExecuteAttack_ValidatesSquadCanAttack(t *testing.T) {
    // Setup
    em := setupTestEntityManager()
    eventBus := events.NewEventBus()
    service := NewCombatService(em, eventBus)

    attackerID := createTestSquad(em, "Attacker", 100) // 100 HP
    targetID := createTestSquad(em, "Target", 50)      // 50 HP

    // Execute
    cmd := &commands.AttackSquadCommand{
        AttackerID: attackerID,
        TargetID:   targetID,
    }
    err := service.ExecuteAttack(cmd)

    // Verify
    assert.NoError(t, err)

    // Check target took damage
    targetHP := getSquadHP(em, targetID)
    assert.Less(t, targetHP, 50, "Target should take damage")
}

func TestSquadBuilderService_AddUnitToSquad_RejectsOverCapacity(t *testing.T) {
    // Setup
    em := setupTestEntityManager()
    eventBus := events.NewEventBus()
    service := NewSquadBuilderService(em, eventBus)

    playerID := createTestPlayer(em)
    squadID, _ := service.CreateSquad(playerID, "Test Squad")

    // Add units until capacity full
    // ... add 6 capacity worth of units ...

    // Try to add one more (should fail)
    cmd := &commands.AddUnitToSquadCommand{
        PlayerID:     playerID,
        SquadID:      squadID,
        TemplateName: "Tank", // 1.0 capacity
        GridRow:      2,
        GridCol:      2,
    }
    err := service.AddUnitToSquad(cmd)

    // Verify
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "insufficient capacity")
}
```

---

## Migration Checklist

Use this checklist to migrate each UI mode:

- [ ] **Identify all game state mutations** in the mode
- [ ] **Create commands** for each user action
- [ ] **Create service methods** to handle commands
- [ ] **Define events** for state changes
- [ ] **Refactor UI** to call services instead of ECS
- [ ] **Add event subscriptions** for UI updates
- [ ] **Remove direct ECS access** from UI code
- [ ] **Write service tests**
- [ ] **Verify UI still works** end-to-end

---

## Benefits Summary

### Immediate Benefits

1. **Testability:** Game logic testable without UI
2. **Reusability:** Same logic works for player, AI, network, replays
3. **Maintainability:** Business rules in one place
4. **Debuggability:** Clear command/event logs

### Long-Term Benefits

1. **Save/Load:** Commands are serializable
2. **Undo/Redo:** Store command history
3. **Network Play:** Send commands over network
4. **AI Development:** AI uses same commands as player
5. **Replay System:** Record and replay command sequences
6. **Analytics:** Track what players actually do

---

## Conclusion

The current GUI architecture mixes presentation and business logic, making the codebase harder to test, maintain, and extend. By introducing a **Service Layer** with **Commands** and **Events**, we can:

1. **Separate concerns** cleanly (UI vs. game logic)
2. **Enable testing** of game logic without UI
3. **Support future features** (networking, AI, replays)
4. **Reduce coupling** between systems

The migration can be done **incrementally** (mode by mode, service by service) without breaking existing functionality.

**Next Steps:**
1. Review this document with the team
2. Start with Phase 1 (foundation)
3. Pick one mode (e.g., CombatMode) for Phase 2 pilot
4. Measure impact and iterate
