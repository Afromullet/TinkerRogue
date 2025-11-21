# TinkerRogue Master Development Roadmap

**Version:** 5.0 - POST-WORLDMAP & GUI REFACTOR | **Updated:** 2025-11-21
**Status:** Squad System 98% Complete, Worldmap Strategy Pattern Complete, GUI Reorganization In Progress

---

## Executive Summary

This roadmap reflects major architectural completions and ongoing GUI reorganization work. All core systems are functional and well-architected.

**What Changed from v4.0:**
- **Worldmap Generator System** → **COMPLETE** (Strategy pattern with 2 algorithms, 2025-11-08)
- **GUI Package Refactoring** → **IN PROGRESS** (Combat/squad modes extracted to subpackages, 2025-11-20)
- Formation Presets → **VERIFIED COMPLETE** (4 presets with full implementations)
- Squad System → **REMAINS 98%** (only graphical rendering and spawning hookup remain)

---

## Current State Summary

### What's ACTUALLY Complete ✅

**Core Infrastructure (100% Complete):**
- ✅ **Input System** - Unified InputCoordinator with 3 specialized controllers (movementcontroller.go, combatcontroller.go, uicontroller.go)
- ✅ **Coordinate System** - Type-safe LogicalPosition/PixelPosition with CoordinateManager (coords package, 2 files)
- ✅ **Entity Templates** - Generic factory pattern with EntityType enum (entitytemplates package, working implementation)
- ✅ **Graphics Shapes** - BaseShape consolidation with 3 variants (drawableshapes.go, 391 LOC)
- ✅ **Position System** - O(1) spatial grid with value-based keys (positionsystem.go, 183 LOC, tested)
- ✅ **Inventory System** - Pure ECS: EntityIDs, system functions, no pointers (Inventory.go, 245 LOC)
- ✅ **Worldmap Generator System** - Strategy pattern with pluggable algorithms (2025-11-08)
  - MapGenerator interface with registry system
  - 2 algorithms: rooms_corridors (default), tactical_biome (5 biomes, cellular automata)
  - Files: generator.go, gen_rooms_corridors.go, gen_tactical_biome.go, gen_helpers.go
  - Removed 180 LOC from GameMap, fixed global state issues
  - Critical: Uses CoordinateManager for result.Tiles indexing

**Squad System (98% Complete - 4951 LOC total):**
- ✅ **Components** (components.go, 331 LOC) - 8 ECS components with perfect data/logic separation
- ✅ **Query System** (squadqueries.go, 286 LOC) - 7 query functions + capacity/range queries
- ✅ **Combat System** (squadcombat.go, 424 LOC) - ExecuteSquadAttack with hit/dodge/crit/cover mechanics
- ✅ **Ability System** (squadabilities.go, 317 LOC) - CheckAndTriggerAbilities with 4 abilities, 5 trigger types
  - Auto-triggers: HP threshold, turn count, enemy count, morale, combat start
  - Abilities: Rally, Heal, Battle Cry, Fireball
  - Cooldown tracking and once-per-combat flags
  - **INTEGRATED**: Turn manager calls CheckAndTriggerAbilities at combat start and turn reset
- ✅ **Formation System** (squadcreation.go, 378 LOC) - CreateSquadFromTemplate, AddUnitToSquad, MoveUnitInSquad
  - 4 formation presets: Balanced (5 units), Defensive (5 units), Offensive (5 units), Ranged (6 units)
  - Multi-cell unit support, collision detection
  - Capacity system with leader-based limits
- ✅ **Visualization** (visualization.go, 176 LOC) - Text-based 3x3 grid rendering
- ✅ **Testing** (squads_test.go, squadcombat_test.go, capacity_test.go) - Comprehensive test suite, all passing
- ✅ **Turn Integration** (combat/turnmanager.go) - Squad abilities called at combat start and turn reset

### What's Remaining ❌

**Squad System (2% remaining):**
- ❌ **Graphical Rendering** - GUI integration for squad visualization (text-only currently)
- ❌ **Enemy Spawning** - SpawnEnemySquad with level scaling (logic exists, needs hooking up)

