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
// Attributes.MovementSpeed is the canonical source of truth for movement;
// MovementRemaining is seeded from it here and reset to it each turn by StartNewTurn.
func CreateCommander(
	manager *common.EntityManager,
	name string,
	startPos coords.LogicalPosition,
	movementSpeed int,
	maxSquads int,
	commanderImage *ebiten.Image,
) ecs.EntityID {
	entity := manager.World.NewEntity()

	attrs := &common.Attributes{MovementSpeed: movementSpeed}

	entity.
		AddComponent(CommanderComponent, &CommanderData{
			Name:     name,
			IsActive: true,
		}).
		AddComponent(CommanderActionStateComponent, &CommanderActionStateData{
			HasMoved:          false,
			HasActed:          false,
			MovementRemaining: attrs.GetMovementSpeed(),
		}).
		AddComponent(common.RenderableComponent, &common.Renderable{
			Image:   commanderImage,
			Visible: true,
		}).
		AddComponent(common.AttributeComponent, attrs).
		AddComponent(roster.SquadRosterComponent, roster.NewSquadRoster(maxSquads)).
		AddComponent(progression.ProgressionComponent, &progression.ProgressionData{})

	// Atomically add position component and register with position system
	manager.RegisterEntityPosition(entity, startPos)

	return entity.GetID()
}
