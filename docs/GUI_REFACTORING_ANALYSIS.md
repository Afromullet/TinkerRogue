# GUI Refactoring Analysis

**Scope:** 76 Go files across 15 packages (~16,200 lines)
**Date:** 2026-02-25

---

## What's Working Well

### BaseMode + UIMode Interface
The `framework.UIMode` interface (8 methods) paired with `framework.BaseMode` (embedded struct) gives every mode a clean lifecycle (`Initialize`, `Enter`, `Exit`, `Update`, `Render`, `HandleInput`) while eliminating boilerplate. Modes embed `BaseMode` and override only what they need. This is the backbone of the GUI and it works.

### Panel Registry System
Declarative panel registration via `framework.RegisterPanel()` in `init()` functions. Each panel is a self-contained `PanelDescriptor` with spec name, content type, and `OnCreate` callback. Modes call `BuildPanels(panelTypes...)` and get type-safe widget access via `GetTextLabel()` / `GetPanelContainer()`. Panel definitions are centralized per mode in `*_panels_registry.go` files. This is clean and scalable.

### State Separation
UI state lives in `TacticalState` / `OverworldState` (selection indices, mode flags, camera). Game state lives in ECS components. The two never bleed into each other.

### Dependency Injection via *Deps Structs
Each mode and handler defines a `*Deps` struct consolidating all dependencies. `CombatModeDeps`, `OverworldModeDeps`, `SpellPanelDeps`, etc. One parameter instead of six. Easy to test, self-documenting.

### Package-per-Mode Organization
Each GUI mode gets its own package: `guicombat/`, `guisquads/`, `guioverworld/`, `guiraid/`, etc. Shared infrastructure lives in `framework/`, `builders/`, `specs/`, `widgets/`. Clear separation of concerns.

### specs/ Package
Responsive layout constants as percentages (`PanelWidthStandard = 0.2`, `PaddingTight = 0.015`, etc.) with mode-specific layout groups. `LayoutConfig` handles screen-size-aware calculations.

### GUIQueries Abstraction Layer
`framework.GUIQueries` wraps ECS access with event-driven caching. Returns pre-built `SquadInfo` / `FactionInfo` structs containing exactly what the UI needs. Squad cache invalidation is event-driven, avoiding per-frame re-queries.

---

## Issues

### 1. Squad Navigation Duplication - Resolved

**Priority: Medium** | **Effort: Low-Medium**

`currentSquadIndex` + `allSquadIDs` + `currentSquadID()` + `syncSquadOrderFromRoster()` are duplicated between `SquadEditorMode` and `ArtifactMode` (both in `guisquads/`). ArtifactMode additionally has `cycleSquad(delta)`, `updateSquadCounter()`, and `updateNavigationButtons()`.

A third variant exists in `CombatMode` (`combat_action_handler.go:CycleSquadSelection`) using a dynamic query instead of a stored array, which is correct for combat where squads can die.

Meanwhile, `CommanderSelector` in `commanderselector.go` already implements the exact index+array+cycle pattern as a reusable helper, but only for commanders.

**Recommendation:** Extract a generic `EntityNavigator` (or rename/generalize `CommanderSelector`) with `CurrentIdx`, `AllIDs`, `CurrentID()`, `Cycle(delta)`, `Sync(ids)`. Use it in both `SquadEditorMode` and `ArtifactMode`. Leave `CombatMode`'s dynamic approach as-is since it has different requirements.

**Files affected:** `squadeditormode.go`, `artifactmode.go`, `artifact_refresh.go`, `squadeditor_refresh.go`, `commanderselector.go`

---

### 2. Action Bar Duplication

**Priority: Medium** | **Effort: Low**

`buildContextActions()` and `buildNavigationActions()` in `SquadEditorMode` and `CombatMode` use identical spacing/anchor/button-group construction:

```
spacing := int(float64(layout.ScreenWidth) * specs.PaddingTight)
bottomPad := int(float64(layout.ScreenHeight) * specs.BottomButtonOffset)
leftPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
anchorLayout := builders.AnchorStartEnd(leftPad, bottomPad)
// CreateButtonGroup(...)
```

