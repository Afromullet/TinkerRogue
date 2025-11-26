# Service Layer Refactoring Recommendations

**Date:** 2025-11-26
**Status:** Analysis Complete
**Focus:** Refactoring services (no unit testing considerations)

---

## Executive Summary

The service layer is **functional but inconsistent**. Four services have been implemented to separate game logic from GUI code:

- `combat/combat_service.go` (271 lines)
- `squads/squad_service.go` (292 lines)
- `squads/squad_builder_service.go` (323 lines)
- `squads/squad_deployment_service.go` (111 lines)

**Main Issues Identified:**
1. Inconsistent API design (error field naming, result patterns)
2. Encapsulation leaks (GUI reaches into ECS and internal systems)
3. Code duplication (CreateSquad exists in two services)
4. Over-exposure of internal subsystems (CombatService exposes 4 managers)
5. Incomplete abstraction (missing convenience methods)

**Recommendation:** Implement P0-P1 refactorings over 1-2 weeks to achieve consistent, maintainable service architecture.

---

## Critical Issues (Priority 0-1)

### 1. Inconsistent Error Field Naming ⚠️ **P0 CRITICAL**

**Problem:** Error fields use three different names across services:

```go
// CombatService - TWO DIFFERENT PATTERNS
AttackResult.ErrorReason        // Line 32
MoveSquadResult.ErrorReason     // Line 75
EndTurnResult.Error             // Line 163 - INCONSISTENT!

// SquadService - Consistent
CreateSquadResult.Error         // Line 28
AddUnitResult.Error             // Line 68

// SquadBuilderService - MOSTLY consistent
PlaceUnitResult.Error           // Line 67
ValidateSquadResult.ErrorMsg    // Line 233 - INCONSISTENT!

// SquadDeploymentService - Consistent
PlaceSquadResult.Error          // Line 28
```

**Impact:** GUI code must check different field names:
```go
// combat_action_handler.go:160
if !result.Success {
    cah.addLog(fmt.Sprintf("Cannot attack: %s", result.ErrorReason))  // Uses ErrorReason
}

// But other handlers expect:
if !result.Success {
    cah.addLog(fmt.Sprintf("Error: %s", result.Error))  // Uses Error
}
```

**Solution:** Standardize to single field name: `Error`

**Files to Change:**
- `combat/combat_service.go`: Lines 32, 75 - Rename `ErrorReason` → `Error`
- `squads/squad_builder_service.go`: Line 233 - Rename `ErrorMsg` → `Error`
- `gui/guicombat/combat_action_handler.go`: Lines 160, 177 - Update field references

**Effort:** 2 hours
**Timeline:** Week 1

---

### 2. Over-Exposure of Internal Subsystems ⚠️ **P0 CRITICAL**

**Problem:** CombatService exposes 4 internal systems through getter methods:

```go
// combat_service.go - Lines 138-155
func (cs *CombatService) GetTurnManager() *TurnManager
func (cs *CombatService) GetFactionManager() *FactionManager
func (cs *CombatService) GetMovementSystem() *MovementSystem
func (cs *CombatService) GetEntityManager() *common.EntityManager
```

**Current GUI Usage (defeats service layer purpose):**
```go
// combatmode.go:154
round := cm.combatService.GetTurnManager().GetCurrentRound()

// combatmode.go:200
squadIDs := cm.combatService.GetFactionManager().GetFactionSquads(factionID)

// combat_action_handler.go:206
entityManager := cah.combatService.GetEntityManager()
if !squads.IsSquadDestroyed(squadID, entityManager) { ... }
```

**Solution:** Remove getters and provide convenience methods:

```go
// Add to CombatService
func (cs *CombatService) GetCurrentRound() int {
    return cs.turnManager.GetCurrentRound()
}

func (cs *CombatService) GetFactionSquads(factionID ecs.EntityID) []ecs.EntityID {
    return cs.factionManager.GetFactionSquads(factionID)
}

func (cs *CombatService) IsSquadDestroyed(squadID ecs.EntityID) bool {
    return squads.IsSquadDestroyed(squadID, cs.entityManager)
}

func (cs *CombatService) GetFactionName(factionID ecs.EntityID) string {
    return cs.factionManager.GetFactionName(factionID)
}

// Remove these 4 methods:
// GetTurnManager(), GetFactionManager(), GetMovementSystem(), GetEntityManager()
```

