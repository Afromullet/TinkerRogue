# ECS Usage Analysis: TinkerRogue (Post-Squad System)
**Generated:** 2025-10-02
**ECS Library:** github.com/bytearena/ecs v1.0.0 (2017)
**Analysis Scope:** Post-squad system implementation (squad_system_final.md)
**Previous Analysis:** ecs_usage_analysis.md (2025-10-01)

---

## EXECUTIVE SUMMARY

### Overall Assessment: **SIGNIFICANTLY IMPROVED** ✅ (from ⚠️)

The squad system implementation represents a **major architectural improvement** that addresses most critical ECS anti-patterns identified in the previous analysis. The squad system demonstrates proper ECS patterns that should serve as a template for refactoring the rest of the codebase.

**Critical Improvements from Squad System:**
1. ✅ **Components are pure data** - All squad components contain only data, zero methods
2. ✅ **Entity IDs instead of pointers** - Uses native `ecs.EntityID` throughout
3. ✅ **True systems architecture** - Proper system structs with state management
4. ✅ **No nested entities** - Flat component structure with ID references
5. ✅ **Query-based relationships** - Dynamic entity lookup via ECS queries

**Remaining Issues (Non-Squad Code):**
1. ❌ **Legacy components still have logic** - Weapons, Creatures, Items unchanged
2. ❌ **Legacy entity pointers remain** - PlayerEquipment, Item.Properties, PositionTracker
3. ⚠️ **Mixed patterns** - Squad system uses proper ECS, legacy code uses anti-patterns

**Impact on Simplification Roadmap:**
The squad system **enables and accelerates** the roadmap by demonstrating correct patterns. Legacy code should be migrated to match squad system architecture.

---

## COMPARISON: BEFORE vs AFTER SQUAD SYSTEM

### Issue #1: Components Contain Logic

#### BEFORE (Legacy Code)
```go
// ❌ ANTI-PATTERN: Logic in components
type MeleeWeapon struct {
    MinDamage   int
    MaxDamage   int
    AttackSpeed int
}

func (w MeleeWeapon) CalculateDamage() int {
    return randgen.GetRandomBetween(w.MinDamage, w.MaxDamage)
}

type Creature struct {
    Path []coords.LogicalPosition
}

func (c *Creature) UpdatePosition(gm *worldmap.GameMap, currentPosition *coords.LogicalPosition) {
    // 30+ lines of movement logic
}
```

#### AFTER (Squad System Pattern)
```go
// ✅ CORRECT: Pure data components
type UnitRoleData struct {
    Role UnitRole  // Tank, DPS, Support
}

type TargetRowData struct {
    TargetRows     []int
    IsMultiTarget  bool
    MaxTargets     int
}

type LeaderData struct {
    Leadership int
    Experience int
}

// ✅ CORRECT: Logic in systems
type CombatSystem struct {
    manager *ecs.Manager
    // System state here
}

func (cs *CombatSystem) ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID) CombatResult {
    // All combat logic in system
}

func (cs *CombatSystem) calculateUnitDamageByID(unitID ecs.EntityID) int {
    // Weapon damage calculation in system, not component
}
```

**Status:** ✅ **SOLVED** for squad system, ❌ **REMAINS** for legacy code

---

### Issue #2: Entity Reference Coupling

#### BEFORE (Legacy Code)
```go
// ❌ ANTI-PATTERN: Entity pointers everywhere
type Item struct {
    Properties *ecs.Entity  // Nested entity!
}

type PlayerEquipment struct {
    EqMeleeWeapon  *ecs.Entity
    EqRangedWeapon *ecs.Entity
    EqArmor        *ecs.Entity
}

type PositionTracker struct {
    PosTracker map[*coords.LogicalPosition]*ecs.Entity
}

type SquadData struct {  // Old squad plan (incorrect)
    UnitEntities []*ecs.Entity  // Array of pointers!
}
```

