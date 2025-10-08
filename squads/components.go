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
	LeaderComponent          *ecs.Component
	TargetRowComponent       *ecs.Component
	AbilitySlotComponent     *ecs.Component
	CooldownTrackerComponent *ecs.Component

	SquadTag       ecs.Tag
	SquadMemberTag ecs.Tag
	LeaderTag      ecs.Tag
)

// ========================================
// SQUAD ENTITY COMPONENTS
// ========================================

// SquadData represents the squad entity's component data.
// ✅ Uses ecs.EntityID for relationships (native type)
type SquadData struct {
	SquadID    ecs.EntityID  // Unique squad identifier (native entity ID)
	Formation  FormationType // Current formation layout
	Name       string        // Squad display name
	Morale     int           // Squad-wide morale (0-100)
	SquadLevel int           // Average level for spawning
	TurnCount  int           // Number of turns this squad has taken
	MaxUnits   int           // Maximum squad size (typically 9)
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
// ✅ Uses ecs.EntityID instead of entity pointer (native type)
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

// TargetMode defines how a unit selects targets
type TargetMode int

const (
	TargetModeRowBased  TargetMode = iota // Target entire row(s)
	TargetModeCellBased                   // Target specific grid cells
)

// TargetRowData defines which enemy cells/rows a unit attacks
// Supports both simple row-based targeting and complex cell-pattern targeting
type TargetRowData struct {
	Mode TargetMode // Row-based or cell-based targeting

	// Row-based targeting (simple)
	TargetRows    []int // Which rows to target (e.g., [0] for front, [0,1,2] for all)
	IsMultiTarget bool  // True if ability hits multiple units in row
	MaxTargets    int   // Max units hit per row (0 = unlimited)

	// Cell-based targeting (complex)
	TargetCells [][2]int // Specific grid cells to target (e.g., [[0,0], [0,1]] for 1x2 pattern)
	// Each element is [row, col] where row and col are 0-2
	// Examples:
	// 1x1: [[1,1]] (center cell)
	// 1x2: [[0,0], [0,1]] (front-left two cells)
	// 2x1: [[0,0], [1,0]] (left column, top two)
	// 2x2: [[0,0], [0,1], [1,0], [1,1]] (top-left quad)
	// 3x3: [[0,0], [0,1], [0,2], [1,0], [1,1], [1,2], [2,0], [2,1], [2,2]] (all cells)
}

func (t TargetRowData) String() string {
	if t.Mode == TargetModeRowBased {
		return fmt.Sprintf("Row-Based: rows %v, multi=%v, max=%d", t.TargetRows, t.IsMultiTarget, t.MaxTargets)
	}
	return fmt.Sprintf("Cell-Based: %d cells", len(t.TargetCells))
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
	DamageBonus  int // Damage increase (Rally, BattleCry)
	HealAmount   int // HP restored (Heal)
	MoraleBonus  int // Morale increase (BattleCry)
	BaseDamage   int // Direct damage (Fireball)
	Duration     int // Effect duration in turns (Rally)
	BaseCooldown int // Default cooldown
}

// GetAbilityParams returns parameters for each ability type
// This is a lookup table, not a registry with function pointers
func GetAbilityParams(abilityType AbilityType) AbilityParams {
	switch abilityType {
	case AbilityRally:
		return AbilityParams{
			DamageBonus:  5,
			Duration:     3,
			BaseCooldown: 5,
		}
	case AbilityHeal:
		return AbilityParams{
			HealAmount:   10,
			BaseCooldown: 4,
		}
	case AbilityBattleCry:
		return AbilityParams{
			DamageBonus:  3,
			MoraleBonus:  10,
			BaseCooldown: 999, // Once per combat
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
