package perks

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat/combatcore"

	"github.com/bytearena/ecs"
)

// MaxPerkSlots is the maximum number of perks a squad can equip.
const MaxPerkSlots = 3

// EquipPerk adds a perk to a squad's perk slot.
// Returns an error if the perk is already equipped, the slot is full,
// or the perk is exclusive with an already-equipped perk.
func EquipPerk(squadID ecs.EntityID, perkID string, maxSlots int, manager *common.EntityManager) error {
	def := GetPerkDefinition(perkID)
	if def == nil {
		return fmt.Errorf("perk %q not found in registry", perkID)
	}

	entity := manager.FindEntityByID(squadID)
	if entity == nil {
		return fmt.Errorf("squad %d not found", squadID)
	}

	slotData := common.GetComponentType[*PerkSlotData](entity, PerkSlotComponent)
	if slotData == nil {
		// Squad doesn't have a PerkSlotComponent yet; add one
		slotData = &PerkSlotData{PerkIDs: []string{}}
		entity.AddComponent(PerkSlotComponent, slotData)
	}

	// Check if already equipped
	for _, id := range slotData.PerkIDs {
		if id == perkID {
			return fmt.Errorf("perk %q already equipped", perkID)
		}
	}

	// Check slot capacity
	if len(slotData.PerkIDs) >= maxSlots {
		return fmt.Errorf("all %d perk slots are full", maxSlots)
	}

	// Check mutual exclusivity
	for _, equippedID := range slotData.PerkIDs {
		for _, exID := range def.ExclusiveWith {
			if equippedID == exID {
				return fmt.Errorf("perk %q is exclusive with already-equipped perk %q", perkID, exID)
			}
		}
	}

	slotData.PerkIDs = append(slotData.PerkIDs, perkID)
	return nil
}

// UnequipPerk removes a perk from a squad's perk slot.
func UnequipPerk(squadID ecs.EntityID, perkID string, manager *common.EntityManager) error {
	slotData := common.GetComponentTypeByID[*PerkSlotData](manager, squadID, PerkSlotComponent)
	if slotData == nil {
		return fmt.Errorf("squad %d has no perks equipped", squadID)
	}

	for i, id := range slotData.PerkIDs {
		if id == perkID {
			slotData.PerkIDs = append(slotData.PerkIDs[:i], slotData.PerkIDs[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("perk %q not equipped on squad %d", perkID, squadID)
}

// InitializeRoundState creates a fresh PerkRoundState on a squad entity for a new combat.
// Should be called during combat initialization for each squad that has perks.
func InitializeRoundState(squadID ecs.EntityID, manager *common.EntityManager) {
	entity := manager.FindEntityByID(squadID)
	if entity == nil {
		return
	}

	state := &PerkRoundState{}
	entity.AddComponent(PerkRoundStateComponent, state)
}

// CleanupRoundState removes the PerkRoundState from a squad entity after combat.
func CleanupRoundState(squadID ecs.EntityID, manager *common.EntityManager) {
	entity := manager.FindEntityByID(squadID)
	if entity == nil {
		return
	}
	entity.RemoveComponent(PerkRoundStateComponent)
}

// InitializePerkRoundStatesForFaction initializes PerkRoundState for all squads in a faction.
func InitializePerkRoundStatesForFaction(factionSquadIDs []ecs.EntityID, manager *common.EntityManager) {
	for _, squadID := range factionSquadIDs {
		if HasAnyPerks(squadID, manager) {
			InitializeRoundState(squadID, manager)
		}
	}
}

// HasAnyPerks returns true if the squad has any perks equipped.
func HasAnyPerks(squadID ecs.EntityID, manager *common.EntityManager) bool {
	slotData := common.GetComponentTypeByID[*PerkSlotData](manager, squadID, PerkSlotComponent)
	return slotData != nil && len(slotData.PerkIDs) > 0
}

// ResetPerkRoundStateTurn resets shared tracking fields at the start of each turn.
// Called before TurnStartHooks run. Per-perk state is NOT reset here —
// perks manage their own per-turn state in their TurnStart hooks.
func ResetPerkRoundStateTurn(s *PerkRoundState) {
	// Snapshot previous turn state for Counterpunch/Deadshot before clearing
	s.WasAttackedLastTurn = s.WasAttackedThisTurn
	s.DidNotAttackLastTurn = !s.AttackedThisTurn
	s.WasIdleLastTurn = !s.MovedThisTurn && !s.AttackedThisTurn

	s.MovedThisTurn = false
	s.AttackedThisTurn = false
	s.WasAttackedThisTurn = false
}

// ResetPerkRoundStateRound clears all per-perk round state at the start of each round.
// Per-battle state (PerkBattleState) is preserved.
func ResetPerkRoundStateRound(s *PerkRoundState) {
	s.PerkState = nil
}

// ========================================
// HOOK RUNNER FUNCTIONS
// ========================================

// RunAttackerDamageModHooks runs AttackerDamageMod hooks for the attacker's perks.
func RunAttackerDamageModHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, manager *common.EntityManager) {
	ctx := buildCombatContext(attackerSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID, manager)
	if ctx == nil {
		return
	}
	forEachPerkHook(attackerSquadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.AttackerDamageMod != nil {
			hooks.AttackerDamageMod(ctx, modifiers)
		}
		return true
	})
}

// RunDefenderDamageModHooks runs DefenderDamageMod hooks for the defender's perks.
func RunDefenderDamageModHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, manager *common.EntityManager) {
	ctx := buildCombatContext(defenderSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID, manager)
	if ctx == nil {
		return
	}
	forEachPerkHook(defenderSquadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.DefenderDamageMod != nil {
			hooks.DefenderDamageMod(ctx, modifiers)
		}
		return true
	})
}

