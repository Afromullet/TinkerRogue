# Squad Combat Architecture: Comprehensive Comparison

**Last Updated:** 2025-10-01
**Purpose:** Help you choose the best squad combat architecture for TinkerRogue
**Status:** Decision-Ready Analysis

---

## Executive Summary

Three distinct architectural approaches have been designed for implementing squad-based tactical combat in TinkerRogue:

1. **Approach 1: Component-Rich Entity Architecture** - Pure ECS, squads as entities with rich component composition
2. **Approach 2: Manager-Centric Architecture** - Squads as structs, centralized SquadManager, procedural combat
3. **Approach 3: Hybrid Command-Pattern Architecture** - Strategy patterns for formations, command pattern for actions, maximum tactical depth

**Key Differences at a Glance:**
- **Approach 1** optimizes for **ECS purity and component-based extensibility**
- **Approach 2** optimizes for **simplicity and centralized control**
- **Approach 3** optimizes for **tactical depth and professional-grade TRPG mechanics**

**Quick Recommendation:**
- **For fastest implementation:** Approach 2 (20-30h)
- **For ECS adherence:** Approach 1 (28-32h)
- **For Fire Emblem/FFT-level depth:** Approach 3 (24-32h)

---

## Quick Comparison Table

| Dimension | Approach 1: Component-Rich | Approach 2: Manager-Centric | Approach 3: Hybrid Command |
|-----------|---------------------------|----------------------------|---------------------------|
| **Complexity** | Medium-High | Medium | High |
| **Implementation Time** | 28-32 hours | 20-30 hours | 24-32 hours |
| **Lines of Code** | ~1400 LOC | ~1200 LOC | ~1500 LOC |
| **ECS Alignment** | Pure (100%) | Hybrid (60%) | Hybrid (70%) |
| **Extensibility** | High (component-based) | Medium (manager methods) | Very High (interfaces) |
| **Maintainability** | High (many small files) | Medium (large manager) | Medium (many abstractions) |
| **Tactical Depth** | Medium-High | Medium | Very High |
| **Learning Curve** | Medium (ECS patterns) | Low (OOP familiar) | High (design patterns) |
| **Best For** | ECS purists, flexible queries | Teams wanting simplicity | Professional TRPG depth |
| **Code Organization** | Distributed (components) | Centralized (manager) | Layered (interfaces) |
| **Testing Ease** | High (component mocking) | Medium (full setup) | Very High (interface mocking) |
| **Performance** | Excellent (component caching) | Good (direct struct access) | Good (interface overhead minimal) |
| **Debug Complexity** | Medium (distributed state) | Low (single authority) | Medium (command tracing) |

---

## Side-by-Side Feature Comparison

### Squad Representation

**Approach 1:**
```go
// Squad is a separate ECS entity
squadEntity := world.Create(
    squad.SquadComponentType,
    coords.LogicalPositionComponentType,
    ecs.RenderComponent,
)

// Units are entities with SquadMemberComponent
unitEntity.AddComponent(squad.SquadMemberComponentType)
member := squad.SquadMemberComponentType.Get(unitEntity)
member.SquadEntity = squadEntity  // Bidirectional link
```

**Approach 2:**
```go
// Squad is a struct, not an entity
type Squad struct {
    ID         int
    Name       string
    Position   coords.LogicalPosition
    Grid       [3][3]*GridSlot  // 3x3 grid
    UnitCount  int
}

// Managed by centralized SquadManager
squadMgr := squads.NewSquadManager(world, coordMgr)
squad, _ := squadMgr.CreateSquad("Alpha Squad", pos, "player")
```

**Approach 3:**
```go
// Hybrid: Squad is BOTH entity AND data structure
type Squad struct {
    EntityID   ecs.EntityID  // For world positioning
    LogicalPos coords.LogicalPosition
    Formation  FormationPattern  // Strategy interface
    Units      [9]*ecs.Entity
    Leader     *ecs.Entity
}

// Squads registered in GameWorld
world.RegisterSquad(squad)
```

