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
func (c *Consumable) ApplyEffect(baseAttr *common.Attributes) {

	baseAttr.AttackBonus += c.AttrModifier.AttackBonus
	baseAttr.BaseArmorClass += c.AttrModifier.BaseArmorClass
	baseAttr.BaseDodgeChance += c.AttrModifier.BaseDodgeChance
	baseAttr.BaseProtection += c.AttrModifier.BaseProtection

	if c.AttrModifier.BaseMovementSpeed != 0 {
		baseAttr.BaseMovementSpeed = c.AttrModifier.BaseMovementSpeed
	}

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
		s += "Heals: " + strconv.Itoa(c.AttrModifier.CurrentHealth)
	}

	if c.AttrModifier.MaxHealth != 0 {
		s += "Max Health: " + strconv.Itoa(c.AttrModifier.MaxHealth)
	}

	if c.AttrModifier.AttackBonus != 0 {
		s += "Attack Bonus: " + strconv.Itoa(c.AttrModifier.AttackBonus)
	}

	if c.AttrModifier.BaseArmorClass != 0 {
		s += "Armor Class: " + strconv.Itoa(c.AttrModifier.BaseArmorClass)
	}

	if c.AttrModifier.BaseMovementSpeed != 0 {
		s += "Movemment Speed: " + strconv.Itoa(c.AttrModifier.BaseMovementSpeed)
	}

	if c.AttrModifier.BaseDodgeChance != 0 {
		s += "Dodge Chance: " + strconv.FormatFloat(float64(c.AttrModifier.BaseDodgeChance), 'f', 2, 32)
	}

	if c.AttrModifier.BaseProtection != 0 {
		s += "Protection: " + strconv.Itoa(c.AttrModifier.BaseProtection)
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
			attr.AttackBonus -= eff.Effect.AttrModifier.AttackBonus
			attr.MaxHealth -= eff.Effect.AttrModifier.MaxHealth
			attr.BaseArmorClass -= eff.Effect.AttrModifier.BaseArmorClass
			attr.BaseDodgeChance -= eff.Effect.AttrModifier.BaseDodgeChance
			attr.BaseMovementSpeed -= eff.Effect.AttrModifier.BaseMovementSpeed

			// A 0 movement speed causes issues with the ActionQueue due to the actionmanager not allowing actions with 0 cost
			if attr.BaseMovementSpeed == 0 {
				attr.BaseMovementSpeed = 1
			}

			attr.BaseProtection -= eff.Effect.AttrModifier.BaseProtection

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
// Consumables change the base attributes, so the TotalNNN stats need to be updated.
func UpdateEntityAttributes(e *ecs.Entity) {

	attr := common.GetComponentType[*common.Attributes](e, common.AttributeComponent)

	// Armor system removed - using only base attributes now
	attr.TotalArmorClass = attr.BaseArmorClass
	attr.TotalProtection = attr.BaseProtection
	attr.TotalDodgeChance = attr.BaseDodgeChance
	attr.TotalMovementSpeed = attr.BaseMovementSpeed

}
