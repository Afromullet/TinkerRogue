# Squad-Based Combat System Refactoring Analysis

## Executive Summary

**Current State:** 1v1 combat system with PlayerData-centric architecture
**Target State:** Multi-squad tactical combat (Nephilim/Soul Nomad style)
**Priority:** HIGH - This is a foundational change that impacts spawning, AI, and level systems
**Risk Level:** CRITICAL - Requires careful incremental refactoring to avoid breaking existing gameplay

---

## 1. ARCHITECTURAL REQUIREMENTS

### 1.1 Core Abstractions Needed

#### Squad Structure
A squad is a tactical unit containing multiple entities with coordinated behavior:

```go
// Squad represents a tactical unit with multiple members
type Squad struct {
    ID           string
    Name         string
    Leader       *ecs.Entity        // Optional leader with command bonuses
    Members      []*ecs.Entity      // All units in the squad
    Formation    Formation          // Squad positioning pattern
    Allegiance   Allegiance         // Player-controlled, enemy, neutral
    IsActive     bool              // Can this squad act this turn?
    ActionPoints int               // Squad-level resource for actions
}

type Allegiance int
const (
    AllegiancePlayer Allegiance = iota
    AllegianceEnemy
    AllegianceNeutral
)

// Formation defines how squad members position themselves
type Formation struct {
    Type         FormationType     // Line, Box, Wedge, Scatter
    Spacing      int              // Tiles between members
    FacingDir    coords.Direction // Squad orientation
    Positions    []coords.LogicalPosition // Relative positions for each member
}

type FormationType int
const (
    FormationLine FormationType = iota
    FormationBox
    FormationWedge
    FormationScatter
)
```

#### Unit Type Differentiation
Units have roles that determine their capabilities and position in formations:

```go
// UnitClass defines the tactical role of a unit within a squad
type UnitClass struct {
    Type         UnitType
    Abilities    []Ability         // Class-specific abilities
    CanLeadSquad bool
    FormationPos FormationPosition // Preferred position in formation
}

type UnitType int
const (
    UnitTypeTank UnitType = iota    // Front-line, high HP/AC
    UnitTypeDamage                  // Damage dealers
    UnitTypeSupport                 // Healers, buffers
    UnitTypeUtility                 // Crowd control, debuffers
    UnitTypeRanged                  // Archers, mages
)

type FormationPosition int
const (
    FormationFront FormationPosition = iota
    FormationMiddle
    FormationRear
)

// Ability represents unit/squad-specific tactical actions
type Ability struct {
    Name        string
    Type        AbilityType
    Cooldown    int
    CurrentCD   int
    Cost        AbilityCost      // Mana, stamina, action points
    TargetType  TargetType       // Self, single, squad, area
    Effect      AbilityEffect
}

type AbilityType int
const (
    AbilityPassive AbilityType = iota
    AbilityActive
    AbilityReaction  // Triggers on conditions (opportunity attacks, counters)
)

type TargetType int
const (
    TargetSelf TargetType = iota
    TargetSingleAlly
    TargetSingleEnemy
    TargetSquad      // Affects entire squad
    TargetArea       // AOE targeting
)

type AbilityCost struct {
    ActionPoints int
    Mana         int
    Stamina      int
}

type AbilityEffect interface {
    Apply(caster *ecs.Entity, targets []*ecs.Entity, em *common.EntityManager) error
    Validate(caster *ecs.Entity, targets []*ecs.Entity) bool
}
```

#### Command Pattern for Squad Actions
Decouple action requests from execution to support undo, replay, and AI decision-making:

```go
// SquadCommand represents an action a squad wants to perform
type SquadCommand interface {
    Execute(em *common.EntityManager, gm *worldmap.GameMap) error
    Validate() bool
    GetActionCost() int
    GetSquad() *Squad
}

// MoveSquadCommand handles squad movement with formation preservation
type MoveSquadCommand struct {
    Squad       *Squad
    Destination coords.LogicalPosition
    MaintainFormation bool
}

func (c *MoveSquadCommand) Execute(em *common.EntityManager, gm *worldmap.GameMap) error {
    if !c.Validate() {
        return fmt.Errorf("invalid move command")
    }

    if c.MaintainFormation {
        return moveSquadWithFormation(c.Squad, c.Destination, gm)
    }
    return moveSquadIndividually(c.Squad, c.Destination, gm)
}

func (c *MoveSquadCommand) Validate() bool {
    // Check if destination is valid, squad has movement points, etc.
    return c.Squad != nil && c.Squad.IsActive
}

func (c *MoveSquadCommand) GetActionCost() int {
    // Calculate cost based on distance and formation complexity
    return 1
}

func (c *MoveSquadCommand) GetSquad() *Squad {
    return c.Squad
}

// AttackSquadCommand handles squad vs squad combat
type AttackSquadCommand struct {
    AttackingSquad *Squad
    DefendingSquad *Squad
    Attacker       *ecs.Entity // Specific unit in squad
    Targets        []*ecs.Entity
    AbilityUsed    *Ability
}

func (c *AttackSquadCommand) Execute(em *common.EntityManager, gm *worldmap.GameMap) error {
    // Use existing PerformAttack logic but with squad context
    return executeSquadAttack(em, gm, c)
}

// UseAbilityCommand handles special abilities (spells, skills)
type UseAbilityCommand struct {
    Squad    *Squad
    Caster   *ecs.Entity
    Ability  *Ability
    Targets  []*ecs.Entity
}
```

