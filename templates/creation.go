package templates

import (
	"game_main/common"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"github.com/bytearena/ecs"
)

// CreateConsumable creates a consumable item entity at the specified position.
// This is the single source of truth for consumable creation.
// Unlike monsters, consumables don't block tiles and are added to the tile's entity list.
// Returns the created consumable entity.
func CreateConsumable(mgr common.EntityManager, gm *worldmap.GameMap, pos coords.LogicalPosition, template JSONAttributeModifier) *ecs.Entity {
	// Create base entity with name, renderable, and position components
	// Note: Initially created as invisible, will be made visible by caller if needed
	entity := createBaseEntity(mgr, template.Name, template.ImgName, AssetPath("items"), false, nil)

	// Make the renderable visible
	common.GetComponentType[*rendering.Renderable](entity, rendering.RenderableComponent).Visible = true

	// Set the position
	entityPos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
	entityPos.X = pos.X
	entityPos.Y = pos.Y

	// Add entity to the tile's entity list (consumables don't block tiles)
	gm.AddEntityToTile(entity, &pos)

	return entity
}

// CreateUnit creates a unit entity with base components only.
// Squad package will add squad-specific components (GridPosition, SquadMember, etc.).
// This is the single source of truth for unit base entity creation.
// Returns entity with: NameComponent, PositionComponent, AttributeComponent
func CreateUnit(mgr common.EntityManager, name string, attributes common.Attributes, pos *coords.LogicalPosition) *ecs.Entity {
	// Create base entity
	entity := mgr.World.NewEntity()

	// Add name component
	entity.AddComponent(common.NameComponent, &common.Name{NameStr: name})

	// Add position component (default to 0,0 if not specified)
	if pos == nil {
		pos = &coords.LogicalPosition{X: 0, Y: 0}
	}
	entity.AddComponent(common.PositionComponent, pos)

	// Add attributes component
	entity.AddComponent(common.AttributeComponent, &attributes)

	return entity
}
