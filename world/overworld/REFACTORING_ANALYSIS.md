# Overworld Package Refactoring Analysis

**Package:** `world/overworld/`
**Date:** 2026-01-26
**Analyzer:** Claude Sonnet 4.5
**Status:** Analysis Only - No Implementation

---

## Executive Summary

The overworld package implements a strategic layer with threat nodes, factions, and turn-based tick simulation. While the package demonstrates good ECS fundamentals and solid test coverage, there are opportunities for improvement in:

1. **ECS Compliance**: Minor violations of pure data component principle
2. **Code Organization**: Better alignment with query/system patterns
3. **Separation of Concerns**: Some system logic mixed with utilities
4. **Component Naming**: Inconsistent naming patterns
5. **File Structure**: Could benefit from better modularity

**Overall Assessment:** The package is functional and well-tested, but would benefit from targeted refactoring to achieve full ECS compliance and improved maintainability.

---

## 1. ECS Compliance Analysis

### 1.1 Pure Data Components - MOSTLY COMPLIANT ✅

**Strengths:**
- Most components are pure data (ThreatNodeData, OverworldFactionData, TickStateData, VictoryStateData)
- No methods that modify state on components
- Clean separation of data from logic

**Issues:**

#### Issue 1.1.1: GetThreatTypeParams violates pure data principle
**File:** `constants.go:132-183`
**Severity:** MEDIUM

```go
// GetThreatTypeParams returns parameters for each threat type
func GetThreatTypeParams(threatType ThreatType) ThreatTypeParams {
    switch threatType {
    case ThreatNecromancer:
        return ThreatTypeParams{
            BaseGrowthRate:   0.05,
            // ...
        }
    }
}
```

**Problem:** This function acts like a method lookup for ThreatType, but it's externalized. This is configuration data that should be:
1. Stored as ECS components OR
2. Loaded from a config file OR
3. Clearly documented as static configuration

**Recommendation:**
- Create a `ThreatTypeConfigComponent` that stores parameters per threat type
- Load from JSON/YAML config file at initialization
- This would make threat types data-driven and moddable

#### Issue 1.1.2: Enum String() methods are acceptable but could be improved
**File:** `constants.go:14-89`
**Severity:** LOW

The String() methods on ThreatType, FactionType, FactionIntent, and InfluenceEffect are acceptable Go convention, but:
- They're scattered across the file
- Some are missing (InfluenceEffect has no String() method)
- Could be generated automatically

**Recommendation:**
- Use go:generate with stringer tool: `//go:generate stringer -type=ThreatType,FactionType,FactionIntent,InfluenceEffect`
- This would ensure consistency and reduce boilerplate

### 1.2 EntityID Only - COMPLIANT ✅

**Excellent compliance:**
- All components use `ecs.EntityID` for relationships
- No entity pointer storage found
- ThreatNodeData.ThreatID, OverworldFactionData.FactionID, SquadChecker interface (victory.go:46-62)

**Example of correct pattern:**
```go
type ThreatNodeData struct {
    ThreatID ecs.EntityID  // ✅ Correct
    // ...
}
```

### 1.3 Query-Based Relationships - COMPLIANT ✅

**Strengths:**
- All entity lookups use queries or FindEntityByID
- No cached entity relationships
- Query functions properly separated in *_queries.go files

**Examples:**
- `faction_queries.go`: GetFactionByID, GetStrongestFaction, GetFactionsByType
- `threat_queries.go`: GetAllThreatNodes, GetThreatsInRadius, GetThreatsByType

### 1.4 System Functions - PARTIALLY COMPLIANT ⚠️

**Issues:**

#### Issue 1.4.1: System logic scattered across multiple files
**Files:** `faction_system.go`, `threat_system.go`, `tick_system.go`, `victory.go`
**Severity:** MEDIUM

**Problem:** System functions are not consistently organized:
- `faction_system.go` contains both system functions AND helper functions (scoring)
- `victory.go` contains both system functions AND query functions
- No clear naming convention for system vs query vs utility functions

