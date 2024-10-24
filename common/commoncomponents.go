package common

import (
	"math"
)

type Position struct {
	X int
	Y int
}

type UserMessage struct {
	AttackMessage       string
	GameStateMessage    string
	StatusEffectMessage string
}

func (p *Position) IsEqual(other *Position) bool {
	return (p.X == other.X && p.Y == other.Y)
}

func (p *Position) ManhattanDistance(other *Position) int {
	xDist := math.Abs(float64(p.X - other.X))
	yDist := math.Abs(float64(p.Y - other.Y))
	return int(xDist) + int(yDist)
}

func (p *Position) ChebyshevDistance(other *Position) int {
	xDist := math.Abs(float64(p.X - other.X))
	yDist := math.Abs(float64(p.Y - other.Y))
	return int(math.Max(xDist, yDist))
}

// Todo determine what kind of distance function you want to use
func (p *Position) InRange(other *Position, distance int) bool {

	return p.ManhattanDistance(other) <= distance

}

type Name struct {
	NameStr string
}

// Anything that displays text in the GUI implements this interface
type StringDisplay interface {
	DisplayString()
}

// Not a component, but there's no need to create a source file for just this

// Interface to create an item of a quality. Used for loot generation.
// Take a look at itemquality.go to see how it's implemented
// Implementation looks like this
// //func (t *Throwable) CreateWithQuality(q common.QualityType) {
// ...}
// Which means that we are changing a refernece. Not the best implementation, since it
// Requires us to create an object first. Todo change that in the future. Maybe use a factory?
type Quality interface {
	CreateWithQuality(q QualityType)
}

type QualityType int

var LowQualStr = "Low Quality"
var NormalQualStr = "Normal Quality"
var HighQualStr = "High Quality"

const (
	LowQuality = iota
	NormalQuality
	HighQuality
)
