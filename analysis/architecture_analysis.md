# TinkerRogue Codebase Architecture Analysis
Generated: 2025-12-29
Scope: Full Codebase

---

## Executive Summary

### Overall Assessment
The codebase demonstrates strong ECS architectural discipline with clear package separation. The squad/combat systems follow excellent patterns, but the GUI layer shows coupling issues and unclear boundaries between UI state, game state, and query services. Several tactical packages could benefit from consolidation.

### High-Value Refactoring Targets
1. **GUI Query Services**: Consolidate `guicomponents.GUIQueries` and combat/squad caches - reduces three-layer indirection
2. **Tactical Package Sprawl**: Merge `combatservices` into `combat` - eliminates wrapper layer serving single client
3. **Input Coordinator**: Split shared state management from controller coordination - clarifies ownership
4. **Templates Package**: Rename to `entities` and clarify creation vs data loading responsibilities
5. **Widget Builders**: Consolidate `gui/widgets` and `gui/builders` - reduces package hopping

### What's Working Well
- **ECS Component Design**: Pure data components throughout (squads, combat, gear)
- **Subsystem Self-Registration**: Clean initialization pattern in `common/ecsutil.go`
- **Position System**: Single source of truth via `GlobalPositionSystem`
- **World Generation**: Plugin-style generator registry

---

## Package Analysis

### common/

**Purpose**: Core ECS utilities, shared components, and global systems

**Cohesion**: Good - all code serves ECS infrastructure

**Issues Identified**:
- **PlayerData Hybrid Responsibility**: Contains both player component marker and throwable state management
  - **Impact**: PlayerData has dual roles - ECS marker and input state container
  - **Recommendation**: Split into:
    ```go
    // In common/playercomponents.go
    type PlayerComponent struct {} // Just the marker

    // In input/playerstate.go
    type PlayerInputState struct {
        Throwables  PlayerThrowable
        InputStates PlayerInputStates
    }
    ```
  - **Signature Change**:
    ```go
    // Before
    type PlayerData struct {
        Throwables     PlayerThrowable
        InputStates    PlayerInputStates
        PlayerEntityID ecs.EntityID
        Pos            *coords.LogicalPosition
    }

    // After (in common)
    type Player struct {
        EntityID ecs.EntityID
    }

    // After (in input)
    type PlayerInputState struct {
        Throwables  PlayerThrowable
        InputStates PlayerInputStates
    }
    ```

**Dependencies**:
- Imports: `world/coords`, `config`
- Imported by: Nearly all packages (appropriate for common infrastructure)
- Issues: None - dependency direction is correct

---

### gui/

**Purpose**: User interface layer (modes, widgets, rendering)

**Cohesion**: Mixed - contains multiple sub-concerns across 8 subdirectories

**Issues Identified**:

#### 1. Query Service Three-Layer Indirection
- **Issue Type**: Over-abstraction
- **Description**: `GUIQueries` wraps `SquadQueryCache` and `CombatQueryCache` which wrap ECS queries
  - Layer 1: `guicomponents.GUIQueries` (GUI facade)
  - Layer 2: `squads.SquadQueryCache`, `combat.CombatQueryCache` (domain caches)
  - Layer 3: ECS `World.Query()` (actual data)
- **Impact**: Forces three indirections for simple squad lookups; unclear which cache to invalidate
- **Recommendation**: Eliminate `GUIQueries` wrapper - GUI modes directly use domain caches
  - **Before**:
    ```go
    // In CombatMode
    queries := guicomponents.NewGUIQueries(manager)
    squadInfo := queries.GetSquadInfo(squadID) // Goes through 3 layers
    ```
  - **After**:
    ```go
    // In CombatMode
    squadCache := squads.NewSquadQueryCache(manager)
    combatCache := combat.NewCombatQueryCache(manager)
    squadInfo := squadCache.GetSquadInfo(squadID) // Direct access
    ```
- **Risks**: Requires updating 10+ GUI mode files
- **Incremental**: Yes - migrate one mode at a time

#### 2. Widget Builders Package Split
- **Issue Type**: Boundary Confusion
- **Description**: Widget creation split across `gui/widgets` (factories) and `gui/builders` (configs)
  - `widgets/button_factory.go` creates buttons
  - `builders/widgets.go` also creates buttons with different config style
  - No clear rule for which to use
- **Impact**: Developers don't know where to add new widget creators; duplicated patterns
- **Recommendation**: Consolidate into `gui/widgets` with single config-based pattern
  - Move all `builders/*.go` widget creation into `widgets/`
  - Keep `builders/` only for high-level panel composition
- **Risks**: Moderate - requires updating imports across GUI modes
- **Incremental**: Yes - consolidate one widget type at a time