**Comparison:**
- **Approach 1:** Most ECS-pure, squads queryable like any entity
- **Approach 2:** Simplest data model, no bidirectional links
- **Approach 3:** Best of both worlds, entity for world, struct for composition

---

### 3x3 Grid System

**Approach 1:**
```go
// GridPosition stored in SquadMemberComponent
type SquadMemberComponent struct {
    SquadEntity    *donburi.Entry
    GridPosition   GridPosition  // {Row, Col}
    IsLeader       bool
    UnitRole       UnitRole  // FRONT_LINE, SUPPORT, RANGED
}

// Grid occupancy tracked separately
type SquadGrid struct {
    Occupied [3][3]bool
    UnitMap  [3][3]*donburi.Entry
}
```

**Approach 2:**
```go
// Grid embedded in Squad struct
type GridSlot struct {
    Row    int
    Col    int
    Unit   *ecs.Entry
    IsSlot bool  // false for blocked slots
}

type Squad struct {
    Grid [3][3]*GridSlot  // Direct array
}

// Access: squad.Grid[row][col].Unit
```

**Approach 3:**
```go
// Grid is simple array, positioning handled by formation
type Squad struct {
    Units [9]*ecs.Entity  // Flat array (0-8)
}

// Grid position calculated from index
func IndexToGridPos(idx int) GridPosition {
    return GridPosition{Row: idx / 3, Col: idx % 3}
}

// Formation strategy rearranges units
formation.ArrangUnits(units) [9]*ecs.Entity
```

**Comparison:**
- **Approach 1:** Most flexible (components queryable), highest overhead
- **Approach 2:** Most intuitive (2D array), easiest to visualize
- **Approach 3:** Most efficient (flat array), formation-driven

---

### Leader Abilities

**Approach 1:**
```go
// LeaderComponent on unit entity
type LeaderComponent struct {
    AbilitySlots   [4]*AbilityInstance
    Leadership     int
    Experience     int
}

// Abilities are data-driven
type AbilityInstance struct {
    AbilityID      string  // "FireballLv2", "Rally"
    Cooldown       int
    MaxCooldown    int
    TargetType     AbilityTargetType
}

// Registry holds definitions
var AbilityRegistry = map[string]*AbilityDefinition{
    "Rally": { ... },
}
```

**Approach 2:**
```go
// Abilities stored in centralized registry
type Ability struct {
    ID          string
    Name        string
    APCost      int
    Execute     func(caster *ecs.Entry, target *Squad, sm *SquadManager) error
}

type AbilityRegistry struct {
    abilities map[string]*Ability
}

// Leader data in UnitDataComponent
type UnitDataComponent struct {
    Type         UnitType  // Leader or Regular
    AbilitySlots [4]string  // Ability IDs
}
```

**Approach 3:**
```go
// Abilities are interfaces (strategy pattern)
type LeaderAbility interface {
    Name() string
    Execute(caster *Squad, target interface{}, world *GameWorld) error
    Validate(caster *Squad, target interface{}, world *GameWorld) error
    CurrentCooldown() int
    SetCooldown(turns int)
}

// Concrete implementations
type RallyAbility struct {
    BaseAbility
    HealAmount int
}

// Stored in LeaderComponent
type LeaderComponent struct {
    AbilitySlots [4]LeaderAbility  // Interface slots
}
```

**Comparison:**
- **Approach 1:** Data-driven, easy to serialize, limited runtime flexibility
- **Approach 2:** Function-based, simplest implementation, hardcoded logic
- **Approach 3:** Interface-based, maximum extensibility, runtime swappable

---

### Combat Resolution (Pooled Damage)