**GUI System Refactoring (60% complete - 2025-11-20 analysis):**
- ✅ **Mode Extraction Complete** - Combat/squad modes split into subpackages (guicombat/, guisquads/, guimodes/)
- ✅ **Package Structure** - 9 directories, 6,398 LOC organized by concern
- ✅ **Core Infrastructure** - UIMode interface, ModeManager, BaseMode pattern established
- ⚠️ **Remaining Issues** (identified in gui_refactoring_analysis.md):
  - ❌ Empty `gui/components/` directory (delete immediately)
  - ❌ Mixed responsibilities in `guicomponents/` (UI components + ECS queries, needs split)
  - ❌ BaseMode location in root package (should move to gui/core or gui/base)
  - ❌ Global state in `guiresources/` (needs dependency injection)
  - ⚠️ Button factory pattern partially applied (needs consistency across all modes)

**Status Effects (85% complete - LOW PRIORITY):**
- ✅ StatusEffects interface (stateffect.go, 381 LOC)
- ✅ 3 effects implemented: Burning, Freezing, Sticky
- ❌ Quality interface extraction (deferred - not blocking)

**Bug Fixes and Polish:**
- ❌ Fix throwable AOE movement issue
- ❌ Ensure entities removed on death
- ❌ Don't allow shooting/throwing through walls
- ❌ Throwing accuracy/miss chance system
- ❌ Level transitions cleanup
- ❌ Add level variety (tile types, visual diversity)

---

## Reality Check: Claimed vs Actual Status (v3.0 → v5.0)

| Claim (Previous Versions) | Actual Status (v5.0) | Evidence |
|---------------------------|---------------------|----------|
| Ability System "NOT STARTED (8-10h)" | **COMPLETE** | squadabilities.go (317 LOC), integrated in turnmanager.go, tests passing |
| Formation Presets "40% COMPLETE (4-6h)" | **COMPLETE** | GetFormationPreset() returns 4 presets (Balanced/Defensive/Offensive/Ranged) with full implementations |
| Map Integration "NOT STARTED (4-6h)" | **COMPLETE** | turnmanager.go calls CheckAndTriggerAbilities, combat system operational |
| Squad System "85% complete" | **98% complete** | Only graphical rendering and spawning hookup remain |
| Worldmap Generator "single monolithic function" | **STRATEGY PATTERN COMPLETE** | MapGenerator interface, 2 algorithms, registry system (2025-11-08) |
| GUI "monolithic package" | **60% REFACTORED** | Split into 9 subpackages, 4 architectural issues remain (2025-11-20) |

---

## Time Estimates (REALISTIC)

**Total Remaining:** 10-16 hours (conservative)

### Breakdown by Priority

**High Priority (Core Functionality):**
- Squad Graphical Rendering: 2-3h (sprites, HP bars, role icons)
- Enemy Squad Spawning: 1-2h (hookup existing logic to level system)
- Bug Fixes: 2-3h (throwable AOE, entity cleanup, wall collision)

**Medium Priority (Architecture & Polish):**
- GUI Refactoring Completion: 3-5h (split guicomponents, relocate BaseMode, fix globals)
- Polish: 1-2h (accuracy, level transitions, visual variety)

**Low Priority (Optional):**
- Status Effects Quality Interface: 1-2h (deferred, not blocking)

---

## Phase 1: Squad System Core (98% → 100%)

### 1.1 Query System ✅ COMPLETE
**File:** squads/squadqueries.go (286 LOC)
All 7 core functions + capacity queries + range queries implemented and tested.

### 1.2 Combat System ✅ COMPLETE
**File:** squads/squadcombat.go (424 LOC)
ExecuteSquadAttack fully operational with:
- Row-based and cell-based targeting
- Hit/dodge/crit mechanics
- Cover system (stacking)
- Multi-cell unit support
- Range checking

