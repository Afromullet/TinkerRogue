package squads

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// FindUnitByID finds a unit entity by its ID
func FindUnitByID(unitID ecs.EntityID, squadmanager *SquadECSManager) *ecs.Entity {
	for _, result := range squadmanager.Manager.Query(SquadMemberTag) {
		if result.Entity.GetID() == unitID {
			return result.Entity
		}

	}
	return nil
}

// GetUnitIDsAtGridPosition returns unit IDs occupying a specific grid cell
func GetUnitIDsAtGridPosition(squadID ecs.EntityID, row, col int, squadmanager *SquadECSManager) []ecs.EntityID {
	var unitIDs []ecs.EntityID

	for _, result := range squadmanager.Manager.Query(SquadMemberTag) {
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
func GetUnitIDsInSquad(squadID ecs.EntityID, squadmanager *SquadECSManager) []ecs.EntityID {
	var unitIDs []ecs.EntityID

	for _, result := range squadmanager.Manager.Query(SquadMemberTag) {
		unitEntity := result.Entity
		memberData := common.GetComponentType[SquadMemberData](unitEntity, SquadMemberComponent)

		if memberData.SquadID == squadID {
			unitID := unitEntity.GetID() // ✅ Native method!
			unitIDs = append(unitIDs, unitID)
		}
	}

	return unitIDs
}

// GetSquadEntity finds squad entity by squad ID
// ✅ Returns entity pointer directly from query
func GetSquadEntity(squadID ecs.EntityID, squadmanager *SquadECSManager) *ecs.Entity {
	for _, result := range squadmanager.Manager.Query(SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

		if squadData.SquadID == squadID {
			return squadEntity
		}
	}

	return nil
}

// GetUnitIDsInRow returns alive unit IDs in a row
func GetUnitIDsInRow(squadID ecs.EntityID, row int, squadmanager *SquadECSManager) []ecs.EntityID {
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
func GetLeaderID(squadID ecs.EntityID, squadmanager *SquadECSManager) ecs.EntityID {
	for _, result := range squadmanager.Manager.Query(LeaderTag) {
		leaderEntity := result.Entity
		memberData := common.GetComponentType[*SquadMemberData](leaderEntity, SquadMemberComponent)

		if memberData.SquadID == squadID {
			return leaderEntity.GetID() // ✅ Native method!
		}
	}

	return 0
}

// IsSquadDestroyed checks if all units are dead
func IsSquadDestroyed(squadID ecs.EntityID, squadmanager *SquadECSManager) bool {
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