#### 3. Mode Builder Abstraction Leakage
- **Issue Type**: Inappropriate Abstraction
- **Description**: `gui/modebuilder.go` tries to abstract mode initialization but modes still do manual setup
  - 50% of modes use ModeBuilder
  - 50% of modes bypass it and build panels manually
  - No enforcement or clear benefit
- **Impact**: Inconsistent initialization; ModeBuilder doesn't reduce boilerplate
- **Recommendation**: Remove ModeBuilder - standardize on direct panel building
  - Each mode builds its own panels in `Initialize()`
  - Share common panel patterns via `widgets/` helpers
- **Risks**: Low - ModeBuilder is thin wrapper
- **Incremental**: Yes - migrate modes gradually

#### 4. GUI State vs Game State Confusion
- **Issue Type**: Misplaced Responsibility
- **Description**: `BattleMapState` appears to store UI selection but comments warn against caching game data
  - Modes inconsistently read from state vs directly querying ECS
  - Unclear whether `SelectedSquadID` is authoritative or derived
- **Impact**: Developers unsure whether to trust state or query ECS
- **Recommendation**: Clarify pattern:
  - `BattleMapState` = ONLY UI ephemeral (selection, mode flags)
  - All game data = Query from ECS/services
  - Add enforcement comments to state struct
- **Risks**: Low - mostly documentation
- **Incremental**: Yes - update mode by mode

**Dependencies**:
- Imports: `common`, `tactical/*`, `world/*`, `gear`, `visual/*`, `config`
- Imported by: `game_main` only
- Issues: GUI imports tactical packages directly - good separation

---

### tactical/

**Purpose**: Combat, squads, AI, and tactical gameplay systems

**Cohesion**: Good overall, but spread across 6 subdirectories

**Issues Identified**:

#### 1. combatservices/ Package
- **Issue Type**: Unnecessary Wrapper Layer
- **Description**: `combatservices.CombatService` wraps systems from `tactical/combat` and `tactical/ai`
  - Only client is `gui/guicombat/combatmode.go`
  - Provides no additional logic beyond aggregation
  - Type aliases (`AIController = ai.AIController`) show it's just re-exporting
- **Impact**: Extra layer of indirection; unclear why it exists
- **Recommendation**: **Merge `combatservices/` into `tactical/combat`**
  - Move `CombatService` struct to `combat/combat_facade.go`
  - GUI imports `combat` directly
  - Eliminates package and type alias layer
- **Signature Change**:
  ```go
  // Before
  import "game_main/tactical/combatservices"
  cs := combatservices.NewCombatService(manager)

  // After
  import "game_main/tactical/combat"
  cs := combat.NewCombatService(manager)
  ```
- **Risks**: Low - only one client to update
- **Incremental**: No - single atomic change

#### 2. squadcommands/ Package Isolation
- **Issue Type**: Premature Abstraction
- **Description**: Command pattern with queue components but unused
  - `squadcommands/` defines `SquadCommand` interface
  - `queue_components.go` has ECS components for command queuing
  - Recent commits removed command queue system
  - Commands used for undo/redo in squad editor only
- **Impact**: Over-engineered for simple undo; confusing architecture
- **Recommendation**: Evaluate two options:
  1. **If commands only used for GUI undo**: Move to `gui/guisquads/commands.go`
  2. **If commands part of gameplay**: Keep package but remove queue components
- **Risks**: Low - localized usage
- **Incremental**: Yes - evaluate usage first

#### 3. behavior/ Package Purpose
- **Issue Type**: Unclear Scope
- **Description**: Contains threat evaluation, danger visualization, and layer painting
  - Some files are AI decision logic (`threat_composite.go`, `action_evaluator.go`)
  - Some files are GUI rendering (`dangervisualizer.go`, `layervisualizer.go`)
  - Mixing concerns in single package
- **Impact**: Unclear whether behavior is AI logic or GUI helpers
- **Recommendation**: Split into:
  - `tactical/ai/threat_evaluation.go` - AI decision logic
  - `gui/combat/visualizers.go` - Rendering helpers
- **Risks**: Moderate - affects combat mode and AI controller
- **Incremental**: Yes - move files one at a time

**Dependencies**:
- Imports: `common`, `world/coords`, `gear` (appropriate)
- Imported by: `gui/guicombat`, `game_main`
- Issues: `behavior/` imported by both AI and GUI - confirms mixed concerns

---

### input/

**Purpose**: Player input handling and controller coordination

**Cohesion**: Good - all input-related

**Issues Identified**:

#### 1. Shared Input State Ownership
- **Issue Type**: Unclear Ownership
- **Description**: `SharedInputState` created by `InputCoordinator` but passed to all controllers
  - Contains shared data (cursor position, turn state)
  - Controllers read/write to shared state
  - No clear owner or lifecycle
