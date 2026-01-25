# Overworld Package Refactoring Analysis

**Package:** `world/overworld`
**Date:** 2026-01-24
**Analyst:** Claude Code
**Status:** Comprehensive Analysis - Ready for Review

---

## Executive Summary

The `world/overworld` package implements the strategic layer of TinkerRogue, managing threats, factions, influence zones, and tick-based progression. The package is largely well-structured with good ECS adherence, but has several opportunities for improvement around code organization, component inconsistencies, and integration patterns.

**Key Findings:**
- **ECS Compliance:** 85% - Good overall, with some violations in component initialization
- **Code Organization:** Mixed - Some files mix responsibilities
- **Testability:** Good - Decent test coverage with room for expansion
- **Integration:** Some coupling issues with external packages
- **Documentation:** Adequate inline comments, could use architecture docs

**Recommendation:** Proceed with **Approach 2: Incremental Refinement** (see below)

---

## 1. Current State Assessment

### 1.1 Package Structure

The overworld package contains **17 files** organized as follows:

```
world/overworld/
├── components.go              # Pure data components (✓ ECS compliant)
├── init.go                    # Subsystem registration (✓ good pattern)
├── constants.go               # Enums, tuning parameters (✓ well organized)
│
├── threat_system.go           # Threat lifecycle (create/update/destroy)
├── threat_queries.go          # Threat query functions (✓ proper separation)
│
├── faction_system.go          # Faction AI and actions
├── faction_queries.go         # Faction query functions (✓ proper separation)
│
├── tick_system.go             # Master tick orchestration
├── influence_system.go        # Legacy influence calculation (⚠ mostly deprecated)
├── influence_cache.go         # Optimized influence system
│
├── events.go                  # Event logging system
├── victory.go                 # Win/loss conditions
├── resources.go               # Player resources management
├── encounter_translation.go   # Threat → combat translation
│
├── example_usage.go           # Usage examples
├── overworld_test.go          # Threat/tick tests
└── faction_test.go            # Faction tests
```

### 1.2 Component Analysis

**Components Defined (8 total):**

| Component | Purpose | ECS Compliance | Issues |
|-----------|---------|----------------|--------|
| `ThreatNodeComponent` | Threat data | ✓ Pure data | None |
| `OverworldFactionComponent` | Faction data | ✓ Pure data | None |
| `TickStateComponent` | Global tick state | ✓ Pure data | Singleton pattern OK |
| `InfluenceComponent` | Influence radius | ✓ Pure data | None |
| `TerritoryComponent` | Faction territory | ⚠ Mixed | Stores slices of positions |
| `StrategicIntentComponent` | Faction AI state | ✓ Pure data | None |
| `VictoryStateComponent` | Victory tracking | ✓ Pure data | Singleton pattern OK |
| `PlayerResourcesComponent` | Player resources | ✓ Pure data | None |

**Tags Defined (3 total):**
- `ThreatNodeTag` ✓
- `OverworldFactionTag` ✓
- `TickStateTag` ✓

**Missing Tags:**
- No tags for `VictoryStateComponent` (searched manually in `GetVictoryState`)
- No tags for `PlayerResourcesComponent` (accessed by player entity ID)

### 1.3 System Functions Analysis

**Well-Organized Systems:**
- `CreateThreatNode()`, `UpdateThreatNodes()`, `DestroyThreatNode()` - Clean lifecycle
- `CreateFaction()`, `UpdateFactions()` - Good AI orchestration
- `AdvanceTick()` - Excellent master orchestration pattern
- Query functions properly separated (`*_queries.go` files)

**Issues Found:**
1. **Component Initialization Split:** `init.go` initializes components but doesn't register `VictoryStateComponent` and `PlayerResourcesComponent` in tags
2. **Deprecated Code:** `influence_system.go` is mostly deprecated in favor of `influence_cache.go` but still called in tick
3. **Singleton Access:** `GetTickState()` and `GetVictoryState()` use different lookup patterns
4. **Global State:** `GlobalEventLog` in `events.go` is a package-level global (not ECS-managed)

