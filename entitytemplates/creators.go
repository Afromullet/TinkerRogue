// Package entitytemplates provides template-based entity creation and data loading systems.
// It handles JSON-based entity definitions, component composition patterns,
// and factory functions for creating entities from reusable templates.
package entitytemplates

import (
	"game_main/common"
	"game_main/coords"

	"game_main/visual/rendering"
	"game_main/worldmap"
	"log"
	"path/filepath"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// createBaseEntity creates a basic entity with common components (name, renderable, position).
// It loads the image from the specified path and sets up the fundamental entity structure.
func createBaseEntity(manager common.EntityManager, name, imagePath, assetDir string, visible bool, pos *coords.LogicalPosition) *ecs.Entity {
	fpath := filepath.Join(assetDir, imagePath)
	img, _, err := ebitenutil.NewImageFromFile(fpath)
	if err != nil {
		log.Fatal(err)
	}

	entity := manager.World.NewEntity()
	entity.AddComponent(common.NameComponent, &common.Name{NameStr: name})
	entity.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
		Image: img, Visible: visible,
	})

	if pos == nil {
		pos = &coords.LogicalPosition{X: 0, Y: 0}
	}
	entity.AddComponent(common.PositionComponent, pos)

	return entity
}

// ComponentAdder is a function type that adds specific components to an entity.
// Used in the template system to compose entities with different component combinations.
type ComponentAdder func(entity *ecs.Entity)

// createFromTemplate creates an entity using a base template and applies additional components.
// It uses the ComponentAdder pattern to compose entities with varying component sets.
func createFromTemplate(manager common.EntityManager, name, imagePath, assetDir string, visible bool, pos *coords.LogicalPosition, adders ...ComponentAdder) *ecs.Entity {
	entity := createBaseEntity(manager, name, imagePath, assetDir, visible, pos)

	for _, adder := range adders {
		adder(entity)
	}

	return entity
}

func addCreatureComponents(m JSONMonster) ComponentAdder {
	return func(entity *ecs.Entity) {

		entity.AddComponent(common.UserMsgComponent, &common.UserMessage{})

		// Mark as monster for ECS queries
		entity.AddComponent(common.MonsterComponent, &common.Monster{})

		// Use the NewAttributesFromJson method to create attributes with proper derivation
		attr := m.Attributes.NewAttributesFromJson()

		// Weapon and armor components removed - will be replaced by squad system

		entity.AddComponent(common.AttributeComponent, &attr)
	}
}

// EntityType represents the category of entity being created.
// Used in the unified factory system to differentiate entity construction logic.
type EntityType int

const (
	EntityConsumable EntityType = iota
	EntityCreature
)

// EntityConfig holds all configuration data needed to create an entity.
// It provides a unified interface for entity construction across different types.
type EntityConfig struct {
	Type      EntityType
	Name      string
	ImagePath string
	AssetDir  string
	Visible   bool
	Position  *coords.LogicalPosition

	// Optional creature-specific fields
	GameMap *worldmap.GameMap // Only used for creatures to block map tiles
}

// CreateEntityFromTemplate is a unified factory function that creates entities of any type.
// It uses the EntityType enum and type assertions to handle entity-specific construction logic.
//
// Parameters:
//   - manager: ECS entity manager
//   - config: Entity configuration (type, paths, position, etc.)
//   - data: Entity-specific data (JSONMeleeWeapon, JSONRangedWeapon, JSONAttributeModifier, or JSONMonster)
//
// Returns the created entity with all appropriate components attached.
func CreateEntityFromTemplate(manager common.EntityManager, config EntityConfig, data any) *ecs.Entity {
	var adders []ComponentAdder

	switch config.Type {

	case EntityCreature:
		m, ok := data.(JSONMonster)
		if !ok {
			log.Fatalf("Expected JSONMonster for EntityCreature, got %T", data)
		}
		adders = []ComponentAdder{addCreatureComponents(m)}

		// Handle creature-specific map blocking logic
		if config.GameMap != nil && config.Position != nil {
			ind := coords.CoordManager.LogicalToIndex(*config.Position)
			config.GameMap.Tiles[ind].Blocked = true
		}

		// Register creature with PositionSystem for O(1) position lookups
		entity := createFromTemplate(manager, config.Name, config.ImagePath, config.AssetDir,
			config.Visible, config.Position, adders...)
		if common.GlobalPositionSystem != nil && config.Position != nil {
			common.GlobalPositionSystem.AddEntity(entity.GetID(), *config.Position)

		}
		return entity

	default:
		log.Fatalf("Unknown entity type: %d", config.Type)
	}

	return createFromTemplate(manager, config.Name, config.ImagePath, config.AssetDir,
		config.Visible, config.Position, adders...)
}
