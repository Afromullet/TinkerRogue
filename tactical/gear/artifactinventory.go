package gear

import (
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
func totalInstanceCount(inv *ArtifactInventoryData) int {
	count := 0
	for _, instances := range inv.OwnedArtifacts {
		count += len(instances)
	}
	return count
}

// GetPlayerArtifactInventory returns the ArtifactInventoryData for the given player entity.
func GetPlayerArtifactInventory(playerID ecs.EntityID, manager *common.EntityManager) *ArtifactInventoryData {
	return common.GetComponentTypeByID[*ArtifactInventoryData](manager, playerID, ArtifactInventoryComponent)
}
