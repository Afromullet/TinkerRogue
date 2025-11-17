# TinkerRogue Codebase Refactoring Analysis

**Date:** 2025-11-16
**Codebase:** ~20,700 LOC across 97 Go files, 14 packages
**Focus:** High-value refactorings that reduce duplication, improve maintainability, and enhance extensibility

---

## Executive Summary

Three critical areas need attention:
1. **GUI Package Organization** - 30 files in flat structure, needs subpackage split
2. **Duplicate Query Patterns** - ~150 LOC of similar entity-finding logic across 4 packages
3. **Type Safety Issues** - `interface{}` usage defeating Go's type system

**Recommended First Sprint:** Entity query consolidation + Spawning config extraction (6-9 hours, high ROI)

---

## Package Structure Overview

| Package | Files | Responsibility | Health |
|---------|-------|----------------|--------|
| `gui/` | 30 | UI modes, widgets, factories | ⚠️ Too large, needs split |
| `combat/` | 9 | Turn-based combat system | ✅ Well structured |
| `squads/` | 11 | Squad management | ✅ Good ECS patterns |
| `worldmap/` | 10 | Map generation | ✅ Clean strategy pattern |
| `common/` | 5 | Core ECS types | ⚠️ interface{} issues |
| Others | 22 | Various subsystems | ✅ Reasonable |

---

## High Priority Refactorings

### 1. GUI Package Restructuring (6-8h)

**Problem:** 30 files flat in `gui/` package - unclear boundaries, difficult navigation

**Proposed Structure:**
```
gui/
├── modes/           # combatmode.go, explorationmode.go, etc. (8 files)
├── factories/       # combat_ui_factory.go, panel_factory.go (3 files)
├── components/      # guicomponents.go, detail panels, lists (4 files)
├── resources/       # guiresources.go, layout_constants.go (3 files)
└── *.go            # modemanager.go, guiqueries.go, uimode.go (core files)
```

**Benefits:**
- Clear separation of concerns (modes vs factories vs resources)
- Easier to locate specific UI code
- Better IDE navigation and comprehension

**Impact:** Cleaner imports, reduced cognitive load when working in GUI

---

### 2. Duplicate Entity Query Patterns (4-6h)

**Problem:** 4 packages implement nearly identical "find entity by X" logic

**Current Duplication:**
- `combat/queries.go` - 10+ finder functions (~95 LOC)
- `squads/squadqueries.go` - Squad-specific queries
- `gear/gearutil.go` - Item finding logic
- `gui/guiqueries.go` - GUI data queries

**All follow this pattern:**
```go
// Iterate entities → filter by component → return match
for result := range manager.Query(tag) {
    if data.FieldName == targetValue { return entity }
}
```

**Consolidation Target:**
```go
// common/entityqueries.go (new file)
func FindEntityByID(manager, entityID) *ecs.Entity
func FindEntitiesByComponent(manager, component) []*ecs.Entity
func QueryWithFilter(manager, tag, predicate) []*ecs.Entity
```

**Benefits:**
- Eliminate ~100-150 LOC duplication
- Single source of truth for entity lookups
- Easier to optimize (add caching once, benefits all)

**Note:** Domain-specific queries (e.g., "find squads in faction") remain in their packages

---

### 3. Type Safety - Fix interface{} Anti-pattern (4-6h)

**Problem:** `common/playerdata.go` uses `interface{}` to avoid circular dependencies

**Current Code:**
```go
PlayerData struct {
    Inventory interface{}  // Actually *gear.Inventory
    ThrowingAOEShape interface{}  // Actually graphics type
}
```

**Root Cause:** playerdata needs gear types, but gear imports common (circular)

**Solution:** Dependency injection pattern
```go
// Store EntityID instead of data
PlayerData struct {
    PlayerEntityID ecs.EntityID
}

// Methods access via manager (breaks circular dependency)
func (p *PlayerData) GetInventory(em *EntityManager) *gear.Inventory {
    return common.GetComponentTypeByID[*gear.Inventory](em, p.PlayerEntityID, gear.InventoryComponent)
}
```

**Benefits:**
- Compile-time type safety (no more runtime assertions)
- Cleaner architecture (no circular deps)
- Follows ECS best practices (EntityID references, not pointers)

---

## Medium Priority Refactorings

### 4. Spawning Configuration Extraction (2-3h)

**Problem:** Hardcoded spawn values scattered across 3 files

**Examples:**
- `spawning/spawnmonsters.go:16` - "30% spawn chance"
- `spawning/loottables.go:12-39` - Quality distribution (50/40/10)
- `spawning/spawnloot.go:67-69` - Drop rates

**Solution:** Create `spawning/config.go`
```go
type SpawnConfig struct {
    MonsterSpawnChance int            // 30
    SpawnTurnsInterval int            // 10
    LootQualities      map[Quality]int // Common:50, Uncommon:40, Rare:10
    ThrowableEffects   []EffectWeight
}
```

**Benefits:**
- Easy game balance tuning (one file)
- Potential for config file externalization
- Self-documenting spawn probabilities

---

### 5. Unified Component Initialization (2h)

**Problem:** Component initialization scattered across 5 files:
- `game_main/componentinit.go` - Core components
- `combat/components.go` - InitCombatComponents()
- `squads/components.go` - InitSquadComponents()
- `gear/stateffect.go` - Init functions
- Forgetting one init causes silent failures

