# Approach 3: Hybrid Command-Pattern Architecture

**Last Updated:** 2025-10-01
**Status:** Architectural Proposal
**Complexity:** High (24-32 hours implementation)

---

## Architecture Philosophy

### Core Concept: Strategic Objects

This approach treats **formations, abilities, and actions as first-class objects**. Squad is both an ECS entity (for world positioning) AND a data structure (for internal composition). Combat uses command pattern with strategy-based formations.

### TRPG Inspirations

**Fire Emblem**: Command-based actions, formation positioning, unit composition strategies
**Final Fantasy Tactics**: Job/ability systems with execute/validate pattern, tactical formations
**Soul Nomad**: Squad-as-unit with internal composition, formation switching

### Philosophy Trade-offs

**Strengths:**
- Extreme extensibility (new formations/abilities are simple classes)
- Tactical depth through formation strategies
- Clean separation of concerns
- Easy to serialize/save formations and abilities
- Testable in isolation

**Weaknesses:**
- More interfaces and abstractions
- Higher initial complexity
- More files to maintain
- Potential over-engineering for simple use cases

---

## Data Structure Design

### Core Squad Structure (Hybrid Entity + Data)

```go
// Package: combat/squad.go

// Squad is BOTH an ECS entity (world position) AND a data structure (internal composition)
type Squad struct {
    // ECS Integration
    EntityID      ecs.EntityID  // The squad's world entity
    LogicalPos    coords.LogicalPosition

    // Internal Composition (3x3 grid)
    Formation     FormationPattern  // Strategy pattern
    Units         [9]*ecs.Entity    // 3x3 grid, nil = empty slot
    Leader        *ecs.Entity       // Points to one unit in Units array
    LeaderPos     GridPosition      // Leader's position within 3x3

    // Squad State
    TotalHP       int               // Pooled health
    Morale        int               // Squad cohesion (future)
    ActionPoints  int               // For turn-based (future)
}

// GridPosition within 3x3 internal grid
type GridPosition struct {
    Row int // 0-2
    Col int // 0-2
}

func (gp GridPosition) ToIndex() int {
    return gp.Row*3 + gp.Col
}

func IndexToGridPos(idx int) GridPosition {
    return GridPosition{Row: idx / 3, Col: idx % 3}
}
```

### Formation Pattern (Strategy Pattern)

```go
// Package: combat/formations.go

// FormationPattern is a pluggable strategy for squad arrangement
type FormationPattern interface {
    // Name of formation for UI display
    Name() string

    // Arrange units within 3x3 grid (returns valid positions)
    ArrangUnits(units []*ecs.Entity) [9]*ecs.Entity

    // Calculate combat modifiers based on formation
    GetCombatModifiers(context TacticalContext) CombatModifiers

    // Validate if formation is legal with given units
    Validate(units []*ecs.Entity) error

    // Visual representation for UI (future)
    GetFormationGrid() [9]bool  // true = occupied
}

// CombatModifiers from formation strategy
type CombatModifiers struct {
    AttackBonus   int     // Damage modifier
    DefenseBonus  int     // Defense modifier
    AccuracyBonus int     // Hit chance modifier
    InitiativeBonus int   // Turn order modifier (future)
}

// TacticalContext provides situational awareness
type TacticalContext struct {
    AttackingSquad *Squad
    DefendingSquad *Squad
    Terrain        TerrainType
    Flanking       bool
    ElevationAdv   bool
}

// --- CONCRETE FORMATIONS ---

// DefensiveFormation: Leader protected, high defense
type DefensiveFormation struct{}

func (f *DefensiveFormation) Name() string { return "Defensive" }

func (f *DefensiveFormation) ArrangUnits(units []*ecs.Entity) [9]*ecs.Entity {
    // Arrangement pattern:
    // [0][1][2]    [R][R][R]
    // [3][4][5] -> [R][L][R]  (L = Leader in center)
    // [6][7][8]    [R][R][R]

    var grid [9]*ecs.Entity
    if len(units) == 0 {
        return grid
    }

    // Find leader (assume first unit with LeaderComponent)
    leaderIdx := 0
    for i, u := range units {
        if u != nil && u.GetComponent("LeaderComponent") != nil {
            leaderIdx = i
            break
        }
    }

    // Place leader in center (index 4)
    grid[4] = units[leaderIdx]

    // Place remaining units around leader
    positions := []int{1, 3, 5, 7, 0, 2, 6, 8}  // Priority order
    unitIdx := 0
    for _, pos := range positions {
        if unitIdx >= len(units) {
            break
        }
        if unitIdx == leaderIdx {
            unitIdx++
            if unitIdx >= len(units) {
                break
            }
        }
        grid[pos] = units[unitIdx]
        unitIdx++
    }

    return grid
}

func (f *DefensiveFormation) GetCombatModifiers(ctx TacticalContext) CombatModifiers {
    return CombatModifiers{
        AttackBonus:   -2,  // Reduced offense
        DefenseBonus:  +5,  // High defense
        AccuracyBonus: 0,
        InitiativeBonus: -1,  // Slower
    }
}

func (f *DefensiveFormation) Validate(units []*ecs.Entity) error {
    if len(units) == 0 {
        return fmt.Errorf("defensive formation requires at least 1 unit")
    }
    return nil
}

func (f *DefensiveFormation) GetFormationGrid() [9]bool {
    return [9]bool{true, true, true, true, true, true, true, true, true}
}

// OffensiveFormation: Leader in front, high attack
type OffensiveFormation struct{}

func (f *OffensiveFormation) Name() string { return "Offensive" }

func (f *OffensiveFormation) ArrangUnits(units []*ecs.Entity) [9]*ecs.Entity {
    // Arrangement pattern:
    // [0][1][2]    [.][L][.]
    // [3][4][5] -> [R][R][R]  (L = Leader front, regulars behind)
    // [6][7][8]    [R][R][R]

    var grid [9]*ecs.Entity
    if len(units) == 0 {
        return grid
    }

    // Find leader
    leaderIdx := 0
    for i, u := range units {
        if u != nil && u.GetComponent("LeaderComponent") != nil {
            leaderIdx = i
            break
        }
    }

    // Place leader in front center (index 1)
    grid[1] = units[leaderIdx]

    // Place remaining units in back rows
    positions := []int{3, 4, 5, 6, 7, 8, 0, 2}
    unitIdx := 0
    for _, pos := range positions {
        if unitIdx >= len(units) {
            break
        }
        if unitIdx == leaderIdx {
            unitIdx++
            if unitIdx >= len(units) {
                break
            }
        }
        grid[pos] = units[unitIdx]
        unitIdx++
    }

    return grid
}

func (f *OffensiveFormation) GetCombatModifiers(ctx TacticalContext) CombatModifiers {
    return CombatModifiers{
        AttackBonus:   +5,  // High offense
        DefenseBonus:  -2,  // Reduced defense
        AccuracyBonus: +2,
        InitiativeBonus: +2,  // Faster
    }
}

func (f *OffensiveFormation) Validate(units []*ecs.Entity) error {
    if len(units) == 0 {
        return fmt.Errorf("offensive formation requires at least 1 unit")
    }
    return nil
}

func (f *OffensiveFormation) GetFormationGrid() [9]bool {
    return [9]bool{false, true, false, true, true, true, true, true, true}
}

// BalancedFormation: All-around, moderate bonuses
type BalancedFormation struct{}

func (f *BalancedFormation) Name() string { return "Balanced" }

func (f *BalancedFormation) ArrangUnits(units []*ecs.Entity) [9]*ecs.Entity {
    // Arrangement pattern:
    // [0][1][2]    [R][R][R]
    // [3][4][5] -> [L][R][R]  (L = Leader front-left)
    // [6][7][8]    [.][.][.]

    var grid [9]*ecs.Entity
    if len(units) == 0 {
        return grid
    }

    // Find leader
    leaderIdx := 0
    for i, u := range units {
        if u != nil && u.GetComponent("LeaderComponent") != nil {
            leaderIdx = i
            break
        }
    }

    // Place leader in front-left (index 3)
    grid[3] = units[leaderIdx]

    // Place remaining units
    positions := []int{0, 1, 2, 4, 5, 6, 7, 8}
    unitIdx := 0
    for _, pos := range positions {
        if unitIdx >= len(units) {
            break
        }
        if unitIdx == leaderIdx {
            unitIdx++
            if unitIdx >= len(units) {
                break
            }
        }
        grid[pos] = units[unitIdx]
        unitIdx++
    }

    return grid
}

func (f *BalancedFormation) GetCombatModifiers(ctx TacticalContext) CombatModifiers {
    return CombatModifiers{
        AttackBonus:   +2,
        DefenseBonus:  +2,
        AccuracyBonus: +1,
        InitiativeBonus: 0,
    }
}

func (f *BalancedFormation) Validate(units []*ecs.Entity) error {
    if len(units) == 0 {
        return fmt.Errorf("balanced formation requires at least 1 unit")
    }
    return nil
}

func (f *BalancedFormation) GetFormationGrid() [9]bool {
    return [9]bool{true, true, true, true, true, true, false, false, false}
}
```

