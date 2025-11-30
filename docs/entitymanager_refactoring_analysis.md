# EntityManager Refactoring Analysis

**Generated:** 2025-11-29
**Analysis Scope:** Complete TinkerRogue codebase (133 Go files, 29 packages)

---

## Executive Summary

The EntityManager in TinkerRogue is a well-designed wrapper around the bytearena/ecs library that follows many ECS best practices. However, analysis reveals several opportunities for improvement:

| Category | Current State | Impact | Effort |
|----------|--------------|--------|--------|
| Performance | O(n) lookups in hot paths | HIGH | MEDIUM |
| Type Safety | String-keyed WorldTags | MEDIUM | LOW |
| Consistency | Mixed EntityManager vs ecs.Manager | LOW | LOW |
| Error Handling | Silent panic recovery | MEDIUM | LOW |
| Entity Lookup | 44 unnecessary pointer lookups | LOW | LOW |

**Recommendation:** Address performance issues first (Priority 1), then type safety (Priority 2).

---

## Architecture Overview

### Current EntityManager Definition
**File:** `common/ecsutil.go:28-41`

```go
type EntityManager struct {
    World     *ecs.Manager          // The underlying ECS world
    WorldTags map[string]ecs.Tag    // Named tags for querying entities
}
```

### Strengths
- **Clean Wrapper Pattern:** Provides centralized access to ECS functionality
- **Parameter Passing:** Never used as global (excellent discipline)
- **Component Purity:** 100% of components use EntityID, not pointers
- **Query-Based Design:** Relationships queried dynamically, not cached
- **GlobalPositionSystem:** O(1) spatial queries already implemented

### Key Files
| File | Purpose | Lines |
|------|---------|-------|
| `common/ecsutil.go` | EntityManager definition, all lookup functions | 302 |
| `systems/positionsystem.go` | O(1) spatial grid (reference implementation) | 182 |
| `game_main/componentinit.go` | Central ECS initialization | 92 |
| `squads/squadqueries.go` | Primary example of best practices | 293 |

---

## Priority 1: Performance Bottlenecks (HIGH IMPACT)

### Issue 1.1: AllEntitiesTag O(n) Full-World Scans

**Severity:** CRITICAL
**Location:** `common/ecsutil.go:44-132`

**Problem:** Four core functions iterate through ALL entities:

```go
// Lines 54-62 - O(n) scan of entire world
func (em *EntityManager) HasComponent(entityID ecs.EntityID, component *ecs.Component) bool {
    for _, result := range em.World.Query(AllEntitiesTag) {  // SCANS EVERYTHING
        if result.Entity.GetID() == entityID {
            _, ok := result.Entity.GetComponentData(component)
            return ok
        }
    }
    return false
}
```

**Affected Functions:**
| Function | Line | Impact |
|----------|------|--------|
| `GetAllEntities()` | 44-50 | Low (rare) |
| `HasComponent()` | 54-62 | HIGH (combat) |
| `GetComponent()` | 79-86 | Medium |
| `GetComponentTypeByID()` | 113-132 | Medium |
| `FindEntityByID()` | 252-259 | Low |

**Recommendation:** Implement entity ID index map

```go
// Add to EntityManager
type EntityManager struct {
    World       *ecs.Manager
    WorldTags   map[string]ecs.Tag
    entityIndex map[ecs.EntityID]*ecs.Entity  // NEW: O(1) lookup
}

// Update on entity creation
func (em *EntityManager) TrackEntity(entity *ecs.Entity) {
    if em.entityIndex == nil {
        em.entityIndex = make(map[ecs.EntityID]*ecs.Entity)
    }
    em.entityIndex[entity.GetID()] = entity
}

// O(1) lookup
func (em *EntityManager) GetEntityByID(entityID ecs.EntityID) *ecs.Entity {
    return em.entityIndex[entityID]
}

// O(1) component check
func (em *EntityManager) HasComponent(entityID ecs.EntityID, component *ecs.Component) bool {
    entity := em.entityIndex[entityID]
    if entity == nil {
        return false
    }
    _, ok := entity.GetComponentData(component)
    return ok
}
```