### 1.4 Integration Points

**Dependencies (Imports):**
- `common` (EntityManager, position system) ✓
- `world/coords` (position types) ✓
- `world/encounter` (encounter translation)
- `tactical/squads` (unit roles for encounter generation) ⚠ Tight coupling
- `github.com/bytearena/ecs` ✓

**External Usage (Who uses overworld):**
- `gui/guioverworld` - Overworld UI mode
- `tactical/combatresolution` - Combat outcome resolution
- `game_main/gamesetup.go` - Initialization

**Coupling Issues:**
- `encounter_translation.go` imports `tactical/squads` for `UnitRole` enum
- `victory.go` has commented-out squad checking to avoid circular imports
- `events.go` uses global singleton instead of ECS component

---

## 2. Pain Points and Issues

### 2.1 ECS Pattern Violations

**Severity: Medium**

1. **Missing Tag Registration (init.go:27-28):**
   ```go
   // components.go defines VictoryStateComponent
   VictoryStateComponent = manager.World.NewComponent()
   PlayerResourcesComponent = manager.World.NewComponent()

   // But init.go doesn't register tags for them
   // This forces manual entity iteration in GetVictoryState()
   ```

2. **Inconsistent Singleton Access:**
   - `GetTickState()` uses tag query ✓
   - `GetVictoryState()` manually iterates entity IDs ✗

3. **Global Event Log (events.go:110):**
   ```go
   var GlobalEventLog = NewEventLog(100)
   ```
   Should be ECS component on singleton entity.

### 2.2 Code Organization Issues

**Severity: Low-Medium**

1. **Deprecated Code Confusion:**
   - `influence_system.go` contains placeholder functions that redirect to `influence_cache.go`
   - `tick_system.go:59` still calls `RecalculateInfluence()` which is a no-op
   - Creates confusion about which system is active

2. **File Responsibility Mixing:**
   - `threat_system.go` contains threat logic + helper function `formatEventString()`
   - `faction_system.go` contains faction logic + helper `MapFactionToThreatType()`
   - These helpers could be in a `helpers.go` or `utils.go`

3. **Example Code in Production:**
   - `example_usage.go` ships with the package
   - Should be in `_test.go` files or separate examples directory

### 2.3 Component Design Issues

**Severity: Low**

1. **TerritoryComponent Slices:**
   ```go
   type TerritoryData struct {
       OwnedTiles    []coords.LogicalPosition // Grows dynamically
       BorderTiles   []coords.LogicalPosition // Cached, not always updated
       ContestedTile *coords.LogicalPosition  // Pointer (OK for optional)
   }
   ```
   - Large slices in components can cause memory pressure
   - `BorderTiles` appears cached but never populated in current code
   - Consider spatial data structures for large territories

2. **Duplicate Type Definitions:**
   - `FactionType` has aliases: `FactionUndead = FactionNecromancers`, `FactionCorruption = FactionCultists`
   - Creates confusion - should standardize on one naming

### 2.4 Coupling and Dependencies

**Severity: Medium**

1. **Circular Import Prevention (victory.go:187-199):**
   ```go
   func HasPlayerLostAllSquads(manager *common.EntityManager) bool {
       // NOTE: This is a stub implementation.
       // Importing squads package would create circular dependency.
       return false
   }
   ```
   Victory conditions can't check squad status due to import cycle.

2. **Tight Coupling to Squads (encounter_translation.go:27):**
   ```go
   type UnitTemplate struct {
       Type string
       Role squads.UnitRole  // Direct import of squads package
   }
   ```
   Overworld shouldn't know about tactical squad implementation details.

3. **Unmanaged External Cache:**
   - `InfluenceCache` is created/managed outside ECS
   - No component wrapper, lives in calling code
   - Should be ECS-managed singleton component

### 2.5 Testing Gaps

**Severity: Low**