**Recommendation:**
- Separate scoring/utility functions into `faction_scoring.go` or `faction_utils.go`
- Move victory query functions to `victory_queries.go`
- Establish naming convention:
  - System functions: `UpdateX`, `ExecuteX`, `ProcessX`, `CreateX`
  - Query functions: `GetX`, `FindX`, `CountX`, `HasX`
  - Utility functions: `calculateX`, `mapX`, `scoreX` (private)

### 1.5 Value-Based Map Keys - COMPLIANT ✅

**File:** `influence_cache.go:13-15`
```go
type InfluenceCache struct {
    cachedInfluence map[int]float64        // ✅ Value key
    trackedThreats  map[ecs.EntityID]int   // ✅ Value key
}
```

Excellent use of value-based map keys throughout the package.

---

## 2. File Organization Issues

### 2.1 File Structure - NEEDS IMPROVEMENT ⚠️

**Current Structure:**
```
overworld/
├── components.go           # All components in one file
├── init.go                 # Initialization
├── constants.go            # Enums + params + String() methods
├── faction_system.go       # System + utilities mixed
├── faction_queries.go      # Query functions
├── faction_test.go         # Tests
├── threat_system.go        # System functions
├── threat_queries.go       # Query functions
├── resources.go            # PlayerResources (separate concern)
├── events.go               # Event logging system
├── victory.go              # System + queries mixed
├── tick_system.go          # Master orchestration
├── influence_cache.go      # Optimization structure
├── encounter_translation.go # Combat bridge
└── utils.go                # Mixed utilities
```

**Problems:**

#### Issue 2.1.1: components.go mixes multiple subsystems
**File:** `components.go`
**Severity:** MEDIUM

**Current:** All components in one file (ThreatNode, Faction, Territory, StrategicIntent, TickState, Victory, PlayerResources, InfluenceCache)

**Problem:** As the system grows, this file will become unwieldy. Different subsystems should have separate component files.

**Recommendation:** Split into:
```
overworld/
├── threat_components.go      # ThreatNodeData, InfluenceData
├── faction_components.go     # OverworldFactionData, TerritoryData, StrategicIntentData
├── victory_components.go     # VictoryStateData
├── resources_components.go   # PlayerResourcesData
├── tick_components.go        # TickStateData
├── cache_components.go       # InfluenceCacheData
```

This follows the pattern recommended in CLAUDE.md for modular ECS subsystems.

#### Issue 2.1.2: constants.go mixes concerns
**File:** `constants.go`
**Severity:** LOW

**Current:** Enums + String() methods + GetThreatTypeParams config lookup + tuning constants

**Recommendation:** Split into:
```
overworld/
├── threat_types.go         # ThreatType enum + String()
├── faction_types.go        # FactionType + FactionIntent + String()
├── influence_types.go      # InfluenceEffect + String()
├── threat_params.go        # ThreatTypeParams + GetThreatTypeParams
├── tuning_constants.go     # All numeric constants
```

#### Issue 2.1.3: utils.go is vague
**File:** `utils.go`
**Severity:** LOW

**Current:** Contains `formatEventString`, `MapFactionToThreatType`, `getThreatTypeName`

**Problem:** "utils" is a code smell - indicates unclear responsibility