#### Turn/Action Ordering System
Initiative system that handles multiple squads with complex ordering:

```go
// TurnManager handles initiative and action ordering for all squads
type TurnManager struct {
    TurnOrder     []*TurnEntry
    CurrentIndex  int
    TurnNumber    int
    Phase         TurnPhase
}

type TurnEntry struct {
    Squad        *Squad
    Initiative   int  // d20 + modifiers
    HasActed     bool
    ActionsUsed  int
    ActionsMax   int  // Based on squad composition
}

type TurnPhase int
const (
    PhaseInitiative TurnPhase = iota  // Roll initiative, determine order
    PhaseMovement                     // Squad movement phase
    PhaseAction                       // Main actions (attacks, abilities)
    PhaseReaction                     // Opportunity attacks, counters
    PhaseCleanup                      // Status effects, cooldowns
)

func (tm *TurnManager) CalculateInitiative(squad *Squad) int {
    // Roll d20 + average squad dexterity + leader bonus
    baseRoll := randgen.GetDiceRoll(20)
    avgDex := calculateAverageAttribute(squad.Members, "Dexterity")
    leaderBonus := 0
    if squad.Leader != nil {
        leaderBonus = getLeadershipBonus(squad.Leader)
    }
    return baseRoll + avgDex + leaderBonus
}

func (tm *TurnManager) NextTurn() *Squad {
    tm.CurrentIndex++
    if tm.CurrentIndex >= len(tm.TurnOrder) {
        tm.CurrentIndex = 0
        tm.TurnNumber++
        tm.processEndOfRound()
    }

    entry := tm.TurnOrder[tm.CurrentIndex]
    entry.HasActed = false
    entry.ActionsUsed = 0
    return entry.Squad
}

func (tm *TurnManager) processEndOfRound() {
    // Apply status effects, regenerate resources, cooldowns
    for _, entry := range tm.TurnOrder {
        updateSquadEffects(entry.Squad)
        regenerateSquadResources(entry.Squad)
        decrementCooldowns(entry.Squad)
    }
}
```

---

## 2. REFACTORING STRATEGY

### 2.1 Can PerformAttack() Be Reused?

**Answer: YES, with wrapper layer**

The core combat math in `PerformAttack()` is solid:
- d20 + attack bonus vs AC
- Dodge roll
- Damage - protection calculation

**PRESERVE:**
```go
// Keep this function unchanged for individual combat resolution
func PerformAttack(em *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap,
    damage int, attacker *ecs.Entity, defender *ecs.Entity, isPlayerAttacking bool) bool {
    // Current implementation stays the same
}
```

**ADD NEW LAYER:**
```go
// New squad-level combat that delegates to PerformAttack
func ResolveSquadCombat(em *common.EntityManager, gm *worldmap.GameMap,
    attackCmd *AttackSquadCommand) error {

    attackerSquad := attackCmd.AttackingSquad
    defenderSquad := attackCmd.DefendingSquad

    // Determine target selection based on formation and abilities
    targets := selectTargets(attackerSquad, defenderSquad, attackCmd.AbilityUsed)

    // Execute attacks for each attacker-target pair
    for _, attacker := range attackCmd.Attacker {
        for _, target := range targets {
            weapon := getWeapon(attacker)
            damage := weapon.CalculateDamage()

            // Apply squad bonuses (flanking, morale, formations)
            damage = applySquadModifiers(damage, attackerSquad, defenderSquad)

            // Reuse existing combat resolution
            isPlayerSquad := attackerSquad.Allegiance == AllegiancePlayer
            hit := PerformAttack(em, nil, gm, damage, attacker, target, isPlayerSquad)

            // Track combat results for squad morale/effects
            recordCombatResult(attackerSquad, defenderSquad, hit)
        }
    }

    return nil
}

// Calculate tactical modifiers based on squad positioning
func applySquadModifiers(baseDamage int, attacker *Squad, defender *Squad) int {
    modifier := 1.0

    // Flanking bonus
    if isSquadFlanking(attacker, defender) {
        modifier += 0.25
    }

    // Formation bonus (wedge formation increases damage)
    if attacker.Formation.Type == FormationWedge {
        modifier += 0.15
    }

    // Morale modifier
    modifier += getMoraleModifier(attacker)

    return int(float64(baseDamage) * modifier)
}
```