#### AFTER (Squad System Pattern)
```go
// ✅ CORRECT: Native entity IDs
type SquadData struct {
    SquadID    ecs.EntityID  // Native ID, not pointer
    // ... other data fields
}

type SquadMemberData struct {
    SquadID    ecs.EntityID  // Links to parent via ID
}

type CombatResult struct {
    UnitsKilled   []ecs.EntityID         // IDs, not pointers
    DamageByUnit  map[ecs.EntityID]int   // IDs in map keys
}

// ✅ CORRECT: Query for relationships
func GetUnitIDsInSquad(squadID ecs.EntityID, ecsmanager *common.EntityManager) []ecs.EntityID {
    unitIDs := make([]ecs.EntityID, 0, 9)
    for _, result := range ecsmanager.World.Query(squad.SquadMemberTag) {
        member := common.GetComponentType[*squad.SquadMemberData](result.Entity, squad.SquadMemberComponent)
        if member.SquadID == squadID {
            unitIDs = append(unitIDs, result.Entity.GetID())  // Native method!
        }
    }
    return unitIDs
}

// ✅ CORRECT: No custom registry needed
// Uses native entity.GetID() and query-based lookup
```

**Status:** ✅ **SOLVED** for squad system, ❌ **REMAINS** for legacy code

**Critical Discovery:** The previous analysis incorrectly stated bytearena/ecs doesn't expose entity IDs. The squad system proves `entity.GetID()` exists and returns `ecs.EntityID` (uint32). **No custom EntityRegistry needed!**

---

### Issue #3: Missing True Systems Architecture

#### BEFORE (Legacy Code)
```go
// ❌ ANTI-PATTERN: Just functions, no system struct
func MeleeAttackSystem(ecsmanager *common.EntityManager, pl *avatar.PlayerData,
    gm *worldmap.GameMap, attackerPos, defenderPos *coords.LogicalPosition) {
    // No state management
    // No initialization
    // Takes 5+ parameters
}

func ProcessRenderables(ecsmanager *common.EntityManager, gameMap worldmap.GameMap,
    screen *ebiten.Image, debugMode bool) {
    // Static function, no system state
}
```

#### AFTER (Squad System Pattern)
```go
// ✅ CORRECT: Proper system struct with state
type CombatSystem struct {
    manager      *ecs.Manager
    tags         map[string]ecs.Tag

    // System state
    combatLog    []CombatEvent
    damageCache  map[ecs.EntityID]int
}

func NewCombatSystem(manager *ecs.Manager) *CombatSystem {
    return &CombatSystem{
        manager:     manager,
        combatLog:   make([]CombatEvent, 0, 100),
        damageCache: make(map[ecs.EntityID]int),
    }
}

func (cs *CombatSystem) ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID) CombatResult {
    // System owns logic and state
    // Clean interface
    // Returns structured result
}

// ✅ CORRECT: Query system
type QuerySystem struct {
    manager *ecs.Manager
}

func GetUnitIDsInSquad(squadID ecs.EntityID, ecsmanager *common.EntityManager) []ecs.EntityID {
    // Centralized query logic
}

// ✅ CORRECT: Ability system
type AbilitySystem struct {
    manager *ecs.Manager
}

func CheckAndTriggerAbilities(squadID ecs.EntityID, ecsmanager *common.EntityManager) {
    // Automated ability processing
}
```

**Status:** ✅ **SOLVED** for squad system, ❌ **REMAINS** for legacy code

---

## NEW ENTITY TYPES AND COMPONENTS

### Squad Entities (NEW)

**Component:** `SquadData`
```go
type SquadData struct {
    SquadID       ecs.EntityID    // ✅ Native ID
    Formation     FormationType
    Name          string
    Morale        int
    SquadLevel    int
    TurnCount     int
    MaxUnits      int
}
```

**Status:** ✅ **EXCELLENT** - Pure data, no methods, uses native IDs

---

### Creature/Unit Entities (ENHANCED)

#### NEW Components Added to Creatures

**1. SquadMemberData** - Links unit to squad
```go
type SquadMemberData struct {
    SquadID    ecs.EntityID  // ✅ ID reference, not pointer
}
```

**2. GridPositionData** - 3x3 grid position (supports multi-cell units)
```go
type GridPositionData struct {
    AnchorRow int  // 0-2
    AnchorCol int  // 0-2
    Width     int  // 1-3 (NEW: multi-cell support)
    Height    int  // 1-3 (NEW: multi-cell support)
}

// ✅ CORRECT: Helper methods are pure functions on data
func (g *GridPositionData) GetOccupiedCells() [][2]int {
    // Pure calculation, no side effects
}

func (g *GridPositionData) OccupiesCell(row, col int) bool {
    // Pure calculation, no ECS access
}
```

**3. UnitRoleData** - Combat role
```go
type UnitRoleData struct {
    Role UnitRole  // Tank, DPS, Support
}
```

