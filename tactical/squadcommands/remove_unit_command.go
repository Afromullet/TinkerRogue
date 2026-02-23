package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// RemoveUnitCommand removes a unit from a squad and returns it to the roster.
// The entity is NOT disposed — it stays alive in the ECS world for re-assignment.
type RemoveUnitCommand struct {
	manager  *common.EntityManager
	playerID ecs.EntityID
	squadID  ecs.EntityID
	unitID   ecs.EntityID

	// Undo state — only need grid position since entity survives
	previousGridRow int
	previousGridCol int
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
	isLeader := c.manager.HasComponent(c.unitID, squads.LeaderComponent)
	if isLeader {
		return fmt.Errorf("cannot remove squad leader")
	}

	return nil
}

func (c *RemoveUnitCommand) Execute() error {
	// Capture grid position for undo
	gridPos, err := getGridPositionOrError(c.unitID, c.manager)
	if err != nil {
		return err
	}
	c.previousGridRow = gridPos.AnchorRow
	c.previousGridCol = gridPos.AnchorCol

	roster := squads.GetPlayerRoster(c.playerID, c.manager)
	if roster == nil {
		return fmt.Errorf("player roster not found")
	}

	// Mark unit as available in roster tracking
	if err := roster.MarkUnitAvailable(c.unitID); err != nil {
		return fmt.Errorf("failed to mark unit available: %w", err)
	}

	// Unassign from squad (entity stays alive, returns to roster pool)
	if err := squads.UnassignUnitFromSquad(c.unitID, c.manager); err != nil {
		return fmt.Errorf("failed to unassign unit from squad: %w", err)
	}

	return nil
}

func (c *RemoveUnitCommand) Undo() error {
	// Place the same entity back into the squad at its previous position
	err := squads.PlaceUnitInSquad(c.squadID, c.unitID, c.manager, c.previousGridRow, c.previousGridCol)
	if err != nil {
		return fmt.Errorf("failed to re-place unit in squad: %w", err)
	}

	roster := squads.GetPlayerRoster(c.playerID, c.manager)
	if roster == nil {
		return fmt.Errorf("player roster not found")
	}

	// Mark as in squad in roster tracking
	err = roster.MarkUnitInSquad(c.unitID, c.squadID)
	if err != nil {
		return fmt.Errorf("failed to mark unit in squad: %w", err)
	}

	return nil
}

func (c *RemoveUnitCommand) Description() string {
	return fmt.Sprintf("Remove unit from squad")
}
