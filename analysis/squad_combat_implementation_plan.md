# Squad-Based Combat System Implementation Plan

**Last Updated:** 2025-10-01
**Target Architecture:** bytearena/ecs (Component-Rich Approach)
**Status:** Ready for Implementation
**Estimated Total Time:** 32-38 hours

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Component Definitions](#component-definitions)
3. [Row-Based Targeting System](#row-based-targeting-system)
4. [Automated Leader Abilities](#automated-leader-abilities)
5. [Squad Composition Flexibility](#squad-composition-flexibility)
6. [Implementation Phases](#implementation-phases)
7. [Code Examples](#code-examples)
8. [Integration Points](#integration-points)
9. [Testing Strategy](#testing-strategy)
10. [Open Questions](#open-questions)

---

## Architecture Overview

### Design Philosophy

**Core Principle:** Squads are separate entities that "own" unit entities through relationship components. The 3x3 grid is internal to each squad, not a global map structure.

**Key Differences from APPROACH_1 (Donburi → bytearena/ecs):**
- Components are `*ecs.Component` pointers (not types like `donburi.ComponentType[T]`)
- No `donburi.Entry` - use `*ecs.Entity` directly
- No `donburi.World` - use `*ecs.Manager` (wrapped as `common.EntityManager`)
- Tags are used for querying (not filters)
- Component data stored as `interface{}`, requires type assertion via `GetComponentType[T]()`

**Entity Hierarchy:**
```
SquadEntity (has SquadComponent, LogicalPosition)
├── UnitEntity1 (has SquadMemberComponent → points back to SquadEntity)
├── UnitEntity2 (has SquadMemberComponent)
└── UnitEntity3 (has LeaderComponent + SquadMemberComponent)
```

**Component Relationships:**
- `SquadComponent`: Lives on squad entity, references unit entities via `[]*ecs.Entity`
- `SquadMemberComponent`: Lives on unit entity, references parent squad via `*ecs.Entity`
- `GridPositionComponent`: Lives on unit entity, stores row/col within 3x3 grid
- `UnitRoleComponent`: Lives on unit entity, stores Tank/DPS/Support role
- `LeaderComponent`: Lives on unit entity marked as squad leader
- `TargetRowComponent`: Lives on unit entity, declares which enemy row to attack

---

## Component Definitions

### File: `squad/components.go`

```go
package squad

import (
	"github.com/bytearena/ecs"
)

// Global component declarations (registered in InitSquadComponents)
var (
	SquadComponent         *ecs.Component
	SquadMemberComponent   *ecs.Component
	GridPositionComponent  *ecs.Component
	UnitRoleComponent      *ecs.Component
	LeaderComponent        *ecs.Component
	TargetRowComponent     *ecs.Component
	AbilityConditionComponent *ecs.Component
)

// InitSquadComponents registers all squad-related components with the ECS manager.
// Call this during game initialization.
func InitSquadComponents(manager *ecs.Manager) {
	SquadComponent = manager.NewComponent()
	SquadMemberComponent = manager.NewComponent()
	GridPositionComponent = manager.NewComponent()
	UnitRoleComponent = manager.NewComponent()
	LeaderComponent = manager.NewComponent()
	TargetRowComponent = manager.NewComponent()
	AbilityConditionComponent = manager.NewComponent()
}

// SquadData represents the squad entity's component data.
// Lives on the squad entity itself (the "container" entity).
type SquadData struct {
	UnitEntities   []*ecs.Entity  // References to unit entities (max 9)
	Formation      FormationType   // Current formation layout
	Name           string          // Squad display name
	Morale         int             // Squad-wide morale (future)
	SquadLevel     int             // Average level for spawning
	TurnCount      int             // Number of turns this squad has taken (for ability triggers)
}

// SquadMemberData links a unit back to its parent squad.
// Lives on each unit entity within the squad.
type SquadMemberData struct {
	SquadEntity    *ecs.Entity    // Parent squad entity
	IsLeader       bool           // True for commander unit
}

// GridPositionData represents a unit's position within the 3x3 grid.
// Lives on each unit entity.
type GridPositionData struct {
	Row    int  // 0-2 (top to bottom: 0=front, 1=mid, 2=back)
	Col    int  // 0-2 (left to right)
}

// UnitRoleData defines a unit's combat role.
// Lives on each unit entity.
type UnitRoleData struct {
	Role UnitRole  // Tank, DPS, or Support
}

// UnitRole affects combat behavior and damage distribution.
type UnitRole int
const (
	ROLE_TANK UnitRole = iota  // Takes hits first, high defense
	ROLE_DPS                    // High damage output
	ROLE_SUPPORT                // Buffs, heals, utility
)

func (r UnitRole) String() string {
	switch r {
	case ROLE_TANK:
		return "Tank"
	case ROLE_DPS:
		return "DPS"
	case ROLE_SUPPORT:
		return "Support"
	default:
		return "Unknown"
	}
}

// FormationType defines squad layout presets.
type FormationType int
const (
	FORMATION_BALANCED FormationType = iota  // Mix of roles
	FORMATION_DEFENSIVE                       // Tank-heavy
	FORMATION_OFFENSIVE                       // DPS-focused
	FORMATION_RANGED                          // Back-line heavy
)

// LeaderData marks a unit as the squad leader with special abilities.
// Lives on the leader unit entity.
type LeaderData struct {
	AbilitySlots   [4]*AbilityInstance  // Equipped abilities (FFT-style)
	Leadership     int                   // Bonus to squad stats
	Experience     int                   // Leader progression (future)
}

// AbilityInstance represents an equipped ability with cooldown tracking.
type AbilityInstance struct {
	AbilityID      string                // "Rally", "Heal", "BattleCry"
	Cooldown       int                   // Turns remaining until ready
	MaxCooldown    int                   // Base cooldown duration
}

// TargetRowData defines which enemy row(s) a unit attacks.
// Lives on each unit entity.
type TargetRowData struct {
	TargetRows     []int       // Which rows to target (e.g., [0] for front, [0,1,2] for all)
	IsMultiTarget  bool        // True if ability hits multiple units in row
	MaxTargets     int         // Max units hit per row (0 = all)
}

// AbilityConditionData defines automated triggers for leader abilities.
// Lives on leader unit entity.
type AbilityConditionData struct {
	Conditions []AbilityCondition  // List of trigger conditions
}

// AbilityCondition represents a single trigger condition.
type AbilityCondition struct {
	AbilityID       string              // Which ability to trigger
	TriggerType     TriggerType         // When to check
	Threshold       float64             // Condition threshold (e.g., 0.5 for 50% HP)
	HasTriggered    bool                // Prevents multiple triggers per combat
}

// TriggerType defines when abilities are checked.
type TriggerType int
const (
	TRIGGER_SQUAD_HP_BELOW TriggerType = iota  // Squad average HP < threshold
	TRIGGER_TURN_COUNT                          // Specific turn number
	TRIGGER_ENEMY_COUNT                         // Number of enemy squads
	TRIGGER_MORALE_BELOW                        // Squad morale < threshold
	TRIGGER_COMBAT_START                        // First turn of combat
)
```

### Tags for Querying

```go
// File: squad/tags.go

package squad

import "github.com/bytearena/ecs"

// Global tags for efficient entity queries
var (
	SquadTag       ecs.Tag
	SquadMemberTag ecs.Tag
	LeaderTag      ecs.Tag
	TankTag        ecs.Tag
	DPSTag         ecs.Tag
	SupportTag     ecs.Tag
)

// InitSquadTags creates tags for querying squad-related entities.
// Call this after InitSquadComponents.
func InitSquadTags() {
	SquadTag = ecs.BuildTag(SquadComponent)
	SquadMemberTag = ecs.BuildTag(SquadMemberComponent)
	LeaderTag = ecs.BuildTag(LeaderComponent, SquadMemberComponent)
	// Role-specific tags would require additional filtering in queries
}

// Example usage:
// for _, result := range ecsmanager.World.Query(squad.SquadTag) {
//     squadEntity := result.Entity
//     squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)
//     // ... process squad
// }
```

---

## Row-Based Targeting System

### Design Overview

**Core Concept:** Units declare which row(s) they target via `TargetRowComponent`. During combat, attacks are resolved row-by-row, with single-target vs AOE logic applied per unit.

**Row Numbering:**
- Row 0: Front line (closest to enemy)
- Row 1: Middle line
- Row 2: Back line (furthest from enemy)

**Targeting Scenarios:**
1. **Single-target front row:** Archer targets one unit in enemy row 0
2. **Multi-target back row:** Mage hits all units in enemy row 2
3. **All-row AOE:** Catapult hits one random unit in each row
4. **Specific grid:** Assassin targets specific grid position (row=1, col=2)

### TargetRowComponent Design

```go
// Already defined in components.go, repeated here for clarity:
type TargetRowData struct {
	TargetRows     []int       // Which rows to target (e.g., [0] for front, [2] for back)
	IsMultiTarget  bool        // True if ability hits multiple units in row
	MaxTargets     int         // Max units hit per row (0 = unlimited)
}

// Example configurations:
// Melee fighter (front row only, single target):
// TargetRowData{TargetRows: []int{0}, IsMultiTarget: false, MaxTargets: 1}

// Archer (back row, single target):
// TargetRowData{TargetRows: []int{2}, IsMultiTarget: false, MaxTargets: 1}

// Mage (entire back row, AOE):
// TargetRowData{TargetRows: []int{2}, IsMultiTarget: true, MaxTargets: 0}

// Pike formation (front and mid rows, limited targets):
// TargetRowData{TargetRows: []int{0, 1}, IsMultiTarget: true, MaxTargets: 2}

// Artillery (all rows, one per row):
// TargetRowData{TargetRows: []int{0, 1, 2}, IsMultiTarget: false, MaxTargets: 1}
```

### Damage Distribution Algorithm

**File:** `squad/combat.go`

```go
package squad

import (
	"game_main/common"
	"game_main/randgen"
	"math/rand"
)

// ExecuteSquadAttack performs row-based combat between two squads.
// Each unit in the attacker squad resolves its attack based on TargetRowComponent.
func ExecuteSquadAttack(attackerSquad, defenderSquad *ecs.Entity, ecsmanager *common.EntityManager) *CombatResult {
	result := &CombatResult{
		DamageByUnit: make(map[*ecs.Entity]int),
		UnitsKilled:  []*ecs.Entity{},
	}

	attackerData := common.GetComponentType[*SquadData](attackerSquad, SquadComponent)
	defenderData := common.GetComponentType[*SquadData](defenderSquad, SquadComponent)

	// Build defender grid for efficient row lookups
	defenderGrid := BuildSquadGrid(defenderData)

	// Process each attacker unit
	for _, attackerUnit := range attackerData.UnitEntities {
		if attackerUnit == nil {
			continue
		}

		// Check if unit is alive
		attackerAttr := common.GetAttributes(attackerUnit)
		if attackerAttr.CurrentHealth <= 0 {
			continue
		}

		// Get targeting data
		targetRowData := common.GetComponentType[*TargetRowData](attackerUnit, TargetRowComponent)

		// Execute attack for each target row
		for _, targetRow := range targetRowData.TargetRows {
			targets := defenderGrid.GetUnitsInRow(targetRow)

			if len(targets) == 0 {
				continue // No units in this row
			}

			// Determine how many targets to hit
			var actualTargets []*ecs.Entity
			if targetRowData.IsMultiTarget {
				// AOE: hit multiple/all units in row
				maxTargets := targetRowData.MaxTargets
				if maxTargets == 0 || maxTargets > len(targets) {
					actualTargets = targets // Hit all
				} else {
					// Randomly select subset
					actualTargets = selectRandomTargets(targets, maxTargets)
				}
			} else {
				// Single target: pick one unit (prefer lowest HP for realism)
				actualTargets = []*ecs.Entity{selectLowestHPTarget(targets)}
			}

			// Apply damage to each selected target
			for _, defenderUnit := range actualTargets {
				damage := calculateUnitDamage(attackerUnit, defenderUnit)
				applyDamageToUnit(defenderUnit, damage, result)
			}
		}
	}

	// Remove dead units from squad
	for _, deadUnit := range result.UnitsKilled {
		removeUnitFromSquad(deadUnit, defenderSquad, defenderData, defenderGrid)
	}

	result.TotalDamage = sumDamageMap(result.DamageByUnit)

	return result
}

// CombatResult holds the outcome of a squad attack.
type CombatResult struct {
	TotalDamage    int
	UnitsKilled    []*ecs.Entity
	DamageByUnit   map[*ecs.Entity]int
}

// calculateUnitDamage computes damage from one unit to another.
// Reuses existing PerformAttack logic (d20 roll, armor, etc.)
func calculateUnitDamage(attacker, defender *ecs.Entity) int {
	attackerAttr := common.GetAttributes(attacker)
	defenderAttr := common.GetAttributes(defender)

	// Base damage (simplified - adapt to existing weapon system)
	baseDamage := attackerAttr.AttackBonus + attackerAttr.DamageBonus

	// d20 variance (reuse existing logic)
	roll := randgen.GetDiceRoll(20)
	if roll >= 18 {
		baseDamage = int(float64(baseDamage) * 1.5) // Critical
	} else if roll <= 3 {
		baseDamage = baseDamage / 2 // Weak hit
	}

	// Apply role modifiers
	attackerRole := getRoleFromUnit(attacker)
	baseDamage = applyRoleModifier(baseDamage, attackerRole)

	// Apply defense
	totalDamage := baseDamage - defenderAttr.TotalProtection
	if totalDamage < 1 {
		totalDamage = 1 // Minimum damage
	}

	return totalDamage
}

// applyRoleModifier adjusts damage based on unit role.
func applyRoleModifier(damage int, role UnitRole) int {
	switch role {
	case ROLE_TANK:
		return int(float64(damage) * 0.8) // -20% (tanks don't deal high damage)
	case ROLE_DPS:
		return int(float64(damage) * 1.3) // +30% (damage dealers)
	case ROLE_SUPPORT:
		return int(float64(damage) * 0.6) // -40% (support units are weak attackers)
	default:
		return damage
	}
}

// applyDamageToUnit reduces HP and tracks kills.
func applyDamageToUnit(unit *ecs.Entity, damage int, result *CombatResult) {
	attr := common.GetAttributes(unit)
	attr.CurrentHealth -= damage
	result.DamageByUnit[unit] = damage

	if attr.CurrentHealth <= 0 {
		result.UnitsKilled = append(result.UnitsKilled, unit)
	}
}

// selectLowestHPTarget picks the unit with the lowest HP (tactical targeting).
func selectLowestHPTarget(units []*ecs.Entity) *ecs.Entity {
	lowest := units[0]
	lowestHP := common.GetAttributes(lowest).CurrentHealth

	for _, u := range units[1:] {
		hp := common.GetAttributes(u).CurrentHealth
		if hp < lowestHP {
			lowest = u
			lowestHP = hp
		}
	}

	return lowest
}

// selectRandomTargets randomly picks N targets from the list.
func selectRandomTargets(units []*ecs.Entity, count int) []*ecs.Entity {
	if count >= len(units) {
		return units
	}

	// Shuffle and take first N
	shuffled := make([]*ecs.Entity, len(units))
	copy(shuffled, units)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}

// getRoleFromUnit extracts the role from a unit entity.
func getRoleFromUnit(unit *ecs.Entity) UnitRole {
	if unit.HasComponent(UnitRoleComponent) {
		roleData := common.GetComponentType[*UnitRoleData](unit, UnitRoleComponent)
		return roleData.Role
	}
	return ROLE_DPS // Default if no role assigned
}

// sumDamageMap totals all damage dealt.
func sumDamageMap(damageMap map[*ecs.Entity]int) int {
	total := 0
	for _, dmg := range damageMap {
		total += dmg
	}
	return total
}
```

### Grid Management for Targeting

**File:** `squad/grid.go`

```go
package squad

import (
	"game_main/common"
	"github.com/bytearena/ecs"
)

// SquadGrid manages the 3x3 internal grid for a squad.
type SquadGrid struct {
	Occupied [3][3]bool
	UnitMap  [3][3]*ecs.Entity
}

// BuildSquadGrid constructs a grid from squad data.
func BuildSquadGrid(squadData *SquadData) *SquadGrid {
	grid := &SquadGrid{}

	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}

		// Get grid position
		if !unit.HasComponent(GridPositionComponent) {
			continue
		}

		gridPos := common.GetComponentType[*GridPositionData](unit, GridPositionComponent)

		// Validate bounds
		if gridPos.Row < 0 || gridPos.Row > 2 || gridPos.Col < 0 || gridPos.Col > 2 {
			continue
		}

		// Place unit
		grid.Occupied[gridPos.Row][gridPos.Col] = true
		grid.UnitMap[gridPos.Row][gridPos.Col] = unit
	}

	return grid
}

// GetUnitsInRow returns all unique units in a given row (0-2).
func (g *SquadGrid) GetUnitsInRow(row int) []*ecs.Entity {
	if row < 0 || row > 2 {
		return []*ecs.Entity{}
	}

	// Use map to deduplicate (in case units span multiple cells)
	unitSet := make(map[*ecs.Entity]bool)

	for col := 0; col < 3; col++ {
		if g.UnitMap[row][col] != nil {
			unitSet[g.UnitMap[row][col]] = true
		}
	}

	// Convert to slice
	units := make([]*ecs.Entity, 0, len(unitSet))
	for u := range unitSet {
		// Check if alive
		attr := common.GetAttributes(u)
		if attr.CurrentHealth > 0 {
			units = append(units, u)
		}
	}

	return units
}

// RemoveUnit clears a unit from the grid.
func (g *SquadGrid) RemoveUnit(unit *ecs.Entity) {
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if g.UnitMap[r][c] == unit {
				g.Occupied[r][c] = false
				g.UnitMap[r][c] = nil
			}
		}
	}
}

// PlaceUnit adds a unit to the grid at the specified position.
// Returns false if the position is already occupied.
func (g *SquadGrid) PlaceUnit(unit *ecs.Entity, row, col int) bool {
	if row < 0 || row > 2 || col < 0 || col > 2 {
		return false // Out of bounds
	}

	if g.Occupied[row][col] {
		return false // Already occupied
	}

	g.Occupied[row][col] = true
	g.UnitMap[row][col] = unit

	return true
}

// GetUnitAt returns the unit at a specific grid position.
func (g *SquadGrid) GetUnitAt(row, col int) *ecs.Entity {
	if row < 0 || row > 2 || col < 0 || col > 2 {
		return nil
	}
	return g.UnitMap[row][col]
}

// IsEmpty checks if the squad grid has no units.
func (g *SquadGrid) IsEmpty() bool {
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if g.UnitMap[r][c] != nil {
				return false
			}
		}
	}
	return true
}
```

### Helper Functions

```go
// removeUnitFromSquad clears a unit from squad data and grid.
func removeUnitFromSquad(unit, squadEntity *ecs.Entity, squadData *SquadData, grid *SquadGrid) {
	// Remove from grid
	grid.RemoveUnit(unit)

	// Remove from squad entity list
	for i, u := range squadData.UnitEntities {
		if u == unit {
			squadData.UnitEntities[i] = nil
			break
		}
	}

	// Mark unit for removal (don't destroy immediately - let cleanup system handle it)
	// unit.Remove() // Uncomment if you want immediate destruction
}
```

---

## Automated Leader Abilities

### Design Overview

**Core Concept:** Leader abilities trigger automatically based on conditions (HP thresholds, turn counts, etc.). NO manual player input during combat.

**Trigger System:**
1. Before each squad's turn, check all ability conditions
2. If condition met and ability is off cooldown, trigger it automatically
3. Display ability activation message to player
4. Apply cooldown

**Condition Types:**
- `TRIGGER_SQUAD_HP_BELOW`: Squad average HP < threshold (e.g., 50%)
- `TRIGGER_TURN_COUNT`: Specific turn number (e.g., turn 1, turn 3)
- `TRIGGER_ENEMY_COUNT`: Number of enemy squads on map
- `TRIGGER_MORALE_BELOW`: Squad morale < threshold (future)
- `TRIGGER_COMBAT_START`: First turn of combat (always turn 1)

### Ability Definitions

**File:** `squad/abilities.go`

```go
package squad

import (
	"game_main/common"
	"fmt"
	"github.com/bytearena/ecs"
)

// AbilityEffect is the interface for all ability implementations.
type AbilityEffect interface {
	Execute(leaderEntity, targetSquad *ecs.Entity, ecsmanager *common.EntityManager) error
}

// AbilityDefinition is the template for an ability.
type AbilityDefinition struct {
	ID             string
	Name           string
	Description    string
	Cooldown       int
	Effect         AbilityEffect
}

// Global ability registry
var AbilityRegistry = map[string]*AbilityDefinition{}

// RegisterAbility adds an ability to the registry.
func RegisterAbility(def *AbilityDefinition) {
	AbilityRegistry[def.ID] = def
}

// InitAbilities registers all built-in abilities.
// Call this during game initialization.
func InitAbilities() {
	RegisterAbility(&AbilityDefinition{
		ID:          "Rally",
		Name:        "Rally",
		Description: "Boost squad damage by 5 for 3 turns",
		Cooldown:    5,
		Effect:      &RallyEffect{DamageBonus: 5, Duration: 3},
	})

	RegisterAbility(&AbilityDefinition{
		ID:          "Heal",
		Name:        "Healing Aura",
		Description: "Restore 10 HP to all units in squad",
		Cooldown:    4,
		Effect:      &HealEffect{HealAmount: 10},
	})

	RegisterAbility(&AbilityDefinition{
		ID:          "BattleCry",
		Name:        "Battle Cry",
		Description: "Boost squad morale and attack (turn 1 only)",
		Cooldown:    999, // Once per combat
		Effect:      &BattleCryEffect{DamageBonus: 3, MoraleBonus: 10},
	})

	RegisterAbility(&AbilityDefinition{
		ID:          "Fireball",
		Name:        "Fireball",
		Description: "Deal 15 damage to all units in target squad",
		Cooldown:    3,
		Effect:      &FireballEffect{BaseDamage: 15},
	})
}

// --- Ability Implementations ---

// RallyEffect: Temporary damage buff to own squad
type RallyEffect struct {
	DamageBonus int
	Duration    int // Turns (future: needs buff tracking system)
}

func (e *RallyEffect) Execute(leaderEntity, targetSquad *ecs.Entity, ecsmanager *common.EntityManager) error {
	// Get leader's squad
	memberData := common.GetComponentType[*SquadMemberData](leaderEntity, SquadMemberComponent)
	squadEntity := memberData.SquadEntity
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	// Apply buff to all units in squad
	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth <= 0 {
			continue
		}

		attr.DamageBonus += e.DamageBonus
		// TODO: Track buff duration (requires turn/buff system)
	}

	fmt.Printf("[ABILITY] %s rallies the squad! +%d damage for %d turns\n",
		squadData.Name, e.DamageBonus, e.Duration)

	return nil
}

// HealEffect: Restore HP to own squad
type HealEffect struct {
	HealAmount int
}

func (e *HealEffect) Execute(leaderEntity, targetSquad *ecs.Entity, ecsmanager *common.EntityManager) error {
	memberData := common.GetComponentType[*SquadMemberData](leaderEntity, SquadMemberComponent)
	squadEntity := memberData.SquadEntity
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	healed := 0
	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth <= 0 {
			continue
		}

		// Cap at max HP
		attr.CurrentHealth += e.HealAmount
		if attr.CurrentHealth > attr.MaxHealth {
			attr.CurrentHealth = attr.MaxHealth
		}
		healed++
	}

	fmt.Printf("[ABILITY] %s heals the squad! %d units restored %d HP\n",
		squadData.Name, healed, e.HealAmount)

	return nil
}

// BattleCryEffect: First turn buff (morale + damage)
type BattleCryEffect struct {
	DamageBonus int
	MoraleBonus int
}

func (e *BattleCryEffect) Execute(leaderEntity, targetSquad *ecs.Entity, ecsmanager *common.EntityManager) error {
	memberData := common.GetComponentType[*SquadMemberData](leaderEntity, SquadMemberComponent)
	squadEntity := memberData.SquadEntity
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	// Boost morale
	squadData.Morale += e.MoraleBonus

	// Boost damage
	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth > 0 {
			attr.DamageBonus += e.DamageBonus
		}
	}

	fmt.Printf("[ABILITY] %s lets out a mighty battle cry! Morale and damage increased!\n",
		squadData.Name)

	return nil
}

// FireballEffect: AOE damage to enemy squad
type FireballEffect struct {
	BaseDamage int
}

func (e *FireballEffect) Execute(leaderEntity, targetSquad *ecs.Entity, ecsmanager *common.EntityManager) error {
	if targetSquad == nil {
		return fmt.Errorf("no target squad for Fireball")
	}

	memberData := common.GetComponentType[*SquadMemberData](leaderEntity, SquadMemberComponent)
	casterSquadEntity := memberData.SquadEntity
	casterSquadData := common.GetComponentType[*SquadData](casterSquadEntity, SquadComponent)

	targetSquadData := common.GetComponentType[*SquadData](targetSquad, SquadComponent)

	killed := 0
	for _, unit := range targetSquadData.UnitEntities {
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth <= 0 {
			continue
		}

		attr.CurrentHealth -= e.BaseDamage
		if attr.CurrentHealth <= 0 {
			killed++
		}
	}

	fmt.Printf("[ABILITY] %s casts Fireball on %s! %d damage dealt, %d units killed\n",
		casterSquadData.Name, targetSquadData.Name, e.BaseDamage, killed)

	return nil
}
```

### Condition Checking System

**File:** `squad/conditions.go`

```go
package squad

import (
	"game_main/common"
	"github.com/bytearena/ecs"
)

// CheckAndTriggerAbilities evaluates all leader abilities for automated triggers.
// Call this at the start of each squad's turn.
func CheckAndTriggerAbilities(squadEntity *ecs.Entity, ecsmanager *common.EntityManager) {
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	// Find leader in squad
	var leaderEntity *ecs.Entity
	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}

		if unit.HasComponent(LeaderComponent) {
			leaderEntity = unit
			break
		}
	}

	if leaderEntity == nil {
		return // No leader, no abilities
	}

	// Check if leader has condition component
	if !leaderEntity.HasComponent(AbilityConditionComponent) {
		return
	}

	conditionData := common.GetComponentType[*AbilityConditionData](leaderEntity, AbilityConditionComponent)
	leaderData := common.GetComponentType[*LeaderData](leaderEntity, LeaderComponent)

	// Evaluate each condition
	for i, condition := range conditionData.Conditions {
		if condition.HasTriggered {
			continue // Already triggered this combat
		}

		// Find ability slot
		abilitySlot := findAbilitySlot(leaderData, condition.AbilityID)
		if abilitySlot == nil {
			continue // Ability not equipped
		}

		// Check cooldown
		if abilitySlot.Cooldown > 0 {
			continue // Ability not ready
		}

		// Evaluate trigger condition
		triggered := evaluateTrigger(condition, squadEntity, ecsmanager)
		if !triggered {
			continue
		}

		// Execute ability
		targetSquad := selectTargetForAbility(condition.AbilityID, squadEntity, ecsmanager)
		executeAbility(leaderEntity, abilitySlot, targetSquad, ecsmanager)

		// Mark as triggered (for TRIGGER_COMBAT_START, etc.)
		conditionData.Conditions[i].HasTriggered = true
	}

	// Tick down cooldowns
	for _, ability := range leaderData.AbilitySlots {
		if ability != nil && ability.Cooldown > 0 {
			ability.Cooldown--
		}
	}
}

// evaluateTrigger checks if a condition is met.
func evaluateTrigger(condition AbilityCondition, squadEntity *ecs.Entity, ecsmanager *common.EntityManager) bool {
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	switch condition.TriggerType {
	case TRIGGER_SQUAD_HP_BELOW:
		avgHP := calculateAverageHP(squadData)
		return avgHP < condition.Threshold

	case TRIGGER_TURN_COUNT:
		return squadData.TurnCount == int(condition.Threshold)

	case TRIGGER_COMBAT_START:
		return squadData.TurnCount == 1

	case TRIGGER_ENEMY_COUNT:
		enemyCount := countEnemySquads(ecsmanager)
		return float64(enemyCount) >= condition.Threshold

	case TRIGGER_MORALE_BELOW:
		return float64(squadData.Morale) < condition.Threshold

	default:
		return false
	}
}

// calculateAverageHP computes the squad's average HP as a percentage (0.0 - 1.0).
func calculateAverageHP(squadData *SquadData) float64 {
	totalHP := 0
	totalMaxHP := 0
	count := 0

	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.MaxHealth == 0 {
			continue
		}

		totalHP += attr.CurrentHealth
		totalMaxHP += attr.MaxHealth
		count++
	}

	if totalMaxHP == 0 {
		return 0.0
	}

	return float64(totalHP) / float64(totalMaxHP)
}

// countEnemySquads counts the number of enemy squads on the map.
func countEnemySquads(ecsmanager *common.EntityManager) int {
	count := 0
	for _, result := range ecsmanager.World.Query(SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

		// Assume enemy squads don't have "Player" prefix (adjust based on your naming)
		if len(squadData.Name) > 0 && squadData.Name[0] != 'P' {
			count++
		}
	}
	return count
}

// findAbilitySlot finds the ability instance in leader's slots.
func findAbilitySlot(leaderData *LeaderData, abilityID string) *AbilityInstance {
	for _, slot := range leaderData.AbilitySlots {
		if slot != nil && slot.AbilityID == abilityID {
			return slot
		}
	}
	return nil
}

// selectTargetForAbility picks a target squad based on ability type.
func selectTargetForAbility(abilityID string, sourceSquad *ecs.Entity, ecsmanager *common.EntityManager) *ecs.Entity {
	abilityDef := AbilityRegistry[abilityID]
	if abilityDef == nil {
		return nil
	}

	// For offensive abilities (like Fireball), pick nearest enemy squad
	// For buffs/heals (like Rally, Heal), target own squad
	switch abilityID {
	case "Rally", "Heal", "BattleCry":
		return sourceSquad // Self-target

	case "Fireball":
		// Find nearest enemy squad (simplified: just find first enemy)
		for _, result := range ecsmanager.World.Query(SquadTag) {
			squadEntity := result.Entity
			if squadEntity == sourceSquad {
				continue // Skip self
			}
			return squadEntity // Return first enemy found
		}
		return nil

	default:
		return sourceSquad
	}
}

// executeAbility triggers the ability effect.
func executeAbility(leaderEntity *ecs.Entity, abilitySlot *AbilityInstance, targetSquad *ecs.Entity, ecsmanager *common.EntityManager) {
	abilityDef := AbilityRegistry[abilitySlot.AbilityID]
	if abilityDef == nil {
		return
	}

	// Execute effect
	err := abilityDef.Effect.Execute(leaderEntity, targetSquad, ecsmanager)
	if err != nil {
		fmt.Printf("[ERROR] Ability %s failed: %v\n", abilityDef.Name, err)
		return
	}

	// Set cooldown
	abilitySlot.Cooldown = abilitySlot.MaxCooldown
}
```

### Equipping Abilities with Conditions

```go
// EquipAbilityWithCondition adds an ability to a leader with an automated trigger.
func EquipAbilityWithCondition(leaderEntity *ecs.Entity, abilityID string, slot int, triggerType TriggerType, threshold float64) error {
	if slot < 0 || slot >= 4 {
		return fmt.Errorf("invalid slot %d", slot)
	}

	abilityDef := AbilityRegistry[abilityID]
	if abilityDef == nil {
		return fmt.Errorf("unknown ability: %s", abilityID)
	}

	// Add ability to slot
	leaderData := common.GetComponentType[*LeaderData](leaderEntity, LeaderComponent)
	leaderData.AbilitySlots[slot] = &AbilityInstance{
		AbilityID:   abilityID,
		Cooldown:    0,
		MaxCooldown: abilityDef.Cooldown,
	}

	// Add condition
	if !leaderEntity.HasComponent(AbilityConditionComponent) {
		leaderEntity.AddComponent(AbilityConditionComponent, &AbilityConditionData{
			Conditions: []AbilityCondition{},
		})
	}

	conditionData := common.GetComponentType[*AbilityConditionData](leaderEntity, AbilityConditionComponent)
	conditionData.Conditions = append(conditionData.Conditions, AbilityCondition{
		AbilityID:    abilityID,
		TriggerType:  triggerType,
		Threshold:    threshold,
		HasTriggered: false,
	})

	return nil
}
```

---

## Squad Composition Flexibility

### Design Goals

**Key Requirement:** Support experimentation like Nephilim, Symphony of War, Ogre Battle, Soul Nomad.

**Flexibility Features:**
1. Variable squad sizes (1-9 units)
2. No hard role requirements (all tanks, all DPS, etc. are valid)
3. Empty grid slots allowed (sparse formations)
4. Leader is optional but recommended
5. Formation templates for quick setup
6. Easy unit swapping/rearrangement

### Squad Creation API

**File:** `squad/creation.go`

```go
package squad

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"
	"github.com/bytearena/ecs"
)

// UnitTemplate defines a unit to be created in a squad.
type UnitTemplate struct {
	EntityType   entitytemplates.EntityType  // Creature, Player, etc.
	EntityConfig entitytemplates.EntityConfig
	EntityData   any                          // JSONMonster, etc.
	GridRow      int                          // 0-2
	GridCol      int                          // 0-2
	Role         UnitRole                     // Tank, DPS, Support
	TargetRows   []int                        // Which rows to attack
	IsMultiTarget bool                        // AOE or single-target
	MaxTargets   int                          // Max targets per row
	IsLeader     bool                         // Squad leader flag
}

// CreateSquadFromTemplate creates a squad entity with units.
func CreateSquadFromTemplate(
	ecsmanager *common.EntityManager,
	squadName string,
	formation FormationType,
	worldPos coords.LogicalPosition,
	unitTemplates []UnitTemplate,
) *ecs.Entity {

	// Create squad entity
	squadEntity := ecsmanager.World.NewEntity()
	squadEntity.AddComponent(SquadComponent, &SquadData{
		Name:         squadName,
		Formation:    formation,
		UnitEntities: make([]*ecs.Entity, 0, 9),
		Morale:       100,
		SquadLevel:   1,
		TurnCount:    0,
	})
	squadEntity.AddComponent(common.PositionComponent, &worldPos)

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	// Create units
	grid := &SquadGrid{}

	for _, template := range unitTemplates {
		// Validate grid position
		if template.GridRow < 0 || template.GridRow > 2 || template.GridCol < 0 || template.GridCol > 2 {
			fmt.Printf("Warning: Invalid grid position (%d, %d) for unit, skipping\n",
				template.GridRow, template.GridCol)
			continue
		}

		// Check if position occupied
		if grid.Occupied[template.GridRow][template.GridCol] {
			fmt.Printf("Warning: Grid position (%d, %d) already occupied, skipping\n",
				template.GridRow, template.GridCol)
			continue
		}

		// Create unit entity
		unitEntity := entitytemplates.CreateEntityFromTemplate(
			*ecsmanager,
			template.EntityConfig,
			template.EntityData,
		)

		// Add squad membership
		unitEntity.AddComponent(SquadMemberComponent, &SquadMemberData{
			SquadEntity: squadEntity,
			IsLeader:    template.IsLeader,
		})

		// Add grid position
		unitEntity.AddComponent(GridPositionComponent, &GridPositionData{
			Row: template.GridRow,
			Col: template.GridCol,
		})

		// Add role
		unitEntity.AddComponent(UnitRoleComponent, &UnitRoleData{
			Role: template.Role,
		})

		// Add targeting data
		unitEntity.AddComponent(TargetRowComponent, &TargetRowData{
			TargetRows:    template.TargetRows,
			IsMultiTarget: template.IsMultiTarget,
			MaxTargets:    template.MaxTargets,
		})

		// Add leader component if needed
		if template.IsLeader {
			unitEntity.AddComponent(LeaderComponent, &LeaderData{
				Leadership:   10,
				AbilitySlots: [4]*AbilityInstance{},
				Experience:   0,
			})
		}

		// Place in grid
		grid.PlaceUnit(unitEntity, template.GridRow, template.GridCol)

		// Add to squad
		squadData.UnitEntities = append(squadData.UnitEntities, unitEntity)
	}

	return squadEntity
}

// AddUnitToSquad adds a unit to an existing squad at a specific position.
// Returns error if position occupied or invalid.
func AddUnitToSquad(
	squadEntity *ecs.Entity,
	unitEntity *ecs.Entity,
	gridRow, gridCol int,
	role UnitRole,
	targetRows []int,
	isMultiTarget bool,
	maxTargets int,
) error {

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	grid := BuildSquadGrid(squadData)

	// Validate position
	if gridRow < 0 || gridRow > 2 || gridCol < 0 || gridCol > 2 {
		return fmt.Errorf("invalid grid position (%d, %d)", gridRow, gridCol)
	}

	if grid.Occupied[gridRow][gridCol] {
		return fmt.Errorf("grid position (%d, %d) already occupied", gridRow, gridCol)
	}

	// Add components
	unitEntity.AddComponent(SquadMemberComponent, &SquadMemberData{
		SquadEntity: squadEntity,
		IsLeader:    false, // Can be changed later
	})

	unitEntity.AddComponent(GridPositionComponent, &GridPositionData{
		Row: gridRow,
		Col: gridCol,
	})

	unitEntity.AddComponent(UnitRoleComponent, &UnitRoleData{
		Role: role,
	})

	unitEntity.AddComponent(TargetRowComponent, &TargetRowData{
		TargetRows:    targetRows,
		IsMultiTarget: isMultiTarget,
		MaxTargets:    maxTargets,
	})

	// Add to squad
	squadData.UnitEntities = append(squadData.UnitEntities, unitEntity)

	return nil
}

// RemoveUnitFromSquad removes a unit from its squad.
func RemoveUnitFromSquad(unitEntity *ecs.Entity) error {
	if !unitEntity.HasComponent(SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)
	squadEntity := memberData.SquadEntity
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	// Remove from squad's unit list
	for i, u := range squadData.UnitEntities {
		if u == unitEntity {
			squadData.UnitEntities = append(squadData.UnitEntities[:i], squadData.UnitEntities[i+1:]...)
			break
		}
	}

	// Remove squad-related components
	// Note: bytearena/ecs doesn't have RemoveComponent, so we leave them orphaned
	// Alternative: Track in SquadMemberData that unit is "inactive"
	memberData.SquadEntity = nil

	return nil
}

// MoveUnitInSquad changes a unit's grid position within the squad.
func MoveUnitInSquad(unitEntity *ecs.Entity, newRow, newCol int) error {
	if !unitEntity.HasComponent(SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)
	squadEntity := memberData.SquadEntity
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	grid := BuildSquadGrid(squadData)

	// Validate new position
	if newRow < 0 || newRow > 2 || newCol < 0 || newCol > 2 {
		return fmt.Errorf("invalid grid position (%d, %d)", newRow, newCol)
	}

	if grid.Occupied[newRow][newCol] {
		return fmt.Errorf("grid position (%d, %d) already occupied", newRow, newCol)
	}

	// Remove from old position
	grid.RemoveUnit(unitEntity)

	// Update component
	gridPosData := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent)
	gridPosData.Row = newRow
	gridPosData.Col = newCol

	// Place in new position
	grid.PlaceUnit(unitEntity, newRow, newCol)

	return nil
}
```

### Formation Presets

```go
// FormationPresets provides quick-start squad configurations.
var FormationPresets = map[FormationType][]struct {
	Row    int
	Col    int
	Role   UnitRole
	Target []int
}{
	FORMATION_BALANCED: {
		{Row: 0, Col: 0, Role: ROLE_TANK, Target: []int{0}},
		{Row: 0, Col: 2, Role: ROLE_TANK, Target: []int{0}},
		{Row: 1, Col: 1, Role: ROLE_SUPPORT, Target: []int{1}},
		{Row: 2, Col: 0, Role: ROLE_DPS, Target: []int{2}},
		{Row: 2, Col: 2, Role: ROLE_DPS, Target: []int{2}},
	},
	FORMATION_DEFENSIVE: {
		{Row: 0, Col: 0, Role: ROLE_TANK, Target: []int{0}},
		{Row: 0, Col: 1, Role: ROLE_TANK, Target: []int{0}},
		{Row: 0, Col: 2, Role: ROLE_TANK, Target: []int{0}},
		{Row: 1, Col: 1, Role: ROLE_SUPPORT, Target: []int{1}},
		{Row: 2, Col: 1, Role: ROLE_DPS, Target: []int{2}},
	},
	FORMATION_OFFENSIVE: {
		{Row: 0, Col: 1, Role: ROLE_TANK, Target: []int{0}},
		{Row: 1, Col: 0, Role: ROLE_DPS, Target: []int{1}},
		{Row: 1, Col: 1, Role: ROLE_DPS, Target: []int{1}},
		{Row: 1, Col: 2, Role: ROLE_DPS, Target: []int{1}},
		{Row: 2, Col: 1, Role: ROLE_SUPPORT, Target: []int{2}},
	},
	FORMATION_RANGED: {
		{Row: 0, Col: 1, Role: ROLE_TANK, Target: []int{0}},
		{Row: 1, Col: 0, Role: ROLE_DPS, Target: []int{1, 2}},
		{Row: 1, Col: 2, Role: ROLE_DPS, Target: []int{1, 2}},
		{Row: 2, Col: 0, Role: ROLE_DPS, Target: []int{2}},
		{Row: 2, Col: 1, Role: ROLE_SUPPORT, Target: []int{2}},
		{Row: 2, Col: 2, Role: ROLE_DPS, Target: []int{2}},
	},
}
```

---

## Implementation Phases

### Phase 1: Core Components and Data Structures (6-8 hours)

**Deliverables:**
- `squad/components.go` - All component definitions
- `squad/tags.go` - Tag initialization
- Component registration in game initialization

**Steps:**
1. Create `squad/` package directory
2. Define all component types and data structures
3. Create `InitSquadComponents()` function
4. Add component registration to `game_main/main.go`
5. Create `InitSquadTags()` and register tags
6. Build and verify no compilation errors

**Code Example:**
```go
// In game_main/main.go, add to initialization:
func initializeGame() {
	// ... existing initialization

	// Register squad components
	squad.InitSquadComponents(ecsmanager.World)
	squad.InitSquadTags()

	// ... rest of initialization
}
```

**Testing:**
- Build succeeds: `go build -o game_main/game_main.exe game_main/*.go`
- No runtime panics on startup

---

### Phase 2: Grid Management System (4-6 hours)

**Deliverables:**
- `squad/grid.go` - SquadGrid struct and methods
- Unit placement, removal, row queries

**Steps:**
1. Implement `SquadGrid` struct
2. Implement `BuildSquadGrid()` from SquadData
3. Implement `GetUnitsInRow()`, `PlaceUnit()`, `RemoveUnit()`
4. Add grid validation functions
5. Write unit tests (optional but recommended)

**Code Example:**
```go
// Test grid functionality in testing package
func TestSquadGrid() {
	manager := ecs.NewManager()
	squad.InitSquadComponents(manager)

	// Create squad
	squadEntity := manager.NewEntity()
	squadEntity.AddComponent(squad.SquadComponent, &squad.SquadData{
		Name:         "Test Squad",
		UnitEntities: []*ecs.Entity{},
	})

	// Create unit
	unitEntity := manager.NewEntity()
	unitEntity.AddComponent(squad.GridPositionComponent, &squad.GridPositionData{
		Row: 0,
		Col: 0,
	})

	// Build grid and test
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)
	squadData.UnitEntities = append(squadData.UnitEntities, unitEntity)

	grid := squad.BuildSquadGrid(squadData)

	// Verify unit in row 0
	units := grid.GetUnitsInRow(0)
	fmt.Printf("Units in row 0: %d\n", len(units)) // Should be 1
}
```

**Testing:**
- Grid correctly tracks unit positions
- `GetUnitsInRow()` returns correct units
- Boundary validation works (row/col 0-2)

---

### Phase 3: Row-Based Combat System (8-10 hours)

**Deliverables:**
- `squad/combat.go` - ExecuteSquadAttack, damage calculation
- Row-based targeting logic
- Integration with existing `PerformAttack()` concepts

**Steps:**
1. Implement `ExecuteSquadAttack()` function
2. Implement `calculateUnitDamage()` (adapt existing combat logic)
3. Implement `applyRoleModifier()` for Tank/DPS/Support
4. Implement targeting logic (single-target vs multi-target)
5. Implement helper functions (selectLowestHPTarget, selectRandomTargets)
6. Add death tracking and unit removal

**Code Example:**
```go
// Example usage in combat controller
func handleSquadCombat(attackerSquad, defenderSquad *ecs.Entity, ecsmanager *common.EntityManager) {
	// Execute attack
	result := squad.ExecuteSquadAttack(attackerSquad, defenderSquad, ecsmanager)

	fmt.Printf("Attack dealt %d total damage\n", result.TotalDamage)
	fmt.Printf("Units killed: %d\n", len(result.UnitsKilled))

	// Counter-attack if defender still alive
	if !isSquadDestroyed(defenderSquad) {
		counterResult := squad.ExecuteSquadAttack(defenderSquad, attackerSquad, ecsmanager)
		fmt.Printf("Counter-attack dealt %d damage\n", counterResult.TotalDamage)
	}
}

func isSquadDestroyed(squadEntity *ecs.Entity) bool {
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}
		attr := common.GetAttributes(unit)
		if attr.CurrentHealth > 0 {
			return false
		}
	}

	return true
}
```

**Testing:**
- Create two squads with different roles
- Execute combat and verify damage distribution
- Verify row targeting (front row hit first, back row protected)
- Verify role modifiers apply correctly
- Test edge cases (empty rows, all dead, etc.)

---

### Phase 4: Automated Ability System (6-8 hours)

**Deliverables:**
- `squad/abilities.go` - Ability definitions and effects
- `squad/conditions.go` - Condition checking and triggering
- Built-in abilities: Rally, Heal, BattleCry, Fireball

**Steps:**
1. Define `AbilityEffect` interface
2. Implement 4 example abilities (Rally, Heal, BattleCry, Fireball)
3. Create `AbilityRegistry` and `InitAbilities()`
4. Implement `CheckAndTriggerAbilities()`
5. Implement condition evaluation functions
6. Implement `EquipAbilityWithCondition()`
7. Add cooldown tick system

**Code Example:**
```go
// Equip leader with abilities
func setupLeaderAbilities(leaderEntity *ecs.Entity) {
	// BattleCry on turn 1
	squad.EquipAbilityWithCondition(
		leaderEntity,
		"BattleCry",
		0, // Slot 0
		squad.TRIGGER_COMBAT_START,
		0, // No threshold
	)

	// Heal when squad HP below 50%
	squad.EquipAbilityWithCondition(
		leaderEntity,
		"Heal",
		1, // Slot 1
		squad.TRIGGER_SQUAD_HP_BELOW,
		0.5, // 50%
	)

	// Rally on turn 3
	squad.EquipAbilityWithCondition(
		leaderEntity,
		"Rally",
		2, // Slot 2
		squad.TRIGGER_TURN_COUNT,
		3.0, // Turn 3
	)
}

// In combat turn processing:
func processSquadTurn(squadEntity *ecs.Entity, ecsmanager *common.EntityManager) {
	// Increment turn count
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)
	squadData.TurnCount++

	// Check and trigger abilities
	squad.CheckAndTriggerAbilities(squadEntity, ecsmanager)

	// ... rest of turn logic
}
```

**Testing:**
- Equip abilities with different trigger conditions
- Simulate combat and verify triggers fire correctly
- Test cooldown system
- Verify condition thresholds (HP < 50%, turn count, etc.)

---

### Phase 5: Squad Creation and Management (4-6 hours)

**Deliverables:**
- `squad/creation.go` - CreateSquadFromTemplate, AddUnitToSquad, etc.
- Formation presets
- Squad manipulation functions

**Steps:**
1. Implement `CreateSquadFromTemplate()`
2. Implement `AddUnitToSquad()`
3. Implement `RemoveUnitFromSquad()`
4. Implement `MoveUnitInSquad()`
5. Define `FormationPresets` map
6. Create helper functions for squad validation

**Code Example:**
```go
// Create a player squad with 5 units
func createPlayerSquad(ecsmanager *common.EntityManager) *ecs.Entity {
	templates := []squad.UnitTemplate{
		// Leader (Tank, front center)
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{Name: "Knight Leader"},
			EntityData:   createWarriorData(), // Your JSON data
			GridRow:      0,
			GridCol:      1,
			Role:         squad.ROLE_TANK,
			TargetRows:   []int{0},
			IsMultiTarget: false,
			MaxTargets:   1,
			IsLeader:     true,
		},
		// Tank (front left)
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{Name: "Shield Warrior"},
			EntityData:   createWarriorData(),
			GridRow:      0,
			GridCol:      0,
			Role:         squad.ROLE_TANK,
			TargetRows:   []int{0},
			IsMultiTarget: false,
			MaxTargets:   1,
			IsLeader:     false,
		},
		// DPS (back left)
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{Name: "Archer"},
			EntityData:   createArcherData(),
			GridRow:      2,
			GridCol:      0,
			Role:         squad.ROLE_DPS,
			TargetRows:   []int{2}, // Target back row
			IsMultiTarget: false,
			MaxTargets:   1,
			IsLeader:     false,
		},
		// Support (mid right)
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{Name: "Cleric"},
			EntityData:   createClericData(),
			GridRow:      1,
			GridCol:      2,
			Role:         squad.ROLE_SUPPORT,
			TargetRows:   []int{1},
			IsMultiTarget: false,
			MaxTargets:   1,
			IsLeader:     false,
		},
		// DPS Mage (back right, AOE)
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{Name: "Fire Mage"},
			EntityData:   createMageData(),
			GridRow:      2,
			GridCol:      2,
			Role:         squad.ROLE_DPS,
			TargetRows:   []int{2}, // Target back row
			IsMultiTarget: true,    // Hits all in row
			MaxTargets:   0,        // Unlimited
			IsLeader:     false,
		},
	}

	squadEntity := squad.CreateSquadFromTemplate(
		ecsmanager,
		"Player Squad",
		squad.FORMATION_BALANCED,
		coords.LogicalPosition{X: 5, Y: 5}, // World position
		templates,
	)

	// Setup leader abilities
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)
	for _, unit := range squadData.UnitEntities {
		if unit != nil && unit.HasComponent(squad.LeaderComponent) {
			setupLeaderAbilities(unit)
			break
		}
	}

	return squadEntity
}
```

**Testing:**
- Create squads with varying sizes (1-9 units)
- Test with sparse formations (empty grid slots)
- Verify leader assignment
- Test formation presets

---

### Phase 6: Integration with Existing Systems (8-10 hours)

**Deliverables:**
- Updated `input/combatcontroller.go` for squad selection
- Updated `spawning/spawnmonsters.go` for squad spawning
- Updated rendering to show squad grid
- Deprecated individual entity combat (optional)

**Steps:**
1. Modify `CombatController.HandleClick()` to detect squad entities
2. Implement squad selection/targeting flow
3. Update spawning system to create enemy squads
4. Add squad rendering with grid overlay
5. Integrate `CheckAndTriggerAbilities()` into turn system
6. Update input handling for squad movement (on map)
7. Add squad destruction cleanup

**Code Example: Combat Controller**
```go
// File: input/combatcontroller.go

package input

import (
	"game_main/common"
	"game_main/coords"
	"game_main/squad"
	"github.com/bytearena/ecs"
)

type CombatController struct {
	ecsmanager    *common.EntityManager
	selectedSquad *ecs.Entity // Currently selected squad
}

func (c *CombatController) HandleClick(worldPos coords.LogicalPosition) {
	clickedEntity := c.getEntityAtPosition(worldPos)

	if clickedEntity == nil {
		c.selectedSquad = nil
		return
	}

	// Check if clicked entity is a squad
	if clickedEntity.HasComponent(squad.SquadComponent) {
		if c.selectedSquad == nil {
			// Select attacker squad
			c.selectedSquad = clickedEntity
			c.highlightSquad(clickedEntity)
		} else if c.selectedSquad != clickedEntity {
			// Target different squad - execute combat
			c.executeSquadCombat(c.selectedSquad, clickedEntity)
			c.selectedSquad = nil
		} else {
			// Clicked same squad - deselect
			c.selectedSquad = nil
		}
		return
	}

	// Fall back to individual entity combat (if not removed)
	// ... existing combat logic
}

func (c *CombatController) executeSquadCombat(attacker, defender *ecs.Entity) {
	// Trigger abilities before combat
	squad.CheckAndTriggerAbilities(attacker, c.ecsmanager)

	// Execute attack
	result := squad.ExecuteSquadAttack(attacker, defender, c.ecsmanager)

	// Display results
	c.showCombatResults(result)

	// Counter-attack if defender still alive
	if !c.isSquadDestroyed(defender) {
		// Trigger defender abilities
		squad.CheckAndTriggerAbilities(defender, c.ecsmanager)

		counterResult := squad.ExecuteSquadAttack(defender, attacker, c.ecsmanager)
		c.showCombatResults(counterResult)
	}

	// Cleanup destroyed squads
	c.checkSquadDestruction(attacker)
	c.checkSquadDestruction(defender)
}

func (c *CombatController) isSquadDestroyed(squadEntity *ecs.Entity) bool {
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}
		attr := common.GetAttributes(unit)
		if attr.CurrentHealth > 0 {
			return false
		}
	}

	return true
}

func (c *CombatController) checkSquadDestruction(squadEntity *ecs.Entity) {
	if c.isSquadDestroyed(squadEntity) {
		// Remove all unit entities
		squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)
		for _, unit := range squadData.UnitEntities {
			if unit != nil {
				unit.Remove()
			}
		}

		// Remove squad entity
		squadEntity.Remove()
	}
}

func (c *CombatController) getEntityAtPosition(pos coords.LogicalPosition) *ecs.Entity {
	// Check squads first
	for _, result := range c.ecsmanager.World.Query(squad.SquadTag) {
		squadEntity := result.Entity
		squadPos := common.GetPosition(squadEntity)
		if squadPos.IsEqual(&pos) {
			return squadEntity
		}
	}

	// Fall back to individual entities
	return common.GetCreatureAtPosition(c.ecsmanager, &pos)
}
```

**Code Example: Spawning**
```go
// File: spawning/spawnmonsters.go

package spawning

import (
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"
	"game_main/squad"
	"math/rand"
)

// SpawnEnemySquad creates an enemy squad based on level.
func SpawnEnemySquad(ecsmanager *common.EntityManager, level int, worldPos coords.LogicalPosition) *ecs.Entity {
	// Determine squad size and composition based on level
	var templates []squad.UnitTemplate

	if level <= 3 {
		// Early game: 3-5 weak units
		templates = []squad.UnitTemplate{
			{
				EntityType:   entitytemplates.EntityCreature,
				EntityConfig: entitytemplates.EntityConfig{Name: "Goblin"},
				EntityData:   loadMonsterData("Goblin"),
				GridRow:      0, GridCol: 0,
				Role:         squad.ROLE_TANK,
				TargetRows:   []int{0},
				IsMultiTarget: false,
				MaxTargets:   1,
			},
			{
				EntityType:   entitytemplates.EntityCreature,
				EntityConfig: entitytemplates.EntityConfig{Name: "Goblin Archer"},
				EntityData:   loadMonsterData("GoblinArcher"),
				GridRow:      2, GridCol: 1,
				Role:         squad.ROLE_DPS,
				TargetRows:   []int{2},
				IsMultiTarget: false,
				MaxTargets:   1,
			},
			{
				EntityType:   entitytemplates.EntityCreature,
				EntityConfig: entitytemplates.EntityConfig{Name: "Goblin Shaman"},
				EntityData:   loadMonsterData("GoblinShaman"),
				GridRow:      1, GridCol: 2,
				Role:         squad.ROLE_SUPPORT,
				TargetRows:   []int{1},
				IsMultiTarget: true,
				MaxTargets:   2,
			},
		}
	} else if level <= 7 {
		// Mid game: 5-7 units with better roles
		templates = []squad.UnitTemplate{
			{
				EntityType:   entitytemplates.EntityCreature,
				EntityConfig: entitytemplates.EntityConfig{Name: "Orc Warrior"},
				EntityData:   loadMonsterData("Orc"),
				GridRow:      0, GridCol: 1,
				Role:         squad.ROLE_TANK,
				TargetRows:   []int{0},
				IsMultiTarget: false,
				MaxTargets:   1,
				IsLeader:     true, // Add leader
			},
			{
				EntityType:   entitytemplates.EntityCreature,
				EntityConfig: entitytemplates.EntityConfig{Name: "Orc Grunt"},
				EntityData:   loadMonsterData("Orc"),
				GridRow:      0, GridCol: 0,
				Role:         squad.ROLE_TANK,
				TargetRows:   []int{0},
			},
			{
				EntityType:   entitytemplates.EntityCreature,
				EntityConfig: entitytemplates.EntityConfig{Name: "Orc Berserker"},
				EntityData:   loadMonsterData("OrcBerserker"),
				GridRow:      1, GridCol: 1,
				Role:         squad.ROLE_DPS,
				TargetRows:   []int{0, 1},
				IsMultiTarget: true,
				MaxTargets:   2,
			},
			// ... more units
		}
	} else {
		// Late game: Full 9-unit squads with synergies
		// ... build late-game composition
	}

	// Create squad
	squadEntity := squad.CreateSquadFromTemplate(
		ecsmanager,
		"Enemy Squad",
		squad.FORMATION_BALANCED,
		worldPos,
		templates,
	)

	// Equip leader abilities if present
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)
	for _, unit := range squadData.UnitEntities {
		if unit != nil && unit.HasComponent(squad.LeaderComponent) {
			equipEnemyLeaderAbilities(unit)
			break
		}
	}

	return squadEntity
}

func equipEnemyLeaderAbilities(leaderEntity *ecs.Entity) {
	// Random ability selection for enemy leaders
	abilities := []string{"Rally", "Heal", "BattleCry", "Fireball"}

	// Equip 2 random abilities
	for i := 0; i < 2; i++ {
		abilityID := abilities[rand.Intn(len(abilities))]

		// Randomize trigger
		triggerType := squad.TRIGGER_SQUAD_HP_BELOW
		threshold := 0.5

		if rand.Float32() < 0.3 {
			triggerType = squad.TRIGGER_COMBAT_START
			threshold = 0
		}

		squad.EquipAbilityWithCondition(leaderEntity, abilityID, i, triggerType, threshold)
	}
}
```

**Testing:**
- Click to select and target squads
- Verify combat resolves correctly
- Spawned enemy squads appear on map
- Squad grid renders when selected
- Abilities trigger during combat
- Dead squads are removed from map

---

## Code Examples

### Example 1: Creating a Full Player Squad

```go
func InitializePlayerSquad(ecsmanager *common.EntityManager) *ecs.Entity {
	templates := []squad.UnitTemplate{
		// Row 0: Front line tanks
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Knight Captain",
				ImagePath: "knight.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: entitytemplates.JSONMonster{
				Name: "Knight Captain",
				Attributes: entitytemplates.JSONAttributes{
					MaxHealth:   50,
					AttackBonus: 5,
					BaseArmorClass: 15,
					BaseProtection: 5,
					BaseDodgeChance: 10.0,
					BaseMovementSpeed: 5,
				},
			},
			GridRow:      0,
			GridCol:      1,
			Role:         squad.ROLE_TANK,
			TargetRows:   []int{0}, // Attack front row
			IsMultiTarget: false,
			MaxTargets:   1,
			IsLeader:     true,
		},
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Shield Bearer",
				ImagePath: "shield.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: createTankData(40, 4, 16, 6),
			GridRow:      0,
			GridCol:      0,
			Role:         squad.ROLE_TANK,
			TargetRows:   []int{0},
			IsMultiTarget: false,
			MaxTargets:   1,
		},
		// Row 1: Mid-line DPS and support
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Battle Cleric",
				ImagePath: "cleric.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: createSupportData(35, 3, 12, 2),
			GridRow:      1,
			GridCol:      1,
			Role:         squad.ROLE_SUPPORT,
			TargetRows:   []int{1},
			IsMultiTarget: false,
			MaxTargets:   1,
		},
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Spearman",
				ImagePath: "spear.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: createDPSData(30, 8, 13, 3),
			GridRow:      1,
			GridCol:      0,
			Role:         squad.ROLE_DPS,
			TargetRows:   []int{0, 1}, // Can hit front and mid
			IsMultiTarget: false,
			MaxTargets:   1,
		},
		// Row 2: Back-line ranged DPS
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Longbowman",
				ImagePath: "archer.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: createDPSData(25, 10, 11, 2),
			GridRow:      2,
			GridCol:      0,
			Role:         squad.ROLE_DPS,
			TargetRows:   []int{2}, // Snipe back row
			IsMultiTarget: false,
			MaxTargets:   1,
		},
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Fire Mage",
				ImagePath: "mage.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: createDPSData(20, 12, 10, 1),
			GridRow:      2,
			GridCol:      2,
			Role:         squad.ROLE_DPS,
			TargetRows:   []int{2}, // AOE back row
			IsMultiTarget: true,    // Hits all units in row
			MaxTargets:   0,        // Unlimited
		},
	}

	squadEntity := squad.CreateSquadFromTemplate(
		ecsmanager,
		"Player's Legion",
		squad.FORMATION_BALANCED,
		coords.LogicalPosition{X: 10, Y: 10},
		templates,
	)

	// Setup leader abilities
	setupPlayerLeaderAbilities(squadEntity)

	return squadEntity
}

func setupPlayerLeaderAbilities(squadEntity *ecs.Entity) {
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

	// Find leader
	var leaderEntity *ecs.Entity
	for _, unit := range squadData.UnitEntities {
		if unit != nil && unit.HasComponent(squad.LeaderComponent) {
			leaderEntity = unit
			break
		}
	}

	if leaderEntity == nil {
		return
	}

	// Equip abilities with automated triggers

	// Battle Cry: Triggers at combat start (turn 1)
	squad.EquipAbilityWithCondition(
		leaderEntity,
		"BattleCry",
		0,
		squad.TRIGGER_COMBAT_START,
		0,
	)

	// Heal: Triggers when squad HP drops below 50%
	squad.EquipAbilityWithCondition(
		leaderEntity,
		"Heal",
		1,
		squad.TRIGGER_SQUAD_HP_BELOW,
		0.5,
	)

	// Rally: Triggers on turn 3
	squad.EquipAbilityWithCondition(
		leaderEntity,
		"Rally",
		2,
		squad.TRIGGER_TURN_COUNT,
		3.0,
	)

	// Fireball: Triggers when 2+ enemy squads present
	squad.EquipAbilityWithCondition(
		leaderEntity,
		"Fireball",
		3,
		squad.TRIGGER_ENEMY_COUNT,
		2.0,
	)
}

// Helper function to create JSON data
func createTankData(hp, atk, ac, prot int) entitytemplates.JSONMonster {
	return entitytemplates.JSONMonster{
		Name: "Tank",
		Attributes: entitytemplates.JSONAttributes{
			MaxHealth:         hp,
			AttackBonus:       atk,
			BaseArmorClass:    ac,
			BaseProtection:    prot,
			BaseDodgeChance:   15.0,
			BaseMovementSpeed: 4,
		},
	}
}

func createDPSData(hp, atk, ac, prot int) entitytemplates.JSONMonster {
	return entitytemplates.JSONMonster{
		Name: "DPS",
		Attributes: entitytemplates.JSONAttributes{
			MaxHealth:         hp,
			AttackBonus:       atk,
			BaseArmorClass:    ac,
			BaseProtection:    prot,
			BaseDodgeChance:   20.0,
			BaseMovementSpeed: 6,
		},
	}
}

func createSupportData(hp, atk, ac, prot int) entitytemplates.JSONMonster {
	return entitytemplates.JSONMonster{
		Name: "Support",
		Attributes: entitytemplates.JSONAttributes{
			MaxHealth:         hp,
			AttackBonus:       atk,
			BaseArmorClass:    ac,
			BaseProtection:    prot,
			BaseDodgeChance:   25.0,
			BaseMovementSpeed: 5,
		},
	}
}
```

### Example 2: Combat Simulation

```go
func TestCombatScenario() {
	// Initialize ECS
	manager := ecs.NewManager()
	ecsmanager := &common.EntityManager{
		World:     manager,
		WorldTags: make(map[string]ecs.Tag),
	}

	// Register components
	common.PositionComponent = manager.NewComponent()
	common.AttributeComponent = manager.NewComponent()
	common.NameComponent = manager.NewComponent()
	squad.InitSquadComponents(manager)
	squad.InitSquadTags()
	squad.InitAbilities()

	// Create two squads
	playerSquad := InitializePlayerSquad(ecsmanager)
	enemySquad := SpawnEnemySquad(ecsmanager, 5, coords.LogicalPosition{X: 15, Y: 15})

	// Simulate 5 turns of combat
	for turn := 1; turn <= 5; turn++ {
		fmt.Printf("\n--- TURN %d ---\n", turn)

		// Player squad attacks
		fmt.Println("Player squad attacks:")
		result := squad.ExecuteSquadAttack(playerSquad, enemySquad, ecsmanager)
		displayCombatResult(result)

		// Check if enemy destroyed
		if isSquadDestroyed(enemySquad) {
			fmt.Println("Enemy squad destroyed!")
			break
		}

		// Enemy counter-attacks
		fmt.Println("Enemy squad counter-attacks:")
		counterResult := squad.ExecuteSquadAttack(enemySquad, playerSquad, ecsmanager)
		displayCombatResult(counterResult)

		// Check if player destroyed
		if isSquadDestroyed(playerSquad) {
			fmt.Println("Player squad destroyed!")
			break
		}

		// Display squad status
		displaySquadStatus(playerSquad)
		displaySquadStatus(enemySquad)
	}
}

func displayCombatResult(result *squad.CombatResult) {
	fmt.Printf("  Total damage: %d\n", result.TotalDamage)
	fmt.Printf("  Units killed: %d\n", len(result.UnitsKilled))
	for unit, dmg := range result.DamageByUnit {
		name := common.GetComponentType[*common.Name](unit, common.NameComponent)
		fmt.Printf("    %s took %d damage\n", name.NameStr, dmg)
	}
}

func displaySquadStatus(squadEntity *ecs.Entity) {
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)
	fmt.Printf("\n%s Status:\n", squadData.Name)

	alive := 0
	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth > 0 {
			alive++
			name := common.GetComponentType[*common.Name](unit, common.NameComponent)
			fmt.Printf("  %s: %d/%d HP\n", name.NameStr, attr.CurrentHealth, attr.MaxHealth)
		}
	}

	fmt.Printf("Total alive: %d\n", alive)
}

func isSquadDestroyed(squadEntity *ecs.Entity) bool {
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}
		attr := common.GetAttributes(unit)
		if attr.CurrentHealth > 0 {
			return false
		}
	}

	return true
}
```

### Example 3: Dynamic Squad Modification

```go
// Add a new unit to an existing squad mid-game
func RecruitUnitToSquad(squadEntity *ecs.Entity, unitName string, gridRow, gridCol int) error {
	// Create new unit entity
	unitEntity := entitytemplates.CreateEntityFromTemplate(
		ecsmanager,
		entitytemplates.EntityConfig{
			Type:      entitytemplates.EntityCreature,
			Name:      unitName,
			ImagePath: "recruit.png",
			AssetDir:  "../assets/",
			Visible:   true,
		},
		createDPSData(30, 7, 12, 3),
	)

	// Add to squad
	err := squad.AddUnitToSquad(
		squadEntity,
		unitEntity,
		gridRow,
		gridCol,
		squad.ROLE_DPS,
		[]int{1}, // Target middle row
		false,    // Single-target
		1,        // Max 1 target
	)

	if err != nil {
		return fmt.Errorf("failed to recruit unit: %v", err)
	}

	fmt.Printf("Recruited %s to squad at position (%d, %d)\n", unitName, gridRow, gridCol)
	return nil
}

// Swap unit positions within squad
func ReorganizeSquad(squadEntity *ecs.Entity) error {
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

	// Find a back-line DPS unit
	var dpsUnit *ecs.Entity
	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}

		roleData := common.GetComponentType[*squad.UnitRoleData](unit, squad.UnitRoleComponent)
		gridPos := common.GetComponentType[*squad.GridPositionData](unit, squad.GridPositionComponent)

		if roleData.Role == squad.ROLE_DPS && gridPos.Row == 2 {
			dpsUnit = unit
			break
		}
	}

	if dpsUnit == nil {
		return fmt.Errorf("no DPS unit found to move")
	}

	// Move to front line (tactical decision)
	err := squad.MoveUnitInSquad(dpsUnit, 0, 2) // Front right
	if err != nil {
		return fmt.Errorf("failed to move unit: %v", err)
	}

	fmt.Println("Moved DPS unit to front line for aggressive tactics")
	return nil
}
```

---

## Integration Points

### 1. Game Initialization (`game_main/main.go`)

```go
func main() {
	// ... existing initialization

	// Register squad components and tags
	squad.InitSquadComponents(ecsmanager.World)
	squad.InitSquadTags()
	squad.InitAbilities()

	// ... rest of game setup
}
```

### 2. Input System (`input/combatcontroller.go`)

**Changes:**
- Detect squad entities on click
- Implement squad selection/targeting flow
- Call `CheckAndTriggerAbilities()` before combat
- Handle squad vs individual entity combat

**See Phase 6 code example above.**

### 3. Spawning System (`spawning/spawnmonsters.go`)

**Changes:**
- Replace individual monster spawning with squad spawning
- Use level-based composition logic
- Equip enemy leaders with random abilities

**See Phase 6 code example above.**

### 4. Rendering System (`graphics/` or `rendering/`)

**Changes:**
- Render squad entity as single sprite on tactical map
- Show 3x3 grid overlay when squad is selected
- Render unit sprites within grid cells
- Add visual indicators for leader, roles, HP bars

**Example Rendering Logic:**
```go
// In rendering system update loop
func RenderSquads(screen *ebiten.Image, ecsmanager *common.EntityManager, selectedSquad *ecs.Entity) {
	for _, result := range ecsmanager.World.Query(squad.SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)
		pos := common.GetPosition(squadEntity)

		// Convert to pixel position
		pixelPos := coords.CoordManager.LogicalToPixel(*pos)

		// Draw squad icon/sprite
		drawSquadIcon(screen, pixelPos, squadData.Name)

		// If selected, draw internal grid
		if squadEntity == selectedSquad {
			drawSquadGrid(screen, squadEntity, pixelPos)
		}
	}
}

func drawSquadGrid(screen *ebiten.Image, squadEntity *ecs.Entity, basePixelPos coords.PixelPosition) {
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)
	cellSize := 16 // Pixels per grid cell

	// Draw grid outline
	gridWidth := cellSize * 3
	gridHeight := cellSize * 3
	// (Draw rectangle at basePixelPos with gridWidth x gridHeight)

	// Draw units in grid
	for _, unit := range squadData.UnitEntities {
		if unit == nil {
			continue
		}

		gridPos := common.GetComponentType[*squad.GridPositionData](unit, squad.GridPositionComponent)
		attr := common.GetAttributes(unit)

		if attr.CurrentHealth <= 0 {
			continue // Don't draw dead units
		}

		// Calculate pixel offset within grid
		offsetX := gridPos.Col * cellSize
		offsetY := gridPos.Row * cellSize

		unitPixelX := basePixelPos.X + offsetX
		unitPixelY := basePixelPos.Y + offsetY

		// Draw unit sprite
		renderable := common.GetComponentType[*rendering.Renderable](unit, rendering.RenderableComponent)
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(unitPixelX), float64(unitPixelY))
		screen.DrawImage(renderable.Image, opts)

		// Draw HP bar
		drawUnitHPBar(screen, unitPixelX, unitPixelY, attr.CurrentHealth, attr.MaxHealth)

		// Draw role indicator (color-coded border)
		roleData := common.GetComponentType[*squad.UnitRoleData](unit, squad.UnitRoleComponent)
		drawRoleBorder(screen, unitPixelX, unitPixelY, cellSize, roleData.Role)
	}
}

func drawRoleBorder(screen *ebiten.Image, x, y, size int, role squad.UnitRole) {
	var color color.Color
	switch role {
	case squad.ROLE_TANK:
		color = color.RGBA{0, 0, 255, 255} // Blue
	case squad.ROLE_DPS:
		color = color.RGBA{255, 0, 0, 255} // Red
	case squad.ROLE_SUPPORT:
		color = color.RGBA{0, 255, 0, 255} // Green
	default:
		color = color.RGBA{255, 255, 255, 255} // White
	}

	// Draw border (simplified)
	// ebitenutil.DrawRect(screen, float64(x), float64(y), float64(size), 2, color) // Top
	// ... draw all 4 sides
}
```

### 5. Turn System (If Exists)

**Changes:**
- Increment `SquadData.TurnCount` each turn
- Call `CheckAndTriggerAbilities()` at turn start
- Tick cooldowns at turn end

### 6. Save/Load System (Future)

**Hooks for Progression:**
- Save squad composition (unit IDs, positions, roles)
- Save leader abilities and cooldowns
- Save squad morale and turn count
- Reconstruct squads on load

---

## Testing Strategy

### Manual Testing

**Phase 1-2: Components and Grid**
1. Create squad entity with components
2. Add 3-5 units to squad
3. Verify grid placement (no overlaps, correct positions)
4. Query units by row (GetUnitsInRow)
5. Remove unit and verify grid updates

**Phase 3: Combat**
1. Create two squads (player vs enemy)
2. Execute attack with mixed roles (Tank, DPS, Support)
3. Verify damage distribution (Tanks hit first)
4. Verify role modifiers (DPS deals +30%, Support deals -40%)
5. Test single-target vs multi-target (AOE)
6. Test row targeting (front row, back row, all rows)
7. Verify unit deaths remove from squad

**Phase 4: Abilities**
1. Equip leader with 4 abilities
2. Trigger COMBAT_START ability on turn 1
3. Reduce squad HP below 50% and verify heal triggers
4. Test cooldown system (ability not usable until ready)
5. Test multiple conditions (turn count, HP threshold, enemy count)

**Phase 5: Squad Management**
1. Create squad with formation preset
2. Add new unit mid-game
3. Remove unit from squad
4. Move unit to different grid position
5. Create squads with varying sizes (1, 5, 9 units)
6. Test with empty grid slots (sparse formations)

**Phase 6: Integration**
1. Click to select squad on map
2. Click enemy squad to initiate combat
3. Verify abilities trigger during combat
4. Spawn enemy squads at different levels
5. Verify squad rendering with grid overlay
6. Test squad destruction (all units dead → remove squad)

### Automated Testing (Optional)

**Unit Tests:**
```go
// squad/grid_test.go
func TestSquadGridPlacement(t *testing.T) {
	grid := &SquadGrid{}

	manager := ecs.NewManager()
	unit := manager.NewEntity()

	// Test valid placement
	if !grid.PlaceUnit(unit, 0, 0) {
		t.Error("Failed to place unit at (0, 0)")
	}

	// Test duplicate placement
	if grid.PlaceUnit(unit, 0, 0) {
		t.Error("Allowed duplicate placement at (0, 0)")
	}

	// Test out of bounds
	if grid.PlaceUnit(unit, 3, 3) {
		t.Error("Allowed out of bounds placement")
	}
}

// squad/combat_test.go
func TestRoleModifiers(t *testing.T) {
	damage := 100

	tankDamage := applyRoleModifier(damage, squad.ROLE_TANK)
	if tankDamage != 80 {
		t.Errorf("Tank modifier incorrect: got %d, want 80", tankDamage)
	}

	dpsDamage := applyRoleModifier(damage, squad.ROLE_DPS)
	if dpsDamage != 130 {
		t.Errorf("DPS modifier incorrect: got %d, want 130", dpsDamage)
	}

	supportDamage := applyRoleModifier(damage, squad.ROLE_SUPPORT)
	if supportDamage != 60 {
		t.Errorf("Support modifier incorrect: got %d, want 60", supportDamage)
	}
}
```

### Balance Testing

**Scenario 1: Tank Wall vs Balanced Squad**
- Create 3-tank squad (all front row)
- Create balanced squad (1 tank, 2 DPS, 1 support)
- Run 10 combats and track win rate

**Scenario 2: Back-Line Sniper vs Front-Heavy**
- Create squad with 2 back-row archers (target row 2)
- Create front-heavy squad (3 tanks, 1 support in row 2)
- Verify archers can snipe support, bypassing tanks

**Scenario 3: AOE Effectiveness**
- Create squad with 1 AOE mage (hits all in back row)
- Create enemy with 3 units in back row
- Verify mage hits all 3 units in one attack

**Scenario 4: Ability Synergy**
- Leader uses Rally (damage boost) on turn 1
- Squad attacks with boosted damage
- Verify damage increase is applied correctly

### Edge Case Testing

1. **Empty Squad:** Squad with 0 units (should not crash, just do nothing)
2. **Single Unit Squad:** 1 unit vs 9 unit squad
3. **All Dead Units:** All units killed in one attack (overflow damage)
4. **Leader Dies:** Leader killed mid-combat (abilities stop triggering)
5. **Invalid Grid Positions:** Unit at (-1, 0) or (3, 3) (should reject)
6. **Targeting Empty Row:** Unit targets row with no enemies (should skip)
7. **Cooldown Edge:** Ability with 0 cooldown (usable every turn)
8. **Multiple Leaders:** Two units marked as IsLeader (should only trigger first)

---

## Open Questions / Decisions

### 1. Damage Formula Tuning

**Question:** What should the exact role modifiers be?

**Current Proposal:**
- Tank: -20% damage dealt (0.8x)
- DPS: +30% damage dealt (1.3x)
- Support: -40% damage dealt (0.6x)

**Decision Needed:**
- Are these percentages balanced?
- Should support deal ANY damage, or focus only on abilities?
- Should tanks take increased damage (e.g., -10% defense)?

**Recommendation:** Start with proposed values, playtest, and adjust based on combat logs.

---

### 2. Ability Trigger Specificity

**Question:** How should turn-based triggers work?

**Current Proposal:**
- `TRIGGER_TURN_COUNT` with threshold 3.0 = triggers ONLY on turn 3
- `TRIGGER_COMBAT_START` = triggers on turn 1 (one-time)

**Alternative:**
- `TRIGGER_EVERY_N_TURNS` = triggers every N turns (e.g., every 3 turns)
- `TRIGGER_AFTER_TURN_N` = triggers on turn N and every turn after

**Decision Needed:**
- Should abilities trigger once or repeatedly?
- Should we add "TRIGGER_EVERY_N_TURNS" type?

**Recommendation:** Start with one-time triggers. Add repeating triggers later if needed.

---

### 3. Squad Movement on Map

**Question:** How do squads move on the tactical map?

**Current Proposal:**
- Squad entity has `LogicalPosition` component (just like individual entities)
- Input controller moves entire squad as one unit
- Squad's position represents the squad's "center" or "leader" position

**Alternative:**
- Squad doesn't move directly - must move each unit individually
- Squad position is abstract (formation only matters internally)

**Decision Needed:**
- Are squads single units on the tactical map?
- Or do individual units move independently, forming squads only for combat?

**Recommendation:** Squads are single units on the map (simpler). Units only separate if you add a "break formation" feature later.

---

### 4. Grid Position Persistence

**Question:** Do units remember their grid positions outside combat?

**Current Proposal:**
- `GridPositionComponent` always exists on squad units
- Positions persist between combats
- Player can rearrange squad formation anytime via UI (future)

**Alternative:**
- Grid positions only assigned during combat initialization
- Units default to auto-placement based on role
- No persistence needed

**Decision Needed:**
- Is formation management a strategic layer (players customize)?
- Or is it automatic based on role?

**Recommendation:** Persist grid positions (enables strategic depth and experimentation).

---

### 5. Leader Ability Targeting

**Question:** How do offensive leader abilities (like Fireball) pick targets?

**Current Proposal:**
- Offensive abilities target nearest enemy squad
- Defensive abilities (Rally, Heal) target own squad
- Targeting is automatic (no player input)

**Alternative:**
- Abilities target based on condition context (e.g., target squad that triggered HP threshold)
- Random targeting
- Target weakest/strongest enemy squad

**Decision Needed:**
- Should targeting be smarter (AI-based)?
- Or simple (nearest enemy)?

**Recommendation:** Start simple (nearest enemy). Add smart targeting in Phase 4 refinement if needed.

---

### 6. Counter-Attack Mechanics

**Question:** Should defender squads always counter-attack?

**Current Proposal:**
- If defender squad survives, they automatically counter-attack
- Counter-attack uses same row-based targeting logic
- Abilities trigger for defender too

**Alternative:**
- Counter-attack based on "initiative" stat (faster squad attacks twice)
- Defender can choose to "defend" instead of counter (future AI decision)
- Counter-attack deals reduced damage (e.g., 50%)

**Decision Needed:**
- Always counter-attack?
- Or add initiative/speed system?

**Recommendation:** Always counter-attack for simplicity. Add initiative system later if combat feels too symmetric.

---

### 7. Progression Hooks

**Question:** Where should progression systems plug in?

**Current Proposal:**
- Leave empty hooks in LeaderData (Experience field)
- Leave empty hooks in SquadData (SquadLevel field)
- Don't implement progression now, but design components to support it

**Hooks to Add:**
- Unit experience (component on unit entity)
- Squad experience (component on squad entity)
- Ability unlock system (registry of locked abilities)
- Equipment upgrades (future)

**Decision Needed:**
- What progression features are planned?
- Should we add placeholder components now?

**Recommendation:** Add Experience and Level fields to components, but leave them unused. Implement progression in a separate phase after core combat works.

---

### 8. Multi-Squad Battles

**Question:** Can multiple squads fight simultaneously?

**Current Proposal:**
- Combat is always 1v1 (squad vs squad)
- Player selects one squad to attack one enemy squad

**Alternative:**
- Multiple squads can gang up on one enemy (3v1)
- Multiple squads fight simultaneously (3v3 battlefield)
- Squads can reinforce each other during combat

**Decision Needed:**
- Is combat always 1v1?
- Or do we need multi-squad battle logic?

**Recommendation:** Start with 1v1. Multi-squad battles are a major feature requiring additional systems (targeting priority, reinforcement, etc.). Add later if desired.

---

### 9. Visual Feedback Priority

**Question:** What visual effects are essential for Phase 6?

**Essential:**
- Squad icon on tactical map
- 3x3 grid overlay when selected
- Unit sprites in grid cells
- HP bars for units

**Nice-to-Have:**
- Role color-coding (borders)
- Leader crown/indicator
- Ability activation animations
- Damage numbers floating
- Death animations

**Decision Needed:**
- What's the MVP for visual feedback?

**Recommendation:** Implement essential features in Phase 6. Nice-to-have features can be added in polish phase.

---

### 10. Backward Compatibility

**Question:** Should individual entity combat still work?

**Current Proposal:**
- Keep existing `PerformAttack()` logic
- Input controller falls back to individual combat if clicked entity is not a squad
- Allows gradual transition (some entities are individuals, some are squads)

**Alternative:**
- Remove individual combat entirely
- Force all entities to be part of squads (even 1-unit squads)
- Cleaner codebase, but more migration work

**Decision Needed:**
- Keep individual combat as fallback?
- Or go all-in on squads?

**Recommendation:** Keep fallback during development for testing. Remove once all spawning uses squads.

---

## Complexity Estimate

### Total Implementation Time: 32-38 hours

**Breakdown:**

| Phase | Description | Hours |
|-------|-------------|-------|
| 1 | Core Components and Data Structures | 6-8 |
| 2 | Grid Management System | 4-6 |
| 3 | Row-Based Combat System | 8-10 |
| 4 | Automated Ability System | 6-8 |
| 5 | Squad Creation and Management | 4-6 |
| 6 | Integration with Existing Systems | 8-10 |
| **Testing** | Manual testing, balance tuning, edge cases | 4-6 |
| **Total** | | **40-54 hours** |

**Adjusted for Documentation/Debugging:** 32-38 hours of pure implementation (excluding documentation reading and debugging time).

### Risk Level: Medium-High

**High Risk Areas:**
1. **Combat Balance:** Damage formulas may need multiple iterations
2. **Ability Triggers:** Condition logic can be tricky (off-by-one errors)
3. **Grid Synchronization:** Keeping SquadData.UnitEntities in sync with GridPositionComponent
4. **Integration:** Input controller changes may break existing systems

**Mitigation:**
- Test combat balance early (Phase 3)
- Write unit tests for condition evaluation (Phase 4)
- Use helper functions to enforce grid consistency (Phase 2)
- Maintain fallback to individual combat during development (Phase 6)

### Lines of Code Estimate

| File | Estimated LOC |
|------|---------------|
| `squad/components.go` | 150-200 |
| `squad/tags.go` | 30-50 |
| `squad/grid.go` | 150-200 |
| `squad/combat.go` | 250-300 |
| `squad/abilities.go` | 200-250 |
| `squad/conditions.go` | 200-250 |
| `squad/creation.go` | 200-250 |
| Integration changes (input, spawning, rendering) | 300-400 |
| **Total New/Modified Code** | **1,480-1,900 LOC** |

---

## Summary

This implementation plan provides a complete roadmap for adding squad-based combat to your tactical roguelike, adapted specifically for bytearena/ecs patterns.

**Key Features Delivered:**
1. ✅ Full 3x3 grid formations with variable squad sizes (1-9 units)
2. ✅ Three distinct roles (Tank, DPS, Support) with mechanical impact
3. ✅ Row-based targeting system (front/mid/back row attacks)
4. ✅ Single-target vs multi-target (AOE) abilities
5. ✅ Automated leader abilities with condition-based triggers
6. ✅ Squad composition flexibility (experimentation encouraged)
7. ✅ Integration with existing combat, spawning, and input systems
8. ✅ Hooks for future progression systems
9. ✅ Backward compatibility during transition

**What This Plan Does NOT Include (Deferred):**
- ❌ Progression systems (experience, leveling, ability unlocks)
- ❌ Advanced AI (enemy squads use basic targeting)
- ❌ Unit tests (optional, recommended but not required)
- ❌ Multi-squad battles (3v3, gang-up mechanics)
- ❌ Formation templates UI (player can't customize formations in-game yet)

**Next Steps:**
1. Review this plan and answer the 10 open questions
2. Decide if you (user) will implement, or if agent should implement
3. Start with Phase 1 (components) to validate architecture
4. Test combat balance early (Phase 3) to avoid late rework
5. Iterate on ability triggers and conditions (Phase 4) based on playtesting

**Estimated Total Time:** 32-38 hours of focused implementation.

**File Location:** `C:\Users\Afromullet\Desktop\TinkerRogue\analysis\squad_combat_implementation_plan.md`

---

**End of Implementation Plan**
