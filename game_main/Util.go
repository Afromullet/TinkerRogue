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

// Todo this can be removed later. Currently using it for debugging
func ApplyColorToMap(g *Game, indices []int) {

	for _, ind := range indices {

		g.gameMap.ApplyColorMatrixToIndex(ind, graphics.GreenColorMatrix)

	}

}

// Todo this can be removed later. Currently using it for debugging
func ApplyColorToInd(g *Game, index int) {
	g.gameMap.ApplyColorMatrixToIndex(index, graphics.GreenColorMatrix)

}