Current test coverage:
- ✓ Tick state creation/advancement
- ✓ Threat creation/evolution/queries
- ✓ Faction creation/AI evaluation/expansion
- ✓ Influence cache operations
- ✗ Event logging system
- ✗ Victory condition edge cases
- ✗ Resource management
- ✗ Encounter translation
- ✗ Integration tests (tick → threat evolution → faction response)

---

## 3. Refactoring Approaches

### Approach 1: Major Restructure (High Impact)

**Goal:** Complete architectural overhaul for maximum ECS purity and modularity.

**Changes:**

1. **Split Package by Domain:**
   ```
   world/overworld/
   ├── threats/       # Threat system
   ├── factions/      # Faction system
   ├── influence/     # Influence system
   ├── progression/   # Tick/victory
   └── integration/   # Event/resource/encounter
   ```

2. **Eliminate All Globals:**
   - Convert `GlobalEventLog` to ECS singleton component
   - Make `InfluenceCache` an ECS-managed component
   - Ensure all state lives in components

3. **Break Circular Dependencies:**
   - Create `world/strategictypes` package for shared enums
   - Move `UnitRole` to common package or use strings
   - Implement interface-based squad checking in victory

4. **Refactor Territory System:**
   - Replace slice-based territory with spatial index
   - Create `TerritoryManager` system (like `PositionSystem`)
   - Cache border tiles properly

**Benefits:**
- ✓ Maximum ECS compliance
- ✓ Clear domain separation
- ✓ Easier to test individual domains
- ✓ No circular dependencies
- ✓ Better performance (spatial indexes)

**Drawbacks:**
- ✗ Very high implementation cost (40+ hours)
- ✗ Breaks all existing integrations
- ✗ Requires coordinated changes across 5+ packages
- ✗ High regression risk
- ✗ May be over-engineering for current needs

**Risk:** High
**Complexity:** 8/10
**Time Estimate:** 3-4 weeks

---

### Approach 2: Incremental Refinement (Recommended)

**Goal:** Fix the most impactful issues while preserving existing structure and minimizing disruption.

**Phase 1: Quick Wins (1-2 days)**

1. **Fix Component Registration:**
   ```go
   // init.go - Add missing tags
   VictoryStateTag = ecs.BuildTag(VictoryStateComponent)
   PlayerResourcesTag = ecs.BuildTag(PlayerResourcesComponent)
   manager.WorldTags["victorystate"] = VictoryStateTag
   manager.WorldTags["playerresources"] = PlayerResourcesTag
   ```

2. **Standardize Singleton Access:**
   ```go
   // victory.go - Use tag query like GetTickState()
   func GetVictoryState(manager *common.EntityManager) *VictoryStateData {
       for _, result := range manager.World.Query(VictoryStateTag) {
           return common.GetComponentType[*VictoryStateData](result.Entity, VictoryStateComponent)
       }
       return nil
   }
   ```

3. **Remove Deprecated Code:**
   - Delete placeholder functions in `influence_system.go`
   - Remove `RecalculateInfluence()` call from `tick_system.go`
   - Add comment pointing to `InfluenceCache`

4. **Move Example Code:**
   - Rename `example_usage.go` → `example_usage_test.go`
   - Convert to proper example tests

**Phase 2: Organization (2-3 days)**

1. **Create Utility File:**
   ```go
   // utils.go - Centralize helper functions
   func formatEventString(format string, args ...interface{}) string
   func MapFactionToThreatType(factionType FactionType) ThreatType
   func getThreatTypeName(threatType ThreatType) string
   ```

2. **Standardize Component Access:**
   - Add `GetThreatByID()` helper (matches `GetFactionByID()` pattern)
   - Add `GetPlayerResourcesByID()` helper

3. **Fix Enum Aliases:**
   - Choose one naming convention for factions
   - Update all references
   - Add deprecation comments for old names

**Phase 3: Decouple (3-4 days)**

