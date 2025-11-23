# TinkerRogue Master Development Roadmap

**Version:** 5.1 - REMAINING WORK ONLY | **Updated:** 2025-11-23
**Status:** Focus on completing Squad System rendering/spawning, GUI cleanup, and bug fixes

---

## What's Left To Do

**Total Remaining:** 9-15 hours across all priorities

### High Priority (Core Functionality - 6-8h)
1. **Squad Graphical Rendering** (2-3h)
   - Integrate squad sprites with existing GUI modes
   - HP bars, role icons, row highlighting
   - Leverages completed GUI mode infrastructure

2. **Enemy Squad Spawning** (1-2h)
   - Create SpawnEnemySquad wrapper function
   - Hook up to level generation system
   - Use completed formation presets

3. **Critical Bug Fixes** (2-3h)
   - Fix throwable AOE movement issue
   - Ensure entities removed on death
   - Block shooting/throwing through walls

### Medium Priority (Architecture - 2-4h)
4. **GUI Architectural Cleanup** (2-4h)
   - Split `guicomponents/` into separate concerns (UI vs queries)
   - Relocate BaseMode from root package to gui/core
   - Refactor global state in `guiresources/` (12+ globals need dependency injection)

### Low Priority (Polish - 2-4h)
5. **Game Polish** (1-2h)
   - Throwing accuracy/miss chance system
   - Level transitions cleanup
   - Visual variety (tile types, diversity)

6. **Status Effects Quality** (1-2h, optional)
   - Interface extraction for quality system

---

## Detailed Task Breakdown

### Task 1: Squad Graphical Rendering (2-3h)

**Current State:**
- Text-based visualization works (visualization.go)
- GUI mode system exists (squadmanagementmode.go, combatmode.go)
- Squad data fully accessible via query functions

**What's Needed:**
- Create sprite rendering for 3x3 squad grid
- HP bars for each unit (color-coded by health percentage)
- Role icons (Tank/DPS/Support visual indicators)
- Row highlighting during targeting
- Multi-cell unit visual representation

**Files to Modify:**
- `gui/guicombat/combatmode.go` - Add squad rendering to render loop
- `gui/guisquads/squadmanagementmode.go` - Add visual squad display
- Potentially create `gui/guicombat/squad_renderer.go` for reusable rendering logic

**Dependencies:**
- None - all squad data systems complete

---

### Task 2: Enemy Squad Spawning (1-2h)

**Current State:**
- `CreateSquadFromTemplate()` function exists and works
- Entity template system operational
- Formation presets complete (4 types available)

**What's Needed:**
- Create `SpawnEnemySquad(level, factionID, position)` wrapper function
- Level scaling logic for squad composition
- Hook into level generation/room spawning system
- Position assignment on map

**Files to Modify:**
- `squads/squadcreation.go` - Add SpawnEnemySquad function
- `spawning/` package - Hook squad spawning into level generation
- `worldmap/` package - Place squads in rooms/areas

**Dependencies:**
- None - all underlying systems complete

---

### Task 3: Critical Bug Fixes (2-3h)

**Bug 1: Throwable AOE Movement Issue**
- Description: Need details on what the issue is
- Estimated: 1h

**Bug 2: Entity Cleanup on Death**
- Description: Entities not being removed from ECS on death
- Files: Check entity removal in combat/health system
- Estimated: 30min

**Bug 3: Wall Collision for Shooting/Throwing**
- Description: Projectiles/throws ignore walls
- Files: Check line-of-sight calculation in combat system
- Estimated: 1-1.5h

---

### Task 4: GUI Architectural Cleanup (2-4h)

**Issue 1: Split guicomponents/ (2-3h)**
- Current: Mixed responsibilities (884 LOC total)
- Target: Separate into `gui/components/` (UI widgets) and separate ECS query functions
- Note: `gui/queries/` directory exists but is empty

**Issue 2: Relocate BaseMode (1h)**
- Current: `gui/basemode.go` in root package
- Target: Move to `gui/core/basemode.go` or `gui/base/basemode.go`
- Update imports in all mode files

**Issue 3: Refactor guiresources/ Globals (1-2h)**
- Current: 12+ global variables
- Target: Dependency injection pattern
- Files: `gui/guiresources/guiresources.go`

---

### Task 5: Game Polish (1-2h)

**Feature 1: Throwing Accuracy/Miss Chance**
- Add accuracy calculation based on distance
- Miss chance for throwing weapons
- Visual feedback for misses

**Feature 2: Level Transitions**
- Clean up level transition code
- Smooth state transitions between levels

**Feature 3: Visual Variety**
- More tile types
- Visual diversity in dungeons
- Different tile graphics

---

### Task 6: Status Effects Quality (1-2h, Optional)

**Current State:**
- StatusEffects interface exists (stateffect.go, 381 LOC)
- 3 effects implemented: Burning, Freezing, Sticky

**What's Needed:**
- Extract quality/cleanliness interface
- Better separation of effect logic

**Priority:** LOW - Not blocking any other work

---

## Implementation Order

### Phase 1: Core Functionality (6-8h)
Complete in this exact order:
1. Enemy Squad Spawning (1-2h) - Unblocks playtesting
2. Squad Graphical Rendering (2-3h) - Makes squads visible
3. Critical Bug Fixes (2-3h) - Makes game stable

### Phase 2: Architecture (2-4h)
After Phase 1 complete:
4. GUI Architectural Cleanup (2-4h) - Improves maintainability

### Phase 3: Polish (2-4h)
After Phase 2 complete:
5. Game Polish (1-2h)
6. Status Effects Quality (1-2h, optional)

---

## Success Criteria

### Phase 1 Complete When:
- ✅ Enemy squads spawn in levels
- ✅ Squad grid renders graphically (not just text)
- ✅ HP bars and role icons visible
- ✅ Entities removed on death
- ✅ Projectiles blocked by walls
- ✅ AOE throwable movement works

### Phase 2 Complete When:
- ✅ guicomponents split into separate concerns
- ✅ BaseMode relocated to proper package
- ✅ No global variables in guiresources

### Phase 3 Complete When:
- ✅ Throwing has accuracy/miss mechanics
- ✅ Level transitions work smoothly
- ✅ Visual variety added to dungeons

---

## Notes

- All core systems (squad combat, abilities, formations, worldmap generation, ECS) are complete
- GUI mode infrastructure is complete
- Focus is on final integration, rendering, and polish
- No major architectural changes needed

---


todos   The squad builder still uses the global unit templates for now. Integrating the roster with the squad builder is a larger refactoring task that can be done separately when needed.
**End of Roadmap**
