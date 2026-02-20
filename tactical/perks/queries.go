package perks

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// getActivePerkIDs returns all perk IDs active for a unit: unit perks + parent squad perks.
func getActivePerkIDs(unitID ecs.EntityID, manager *common.EntityManager) []string {
	var ids []string

	// Unit perks
	if data := common.GetComponentTypeByID[*UnitPerkData](manager, unitID, UnitPerkComponent); data != nil {
		for _, id := range data.EquippedPerks {
			if id != "" {
				ids = append(ids, id)
			}
		}
	}

	// Squad perks (from parent squad)
	if memberData := common.GetComponentTypeByID[*squads.SquadMemberData](manager, unitID, squads.SquadMemberComponent); memberData != nil {
		ids = append(ids, getSquadPerkIDs(memberData.SquadID, manager)...)
	}

	return ids
}

// getSquadPerkIDs returns perk IDs equipped on a squad entity.
func getSquadPerkIDs(squadID ecs.EntityID, manager *common.EntityManager) []string {
	var ids []string
	if data := common.GetComponentTypeByID[*SquadPerkData](manager, squadID, SquadPerkComponent); data != nil {
		for _, id := range data.EquippedPerks {
			if id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// HasPerk checks if an entity has a specific perk equipped (unit or squad level).
func HasPerk(entityID ecs.EntityID, perkID string, manager *common.EntityManager) bool {
	for _, id := range getActivePerkIDs(entityID, manager) {
		if id == perkID {
			return true
		}
	}
	return false
}

// GetEquippedPerks returns all equipped perk IDs for an entity.
// Checks unit, squad, and commander perk components.
func GetEquippedPerks(entityID ecs.EntityID, manager *common.EntityManager) []string {
	var ids []string

	if data := common.GetComponentTypeByID[*UnitPerkData](manager, entityID, UnitPerkComponent); data != nil {
		for _, id := range data.EquippedPerks {
			if id != "" {
				ids = append(ids, id)
			}
		}
	}

	if data := common.GetComponentTypeByID[*SquadPerkData](manager, entityID, SquadPerkComponent); data != nil {
		for _, id := range data.EquippedPerks {
			if id != "" {
				ids = append(ids, id)
			}
		}
	}

	if data := common.GetComponentTypeByID[*CommanderPerkData](manager, entityID, CommanderPerkComponent); data != nil {
		for _, id := range data.EquippedPerks {
			if id != "" {
				ids = append(ids, id)
			}
		}
	}

	return ids
}

// ========================================
// HOOK RUNNER FUNCTIONS
// ========================================

// RunDamageModHooks runs all DamageMod hooks for an attacker's perks.
func RunDamageModHooks(attackerID, defenderID ecs.EntityID,
	modifiers *squads.DamageModifiers, manager *common.EntityManager) {
	for _, perkID := range getActivePerkIDs(attackerID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DamageMod != nil {
			hooks.DamageMod(attackerID, defenderID, modifiers, manager)
		}
	}
}

// RunDefenderDamageModHooks runs DamageMod hooks for the DEFENDER's perks.
// Used for perks like Stone Wall that reduce damage taken.
func RunDefenderDamageModHooks(attackerID, defenderID ecs.EntityID,
	modifiers *squads.DamageModifiers, manager *common.EntityManager) {
	for _, perkID := range getActivePerkIDs(defenderID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.DamageMod != nil {
			hooks.DamageMod(attackerID, defenderID, modifiers, manager)
		}
	}
}

// RunTargetOverrideHooks applies target overrides from attacker perks.
func RunTargetOverrideHooks(attackerID, defenderSquadID ecs.EntityID,
	targets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	for _, perkID := range getActivePerkIDs(attackerID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.TargetOverride != nil {
			targets = hooks.TargetOverride(attackerID, defenderSquadID, targets, manager)
		}
	}
	return targets
}

// RunCounterModHooks checks if counterattack should be suppressed or modified.
// Returns true if the counterattack should be skipped entirely.
func RunCounterModHooks(defenderID, attackerID ecs.EntityID,
	modifiers *squads.DamageModifiers, manager *common.EntityManager) bool {
	for _, perkID := range getActivePerkIDs(defenderID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.CounterMod != nil {
			if hooks.CounterMod(defenderID, attackerID, modifiers, manager) {
				return true // Skip counter
			}
		}
	}
	return false
}

// RunPostDamageHooks runs post-damage hooks for the attacker.
// Note: runs with pre-damage HP. Use damageDealt/wasKill params.
func RunPostDamageHooks(attackerID, defenderID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	for _, perkID := range getActivePerkIDs(attackerID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.PostDamage != nil {
			hooks.PostDamage(attackerID, defenderID, damageDealt, wasKill, manager)
		}
	}
}

// RunTurnStartHooks runs turn-start hooks for all units in a squad and squad-level perks.
func RunTurnStartHooks(squadID ecs.EntityID, manager *common.EntityManager) {
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	for _, unitID := range unitIDs {
		for _, perkID := range getActivePerkIDs(unitID, manager) {
			hooks := GetPerkHooks(perkID)
			if hooks != nil && hooks.TurnStart != nil {
				hooks.TurnStart(squadID, manager)
			}
		}
	}
	// Also run squad-level turn start hooks
	for _, perkID := range getSquadPerkIDs(squadID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.TurnStart != nil {
			hooks.TurnStart(squadID, manager)
		}
	}
}

// RunCoverModHooks runs cover modification hooks for the attacker's perks.
func RunCoverModHooks(attackerID, defenderID ecs.EntityID,
	coverBreakdown *squads.CoverBreakdown, manager *common.EntityManager) {
	for _, perkID := range getActivePerkIDs(attackerID, manager) {
		hooks := GetPerkHooks(perkID)
		if hooks != nil && hooks.CoverMod != nil {
			hooks.CoverMod(attackerID, defenderID, coverBreakdown, manager)
		}
	}
}
