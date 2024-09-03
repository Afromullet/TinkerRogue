package main

import (
	"fmt"

	"github.com/bytearena/ecs"
)

const BURNING_NAME = "Burning"
const FREEZING_NAME = "Freezing"
const STICKY_NAME = "Sticky"
const THROWABLE_NAME = "Throwable"

var EffectNames = []string{BURNING_NAME, FREEZING_NAME, STICKY_NAME, THROWABLE_NAME}

var ItemComponent *ecs.Component
var StickyComponent *ecs.Component
var BurningComponent *ecs.Component
var FreezingComponent *ecs.Component
var ThrowableComponent *ecs.Component

/*
The AllItemEffects makes it easier to query an Item for all of its Properties.

entity.GetComponentData takes a component as input, so we use AllItemEffects
to keep track of all proeprties an item might have
*/
var AllItemEffects []*ecs.Component

/*
Quasi-Polymorphism that tries to make Item Properties interchangable.
The ECS library makes it tedious to access component data. I don't
want to modify the ECS library, so this is a workaround.

Each Effects implements GetPropertyComponent and GetPropertyName,
Which is called by the generic GetPropertyName and GetPropertyComponent functions.const

That lets us get the components and name without having to assert it to a specfic type.
*/
type Effects interface {
	GetEffectComponent() *ecs.Component
	GetEffectName() string
	ApplyToCreature(c *Creature)
}

func GetEffectName[T Effects](prop *T) string {
	return (*prop).GetEffectName()
}

func GetEffectComponent[T Effects](prop *T) *ecs.Component {
	return (*prop).GetEffectComponent()
}

func GetEffect(effects *ecs.Entity) any {

	for _, e := range AllItemEffects {

		data, ok := effects.GetComponentData(e)

		if ok {

			d := *data.(*Effects)
			p := d.(any)
			return p

		}

	}

	return nil

}

// Item Properties
type CommonItemProperties struct {
	Duration int
	Name     string
}

type Sticky struct {
	MainProps CommonItemProperties
	Spread    int //Sticky effects can spread

}

func (s Sticky) GetEffectComponent() *ecs.Component {
	return StickyComponent
}

func (s Sticky) GetEffectName() string {
	return s.MainProps.Name

}

func (s Sticky) ApplyToCreature(c *Creature) {
	fmt.Println("Applying ", s, " To Creature")

}

func NewSticky(dur int, spr int) Sticky {

	return Sticky{
		MainProps: CommonItemProperties{
			Name:     STICKY_NAME,
			Duration: dur,
		},
		Spread: spr,
	}

}

type Burning struct {
	MainProps CommonItemProperties

	Temperature int
}

func (b Burning) GetEffectComponent() *ecs.Component {
	return BurningComponent
}

func (b Burning) GetEffectName() string {
	return b.MainProps.Name
}

func (b Burning) ApplyToCreature(c *Creature) {
	fmt.Println("Applying ", b, " To Creature")

}

func NewBurning(dur int, temp int) Burning {

	return Burning{
		MainProps: CommonItemProperties{
			Name:     BURNING_NAME,
			Duration: dur,
		},
		Temperature: temp,
	}

}

type Freezing struct {
	MainProps CommonItemProperties
	Thickness int //How thick the ice is.

}

func (f Freezing) GetEffectComponent() *ecs.Component {
	return FreezingComponent
}

func (f Freezing) GetEffectName() string {
	return f.MainProps.Name

}

func (f Freezing) ApplyToCreature(c *Creature) {
	fmt.Println("Applying ", f, " To Creature")

}

func NewFreezing(dur int, t int) Freezing {

	return Freezing{
		MainProps: CommonItemProperties{
			Name:     FREEZING_NAME,
			Duration: dur,
		},
		Thickness: t,
	}

}

// Throwable doesn't work like any other effects,
// So treating it as an "Effect" does not make much sense because we
// Have to implement methods that do not apply to it such as
// ApplyToCreature. MainProps.duration also doesn't mean much for this
// But it works for now...
type Throwable struct {
	MainProps     CommonItemProperties
	throwingRange int //How many tiles it can be thrown
	damage        int
	shape         TileBasedShape
}

func (t Throwable) GetEffectComponent() *ecs.Component {
	return ThrowableComponent
}

func (t Throwable) GetEffectName() string {
	return t.MainProps.Name

}

func (t Throwable) ApplyToCreature(c *Creature) {
	fmt.Println("Applying ", t, " To Creature")

}

func NewThrowable(dur, throwRange, dam int, shape TileBasedShape) Throwable {

	return Throwable{
		MainProps: CommonItemProperties{
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

	AllItemEffects = append(AllItemEffects, StickyComponent)
	AllItemEffects = append(AllItemEffects, BurningComponent)
	AllItemEffects = append(AllItemEffects, FreezingComponent)
	AllItemEffects = append(AllItemEffects, ThrowableComponent)

	items := ecs.BuildTag(ItemComponent, position) //todo add all the tags
	tags["items"] = items

	//Can I use this instead of the AllItemproperties slice? Todo see later if you ran replace it

	sticking := ecs.BuildTag(StickyComponent)
	tags["sticking"] = sticking

	burning := ecs.BuildTag(BurningComponent)
	tags["burning"] = burning

	freezing := ecs.BuildTag(FreezingComponent)
	tags["freezing"] = freezing

}
