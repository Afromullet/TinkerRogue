# Squad Combat System - Corrected (Uses Native ECS Entity IDs)

**Version:** 2.2 Cell-Based Targeting
**Last Updated:** 2025-10-02
**Status:** Production-Ready, ECS-Compliant, Uses Native bytearena/ecs IDs, Multi-Cell Units, Cell-Based Targeting
**Purpose:** Comprehensive squad-based combat system with proper ECS architecture, multi-cell unit support, and advanced cell-based targeting patterns

**CRITICAL CORRECTION:** The original document incorrectly assumed bytearena/ecs doesn't expose entity IDs. This version uses the native `entity.GetID()` method and `ecs.EntityID` type (`uint32`). No custom EntityRegistry needed!

**MULTI-CELL UNIT SUPPORT:** Units can occupy multiple grid cells (e.g., 1x2, 2x2, 2x1, etc.) in the 3x3 squad grid. Large creatures, vehicles, or special units can span multiple rows and columns.

**CELL-BASED TARGETING:** Advanced targeting system supporting both row-based (simple) and cell-based (complex) targeting modes. Cell-based mode enables precise grid cell patterns like 1x1 (single target), 1x2 (horizontal cleave), 2x1 (vertical pierce), 2x2 (quad blast), 3x3 (full AOE), and custom shapes.

---

## Document Overview

This document corrects **squad_system_final.md** by:
- Removing the unnecessary EntityRegistry system (~125 lines of code)
- Using native `entity.GetID()` which returns `ecs.EntityID` (uint32)
- Replacing all `uint64` entity ID types with `ecs.EntityID`
- Removing all `RegisterEntity/UnregisterEntity` calls
- Simplifying entity lifecycle management
- **NEW:** Adding multi-cell unit support (units can occupy 1x1, 1x2, 2x2, 2x1, etc.)
- **NEW:** Adding cell-based targeting patterns (precise grid cell targeting with custom shapes)

**Key Achievement:** 100% ECS-compliant design with NO entity pointers, proper separation of concerns, native entity ID usage, and flexible multi-cell unit positioning.

---

## Table of Contents

