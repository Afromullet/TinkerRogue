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

// collectAllNodes gathers all overworld nodes with their positions and influence radii.
// Uses the unified OverworldNodeTag for a single query instead of querying two separate tags.
func collectAllNodes(manager *common.EntityManager) []overworldNode {
	var nodes []overworldNode
	for _, result := range manager.World.Query(core.OverworldNodeTag) {
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

// GetThreatNodesInRadius returns all threat node entity IDs within Manhattan distance of pos.
// Uses unified OverworldNodeTag, filters by hostile owner.
func GetThreatNodesInRadius(manager *common.EntityManager, pos coords.LogicalPosition, radius int) []ecs.EntityID {
	var result []ecs.EntityID
	for _, qr := range manager.World.Query(core.OverworldNodeTag) {
		entity := qr.Entity
		nodeData := common.GetComponentType[*core.OverworldNodeData](entity, core.OverworldNodeComponent)
		if nodeData == nil || !core.IsHostileOwner(nodeData.OwnerID) {
			continue
		}
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

// GetPlayerNodesInRadius returns all player node entity IDs within Manhattan distance of pos.
// Uses unified OverworldNodeTag, filters by player owner.
func GetPlayerNodesInRadius(manager *common.EntityManager, pos coords.LogicalPosition, radius int) []ecs.EntityID {
	var result []ecs.EntityID
	for _, qr := range manager.World.Query(core.OverworldNodeTag) {
		entity := qr.Entity
		nodeData := common.GetComponentType[*core.OverworldNodeData](entity, core.OverworldNodeComponent)
		if nodeData == nil || nodeData.OwnerID != core.OwnerPlayer {
			continue
		}
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
