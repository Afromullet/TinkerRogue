package perks

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

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