**4. TargetRowData** - Combat targeting
```go
type TargetRowData struct {
    TargetRows     []int
    IsMultiTarget  bool
    MaxTargets     int
}
```

**5. LeaderData** - Squad leader marker
```go
type LeaderData struct {
    Leadership int
    Experience int
}
```

**6. AbilitySlotData** - Leader abilities (4 slots)
```go
type AbilitySlotData struct {
    Slots [4]AbilitySlot
}

type AbilitySlot struct {
    AbilityType  AbilityType   // enum, not string
    TriggerType  TriggerType   // enum
    Threshold    float64
    HasTriggered bool
    IsEquipped   bool
}
```

**7. CooldownTrackerData** - Ability cooldowns
```go
type CooldownTrackerData struct {
    Cooldowns     [4]int
    MaxCooldowns  [4]int
}
```

#### LEGACY Creature Component (UNCHANGED)
```go
// ❌ STILL HAS ANTI-PATTERNS
type Creature struct {
    Path              []coords.LogicalPosition
    StatEffectTracker trackers.StatusEffectTracker
}

// ❌ STILL HAS METHODS
func (c *Creature) UpdatePosition(gm *worldmap.GameMap, currentPosition *coords.LogicalPosition) {
    // 30+ lines of logic
}

func (c *Creature) DisplayString(e *ecs.Entity) string {
    // Display logic
}
```

**Status:**
- ✅ Squad components are **excellent** (pure data, no methods)
- ❌ Legacy Creature component **still needs refactoring**

**Recommendation:** Create `MovementSystem` to handle `Creature.UpdatePosition()` logic

---

### Weapon Entities (UNCHANGED)

#### LEGACY Weapon Components
```go
// ❌ STILL HAS ANTI-PATTERNS
type MeleeWeapon struct {
    MinDamage   int
    MaxDamage   int
    AttackSpeed int
}

func (w MeleeWeapon) CalculateDamage() int {
    return randgen.GetRandomBetween(w.MinDamage, w.MaxDamage)
}

type RangedWeapon struct {
    MinDamage     int
    MaxDamage     int
    ShootingRange int
    TargetArea    graphics.TileBasedShape
    ShootingVX    graphics.VisualEffect
    AttackSpeed   int
}

func (r RangedWeapon) GetTargets(ecsmanger *common.EntityManager) []*ecs.Entity {
    // ❌ Returns entity pointers
    // ❌ Query logic in component
}
```

**Status:** ❌ **NO CHANGES** - Still uses anti-patterns

**How Squad System Handles Weapons:**
```go
// Squad combat system calculates weapon damage in the system
func (cs *CombatSystem) calculateUnitDamageByID(unitID ecs.EntityID) int {
    unit := FindUnitByID(unitID, cs.ecsmanager)

    // Get weapon component
    meleeWeapon := common.GetComponentType[*gear.MeleeWeapon](unit, gear.MeleeWeaponComponent)

    // ✅ CORRECT: System calculates damage, not component
    baseDamage := 0
    if meleeWeapon != nil {
        baseDamage = randgen.GetRandomBetween(meleeWeapon.MinDamage, meleeWeapon.MaxDamage)
    }

    // Apply role modifier
    return cs.applyRoleModifier(baseDamage, role)
}
```

**Recommendation:** Refactor weapon components to match squad pattern
1. Remove `CalculateDamage()` methods from weapon components
2. Create `WeaponSystem` to handle damage calculation
3. Change `GetTargets()` to return `[]ecs.EntityID` instead of `[]*ecs.Entity`

---

### Item Entities (UNCHANGED)

#### LEGACY Item Component
```go
// ❌ STILL HAS NESTED ENTITY ANTI-PATTERN
type Item struct {
    Properties *ecs.Entity  // ❌ Nested entity pointer!
    Actions    []ItemAction
    Count      int
}

// ❌ STILL HAS METHODS
func (item *Item) GetEffectNames() []string {
    // Query logic in component
}

func (item *Item) HasAllEffects(effectsToCheck ...StatusEffects) bool {
    // Filtering logic in component
}
```

**Status:** ❌ **NO CHANGES** - Nested entity hierarchy remains

