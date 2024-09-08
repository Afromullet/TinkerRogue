package ecshelper

import (
	"math"

	"github.com/bytearena/ecs"
)

// This file contains

var PositionComponent *ecs.Component

type Position struct {
	X int
	Y int
}

func (p *Position) IsEqual(other *Position) bool {
	return (p.X == other.X && p.Y == other.Y)
}

func (p *Position) ManhattanDistance(other *Position) int {
	xDist := math.Abs(float64(p.X - other.X))
	yDist := math.Abs(float64(p.Y - other.Y))
	return int(xDist) + int(yDist)
}

func (p *Position) InRange(other *Position, distance int) bool {

	return p.ManhattanDistance(other) <= distance

}
