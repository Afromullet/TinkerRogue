package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

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

	// Get an available (not in any squad) unit entity from roster
	unitEntityID := roster.GetUnitEntityForTemplate(c.templateName, c.manager)
	if unitEntityID == 0 {
		return fmt.Errorf("no available unit entity for template '%s'", c.templateName)
	}

	// Place the existing entity into the squad (no cloning)
	err := squads.PlaceUnitInSquad(c.squadID, unitEntityID, c.manager, c.gridRow, c.gridCol)
	if err != nil {
		return fmt.Errorf("failed to place unit in squad: %w", err)
	}

	c.addedUnitID = unitEntityID

	// Mark unit as in squad in roster tracking
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

	// Mark as available in roster tracking
	err := roster.MarkUnitAvailable(c.addedUnitID)
	if err != nil {
		return fmt.Errorf("failed to mark unit available: %w", err)
	}

	// Unassign from squad (entity stays alive, returns to roster pool)
	err = squads.UnassignUnitFromSquad(c.addedUnitID, c.manager)
	if err != nil {
		return fmt.Errorf("failed to unassign unit from squad: %w", err)
	}

	c.addedUnitID = 0
	return nil
}

func (c *AddUnitCommand) Description() string {
	return fmt.Sprintf("Add unit '%s' to squad at [%d,%d]", c.templateName, c.gridRow, c.gridCol)
}
