package gear

import (
	"game_main/common"
	"strconv"

	"github.com/bytearena/ecs"
)

var ConsumableComponent *ecs.Component

// A consumable applies the attrMod to the baseAttr.
// For example, a Healing Potion would add the attrMod.CurrentHealth to the,
// or a speed potion would increase the BaseMovementSpeed
// The other option was to use a ConsumableEffects inteface
// For now, let's keep it simple

type Consumable struct {
	Name         string
	AttrModifier common.Attributes
}

// Only effects the Base Attributes, current health, and max health
func (c *Consumable) ApplyEffect(baseAttr *common.Attributes) {

	baseAttr.CurrentHealth += baseAttr.CurrentHealth
	baseAttr.MaxHealth += baseAttr.MaxHealth
	baseAttr.AttackBonus += baseAttr.AttackBonus
	baseAttr.BaseArmorClass += baseAttr.BaseArmorClass
	baseAttr.BaseMovementSpeed += baseAttr.BaseMovementSpeed
	baseAttr.BaseDodgeChance += baseAttr.BaseDodgeChance
	baseAttr.BaseProtection += baseAttr.BaseProtection

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

func InitializeConsumableComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	ConsumableComponent = manager.NewComponent()

}
