package evaluation

import (
	"game_main/common"
	gameconfig "game_main/config"
	"game_main/tactical/squads"
	"math"

	"github.com/bytearena/ecs"
)

// CalculateUnitPower computes the power rating for a single unit.
// Uses configurable weights to balance offensive, defensive, and utility stats.
// Returns a float64 power score (higher = more powerful).
// Used by both encounter generation and AI threat assessment.
func CalculateUnitPower(
	unitID ecs.EntityID,
	manager *common.EntityManager,
	config *PowerConfig,
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
	offensivePower := CalculateOffensivePower(attr, config)

	// Calculate defensive power
	defensivePower := CalculateDefensivePower(attr, config)

	// Calculate utility power
	utilityPower := CalculateUtilityPower(entity, attr, roleData, config)

	// Weighted sum
	totalPower := (offensivePower * config.OffensiveWeight) +
		(defensivePower * config.DefensiveWeight) +
		(utilityPower * config.UtilityWeight)

	return totalPower
}

// CalculateOffensivePower evaluates a unit's damage output potential.
// Returns expected damage per attack accounting for hit rate and crits.
func CalculateOffensivePower(attr *common.Attributes, config *PowerConfig) float64 {
	// Average of physical and magic damage
	physicalDamage := float64(attr.GetPhysicalDamage())
	magicDamage := float64(attr.GetMagicDamage())
	avgDamage := (physicalDamage + magicDamage) / 2.0

	// Calculate expected damage (damage * hit rate * crit multiplier)
	hitRate := float64(attr.GetHitRate()) / 100.0
	critChance := float64(attr.GetCritChance()) / 100.0
	critMultiplier := 1.0 + (critChance * gameconfig.CritDamageBonus)

	return avgDamage * hitRate * critMultiplier
}

// CalculateDefensivePower evaluates a unit's survivability.
// Returns effective HP accounting for current health, resistance, and dodge.
func CalculateDefensivePower(attr *common.Attributes, config *PowerConfig) float64 {
	// Effective health based on current HP
	maxHP := float64(attr.GetMaxHealth())
	currentHP := float64(attr.CurrentHealth)
	healthRatio := currentHP / math.Max(maxHP, 1.0)
	effectiveHealth := maxHP * healthRatio

	// Average resistance provides damage reduction
	physicalResist := float64(attr.GetPhysicalResistance())
	magicResist := float64(attr.GetMagicDefense())
	avgResistance := (physicalResist + magicResist) / 2.0

	// Dodge provides effective HP multiplier: HP / (1 - dodgeChance)
	// e.g., 20% dodge = HP / 0.8 = 1.25x effective HP
	dodgeChance := float64(attr.GetDodgeChance()) / 100.0
	dodgeMultiplier := 1.0 / math.Max(1.0-dodgeChance, 0.5) // Cap at 2x for 50% dodge

	// Combine: effective HP * dodge multiplier + resistance bonus
	// Resistance adds flat survivability (roughly 1 HP per point)
	return (effectiveHealth * dodgeMultiplier) + avgResistance
}

// CalculateUtilityPower evaluates a unit's support and tactical value.
// Sums role value, leader abilities, and cover provision.
func CalculateUtilityPower(
	entity *ecs.Entity,
	attr *common.Attributes,
	roleData *squads.UnitRoleData,
	config *PowerConfig,
) float64 {
	// Sum all utility components (no sub-weights, just add them together)
	return calculateRoleValue(roleData) + calculateAbilityValue(entity) + calculateCoverValue(entity)
}

// calculateRoleValue returns power value based on unit role.
func calculateRoleValue(roleData *squads.UnitRoleData) float64 {
	roleMultiplier := GetRoleMultiplierFromConfig(roleData.Role)
	scalingConstants := GetScalingConstants()
	return roleMultiplier * scalingConstants.RoleScaling
}

