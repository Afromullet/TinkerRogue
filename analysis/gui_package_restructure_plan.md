# GUI Package Restructuring Implementation Plan

**Generated:** 2025-11-16
**Objective:** Reorganize flat 30-file `gui/` package into logical subpackages without breaking functionality

---

## EXECUTIVE SUMMARY

### Current State
- **30 files** in flat `gui/` package structure
- Clear functional boundaries but poor discoverability
- All files in `package gui` namespace - no internal organization

### Target State
```
gui/
├── core/            # Core infrastructure (4 files)
├── modes/           # UI mode implementations (8 files)
├── factories/       # UI factory pattern implementations (3 files)
├── components/      # Reusable UI components (4 files)
├── resources/       # Shared resources and constants (3 files)
└── helpers/         # Helper utilities (3 files)
```

### Migration Strategy
**Phased approach with backward compatibility:**
1. **Phase 1:** Move resources (zero dependencies) - 30 minutes
2. **Phase 2:** Move components and helpers - 45 minutes
3. **Phase 3:** Move factories - 30 minutes
4. **Phase 4:** Move modes - 1 hour
5. **Phase 5:** Cleanup and verification - 30 minutes

**Total Effort:** 3-4 hours

### Key Constraints
- **Minimize import cycles:** Keep core types in parent `gui/` package
- **Backward compatibility:** Exported types remain accessible
- **No functional changes:** Pure reorganization, no refactoring

---

## DETAILED FILE MIGRATION PLAN

### Phase 1: Resources (LOWEST RISK - No Dependencies)

**Target:** `gui/resources/`

| File | LOC | Dependencies | Export Requirements |
|------|-----|--------------|---------------------|
| `guiresources.go` | 307 | None (loads assets) | All exports stay public |
| `layout_constants.go` | 130 | None | All constants stay public |
| `panelconfig.go` | 279 | Only ebitenui/widget | `StandardPanels`, `PanelSpec` |

**Migration:**
```bash
mkdir gui/resources
mv gui/guiresources.go gui/resources/
mv gui/layout_constants.go gui/resources/
mv gui/panelconfig.go gui/resources/
```

**Import Changes:**
```go
// Before
package gui

// After (in resources/*.go)
package resources

// Users import as:
import "game_main/gui/resources"
```

**Re-exports in `gui/` (for backward compatibility):**
```go
// gui/resources.go (NEW FILE)
package gui

import "game_main/gui/resources"

// Re-export commonly used resources
var (
    SmallFace      = resources.SmallFace
    LargeFace      = resources.LargeFace
    PanelRes       = resources.PanelRes
    ListRes        = resources.ListRes
    TextAreaRes    = resources.TextAreaRes
    StandardPanels = resources.StandardPanels
)

// Re-export constants
const (
    PanelWidthNarrow    = resources.PanelWidthNarrow
    PanelWidthStandard  = resources.PanelWidthStandard
    PanelWidthMedium    = resources.PanelWidthMedium
    PanelWidthWide      = resources.PanelWidthWide
    PanelWidthExtraWide = resources.PanelWidthExtraWide

    PanelHeightTiny    = resources.PanelHeightTiny
    PanelHeightSmall   = resources.PanelHeightSmall
    PanelHeightQuarter = resources.PanelHeightQuarter
    PanelHeightThird   = resources.PanelHeightThird
    PanelHeightHalf    = resources.PanelHeightHalf
    PanelHeightTall    = resources.PanelHeightTall
    PanelHeightFull    = resources.PanelHeightFull

    PaddingTight    = resources.PaddingTight
    PaddingStandard = resources.PaddingStandard
    PaddingLoose    = resources.PaddingLoose
)

// Re-export types
type PanelSpec = resources.PanelSpec
```

**Validation:**
```bash
go build -o game_main/game_main.exe game_main/*.go
go test ./gui/...
```

---

### Phase 2: Components & Helpers (LOW RISK)

#### 2a. Components Subpackage

**Target:** `gui/components/`

| File | LOC | Dependencies | Notes |
|------|-----|--------------|-------|
| `guicomponents.go` | 590 | `gear`, `common`, ECS | Contains 7 component types |
| `filter_helper.go` | 96 | None | Filter functions |
| `createwidgets.go` | 346 | resources, ebitenui | Widget factories |
| `guirenderers.go` | 161 | `coords`, `combat` | Rendering helpers |

**Migration:**
```bash
mkdir gui/components
mv gui/guicomponents.go gui/components/
mv gui/filter_helper.go gui/components/
mv gui/createwidgets.go gui/components/
mv gui/guirenderers.go gui/components/
```

