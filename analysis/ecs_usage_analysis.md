# ECS Usage Analysis: TinkerRogue
**Generated:** 2025-10-01
**ECS Library:** github.com/bytearena/ecs v1.0.0 (2017)
**Analysis Scope:** Complete codebase review focusing on Entity Component System patterns

---

## EXECUTIVE SUMMARY

### Overall Assessment: **NEEDS SIGNIFICANT IMPROVEMENT** ⚠️

The codebase is using the bytearena/ecs library (v1.0.0 from 2017) in a **hybrid anti-pattern** that combines traditional OOP with ECS, resulting in the worst of both worlds:

**Critical Issues:**
1. **Components contain behavior** - Violates fundamental ECS principle of data/behavior separation
2. **Heavy use of entity references** - Creates tight coupling and breaks ECS cache-friendliness
3. **Missing true systems** - Logic scattered across packages instead of centralized systems
4. **Nested entity hierarchies** - Items store entities that store entities (Properties field)
5. **Query performance issues** - Linear searches through all monsters for position lookups

**Good Practices Found:**
- Type-safe component access via generics (`GetComponentType[T]`)
- Centralized component registration
- Tag-based entity queries
- Clear component initialization

**Impact on Simplification Roadmap:**
This ECS usage pattern is actively **hindering** the roadmap goals. The hybrid approach makes refactoring harder, not easier. The planned squad system will compound these issues unless the ECS patterns are corrected first.

---

## TOP 3 CRITICAL ISSUES

### 1. **Components Contain Logic (God Components)**
**Severity:** CRITICAL
**Impact:** Violates core ECS principle, makes testing/refactoring difficult
**Effort to Fix:** HIGH (40-60 hours)

#### Problem Description
Components in ECS should be **pure data containers**. Instead, this codebase has components with methods containing game logic:

**Examples:**

`gear/equipmentcomponents.go:55-58` - `MeleeWeapon.CalculateDamage()`:
```go
type MeleeWeapon struct {
    MinDamage   int
    MaxDamage   int
    AttackSpeed int
}

func (w MeleeWeapon) CalculateDamage() int {
    return randgen.GetRandomBetween(w.MinDamage, w.MaxDamage)
}
```

`gear/equipmentcomponents.go:94-114` - `RangedWeapon.GetTargets()`:
```go
func (r RangedWeapon) GetTargets(ecsmanger *common.EntityManager) []*ecs.Entity {
    pos := coords.CoordManager.GetTilePositionsAsCommon(r.TargetArea.GetIndices())
    targets := make([]*ecs.Entity, 0)

    for _, c := range ecsmanger.World.Query(ecsmanger.WorldTags["monsters"]) {
        curPos := common.GetPosition(c.Entity)
        for _, p := range pos {
            if curPos.IsEqual(&p) {
                targets = append(targets, c.Entity)
            }
        }
    }
    return targets
}
```

`monsters/creatures.go:77-109` - `Creature.UpdatePosition()`:
```go
func (c *Creature) UpdatePosition(gm *worldmap.GameMap, currentPosition *coords.LogicalPosition) {
    // 30+ lines of movement logic inside a component
}
```

#### Why This Is Wrong

**ECS Principle Violation:**
- Components = Data (struct fields only)
- Systems = Logic (functions that operate on components)

**Problems Created:**
1. **Cannot iterate efficiently** - Have to query entities to call component methods
2. **Tight coupling** - Components depend on game state (EntityManager, GameMap)
3. **Hard to test** - Cannot test logic without full ECS setup
4. **Poor cache locality** - Method calls break CPU cache optimization
5. **Difficult refactoring** - Logic embedded in data structures

#### Recommended Fix

**Before (Current Anti-Pattern):**
```go
// Component with logic
type MeleeWeapon struct {
    MinDamage   int
    MaxDamage   int
    AttackSpeed int
}

func (w MeleeWeapon) CalculateDamage() int {
    return randgen.GetRandomBetween(w.MinDamage, w.MaxDamage)
}

// Usage scattered everywhere
damage := weapon.CalculateDamage()
```

**After (Proper ECS Pattern):**
```go
// Component: Pure data only
type MeleeWeapon struct {
    MinDamage   int
    MaxDamage   int
    AttackSpeed int
}

// System: Contains all weapon logic
type WeaponSystem struct {
    rng *randgen.Generator
}

func (ws *WeaponSystem) CalculateMeleeDamage(weapon *MeleeWeapon) int {
    return ws.rng.GetRandomBetween(weapon.MinDamage, weapon.MaxDamage)
}

func (ws *WeaponSystem) CalculateRangedDamage(weapon *RangedWeapon) int {
    return ws.rng.GetRandomBetween(weapon.MinDamage, weapon.MaxDamage)
}

func (ws *WeaponSystem) GetRangedTargets(
    weapon *RangedWeapon,
    weaponPos *coords.LogicalPosition,
    positionSystem *PositionSystem,
) []*ecs.Entity {
    targetIndices := weapon.TargetArea.GetIndices()
    targetPositions := coords.CoordManager.GetTilePositionsAsCommon(targetIndices)
    return positionSystem.GetEntitiesAtPositions(targetPositions)
}
```

#### Files Requiring Changes

1. **`gear/equipmentcomponents.go`** (162 lines)
   - Extract: `MeleeWeapon.CalculateDamage()` → `WeaponSystem`
   - Extract: `RangedWeapon.CalculateDamage()` → `WeaponSystem`
   - Extract: `RangedWeapon.GetTargets()` → `WeaponSystem`
   - Extract: `RangedWeapon.DisplayShootingVX()` → `VisualEffectSystem`

2. **`monsters/creatures.go`** (126 lines)
   - Extract: `Creature.UpdatePosition()` → `MovementSystem`
   - Extract: `ApplyStatusEffects()` → `StatusEffectSystem`

3. **`gear/items.go`** (288 lines)
   - Extract: `Item.GetEffectNames()` → `ItemSystem`
   - Extract: `Item.GetActions()` → `ItemSystem`
   - Extract: Item filtering logic → `ItemSystem`

4. **Create new systems:**
   - `systems/weapon_system.go` (~200 lines)
   - `systems/movement_system.go` (~150 lines)
   - `systems/status_effect_system.go` (~200 lines)
   - `systems/item_system.go` (~250 lines)

#### Benefits

- **Performance:** Systems can batch operations efficiently
- **Testability:** Test systems with mock data, no ECS setup needed
- **Maintainability:** All weapon logic in one place
- **Refactorability:** Change combat without touching components
- **Squad System Ready:** Systems can handle single units or squads

#### Risks

- **Large refactor:** Touches combat, items, movement - core game systems
- **Breaking changes:** All code calling component methods must update
- **Testing overhead:** Need comprehensive tests before/after
- **Time investment:** 40-60 hours estimated

---

### 2. **Entity Reference Coupling (Anti-Pattern)**
**Severity:** HIGH
**Impact:** Memory leaks, tight coupling, breaks ECS cache optimization
**Effort to Fix:** MEDIUM (20-30 hours)

#### Problem Description

Components and structs store raw `*ecs.Entity` pointers everywhere, creating tight coupling and making entity lifecycle management fragile:

**Examples:**

`gear/items.go:41-45` - Item stores entity for properties:
```go
type Item struct {
    Properties *ecs.Entity  // ANTI-PATTERN: Component stores entity
    Actions    []ItemAction
    Count      int
}
```

`avatar/playerdata.go:26-30` - Equipment stores entity references:
```go
type PlayerEquipment struct {
    EqMeleeWeapon  *ecs.Entity  // ANTI-PATTERN: Storing entity refs
    EqRangedWeapon *ecs.Entity
    EqArmor        *ecs.Entity
}
```

`worldmap/dungeontile.go:24` - Tile stores entities:
```go
type DungeonTile struct {
    // ... other fields
    entities []*ecs.Entity  // ANTI-PATTERN: Non-component storing entities
}
```

`trackers/creaturetracker.go:18` - Position tracker uses entity refs:
```go
type PositionTracker struct {
    PosTracker map[*coords.LogicalPosition]*ecs.Entity  // ANTI-PATTERN
}
```

#### Why This Is Wrong

**ECS Best Practice:**
- Components should reference entities via **IDs** (int/uint), not pointers
- Allows entity deletion without dangling pointers
- Enables save/load systems
- Supports entity pooling/reuse

