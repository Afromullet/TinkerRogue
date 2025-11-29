# Codebase Architecture Analysis
Generated: 2025-11-28
Scope: Full Codebase

---

## Executive Summary

### Overall Assessment
The TinkerRogue codebase demonstrates strong ECS fundamentals with good separation between data (components) and logic (systems), but suffers from **scattered entity creation responsibilities** and **service layer pattern misuse**. The core architecture (squads, combat, coords) is solid, while entity spawning and GUI boundaries need consolidation.

### High-Value Refactoring Targets
1. **Entity Creation Scattered Across 3+ Packages**: `spawning/`, `entitytemplates/`, and `squads/squadcreation.go` all create entities - consolidate into single source of truth
2. **Service Layer Adds Unnecessary Indirection**: `squadservices/`, `combatservices/`, and `gear/inventory_service.go` wrap existing system functions with no added value
3. **GUI Package Violates Layer Boundaries**: GUI modes directly manipulate game state instead of going through combat/squad systems

### What's Working Well
- **squads/**: Exemplary ECS implementation - pure data components, query-based relationships, system functions
- **coords/**: Clean coordinate system abstraction with global CoordManager
- **combat/**: Good separation of turn management, faction tracking, and action state
- **common/ecsutil.go**: Type-safe component access helpers prevent common mistakes

---

## Package Analysis

### common/

**Purpose**: Core ECS infrastructure and shared components (Position, Attributes, Name)

**Cohesion**: Good - focused on ECS utilities and fundamental game components

**Issues Identified**: None significant

**Dependencies**:
- Imports: `coords`, `systems`, `github.com/bytearena/ecs`
- Imported by: Every package
- Issues: None - appropriate as foundation package

---

### coords/

**Purpose**: Coordinate system management (Logical, Pixel, Screen positions and conversions)

**Cohesion**: Excellent - single responsibility (coordinate math)

**Issues Identified**: None

**Dependencies**:
- Imports: None (leaf package)
- Imported by: Most packages
- Issues: None - clean abstraction

---

### squads/

**Purpose**: Squad-based tactical combat ECS implementation

**Cohesion**: Excellent - all 8 components serve squad/unit management

**Issues Identified**:
- **Mixed Responsibilities in squadcreation.go**: Contains both high-level functions (`CreateSquadFromTemplate`) and low-level helpers (`CreateEmptySquad`, `AddUnitToSquad`). The low-level functions duplicate logic found in `squadservices/squad_service.go`.

**Recommendation**:
- Keep `CreateSquadFromTemplate` (complex multi-step initialization)
- Remove `CreateEmptySquad` and `AddUnitToSquad` - these are thin wrappers that duplicate service layer

**Dependencies**:
- Imports: `common`, `coords`, `entitytemplates`, `github.com/bytearena/ecs`
- Imported by: `combat`, `gui`, `spawning`, `game_main`
- Issues: None

---

### squads/squadservices/

**Purpose**: Service layer wrapping squad system functions

**Cohesion**: Poor - unnecessary abstraction layer

**Issues Identified**:
- **No Added Value**: Functions like `SquadService.CreateSquad()` and `SquadService.AddUnitToSquad()` are thin wrappers around `squads.CreateEmptySquad()` and `squads.AddUnitToSquad()`
- **Result Structs Add Verbosity**: `CreateSquadResult`, `AddUnitResult`, etc. add ~50 lines per operation with no meaningful benefit
- **Not Actually Used**: GUI modes still call `squads.` functions directly, bypassing services

**Impact**: Maintenance burden - two parallel APIs for same operations

**Recommendation**: **Remove entire `squadservices/` package**
- GUI should call `squads.CreateSquadFromTemplate()`, `squads.AddUnitToSquad()`, etc. directly
- If result tracking needed, add to squad system functions themselves
- Keep only `squad_deployment_service.go` if it contains actual business logic (need to verify)

**Files to Remove**:
```
squads/squadservices/squad_service.go         (210 lines of wrapper code)
squads/squadservices/squad_builder_service.go (similar wrapper pattern)
```

---

### combat/

**Purpose**: Turn-based combat system (turn management, factions, action states)

**Cohesion**: Good - focused on combat mechanics

**Issues Identified**:
- **MapPositionComponent Duplicates PositionComponent**: Both `combat.MapPositionData` and `common.PositionComponent` store squad positions. MapPositionData adds FactionID but creates sync burden.

**Recommendation**:
- Add `FactionID` field to `squads.SquadData` component
- Remove `MapPositionComponent` entirely
- Use `common.PositionComponent` + squad's embedded FactionID

**Before**:
```go
type MapPositionData struct {
    SquadID   ecs.EntityID
    Position  coords.LogicalPosition // Duplicate!
    FactionID ecs.EntityID
}
```

**After**:
```go
type SquadData struct {
    SquadID       ecs.EntityID
    FactionID     ecs.EntityID  // Add this
    // ... other fields
}
// Use common.PositionComponent for position
```

**Dependencies**:
- Imports: `common`, `coords`, `squads`, `worldmap`
- Imported by: `gui`, `combat/combatservices`, `game_main`
- Issues: None

---

### combat/combatservices/

**Purpose**: Service layer wrapping combat systems

**Cohesion**: Mixed - some functions add value, others don't

**Issues Identified**:
- **Partial Wrapper Syndrome**: `CombatService` wraps `TurnManager`, `FactionManager`, and `CombatMovementSystem` but GUI still accesses those directly
- **Result Structs Inconsistently Used**: `AttackResult`, `MoveSquadResult` used in tests but GUI uses direct combat functions

**Recommendation**: **Simplify to Facade Pattern**
- Keep `CombatService` as a convenience facade (one-stop access to combat systems)
- Remove result structs - return errors directly
- Remove thin wrapper functions - expose underlying systems

**Before**:
```go
func (cs *CombatService) ExecuteSquadAttack(attackerID, targetID ecs.EntityID) *AttackResult {
    combatSys := combat.NewCombatActionSystem(cs.entityManager)
    reason, canAttack := combatSys.CanSquadAttackWithReason(attackerID, targetID)
    // ... 20 lines wrapping existing function
}
```

**After**:
```go
func (cs *CombatService) GetActionSystem() *combat.CombatActionSystem {
    return combat.NewCombatActionSystem(cs.entityManager)
}
// GUI calls: combatService.GetActionSystem().ExecuteAttackAction(...)
```

**Dependencies**:
- Imports: `combat`, `common`, `coords`, `squads`
- Imported by: `gui`, tests
- Issues: Creates parallel API that GUI doesn't use

---

### spawning/

**Purpose**: Entity spawning logic (monsters, loot, throwables)

**Cohesion**: Poor - mixed responsibilities

**Issues Identified**:
- **Entity Creation Split Across Packages**: `spawning/spawnmonsters.go` calls `entitytemplates.CreateEntityFromTemplate()`, creating a 3-level dependency chain
- **Spawning Logic vs Entity Creation**: Package mixes "when/where to spawn" (spawning logic) with "how to create" (entity construction)
- **Direct GameMap Manipulation**: `gm.Tiles[index].Blocked = true` scattered across spawn functions

**Recommendation**: **Consolidate Entity Creation**
- Move all entity construction to `entitytemplates/`
- Keep only spawn *logic* in `spawning/` (probability tables, spawn timing, location selection)
- Spawn functions should call `entitytemplates.CreateMonster(...)` or `entitytemplates.CreateLoot(...)`

**Example Refactor**:

**Before** (spawning/spawnmonsters.go):
```go
func SpawnMonster(ecsmanager common.EntityManager, gm *worldmap.GameMap) {
    // Spawn logic mixed with entity creation
    entitytemplates.CreateEntityFromTemplate(ecsmanager, entitytemplates.EntityConfig{
        Type: entitytemplates.EntityCreature,
        Name: entitytemplates.MonsterTemplates[0].Name,
        // ... 10 more config fields
    }, entitytemplates.MonsterTemplates[0])
    gm.Tiles[index].Blocked = true
}
```

**After**:
```go
// spawning/spawnmonsters.go - only spawn logic
func SpawnMonster(ecsmanager common.EntityManager, gm *worldmap.GameMap) {
    pos := selectSpawnPosition(gm) // Spawn logic
    entitytemplates.CreateMonsterAtPosition(ecsmanager, gm, pos, monsterType)
}

// entitytemplates/monsters.go - entity creation
func CreateMonsterAtPosition(mgr common.EntityManager, gm *worldmap.GameMap, pos coords.LogicalPosition, template JSONMonster) {
    // All entity construction logic here
    entity := createMonsterEntity(mgr, template)
    addComponentsToMonster(entity, template)
    common.GlobalPositionSystem.AddEntity(entity.GetID(), pos)
    gm.Tiles[coords.CoordManager.LogicalToIndex(pos)].Blocked = true
}
```

**Dependencies**:
- Imports: `common`, `coords`, `entitytemplates`, `worldmap`
- Imported by: `game_main`
- Issues: Creates unnecessary 3-layer dependency (`spawning` -> `entitytemplates` -> entity creation)

---

### entitytemplates/

**Purpose**: Template-based entity creation from JSON data

**Cohesion**: Good - focused on entity construction

**Issues Identified**:
- **Scope Creep**: Package currently handles both template *definition* (JSON structs) and entity *construction* (factory functions)
- **createBaseEntity is Too Generic**: Function signature with `imagePath, assetDir, visible` suggests it's trying to handle all entity types uniformly

**Recommendation**:
- Split into `entitytemplates/` (data definitions) and `entitycreation/` (factory functions)
- OR keep unified but create type-specific constructors: `CreateMonster()`, `CreateItem()`, `CreateSquad()`

**Dependencies**:
- Imports: `common`, `coords`, `rendering`, `worldmap`
- Imported by: `spawning`, `squads`, `game_main`
- Issues: Bidirectional dependency with `spawning` (both call each other)

---

### gear/

**Purpose**: Item and inventory system (pure ECS implementation)

**Cohesion**: Good - focused on items/inventory

**Issues Identified**:
- **gear/inventory_service.go** - Same service layer anti-pattern as squads/combat

**Recommendation**: **Remove inventory_service.go**
- `SelectThrowable()` is thin wrapper around existing functions
- `GetInventoryItemInfo()` just formats existing data
- GUI should call `gear.GetItemByID()`, `gear.GetItemEffectNames()` directly

**Dependencies**:
- Imports: `common`, `github.com/bytearena/ecs`
- Imported by: `gui`, `spawning`
- Issues: None

---

### gui/

**Purpose**: All GUI and mode management

**Cohesion**: Mixed - combines mode framework (good) with mode implementations (scattered)

**Issues Identified**:
- **GUI Modes Directly Manipulate Game State**: `guicombat/combatmode.go` and `guisquads/squadmanagementmode.go` call combat/squad systems directly instead of going through services
- **Scattered Subpackages**: `gui/core/`, `gui/widgets/`, `gui/guicombat/`, `gui/guisquads/`, `gui/guimodes/`, `gui/guicomponents/` - hard to understand organization
- **BattleMapState vs Combat State Confusion**: `gui/core/contextstate.go` defines UI state that duplicates combat system state

**Recommendation**: **Consolidate GUI Organization**

**Current Structure**:
```
gui/
├── core/              (mode framework)
├── widgets/           (reusable UI components)
├── guicombat/        (combat mode)
├── guisquads/        (squad modes)
├── guimodes/         (other modes)
└── guicomponents/    (query helpers)
```

**Proposed Structure**:
```
gui/
├── framework/         (mode manager, UIContext, input state)
├── widgets/           (buttons, panels, layouts)
├── modes/
│   ├── combat.go
│   ├── squads.go
│   ├── inventory.go
│   └── exploration.go
└── queries/           (rename guicomponents -> queries)
```

**Why**: Reduces nesting, clarifies that modes are peers

**Dependencies**:
- Imports: Everything (combat, squads, gear, coords, common, worldmap)
- Imported by: `game_main`
- Issues: **Circular dependency risk** - GUI imports game packages, some game packages expose GUI-specific data

---

### gui/guicomponents/

**Purpose**: Centralized ECS queries for UI (GUIQueries service)

**Cohesion**: Excellent - single responsibility (query aggregation)

**Issues Identified**: None - this is **good architecture**

**Why This Works**:
- Provides read-only aggregated queries (`GetSquadInfo`, `GetFactionInfo`)
- Eliminates query duplication across modes
- Clean separation: GUIQueries reads, services/systems write

**Recommendation**: Keep as-is, consider renaming package to `gui/queries` for clarity

---

### worldmap/

**Purpose**: Map generation algorithms and tile management

**Cohesion**: Good - focused on world/map structure

**Issues Identified**:
- **Generator Registration Mechanism**: `RegisterGenerator()` uses `init()` functions, which is appropriate

**Dependencies**:
- Imports: `coords`, `common`
- Imported by: `game_main`, `spawning`, `combat`
- Issues: None

---

### systems/

**Purpose**: Cross-cutting game systems (currently only PositionSystem)

**Cohesion**: Good (but underpopulated)

**Issues Identified**: None

**Observation**: Only contains `positionsystem.go` - other "systems" are in domain packages (combat, squads)

**Recommendation**: Consider moving turn/faction management here if they become cross-domain

---

### input/

**Purpose**: Input handling coordination

**Cohesion**: Good - focused on input processing

**Issues Identified**: None

**Dependencies**:
- Imports: `common`, `coords`, `worldmap`
- Imported by: `game_main`
- Issues: None

---

### game_main/

**Purpose**: Application entry point and initialization

**Cohesion**: Good - startup and wiring

**Issues Identified**: None

---

## Dependency Issues

### Circular Dependencies
**None detected** - clean unidirectional flow

### Inappropriate Dependencies

| From | To | Issue | Recommendation |
|------|-----|-------|----------------|
| spawning | entitytemplates | 3-level entity creation chain | Consolidate all entity creation in entitytemplates |
| gui/* | combat, squads, gear | GUI manipulates game state directly | Use command pattern or event system |
| combat | squads | MapPositionComponent duplicates PositionComponent | Remove MapPositionComponent |

---

## Game Architecture Assessment

### ECS Organization
**Rating**: Excellent (9/10)

**Strengths**:
- Pure data components (SquadData, CombatData, GridPositionData)
- Query-based relationships (no entity pointer caching)
- System functions properly separated from components
- Value-based map keys in PositionSystem

**Issues**:
- Minor: Some duplication between PositionComponent and MapPositionComponent
- Minor: Service layers add indirection without value

### State Management
**Rating**: Good (7/10)

**Strengths**:
- Clear separation: `BattleMapState` (UI) vs `TurnStateData` (game)
- Context switching properly saves/restores state
- GlobalPositionSystem provides O(1) lookups

**Issues**:
- BattleMapState.ValidMoveTiles duplicates data from MovementSystem
- GUI modes cache game state instead of querying systems

### Input Flow
**Rating**: Good (8/10)

**Strengths**:
- Clean priority hierarchy: UI -> Combat -> Movement
- InputCoordinator pattern prevents conflicts
- Controller interfaces allow extensibility

**Issues**: None significant

---

## Prioritized Recommendations

### High Priority (Significant Impact)

1. **Consolidate Entity Creation**
   - Package(s): `spawning/`, `entitytemplates/`, `squads/squadcreation.go`
   - Change: Move all entity construction to `entitytemplates/` with type-specific factories
   - Why: Eliminates 3-level dependency chain, single source of truth for entity creation
   - Effort: Medium (2-3 days)

2. **Remove Service Layer Wrappers**
   - Package(s): `squads/squadservices/`, `combat/combatservices/`, `gear/inventory_service.go`
   - Change: Delete service files, have GUI call system functions directly
   - Why: Reduces maintenance burden (210+ lines of wrapper code), eliminates duplicate APIs
   - Effort: Low (4-6 hours - mostly delete code and update call sites)

3. **Eliminate MapPositionComponent**
   - Package(s): `combat/combatcomponents.go`, `squads/squadcomponents.go`
   - Change: Add FactionID to SquadData, use common.PositionComponent
   - Why: Removes data duplication, simplifies position tracking
   - Effort: Medium (1-2 days - requires careful component migration)
   - **Signature Change**:
     ```go
     // Before
     type MapPositionData struct {
         SquadID   ecs.EntityID
         Position  coords.LogicalPosition
         FactionID ecs.EntityID
     }

     // After (add to SquadData)
     type SquadData struct {
         SquadID   ecs.EntityID
         FactionID ecs.EntityID  // NEW
         // ... existing fields
     }
     // Use common.PositionComponent for position
     ```

### Medium Priority (Moderate Impact)

4. **Reorganize GUI Package Structure**
   - Package(s): `gui/core/`, `gui/guicombat/`, `gui/guisquads/`, `gui/guimodes/`
   - Change: Flatten to `gui/framework/`, `gui/modes/`, `gui/widgets/`, `gui/queries/`
   - Why: Reduces nesting, clarifies peer relationships
   - Effort: Low (4-6 hours - mostly directory moves and import updates)

5. **Split Entity Templates**
   - Package(s): `entitytemplates/`
   - Change: Separate data definitions from factory functions
   - Why: Clarifies package responsibility (data vs construction)
   - Effort: Medium (1-2 days)

### Low Priority (Nice to Have)

6. **Extract Spawn Logic from Entity Creation**
   - Package(s): `spawning/`
   - Change: Keep probability/timing logic, delegate entity creation to `entitytemplates/`
   - Why: Clarifies "when/where" vs "how" responsibilities
   - Effort: Low (4-6 hours)

---

## What NOT to Change

### Leave These Alone (They're Working Well)

1. **squads/ ECS Implementation**
   - Why: Perfect example of ECS best practices - pure data components, query-based, system functions
   - Don't: Try to "improve" by adding OOP patterns or inheritance

2. **coords/ Coordinate System**
   - Why: Clean abstraction, well-tested, single responsibility
   - Don't: Merge into common or split into smaller packages

3. **gui/guicomponents/GUIQueries**
   - Why: Excellent read-only query aggregation pattern
   - Don't: Add write operations or turn into service layer

4. **common/ecsutil.go Helper Functions**
   - Why: Type-safe component access prevents common ECS mistakes
   - Don't: Remove in favor of "simpler" direct access

5. **combat/ Turn and Faction Management**
   - Why: Clear separation of concerns, testable, extensible
   - Don't: Merge into single "game manager" class

6. **worldmap/ Generator Registry**
   - Why: Proper use of init() for plugin-style architecture
   - Don't: Replace with manual registration or reflection

---

## Architecture Patterns Assessment

### What's Applied Well

| Pattern | Where Used | Why It Works |
|---------|------------|--------------|
| **ECS** | squads, combat, gear | Pure data components, query-based relationships |
| **Facade** | gui/guicomponents/GUIQueries | Aggregates complex queries into simple API |
| **Registry** | worldmap/generator.go | Clean plugin architecture for generators |
| **Coordinator** | input/inputcoordinator.go | Priority-based input handling |

### What's Misapplied

| Pattern | Where Misused | Problem | Fix |
|---------|---------------|---------|-----|
| **Service Layer** | squadservices, combatservices | Adds indirection without value | Remove - use system functions directly |
| **Factory** | entitytemplates/creators.go | Too generic, handles all entity types uniformly | Create type-specific factories |
| **State** | gui/core/contextstate.go | Duplicates game state in UI state | Cache only UI-specific state, query for game state |

---

## Next Steps

### Immediate Actions (This Week)
1. Remove `squads/squadservices/squad_service.go` and update GUI to call `squads.*` directly
2. Remove `gear/inventory_service.go` and update GUI to call `gear.*` directly
3. Document the consolidation plan for entity creation

### Short Term (Next Sprint)
4. Consolidate entity creation into `entitytemplates/` with type-specific functions
5. Add FactionID to SquadData and migrate away from MapPositionComponent
6. Reorganize GUI package structure

### Long Term (Next Month)
7. Review if combat/combatservices should be simplified or removed
8. Consider extracting common patterns into `systems/` if they become cross-domain
9. Performance profiling of PositionSystem and component queries

---

END OF ANALYSIS
