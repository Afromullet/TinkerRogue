# Critical Patterns & Gotchas

Supplementary to CLAUDE.md's Critical Warnings section. These are patterns found via codebase audit that can cause silent bugs, hard-to-debug issues, or initialization failures.

---

## High Priority — Silent Bugs, Hard to Debug

### 1. GetComponentType Swallows Panics

**File:** `common/ecsutil.go:60-72`

`GetComponentType` has a `defer recover()` that catches **any** panic and silently returns the zero value. If you pass the wrong component constant, misspell a type assertion, or hit a nil pointer inside the ECS library, you get `nil` back with no error and no log.

```go
// common/ecsutil.go
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {
    defer func() {
        if r := recover(); r != nil {
            // ERROR HANDLING IN FUTURE  ← swallows everything
        }
    }()
    if c, ok := entity.GetComponentData(component); ok {
        return c.(T)
    }
    var nilValue T
    return nilValue
}
```

**Impact:** Wrong component lookups silently return nil/zero. Code continues with bad data instead of failing fast.

**Mitigation:** Always nil-check the return value before using it. If something behaves as though a component doesn't exist, suspect a wrong component constant before suspecting missing data.

```go
// CORRECT — always check
data := common.GetComponentType[*SquadData](entity, SquadComponent)
if data == nil {
    // Handle missing component explicitly
    return
}

// WRONG — assumes component exists
data := common.GetComponentType[*SquadData](entity, SquadComponent)
data.Name = "foo"  // nil pointer dereference if component missing
```

---

### 2. attr.MaxHealth Cached Field Desync

**File:** `common/commoncomponents.go:37-38, 56, 77-79`

`Attributes.MaxHealth` is a cached field derived from `GetMaxHealth()` (which computes `20 + Strength*2`). If Strength changes after construction (effects, leveling, equipment) and `MaxHealth` isn't recached, callers reading `attr.MaxHealth` get stale values.

The codebase is split — some callers use `attr.MaxHealth` (the cached field), others use `attr.GetMaxHealth()` (the live calculation):

```go
// Uses cached field (stale if Strength changed):
attr.CurrentHealth > attr.MaxHealth           // squadabilities.go:176
heal := attr.MaxHealth * hpPercent / 100      // combatpipeline/cleanup.go:19
MaxHP: attr.MaxHealth                         // gui/builders/lists.go:112

// Uses live calculation (always correct):
maxHP := attr.GetMaxHealth()                  // mind/evaluation/power.go:74
missingHP := targetAttr.GetMaxHealth() - ...  // squadcombat.go:945
```

**Impact:** After stat modifications (buffs, level-ups, equipment), HP caps, healing limits, and UI displays may use the old MaxHealth value.

**Mitigation:** After any Strength change, recache the field:

```go
attr.Strength += bonus
attr.MaxHealth = attr.GetMaxHealth()  // Must recache!
```

The `experience.go:94` leveling code already does this correctly. Other stat-changing paths should follow the same pattern. When in doubt, prefer `attr.GetMaxHealth()` over `attr.MaxHealth`.

---

### 3. Position Dual-State Invariant

**File:** `common/ecsutil.go:99-126`

Entity positions are stored in **two** places that must stay in sync:
1. The entity's `PositionComponent` (a `*coords.LogicalPosition`)
2. The `GlobalPositionSystem` spatial grid

Only `manager.MoveEntity()` updates both atomically. Modifying either one directly causes desync — the position system thinks the entity is at the old position while the component says the new one (or vice versa).

```go
// CORRECT — atomic update
manager.MoveEntity(entityID, entity, oldPos, newPos)

// WRONG — only updates component, position system is now stale
posData := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
posData.X = newX
posData.Y = newY

// WRONG — only updates position system, component is now stale
common.GlobalPositionSystem.RemoveEntity(entityID, oldPos)
common.GlobalPositionSystem.AddEntity(entityID, newPos)
```

**Impact:** `GetEntitiesAtPosition()` returns wrong results. Entities appear at old positions or become invisible to spatial queries.

**Mitigation:** Use the atomic helpers on `EntityManager` for the full position lifecycle:

```go
// CREATION — atomically adds PositionComponent + registers with GlobalPositionSystem
manager.RegisterEntityPosition(entity, pos)

// MOVEMENT — atomically updates component + position system (already existed)
manager.MoveEntity(entityID, entity, oldPos, newPos)
manager.MoveSquadAndMembers(squadID, squadEntity, unitIDs, oldPos, newPos)

// REMOVAL (keep entity) — atomically removes from position system + removes component
manager.UnregisterEntityPosition(entity)

// REMOVAL (dispose entity) — atomically removes from position system + disposes entity
manager.CleanDisposeEntity(entity, pos)
```

Never call `GlobalPositionSystem.AddEntity/RemoveEntity` directly from outside the `common` package. All external callers have been migrated to use these atomic helpers.

For debug validation, call `GlobalPositionSystem.ValidatePositionSync(manager)` which returns a list of desync errors (empty if all positions are consistent).

---

### 4. ParseStatType Silent Fallback to StatStrength