### Squad Command Pattern

```go
// Package: combat/squadcommands.go

// SquadCommand represents an executable action
type SquadCommand interface {
    // Execute the command
    Execute(world *GameWorld) error

    // Validate if command is legal
    Validate(world *GameWorld) error

    // Preview expected outcome (for UI)
    Preview(world *GameWorld) CommandPreview

    // Command type for serialization
    Type() string
}

// CommandPreview for player feedback
type CommandPreview struct {
    Valid          bool
    ErrorMessage   string
    ExpectedDamage [2]int  // Min/max range
    HitChance      int
    Effects        []string
}

// --- MOVE COMMAND ---

type MoveSquadCommand struct {
    SquadID     ecs.EntityID
    Destination coords.LogicalPosition
}

func (cmd *MoveSquadCommand) Type() string { return "MoveSquad" }

func (cmd *MoveSquadCommand) Validate(world *GameWorld) error {
    squad := world.GetSquad(cmd.SquadID)
    if squad == nil {
        return fmt.Errorf("squad not found: %v", cmd.SquadID)
    }

    // Check if destination is walkable
    if !world.CoordManager.IsWalkable(cmd.Destination) {
        return fmt.Errorf("destination not walkable")
    }

    // Check movement range (future: use squad movement stat)
    distance := coords.ManhattanDistance(squad.LogicalPos, cmd.Destination)
    if distance > 5 {  // Example max movement
        return fmt.Errorf("destination too far (max 5 tiles)")
    }

    return nil
}

func (cmd *MoveSquadCommand) Execute(world *GameWorld) error {
    if err := cmd.Validate(world); err != nil {
        return err
    }

    squad := world.GetSquad(cmd.SquadID)

    // Update squad entity position
    posComp := squad.EntityID.GetComponent("PositionComponent").(*ecs.PositionComponent)
    pixelPos := world.CoordManager.LogicalToPixel(cmd.Destination)
    posComp.X = pixelPos.X
    posComp.Y = pixelPos.Y

    // Update squad logical position
    squad.LogicalPos = cmd.Destination

    return nil
}

func (cmd *MoveSquadCommand) Preview(world *GameWorld) CommandPreview {
    err := cmd.Validate(world)
    return CommandPreview{
        Valid:        err == nil,
        ErrorMessage: func() string { if err != nil { return err.Error() }; return "" }(),
    }
}

// --- ATTACK COMMAND ---

type AttackSquadCommand struct {
    AttackerID ecs.EntityID
    DefenderID ecs.EntityID
}

func (cmd *AttackSquadCommand) Type() string { return "AttackSquad" }

func (cmd *AttackSquadCommand) Validate(world *GameWorld) error {
    attacker := world.GetSquad(cmd.AttackerID)
    defender := world.GetSquad(cmd.DefenderID)

    if attacker == nil || defender == nil {
        return fmt.Errorf("invalid squad IDs")
    }

    // Check range (adjacent for melee)
    distance := coords.ManhattanDistance(attacker.LogicalPos, defender.LogicalPos)
    if distance > 1 {
        return fmt.Errorf("target out of range (melee requires adjacent)")
    }

    return nil
}

func (cmd *AttackSquadCommand) Execute(world *GameWorld) error {
    if err := cmd.Validate(world); err != nil {
        return err
    }

    attacker := world.GetSquad(cmd.AttackerID)
    defender := world.GetSquad(cmd.DefenderID)

    // Build tactical context
    ctx := TacticalContext{
        AttackingSquad: attacker,
        DefendingSquad: defender,
        Terrain:        world.GetTerrainAt(defender.LogicalPos),
        Flanking:       world.IsFlanking(attacker, defender),
        ElevationAdv:   false,  // Future
    }

    // Execute squad-vs-squad combat
    result := ExecuteSquadCombat(attacker, defender, ctx)

    // Apply damage to defender's pooled HP
    defender.TotalHP -= result.TotalDamage

    // Check for squad destruction
    if defender.TotalHP <= 0 {
        world.DestroySquad(defender)
    }

    return nil
}

func (cmd *AttackSquadCommand) Preview(world *GameWorld) CommandPreview {
    err := cmd.Validate(world)
    if err != nil {
        return CommandPreview{Valid: false, ErrorMessage: err.Error()}
    }

    attacker := world.GetSquad(cmd.AttackerID)
    defender := world.GetSquad(cmd.DefenderID)

    ctx := TacticalContext{
        AttackingSquad: attacker,
        DefendingSquad: defender,
    }

    preview := PreviewSquadCombat(attacker, defender, ctx)

    return CommandPreview{
        Valid:          true,
        ExpectedDamage: preview.DamageRange,
        HitChance:      preview.AverageHitChance,
        Effects:        preview.SpecialEffects,
    }
}

// --- FORMATION CHANGE COMMAND ---

type ChangeFormationCommand struct {
    SquadID        ecs.EntityID
    NewFormation   FormationPattern
}

func (cmd *ChangeFormationCommand) Type() string { return "ChangeFormation" }

func (cmd *ChangeFormationCommand) Validate(world *GameWorld) error {
    squad := world.GetSquad(cmd.SquadID)
    if squad == nil {
        return fmt.Errorf("squad not found")
    }

    // Collect non-nil units
    var units []*ecs.Entity
    for _, u := range squad.Units {
        if u != nil {
            units = append(units, u)
        }
    }

    return cmd.NewFormation.Validate(units)
}

func (cmd *ChangeFormationCommand) Execute(world *GameWorld) error {
    if err := cmd.Validate(world); err != nil {
        return err
    }

    squad := world.GetSquad(cmd.SquadID)

    // Collect non-nil units
    var units []*ecs.Entity
    for _, u := range squad.Units {
        if u != nil {
            units = append(units, u)
        }
    }

    // Rearrange with new formation
    squad.Units = cmd.NewFormation.ArrangUnits(units)
    squad.Formation = cmd.NewFormation

    // Update leader position
    for i, u := range squad.Units {
        if u != nil && u == squad.Leader {
            squad.LeaderPos = IndexToGridPos(i)
            break
        }
    }

    return nil
}

func (cmd *ChangeFormationCommand) Preview(world *GameWorld) CommandPreview {
    err := cmd.Validate(world)
    return CommandPreview{
        Valid:        err == nil,
        ErrorMessage: func() string { if err != nil { return err.Error() }; return "" }(),
        Effects:      []string{fmt.Sprintf("Change to %s formation", cmd.NewFormation.Name())},
    }
}
```