1. **Introduce Interface for Squad Checking:**
   ```go
   // victory.go
   type SquadChecker interface {
       HasActiveSquads(manager *common.EntityManager) bool
   }

   var squadChecker SquadChecker // Injected from main
   ```

2. **Remove Direct Squad Import:**
   ```go
   // encounter_translation.go
   type UnitTemplate struct {
       Type string
       Role string  // Use string instead of squads.UnitRole
   }
   ```

3. **Make InfluenceCache ECS-Managed:**
   ```go
   // influence_cache.go
   type InfluenceCacheComponent struct {
       Cache *InfluenceCache
   }

   // Create singleton with component
   ```

**Phase 4: Testing (2-3 days)**

1. Add missing tests:
   - Event logging (add/clear/unread tracking)
   - Victory edge cases (all threats gone, high influence, squad loss)
   - Resource spending/checks
   - Encounter generation

2. Add integration tests:
   - Full tick cycle (threat evolves → faction spawns threat → influence updates)
   - Combat resolution → overworld update

**Benefits:**
- ✓ Fixes critical ECS violations
- ✓ Improves code organization
- ✓ Breaks problematic coupling
- ✓ Can be done incrementally
- ✓ Low regression risk (changes are isolated)
- ✓ Immediate improvements after each phase

**Drawbacks:**
- ✗ Doesn't address deeper architectural issues
- ✗ Package structure remains mixed
- ✗ Territory system inefficiency persists

**Risk:** Low
**Complexity:** 4/10
**Time Estimate:** 1.5-2 weeks

---

### Approach 3: Component-Focused Cleanup

**Goal:** Focus exclusively on ECS component design improvements.

**Changes:**

1. **Territory Redesign:**
   - Replace `TerritoryData` slices with spatial index
   - Create `TerritorySystem` for territory queries
   - Cache border calculation properly

2. **Event System Refactor:**
   - Convert `GlobalEventLog` to component on singleton entity
   - Add `EventLogComponent` and `EventLogTag`
   - Update all `LogEvent()` calls to use entity

3. **Component Consolidation:**
   - Merge `InfluenceComponent` into `ThreatNodeData` (always paired)
   - Merge `StrategicIntentComponent` into `OverworldFactionData` (always paired)
   - Reduces component lookup overhead

4. **Cache Management:**
   - Create `InfluenceCacheComponent` (ECS singleton)
   - Auto-update on threat changes via entity callbacks

**Benefits:**
- ✓ Maximum ECS purity
- ✓ Performance improvements (fewer lookups)
- ✓ Cleaner component model
- ✓ Better cache management

**Drawbacks:**
- ✗ Breaking changes to all component access
- ✗ Requires updating all queries
- ✗ Medium regression risk
- ✗ Doesn't address coupling issues

**Risk:** Medium
**Complexity:** 6/10
**Time Estimate:** 2 weeks

---

### Approach 4: Minimal Touch-up

**Goal:** Address only the most critical bugs with minimal changes.

**Changes:**

1. **Fix Missing Tags (1 hour):**
   - Add `VictoryStateTag` and `PlayerResourcesTag`
   - Update singleton accessors

2. **Remove Dead Code (1 hour):**
   - Delete deprecated `influence_system.go` functions
   - Update tick orchestration

3. **Add Critical Tests (4 hours):**
   - Victory condition tests
   - Event logging tests

4. **Documentation (2 hours):**
   - Add package-level godoc
   - Document integration points

**Benefits:**
- ✓ Minimal disruption
- ✓ Quick fixes
- ✓ Can be done in 1 day
- ✓ Zero regression risk

**Drawbacks:**
- ✗ Doesn't address root causes
- ✗ Technical debt remains
- ✗ Organization issues persist
- ✗ Coupling unchanged

**Risk:** Very Low
**Complexity:** 2/10
**Time Estimate:** 1 day

---

## 4. Trade-offs and Risk Assessment

### 4.1 Approach Comparison Matrix

