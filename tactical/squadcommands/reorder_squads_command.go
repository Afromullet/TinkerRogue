package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// ReorderSquadsCommand moves a squad from one position to another in the squad roster
type ReorderSquadsCommand struct {
	manager   *common.EntityManager
	playerID  ecs.EntityID
	fromIndex int
	toIndex   int

	// Undo state
	oldOrder []ecs.EntityID
}

// NewReorderSquadsCommand creates a new command to reorder squads
func NewReorderSquadsCommand(
	manager *common.EntityManager,
	playerID ecs.EntityID,
	fromIndex int,
	toIndex int,
) *ReorderSquadsCommand {
	return &ReorderSquadsCommand{
		manager:   manager,
		playerID:  playerID,
		fromIndex: fromIndex,
		toIndex:   toIndex,
	}
}

func (c *ReorderSquadsCommand) Validate() error {
	roster := squads.GetPlayerSquadRoster(c.playerID, c.manager)
	if roster == nil {
		return fmt.Errorf("player squad roster not found")
	}

	squadCount := len(roster.OwnedSquads)

	// Check indices are in valid range
	if c.fromIndex < 0 || c.fromIndex >= squadCount {
		return fmt.Errorf("fromIndex %d out of range (0-%d)", c.fromIndex, squadCount-1)
	}

	if c.toIndex < 0 || c.toIndex >= squadCount {
		return fmt.Errorf("toIndex %d out of range (0-%d)", c.toIndex, squadCount-1)
	}

	// Check not moving to same position
	if c.fromIndex == c.toIndex {
		return fmt.Errorf("cannot move squad to same position")
	}

	return nil
}

func (c *ReorderSquadsCommand) Execute() error {
	roster := squads.GetPlayerSquadRoster(c.playerID, c.manager)
	if roster == nil {
		return fmt.Errorf("player squad roster not found")
	}

	// Capture old order for undo
	c.oldOrder = make([]ecs.EntityID, len(roster.OwnedSquads))
	copy(c.oldOrder, roster.OwnedSquads)

	// Get the squad being moved
	squadID := roster.OwnedSquads[c.fromIndex]

	// Remove from original position
	roster.OwnedSquads = append(
		roster.OwnedSquads[:c.fromIndex],
		roster.OwnedSquads[c.fromIndex+1:]...,
	)

	// Insert at new position
	roster.OwnedSquads = append(
		roster.OwnedSquads[:c.toIndex],
		append([]ecs.EntityID{squadID}, roster.OwnedSquads[c.toIndex:]...)...,
	)

	return nil
}

func (c *ReorderSquadsCommand) Undo() error {
	if c.oldOrder == nil {
		return fmt.Errorf("no order to restore (command was not executed)")
	}

	roster := squads.GetPlayerSquadRoster(c.playerID, c.manager)
	if roster == nil {
		return fmt.Errorf("player squad roster not found")
	}

	// Restore old order
	roster.OwnedSquads = make([]ecs.EntityID, len(c.oldOrder))
	copy(roster.OwnedSquads, c.oldOrder)

	return nil
}

func (c *ReorderSquadsCommand) Description() string {
	return fmt.Sprintf("Reorder squad from position %d to %d", c.fromIndex, c.toIndex)
}
