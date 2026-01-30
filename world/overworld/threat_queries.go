package overworld

import (
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// GetThreatNodeAt returns the EntityID of a threat node at a specific position.
// Returns 0 if no threat exists at the position.
// Prefer using queries directly when iterating over multiple threats.
func GetThreatNodeAt(manager *common.EntityManager, pos coords.LogicalPosition) ecs.EntityID {
	entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(pos)
	for _, entityID := range entityIDs {
		if manager.HasComponent(entityID, ThreatNodeComponent) {
			return entityID
		}
	}
	return 0
}

// CountThreatNodes returns the total number of threat nodes
func CountThreatNodes(manager *common.EntityManager) int {
	count := 0
	for range manager.World.Query(ThreatNodeTag) {
		count++
	}
	return count
}

// CountHighIntensityThreats returns the number of threats at or above the threshold
func CountHighIntensityThreats(manager *common.EntityManager, threshold int) int {
	count := 0
	for _, result := range manager.World.Query(ThreatNodeTag) {
		threatData := common.GetComponentType[*ThreatNodeData](result.Entity, ThreatNodeComponent)
		if threatData != nil && threatData.Intensity >= threshold {
			count++
		}
	}
	return count
}

// CalculateAverageIntensity returns the average intensity of all threats
func CalculateAverageIntensity(manager *common.EntityManager) float64 {
	totalIntensity := 0
	count := 0

	for _, result := range manager.World.Query(ThreatNodeTag) {
		data := common.GetComponentType[*ThreatNodeData](result.Entity, ThreatNodeComponent)
		if data != nil {
			totalIntensity += data.Intensity
			count++
		}
	}

	if count == 0 {
		return 0.0
	}
	return float64(totalIntensity) / float64(count)
}
