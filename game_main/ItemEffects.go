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
Each Effects implements GetPropertyComponent and GetPropertyName,
Which is called by the generic GetPropertyName and GetPropertyComponent functions.const

# Copy() implementation must return a shallow copy

ApplyToCreature() takes a query result. The implementing method will define how the effect changes components.

That lets us get the components and name without having to assert it to a specfic type.
*/
type Effects interface {
	EffectComponent() *ecs.Component
	EffectName() string
	Duration() int
	ApplyToCreature(c *ecs.QueryResult)
	Copy() Effects
}

func EffectName[T Effects](prop *T) string {
	return (*prop).EffectName()
}

func EffectComponent[T Effects](prop *T) *ecs.Component {
	return (*prop).EffectComponent()
}

func AllEffects(effects *ecs.Entity) []Effects {

	eff := make([]Effects, 0)

	for _, e := range AllItemEffects {

		data, ok := effects.GetComponentData(e)

		if ok {

			d := *data.(*Effects)
			//p := d.(any) originally added tis to eff = append.
			eff = append(eff, d.Copy())

		}

	}

	return eff

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

func (s *Sticky) EffectComponent() *ecs.Component {
	return StickyComponent
}

func (s *Sticky) EffectName() string {
	return s.MainProps.Name

}

func (s Sticky) Duration() int {

	return s.MainProps.Duration

}

func (s *Sticky) Copy() Effects {
	return &Sticky{
		MainProps: s.MainProps,
		Spread:    s.Spread,
	}
}

func (s *Sticky) ApplyToCreature(c *ecs.QueryResult) {
	fmt.Println("Applying ", s, " To Creature")
	s.MainProps.Duration -= 1
	fmt.Println("Remaining duration ", s.MainProps.Duration)

}

func NewSticky(dur int, spr int) *Sticky {

	return &Sticky{
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

func (b *Burning) EffectComponent() *ecs.Component {
	return BurningComponent
}

func (b *Burning) EffectName() string {
	return b.MainProps.Name
}

func (b Burning) Duration() int {

	return b.MainProps.Duration

}

func (b *Burning) Copy() Effects {
	return &Burning{
		MainProps:   b.MainProps,
		Temperature: b.Temperature,
	}
}

func (b *Burning) ApplyToCreature(c *ecs.QueryResult) {

	b.MainProps.Duration -= 1

	h := GetComponentType[*Attributes](c.Entity, attributeComponent)

	h.CurrentHealth -= b.Temperature

	fmt.Println(h)

	fmt.Println("Remaining duration ", b.MainProps.Duration)
	fmt.Println("Remaining health ", h.CurrentHealth)
}

func NewBurning(dur int, temp int) *Burning {

	return &Burning{
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

func (f *Freezing) EffectComponent() *ecs.Component {
	return FreezingComponent
}

func (f *Freezing) EffectName() string {
	return f.MainProps.Name

}

func (f Freezing) Duration() int {

	return f.MainProps.Duration

}

func (f *Freezing) Copy() Effects {
	return &Freezing{
		MainProps: f.MainProps,
		Thickness: f.Thickness,
	}
}

func (f *Freezing) ApplyToCreature(c *ecs.QueryResult) {
	fmt.Println("Applying ", f, " To Creature")
	f.MainProps.Duration -= 1
	fmt.Println("Remaining duration ", f.MainProps.Duration)

}

func NewFreezing(dur int, t int) *Freezing {

	return &Freezing{
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
	ThrowingRange int //How many tiles it can be thrown
	Damage        int
	Shape         TileBasedShape
}

func (t *Throwable) EffectComponent() *ecs.Component {
	return ThrowableComponent
}

func (t *Throwable) EffectName() string {
	return t.MainProps.Name

}

func (t Throwable) Duration() int {

	return t.MainProps.Duration

}

func (t *Throwable) Copy() Effects {
	return &Throwable{
		MainProps:     t.MainProps,
		ThrowingRange: t.ThrowingRange,
		Damage:        t.Damage,
		Shape:         t.Shape,
	}
}

func (t *Throwable) ApplyToCreature(c *ecs.QueryResult) {
	fmt.Println("Applying ", t, " To Creature")

}

func NewThrowable(dur, throwRange, dam int, shape TileBasedShape) *Throwable {

	return &Throwable{
		MainProps: CommonItemProperties{
			Name:     THROWABLE_NAME,
			Duration: dur,
		},
		ThrowingRange: throwRange,
		Damage:        dam,
		Shape:         shape,
	}

}

func InitializeItemComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	ItemComponent = manager.NewComponent()
	StickyComponent = manager.NewComponent()
	BurningComponent = manager.NewComponent()
	FreezingComponent = manager.NewComponent()

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