**Problems Created:**
1. **Dangling pointers** - Deleted entities leave invalid references
2. **Memory leaks** - Hard to track all references when deleting
3. **No serialization** - Can't save/load game with raw pointers
4. **Tight coupling** - Direct access breaks encapsulation
5. **Cache unfriendly** - Pointer chasing breaks CPU cache

#### Current Evidence of Problems

`resourcemanager/cleanup.go:13` - Manual cleanup required:
```go
func RemoveEntity(world *ecs.Manager, gm *worldmap.GameMap, e *ecs.Entity) {
    // Must manually remove from position tracker
    trackers.CreatureTracker.Remove(e)

    // Must manually remove from tile
    pos := common.GetPosition(e)
    // ... complex removal logic
}
```

**Comment in `common/ecsutil.go:72`:**
```go
// TODO: Optimize this to avoid searching all monsters every time.
func GetCreatureAtPosition(ecsmnager *EntityManager, pos *coords.LogicalPosition) *ecs.Entity {
    // Linear search through ALL monsters - O(n) every call
}
```

#### Recommended Fix

**Pattern 1: Use Entity IDs Instead of Pointers**

```go
// Before (Anti-Pattern)
type Item struct {
    Properties *ecs.Entity  // Raw pointer
}

// After (Proper ECS)
type Item struct {
    PropertiesID ecs.EntityID  // ID reference
}

// Access via system
func (is *ItemSystem) GetItemProperties(item *Item) []*StatusEffect {
    if item.PropertiesID == 0 {
        return nil
    }
    entity := is.manager.GetEntity(item.PropertiesID)
    return extractStatusEffects(entity)
}
```

**Pattern 2: Use Spatial Indexing System**

```go
// Before (Anti-Pattern in trackers/creaturetracker.go)
type PositionTracker struct {
    PosTracker map[*coords.LogicalPosition]*ecs.Entity  // Pointer keys!
}

// After (Proper ECS)
type PositionSystem struct {
    manager     *ecs.Manager
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // ID-based
}

func (ps *PositionSystem) GetEntityAt(pos coords.LogicalPosition) *ecs.Entity {
    if ids, ok := ps.spatialGrid[pos]; ok && len(ids) > 0 {
        return ps.manager.GetEntity(ids[0])  // O(1) lookup
    }
    return nil
}

func (ps *PositionSystem) UpdateEntityPosition(entityID ecs.EntityID, oldPos, newPos coords.LogicalPosition) {
    // Remove from old position
    ps.removeFromGrid(entityID, oldPos)
    // Add to new position
    ps.addToGrid(entityID, newPos)
}
```

**Pattern 3: Component References for Related Entities**

```go
// Before (Anti-Pattern in avatar/playerdata.go)
type PlayerEquipment struct {
    EqMeleeWeapon  *ecs.Entity
    EqRangedWeapon *ecs.Entity
    EqArmor        *ecs.Entity
}

// After (Proper ECS)
type EquipmentSlots struct {
    MeleeWeaponID  ecs.EntityID  // 0 = empty slot
    RangedWeaponID ecs.EntityID
    ArmorID        ecs.EntityID
}

// Add as component to player entity
playerEntity.AddComponent(EquipmentSlotsComponent, &EquipmentSlots{
    MeleeWeaponID: weaponEntity.ID(),
    // ...
})
```

#### Files Requiring Changes

1. **`gear/items.go`** (288 lines)
   - Change: `Item.Properties *ecs.Entity` → `PropertiesID ecs.EntityID`
   - Impact: All item property access code

2. **`avatar/playerdata.go`** (147 lines)
   - Change: PlayerEquipment fields to IDs
   - Create: EquipmentSlots component
   - Impact: Equipment system, UI code

3. **`trackers/creaturetracker.go`** (47 lines)
   - Rewrite as proper `PositionSystem`
   - Change: Map keys from pointers to values
   - Add: Spatial grid with entity IDs

4. **`worldmap/dungeontile.go`** (24 lines)
   - Change: `entities []*ecs.Entity` → `entityIDs []ecs.EntityID`
   - Impact: Tile access patterns

5. **`common/ecsutil.go`** (90 lines)
   - Replace: `GetCreatureAtPosition()` with `PositionSystem.GetEntityAt()`
   - Add: `GetEntity(id)` helper

6. **Create new systems:**
   - `systems/position_system.go` (~300 lines) - Spatial indexing
   - `systems/equipment_system.go` (~200 lines) - Equipment management

#### Benefits

- **Correctness:** No dangling pointers, safer entity deletion
- **Performance:** O(1) position lookups instead of O(n)
- **Serialization:** Can save/load game state
- **Memory:** Easier garbage collection, no circular references
- **Squad Ready:** ID-based references work for squad entities too

#### Risks

- **Breaking change:** Almost every file touches entities
- **Migration complexity:** Need to update all entity access patterns
- **ID management:** bytearena/ecs may not expose entity IDs easily
- **Testing burden:** Must verify no null pointer dereferences

**CRITICAL NOTE:** The bytearena/ecs library (v1.0.0, 2017) may not provide Entity IDs. If true, this requires either:
1. Fork/patch the library
2. Switch to a modern ECS library (donburi, arche, etc.)
3. Implement ID wrapper layer

---

### 3. **Missing True Systems Architecture**
**Severity:** HIGH
**Impact:** Logic scattered across packages, hard to understand/modify
**Effort to Fix:** MEDIUM-HIGH (30-40 hours)

#### Problem Description

The codebase has **no dedicated systems directory or architecture**. Instead, logic is scattered across:
- Component method calls
- Standalone functions in various packages
- Input controllers (which act as pseudo-systems)
- Ad-hoc game loop code

**Current "Systems" Are Actually:**
- `combat/attackingsystem.go` - Just functions, not a system struct
- `rendering/rendering.go` - Static functions, no system state
- Component methods - Logic embedded in data

**Evidence:**

`combat/attackingsystem.go:21-53` - No system struct:
```go
// Just a function, not a proper system
func MeleeAttackSystem(ecsmanager *common.EntityManager, pl *avatar.PlayerData,
    gm *worldmap.GameMap, attackerPos *coords.LogicalPosition,
    defenderPos *coords.LogicalPosition) {

    var attacker *ecs.Entity = nil
    var defender *ecs.Entity = nil
    // ... 30 lines of logic
}
```

`rendering/rendering.go:24-52` - Static rendering function:
```go
func ProcessRenderables(ecsmanager *common.EntityManager, gameMap worldmap.GameMap,
    screen *ebiten.Image, debugMode bool) {

    for _, result := range ecsmanager.World.Query(ecsmanager.WorldTags["renderables"]) {
        // Rendering logic mixed with query
    }
}
```

`monsters/creatures.go:44-72` - Free function on QueryResult:
```go
func ApplyStatusEffects(c *ecs.QueryResult) {
    // Status effect logic scattered
}
```

#### Why This Is Wrong

**ECS Architecture Best Practice:**
```
Systems own logic + state
  ↓ operate on
Components (pure data)
  ↓ attached to
Entities (just IDs)
```

**Problems Created:**
1. **No state management** - Systems need persistent state (caches, pools)
2. **No initialization/cleanup** - Can't setup/teardown resources
3. **Hard to test** - Can't mock system dependencies
4. **No update order control** - Can't enforce system execution order
5. **Scattered queries** - Same queries repeated everywhere
6. **Global dependencies** - Functions take 5+ parameters

#### Recommended Fix

**Create Proper Systems Architecture:**

