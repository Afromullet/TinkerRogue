# TinkerRogue Master Development Roadmap

**Version:** 1.0
**Created:** 2025-10-02
**Total Estimated Time:** 106-144 hours (13-18 workdays)
**Status:** Ready for Implementation

---

## Executive Summary

### Current State

TinkerRogue is a Go-based roguelike game using the Ebiten engine with an ECS architecture (bytearena/ecs). The project has achieved 80% completion of its simplification roadmap, with major accomplishments including:

- **Completed Systems:** Input consolidation, coordinate standardization, entity templates, graphics shapes
- **Architecture:** Proper ECS patterns with native entity ID usage, query-based relationships
- **Recent Achievement:** Entity Template System completed, unblocking spawning implementation
- **Code Quality:** Mixed - new code follows proper ECS patterns, legacy code needs migration

### Strategic Goals

1. **Primary Goal:** Implement squad-based tactical combat system (similar to Symphony of War, Nephilim, Soul Nomad)
2. **Secondary Goals:** Complete todos implementation, finish simplification roadmap
3. **Architecture Goals:** Migrate legacy code to proper ECS patterns, achieve 100% pattern compliance

### Key Innovation: Testing Approach

**User wants to test squad building and combat BEFORE implementing map movement/representation.** This means:
- Squad creation and management can be tested in isolation
- Combat system can be validated without full map integration
- Map representation and movement come AFTER core squad mechanics are proven

### Total Time Estimate

- **Phase 0 (Prerequisites):** 4 hours
- **Phase 1 (Squad Core - Priority):** 36-48 hours
- **Phase 2 (Squad Integration):** 16-24 hours
- **Phase 3 (Legacy Refactoring):** 38-54 hours (can parallelize)
- **Phase 4 (Todos Implementation):** 8-12 hours
- **Phase 5 (Roadmap Completion):** 4 hours

**Total Sequential (Critical Path):** 60-92 hours
**Total Parallel Work Available:** 46-56 hours
**Overall Timeline:** 13-18 workdays

---

## Phase 0: Prerequisites and Setup

**Time Estimate:** 4 hours
**Dependencies:** None
**Can Start:** Immediately

### Deliverables

1. Complete GUI Button Factory (2 hours)
2. Complete Status Effects Quality Interface (2 hours)

### Why These First?

These are quick wins that complete the simplification roadmap and provide confidence-building momentum before tackling the major squad system implementation. Neither blocks any other work.

### Success Criteria

- [ ] GUI Button Factory: 3 duplicate functions consolidated into 1 factory with ButtonConfig pattern
- [ ] Status Effects: Quality interface extracted, effects no longer coupled to loot quality system
- [ ] Simplification roadmap progress: 80% → 95%
- [ ] Net code reduction: -35 lines
- [ ] Build succeeds, game runs without errors

### Testing Approach

- Verify all three menu buttons work identically after factory refactoring
- Confirm loot generation still applies quality correctly after extraction
- Run `go test ./...` to ensure no regressions

---

## Phase 1: Squad System Core (PRIORITY - Test Without Movement)

**Time Estimate:** 36-48 hours (4.5-6 workdays)
**Dependencies:** Phase 0 complete
**Testing:** Can validate squad combat WITHOUT map integration
**Risk Level:** Medium

### Strategic Importance

This is the HIGHEST PRIORITY work. The squad system enables:
- "Command several squads" gameplay (todos.txt:25)
- Tactical combat with formations and roles
- Foundation for AI improvements and balance system
- Testing squad mechanics in isolation before map integration

### Implementation Phases

#### Phase 1.1: Components and Data Structures (6-8 hours)

**Deliverables:**
- `squad/components.go` - 8 component types with pure data (350-400 lines)
- `squad/tags.go` - Tag initialization (30-40 lines)
- Updated `game_main/main.go` - Component registration

**Key Components:**
- SquadData - Squad entity with formation, morale, turn count
- SquadMemberData - Links units to parent squad via entity ID
- GridPositionData - 3x3 grid position with multi-cell support (Width/Height for 2x2 bosses, etc.)
- UnitRoleData - Tank/DPS/Support roles
- TargetRowData - Row-based combat targeting (front/mid/back)
- LeaderData - Leader bonuses and experience
- AbilitySlotData - 4 FFT-style auto-trigger ability slots
- CooldownTrackerData - Ability cooldown tracking

