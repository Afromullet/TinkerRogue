# Approach 1: Component-Rich Entity Architecture

**Last Updated:** 2025-10-01
**Status:** Architectural Proposal
**Complexity:** Medium-High (28-32 hours)

---

## Philosophy

**Core Principle:** Squads are emergent behavior from component composition, not monolithic structures.

This approach follows traditional ECS patterns where behavior emerges from component combinations. A squad is a separate entity that "owns" unit entities through relationship components. The 3x3 grid positions are component data, not external data structures.

**TRPG Inspirations:**
- **Final Fantasy Tactics**: Leader customization with job abilities
- **Soul Nomad**: Multi-unit squads with internal formations
- **Symphony of War**: Squad-level positioning with unit composition

**ECS Alignment:** Maximum adherence to component-based architecture. Systems operate on components, not complex nested structures.

---

## Data Structure Design

### Core Components

```go
// squad/components.go

package squad

import "github.com/yohamta/donburi"

// SquadComponent lives on the squad entity itself
type SquadComponent struct {
    UnitEntities   []*donburi.Entry  // References to unit entities (max 9)
    Formation      FormationType      // Current formation layout
    Name           string             // Squad display name
    Morale         int                // Squad-wide morale (future)
    SquadLevel     int                // Average level for spawning
}

var SquadComponentType = donburi.NewComponentType[SquadComponent]()

// SquadMemberComponent lives on each unit entity
// Links unit back to its parent squad
type SquadMemberComponent struct {
    SquadEntity    *donburi.Entry    // Parent squad entity
    GridPosition   GridPosition       // Position within 3x3 grid
    IsLeader       bool               // True for commander unit
    UnitRole       UnitRole           // FRONT_LINE, SUPPORT, RANGED
}

var SquadMemberComponentType = donburi.NewComponentType[SquadMemberComponent]()

// GridPosition represents position in 3x3 internal grid
type GridPosition struct {
    Row    int  // 0-2 (top to bottom)
    Col    int  // 0-2 (left to right)
}

// UnitRole affects combat behavior
type UnitRole int
const (
    FRONT_LINE UnitRole = iota  // Takes hits first, melee focused
    SUPPORT                      // Buffs, heals, mid-line
    RANGED                       // Back line, projectile attacks
)

// FormationType defines squad layout
type FormationType int
const (
    FORMATION_BALANCED FormationType = iota  // Mix of roles
    FORMATION_DEFENSIVE                       // Front-heavy
    FORMATION_OFFENSIVE                       // Damage-focused
    FORMATION_RANGED                          // Back-line heavy
)

// LeaderComponent marks unit as customizable leader
type LeaderComponent struct {
    AbilitySlots   [4]*AbilityInstance  // Equipped abilities (FFT-style)
    Leadership     int                   // Bonus to squad stats
    Experience     int                   // Leader progression
}

var LeaderComponentType = donburi.NewComponentType[LeaderComponent]()

// AbilityInstance represents an equipped ability
type AbilityInstance struct {
    AbilityID      string                // "FireballLv2", "Heal", "Rally"
    Cooldown       int                   // Turns remaining
    MaxCooldown    int                   // Base cooldown
    TargetType     AbilityTargetType     // SELF_SQUAD, ENEMY_SQUAD, POSITION
}

type AbilityTargetType int
const (
    TARGET_SELF_SQUAD AbilityTargetType = iota
    TARGET_ENEMY_SQUAD
    TARGET_POSITION   // AOE at map coordinate
    TARGET_ALLY_SQUAD
)

// RegularUnitComponent marks unit as template-based
type RegularUnitComponent struct {
    TemplateID     string  // "HumanWarrior", "ElfArcher"
    IsFixed        bool    // Cannot be customized (always true for regulars)
}

var RegularUnitComponentType = donburi.NewComponentType[RegularUnitComponent]()

// UnitSizeComponent defines grid footprint
type UnitSizeComponent struct {
    Width   int  // 1 or 2
    Height  int  // 1 or 2
}

var UnitSizeComponentType = donburi.NewComponentType[UnitSizeComponent]()
```

### Unit Size Constraints