**Package Declaration:**
```go
package components

import (
    "game_main/gui/resources"
    // ... other imports
)
```

**Key Exports (must remain public):**
- `SquadListComponent`
- `DetailPanelComponent`
- `TextDisplayComponent`
- `PanelListComponent`
- `ButtonListComponent`
- `ItemListComponent`
- `StatsDisplayComponent`
- `ColorLabelComponent`
- `CreateButtonWithConfig`, `CreateTextWithConfig`, `CreateTextAreaWithConfig`
- `MovementTileRenderer`, `SquadHighlightRenderer`

**Re-export in `gui/components.go` (NEW):**
```go
package gui

import "game_main/gui/components"

// Re-export component types
type (
    SquadListComponent   = components.SquadListComponent
    DetailPanelComponent = components.DetailPanelComponent
    TextDisplayComponent = components.TextDisplayComponent
    // ... etc
)

// Re-export widget factories
var (
    CreateButtonWithConfig   = components.CreateButtonWithConfig
    CreateTextWithConfig     = components.CreateTextWithConfig
    CreateTextAreaWithConfig = components.CreateTextAreaWithConfig
    // ... etc
)
```

#### 2b. Helpers Subpackage

**Target:** `gui/helpers/`

| File | LOC | Dependencies | Notes |
|------|-----|--------------|-------|
| `modehelpers.go` | 162 | None (gui types only) | 9 helper functions |
| `layout.go` | 54 | ebitenui/widget | Layout utilities |
| `panels.go` | 104 | ebitenui/widget | Panel builders |

**Migration:**
```bash
mkdir gui/helpers
mv gui/modehelpers.go gui/helpers/
mv gui/layout.go gui/helpers/
mv gui/panels.go gui/helpers/
```

**Package Declaration:**
```go
package helpers

import (
    "game_main/gui/resources"
    "github.com/ebitenui/ebitenui/widget"
)
```

**Key Exports:**
- `PanelBuilders`
- `LayoutConfig`
- `PanelOption`, `PanelOptionFunc`
- `CreateCloseButton`, `CreateBottomCenterButtonContainer`, etc.

**Re-export in `gui/helpers.go` (NEW):**
```go
package gui

import "game_main/gui/helpers"

type (
    PanelBuilders = helpers.PanelBuilders
    LayoutConfig  = helpers.LayoutConfig
    PanelOption   = helpers.PanelOption
)

var (
    CreateCloseButton                = helpers.CreateCloseButton
    CreateBottomCenterButtonContainer = helpers.CreateBottomCenterButtonContainer
    AddActionButton                   = helpers.AddActionButton
    // ... etc
)
```

---

### Phase 3: Factories (MEDIUM RISK - Used by Modes)

**Target:** `gui/factories/`

| File | LOC | Dependencies | Imports From |
|------|-----|--------------|--------------|
| `combat_ui_factory.go` | 128 | components, helpers | BaseMode |
| `panel_factory.go` | 194 | components, helpers | BaseMode |
| `squad_builder_ui_factory.go` | 206 | components, resources | `squads` |

**Migration:**
```bash
mkdir gui/factories
mv gui/combat_ui_factory.go gui/factories/
mv gui/panel_factory.go gui/factories/
mv gui/squad_builder_ui_factory.go gui/factories/
```

**Package Declaration:**
```go
package factories

import (
    "game_main/gui/components"
    "game_main/gui/helpers"
    "game_main/gui/resources"
    // ... game packages
)
```

**Challenge:** Factories reference `GUIQueries`, `PanelBuilders` which need to stay in parent package

**Solution:** Keep factories in `package gui` but in subdirectory

**REVISED Approach:**
```go
// Keep these as package gui but in factories/ subdirectory
// File: gui/factories/combat_ui_factory.go
package gui  // NOT package factories!

// No import cycle because they import from parent
```

**Alternative (Better):** Move to subpackage but pass interfaces
```go
// gui/factories/combat_ui_factory.go
package factories

import "game_main/gui" // Import parent for types

func NewCombatUIFactory(
    queries *gui.GUIQueries,
    builders *gui.PanelBuilders,
    layout *gui.LayoutConfig,
) *CombatUIFactory {
    // ...
}
```

**Chosen Approach:** Keep as `package gui` in subdirectory (avoids circular imports)

---

### Phase 4: Modes (HIGHEST COMPLEXITY)

**Target:** `gui/modes/`

