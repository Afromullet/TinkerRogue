# GUI Package Refactoring Opportunities

**Generated:** 2025-11-18
**Package Size:** 30 files, 6,371 LOC
**Constraint:** Sub-packages NOT allowed (Go limitation)
**Current Status:** Well-organized with consistent patterns, but opportunities exist for improvement

---

## Executive Summary

The GUI package demonstrates strong architectural patterns:
- ✅ Clean layered architecture (infrastructure → utilities → components → modes)
- ✅ Consistent design patterns (Config, Functional Options, Component, Factory, Manager)
- ✅ Minimal global state (immutable resources only)
- ✅ Good separation of concerns

However, **12 refactoring opportunities** exist to improve readability, maintainability, and extensibility.

---

## Priority Matrix

| Priority | Count | Focus Area |
|----------|-------|------------|
| **HIGH** | 4 | Duplication elimination, consistency improvements |
| **MEDIUM** | 5 | Complexity reduction, better organization |
| **LOW** | 3 | Optional improvements, minor cleanup |

---

## High Priority Refactoring Opportunities

### 1. **Consolidate Panel Creation via Registry Pattern**
**Current State:**
- `panel_factory.go` defines `StandardPanels` registry (9 predefined specs)
- Some modes use the registry: `combat_ui_factory.go` uses `StandardPanels["statsPanel"]`
- Other modes hardcode panel creation: `explorationmode.go`, `squaddeploymentmode.go`
- Result: Inconsistent panel creation patterns across codebase

**Duplication Example:**
```go
// explorationmode.go - Hardcoded
statsPanel := BuildPanel(TopLeft(), Size(PanelWidthStandard, PanelHeightSmall), ...)

// combat_ui_factory.go - Registry
spec := StandardPanels["statsPanel"]
statsPanel := BuildPanelFromSpec(spec)
```

**Refactoring:**
1. Audit all panel creation across modes (8 mode files)
2. Extract hardcoded panel specs to `StandardPanels` registry
3. Update all modes to use `BuildPanelFromSpec()` consistently
4. Add new specs for mode-specific panels (deployment grid, squad builder grid, etc.)

**Benefits:**
- ✅ Single source of truth for panel layouts
- ✅ Easier to adjust UI layout globally
- ✅ Reduces duplication (estimated ~80 LOC savings)
- ✅ Improves consistency across modes

**Impact:** High - affects 8+ files, foundational UI pattern

---

### 2. **Unify Squad Filtering Logic**
**Current State:**
- Squad filtering logic exists in **2 locations**:
  - `guicomponents.go`: `PlayerSquadsOnly()`, `AliveSquadsOnly()`, `FactionSquadsOnly()` (component-level)
  - `filter_helper.go`: `FilterPlayerFactionSquads()`, `FilterAliveSquads()`, `FilterFactionSquads()` (helper-level)
- Duplication of filtering predicates
- Inconsistent naming (`PlayerSquadsOnly` vs `FilterPlayerFactionSquads`)

**Refactoring:**
1. **Choose canonical location:** Move all filtering to `guiqueries.go` (centralized query service)
2. **Standardize naming:** `FilterSquadsByPlayer()`, `FilterSquadsByFaction()`, `FilterSquadsAlive()`
3. **Composable predicates:** Allow combining filters (e.g., `Filter(PlayerOwned, AliveOnly)`)
4. **Update consumers:**
   - guicomponents.go → delegate to guiqueries
   - filter_helper.go → remove and redirect callers

**Benefits:**
- ✅ Single source of truth for filtering logic
- ✅ Easier to add new filters
- ✅ More testable (pure functions)
- ✅ Consistent naming

**Impact:** Medium-High - affects 2 files + all filter consumers

---

### 3. **Extract Combat Mode Input Handling to Strategy Pattern**
**Current State:**
- `combatmode.go` has 200+ LOC `HandleInput()` function
- Handles 5 distinct input modes: normal, attack selection, move selection, target selection, ability targeting
- Complex nested if/else chains for different modes
- Difficult to test individual input modes
- Hard to add new input modes (e.g., formation switching, ability menu)

