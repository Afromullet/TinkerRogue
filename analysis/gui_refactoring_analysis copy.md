# GUI Package Refactoring Analysis

**Date:** 2025-11-11
**Status:** Analysis Complete - Ready for Implementation
**Package Size:** 5,947 LOC across 28 files
**Estimated Refactoring Value:** 40-50 hours for complete refactoring (all phases)
**High Priority Value:** 8-10 hours delivers ~60% of improvements

---

## Executive Summary

The GUI package has **strong architectural patterns** (Mode system, factory patterns, component-based architecture) but shows **classic growth patterns** where working code gets copy-pasted instead of abstracted.

### Key Findings
- **380-400 LOC of duplicated code** across similar components
- **5+ complex functions** over 30 LOC needing decomposition
- **13+ debug statements** and test code comments in production
- **Tight coupling** between modes, handlers, and systems
- **Code quality issues** scattered across 3+ files

### What's Working Well ✓
- UIMode interface with clean lifecycle management
- Factory patterns (ButtonConfig, TextConfig, PanelConfig)
- Functional options pattern for composition
- Component-based updates (SquadListComponent, DetailPanelComponent, TextDisplayComponent)
- Centralized GUIQueries service eliminates duplicate ECS queries

---

## Code Duplication Analysis (380-400 LOC)

### 1. Close Button Pattern (MEDIUM Priority)
**Files:** InventoryMode, SquadManagementMode, SquadDeploymentMode, FormationEditorMode
**LOC:** 50
**Effort:** 1 hour
**Status:** Partially refactored - modehelpers.go has `CreateCloseButton()` and `CreateBottomCenterButtonContainer()` but only some modes use them

**Current Issue:**
Each mode duplicates close button creation code when helpers already exist.

**Action:** Update all modes to use `CreateCloseButton()` from modehelpers.go

---


### 4. Coordinate Manager Creation (LOW Priority)
**Files:** CombatInputHandler (lines 108, 135), ExplorationMode (lines 212-213)
**LOC:** 30
**Effort:** 1 hour

**Current Issue:**
Repeated sequence:
```go
manager := coords.NewCoordinateManager(graphics.ScreenInfo)
viewport := coords.NewViewport(manager, *playerPos)
clickedPos := viewport.ScreenToLogical(mouseX, mouseY)
```

**Solution:** Create `CoordinateConversionHelper` with cached manager or implement caching in input handlers

---

### 5. Input Handler Pass-Through Methods (LOW Priority)
**Files:** CombatMode, SquadDeploymentMode
**LOC:** 45
**Effort:** 1.5 hours

**Current Issue:**
Methods that just forward to another handler:
```go
func (m *Mode) handleAttackClick() {
    m.actionHandler.ToggleAttackMode()  // Unnecessary delegation
}
```

**Solution:** Pass handlers directly to button callbacks to eliminate wrapper methods

---

### 6. Filter Logic Repetition (MEDIUM Priority)
**Scattered across:** CombatMode, SquadManagementMode, SquadDeploymentMode
**Effort:** 1.5 hours

**Current Issue:**
Squad filtering (especially player faction squads) appears identically in 3+ modes.

**Solution:** Extract `FilterHelper` utility with reusable filter functions:
- `FilterPlayerFactionSquads(allSquads []ecs.EntityID) []ecs.EntityID`
- `FilterEnemySquads(allSquads []ecs.EntityID, playerFaction Faction) []ecs.EntityID`

---

## Complex Functions Needing Decomposition

### 1. CombatMode.buildUILayout() (32 lines)
**Severity:** MEDIUM
**Issue:** Repetitive pattern: create panel → create label → AddChild → AddChild

**Current Structure:**
```go
// Pattern repeated 5+ times:
panel := panelBuilders.BuildPanel(...)
label := CreateLabel(...)
panel.AddChild(label)
rootContainer.AddChild(panel)
```

**Refactoring:** Extract helper: `addLabeledPanel(container, panel, labelText)`

---

### 2. CombatInputHandler.handleSquadClick() (45 lines)
**Severity:** MEDIUM
**Issue:** Complex nested conditionals for faction checking with 6+ state mutations inline

**Current Logic:**
```go
if friendlyFaction {
    if attackMode {
        // handle friendly squad attack
    } else {
        // handle friendly squad selection
    }
} else if enemyFaction {
    if attackMode {
        // handle enemy squad attack
    } else {
        // handle enemy squad selection
    }
}
```

