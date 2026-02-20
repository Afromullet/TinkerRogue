// Package squads implements an Entity Component System (ECS) for squad-based
// tactical combat. It provides components for squad management, unit positioning
// in a 3x3 grid, role-based combat behavior, and leader abilities.
//
// The package is designed for turn-based tactical games where players and
// enemies control squads of units in formation-based combat.
package squads

import (
	"fmt"

	"github.com/bytearena/ecs"
)

// Global components
var (
	SquadComponent           *ecs.Component
	SquadMemberComponent     *ecs.Component
	GridPositionComponent    *ecs.Component
	UnitRoleComponent        *ecs.Component
	CoverComponent           *ecs.Component
	LeaderComponent          *ecs.Component
	TargetRowComponent       *ecs.Component
	AbilitySlotComponent     *ecs.Component
	CooldownTrackerComponent *ecs.Component
	AttackRangeComponent     *ecs.Component
	MovementSpeedComponent   *ecs.Component
	ExperienceComponent      *ecs.Component
	StatGrowthComponent      *ecs.Component
	UnitTypeComponent        *ecs.Component

	SquadTag       ecs.Tag
	SquadMemberTag ecs.Tag
	LeaderTag      ecs.Tag
)

// DefaultSquadCapacity is the base capacity for squads without a leader.
const DefaultSquadCapacity = 6

// ========================================
// SQUAD ENTITY COMPONENTS
// ========================================

// SquadData represents the squad entity's component data.
type SquadData struct {
	SquadID            ecs.EntityID  // Unique squad identifier (native entity ID)
	Formation          FormationType // Current formation layout
	Name               string        // Squad display name
	Morale             int           // Squad-wide morale (0-100)
	SquadLevel         int           // Average level for spawning
	TurnCount          int           // Number of turns this squad has taken
	MaxUnits           int           // Maximum squad size (typically 9)
	IsDeployed         bool          // true if squad is on the tactical map, false if in reserves
	GarrisonedAtNodeID ecs.EntityID  // 0 = not garrisoned, >0 = garrisoned at this node entity
}

// FormationType defines squad layout presets
type FormationType int

const (
	FormationBalanced  FormationType = iota // Mix of roles
	FormationDefensive                      // Tank-heavy
	FormationOffensive                      // DPS-focused
	FormationRanged                         // Back-line heavy
)

func (f FormationType) String() string {
	switch f {
	case FormationBalanced:
		return "Balanced"
	case FormationDefensive:
		return "Defensive"
	case FormationOffensive:
		return "Offensive"
	case FormationRanged:
		return "Ranged"
	default:
		return "Unknown"
	}
}

// ========================================
// UNIT ENTITY COMPONENTS
// ========================================

// SquadMemberData links a unit back to its parent squad.
type SquadMemberData struct {
	SquadID ecs.EntityID // Parent squad's entity ID
}

// GridPositionData represents a unit's position within the 3x3 grid.
// Pure data - systems query for units at specific positions
// Supports multi-cell units (e.g., 1x2, 2x2, 2x1, etc.)
type GridPositionData struct {
	AnchorRow int // Top-left row (0-2)
	AnchorCol int // Top-left col (0-2)
	Width     int // Number of columns occupied (1-3)
	Height    int // Number of rows occupied (1-3)
}

// GetOccupiedCells returns all grid cells this unit occupies
func (g *GridPositionData) GetOccupiedCells() [][2]int {
	var cells [][2]int
	for r := g.AnchorRow; r < g.AnchorRow+g.Height && r < 3; r++ {
		for c := g.AnchorCol; c < g.AnchorCol+g.Width && c < 3; c++ {
			cells = append(cells, [2]int{r, c})
		}
	}
	return cells
}

// OccupiesCell checks if this unit occupies a specific grid cell
func (g *GridPositionData) OccupiesCell(row, col int) bool {
	return row >= g.AnchorRow && row < g.AnchorRow+g.Height &&
		col >= g.AnchorCol && col < g.AnchorCol+g.Width
}