A `CreateBottomActionBar()` helper already exists in `builders/layout.go` and is used by `SquadDeploymentMode` and `UnitPurchaseMode` -- but `SquadEditorMode`, `CombatMode`, and `ExplorationMode` still do it manually.

**Recommendation:** Add `BuildContextActionBar(layout, buttons)` (left-anchored) and `BuildNavigationActionBar(layout, buttons)` (right-anchored) to `builders/layout.go`. Migrate the 5 manual implementations to use them.

**Files affected:** `squadeditormode.go`, `combatmode.go`, `artifactmode.go`, `exploration_panels_registry.go`, `builders/layout.go`

---

### 3. Hardcoded Spacing and Padding

**Priority: Low-Medium** | **Effort: Low**

34 `Spacing(N)` calls across 13 files using 7 different values (2, 3, 5, 8, 10, 15, 20). `Spacing(5)` accounts for 47% of calls. 8 `NewInsetsSimple(N)` calls using 3 values (5, 10, 40). The `specs/` package defines responsive padding percentages but has **no spacing constants**.

The variety of values (especially 3, 8, and 40) suggests ad-hoc tuning rather than design system consistency.

**Recommendation:** Add spacing constants to `specs/layout.go`:
- `SpacingTight = 3` (compact lists)
- `SpacingNormal = 5` (default, covers 47% of usage)
- `SpacingComfortable = 10` (panel gaps)
- `SpacingWide = 15` (dialog/section separators)
- `SpacingExtraWide = 20` (major section gaps)

Replace hardcoded values incrementally. The outlier values (2, 8, 40) can stay as-is or be reviewed case-by-case.

**Files affected:** All `*_panels_registry.go` files, `builders/panels.go`, `builders/widgets.go`, `builders/dialogs.go`

---

### 4. Direct ECS Access in GUI Layer

**Priority: Low-Medium** | **Effort: Medium**

15 GUI files call `common.GetComponentType` / `common.GetComponentTypeByID` directly instead of routing through `GUIQueries`. The abstraction boundary is inconsistent -- some modes use `GUIQueries` for squad/faction data while reaching past it for grid positions, attributes, unit types, and overworld node data.

**Heaviest offenders:**
- `overworld_action_handler.go` -- 6+ direct ECS calls for positions and node data
- `squadeditor_grid.go` -- direct `GridPositionData` and `LeaderComponent` access
- `inspect_panel.go` -- duplicates grid rendering logic with direct ECS calls
- `unitviewmode.go` -- direct `UnitTypeData` and `Attributes` access

**Root cause:** GUIQueries was built around squad/faction/combat data. Domain-specific data (grid positions, unit types, overworld nodes) has no query wrapper.

**Recommendation:** Expand the query layer incrementally. Don't try to wrap everything at once. Start with the most-called patterns:
1. Grid/formation data queries (used by squad editor + inspect)
2. Overworld node queries (used by overworld action handler + formatters)
3. Unit detail queries (used by unit view mode)

**Files affected:** `guiqueries.go`, `overworld_action_handler.go`, `overworld_formatters.go`, `squadeditor_grid.go`, `inspect_panel.go`, `unitviewmode.go`, `deploy_panel.go`

---

### 5. Visibility Management Inconsistency

**Priority: Low** | **Effort: Medium**

38 visibility toggles across 14 files using 4 different approaches:

| Approach | Count | Description |
|----------|-------|-------------|
| Panel registry defaults | 15 | `.Visibility = Hide` in `init()` -- necessary, not a problem |
| Inline state toggles | 13 | Direct `.Visibility =` in mode logic -- scattered, hard to trace |
| SubMenuController | 4+ | Framework-managed mutual exclusion -- cleanest pattern |
| Panel controller methods | 6 | `Show()`/`Hide()` wrappers delegating to SubMenuController or direct set |

The 15 registry defaults are fine. The problem is the remaining 23 toggles split across three patterns with no consistency about when to use which.

