package playernode

import (
	"math"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"
)

// PlacementResult indicates whether a placement is valid and why.
type PlacementResult struct {
	Valid  bool
	Reason string
}

// ValidatePlacement checks whether a player node can be placed at the given position.
func ValidatePlacement(manager *common.EntityManager, pos coords.LogicalPosition, playerData *common.PlayerData) PlacementResult {
	// Check walkable terrain
	if !core.IsTileWalkable(pos) {
		return PlacementResult{Valid: false, Reason: "Terrain is not walkable"}
	}

	// Check no existing node (threat or player) at position
	if core.IsAnyNodeAtPosition(manager, pos) {
		return PlacementResult{Valid: false, Reason: "Position already occupied by a node"}
	}

	// Check max node limit
	maxNodes := core.GetMaxPlayerNodes()
	if CountPlayerNodes(manager) >= maxNodes {
		return PlacementResult{Valid: false, Reason: "Maximum node limit reached"}
	}

	// Check placement range from player or any existing player node
	maxRange := float64(core.GetMaxPlacementRange())

	inRange := false

	// Check distance from player position
	if playerData != nil && playerData.Pos != nil {
		dx := float64(pos.X - playerData.Pos.X)
		dy := float64(pos.Y - playerData.Pos.Y)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist <= maxRange {
			inRange = true
		}
	}

	// Check distance from any existing player node
	if !inRange {
		nearestDist := GetNearestPlayerNodeDistance(manager, pos)
		if nearestDist <= maxRange {
			inRange = true
		}
	}

	if !inRange {
		return PlacementResult{Valid: false, Reason: "Too far from player or existing nodes"}
	}

	return PlacementResult{Valid: true, Reason: ""}
}