### 2.2 PlayerData Transition Strategy

**Problem:** Current system is PlayerData-centric. Squad system needs faction-agnostic design.

**Incremental Migration Path:**

#### Phase 1: Introduce PlayerSquad Wrapper (2-3 hours)
```go
// Wrapper that converts PlayerData into Squad format
type PlayerSquad struct {
    Squad *Squad
    Data  *avatar.PlayerData  // Keep existing PlayerData temporarily
}

func (ps *PlayerSquad) ToSquad() *Squad {
    return &Squad{
        ID:         "player_squad_main",
        Name:       "Player",
        Members:    []*ecs.Entity{ps.Data.PlayerEntity},
        Allegiance: AllegiancePlayer,
        IsActive:   true,
        ActionPoints: 3,
    }
}

// Update combat calls to use squad wrapper
func (ps *PlayerSquad) Attack(em *common.EntityManager, gm *worldmap.GameMap,
    target *ecs.Entity) error {
    cmd := &AttackSquadCommand{
        AttackingSquad: ps.ToSquad(),
        Attacker:       ps.Data.PlayerEntity,
        Targets:        []*ecs.Entity{target},
    }
    return cmd.Execute(em, gm)
}
```

#### Phase 2: Decouple PlayerData from Combat (4-6 hours)
```go
// Replace PlayerData parameter with Squad/Allegiance
func PerformAttackV2(em *common.EntityManager, gm *worldmap.GameMap,
    damage int, attacker *ecs.Entity, defender *ecs.Entity,
    attackerAllegiance Allegiance) bool {

    // Remove PlayerData dependency
    isPlayerAttacking := attackerAllegiance == AllegiancePlayer

    // Same combat logic, but entity removal is squad-aware
    if isPlayerAttacking {
        resmanager.RemoveEntityFromSquad(em.World, gm, defender)
    }

    return false
}
```

#### Phase 3: Multi-Squad Support (6-8 hours)
```go
// SquadManager replaces single PlayerData
type SquadManager struct {
    Squads         map[string]*Squad
    PlayerSquads   []*Squad  // Player controls multiple squads
    EnemySquads    []*Squad
    CurrentSquad   *Squad    // Which squad is active
    TurnManager    *TurnManager
}

func (sm *SquadManager) GetPlayerSquads() []*Squad {
    return sm.PlayerSquads
}

func (sm *SquadManager) AddSquad(squad *Squad) {
    sm.Squads[squad.ID] = squad

    switch squad.Allegiance {
    case AllegiancePlayer:
        sm.PlayerSquads = append(sm.PlayerSquads, squad)
    case AllegianceEnemy:
        sm.EnemySquads = append(sm.EnemySquads, squad)
    }
}
```

### 2.3 Input System Integration

**Good News:** Input system refactoring is COMPLETE (per CLAUDE.md).

**Integration Point:**
```go
// Input already uses controller pattern - extend for squad selection
type SquadController struct {
    squadManager  *SquadManager
    selectedSquad *Squad
    commandQueue  []SquadCommand
}

func (sc *SquadController) HandleInput(input *InputState) {
    // Tab key: cycle through player squads
    if input.KeyPressed(ebiten.KeyTab) {
        sc.selectedSquad = sc.squadManager.NextPlayerSquad()
    }

    // Number keys: directly select squad
    if input.KeyPressed(ebiten.Key1) {
        sc.selectedSquad = sc.squadManager.GetPlayerSquad(0)
    }

    // Movement: move selected squad
    if input.MoveDirection != nil {
        cmd := &MoveSquadCommand{
            Squad: sc.selectedSquad,
            Destination: sc.selectedSquad.Members[0].Position.Add(input.MoveDirection),
        }
        sc.commandQueue = append(sc.commandQueue, cmd)
    }
}
```

---

## 3. CODE EXAMPLES

### Example 1: Squad Creation and Management

