package gear

import (
	"game_main/common"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"log"

	ecs "github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var (
	ItemComponent *ecs.Component
	ItemsTag      ecs.Tag // Tag for querying item entities
)

// init registers the gear subsystem with the ECS component registry.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		InitializeItemComponents(em.World, em.WorldTags)
		InitializeItemTags(em.WorldTags)
	})
}

func InitializeItemComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {
	ItemComponent = manager.NewComponent()
	InventoryComponent = manager.NewComponent()
}

// InitializeItemTags creates tags for querying item-related entities.
// Call this after InitializeItemComponents.
func InitializeItemTags(tags map[string]ecs.Tag) {
	ItemsTag = ecs.BuildTag(ItemComponent, common.PositionComponent)
	tags["items"] = ItemsTag
}

// Item is a pure data component (ECS best practice)
// Use system functions in gearutil.go for all logic operations
type Item struct {
	Count int
}

// CreateItem creates an item entity (ECS best practice compliant)
func CreateItem(manager *ecs.Manager, name string, pos coords.LogicalPosition, imagePath string) *ecs.Entity {

	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	item := &Item{
		Count: 1,
	}

	itemEntity := manager.NewEntity().
		AddComponent(rendering.RenderableComponent, &rendering.Renderable{
			Image:   img,
			Visible: true,
		}).
		AddComponent(common.PositionComponent, &coords.LogicalPosition{
			X: pos.X,
			Y: pos.Y,
		}).
		AddComponent(common.NameComponent, &common.Name{
			NameStr: name,
		}).
		AddComponent(ItemComponent, item)

	return itemEntity
}