**Recommendation:**
- Move `formatEventString` to `events.go` (it's only used there)
- Move `MapFactionToThreatType` to `faction_threat_mapping.go` or inline into `faction_system.go`
- Delete `getThreatTypeName` (redundant with ThreatType.String())
- Delete `utils.go` after moving functions

### 2.2 Missing Files - IMPROVEMENT OPPORTUNITY ⚠️

**Recommendation:** Add these files for better organization:

1. **`faction_scoring.go`** - Extract scoring functions from faction_system.go:
   - ScoreExpansion
   - ScoreFortification
   - ScoreRaiding
   - ScoreRetreat

2. **`victory_queries.go`** - Extract query functions from victory.go:
   - IsPlayerDefeated
   - HasPlayerEliminatedAllThreats
   - HasPlayerDefeatedFactionType
   - GetTotalThreatInfluence
   - GetDefeatReason

3. **`resources_queries.go`** - Extract query functions from resources.go:
   - GetPlayerResources
   - CanAfford
   - GetGold
   - GetExperience
   - GetReputation

---

## 3. Naming Convention Issues

### 3.1 Component Naming - INCONSISTENT ⚠️

**Current Patterns:**
```go
ThreatNodeData         // ✅ Correct: *Data suffix
OverworldFactionData   // ✅ Correct: *Data suffix
TickStateData          // ✅ Correct: *Data suffix
VictoryStateData       // ✅ Correct: *Data suffix
PlayerResourcesData    // ✅ Correct: *Data suffix
InfluenceData          // ✅ Correct: *Data suffix
TerritoryData          // ✅ Correct: *Data suffix
StrategicIntentData    // ✅ Correct: *Data suffix
InfluenceCacheData     // ⚠️ Wrapper component, but OK
```

**Observation:** Actually quite consistent! All use *Data suffix as recommended in CLAUDE.md.

**Minor Issue:** Component variables don't follow consistent naming:
```go
var (
    ThreatNodeComponent       *ecs.Component  // ✅ Good
    OverworldFactionComponent *ecs.Component  // ⚠️ Could be FactionComponent
    TickStateComponent        *ecs.Component  // ✅ Good
)
```

**Recommendation:** Consider shortening `OverworldFactionComponent` to `FactionComponent` since the package is already `overworld`, the prefix is redundant.

### 3.2 Function Naming - MOSTLY CORRECT ✅

**Query Functions:** Follow GetX, FindX, CountX, HasX patterns consistently
```go
GetFactionByID()           // ✅
CountFactions()            // ✅
GetStrongestFaction()      // ✅
GetThreatsInRadius()       // ✅
HasPlayerEliminatedAllThreats()  // ✅
```

**System Functions:** Follow CreateX, UpdateX, ExecuteX patterns
```go
CreateFaction()            // ✅
UpdateFactions()           // ✅
EvolveThreatNode()         // ✅
ExpandTerritory()          // ✅
```

**Minor Issue:** Some functions lack consistency:
```go
IsTileOwnedByAnyFaction()  // Should be: HasFactionAtPosition()
MapFactionToThreatType()   // Should be: GetThreatTypeForFaction()
```

---

## 4. Separation of Concerns

### 4.1 Mixing System Logic with Queries - ISSUE ⚠️

**File:** `victory.go`
**Severity:** MEDIUM

**Problem:** The file mixes:
1. System functions (CheckVictoryCondition)
2. Query functions (IsPlayerDefeated, HasPlayerEliminatedAllThreats, GetDefeatReason)
3. Entity creation (CreateVictoryStateEntity)
4. Interface definition (SquadChecker)

**Current Structure:**
```go
// victory.go (213 lines)
├── Enums/Types (1-35)
├── Component Data (37-44)
├── Interface (46-62)
├── System Function (64-156)
├── Query Functions (158-258)
├── Entity Creation (260-277)
└── Utility Functions (279-324)
```

**Recommendation:** Split into:
```
victory/
├── victory_components.go    # VictoryStateData, VictoryCondition enum
├── victory_system.go        # CheckVictoryCondition, CreateVictoryStateEntity
├── victory_queries.go       # IsPlayerDefeated, HasPlayerEliminated*, GetDefeatReason
├── victory_interfaces.go    # SquadChecker interface
└── victory_test.go          # All tests
```

### 4.2 PlayerResources in Overworld Package - DESIGN QUESTION ⚠️

**File:** `resources.go`
**Severity:** LOW

**Current:** PlayerResourcesData lives in overworld package

**Question:** Should PlayerResources be a separate package? It's used by:
- Overworld (for granting rewards after combat)
- Potentially squad purchasing
- Potentially shop/trading systems

**Recommendation:**
- **Option A (Keep):** Resources are overworld strategic-layer concept, keep here
- **Option B (Move):** Move to separate `game_main/resources/` package for cross-system use
- **Option C (Common):** Move to `common/resources/` if multiple systems need it

**Decision Needed:** Depends on future use cases. If only overworld uses it, keep here. If squads/shop/trading need it, extract to separate package.

### 4.3 Event System - ACCEPTABLE BUT COULD IMPROVE 📊

**File:** `events.go`
**Severity:** LOW

**Current Design:**
- Global EventLog singleton
- Global OverworldRecorder singleton
- LogEvent() adds to both logs

**Issues:**
- Global state (not ideal, but pragmatic for logging)
- Mixes in-memory event log (UI) with persistent recording (export)

**Recommendation:**
- **Short-term:** Keep as-is, global logging is pragmatic
- **Long-term:** Consider event bus pattern or observer pattern for more flexibility
- **Optional:** Extract logging to `overworld/logging/` subpackage if it grows

---

## 5. Architecture & Design Patterns

### 5.1 Influence Cache - EXCELLENT DESIGN ✅

**File:** `influence_cache.go`
**Strengths:**
- Dirty flagging optimization
- Lazy recalculation
- ECS-managed via component wrapper
- Clear separation of concerns

**Example of good ECS pattern:**
```go
// Wrapped in ECS component
type InfluenceCacheData struct {
    Cache *InfluenceCache
}

// Singleton accessor
func GetInfluenceCache(manager *common.EntityManager) *InfluenceCache
```

This is a **reference implementation** for performance-critical caching in ECS.

### 5.2 Tick System Orchestration - GOOD DESIGN ✅

**File:** `tick_system.go`
**Strengths:**
- Clear master orchestration function (AdvanceTick)
- Sequential subsystem updates
- Victory check at end of tick
- IsGameOver flag prevents further ticks

**Minor Issue:** No explicit ordering documentation

**Recommendation:** Add comment documenting subsystem update order:
```go
// AdvanceTick executes subsystems in this order:
// 1. UpdateThreatNodes - threats evolve and spawn
// 2. UpdateFactions - faction AI executes intents
// 3. ProcessEvents - (currently no-op, reserved for future)
// 4. CheckVictoryCondition - win/loss evaluation
```

### 5.3 Encounter Translation - GOOD BRIDGE PATTERN ✅

**File:** `encounter_translation.go`
**Strengths:**
- Clean bridge between strategic and tactical layers
- Encapsulates threat-to-combat translation
- Reward calculation logic centralized

**Issue:** GetBaseThreatUnits has hardcoded composition
```go
// TODO. This should be configurable through a file
func GetBaseThreatUnits(threatType ThreatType) []UnitTemplate
```

**Recommendation:** Move to JSON config:
```json
{
  "threat_types": {
    "necromancer": {
      "base_units": [
        {"type": "Skeleton", "role": "Tank"},
        {"type": "Zombie", "role": "DPS"}
      ]
    }
  }
}
```

### 5.4 SquadChecker Dependency Injection - EXCELLENT PATTERN ✅

**File:** `victory.go:46-62`
```go
type SquadChecker interface {
    HasActiveSquads(manager *common.EntityManager) bool
}

var squadChecker SquadChecker

func SetSquadChecker(checker SquadChecker) {
    squadChecker = checker
}
```

**Strengths:**
- Avoids circular dependency with squads package
- Interface-based dependency injection
- Clean separation of concerns

This is a **reference implementation** for breaking circular dependencies in ECS.

---

## 6. Test Coverage Analysis

### 6.1 Test Organization - GOOD ✅

**Test Files:**
- `faction_test.go` (223 lines, 8 tests)
- `resources_test.go` (378 lines, 17 tests)
- `victory_test.go` (359 lines, 18 tests)
- `events_test.go` (exists, not read)
- `example_usage_test.go` (117 lines, example functions)

**Strengths:**
- Comprehensive test coverage for major systems
- Tests follow AAA pattern (Arrange, Act, Assert)
- Good use of testfx.NewTestEntityManager()
- Clear test naming (TestFactionCreation, TestFactionExpansion, etc.)

### 6.2 Missing Tests - GAPS IDENTIFIED ⚠️

**Untested Files:**
- `threat_system.go` - NO dedicated test file
- `tick_system.go` - NO dedicated test file
- `influence_cache.go` - Partially tested in faction_test.go, needs dedicated tests
- `encounter_translation.go` - NO test file

**Recommendation:** Create:
1. `threat_test.go` - Test threat evolution, spawning, destruction
2. `tick_system_test.go` - Test AdvanceTick orchestration, game over conditions
3. `encounter_translation_test.go` - Test threat-to-encounter translation, reward calculation

### 6.3 Integration Tests - MISSING ⚠️

**Observation:** Tests are mostly unit tests for individual systems. No integration tests for full tick cycles.

**Recommendation:** Add `overworld_integration_test.go`:
```go
func TestFullGameLoop(t *testing.T)
func TestVictoryScenario(t *testing.T)
func TestDefeatScenario(t *testing.T)
func TestFactionExpansionOverTime(t *testing.T)
```

---

## 7. Code Quality Issues

### 7.1 Magic Numbers - MINOR ISSUE ⚠️

**File:** `faction_system.go:260`
```go
// Check bounds (100x80 map)
if adj.X < 0 || adj.X >= 100 || adj.Y < 0 || adj.Y >= 80 {
```

**Problem:** Hardcoded map dimensions

**Recommendation:** Pass map dimensions as parameters or use constants:
```go
const (
    OverworldMapWidth  = 100
    OverworldMapHeight = 80
)
```

### 7.2 Printf Debugging - CLEANUP NEEDED ⚠️

**Files:** Multiple files use fmt.Printf for debugging
- `events.go:135` - LogEvent prints to console
- `encounter_translation.go:332` - TriggerCombatFromThreat prints
- `resources.go:52` - GrantResources prints
- `resources.go:148` - InitializePlayerResources prints

**Recommendation:**
- Replace with proper logging package (e.g., `log`, `logrus`, or custom logger)
- Add log levels (DEBUG, INFO, WARN, ERROR)
- Make logging configurable via config flag

### 7.3 Error Handling - ACCEPTABLE ✅

**Observation:** Most system functions return errors appropriately:
```go
func UpdateThreatNodes(manager *common.EntityManager, currentTick int64) error
func UpdateFactions(manager *common.EntityManager, currentTick int64) error
func AdvanceTick(manager *common.EntityManager) error
```

Good error propagation throughout the package.

### 7.4 Documentation - GOOD BUT COULD IMPROVE 📝

**Strengths:**
- Most public functions have comments
- Clear package-level comments in tick_system.go
- Example usage file (example_usage_test.go)

**Gaps:**
- Some functions lack "why" documentation (only "what")
- No package-level doc.go file
- Complex algorithms (faction AI scoring) lack explanation

**Recommendation:** Add `doc.go`:
```go
// Package overworld implements the strategic layer of TinkerRogue.
//
// The overworld operates on a turn-based tick system where threats evolve,
// factions compete for territory, and victory/defeat conditions are evaluated.
//
// Core Subsystems:
//   - Threat Nodes: Growing threats that spawn encounters
//   - Factions: AI-controlled strategic entities
//   - Influence System: Spatial pressure from threats
//   - Victory System: Win/loss condition evaluation
//   - Event Logging: Strategic event tracking and export
//
// Architecture:
// The overworld follows strict ECS principles with pure data components,
// query-based relationships, and system functions for behavior.
package overworld
```

---

## 8. Performance Considerations

### 8.1 Influence Cache Performance - OPTIMIZED ✅

**File:** `influence_cache.go`
**Analysis:**
- Lazy evaluation with dirty flagging
- O(1) lookups after cache build
- Only rebuilds when threats change
- Uses value-based map keys (fast)

**Measured Performance:** Not benchmarked, but design is sound.

**Recommendation:** Add benchmarks:
```go
func BenchmarkInfluenceCacheLookup(b *testing.B)
func BenchmarkInfluenceCacheRebuild(b *testing.B)
```

### 8.2 Query Performance - ACCEPTABLE ⚠️

**Observation:** Many functions iterate all entities:
- `CountFactions` - O(n) iteration
- `GetStrongestFaction` - O(n) iteration
- `GetThreatsInRadius` - O(n) iteration + distance calculation

**For Small Scale:** Acceptable (overworld typically has <100 threats, <10 factions)

**For Large Scale:** Could be optimized with spatial indexing

**Recommendation:**
- **Short-term:** Keep as-is, queries are infrequent (once per tick)
- **Long-term:** If >1000 threats, add spatial grid for radius queries

### 8.3 Territory Storage - POTENTIAL ISSUE ⚠️

**File:** `components.go:66-70`
```go
type TerritoryData struct {
    OwnedTiles    []coords.LogicalPosition  // List of controlled tiles
    BorderTiles   []coords.LogicalPosition  // Cached borders
    ContestedTile *coords.LogicalPosition   // Currently contested
}
```

**Issue:** Linear search required to check if position is owned (O(n) for each tile check)

**Current Usage:** `IsTileOwnedByAnyFaction` iterates all factions and all their tiles - O(factions × tiles)

**Recommendation:**
- Add map-based lookup: `map[coords.LogicalPosition]ecs.EntityID` for O(1) ownership checks
- Store in component or global system
- Update on faction expansion/retreat

---

## 9. Recommended Refactoring Priority

### Priority 1 (High Impact, Low Risk) 🔴

1. **Split components.go into subsystem files** (Issue 2.1.1)
   - Files: threat_components.go, faction_components.go, victory_components.go, etc.
   - Impact: Better code organization, easier navigation
   - Risk: Low (mechanical refactoring)

2. **Extract scoring functions to faction_scoring.go** (Issue 1.4.1)
   - Move ScoreExpansion, ScoreFortification, ScoreRaiding, ScoreRetreat
   - Impact: Clearer separation of concerns
   - Risk: Low (no logic changes)

3. **Split victory.go into victory_system.go and victory_queries.go** (Issue 4.1)
   - Impact: Consistent file structure across package
   - Risk: Low (mechanical refactoring)

4. **Add missing test files** (Issue 6.2)
   - Create threat_test.go, tick_system_test.go, encounter_translation_test.go
   - Impact: Better test coverage, catch bugs
   - Risk: Low (additive)

### Priority 2 (Medium Impact, Low Risk) 🟡

5. **Split constants.go into type-specific files** (Issue 2.1.2)
   - threat_types.go, faction_types.go, tuning_constants.go
   - Impact: Better code organization
   - Risk: Low (mechanical refactoring)

6. **Delete utils.go and move functions to proper homes** (Issue 2.1.3)
   - Impact: Eliminates vague "utils" file
   - Risk: Low (move functions to proper files)

7. **Replace fmt.Printf with proper logging** (Issue 7.2)
   - Impact: Better debugging control, production-ready
   - Risk: Low (replace calls)

8. **Add package documentation (doc.go)** (Issue 7.4)
   - Impact: Better developer onboarding
   - Risk: None (documentation only)

### Priority 3 (High Impact, Medium Risk) 🟠

9. **Convert GetThreatTypeParams to config-driven system** (Issue 1.1.1)
   - Load threat parameters from JSON/YAML
   - Impact: Data-driven design, moddable content
   - Risk: Medium (requires config loading infrastructure)

10. **Add territory ownership map optimization** (Issue 8.3)
    - O(1) ownership checks instead of O(n×m)
    - Impact: Performance improvement for large maps
    - Risk: Medium (requires careful state synchronization)

11. **Extract encounter composition to config files** (Issue 5.3)
    - Move GetBaseThreatUnits to JSON config
    - Impact: Data-driven encounters, easier balancing
    - Risk: Medium (requires config system)

### Priority 4 (Low Impact, Optional) ⚪

12. **Add integration tests** (Issue 6.3)
    - Full game loop tests
    - Impact: Better coverage of system interactions
    - Risk: Low (additive)

13. **Add performance benchmarks** (Issue 8.1)
    - Benchmark cache, queries
    - Impact: Performance visibility
    - Risk: None (additive)

14. **Consider extracting PlayerResources** (Issue 4.2)
    - Decision: Keep in overworld or extract to separate package?
    - Impact: Depends on future use cases
    - Risk: Low if extracted

---

## 10. Refactoring Implementation Plan

### Phase 1: File Organization (1-2 hours)
**Goal:** Improve code navigability without logic changes

1. Split components.go into subsystem files
2. Split constants.go into type-specific files
3. Extract faction scoring functions
4. Split victory.go
5. Delete utils.go

**Verification:** All tests pass, no logic changes

### Phase 2: Test Coverage (2-3 hours)
**Goal:** Achieve comprehensive test coverage

1. Create threat_test.go
2. Create tick_system_test.go
3. Create encounter_translation_test.go
4. Add integration tests

**Verification:** `go test -cover ./...` shows >80% coverage

### Phase 3: Quality Improvements (1-2 hours)
**Goal:** Production-ready code quality

1. Replace fmt.Printf with logging package
2. Add doc.go package documentation
3. Document complex algorithms
4. Fix magic numbers (map dimensions)

**Verification:** Code review, documentation review

### Phase 4: Data-Driven Design (3-4 hours)
**Goal:** Make content moddable

1. Create threat type config JSON schema
2. Implement config loader
3. Convert GetThreatTypeParams to config-driven
4. Convert encounter composition to config-driven

**Verification:** Load custom threat types from config, tests pass

### Phase 5: Performance Optimization (2-3 hours)
**Goal:** Optimize for scale

1. Add territory ownership map
2. Add performance benchmarks
3. Profile and optimize hot paths

**Verification:** Benchmarks show improvement

**Total Estimated Time:** 9-14 hours

---

## 11. Code Examples

### Example 1: Split Components File

**Before (components.go):**
```go
package overworld

var (
    ThreatNodeComponent         *ecs.Component
    OverworldFactionComponent   *ecs.Component
    // ... 8 more components
)

type ThreatNodeData struct { ... }
type OverworldFactionData struct { ... }
// ... 8 more data structs
```

**After (threat_components.go):**
```go
package overworld

import "github.com/bytearena/ecs"

var (
    ThreatNodeComponent *ecs.Component
    InfluenceComponent  *ecs.Component
)

// ThreatNodeData represents a growing threat on the overworld map.
// Pure data component - no methods, only fields.
type ThreatNodeData struct {
    ThreatID       ecs.EntityID
    ThreatType     ThreatType
    Intensity      int
    GrowthProgress float64
    GrowthRate     float64
    IsContained    bool
    SpawnedTick    int64
}

// InfluenceData represents spatial pressure from a threat.
type InfluenceData struct {
    Radius         int
    EffectType     InfluenceEffect
    EffectStrength float64
}
```

**After (faction_components.go):**
```go
package overworld

import "github.com/bytearena/ecs"
import "game_main/world/coords"

var (
    OverworldFactionComponent *ecs.Component
    TerritoryComponent        *ecs.Component
    StrategicIntentComponent  *ecs.Component
)

// OverworldFactionData represents a persistent strategic faction.
type OverworldFactionData struct {
    FactionID     ecs.EntityID
    FactionType   FactionType
    Strength      int
    TerritorySize int
    Disposition   int
    CurrentIntent FactionIntent
    GrowthRate    float64
}

// TerritoryData tracks tiles controlled by a faction.
type TerritoryData struct {
    OwnedTiles    []coords.LogicalPosition
    BorderTiles   []coords.LogicalPosition
    ContestedTile *coords.LogicalPosition
}

// StrategicIntentData represents faction's current goal.
type StrategicIntentData struct {
    Intent         FactionIntent
    TargetPosition *coords.LogicalPosition
    TicksRemaining int
    Priority       float64
}
```

### Example 2: Extract Scoring Functions

**Before (faction_system.go):**
```go
// Line 119-212: Scoring functions mixed with system logic
func EvaluateFactionIntent(...) { ... }
func ScoreExpansion(...) float64 { ... }
func ScoreFortification(...) float64 { ... }
func ScoreRaiding(...) float64 { ... }
func ScoreRetreat(...) float64 { ... }
func ExecuteFactionIntent(...) { ... }
```

**After (faction_system.go):**
```go
// Only system execution functions remain
func EvaluateFactionIntent(
    manager *common.EntityManager,
    entity *ecs.Entity,
    factionData *OverworldFactionData,
    intentData *StrategicIntentData,
) {
    // Score possible actions (scoring logic in faction_scoring.go)
    expandScore := ScoreExpansion(manager, entity, factionData)
    fortifyScore := ScoreFortification(manager, entity, factionData)
    raidScore := ScoreRaiding(manager, entity, factionData)
    retreatScore := ScoreRetreat(manager, entity, factionData)

    // Decision logic remains here
    // ...
}
```

**After (faction_scoring.go - NEW FILE):**
```go
package overworld

import (
    "game_main/common"
    "github.com/bytearena/ecs"
)

// ScoreExpansion evaluates faction's expansion potential.
// Returns score 0-10+, higher = more favorable.
//
// Factors:
//   - Faction strength (needs >5 to expand)
//   - Current territory size (penalized if >20)
//   - Faction type modifiers (cultists favor expansion)
func ScoreExpansion(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
    score := 0.0

    // Favor expansion when strong
    if factionData.Strength >= ExpansionStrengthThreshold {
        score += 5.0
    }

    // ... rest of scoring logic
    return score
}

// ScoreFortification evaluates defensive posture.
// Returns score 0-10+, higher = more favorable.
func ScoreFortification(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
    // ... scoring logic
}

// ScoreRaiding evaluates aggressive action potential.
func ScoreRaiding(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
    // ... scoring logic
}

// ScoreRetreat evaluates abandoning territory.
func ScoreRetreat(manager *common.EntityManager, entity *ecs.Entity, factionData *OverworldFactionData) float64 {
    // ... scoring logic
}
```

### Example 3: Config-Driven Threat Parameters

**Before (constants.go):**
```go
func GetThreatTypeParams(threatType ThreatType) ThreatTypeParams {
    switch threatType {
    case ThreatNecromancer:
        return ThreatTypeParams{
            BaseGrowthRate:   0.05,
            BaseRadius:       3,
            PrimaryEffect:    InfluenceSpawnBoost,
            CanSpawnChildren: true,
            MaxIntensity:     10,
        }
    // ... 5 more cases
    }
}
```

**After (threat_config.json - NEW FILE):**
```json
{
  "threat_types": {
    "necromancer": {
      "base_growth_rate": 0.05,
      "base_radius": 3,
      "primary_effect": "spawn_boost",
      "can_spawn_children": true,
      "max_intensity": 10
    },
    "bandit_camp": {
      "base_growth_rate": 0.08,
      "base_radius": 2,
      "primary_effect": "resource_drain",
      "can_spawn_children": false,
      "max_intensity": 7
    }
  }
}
```

**After (threat_config.go - NEW FILE):**
```go
package overworld

import (
    "encoding/json"
    "fmt"
    "os"
)

// ThreatConfig stores all threat type parameters loaded from config.
type ThreatConfig struct {
    ThreatTypes map[string]ThreatTypeParams `json:"threat_types"`
}

var globalThreatConfig *ThreatConfig

// LoadThreatConfig loads threat parameters from JSON file.
func LoadThreatConfig(filepath string) error {
    data, err := os.ReadFile(filepath)
    if err != nil {
        return fmt.Errorf("failed to read threat config: %w", err)
    }

    config := &ThreatConfig{}
    if err := json.Unmarshal(data, config); err != nil {
        return fmt.Errorf("failed to parse threat config: %w", err)
    }

    globalThreatConfig = config
    return nil
}

// GetThreatTypeParams retrieves parameters for a threat type.
// Now reads from loaded config instead of hardcoded switch.
func GetThreatTypeParams(threatType ThreatType) ThreatTypeParams {
    if globalThreatConfig == nil {
        // Fallback to defaults if config not loaded
        return getDefaultThreatParams(threatType)
    }

    key := threatTypeToConfigKey(threatType)
    if params, exists := globalThreatConfig.ThreatTypes[key]; exists {
        return params
    }

    return getDefaultThreatParams(threatType)
}

// threatTypeToConfigKey converts enum to config key.
func threatTypeToConfigKey(threatType ThreatType) string {
    switch threatType {
    case ThreatNecromancer:
        return "necromancer"
    case ThreatBanditCamp:
        return "bandit_camp"
    // ...
    }
}
```

---

## 12. Conclusion

The overworld package demonstrates solid ECS fundamentals with good test coverage. The code is functional and maintainable, but would benefit from:

1. **Better file organization** - Split components and separate concerns
2. **Clearer separation** - Extract scoring/utility functions
3. **Improved testability** - Add missing test coverage
4. **Data-driven design** - Move configuration to external files
5. **Production quality** - Replace debug prints with proper logging

**Recommended Approach:**
- Start with Phase 1 (file organization) - low risk, high clarity
- Add Phase 2 (test coverage) - ensures safety for further refactoring
- Proceed to Phases 3-5 based on project priorities

**No Blocking Issues:** The code is production-ready as-is. Refactoring is recommended for long-term maintainability, not critical bugs.

---

**End of Analysis**