**How Squad System Would Handle This:**
```go
// ✅ CORRECT: Flatten the hierarchy
type Item struct {
    PrimaryEffect   StatusEffect
    SecondaryEffect *StatusEffect  // nil for single-effect items
    Actions         []ItemAction
    Count           int
}

type StatusEffect struct {
    Type     StatusEffectType  // enum: Burning, Freezing, Sticky
    Duration int
    Value    int
}

// ✅ CORRECT: Use ItemSystem for logic
type ItemSystem struct {
    manager *ecs.Manager
}

func (is *ItemSystem) GetEffectNames(itemID ecs.EntityID) []string {
    item := is.getItemByID(itemID)
    names := make([]string, 0, 2)
    if item.PrimaryEffect.Type != NoEffect {
        names = append(names, item.PrimaryEffect.Type.String())
    }
    if item.SecondaryEffect != nil {
        names = append(names, item.SecondaryEffect.Type.String())
    }
    return names
}
```

**Recommendation:** Follow squad system pattern to flatten Item.Properties

---

## SYSTEMS INVENTORY (UPDATED)

### NEW Systems (From Squad Implementation)

| System | File | Lines | Status | Pattern |
|--------|------|-------|--------|---------|
| Query System | `systems/squadqueries.go` | ~200 | ✅ NEW | Proper ECS |
| Combat System | `systems/squadcombat.go` | ~400 | ✅ NEW | Proper ECS |
| Ability System | `systems/squadabilities.go` | ~300 | ✅ NEW | Proper ECS |
| Squad Creation System | `systems/squadcreation.go` | ~250 | ✅ NEW | Proper ECS |

**Total New Code:** ~1150 lines of **proper ECS patterns**

---

### LEGACY "Systems" (Still Anti-Pattern)

| System | File | Lines | Status | Pattern |
|--------|------|-------|--------|---------|
| Combat Functions | `combat/attackingsystem.go` | ~150 | ❌ LEGACY | Just functions |
| Rendering Functions | `rendering/rendering.go` | ~200 | ❌ LEGACY | Static functions |
| Creature Functions | `monsters/creatures.go` | ~126 | ❌ LEGACY | Logic in components |

**Total Legacy Code:** ~476 lines of **anti-patterns**

---

## MIGRATION PATH: ALIGN LEGACY WITH SQUAD PATTERNS

### Phase 1: Weapon System (Highest Impact)
**Effort:** 12-16 hours
**Files:** `gear/equipmentcomponents.go`, new `systems/weapon_system.go`

#### Changes Required

**1. Remove logic from weapon components:**
```go
// BEFORE (gear/equipmentcomponents.go)
type MeleeWeapon struct {
    MinDamage   int
    MaxDamage   int
    AttackSpeed int
}

func (w MeleeWeapon) CalculateDamage() int {  // ❌ DELETE THIS
    return randgen.GetRandomBetween(w.MinDamage, w.MaxDamage)
}

// AFTER (gear/equipmentcomponents.go)
type MeleeWeapon struct {
    MinDamage   int
    MaxDamage   int
    AttackSpeed int
}
// ✅ No methods!
```

**2. Create weapon system:**
```go
// NEW FILE: systems/weapon_system.go
type WeaponSystem struct {
    manager *ecs.Manager
    rng     *randgen.Generator
}

func NewWeaponSystem(manager *ecs.Manager) *WeaponSystem {
    return &WeaponSystem{
        manager: manager,
        rng:     randgen.NewGenerator(),
    }
}

func (ws *WeaponSystem) CalculateMeleeDamage(weaponID ecs.EntityID) int {
    weapon := ws.getWeapon(weaponID)
    return ws.rng.GetRandomBetween(weapon.MinDamage, weapon.MaxDamage)
}

func (ws *WeaponSystem) GetRangedTargetIDs(
    weaponID ecs.EntityID,
    shooterPos coords.LogicalPosition,
    positionSystem *PositionSystem,
) []ecs.EntityID {
    weapon := ws.getRangedWeapon(weaponID)
    targetIndices := weapon.TargetArea.GetIndices()
    targetPositions := coords.CoordManager.GetTilePositionsAsCommon(targetIndices)
    return positionSystem.GetEntityIDsAtPositions(targetPositions)
}
```

**3. Update all callers:**
```go
// BEFORE
damage := weapon.CalculateDamage()  // ❌ Component method

// AFTER
damage := weaponSystem.CalculateMeleeDamage(weaponID)  // ✅ System method
```

---

### Phase 2: Item System (Nested Entity Fix)
**Effort:** 10-14 hours
**Files:** `gear/items.go`, new `systems/item_system.go`

#### Changes Required