// RunAttackerPostDamageHooks runs post-damage hooks for the attacker's perks.
func RunAttackerPostDamageHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	ctx := buildCombatContext(attackerSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID, manager)
	if ctx == nil {
		return
	}
	forEachPerkHook(attackerSquadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.AttackerPostDamage != nil {
			hooks.AttackerPostDamage(ctx, damageDealt, wasKill)
		}
		return true
	})
}

// RunDefenderPostDamageHooks runs post-damage hooks for the defender's perks.
func RunDefenderPostDamageHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	ctx := buildCombatContext(defenderSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID, manager)
	if ctx == nil {
		return
	}
	forEachPerkHook(defenderSquadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.DefenderPostDamage != nil {
			hooks.DefenderPostDamage(ctx, damageDealt, wasKill)
		}
		return true
	})
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
	forEachPerkHook(attackerSquadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.TargetOverride != nil {
			targets = hooks.TargetOverride(ctx, targets)
		}
		return true
	})
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
	skip := false
	forEachPerkHook(defenderSquadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.CounterMod != nil && hooks.CounterMod(ctx, modifiers) {
			skip = true
			return false
		}
		return true
	})
	return skip
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
	forEachPerkHook(squadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.TurnStart != nil {
			hooks.TurnStart(ctx)
		}
		return true
	})
}

// RunCoverModHooks runs cover modification hooks for defender perks.
func RunCoverModHooks(attackerID, defenderID ecs.EntityID,
	coverBreakdown *combatcore.CoverBreakdown, manager *common.EntityManager) {
	attackerSquadID := getSquadIDForUnit(attackerID, manager)
	defenderSquadID := getSquadIDForUnit(defenderID, manager)
	ctx := buildCombatContext(defenderSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID, manager)
	if ctx == nil {
		return
	}
	forEachPerkHook(defenderSquadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.DefenderCoverMod != nil {
			hooks.DefenderCoverMod(ctx, coverBreakdown)
		}
		return true
	})
}

// RunDeathOverrideHooks checks if lethal damage should be prevented.
func RunDeathOverrideHooks(unitID, squadID ecs.EntityID, manager *common.EntityManager) bool {
	ctx := buildHookContext(squadID, manager)
	if ctx == nil {
		return false
	}
	ctx.UnitID = unitID
	ctx.SquadID = squadID
	prevented := false
	forEachPerkHook(squadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.DeathOverride != nil && hooks.DeathOverride(ctx) {
			prevented = true
			return false
		}
		return true
	})
	return prevented
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
	reducedDmg, redirectTarget, redirectAmt := damageAmount, ecs.EntityID(0), 0
	forEachPerkHook(defenderSquadID, manager, func(perkID string, hooks *PerkHooks) bool {
		if hooks.DamageRedirect != nil {
			rd, rt, ra := hooks.DamageRedirect(ctx)
			if rt != 0 {
				reducedDmg, redirectTarget, redirectAmt = rd, rt, ra
				return false
			}
		}
		return true
	})
	return reducedDmg, redirectTarget, redirectAmt
}