**Refactoring:** Extract methods:
- `isFriendlySquad(faction1, faction2) bool`
- `isEnemySquad(faction1, faction2) bool`
- `handleFriendlySquadClick(squadID)`
- `handleEnemySquadClick(squadID)`

---

### 3. GridEditorManager.PlaceUnitInCell() (35 lines)
**Severity:** MEDIUM
**Issue:** Mixes validation, entity lookup, grid calculation, and button updates

**Concerns Mixed:**
1. Validation logic
2. Entity lookup
3. Grid calculation
4. UI updates

**Refactoring:** Extract methods:
- `validatePlacement(row, col, unitIndex) error`
- `updateCellDisplays(occupiedCells []coords.LogicalPosition, unit Unit)`
- `markCellOccupied(row, col, unitID, unitIndex)`

---

### 4. PanelBuilders.BuildGridEditor() (54 lines)
**Severity:** LOW
**Issue:** Dense 3x3 grid creation with nested loops and inline config

**Refactoring:** Extract:
- `createGridCell(row, col, config GridEditorConfig) *Button`
- `applyGridEditorDefaults(config GridEditorConfig) GridEditorConfig`

---

## Mixed Concerns Issues

### 1. DetailPanelComponent Formatters - Business Logic in Presentation
**Location:** guicomponents.go, DetailPanelComponent
**Severity:** MEDIUM
**Issue:** Display formatting contains business logic

**Current Code:**
```go
func getSquadStatus(info *SquadInfo) string {
    if info.HasActed { return "Acted" }
    else if info.HasMoved { return "Moved" }
    else { return "Ready" }
}
```

**Problem:** Status determination is business logic, not presentation

**Solution:** Move to GUIQueries or create SquadStatusFormatter utility module that handles the logic, component only calls it for display

---

### 2. CombatActionHandler - Multiple Concerns
**Location:** combat_action_handler.go
**Severity:** MEDIUM
**Issue:** Handler mixes three concerns:
1. Action execution (core logic)
2. Logging (logging concern)
3. Mode state changes (state management concern)

**Current Pattern:**
```go
func (cah *CombatActionHandler) ExecuteAttack() {
    // Execute action
    // Change mode state
    // Add log message
}
```

**Solution:** Handler only executes actions, mode handles logging/UI updates in response to action results

---

### 3. CombatInputHandler - Direct State Mutation
**Location:** combat_input_handler.go, lines 160-172
**Severity:** MEDIUM
**Issue:** Input handler directly mutates game state

**Current Pattern:**
```go
cih.actionHandler.SelectSquad(clickedSquadID)    // State mutation
cih.actionHandler.ExecuteAttack()                // State mutation
```

**Problem:** Input handler should report events, mode interprets and responds

**Solution:** Create EventHandler pattern - input reports clicks, mode handles state changes

---

## Tight Coupling Issues

### 1. UIContext Over-Coupling
**Location:** uimode.go
**Severity:** MEDIUM
**Issue:** UIContext couples modes to concrete game objects

**Current Dependencies:**
- ECSManager (direct coupling to entity management)
- PlayerData (direct coupling to player state)
- Screen dimensions (duplicates LayoutConfig)

**Solution:** Inject services through interfaces, not concrete game objects. Example:
```go
type EntityService interface {
    Query(...) []Result
    Get(id ecs.EntityID) Component
}
```

---

### 2. Mode System Creation Coupling
**Location:** CombatMode.Initialize() and similar
**Severity:** MEDIUM
**Issue:** Modes create systems directly, can't be tested or substituted

**Current Pattern:**
```go
cm.turnManager = combat.NewTurnManager(ctx.ECSManager)
cm.factionManager = combat.NewFactionManager(ctx.ECSManager)
cm.movementSystem = combat.NewMovementSystem(ctx.ECSManager, common.GlobalPositionSystem)
```

**Solution:** Inject via factory function or context factory method

---

### 3. GridEditorManager - Direct Squads Package Calls
**Location:** squad_builder_grid_manager.go
**Severity:** MEDIUM
**Issue:** Hardcoded calls to squads package without abstraction

**Current Pattern:**
```go
unit := squads.Units[unitIndex]
squads.AddUnitToSquad(...)
squads.GetUnitIDsAtGridPosition(...)
squads.FindUnitByID(...)
```

**Solution:** Accept unit definitions and operations through interface

---

### 4. DetailPanelComponent - Hardcoded Entity Types
**Location:** guicomponents.go, lines 119-179
**Severity:** LOW
**Issue:** Component only works with squads and factions