**Testing Without Map:**
- Components register successfully
- Helper methods work (GetOccupiedCells for multi-cell units)
- No panics on game startup

#### Phase 1.2: Query System (4-6 hours)

**Deliverables:**
- `systems/squadqueries.go` - Query helper functions (150-200 lines)

**Key Functions:**
- GetUnitIDsInSquad - Returns all units in a squad
- GetUnitIDsInRow - Returns alive units in specific row (handles multi-cell units)
- GetUnitIDsAtGridPosition - Returns units at specific cell
- GetLeaderID - Finds squad leader
- IsSquadDestroyed - Checks if all units are dead

**Testing Without Map:**
- Create test squad with 3 units
- Verify query functions return correct results
- Test multi-cell unit appears in multiple row queries
- Test deduplication works correctly

#### Phase 1.3: Row-Based Combat System (8-10 hours)

**Deliverables:**
- `systems/squadcombat.go` - Combat execution logic (300-350 lines)

**Key Features:**
- ExecuteSquadAttack - Main combat flow between two squads
- Row-based targeting (front row protects back row)
- Role modifiers (Tank -20% damage, DPS +30%, Support -40%)
- Single-target vs AOE logic
- Multi-cell unit targeting from any occupied row

**Testing Without Map:**
- Create two test squads (player and enemy)
- Execute combat programmatically
- Verify damage calculation, role modifiers, row targeting
- Test multi-cell units can be targeted from multiple rows
- Validate combat results structure

#### Phase 1.4: Automated Ability System (6-8 hours)

**Deliverables:**
- `systems/squadabilities.go` - Ability trigger and execution (300-350 lines)

**Key Features:**
- CheckAndTriggerAbilities - Evaluates 4 ability slots per leader
- Trigger conditions (HP threshold, turn count, combat start, enemy count, morale)
- Ability effects (Rally, Heal, Battle Cry, Fireball)
- Cooldown management

**Testing Without Map:**
- Equip abilities to test leader
- Manually set trigger conditions (HP, turn count)
- Verify abilities fire automatically
- Test cooldown system prevents repeated firing
- Validate ability effects apply correctly

#### Phase 1.5: Squad Creation and Management (4-6 hours)

**Deliverables:**
- `systems/squadcreation.go` - Squad creation functions (250-300 lines)
- `squad/formations.go` - Formation presets (100-150 lines)

**Key Features:**
- CreateSquadFromTemplate - Creates squad with units programmatically
- Formation presets (Balanced, Defensive, Offensive, Ranged)
- Multi-cell unit support (2x2 giants, 1x3 dragons, etc.)
- Grid collision detection
- AddUnitToSquad, RemoveUnitFromSquad, MoveUnitInSquad

**Testing Without Map:**
- Create squads programmatically with different formations
- Verify all units created with correct components
- Test multi-cell units occupy all specified cells
- Test collision detection prevents overlapping units
- Test unit management functions (add/remove/move)

### Success Criteria (Phase 1 Complete)

- [ ] Can create squads programmatically with 1-9 units
- [ ] Can execute squad-vs-squad combat without map interaction
- [ ] Row-based targeting works (front row protects back row)
- [ ] Role modifiers affect damage correctly
- [ ] Multi-cell units (2x2, 1x3, etc.) work in combat
- [ ] Abilities trigger automatically based on conditions
- [ ] Cooldowns prevent repeated ability firing
- [ ] Formation presets generate valid squads
- [ ] All unit tests pass
- [ ] Zero dependency on map movement or rendering

### Testing Strategy (Isolated from Map)

**Create test file: `systems/squad_integration_test.go`**

```go
func TestSquadCombatWithoutMap(t *testing.T) {
    // Setup ECS manager
    manager := ecs.NewManager()
    ecsmanager := setupTestECS(manager)

    // Create two squads (no map positions needed)
    playerSquad := CreateSquadFromTemplate(
        ecsmanager,
        "Player Squad",
        FORMATION_BALANCED,
        coords.LogicalPosition{X: 0, Y: 0}, // Dummy position
        createTestUnitTemplates(),
    )

    enemySquad := CreateSquadFromTemplate(
        ecsmanager,
        "Enemy Squad",
        FORMATION_DEFENSIVE,
        coords.LogicalPosition{X: 1, Y: 1}, // Dummy position
        createEnemyUnitTemplates(),
    )

    // Test combat without any map interaction
    result := ExecuteSquadAttack(playerSquad, enemySquad, ecsmanager)

    assert.Greater(t, result.TotalDamage, 0)
    assert.GreaterOrEqual(t, len(result.DamageByUnit), 1)

    // Test abilities trigger
    CheckAndTriggerAbilities(playerSquad, ecsmanager)

    // Validate results
}
```

