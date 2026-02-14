package commander

import (
	"game_main/common"
	"game_main/tactical/spells"
	"game_main/tactical/squads"
	"game_main/visual/rendering"
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
		AddComponent(common.PositionComponent, &coords.LogicalPosition{
			X: startPos.X,
			Y: startPos.Y,
		}).
		AddComponent(rendering.RenderableComponent, &rendering.Renderable{
			Image:   commanderImage,
			Visible: true,
		}).
		AddComponent(common.AttributeComponent, &common.Attributes{
			MovementSpeed: movementSpeed,
		}).
		AddComponent(squads.SquadRosterComponent, squads.NewSquadRoster(maxSquads)).
		AddComponent(spells.ManaComponent, &spells.ManaData{
			CurrentMana: startingMana,
			MaxMana:     maxMana,
		}).
		AddComponent(spells.SpellBookComponent, &spells.SpellBookData{
			SpellIDs: initialSpells,
		})

	// Add to position system
	if common.GlobalPositionSystem != nil {
		common.GlobalPositionSystem.AddEntity(commanderID, startPos)
	}

	return commanderID
}
