package common

import (
	"fmt"
)

type Name struct {
	NameStr string
}

type UserMessage struct {
	AttackMessage       string
	GameStateMessage    string
	StatusEffectMessage string
}

// Attributes represents a unit's core stats and combat capabilities.
// Uses 6 core attributes that derive all combat stats.
type Attributes struct {
	// ========================================
	// CORE ATTRIBUTES (Primary Stats)
	// ========================================

	Strength   int // Physical Damage, Physical Resistance, Max HP
	Dexterity  int // Hit Rate, Crit Chance, Dodge
	Magic      int // Magic Damage, Healing Amount, Magic Defense
	Leadership int // Unit Capacity (squad size)
	Armor      int // Damage Reduction Modifier
	Weapon     int // Damage Increase Modifier

	// ========================================
	// RUNTIME STATE (Not Derived)
	// ========================================

	CurrentHealth int  // Current HP (changes during combat)
	MaxHealth     int  // Cached derived stat for performance
	CanAct        bool // Can unit act this turn
}

// NewAttributes creates a new Attributes instance with calculated MaxHealth
func NewAttributes(strength, dexterity, magic, leadership, armor, weapon int) Attributes {
	attr := Attributes{
		Strength:   strength,
		Dexterity:  dexterity,
		Magic:      magic,
		Leadership: leadership,
		Armor:      armor,
		Weapon:     weapon,
		CanAct:     true,
	}

	// Calculate and cache MaxHealth
	attr.MaxHealth = attr.GetMaxHealth()
	attr.CurrentHealth = attr.MaxHealth

	return attr
}

// ========================================
// DERIVED STAT METHODS (Physical Combat)
// ========================================

// GetPhysicalDamage calculates physical damage output
// Formula: (Strength / 2) + (Weapon * 2)
func (a *Attributes) GetPhysicalDamage() int {
	return (a.Strength / 2) + (a.Weapon * 2)
}

// GetPhysicalResistance calculates physical damage reduction
// Formula: (Strength / 4) + (Armor * 2)
func (a *Attributes) GetPhysicalResistance() int {
	return (a.Strength / 4) + (a.Armor * 2)
}

// GetMaxHealth calculates maximum health points
// Formula: 20 + (Strength * 2)
func (a *Attributes) GetMaxHealth() int {
	return 20 + (a.Strength * 2)
}

// ========================================
// DERIVED STAT METHODS (Accuracy & Avoidance)
// ========================================

// GetHitRate calculates chance to hit (0-100%)
// Formula: 80 + (Dexterity * 2), capped at 100
func (a *Attributes) GetHitRate() int {
	hitRate := 80 + (a.Dexterity * 2)
	if hitRate > 100 {
		hitRate = 100
	}
	return hitRate
}

// GetCritChance calculates critical hit chance (0-50%)
// Formula: Dexterity / 2, capped at 50
func (a *Attributes) GetCritChance() int {
	critChance := a.Dexterity / 2
	if critChance > 50 {
		critChance = 50
	}
	return critChance
}

// GetDodgeChance calculates dodge chance (0-40%)
// Formula: Dexterity / 3, capped at 40
func (a *Attributes) GetDodgeChance() int {
	dodge := a.Dexterity / 3
	if dodge > 40 {
		dodge = 40
	}
	return dodge
}

// ========================================
// DERIVED STAT METHODS (Magic System)
// ========================================

// GetMagicDamage calculates magic damage output
// Formula: Magic * 3
func (a *Attributes) GetMagicDamage() int {
	return a.Magic * 3
}

// GetHealingAmount calculates healing power
// Formula: Magic * 2
func (a *Attributes) GetHealingAmount() int {
	return a.Magic * 2
}

// GetMagicDefense calculates magic damage reduction
// Formula: Magic / 2
func (a *Attributes) GetMagicDefense() int {
	return a.Magic / 2
}

// ========================================
// DERIVED STAT METHODS (Squad System)
// ========================================

// GetUnitCapacity calculates maximum squad size
// Formula: 6 + (Leadership / 3), capped at 9
func (a *Attributes) GetUnitCapacity() int {
	capacity := 6 + (a.Leadership / 3)
	if capacity > 9 {
		capacity = 9
	}
	return capacity
}

// ========================================
// DISPLAY & UTILITY
// ========================================

// DisplayString formats attributes for player display
func (a Attributes) DisplayString() string {
	res := ""
	res += fmt.Sprintf("HP: %d/%d\n", a.CurrentHealth, a.MaxHealth)
	res += fmt.Sprintf("Strength: %d (Damage: %d, Resistance: %d)\n",
		a.Strength, a.GetPhysicalDamage(), a.GetPhysicalResistance())
	res += fmt.Sprintf("Dexterity: %d (Hit: %d%%, Crit: %d%%, Dodge: %d%%)\n",
		a.Dexterity, a.GetHitRate(), a.GetCritChance(), a.GetDodgeChance())

	if a.Magic > 0 {
		res += fmt.Sprintf("Magic: %d (Damage: %d, Healing: %d, Defense: %d)\n",
			a.Magic, a.GetMagicDamage(), a.GetHealingAmount(), a.GetMagicDefense())
	}

	if a.Leadership > 0 {
		res += fmt.Sprintf("Leadership: %d (Unit Capacity: %d)\n",
			a.Leadership, a.GetUnitCapacity())
	}

	res += fmt.Sprintf("Armor: %d | Weapon: %d\n", a.Armor, a.Weapon)
	return res
}
