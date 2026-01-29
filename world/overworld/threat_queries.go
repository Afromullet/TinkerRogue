package overworld

import (
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// GetAllThreatNodes returns all threat node entities
func GetAllThreatNodes(manager *common.EntityManager) []*ecs.Entity {
	var threats []*ecs.Entity
	for _, result := range manager.World.Query(ThreatNodeTag) {
		threats = append(threats, result.Entity)
	}
	return threats
}

// GetThreatNodeAt returns threat node at specific position
func GetThreatNodeAt(manager *common.EntityManager, pos coords.LogicalPosition) *ecs.Entity {
	entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(pos)
	for _, entityID := range entityIDs {
		if manager.HasComponent(entityID, ThreatNodeComponent) {
			return manager.FindEntityByID(entityID)
		}
	}
	return nil
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
