# GUI Package Refactoring Recommendations

**Updated:** 2025-11-28
**Scope:** gui/ package - duplication reduction and maintainability improvements

---

## High Priority


---

### 2. Extract CommandHistory Initialization

**Files:** `squadmanagementmode.go`, `formationeditormode.go`, `unitpurchasemode.go`

**Issue:** Identical CommandHistory setup in 3+ modes:
```go
mode.commandHistory = gui.NewCommandHistory(mode.setStatus, mode.refreshAfterUndoRedo)
```

**Action:** Add `InitializeCommandHistory(refreshCallback func())` to BaseMode that uses base SetStatus and the provided refresh callback.

---

### 3. Standardize Enter/Exit Logging

**Files:** All mode files

**Issue:** Every mode has `fmt.Println("Entering/Exiting X Mode")` in Enter/Exit.

**Action:** Move logging into BaseMode. Call mode-specific OnEnter/OnExit hooks that modes override. Print automatically uses `GetModeName()`.

---

### 4. Unit Template Lookup Utility

**Files:** `squadbuilder.go:173-179`, `squadbuilder.go:302-308`, `unitpurchasemode.go`

**Issue:** Multiple places loop through `squads.Units` to find template by name.

**Action:** Add `squads.GetTemplateByName(name string) *UnitTemplate` function.

---

## Medium Priority

### 5. Unify Close Button Handler Pattern

**Files:** Various modes

**Issue:** Some modes use `gui.CreateCloseButton`, others create buttons inline with same logic. Mix of `ModeCoordinator` and `ModeManager.RequestTransition` calls.

**Action:** Use `gui.CreateCloseButton` everywhere. Document when to use `ModeCoordinator` (context switches) vs `ModeManager` (same-context transitions).

---

### 6. Consolidate Panel Creation Approach

**Files:** `modehelpers.go`, various mode files

**Issue:** Three different approaches for panels with text areas:
- `gui.CreateDetailPanel()` - manual positioning
- `gui.CreateStandardDetailPanel()` - uses StandardPanels lookup
- Direct `widgets.CreatePanelWithConfig()` + `widgets.CreateTextAreaWithConfig()`

**Action:** Prefer `CreateStandardDetailPanel` with StandardPanels. Add specs to StandardPanels for common panel types. Migrate manual panel+textarea creation.

---

### 7. Create Bottom Action Button Builder

**Files:** 6+ mode files

**Issue:** Pattern of creating bottom-center container + adding action buttons repeated with identical layout logic.

**Action:** Extend helpers to accept button specs:
```go
CreateActionButtonGroup(specs []ButtonSpec, position PanelOption) *widget.Container
```

---

### 8. Consolidate Dialog Functions

**File:** `widgets/dialogs.go`

**Issue:** Three dialog functions duplicate 80% of their code (container setup, button layout, window creation).

**Action:** Create single `CreateDialog()` with config struct specifying type, content, buttons. Extract common setup.

---

### 9. Extract Role/TargetMode String Methods

**File:** `unitpurchasemode.go`

**Issue:** `getRoleName()` and `getTargetModeName()` helper methods belong on the types themselves.

**Action:** Add `String()` methods to `squads.UnitRole` and `squads.TargetMode` types. Remove local helpers.

---

## Low Priority

### 10. Consistent HandleInput Pattern

**Files:** Various modes, especially `unitpurchasemode.go`

**Issue:** Some modes handle ESC directly instead of calling `HandleCommonInput` first.

**Action:** All modes should call `HandleCommonInput` first for consistent ESC handling.

---

### 11. Consistent GUIQueries Usage

**Files:** Various modes

**Issue:** Some modes access ECS directly via `Context.ECSManager`, others use `Queries`. Mixed in same file (e.g., `formationeditormode.go` in `loadSquadFormation`).

**Action:** Prefer `Queries` for all ECS access in GUI code to maintain abstraction layer.

---

### 12. Remove Unused Components

**File:** `guicomponents/guicomponents.go`

**Issue:** `PanelListComponent`, `ButtonListComponent`, `ColorLabelComponent` may be unused or underutilized.

**Action:** Verify usage. Remove if unused. Document if used.

---

### 13. Fix ColorLabelComponent.SetColor

**File:** `guicomponents/guicomponents.go`

**Issue:** `SetColor()` has a note that it doesn't actually change color after creation.

**Action:** Either implement properly or document as read-only. Consider removing if not needed.

---

## Not Recommended

- **Factory Over-Abstraction:** Current factories (CombatUIFactory, SquadBuilderUIFactory) are appropriate. Don't create factories for simpler modes.
- **Widget Type Aliases:** Don't create aliases for ebitenui types.
- **Premature Interfaces:** Don't create `RefreshableMode`, `CommandableMode` interfaces unless 3+ modes would implement them.

---

## Implementation Order

1. **setStatus consolidation** - Quick win, 4 files, high impact
2. **Template lookup utility** - Prevents copy-paste bugs
3. **Enter/Exit logging** - Simplifies all modes
4. **CommandHistory initialization** - Reduces boilerplate
5. **HandleInput consistency** - Prevents subtle bugs
6. **Panel creation standardization** - Gradual migration
7. **Action button builder** - Reduces per-mode boilerplate
8. **Dialog consolidation** - Moderate effort, clear win