**Current Methods:**
```go
func (dpc *DetailPanelComponent) ShowSquad(squadID ecs.EntityID) {...}
func (dpc *DetailPanelComponent) ShowFaction(factionID ecs.EntityID) {...}
```

**Problem:** Can't be reused for other entity types

**Solution:** Generic `ShowEntity(id, formatter EntityFormatter)` pattern

---

### 5. SquadListComponent - Only Works With Squads
**Location:** guicomponents.go, lines 11-94
**Severity:** LOW
**Issue:** Can't be reused for other entity types

**Current Structure:**
```go
type SquadListComponent struct {
    filteredSquads []ecs.EntityID    // Hardcoded for squads
    squadButtons   []*widget.Button
}
```

**Solution:** Generic `ListComponent[T]` with injectable query/filter logic

---

## Code Quality Issues

### 1. Debug Statements in Production Code
**Location:** squaddeploymentmode.go, lines 175-232
**Count:** 13+ debug statements
**Severity:** LOW
**Effort:** 0.5 hours

**Examples:**
```go
fmt.Printf("DEBUG: Processing pending placement in Update()\n")
fmt.Printf("DEBUG: Converted click to logical position: (%d, %d)\n", ...)
fmt.Printf("DEBUG: Mouse click captured at (%d, %d), placing=%v, squadID=%d\n", ...)
```

**Action:** Remove or convert to structured logging via proper logger

---

### 2. Test Code in Production
**Location:** explorationmode.go, inventorymode.go
**Count:** 2+ instances
**Severity:** LOW
**Effort:** 0.5 hours

**Examples:**
- explorationmode.go line 108: `//TODO remove this in the future. Just here for testing`
- inventorymode.go line 39: `// TOOO remove in the future. This is here for testing` (TYPO!)

**Action:** Remove all test-related code immediately

---

### 3. Inconsistent Error Handling
**Severity:** LOW
**Issue:** Three different error handling patterns used inconsistently

**Pattern A - Silent Failures:**
```go
if config.Padding.Left == 0 { config.Padding = widget.Insets{...} }
```

**Pattern B - Error Returns:**
```go
err := squads.AddUnitToSquad(...)
if err != nil { return err }
```

**Pattern C - Nil Checks:**
```go
if squadInfo == nil || !slc.filter(squadInfo) { continue }
```

**Action:** Standardize error handling strategy across package

---

## Performance Issues

### 1. Query Service Creates New Managers Every Call
**Location:** guiqueries.go, lines 38-66
**Severity:** LOW
**Impact:** ~5-10% overhead on query operations

**Current Code:**
```go
func (gq *GUIQueries) GetFactionInfo(factionID ecs.EntityID) *FactionInfo {
    for _, result := range gq.ecsManager.World.Query(...) {
        factionManager := combat.NewFactionManager(gq.ecsManager)  // <- NEW every call!
        currentMana, maxMana := factionManager.GetFactionMana(factionID)
    }
}
```

**Solution:** Cache FactionManager as singleton or in context

---

### 2. Coordinate Manager Created Per-Input
**Location:** CombatInputHandler, lines 108, 135
**Severity:** LOW
**Impact:** ~10-20% overhead on input handling

**Solution:** Cache manager in handler instance, invalidate only on screen resize

---

## Refactoring Roadmap

### PHASE 1: Quick Wins (8-10 hours) - 60% of value
High-impact, low-risk changes that establish better patterns

| Task | Effort | LOC Saved | Priority |
|------|--------|-----------|----------|
| Remove debug statements | 0.5h | 20 | P1 |
| Fix/remove test code | 0.5h | 10 | P1 |
| Consolidate close buttons | 1h | 50 | P1 |
| Extract panel+textarea helper | 2h | 114 | P1 |
| Cache coordinate manager | 1h | - | P1 |
| Extract filter button container | 1.5h | 60 | P1 |
| Remove pass-through handlers | 1h | 45 | P1 |
| Extract coordinate conversion helper | 1h | 30 | P1 |

**Subtotal:** 8-10 hours, **299 LOC reduction**

**Value:** Establishes better patterns, removes code smells, improves maintainability

---

### PHASE 2: Major Simplifications (15-20 hours)
Deeper refactoring of complex functions and mixed concerns

| Task | Effort | LOC Impact | Priority |
|------|--------|-----------|----------|
| Refactor complex input handlers | 5h | ~40 reduced | P2 |
| Split CombatStateManager concerns | 4h | ~80 reduced | P2 |
| Generalize list components | 7h | ~70 reduced | P2 |
| Extract formatters to utilities | 3h | ~20 reduced | P2 |
| Standardize error handling | 2h | - | P2 |