- **Impact**: Unclear who updates what; potential race conditions in future
- **Recommendation**: Make state ownership explicit:
  ```go
  // Before
  type SharedInputState struct {
      PrevCursor coords.PixelPosition
      TurnTaken  bool
  }

  // After - Split into coordinator-owned and read-only views
  type InputCoordinatorState struct {
      turnTaken bool
      cursor    coords.PixelPosition
  }

  type InputStateView struct {
      Cursor    coords.PixelPosition
      TurnTaken bool
  }

  func (ic *InputCoordinator) GetStateView() InputStateView {
      return InputStateView{
          Cursor:    ic.state.cursor,
          TurnTaken: ic.state.turnTaken,
      }
  }
  ```
- **Risks**: Moderate - affects all controllers
- **Incremental**: Yes - migrate controllers one by one

**Dependencies**:
- Imports: `common`, `world/*`
- Imported by: `game_main` only
- Issues: None

---

### templates/

**Purpose**: Entity creation from JSON data templates

**Cohesion**: Poor - mixing data loading, template storage, and entity creation

**Issues Identified**:

#### 1. Package Name Misleading
- **Issue Type**: Naming Clarity
- **Description**: "templates" suggests patterns, but package creates entities
  - `creators.go` - Entity creation functions
  - `readdata.go` - JSON loading
  - `jsonstructs.go` - Data structures
  - `templatelib.go` - Global template storage
- **Impact**: Unclear what package does; ambiguous about responsibility
- **Recommendation**: **Rename to `entities/`**
  - `entities/loading.go` - JSON reading
  - `entities/templates.go` - Template storage
  - `entities/creation.go` - Entity creation
- **Risks**: Low - rename is mechanical
- **Incremental**: No - single refactor

#### 2. Global Template Variables
- **Issue Type**: Singleton Anti-pattern
- **Description**: Package-level vars hold all templates
  ```go
  var MonsterTemplates []JSONMonster
  var MeleeWeaponTemplates []JSONMeleeWeapon
  ```
- **Impact**: Implicit initialization order dependency; hard to test
- **Recommendation**: Wrap in service:
  ```go
  type TemplateRegistry struct {
      monsters  []JSONMonster
      weapons   []JSONMeleeWeapon
  }

  func NewTemplateRegistry() *TemplateRegistry {
      r := &TemplateRegistry{}
      r.Load()
      return r
  }
  ```
- **Risks**: Low - single usage point
- **Incremental**: Yes

**Dependencies**:
- Imports: `common`, `gear`, `world/*`
- Imported by: `game_main`
- Issues: None

---

### gear/

**Purpose**: Inventory and item system

**Cohesion**: Excellent - pure ECS pattern, reference implementation

**Issues Identified**: None

**Dependencies**:
- Imports: `common`, `world/coords`, `visual/graphics`
- Imported by: `gui`, `templates`, `tactical`
- Issues: None - gear is properly domain-focused

---

### world/

**Purpose**: Coordinate systems, map generation, pathfinding

**Cohesion**: Good - clear world simulation responsibilities

**Issues Identified**:

#### 1. Generator Registry in worldmap/
- **Issue Type**: Minor Organization
- **Description**: Generator registration in `generator.go` is clean plugin pattern
  - Each generator self-registers via `init()`
  - Registry provides lookup and fallback
  - **This is GOOD ARCHITECTURE** - no changes needed
- **Impact**: N/A - highlighting as reference implementation
- **Recommendation**: None - keep as-is, use as pattern for other registries

**Dependencies**:
- Imports: None (bottom layer)
- Imported by: Nearly all packages
- Issues: None - world is foundation layer

---

### visual/

**Purpose**: Rendering, graphics, sprite management

**Cohesion**: Good - rendering concerns isolated

**Issues Identified**: None observed at package level

**Dependencies**:
- Imports: `world/coords`
- Imported by: `gui`, `game_main`
- Issues: None

---

## Dependency Issues

### Circular Dependencies
None detected - package dependency graph is acyclic.

### Inappropriate Dependencies
| From | To | Issue | Recommendation |
|------|-----|-------|----------------|
| tactical/behavior | gui (via visualizers) | AI package shouldn't have GUI code | Move visualizers to gui/combat/ |
| gui/guicomponents | tactical/combat, tactical/squads | GUI wraps domain caches | Remove GUIQueries wrapper |

---

## Game Architecture Assessment

### ECS Organization
**Excellent** - Components are pure data throughout:
- ✅ `squads/squadcomponents.go` - Zero logic, only fields
- ✅ `combat/combatcomponents.go` - Pure data
- ✅ `gear/Inventory.go` - System functions, not methods
- ✅ EntityID usage - No `*ecs.Entity` storage

