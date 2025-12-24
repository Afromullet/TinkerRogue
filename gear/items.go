package gear

import (
	"game_main/common"
	"game_main/world/coords"
	"game_main/visual/rendering"
	"log"

	ecs "github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

/*
Propeties is an Entity. That helps us track the components

Everything that's a property implements the Effects interface. There's
nothing to enforce this at compile time.

# So an Items Properties could look like this for Example

Item
-properties *ecs.Entity
--Freezing Component
--Burning Component
...
--Other COmponents

Each Item Property also has a CommonItemProperties, where the "name" is common between
all components of a specific type.const

I.E, burning will always have BURNING_NAME, freezing has FREEZING_NAME, and so on

That's so we have an easy time displaying the Effect to the player. It may change later.

Two examples below of how we'd create an item

	CreateItem(manager, "Item"+strconv.Itoa(1), Position{X: 40, Y: 25}, itemImageLoc, NewBurning(1, 1))
	CreateItem(manager, "Item"+strconv.Itoa(2), Position{X: 40, Y: 25}, itemImageLoc, NewBurning(1, 1), NewFreezing(1, 2))
*/
// Item is a pure data component (ECS best practice)
// Use system functions in gearutil.go for all logic operations
type Item struct {
	Properties ecs.EntityID // Status effects entity ID (ECS best practice: use EntityID, not pointer)
	Actions    []ItemAction // Actions like throwables, consumables, etc.
	Count      int
}

// GetAction retrieves an action by name
func (item *Item) GetAction(actionName string) ItemAction {
	for _, action := range item.Actions {
		if action.ActionName() == actionName {
			return action
		}
	}
	return nil
}

// HasAction checks if the item has a specific action
func (item *Item) HasAction(actionName string) bool {
	return item.GetAction(actionName) != nil
}

// GetActions returns all actions for this item
func (item *Item) GetActions() []ItemAction {
	actionsCopy := make([]ItemAction, len(item.Actions))
	for i, action := range item.Actions {
		actionsCopy[i] = action.Copy()
	}
	return actionsCopy
}

// GetThrowableAction returns the first throwable action found, or nil if none exists
func (item *Item) GetThrowableAction() *ThrowableAction {
	for _, action := range item.Actions {
		if throwable, ok := action.(*ThrowableAction); ok {
			return throwable
		}
	}
	return nil
}

// HasThrowableAction checks if the item has any throwable action
func (item *Item) HasThrowableAction() bool {
	return item.GetThrowableAction() != nil
}

// CreateItem creates an item entity with status effects (ECS best practice compliant)
func CreateItem(manager *ecs.Manager, name string, pos coords.LogicalPosition, imagePath string, effects ...StatusEffects) *ecs.Entity {

	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	// Create properties entity to hold status effects
	propsEntity := manager.NewEntity()
	for _, prop := range effects {
		propsEntity.AddComponent(prop.StatusEffectComponent(), &prop)
	}

	// Create item component with EntityID reference (ECS best practice)
	item := &Item{
		Count:      1,
		Properties: propsEntity.GetID(), // Use EntityID instead of pointer
		Actions:    make([]ItemAction, 0),
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

// CreateItemWithActions creates an item with both status effects and actions (ECS best practice compliant)
func CreateItemWithActions(manager *ecs.Manager, name string, pos coords.LogicalPosition, imagePath string, actions []ItemAction, effects ...StatusEffects) *ecs.Entity {

	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	// Create properties entity to hold status effects
	propsEntity := manager.NewEntity()
	for _, prop := range effects {
		propsEntity.AddComponent(prop.StatusEffectComponent(), &prop)
	}

	// Create item component with EntityID reference (ECS best practice)
	item := &Item{
		Count:      1,
		Properties: propsEntity.GetID(), // Use EntityID instead of pointer
		Actions:    make([]ItemAction, len(actions)),
	}

	// Add actions
	copy(item.Actions, actions)

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
