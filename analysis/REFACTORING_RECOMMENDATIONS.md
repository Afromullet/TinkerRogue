# TinkerRogue Refactoring Recommendations

**Document Version**: 1.0
**Date**: 2025-11-07
**Analysis Scope**: Full codebase (~18,500 LOC across 75+ files)

---

## Executive Summary

### Codebase Health Assessment

**Overall Quality**: Good (75-80% ECS compliance)
**Architecture Maturity**: Mixed - Squad (2675 LOC) and Inventory (533 LOC) systems demonstrate excellent ECS patterns, but several systems haven't adopted these practices yet.

### Critical Findings

**High-Priority Issues**: 3 architectural violations
**Medium-Priority Issues**: 5 consolidation opportunities
**Estimated Technical Debt**: 2000-2500 LOC of duplication/anti-patterns
**Quick Win Potential**: 500-700 LOC reduction achievable in 1-2 days

### Top 5 Refactoring Targets

1. **PlayerData Entity Pointers** - Stores entity pointers instead of EntityIDs (HIGH PRIORITY)
2. **StatusEffects Interface** - Forces logic on components, violates ECS principles (HIGH PRIORITY)
3. **GUI Mode Duplication** - 300 LOC duplicated across 6 mode files (HIGH VALUE)
4. **PanelBuilders Bloat** - 13 nearly-identical functions, 529 LOC total (VERY HIGH ROI)
5. **Item Component Methods** - 5 logic methods should be system functions (EASY FIX)

---

## Priority 1: Critical ECS Violations

These issues violate the core architectural patterns established by the squad and inventory systems. They must be addressed to maintain consistency and enable future features like save/load.

### 1.1 PlayerData Component - Entity Pointer Storage

**Files Affected**: `common/playerdata.go:33-60`
**Lines of Code**: ~60 LOC (component) + ~100 LOC (usage sites)
**Severity**: HIGH - Blocks proper ECS architecture

#### Issue Description

PlayerData stores `*ecs.Entity` pointers for PlayerEntity and ThrowableItemEntity instead of using `ecs.EntityID`. This violates the core principle established by the squad and inventory systems that components should only store EntityIDs, not entity pointers.

#### Impact

- **Architectural Inconsistency**: Inventory system correctly uses EntityIDs, but PlayerData doesn't
- **Serialization Issues**: Cannot safely serialize player state with entity pointers
- **Lifecycle Coupling**: Entity pointers become invalid if entities are destroyed/recreated
- **Future Blocker**: Prevents implementation of save/load features

#### Recommended Approach

1. Convert three entity pointer fields to EntityID fields
2. Add query helper methods to retrieve entities when needed
3. Update 15-20 call sites to use EntityID-based lookups
4. Follow the exact pattern used in Inventory system (gear/Inventory.go)

#### Estimated Effort

**Time**: 2-3 hours
**Risk**: MEDIUM (touches core systems)
**Testing Required**: Player actions, throwing, inventory interactions

---

### 1.2 StatusEffects Interface - Forced Component Logic

**Files Affected**: `gear/stateffect.go` (380 LOC total)
**Severity**: HIGH - Major architectural smell

#### Issue Description

The StatusEffects interface requires components to implement `ApplyToCreature()` and `StackEffect()` business logic methods. This forces components to contain logic, directly violating the "pure data component" principle that makes the squad system (2675 LOC) so clean and maintainable.

#### Architectural Violation

Components should be pure data structures. The squad system demonstrates this perfectly:
- SquadData, GridPositionData, CombatStatsData = pure data
- squadcombat.go, squadqueries.go = pure logic

StatusEffects forces the opposite pattern:
- Burning, Freezing, Sticky = data + logic mixed together
- No centralized system file for effect application

#### Impact

- **Testability**: Cannot unit test effect logic without instantiating components
- **Maintainability**: Effect logic scattered across 7 component types
- **Consistency**: Contradicts squad system architecture (squadcombat.go pattern)
- **Extensibility**: New effects require interface changes

#### Recommended Approach