```go
// Valid unit sizes in 3x3 grid
const (
    SIZE_1x1 = iota  // Single cell (most common)
    SIZE_2x1         // Horizontal (cavalry, war machines)
    SIZE_1x2         // Vertical (banner carriers, tall units)
)

// Grid occupancy rules
type GridLayout struct {
    Occupied [3][3]bool           // Which cells are filled
    UnitMap  [3][3]*donburi.Entry // Entity at each position
}

// Example: 2x1 unit at (1,0) occupies (1,0) and (1,1)
// Example: 1x2 unit at (0,1) occupies (0,1) and (1,1)
```

---

## Combat Flow: Simultaneous Pooled Damage

### High-Level Sequence

```
1. Player selects squad → targets enemy squad
2. AttackingSystem collects all units in attacker squad
3. For each attacking unit:
   - Calculate individual damage (existing PerformAttack logic)
   - Apply role modifiers (FRONT_LINE gets melee bonus)
   - Add to damage pool
4. Distribute pooled damage to defender squad:
   - FRONT_LINE units absorb damage first
   - Overflow damage to SUPPORT/RANGED
   - Apply unit deaths and remove from grid
5. Counter-attack (if defender still alive):
   - Same pooled damage logic in reverse
6. Update squad states (remove dead units, check morale)
```

### Damage Distribution Algorithm

```go
// combat/squaddamage.go

package combat

import (
    "tinkerrogue/ecs"
    "tinkerrogue/squad"
    "sort"
)

// PooledDamageResult holds combat outcome
type PooledDamageResult struct {
    TotalDamage    int
    UnitsKilled    []*donburi.Entry
    DamageByUnit   map[*donburi.Entry]int
    Overkill       int  // Unused damage
}

// ExecuteSquadAttack performs simultaneous squad combat
func ExecuteSquadAttack(attackerSquad, defenderSquad *donburi.Entry, world donburi.World) *PooledDamageResult {
    result := &PooledDamageResult{
        DamageByUnit: make(map[*donburi.Entry]int),
        UnitsKilled:  []*donburi.Entry{},
    }

    // Step 1: Collect damage pool from all attackers
    attackerSquadData := squad.SquadComponentType.Get(attackerSquad)
    damagePool := 0

    for _, unitEntity := range attackerSquadData.UnitEntities {
        if unitEntity == nil {
            continue // Dead/empty slot
        }

        // Use existing PerformAttack logic per unit
        damage := calculateUnitDamage(unitEntity, world)

        // Apply role modifiers
        member := squad.SquadMemberComponentType.Get(unitEntity)
        damage = applyRoleModifier(damage, member.UnitRole)

        damagePool += damage
    }

    result.TotalDamage = damagePool

    // Step 2: Distribute damage to defenders
    defenderSquadData := squad.SquadComponentType.Get(defenderSquad)
    remainingDamage := damagePool

    // Priority order: FRONT_LINE → SUPPORT → RANGED
    defenders := sortDefendersByRole(defenderSquadData.UnitEntities)

    for _, defenderUnit := range defenders {
        if remainingDamage <= 0 {
            break
        }

        hp := ecs.HitpointsComponent.Get(defenderUnit)
        damageToUnit := min(hp.Current, remainingDamage)

        hp.Current -= damageToUnit
        result.DamageByUnit[defenderUnit] = damageToUnit
        remainingDamage -= damageToUnit

        if hp.Current <= 0 {
            result.UnitsKilled = append(result.UnitsKilled, defenderUnit)
            removeUnitFromSquad(defenderUnit, defenderSquad, world)
        }
    }

    result.Overkill = remainingDamage

    return result
}

// Helper: Calculate individual unit damage
func calculateUnitDamage(unitEntity *donburi.Entry, world donburi.World) int {
    // Reuse existing PerformAttack logic
    stats := ecs.StatsComponent.Get(unitEntity)
    gear := ecs.EquippedGearComponent.Get(unitEntity)

    baseDamage := stats.Strength
    if gear.Weapon != nil {
        baseDamage += gear.Weapon.Damage
    }

    // d20 variance
    roll := rollD20()
    if roll >= 18 {
        baseDamage = int(float64(baseDamage) * 1.5) // Critical
    } else if roll <= 3 {
        baseDamage = baseDamage / 2 // Weak hit
    }

    return baseDamage
}

// Helper: Role-based damage modifiers
func applyRoleModifier(damage int, role squad.UnitRole) int {
    switch role {
    case squad.FRONT_LINE:
        return int(float64(damage) * 1.2)  // +20% melee
    case squad.RANGED:
        return int(float64(damage) * 1.1)  // +10% ranged
    case squad.SUPPORT:
        return int(float64(damage) * 0.8)  // -20% (support isn't damage)
    default:
        return damage
    }
}

// Helper: Sort defenders by role priority
func sortDefendersByRole(units []*donburi.Entry) []*donburi.Entry {
    alive := []*donburi.Entry{}
    for _, u := range units {
        if u != nil {
            hp := ecs.HitpointsComponent.Get(u)
            if hp.Current > 0 {
                alive = append(alive, u)
            }
        }
    }

    sort.Slice(alive, func(i, j int) bool {
        roleI := squad.SquadMemberComponentType.Get(alive[i]).UnitRole
        roleJ := squad.SquadMemberComponentType.Get(alive[j]).UnitRole
        return roleI < roleJ  // FRONT_LINE (0) comes before RANGED (2)
    })

    return alive
}

// Helper: Remove dead unit from squad
func removeUnitFromSquad(unitEntity, squadEntity *donburi.Entry, world donburi.World) {
    squadData := squad.SquadComponentType.Get(squadEntity)
    member := squad.SquadMemberComponentType.Get(unitEntity)

    // Clear grid position
    gridPos := member.GridPosition
    // (Grid occupancy would be tracked separately)

    // Remove from squad's unit list
    for i, u := range squadData.UnitEntities {
        if u == unitEntity {
            squadData.UnitEntities[i] = nil
            break
        }
    }

    // Destroy entity
    unitEntity.Remove()
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func rollD20() int {
    return rand.Intn(20) + 1
}
```

