package common

import (
	"game_main/config"
)

type Name struct {
	NameStr string
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
	// TURN-BASED COMBAT ATTRIBUTES
	// ========================================

	MovementSpeed int // Tiles per turn (default: 3)
	AttackRange   int // Attack distance in tiles (default: 1 for melee)

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
		Strength:      strength,
		Dexterity:     dexterity,
		Magic:         magic,
		Leadership:    leadership,
		Armor:         armor,
		Weapon:        weapon,
		MovementSpeed: 3, // Default movement
		AttackRange:   1, // Default melee
		CanAct:        true,
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
func (a *Attributes) GetPhysicalDamage() int {
	return (a.Strength / 2) + (a.Weapon * 2)
}

// GetPhysicalResistance calculates physical damage reduction
func (a *Attributes) GetPhysicalResistance() int {
	return (a.Strength / 4) + (a.Armor * 2)
}

// GetMaxHealth calculates maximum health points
func (a *Attributes) GetMaxHealth() int {
	return 20 + (a.Strength * 2)
}

// ========================================
// DERIVED STAT METHODS (Accuracy & Avoidance)
// ========================================

// GetHitRate calculates chance to hit (0-100%)
func (a *Attributes) GetHitRate() int {
	hitRate := config.DefaultBaseHitChance + (a.Dexterity * 2)
	if hitRate > config.DefaultMaxHitRate {
		hitRate = config.DefaultMaxHitRate
	}
	return hitRate
}

// GetCritChance calculates critical hit chance (0-50%)
func (a *Attributes) GetCritChance() int {
	critChance := a.Dexterity / 2
	if critChance > config.DefaultMaxCritChance {
		critChance = config.DefaultMaxCritChance
	}
	return critChance
}

// GetDodgeChance calculates dodge chance (0-40%)
func (a *Attributes) GetDodgeChance() int {
	dodge := a.Dexterity / 3
	if dodge > config.DefaultMaxDodgeChance {
		dodge = config.DefaultMaxDodgeChance
	}
	return dodge
}

// ========================================
// DERIVED STAT METHODS (Magic System)
// ========================================

// GetMagicDamage calculates magic damage output
func (a *Attributes) GetMagicDamage() int {
	return a.Magic * 3
}

// GetMagicDefense calculates magic damage reduction
func (a *Attributes) GetMagicDefense() int {
	return a.Magic / 2
}

// GetHealingAmount calculates healing power
func (a *Attributes) GetHealingAmount() int {
	return a.Magic * 2
}

// ========================================
// DERIVED STAT METHODS (Squad System)
// ========================================

// GetUnitCapacity calculates maximum squad size (total capacity)
func (a *Attributes) GetUnitCapacity() int {
	capacity := config.DefaultBaseCapacity + (a.Leadership / 3)
	if capacity > config.DefaultMaxCapacity {
		capacity = config.DefaultMaxCapacity
	}
	return capacity
}

// GetCapacityCost calculates how much capacity this unit consumes in a squad
// Stronger units cost more capacity to field
func (a *Attributes) GetCapacityCost() float64 {
	return float64(a.Strength+a.Weapon+a.Armor) / 5.0
}

// ========================================
// DERIVED STAT METHODS (Turn-Based Combat)
// ========================================

// GetMovementSpeed returns tiles per turn with default
func (a *Attributes) GetMovementSpeed() int {
	if a.MovementSpeed <= 0 {
		return config.DefaultMovementSpeed // Default movement speed
	}
	return a.MovementSpeed
}

// GetAttackRange returns attack distance with default
func (a *Attributes) GetAttackRange() int {
	if a.AttackRange <= 0 {
		return config.DefaultAttackRange // Default melee range
	}
	return a.AttackRange
}