**Migration Steps:**
1. Add `entityIndex` field to EntityManager
2. Update `NewEntityManager()` to initialize map
3. Create `TrackEntity()` and `UntrackEntity()` methods
4. Update all entity creation points to call `TrackEntity()`
5. Update `DisposeEntities()` calls to also call `UntrackEntity()`
6. Refactor lookup functions to use index

---

### Issue 1.2: containsEntity Linear Search in Combat Hot Path

**Severity:** HIGH
**Location:** `combat/combatactionsystem.go:54`

**Problem:** Linear search on every attack action:

```go
for _, unitID := range allUnits {
    if !containsEntity(attackingUnits, unitID) {  // O(k) per unit
        // ...
    }
}

func containsEntity(entities []ecs.EntityID, entityID ecs.EntityID) bool {
    for _, e := range entities {
        if e == entityID { return true }
    }
    return false
}
```

**Impact:** O(n × k) where n = allUnits, k = attackingUnits

**Recommendation:** Use map for O(1) lookup:

```go
// Convert slice to map before loop
attackingUnitsMap := make(map[ecs.EntityID]bool, len(attackingUnits))
for _, id := range attackingUnits {
    attackingUnitsMap[id] = true
}

for _, unitID := range allUnits {
    if !attackingUnitsMap[unitID] {  // O(1) lookup
        // ...
    }
}
```

---
I
### Issue 1.3: IsSquadDestroyed Repeated Queries

**Severity:** HIGH
**Location:** `combat/combatservices/combat_service.go:190-207`

**Problem:** Victory check queries IsSquadDestroyed() for every squad every turn:

```go
for _, queryResult := range cs.entityManager.World.Query(squads.SquadTag) {
    // Each IsSquadDestroyed does: GetUnitIDsInSquad (query) + N health checks
    if !squads.IsSquadDestroyed(entity.GetID(), cs.entityManager) {
        // ...
    }
}
```

**Recommendation:** Add destroyed flag to SquadData component:

```go
// squads/squadcomponents.go
type SquadData struct {
    SquadID     ecs.EntityID
    SquadName   string
    IsDestroyed bool  // NEW: cached destruction state
}

// Update when unit dies
func UpdateSquadDestroyedStatus(squadID ecs.EntityID, manager *common.EntityManager) {
    unitIDs := GetUnitIDsInSquad(squadID, manager)
    destroyed := true
    for _, unitID := range unitIDs {
        attr := common.GetAttributesByIDWithTag(manager, unitID, SquadMemberTag)
        if attr != nil && attr.CurrentHealth > 0 {
            destroyed = false
            break
        }
    }

    squadData := GetSquadDataByID(squadID, manager)
    if squadData != nil {
        squadData.IsDestroyed = destroyed
    }
}

// O(1) check instead of query
func IsSquadDestroyed(squadID ecs.EntityID, manager *common.EntityManager) bool {
    squadData := GetSquadDataByID(squadID, manager)
    return squadData == nil || squadData.IsDestroyed
}
```

---

### Issue 1.4: GetUnitIDsInRow Nested Queries

**Severity:** MEDIUM
**Location:** `squads/squadqueries.go:75-97`

**Problem:** Calls GetUnitIDsAtGridPosition() 3 times (once per column), each doing a full query:

```go
func GetUnitIDsInRow(squadID ecs.EntityID, row int, squadmanager *common.EntityManager) []ecs.EntityID {
    for col := 0; col < 3; col++ {
        idsAtPos := GetUnitIDsAtGridPosition(squadID, row, col, squadmanager)  // Query × 3
    }
}
```

**Recommendation:** Single query with post-filtering:

```go
func GetUnitIDsInRow(squadID ecs.EntityID, row int, manager *common.EntityManager) []ecs.EntityID {
    var unitIDs []ecs.EntityID

    // Single query
    for _, result := range manager.World.Query(SquadMemberTag) {
        memberData := common.GetComponentType[*SquadMemberData](result.Entity, SquadMemberComponent)
        if memberData == nil || memberData.SquadID != squadID {
            continue
        }

        gridPos := common.GetComponentType[*GridPositionData](result.Entity, GridPositionComponent)
        if gridPos != nil && gridPos.Row == row {  // Filter by row, any column
            unitIDs = append(unitIDs, result.Entity.GetID())
        }
    }

    return unitIDs
}
```