// GetRows returns all row indices this unit occupies
func (g *GridPositionData) GetRows() []int {
	var rows []int
	for r := g.AnchorRow; r < g.AnchorRow+g.Height && r < 3; r++ {
		rows = append(rows, r)
	}
	return rows
}

// UnitRoleData defines a unit's combat role
// Affects combat behavior and damage distribution
type UnitRoleData struct {
	Role UnitRole // Tank, DPS, or Support
}

type UnitRole int

const (
	RoleTank    UnitRole = iota // Takes hits first, high defense
	RoleDPS                     // High damage output
	RoleSupport                 // Buffs, heals, utility
	RoleError                   // This is an error
)

func (r UnitRole) String() string {
	switch r {
	case RoleTank:
		return "Tank"
	case RoleDPS:
		return "DPS"
	case RoleSupport:
		return "Support"
	default:
		return "Unknown"
	}
}

// CoverData defines how a unit provides defensive cover to units behind it
// Cover reduces incoming damage for units in protected columns/rows
type CoverData struct {
	CoverValue     float64 // Damage reduction percentage (0.0 to 1.0, e.g., 0.25 = 25% reduction)
	CoverRange     int     // How many rows behind can receive cover (1 = immediate row, 2 = two rows, etc.)
	RequiresActive bool    // If true, dead/stunned units don't provide cover (typically true)
}

// GetCoverBonus returns the cover value if the unit is active, 0 otherwise
// Systems should call this and check unit health/status before applying cover
func (c *CoverData) GetCoverBonus(isActive bool) float64 {
	if c.RequiresActive && !isActive {
		return 0.0
	}
	return c.CoverValue
}

// AttackRangeData defines the world-based attack range of a unit
// Range determines maximum distance between squads for unit to participate in combat
type AttackRangeData struct {
	Range int // World tiles (Melee=1, Ranged=3, Magic=4)
}

// MovementSpeedData defines a unit's movement speed on the world map
// Squad movement speed is the minimum of all its units' speeds
type MovementSpeedData struct {
	Speed int // Tiles per turn (typically 1-5)
}

// TargetRowData defines attack targeting based on unit type
// Component name kept for compatibility
type TargetRowData struct {
	AttackType  AttackType // MeleeRow, MeleeColumn, Ranged, or Magic
	TargetCells [][2]int   // For magic: specific cells (no pierce)
}

// AttackType defines how a unit selects targets
type AttackType int

const (
	AttackTypeMeleeRow    AttackType = iota // Targets front row (3 targets max)
	AttackTypeMeleeColumn                   // Targets column (1 target, spear-type)
	AttackTypeRanged                        // Targets same row as attacker
	AttackTypeMagic                         // Cell-based patterns
)

func (a AttackType) String() string {
	switch a {
	case AttackTypeMeleeRow:
		return "MeleeRow"
	case AttackTypeMeleeColumn:
		return "MeleeColumn"
	case AttackTypeRanged:
		return "Ranged"
	case AttackTypeMagic:
		return "Magic"
	default:
		return "Unknown"
	}
}

func (t TargetRowData) String() string {
	return fmt.Sprintf("%s targeting", t.AttackType.String())
}

// UnitTypeData stores the original unit type string for roster grouping.
// Separated from NameComponent which holds the display name.
type UnitTypeData struct {
	UnitType string
}

// ========================================
// LEADER ABILITY COMPONENTS
// ========================================

// LeaderData marks a unit as the squad leader with special abilities
type LeaderData struct {
	Leadership int // Bonus to squad stats
	Experience int // Leader progression (future)
}

// AbilitySlotData represents equipped abilities on a leader (4 slots, FFT-style)
// bytearena/ecs limitation: can't have multiple components of same type,
// so we store all slots in one component as an array
type AbilitySlotData struct {
	Slots [4]AbilitySlot // 4 ability slots
}

