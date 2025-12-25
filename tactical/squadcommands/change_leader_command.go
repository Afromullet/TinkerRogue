package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// ChangeLeaderCommand changes the squad leader to a different unit
type ChangeLeaderCommand struct {
	manager     *common.EntityManager
	squadID     ecs.EntityID
	newLeaderID ecs.EntityID

	// Undo state
	oldLeaderID ecs.EntityID
}

func NewChangeLeaderCommand(
	manager *common.EntityManager,
	squadID ecs.EntityID,
	newLeaderID ecs.EntityID,
) *ChangeLeaderCommand {
	return &ChangeLeaderCommand{
		manager:     manager,
		squadID:     squadID,
		newLeaderID: newLeaderID,
	}
}

func (c *ChangeLeaderCommand) Validate() error {
	// Check squad exists
	if err := validateSquadExists(c.squadID, c.manager); err != nil {
		return err
	}

	// Check new leader is in squad
	if err := validateUnitInSquad(c.newLeaderID, c.squadID, c.manager); err != nil {
		return err
	}

	// Check new leader is not already the leader
	isLeader := c.manager.HasComponent(c.newLeaderID, squads.LeaderComponent)
	if isLeader {
		return fmt.Errorf("unit is already the leader")
	}

	return nil
}

func (c *ChangeLeaderCommand) Execute() error {
	// Find current leader
	c.oldLeaderID = squads.GetLeaderID(c.squadID, c.manager)

	// Remove leader component from old leader (if exists)
	if c.oldLeaderID != 0 {
		oldLeaderEntity := c.manager.FindEntityByID(c.oldLeaderID)
		if oldLeaderEntity != nil {
			removeLeaderComponents(oldLeaderEntity)
		}
	}

	// Add leader component to new leader
	newLeaderEntity := c.manager.FindEntityByID(c.newLeaderID)
	if newLeaderEntity == nil {
		return fmt.Errorf("new leader entity not found")
	}

	addLeaderComponents(newLeaderEntity)

	// Update squad capacity based on new leader
	squads.UpdateSquadCapacity(c.squadID, c.manager)

	return nil
}

func (c *ChangeLeaderCommand) Undo() error {
	// Remove leader component from new leader
	newLeaderEntity := c.manager.FindEntityByID(c.newLeaderID)
	if newLeaderEntity != nil {
		removeLeaderComponents(newLeaderEntity)
	}

	// Restore old leader (if there was one)
	if c.oldLeaderID != 0 {
		oldLeaderEntity := c.manager.FindEntityByID(c.oldLeaderID)
		if oldLeaderEntity != nil {
			addLeaderComponents(oldLeaderEntity)
		}
	}

	// Update squad capacity
	squads.UpdateSquadCapacity(c.squadID, c.manager)

	return nil
}

func (c *ChangeLeaderCommand) Description() string {
	newLeaderName := getUnitName(c.newLeaderID, c.manager)
	return fmt.Sprintf("Change leader to '%s'", newLeaderName)
}