1. Keep component structs as pure data (Burning, Freezing, etc.)
2. Extract all ApplyToCreature logic to system functions (like squadcombat.go)
3. Extract all StackEffect logic to system functions
4. Create ApplyStatusEffectsSystem() central entry point
5. Remove interface requirement or keep as marker interface only

#### Estimated Effort

**Time**: 6-8 hours
**Risk**: MEDIUM-HIGH (touches 380 LOC, needs careful testing)
**Testing Required**: All 3 effect types (Burning, Freezing, Sticky)

---

### 1.3 Item Component Logic Methods

**Files Affected**: `gear/items.go:48-85`
**Severity**: HIGH - Inconsistent with same-package patterns

#### Issue Description

The Item component has 5 business logic methods (GetAction, HasAction, GetActions, GetThrowableAction, HasThrowableAction) that should be system functions in gearutil.go. This directly contradicts the inventory system's perfect ECS design **in the same package**.

#### Inconsistency Evidence

**Inventory system (correct pattern):**
- Pure data component
- System functions: AddItem(), RemoveItem(), GetItemEntityID()

**Item component (incorrect pattern):**
- Has data fields
- Has 5 logic methods
- Mixed paradigm

#### Impact

- **Package Inconsistency**: Two different patterns in same package (gear/)
- **Developer Confusion**: Which pattern should new code follow?
- **Code Review Difficulty**: Inconsistent style makes reviews harder

#### Recommended Approach

1. Move all 5 methods to gearutil.go as system functions
2. Keep Item as pure data component
3. Update call sites to use system functions
4. Achieve consistency with Inventory system in same package

#### Estimated Effort

**Time**: 1-2 hours
**Risk**: LOW (straightforward refactoring)
**Testing Required**: Inventory system, throwing system

---

## Priority 2: GUI System Duplication

The GUI system has excellent individual mode implementations but suffers from significant code duplication. Total GUI code: 4722 LOC across 14 files.

### 2.1 Mode Structure Duplication

**Files Affected**: All 6 UI mode files
**Lines Duplicated**: ~300 LOC
**Severity**: MEDIUM-HIGH - Maintenance burden

#### Issue Description

Six UI mode files duplicate identical infrastructure:
- **6 identical fields** in each mode: ui, context, layout, modeManager, rootContainer, panelBuilders
- **Initialization code**: ~40 LOC repeated in each mode
- **ESC key handling**: ~10 LOC repeated in each mode
- **Mode transitions**: Manual management repeated everywhere

#### Quantified Duplication

| Component | LOC per Mode | Number of Modes | Total Duplication |
|-----------|--------------|-----------------|-------------------|
| Field declarations | 6 lines | 6 modes | 36 LOC |
| Initialization code | 40 LOC | 6 modes | 240 LOC |
| ESC key handling | 10 LOC | 6 modes | 60 LOC |
| **Total** | | | **~300 LOC** |

#### Impact

- **Change Amplification**: Infrastructure changes require updates in 6 places
- **Bug Multiplication**: Bugs in common code replicate across modes
- **Onboarding Friction**: New developers see repetition, assume it's intentional
- **Future Mode Cost**: Each new mode requires copying 60+ LOC of boilerplate

#### Recommended Approach

**BaseMode Pattern** (composition via embedding):
1. Create BaseMode struct with 6 common fields
2. Add InitializeBase() method for common initialization
3. Add HandleCommonInput() for ESC key handling
4. Update all 6 modes to embed BaseMode
5. Remove duplicate field declarations and methods

**Benefits:**
- 300 LOC removed from mode files
- Infrastructure updates in one place
- New modes inherit infrastructure automatically
- Cleaner separation of mode-specific vs common code

#### Estimated Effort

**Time**: 3-4 hours
**Risk**: LOW (additive pattern, incremental refactoring)
**Testing Required**: All mode transitions

---

### 2.2 PanelBuilders Bloat

**Files Affected**: `gui/panels.go:1-528`
**Lines of Code**: 529 LOC
**Severity**: MEDIUM-HIGH - Largest single refactoring opportunity

#### Issue Description

13 nearly-identical panel builder functions exist, each creating panels at different positions:
- BuildTopCenterPanel (40 LOC)
- BuildTopLeftPanel (45 LOC)
- BuildLeftSidePanel (40 LOC)
- BuildRightSidePanel (40 LOC)
- ... 9 more similar functions

