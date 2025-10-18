# Turn-Based Combat System - Gap Analysis & Implementation Specifications

**Version:** 2.0
**Date:** 2025-10-18
**Analysis Type:** Comprehensive Gap Analysis with Detailed Specifications
**Analyzed Plan:** turn_based_combat_plan.md v1.0 (2025-10-17)

---

## TABLE OF CONTENTS

1. [Executive Summary](#executive-summary)
2. [Critical Gaps Identified](#critical-gaps-identified)
3. [Detailed Gap Analysis](#detailed-gap-analysis)
4. [Missing Implementation Specifications](#missing-implementation-specifications)
5. [Integration Issues & Solutions](#integration-issues--solutions)
6. [Architectural Concerns](#architectural-concerns)
7. [Recommended Implementation Revisions](#recommended-implementation-revisions)
8. [Code Examples for Missing Pieces](#code-examples-for-missing-pieces)
9. [Testing Strategy Gaps](#testing-strategy-gaps)
10. [Risk Assessment](#risk-assessment)
11. [Next Steps](#next-steps)

---

## EXECUTIVE SUMMARY

### Overall Assessment

The existing turn_based_combat_plan.md (v1.0) provides a **solid foundation** with well-designed ECS components and system architecture. However, critical implementation details are **missing or underspecified** in several key areas that will cause blockers during development.

**Completeness Score: 70%**

### What's Working Well

✅ **Strong architectural foundation**
- Pure data components (FactionData, TurnStateData, ActionStateData, MapPositionData)
- Query-based relationship patterns
- Clear separation of concerns

✅ **Good integration planning**
- Uses existing ExecuteSquadAttack()
- Leverages PositionSystem for O(1) lookups
- Follows squad system ECS patterns

✅ **Well-structured phases**
- 6 clear implementation phases
- Reasonable effort estimates (28-40 hours)
- Logical dependency ordering

### Critical Gaps Requiring Immediate Attention

🔴 **CRITICAL GAP 1: Component Registration Missing**
- No component/tag registration code provided
- No init() function shown
- Will cause runtime panics without this

🔴 **CRITICAL GAP 2: Squad-to-Map Position Integration Undefined**
- How do squads get placed on the map initially?
- How does MapPositionData relate to PositionSystem spatial grid?
- What happens when a squad moves - which system updates which data structure?

🔴 **CRITICAL GAP 3: Distance Calculation Ambiguity**
- Plan uses Chebyshev distance (8-directional)
- Requirements unclear if movement is 8-directional or 4-directional
- Pathfinding not addressed at all
- Movement cost (tiles) vs actual path unclear

🔴 **CRITICAL GAP 4: Query Helper Functions Undefined**
- Functions like `findEntityByID()`, `getFactionOwner()`, `isSquad()` referenced but not implemented
- No specification for these critical helper functions

🔴 **CRITICAL GAP 5: Action State Lifecycle Incomplete**
- When is MovementRemaining initialized?
- How does partial movement work?
- Can a squad move 2 tiles, attack, then move 1 more tile?

🔴 **CRITICAL GAP 6: Map Integration Missing**
- No specification for map terrain interaction
- No pathfinding or obstacle avoidance
- No vision/fog of war consideration
- No spawn point system

🔴 **CRITICAL GAP 7: UI/Input Layer Undefined**
- How does player select squads?
- How does player choose movement destination?
- How does player choose attack target?
- No event system or command pattern specified

🔴 **CRITICAL GAP 8: Partial Squad Attacks Underspecified**
- "Only units with enemy in range will attack" mentioned but not implemented
- How to filter attacking units by individual unit range?
- Does this require modifying ExecuteSquadAttack()?

---

## DETAILED GAP ANALYSIS

### Gap Category 1: Component Registration & Initialization

**Problem:**
The plan defines four new components (FactionData, TurnStateData, ActionStateData, MapPositionData) but never shows how to register them with the ECS manager.

**Impact:** **BLOCKER**
Without component registration, the code will panic at runtime when trying to use components.

**What's Missing:**

```go
// combat/components.go - MISSING INIT FUNCTION

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

func init() {
    // THIS IS COMPLETELY MISSING FROM THE PLAN
    FactionComponent = ecs.NewComponent()
    TurnStateComponent = ecs.NewComponent()
    ActionStateComponent = ecs.NewComponent()
    MapPositionComponent = ecs.NewComponent()

    FactionTag = ecs.BuildTag(FactionComponent)
    TurnStateTag = ecs.BuildTag(TurnStateComponent)
    ActionStateTag = ecs.BuildTag(ActionStateComponent)
    MapPositionTag = ecs.BuildTag(MapPositionComponent)
}
```

**Required Specification:**

1. Complete component registration code
2. Tag building for all components
3. Example of creating entities with these components
4. How to query these components using the tags

---

### Gap Category 2: Squad-Map Position Integration

**Problem:**
Two position systems exist but their relationship is undefined:
1. `MapPositionData` (new combat package component)
2. `PositionSystem` spatial grid (existing systems package)

**Impact:** **CRITICAL**
Unclear which system is the source of truth for squad positions. Risk of desynchronization.

**Questions Unanswered:**

1. Does `MapPositionData.Position` duplicate `PositionSystem` data?
2. When a squad moves, do we update both systems?
3. Does `PositionSystem` track squad entities or individual unit entities?
4. How do we handle squad entities vs unit entities in position tracking?

**Current Plan Shows:**

```go
type MapPositionData struct {
    SquadID   ecs.EntityID          // Squad entity
    Position  coords.LogicalPosition // Current tile position
    FactionID ecs.EntityID          // Faction owner
}
```

**But Doesn't Specify:**

- Is the squad entity itself registered in PositionSystem?
- Or are only individual units registered?
- If both, how do we prevent conflicts?

**Recommended Specification:**

```go
// OPTION A: MapPositionData is canonical, PositionSystem is just an index
//
// Pros: Single source of truth (MapPositionData)
// Cons: Requires querying MapPositionData for every position lookup

func (ms *MovementSystem) MoveSquad(squadID ecs.EntityID, targetPos coords.LogicalPosition) error {
    // Step 1: Update MapPositionData (canonical source)
    mapPosEntity := findMapPositionEntity(squadID, ms.manager)
    mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
    oldPos := mapPos.Position
    mapPos.Position = targetPos

    // Step 2: Update PositionSystem spatial index
    ms.posSystem.MoveEntity(squadID, oldPos, targetPos)

    return nil
}

// OPTION B: PositionSystem is canonical, MapPositionData is derived
//
// Pros: PositionSystem already exists and works
// Cons: MapPositionData becomes redundant for position storage

func (ms *MovementSystem) MoveSquad(squadID ecs.EntityID, targetPos coords.LogicalPosition) error {
    // Step 1: Get current position from PositionSystem
    oldPos := ms.posSystem.GetSquadPosition(squadID)

    // Step 2: Move in PositionSystem (canonical source)
    ms.posSystem.MoveEntity(squadID, oldPos, targetPos)

    // Step 3: Sync MapPositionData for faction lookup
    // (MapPositionData only stores FactionID, position is queried from PositionSystem)
    return nil
}
```

**Missing from Plan:**

1. Which option to use (canonical source decision)
2. How to initialize squad position when added to faction
3. How to handle squad entity vs unit entity positions
4. Collision detection implementation details

---

### Gap Category 3: Movement & Distance Calculation

**Problem:**
The plan uses `ChebyshevDistance` but doesn't specify:
- Movement grid type (4-directional vs 8-directional)
- Pathfinding algorithm
- Movement cost calculation
- Obstacle avoidance

**Impact:** **HIGH**
Without pathfinding, squads can't navigate around obstacles or other squads.

**Current Plan Shows:**

```go
distance := attackerPos.ChebyshevDistance(&defenderPos)
```

**What's Missing:**

1. **Movement Type Specification:**
   - Can squads move diagonally?
   - Does diagonal movement cost 1 tile or 1.4 tiles?
   - Is movement 4-directional (cardinal only) or 8-directional?

2. **Pathfinding:**
   - A* algorithm implementation?
   - How to calculate actual path length?
   - How to avoid obstacles?

3. **Movement Validation:**
   - What if no path exists?
   - What if path is longer than movement speed?
   - Can squad move partway along a path?

**Required Specification:**

```go
// combat/movementsystem.go - MISSING PATHFINDING

// GetValidMovementTiles returns all tiles squad can reach this turn
// CURRENT PLAN: This function is declared but NOT IMPLEMENTED
func (ms *MovementSystem) GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition {
    // MISSING IMPLEMENTATION:
    // 1. Get squad position
    // 2. Get movement speed
    // 3. Run flood-fill or A* to find all reachable tiles
    // 4. Filter by collision (can't stop on enemies)
    // 5. Return valid tiles

    // QUESTION: Use existing pathfinding? If so, where is it?
    // QUESTION: 4-directional or 8-directional movement?
    // QUESTION: Movement cost per tile (always 1 or terrain-dependent)?
}

// CalculateMovementCost calculates tiles consumed by a movement
// COMPLETELY MISSING FROM PLAN
func (ms *MovementSystem) CalculateMovementCost(from, to coords.LogicalPosition) int {
    // Option 1: Chebyshev (8-directional, diagonal = 1 tile)
    // return from.ChebyshevDistance(&to)

    // Option 2: Manhattan (4-directional, diagonal = 2 tiles)
    // return from.ManhattanDistance(&to)

    // Option 3: Pathfinding (actual path length)
    // path := astar.FindPath(from, to, ms.collisionCheck)
    // return len(path)

    // WHICH ONE TO USE???
}
```

**Decision Required:**

Choose one of these movement models:

| Model | Movement | Distance Metric | Pathfinding Required? |
|-------|----------|-----------------|----------------------|
| **Simple 8-Dir** | Can move diagonally, cost = 1 tile | Chebyshev | No (just check Chebyshev distance) |
| **Simple 4-Dir** | Cardinal only, no diagonals | Manhattan | No (just check Manhattan distance) |
| **Pathfinding** | Any route, obstacles matter | Path length | Yes (A* required) |

**Recommendation:**
Start with **Simple 8-Dir** for MVP, add pathfinding in Phase 2.5 (between movement and combat).

---

### Gap Category 4: Query Helper Functions

**Problem:**
The plan references many helper functions that are never defined or implemented.

**Impact:** **BLOCKER**
Code will not compile without these functions.

**Undefined Functions Referenced in Plan:**

```go
// Referenced in turn_based_combat_plan.md but NEVER DEFINED:

findEntityByID(entityID ecs.EntityID, manager *EntityManager) *ecs.Entity
getFactionOwner(squadID ecs.EntityID, manager *EntityManager) ecs.EntityID
isSquad(entityID ecs.EntityID, manager *EntityManager) bool
canSquadAct(squadID ecs.EntityID, manager *EntityManager) bool
markSquadAsActed(squadID ecs.EntityID, manager *EntityManager)
removeSquadFromMap(squadID ecs.EntityID, manager *EntityManager)
getSquadMapPosition(squadID ecs.EntityID, manager *EntityManager) (coords.LogicalPosition, error)
findTurnStateEntity(manager *EntityManager) *ecs.Entity
findFactionByID(factionID ecs.EntityID, manager *EntityManager) *ecs.Entity
findSquadByID(squadID ecs.EntityID, manager *EntityManager) *ecs.Entity
findMapPositionEntity(squadID ecs.EntityID, manager *EntityManager) *ecs.Entity
logCombatResult(result *CombatResult)
shuffleFactionOrder(factionIDs []ecs.EntityID)
GetSquadsForFaction(factionID ecs.EntityID, manager *EntityManager) []ecs.EntityID
combatActive(manager *EntityManager) bool
```

**Required Specification:**

These functions MUST be implemented. Here are the critical ones:

```go
// combat/queries.go - NEW FILE NEEDED

package combat

import (
    "fmt"
    "game_main/common"
    "game_main/coords"
    "github.com/bytearena/ecs"
)

// findEntityByID searches all entities for a matching ID
// This is a generic helper used by all combat systems
func findEntityByID(entityID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
    // QUESTION: Does bytearena/ecs have a GetEntityByID method?
    // If not, we need to cache entities or search all queries

    // Fallback: Search all entities (expensive)
    for _, result := range manager.World.Query(ecs.BuildTag()) {
        if result.Entity.GetID() == entityID {
            return result.Entity
        }
    }
    return nil
}

// getFactionOwner returns the faction that owns a squad
func getFactionOwner(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
    for _, result := range manager.World.Query(MapPositionTag) {
        mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)
        if mapPos.SquadID == squadID {
            return mapPos.FactionID
        }
    }
    return 0 // No owner found
}

// getSquadMapPosition returns the current map position of a squad
func getSquadMapPosition(squadID ecs.EntityID, manager *common.EntityManager) (coords.LogicalPosition, error) {
    for _, result := range manager.World.Query(MapPositionTag) {
        mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)
        if mapPos.SquadID == squadID {
            return mapPos.Position, nil
        }
    }
    return coords.LogicalPosition{}, fmt.Errorf("squad %d not found on map", squadID)
}

// canSquadAct checks if a squad has available actions this turn
func canSquadAct(squadID ecs.EntityID, manager *common.EntityManager) bool {
    for _, result := range manager.World.Query(ActionStateTag) {
        actionState := common.GetComponentType[*ActionStateData](result.Entity, ActionStateComponent)
        if actionState.SquadID == squadID {
            // Squad can act if it hasn't performed its combat action yet
            return !actionState.HasActed
        }
    }
    return false // No action state found (shouldn't happen)
}

// markSquadAsActed marks a squad as having used its combat action
func markSquadAsActed(squadID ecs.EntityID, manager *common.EntityManager) {
    for _, result := range manager.World.Query(ActionStateTag) {
        actionState := common.GetComponentType[*ActionStateData](result.Entity, ActionStateComponent)
        if actionState.SquadID == squadID {
            actionState.HasActed = true
            return
        }
    }
}

// isSquad checks if an entity is a squad entity (vs terrain, item, etc.)
func isSquad(entityID ecs.EntityID, manager *common.EntityManager) bool {
    entity := findEntityByID(entityID, manager)
    if entity == nil {
        return false
    }
    // QUESTION: How to identify squad entities?
    // Option 1: Check for SquadComponent
    // Option 2: Check for MapPositionComponent
    // Option 3: Check entity ID against known squad IDs

    // Assuming squads have MapPositionComponent:
    return entity.HasComponent(MapPositionComponent)
}

// Additional missing helpers...
```

**CRITICAL QUESTION:**
Does bytearena/ecs provide an entity lookup by ID? If not, we need to implement entity caching.

---

### Gap Category 5: Action State Lifecycle

**Problem:**
`ActionStateData.MovementRemaining` field exists but its lifecycle is undefined.

**Current Plan Shows:**

```go
type ActionStateData struct {
    SquadID           ecs.EntityID
    HasMoved          bool
    HasActed          bool
    MovementRemaining int  // ← When is this set? When is it decremented?
}
```

**Questions Unanswered:**

1. When is `MovementRemaining` initialized?
   - In `TurnManager.InitializeCombat()`? (It's set to 0 there)
   - In `TurnManager.ResetSquadActions()`? (Not shown)
   - In `MovementSystem` on first movement?

2. How does partial movement work?
   - Can squad move 2 tiles, attack, then move 1 more tile?
   - Or is movement a single atomic action?

3. When is `HasMoved` set to true?
   - After ANY movement?
   - Only after squad uses ALL movement?
   - What if squad moves 1 tile of 3 available?

**Required Specification:**

```go
// combat/turnmanager.go - MISSING SPECIFICATION

func (tm *TurnManager) ResetSquadActions(factionID ecs.EntityID) error {
    squads := GetSquadsForFaction(factionID, tm.manager)

    for _, squadID := range squads {
        // Find or create ActionStateData
        for _, result := range tm.manager.World.Query(ActionStateTag) {
            actionState := common.GetComponentType[*ActionStateData](result.Entity, ActionStateComponent)
            if actionState.SquadID == squadID {
                // Reset flags
                actionState.HasMoved = false
                actionState.HasActed = false

                // MISSING: Initialize MovementRemaining
                // Get squad's movement speed and set it
                moveSys := NewMovementSystem(tm.manager)
                actionState.MovementRemaining = moveSys.GetSquadMovementSpeed(squadID)

                break
            }
        }
    }

    return nil
}
```

**Decision Required:**

Choose one action model:

| Model | Movement Behavior | HasMoved Trigger | MovementRemaining |
|-------|-------------------|------------------|-------------------|
| **Atomic Move** | Squad moves to destination in one action | Set to true after any move | Not used (just bool) |
| **Partial Move** | Squad can move tiles individually | Set to true when MovementRemaining = 0 | Decremented per tile moved |
| **Move Then Act** | Squad moves full distance, then acts | Set to true after first move | Fully consumed on first move |

**Recommendation:**
Use **Move Then Act** for simplicity (matches FFT, Fire Emblem). Partial movement adds UI complexity.

---

### Gap Category 6: Map Integration

**Problem:**
The plan assumes a map exists but doesn't specify how to integrate with it.

**Missing Specifications:**

1. **Terrain System:**
   - How to check if a tile is walkable?
   - How to get terrain movement cost?
   - How to handle terrain effects (water, forest, etc.)?

2. **Spawn System:**
   - Where do squads start on the map?
   - How to place squads at combat start?
   - What if spawn points are occupied?

3. **Vision System:**
   - Do factions have fog of war?
   - Can squads attack unseen enemies?
   - How is vision calculated?

4. **Map Boundaries:**
   - How to prevent squads from moving off-map?
   - What are map dimensions?
   - Are they fixed or dynamic per encounter?

**Required Specification:**

```go
// combat/mapintegration.go - NEW FILE NEEDED

package combat

import (
    "game_main/common"
    "game_main/coords"
    "github.com/bytearena/ecs"
)

// MapConfig defines the combat map parameters
type MapConfig struct {
    Width        int                    // Map width in tiles
    Height       int                    // Map height in tiles
    SpawnPoints  map[ecs.EntityID]coords.LogicalPosition // Faction spawn positions
    TerrainGrid  [][]TerrainType        // Terrain data (if needed)
}

// TerrainType defines tile properties
type TerrainType int

const (
    TerrainNormal TerrainType = iota
    TerrainWall      // Blocks movement
    TerrainWater     // Slows movement
    TerrainForest    // Provides cover bonus
)

// IsWalkable checks if a position is walkable
func (mc *MapConfig) IsWalkable(pos coords.LogicalPosition) bool {
    // Check bounds
    if pos.X < 0 || pos.X >= mc.Width || pos.Y < 0 || pos.Y >= mc.Height {
        return false
    }

    // Check terrain (if using terrain grid)
    if mc.TerrainGrid != nil {
        terrain := mc.TerrainGrid[pos.Y][pos.X]
        return terrain != TerrainWall
    }

    return true
}

// PlaceSquadAtSpawn places a squad at its faction's spawn point
func PlaceSquadAtSpawn(factionID, squadID ecs.EntityID, mapConfig *MapConfig, manager *common.EntityManager) error {
    spawnPos, ok := mapConfig.SpawnPoints[factionID]
    if !ok {
        return fmt.Errorf("no spawn point defined for faction %d", factionID)
    }

    // Create MapPositionData for this squad
    mapPosEntity := manager.World.NewEntity()
    mapPosEntity.AddComponent(MapPositionComponent, &MapPositionData{
        SquadID:   squadID,
        Position:  spawnPos,
        FactionID: factionID,
    })

    // Register in PositionSystem for collision detection
    // QUESTION: How to get PositionSystem reference here?
    // Need GlobalPositionSystem or pass it as parameter?

    return nil
}
```

**CRITICAL DECISION:**
Do we integrate with existing worldmap package or create combat-specific maps?

---

### Gap Category 7: UI/Input Layer

**Problem:**
The plan focuses on backend logic but provides NO specification for player interaction.

**Missing Specifications:**

1. **Squad Selection:**
   - How does player click on a squad?
   - How to show which squads belong to player?
   - How to highlight selected squad?

2. **Movement Input:**
   - How does player choose destination tile?
   - How to show valid movement tiles?
   - How to preview movement path?

3. **Attack Target Selection:**
   - How does player choose attack target?
   - How to show valid targets?
   - How to display attack range?

4. **Turn Management:**
   - How does player end turn?
   - How to show whose turn it is?
   - How to show action state (moved/acted)?

**Required Specification:**

```go
// combat/combatcontroller.go - NEW FILE NEEDED

package combat

import (
    "game_main/common"
    "game_main/coords"
    "github.com/bytearena/ecs"
)

// CombatController bridges UI input to combat systems
// This is the missing link between player actions and combat logic
type CombatController struct {
    turnMgr    *TurnManager
    moveSys    *MovementSystem
    combatSys  *CombatActionSystem
    factionMgr *FactionManager

    selectedSquad   ecs.EntityID          // Currently selected squad
    hoveredPosition coords.LogicalPosition // Mouse cursor position
    validMoves      []coords.LogicalPosition // Cached valid tiles
    validTargets    []ecs.EntityID        // Cached valid attack targets
}

// HandleSquadSelection processes player clicking on a squad
func (cc *CombatController) HandleSquadSelection(clickedPos coords.LogicalPosition) error {
    // 1. Get current faction turn
    currentFaction := cc.turnMgr.GetCurrentFaction()

    // 2. Check if player controls this faction
    // QUESTION: How to check if faction is player-controlled?

    // 3. Get squad at clicked position
    squadID := cc.getSquadAtPosition(clickedPos)
    if squadID == 0 {
        return fmt.Errorf("no squad at position %v", clickedPos)
    }

    // 4. Verify squad belongs to current faction
    squadFaction := getFactionOwner(squadID, cc.manager)
    if squadFaction != currentFaction {
        return fmt.Errorf("cannot select enemy squad")
    }

    // 5. Select squad
    cc.selectedSquad = squadID

    // 6. Calculate valid moves and targets
    cc.validMoves = cc.moveSys.GetValidMovementTiles(squadID)
    cc.validTargets = cc.combatSys.GetSquadsInRange(squadID)

    return nil
}

// HandleMoveCommand processes player clicking on movement destination
func (cc *CombatController) HandleMoveCommand(targetPos coords.LogicalPosition) error {
    if cc.selectedSquad == 0 {
        return fmt.Errorf("no squad selected")
    }

    // Validate target is in valid moves
    if !contains(cc.validMoves, targetPos) {
        return fmt.Errorf("invalid movement destination")
    }

    // Execute movement
    return cc.moveSys.MoveSquad(cc.selectedSquad, targetPos)
}

// HandleAttackCommand processes player clicking on attack target
func (cc *CombatController) HandleAttackCommand(targetSquadID ecs.EntityID) error {
    if cc.selectedSquad == 0 {
        return fmt.Errorf("no squad selected")
    }

    // Validate target is in range
    if !containsEntity(cc.validTargets, targetSquadID) {
        return fmt.Errorf("target not in range")
    }

    // Execute attack
    return cc.combatSys.ExecuteAttackAction(cc.selectedSquad, targetSquadID)
}

// HandleEndTurn processes player ending their turn
func (cc *CombatController) HandleEndTurn() error {
    // Clear selection
    cc.selectedSquad = 0
    cc.validMoves = nil
    cc.validTargets = nil

    // Advance turn
    return cc.turnMgr.EndTurn()
}
```

**CRITICAL GAP:**
No rendering/visualization layer specified. How to draw:
- Squad positions on map?
- Valid movement tiles?
- Attack range indicators?
- Turn order display?

---

### Gap Category 8: Partial Squad Attacks

**Problem:**
Requirement states "Only units with enemy in range will attack" but implementation is unclear.

**Requirement:**
> Can attack X tiles away where X = attack range
> Only units with attack range > 1 will attack from distance
> Partial squad attacks (some units in range, some not)

**Current Plan Shows:**

```go
func (cas *CombatActionSystem) GetSquadAttackRange(squadID ecs.EntityID) int {
    unitIDs := squads.GetUnitIDsInSquad(squadID, cas.manager)

    maxRange := 1
    for _, unitID := range unitIDs {
        // Get max range across all units
        unitRange := attr.GetAttackRange()
        if unitRange > maxRange {
            maxRange = unitRange
        }
    }

    return maxRange // Squad can attack at max range
}
```

**But Doesn't Show:**

How to filter which units actually attack when target is at range 3 but some units have range 1?

**Missing Implementation:**

```go
// combat/combatactionsystem.go - MISSING METHOD

// GetAttackingUnits returns unit IDs that can attack target from current position
// Referenced in plan but NEVER IMPLEMENTED
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

        // Check if this unit can reach the target
        unitRange := attr.GetAttackRange()
        if distance <= unitRange {
            attackingUnits = append(attackingUnits, unitID)
        }
    }

    return attackingUnits
}
```

**CRITICAL QUESTION:**
Does this require modifying `squads.ExecuteSquadAttack()` to accept a filtered unit list?

**Current ExecuteSquadAttack signature:**
```go
func ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID, manager *EntityManager) *CombatResult
```

**Proposed new signature:**
```go
func ExecuteSquadAttackFiltered(
    attackerSquadID,
    defenderSquadID ecs.EntityID,
    attackingUnitIDs []ecs.EntityID, // FILTER: only these units attack
    manager *EntityManager
) *CombatResult
```

**Or alternative approach:**
Temporarily disable units that are out of range before calling ExecuteSquadAttack?

```go
func (cas *CombatActionSystem) ExecuteAttackAction(attackerID, defenderID ecs.EntityID) error {
    // Get units in range
    attackingUnits := cas.GetAttackingUnits(attackerID, defenderID)

    // Temporarily disable units out of range
    allUnits := squads.GetUnitIDsInSquad(attackerID, cas.manager)
    disabledUnits := []ecs.EntityID{}

    for _, unitID := range allUnits {
        if !contains(attackingUnits, unitID) {
            // Temporarily mark as unable to act
            unit := squads.FindUnitByID(unitID, cas.manager)
            attr := common.GetAttributes(unit)
            if attr.CanAct {
                attr.CanAct = false
                disabledUnits = append(disabledUnits, unitID)
            }
        }
    }

    // Execute attack (only CanAct units will participate)
    result := squads.ExecuteSquadAttack(attackerID, defenderID, cas.manager)

    // Re-enable disabled units
    for _, unitID := range disabledUnits {
        unit := squads.FindUnitByID(unitID, cas.manager)
        attr := common.GetAttributes(unit)
        attr.CanAct = true
    }

    return nil
}
```

This is a HACKY workaround. Better to add a filtered attack method to squad combat system.

---

## MISSING IMPLEMENTATION SPECIFICATIONS

### 1. Component Tag Constants

**File:** `combat/components.go`

```go
package combat

import "github.com/bytearena/ecs"

// Component and tag variables (MUST be initialized in init())
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
// CRITICAL: This function is COMPLETELY MISSING from the original plan
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

// Component data structures (already defined in plan)
type FactionData struct {
    FactionID          ecs.EntityID
    Name               string
    Mana               int
    MaxMana            int
    IsPlayerControlled bool
    SquadIDs           []ecs.EntityID // NOTE: Plan suggests NOT storing this, but it's here anyway
}

type TurnStateData struct {
    CurrentRound     int
    TurnOrder        []ecs.EntityID
    CurrentTurnIndex int
    CombatActive     bool
}

type ActionStateData struct {
    SquadID           ecs.EntityID
    HasMoved          bool
    HasActed          bool
    MovementRemaining int
}

type MapPositionData struct {
    SquadID   ecs.EntityID
    Position  coords.LogicalPosition
    FactionID ecs.EntityID
}
```

---

### 2. Query Helper Functions

**File:** `combat/queries.go` (NEW FILE)

```go
package combat

import (
    "fmt"
    "game_main/common"
    "game_main/coords"
    "game_main/squads"
    "github.com/bytearena/ecs"
)

// ========================================
// ENTITY LOOKUP HELPERS
// ========================================

// findEntityByID finds an entity by its ID
// WARNING: This is O(n) without entity caching
// Consider implementing entity ID map for O(1) lookup
func findEntityByID(entityID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
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
    // Check if entity has MapPositionComponent (all squads on map have this)
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
// Called when squad is destroyed
func removeSquadFromMap(squadID ecs.EntityID, manager *common.EntityManager) error {
    // Find and remove MapPositionData
    mapPosEntity := findMapPositionEntity(squadID, manager)
    if mapPosEntity == nil {
        return fmt.Errorf("squad %d not on map", squadID)
    }

    // Get position before removal for PositionSystem cleanup
    mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
    position := mapPos.Position

    // Remove from ECS
    manager.World.RemoveEntity(mapPosEntity)

    // TODO: Remove from PositionSystem spatial grid
    // Need access to GlobalPositionSystem or pass PositionSystem as parameter
    // GlobalPositionSystem.RemoveEntity(squadID, position)

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
    // TODO: Implement logging/event system
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

### 3. Movement System Complete Implementation

**File:** `combat/movementsystem.go`

```go
package combat

import (
    "fmt"
    "game_main/common"
    "game_main/coords"
    "game_main/squads"
    "game_main/systems"
    "github.com/bytearena/ecs"
)

type MovementSystem struct {
    manager   *common.EntityManager
    posSystem *systems.PositionSystem
}

func NewMovementSystem(manager *common.EntityManager, posSystem *systems.PositionSystem) *MovementSystem {
    return &MovementSystem{
        manager:   manager,
        posSystem: posSystem,
    }
}

// GetSquadMovementSpeed returns tiles per turn (slowest unit)
func (ms *MovementSystem) GetSquadMovementSpeed(squadID ecs.EntityID) int {
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
        if attr == nil {
            continue
        }

        // Use new MovementSpeed attribute
        speed := attr.GetMovementSpeed()

        if speed < minSpeed {
            minSpeed = speed
        }
    }

    if minSpeed == 999 {
        return 3 // Default if no valid units
    }

    return minSpeed
}

// GetSquadPosition returns current map position
func (ms *MovementSystem) GetSquadPosition(squadID ecs.EntityID) (coords.LogicalPosition, error) {
    return getSquadMapPosition(squadID, ms.manager)
}

// CanMoveTo checks if squad can legally move to target position
func (ms *MovementSystem) CanMoveTo(squadID ecs.EntityID, targetPos coords.LogicalPosition) bool {
    // Check 1: Is position in bounds?
    // TODO: Need map bounds from MapConfig
    // For now, assume all positions are valid

    // Check 2: Is tile occupied?
    occupyingID := ms.posSystem.GetEntityIDAt(targetPos)

    if occupyingID == 0 {
        return true // Empty tile - can move
    }

    // Check 3: If occupied, is it a squad?
    if !isSquad(occupyingID, ms.manager) {
        // Occupied by terrain/obstacle
        return false
    }

    // Check 4: If occupied by squad, is it friendly?
    occupyingFaction := getFactionOwner(occupyingID, ms.manager)
    squadFaction := getFactionOwner(squadID, ms.manager)

    // Can pass through friendlies, NOT enemies
    return occupyingFaction == squadFaction
}

// MoveSquad executes movement and updates position
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

    // Calculate movement cost
    movementCost := ms.calculateMovementCost(currentPos, targetPos)

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

    // Update MapPositionData
    mapPosEntity := findMapPositionEntity(squadID, ms.manager)
    if mapPosEntity == nil {
        return fmt.Errorf("squad not on map")
    }

    mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
    mapPos.Position = targetPos

    // Update PositionSystem spatial grid
    ms.posSystem.MoveEntity(squadID, currentPos, targetPos)

    // Update action state
    decrementMovementRemaining(squadID, movementCost, ms.manager)
    markSquadAsMoved(squadID, ms.manager)

    return nil
}

// GetValidMovementTiles returns all tiles squad can reach this turn
// MISSING FROM ORIGINAL PLAN - CRITICAL IMPLEMENTATION
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
        return []coords.LogicalPosition{} // No movement left
    }

    // Simple flood-fill for valid tiles (no pathfinding)
    // This assumes 8-directional movement with Chebyshev distance

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

// calculateMovementCost calculates tiles consumed by a movement
// MISSING FROM ORIGINAL PLAN
func (ms *MovementSystem) calculateMovementCost(from, to coords.LogicalPosition) int {
    // Using Chebyshev distance (8-directional movement, diagonal = 1 tile)
    return from.ChebyshevDistance(&to)

    // Alternative: Manhattan distance (4-directional only)
    // return from.ManhattanDistance(&to)

    // Alternative: Pathfinding-based cost
    // path := astar.FindPath(from, to, ms.isWalkable)
    // return len(path)
}
```

---

## INTEGRATION ISSUES & SOLUTIONS

### Issue 1: Position System Dual Ownership

**Problem:**
Both `MapPositionData` and `PositionSystem` track squad positions. Risk of desynchronization.

**Solution:**

Make `PositionSystem` the **canonical source** for position data. Use `MapPositionData` ONLY for faction ownership.

**Revised MapPositionData:**

```go
type MapPositionData struct {
    SquadID   ecs.EntityID
    // Position  coords.LogicalPosition  // ← REMOVE THIS
    FactionID ecs.EntityID
}
```

**Get position from PositionSystem instead:**

```go
func getSquadMapPosition(squadID ecs.EntityID, posSystem *systems.PositionSystem) (coords.LogicalPosition, error) {
    // Query PositionSystem spatial grid (canonical source)
    for _, pos := range posSystem.GetOccupiedPositions() {
        if posSystem.GetEntityIDAt(pos) == squadID {
            return pos, nil
        }
    }
    return coords.LogicalPosition{}, fmt.Errorf("squad %d not found", squadID)
}
```

**OR keep Position in MapPositionData but make it derived/cached:**

```go
// Keep MapPositionData.Position but always sync from PositionSystem
func (ms *MovementSystem) MoveSquad(squadID ecs.EntityID, targetPos coords.LogicalPosition) error {
    // ... validation ...

    // Update PositionSystem (canonical source)
    ms.posSystem.MoveEntity(squadID, currentPos, targetPos)

    // Sync MapPositionData (derived/cached)
    mapPosEntity := findMapPositionEntity(squadID, ms.manager)
    mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
    mapPos.Position = targetPos // Keep in sync

    return nil
}
```

**Recommendation:** Use cached position approach for performance.

---

### Issue 2: Squad Entity vs Unit Entity Positioning

**Problem:**
Squad combat system tracks **individual unit entities** with positions. Turn-based system needs to track **squad entities** as single map tiles.

**Current State:**
- Individual units have GridPositionData (3x3 grid within squad)
- Squads need LogicalPosition (map tile)

**Solution:**

Squads and units use DIFFERENT position systems:

```
Squad Level (Map):
- Squad entity has MapPositionData (or PositionSystem tracks squad entity)
- Squad occupies ONE map tile (e.g., X=10, Y=5)

Unit Level (Formation):
- Unit entities have GridPositionData (row/col within squad)
- Units occupy cells in 3x3 grid relative to squad
```

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

Map View:
+---+---+---+---+---+
|   |   | S |   |   |  ← Squad S at (10, 5)
+---+---+---+---+---+
 8   9  10  11  12
```

**Implementation:**

```go
// Squads registered in PositionSystem
posSystem.AddEntity(squadID, coords.LogicalPosition{X: 10, Y: 5})

// Units NOT registered in PositionSystem (only GridPositionData for formation)
// Units are children of squad entity, not independent map entities
```

---

### Issue 3: Existing Squad Combat Integration

**Problem:**
`squads.ExecuteSquadAttack()` doesn't know about map positions or distance.

**Solution:**

Add range filtering BEFORE calling ExecuteSquadAttack:

```go
func (cas *CombatActionSystem) ExecuteAttackAction(attackerID, defenderID ecs.EntityID) error {
    // Step 1: Validate range
    attackerPos, _ := getSquadMapPosition(attackerID, cas.manager)
    defenderPos, _ := getSquadMapPosition(defenderID, cas.manager)
    distance := attackerPos.ChebyshevDistance(&defenderPos)

    // Step 2: Filter attacking units by range
    attackingUnits := cas.getUnitsInRange(attackerID, distance)

    // Step 3: Temporarily disable out-of-range units
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

    // Step 4: Execute attack (only CanAct=true units participate)
    result := squads.ExecuteSquadAttack(attackerID, defenderID, cas.manager)

    // Step 5: Re-enable disabled units
    for _, unitID := range disabledUnits {
        unit := squads.FindUnitByID(unitID, cas.manager)
        attr := common.GetAttributes(unit)
        attr.CanAct = true
    }

    // Step 6: Handle defeated squads
    if squads.IsSquadDestroyed(defenderID, cas.manager) {
        removeSquadFromMap(defenderID, cas.manager)
    }

    return nil
}

func (cas *CombatActionSystem) getUnitsInRange(squadID ecs.EntityID, distance int) []ecs.EntityID {
    unitIDs := squads.GetUnitIDsInSquad(squadID, cas.manager)
    var inRange []ecs.EntityID

    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, cas.manager)
        if unit == nil {
            continue
        }

        attr := common.GetAttributes(unit)
        unitRange := attr.GetAttackRange()

        if distance <= unitRange {
            inRange = append(inRange, unitID)
        }
    }

    return inRange
}
```

---

## ARCHITECTURAL CONCERNS

### Concern 1: Global PositionSystem Access

**Problem:**
MovementSystem needs PositionSystem reference, but how to access it?

**Options:**

1. **Pass as constructor parameter:**
```go
moveSys := NewMovementSystem(manager, globalPositionSystem)
```

2. **Global variable:**
```go
var GlobalPositionSystem *systems.PositionSystem
```

3. **Store in EntityManager:**
```go
type EntityManager struct {
    World        *ecs.Manager
    PositionSys  *systems.PositionSystem  // Add this
}
```

**Recommendation:** Use constructor parameter (option 1) for testability.

---

### Concern 2: Entity ID Lookup Performance

**Problem:**
`findEntityByID()` is O(n) without entity caching.

**Solution:**

Add entity ID map to EntityManager:

```go
type EntityManager struct {
    World      *ecs.Manager
    entityMap  map[ecs.EntityID]*ecs.Entity  // NEW: O(1) lookup
}

func (em *EntityManager) GetEntityByID(id ecs.EntityID) *ecs.Entity {
    return em.entityMap[id]
}

func (em *EntityManager) RegisterEntity(entity *ecs.Entity) {
    em.entityMap[entity.GetID()] = entity
}

func (em *EntityManager) UnregisterEntity(entity *ecs.Entity) {
    delete(em.entityMap, entity.GetID())
}
```

---

## RECOMMENDED IMPLEMENTATION REVISIONS

### Revision 1: Add Phase 0 - Foundation

**New Phase 0: Core Infrastructure (4-6 hours)**

Before implementing turn system, add:

1. Component registration (init() function)
2. Query helper functions (queries.go)
3. Entity ID map (EntityManager extension)
4. PositionSystem integration decision

---

### Revision 2: Split Phase 2 - Movement

**Revised Phase 2:**

**Phase 2A: Basic Movement (4-6 hours)**
- GetSquadMovementSpeed
- CanMoveTo (basic collision)
- MoveSquad (direct teleport)
- No pathfinding yet

**Phase 2B: Advanced Movement (4-6 hours)**
- GetValidMovementTiles (flood-fill)
- Pathfinding integration
- Movement preview
- Terrain integration

---

### Revision 3: Add Phase 6.5 - UI/Input

**New Phase 6.5: Player Input Layer (6-8 hours)**

After stubs, before full integration:

1. CombatController creation
2. Squad selection
3. Movement input
4. Attack targeting
5. Turn end command
6. Event system for UI updates

---

## CODE EXAMPLES FOR MISSING PIECES

### Example 1: Complete TurnManager with Missing Methods

```go
package combat

import (
    "fmt"
    "game_main/common"
    "github.com/bytearena/ecs"
    "math/rand/v2"
)

type TurnManager struct {
    manager *common.EntityManager
}

func NewTurnManager(manager *common.EntityManager) *TurnManager {
    return &TurnManager{manager: manager}
}

// InitializeCombat sets up turn order and combat state
func (tm *TurnManager) InitializeCombat(factionIDs []ecs.EntityID) error {
    // Randomize turn order
    turnOrder := make([]ecs.EntityID, len(factionIDs))
    copy(turnOrder, factionIDs)
    shuffleFactionOrder(turnOrder)

    // Create TurnStateData entity
    turnEntity := tm.manager.World.NewEntity()
    turnEntity.AddComponent(TurnStateComponent, &TurnStateData{
        CurrentRound:     1,
        TurnOrder:        turnOrder,
        CurrentTurnIndex: 0,
        CombatActive:     true,
    })

    // Create ActionStateData for all squads
    for _, factionID := range factionIDs {
        squads := GetSquadsForFaction(factionID, tm.manager)
        for _, squadID := range squads {
            tm.createActionStateForSquad(squadID)
        }
    }

    // Reset actions for first faction
    firstFaction := turnOrder[0]
    tm.ResetSquadActions(firstFaction)

    return nil
}

// createActionStateForSquad creates an ActionStateData entity for a squad
// MISSING FROM ORIGINAL PLAN
func (tm *TurnManager) createActionStateForSquad(squadID ecs.EntityID) {
    actionEntity := tm.manager.World.NewEntity()
    actionEntity.AddComponent(ActionStateComponent, &ActionStateData{
        SquadID:           squadID,
        HasMoved:          false,
        HasActed:          false,
        MovementRemaining: 0, // Will be set by ResetSquadActions
    })
}

// GetCurrentFaction returns the active faction's ID
func (tm *TurnManager) GetCurrentFaction() ecs.EntityID {
    turnEntity := findTurnStateEntity(tm.manager)
    if turnEntity == nil {
        return 0
    }

    turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)
    return turnState.TurnOrder[turnState.CurrentTurnIndex]
}

// EndTurn advances to the next faction's turn
func (tm *TurnManager) EndTurn() error {
    turnEntity := findTurnStateEntity(tm.manager)
    if turnEntity == nil {
        return fmt.Errorf("no active combat")
    }

    turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)

    // Advance turn index
    turnState.CurrentTurnIndex++

    // Wrap around
    if turnState.CurrentTurnIndex >= len(turnState.TurnOrder) {
        turnState.CurrentTurnIndex = 0
        turnState.CurrentRound++
    }

    // Reset actions for new faction
    newFactionID := turnState.TurnOrder[turnState.CurrentTurnIndex]
    tm.ResetSquadActions(newFactionID)

    return nil
}

// ResetSquadActions resets HasMoved/HasActed for all faction's squads
// REVISED WITH MISSING MovementRemaining INITIALIZATION
func (tm *TurnManager) ResetSquadActions(factionID ecs.EntityID) error {
    squads := GetSquadsForFaction(factionID, tm.manager)

    // Create MovementSystem to get squad speeds
    moveSys := NewMovementSystem(tm.manager, nil) // TODO: Need PositionSystem reference

    for _, squadID := range squads {
        actionEntity := findActionStateEntity(squadID, tm.manager)
        if actionEntity == nil {
            continue
        }

        actionState := common.GetComponentType[*ActionStateData](actionEntity, ActionStateComponent)

        // Reset flags
        actionState.HasMoved = false
        actionState.HasActed = false

        // CRITICAL FIX: Initialize MovementRemaining
        squadSpeed := moveSys.GetSquadMovementSpeed(squadID)
        actionState.MovementRemaining = squadSpeed
    }

    return nil
}

// IsSquadActivatable checks if a squad can act this turn
func (tm *TurnManager) IsSquadActivatable(squadID ecs.EntityID) bool {
    // Check 1: Is it this squad's faction's turn?
    currentFaction := tm.GetCurrentFaction()
    squadFaction := getFactionOwner(squadID, tm.manager)

    if currentFaction != squadFaction {
        return false
    }

    // Check 2: Does squad have actions remaining?
    return canSquadAct(squadID, tm.manager) || canSquadMove(squadID, tm.manager)
}

// GetCurrentRound returns the current round number
func (tm *TurnManager) GetCurrentRound() int {
    turnEntity := findTurnStateEntity(tm.manager)
    if turnEntity == nil {
        return 0
    }

    turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)
    return turnState.CurrentRound
}

// EndCombat marks combat as finished
func (tm *TurnManager) EndCombat() error {
    turnEntity := findTurnStateEntity(tm.manager)
    if turnEntity == nil {
        return fmt.Errorf("no active combat")
    }

    turnState := common.GetComponentType[*TurnStateData](turnEntity, TurnStateComponent)
    turnState.CombatActive = false

    return nil
}
```

---

## TESTING STRATEGY GAPS

### Gap 1: No Mock/Stub Specifications

**Problem:**
Plan mentions tests but doesn't show how to create test fixtures.

**Solution:**

Add test helper functions:

```go
// combat/testing.go - NEW FILE

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
        SquadID:    squadID,
        Name:       name,
        Formation:  squads.FormationBalanced,
        MaxUnits:   9,
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

    // Add components
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

    unitEntity.AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
        SquadID: squadID,
    })

    // Position in grid
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
}
```

---

## RISK ASSESSMENT

| Risk | Severity | Probability | Mitigation |
|------|----------|-------------|------------|
| **Component registration forgotten** | CRITICAL | HIGH | Add Phase 0 with init() function as first task |
| **Position system desync** | HIGH | MEDIUM | Choose canonical source (PositionSystem recommended) |
| **Missing query helpers cause compile errors** | CRITICAL | HIGH | Implement queries.go before any system |
| **Pathfinding not specified** | MEDIUM | HIGH | Start with simple 8-dir movement, defer pathfinding |
| **UI layer completely missing** | HIGH | MEDIUM | Add Phase 6.5 for input controller |
| **ExecuteSquadAttack doesn't support partial attacks** | MEDIUM | MEDIUM | Use CanAct filtering workaround |
| **MovementRemaining lifecycle unclear** | MEDIUM | MEDIUM | Initialize in ResetSquadActions |
| **Entity ID lookup O(n) performance** | MEDIUM | LOW | Add entity map to EntityManager |

---

## NEXT STEPS

### Immediate Actions (Week 1)

1. **Implement Phase 0 (NEW):**
   - Create combat/queries.go with all helper functions
   - Add component registration init() function
   - Add entity ID map to EntityManager
   - Test component creation and queries

2. **Revise Phase 1:**
   - Add MovementRemaining initialization to ResetSquadActions
   - Implement complete TurnManager with all missing methods
   - Test turn progression and action state lifecycle

3. **Revise Phase 2:**
   - Decide on position system canonical source
   - Implement basic movement without pathfinding
   - Test squad movement and collision

### Short-Term Actions (Week 2)

4. **Implement Phase 3:**
   - Add AttackRange to Attributes
   - Implement range-based combat with CanAct filtering
   - Test partial squad attacks

5. **Implement Phase 6.5 (NEW):**
   - Create CombatController for player input
   - Test squad selection and command execution

### Long-Term Actions (Week 3+)

6. **Add pathfinding:**
   - Integrate with worldmap package or implement A*
   - Replace simple distance checks with path-based movement

7. **Add map integration:**
   - Define MapConfig and terrain system
   - Implement spawn points and map boundaries

8. **Polish & optimization:**
   - Add event system for UI updates
   - Performance testing with large battles
   - Balance testing

---

## SUMMARY

The existing turn_based_combat_plan.md provides a **strong architectural foundation** but has **critical implementation gaps** that will block development:

**CRITICAL GAPS (MUST FIX BEFORE CODING):**
1. ✅ Component registration init() function
2. ✅ Query helper functions (queries.go)
3. ✅ MovementRemaining lifecycle specification
4. ✅ Position system canonical source decision
5. ✅ Partial squad attack implementation
6. ✅ UI/input layer specification

**RECOMMENDED REVISIONS:**
1. Add Phase 0 for infrastructure
2. Split Phase 2 into basic/advanced movement
3. Add Phase 6.5 for player input
4. Implement queries.go before any system
5. Choose position system architecture

**ESTIMATED ADDITIONAL EFFORT:**
- Phase 0 (NEW): 4-6 hours
- Phase 6.5 (NEW): 6-8 hours
- Revisions to existing phases: 4-6 hours
- **Total revised estimate: 40-56 hours (5-7 workdays)**

This analysis provides the missing specifications needed to implement the turn-based combat system successfully.

---

**END OF GAP ANALYSIS**
