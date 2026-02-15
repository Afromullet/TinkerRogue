package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// MoveUnitCommand moves a unit to a new position in the squad grid
type MoveUnitCommand struct {
	manager *common.EntityManager
	squadID ecs.EntityID
	unitID  ecs.EntityID
	newRow  int
	newCol  int

	// Undo state
	oldRow int
	oldCol int
}

func NewMoveUnitCommand(
	manager *common.EntityManager,
	squadID ecs.EntityID,
	unitID ecs.EntityID,
	newRow int,
	newCol int,
) *MoveUnitCommand {
	return &MoveUnitCommand{
		manager: manager,
		squadID: squadID,
		unitID:  unitID,
		newRow:  newRow,
		newCol:  newCol,
	}
}

func (c *MoveUnitCommand) Validate() error {
	// Check squad exists
	if err := validateSquadExists(c.squadID, c.manager); err != nil {
		return err
	}

	// Check unit is in squad
	if err := validateUnitInSquad(c.unitID, c.squadID, c.manager); err != nil {
		return err
	}

	// Validate new position
	if err := validateGridPosition(c.newRow, c.newCol); err != nil {
		return err
	}

	// Check if new position is occupied (excluding the unit being moved)
	return validateGridPositionNotOccupied(c.squadID, c.newRow, c.newCol, c.manager, c.unitID)
}

func (c *MoveUnitCommand) Execute() error {
	// Capture old position for undo
	gridPos, err := getGridPositionOrError(c.unitID, c.manager)
	if err != nil {
		return err
	}

	c.oldRow = gridPos.AnchorRow
	c.oldCol = gridPos.AnchorCol

	// Move unit to new position
	if err := squads.MoveUnitInSquad(c.unitID, c.newRow, c.newCol, c.manager); err != nil {
		return fmt.Errorf("failed to move unit: %w", err)
	}

	return nil
}

func (c *MoveUnitCommand) Undo() error {
	// Move unit back to old position
	err := squads.MoveUnitInSquad(c.unitID, c.oldRow, c.oldCol, c.manager)
	if err != nil {
		return fmt.Errorf("failed to restore unit position: %w", err)
	}

	return nil
}

func (c *MoveUnitCommand) Description() string {
	unitName := common.GetEntityName(c.manager, c.unitID, "Unit")
	return fmt.Sprintf("Move '%s' to [%d,%d]", unitName, c.newRow, c.newCol)
}