**Approach 1:**
```go
// ExecuteSquadAttack in combat/squaddamage.go
func ExecuteSquadAttack(attackerSquad, defenderSquad *donburi.Entry, world donburi.World) *PooledDamageResult {
    // Step 1: Pool damage from all attackers
    for _, unitEntity := range attackerSquadData.UnitEntities {
        damage := calculateUnitDamage(unitEntity, world)
        member := squad.SquadMemberComponentType.Get(unitEntity)
        damage = applyRoleModifier(damage, member.UnitRole)
        damagePool += damage
    }

    // Step 2: Distribute to defenders (FRONT_LINE first)
    defenders := sortDefendersByRole(defenderSquadData.UnitEntities)
    for _, defenderUnit := range defenders {
        damageToUnit := min(hp.Current, remainingDamage)
        hp.Current -= damageToUnit
        remainingDamage -= damageToUnit
    }
}
```

**Approach 2:**
```go
// ExecuteSquadCombat in squads/combat.go
func (sm *SquadManager) ExecuteSquadCombat(attackerID, defenderID int) (*SquadCombatResult, error) {
    // Phase 1: Pool attacks
    for _, unit := range attackerUnits {
        damage := sm.calculateUnitDamage(unit, target)
        attackerDamage += damage
    }

    // Phase 2: Apply pooled damage (front to back)
    for row := 0; row < 3 && remainingDamage > 0; row++ {
        for col := 0; col < 3 && remainingDamage > 0; col++ {
            unit := squad.Grid[row][col].Unit
            // Apply damage front-to-back
        }
    }
}
```

**Approach 3:**
```go
// ExecuteSquadCombat in combat/squadcombat.go
func ExecuteSquadCombat(attacker, defender *Squad, ctx TacticalContext) CombatResult {
    // Get formation modifiers
    attackMods := attacker.Formation.GetCombatModifiers(ctx)
    defenseMods := defender.Formation.GetCombatModifiers(ctx)

    // Individual attacks with formation bonuses
    for i, attackUnit := range attacker.Units {
        defenderUnit := SelectDefenderTarget(defender, IndexToGridPos(i), ctx)
        attack := PerformIndividualAttack(attackUnit, defenderUnit, attackMods, defenseMods)
        result.TotalDamage += attack.Damage
    }

    // Damage applied to squad's pooled HP
    defender.TotalHP -= result.TotalDamage
}
```

**Comparison:**
- **Approach 1:** Unit-level damage distribution, kills tracked individually
- **Approach 2:** Row-by-row damage distribution, position matters tactically
- **Approach 3:** Formation-aware targeting, modifiers create rock-paper-scissors

---

### Formation System

**Approach 1:**
```go
// FormationType enum
type FormationType int
const (
    FORMATION_BALANCED FormationType = iota
    FORMATION_DEFENSIVE
    FORMATION_OFFENSIVE
    FORMATION_RANGED
)

// Stored in SquadComponent
type SquadComponent struct {
    Formation FormationType
}

// Combat modifiers applied manually in combat code
```

**Approach 2:**
```go
// No explicit formation system
// Positioning handled by initial placement
// Could add formations as manager methods:

func (sm *SquadManager) SetFormation(squadID int, formation string) {
    // Rearrange units based on formation string
}
```

**Approach 3:**
```go
// Formation is a strategy interface
type FormationPattern interface {
    Name() string
    ArrangUnits(units []*ecs.Entity) [9]*ecs.Entity
    GetCombatModifiers(ctx TacticalContext) CombatModifiers
    Validate(units []*ecs.Entity) error
}

// Concrete implementations
type DefensiveFormation struct{}  // +5 DEF, -2 ATK
type OffensiveFormation struct{}  // +5 ATK, -2 DEF
type BalancedFormation struct{}   // +2 ATK, +2 DEF

// Stored in Squad
type Squad struct {
    Formation FormationPattern  // Interface, swappable at runtime
}
```

