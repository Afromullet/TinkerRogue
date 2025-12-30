# TinkerRogue Architecture Analysis: Top 3 Refactoring Priorities
Generated: 2025-12-30
Scope: Full Codebase

---

## Executive Summary

### Overall Assessment
The codebase shows solid ECS compliance in tactical systems (squads, combat) but has significant architectural issues in the `common` package (boundary violations, mixed responsibilities) and the `gui` package (excessive complexity, tight coupling to game state). The `input` package is functionally obsolete with the new GUI system.

### High-Value Refactoring Targets
1. **common package**: Mixed responsibilities (ECS core + player-specific data + resources) - Split into focused packages
2. **gui package**: Massive package with unclear boundaries - Restructure subpackages and reduce coupling
3. **input package**: Obsolete legacy input system - Remove or repurpose as backend for GUI modes

### What's Working Well
- **tactical/squads**: Exemplary ECS implementation - pure data components, query-based patterns, value map keys
- **tactical/combat**: Clean separation between combat logic, services, and components
- **world/coords**: Well-defined coordinate system with clear purpose
- **visual/rendering**: Focused rendering package with batch operations

---

## Package Analysis

### 1. common (HIGH PRIORITY)

**Purpose**: Should be ECS core utilities and shared components - currently doing much more

**Cohesion**: Poor - mixing multiple unrelated concerns

**Issues Identified**:

#### Issue 1: Player-Specific Data Mixed with ECS Core
- **Evidence**:
  - Searched `common.PlayerData` - found in 11 files including `input/`, `gui/core/`, `game_main/`
  - `playerdata.go` contains player-specific components (`PlayerThrowable`, `PlayerInputStates`) mixed with ECS core
  - `PlayerResources` in separate file but same package
- **Impact**: Violates single responsibility - `common` should be framework-level, not game-specific
- **Why This Matters**:
  - Forces everything to depend on player-specific concepts
  - Makes ECS utilities tied to player mechanics
  - Prevents clean separation between engine and game logic

**Recommendation 1**: Extract player-specific code to new `player` package
```
Before:
common/
├── ecsutil.go          # ECS framework utilities
├── commoncomponents.go # Shared components (Name, Attributes, Position)
├── playerdata.go       # Player-specific components (MISPLACED)
├── playerresources.go  # Player resources (MISPLACED)
└── positionsystem.go   # Spatial grid system

After:
common/
├── ecsutil.go          # ECS framework utilities
├── components.go       # Truly shared components (Name, Attributes, Position)
└── positionsystem.go   # Spatial grid system

player/               # NEW PACKAGE
├── components.go     # PlayerComponent, PlayerInputStates, PlayerThrowable
├── resources.go      # PlayerResources, resource management
└── playerdata.go     # PlayerData struct (aggregator)
```

**Files to Move**:
- `common/playerdata.go` → `player/components.go` (extract player components)
- `common/playerresources.go` → `player/resources.go`

**Update Imports**: 11 files currently importing `common.PlayerData` need updating:
- `input/inputcoordinator.go`, `input/movementcontroller.go`, `input/combatcontroller.go`, `input/uicontroller.go`
- `gui/core/uimode.go`, `gui/core/contextstate.go`
- `game_main/gameinit.go`, `game_main/gamesetup.go`, `game_main/main.go`
- `testing/testingdata.go`

---

#### Issue 2: Attributes Component Has Methods (ECS Pattern Violation)

- **Evidence**:
  - `commoncomponents.go` lines 73-174: `Attributes` struct has 13 methods
  - Methods include `GetPhysicalDamage()`, `GetHitRate()`, `GetCritChance()`, etc.
  - Per CLAUDE.md: "Pure Data Components - Zero logic, only fields"