**Current Structure:**
```go
func (m *CombatMode) HandleInput(inputState input.InputState) {
    if m.StateManager.InAttackMode() {
        // 50 LOC attack handling
    } else if m.StateManager.InMoveMode() {
        // 50 LOC move handling
    } else if m.StateManager.TargetingAbility() {
        // 40 LOC ability targeting
    } else {
        // 60 LOC normal mode handling
    }
}
```

**Refactoring:**
1. **Create input handler interface:**
```go
type CombatInputHandler interface {
    HandleInput(inputState input.InputState, ctx *CombatContext) error
}
```

2. **Extract handlers:** (in `combat_input_strategies.go`)
   - `NormalInputHandler` - Default squad/tile selection
   - `AttackSelectionHandler` - Target selection for attacks
   - `MoveSelectionHandler` - Valid tile selection for movement
   - `AbilityTargetingHandler` - Ability target selection

3. **Update CombatMode:**
```go
type CombatMode struct {
    currentHandler CombatInputHandler
    handlers       map[CombatState]CombatInputHandler
}1

func (m *CombatMode) HandleInput(inputState input.InputState) {
    m.currentHandler.HandleInput(inputState, m.getContext())
}
```

4. **State transitions:** `StateManager.SetState()` switches `currentHandler`

**Benefits:**
- ✅ Reduces complexity in `combatmode.go` (200 → 50 LOC in HandleInput)
- ✅ Each handler testable in isolation
- ✅ Easy to add new input modes without modifying existing code (Open/Closed Principle)
- ✅ Clear separation of input handling logic

**Impact:** High - major complexity reduction, improves extensibility

---

### 4. **Standardize Component Update Pattern**
**Current State:**
- `guicomponents.go` has 7 components with similar but inconsistent update patterns
- Some use `Refresh()`, others `Update()`, some both
- Some components have `Clear()`, others don't
- Makes it hard to understand component lifecycle

**Examples:**
```go
// SquadListComponent - Has Refresh() and Clear()
type SquadListComponent struct { ... }
func (c *SquadListComponent) Refresh(ctx *UIContext) { ... }
func (c *SquadListComponent) Clear() { ... }

// DetailPanelComponent - Has Update(), no Clear()
type DetailPanelComponent struct { ... }
func (d *DetailPanelComponent) Update(entityID ecs.EntityID) { ... }

// TextDisplayComponent - Has UpdateText(), no Clear()
type TextDisplayComponent struct { ... }
func (t *TextDisplayComponent) UpdateText(text string) { ... }
```

**Refactoring:**
1. **Define standard component interface:**
```go
type UIComponent interface {
    Refresh(ctx *UIContext) error  // Full rebuild with latest data
    Clear()                        // Reset to empty state
}

// Optional: For components with specific update needs
type UpdatableComponent interface {
    UIComponent
    Update(data interface{}) error  // Partial update with specific data
}
```

2. **Rename methods consistently:**
   - `Refresh()` - Full component rebuild from ECS/context
   - `Clear()` - Reset to empty/default state
   - `Update()` - Optional incremental update for performance

3. **Document lifecycle:**
   - Create → Refresh (initial) → [Update* | Refresh*] → Clear → Refresh (reset)

**Benefits:**
- ✅ Predictable component API across all 7 components
- ✅ Easier to understand component lifecycle
- ✅ Consistent naming improves code navigation
- ✅ Clear contract for new components

**Impact:** Medium - affects 7 components, improves consistency

---

## Medium Priority Refactoring Opportunities

### 5. **Create Combat Mode Facade**
**Current State:**
- `CombatMode` directly manages 6 dependencies:
  - `CombatStateManager`
  - `CombatLogManager`
  - `CombatActionHandler`
  - `CombatInputHandler`
  - `CombatUIFactory`
  - `GUIQueries`
- Constructor has 10+ parameters
- Mode file must understand internals of all managers

**Refactoring:**
1. **Create facade:** (in `combat_facade.go`)
```go
type CombatFacade struct {
    State   *CombatStateManager
    Log     *CombatLogManager
    Actions *CombatActionHandler
    Input   *CombatInputHandler
    UI      *CombatUIFactory
    Queries *GUIQueries
}

func NewCombatFacade(ctx *UIContext) *CombatFacade {
    return &CombatFacade{
        State:   NewCombatStateManager(),
        Log:     NewCombatLogManager(),
        Actions: NewCombatActionHandler(ctx),
        Input:   NewCombatInputHandler(ctx),
        UI:      NewCombatUIFactory(),
        Queries: ctx.GUIQueries,
    }
}
```