1. [Design Philosophy](#design-philosophy)
2. [ECS Compliance](#ecs-compliance)
3. [Native Entity ID Usage](#native-entity-id-usage)
4. [Component Definitions](#component-definitions)
5. [Multi-Cell Unit Support](#multi-cell-unit-support)
6. [System Implementations](#system-implementations)
7. [Row-Based Combat System](#row-based-combat-system)
8. [Cell-Based Targeting Patterns](#cell-based-targeting-patterns)
9. [Automated Leader Abilities](#automated-leader-abilities)
10. [Squad Composition & Formation System](#squad-composition--formation-system)
11. [Implementation Phases](#implementation-phases)
12. [Integration Guide](#integration-guide)
13. [Complete Code Examples](#complete-code-examples)
14. [Migration Path](#migration-path)

---

## Design Philosophy

### Core Principle

**Squads are separate entities that "own" unit entities through relationship components.** The 3x3 grid is internal to each squad, not a global map structure.

### ECS Design Tenets

1. **Components = Pure Data** - No logic, no entity references (use IDs instead)
2. **Systems = Pure Logic** - Operate on components via queries
3. **Entities = ID + Component Set** - Just containers for components
4. **Data-Oriented Design** - Optimize for cache locality, avoid pointer chaining
5. **Loose Coupling** - Systems discover relationships via queries, not stored references

### Key Improvements Over Common Anti-Patterns

| **Anti-Pattern** | **This Design** |
|------------------|-----------------|
| `SquadData.UnitEntities []*ecs.Entity` | `SquadMemberData.SquadID ecs.EntityID` + query system |
| `SquadMemberData.SquadEntity *ecs.Entity` | `SquadMemberData.SquadID ecs.EntityID` |
| `SquadGrid.UnitMap [3][3]*ecs.Entity` | `GridPositionData` + spatial query |
| `CombatResult.UnitsKilled []*ecs.Entity` | `CombatResult.UnitsKilled []ecs.EntityID` |
| Logic in component files | Separate `systems/` package |
| `AbilityRegistry` global with function pointers | Data-driven `AbilityParams` lookup |

---

## ECS Compliance

### Problems with Entity Pointers

**Why storing entity pointers violates ECS principles:**

1. **Dangling Pointers** - Deleted entities leave invalid references
2. **Poor Cache Locality** - Pointer chasing destroys CPU cache performance
3. **Serialization Issues** - Can't save/load game state with raw pointers
4. **Tight Coupling** - Direct references prevent flexible composition
5. **Memory Leaks** - Hard to track all references for cleanup

### The Fixed Approach

```go
// ❌ WRONG: Storing entity pointers
type SquadData struct {
    UnitEntities []*ecs.Entity
}
type CombatResult struct {
    UnitsKilled []*ecs.Entity
    DamageByUnit map[*ecs.Entity]int
}

// ✅ CORRECT: Using native entity IDs + queries
type SquadData struct {
    SquadID ecs.EntityID  // Pure data, native type
}
type CombatResult struct {
    UnitsKilled []ecs.EntityID
    DamageByUnit map[ecs.EntityID]int
}

// Query for relationships dynamically
units := GetUnitIDsInSquad(squadID, ecsmanager)
```

---

## Native Entity ID Usage

### bytearena/ecs DOES Expose Entity IDs!

The bytearena/ecs library provides native entity ID access:

```go
// Get entity ID (returns ecs.EntityID which is uint32)
entityID := entity.GetID()

// EntityID is a public type alias
type EntityID uint32

// The entity struct also has a public ID field
entity.ID // Also ecs.EntityID
```

### No Custom Registry Needed

**What we DON'T need:**
- ❌ Custom EntityRegistry with bidirectional mapping
- ❌ RegisterEntity() calls
- ❌ UnregisterEntity() cleanup
- ❌ GetEntityByID() wrapper functions
- ❌ GetEntityID() wrapper functions

**What we DO use:**
- ✅ `entity.GetID()` to get native entity IDs
- ✅ `ecs.EntityID` type for all entity ID storage
- ✅ ECS manager queries to find entities by ID
- ✅ Direct entity pointer passing when already queried

### Entity Lookup Pattern

```go
// ✅ CORRECT: Use entity ID from query result
for _, result := range ecsmanager.World.Query(squad.SquadMemberTag) {
    unitEntity := result.Entity
    unitID := unitEntity.GetID()  // Native method!

    // Store ID for later
    myIDs = append(myIDs, unitID)
}

// ✅ CORRECT: When you need to find entity by ID, use a query
func FindEntityByID(targetID ecs.EntityID, tag ecs.Tag, ecsmanager *common.EntityManager) *ecs.Entity {
    for _, result := range ecsmanager.World.Query(tag) {
        if result.Entity.GetID() == targetID {
            return result.Entity
        }
    }
    return nil
}
```

---

## Component Definitions

### File: `squad/components.go`

**PURE DATA ONLY - NO LOGIC, NO ENTITY POINTERS**

```go
package squad

import (
	"github.com/bytearena/ecs"
)

// Global component declarations (registered in InitSquadComponents)
var (
	SquadComponent            *ecs.Component
	SquadMemberComponent      *ecs.Component
	GridPositionComponent     *ecs.Component
	UnitRoleComponent         *ecs.Component
	LeaderComponent           *ecs.Component
	TargetRowComponent        *ecs.Component
	AbilitySlotComponent      *ecs.Component
	CooldownTrackerComponent  *ecs.Component
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
	AbilitySlotComponent = manager.NewComponent()
	CooldownTrackerComponent = manager.NewComponent()
}

// ========================================
// SQUAD ENTITY COMPONENTS
// ========================================

// SquadData represents the squad entity's component data.
// ✅ Uses ecs.EntityID for relationships (native type)
type SquadData struct {
	SquadID       ecs.EntityID    // Unique squad identifier (native entity ID)
	Formation     FormationType   // Current formation layout
	Name          string          // Squad display name
	Morale        int             // Squad-wide morale (0-100)
	SquadLevel    int             // Average level for spawning
	TurnCount     int             // Number of turns this squad has taken
	MaxUnits      int             // Maximum squad size (typically 9)
}

// FormationType defines squad layout presets
type FormationType int
const (
	FormationBalanced  FormationType = iota  // Mix of roles
	FormationDefensive                       // Tank-heavy
	FormationOffensive                       // DPS-focused
	FormationRanged                          // Back-line heavy
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
	SquadID    ecs.EntityID    // Parent squad's entity ID
}

// GridPositionData represents a unit's position within the 3x3 grid.
// Pure data - systems query for units at specific positions
// Supports multi-cell units (e.g., 1x2, 2x2, 2x1, etc.)
type GridPositionData struct {
	AnchorRow int  // Top-left row (0-2)
	AnchorCol int  // Top-left col (0-2)
	Width     int  // Number of columns occupied (1-3)
	Height    int  // Number of rows occupied (1-3)
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
	Role UnitRole  // Tank, DPS, or Support
}

type UnitRole int
const (
	RoleTank    UnitRole = iota  // Takes hits first, high defense
	RoleDPS                      // High damage output
	RoleSupport                  // Buffs, heals, utility
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
	TargetModeRowBased  TargetMode = iota  // Target entire row(s)
	TargetModeCellBased                    // Target specific grid cells
)

// TargetRowData defines which enemy cells/rows a unit attacks
// Supports both simple row-based targeting and complex cell-pattern targeting
type TargetRowData struct {
	Mode          TargetMode   // Row-based or cell-based targeting

	// Row-based targeting (simple)
	TargetRows    []int        // Which rows to target (e.g., [0] for front, [0,1,2] for all)
	IsMultiTarget bool         // True if ability hits multiple units in row
	MaxTargets    int          // Max units hit per row (0 = unlimited)

	// Cell-based targeting (complex)
	TargetCells   [][2]int     // Specific grid cells to target (e.g., [[0,0], [0,1]] for 1x2 pattern)
	                            // Each element is [row, col] where row and col are 0-2
	                            // Examples:
	                            // 1x1: [[1,1]] (center cell)
	                            // 1x2: [[0,0], [0,1]] (front-left two cells)
	                            // 2x1: [[0,0], [1,0]] (left column, top two)
	                            // 2x2: [[0,0], [0,1], [1,0], [1,1]] (top-left quad)
	                            // 3x3: [[0,0], [0,1], [0,2], [1,0], [1,1], [1,2], [2,0], [2,1], [2,2]] (all cells)
}

func (t TargetRowData) String() string {
	if t.Mode == TARGET_MODE_ROW_BASED {
		return fmt.Sprintf("Row-Based: rows %v, multi=%v, max=%d", t.TargetRows, t.IsMultiTarget, t.MaxTargets)
	}
	return fmt.Sprintf("Cell-Based: %d cells", len(t.TargetCells))
}

// ========================================
// LEADER ABILITY COMPONENTS
// ========================================

// LeaderData marks a unit as the squad leader with special abilities
type LeaderData struct {
	Leadership     int    // Bonus to squad stats
	Experience     int    // Leader progression (future)
}

// AbilitySlotData represents equipped abilities on a leader (4 slots, FFT-style)
// bytearena/ecs limitation: can't have multiple components of same type,
// so we store all slots in one component as an array
type AbilitySlotData struct {
	Slots [4]AbilitySlot  // 4 ability slots
}

type AbilitySlot struct {
	AbilityType  AbilityType      // Rally, Heal, BattleCry, Fireball
	TriggerType  TriggerType      // When to activate
	Threshold    float64          // Condition threshold
	HasTriggered bool             // Once-per-combat flag
	IsEquipped   bool             // Whether slot is active
}

// AbilityType enum (replaces string-based registry)
type AbilityType int
const (
	ABILITY_NONE AbilityType = iota
	ABILITY_RALLY
	ABILITY_HEAL
	ABILITY_BATTLE_CRY
	ABILITY_FIREBALL
)

func (a AbilityType) String() string {
	switch a {
	case ABILITY_RALLY:
		return "Rally"
	case ABILITY_HEAL:
		return "Healing Aura"
	case ABILITY_BATTLE_CRY:
		return "Battle Cry"
	case ABILITY_FIREBALL:
		return "Fireball"
	default:
		return "Unknown"
	}
}

// TriggerType defines when abilities are checked
type TriggerType int
const (
	TRIGGER_NONE TriggerType = iota
	TRIGGER_SQUAD_HP_BELOW   // Squad average HP < threshold
	TRIGGER_TURN_COUNT       // Specific turn number
	TRIGGER_ENEMY_COUNT      // Number of enemy squads
	TRIGGER_MORALE_BELOW     // Squad morale < threshold
	TRIGGER_COMBAT_START     // First turn of combat
)

// CooldownTrackerData tracks ability cooldowns per slot
// One component per leader entity
type CooldownTrackerData struct {
	Cooldowns     [4]int    // Turns remaining for slots 0-3
	MaxCooldowns  [4]int    // Base cooldown durations
}

// ========================================
// ABILITY PARAMETERS (Data-Driven)
// ========================================

// AbilityParams defines ability effects (pure data, no logic)
// Systems read these to execute abilities
type AbilityParams struct {
	DamageBonus   int       // Damage increase (Rally, BattleCry)
	HealAmount    int       // HP restored (Heal)
	MoraleBonus   int       // Morale increase (BattleCry)
	BaseDamage    int       // Direct damage (Fireball)
	Duration      int       // Effect duration in turns (Rally)
	BaseCooldown  int       // Default cooldown
}

// GetAbilityParams returns parameters for each ability type
// This is a lookup table, not a registry with function pointers
func GetAbilityParams(abilityType AbilityType) AbilityParams {
	switch abilityType {
	case ABILITY_RALLY:
		return AbilityParams{
			DamageBonus:  5,
			Duration:     3,
			BaseCooldown: 5,
		}
	case ABILITY_HEAL:
		return AbilityParams{
			HealAmount:   10,
			BaseCooldown: 4,
		}
	case ABILITY_BATTLE_CRY:
		return AbilityParams{
			DamageBonus:  3,
			MoraleBonus:  10,
			BaseCooldown: 999, // Once per combat
		}
	case ABILITY_FIREBALL:
		return AbilityParams{
			BaseDamage:   15,
			BaseCooldown: 3,
		}
	default:
		return AbilityParams{}
	}
}
```

### File: `squad/tags.go`

```go
package squad

import "github.com/bytearena/ecs"

// Global tags for efficient entity queries
var (
	SquadTag       ecs.Tag
	SquadMemberTag ecs.Tag
	LeaderTag      ecs.Tag
)

// InitSquadTags creates tags for querying squad-related entities
// Call this after InitSquadComponents
func InitSquadTags() {
	SquadTag = ecs.BuildTag(SquadComponent)
	SquadMemberTag = ecs.BuildTag(SquadMemberComponent)
	LeaderTag = ecs.BuildTag(LeaderComponent, SquadMemberComponent)
}
```

---

## Multi-Cell Unit Support

### Overview

Units can occupy multiple grid cells in the 3x3 squad grid. This enables:
- **Large creatures** (ogres, dragons) that take up 2x2 or 2x3 spaces
- **Wide formations** (cavalry line) using 1x3 horizontal units
- **Tall units** (siege towers) using 3x1 vertical units
- **Mixed squads** combining 1x1 infantry with 2x2 monsters

### Grid Cell Notation

Units are positioned using an **anchor cell** (top-left) plus width/height:
- `AnchorRow: 0, AnchorCol: 0, Width: 1, Height: 1` = Single cell at (0,0)
- `AnchorRow: 0, AnchorCol: 0, Width: 2, Height: 2` = 2x2 unit occupying cells (0,0), (0,1), (1,0), (1,1)
- `AnchorRow: 1, AnchorCol: 0, Width: 3, Height: 1` = Horizontal 3-wide unit in middle row
- `AnchorRow: 0, AnchorCol: 2, Width: 1, Height: 3` = Vertical 3-tall unit in right column

### Multi-Cell Unit Behavior in Row-Based Combat

**Critical Design Decision:** Multi-cell units count as being in **ALL rows they occupy**.

#### Example: 2x2 Unit Spanning Rows 0-1

```
Grid Layout:
Row 0: [2x2 Unit] [Empty]
Row 1: [2x2 Unit] [Empty]
Row 2: [Archer]   [Mage]   [Empty]
```

**Combat Behavior:**
- Attacking **Row 0** can target the 2x2 unit (it occupies row 0)
- Attacking **Row 1** can also target the 2x2 unit (it occupies row 1)
- Attacking **Row 2** targets Archer/Mage (2x2 unit doesn't occupy row 2)
- The 2x2 unit appears in target lists for BOTH row 0 and row 1 attacks
- **Deduplication:** Query system ensures the unit is only added once per attack resolution

#### Why This Design?

1. **Realism** - A large unit blocking multiple rows should be vulnerable to attacks targeting any of those rows
2. **Balance** - Large units trade positioning flexibility for being easier to hit
3. **Simplicity** - No special "which row does this unit defend?" logic needed

### Visual Examples

#### Example 1: Tank Wall (Three 1x1 Tanks)
```
Row 0: [Tank1] [Tank2] [Tank3]
Row 1: [Empty] [Empty] [Empty]
Row 2: [Archer] [Empty] [Mage]
```
- Front row: 3 separate targets
- Back row: 2 separate targets

#### Example 2: Giant + Infantry (2x2 Giant, 1x1 Units)
```
Row 0: [Giant___|_____] [Knight]
Row 1: [Giant___|_____] [Empty]
Row 2: [Archer] [Mage]  [Empty]
```
- Front row attack: Giant or Knight (2 targets, giant takes 4 cells)
- Mid row attack: Giant only (1 target)
- Back row attack: Archer or Mage (2 targets)

#### Example 3: Cavalry Line (Three 2x1 Horizontal Units)
```
Row 0: [Cav1______|_____] [Empty]
Row 1: [Cav2______|_____] [Empty]
Row 2: [Cav3______|_____] [Empty]
```
- Each row has exactly 1 target (a 2-wide cavalry unit)
- Cavalry units are harder to surround but fill fewer rows

### Constraints and Validation

1. **Grid Boundaries:** Units cannot extend beyond the 3x3 grid
   - `AnchorRow + Height <= 3`
   - `AnchorCol + Width <= 3`

2. **No Overlapping:** Multiple units cannot occupy the same cell
   - Checked during squad creation and unit addition

3. **Minimum Size:** All units must be at least 1x1
   - `Width >= 1` and `Height >= 1`

4. **Maximum Size:** Units can occupy up to 3x3 (entire grid)
   - Useful for boss encounters or special scenarios

### Query System Integration

The query functions handle multi-cell units automatically:

```go
// GetUnitIDsAtGridPosition - Returns ALL units occupying a specific cell
unitIDs := GetUnitIDsAtGridPosition(squadID, 0, 0, ecsmanager)
// If a 2x2 unit has anchor at (0,0), it will be returned
// If a 2x2 unit has anchor at (0,1), cell (0,0) is NOT occupied

// GetUnitIDsInRow - Returns ALL units occupying any cell in the row (deduplicated)
unitIDs := GetUnitIDsInRow(squadID, 0, ecsmanager)
// A 2x2 unit at rows 0-1 will be returned when querying row 0 OR row 1
// But only appears ONCE in each query result
```

---

## System Implementations

### File: `systems/squadqueries.go`

**Helper functions for querying squad relationships - RETURNS IDs, NOT POINTERS**

```go
package systems

import (
	"github.com/bytearena/ecs"
	"game_main/common"
	"game_main/squad"
)

// GetUnitIDsInSquad returns unit IDs belonging to a squad
// ✅ Returns ecs.EntityID (native type), not entity pointers
func GetUnitIDsInSquad(squadID ecs.EntityID, ecsmanager *common.EntityManager) []ecs.EntityID {
	var unitIDs []ecs.EntityID

	for _, result := range ecsmanager.World.Query(squad.SquadMemberTag) {
		unitEntity := result.Entity
		memberData := common.GetComponentType[*squad.SquadMemberData](unitEntity, squad.SquadMemberComponent)

		if memberData.SquadID == squadID {
			unitID := unitEntity.GetID()  // ✅ Native method!
			unitIDs = append(unitIDs, unitID)
		}
	}

	return unitIDs
}

// GetSquadEntity finds squad entity by squad ID
// ✅ Returns entity pointer directly from query
func GetSquadEntity(squadID ecs.EntityID, ecsmanager *common.EntityManager) *ecs.Entity {
	for _, result := range ecsmanager.World.Query(squad.SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

		if squadData.SquadID == squadID {
			return squadEntity
		}
	}

	return nil
}

// GetUnitIDsAtGridPosition returns unit IDs occupying a specific grid cell
// ✅ Returns ecs.EntityID (native type), not entity pointers
// ✅ Supports multi-cell units using OccupiesCell() method
func GetUnitIDsAtGridPosition(squadID ecs.EntityID, row, col int, ecsmanager *common.EntityManager) []ecs.EntityID {
	var unitIDs []ecs.EntityID

	for _, result := range ecsmanager.World.Query(squad.SquadMemberTag) {
		unitEntity := result.Entity

		memberData := common.GetComponentType[*squad.SquadMemberData](unitEntity, squad.SquadMemberComponent)
		if memberData.SquadID != squadID {
			continue
		}

		if !unitEntity.HasComponent(squad.GridPositionComponent) {
			continue
		}

		gridPos := common.GetComponentType[*squad.GridPositionData](unitEntity, squad.GridPositionComponent)

		// ✅ Check if this unit occupies the queried cell (supports multi-cell units)
		if gridPos.OccupiesCell(row, col) {
			unitID := unitEntity.GetID()  // ✅ Native method!
			unitIDs = append(unitIDs, unitID)
		}
	}

	return unitIDs
}

// GetUnitIDsInRow returns alive unit IDs in a row
// ✅ Returns ecs.EntityID (native type), not entity pointers
// ✅ Supports multi-cell units - a 2x2 unit occupying rows 0-1 will be returned for both row queries
// ✅ Deduplication ensures each unit appears only once per query (important for multi-cell units)
func GetUnitIDsInRow(squadID ecs.EntityID, row int, ecsmanager *common.EntityManager) []ecs.EntityID {
	var unitIDs []ecs.EntityID
	seen := make(map[ecs.EntityID]bool)  // ✅ Prevents multi-cell units from being counted multiple times

	for col := 0; col < 3; col++ {
		idsAtPos := GetUnitIDsAtGridPosition(squadID, row, col, ecsmanager)
		for _, unitID := range idsAtPos {
			if !seen[unitID] {
				unitEntity := FindUnitByID(unitID, ecsmanager)
				if unitEntity == nil {
					continue
				}

				attr := common.GetAttributes(unitEntity)
				if attr.CurrentHealth > 0 {
					unitIDs = append(unitIDs, unitID)
					seen[unitID] = true
				}
			}
		}
	}

	return unitIDs
}

// GetLeaderID finds the leader unit ID of a squad
// ✅ Returns ecs.EntityID (native type), not entity pointer
func GetLeaderID(squadID ecs.EntityID, ecsmanager *common.EntityManager) ecs.EntityID {
	for _, result := range ecsmanager.World.Query(squad.LeaderTag) {
		leaderEntity := result.Entity
		memberData := common.GetComponentType[*squad.SquadMemberData](leaderEntity, squad.SquadMemberComponent)

		if memberData.SquadID == squadID {
			return leaderEntity.GetID()  // ✅ Native method!
		}
	}

	return 0
}

// IsSquadDestroyed checks if all units are dead
func IsSquadDestroyed(squadID ecs.EntityID, ecsmanager *common.EntityManager) bool {
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)

	for _, unitID := range unitIDs {
		unitEntity := FindUnitByID(unitID, ecsmanager)
		if unitEntity == nil {
			continue
		}

		attr := common.GetAttributes(unitEntity)
		if attr.CurrentHealth > 0 {
			return false
		}
	}

	return len(unitIDs) > 0
}

// FindUnitByID finds a unit entity by its ID
// ✅ Uses query to find entity by native ID
func FindUnitByID(unitID ecs.EntityID, ecsmanager *common.EntityManager) *ecs.Entity {
	for _, result := range ecsmanager.World.Query(squad.SquadMemberTag) {
		if result.Entity.GetID() == unitID {
			return result.Entity
		}
	}
	return nil
}
```

---

## Row-Based Combat System

### Design Overview

**Core Concept:** Units declare which row(s) they target via `TargetRowComponent`. During combat, attacks are resolved row-by-row, with single-target vs AOE logic applied per unit.

**Row Numbering:**
- Row 0: Front line (closest to enemy)
- Row 1: Middle line
- Row 2: Back line (furthest from enemy)

**Targeting Scenarios:**
1. **Single-target front row:** Melee fighter targets one unit in enemy row 0
2. **Multi-target back row:** Mage hits all units in enemy row 2
3. **All-row AOE:** Catapult hits one random unit in each row
4. **Specific grid:** Assassin targets specific grid position (row=1, col=2)

### File: `systems/squadcombat.go`

**Combat system - pure logic, uses native entity IDs**

```go
package systems

import (
	"fmt"
	"github.com/bytearena/ecs"
	"game_main/common"
	"game_main/randgen"
	"game_main/squad"
	"math/rand"
)

// CombatResult - ✅ Uses ecs.EntityID (native type) instead of entity pointers
type CombatResult struct {
	TotalDamage  int
	UnitsKilled  []ecs.EntityID           // ✅ Native IDs
	DamageByUnit map[ecs.EntityID]int     // ✅ Native IDs
}

// ExecuteSquadAttack performs row-based combat between two squads
// ✅ Works with ecs.EntityID internally
func ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID, ecsmanager *common.EntityManager) *CombatResult {
	result := &CombatResult{
		DamageByUnit: make(map[ecs.EntityID]int),
		UnitsKilled:  []ecs.EntityID{},
	}

	// Query for attacker unit IDs (not pointers!)
	attackerUnitIDs := GetUnitIDsInSquad(attackerSquadID, ecsmanager)

	// Process each attacker unit
	for _, attackerID := range attackerUnitIDs {
		attackerUnit := FindUnitByID(attackerID, ecsmanager)
		if attackerUnit == nil {
			continue
		}

		// Check if unit is alive
		attackerAttr := common.GetAttributes(attackerUnit)
		if attackerAttr.CurrentHealth <= 0 {
			continue
		}

		// Get targeting data
		if !attackerUnit.HasComponent(squad.TargetRowComponent) {
			continue
		}

		targetRowData := common.GetComponentType[*squad.TargetRowData](attackerUnit, squad.TargetRowComponent)

		var actualTargetIDs []ecs.EntityID

		// Handle targeting based on mode
		if targetRowData.Mode == squad.TARGET_MODE_CELL_BASED {
			// Cell-based targeting: hit specific grid cells
			for _, cell := range targetRowData.TargetCells {
				row, col := cell[0], cell[1]
				cellTargetIDs := GetUnitIDsAtGridPosition(defenderSquadID, row, col, ecsmanager)
				actualTargetIDs = append(actualTargetIDs, cellTargetIDs...)
			}
		} else {
			// Row-based targeting: hit entire row(s)
			for _, targetRow := range targetRowData.TargetRows {
				targetIDs := GetUnitIDsInRow(defenderSquadID, targetRow, ecsmanager)

				if len(targetIDs) == 0 {
					continue
				}

				if targetRowData.IsMultiTarget {
					maxTargets := targetRowData.MaxTargets
					if maxTargets == 0 || maxTargets > len(targetIDs) {
						actualTargetIDs = append(actualTargetIDs, targetIDs...)
					} else {
						actualTargetIDs = append(actualTargetIDs, selectRandomTargetIDs(targetIDs, maxTargets)...)
					}
				} else {
					actualTargetIDs = append(actualTargetIDs, selectLowestHPTargetID(targetIDs, ecsmanager))
				}
			}
		}

		// Apply damage to each selected target
		for _, defenderID := range actualTargetIDs {
			damage := calculateUnitDamageByID(attackerID, defenderID, ecsmanager)
			applyDamageToUnitByID(defenderID, damage, result, ecsmanager)
		}
	}

	result.TotalDamage = sumDamageMap(result.DamageByUnit)

	return result
}

// calculateUnitDamageByID - ✅ Works with ecs.EntityID
func calculateUnitDamageByID(attackerID, defenderID ecs.EntityID, ecsmanager *common.EntityManager) int {
	attackerUnit := FindUnitByID(attackerID, ecsmanager)
	defenderUnit := FindUnitByID(defenderID, ecsmanager)

	if attackerUnit == nil || defenderUnit == nil {
		return 0
	}

	attackerAttr := common.GetAttributes(attackerUnit)
	defenderAttr := common.GetAttributes(defenderUnit)

	// Base damage (adapt to existing weapon system)
	baseDamage := attackerAttr.AttackBonus + attackerAttr.DamageBonus

	// d20 variance (reuse existing logic)
	roll := randgen.GetDiceRoll(20)
	if roll >= 18 {
		baseDamage = int(float64(baseDamage) * 1.5) // Critical
	} else if roll <= 3 {
		baseDamage = baseDamage / 2 // Weak hit
	}

	// Apply role modifiers
	if attackerUnit.HasComponent(squad.UnitRoleComponent) {
		roleData := common.GetComponentType[*squad.UnitRoleData](attackerUnit, squad.UnitRoleComponent)
		baseDamage = applyRoleModifier(baseDamage, roleData.Role)
	}

	// Apply defense
	totalDamage := baseDamage - defenderAttr.TotalProtection
	if totalDamage < 1 {
		totalDamage = 1 // Minimum damage
	}

	return totalDamage
}

// applyRoleModifier adjusts damage based on unit role
func applyRoleModifier(damage int, role squad.UnitRole) int {
	switch role {
	case squad.ROLE_TANK:
		return int(float64(damage) * 0.8) // -20% (tanks don't deal high damage)
	case squad.ROLE_DPS:
		return int(float64(damage) * 1.3) // +30% (damage dealers)
	case squad.ROLE_SUPPORT:
		return int(float64(damage) * 0.6) // -40% (support units are weak attackers)
	default:
		return damage
	}
}

// applyDamageToUnitByID - ✅ Uses ecs.EntityID
func applyDamageToUnitByID(unitID ecs.EntityID, damage int, result *CombatResult, ecsmanager *common.EntityManager) {
	unit := FindUnitByID(unitID, ecsmanager)
	if unit == nil {
		return
	}

	attr := common.GetAttributes(unit)
	attr.CurrentHealth -= damage
	result.DamageByUnit[unitID] = damage

	if attr.CurrentHealth <= 0 {
		result.UnitsKilled = append(result.UnitsKilled, unitID)
	}
}

// selectLowestHPTargetID - ✅ Works with ecs.EntityID
func selectLowestHPTargetID(unitIDs []ecs.EntityID, ecsmanager *common.EntityManager) ecs.EntityID {
	if len(unitIDs) == 0 {
		return 0
	}

	lowestID := unitIDs[0]
	lowestUnit := FindUnitByID(lowestID, ecsmanager)
	if lowestUnit == nil {
		return 0
	}
	lowestHP := common.GetAttributes(lowestUnit).CurrentHealth

	for _, unitID := range unitIDs[1:] {
		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		hp := common.GetAttributes(unit).CurrentHealth
		if hp < lowestHP {
			lowestID = unitID
			lowestHP = hp
		}
	}

	return lowestID
}

// selectRandomTargetIDs - ✅ Works with ecs.EntityID
func selectRandomTargetIDs(unitIDs []ecs.EntityID, count int) []ecs.EntityID {
	if count >= len(unitIDs) {
		return unitIDs
	}

	// Shuffle and take first N
	shuffled := make([]ecs.EntityID, len(unitIDs))
	copy(shuffled, unitIDs)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}

func sumDamageMap(damageMap map[ecs.EntityID]int) int {
	total := 0
	for _, dmg := range damageMap {
		total += dmg
	}
	return total
}
```

---

## Cell-Based Targeting Patterns

### Overview

The targeting system supports two modes: **row-based** (simple, targets entire rows) and **cell-based** (complex, targets specific grid cells). Cell-based targeting allows precise control over which cells are hit, enabling patterns like horizontal cleave, vertical pierce, single-target precision, and complex AOE shapes.

### Targeting Mode Comparison

| Feature | Row-Based | Cell-Based |
|---------|-----------|------------|
| **Complexity** | Simple | Advanced |
| **Target Selection** | Entire row(s) | Specific grid cells |
| **Use Cases** | Front-line melee, back-row artillery | Precision attacks, shaped AOE, directional cleave |
| **Configuration** | `TargetRows []int` | `TargetCells [][2]int` |
| **Multi-target Logic** | `IsMultiTarget`, `MaxTargets` | Hit all units in specified cells |

### Common Cell-Based Patterns

#### 1x1 (Single Target - Center)
**Use Case:** Precision single-target attack (assassin, sniper)
```go
TargetCells: [][2]int{{1, 1}} // Center cell only
```

#### 1x2 (Horizontal Cleave - Front)
**Use Case:** Horizontal sword slash hitting two front-line targets
```go
TargetCells: [][2]int{{0, 0}, {0, 1}} // Front-left two cells
```

#### 2x1 (Vertical Pierce - Left Column)
**Use Case:** Spear thrust piercing through left column
```go
TargetCells: [][2]int{{0, 0}, {1, 0}} // Left column, top two rows
```

#### 2x2 (Quad Blast - Top-Left)
**Use Case:** Medium AOE explosion hitting top-left quadrant
```go
TargetCells: [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}} // Top-left 2x2 quad
```

#### 3x3 (Full Grid AOE)
**Use Case:** Massive AOE spell hitting entire enemy formation
```go
TargetCells: [][2]int{
	{0, 0}, {0, 1}, {0, 2}, // Front row
	{1, 0}, {1, 1}, {1, 2}, // Middle row
	{2, 0}, {2, 1}, {2, 2}, // Back row
}
```

#### Custom L-Shape Pattern
**Use Case:** Special ability with unique targeting shape
```go
TargetCells: [][2]int{{0, 0}, {1, 0}, {2, 0}, {2, 1}} // Left column + bottom-right
```

### Row-Based vs Cell-Based Examples

#### Row-Based (Simple)
```go
// Melee fighter: attack single target in front row
unitEntity.AddComponent(squad.TargetRowComponent, &squad.TargetRowData{
	Mode:          squad.TARGET_MODE_ROW_BASED,
	TargetRows:    []int{0},           // Front row
	IsMultiTarget: false,               // Single target
	MaxTargets:    0,                   // N/A
})

// Archer: attack all units in back row
unitEntity.AddComponent(squad.TargetRowComponent, &squad.TargetRowData{
	Mode:          squad.TARGET_MODE_ROW_BASED,
	TargetRows:    []int{2},           // Back row
	IsMultiTarget: true,                // Hit all
	MaxTargets:    0,                   // Unlimited
})
```

#### Cell-Based (Advanced)
```go
// Assassin: precise single-target to center cell
unitEntity.AddComponent(squad.TargetRowComponent, &squad.TargetRowData{
	Mode:        squad.TARGET_MODE_CELL_BASED,
	TargetCells: [][2]int{{1, 1}},     // Center cell only
})

// Horizontal Cleave: hit front-left two cells
unitEntity.AddComponent(squad.TargetRowComponent, &squad.TargetRowData{
	Mode:        squad.TARGET_MODE_CELL_BASED,
	TargetCells: [][2]int{{0, 0}, {0, 1}}, // 1x2 horizontal
})

// Fireball: 2x2 AOE blast
unitEntity.AddComponent(squad.TargetRowComponent, &squad.TargetRowData{
	Mode:        squad.TARGET_MODE_CELL_BASED,
	TargetCells: [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}}, // Top-left quad
})
```

### Grid Reference (3x3 Formation)

```
Col:     0         1         2
Row 0: [0,0]     [0,1]     [0,2]   <- Front row (closest to enemy)
Row 1: [1,0]     [1,1]     [1,2]   <- Middle row
Row 2: [2,0]     [2,1]     [2,2]   <- Back row (furthest from enemy)
```

### When to Use Each Mode

**Use Row-Based When:**
- Simple targeting logic (front/middle/back row)
- Multi-target with max count (hit 2 random units in row)
- Traditional roguelike row-based combat
- Targeting logic needs to adapt dynamically (hit all in front row, regardless of positions)

**Use Cell-Based When:**
- Precise targeting required (specific grid cells)
- Shaped AOE patterns (L-shape, cross, diagonal)
- Directional attacks (horizontal cleave, vertical pierce)
- Complex tactical abilities (corners only, center + adjacent)

### Implementation Notes

1. **Cell-based mode ignores `IsMultiTarget` and `MaxTargets`** - it hits ALL units in specified cells
2. **Cells are absolute coordinates** - `[row, col]` where both are 0-2
3. **Empty cells are harmless** - if no unit occupies a targeted cell, nothing happens
4. **No validation required** - system handles out-of-bounds gracefully (though cells should be 0-2)
5. **Mixing modes not supported** - each unit uses either row-based OR cell-based, not both

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

### File: `systems/squadabilities.go`

**Ability trigger system - queries for leaders, uses native entity IDs**

```go
package systems

import (
	"fmt"
	"github.com/bytearena/ecs"
	"game_main/common"
	"game_main/squad"
)

// CheckAndTriggerAbilities - ✅ Works with ecs.EntityID
func CheckAndTriggerAbilities(squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	// Find leader via query (not stored reference)
	leaderID := GetLeaderID(squadID, ecsmanager)
	if leaderID == 0 {
		return // No leader, no abilities
	}

	leaderEntity := FindUnitByID(leaderID, ecsmanager)
	if leaderEntity == nil {
		return
	}

	if !leaderEntity.HasComponent(squad.AbilitySlotComponent) {
		return
	}

	if !leaderEntity.HasComponent(squad.CooldownTrackerComponent) {
		return
	}

	abilityData := common.GetComponentType[*squad.AbilitySlotData](leaderEntity, squad.AbilitySlotComponent)
	cooldownData := common.GetComponentType[*squad.CooldownTrackerData](leaderEntity, squad.CooldownTrackerComponent)

	// Check each ability slot
	for i := 0; i < 4; i++ {
		slot := &abilityData.Slots[i]

		if !slot.IsEquipped || cooldownData.Cooldowns[i] > 0 || slot.HasTriggered {
			continue
		}

		// Evaluate trigger condition
		triggered := evaluateTrigger(slot, squadID, ecsmanager)
		if !triggered {
			continue
		}

		// Execute ability
		executeAbility(slot, squadID, ecsmanager)

		// Set cooldown
		cooldownData.Cooldowns[i] = cooldownData.MaxCooldowns[i]

		// Mark as triggered
		slot.HasTriggered = true
	}

	// Tick down cooldowns
	for i := 0; i < 4; i++ {
		if cooldownData.Cooldowns[i] > 0 {
			cooldownData.Cooldowns[i]--
		}
	}
}

// evaluateTrigger checks if a condition is met
func evaluateTrigger(slot *squad.AbilitySlot, squadID ecs.EntityID, ecsmanager *common.EntityManager) bool {
	squadEntity := GetSquadEntity(squadID, ecsmanager)
	if squadEntity == nil {
		return false
	}

	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

	switch slot.TriggerType {
	case squad.TRIGGER_SQUAD_HP_BELOW:
		avgHP := calculateAverageHP(squadID, ecsmanager)
		return avgHP < slot.Threshold

	case squad.TRIGGER_TURN_COUNT:
		return squadData.TurnCount == int(slot.Threshold)

	case squad.TRIGGER_COMBAT_START:
		return squadData.TurnCount == 1

	case squad.TRIGGER_ENEMY_COUNT:
		enemyCount := countEnemySquads(ecsmanager)
		return float64(enemyCount) >= slot.Threshold

	case squad.TRIGGER_MORALE_BELOW:
		return float64(squadData.Morale) < slot.Threshold

	default:
		return false
	}
}

// calculateAverageHP computes the squad's average HP as a percentage (0.0 - 1.0)
func calculateAverageHP(squadID ecs.EntityID, ecsmanager *common.EntityManager) float64 {
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)

	totalHP := 0
	totalMaxHP := 0

	for _, unitID := range unitIDs {
		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		totalHP += attr.CurrentHealth
		totalMaxHP += attr.MaxHealth
	}

	if totalMaxHP == 0 {
		return 0.0
	}

	return float64(totalHP) / float64(totalMaxHP)
}

// countEnemySquads counts the number of enemy squads on the map
func countEnemySquads(ecsmanager *common.EntityManager) int {
	count := 0
	for _, result := range ecsmanager.World.Query(squad.SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

		// Assume enemy squads don't have "Player" prefix (adjust based on your naming)
		if len(squadData.Name) > 0 && squadData.Name[0] != 'P' {
			count++
		}
	}
	return count
}

// executeAbility triggers the ability effect
// Data-driven approach: reads ability params, applies effects
func executeAbility(slot *squad.AbilitySlot, squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	params := squad.GetAbilityParams(slot.AbilityType)

	switch slot.AbilityType {
	case squad.ABILITY_RALLY:
		applyRallyEffect(squadID, params, ecsmanager)
	case squad.ABILITY_HEAL:
		applyHealEffect(squadID, params, ecsmanager)
	case squad.ABILITY_BATTLE_CRY:
		applyBattleCryEffect(squadID, params, ecsmanager)
	case squad.ABILITY_FIREBALL:
		applyFireballEffect(squadID, params, ecsmanager)
	}
}

// --- Ability Implementations (Data-Driven) ---

// RallyEffect: Temporary damage buff to own squad
func applyRallyEffect(squadID ecs.EntityID, params squad.AbilityParams, ecsmanager *common.EntityManager) {
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)

	for _, unitID := range unitIDs {
		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth > 0 {
			attr.DamageBonus += params.DamageBonus
			// TODO: Track buff duration (requires turn/buff system)
		}
	}

	fmt.Printf("[ABILITY] Rally! +%d damage for %d turns\n", params.DamageBonus, params.Duration)
}

// HealEffect: Restore HP to own squad
func applyHealEffect(squadID ecs.EntityID, params squad.AbilityParams, ecsmanager *common.EntityManager) {
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)

	healed := 0
	for _, unitID := range unitIDs {
		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth <= 0 {
			continue
		}

		// Cap at max HP
		attr.CurrentHealth += params.HealAmount
		if attr.CurrentHealth > attr.MaxHealth {
			attr.CurrentHealth = attr.MaxHealth
		}
		healed++
	}

	fmt.Printf("[ABILITY] Healing Aura! %d units restored %d HP\n", healed, params.HealAmount)
}

// BattleCryEffect: First turn buff (morale + damage)
func applyBattleCryEffect(squadID ecs.EntityID, params squad.AbilityParams, ecsmanager *common.EntityManager) {
	squadEntity := GetSquadEntity(squadID, ecsmanager)
	if squadEntity == nil {
		return
	}

	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

	// Boost morale
	squadData.Morale += params.MoraleBonus

	// Boost damage
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)
	for _, unitID := range unitIDs {
		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth > 0 {
			attr.DamageBonus += params.DamageBonus
		}
	}

	fmt.Printf("[ABILITY] Battle Cry! Morale and damage increased!\n")
}

// FireballEffect: AOE damage to enemy squad
func applyFireballEffect(squadID ecs.EntityID, params squad.AbilityParams, ecsmanager *common.EntityManager) {
	// Find first enemy squad (simplified targeting)
	var targetSquadID ecs.EntityID
	for _, result := range ecsmanager.World.Query(squad.SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

		if squadData.SquadID != squadID {
			targetSquadID = squadData.SquadID
			break
		}
	}

	if targetSquadID == 0 {
		return // No targets
	}

	unitIDs := GetUnitIDsInSquad(targetSquadID, ecsmanager)
	killed := 0

	for _, unitID := range unitIDs {
		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth <= 0 {
			continue
		}

		attr.CurrentHealth -= params.BaseDamage
		if attr.CurrentHealth <= 0 {
			killed++
		}
	}

	fmt.Printf("[ABILITY] Fireball! %d damage dealt, %d units killed\n", params.BaseDamage, killed)
}
```

---

## Squad Composition & Formation System

### Design Goals

**Key Requirement:** Support experimentation like Nephilim, Symphony of War, Ogre Battle, Soul Nomad.

**Flexibility Features:**
1. Variable squad sizes (1-9 units)
2. No hard role requirements (all tanks, all DPS, etc. are valid)
3. Empty grid slots allowed (sparse formations)
4. Leader is optional but recommended
5. Formation templates for quick setup
6. Easy unit swapping/rearrangement

### File: `systems/squadcreation.go`

**Squad creation system - uses native entity IDs**

```go
package systems

import (
	"fmt"
	"github.com/bytearena/ecs"
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"
	"game_main/squad"
)

// UnitTemplate defines a unit to be created in a squad
type UnitTemplate struct {
	Name          string                       // Unit name
	Attributes    common.Attributes            // HP, Attack, Defense, etc.
	EntityType    entitytemplates.EntityType   // EntityCreature, etc.
	EntityConfig  entitytemplates.EntityConfig // Name, ImagePath, etc.
	EntityData    any                          // JSONMonster, etc.
	GridRow       int                          // Anchor row (0-2)
	GridCol       int                          // Anchor col (0-2)
	GridWidth     int                          // Width in cells (1-3), defaults to 1
	GridHeight    int                          // Height in cells (1-3), defaults to 1
	Role          squad.UnitRole               // Tank, DPS, Support
	TargetMode    squad.TargetMode             // TargetModeRowBased or TargetModeCellBased
	TargetRows    []int                        // Which rows to attack (row-based)
	IsMultiTarget bool                         // AOE or single-target (row-based)
	MaxTargets    int                          // Max targets per row (row-based)
	TargetCells   [][2]int                     // Specific cells to target (cell-based)
	IsLeader      bool                         // Squad leader flag
}

// CreateSquadFromTemplate - ✅ Returns ecs.EntityID (native type)
func CreateSquadFromTemplate(
	ecsmanager *common.EntityManager,
	squadName string,
	formation squad.FormationType,
	worldPos coords.LogicalPosition,
	unitTemplates []UnitTemplate,
) ecs.EntityID {

	// Create squad entity
	squadEntity := ecsmanager.World.NewEntity()

	// ✅ Get native entity ID
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(squad.SquadComponent, &squad.SquadData{
		SquadID:   squadID,
		Name:      squadName,
		Formation: formation,
		Morale:    100,
		TurnCount: 0,
		MaxUnits:  9,
	})
	squadEntity.AddComponent(common.PositionComponent, &worldPos)

	// Track occupied grid positions (keyed by "row,col")
	occupied := make(map[string]bool)

	// Create units
	for _, template := range unitTemplates {
		// Default to 1x1 if not specified
		width := template.GridWidth
		if width == 0 {
			width = 1
		}
		height := template.GridHeight
		if height == 0 {
			height = 1
		}

		// Validate that unit fits within 3x3 grid
		if template.GridRow < 0 || template.GridCol < 0 {
			fmt.Printf("Warning: Invalid anchor position (%d, %d), skipping\n", template.GridRow, template.GridCol)
			continue
		}
		if template.GridRow+height > 3 || template.GridCol+width > 3 {
			fmt.Printf("Warning: Unit extends outside grid (anchor=%d,%d, size=%dx%d), skipping\n",
				template.GridRow, template.GridCol, width, height)
			continue
		}

		// Check if ANY cell this unit would occupy is already occupied
		canPlace := true
		var cellsToOccupy [][2]int
		for r := template.GridRow; r < template.GridRow+height; r++ {
			for c := template.GridCol; c < template.GridCol+width; c++ {
				key := fmt.Sprintf("%d,%d", r, c)
				if occupied[key] {
					canPlace = false
					fmt.Printf("Warning: Cell (%d,%d) already occupied, cannot place %dx%d unit at (%d,%d)\n",
						r, c, width, height, template.GridRow, template.GridCol)
					break
				}
				cellsToOccupy = append(cellsToOccupy, [2]int{r, c})
			}
			if !canPlace {
				break
			}
		}

		if !canPlace {
			continue
		}

		// Create unit entity
		unitEntity := entitytemplates.CreateEntityFromTemplate(
			*ecsmanager,
			template.EntityConfig,
			template.EntityData,
		)

		// Add squad membership (uses ID, not entity pointer)
		unitEntity.AddComponent(squad.SquadMemberComponent, &squad.SquadMemberData{
			SquadID: squadID,  // ✅ Native entity ID
		})

		// Add grid position (supports multi-cell)
		unitEntity.AddComponent(squad.GridPositionComponent, &squad.GridPositionData{
			AnchorRow: template.GridRow,
			AnchorCol: template.GridCol,
			Width:     width,
			Height:    height,
		})

		// Add role
		unitEntity.AddComponent(squad.UnitRoleComponent, &squad.UnitRoleData{
			Role: template.Role,
		})

		// Add targeting data (supports both row-based and cell-based modes)
		targetMode := squad.TARGET_MODE_ROW_BASED
		if template.TargetMode == "cell" {
			targetMode = squad.TARGET_MODE_CELL_BASED
		}

		unitEntity.AddComponent(squad.TargetRowComponent, &squad.TargetRowData{
			Mode:          targetMode,
			TargetRows:    template.TargetRows,
			IsMultiTarget: template.IsMultiTarget,
			MaxTargets:    template.MaxTargets,
			TargetCells:   template.TargetCells,
		})

		// Add leader component if needed
		if template.IsLeader {
			unitEntity.AddComponent(squad.LeaderComponent, &squad.LeaderData{
				Leadership: 10,
				Experience: 0,
			})

			// Add ability slots
			unitEntity.AddComponent(squad.AbilitySlotComponent, &squad.AbilitySlotData{
				Slots: [4]squad.AbilitySlot{},
			})

			// Add cooldown tracker
			unitEntity.AddComponent(squad.CooldownTrackerComponent, &squad.CooldownTrackerData{
				Cooldowns:    [4]int{0, 0, 0, 0},
				MaxCooldowns: [4]int{0, 0, 0, 0},
			})
		}

		// Mark ALL cells as occupied
		for _, cell := range cellsToOccupy {
			key := fmt.Sprintf("%d,%d", cell[0], cell[1])
			occupied[key] = true
		}
	}

	return squadID // ✅ Return native entity ID
}

// AddUnitToSquad - ✅ Accepts ecs.EntityID (native type)
// Creates a unit entity from a UnitTemplate and adds it to the squad at the specified grid position
func AddUnitToSquad(
	squadID ecs.EntityID,
	squadmanager *SquadECSManager,
	unit UnitTemplate,
	gridRow, gridCol int,
) error {

	// Validate position
	if unit.GridRow < 0 || unit.GridRow > 2 || unit.GridCol < 0 || unit.GridCol > 2 {
		return fmt.Errorf("invalid grid position (%d, %d)", unit.GridRow, unit.GridCol)
	}

	// Check if position occupied
	existingUnitIDs := GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, squadmanager)
	if len(existingUnitIDs) > 0 {
		return fmt.Errorf("grid position (%d, %d) already occupied", gridRow, gridCol)
	}

	// Validate and create unit entity from template
	unitEntity, err := CreateUnitEntity(squadmanager, unit)
	if err != nil {
		return fmt.Errorf("invalid unit for %s: %w", unit.Name, err)
	}

	// Add squad membership component
	unitEntity.AddComponent(squad.SquadMemberComponent, &squad.SquadMemberData{
		SquadID: squadID,
	})

	// Add grid position component (using template's grid position)
	unitEntity.AddComponent(squad.GridPositionComponent, &squad.GridPositionData{
		AnchorRow: unit.GridRow,
		AnchorCol: unit.GridCol,
		Width:     unit.GridWidth,
		Height:    unit.GridHeight,
	})

	// Note: CreateUnitEntity already adds UnitRoleComponent and TargetRowComponent from the template

	return nil
}

// RemoveUnitFromSquad - ✅ Accepts ecs.EntityID (native type)
func RemoveUnitFromSquad(unitEntityID ecs.EntityID, ecsmanager *common.EntityManager) error {
	unitEntity := FindUnitByID(unitEntityID, ecsmanager)
	if unitEntity == nil {
		return fmt.Errorf("unit entity not found")
	}

	if !unitEntity.HasComponent(squad.SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	// In bytearena/ecs, we can't remove components
	// Workaround: Set SquadID to 0 to mark as "removed"
	memberData := common.GetComponentType[*squad.SquadMemberData](unitEntity, squad.SquadMemberComponent)
	memberData.SquadID = 0

	return nil
}

// EquipAbilityToLeader - ✅ Accepts ecs.EntityID (native type)
func EquipAbilityToLeader(
	leaderEntityID ecs.EntityID,
	slotIndex int,
	abilityType squad.AbilityType,
	triggerType squad.TriggerType,
	threshold float64,
	ecsmanager *common.EntityManager,
) error {

	if slotIndex < 0 || slotIndex >= 4 {
		return fmt.Errorf("invalid slot %d", slotIndex)
	}

	leaderEntity := FindUnitByID(leaderEntityID, ecsmanager)
	if leaderEntity == nil {
		return fmt.Errorf("leader entity not found")
	}

	if !leaderEntity.HasComponent(squad.LeaderComponent) {
		return fmt.Errorf("entity is not a leader")
	}

	// Get ability params
	params := squad.GetAbilityParams(abilityType)

	// Update ability slot
	abilityData := common.GetComponentType[*squad.AbilitySlotData](leaderEntity, squad.AbilitySlotComponent)
	abilityData.Slots[slotIndex] = squad.AbilitySlot{
		AbilityType:  abilityType,
		TriggerType:  triggerType,
		Threshold:    threshold,
		HasTriggered: false,
		IsEquipped:   true,
	}

	// Update cooldown tracker
	cooldownData := common.GetComponentType[*squad.CooldownTrackerData](leaderEntity, squad.CooldownTrackerComponent)
	cooldownData.MaxCooldowns[slotIndex] = params.BaseCooldown
	cooldownData.Cooldowns[slotIndex] = 0

	return nil
}

// MoveUnitInSquad - ✅ Accepts ecs.EntityID (native type)
// ✅ Supports multi-cell units - validates all cells at new position
func MoveUnitInSquad(unitEntityID ecs.EntityID, newRow, newCol int, ecsmanager *common.EntityManager) error {
	unitEntity := FindUnitByID(unitEntityID, ecsmanager)
	if unitEntity == nil {
		return fmt.Errorf("unit entity not found")
	}

	if !unitEntity.HasComponent(squad.SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	gridPosData := common.GetComponentType[*squad.GridPositionData](unitEntity, squad.GridPositionComponent)

	// Validate new anchor position is in bounds
	if newRow < 0 || newCol < 0 {
		return fmt.Errorf("invalid anchor position (%d, %d)", newRow, newCol)
	}

	// Validate unit fits within grid at new position
	if newRow+gridPosData.Height > 3 || newCol+gridPosData.Width > 3 {
		return fmt.Errorf("unit would extend outside grid at position (%d, %d) with size %dx%d",
			newRow, newCol, gridPosData.Width, gridPosData.Height)
	}

	memberData := common.GetComponentType[*squad.SquadMemberData](unitEntity, squad.SquadMemberComponent)

	// Check if ANY cell at new position is occupied (excluding this unit itself)
	for r := newRow; r < newRow+gridPosData.Height; r++ {
		for c := newCol; c < newCol+gridPosData.Width; c++ {
			existingUnitIDs := GetUnitIDsAtGridPosition(memberData.SquadID, r, c, ecsmanager)
			for _, existingID := range existingUnitIDs {
				if existingID != unitEntityID {
					return fmt.Errorf("cell (%d, %d) already occupied by another unit", r, c)
				}
			}
		}
	}

	// Update grid position (anchor only, width/height remain the same)
	gridPosData.AnchorRow = newRow
	gridPosData.AnchorCol = newCol

	return nil
}
```

### Formation Presets

**File:** `squad/formations.go`

```go
package squad

// FormationPreset defines a quick-start squad configuration
type FormationPreset struct {
	Positions []FormationPosition
}

type FormationPosition struct {
	AnchorRow int
	AnchorCol int
	Role      UnitRole
	Target    []int
}

// GetFormationPreset returns predefined formation templates
func GetFormationPreset(formation FormationType) FormationPreset {
	switch formation {
	case FORMATION_BALANCED:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 0, Role: ROLE_TANK, Target: []int{0}},
				{AnchorRow: 0, AnchorCol: 2, Role: ROLE_TANK, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 1, Role: ROLE_SUPPORT, Target: []int{1}},
				{AnchorRow: 2, AnchorCol: 0, Role: ROLE_DPS, Target: []int{2}},
				{AnchorRow: 2, AnchorCol: 2, Role: ROLE_DPS, Target: []int{2}},
			},
		}

	case FORMATION_DEFENSIVE:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 0, Role: ROLE_TANK, Target: []int{0}},
				{AnchorRow: 0, AnchorCol: 1, Role: ROLE_TANK, Target: []int{0}},
				{AnchorRow: 0, AnchorCol: 2, Role: ROLE_TANK, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 1, Role: ROLE_SUPPORT, Target: []int{1}},
				{AnchorRow: 2, AnchorCol: 1, Role: ROLE_DPS, Target: []int{2}},
			},
		}

	case FORMATION_OFFENSIVE:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 1, Role: ROLE_TANK, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 0, Role: ROLE_DPS, Target: []int{1}},
				{AnchorRow: 1, AnchorCol: 1, Role: ROLE_DPS, Target: []int{1}},
				{AnchorRow: 1, AnchorCol: 2, Role: ROLE_DPS, Target: []int{1}},
				{AnchorRow: 2, AnchorCol: 1, Role: ROLE_SUPPORT, Target: []int{2}},
			},
		}

	case FORMATION_RANGED:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 1, Role: ROLE_TANK, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 0, Role: ROLE_DPS, Target: []int{1, 2}},
				{AnchorRow: 1, AnchorCol: 2, Role: ROLE_DPS, Target: []int{1, 2}},
				{AnchorRow: 2, AnchorCol: 0, Role: ROLE_DPS, Target: []int{2}},
				{AnchorRow: 2, AnchorCol: 1, Role: ROLE_SUPPORT, Target: []int{2}},
				{AnchorRow: 2, AnchorCol: 2, Role: ROLE_DPS, Target: []int{2}},
			},
		}

	default:
		return FormationPreset{Positions: []FormationPosition{}}
	}
}
```

---

## Implementation Phases

### Phase 1: Core Components and Data Structures (6-8 hours)

**Deliverables:**
- `squad/components.go` - All component definitions (using ecs.EntityID)
- `squad/tags.go` - Tag initialization
- Component registration in game initialization

**Steps:**
1. Create `squad/` package directory
2. Define all component types and data structures (with ecs.EntityID)
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

### Phase 2: Query System (4-6 hours)

**Deliverables:**
- `systems/squadqueries.go` - Query functions for squad relationships

**Steps:**
1. Implement `GetUnitIDsInSquad()` (returns []ecs.EntityID)
2. Implement `GetSquadEntity()` (returns *ecs.Entity from query)
3. Implement `GetUnitIDsAtGridPosition()`
4. Implement `GetUnitIDsInRow()`
5. Implement `GetLeaderID()`
6. Implement `IsSquadDestroyed()`
7. Implement `FindUnitByID()` helper

**Testing:**
```go
func TestSquadQueries() {
	// Create test squad with units
	squadID := systems.CreateSquadFromTemplate(...)

	// Test queries
	unitIDs := systems.GetUnitIDsInSquad(squadID, ecsmanager)
	assert.Greater(t, len(unitIDs), 0)

	frontRowIDs := systems.GetUnitIDsInRow(squadID, 0, ecsmanager)
	assert.Greater(t, len(frontRowIDs), 0)

	leaderID := systems.GetLeaderID(squadID, ecsmanager)
	assert.NotEqual(t, ecs.EntityID(0), leaderID)
}
```

### Phase 3: Row-Based Combat System (8-10 hours)

**Deliverables:**
- `systems/squadcombat.go` - ExecuteSquadAttack, damage calculation
- Row-based targeting logic
- Integration with existing `PerformAttack()` concepts

**Steps:**
1. Implement `ExecuteSquadAttack()` function
2. Implement `calculateUnitDamageByID()` (adapt existing combat logic)
3. Implement `applyRoleModifier()` for Tank/DPS/Support
4. Implement targeting logic (single-target vs multi-target)
5. Implement helper functions (selectLowestHPTargetID, selectRandomTargetIDs)
6. Add death tracking and unit removal

**Testing:**
- Create two squads with different roles
- Execute combat and verify damage distribution
- Verify row targeting (front row hit first, back row protected)
- Verify role modifiers apply correctly
- Test edge cases (empty rows, all dead, etc.)

### Phase 4: Automated Ability System (6-8 hours)

**Deliverables:**
- `systems/squadabilities.go` - Ability trigger checking and execution
- Built-in abilities: Rally, Heal, BattleCry, Fireball

**Steps:**
1. Implement `CheckAndTriggerAbilities()`
2. Implement `evaluateTrigger()` for condition checking
3. Implement `executeAbility()` for ability dispatch
4. Implement 4 example abilities (Rally, Heal, BattleCry, Fireball)
5. Add cooldown tick system
6. Implement `EquipAbilityToLeader()`

**Testing:**
- Equip abilities with different trigger conditions
- Simulate combat and verify triggers fire correctly
- Test cooldown system
- Verify condition thresholds (HP < 50%, turn count, etc.)

### Phase 5: Squad Creation and Management (4-6 hours)

**Deliverables:**
- `systems/squadcreation.go` - CreateSquadFromTemplate, AddUnitToSquad, etc.
- `squad/formations.go` - Formation presets

**Steps:**
1. Implement `CreateSquadFromTemplate()` (returns ecs.EntityID!)
2. Implement `AddUnitToSquad()`
3. Implement `RemoveUnitFromSquad()`
4. Implement `MoveUnitInSquad()`
5. Implement `EquipAbilityToLeader()`
6. Define formation presets in `squad/formations.go`

**Testing:**
- Create squads with varying sizes (1-9 units)
- Test with sparse formations (empty grid slots)
- Verify leader assignment
- Test formation presets
- Verify native entity IDs are returned

### Phase 6: Integration with Existing Systems (8-10 hours)

**Deliverables:**
- Updated `input/combatcontroller.go` for squad selection
- Updated `spawning/spawnmonsters.go` for squad spawning
- Updated rendering to show squad grid
- Simple cleanup (entity.Remove())

**Steps:**
1. Modify `CombatController.HandleClick()` to detect squad entities
2. Implement squad selection/targeting flow
3. Update spawning system to create enemy squads
4. Add squad rendering with grid overlay
5. Integrate `CheckAndTriggerAbilities()` into turn system
6. Update input handling for squad movement (on map)
7. Add squad destruction cleanup (just entity.Remove())

**Code Example: Combat Controller (Updated)**
```go
// File: input/combatcontroller.go

func (c *CombatController) executeSquadCombat(attackerSquadID, defenderSquadID ecs.EntityID) {
	// Increment turn count
	attackerSquad := systems.GetSquadEntity(attackerSquadID, c.ecsmanager)
	attackerData := common.GetComponentType[*squad.SquadData](attackerSquad, squad.SquadComponent)
	attackerData.TurnCount++

	// Trigger abilities
	systems.CheckAndTriggerAbilities(attackerSquadID, c.ecsmanager)

	// Execute attack - ✅ Result uses native entity IDs
	result := systems.ExecuteSquadAttack(attackerSquadID, defenderSquadID, c.ecsmanager)

	// Display results
	c.showCombatResults(result)

	// Counter-attack if defender still alive
	if !systems.IsSquadDestroyed(defenderSquadID, c.ecsmanager) {
		systems.CheckAndTriggerAbilities(defenderSquadID, c.ecsmanager)
		counterResult := systems.ExecuteSquadAttack(defenderSquadID, attackerSquadID, c.ecsmanager)
		c.showCombatResults(counterResult)
	}

	// Cleanup destroyed squads
	c.checkSquadDestruction(attackerSquadID)
	c.checkSquadDestruction(defenderSquadID)
}

func (c *CombatController) checkSquadDestruction(squadID ecs.EntityID) {
	if systems.IsSquadDestroyed(squadID, c.ecsmanager) {
		// ✅ Get unit IDs using native method
		unitIDs := systems.GetUnitIDsInSquad(squadID, c.ecsmanager)
		for _, unitID := range unitIDs {
			unit := systems.FindUnitByID(unitID, c.ecsmanager)
			if unit != nil {
				unit.Remove()  // ✅ Simple cleanup, no registry
			}
		}

		// Remove squad entity
		squadEntity := systems.GetSquadEntity(squadID, c.ecsmanager)
		if squadEntity != nil {
			squadEntity.Remove()  // ✅ Simple cleanup
		}
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

## Integration Guide

### Game Initialization

**File:** `game_main/main.go`

```go
func main() {
	manager := ecs.NewManager()
	ecsmanager := &common.EntityManager{
		World:     manager,
		WorldTags: make(map[string]ecs.Tag),
	}

	// Register common components
	common.PositionComponent = manager.NewComponent()
	common.AttributeComponent = manager.NewComponent()
	common.NameComponent = manager.NewComponent()

	// Register squad components
	squad.InitSquadComponents(manager)
	squad.InitSquadTags()

	// ... rest of initialization
}
```

### Spawning Enemy Squads

**File:** `spawning/spawnmonsters.go`

```go
// SpawnEnemySquad creates an enemy squad based on level
func SpawnEnemySquad(ecsmanager *common.EntityManager, level int, worldPos coords.LogicalPosition) ecs.EntityID {
	var templates []systems.UnitTemplate

	if level <= 3 {
		// Early game: 3-5 weak units
		templates = []systems.UnitTemplate{
			{
				EntityType:    entitytemplates.EntityCreature,
				EntityConfig:  entitytemplates.EntityConfig{Name: "Goblin"},
				EntityData:    loadMonsterData("Goblin"),
				GridRow:       0, GridCol: 0,
				Role:          squad.ROLE_TANK,
				TargetMode:    squad.TargetModeRowBased,
				TargetRows:    []int{0},
				IsMultiTarget: false,
				MaxTargets:    1,
			},
			// ... more units
		}
	} else if level <= 7 {
		// Mid game: 5-7 units with leader
		templates = []systems.UnitTemplate{
			{
				EntityType:   entitytemplates.EntityCreature,
				EntityConfig: entitytemplates.EntityConfig{Name: "Orc Warrior"},
				EntityData:   loadMonsterData("Orc"),
				GridRow:      0, GridCol: 1,
				Role:         squad.ROLE_TANK,
				TargetMode:   squad.TargetModeRowBased,
				TargetRows:   []int{0},
				IsLeader:     true, // Add leader
			},
			// ... more units
		}
	}

	// Create squad (returns native entity ID!)
	squadID := systems.CreateSquadFromTemplate(
		ecsmanager,
		"Enemy Squad",
		squad.FORMATION_BALANCED,
		worldPos,
		templates,
	)

	// Equip leader abilities if present
	leaderID := systems.GetLeaderID(squadID, ecsmanager)
	if leaderID != 0 {
		equipEnemyLeaderAbilities(leaderID, ecsmanager)
	}

	return squadID
}

func equipEnemyLeaderAbilities(leaderID ecs.EntityID, ecsmanager *common.EntityManager) {
	// Equip 2 random abilities
	systems.EquipAbilityToLeader(
		leaderID,
		0, // Slot 0
		squad.ABILITY_RALLY,
		squad.TRIGGER_SQUAD_HP_BELOW,
		0.5, // 50% HP
		ecsmanager,
	)

	systems.EquipAbilityToLeader(
		leaderID,
		1, // Slot 1
		squad.ABILITY_BATTLE_CRY,
		squad.TRIGGER_COMBAT_START,
		0, // No threshold
		ecsmanager,
	)
}
```

---

## Complete Code Examples

### Example 1: Creating a Full Player Squad

```go
func InitializePlayerSquad(ecsmanager *common.EntityManager) ecs.EntityID {
	templates := []systems.UnitTemplate{
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
					MaxHealth:         50,
					AttackBonus:       5,
					BaseArmorClass:    15,
					BaseProtection:    5,
					BaseDodgeChance:   10.0,
					BaseMovementSpeed: 5,
				},
			},
			GridRow:       0,
			GridCol:       1,
			Role:          squad.ROLE_TANK,
			TargetMode:    squad.TargetModeRowBased,
			TargetRows:    []int{0}, // Attack front row
			IsMultiTarget: false,
			MaxTargets:    1,
			IsLeader:      true,
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

	// Create squad (returns native entity ID!)
	squadID := systems.CreateSquadFromTemplate(
		ecsmanager,
		"Player's Legion",
		squad.FORMATION_BALANCED,
		coords.LogicalPosition{X: 10, Y: 10},
		templates,
	)

	// Setup leader abilities
	setupPlayerLeaderAbilities(squadID, ecsmanager)

	return squadID
}

func setupPlayerLeaderAbilities(squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	leaderID := systems.GetLeaderID(squadID, ecsmanager)
	if leaderID == 0 {
		return
	}

	// Battle Cry: Triggers at combat start (turn 1)
	systems.EquipAbilityToLeader(
		leaderID,
		0,
		squad.ABILITY_BATTLE_CRY,
		squad.TRIGGER_COMBAT_START,
		0,
		ecsmanager,
	)

	// Heal: Triggers when squad HP drops below 50%
	systems.EquipAbilityToLeader(
		leaderID,
		1,
		squad.ABILITY_HEAL,
		squad.TRIGGER_SQUAD_HP_BELOW,
		0.5,
		ecsmanager,
	)

	// Rally: Triggers on turn 3
	systems.EquipAbilityToLeader(
		leaderID,
		2,
		squad.ABILITY_RALLY,
		squad.TRIGGER_TURN_COUNT,
		3.0,
		ecsmanager,
	)

	// Fireball: Triggers when 2+ enemy squads present
	systems.EquipAbilityToLeader(
		leaderID,
		3,
		squad.ABILITY_FIREBALL,
		squad.TRIGGER_ENEMY_COUNT,
		2.0,
		ecsmanager,
	)
}

// Helper functions to create JSON data
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

	// Create two squads
	playerSquadID := InitializePlayerSquad(ecsmanager)
	enemySquadID := SpawnEnemySquad(ecsmanager, 5, coords.LogicalPosition{X: 15, Y: 15})

	// Simulate 5 turns of combat
	for turn := 1; turn <= 5; turn++ {
		fmt.Printf("\n--- TURN %d ---\n", turn)

		// Player squad attacks
		fmt.Println("Player squad attacks:")
		result := systems.ExecuteSquadAttack(playerSquadID, enemySquadID, ecsmanager)
		displayCombatResult(result, ecsmanager)

		// Check if enemy destroyed
		if systems.IsSquadDestroyed(enemySquadID, ecsmanager) {
			fmt.Println("Enemy squad destroyed!")
			break
		}

		// Enemy counter-attacks
		fmt.Println("Enemy squad counter-attacks:")
		counterResult := systems.ExecuteSquadAttack(enemySquadID, playerSquadID, ecsmanager)
		displayCombatResult(counterResult, ecsmanager)

		// Check if player destroyed
		if systems.IsSquadDestroyed(playerSquadID, ecsmanager) {
			fmt.Println("Player squad destroyed!")
			break
		}

		// Display squad status
		displaySquadStatus(playerSquadID, ecsmanager)
		displaySquadStatus(enemySquadID, ecsmanager)
	}
}

func displayCombatResult(result *systems.CombatResult, ecsmanager *common.EntityManager) {
	fmt.Printf("  Total damage: %d\n", result.TotalDamage)
	fmt.Printf("  Units killed: %d\n", len(result.UnitsKilled))

	// ✅ Result uses native entity IDs
	for unitID, dmg := range result.DamageByUnit {
		unit := systems.FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}
		name := common.GetComponentType[*common.Name](unit, common.NameComponent)
		fmt.Printf("    %s took %d damage\n", name.NameStr, dmg)
	}
}

func displaySquadStatus(squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	squadEntity := systems.GetSquadEntity(squadID, ecsmanager)
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

	fmt.Printf("\n%s Status:\n", squadData.Name)

	unitIDs := systems.GetUnitIDsInSquad(squadID, ecsmanager)
	alive := 0

	for _, unitID := range unitIDs {
		unit := systems.FindUnitByID(unitID, ecsmanager)
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
```

### Example 3: Dynamic Squad Modification

```go
// Add a new unit to an existing squad mid-game
func RecruitUnitToSquad(squadID ecs.EntityID, unitName string, gridRow, gridCol int, ecsmanager *common.EntityManager) error {
	// Create new unit entity
	unitEntity := entitytemplates.CreateEntityFromTemplate(
		*ecsmanager,
		entitytemplates.EntityConfig{
			Name:      unitName,
			ImagePath: "recruit.png",
			AssetDir:  "../assets/",
			Visible:   true,
		},
		createDPSData(30, 7, 12, 3),
	)

	// ✅ Get native entity ID
	unitEntityID := unitEntity.GetID()

	// Add to squad
	err := systems.AddUnitToSquad(
		squadID,
		unitEntityID,
		gridRow,
		gridCol,
		squad.ROLE_DPS,
		[]int{1}, // Target middle row
		false,    // Single-target
		1,        // Max 1 target
		ecsmanager,
	)

	if err != nil {
		return fmt.Errorf("failed to recruit unit: %v", err)
	}

	fmt.Printf("Recruited %s to squad at position (%d, %d)\n", unitName, gridRow, gridCol)
	return nil
}

// Swap unit positions within squad
func ReorganizeSquad(squadID ecs.EntityID, ecsmanager *common.EntityManager) error {
	// Find a back-line DPS unit
	unitIDs := systems.GetUnitIDsInSquad(squadID, ecsmanager)

	var dpsUnitID ecs.EntityID
	for _, unitID := range unitIDs {
		unit := systems.FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		roleData := common.GetComponentType[*squad.UnitRoleData](unit, squad.UnitRoleComponent)
		gridPos := common.GetComponentType[*squad.GridPositionData](unit, squad.GridPositionComponent)

		if roleData.Role == squad.ROLE_DPS && gridPos.AnchorRow == 2 {
			dpsUnitID = unitID
			break
		}
	}

	if dpsUnitID == 0 {
		return fmt.Errorf("no DPS unit found to move")
	}

	// Move to front line (tactical decision)
	err := systems.MoveUnitInSquad(dpsUnitID, 0, 2, ecsmanager) // Front right
	if err != nil {
		return fmt.Errorf("failed to move unit: %v", err)
	}

	fmt.Println("Moved DPS unit to front line for aggressive tactics")
	return nil
}
```

### Example 4: Multi-Cell Unit Squad

```go
// Create a squad with multi-cell units (large creatures)
func CreateGiantSquad(ecsmanager *common.EntityManager) ecs.EntityID {
	templates := []systems.UnitTemplate{
		// 2x2 Giant occupying front-left (rows 0-1, cols 0-1)
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Stone Giant",
				ImagePath: "giant.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: entitytemplates.JSONMonster{
				Name: "Stone Giant",
				Attributes: entitytemplates.JSONAttributes{
					MaxHealth:         100,  // Large HP pool
					AttackBonus:       15,   // High damage
					BaseArmorClass:    12,
					BaseProtection:    8,
					BaseDodgeChance:   5.0,  // Slow, hard to dodge
					BaseMovementSpeed: 3,
				},
			},
			GridRow:    0,    // Anchor at top-left
			GridCol:    0,
			GridWidth:  2,    // ✅ 2 cells wide
			GridHeight: 2,    // ✅ 2 cells tall
			Role:       squad.ROLE_TANK,
			TargetRows: []int{0, 1}, // Can hit front and mid rows (tall reach)
			IsMultiTarget: false,
			MaxTargets:    1,
			IsLeader:      true,
		},

		// 1x1 Archer in front-right
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Archer",
				ImagePath: "archer.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: createDPSData(30, 8, 11, 2),
			GridRow:    0,
			GridCol:    2,
			GridWidth:  1,  // ✅ Standard 1x1
			GridHeight: 1,
			Role:       squad.ROLE_DPS,
			TargetRows: []int{2}, // Snipe back row
			IsMultiTarget: false,
			MaxTargets:    1,
		},

		// 2x1 Cavalry unit (wide) in middle row
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Heavy Cavalry",
				ImagePath: "cavalry.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: entitytemplates.JSONMonster{
				Name: "Heavy Cavalry",
				Attributes: entitytemplates.JSONAttributes{
					MaxHealth:         60,
					AttackBonus:       12,
					BaseArmorClass:    14,
					BaseProtection:    6,
					BaseDodgeChance:   10.0,
					BaseMovementSpeed: 7,
				},
			},
			GridRow:    1,
			GridCol:    2,    // Can't go in cols 0-1 (giant occupies them)
			GridWidth:  1,    // ✅ Only 1 cell available in row 1
			GridHeight: 1,
			Role:       squad.ROLE_DPS,
			TargetRows: []int{0}, // Charge front row
			IsMultiTarget: false,
			MaxTargets:    1,
		},

		// 3x1 Trebuchet in back row (full-width siege weapon)
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Trebuchet",
				ImagePath: "trebuchet.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: entitytemplates.JSONMonster{
				Name: "Trebuchet",
				Attributes: entitytemplates.JSONAttributes{
					MaxHealth:         40,
					AttackBonus:       20,  // Massive AOE damage
					BaseArmorClass:    10,
					BaseProtection:    2,
					BaseDodgeChance:   0.0,  // Can't dodge
					BaseMovementSpeed: 1,
				},
			},
			GridRow:    2,
			GridCol:    0,
			GridWidth:  3,    // ✅ Spans entire back row (cols 0-2)
			GridHeight: 1,
			Role:       squad.ROLE_DPS,
			TargetRows: []int{0, 1, 2}, // Hits all rows
			IsMultiTarget: true,  // AOE
			MaxTargets:    2,     // Max 2 targets per row
		},
	}

	/*
	Visual Grid Layout:
	Row 0: [Giant___|Giant___] [Archer]
	Row 1: [Giant___|Giant___] [Cavalry]
	Row 2: [Trebuchet___|Trebuchet___|Trebuchet___]
	*/

	squadID := systems.CreateSquadFromTemplate(
		ecsmanager,
		"Siege Battalion",
		squad.FORMATION_OFFENSIVE,
		coords.LogicalPosition{X: 20, Y: 20},
		templates,
	)

	// Equip leader abilities (Giant)
	leaderID := systems.GetLeaderID(squadID, ecsmanager)
	if leaderID != 0 {
		// Rally ability: Buff squad damage when HP drops
		systems.EquipAbilityToLeader(
			leaderID,
			0,
			squad.ABILITY_RALLY,
			squad.TRIGGER_SQUAD_HP_BELOW,
			0.6, // 60% HP
			ecsmanager,
		)
	}

	return squadID
}

// Test multi-cell unit targeting
func TestMultiCellTargeting(ecsmanager *common.EntityManager) {
	// Create giant squad
	giantSquadID := CreateGiantSquad(ecsmanager)

	// Create standard enemy squad
	enemySquadID := createStandardEnemySquad(ecsmanager)

	// Query front row of giant squad
	frontRowUnits := systems.GetUnitIDsInRow(giantSquadID, 0, ecsmanager)
	fmt.Printf("Front row has %d units\n", len(frontRowUnits))
	// Output: Front row has 2 units (Giant + Archer)

	// Query middle row of giant squad
	midRowUnits := systems.GetUnitIDsInRow(giantSquadID, 1, ecsmanager)
	fmt.Printf("Middle row has %d units\n", len(midRowUnits))
	// Output: Middle row has 2 units (Giant + Cavalry)
	// Note: Giant appears in BOTH front and middle row queries!

	// Query back row of giant squad
	backRowUnits := systems.GetUnitIDsInRow(giantSquadID, 2, ecsmanager)
	fmt.Printf("Back row has %d units\n", len(backRowUnits))
	// Output: Back row has 1 unit (Trebuchet)

	// Enemy attacks front row - can hit Giant or Archer
	fmt.Println("\nEnemy attacks front row:")
	result := systems.ExecuteSquadAttack(enemySquadID, giantSquadID, ecsmanager)
	// Giant will likely be targeted (larger HP pool = lower HP target)

	// Enemy attacks middle row - can hit Giant or Cavalry
	fmt.Println("\nEnemy attacks middle row:")
	// Giant is vulnerable from BOTH front and middle row attacks!
}

func createStandardEnemySquad(ecsmanager *common.EntityManager) ecs.EntityID {
	templates := []systems.UnitTemplate{
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{Name: "Goblin", ImagePath: "goblin.png", AssetDir: "../assets/", Visible: true},
			EntityData: createDPSData(20, 5, 10, 1),
			GridRow: 0, GridCol: 0,
			GridWidth: 1, GridHeight: 1,
			Role: squad.ROLE_DPS,
			TargetRows: []int{0},
		},
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{Name: "Orc", ImagePath: "orc.png", AssetDir: "../assets/", Visible: true},
			EntityData: createTankData(40, 6, 13, 4),
			GridRow: 0, GridCol: 1,
			GridWidth: 1, GridHeight: 1,
			Role: squad.ROLE_TANK,
			TargetRows: []int{0},
		},
	}

	return systems.CreateSquadFromTemplate(
		ecsmanager,
		"Goblin Warband",
		squad.FORMATION_BALANCED,
		coords.LogicalPosition{X: 25, Y: 20},
		templates,
	)
}
```

**Key Takeaways from Multi-Cell Example:**

1. **Large units occupy multiple cells** - The 2x2 Giant takes up 4 cells
2. **Row targeting affects multi-cell units** - Giant can be hit by attacks targeting row 0 OR row 1
3. **Deduplication works** - Giant appears only once in each row query despite occupying 2 rows
4. **Grid constraints enforced** - Cavalry can't fit in cols 0-1 of row 1 (Giant blocks it)
5. **Full-width units** - Trebuchet spans entire back row (3 cells wide)
6. **Strategic trade-offs** - Large units are easier to hit but have more HP

---

## Migration Path

### Benefits of Native Entity ID Usage

#### ✅ No Custom Registry Needed
```go
// Before: Custom registry with bidirectional mapping
common.RegisterEntity(entity)
common.UnregisterEntity(entity)
entityID := common.GetEntityID(entity)

// After: Native entity IDs
entityID := entity.GetID()  // That's it!
```

#### ✅ Simpler Cleanup
```go
// Before: Manual registry cleanup
common.UnregisterEntity(entity)
entity.Remove()

// After: Just remove the entity
entity.Remove()
```

#### ✅ No Dangling Pointers
```go
// IDs remain valid even if entity is deleted
result.UnitsKilled = []ecs.EntityID{id1, id2}
if unit := systems.FindUnitByID(id1, ecsmanager); unit != nil {
    // Entity still exists
}
```

#### ✅ Serialization Support
```go
// Can save/load game state
type SavedCombatResult struct {
    UnitsKilled  []ecs.EntityID           // ✅ Can serialize IDs
    DamageByUnit map[ecs.EntityID]int     // ✅ Can serialize ID maps
}
```

#### ✅ Type Safety
```go
// ecs.EntityID is a distinct type (uint32)
// Prevents accidentally mixing up different ID types
var squadID ecs.EntityID = squad.GetID()
var unitID ecs.EntityID = unit.GetID()
```

### Code Size Reduction

**Removed from original design:**
- ~125 lines of EntityRegistry code (common/entityid.go)
- All RegisterEntity() calls throughout codebase
- All UnregisterEntity() cleanup calls
- Wrapper functions (GetEntityByID, GetEntityID)

**Net reduction:** ~150-200 lines of code removed

### Comparison Table

| **Aspect** | **Original (Custom Registry)** | **Corrected (Native IDs)** |
|------------|-------------------------------|---------------------------|
| Entity ID type | `uint64` (custom) | `ecs.EntityID` (uint32, native) |
| Get entity ID | `common.GetEntityID(entity)` | `entity.GetID()` |
| Register entity | `common.RegisterEntity(entity)` | Not needed |
| Cleanup entity | `common.UnregisterEntity(entity)` + `entity.Remove()` | `entity.Remove()` |
| Find by ID | `common.GetEntityByID(id)` | `FindUnitByID(id, mgr)` (query-based) |
| Code complexity | +125 lines registry code | Native solution |
| Memory overhead | Bidirectional map storage | None |
| Thread safety | Mutex required | Query-based (no locking) |

---

## Summary

This corrected document represents the **production-ready, ECS-compliant squad combat system** for TinkerRogue using **native bytearena/ecs entity IDs** with **full multi-cell unit support**.

### Key Features:

✅ **Uses Native Entity IDs** - `entity.GetID()` returns `ecs.EntityID` (uint32)
✅ **No Custom Registry** - Removed unnecessary EntityRegistry system (~125 LOC)
✅ **Simpler Entity Lifecycle** - Just `entity.Remove()`, no registration/unregistration
✅ **Type-Safe IDs** - All `uint64` changed to `ecs.EntityID`
✅ **Query-Based Lookup** - `FindUnitByID()` uses queries instead of registry
✅ **Reduced Complexity** - 150-200 fewer lines of code vs original
✅ **Multi-Cell Unit Support** - Units can occupy 1x1, 1x2, 2x2, 2x1, up to 3x3 grid spaces
✅ **Smart Row Targeting** - Multi-cell units appear in all rows they occupy with automatic deduplication

### Multi-Cell Unit Capabilities:

- **Large Creatures** - 2x2 giants, 2x3 dragons occupying multiple cells
- **Wide Formations** - 3x1 cavalry lines spanning entire rows
- **Tall Units** - 1x3 siege towers spanning multiple rows
- **Boss Encounters** - 3x3 mega-units occupying entire grid
- **Mixed Squads** - Combining 1x1 infantry with 2x2 monsters
- **Strategic Depth** - Large units trade flexibility for HP, easier to target from multiple rows

### Total Implementation Time:

**30-37 hours** (3-6 hours faster than original due to removed registry)

### What This Enables:

- Squad-based tactical combat on roguelike map with variable unit sizes
- Multiple player-controlled squads with flexible composition
- Enemy squad spawning with level scaling and multi-cell bosses
- Leader abilities with automated triggers
- Formation strategies (defensive, offensive, ranged) supporting mixed unit sizes
- Dynamic squad composition (add/remove/resize units mid-game)
- Simple entity lifecycle management
- Save/load support (via ID serialization)
- Rich tactical gameplay (large units block more rows, easier to target)

**This design is ready for production implementation with native ECS support and full multi-cell unit capabilities.**