- **Usages Found**: Searched `func.*\*Attributes\)` - all methods in `commoncomponents.go`
- **Impact**: Violates pure data component rule from project's ECS guidelines
- **Why This Matters**:
  - Methods on components blur line between data and systems
  - Makes testing harder (can't test attribute calculations separately from struct)
  - Contradicts reference implementation (`squads/` has zero component methods)

**Recommendation 2**: Extract attribute calculations to `combat/stats` system package

```go
Before (in common/commoncomponents.go):
type Attributes struct {
    Strength int
    // ... fields
}
func (a *Attributes) GetPhysicalDamage() int {
    return (a.Strength / 2) + (a.Weapon * 2)
}

After:
// common/components.go - Pure data only
type Attributes struct {
    Strength   int
    Dexterity  int
    Magic      int
    // ... fields only, NO methods
}

// combat/stats/calculator.go - Systems calculate derived stats
func CalculatePhysicalDamage(attr *common.Attributes) int {
    return (attr.Strength / 2) + (attr.Weapon * 2)
}
func CalculateHitRate(attr *common.Attributes) int {
    hitRate := config.DefaultBaseHitChance + (attr.Dexterity * 2)
    if hitRate > config.DefaultMaxHitRate {
        hitRate = config.DefaultMaxHitRate
    }
    return hitRate
}
```

**Migration Strategy**:
1. Create `combat/stats/calculator.go` with all calculation functions
2. Update callsites (grep shows 10 files use `Attributes{}`) to use calculator functions
3. Remove methods from `Attributes` struct
4. Verify tests pass

---

#### Issue 3: Position System Global Variable

- **Evidence**: `common/ecsutil.go` line 22: `GlobalPositionSystem *PositionSystem`
- **Impact**: Global state reduces testability and creates hidden dependencies
- **Usage**: Searched across codebase - used in squad movement, combat, entity cleanup
- **Why This Matters**:
  - Makes testing difficult (can't isolate tests)
  - Creates initialization order dependencies
  - Hidden coupling between packages

**Recommendation 3**: Make PositionSystem a field in EntityManager
```go
// Before
var GlobalPositionSystem *PositionSystem

// After
type EntityManager struct {
    World          *ecs.Manager
    WorldTags      map[string]ecs.Tag
    PositionSystem *PositionSystem  // NEW: Instance field
}
```

**Update Sites**: All references to `common.GlobalPositionSystem` → `manager.PositionSystem`

---

### 2. gui Package (HIGH PRIORITY)

**Purpose**: User interface rendering and interaction management

**Cohesion**: Mixed - combines UI state, rendering, mode management, components, builders, and resources

**Issues Identified**:

#### Issue 1: Package Size and Unclear Organization

- **Evidence**:
  - Glob shows 30 files in `gui` package across 6 subdirectories
  - Subdirectories: `core/`, `builders/`, `widgets/`, `guicomponents/`, `guimodes/`, `guicombat/`, `guisquads/`, `guiresources/`, `specs/`
  - Root package has 11 files with mixed responsibilities
- **Impact**: Hard to navigate, unclear where new UI code belongs
- **Why This Matters**:
  - High cognitive load for developers
  - Encourages adding to root package instead of organized subdirectories
  - Difficult to understand dependency flow

**Recommendation 1**: Consolidate and clarify package structure

```
Current Structure (confusing):
gui/
├── basemode.go              # Mode infrastructure (belongs in core/)
├── buttonbuilders.go        # Button helpers (belongs in builders/)
├── componentfactory.go      # Factory pattern (belongs in builders/)
├── modebuilder.go          # Builder pattern (belongs in builders/)
├── modehelpers.go          # Utilities (unclear purpose)
├── ui_helpers.go           # Generic utilities (unclear purpose)
├── commandhistory.go       # Command pattern (specific feature)
├── core/                   # Mode management ✓
├── builders/               # UI construction ✓
├── widgets/                # Custom widgets ✓
├── guicomponents/          # Live UI components ✓
├── guimodes/               # Specific game modes ✓
├── guicombat/              # Combat UI ✓
├── guisquads/              # Squad UI ✓
├── guiresources/           # Assets/resources ✓
└── specs/                  # Layout specs ✓

Proposed Structure (clearer):
gui/
├── core/                   # Mode management, base types
│   ├── modemanager.go
│   ├── uimode.go
│   ├── contextstate.go
│   ├── basemode.go         # MOVED from root
│   └── gamemodecoordinator.go
├── builders/               # UI construction and factories
│   ├── dialogs.go
│   ├── panels.go
│   ├── widgets.go
│   ├── buttons.go          # MOVED from buttonbuilders.go
│   ├── componentfactory.go # MOVED from root
│   └── modebuilder.go      # MOVED from root
├── components/             # RENAMED from guicomponents (shorter)
│   ├── guicomponents.go
│   ├── guiqueries.go
│   └── squadinfo_cache.go
├── modes/                  # RENAMED from guimodes (shorter)
│   ├── combat/             # NESTED: combat-specific modes
│   │   ├── combatmode.go
│   │   ├── combat_action_handler.go
│   │   ├── combat_input_handler.go
│   │   └── combat_animation_mode.go
│   ├── squads/             # NESTED: squad-specific modes
│   │   ├── squadmanagementmode.go
│   │   ├── squadeditormode.go
│   │   └── squaddeploymentmode.go
│   ├── explorationmode.go
│   ├── inventorymode.go
│   └── renderers.go        # RENAMED from guirenderers.go
├── widgets/                # Custom widget wrappers
├── resources/              # RENAMED from guiresources (shorter)
└── utils/                  # NEW: consolidate helpers
    ├── helpers.go          # Extracted from ui_helpers.go, modehelpers.go
    └── commandhistory.go   # Specific utility
```

**Files to Move**:
- `gui/basemode.go` → `gui/core/basemode.go`
- `gui/buttonbuilders.go` → `gui/builders/buttons.go`
- `gui/componentfactory.go` → `gui/builders/componentfactory.go`
- `gui/modebuilder.go` → `gui/builders/modebuilder.go`
- Merge `gui/ui_helpers.go` + `gui/modehelpers.go` → `gui/utils/helpers.go`

---

#### Issue 2: Tight Coupling Between GUI and Game State

- **Evidence**:
  - `gui/guicombat/combatmode.go` lines 27-34: Direct access to `combatService`, `threatManager`, `dangerVisualizer`
  - `gui/guicomponents/guiqueries.go`: GUI components directly querying ECS manager
  - Mode classes compute game data instead of displaying pre-computed data
- **Impact**: GUI modes responsible for both display AND game logic calculation
- **Why This Matters**:
  - Violates separation of concerns (View shouldn't calculate Model data)
  - Makes testing difficult (can't test UI without full game state)
  - Tightly couples GUI to combat/squad internals

**Recommendation 2**: Introduce ViewModel/Presenter layer

```go
// Current: GUI directly accesses game state
func (cm *CombatMode) Update(deltaTime float64) error {
    currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
    // ... compute squad status, threat levels, etc.
}

// Proposed: GUI receives pre-computed view models
type CombatViewModel struct {
    CurrentFactionName string
    CurrentRound       int
    SquadInfos         []SquadInfo
    ThreatLevels       map[ecs.EntityID]float64
}

func (cm *CombatMode) Update(deltaTime float64, viewModel *CombatViewModel) error {
    // Just display the data, don't compute it
    cm.displayFactionInfo(viewModel.CurrentFactionName, viewModel.CurrentRound)
    cm.displaySquads(viewModel.SquadInfos)
}
```

Create new package: `gui/viewmodels/combat_viewmodel.go`

**Note**: This is a larger architectural shift but would significantly improve testability and separation of concerns.

---

#### Issue 3: Mode Classes Are Too Large

- **Evidence**:
  - `gui/guicombat/combatmode.go`: 844 lines
  - Manages: rendering, input, UI updates, AI execution, animations, threat visualization, layer visualization
  - Too many responsibilities for a single class
- **Impact**: Violates Single Responsibility Principle
- **Why This Matters**:
  - Difficult to understand and maintain
  - Hard to test individual responsibilities
  - Changes to one feature affect others

**Recommendation 3**: Extract responsibilities to separate managers

```
Current:
CombatMode (844 lines)
├── Rendering (danger maps, highlights, movement tiles)
├── Input handling (combat actions, keyboard shortcuts)
├── UI updates (panels, lists, details)
├── AI execution (turn management, attack animations)
├── Threat visualization
└── Layer visualization

Proposed:
CombatMode (coordinator - ~200 lines)
├── CombatRenderer (rendering all visual elements)
├── CombatInputHandler (already exists ✓)
├── CombatUIManager (panel updates)
├── CombatAICoordinator (AI turn execution)
└── VisualizationManager (threat/layer display)
```

**Next Steps for Refactoring**:
1. Extract rendering logic to `CombatRenderer`
2. Extract AI coordination to `CombatAICoordinator`
3. Extract UI update logic to `CombatUIManager`
4. `CombatMode` becomes thin coordinator delegating to managers

---

### 3. input Package (MEDIUM PRIORITY - OBSOLETE)

**Purpose**: Originally handled player input for movement and combat - now replaced by GUI system

**Cohesion**: Good (focused on input) but **functionally obsolete**

**Issues Identified**:

#### Issue 1: Redundant Input System

- **Evidence**:
  - `input/inputcoordinator.go`: Coordinates `MovementController`, `CombatController`, `UIController`
  - `gui/core/modemanager.go`: New GUI system has its own input handling via `HandleInput(inputState *core.InputState)`
  - Searched `NewInputCoordinator` - only called in `game_main/gamesetup.go`
- **Usages Found**:
  - Grep `InputCoordinator` shows usage in `game_main/gamesetup.go`, `game_main/main.go`
  - **No usage in GUI modes** - GUI uses `core.InputState` instead
- **Impact**: Dead code maintained alongside new system
- **Why This Matters**:
  - Confuses developers ("which input system do I use?")
  - Maintenance burden for unused code
  - Risk of bugs if accidentally called

**Recommendation 1**: Remove obsolete input system OR repurpose as backend

**Option A: Complete Removal**
```
Files to Remove:
- input/inputcoordinator.go
- input/movementcontroller.go
- input/combatcontroller.go
- input/uicontroller.go

Update:
- game_main/gamesetup.go: Remove InputCoordinator initialization
- game_main/main.go: Remove InputCoordinator.HandleInput() calls
```

**Evidence of Safety**:
- Searched `ic.HandleInput()` in `game_main/main.go` - found 1 call site
- Searched `InputCoordinator` - 3 files use it, all in `game_main/` and `input/` packages
- No references in `gui/` package (GUI uses new system)

**Option B: Repurpose as GUI Input Backend**
```go
// Keep input package as backend for GUI modes
// GUI modes delegate to input controllers for complex game actions

package input

// Action represents a game action triggered by input
type Action interface {
    Execute() error
}

type MoveAction struct { /* ... */ }
type AttackAction struct { /* ... */ }

// InputActionFactory creates actions from input events
type InputActionFactory struct { /* ... */ }

// GUI modes use this:
action := inputFactory.CreateMoveAction(fromPos, toPos)
action.Execute()
```

**Recommendation**: Start with **Option A (removal)** - cleaner architecture. If complex input logic is needed later, implement directly in GUI mode handlers.

---

## Dependency Issues

### Circular Dependencies
None detected. Package dependency direction is sensible:
- `gui` → `tactical` → `common` ✓
- `input` → `common` ✓ (but input is obsolete)

### Inappropriate Dependencies
None. Dependencies flow from high-level (GUI, input) to low-level (common, world, tactical).

---

## Game Architecture Assessment

### ECS Organization
**Excellent** in tactical systems, **needs improvement** in common:
- ✅ **tactical/squads**: Pure data components, system-based logic, query patterns
- ✅ **tactical/combat**: Proper separation of concerns
- ❌ **common**: Attributes component has methods (violates pure data rule)
- ❌ **common**: Player-specific components mixed with framework code

### State Management
**Good** separation between game state (ECS) and UI state:
- ✅ `BattleMapState` / `OverworldState` contain only UI selections and mode flags
- ✅ Game state stored in ECS components
- ⚠️ GUI modes compute derived data instead of receiving view models (improvement opportunity)

### Input Flow
**Transitional** - old system obsolete, new system in place:
- ✅ GUI mode system handles input via `HandleInput(inputState)`
- ❌ Legacy `input/` package still exists but unused by GUI
- **Action**: Remove legacy input system

---

## Prioritized Recommendations

### High Priority (Significant Impact)

#### 1. **Split common Package**
   - **Packages**: common
   - **Change**: Extract player-specific code to new `player/` package
   - **Why**: Eliminates boundary violation, clarifies framework vs. game logic separation
   - **Files Affected**: 11 files importing `common.PlayerData`, 3 files in `common/`

#### 2. **Extract Attributes Calculations to System Package**
   - **Packages**: common, combat (new `combat/stats/` subpackage)
   - **Change**: Remove methods from `Attributes`, create `stats.Calculator` functions
   - **Why**: Enforces ECS pure data component rule, improves testability
   - **Files Affected**: `common/commoncomponents.go`, 10 files using `Attributes{}`

#### 3. **Restructure GUI Package**
   - **Packages**: gui and all subpackages
   - **Change**:
     - Move root-level files to appropriate subdirectories (`core/`, `builders/`, `utils/`)
     - Rename `guicomponents/` → `components/`, `guimodes/` → `modes/`
     - Nest mode-specific packages (`modes/combat/`, `modes/squads/`)
   - **Why**: Improves navigability, clarifies organization, reduces cognitive load
   - **Files Affected**: 30 files in gui package

### Medium Priority (Moderate Impact)

#### 4. **Remove Obsolete Input Package**
   - **Packages**: input, game_main
   - **Change**: Delete `input/` package, remove from game initialization
   - **Why**: Eliminates dead code, reduces confusion
   - **Files Affected**: 4 files in `input/`, 2 files in `game_main/`

#### 5. **Convert GlobalPositionSystem to Instance Field**
   - **Packages**: common
   - **Change**: Make `PositionSystem` a field in `EntityManager`
   - **Why**: Improves testability, eliminates global state
   - **Files Affected**: All references to `common.GlobalPositionSystem` (search required for count)

#### 6. **Extract CombatMode Responsibilities**
   - **Packages**: gui/guicombat
   - **Change**: Create `CombatRenderer`, `CombatUIManager`, `CombatAICoordinator` classes
   - **Why**: Reduces class size, clarifies responsibilities, improves testability
   - **Files Affected**: `gui/guicombat/combatmode.go`

### Low Priority (Nice to Have)

#### 7. **Introduce ViewModel Layer for GUI**
   - **Packages**: gui (new `viewmodels/` subpackage)
   - **Change**: Create view model types, update modes to consume them
   - **Why**: Decouples GUI from game state computation, improves testability
   - **Files Affected**: All mode classes in `gui/guimodes/`, `gui/guicombat/`, `gui/guisquads/`

---

## What NOT to Change

### Keep These Patterns

1. **tactical/squads ECS implementation** - This is the reference implementation. Do NOT change:
   - Pure data components (no methods)
   - Query-based entity lookup
   - Value map keys (not pointer keys)

2. **coords.CoordManager indexing** - Already has critical warnings in CLAUDE.md. Leave as-is.

3. **combat/factionmanager separation** - Clean separation between faction logic and combat. Don't merge.

4. **world/worldmap generator registry** - `init()` registration pattern works well for extensibility.

---

## Verification Evidence Summary

All recommendations above are backed by verification:

| Recommendation | Verification Method | Evidence |
|----------------|---------------------|----------|
| Split common package | Grep `common.PlayerData` | 11 files found (listed above) |
| Attributes methods | Grep `func.*\*Attributes\)` | 13 methods in commoncomponents.go |
| GUI restructure | Glob `gui/**/*.go` | 30 files across 9 subdirectories |
| Remove input package | Grep `InputCoordinator`, `ic.HandleInput()` | 3 files in game_main, 4 in input, 0 in gui |
| GlobalPositionSystem | Grep `GlobalPositionSystem` | Used in squad movement, combat, entity cleanup |

---

END OF ANALYSIS