**Files to Change:**
- `combat/combat_service.go`: Remove lines 138-155, add convenience methods
- `gui/guicombat/combatmode.go`: Lines 154, 200, 224, 237 - use service methods
- `gui/guicombat/combat_action_handler.go`: Lines 200-210 - use service methods

**Effort:** 3 hours
**Timeline:** Week 1

---

### 3. Duplicate CreateSquad Implementation ⚠️ **P1 HIGH**

**Problem:** Both `SquadService` and `SquadBuilderService` have identical `CreateSquad()` methods:

```go
// squad_service.go:32-60
func (ss *SquadService) CreateSquad(squadName string) *CreateSquadResult {
    squadEntity := ss.entityManager.World.NewEntity()
    squadEntity.AddComponent(SquadComponent, &SquadData{
        SquadID:       squadID,
        Name:          squadName,
        Morale:        100,
        MaxUnits:      9,
        UsedCapacity:  0.0,
        TotalCapacity: 6,
    })
    // ... identical code
}

// squad_builder_service.go:32-60
func (sbs *SquadBuilderService) CreateSquad(squadName string) *SquadBuilderSquadResult {
    // EXACT SAME IMPLEMENTATION
    squadEntity := sbs.entityManager.World.NewEntity()
    squadEntity.AddComponent(SquadComponent, &SquadData{
        SquadID:       squadID,
        Name:          squadName,
        Morale:        100,
        MaxUnits:      9,
        UsedCapacity:  0.0,
        TotalCapacity: 6,
    })
    // ... identical code
}
```

**Current Usage:**
- GUI calls: `squadBuilderSvc.CreateSquad()` (squadbuilder.go:264)
- Never calls: `SquadService.CreateSquad()`

**Solution - Option A (Recommended):** Remove from SquadBuilderService, inject SquadService

```go
// squad_builder_service.go
type SquadBuilderService struct {
    entityManager *common.EntityManager
    squadService  *squads.SquadService  // ADD THIS
}

// Remove CreateSquad() method entirely from SquadBuilderService

// If needed for building workflow, delegate:
func (sbs *SquadBuilderService) CreateBuildingSquad(squadName string) *CreateSquadResult {
    return sbs.squadService.CreateSquad(squadName)
}
```

**Solution - Option B:** Keep only in SquadService, have GUI use it directly

**Files to Change:**
- `squads/squad_builder_service.go`: Remove `CreateSquad()` and `SquadBuilderSquadResult` type (lines 24-60)
- `gui/guisquads/squadbuilder.go`: Line 264 - inject and use SquadService instead

**Effort:** 1-2 hours
**Timeline:** Week 1-2

---

### 4. Missing Service Convenience Methods ⚠️ **P1 HIGH**

**Problem:** GUI code accesses ECS directly because services don't provide needed wrappers:

```go
// squadbuilder.go:344 - Direct ECS access
unitIDs := squads.GetUnitIDsInSquad(sbm.currentSquadID, sbm.Context.ECSManager)

// squadbuilder.go:373 - Direct query function call
visualization := squads.VisualizeSquad(sbm.currentSquadID, sbm.Context.ECSManager)

// squadbuilder.go:97 - Direct query function call
if !CanAddUnitToSquad(squadID, unitCapacityCost, sbs.entityManager) {

// squaddeploymentmode.go:247 - Direct map access
oldPos := sdm.deploymentService.GetAllSquadPositions()[squadID]
```

**Solution:** Add convenience methods to services:

```go
// squad_service.go additions
func (ss *SquadService) GetUnitIDsInSquad(squadID ecs.EntityID) []ecs.EntityID {
    return GetUnitIDsInSquad(squadID, ss.entityManager)
}

func (ss *SquadService) VisualizeSquad(squadID ecs.EntityID) string {
    return VisualizeSquad(squadID, ss.entityManager)
}

func (ss *SquadService) CanAddMoreUnits(squadID ecs.EntityID, cost float64) bool {
    return CanAddUnitToSquad(squadID, cost, ss.entityManager)
}

// squad_deployment_service.go additions
func (sds *SquadDeploymentService) GetSquadPosition(squadID ecs.EntityID) (coords.LogicalPosition, error) {
    squadEntity := common.FindEntityByIDWithTag(sds.entityManager, squadID, SquadTag)
    if squadEntity == nil {
        return coords.LogicalPosition{}, fmt.Errorf("squad not found")
    }

    posPtr := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
    if posPtr == nil {
        return coords.LogicalPosition{}, fmt.Errorf("squad has no position")
    }
    return *posPtr, nil
}
```

**Files to Change:**
- `squads/squad_service.go`: Add 3 convenience methods
- `squads/squad_deployment_service.go`: Add GetSquadPosition() method
- `gui/guisquads/squadbuilder.go`: Lines 97, 344, 373 - use service methods
- `gui/guisquads/squaddeploymentmode.go`: Line 247 - use service method

**Effort:** 2 hours
**Timeline:** Week 1-2

---

## Medium Priority Issues (Priority 2)

### 5. Inconsistent Result Type Naming 📋 **P2 MEDIUM**

**Problem:** Result type names are inconsistent:

```go
// Awkward naming in SquadBuilderService
type SquadBuilderSquadResult struct { ... }  // "SquadBuilder" prefix is redundant

// But SquadService has clearer naming
type CreateSquadResult struct { ... }

// Similar inconsistency
type RemoveUnitFromGridResult struct { ... }  // SquadBuilder
type RemoveUnitResult struct { ... }          // SquadService
```

**Solution:** Rename for clarity:

```go
// squad_builder_service.go
type SquadBuilderSquadResult → type CreateSquadResult  // Or remove entirely (see issue #3)
type RemoveUnitFromGridResult → type RemoveUnitResult
```

**Files to Change:**
- `squads/squad_builder_service.go`: Lines 24, 134 - rename types
- Update all references to these types

**Effort:** 1 hour
**Timeline:** Week 2

---

### 6. Consolidate Capacity Query Methods 📋 **P2 MEDIUM**

**Problem:** Capacity queries exist in both SquadService and SquadBuilderService:

```go
// SquadService
func (ss *SquadService) CanAddMoreUnits(squadID, cost) bool
func (ss *SquadService) GetSquadRemainingCapacity(squadID) float64

// SquadBuilderService
func (sbs *SquadBuilderService) GetCapacityInfo(squadID) *SquadCapacityInfo
```

**Current Usage:**
- `squadbuilder.go:97`: Calls package-level `CanAddUnitToSquad()` query function
- `squadbuilder.go:280`: Calls `squadBuilderSvc.GetCapacityInfo()`

**Solution:** Consolidate in SquadService:

```go
// squad_service.go - Single source of truth for capacity
func (ss *SquadService) GetCapacityInfo(squadID ecs.EntityID) *SquadCapacityInfo {
    // Move implementation from SquadBuilderService here
}

// squad_builder_service.go - Delegate if needed
func (sbs *SquadBuilderService) GetCapacityInfo(squadID ecs.EntityID) *SquadCapacityInfo {
    return sbs.squadService.GetCapacityInfo(squadID)  // Requires SquadService injection
}
```

**Alternative:** Remove from SquadBuilderService entirely, have GUI use SquadService directly.

**Files to Change:**
- `squads/squad_service.go`: Add `GetCapacityInfo()` if using delegation approach
- `squads/squad_builder_service.go`: Either delegate or remove method
- `gui/guisquads/squadbuilder.go`: Update service calls if needed

**Effort:** 2 hours
**Timeline:** Week 2

---

### 7. Clarify SquadBuilderService Single Responsibility 📋 **P2 MEDIUM**

