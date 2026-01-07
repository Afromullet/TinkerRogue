# guicombat Package Refactoring Recommendations

**Analysis Date:** 2026-01-06
**Package:** `game_main/gui/guicombat`
**Files Analyzed:** 6 files, ~2,466 total lines

---

## Executive Summary

The `guicombat` package suffers from three primary architectural issues:

1. **God Object Pattern** - `CombatMode` has 74 struct fields and 1,061 lines handling UI, combat flow, AI orchestration, lifecycle, and visualization
2. **Complex AI State Machine** - AI turn execution spans 5 tightly-coupled methods with recursive calls and callback chains
3. **Code Duplication** - `SquadCombatRenderer` duplicates 80+ lines of grid positioning and rendering logic

These issues make the code difficult to understand, test, and maintain. Below are three refactoring recommendations prioritized by impact.

---

## Recommendation 1: Extract AI Turn Orchestration (HIGH PRIORITY)

### Problem

The AI turn execution logic in `combatmode.go` forms a complex state machine spread across 5 methods:

```
handleEndTurn()
  → executeAITurnIfNeeded()
    → playAIAttackAnimations()
      → playNextAIAttack()
        → advanceAfterAITurn()
          → executeAITurnIfNeeded() [RECURSIVE]
```

**Issues:**
- **Recursive execution**: `advanceAfterAITurn()` calls `executeAITurnIfNeeded()`, creating a hard-to-debug recursion loop
- **Callback hell**: Attack animations are chained through callbacks (`SetOnComplete()`)
- **Scattered state**: Animation queue, auto-play mode, and turn state are managed across multiple locations
- **Hard to test**: Can't test AI turn logic without full combat mode setup
- **Error-prone**: Missing callback or wrong transition breaks the entire AI turn flow

**Code References:**
- `combatmode.go:472-612` - 140 lines of AI turn orchestration
- `combatmode.go:508-568` - Attack animation chaining with callbacks

### Solution: Create `AITurnOrchestrator`

Extract AI turn logic into a dedicated orchestrator that manages the state machine explicitly:

```go
// tactical/ai/aiturnorchestrator.go
type AITurnOrchestrator struct {
    combatService  *combatservices.CombatService
    modeManager    *core.UIModeManager
    logManager     *CombatLogManager
    queries        *guicomponents.GUIQueries

    // State for current AI turn
    currentFactionID ecs.EntityID
    attackQueue      []combatservices.QueuedAttack
    currentAttackIdx int
}

// Public API - clear, testable methods
func (orch *AITurnOrchestrator) ShouldExecuteAITurn(factionID ecs.EntityID) bool
func (orch *AITurnOrchestrator) ExecuteAITurn(factionID ecs.EntityID) error
func (orch *AITurnOrchestrator) OnAnimationComplete()
func (orch *AITurnOrchestrator) AdvanceToNextTurn()
```

**Benefits:**
- **Single Responsibility**: Only handles AI turn execution flow
- **Testable**: Can test AI logic without UI or mode manager
- **Explicit State**: No hidden recursion, clear state transitions
- **Reusable**: Could support AI vs AI battles or replay systems

**Effort:** Medium (4-6 hours)
- Extract 5 methods (~140 lines)
- Add tests for AI turn state machine
- Update CombatMode to delegate to orchestrator

---

## Recommendation 2: Extract Combat Lifecycle Manager (MEDIUM PRIORITY)

### Problem

Combat initialization and cleanup logic is scattered across `CombatMode`:

**Initialization Spread:**
- `SetupEncounter()` (combatmode.go:614-653) - Spawns combat entities
- `initialzieCombatFactions()` (combatmode.go:780-806) - Initializes factions [NOTE: typo in name]
- `Enter()` (combatmode.go:655-692) - Orchestrates setup
- Multiple build panel methods setup UI state

**Cleanup Spread:**
- `cleanupCombatEntities()` (combatmode.go:841-922) - 82 lines of entity removal
- `markEncounterDefeatedIfVictorious()` (combatmode.go:809-838)
- `Exit()` (combatmode.go:694-738) - Orchestrates cleanup
- Battle recording export logic mixed in Exit()

