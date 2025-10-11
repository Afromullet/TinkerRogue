# CombatController Squad Combat Implementation Plan

**Document Version:** 1.0
**Date:** 2025-10-11
**Status:** Design Complete, Ready for Implementation

---

## Executive Summary

This document provides a complete implementation plan for integrating squad-based tactical combat into `input/combatcontroller.go`. The squad combat system is **35% complete (621 LOC)** with all core combat logic already implemented in `squads/squadcombat.go`. This implementation focuses on **input handling and UI integration** to make the existing squad combat system player-accessible.

**Estimated Implementation Time:** 6-10 hours

---

## Table of Contents

1. [Current State Assessment](#current-state-assessment)
2. [Architecture Overview](#architecture-overview)
3. [Implementation Steps](#implementation-steps)
4. [Detailed Code Implementation](#detailed-code-implementation)
5. [Integration Points](#integration-points)
6. [Testing Strategy](#testing-strategy)
7. [Future Enhancements](#future-enhancements)

---

## Current State Assessment

### What Already Exists ✅

**squadcombat.go (387 LOC)**
- `ExecuteSquadAttack()` - Complete row-based combat with multi-cell unit support
- `CalculateTotalCover()` - Cover system with stacking bonuses
- `calculateUnitDamageByID()` - Damage calculation with role modifiers
- `applyDamageToUnitByID()` - Damage application and death tracking
- Cell-based and row-based targeting modes

**squadqueries.go (140 LOC)**
- `GetUnitIDsInSquad()` - Query all units in a squad
- `GetUnitIDsInRow()` - Query units in specific row (front/mid/back)
- `GetUnitIDsAtGridPosition()` - Query units at specific grid cell
- `GetLeaderID()` - Find squad leader
- `IsSquadDestroyed()` - Check if squad is eliminated
- `FindUnitByID()` - Entity lookup by ID

**components.go (300+ LOC)**
- All 8 component types defined
- SquadData, SquadMemberData, GridPositionData
- UnitRoleData, CoverData, TargetRowData
- LeaderData, AbilitySlotData, CooldownTrackerData

**squadmanager.go (61 LOC)**
- ECS component registration
- Manager initialization

### What's Missing ❌

**CombatController.go Integration**
- Squad combat state tracking
- Player input handling for squad combat
- Row targeting selection UI
- Squad entity click detection
- Ability triggering integration
- Squad destruction cleanup integration

**Current State:** CombatController only handles throwable items (lines 1-178)

---

## Architecture Overview

### Input Flow Design

```
Player Action Flow:
1. Player clicks on enemy squad entity (or presses attack key near squad)
2. CombatController detects squad entity → Enter Squad Combat Mode
3. UI displays row selection (Front/Mid/Back)
4. Player selects target row (keys 1/2/3)
5. Player confirms attack (Enter key)
6. CombatController calls ExecuteSquadAttack()
7. Combat result displayed
8. Destroyed squads cleaned up
9. Abilities checked and triggered
10. Return to normal mode
```

### State Management

```
SharedInputState {
    PrevCursor        Cursor
    PrevThrowInds     []int
    SquadCombatState  *SquadCombatState  // NEW
}

SquadCombatState {
    IsInSquadCombat  bool
    PlayerSquadID    ecs.EntityID
    TargetSquadID    ecs.EntityID
    SelectedRow      int  // 0=front, 1=mid, 2=back
}
```

### Controller Pattern

Following the established pattern from `MovementController`:
- `CanHandle()` - Returns true when squad combat is active
- `OnActivate()` - Initialize squad combat state
- `OnDeactivate()` - Clean up squad combat state
- `HandleInput()` - Process squad combat input

---

## Implementation Steps

### Phase 1: Add Squad Manager (30 minutes)

**File:** `input/combatcontroller.go`

Add squadManager field and update constructor:

```go
type CombatController struct {
    ecsManager   *common.EntityManager
    squadManager *squads.SquadECSManager  // ADD THIS
    playerData   *avatar.PlayerData
    gameMap      *worldmap.GameMap
    playerUI     *gui.PlayerUI
    sharedState  *SharedInputState
}

func NewCombatController(
    ecsManager *common.EntityManager,
    squadManager *squads.SquadECSManager,  // ADD THIS PARAMETER
    playerData *avatar.PlayerData,
    gameMap *worldmap.GameMap,
    playerUI *gui.PlayerUI,
    sharedState *SharedInputState,
) *CombatController {
    return &CombatController{
        ecsManager:   ecsManager,
        squadManager: squadManager,  // ADD THIS
        playerData:   playerData,
        gameMap:      gameMap,
        playerUI:     playerUI,
        sharedState:  sharedState,
    }
}
```

### Phase 2: Add Squad Combat State (30 minutes)

**File:** `input/inputcoordinator.go`

Add SquadCombatState to SharedInputState:

```go
type SharedInputState struct {
    PrevCursor       Cursor
    PrevThrowInds    []int
    SquadCombatState *SquadCombatState  // ADD THIS
}

type SquadCombatState struct {
    IsInSquadCombat bool
    PlayerSquadID   ecs.EntityID
    TargetSquadID   ecs.EntityID
    SelectedRow     int  // Which enemy row to target (0=front, 1=mid, 2=back)
}

// Update NewSharedInputState
func NewSharedInputState() *SharedInputState {
    return &SharedInputState{
        PrevCursor: Cursor{X: 0, Y: 0},
        PrevThrowInds: []int{},
        SquadCombatState: &SquadCombatState{  // ADD THIS
            IsInSquadCombat: false,
            SelectedRow:     0,
        },
    }
}
```

### Phase 3: Update CanHandle() (5 minutes)

**File:** `input/combatcontroller.go` (line 36)

```go
func (cc *CombatController) CanHandle() bool {
    return cc.playerData.InputStates.IsThrowing ||
           cc.sharedState.SquadCombatState.IsInSquadCombat
}
```

### Phase 4: Implement Squad Combat Detection (1 hour)

**File:** `input/combatcontroller.go` (add after line 62)

```go
// HandleInput processes combat input for both throwables and squad combat
func (cc *CombatController) HandleInput() bool {
    inputHandled := false

    // Handle squad combat mode
    if cc.sharedState.SquadCombatState.IsInSquadCombat {
        inputHandled = cc.handleSquadCombat() || inputHandled
    }

    // Handle throwing mode
    if cc.playerData.InputStates.IsThrowing {
        inputHandled = cc.handleThrowable() || inputHandled
    }

    // Check for squad combat initiation (when not in other modes)
    if !cc.playerData.InputStates.IsThrowing &&
       !cc.sharedState.SquadCombatState.IsInSquadCombat {
        inputHandled = cc.checkSquadCombatInitiation() || inputHandled
    }

    return inputHandled
}

// checkSquadCombatInitiation detects clicks on squad entities
func (cc *CombatController) checkSquadCombatInitiation() bool {
    // Check for attack key press near squad
    if inpututil.IsKeyJustReleased(ebiten.KeyF) {
        // Get adjacent position in direction player is facing
        targetPos := cc.getAdjacentPosition()
        squadID := cc.findSquadAtPosition(targetPos)

        if squadID != 0 {
            cc.initiateSquadCombat(squadID)
            return true
        }
    }

    // TODO: Add mouse click detection for squad entities

    return false
}

// getAdjacentPosition returns the position in front of the player
// TODO: Add direction tracking to player data
func (cc *CombatController) getAdjacentPosition() coords.LogicalPosition {
    // For now, default to position directly above player
    return coords.LogicalPosition{
        X: cc.playerData.Pos.X,
        Y: cc.playerData.Pos.Y - 1,
    }
}

// findSquadAtPosition checks if a squad exists at the given position
func (cc *CombatController) findSquadAtPosition(pos coords.LogicalPosition) ecs.EntityID {
    // Query all squads
    for _, result := range cc.squadManager.Manager.Query(squads.SquadTag) {
        squadEntity := result.Entity
        squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)

        // Get squad position (from first unit or squad entity itself)
        // TODO: Add position component to squad entities OR
        // Check if any unit in the squad is at this position
        unitIDs := squads.GetUnitIDsInSquad(squadData.SquadID, cc.squadManager)
        for _, unitID := range unitIDs {
            unit := squads.FindUnitByID(unitID, cc.squadManager)
            if unit != nil && unit.HasComponent(common.PositionComponent) {
                unitPos := common.GetPosition(unit)
                if unitPos.X == pos.X && unitPos.Y == pos.Y {
                    return squadData.SquadID
                }
            }
        }
    }

    return 0
}

// initiateSquadCombat starts squad combat mode
func (cc *CombatController) initiateSquadCombat(enemySquadID ecs.EntityID) {
    // Find or create player's squad
    playerSquadID := cc.findOrCreatePlayerSquad()

    if playerSquadID == 0 {
        // No player squad available
        return
    }

    // Set combat state
    cc.sharedState.SquadCombatState.IsInSquadCombat = true
    cc.sharedState.SquadCombatState.PlayerSquadID = playerSquadID
    cc.sharedState.SquadCombatState.TargetSquadID = enemySquadID
    cc.sharedState.SquadCombatState.SelectedRow = 0 // Default to front row

    // TODO: Show UI overlay for row selection
}

// findOrCreatePlayerSquad finds existing player squad or creates one
func (cc *CombatController) findOrCreatePlayerSquad() ecs.EntityID {
    // TODO: Implement player squad creation/retrieval
    // For now, return 0 (will need proper implementation)

    // Query for squads with player tag
    // OR maintain player squad ID in PlayerData

    return 0
}
```

### Phase 5: Implement Row Targeting (1 hour)

**File:** `input/combatcontroller.go`

```go
// handleSquadCombat processes squad combat input
func (cc *CombatController) handleSquadCombat() bool {
    // Row selection (keys 1, 2, 3)
    if inpututil.IsKeyJustReleased(ebiten.Key1) {
        cc.sharedState.SquadCombatState.SelectedRow = 0 // Front row
        cc.updateRowTargetingUI()
        return true
    }

    if inpututil.IsKeyJustReleased(ebiten.Key2) {
        cc.sharedState.SquadCombatState.SelectedRow = 1 // Mid row
        cc.updateRowTargetingUI()
        return true
    }

    if inpututil.IsKeyJustReleased(ebiten.Key3) {
        cc.sharedState.SquadCombatState.SelectedRow = 2 // Back row
        cc.updateRowTargetingUI()
        return true
    }

    // Confirm attack
    if inpututil.IsKeyJustReleased(ebiten.KeyEnter) {
        cc.executeSquadCombat(
            cc.sharedState.SquadCombatState.PlayerSquadID,
            cc.sharedState.SquadCombatState.TargetSquadID,
        )
        cc.exitSquadCombat()
        return true
    }

    // Cancel squad combat
    if inpututil.IsKeyJustReleased(ebiten.KeyEscape) {
        cc.exitSquadCombat()
        return true
    }

    return false
}

// updateRowTargetingUI updates visual feedback for row selection
func (cc *CombatController) updateRowTargetingUI() {
    // TODO: Highlight selected row on game map
    // Clear previous highlights
    // Apply new highlights based on SelectedRow

    selectedRow := cc.sharedState.SquadCombatState.SelectedRow
    targetSquadID := cc.sharedState.SquadCombatState.TargetSquadID

    // Get units in selected row
    unitIDs := squads.GetUnitIDsInRow(targetSquadID, selectedRow, cc.squadManager)

    // Highlight each unit's position
    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, cc.squadManager)
        if unit != nil && unit.HasComponent(common.PositionComponent) {
            unitPos := common.GetPosition(unit)
            logicalPos := coords.LogicalPosition{X: unitPos.X, Y: unitPos.Y}
            index := coords.CoordManager.LogicalToIndex(logicalPos)
            cc.gameMap.ApplyColorMatrixToIndex(index, graphics.YellowColorMatrix)
        }
    }
}

// exitSquadCombat cleans up squad combat state
func (cc *CombatController) exitSquadCombat() {
    // Clear UI highlights
    // TODO: Clear row highlighting

    // Reset combat state
    cc.sharedState.SquadCombatState.IsInSquadCombat = false
    cc.sharedState.SquadCombatState.PlayerSquadID = 0
    cc.sharedState.SquadCombatState.TargetSquadID = 0
    cc.sharedState.SquadCombatState.SelectedRow = 0
}
```

### Phase 6: Integrate ExecuteSquadAttack (1 hour)

**File:** `input/combatcontroller.go`

```go
// executeSquadCombat performs squad combat between player and enemy squad
func (cc *CombatController) executeSquadCombat(attackerSquadID, defenderSquadID ecs.EntityID) {
    // Get attacker squad
    attackerSquad := squads.GetSquadEntity(attackerSquadID, cc.squadManager)
    if attackerSquad == nil {
        return
    }

    // Increment turn count
    attackerData := common.GetComponentType[*squads.SquadData](attackerSquad, squads.SquadComponent)
    attackerData.TurnCount++

    // Check and trigger abilities BEFORE combat
    cc.checkAndTriggerAbilities(attackerSquadID)

    // Execute the squad attack (uses existing squadcombat.go function!)
    result := squads.ExecuteSquadAttack(attackerSquadID, defenderSquadID, cc.squadManager)

    // Display combat results
    cc.displayCombatResult(result)

    // Process enemy squad turn (if not destroyed)
    if !squads.IsSquadDestroyed(defenderSquadID, cc.squadManager) {
        cc.executeEnemyTurn(defenderSquadID, attackerSquadID)
    }

    // Cleanup destroyed squads
    cc.checkSquadDestruction(attackerSquadID)
    cc.checkSquadDestruction(defenderSquadID)

    // Update player state
    cc.playerData.InputStates.HasKeyInput = true
}

// executeEnemyTurn performs enemy squad's counter-attack
func (cc *CombatController) executeEnemyTurn(enemySquadID, playerSquadID ecs.EntityID) {
    // Get enemy squad
    enemySquad := squads.GetSquadEntity(enemySquadID, cc.squadManager)
    if enemySquad == nil {
        return
    }

    // Increment enemy turn count
    enemyData := common.GetComponentType[*squads.SquadData](enemySquad, squads.SquadComponent)
    enemyData.TurnCount++

    // Check and trigger enemy abilities
    cc.checkAndTriggerAbilities(enemySquadID)

    // Enemy attacks player (simple AI: target front row)
    result := squads.ExecuteSquadAttack(enemySquadID, playerSquadID, cc.squadManager)

    // Display enemy combat results
    cc.displayCombatResult(result)
}

// displayCombatResult shows combat results to player
func (cc *CombatController) displayCombatResult(result *squads.CombatResult) {
    // TODO: Integrate with game's message/log system
    // For now, print to console (temporary)

    fmt.Printf("Combat Result:\n")
    fmt.Printf("  Total Damage: %d\n", result.TotalDamage)
    fmt.Printf("  Units Killed: %d\n", len(result.UnitsKilled))

    for unitID, damage := range result.DamageByUnit {
        unit := squads.FindUnitByID(unitID, cc.squadManager)
        if unit != nil {
            name := common.GetComponentType[*common.Name](unit, common.NameComponent)
            attr := common.GetAttributes(unit)
            fmt.Printf("  %s took %d damage (%d HP remaining)\n",
                name.NameStr, damage, attr.CurrentHealth)
        }
    }
}
```

### Phase 7: Implement Squad Destruction Cleanup (30 minutes)

**File:** `input/combatcontroller.go`

```go
// checkSquadDestruction checks if squad is destroyed and cleans up
func (cc *CombatController) checkSquadDestruction(squadID ecs.EntityID) {
    if !squads.IsSquadDestroyed(squadID, cc.squadManager) {
        return
    }

    // Squad is destroyed - clean up

    // Get all unit IDs
    unitIDs := squads.GetUnitIDsInSquad(squadID, cc.squadManager)

    // Remove each unit entity
    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, cc.squadManager)
        if unit != nil {
            // Remove from map if unit has position
            if unit.HasComponent(common.PositionComponent) {
                unitPos := common.GetPosition(unit)
                logicalPos := coords.LogicalPosition{X: unitPos.X, Y: unitPos.Y}
                index := coords.CoordManager.LogicalToIndex(logicalPos)
                tile := cc.gameMap.Tiles[index]
                tile.Blocked = false
            }

            // Remove unit entity from ECS
            unit.Remove()
        }
    }

    // Remove squad entity
    squadEntity := squads.GetSquadEntity(squadID, cc.squadManager)
    if squadEntity != nil {
        squadEntity.Remove()
    }

    // TODO: Award XP, loot, etc.
}
```

### Phase 8: Implement Ability Triggering (2 hours)

**File:** `input/combatcontroller.go`

```go
// checkAndTriggerAbilities checks if any leader abilities should trigger
func (cc *CombatController) checkAndTriggerAbilities(squadID ecs.EntityID) {
    // Get squad leader
    leaderID := squads.GetLeaderID(squadID, cc.squadManager)
    if leaderID == 0 {
        return // No leader in squad
    }

    // Get leader entity
    leader := squads.FindUnitByID(leaderID, cc.squadManager)
    if leader == nil || !leader.HasComponent(squads.AbilitySlotComponent) {
        return // Leader has no abilities
    }

    // Get ability slots
    abilitySlots := common.GetComponentType[*squads.AbilitySlotData](leader, squads.AbilitySlotComponent)

    // Get cooldown tracker
    var cooldowns *squads.CooldownTrackerData
    if leader.HasComponent(squads.CooldownTrackerComponent) {
        cooldowns = common.GetComponentType[*squads.CooldownTrackerData](leader, squads.CooldownTrackerComponent)
    }

    // Check each ability slot
    for i := 0; i < 4; i++ {
        slot := &abilitySlots.Slots[i]

        // Skip if not equipped or already triggered
        if !slot.IsEquipped || slot.HasTriggered {
            continue
        }

        // Check cooldown
        if cooldowns != nil && cooldowns.Cooldowns[i] > 0 {
            cooldowns.Cooldowns[i]-- // Decrement cooldown
            continue
        }

        // Check trigger condition
        if cc.checkTriggerCondition(squadID, slot.TriggerType, slot.Threshold) {
            // Execute ability
            cc.executeAbility(squadID, slot.AbilityType)

            // Mark as triggered (for once-per-combat abilities)
            slot.HasTriggered = true

            // Set cooldown
            if cooldowns != nil {
                params := squads.GetAbilityParams(slot.AbilityType)
                cooldowns.Cooldowns[i] = params.BaseCooldown
                cooldowns.MaxCooldowns[i] = params.BaseCooldown
            }
        }
    }
}

// checkTriggerCondition evaluates if an ability trigger condition is met
func (cc *CombatController) checkTriggerCondition(squadID ecs.EntityID, triggerType squads.TriggerType, threshold float64) bool {
    switch triggerType {
    case squads.TriggerSquadHPBelow:
        avgHP := cc.calculateAverageSquadHP(squadID)
        return avgHP < threshold

    case squads.TriggerTurnCount:
        squadEntity := squads.GetSquadEntity(squadID, cc.squadManager)
        if squadEntity != nil {
            squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
            return float64(squadData.TurnCount) >= threshold
        }

    case squads.TriggerCombatStart:
        squadEntity := squads.GetSquadEntity(squadID, cc.squadManager)
        if squadEntity != nil {
            squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
            return squadData.TurnCount == 1
        }

    case squads.TriggerMoraleBelow:
        squadEntity := squads.GetSquadEntity(squadID, cc.squadManager)
        if squadEntity != nil {
            squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
            return float64(squadData.Morale) < threshold
        }
    }

    return false
}

// calculateAverageSquadHP returns the average HP percentage of alive units
func (cc *CombatController) calculateAverageSquadHP(squadID ecs.EntityID) float64 {
    unitIDs := squads.GetUnitIDsInSquad(squadID, cc.squadManager)

    if len(unitIDs) == 0 {
        return 0.0
    }

    totalHPPercent := 0.0
    aliveCount := 0

    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, cc.squadManager)
        if unit == nil {
            continue
        }

        attr := common.GetAttributes(unit)
        if attr.CurrentHealth > 0 {
            hpPercent := float64(attr.CurrentHealth) / float64(attr.MaxHealth)
            totalHPPercent += hpPercent
            aliveCount++
        }
    }

    if aliveCount == 0 {
        return 0.0
    }

    return totalHPPercent / float64(aliveCount)
}

// executeAbility performs the ability effect on the squad
func (cc *CombatController) executeAbility(squadID ecs.EntityID, abilityType squads.AbilityType) {
    params := squads.GetAbilityParams(abilityType)

    switch abilityType {
    case squads.AbilityRally:
        cc.applyRally(squadID, params)

    case squads.AbilityHeal:
        cc.applyHeal(squadID, params)

    case squads.AbilityBattleCry:
        cc.applyBattleCry(squadID, params)

    case squads.AbilityFireball:
        // Fireball targets enemy squad
        // TODO: Need to know enemy squad ID in context
        // For now, skip enemy-targeting abilities
    }

    // TODO: Display ability activation message
    fmt.Printf("Squad %d activated %s!\n", squadID, abilityType.String())
}

// applyRally increases squad damage for duration
func (cc *CombatController) applyRally(squadID ecs.EntityID, params squads.AbilityParams) {
    unitIDs := squads.GetUnitIDsInSquad(squadID, cc.squadManager)

    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, cc.squadManager)
        if unit == nil {
            continue
        }

        attr := common.GetAttributes(unit)
        attr.DamageBonus += params.DamageBonus

        // TODO: Add duration tracking (requires temporary buff system)
    }
}

// applyHeal restores HP to all squad units
func (cc *CombatController) applyHeal(squadID ecs.EntityID, params squads.AbilityParams) {
    unitIDs := squads.GetUnitIDsInSquad(squadID, cc.squadManager)

    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, cc.squadManager)
        if unit == nil {
            continue
        }

        attr := common.GetAttributes(unit)
        attr.CurrentHealth += params.HealAmount

        // Cap at max HP
        if attr.CurrentHealth > attr.MaxHealth {
            attr.CurrentHealth = attr.MaxHealth
        }
    }
}

// applyBattleCry increases damage and morale
func (cc *CombatController) applyBattleCry(squadID ecs.EntityID, params squads.AbilityParams) {
    // Increase squad morale
    squadEntity := squads.GetSquadEntity(squadID, cc.squadManager)
    if squadEntity != nil {
        squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
        squadData.Morale += params.MoraleBonus
        if squadData.Morale > 100 {
            squadData.Morale = 100
        }
    }

    // Increase unit damage
    unitIDs := squads.GetUnitIDsInSquad(squadID, cc.squadManager)
    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, cc.squadManager)
        if unit == nil {
            continue
        }

        attr := common.GetAttributes(unit)
        attr.DamageBonus += params.DamageBonus
    }
}
```

---

## Integration Points

### Files to Modify

1. **input/combatcontroller.go** (Primary implementation file)
   - Add all methods listed above
   - Update constructor signature
   - Add squadManager field

2. **input/inputcoordinator.go**
   - Add SquadCombatState struct
   - Update SharedInputState
   - Pass squadManager to CombatController constructor

3. **game_main/main.go** (or wherever InputCoordinator is created)
   - Pass squadManager to CombatController via InputCoordinator

### Import Statements

Add to `input/combatcontroller.go`:

```go
import (
    "fmt"  // For debug output (temporary)
    "game_main/avatar"
    "game_main/common"
    "game_main/coords"
    "game_main/gear"
    "game_main/graphics"
    "game_main/gui"
    "game_main/squads"  // ADD THIS
    "game_main/worldmap"

    "github.com/bytearena/ecs"  // ADD THIS
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)
```

### Constructor Call Updates

**Example:** If InputCoordinator is created in `main.go`:

```go
// Before
combatController := input.NewCombatController(
    ecsManager,
    playerData,
    gameMap,
    playerUI,
    sharedState,
)

// After
combatController := input.NewCombatController(
    ecsManager,
    squadManager,  // ADD THIS
    playerData,
    gameMap,
    playerUI,
    sharedState,
)
```

---

## Testing Strategy

### Phase 1: Unit Testing (Standalone)

Test combat logic in isolation (no map integration):

```go
// File: squads/combatcontroller_test.go

func TestExecuteSquadCombat(t *testing.T) {
    // Create test squads
    manager := squads.NewSquadECSManager()
    squads.InitializeSquadData(manager)

    // Create attacker squad
    attackerSquad := squads.CreateEmptySquad(manager, "Test Attackers")
    // Add units...

    // Create defender squad
    defenderSquad := squads.CreateEmptySquad(manager, "Test Defenders")
    // Add units...

    // Execute combat
    result := squads.ExecuteSquadAttack(
        attackerSquad.GetID(),
        defenderSquad.GetID(),
        manager,
    )

    // Assert
    assert.True(t, result.TotalDamage > 0)
}
```

### Phase 2: Integration Testing

Test input flow with mock components:

1. **Test Squad Detection**
   - Verify `findSquadAtPosition()` finds squad entities
   - Verify `initiateSquadCombat()` sets correct state

2. **Test Row Targeting**
   - Verify key presses change SelectedRow
   - Verify UI updates correctly
   - Verify Enter key triggers combat

3. **Test Combat Execution**
   - Verify `executeSquadCombat()` calls `ExecuteSquadAttack()`
   - Verify results are displayed
   - Verify destroyed squads are cleaned up

### Phase 3: End-to-End Testing

Test full gameplay flow:

1. Spawn enemy squad on map
2. Move player next to enemy squad
3. Press attack key (F)
4. Select row (1/2/3)
5. Confirm attack (Enter)
6. Verify combat executes
7. Verify results displayed
8. Verify destroyed squads removed

### Phase 4: Edge Case Testing

- Empty squads
- Player squad with no units
- Target squad already destroyed
- Abilities with no valid targets
- Multiple squads at same position
- Combat during throwing mode

---

## Future Enhancements

### Short Term (Next 2-4 hours)

1. **Player Squad Management**
   - Create player squad on game start
   - Maintain player squad ID in PlayerData
   - Add companion units to player squad

2. **UI Improvements**
   - Visual squad health bars
   - Row highlighting during targeting
   - Combat log/message display
   - Ability activation notifications

3. **Mouse Input Support**
   - Click squad to initiate combat
   - Click row to select
   - Click confirm to attack

### Medium Term (Next 10-20 hours)

1. **Advanced Abilities**
   - Enemy-targeting abilities (Fireball)
   - Buff duration tracking system
   - Cooldown visualization
   - Ability unlock/progression

2. **AI Improvements**
   - Smart row selection (target weakest row)
   - Ability usage by AI squads
   - Formation awareness

3. **Spawning Integration**
   - Enemy squad spawning system
   - Squad templates from monsterdata.json
   - Level-appropriate squad generation

### Long Term (Next 30-50 hours)

1. **Formation System**
   - Formation presets (Balanced, Defensive, Offensive, Ranged)
   - Formation switching during combat
   - Formation bonuses

2. **Squad Progression**
   - XP and leveling for squads
   - Leader experience system
   - Unit recruitment/upgrade

3. **Map Integration**
   - Squad positioning on world map
   - Squad movement across tiles
   - Multi-squad battles

---

## Summary

### Implementation Checklist

- [ ] Phase 1: Add squadManager field to CombatController (30 min)
- [ ] Phase 2: Add SquadCombatState to SharedInputState (30 min)
- [ ] Phase 3: Update CanHandle() method (5 min)
- [ ] Phase 4: Implement squad combat detection (1 hour)
- [ ] Phase 5: Implement row targeting (1 hour)
- [ ] Phase 6: Integrate ExecuteSquadAttack (1 hour)
- [ ] Phase 7: Implement squad destruction cleanup (30 min)
- [ ] Phase 8: Implement ability triggering (2 hours)
- [ ] Phase 9: Testing and debugging (2-3 hours)

**Total Estimated Time:** 8-10 hours

### Key Benefits

✅ **Reuses existing combat logic** - All complex combat calculations already implemented
✅ **Follows established patterns** - Matches MovementController architecture
✅ **Clean separation of concerns** - Input handling separate from combat logic
✅ **Incremental implementation** - Can test each phase independently
✅ **ECS compliant** - Uses native entity IDs, query-based relationships

### Next Steps

1. **Start with Phase 1-3** - Basic infrastructure (1 hour)
2. **Implement Phase 4** - Squad detection (1 hour)
3. **Test detection** - Verify squad entities are found
4. **Implement Phase 5-6** - Row targeting and combat execution (2 hours)
5. **Test combat flow** - Verify full attack sequence
6. **Implement Phase 7-8** - Cleanup and abilities (2.5 hours)
7. **Full integration testing** - End-to-end gameplay (2-3 hours)

---

**Document Status:** Complete and ready for implementation
**Next Action:** Begin Phase 1 implementation