| File | LOC | Dependencies | Embeds |
|------|-----|--------------|--------|
| `combatmode.go` | 348 | factories, components, `combat` | `BaseMode` |
| `explorationmode.go` | 203 | helpers, components | `BaseMode` |
| `inventorymode.go` | 236 | components, `gear` | `BaseMode` |
| `infomode.go` | 141 | helpers | `BaseMode` |
| `formationeditormode.go` | 120 | helpers | `BaseMode` |
| `squadmanagementmode.go` | 217 | components, `squads` | `BaseMode` |
| `squaddeploymentmode.go` | 299 | components, `squads` | `BaseMode` |
| `squadbuilder.go` | 375 | factories, `squads` | `BaseMode` |

**Challenge:** All modes embed `BaseMode` and implement `UIMode` interface

**Solution:** Keep core types in parent `gui/` package, move mode implementations

**Files to KEEP in `gui/`:**
- `uimode.go` - Interface definition
- `basemode.go` - Base implementation
- `modemanager.go` - Mode coordinator
- `guiqueries.go` - Shared query service

**Migration:**
```bash
mkdir gui/modes
mv gui/combatmode.go gui/modes/
mv gui/explorationmode.go gui/modes/
mv gui/inventorymode.go gui/modes/
mv gui/infomode.go gui/modes/
mv gui/formationeditormode.go gui/modes/
mv gui/squadmanagementmode.go gui/modes/
mv gui/squaddeploymentmode.go gui/modes/
mv gui/squaddeploymentmode.go gui/modes/
mv gui/squadbuilder.go gui/modes/
```

**Package Declaration - TWO OPTIONS:**

#### Option A: Subpackage with parent import (RECOMMENDED)
```go
// gui/modes/combatmode.go
package modes

import (
    "game_main/gui" // Import parent for BaseMode, UIMode
    "game_main/gui/components"
    "game_main/gui/helpers"
    // ...
)

type CombatMode struct {
    gui.BaseMode // Embed from parent
    // ...
}

var _ gui.UIMode = (*CombatMode)(nil) // Verify interface
```

**Pros:**
- Clear separation of concerns
- Easy to navigate
- Modes grouped together

**Cons:**
- Requires importing parent package (minor)
- Need re-exports for registration

#### Option B: Keep as `package gui` in subdirectory
```go
// gui/modes/combatmode.go
package gui

// No imports of parent needed
type CombatMode struct {
    BaseMode
    // ...
}
```

**Pros:**
- No import cycles possible
- Simpler imports
- Backward compatible

**Cons:**
- Less clear package boundaries
- Modes still in global `gui` namespace

**CHOSEN APPROACH:** Option A (subpackage) with factory re-exports

**Re-exports in `gui/modes.go` (NEW):**
```go
package gui

import "game_main/gui/modes"

// Re-export mode constructors for registration
func NewCombatMode(manager *UIModeManager) UIMode {
    return modes.NewCombatMode(manager)
}

func NewExplorationMode(manager *UIModeManager) UIMode {
    return modes.NewExplorationMode(manager)
}

// ... etc for all 8 modes
```

---

### Phase 5: Combat Subsystem (KEEP IN PARENT)

**Files to KEEP in `gui/`:**

| File | LOC | Reason |
|------|-----|--------|
| `combat_action_handler.go` | 264 | Tightly coupled to CombatMode |
| `combat_input_handler.go` | 163 | Tightly coupled to CombatMode |
| `combat_state_manager.go` | 130 | Tightly coupled to CombatMode |
| `combat_log_manager.go` | 72 | Tightly coupled to CombatMode |
| `squad_builder_grid_manager.go` | 252 | Tightly coupled to SquadBuilder |

**Rationale:**
- These are **support classes** for modes, not standalone components
- Moving them creates more complexity than clarity
- Could move to `modes/` subdirectory but keep as `package gui`

**Alternative:** Move to `modes/support/` as `package gui`
```
gui/modes/
  ├── support/
  │   ├── combat_action_handler.go
  │   ├── combat_input_handler.go
  │   ├── combat_state_manager.go
  │   ├── combat_log_manager.go
  │   └── squad_builder_grid_manager.go
  └── (mode files)
```

---

## FINAL PACKAGE STRUCTURE

