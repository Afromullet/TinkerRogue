package encounter

import (
	"game_main/common"
	"game_main/tactical/squads"
	"github.com/bytearena/ecs"
	"math"
)

// CalculateUnitPower computes the power rating for a single unit
// Uses configurable weights to balance offensive, defensive, and utility stats
// Returns a float64 power score (higher = more powerful)
func CalculateUnitPower(
	unitID ecs.EntityID,
	manager *common.EntityManager,
	config *EvaluationConfigData,
) float64 {
	entity := manager.FindEntityByID(unitID)
	if entity == nil {
		return 0.0
	}

	// Get required components
	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr == nil {
		return 0.0
	}

	roleData := common.GetComponentType[*squads.UnitRoleData](entity, squads.UnitRoleComponent)
	if roleData == nil {
		return 0.0
	}

	// Calculate offensive power
	offensivePower := calculateOffensivePower(attr, config)

	// Calculate defensive power
	defensivePower := calculateDefensivePower(attr, config)

	// Calculate utility power
	utilityPower := calculateUtilityPower(entity, attr, roleData, config, manager)

	// Weighted sum
	totalPower := (offensivePower * config.OffensiveWeight) +
		(defensivePower * config.DefensiveWeight) +
		(utilityPower * config.UtilityWeight)

	return totalPower
}

// calculateOffensivePower evaluates a unit's damage output potential
func calculateOffensivePower(attr *common.Attributes, config *EvaluationConfigData) float64 {
	// Damage component (physical + magic)
	physicalDamage := float64(attr.GetPhysicalDamage())
	magicDamage := float64(attr.GetMagicDamage())
	avgDamage := (physicalDamage + magicDamage) / 2.0

	// Accuracy component (hit rate and crit chance)
	hitRate := float64(attr.GetHitRate()) / 100.0 // Normalize to 0-1
	critChance := float64(attr.GetCritChance()) / 100.0
	critMultiplier := 1.0 + (critChance * CritDamageMultiplier) // Expected damage multiplier from crits

	effectiveDamage := avgDamage * hitRate * critMultiplier

	// Sub-weighted combination
	damageComponent := avgDamage * config.DamageWeight
	accuracyComponent := effectiveDamage * config.AccuracyWeight

	return damageComponent + accuracyComponent
}

// calculateDefensivePower evaluates a unit's survivability
func calculateDefensivePower(attr *common.Attributes, config *EvaluationConfigData) float64 {
	// Health component (current and max)
	maxHP := float64(attr.GetMaxHealth())
	currentHP := float64(attr.CurrentHealth)
	healthRatio := currentHP / math.Max(maxHP, 1.0)

	effectiveHealth := maxHP * healthRatio

	// Resistance component (physical + magic)
	physicalResist := float64(attr.GetPhysicalResistance())
	magicResist := float64(attr.GetMagicDefense())
	avgResistance := (physicalResist + magicResist) / 2.0

	// Avoidance component (dodge)
	dodgeChance := float64(attr.GetDodgeChance()) / 100.0

	// Sub-weighted combination
	healthComponent := effectiveHealth * config.HealthWeight
	resistanceComponent := avgResistance * config.ResistanceWeight
	avoidanceComponent := dodgeChance * DodgeScalingFactor * config.AvoidanceWeight // Scale dodge to 0-40 range

	return healthComponent + resistanceComponent + avoidanceComponent
}

// calculateUtilityPower evaluates a unit's support and tactical value
func calculateUtilityPower(
	entity *ecs.Entity,
	attr *common.Attributes,
	roleData *squads.UnitRoleData,
	config *EvaluationConfigData,
	manager *common.EntityManager,
) float64 {
	// Calculate individual utility components using helper functions
	roleComponent := calculateRoleValue(roleData) * config.RoleWeight
	abilityComponent := calculateAbilityValue(entity) * config.AbilityWeight
	coverComponent := calculateCoverValue(entity) * config.CoverWeight

	return roleComponent + abilityComponent + coverComponent
}
