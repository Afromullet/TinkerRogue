package core

import (
	"game_main/config"
	"game_main/world/coords"
)

// WalkableGrid tracks which overworld tiles are walkable (passable terrain).
// Populated from the map generator during game setup.
var WalkableGrid []bool

// InitWalkableGrid creates the walkable grid sized for the overworld map
func InitWalkableGrid(width, height int) {
	WalkableGrid = make([]bool, width*height)
}

// SetTileWalkable marks a tile as walkable or not
func SetTileWalkable(pos coords.LogicalPosition, walkable bool) {
	idx := pos.Y*config.DefaultMapWidth + pos.X
	if idx >= 0 && idx < len(WalkableGrid) {
		WalkableGrid[idx] = walkable
	}
}

// IsTileWalkable returns whether a tile is walkable terrain.
// Returns true if the grid hasn't been initialized (graceful fallback).
func IsTileWalkable(pos coords.LogicalPosition) bool {
	if WalkableGrid == nil {
		return true
	}
	idx := pos.Y*config.DefaultMapWidth + pos.X
	if idx < 0 || idx >= len(WalkableGrid) {
		return false
	}
	return WalkableGrid[idx]
}