### Leader Ability System

```go
// Package: combat/abilities.go

// LeaderAbility is an executable ability with cooldowns
type LeaderAbility interface {
    // Ability metadata
    Name() string
    Description() string

    // Execute the ability
    Execute(caster *Squad, target interface{}, world *GameWorld) error

    // Validate if ability can be used
    Validate(caster *Squad, target interface{}, world *GameWorld) error

    // Cooldown tracking
    CurrentCooldown() int
    MaxCooldown() int
    SetCooldown(turns int)

    // Targeting requirements
    RequiresTarget() bool
    ValidTarget(target interface{}) bool
}

// BaseAbility provides common functionality
type BaseAbility struct {
    AbilityName string
    AbilityDesc string
    Cooldown    int
    MaxCD       int
}

func (a *BaseAbility) Name() string { return a.AbilityName }
func (a *BaseAbility) Description() string { return a.AbilityDesc }
func (a *BaseAbility) CurrentCooldown() int { return a.Cooldown }
func (a *BaseAbility) MaxCooldown() int { return a.MaxCD }
func (a *BaseAbility) SetCooldown(turns int) { a.Cooldown = turns }

// --- CONCRETE ABILITIES ---

// RallyAbility: Restore squad HP and morale
type RallyAbility struct {
    BaseAbility
    HealAmount int
}

func NewRallyAbility() *RallyAbility {
    return &RallyAbility{
        BaseAbility: BaseAbility{
            AbilityName: "Rally",
            AbilityDesc: "Restore squad HP and boost morale",
            Cooldown:    0,
            MaxCD:       3,
        },
        HealAmount: 20,
    }
}

func (a *RallyAbility) RequiresTarget() bool { return false }
func (a *RallyAbility) ValidTarget(target interface{}) bool { return target == nil }

func (a *RallyAbility) Validate(caster *Squad, target interface{}, world *GameWorld) error {
    if a.Cooldown > 0 {
        return fmt.Errorf("ability on cooldown (%d turns)", a.Cooldown)
    }
    return nil
}

func (a *RallyAbility) Execute(caster *Squad, target interface{}, world *GameWorld) error {
    if err := a.Validate(caster, target, world); err != nil {
        return err
    }

    // Heal squad
    caster.TotalHP += a.HealAmount

    // Cap at max HP
    maxHP := CalculateSquadMaxHP(caster)
    if caster.TotalHP > maxHP {
        caster.TotalHP = maxHP
    }

    // Boost morale (future)
    caster.Morale += 10

    // Set cooldown
    a.SetCooldown(a.MaxCD)

    return nil
}

// ChargeAbility: Bonus damage on next attack
type ChargeAbility struct {
    BaseAbility
    DamageMultiplier float64
}

func NewChargeAbility() *ChargeAbility {
    return &ChargeAbility{
        BaseAbility: BaseAbility{
            AbilityName: "Charge",
            AbilityDesc: "Next attack deals 150% damage",
            Cooldown:    0,
            MaxCD:       4,
        },
        DamageMultiplier: 1.5,
    }
}

func (a *ChargeAbility) RequiresTarget() bool { return false }
func (a *ChargeAbility) ValidTarget(target interface{}) bool { return target == nil }

func (a *ChargeAbility) Validate(caster *Squad, target interface{}, world *GameWorld) error {
    if a.Cooldown > 0 {
        return fmt.Errorf("ability on cooldown (%d turns)", a.Cooldown)
    }
    return nil
}

func (a *ChargeAbility) Execute(caster *Squad, target interface{}, world *GameWorld) error {
    if err := a.Validate(caster, target, world); err != nil {
        return err
    }

    // Apply buff to squad (stored in component)
    buffComp := caster.Leader.GetComponent("BuffComponent")
    if buffComp == nil {
        buffComp = &BuffComponent{Buffs: make(map[string]Buff)}
        caster.Leader.AddComponent("BuffComponent", buffComp)
    }

    bc := buffComp.(*BuffComponent)
    bc.Buffs["Charge"] = Buff{
        Name:       "Charge",
        Duration:   1,  // Next attack only
        DamageMult: a.DamageMultiplier,
    }

    a.SetCooldown(a.MaxCD)
    return nil
}

// TauntAbility: Force enemy to attack this squad
type TauntAbility struct {
    BaseAbility
    Duration int
}

func NewTauntAbility() *TauntAbility {
    return &TauntAbility{
        BaseAbility: BaseAbility{
            AbilityName: "Taunt",
            AbilityDesc: "Force enemy squad to attack you",
            Cooldown:    0,
            MaxCD:       5,
        },
        Duration: 2,
    }
}

func (a *TauntAbility) RequiresTarget() bool { return true }
func (a *TauntAbility) ValidTarget(target interface{}) bool {
    _, ok := target.(*Squad)
    return ok
}

func (a *TauntAbility) Validate(caster *Squad, target interface{}, world *GameWorld) error {
    if a.Cooldown > 0 {
        return fmt.Errorf("ability on cooldown (%d turns)", a.Cooldown)
    }

    if !a.ValidTarget(target) {
        return fmt.Errorf("invalid target (requires enemy squad)")
    }

    targetSquad := target.(*Squad)
    distance := coords.ManhattanDistance(caster.LogicalPos, targetSquad.LogicalPos)
    if distance > 3 {
        return fmt.Errorf("target too far (max 3 tiles)")
    }

    return nil
}

func (a *TauntAbility) Execute(caster *Squad, target interface{}, world *GameWorld) error {
    if err := a.Validate(caster, target, world); err != nil {
        return err
    }

    targetSquad := target.(*Squad)

    // Apply taunt debuff to target
    debuffComp := targetSquad.Leader.GetComponent("DebuffComponent")
    if debuffComp == nil {
        debuffComp = &DebuffComponent{Debuffs: make(map[string]Debuff)}
        targetSquad.Leader.AddComponent("DebuffComponent", debuffComp)
    }

    dc := debuffComp.(*DebuffComponent)
    dc.Debuffs["Taunt"] = Debuff{
        Name:       "Taunt",
        Duration:   a.Duration,
        ForcedTarget: caster.EntityID,
    }

    a.SetCooldown(a.MaxCD)
    return nil
}

// LeaderComponent stores customizable abilities
type LeaderComponent struct {
    AbilitySlots [4]LeaderAbility  // 4 customizable slots
}

// Example: Equip abilities to leader
func EquipAbilities(leader *ecs.Entity, abilities ...LeaderAbility) {
    leaderComp := &LeaderComponent{}
    for i, ability := range abilities {
        if i >= 4 {
            break  // Max 4 slots
        }
        leaderComp.AbilitySlots[i] = ability
    }
    leader.AddComponent("LeaderComponent", leaderComp)
}
```

