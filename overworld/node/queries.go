package node

import (
	"math"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"
)

// CountPlayerNodes returns the total number of player-placed nodes.
func CountPlayerNodes(manager *common.EntityManager) int {
	return CountNodesByOwner(manager, core.OwnerPlayer)
}

// GetNearestPlayerNodeDistance returns the distance to the nearest player node from pos.
// Returns math.MaxFloat64 if no player nodes exist.
func GetNearestPlayerNodeDistance(manager *common.EntityManager, pos coords.LogicalPosition) float64 {
	return GetNearestNodeDistance(manager, pos, core.OwnerPlayer)
}

// CountNodesByOwner returns the number of nodes owned by a specific owner.
func CountNodesByOwner(manager *common.EntityManager, ownerID string) int {
	count := 0
	for _, result := range core.OverworldNodeView.Get() {
		data := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
		if data != nil && data.OwnerID == ownerID {
			count++
		}
	}
	return count
}

// GetNearestNodeDistance returns the Euclidean distance to the nearest node owned by ownerID.
// Returns math.MaxFloat64 if no matching nodes exist.
func GetNearestNodeDistance(manager *common.EntityManager, pos coords.LogicalPosition, ownerID string) float64 {
	nearest := math.MaxFloat64
	for _, result := range core.OverworldNodeView.Get() {
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
