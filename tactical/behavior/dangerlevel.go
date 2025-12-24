package behavior

import (
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// Threat calculation constants
const (
	// Movement defaults
	DefaultSquadMovement = 3   // Base movement when no data available
	MaxSpeedSentinel     = 999 // Sentinel for finding minimum speed

	// Role threat multipliers
	ThreatMultiplierDPS     = 1.5 // DPS units deal highest threat
	ThreatMultiplierTank    = 1.2 // Tanks provide high durability threat
	ThreatMultiplierSupport = 1.0 // Support provides utility baseline

	// Leader and composition bonuses
	ThreatLeaderBonus       = 1.3 // Threat multiplier when unit is squad leader
	CompositionPenaltyPure  = 0.8 // 1 attack type (mono-composition)
	CompositionBonusDual    = 1.1 // 2 attack types (good diversity)
	CompositionBonusTriple  = 1.2 // 3 attack types (excellent)
	CompositionBonusQuad    = 1.3 // 4 attack types (optimal, rare)
	CompositionBonusDefault = 1.0 // Fallback

	// Reference target for expected damage calculations
	// Represents a medium-difficulty enemy unit with typical stats
	ReferenceTargetStrength  = 10 // → 12 resistance (10/4 + 5*2)
	ReferenceTargetDexterity = 15 // → 5% dodge (15/3)
	ReferenceTargetArmor     = 5  // → Contributes to resistance
)

var avgCombatStats = &averageCombatStats{}

// Keeps track of Each Factions Danger Level.
type FactionThreatLevelManager struct {
	manager  *common.EntityManager
	cache    *combat.CombatQueryCache
	factions map[ecs.EntityID]*FactionThreatLevel
}

func NewFactionThreatLevelManager(manager *common.EntityManager, cache *combat.CombatQueryCache) *FactionThreatLevelManager {

	calculateAverageAttributes()
	return &FactionThreatLevelManager{
		manager:  manager,
		cache:    cache,
		factions: make(map[ecs.EntityID]*FactionThreatLevel),
	}
}

func (ftlm *FactionThreatLevelManager) AddFaction(factionID ecs.EntityID) {

	if _, exists := ftlm.factions[factionID]; !exists {
		ftlm.factions[factionID] = NewFactionThreatLevel(factionID, ftlm.manager, ftlm.cache)
	}

	ftlm.factions[factionID].UpdateThreatRatings()
}

func (ftlm *FactionThreatLevelManager) UpdateFaction(factionID ecs.EntityID) {
	if faction, exists := ftlm.factions[factionID]; exists {
		faction.UpdateThreatRatings()
	}
}

func (ftlm *FactionThreatLevelManager) UpdateAllFactions() {
	for _, faction := range ftlm.factions {
		faction.UpdateThreatRatings()
	}
}

type FactionThreatLevel struct {
	manager          *common.EntityManager
	cache            *combat.CombatQueryCache
	factionID        ecs.EntityID
	squadDangerLevel map[ecs.EntityID]*SquadThreatLevel //Key is the squad ID. Value is the danger level
}

func NewFactionThreatLevel(factionID ecs.EntityID, manager *common.EntityManager, cache *combat.CombatQueryCache) *FactionThreatLevel {

	squadIDs := combat.GetSquadsForFaction(factionID, manager)

	ftl := &FactionThreatLevel{

		factionID:        factionID,
		squadDangerLevel: make(map[ecs.EntityID]*SquadThreatLevel, len(squadIDs)),
		manager:          manager,
		cache:            cache,
	}

	for _, ID := range squadIDs {
		ftl.squadDangerLevel[ID] = NewSquadThreatLevel(ftl.manager, ftl.cache, ID)
	}

	return ftl
}

func (ftr *FactionThreatLevel) UpdateThreatRatings() {

	squadIDs := combat.GetSquadsForFaction(ftr.factionID, ftr.manager)

	for _, squadID := range squadIDs {
		ftr.squadDangerLevel[squadID].CalculateSquadDangerLevel()
	}

}

func (ftr *FactionThreatLevel) UpdateThreatRatingForSquad(squadID ecs.EntityID) {
	ftr.squadDangerLevel[squadID].CalculateSquadDangerLevel()

}

type SquadThreatLevel struct {
	manager       *common.EntityManager
	cache         *combat.CombatQueryCache
	squadID       ecs.EntityID
	DangerByRange map[int]float64 //Key is the range. Value is the danger level

	// Expected damage output per range using actual damage formulas
	ExpectedDamageByRange map[int]float64 //Key is the range. Value is expected damage
}

func NewSquadThreatLevel(manager *common.EntityManager, cache *combat.CombatQueryCache, squadID ecs.EntityID) *SquadThreatLevel {

	return &SquadThreatLevel{
		manager: manager,
		cache:   cache,
		squadID: squadID,
	}
}

// getSquadMovementRange returns the movement range for a squad in combat.
// Returns MovementRemaining if squad has ActionStateData (in combat).
// Returns squad's base movement speed if not in combat.
// Returns 0 if squad has exhausted movement.
func (stl *SquadThreatLevel) getSquadMovementRange() int {
	// Use cached query instead of World.Query (50-200x faster)
	actionStateEntity := stl.cache.FindActionStateEntity(stl.squadID, stl.manager)
	if actionStateEntity != nil {
		actionState := common.GetComponentType[*combat.ActionStateData](
			actionStateEntity,
			combat.ActionStateComponent,
		)
		if actionState != nil {
			return actionState.MovementRemaining
		}
	}

	// Not in combat - use base squad movement speed
	unitIDs := squads.GetUnitIDsInSquad(stl.squadID, stl.manager)
	if len(unitIDs) == 0 {
		return DefaultSquadMovement
	}

	// Find slowest unit speed (squad moves as one)
	minSpeed := MaxSpeedSentinel
	for _, unitID := range unitIDs {
		entity := common.FindEntityByID(stl.manager, unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil {
			continue
		}

		speed := attr.GetMovementSpeed()
		if speed < minSpeed {
			minSpeed = speed
		}
	}

	if minSpeed == MaxSpeedSentinel {
		return DefaultSquadMovement
	}

	return minSpeed
}

// unitData represents attack data for a unit in the squad
type unitData struct {
	entity         *ecs.Entity
	attackRange    int
	power          float64
	roleMultiplier float64
	isLeader       bool
	attackType     squads.AttackType
}

// Returns a map of squadID -> danger levels by range.
func (stl *SquadThreatLevel) CalculateSquadDangerLevel() {
	unitIDs := squads.GetUnitIDsInSquad(stl.squadID, stl.manager)

	// Collect all units with their attack data

	var units []unitData
	attackTypeCount := make(map[squads.AttackType]int)

	for _, unitID := range unitIDs {
		unitEntity := common.FindEntityByID(stl.manager, unitID)
		if unitEntity == nil {
			continue
		}

		// Get unit role
		roleData := common.GetComponentType[*squads.UnitRoleData](unitEntity, squads.UnitRoleComponent)
		if roleData == nil {
			continue
		}

		// Get unit target row (attack type)
		targetRowData := common.GetComponentType[*squads.TargetRowData](unitEntity, squads.TargetRowComponent)
		if targetRowData == nil {
			continue
		}

		// Get attack range
		attackRangeData := common.GetComponentType[*squads.AttackRangeData](unitEntity, squads.AttackRangeComponent)
		attackRange := 1
		if attackRangeData != nil {
			attackRange = attackRangeData.Range
		}

		// Get unit attributes
		attr := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
		if attr == nil {
			continue
		}

		// Check if unit is leader by checking for LeaderComponent
		isLeader := unitEntity.HasComponent(squads.LeaderComponent)

		// Calculate base power from weapon and dexterity
		// Weapon damage is primary, dexterity improves hit rate
		basePower := float64(attr.Weapon + attr.Dexterity/2)

		// Get role multiplier
		roleMultiplier := stl.getRoleMultiplier(roleData.Role)

		// Track attack type for composition bonus
		attackTypeCount[targetRowData.AttackType]++

		units = append(units, unitData{
			entity:         unitEntity,
			attackRange:    attackRange,
			power:          basePower,
			roleMultiplier: roleMultiplier,
			isLeader:       isLeader,
			attackType:     targetRowData.AttackType,
		})
	}

	// Get squad movement range (combat: MovementRemaining, default: base speed)
	movementRange := stl.getSquadMovementRange()

	// Find maximum threat range (movement + attack)
	maxThreatRange := 0
	for _, ud := range units {
		threatRange := movementRange + ud.attackRange
		if threatRange > maxThreatRange {
			maxThreatRange = threatRange
		}
	}

	// Calculate danger at each range from 1 to maxThreatRange
	// Units threaten a range if movement + attack >= currentRange
	dangerByRange := make(map[int]float64, maxThreatRange)

	for currentRange := 1; currentRange <= maxThreatRange; currentRange++ {
		var rangeDanger float64 = 0

		// Sum danger from units that can threaten this range
		for _, ud := range units {
			// Unit threatens currentRange if movement + attack >= currentRange
			effectiveThreatRange := movementRange + ud.attackRange

			if effectiveThreatRange >= currentRange {
				// Apply leader bonus
				leaderBonus := 1.0
				if ud.isLeader {
					leaderBonus = ThreatLeaderBonus
				}

				// Calculate unit danger contribution at this range
				unitDanger := ud.power * ud.roleMultiplier * leaderBonus
				rangeDanger += unitDanger
			}
		}

		dangerByRange[currentRange] = rangeDanger
	}

	// Apply composition bonus to each range
	compositionBonus := stl.calculateCompositionBonus(attackTypeCount)
	for range_, danger := range dangerByRange {
		dangerByRange[range_] = danger * compositionBonus
	}

	stl.DangerByRange = dangerByRange

	// Calculate expected damage by range using actual damage formulas
	stl.ExpectedDamageByRange = stl.calculateSquadExpectedDamageByRange(
		units,
		movementRange,
		maxThreatRange,
	)

}

// getRoleMultiplier returns a damage multiplier based on unit role
func (stl *SquadThreatLevel) getRoleMultiplier(role squads.UnitRole) float64 {
	switch role {
	case squads.RoleDPS:
		return ThreatMultiplierDPS
	case squads.RoleTank:
		return ThreatMultiplierTank
	case squads.RoleSupport:
		return ThreatMultiplierSupport
	default:
		return ThreatMultiplierSupport
	}
}

// calculateCompositionBonus returns a bonus multiplier based on attack type diversity
// Squads with diverse attack types (melee + ranged + magic) are more effective
func (stl *SquadThreatLevel) calculateCompositionBonus(attackTypeCount map[squads.AttackType]int) float64 {
	// Bonus for having diverse attack types
	// Pure squads (1 type) are less effective
	// Balanced squads (2-3 types) are stronger
	// Highly diverse (4 types) is ideal but rare
	uniqueTypes := len(attackTypeCount)
	switch uniqueTypes {
	case 1:
		return CompositionPenaltyPure
	case 2:
		return CompositionBonusDual
	case 3:
		return CompositionBonusTriple
	case 4:
		return CompositionBonusQuad
	default:
		return CompositionBonusDefault
	}
}

// calculateExpectedDamageForUnit computes expected damage for one attacker unit.
// Uses actual damage formulas from squadcombat.go with probability weighting.
//
// Parameters:
//   - attackerAttr: Attacker's attributes
//   - defenderAttr: Reference target attributes
//
// Returns: Expected damage (float64) accounting for hit/dodge/crit probabilities
func (stl *SquadThreatLevel) calculateExpectedDamageForUnit(
	attackerAttr *common.Attributes,
	defenderAttr *common.Attributes,
) float64 {

	// 1. Base damage (same as GetPhysicalDamage)
	baseDamage := float64(attackerAttr.GetPhysicalDamage())

	// 2. Hit probability
	hitRate := float64(attackerAttr.GetHitRate()) / 100.0
	if hitRate > 1.0 {
		hitRate = 1.0
	}

	dodgeChance := float64(defenderAttr.GetDodgeChance()) / 100.0
	pHit := hitRate * (1.0 - dodgeChance)

	// 3. Expected crit multiplier
	// Expected value: P(normal) × 1.0 + P(crit) × 1.5
	critChance := float64(attackerAttr.GetCritChance()) / 100.0
	expectedCritMult := 1.0 + (critChance * 0.5)

	// 4. Apply resistance (subtraction, not percentage)
	resistance := float64(defenderAttr.GetPhysicalResistance())
	damageAfterResist := baseDamage - resistance
	if damageAfterResist < 1.0 {
		damageAfterResist = 1.0 // Minimum damage
	}

	// 5. Apply cover (initially 0, no cover assumed - conservative)
	coverReduction := 0.0
	coverMultiplier := 1.0 - coverReduction

	// 6. Combine all factors
	expectedDamage := damageAfterResist * pHit * expectedCritMult * coverMultiplier

	return expectedDamage
}

// calculateSquadExpectedDamageByRange computes expected damage output for the squad
// at each range. Similar to DangerByRange but uses actual damage formulas.
//
// This reuses unit data from CalculateSquadDangerLevel to avoid redundant queries.
func (stl *SquadThreatLevel) calculateSquadExpectedDamageByRange(
	units []unitData,
	movementRange int,
	maxThreatRange int,
) map[int]float64 {

	// Create reference target once

	// Calculate expected damage at each range
	expectedDamageByRange := make(map[int]float64, maxThreatRange)

	for currentRange := 1; currentRange <= maxThreatRange; currentRange++ {
		var rangeDamage float64 = 0.0

		// Sum expected damage from units that can threaten this range
		for _, ud := range units {
			effectiveThreatRange := movementRange + ud.attackRange

			if effectiveThreatRange >= currentRange {
				// Get attacker attributes
				attr := common.GetComponentType[*common.Attributes](
					ud.entity,
					common.AttributeComponent,
				)
				if attr == nil {
					continue
				}

				// Calculate expected damage for this unit
				unitExpectedDamage := stl.calculateExpectedDamageForUnit(attr, &avgCombatStats.avgAttributes)

				// Apply leader bonus (same as DangerByRange)
				if ud.isLeader {
					unitExpectedDamage *= ThreatLeaderBonus
				}

				rangeDamage += unitExpectedDamage
			}
		}

		expectedDamageByRange[currentRange] = rangeDamage
	}

	// Note: We do NOT apply composition bonus to damage (only to threat)
	// Damage is raw output; composition affects tactical value

	return expectedDamageByRange
}

// Used for predicting damage for danger level calculations
// Average attributes of all creatures

type averageCombatStats struct {
	avgAttributes     common.Attributes
	avgPhyslDamage    int
	avgPhysResistance int
	avgMaxHealth      int
	avgHitRate        int
	avgCritChance     int
	avgDodgeChance    int
	avgMagicDamage    int
	avgMMagicDefense  int
	avgHealingAmount  int
}

func calculateAverageAttributes() {

	num_units := len(squads.Units)

	for _, unit := range squads.Units {

		avgCombatStats.avgAttributes.Strength += unit.Attributes.Strength
		avgCombatStats.avgAttributes.Dexterity += unit.Attributes.Dexterity
		avgCombatStats.avgAttributes.Magic += unit.Attributes.Magic
		avgCombatStats.avgAttributes.Leadership += unit.Attributes.Leadership
		avgCombatStats.avgAttributes.Armor += unit.Attributes.Armor
		avgCombatStats.avgAttributes.Weapon += unit.Attributes.Weapon
		avgCombatStats.avgAttributes.MovementSpeed += unit.Attributes.MovementSpeed
		avgCombatStats.avgAttributes.AttackRange += unit.Attributes.AttackRange
		avgCombatStats.avgAttributes.CurrentHealth += unit.Attributes.CurrentHealth
		avgCombatStats.avgMaxHealth += unit.Attributes.GetMaxHealth()
		avgCombatStats.avgHitRate += unit.Attributes.GetHitRate()
		avgCombatStats.avgCritChance += unit.Attributes.GetCritChance()
		avgCombatStats.avgDodgeChance += unit.Attributes.GetDodgeChance()
		avgCombatStats.avgMagicDamage += unit.Attributes.GetMagicDamage()
		avgCombatStats.avgMMagicDefense += unit.Attributes.GetMagicDefense()
		avgCombatStats.avgHealingAmount += unit.Attributes.GetHealingAmount()

	}

	avgCombatStats.avgAttributes.Strength /= num_units
	avgCombatStats.avgAttributes.Dexterity /= num_units
	avgCombatStats.avgAttributes.Magic /= num_units
	avgCombatStats.avgAttributes.Leadership /= num_units
	avgCombatStats.avgAttributes.Armor /= num_units
	avgCombatStats.avgAttributes.Weapon /= num_units
	avgCombatStats.avgAttributes.MovementSpeed /= num_units
	avgCombatStats.avgAttributes.AttackRange /= num_units
	avgCombatStats.avgAttributes.CurrentHealth /= num_units
	avgCombatStats.avgMaxHealth /= num_units
	avgCombatStats.avgHitRate /= num_units
	avgCombatStats.avgCritChance /= num_units
	avgCombatStats.avgDodgeChance /= num_units
	avgCombatStats.avgMagicDamage /= num_units
	avgCombatStats.avgMMagicDefense /= num_units
	avgCombatStats.avgHealingAmount /= num_units

}