---

## 3x3 Grid Mechanics

### Grid Position System

```go
// squad/gridmanager.go

package squad

import "github.com/yohamta/donburi"

// SquadGrid manages 3x3 internal grid
type SquadGrid struct {
    Occupied [3][3]bool
    UnitMap  [3][3]*donburi.Entry
}

// PlaceUnit adds unit to grid, respecting size
func (g *SquadGrid) PlaceUnit(unit *donburi.Entry, pos GridPosition, size UnitSizeComponent) bool {
    // Check if space available
    for r := 0; r < size.Height; r++ {
        for c := 0; c < size.Width; c++ {
            checkRow := pos.Row + r
            checkCol := pos.Col + c

            if checkRow >= 3 || checkCol >= 3 {
                return false // Out of bounds
            }

            if g.Occupied[checkRow][checkCol] {
                return false // Already occupied
            }
        }
    }

    // Place unit
    for r := 0; r < size.Height; r++ {
        for c := 0; c < size.Width; c++ {
            g.Occupied[pos.Row+r][pos.Col+c] = true
            g.UnitMap[pos.Row+r][pos.Col+c] = unit
        }
    }

    return true
}

// GetUnitsInRow returns all units in a given row (0-2)
func (g *SquadGrid) GetUnitsInRow(row int) []*donburi.Entry {
    units := make(map[*donburi.Entry]bool)

    for col := 0; col < 3; col++ {
        if g.UnitMap[row][col] != nil {
            units[g.UnitMap[row][col]] = true
        }
    }

    result := []*donburi.Entry{}
    for u := range units {
        result = append(result, u)
    }
    return result
}

// RemoveUnit clears unit from all grid positions
func (g *SquadGrid) RemoveUnit(unit *donburi.Entry) {
    for r := 0; r < 3; r++ {
        for c := 0; c < 3; c++ {
            if g.UnitMap[r][c] == unit {
                g.Occupied[r][c] = false
                g.UnitMap[r][c] = nil
            }
        }
    }
}
```

### Position Effects on Combat

```go
// Combat bonuses based on grid position
func getPositionModifier(pos GridPosition) float64 {
    // Front row (row 0): +10% damage taken, +5% damage dealt
    // Middle row (row 1): Neutral
    // Back row (row 2): -20% damage taken, -10% damage dealt

    switch pos.Row {
    case 0:
        return 1.05  // Front line damage boost
    case 1:
        return 1.0   // Neutral
    case 2:
        return 0.9   // Back line damage penalty
    default:
        return 1.0
    }
}

// Damage absorption based on position
func getDamageReduction(pos GridPosition) float64 {
    switch pos.Row {
    case 0:
        return 0.9   // Front row takes more damage
    case 1:
        return 1.0
    case 2:
        return 1.2   // Back row takes less damage (harder to reach)
    default:
        return 1.0
    }
}
```

