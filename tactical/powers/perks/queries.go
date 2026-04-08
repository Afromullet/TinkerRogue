package perks

import (
	"game_main/common"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// GetEquippedPerkIDs returns all perk IDs equipped on a squad (public accessor for GUI).
func GetEquippedPerkIDs(squadID ecs.EntityID, manager *common.EntityManager) []string {
	return getActivePerkIDs(squadID, manager)
}

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

// buildCombatContext constructs a HookContext with attacker/defender fields populated.
// ownerSquadID determines whose perks will be iterated.
func buildCombatContext(ownerSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	manager *common.EntityManager) *HookContext {
	ctx := buildHookContext(ownerSquadID, manager)
	if ctx == nil {
		return nil
	}
	ctx.AttackerID = attackerID
	ctx.DefenderID = defenderID
	ctx.AttackerSquadID = attackerSquadID
	ctx.DefenderSquadID = defenderSquadID
	return ctx
}

// forEachPerkHook iterates over active perks for ownerSquadID, calling fn
// for each registered PerkHooks. If fn returns false, iteration stops early.
func forEachPerkHook(ownerSquadID ecs.EntityID, manager *common.EntityManager,
	fn func(perkID string, hooks *PerkHooks) bool) {
	for _, perkID := range getActivePerkIDs(ownerSquadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks == nil {
			continue
		}
		if !fn(perkID, hooks) {
			return
		}
	}
}

