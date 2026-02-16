package gear

import (
	"game_main/common"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// GetEquipmentData returns the EquipmentData for a squad, or nil if none.
func GetEquipmentData(squadID ecs.EntityID, manager *common.EntityManager) *EquipmentData {
	return common.GetComponentTypeByID[*EquipmentData](manager, squadID, EquipmentComponent)
}

// hasArtifactOfTier returns true if the squad has any equipped artifact of the given tier.
func hasArtifactOfTier(squadID ecs.EntityID, tier string, manager *common.EntityManager) bool {
	data := GetEquipmentData(squadID, manager)
	if data == nil {
		return false
	}
	for _, id := range data.EquippedArtifacts {
		def := templates.GetArtifactDefinition(id)
		if def != nil && def.Tier == tier {
			return true
		}
	}
	return false
}

// GetArtifactDefinitions returns artifact definitions for all equipped artifacts on a squad.
func GetArtifactDefinitions(squadID ecs.EntityID, manager *common.EntityManager) []*templates.ArtifactDefinition {
	data := GetEquipmentData(squadID, manager)
	if data == nil {
		return nil
	}
	var defs []*templates.ArtifactDefinition
	for _, id := range data.EquippedArtifacts {
		def := templates.GetArtifactDefinition(id)
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}

// HasArtifactBehavior returns true if any equipped artifact on the squad has the given behavior.
func HasArtifactBehavior(squadID ecs.EntityID, behavior string, manager *common.EntityManager) bool {
	data := GetEquipmentData(squadID, manager)
	if data == nil {
		return false
	}
	for _, id := range data.EquippedArtifacts {
		def := templates.GetArtifactDefinition(id)
		if def != nil && def.Behavior == behavior {
			return true
		}
	}
	return false
}

// HasBehaviorInFaction returns true if any squad in the list has an artifact with the given behavior.
func HasBehaviorInFaction(squadIDs []ecs.EntityID, behavior string, manager *common.EntityManager) bool {
	return GetFactionSquadWithBehavior(squadIDs, behavior, manager) != 0
}

// GetFactionSquadWithBehavior returns the first squad with the given artifact behavior, or 0.
func GetFactionSquadWithBehavior(squadIDs []ecs.EntityID, behavior string, manager *common.EntityManager) ecs.EntityID {
	for _, sid := range squadIDs {
		if HasArtifactBehavior(sid, behavior, manager) {
			return sid
		}
	}
	return 0
}

// --- ArtifactInventoryData query functions (read-only) ---

// CanAddArtifact returns true if the inventory has room for another artifact instance.
func CanAddArtifact(inv *ArtifactInventoryData) bool {
	return totalInstanceCount(inv) < inv.MaxArtifacts
}

// OwnsArtifact returns true if the artifact is in the inventory (any instances).
func OwnsArtifact(inv *ArtifactInventoryData, artifactID string) bool {
	instances, exists := inv.OwnedArtifacts[artifactID]
	return exists && len(instances) > 0
}

// IsArtifactAvailable returns true if any instance of the artifact is not equipped.
func IsArtifactAvailable(inv *ArtifactInventoryData, artifactID string) bool {
	instances, exists := inv.OwnedArtifacts[artifactID]
	if !exists {
		return false
	}
	for _, inst := range instances {
		if inst.EquippedOn == 0 {
			return true
		}
	}
	return false
}

// GetArtifactCount returns (total instance count, max capacity).
func GetArtifactCount(inv *ArtifactInventoryData) (int, int) {
	return totalInstanceCount(inv), inv.MaxArtifacts
}

// GetAllInstances returns a flat list of all artifact instances for GUI display.
func GetAllInstances(inv *ArtifactInventoryData) []ArtifactInstanceInfo {
	var result []ArtifactInstanceInfo
	for defID, instances := range inv.OwnedArtifacts {
		for i, inst := range instances {
			result = append(result, ArtifactInstanceInfo{
				DefinitionID:  defID,
				EquippedOn:    inst.EquippedOn,
				InstanceIndex: i + 1,
			})
		}
	}
	return result
}

// GetInstanceCount returns the number of instances of a specific artifact.
func GetInstanceCount(inv *ArtifactInventoryData, artifactID string) int {
	return len(inv.OwnedArtifacts[artifactID])
}
