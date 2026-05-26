package artifacts

import (
	"fmt"
	"game_main/core/common"
	"game_main/tactical/powers/effects"
	"game_main/tactical/squads/squadcore"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// defaultMaxArtifactSlots returns the configured default slot count for a
// commander's equipment. Used when an EquipmentData is created without an
// explicit MaxSlots value, and as a fall-back for older save files that lack
// the field.
func defaultMaxArtifactSlots() int {
	return templates.GameConfig.Player.Limits.MaxArtifactsPerCommander
}

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
		data = &EquipmentData{MaxSlots: defaultMaxArtifactSlots()}
		entity.AddComponent(EquipmentComponent, data)
	}

	// Check slot availability. MaxSlots == 0 means an older component (or save)
	// that pre-dates the field; backfill from config so the limit is enforced.
	if data.MaxSlots == 0 {
		data.MaxSlots = defaultMaxArtifactSlots()
	}
	if len(data.EquippedArtifacts) >= data.MaxSlots {
		return fmt.Errorf("all %d artifact slots are occupied", data.MaxSlots)
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

// ApplyArtifactStatEffects applies permanent stat effects from all equipped
// artifacts (of any tier) to all units in the given squads. Called at battle
// start by CombatService.InitializeCombat, before turn initialization.
//
// Inverse: there is no artifact-specific remove. Cleanup happens at battle
// exit via CombatService.cleanupEffects, which calls effects.RemoveAllEffects
// per surviving player unit (combat_service.go:267). That removes ALL active
// effects (artifacts + spells) uniformly — both layers attach as ActiveEffect
// entries on the unit, so a single bulk-remove is the correct inverse.
func ApplyArtifactStatEffects(squadIDs []ecs.EntityID, manager *common.EntityManager) {
	for _, squadID := range squadIDs {
		defs := GetArtifactDefinitions(squadID, manager)
		if len(defs) == 0 {
			continue
		}

		unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
		if len(unitIDs) == 0 {
			continue
		}

		for _, def := range defs {
			mods := make([]effects.StatModifier, len(def.StatModifiers))
			for i, m := range def.StatModifiers {
				mods[i] = effects.StatModifier{Stat: m.Stat, Modifier: m.Modifier}
			}
			effects.ApplyStatModifiers(unitIDs, def.Name, mods, effects.SourceItem, -1, manager)
		}
	}
}