### 1.3 Ability System ✅ COMPLETE
**File:** squads/squadabilities.go (317 LOC)
**Deliverables:**
- ✅ CheckAndTriggerAbilities(squadID, manager)
- ✅ 5 trigger types: HP threshold, turn count, combat start, enemy count, morale
- ✅ 4 abilities: Rally (+5 dmg, 3 turns), Heal (10 HP), Battle Cry (+3 dmg, +10 morale), Fireball (15 dmg AOE)
- ✅ Cooldown management with once-per-combat tracking
- ✅ Integration with turn manager (combat start + turn reset)

### 1.4 Formation System ✅ COMPLETE
**File:** squads/squadcreation.go (378 LOC)
**Deliverables:**
- ✅ CreateEmptySquad() - creates squad entity
- ✅ AddUnitToSquad() - validates capacity, creates entity, updates grid
- ✅ CreateSquadFromTemplate(manager, formation, unitTemplates) - full squad creation
- ✅ 4 formation presets: Balanced (2F/1S/2B), Defensive (3F/1S/1B), Offensive (1F/3M/1B), Ranged (1F/2M/1S/2B)
- ✅ Grid collision detection for multi-cell units
- ✅ RemoveUnitFromSquad(), MoveUnitInSquad()

### 1.5 Testing ✅ COMPLETE
**Files:** squads/squads_test.go, squadcombat_test.go, capacity_test.go
Comprehensive test suite: 30+ tests covering combat, abilities, capacity, queries.

---

## Phase 2: Map Integration ✅ COMPLETE

### 2.1 Turn Manager Integration ✅ COMPLETE
**File:** combat/turnmanager.go
- ✅ CheckAndTriggerAbilities called at combat start (line 42)
- ✅ CheckAndTriggerAbilities called at turn reset (line 85)
- ✅ Movement system queries squad speed
- ✅ Action state tracking per squad

### 2.2 Rendering ⚠️ PARTIAL (20% remaining)
- ✅ Text-based visualization complete (visualization.go)
- ✅ GUI mode system exists (squadmanagementmode.go, combatmode.go)
- ❌ Graphical rendering needed (sprites, HP bars, role icons, row highlighting)

### 2.3 Spawning ⚠️ NEEDS HOOKUP (10% remaining)
- ✅ CreateSquadFromTemplate function exists
- ✅ Entity template system operational
- ❌ SpawnEnemySquad function needs creation (trivial wrapper)
- ❌ Level scaling logic needs hookup

---

## Phase 3: Worldmap Generator System ✅ COMPLETE

### 3.1 Strategy Pattern Implementation ✅ COMPLETE (2025-11-08)
**Files:** worldmap/generator.go, gen_rooms_corridors.go, gen_tactical_biome.go, gen_helpers.go

**Deliverables:**
- ✅ MapGenerator interface with 3 methods (GenerateTiles, Name, Description)
- ✅ Registry system with RegisterGenerator() and ListGenerators()
- ✅ 2 algorithms implemented:
  - rooms_corridors: Classic roguelike (default)
  - tactical_biome: Cellular automata with 5 biomes for squad combat
- ✅ Removed 180 LOC from GameMap by extracting generation logic
- ✅ Fixed global state issues (TileImageSet replaces global vars)
- ✅ Fixed ECS violation (TileContents uses EntityIDs, not entity pointers)
- ✅ CoordinateManager integration for correct indexing

**Benefits Achieved:**
- Open/Closed Principle: Add new algorithms without modifying existing code
- Each generator independently testable
- Zero changes to existing code required for new generators
- Critical bug fix: CoordinateManager prevents index out of range panics

**Usage:**
```go
// Default generator
gameMap := worldmap.NewGameMapDefault()

// Specify algorithm
gameMap := worldmap.NewGameMap("tactical_biome")

// List available
generators := worldmap.ListGenerators()
```

---

## Phase 4: GUI Package Refactoring ⚠️ IN PROGRESS

### 4.1 Mode Extraction ✅ COMPLETE (2025-11-20)
**Analysis:** analysis/gui_refactoring_analysis.md