---

## Combat Flow: Simultaneous Pooled Damage

### Step-by-Step Combat Sequence

```go
// Package: combat/squadcombat.go

type CombatResult struct {
    TotalDamage     int
    IndividualHits  []IndividualAttack
    CriticalHits    int
    Misses          int
}

type IndividualAttack struct {
    AttackerUnit *ecs.Entity
    DefenderUnit *ecs.Entity
    Damage       int
    Hit          bool
    Critical     bool
}

type CombatPreviewResult struct {
    DamageRange       [2]int   // Min/max expected damage
    AverageHitChance  int
    SpecialEffects    []string
}

// ExecuteSquadCombat: Simultaneous attack resolution
func ExecuteSquadCombat(attacker, defender *Squad, ctx TacticalContext) CombatResult {
    result := CombatResult{
        IndividualHits: make([]IndividualAttack, 0),
    }

    // Get formation modifiers
    attackMods := attacker.Formation.GetCombatModifiers(ctx)
    defenseMods := defender.Formation.GetCombatModifiers(ctx)

    // Iterate through all attacker units
    for i, attackUnit := range attacker.Units {
        if attackUnit == nil {
            continue  // Empty slot
        }

        // Find target in defender squad (use proximity/formation logic)
        defenderUnit := SelectDefenderTarget(defender, IndexToGridPos(i), ctx)
        if defenderUnit == nil {
            continue  // No valid target
        }

        // Perform individual attack using existing PerformAttack()
        attack := PerformIndividualAttack(attackUnit, defenderUnit, attackMods, defenseMods)
        result.IndividualHits = append(result.IndividualHits, attack)

        // Accumulate damage
        if attack.Hit {
            result.TotalDamage += attack.Damage
            if attack.Critical {
                result.CriticalHits++
            }
        } else {
            result.Misses++
        }
    }

    return result
}

// PerformIndividualAttack: Wrapper around existing PerformAttack()
func PerformIndividualAttack(attacker, defender *ecs.Entity, attackMods, defenseMods CombatModifiers) IndividualAttack {
    // Get base stats
    attackerStats := attacker.GetComponent("StatsComponent").(*StatsComponent)
    defenderStats := defender.GetComponent("StatsComponent").(*StatsComponent)

    // Apply formation modifiers
    modifiedAttack := attackerStats.Attack + attackMods.AttackBonus
    modifiedDefense := defenderStats.Defense + defenseMods.DefenseBonus
    modifiedAccuracy := attackerStats.Accuracy + attackMods.AccuracyBonus

    // Use existing d20 combat system
    hitRoll := rand.Intn(20) + 1
    hit := hitRoll + modifiedAccuracy >= defenderStats.Evasion

    attack := IndividualAttack{
        AttackerUnit: attacker,
        DefenderUnit: defender,
        Hit:          hit,
    }

    if !hit {
        return attack
    }

    // Calculate damage (existing formula)
    baseDamage := modifiedAttack - modifiedDefense
    if baseDamage < 1 {
        baseDamage = 1
    }

    // Check critical (natural 20)
    if hitRoll == 20 {
        attack.Critical = true
        baseDamage *= 2
    }

    attack.Damage = baseDamage
    return attack
}

// SelectDefenderTarget: Formation-aware target selection
func SelectDefenderTarget(defender *Squad, attackerPos GridPosition, ctx TacticalContext) *ecs.Entity {
    // Strategy: Target closest unit in defender's formation
    // For offensive formation: prioritize front row
    // For defensive formation: harder to reach leader (center protected)

    formationGrid := defender.Formation.GetFormationGrid()

    // Calculate "distance" within 3x3 grid (Manhattan)
    minDist := 999
    var closestUnit *ecs.Entity

    for i, occupied := range formationGrid {
        if !occupied || defender.Units[i] == nil {
            continue
        }

        defenderPos := IndexToGridPos(i)
        dist := abs(attackerPos.Row-defenderPos.Row) + abs(attackerPos.Col-defenderPos.Col)

        if dist < minDist {
            minDist = dist
            closestUnit = defender.Units[i]
        }
    }

    return closestUnit
}

// PreviewSquadCombat: Calculate expected outcome for UI
func PreviewSquadCombat(attacker, defender *Squad, ctx TacticalContext) CombatPreviewResult {
    attackMods := attacker.Formation.GetCombatModifiers(ctx)
    defenseMods := defender.Formation.GetCombatModifiers(ctx)

    minDamage := 0
    maxDamage := 0
    totalHitChance := 0
    attackCount := 0

    var effects []string

    // Simulate each attacking unit
    for i, attackUnit := range attacker.Units {
        if attackUnit == nil {
            continue
        }

        defenderUnit := SelectDefenderTarget(defender, IndexToGridPos(i), ctx)
        if defenderUnit == nil {
            continue
        }

        attackerStats := attackUnit.GetComponent("StatsComponent").(*StatsComponent)
        defenderStats := defenderUnit.GetComponent("StatsComponent").(*StatsComponent)

        modifiedAttack := attackerStats.Attack + attackMods.AttackBonus
        modifiedDefense := defenderStats.Defense + defenseMods.DefenseBonus
        modifiedAccuracy := attackerStats.Accuracy + attackMods.AccuracyBonus

        // Hit chance (simplified: d20 + accuracy >= evasion)
        // Approximate: (21 - (evasion - accuracy)) / 20
        hitChance := ((21 - (defenderStats.Evasion - modifiedAccuracy)) * 100) / 20
        if hitChance < 5 {
            hitChance = 5  // Minimum 5%
        }
        if hitChance > 95 {
            hitChance = 95  // Maximum 95%
        }

        totalHitChance += hitChance
        attackCount++

        // Damage range
        baseDamage := modifiedAttack - modifiedDefense
        if baseDamage < 1 {
            baseDamage = 1
        }

        minDamage += baseDamage
        maxDamage += baseDamage * 2  // Account for crits
    }

    // Average hit chance
    avgHitChance := 0
    if attackCount > 0 {
        avgHitChance = totalHitChance / attackCount
    }

    // Formation effects
    effects = append(effects, fmt.Sprintf("%s formation: +%d ATK, +%d DEF",
        attacker.Formation.Name(), attackMods.AttackBonus, attackMods.DefenseBonus))

    if ctx.Flanking {
        effects = append(effects, "Flanking bonus")
    }

    return CombatPreviewResult{
        DamageRange:      [2]int{minDamage, maxDamage},
        AverageHitChance: avgHitChance,
        SpecialEffects:   effects,
    }
}

func abs(x int) int {
    if x < 0 {
        return -x
    }
    return x
}
```

