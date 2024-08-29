package main

import (
	"github.com/bytearena/ecs"
)

const BURNING_NAME = "Burning"
const FREEZING_NAME = "Freezing"
const STICKY_NAME = "Sticky"
const THROWABLE_NAME = "Throwable"

var ItemComponent *ecs.Component
var StickyComponent *ecs.Component
var BurningComponent *ecs.Component
var FreezingComponent *ecs.Component
var ThrowableComponent *ecs.Component

/*
The AllItemProperties makes it easier to query an Item for all of its Properties.

entity.GetComponentData takes a component as input, so we use AllItemProperties
to keep track of all proeprties an item might have
*/
var AllItemProperties []*ecs.Component

/*
Quasi-Polymorphism that tries to make Item Properties interchangable.
The ECS library makes it tedious to access component data. I don't
want to modify the ECS library, so this is a workaround.

Each ItemProperty implements GetPropertyComponent and GetPropertyName,
Which is called by the generic GetPropertyName and GetPropertyComponent functions.const

That lets us get the components and name without having to assert it to a specfic type.
*/
type ItemProperty interface {
	GetPropertyComponent() *ecs.Component
	GetPropertyName() string
}

func GetPropertyName[T ItemProperty](prop *T) string {
	return (*prop).GetPropertyName()
}

func GetPropertyComponent[T ItemProperty](prop *T) *ecs.Component {
	return (*prop).GetPropertyComponent()
}

// Item Properties
type CommonItemProperties struct {
	Duration int
	Name     string
}

type Sticky struct {
	CommonItemProperties
	Spread int //Sticky effects can spread

}

func (s Sticky) GetPropertyComponent() *ecs.Component {
	return StickyComponent
}

func (s Sticky) GetPropertyName() string {
	return s.CommonItemProperties.Name

}

func NewSticky(dur int, spr int) Sticky {

	return Sticky{
		CommonItemProperties: CommonItemProperties{
			Name:     STICKY_NAME,
			Duration: dur,
		},
		Spread: spr,
	}

}

type Burning struct {
	CommonItemProperties

	Temperature int
}

func (b Burning) GetPropertyComponent() *ecs.Component {
	return BurningComponent
}

func (b Burning) GetPropertyName() string {
	return b.CommonItemProperties.Name

}

func NewBurning(dur int, temp int) Burning {

	return Burning{
		CommonItemProperties: CommonItemProperties{
			Name:     BURNING_NAME,
			Duration: dur,
		},
		Temperature: temp,
	}

}

type Freezing struct {
	CommonItemProperties
	Thickness int //How thick the ice is.

}

func (f Freezing) GetPropertyComponent() *ecs.Component {
	return FreezingComponent
}

func (f Freezing) GetPropertyName() string {
	return f.CommonItemProperties.Name

}

func NewFreezing(dur int, t int) Freezing {

	return Freezing{
		CommonItemProperties: CommonItemProperties{
			Name:     FREEZING_NAME,
			Duration: dur,
		},
		Thickness: t,
	}

}

type Throwable struct {
	CommonItemProperties
	throwingRange int //How many tiles it can be thrown
	damage        int
	shape         TileBasedShape
}

func (t Throwable) GetPropertyComponent() *ecs.Component {
	return ThrowableComponent
}

func (t Throwable) GetPropertyName() string {
	return t.CommonItemProperties.Name

}

func NewThrowable(dur, throwRange, dam int, shape TileBasedShape) Throwable {

	return Throwable{
		CommonItemProperties: CommonItemProperties{
			Name:     THROWABLE_NAME,
			Duration: dur,
		},
		throwingRange: throwRange,
		damage:        dam,
		shape:         shape,
	}

}

func InitializeItemComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	ItemComponent = manager.NewComponent()
	StickyComponent = manager.NewComponent()
	BurningComponent = manager.NewComponent()
	FreezingComponent = manager.NewComponent()
	WeaponComponent = manager.NewComponent()
	ThrowableComponent = manager.NewComponent()

	AllItemProperties = append(AllItemProperties, StickyComponent)
	AllItemProperties = append(AllItemProperties, BurningComponent)
	AllItemProperties = append(AllItemProperties, FreezingComponent)
	AllItemProperties = append(AllItemProperties, ThrowableComponent)

	items := ecs.BuildTag(ItemComponent, position) //todo add all the tags
	tags["items"] = items

	sticking := ecs.BuildTag(StickyComponent)
	tags["sticking"] = sticking

	burning := ecs.BuildTag(BurningComponent)
	tags["burning"] = burning

	freezing := ecs.BuildTag(FreezingComponent)
	tags["freezing"] = freezing

}