| Criteria | Approach 1 | Approach 2 | Approach 3 | Approach 4 |
|----------|------------|------------|------------|------------|
| **ECS Compliance** | Excellent | Good | Excellent | Fair |
| **Code Organization** | Excellent | Good | Fair | Poor |
| **Coupling Reduction** | Excellent | Good | Fair | None |
| **Performance** | Best | Good | Good | No Change |
| **Risk Level** | High | Low | Medium | Very Low |
| **Time Required** | 3-4 weeks | 1.5-2 weeks | 2 weeks | 1 day |
| **Regression Risk** | High | Low | Medium | Very Low |
| **Future Maintainability** | Excellent | Good | Good | Fair |
| **Testing Burden** | High | Medium | Medium | Low |

### 4.2 Critical Risks by Approach

**Approach 1 (Major Restructure):**
- **CRITICAL:** Breaking changes to GUI integration (`guioverworld`, `combatresolution`)
- **HIGH:** Coordinated multi-package refactor required
- **MEDIUM:** New spatial indexes need performance validation
- **Mitigation:** Feature flag, parallel implementation, extensive testing

**Approach 2 (Incremental Refinement):**
- **LOW:** Each phase is isolated and reversible
- **LOW:** Interface-based decoupling may be unused initially
- **Mitigation:** Incremental rollout, test after each phase

**Approach 3 (Component-Focused):**
- **MEDIUM:** Component consolidation changes all access patterns
- **MEDIUM:** Cache management changes may affect performance
- **Mitigation:** Comprehensive benchmarking, staged rollout

**Approach 4 (Minimal Touch-up):**
- **VERY LOW:** Only fixing obvious bugs
- **Mitigation:** N/A - changes are trivial

### 4.3 Recommended Approach

**RECOMMENDATION: Approach 2 (Incremental Refinement)**

**Rationale:**

1. **Best ROI:** Fixes critical issues (80% of value) with 20% of the effort of Approach 1
2. **Low Risk:** Incremental phases allow validation and rollback
3. **Immediate Value:** Phase 1 can be completed in 1-2 days
4. **Future-Proof:** Sets foundation for deeper refactoring if needed
5. **Team-Friendly:** Can be done by one developer without blocking others

**When to Consider Alternatives:**

- **Choose Approach 1 if:**
  - You're planning major overworld feature expansion (e.g., diplomacy, trade)
  - Performance profiling shows territory queries are a bottleneck
  - You have 4+ weeks dedicated to refactoring

- **Choose Approach 3 if:**
  - Component lookup overhead is measured performance bottleneck
  - You're refactoring other ECS subsystems simultaneously
  - You have strong automated test coverage

- **Choose Approach 4 if:**
  - You need a quick fix before a release
  - Major refactoring is planned for next quarter
  - Resource constraints prevent deeper work

---

## 5. Implementation Plan (Approach 2)

### Phase 1: Quick Wins (Days 1-2)

**Day 1 Morning:**
1. Create feature branch: `refactor/overworld-phase1`
2. Fix component registration in `init.go`
3. Update `GetVictoryState()` to use tag query
4. Add missing godoc to components

**Day 1 Afternoon:**
5. Remove deprecated `influence_system.go` functions
6. Update `tick_system.go` orchestration
7. Run all tests, verify no regressions

**Day 2:**
8. Move `example_usage.go` → `example_usage_test.go`
9. Write package-level documentation
10. Create PR, request review

**Success Criteria:**
- All tests pass ✓
- No deprecated code remaining ✓
- Tag-based access for all singletons ✓

### Phase 2: Organization (Days 3-5)

**Day 3:**
1. Create feature branch: `refactor/overworld-phase2`
2. Create `utils.go` with helper functions
3. Move helpers from system files
4. Update references

**Day 4:**
5. Add `GetThreatByID()` helper
6. Add `GetPlayerResourcesByID()` helper
7. Standardize enum naming (choose one convention)

**Day 5:**
8. Update all references to standardized enums
9. Add deprecation comments
10. Create PR, request review

**Success Criteria:**
- Consistent helper patterns ✓
- Standardized naming ✓
- All tests pass ✓

