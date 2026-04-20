package squadcore

import (
	"fmt"
	"game_main/core/common"

	"github.com/bytearena/ecs"
)

// EquipAbilityToLeader -

func EquipAbilityToLeader(
	leaderEntityID ecs.EntityID,
	slotIndex int,
	abilityType AbilityType,
	triggerType TriggerType,
	threshold float64,
	ecsmanager *common.EntityManager,
) error {

	if slotIndex < 0 || slotIndex >= 4 {
		return fmt.Errorf("invalid slot %d", slotIndex)
	}

	leaderEntity := ecsmanager.FindEntityByID(leaderEntityID)
	if leaderEntity == nil {
		return fmt.Errorf("leader entity not found")
	}

	if !leaderEntity.HasComponent(LeaderComponent) {
		return fmt.Errorf("entity is not a leader")
	}

	// Get ability params
	params := GetAbilityParams(abilityType)

	// Update ability slot
	abilityData := common.GetComponentType[*AbilitySlotData](leaderEntity, AbilitySlotComponent)
	if abilityData == nil {
		return fmt.Errorf("leader missing ability slot component")
	}

	abilityData.Slots[slotIndex] = AbilitySlot{
		AbilityType:  abilityType,
		TriggerType:  triggerType,
		Threshold:    threshold,
		HasTriggered: false,
		IsEquipped:   true,
	}

	// Update cooldown tracker
	cooldownData := common.GetComponentType[*CooldownTrackerData](leaderEntity, CooldownTrackerComponent)
	if cooldownData != nil {
		cooldownData.MaxCooldowns[slotIndex] = params.BaseCooldown
		cooldownData.Cooldowns[slotIndex] = 0
	}

	return nil
}