**Key Insight:** All core squad mechanics can be tested with dummy positions, no map integration required.

---

## Phase 2: Squad Integration (After Core Testing Works)

**Time Estimate:** 16-24 hours (2-3 workdays)
**Dependencies:** Phase 1 complete and tested
**Risk Level:** High (touches multiple systems)

### Why After Phase 1?

Once squad building and combat are proven to work in isolation, we can confidently integrate with map movement, spawning, rendering, and input systems.

### Implementation Phases

#### Phase 2.1: Map Representation (4-6 hours)

**Deliverables:**
- Squad positioning on game map
- Squad movement as single unit
- Squad collision detection with map tiles

**Key Features:**
- Squads occupy single map tile (internal 3x3 grid is abstract)
- Moving squad moves all units together
- Map blocks squad movement, not individual units

**Testing:**
- Place squad on map at specific coordinates
- Move squad to new position
- Verify collision with walls and other squads

#### Phase 2.2: Input and UI Integration (4-6 hours)

**Deliverables:**
- Updated `input/combatcontroller.go` - Squad selection and targeting
- Click handling for squad combat initiation

**Key Features:**
- Click to select player squad
- Click enemy squad to initiate combat
- ESC to deselect
- Squad selection state management

**Testing:**
- Click squad entities on map
- Verify selection state changes
- Test combat initiation flow

#### Phase 2.3: Rendering Integration (4-6 hours)

**Deliverables:**
- `rendering/squadrenderer.go` - Squad grid visualization (200-250 lines)

**Key Features:**
- Draw 3x3 grid overlay on selected squad
- Show unit positions in grid cells
- Multi-cell units span multiple visual cells
- HP bars for each unit
- Role icons (Tank/DPS/Support)
- Row highlighting (front/mid/back)

**Testing:**
- Select squad, verify grid renders
- Verify multi-cell units render correctly
- Test HP bar updates after damage
- Verify grid disappears on deselect

#### Phase 2.4: Spawning Integration (4-6 hours)

**Deliverables:**
- Updated `spawning/spawnmonsters.go` - Enemy squad spawning

**Key Features:**
- SpawnEnemySquad function with level scaling
- Level 1-3: 3-5 weak units, no leader
- Level 4-7: 5-7 units with leader, 2 abilities
- Level 8+: 7-9 units with leader, 4 abilities, multi-cell bosses
- Squads spawn at valid map positions

**Testing:**
- Spawn squads at different levels
- Verify squad composition matches level
- Test multi-cell bosses spawn at high levels
- Verify squads appear on map correctly

### Success Criteria (Phase 2 Complete)

- [ ] Player can click to select squad on map
- [ ] Squad grid renders with unit positions visible
- [ ] Multi-cell units render across multiple grid cells
- [ ] Player can click enemy squad to attack
- [ ] Combat resolves with visual feedback
- [ ] Abilities trigger and show on-screen messages
- [ ] Dead squads removed from map
- [ ] Enemy squads spawn with level-appropriate composition
- [ ] Squad movement works on map
- [ ] HP bars update after combat

### Dependencies with Phase 1

**Required from Phase 1:**
- CreateSquadFromTemplate function
- ExecuteSquadAttack function
- CheckAndTriggerAbilities function
- GetUnitIDsInSquad query
- IsSquadDestroyed query

**Phase 2 adds:**
- Map position tracking
- Visual rendering
- User input handling
- Automated spawning

---

## Phase 3: Legacy System Refactoring (Can Happen in Parallel)

**Time Estimate:** 38-54 hours (5-7 workdays)
**Dependencies:** None for most items
**Can Parallelize:** With Phase 1 or Phase 2
**Risk Level:** Low-Medium

### Strategic Value

Migrating legacy code to proper ECS patterns improves performance, maintainability, and consistency with the new squad system architecture. These refactorings can happen in parallel with squad implementation.

### Refactoring Items

#### 3.1: Position System (8-12 hours) - HIGHEST IMPACT

**Status:** 0% → 100%
**Priority:** 1 (highest performance impact)
**Can Start:** Anytime