---

## Leader Ability System

### Ability Definition

```go
// squad/abilities.go

package squad

import "github.com/yohamta/donburi"

// AbilityDefinition is the template for an ability
type AbilityDefinition struct {
    ID             string
    Name           string
    Description    string
    Cooldown       int
    TargetType     AbilityTargetType
    Effect         AbilityEffect
}

// AbilityEffect is executed when ability is used
type AbilityEffect interface {
    Execute(caster, target *donburi.Entry, world donburi.World) error
}

// Example: Rally ability (buff own squad)
type RallyEffect struct {
    DamageBonus    int
    Duration       int  // Turns
}

func (e *RallyEffect) Execute(caster, target *donburi.Entry, world donburi.World) error {
    // Get caster's squad
    member := SquadMemberComponentType.Get(caster)
    squadEntity := member.SquadEntity
    squadData := SquadComponentType.Get(squadEntity)

    // Apply buff to all units in squad
    for _, unit := range squadData.UnitEntities {
        if unit == nil {
            continue
        }

        // Add damage buff (would need BuffComponent)
        stats := ecs.StatsComponent.Get(unit)
        stats.Strength += e.DamageBonus

        // Schedule buff removal after duration
        // (Requires turn tracking system)
    }

    return nil
}

// Example: Fireball ability (AOE damage to enemy squad)
type FireballEffect struct {
    BaseDamage     int
    AOERadius      int  // Affects multiple grid positions
}

func (e *FireballEffect) Execute(caster, targetSquad *donburi.Entry, world donburi.World) error {
    squadData := SquadComponentType.Get(targetSquad)

    // Deal damage to all units in target squad
    for _, unit := range squadData.UnitEntities {
        if unit == nil {
            continue
        }

        hp := ecs.HitpointsComponent.Get(unit)
        hp.Current -= e.BaseDamage

        if hp.Current <= 0 {
            removeUnitFromSquad(unit, targetSquad, world)
        }
    }

    return nil
}

// Ability Registry (global or in squad system)
var AbilityRegistry = map[string]*AbilityDefinition{
    "Rally": {
        ID:         "Rally",
        Name:       "Rally",
        Description: "Boost squad damage by 3 for 3 turns",
        Cooldown:   5,
        TargetType: TARGET_SELF_SQUAD,
        Effect:     &RallyEffect{DamageBonus: 3, Duration: 3},
    },
    "Fireball": {
        ID:         "Fireball",
        Name:       "Fireball",
        Description: "Deal 15 damage to all units in target squad",
        Cooldown:   3,
        TargetType: TARGET_ENEMY_SQUAD,
        Effect:     &FireballEffect{BaseDamage: 15},
    },
    "Heal": {
        ID:         "Heal",
        Name:       "Heal Squad",
        Description: "Restore 10 HP to all units in own squad",
        Cooldown:   4,
        TargetType: TARGET_SELF_SQUAD,
        Effect:     &HealEffect{HealAmount: 10},
    },
}
```

### Leader Ability Management

```go
// EquipAbility adds ability to leader's slot
func EquipAbility(leaderEntity *donburi.Entry, abilityID string, slot int) error {
    leader := LeaderComponentType.Get(leaderEntity)

    if slot < 0 || slot >= 4 {
        return fmt.Errorf("invalid slot %d", slot)
    }

    abilityDef, exists := AbilityRegistry[abilityID]
    if !exists {
        return fmt.Errorf("unknown ability: %s", abilityID)
    }

    leader.AbilitySlots[slot] = &AbilityInstance{
        AbilityID:   abilityID,
        Cooldown:    0,
        MaxCooldown: abilityDef.Cooldown,
        TargetType:  abilityDef.TargetType,
    }

    return nil
}

// UseAbility executes leader ability
func UseAbility(leaderEntity *donburi.Entry, slot int, target *donburi.Entry, world donburi.World) error {
    leader := LeaderComponentType.Get(leaderEntity)

    ability := leader.AbilitySlots[slot]
    if ability == nil {
        return fmt.Errorf("no ability in slot %d", slot)
    }

    if ability.Cooldown > 0 {
        return fmt.Errorf("ability on cooldown: %d turns", ability.Cooldown)
    }

    // Execute effect
    abilityDef := AbilityRegistry[ability.AbilityID]
    err := abilityDef.Effect.Execute(leaderEntity, target, world)
    if err != nil {
        return err
    }

    // Set cooldown
    ability.Cooldown = ability.MaxCooldown

    return nil
}

// TickCooldowns reduces all ability cooldowns by 1 (call each turn)
func TickCooldowns(leaderEntity *donburi.Entry) {
    leader := LeaderComponentType.Get(leaderEntity)

    for _, ability := range leader.AbilitySlots {
        if ability != nil && ability.Cooldown > 0 {
            ability.Cooldown--
        }
    }
}
```

