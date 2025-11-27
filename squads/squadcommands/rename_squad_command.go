package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// RenameSquadCommand renames a squad
// Simple command with straightforward undo (restore old name)
type RenameSquadCommand struct {
	entityManager *common.EntityManager
	squadID       ecs.EntityID
	newName       string

	// Captured state for undo
	oldName string
}

// NewRenameSquadCommand creates a new rename squad command
func NewRenameSquadCommand(
	manager *common.EntityManager,
	squadID ecs.EntityID,
	newName string,
) *RenameSquadCommand {
	return &RenameSquadCommand{
		entityManager: manager,
		squadID:       squadID,
		newName:       newName,
	}
}

// Validate checks if the squad can be renamed
func (cmd *RenameSquadCommand) Validate() error {
	if cmd.squadID == 0 {
		return fmt.Errorf("invalid squad ID")
	}

	if cmd.newName == "" {
		return fmt.Errorf("squad name cannot be empty")
	}

	// Check if squad exists
	squadEntity := squads.GetSquadEntity(cmd.squadID, cmd.entityManager)
	if squadEntity == nil {
		return fmt.Errorf("squad does not exist")
	}

	return nil
}

// Execute renames the squad
func (cmd *RenameSquadCommand) Execute() error {
	// Get squad entity
	squadEntity := squads.GetSquadEntity(cmd.squadID, cmd.entityManager)
	if squadEntity == nil {
		return fmt.Errorf("squad not found")
	}

	// Get squad data
	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	if squadData == nil {
		return fmt.Errorf("squad has no data component")
	}

	// Save old name for undo
	cmd.oldName = squadData.Name

	// Set new name
	squadData.Name = cmd.newName

	return nil
}

// Undo restores the old squad name
func (cmd *RenameSquadCommand) Undo() error {
	if cmd.oldName == "" {
		return fmt.Errorf("no saved name available for undo")
	}

	// Get squad entity
	squadEntity := squads.GetSquadEntity(cmd.squadID, cmd.entityManager)
	if squadEntity == nil {
		return fmt.Errorf("squad not found")
	}

	// Get squad data
	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	if squadData == nil {
		return fmt.Errorf("squad has no data component")
	}

	// Restore old name
	squadData.Name = cmd.oldName

	return nil
}

// Description returns a human-readable description
func (cmd *RenameSquadCommand) Description() string {
	if cmd.oldName != "" {
		return fmt.Sprintf("Rename squad from '%s' to '%s'", cmd.oldName, cmd.newName)
	}
	return fmt.Sprintf("Rename squad to '%s'", cmd.newName)
}
