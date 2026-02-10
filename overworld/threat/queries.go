package threat

import (
	"game_main/common"
	"game_main/overworld/core"
)

// CountThreatNodes returns the total number of threat nodes.
// Uses unified OverworldNodeComponent, filters by threat category.
func CountThreatNodes(manager *common.EntityManager) int {
	count := 0
	for _, result := range manager.World.Query(core.OverworldNodeTag) {
		data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
		if data != nil && data.Category == core.NodeCategoryThreat {
			count++
		}
	}
	return count
}

// CountHighIntensityThreats returns the number of threats at or above the threshold.
// Uses unified OverworldNodeComponent.
func CountHighIntensityThreats(manager *common.EntityManager, threshold int) int {
	count := 0
	for _, result := range manager.World.Query(core.OverworldNodeTag) {
		data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
		if data != nil && data.Category == core.NodeCategoryThreat && data.Intensity >= threshold {
			count++
		}
	}
	return count
}

// CalculateAverageIntensity returns the average intensity of all threats.
// Uses unified OverworldNodeComponent.
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