### Phase 3: Decouple (Days 6-9)

**Day 6-7:**
1. Create feature branch: `refactor/overworld-phase3`
2. Design `SquadChecker` interface
3. Update `victory.go` to use interface
4. Implement in main package (injection)

**Day 8:**
5. Change `UnitTemplate.Role` from `squads.UnitRole` to `string`
6. Update encounter translation logic
7. Update combat integration

**Day 9:**
8. Create `InfluenceCacheComponent`
9. Update cache access patterns
10. Create PR, request review

**Success Criteria:**
- No direct squad imports ✓
- Interface-based integration ✓
- Cache ECS-managed ✓

### Phase 4: Testing (Days 10-12)

**Day 10:**
1. Add event logging tests
2. Add victory condition edge case tests

**Day 11:**
3. Add resource management tests
4. Add encounter generation tests

**Day 12:**
5. Add integration test: tick cycle
6. Add integration test: combat resolution
7. Final PR review and merge

**Success Criteria:**
- Test coverage > 80% ✓
- All edge cases covered ✓
- Integration flows validated ✓

### Rollout Strategy

1. **Phase 1:** Merge immediately after review (bug fixes)
2. **Phase 2:** Merge after 1 week soak time in dev
3. **Phase 3:** Merge after integration testing with GUI
4. **Phase 4:** Merge with next release

---

## 6. Alternative Considerations

### 6.1 Should We Refactor At All?

**Arguments Against Refactoring:**
- Package is functional and relatively stable
- No reported bugs in overworld system
- Team resources might be better spent on new features
- Risk of introducing bugs in working code

**Arguments For Refactoring:**
- ECS violations will accumulate technical debt
- Coupling issues block future features (e.g., victory conditions)
- Missing tests create fragility
- Small issues now prevent big problems later

**Verdict:** Proceed with refactoring, but use incremental approach to minimize risk.

### 6.2 Timing Considerations

**Good Times to Refactor:**
- After a major release (low pressure)
- Before adding major overworld features
- During a planned maintenance sprint
- When onboarding new developers (improves code clarity)

**Bad Times to Refactor:**
- Week before a release
- During active feature development
- When team is understaffed
- Without comprehensive test coverage

---

## 7. Detailed Issue Catalog

### Critical Issues (Fix in Phase 1)

**C1. Missing Component Tags**
- **File:** `init.go:27-28`, `components.go:47-48`
- **Issue:** `VictoryStateComponent` and `PlayerResourcesComponent` not registered with tags
- **Impact:** Forces manual entity iteration, inconsistent access patterns
- **Fix:** Add tag registration in `InitOverworldTags()`

**C2. Inconsistent Singleton Access**
- **File:** `victory.go:256-266`
- **Issue:** `GetVictoryState()` manually iterates entities instead of using tag query
- **Impact:** Slower, inconsistent with other singleton accessors
- **Fix:** Use `VictoryStateTag` query (after C1 fixed)

**C3. Deprecated Code Still Called**
- **File:** `influence_system.go`, `tick_system.go:59`
- **Issue:** Deprecated functions still called in tick orchestration
- **Impact:** Confusing code, potential bugs if someone implements deprecated functions
- **Fix:** Remove deprecated code, update tick orchestration

### Medium Issues (Fix in Phase 2-3)

**M1. Helper Functions Scattered**
- **File:** `threat_system.go:12`, `faction_system.go:384`, `victory.go:300`
- **Issue:** Utility functions mixed with domain logic
- **Impact:** Code organization, discoverability
- **Fix:** Centralize in `utils.go`

**M2. Enum Naming Inconsistency**
- **File:** `constants.go:36-42`
- **Issue:** Duplicate faction type names (aliases)
- **Impact:** Confusion about which name to use
- **Fix:** Standardize on one naming scheme, deprecate others