```go
// squadmanager.go - New file for squad operations

package combat

import (
    "game_main/common"
    "game_main/coords"
    "github.com/bytearena/ecs"
)

// CreateSquadFromEntities assembles entities into a tactical squad
func CreateSquadFromEntities(id, name string, members []*ecs.Entity,
    allegiance Allegiance, formation FormationType) *Squad {

    squad := &Squad{
        ID:         id,
        Name:       name,
        Members:    members,
        Allegiance: allegiance,
        IsActive:   true,
        ActionPoints: calculateSquadActionPoints(members),
        Formation: Formation{
            Type:    formation,
            Spacing: 1,
        },
    }

    // Determine leader (highest leadership attribute)
    squad.Leader = findSquadLeader(members)

    // Calculate formation positions
    squad.Formation.Positions = calculateFormationPositions(
        len(members), formation, squad.Formation.Spacing)

    return squad
}

func calculateSquadActionPoints(members []*ecs.Entity) int {
    // Base 2 actions + bonus for each member above 1
    base := 2
    bonus := len(members) - 1
    if bonus < 0 {
        bonus = 0
    }
    return base + bonus
}

func findSquadLeader(members []*ecs.Entity) *ecs.Entity {
    var leader *ecs.Entity
    maxLeadership := -1

    for _, member := range members {
        if class := common.GetComponentType[*UnitClass](member, UnitClassComponent); class != nil {
            if class.CanLeadSquad {
                // Use intelligence or charisma as leadership stat
                attr := common.GetAttributes(member)
                leadership := attr.AttackBonus // Placeholder for leadership stat
                if leadership > maxLeadership {
                    maxLeadership = leadership
                    leader = member
                }
            }
        }
    }

    return leader
}

func calculateFormationPositions(count int, formation FormationType,
    spacing int) []coords.LogicalPosition {

    positions := make([]coords.LogicalPosition, count)

    switch formation {
    case FormationLine:
        // Horizontal line
        for i := 0; i < count; i++ {
            positions[i] = coords.LogicalPosition{
                X: i * spacing,
                Y: 0,
            }
        }

    case FormationBox:
        // 2x2 or 2x3 box
        cols := 2
        for i := 0; i < count; i++ {
            positions[i] = coords.LogicalPosition{
                X: (i % cols) * spacing,
                Y: (i / cols) * spacing,
            }
        }

    case FormationWedge:
        // Triangle formation
        //     0
        //   1   2
        // 3   4   5
        row := 0
        col := 0
        for i := 0; i < count; i++ {
            positions[i] = coords.LogicalPosition{
                X: col * spacing,
                Y: row * spacing,
            }
            col++
            if col > row {
                row++
                col = 0
            }
        }

    case FormationScatter:
        // Random positions within radius
        for i := 0; i < count; i++ {
            positions[i] = coords.LogicalPosition{
                X: randgen.GetRandomBetween(-2, 2) * spacing,
                Y: randgen.GetRandomBetween(-2, 2) * spacing,
            }
        }
    }

    return positions
}

// ApplyFormation moves squad members to formation positions relative to leader
func (s *Squad) ApplyFormation(anchor coords.LogicalPosition, gm *worldmap.GameMap) error {
    if len(s.Members) != len(s.Formation.Positions) {
        return fmt.Errorf("formation mismatch: %d members, %d positions",
            len(s.Members), len(s.Formation.Positions))
    }

    for i, member := range s.Members {
        targetPos := coords.LogicalPosition{
            X: anchor.X + s.Formation.Positions[i].X,
            Y: anchor.Y + s.Formation.Positions[i].Y,
        }

        // Check if position is valid
        if !gm.IsWalkable(targetPos) {
            // Find nearest valid position
            targetPos = findNearestWalkable(targetPos, gm)
        }

        // Update entity position
        pos := common.GetPosition(member)
        pos.X = targetPos.X
        pos.Y = targetPos.Y

        // Update map blocking
        oldIndex := coords.CoordManager.LogicalToIndex(*pos)
        newIndex := coords.CoordManager.LogicalToIndex(targetPos)
        gm.Tiles[oldIndex].Blocked = false
        gm.Tiles[newIndex].Blocked = true
    }

    return nil
}
```

### Example 2: Squad vs Squad Combat Flow

