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

type sticky struct {
	CommonItemProperties
	Spread int //Sticky effects can spread

}

func (s sticky) GetPropertyComponent() *ecs.Component {
	return StickyComponent
}

func (s sticky) GetPropertyName() string {
	return s.CommonItemProperties.Name

}

func NewSticky(dur int, spr int) sticky {

	return sticky{
		CommonItemProperties: CommonItemProperties{
			Name:     STICKY_NAME,
			Duration: dur,
		},
		Spread: spr,
	}

}

type burning struct {
	CommonItemProperties

	Temperature int
}

func (b burning) GetPropertyComponent() *ecs.Component {
	return BurningComponent
}

func (b burning) GetPropertyName() string {
	return b.CommonItemProperties.Name

}

func NewBurning(dur int, temp int) burning {

	return burning{
		CommonItemProperties: CommonItemProperties{
			Name:     BURNING_NAME,
			Duration: dur,
		},
		Temperature: temp,
	}

}

type freezing struct {
	CommonItemProperties
	Thickness int //How thick the ice is.

}

func (f freezing) GetPropertyComponent() *ecs.Component {
	return FreezingComponent
}

func (f freezing) GetPropertyName() string {
	return f.CommonItemProperties.Name

}

func NewFreezing(dur int, t int) freezing {

	return freezing{
		CommonItemProperties: CommonItemProperties{
			Name:     FREEZING_NAME,
			Duration: dur,
		},
		Thickness: t,
	}

}

type throwable struct {
	CommonItemProperties
	throwingRange int //How many tiles it can be thrown
	damage        int
	area          int //The area is the number of squares
}

func (t throwable) GetPropertyComponent() *ecs.Component {
	return ThrowableComponent
}

func (t throwable) GetPropertyName() string {
	return t.CommonItemProperties.Name

}

func NewThrowable(dur int, throwRange int, dam int) throwable {

	return throwable{
		CommonItemProperties: CommonItemProperties{
			Name:     THROWABLE_NAME,
			Duration: dur,
		},
		throwingRange: throwRange,
		damage:        dam,
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