**Recommendation:** This isn't urgent. The SubMenuController pattern works well for mutually exclusive panels. For simple show/hide (attack pattern toggle, tab switching), inline is acceptable. The main action item is to avoid mixing approaches within a single mode -- pick one per mode and stay consistent.

---

### 6. Large Files Need Splitting

**Priority: Medium** | **Effort: Medium**

Files over 500 lines in `gui/`:

| Lines | File | Issue |
|-------|------|-------|
| 655 | `combat_input_handler.go` | Handles all combat input in one file |
| 564 | `combatmode.go` | Mode setup + orchestration + action bar building |
| 556 | `builders/panels.go` | All panel builder utilities |
| 549 | `combat_animation_mode.go` | Animation sequencing + rendering |

`combat_input_handler.go` is the worst offender. It processes keyboard input for movement, attack, spell casting, artifact activation, debug commands, and threat visualization all in one file.

**Recommendation:**
- Split `combat_input_handler.go` by input domain: movement input, attack input, spell/artifact input, debug input
- Extract action bar building from `combatmode.go` (it already has `combat_action_handler.go` -- the bar construction should follow)
- `builders/panels.go` and `combat_animation_mode.go` are borderline; split only if they keep growing

---

### 7. Missing ModalPanelController Interface

**Priority: Low** | **Effort: Low**

`SpellPanelController`, `ArtifactPanelController`, and `InspectPanelController` all serve the same role in `CombatMode` -- toggleable side panels with show/hide behavior. But they share no common interface.

Spell and Artifact controllers are nearly identical: both have `Show()`, `Hide()`, `Toggle()`, `Refresh()`, `SetWidgets()`, `Handler()`, a list widget, a detail area, and a selected item field. Inspect is simpler (no list/handler) but still implements `Show()`/`Hide()`.

`CombatMode` manages all three as separate typed fields with no polymorphism.

**Recommendation:** Define a minimal interface:

```go
type ModalPanel interface {
    Show()
    Hide()
}
```

This would let `CombatMode` manage panels uniformly for operations like "hide all panels" or "is any panel open". Don't over-abstract -- Spell and Artifact have domain-specific methods (`OnCastClicked`, `OnActivateClicked`) that shouldn't be on the interface.

**Files affected:** `spell_panel.go`, `artifact_panel.go`, `inspect_panel.go`, `combatmode.go`

---

### 8. CombatMode Orchestration Bloat

**Priority: Medium** | **Effort: High**

`CombatMode` has 14+ fields managing 7 subsystems: action handler, input handler, spell panel, artifact panel, visualization, turn flow, and sub-menu controller. It acts as an orchestrator rather than a simple UI mode.

The current decomposition (separate handler files, panel controllers in their own packages) is already good. The issue is that `CombatMode` is the single point wiring everything together, making initialization complex and the struct hard to reason about.

**Recommendation:** This is not critical to fix now. The existing decomposition into handlers and controllers already keeps individual files manageable. If it grows further, consider:
1. Grouping related fields into sub-structs (e.g., `panelGroup { spell, artifact, inspect }`)
2. Moving panel wiring into a dedicated `combat_panels_setup.go`
3. Using the `ModalPanel` interface (issue #7) to reduce field count

---

## Priority Summary

| # | Issue | Priority | Effort | Impact |
|---|-------|----------|--------|--------|
| 2 | Action bar duplication | Medium | Low | Quick win, 5 files |
| 1 | Squad navigation duplication | Medium | Low-Med | Eliminates copy-paste bugs |
| 6 | Large files | Medium | Medium | Readability and maintainability |
| 3 | Hardcoded spacing | Low-Med | Low | Consistency, easy to do incrementally |
| 4 | Direct ECS access | Low-Med | Medium | Cleaner abstraction boundary |
| 7 | Missing ModalPanel interface | Low | Low | Small code quality improvement |
| 8 | CombatMode bloat | Medium | High | Not urgent, already decomposed |
| 5 | Visibility inconsistency | Low | Medium | Not causing real problems |

**Suggested order:** Start with #2 (action bars) and #1 (squad navigation) as quick wins, then #6 (file splitting) for maintainability. The rest can be done incrementally as those areas are touched for other reasons.