---

## Integration Points

### 1. Entity Creation (entitytemplates/creators.go)

```go
// CreateSquadFromTemplate creates squad entity + unit entities
func CreateSquadFromTemplate(
    world donburi.World,
    squadName string,
    formation squad.FormationType,
    unitTemplates []UnitTemplate,
) *donburi.Entry {

    // Create squad entity
    squadEntity := world.Create(
        squad.SquadComponentType,
        coords.LogicalPositionComponentType,
        ecs.RenderComponent,
    )

    squadData := squad.SquadComponentType.Get(squadEntity)
    squadData.Name = squadName
    squadData.Formation = formation
    squadData.UnitEntities = make([]*donburi.Entry, 9)

    // Create unit entities
    grid := &squad.SquadGrid{}

    for i, template := range unitTemplates {
        // Create unit entity
        unitEntity := CreateEntityFromTemplate(
            world,
            template.EntityType,
            template.Config,
        )

        // Add squad membership
        unitEntity.AddComponent(squad.SquadMemberComponentType)
        member := squad.SquadMemberComponentType.Get(unitEntity)
        member.SquadEntity = squadEntity
        member.GridPosition = template.GridPosition
        member.UnitRole = template.Role
        member.IsLeader = template.IsLeader

        // Add size component
        unitEntity.AddComponent(squad.UnitSizeComponentType)
        size := squad.UnitSizeComponentType.Get(unitEntity)
        size.Width = template.Size.Width
        size.Height = template.Size.Height

        // Add leader or regular component
        if template.IsLeader {
            unitEntity.AddComponent(squad.LeaderComponentType)
            leader := squad.LeaderComponentType.Get(unitEntity)
            leader.Leadership = 10
            leader.AbilitySlots = [4]*squad.AbilityInstance{}
        } else {
            unitEntity.AddComponent(squad.RegularUnitComponentType)
            regular := squad.RegularUnitComponentType.Get(unitEntity)
            regular.TemplateID = template.TemplateID
            regular.IsFixed = true
        }

        // Place in grid
        grid.PlaceUnit(unitEntity, template.GridPosition, *size)

        squadData.UnitEntities[i] = unitEntity
    }

    return squadEntity
}

type UnitTemplate struct {
    EntityType   EntityType
    Config       EntityConfig
    GridPosition squad.GridPosition
    Role         squad.UnitRole
    IsLeader     bool
    TemplateID   string
    Size         squad.UnitSizeComponent
}
```

### 2. Input System (input/combatcontroller.go)

```go
// Modified to handle squad selection and targeting
func (c *CombatController) HandleClick(worldPos coords.LogicalPosition) {
    clickedEntity := c.getEntityAtPosition(worldPos)

    if clickedEntity == nil {
        return
    }

    // Check if clicked entity is a squad
    if donburi.HasComponent[squad.SquadComponent](clickedEntity) {
        if c.selectedSquad == nil {
            // Select attacker squad
            c.selectedSquad = clickedEntity
            c.highlightSquad(clickedEntity)
        } else {
            // Target defender squad
            c.executeSquadCombat(c.selectedSquad, clickedEntity)
            c.selectedSquad = nil
        }
        return
    }

    // Legacy: individual entity combat
    c.handleIndividualCombat(clickedEntity)
}

func (c *CombatController) executeSquadCombat(attacker, defender *donburi.Entry) {
    result := combat.ExecuteSquadAttack(attacker, defender, c.world)

    // Display combat results
    c.showCombatResults(result)

    // Counter-attack if defender still alive
    if c.isSquadAlive(defender) {
        counterResult := combat.ExecuteSquadAttack(defender, attacker, c.world)
        c.showCombatResults(counterResult)
    }

    // Check for squad destruction
    c.checkSquadDestruction(attacker)
    c.checkSquadDestruction(defender)
}

func (c *CombatController) isSquadAlive(squadEntity *donburi.Entry) bool {
    squadData := squad.SquadComponentType.Get(squadEntity)

    for _, unit := range squadData.UnitEntities {
        if unit != nil {
            hp := ecs.HitpointsComponent.Get(unit)
            if hp.Current > 0 {
                return true
            }
        }
    }

    return false
}

func (c *CombatController) checkSquadDestruction(squadEntity *donburi.Entry) {
    if !c.isSquadAlive(squadEntity) {
        // Remove squad entity
        squadEntity.Remove()
    }
}
```

