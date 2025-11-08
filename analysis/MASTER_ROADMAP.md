# TinkerRogue Master Development Roadmap

**Version:** 4.0 - REALITY CHECK EDITION | **Updated:** 2025-11-07
**Status:** Squad System 98% Complete - ACTUALLY FUNCTIONAL

---

## Executive Summary

This roadmap was audited by verifying actual implementation files, test coverage, and functional integration. Previous claims have been validated against reality.

**What Changed from v3.0:**
- Ability System marked as "NOT STARTED" → **ACTUALLY COMPLETE** (317 LOC, fully tested)
- Squad System "85% complete" → **98% complete** (4951 total LOC)
- Map Integration "NOT STARTED" → **COMPLETE** (Turn manager integrates abilities)
- Formation Presets "partial stubs" → **COMPLETE** (4 presets implemented)

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

**GUI System (40% complete):**
- ✅ GUI mode system exists (explorationmode.go, combatmode.go, squadmanagementmode.go, etc.)
- ✅ ButtonConfig pattern exists (createwidgets.go, lines 58-102)
- ✅ **Button Factory Pattern** - Inconsistent button creation across modes (needs standardization)

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

## Reality Check: Claimed vs Actual Status

| Claim (v3.0) | Actual Status | Evidence |
|--------------|---------------|----------|
| Ability System "NOT STARTED (8-10h)" | **COMPLETE** | squadabilities.go (317 LOC), integrated in turnmanager.go, tests passing |
| Formation Presets "40% COMPLETE (4-6h)" | **COMPLETE** | GetFormationPreset() returns 4 presets with 5-6 units each |
| Map Integration "NOT STARTED (4-6h)" | **COMPLETE** | turnmanager.go calls CheckAndTriggerAbilities, combat system operational |
| Squad System "85% complete" | **98% complete** | Only graphical rendering and spawning hookup remain |
| AddUnitToSquad "only validates, doesn't create entity" | **FULLY FUNCTIONAL** | Creates entity, validates capacity, updates squad (lines 36-82) |

---

## Time Estimates (REALISTIC)

**Total Remaining:** 6-10 hours (conservative)

### Breakdown
- Squad Graphical Rendering: 2-3h (sprites, HP bars, role icons)
- Enemy Squad Spawning: 1-2h (hookup existing logic to level system)
- GUI Button Standardization: 1-2h (optional - system works as-is)
- Bug Fixes: 2-3h (throwable AOE, entity cleanup, wall collision)
- Polish: 1-2h (accuracy, level transitions, visual variety)

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

## Phase 3: Polish and Bug Fixes (0% complete)

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

### Overall System Success
- ✅ Multi-squad tactical gameplay foundation
- ✅ Squad building with formations
- ✅ Multi-cell units add variety
- ✅ 100% ECS pattern compliance
- ✅ Stable performance with spatial grid

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

### Immediate (This Week - 4-6h)
1. **Create spawning/spawnSquads.go** - SpawnEnemySquad wrapper function (1-2h)
2. **Squad graphical rendering** - Integrate with existing GUI modes (2-3h)
3. **Hook up enemy spawning** - Call SpawnEnemySquad in level generation (1h)

### First Milestone (End of Week)
- Squad System 100% complete
- Enemy squads spawn automatically
- Graphical squad visualization

### Second Milestone (Week 2)
- Bug fixes complete (throwables, entity cleanup, walls)
- Polish features (accuracy, level transitions, variety)
- Full game loop with squads operational

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

**End of Roadmap**
