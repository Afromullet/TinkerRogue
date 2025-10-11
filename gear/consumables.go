package gear

import (
	"game_main/common"
	"strconv"

	"github.com/bytearena/ecs"
)

var ConsumableComponent *ecs.Component
var ConsEffectTrackerComponent *ecs.Component

// A consumable applies the attrMod to the baseAttr. The attrMod is the "buff" the consumable provides
// For example, a Healing Potion would add to attrMod.CurrentHealth
// And a speed potion would increase the BaseMovementSpeed

type Consumable struct {
	Name         string
	AttrModifier common.Attributes
	Duration     int
}

// Anything other than health is applied every turn.
// Applies temporary buffs to core attributes
func (c *Consumable) ApplyEffect(baseAttr *common.Attributes) {
	baseAttr.Weapon += c.AttrModifier.Weapon
	baseAttr.Armor += c.AttrModifier.Armor
	baseAttr.Strength += c.AttrModifier.Strength
	baseAttr.Dexterity += c.AttrModifier.Dexterity
	baseAttr.Magic += c.AttrModifier.Magic
}

func (c *Consumable) ApplyHealingEffect(baseAttr *common.Attributes) {
	baseAttr.CurrentHealth += c.AttrModifier.CurrentHealth
	baseAttr.MaxHealth += c.AttrModifier.MaxHealth

}

// For displaying consumable info in the GUI
func (c Consumable) DisplayString() string {
	s := ""

	s += "Name " + c.Name + "\n"

	if c.AttrModifier.CurrentHealth != 0 {
		s += "Heals: " + strconv.Itoa(c.AttrModifier.CurrentHealth) + "\n"
	}

	if c.AttrModifier.MaxHealth != 0 {
		s += "Max Health: " + strconv.Itoa(c.AttrModifier.MaxHealth) + "\n"
	}

	if c.AttrModifier.Strength != 0 {
		s += "Strength: +" + strconv.Itoa(c.AttrModifier.Strength) + "\n"
	}

	if c.AttrModifier.Dexterity != 0 {
		s += "Dexterity: +" + strconv.Itoa(c.AttrModifier.Dexterity) + "\n"
	}

	if c.AttrModifier.Magic != 0 {
		s += "Magic: +" + strconv.Itoa(c.AttrModifier.Magic) + "\n"
	}

	if c.AttrModifier.Armor != 0 {
		s += "Armor: +" + strconv.Itoa(c.AttrModifier.Armor) + "\n"
	}

	if c.AttrModifier.Weapon != 0 {
		s += "Weapon: +" + strconv.Itoa(c.AttrModifier.Weapon) + "\n"
	}

	return s
}

// ConsumableEffect tracks the duration of an effect.
// Used by the ConsumableEffectTracker
type ConsumableEffect struct {
	currentDuration int
	Effect          Consumable
}

// Todo replace baseAttr with GetAttributes from common
func (eff *ConsumableEffect) Apply(e *ecs.Entity) {
	baseAttr := common.GetComponentType[*common.Attributes](e, common.AttributeComponent)

	//Non-health consumables are applied only once. Health consumable are applied at the Start, and last until the end of the duration.
	//(I.E, a regeneration potion would have a duration that lasts over mulitple turns)
	//
	if eff.currentDuration == 0 {
		eff.Effect.ApplyEffect(baseAttr)
		eff.Effect.ApplyHealingEffect(baseAttr)

	} else {

		eff.Effect.ApplyHealingEffect(baseAttr)

	}

	eff.currentDuration++

}

// The effect duration expired. Used by the ConsumableEffectTracker
func (eff ConsumableEffect) IsDone() bool {
	return eff.currentDuration == eff.Effect.Duration

}

// Every effect on an entity is tracked with the ConsumableEffects.
// It hnadles applying, adding, and removing effects when they're done.
type ConsumableEffects struct {
	effects []ConsumableEffect
}

func (ce *ConsumableEffects) AddEffect(cons ConsumableEffect) {

	ce.effects = append(ce.effects, cons)

}

func (ce *ConsumableEffects) ApplyEffects(ent *ecs.Entity) {

	remainingEffects := make([]ConsumableEffect, 0)
	attr := common.GetComponentType[*common.Attributes](ent, common.AttributeComponent)

	for _, eff := range ce.effects {

		eff.Apply(ent)

		if !eff.IsDone() {
			remainingEffects = append(remainingEffects, eff)

		} else {

			//Restore everything to the original state except CurrentHealth.
			attr.Weapon -= eff.Effect.AttrModifier.Weapon
			attr.Armor -= eff.Effect.AttrModifier.Armor
			attr.Strength -= eff.Effect.AttrModifier.Strength
			attr.Dexterity -= eff.Effect.AttrModifier.Dexterity
			attr.Magic -= eff.Effect.AttrModifier.Magic
			attr.MaxHealth -= eff.Effect.AttrModifier.MaxHealth

		}
	}

	ce.effects = remainingEffects
}

// Adds a consumable to an entities ConsumableEffectTracker.
// Anything that can use or be afffected by a consumable will use a ConsumablEffeftTracker
func AddEffectToTracker(ent *ecs.Entity, cons Consumable) {

	tracker := common.GetComponentType[*ConsumableEffects](ent, ConsEffectTrackerComponent)

	eff := ConsumableEffect{
		currentDuration: 0,
		Effect:          cons,
	}

	if tracker == nil {

		tracker = &ConsumableEffects{
			effects: make([]ConsumableEffect, 0),
		}

		ent.AddComponent(ConsEffectTrackerComponent, tracker)

	}

	tracker.AddEffect(eff)

}

func RunEffectTracker(ent *ecs.Entity) {

	//To get a random position for spawning the item

	tracker := common.GetComponentType[*ConsumableEffects](ent, ConsEffectTrackerComponent)

	if tracker != nil && len(tracker.effects) > 0 {
		tracker.ApplyEffects(ent)
	}

	UpdateEntityAttributes(ent)

}

// Does not fit in the common package because referencing gear will cause a circular inclusion issue
// Consumables change the core attributes - derived stats are calculated automatically
func UpdateEntityAttributes(e *ecs.Entity) {
	// With new attribute system, derived stats are calculated on-demand via methods
	// No need to update "Total" fields since they no longer exist
	// MaxHealth is cached, so recalculate it
	attr := common.GetComponentType[*common.Attributes](e, common.AttributeComponent)
	attr.MaxHealth = attr.GetMaxHealth()
}
