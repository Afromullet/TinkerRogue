package commander

import (
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/tactical/powers/progression"
	"game_main/tactical/squads/roster"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// CreateCommander creates a new commander entity with all required components.
// Returns the new commander's entity ID.
func CreateCommander(
	manager *common.EntityManager,
	name string,
	startPos coords.LogicalPosition,
	movementSpeed int,
	maxSquads int,
	commanderImage *ebiten.Image,
) ecs.EntityID {
	entity := manager.World.NewEntity()
	commanderID := entity.GetID()

	entity.
		AddComponent(CommanderComponent, &CommanderData{
			CommanderID: commanderID,
			Name:        name,
			IsActive:    true,
		}).
		AddComponent(CommanderActionStateComponent, &CommanderActionStateData{
			CommanderID:       commanderID,
			HasMoved:          false,
			HasActed:          false,
			MovementRemaining: movementSpeed,
		}).
		AddComponent(common.RenderableComponent, &common.Renderable{
			Image:   commanderImage,
			Visible: true,
		}).
		AddComponent(common.AttributeComponent, &common.Attributes{
			MovementSpeed: movementSpeed,
		}).
		AddComponent(roster.SquadRosterComponent, roster.NewSquadRoster(maxSquads)).
		AddComponent(progression.ProgressionComponent, progression.NewProgressionData())

	// Atomically add position component and register with position system
	manager.RegisterEntityPosition(entity, startPos)

	return commanderID
}
