# Project Configuration for Claude Code

## Build Commands
- Build: `go build -o game_main/game_main.exe game_main/*.go`
- Run: `go run game_main/*.go`
- Test: `go test ./...`
- Clean: `go clean`

## Dependencies
- Install dependencies: `go mod tidy`

## Development Notes
- This is a Go-based roguelike game using the Ebiten engine
- Main entry point: `game_main/main.go`
- Assets directory: `../assets/` (relative to game_main)

## Common Issues
- Ensure assets directory exists with required tile images
- Run `go mod tidy` after pulling changes

## Simplification Roadmap (Priority Order)

### ✅ 1. Input System Consolidation *COMPLETED*
**Files:** `input/inputcoordinator.go`, controller files
- **Problem:** Scattered global state, tight coupling, mixed responsibilities
- **Status:** ✅ Implemented proper InputCoordinator with MovementController, CombatController, UIController
- **Achievement:** Eliminated scattered input handling and global state issues

### ✅ 2. Coordinate System Standardization *COMPLETED*
**Files:** `coords/cordmanager.go`, `coords/position.go`
- **Problem:** Multiple coordinate systems causing bugs (noted in LessonsLearned.txt)
- **Status:** ✅ Unified CoordinateManager replaces scattered CoordTransformer calls
- **Achievement:** Type-safe coordinate handling with LogicalPosition/PixelPosition

### 🔄 3. Status Effects vs Item Behaviors *85% COMPLETED*
**Files:** `gear/stateffect.go`, `gear/itemactions.go`
- **Problem:** Throwables forced into StatusEffect interface when they're actions, not effects
- **Status:** 🔄 85% Complete - ItemAction interface created with proper composition pattern
- **Achievement:** Conceptual separation achieved, throwables contain effects (not "are" effects)
- **Remaining:** Extract quality interface for true separation (15%)

### ✅ 4. Entity Template System *100% COMPLETED* 🎉
**File:** `entitytemplates/creators.go` (283 lines)
- **Problem:** Multiple `CreateXFromTemplate()` functions with identical structure
- **Status:** ✅ 100% Complete - Generic factory with configuration-based pattern implemented
- **Achievement:** 4 specialized functions → 1 unified `CreateEntityFromTemplate()` factory
- **Impact:** Type-safe entity creation with EntityType enum and EntityConfig struct
- **Note:** Backward-compatible wrappers maintained for existing code

### ✅ 5. Graphics Shape System *95% COMPLETED* 🎉
**File:** `graphics/drawableshapes.go` (390 lines)
- **Problem:** 8+ shape types with complex algorithms and code duplication
- **Status:** ✅ 95% Complete - Successfully consolidated into unified BaseShape system!
- **Achievement:** 8+ separate shape types → 1 BaseShape with 3 variants (Circular, Rectangular, Linear)
- **Impact:** Massive code duplication eliminated, quality system integrated into factories
- **Remaining:** Extract direction system to separate file (5%)
- **Note:** This represents the LARGEST simplification achievement in the roadmap!

### ❌ 6. GUI Button Factory *10% COMPLETED*
**File:** `gui/playerUI.go` (155 lines)
- **Problem:** 3 separate button creation functions with 90% duplicate code
- **Status:** ❌ 10% Complete - Basic CreateButton() exists
- **Remaining:** Implement ButtonConfig struct and CreateMenuButton(config) factory (90%)
- **Approach:** Configuration-based pattern with WindowDisplay interface

### 🔄 7. Squad System Infrastructure *35% COMPLETED* ⭐ MULTI-AGENT VALIDATED
**Files:** `squads/components.go` (300 LOC), `squads/squadmanager.go` (61 LOC), `squads/units.go` (202 LOC), `squads/squadcreation.go` (58 LOC)
- **Problem:** Need squad-based tactical combat for "command several squads" gameplay
- **Status:** 🔄 35% COMPLETE - 621 LOC implemented with perfect ECS patterns (MASTER_ROADMAP v2.0 validated)
- **Critical Discovery:** Squad system demonstrates PERFECT ECS architecture and serves as migration template for ALL legacy code