**Comparison:**
- **Approach 1:** Static formations, requires code changes for new formations
- **Approach 2:** No formation system initially (simplest), add later if needed
- **Approach 3:** Dynamic formations, new formations are simple classes, pluggable

---

### Integration with Existing Code

**Approach 1: PerformAttack() Wrapper**
```go
// Reuse per-unit damage calculation
func calculateUnitDamage(unitEntity *donburi.Entry, world donburi.World) int {
    stats := ecs.StatsComponent.Get(unitEntity)
    gear := ecs.EquippedGearComponent.Get(unitEntity)
    baseDamage := stats.Strength + gear.Weapon.Damage
    roll := rollD20()
    if roll >= 18 { baseDamage = int(float64(baseDamage) * 1.5) }
    return baseDamage
}
```

**Approach 2: PerformAttack() Wrapper**
```go
// Wrapper extracts damage without applying
func (sm *SquadManager) calculateUnitDamage(attacker, defender *ecs.Entry) int {
    // Clone defender HP
    originalHP := getHP(defender)
    combat.PerformAttack(attacker, defender)
    damage := originalHP - getHP(defender)
    setHP(defender, originalHP)  // Restore
    return damage
}
```

**Approach 3: PerformAttack() Wrapper**
```go
// Clean wrapper with formation modifiers
func PerformIndividualAttack(attacker, defender *ecs.Entity, attackMods, defenseMods CombatModifiers) IndividualAttack {
    attackerStats := attacker.GetComponent("StatsComponent").(*StatsComponent)
    modifiedAttack := attackerStats.Attack + attackMods.AttackBonus
    // Use existing d20 combat system
    return attack
}
```

**Comparison:**
- **Approach 1:** Most faithful to existing system, per-unit variance preserved
- **Approach 2:** Hacky HP cloning, but preserves existing code completely
- **Approach 3:** Cleanest wrapper, modifiers integrated naturally

---

### Unit Size Handling (2x1, 1x2)

**Approach 1:**
```go
// UnitSizeComponent
type UnitSizeComponent struct {
    Width  int  // 1 or 2
    Height int  // 1 or 2
}

// Grid placement validates size
func (g *SquadGrid) PlaceUnit(unit *donburi.Entry, pos GridPosition, size UnitSizeComponent) bool {
    for r := 0; r < size.Height; r++ {
        for c := 0; c < size.Width; c++ {
            if g.Occupied[pos.Row+r][pos.Col+c] { return false }
        }
    }
    // Occupy all cells
}
```

**Approach 2:**
```go
// PlaceUnitInGrid method
func (s *Squad) PlaceUnitInGrid(unit *ecs.Entry, row, col int) error {
    unitData := GetUnitData(unit)

    switch unitData.Size {
    case Size2x1:
        s.Grid[row][col].Unit = unit
        s.Grid[row][col+1].Unit = unit  // Same reference
        s.Grid[row][col+1].IsSlot = false  // Blocked
    case Size1x2:
        s.Grid[row][col].Unit = unit
        s.Grid[row+1][col].Unit = unit
        s.Grid[row+1][col].IsSlot = false
    }
}
```

**Approach 3:**
```go
// UnitPlacement helper
type UnitPlacement struct {
    Entity    *ecs.Entity
    Size      UnitSize
    AnchorPos GridPosition
}

func (up *UnitPlacement) CalculateOccupiedSlots() []int {
    switch up.Size {
    case UnitSize2x1:
        return []int{anchor, anchor + 1}
    case UnitSize1x2:
        return []int{anchor, anchor + 3}
    }
}
```

**Comparison:**
- **Approach 1:** Most flexible (component-based), easy to query large units
- **Approach 2:** Most intuitive (grid-based), duplicate references in grid
- **Approach 3:** Most efficient (flat array calculations), index-based

---

## Recommendations by Scenario

### If You Prioritize ECS Purity
**Choose: Approach 1 (Component-Rich Entity Architecture)**

