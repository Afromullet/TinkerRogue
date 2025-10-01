# Refactoring Priorities Before Implementing Todos

**Analysis Date:** 2025-10-01
**Current Roadmap Completion:** 73%
**Target:** Determine what must be refactored before implementing todos.txt items

---

## Executive Summary

### Critical Findings

1. **Entity Template System (50% → 100%) BLOCKS Spawning System** ⚠️
   - Current spawning uses hardcoded MonsterTemplates[0] or random selection
   - Generic factory needed for probability-based spawning with difficulty ratings
   - **Must complete before implementing spawning todo (line 31)**

2. **Squad Combat System Requires Major Refactoring** ⚠️
   - Current 1v1 combat system incompatible with "command several squads" requirement (line 25)
   - 40-60 hours effort for full implementation
   - Partially blocks AI, spawning quality, and balance features
   - Can be implemented incrementally alongside other work

3. **Quick Wins Available** ✅
   - GUI Button Factory: 2 hours to complete
   - Status Effects Quality: 2 hours to complete
   - Neither blocks todos, but improves maintainability

---

## Priority Order (By Blocking Impact)

### PRIORITY 1: Complete Entity Template System (4 hours) 🔴 CRITICAL

**Status:** 50% → 100%
**Blocks:** Spawning system implementation
**File:** `entitytemplates/creators.go`

**Why Critical:**
- Todo line 31: "Develop a spawning system... Use probabilities"
- Current system has 4 specialized `CreateXFromTemplate()` functions
- Generic factory enables data-driven, probability-based entity creation
- Foundation already exists (ComponentAdder pattern complete)

**Implementation:**
```go
// Consolidate 4 functions into generic factory
CreateEntityFromTemplate(manager, EntityConfig{
    Type: EntityCreature,
    Name: "Goblin",
    // ... other config
}, monsterData)
```

**Effort:** ~60 LOC changes, 4 hours
**Risk:** Low-Medium (well-tested pattern)
**Impact:** Unblocks entire spawning system

---

### PRIORITY 2: Decide Squad Combat Approach (Planning: 2-4 hours) 🟡 STRATEGIC

**Status:** 0% → Planning Phase
**Blocks:** AI system (high), balance system (high), spawning quality (medium)
**Files:** `combat/attackingsystem.go` (needs major expansion)

**Why Strategic:**
- Todo line 25: "Create a combat system that allows squad building like Nephilim or Soul Nomad"
- Current 1v1 system incompatible with multi-squad command structure
- Impacts how entities spawn (individual vs squads) and behave (AI)

**Two Approaches:**

**Option A: Incremental Migration (Recommended)**
- Phase 1: PlayerSquad wrapper (backward compatible) - 12-16 hours
- Phase 2: Multi-squad support - 16-20 hours
- Phase 3: Squad-aware spawning & AI - 12-16 hours
- **Total:** 40-52 hours, can parallelize with other features

**Option B: Defer Squad Combat**
- Implement spawning/throwing/levels with current 1v1 system
- Refactor to squads later (more rework, but faster short-term progress)

**Recommendation:** Start Phase 1 (PlayerSquad wrapper) immediately after Entity Templates
**Why:** Provides foundation without breaking existing systems, enables squad-aware spawning

---

### PRIORITY 3: GUI Button Factory (2 hours) 🟢 QUICK WIN

**Status:** 10% → 100%
**Blocks:** Nothing
**File:** `gui/playerUI.go`

**Why Quick Win:**
- 3 duplicate button functions → 1 factory
- Net -35 lines (23% reduction)
- Demonstrates progress, builds momentum
- Zero dependencies on other refactorings

**Implementation:**
```go
CreateMenuButton(playerUI, ui, ButtonConfig{
    Label: "Throwables",
    WindowGetter: func(p *PlayerUI) WindowDisplay {
        return p.ItemsUI.ThrowableItemDisplay
    },
})
```

**Effort:** ~30 LOC changes, 2 hours
**Risk:** Minimal (pure structural change)
**Impact:** Maintainability improvement

---

### PRIORITY 4: Status Effects Quality Interface (2 hours) 🟢 CLEANUP

**Status:** 85% → 100%
**Blocks:** Nothing
**Files:** `gear/stateffect.go`, `gear/itemactions.go`

**Why Cleanup:**
- Completes conceptual separation (effects ≠ quality)
- Quality belongs at item entity level, not behavior level
- Helps spawning system by clarifying quality management

**Implementation:**
- Remove `common.Quality` embedding from StatusEffects and ItemAction interfaces
- Move quality management to item entity level
- Update spawning to set quality on items, not effects

**Effort:** ~25 LOC changes, 2 hours
**Risk:** Minimal (additive change)
**Impact:** Architectural clarity

---

## Recommended Implementation Timeline

### Week 1: Foundation (8 hours)

**Day 1-2: Complete Roadmap Items**
- [ ] GUI Button Factory (2 hours) - Quick confidence builder
- [ ] Entity Template System (4 hours) - **Unblocks spawning**
- [ ] Status Effects Quality (2 hours) - Completes roadmap

**Result:** 100% roadmap completion, spawning system unblocked

### Week 2-4: Squad Combat Foundation (40 hours)