**Architectural Achievements (Multi-Agent Analysis):**
- ✅ **Perfect ECS Compliance:** Pure data components, zero logic methods, native `ecs.EntityID` usage
- ✅ **Query-Based Relationships:** No stored entity pointers, all relationships discovered via ECS queries
- ✅ **Multi-Cell Unit Support:** Units occupy 1x1 to 3x3 grid cells (2x2 giants, 1x3 cavalry)
- ✅ **Cell-Based Targeting:** Advanced patterns (1x2 cleave, 2x2 blast, 3x3 AOE)
- ✅ **Template System:** JSON-driven unit creation from `monsterdata.json`

**Implementation Status:**
| Subsystem | LOC | Status | Detail |
|-----------|-----|--------|--------|
| Components & Data | 300 | ✅ 100% | All 8 components defined with helper methods |
| Manager & Init | 61 | ✅ 80% | ECS registration complete, minor cleanup needed |
| Unit Templates | 202 | ✅ 75% | JSON loading works, needs SquadMemberData component |
| Squad Creation | 58 | 🔄 25% | CreateEmptySquad works, AddUnitToSquad is stub |
| Query System | 40 | 🔄 20% | 2 of ~6 functions implemented |
| Combat System | 0 | ❌ 0% | Documented in analysis, not coded |
| Ability System | 0 | ❌ 0% | Documented in analysis, not coded |
| Formation System | 0 | ❌ 0% | Not started |

**Functions Working NOW:**
```go
// ✅ Callable functions (tested in main.go)
InitializeSquadData() error                                    // Full ECS setup
CreateEmptySquad(manager, name) *ecs.Entity                   // Creates squad entity
CreateUnitEntity(manager, template) *ecs.Entity               // Creates unit with components
GetUnitIDsAtGridPosition(squadID, row, col, manager) []ecs.EntityID  // Spatial query
FindUnitByID(unitID, manager) *ecs.Entity                     // Entity lookup
```

**Functions Documented but NOT Implemented:**
```go
// ❌ Missing (exist in analysis files only)
GetUnitIDsInSquad(squadID, manager) []ecs.EntityID           // All units in squad
GetUnitIDsInRow(squadID, row, manager) []ecs.EntityID        // Row-based query
GetLeaderID(squadID, manager) ecs.EntityID                   // Find leader
IsSquadDestroyed(squadID, manager) bool                      // Check destruction
ExecuteSquadAttack(attackerID, defenderID, manager) *CombatResult  // Combat flow
CheckAndTriggerAbilities(squadID, manager)                   // Ability system
CreateSquadFromTemplate(manager, formation, units)           // Formation creation
```

**Remaining Work:** 28-36 hours (MASTER_ROADMAP v2.0)
- Query completion (4-6h) → Get all row/squad queries working
- Combat system (10-12h) → Implement ExecuteSquadAttack with row targeting
- Ability system (8-10h) → Auto-triggering leader abilities
- Formation/Creation (6-8h) → Formation presets and complete squad creation
- Testing infrastructure (4-6h) → Unit tests for combat WITHOUT map integration

**Key Insight:** Squad combat can be tested in isolation (no map dependency) due to component-based design

**Migration Template:** Use squad system's perfect ECS patterns to fix legacy Position, Weapon, Item, and Movement systems

## Overall Progress
**Roadmap Completion:** 80% (weighted average across all items)
- **Fully Complete:** 4 of 6 items (Input System, Coordinate System, Graphics Shapes, Entity Templates)
- **In Progress:** 1 of 6 items (Status Effects 85%)
- **Minimal Progress:** 1 of 6 items (GUI Buttons 10%)

## Completed Simplifications
- ✅ **Action Queue System Removal** - Removed complex ActionQueue/Turn system, implemented direct player actions
- ✅ **Graphics Shape System** - Consolidated 8+ shape types into unified BaseShape with type variants
- ✅ **Entity Template System** - 4 specialized functions → 1 generic factory with configuration pattern

---

## Refactoring Priorities for Todos Implementation

**Last Updated:** 2025-10-07 (v3.0 based on MASTER_ROADMAP v2.0 multi-agent analysis)
**Primary Source:** `analysis/MASTER_ROADMAP.md` v2.0

### Critical Path to Unblock Todos

#### ⭐ PHASE 0: Position System (8-12 hours) - CRITICAL PRIORITY
**Status:** 0% → Must Complete BEFORE Squad Combat
**Blocks:** Squad Combat Performance (critical), Multi-squad gameplay (critical)
**Impact:** CRITICAL - **50x performance improvement** required for efficient squad combat

