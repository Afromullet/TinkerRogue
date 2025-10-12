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
	BURNING_NAME  = "Burning"
	FREEZING_NAME = "Freezing"
	STICKY_NAME   = "Sticky"
)

var (
	EffectNames = []string{BURNING_NAME, FREEZING_NAME, STICKY_NAME}

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
// StackEffect decides how to stack the effect if it's applied again. Need to do a type assertion on the any.
// Copy creates a shallow copy
// Even though all of them have the name and duration in the common properties,
// we have it as a method of the interface so that we can access it without type assertions
type StatusEffects interface {
	StatusEffectComponent() *ecs.Component
	StatusEffectName() string
	Duration() int
	ApplyToCreature(c *ecs.QueryResult)
	DisplayString() string
	StackEffect(eff any)
	Copy() StatusEffects
	//Todo for the future, figure out if you can make this an interface
	//Figure out how you have to implement methods for nested interfaces

	common.Quality
}

func GetVisualEffect(eff StatusEffects) graphics.VisualEffect {
	switch eff.(type) {
	case *Burning: // Check if it's of type Burning
		return graphics.NewFireEffect(0, 0, 1, 2, 1, 0.5)
	case *Freezing: // Check if it's of type Freezing
		return graphics.NewIceEffect(0, 0, 2)
	case *Sticky: // Check if it's of type Sticky
		return graphics.NewStickyGroundEffect(0, 0, 2)
	default:
		// Invalid status effect type
		return nil
	}
}

// Any item that implements the interface defines the random ranges of the item stats

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
	Quality  common.QualityType
}

// Adds the duration of the other to the CommonItemProperty
func (c *CommonItemProperties) AddDuration(other CommonItemProperties) {

	c.Duration += other.Duration

}

func (c *CommonItemProperties) QualityName() string {

	if c.Quality == common.LowQuality {
		return "Low Quality"
	} else if c.Quality == common.NormalQuality {
		return "Medium Quality"
	} else if c.Quality == common.HighQuality {
		return "High Quality"
	} else {
		return "Invalid Quality"
	}
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

// Takes the larger spread.
func (s *Sticky) StackEffect(eff any) {

	e := eff.(*Sticky)
	e.MainProps.AddDuration(e.MainProps)

	if s.Spread < e.Spread {
		s.Spread = e.Spread
	}

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

// Using a closure to track the original dexterity so that we don't have to track that outside of the function
func (s *Sticky) ApplyToCreature(c *ecs.QueryResult) {
	var originalDexterity int
	var initialized bool

	applyEffect := func(c *ecs.QueryResult) {
		attr := common.GetComponentType[*common.Attributes](c.Entity, common.AttributeComponent)

		if !initialized {
			originalDexterity = attr.Dexterity
			initialized = true
		}

		// Sticky reduces dexterity (agility) by 5
		attr.Dexterity -= 5

		if attr.Dexterity <= 0 {
			attr.Dexterity = 1
		}
		s.MainProps.Duration--

		if s.MainProps.Duration == 0 {
			attr.Dexterity = originalDexterity
		}
	}

	applyEffect(c)

}
func (s *Sticky) DisplayString() string {
	result := ""
	result += fmt.Sprintln("Movement slowed down by stickiness")

	return result
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

func (b *Burning) StackEffect(eff any) {

	e := eff.(*Burning)
	e.MainProps.AddDuration(e.MainProps)

	b.Temperature += e.Temperature

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

func (b *Burning) DisplayString() string {
	result := ""
	result += fmt.Sprintln("Burning with a temperature of ", b.Temperature)

	return result
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

func (f *Freezing) StackEffect(eff any) {

	e := eff.(*Freezing)
	e.MainProps.AddDuration(e.MainProps)

	f.Thickness += e.Thickness

}

func (f *Freezing) Copy() StatusEffects {
	return &Freezing{
		MainProps: f.MainProps,
		Thickness: f.Thickness,
	}
}

func (f *Freezing) ApplyToCreature(c *ecs.QueryResult) {

	attr := common.GetComponentType[*common.Attributes](c.Entity, common.AttributeComponent)

	if f.MainProps.Duration > 0 {
		attr.CanAct = false
	} else {
		attr.CanAct = true
	}

	f.MainProps.Duration -= 1
}

func (f *Freezing) DisplayString() string {
	result := ""
	result += fmt.Sprintln("Frozen Effect Active")

	return result
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

// Throwable functionality has been moved to itemactions.go as ThrowableAction
// This removes the forced coupling between throwables and status effects
// Use item.GetThrowableAction() instead of string-based lookups

func InitializeItemComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	ItemComponent = manager.NewComponent()
	StickyComponent = manager.NewComponent()
	BurningComponent = manager.NewComponent()
	FreezingComponent = manager.NewComponent()

	ThrowableComponent = manager.NewComponent()

	AllItemEffects = append(AllItemEffects, StickyComponent, BurningComponent, FreezingComponent)

	items := ecs.BuildTag(ItemComponent, common.PositionComponent) //todo add all the tags
	tags["items"] = items

}
