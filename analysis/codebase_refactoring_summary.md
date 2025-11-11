# Codebase Refactoring Summary
**Generated**: 2025-11-10
**Total Files Analyzed**: 88 Go files across 15 packages
**Analysis Scope**: Entire codebase including combat, GUI, worldmap, squads, gear, spawning, input, rendering, graphics

---

## Issue 1: Component Methods Violate ECS Architecture (Attributes)
**Priority**: HIGH
**Scope**: `common/commoncomponents.go` (13 methods on Attributes)
**Problem**: Attributes component has 13 calculation methods (GetPhysicalDamage, GetHitRate, etc.), violating "pure data components" principle. This couples game logic to data structures.
**Solution**: Extract all 13 methods to `common/attributesystems.go` as system functions taking `*Attributes` parameter. Replace `attr.GetPhysicalDamage()` with `GetPhysicalDamage(attr)`.
**Effort**: 4-6 hours (13 methods, ~50 call sites across codebase)

---

## Issue 2: Item Component Methods Violate ECS Pattern
**Priority**: MEDIUM
**Scope**: `gear/items.go` (5 methods on Item component)
**Problem**: Item component has 5 methods (GetAction, HasAction, GetThrowableAction, etc.), mixing logic with data. Already flagged as "ECS best practice compliant" but still has methods.
**Solution**: Move all 5 methods to `gear/itemqueries.go` as system functions. Item should be pure data only.
**Effort**: 2-3 hours (5 methods, ~20 call sites)

---

## Issue 3: PlayerData Uses Entity Pointers Instead of EntityIDs
**Priority**: HIGH
**Scope**: `common/playerdata.go` (3 entity pointer fields)
**Problem**: PlayerThrowable has `SelectedThrowable *ecs.Entity` and `ThrowableItemEntity *ecs.Entity`. PlayerData has `PlayerEntity *ecs.Entity`. Violates ECS best practice of using EntityIDs for references.
**Solution**: Convert all 3 entity pointer fields to `ecs.EntityID`. Update ~15 call sites in input, GUI, and spawning packages. Add helper functions for entity lookup.
**Effort**: 6-8 hours (impacts input controllers, inventory mode, exploration mode)

---

## Issue 4: Duplicate Entity Lookup Queries Across Packages
**Priority**: MEDIUM
**Scope**: `combat/queries.go`, `squads/squadqueries.go`, `gear/gearutil.go`, `common/ecsutil.go`
**Problem**: Each package reimplements `findEntityByID` with O(n) linear search. Found 7+ variations of same pattern: "for _, result := range manager.World.Query(tag) { if result.Entity.GetID() == entityID ..."
**Solution**: Create `common/entitylookup.go` with centralized `FindEntityByID(manager, entityID, tag)` function. Replace all 7+ duplicates with single implementation.
**Effort**: 2-3 hours (consolidate 7+ functions into 1 shared utility)

---

## Issue 5: Global Mutable State in Multiple Packages
**Priority**: MEDIUM
**Scope**: `worldmap/dungeongen.go`, `graphics/graphictypes.go`, `gui/guiresources.go`, `squads/squadmanager.go`
**Problem**: Found 8+ global variables with mutable state: `ValidPos ValidPositions`, `ScreenInfo`, `PanelRes`, `Units []UnitTemplate`. Hard to test, hidden dependencies, race condition risks.
**Solution**: Convert to dependency injection: pass ValidPositions to spawning functions, move ScreenInfo to UIContext, pass Units to squad creation functions. Only keep truly immutable globals (resources).
**Effort**: 8-10 hours (impacts spawning, worldmap, GUI initialization)

---

## Issue 6: GUI Package Complexity and Duplication
**Priority**: HIGH
**Scope**: `gui/` package (18 files, 4794 LOC, largest files: combatmode.go 987 LOC, squadbuilder.go 774 LOC)
**Problem**: Massive GUI files with duplicated patterns for list handling, detail panels, button creation. No shared abstractions for common UI patterns (squad lists appear in 4+ modes).
**Solution**: Extract shared components: `gui/listcomponent.go` for reusable list+detail pattern, `gui/squadui.go` for common squad display logic, `gui/buttonpatterns.go` for action button sets. Target 30% LOC reduction.
**Effort**: 12-16 hours (largest refactoring, high impact on maintainability)

---

## Issue 7: WorldMap ValidPos Global State Prevents Generator Flexibility
**Priority**: MEDIUM
**Scope**: `worldmap/dungeongen.go` (ValidPos global), `spawning/*.go` (3 files depend on it)
**Problem**: Global `ValidPos` variable updated by generators makes it impossible to generate multiple maps or test generators in parallel. Spawning system tightly coupled to this global state.
**Solution**: Return ValidPositions from generator interface alongside GenerationResult. Pass to spawning functions as parameter. Remove global variable entirely.
**Effort**: 4-5 hours (update generator interface, fix spawning calls)