---

## 3x3 Grid Mechanics

### Grid Positioning with Unit Sizes

```go
// Package: combat/squadgrid.go

// UnitSize defines unit dimensions within 3x3 grid
type UnitSize int

const (
    UnitSize1x1 UnitSize = iota  // Single tile
    UnitSize2x1                  // 2 wide, 1 tall (horizontal)
    UnitSize1x2                  // 1 wide, 2 tall (vertical)
)

// UnitPlacement tracks unit position and size
type UnitPlacement struct {
    Entity    *ecs.Entity
    Size      UnitSize
    AnchorPos GridPosition  // Top-left corner
}

// CalculateOccupiedSlots returns all grid indices occupied by a unit
func (up *UnitPlacement) CalculateOccupiedSlots() []int {
    anchor := up.AnchorPos.ToIndex()

    switch up.Size {
    case UnitSize1x1:
        return []int{anchor}

    case UnitSize2x1:
        // Occupies [anchor] and [anchor+1] (horizontal)
        if up.AnchorPos.Col > 1 {
            return []int{anchor}  // Invalid, shrink to 1x1
        }
        return []int{anchor, anchor + 1}

    case UnitSize1x2:
        // Occupies [anchor] and [anchor+3] (vertical)
        if up.AnchorPos.Row > 1 {
            return []int{anchor}  // Invalid, shrink to 1x1
        }
        return []int{anchor, anchor + 3}

    default:
        return []int{anchor}
    }
}

// ValidatePlacement checks if unit can fit in grid
func (up *UnitPlacement) ValidatePlacement(occupiedSlots map[int]bool) error {
    slots := up.CalculateOccupiedSlots()

    for _, slot := range slots {
        if slot < 0 || slot >= 9 {
            return fmt.Errorf("unit placement out of bounds")
        }

        if occupiedSlots[slot] {
            return fmt.Errorf("slot %d already occupied", slot)
        }
    }

    return nil
}

// PlaceUnitInGrid: Helper for formation arrangement with unit sizes
func PlaceUnitInGrid(grid *[9]*ecs.Entity, unit *ecs.Entity, placement UnitPlacement) error {
    occupiedSlots := make(map[int]bool)
    for i, u := range grid {
        if u != nil {
            occupiedSlots[i] = true
        }
    }

    if err := placement.ValidatePlacement(occupiedSlots); err != nil {
        return err
    }

    // Place unit in all occupied slots (2x1 and 1x2 occupy multiple)
    slots := placement.CalculateOccupiedSlots()
    for _, slot := range slots {
        grid[slot] = unit
    }

    return nil
}

// Example: Formation with mixed unit sizes
func ExampleMixedSizeFormation() {
    var grid [9]*ecs.Entity

    // Create units
    leader := createUnit("Knight", UnitSize1x1)
    tank := createUnit("Heavy", UnitSize2x1)  // Wide unit
    archer := createUnit("Archer", UnitSize1x1)

    // Place in formation
    // [T][T][A]     (T = Tank 2x1, A = Archer, L = Leader)
    // [.][L][.]
    // [.][.][.]

    PlaceUnitInGrid(&grid, tank, UnitPlacement{
        Entity:    tank,
        Size:      UnitSize2x1,
        AnchorPos: GridPosition{Row: 0, Col: 0},  // Occupies [0] and [1]
    })

    PlaceUnitInGrid(&grid, archer, UnitPlacement{
        Entity:    archer,
        Size:      UnitSize1x1,
        AnchorPos: GridPosition{Row: 0, Col: 2},
    })

    PlaceUnitInGrid(&grid, leader, UnitPlacement{
        Entity:    leader,
        Size:      UnitSize1x1,
        AnchorPos: GridPosition{Row: 1, Col: 1},
    })

    // grid now has proper placements
}
```

---

## Integration Points

### 1. EntityManager Integration

```go
// Package: ecs/entitymanager.go

// Add squad creation to EntityManager
func (em *EntityManager) CreateSquad(template string, position coords.LogicalPosition) (*Squad, error) {
    // Create squad entity for world positioning
    squadEntity := em.CreateEntityFromTemplate(template)

    // Add SquadComponent to track squad membership
    squadComp := &SquadComponent{
        IsSquad: true,
        SquadID: squadEntity.ID,
    }
    squadEntity.AddComponent("SquadComponent", squadComp)

    // Create Squad data structure
    squad := &Squad{
        EntityID:   squadEntity.ID,
        LogicalPos: position,
        Formation:  &BalancedFormation{},  // Default formation
        Units:      [9]*ecs.Entity{},
        TotalHP:    0,
    }

    // Add units to squad (from template)
    unitTemplates := getSquadUnitTemplates(template)
    for i, unitTemplate := range unitTemplates {
        unit := em.CreateEntityFromTemplate(unitTemplate)
        squad.Units[i] = unit

        // Accumulate HP
        stats := unit.GetComponent("StatsComponent").(*StatsComponent)
        squad.TotalHP += stats.MaxHP
    }

    // Arrange with formation
    var units []*ecs.Entity
    for _, u := range squad.Units {
        if u != nil {
            units = append(units, u)
        }
    }
    squad.Units = squad.Formation.ArrangUnits(units)

    // Register squad in world
    GetGameWorld().RegisterSquad(squad)

    return squad, nil
}

// SquadComponent marks entity as squad
type SquadComponent struct {
    IsSquad bool
    SquadID ecs.EntityID
}
```

