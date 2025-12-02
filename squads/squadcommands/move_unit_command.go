package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/squads"

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
	squadEntity := squads.GetSquadEntity(c.squadID, c.manager)
	if squadEntity == nil {
		return fmt.Errorf("squad not found")
	}

	// Check unit is in squad
	if !c.manager.HasComponentByIDWithTag(c.unitID, squads.SquadMemberTag, squads.SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	memberData := common.GetComponentTypeByID[*squads.SquadMemberData](c.manager, c.unitID, squads.SquadMemberComponent)
	if memberData == nil || memberData.SquadID != c.squadID {
		return fmt.Errorf("unit is not in this squad")
	}

	// Validate new position
	if c.newRow < 0 || c.newRow > 2 || c.newCol < 0 || c.newCol > 2 {
		return fmt.Errorf("invalid grid position (%d, %d)", c.newRow, c.newCol)
	}

	// Check if new position is occupied (excluding the unit being moved)
	existingUnits := squads.GetUnitIDsAtGridPosition(c.squadID, c.newRow, c.newCol, c.manager)
	for _, existingID := range existingUnits {
		if existingID != c.unitID {
			return fmt.Errorf("grid position (%d, %d) is already occupied", c.newRow, c.newCol)
		}
	}

	return nil
}

func (c *MoveUnitCommand) Execute() error {
	// Capture old position for undo
	gridPos := common.GetComponentTypeByID[*squads.GridPositionData](c.manager, c.unitID, squads.GridPositionComponent)
	if gridPos == nil {
		return fmt.Errorf("unit has no grid position")
	}

	c.oldRow = gridPos.AnchorRow
	c.oldCol = gridPos.AnchorCol

	// Move unit to new position
	err := squads.MoveUnitInSquad(c.unitID, c.newRow, c.newCol, c.manager)
	if err != nil {
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
	// Get unit name for better description
	unitName := "Unit"
	if nameComp, ok := c.manager.GetComponent(c.unitID, common.NameComponent); ok {
		if name := nameComp.(*common.Name); name != nil {
			unitName = name.NameStr
		}
	}

	return fmt.Sprintf("Move '%s' to [%d,%d]", unitName, c.newRow, c.newCol)
}