---

### Issue 1.5: GetSquadAtPosition Linear Scan (Should Use GlobalPositionSystem)

**Severity:** MEDIUM
**Location:** `combat/combatqueries.go:108-128`

**Problem:** Scans all squads when GlobalPositionSystem exists for O(1) lookups:

```go
for _, result := range manager.World.Query(squads.SquadTag) {
    squadPos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
    if squadPos.X == pos.X && squadPos.Y == pos.Y {
        return entity  // Found
    }
}
```

**Recommendation:** Use GlobalPositionSystem:

```go
func GetSquadAtPosition(manager *common.EntityManager, pos *coords.LogicalPosition) ecs.EntityID {
    entityIDs := common.GlobalPositionSystem.GetEntitiesAtPosition(*pos)

    for _, entityID := range entityIDs {
        // Check if this entity is a squad
        squadData := common.GetComponentTypeByIDWithTag[*squads.SquadData](
            manager, entityID, squads.SquadTag, squads.SquadComponent)
        if squadData != nil {
            return entityID
        }
    }
    return 0
}
```

---

## Priority 2: Type Safety (MEDIUM IMPACT)

### Issue 2.1: WorldTags String Keys Lack Type Safety

**Severity:** MEDIUM
**Location:** Throughout codebase (13 string keys)

**Problem:** Raw string keys are error-prone:

```go
// Current - typo would fail silently
manager.WorldTags["monstrs"]  // Bug: missing 'e'

// Accessed in hot paths without validation
for _, c := range manager.World.Query(manager.WorldTags["monsters"]) { ... }
```

**Current Tags:**
```
"all", "renderables", "messengers", "items", "monsters",
"squad", "squadmember", "leader", "faction", "turnstate",
"actionstate", "combatfaction", "players"
```

**Recommendation:** Create tag key constants:

```go
// common/tagkeys.go (NEW FILE)
package common

// TagKey provides type-safe keys for WorldTags map
type TagKey string

const (
    TagAll          TagKey = "all"
    TagRenderables  TagKey = "renderables"
    TagMessengers   TagKey = "messengers"
    TagItems        TagKey = "items"
    TagMonsters     TagKey = "monsters"
    TagSquad        TagKey = "squad"
    TagSquadMember  TagKey = "squadmember"
    TagLeader       TagKey = "leader"
    TagFaction      TagKey = "faction"
    TagTurnState    TagKey = "turnstate"
    TagActionState  TagKey = "actionstate"
    TagCombatFaction TagKey = "combatfaction"
    TagPlayers      TagKey = "players"
)

// Update EntityManager
type EntityManager struct {
    World     *ecs.Manager
    WorldTags map[TagKey]ecs.Tag  // Changed from string to TagKey
}

// Safe accessor with validation
func (em *EntityManager) GetTag(key TagKey) (ecs.Tag, bool) {
    tag, ok := em.WorldTags[key]
    return tag, ok
}

// Query with validation
func (em *EntityManager) QueryByTag(key TagKey) []ecs.QueryResult {
    tag, ok := em.WorldTags[key]
    if !ok {
        return nil  // Or log warning
    }
    return em.World.Query(tag)
}
```

**Migration:** Search and replace all `WorldTags["...]` usages with constants.

---

### Issue 2.2: Tag Registration Scattered Across Packages

**Severity:** LOW
**Locations:**
- `game_main/componentinit.go` - Core tags
- `squads/squadmanager.go` - Squad tags
- `combat/combatcomponents.go` - Combat tags
- `game_main/gameinit.go` - Player tags (inconsistent location)

**Recommendation:** Document registration order and centralize validation:

```go
// game_main/componentinit.go - Add validation function
func ValidateAllTagsRegistered(em *common.EntityManager) error {
    requiredTags := []common.TagKey{
        common.TagAll,
        common.TagRenderables,
        common.TagMonsters,
        common.TagSquad,
        common.TagSquadMember,
        common.TagFaction,
        // ... all required tags
    }

    for _, key := range requiredTags {
        if _, ok := em.WorldTags[key]; !ok {
            return fmt.Errorf("missing required tag: %s", key)
        }
    }
    return nil
}
```