**Why Elevated to Phase 0 (Multi-Agent Analysis Finding):**
- Current: O(n) linear search using `map[*coords.LogicalPosition]*ecs.Entity` (pointer keys!)
- With Squads: 5 squads × 9 units = 45 entities → O(45) lookups per action
- After Phase 0: O(1) hash-based lookup → **45-50x faster**
- Position queries are HOT PATH in combat (called every attack/movement)

**Current Anti-Pattern:**
```go
type PositionTracker struct {
    PosTracker map[*coords.LogicalPosition]*ecs.Entity  // Pointer keys cause O(n) scan
}
```

**Squad System Template (Perfect ECS Pattern):**
```go
type PositionSystem struct {
    manager     *ecs.Manager
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // Value keys = O(1) hash
}

func (ps *PositionSystem) GetEntityIDAt(pos coords.LogicalPosition) ecs.EntityID {
    if ids, ok := ps.spatialGrid[pos]; ok && len(ids) > 0 {
        return ids[0]  // O(1) lookup
    }
    return 0
}
```

**Deliverables:**
1. Create `PositionSystem` struct replacing `trackers/creaturetracker.go` PositionTracker (6-8h)
2. Implement O(1) spatial grid with native `ecs.EntityID` references
3. Update all position lookups in combat/input systems (2-4h)
4. Methods: `GetEntityIDAt()`, `AddEntity()`, `RemoveEntity()`, `MoveEntity()`

**Success Criteria:**
- [ ] O(1) position lookups via hash-based spatial grid
- [ ] All entity references use `ecs.EntityID` (not pointers)
- [ ] Benchmark shows 30-50x improvement with 30+ entities
- [ ] All existing combat/movement queries still work
- [ ] Game runs without errors

**Testing:**
- Benchmark position lookups before/after (target: 30-50x speedup)
- Test with 50 entities at different positions
- Verify combat position queries work correctly

**Why Before Squad Combat:**
Squad combat makes HEAVY use of position queries:
- Row targeting needs position lookups for front/mid/back rows
- Multi-cell units (2x2, 1x3) need spatial queries to find all occupied cells
- AOE abilities need area position lookups
- Efficient position system UNBLOCKS performant multi-squad gameplay

---

#### ✅ PRIORITY 1: Complete Entity Template System ~~(4 hours)~~ COMPLETED
**Status:** 50% → 100% ✅
**Blocks:** ~~Spawning system (todos.txt:31)~~ UNBLOCKED
**Impact:** HIGH - Probability-based entity spawning now available

**Completed Work:**
- ✅ Consolidated 4 `CreateXFromTemplate()` functions into generic `CreateEntityFromTemplate()` factory
- ✅ Added EntityType enum and EntityConfig struct for type-safe creation
- ✅ Maintained backward compatibility with deprecated wrappers
- ✅ Spawning system can now use flexible entity creation

**Result:** +106 LOC (includes documentation), 283 lines total, builds successfully

---

#### 🔄 PRIORITY 2: Squad Combat Foundation (28-36 hours remaining)
**Status:** 0% → **35% COMPLETE** (621 LOC implemented, MASTER_ROADMAP v2.0 validated)
**Blocks:** AI system (high), balance (high), spawning quality (medium)
**Impact:** CRITICAL - Major architectural change for "command several squads" (todos.txt:25)

**Multi-Agent Discovery:** Squad infrastructure already 35% implemented with perfect ECS patterns!

**What's Already Done (621 LOC):**
- ✅ Components & Data: 100% complete (300 LOC) - All 8 component types
- ✅ Manager & Init: 80% complete (61 LOC) - ECS registration working
- ✅ Unit Templates: 75% complete (202 LOC) - JSON loading functional
- ✅ Squad Creation: 25% complete (58 LOC) - CreateEmptySquad works
- 🔄 Query System: 20% complete (40 LOC) - 2 of ~6 functions done

**Remaining Work (28-36 hours):**
- Query completion (4-6h) → Implement GetUnitIDsInSquad, GetUnitIDsInRow, GetLeaderID, IsSquadDestroyed
- Combat system (10-12h) → Implement ExecuteSquadAttack with row targeting and role modifiers
- Ability system (8-10h) → Auto-triggering leader abilities with condition evaluation
- Formation/Creation (6-8h) → Formation presets and complete CreateSquadFromTemplate
- Testing (4-6h) → Unit tests for combat WITHOUT map integration

