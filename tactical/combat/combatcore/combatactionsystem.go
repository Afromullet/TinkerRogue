package combatcore

import (
	"fmt"
	"game_main/core/common"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combat/combatmath"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

type CombatActionSystem struct {
	manager        *common.EntityManager
	combatCache    *combatstate.CombatQueryCache
	battleRecorder *battlelog.BattleRecorder

	// Post-action hook (fired after successful attack)
	onAttackComplete func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)

	// Perk dispatcher (injected from perks package via combatservices)
	perkDispatcher combattypes.PerkDispatcher
}

func NewCombatActionSystem(manager *common.EntityManager, cache *combatstate.CombatQueryCache) *CombatActionSystem {
	return &CombatActionSystem{
		manager:     manager,
		combatCache: cache,
	}
}

// SetBattleRecorder sets the battle recorder for combat log export.
func (cas *CombatActionSystem) SetBattleRecorder(recorder *battlelog.BattleRecorder) {
	cas.battleRecorder = recorder
}

// SetOnAttackComplete sets the callback fired after a successful attack.
func (cas *CombatActionSystem) SetOnAttackComplete(fn func(ecs.EntityID, ecs.EntityID, *combattypes.CombatResult)) {
	cas.onAttackComplete = fn
}

// SetPerkDispatcher injects the perk dispatcher for damage pipeline hooks.
// Must be called before combat begins for perk hooks to fire.
func (cas *CombatActionSystem) SetPerkDispatcher(dispatcher combattypes.PerkDispatcher) {
	cas.perkDispatcher = dispatcher
}

func (cas *CombatActionSystem) ExecuteAttackAction(attackerID, defenderID ecs.EntityID) *combattypes.CombatResult {

	// Validation
	reason, canAttack := cas.canSquadAttackWithReason(attackerID, defenderID)
	if !canAttack {
		return &combattypes.CombatResult{
			Success:     false,
			ErrorReason: reason,
		}
	}

	result := &combattypes.CombatResult{
		DamageByUnit:  make(map[ecs.EntityID]int),
		HealingByUnit: make(map[ecs.EntityID]int),
		UnitsKilled:   []ecs.EntityID{},
	}

	// Initialize combat log with squad info
	combatLog := battlelog.InitializeCombatLog(attackerID, defenderID, cas.manager)
	if combatLog.SquadDistance < 0 {
		result.CombatLog = combatLog
		result.Success = false
		result.ErrorReason = "Squads not found or missing position"
		return result
	}

	// Snapshot units that will participate (for logging)
	combatLog.AttackingUnits = battlelog.SnapshotAttackingUnits(attackerID, combatLog.SquadDistance, cas.manager)
	combatLog.DefendingUnits = battlelog.SnapshotAllUnits(defenderID, cas.manager)

	// Execute combat phases
	attackIndex := cas.executeMainAttack(attackerID, defenderID, result, combatLog)
	_ = cas.executeCounterattack(attackerID, defenderID, result, combatLog, attackIndex)
	cas.applyPostCombatEffects(attackerID, defenderID, result, combatLog)

	return result
}

// executeMainAttack processes each attacking unit's action (attack or heal).
// Returns the final attack index for sequencing subsequent events.
func (cas *CombatActionSystem) executeMainAttack(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult, combatLog *combattypes.CombatLog) int {
	attackIndex := 0
	attackerUnitIDs := squadcore.GetUnitIDsInSquad(attackerID, cas.manager)

	for _, attackerUnitID := range attackerUnitIDs {
		if !combatmath.CanUnitAttack(attackerUnitID, combatLog.SquadDistance, cas.manager) {
			continue
		}

		if combatmath.IsHealUnit(attackerUnitID, cas.manager) {
			healTargets := combatmath.SelectHealTargets(attackerUnitID, attackerID, cas.manager)
			attackIndex = ProcessHealOnTargets(attackerUnitID, healTargets, result, combatLog, attackIndex, cas.manager)
		} else {
			targetIDs := combatmath.SelectTargetUnits(attackerUnitID, defenderID, cas.manager)
			attackIndex = ProcessAttackOnTargets(attackerUnitID, defenderID, targetIDs, result, combatLog, attackIndex, cas.perkDispatcher, cas.manager)
		}
	}

	return attackIndex
}

// executeCounterattack handles the defender's counterattack phase.
// Checks survival, applies perk modifiers, filters eligible units, and processes their actions.
func (cas *CombatActionSystem) executeCounterattack(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult, combatLog *combattypes.CombatLog, attackIndex int) int {
	// Check if defender would survive the main attack
	defenderWouldSurvive := squadcore.WouldSquadSurvive(defenderID, result.DamageByUnit, cas.manager)

	// Build counterattack modifiers (may be modified by perk hooks)
	counterModifiers := combattypes.DamageModifiers{
		HitPenalty:       counterattackHitPenalty(),
		DamageMultiplier: counterattackDamageMultiplier(),
		IsCounterattack:  true,
	}

	// Check if counter should be suppressed by perks
	skipCounter := false
	if cas.perkDispatcher != nil {
		skipCounter = cas.perkDispatcher.CounterMod(defenderID, attackerID, &counterModifiers, cas.manager)
	}

	if !defenderWouldSurvive || skipCounter || counterModifiers.SkipCounter {
		return attackIndex
	}

	counterattackers := cas.getCounterattackingUnits(defenderID, attackerID)

	for _, counterAttackerID := range counterattackers {
		if !cas.wouldUnitSurviveDamage(counterAttackerID, result) {
			continue
		}

		if combatmath.IsHealUnit(counterAttackerID, cas.manager) {
			healTargets := combatmath.SelectHealTargets(counterAttackerID, defenderID, cas.manager)
			attackIndex = ProcessHealOnTargets(counterAttackerID, healTargets, result, combatLog, attackIndex, cas.manager)
		} else {
			targetIDs := combatmath.SelectTargetUnits(counterAttackerID, attackerID, cas.manager)
			attackIndex = ProcessCounterattackOnTargets(counterAttackerID, attackerID, targetIDs, result, combatLog, attackIndex, counterModifiers, cas.perkDispatcher, cas.manager)
		}
	}

	return attackIndex
}

