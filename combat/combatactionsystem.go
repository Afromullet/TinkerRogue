package combat

import (
	"fmt"
	"game_main/common"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

type CombatActionSystem struct {
	manager *common.EntityManager
}

func NewCombatActionSystem(manager *common.EntityManager) *CombatActionSystem {
	return &CombatActionSystem{
		manager: manager,
	}
}

func (cas *CombatActionSystem) ExecuteAttackAction(attackerID, defenderID ecs.EntityID) error {

	attackerPos, err := getSquadMapPosition(attackerID, cas.manager)
	if err != nil {
		return fmt.Errorf("cannot find attacker position: %w", err)
	}

	defenderPos, err := getSquadMapPosition(defenderID, cas.manager)
	if err != nil {
		return fmt.Errorf("cannot find defender position: %w", err)
	}

	distance := attackerPos.ChebyshevDistance(&defenderPos)

	maxRange := cas.GetSquadAttackRange(attackerID)

	if distance > maxRange {
		return fmt.Errorf("target out of range: %d tiles away, max range %d", distance, maxRange)
	}

	if !canSquadAct(attackerID, cas.manager) {
		return fmt.Errorf("squad has already acted this turn")
	}

	//Filter units by range (partial squad attacks)
	attackingUnits := cas.GetAttackingUnits(attackerID, defenderID)

	// Temporarily disable out-of-range units
	allUnits := squads.GetUnitIDsInSquad(attackerID, cas.manager)
	disabledUnits := []ecs.EntityID{}

	for _, unitID := range allUnits {
		if !containsEntity(attackingUnits, unitID) {
			unit := squads.FindUnitByID(unitID, cas.manager)
			attr := common.GetAttributes(unit)
			if attr.CanAct {
				attr.CanAct = false
				disabledUnits = append(disabledUnits, unitID)
			}
		}
	}

	// Execute attack (only CanAct=true units participate)
	result := squads.ExecuteSquadAttack(attackerID, defenderID, cas.manager)

	// Re-enable disabled units
	for _, unitID := range disabledUnits {
		unit := squads.FindUnitByID(unitID, cas.manager)
		attr := common.GetAttributes(unit)
		attr.CanAct = true
	}

	markSquadAsActed(attackerID, cas.manager)

	if squads.IsSquadDestroyed(defenderID, cas.manager) {
		removeSquadFromMap(defenderID, cas.manager)
	}

	logCombatResult(result)

	return nil
}

func (cas *CombatActionSystem) GetSquadAttackRange(squadID ecs.EntityID) int {
	unitIDs := squads.GetUnitIDsInSquad(squadID, cas.manager)

	maxRange := 1 // Default melee
	for _, unitID := range unitIDs {
		unit := squads.FindUnitByID(unitID, cas.manager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		unitRange := attr.GetAttackRange()

		if unitRange > maxRange {
			maxRange = unitRange
		}
	}

	return maxRange // Squad can attack at max range of any unit
}

func (cas *CombatActionSystem) GetAttackingUnits(squadID, targetID ecs.EntityID) []ecs.EntityID {
	// Get positions
	attackerPos, _ := getSquadMapPosition(squadID, cas.manager)
	defenderPos, _ := getSquadMapPosition(targetID, cas.manager)

	distance := attackerPos.ChebyshevDistance(&defenderPos)

	allUnits := squads.GetUnitIDsInSquad(squadID, cas.manager)

	var attackingUnits []ecs.EntityID

	for _, unitID := range allUnits {
		unit := squads.FindUnitByID(unitID, cas.manager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		unitRange := attr.GetAttackRange()

		// Check if this unit can reach the target
		if distance <= unitRange {
			attackingUnits = append(attackingUnits, unitID)
		}
	}

	return attackingUnits
}

func (cas *CombatActionSystem) GetSquadsInRange(squadID ecs.EntityID) []ecs.EntityID {
	var squadsInRange []ecs.EntityID

	attackerPos, err := getSquadMapPosition(squadID, cas.manager)
	if err != nil {
		return squadsInRange // Return empty if squad not on map
	}

	attackerFaction := getFactionOwner(squadID, cas.manager)
	if attackerFaction == 0 {
		return squadsInRange // Return empty if no faction owner
	}

	maxRange := cas.GetSquadAttackRange(squadID)

	// Query all MapPositionData entities to find enemy squads
	for _, result := range cas.manager.World.Query(MapPositionTag) {
		mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)

		// Skip friendly squads
		if mapPos.FactionID == attackerFaction {
			continue
		}

		// Skip self (shouldn't happen, but safety check)
		if mapPos.SquadID == squadID {
			continue
		}

		// Calculate distance using Chebyshev distance
		distance := attackerPos.ChebyshevDistance(&mapPos.Position)

		// Check if in range
		if distance <= maxRange {
			squadsInRange = append(squadsInRange, mapPos.SquadID)
		}
	}

	return squadsInRange
}

func (cas *CombatActionSystem) CanSquadAttack(squadID, targetID ecs.EntityID) bool {
	// Check if squad has action available
	if !canSquadAct(squadID, cas.manager) {
		return false // Already acted this turn
	}

	// Get positions
	attackerPos, err := getSquadMapPosition(squadID, cas.manager)
	if err != nil {
		return false // Attacker not on map
	}

	defenderPos, err := getSquadMapPosition(targetID, cas.manager)
	if err != nil {
		return false // Defender not on map
	}

	// Check factions (can't attack allies)
	attackerFaction := getFactionOwner(squadID, cas.manager)
	defenderFaction := getFactionOwner(targetID, cas.manager)

	if attackerFaction == 0 || defenderFaction == 0 {
		return false // One or both squads have no faction
	}

	if attackerFaction == defenderFaction {
		return false // Can't attack own faction
	}

	// Calculate distance
	distance := attackerPos.ChebyshevDistance(&defenderPos)

	// Check range
	maxRange := cas.GetSquadAttackRange(squadID)
	if distance > maxRange {
		return false // Out of range
	}

	return true
}
