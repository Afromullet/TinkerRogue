package squads

import (
	"game_main/common"
	"game_main/world/coords"
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

		// Check if this unit occupies the queried cell (supports multi-cell units)
		if gridPos.OccupiesCell(row, col) {
			unitID := unitEntity.GetID() //  Native method!
			unitIDs = append(unitIDs, unitID)
		}
	}

	return unitIDs
}

// GetUnitIDsInSquad returns unit IDs belonging to a squad
// Uses package-level ecs.View for zero-allocation reads instead of World.Query.
// Returns ecs.EntityID (native type), not entity pointers
func GetUnitIDsInSquad(squadID ecs.EntityID, squadmanager *common.EntityManager) []ecs.EntityID {
	var unitIDs []ecs.EntityID

	for _, result := range squadMemberView.Get() {
		unitEntity := result.Entity
		memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)

		if memberData.SquadID == squadID {
			unitID := unitEntity.GetID()
			unitIDs = append(unitIDs, unitID)
		}
	}

	return unitIDs
}

// GetSquadEntity finds squad entity by squad ID
// NOTE: This is the non-cached version (O(n)). Prefer using SquadQueryCache.GetSquadEntity() when available for better performance.
// Returns entity pointer directly from query
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

// GetLeaderID finds the leader unit ID of a squad
// NOTE: This is the non-cached version (O(n)). Prefer using SquadQueryCache.GetLeaderID() when available for better performance.
// Returns ecs.EntityID (native type), not entity pointer
func GetLeaderID(squadID ecs.EntityID, squadmanager *common.EntityManager) ecs.EntityID {
	for _, result := range squadmanager.World.Query(LeaderTag) {
		leaderEntity := result.Entity
		memberData := common.GetComponentType[*SquadMemberData](leaderEntity, SquadMemberComponent)

		if memberData.SquadID == squadID {
			return leaderEntity.GetID() //  Native method!
		}
	}

	return 0
}

// IsSquadDestroyed checks if all units are dead by iterating alive units.
// Returns true if squad not found, has no units, or all units are dead.
func IsSquadDestroyed(squadID ecs.EntityID, squadmanager *common.EntityManager) bool {
	unitIDs := GetUnitIDsInSquad(squadID, squadmanager)
	if len(unitIDs) == 0 {
		return true
	}

	for _, unitID := range unitIDs {
		if getAliveUnitAttributes(unitID, squadmanager) != nil {
			return false
		}
	}

	return true
}

// WouldSquadSurvive checks if a squad would have any survivors after predicted damage is applied
// Used during combat calculation to determine if counterattacks should happen
func WouldSquadSurvive(squadID ecs.EntityID, predictedDamage map[ecs.EntityID]int, manager *common.EntityManager) bool {
	unitIDs := GetUnitIDsInSquad(squadID, manager)

	for _, unitID := range unitIDs {
		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil {
			continue
		}

		// Check if unit would survive after predicted damage
		damageToThisUnit := predictedDamage[unitID]
		if attr.CurrentHealth-damageToThisUnit > 0 {
			return true // Found at least one unit that would survive
		}
	}

	return false // No survivors
}

// ========================================
// HELPER FUNCTIONS
// ========================================

// getAliveUnitAttributes returns the attributes component for a unit if it's alive
// Returns nil if entity not found, no attributes, or unit is dead
func getAliveUnitAttributes(unitID ecs.EntityID, manager *common.EntityManager) *common.Attributes {
	entity := manager.FindEntityByID(unitID)
	if entity == nil {
		return nil
	}

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr == nil || attr.CurrentHealth <= 0 {
		return nil
	}

	return attr
}

// getUnitAttributes returns the attributes component for a unit (dead or alive)
// Returns nil if entity not found or no attributes
func getUnitAttributes(unitID ecs.EntityID, manager *common.EntityManager) *common.Attributes {
	entity := manager.FindEntityByID(unitID)
	if entity == nil {
		return nil
	}

	return common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
}

// ========================================
// CAPACITY SYSTEM QUERIES
// ========================================

// GetSquadUsedCapacity calculates total capacity consumed by all units in squad
func GetSquadUsedCapacity(squadID ecs.EntityID, squadmanager *common.EntityManager) float64 {
	unitIDs := GetUnitIDsInSquad(squadID, squadmanager)
	totalUsed := 0.0

	for _, unitID := range unitIDs {
		attr := getUnitAttributes(unitID, squadmanager)
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
		return DefaultSquadCapacity
	}

	attr := getUnitAttributes(leaderID, squadmanager)
	if attr == nil {
		return DefaultSquadCapacity
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
		// Use helper to get alive unit attributes
		if getAliveUnitAttributes(unitID, squadmanager) == nil {
			continue
		}

		// Get entity for movement speed component check
		entity := squadmanager.FindEntityByID(unitID)
		if entity == nil || !entity.HasComponent(MovementSpeedComponent) {
			continue
		}

		speedData := common.GetComponentType[*MovementSpeedData](entity, MovementSpeedComponent)
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
// NOTE: This is the non-cached version (O(n)). Prefer using SquadQueryCache.GetSquadName() when available for better performance.
// Returns "Unknown Squad" if squad not found
func GetSquadName(squadID ecs.EntityID, squadmanager *common.EntityManager) string {
	squadEntity := GetSquadEntity(squadID, squadmanager)
	if squadEntity == nil {
		return "Unknown Squad"
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	return squadData.Name
}

// ========================================
// ROLE AND HEALTH QUERIES
// ========================================

// GetSquadPrimaryRole determines the dominant role based on unit composition
// Returns the role with the highest count of units
// Defaults to RoleDPS if no units found or no role data available
func GetSquadPrimaryRole(squadID ecs.EntityID, manager *common.EntityManager) UnitRole {
	unitIDs := GetUnitIDsInSquad(squadID, manager)

	roleCounts := map[UnitRole]int{
		RoleTank:    0,
		RoleDPS:     0,
		RoleSupport: 0,
	}

	for _, unitID := range unitIDs {
		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		roleData := common.GetComponentType[*UnitRoleData](entity, UnitRoleComponent)
		if roleData != nil {
			roleCounts[roleData.Role]++
		}
	}

	// Return role with highest count
	maxRole := RoleDPS // Default
	maxCount := 0
	for role, count := range roleCounts {
		if count > maxCount {
			maxCount = count
			maxRole = role
		}
	}

	return maxRole
}

// GetSquadHealthPercent returns the average HP percentage of alive units in squad (0.0-1.0)
// Returns 0.0 if no alive units found
func GetSquadHealthPercent(squadID ecs.EntityID, manager *common.EntityManager) float64 {
	unitIDs := GetUnitIDsInSquad(squadID, manager)
	if len(unitIDs) == 0 {
		return 0.0
	}

	totalHP := 0.0
	aliveCount := 0

	for _, unitID := range unitIDs {
		attr := getAliveUnitAttributes(unitID, manager)
		if attr == nil {
			continue
		}

		maxHP := attr.GetMaxHealth()
		if maxHP > 0 {
			totalHP += float64(attr.CurrentHealth) / float64(maxHP)
			aliveCount++
		}
	}

	if aliveCount == 0 {
		return 0.0
	}

	return totalHP / float64(aliveCount)
}
