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

**Last Updated:** 2025-10-01
**Analysis Location:** `analysis/REFACTORING_PRIORITIES.md`

### Critical Path to Unblock Todos

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

#### 🟡 PRIORITY 2: Squad Combat Foundation (12-40 hours)
**Status:** 0% → Incremental Implementation
**Blocks:** AI system (high), balance (high), spawning quality (medium)
**Impact:** CRITICAL - Major architectural change for "command several squads" (todos.txt:25)

**Why Strategic:**
- Current 1v1 combat incompatible with squad-based gameplay
- Affects spawning (squads vs individuals), AI, and balance systems
- Can preserve existing `PerformAttack()` logic with wrapper pattern

**Incremental Approach:**
- **Phase 1** (12h): PlayerSquad wrapper, backward compatible
- **Phase 2** (16h): Multi-squad support, formations, turn system
- **Phase 3** (12h): Squad-aware spawning and AI integration

**Analysis:** See `analysis/combat_refactoring.md` for detailed architecture

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

### Implementation Timeline

**Week 1: Roadmap Completion ~~(8 hours)~~ 4 hours remaining**
1. ~~Entity Template System (4h)~~ ✅ COMPLETED - **Spawning unblocked**
2. GUI Button Factory (2h) - Quick win
3. Status Effects Quality (2h) - Completes roadmap

**Result:** Spawning system unblocked, 4 hours to 100% roadmap completion

**Weeks 2-4: Squad Combat Foundation (40 hours)**
- Phase 1: Non-breaking infrastructure (12h)
- Phase 2: Multi-squad support (16h)
- Phase 3: Integration with spawning/AI (12h)

**Result:** Ready for all todos implementation

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
- **Status:** UNBLOCKED - Entity Template System completed
- **Available:** Generic `CreateEntityFromTemplate()` factory with EntityConfig pattern

❌ **Squad Combat Features** (todos.txt line 25)
- **Blocked by:** Squad system doesn't exist (12-40 hours)
- **Approach:** Incremental Phase 1 can start after Entity Templates

❌ **Balance/Difficulty** (todos.txt line 13)
- **Blocked by:** Squad combat (high impact on difficulty calculation)
- **Workaround:** Basic individual entity difficulty possible

---

### Analysis Files

Detailed analysis available in `analysis/` directory:
- `REFACTORING_PRIORITIES.md` - Comprehensive priority analysis and timeline
- `combat_refactoring.md` - Squad combat architecture and migration strategy
- `roadmap_completion.md` - Detailed completion plan for roadmap items

**See these files for code examples, risk mitigation, and implementation details.**