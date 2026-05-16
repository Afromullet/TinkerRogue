// Package squadcore implements an Entity Component System (ECS) for squad-based
// tactical combat. It provides components for squad management, unit positioning
// in a 3x3 grid, role-based combat behavior, and leader abilities.
//
// The package is designed for turn-based tactical games where players and
// enemies control squads of units in formation-based combat.
package squadcore

import (
	"fmt"
	"game_main/tactical/squads/unitdefs"

	"github.com/bytearena/ecs"
)

// Grid layout limits. Change these to resize the squad grid; all queries,
// validators, and creation paths derive bounds from these constants.
const (
	SquadGridSize = 3                             // Width and height of the squad grid (3x3)
	SquadMaxUnits = SquadGridSize * SquadGridSize // Maximum units a squad can hold
)

// Global components
var (
	SquadComponent        *ecs.Component
	SquadMemberComponent  *ecs.Component
	GridPositionComponent *ecs.Component
	UnitRoleComponent     *ecs.Component
	CoverComponent        *ecs.Component
	LeaderComponent       *ecs.Component
	TargetRowComponent    *ecs.Component
	AttackRangeComponent  *ecs.Component
	MovementSpeedComponent   *ecs.Component
	UnitTypeComponent        *ecs.Component

	SquadTag       ecs.Tag
	SquadMemberTag ecs.Tag
	LeaderTag      ecs.Tag
)

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
	AnchorRow  int // Top-left row (0-2)
	AnchorCol  int // Top-left col (0-2)
	CellWidth  int // Number of grid columns occupied (1-3)
	CellHeight int // Number of grid rows occupied (1-3)
}

// GetOccupiedCells returns all grid cells this unit occupies
func (g *GridPositionData) GetOccupiedCells() [][2]int {
	var cells [][2]int
	for r := g.AnchorRow; r < g.AnchorRow+g.CellHeight && r < SquadGridSize; r++ {
		for c := g.AnchorCol; c < g.AnchorCol+g.CellWidth && c < SquadGridSize; c++ {
			cells = append(cells, [2]int{r, c})
		}
	}
	return cells
}

// OccupiesCell checks if this unit occupies a specific grid cell
func (g *GridPositionData) OccupiesCell(row, col int) bool {
	return row >= g.AnchorRow && row < g.AnchorRow+g.CellHeight &&
		col >= g.AnchorCol && col < g.AnchorCol+g.CellWidth
}

// GetRows returns all row indices this unit occupies
func (g *GridPositionData) GetRows() []int {
	var rows []int
	for r := g.AnchorRow; r < g.AnchorRow+g.CellHeight && r < SquadGridSize; r++ {
		rows = append(rows, r)
	}
	return rows
}

// UnitRoleData defines a unit's combat role
// Affects combat behavior and damage distribution
type UnitRoleData struct {
	Role unitdefs.UnitRole // Tank, DPS, or Support
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
	AttackType  unitdefs.AttackType // MeleeRow, MeleeColumn, Ranged, or Magic
	TargetCells [][2]int            // For magic: specific cells (no pierce)
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

// LeaderData marks a unit as the squad leader.
type LeaderData struct {
	Leadership int // Bonus to squad stats
	Experience int // Leader progression (future)
}
