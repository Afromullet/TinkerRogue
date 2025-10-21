# TinkerRogue Master Development Roadmap

**Version:** 3.0 | **Updated:** 2025-10-12
**Status:** Squad System 85% Complete, 16-24 hours remaining (2-3 workdays)

---

## Current State Summary

### What's Complete ✅
- **Position System** (399 LOC) - O(1) spatial grid, 50x performance improvement
- **Squad Components** (300 LOC) - All 8 ECS components defined
- **Query System** (140 LOC) - All 7 query functions operational
- **Combat System** (406 LOC) - Full squad combat with hit/dodge/crit/cover mechanics
- **Visualization** (175 LOC) - Text-based 3x3 grid rendering
- **Testing Infrastructure** (1000+ LOC) - Comprehensive test suite exists

### What's Remaining ❌
- **Ability System** (0 LOC) - Auto-triggering leader abilities → 8-10 hours
- **Formation Presets** (85 LOC partial) - Balanced/Defensive/Offensive templates → 4-6 hours
- **Map Integration** - Squad positioning, movement, spawning → 4-6 hours
- **Rendering Integration** - Graphical sprites instead of text → 2-3 hours
- **Todo Items** - Bug fixes, throwing accuracy, level variety → 2-4 hours

### Systems Removed (No Longer in Codebase)
- Creature components (replaced by squad units)
- Weapon components (replaced by attributes system)
- Tracker system (replaced by Position System)
- Individual 1v1 combat (replaced by squad combat)

---

## Time Estimates

**Total Remaining:** 18-28 hours
**Conservative Estimate:** 2-3 workdays (8-10 hour days)
**Reduction from v2.0:** 54-88 hours saved

### Breakdown
- Phase 1.3 (Abilities): 8-10h ❌
- Phase 1.4 (Formations): 4-6h ⚠️
- Phase 1.5 (Testing): 0-2h ✅
- Phase 2 (Map Integration): 8-12h ❌
- Phase 4 (Todos): 2-4h ❌

---

## Phase 1: Squad System Core (85% → 100%)

### 1.1 Query System ✅ COMPLETE
**File:** squads/squadqueries.go (140 LOC)
All 7 functions implemented: FindUnitByID, GetUnitIDsAtGridPosition, GetUnitIDsInSquad, GetSquadEntity, GetUnitIDsInRow, GetLeaderID, IsSquadDestroyed

### 1.2 Combat System ✅ COMPLETE
**File:** squads/squadcombat.go (406 LOC)
**Features:** ExecuteSquadAttack, row-based targeting, cell-based patterns, hit/dodge/crit mechanics, cover system, multi-cell unit support

### 1.3 Ability System ❌ NOT STARTED (8-10h)
**File:** squads/abilities.go - DOES NOT EXIST
**Deliverables:**
- [ ] CheckAndTriggerAbilities(squadID, manager)
- [ ] Trigger conditions: HP threshold, turn count, combat start, enemy count, morale
- [ ] Ability effects: Rally (+5 damage), Heal (10 HP), Battle Cry (+3 dmg, +10 morale), Fireball (15 damage)
- [ ] Cooldown management: CooldownRemaining, HasTriggered flag
- [ ] Integration with combat flow

### 1.4 Formation System ⚠️ 40% COMPLETE (4-6h)
**File:** squads/squadcreation.go (85 LOC partial)
**Deliverables:**
- [x] CreateEmptySquad() - works
- [ ] Fix AddUnitToSquad() - currently only validates, doesn't create entity
- [ ] CreateSquadFromTemplate(manager, formation, unitTemplates)
- [ ] Formation presets: Balanced (2/4/3), Defensive (4/3/2), Offensive (1/6/2), Ranged (1/2/6)
- [ ] Grid collision detection (prevent overlapping units)
- [ ] RemoveUnitFromSquad(), MoveUnitInSquad()

### 1.5 Testing ✅ INFRASTRUCTURE EXISTS (0-2h)
**File:** squads/squads_test.go (1000+ LOC)
Test infrastructure exists, needs additional ability/formation tests when complete.

---

## Phase 2: Map Integration (4-6h)

### 2.1 Map & Input (4-6h)
- [ ] Squad positioning on game map
- [ ] Squad movement as single unit
- [ ] Click handling for squad selection/targeting
- [ ] Squad collision with map tiles

### 2.2 Rendering (2-3h)
- [x] Text-based visualization exists (visualization.go)
- [ ] Graphical rendering (sprites, HP bars, role icons, row highlighting)