```
gui/
├── resources/              # Package: resources
│   ├── guiresources.go    # Fonts, images, assets
│   ├── layout_constants.go # Size constants
│   └── panelconfig.go     # Panel specifications
│
├── components/            # Package: components
│   ├── guicomponents.go   # 7 component types
│   ├── filter_helper.go   # Filter functions
│   ├── createwidgets.go   # Widget factories
│   └── guirenderers.go    # Rendering helpers
│
├── helpers/               # Package: helpers
│   ├── modehelpers.go     # Helper functions
│   ├── layout.go          # Layout config
│   └── panels.go          # Panel builders
│
├── factories/             # Package: gui (in subdirectory)
│   ├── combat_ui_factory.go
│   ├── panel_factory.go
│   └── squad_builder_ui_factory.go
│
├── modes/                 # Package: modes
│   ├── combatmode.go
│   ├── explorationmode.go
│   ├── inventorymode.go
│   ├── infomode.go
│   ├── formationeditormode.go
│   ├── squadmanagementmode.go
│   ├── squaddeploymentmode.go
│   ├── squadbuilder.go
│   │
│   └── support/           # Package: gui (in subdirectory)
│       ├── combat_action_handler.go
│       ├── combat_input_handler.go
│       ├── combat_state_manager.go
│       ├── combat_log_manager.go
│       └── squad_builder_grid_manager.go
│
├── uimode.go              # Core: UIMode interface
├── basemode.go            # Core: BaseMode implementation
├── modemanager.go         # Core: UIModeManager
├── guiqueries.go          # Core: GUIQueries service
│
└── (re-export files)      # Backward compatibility
    ├── resources.go       # Re-exports from resources/
    ├── components.go      # Re-exports from components/
    ├── helpers.go         # Re-exports from helpers/
    └── modes.go           # Re-exports from modes/
```

---

## IMPORT STRATEGY

### No Circular Dependencies

**Dependency Graph:**
```
resources/ (no internal dependencies)
    ↑
helpers/ (imports resources/)
    ↑
components/ (imports resources/, helpers/)
    ↑
gui/ (core types: UIMode, BaseMode, GUIQueries)
    ↑
factories/ (package gui, imports components/, helpers/)
    ↑
modes/ (imports gui/, components/, helpers/, factories/)
```

**Key Insight:** Core types (`UIMode`, `BaseMode`, `GUIQueries`) stay in parent `gui/` package to prevent circular imports

### Import Examples

**Mode importing components:**
```go
// gui/modes/combatmode.go
package modes

import (
    "game_main/gui"               // For BaseMode, UIMode, GUIQueries
    "game_main/gui/components"    // For SquadListComponent
    "game_main/gui/helpers"       // For PanelBuilders
)

type CombatMode struct {
    gui.BaseMode
    squadListComponent *components.SquadListComponent
}
```

**Component importing resources:**
```go
// gui/components/createwidgets.go
package components

import (
    "game_main/gui/resources"
)

func CreateButtonWithConfig(config ButtonConfig) *widget.Button {
    return widget.NewButton(
        widget.ButtonOpts.Image(resources.ButtonImage),
        widget.ButtonOpts.TextFace(resources.SmallFace),
        // ...
    )
}
```

**Game code using GUI (backward compatible):**
```go
// game_main/main.go
package main

import "game_main/gui"

func main() {
    modeManager := gui.NewUIModeManager(ctx)

    // These still work via re-exports
    combatMode := gui.NewCombatMode(modeManager)
    explorationMode := gui.NewExplorationMode(modeManager)

    modeManager.RegisterMode(combatMode)
    modeManager.RegisterMode(explorationMode)
}
```

---

## EXPORT/VISIBILITY STRATEGY

### Public Exports (Stay Public)

**Core Types (in `gui/`):**
- `UIMode`, `UIContext`, `InputState`, `ModeTransition`
- `BaseMode`, `UIModeManager`
- `GUIQueries`, `SquadInfo`, `FactionInfo`

**Resources (via `gui/resources.go` re-export):**
- `SmallFace`, `LargeFace`
- `PanelRes`, `ListRes`, `TextAreaRes`
- All `PanelWidth*`, `PanelHeight*`, `Padding*` constants
- `StandardPanels`, `PanelSpec`

**Components (via `gui/components.go` re-export):**
- All 7 component types
- All widget factory functions

**Helpers (via `gui/helpers.go` re-export):**
- `PanelBuilders`, `LayoutConfig`
- All helper functions

**Modes (via `gui/modes.go` re-export):**
- Constructor functions for all 8 modes

### Internal/Package-Private

**NEW package-private types:**
- Internal factory helpers
- Internal component state
- Private rendering details

**Strategy:** Don't change visibility during restructure - maintain current public API

---

## TESTING STRATEGY

### Pre-Migration Baseline
```bash
# 1. Record current test results
go test ./gui/... -v > baseline_tests.txt

# 2. Record build success
go build -o game_main/game_main.exe game_main/*.go

# 3. Run game smoke test
# Manual: Launch game, test each mode transition
```

### Post-Phase Validation