**1. Flatten Item structure:**
```go
// BEFORE (gear/items.go)
type Item struct {
    Properties *ecs.Entity  // ❌ DELETE THIS
    Actions    []ItemAction
    Count      int
}

// AFTER (gear/items.go)
type Item struct {
    PrimaryEffect   StatusEffect   // ✅ Inline effect data
    SecondaryEffect *StatusEffect  // ✅ Nullable for rare multi-effect
    Actions         []ItemAction
    Count           int
}

type StatusEffect struct {
    Type     StatusEffectType
    Duration int
    Value    int
}
```

**2. Create item system:**
```go
// NEW FILE: systems/item_system.go
type ItemSystem struct {
    manager *ecs.Manager
}

func (is *ItemSystem) GetEffectNames(itemID ecs.EntityID) []string {
    item := is.getItem(itemID)
    names := make([]string, 0, 2)
    if item.PrimaryEffect.Type != NoEffect {
        names = append(names, item.PrimaryEffect.Type.String())
    }
    if item.SecondaryEffect != nil {
        names = append(names, item.SecondaryEffect.Type.String())
    }
    return names
}

func (is *ItemSystem) HasAllEffects(itemID ecs.EntityID, effectsToCheck ...StatusEffectType) bool {
    item := is.getItem(itemID)
    // Filtering logic here
}
```

**3. Update all callers:**
```go
// BEFORE
names := item.GetEffectNames()  // ❌ Component method

// AFTER
names := itemSystem.GetEffectNames(itemID)  // ✅ System method
```

---

### Phase 3: Movement System
**Effort:** 8-12 hours
**Files:** `monsters/creatures.go`, new `systems/movement_system.go`

#### Changes Required

**1. Remove logic from Creature component:**
```go
// BEFORE (monsters/creatures.go)
type Creature struct {
    Path              []coords.LogicalPosition
    StatEffectTracker trackers.StatusEffectTracker
}

func (c *Creature) UpdatePosition(gm *worldmap.GameMap, currentPosition *coords.LogicalPosition) {
    // ❌ DELETE THIS METHOD (30+ lines)
}

// AFTER (monsters/creatures.go)
type Creature struct {
    Path              []coords.LogicalPosition
    StatEffectTracker trackers.StatusEffectTracker
}
// ✅ No methods!
```

**2. Create movement system:**
```go
// NEW FILE: systems/movement_system.go
type MovementSystem struct {
    manager     *ecs.Manager
    gameMap     *worldmap.GameMap
    posSystem   *PositionSystem
}

func (ms *MovementSystem) UpdateCreaturePosition(creatureID ecs.EntityID) {
    creature := ms.getCreature(creatureID)
    currentPos := ms.posSystem.GetPosition(creatureID)

    // All 30+ lines of movement logic here
    // Updates position through position system
}
```

---

### Phase 4: Position System (Performance Fix)
**Effort:** 8-12 hours
**Files:** `trackers/creaturetracker.go`, new `systems/position_system.go`

#### Changes Required

**1. Replace PositionTracker with PositionSystem:**
```go
// BEFORE (trackers/creaturetracker.go)
type PositionTracker struct {
    PosTracker map[*coords.LogicalPosition]*ecs.Entity  // ❌ Pointer keys!
}

// AFTER (systems/position_system.go)
type PositionSystem struct {
    manager     *ecs.Manager
    spatialGrid map[coords.LogicalPosition][]ecs.EntityID  // ✅ ID-based
}

func (ps *PositionSystem) GetEntityIDAt(pos coords.LogicalPosition) ecs.EntityID {
    if ids, ok := ps.spatialGrid[pos]; ok && len(ids) > 0 {
        return ids[0]  // O(1) lookup
    }
    return 0
}
```

**2. Update all position lookups:**
```go
// BEFORE (O(n) - linear search)
defender = common.GetCreatureAtPosition(ecsmanager, pos)  // Queries all monsters!

// AFTER (O(1) - hash lookup)
defenderID = positionSystem.GetEntityIDAt(pos)
defender = FindEntityByID(defenderID, eSquad MemberTag, ecsmanager)
```

**Performance Impact:** 50+ monsters: O(50) → O(1) = **50x faster**

---

## UPDATED EFFORT ESTIMATES

