package combat

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// CreateActionStateForSquad creates the ActionStateData component for a squad.
// Exported for use by encounter package during legacy setup.
func CreateActionStateForSquad(manager *common.EntityManager, squadID ecs.EntityID) error {
	actionEntity := manager.World.NewEntity()

	movementSpeed := squads.GetSquadMovementSpeed(squadID, manager)
	if movementSpeed == 0 {
		movementSpeed = 3 // Default if no valid units found
	}

	actionEntity.AddComponent(ActionStateComponent, &ActionStateData{
		SquadID:           squadID,
		HasMoved:          false,
		HasActed:          false,
		MovementRemaining: movementSpeed,
	})

	return nil
}
