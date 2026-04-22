# TinkerRogue Developer Guide

**Last Updated:** 2026-04-21

Quick reference for working with TinkerRogue. For detailed ECS patterns, see `docs/project_documentation/ECS_BEST_PRACTICES.md`.

---

## Quick Commands

```bash
# Build and run
go build -o game_main/game_main.exe game_main/*.go && ./game_main/game_main.exe

# Test
go test ./...                    # All tests
go test ./squads/...            # Specific package
go test -v ./...                # Verbose
go test -cover ./...            # With coverage

# Maintenance
go mod tidy                     # Update dependencies
go fmt ./...                    # Format code
go vet ./...                    # Check for mistakes
```

---

## Core Architecture

### Global Instances
- `core/coords.CoordManager` - Coordinate conversions (use for ALL tile indexing)
- `core/common.GlobalPositionSystem` - O(1) spatial grid queries
- `core/common.EntityManager` - Passed as parameter (NOT global)

### Key Systems
- **ECS Core:** `core/common/ecsutil.go`, `core/common/commoncomponents.go`
- **Coordinates:** `core/coords/cordmanager.go`, `core/coords/position.go`
- **Position:** `core/common/positionsystem.go`, `core/coords/position.go`
- **Config:** `core/config/` (Global game config constants and flags)
- **Input:** `input/cameracontroller.go` (Camera and input control)
- **Squads:** `tactical/squads/` (Squad and squad member management)
- **Commander:** `tactical/commander/` (Roster, turn state, movement commands)
- **Combat:** `tactical/combat/` (Turn manager, combat state, combat actions)
- **Power Core:** `tactical/powers/powercore/` (Shared power pipeline, context, logging)
- **Spells:** `tactical/powers/spells/` (Spell casting system)
- **Effects:** `tactical/powers/effects/` (Effect system, used by spells and artifacts)
- **Artifacts:** `tactical/powers/artifacts/` (Artifact inventory, behaviors, charges)
- **Perks:** `tactical/powers/perks/` (Perk registry, behaviors, hooks, dispatcher)
- **Progression:** `tactical/powers/progression/` (Progression components and perk/spell library)
- **AI:** `mind/ai/`, `mind/behavior/`, `mind/evaluation/` (Game AI, controls enemies)
- **Combat Lifecycle:** `mind/combatlifecycle/` (Combat setup, cleanup, rewards, casualties)
- **Encounter Generation:** `mind/encounter/` (Random encounter generation)
- **Spawning:** `mind/spawning/` (Automatic Squad creation and composition)
- **Overworld:** `campaign/overworld/` (Factions, garrisons, influence, threats, victory conditions)
- **Raid:** `campaign/raid/` (Raid encounters, floor graph, deployment, recovery)
- **Templates:** `templates/` (Entity factory, artifact/spell definitions, game config, name generation)
- **GUI Framework:** `gui/framework/`, `gui/specs/` (Mode manager, context switching, layout specs)
- **GUI Widgets:** `gui/builders/`, `gui/widgetresources/`, `gui/widgets/` (Reusable GUI elements)
- **GUI Combat:** `gui/guicombat/` (Combat mode, combat animation, attacking, moving)
- **GUI Squads:** `gui/guisquads/` (Editing squads, purchasing units, deploying squads)
- **GUI Overworld:** `gui/guioverworld/` (Overworld mode, actions, rendering)
- **GUI Raid:** `gui/guiraid/` (Raid mode, deploy panel, floor map)
- **GUI Exploration:** `gui/guiexploration/` (Exploration mode)
- **GUI Node Placement:** `gui/guinodeplacement/` (Node placement mode and rendering)
- **GUI Artifacts:** `gui/guiartifacts/` (Artifact panel and handlers)
- **GUI Spells:** `gui/guispells/` (Spell panel and handlers)
- **GUI Progression:** `gui/guiprogression/` (Progression mode, panels, controller)
- **GUI Inspect:** `gui/guiinspect/` (Unit inspection panel)
- **GUI Unit View:** `gui/guiunitview/` (Unit view mode)
- **GUI Start Menu:** `gui/guistartmenu/` (Start menu)
- **Graphics:** `visual/graphics/`, `visual/rendering/` (Game graphics and batch drawing)
- **Map Rendering:** `visual/maprender/` (Map and tile rendering)
- **Combat Rendering:** `visual/combatrender/` (Squad renderer, combat overlays, highlights)
- **VFX:** `visual/vfx/` (Visual effects, animators, renderers)
- **Worldgen:** `world/worldgen/` (Map generator algorithms and registry)
- **Worldmap Core:** `world/worldmapcore/` (Dungeon tiles, biomes, generator interface)
- **Garrison Gen:** `world/garrisongen/` (Garrison map generation, DAG, terrain)
- **Setup:** `setup/gamesetup/`, `setup/savesystem/` (Booting, save system)
- **Testing:** `testing/` (Test fixtures and bootstrap utilities)

