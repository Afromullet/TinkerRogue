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

// buildHookContext constructs a HookContext with the round state for the specified owner squad.
// Returns nil if the owner squad has no PerkRoundState.
func buildHookContext(ownerSquadID ecs.EntityID, manager *common.EntityManager) *HookContext {
	roundState := GetRoundState(ownerSquadID, manager)
	if roundState == nil {
		return nil
	}
	return &HookContext{
		RoundState: roundState,
		Manager:    manager,
	}
}

// ========================================
// HOOK RUNNER FUNCTIONS
// All runners use buildHookContext for consistent context construction.
// ========================================

// RunAttackerDamageModHooks runs AttackerDamageMod hooks for the attacker's perks.
func RunAttackerDamageModHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, manager *common.EntityManager) {
	ctx := buildHookContext(attackerSquadID, manager)
	if ctx == nil {
		return
	}
	ctx.AttackerID = attackerID
	ctx.DefenderID = defenderID
	ctx.AttackerSquadID = attackerSquadID
	ctx.DefenderSquadID = defenderSquadID
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
	ctx := buildHookContext(defenderSquadID, manager)
	if ctx == nil {
		return
	}
	ctx.AttackerID = attackerID
	ctx.DefenderID = defenderID
	ctx.AttackerSquadID = attackerSquadID
	ctx.DefenderSquadID = defenderSquadID
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
	ctx := buildHookContext(attackerSquadID, manager)
	if ctx == nil {
		return targets
	}
	ctx.AttackerID = attackerID
	ctx.AttackerSquadID = attackerSquadID
	ctx.DefenderSquadID = defenderSquadID
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
	ctx := buildHookContext(defenderSquadID, manager)
	if ctx == nil {
		return false
	}
	ctx.DefenderSquadID = defenderSquadID
	ctx.AttackerID = attackerID
	ctx.SquadID = defenderSquadID
	for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.CounterMod != nil {
			if hooks.CounterMod(ctx, modifiers) {
				return true
			}
		}
	}
	return false
}

// RunAttackerPostDamageHooks runs post-damage hooks for the attacker's perks.
func RunAttackerPostDamageHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	ctx := buildHookContext(attackerSquadID, manager)
	if ctx == nil {
		return
	}
	ctx.AttackerID = attackerID
	ctx.DefenderID = defenderID
	ctx.AttackerSquadID = attackerSquadID
	ctx.DefenderSquadID = defenderSquadID
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
	ctx := buildHookContext(defenderSquadID, manager)
	if ctx == nil {
		return
	}
	ctx.AttackerID = attackerID
	ctx.DefenderID = defenderID
	ctx.AttackerSquadID = attackerSquadID
	ctx.DefenderSquadID = defenderSquadID
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
	ctx := &HookContext{
		SquadID:     squadID,
		RoundNumber: roundNumber,
		RoundState:  roundState,
		Manager:     manager,
	}
	for _, perkID := range getActivePerkIDs(squadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.TurnStart != nil {
			hooks.TurnStart(ctx)
		}
	}
}

// RunCoverModHooks runs cover modification hooks for defender perks.
func RunCoverModHooks(attackerID, defenderID ecs.EntityID,
	coverBreakdown *combatcore.CoverBreakdown, manager *common.EntityManager) {
	attackerSquadID := getSquadIDForUnit(attackerID, manager)
	defenderSquadID := getSquadIDForUnit(defenderID, manager)
	ctx := buildHookContext(defenderSquadID, manager)
	if ctx == nil {
		return
	}
	ctx.AttackerID = attackerID
	ctx.DefenderID = defenderID
	ctx.AttackerSquadID = attackerSquadID
	ctx.DefenderSquadID = defenderSquadID
	for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DefenderCoverMod != nil {
			hooks.DefenderCoverMod(ctx, coverBreakdown)
		}
	}
}

// RunDeathOverrideHooks checks if lethal damage should be prevented.
func RunDeathOverrideHooks(unitID, squadID ecs.EntityID, manager *common.EntityManager) bool {
	ctx := buildHookContext(squadID, manager)
	if ctx == nil {
		return false
	}
	ctx.UnitID = unitID
	ctx.SquadID = squadID
	for _, perkID := range getActivePerkIDs(squadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DeathOverride != nil {
			if hooks.DeathOverride(ctx) {
				return true
			}
		}
	}
	return false
}

// RunDamageRedirectHooks checks if damage should be redirected.
// Returns reduced damage for original target, redirect target ID, and redirect amount.
func RunDamageRedirectHooks(defenderID, defenderSquadID ecs.EntityID,
	damageAmount int, manager *common.EntityManager) (int, ecs.EntityID, int) {
	ctx := buildHookContext(defenderSquadID, manager)
	if ctx == nil {
		return damageAmount, 0, 0
	}
	ctx.UnitID = defenderID
	ctx.SquadID = defenderSquadID
	ctx.DefenderID = defenderID
	ctx.DefenderSquadID = defenderSquadID
	ctx.DamageAmount = damageAmount
	for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DamageRedirect != nil {
			reducedDmg, redirectTarget, redirectAmt := hooks.DamageRedirect(ctx)
			if redirectTarget != 0 {
				return reducedDmg, redirectTarget, redirectAmt
			}
		}
	}
	return damageAmount, 0, 0
}