// calculateAbilityValue returns power value from leader abilities.
func calculateAbilityValue(entity *ecs.Entity) float64 {
	// Check if unit is a leader
	if !entity.HasComponent(squads.LeaderComponent) {
		return 0.0
	}

	// Get ability slots to find equipped abilities
	abilitySlots := common.GetComponentType[*squads.AbilitySlotData](entity, squads.AbilitySlotComponent)
	if abilitySlots == nil {
		return 0.0
	}

	// Sum power values from all equipped abilities
	totalAbilityPower := 0.0
	for _, slot := range abilitySlots.Slots {
		if slot.IsEquipped {
			totalAbilityPower += GetAbilityPowerValue(slot.AbilityType)
		}
	}
	return totalAbilityPower
}

// calculateCoverValue returns power value from cover provision.
func calculateCoverValue(entity *ecs.Entity) float64 {
	coverData := common.GetComponentType[*squads.CoverData](entity, squads.CoverComponent)
	if coverData == nil {
		return 0.0
	}

	// Cover value scaled by how many units it can protect
	scalingConstants := GetScalingConstants()
	return coverData.CoverValue * scalingConstants.CoverScaling * scalingConstants.CoverBeneficiaryMultiplier
}

// CalculateSquadPower computes the power rating for a full squad.
// Aggregates unit power scores and applies squad-level modifiers.
func CalculateSquadPower(
	squadID ecs.EntityID,
	manager *common.EntityManager,
	config *PowerConfig,
) float64 {
	squadEntity := squads.GetSquadEntity(squadID, manager)
	if squadEntity == nil {
		return 0.0
	}

	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	if squadData == nil {
		return 0.0
	}

	// Sum unit power scores
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	if len(unitIDs) == 0 {
		return 0.0
	}

	totalUnitPower := 0.0
	for _, unitID := range unitIDs {
		unitPower := CalculateUnitPower(unitID, manager, config)
		totalUnitPower += unitPower
	}

	// Apply squad-level modifiers
	basePower := totalUnitPower

	// Composition bonus (attack type diversity)
	compositionMod := CalculateSquadCompositionBonus(squadID, manager)
	basePower *= compositionMod

	// Health penalty (low HP squads are less effective)
	basePower *= CalculateHealthMultiplier(
		squads.GetSquadHealthPercent(squadID, manager),
		config.HealthPenalty,
	)

	return basePower
}

// CalculateSquadCompositionBonus evaluates attack type diversity.
// Uses shared CompositionBonuses from roles.go (no longer configurable per-profile).
func CalculateSquadCompositionBonus(
	squadID ecs.EntityID,
	manager *common.EntityManager,
) float64 {
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	attackTypes := make(map[squads.AttackType]bool)

	for _, unitID := range unitIDs {
		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		targetRowData := common.GetComponentType[*squads.TargetRowData](entity, squads.TargetRowComponent)
		if targetRowData != nil {
			attackTypes[targetRowData.AttackType] = true
		}
	}

	return GetCompositionBonusFromConfig(len(attackTypes))
}

// CalculateHealthMultiplier returns power multiplier based on squad health.
func CalculateHealthMultiplier(healthPercent float64, healthPenalty float64) float64 {
	// healthPenalty acts as an exponent - lower health = less power
	// e.g., 50% health with penalty 2.0 = 0.5^2 = 0.25 power
	// Note: With penalty=2.0, even at 10% health the multiplier is 0.01,
	// which is functionally near-zero but never exactly zero.
	return math.Pow(healthPercent, healthPenalty)
}