---

## ECS Quick Reference

**Core Principles** (see `docs/project_documentation/ECS_BEST_PRACTICES.md` for details):

1. **Pure Data Components** - Zero logic, only fields
2. **EntityID Only** - Never store `*ecs.Entity` pointers
3. **Query-Based** - Don't cache relationships
4. **System Functions** - Logic outside components
5. **Value Map Keys** - Not pointer keys (50x faster)

### File Structure
```
package_name/
├── components.go      # Data definitions only
├── *queries.go        # Search/filter functions
├── *system.go         # Logic/behavior functions
└── *_test.go         # Tests
```

### Naming
- Data structs: `SquadData`, `ActionStateData`
- Components: `SquadComponent`, `ActionStateComponent`
- Tags: `SquadTag`, `SquadMemberTag`
- Queries: `GetSquad*`, `Find*`, `Is*`, `Can*`
- Systems: `ExecuteAttack`, `Update*`, `Create*`

### Common Patterns
```go
// Component access from entity (preferred when you have entity)
data := common.GetComponentType[*SquadData](entity, SquadComponent)

// Component access by ID (when you only have entityID)
data := common.GetComponentTypeByID[*SquadData](manager, entityID, component)

// Check if entity has component
if manager.HasComponent(entityID, SquadComponent) {
    // Component exists
}

// Get component with existence check
if componentData, ok := manager.GetComponent(entityID, SquadComponent); ok {
    data := componentData.(*SquadData)
}

// Query pattern (preferred for iteration)
for _, result := range manager.World.Query(SquadTag) {
    entity := result.Entity
    data := common.GetComponentType[*SquadData](entity, SquadComponent)
}

// Find entity by ID (when you need entity pointer)
entity := manager.FindEntityByID(entityID)

// Move entity (atomically updates position component + position system)
manager.MoveEntity(entityID, entity, oldPos, newPos)

// Move squad and all members together
manager.MoveSquadAndMembers(squadID, squadEntity, unitIDs, oldPos, newPos)

// Position system (direct access when needed)
common.GlobalPositionSystem.AddEntity(entityID, logicalPos)
common.GlobalPositionSystem.RemoveEntity(entityID, logicalPos)
entityIDs := common.GlobalPositionSystem.GetEntitiesAtPosition(logicalPos)
```

### Component Access Patterns

Use the pattern that best fits your situation:

**Pattern 1: From Query Result (Preferred)**

Use when you already have the entity from a query.

```go
for _, result := range manager.World.Query(SquadTag) {
    entity := result.Entity
    data := common.GetComponentType[*SquadData](entity, SquadComponent)
}
```

**Pattern 2: By EntityID**

Use when you only have the EntityID:

```go
// Direct component access (preferred)
data := common.GetComponentTypeByID[*DataType](manager, entityID, SquadComponent)

// Or get entity pointer first (when you need the entity for multiple operations)
entity := manager.FindEntityByID(entityID)
if entity != nil {
    data := common.GetComponentType[*DataType](entity, SquadComponent)
}

// With existence check
if componentData, ok := manager.GetComponent(entityID, SquadComponent); ok {
    data := componentData.(*DataType)
}
```

**Reference Implementations:**
- `squads/` - Perfect ECS example
- `gear/Inventory.go` - Pure ECS component
- `systems/positionsystem.go` - Value-based map keys

### Subsystem Registration

Use the self-registration pattern for initializing ECS subsystems:

```go
// In your subsystem package (e.g., squads/init.go)
func init() {
    common.RegisterSubsystem(func(em *common.EntityManager) {
        // Initialize components
        SquadComponent = em.World.NewComponent()
        SquadMemberComponent = em.World.NewComponent()

        // Initialize tags
        SquadTag = em.World.NewTag(SquadComponent)
        SquadMemberTag = em.World.NewTag(SquadMemberComponent)
    })
}

// In main, after creating EntityManager
manager := common.NewEntityManager()
common.InitializeSubsystems(manager)  // Calls all registered init functions
```

This pattern ensures subsystems initialize themselves without manual coordination.

---

## Critical Warnings

### ⚠️ CoordinateManager Indexing (CRITICAL)

**ALWAYS use `coords.CoordManager.LogicalToIndex()` for tile arrays:**

```go
// ✅ CORRECT
tileIdx := coords.CoordManager.LogicalToIndex(logicalPos)
result.Tiles[tileIdx] = &tile

// ❌ WRONG - Causes index out of range panics
idx := y*width + x  // Width may differ from CoordManager.dungeonWidth!
result.Tiles[idx] = &tile
```

**Why:** `CoordinateManager.dungeonWidth` may not match function parameters. Manual calculation creates wrong indices.

### ⚠️ Entity Lifecycle

**Removing entities with positions:**
```go
pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
if pos != nil {
    manager.CleanDisposeEntity(entity, pos)  // Removes from position system + ECS
} else {
    manager.World.DisposeEntities(entity)
}
```

**Moving entities:**
```go
// Single entity - atomically updates component + position system
manager.MoveEntity(entityID, entity, oldPos, newPos)

// Squad with members - moves all units together
manager.MoveSquadAndMembers(squadID, squadEntity, unitIDs, oldPos, newPos)
```

**Manual cleanup (if not using helpers):**
1. Remove from `GlobalPositionSystem.RemoveEntity(entityID, position)`
2. Remove from all other systems
3. Call `manager.World.DisposeEntities(entity)`

### ⚠️ GUI State Separation

- `BattleMapState` / `OverworldState` = ONLY UI state (selection, mode flags)
- Game state = ECS components (combat, squads, positions)
- Never store game logic in UI state structures

### ⚠️ Generator Registration

New worldmap generators must register in `init()`:
```go
func init() {
    RegisterGenerator("my_algorithm", &MyGenerator{})
}
```

---

## Code Style

### Go Conventions
- `camelCase` for private, `PascalCase` for public
- Package names: lowercase, single word
- Run `go fmt ./...` before committing

### ECS Conventions
- Components: Pure data, no methods
- Queries: Read-only functions in `*queries.go`
- Systems: Logic functions in `*system.go`
- Always use `ecs.EntityID`, never `*ecs.Entity`

### Comments
```go
// Public functions: Document purpose and return values
// GetSquadEntity finds squad by ID. Returns nil if not found.
func GetSquadEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity

// Complex logic: Explain WHY, not WHAT
// Use CoordinateManager to prevent index out of bounds

// TODOs: Include context
// TODO: Add formation validation (30min)
```

---

## Development Workflow

### Before Coding
1. Read existing implementation
2. Check `docs/project_documentation/ECS_BEST_PRACTICES.md`
3. Search for similar patterns
4. Consider entity lifecycle impact

### Adding Features
1. Design components (pure data)
2. Create query functions
3. Implement system functions
4. Write tests
5. Integrate with existing systems

### Documentation Maintenance

**When adding, removing, or moving source files or packages, update:**
- `resources/docs/PROJECT_LAYOUT.md` — directory/package layout overview
- `resources/docs/PACKAGE_DEPENDENCIES.md` — package dependency graph

Keep these in sync with the code in the same change. Also update the **Key Systems** list above in this file when a new top-level package is introduced or an existing one is renamed/relocated.

### Code Review Checklist
- [ ] No logic in components
- [ ] Uses `ecs.EntityID` not entity pointers
- [ ] Query-based relationships (no caching)
- [ ] Logic in system functions
- [ ] Uses `CoordManager.LogicalToIndex()` for tiles
- [ ] Proper entity cleanup
- [ ] Naming conventions followed
- [ ] Tests included

---

## Resources

- **Detailed ECS Guide:** `docs/project_documentation/ECS_BEST_PRACTICES.md`
- **Tests:** Run `go test ./...` frequently
- **Reference Code:** Study `squads/`, `gear/Inventory.go`, `systems/positionsystem.go`