**Problem:**
- Current PositionTracker uses pointer keys: `map[*coords.LogicalPosition]*ecs.Entity`
- O(n) linear search for entity lookups
- 50x slower than necessary with 50+ entities

**Solution:**
- Create PositionSystem with entity ID-based spatial grid
- Hash-based lookup: `map[coords.LogicalPosition][]ecs.EntityID`
- O(1) position queries

**Impact:**
- 50+ monsters: O(50) → O(1) = 50x performance improvement
- Enables efficient squad positioning
- Required for performant multi-squad gameplay

**Testing:**
- Benchmark position lookups before/after
- Verify all combat position queries still work
- Test with 100+ entities to validate performance

#### 3.2: Weapon System (12-16 hours) - HIGH IMPACT

**Status:** 0% → 100%
**Priority:** 2 (most frequently called code)
**Can Start:** Anytime

**Problem:**
- Weapon components have logic methods (CalculateDamage, GetTargets)
- Returns entity pointers instead of IDs
- Anti-pattern: logic in components, not systems

**Solution:**
- Create WeaponSystem struct with all weapon logic
- Remove methods from MeleeWeapon and RangedWeapon components
- Return entity IDs from target queries

**Impact:**
- Aligns weapons with squad combat patterns
- Enables proper weapon integration with squads
- Template for other system migrations

**Testing:**
- Verify damage calculation matches old behavior
- Test with existing combat system
- Integrate with squad combat system

#### 3.3: Item System (10-14 hours) - MEDIUM IMPACT

**Status:** 0% → 100%
**Priority:** 3 (completes roadmap item)
**Can Start:** Anytime

**Problem:**
- Item.Properties is nested entity pointer
- Query logic in component methods
- Anti-pattern: nested entity hierarchy

**Solution:**
- Flatten Item structure with inline effect data
- Create ItemSystem for all item logic
- Use entity IDs for item references

**Impact:**
- Completes Status Effects roadmap item (85% → 100%)
- Removes nested entity anti-pattern
- Prepares for quality-based item spawning

**Testing:**
- Verify item creation still works
- Test effect application
- Validate quality system integration

#### 3.4: Movement System (8-12 hours) - MEDIUM IMPACT

**Status:** 0% → 100%
**Priority:** 4 (organizational improvement)
**Recommended After:** Position System complete

**Problem:**
- Creature.UpdatePosition method contains 30+ lines of logic
- Movement logic in component, not system
- Anti-pattern: behavior in data structure

**Solution:**
- Create MovementSystem struct
- Move all movement logic to system
- Remove methods from Creature component

**Impact:**
- Prepares for squad formations and movement
- Cleaner separation of concerns
- Benefits from Position System optimization

**Testing:**
- Verify creature pathfinding works
- Test with existing AI
- Validate position updates correct

### Success Criteria (Phase 3 Complete)

- [ ] Position System: O(1) position lookups, 50x faster
- [ ] Weapon System: All weapon logic in WeaponSystem, no component methods
- [ ] Item System: Flat item structure, no nested entities
- [ ] Movement System: All movement logic in MovementSystem
- [ ] All legacy code follows proper ECS patterns
- [ ] Zero component methods remaining (except pure helpers)
- [ ] Native entity IDs used throughout

### Parallelization Strategy

**Can work on Phase 3 while Phase 1 is in progress:**
- Different files, no conflicts
- Position System doesn't touch squad code
- Weapon System can be tested independently
- Only merges at Phase 2 (squad integration)

**Recommended parallel approach:**
- Week 1: Phase 0 (4h) + start Phase 1 (16h) + start Position System (8h)
- Week 2: Continue Phase 1 (20h) + complete Position System (4h) + start Weapon System (8h)
- Week 3: Complete Phase 1 (8h) + complete Weapon System (8h) + start Phase 2 (8h)

---

## Phase 4: Todos Implementation

**Time Estimate:** 8-12 hours (1-1.5 workdays)
**Dependencies:** Phase 1 complete (for spawning), Phase 2 complete (for squad features)
**Risk Level:** Low

### Items from todos.txt

#### 4.1: Bug Fixes (2-3 hours)

**Immediate fixes, no blockers:**

1. **Fix throwable AOE movement issue** (30 minutes)
   - Problem: AOE moves with player after throwing
   - Solution: Decouple visual effect from player position