// CalculateSquadPowerByRange computes power contribution at each attack range.
// Used by AI threat assessment to understand how dangerous a squad is at different distances.
// Returns a map of range -> power where power indicates threat at that range.
func CalculateSquadPowerByRange(
	squadID ecs.EntityID,
	manager *common.EntityManager,
	config *PowerConfig,
) map[int]float64 {
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	if len(unitIDs) == 0 {
		return nil
	}

	// Collect unit data with attack ranges
	type unitPowerData struct {
		power       float64
		attackRange int
		isLeader    bool
	}
	units := []unitPowerData{}
	attackTypeCount := make(map[squads.AttackType]int)

	movementRange := squads.GetSquadMovementSpeed(squadID, manager)

	for _, unitID := range unitIDs {
		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil {
			continue
		}

		roleData := common.GetComponentType[*squads.UnitRoleData](entity, squads.UnitRoleComponent)
		if roleData == nil {
			continue
		}

		// Get attack range
		attackRange := 1
		if rangeData := common.GetComponentType[*squads.AttackRangeData](entity, squads.AttackRangeComponent); rangeData != nil {
			attackRange = rangeData.Range
		}

		// Track attack types for composition bonus
		if targetRowData := common.GetComponentType[*squads.TargetRowData](entity, squads.TargetRowComponent); targetRowData != nil {
			attackTypeCount[targetRowData.AttackType]++
		}

		// Calculate base unit power (simplified - weapon + dex/2 for threat)
		basePower := float64(attr.Weapon + attr.Dexterity/2)
		roleMultiplier := GetRoleMultiplierFromConfig(roleData.Role)

		units = append(units, unitPowerData{
			power:       basePower * roleMultiplier,
			attackRange: attackRange,
			isLeader:    entity.HasComponent(squads.LeaderComponent),
		})
	}

	if len(units) == 0 {
		return nil
	}

	// Find maximum threat range (movement + attack)
	maxThreatRange := 0
	for _, ud := range units {
		threatRange := movementRange + ud.attackRange
		if threatRange > maxThreatRange {
			maxThreatRange = threatRange
		}
	}

	// Calculate power at each range from 1 to maxThreatRange
	// Units threaten a range if movement + attack >= currentRange
	powerByRange := make(map[int]float64, maxThreatRange)

	for currentRange := 1; currentRange <= maxThreatRange; currentRange++ {
		var rangePower float64 = 0

		// Sum power from units that can threaten this range
		for _, ud := range units {
			effectiveThreatRange := movementRange + ud.attackRange

			if effectiveThreatRange >= currentRange {
				// Apply leader bonus
				leaderBonus := 1.0
				if ud.isLeader {
					leaderBonus = GetLeaderBonusFromConfig()
				}

				// Calculate unit power contribution at this range
				unitPower := ud.power * leaderBonus
				rangePower += unitPower
			}
		}

		powerByRange[currentRange] = rangePower
	}

	// Apply composition bonus to each range
	compositionBonus := GetCompositionBonusFromConfig(len(attackTypeCount))
	for range_, power := range powerByRange {
		powerByRange[range_] = power * compositionBonus
	}

	return powerByRange
}

// EstimateUnitPowerFromTemplate calculates power for a UnitTemplate (no ECS entity).
// Used by encounter generation when building squads from templates.
func EstimateUnitPowerFromTemplate(unit squads.UnitTemplate, config *PowerConfig) float64 {
	attr := &unit.Attributes

	// === OFFENSIVE POWER ===
	offensivePower := CalculateOffensivePower(attr, config)

	// === DEFENSIVE POWER ===
	// NOTE: Uses full HP assumption for templates (no current HP state)
	maxHP := float64(attr.GetMaxHealth())
	effectiveHealth := maxHP // Assume full HP for new units

	// Average resistance
	physicalResist := float64(attr.GetPhysicalResistance())
	magicResist := float64(attr.GetMagicDefense())
	avgResistance := (physicalResist + magicResist) / 2.0

	// Dodge multiplier
	dodgeChance := float64(attr.GetDodgeChance()) / 100.0
	dodgeMultiplier := 1.0 / math.Max(1.0-dodgeChance, 0.5)

	defensivePower := (effectiveHealth * dodgeMultiplier) + avgResistance

	// === UTILITY POWER ===
	scalingConstants := GetScalingConstants()

	// Role value
	roleMultiplier := GetRoleMultiplierFromConfig(unit.Role)
	roleValue := roleMultiplier * scalingConstants.RoleScaling

	// Ability value (simplified - assume leader gets average ability value)
	abilityValue := 0.0
	if unit.IsLeader {
		abilityValue = 15.0 // Average of Rally (15.0), Heal (20.0), BattleCry (12.0)
	}

	// Cover value
	coverValue := 0.0
	if unit.CoverValue > 0 {
		coverValue = unit.CoverValue * scalingConstants.CoverScaling * scalingConstants.CoverBeneficiaryMultiplier
	}

	utilityPower := roleValue + abilityValue + coverValue

	// === WEIGHTED SUM ===
	totalPower := (offensivePower * config.OffensiveWeight) +
		(defensivePower * config.DefensiveWeight) +
		(utilityPower * config.UtilityWeight)

	return totalPower
}
