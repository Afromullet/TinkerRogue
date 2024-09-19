package gear

import (
	"fmt"
	"game_main/common"
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

// StatusEffectComponent returns the underlying component
// StatusEffectNames returns the name from the Common Properties
// Duration returns the duration from the common properties
// ApplyToCreature decides what the effect does to the creature
// Copy creates a shallow copy
// Even though all of them have the name and duration in the common properties,
// we have it as a method of the interface so that we can access it without type assertions
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

// Status Effects are stored as entity components in Items.
// This function converts the component to the struct
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

// Every status effect/property has a Duration and a name
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
	s.MainProps.Duration -= 1
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
	MainProps   CommonItemProperties
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

	h := common.GetComponentType[*common.Attributes](c.Entity, common.AttributeComponent)

	h.CurrentHealth -= b.Temperature
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
	f.MainProps.Duration -= 1
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

// Throwable doesn't work like any other effects, so treating it as an "Effect" does not make much sense because we
type Throwable struct {
	MainProps     CommonItemProperties
	ThrowingRange int //How many tiles it can be thrown
	Damage        int
	Shape         graphics.TileBasedShape
	VX            graphics.VisualEffect
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

func (t *Throwable) InRange(endPos *common.Position) bool {

	//gd := graphics.NewScreenData()

	startPos := common.GridPositionFromPixels(t.Shape.StartPosition())

	return endPos.InRange(&startPos, t.ThrowingRange)

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

	ConsumableComponent = manager.NewComponent()
	ConsEffectTrackerComponent = manager.NewComponent()

	AllItemEffects = append(AllItemEffects, StickyComponent, BurningComponent, FreezingComponent, ThrowableComponent)

	items := ecs.BuildTag(ItemComponent, common.PositionComponent) //todo add all the tags
	tags["items"] = items

}