**File:** `tactical/effects/components.go:47-68`

`ParseStatType()` returns `StatStrength` for any unrecognized string. Since `StatStrength` is iota value 0, a typo in JSON stat names silently buffs/debuffs Strength instead of the intended stat.

```go
func ParseStatType(stat string) StatType {
    switch strings.ToLower(stat) {
    case "strength":
        return StatStrength
    // ... other cases ...
    default:
        return StatStrength  // ← typo "stregth" silently becomes Strength
    }
}
```

**Impact:** JSON spell/effect definitions with typos in stat names will compile and run without error, but modify the wrong attribute.

**Mitigation:** When adding new effects or spells in JSON, double-check stat name strings against the exact values: `strength`, `dexterity`, `magic`, `leadership`, `armor`, `weapon`, `movementspeed`, `attackrange`.

---

### 5. Two Unit Creation Paths Must Stay in Sync

**Files:** `tactical/squads/units.go:194-298` and `tactical/squads/squadcreation.go:289-470`

There are two independent code paths that create unit entities:

1. **`CreateUnitEntity()`** — Used by `AddUnitToSquad()` for individual unit creation (player purchasing, roster management). Delegates base entity creation to `templates.CreateUnit()`.

2. **`CreateSquadFromTemplate()`** — Used for batch squad creation (enemy squads, initial setup). Delegates to `templates.CreateEntityFromTemplate()`.

Both paths add the same components (GridPosition, UnitRole, TargetRow, Cover, AttackRange, MovementSpeed, UnitType, Experience, StatGrowth) but through different code. Adding a new component to one path and forgetting the other creates units with inconsistent component sets.

**Impact:** Units created through different paths may be missing components, causing nil returns from `GetComponentType` and silent failures.

**Mitigation:** When adding a new component to units, grep for both `CreateUnitEntity` and `CreateSquadFromTemplate` and update both. Consider the component checklist:

| Component | CreateUnitEntity | CreateSquadFromTemplate |
|-----------|-----------------|------------------------|
| GridPositionComponent | Line 243 | Line 394 |
| UnitRoleComponent | Line 250 | Line 402 |
| TargetRowComponent | Line 255 | Line 407 |
| CoverComponent | Line 262 | Line 414 |
| AttackRangeComponent | Line 270 | Line 422 |
| MovementSpeedComponent | Line 275 | Line 427 |
| UnitTypeComponent | Line 238 | Line 432 |
| ExperienceComponent | Line 280 | Line 437 |
| StatGrowthComponent | Line 287 | Line 444 |

---

## Medium Priority — Initialization & Lifecycle

### 6. Boot Sequence Order

**File:** `gamesetup/bootstrap.go`

Game initialization follows a fixed 5-phase order. Calling phases out of order causes panics or nil dereferences:

```
Phase 1: LoadGameData()          — JSON templates (no dependencies)
Phase 2: InitializeCoreECS()     — ECS world, GlobalPositionSystem, graphics
Phase 3: CreateWorld()           — Map generation (needs coords from Phase 2)
Phase 4: CreatePlayer()          — Player entity, commanders, initial squads
Phase 5: InitializeGameplay()    — Overworld tick, factions, walkable grid
Debug:   SetupDebugContent()     — Test data (only when DEBUG_MODE=true)
```

**Impact:** Reordering phases (e.g., creating squads before ECS init, or placing entities before the map exists) will panic.

**Note:** The save-game loading path (`savesystem/`) has its own initialization sequence that bypasses `CreatePlayer` and `CreateWorld`, reconstructing state from save data instead. Be aware that changes to the boot sequence may not automatically apply to save-game loading.

---

### 7. Blank Imports for Subsystem Registration

**Files:** `game_main/main.go:28`, `gamesetup/playerinit.go:9`

Some subsystems register themselves via `init()` functions (the self-registration pattern described in CLAUDE.md). These packages must be blank-imported somewhere in the import chain, or their `init()` never runs and their components are never registered.

```go
// game_main/main.go
import (
    _ "game_main/savesystem/chunks" // Blank import to register SaveChunks via init()
)

// gamesetup/playerinit.go
import (
    _ "game_main/tactical/squadcommands" // Blank import to trigger init() for command queue components
)
```

**Impact:** If a blank import is removed (e.g., during an "unused import" cleanup), the subsystem's components will be nil at runtime. This causes panics or silent failures when code tries to use those components.

**Mitigation:** Never remove blank imports without understanding why they exist. The comment after the import explains its purpose. If adding a new self-registering subsystem, add a blank import with a comment explaining why.

---

### 8. IsOpaque() Has No Bounds Check

**File:** `world/worldmap/dungeongen.go:249-253`

`IsOpaque()` does not check if `(x, y)` is within map bounds before indexing into the Tiles slice. Compare with `GetBiomeAt()` which does check:

```go
// NO bounds check — panics on out-of-bounds
func (gameMap GameMap) IsOpaque(x, y int) bool {
    logicalPos := coords.LogicalPosition{X: x, Y: y}
    idx := coords.CoordManager.LogicalToIndex(logicalPos)
    return gameMap.Tiles[idx].TileType == WALL  // ← index out of range if (x,y) is outside map
}

// HAS bounds check — safe
func (gm *GameMap) GetBiomeAt(pos coords.LogicalPosition) Biome {
    if gm.BiomeMap == nil {
        return BiomeGrassland
    }
    idx := coords.CoordManager.LogicalToIndex(pos)
    if idx < 0 || idx >= len(gm.BiomeMap) {
        return BiomeGrassland
    }
    return gm.BiomeMap[idx]
}

// Also has bounds check
func (gameMap GameMap) InBounds(x, y int) bool {
    if x < 0 || x >= graphics.ScreenInfo.DungeonWidth || y < 0 || y >= graphics.ScreenInfo.DungeonHeight {
        return false
    }
    return true
}
```

**Impact:** FOV or line-of-sight calculations that call `IsOpaque()` with coordinates at or beyond the map edge will panic with an index out of range error.

**Mitigation:** Callers of `IsOpaque()` should check `InBounds()` first, or the function itself should be updated with a bounds guard.

---

### 9. UnassignUnitFromSquad Orphans LeaderComponent

**File:** `tactical/squads/squadcreation.go:211-232`

`UnassignUnitFromSquad()` removes `SquadMemberComponent` and resets the grid position, but does **not** remove `LeaderComponent` (or its associated `AbilitySlotComponent` and `CooldownTrackerComponent`). If the unassigned unit was the squad leader, it retains leader components after leaving the squad.

```go
func UnassignUnitFromSquad(unitEntityID ecs.EntityID, manager *common.EntityManager) error {
    // ...
    unitEntity.RemoveComponent(SquadMemberComponent)  // Removes membership

    // Reset grid position
    gridPos := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent)
    if gridPos != nil {
        gridPos.AnchorRow = 0
        gridPos.AnchorCol = 0
    }

    return nil
    // ← No RemoveLeaderComponents() call
}
```

Compare with `RemoveLeaderComponents()` which exists and handles cleanup:

```go
func RemoveLeaderComponents(entity *ecs.Entity) {
    if entity.HasComponent(LeaderComponent) {
        entity.RemoveComponent(LeaderComponent)
    }
    if entity.HasComponent(AbilitySlotComponent) {
        entity.RemoveComponent(AbilitySlotComponent)
    }
    if entity.HasComponent(CooldownTrackerComponent) {
        entity.RemoveComponent(CooldownTrackerComponent)
    }
}
```

**Impact:** Orphaned leader components on roster units. If the unit is later placed into a different squad, it may incorrectly be treated as that squad's leader, or the squad may end up with two leaders.

---

## Lower Priority — Edge Cases

### 10. Soft vs Hard JSON Requirements

**File:** `templates/readdata.go`

Most JSON config files use `readAndUnmarshal()` which panics if the file is missing or malformed. The exception is `mapgenconfig.json`:

```go
// HARD requirement — panics if missing
func readAndUnmarshal[T any](path string, target *T) {
    data, err := os.ReadFile(AssetPath(path))
    if err != nil {
        panic(err)  // ← crash on missing file
    }
    // ...
}

// SOFT requirement — gracefully falls back to code defaults
func ReadMapGenConfig() {
    data, err := os.ReadFile(AssetPath("gamedata/mapgenconfig.json"))
    if err != nil {
        println("Map gen config not found, using code defaults")
        return  // ← no panic, just uses defaults
    }
    // ...
}
```

**JSON files and their requirement level:**

| File | Required | On Missing |
|------|----------|------------|
| monsterdata.json | Hard | Panic |
| nodeDefinitions.json | Hard | Panic |
| encounterdata.json | Hard | Panic |
| namedata.json | Hard | Panic |
| aiconfig.json | Hard | Panic |
| powerconfig.json | Hard | Panic |
| overworldconfig.json | Hard | Panic |
| influenceconfig.json | Hard | Panic |
| difficultyconfig.json | Hard | Panic |
| spelldata.json | Hard | Panic |
| minor_artifacts.json | Hard | Panic |
| major_artifacts.json | Hard | Panic |
| **mapgenconfig.json** | **Soft** | **Falls back to code defaults** |

**Impact:** Deleting or renaming any hard-required JSON file crashes the game on startup with a panic. Only `mapgenconfig.json` is safe to remove.

---

### 11. DEBUG_MODE and ENABLE_BENCHMARKING Hardcoded True

**File:** `config/config.go:17-20`

Both flags are compile-time constants set to `true`:

```go
const (
    DEBUG_MODE          = true  // Enables debug content, test commanders, factions
    ENABLE_BENCHMARKING = true  // Starts pprof server on localhost:6060
)
```

**Impact:**
- Every launch starts a pprof HTTP server on `localhost:6060` (see `gamesetup/helpers.go:38-42`)
- Debug content is always created (test commanders, extra roster units, artifact seeding)
- This is expected during development, but these must be set to `false` before any release build

**Note:** These are Go `const` values, not environment variables. They can only be changed by editing `config/config.go` and recompiling. There is no runtime toggle.
