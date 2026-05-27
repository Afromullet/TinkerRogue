// unithelpers.go — shared unit-query primitives used by multiple perk behaviors.
//
// These helpers encapsulate the "iterate a squad's units, filter to alive,
// optionally filter by role, find the min/max of an attribute" pattern that
// would otherwise be copy-pasted into every perk that inspects unit state.
//
// forEachAliveUnit and forEachAliveUnitWithRole are the iteration primitives;
// the public Find* / Count* / Has* functions are reducers built on top.
package perks

import (
	"math"

	"game_main/core/common"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"

	"github.com/bytearena/ecs"
)

// forEachAliveUnit iterates over alive units in squadID, calling fn for each.
// fn returns false to stop iteration early.
func forEachAliveUnit(squadID ecs.EntityID, manager *common.EntityManager,
	fn func(unitID ecs.EntityID, attr *common.Attributes) bool) {
	for _, uid := range squadcore.GetUnitIDsInSquad(squadID, manager) {
		attr := squadcore.GetAliveUnitAttributes(uid, manager)
		if attr == nil {
			continue
		}
		if !fn(uid, attr) {
			return
		}
	}
}

// forEachAliveUnitWithRole iterates over alive units in squadID matching the
// given role, calling fn for each. fn returns false to stop iteration early.
func forEachAliveUnitWithRole(squadID ecs.EntityID, role unitdefs.UnitRole, manager *common.EntityManager,
	fn func(unitID ecs.EntityID, attr *common.Attributes) bool) {
	forEachAliveUnit(squadID, manager, func(uid ecs.EntityID, attr *common.Attributes) bool {
		entity := manager.FindEntityByID(uid)
		if entity == nil {
			return true
		}
		roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
		if roleData == nil || roleData.Role != role {
			return true
		}
		return fn(uid, attr)
	})
}

// FindLowestHPUnit returns the alive unit in squadID with the lowest CurrentHealth,
// or 0 if the squad has no alive units.
func FindLowestHPUnit(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	var lowestID ecs.EntityID
	lowest := math.MaxInt
	forEachAliveUnit(squadID, manager, func(uid ecs.EntityID, attr *common.Attributes) bool {
		if attr.CurrentHealth < lowest {
			lowest = attr.CurrentHealth
			lowestID = uid
		}
		return true
	})
	return lowestID
}

// FindHighestDexUnitByRole returns the alive unit in squadID with the given role
// and the highest Dexterity, or 0 if no matching unit exists.
func FindHighestDexUnitByRole(squadID ecs.EntityID, role unitdefs.UnitRole, manager *common.EntityManager) ecs.EntityID {
	var bestID ecs.EntityID
	bestDex := 0
	forEachAliveUnitWithRole(squadID, role, manager, func(uid ecs.EntityID, attr *common.Attributes) bool {
		if attr.Dexterity > bestDex {
			bestDex = attr.Dexterity
			bestID = uid
		}
		return true
	})
	return bestID
}

// CountTanksInRow returns the number of alive Tank units in squadID whose
// AnchorRow equals row.
func CountTanksInRow(squadID ecs.EntityID, row int, manager *common.EntityManager) int {
	count := 0
	forEachAliveUnitWithRole(squadID, unitdefs.RoleTank, manager, func(uid ecs.EntityID, _ *common.Attributes) bool {
		entity := manager.FindEntityByID(uid)
		if entity == nil {
			return true
		}
		gridPos := common.GetComponentType[*squadcore.GridPositionData](entity, squadcore.GridPositionComponent)
		if gridPos != nil && gridPos.AnchorRow == row {
			count++
		}
		return true
	})
	return count
}

// HasWoundedUnit returns true if any alive unit in squadID has
// CurrentHealth/MaxHealth strictly below threshold.
func HasWoundedUnit(squadID ecs.EntityID, threshold float64, manager *common.EntityManager) bool {
	wounded := false
	forEachAliveUnit(squadID, manager, func(_ ecs.EntityID, attr *common.Attributes) bool {
		maxHP := attr.GetMaxHealth()
		if maxHP > 0 && float64(attr.CurrentHealth)/float64(maxHP) < threshold {
			wounded = true
			return false
		}
		return true
	})
	return wounded
}

// FindFirstTankInSquad returns the first alive Tank unit in squadID, or 0.
// Iteration order matches squadcore.GetUnitIDsInSquad.
func FindFirstTankInSquad(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	var firstID ecs.EntityID
	forEachAliveUnitWithRole(squadID, unitdefs.RoleTank, manager, func(uid ecs.EntityID, _ *common.Attributes) bool {
		firstID = uid
		return false
	})
	return firstID
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