**Reference Pattern**: `tactical/squads/` is textbook ECS implementation.

### State Management
**Good with Caveats**:
- Game state: Pure ECS (correct)
- UI state: Properly separated in `gui/core/contextstate.go`
- **Issue**: Comments warn against caching but pattern is unclear
- **Fix**: Document state usage patterns explicitly

### Input Flow
**Good Architecture**:
- InputCoordinator → Controllers → Game Systems
- Priority-based handling (UI > Combat > Movement)
- **Issue**: Shared state ownership unclear
- **Fix**: Explicit state ownership pattern

---

## Prioritized Recommendations

### High Priority (Significant Impact)

#### 1. Consolidate Query Services
- **Packages**: `gui/guicomponents`, `tactical/squads`, `tactical/combat`
- **Change**: Remove `GUIQueries` wrapper - modes use domain caches directly
- **Why**: Eliminates 3-layer indirection, clarifies cache invalidation, reduces complexity
- **Effort**: Medium (affects 10+ GUI modes)

#### 2. Merge combatservices/ into combat/
- **Packages**: `tactical/combatservices`, `tactical/combat`
- **Change**: Move `CombatService` to `combat/combat_facade.go`
- **Why**: Eliminates wrapper package serving single client, simplifies imports
- **Effort**: Low (only 1 client)

#### 3. Split behavior/ Package
- **Packages**: `tactical/behavior`, `tactical/ai`, `gui/combat`
- **Change**: Move threat evaluation to `ai/`, visualizers to `gui/combat/`
- **Why**: Clarifies AI vs GUI responsibilities, proper package cohesion
- **Effort**: Medium (affects CombatMode and AIController)

### Medium Priority (Moderate Impact)

#### 4. Consolidate Widget Builders
- **Packages**: `gui/widgets`, `gui/builders`
- **Change**: Merge all widget creation into `gui/widgets/`, keep `builders/` for panel composition only
- **Why**: Single location for widget patterns, reduces package hopping
- **Effort**: Medium (many import updates)

#### 5. Clarify Input State Ownership
- **Packages**: `input/`
- **Change**: Make InputCoordinator explicit owner, provide read-only views to controllers
- **Why**: Clear ownership prevents future bugs, easier to reason about
- **Effort**: Medium (affects all controllers)

#### 6. Rename templates/ to entities/
- **Packages**: `templates/`
- **Change**: Rename package, organize into loading/templates/creation files
- **Why**: Clearer purpose, better file organization
- **Effort**: Low (mechanical rename)

### Low Priority (Nice to Have)

#### 7. Remove ModeBuilder Abstraction
- **Packages**: `gui/`
- **Change**: Delete `modebuilder.go`, standardize on direct panel building
- **Why**: Unused abstraction that doesn't reduce boilerplate
- **Effort**: Low (thin wrapper)

#### 8. Split PlayerData
- **Packages**: `common/`, `input/`
- **Change**: Keep Player marker in common, move input state to input package
- **Why**: Single responsibility, clearer boundaries
- **Effort**: Low (localized change)

#### 9. Wrap Template Globals in Service
- **Packages**: `templates/` (or `entities/` after rename)
- **Change**: Create `TemplateRegistry` struct to hold templates
- **Why**: Testability, explicit initialization
- **Effort**: Low (single usage point)

---

## What NOT to Change

### Keep These Patterns
1. **Subsystem Self-Registration** (`common/ecsutil.go`) - Elegant initialization pattern
2. **Generator Registry** (`world/worldmap/generator.go`) - Clean plugin architecture
3. **Position System** (`common/positionsystem.go`) - Single source of truth working well
4. **Pure Data Components** - Zero logic in components throughout codebase
5. **EntityID Usage** - Correct ECS pattern, no entity pointer storage
6. **Squad Package Structure** - `squadcomponents.go` + `squadqueries.go` + `*system.go` is reference implementation

### Don't Over-Engineer
1. **Don't add service layers** - Keep flat package structure
2. **Don't extract interfaces prematurely** - Concrete types are fine for game code
3. **Don't add dependency injection** - Direct construction is clearer
4. **Don't cache everything** - Turn-based game, event-driven invalidation is sufficient

---

## Implementation Strategy

### Phase 1: Foundation Cleanup (2-3 days)
1. Merge `combatservices/` into `combat/`
2. Rename `templates/` to `entities/`
3. Split `PlayerData` (common vs input)

### Phase 2: GUI Refactoring (4-5 days)
1. Remove `GUIQueries` wrapper (migrate modes one by one)
2. Consolidate widget builders
3. Document state usage patterns

### Phase 3: Tactical Refinement (3-4 days)
1. Split `behavior/` package
2. Evaluate `squadcommands/` usage
3. Clarify input state ownership

---

END OF ANALYSIS