2. **Update CombatMode:**
```go
type CombatMode struct {
    BaseMode
    facade *CombatFacade
    // ... UI widgets ...
}

func NewCombatMode(ctx *UIContext) *CombatMode {
    return &CombatMode{
        facade: NewCombatFacade(ctx),
    }
}
```

**Benefits:**
- ✅ Simplified constructor (10 params → 1 facade)
- ✅ Clear dependency grouping
- ✅ Easier to mock for testing
- ✅ Reduces cognitive load when working with combat mode

**Impact:** Medium - improves combat code organization, but doesn't change logic

---

### 6. **Split guicomponents.go by Component Category**
**Current State:**
- Single file with 7 components (589 LOC)
- Components serve different purposes:
  - **List-based:** SquadListComponent, PanelListComponent, ButtonListComponent, ItemListComponent
  - **Display-based:** DetailPanelComponent, TextDisplayComponent, StatsDisplayComponent
- All mixed together in one file

**Refactoring:**
1. **Split by category:**
   - `guicomponents_lists.go` (296 LOC) - 4 list-based components
   - `guicomponents_displays.go` (293 LOC) - 3 display-based components

2. **Add file-level documentation:**
   - Each file explains component category
   - Cross-reference related components

3. **Maintain consistency:**
   - Keep naming convention (`{Purpose}Component`)
   - Keep standard interface (Create/Refresh/Clear)

**Benefits:**
- ✅ Easier to find specific component (navigation)
- ✅ Related components grouped together (cohesion)
- ✅ Reduces file size (589 → ~300 LOC per file)
- ✅ Clearer organization

**Alternative:** Keep single file but add clear separator comments (current state already has this)

**Impact:** Low-Medium - improves navigation, but current file already readable

---

### 7. **Consolidate Mode Helper Functions**
**Current State:**
- `modehelpers.go` has generic helpers (CloseButton, CreateButtonContainer)
- Mode-specific UI factories exist: `combat_ui_factory.go`, `squad_builder_ui_factory.go`
- Panel building split across 3 files: `panelconfig.go`, `panel_factory.go`, `modehelpers.go`
- Unclear where to add new helper functions

**Refactoring:**
1. **Clarify helper file purposes:**
   - `modehelpers.go` → **Common Mode Utilities** (buttons, containers, transitions)
   - `panel_factory.go` → **Panel Specifications** (StandardPanels registry only)
   - `panelconfig.go` → **Panel Options API** (functional options, no factories)
   - Mode-specific factories stay in `{mode}_ui_factory.go`

2. **Move panel building functions:**
   - `BuildPanel()`, `BuildPanelFromSpec()` → Move to `panel_factory.go`
   - Keep only widget-agnostic helpers in `modehelpers.go`

3. **Document guidelines:**
   - Where to add mode-specific UI builders
   - Where to add shared helpers

**Benefits:**
- ✅ Clear purpose for each helper file
- ✅ Easier to find helper functions
- ✅ Prevents helper file sprawl

**Impact:** Medium - organizational improvement, no logic changes

---

### 8. **Extract Squad Builder Grid Logic to Dedicated File**
**Current State:**
- `squadbuilder.go` has 425 LOC with mixed concerns:
  - Mode lifecycle (Enter, Exit, Update, Render)
  - UI building (buildUI, buildGridButtons)
  - Grid operations (placeUnit, removeUnit, setLeader)
  - Squad creation (createSquad, updateSquad)
  - Display updates (updateCapacityDisplay, updateDetailDisplay)

**Refactoring:**
1. **Split into 2 files:**
   - `squadbuilder.go` (250 LOC) - Mode implementation, UI building, lifecycle
   - `squadbuilder_grid_ops.go` (175 LOC) - Grid operations (place/remove/setLeader), squad creation

2. **Keep GridEditorManager separate** (already exists)

**Benefits:**
- ✅ Clearer separation: Mode vs Grid Operations
- ✅ Easier to test grid operations independently
- ✅ Reduced file size (425 → 250 LOC)

**Impact:** Medium - improves organization for complex mode