**Problem:** SquadBuilderService has 7 distinct concerns:

1. **Unit Placement** - `PlaceUnit()`
2. **Unit Removal** - `RemoveUnitFromGrid()`
3. **Leader Management** - `DesignateLeader()`
4. **Validation** - `ValidateSquad()`
5. **Finalization** - `FinalizeSquad()`
6. **Naming** - `UpdateSquadName()`
7. **Information** - `GetCapacityInfo()`

**Analysis:**
- `FinalizeSquad()` is just a wrapper around `ValidateSquad()` - no actual state changes
- `GetCapacityInfo()` duplicates SquadService functionality
- `UpdateSquadName()` is squad configuration, not building

**Solution:** Refocus on core "building" concerns:

```go
// Keep in SquadBuilderService (core building operations)
PlaceUnit()          // Unit placement on grid
RemoveUnitFromGrid() // Unit removal from grid
DesignateLeader()    // Leader designation
ValidateSquad()      // Validation of build completion

// Remove or move elsewhere
FinalizeSquad()      // Remove - just calls ValidateSquad()
GetCapacityInfo()    // Move to SquadService or delegate
UpdateSquadName()    // Move to SquadService (squad configuration)
```

**Refactored Method:**
```go
// Consolidate validation and finalization
func (sbs *SquadBuilderService) ValidateSquadForCompletion(squadID ecs.EntityID) *ValidateSquadResult {
    // Combines ValidateSquad + FinalizeSquad logic
    // Returns comprehensive validation result
}
```

**Files to Change:**
- `squads/squad_builder_service.go`: Remove FinalizeSquad(), move UpdateSquadName()
- `squads/squad_service.go`: Add UpdateSquadName() if moving
- GUI files: Update method calls if needed

**Effort:** 3 hours
**Timeline:** Week 2-3

---

### 8. Standardize Result Type Patterns 📋 **P2 MEDIUM**

**Problem:** Mixed patterns for queries vs mutations:

```go
// Mutations return explicit Result types
ExecuteSquadAttack() *AttackResult
AddUnitToSquad() *AddUnitResult

// But queries return primitives
GetValidMovementTiles() []coords.LogicalPosition
GetSquadsInRange() []ecs.EntityID

// Some queries return Result-like types
GetSquadInfo() *GetSquadInfoResult
GetCapacityInfo() *SquadCapacityInfo

// Some operations return raw errors
InitializeCombat(factionIDs []ecs.EntityID) error
```

**Current Pattern (mostly used):**
- **Mutations** → Return `*ResultType` with Success/Error fields
- **Queries** → Return primitives or empty collections on failure

**Recommendation:** Formalize this pattern consistently:

**Pattern A - Explicit Results (Recommended):**
```go
// Mutations: Always return Result types
type AttackResult struct {
    Success bool
    Error   string
    // ... operation data
}

// Queries: Return primitives/structs, empty on failure
GetValidMovementTiles(squadID) []coords.LogicalPosition  // Returns [] if fail
GetSquadInfo(squadID) *GetSquadInfoResult                // Returns populated struct or empty struct
```

**Pattern B - Uniform Wrapper:**
```go
type Result[T any] struct {
    Success bool
    Error   string
    Data    T
}

ExecuteSquadAttack() *Result[AttackData]
GetValidMovementTiles() *Result[[]coords.LogicalPosition]
```

**Recommendation:** Use Pattern A (current majority pattern) but enforce strictly.

**Files to Change:**
- All 4 services: Audit method signatures for consistency
- Document pattern in service header comments

**Effort:** 2 hours (audit + documentation)
**Timeline:** Week 2

---

## Low Priority Issues (Priority 3)

### 9. Add Validation to SquadDeploymentService 📝 **P3 LOW**

**Problem:** SquadDeploymentService is minimal - just position manipulation, no validation:

```go
// Current methods
PlaceSquadAtPosition(squadID, pos)  // No bounds checking
ClearAllSquadPositions()
GetAllSquadPositions()
```

**Missing Validation:**
- Position bounds checking (is position within map?)
- Squad readiness validation (has leader? has units?)
- Deployment state validation (all required squads placed?)