**Incremental Approach (REVISED):**
- **Phase 1.1-1.5** (28-36h): Complete squad core systems (queries, combat, abilities, formations, tests)
- **Phase 2** (16-24h): Map integration (rendering, input, spawning)
- **Phase 3** (parallel): Legacy code migration using squad patterns

**Key Insight:** Can test squad combat in isolation (no map dependency) due to component-based design

**Analysis:** See `analysis/MASTER_ROADMAP.md` for detailed phase breakdown and `analysis/squad_system_final.md` for architecture

---

#### 🟢 PRIORITY 3: Quick Wins (4 hours total)
**Status:** GUI Buttons 10% → 100%, Status Effects 85% → 100%
**Blocks:** Nothing
**Impact:** LOW - Maintainability and roadmap completion

**GUI Button Factory (2 hours):**
- 3 duplicate functions → 1 factory
- Net -35 LOC (23% reduction in playerUI.go)
- Zero functional impact, pure maintainability gain

**Status Effects Quality (2 hours):**
- Extract quality interface from StatusEffects/ItemAction
- Complete conceptual separation (effects ≠ loot quality)
- Helps spawning system by clarifying quality management

---

### Implementation Timeline (REVISED v3.0 - MASTER_ROADMAP v2.0)

**OVERALL TIMELINE:** 9-13 workdays (72-106 hours)
**TIME SAVED:** 34-38 hours due to existing squad infrastructure (621 LOC)

**Critical Path (Sequential):** 52-72 hours
**Parallel Work Available:** 30-42 hours (legacy system refactoring)

---

**Week 1: Phase 0 + Squad Core Start (30-40 hours)**
1. ⭐ **Phase 0: Position System** (8-12h) - CRITICAL for squad performance
   - Create PositionSystem with O(1) spatial grid
   - Update all combat/input position lookups
   - Benchmark 30-50x speedup
2. **Phase 1.1: Query Completion** (4-6h) - GetUnitIDsInSquad, GetUnitIDsInRow, etc.
3. **Phase 1.2: Combat System Start** (10-12h) - ExecuteSquadAttack with row targeting
4. **Phase 1.3: Ability System Start** (8-10h) - Auto-triggering abilities

**Result:** Position system optimized, squad queries complete, combat system in progress

---

**Week 2: Squad Core Completion + Testing (22-30 hours + testing)**
1. **Phase 1.3: Ability System Complete** (remaining hours)
2. **Phase 1.4: Formation/Creation** (6-8h) - Formation presets, CreateSquadFromTemplate
3. **Phase 1.5: Testing Infrastructure** (4-6h) - Unit tests for squad combat WITHOUT map
4. **PARALLEL: Phase 3.2 Weapon System** (12-16h) - Migrate weapon logic to proper ECS

**MILESTONE:** ✅ Squad combat works in isolation (unit tests pass, no map needed)

**Result:** Squad mechanics validated, ready for map integration

---

**Week 3: Map Integration + Todos (34-50 hours)**
1. **Phase 2.1: Map Representation** (4-6h) - Squad positioning on map
2. **Phase 2.2: Input Integration** (4-6h) - Squad selection and combat initiation
3. **Phase 2.3: Rendering Integration** (4-6h) - Squad grid visualization
4. **Phase 2.4: Spawning Integration** (4-6h) - Enemy squad spawning with level scaling
5. **PARALLEL: Phase 3.3 Item System** (10-14h) - Flatten item structure, remove nested entities
6. **Phase 4: Todos Implementation** (8-12h) - Bug fixes, throwing accuracy, level variety, spawning

**MILESTONE:** ✅ Full game loop with squad combat on map

**Result:** Squads integrated, todos complete, item system migrated

---

**Week 4: Legacy Migration + Polish (optional, 18-26 hours)**
1. **Phase 3.4: Movement System** (8-12h) - Extract movement logic to MovementSystem
2. **Polish & Bug Fixes** (10-14h) - Balance, UI improvements, testing

**MILESTONE:** ✅ 100% ECS compliance, all legacy patterns migrated

**Result:** Production-ready tactical squad roguelike

---

**Fastest Completion:** 9 workdays (72 hours minimum, critical path only)
**Realistic Completion:** 11 workdays (92 hours with parallel work)
**Conservative Estimate:** 13 workdays (106 hours with polish)

