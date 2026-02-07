package combat

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

type CombatActionSystem struct {
	manager        *common.EntityManager
	combatCache    *CombatQueryCache
	battleRecorder *battlelog.BattleRecorder
}

func NewCombatActionSystem(manager *common.EntityManager, cache *CombatQueryCache) *CombatActionSystem {
	return &CombatActionSystem{
		manager:     manager,
		combatCache: cache,
	}
}

// SetBattleRecorder sets the battle recorder for combat log export.
func (cas *CombatActionSystem) SetBattleRecorder(recorder *battlelog.BattleRecorder) {
	cas.battleRecorder = recorder
}

func (cas *CombatActionSystem) ExecuteAttackAction(attackerID, defenderID ecs.EntityID) *squads.CombatResult {

	//Validation
	reason, canAttack := cas.CanSquadAttackWithReason(attackerID, defenderID)
	if !canAttack {
		return &squads.CombatResult{
			Success:     false,
			ErrorReason: reason,
		}
	}

	// Main Attack calculation

	result := &squads.CombatResult{
		DamageByUnit: make(map[ecs.EntityID]int),
		UnitsKilled:  []ecs.EntityID{},
	}

	// Initialize combat log with squad info
	combatLog := squads.InitializeCombatLog(attackerID, defenderID, cas.manager)
	if combatLog.SquadDistance < 0 {
		result.CombatLog = combatLog
		result.Success = false
		result.ErrorReason = "Squads not found or missing position"
		return result
	}

	// Snapshot units that will participate (for logging)
	combatLog.AttackingUnits = squads.SnapshotAttackingUnits(attackerID, combatLog.SquadDistance, cas.manager)
	combatLog.DefendingUnits = squads.SnapshotAllUnits(defenderID, cas.manager)

	// Process each attacking unit
	attackIndex := 0
	attackerUnitIDs := squads.GetUnitIDsInSquad(attackerID, cas.manager)

	for _, attackerUnitID := range attackerUnitIDs {
		// Check if unit can attack (alive, can act, and in range)
		if !squads.CanUnitAttack(attackerUnitID, combatLog.SquadDistance, cas.manager) {
			continue
		}

		// Get targets for this attacker
		targetIDs := squads.SelectTargetUnits(attackerUnitID, defenderID, cas.manager)

		// Attack each target
		attackIndex = squads.ProcessAttackOnTargets(attackerUnitID, targetIDs, result, combatLog, attackIndex, cas.manager)
	}

	// Counterattack

	// Check if defender would survive the main attack (checking HP after predicted damage)
	defenderWouldSurvive := squads.WouldSquadSurvive(defenderID, result.DamageByUnit, cas.manager)

	if defenderWouldSurvive {
		// Check if any defender units can reach the attacker
		counterattackers := cas.getCounterattackingUnits(defenderID, attackerID)

		if len(counterattackers) > 0 {
			// Process each counterattacking unit (using same distance)
			defenderUnitIDs := squads.GetUnitIDsInSquad(defenderID, cas.manager)

			for _, counterAttackerID := range defenderUnitIDs {
				// Check if unit can counterattack (alive after main attack, in range)
				entity := cas.manager.FindEntityByID(counterAttackerID)
				if entity == nil {
					continue
				}

				attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
				if attr == nil {
					continue
				}

				// Check if unit would survive the main attack damage
				damageToThisUnit := result.DamageByUnit[counterAttackerID]
				if attr.CurrentHealth-damageToThisUnit <= 0 {
					continue // Unit would be dead, can't counterattack
				}

				rangeData := common.GetComponentType[*squads.AttackRangeData](entity, squads.AttackRangeComponent)
				if rangeData == nil || rangeData.Range < combatLog.SquadDistance {
					continue
				}

				// Get targets (same targeting logic as normal attacks)
				targetIDs := squads.SelectTargetUnits(counterAttackerID, attackerID, cas.manager)

				// Counterattack each target with penalties
				attackIndex = squads.ProcessCounterattackOnTargets(counterAttackerID, targetIDs, result, combatLog, attackIndex, cas.manager)
			}

		}
	}

	// Finalize combat log with summary statistics
	squads.FinalizeCombatLog(result, combatLog, defenderID, attackerID, cas.manager)
	result.CombatLog = combatLog

	// Determien Destruction Status

	// Predict destruction based on recorded damage (before applying)
	// Reuse defenderWouldSurvive from Phase 3 (already calculated)
	attackerDestroyed := !squads.WouldSquadSurvive(attackerID, result.DamageByUnit, cas.manager)
	defenderDestroyed := !defenderWouldSurvive // Reuse cached value

	result.TargetDestroyed = defenderDestroyed

	// post combat

	// Apply all recorded damage to unit HP (STATE MODIFICATION STARTS HERE)
	squads.ApplyRecordedDamage(result, cas.manager)

	// Mark attacker squad as acted (turn state modification)
	markSquadAsActed(cas.combatCache, attackerID, cas.manager)

	// Record combat log for export (if enabled)
	if cas.battleRecorder != nil && cas.battleRecorder.IsEnabled() {
		cas.battleRecorder.RecordEngagement(result.CombatLog)
	}

	// Remove destroyed squads from map
	if attackerDestroyed {
		removeSquadFromMap(attackerID, cas.manager)
	}

	if defenderDestroyed {
		removeSquadFromMap(defenderID, cas.manager)
	}

	// Trigger abilities for surviving squads
	if !attackerDestroyed {
		squads.CheckAndTriggerAbilities(attackerID, cas.manager)
		squads.DisposeDeadUnitsInSquad(attackerID, cas.manager)
	}

	if !defenderDestroyed {
		squads.CheckAndTriggerAbilities(defenderID, cas.manager)
		squads.DisposeDeadUnitsInSquad(defenderID, cas.manager)
	}

	// Mark as successful
	result.Success = true
	return result
}