// wouldUnitSurviveDamage checks if a unit would survive the damage already recorded against it.
func (cas *CombatActionSystem) wouldUnitSurviveDamage(unitID ecs.EntityID, result *combattypes.CombatResult) bool {
	damageToUnit := result.DamageByUnit[unitID]
	if damageToUnit <= 0 {
		return true
	}

	entity := cas.manager.FindEntityByID(unitID)
	if entity == nil {
		return false
	}

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	return attr != nil && attr.CurrentHealth-damageToUnit > 0
}

// applyPostCombatEffects finalizes the combat log, applies damage/healing, handles squad
// destruction, triggers abilities, and fires the post-attack hook.
func (cas *CombatActionSystem) applyPostCombatEffects(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult, combatLog *combattypes.CombatLog) {
	// Finalize combat log with summary statistics
	battlelog.FinalizeCombatLog(result, combatLog, defenderID, attackerID, cas.manager)
	result.CombatLog = combatLog

	// Determine destruction status (before applying damage)
	attackerDestroyed := !squadcore.WouldSquadSurvive(attackerID, result.DamageByUnit, cas.manager)
	defenderDestroyed := !squadcore.WouldSquadSurvive(defenderID, result.DamageByUnit, cas.manager)

	result.TargetDestroyed = defenderDestroyed
	result.AttackerDestroyed = attackerDestroyed

	// Apply all recorded damage and healing (STATE MODIFICATION STARTS HERE)
	combatmath.ApplyRecordedDamage(result, cas.manager)
	combatmath.ApplyRecordedHealing(result, cas.manager)

	// Mark attacker squad as acted
	combatstate.MarkSquadAsActed(cas.combatCache, attackerID, cas.manager)

	// Record combat log for export (if enabled)
	if cas.battleRecorder != nil && cas.battleRecorder.IsEnabled() {
		cas.battleRecorder.RecordEngagement(result.CombatLog)
	}

	// Remove destroyed squads from map
	if attackerDestroyed {
		combatstate.RemoveSquadFromMap(attackerID, cas.manager)
	}
	if defenderDestroyed {
		combatstate.RemoveSquadFromMap(defenderID, cas.manager)
	}

	// Trigger abilities and dispose dead units for surviving squads
	if !attackerDestroyed {
		CheckAndTriggerAbilities(attackerID, cas.manager)
		squadcore.DisposeDeadUnitsInSquad(attackerID, cas.manager)
	}
	if !defenderDestroyed {
		CheckAndTriggerAbilities(defenderID, cas.manager)
		squadcore.DisposeDeadUnitsInSquad(defenderID, cas.manager)
	}

	result.Success = true

	// Fire post-attack hook
	if cas.onAttackComplete != nil {
		cas.onAttackComplete(attackerID, defenderID, result)
	}
}

// getSquadAttackRange returns the maximum attack range of any unit in the squad
func (cas *CombatActionSystem) getSquadAttackRange(squadID ecs.EntityID) int {
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, cas.manager)

	maxRange := 1 // Default melee
	for _, unitID := range unitIDs {

		entity := cas.manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		// Read from AttackRangeComponent (correct source for attack range)
		rangeData := common.GetComponentType[*squadcore.AttackRangeData](entity, squadcore.AttackRangeComponent)
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
	distance := squadcore.GetSquadDistance(squadID, targetID, cas.manager)
	if distance < 0 {
		return []ecs.EntityID{} // Squad not found or missing position
	}

	allUnits := squadcore.GetUnitIDsInSquad(squadID, cas.manager)
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
		rangeData := common.GetComponentType[*squadcore.AttackRangeData](entity, squadcore.AttackRangeComponent)
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

// canSquadAttackWithReason returns detailed info about why an attack can/cannot happen
func (cas *CombatActionSystem) canSquadAttackWithReason(squadID, targetID ecs.EntityID) (string, bool) {
	// Check if squad has action available
	if !combatstate.CanSquadAct(cas.combatCache, squadID, cas.manager) {
		return "Squad has already acted this turn", false
	}

	// Get positions
	attackerPos, err := combatstate.GetSquadMapPosition(squadID, cas.manager)
	if err != nil {
		return "Attacker squad not found on map", false
	}

	defenderPos, err := combatstate.GetSquadMapPosition(targetID, cas.manager)
	if err != nil {
		return "Target squad not found on map", false
	}

	// Check factions (can't attack allies)
	attackerFaction := combatstate.GetSquadFaction(squadID, cas.manager)
	defenderFaction := combatstate.GetSquadFaction(targetID, cas.manager)

	if attackerFaction == 0 || defenderFaction == 0 {
		return "One or both squads have no faction", false
	}

	if attackerFaction == defenderFaction {
		return "Cannot attack your own faction", false
	}

	// Calculate distance
	distance := attackerPos.ChebyshevDistance(&defenderPos)

	// Check range
	maxRange := cas.getSquadAttackRange(squadID)
	if distance > maxRange {
		return fmt.Sprintf("Target out of range: %d tiles away (max range %d)", distance, maxRange), false
	}

	return "Attack valid", true
}
