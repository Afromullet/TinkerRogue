package main

import (
	"fmt"
	"game_main/graphics"

	"github.com/bytearena/ecs"
)

/*
To create a new item property:

1) Create a const string containing the component name. All components use the same name.
2) Add the name to the EffectNames slice.
3) Create the component type
4) Create the struct associated with the component
5) In InitializeItemComponents, initialize the component
6) In InitializeItemComponents, add the component to the AllItemEffects slice.
*/
const (
	BURNING_NAME   = "Burning"
	FREEZING_NAME  = "Freezing"
	STICKY_NAME    = "Sticky"
	THROWABLE_NAME = "Throwable"
)

var (
	EffectNames = []string{BURNING_NAME, FREEZING_NAME, STICKY_NAME, THROWABLE_NAME}

	ItemComponent      *ecs.Component
	StickyComponent    *ecs.Component
	BurningComponent   *ecs.Component
	FreezingComponent  *ecs.Component
	ThrowableComponent *ecs.Component
)

/*
The AllItemEffects makes it easier to query an Item for all of its Properties.

entity.GetComponentData takes a component as input, so we use AllItemEffects
to keep track of all proeprties an item might have
*/
var AllItemEffects []*ecs.Component

/*
Each StatusEffects implements GetPropertyComponent and GetPropertyName,
Which is called by the generic GetPropertyName and GetPropertyComponent functions.const

# Copy() implementation must return a shallow copy

ApplyToCreature() takes a query result. The implementing method will define how the effect changes components.

That lets us get the components and name without having to assert it to a specfic type.
*/
type StatusEffects interface {
	StatusEffectComponent() *ecs.Component
	StatusEffectName() string
	Duration() int
	ApplyToCreature(c *ecs.QueryResult)
	Copy() StatusEffects
}

func StatusEffectName[T StatusEffects](prop *T) string {
	return (*prop).StatusEffectName()
}

func StatusEffectComponent[T StatusEffects](prop *T) *ecs.Component {
	return (*prop).StatusEffectComponent()
}

func AllStatusEffects(effects *ecs.Entity) []StatusEffects {

	eff := make([]StatusEffects, 0)

	for _, e := range AllItemEffects {

		data, ok := effects.GetComponentData(e)

		if ok {

			d := *data.(*StatusEffects)
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

func (s *Sticky) StatusEffectComponent() *ecs.Component {
	return StickyComponent
}

func (s *Sticky) StatusEffectName() string {
	return s.MainProps.Name

}

func (s Sticky) Duration() int {

	return s.MainProps.Duration

}

func (s *Sticky) Copy() StatusEffects {
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

func (b *Burning) StatusEffectComponent() *ecs.Component {
	return BurningComponent
}

func (b *Burning) StatusEffectName() string {
	return b.MainProps.Name
}

func (b Burning) Duration() int {

	return b.MainProps.Duration

}

func (b *Burning) Copy() StatusEffects {
	return &Burning{
		MainProps:   b.MainProps,
		Temperature: b.Temperature,
	}
}

func (b *Burning) ApplyToCreature(c *ecs.QueryResult) {

	b.MainProps.Duration -= 1

	h := GetComponentType[*Attributes](c.Entity, AttributeComponent)

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

func (f *Freezing) StatusEffectComponent() *ecs.Component {
	return FreezingComponent
}

func (f *Freezing) StatusEffectName() string {
	return f.MainProps.Name

}

func (f Freezing) Duration() int {

	return f.MainProps.Duration

}

func (f *Freezing) Copy() StatusEffects {
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
// vx is the VisualEffect which will be drawn in the AOE Shape
type Throwable struct {
	MainProps     CommonItemProperties
	ThrowingRange int //How many tiles it can be thrown
	Damage        int
	Shape         graphics.TileBasedShape
	vx            graphics.VisualEffect
}

func (t *Throwable) StatusEffectComponent() *ecs.Component {
	return ThrowableComponent
}

func (t *Throwable) StatusEffectName() string {
	return t.MainProps.Name

}

func (t Throwable) Duration() int {

	return t.MainProps.Duration

}

func (t *Throwable) Copy() StatusEffects {
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

// Adds the Throwing Weapons VisualEffectArea to the VisualEffectHandler. It will be drawn.
func (t *Throwable) ReadyThrowAreaVX() {

	//AddVXArea(t.vxArea)

}

func NewThrowable(dur, throwRange, dam int, shape graphics.TileBasedShape) *Throwable {

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

	AllItemEffects = append(AllItemEffects, StickyComponent, BurningComponent, FreezingComponent, ThrowableComponent)

	items := ecs.BuildTag(ItemComponent, PositionComponent) //todo add all the tags
	tags["items"] = items

}