### 2. Input System Integration

```go
// Package: input/combatcontroller.go

// Handle squad combat actions via command pattern
func (cc *CombatController) HandleSquadAttack(attackerID, defenderID ecs.EntityID) error {
    // Create attack command
    cmd := &combat.AttackSquadCommand{
        AttackerID: attackerID,
        DefenderID: defenderID,
    }

    // Preview for UI feedback
    preview := cmd.Preview(cc.world)
    if !preview.Valid {
        return fmt.Errorf("invalid attack: %s", preview.ErrorMessage)
    }

    // Show preview to player
    cc.ui.ShowCombatPreview(preview)

    // Execute command
    return cmd.Execute(cc.world)
}

// Handle formation changes
func (cc *CombatController) HandleFormationChange(squadID ecs.EntityID, formationType string) error {
    squad := cc.world.GetSquad(squadID)

    // Map string to formation pattern
    var formation combat.FormationPattern
    switch formationType {
    case "defensive":
        formation = &combat.DefensiveFormation{}
    case "offensive":
        formation = &combat.OffensiveFormation{}
    case "balanced":
        formation = &combat.BalancedFormation{}
    default:
        return fmt.Errorf("unknown formation: %s", formationType)
    }

    cmd := &combat.ChangeFormationCommand{
        SquadID:      squadID,
        NewFormation: formation,
    }

    return cmd.Execute(cc.world)
}

// Handle leader abilities
func (cc *CombatController) HandleAbilityUse(squadID ecs.EntityID, abilitySlot int, target interface{}) error {
    squad := cc.world.GetSquad(squadID)

    leaderComp := squad.Leader.GetComponent("LeaderComponent").(*combat.LeaderComponent)
    if abilitySlot < 0 || abilitySlot >= len(leaderComp.AbilitySlots) {
        return fmt.Errorf("invalid ability slot")
    }

    ability := leaderComp.AbilitySlots[abilitySlot]
    if ability == nil {
        return fmt.Errorf("no ability equipped in slot %d", abilitySlot)
    }

    // Validate and execute
    if err := ability.Validate(squad, target, cc.world); err != nil {
        return err
    }

    return ability.Execute(squad, target, cc.world)
}
```

### 3. Spawning System Integration

```go
// Package: spawning/spawnmonsters.go

// Spawn enemy squads using probability system
func SpawnEnemySquad(level int, position coords.LogicalPosition) (*combat.Squad, error) {
    // Determine squad composition
    squadTemplate := selectSquadTemplate(level)

    // Use EntityManager to create squad
    em := ecs.GetEntityManager()
    squad, err := em.CreateSquad(squadTemplate, position)
    if err != nil {
        return nil, err
    }

    // Set AI behavior (future)
    aiComp := &AIComponent{
        Behavior:  "Aggressive",
        TargetID:  0,  // Will be set by AI system
    }
    squad.EntityID.AddComponent("AIComponent", aiComp)

    return squad, nil
}

// Squad templates for spawning
func selectSquadTemplate(level int) string {
    templates := map[int][]string{
        1: {"WeakGoblins", "ScoutSquad"},
        2: {"MixedGoblins", "OrcsSquad"},
        3: {"EliteOrcs", "TrollSquad"},
    }

    options := templates[level]
    if len(options) == 0 {
        return "WeakGoblins"  // Default
    }

    return options[rand.Intn(len(options))]
}
```

### 4. PerformAttack() Wrapper Pattern

```go
// Package: combat/attackingsystem.go

// Existing PerformAttack() PRESERVED for backward compatibility
func PerformAttack(attacker, defender *ecs.Entity, world *GameWorld) AttackResult {
    // Original implementation unchanged
    // Used for non-squad combat (future: individual duels, special scenarios)
    // ...existing code...
}

// NEW: Squad-aware attack wrapper
func PerformSquadOrIndividualAttack(attacker, defender *ecs.Entity, world *GameWorld) interface{} {
    attackerSquad := world.GetSquadByEntityID(attacker.ID)
    defenderSquad := world.GetSquadByEntityID(defender.ID)

    // Both are squads: use squad combat
    if attackerSquad != nil && defenderSquad != nil {
        ctx := TacticalContext{
            AttackingSquad: attackerSquad,
            DefendingSquad: defenderSquad,
        }
        return ExecuteSquadCombat(attackerSquad, defenderSquad, ctx)
    }

    // At least one is individual: use original PerformAttack()
    return PerformAttack(attacker, defender, world)
}
```

---

## Pros & Cons Analysis

### Advantages

**Extreme Extensibility**
- New formations are simple classes implementing FormationPattern interface
- New abilities are simple classes implementing LeaderAbility interface
- Command pattern allows easy addition of new action types
- Easy to serialize/save formations and abilities

**Tactical Depth**
- Formation switching creates meaningful mid-battle decisions
- Leader abilities provide unique squad identities
- Command preview enables informed player choices
- Formation modifiers create rock-paper-scissors dynamics

**Clean Separation of Concerns**
- Squad positioning (ECS entity) separate from composition (data structure)
- Combat logic separate from validation (command pattern)
- Formation strategies separate from squad data
- Easy to test in isolation (mock FormationPattern, mock LeaderAbility)

**Genre Convention Alignment**
- Command-based actions match Fire Emblem/FFT player expectations
- Formation strategies match Soul Nomad/Symphony of War
- Ability systems match FFT job system patterns
- Preview system matches TRPG UI conventions

**Future-Proof**
- Easy to add: flanking bonuses, terrain effects, morale, multi-target abilities
- Command serialization enables replay/undo systems
- Interface-based design allows runtime swapping (DLC formations, modding)

### Disadvantages

**High Initial Complexity**
- More interfaces and abstractions than Approaches 1 or 2
- Steeper learning curve for future contributors
- More files to maintain (formations.go, abilities.go, squadcommands.go, squadgrid.go)
- Potential over-engineering for simple use cases

**Performance Overhead**
- Interface calls have slight performance cost (negligible for turn-based)
- Command validation runs twice (preview + execute)
- More heap allocations (interface boxes)

