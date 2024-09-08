package main

import (
	"crypto/rand"
	"game_main/ecshelper"
	"game_main/graphics"
	"math/big"

	"github.com/bytearena/ecs"
)

// A TileBasedShape returns indices that correspond to the elements in the gamemaps Tiles slice
// This returns the X,Y positions since we handle player and creature location through Position
func GetTilePositions(ts graphics.TileBasedShape) []ecshelper.Position {

	gd := graphics.NewScreenData()
	indices := ts.GetIndices()

	pos := make([]ecshelper.Position, len(indices))

	for i, inds := range indices {

		pos[i] = ecshelper.PositionFromIndex(inds, gd.ScreenWidth, gd.ScreenHeight)

	}

	return pos

}

func DistanceBetween(e1 *ecs.Entity, e2 *ecs.Entity) int {

	pos1 := ecshelper.GetPosition(e1)
	pos2 := ecshelper.GetPosition(e2)

	return pos1.ManhattanDistance(pos2)

}

// GetDiceRoll returns an integer from 1 to the number
func GetDiceRoll(num int) int {
	x, _ := rand.Int(rand.Reader, big.NewInt(int64(num)))
	return int(x.Int64()) + 1

}

func GetRandomBetween(low int, high int) int {
	var randy int = -1
	for {
		randy = GetDiceRoll(high)
		if randy >= low {
			break
		}
	}
	return randy
}

// Max returns the larger of x or y.
func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// Min returns the smaller of x or y.
func min(x, y int) int {
	if x > y {
		return y
	}
	return x
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
