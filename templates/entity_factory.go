// Package entitytemplates provides template-based entity creation and data loading systems.
// It handles JSON-based entity definitions, component composition patterns,
// and factory functions for creating entities from reusable templates.
package templates

import (
	"game_main/common"
	"game_main/world/coords"

	"game_main/visual/rendering"
	"game_main/world/worldmap"
	"log"
	"path/filepath"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// EntityType represents the category of entity being created.
type EntityType int

const (
	EntityCreature EntityType = iota
)

// EntityConfig holds all configuration data needed to create an entity.
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

// CreateEntityFromTemplate creates a creature entity from a JSONMonster template.
//
// Parameters:
//   - manager: ECS entity manager
//   - config: Entity configuration (type, paths, position, etc.)
//   - data: Entity-specific data (JSONMonster)
//
// Returns the created entity with all appropriate components attached.
func CreateEntityFromTemplate(manager *common.EntityManager, config EntityConfig, data any) *ecs.Entity {
	switch config.Type {
	case EntityCreature:
		m, ok := data.(JSONMonster)
		if !ok {
			log.Fatalf("Expected JSONMonster for EntityCreature, got %T", data)
		}
		return createCreatureEntity(manager, config, m)

	default:
		log.Fatalf("Unknown entity type: %d", config.Type)
	}

	// Unreachable: log.Fatalf exits, but Go requires a return.
	return nil
}

// createCreatureEntity creates a creature entity with all components.
func createCreatureEntity(manager *common.EntityManager, config EntityConfig, m JSONMonster) *ecs.Entity {
	// Load image
	fpath := filepath.Join(config.AssetDir, config.ImagePath)
	img, _, err := ebitenutil.NewImageFromFile(fpath)
	if err != nil {
		log.Fatal(err)
	}

	// Create entity with base components
	entity := manager.World.NewEntity()
	entity.AddComponent(common.NameComponent, &common.Name{NameStr: config.Name})
	entity.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
		Image: img, Visible: config.Visible,
	})

	// Add position: use atomic registration if explicit position provided,
	// otherwise just add a default component (no position system registration)
	if config.Position != nil {
		manager.RegisterEntityPosition(entity, *config.Position)
	} else {
		entity.AddComponent(common.PositionComponent, &coords.LogicalPosition{X: 0, Y: 0})
	}

	// Add creature-specific attributes
	attr := m.Attributes.NewAttributesFromJson()
	entity.AddComponent(common.AttributeComponent, &attr)

	// Block map tile if applicable
	if config.GameMap != nil && config.Position != nil {
		ind := coords.CoordManager.LogicalToIndex(*config.Position)
		config.GameMap.Tiles[ind].Blocked = true
	}

	return entity
}

// CreateUnit creates a unit entity with base components only.
// Squad package will add squad-specific components (GridPosition, SquadMember, etc.).
// Returns entity with: NameComponent, PositionComponent, AttributeComponent
func CreateUnit(mgr common.EntityManager, name string, attributes common.Attributes, pos *coords.LogicalPosition) *ecs.Entity {
	entity := mgr.World.NewEntity()

	entity.AddComponent(common.NameComponent, &common.Name{NameStr: name})

	if pos == nil {
		pos = &coords.LogicalPosition{X: 0, Y: 0}
	}
	entity.AddComponent(common.PositionComponent, pos)

	entity.AddComponent(common.AttributeComponent, &attributes)

	return entity
}
