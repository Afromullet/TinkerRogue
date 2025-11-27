# GUI Refactoring Opportunities

**Last Updated:** 2025-11-27

This document lists practical refactoring opportunities for the GUI codebase that reduce duplication, improve maintainability, and make adding new functionality easier. Each suggestion is based on actual code analysis and provides real value.

---

## 1. Extract Viewport Coordinate Conversion Helper

**Location:** `ExplorationMode.HandleInput()`, `CombatInputHandler.handleMovementClick()`, `CombatInputHandler.handleSquadClick()`, `SquadDeploymentMode.Update()`

**Problem:** The same viewport coordinate conversion pattern is repeated 3+ times:
```
manager := coords.NewCoordinateManager(graphics.ScreenInfo)
viewport := coords.NewViewport(manager, playerPos)
logicalPos := viewport.ScreenToLogical(mouseX, mouseY)
```

**Value:**
- Reduces code duplication
- Prevents subtle bugs from inconsistent viewport calculations
- Single source of truth for coordinate conversion logic
- Possible performance improvement (could cache viewport per position)

**Suggested Implementation:** Create a `ViewportHelper` utility in `guicomponents` package with a method like `ConvertMouseToLogicalPosition(mouseX, mouseY, playerPos)` that all modes use.

---

## 2. Unify Grid Editor Components

**Location:** `SquadBuilderMode` (uses `GridEditorManager` + `GridCellButton` wrapper) vs `FormationEditorMode` (direct `[3][3]*widget.Button` array)

**Problem:** Two modes manage 3x3 grids with similar but divergent implementations:
- SquadBuilderMode: Complex `GridEditorManager` with separate cell metadata wrapper
- FormationEditorMode: Direct widget array with inline cell update logic
- No shared code despite identical UI pattern

**Value:**
- Eliminates duplicate grid management logic
- Ensures consistent grid behavior across modes
- Single place to fix grid interaction bugs
- Easier to add new grid-based features

**Suggested Implementation:** Create a unified `GridEditorComponent` in `guicomponents` that:
- Manages cell state and relationships (leader cells, secondary cells for multi-unit formations)
- Provides cell update callbacks
- Handles both formation editing and unit selection patterns
- Both modes would instantiate and use this component

---

## 3. Create Base UI Factory for Common Patterns

**Location:** `CombatUIFactory` and `SquadBuilderUIFactory`

**Problem:** Both factories follow the same pattern independently:
- Wrap `PanelBuilders` and layout configuration
- Implement methods for creating panels with specific content
- Duplicate panel creation logic (title, borders, content layout)

**Value:**
- Reduces duplication when adding new UI modes
- Ensures consistent panel styling across all modes
- Makes factory methods more discoverable
- Easier to change factory behavior globally

**Suggested Implementation:** Create a `BaseUIFactory` that both `CombatUIFactory` and `SquadBuilderUIFactory` extend, providing:
- Common panel creation methods
- Standardized title/border/styling logic
- Pattern for adding mode-specific panels

---

## 4. Consolidate Squad List Building Pattern

**Location:** `SquadDeploymentMode.buildSquadListPanel()`, `FormationEditorMode.buildSquadSelector()`, `SquadManagementMode` navigation logic

**Problem:** Multiple modes build squad lists independently:
- `SquadDeploymentMode` and `CombatMode` use `SquadListComponent` (correct pattern)
- `FormationEditorMode` builds string-based list, loses EntityID associations
- `SquadManagementMode` uses navigation pattern instead of list

**Value:**
- Consistent squad selection across all modes
- Less code in each mode
- Easier to add filtering/sorting to squad selection
- Single place to maintain squad queries

**Suggested Implementation:** All modes should use `SquadListComponent` from `guicomponents` for squad selection, instead of building lists independently.

---

## 5. Standardize Input Handler Pattern

**Location:** All mode `HandleInput()` implementations

**Problem:** Different modes use different input handling patterns:
- `CombatMode`: Delegates to `CombatInputHandler` (good pattern)
- `ExplorationMode`: Inline click handling mixed with UI updates
- `SquadDeploymentMode`: Deferred click processing stored in state
- `FormationEditorMode`: Inline cell click handlers

**Value:**
- Easier to add new modes with predictable input patterns
- Easier to understand input flow across codebase
- Single pattern for testing input handlers

**Suggested Implementation:** Create an `InputHandler` interface and have modes that do complex input processing delegate to a handler class, following the CombatInputHandler pattern as the template.

---

## 6. Cache Viewport/Coordinate System in Rendering Pipeline

**Location:** `MovementTileRenderer`, `SquadHighlightRenderer`, combat input handlers

**Problem:** Every render call and input handler independently creates:
```
manager := coords.NewCoordinateManager(graphics.ScreenInfo)
viewport := coords.NewViewport(manager, playerPos)
```

**Value:**
- Reduces coordinate system allocations during high-frequency rendering
- Performance improvement in rendering-heavy scenarios
- Single coordinate system used consistently

**Suggested Implementation:** Pass viewport as a parameter through the rendering pipeline, or cache it in a context object that renderers and handlers reference.

---

## 7. Consolidate Detail Panel Formatters

**Location:** `DetailPanelComponent`, `CombatUIFactory.GetFormattedSquadDetails()`, individual mode formatting

**Problem:** Squad and entity detail formatting is scattered:
- `DefaultSquadFormatter` in guicomponents
- `CombatUIFactory` duplicates squad formatting
- Other modes implement formatting inline
- No consistent place to change how details are displayed

**Value:**
- Single source of truth for entity detail formatting
- Easier to change detail display format globally
- Reduces code in individual modes

**Suggested Implementation:** Create a `Formatters` module in `guicomponents` that centralizes all detail formatting functions (`FormatSquadDetails()`, `FormatFactionDetails()`, etc.).

---

## 8. Extract Common Mode Initialization Patterns

**Location:** Mode `Initialize()` methods, particularly component setup

**Problem:** Many modes have repetitive initialization patterns:
- Panel creation with identical structure
- Component setup for common UI elements (detail panels, stats displays)
- Hotkey registration boilerplate

**Value:**
- Less boilerplate in new modes
- Consistent initialization across modes
- Easier to add new common components

**Suggested Implementation:** Create helper functions in `basemode.go` or `modehelpers.go` for common initialization patterns (e.g., `InitializeDetailPanel()`, `RegisterCommonHotkeys()`).

---

## 9. Remove Deprecated UI Factory Method

**Location:** `SquadBuilderUIFactory.CreatePalettePanel()` (lines 79-108)

**Problem:** Method is marked as deprecated with comment "kept for compatibility" but `CreateRosterPalettePanel()` is the current replacement.

**Value:**
- Reduces confusion about which method to use
- Prevents accidental use of outdated pattern
- Slightly simpler codebase

**Suggested Implementation:** Remove `CreatePalettePanel()` entirely and verify no code calls it.

---

## Implementation Priority

**High Value / Lower Effort:**
1. Extract viewport coordinate helper
2. Consolidate squad list building to use `SquadListComponent`
3. Consolidate detail panel formatters
4. Remove deprecated factory method

**Medium Value / Medium Effort:**
5. Standardize input handler pattern
6. Cache viewport in rendering pipeline
7. Extract common mode initialization patterns

**Medium Value / Higher Effort:**
8. Unify grid editor components
9. Create base UI factory

---

## Notes

- These suggestions focus on real code duplication and maintainability issues, not theoretical improvements
- The current architecture is generally sound; these are incremental improvements that reduce friction
- Estimated total benefit: 20-30% reduction in GUI-related code duplication, easier addition of new modes