**Deliverables:**
- ✅ Split monolithic GUI into 9 subpackages (6,398 LOC organized)
- ✅ guicombat/ - Combat mode system (1,179 LOC)
- ✅ guisquads/ - Squad modes: builder, deployment, management (1,541 LOC)
- ✅ guimodes/ - General modes: exploration, inventory, info (770 LOC)
- ✅ Core infrastructure: UIMode interface, ModeManager, BaseMode pattern
- ✅ Dependency flow properly structured

### 4.2 Architectural Cleanup ⚠️ REMAINING (3-5h)
**Issues Identified:**
- ❌ Delete empty `gui/components/` directory (1 minute)
- ❌ Split `guicomponents/` into UI components and ECS queries (2-3h)
  - Current: Mixed responsibilities (571 LOC UI + 313 LOC queries)
  - Target: Separate `gui/components/` and `gui/queries/` packages
- ❌ Relocate BaseMode from root package to gui/core or gui/base (1h)
- ❌ Refactor global state in `guiresources/` (1-2h)
  - Replace global variables with dependency injection
- ⚠️ Standardize button factory pattern across all modes (optional, 1-2h)

---

## Phase 5: Polish and Bug Fixes (0% complete)

### Bug Fixes (2-3h)
- [ ] Fix throwable AOE movement issue
- [ ] Ensure entities removed on death
- [ ] Don't allow shooting/throwing through walls

### Features (1-2h)
- [ ] Throwing accuracy/miss chance
- [ ] Level transitions cleanup
- [ ] Add level variety (tile types, visual diversity)

---

## Success Metrics

### Phase 1 Complete ✅
- ✅ Can create squads programmatically
- ✅ Squad combat works without map
- ✅ Row targeting, multi-cell units work
- ✅ Abilities trigger automatically
- ✅ Formation presets generate valid squads
- ✅ All unit tests pass
- ✅ Integrated with turn manager

### Phase 2 Complete (98%)
- ✅ Combat initiated via turn system
- ✅ Turn manager tracks squad actions
- ✅ Abilities auto-trigger during combat
- ⚠️ Squad grid renders (text only, graphical pending)
- ❌ Enemy squads spawn at levels (needs hookup)

### Phase 3 Complete (Worldmap) ✅
- ✅ Strategy pattern allows plug-and-play algorithms
- ✅ Two distinct generators (classic roguelike + tactical biomes)
- ✅ No global state, proper ECS compliance
- ✅ CoordinateManager prevents index bugs
- ✅ Open for extension, closed for modification

### Phase 4 Progress (GUI) ⚠️
- ✅ Mode system properly organized into subpackages
- ✅ Clean dependency flow established
- ✅ Core infrastructure (UIMode, ModeManager) solid
- ⚠️ Architectural issues identified and documented
- ❌ Some mixed responsibilities and globals remain

### Overall System Success
- ✅ Multi-squad tactical gameplay foundation
- ✅ Squad building with formations
- ✅ Multi-cell units add variety
- ✅ 100% ECS pattern compliance across core systems
- ✅ Stable performance with spatial grid
- ✅ Pluggable map generation architecture
- ⚠️ GUI architecture improving but not yet perfect

---

## ECS Best Practices (Reference)

**Verified Implementations:**
1. ✅ Pure data components - Zero logic methods (squads/components.go)
2. ✅ Native EntityID - No pointers (all squad files use ecs.EntityID)
3. ✅ Query-based relationships - Discover via ECS queries (squadqueries.go)
4. ✅ System-based logic - All behavior in systems (squadcombat.go, squadabilities.go)
5. ✅ Value map keys - O(1) performance (systems/positionsystem.go)

**Reference Implementations:**
- `squads/*.go` - 4951 LOC, 8 components, 7+ query functions, system-based combat + abilities
- `gear/Inventory.go` - 245 LOC, pure data component, 9 system functions
- `systems/positionsystem.go` - 183 LOC, O(1) spatial grid