```go
// squadcombat.go - New file for squad combat resolution

package combat

import (
    "fmt"
    "game_main/common"
    "game_main/coords"
    "game_main/worldmap"
    "github.com/bytearena/ecs"
)

// SquadCombatResolver handles combat between two squads
type SquadCombatResolver struct {
    em *common.EntityManager
    gm *worldmap.GameMap
}

func NewSquadCombatResolver(em *common.EntityManager, gm *worldmap.GameMap) *SquadCombatResolver {
    return &SquadCombatResolver{em: em, gm: gm}
}

// ExecuteSquadAttack processes an attack command from one squad to another
func (scr *SquadCombatResolver) ExecuteSquadAttack(cmd *AttackSquadCommand) error {
    if err := scr.validateAttackCommand(cmd); err != nil {
        return err
    }

    // Check if this is a special ability or normal attack
    if cmd.AbilityUsed != nil {
        return scr.executeAbilityAttack(cmd)
    }
    return scr.executeNormalAttack(cmd)
}

func (scr *SquadCombatResolver) executeNormalAttack(cmd *AttackSquadCommand) error {
    results := &CombatResults{
        Hits:   0,
        Misses: 0,
        Kills:  0,
    }

    // For each attacker in the attacking squad
    attackers := cmd.AttackingSquad.Members
    if cmd.Attacker != nil {
        // Single unit attacking
        attackers = []*ecs.Entity{cmd.Attacker}
    }

    for _, attacker := range attackers {
        // Determine targets based on range and targeting rules
        targets := scr.selectTargetsForAttacker(attacker, cmd.DefendingSquad)

        for _, target := range targets {
            // Check if attacker can reach target
            attackerPos := common.GetPosition(attacker)
            targetPos := common.GetPosition(target)

            weapon := scr.getWeaponForAttacker(attacker)
            if !scr.canAttack(attackerPos, targetPos, weapon) {
                continue
            }

            // Calculate base damage
            damage := weapon.CalculateDamage()

            // Apply squad-level modifiers
            damage = scr.applySquadModifiers(damage, cmd.AttackingSquad,
                cmd.DefendingSquad, attacker, target)

            // Execute the attack using existing PerformAttack logic
            isPlayerAttacking := cmd.AttackingSquad.Allegiance == AllegiancePlayer
            hit := PerformAttack(scr.em, nil, scr.gm, damage, attacker, target,
                isPlayerAttacking)

            // Record results
            if hit {
                results.Hits++
                // Check if target was killed
                if scr.isEntityDead(target) {
                    results.Kills++
                    scr.removeFromSquad(target, cmd.DefendingSquad)
                }
            } else {
                results.Misses++
            }
        }
    }

    // Update combat messages and morale
    scr.updateCombatFeedback(cmd.AttackingSquad, cmd.DefendingSquad, results)

    return nil
}

// selectTargetsForAttacker determines which enemies to attack
func (scr *SquadCombatResolver) selectTargetsForAttacker(attacker *ecs.Entity,
    defendingSquad *Squad) []*ecs.Entity {

    attackerPos := common.GetPosition(attacker)
    weapon := scr.getWeaponForAttacker(attacker)

    targets := []*ecs.Entity{}

    // Melee weapons target closest enemy
    if weapon.IsMelee() {
        closest := scr.findClosestMember(attackerPos, defendingSquad)
        if closest != nil {
            targets = append(targets, closest)
        }
    } else {
        // Ranged weapons can target multiple based on AOE
        for _, member := range defendingSquad.Members {
            memberPos := common.GetPosition(member)
            if attackerPos.InRange(memberPos, weapon.GetRange()) {
                targets = append(targets, member)
            }
        }
    }

    return targets
}

// applySquadModifiers calculates tactical bonuses/penalties
func (scr *SquadCombatResolver) applySquadModifiers(baseDamage int,
    attackSquad *Squad, defendSquad *Squad,
    attacker *ecs.Entity, defender *ecs.Entity) int {

    modifier := 1.0

    // Flanking: attacker's squad surrounds defender's squad
    if scr.isSquadFlanking(attackSquad, defendSquad) {
        modifier += 0.25
    }

    // Formation bonuses
    switch attackSquad.Formation.Type {
    case FormationWedge:
        modifier += 0.15 // Damage bonus
    case FormationBox:
        modifier += 0.05 // Small damage bonus, mainly defensive
    }

    // Leader aura bonus
    if attackSquad.Leader != nil {
        leaderPos := common.GetPosition(attackSquad.Leader)
        attackerPos := common.GetPosition(attacker)
        if leaderPos.InRange(attackerPos, 5) { // Leader must be nearby
            modifier += 0.10
        }
    }

    // Numerical advantage
    if len(attackSquad.Members) > len(defendSquad.Members) {
        modifier += 0.05 * float64(len(attackSquad.Members)-len(defendSquad.Members))
    }

    return int(float64(baseDamage) * modifier)
}

// isSquadFlanking checks if attacking squad has members on opposite sides
func (scr *SquadCombatResolver) isSquadFlanking(attacker *Squad, defender *Squad) bool {
    if len(attacker.Members) < 2 || len(defender.Members) == 0 {
        return false
    }

    // Calculate center of defending squad
    defenderCenter := scr.calculateSquadCenter(defender)

    // Check if attackers are on opposite sides of defender center
    angles := make([]float64, len(attacker.Members))
    for i, member := range attacker.Members {
        pos := common.GetPosition(member)
        angles[i] = calculateAngle(defenderCenter, *pos)
    }

    // If any two attackers are >135 degrees apart, consider it flanking
    for i := 0; i < len(angles); i++ {
        for j := i + 1; j < len(angles); j++ {
            diff := abs(angles[i] - angles[j])
            if diff > 135 && diff < 225 {
                return true
            }
        }
    }

    return false
}

func (scr *SquadCombatResolver) calculateSquadCenter(squad *Squad) coords.LogicalPosition {
    if len(squad.Members) == 0 {
        return coords.LogicalPosition{X: 0, Y: 0}
    }

    sumX, sumY := 0, 0
    for _, member := range squad.Members {
        pos := common.GetPosition(member)
        sumX += pos.X
        sumY += pos.Y
    }

    return coords.LogicalPosition{
        X: sumX / len(squad.Members),
        Y: sumY / len(squad.Members),
    }
}

func (scr *SquadCombatResolver) removeFromSquad(entity *ecs.Entity, squad *Squad) {
    for i, member := range squad.Members {
        if member.ID() == entity.ID() {
            squad.Members = append(squad.Members[:i], squad.Members[i+1:]...)
            break
        }
    }

    // If squad is now empty, mark as inactive
    if len(squad.Members) == 0 {
        squad.IsActive = false
    }
}

type CombatResults struct {
    Hits   int
    Misses int
    Kills  int
}

func (scr *SquadCombatResolver) updateCombatFeedback(attacker *Squad,
    defender *Squad, results *CombatResults) {

    // Update messages for player-visible squads
    if attacker.Allegiance == AllegiancePlayer {
        msg := fmt.Sprintf("%s attacks: %d hits, %d misses, %d kills",
            attacker.Name, results.Hits, results.Misses, results.Kills)

        // Set message on first member (temporary until UI update)
        if len(attacker.Members) > 0 {
            userMsg := common.GetComponentType[*common.UserMessage](
                attacker.Members[0], common.UserMsgComponent)
            userMsg.AttackMessage = msg
        }
    }
}

// Helper methods
func (scr *SquadCombatResolver) validateAttackCommand(cmd *AttackSquadCommand) error {
    if cmd.AttackingSquad == nil || cmd.DefendingSquad == nil {
        return fmt.Errorf("missing squad in attack command")
    }
    if !cmd.AttackingSquad.IsActive {
        return fmt.Errorf("attacking squad is not active")
    }
    if len(cmd.DefendingSquad.Members) == 0 {
        return fmt.Errorf("defending squad has no members")
    }
    return nil
}

func (scr *SquadCombatResolver) getWeaponForAttacker(attacker *ecs.Entity) Weapon {
    // Try melee first
    if melee := common.GetComponentType[*gear.MeleeWeapon](attacker,
        gear.MeleeWeaponComponent); melee != nil {
        return melee
    }
    // Try ranged
    if ranged := common.GetComponentType[*gear.RangedWeapon](attacker,
        gear.RangedWeaponComponent); ranged != nil {
        return ranged
    }
    // Default to fists (not yet implemented)
    return nil
}

func (scr *SquadCombatResolver) canAttack(attackerPos, targetPos *coords.LogicalPosition,
    weapon Weapon) bool {
    if weapon == nil {
        return false
    }

    range := weapon.GetRange()
    if weapon.IsMelee() {
        range = 1
    }

    return attackerPos.InRange(targetPos, range)
}

func (scr *SquadCombatResolver) findClosestMember(pos *coords.LogicalPosition,
    squad *Squad) *ecs.Entity {
    var closest *ecs.Entity
    minDist := 999999

    for _, member := range squad.Members {
        memberPos := common.GetPosition(member)
        dist := pos.Distance(memberPos)
        if dist < minDist {
            minDist = dist
            closest = member
        }
    }

    return closest
}

func (scr *SquadCombatResolver) isEntityDead(entity *ecs.Entity) bool {
    attr := common.GetAttributes(entity)
    return attr.CurrentHealth <= 0
}

// Weapon interface for unified weapon handling
type Weapon interface {
    CalculateDamage() int
    GetRange() int
    IsMelee() bool
}
```

