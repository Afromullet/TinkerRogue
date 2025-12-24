package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// FormationAssignment represents where a unit should be positioned in formation
type FormationAssignment struct {
	UnitID  ecs.EntityID // Unit to position
	GridRow int          // Target row (0-2)
	GridCol int          // Target col (0-2)
}

// ChangeFormationCommand changes squad formation by repositioning units
// Validates that new positions don't conflict and are valid
type ChangeFormationCommand struct {
	entityManager *common.EntityManager
	squadID       ecs.EntityID
	newFormation  []FormationAssignment

	// Captured state for undo
	oldFormation []FormationAssignment
}

// NewChangeFormationCommand creates a new change formation command
func NewChangeFormationCommand(
	manager *common.EntityManager,
	squadID ecs.EntityID,
	newFormation []FormationAssignment,
) *ChangeFormationCommand {
	return &ChangeFormationCommand{
		entityManager: manager,
		squadID:       squadID,
		newFormation:  newFormation,
	}
}

// Validate checks if the formation change is valid
func (cmd *ChangeFormationCommand) Validate() error {
	// Check if squad exists
	if err := validateSquadExists(cmd.squadID, cmd.entityManager); err != nil {
		return err
	}

	if len(cmd.newFormation) == 0 {
		return fmt.Errorf("formation cannot be empty")
	}

	// Validate each position
	occupiedCells := make(map[[2]int]bool)

	for _, assignment := range cmd.newFormation {
		// Validate position bounds
		if err := validateGridPosition(assignment.GridRow, assignment.GridCol); err != nil {
			return err
		}

		// Check for position conflicts within new formation
		cell := [2]int{assignment.GridRow, assignment.GridCol}
		if occupiedCells[cell] {
			return fmt.Errorf("position conflict at (%d, %d)", assignment.GridRow, assignment.GridCol)
		}
		occupiedCells[cell] = true

		// Verify unit exists and belongs to this squad
		unitEntity := common.FindEntityByID(cmd.entityManager, assignment.UnitID)
		if unitEntity == nil {
			return fmt.Errorf("unit %d not found", assignment.UnitID)
		}

		memberData := common.GetComponentType[*squads.SquadMemberData](unitEntity, squads.SquadMemberComponent)
		if memberData == nil || memberData.SquadID != cmd.squadID {
			return fmt.Errorf("unit %d does not belong to squad %d", assignment.UnitID, cmd.squadID)
		}

		// Check if unit is multi-cell (would need additional validation)
		if unitEntity.HasComponent(squads.GridPositionComponent) {
			gridPos := common.GetComponentType[*squads.GridPositionData](unitEntity, squads.GridPositionComponent)
			if gridPos != nil {
				// For multi-cell units, verify all cells they occupy are free
				for r := assignment.GridRow; r < assignment.GridRow+gridPos.Height && r < 3; r++ {
					for c := assignment.GridCol; c < assignment.GridCol+gridPos.Width && c < 3; c++ {
						if r != assignment.GridRow || c != assignment.GridCol {
							extraCell := [2]int{r, c}
							if occupiedCells[extraCell] {
								return fmt.Errorf("multi-cell unit position conflict at (%d, %d)", r, c)
							}
							occupiedCells[extraCell] = true
						}
					}
				}

				// Verify multi-cell unit fits in grid at new position
				if assignment.GridRow+gridPos.Height > 3 || assignment.GridCol+gridPos.Width > 3 {
					return fmt.Errorf("multi-cell unit would extend outside grid at (%d, %d)",
						assignment.GridRow, assignment.GridCol)
				}
			}
		}
	}

	return nil
}

// Execute applies the new formation
func (cmd *ChangeFormationCommand) Execute() error {
	// Capture current positions for undo
	if err := cmd.captureCurrentFormation(); err != nil {
		return fmt.Errorf("failed to capture current formation: %w", err)
	}

	// Apply new formation
	for _, assignment := range cmd.newFormation {
		// Update grid position
		gridPos := common.GetComponentTypeByID[*squads.GridPositionData](
			cmd.entityManager, assignment.UnitID, squads.GridPositionComponent)
		if gridPos == nil {
			return fmt.Errorf("unit %d not found or has no grid position component", assignment.UnitID)
		}

		gridPos.AnchorRow = assignment.GridRow
		gridPos.AnchorCol = assignment.GridCol
	}

	return nil
}

// Undo restores the old formation
func (cmd *ChangeFormationCommand) Undo() error {
	if len(cmd.oldFormation) == 0 {
		return fmt.Errorf("no saved formation available for undo")
	}

	// Restore old positions
	for _, assignment := range cmd.oldFormation {
		// Restore grid position
		gridPos := common.GetComponentTypeByID[*squads.GridPositionData](
			cmd.entityManager, assignment.UnitID, squads.GridPositionComponent)
		if gridPos == nil {
			// Unit might have been removed - skip it
			continue
		}

		gridPos.AnchorRow = assignment.GridRow
		gridPos.AnchorCol = assignment.GridCol
	}

	return nil
}

// Description returns a human-readable description
func (cmd *ChangeFormationCommand) Description() string {
	squadName := squads.GetSquadName(cmd.squadID, cmd.entityManager)
	return fmt.Sprintf("Change formation for squad '%s' (%d units repositioned)",
		squadName, len(cmd.newFormation))
}

// captureCurrentFormation saves current unit positions for undo
func (cmd *ChangeFormationCommand) captureCurrentFormation() error {
	cmd.oldFormation = make([]FormationAssignment, 0)

	// Get all units in squad
	unitIDs := squads.GetUnitIDsInSquad(cmd.squadID, cmd.entityManager)

	for _, unitID := range unitIDs {
		unitEntity := common.FindEntityByID(cmd.entityManager, unitID)
		if unitEntity == nil {
			continue
		}

		// Get current grid position
		if unitEntity.HasComponent(squads.GridPositionComponent) {
			gridPos := common.GetComponentType[*squads.GridPositionData](unitEntity, squads.GridPositionComponent)
			if gridPos != nil {
				assignment := FormationAssignment{
					UnitID:  unitID,
					GridRow: gridPos.AnchorRow,
					GridCol: gridPos.AnchorCol,
				}
				cmd.oldFormation = append(cmd.oldFormation, assignment)
			}
		}
	}

	return nil
}
