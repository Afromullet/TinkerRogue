package gear

import (
	"game_main/common"
	"strconv"

	"github.com/bytearena/ecs"
)

var ConsumableComponent *ecs.Component
var ConsEffectTrackerComponent *ecs.Component

// A consumable applies the attrMod to the baseAttr.
// For example, a Healing Potion would add the attrMod.CurrentHealth to the,
// or a speed potion would increase the BaseMovementSpeed
// The other option was to use a ConsumableEffects inteface
// For now, let's keep it simple

type Consumable struct {
	Name         string
	AttrModifier common.Attributes
	Duration     int
}

// Non health effects are applied every turn. Others only once
func (c *Consumable) ApplyEffect(baseAttr *common.Attributes) {
	baseAttr.MaxHealth += c.AttrModifier.MaxHealth
	baseAttr.AttackBonus += c.AttrModifier.AttackBonus
	baseAttr.BaseArmorClass += c.AttrModifier.BaseArmorClass
	baseAttr.BaseMovementSpeed = c.AttrModifier.BaseMovementSpeed //Set the movement speed to the value. Adding it makes us slower
	baseAttr.BaseDodgeChance += c.AttrModifier.BaseDodgeChance
	baseAttr.BaseProtection += c.AttrModifier.BaseProtection

}

func (c *Consumable) ApplyHealingEffect(baseAttr *common.Attributes) {
	baseAttr.CurrentHealth += c.AttrModifier.CurrentHealth

}

// Gets a string representing the consumable effects
func (c Consumable) ConsumableInfo() string {
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

// Applies the ConsumableEffect to the entity
type ConsumableEffect struct {
	currentDuration int
	Effect          Consumable
}

func (eff *ConsumableEffect) Apply(e *ecs.Entity) {
	baseAttr := common.GetComponentType[*common.Attributes](e, common.AttributeComponent)
	//Healing is applied every turn.
	//Everything else only at the start. Otherwise we'd keep on increasing the base attributes
	if eff.currentDuration == 0 {
		eff.Effect.ApplyEffect(baseAttr)
		eff.Effect.ApplyHealingEffect(baseAttr)

	} else {

		eff.Effect.ApplyHealingEffect(baseAttr)

	}

	eff.currentDuration++

}

func (eff ConsumableEffect) IsDone() bool {
	return eff.currentDuration == eff.Effect.Duration

}

// An entity can have more than once consumable effect
type ConsumableEffectTracker struct {
	effects []ConsumableEffect
}

func (ce *ConsumableEffectTracker) AddEffect(cons ConsumableEffect) {

	ce.effects = append(ce.effects, cons)

}

func (ce *ConsumableEffectTracker) ApplyEffects(ent *ecs.Entity) {

	remainingEffects := make([]ConsumableEffect, 0)
	attr := common.GetComponentType[*common.Attributes](ent, common.AttributeComponent)

	for _, eff := range ce.effects {

		eff.Apply(ent)

		if !eff.IsDone() {
			remainingEffects = append(remainingEffects, eff)

		} else {

			//eff.RestoreAttributes(ent)

			attr.AttackBonus -= eff.Effect.AttrModifier.AttackBonus
			attr.MaxHealth -= eff.Effect.AttrModifier.MaxHealth
			attr.BaseArmorClass -= eff.Effect.AttrModifier.BaseArmorClass
			attr.BaseDodgeChance -= eff.Effect.AttrModifier.BaseDodgeChance
			attr.BaseMovementSpeed -= eff.Effect.AttrModifier.BaseMovementSpeed

			if attr.BaseMovementSpeed == 0 {
				attr.BaseMovementSpeed = 1
			}

			attr.BaseProtection -= eff.Effect.AttrModifier.BaseProtection

		}
	}

	ce.effects = remainingEffects
}

func (ce *ConsumableEffectTracker) HasEffects() bool {
	return len(ce.effects) > 0
}
func AddEffectToTracker(ent *ecs.Entity, cons Consumable) {

	tracker := common.GetComponentType[*ConsumableEffectTracker](ent, ConsEffectTrackerComponent)

	eff := ConsumableEffect{
		currentDuration: 0,
		Effect:          cons,
	}

	if tracker == nil {

		tracker = &ConsumableEffectTracker{
			effects: make([]ConsumableEffect, 0),
		}

		ent.AddComponent(ConsEffectTrackerComponent, tracker)

	}

	tracker.AddEffect(eff)

}

func RunEffectTracker(ent *ecs.Entity) {
	tracker := common.GetComponentType[*ConsumableEffectTracker](ent, ConsEffectTrackerComponent)

	if tracker != nil && tracker.HasEffects() {
		tracker.ApplyEffects(ent)
	}

	UpdateEntityAttributes(ent)

}

// Does not fit in the common package because referencing gear will cause a circular inclusion issue
func UpdateEntityAttributes(e *ecs.Entity) {

	armor := common.GetComponentType[*Armor](e, ArmorComponent)
	attr := common.GetComponentType[*common.Attributes](e, common.AttributeComponent)

	ac := 0
	prot := 0
	dodge := float32(0.0)

	if armor != nil {

		ac = armor.ArmorClass
		prot = armor.Protection
		dodge = float32(armor.DodgeChance)
	}

	attr.TotalArmorClass = attr.BaseArmorClass + ac
	attr.TotalProtection = attr.BaseProtection + prot
	attr.TotalDodgeChance = attr.BaseDodgeChance + dodge

	//Nothing else affecting these
	attr.TotalMovementSpeed = attr.BaseMovementSpeed

}