**Maintenance Burden**
- Each new formation requires full interface implementation
- More moving parts to debug
- Harder to refactor (more dependencies on interfaces)

**Boilerplate Code**
- BaseAbility pattern requires repetitive code
- Formation implementations have similar structure
- Command pattern requires validate/execute/preview for each action

---

## Implementation Effort

### Complexity Assessment

**Risk Level:** HIGH
**Effort:** 24-32 hours
**Lines of Code:** ~1200-1500 LOC

### Breakdown by Component

| Component | Estimated Hours | LOC | Complexity |
|-----------|----------------|-----|------------|
| Squad data structure | 2h | 100 | Low |
| Formation interface + 3 implementations | 4h | 250 | Medium |
| Command pattern (Move/Attack/Formation) | 4h | 200 | Medium |
| Squad combat resolution | 4h | 200 | High |
| Leader ability interface + 3 abilities | 4h | 250 | Medium |
| 3x3 grid mechanics with unit sizes | 3h | 150 | Medium |
| EntityManager integration | 2h | 100 | Low |
| Input system integration | 3h | 150 | Medium |
| Spawning integration | 2h | 100 | Low |
| Testing & balancing | 4h | - | High |
| **Total** | **32h** | **~1500** | **High** |

### Phased Implementation Plan

**Phase 1: Core Infrastructure (8h)**
- Squad data structure
- Basic formation interface + Balanced formation
- EntityManager.CreateSquad()
- Builds and compiles

**Phase 2: Combat System (8h)**
- Squad combat resolution
- Command pattern (Attack/Move)
- PerformSquadCombat() with pooled damage
- Basic testing

**Phase 3: Formations (6h)**
- Defensive formation
- Offensive formation
- Formation change command
- Combat modifiers

**Phase 4: Leader Abilities (6h)**
- Ability interface
- 3 concrete abilities (Rally/Charge/Taunt)
- LeaderComponent
- Ability execution

**Phase 5: Polish & Integration (4h)**
- 3x3 grid with unit sizes
- Input system integration
- Spawning integration
- Balance testing

---

## Code Examples

### Example 1: Creating a Squad with Formation

```go
// Create player squad with offensive formation
func CreatePlayerSquad(world *GameWorld, startPos coords.LogicalPosition) (*combat.Squad, error) {
    em := ecs.GetEntityManager()

    // Create squad entity
    squad, err := em.CreateSquad("PlayerSquadTemplate", startPos)
    if err != nil {
        return nil, err
    }

    // Set offensive formation
    changeFormation := &combat.ChangeFormationCommand{
        SquadID:      squad.EntityID,
        NewFormation: &combat.OffensiveFormation{},
    }

    if err := changeFormation.Execute(world); err != nil {
        return nil, err
    }

    // Equip leader abilities
    leaderAbilities := []combat.LeaderAbility{
        combat.NewChargeAbility(),
        combat.NewRallyAbility(),
        combat.NewTauntAbility(),
    }

    combat.EquipAbilities(squad.Leader, leaderAbilities...)

    return squad, nil
}
```

### Example 2: Executing Squad-vs-Squad Attack

```go
// Player attacks enemy squad
func PlayerAttacksEnemy(world *GameWorld, playerSquadID, enemySquadID ecs.EntityID) error {
    // Create attack command
    cmd := &combat.AttackSquadCommand{
        AttackerID: playerSquadID,
        DefenderID: enemySquadID,
    }

    // Preview outcome for UI
    preview := cmd.Preview(world)
    if !preview.Valid {
        fmt.Printf("Attack failed: %s\n", preview.ErrorMessage)
        return fmt.Errorf(preview.ErrorMessage)
    }

    // Show preview to player
    fmt.Printf("Attack Preview:\n")
    fmt.Printf("  Hit Chance: %d%%\n", preview.HitChance)
    fmt.Printf("  Expected Damage: %d-%d\n", preview.ExpectedDamage[0], preview.ExpectedDamage[1])
    fmt.Printf("  Effects: %v\n", preview.Effects)

    // Execute attack
    if err := cmd.Execute(world); err != nil {
        return err
    }

    fmt.Println("Attack successful!")
    return nil
}
```

### Example 3: Leader Ability Swap

```go
// Swap leader ability mid-game (at camp/shop)
func SwapLeaderAbility(squad *combat.Squad, slotIndex int, newAbility combat.LeaderAbility) error {
    leaderComp := squad.Leader.GetComponent("LeaderComponent").(*combat.LeaderComponent)

    if slotIndex < 0 || slotIndex >= 4 {
        return fmt.Errorf("invalid ability slot (0-3)")
    }

    // Store old ability (for inventory/reuse)
    oldAbility := leaderComp.AbilitySlots[slotIndex]
    if oldAbility != nil {
        // Add to inventory (future feature)
        fmt.Printf("Removed ability: %s\n", oldAbility.Name())
    }

    // Equip new ability
    leaderComp.AbilitySlots[slotIndex] = newAbility
    fmt.Printf("Equipped ability: %s\n", newAbility.Name())

    return nil
}

// Example usage:
func ExampleAbilitySwap() {
    // At camp: swap Charge for Taunt
    playerSquad := getPlayerSquad()

    newAbility := combat.NewTauntAbility()
    SwapLeaderAbility(playerSquad, 0, newAbility)  // Slot 0: Charge -> Taunt
}
```

### Example 4: Formation Switching Mid-Battle

```go
// Switch formation based on tactical situation
func AdaptFormation(world *GameWorld, squadID ecs.EntityID, enemyCount int) error {
    squad := world.GetSquad(squadID)

    // Determine best formation
    var newFormation combat.FormationPattern

    if squad.TotalHP < 30 {
        // Low HP: defensive
        newFormation = &combat.DefensiveFormation{}
        fmt.Println("Switching to DEFENSIVE formation (low HP)")
    } else if enemyCount > 3 {
        // Outnumbered: defensive
        newFormation = &combat.DefensiveFormation{}
        fmt.Println("Switching to DEFENSIVE formation (outnumbered)")
    } else {
        // Strong position: offensive
        newFormation = &combat.OffensiveFormation{}
        fmt.Println("Switching to OFFENSIVE formation (advantage)")
    }

    // Execute formation change
    cmd := &combat.ChangeFormationCommand{
        SquadID:      squadID,
        NewFormation: newFormation,
    }

    return cmd.Execute(world)
}
```

### Example 5: Complete Combat Turn