---

## Priority 3: Consistency Improvements (LOW IMPACT)

---

---

## Priority 4: Error Handling (MEDIUM IMPACT)

### Issue 4.1: Silent Panic Recovery in Generic Functions

**Severity:** MEDIUM
**Location:** `common/ecsutil.go:91-108, 113-132, 134-155`

**Problem:** Panics are caught and silently ignored:

```go
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {
    defer func() {
        if r := recover(); r != nil {
            // ERROR HANDLING IN FUTURE  <-- Silent failure
        }
    }()
    // ...
}
```

**Recommendation:** Add optional error return or logging:

```go
// Option A: Return error
func GetComponentTypeSafe[T any](entity *ecs.Entity, component *ecs.Component) (T, error) {
    var result T
    if entity == nil {
        return result, errors.New("nil entity")
    }

    c, ok := entity.GetComponentData(component)
    if !ok {
        return result, errors.New("component not found")
    }

    result, ok = c.(T)
    if !ok {
        return result, fmt.Errorf("type assertion failed: expected %T, got %T", result, c)
    }

    return result, nil
}

// Option B: Keep current API but add debug logging
var DebugComponentAccess = false

func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {
    defer func() {
        if r := recover(); r != nil {
            if DebugComponentAccess {
                log.Printf("GetComponentType panic: %v", r)
            }
        }
    }()
    // ...
}
```

---

## Implementation Roadmap

### Phase 1: Critical Performance (1-2 days)
1. Add `entityIndex` map to EntityManager
2. Update entity creation/disposal to maintain index
3. Refactor O(n) lookup functions to use index
4. Fix `containsEntity` in combat hot path

### Phase 2: Combat Optimization (1 day)
1. Add `IsDestroyed` flag to SquadData
2. Update squad destruction detection to set flag
3. Refactor `IsSquadDestroyed()` to use flag
4. Optimize `GetUnitIDsInRow()` to single query

### Phase 3: Type Safety (0.5 days)
1. Create `common/tagkeys.go` with TagKey constants
2. Update EntityManager to use `map[TagKey]ecs.Tag`
3. Search/replace all string key usages
4. Add tag validation function

### Phase 4: Cleanup (0.5 days)
1. Standardize on EntityManager wrapper
2. Update gear package functions
3. Review FindEntityByIDWithTag usages
4. Add debug logging for component access failures

---

## Metrics for Success

| Metric | Before | Target |
|--------|--------|--------|
| Entity lookup complexity | O(n) | O(1) |
| Combat containsEntity | O(n×k) | O(n) |
| IsSquadDestroyed queries | N per call | 0 (cached) |
| String tag keys | 13 | 0 (all constants) |
| Mixed Manager types | 2 packages | 0 |

---

## Testing Recommendations

1. **Benchmark entity lookups** before/after index implementation
2. **Profile combat sequences** to measure attack action performance
3. **Unit test tag registration** to catch missing tags early
4. **Integration test entity lifecycle** to verify index maintenance

---

## Appendix: File Reference

### Files to Modify

| File | Changes | Priority |
|------|---------|----------|
| `common/ecsutil.go` | Add entityIndex, refactor lookups | P1 |
| `combat/combatactionsystem.go` | Fix containsEntity | P1 |
| `combat/combatservices/combat_service.go` | Use cached destruction flag | P1 |
| `squads/squadcomponents.go` | Add IsDestroyed field | P1 |
| `squads/squadqueries.go` | Optimize GetUnitIDsInRow | P1 |
| `combat/combatqueries.go` | Use GlobalPositionSystem | P2 |
| `common/tagkeys.go` | NEW: Tag key constants | P2 |
| `gear/Inventory.go` | Use EntityManager wrapper | P3 |
| `gear/gearutil.go` | Use EntityManager wrapper | P3 |

### Files for Reference (Good Patterns)
- `systems/positionsystem.go` - O(1) spatial grid (exemplary)
- `squads/squadqueries.go` - Query-based patterns (exemplary)
- `squads/squadcomponents.go` - EntityID usage (exemplary)
