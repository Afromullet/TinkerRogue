# GUI Refactoring Recommendations

**Analysis Date:** 2025-11-27
**Scope:** gui/ package analysis for duplication reduction and maintainability improvements

---

## High-Value Refactorings

### 1. Consolidate Dialog Creation Functions

**Location:** `gui/widgets/dialogs.go`

**Problem:** Three separate dialog functions (`CreateConfirmationDialog`, `CreateTextInputDialog`, `CreateMessageDialog`) duplicate 80% of their code for container setup, button layout, and window creation.

**Recommendation:** Create single `CreateDialog()` function accepting config struct that specifies dialog type, content widgets, and button set. Extract common container/window/button setup logic.

**Value:** Reduces 200+ lines to ~100 lines, makes adding new dialog types trivial.

---

### 2. Extract Common Status Label Pattern

**Location:** `guisquads/squadmanagementmode.go:771`, `guisquads/formationeditormode.go:404`

**Problem:** Multiple modes implement identical `setStatus(message string)` method that updates label text and logs to console.

**Recommendation:** Add `SetStatus(message string)` to `BaseMode`. Modes declare `statusLabel *widget.Text` and BaseMode handles update + logging.

**Value:** Eliminates 10-15 lines per mode, ensures consistent status handling across all modes.

---

### 3. Standardize Command Executor Integration

**Location:** `guisquads/squadmanagementmode.go`, `guisquads/formationeditormode.go`

**Problem:** Both modes duplicate:
- Command executor initialization
- Identical `onUndo()` / `onRedo()` implementations
- Undo/redo keyboard input handling (Ctrl+Z, Ctrl+Y)
- Undo/redo button creation

**Recommendation:** Create `CommandExecutorMixin` or helper struct in `gui/` that provides:
- Pre-configured command executor
- Standard undo/redo handlers
- Button spec generation for undo/redo buttons
- Input handling for Ctrl+Z/Ctrl+Y

**Value:** Reduces 100+ lines per mode using commands, ensures consistent undo/redo UX.

---

### 4. Create Bottom Action Button Builder

**Location:** Multiple modes create bottom-center button containers

**Problem:** Pattern of creating bottom-center container + adding action buttons is repeated in 6+ modes with identical layout logic but manual button-by-button construction.

**Recommendation:** Extend `CreateBottomCenterButtonContainer()` to accept button specs:
```go
CreateActionButtonGroup(specs []ButtonSpec, position PanelOption) *widget.Container
```

**Value:** Reduces button group creation from 20-30 lines to 5-10 lines per mode.

---

### 5. Remove Empty Enter/Exit Stubs

**Location:** Most mode files have `Enter()` / `Exit()` with just print statements

**Problem:** BaseMode already provides default implementations. Empty stubs obscure which modes have actual initialization logic.

**Recommendation:** Delete stub implementations. Only override `Enter()`/`Exit()` when mode has specific logic beyond logging.

**Value:** Cleaner codebase, easier to identify modes with special initialization needs.

---

### 6. Consolidate Context Switching Helpers

**Location:** `guimodes/explorationmode.go:103`, `guisquads/squadmanagementmode.go:191`

**Problem:** Battle Map ↔ Overworld context switch code duplicated in multiple button callbacks.

**Recommendation:** Add helpers to `BaseMode`:
```go
func (bm *BaseMode) SwitchToOverworld(targetMode string)
func (bm *BaseMode) SwitchToBattleMap(targetMode string)
```

**Value:** Eliminates 5-10 lines per context switch, centralizes coordinator calls.

---

## Medium-Value Refactorings

### 7. Unify Factory Pattern Usage

**Location:** `guicombat/combat_ui_factory.go`, `guisquads/squad_builder_ui_factory.go`, inline UI building

**Problem:** CombatMode and SquadBuilderMode use dedicated factories while ExplorationMode, InventoryMode, FormationEditorMode build UI inline. Inconsistent pattern makes code navigation harder.

**Recommendation:** Make architectural decision:
- **Option A:** All modes use factories (more boilerplate but consistent)
- **Option B:** Remove factories, move methods to shared helpers (less abstraction)

**Value:** Consistent codebase structure, easier onboarding for new contributors.

---

### 8. Create Mode Initialization Builder

**Location:** Every mode's `Initialize()` method

