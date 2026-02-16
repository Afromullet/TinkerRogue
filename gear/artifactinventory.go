package gear

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// NewArtifactInventory creates a new empty artifact inventory with the given capacity.
func NewArtifactInventory(maxArtifacts int) *ArtifactInventoryData {
	return &ArtifactInventoryData{
		OwnedArtifacts: make(map[string][]*ArtifactInstance),
		MaxArtifacts:   maxArtifacts,
	}
}

// totalInstanceCount returns the total number of artifact instances across all definition IDs.
func (inv *ArtifactInventoryData) totalInstanceCount() int {
	count := 0
	for _, instances := range inv.OwnedArtifacts {
		count += len(instances)
	}
	return count
}

// CanAddArtifact returns true if the inventory has room for another artifact instance.
func (inv *ArtifactInventoryData) CanAddArtifact() bool {
	return inv.totalInstanceCount() < inv.MaxArtifacts
}

// AddArtifact adds an artifact instance to the inventory as available (not equipped).
// Multiple copies of the same artifact are allowed.
func (inv *ArtifactInventoryData) AddArtifact(artifactID string) error {
	if !inv.CanAddArtifact() {
		return fmt.Errorf("inventory full (%d/%d)", inv.totalInstanceCount(), inv.MaxArtifacts)
	}
	inv.OwnedArtifacts[artifactID] = append(inv.OwnedArtifacts[artifactID], &ArtifactInstance{EquippedOn: 0})
	return nil
}

// RemoveArtifact removes the first unequipped instance of the given artifact.
func (inv *ArtifactInventoryData) RemoveArtifact(artifactID string) error {
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
func (inv *ArtifactInventoryData) MarkArtifactEquipped(artifactID string, squadID ecs.EntityID) error {
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
func (inv *ArtifactInventoryData) MarkArtifactAvailable(artifactID string, squadID ecs.EntityID) error {
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

// OwnsArtifact returns true if the artifact is in the inventory (any instances).
func (inv *ArtifactInventoryData) OwnsArtifact(artifactID string) bool {
	instances, exists := inv.OwnedArtifacts[artifactID]
	return exists && len(instances) > 0
}

// IsArtifactAvailable returns true if any instance of the artifact is not equipped.
func (inv *ArtifactInventoryData) IsArtifactAvailable(artifactID string) bool {
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

// GetAvailableArtifacts returns unique definition IDs that have at least one available instance.
func (inv *ArtifactInventoryData) GetAvailableArtifacts() []string {
	var result []string
	for id, instances := range inv.OwnedArtifacts {
		for _, inst := range instances {
			if inst.EquippedOn == 0 {
				result = append(result, id)
				break
			}
		}
	}
	return result
}

// GetEquippedArtifacts returns unique definition IDs that have at least one equipped instance.
func (inv *ArtifactInventoryData) GetEquippedArtifacts() []string {
	var result []string
	for id, instances := range inv.OwnedArtifacts {
		for _, inst := range instances {
			if inst.EquippedOn != 0 {
				result = append(result, id)
				break
			}
		}
	}
	return result
}

// GetArtifactCount returns (total instance count, max capacity).
func (inv *ArtifactInventoryData) GetArtifactCount() (int, int) {
	return inv.totalInstanceCount(), inv.MaxArtifacts
}

// GetSquadWithArtifact returns the first squad ID that has the artifact equipped, or 0 if none.
func (inv *ArtifactInventoryData) GetSquadWithArtifact(artifactID string) ecs.EntityID {
	instances, exists := inv.OwnedArtifacts[artifactID]
	if !exists {
		return 0
	}
	for _, inst := range instances {
		if inst.EquippedOn != 0 {
			return inst.EquippedOn
		}
	}
	return 0
}

// GetAllInstances returns a flat list of all artifact instances for GUI display.
func (inv *ArtifactInventoryData) GetAllInstances() []ArtifactInstanceInfo {
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
func (inv *ArtifactInventoryData) GetInstanceCount(artifactID string) int {
	return len(inv.OwnedArtifacts[artifactID])
}

// GetPlayerArtifactInventory returns the ArtifactInventoryData for the given player entity.
func GetPlayerArtifactInventory(playerID ecs.EntityID, manager *common.EntityManager) *ArtifactInventoryData {
	return common.GetComponentTypeByID[*ArtifactInventoryData](manager, playerID, ArtifactInventoryComponent)
}

