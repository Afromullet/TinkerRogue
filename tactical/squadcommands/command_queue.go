package squadcommands

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// ProcessCommandQueues executes one command per squad with queued commands
// Call this each game turn to progress queues
// IMPORTANT: Does NOT add to CommandExecutor history (micro-steps only)
func ProcessCommandQueues(manager *common.EntityManager) {
	for _, result := range manager.World.Query(CommandQueueTag) {
		entity := result.Entity

		queueData := common.GetComponentType[*CommandQueueData](entity, CommandQueueComponent)
		if queueData == nil || queueData.Paused || len(queueData.Commands) == 0 {
			continue
		}

		// Execute first command
		currentCommand := queueData.Commands[0]

		// Re-validate (state may have changed since queuing)
		if err := currentCommand.Validate(); err != nil {
			// Invalid - remove and skip
			removeFirstCommand(manager, entity.ID)
			continue
		}

		// Execute (bypasses CommandExecutor - no history tracking)
		if err := currentCommand.Execute(); err != nil {
			// Failed - remove and skip
			removeFirstCommand(manager, entity.ID)
			continue
		}

		// Success - remove executed command
		removeFirstCommand(manager, entity.ID)
	}
}

// QueueCommand adds a command to a squad's queue
// Does NOT go through CommandExecutor (no undo history)
func QueueCommand(manager *common.EntityManager, squadID ecs.EntityID, cmd SquadCommand) error {
	// Validate before queueing
	if err := cmd.Validate(); err != nil {
		return fmt.Errorf("command validation failed: %w", err)
	}

	squad := manager.FindEntityByID(squadID)
	if squad == nil {
		return fmt.Errorf("squad not found: %d", squadID)
	}

	// Get or create queue
	queueData := common.GetComponentType[*CommandQueueData](squad, CommandQueueComponent)
	if queueData == nil {
		queueData = &CommandQueueData{Commands: make([]SquadCommand, 0)}
		// Adding component automatically applies CommandQueueTag
		squad.AddComponent(CommandQueueComponent, queueData)
	}

	queueData.Commands = append(queueData.Commands, cmd)
	return nil
}

// ClearCommandQueue removes all queued commands for a squad
func ClearCommandQueue(manager *common.EntityManager, squadID ecs.EntityID) {
	squad := manager.FindEntityByID(squadID)
	if squad == nil {
		return
	}

	// Removing component automatically removes CommandQueueTag
	squad.RemoveComponent(CommandQueueComponent)
}

// GetQueuedCommands returns copy of queued commands (for UI display)
func GetQueuedCommands(manager *common.EntityManager, squadID ecs.EntityID) []SquadCommand {
	squad := manager.FindEntityByID(squadID)
	if squad == nil {
		return nil
	}

	queueData := common.GetComponentType[*CommandQueueData](squad, CommandQueueComponent)
	if queueData == nil {
		return nil
	}

	// Return copy
	commandsCopy := make([]SquadCommand, len(queueData.Commands))
	copy(commandsCopy, queueData.Commands)
	return commandsCopy
}

// HasQueuedCommands checks if squad has pending commands
func HasQueuedCommands(manager *common.EntityManager, squadID ecs.EntityID) bool {
	squad := manager.FindEntityByID(squadID)
	if squad == nil {
		return false
	}

	// Check if entity has the component (tag is automatic)
	return squad.HasComponent(CommandQueueComponent)
}

func removeFirstCommand(manager *common.EntityManager, squadID ecs.EntityID) {
	squad := manager.FindEntityByID(squadID)
	if squad == nil {
		return
	}

	queueData := common.GetComponentType[*CommandQueueData](squad, CommandQueueComponent)
	if queueData != nil && len(queueData.Commands) > 0 {
		queueData.Commands = queueData.Commands[1:]

		// If queue is empty, remove the component (tag auto-removed)
		if len(queueData.Commands) == 0 {
			squad.RemoveComponent(CommandQueueComponent)
		}
	}
}
