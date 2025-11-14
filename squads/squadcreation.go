package squads

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"

	"github.com/bytearena/ecs"
)

// ========================================
// SQUAD RELATED
// ========================================

func CreateEmptySquad(squadmanager *common.EntityManager,
	squadName string) {

	squadEntity := squadmanager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(SquadComponent, &SquadData{
		SquadID:       squadID,
		Name:          squadName,
		Morale:        100,
		TurnCount:     0,
		MaxUnits:      9,
		UsedCapacity:  0.0,
		TotalCapacity: 6, // Default capacity (no leader yet)
	})

	squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{})

}

// gridRow and gridCol are the row and col we want to anchor the unit at
func AddUnitToSquad(
	squadID ecs.EntityID,
	squadmanager *common.EntityManager,
	unit UnitTemplate,
	gridRow, gridCol int) error {

	// Validate position using the provided parameters, not unit template values
	if gridRow < 0 || gridRow > 2 || gridCol < 0 || gridCol > 2 {
		return fmt.Errorf("invalid grid position (%d, %d)", gridRow, gridCol)
	}

	// Check if position occupied
	existingUnitIDs := GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, squadmanager)
	if len(existingUnitIDs) > 0 {
		return fmt.Errorf("grid position (%d, %d) already occupied", gridRow, gridCol)
	}

	// Check capacity before adding unit
	unitCapacityCost := unit.Attributes.GetCapacityCost()
	if !CanAddUnitToSquad(squadID, unitCapacityCost, squadmanager) {
		remaining := GetSquadRemainingCapacity(squadID, squadmanager)
		return fmt.Errorf("insufficient squad capacity: need %.2f, have %.2f remaining (unit %s costs %.2f)",
			unitCapacityCost, remaining, unit.Name, unitCapacityCost)
	}

	// Create unit entity (adds GridPositionComponent with default 0,0)
	unitEntity, err := CreateUnitEntity(squadmanager, unit)
	if err != nil {
		return fmt.Errorf("invalid unit for %s: %w", unit.Name, err)
	}

	// Add SquadMemberComponent to link unit to squad
	unitEntity.AddComponent(SquadMemberComponent, &SquadMemberData{
		SquadID: squadID,
	})

	// Update GridPositionComponent with actual grid position
	gridPos := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent)
	gridPos.AnchorRow = gridRow
	gridPos.AnchorCol = gridCol

	// Update squad capacity tracking
	UpdateSquadCapacity(squadID, squadmanager)

	return nil
}

// RemoveUnitFromSquad - ✅ Accepts ecs.EntityID (native type)
func RemoveUnitFromSquad(unitEntityID ecs.EntityID, squadmanager *common.EntityManager) error {
	unitEntity := common.FindEntityByIDWithTag(squadmanager, unitEntityID, SquadMemberTag)
	if unitEntity == nil {
		return fmt.Errorf("unit entity not found")
	}

	if !unitEntity.HasComponent(SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	// Get the squad ID before removing to update capacity
	memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)
	squadID := memberData.SquadID

	// In bytearena/ecs, we can't remove components
	// Workaround: Set SquadID to 0 to mark as "removed"
	memberData.SquadID = 0

	// Update squad capacity tracking after removal
	UpdateSquadCapacity(squadID, squadmanager)

	return nil
}

