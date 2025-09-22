package gear

import (
	"fmt"
	"game_main/common"
	"game_main/rendering"
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
type Item struct {
	Properties *ecs.Entity  // Status effects only
	Actions    []ItemAction // Actions like throwables, consumables, etc.
	Count      int
}

func (item *Item) IncrementCount() {
	item.Count += 1
}

func (item *Item) DecrementCount() {
	item.Count -= 1
}

// Returns the names of every property the item has
func (item *Item) GetEffectNames() []string {

	names := make([]string, 0)

	if item.Properties == nil {
		return names
	}

	for _, c := range AllItemEffects {
		data, ok := item.Properties.GetComponentData(c)
		if ok {

			d := data.(*StatusEffects)
			names = append(names, StatusEffectName(d))
		}
	}
	return names

}

//This will eventually fully replace GetEffectNames.

func (item *Item) GetEffectString() string {

	if item.Properties == nil {
		return ""
	}

	result := ""

	for _, c := range AllItemEffects {
		data, ok := item.Properties.GetComponentData(c)
		if ok {

			d := data.(*StatusEffects)

			result += fmt.Sprintln(StatusEffectName(d))
		}
	}

	return result
}

/*
I didn't understand Go Interfaces well enough when implementing item properties
So accessing Item Properties takes some extra work

Takes the component identifying string as input and returns the
struct that represents the property

Example for status effects:
item := GetComponentStruct[*Item](itemEntity, ItemComponent)
burning := item.ItemEffect(BURNING_NAME).(*Burning)

For actions, use the type-safe methods:
throwable := item.GetThrowableAction()

*/

func (item *Item) ItemEffect(effectName string) any {

	for _, c := range AllItemEffects {
		data, ok := item.Properties.GetComponentData(c)
		if ok {

			d := *data.(*StatusEffects)
			if StatusEffectName(&d) == effectName {
				p := d.(any)
				return p
			}
		}
	}
	return nil
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

// GetFirstActionOfType returns the first action of the specified type, or nil if none exists
// Usage: throwable := item.GetFirstActionOfType[*ThrowableAction]()
func GetFirstActionOfType[T ItemAction](item *Item) T {
	var zero T
	for _, action := range item.Actions {
		if typedAction, ok := action.(T); ok {
			return typedAction
		}
	}
	return zero
}

// Not the best way to check if an item has all propeties, but it will work for now
func (item *Item) HasAllEffects(effectsToCheck ...StatusEffects) bool {

	if len(effectsToCheck) == 0 {
		return true
	}

	for _, eff := range effectsToCheck {

		if !item.HasEffect(eff) {
			return false
		}
	}

	return true

}

// Check for an effect by name.
func (item *Item) HasEffect(effectToCheck StatusEffects) bool {

	names := item.GetEffectNames()
	comp := effectToCheck.StatusEffectName()

	for _, n := range names {

		if n == comp {
			return true

		}

	}

	return false

}

// The testing package has the same function. Todo remove the one from testing package
func CreateItem(manager *ecs.Manager, name string, pos common.Position, imagePath string, effects ...StatusEffects) *ecs.Entity {

	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	item := &Item{Count: 1, Properties: manager.NewEntity(), Actions: make([]ItemAction, 0)}

	for _, prop := range effects {
		item.Properties.AddComponent(prop.StatusEffectComponent(), &prop)

	}

	itemEntity := manager.NewEntity().
		AddComponent(rendering.RenderableComponent, &rendering.Renderable{
			Image:   img,
			Visible: true,
		}).
		AddComponent(common.PositionComponent, &common.Position{
			X: pos.X,
			Y: pos.Y,
		}).
		AddComponent(common.NameComponent, &common.Name{
			NameStr: name,
		}).
		AddComponent(ItemComponent, item)

	//TODO where shoudl I add the tags?

	return itemEntity

}

// CreateItemWithActions creates an item with both status effects and actions
func CreateItemWithActions(manager *ecs.Manager, name string, pos common.Position, imagePath string, actions []ItemAction, effects ...StatusEffects) *ecs.Entity {

	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	item := &Item{Count: 1, Properties: manager.NewEntity(), Actions: make([]ItemAction, len(actions))}

	// Add status effects
	for _, prop := range effects {
		item.Properties.AddComponent(prop.StatusEffectComponent(), &prop)
	}

	// Add actions
	copy(item.Actions, actions)

	itemEntity := manager.NewEntity().
		AddComponent(rendering.RenderableComponent, &rendering.Renderable{
			Image:   img,
			Visible: true,
		}).
		AddComponent(common.PositionComponent, &common.Position{
			X: pos.X,
			Y: pos.Y,
		}).
		AddComponent(common.NameComponent, &common.Name{
			NameStr: name,
		}).
		AddComponent(ItemComponent, item)

	return itemEntity
}