### Example 3: Integration with Existing Systems

```go
// game_main/main.go - Integration points

package main

import (
    "game_main/avatar"
    "game_main/combat"
    "game_main/common"
    "game_main/worldmap"
)

type Game struct {
    // BEFORE: Single player entity
    // player *avatar.PlayerData

    // AFTER: Squad management
    squadManager *combat.SquadManager
    turnManager  *combat.TurnManager

    // Existing systems
    ecsManager   *common.EntityManager
    gameMap      *worldmap.GameMap
    // ... other fields
}

func (g *Game) initializeSquadSystem() {
    g.squadManager = combat.NewSquadManager()
    g.turnManager = combat.NewTurnManager()

    // Convert existing player to player squad (backward compatibility)
    playerSquad := g.createPlayerSquad()
    g.squadManager.AddSquad(playerSquad)

    // Create initial enemy squads
    enemySquad1 := g.createEnemySquad("Goblin Warband", 3)
    g.squadManager.AddSquad(enemySquad1)

    // Initialize turn order
    g.turnManager.RollInitiative(g.squadManager.GetAllSquads())
}

func (g *Game) createPlayerSquad() *combat.Squad {
    // For now, single-entity squad for backward compatibility
    members := []*ecs.Entity{g.playerEntity}

    squad := combat.CreateSquadFromEntities(
        "player_main",
        "Hero Squad",
        members,
        combat.AllegiancePlayer,
        combat.FormationLine,
    )

    return squad
}

func (g *Game) createEnemySquad(name string, memberCount int) *combat.Squad {
    members := make([]*ecs.Entity, memberCount)

    // Use existing entity creation
    for i := 0; i < memberCount; i++ {
        monster := entitytemplates.CreateCreatureFromTemplate(
            *g.ecsManager,
            monsterTemplates["goblin"],
            g.gameMap,
            10 + i*2, // Spread out
            10,
        )
        members[i] = monster
    }

    squad := combat.CreateSquadFromEntities(
        "enemy_"+name,
        name,
        members,
        combat.AllegianceEnemy,
        combat.FormationLine,
    )

    return squad
}

// Update combat flow to use squad system
func (g *Game) handleCombat() {
    currentSquad := g.turnManager.GetCurrentSquad()

    if currentSquad.Allegiance == combat.AllegiancePlayer {
        // Player's turn - wait for input
        g.handlePlayerSquadTurn(currentSquad)
    } else {
        // AI's turn
        g.handleAISquadTurn(currentSquad)
    }
}

func (g *Game) handlePlayerSquadTurn(squad *combat.Squad) {
    // Input controller already refactored, just extend it
    cmd := g.inputCoordinator.GetSquadCommand(squad)

    if cmd != nil {
        cmd.Execute(g.ecsManager, g.gameMap)
        g.turnManager.NextTurn()
    }
}

func (g *Game) handleAISquadTurn(squad *combat.Squad) {
    // AI creates attack command
    playerSquads := g.squadManager.GetPlayerSquads()
    if len(playerSquads) > 0 {
        target := playerSquads[0] // Target first player squad

        cmd := &combat.AttackSquadCommand{
            AttackingSquad: squad,
            DefendingSquad: target,
        }

        cmd.Execute(g.ecsManager, g.gameMap)
    }

    g.turnManager.NextTurn()
}
```

