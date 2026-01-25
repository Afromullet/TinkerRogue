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

// GetThreatsInRadius returns all threats within distance of a position
func GetThreatsInRadius(manager *common.EntityManager, center coords.LogicalPosition, radius int) []*ecs.Entity {
	var threats []*ecs.Entity
	for _, result := range manager.World.Query(ThreatNodeTag) {
		entity := result.Entity
		pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		if pos != nil && center.ChebyshevDistance(pos) <= radius {
			threats = append(threats, entity)
		}
	}
	return threats
}

// GetThreatsByType returns all threats of a specific type
func GetThreatsByType(manager *common.EntityManager, threatType ThreatType) []*ecs.Entity {
	var threats []*ecs.Entity
	for _, result := range manager.World.Query(ThreatNodeTag) {
		entity := result.Entity
		data := common.GetComponentType[*ThreatNodeData](entity, ThreatNodeComponent)
		if data != nil && data.ThreatType == threatType {
			threats = append(threats, entity)
		}
	}
	return threats
}

// GetThreatIntensity returns intensity level of a threat
func GetThreatIntensity(entity *ecs.Entity) int {
	data := common.GetComponentType[*ThreatNodeData](entity, ThreatNodeComponent)
	if data != nil {
		return data.Intensity
	}
	return 0
}

// GetThreatNodeByID finds a threat node by its entity ID
func GetThreatNodeByID(manager *common.EntityManager, threatID ecs.EntityID) *ecs.Entity {
	entity := manager.FindEntityByID(threatID)
	if entity != nil && manager.HasComponent(threatID, ThreatNodeComponent) {
		return entity
	}
	return nil
}

// GetThreatByID retrieves threat data by entity ID (matches GetFactionByID pattern)
func GetThreatByID(manager *common.EntityManager, threatID ecs.EntityID) *ThreatNodeData {
	entity := manager.FindEntityByID(threatID)
	if entity == nil {
		return nil
	}
	return common.GetComponentType[*ThreatNodeData](entity, ThreatNodeComponent)
}

// CountThreatNodes returns the total number of threat nodes
func CountThreatNodes(manager *common.EntityManager) int {
	count := 0
	for range manager.World.Query(ThreatNodeTag) {
		count++
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