---

## Next Steps (Priority Order)

### Immediate Priority (High Impact - 6-8h)
1. **Squad Graphical Rendering** (2-3h)
   - Integrate squad sprites with existing GUI modes
   - HP bars, role icons, row highlighting
   - Leverages completed GUI mode infrastructure
2. **Enemy Squad Spawning** (1-2h)
   - Create SpawnEnemySquad wrapper function
   - Hook up to level generation system
   - Use completed formation presets
3. **Critical Bug Fixes** (2-3h)
   - Fix throwable AOE movement issue
   - Ensure entities removed on death
   - Block shooting/throwing through walls

### Medium Priority (Architecture - 3-5h)
4. **GUI Architectural Cleanup** (3-5h)
   - Delete empty `gui/components/` directory (1 min)
   - Split `guicomponents/` into separate concerns (2-3h)
   - Relocate BaseMode to proper package (1h)
   - Refactor globals in `guiresources/` (1-2h)

### Low Priority (Polish - 2-4h)
5. **Game Polish** (1-2h)
   - Throwing accuracy/miss chance
   - Level transitions cleanup
   - Visual variety (tile types, diversity)
6. **Status Effects Quality** (1-2h, optional)
   - Interface extraction for quality system

---

## Milestones

### Milestone 1: Squad System Complete (6-8h remaining)
- ✅ Combat, abilities, formations, testing all complete
- ⚠️ Squad graphical rendering (2-3h)
- ⚠️ Enemy spawning hookup (1-2h)
- ⚠️ Critical bug fixes (2-3h)

### Milestone 2: Architecture Complete (3-5h after M1)
- ✅ Worldmap strategy pattern complete
- ✅ GUI mode extraction complete
- ⚠️ GUI architectural cleanup (3-5h)

### Milestone 3: Game Polish (2-4h after M2)
- Polish features, visual variety, accuracy systems
- Status effects quality improvements (optional)

---

## Risk Mitigation

**Low Risk Areas (Verified Complete):**
- Core combat system operational and tested
- Query system complete with full test coverage
- Position system optimized (50x performance validated)
- Ability system integrated and functional

**No Significant Risks Remaining:**
- Remaining work is polish and integration (low complexity)
- Test suite provides safety net for changes
- ECS architecture proven stable

---

## Key Achievements Summary (v5.0)

### Major Systems Complete
1. **Squad System (98%)** - 4,951 LOC of ECS-compliant tactical combat
   - All core systems operational: components, queries, combat, abilities, formations
   - 30+ tests passing, turn manager integration complete
   - Only graphical rendering and enemy spawning hookup remain

2. **Worldmap Generator (100%)** - Strategy pattern implementation
   - Pluggable algorithm system with 2 generators
   - Fixed critical indexing bugs via CoordinateManager
   - Removed 180 LOC of monolithic code, eliminated global state

3. **Core Infrastructure (100%)** - Foundation is rock-solid
   - Input system, coordinates, entity templates, graphics shapes
   - Position system with O(1) performance
   - Inventory system as ECS reference implementation

4. **GUI Refactoring (60%)** - Significant progress on organization
   - Split into 9 logical subpackages (6,398 LOC)
   - Mode system properly extracted and organized
   - Architectural issues identified with clear remediation path

### Lines of Code by System
- Squad System: 4,951 LOC (components, queries, combat, abilities, formations, testing)
- GUI Package: 6,398 LOC (9 subpackages with mode system)
- Worldmap Generators: ~800 LOC (strategy pattern + 2 algorithms)
- Core Infrastructure: ~2,000 LOC (input, coords, templates, shapes, position, inventory)

### What's Left
- **High Priority:** Squad rendering (2-3h), enemy spawning (1-2h), bug fixes (2-3h)
- **Medium Priority:** GUI architectural cleanup (3-5h)
- **Low Priority:** Polish and optional improvements (2-4h)

**Total Remaining Work:** 10-16 hours across all priorities

---

**End of Roadmap**
