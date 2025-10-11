package gear

import (
	"game_main/common"
	"math/rand"
)

// ConsumableType helps us tell potions apart.
type ConsumableType int

const (
	HealingPotion = iota
	ProtectionPotion
	SpeedPotion
)

func (c *Consumable) CreateConsumable(consType ConsumableType, q common.QualityType) {

	if consType == HealingPotion {
		c.QualityHealingPotion(q)

	} else if consType == ProtectionPotion {
		c.QualityProtectionPotion(q)

	} else if consType == SpeedPotion {
		c.QualitySpeedPotion(q)

	}

}

func (c *Consumable) QualityHealingPotion(q common.QualityType) *Consumable {

	c.Duration = 1 //All healing potions have a duration of 1
	if q == common.LowQuality {

		c.Name = "Light Healing Potion"
		c.Duration = rand.Intn(10) + 1
		c.AttrModifier.CurrentHealth = rand.Intn(5) + 1
	} else if q == common.NormalQuality {

		c.Name = "Moderate Healing Potion"
		c.Duration = rand.Intn(15) + 1
		c.AttrModifier.CurrentHealth = rand.Intn(5) + 1
	} else if q == common.HighQuality {

		c.Name = "Strong Healing Potion"
		c.Duration = rand.Intn(30) + 1
		c.AttrModifier.CurrentHealth = rand.Intn(5) + 1
	}

	return c

}

func (c *Consumable) QualityProtectionPotion(q common.QualityType) *Consumable {

	if q == common.LowQuality {
		c.Name = "Light Protection Potion"
		c.Duration = rand.Intn(3) + 1
		c.AttrModifier.Armor = rand.Intn(5) + 1
	} else if q == common.NormalQuality {
		c.Name = "Moderate Protection Potion"
		c.Duration = rand.Intn(5) + 1
		c.AttrModifier.Armor = rand.Intn(15) + 1
	} else if q == common.HighQuality {
		c.Name = "Strong Protection Potion"
		c.Duration = rand.Intn(10) + 1
		c.AttrModifier.Armor = rand.Intn(25) + 1
	}

	return c

}

func (c *Consumable) QualitySpeedPotion(q common.QualityType) *Consumable {

	if q == common.LowQuality {
		c.Name = "Light Speed Potion"
		c.Duration = rand.Intn(3) + 1
		c.AttrModifier.Dexterity = rand.Intn(5) + 1
	} else if q == common.NormalQuality {
		c.Name = "Moderate Speed Potion"
		c.Duration = rand.Intn(5) + 1
		c.AttrModifier.Dexterity = rand.Intn(5) + 1
	} else if q == common.HighQuality {
		c.Name = "Strong Speed Potion"
		c.Duration = rand.Intn(7) + 1
		c.AttrModifier.Dexterity = rand.Intn(5) + 1
	}

	return c

}