**After Each Phase:**
```bash
# 1. Build test
go build -o game_main/game_main.exe game_main/*.go

# 2. Package tests
go test ./gui/... -v

# 3. Import verification
go list -f '{{.ImportPath}}: {{.Imports}}' ./gui/...

# 4. Circular dependency check
go list -f '{{if .Incomplete}}{{.ImportPath}}: {{.Error}}{{end}}' ./...
```

### Integration Testing Checklist

After full migration:
- [ ] Game launches without errors
- [ ] Exploration mode renders correctly
- [ ] Combat mode transition works
- [ ] Squad management mode accessible
- [ ] Squad builder mode functional
- [ ] Formation editor mode works
- [ ] Info mode displays correctly
- [ ] Inventory mode opens
- [ ] All hotkeys work (E, C, I, Tab, etc.)
- [ ] ESC returns to correct modes
- [ ] No console errors during mode transitions

### Automated Test Coverage

**Create new test file:** `gui/integration_test.go`
```go
package gui_test

import (
    "testing"
    "game_main/gui"
)

func TestPackageStructure(t *testing.T) {
    // Verify core types accessible
    var _ gui.UIMode
    var _ gui.BaseMode
    var _ gui.UIModeManager

    // Verify re-exports work
    _ = gui.SmallFace
    _ = gui.StandardPanels
    _ = gui.CreateButtonWithConfig
}

func TestModeConstructors(t *testing.T) {
    // Verify all modes can be constructed
    // (requires mock context)
}
```

---

## RISK ASSESSMENT & MITIGATION

### High-Risk Areas

#### 1. **Circular Import Cycles**

**Risk:** Modes importing components importing helpers importing modes
**Probability:** Medium
**Impact:** High (build failure)

**Mitigation:**
- Keep core types (`BaseMode`, `UIMode`, `GUIQueries`) in parent `gui/`
- Factories stay as `package gui` in subdirectory
- Test imports after each phase
- Use `go list` to detect cycles immediately

**Detection:**
```bash
go list -f '{{.ImportPath}}: {{.Imports}}' ./gui/... | grep circular
```

#### 2. **Breaking External Code**