---

### 9. **Create Input Handling Guidelines Document**
**Current State:**
- Each mode implements `HandleInput()` differently
- Some use BaseMode.commonInputHandling(), others don't
- No clear pattern for hotkey vs mouse input
- Inconsistent escape key handling (sometimes closes mode, sometimes clears state)

**Refactoring:**
1. **Document input handling patterns:** (in `INPUT_PATTERNS.md`)
   - **Standard flow:** BaseMode.commonInputHandling() → Mode-specific input
   - **Hotkey precedence:** Global hotkeys → Mode hotkeys → Widget input
   - **Escape key convention:** Clear active state first, then close mode
   - **Mouse input:** Check widget bounds before mode-specific handling

2. **Update modes to follow pattern:**
   - Ensure all modes call `BaseMode.commonInputHandling()` first
   - Standardize escape key behavior

3. **Create input handler template:**
```go
func (m *MyMode) HandleInput(inputState input.InputState) {
    // 1. Common input (hotkeys, escape)
    if m.commonInputHandling(inputState) {
        return
    }

    // 2. Mode-specific state handling
    if m.hasActiveState() {
        m.handleStateInput(inputState)
        return
    }

    // 3. Default mode input
    m.handleDefaultInput(inputState)
}
```

**Benefits:**
- ✅ Consistent input handling across modes
- ✅ Easier to debug input issues
- ✅ Clear pattern for new modes
- ✅ Predictable escape key behavior

**Impact:** Medium - improves consistency and debuggability

---

## Low Priority Refactoring Opportunities

### 10. **Add Component Lifecycle Documentation**
**Current State:**
- Components follow Create → Refresh → Clear pattern
- Pattern not documented
- No guidance on when to use Refresh vs Update
- Unclear component ownership model

**Refactoring:**
1. **Create documentation:** (in `COMPONENT_PATTERNS.md`)
   - Component lifecycle diagram
   - When to use each method (Create/Refresh/Update/Clear)
   - Ownership model (mode owns components, components own widgets)
   - Best practices (avoid storing ECS entities, use EntityIDs)

2. **Add code examples:**
   - Typical component usage in modes
   - How to create new components

**Benefits:**
- ✅ Clearer component mental model
- ✅ Easier onboarding for new contributors
- ✅ Reduces component misuse

**Impact:** Low - documentation only, but valuable for maintainability

---

### 11. **Consolidate Layout Constants Usage**
**Current State:**
- `layout_constants.go` defines panel size constants (PanelWidthNarrow, PanelHeightSmall, etc.)
- Some modes use constants: `Size(PanelWidthStandard, PanelHeightSmall)`
- Other modes hardcode values: `Size(0.2, 0.15)`
- Inconsistent constant usage

**Refactoring:**
1. **Audit all panel Size() calls** (8 mode files + factories)
2. **Replace hardcoded values with constants:**
   - `Size(0.2, 0.15)` → `Size(PanelWidthStandard, PanelHeightSmall)`
   - `Size(0.15, 0.05)` → `Size(PanelWidthNarrow, PanelHeightTiny)`

3. **Add missing constants if needed:**
   - If custom sizes are truly needed, document why
   - Consider adding to constants if used multiple times

**Benefits:**
- ✅ Consistent sizing across UI
- ✅ Easier to adjust layout globally
- ✅ Self-documenting sizes (PanelHeightSmall vs 0.15)

**Impact:** Low - minor consistency improvement

---

### 12. **Add File Organization Comments**
**Current State:**
- 30 files in package, no organizational comments
- Unclear which files are core infrastructure vs mode-specific
- No guidance on where to add new code

**Refactoring:**
1. **Add package-level README comment in a core file** (e.g., `uimode.go`):
```go
/*
Package gui provides the UI mode system for TinkerRogue.

FILE ORGANIZATION:
- Core Infrastructure:
    uimode.go, basemode.go, modemanager.go
    layout.go, layout_constants.go, panelconfig.go
    guiresources.go, createwidgets.go

- Query & Helper Services:
    guiqueries.go, filter_helper.go, modehelpers.go

- Reusable Components:
    guicomponents.go (7 major components)

- Mode Implementations:
    combatmode.go + combat_*.go (6 files)
    squadbuilder.go + squad_builder_*.go (3 files)
    explorationmode.go, inventorymode.go, infomode.go, etc.

- Panel & Factory Utilities:
    panel_factory.go, guirenderers.go

ADDING NEW CODE:
- New mode → {name}mode.go (embed BaseMode)
- Mode-specific manager → {mode}_{subject}_manager.go
- Mode-specific UI factory → {mode}_ui_factory.go
- Shared helper → modehelpers.go
- Reusable component → guicomponents.go
*/
```

