# Codebase Refactoring Recommendations

**Updated:** 2025-11-28
**Focus:** Value-adding refactors only - cleanup, maintainability, extensibility

---

## High Priority

### 1. GUI setStatus/CommandHistory Consolidation

**Files:** `gui/guisquads/squadmanagementmode.go`, `formationeditormode.go`, `unitpurchasemode.go`

**Issue:** 3 modes duplicate identical `setStatus()` methods and CommandHistory initialization when BaseMode already provides `SetStatus()`.

**Action:** Remove mode-specific `setStatus`. Add `BaseMode.InitializeCommandHistory(refreshCallback)` helper.

**Impact:** Eliminates ~50 lines of duplication, simplifies adding new modes.

---

### 2. Unit Template Lookup Utility

**Files:** `gui/guisquads/squadbuilder.go:173-179`, `squadbuilder.go:302-308`, `unitpurchasemode.go`

**Issue:** Three locations loop through `squads.Units` to find template by name.

**Action:** Add to `squads/unitroster.go`:
```go
func GetTemplateByName(name string) *UnitTemplate
```

**Impact:** Single source of truth, prevents copy-paste bugs.

---

### 3. Fix Parameter Typo

**File:** `common/ecsutil.go:198`

**Issue:** Parameter named `ecsmnager` instead of `manager`.

**Action:** Rename to `manager` for consistency.

**Impact:** Trivial fix, improves code quality.

---

### 4. Add String() Methods to Squad Enums

**Files:** `squads/squadcomponents.go`, `gui/guisquads/unitpurchasemode.go`

**Issue:** `getRoleName()` and `getTargetModeName()` helper functions in GUI duplicate enum-to-string logic.

**Action:** Add `String()` methods to `UnitRole` and `TargetMode` types in squadcomponents.go.

**Impact:** Idiomatic Go, eliminates GUI helpers, enables fmt.Printf("%v", role).

---

## Medium Priority

### 5. Remove Debug Printf Statements

**Files:** `gui/guisquads/squaddeploymentmode.go:174-244` (8+ statements), various others

**Issue:** Production code contains `fmt.Printf("DEBUG:...")` statements.

**Action:** Either remove or gate behind `DEBUG_MODE` constant from `game_main/config.go`.

**Impact:** Cleaner logs, slight performance improvement.

---

### 6. Deprecation Warning for FindEntityByIDWithTag

**File:** `common/ecsutil.go:280-301`

**Issue:** Function marked deprecated in comment but still used in `combat/factionmanager.go:61`, `combat/combat_testing_2.go:178`.

**Action:** Update callers to use recommended alternative, then remove deprecated function.

**Impact:** Reduces maintenance burden, eliminates technical debt.

---

### 7. Remove O(n) Fallback in GetCreatureAtPosition

**File:** `common/ecsutil.go:195-223`

**Issue:** Falls back from GlobalPositionSystem to O(n) entity iteration. Fallback may mask bugs.

**Action:** Remove fallback after verifying all entities register with PositionSystem. Add warning log if fallback would have triggered.

**Impact:** Catches missing registrations early, improves performance consistency.

---

### 8. Move Testing Code to _test.go Files

**File:** `combat/combat_testing_2.go` (178 lines)

**Issue:** Test scenarios in main package ship with production code.

**Action:** Move to `combat/combat_test.go` or `testing/` package.

**Impact:** Cleaner production builds, clearer test organization.

---

### 9. Standardize Dialog Creation

**File:** `gui/widgets/dialogs.go`

**Issue:** Three dialog functions (`CreateConfirmationDialog`, `CreateInputDialog`, `CreateMessageDialog`) duplicate 80% of setup code.

**Action:** Create single `CreateDialog(config DialogConfig)` with type/content/buttons in config.

**Impact:** Easier to add new dialog types, reduces ~60 lines of duplication.

---

## Low Priority

### 10. Standardize Enter/Exit Logging

**Files:** All mode files in `gui/guimodes/`, `gui/guisquads/`, `gui/guicombat/`

**Issue:** Every mode has `fmt.Println("Entering/Exiting X Mode")`.

**Action:** Move to BaseMode. Auto-print using `GetModeName()`. Modes override `OnEnter()`/`OnExit()` for custom logic.

**Impact:** Consistent logging, less boilerplate per mode.

---

### 11. Format TODO Comments

**Files:** 30+ TODO comments across codebase

**Issue:** TODOs lack priority/context: `// TODO: Implement`

**Action:** Standardize format: `// TODO(feature): description`

**Impact:** Better discoverability, clearer technical debt tracking.

---

### 12. Consistent GUIQueries Usage

**Files:** Various modes access ECS directly vs using `Queries` helper

**Issue:** Mixed patterns: `Context.ECSManager.World.Query()` vs `mode.Queries.GetX()`

**Action:** Prefer `Queries` for all GUI ECS access.

**Impact:** Maintains abstraction layer, easier to test/mock.

---

### 13. Remove/Document Unused GUI Components

**File:** `gui/guicomponents/guicomponents.go`

**Issue:** `PanelListComponent`, `ButtonListComponent`, `ColorLabelComponent` appear unused.

**Action:** Verify usage with grep. Remove if unused, document if used.

**Impact:** Reduces confusion, cleaner codebase.

---

## Not Recommended

These changes add complexity without proportional value:

- **Generic Component Accessor Consolidation** - Current type-safe accessors in `ecsutil.go` are clear despite similarity
- **Factory Abstraction for Simple Modes** - Current factories are appropriate; simpler modes don't need them
- **Logging Framework** - fmt.Println is fine for game development; logging framework is overkill
- **Interface Extraction** - Don't create interfaces (`RefreshableMode`, etc.) until 3+ implementations exist
- **O(1) Entity Lookup Cache** - Query-based ECS pattern is correct; caching adds complexity and sync bugs

---

## Implementation Order

Quick wins first, then progressively larger changes:

1. Fix typo (5 min)
2. Add String() to enums (15 min)
3. Template lookup utility (15 min)
4. setStatus/CommandHistory consolidation (30 min)
5. Remove debug printfs (30 min)
6. Move test code (30 min)
7. Dialog consolidation (1 hr)
8. Enter/Exit logging (1 hr)
9. Deprecation cleanup (1 hr)
10. O(n) fallback removal (1 hr)

---

## Related Documentation

- **GUI-specific details:** `analysis/gui_refactoring_recommendations.md`
- **ECS patterns:** `docs/ecs_best_practices.md`
- **Project guidelines:** `CLAUDE.md`