**Why:**
- Squads are first-class entities, queryable with `query.NewQuery(filter.Contains(squad.SquadComponentType))`
- All behavior emerges from component composition
- No external managers breaking ECS patterns
- Easy to add new components (MoraleComponent, FormationComponent, etc.)

**Trade-off:** Higher component overhead, more files to maintain

---

### If You Want Fastest Implementation
**Choose: Approach 2 (Manager-Centric Architecture)**

**Why:**
- Simplest data model (Squad is a struct)
- Centralized logic (all in SquadManager)
- Familiar OOP patterns (no complex abstractions)
- Fastest to prototype and test

**Trade-off:** Manager becomes large (~500-800 LOC), less ECS-pure

---

### If You Want Maximum Tactical Depth
**Choose: Approach 3 (Hybrid Command-Pattern Architecture)**

**Why:**
- Formation switching creates meaningful mid-battle decisions
- Strategy pattern allows unlimited formation types (add new formations without touching existing code)
- Command pattern enables preview system (Fire Emblem-style combat forecasting)
- Interface-based abilities are highly extensible
- Best supports future features (terrain bonuses, morale, multi-target abilities)

**Trade-off:** Highest initial complexity, steeper learning curve

---

### If You're Building for Long-Term Extensibility
**Choose: Approach 3 (Hybrid Command-Pattern Architecture)**

**Why:**
- New formations: implement `FormationPattern` interface (no existing code changes)
- New abilities: implement `LeaderAbility` interface (drop-in additions)
- Easy to serialize/save (formations and abilities are data)
- Supports modding (players can add formations via scripts)
- Command pattern enables replay/undo systems

**Trade-off:** More abstractions to learn initially

---

### If You Want Simplicity and Debuggability
**Choose: Approach 2 (Manager-Centric Architecture)**

**Why:**
- Single source of truth (SquadManager)
- Predictable execution flow (procedural combat)
- Easy to debug (check manager state)
- Clear ownership (no hunting for "who owns this data?")
- Familiar pattern for most developers

**Trade-off:** Less extensible, manager becomes large

---

### If You Need Best Testing Support
**Choose: Approach 3 (Hybrid Command-Pattern Architecture)**

**Why:**
- Interface mocking is trivial (`type MockFormation struct { ... }`)
- Commands testable in isolation (validate/execute/preview)
- Formation strategies testable independently
- Abilities testable independently
- No need to mock entire world for unit tests

**Trade-off:** More test boilerplate

---

## Migration Paths

### Can Approach 2 Evolve into Approach 3?

**Yes, incrementally:**

**Phase 1:** Start with Approach 2 (Squad struct + SquadManager)
```go
// Week 1: Basic squad combat (20h)
type Squad struct {
    Grid [3][3]*GridSlot
}
squadMgr := NewSquadManager(world)
```

**Phase 2:** Add formation interface
```go
// Week 2: Formation system (+4h)
type FormationPattern interface {
    ArrangUnits(units []*ecs.Entity) [9]*ecs.Entity
}

type Squad struct {
    Grid      [3][3]*GridSlot
    Formation FormationPattern  // NEW
}
```

**Phase 3:** Add command pattern
```go
// Week 3: Command pattern (+5h)
type AttackSquadCommand struct {
    AttackerID int
    DefenderID int
}

// Wrap existing SquadManager.ExecuteSquadCombat()
cmd.Execute(world)
```

**Phase 4:** Add leader abilities
```go
// Week 4: Abilities (+8h)
type LeaderAbility interface { ... }
```

**Result:** Same feature set as Approach 3, incremental delivery, lower risk

---

### Can Approach 1 Be Refactored to Approach 2?

**Yes, but painful:**

**Challenge:** Moving from component-based to manager-based requires:
1. Extracting squad state from components into Squad struct
2. Creating SquadManager to replace distributed systems
3. Replacing component queries with manager lookups
4. Potentially breaking existing ECS workflows