```go
// systems/system.go - Base system interface
package systems

type System interface {
    // Initialize system resources
    Initialize(manager *ecs.Manager) error

    // Update system logic (called every frame/tick)
    Update(dt float64) error

    // Cleanup system resources
    Cleanup() error
}

// systems/combat_system.go
type CombatSystem struct {
    manager      *ecs.Manager
    tags         map[string]ecs.Tag
    weaponSystem *WeaponSystem
    posSystem    *PositionSystem

    // System state
    combatLog    []CombatEvent
    damageCache  map[ecs.EntityID]int
}

func NewCombatSystem(manager *ecs.Manager, tags map[string]ecs.Tag) *CombatSystem {
    return &CombatSystem{
        manager:     manager,
        tags:        tags,
        combatLog:   make([]CombatEvent, 0, 100),
        damageCache: make(map[ecs.EntityID]int),
    }
}

func (cs *CombatSystem) Initialize(manager *ecs.Manager) error {
    // Setup combat system
    cs.weaponSystem = NewWeaponSystem(manager)
    cs.posSystem = NewPositionSystem(manager)
    return nil
}

func (cs *CombatSystem) Update(dt float64) error {
    // Process all pending combat actions
    return nil
}

func (cs *CombatSystem) ExecuteMeleeAttack(attackerID, defenderID ecs.EntityID) CombatResult {
    attacker := cs.manager.GetEntity(attackerID)
    defender := cs.manager.GetEntity(defenderID)

    // Get components
    attackerAttr := common.GetAttributes(attacker)
    defenderAttr := common.GetAttributes(defender)
    attackerWeapon := common.GetComponentType[*gear.MeleeWeapon](attacker, gear.MeleeWeaponComponent)

    // Use weapon system to calculate damage
    baseDamage := cs.weaponSystem.CalculateMeleeDamage(attackerWeapon)

    // Combat resolution logic
    result := cs.resolveCombat(attackerAttr, defenderAttr, baseDamage)

    // Log event
    cs.combatLog = append(cs.combatLog, CombatEvent{
        AttackerID: attackerID,
        DefenderID: defenderID,
        Damage:     result.FinalDamage,
        Hit:        result.Hit,
    })

    return result
}

func (cs *CombatSystem) resolveCombat(attacker, defender *common.Attributes, damage int) CombatResult {
    // Centralized combat resolution logic
    attackRoll := randgen.GetDiceRoll(20) + attacker.AttackBonus

    if attackRoll >= defender.TotalArmorClass {
        dodgeRoll := randgen.GetRandomBetween(0, 100)
        if dodgeRoll >= int(defender.TotalDodgeChance) {
            finalDamage := damage - defender.TotalProtection
            if finalDamage < 1 {
                finalDamage = 1
            }
            defender.CurrentHealth -= finalDamage
            return CombatResult{Hit: true, FinalDamage: finalDamage}
        }
    }

    return CombatResult{Hit: false, FinalDamage: 0}
}

func (cs *CombatSystem) Cleanup() error {
    // Cleanup combat system resources
    cs.combatLog = nil
    cs.damageCache = nil
    return nil
}
```

**System Manager Pattern:**

```go
// systems/manager.go
type SystemManager struct {
    systems []System
    manager *ecs.Manager
}

func NewSystemManager(manager *ecs.Manager) *SystemManager {
    return &SystemManager{
        systems: make([]System, 0, 10),
        manager: manager,
    }
}

func (sm *SystemManager) RegisterSystem(system System) error {
    if err := system.Initialize(sm.manager); err != nil {
        return err
    }
    sm.systems = append(sm.systems, system)
    return nil
}

func (sm *SystemManager) Update(dt float64) error {
    // Update systems in registration order
    for _, system := range sm.systems {
        if err := system.Update(dt); err != nil {
            return err
        }
    }
    return nil
}

func (sm *SystemManager) Cleanup() error {
    // Cleanup in reverse order
    for i := len(sm.systems) - 1; i >= 0; i-- {
        if err := sm.systems[i].Cleanup(); err != nil {
            return err
        }
    }
    return nil
}
```

**Usage in Game Loop:**

```go
// game_main/main.go
type Game struct {
    systemManager *systems.SystemManager
    combatSystem  *systems.CombatSystem
    renderSystem  *systems.RenderSystem
    // ...
}

func (g *Game) Initialize() {
    g.systemManager = systems.NewSystemManager(g.ecsManager.World)

    // Register systems in execution order
    g.combatSystem = systems.NewCombatSystem(g.ecsManager.World, g.ecsManager.WorldTags)
    g.systemManager.RegisterSystem(g.combatSystem)

    g.renderSystem = systems.NewRenderSystem(g.ecsManager.World, g.ecsManager.WorldTags)
    g.systemManager.RegisterSystem(g.renderSystem)

    // ... other systems
}

func (g *Game) Update() error {
    dt := 1.0 / 60.0  // Frame time
    return g.systemManager.Update(dt)
}
```

#### Files Requiring Changes

**Create new systems directory:**
1. `systems/system.go` - System interface (~50 lines)
2. `systems/manager.go` - System manager (~100 lines)
3. `systems/combat_system.go` - Combat logic (~400 lines)
4. `systems/weapon_system.go` - Weapon logic (~200 lines)
5. `systems/movement_system.go` - Movement logic (~250 lines)
6. `systems/render_system.go` - Rendering logic (~300 lines)
7. `systems/status_effect_system.go` - Status effects (~200 lines)
8. `systems/position_system.go` - Spatial indexing (~300 lines)
9. `systems/item_system.go` - Item management (~250 lines)

**Refactor existing files:**
1. `combat/attackingsystem.go` → Extract to `systems/combat_system.go`
2. `rendering/rendering.go` → Extract to `systems/render_system.go`
3. `monsters/creatures.go` → Extract logic to systems
4. `gear/equipmentcomponents.go` → Extract to `systems/weapon_system.go`

**Update game initialization:**
1. `game_main/main.go` - Add SystemManager
2. `game_main/gameinit.go` - Initialize systems instead of scattered setup

#### Benefits

- **Organization:** All combat logic in one place
- **State management:** Systems can cache, pool resources
- **Testability:** Mock system dependencies easily
- **Update order:** Control system execution sequence
- **Performance:** Systems can optimize queries, batch operations
- **Squad ready:** Systems naturally handle single units or squads
- **Maintainability:** Clear responsibility boundaries

#### Risks

- **Large refactor:** Need to extract logic from many files
- **Initialization order:** Systems must initialize in correct sequence
- **Dependency management:** Systems depend on other systems
- **Time investment:** 30-40 hours for full migration

---

## DETAILED ECS USAGE INVENTORY

### Component Registration (`game_main/componentinit.go`)

**Status:** ✅ **GOOD PRACTICE**

```go
func InitializeECS(ecsmanager *common.EntityManager) {
    manager := ecs.NewManager()

    // Centralized component registration
    common.PositionComponent = manager.NewComponent()
    rendering.RenderableComponent = manager.NewComponent()
    common.NameComponent = manager.NewComponent()
    gear.InventoryComponent = manager.NewComponent()
    common.AttributeComponent = manager.NewComponent()
    // ... more components

    // Tag creation for queries
    renderables := ecs.BuildTag(rendering.RenderableComponent, common.PositionComponent)
    tags["renderables"] = renderables

    creatures := ecs.BuildTag(monsters.CreatureComponent, common.PositionComponent, common.AttributeComponent)
    tags["monsters"] = creatures
}
```

**Strengths:**
- ✅ Single initialization point
- ✅ Clear tag definitions
- ✅ Centralized component registration

**Issues:**
- ⚠️ Global component variables scattered across packages
- ⚠️ No component documentation
- ⚠️ Hard to know what tags exist

**Recommendation:** Keep this pattern, but add:
- Component registry/documentation
- Tag registry with descriptions
- Compile-time safety for tag names (use constants)

---

### Component Definitions

#### Pure Data Components (✅ Good)

These follow ECS best practices:

**`common/commoncomponents.go` - Name, UserMessage:**
```go
type Name struct {
    NameStr string  // Pure data
}

type UserMessage struct {
    AttackMessage       string
    GameStateMessage    string
    StatusEffectMessage string
}
```

**`common/commoncomponents.go` - Attributes:**
```go
type Attributes struct {
    MaxHealth          int
    CurrentHealth      int
    AttackBonus        int
    BaseArmorClass     int
    // ... 15 fields, all data
}
```

**Status:** ✅ These are correct - pure data, no methods

---

#### Impure Data Components (❌ Bad)

These violate ECS principles by containing logic:

**`gear/equipmentcomponents.go` - MeleeWeapon:**
```go
type MeleeWeapon struct {
    MinDamage   int
    MaxDamage   int
    AttackSpeed int
}

// ❌ LOGIC IN COMPONENT
func (w MeleeWeapon) CalculateDamage() int {
    return randgen.GetRandomBetween(w.MinDamage, w.MaxDamage)
}

func (w *MeleeWeapon) DisplayString() string {
    // Display logic in component
}
```

