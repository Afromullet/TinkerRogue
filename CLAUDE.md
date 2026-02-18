# TinkerRogue Developer Guide

**Last Updated:** 2026-01-11

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
- `coords.CoordManager` - Coordinate conversions (use for ALL tile indexing)
- `common.GlobalPositionSystem` - O(1) spatial grid queries
- `common.EntityManager` - Passed as parameter (NOT global)

### Key Systems
- **ECS Core:** `common/ecsutil.go`, `common/commoncomponents.go`
- **Coordinates:** `world/coords/cordmanager.go`, ``world/coords/position.go`
- **Position:** `common/positionsystem.go`,  `coords/position.go`
- **Input:** `input/inputcoordinator.go` + controllers
- **Squads:** `tactical/squads`, `tactical/squadcommands`, `tactical/squadservices`,  (Squad and Squad Management)
- **Combat:** `tactical/combat/`, `tactical/combatservices/` (turn manager, combat state, combat actions)
- **AI:** `tactical/ai`, `tactical/behavior` (game ai, controls enemies)
- **GUI :** `gui/core/` (mode manager, context switching)
- **GUI Combat:** `gui/guicombat/` (combat mode, combat animation, attacking, moving)
- **GUI SQuad:** `gui/guisquads/` (Editing squads, purchasing units, deploying squads)
- **Items:** `gear/` (pure ECS inventory)
- **Graphics:** `visual/graphics`, `visual/rendering` (Game graphics and rendering. Batch drawing operations)
- **Worldmap:** `world/worldmap/` (generator registry, algorithms)

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