**Recommendation:** Don't start with Approach 1 if you might want Approach 2 later. Go Manager-Centric from the start if simplicity is priority.

---

### What's the Lowest-Risk Path?

**Recommended: Start with Approach 2, evolve to Approach 3 as needed**

**Timeline:**
- **Week 1 (20h):** Implement Approach 2 (basic squad combat working)
- **Week 2 (4h):** Add formation interface if tactical depth needed
- **Week 3 (5h):** Add command pattern if preview/undo needed
- **Week 4 (8h):** Add leader abilities if customization needed

**Benefits:**
- Incremental delivery (basic combat works after Week 1)
- Lower risk (can stop at any phase if time runs out)
- Easier testing (simple system tested first, complexity added gradually)
- Budget-friendly (can defer advanced features if budget tight)

**Total Effort:** 20-37 hours depending on how far you go

---

## My Recommendation

**Context:**
- Entity template system complete (flexible entity creation available)
- Input system complete (controllers ready for squad actions)
- Coordinate system complete (positioning infrastructure solid)
- Project emphasizes tactical gameplay ("command several squads")

**Recommendation: Hybrid Approach (2 → 3 Incremental)**

### Phase 1: Implement Approach 2 Foundation (20 hours)

**What to build:**
```go
// squads/squad.go
type Squad struct {
    ID         int
    Position   coords.LogicalPosition
    Grid       [3][3]*GridSlot
    UnitCount  int
}

// squads/squadmanager.go
type SquadManager struct {
    squads        map[int]*Squad
    positionIndex map[coords.LogicalPosition]*Squad
}

// squads/combat.go
func (sm *SquadManager) ExecuteSquadCombat(attackerID, defenderID int) (*SquadCombatResult, error) {
    // Basic pooled damage combat
}
```

**Deliverables:**
- Squad creation and movement working
- Basic squad-vs-squad combat functional
- Front-to-back damage distribution implemented
- Integrates with existing spawning/input

**Timeline:** Week 1-2 (20 hours)

---

### Phase 2: Add Formation System (4 hours)

**What to build:**
```go
// squads/formations.go
type FormationPattern interface {
    ArrangUnits(units []*ecs.Entity) [9]*ecs.Entity
    GetCombatModifiers(ctx TacticalContext) CombatModifiers
}

type DefensiveFormation struct{}  // +5 DEF
type OffensiveFormation struct{}  // +5 ATK
type BalancedFormation struct{}   // +2 ATK/DEF
```

**Deliverables:**
- 3 formations implemented
- Formation switching mid-battle
- Combat modifiers applied
- Player can change formations via UI

**Timeline:** Week 3 (4 hours)

---

### Phase 3: Add Command Pattern (5 hours)

**What to build:**
```go
// squads/commands.go
type SquadCommand interface {
    Execute(world *GameWorld) error
    Validate(world *GameWorld) error
    Preview(world *GameWorld) CommandPreview
}

type AttackSquadCommand struct { ... }
type MoveSquadCommand struct { ... }
type ChangeFormationCommand struct { ... }
```

**Deliverables:**
- Command validation before execution
- Combat preview system (hit chance, damage range)
- Cleaner input controller integration

**Timeline:** Week 4 (5 hours)

---

### Phase 4: Add Leader Abilities (8 hours)

**What to build:**
```go
// squads/abilities.go
type LeaderAbility interface {
    Execute(caster *Squad, target interface{}, world *GameWorld) error
    CurrentCooldown() int
}

type RallyAbility struct { ... }   // Heal squad
type ChargeAbility struct { ... }  // +50% damage
type TauntAbility struct { ... }   // Force targeting
```

**Deliverables:**
- 3 leader abilities implemented
- Cooldown system working
- Leader ability slots (4 customizable)
- Ability UI in combat

**Timeline:** Week 5 (8 hours)

---