**Solution:** Add validation methods:

```go
// squad_deployment_service.go additions
func (sds *SquadDeploymentService) IsValidPosition(pos coords.LogicalPosition) bool {
    // Check if position is within dungeon bounds
    // Check if position is walkable/valid
}

func (sds *SquadDeploymentService) CanDeploySquad(squadID ecs.EntityID) bool {
    // Check if squad has leader
    // Check if squad has units
    // Check if squad is valid for deployment
}

func (sds *SquadDeploymentService) ValidateAllDeployments() *DeploymentValidationResult {
    // Check all squads have been placed
    // Check no position conflicts
    // Check all required squads are deployed
}
```

**Files to Change:**
- `squads/squad_deployment_service.go`: Add 3 validation methods
- `gui/guisquads/squaddeploymentmode.go`: Use validation before allowing combat start

**Effort:** 2 hours
**Timeline:** Week 3+

---

### 10. Refactor UpdateUnitPositions Placement 📝 **P3 LOW**

**Problem:** `CombatService.UpdateUnitPositions()` is a one-off method (line 244):

```go
// Called from combat_action_handler.go:183
cah.combatService.UpdateUnitPositions(squadID, newPos)
```

**Issue:**
- Tightly coupled to squad movement
- Not part of MovementSystem
- Manually syncs unit positions to squad position

**Options:**
1. **Integrate into MoveSquad()** - Make automatic
2. **Move to MovementSystem** - Better logical home
3. **Keep as is** - Document why it exists

**Recommended Solution:** Make it automatic in `MoveSquad()`:

```go
// combat_service.go
func (cs *CombatService) MoveSquad(squadID ecs.EntityID, newPos coords.LogicalPosition) *MoveSquadResult {
    result := &MoveSquadResult{ NewPosition: newPos }

    // Execute movement
    err := cs.movementSystem.MoveSquad(squadID, newPos)
    if err != nil {
        result.Success = false
        result.Error = err.Error()
        return result
    }

    // Update unit positions AUTOMATICALLY (private method)
    cs.updateUnitPositions(squadID, newPos)

    result.Success = true
    result.SquadName = getSquadNameByID(squadID, cs.entityManager)
    return result
}

// Make private
func (cs *CombatService) updateUnitPositions(...) { ... }
```

**Files to Change:**
- `combat/combat_service.go`: Make updateUnitPositions private, integrate into MoveSquad
- `gui/guicombat/combat_action_handler.go`: Line 183 - remove explicit call

**Effort:** 1 hour
**Timeline:** Week 3+

---

## Code Duplication Summary

### Exact Duplications

| Duplication | Location 1 | Location 2 | Recommendation |
|-------------|-----------|-----------|----------------|
| CreateSquad() implementation | squad_service.go:32-60 | squad_builder_service.go:32-60 | Remove from SquadBuilderService |
| Leader checking loop | squad_builder_service.go:241-256 | squad_builder_service.go:305-309 | Extract to private helper |

### Near-Duplications

| Pattern | Locations | Recommendation |
|---------|-----------|----------------|
| Result type structure | All 4 services | Standardize with base ServiceResult type |
| EntityManager access pattern | All services | Document pattern consistently |
| Error checking pattern | All GUI files | Standardize error field name |

---

## Service Architecture Patterns

### Recommended Standard Patterns

**1. Result Type Pattern:**
```go
// Base result type (optional - can use embedding)
type ServiceResult struct {
    Success bool
    Error   string
}

// Operation-specific result
type CreateSquadResult struct {
    Success   bool
    Error     string
    SquadID   ecs.EntityID
    SquadName string
}
```

**2. Service Construction:**
```go
func NewXxxService(manager *common.EntityManager, deps ...ServiceDep) *XxxService {
    return &XxxService{
        entityManager: manager,
        // dependencies
    }
}
```