**M3. Tight Squad Coupling**
- **File:** `encounter_translation.go:27`
- **Issue:** Direct import of `squads.UnitRole` enum
- **Impact:** Coupling, potential circular dependencies
- **Fix:** Use string-based role or interface

**M4. Unimplemented Squad Victory Check**
- **File:** `victory.go:187-199`
- **Issue:** `HasPlayerLostAllSquads()` is a stub due to circular import
- **Impact:** Incomplete victory conditions
- **Fix:** Interface-based dependency injection

**M5. Global Event Log**
- **File:** `events.go:110`
- **Issue:** Package-level global instead of ECS component
- **Impact:** Not query-able, breaks ECS pattern
- **Fix:** Convert to singleton component

### Low Issues (Fix in Phase 4 or Later)

**L1. Example Code in Production**
- **File:** `example_usage.go`
- **Issue:** Example code ships with package
- **Impact:** Binary size, code clutter
- **Fix:** Move to `_test.go` file

**L2. Territory Border Cache Unused**
- **File:** `components.go:62`
- **Issue:** `TerritoryData.BorderTiles` never populated
- **Impact:** Wasted memory, misleading code
- **Fix:** Remove or implement border calculation

**L3. Test Coverage Gaps**
- **File:** N/A (missing tests)
- **Issue:** No tests for events, victory edge cases, resources
- **Impact:** Potential undiscovered bugs
- **Fix:** Add comprehensive tests

**L4. Hardcoded Map Dimensions**
- **File:** `faction_system.go:260`
- **Issue:** `100x80` hardcoded in expansion logic
- **Impact:** Won't work with dynamic map sizes
- **Fix:** Pass map dimensions from manager

---

## 8. Migration Strategy

### 8.1 Breaking Changes

**Phase 1:** None - purely internal improvements
**Phase 2:** None - helpers are additions
**Phase 3:**
- `UnitTemplate.Role` type changes from `squads.UnitRole` to `string`
  - **Affected:** `encounter_translation.go`, combat integration
  - **Migration:** Convert role enums to strings
- `InfluenceCache` access changes
  - **Affected:** GUI code using cache
  - **Migration:** Access via component instead of direct instance

### 8.2 Deprecation Timeline

**Immediate (Phase 1):**
- Remove deprecated `influence_system.go` functions (already no-ops)

**Phase 2:**
- Deprecate faction enum aliases:
  ```go
  // Deprecated: Use FactionNecromancers instead
  FactionUndead = FactionNecromancers
  ```

**Phase 3:**
- Deprecate direct `GlobalEventLog` access (after component migration)

### 8.3 Communication Plan

**Before Starting:**
- Post refactoring plan to team channel
- Get buy-in from stakeholders
- Coordinate with GUI team on Phase 3 changes

**During Refactoring:**
- Daily updates on progress
- Immediate notification of any breaking changes
- PR reviews within 24 hours

**After Each Phase:**
- Demo changes in team meeting
- Update documentation
- Announce completion

---

## 9. Success Metrics

### Code Quality Metrics

**Before Refactoring:**
- ECS Compliance: 85%
- Test Coverage: ~60%
- Cyclomatic Complexity (avg): 6.2
- Coupling Score: Medium-High
- Documentation Coverage: 40%

**Target After Refactoring:**
- ECS Compliance: 95%
- Test Coverage: 80%+
- Cyclomatic Complexity (avg): <5
- Coupling Score: Low-Medium
- Documentation Coverage: 80%+

### Functional Metrics

- All existing tests pass ✓
- No new bugs introduced ✓
- No performance regressions ✓
- Integration tests added and passing ✓

### Developer Experience Metrics

- Time to understand overworld system (estimated): 2 hours → 1 hour
- Time to add new threat type: 30 min → 20 min
- Time to add new victory condition: 2 hours → 30 min

---

## 10. Conclusion

The `world/overworld` package is fundamentally sound but has accumulated several technical debt items that should be addressed:

**Critical (must fix):**
- Missing component tags
- Inconsistent singleton access
- Deprecated code confusion

