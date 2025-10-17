# Turn-Based Tactical Combat System - Implementation Plan

**Version:** 1.0
**Date:** 2025-10-17
**Estimated Effort:** 28-40 hours (4-5 workdays)

## Table of Contents
1. [Overview](#overview)
2. [Requirements](#requirements)
3. [Architecture](#architecture)
4. [Component Specifications](#component-specifications)
5. [System Specifications](#system-specifications)
6. [Attribute Extensions](#attribute-extensions)
7. [File Structure](#file-structure)
8. [Implementation Phases](#implementation-phases)
9. [Integration Patterns](#integration-patterns)
10. [Testing Strategy](#testing-strategy)

---

## Overview

This document outlines the implementation plan for a turn-based tactical combat system that builds on the existing squad combat system. The design follows ECS (Entity-Component-System) architecture principles with pure data components and system-based logic.

### Key Features
- **Multi-Faction Combat**: Support for N factions with multiple squads each
- **Turn-Based Mechanics**: Randomized turn order with free-form squad activation
- **Tactical Movement**: Tile-based movement with terrain and collision
- **Range-Based Combat**: Units can attack from distance based on range attributes
- **Action Economy**: One action per squad per turn (Move+Attack OR Move+Skill)
- **Extensible**: Stub interfaces for skills, magic, and victory conditions

---

## Requirements

### Core Turn-Based Mechanics

1. **Factions/Sides**
   - N number of factions can participate in combat
   - Each faction owns one or more squads
   - Factions have mana/resources for magic abilities

2. **Turn Structure**
   - Turn order randomized once at combat start
   - Order repeats in cycle (A → B → C → A → B → C...)
   - Active faction can move ALL their squads in any order (free-form)
   - Faction explicitly ends turn to advance

3. **Action Economy**
   - Each squad gets ONE action per turn
   - Actions: (Move + Attack) OR (Move + Skill)
   - Cannot perform same action twice

### Movement System

- **Tile-based movement** on existing map grid using LogicalPosition
- **Squad Movement Speed**: Squad moves at speed of SLOWEST unit
- **Movement Rules**:
  - Can move through friendly squads
  - CANNOT move through enemy squads
  - Terrain can restrict movement (existing system integration)
- **Defeated Squads**: Removed immediately from map

### Combat System Extensions

- **Attack Range**: New attribute for units (distance in tiles)
- **Ranged Combat**:
  - If ANY unit in squad has range > 1, squad can attack from that distance
  - Can attack targets X tiles away (X = max range in squad)
  - Only units with enemy in their range actually attack (partial squad attacks)
- **Combat Resolution**: Uses existing `ExecuteSquadAttack()` from `squads/squadcombat.go`
- **Positioning Matters**: Distance affects who can attack

### Actions & Abilities

- **Attack Action**: Uses existing squad combat system with range validation
- **Skill Action**: Squad-level skills (STUB for now - interfaces only)
- **Magic Action**: Faction-wide magic ability that costs mana (STUB for now)
- **Victory Conditions**: Extensible interface (STUB for now)

### AI & Player Support

- System supports:
  - Player vs AI
  - Player vs Player
  - AI vs AI simulation

---

## Architecture

### Design Principles

1. **Pure Data Components**: No logic methods, only data fields
2. **System-Based Logic**: All behavior in systems, not components
3. **Query-Based Relationships**: Use ECS queries, not stored entity pointers
4. **Native EntityID**: Use `ecs.EntityID` throughout, not custom types
5. **Value-Based Keys**: Use value types in maps for O(1) performance

### Package Structure

```
combat/           # New package for turn-based combat
  components.go   # FactionData, TurnStateData, ActionStateData, MapPositionData
  turnmanager.go  # Turn progression logic
  movementsystem.go # Squad movement on map
  combatactionsystem.go # Attack/skill execution
  factionmanager.go # Faction operations
  victory.go      # Victory condition interfaces (STUB)
  combat_test.go  # Integration tests

common/           # Existing package - EXTEND
  commoncomponents.go # Add MovementSpeed, AttackRange to Attributes

squads/           # Existing package - USE
  squadcombat.go  # Use ExecuteSquadAttack()
  squadqueries.go # Use GetUnitIDsInSquad()
```

---

## Component Specifications

### 1. FactionData

Represents a side/team in combat.

```go
// combat/components.go

type FactionData struct {
    FactionID         ecs.EntityID   // Unique faction identifier
    Name              string          // Display name (e.g., "Player", "Goblins")
    Mana              int             // Current mana for magic abilities
    MaxMana           int             // Maximum mana capacity
    IsPlayerControlled bool           // True if controlled by player
    SquadIDs          []ecs.EntityID // Squads owned by this faction (cached)
}
```

**Purpose**: Tracks faction identity, resources, and squad ownership.

**Usage**:
- One entity per faction in combat
- Query via FactionTag
- SquadIDs maintained by FactionManager

---

### 2. TurnStateData

Tracks the global turn state for the combat encounter.

```go
type TurnStateData struct {
    CurrentRound      int             // Current round number (starts at 1)
    TurnOrder         []ecs.EntityID  // Faction IDs in order (randomized at start)
    CurrentTurnIndex  int             // Index into TurnOrder (0 to len-1)
    CombatActive      bool            // True if combat is ongoing
}
```

**Purpose**: Manages turn progression and round tracking.

**Usage**:
- Only ONE entity with TurnStateData exists per combat
- Query via TurnStateTag
- Modified by TurnManager.EndTurn()

---

### 3. ActionStateData

Tracks what actions a squad has performed this turn.

```go
type ActionStateData struct {
    SquadID           ecs.EntityID // Squad this action state belongs to
    HasMoved          bool          // True if squad moved this turn
    HasActed          bool          // True if squad attacked/used skill this turn
    MovementRemaining int           // Tiles left to move (starts at squad speed)
}
```

**Purpose**: Enforces action economy (one action per squad per turn).

**Usage**:
- One entity per squad in combat
- Query via ActionStateTag
- Reset at turn start by TurnManager.ResetSquadActions()
- Checked before allowing actions

---

### 4. MapPositionData

Links squads to map coordinates and faction ownership.

```go
type MapPositionData struct {
    SquadID           ecs.EntityID          // Squad entity
    Position          coords.LogicalPosition // Current tile position on map
    FactionID         ecs.EntityID          // Faction that owns this squad
}
```

**Purpose**: Provides spatial positioning and faction association for squads.

**Usage**:
- One entity per squad in combat
- Query via MapPositionTag
- Updated by MovementSystem.MoveSquad()
- Used for distance calculations and collision detection

---

## System Specifications

### 1. TurnManager

**File**: `combat/turnmanager.go`

**Responsibilities**:
- Initialize combat with randomized turn order
- Track current active faction
- Advance turns and rounds
- Reset squad action states at turn start

**Interface**:

```go
type TurnManager struct {
    manager *common.EntityManager
}

// InitializeCombat sets up turn order and combat state
func (tm *TurnManager) InitializeCombat(factionIDs []ecs.EntityID) error

// GetCurrentFaction returns the active faction's ID
func (tm *TurnManager) GetCurrentFaction() ecs.EntityID

// EndTurn advances to the next faction's turn
func (tm *TurnManager) EndTurn() error

// IsSquadActivatable checks if a squad can act this turn
func (tm *TurnManager) IsSquadActivatable(squadID ecs.EntityID) bool

// ResetSquadActions resets HasMoved/HasActed for all faction's squads
func (tm *TurnManager) ResetSquadActions(factionID ecs.EntityID) error

// GetCurrentRound returns the current round number
func (tm *TurnManager) GetCurrentRound() int

// EndCombat marks combat as finished
func (tm *TurnManager) EndCombat() error
```

**Key Methods**:

```go
// Example: InitializeCombat
func (tm *TurnManager) InitializeCombat(factionIDs []ecs.EntityID) error {
    // 1. Randomize turn order using Fisher-Yates shuffle
    turnOrder := make([]ecs.EntityID, len(factionIDs))
    copy(turnOrder, factionIDs)
    shuffleFactionOrder(turnOrder)

    // 2. Create TurnStateData entity
    turnEntity := tm.manager.World.NewEntity()
    turnEntity.AddComponent(TurnStateComponent, &TurnStateData{
        CurrentRound:     1,
        TurnOrder:        turnOrder,
        CurrentTurnIndex: 0,
        CombatActive:     true,
    })

    // 3. Create ActionStateData for all squads
    for _, factionID := range factionIDs {
        squads := GetSquadsForFaction(factionID, tm.manager)
        for _, squadID := range squads {
            actionEntity := tm.manager.World.NewEntity()
            actionEntity.AddComponent(ActionStateComponent, &ActionStateData{
                SquadID:           squadID,
                HasMoved:          false,
                HasActed:          false,
                MovementRemaining: 0, // Set by MovementSystem
            })
        }
    }

    return nil
}
```

---

### 2. MovementSystem

**File**: `combat/movementsystem.go`

**Responsibilities**:
- Calculate squad movement speed (slowest unit)
- Validate movement destinations
- Execute squad movement
- Handle collision detection

**Interface**:

```go
type MovementSystem struct {
    manager   *common.EntityManager
    posSystem *systems.PositionSystem // For O(1) collision detection
}

// GetSquadMovementSpeed returns tiles per turn (slowest unit)
func (ms *MovementSystem) GetSquadMovementSpeed(squadID ecs.EntityID) int

// CanMoveTo checks if squad can legally move to target position
func (ms *MovementSystem) CanMoveTo(squadID ecs.EntityID, targetPos coords.LogicalPosition) bool

// MoveSquad executes movement and updates position
func (ms *MovementSystem) MoveSquad(squadID ecs.EntityID, targetPos coords.LogicalPosition) error

// GetValidMovementTiles returns all tiles squad can reach this turn
func (ms *MovementSystem) GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition

// GetSquadPosition returns current map position
func (ms *MovementSystem) GetSquadPosition(squadID ecs.EntityID) (coords.LogicalPosition, error)
```

**Key Methods**:

```go
// Example: GetSquadMovementSpeed
func (ms *MovementSystem) GetSquadMovementSpeed(squadID ecs.EntityID) int {
    // Get all units in squad
    unitIDs := squads.GetUnitIDsInSquad(squadID, ms.manager)

    minSpeed := 999
    for _, unitID := range unitIDs {
        unit := findEntityByID(unitID, ms.manager)
        if unit == nil {
            continue
        }

        attr := common.GetAttributes(unit)
        speed := attr.GetMovementSpeed()

        if speed < minSpeed {
            minSpeed = speed
        }
    }

    if minSpeed == 999 {
        return 3 // Default if no units found
    }

    return minSpeed // Squad moves at slowest unit's speed
}

// Example: CanMoveTo
func (ms *MovementSystem) CanMoveTo(squadID ecs.EntityID, targetPos coords.LogicalPosition) bool {
    // 1. Check if tile is occupied using PositionSystem
    occupyingID := ms.posSystem.GetEntityIDAt(targetPos)
    if occupyingID == 0 {
        return true // Empty tile - can move
    }

    // 2. If occupied, check if it's a friendly squad
    occupyingFaction := getFactionOwner(occupyingID, ms.manager)
    squadFaction := getFactionOwner(squadID, ms.manager)

    // Can pass through friendlies, NOT enemies
    return occupyingFaction == squadFaction
}
```

---

### 3. CombatActionSystem

**File**: `combat/combatactionsystem.go`

**Responsibilities**:
- Calculate squad attack range (max range of any unit)
- Validate attack targets are in range
- Execute attacks using existing squad combat system
- Execute skills (STUB)
- Track action usage

**Interface**:

```go
type CombatActionSystem struct {
    manager *common.EntityManager
}

// GetSquadAttackRange returns max attack range (from all units)
func (cas *CombatActionSystem) GetSquadAttackRange(squadID ecs.EntityID) int

// GetSquadsInRange returns all enemy squads within attack range
func (cas *CombatActionSystem) GetSquadsInRange(squadID ecs.EntityID) []ecs.EntityID

// CanSquadAttack checks if squad can attack target (range, actions)
func (cas *CombatActionSystem) CanSquadAttack(squadID, targetID ecs.EntityID) bool

// ExecuteAttackAction performs attack with range validation
func (cas *CombatActionSystem) ExecuteAttackAction(attackerID, defenderID ecs.EntityID) error

// ExecuteSkillAction performs squad skill (STUB)
func (cas *CombatActionSystem) ExecuteSkillAction(squadID ecs.EntityID, skillID int) error

// GetAttackingUnits returns unit IDs that can attack target from current position
func (cas *CombatActionSystem) GetAttackingUnits(squadID, targetID ecs.EntityID) []ecs.EntityID
```

**Key Methods**:

```go
// Example: ExecuteAttackAction (CRITICAL INTEGRATION)
func (cas *CombatActionSystem) ExecuteAttackAction(attackerID, defenderID ecs.EntityID) error {
    // 1. Get positions using map position queries
    attackerPos, err := getSquadMapPosition(attackerID, cas.manager)
    if err != nil {
        return fmt.Errorf("cannot find attacker position: %w", err)
    }

    defenderPos, err := getSquadMapPosition(defenderID, cas.manager)
    if err != nil {
        return fmt.Errorf("cannot find defender position: %w", err)
    }

    // 2. Calculate distance
    distance := attackerPos.ChebyshevDistance(&defenderPos)

    // 3. Get attacker's max range
    maxRange := cas.GetSquadAttackRange(attackerID)

    // 4. Validate range
    if distance > maxRange {
        return fmt.Errorf("target out of range: %d tiles away, max range %d", distance, maxRange)
    }

    // 5. Check if squad has acted
    if !canSquadAct(attackerID, cas.manager) {
        return fmt.Errorf("squad has already acted this turn")
    }

    // 6. Call existing combat system (INTEGRATION POINT)
    result := squads.ExecuteSquadAttack(attackerID, defenderID, cas.manager)

    // 7. Mark squad as acted
    markSquadAsActed(attackerID, cas.manager)

    // 8. Check if defender was destroyed
    if squads.IsSquadDestroyed(defenderID, cas.manager) {
        // Remove from map
        removeSquadFromMap(defenderID, cas.manager)
    }

    return nil
}

// Example: GetSquadAttackRange
func (cas *CombatActionSystem) GetSquadAttackRange(squadID ecs.EntityID) int {
    unitIDs := squads.GetUnitIDsInSquad(squadID, cas.manager)

    maxRange := 1 // Default melee
    for _, unitID := range unitIDs {
        unit := findEntityByID(unitID, cas.manager)
        if unit == nil {
            continue
        }

        attr := common.GetAttributes(unit)
        unitRange := attr.GetAttackRange()

        if unitRange > maxRange {
            maxRange = unitRange
        }
    }

    return maxRange // Squad can attack at max range of any unit
}
```

---

### 4. FactionManager

**File**: `combat/factionmanager.go`

**Responsibilities**:
- Create and manage factions
- Assign squads to factions
- Track faction resources (mana)
- Execute faction-wide magic (STUB)
- Check victory conditions (STUB)

**Interface**:

```go
type FactionManager struct {
    manager *common.EntityManager
}

// CreateFaction creates a new faction entity
func (fm *FactionManager) CreateFaction(name string, isPlayer bool) ecs.EntityID

// AddSquadToFaction assigns a squad to a faction
func (fm *FactionManager) AddSquadToFaction(factionID, squadID ecs.EntityID) error

// RemoveSquadFromFaction removes squad from faction (when destroyed)
func (fm *FactionManager) RemoveSquadFromFaction(factionID, squadID ecs.EntityID) error

// GetFactionSquads returns all squads owned by faction
func (fm *FactionManager) GetFactionSquads(factionID ecs.EntityID) []ecs.EntityID

// UseFactionMagic executes faction-wide ability (STUB)
func (fm *FactionManager) UseFactionMagic(factionID ecs.EntityID, abilityID int) error

// CheckVictoryCondition checks if any faction has won (STUB)
func (fm *FactionManager) CheckVictoryCondition() (won bool, winnerID ecs.EntityID)

// GetFactionMana returns current/max mana
func (fm *FactionManager) GetFactionMana(factionID ecs.EntityID) (current, max int)
```

**Key Methods**:

```go
// Example: CreateFaction
func (fm *FactionManager) CreateFaction(name string, isPlayer bool) ecs.EntityID {
    faction := fm.manager.World.NewEntity()
    factionID := faction.GetID()

    faction.AddComponent(FactionComponent, &FactionData{
        FactionID:         factionID,
        Name:              name,
        Mana:              100,
        MaxMana:           100,
        IsPlayerControlled: isPlayer,
        SquadIDs:          []ecs.EntityID{},
    })

    return factionID
}

// Example: GetFactionSquads (Query-based, not cached)
func (fm *FactionManager) GetFactionSquads(factionID ecs.EntityID) []ecs.EntityID {
    var squadIDs []ecs.EntityID

    // Query all MapPositionData entities
    for _, result := range fm.manager.World.Query(MapPositionTag) {
        mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)
        if mapPos.FactionID == factionID {
            squadIDs = append(squadIDs, mapPos.SquadID)
        }
    }

    return squadIDs
}
```

---

## Attribute Extensions

### Extend common.Attributes

**File**: `common/commoncomponents.go`

Add two new fields to the existing `Attributes` struct:

```go
type Attributes struct {
    // ... existing fields (Strength, Dexterity, Magic, etc.)

    // ========================================
    // TURN-BASED COMBAT ATTRIBUTES
    // ========================================

    MovementSpeed int // Tiles per turn (typical range: 3-7)
    AttackRange   int // Attack distance in tiles (1 = melee, 2+ = ranged)

    // ... existing runtime state
}
```

### Add Derived Methods

```go
// GetMovementSpeed returns tiles per turn with default
func (a *Attributes) GetMovementSpeed() int {
    if a.MovementSpeed <= 0 {
        return 3 // Default movement speed
    }
    return a.MovementSpeed
}

// GetAttackRange returns attack distance with default
func (a *Attributes) GetAttackRange() int {
    if a.AttackRange <= 0 {
        return 1 // Default melee range
    }
    return a.AttackRange
}
```

### Update NewAttributes Constructor

```go
// NewAttributes creates a new Attributes instance with calculated MaxHealth
func NewAttributes(strength, dexterity, magic, leadership, armor, weapon int) Attributes {
    attr := Attributes{
        Strength:      strength,
        Dexterity:     dexterity,
        Magic:         magic,
        Leadership:    leadership,
        Armor:         armor,
        Weapon:        weapon,
        MovementSpeed: 3,  // Default movement
        AttackRange:   1,  // Default melee
        CanAct:        true,
    }

    // Calculate and cache MaxHealth
    attr.MaxHealth = attr.GetMaxHealth()
    attr.CurrentHealth = attr.MaxHealth

    return attr
}
```

### Template Updates

Update unit templates (e.g., `monsterdata.json`) to include new attributes:

```json
{
    "name": "Goblin Warrior",
    "strength": 10,
    "dexterity": 8,
    "magic": 0,
    "leadership": 0,
    "armor": 2,
    "weapon": 2,
    "movementSpeed": 4,
    "attackRange": 1
}
```

---

## File Structure

```
TinkerRogue/
├── combat/                    # NEW PACKAGE
│   ├── components.go          # FactionData, TurnStateData, ActionStateData, MapPositionData
│   ├── turnmanager.go         # Turn progression system
│   ├── movementsystem.go      # Squad movement with collision
│   ├── combatactionsystem.go  # Attack/skill execution
│   ├── factionmanager.go      # Faction operations
│   ├── victory.go             # Victory condition interfaces (STUB)
│   └── combat_test.go         # Integration tests
│
├── common/                    # EXTEND EXISTING
│   └── commoncomponents.go    # Add MovementSpeed, AttackRange to Attributes
│
├── squads/                    # USE EXISTING
│   ├── squadcombat.go         # Use ExecuteSquadAttack()
│   └── squadqueries.go        # Use GetUnitIDsInSquad()
│
└── analysis/                  # DOCUMENTATION
    └── turn_based_combat_plan.md  # This file
```

---

## Implementation Phases

### Phase 1: Core Turn Structure (4-6 hours)

**Goal**: Implement turn order, turn progression, and basic state tracking.

**Components**:
- `FactionData`
- `TurnStateData`
- `ActionStateData`

**Systems**:
- `TurnManager` (basic methods)

**Files**:
- `combat/components.go`
- `combat/turnmanager.go`
- `combat/combat_test.go`

**Tasks**:
1. Define component structs with ECS component registration
2. Implement TurnManager.InitializeCombat() with turn order randomization
3. Implement TurnManager.GetCurrentFaction()
4. Implement TurnManager.EndTurn() with wraparound logic
5. Implement TurnManager.ResetSquadActions()

**Tests**:
```go
TestInitializeCombat_RandomizesTournOrder
TestGetCurrentFaction_ReturnsActiveFaction
TestEndTurn_AdvancesToNextFaction
TestEndTurn_WrapsAroundToFirstFaction
TestResetSquadActions_ClearsActionFlags
```

**Deliverable**: Can create combat with multiple factions, cycle through turns, track active faction.

---

### Phase 2: Movement System (6-8 hours)

**Goal**: Implement squad movement with collision detection and speed calculation.

**Components**:
- `MapPositionData`

**Systems**:
- `MovementSystem`

**Attributes**:
- Add `MovementSpeed` to `common.Attributes`

**Files**:
- Extend `combat/components.go`
- `combat/movementsystem.go`
- Extend `common/commoncomponents.go`

**Tasks**:
1. Add MovementSpeed field and GetMovementSpeed() method to Attributes
2. Implement MovementSystem.GetSquadMovementSpeed() (slowest unit logic)
3. Implement MovementSystem.CanMoveTo() with collision detection
4. Implement MovementSystem.MoveSquad() with position updates
5. Integrate with existing PositionSystem for O(1) lookups
6. Implement MapPositionData creation for squads

**Tests**:
```go
TestGetSquadMovementSpeed_ReturnsSlowesttUnit
TestCanMoveTo_AllowsEmptyTiles
TestCanMoveTo_AllowsFriendlySquads
TestCanMoveTo_BlocksEnemySquads
TestMoveSquad_UpdatesPosition
TestMoveSquad_UpdatesActionState
```

**Deliverable**: Squads can move on map, collision detection works, movement speed varies by squad composition.

---

### Phase 3: Range & Combat (6-8 hours)

**Goal**: Implement range-based combat with integration to existing squad combat system.

**Systems**:
- `CombatActionSystem`

**Attributes**:
- Add `AttackRange` to `common.Attributes`

**Files**:
- `combat/combatactionsystem.go`
- Extend `common/commoncomponents.go`

**Tasks**:
1. Add AttackRange field and GetAttackRange() method to Attributes
2. Implement CombatActionSystem.GetSquadAttackRange() (max range logic)
3. Implement CombatActionSystem.GetSquadsInRange() with distance calculation
4. Implement CombatActionSystem.CanSquadAttack() with range validation
5. Implement CombatActionSystem.ExecuteAttackAction() calling existing ExecuteSquadAttack()
6. Implement partial squad attacks (only units in range)
7. Handle defeated squad removal from map

**Tests**:
```go
TestGetSquadAttackRange_ReturnsMaxRange
TestGetSquadsInRange_FilterssByDistance
TestCanSquadAttack_ValidatesRange
TestExecuteAttackAction_MeleeAttack
TestExecuteAttackAction_RangedAttack
TestExecuteAttackAction_PartialSquadAttack
TestExecuteAttackAction_MarksSquadAsActed
TestExecuteAttackAction_RemovesDefeatedSquad
```

**Deliverable**: Squads can attack based on range, combat integrates with existing system, positioning matters tactically.

---

### Phase 4: Action Management (4-6 hours)

**Goal**: Enforce action economy and prevent double actions.

**Systems**:
- Complete `ActionStateData` tracking in all systems

**Files**:
- Extend `combat/turnmanager.go`
- Extend `combat/movementsystem.go`
- Extend `combat/combatactionsystem.go`

**Tasks**:
1. Implement action state checking before movement
2. Implement action state checking before combat
3. Update ActionStateData.HasMoved when moving
4. Update ActionStateData.HasActed when attacking
5. Implement MovementRemaining tracking
6. Add validation: cannot move + attack if already acted

**Tests**:
```go
TestActionState_PreventsDualActions
TestActionState_AllowsMoveAndAttack
TestActionState_AllowsMoveAndSkill
TestActionState_ResetOnTurnStart
TestMovementRemaining_DecrementsOnMove
TestHasActed_PreventsSecondAction
```

**Deliverable**: Action economy enforced, squads limited to one action per turn, state resets correctly.

---

### Phase 5: Faction System (4-6 hours)

**Goal**: Implement faction management, squad ownership, and resource tracking.

**Systems**:
- `FactionManager`

**Files**:
- `combat/factionmanager.go`

**Tasks**:
1. Implement FactionManager.CreateFaction()
2. Implement FactionManager.AddSquadToFaction()
3. Implement FactionManager.RemoveSquadFromFaction()
4. Implement FactionManager.GetFactionSquads() (query-based)
5. Implement faction mana tracking (basic increment/decrement)
6. Create MapPositionData when adding squad to faction

**Tests**:
```go
TestCreateFaction_CreatesValidFaction
TestAddSquadToFaction_AssignsOwnership
TestGetFactionSquads_ReturnsOwnedSquads
TestRemoveSquadFromFaction_RemovesOwnership
TestFactionMana_TracksResources
```

**Deliverable**: Factions own squads, squad ownership tracked via queries, mana system exists.

---

### Phase 6: Stubs & Integration (4-6 hours)

**Goal**: Create stub interfaces for future features and full combat loop.

**Systems**:
- Skill system interfaces
- Magic system interfaces
- Victory condition interfaces

**Files**:
- `combat/victory.go`
- Extend all systems with stub methods

**Tasks**:
1. Define VictoryCondition interface with CheckVictory() method
2. Create DefaultVictoryCondition stub (always returns false)
3. Define SquadSkill interface with Execute() method
4. Implement CombatActionSystem.ExecuteSkillAction() stub
5. Implement FactionManager.UseFactionMagic() stub
6. Create full combat loop integration test (Player vs AI simulation)
7. Add UI hook points (turn indication, action feedback)

**Tests**:
```go
TestFullCombatLoop_TwoFactions
TestFullCombatLoop_ThreeFactions
TestVictoryCondition_StubAlwaysFalse
TestSkillExecution_StubDoesNothing
TestMagicExecution_StubCostsMana
```

**Deliverable**: Complete turn-based combat system with stubs for future features, full combat loop tested.

---

## Integration Patterns

### Pattern 1: Range-Based Combat Integration

**Problem**: How to integrate existing `ExecuteSquadAttack()` with range validation?

**Solution**:

```go
// combat/combatactionsystem.go

func (cas *CombatActionSystem) ExecuteAttackAction(attackerID, defenderID ecs.EntityID) error {
    // Step 1: Get positions using map position queries
    attackerPos, err := getSquadMapPosition(attackerID, cas.manager)
    if err != nil {
        return fmt.Errorf("cannot find attacker position: %w", err)
    }

    defenderPos, err := getSquadMapPosition(defenderID, cas.manager)
    if err != nil {
        return fmt.Errorf("cannot find defender position: %w", err)
    }

    // Step 2: Calculate distance using existing coords system
    distance := attackerPos.ChebyshevDistance(&defenderPos)

    // Step 3: Get attacker's max range from unit attributes
    maxRange := cas.GetSquadAttackRange(attackerID)

    // Step 4: Validate range
    if distance > maxRange {
        return fmt.Errorf("target out of range: %d tiles away, max range %d", distance, maxRange)
    }

    // Step 5: Check action state
    if !canSquadAct(attackerID, cas.manager) {
        return fmt.Errorf("squad has already acted this turn")
    }

    // Step 6: Call existing combat system (INTEGRATION POINT)
    // ExecuteSquadAttack already handles:
    // - Target selection (lowest HP)
    // - Damage calculation (hit/dodge/crit)
    // - Cover system
    // - Unit deaths
    result := squads.ExecuteSquadAttack(attackerID, defenderID, cas.manager)

    // Step 7: Update action state
    markSquadAsActed(attackerID, cas.manager)

    // Step 8: Handle defeated squads
    if squads.IsSquadDestroyed(defenderID, cas.manager) {
        removeSquadFromMap(defenderID, cas.manager)
    }

    // Step 9: Log combat result for UI
    logCombatResult(result)

    return nil
}
```

**Key Points**:
- Uses existing `ExecuteSquadAttack()` for all combat logic
- Only adds range validation layer
- Leverages existing cover, hit/dodge, damage systems
- No duplication of combat mechanics

---

### Pattern 2: Movement Validation with PositionSystem

**Problem**: How to efficiently check tile occupancy and friendliness?

**Solution**:

```go
// combat/movementsystem.go

func (ms *MovementSystem) CanMoveTo(squadID ecs.EntityID, targetPos coords.LogicalPosition) bool {
    // Step 1: Use PositionSystem for O(1) lookup
    occupyingID := common.GlobalPositionSystem.GetEntityIDAt(targetPos)

    // Empty tile - always valid
    if occupyingID == 0 {
        return true
    }

    // Step 2: Check if occupied by a squad (not terrain/item)
    if !isSquad(occupyingID, ms.manager) {
        // Occupied by terrain/obstacle
        return false
    }

    // Step 3: Query-based faction lookup (no stored pointers)
    occupyingFaction := getFactionOwner(occupyingID, ms.manager)
    squadFaction := getFactionOwner(squadID, ms.manager)

    // Can pass through friendlies, NOT enemies
    return occupyingFaction == squadFaction
}

// Helper: Query-based faction lookup
func getFactionOwner(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
    for _, result := range manager.World.Query(MapPositionTag) {
        mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)
        if mapPos.SquadID == squadID {
            return mapPos.FactionID
        }
    }
    return 0 // No owner found
}
```

**Key Points**:
- Uses existing PositionSystem for O(1) spatial queries
- Query-based faction lookup (proper ECS pattern)
- Handles empty tiles, terrain, friendly/enemy squads
- No entity pointers stored

---

### Pattern 3: Turn State Queries

**Problem**: How to track current turn without global state?

**Solution**:

```go
// combat/turnmanager.go

// Query-based approach (no stored pointers)
func GetCurrentActiveFaction(manager *common.EntityManager) ecs.EntityID {
    // There is only one TurnStateData entity in combat
    for _, result := range manager.World.Query(TurnStateTag) {
        turnState := common.GetComponentType[*TurnStateData](result.Entity, TurnStateComponent)
        currentIndex := turnState.CurrentTurnIndex
        return turnState.TurnOrder[currentIndex]
    }
    return 0 // No active combat
}

func (tm *TurnManager) EndTurn() error {
    // Step 1: Query turn state
    turnEntity := findTurnStateEntity(tm.manager)
    if turnEntity == nil {
        return fmt.Errorf("no active combat")
    }

    turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)

    // Step 2: Advance turn index
    turnState.CurrentTurnIndex++

    // Step 3: Wrap around to start of turn order
    if turnState.CurrentTurnIndex >= len(turnState.TurnOrder) {
        turnState.CurrentTurnIndex = 0
        turnState.CurrentRound++ // Increment round
    }

    // Step 4: Reset actions for new faction
    newFactionID := turnState.TurnOrder[turnState.CurrentTurnIndex]
    tm.ResetSquadActions(newFactionID)

    return nil
}
```

**Key Points**:
- Single TurnStateData entity per combat
- Query-based access (no global variable)
- Turn order as slice of EntityIDs
- Wraparound logic for continuous rounds

---

### Pattern 4: Squad Speed Calculation

**Problem**: How to calculate squad speed from individual unit speeds?

**Solution**:

```go
// combat/movementsystem.go

func (ms *MovementSystem) GetSquadMovementSpeed(squadID ecs.EntityID) int {
    // Step 1: Get all units in squad using existing query
    unitIDs := squads.GetUnitIDsInSquad(squadID, ms.manager)

    if len(unitIDs) == 0 {
        return 3 // Default if no units
    }

    // Step 2: Find slowest unit
    minSpeed := 999
    for _, unitID := range unitIDs {
        // Query entity by ID
        unit := findEntityByID(unitID, ms.manager)
        if unit == nil {
            continue // Skip if unit not found
        }

        // Get attributes
        attr := common.GetAttributes(unit)
        if attr == nil {
            continue // Skip if no attributes
        }

        // Get movement speed
        speed := attr.GetMovementSpeed()

        if speed < minSpeed {
            minSpeed = speed
        }
    }

    if minSpeed == 999 {
        return 3 // Default if no valid units
    }

    return minSpeed // Squad moves at slowest unit's speed
}
```

**Key Points**:
- Uses existing `GetUnitIDsInSquad()` query
- Iterates all units to find minimum speed
- Handles edge cases (no units, missing attributes)
- Squad movement limited by slowest member

---

### Pattern 5: Faction-Squad Relationships

**Problem**: How to track squad ownership without storing entity pointers?

**Solution**:

```go
// combat/factionmanager.go

// Query-based approach (proper ECS pattern)
func (fm *FactionManager) GetFactionSquads(factionID ecs.EntityID) []ecs.EntityID {
    var squadIDs []ecs.EntityID

    // Query all MapPositionData entities
    for _, result := range fm.manager.World.Query(MapPositionTag) {
        mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)

        // Check if squad belongs to this faction
        if mapPos.FactionID == factionID {
            squadIDs = append(squadIDs, mapPos.SquadID)
        }
    }

    return squadIDs
}

// Adding squad to faction
func (fm *FactionManager) AddSquadToFaction(factionID, squadID ecs.EntityID) error {
    // Step 1: Verify faction exists
    faction := findFactionByID(factionID, fm.manager)
    if faction == nil {
        return fmt.Errorf("faction %d not found", factionID)
    }

    // Step 2: Verify squad exists
    squad := findSquadByID(squadID, fm.manager)
    if squad == nil {
        return fmt.Errorf("squad %d not found", squadID)
    }

    // Step 3: Create MapPositionData to establish relationship
    mapPosEntity := fm.manager.World.NewEntity()
    mapPosEntity.AddComponent(MapPositionComponent, &MapPositionData{
        SquadID:   squadID,
        Position:  coords.LogicalPosition{X: 0, Y: 0}, // Set by placement
        FactionID: factionID,
    })

    // Note: NO entity pointers stored, relationship discovered via query

    return nil
}
```

**Key Points**:
- Relationships stored in MapPositionData, not in FactionData.SquadIDs
- Query-based lookup (proper ECS pattern)
- No entity pointers stored
- Decoupled architecture

---

## Testing Strategy

### Unit Tests (Per System)

Each system has isolated unit tests:

```go
// combat/turnmanager_test.go
TestInitializeCombat_RandomizesTurnOrder
TestGetCurrentFaction_ReturnsActiveFaction
TestEndTurn_AdvancesToNextFaction
TestEndTurn_WrapsAround
TestResetSquadActions_ClearsFlags

// combat/movementsystem_test.go
TestGetSquadMovementSpeed_ReturnsSlowest
TestCanMoveTo_AllowsEmptyTiles
TestCanMoveTo_BlocksEnemies
TestMoveSquad_UpdatesPosition

// combat/combatactionsystem_test.go
TestGetSquadAttackRange_ReturnsMax
TestExecuteAttackAction_ValidatesRange
TestExecuteAttackAction_CallsSquadCombat
TestExecuteAttackAction_MarksAsActed

// combat/factionmanager_test.go
TestCreateFaction_CreatesValid
TestGetFactionSquads_ReturnsOwned
TestAddSquadToFaction_EstablishesLink
```

### Integration Tests

Full combat simulation tests:

```go
// combat/combat_test.go

func TestFullCombatLoop_TwoFactionsPlayerVsAI(t *testing.T) {
    // Setup
    manager := common.NewEntityManager()
    turnMgr := NewTurnManager(manager)
    factionMgr := NewFactionManager(manager)
    moveSys := NewMovementSystem(manager)
    combatSys := NewCombatActionSystem(manager)

    // Create factions
    playerID := factionMgr.CreateFaction("Player", true)
    aiID := factionMgr.CreateFaction("Goblins", false)

    // Create squads
    playerSquad1 := createTestSquad("Knights", 5, 5)
    playerSquad2 := createTestSquad("Archers", 7, 7)
    aiSquad1 := createTestSquad("Goblin Warriors", 15, 15)

    // Assign to factions
    factionMgr.AddSquadToFaction(playerID, playerSquad1)
    factionMgr.AddSquadToFaction(playerID, playerSquad2)
    factionMgr.AddSquadToFaction(aiID, aiSquad1)

    // Initialize combat
    turnMgr.InitializeCombat([]ecs.EntityID{playerID, aiID})

    // Simulate turns
    for round := 0; round < 5; round++ {
        currentFaction := turnMgr.GetCurrentFaction()

        if currentFaction == playerID {
            // Player turn: move and attack
            moveSys.MoveSquad(playerSquad1, coords.LogicalPosition{X: 10, Y: 10})
            combatSys.ExecuteAttackAction(playerSquad1, aiSquad1)
        } else {
            // AI turn: move towards player
            moveSys.MoveSquad(aiSquad1, coords.LogicalPosition{X: 12, Y: 12})
        }

        // End turn
        turnMgr.EndTurn()
    }

    // Verify combat state
    assert.True(t, combatActive(manager))
}
```

### Performance Tests

Benchmark critical paths:

```go
// combat/combat_test.go

func BenchmarkGetSquadMovementSpeed(b *testing.B) {
    // Setup squad with 9 units
    manager := setupBenchmarkManager()
    moveSys := NewMovementSystem(manager)
    squadID := createLargeSquad(9)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        moveSys.GetSquadMovementSpeed(squadID)
    }
}

func BenchmarkCanMoveTo(b *testing.B) {
    // Setup map with multiple squads
    manager := setupBenchmarkManager()
    moveSys := NewMovementSystem(manager)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        moveSys.CanMoveTo(testSquadID, coords.LogicalPosition{X: 10, Y: 10})
    }
}
```

---

## Summary

This implementation plan provides:

### Architecture
- ✅ **4 new ECS components** for factions, turns, actions, positions
- ✅ **4 new systems** for turns, movement, combat, factions
- ✅ **2 attribute extensions** for movement speed and attack range
- ✅ **6 implementation phases** (28-40 hours total)

### Integration
- ✅ Uses existing `ExecuteSquadAttack()` with range validation
- ✅ Leverages `GlobalPositionSystem` for O(1) collision detection
- ✅ Integrates with existing squad queries and attributes
- ✅ Query-based relationships (no stored entity pointers)

### Extensibility
- ✅ Stub interfaces for skills, magic, victory conditions
- ✅ Support for Player vs AI, PvP, AI vs AI
- ✅ Easy to add new actions, abilities, faction types

### Best Practices
- ✅ **Pure data components** (no logic methods)
- ✅ **System-based logic** (proper separation of concerns)
- ✅ **Query-based patterns** (ECS compliance)
- ✅ **Go standards** (error handling, clear APIs)
- ✅ **Performance** (O(1) lookups where possible)

### Next Steps

1. **Review and approve this plan**
2. **Begin Phase 1**: Core turn structure (4-6 hours)
3. **Test each phase** before moving to next
4. **Integrate incrementally** with existing codebase
5. **Document as you go** (update this file with learnings)

---

**End of Implementation Plan**
