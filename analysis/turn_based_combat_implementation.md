# Turn-Based Tactical Combat System - Implementation Guide

**Version:** 2.0
**Date:** 2025-10-21
**Estimated Effort:** 40-56 hours (5-7 workdays)

---

## TABLE OF CONTENTS

1. [Overview](#overview)
2. [Requirements](#requirements)
3. [Architecture](#architecture)
4. [Component Specifications](#component-specifications)
5. [System Specifications](#system-specifications)
6. [Attribute Extensions](#attribute-extensions)
7. [Helper Functions](#helper-functions)
8. [File Structure](#file-structure)
9. [Implementation Phases](#implementation-phases)
10. [Integration Patterns](#integration-patterns)
11. [Testing Strategy](#testing-strategy)
12. [Code Examples](#code-examples)

---

## OVERVIEW

This document provides a complete implementation guide for a turn-based tactical combat system that builds on the existing squad combat system. The design follows ECS (Entity-Component-System) architecture principles with pure data components and system-based logic.

### Key Features

- **Multi-Faction Combat**: Support for N factions with multiple squads each
- **Turn-Based Mechanics**: Randomized turn order with free-form squad activation
- **Tactical Movement**: Tile-based movement with terrain and collision
- **Range-Based Combat**: Units can attack from distance based on range attributes
- **Action Economy**: One action per squad per turn (Move + Attack OR Move + Skill)
- **Extensible**: Stub interfaces for skills, magic, and victory conditions

### Integration Points

- Uses existing `squads.ExecuteSquadAttack()` for combat resolution
- Leverages `PositionSystem` for O(1) spatial queries
- Extends `common.Attributes` with MovementSpeed and AttackRange
- Integrates with existing squad queries and unit management

---

## REQUIREMENTS

### Core Turn-Based Mechanics

**Factions/Sides:**
- N number of factions can participate in combat
- Each faction owns one or more squads
- Factions have mana/resources for magic abilities

**Turn Structure:**
- Turn order randomized once at combat start
- Order repeats in cycle (A → B → C → A → B → C...)
- Active faction can move ALL their squads in any order (free-form)
- Faction explicitly ends turn to advance

**Action Economy:**
- Each squad gets ONE action per turn
- Actions: (Move + Attack) OR (Move + Skill)
- Cannot perform same action twice

### Movement System

- **Tile-based movement** on existing map grid using LogicalPosition
- **Squad Movement Speed**: Squad moves at speed of SLOWEST unit
- **Movement Rules**:
  - Can move through friendly squads
  - CANNOT move through enemy squads
  - Terrain can restrict movement
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

System supports:
- Player vs AI
- Player vs Player
- AI vs AI simulation

---

## ARCHITECTURE

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
  queries.go      # Helper functions for entity/component queries
  turnmanager.go  # Turn progression logic
  movementsystem.go # Squad movement on map
  combatactionsystem.go # Attack/skill execution
  factionmanager.go # Faction operations
  victory.go      # Victory condition interfaces (STUB)
  testing.go      # Test helper functions
  combat_test.go  # Integration tests

common/           # Existing package - EXTEND
  commoncomponents.go # Add MovementSpeed, AttackRange to Attributes

squads/           # Existing package - USE
  squadcombat.go  # Use ExecuteSquadAttack()
  squadqueries.go # Use GetUnitIDsInSquad()
```

### Position System Architecture

**Two-Level Position System:**

**Squad Level (Map):**
- Squad entity tracked by `PositionSystem` spatial grid (canonical source)
- Squad has `MapPositionData` component (stores faction ownership + cached position)
- Squad occupies ONE map tile (e.g., X=10, Y=5)

**Unit Level (Formation):**
- Unit entities have `GridPositionData` (row/col within 3x3 squad grid)
- Units NOT tracked by PositionSystem (they're internal to squad)
- Units positioned relative to squad anchor

**Example:**
```
Squad at Map Position (10, 5):
+---+---+---+
| U | U | U |  Row 0
+---+---+---+
| U |   | U |  Row 1
+---+---+---+
| U | U |   |  Row 2
+---+---+---+
  0   1   2  (Columns)
```

---

## COMPONENT SPECIFICATIONS

### 1. FactionData

Represents a side/team in combat.

```go
// combat/components.go

type FactionData struct {
    FactionID          ecs.EntityID   // Unique faction identifier
    Name               string          // Display name (e.g., "Player", "Goblins")
    Mana               int             // Current mana for magic abilities
    MaxMana            int             // Maximum mana capacity
    IsPlayerControlled bool           // True if controlled by player
}
```

**Purpose**: Tracks faction identity and resources.

**Usage**:
- One entity per faction in combat
- Query via FactionTag
- Squad ownership discovered via MapPositionData queries (not stored here)

---

### 2. TurnStateData

Tracks the global turn state for the combat encounter.

```go
type TurnStateData struct {
    CurrentRound     int             // Current round number (starts at 1)
    TurnOrder        []ecs.EntityID  // Faction IDs in order (randomized at start)
    CurrentTurnIndex int             // Index into TurnOrder (0 to len-1)
    CombatActive     bool            // True if combat is ongoing
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
- MovementRemaining initialized to squad speed each turn
- Decremented as squad moves
- Checked before allowing actions

---

### 4. MapPositionData

Links squads to map coordinates and faction ownership.

```go
type MapPositionData struct {
    SquadID   ecs.EntityID          // Squad entity
    Position  coords.LogicalPosition // Current tile position (cached from PositionSystem)
    FactionID ecs.EntityID          // Faction that owns this squad
}
```

**Purpose**: Provides spatial positioning and faction association for squads.

**Usage**:
- One entity per squad in combat
- Query via MapPositionTag
- Position is cached from PositionSystem (canonical source)
- Updated by MovementSystem.MoveSquad()
- Used for distance calculations and faction queries

---

### Component Registration

```go
// combat/components.go

package combat

import (
    "game_main/coords"
    "github.com/bytearena/ecs"
)

// Component and tag variables
var (
    FactionComponent      *ecs.Component
    TurnStateComponent    *ecs.Component
    ActionStateComponent  *ecs.Component
    MapPositionComponent  *ecs.Component

    FactionTag      ecs.Tag
    TurnStateTag    ecs.Tag
    ActionStateTag  ecs.Tag
    MapPositionTag  ecs.Tag
)

// init registers all combat components with the ECS
func init() {
    // Create components
    FactionComponent = ecs.NewComponent()
    TurnStateComponent = ecs.NewComponent()
    ActionStateComponent = ecs.NewComponent()
    MapPositionComponent = ecs.NewComponent()

    // Build query tags
    FactionTag = ecs.BuildTag(FactionComponent)
    TurnStateTag = ecs.BuildTag(TurnStateComponent)
    ActionStateTag = ecs.BuildTag(ActionStateComponent)
    MapPositionTag = ecs.BuildTag(MapPositionComponent)
}

// Component data structures (defined above)
```

---

## SYSTEM SPECIFICATIONS

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

func NewTurnManager(manager *common.EntityManager) *TurnManager

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

**Key Implementation Details**:

```go
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
            tm.createActionStateForSquad(squadID)
        }
    }

    // 4. Reset actions for first faction
    firstFaction := turnOrder[0]
    tm.ResetSquadActions(firstFaction)

    return nil
}

func (tm *TurnManager) createActionStateForSquad(squadID ecs.EntityID) {
    actionEntity := tm.manager.World.NewEntity()
    actionEntity.AddComponent(ActionStateComponent, &ActionStateData{
        SquadID:           squadID,
        HasMoved:          false,
        HasActed:          false,
        MovementRemaining: 0, // Set by ResetSquadActions
    })
}

func (tm *TurnManager) ResetSquadActions(factionID ecs.EntityID) error {
    squads := GetSquadsForFaction(factionID, tm.manager)

    // Create MovementSystem to get squad speeds
    moveSys := NewMovementSystem(tm.manager, common.GlobalPositionSystem)

    for _, squadID := range squads {
        actionEntity := findActionStateEntity(squadID, tm.manager)
        if actionEntity == nil {
            continue
        }

        actionState := common.GetComponentType[*ActionStateData](actionEntity, ActionStateComponent)

        // Reset flags
        actionState.HasMoved = false
        actionState.HasActed = false

        // Initialize MovementRemaining from squad speed
        squadSpeed := moveSys.GetSquadMovementSpeed(squadID)
        actionState.MovementRemaining = squadSpeed
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

func NewMovementSystem(manager *common.EntityManager, posSystem *systems.PositionSystem) *MovementSystem

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

**Key Implementation Details**:

```go
func (ms *MovementSystem) GetSquadMovementSpeed(squadID ecs.EntityID) int {
    // Get all units in squad
    unitIDs := squads.GetUnitIDsInSquad(squadID, ms.manager)

    if len(unitIDs) == 0 {
        return 3 // Default if no units
    }

    minSpeed := 999
    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, ms.manager)
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
        return 3 // Default if no valid units
    }

    return minSpeed // Squad moves at slowest unit's speed
}

func (ms *MovementSystem) CanMoveTo(squadID ecs.EntityID, targetPos coords.LogicalPosition) bool {
    // 1. Check if tile is occupied using PositionSystem
    occupyingID := ms.posSystem.GetEntityIDAt(targetPos)
    if occupyingID == 0 {
        return true // Empty tile - can move
    }

    // 2. Check if occupied by a squad (not terrain/item)
    if !isSquad(occupyingID, ms.manager) {
        return false // Occupied by terrain/obstacle
    }

    // 3. If occupied by squad, check if it's friendly
    occupyingFaction := getFactionOwner(occupyingID, ms.manager)
    squadFaction := getFactionOwner(squadID, ms.manager)

    // Can pass through friendlies, NOT enemies
    return occupyingFaction == squadFaction
}

func (ms *MovementSystem) MoveSquad(squadID ecs.EntityID, targetPos coords.LogicalPosition) error {
    // Validate squad can move
    if !canSquadMove(squadID, ms.manager) {
        return fmt.Errorf("squad has no movement remaining")
    }

    // Get current position
    currentPos, err := ms.GetSquadPosition(squadID)
    if err != nil {
        return fmt.Errorf("cannot get current position: %w", err)
    }

    // Calculate movement cost (using Chebyshev distance for 8-directional movement)
    movementCost := currentPos.ChebyshevDistance(&targetPos)

    // Check if squad has enough movement
    actionStateEntity := findActionStateEntity(squadID, ms.manager)
    if actionStateEntity == nil {
        return fmt.Errorf("no action state for squad")
    }

    actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
    if actionState.MovementRemaining < movementCost {
        return fmt.Errorf("insufficient movement: need %d, have %d", movementCost, actionState.MovementRemaining)
    }

    // Validate destination
    if !ms.CanMoveTo(squadID, targetPos) {
        return fmt.Errorf("cannot move to %v", targetPos)
    }

    // Update MapPositionData (cached position)
    mapPosEntity := findMapPositionEntity(squadID, ms.manager)
    if mapPosEntity == nil {
        return fmt.Errorf("squad not on map")
    }

    mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
    mapPos.Position = targetPos

    // Update PositionSystem spatial grid (canonical source)
    ms.posSystem.MoveEntity(squadID, currentPos, targetPos)

    // Update action state
    decrementMovementRemaining(squadID, movementCost, ms.manager)
    markSquadAsMoved(squadID, ms.manager)

    return nil
}

func (ms *MovementSystem) GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition {
    currentPos, err := ms.GetSquadPosition(squadID)
    if err != nil {
        return []coords.LogicalPosition{}
    }

    // Get remaining movement
    actionStateEntity := findActionStateEntity(squadID, ms.manager)
    if actionStateEntity == nil {
        return []coords.LogicalPosition{}
    }

    actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
    movementRange := actionState.MovementRemaining

    if movementRange <= 0 {
        return []coords.LogicalPosition{}
    }

    // Simple flood-fill for valid tiles (8-directional movement with Chebyshev distance)
    validTiles := []coords.LogicalPosition{}

    for x := currentPos.X - movementRange; x <= currentPos.X + movementRange; x++ {
        for y := currentPos.Y - movementRange; y <= currentPos.Y + movementRange; y++ {
            testPos := coords.LogicalPosition{X: x, Y: y}

            // Check if within Chebyshev distance
            distance := currentPos.ChebyshevDistance(&testPos)
            if distance > movementRange {
                continue
            }

            // Check if can move to this tile
            if ms.CanMoveTo(squadID, testPos) {
                validTiles = append(validTiles, testPos)
            }
        }
    }

    return validTiles
}
```

---

### 3. CombatActionSystem

**File**: `combat/combatactionsystem.go`

**Responsibilities**:
- Calculate squad attack range (max range of any unit)
- Validate attack targets are in range
- Execute attacks using existing squad combat system
- Handle partial squad attacks (only units in range)
- Execute skills (STUB)
- Track action usage

**Interface**:

```go
type CombatActionSystem struct {
    manager *common.EntityManager
}

func NewCombatActionSystem(manager *common.EntityManager) *CombatActionSystem

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

**Key Implementation Details**:

```go
func (cas *CombatActionSystem) ExecuteAttackAction(attackerID, defenderID ecs.EntityID) error {
    // 1. Get positions
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

    // 6. Filter units by range (partial squad attacks)
    attackingUnits := cas.GetAttackingUnits(attackerID, defenderID)

    // Temporarily disable out-of-range units
    allUnits := squads.GetUnitIDsInSquad(attackerID, cas.manager)
    disabledUnits := []ecs.EntityID{}

    for _, unitID := range allUnits {
        if !containsEntity(attackingUnits, unitID) {
            unit := squads.FindUnitByID(unitID, cas.manager)
            attr := common.GetAttributes(unit)
            if attr.CanAct {
                attr.CanAct = false
                disabledUnits = append(disabledUnits, unitID)
            }
        }
    }

    // 7. Execute attack (only CanAct=true units participate)
    result := squads.ExecuteSquadAttack(attackerID, defenderID, cas.manager)

    // 8. Re-enable disabled units
    for _, unitID := range disabledUnits {
        unit := squads.FindUnitByID(unitID, cas.manager)
        attr := common.GetAttributes(unit)
        attr.CanAct = true
    }

    // 9. Mark squad as acted
    markSquadAsActed(attackerID, cas.manager)

    // 10. Check if defender was destroyed
    if squads.IsSquadDestroyed(defenderID, cas.manager) {
        removeSquadFromMap(defenderID, cas.manager)
    }

    // 11. Log result
    logCombatResult(result)

    return nil
}

func (cas *CombatActionSystem) GetSquadAttackRange(squadID ecs.EntityID) int {
    unitIDs := squads.GetUnitIDsInSquad(squadID, cas.manager)

    maxRange := 1 // Default melee
    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, cas.manager)
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

func (cas *CombatActionSystem) GetAttackingUnits(squadID, targetID ecs.EntityID) []ecs.EntityID {
    // Get positions
    attackerPos, _ := getSquadMapPosition(squadID, cas.manager)
    defenderPos, _ := getSquadMapPosition(targetID, cas.manager)

    distance := attackerPos.ChebyshevDistance(&defenderPos)

    // Get all units in attacking squad
    allUnits := squads.GetUnitIDsInSquad(squadID, cas.manager)

    var attackingUnits []ecs.EntityID

    for _, unitID := range allUnits {
        unit := squads.FindUnitByID(unitID, cas.manager)
        if unit == nil {
            continue
        }

        attr := common.GetAttributes(unit)
        unitRange := attr.GetAttackRange()

        // Check if this unit can reach the target
        if distance <= unitRange {
            attackingUnits = append(attackingUnits, unitID)
        }
    }

    return attackingUnits
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

func NewFactionManager(manager *common.EntityManager) *FactionManager

// CreateFaction creates a new faction entity
func (fm *FactionManager) CreateFaction(name string, isPlayer bool) ecs.EntityID

// AddSquadToFaction assigns a squad to a faction
func (fm *FactionManager) AddSquadToFaction(factionID, squadID ecs.EntityID, position coords.LogicalPosition) error

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

**Key Implementation Details**:

```go
func (fm *FactionManager) CreateFaction(name string, isPlayer bool) ecs.EntityID {
    faction := fm.manager.World.NewEntity()
    factionID := faction.GetID()

    faction.AddComponent(FactionComponent, &FactionData{
        FactionID:          factionID,
        Name:               name,
        Mana:               100,
        MaxMana:            100,
        IsPlayerControlled: isPlayer,
    })

    return factionID
}

func (fm *FactionManager) AddSquadToFaction(factionID, squadID ecs.EntityID, position coords.LogicalPosition) error {
    // Verify faction exists
    faction := findFactionByID(factionID, fm.manager)
    if faction == nil {
        return fmt.Errorf("faction %d not found", factionID)
    }

    // Verify squad exists
    squad := squads.FindSquadByID(squadID, fm.manager)
    if squad == nil {
        return fmt.Errorf("squad %d not found", squadID)
    }

    // Create MapPositionData to establish relationship
    mapPosEntity := fm.manager.World.NewEntity()
    mapPosEntity.AddComponent(MapPositionComponent, &MapPositionData{
        SquadID:   squadID,
        Position:  position,
        FactionID: factionID,
    })

    // Register in PositionSystem
    common.GlobalPositionSystem.AddEntity(squadID, position)

    return nil
}

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

## ATTRIBUTE EXTENSIONS

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

## HELPER FUNCTIONS

**File**: `combat/queries.go`

```go
package combat

import (
    "fmt"
    "game_main/common"
    "game_main/coords"
    "game_main/squads"
    "github.com/bytearena/ecs"
    "math/rand/v2"
)

// ========================================
// ENTITY LOOKUP HELPERS
// ========================================

// findEntityByID finds an entity by its ID
func findEntityByID(entityID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
    // Use entity map if available (O(1))
    if manager.EntityMap != nil {
        return manager.EntityMap[entityID]
    }

    // Fallback: Search all entities (O(n))
    for _, result := range manager.World.Query(ecs.BuildTag()) {
        if result.Entity.GetID() == entityID {
            return result.Entity
        }
    }
    return nil
}

// findFactionByID finds a faction entity by faction ID
func findFactionByID(factionID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
    for _, result := range manager.World.Query(FactionTag) {
        faction := result.Entity
        factionData := common.GetComponentType[*FactionData](faction, FactionComponent)
        if factionData.FactionID == factionID {
            return faction
        }
    }
    return nil
}

// findTurnStateEntity finds the single TurnStateData entity
func findTurnStateEntity(manager *common.EntityManager) *ecs.Entity {
    for _, result := range manager.World.Query(TurnStateTag) {
        return result.Entity // Only one should exist
    }
    return nil
}

// findMapPositionEntity finds MapPositionData for a squad
func findMapPositionEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
    for _, result := range manager.World.Query(MapPositionTag) {
        mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)
        if mapPos.SquadID == squadID {
            return result.Entity
        }
    }
    return nil
}

// findActionStateEntity finds ActionStateData for a squad
func findActionStateEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
    for _, result := range manager.World.Query(ActionStateTag) {
        actionState := common.GetComponentType[*ActionStateData](result.Entity, ActionStateComponent)
        if actionState.SquadID == squadID {
            return result.Entity
        }
    }
    return nil
}

// ========================================
// SQUAD RELATIONSHIP HELPERS
// ========================================

// getFactionOwner returns the faction that owns a squad
func getFactionOwner(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
    mapPosEntity := findMapPositionEntity(squadID, manager)
    if mapPosEntity == nil {
        return 0
    }

    mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
    return mapPos.FactionID
}

// getSquadMapPosition returns the current map position of a squad
func getSquadMapPosition(squadID ecs.EntityID, manager *common.EntityManager) (coords.LogicalPosition, error) {
    mapPosEntity := findMapPositionEntity(squadID, manager)
    if mapPosEntity == nil {
        return coords.LogicalPosition{}, fmt.Errorf("squad %d not on map", squadID)
    }

    mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
    return mapPos.Position, nil
}

// GetSquadsForFaction returns all squads owned by a faction
func GetSquadsForFaction(factionID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    var squadIDs []ecs.EntityID

    for _, result := range manager.World.Query(MapPositionTag) {
        mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)
        if mapPos.FactionID == factionID {
            squadIDs = append(squadIDs, mapPos.SquadID)
        }
    }

    return squadIDs
}

// isSquad checks if an entity ID represents a squad
func isSquad(entityID ecs.EntityID, manager *common.EntityManager) bool {
    mapPosEntity := findMapPositionEntity(entityID, manager)
    return mapPosEntity != nil
}

// ========================================
// ACTION STATE HELPERS
// ========================================

// canSquadAct checks if a squad can perform an action this turn
func canSquadAct(squadID ecs.EntityID, manager *common.EntityManager) bool {
    actionStateEntity := findActionStateEntity(squadID, manager)
    if actionStateEntity == nil {
        return false
    }

    actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
    return !actionState.HasActed
}

// canSquadMove checks if a squad can still move this turn
func canSquadMove(squadID ecs.EntityID, manager *common.EntityManager) bool {
    actionStateEntity := findActionStateEntity(squadID, manager)
    if actionStateEntity == nil {
        return false
    }

    actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
    return actionState.MovementRemaining > 0
}

// markSquadAsActed marks a squad as having used its combat action
func markSquadAsActed(squadID ecs.EntityID, manager *common.EntityManager) {
    actionStateEntity := findActionStateEntity(squadID, manager)
    if actionStateEntity == nil {
        return
    }

    actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
    actionState.HasActed = true
}

// markSquadAsMoved marks a squad as having used movement
func markSquadAsMoved(squadID ecs.EntityID, manager *common.EntityManager) {
    actionStateEntity := findActionStateEntity(squadID, manager)
    if actionStateEntity == nil {
        return
    }

    actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
    actionState.HasMoved = true
}

// decrementMovementRemaining reduces squad's remaining movement
func decrementMovementRemaining(squadID ecs.EntityID, amount int, manager *common.EntityManager) {
    actionStateEntity := findActionStateEntity(squadID, manager)
    if actionStateEntity == nil {
        return
    }

    actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
    actionState.MovementRemaining -= amount
    if actionState.MovementRemaining < 0 {
        actionState.MovementRemaining = 0
    }
}

// ========================================
// COMBAT STATE HELPERS
// ========================================

// removeSquadFromMap removes a squad from the combat map
func removeSquadFromMap(squadID ecs.EntityID, manager *common.EntityManager) error {
    // Find and remove MapPositionData
    mapPosEntity := findMapPositionEntity(squadID, manager)
    if mapPosEntity == nil {
        return fmt.Errorf("squad %d not on map", squadID)
    }

    // Get position before removal
    mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
    position := mapPos.Position

    // Remove from ECS
    manager.World.RemoveEntity(mapPosEntity)

    // Remove from PositionSystem spatial grid
    common.GlobalPositionSystem.RemoveEntity(squadID, position)

    return nil
}

// combatActive checks if combat is currently ongoing
func combatActive(manager *common.EntityManager) bool {
    turnStateEntity := findTurnStateEntity(manager)
    if turnStateEntity == nil {
        return false
    }

    turnState := common.GetComponentType[*TurnStateData](turnStateEntity, TurnStateComponent)
    return turnState.CombatActive
}

// ========================================
// UTILITY HELPERS
// ========================================

// shuffleFactionOrder randomizes faction turn order using Fisher-Yates
func shuffleFactionOrder(factionIDs []ecs.EntityID) {
    for i := len(factionIDs) - 1; i > 0; i-- {
        j := rand.IntN(i + 1)
        factionIDs[i], factionIDs[j] = factionIDs[j], factionIDs[i]
    }
}

// logCombatResult logs combat result for debugging/UI
func logCombatResult(result *squads.CombatResult) {
    // TODO: Implement event system for UI
    fmt.Printf("Combat result: %d damage, %d kills\n", result.TotalDamage, len(result.UnitsKilled))
}

// contains checks if a slice contains a position
func contains(positions []coords.LogicalPosition, pos coords.LogicalPosition) bool {
    for _, p := range positions {
        if p.X == pos.X && p.Y == pos.Y {
            return true
        }
    }
    return false
}

// containsEntity checks if a slice contains an entity ID
func containsEntity(entities []ecs.EntityID, entityID ecs.EntityID) bool {
    for _, e := range entities {
        if e == entityID {
            return true
        }
    }
    return false
}
```

---

## FILE STRUCTURE

```
TinkerRogue/
├── combat/                    # NEW PACKAGE
│   ├── components.go          # Components + init() registration
│   ├── queries.go             # Helper query functions
│   ├── turnmanager.go         # Turn progression system
│   ├── movementsystem.go      # Squad movement with collision
│   ├── combatactionsystem.go  # Attack/skill execution
│   ├── factionmanager.go      # Faction operations
│   ├── victory.go             # Victory condition interfaces (STUB)
│   ├── testing.go             # Test helper functions
│   └── combat_test.go         # Integration tests
│
├── common/                    # EXTEND EXISTING
│   └── commoncomponents.go    # Add MovementSpeed, AttackRange to Attributes
│
├── squads/                    # USE EXISTING
│   ├── squadcombat.go         # Use ExecuteSquadAttack()
│   └── squadqueries.go        # Use GetUnitIDsInSquad()
│
└── analysis/
    └── turn_based_combat_implementation.md  # This file
```

---

## IMPLEMENTATION PHASES

### Phase 0: Core Infrastructure (4-6 hours)

**Goal**: Set up component registration and helper query functions.

**Deliverables**:
1. `combat/components.go` with init() function
2. `combat/queries.go` with all helper functions
3. Component registration tests
4. Query function tests

**Tasks**:
1. Create combat package directory
2. Define component structs (FactionData, TurnStateData, ActionStateData, MapPositionData)
3. Implement init() function for component registration
4. Implement all query helper functions from queries.go
5. Write unit tests for entity lookup functions

**Tests**:
```go
TestComponentRegistration_AllComponentsExist
TestFindEntityByID_ReturnsCorrectEntity
TestGetFactionOwner_ReturnsOwningFaction
TestGetSquadMapPosition_ReturnsPosition
TestCanSquadAct_ChecksActionState
```

---

### Phase 1: Core Turn Structure (4-6 hours)

**Goal**: Implement turn order, turn progression, and basic state tracking.

**Deliverables**:
1. Complete TurnManager implementation
2. Turn order randomization working
3. Action state lifecycle working
4. Turn advancement tests passing

**Tasks**:
1. Implement TurnManager struct and constructor
2. Implement InitializeCombat() with Fisher-Yates shuffle
3. Implement GetCurrentFaction()
4. Implement EndTurn() with wraparound logic
5. Implement ResetSquadActions() with MovementRemaining initialization
6. Implement createActionStateForSquad()
7. Write comprehensive tests

**Tests**:
```go
TestInitializeCombat_RandomizesTurnOrder
TestGetCurrentFaction_ReturnsActiveFaction
TestEndTurn_AdvancesToNextFaction
TestEndTurn_WrapsAroundToFirstFaction
TestResetSquadActions_ClearsActionFlags
TestResetSquadActions_InitializesMovementRemaining
```

---

### Phase 2: Movement System (6-8 hours)

**Goal**: Implement squad movement with collision detection and speed calculation.

**Deliverables**:
1. Complete MovementSystem implementation
2. Squad speed calculation (slowest unit)
3. Collision detection working
4. Valid movement tiles calculation
5. Position system integration

**Tasks**:
1. Add MovementSpeed field to common.Attributes
2. Implement GetMovementSpeed() method
3. Implement MovementSystem struct and constructor
4. Implement GetSquadMovementSpeed() (slowest unit logic)
5. Implement CanMoveTo() with collision detection
6. Implement MoveSquad() with position updates
7. Implement GetValidMovementTiles() (flood-fill)
8. Integrate with PositionSystem for O(1) lookups
9. Write movement tests

**Tests**:
```go
TestGetSquadMovementSpeed_ReturnsSlowestUnit
TestCanMoveTo_AllowsEmptyTiles
TestCanMoveTo_AllowsFriendlySquads
TestCanMoveTo_BlocksEnemySquads
TestMoveSquad_UpdatesPosition
TestMoveSquad_UpdatesActionState
TestMoveSquad_UpdatesPositionSystem
TestGetValidMovementTiles_ReturnsReachableTiles
```

---

### Phase 3: Range & Combat (6-8 hours)

**Goal**: Implement range-based combat with integration to existing squad combat system.

**Deliverables**:
1. Complete CombatActionSystem implementation
2. Attack range calculation (max range)
3. Range validation working
4. Partial squad attacks implemented
5. Integration with ExecuteSquadAttack()

**Tasks**:
1. Add AttackRange field to common.Attributes
2. Implement GetAttackRange() method
3. Implement CombatActionSystem struct and constructor
4. Implement GetSquadAttackRange() (max range logic)
5. Implement GetSquadsInRange() with distance calculation
6. Implement CanSquadAttack() with range validation
7. Implement GetAttackingUnits() for partial attacks
8. Implement ExecuteAttackAction() with unit filtering
9. Handle defeated squad removal from map
10. Write combat tests

**Tests**:
```go
TestGetSquadAttackRange_ReturnsMaxRange
TestGetSquadsInRange_FiltersByDistance
TestCanSquadAttack_ValidatesRange
TestExecuteAttackAction_MeleeAttack
TestExecuteAttackAction_RangedAttack
TestExecuteAttackAction_PartialSquadAttack
TestExecuteAttackAction_MarksSquadAsActed
TestExecuteAttackAction_RemovesDefeatedSquad
TestGetAttackingUnits_FiltersbyUnitRange
```

---

### Phase 4: Action Management (4-6 hours)

**Goal**: Enforce action economy and prevent double actions.

**Deliverables**:
1. Complete action state validation
2. Action economy enforced
3. Movement + Attack interactions working
4. State reset working correctly

**Tasks**:
1. Add action state checking before movement (already in Phase 2)
2. Add action state checking before combat (already in Phase 3)
3. Verify MovementRemaining tracking works correctly
4. Test action combinations (Move+Attack, Attack only, Move only)
5. Verify state resets on turn start
6. Write action economy tests

**Tests**:
```go
TestActionState_PreventsDualActions
TestActionState_AllowsMoveAndAttack
TestActionState_AllowsMoveAndSkill
TestActionState_ResetOnTurnStart
TestMovementRemaining_DecrementsOnMove
TestHasActed_PreventsSecondAction
TestMoveAndAttack_InSameTurn
```

---

### Phase 5: Faction System (4-6 hours)

**Goal**: Implement faction management, squad ownership, and resource tracking.

**Deliverables**:
1. Complete FactionManager implementation
2. Faction creation working
3. Squad ownership tracked via queries
4. Mana system operational

**Tasks**:
1. Implement FactionManager struct and constructor
2. Implement CreateFaction()
3. Implement AddSquadToFaction() with position placement
4. Implement RemoveSquadFromFaction()
5. Implement GetFactionSquads() (query-based)
6. Implement faction mana tracking (basic)
7. Write faction tests

**Tests**:
```go
TestCreateFaction_CreatesValidFaction
TestAddSquadToFaction_AssignsOwnership
TestAddSquadToFaction_PlacesOnMap
TestGetFactionSquads_ReturnsOwnedSquads
TestRemoveSquadFromFaction_RemovesOwnership
TestFactionMana_TracksResources
```

---

### Phase 6: Stubs & Integration (4-6 hours)

**Goal**: Create stub interfaces for future features and full combat loop.

**Deliverables**:
1. Skill system interface
2. Magic system interface
3. Victory condition interface
4. Full combat loop tested
5. Integration tests passing

**Tasks**:
1. Define VictoryCondition interface
2. Create DefaultVictoryCondition stub
3. Define SquadSkill interface
4. Implement ExecuteSkillAction() stub
5. Implement UseFactionMagic() stub
6. Create full combat loop integration test
7. Write stub tests

**Tests**:
```go
TestFullCombatLoop_TwoFactions
TestFullCombatLoop_ThreeFactions
TestVictoryCondition_StubAlwaysFalse
TestSkillExecution_StubDoesNothing
TestMagicExecution_StubCostsMana
TestFullCombat_PlayerVsAI
```

---

### Phase 7: UI/Input Layer (6-8 hours)

**Goal**: Implement player input controller and command handling.

**Deliverables**:
1. CombatController implementation
2. Squad selection working
3. Movement command handling
4. Attack targeting working
5. Turn end command functional

**Tasks**:
1. Create CombatController struct
2. Implement HandleSquadSelection()
3. Implement HandleMoveCommand()
4. Implement HandleAttackCommand()
5. Implement HandleEndTurn()
6. Cache valid moves/targets for UI
7. Write input handling tests

**Tests**:
```go
TestSquadSelection_SelectsOwnedSquad
TestSquadSelection_RejectsEnemySquad
TestMoveCommand_ValidatesDestination
TestAttackCommand_ValidatesTarget
TestEndTurn_AdvancesTurn
TestValidMoves_CachedCorrectly
TestValidTargets_CachedCorrectly
```

---

## INTEGRATION PATTERNS

### Pattern 1: Range-Based Combat Integration

**Integration with existing squad combat system:**

```go
func (cas *CombatActionSystem) ExecuteAttackAction(attackerID, defenderID ecs.EntityID) error {
    // Step 1: Get positions
    attackerPos, _ := getSquadMapPosition(attackerID, cas.manager)
    defenderPos, _ := getSquadMapPosition(defenderID, cas.manager)

    // Step 2: Calculate distance
    distance := attackerPos.ChebyshevDistance(&defenderPos)

    // Step 3: Validate range
    maxRange := cas.GetSquadAttackRange(attackerID)
    if distance > maxRange {
        return fmt.Errorf("target out of range")
    }

    // Step 4: Check action state
    if !canSquadAct(attackerID, cas.manager) {
        return fmt.Errorf("squad already acted")
    }

    // Step 5: Filter units by range (partial squad attacks)
    attackingUnits := cas.GetAttackingUnits(attackerID, defenderID)

    // Step 6: Disable out-of-range units temporarily
    disabledUnits := cas.disableUnitsNotInRange(attackerID, attackingUnits)

    // Step 7: Execute attack using existing system
    result := squads.ExecuteSquadAttack(attackerID, defenderID, cas.manager)

    // Step 8: Re-enable units
    cas.enableUnits(disabledUnits)

    // Step 9: Update state
    markSquadAsActed(attackerID, cas.manager)

    // Step 10: Handle defeated squads
    if squads.IsSquadDestroyed(defenderID, cas.manager) {
        removeSquadFromMap(defenderID, cas.manager)
    }

    return nil
}
```

**Key Points**:
- Uses existing ExecuteSquadAttack() for all combat logic
- Only adds range validation and unit filtering
- Leverages existing cover, hit/dodge, damage systems
- No duplication of combat mechanics

---

### Pattern 2: Movement Validation with PositionSystem

**Efficient collision detection:**

```go
func (ms *MovementSystem) CanMoveTo(squadID ecs.EntityID, targetPos coords.LogicalPosition) bool {
    // Step 1: Use PositionSystem for O(1) lookup
    occupyingID := ms.posSystem.GetEntityIDAt(targetPos)

    // Empty tile - always valid
    if occupyingID == 0 {
        return true
    }

    // Step 2: Check if occupied by squad
    if !isSquad(occupyingID, ms.manager) {
        return false // Terrain/obstacle
    }

    // Step 3: Query-based faction lookup
    occupyingFaction := getFactionOwner(occupyingID, ms.manager)
    squadFaction := getFactionOwner(squadID, ms.manager)

    // Can pass through friendlies, NOT enemies
    return occupyingFaction == squadFaction
}
```

**Key Points**:
- Uses existing PositionSystem for O(1) spatial queries
- Query-based faction lookup (proper ECS pattern)
- Handles empty tiles, terrain, friendly/enemy squads
- No entity pointers stored

---

### Pattern 3: Turn State Queries

**Query-based turn tracking:**

```go
func (tm *TurnManager) GetCurrentFaction() ecs.EntityID {
    // Only one TurnStateData entity exists
    turnEntity := findTurnStateEntity(tm.manager)
    if turnEntity == nil {
        return 0
    }

    turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)
    currentIndex := turnState.CurrentTurnIndex
    return turnState.TurnOrder[currentIndex]
}

func (tm *TurnManager) EndTurn() error {
    turnEntity := findTurnStateEntity(tm.manager)
    if turnEntity == nil {
        return fmt.Errorf("no active combat")
    }

    turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)

    // Advance turn index
    turnState.CurrentTurnIndex++

    // Wrap around to start
    if turnState.CurrentTurnIndex >= len(turnState.TurnOrder) {
        turnState.CurrentTurnIndex = 0
        turnState.CurrentRound++
    }

    // Reset actions for new faction
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

**Aggregate unit speeds:**

```go
func (ms *MovementSystem) GetSquadMovementSpeed(squadID ecs.EntityID) int {
    // Get all units using existing query
    unitIDs := squads.GetUnitIDsInSquad(squadID, ms.manager)

    if len(unitIDs) == 0 {
        return 3 // Default
    }

    // Find slowest unit
    minSpeed := 999
    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, ms.manager)
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
        return 3 // Default if no valid units
    }

    return minSpeed // Squad limited by slowest member
}
```

**Key Points**:
- Uses existing GetUnitIDsInSquad() query
- Iterates all units to find minimum speed
- Handles edge cases (no units, missing attributes)
- Squad movement limited by slowest member

---

### Pattern 5: Faction-Squad Relationships

**Query-based ownership:**

```go
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

func (fm *FactionManager) AddSquadToFaction(factionID, squadID ecs.EntityID, position coords.LogicalPosition) error {
    // Verify faction and squad exist
    faction := findFactionByID(factionID, fm.manager)
    squad := squads.FindSquadByID(squadID, fm.manager)

    if faction == nil || squad == nil {
        return fmt.Errorf("faction or squad not found")
    }

    // Create MapPositionData to establish relationship
    mapPosEntity := fm.manager.World.NewEntity()
    mapPosEntity.AddComponent(MapPositionComponent, &MapPositionData{
        SquadID:   squadID,
        Position:  position,
        FactionID: factionID,
    })

    // Register in PositionSystem
    common.GlobalPositionSystem.AddEntity(squadID, position)

    return nil
}
```

**Key Points**:
- Relationships stored in MapPositionData
- Query-based lookup (proper ECS pattern)
- No entity pointers stored
- Decoupled architecture

---

## TESTING STRATEGY

### Unit Tests (Per System)

**TurnManager Tests:**
```go
// combat/turnmanager_test.go
TestInitializeCombat_RandomizesTurnOrder
TestGetCurrentFaction_ReturnsActiveFaction
TestEndTurn_AdvancesToNextFaction
TestEndTurn_WrapsAround
TestResetSquadActions_ClearsFlags
TestResetSquadActions_InitializesMovement
TestIsSquadActivatable_ChecksFactionTurn
TestGetCurrentRound_ReturnsRoundNumber
TestEndCombat_MarksCombatInactive
```

**MovementSystem Tests:**
```go
// combat/movementsystem_test.go
TestGetSquadMovementSpeed_ReturnsSlowest
TestCanMoveTo_AllowsEmptyTiles
TestCanMoveTo_BlocksEnemies
TestCanMoveTo_AllowsFriendlies
TestMoveSquad_UpdatesPosition
TestMoveSquad_UpdatesActionState
TestMoveSquad_ValidatesMovementRemaining
TestGetValidMovementTiles_ReturnsReachable
```

**CombatActionSystem Tests:**
```go
// combat/combatactionsystem_test.go
TestGetSquadAttackRange_ReturnsMax
TestExecuteAttackAction_ValidatesRange
TestExecuteAttackAction_CallsSquadCombat
TestExecuteAttackAction_MarksAsActed
TestExecuteAttackAction_RemovesDefeatedSquad
TestGetAttackingUnits_FiltersByRange
TestPartialSquadAttack_OnlySomeUnitsAttack
```

**FactionManager Tests:**
```go
// combat/factionmanager_test.go
TestCreateFaction_CreatesValid
TestGetFactionSquads_ReturnsOwned
TestAddSquadToFaction_EstablishesLink
TestRemoveSquadFromFaction_RemovesLink
TestGetFactionMana_ReturnsValues
```

---

### Integration Tests

**Full Combat Simulation:**

```go
// combat/combat_test.go

func TestFullCombatLoop_TwoFactionsPlayerVsAI(t *testing.T) {
    // Setup
    manager := common.NewEntityManager()
    turnMgr := NewTurnManager(manager)
    factionMgr := NewFactionManager(manager)
    moveSys := NewMovementSystem(manager, common.GlobalPositionSystem)
    combatSys := NewCombatActionSystem(manager)

    // Create factions
    playerID := factionMgr.CreateFaction("Player", true)
    aiID := factionMgr.CreateFaction("Goblins", false)

    // Create squads
    playerSquad1 := CreateTestSquad(manager, "Knights", 5)
    playerSquad2 := CreateTestSquad(manager, "Archers", 7)
    aiSquad1 := CreateTestSquad(manager, "Goblin Warriors", 5)

    // Assign to factions
    factionMgr.AddSquadToFaction(playerID, playerSquad1, coords.LogicalPosition{X: 5, Y: 5})
    factionMgr.AddSquadToFaction(playerID, playerSquad2, coords.LogicalPosition{X: 7, Y: 7})
    factionMgr.AddSquadToFaction(aiID, aiSquad1, coords.LogicalPosition{X: 15, Y: 15})

    // Initialize combat
    turnMgr.InitializeCombat([]ecs.EntityID{playerID, aiID})

    // Simulate 5 rounds
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

---

### Test Helper Functions

**File**: `combat/testing.go`

```go
package combat

import (
    "game_main/common"
    "game_main/coords"
    "game_main/squads"
    "github.com/bytearena/ecs"
)

// CreateTestFaction creates a faction for testing
func CreateTestFaction(manager *common.EntityManager, name string, isPlayer bool) ecs.EntityID {
    fm := NewFactionManager(manager)
    return fm.CreateFaction(name, isPlayer)
}

// CreateTestSquad creates a squad with test units
func CreateTestSquad(manager *common.EntityManager, name string, unitCount int) ecs.EntityID {
    // Create squad entity
    squadEntity := manager.World.NewEntity()
    squadID := squadEntity.GetID()

    squadEntity.AddComponent(squads.SquadComponent, &squads.SquadData{
        SquadID:   squadID,
        Name:      name,
        Formation: squads.FormationBalanced,
        MaxUnits:  9,
    })

    // Create test units
    for i := 0; i < unitCount; i++ {
        CreateTestUnit(manager, squadID, i)
    }

    return squadID
}

// CreateTestUnit creates a unit entity
func CreateTestUnit(manager *common.EntityManager, squadID ecs.EntityID, index int) ecs.EntityID {
    unitEntity := manager.World.NewEntity()
    unitID := unitEntity.GetID()

    // Add attributes
    unitEntity.AddComponent(common.AttributesComponent, &common.Attributes{
        Strength:      10,
        Dexterity:     10,
        Magic:         0,
        Leadership:    0,
        Armor:         2,
        Weapon:        2,
        MovementSpeed: 5,
        AttackRange:   1,
        CurrentHealth: 30,
        MaxHealth:     30,
        CanAct:        true,
    })

    // Add squad membership
    unitEntity.AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
        SquadID: squadID,
    })

    // Position in 3x3 grid
    row := index / 3
    col := index % 3
    unitEntity.AddComponent(squads.GridPositionComponent, &squads.GridPositionData{
        AnchorRow: row,
        AnchorCol: col,
        Width:     1,
        Height:    1,
    })

    return unitID
}

// PlaceSquadOnMap places a squad at a position
func PlaceSquadOnMap(manager *common.EntityManager, factionID, squadID ecs.EntityID, pos coords.LogicalPosition) {
    mapPosEntity := manager.World.NewEntity()
    mapPosEntity.AddComponent(MapPositionComponent, &MapPositionData{
        SquadID:   squadID,
        Position:  pos,
        FactionID: factionID,
    })

    // Register in PositionSystem
    common.GlobalPositionSystem.AddEntity(squadID, pos)
}
```

---

## CODE EXAMPLES

### Example 1: Complete TurnManager Implementation

See [System Specifications - TurnManager](#1-turnmanager) for full implementation with all methods.

---

### Example 2: Complete MovementSystem Implementation

See [System Specifications - MovementSystem](#2-movementsystem) for full implementation with all methods.

---

### Example 3: Setting Up a Combat Encounter

```go
package main

import (
    "game_main/combat"
    "game_main/common"
    "game_main/coords"
)

func SetupCombat() {
    // Create entity manager
    manager := common.NewEntityManager()

    // Create systems
    turnMgr := combat.NewTurnManager(manager)
    factionMgr := combat.NewFactionManager(manager)
    moveSys := combat.NewMovementSystem(manager, common.GlobalPositionSystem)
    combatSys := combat.NewCombatActionSystem(manager)

    // Create factions
    playerFaction := factionMgr.CreateFaction("Player", true)
    enemyFaction := factionMgr.CreateFaction("Bandits", false)

    // Create squads (using existing squad system)
    playerSquad := CreatePlayerSquad(manager)
    enemySquad := CreateEnemySquad(manager)

    // Place squads on map
    factionMgr.AddSquadToFaction(playerFaction, playerSquad, coords.LogicalPosition{X: 5, Y: 5})
    factionMgr.AddSquadToFaction(enemyFaction, enemySquad, coords.LogicalPosition{X: 20, Y: 20})

    // Initialize combat
    turnMgr.InitializeCombat([]ecs.EntityID{playerFaction, enemyFaction})

    // Combat is now ready
    currentFaction := turnMgr.GetCurrentFaction()
    fmt.Printf("Combat started! Current turn: %d\n", currentFaction)
}
```

---

### Example 4: Player Turn Execution

```go
func ExecutePlayerTurn(squadID ecs.EntityID, moveSys *combat.MovementSystem, combatSys *combat.CombatActionSystem) {
    // Get valid movement tiles
    validMoves := moveSys.GetValidMovementTiles(squadID)

    // Player chooses destination (from UI)
    targetPos := coords.LogicalPosition{X: 10, Y: 10}

    // Validate and execute movement
    if contains(validMoves, targetPos) {
        err := moveSys.MoveSquad(squadID, targetPos)
        if err != nil {
            fmt.Printf("Movement failed: %v\n", err)
            return
        }
    }

    // Get valid attack targets
    targets := combatSys.GetSquadsInRange(squadID)

    // Player chooses target (from UI)
    if len(targets) > 0 {
        targetSquad := targets[0]

        // Execute attack
        err := combatSys.ExecuteAttackAction(squadID, targetSquad)
        if err != nil {
            fmt.Printf("Attack failed: %v\n", err)
            return
        }
    }

    fmt.Println("Turn complete!")
}
```

---

### Example 5: Victory Condition Check

```go
// combat/victory.go

package combat

import (
    "game_main/common"
    "github.com/bytearena/ecs"
)

// VictoryCondition interface for extensible win conditions
type VictoryCondition interface {
    CheckVictory(manager *common.EntityManager) (won bool, winnerID ecs.EntityID)
}

// DefaultVictoryCondition checks if only one faction has squads remaining
type DefaultVictoryCondition struct{}

func (dvc *DefaultVictoryCondition) CheckVictory(manager *common.EntityManager) (bool, ecs.EntityID) {
    // Get all factions
    var factionIDs []ecs.EntityID
    for _, result := range manager.World.Query(FactionTag) {
        factionData := common.GetComponentType[*FactionData](result.Entity, FactionComponent)
        factionIDs = append(factionIDs, factionData.FactionID)
    }

    // Count factions with squads
    var activeFactions []ecs.EntityID
    for _, factionID := range factionIDs {
        squads := GetSquadsForFaction(factionID, manager)
        if len(squads) > 0 {
            activeFactions = append(activeFactions, factionID)
        }
    }

    // Victory if only one faction remains
    if len(activeFactions) == 1 {
        return true, activeFactions[0]
    }

    return false, 0
}
```

---

## SUMMARY

This implementation guide provides:

### Architecture
- 4 new ECS components for factions, turns, actions, positions
- 4 new systems for turns, movement, combat, factions
- 2 attribute extensions for movement speed and attack range
- Complete helper query functions
- 7 implementation phases (40-56 hours total)

### Integration
- Uses existing ExecuteSquadAttack() with range validation
- Leverages GlobalPositionSystem for O(1) collision detection
- Integrates with existing squad queries and attributes
- Query-based relationships (no stored entity pointers)

### Extensibility
- Stub interfaces for skills, magic, victory conditions
- Support for Player vs AI, PvP, AI vs AI
- Easy to add new actions, abilities, faction types

### Best Practices
- Pure data components (no logic methods)
- System-based logic (proper separation of concerns)
- Query-based patterns (ECS compliance)
- Go standards (error handling, clear APIs)
- Performance (O(1) lookups where possible)

### Next Steps

1. **Phase 0**: Implement component registration and helper functions
2. **Phase 1**: Implement TurnManager with complete action lifecycle
3. **Phase 2**: Implement MovementSystem with collision detection
4. **Phase 3**: Implement CombatActionSystem with range validation
5. **Phase 4**: Verify action economy enforcement
6. **Phase 5**: Implement FactionManager with squad ownership
7. **Phase 6**: Add stub interfaces for future features
8. **Phase 7**: Implement UI/input controller layer

---

**END OF IMPLEMENTATION GUIDE**