**3. Method Naming:**
- **Mutations:** `ExecuteXxx`, `CreateXxx`, `AddXxx`, `RemoveXxx`, `UpdateXxx`
- **Queries:** `GetXxx`, `FindXxx`, `IsXxx`, `CanXxx`, `HasXxx`
- **Validation:** `ValidateXxx`, `CanDoXxx`

**4. Error Handling:**
- All Result types use `Error string` field
- Mutations return `*ResultType`
- Queries return primitives or structs (empty on failure)

---

## Implementation Roadmap

### Week 1: Critical Fixes (P0)
**Total Effort:** 5 hours

- [ ] **Day 1-2:** Standardize error field naming (2 hours)
  - Rename `ErrorReason` → `Error` in CombatService
  - Rename `ErrorMsg` → `Error` in SquadBuilderService
  - Update GUI references
  - Test compilation

- [ ] **Day 3-5:** Remove CombatService internal system exposure (3 hours)
  - Add convenience methods to CombatService
  - Remove GetTurnManager(), GetFactionManager(), GetMovementSystem(), GetEntityManager()
  - Update all GUI calls
  - Test compilation and runtime behavior

### Week 2: High Priority Refactoring (P1-P2)
**Total Effort:** 7-9 hours

- [ ] **Day 1-2:** Remove CreateSquad duplication (2 hours)
  - Remove from SquadBuilderService
  - Update GUI to use SquadService
  - Test squad creation workflow

- [ ] **Day 3:** Add missing convenience methods (2 hours)
  - Add GetUnitIDsInSquad, VisualizeSquad, CanAddMoreUnits to SquadService
  - Add GetSquadPosition to SquadDeploymentService
  - Update GUI to use new methods

- [ ] **Day 4-5:** Consolidate capacity queries (2-3 hours)
  - Move GetCapacityInfo to SquadService
  - Update SquadBuilderService to delegate or remove
  - Update GUI calls

- [ ] **Day 5:** Rename result types for consistency (1 hour)
  - Rename SquadBuilderSquadResult
  - Update all references

### Week 3+: Polish and Documentation (P3)
**Total Effort:** 5 hours

- [ ] Add validation to SquadDeploymentService (2 hours)
- [ ] Refactor UpdateUnitPositions placement (1 hour)
- [ ] Document service contracts and patterns (2 hours)

---

## Testing Strategy (Post-Refactoring)

**Note:** User requested no unit testing considerations, but manual testing checklist:

**After Each Refactoring:**
1. Compile all packages: `go build ./...`
2. Run game executable: Test affected workflows
3. Check error messages display correctly
4. Verify no runtime panics

**Critical Workflows to Test:**
- Squad creation in squad builder
- Unit placement and removal
- Squad deployment positioning
- Combat turn progression
- Squad movement in combat
- Attack execution

---

## Risk Assessment

### Low Risk Refactorings
- Error field renaming (mechanical change)
- Adding convenience methods (pure additions)
- Documentation updates

### Medium Risk Refactorings
- Removing CreateSquad duplication (changes initialization flow)
- Consolidating capacity queries (changes data flow)
- Removing internal system getters (widespread GUI changes)

### High Risk Refactorings
- None identified (all proposed changes are low-medium risk)

**Mitigation:** Implement P0 items first to establish patterns before tackling P1-P2 items.

---

## Conclusion

The service layer successfully separates game logic from GUI code but suffers from:
- **Inconsistent API design** (3 different error field names)
- **Encapsulation leaks** (4 internal system getters)
- **Code duplication** (CreateSquad in 2 places)
- **Incomplete abstraction** (missing convenience methods)

**Recommended Approach:**
1. **Week 1:** Fix critical inconsistencies (error naming, remove system exposure)
2. **Week 2:** Eliminate duplication and add missing methods
3. **Week 3+:** Polish and document

**Expected Outcome:**
- Consistent, maintainable service layer
- Complete encapsulation of game logic
- GUI code purely orchestrates services
- Clear service boundaries and responsibilities

**Success Metrics:**
- Zero direct ECS access from GUI code
- All error fields named consistently
- No internal system getters in CombatService
- All capacity queries in single location
- Documented service contracts
