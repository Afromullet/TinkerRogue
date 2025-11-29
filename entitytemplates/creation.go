package entitytemplates

import (
	"game_main/common"
	"game_main/coords"
	"game_main/rendering"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
)

// CreateMonster creates a monster entity at the specified position.
// This is the single source of truth for monster creation.
// It handles all entity construction, component attachment, tile blocking, and position system registration.
//
// Parameters:
//   - mgr: ECS entity manager
//   - gm: Game map (for tile blocking)
//   - pos: Logical position where monster should spawn
//   - template: Monster template data (attributes, name, image)
//
// Returns the created monster entity with all components attached.
func CreateMonster(mgr common.EntityManager, gm *worldmap.GameMap, pos coords.LogicalPosition, template JSONMonster) *ecs.Entity {
	// Create base entity with name, renderable, and position components
	entity := createBaseEntity(mgr, template.Name, template.ImageName, "../assets/creatures/", true, &pos)

	// Add monster-specific components (attributes, monster tag, user message)
	addCreatureComponents(template)(entity)

	// Register with position system for O(1) spatial queries
	if common.GlobalPositionSystem != nil {
		common.GlobalPositionSystem.AddEntity(entity.GetID(), pos)
	}

	// Block the tile to prevent other entities from spawning here
	// SINGLE SOURCE OF TRUTH for tile blocking during monster creation
	idx := coords.CoordManager.LogicalToIndex(pos)
	gm.Tiles[idx].Blocked = true

	return entity
}

// CreateConsumable creates a consumable item entity at the specified position.
// This is the single source of truth for consumable creation.
// Unlike monsters, consumables don't block tiles and are added to the tile's entity list.
//
// Parameters:
//   - mgr: ECS entity manager
//   - gm: Game map (for adding to tile entity list)
//   - pos: Logical position where item should spawn
//   - template: Consumable template data
//
// Returns the created consumable entity.
func CreateConsumable(mgr common.EntityManager, gm *worldmap.GameMap, pos coords.LogicalPosition, template JSONAttributeModifier) *ecs.Entity {
	// Create base entity with name, renderable, and position components
	// Note: Initially created as invisible, will be made visible by caller if needed
	entity := createBaseEntity(mgr, template.Name, template.ImgName, "../assets/items/", false, nil)

	// Make the renderable visible
	common.GetComponentType[*rendering.Renderable](entity, rendering.RenderableComponent).Visible = true

	// Set the position
	entityPos := common.GetPosition(entity)
	entityPos.X = pos.X
	entityPos.Y = pos.Y

	// Add entity to the tile's entity list (consumables don't block tiles)
	gm.AddEntityToTile(entity, &pos)

	return entity
}