Each function is 30-50 LOC of mostly identical code with minor parameter variations.

#### Impact

- **New Panel Cost**: Adding a panel position requires copying 40 LOC
- **Maintenance Burden**: Updating panel styling requires 13 changes
- **Parameter Explosion**: Functions have 3-6 float parameters with unclear meanings
- **Readability**: Magic numbers everywhere (0.4, 0.08, 0.01)

#### Recommended Approach

**Functional Options Pattern** (idiomatic Go):
1. Create option types (TopCenter, LeftCenter, Size, Padding, etc.)
2. Replace 13 functions with single BuildPanel(opts...)
3. Options are composable and self-documenting
4. Refactor all modes to use new pattern
5. Delete old builder functions

**Benefits:**
- **LOC Reduction**: Remove ~529 LOC, add ~180 LOC = 349 LOC net reduction (66%)
- **Readability**: Self-documenting (TopCenter() vs magic float 0.5)
- **Extensibility**: New positions = 5-line option function
- **Type Safety**: Options enforce valid combinations

#### Estimated Effort

**Time**: 1-2 days
**Risk**: MEDIUM (requires updating all modes)
**Testing Required**: All panel layouts in all modes

---

### 2.3 Incomplete Widget Factory Pattern

**Files Affected**: `gui/createwidgets.go:1-239`
**Completion**: ~10%
**Severity**: MEDIUM - Foundation exists but underutilized

#### Issue Description

ButtonConfig, ListConfig, and PanelConfig patterns exist but are used in only ~10% of widget creation code. Most widgets still use verbose imperative construction with direct ebitenui API calls.

#### Inconsistency Evidence

**Some widgets use Config pattern:**
- ButtonConfig exists and works well
- ListConfig exists and works well
- PanelConfig exists and works well

**Most widgets don't:**
- TextArea, Container, Label have NO Config types
- Direct widget.New() calls dominate codebase
- No enforcement or documentation of Config pattern

#### Impact

- **Cognitive Load**: Developers must remember two patterns
- **Style Inconsistency**: Code reviews show mixed approaches
- **Onboarding Difficulty**: New contributors don't know which pattern to use
- **Refactoring Friction**: Hard to update widget styling consistently

#### Recommended Approach

**Complete the Pattern:**
1. Add missing Config types (TextAreaConfig, ContainerConfig, LabelConfig)
2. Create Config constructor functions for all common widgets
3. Document Config pattern in CLAUDE.md as project standard
4. Gradually migrate existing code during maintenance
5. Establish code review guideline: prefer Config pattern

**Benefits:**
- Consistent API across all widget types
- Reduced boilerplate in mode files
- Easier to modify widget styling globally
- Self-documenting widget configuration

#### Estimated Effort