type AbilitySlot struct {
	AbilityType  AbilityType // Rally, Heal, BattleCry, Fireball
	TriggerType  TriggerType // When to activate
	Threshold    float64     // Condition threshold
	HasTriggered bool        // Once-per-combat flag
	IsEquipped   bool        // Whether slot is active
}

// AbilityType enum (replaces string-based registry)
type AbilityType int

const (
	AbilityNone AbilityType = iota
	AbilityRally
	AbilityHeal
	AbilityBattleCry
	AbilityFireball
)

func (a AbilityType) String() string {
	switch a {
	case AbilityRally:
		return "Rally"
	case AbilityHeal:
		return "Healing Aura"
	case AbilityBattleCry:
		return "Battle Cry"
	case AbilityFireball:
		return "Fireball"
	default:
		return "Unknown"
	}
}

// TriggerType defines when abilities are checked
type TriggerType int

const (
	TriggerNone         TriggerType = iota
	TriggerSquadHPBelow             // Squad average HP < threshold
	TriggerTurnCount                // Specific turn number
	TriggerEnemyCount               // Number of enemy squads
	TriggerMoraleBelow              // Squad morale < threshold
	TriggerCombatStart              // First turn of combat
)

// CooldownTrackerData tracks ability cooldowns per slot
// One component per leader entity
type CooldownTrackerData struct {
	Cooldowns    [4]int // Turns remaining for slots 0-3
	MaxCooldowns [4]int // Base cooldown durations
}

// ========================================
// ABILITY PARAMETERS (Data-Driven)
// ========================================

// AbilityParams defines ability effects (pure data, no logic)
// Systems read these to execute abilities
type AbilityParams struct {
	StrengthBonus int // Damage increase (Rally, BattleCry)
	HealAmount    int // HP restored (Heal)
	MoraleBonus   int // Morale increase (BattleCry)
	BaseDamage    int // Direct damage (Fireball)
	Duration      int // Effect duration in turns (Rally)
	BaseCooldown  int // Default cooldown
}

// GetAbilityParams returns parameters for each ability type
// This is a lookup table, not a registry with function pointers
func GetAbilityParams(abilityType AbilityType) AbilityParams {
	switch abilityType {
	case AbilityRally:
		return AbilityParams{
			StrengthBonus: 5,
			Duration:      3,
			BaseCooldown:  5,
		}
	case AbilityHeal:
		return AbilityParams{
			HealAmount:   10,
			BaseCooldown: 4,
		}
	case AbilityBattleCry:
		return AbilityParams{
			StrengthBonus: 3,
			MoraleBonus:   10,
			BaseCooldown:  999, // Once per combat
		}
	case AbilityFireball:
		return AbilityParams{
			BaseDamage:   15,
			BaseCooldown: 3,
		}
	default:
		return AbilityParams{}
	}
}

// ========================================
// EXPERIENCE & STAT GROWTH COMPONENTS
// ========================================

// ExperienceData tracks a unit's level and XP progress.
type ExperienceData struct {
	Level         int // Current level (starts at 1)
	CurrentXP     int // XP accumulated toward next level
	XPToNextLevel int // XP required to level up (fixed 100)
}

// GrowthGrade represents a stat growth rate grade.
type GrowthGrade string

const (
	GradeS GrowthGrade = "S" // 90% chance
	GradeA GrowthGrade = "A" // 75% chance
	GradeB GrowthGrade = "B" // 60% chance
	GradeC GrowthGrade = "C" // 45% chance
	GradeD GrowthGrade = "D" // 30% chance
	GradeE GrowthGrade = "E" // 15% chance
	GradeF GrowthGrade = "F" // 5% chance
)

// StatGrowthData defines per-stat growth rates for a unit.
// Each field is a GrowthGrade that determines the chance of +1 on level up.
type StatGrowthData struct {
	Strength   GrowthGrade
	Dexterity  GrowthGrade
	Magic      GrowthGrade
	Leadership GrowthGrade
	Armor      GrowthGrade
	Weapon     GrowthGrade
}
