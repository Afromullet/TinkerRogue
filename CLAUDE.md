# Project Configuration for Claude Code

## Build Commands
- Build: `go build -o game_main/game_main.exe game_main/*.go`
- Run: `go run game_main/*.go`
- Test: `go test ./...`
- Install dependencies: `go mod tidy`

## Development Notes
- Go-based roguelike game using Ebiten engine
- Main entry point: `game_main/main.go`
- Assets directory: `../assets/` (relative to game_main)

---

## Simplification Progress Summary

### ✅ Completed (8/8 items - 100% overall)
1. **Input System** - Unified InputCoordinator with specialized controllers
2. **Coordinate System** - Type-safe LogicalPosition/PixelPosition with CoordinateManager
3. **Entity Templates** - Generic factory pattern with EntityType enum (283 LOC)
4. **Graphics Shapes** - Consolidated 8+ types into BaseShape with 3 variants (390 LOC)
5. **Position System** - O(1) spatial grid, 50x performance improvement (399 LOC)
6. **Inventory System** - ECS refactor: EntityIDs, system functions, pure data (533 LOC) - 2025-10-21
7. **GUI Button Factory** - ButtonConfig pattern consistently applied throughout GUI package (2025-11-07)
8. **Worldmap Generator Strategy Pattern** - Pluggable map generation algorithms with rooms-and-corridors + BSP generators (2025-11-08)

### 🔄 In Progress (0/7 items)
(None - Status Effects moved to low priority)

### ❌ Remaining (0/7 items)
(All core simplification tasks complete!)

### 📋 Low Priority / Optional
- **Status Effects** - 85% complete, needs quality interface extraction (deferred)

---

## Squad System Status: 95% Complete (2675 LOC)

**Operational NOW:**
- ✅ All 8 ECS components with perfect data/logic separation
- ✅ Query system (7 functions) - Find units, check positions, get leaders
- ✅ Combat system - ExecuteSquadAttack with hit/dodge/crit/cover mechanics
- ✅ Visualization - Text-based 3x3 grid rendering
- ✅ Testing infrastructure - Comprehensive test suite exists
- ✅ **Ability System (317 LOC)** - Auto-triggering leader abilities fully integrated (2025-11-02)
  - 4 abilities: Rally, Heal, Battle Cry, Fireball
  - 5 trigger types: HP threshold, turn count, enemy count, morale, combat start
  - Integrated with turn manager and combat system
  - Cooldown and once-per-combat tracking

**Remaining Work (4-6 hours):**
- ⚠️ Formation Presets (4-6h) - Balanced/Defensive/Offensive/Ranged templates (stubs exist)

**Key Achievement:** Squad combat fully operational with abilities triggering automatically

**Timeline:** Formation presets are the final piece before 100% completion

---

## ECS Best Practices (Squad & Inventory System Templates)

The squad and inventory systems demonstrate perfect ECS architecture. Apply these patterns to all new code:

1. **Pure Data Components** - Zero logic methods, only data fields
2. **Native EntityID** - Use `ecs.EntityID` everywhere, not pointers
3. **Query-Based Relationships** - Discover via ECS queries, don't store references
4. **System-Based Logic** - All behavior in systems, not component methods
5. **Value Map Keys** - Use value-based keys for O(1) performance

**Anti-Patterns Fixed:**
- ✅ Position system: Pointer map keys → Value keys (50x faster)
- ✅ Legacy weapon/creature components removed (replaced by squad system)
- ✅ Inventory system: Entity pointers → EntityIDs, methods → system functions (2025-10-21)
- ✅ Item.Properties: `*ecs.Entity` → `ecs.EntityID` (2025-10-21)
- ✅ TileContents.entities: `[]*ecs.Entity` → `[]ecs.EntityID` (2025-11-08)
- ⚠️ Equipment system still uses entity pointers (scheduled for refactoring)

**Reference Implementations:**
- `squads/*.go` - Perfect ECS: 2675 LOC, 8 components, 7 query functions, system-based combat + abilities
  - `squadabilities.go` - 317 LOC, data-driven ability system with auto-triggers
  - `squadcombat.go` - 387 LOC, row-based combat with multi-cell units
  - `squadqueries.go` - 140 LOC, query functions for ECS data access
- `gear/Inventory.go` - Perfect ECS: 241 LOC, pure data component, 9 system functions
- `gear/items.go` - Perfect ECS: 177 LOC, EntityID-based relationships
- `gear/gearutil.go` - Perfect ECS: 115 LOC, query-based entity lookup

---

## Worldmap Generator System (2025-11-08)

**Status:** Complete - Strategy Pattern implementation with 2 algorithms

**Architecture:**
- `worldmap/generator.go` - MapGenerator interface, registry, GenerationResult
- `worldmap/gen_rooms_corridors.go` - Classic roguelike algorithm (default)
- `worldmap/gen_bsp.go` - Binary Space Partitioning for structured layouts
- `worldmap/GameMapUtil.go` - TileImageSet struct (no global variables)

**Usage:**
```go
// Default generator (rooms and corridors)
gameMap := worldmap.NewGameMapDefault()

// Specify algorithm
gameMap := worldmap.NewGameMap("bsp")

// List available generators
generators := worldmap.ListGenerators() // ["rooms_corridors", "bsp"]
```

**Benefits Achieved:**
- ✅ Add new algorithms without modifying existing code (Open/Closed Principle)
- ✅ Each generator independently testable
- ✅ Removed 180 LOC from GameMap (extracted to generators)
- ✅ Fixed global state issues (TileImageSet replaces global vars)
- ✅ Fixed ECS violation (TileContents uses EntityIDs, not entity pointers)

**Adding New Generators:**
1. Create `worldmap/gen_<name>.go`
2. Implement MapGenerator interface (3 methods)
3. Register in `init()` function
4. Zero changes to existing code required

**Reference:** See `analysis/worldmap_strategy_pattern_refactoring.md` for full implementation plan

---

## Current Implementation Priorities

### Ready to Implement (No Blockers)
- Bug fixes: throwable AOE, entity cleanup on death, wall collision
- Throwing accuracy/miss chance system
- Level transitions and tile variety
- Spawning system improvements (Entity Template System unblocked this)

### Blocked/Waiting
- **Squad formations** (4-6h remaining) - Formation presets need implementation
- **Balance/difficulty** (needs formation presets complete)
- **Map integration for squads** (Complete! Squads already deployed and functional on map)

---

## Analysis Files Reference

**Primary:** `analysis/MASTER_ROADMAP.md` v3.0 (2025-10-12)
- Executive summary: 16-24 hours remaining for squad completion
- Phase breakdown with actual LOC counts from codebase audit
- Critical discoveries: Position/Query/Combat/Visualization already complete

**Supporting:**
- `squad_system_final.md` - Architecture details (components, abilities, targeting)
- `combat_refactoring.md` - OBSOLETE (combat implemented)
- `roadmap_completion.md` - OBSOLETE (superseded by v3.0)

**Note:** v3.0 based on actual file analysis, not estimates.