---

## 4. PRIORITY ASSESSMENT

### Is Squad Combat Blocking Other Todos?

**Answer: PARTIALLY BLOCKING**

#### Blocking Impact by Todo:

1. **Spawning System** - BLOCKED (Medium Impact)
   - Current spawning creates individual entities
   - Squad system changes how entities are grouped and managed
   - **Recommendation:** Implement basic squad system BEFORE finalizing spawning
   - **Workaround:** Can spawn individual entities, add squad grouping later

2. **Throwing System** - NOT BLOCKED (Low Impact)
   - Throwing is entity-level, not squad-level
   - Can implement independently
   - Will need squad-awareness for AOE targeting later

3. **Level Transitions** - NOT BLOCKED (Low Impact)
   - Level changes are map-level operations
   - Squad membership persists across levels
   - Can implement level transitions, add squad persistence later

4. **AI System** - BLOCKED (High Impact)
   - Current AI targets individual player entity
   - Squad-based combat requires squad-aware AI
   - AI needs to understand formations, flanking, targeting
   - **Cannot implement proper tactical AI without squad system**

5. **Balance/Difficulty** - BLOCKED (High Impact)
   - Difficulty calculation changes dramatically with squads
   - Need squad-level power metrics
   - Cannot properly balance individual monsters without squad context