### 3. Spawning System (spawning/spawnmonsters.go)

```go
// Spawn enemy squads instead of individual monsters
func SpawnEnemySquad(world donburi.World, level int) *donburi.Entry {
    // Determine squad composition based on level
    templates := []entitytemplates.UnitTemplate{}

    if level <= 3 {
        // Early game: 3-5 weak units
        templates = append(templates, entitytemplates.UnitTemplate{
            EntityType:   entitytemplates.ENTITY_MONSTER,
            Config:       entitytemplates.EntityConfig{TemplateName: "Goblin"},
            GridPosition: squad.GridPosition{Row: 0, Col: 0},
            Role:         squad.FRONT_LINE,
            Size:         squad.UnitSizeComponent{Width: 1, Height: 1},
        })
        // ... add more units
    } else {
        // Late game: Larger units, better composition
        templates = append(templates, entitytemplates.UnitTemplate{
            EntityType:   entitytemplates.ENTITY_MONSTER,
            Config:       entitytemplates.EntityConfig{TemplateName: "OrcChampion"},
            GridPosition: squad.GridPosition{Row: 0, Col: 0},
            Role:         squad.FRONT_LINE,
            Size:         squad.UnitSizeComponent{Width: 2, Height: 1},
        })
    }

    squadEntity := entitytemplates.CreateSquadFromTemplate(
        world,
        "Enemy Squad",
        squad.FORMATION_BALANCED,
        templates,
    )

    return squadEntity
}
```

### 4. Rendering (graphics/)

```go
// Render squad as single entity on tactical map
// Internal 3x3 grid rendered when squad is selected/inspected

func RenderSquad(screen *ebiten.Image, squadEntity *donburi.Entry) {
    pos := coords.LogicalPositionComponentType.Get(squadEntity)
    pixelPos := coordManager.LogicalToPixel(pos.Position)

    // Draw squad sprite at tactical position
    squadSprite := getSquadSprite(squadEntity)
    opts := &ebiten.DrawImageOptions{}
    opts.GeoM.Translate(float64(pixelPos.X), float64(pixelPos.Y))
    screen.DrawImage(squadSprite, opts)

    // If selected, show 3x3 grid overlay
    if isSelected(squadEntity) {
        renderSquadGrid(screen, squadEntity, pixelPos)
    }
}

func renderSquadGrid(screen *ebiten.Image, squadEntity *donburi.Entry, basePos coords.PixelPosition) {
    squadData := squad.SquadComponentType.Get(squadEntity)
    cellSize := 16  // Pixels per grid cell

    for _, unitEntity := range squadData.UnitEntities {
        if unitEntity == nil {
            continue
        }

        member := squad.SquadMemberComponentType.Get(unitEntity)
        gridPos := member.GridPosition

        // Calculate pixel offset within squad
        offsetX := gridPos.Col * cellSize
        offsetY := gridPos.Row * cellSize

        // Draw unit sprite in grid
        unitSprite := getUnitSprite(unitEntity)
        opts := &ebiten.DrawImageOptions{}
        opts.GeoM.Translate(
            float64(basePos.X + offsetX),
            float64(basePos.Y + offsetY),
        )
        screen.DrawImage(unitSprite, opts)
    }
}
```

---

## Pros & Cons

### Advantages

**1. Maximum ECS Alignment**
- Pure component-based design
- Systems operate on components, not complex structures
- Easy to extend with new components

**2. Flexible Querying**
- Can query all leaders: `query.NewQuery(filter.Contains(squad.LeaderComponentType))`
- Can find all front-line units easily
- Component composition enables emergent behavior

**3. Clear Ownership**
- Each unit entity knows its parent squad
- Squad entity knows its child units
- Bidirectional relationship is explicit

**4. Incremental Migration**
- Individual entities can coexist with squads
- Existing PerformAttack() reused per-unit
- Can phase in squad combat gradually

