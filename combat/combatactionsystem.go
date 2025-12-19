package combat

import (
	"fmt"
	"game_main/common"
	"game_main/config"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

type CombatActionSystem struct {
	manager     *common.EntityManager
	combatCache *CombatQueryCache
}

func NewCombatActionSystem(manager *common.EntityManager) *CombatActionSystem {
	return &CombatActionSystem{
		manager:     manager,
		combatCache: NewCombatQueryCache(manager),
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

	if !canSquadAct(cas.combatCache, attackerID, cas.manager) {
		return fmt.Errorf("squad has already acted this turn")
	}

	//Filter units by range (partial squad attacks)
	attackingUnits := cas.GetAttackingUnits(attackerID, defenderID)

	// Temporarily disable out-of-range units
	allUnits := squads.GetUnitIDsInSquad(attackerID, cas.manager)
	disabledUnits := []ecs.EntityID{}

	for _, unitID := range allUnits {
		if !containsEntity(attackingUnits, unitID) {
			entity := common.FindEntityByID(cas.manager, unitID)
			if entity == nil {
				continue
			}

			attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
			if attr != nil && attr.CanAct {
				attr.CanAct = false
				disabledUnits = append(disabledUnits, unitID)
			}
		}
	}

	// Execute attack (only CanAct=true units participate)
	result := squads.ExecuteSquadAttack(attackerID, defenderID, cas.manager)

	// Re-enable disabled units. TODO: This might allow squads to attack twice. Once ranged, and then melee
	// (If the squad has a mix of units, and melee units did not attack due to the range). Test and fix this
	for _, unitID := range disabledUnits {
		entity := common.FindEntityByID(cas.manager, unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr != nil {
			attr.CanAct = true
		}
	}

	markSquadAsActed(cas.combatCache, attackerID, cas.manager)

	if squads.IsSquadDestroyed(defenderID, cas.manager) {
		removeSquadFromMap(defenderID, cas.manager)
	}

	// Display detailed combat log. Only prints in display mode.
	if config.DEBUG_MODE && result.CombatLog != nil {
		DisplayCombatLog(result.CombatLog, cas.manager)
	}

	// Check abilities for both squads after combat
	// Attacker abilities: might trigger based on damage dealt, turn count, etc.
	squads.CheckAndTriggerAbilities(attackerID, cas.manager)

	// Defender abilities: might trigger healing if HP is low, or other defensive abilities
	if !squads.IsSquadDestroyed(defenderID, cas.manager) {
		squads.CheckAndTriggerAbilities(defenderID, cas.manager)
	}

	return nil
}

// GetSquadAttackRange returns the maximum attack range of any unit in the squad
func (cas *CombatActionSystem) GetSquadAttackRange(squadID ecs.EntityID) int {
	unitIDs := squads.GetUnitIDsInSquad(squadID, cas.manager)

	maxRange := 1 // Default melee
	for _, unitID := range unitIDs {

		entity := common.FindEntityByID(cas.manager, unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil {
			continue
		}

		unitRange := attr.GetAttackRange()

		if unitRange > maxRange {
			maxRange = unitRange
		}
	}

	return maxRange // Squad can attack at max range of any unit
}

// GetAttackingUnits returns units that can attack the target based on their range
func (cas *CombatActionSystem) GetAttackingUnits(squadID, targetID ecs.EntityID) []ecs.EntityID {
	// Use GetSquadDistance for consistent Chebyshev distance calculation
	distance := squads.GetSquadDistance(squadID, targetID, cas.manager)
	if distance < 0 {
		return []ecs.EntityID{} // Squad not found or missing position
	}

	allUnits := squads.GetUnitIDsInSquad(squadID, cas.manager)

	var attackingUnits []ecs.EntityID

	for _, unitID := range allUnits {

		entity := common.FindEntityByID(cas.manager, unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil {
			continue
		}

		// Check if unit is alive and can act (matches canUnitAttack logic)
		if attr.CurrentHealth <= 0 || !attr.CanAct {
			continue
		}

		unitRange := attr.GetAttackRange()

		// Check if this unit can reach the target
		if unitRange >= distance {
			attackingUnits = append(attackingUnits, unitID)
		}
	}

	return attackingUnits
}

// CanSquadAttackWithReason returns detailed info about why an attack can/cannot happen
func (cas *CombatActionSystem) CanSquadAttackWithReason(squadID, targetID ecs.EntityID) (string, bool) {
	// Check if squad has action available
	if !canSquadAct(cas.combatCache, squadID, cas.manager) {
		return "Squad has already acted this turn", false
	}

	// Get positions
	attackerPos, err := getSquadMapPosition(squadID, cas.manager)
	if err != nil {
		return "Attacker squad not found on map", false
	}

	defenderPos, err := getSquadMapPosition(targetID, cas.manager)
	if err != nil {
		return "Target squad not found on map", false
	}

	// Check factions (can't attack allies)
	attackerFaction := getFactionOwner(squadID, cas.manager)
	defenderFaction := getFactionOwner(targetID, cas.manager)

	if attackerFaction == 0 || defenderFaction == 0 {
		return "One or both squads have no faction", false
	}

	if attackerFaction == defenderFaction {
		return "Cannot attack your own faction", false
	}

	// Calculate distance
	distance := attackerPos.ChebyshevDistance(&defenderPos)

	// Check range
	maxRange := cas.GetSquadAttackRange(squadID)
	if distance > maxRange {
		return fmt.Sprintf("Target out of range: %d tiles away (max range %d)", distance, maxRange), false
	}

	return "Attack valid", true
}
