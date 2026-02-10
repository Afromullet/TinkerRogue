package node

import (
	"math"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// GetNodeAt returns the EntityID of any unified overworld node at the given position.
// Returns 0 if no node exists at the position.
func GetNodeAt(manager *common.EntityManager, pos coords.LogicalPosition) ecs.EntityID {
	entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(pos)
	for _, entityID := range entityIDs {
		if manager.HasComponent(entityID, core.OverworldNodeComponent) {
			return entityID
		}
	}
	return 0
}

// IsAnyNodeAtPosition checks if any unified overworld node exists at the given position.
func IsAnyNodeAtPosition(manager *common.EntityManager, pos coords.LogicalPosition) bool {
	return GetNodeAt(manager, pos) != 0
}

// CountNodesByOwner returns the number of nodes owned by a specific owner.
func CountNodesByOwner(manager *common.EntityManager, ownerID string) int {
	count := 0
	for _, result := range manager.World.Query(core.OverworldNodeTag) {
		data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
		if data != nil && data.OwnerID == ownerID {
			count++
		}
	}
	return count
}

// CountNodesByCategory returns the number of nodes in a specific category.
func CountNodesByCategory(manager *common.EntityManager, category core.NodeCategory) int {
	count := 0
	for _, result := range manager.World.Query(core.OverworldNodeTag) {
		data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
		if data != nil && data.Category == category {
			count++
		}
	}
	return count
}

// GetNodePositionsByOwner returns all positions of nodes owned by a specific owner.
func GetNodePositionsByOwner(manager *common.EntityManager, ownerID string) []coords.LogicalPosition {
	var positions []coords.LogicalPosition
	for _, result := range manager.World.Query(core.OverworldNodeTag) {
		data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
		if data == nil || data.OwnerID != ownerID {
			continue
		}
		pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
		if pos != nil {
			positions = append(positions, *pos)
		}
	}
	return positions
}

// GetNearestNodeDistance returns the Euclidean distance to the nearest node owned by ownerID.
// Returns math.MaxFloat64 if no matching nodes exist.
func GetNearestNodeDistance(manager *common.EntityManager, pos coords.LogicalPosition, ownerID string) float64 {
	nearest := math.MaxFloat64
	for _, result := range manager.World.Query(core.OverworldNodeTag) {
		data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
		if data == nil || data.OwnerID != ownerID {
			continue
		}
		nodePos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
		if nodePos == nil {
			continue
		}
		dx := float64(pos.X - nodePos.X)
		dy := float64(pos.Y - nodePos.Y)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < nearest {
			nearest = dist
		}
	}
	return nearest
}

// CountHighIntensityNodes returns the number of nodes at or above the intensity threshold.
func CountHighIntensityNodes(manager *common.EntityManager, threshold int) int {
	count := 0
	for _, result := range manager.World.Query(core.OverworldNodeTag) {
		data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
		if data != nil && data.Intensity >= threshold {
			count++
		}
	}
	return count
}

// CalculateAverageIntensity returns the average intensity of all nodes with intensity > 0.
func CalculateAverageIntensity(manager *common.EntityManager) float64 {
	totalIntensity := 0
	count := 0
	for _, result := range manager.World.Query(core.OverworldNodeTag) {
		data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
		if data != nil && data.Category == core.NodeCategoryThreat {
			totalIntensity += data.Intensity
			count++
		}
	}
	if count == 0 {
		return 0.0
	}
	return float64(totalIntensity) / float64(count)
}
