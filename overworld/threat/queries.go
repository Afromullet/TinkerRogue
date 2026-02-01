package threat

import (
	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// GetThreatNodeAt returns the EntityID of a threat node at a specific position.
// Returns 0 if no threat exists at the position.
// Prefer using queries directly when iterating over multiple threats.
func GetThreatNodeAt(manager *common.EntityManager, pos coords.LogicalPosition) ecs.EntityID {
	return core.GetThreatNodeAt(manager, pos)
}

// CountThreatNodes returns the total number of threat nodes
func CountThreatNodes(manager *common.EntityManager) int {
	count := 0
	for range manager.World.Query(core.ThreatNodeTag) {
		count++
	}
	return count
}

// CountHighIntensityThreats returns the number of threats at or above the threshold
func CountHighIntensityThreats(manager *common.EntityManager, threshold int) int {
	count := 0
	for _, result := range manager.World.Query(core.ThreatNodeTag) {
		threatData := common.GetComponentType[*core.ThreatNodeData](result.Entity, core.ThreatNodeComponent)
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

	for _, result := range manager.World.Query(core.ThreatNodeTag) {
		data := common.GetComponentType[*core.ThreatNodeData](result.Entity, core.ThreatNodeComponent)
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

// IsThreatAtPosition checks if any threat node exists at the given position
func IsThreatAtPosition(manager *common.EntityManager, pos coords.LogicalPosition) bool {
	return core.IsThreatAtPosition(manager, pos)
}