2. **Ensure entities removed on death** (1 hour)
   - Problem: Dead entities not cleaned up
   - Solution: Add entity removal to death handling

3. **Don't allow shooting/throwing through walls** (1 hour)
   - Problem: Line-of-sight not checked
   - Solution: Add wall collision to targeting

**Testing:**
- Throw item, move player, verify AOE stays in place
- Kill entity, verify removed from ECS
- Attempt to shoot through wall, verify blocked

#### 4.2: Throwing Improvements (2-3 hours)

**Uses existing ItemAction system:**

1. **Make thrown items miss sometimes** (2-3 hours)
   - Add accuracy calculation based on distance
   - Add d20 roll for hit/miss
   - Show miss feedback to player

**Testing:**
- Throw at various distances
- Verify hit rate decreases with distance
- Test miss feedback appears

#### 4.3: Level Transitions (2-3 hours)

**Map management improvements:**

1. **Clear entities on level change** (1 hour)
   - Remove all entities except those near stairs
   - Clean up squad entities properly

2. **Add level variety** (1-2 hours)
   - Different tile types per level
   - Placeable items (chests, props)
   - Visual variety

**Testing:**
- Change levels, verify entities cleared
- Test stairs room entities persist
- Verify visual variety on different levels

#### 4.4: Spawning System (Unblocked by Entity Templates) (2-3 hours)

**Now ready to implement:**

1. **Probability-based entity spawning** (2-3 hours)
   - Use generic CreateEntityFromTemplate factory
   - Add probability tables by dungeon level
   - Weight spawns by difficulty rating

**Testing:**
- Spawn 100 entities, verify probability distribution
- Test different dungeon levels spawn appropriate entities
- Validate difficulty scaling

### Success Criteria (Phase 4 Complete)

- [ ] Throwable AOE doesn't move with player
- [ ] Dead entities removed from game
- [ ] Can't shoot through walls
- [ ] Thrown items have miss chance based on accuracy
- [ ] Level transitions clear entities correctly
- [ ] Levels have visual variety
- [ ] Spawning system uses probabilities
- [ ] Entity difficulty affects spawn rates

---

## Phase 5: Simplification Roadmap Completion

**Time Estimate:** 4 hours (included in Phase 0)
**Dependencies:** None
**Status:** Completed in Phase 0

### Items

1. **GUI Button Factory** - Completed in Phase 0
2. **Status Effects Quality** - Completed in Phase 0

### Final Roadmap Status

**After Phase 0 completion:**
1. ✅ Input System Consolidation - 100%
2. ✅ Coordinate System Standardization - 100%
3. ✅ Status Effects vs Item Behaviors - 100%
4. ✅ Entity Template System - 100%
5. ✅ Graphics Shape System - 95%
6. ✅ GUI Button Factory - 100%

**Overall Simplification Roadmap:** 99% (Graphics Shapes 95% pulls down average)

---

## Critical Path Analysis

### Sequential Requirements (Must Be Done in Order)

```
Phase 0 (4h)
    ↓
Phase 1.1: Components (6-8h)
    ↓
Phase 1.2: Query System (4-6h)
    ↓
Phase 1.3: Combat System (8-10h)
    ↓
Phase 1.4: Ability System (6-8h)
    ↓
Phase 1.5: Squad Creation (4-6h)
    ↓
[TESTING PHASE - Validate squad mechanics work]
    ↓
Phase 2.1: Map Representation (4-6h)
    ↓
Phase 2.2: Input Integration (4-6h)
    ↓
Phase 2.3: Rendering Integration (4-6h)
    ↓
Phase 2.4: Spawning Integration (4-6h)
    ↓
Phase 4: Todos Implementation (8-12h)
```

**Critical Path Total:** 60-92 hours

### Parallel Work Available

**Can be done anytime, in parallel with critical path:**

- Phase 3.1: Position System (8-12h)
- Phase 3.2: Weapon System (12-16h)
- Phase 3.3: Item System (10-14h)
- Phase 3.4: Movement System (8-12h)

**Parallel Work Total:** 38-54 hours

### Optimized Timeline

**Week 1 (40 hours):**
- Phase 0: GUI + Status Effects (4h)
- Phase 1.1-1.2: Components + Queries (10-14h)
- Phase 3.1: Position System (8-12h) [PARALLEL]
- Phase 1.3: Combat System start (8-10h)
- **Total:** 30-40h

