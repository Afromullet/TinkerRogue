package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// AddUnitCommand adds a unit from the player's roster to a squad at a specific grid position
type AddUnitCommand struct {
	manager      *common.EntityManager
	playerID     ecs.EntityID
	squadID      ecs.EntityID
	templateName string
	gridRow      int
	gridCol      int

	// Undo state
	addedUnitID ecs.EntityID
}

func NewAddUnitCommand(
	manager *common.EntityManager,
	playerID ecs.EntityID,
	squadID ecs.EntityID,
	templateName string,
	gridRow int,
	gridCol int,
) *AddUnitCommand {
	return &AddUnitCommand{
		manager:      manager,
		playerID:     playerID,
		squadID:      squadID,
		templateName: templateName,
		gridRow:      gridRow,
		gridCol:      gridCol,
	}
}

func (c *AddUnitCommand) Validate() error {
	// Check squad exists
	if err := validateSquadExists(c.squadID, c.manager); err != nil {
		return err
	}

	// Check roster exists
	roster := squads.GetPlayerRoster(c.playerID, c.manager)
	if roster == nil {
		return fmt.Errorf("player roster not found")
	}

	// Check template exists and is available
	availableCount := roster.GetAvailableCount(c.templateName)
	if availableCount == 0 {
		return fmt.Errorf("no available units of type '%s'", c.templateName)
	}

	// Validate grid position
	if err := validateGridPosition(c.gridRow, c.gridCol); err != nil {
		return err
	}

	// Check if position is occupied
	return validateGridPositionNotOccupied(c.squadID, c.gridRow, c.gridCol, c.manager, 0)
}

func (c *AddUnitCommand) Execute() error {
	roster := squads.GetPlayerRoster(c.playerID, c.manager)
	if roster == nil {
		return fmt.Errorf("player roster not found")
	}

	// Get available unit from roster
	unitEntityID := roster.GetUnitEntityForTemplate(c.templateName)
	if unitEntityID == 0 {
		return fmt.Errorf("no available unit entity for template '%s'", c.templateName)
	}

	// Create unit template from the existing unit's data
	// Get unit attributes
	attr := common.GetAttributesByIDWithTag(c.manager, unitEntityID, squads.SquadMemberTag)
	if attr == nil {
		return fmt.Errorf("unit entity has no attributes")
	}

	// Get unit name
	nameStr := c.templateName
	if nameComp, ok := c.manager.GetComponent(unitEntityID, common.NameComponent); ok {
		if name := nameComp.(*common.Name); name != nil {
			nameStr = name.NameStr
		}
	}

	// Create unit template
	unitTemplate := squads.UnitTemplate{
		Name:       nameStr,
		GridRow:    c.gridRow,
		GridCol:    c.gridCol,
		GridWidth:  1,
		GridHeight: 1,
		Role:       squads.RoleDPS, // Default role
		Attributes: *attr,
	}

	// Add unit to squad
	unitID, err := squads.AddUnitToSquad(c.squadID, c.manager, unitTemplate, c.gridRow, c.gridCol)
	if err != nil {
		return fmt.Errorf("failed to add unit to squad: %w", err)
	}

	c.addedUnitID = unitID

	// Register the newly created squad entity in roster and mark as in squad
	err = roster.AddUnit(c.addedUnitID, c.templateName)
	if err != nil {
		return fmt.Errorf("failed to add unit to roster: %w", err)
	}

	err = roster.MarkUnitInSquad(c.addedUnitID, c.squadID)
	if err != nil {
		return fmt.Errorf("failed to mark unit in roster: %w", err)
	}

	return nil
}

func (c *AddUnitCommand) Undo() error {
	if c.addedUnitID == 0 {
		return fmt.Errorf("no unit to remove (command was not executed)")
	}

	roster := squads.GetPlayerRoster(c.playerID, c.manager)
	if roster == nil {
		return fmt.Errorf("player roster not found")
	}

	// Mark as available first (decrement in-squad count)
	err := roster.MarkUnitAvailable(c.addedUnitID)
	if err != nil {
		return fmt.Errorf("failed to mark unit available: %w", err)
	}

	// Remove unit from squad (this disposes the entity)
	err = squads.RemoveUnitFromSquad(c.addedUnitID, c.manager)
	if err != nil {
		return fmt.Errorf("failed to remove unit from squad: %w", err)
	}

	// Remove from roster completely (reduces TotalOwned, removes disposed entity ID)
	if !roster.RemoveUnit(c.addedUnitID) {
		return fmt.Errorf("failed to remove unit from roster")
	}

	c.addedUnitID = 0
	return nil
}

func (c *AddUnitCommand) Description() string {
	return fmt.Sprintf("Add unit '%s' to squad at [%d,%d]", c.templateName, c.gridRow, c.gridCol)
}