**Issues:**
- **Hard to follow**: Setup/cleanup logic jumps between multiple methods
- **Error-prone**: Easy to forget cleanup steps (position system, caches, etc.)
- **Testing difficulty**: Can't test initialization without full combat mode
- **Typo in method name**: `initialzieCombatFactions()` indicates this code hasn't been reviewed carefully

**Code References:**
- Setup: `combatmode.go:614-806` (~190 lines across 3 methods)
- Cleanup: `combatmode.go:694-922` (~228 lines across 4 methods)

### Solution: Create `CombatLifecycleManager`

Consolidate all combat setup/teardown into a dedicated manager:

```go
// gui/guicombat/combatlifecycle.go
type CombatLifecycleManager struct {
    ecsManager      *common.EntityManager
    queries         *guicomponents.GUIQueries
    combatService   *combatservices.CombatService
    logManager      *CombatLogManager
    battleRecorder  *battlelog.BattleRecorder
}

// Setup phase - returns error if initialization fails
func (clm *CombatLifecycleManager) SetupEncounter(encounterID ecs.EntityID, playerPos coords.LogicalPosition) error
func (clm *CombatLifecycleManager) InitializeFactions() ([]ecs.EntityID, error)
func (clm *CombatLifecycleManager) StartBattleRecording(round int)

// Cleanup phase - idempotent, safe to call multiple times
func (clm *CombatLifecycleManager) CleanupCombatEntities(preserveExplorationSquads bool)
func (clm *CombatLifecycleManager) MarkEncounterDefeated(encounterID ecs.EntityID, victorFactionID ecs.EntityID) error
func (clm *CombatLifecycleManager) ExportBattleLog(victoryInfo *battlelog.VictoryInfo) error
```

**Benefits:**
- **Clear Separation**: Setup and cleanup are distinct phases with clear entry/exit points
- **Idempotent Cleanup**: Can call cleanup multiple times safely
- **Testable**: Can test spawn/cleanup logic independently
- **Easier to Debug**: All lifecycle code in one place

**Effort:** Medium (4-5 hours)
- Extract ~400 lines from CombatMode
- Consolidate 7 methods into lifecycle manager
- Add tests for setup/cleanup edge cases

---

## Recommendation 3: Consolidate Duplicate Rendering Code (LOW PRIORITY)

### Problem

`squad_combat_renderer.go` has significant code duplication:

**Duplicate Methods:**
- `RenderUnit()` (lines 49-142) - 94 lines
- `RenderUnitWithColor()` (lines 177-267) - 91 lines
- **Difference**: Only 3 lines (lines 263-264) apply ColorScale
- **Duplication**: 88 lines of identical grid calculation, scaling, and positioning logic

**Grid Position Calculation:**
- Duplicated in 3 places: `RenderUnit()`, `RenderUnitWithColor()`, and `RenderSquadWithHighlight()`
- Same rotation logic (90° clockwise/counter-clockwise)
- Same pixel position calculation

**Code References:**
- `squad_combat_renderer.go:49-142` - RenderUnit
- `squad_combat_renderer.go:177-267` - RenderUnitWithColor
- Duplication: ~88 lines (93% overlap)

### Solution: Consolidate Rendering Methods