**Week 2 (40 hours):**
- Phase 1.3: Combat System complete (remaining hours)
- Phase 1.4: Ability System (6-8h)
- Phase 1.5: Squad Creation (4-6h)
- Phase 3.2: Weapon System (12-16h) [PARALLEL]
- **Squad Testing Milestone**
- **Total:** 22-30h + testing

**Week 3 (40 hours):**
- Phase 2.1-2.4: Squad Integration (16-24h)
- Phase 3.3: Item System (10-14h) [PARALLEL]
- Phase 4: Todos Implementation (8-12h)
- **Total:** 34-50h

**Week 4 (optional, 26 hours):**
- Phase 3.4: Movement System (8-12h)
- Polish and bug fixes (10-14h)
- **Total:** 18-26h

**Fastest Completion:** 3 weeks (86 hours minimum)
**Realistic Completion:** 4 weeks (106-120 hours)
**Conservative Estimate:** 5 weeks (130-144 hours with polish)

---

## Testing Strategy

### Phase 1 Testing (Squad Core - No Map Required)

**Unit Tests:**
- Component registration and helper methods
- Query functions with various squad configurations
- Combat damage calculations and role modifiers
- Ability trigger conditions and effects
- Squad creation with multi-cell units

**Integration Tests:**
- Full combat flow between two squads
- Multiple abilities triggering in one turn
- Squad destruction and cleanup
- Formation presets generate valid layouts

**Key Advantage:** Can test all squad mechanics without map dependency

### Phase 2 Testing (Map Integration)

**Manual Tests:**
- Click to select squad on map
- Squad grid renders correctly
- Combat initiated via UI clicks
- Abilities show visual feedback
- Dead squads removed from map

**Integration Tests:**
- Full game flow with squad spawning
- Level transitions preserve squad state
- Multiple squads on map interact correctly

### Phase 3 Testing (Legacy Refactoring)

**Performance Tests:**
- Benchmark position lookups (50x improvement target)
- Measure combat execution time with new systems

**Regression Tests:**
- Verify old behavior maintained with new systems
- Test all combat scenarios still work
- Validate item and weapon functionality

### Phase 4 Testing (Todos)

**Bug Fix Verification:**
- Each bug has specific test case
- Regression tests for fixed issues

**Feature Validation:**
- Throwing accuracy measured over 100 throws
- Spawning probabilities validated over 1000 spawns

---

## Risk Mitigation

### High-Risk Areas

#### 1. Squad Combat Integration (Phase 2.2)
**Risk:** Breaking existing combat system
**Mitigation:**
- Keep 1v1 combat code intact initially
- Add squad combat as separate code path
- Feature flag to toggle between systems
- Gradual migration, not replacement

#### 2. Multi-Cell Unit Rendering (Phase 2.3)
**Risk:** Complex visual layout
**Mitigation:**
- Start with simple bounding box
- Test with single multi-cell unit first
- Iterate on polish after functionality works

#### 3. Performance with Multiple Squads (Phase 2)
**Risk:** Query overhead with many squads
**Mitigation:**
- Cache query results where possible
- Profile with 10+ squads
- Optimize queries if needed
- Position System refactoring provides 50x boost

### Medium-Risk Areas

#### 4. Weapon System Migration (Phase 3.2)
**Risk:** Changing frequently-called code
**Mitigation:**
- Comprehensive tests before migration
- Keep old methods temporarily
- Validate damage calculations match exactly

#### 5. Ability Trigger Timing (Phase 1.4)
**Risk:** Abilities fire at wrong time
**Mitigation:**
- HasTriggered flag for one-time abilities
- Thorough cooldown testing
- Debug logging for triggers

### Rollback Plans

- Each phase is independent - can rollback individually
- Feature branches for major changes
- Keep old code as deprecated wrappers during migration
- Ability to toggle squad combat on/off

---

## Success Metrics

### Phase 0 Success
- [ ] Simplification roadmap at 99%
- [ ] GUI buttons consolidated
- [ ] Status effects decoupled from quality

### Phase 1 Success (Squad Core)
- [ ] Can create squads programmatically
- [ ] Squad combat works without map
- [ ] Row targeting functions correctly
- [ ] Multi-cell units work in combat
- [ ] Abilities trigger automatically
- [ ] All unit tests pass

### Phase 2 Success (Integration)
- [ ] Player can control squads on map
- [ ] Combat initiated via UI
- [ ] Squad grid renders correctly
- [ ] Enemy squads spawn at levels
- [ ] Visual feedback for all actions

