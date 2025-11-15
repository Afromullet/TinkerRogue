# Combat and Squad Package Duplicates

**Date:** 2025-11-13
**Focus:** Functions that are duplicated or have inconsistent implementations

---

## 1. GetSquadMovementSpeed - TRUE DUPLICATE WITH BEHAVIORAL DIFFERENCE

**Severity:** HIGH - Different behavior with dead units

### Combat Version
**Location:** `combat/movementsystem.go:27-56`

```go
func (ms *MovementSystem) GetSquadMovementSpeed(squadID ecs.EntityID) int {
    unitIDs := squads.GetUnitIDsInSquad(squadID, ms.manager)
    minSpeed := 999
    for _, unitID := range unitIDs {
        unit := squads.FindUnitByID(unitID, ms.manager)
        attr := common.GetAttributes(unit)
        speed := attr.GetMovementSpeed()
        if speed < minSpeed {
            minSpeed = speed
        }
    }
    return minSpeed
}
```

**Issues:**
- ❌ Includes dead units in calculation
- ❌ Uses `GetMovementSpeed()` method instead of component
- ❌ Returns 999 for destroyed squads (should be 0)
- ❌ Uses hardcoded 999 instead of `math.MaxInt32`

### Squad Version
**Location:** `squads/squadqueries.go:249-285`

```go
func GetSquadMovementSpeed(squadID ecs.EntityID, squadmanager *common.EntityManager) int {
    unitIDs := GetUnitIDsInSquad(squadID, squadmanager)
    minSpeed := math.MaxInt32
    foundValidUnit := false

    for _, unitID := range unitIDs {
        unitEntity := FindUnitByID(unitID, squadmanager)
        attr := common.GetAttributes(unitEntity)

        // CRITICAL: Filters dead units
        if attr.CurrentHealth <= 0 {
            continue
        }

        speedData := common.GetComponentType[*MovementSpeedData](unitEntity, MovementSpeedComponent)
        if speedData.Speed < minSpeed {
            minSpeed = speedData.Speed
            foundValidUnit = true
        }
    }

    if !foundValidUnit {
        return 0 // Squad is destroyed
    }
    return minSpeed
}
```

**Advantages:**
- ✅ Filters dead units correctly
- ✅ Uses `MovementSpeedComponent` (proper ECS)
- ✅ Returns 0 for destroyed squads
- ✅ Proper initialization with `math.MaxInt32`

### Comparison Table

| Aspect | Combat Version | Squad Version | Winner |
|--------|----------------|---------------|--------|
| Dead unit handling | Includes dead units | Filters dead units | Squad ✓ |
| Data source | `GetMovementSpeed()` method | `MovementSpeedComponent` | Squad ✓ |
| Destroyed squad | Returns 999 | Returns 0 | Squad ✓ |
| Initialization | `minSpeed := 999` | `minSpeed := math.MaxInt32` | Squad ✓ |

### Recommendation
**Delete combat version, use `squads.GetSquadMovementSpeed()` everywhere**

---

## 2. Distance Calculation - ALGORITHMIC INCONSISTENCY

**Severity:** HIGH - Different algorithms give different results

### Combat Package (Chebyshev Distance)
**Location:** `combat/combatactionsystem.go:33`

```go
distance := attackerPos.ChebyshevDistance(&defenderPos)
```

**Implementation:** `coords/position.go`
```go
func (lp LogicalPosition) ChebyshevDistance(other *LogicalPosition) int {
    dx := abs(lp.X - other.X)
    dy := abs(lp.Y - other.Y)
    return max(dx, dy)  // Diagonal = 1 step
}
```

### Squad Package (Manhattan Distance)
**Location:** `squads/squadqueries.go:240-243`

```go
func GetSquadDistance(squad1ID ecs.EntityID, squad2ID ecs.EntityID, squadmanager *common.EntityManager) int {
    dx := math.Abs(float64(pos1.X - pos2.X))
    dy := math.Abs(float64(pos1.Y - pos2.Y))
    return int(dx + dy)  // Manhattan distance
}
```

### Problem Example: Distance from (0,0) to (3,3)

| Algorithm | Calculation | Result |
|-----------|-------------|--------|
| **Chebyshev** (Combat) | `max(3, 3)` | **3** |
| **Manhattan** (Squad) | `3 + 3` | **6** |

**Impact:** A squad at (0,0) attacking (3,3) with range 3 would:
- ✅ Be in range according to combat system (Chebyshev = 3)
- ❌ Be out of range according to squad distance query (Manhattan = 6)

### More Examples

