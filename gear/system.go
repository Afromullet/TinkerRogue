package gear

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/effects"
	"game_main/tactical/squads"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// EquipArtifact equips an artifact on a squad. The player must own the artifact
// and it must be available (not equipped elsewhere). The squad must have a free slot.
func EquipArtifact(playerID, squadID ecs.EntityID, artifactID string, manager *common.EntityManager) error {
	def := templates.GetArtifactDefinition(artifactID)
	if def == nil {
		return fmt.Errorf("artifact %q not found in registry", artifactID)
	}

	// Verify player owns the artifact and it's available
	inventory := GetPlayerArtifactInventory(playerID, manager)
	if inventory == nil {
		return fmt.Errorf("player %d has no artifact inventory", playerID)
	}
	if !inventory.IsArtifactAvailable(artifactID) {
		if !inventory.OwnsArtifact(artifactID) {
			return fmt.Errorf("player does not own artifact %q", artifactID)
		}
		return fmt.Errorf("artifact %q is already equipped on another squad", artifactID)
	}

	entity := manager.FindEntityByID(squadID)
	if entity == nil {
		return fmt.Errorf("squad %d not found", squadID)
	}

	// Get or create equipment component
	var data *EquipmentData
	if entity.HasComponent(EquipmentComponent) {
		data = common.GetComponentType[*EquipmentData](entity, EquipmentComponent)
	} else {
		data = &EquipmentData{}
		entity.AddComponent(EquipmentComponent, data)
	}

	// Check slot availability
	if len(data.EquippedArtifacts) >= MaxArtifactSlots {
		return fmt.Errorf("all %d artifact slots are occupied", MaxArtifactSlots)
	}

	// Check not already equipped on this squad
	for _, equipped := range data.EquippedArtifacts {
		if equipped == artifactID {
			return fmt.Errorf("artifact %q is already equipped on this squad", artifactID)
		}
	}

	data.EquippedArtifacts = append(data.EquippedArtifacts, artifactID)

	// Mark as equipped in inventory
	if err := inventory.MarkArtifactEquipped(artifactID, squadID); err != nil {
		// Rollback: remove the artifact we just appended
		data.EquippedArtifacts = data.EquippedArtifacts[:len(data.EquippedArtifacts)-1]
		return fmt.Errorf("failed to mark artifact equipped: %w", err)
	}

	return nil
}

// UnequipArtifact removes a specific artifact from a squad and returns it to inventory.
func UnequipArtifact(playerID, squadID ecs.EntityID, artifactID string, manager *common.EntityManager) error {
	data := GetEquipmentData(squadID, manager)
	if data == nil {
		return fmt.Errorf("squad %d has no equipment component", squadID)
	}

	// Find and remove the artifact from the slice
	found := false
	for i, equipped := range data.EquippedArtifacts {
		if equipped == artifactID {
			data.EquippedArtifacts = append(data.EquippedArtifacts[:i], data.EquippedArtifacts[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("artifact %q is not equipped on squad %d", artifactID, squadID)
	}

	// Return to inventory
	inventory := GetPlayerArtifactInventory(playerID, manager)
	if inventory == nil {
		// Rollback: re-add the artifact
		data.EquippedArtifacts = append(data.EquippedArtifacts, artifactID)
		return fmt.Errorf("player %d has no artifact inventory", playerID)
	}
	if err := inventory.MarkArtifactAvailable(artifactID, squadID); err != nil {
		// Rollback: re-add the artifact
		data.EquippedArtifacts = append(data.EquippedArtifacts, artifactID)
		return fmt.Errorf("failed to return artifact to inventory: %w", err)
	}

	return nil
}

// ApplyArtifactStatEffects applies permanent stat effects from all equipped artifacts
// (of any tier) to all units in the given squads.
// Should be called at battle start before turn initialization.
func ApplyArtifactStatEffects(squadIDs []ecs.EntityID, manager *common.EntityManager) {
	for _, squadID := range squadIDs {
		defs := GetArtifactDefinitions(squadID, manager)
		if len(defs) == 0 {
			continue
		}

		unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
		if len(unitIDs) == 0 {
			continue
		}

		for _, def := range defs {
			for _, mod := range def.StatModifiers {
				effect := effects.ActiveEffect{
					Name:           def.Name,
					Source:         effects.SourceItem,
					Stat:           effects.ParseStatType(mod.Stat),
					Modifier:       mod.Modifier,
					RemainingTurns: -1, // Permanent
				}
				effects.ApplyEffectToUnits(unitIDs, effect, manager)
			}
		}
	}
}
