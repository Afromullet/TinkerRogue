// unithelpers.go — shared unit-query primitives used by multiple perk behaviors.
//
// These helpers encapsulate the "iterate a squad's units, filter to alive,
// optionally filter by role, find the min/max of an attribute" pattern that
// would otherwise be copy-pasted into every perk that inspects unit state.
//
// All helpers reuse squadcore.GetAliveUnitAttributes so the alive-check logic
// lives in one place.
package perks

import (
	"math"

	"game_main/common"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"

	"github.com/bytearena/ecs"
)

// FindLowestHPUnit returns the alive unit in squadID with the lowest CurrentHealth,
// or 0 if the squad has no alive units.
func FindLowestHPUnit(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	var lowestID ecs.EntityID
	lowest := math.MaxInt
	for _, uid := range squadcore.GetUnitIDsInSquad(squadID, manager) {
		attr := squadcore.GetAliveUnitAttributes(uid, manager)
		if attr == nil {
			continue
		}
		if attr.CurrentHealth < lowest {
			lowest = attr.CurrentHealth
			lowestID = uid
		}
	}
	return lowestID
}

// FindHighestDexUnitByRole returns the alive unit in squadID with the given role
// and the highest Dexterity, or 0 if no matching unit exists.
func FindHighestDexUnitByRole(squadID ecs.EntityID, role unitdefs.UnitRole, manager *common.EntityManager) ecs.EntityID {
	var bestID ecs.EntityID
	bestDex := 0
	for _, uid := range squadcore.GetUnitIDsInSquad(squadID, manager) {
		attr := squadcore.GetAliveUnitAttributes(uid, manager)
		if attr == nil {
			continue
		}
		entity := manager.FindEntityByID(uid)
		if entity == nil {
			continue
		}
		roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
		if roleData == nil || roleData.Role != role {
			continue
		}
		if attr.Dexterity > bestDex {
			bestDex = attr.Dexterity
			bestID = uid
		}
	}
	return bestID
}

// CountTanksInRow returns the number of alive Tank units in squadID whose
// AnchorRow equals row.
func CountTanksInRow(squadID ecs.EntityID, row int, manager *common.EntityManager) int {
	count := 0
	for _, uid := range squadcore.GetUnitIDsInSquad(squadID, manager) {
		if squadcore.GetAliveUnitAttributes(uid, manager) == nil {
			continue
		}
		entity := manager.FindEntityByID(uid)
		if entity == nil {
			continue
		}
		roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
		if roleData == nil || roleData.Role != unitdefs.RoleTank {
			continue
		}
		gridPos := common.GetComponentType[*squadcore.GridPositionData](entity, squadcore.GridPositionComponent)
		if gridPos != nil && gridPos.AnchorRow == row {
			count++
		}
	}
	return count
}

// HasWoundedUnit returns true if any alive unit in squadID has
// CurrentHealth/MaxHealth strictly below threshold.
func HasWoundedUnit(squadID ecs.EntityID, threshold float64, manager *common.EntityManager) bool {
	for _, uid := range squadcore.GetUnitIDsInSquad(squadID, manager) {
		attr := squadcore.GetAliveUnitAttributes(uid, manager)
		if attr == nil {
			continue
		}
		maxHP := attr.GetMaxHealth()
		if maxHP > 0 && float64(attr.CurrentHealth)/float64(maxHP) < threshold {
			return true
		}
	}
	return false
}

// FindFirstTankInSquad returns the first alive Tank unit in squadID, or 0.
// Iteration order matches squadcore.GetUnitIDsInSquad.
func FindFirstTankInSquad(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	for _, uid := range squadcore.GetUnitIDsInSquad(squadID, manager) {
		if squadcore.GetAliveUnitAttributes(uid, manager) == nil {
			continue
		}
		entity := manager.FindEntityByID(uid)
		if entity == nil {
			continue
		}
		roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
		if roleData != nil && roleData.Role == unitdefs.RoleTank {
			return uid
		}
	}
	return 0
}

// GetUnitsInRow returns all alive units in a specific row of a squad.
func GetUnitsInRow(squadID ecs.EntityID, row int, manager *common.EntityManager) []ecs.EntityID {
	var units []ecs.EntityID
	seen := make(map[ecs.EntityID]bool)

	for col := 0; col <= 2; col++ {
		cellUnits := squadcore.GetUnitIDsAtGridPosition(squadID, row, col, manager)
		for _, unitID := range cellUnits {
			if !seen[unitID] {
				if squadcore.GetAliveUnitAttributes(unitID, manager) != nil {
					units = append(units, unitID)
					seen[unitID] = true
				}
			}
		}
	}

	return units
}