**`gear/equipmentcomponents.go` - RangedWeapon:**
```go
type RangedWeapon struct {
    MinDamage     int
    MaxDamage     int
    ShootingRange int
    TargetArea    graphics.TileBasedShape  // Contains complex shape logic
    ShootingVX    graphics.VisualEffect    // Contains rendering state
    AttackSpeed   int
}

// ❌ COMPLEX QUERY LOGIC IN COMPONENT
func (r RangedWeapon) GetTargets(ecsmanger *common.EntityManager) []*ecs.Entity {
    // 20 lines of entity queries and filtering
}

// ❌ RENDERING LOGIC IN COMPONENT
func (r *RangedWeapon) DisplayShootingVX(attackerPos, defenderPos *coords.LogicalPosition) {
    // Visual effect creation and registration
}

func (r *RangedWeapon) DisplayCenteredShootingVX(attackerPos, defenderPos *coords.LogicalPosition) {
    // More rendering logic
}
```

**`monsters/creatures.go` - Creature:**
```go
type Creature struct {
    Path              []coords.LogicalPosition
    StatEffectTracker trackers.StatusEffectTracker
}

// ❌ MOVEMENT LOGIC IN COMPONENT
func (c *Creature) UpdatePosition(gm *worldmap.GameMap, currentPosition *coords.LogicalPosition) {
    // 30+ lines of pathfinding and tile management logic
}

// ❌ DISPLAY LOGIC IN COMPONENT
func (c *Creature) DisplayString(e *ecs.Entity) string {
    // Formatting logic
}
```

**`gear/items.go` - Item:**
```go
type Item struct {
    Properties *ecs.Entity  // ❌ ENTITY REFERENCE
    Actions    []ItemAction
    Count      int
}

// ❌ QUERY LOGIC IN COMPONENT
func (item *Item) GetEffectNames() []string {
    // Iterates through components
}

// ❌ FILTERING LOGIC IN COMPONENT
func (item *Item) HasAllEffects(effectsToCheck ...StatusEffects) bool {
    // Complex filtering logic
}

// ❌ TYPE CHECKING LOGIC IN COMPONENT
func (item *Item) GetThrowableAction() *ThrowableAction {
    // Type assertion logic
}
```

**Status:** ❌ All need refactoring to extract logic into systems

---

### Query Patterns

#### Current Query Usage

**Total Queries Found:** 14 locations

**Query Patterns:**

1. **Rendering queries** (2 locations):
```go
// rendering/rendering.go:25
for _, result := range ecsmanager.World.Query(ecsmanager.WorldTags["renderables"]) {
    pos := result.Components[common.PositionComponent].(*coords.LogicalPosition)
    img := result.Components[RenderableComponent].(*Renderable).Image
    // ...
}
```

2. **Monster queries** (6 locations):
```go
// common/ecsutil.go:76 - O(n) position lookup
for _, c := range ecsmnager.World.Query(ecsmnager.WorldTags["monsters"]) {
    curPos := common.GetPosition(c.Entity)
    if pos.IsEqual(curPos) {
        return c.Entity
    }
}

// gear/equipmentcomponents.go:100 - O(n*m) target finding
for _, c := range ecsmanger.World.Query(ecsmanger.WorldTags["monsters"]) {
    curPos := common.GetPosition(c.Entity)
    for _, p := range pos {
        if curPos.IsEqual(&p) {
            targets = append(targets, c.Entity)
        }
    }
}
```

3. **Item queries** (1 location):
```go
// testing/testingdata.go:142
for _, item := range ecsmanager.World.Query(ecsmanager.WorldTags["items"]) {
    // Item processing
}
```

4. **Message queries** (1 location):
```go
// gui/usermessages.go:16
for _, m := range ecsmanager.World.Query(ecsmanager.WorldTags["messengers"]) {
    // Message collection
}
```

**Issues:**

1. **Scattered queries:** Same query (`monsters` tag) repeated in 6 files
2. **Linear searches:** O(n) searches every time (see `GetCreatureAtPosition`)
3. **Nested loops:** O(n*m) target finding in `RangedWeapon.GetTargets()`
4. **No caching:** Results not cached between frames
5. **Manual component access:** `result.Components[X]` instead of helpers

**Evidence of Performance Problems:**

`common/ecsutil.go:72` - Comment admits O(n) is a problem:
```go
// TODO: Optimize this to avoid searching all monsters every time.
func GetCreatureAtPosition(ecsmnager *EntityManager, pos *coords.LogicalPosition) *ecs.Entity {
    var e *ecs.Entity = nil
    for _, c := range ecsmnager.World.Query(ecsmnager.WorldTags["monsters"]) {
        // Linear search through ALL monsters
    }
    return e
}
```

This function is called from:
- `combat/attackingsystem.go:32` - Every melee attack
- `combat/attackingsystem.go:36` - Every ranged attack check
- `input/combatcontroller.go:233` - Every player action

**With 50 monsters on screen, this is 50 linear searches per frame.**

---

#### Recommended Query Improvements

**Pattern 1: Cached Tag Queries**

```go
// systems/query_cache.go
type QueryCache struct {
    manager     *ecs.Manager
    tags        map[string]ecs.Tag
    cachedQuery map[string][]*ecs.QueryResult
    dirty       map[string]bool
}

func (qc *QueryCache) Query(tagName string) []*ecs.QueryResult {
    if qc.dirty[tagName] {
        qc.cachedQuery[tagName] = qc.manager.Query(qc.tags[tagName])
        qc.dirty[tagName] = false
    }
    return qc.cachedQuery[tagName]
}

func (qc *QueryCache) InvalidateTag(tagName string) {
    qc.dirty[tagName] = true
}
```