**IMPROVEMENT OVER v1.0:** 4-5 workdays saved (34-38 hours) thanks to existing squad infrastructure

---

## ECS Architecture Insights (Multi-Agent Analysis)

**Primary Source:** MASTER_ROADMAP v2.0 Current State Assessment

### Perfect ECS Patterns (Squad System Template)

The squad system (35% complete, 621 LOC) demonstrates **production-quality ECS architecture** and should serve as the migration template for ALL legacy code.

**✅ Perfect Patterns to Replicate:**

1. **Pure Data Components** (Zero Logic Methods)
   ```go
   // ✅ GOOD: Squad system components (squads/components.go)
   type GridPositionData struct {
       Row    int
       Col    int
       Width  int  // Multi-cell support
       Height int
   }

   // Helper methods are OK (they don't modify state or query other entities)
   func (g *GridPositionData) GetOccupiedCells() []GridCell {
       // Pure calculation, no ECS queries, no state mutation
   }
   ```

2. **Native EntityID Usage** (No Custom Registry)
   ```go
   // ✅ GOOD: Squad system uses ecs.EntityID everywhere
   func GetUnitIDsInSquad(squadID ecs.EntityID, manager *ecs.Manager) []ecs.EntityID {
       // Returns native IDs, not pointers or custom types
   }

   // ❌ BAD: Legacy Item system uses nested entity pointers
   type Item struct {
       Properties *ecs.Entity  // Anti-pattern: nested entity pointer
   }
   ```

3. **Query-Based Relationships** (No Stored Entity References)
   ```go
   // ✅ GOOD: Squad system discovers relationships via ECS queries
   func GetLeaderID(squadID ecs.EntityID, manager *ecs.Manager) ecs.EntityID {
       // Query for entity with LeaderTag and SquadMemberData pointing to squadID
       // Relationship discovered dynamically, not stored
   }

   // ❌ BAD: Storing entity pointers creates coupling
   type Squad struct {
       Leader *ecs.Entity  // Anti-pattern: stored pointer
   }
   ```

4. **System-Based Logic** (Not Component Methods)
   ```go
   // ✅ GOOD: Combat logic in system (future squads/combat.go)
   type CombatSystem struct {
       manager *ecs.Manager
   }
   func (cs *CombatSystem) ExecuteSquadAttack(attackerID, defenderID ecs.EntityID) *CombatResult {
       // All combat logic in system, components are just data
   }

   // ❌ BAD: Legacy Weapon has logic in component methods
   type MeleeWeapon struct {
       // ... component fields
   }
   func (w *MeleeWeapon) CalculateDamage() int {
       // Anti-pattern: logic in component method
   }
   ```

5. **Value-Based Map Keys** (Not Pointer Keys)
   ```go
   // ✅ GOOD: Squad system spatial grid (squads/units.go pattern)
   type PositionSystem struct {
       spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // Value keys = O(1) hash
   }

   // ❌ BAD: Legacy PositionTracker (trackers/creaturetracker.go)
   type PositionTracker struct {
       PosTracker map[*coords.LogicalPosition]*ecs.Entity  // Pointer keys = O(n) scan!
   }
   ```

### Legacy Anti-Patterns (To Be Fixed)

**❌ Anti-Patterns Found in Legacy Code:**

| System | Anti-Pattern | Impact | Fix Via |
|--------|-------------|--------|---------|
| Position | Pointer map keys (`map[*LogicalPosition]*Entity`) | O(n) lookups, 50x slower | Phase 0 (8-12h) |
| Weapon | Logic methods (`CalculateDamage()` in component) | Tight coupling, hard to test | Phase 3.2 (12-16h) |
| Item | Nested entity pointers (`Item.Properties *Entity`) | Circular refs, query complexity | Phase 3.3 (10-14h) |
| Movement | Component methods (`Creature.UpdatePosition()`) | Business logic in data structs | Phase 3.4 (8-12h) |

**Total Legacy Refactoring:** 38-54 hours (can parallelize with squad development)

### Migration Strategy

**Use squad system as template for fixing each legacy system:**

1. **Create System Struct** - Move all logic from components to dedicated system
2. **Pure Data Components** - Remove all methods except pure helpers
3. **Native EntityID** - Replace entity pointers with `ecs.EntityID`
4. **Query Functions** - Discover relationships dynamically via ECS queries
5. **Value Keys** - Use value-based map keys for O(1) performance

