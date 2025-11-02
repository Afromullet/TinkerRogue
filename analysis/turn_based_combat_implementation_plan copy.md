









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
// Constructor


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

This file contains all helper query functions for entity lookup, faction relationships, action state management, and utility functions. These are essential for the ECS query-based architecture.

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

// FindSquadByID finds a squad entity by its ID
// Note: This function doesn't exist in the codebase yet - add it to squads/squadqueries.go
// or use it from combat/queries.go
func FindSquadByID(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
    // Import squads package: "game_main/squads"
    for _, result := range manager.World.Query(squads.SquadTag) {
        if result.Entity.GetID() == squadID {
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
    └── turn_based_combat_implementation_plan.md  # This file
```

---

## GAME INITIALIZATION

### Integration into Game Setup

The combat system must be initialized explicitly during game setup, following the same pattern as the squad system. Add this to your `game_main/gameinit.go` or equivalent initialization file:

```go
// game_main/gameinit.go

import (
    "game_main/combat"
    "game_main/common"
    "game_main/squads"
    // ... other imports
)

func InitializeGame() error {
    // Create entity manager
    manager := common.NewEntityManager()

    // Initialize core components (existing code)
    InitializeECS(manager)

    // Initialize squad system (existing code)
    if err := squads.InitializeSquadData(manager); err != nil {
        return fmt.Errorf("failed to initialize squads: %w", err)
    }

    // Initialize combat system (NEW)
    combat.InitializeCombatSystem(manager)

    // ... rest of game initialization

    return nil
}
```

### Initialization Order

**Critical**: Combat system initialization must happen **after** squad system initialization because combat depends on squad components (SquadTag, SquadComponent, etc.).

**Correct Order:**
1. Core ECS components (`InitializeECS`)
2. Squad system (`squads.InitializeSquadData`)
3. Combat system (`combat.InitializeCombatSystem`)
4. Other game systems

**Why This Matters:**
- Combat system queries use `squads.SquadTag` and `squads.SquadMemberTag`
- `FindSquadByID` and other helpers rely on squad components being registered
- Movement and combat actions need access to unit attributes

### Testing Initialization

Verify combat system is initialized correctly:

```go
func TestCombatInitialization(t *testing.T) {
    manager := common.NewEntityManager()

    // Initialize dependencies
    squads.InitializeSquadData(manager)
    combat.InitializeCombatSystem(manager)

    // Verify components exist
    assert.NotNil(t, combat.FactionComponent)
    assert.NotNil(t, combat.TurnStateComponent)
    assert.NotNil(t, combat.ActionStateComponent)
    assert.NotNil(t, combat.MapPositionComponent)

    // Verify tags are registered
    _, ok := manager.Tags["faction"]
    assert.True(t, ok, "faction tag should be registered")

    _, ok = manager.Tags["turnstate"]
    assert.True(t, ok, "turnstate tag should be registered")
}
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
TestGetAttackingUnits_FiltersByUnitRange
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

## SUMMARY

This implementation plan provides:

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

**END OF IMPLEMENTATION PLAN**
