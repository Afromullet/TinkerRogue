package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// RemoveUnitCommand removes a unit from a squad and returns it to the roster
type RemoveUnitCommand struct {
	manager  *common.EntityManager
	playerID ecs.EntityID
	squadID  ecs.EntityID
	unitID   ecs.EntityID

	// Undo state
	previousGridRow int
	previousGridCol int
	unitTemplate    squads.UnitTemplate
}

func NewRemoveUnitCommand(
	manager *common.EntityManager,
	playerID ecs.EntityID,
	squadID ecs.EntityID,
	unitID ecs.EntityID,
) *RemoveUnitCommand {
	return &RemoveUnitCommand{
		manager:  manager,
		playerID: playerID,
		squadID:  squadID,
		unitID:   unitID,
	}
}

func (c *RemoveUnitCommand) Validate() error {
	// Check squad exists
	if err := validateSquadExists(c.squadID, c.manager); err != nil {
		return err
	}

	// Check unit is in squad
	if err := validateUnitInSquad(c.unitID, c.squadID, c.manager); err != nil {
		return err
	}

	// Check unit is not the leader
	isLeader := c.manager.HasComponentByIDWithTag(c.unitID, squads.SquadMemberTag, squads.LeaderComponent)
	if isLeader {
		return fmt.Errorf("cannot remove squad leader")
	}

	return nil
}

func (c *RemoveUnitCommand) Execute() error {
	// Capture unit state for undo
	gridPos, err := getGridPositionOrError(c.unitID, c.manager)
	if err != nil {
		return err
	}
	c.previousGridRow = gridPos.AnchorRow
	c.previousGridCol = gridPos.AnchorCol

	// Capture unit attributes
	attr := common.GetAttributesByIDWithTag(c.manager, c.unitID, squads.SquadMemberTag)
	if attr == nil {
		return fmt.Errorf("unit has no attributes")
	}

	// Capture unit name
	nameStr := "Unit"
	if nameComp, ok := c.manager.GetComponent(c.unitID, common.NameComponent); ok {
		if name := nameComp.(*common.Name); name != nil {
			nameStr = name.NameStr
		}
	}

	// Capture unit role
	roleData := common.GetComponentTypeByID[*squads.UnitRoleData](c.manager, c.unitID, squads.UnitRoleComponent)
	role := squads.RoleDPS
	if roleData != nil {
		role = roleData.Role
	}

	// Save unit template for undo
	c.unitTemplate = squads.UnitTemplate{
		Name:       nameStr,
		GridRow:    c.previousGridRow,
		GridCol:    c.previousGridCol,
		GridWidth:  gridPos.Width,
		GridHeight: gridPos.Height,
		Role:       role,
		Attributes: *attr,
	}

	roster := squads.GetPlayerRoster(c.playerID, c.manager)
	if roster == nil {
		return fmt.Errorf("player roster not found")
	}

	// Mark unit as available in roster (decrement in-squad count)
	// This makes the unit available to add to squads again
	if err := roster.MarkUnitAvailable(c.unitID); err != nil {
		return fmt.Errorf("failed to mark unit available: %w", err)
	}

	// Remove unit from squad (this disposes the entity)
	// The disposed entity ID remains in roster's UnitEntities list but that's OK
	// because availability is tracked by counts, not entity validity
	if err := squads.RemoveUnitFromSquad(c.unitID, c.manager); err != nil {
		return fmt.Errorf("failed to remove unit from squad: %w", err)
	}

	return nil
}

func (c *RemoveUnitCommand) Undo() error {
	// Re-add unit to squad at previous position
	err := squads.AddUnitToSquad(c.squadID, c.manager, c.unitTemplate, c.previousGridRow, c.previousGridCol)
	if err != nil {
		return fmt.Errorf("failed to re-add unit: %w", err)
	}

	// Get the re-added unit ID
	readdedUnits := squads.GetUnitIDsAtGridPosition(c.squadID, c.previousGridRow, c.previousGridCol, c.manager)
	if len(readdedUnits) == 0 {
		return fmt.Errorf("unit was not re-added successfully")
	}

	roster := squads.GetPlayerRoster(c.playerID, c.manager)
	if roster == nil {
		return fmt.Errorf("player roster not found")
	}

	// Register the newly created squad entity in roster
	err = roster.AddUnit(readdedUnits[0], c.unitTemplate.Name)
	if err != nil {
		return fmt.Errorf("failed to add unit to roster: %w", err)
	}

	// Mark as assigned to squad
	err = roster.MarkUnitInSquad(readdedUnits[0], c.squadID)
	if err != nil {
		return fmt.Errorf("failed to mark unit in squad: %w", err)
	}

	return nil
}

func (c *RemoveUnitCommand) Description() string {
	return fmt.Sprintf("Remove unit from squad")
}