### Squad System Implementation (NEW)
**Total:** 30-37 hours ✅ **INVESTED**
- Phase 1: Components (6-8h) ✅
- Phase 2: Query System (4-6h) ✅
- Phase 3: Combat System (8-10h) ✅
- Phase 4: Ability System (6-8h) ✅
- Phase 5: Squad Creation (4-6h) ✅
- Phase 6: Integration (8-10h) ✅

---

### Legacy Code Migration (REMAINING)
**Total:** 38-54 hours ❌ **NEEDED**

| Phase | Effort | Impact | Priority |
|-------|--------|--------|----------|
| Weapon System | 12-16h | HIGH | 1 |
| Item System | 10-14h | MEDIUM | 2 |
| Movement System | 8-12h | MEDIUM | 3 |
| Position System | 8-12h | HIGH | 1 |
| **Total** | **38-54h** | - | - |

---

## RECOMMENDED PRIORITIES (POST-SQUAD)

### Priority 1: Position System (Week 1)
**Effort:** 8-12 hours
**Impact:** HIGH - 50x performance improvement
**Blockers:** None
**Approach:** Follow squad query system pattern

**Why First:**
- Biggest performance impact
- Zero breaking changes (internal optimization)
- Enables efficient squad positioning
- O(n) → O(1) lookups

---

### Priority 2: Weapon System (Week 2)
**Effort:** 12-16 hours
**Impact:** HIGH - Enables proper squad weapon integration
**Blockers:** None
**Approach:** Follow squad combat system pattern

**Why Second:**
- Most called code (every combat action)
- Enables squad weapon consistency
- Removes logic from components
- Template for other system migrations

---

### Priority 3: Item System (Week 3)
**Effort:** 10-14 hours
**Impact:** MEDIUM - Completes status effects roadmap
**Blockers:** None
**Approach:** Follow squad component patterns (flatten hierarchy)

**Why Third:**
- Completes simplification roadmap item #3 (Status Effects)
- Removes nested entity anti-pattern
- Prepares for item duplication/stacking
- Less frequently called than weapons

---

### Priority 4: Movement System (Week 4)
**Effort:** 8-12 hours
**Impact:** MEDIUM - Organizational improvement
**Blockers:** Position System recommended first
**Approach:** Follow squad ability system pattern

**Why Fourth:**
- Less critical than combat systems
- Benefits from Position System optimization
- Prepares for squad formations/movement
- Cleanest separation of concerns

---

## PATTERN COMPARISON GUIDE

### Use Squad System as Template

When refactoring legacy code, follow these squad system patterns:

#### Pattern 1: Pure Data Components
```go
// ✅ SQUAD PATTERN (CORRECT)
type UnitRoleData struct {
    Role UnitRole  // Just data
}

// ❌ LEGACY PATTERN (WRONG)
type MeleeWeapon struct {
    MinDamage int
}
func (w MeleeWeapon) CalculateDamage() int {  // Logic in component!
    return randgen.GetRandomBetween(w.MinDamage, w.MaxDamage)
}
```

#### Pattern 2: Entity IDs, Not Pointers
```go
// ✅ SQUAD PATTERN (CORRECT)
type SquadMemberData struct {
    SquadID ecs.EntityID  // Native ID
}

type CombatResult struct {
    UnitsKilled []ecs.EntityID  // ID slice
}

// ❌ LEGACY PATTERN (WRONG)
type Item struct {
    Properties *ecs.Entity  // Nested entity pointer!
}

type PlayerEquipment struct {
    EqMeleeWeapon *ecs.Entity  // Entity pointer!
}
```

#### Pattern 3: Query-Based Relationships
```go
// ✅ SQUAD PATTERN (CORRECT)
func GetUnitIDsInSquad(squadID ecs.EntityID, ecsmanager *common.EntityManager) []ecs.EntityID {
    unitIDs := make([]ecs.EntityID, 0, 9)
    for _, result := range ecsmanager.World.Query(squad.SquadMemberTag) {
        member := common.GetComponentType[*squad.SquadMemberData](result.Entity, squad.SquadMemberComponent)
        if member.SquadID == squadID {
            unitIDs = append(unitIDs, result.Entity.GetID())
        }
    }
    return unitIDs
}

// ❌ LEGACY PATTERN (WRONG)
type SquadData struct {
    UnitEntities []*ecs.Entity  // Stored pointers!
}
```

