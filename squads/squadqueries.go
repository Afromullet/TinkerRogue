package squads

import (
	"game_main/common"
	"game_main/coords"
	"math"

	"github.com/bytearena/ecs"
)

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
	// Note: This uses a component field match (SquadData.SquadID), not a direct entity ID match
	// So it cannot use the generic FindEntityByIDWithTag helper. Keeping specialized implementation.
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
				attr := common.GetAttributesByIDWithTag(squadmanager, unitID, SquadMemberTag)
				if attr == nil {
					continue
				}

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
// O(1) operation - uses cached IsDestroyed flag from SquadData
// The flag is maintained by UpdateSquadDestroyedStatus() when units die or are removed
func IsSquadDestroyed(squadID ecs.EntityID, squadmanager *common.EntityManager) bool {
	squadEntity := GetSquadEntity(squadID, squadmanager)
	if squadEntity == nil {
		return true // Squad not found, consider destroyed
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	if squadData == nil {
		return true // Invalid squad data, consider destroyed
	}

	return squadData.IsDestroyed
}

// ========================================
// CAPACITY SYSTEM QUERIES
// ========================================

// GetSquadUsedCapacity calculates total capacity consumed by all units in squad
func GetSquadUsedCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) float64 {
	unitIDs := GetUnitIDsInSquad(squadID, squadmanager)
	totalUsed := 0.0

	for _, unitID := range unitIDs {
		attr := common.GetAttributesByIDWithTag(squadmanager, unitID, SquadMemberTag)
		if attr == nil {
			continue
		}

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

	attr := common.GetAttributesByIDWithTag(squadmanager, leaderID, SquadMemberTag)
	if attr == nil {
		return 6
	}

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

// ========================================
// RANGE AND DISTANCE QUERIES
// ========================================

// GetSquadDistance calculates the Chebyshev distance (max(|dx|, |dy|)) between two squads
// based on their world positions. This matches the tactical grid movement system where
// diagonal moves cost 1. Returns -1 if either squad is not found or missing a position component.
func GetSquadDistance(squad1ID, squad2ID ecs.EntityID, squadmanager *common.EntityManager) int {
	squad1Entity := GetSquadEntity(squad1ID, squadmanager)
	squad2Entity := GetSquadEntity(squad2ID, squadmanager)

	if squad1Entity == nil || squad2Entity == nil {
		return -1
	}

	// Check if both squads have position components
	if !squad1Entity.HasComponent(common.PositionComponent) || !squad2Entity.HasComponent(common.PositionComponent) {
		return -1
	}

	pos1 := common.GetComponentType[*coords.LogicalPosition](squad1Entity, common.PositionComponent)
	pos2 := common.GetComponentType[*coords.LogicalPosition](squad2Entity, common.PositionComponent)

	// Use Chebyshev distance consistently with combat system
	return pos1.ChebyshevDistance(pos2)
}

// GetSquadMovementSpeed returns the squad's movement speed (minimum of all alive units)
// The squad moves at the speed of its slowest member
// Returns 0 if squad has no alive units or units have no movement speed component
func GetSquadMovementSpeed(squadID ecs.EntityID, squadmanager *common.EntityManager) int {
	unitIDs := GetUnitIDsInSquad(squadID, squadmanager)

	minSpeed := math.MaxInt32
	foundValidUnit := false

	for _, unitID := range unitIDs {
		// Only count alive units
		attr := common.GetAttributesByIDWithTag(squadmanager, unitID, SquadMemberTag)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}

		// Check if unit has movement speed component
		if !squadmanager.HasComponentByIDWithTag(unitID, SquadMemberTag, MovementSpeedComponent) {
			continue
		}

		speedData := common.GetComponentTypeByID[*MovementSpeedData](squadmanager, unitID, MovementSpeedComponent)
		if speedData != nil && speedData.Speed < minSpeed {
			minSpeed = speedData.Speed
			foundValidUnit = true
		}
	}

	// Return 0 if no valid units found
	if !foundValidUnit {
		return 0
	}

	return minSpeed
}

// GetSquadName returns the squad name
// Returns "Unknown Squad" if squad not found
func GetSquadName(squadID ecs.EntityID, squadmanager *common.EntityManager) string {
	squadEntity := GetSquadEntity(squadID, squadmanager)
	if squadEntity == nil {
		return "Unknown Squad"
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	return squadData.Name
}

// FindAllSquads returns all squad entity IDs in the game
// Uses efficient ECS query pattern with SquadTag
func FindAllSquads(squadmanager *common.EntityManager) []ecs.EntityID {
	allSquads := make([]ecs.EntityID, 0)

	for _, result := range squadmanager.World.Query(SquadTag) {
		squadData := common.GetComponentType[*SquadData](result.Entity, SquadComponent)
		allSquads = append(allSquads, squadData.SquadID)
	}

	return allSquads
}

// UpdateSquadDestroyedStatus updates the cached IsDestroyed flag for a squad
// This should be called whenever unit health changes or units are added/removed
// O(n) where n = number of units in squad, but only called when needed
func UpdateSquadDestroyedStatus(squadID ecs.EntityID, manager *common.EntityManager) {
	squadEntity := GetSquadEntity(squadID, manager)
	if squadEntity == nil {
		return
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	if squadData == nil {
		return
	}

	// Get all units in the squad
	unitIDs := GetUnitIDsInSquad(squadID, manager)
	if len(unitIDs) == 0 {
		// No units means destroyed
		squadData.IsDestroyed = true
		return
	}

	// Check if any unit is alive
	hasAliveUnit := false
	for _, unitID := range unitIDs {
		attr := common.GetAttributesByIDWithTag(manager, unitID, SquadMemberTag)
		if attr != nil && attr.CurrentHealth > 0 {
			hasAliveUnit = true
			break
		}
	}

	squadData.IsDestroyed = !hasAliveUnit
}

// GetUnitIdentity extracts display info for a unit (query-based, no caching)
func GetUnitIdentity(unitID ecs.EntityID, manager *common.EntityManager) UnitIdentity {
	// Query for name
	name := common.GetComponentTypeByID[*common.Name](manager, unitID, common.NameComponent)
	nameStr := "Unknown"
	if name != nil {
		nameStr = name.NameStr
	}

	// Query for grid position
	gridPos := common.GetComponentTypeByID[*GridPositionData](manager, unitID, GridPositionComponent)
	row, col := 0, 0
	if gridPos != nil {
		row, col = gridPos.AnchorRow, gridPos.AnchorCol
	}

	// Query for health
	attr := common.GetAttributesByIDWithTag(manager, unitID, SquadMemberTag)
	currentHP, maxHP := 0, 0
	if attr != nil {
		currentHP, maxHP = attr.CurrentHealth, attr.MaxHealth
	}

	return UnitIdentity{
		ID:        unitID,
		Name:      nameStr,
		GridRow:   row,
		GridCol:   col,
		CurrentHP: currentHP,
		MaxHP:     maxHP,
	}
}