## Why This Approach?

### Technical Justification

**Foundation (Approach 2):**
- Fastest path to working squad combat (critical for testing balance)
- Simplest architecture (easier to debug early issues)
- Centralized manager (easy to refactor later)

**Evolution (Approach 3):**
- Formation interface is a small addition (4h) with huge gameplay value
- Command pattern integrates cleanly (5h) with existing InputCoordinator
- Abilities are final polish (8h) for depth, can defer if needed

### Gameplay Justification

**Week 1-2:** Basic squad combat is playable
- **Player can:** Create squads, move squads, attack enemies
- **Gameplay:** Functional but basic (one formation, no abilities)
- **Value:** Core loop established, balance testing can begin

**Week 3:** Formation system adds tactical depth
- **Player can:** Switch between defensive/offensive/balanced formations
- **Gameplay:** Mid-battle decisions emerge ("low HP? go defensive")
- **Value:** Strategic layer appears

**Week 4:** Command pattern adds polish
- **Player can:** Preview combat outcomes before committing
- **Gameplay:** Informed decisions (Fire Emblem-style forecasting)
- **Value:** Reduces frustration ("I didn't know that would happen!")

**Week 5:** Leader abilities add identity
- **Player can:** Customize squad leaders, use unique abilities
- **Gameplay:** Squad personalities emerge (Rally squad vs Charge squad)
- **Value:** Replayability and build variety

---

## Risk Mitigation

### Risk: What if implementation takes longer than estimated?

**Mitigation:** Incremental delivery means partial completion is still valuable

- **Worst case (20h):** Basic squad combat works (Phase 1 only)
- **Good case (24h):** Formations working (Phase 1 + 2)
- **Great case (29h):** Command preview working (Phase 1 + 2 + 3)
- **Best case (37h):** Full Approach 3 feature set (all phases)

Every phase adds value independently.

---

### Risk: Manager becomes too large (god object)?

**Mitigation:** Split manager across files

```
squads/
  squadmanager.go    (core state management)
  combat.go          (combat logic)
  formations.go      (formation system)
  commands.go        (command pattern)
  abilities.go       (leader abilities)
  integration.go     (spawning/input integration)
```

Each file ~200-300 LOC, manageable size.

---

### Risk: Performance issues with squad count?

**Mitigation:** Architecture supports optimization

- **Phase 1:** O(n) squad iteration (acceptable for <50 squads)
- **If needed:** Add spatial partitioning to SquadManager (O(log n) lookup)
- **If needed:** Component-based caching (Approach 1 patterns)

Performance is unlikely to be an issue for turn-based tactical game.

---

## Next Steps

### 1. Decision Point

**Choose your starting approach:**

- [ ] **Option A:** Implement full Approach 1 (28-32h, pure ECS)
- [ ] **Option B:** Implement full Approach 2 (20-30h, simplicity first)
- [ ] **Option C:** Implement full Approach 3 (24-32h, maximum depth)
- [x] **Option D (RECOMMENDED):** Incremental 2→3 (20-37h, low risk)

---

### 2. If You Choose Option D (Incremental 2→3)

**Immediate next steps:**

1. **Create task breakdown for Phase 1 (Approach 2 foundation)**
   - Files to create: `squads/squad.go`, `squads/squadmanager.go`, `squads/combat.go`
   - Integration points: `entitytemplates/creators.go`, `input/combatcontroller.go`, `spawning/spawnmonsters.go`
   - Estimated: 20 hours

2. **Set up test environment**
   - Create `squads/squadmanager_test.go`
   - Mock dependencies (EntityManager, CoordinateManager)
   - Write basic tests (CreateSquad, ExecuteSquadCombat)

3. **Implement Phase 1**
   - Follow Approach 2 design patterns
   - Get basic squad combat working
   - Integrate with existing input/spawning systems

