# Pre-Feature Refactoring Analysis
**Date:** 2025-11-09
**Status:** Ready for cleanup before new features
**Codebase Size:** 19,414 LOC across 91 Go files

## Executive Summary

This analysis identifies technical debt and refactoring opportunities that should be addressed BEFORE implementing new gameplay features. Focus areas: random number generation inconsistency, duplicate functionality, excessive global state, and incomplete ECS migration.

**Priority Breakdown:**
- **High Priority:** 5 items (blocking new features or causing bugs)
- **Medium Priority:** 7 items (maintenance burden, performance)
- **Low Priority:** 4 items (nice-to-have, minor cleanup)

---

## HIGH PRIORITY (Blocking or Bug-Prone)

### 1. Random Number Generation Inconsistency (CRITICAL)
**Impact:** HIGH - Introduces non-determinism and makes testing impossible
**Affected Files:** 11 files use `math/rand`, 1 file uses `crypto/rand`

**Problem:**
- Most codebase uses `math/rand` (deterministic, seedable for testing)
- `common/randnumgen.go` uses `crypto/rand` (non-deterministic, can't seed)
- Creates inconsistent behavior across systems
- Makes reproducible testing impossible
- Impacts gameplay randomness (combat, spawning, loot)

**Files Using math/rand:**
- `spawning/*.go` (monsters, loot, throwables)
- `graphics/vx.go`, `graphics/drawableshapes.go`
- `combat/queries.go`, `squads/squadcombat.go`
- `gear/itemactions.go`, `gear/itemquality.go`

**Files Using crypto/rand:**
- `common/randnumgen.go` - `GetDiceRoll()`, `GetRandomBetween()`

**Solution:**
- Standardize on `math/rand/v2` (better API, still deterministic)
- Create centralized `RNG` wrapper in `common/` package
- Provide seed control for deterministic testing
- Replace all direct `rand` imports with wrapper

**Effort:** 2-3 hours (mechanical refactor, add wrapper)

---

### 2. Testing Infrastructure Pollution (CRITICAL)
**Impact:** HIGH - Test scaffolding leaking into production code
**Affected:** `combat/gameplayfactions.go`, `game_main/gamesetup.go`

**Problem:**
- `SetupGameplayFactions()` marked "TODO: Remove this in the future" but still called in production
- Creates 6 squads (3 player, 3 AI) hardcoded for gameplay testing
- Mixes test setup with real game initialization
- No clear separation between debug/test and production code
- `DEBUG_MODE` flag controls behavior but test code always present

**Current Production Flow:**
```
SetupNewGame() -> SetupSquadSystem() -> SetupGameplayFactions()
                                        ^^^^ Test code in production!
```

**Solution:**
- Move faction/squad spawning to proper spawning system
- Create data-driven faction definitions (JSON config)
- Remove `SetupGameplayFactions()` from production path
- Keep test version only in `testing/` or test files
- Use proper dependency injection for test vs. production

**Effort:** 4-6 hours (extract to spawning, add config)

---

### 3. Incomplete ECS Migration - TileContents (HIGH)
**Impact:** HIGH - Performance and correctness issues
**File:** `worldmap/tile.go` (TileContents struct)

**Problem:**
- `TileContents.entityIDs` uses `[]ecs.EntityID` (good!)
- But lowercase field name `entityIDs` suggests private field
- Inconsistent naming with other ECS code (should be `EntityIDs`)
- May cause silent bugs when accessing from other packages
- Fixed for inventory/items but not worldmap tiles

**Current State:**
```go
type TileContents struct {
    entityIDs []ecs.EntityID  // Should be EntityIDs
}
```

**Solution:**
- Rename `entityIDs` -> `EntityIDs` (public field)
- Audit all access in `worldmap/*.go` for correctness
- Verify CLAUDE.md note about 2025-11-08 fix was complete

**Effort:** 1-2 hours (mechanical rename, verify access)

---

### 4. GetRandomBetween() Infinite Loop Risk (HIGH)
**Impact:** HIGH - Can hang game if parameters invalid
**File:** `common/randnumgen.go:16-26`

**Problem:**
```go
func GetRandomBetween(low int, high int) int {
    var randy int = -1
    for {
        randy = GetDiceRoll(high)  // Returns 1 to high
        if randy >= low {
            break
        }
    }
    return randy
}
```

**Issues:**
- If `low > high`, infinite loop guaranteed
- `GetDiceRoll(high)` returns 1 to high, not 0 to high
- If `low == 0`, will loop forever (GetDiceRoll minimum is 1)
- No validation or panic on invalid input
- Used throughout spawning system

**Solution:**
- Add parameter validation (panic or return error if low > high)
- Fix off-by-one error: `GetDiceRoll(high - low + 1) + low - 1`
- Or replace with standard library: `rand.IntN(high-low+1) + low`
- Add unit tests for edge cases

**Effort:** 1 hour (fix + tests)

---

### 5. ValidPos Global State Anti-Pattern (HIGH)
**Impact:** MEDIUM-HIGH - Breaks encapsulation, causes bugs
**File:** `worldmap/dungeongen.go:18`

**Problem:**
```go
var ValidPos ValidPositions  // Package-level mutable global
```

- Global mutable state updated during map generation
- Multiple generators modify same global variable
- Creates hidden coupling between worldmap and spawning systems
- Race condition risk if generators run concurrently
- Contradicts strategy pattern improvements (2025-11-08)

**Usage:**
- Updated by: `NewGameMap()` from generator results
- Read by: `PlaceStairs()`, spawning systems
- Fallback mechanism in worldmap functions

**Solution:**
- Add `ValidPositions` field to `GameMap` struct
- Pass positions explicitly through function parameters
- Remove global variable entirely
- Update spawning to query GameMap instead of global

**Effort:** 3-4 hours (touch multiple systems)

---

## MEDIUM PRIORITY (Maintenance Burden)

### 6. GUI Mode Files Too Large (MEDIUM)
**Impact:** MEDIUM - Hard to navigate and maintain
**Files:**
- `gui/combatmode.go` (987 LOC)
- `gui/squadbuilder.go` (774 LOC)

**Problem:**
- Combat mode mixing UI, combat logic, state management
- Squad builder combining editor, palette, validation
- Violates single responsibility principle
- Hard to test individual subsystems
- Difficult to find specific functionality

**Solution:**
- Extract combat UI panels to separate files (turn order, squad list, action buttons)
- Extract squad builder editor logic from UI rendering
- Create dedicated state managers for complex modes
- Keep mode files under 400 LOC

**Effort:** 6-8 hours per file (extract + refactor)

---

### 7. Visual Effects File Bloat (MEDIUM)
**Impact:** MEDIUM - 894 LOC in single file
**File:** `graphics/vx.go`

**Problem:**
- All visual effects in one massive file
- Mixes: base effect system, animator interface, specific effects (fire, ice, sticky, shake, etc.)
- Hard to find specific effect implementations
- `VXHandler` global variable for effect management

**Current Structure:**
- Lines 1-100: Interfaces and base system
- Lines 100-400: Animator implementations
- Lines 400-700: Specific effect types
- Lines 700-894: Handler and utilities

**Solution:**
- Split into 4-5 files:
  - `vx_base.go` - Interfaces, BaseEffect, AnimationState
  - `vx_animators.go` - Animator implementations
  - `vx_effects.go` - Specific effects (fire, ice, etc.)
  - `vx_handler.go` - VisualEffectHandler, global management
- Keep related code together by responsibility

**Effort:** 4-5 hours (split + organize)

---

### 8. Duplicate Test Data Creation (MEDIUM)
**Impact:** MEDIUM - Repeated boilerplate in tests
**Files:** `combat/combat_test.go`, `squads/*_test.go`, `testing/testingdata.go`

**Problem:**
- Each test file recreates: EntityManager, PositionSystem, components
- Copy-paste squad setup code across 3+ test files
- No shared test fixtures or helpers
- Changes to component initialization require N file updates

**Pattern Repeated 8+ Times:**
```go
manager := common.NewEntityManager()
common.InitializeCommonComponents(manager)
common.GlobalPositionSystem = systems.NewPositionSystem(manager.World)
squads.InitializeSquadData(manager)
// ... squad creation boilerplate
```

**Solution:**
- Create `testing/fixtures.go` with:
  - `NewTestEntityManager()` - fully initialized manager
  - `CreateTestSquad(name, units)` - squad factory
  - `CreateTestUnit(role, stats)` - unit factory
- Use fixtures in all test files
- Single source of truth for test setup

**Effort:** 3-4 hours (create fixtures, update tests)

---

### 9. Status Effects Interface Complexity (MEDIUM)
**Impact:** MEDIUM - Hard to add new effects
**File:** `gear/stateffect.go` (380 LOC)

**Problem:**
- 7-step process to add new status effect (documented in comments)
- Manual registration in multiple places (const, slice, init)
- Interface requires both `StatusEffects` AND `common.Quality`
- Type assertions scattered across code
- CLAUDE.md notes "85% complete, needs quality interface extraction"

**Steps to Add Effect (Current):**
1. Create const name
2. Add to EffectNames slice
3. Create component type
4. Create struct
5. Initialize component in InitializeItemComponents
6. Add to AllItemEffects slice
7. Implement 6 interface methods

**Solution:**
- Use struct tags for registration: `type Burning struct { Name string \`effect:"Burning"\` }`
- Automatic registration via reflection in init()
- Extract Quality to separate concern (composition over inheritance)
- Reduce to 3 steps: struct definition, implement Apply/Stack, done

**Effort:** 5-6 hours (reflection system + migration)

---

### 10. Component Initialization Fragmentation (MEDIUM)
**Impact:** MEDIUM - Error-prone initialization
**Files:** Multiple `components.go`, `game_main/componentinit.go`

**Problem:**
- Components initialized across 6+ files:
  - `common/ecsutil.go` - InitializeCommonComponents()
  - `combat/components.go` - InitializeCombatSystem()
  - `squads/components.go` - InitializeSquadData()
  - `gear/stateffect.go` - InitializeItemComponents()
  - `game_main/componentinit.go` - InitializeECS()
- Easy to miss initialization in tests
- No compile-time guarantee of initialization order
- Some inits called multiple times (checks for nil)

**Solution:**
- Single `InitializeAllComponents(manager)` function
- Call order enforced internally
- Return error if already initialized
- Clear documentation of dependencies

**Effort:** 3-4 hours (consolidate + test)

---

### 11. Entity Pointer Usage Still Present (MEDIUM)
**Impact:** MEDIUM - Contradicts ECS best practices
**Files:** 27 function signatures take `*ecs.Entity`

**Problem:**
- CLAUDE.md Section 9 mandates EntityID-only
- 27 functions still use entity pointers (Grep results)
- Most in: `gear/stateffect.go`, `common/ecsutil.go`, legacy code
- Squad and inventory systems fully migrated
- Creates inconsistent patterns

**Files with Entity Pointers:**
- `gear/stateffect.go` - `ApplyToCreature(c *ecs.QueryResult)`
- `common/ecsutil.go` - Helper functions
- Some query functions

**Solution:**
- Audit 27 functions, categorize by migration difficulty
- Prioritize public APIs and frequently-called functions
- Convert to EntityID + manager pattern
- Keep *ecs.Entity only in low-level utilities if needed

**Effort:** 6-8 hours (case-by-case analysis)

---

### 12. Throwable Item Naming (MEDIUM)
**Impact:** LOW-MEDIUM - Poor UX
**File:** `spawning/spawnthrowable.go:33`

**Problem:**
```go
itemName := ""
for _ = range RandomNumProperties() {
    // ... generate effects
    itemName += entry.StatusEffectName()  // "BurningFreezingSticky"
}
```

- Names are concatenated effect names: "BurningFreezingSticky"
- No spaces, capitalization issues
- No quality indicator ("Rare Burning Grenade")
- Comment says "Todo need better way to create a name"

**Solution:**
- Generate descriptive names: "Rare Burning Grenade", "Common Ice Bomb"
- Use quality prefix + primary effect + item type
- Store name templates in JSON config

**Effort:** 2-3 hours (naming system + config)

---

## LOW PRIORITY (Minor Cleanup)

### 13. TODO Comments Accumulation (LOW)
**Impact:** LOW - Technical debt markers
**Count:** 19 files contain TODO/FIXME/HACK

**Notable TODOs:**
- `combat/gameplayfactions.go:12` - "Remove this in the future" (see Priority #2)
- `worldmap/dungeongen.go:21` - "Determine if ValidPos still needed" (see Priority #5)
- `worldmap/dungeongen.go:339` - "Change to check WALL not blocked"
- `spawning/spawnthrowable.go:13-16` - Config file for properties
- Multiple spawning system TODOs for error handling

**Solution:**
- Convert high-value TODOs to GitHub issues
- Remove obsolete TODOs (already fixed)
- Group related TODOs into refactoring tasks

**Effort:** 2-3 hours (audit + triage)

---

### 14. Map Generator Comment Inconsistency (LOW)
**Impact:** LOW - Stale documentation
**File:** `worldmap/dungeongen.go:277`

**Problem:**
```go
// Old generation methods removed - now handled by generator implementations
// See gen_rooms_corridors.go for the extracted algorithm
```

- Comment about removed code left in place
- Good note but should be in package doc or design doc
- Function-level comments reference old architecture

**Solution:**
- Move architectural notes to package godoc
- Update inline comments to describe current behavior
- Remove references to "old" code

**Effort:** 1 hour (documentation pass)

---

### 15. Rect Center Method Inconsistency (LOW)
**Impact:** LOW - Minor API wart
**File:** `worldmap/dungeongen.go:90-93`

**Problem:**
```go
func (r *Rect) Center() (int, int) {  // Pointer receiver
    centerX := (r.X1 + r.X2) / 2
    centerY := (r.Y1 + r.Y2) / 2
    return centerX, centerY
}
```

- All other Rect methods use value receivers
- Center() uses pointer receiver for no reason
- No mutation, pure calculation

**Solution:**
- Change to value receiver: `func (r Rect) Center()`
- Consistent with `IsInRoom(r Rect)`, `GetCoordinates(r Rect)`

**Effort:** 15 minutes (mechanical change)

---

### 16. Attributes CanAct Field Unused (LOW)
**Impact:** LOW - Dead field cluttering struct
**File:** `common/commoncomponents.go:44`

**Problem:**
```go
type Attributes struct {
    // ... other fields
    CanAct bool  // Can unit act this turn
}
```

- Field initialized to `true` in NewAttributes()
- Never read or modified anywhere in codebase
- Turn-based logic uses `ActionStateData` instead
- Leftover from old combat system

**Solution:**
- Remove field from Attributes struct
- Remove initialization from NewAttributes()
- Verify no tests depend on it

**Effort:** 30 minutes (remove + verify)

---

## SUMMARY TABLE

| Priority | Issue | Impact | Effort | Blocking? |
|----------|-------|--------|--------|-----------|
| HIGH | 1. RNG Inconsistency | Critical | 2-3h | Yes - Testing |
| HIGH | 2. Test Code in Production | Critical | 4-6h | Yes - Architecture |
| HIGH | 3. TileContents Field Name | High | 1-2h | Partial |
| HIGH | 4. GetRandomBetween Loop | High | 1h | Partial - Spawning |
| HIGH | 5. ValidPos Global State | Medium-High | 3-4h | No |
| MEDIUM | 6. Large GUI Mode Files | Medium | 12-16h | No |
| MEDIUM | 7. Visual Effects Bloat | Medium | 4-5h | No |
| MEDIUM | 8. Duplicate Test Setup | Medium | 3-4h | No |
| MEDIUM | 9. Status Effects Interface | Medium | 5-6h | No |
| MEDIUM | 10. Component Init Fragmentation | Medium | 3-4h | No |
| MEDIUM | 11. Entity Pointer Usage | Medium | 6-8h | No |
| MEDIUM | 12. Throwable Item Names | Low-Medium | 2-3h | No |
| LOW | 13. TODO Accumulation | Low | 2-3h | No |
| LOW | 14. Generator Comments | Low | 1h | No |
| LOW | 15. Rect Center Method | Low | 15min | No |
| LOW | 16. Attributes CanAct | Low | 30min | No |

**Total Estimated Effort:**
- **High Priority:** 11-17 hours
- **Medium Priority:** 35-46 hours
- **Low Priority:** 4-5 hours
- **Grand Total:** 50-68 hours

---

## RECOMMENDED ACTION PLAN

### Phase 1: Critical Fixes (1-2 days)
Address items that block new features or cause correctness issues:
1. Fix GetRandomBetween() infinite loop (1h)
2. Standardize RNG to math/rand/v2 wrapper (2-3h)
3. Rename TileContents.entityIDs field (1-2h)
4. Remove test faction setup from production (4-6h)

**Result:** Solid foundation for new features, no known correctness bugs

### Phase 2: Architecture Cleanup (3-5 days)
Improve maintainability before adding complexity:
1. Extract ValidPos global to GameMap field (3-4h)
2. Create shared test fixtures (3-4h)
3. Consolidate component initialization (3-4h)
4. Split large GUI mode files (12-16h)

**Result:** Easier to add new UI modes, more testable

### Phase 3: Polish (2-3 days)
Nice-to-have improvements for code quality:
1. Split visual effects file (4-5h)
2. Improve status effects interface (5-6h)
3. Migrate entity pointer functions (6-8h)
4. Fix throwable naming (2-3h)
5. Clean up TODOs and minor issues (3-4h)

**Result:** Professional codebase ready for long-term maintenance

---

## NOTES

**What This Analysis Excludes:**
- Feature implementations (throwing accuracy, level transitions, etc.)
- Balance and difficulty tuning
- Squad formation presets (already tracked in CLAUDE.md)
- New gameplay systems

**ECS Migration Status:**
- Squad system: 100% compliant (2675 LOC)
- Inventory system: 100% compliant (533 LOC)
- Combat system: 90% compliant (some entity pointers remain)
- Worldmap: 85% compliant (TileContents needs field rename)
- Status effects: 70% compliant (interface needs work)

**Testing Coverage:**
- 8 test files present
- Squad system has comprehensive tests
- Most systems lack unit tests
- Integration testing through manual play

**Code Quality Metrics:**
- Average file size: 213 LOC
- Largest files: 987 LOC (combatmode), 894 LOC (vx), 774 LOC (squadbuilder)
- 19 files with TODO comments
- 11 files mixing math/rand and crypto/rand

---

**Last Updated:** 2025-11-09
**Codebase Version:** Post-worldmap strategy pattern (2025-11-08)
