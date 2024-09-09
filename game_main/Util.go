package main

import (
	"game_main/common"
	"game_main/graphics"
)

// A TileBasedShape returns indices that correspond to the elements in the gamemaps Tiles slice
// This returns the X,Y positions since we handle player and creature location through Position
func GetTilePositions(ts graphics.TileBasedShape) []common.Position {

	gd := graphics.NewScreenData()
	indices := ts.GetIndices()

	pos := make([]common.Position, len(indices))

	for i, inds := range indices {

		pos[i] = common.PositionFromIndex(inds, gd.ScreenWidth)

	}

	return pos

}