### 2.3 Spawning (2-3h)
- [ ] SpawnEnemySquad function with level scaling
- [ ] Level 1-3: 3-5 weak units, no leader
- [ ] Level 4-7: 5-7 units with leader, 2 abilities
- [ ] Level 8+: 7-9 units with leader, 4 abilities, multi-cell bosses

---

## Phase 4: Todo Items (2-4h)

### Bug Fixes (1-2h)
- [ ] Fix throwable AOE movement issue
- [ ] Ensure entities removed on death
- [ ] Don't allow shooting/throwing through walls

### Features (1-2h)
- [ ] Throwing accuracy/miss chance
- [ ] Level transitions cleanup
- [ ] Add level variety (tile types, visual diversity)

---

## Critical Path (Sequential)

```
Phase 1.3: Ability System (8-10h)
    ↓
Phase 1.4: Formation System (4-6h)
    ↓
[TESTING - Validate abilities work]
    ↓
Phase 2: Map + Input + Rendering + Spawning (8-12h)
    ↓
Phase 4: Todos (2-4h)
```

**Total Critical Path:** 20-32 hours

---

## Timeline

### Week 1 (16-24h)
- **Day 1-2:** Ability System (8-10h) - CheckAndTriggerAbilities, trigger conditions, effects, cooldowns
- **Day 2-3:** Formation System (4-6h) - Fix AddUnitToSquad, formation presets, collision detection
- **Day 3:** Additional Testing (0-2h) - Ability/formation tests
- **Day 3-4:** Map + Input Integration (4-6h) - Squad positioning, movement, click handling

### Week 2 (6-10h)
- **Day 1:** Rendering (2-3h) - Graphical sprites, HP bars, role icons
- **Day 1-2:** Spawning (2-3h) - Enemy squad spawning with level scaling
- **Day 2:** Todos (2-4h) - Bug fixes, throwing accuracy, level variety

**Fastest Completion:** 2 workdays (16h minimum)
**Realistic:** 3 workdays (24h)
**Conservative:** 4 workdays (32h with buffer)

---

## Next Steps (Priority Order)

### Immediate (This Week)
1. **Create squads/abilities.go** - Implement CheckAndTriggerAbilities with all trigger conditions and effects
2. **Fix squads/squadcreation.go** - Complete AddUnitToSquad and CreateSquadFromTemplate
3. **Create squads/formations.go** - Formation presets and collision detection
4. **Add ability tests** - Validate triggers, cooldowns, effects

### First Milestone (End of Week 1)
- Squad System 100% complete
- All abilities trigger correctly
- Formation presets operational
- All unit tests passing

### Second Milestone (Week 2)
- Map integration complete
- Squads controllable via UI
- Enemy squads spawn automatically
- Full game loop with squads

---

## ECS Best Practices (Reference)

**Based on squad & inventory systems:**
1. Pure data components - Zero logic methods, only data fields
2. Native EntityID - Use `ecs.EntityID` everywhere, not pointers
3. Query-based relationships - Discover via ECS queries, don't store references
4. System-based logic - All behavior in systems, not component methods
5. Value map keys - Use value-based keys for O(1) performance

**Reference Implementations:**
- `squads/*.go` - 2358 LOC, 8 components, 7 query functions, system-based combat
- `gear/Inventory.go` - 241 LOC, pure data component, 9 system functions
- `gear/items.go` - 177 LOC, EntityID-based relationships

---

## Risk Mitigation

**Low Risk Areas (Already Complete):**
- Core combat system operational (tested)
- Query system complete
- Position system optimized (50x performance)

**Medium Risk Areas:**
1. **Ability Trigger Timing** - Use HasTriggered flag, thorough cooldown testing, debug logging
2. **Formation Collision** - Test 2x2/1x3 units explicitly, validate presets, debug visualization

---

## Success Metrics

### Phase 1 Complete
- [x] Can create squads programmatically
- [x] Squad combat works without map
- [x] Row targeting, multi-cell units work
- [ ] Abilities trigger automatically ← REMAINING
- [ ] Formation presets generate valid squads ← REMAINING
- [x] All unit tests pass

### Phase 2 Complete
- [ ] Player can control squads on map
- [ ] Combat initiated via UI
- [ ] Squad grid renders graphically
- [ ] Enemy squads spawn at levels

### Overall System Success
- [ ] Command multiple squads tactical gameplay
- [ ] Squad building with formations
- [x] Multi-cell units add variety
- [x] 100% ECS pattern compliance
- [x] Stable performance with 10+ squads

---

**End of Roadmap**
