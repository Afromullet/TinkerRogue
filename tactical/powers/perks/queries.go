package perks

import (
	"game_main/common"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// GetEquippedPerkIDs returns all perk IDs equipped on a squad, or nil if none.
func GetEquippedPerkIDs(squadID ecs.EntityID, manager *common.EntityManager) []PerkID {
	if data := common.GetComponentTypeByID[*PerkSlotData](
		manager, squadID, PerkSlotComponent,
	); data != nil {
		return data.PerkIDs
	}
	return nil
}

// HasPerk checks if a squad has a specific perk equipped.
func HasPerk(squadID ecs.EntityID, perkID PerkID, manager *common.EntityManager) bool {
	for _, id := range GetEquippedPerkIDs(squadID, manager) {
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


// forEachPerkBehavior iterates over active perks for ownerSquadID, calling fn
// for each registered PerkBehavior. If fn returns false, iteration stops early.
func forEachPerkBehavior(ownerSquadID ecs.EntityID, manager *common.EntityManager,
	fn func(PerkBehavior) bool) {
	for _, perkID := range GetEquippedPerkIDs(ownerSquadID, manager) {
		behavior := GetPerkBehavior(perkID)
		if behavior == nil {
			continue
		}
		if !fn(behavior) {
			return
		}
	}
}