**Phase 1: Non-Breaking Squad Infrastructure (12 hours)**
- [ ] Create Squad, SquadManager, Formation types
- [ ] Implement PlayerSquad wrapper around PlayerData
- [ ] Add PerformAttackV2 with Allegiance parameter
- [ ] Test single-squad combat (backward compatible)

**Phase 2: Multi-Squad Support (16 hours)**
- [ ] Turn manager and initiative system
- [ ] Formation positioning and movement
- [ ] Squad vs squad combat testing

**Phase 3: Integration (12 hours)**
- [ ] Update spawning for squad-aware entity creation
- [ ] Basic squad AI
- [ ] Integration testing

**Result:** Squad combat foundation ready for todos implementation

### Week 5+: Feature Implementation

With refactoring complete, implement todos:
- ✅ **Spawning system** (line 31) - No longer blocked
- ✅ **Squad combat features** (line 25) - Foundation ready
- ✅ **Throwing improvements** (line 36) - Not blocked
- ✅ **Level transitions** (line 42) - Not blocked
- ✅ **Bug fixes** (lines 4, 6, 8) - Not blocked

---

## Dependency Analysis

### Critical Path (Sequential)
```
Entity Template System (4h)
    ↓
Spawning System Implementation (todos.txt:31)
    ↓
Balance/Difficulty System (todos.txt:13)
```

### Squad Combat Path (Sequential)
```
Squad Foundation Phase 1 (12h)
    ↓
Squad Foundation Phase 2 (16h)
    ↓
Squad-Aware AI & Spawning (12h)
    ↓
Full Squad Combat Features (todos.txt:25)
```

### Independent Work (Can Parallelize)
- GUI Button Factory (2h) - Anytime
- Status Effects Quality (2h) - Anytime
- Throwing improvements (todos.txt:36) - Anytime
- Level transitions (todos.txt:42) - Anytime
- Bug fixes (todos.txt:4,6,8) - Anytime

---

## What Can Be Implemented NOW

### ✅ Immediately Available (No Blockers)
1. **Bug Fixes** (todos.txt lines 4, 6, 8)
   - Fix throwable AOE movement
   - Fix entity removal on death
   - Don't allow shooting through walls

2. **Throwing Improvements** (todos.txt line 36)
   - Make thrown items miss sometimes
   - Uses existing ItemAction system (85% complete)

3. **Level Transitions** (todos.txt line 42)
   - Clear entities on level change
   - Add level variety

### ⏳ Blocked Until Refactoring Complete
1. **Spawning System** (todos.txt line 31)
   - **BLOCKED BY:** Entity Template System (50% complete)
   - **Time to unblock:** 4 hours
   - **Then ready for:** Probability-based spawning implementation

2. **Squad Combat** (todos.txt line 25)
   - **BLOCKED BY:** Squad system doesn't exist
   - **Time to unblock:** 12 hours (Phase 1), 40 hours (full system)
   - **Can start:** Incremental Phase 1 after Entity Templates

3. **Balance/Difficulty** (todos.txt line 13)
   - **BLOCKED BY:** Squad combat system (high impact)
   - **Workaround:** Can implement basic difficulty for individual entities
   - **Full system:** Needs squad combat for accurate power ratings

---

## Risk Assessment

### High Risk Items
1. **Squad Combat Refactoring** - CRITICAL complexity
   - Mitigation: Incremental migration with backward compatibility
   - Feature flag: `SQUAD_COMBAT_ENABLED` for gradual rollout
   - Preserve `PerformAttack()` logic (don't rewrite proven math)

### Medium Risk Items
2. **Entity Template System** - LOW-MEDIUM
   - Mitigation: Keep old functions as wrappers during migration
   - Type safety: Add validation for `data any` type assertions
   - Testing: Verify all entity creation paths after changes

### Low Risk Items
3. **GUI Button Factory** - MINIMAL
4. **Status Effects Quality** - MINIMAL

---

## Success Metrics

### Short-Term (Week 1)
- ✅ Roadmap at 100% completion
- ✅ Spawning system unblocked
- ✅ -20 net lines of code from cleanup

### Medium-Term (Weeks 2-4)
- ✅ Squad combat foundation complete
- ✅ Backward compatible migration (existing gameplay works)
- ✅ Ready for squad-aware spawning

### Long-Term (Week 5+)
- ✅ All non-squad todos implemented (bugs, throwing, levels)
- ✅ Spawning system with probabilities operational
- ✅ Squad combat features being added incrementally

---

## Conclusion

### Critical Actions Before Starting Todos

1. **MUST DO:** Complete Entity Template System (4 hours)
   - Unblocks spawning system (key todo line 31)
   - Enables probability-based entity generation
   - Foundation for balanced spawning

2. **SHOULD DO:** Begin Squad Combat Phase 1 (12 hours)
   - Provides foundation for "command several squads" (todo line 25)
   - Can run parallel to other feature work
   - Incremental approach reduces risk

3. **NICE TO HAVE:** Complete GUI Buttons + Status Effects (4 hours)
   - Achieves 100% roadmap completion
   - Demonstrates progress
   - Improves maintainability

### Recommended Starting Point

**Start with Priority 1 (Entity Templates)** - This unlocks the most value in shortest time.

**Total time to unblock all todos:** 8 hours (roadmap completion) + 12 hours (squad Phase 1) = 20 hours

After 20 hours of focused refactoring, all todos can be implemented without architectural blockers.