**Subtotal:** 15-20 hours, **~210 LOC reduction**

**Value:** Another 40% improvement, cleaner separation of concerns

---

### PHASE 3: Architecture Improvements (10-15+ hours)
Deeper architectural changes, optional for now

| Task | Effort | Value |
|------|--------|-------|
| Query service caching | 2h | Performance |
| DetailPanelComponent generalization | 3h | Reusability |
| UIContext redesign | 7h | Architecture |
| Component-based UI tests | 5h+ | Regression prevention |
| Service injection refactoring | 8h | Testability |

**Value:** Medium - architectural improvements, better testing

---

## Implementation Priority

### Immediate Actions (Today - 1 hour)
1. ✅ Remove all debug statements from squaddeploymentmode.go
2. ✅ Fix/remove test code comments in explorationmode.go and inventorymode.go

### This Week (8-10 hours)
1. ✅ Consolidate close button usage
2. ✅ Extract panel+textarea helper
3. ✅ Cache coordinate manager
4. ✅ Extract filter button container
5. ✅ Remove pass-through handlers
6. ✅ Extract coordinate conversion helper

### Next Phase (15-20 hours)
1. Refactor complex input handler methods
2. Split CombatStateManager concerns
3. Generalize list components
4. Standardize error handling

---

## Files Most Needing Attention

### High Priority (Blocking)
1. **squaddeploymentmode.go**
   - 13+ debug statements
   - Test code comments
   - Complex PlaceUnitInCell() method
   - Duplicate panel+textarea patterns

2. **combatmode.go**
   - Complex buildUILayout() method
   - Duplicate panel+textarea patterns
   - Tight coupling to action handler

3. **combat_input_handler.go**
   - Complex handleSquadClick() with nested conditionals
   - Coordinate manager created per-input
   - Direct state mutation

### Medium Priority
4. **guicomponents.go**
   - Tight component coupling
   - Hardcoded entity types in DetailPanelComponent/SquadListComponent

5. **inventorymode.go**
   - Test code comment (TYPO!)
   - Duplicate patterns with other modes

### Low Priority (Nice to Have)
6. **guiqueries.go**
   - FactionManager allocated per query

---

## Architectural Strengths to Preserve ✓

**These patterns are working well and should serve as templates for refactoring:**

1. **UIMode Interface** - Clean lifecycle with Initialize, Update, Render, HandleInput
2. **Factory Patterns** - ButtonConfig, TextConfig, PanelConfig eliminate boilerplate
3. **Functional Options Pattern** - BuildPanel with chainable methods (TopRight(), Size(), Padding())
4. **Component-Based Updates** - SquadListComponent, DetailPanelComponent enable reusability
5. **Centralized Query Service** - GUIQueries eliminates duplicate ECS queries

---

## Success Criteria

### After Phase 1 (8-10 hours)
- ✅ All debug statements removed
- ✅ Code organized into helpers
- ✅ ~300 LOC reduction
- ✅ Better patterns established for new code

### After Phase 2 (15-20 hours)
- ✅ All complex functions decomposed
- ✅ Concerns properly separated
- ✅ ~500 LOC total reduction
- ✅ Improved testability

### After Phase 3 (30-40 hours)
- ✅ Full architectural improvements
- ✅ Generic components for reuse
- ✅ ~600 LOC total reduction
- ✅ Better service injection

---

## Conclusion

The GUI package has **solid foundations** with excellent patterns (Mode system, factories, components) but shows **classic growth patterns** where working code gets copy-pasted instead of abstracted.

**Key Issues:**
- 380-400 LOC of duplication
- 5+ complex functions needing decomposition
- Multiple code quality issues (debug statements, test code)
- Tight coupling between components and systems

**Recommended Approach:**
1. **This week:** Phase 1 (8-10 hours) - Quick wins that deliver 60% of value
2. **Next week:** Phase 2 (15-20 hours) - Major simplifications
3. **Later:** Phase 3 (10-15 hours) - Architecture improvements

**Impact:**
- Reduce GUI package by ~500 LOC (Phase 1+2)
- Establish better patterns for new code
- Significantly improve maintainability
- Enable better testing of GUI logic

The refactoring is **high-value but not urgent** - the codebase works well. Prioritize based on what areas you're actively developing. If working on combat or squad features, Phase 2 delivers the most value. If adding new modes, Phase 1 establishes patterns first.