**Example: Weapon System Migration (Phase 3.2)**
```go
// BEFORE (Anti-pattern)
type MeleeWeapon struct { /* fields */ }
func (w *MeleeWeapon) CalculateDamage() int { /* logic in component */ }

// AFTER (Squad system pattern)
type MeleeWeapon struct { /* pure data fields only */ }

type WeaponSystem struct { manager *ecs.Manager }
func (ws *WeaponSystem) CalculateDamage(weaponID ecs.EntityID) int {
    // Logic in system, component is just data
}
```

**Result:** 100% ECS compliance, proper separation of concerns, 50x performance improvement in hot paths

---

### What Can Be Implemented NOW (No Blockers)

✅ **Bug Fixes** (todos.txt lines 4, 6, 8)
- Fix throwable AOE movement issue
- Ensure entities removed on death
- Prevent shooting/throwing through walls

✅ **Throwing Improvements** (todos.txt line 36)
- Add accuracy/miss chance for thrown items
- Uses existing ItemAction system (85% complete)

✅ **Level Transitions** (todos.txt line 42)
- Clear entities on level change
- Add level variety and tile diversity

⏳ **BLOCKED Until Refactoring:**

✅ **Spawning System** (todos.txt line 31)
- **Status:** UNBLOCKED - Entity Template System completed ✅
- **Available:** Generic `CreateEntityFromTemplate()` factory with EntityConfig pattern
- **Ready to implement:** Probability-based spawning with level scaling

🔄 **Squad System Features** (todos.txt line 25)
- **Status:** 35% COMPLETE (621 LOC implemented, MASTER_ROADMAP v2.0 validated)
- **Available NOW:** Squad creation, unit templates, basic queries, multi-cell units, component infrastructure
- **Working functions:** `InitializeSquadData()`, `CreateEmptySquad()`, `CreateUnitEntity()`, `GetUnitIDsAtGridPosition()`, `FindUnitByID()`
- **Remaining:** Combat (10-12h), abilities (8-10h), queries (4-6h), formations (6-8h), testing (4-6h) = 28-36 hours
- **Can start:** Query completion (Phase 1.1) immediately after Phase 0 (Position System)

⭐ **Position System Optimization** (CRITICAL - Must do FIRST)
- **Status:** 0% - BLOCKING squad combat performance
- **Impact:** 50x performance improvement (O(n) → O(1) position lookups)
- **Time:** 8-12 hours
- **Unblocks:** Efficient multi-squad gameplay, all squad combat features
- **Priority:** PHASE 0 - Do BEFORE squad combat implementation

🔄 **Balance/Difficulty** (todos.txt line 13)
- **Status:** Partially blocked by squad combat (high impact on difficulty calculation)
- **Available:** Basic individual entity difficulty calculations
- **Blocked:** Squad-based difficulty scaling (needs squad combat complete)

---

### Analysis Files

**Primary Reference:** `analysis/MASTER_ROADMAP.md` v2.0 (2025-10-06)
- **Executive Summary:** 72-106 hours (9-13 workdays), squad system 35% complete (621 LOC)
- **Phase Breakdown:** Phase 0 (Position 8-12h) → Phase 1 (Squad Core 28-36h) → Phase 2 (Integration 16-24h) → Phase 3 (Legacy 30-42h) → Phase 4 (Todos 8-12h)
- **Current State Assessment:** Exact code inventory showing implemented vs documented-only functions
- **ECS Compliance Analysis:** Perfect patterns (squad) vs legacy anti-patterns, migration strategy
- **Testing Strategy:** Squad combat can be tested WITHOUT map integration
- **Multi-Agent Validated:** karen + insight-synthesizer + docs-architect comprehensive analysis

**Supporting Documentation:**
- `squad_system_final.md` - Detailed squad architecture (components, abilities, targeting, multi-cell units)
- `combat_refactoring.md` - Squad combat migration strategy (may be superseded by MASTER_ROADMAP v2.0)
- `roadmap_completion.md` - Simplification roadmap status (may be superseded by MASTER_ROADMAP v2.0)

**IMPORTANT:** MASTER_ROADMAP v2.0 is the authoritative source for timelines, status, and implementation strategy. Other analysis files may contain outdated estimates.