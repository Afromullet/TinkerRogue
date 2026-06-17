package squadcommands

import (
	"fmt"
	"game_main/core/common"
	rstr "game_main/tactical/squads/roster"
	"game_main/tactical/squads/squadcore"

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
	isLeader := c.manager.HasComponent(c.unitID, squadcore.LeaderComponent)
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

	roster, err := getPlayerRosterOrError(c.playerID, c.manager)
	if err != nil {
		return err
	}

	return rstr.UnassignUnitFromSquad(roster, c.unitID, c.squadID, c.manager)
}

func (c *RemoveUnitCommand) Undo() error {
	roster, err := getPlayerRosterOrError(c.playerID, c.manager)
	if err != nil {
		return err
	}
	return rstr.AssignUnitToSquad(roster, c.unitID, c.squadID, c.previousGridRow, c.previousGridCol, c.manager)
}

func (c *RemoveUnitCommand) Description() string {
	return fmt.Sprintf("Remove unit from squad")
}