**Solution:** Single `InitializeAllComponents()` in componentinit.go
```go
func InitializeAllComponents(manager) {
    registerCoreComponents()
    registerCombatComponents()
    registerSquadComponents()
    registerGearComponents()
    buildAllTags()
}
```

**Benefits:**
- Impossible to forget initialization
- Clear ordering/dependencies
- Easier testing (single setup call)

---

### 6. Standardize Component Access Pattern (2h)

**Problem:** Mix of verbose and clean patterns

**Current (verbose):**
```go
// rendering/rendering.go:26-29, 70-73
pos := result.Components[common.PositionComponent].(*coords.LogicalPosition)
```

**Target (consistent with squad system):**
```go
pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
```

**Files to update:**
- `rendering/rendering.go:26-29,70-73` (6 instances)
- `gear/itemactions.go:77` (1 instance)
- `input/combatcontroller.go:187` (1 instance)

---

### 7. Combat/Squads Circular Dependency (3-4h)

**Problem:** combat imports squads, squads imports combat (risky)

**Solution:** Extract shared interfaces to neutral package
```go
// common/combatinterfaces.go (new file)
type CombatUnit interface {
    GetID() ecs.EntityID
    GetHP() int
    TakeDamage(amount int)
}
```

**Benefits:**
- Break circular dependency
- Easier to test in isolation
- Clearer contracts between systems

---

### 8. Standardize Query Patterns (2-3h)

**Problem:** Three different patterns in use

**Current:**
```go
// Pattern A: On-the-fly tag building
manager.Query(ecs.BuildTag(ItemComponent))

// Pattern B: Pre-built tags (PREFERRED)
manager.World.Query(MapPositionTag)

// Pattern C: Registry lookup
manager.World.Query(manager.Tags["squad"])
```

**Recommendation:** Standardize on Pattern B (pre-built tags)
- Already done: combat, squad systems
- Apply to: worldmap, gear, spawning packages

**Benefits:**
- Performance (no runtime tag building)
- Consistency across codebase
- Self-documenting (see all tags at top of file)

---

## Low Priority / Polish

### 9. Remove Debug Code

**Files:**
- `gui/squaddeploymentmode.go:169-273` - 20+ fmt.Printf() debug statements

**Action:** Remove or gate behind `DEBUG_MODE` constant

---

### 10. Resolve TODOs (20+ scattered)

**Examples:**
- `coords/cordmanager.go:41` - Screen data optimization
- `spawning/spawnthrowable.go:12,46,76` - Config externalization
- `combat/victory.go:39,63,71` - Faction/objective implementation
- `input/combatcontroller.go:183` - Squad effect application

**Action:** Convert to GitHub issues or resolve

---

### 11. Parameter Naming Consistency

**Problem:** Inconsistent naming conventions
- `ecsManager` vs `manager` vs `ecsmanager` (typo in ecsutil.go)

**Action:** Standardize to Go style (`ecsManager` - camelCase)

---

### 12. Add Test Coverage

**Current State:**
- Only 5 test files for 97 Go files (~5% coverage)
- Tested: squads, combat, positionsystem, capacity
- **Missing:** gui (0 tests), spawning (0 tests), worldmap (0 tests), gear (0 tests)

**Recommendation:** Focus on high-risk areas (combat, squads - already done ✓)

---

## Implementation Roadmap

### Sprint 1 (6-9 hours) - Quick Wins, High ROI
1. ✅ **COMPLETED (2025-11-16)** Extract common entity query helpers (4-6h)
2. Spawning configuration extraction (2-3h)

**Sprint 1.1 Results:**
- Created `common/entityqueries.go` with 6 composable query helpers
- Refactored 13 functions across 3 packages (combat, squads, gui)
- Eliminated ~90 lines of iteration boilerplate
- All tests passing, build successful

**Expected Impact:** Eliminate ~150 LOC duplication, improve game balance workflow

### Sprint 2 (8-10 hours) - Structural Improvements
3. Fix interface{} in PlayerData (4-6h)
4. Unified component initialization (2h)
5. Standardize component access (2h)

**Expected Impact:** Type safety, prevent silent failures, consistency

### Sprint 3 (8-10 hours) - Organization & Polish
6. GUI package restructuring (6-8h)
7. Standardize query patterns (2-3h)

**Expected Impact:** Better navigation, reduced cognitive load

### Future (As Needed)
- Combat/Squads circular dependency refactoring
- Add test coverage for untested packages
- Data-driven ability system (extensibility)
- Formation presets (already scheduled - 4-6h)

---

## Summary

**Total Refactoring Effort:** 22-29 hours across high/medium priority items

**Biggest Value:**
1. Entity query consolidation - eliminates most code duplication
2. GUI restructuring - long-term maintainability
3. Type safety fixes - prevents runtime errors

**Simplification Focus:**
- Entity query duplication (4 packages → 1 helper)
- Spawning config (scattered → centralized)
- Component init (5 files → 1 entry point)

**Maintainability Focus:**
- GUI package split (30 flat → organized subpackages)
- Type safety (interface{} → proper types)
- Consistent patterns (3 query styles → 1)

**Extensibility Focus:**
- Spawning config externalization (hardcodes → config file ready)
- Unified init (easy to add new subsystems)
- Clear package boundaries (easier to extend individual systems)