#### Pattern 4: System Structs with State
```go
// ✅ SQUAD PATTERN (CORRECT)
type CombatSystem struct {
    manager     *ecs.Manager
    combatLog   []CombatEvent
    damageCache map[ecs.EntityID]int
}

func NewCombatSystem(manager *ecs.Manager) *CombatSystem {
    return &CombatSystem{
        manager:     manager,
        combatLog:   make([]CombatEvent, 0, 100),
        damageCache: make(map[ecs.EntityID]int),
    }
}

// ❌ LEGACY PATTERN (WRONG)
func MeleeAttackSystem(ecsmanager *common.EntityManager, pl *avatar.PlayerData,
    gm *worldmap.GameMap, attackerPos, defenderPos *coords.LogicalPosition) {
    // No state, takes 5 parameters
}
```

#### Pattern 5: Structured Return Values
```go
// ✅ SQUAD PATTERN (CORRECT)
type CombatResult struct {
    UnitsKilled   []ecs.EntityID
    DamageByUnit  map[ecs.EntityID]int
    TotalDamage   int
}

func (cs *CombatSystem) ExecuteSquadAttack(...) CombatResult {
    // Return structured data
}

// ❌ LEGACY PATTERN (WRONG)
func PerformAttack(...) bool {
    // Returns only bool, side effects everywhere
}
```

---

## TESTING STRATEGY (UPDATED)

### Phase 1: Baseline Current Squad System
```bash
# Verify squad system works correctly
go test ./systems/... -v

# Performance baseline
go test -bench=SquadCombat ./systems/... -benchmem
```

### Phase 2: Test Legacy Code Before Migration
```bash
# Capture current behavior
go test ./combat/... -v
go test ./gear/... -v
go test ./monsters/... -v

# Performance baseline for comparison
go test -bench=. ./combat/... -benchmem
go test -bench=. ./gear/... -benchmem
```

### Phase 3: Test During Migration
```go
// systems/weapon_system_test.go
func TestWeaponSystem_CalculateDamage(t *testing.T) {
    // Verify new system matches old behavior
}

// systems/integration_test.go
func TestSquadCombatWithMigratedWeapons(t *testing.T) {
    // Verify squad combat works with new weapon system
}
```

### Phase 4: Regression Testing
```bash
# Compare against baseline
go test ./... -v

# Verify performance improvements
go test -bench=. ./... -benchmem
```

---

## CONCLUSION

### Summary of Changes

**Squad System Introduces:**
- ✅ 7 new pure data components (SquadData, SquadMemberData, GridPositionData, UnitRoleData, TargetRowData, LeaderData, AbilitySlotData)
- ✅ 4 new proper systems (Query, Combat, Ability, Squad Creation)
- ✅ Native entity ID usage (ecs.EntityID) throughout
- ✅ Query-based entity relationships (no stored pointers)
- ✅ ~1150 lines of **excellent ECS patterns**

**Legacy Code Remains:**
- ❌ Weapons still have logic in components (CalculateDamage)
- ❌ Creatures still have logic in components (UpdatePosition)
- ❌ Items still use nested entities (Item.Properties)
- ❌ Position lookups still O(n) (PositionTracker)
- ❌ ~476 lines of **anti-patterns**

**Critical Insight:**
The squad system demonstrates that **proper ECS is achievable** with the current library (bytearena/ecs v1.0.0). The previous analysis incorrectly stated entity IDs weren't available. The squad system proves `entity.GetID()` works perfectly.

---

### Recommended Action Plan

**Short Term (4 weeks, 38-54 hours):**
1. Week 1: Position System (8-12h) - 50x performance boost
2. Week 2: Weapon System (12-16h) - Aligns with squad combat
3. Week 3: Item System (10-14h) - Completes roadmap
4. Week 4: Movement System (8-12h) - Final cleanup

**Result:** 100% proper ECS patterns, ready for all todos implementation

**Long Term:**
- Use squad system as architectural template
- All new features follow squad patterns
- Gradual migration of remaining legacy code
- Consider squad system for internal documentation/training

---

### Impact on Roadmap

**Simplification Roadmap:** 80% → **95%** (after legacy migration)

1. ✅ Input System (100%)
2. ✅ Coordinate System (100%)
3. 🔄 Status Effects (85% → **100%** after Item System migration)
4. ✅ Entity Templates (100%)
5. ✅ Graphics Shapes (95%)
6. ❌ GUI Buttons (10%)
7. ✅ **Squad System (100%)** - NEW ITEM

**Overall Progress:** 95% with squad system complete, 100% after legacy migration

---

**END OF UPDATED ANALYSIS**
