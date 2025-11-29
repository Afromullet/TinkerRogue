# Package Refactoring Recommendations
Generated: 2025-11-29
Scope: Full Codebase Analysis

---

## Executive Summary

### Overall Assessment
The codebase demonstrates strong ECS architecture with well-organized packages. The squads and gear packages serve as exemplary ECS implementations. Primary issues center on entity creation responsibilities being scattered across packages and the spawning package having unclear ownership boundaries.

### Critical Refactoring Targets
1. **Entity Creation Consolidation**: Entity creation split across `entitytemplates`, `squads`, `spawning` - consolidate ownership
2. **Spawning Package Boundaries**: Unclear if spawning owns "when to spawn" or "how to create" - needs clarification
3. **GUI Package Organization**: Multiple `gui/*` subpackages with overlapping responsibilities

### What's Working Well
- **squads/**: Exemplary ECS organization - pure data components, query-based systems, EntityID-only relationships
- **gear/Inventory.go**: Perfect pure data component example with system-based logic
- **systems/positionsystem.go**: Value-based map keys, clean O(1) spatial queries
- **combat/**: Clean component separation, proper turn management
- **gui/core/**: Well-designed mode management system with clear state boundaries

---

## Package Analysis

### common/

**Purpose**: Core ECS utilities, shared components (Position, Attributes, Name), EntityManager wrapper

**Cohesion**: Good

**Issues Identified**:
- **Mixed Concerns**: Package contains both ECS utilities (ecsutil.go) and game-specific player data (playerdata.go, playerresources.go)
  - **Impact**: Blurs the line between "common ECS infrastructure" and "game domain logic"
  - **Recommendation**: Move player-specific data to a dedicated `player` package
  - **Priority**: Medium

**Dependencies**:
- Imports: coords, systems, ecs
- Imported by: Nearly all packages (correct for common utilities)

---

### entitytemplates/

**Purpose**: Template-based entity creation from JSON, factory functions

**Cohesion**: Mixed - handles both data loading AND entity creation

**Issues Identified**:
- **Boundary Confusion with spawning/**: Both packages create entities
  - **Impact**: Unclear ownership - "who creates entities?"
  - **Current State**:
    - `entitytemplates.CreateMonster()` - creates monster with components
    - `spawning.SpawnMonster()` - calls entitytemplates, adds position logic
    - `squads.CreateUnitEntity()` - creates units directly without entitytemplates
  - **Recommendation**: Choose ONE pattern:
    - **Option A (Preferred)**: `entitytemplates` owns ALL entity creation, `spawning` owns spawn timing/positioning
    - **Option B**: `spawning` owns ALL monster creation, `entitytemplates` only loads JSON data
  - **Signature Change** (Option A):
    ```go
    // entitytemplates - owns creation
    func CreateMonster(manager EntityManager, template JSONMonster, pos LogicalPosition) ecs.EntityID

    // spawning - owns timing/positioning only
    func SpawnMonster(ecsmanager EntityManager, gm *GameMap) {
        pos := selectValidSpawnPosition(gm, 3)
        template := entitytemplates.MonsterTemplates[0]
        entitytemplates.CreateMonster(ecsmanager, template, *pos)
    }
    ```
  - **Priority**: High

- **Inconsistent Entity Creation**: `squads` creates units directly, not using entitytemplates
  - **Impact**: Multiple paths to create similar entities
  - **Recommendation**: Unify unit creation through entitytemplates or clearly separate "squad units" from "world monsters"
  - **Priority**: Medium

**Dependencies**:
- Imports: common, coords, worldmap, rendering
- Imported by: spawning, squads, game_main, testing

---

### spawning/

**Purpose**: Unclear - either "spawn timing/positioning" OR "entity creation orchestration"

**Cohesion**: Poor - mixed responsibilities

**Issues Identified**:
- **Ownership Ambiguity**: Does spawning own creation logic or just spawning rules?
  - **Impact**: Developers unsure where to add new spawn behavior
  - **Current Split**:
    - `spawnmonsters.go` - calls entitytemplates
    - `spawnloot.go` - calls entitytemplates
    - `loottables.go`, `probtables.go` - probability/randomization logic
  - **Recommendation**: Narrow scope to ONE responsibility:
    - **Preferred**: Spawning owns WHEN/WHERE to spawn, entitytemplates owns WHAT to create
    - Move all `Create*()` calls to entitytemplates, spawning only selects templates and positions
  - **Priority**: High

- **No Clear Public API**: Package mixes exported spawn functions with internal probability helpers
  - **Impact**: Unclear what external code should call
  - **Recommendation**: Define clear API surface:
    - Public: `SpawnMonster()`, `SpawnStartingCreatures()`, `SpawnLoot()`
    - Private: probability tables, position selection
  - **Priority**: Low

**Dependencies**:
- Imports: common, coords, entitytemplates, worldmap
- Imported by: game_main only

---

### squads/

**Purpose**: Squad-based tactical combat ECS system

**Cohesion**: Excellent - textbook ECS organization

**Issues Identified**:
- **None** - This package is exemplary. Keep as reference implementation.

**What's Working**:
- Pure data components (squadcomponents.go)
- Query functions (squadqueries.go)
- System functions (squadcreation.go, squadcombat.go, squadabilities.go)
- EntityID-only relationships
- Value-based map keys (not pointer keys)
- Command pattern for operations (squadcommands/)
- Service layer for complex operations (squadservices/)

**Dependencies**:
- Imports: common, coords, entitytemplates, combat
- Imported by: gui/guisquads, combat, game_main

---

### combat/

**Purpose**: Turn-based combat system (factions, turns, actions)

**Cohesion**: Good

**Issues Identified**:
- **Circular Dependency with squads**: combat imports squads for abilities, squads imports combat for faction data
  - **Impact**: Tight coupling between packages
  - **Current Dependency**: `combat/turnmanager.go` calls `squads.CheckAndTriggerAbilities()`
  - **Recommendation**: Extract ability system to separate package OR move ability execution to combat
  - **Signature Change**:
    ```go
    // Option 1: Move to combat package
    // combat/abilities.go
    func CheckAndTriggerSquadAbilities(squadID ecs.EntityID, manager *EntityManager)

    // Option 2: Create abilities package
    // abilities/trigger.go
    func CheckAndTriggerAbilities(squadID ecs.EntityID, manager *EntityManager)
    ```
  - **Priority**: Medium

**Dependencies**:
- Imports: common, squads, coords
- Imported by: combat/combatservices, gui/guicombat, game_main

---

### gui/

**Purpose**: All UI/GUI code

**Cohesion**: Mixed - multiple subpackages with overlapping concerns

**Subpackages**:
- `gui/core/` - Mode management, context, UIMode interface
- `gui/widgets/` - Widget creation, layout, panels
- `gui/guicomponents/` - UI update components, query wrappers
- `gui/guimodes/` - Specific mode implementations (exploration, inventory, info)
- `gui/guicombat/` - Combat-specific UI
- `gui/guisquads/` - Squad-specific UI
- `gui/guiresources/` - Shared GUI resources

**Issues Identified**:
- **Overlapping Responsibilities**: `gui/widgets/` vs `gui/guicomponents/` both create UI elements
  - **Impact**: Unclear where new UI code belongs
  - **Current State**:
    - `widgets/` - Low-level widget factories (buttons, panels, labels)
    - `guicomponents/` - Higher-level components (SquadListComponent, DetailPanelComponent)
  - **Recommendation**: Clear naming/documentation of boundary:
    - `widgets/` - Stateless factories, pure creation
    - `guicomponents/` - Stateful components with update logic
  - **Priority**: Low

- **GUI State vs Game State Properly Separated**: BattleMapState and OverworldState are UI-only (GOOD)
  - **What's Working**: Clear separation between UI selection state and ECS game state
  - **Keep This Pattern**: Don't add game logic to UI state structures

**Dependencies**:
- Imports: common, coords, combat, squads, gear, worldmap, rendering, input
- Imported by: game_main

---

### input/

**Purpose**: Player input handling (movement, combat, UI)

**Cohesion**: Good - coordinator pattern with separate controllers

**Issues Identified**:
- **None** - Clean separation of concerns with InputCoordinator delegating to specialized controllers

**What's Working**:
- Controller pattern (MovementController, CombatController, UIController)
- Priority-based input handling
- Shared state for coordination

**Dependencies**:
- Imports: common, coords, worldmap
- Imported by: game_main

---

### gear/

**Purpose**: Inventory and item system

**Cohesion**: Excellent - pure ECS implementation

**Issues Identified**:
- **None** - gear/Inventory.go is exemplary pure data component with system functions

**What's Working**:
- Pure data component (Inventory struct)
- All logic in system functions (AddItem, RemoveItem, etc.)
- EntityID-based item references
- Clear separation of display logic

**Dependencies**:
- Imports: common, ecs
- Imported by: gui/guimodes, entitytemplates, testing

---

### worldmap/

**Purpose**: Map generation and tile management

**Cohesion**: Good - generator registry pattern with multiple algorithms

**Issues Identified**:
- **None** - Clean generator pattern with good extensibility

**What's Working**:
- Generator registry (generator.go)
- Multiple algorithm implementations (gen_*.go)
- Clear interface (MapGenerator)
- Proper init() registration

**Dependencies**:
- Imports: coords, graphics, rendering, common
- Imported by: spawning, entitytemplates, gui, combat, game_main

---

### coords/

**Purpose**: Coordinate system management (logical, pixel, tile indexing)

**Cohesion**: Good

**Issues Identified**:
- **None** - Critical system working correctly

**What's Working**:
- Global CoordManager prevents index out of bounds bugs
- Clear conversion functions
- Properly used throughout codebase

**Dependencies**:
- Imports: None (pure coordinate logic)
- Imported by: Nearly all packages (correct for coordinate system)

---

### systems/

**Purpose**: ECS systems (PositionSystem)

**Cohesion**: Good

**Issues Identified**:
- **Single System in Package**: Only contains PositionSystem
  - **Impact**: Package seems underutilized
  - **Recommendation**: Either add more systems here OR move PositionSystem to common (it's already used as global)
  - **Priority**: Low

**What's Working**:
- Value-based map keys (50x faster than pointer keys)
- O(1) spatial queries
- Clean API

**Dependencies**:
- Imports: coords, ecs
- Imported by: common (as GlobalPositionSystem), combat

---

### rendering/

**Purpose**: Rendering components and logic

**Cohesion**: Good

**Issues Identified**:
- **Single File Package**: Only rendering.go
  - **Impact**: Could be merged with graphics package
  - **Recommendation**: Merge with graphics OR expand to include more rendering systems
  - **Priority**: Low

**Dependencies**:
- Imports: coords, ebiten
- Imported by: entitytemplates, worldmap, game_main

---

### graphics/

**Purpose**: Graphics utilities (shapes, color matrices)

**Cohesion**: Good - pure graphics utilities

**Issues Identified**:
- **None** - Clean utility package

**Dependencies**:
- Imports: ebiten
- Imported by: worldmap, squads

---

## Dependency Issues

### Circular Dependencies
- **combat <-> squads**: combat imports squads for abilities, squads imports combat for CombatFactionData
  - Recommendation: Extract ability system to break cycle

### Inappropriate Dependencies
| From | To | Issue | Recommendation |
|------|-----|-------|----------------|
| spawning | entitytemplates | Unclear ownership of entity creation | Clarify: entitytemplates owns creation, spawning owns timing |
| squads | entitytemplates | Units created via entitytemplates, monsters created directly | Unify entity creation path |
| common | systems | GlobalPositionSystem stored in common but defined in systems | Move PositionSystem to common OR keep in systems and import |

---

## ECS Architecture Assessment

### ECS Compliance by Package

**Exemplary (Follow These Patterns)**:
- squads/ - Perfect implementation
- gear/Inventory.go - Pure data component
- systems/positionsystem.go - Value-based keys, system logic

**Good**:
- combat/ - Clean component separation
- common/ - Proper EntityID usage

**Needs Improvement**:
- entitytemplates/ - Mixed data and creation logic
- spawning/ - Unclear responsibility ownership

### Key ECS Patterns Observed

**What's Working**:
- EntityID-only relationships (no entity pointers stored)
- Query-based access patterns
- Pure data components
- System functions for logic
- Value-based map keys

**Anti-Patterns Found**:
- None detected - codebase follows ECS best practices

---

## State Management Assessment

### Game State vs UI State

**Properly Separated**:
- BattleMapState (gui/core/) - UI selection and mode flags only
- OverworldState (gui/core/) - UI navigation state only
- Game state in ECS components (combat, squads, positions)

**Recommendation**: Maintain this pattern - never store game logic in UI state structures

---

## Prioritized Recommendations

### High Priority (Significant Impact)

1. **Clarify Entity Creation Ownership**
   - Packages: entitytemplates, spawning, squads
   - Change: Decide if entitytemplates owns ALL creation or if creation is distributed
   - Why: Eliminates confusion about where to add new entity types
   - Effort: 2-4 hours

2. **Narrow spawning Package Scope**
   - Package: spawning
   - Change: Focus on WHEN/WHERE to spawn, delegate creation to entitytemplates
   - Why: Clear separation of concerns, easier to understand spawn logic
   - Effort: 1-2 hours

### Medium Priority (Moderate Impact)

3. **Extract Player Data from common**
   - Package: common
   - Change: Move playerdata.go, playerresources.go to new `player` package
   - Why: Keep common package focused on ECS infrastructure
   - Effort: 1 hour

4. **Break combat <-> squads Circular Dependency**
   - Packages: combat, squads
   - Change: Extract ability triggering logic to abilities package OR move to combat
   - Why: Reduces coupling, improves testability
   - Effort: 2-3 hours

5. **Unify Unit Creation Path**
   - Packages: squads, entitytemplates
   - Change: Use entitytemplates for unit creation OR clearly separate "squad units" from "world monsters"
   - Why: Consistent entity creation patterns
   - Effort: 2 hours

### Low Priority (Nice to Have)

6. **Document gui/ Package Boundaries**
   - Package: gui
   - Change: Add package-level documentation clarifying widgets vs guicomponents
   - Why: Helps developers know where to add new UI code
   - Effort: 30 minutes

7. **Consolidate rendering and graphics**
   - Packages: rendering, graphics
   - Change: Merge into single package OR expand rendering with more systems
   - Why: Reduce package fragmentation
   - Effort: 1 hour

8. **Move PositionSystem Decision**
   - Packages: systems, common
   - Change: Either move PositionSystem to common OR keep in systems and clarify import pattern
   - Why: Clearer ownership of global systems
   - Effort: 30 minutes

---

## What NOT to Change

### Patterns to Preserve

1. **squads/ Package Structure** - Perfect ECS example, keep all patterns intact
2. **gear/Inventory.go Pure Data Component** - Textbook implementation
3. **EntityID-Only Relationships** - No entity pointers stored anywhere, maintain this
4. **GUI State Separation** - BattleMapState/OverworldState correctly separate from game state
5. **Generator Registry Pattern** - worldmap/ extensibility is excellent
6. **Input Coordinator Pattern** - Clean priority-based delegation
7. **Value-Based Map Keys** - systems/positionsystem.go performance optimization

### Areas That Look Problematic But Are Fine

- **Multiple gui/ Subpackages** - Appropriate for large UI codebase, just needs documentation
- **GlobalPositionSystem in common** - Correct pattern for O(1) spatial queries used everywhere
- **Command Pattern in squads/squadcommands/** - Good for undo/redo operations

---

## Implementation Roadmap

### Phase 1: High-Priority Refactoring (4-6 hours)
1. Clarify entity creation ownership
2. Narrow spawning package scope
3. Update documentation

### Phase 2: Medium-Priority Refactoring (5-7 hours)
1. Extract player package from common
2. Break combat/squads circular dependency
3. Unify unit creation path

### Phase 3: Low-Priority Cleanup (2-3 hours)
1. Document GUI package boundaries
2. Consolidate rendering packages
3. Clarify PositionSystem ownership

---

END OF ANALYSIS