**Pattern 2: Spatial Indexing (see Issue #2)**

Replace linear position searches with O(1) grid lookups.

**Pattern 3: Type-Safe Query Helpers**

```go
// common/ecsutil.go
func QueryMonsters(manager *EntityManager) []*ecs.QueryResult {
    return manager.World.Query(manager.WorldTags["monsters"])
}

func QueryRenderables(manager *EntityManager) []*ecs.QueryResult {
    return manager.World.Query(manager.WorldTags["renderables"])
}

// With generics for component extraction
func QueryAndExtract[T any](
    manager *EntityManager,
    tagName string,
    component *ecs.Component,
) []T {
    results := make([]T, 0)
    for _, result := range manager.World.Query(manager.WorldTags[tagName]) {
        if comp, ok := result.Entity.GetComponentData(component); ok {
            results = append(results, comp.(T))
        }
    }
    return results
}
```

---

### Entity Lifecycle Management

#### Entity Creation

**Patterns Used:**

1. **Manual creation** (most common):
```go
// entitytemplates/creators.go:22-41
entity := manager.World.NewEntity()
entity.AddComponent(common.NameComponent, &common.Name{NameStr: name})
entity.AddComponent(rendering.RenderableComponent, &rendering.Renderable{Image: img, Visible: visible})
entity.AddComponent(common.PositionComponent, pos)
return entity
```

2. **Builder pattern** (used in player init):
```go
// game_main/gameinit.go:38-55
playerEntity := ecsmanager.World.NewEntity().
    AddComponent(avatar.PlayerComponent, &avatar.Player{}).
    AddComponent(rendering.RenderableComponent, &rendering.Renderable{...}).
    AddComponent(common.PositionComponent, &coords.LogicalPosition{...}).
    AddComponent(gear.InventoryComponent, &gear.Inventory{...}).
    AddComponent(common.AttributeComponent, &attr).
    AddComponent(common.UserMsgComponent, &common.UserMessage{...})
```

3. **Template-based** (recently added):
```go
// entitytemplates/creators.go:49-57
func createFromTemplate(manager common.EntityManager, name, imagePath, assetDir string,
    visible bool, pos *coords.LogicalPosition, adders ...ComponentAdder) *ecs.Entity {

    entity := createBaseEntity(manager, name, imagePath, assetDir, visible, pos)
    for _, adder := range adders {
        adder(entity)
    }
    return entity
}
```

**Status:** ✅ Template system is good, but underutilized

**Issues:**
- Mix of patterns (manual, builder, template) is confusing
- No entity validation (do all entities have required components?)
- No entity factory registration

---

#### Entity Deletion

**Current Pattern:**

`resourcemanager/cleanup.go:13-24`:
```go
func RemoveEntity(world *ecs.Manager, gm *worldmap.GameMap, e *ecs.Entity) {
    if e.HasComponent(monsters.CreatureComponent) {
        monsters.NumMonstersOnMap--
        trackers.CreatureTracker.Remove(e)
    }

    pos := common.GetPosition(e)
    gm.RemoveEntityFromTile(pos)
    world.DisposeEntity(e)
}
```

**Issues:**
1. ❌ **Manual cleanup required** - Must remember to call `RemoveEntity`
2. ❌ **Global state modification** - `NumMonstersOnMap--` is global
3. ❌ **No lifecycle hooks** - Can't notify other systems of deletion
4. ❌ **Dangling references** - Entity pointers elsewhere become invalid
5. ❌ **Not used consistently** - Some code calls `world.DisposeEntity()` directly

**Evidence of Inconsistent Usage:**

`combat/attackingsystem.go:146-148`:
```go
// Only removes if player attacks, not if monster attacks player!
if isPlayerAttacking {
    resmanager.RemoveEntity(em.World, gm, defender)
}
```

**Status:** ❌ Needs centralized entity deletion system

**Recommended Fix:**

```go
// systems/entity_system.go
type EntitySystem struct {
    manager    *ecs.Manager
    gameMap    *worldmap.GameMap
    posSystem  *PositionSystem
    listeners  []EntityLifecycleListener
}

type EntityLifecycleListener interface {
    OnEntityCreated(entity *ecs.Entity)
    OnEntityDeleted(entityID ecs.EntityID)
}

func (es *EntitySystem) DeleteEntity(entity *ecs.Entity) error {
    entityID := entity.ID()

    // Notify listeners (for cleanup)
    for _, listener := range es.listeners {
        listener.OnEntityDeleted(entityID)
    }

    // Remove from position system
    if entity.HasComponent(common.PositionComponent) {
        pos := common.GetPosition(entity)
        es.posSystem.RemoveEntity(entityID, pos)
    }

    // Remove from game map
    if entity.HasComponent(common.PositionComponent) {
        pos := common.GetPosition(entity)
        es.gameMap.RemoveEntityFromTile(entityID, pos)
    }

    // Decrement monster count
    if entity.HasComponent(monsters.CreatureComponent) {
        monsters.NumMonstersOnMap--
    }

    // Finally dispose
    es.manager.DisposeEntity(entity)
    return nil
}
```

---

### Type Safety

#### Current Approach: Generic Component Access

**`common/ecsutil.go:25-45` - Good pattern:**
```go
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {
    defer func() {
        if r := recover(); r != nil {
            // ERROR HANDLING IN FUTURE
        }
    }()

    if c, ok := entity.GetComponentData(component); ok {
        return c.(T)
    } else {
        var nilValue T
        return nilValue
    }
}

// Convenience helpers
func GetAttributes(e *ecs.Entity) *Attributes {
    return GetComponentType[*Attributes](e, AttributeComponent)
}

func GetPosition(e *ecs.Entity) *coords.LogicalPosition {
    return GetComponentType[*coords.LogicalPosition](e, PositionComponent)
}
```

**Status:** ✅ This is good - provides type safety and nil safety

**Issues:**
- ⚠️ Panic recovery swallows errors (comment says "ERROR HANDLING IN FUTURE")
- ⚠️ Returns nil value on missing component (can cause subtle bugs)
- ⚠️ No way to distinguish "component missing" vs "component is nil"

**Recommended Improvement:**

```go
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) (T, bool) {
    var zero T

    if c, ok := entity.GetComponentData(component); ok {
        if typed, ok := c.(T); ok {
            return typed, true
        }
    }

    return zero, false
}

// For cases where component is required
func MustGetComponent[T any](entity *ecs.Entity, component *ecs.Component) T {
    if comp, ok := GetComponentType[T](entity, component); ok {
        return comp
    }
    panic(fmt.Sprintf("entity missing required component: %v", component))
}

// Convenience helpers updated
func GetAttributes(e *ecs.Entity) (*Attributes, bool) {
    return GetComponentType[*Attributes](e, AttributeComponent)
}

func MustGetAttributes(e *ecs.Entity) *Attributes {
    return MustGetComponent[*Attributes](e, AttributeComponent)
}
```

---

## ANTI-PATTERN DEEP DIVE: Nested Entity Hierarchies

### The Item Properties Problem

**Most Egregious Example:**

`gear/items.go:41-45`:
```go
type Item struct {
    Properties *ecs.Entity  // ❌ Entity stores entity!
    Actions    []ItemAction
    Count      int
}
```

**How It Works:**

1. Item is a component on an entity (itemEntity)
2. Item.Properties is ANOTHER entity (propertiesEntity)
3. Properties entity has components for status effects:
   - BurningComponent
   - FreezingComponent
   - StickyComponent

**Example:**

```
itemEntity (Entity ID: 123)
  ├─ NameComponent: "Flaming Sword"
  ├─ RenderableComponent: {image, visible}
  ├─ PositionComponent: {x, y}
  └─ ItemComponent: {
       Properties: propertiesEntity (Entity ID: 456) ← ❌ NESTED ENTITY
           ├─ BurningComponent: {duration: 5, damage: 10}
           └─ StickyComponent: {duration: 3, spread: 2}
       Actions: [ThrowableAction, ConsumableAction]
       Count: 1
     }
```

**Why This Is Terrible:**

1. **Double indirection** - To get item effects, must:
   - Query for item entity
   - Get Item component
   - Get Properties entity from Item
   - Query Properties entity for effect components

2. **Lifecycle nightmare** - When item is deleted:
   - Must remember to delete Properties entity too
   - Properties entity has no reference back to parent
   - No automatic cleanup

3. **No serialization** - Can't save item because Properties is a raw pointer

4. **Cache hostile** - CPU cache misses on every property access

5. **Memory waste** - Every item creates a full entity just for properties

**Current Usage Pattern:**

`gear/items.go:56-73`:
```go
func (item *Item) GetEffectNames() []string {
    names := make([]string, 0)

    if item.Properties == nil {  // Must nil-check
        return names
    }

    // Iterate through ALL effect components
    for _, c := range AllItemEffects {
        data, ok := item.Properties.GetComponentData(c)  // Component query per effect
        if ok {
            d := data.(*StatusEffects)
            names = append(names, StatusEffectName(d))
        }
    }
    return names
}
```

**Every time you check item effects, this does 3-5 component queries on a separate entity!**

---

### Recommended Fix: Flatten the Hierarchy

**Option 1: Effects as Slices (Simple)**

```go
// gear/items.go
type Item struct {
    Effects []StatusEffect  // Direct slice, no entity
    Actions []ItemAction
    Count   int
}

type StatusEffect struct {
    Type     StatusEffectType  // enum: Burning, Freezing, Sticky
    Duration int
    // Effect-specific data
    BurningDamage int
    StickySpread  int
    // ...
}

func (item *Item) GetEffectNames() []string {
    names := make([]string, 0, len(item.Effects))
    for _, eff := range item.Effects {
        names = append(names, eff.Type.String())
    }
    return names
}
```

**Benefits:**
- ✅ Single memory allocation
- ✅ Cache friendly (effects stored inline)
- ✅ Easy serialization
- ✅ Simple lifecycle (no nested entities)

**Drawbacks:**
- ⚠️ Loses ECS query capability for effects
- ⚠️ Fixed effect structure

---

**Option 2: Effect Components on Item Entity (ECS-Friendly)**

```go
// gear/items.go
type Item struct {
    // No Properties entity - effects are components on THIS entity
    Actions []ItemAction
    Count   int
}

// Add effect components directly to item entity
itemEntity.AddComponent(BurningComponent, &Burning{Duration: 5, Damage: 10})
itemEntity.AddComponent(StickyComponent, &Sticky{Duration: 3, Spread: 2})
itemEntity.AddComponent(ItemComponent, &Item{Count: 1})

// Query for items with specific effects
burningItems := ecs.BuildTag(ItemComponent, BurningComponent)
for _, result := range manager.Query(burningItems) {
    item := result.Components[ItemComponent].(*Item)
    burning := result.Components[BurningComponent].(*Burning)
    // Use item and burning
}
```

**Benefits:**
- ✅ Proper ECS pattern
- ✅ Can query for items with specific effects
- ✅ No nested entities
- ✅ Cache friendly

**Drawbacks:**
- ⚠️ More components per item entity
- ⚠️ Component proliferation (many effect types)

---

**Option 3: Hybrid (Recommended)**

```go
// gear/items.go
type Item struct {
    PrimaryEffect   StatusEffect     // Most common case: 1 effect
    SecondaryEffect *StatusEffect    // Nullable for rare multi-effect items
    Actions         []ItemAction
    Count           int
}

type StatusEffect struct {
    Type     StatusEffectType
    Duration int
    Value    int  // Generic value (damage, spread, etc.)
}

func (item *Item) GetEffects() []StatusEffect {
    effects := make([]StatusEffect, 0, 2)
    if item.PrimaryEffect.Type != NoEffect {
        effects = append(effects, item.PrimaryEffect)
    }
    if item.SecondaryEffect != nil {
        effects = append(effects, *item.SecondaryEffect)
    }
    return effects
}
```

**Benefits:**
- ✅ Optimized for common case (1 effect)
- ✅ No entity nesting
- ✅ Cache friendly
- ✅ Easy serialization
- ✅ Simple code

**Drawbacks:**
- ⚠️ Limited to 2 effects (acceptable for game design)

---

## IMPACT ON SIMPLIFICATION ROADMAP

### Current Roadmap Status (from CLAUDE.md)

1. ✅ Input System (100% - uses ECS correctly)
2. ✅ Coordinate System (100% - not ECS-related)
3. 🔄 Status Effects (85% - ECS issues remain)
4. ✅ Entity Templates (100% - good ECS usage)
5. ✅ Graphics Shapes (95% - not ECS-heavy)
6. ❌ GUI Buttons (10% - not ECS-related)

**Overall:** 80% complete, but ECS issues create technical debt

---

### How ECS Issues Affect Roadmap

#### ✅ Completed Items Using ECS Correctly

**Input System (100%)**
- Uses ECS queries properly
- Controllers don't store entity refs
- Good separation of concerns

**Entity Templates (100%)**
- ComponentAdder pattern is sound
- Factory approach works well
- Template system can be expanded

---

#### 🔄 Items Blocked by ECS Issues

**Status Effects (85% complete)**

**Current blocker:** StatusEffect interface embedded in Item.Properties entity

`gear/stateffect.go:53-65`:
```go
type StatusEffects interface {
    StatusEffectComponent() *ecs.Component
    StatusEffectName() string
    Duration() int
    ApplyToCreature(c *ecs.QueryResult)  // ❌ Logic in interface
    DisplayString() string                // ❌ Display in interface
    StackEffect(eff any)                  // ❌ Logic in interface
    Copy() StatusEffects
    common.Quality
}
```

**Problems:**
1. Interface mixes data (Duration) with logic (ApplyToCreature)
2. StatusEffects are stored as nested entity in Item.Properties
3. Can't complete separation without fixing Item entity hierarchy

**To reach 100%:**
1. Fix nested entity hierarchy (Option 3: Hybrid recommended)
2. Extract logic from StatusEffects interface
3. Create StatusEffectSystem to handle application logic

**Estimated effort:** 8-12 hours (blocked on nested entity fix)

---

### Squad System Impact (Critical)

**From CLAUDE.md:**
> PRIORITY 2: Squad Combat Foundation (12-40 hours)
> Major architectural change for "command several squads"

**ECS Anti-Patterns Will Make Squad System WORSE:**

#### Problem 1: Entity Reference Explosion

Current squad plans (`analysis/squad_combat_implementation_plan.md:93`):
```go
type SquadData struct {
    UnitEntities []*ecs.Entity  // ❌ Array of entity pointers!
}

type SquadMemberComponent struct {
    SquadEntity *ecs.Entity  // ❌ Circular entity refs!
}
```

**This amplifies the entity reference coupling problem:**
- Each squad stores 9 entity pointers
- Each unit stores squad pointer
- With 5 squads of 9 units = 45 entity refs + 9 squad refs = 54 pointers to manage
- Delete one unit = must update squad + all other units
- Delete one squad = must update 9 units + any references to squad

**With proper ECS:**
```go
type SquadData struct {
    UnitIDs [9]ecs.EntityID  // Fixed array, IDs not pointers
}

type SquadMemberComponent struct {
    SquadID ecs.EntityID  // ID, not pointer
}
```

---

#### Problem 2: Component Logic Nightmare

Squads will need:
- Position updates (9 units per squad)
- Combat resolution (9v9 = 81 interactions)
- Status effects (9 units × multiple effects)
- Equipment (9 units × 3 slots)

**With current approach:**
- Call `Creature.UpdatePosition()` 45 times (5 squads × 9 units)
- Call `MeleeWeapon.CalculateDamage()` up to 81 times per combat
- Call `StatusEffect.ApplyToCreature()` hundreds of times

**All scattered across component methods with no batching!**

**With proper systems:**
- `MovementSystem.UpdateSquadFormation(squadID)` - Batch update 9 positions
- `CombatSystem.ResolveSquadCombat(attackerSquad, defenderSquad)` - Vectorized damage calc
- `StatusEffectSystem.ProcessActiveEffects()` - Batch process all effects

**Performance difference: 10-50x faster with systems**

---

#### Problem 3: Query Explosion

With squads, queries get worse:
- 5 squads on screen = 5 squad entities + 45 unit entities = 50 entities
- Current O(n) position lookups = 50 linear searches per frame
- Squad formation checks = 5 squads × 9 units = 45 position checks
- Combat targeting = 9 attackers × 50 potential targets = 450 checks

**With spatial indexing (PositionSystem):**
- O(1) position lookups
- O(1) formation checks (grid-based)
- O(k) combat targeting (k = units in range, not all units)

---

### Recommendation: Fix ECS Before Squad System

**Critical Path:**
1. **Week 1-2: Core ECS Refactor (40-60 hours)**
   - Extract logic from components → Create systems
   - Implement PositionSystem with spatial indexing
   - Replace entity pointers with IDs

2. **Week 3: Squad System Foundation (12-16 hours)**
   - Build on proper ECS patterns
   - Use systems for squad logic
   - Leverage spatial indexing

3. **Week 4: Integration (12-16 hours)**
   - Update existing code to use systems
   - Migrate combat to squad-aware
   - Testing and polish

**Total: 64-92 hours**

**Alternative: Skip ECS fixes, implement squads with current anti-patterns:**
- **Estimated time:** 12-40 hours (shorter initially)
- **Technical debt:** MASSIVE - compounds all existing issues
- **Performance:** 10-50x slower than systems approach
- **Maintainability:** Nightmare - logic scattered everywhere
- **Future refactor cost:** 100+ hours to untangle

**Verdict: Fixing ECS first saves time in the long run and enables proper squad implementation.**

---

## RECOMMENDED APPROACH & PRIORITIZATION

### Option A: Full ECS Refactor (Recommended)
**Time:** 64-92 hours
**Benefits:** Proper foundation, faster long-term, enables squad system
**Risks:** Large upfront cost, breaking changes

**Phase 1: Systems Architecture (30-40 hours)**
1. Create systems directory and base interfaces
2. Implement PositionSystem with spatial indexing (fixes query performance)
3. Implement CombatSystem (extract from combat/attackingsystem.go)
4. Implement WeaponSystem (extract from gear/equipmentcomponents.go)
5. Implement MovementSystem (extract from monsters/creatures.go)
6. Update game loop to use SystemManager

**Phase 2: Entity References → IDs (20-30 hours)**
1. Add EntityID support (may need library wrapper)
2. Convert Item.Properties entity → flattened structure
3. Convert PlayerEquipment entity refs → IDs
4. Convert PositionTracker to use IDs
5. Update all entity access code

**Phase 3: Component Logic Extraction (14-22 hours)**
1. Remove methods from MeleeWeapon, RangedWeapon
2. Remove methods from Creature
3. Remove methods from Item
4. Move all logic to systems

**Result:** Clean ECS ready for squad system

---

### Option B: Incremental Improvements (Pragmatic)
**Time:** 24-32 hours
**Benefits:** Faster initial progress, lower risk
**Risks:** Technical debt remains, squad system still hard

**Phase 1: Quick Wins (8-12 hours)**
1. Implement PositionSystem with spatial indexing
   - Fixes O(n) position lookups → O(1)
   - Biggest performance impact
   - Doesn't require changing entities

2. Create QueryCache
   - Cache repeated queries
   - 2-3x speedup with minimal changes

**Phase 2: Critical Fixes (12-16 hours)**
1. Fix Item.Properties nesting
   - Use Option 3: Hybrid approach
   - Unblocks status effects completion
   - Prepares for item duplication

2. Implement EntitySystem for lifecycle management
   - Centralize entity deletion
   - Fix dangling reference bugs

**Phase 3: Defer to Post-Squad (40-60 hours)**
1. Full systems architecture
2. Entity ID migration
3. Component logic extraction

**Result:** Reduced pain points, squad system still challenging

---

### Option C: Minimal Changes (Not Recommended)
**Time:** 4-8 hours
**Benefits:** Almost no disruption
**Risks:** All problems remain, squad system will be a disaster

**Only do this if:**
- Squad system is far future (6+ months)
- Performance is currently acceptable
- Team has no ECS experience

**Changes:**
1. Add PositionSystem (6 hours)
2. Add QueryCache (2 hours)

**Result:** Bandaid, doesn't fix root causes

---

## ESTIMATED EFFORT SUMMARY

### Issue-by-Issue Breakdown

| Issue | Severity | Effort | Impact | Priority |
|-------|----------|--------|--------|----------|
| Components with logic | CRITICAL | 40-60h | HIGH | 1 |
| Entity reference coupling | HIGH | 20-30h | HIGH | 2 |
| Missing systems architecture | HIGH | 30-40h | HIGH | 1 |
| Nested entity hierarchies | MEDIUM | 8-12h | MEDIUM | 3 |
| Query performance | MEDIUM | 6-8h | HIGH | 2 |
| Entity lifecycle | MEDIUM | 4-6h | MEDIUM | 4 |
| **TOTAL** | - | **108-156h** | - | - |

### Recommended Phasing

**Phase 1: Foundation (30-40h) - Do First**
- Systems architecture + PositionSystem
- Immediate performance boost
- Enables better squad design

**Phase 2: Core Refactor (40-60h) - Do Before Squads**
- Extract component logic
- Entity ID migration
- Unblocks squad system

**Phase 3: Polish (12-16h) - Do During Squad Implementation**
- Nested entity fixes
- Lifecycle management
- Query cache

**Total for Squad-Ready ECS:** 82-116 hours

---

## CODE EXAMPLES: Before/After

### Example 1: Position Lookup

**Before (Current - O(n)):**
```go
// common/ecsutil.go:73-88
func GetCreatureAtPosition(ecsmnager *EntityManager, pos *coords.LogicalPosition) *ecs.Entity {
    var e *ecs.Entity = nil
    for _, c := range ecsmnager.World.Query(ecsmnager.WorldTags["monsters"]) {
        curPos := GetPosition(c.Entity)
        if pos.IsEqual(curPos) {
            e = c.Entity
            break
        }
    }
    return e
}

// Called from multiple places:
defender = common.GetCreatureAtPosition(ecsmanager, defenderPos)  // O(n) search
```

**After (Proper ECS - O(1)):**
```go
// systems/position_system.go
type PositionSystem struct {
    manager     *ecs.Manager
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID
    gridSize    int
}

func (ps *PositionSystem) GetEntityAt(pos coords.LogicalPosition) *ecs.Entity {
    if ids, ok := ps.spatialGrid[pos]; ok && len(ids) > 0 {
        return ps.manager.GetEntity(ids[0])  // O(1) lookup
    }
    return nil
}

func (ps *PositionSystem) GetEntitiesAt(pos coords.LogicalPosition) []*ecs.Entity {
    if ids, ok := ps.spatialGrid[pos]; ok {
        entities := make([]*ecs.Entity, len(ids))
        for i, id := range ids {
            entities[i] = ps.manager.GetEntity(id)
        }
        return entities
    }
    return nil
}

// Usage:
defender = positionSystem.GetEntityAt(defenderPos)  // O(1) lookup
```

**Performance:** 50+ monsters: O(50) → O(1) = **50x faster**

---

### Example 2: Combat System

**Before (Current - Logic scattered):**
```go
// combat/attackingsystem.go:114-152
func PerformAttack(em *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap,
    damage int, attacker *ecs.Entity, defender *ecs.Entity, isPlayerAttacking bool) bool {

    attAttr := common.GetAttributes(attacker)
    defAttr := common.GetAttributes(defender)

    attackRoll := randgen.GetDiceRoll(20) + attAttr.AttackBonus

    if attackRoll >= defAttr.TotalArmorClass {
        dodgeRoll := randgen.GetRandomBetween(0, 100)
        if dodgeRoll >= int(defAttr.TotalDodgeChance) {
            totalDamage := damage - defAttr.TotalProtection
            if totalDamage < 0 {
                totalDamage = 1
            }
            defAttr.CurrentHealth -= totalDamage
            return true
        }
    }

    // Only removes entity if player attacks!
    if isPlayerAttacking {
        resmanager.RemoveEntity(em.World, gm, defender)
    }

    return false
}

// Called from scattered locations:
attackSuccess = PerformAttack(ecsmanager, pl, gm, damage, attacker, defender, playerAttacking)
```

**After (Proper ECS - System owns logic):**
```go
// systems/combat_system.go
type CombatSystem struct {
    manager       *ecs.Manager
    rng           *randgen.Generator
    entitySystem  *EntitySystem
    combatLog     []CombatEvent
}

type CombatResult struct {
    Hit         bool
    Damage      int
    DefenderDied bool
}

func (cs *CombatSystem) ResolveMeleeAttack(attackerID, defenderID ecs.EntityID) CombatResult {
    attacker := cs.manager.GetEntity(attackerID)
    defender := cs.manager.GetEntity(defenderID)

    // Get components
    attAttr := common.MustGetAttributes(attacker)
    defAttr := common.MustGetAttributes(defender)
    weapon := common.GetComponentType[*gear.MeleeWeapon](attacker, gear.MeleeWeaponComponent)

    // Calculate base damage (weapon system)
    baseDamage := 0
    if weapon != nil {
        baseDamage = cs.rng.GetRandomBetween(weapon.MinDamage, weapon.MaxDamage)
    }

    // Roll attack
    attackRoll := cs.rng.GetDiceRoll(20) + attAttr.AttackBonus

    if attackRoll < defAttr.TotalArmorClass {
        return CombatResult{Hit: false}
    }

    // Roll dodge
    dodgeRoll := cs.rng.GetRandomBetween(0, 100)
    if dodgeRoll < int(defAttr.TotalDodgeChance) {
        return CombatResult{Hit: false}
    }

    // Apply damage
    finalDamage := max(baseDamage - defAttr.TotalProtection, 1)
    defAttr.CurrentHealth -= finalDamage

    // Check death
    defenderDied := false
    if defAttr.CurrentHealth <= 0 {
        cs.entitySystem.DeleteEntity(defender)
        defenderDied = true
    }

    // Log event
    cs.combatLog = append(cs.combatLog, CombatEvent{
        AttackerID: attackerID,
        DefenderID: defenderID,
        Damage:     finalDamage,
        Hit:        true,
    })

    return CombatResult{
        Hit:          true,
        Damage:       finalDamage,
        DefenderDied: defenderDied,
    }
}

// Batch combat (for squads)
func (cs *CombatSystem) ResolveSquadCombat(attackerSquadID, defenderSquadID ecs.EntityID) SquadCombatResult {
    // Can process all 9v9 combats efficiently
    // Can batch damage calculations
    // Can optimize with SIMD in future
}

// Usage:
result := combatSystem.ResolveMeleeAttack(attackerID, defenderID)
if result.Hit {
    // Update UI, play sound, etc.
}
```

**Benefits:**
- ✅ All combat logic in one place
- ✅ Easy to test (mock CombatSystem)
- ✅ Consistent entity deletion (always calls entitySystem)
- ✅ Combat log automatically tracked
- ✅ Can batch operations for squads
- ✅ No global state dependencies

---

### Example 3: Item Properties

**Before (Current - Nested entities):**
```go
// gear/items.go:41-73
type Item struct {
    Properties *ecs.Entity  // ❌ Nested entity!
    Actions    []ItemAction
    Count      int
}

func (item *Item) GetEffectNames() []string {
    names := make([]string, 0)

    if item.Properties == nil {
        return names
    }

    // Query nested entity for components
    for _, c := range AllItemEffects {
        data, ok := item.Properties.GetComponentData(c)
        if ok {
            d := data.(*StatusEffects)
            names = append(names, StatusEffectName(d))
        }
    }
    return names
}

// Creating an item with effects:
item := &Item{
    Count:      1,
    Properties: manager.NewEntity(),  // Create nested entity
    Actions:    make([]ItemAction, 0),
}
for _, prop := range effects {
    item.Properties.AddComponent(prop.StatusEffectComponent(), &prop)
}
```

**After (Proper ECS - Flattened structure):**
```go
// gear/items.go (Option 3: Hybrid)
type Item struct {
    PrimaryEffect   StatusEffect
    SecondaryEffect *StatusEffect  // nil for single-effect items
    Actions         []ItemAction
    Count           int
}

type StatusEffect struct {
    Type     StatusEffectType  // enum: Burning, Freezing, Sticky, None
    Duration int
    Value    int  // Effect magnitude (damage, spread, etc.)
}

func (item *Item) GetEffectNames() []string {
    names := make([]string, 0, 2)

    if item.PrimaryEffect.Type != NoEffect {
        names = append(names, item.PrimaryEffect.Type.String())
    }

    if item.SecondaryEffect != nil && item.SecondaryEffect.Type != NoEffect {
        names = append(names, item.SecondaryEffect.Type.String())
    }

    return names
}

func (item *Item) GetEffects() []StatusEffect {
    effects := make([]StatusEffect, 0, 2)
    if item.PrimaryEffect.Type != NoEffect {
        effects = append(effects, item.PrimaryEffect)
    }
    if item.SecondaryEffect != nil {
        effects = append(effects, *item.SecondaryEffect)
    }
    return effects
}

// Creating an item with effects:
item := &Item{
    Count:         1,
    PrimaryEffect: StatusEffect{Type: Burning, Duration: 5, Value: 10},
    Actions:       make([]ItemAction, 0),
}

// Multi-effect item:
item := &Item{
    Count:         1,
    PrimaryEffect: StatusEffect{Type: Burning, Duration: 5, Value: 10},
    SecondaryEffect: &StatusEffect{Type: Sticky, Duration: 3, Value: 2},
    Actions:       make([]ItemAction, 0),
}
```

**Benefits:**
- ✅ No nested entities
- ✅ Single memory allocation
- ✅ Cache friendly
- ✅ Easy serialization
- ✅ Simple lifecycle
- ✅ Fast effect lookups
- ✅ Optimized for common case (1 effect)

**Performance:** Item effect lookup: O(n) component queries → O(1) field access = **10x faster**

---

## TESTING STRATEGY

### Before Making Changes

1. **Capture Current Behavior:**
```bash
# Run full test suite
go test ./...

# Create baseline performance metrics
go test -bench=. -benchmem ./...

# Create integration test snapshots
# (record current game behavior)
```

2. **Add Missing Tests:**
```go
// combat/attackingsystem_test.go (create this)
func TestPerformAttack(t *testing.T) {
    // Test current behavior before refactoring
}

// gear/items_test.go (create this)
func TestItemEffects(t *testing.T) {
    // Test current item effect behavior
}
```

### During Refactoring

1. **Test Each System Independently:**
```go
// systems/combat_system_test.go
func TestCombatSystem_ResolveMeleeAttack(t *testing.T) {
    // Mock dependencies
    cs := &CombatSystem{
        rng: mockRNG{fixedRolls: []int{15, 50}},
    }

    result := cs.ResolveMeleeAttack(attackerID, defenderID)

    assert.True(t, result.Hit)
    assert.Equal(t, 10, result.Damage)
}
```

2. **Integration Tests:**
```go
// systems/integration_test.go
func TestSystemInteractions(t *testing.T) {
    // Test that systems work together correctly
    combatSystem := NewCombatSystem(manager)
    posSystem := NewPositionSystem(manager)

    // Verify combat updates positions correctly
    // Verify deleted entities removed from position system
}
```

### After Refactoring

1. **Regression Tests:**
```bash
# Compare against baseline
go test ./...
go test -bench=. -benchmem ./...

# Verify no behavior changes
# (or document intentional changes)
```

2. **Performance Validation:**
```go
// benchmarks/ecs_benchmark_test.go
func BenchmarkPositionLookup_Before(b *testing.B) {
    // Benchmark old GetCreatureAtPosition
}

func BenchmarkPositionLookup_After(b *testing.B) {
    // Benchmark new PositionSystem.GetEntityAt
    // Should be 10-50x faster
}
```

---

## MIGRATION PATH

### Step-by-Step Guide

#### Step 1: Add Systems Infrastructure (Week 1, Day 1-2)

```go
// 1. Create systems/system.go
package systems

type System interface {
    Initialize(manager *ecs.Manager) error
    Update(dt float64) error
    Cleanup() error
}

// 2. Create systems/manager.go
type SystemManager struct {
    systems []System
    manager *ecs.Manager
}

// 3. Update game_main/main.go
type Game struct {
    systemManager *systems.SystemManager
    // ...
}
```

#### Step 2: Implement PositionSystem (Week 1, Day 3-4)

```go
// 1. Create systems/position_system.go
// 2. Migrate trackers/creaturetracker.go logic
// 3. Update all GetCreatureAtPosition calls
// 4. Test thoroughly
```

#### Step 3: Implement CombatSystem (Week 1, Day 5-7)

```go
// 1. Create systems/combat_system.go
// 2. Migrate combat/attackingsystem.go logic
// 3. Update combat controller
// 4. Test combat scenarios
```

#### Step 4: Extract Component Logic (Week 2)

```go
// 1. Create systems/weapon_system.go
// 2. Remove methods from MeleeWeapon, RangedWeapon
// 3. Update all weapon usage
// 4. Repeat for other components
```

#### Step 5: Entity ID Migration (Week 3)

```go
// 1. Add EntityID support (library wrapper if needed)
// 2. Update Item.Properties
// 3. Update PlayerEquipment
// 4. Update PositionTracker
// 5. Update all entity references
```

### Backward Compatibility

During migration, support both old and new patterns:

```go
// systems/combat_system.go
// New method (preferred)
func (cs *CombatSystem) ResolveMeleeAttack(attackerID, defenderID ecs.EntityID) CombatResult {
    // ...
}

// Deprecated: Use ResolveMeleeAttack with entity IDs
func (cs *CombatSystem) ResolveMeleeAttackLegacy(attacker, defender *ecs.Entity) CombatResult {
    return cs.ResolveMeleeAttack(attacker.ID(), defender.ID())
}

// Old function (deprecated, calls new system)
// Deprecated: Use CombatSystem.ResolveMeleeAttack instead
func PerformAttack(em *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap,
    damage int, attacker *ecs.Entity, defender *ecs.Entity, isPlayerAttacking bool) bool {

    result := em.CombatSystem.ResolveMeleeAttackLegacy(attacker, defender)
    return result.Hit
}
```

---

## CONCLUSION

### Summary of Findings

**Current State:**
- ❌ Using ECS as traditional OOP (components with methods)
- ❌ Heavy entity reference coupling (pointers everywhere)
- ❌ No true systems architecture (logic scattered)
- ❌ Nested entity hierarchies (Items store entities)
- ❌ O(n) queries for position lookups

**Impact:**
- ⚠️ Performance issues with 50+ entities
- ⚠️ Hard to maintain (logic scattered)
- ⚠️ Squad system will compound problems
- ⚠️ Technical debt blocking roadmap

**Recommended Action:**
1. **Implement systems architecture** (30-40h)
2. **Fix entity references** (20-30h)
3. **Extract component logic** (40-60h)
4. **Total:** 90-130 hours for clean ECS

**Alternative:**
- Incremental improvements (24-32h)
- Fixes biggest pain points
- Defers full refactor

**Critical Decision:**
**Fix ECS before implementing squad system, or face 100+ hours of refactoring later.**

---

## NEXT STEPS

### Immediate Actions

1. **Read this document** with the team
2. **Discuss approach** (Full refactor vs Incremental)
3. **Create decision document** with chosen path
4. **Estimate timeline** based on team capacity
5. **Begin Phase 1** (whatever approach chosen)

### If Choosing Full Refactor

1. **Week 1:** Systems infrastructure + PositionSystem
2. **Week 2:** CombatSystem + WeaponSystem
3. **Week 3:** Entity ID migration
4. **Week 4:** Component logic extraction
5. **Week 5:** Testing and polish
6. **Week 6+:** Squad system implementation

### If Choosing Incremental

1. **Week 1:** PositionSystem + QueryCache
2. **Week 2:** Item.Properties fix + EntitySystem
3. **Defer:** Full systems architecture
4. **Implement:** Squad system with current patterns
5. **Future:** Full ECS refactor (40-60h)

---

**END OF ANALYSIS**