**5. Ability System Modularity**
- Abilities are data-driven (AbilityDefinition registry)
- Easy to add new abilities without code changes
- Leader progression is component-based

### Disadvantages

**1. Component Overhead**
- Many small components (SquadMember, GridPosition, UnitSize, etc.)
- More memory allocations
- Component lookups throughout code

**2. Synchronization Complexity**
- Must keep SquadComponent.UnitEntities in sync with actual entities
- Removing units requires updating multiple data structures
- Grid state must match component state

**3. Circular References**
- Squad → Units (UnitEntities slice)
- Units → Squad (SquadMemberComponent.SquadEntity)
- Must carefully handle entity removal to avoid dangling pointers

**4. No Built-In Grid Validation**
- Grid occupancy must be manually tracked
- Placing units requires separate validation logic
- Position changes need grid updates

**5. More Boilerplate**
- Need to register many component types
- Entity creation is verbose
- Requires more helper functions

---

## Complexity Estimate

### Implementation Time: 28-32 hours

**Phase 1: Components & Data Structures (6-8 hours)**
- Create squad/components.go with all component types
- Implement GridPosition and UnitSize logic
- Register components in ECS

**Phase 2: Combat System (8-10 hours)**
- Implement ExecuteSquadAttack with pooled damage
- Create damage distribution algorithm
- Add role-based modifiers
- Test combat scenarios

**Phase 3: Leader Abilities (6-8 hours)**
- Build AbilityDefinition registry
- Implement ability effects (Rally, Fireball, Heal)
- Add cooldown system
- Create ability UI hooks

**Phase 4: Integration (8-10 hours)**
- Modify InputCoordinator for squad selection
- Update spawning system for squad creation
- Add squad rendering with grid overlay
- Create CreateSquadFromTemplate factory

**Testing & Polish (2-4 hours)**
- Manual combat testing
- Balance verification
- Edge case handling (empty squads, leader death)

---

## Maintainability

### High Maintainability Score: 8/10

**Strengths:**
- Pure ECS design is familiar to team
- Component queries are straightforward
- Ability system is data-driven and extensible
- Clear separation of concerns (combat, grid, abilities)

**Challenges:**
- Many components to manage (documentation crucial)
- Synchronization between squad and units requires discipline
- Grid validation logic must be centralized

**Recommendation:** Excellent for teams comfortable with ECS patterns and willing to invest in upfront component design.

---

## Performance

### Expected Performance: Good

**Memory:**
- Component overhead: ~200-300 bytes per unit
- 9 units per squad = ~2.7KB per squad
- 100 squads on map = ~270KB (negligible)

**CPU:**
- Component lookups: O(1) with Donburi
- Damage distribution: O(n) where n = units in squad (max 9)
- Query performance: Excellent with ECS iteration

**Bottlenecks:**
- None expected for typical squad counts (< 50 active squads)
- Grid validation is O(size) but size is tiny (3x3)

---

## Code Examples

### Example 1: Creating a Player Squad

