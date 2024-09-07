package main

import (
	"crypto/rand"
	"math/big"

	"github.com/bytearena/ecs"
)

var GreenColorMatrix = ColorMatrix{0, 1, 0, 1, true}
var RedColorMatrix = ColorMatrix{1, 0, 0, 1, true}

var levelHeight int = 0

func DistanceBetween(e1 *ecs.Entity, e2 *ecs.Entity) int {

	pos1 := GetPosition(e1)
	pos2 := GetPosition(e2)

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

		g.gameMap.ApplyColorMatrixToIndex(ind, GreenColorMatrix)

	}

}

// Todo this can be removed later. Currently using it for debugging
func ApplyColorToInd(g *Game, index int) {
	g.gameMap.ApplyColorMatrixToIndex(index, GreenColorMatrix)

}
