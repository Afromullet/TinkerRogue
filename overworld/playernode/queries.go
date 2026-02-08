package playernode

import (
	"math"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"
)

// CountPlayerNodes returns the total number of player-placed nodes.
func CountPlayerNodes(manager *common.EntityManager) int {
	count := 0
	for range manager.World.Query(core.PlayerNodeTag) {
		count++
	}
	return count
}

// GetAllPlayerNodePositions returns positions of all player nodes.
func GetAllPlayerNodePositions(manager *common.EntityManager) []coords.LogicalPosition {
	var positions []coords.LogicalPosition
	for _, result := range manager.World.Query(core.PlayerNodeTag) {
		pos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
		if pos != nil {
			positions = append(positions, *pos)
		}
	}
	return positions
}

// GetNearestPlayerNodeDistance returns the distance to the nearest player node from pos.
// Returns math.MaxFloat64 if no player nodes exist.
func GetNearestPlayerNodeDistance(manager *common.EntityManager, pos coords.LogicalPosition) float64 {
	nearest := math.MaxFloat64
	for _, result := range manager.World.Query(core.PlayerNodeTag) {
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
