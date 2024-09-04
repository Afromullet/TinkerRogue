package main

import (
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
	Properties *ecs.Entity
	Count      int
}

func (item *Item) IncrementCount() {
	item.Count += 1
}

func (item *Item) DecrementCount() {
	item.Count -= 1
}

// Returns the names of this items properties.
func (item *Item) GetEffectNames() []string {

	names := make([]string, 0)

	for _, c := range AllItemEffects {
		data, ok := item.Properties.GetComponentData(c)
		if ok {
			print(data)

			d := data.(*Effects)
			names = append(names, EffectName(d))
		}
	}
	return names

}

/*
I didn't understand Go Interfaces well enough when implementing item properties
So accessing Item Properties takes some extra work

Takes the component identifying string as input and returns the
struct that represents the property

Here's an example of how it's used:

item := GetComponentStruct[*Item](itemEntity, ItemComponent)
t := item.GetItemEffect(THROWABLE_NAME).(throwable)
fmt.Println(t.shape)
*/

func (item *Item) ItemEffect(effectName string) any {

	for _, c := range AllItemEffects {
		data, ok := item.Properties.GetComponentData(c)
		if ok {
			print(data)

			d := *data.(*Effects)
			if EffectName(&d) == effectName {
				p := d.(any)
				return p
			}
		}
	}
	return nil
}

// Not the best way to check if an item has all propeties, but it will work for now
func (item *Item) HasAllEffects(effectsToCheck ...Effects) bool {

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

func (item *Item) HasEffect(effectToCheck Effects) bool {

	names := item.GetEffectNames()
	comp := effectToCheck.EffectName()

	for _, n := range names {

		if n == comp {
			return true

		}

	}

	return false

}

// Create an item with any number of Effects. ItemEffect is a wrapper around an ecs.Component to make
// Manipulating it easier
func CreateItem(manager *ecs.Manager, name string, pos Position, imagePath string, effects ...Effects) *ecs.Entity {

	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	item := &Item{Count: 1, Properties: manager.NewEntity()}

	for _, prop := range effects {
		item.Properties.AddComponent(prop.EffectComponent(), &prop)

	}

	itemEntity := manager.NewEntity().
		AddComponent(renderable, &Renderable{
			Image:   img,
			Visible: true,
		}).
		AddComponent(position, &Position{
			X: pos.X,
			Y: pos.Y,
		}).
		AddComponent(nameComponent, &Name{
			NameStr: name,
		}).
		AddComponent(ItemComponent, item)

	//TODO where shoudl I add the tags?

	return itemEntity

}

// A weapon is an Item with a weapon component
func CreateWeapon(manager *ecs.Manager, name string, pos Position, imagePath string, MinDamage int, MaxDamage int, properties ...Effects) *ecs.Entity {

	weapon := CreateItem(manager, name, pos, imagePath, properties...)

	weapon.AddComponent(WeaponComponent, &Weapon{
		MinDamage: MinDamage,
		MaxDamage: MaxDamage,
	})

	return weapon

}