2. **Add section comments in large files:**
   - `guicomponents.go` - Already has separators, ensure they're clear
   - `createwidgets.go` - Add "Widget Factory Functions" section headers

**Benefits:**
- ✅ Easier to navigate package
- ✅ Clear guidance for new developers
- ✅ Reduces "where does this go?" questions

**Impact:** Low - documentation only, but improves onboarding

---

## Refactoring Roadmap

### Phase 1: High-Impact, Low-Risk (Estimated: 4-6 hours)
1. ✅ Consolidate Panel Creation via Registry (#1)
2. ✅ Unify Squad Filtering Logic (#2)
3. ✅ Standardize Component Update Pattern (#4)
4. ✅ Consolidate Layout Constants Usage (#11)

**Result:** Eliminates duplication, improves consistency

---

### Phase 2: Complexity Reduction (Estimated: 6-8 hours)
5. ✅ Extract Combat Mode Input Handling to Strategy Pattern (#3)
6. ✅ Create Combat Mode Facade (#5)
7. ✅ Consolidate Mode Helper Functions (#7)

**Result:** Reduces combat mode complexity, clearer dependencies

---

### Phase 3: Organization & Documentation (Estimated: 2-3 hours)
8. ✅ Extract Squad Builder Grid Logic (#8)
9. ✅ Create Input Handling Guidelines Document (#9)
10. ✅ Add Component Lifecycle Documentation (#10)
11. ✅ Add File Organization Comments (#12)

**Result:** Better navigation, clearer patterns

---

### Phase 4: Optional Improvements (Estimated: 2-3 hours)
12. ✅ Split guicomponents.go by Component Category (#6) - **Optional**

**Result:** Further file size reduction (if desired)

---

## Metrics

### Current State
- **Files:** 30
- **Total LOC:** 6,371
- **Average File Size:** 212 LOC
- **Largest Files:** guicomponents.go (589), squadbuilder.go (425), createwidgets.go (391), combatmode.go (352)
- **Duplication:** ~150 LOC (panel creation, filtering, input patterns)
- **Global State:** Minimal (immutable resources only) ✅

### Post-Refactoring Projections
- **LOC Reduction:** ~200-300 LOC (duplication elimination)
- **File Count:** 30-32 (minimal change, possible strategy pattern additions)
- **Largest Files:** All < 400 LOC
- **Duplication:** < 50 LOC
- **Complexity Metrics:** Reduced cyclomatic complexity in combat mode (estimated 30% reduction)

---

## Testing Strategy

For each refactoring:
1. **Preserve behavior:** Ensure all modes work identically before/after
2. **Manual testing:** Test affected modes (combat, squad builder, exploration)
3. **Visual regression:** Check UI layout unchanged
4. **Input testing:** Verify all hotkeys and mouse input work

**High-risk refactorings:**
- #3 Combat Input Strategy - Extensive combat testing needed
- #5 Combat Facade - Verify all combat actions still work

**Low-risk refactorings:**
- #1 Panel Registry - Visual check of panel positions
- #2 Squad Filtering - Verify filtering results unchanged
- #11 Layout Constants - Visual regression test

---

## Conclusion

The GUI package is **fundamentally well-architected** but has **tactical refactoring opportunities** to improve:
- **Consistency:** Panel creation, filtering, component updates
- **Complexity:** Combat mode input handling, dependency management
- **Organization:** File organization, helper consolidation
- **Documentation:** Patterns, lifecycle, guidelines

**Recommended approach:** Execute refactorings in phases, prioritizing high-impact consistency improvements first, then complexity reduction, then documentation.

**Estimated total effort:** 14-20 hours for all refactorings (Phases 1-3)

**Critical constraint:** Single-package design is already appropriate - no sub-package refactoring needed.
