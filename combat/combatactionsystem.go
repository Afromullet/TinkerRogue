package combat

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

type CombatActionSystem struct {
	manager     *common.EntityManager
	combatCache *CombatQueryCache // Cached queries for O(k) instead of O(n)
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
			attr := common.GetAttributesByIDWithTag(cas.manager, unitID, squads.SquadMemberTag)
			if attr != nil && attr.CanAct {
				attr.CanAct = false
				disabledUnits = append(disabledUnits, unitID)
			}
		}
	}

	// Execute attack (only CanAct=true units participate)
	result := squads.ExecuteSquadAttack(attackerID, defenderID, cas.manager)

	// Re-enable disabled units
	for _, unitID := range disabledUnits {
		attr := common.GetAttributesByIDWithTag(cas.manager, unitID, squads.SquadMemberTag)
		if attr != nil {
			attr.CanAct = true
		}
	}

	markSquadAsActed(cas.combatCache, attackerID, cas.manager)

	if squads.IsSquadDestroyed(defenderID, cas.manager) {
		removeSquadFromMap(defenderID, cas.manager)
	}

	// Display detailed combat log
	if result.CombatLog != nil {
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

func (cas *CombatActionSystem) GetSquadAttackRange(squadID ecs.EntityID) int {
	unitIDs := squads.GetUnitIDsInSquad(squadID, cas.manager)

	maxRange := 1 // Default melee
	for _, unitID := range unitIDs {
		attr := common.GetAttributesByIDWithTag(cas.manager, unitID, squads.SquadMemberTag)
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

func (cas *CombatActionSystem) GetAttackingUnits(squadID, targetID ecs.EntityID) []ecs.EntityID {
	// Use GetSquadDistance for consistent Chebyshev distance calculation
	distance := squads.GetSquadDistance(squadID, targetID, cas.manager)
	if distance < 0 {
		return []ecs.EntityID{} // Squad not found or missing position
	}

	allUnits := squads.GetUnitIDsInSquad(squadID, cas.manager)

	var attackingUnits []ecs.EntityID

	for _, unitID := range allUnits {
		attr := common.GetAttributesByIDWithTag(cas.manager, unitID, squads.SquadMemberTag)
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

	// Query all squads in combat (NO MORE MAPPOSITION QUERIES!)
	for _, result := range cas.manager.World.Query(squads.SquadTag) {
		targetSquadID := result.Entity.GetID()

		// Skip self
		if targetSquadID == squadID {
			continue
		}

		// Get target faction (only consider squads in combat)
		combatFaction := common.GetComponentType[*CombatFactionData](result.Entity, CombatFactionComponent)
		if combatFaction == nil {
			continue // Squad not in combat
		}

		// Skip friendly squads
		if combatFaction.FactionID == attackerFaction {
			continue
		}

		// Get target position
		targetPos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
		if targetPos == nil {
			continue
		}

		// Calculate distance using Chebyshev distance
		distance := attackerPos.ChebyshevDistance(targetPos)

		// Check if in range
		if distance <= maxRange {
			squadsInRange = append(squadsInRange, targetSquadID)
		}
	}

	return squadsInRange
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
