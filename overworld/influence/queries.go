package influence

import (
	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// NodePair represents two overlapping influence nodes
type NodePair struct {
	EntityA  *ecs.Entity
	EntityB  *ecs.Entity
	Distance int
}

// overworldNode is a cached node reference used during overlap detection
type overworldNode struct {
	Entity   *ecs.Entity
	EntityID ecs.EntityID
	Pos      coords.LogicalPosition
	Radius   int
}

// FindOverlappingNodes returns all pairs of nodes whose influence radii overlap.
// O(N^2) scan, acceptable for small node counts (~10-50).
func FindOverlappingNodes(manager *common.EntityManager) []NodePair {
	nodes := collectAllNodes(manager)

	var pairs []NodePair
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			dist := nodes[i].Pos.ManhattanDistance(&nodes[j].Pos)
			combinedRadii := nodes[i].Radius + nodes[j].Radius
			if dist <= combinedRadii {
				pairs = append(pairs, NodePair{
					EntityA:  nodes[i].Entity,
					EntityB:  nodes[j].Entity,
					Distance: dist,
				})
			}
		}
	}
	return pairs
}

// collectAllNodes gathers all threat and player nodes with their positions and influence radii.
func collectAllNodes(manager *common.EntityManager) []overworldNode {
	var nodes []overworldNode
	nodes = appendNodesFromTag(nodes, manager, core.ThreatNodeTag)
	nodes = appendNodesFromTag(nodes, manager, core.PlayerNodeTag)
	return nodes
}

// appendNodesFromTag queries entities by tag and appends those with position + influence data.
func appendNodesFromTag(nodes []overworldNode, manager *common.EntityManager, tag ecs.Tag) []overworldNode {
	for _, result := range manager.World.Query(tag) {
		entity := result.Entity
		pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		inf := common.GetComponentType[*core.InfluenceData](entity, core.InfluenceComponent)
		if pos == nil || inf == nil {
			continue
		}
		nodes = append(nodes, overworldNode{
			Entity:   entity,
			EntityID: entity.GetID(),
			Pos:      *pos,
			Radius:   inf.Radius,
		})
	}
	return nodes
}

// getNodesInRadius returns all entity IDs matching a tag within Manhattan distance of pos.
func getNodesInRadius(manager *common.EntityManager, pos coords.LogicalPosition, radius int, tag ecs.Tag) []ecs.EntityID {
	var result []ecs.EntityID
	for _, qr := range manager.World.Query(tag) {
		entity := qr.Entity
		nodePos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		if nodePos == nil {
			continue
		}
		if pos.ManhattanDistance(nodePos) <= radius {
			result = append(result, entity.GetID())
		}
	}
	return result
}

// GetThreatNodesInRadius returns all threat node entity IDs within Manhattan distance of pos.
func GetThreatNodesInRadius(manager *common.EntityManager, pos coords.LogicalPosition, radius int) []ecs.EntityID {
	return getNodesInRadius(manager, pos, radius, core.ThreatNodeTag)
}

// GetPlayerNodesInRadius returns all player node entity IDs within Manhattan distance of pos.
func GetPlayerNodesInRadius(manager *common.EntityManager, pos coords.LogicalPosition, radius int) []ecs.EntityID {
	return getNodesInRadius(manager, pos, radius, core.PlayerNodeTag)
}
