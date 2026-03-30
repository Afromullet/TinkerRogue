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

// GetRoundState returns the PerkRoundState for a squad, or nil if none exists.
func GetRoundState(squadID ecs.EntityID, manager *common.EntityManager) *PerkRoundState {
	return common.GetComponentTypeByID[*PerkRoundState](
		manager, squadID, PerkRoundStateComponent,
	)
}

// ========================================
// HOOK RUNNER FUNCTIONS
// Runners construct HookContext internally so callers pass flat params.
// Attacker/Defender variants read the corresponding PerkHooks field.
// ========================================

// RunAttackerDamageModHooks runs AttackerDamageMod hooks for the attacker's perks.
func RunAttackerDamageModHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, manager *common.EntityManager) {
	roundState := GetRoundState(attackerSquadID, manager)
	if roundState == nil {
		return
	}
	ctx := &HookContext{
		AttackerID: attackerID, DefenderID: defenderID,
		AttackerSquadID: attackerSquadID, DefenderSquadID: defenderSquadID,
		RoundState: roundState, Manager: manager,
	}
	for _, perkID := range getActivePerkIDs(attackerSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.AttackerDamageMod != nil {
			hooks.AttackerDamageMod(ctx, modifiers)
		}
	}
}

// RunDefenderDamageModHooks runs DefenderDamageMod hooks for the defender's perks.
func RunDefenderDamageModHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, manager *common.EntityManager) {
	roundState := GetRoundState(defenderSquadID, manager)
	if roundState == nil {
		return
	}
	ctx := &HookContext{
		AttackerID: attackerID, DefenderID: defenderID,
		AttackerSquadID: attackerSquadID, DefenderSquadID: defenderSquadID,
		RoundState: roundState, Manager: manager,
	}
	for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DefenderDamageMod != nil {
			hooks.DefenderDamageMod(ctx, modifiers)
		}
	}
}

// RunTargetOverrideHooks applies target overrides from attacker perks.
func RunTargetOverrideHooks(attackerID, defenderSquadID ecs.EntityID,
	targets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	attackerSquadID := getSquadIDForUnit(attackerID, manager)
	roundState := GetRoundState(attackerSquadID, manager)
	ctx := &HookContext{
		AttackerID: attackerID, DefenderID: 0,
		AttackerSquadID: attackerSquadID, DefenderSquadID: defenderSquadID,
		RoundState: roundState, Manager: manager,
	}
	for _, perkID := range getActivePerkIDs(attackerSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.TargetOverride != nil {
			targets = hooks.TargetOverride(ctx, targets)
		}
	}
	return targets
}

// RunCounterModHooks checks if counterattack should be suppressed or modified.
// Returns true if counter should be skipped.
func RunCounterModHooks(defenderSquadID, attackerID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, manager *common.EntityManager) bool {
	roundState := GetRoundState(defenderSquadID, manager)
	if roundState == nil {
		return false
	}
	for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.CounterMod != nil {
			if hooks.CounterMod(defenderSquadID, attackerID, modifiers, roundState, manager) {
				return true
			}
		}
	}
	return false
}

// RunAttackerPostDamageHooks runs post-damage hooks for the attacker's perks.
func RunAttackerPostDamageHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	roundState := GetRoundState(attackerSquadID, manager)
	if roundState == nil {
		return
	}
	ctx := &HookContext{
		AttackerID: attackerID, DefenderID: defenderID,
		AttackerSquadID: attackerSquadID, DefenderSquadID: defenderSquadID,
		RoundState: roundState, Manager: manager,
	}
	for _, perkID := range getActivePerkIDs(attackerSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.PostDamage != nil {
			hooks.PostDamage(ctx, damageDealt, wasKill)
		}
	}
}

// RunDefenderPostDamageHooks runs post-damage hooks for the defender's perks.
func RunDefenderPostDamageHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	roundState := GetRoundState(defenderSquadID, manager)
	if roundState == nil {
		return
	}
	ctx := &HookContext{
		AttackerID: attackerID, DefenderID: defenderID,
		AttackerSquadID: attackerSquadID, DefenderSquadID: defenderSquadID,
		RoundState: roundState, Manager: manager,
	}
	for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.PostDamage != nil {
			hooks.PostDamage(ctx, damageDealt, wasKill)
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

// RunCoverModHooks runs cover modification hooks for defender perks.
func RunCoverModHooks(attackerID, defenderID ecs.EntityID,
	coverBreakdown *combatcore.CoverBreakdown, manager *common.EntityManager) {
	attackerSquadID := getSquadIDForUnit(attackerID, manager)
	defenderSquadID := getSquadIDForUnit(defenderID, manager)

	defenderState := GetRoundState(defenderSquadID, manager)
	if defenderState == nil {
		return
	}
	ctx := &HookContext{
		AttackerID: attackerID, DefenderID: defenderID,
		AttackerSquadID: attackerSquadID, DefenderSquadID: defenderSquadID,
		RoundState: defenderState, Manager: manager,
	}
	for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DefenderCoverMod != nil {
			hooks.DefenderCoverMod(ctx, coverBreakdown)
		}
	}
}

// RunDeathOverrideHooks checks if lethal damage should be prevented.
func RunDeathOverrideHooks(unitID, squadID ecs.EntityID, manager *common.EntityManager) bool {
	roundState := GetRoundState(squadID, manager)
	if roundState == nil {
		return false
	}
	for _, perkID := range getActivePerkIDs(squadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DeathOverride != nil {
			if hooks.DeathOverride(unitID, squadID, roundState, manager) {
				return true
			}
		}
	}
	return false
}
