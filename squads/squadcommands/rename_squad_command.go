package squadcommands

import (
	"fmt"
	"game_main/common"

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
	if cmd.newName == "" {
		return fmt.Errorf("squad name cannot be empty")
	}

	// Check if squad exists
	return validateSquadExists(cmd.squadID, cmd.entityManager)
}

// Execute renames the squad
func (cmd *RenameSquadCommand) Execute() error {
	// Get squad entity
	squadEntity, err := getSquadOrError(cmd.squadID, cmd.entityManager)
	if err != nil {
		return err
	}

	// Get squad data
	squadData, err := getSquadDataOrError(squadEntity)
	if err != nil {
		return err
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
	squadEntity, err := getSquadOrError(cmd.squadID, cmd.entityManager)
	if err != nil {
		return err
	}

	// Get squad data
	squadData, err := getSquadDataOrError(squadEntity)
	if err != nil {
		return err
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
