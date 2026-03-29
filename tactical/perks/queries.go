package perks

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// getActivePerkIDs returns all perk IDs equipped on a squad.
func getActivePerkIDs(squadID ecs.EntityID, manager *common.EntityManager) []string {
	if data := common.GetComponentTypeByID[*PerkSlotData](
		manager, squadID, PerkSlotComponent,
	); data != nil {
		return data.PerkIDs
	}
	return nil
}

// HasPerk checks if a squad has a specific perk equipped.
func HasPerk(squadID ecs.EntityID, perkID string, manager *common.EntityManager) bool {
	for _, id := range getActivePerkIDs(squadID, manager) {
		if id == perkID {
			return true
		}
	}
	return false
}

// getSquadIDForUnit returns the parent squad ID for a unit.
func getSquadIDForUnit(unitID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	if memberData := common.GetComponentTypeByID[*squadcore.SquadMemberData](
		manager, unitID, squadcore.SquadMemberComponent,
	); memberData != nil {
		return memberData.SquadID
	}
	return 0
}

// GetRoundState returns the PerkRoundState for a squad, creating one if it doesn't exist.
func GetRoundState(squadID ecs.EntityID, manager *common.EntityManager) *PerkRoundState {
	state := common.GetComponentTypeByID[*PerkRoundState](
		manager, squadID, PerkRoundStateComponent,
	)
	return state
}

// ========================================
// HOOK RUNNER FUNCTIONS
// ========================================

// RunDamageModHooks runs all DamageMod hooks for an attacker's perks.
func RunDamageModHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState == nil {
		return
	}
	for _, perkID := range getActivePerkIDs(attackerSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DamageMod != nil {
			hooks.DamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID, modifiers, roundState, manager)
		}
	}
}

// RunDefenderDamageModHooks runs hooks for the DEFENDER's perks
// (e.g., Shieldwall Discipline, Vigilance, Adaptive Armor).
func RunDefenderDamageModHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState == nil {
		return
	}
	for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DamageMod != nil {
			hooks.DamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID, modifiers, roundState, manager)
		}
	}
}

// RunTargetOverrideHooks applies target overrides from attacker perks.
func RunTargetOverrideHooks(attackerID, defenderSquadID ecs.EntityID,
	targets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	attackerSquadID := getSquadIDForUnit(attackerID, manager)
	for _, perkID := range getActivePerkIDs(attackerSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.TargetOverride != nil {
			targets = hooks.TargetOverride(attackerID, defenderSquadID, targets, manager)
		}
	}
	return targets
}

// RunCounterModHooks checks if counterattack should be suppressed or modified.
// Returns true if counter should be skipped.
func RunCounterModHooks(defenderSquadID, attackerID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) bool {
	if roundState == nil {
		return false
	}
	for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.CounterMod != nil {
			if hooks.CounterMod(defenderSquadID, attackerID, modifiers, roundState, manager) {
				return true // Skip counter
			}
		}
	}
	return false
}

// RunPostDamageHooks runs post-damage hooks for the attacker's perks.
func RunPostDamageHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState == nil {
		return
	}
	for _, perkID := range getActivePerkIDs(attackerSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.PostDamage != nil {
			hooks.PostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID, damageDealt, wasKill, roundState, manager)
		}
	}
}

// RunDefenderPostDamageHooks runs post-damage hooks for the defender's perks.
// Used by Grudge Bearer to track who attacked the defending squad.
func RunDefenderPostDamageHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState == nil {
		return
	}
	for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.PostDamage != nil {
			hooks.PostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID, damageDealt, wasKill, roundState, manager)
		}
	}
}

// RunTurnStartHooks runs turn-start hooks for a squad.
func RunTurnStartHooks(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState == nil {
		return
	}
	for _, perkID := range getActivePerkIDs(squadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.TurnStart != nil {
			hooks.TurnStart(squadID, roundNumber, roundState, manager)
		}
	}
}

// RunCoverModHooks runs cover modification hooks for both attacker and defender perks.
func RunCoverModHooks(attackerID, defenderID ecs.EntityID,
	coverBreakdown *combatcore.CoverBreakdown, manager *common.EntityManager) {
	attackerSquadID := getSquadIDForUnit(attackerID, manager)
	defenderSquadID := getSquadIDForUnit(defenderID, manager)

	// Check attacker perks
	attackerState := GetRoundState(attackerSquadID, manager)
	if attackerState != nil {
		for _, perkID := range getActivePerkIDs(attackerSquadID, manager) {
			hooks := GetPerkHooks(perkID)
			if hooks != nil && hooks.CoverMod != nil {
				hooks.CoverMod(attackerID, defenderID, coverBreakdown, attackerState, manager)
			}
		}
	}

	// Check defender perks (Brace for Impact, Fortify)
	defenderState := GetRoundState(defenderSquadID, manager)
	if defenderState != nil {
		for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
			hooks := GetPerkHooks(perkID)
			if hooks != nil && hooks.CoverMod != nil {
				hooks.CoverMod(attackerID, defenderID, coverBreakdown, defenderState, manager)
			}
		}
	}
}

// RunDamageRedirectHooks checks if damage should be redirected.
// Returns the modified damage amount for the original target,
// plus a redirect target and amount.
func RunDamageRedirectHooks(defenderID, defenderSquadID ecs.EntityID,
	damageAmount int, manager *common.EntityManager) (int, ecs.EntityID, int) {
	for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DamageRedirect != nil {
			reducedDmg, redirectTarget, redirectAmt := hooks.DamageRedirect(defenderID, defenderSquadID, damageAmount, manager)
			if redirectTarget != 0 {
				return reducedDmg, redirectTarget, redirectAmt
			}
		}
	}
	return damageAmount, 0, 0
}

// RunDeathOverrideHooks checks if lethal damage should be prevented.
func RunDeathOverrideHooks(unitID, squadID ecs.EntityID,
	roundState *PerkRoundState, manager *common.EntityManager) bool {
	if roundState == nil {
		return false
	}
	for _, perkID := range getActivePerkIDs(squadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DeathOverride != nil {
			if hooks.DeathOverride(unitID, squadID, roundState, manager) {
				return true // Prevent death
			}
		}
	}
	return false
}