// MoveUnitInSquad - ✅ Accepts ecs.EntityID (native type)
// ✅ Supports multi-cell units - validates all cells at new position
func MoveUnitInSquad(unitEntityID ecs.EntityID, newRow, newCol int, ecsmanager *common.EntityManager) error {
	unitEntity := common.FindEntityByIDWithTag(ecsmanager, unitEntityID, SquadMemberTag)
	if unitEntity == nil {
		return fmt.Errorf("unit entity not found")
	}

	if !unitEntity.HasComponent(SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	gridPosData := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent)

	// Validate new anchor position is in bounds
	if newRow < 0 || newCol < 0 {
		return fmt.Errorf("invalid anchor position (%d, %d)", newRow, newCol)
	}

	// Validate unit fits within grid at new position
	if newRow+gridPosData.Height > 3 || newCol+gridPosData.Width > 3 {
		return fmt.Errorf("unit would extend outside grid at position (%d, %d) with size %dx%d",
			newRow, newCol, gridPosData.Width, gridPosData.Height)
	}

	memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)

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
	case FormationBalanced:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 0, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 0, AnchorCol: 2, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 1, Role: RoleSupport, Target: []int{1}},
				{AnchorRow: 2, AnchorCol: 0, Role: RoleDPS, Target: []int{2}},
				{AnchorRow: 2, AnchorCol: 2, Role: RoleDPS, Target: []int{2}},
			},
		}

	case FormationDefensive:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 0, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 0, AnchorCol: 1, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 0, AnchorCol: 2, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 1, Role: RoleSupport, Target: []int{1}},
				{AnchorRow: 2, AnchorCol: 1, Role: RoleDPS, Target: []int{2}},
			},
		}

	case FormationOffensive:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 1, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 0, Role: RoleDPS, Target: []int{1}},
				{AnchorRow: 1, AnchorCol: 1, Role: RoleDPS, Target: []int{1}},
				{AnchorRow: 1, AnchorCol: 2, Role: RoleDPS, Target: []int{1}},
				{AnchorRow: 2, AnchorCol: 1, Role: RoleSupport, Target: []int{2}},
			},
		}

	case FormationRanged:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 1, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 0, Role: RoleDPS, Target: []int{1, 2}},
				{AnchorRow: 1, AnchorCol: 2, Role: RoleDPS, Target: []int{1, 2}},
				{AnchorRow: 2, AnchorCol: 0, Role: RoleDPS, Target: []int{2}},
				{AnchorRow: 2, AnchorCol: 1, Role: RoleSupport, Target: []int{2}},
				{AnchorRow: 2, AnchorCol: 2, Role: RoleDPS, Target: []int{2}},
			},
		}

	default:
		return FormationPreset{Positions: []FormationPosition{}}
	}
}

// CreateSquadFromTemplate - ✅ Returns ecs.EntityID (native type)
func CreateSquadFromTemplate(
	ecsmanager *common.EntityManager,
	squadName string,
	formation FormationType,
	worldPos coords.LogicalPosition,
	unitTemplates []UnitTemplate,
) ecs.EntityID {

	// Create squad entity
	squadEntity := ecsmanager.World.NewEntity()

	// ✅ Get native entity ID
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(SquadComponent, &SquadData{
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

		// Update unit's world position to match squad position
		// (CreateEntityFromTemplate sets position to 0,0 by default)
		// Re-add PositionComponent with correct world position
		unitEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{
			X: worldPos.X,
			Y: worldPos.Y,
		})

		// Add squad membership (uses ID, not entity pointer)
		unitEntity.AddComponent(SquadMemberComponent, &SquadMemberData{
			SquadID: squadID, // ✅ Native entity ID
		})

		// Add grid position (supports multi-cell)
		unitEntity.AddComponent(GridPositionComponent, &GridPositionData{
			AnchorRow: template.GridRow,
			AnchorCol: template.GridCol,
			Width:     width,
			Height:    height,
		})

		// Add role
		unitEntity.AddComponent(UnitRoleComponent, &UnitRoleData{
			Role: template.Role,
		})

		// Add targeting data (supports both row-based and cell-based modes)
		targetMode := TargetModeRowBased
		if template.TargetMode == TargetModeCellBased {
			targetMode = TargetModeCellBased
		}

		unitEntity.AddComponent(TargetRowComponent, &TargetRowData{
			Mode:          targetMode,
			TargetRows:    template.TargetRows,
			IsMultiTarget: template.IsMultiTarget,
			MaxTargets:    template.MaxTargets,
			TargetCells:   template.TargetCells,
		})

		// Add cover component if unit provides cover
		if template.CoverValue > 0.0 {
			unitEntity.AddComponent(CoverComponent, &CoverData{
				CoverValue:     template.CoverValue,
				CoverRange:     template.CoverRange,
				RequiresActive: template.RequiresActive,
			})
		}

		// Add leader component if needed
		if template.IsLeader {
			unitEntity.AddComponent(LeaderComponent, &LeaderData{
				Leadership: 10,
				Experience: 0,
			})

			// Add ability slots
			unitEntity.AddComponent(AbilitySlotComponent, &AbilitySlotData{
				Slots: [4]AbilitySlot{},
			})

			// Add cooldown tracker
			unitEntity.AddComponent(CooldownTrackerComponent, &CooldownTrackerData{
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