| From → To | Chebyshev | Manhattan | Difference |
|-----------|-----------|-----------|------------|
| (0,0) → (1,1) | 1 | 2 | 1 |
| (0,0) → (2,2) | 2 | 4 | 2 |
| (0,0) → (3,3) | 3 | 6 | 3 |
| (0,0) → (5,0) | 5 | 5 | 0 |
| (0,0) → (3,4) | 4 | 7 | 3 |

**Pattern:** Manhattan distance is always ≥ Chebyshev distance, with divergence on diagonals

### Recommendation
**Standardize on Chebyshev everywhere** (matches tactical grid movement)

**Fix:** Update `GetSquadDistance()` to use Chebyshev:
```go
func GetSquadDistance(squad1ID ecs.EntityID, squad2ID ecs.EntityID, squadmanager *common.EntityManager) int {
    pos1, err1 := getSquadPosition(squad1ID, squadmanager)
    pos2, err2 := getSquadPosition(squad2ID, squadmanager)

    if err1 != nil || err2 != nil {
        return -1
    }

    // CHANGE: Use Chebyshev instead of Manhattan
    return pos1.ChebyshevDistance(&pos2)
}
```

---

## 3. FindSquadByID vs GetSquadEntity - DUPLICATE QUERY WITH DIFFERENT LOGIC

**Severity:** MEDIUM - Same purpose, different implementation

### Combat Version: FindSquadByID
**Location:** `combat/queries.go:68-76`

```go
func FindSquadByID(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
    for _, result := range manager.World.Query(squads.SquadTag) {
        if result.Entity.GetID() == squadID {
            return result.Entity
        }
    }
    return nil
}
```

**Logic:** Compares `Entity.GetID()` directly with `squadID`

### Squad Version: GetSquadEntity
**Location:** `squads/squadqueries.go:68-81`

```go
func GetSquadEntity(squadID ecs.EntityID, squadmanager *common.EntityManager) *ecs.Entity {
    for _, result := range squadmanager.World.Query(SquadTag) {
        squadEntity := result.Entity
        squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
        if squadData.SquadID == squadID {
            return squadEntity
        }
    }
    return nil
}
```

**Logic:** Extracts `SquadData` component and compares `SquadData.SquadID`

### Key Difference

| Aspect | Combat Version | Squad Version |
|--------|----------------|---------------|
| **Comparison** | `Entity.GetID() == squadID` | `SquadData.SquadID == squadID` |
| **Component Access** | No | Yes (reads SquadData) |
| **Performance** | Faster (no component read) | Slower (reads component) |

### Potential Issue

**If `Entity.GetID() != SquadData.SquadID`, these functions return different results!**

This could happen if:
- Squad creation sets `SquadData.SquadID` to a custom value
- Entity ID is auto-generated, but SquadData.SquadID is manually set

### Verification Required

Check if this invariant holds:
```go
// For all squad entities:
Entity.GetID() == SquadData.SquadID
```

### Recommendation

**Option 1:** Use squad version everywhere (safer, guarantees SquadData.SquadID match)
```go
// Delete combat/queries.go:FindSquadByID
// Replace all calls with:
squads.GetSquadEntity(squadID, manager)
```

**Option 2:** Verify invariant, then use combat version (faster)
```go
// Add assertion in CreateSquadFromTemplate:
assert(entity.GetID() == squadData.SquadID)
```

---

## Summary Table

| Function | Combat Location | Squad Location | Issue | Recommendation |
|----------|-----------------|----------------|-------|----------------|
| **GetSquadMovementSpeed** | movementsystem.go:27 | squadqueries.go:249 | Different behavior with dead units | Use squad version |
| **Distance Calculation** | combatactionsystem.go:33 | squadqueries.go:240 | Chebyshev vs Manhattan | Standardize to Chebyshev |
| **FindSquadByID / GetSquadEntity** | queries.go:68 | squadqueries.go:68 | Different comparison logic | Use squad version |

---

## Quick Fix Checklist

### Fix 1: GetSquadMovementSpeed (30 min)
- [ ] Delete `combat/movementsystem.go:27-56`
- [ ] Update calls to use `squads.GetSquadMovementSpeed(squadID, manager)`
- [ ] Test with squads containing dead units

### Fix 2: Distance Calculation (30 min)
- [ ] Update `squads/squadqueries.go:240-243` to use Chebyshev
- [ ] Test diagonal distance calculations
- [ ] Verify attack range matches distance queries

### Fix 3: FindSquadByID (15 min)
- [ ] Delete `combat/queries.go:68-76`
- [ ] Replace all calls with `squads.GetSquadEntity()`
- [ ] Verify `Entity.GetID() == SquadData.SquadID` invariant

**Total Time:** ~75 minutes