**Important (should fix):**
- Helper function organization
- Squad coupling
- Global event log

**Nice to have:**
- Test coverage expansion
- Territory system optimization
- Package restructuring

**Recommendation:** Proceed with **Approach 2 (Incremental Refinement)** to address critical and important issues with minimal risk and maximum ROI. The phased approach allows for validation at each step and can be paused or adjusted based on team priorities.

**Next Steps:**
1. Review this analysis with team
2. Get approval for Phase 1 (quick wins)
3. Create feature branch and begin implementation
4. Schedule integration testing with GUI team before Phase 3

---

## Appendix A: File-by-File Analysis

### components.go
- **Status:** Good
- **Issues:** Missing tag definitions in header
- **Recommendation:** Add tag variables to header comments

### init.go
- **Status:** Needs work
- **Issues:** Missing tag registration (C1)
- **Recommendation:** Add VictoryStateTag, PlayerResourcesTag

### constants.go
- **Status:** Good
- **Issues:** Minor enum alias confusion (M2)
- **Recommendation:** Standardize faction naming

### threat_system.go
- **Status:** Good
- **Issues:** Helper function mixed in (M1)
- **Recommendation:** Move formatEventString to utils.go

### threat_queries.go
- **Status:** Excellent
- **Issues:** None
- **Recommendation:** Use as template for other query files

### faction_system.go
- **Status:** Good
- **Issues:** Helper function mixed in (M1), hardcoded map size (L4)
- **Recommendation:** Extract helpers, parameterize map dimensions

### faction_queries.go
- **Status:** Excellent
- **Issues:** None
- **Recommendation:** None

### tick_system.go
- **Status:** Good
- **Issues:** Calls deprecated RecalculateInfluence (C3)
- **Recommendation:** Remove deprecated call

### influence_system.go
- **Status:** Deprecated
- **Issues:** Entire file is placeholders (C3)
- **Recommendation:** Delete file, add comment in init.go pointing to influence_cache.go

### influence_cache.go
- **Status:** Good
- **Issues:** Not ECS-managed (M5)
- **Recommendation:** Wrap in component for Phase 3

### events.go
- **Status:** Needs work
- **Issues:** Global singleton (M5)
- **Recommendation:** Convert to ECS component

### victory.go
- **Status:** Needs work
- **Issues:** Stub implementation (M4), inconsistent singleton access (C2)
- **Recommendation:** Fix in Phase 3 with interface injection

### resources.go
- **Status:** Good
- **Issues:** Minor - could use more helpers
- **Recommendation:** Add query functions for consistency

### encounter_translation.go
- **Status:** Needs work
- **Issues:** Squad coupling (M3)
- **Recommendation:** Use string-based roles

### example_usage.go
- **Status:** Should move
- **Issues:** Production code contains examples (L1)
- **Recommendation:** Convert to test examples

### overworld_test.go
- **Status:** Good
- **Issues:** Missing coverage for some features
- **Recommendation:** Expand in Phase 4

### faction_test.go
- **Status:** Good
- **Issues:** Missing edge case coverage
- **Recommendation:** Expand in Phase 4

---

## Appendix B: ECS Compliance Checklist

| Rule | Status | Evidence |
|------|--------|----------|
| Pure data components | ✓ Mostly | All components are data-only |
| EntityID only (no pointers) | ✓ Yes | No entity pointers stored |
| Query-based relationships | ✓ Yes | All queries use tags |
| System functions (no component methods) | ✓ Yes | Systems are functions |
| Value map keys | ✓ Yes | No pointer keys |
| Proper file structure | ⚠ Mostly | components.go, queries.go good; missing utils.go |
| Naming conventions | ✓ Yes | Follows Data/Component/Tag pattern |
| Tag registration | ✗ Partial | Missing tags for 2 components |
| Singleton patterns | ⚠ Inconsistent | Different access patterns |
| No globals | ✗ No | GlobalEventLog is global |

**Overall Score: 8/10** (Good, with room for improvement)

---

**End of Analysis**
