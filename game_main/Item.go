package main

import (
	"log"

	ecs "github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

/*
An Items property is an Entity. That helps us track the components

# So an Items Properties could look like this for Example

Item
-properties *ecs.Entity
--Freezing Component
--Burning Component
...
--Other COmponents

All properties are structs that implement the ItemProperty interface, which implements
GetPropertyComonent to get the associated component.

Each Item Property also has a CommonItemProperties, where the "name" is common between
all components of a specific type.const

I.E, burning will always have BURNING_NAME, freezing has FREEZING_NAME, and so on

That's so we have an easy time displaying the property to the player. It may change later.

Since all components of a type have the same name, the struct is "private", with the construcotr
Used to create a component of that type.

Two examples below of how we'd create an item

	CreateItem(manager, "Item"+strconv.Itoa(1), Position{X: 40, Y: 25}, itemImageLoc, NewBurning(1, 1))
	CreateItem(manager, "Item"+strconv.Itoa(2), Position{X: 40, Y: 25}, itemImageLoc, NewBurning(1, 1), NewFreezing(1, 2))
*/
type Item struct {
	properties *ecs.Entity
	count      int
}

func (item *Item) IncrementCount() {
	item.count += 1
}

func (item *Item) DecrementCount() {
	item.count -= 1
}

// Returns the name of this items properties.
func (item *Item) GetPropertyNames() []string {

	names := make([]string, 0)

	for _, c := range AllItemProperties {
		data, ok := item.properties.GetComponentData(c)
		if ok {
			print(data)

			d := data.(*ItemProperty)
			names = append(names, GetPropertyName(d))
		}
	}
	return names

}

// Not the best way to check if an item has all propeties, but it will work for now
func (item *Item) HasAllProperties(propsToCheck ...ItemProperty) bool {

	if len(propsToCheck) == 0 {
		return true
	}

	for _, prop := range propsToCheck {

		if !item.HasProperty(prop) {
			return false
		}

	}

	return true

}
func (item *Item) HasProperty(propToCheck ItemProperty) bool {

	names := item.GetPropertyNames()
	comp := propToCheck.GetPropertyName()

	for _, n := range names {

		if n == comp {
			return true

		}

	}

	return false

}

type Weapon struct {
	damage int
}

// Create an item with any number of Properties. ItemProperty is a wrapper around an ecs.Component to make
// Manipulating it easier
func CreateItem(manager *ecs.Manager, name string, pos Position, imagePath string, properties ...ItemProperty) *ecs.Entity {

	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	item := &Item{count: 1, properties: manager.NewEntity()}

	for _, prop := range properties {
		item.properties.AddComponent(prop.GetPropertyComponent(), &prop)

	}

	itemEntity := manager.NewEntity().
		AddComponent(renderable, &Renderable{
			Image:   img,
			visible: true,
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
func CreateWeapon(manager *ecs.Manager, name string, pos Position, imagePath string, dam int, properties ...ItemProperty) *ecs.Entity {

	weapon := CreateItem(manager, name, pos, imagePath, properties...)

	weapon.AddComponent(WeaponComponent, &Weapon{
		damage: dam,
	})

	return weapon

}