4. **Playtest Phase 1**
   - Test balance (squad-vs-squad feels right?)
   - Identify pain points (need formations? need preview?)
   - Decide if Phase 2+ worth implementing

5. **Iterate phases 2-4 as needed**
   - Add formations if tactical depth lacking
   - Add command pattern if UI needs preview
   - Add abilities if customization desired

---

### 3. Files to Create

**Phase 1 (20h):**
```
squads/
  squad.go              (Squad struct, GridSlot)
  squadmanager.go       (SquadManager, squad registry)
  combat.go             (ExecuteSquadCombat, damage distribution)
  unitdata.go           (UnitDataComponent, unit types)
  integration.go        (CreateSquadWithUnits helper)

Modified files:
  entitytemplates/creators.go  (add squad creation)
  input/combatcontroller.go    (add squad attack handling)
  spawning/spawnmonsters.go    (spawn squads instead of individuals)
```

**Phase 2 (+4h):**
```
squads/
  formations.go         (FormationPattern interface, 3 implementations)
```

**Phase 3 (+5h):**
```
squads/
  commands.go           (SquadCommand interface, Attack/Move/Formation commands)
```

**Phase 4 (+8h):**
```
squads/
  abilities.go          (LeaderAbility interface, 3 implementations)
```

---

### 4. Suggested Implementation Order

**Week 1:**
1. Create `squads/squad.go` (Squad struct, GridSlot)
2. Create `squads/squadmanager.go` (core state management)
3. Integrate with EntityManager (squad creation)
4. Test: Create squad, add units, verify grid

**Week 2:**
5. Create `squads/combat.go` (ExecuteSquadCombat)
6. Integrate with existing PerformAttack() (damage wrapper)
7. Add front-to-back damage distribution
8. Test: Squad-vs-squad combat, verify damage pooling

**Week 3:**
9. Integrate with InputCoordinator (squad attack handling)
10. Integrate with spawning system (spawn squads)
11. Manual playtest: Create squads, attack, verify behavior
12. Balance testing: Adjust damage formulas

**Week 4+ (if continuing):**
13. Add formations (Phase 2)
14. Add command pattern (Phase 3)
15. Add leader abilities (Phase 4)

---

## Final Recommendation Summary

**Best Choice: Incremental Approach 2 → 3**

**Phase 1 (20h):** Approach 2 foundation
- **Delivers:** Working squad combat
- **Risk:** Low (simplest architecture)
- **Value:** High (core gameplay functional)

**Phase 2-4 (4h + 5h + 8h):** Evolve to Approach 3
- **Delivers:** Formations, command preview, leader abilities
- **Risk:** Low (incremental additions)
- **Value:** High (tactical depth and polish)

**Total:** 20-37 hours depending on how far you go

**Why this is best:**
1. **Lowest risk:** Basic combat works after Week 1
2. **Budget-friendly:** Can stop at any phase
3. **Incremental value:** Every phase adds playable features
4. **Tactical depth:** Eventually matches Approach 3 if you complete all phases
5. **Testing-friendly:** Simple system tested first, complexity added gradually

**Comparison to alternatives:**
- **vs Approach 1:** Faster implementation (20h vs 28h), simpler initially
- **vs Approach 2 only:** More tactical depth if you continue phases 2-4
- **vs Approach 3 directly:** Lower risk (working system earlier), easier debugging

---

**Ready to proceed?** Review this comparison with your team, decide on Option A/B/C/D, and I'll create the detailed implementation plan for your chosen approach.

---

**Document Version:** 1.0
**Cross-References:**
- Approach 1 details: `analysis/APPROACH_1_ComponentRich.md`
- Approach 2 details: `analysis/APPROACH_2_ManagerCentric.md`
- Approach 3 details: `analysis/APPROACH_3_HybridCommand.md`
- Refactoring priorities: `analysis/REFACTORING_PRIORITIES.md`
- Combat refactoring analysis: `analysis/combat_refactoring.md`
