package commander

import (
	"game_main/common"
	"game_main/tactical/squads/roster"
	"game_main/tactical/powers/spells"
	"game_main/world/coords"

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
	startingMana int,
	maxMana int,
	initialSpells []string,
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
		AddComponent(spells.ManaComponent, &spells.ManaData{
			CurrentMana: startingMana,
			MaxMana:     maxMana,
		}).
		AddComponent(spells.SpellBookComponent, &spells.SpellBookData{
			SpellIDs: initialSpells,
		})

	// Atomically add position component and register with position system
	manager.RegisterEntityPosition(entity, startPos)

	return commanderID
}