// GetSquadAttackRange returns the maximum attack range of any unit in the squad
func (cas *CombatActionSystem) GetSquadAttackRange(squadID ecs.EntityID) int {
	unitIDs := squads.GetUnitIDsInSquad(squadID, cas.manager)

	maxRange := 1 // Default melee
	for _, unitID := range unitIDs {

		entity := cas.manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		// Read from AttackRangeComponent (correct source for attack range)
		rangeData := common.GetComponentType[*squads.AttackRangeData](entity, squads.AttackRangeComponent)
		if rangeData == nil {
			continue
		}

		unitRange := rangeData.Range

		if unitRange > maxRange {
			maxRange = unitRange
		}
	}

	return maxRange // Squad can attack at max range of any unit
}

// getUnitsInRange returns units that can reach the target based on their range
// If checkCanAct is true, filters out units that have already acted (for attacks)
// If checkCanAct is false, includes all alive units regardless of action state (for counterattacks)
func (cas *CombatActionSystem) getUnitsInRange(squadID, targetID ecs.EntityID, checkCanAct bool) []ecs.EntityID {
	// Use GetSquadDistance for consistent Chebyshev distance calculation
	distance := squads.GetSquadDistance(squadID, targetID, cas.manager)
	if distance < 0 {
		return []ecs.EntityID{} // Squad not found or missing position
	}

	allUnits := squads.GetUnitIDsInSquad(squadID, cas.manager)
	var unitsInRange []ecs.EntityID

	for _, unitID := range allUnits {
		entity := cas.manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}

		// For attacks, check CanAct flag. For counterattacks, skip this check
		if checkCanAct && !attr.CanAct {
			continue
		}

		// Read from AttackRangeComponent (correct source for attack range)
		rangeData := common.GetComponentType[*squads.AttackRangeData](entity, squads.AttackRangeComponent)
		if rangeData == nil {
			continue
		}

		// Check if this unit can reach the target
		if rangeData.Range >= distance {
			unitsInRange = append(unitsInRange, unitID)
		}
	}

	return unitsInRange
}

// GetAttackingUnits returns units that can attack the target based on their range
// Only includes units that haven't acted yet (CanAct=true)
func (cas *CombatActionSystem) GetAttackingUnits(squadID, targetID ecs.EntityID) []ecs.EntityID {
	return cas.getUnitsInRange(squadID, targetID, true)
}

// getCounterattackingUnits returns defender units that can counterattack the attacker
// Counterattacks are free actions, so includes all alive units regardless of action state
func (cas *CombatActionSystem) getCounterattackingUnits(defenderID, attackerID ecs.EntityID) []ecs.EntityID {
	return cas.getUnitsInRange(defenderID, attackerID, false)
}

// CanSquadAttackWithReason returns detailed info about why an attack can/cannot happen
func (cas *CombatActionSystem) CanSquadAttackWithReason(squadID, targetID ecs.EntityID) (string, bool) {
	// Check if squad has action available
	if !canSquadAct(cas.combatCache, squadID, cas.manager) {
		return "Squad has already acted this turn", false
	}

	// Get positions
	attackerPos, err := GetSquadMapPosition(squadID, cas.manager)
	if err != nil {
		return "Attacker squad not found on map", false
	}

	defenderPos, err := GetSquadMapPosition(targetID, cas.manager)
	if err != nil {
		return "Target squad not found on map", false
	}

	// Check factions (can't attack allies)
	attackerFaction := GetSquadFaction(squadID, cas.manager)
	defenderFaction := GetSquadFaction(targetID, cas.manager)

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