---

## Issue 8: Lack of Entity Cleanup on Death/Removal
**Priority**: HIGH
**Scope**: Combat system, spawning, worldmap (entity lifecycle management)
**Problem**: No systematic entity cleanup when units die or items are removed. PositionSystem.RemoveEntity called manually but no guarantee. Entity pointers in PlayerData become dangling. Mentioned in CLAUDE.md as known bug.
**Solution**: Create `systems/entitylifecycle.go` with `DestroyEntity(entityID)` function that: removes from PositionSystem, clears from tiles, removes references, disposes ECS entity. Call from combat death, item pickup, level transitions.
**Effort**: 6-8 hours (centralized cleanup, fix 5+ entity removal sites)

---

## Issue 9: Circular Dependency Workarounds with interface{}
**Priority**: MEDIUM
**Scope**: `common/playerdata.go` (2 interface{} fields)
**Problem**: PlayerThrowable.ThrowingAOEShape and PlayerData.Inventory stored as `interface{}` to avoid circular imports. Type-unsafe, requires casts, hides real dependencies.
**Solution**: Restructure packages: move PlayerData to new `player/` package that can import both `graphics` and `gear`. Eliminates need for interface{} workarounds. Or use dependency injection through UIContext.
**Effort**: 6-8 hours (package restructuring, update all imports)

---

## Issue 10: Query Pattern Inconsistency and Performance
**Priority**: MEDIUM
**Scope**: All packages with ECS queries (95 query loops found)
**Problem**: Inconsistent patterns for ECS queries: some use tags, some use BuildTag() inline, some cache results, some don't. No consistent error handling for missing components. Performance varies widely.
**Solution**: Establish query patterns: (1) Cached tags for frequent queries, (2) Helper functions for common patterns in each package, (3) Consistent nil-checking. Document in ECS best practices guide.
**Effort**: 5-6 hours (create patterns, update documentation, spot-fix worst cases)

---

## SUMMARY STATISTICS

### ECS Violations Found
- **Component methods**: 18 total (13 Attributes, 5 Item)
- **Entity pointers**: 3 in PlayerData, 24 files total use `*ecs.Entity`
- **Mixed concerns**: Attributes calculations, Item action lookups

### Code Duplication
- **Entity lookup**: 7+ duplicate implementations
- **GUI patterns**: List+detail pattern in 4+ modes
- **Query helpers**: 3 packages reimplement similar query functions

### Global State Issues
- **Mutable globals**: 8+ variables across 4 packages
- **Hidden dependencies**: ValidPos affects 3+ spawning functions
- **Testing blockers**: Cannot test generators in isolation

### Architecture Debt
- **Circular dependencies**: 2 interface{} workarounds
- **Missing abstractions**: Entity lifecycle, GUI components
- **Inconsistent patterns**: 95 query loops with varying styles

---

## RECOMMENDED PRIORITIZATION

### Phase 1: Critical ECS Violations (16-22 hours)
1. Issue 1: Attributes component methods → system functions
2. Issue 3: PlayerData entity pointers → EntityIDs
3. Issue 8: Entity cleanup system

### Phase 2: Code Quality and Duplication (19-24 hours)
4. Issue 4: Centralized entity lookup
5. Issue 6: GUI component extraction
6. Issue 10: Query pattern standardization

### Phase 3: Architecture Improvements (18-23 hours)
7. Issue 5: Global state removal
8. Issue 7: ValidPos dependency injection
9. Issue 9: Circular dependency fixes
10. Issue 2: Item component methods (lower priority, less impact)

**Total Estimated Effort**: 53-69 hours for all issues

---

## NOTES

**Strengths Observed:**
- Squad system is exemplary ECS (2675 LOC, perfect data/logic separation)
- Inventory system follows ECS best practices post-refactoring
- Position system O(1) spatial grid is excellent
- Generator strategy pattern is well-implemented

**Anti-Patterns to Avoid:**
- Do NOT add more methods to components
- Do NOT use entity pointers for references
- Do NOT create more global mutable state
- DO follow squad/inventory system as templates

**Quick Wins (< 4 hours each):**
- Issue 4: Centralized entity lookup (2-3h, immediate benefit)
- Issue 2: Item component methods (2-3h, completes gear package)
- Parts of Issue 10: Document query patterns (2h, prevents future issues)

---

END OF ANALYSIS
