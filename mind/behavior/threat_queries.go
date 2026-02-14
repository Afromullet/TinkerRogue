package behavior

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// UnitCombatData bundles combat-relevant component data for a unit.
// Provides a single-call way to get all data needed for threat calculations,
// eliminating repetitive nil-checking across multiple component accesses.
// Used by both threat assessment (behavior) and power calculation (evaluation).
type UnitCombatData struct {
	Entity      *ecs.Entity
	EntityID    ecs.EntityID
	Role        squads.UnitRole
	AttackType  squads.AttackType
	AttackRange int
	Attributes  *common.Attributes
	IsLeader    bool
}

// GetUnitCombatData retrieves all combat-relevant data for a unit.
// Returns nil if any required component is missing (entity, role, targetRow, attributes).
// AttackRange defaults to 1 if AttackRangeComponent is missing.
// Pre-calculates BasePower and RoleMultiplier for threat/power calculations.
func GetUnitCombatData(unitID ecs.EntityID, manager *common.EntityManager) *UnitCombatData {
	entity := manager.FindEntityByID(unitID)
	if entity == nil {
		return nil
	}

	roleData := common.GetComponentType[*squads.UnitRoleData](entity, squads.UnitRoleComponent)
	if roleData == nil {
		return nil
	}

	targetRowData := common.GetComponentType[*squads.TargetRowData](entity, squads.TargetRowComponent)
	if targetRowData == nil {
		return nil
	}

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr == nil {
		return nil
	}

	attackRange := 1
	if rangeData := common.GetComponentType[*squads.AttackRangeData](entity, squads.AttackRangeComponent); rangeData != nil {
		attackRange = rangeData.Range
	}

	return &UnitCombatData{
		Entity:      entity,
		EntityID:    unitID,
		Role:        roleData.Role,
		AttackType:  targetRowData.AttackType,
		AttackRange: attackRange,
		Attributes:  attr,
		IsLeader:    entity.HasComponent(squads.LeaderComponent),
	}
}

// AttackTypeFilter represents attack types to filter for
type AttackTypeFilter []squads.AttackType

var (
	// MeleeAttackTypes includes all melee attack types
	MeleeAttackTypes = AttackTypeFilter{squads.AttackTypeMeleeRow, squads.AttackTypeMeleeColumn}

	// RangedAttackTypes includes ranged and magic attack types
	RangedAttackTypes = AttackTypeFilter{squads.AttackTypeRanged, squads.AttackTypeMagic}
)

// Matches returns true if the given attack type is in this filter
func (f AttackTypeFilter) Matches(attackType squads.AttackType) bool {
	for _, t := range f {
		if t == attackType {
			return true
		}
	}
	return false
}

// getUnitsWithAttackTypes returns combat data for all units matching the attack types
func getUnitsWithAttackTypes(
	squadID ecs.EntityID,
	manager *common.EntityManager,
	attackTypes AttackTypeFilter,
) []*UnitCombatData {
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	var matching []*UnitCombatData

	for _, unitID := range unitIDs {
		data := GetUnitCombatData(unitID, manager)
		if data == nil {
			continue
		}
		if attackTypes.Matches(data.AttackType) {
			matching = append(matching, data)
		}
	}

	return matching
}

// hasUnitsWithAttackType checks if squad has any units matching attack types
func hasUnitsWithAttackType(
	squadID ecs.EntityID,
	manager *common.EntityManager,
	attackTypes AttackTypeFilter,
) bool {
	return len(getUnitsWithAttackTypes(squadID, manager, attackTypes)) > 0
}

// getMaxRangeForAttackTypes returns maximum attack range among matching units
func getMaxRangeForAttackTypes(
	squadID ecs.EntityID,
	manager *common.EntityManager,
	attackTypes AttackTypeFilter,
	defaultRange int,
) int {
	maxRange := defaultRange
	for _, data := range getUnitsWithAttackTypes(squadID, manager, attackTypes) {
		if data.AttackRange > maxRange {
			maxRange = data.AttackRange
		}
	}
	return maxRange
}