### Phase 3 Success (Refactoring)
- [ ] 50x performance improvement (Position System)
- [ ] Zero component methods
- [ ] 100% proper ECS patterns
- [ ] All legacy code migrated

### Phase 4 Success (Todos)
- [ ] All bugs fixed
- [ ] Throwing accuracy implemented
- [ ] Level variety added
- [ ] Spawning system operational

### Overall System Success
- [ ] Command multiple squads tactical gameplay
- [ ] Squad building with formations
- [ ] Automated leader abilities
- [ ] Multi-cell units add variety
- [ ] Stable performance with 10+ squads
- [ ] 100% ECS pattern compliance
- [ ] All tests passing

---

## Next Steps

### Immediate Actions (This Week)

1. **Start Phase 0** (4 hours)
   - Complete GUI Button Factory
   - Complete Status Effects Quality Interface
   - Achieve 99% simplification roadmap

2. **Begin Phase 1.1** (6-8 hours)
   - Create squad package structure
   - Define 8 component types
   - Register components with ECS
   - **Milestone:** Game builds with squad components

3. **Optional Parallel Work**
   - Start Position System refactoring (8-12h)
   - Can work alongside Phase 1.1

### First Major Milestone (Week 2)

**Squad Testing Checkpoint:**
- Phase 1 complete (all 5 sub-phases)
- Can create squads programmatically
- Can execute squad-vs-squad combat
- All unit tests passing
- **VALIDATION:** Squad mechanics work before map integration

### Second Major Milestone (Week 3)

**Map Integration Complete:**
- Phase 2 complete (all 4 sub-phases)
- Squads visible and controllable on map
- Enemy squads spawn automatically
- Combat works through UI
- **VALIDATION:** Full game loop with squads

### Final Milestone (Week 4-5)

**Production Ready:**
- Phase 3 complete (legacy code migrated)
- Phase 4 complete (todos implemented)
- All tests passing
- Performance validated
- **VALIDATION:** Ready for gameplay testing

---

## Resource Requirements

### Development Tools
- Go 1.21+ compiler
- Git for version control
- Testing framework (testify recommended)
- Profiling tools for performance validation

### Documentation
- All source files in analysis/ folder
- CLAUDE.md for project instructions
- This master roadmap

### External Libraries (Already Present)
- github.com/bytearena/ecs - ECS framework
- github.com/hajimehoshi/ebiten/v2 - Game engine
- Internal packages (common, coords, entitytemplates, etc.)

---

## Appendix: Key Design Decisions

### Why Squad Testing Before Map Integration?

**User Requirement:** Test squad building and combat in isolation first.

**Benefits:**
- Validate core mechanics without rendering complexity
- Faster iteration on combat balance
- Easier debugging (fewer moving parts)
- Can change squad mechanics without touching map code

**Approach:**
- Phase 1 creates squads with dummy positions
- Combat tested via unit tests and programmatic execution
- Map integration deferred to Phase 2

### Why Multi-Cell Units?

**Strategic Value:**
- Boss variety (2x2 giants, 3x3 dragons)
- Tactical positioning (large units control more rows)
- Visual distinctiveness (big units look impressive)

**Implementation:**
- GridPositionData has Width/Height fields
- Units can be 1x1, 1x2, 2x2, 1x3, up to 3x3
- Query system handles multi-cell automatically

### Why Automated Abilities?

**Design Goal:** FFT-style abilities that don't interrupt flow.

**Benefits:**
- No manual ability selection interrupting combat
- Strategic depth from ability composition
- Emergent gameplay from condition combinations

**Implementation:**
- 4 ability slots per leader
- Each slot has trigger condition
- Abilities fire automatically when condition met

### Why Entity IDs Instead of Pointers?

**ECS Best Practice:** Native entity IDs prevent coupling.

**Benefits:**
- No circular references
- Query-based relationships
- Entities can be safely removed
- Proper ECS architecture

**Implementation:**
- All references use ecs.EntityID (uint32)
- Query functions return []ecs.EntityID
- No entity pointers in components

---

## Document Change History

- **v1.0 (2025-10-02):** Initial master roadmap created
  - Synthesized 5 source documents
  - Defined 5 implementation phases
  - Established testing-first approach for squad system
  - Total timeline: 106-144 hours (13-18 workdays)

---

**End of Master Roadmap**
