package squads

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// FindUnitByID finds a unit entity by its ID
func FindUnitByID(unitID ecs.EntityID, squadmanager *common.EntityManager) *ecs.Entity {
	for _, result := range squadmanager.World.Query(SquadMemberTag) {
		if result.Entity.GetID() == unitID {
			return result.Entity
		}

	}
	return nil
}

// GetUnitIDsAtGridPosition returns unit IDs occupying a specific grid cell
func GetUnitIDsAtGridPosition(squadID ecs.EntityID, row, col int, squadmanager *common.EntityManager) []ecs.EntityID {
	var unitIDs []ecs.EntityID

	for _, result := range squadmanager.World.Query(SquadMemberTag) {
		unitEntity := result.Entity

		memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)
		if memberData.SquadID != squadID {
			continue
		}

		if !unitEntity.HasComponent(GridPositionComponent) {
			continue
		}

		gridPos := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent)

		// ✅ Check if this unit occupies the queried cell (supports multi-cell units)
		if gridPos.OccupiesCell(row, col) {
			unitID := unitEntity.GetID() // ✅ Native method!
			unitIDs = append(unitIDs, unitID)
		}
	}

	return unitIDs
}

// GetUnitIDsInSquad returns unit IDs belonging to a squad
// ✅ Returns ecs.EntityID (native type), not entity pointers
func GetUnitIDsInSquad(squadID ecs.EntityID, squadmanager *common.EntityManager) []ecs.EntityID {
	var unitIDs []ecs.EntityID

	for _, result := range squadmanager.World.Query(SquadMemberTag) {
		unitEntity := result.Entity
		memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)

		if memberData.SquadID == squadID {
			unitID := unitEntity.GetID() // ✅ Native method!
			unitIDs = append(unitIDs, unitID)
		}
	}

	return unitIDs
}

// GetSquadEntity finds squad entity by squad ID
// ✅ Returns entity pointer directly from query
func GetSquadEntity(squadID ecs.EntityID, squadmanager *common.EntityManager) *ecs.Entity {
	for _, result := range squadmanager.World.Query(SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

		if squadData.SquadID == squadID {
			return squadEntity
		}
	}

	return nil
}

// GetUnitIDsInRow returns alive unit IDs in a row
func GetUnitIDsInRow(squadID ecs.EntityID, row int, squadmanager *common.EntityManager) []ecs.EntityID {
	var unitIDs []ecs.EntityID
	seen := make(map[ecs.EntityID]bool) // ✅ Prevents multi-cell units from being counted multiple times

	for col := 0; col < 3; col++ {
		idsAtPos := GetUnitIDsAtGridPosition(squadID, row, col, squadmanager)
		for _, unitID := range idsAtPos {
			if !seen[unitID] {
				unitEntity := FindUnitByID(unitID, squadmanager)
				if unitEntity == nil {
					continue
				}

				attr := common.GetAttributes(unitEntity)
				if attr.CurrentHealth > 0 {
					unitIDs = append(unitIDs, unitID)
					seen[unitID] = true
				}
			}
		}
	}

	return unitIDs
}

// GetLeaderID finds the leader unit ID of a squad
// ✅ Returns ecs.EntityID (native type), not entity pointer
func GetLeaderID(squadID ecs.EntityID, squadmanager *common.EntityManager) ecs.EntityID {
	for _, result := range squadmanager.World.Query(LeaderTag) {
		leaderEntity := result.Entity
		memberData := common.GetComponentType[*SquadMemberData](leaderEntity, SquadMemberComponent)

		if memberData.SquadID == squadID {
			return leaderEntity.GetID() // ✅ Native method!
		}
	}

	return 0
}

// IsSquadDestroyed checks if all units are dead
func IsSquadDestroyed(squadID ecs.EntityID, squadmanager *common.EntityManager) bool {
	unitIDs := GetUnitIDsInSquad(squadID, squadmanager)

	for _, unitID := range unitIDs {
		unitEntity := FindUnitByID(unitID, squadmanager)
		if unitEntity == nil {
			continue
		}

		attr := common.GetAttributes(unitEntity)
		if attr.CurrentHealth > 0 {
			return false
		}
	}

	return len(unitIDs) > 0
}

// ========================================
// CAPACITY SYSTEM QUERIES
// ========================================

// GetSquadUsedCapacity calculates total capacity consumed by all units in squad
func GetSquadUsedCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) float64 {
	unitIDs := GetUnitIDsInSquad(squadID, squadmanager)
	totalUsed := 0.0

	for _, unitID := range unitIDs {
		unitEntity := FindUnitByID(unitID, squadmanager)
		if unitEntity == nil {
			continue
		}

		attr := common.GetAttributes(unitEntity)
		totalUsed += attr.GetCapacityCost()
	}

	return totalUsed
}

// GetSquadTotalCapacity returns the squad's total capacity based on leader's Leadership
// Returns 0 if squad has no leader (or defaults to 6 if no leader found)
func GetSquadTotalCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) int {
	leaderID := GetLeaderID(squadID, squadmanager)
	if leaderID == 0 {
		// No leader found - return default minimum capacity
		return 6
	}

	leaderEntity := FindUnitByID(leaderID, squadmanager)
	if leaderEntity == nil {
		return 6
	}

	attr := common.GetAttributes(leaderEntity)
	return attr.GetUnitCapacity()
}

// GetSquadRemainingCapacity returns how much capacity is available
func GetSquadRemainingCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) float64 {
	total := float64(GetSquadTotalCapacity(squadID, squadmanager))
	used := GetSquadUsedCapacity(squadID, squadmanager)
	return total - used
}

// CanAddUnitToSquad checks if a unit can be added without exceeding capacity
// Returns true if there's enough remaining capacity, false otherwise
func CanAddUnitToSquad(squadID ecs.EntityID, unitCapacityCost float64, squadmanager *common.EntityManager) bool {
	remaining := GetSquadRemainingCapacity(squadID, squadmanager)
	return remaining >= unitCapacityCost
}

// IsSquadOverCapacity checks if squad currently exceeds its capacity limit
// Used for displaying warnings when leader changes or dies
func IsSquadOverCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) bool {
	return GetSquadRemainingCapacity(squadID, squadmanager) < 0
}

// UpdateSquadCapacity recalculates and updates the cached capacity values in SquadData
// Should be called when: adding/removing units, leader changes, or leader attributes change
func UpdateSquadCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) {
	squadEntity := GetSquadEntity(squadID, squadmanager)
	if squadEntity == nil {
		return
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	squadData.TotalCapacity = GetSquadTotalCapacity(squadID, squadmanager)
	squadData.UsedCapacity = GetSquadUsedCapacity(squadID, squadmanager)
}
