package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

/*
 */

type Carryable struct{}

type Position struct {
	X int
	Y int
}

func (p *Position) IsEqual(other *Position) bool {
	return (p.X == other.X && p.Y == other.Y)
}

func (p *Position) GetManhattanDistance(other *Position) int {
	xDist := math.Abs(float64(p.X - other.X))
	yDist := math.Abs(float64(p.Y - other.Y))
	return int(xDist) + int(yDist)
}

func (p *Position) BuildPath(g *Game, other *Position) []Position {

	astar := AStar{}
	return astar.GetPath(g.gameMap, p, other)

}

type Renderable struct {
	Image   *ebiten.Image
	visible bool
}

type Name struct {
	NameStr string
}

type UserMessage struct {
	BasicMessage string
}
