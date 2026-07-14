package templates

import (
	"fmt"
	"path/filepath"

	"game_main/core/common"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// EntityConfig holds the parameters CreateCreatureEntity needs to build a
// combat-ready creature entity.
//
// NOTE: This file is the only non-data-layer code remaining in templates/.
// The original plan was to relocate it to setup/gamesetup so templates could
// become a pure leaf, but that introduces an import cycle:
// gamesetup→squadcore (already), so squadcore (the caller) cannot
// in turn import gamesetup. Keeping it here is the pragmatic choice; the
// worldmapcore dependency has been dropped, which was the main leak.
type EntityConfig struct {
	Name      string
	ImagePath string
	AssetDir  string
	Visible   bool
	Position  *coords.LogicalPosition
}

// CreateCreatureEntity creates a creature entity from a JSONMonster template with
// the components required for combat (Name, Renderable, Position, Attribute).
// Returns an error (never panics) if the creature image asset cannot be loaded.
func CreateCreatureEntity(manager *common.EntityManager, config EntityConfig, m JSONMonster) (*ecs.Entity, error) {
	fpath := filepath.Join(config.AssetDir, config.ImagePath)
	img, _, err := ebitenutil.NewImageFromFile(fpath)
	if err != nil {
		return nil, fmt.Errorf("failed to load creature image %q: %w", fpath, err)
	}

	entity := manager.World.NewEntity()
	entity.AddComponent(common.NameComponent, &common.Name{NameStr: config.Name})
	entity.AddComponent(common.RenderableComponent, &common.Renderable{
		Image: img, Visible: config.Visible,
	})

	// Add position: use atomic registration if explicit position provided,
	// otherwise just add a default component (no position system registration).
	if config.Position != nil {
		manager.RegisterEntityPosition(entity, *config.Position)
	} else {
		entity.AddComponent(common.PositionComponent, &coords.LogicalPosition{X: 0, Y: 0})
	}

	attr := m.Attributes.NewAttributesFromJson()
	entity.AddComponent(common.AttributeComponent, &attr)

	return entity, nil
}

// CreateUnit creates a unit entity with base components only. Squad packages
// add squad-specific components (GridPosition, SquadMember, etc.).
// Returns entity with: NameComponent, PositionComponent, AttributeComponent.
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
