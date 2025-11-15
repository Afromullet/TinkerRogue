# TinkerRogue Refactoring Opportunities

**Analysis Date:** 2025-11-15
**Codebase:** ~20,700 LOC across 97 Go files

---

## High Priority (ECS Violations & Type Safety)




### 2. interface{} Anti-pattern (common/playerdata.go)
**Problem:** Defeats type safety to avoid circular dependencies
```go
// Current (WRONG)
PlayerData struct {
    Inventory interface{}  // Actually *gear.Inventory
}
PlayerThrowable struct {
    ThrowingAOEShape interface{}  // Actually graphics type
}
```
**Impact:** Runtime type assertions everywhere, no compile-time safety
**Solution:** Break circular deps via dependency injection or package restructure

---

### 3. Inconsistent result.Components Access (5+ files)
**Problem:** Mix of patterns for getting components
```go
// Current (verbose)
pos := result.Components[common.PositionComponent].(*coords.LogicalPosition)

// Should use (like squad system)
pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
```
**Files:** `rendering/rendering.go:26-29,70-73`, `gear/itemactions.go:77`, `input/combatcontroller.go:185`

---

## Medium Priority (Duplication & Inconsistency)

### 4. Entity Finding Boilerplate (4 packages)
**Problem:** Every package reimplements "find entity by ID" pattern
- `combat/queries.go`: 10+ finder functions (lines 13-95)
- `squads/squadqueries.go`: GetSquadEntity (line 59)
- `gear/gearutil.go`: FindItemEntityByID (lines 16-54)
- `gui/guiqueries.go`: GetSquadAtPosition (lines 200-232)

**Solution:** Create unified `common.FindEntityByID(manager, id, ...requiredComponents)` helper
**Savings:** ~50 LOC, centralized optimization

---

### 5. Scattered Component Initialization
**Problem:** No unified init sequence
- `game_main/componentinit.go`
- `combat/components.go`: InitCombatComponents()
- `squads/components.go`: InitSquadComponents()
- `gear/stateffect.go`: Init functions

**Impact:** Fragile - forgetting one Init() breaks silently
**Solution:** Single `InitializeAllComponents()` entry point

---

### 6. Three Different Query Patterns
**Problem:** Inconsistent tag usage
```go
// Pattern A - on-the-fly
itemTag := ecs.BuildTag(ItemComponent)
manager.Query(itemTag)

// Pattern B - pre-built
manager.World.Query(MapPositionTag)

// Pattern C - registry
manager.World.Query(manager.Tags["squad"])
```
**Solution:** Pick one pattern (prefer pre-built Tags), enforce across codebase

---

### 7. WorldTags vs Tags Confusion (input/combatcontroller.go:184)
**Problem:** `ecsManager.WorldTags["monsters"]` vs `ecsManager.Tags["monsters"]`
**Impact:** Two tag namespaces; unclear which is correct

---

### 8. Error Handling Inconsistency
**Problem:** Mix of patterns
- Some return errors
- Some return nil silently
- Some use `log.Fatal()` panics
- Some use defer/recover

**Solution:** Document standard error handling approach

---

## Low Priority (Polish)

### 9. Debug Code in Production
**Files:** `gui/squaddeploymentmode.go:169-273` (debug printf statements)
**Action:** Remove before release

---

### 10. TODOs (20+ scattered)
**Examples:**
- `coords/cordmanager.go:41` - optimization TODO
- `gear/gearutil.go:18` - BuildTag usage change
- `spawning/spawnthrowable.go:12` - config externalization

**Action:** Convert to GitHub issues or resolve

---

### 11. Parameter Naming (typo in common/ecsutil.go)
**Problem:** `ecsmnager` typo, inconsistent `ecsManager` vs `manager` vs `ecsmanager`
**Action:** Standardize to Go style: `ecsManager`

---

## Feature Extensibility

### 12. Hard-coded Ability System (squads/components.go:242-250)
**Problem:** Adding abilities requires code changes
```go
const (
    AbilityRally = iota
    AbilityHeal
    AbilityBattleCry
    AbilityFireball  // Can't add new abilities via data
)
```
**Solution:** Data-driven ability definitions (JSON/TOML config)

---

### 13. Rigid Entity Templates (entitytemplates/creators.go)
**Problem:** No plugin system; all templates hardcoded
**Impact:** Limits moddability

---

## Actionable Roadmap

### Sprint 1 (4-6 hours)
1. ✅ Convert `common/playerdata.go` entity pointers → EntityIDs
2. ✅ Create `common.FindEntityByID()` unified helper
3. ✅ Standardize to `GetComponentType[T]()` across all files

### Sprint 2 (6-8 hours)
4. ✅ Fix interface{} in PlayerData (break circular deps)
5. ✅ Single `InitializeAllComponents()` function
6. ✅ Standardize query pattern (choose one, refactor all)
7. ✅ Fix WorldTags/Tags confusion

### Sprint 3 (4-6 hours)
8. ✅ Remove debug printf statements
9. ✅ Resolve or ticket all TODOs
10. ✅ Document error handling standard

### Future Enhancements
- Data-driven ability system
- Equipment system ECS refactor (already noted in CLAUDE.md)
- Formation presets (already scheduled - 4-6h)

---

## Summary

**Simplification:** Entity finding duplication (4 packages → 1 helper)
**Maintainability:** Unified component init, standardized error handling
**Consistency:** Single query pattern, fix Tags confusion, standardize naming
**Extensibility:** Data-driven abilities, entity template plugins
**Duplication:** Entity finders, component access patterns

**Total Estimated Effort:** 14-20 hours across 3 sprints for high/medium priority items