```go
// Full player turn with abilities and formation
func ExecutePlayerTurn(world *GameWorld, playerSquadID ecs.EntityID) error {
    squad := world.GetSquad(playerSquadID)

    // 1. Use Rally ability (if available)
    leaderComp := squad.Leader.GetComponent("LeaderComponent").(*combat.LeaderComponent)
    rallyAbility := leaderComp.AbilitySlots[1]  // Assume Rally in slot 1

    if rallyAbility != nil && rallyAbility.CurrentCooldown() == 0 && squad.TotalHP < 50 {
        fmt.Println("Using Rally ability...")
        rallyAbility.Execute(squad, nil, world)
    }

    // 2. Switch to offensive formation
    fmt.Println("Switching to offensive formation...")
    formationCmd := &combat.ChangeFormationCommand{
        SquadID:      playerSquadID,
        NewFormation: &combat.OffensiveFormation{},
    }
    formationCmd.Execute(world)

    // 3. Move toward enemy
    enemySquad := world.FindNearestEnemySquad(squad.LogicalPos)
    if enemySquad != nil {
        targetPos := coords.LogicalPosition{
            X: enemySquad.LogicalPos.X - 1,  // Move adjacent
            Y: enemySquad.LogicalPos.Y,
        }

        moveCmd := &combat.MoveSquadCommand{
            SquadID:     playerSquadID,
            Destination: targetPos,
        }

        if err := moveCmd.Execute(world); err != nil {
            fmt.Printf("Move failed: %v\n", err)
        } else {
            fmt.Println("Moved to attack position")
        }
    }

    // 4. Attack if in range
    if enemySquad != nil {
        attackCmd := &combat.AttackSquadCommand{
            AttackerID: playerSquadID,
            DefenderID: enemySquad.EntityID,
        }

        preview := attackCmd.Preview(world)
        if preview.Valid {
            fmt.Printf("Attacking! Expected damage: %d-%d\n",
                preview.ExpectedDamage[0], preview.ExpectedDamage[1])
            attackCmd.Execute(world)
        }
    }

    // 5. Decrement ability cooldowns (end of turn)
    for _, ability := range leaderComp.AbilitySlots {
        if ability != nil && ability.CurrentCooldown() > 0 {
            ability.SetCooldown(ability.CurrentCooldown() - 1)
        }
    }

    return nil
}
```

---

## Testing Strategy

### Build Verification
```bash
go build -o game_main/game_main.exe game_main/*.go
```

### Manual Testing Scenarios

**Formation Testing:**
1. Create squad with defensive formation
2. Verify leader in center (index 4)
3. Attack with offensive enemy squad
4. Verify defensive bonuses applied (+5 DEF, -2 ATK)
5. Switch to offensive formation mid-battle
6. Verify leader moved to front (index 1)
7. Verify offensive bonuses applied (+5 ATK, -2 DEF)

**Combat Testing:**
1. Squad vs Squad attack
2. Verify all units participate
3. Verify damage pooling (accumulated to TotalHP)
4. Verify formation modifiers affect outcome
5. Test squad destruction (TotalHP <= 0)

**Ability Testing:**
1. Use Rally ability (verify HP increase, cooldown set)
2. Wait 3 turns (verify cooldown decrements)
3. Use Charge ability (verify buff applied)
4. Attack (verify 1.5x damage, buff consumed)
5. Use Taunt on enemy (verify forced targeting)

**Grid Testing:**
1. Place 2x1 unit in grid (verify occupies 2 slots)
2. Attempt invalid placement (verify error)
3. Mix 1x1, 2x1, 1x2 units (verify correct arrangement)

### Balance Testing

**Formation Balance:**
- Offensive vs Defensive: Both viable depending on HP/situation
- Balanced formation: Jack-of-all-trades fallback
- No dominant strategy (player must adapt)

**Ability Balance:**
- Rally: Useful when low HP (not always best choice)
- Charge: High risk/reward (wasted if miss)
- Taunt: Tactical (protects weak allies)
- All have trade-offs (cooldowns prevent spam)

**Combat Power Curve:**
- Early game: 1-2 units per squad (simple)
- Mid game: 3-5 units (tactical depth emerges)
- Late game: 6-9 units (full formation strategies)

---

## Recommendations

### When to Use This Approach

**Choose Hybrid Command-Pattern if:**
- You want maximum tactical depth and extensibility
- You plan many formations and leader abilities
- You value clean separation of concerns
- You expect frequent balance iterations (formations are configurable)
- You want replay/undo systems (command pattern enables this)
- You prioritize testability (easy to mock interfaces)

### When to Use Simpler Approach

**Choose Approach 1 or 2 if:**
- You want faster initial implementation
- You have limited formation variety (2-3 total)
- You don't need complex ability systems
- You prioritize simplicity over extensibility
- You want fewer files to maintain

### Hybrid Decision: Start Simple, Upgrade Later

**Recommendation:** Start with Approach 2 (Squad-as-Entity), add command pattern later

**Phase 1:** Implement Squad-as-Entity (Approach 2)
- Get basic squad combat working (12-16h)
- Single formation (balanced)
- No abilities yet

**Phase 2:** Add formation system (Approach 3)
- Implement FormationPattern interface (+4h)
- Add defensive/offensive formations (+3h)
- Minimal disruption (formation is just a field)

**Phase 3:** Add command pattern (Approach 3)
- Wrap existing combat in AttackSquadCommand (+3h)
- Add ChangeFormationCommand (+2h)
- Enable preview system (+2h)

**Phase 4:** Add leader abilities (Approach 3)
- Implement LeaderAbility interface (+4h)
- Add 3 abilities (+4h)

**Total:** Same effort (~32h), but incremental delivery and risk mitigation

---

## Summary

**Approach 3: Hybrid Command-Pattern Architecture**

**Best For:** Maximum tactical depth, extensibility, and genre convention alignment
**Complexity:** High (24-32 hours)
**Strengths:** Extreme extensibility, clean separation, tactical depth, future-proof
**Weaknesses:** High initial complexity, more abstractions, maintenance burden
**Recommended Path:** Incremental adoption starting from Approach 2

**Key Insight:** Command pattern and strategy pattern provide TRPG-grade tactical systems but require upfront architectural investment. Best suited for projects expecting heavy feature iteration (balance patches, DLC formations, modding support).

**Tactical Gameplay Impact:**
- Formation switching creates meaningful mid-battle decisions
- Leader abilities differentiate squad identities (offensive leader vs defensive leader)
- Command preview enables informed player choices (Fire Emblem-style)
- Modular design allows easy addition of new formations and abilities

**Integration Compatibility:**
- ✅ Works with existing EntityManager
- ✅ Works with InputCoordinator (command pattern fits controller pattern)
- ✅ Works with PerformAttack() (wrapper pattern preserves existing logic)
- ✅ Works with spawning system (CreateSquad factory)

**Final Verdict:** Choose this if you want a **professional-grade tactical RPG combat system** similar to Fire Emblem/FFT, and you're willing to invest in proper architecture upfront.