**Problem:** First 5-10 lines of every `Initialize()` are identical: call `InitializeBase()`, register hotkeys, initialize queries.

**Recommendation:** Create initialization helper:
```go
func (bm *BaseMode) InitializeWithHotkeys(ctx *UIContext, hotkeys map[ebiten.Key]string) error
```

**Value:** Reduces per-mode boilerplate by 10-15 lines, standardizes initialization order.

---

### 9. Extract List Creation Helpers

**Location:** FormationEditorMode, SquadManagementMode, InventoryMode all create lists

**Problem:** Similar list creation patterns with minor variations. Common behaviors (empty states, entry formatting) reimplemented each time.

**Recommendation:** Create typed list helpers:
```go
CreateSquadList(onSelect func(squadID ecs.EntityID)) *widget.List
CreateUnitList(entries []UnitEntry, onSelect func(unitID ecs.EntityID)) *widget.List
```

**Value:** Reduces list creation from 30-40 lines to 5-10 lines, ensures consistent list behavior.

---

### 10. Merge Duplicate Panel Creation Functions

**Location:** `gui/modehelpers.go` has `CreateDetailPanel()` and `CreateStandardDetailPanel()`

**Problem:** Two functions do nearly identical work - create panel with textarea. Unclear when to use which.

**Recommendation:** Merge into single function accepting optional spec name:
```go
func CreateDetailPanel(pb *PanelBuilders, layout *LayoutConfig,
    specName string, // empty string means use raw params
    position PanelOption, width, height, padding float64,
    defaultText string) (*widget.Container, *widget.TextArea)
```

**Value:** Reduces confusion, eliminates 20 lines of duplicate code.

---

## Lower-Priority Opportunities

### 11. Simplify StandardPanels Configuration

**Location:** `gui/widgets/panelconfig.go`

**Problem:** Panel specs are verbose. Creating panel from spec requires manual option building.

**Recommendation:** Use builder pattern or reduce specs to presets (small/medium/large) + position.

**Value:** Minimal - current system works fine, just verbose.

---

### 12. Consolidate Grid Editor Pattern

**Location:** FormationEditorMode and SquadBuilderMode both manage 3x3 grids

**Problem:** Similar but separate implementations of grid cell management.

**Recommendation:** Extract shared grid management behavior into reusable component.

**Value:** Moderate - only affects two modes currently, may not justify extraction.

---

### 13. Initialize Components With Their Widgets

**Location:** CombatMode separates widget creation from component initialization

**Problem:** UI building and component initialization are separate phases, creating confusion about when components are ready.

**Recommendation:** Initialize components immediately when creating widgets. Keep related code co-located.

**Value:** Improves code readability, reduces potential nil pointer issues.

---

## Implementation Priority Tiers

### Tier 1 - Quick Wins (Do First)
1. **Remove Empty Enter/Exit Stubs** - Pure cleanup, no risk
2. **Extract Common Status Label** - High reuse, simple change
3. **Create Bottom Action Button Builder** - Touches many files, clear improvement

### Tier 2 - High Value (Plan Soon)
4. **Consolidate Dialog Functions** - Clear win, moderate effort
5. **Standardize Command Executor Integration** - Helps squad modes significantly
6. **Context Switching Helpers** - Small change, improves clarity

### Tier 3 - Architectural (Decide Then Execute)
7. **Unify Factory Pattern** - Needs architectural decision first
8. **Mode Initialization Builder** - Good cleanup, affects all modes
9. **Extract List Creation Helpers** - Useful but needs type design

### Defer Until More Patterns Emerge
- Items 10-13 - Wait to see if more modes need these patterns

---

## Anti-Patterns to Avoid

**Don't:**
- Force consistency where modes have genuinely different needs
- Abstract too early - wait until pattern appears 3+ times
- Refactor stable code that isn't causing problems
- Create "framework" features that no mode currently needs

**Do:**
- Focus on areas where new features will be added
- Reduce duplication in recently-changed code
- Make common tasks easier for future mode development
- Keep refactorings small and reversible

---

## Notes

- Current GUI architecture is generally sound
- Most duplication emerged naturally as modes were added incrementally
- Focus refactoring efforts on squad management and combat UI areas
- Estimated impact: 20-30% reduction in GUI code, easier mode addition