**Risk:** Code outside `gui/` package breaks due to missing imports
**Probability:** Medium
**Impact:** High (game doesn't build)

**Mitigation:**
- Re-export all public types in parent package
- Test `game_main/` build after each phase
- Search for direct type references:
```bash
grep -r "gui\\.CombatMode" . --include="*.go" --exclude-dir=gui
```

**Rollback Plan:**
- Each phase is independently reversible
- Git commit after each successful phase
- Keep re-export files until full migration validated

#### 3. **Type Assertion Failures**

**Risk:** Interface implementations break due to package changes
**Probability:** Low
**Impact:** Medium (runtime panics)

**Mitigation:**
- Compile-time interface checks:
```go
var _ gui.UIMode = (*modes.CombatMode)(nil)
```
- Test mode registration in integration tests

#### 4. **Resource Path Breakage**

**Risk:** Asset loading fails if paths assumed to be relative to `gui/`
**Probability:** Low
**Impact:** Medium (runtime errors)

**Mitigation:**
- Review `guiresources.go` for relative paths
- Test asset loading in each mode
- Paths are currently `../assets/...` (relative to executable, not package)

### Medium-Risk Areas

#### 5. **Incomplete Re-exports**

**Risk:** Forgetting to re-export a used type/function
**Probability:** Medium
**Impact:** Medium (build errors)

**Mitigation:**
- Grep for all `gui.` references in codebase
- Compile after each re-export file created
- Systematic review of each moved file's exports

#### 6. **Test File Package Declarations**

**Risk:** Test files in wrong package namespace
**Probability:** Low
**Impact:** Low (test failures)

**Mitigation:**
- Search for `*_test.go` files: `find gui -name "*_test.go"`
- Update package declarations to match new structure
- Run tests after each phase

### Low-Risk Areas

#### 7. **Documentation Drift**

**Risk:** Comments/docs reference old package structure
**Probability:** High
**Impact:** Low (confusion)

**Mitigation:**
- Update CLAUDE.md after migration
- Add package documentation to each new subpackage
- No immediate fix needed - address in documentation pass

---

## STEP-BY-STEP IMPLEMENTATION

### Phase 1: Resources (30 minutes)

**Step 1.1:** Create resources subpackage
```bash
cd C:/Users/Afromullet/Desktop/TinkerRogue
mkdir gui/resources
```

**Step 1.2:** Move files
```bash
mv gui/guiresources.go gui/resources/
mv gui/layout_constants.go gui/resources/
mv gui/panelconfig.go gui/resources/
```

**Step 1.3:** Update package declarations
```bash
# In each moved file, change:
# package gui → package resources
```

**Step 1.4:** Create re-export file
```bash
# Create gui/resources.go with re-exports (see detailed structure above)
```

**Step 1.5:** Validate
```bash
go build -o game_main/game_main.exe game_main/*.go
go test ./gui/...
```

**Step 1.6:** Commit
```bash
git add gui/resources gui/resources.go
git commit -m "gui: Move resources to gui/resources/ subpackage"
```

---

### Phase 2: Components (45 minutes)

**Step 2.1:** Create components subpackage
```bash
mkdir gui/components
```

**Step 2.2:** Move files
```bash
mv gui/guicomponents.go gui/components/
mv gui/filter_helper.go gui/components/
mv gui/createwidgets.go gui/components/
mv gui/guirenderers.go gui/components/
```

**Step 2.3:** Update package declarations
```bash
# In each moved file:
# package gui → package components
```

**Step 2.4:** Update imports in moved files
```bash
# Change all references:
# SmallFace → resources.SmallFace
# PanelRes → resources.PanelRes
# Add: import "game_main/gui/resources"
```

**Step 2.5:** Create re-export file
```bash
# Create gui/components.go with re-exports
```

**Step 2.6:** Validate
```bash
go build -o game_main/game_main.exe game_main/*.go
go test ./gui/...
```

**Step 2.7:** Commit
```bash
git add gui/components gui/components.go
git commit -m "gui: Move components to gui/components/ subpackage"
```

---

### Phase 3: Helpers (30 minutes)

**Step 3.1:** Create helpers subpackage
```bash
mkdir gui/helpers
```

**Step 3.2:** Move files
```bash
mv gui/modehelpers.go gui/helpers/
mv gui/layout.go gui/helpers/
mv gui/panels.go gui/helpers/
```

**Step 3.3:** Update package declarations
```bash
# package gui → package helpers
```

**Step 3.4:** Update imports
```bash
# Add imports for resources, components as needed
```

**Step 3.5:** Create re-export file
```bash
# Create gui/helpers.go
```

**Step 3.6:** Validate & commit
```bash
go build -o game_main/game_main.exe game_main/*.go
go test ./gui/...
git add gui/helpers gui/helpers.go
git commit -m "gui: Move helpers to gui/helpers/ subpackage"
```

---

### Phase 4: Factories (30 minutes)

**Step 4.1:** Create factories subdirectory
```bash
mkdir gui/factories
```

**Step 4.2:** Move files (KEEP AS PACKAGE GUI)
```bash
mv gui/combat_ui_factory.go gui/factories/
mv gui/panel_factory.go gui/factories/
mv gui/squad_builder_ui_factory.go gui/factories/
```

**Step 4.3:** Update imports in factory files
```bash
# Add imports:
# "game_main/gui/components"
# "game_main/gui/helpers"
# "game_main/gui/resources"
```

**Step 4.4:** DO NOT change package declaration
```go
// Keep as: package gui
// No re-export needed - still in gui namespace
```

**Step 4.5:** Validate & commit
```bash
go build -o game_main/game_main.exe game_main/*.go
git add gui/factories
git commit -m "gui: Move factories to gui/factories/ subdirectory"
```

---

### Phase 5: Modes (1 hour)

**Step 5.1:** Create modes subdirectory and support subdirectory
```bash
mkdir gui/modes
mkdir gui/modes/support
```

**Step 5.2:** Move mode files
```bash
mv gui/combatmode.go gui/modes/
mv gui/explorationmode.go gui/modes/
mv gui/inventorymode.go gui/modes/
mv gui/infomode.go gui/modes/
mv gui/formationeditormode.go gui/modes/
mv gui/squadmanagementmode.go gui/modes/
mv gui/squaddeploymentmode.go gui/modes/
mv gui/squadbuilder.go gui/modes/
```

**Step 5.3:** Move support files (KEEP AS PACKAGE GUI)
```bash
mv gui/combat_action_handler.go gui/modes/support/
mv gui/combat_input_handler.go gui/modes/support/
mv gui/combat_state_manager.go gui/modes/support/
mv gui/combat_log_manager.go gui/modes/support/
mv gui/squad_builder_grid_manager.go gui/modes/support/
```

**Step 5.4:** Update mode package declarations
```bash
# In gui/modes/*.go:
# package gui → package modes
```

**Step 5.5:** Update mode imports
```bash
# Add to each mode file:
import (
    "game_main/gui"               // For BaseMode, UIMode, etc.
    "game_main/gui/components"
    "game_main/gui/helpers"
    // ... existing imports
)

# Change references:
# BaseMode → gui.BaseMode
# UIMode → gui.UIMode
# GUIQueries → gui.GUIQueries
# CreateButtonWithConfig → components.CreateButtonWithConfig
# PanelBuilders → helpers.PanelBuilders
```

**Step 5.6:** Keep support files as package gui
```bash
# In gui/modes/support/*.go:
# Keep: package gui
# Update imports to use gui/components, gui/helpers
```

**Step 5.7:** Create mode re-export file
```bash
# Create gui/modes.go with constructor re-exports
```

**Step 5.8:** Update mode constructors to return gui.UIMode
```go
// In each mode file:
func NewCombatMode(manager *gui.UIModeManager) gui.UIMode {
    return &CombatMode{
        BaseMode: gui.BaseMode{
            // ...
        },
    }
}
```

**Step 5.9:** Validate & commit
```bash
go build -o game_main/game_main.exe game_main/*.go
go test ./gui/...
git add gui/modes gui/modes.go
git commit -m "gui: Move modes to gui/modes/ subpackage with support/"
```

---

### Phase 6: Final Verification (30 minutes)

**Step 6.1:** Full build test
```bash
go build -o game_main/game_main.exe game_main/*.go
```

**Step 6.2:** Run all tests
```bash
go test ./... -v
```

**Step 6.3:** Check for circular imports
```bash
go list -f '{{.ImportPath}}: {{.Imports}}' ./gui/...
```

**Step 6.4:** Manual integration test
- Launch game
- Test each mode transition
- Verify no console errors
- Test hotkeys

**Step 6.5:** Update documentation
```bash
# Update CLAUDE.md with new structure
# Add package docs to each subpackage
```

**Step 6.6:** Final commit
```bash
git add -A
git commit -m "gui: Complete package restructure - 30 files organized into subpackages

- gui/resources/: Layout constants, assets, panel specs
- gui/components/: Reusable UI components (7 types)
- gui/helpers/: Helper functions and panel builders
- gui/factories/: UI factory implementations (package gui)
- gui/modes/: 8 mode implementations (package modes)
- gui/modes/support/: Mode support classes (package gui)
- Core types remain in gui/ for import simplicity
- Backward compatibility via re-exports"
```

---

## EFFORT ESTIMATE

### Time Breakdown

| Phase | Task | Time | Complexity |
|-------|------|------|------------|
| 1 | Resources migration | 30 min | Low |
| 2 | Components migration | 45 min | Medium |
| 3 | Helpers migration | 30 min | Low |
| 4 | Factories migration | 30 min | Medium |
| 5 | Modes migration | 60 min | High |
| 6 | Verification & docs | 30 min | Low |
| **Total** | **Full restructure** | **3-4 hours** | **Medium** |

### Complexity Factors

**Low Complexity:**
- No external dependencies (resources)
- Clear boundaries
- Simple re-exports

**Medium Complexity:**
- Internal dependencies
- Multiple import updates
- Package declaration changes

**High Complexity:**
- BaseMode embedding
- Interface implementations
- Factory coupling
- Support class organization

### Parallelization Potential

**Can be parallelized:**
- Phase 1 (resources) is independent
- Phase 2 (components) after Phase 1
- Phase 3 (helpers) after Phase 1

**Must be sequential:**
- Phase 4 (factories) after Phases 2-3
- Phase 5 (modes) after Phase 4
- Phase 6 (verification) after all

**Realistic Timeline:**
- **Single developer:** 3-4 hours (sequential)
- **With testing breaks:** 4-5 hours
- **Including documentation:** 5-6 hours

---

## GO PACKAGING BEST PRACTICES APPLIED

### 1. **Clear Package Names**
✅ `resources`, `components`, `helpers`, `modes`, `factories`
- Self-documenting
- No abbreviations
- Domain-specific

### 2. **Minimize Public API**
✅ Core types in parent package
- Reduces import complexity
- Clear dependency tree
- Easier to understand

### 3. **Avoid Circular Dependencies**
✅ Unidirectional dependency graph:
```
resources → helpers → components → gui → factories → modes
```

### 4. **Package Cohesion**
✅ Each package has single responsibility:
- `resources`: Constants and asset loading
- `components`: Reusable UI widgets
- `helpers`: Utility functions
- `modes`: UI mode implementations
- `factories`: UI creation patterns

### 5. **Interface Segregation**
✅ `UIMode` interface stays in parent
- Modes implement parent interface
- No mode-specific interfaces exported
- Clean contract

### 6. **Internal Packages** (Not Used - Alternative Considered)
⚠️ Could use `gui/internal/` for truly private code
- Decision: Re-exports provide sufficient encapsulation
- `internal/` reserved for future if needed

### 7. **Flat Over Nested**
✅ Only 2 levels deep:
- `gui/resources/`
- `gui/modes/support/` (exception - closely related)

### 8. **Package Documentation**
✅ Each package should have doc.go:
```go
// Package resources provides GUI assets, fonts, and layout constants
// for the TinkerRogue game UI system.
package resources
```

---

## BACKWARD COMPATIBILITY GUARANTEE

### Existing Code Continues to Work

**Before restructure:**
```go
import "game_main/gui"

mode := &gui.CombatMode{}
button := gui.CreateButtonWithConfig(gui.ButtonConfig{...})
font := gui.SmallFace
```

**After restructure (SAME CODE WORKS):**
```go
import "game_main/gui"

// Via re-export in gui/modes.go
mode := gui.NewCombatMode(manager)

// Via re-export in gui/components.go
button := gui.CreateButtonWithConfig(gui.ButtonConfig{...})

// Via re-export in gui/resources.go
font := gui.SmallFace
```

### New Code Can Use Subpackages Directly

**New code can be more explicit:**
```go
import (
    "game_main/gui"
    "game_main/gui/components"
    "game_main/gui/resources"
)

mode := gui.NewCombatMode(manager)
button := components.CreateButtonWithConfig(components.ButtonConfig{...})
font := resources.SmallFace
```

### Migration Path for Downstream Code

**No forced migration:**
- Existing code continues working via re-exports
- Can gradually adopt subpackage imports
- No breaking changes
- Deprecation warnings (optional future enhancement)

---

## SUCCESS CRITERIA

### Functional Requirements
- [ ] Game builds without errors
- [ ] All 8 modes functional
- [ ] No runtime panics
- [ ] All tests pass
- [ ] No circular import errors

### Code Quality Requirements
- [ ] Clear package boundaries
- [ ] Logical file organization
- [ ] No duplicate code
- [ ] Consistent naming
- [ ] Documented packages

### Developer Experience Requirements
- [ ] Easy to find files
- [ ] Clear dependency graph
- [ ] Simple import paths
- [ ] Backward compatible
- [ ] Documented structure

### Performance Requirements
- [ ] No build time regression
- [ ] No runtime performance change
- [ ] No additional allocations

---

## ROLLBACK PLAN

### Per-Phase Rollback

**If Phase N fails:**
```bash
git reset --hard HEAD~1  # Undo last commit
git clean -fd            # Remove untracked files/dirs
```

**Verify rollback:**
```bash
go build -o game_main/game_main.exe game_main/*.go
go test ./gui/...
```

### Full Rollback

**If multiple phases need reversal:**
```bash
# Find commit before restructure
git log --oneline | grep "before restructure"

# Reset to that commit
git reset --hard <commit-hash>

# Verify
go build -o game_main/game_main.exe game_main/*.go
```

### Partial Rollback

**Keep some phases, undo others:**
```bash
# Example: Keep resources/ but undo components/
git log --oneline
git revert <components-commit-hash>
```

---

## FUTURE ENHANCEMENTS

### After Restructure is Stable

1. **Remove Re-exports** (Breaking Change - Major Version Bump)
   - Force downstream code to use subpackages directly
   - Clearer dependencies
   - Smaller `gui` package API surface

2. **Add `internal/` Packages**
   - Move truly private helpers to `gui/internal/`
   - Enforce encapsulation via compiler

3. **Split Modes Further**
   - `gui/modes/combat/` (combatmode + support classes)
   - `gui/modes/squad/` (squad-related modes)
   - `gui/modes/info/` (info/inventory modes)

4. **Extract Combat UI to Subpackage**
   - `gui/combat/` for combat-specific UI
   - Separate from mode implementations

5. **Deprecation Warnings**
   ```go
   // Deprecated: Use resources.SmallFace instead
   var SmallFace = resources.SmallFace
   ```

---

## CONCLUSION

This restructuring plan provides:

✅ **Clear organization** - 30 files grouped into 6 logical packages
✅ **No breaking changes** - Backward compatible via re-exports
✅ **Low risk** - Phased approach with per-phase validation
✅ **Testable** - Validation after each phase
✅ **Reversible** - Git commits enable easy rollback
✅ **Best practices** - Follows Go package conventions
✅ **3-4 hour effort** - Realistic timeline with clear steps

**Recommendation:** Proceed with phased implementation, commit after each phase, test thoroughly.

---

**END OF IMPLEMENTATION PLAN**