### Recommended Implementation Order:

```
Priority 1 (CRITICAL - Do First):
├── Basic Squad Structure (Squad, SquadManager)
├── PlayerData → PlayerSquad wrapper
└── Single-squad combat testing

Priority 2 (HIGH - Do Before AI):
├── Formation system
├── Multi-squad support
├── Turn/initiative system
└── Command pattern implementation

Priority 3 (MEDIUM - Can Parallelize):
├── Spawning system (squad-aware)
├── Squad-based AI
└── Ability/spell system

Priority 4 (LOW - Polish):
├── Advanced formations
├── Morale system
└── Squad-level UI improvements
```

### Incremental Rollout Strategy

**Week 1: Foundation (12-16 hours)**
- Day 1-2: Create Squad struct, SquadManager, basic squad creation
- Day 3: Wrap PlayerData in PlayerSquad, update combat calls
- Day 4-5: Test single squad vs single entity combat

**Week 2: Multi-Squad Support (16-20 hours)**
- Day 1-2: Formation system and positioning
- Day 3: Turn manager and initiative
- Day 4-5: Squad vs squad combat testing

**Week 3: Integration (12-16 hours)**
- Day 1-2: Update spawning for squad awareness
- Day 3-4: Basic squad AI
- Day 5: Integration testing, bug fixes

**Week 4: Polish & Abilities (8-12 hours)**
- Day 1-2: Ability system foundation
- Day 3: UI updates for squad selection
- Day 4-5: Balance testing

### Risk Mitigation

**Critical Risks:**
1. **Breaking Existing Gameplay** - Use wrapper pattern to maintain backward compatibility
2. **Performance Degradation** - Profile early, optimize pathfinding and combat loops
3. **AI Complexity Explosion** - Start with simple AI, iterate

**Risk Reduction Tactics:**
- Keep `PerformAttack()` unchanged - proven combat math
- Use command pattern for easy undo/debugging
- Feature flag squad system (`SQUAD_COMBAT_ENABLED`) for gradual rollout
- Maintain PlayerData alongside Squad system during transition

---

## 5. MIGRATION CHECKLIST

### Phase 1: Infrastructure (Non-Breaking)
- [ ] Create `combat/squad.go` with Squad, Formation, UnitClass types
- [ ] Create `combat/squadmanager.go` with SquadManager
- [ ] Create `combat/squadcommand.go` with command pattern interfaces
- [ ] Create `combat/turnmanager.go` with initiative system
- [ ] Add feature flag `SQUAD_COMBAT_ENABLED = false`

### Phase 2: Wrapper Layer (Non-Breaking)
- [ ] Create PlayerSquad wrapper around PlayerData
- [ ] Update InputCoordinator to support squad selection (optional mode)
- [ ] Add PerformAttackV2 that accepts Allegiance instead of PlayerData
- [ ] Create SquadCombatResolver with squad-level attack logic

### Phase 3: Parallel Systems (Breaking for New Features)
- [ ] Enable feature flag `SQUAD_COMBAT_ENABLED = true`
- [ ] Convert player to multi-squad support
- [ ] Update spawning to create squads instead of individuals
- [ ] Create squad-aware AI

### Phase 4: Deprecation (Breaking)
- [ ] Remove PlayerData from combat functions
- [ ] Replace all PerformAttack calls with PerformAttackV2
- [ ] Remove single-entity combat paths
- [ ] Update all systems to be squad-native

---

## 6. CONCLUSION

**Key Takeaways:**

1. **Preserve Core Combat Math:** `PerformAttack()` is solid - wrap, don't replace
2. **Incremental Migration:** Use PlayerSquad wrapper to avoid big-bang refactor
3. **Command Pattern:** Essential for squad actions, AI, and future features
4. **Formation System:** Core tactical depth comes from positioning
5. **Turn Management:** Initiative and action economy create tactical decisions

**Effort Estimate:** 40-60 hours for full implementation
**Risk Level:** CRITICAL - requires careful testing
**Blocking Status:** Partially blocks AI and spawning, does not block throwing/levels

**Next Immediate Steps:**
1. Create `combat/squad.go` with basic types
2. Implement SquadManager for squad tracking
3. Build PlayerSquad wrapper for backward compatibility
4. Test single squad combat using existing PerformAttack logic