```go
// Consolidated rendering with optional color
func (r *SquadCombatRenderer) RenderUnit(
    screen *ebiten.Image,
    unitID ecs.EntityID,
    baseX, baseY int,
    cellSize int,
    facingLeft bool,
    colorScale *ebiten.ColorScale, // nil = no color overlay
) {
    entity := r.queries.ECSManager.FindEntityByID(unitID)
    if entity == nil {
        return
    }

    // Get components
    gridPos, renderable, alive := r.getUnitRenderData(entity)
    if gridPos == nil || renderable == nil || !alive {
        return
    }

    // Calculate position (extracted helper)
    pixelX, pixelY := r.calculateUnitPosition(gridPos, baseX, baseY, cellSize, facingLeft)

    // Calculate scale and offset (extracted helper)
    scale, offsetX, offsetY := r.calculateUnitScale(gridPos, renderable, cellSize)

    // Build draw options
    op := r.buildDrawOptions(scale, offsetX, offsetY, pixelX, pixelY, facingLeft, colorScale)

    screen.DrawImage(renderable.Image, op)
}

// Helper methods (DRY)
func (r *SquadCombatRenderer) getUnitRenderData(entity *ecs.Entity) (*squads.GridPositionData, *rendering.Renderable, bool)
func (r *SquadCombatRenderer) calculateUnitPosition(gridPos *squads.GridPositionData, baseX, baseY, cellSize int, facingLeft bool) (int, int)
func (r *SquadCombatRenderer) calculateUnitScale(gridPos *squads.GridPositionData, renderable *rendering.Renderable, cellSize int) (float64, float64, float64)
func (r *SquadCombatRenderer) buildDrawOptions(scale, offsetX, offsetY float64, pixelX, pixelY int, facingLeft bool, colorScale *ebiten.ColorScale) *ebiten.DrawImageOptions
```

**Benefits:**
- **Reduced Code**: ~268 lines → ~180 lines (33% reduction)
- **Single Source of Truth**: Grid calculation in one place
- **Easier to Modify**: Change rotation logic once, applies everywhere
- **Better Tested**: Test one rendering path instead of three

**Effort:** Low (2-3 hours)
- Consolidate 3 methods into 1 with helpers
- Update 2 call sites to pass nil for no color
- Add unit tests for edge cases

---

## Impact Summary

| Recommendation | Priority | LOC Reduced | Complexity Reduced | Testability Gain | Effort |
|----------------|----------|-------------|-------------------|------------------|--------|
| AI Turn Orchestration | HIGH | ~140 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Medium |
| Lifecycle Manager | MEDIUM | ~400 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Medium |
| Rendering Consolidation | LOW | ~88 | ⭐⭐ | ⭐⭐⭐ | Low |

**Total Impact:**
- **Lines Removed**: ~628 lines
- **New Lines Added**: ~400 lines (net reduction: 228 lines)
- **Complexity**: Significantly reduced in AI turn flow and lifecycle management
- **Testability**: Major improvement - can test AI, lifecycle, and rendering independently

---

## Implementation Order

**Phase 1: Quick Win**
1. Rendering Consolidation (2-3 hours) - Low risk, immediate clarity benefit

**Phase 2: Core Refactoring**
2. Lifecycle Manager (4-5 hours) - Reduces CombatMode complexity significantly
3. AI Turn Orchestration (4-6 hours) - Highest complexity reduction

**Total Effort:** 10-14 hours across 3 phases

---

## Additional Notes

### Other Issues Found (Not Prioritized)

1. **Typo**: `initialzieCombatFactions()` should be `initializeCombatFactions()`
2. **TODO Comments**: `combatmode.go:227-228` mentions removing wrapper functions
3. **Large Struct**: CombatMode has 74 fields - consider grouping related fields into sub-structs
4. **Magic Numbers**: Animation timings in `combat_animation_mode.go` should be named constants (they are, good!)

### Testing Recommendations

After refactoring, add tests for:
- AI turn state machine (all transitions)
- Combat lifecycle (setup → cleanup → setup again)
- Rendering with various grid positions and color overlays
- Edge cases: fleeing mid-combat, encounter cleanup, battle recorder export

---

## References

**Project Guidelines:** `CLAUDE.md`
- ECS patterns: Pure data components, query-based, system functions
- Code style: Avoid over-engineering, only necessary changes
- Testing: Frequent `go test ./...` runs

**Files Modified:**
- `gui/guicombat/combatmode.go` (1,061 lines → ~600 lines after refactoring)
- `gui/guicombat/squad_combat_renderer.go` (268 lines → ~180 lines)
- New files: `tactical/ai/aiturnorchestrator.go`, `gui/guicombat/combatlifecycle.go`
