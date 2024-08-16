package main

import (
	ecs "github.com/bytearena/ecs"
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