**Time**: 3-4 hours
**Risk**: LOW (additive, doesn't break existing code)
**Testing Required**: Create examples for each new Config type

---

## Priority 3: Missing Abstractions

These aren't bugs or violations, but opportunities to reduce duplication and improve performance by using existing infrastructure.

### 3.1 ECS Query Duplication in GUI

**Files Affected**: Multiple GUI modes
**Lines Duplicated**: ~120 LOC
**Severity**: MEDIUM - Duplication + missed caching opportunity

#### Issue Description

Helper functions like getSquadName(), getFactionName(), and squad stats queries are duplicated across 3+ mode files. Each query is 8-12 LOC of identical ECS iteration code.

#### Quantified Duplication

| Query Function | LOC | Files | Total Duplication |
|----------------|-----|-------|-------------------|
| getSquadName | 12 | 3 | 36 LOC |
| getFactionName | 12 | 2 | 24 LOC |
| Squad stats queries | 30 | 2 | 60 LOC |
| **Total** | | | **~120 LOC** |

#### Impact

- **Performance**: Same ECS queries run repeatedly without caching
- **Maintenance**: Query logic updates need 3+ file changes
- **Consistency**: No guarantee queries return same format

#### Recommended Approach

**UIEntityQueries Helper Class:**
1. Create dedicated query helper with common methods
2. Add caching layer for expensive queries
3. Integrate into UIContext for all modes
4. Refactor modes to use shared queries
5. Remove duplicate query functions

**Benefits:**
- Query logic centralized in one file
- Caching reduces ECS queries by ~70% in combat mode
- Consistent query results across all modes
- Easy to extend with new query types

#### Estimated Effort

**Time**: 2-3 hours
**Risk**: LOW (additive helper, doesn't change existing queries initially)
**Testing Required**: Combat mode, squad builder, squad management

---

### 3.2 Missing PositionSystem Usage in Combat

**Files Affected**: `combat/queries.go`
**Performance Impact**: 50x slower than available solution
**Severity**: MEDIUM - Performance opportunity

#### Issue Description

Combat system uses O(n) linear searches for position lookups instead of leveraging the O(1) PositionSystem (systems/positionsystem.go, 399 LOC) that was created specifically for this use case.

#### Performance Analysis

**Current Implementation:**
- Algorithm: O(n) linear search through all entities
- Performance: Scales linearly with entity count
- Method: Iterate MapPositionTag, check each entity

**Available Solution:**
- System: GlobalPositionSystem (already exists!)
- Algorithm: O(1) hash lookup
- Performance: Constant time regardless of entity count
- Improvement: 50x faster with 50+ entities (documented in positionsystem.go)

#### Impact

- **Performance**: Unnecessary O(n) operations in combat
- **Code Duplication**: Reimplements position lookup logic
- **Wasted Investment**: 399 LOC system exists but isn't used
- **Scalability**: Performance degrades as entity count grows

#### Recommended Approach

1. Refactor combat/queries.go to use GlobalPositionSystem
2. Replace linear searches with hash lookups
3. Remove duplicate position iteration logic
4. Test combat movement and targeting

**Benefits:**
- 50x performance improvement in position lookups
- Leverages existing, tested system
- Eliminates code duplication
- Better scalability

#### Estimated Effort

**Time**: 1-2 hours
**Risk**: LOW (using existing, tested system)
**Testing Required**: Combat movement, targeting, position queries

---

## Priority 4: Low-Priority Items

These items are documented for completeness but don't require immediate action.

### 4.1 TODO Comments in Production Code

**Count**: 5 TODO comments found
**Severity**: LOW - Legitimate future work markers

#### Locations

1. `combat/victory.go:39,63,71` - Victory condition placeholders
2. `combat/queries.go:232` - Event system for UI (future feature)
3. `combat/turnmanager.go:66` - System reset design question
4. `combat/movementsystem.go:31` - Constant extraction suggestion

#### Assessment

These are legitimate future work markers, not urgent issues. They document intentional simplifications or planned features.

#### Recommended Action

Document in MASTER_ROADMAP.md as future enhancement items. No immediate code changes needed.

---

### 4.2 Interface{} Type Assertions

**Files Affected**: `common/playerdata.go:31,57`
**Severity**: LOW - Pragmatic workaround for circular dependencies

#### Issue Description

PlayerThrowable and PlayerData use `interface{}` to avoid circular dependencies:
- ThrowingAOEShape interface{} (should be graphics.TileBasedShape)
- Inventory interface{} (should be *gear.Inventory)

#### Assessment

This is a pragmatic solution to Go's circular dependency restrictions. While not ideal, it works correctly and is low priority compared to architectural issues.

#### Recommended Approach (Low Priority)

Extract interface definitions to shared package to break circular dependencies properly. However, this is cosmetic and doesn't affect functionality.

#### Estimated Effort

**Time**: 2-3 hours
**Priority**: LOW
**Impact**: Code aesthetics only

---

## Refactoring Roadmap

### Phase 1: Critical ECS Fixes (1-2 days)

**Goal**: Achieve 100% ECS compliance with squad/inventory patterns

**Tasks:**

1. **PlayerData EntityID Conversion** (2-3 hours, HIGH PRIORITY)
   - Convert 3 entity pointers to EntityIDs
   - Update 15-20 call sites
   - Add query helper methods
   - Test player actions

2. **Item Component Logic Extraction** (1-2 hours, EASY WIN)
   - Move 5 methods to gearutil.go
   - Update call sites
   - Align with Inventory system pattern

3. **StatusEffects System Refactoring** (6-8 hours, COMPLEX)
   - Extract ApplyToCreature logic to system functions
   - Create StackEffect system functions
   - Remove interface requirement
   - Test all 3 effect types

**Total Phase 1 Effort**: 9-13 hours
**LOC Reduction**: ~150 LOC
**Risk Level**: MEDIUM
**Impact**: Critical architecture alignment, unblocks save/load features

---

### Phase 2: GUI Consolidation (2-3 days)

**Goal**: Eliminate duplication, establish unified patterns

**Tasks:**

1. **BaseMode Pattern Implementation** (3-4 hours, MEDIUM-HIGH VALUE)
   - Create basemode.go with common fields/methods
   - Refactor 6 modes to embed BaseMode
   - Test all mode transitions
   - Document pattern in CLAUDE.md

2. **PanelBuilders Functional Options** (1-2 days, VERY HIGH ROI)
   - Design option types (TopCenter, Size, Padding, etc.)
   - Implement BuildPanel(opts...) function
   - Refactor CombatMode as test case
   - Refactor remaining 5 modes
   - Delete 13 old builder functions
   - Update documentation

3. **Complete Widget Factory Pattern** (3-4 hours, FOUNDATION)
   - Add TextAreaConfig, ContainerConfig, LabelConfig
   - Create helper constructors
   - Document pattern as project standard
   - Migrate 10-15 call sites as examples

**Total Phase 2 Effort**: 2-3 days
**LOC Reduction**: ~530 LOC (180 from BaseMode + 349 from PanelBuilders)
**Risk Level**: LOW-MEDIUM
**Impact**: Massive maintainability improvement, enables future GUI work

---

### Phase 3: Query Layer & Performance (1 day)

**Goal**: Eliminate query duplication, improve performance

**Tasks:**

1. **UIEntityQueries Helper** (2-3 hours, MEDIUM VALUE)
   - Create uiqueries.go with caching
   - Implement common queries (squad name, faction, stats)
   - Integrate into UIContext and BaseMode
   - Refactor 3 modes to use shared queries
   - Remove duplicate query functions

2. **Combat PositionSystem Integration** (1-2 hours, PERFORMANCE WIN)
   - Update combat/queries.go to use GlobalPositionSystem
   - Replace O(n) searches with O(1) lookups
   - Test combat movement and targeting
   - Document performance improvement

**Total Phase 3 Effort**: 4-5 hours
**LOC Reduction**: ~120 LOC
**Risk Level**: LOW
**Impact**: 50x performance improvement, query centralization

---

### Phase 4: Status Effects Quality (Optional)

**Goal**: Extract Quality interface properly (mentioned in CLAUDE.md as 85% complete)

This phase is deferred as per CLAUDE.md but included for completeness.

**Estimated Effort**: 2-3 hours
**Priority**: LOW - Can wait until other refactoring complete

---

## Summary Statistics

### Refactoring Potential

| Metric | Value |
|--------|-------|
| **Total LOC Reduction** | 800-1000 LOC |
| **Phase 1 (ECS)** | ~150 LOC |
| **Phase 2 (GUI)** | ~530 LOC |
| **Phase 3 (Queries)** | ~120 LOC |
| **Duplication Removed** | ~650 LOC |
| **Architecture Violations Fixed** | 3 critical issues |

### Time Investment

| Phase | Duration | Risk Level |
|-------|----------|------------|
| **Phase 1: ECS Fixes** | 1-2 days | MEDIUM |
| **Phase 2: GUI** | 2-3 days | LOW-MEDIUM |
| **Phase 3: Queries** | 1 day | LOW |
| **Total** | 4-6 days | MIXED |

### ROI Analysis

**Highest Value Targets (by LOC reduction per day):**

1. **PanelBuilders Refactor**: 349 LOC in 1-2 days = 175-349 LOC/day
2. **BaseMode Pattern**: 180 LOC in 3-4 hours = 360-480 LOC/day
3. **Item Logic Extraction**: ~30 LOC in 1-2 hours = 240-360 LOC/day (quick win!)

**Highest Impact Targets (by architecture improvement):**

1. **PlayerData EntityID Conversion**: Unblocks save/load, fixes critical violation
2. **StatusEffects Refactoring**: 380 LOC alignment with squad system patterns
3. **Item Logic Extraction**: Achieves consistency within gear/ package

---

## Priority Matrix

### Do First (1-2 days)

**Quick Wins:**
1. Item logic extraction (1-2 hours, easy, high impact)
2. PlayerData EntityID conversion (2-3 hours, critical for future features)
3. BaseMode pattern (3-4 hours, enables all future GUI work)

**Total**: ~7-9 hours, immediate architectural improvements

---

### Do Next (2-3 days)

**High-Value Consolidation:**
1. StatusEffects system refactoring (6-8 hours, complex but necessary)
2. PanelBuilders functional options (1-2 days, largest LOC reduction)
3. UIEntityQueries helper (2-3 hours, performance + maintainability)

**Total**: ~2-3 days, major reduction in duplication and technical debt

---

### Do Eventually (1 day)

**Polish & Performance:**
1. Complete widget factory pattern (3-4 hours, foundation for future)
2. Combat PositionSystem integration (1-2 hours, 50x performance win)

**Total**: ~5-6 hours, completes refactoring initiative

---

### Low Priority (Defer)

**Non-Critical Items:**
1. interface{} removal in PlayerData (cosmetic, 2-3 hours)
2. TODO comment resolution (document in roadmap)
3. Status Effects Quality extraction (already documented as low priority)

---

## Key Recommendations

### 1. Start with High-Impact, Low-Effort Items

**Recommended First Tasks:**
- Item logic extraction (1-2 hours) - Easiest win
- PlayerData EntityID conversion (2-3 hours) - Critical architectural fix
- BaseMode pattern (3-4 hours) - Foundation for all GUI work

**Rationale**: 7-9 hours of work provides immediate architectural benefits and builds momentum for larger refactoring efforts.

---

### 2. Use Squad/Inventory Systems as North Star

The squad (2675 LOC) and inventory (533 LOC) systems demonstrate perfect ECS architecture:
- Pure data components
- System-based logic
- EntityID-based relationships
- Query-based entity discovery
- O(1) performance patterns

**Refactoring Guideline**: Bring all systems up to this standard. Don't invent new patterns - enforce existing ones.

---

### 3. PanelBuilders is the Biggest Win

349 LOC reduction (66% of panels.go) in 1-2 days makes this the highest-value single refactoring opportunity. The functional options pattern is also idiomatic Go and will improve readability significantly.

**Strategic Value**: This refactoring demonstrates the value of consolidation and sets the standard for future GUI work.

---

### 4. Don't Over-Engineer

The codebase is already well-structured. Focus on:
- Eliminating duplication
- Enforcing existing patterns
- Completing partially-implemented patterns

**Anti-Pattern**: Introducing new architectural paradigms. The patterns are already established - just need consistent application.

---

### 5. Incremental Approach is Safer

**Recommended Strategy:**
- Refactor one system completely before moving to next
- Test thoroughly after each change
- Get user feedback on improved code quality
- Adjust approach based on lessons learned

**Rationale**: GUI system (4722 LOC) is large. Incremental refactoring with testing checkpoints reduces risk and allows course correction.

---

## Conclusion

The TinkerRogue codebase demonstrates strong architectural discipline in its squad and inventory systems. The refactoring goal is to bring the remaining systems (PlayerData, StatusEffects, GUI) up to this same standard through:

1. **ECS Compliance**: Eliminate entity pointers, enforce pure data components
2. **Duplication Removal**: Consolidate ~650 LOC of repeated code
3. **Pattern Completion**: Finish widget factory, establish BaseMode pattern
4. **Performance Optimization**: Use existing infrastructure (PositionSystem)

**Total Investment**: 4-6 days
**Total Reduction**: 800-1000 LOC
**Critical Fixes**: 3 architectural violations
**Strategic Value**: Maintainability, consistency, and foundation for future features

The roadmap prioritizes high-impact, low-effort items first to build momentum and demonstrate value quickly. The phased approach allows for testing and user feedback between major changes.