```go
func CreatePlayerSquad(world donburi.World) *donburi.Entry {
    templates := []entitytemplates.UnitTemplate{
        // Leader (front line, 1x1)
        {
            EntityType: entitytemplates.ENTITY_PLAYER,
            Config: entitytemplates.EntityConfig{
                TemplateName: "HumanWarrior",
                Stats:        &ecs.Stats{Strength: 15, Defense: 12},
            },
            GridPosition: squad.GridPosition{Row: 0, Col: 1},
            Role:         squad.FRONT_LINE,
            IsLeader:     true,
            Size:         squad.UnitSizeComponent{Width: 1, Height: 1},
        },
        // Archer (back line, 1x1)
        {
            EntityType:   entitytemplates.ENTITY_PLAYER,
            Config:       entitytemplates.EntityConfig{TemplateName: "ElfArcher"},
            GridPosition: squad.GridPosition{Row: 2, Col: 0},
            Role:         squad.RANGED,
            IsLeader:     false,
            TemplateID:   "ElfArcher",
            Size:         squad.UnitSizeComponent{Width: 1, Height: 1},
        },
        // Support (middle, 1x1)
        {
            EntityType:   entitytemplates.ENTITY_PLAYER,
            Config:       entitytemplates.EntityConfig{TemplateName: "Cleric"},
            GridPosition: squad.GridPosition{Row: 1, Col: 2},
            Role:         squad.SUPPORT,
            IsLeader:     false,
            TemplateID:   "Cleric",
            Size:         squad.UnitSizeComponent{Width: 1, Height: 1},
        },
        // Cavalry (front, 2x1)
        {
            EntityType:   entitytemplates.ENTITY_PLAYER,
            Config:       entitytemplates.EntityConfig{TemplateName: "Knight"},
            GridPosition: squad.GridPosition{Row: 0, Col: 0},
            Role:         squad.FRONT_LINE,
            IsLeader:     false,
            TemplateID:   "Knight",
            Size:         squad.UnitSizeComponent{Width: 2, Height: 1},
        },
    }

    squadEntity := entitytemplates.CreateSquadFromTemplate(
        world,
        "Player Squad",
        squad.FORMATION_BALANCED,
        templates,
    )

    // Equip leader with abilities
    squadData := squad.SquadComponentType.Get(squadEntity)
    leaderEntity := findLeader(squadData.UnitEntities)

    squad.EquipAbility(leaderEntity, "Rally", 0)
    squad.EquipAbility(leaderEntity, "Heal", 1)

    return squadEntity
}

func findLeader(units []*donburi.Entry) *donburi.Entry {
    for _, u := range units {
        if u != nil && donburi.HasComponent[squad.LeaderComponent](u) {
            return u
        }
    }
    return nil
}
```

### Example 2: Executing Squad Combat

```go
func TestSquadCombat() {
    world := ecs.CreateWorld()

    // Create two squads
    playerSquad := CreatePlayerSquad(world)
    enemySquad := spawning.SpawnEnemySquad(world, 5)

    // Execute attack
    result := combat.ExecuteSquadAttack(playerSquad, enemySquad, world)

    fmt.Printf("Total Damage: %d\n", result.TotalDamage)
    fmt.Printf("Units Killed: %d\n", len(result.UnitsKilled))
    for unit, dmg := range result.DamageByUnit {
        fmt.Printf("  Unit dealt %d damage\n", dmg)
    }

    // Counter-attack
    if isSquadAlive(enemySquad) {
        counterResult := combat.ExecuteSquadAttack(enemySquad, playerSquad, world)
        fmt.Printf("Counter Damage: %d\n", counterResult.TotalDamage)
    }
}
```

### Example 3: Leader Ability Swap

```go
func SwapLeaderAbility(leaderEntity *donburi.Entry, oldSlot int, newAbilityID string) {
    leader := squad.LeaderComponentType.Get(leaderEntity)

    // Remove old ability
    oldAbility := leader.AbilitySlots[oldSlot]
    if oldAbility != nil {
        fmt.Printf("Removing %s\n", oldAbility.AbilityID)
    }

    // Equip new ability
    err := squad.EquipAbility(leaderEntity, newAbilityID, oldSlot)
    if err != nil {
        fmt.Printf("Error equipping ability: %v\n", err)
        return
    }

    fmt.Printf("Equipped %s in slot %d\n", newAbilityID, oldSlot)
}

// Usage in game
func TestAbilitySwap() {
    world := ecs.CreateWorld()
    playerSquad := CreatePlayerSquad(world)

    squadData := squad.SquadComponentType.Get(playerSquad)
    leaderEntity := findLeader(squadData.UnitEntities)

    // Swap slot 1 from "Heal" to "Fireball"
    SwapLeaderAbility(leaderEntity, 1, "Fireball")

    // Use new ability
    enemySquad := spawning.SpawnEnemySquad(world, 3)
    err := squad.UseAbility(leaderEntity, 1, enemySquad, world)
    if err != nil {
        fmt.Printf("Ability failed: %v\n", err)
    }
}
```

---

## Next Steps

1. **Approval Decision:** Review this approach vs other architectural options
2. **Prototype:** Implement Phase 1 (components) and test ECS integration
3. **Combat Testing:** Build Phase 2 and verify pooled damage distribution
4. **Balance Iteration:** Adjust role modifiers and damage formulas based on playtesting

**Recommendation:** This approach is ideal if the team values ECS purity and long-term extensibility over short-term implementation speed. The component overhead is negligible, and the ability to query/extend via components is powerful.

---

**File Location:** `C:\Users\Afromullet\Desktop\TinkerRogue\analysis\APPROACH_1_ComponentRich.md`
