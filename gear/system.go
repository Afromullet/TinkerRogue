package gear

import (
	"fmt"
	"game_main/common"
	"game_main/config"
	"game_main/tactical/effects"
	"game_main/tactical/squads"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// --- ArtifactInventoryData mutation functions ---

// AddArtifactToInventory adds an artifact instance to the inventory as available (not equipped).
// Multiple copies of the same artifact are allowed.
func AddArtifactToInventory(inv *ArtifactInventoryData, artifactID string) error {
	if !CanAddArtifact(inv) {
		return fmt.Errorf("inventory full (%d/%d)", totalInstanceCount(inv), inv.MaxArtifacts)
	}
	inv.OwnedArtifacts[artifactID] = append(inv.OwnedArtifacts[artifactID], &ArtifactInstance{EquippedOn: 0})
	return nil
}

// RemoveArtifactFromInventory removes the first unequipped instance of the given artifact.
func RemoveArtifactFromInventory(inv *ArtifactInventoryData, artifactID string) error {
	instances, exists := inv.OwnedArtifacts[artifactID]
	if !exists || len(instances) == 0 {
		return fmt.Errorf("artifact %q not in inventory", artifactID)
	}
	for i, inst := range instances {
		if inst.EquippedOn == 0 {
			inv.OwnedArtifacts[artifactID] = append(instances[:i], instances[i+1:]...)
			if len(inv.OwnedArtifacts[artifactID]) == 0 {
				delete(inv.OwnedArtifacts, artifactID)
			}
			return nil
		}
	}
	return fmt.Errorf("artifact %q has no unequipped instances, unequip first", artifactID)
}

// MarkArtifactEquipped marks the first available instance of an artifact as equipped on a squad.
func MarkArtifactEquipped(inv *ArtifactInventoryData, artifactID string, squadID ecs.EntityID) error {
	instances, exists := inv.OwnedArtifacts[artifactID]
	if !exists || len(instances) == 0 {
		return fmt.Errorf("artifact %q not in inventory", artifactID)
	}
	for _, inst := range instances {
		if inst.EquippedOn == 0 {
			inst.EquippedOn = squadID
			return nil
		}
	}
	return fmt.Errorf("artifact %q has no available instances", artifactID)
}

// MarkArtifactAvailable marks the instance of an artifact equipped on the given squad as available.
func MarkArtifactAvailable(inv *ArtifactInventoryData, artifactID string, squadID ecs.EntityID) error {
	instances, exists := inv.OwnedArtifacts[artifactID]
	if !exists || len(instances) == 0 {
		return fmt.Errorf("artifact %q not in inventory", artifactID)
	}
	for _, inst := range instances {
		if inst.EquippedOn == squadID {
			inst.EquippedOn = 0
			return nil
		}
	}
	return fmt.Errorf("artifact %q not equipped on squad %d", artifactID, squadID)
}

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
	if !IsArtifactAvailable(inventory, artifactID) {
		if !OwnsArtifact(inventory, artifactID) {
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
	if len(data.EquippedArtifacts) >= config.DefaultMaxArtifactsPerCommander {
		return fmt.Errorf("all %d artifact slots are occupied", config.DefaultMaxArtifactsPerCommander)
	}

	// Check not already equipped on this squad
	for _, equipped := range data.EquippedArtifacts {
		if equipped == artifactID {
			return fmt.Errorf("artifact %q is already equipped on this squad", artifactID)
		}
	}

	data.EquippedArtifacts = append(data.EquippedArtifacts, artifactID)

	// Mark as equipped in inventory
	if err := MarkArtifactEquipped(inventory, artifactID, squadID); err != nil {
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
	if err := MarkArtifactAvailable(inventory, artifactID, squadID); err != nil {
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
