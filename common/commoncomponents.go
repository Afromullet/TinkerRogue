package common

import (
	"game_main/graphics"
	"math"
	"strconv"
)

type Position struct {
	X int
	Y int
}

type UserMessage struct {
	AttackMessage    string
	GameStateMessage string
}

func (p *Position) IsEqual(other *Position) bool {
	return (p.X == other.X && p.Y == other.Y)
}

func (p *Position) ManhattanDistance(other *Position) int {
	xDist := math.Abs(float64(p.X - other.X))
	yDist := math.Abs(float64(p.Y - other.Y))
	return int(xDist) + int(yDist)
}

func (p *Position) EuclidiannDistance(other *Position) int {
	xDist := float64(p.X - other.X)
	yDist := float64(p.Y - other.Y)
	return int(math.Sqrt(xDist*xDist+yDist*yDist)) - 1
}

func (p *Position) ChebyshevDistance(other *Position) int {
	xDist := math.Abs(float64(p.X - other.X))
	yDist := math.Abs(float64(p.Y - other.Y))
	return int(math.Max(xDist, yDist))
}

func (p *Position) InRange(other *Position, distance int) bool {

	return p.ManhattanDistance(other) <= distance

}

// A TileBasedShape returns indices that correspond to the tiles on the GameMap
// The caller of this function has to decide what to do with the positions.
func GetTilePositions(indices []int, dungeinWidth int) []Position {

	pos := make([]Position, len(indices))

	x, y := 0, 0
	for i, tileIndex := range indices {

		x, y = graphics.CoordTransformer.LogicalXYFromIndex(tileIndex)
		pos[i] = Position{X: x, Y: y}

	}

	return pos

}

type Attributes struct {
	MaxHealth          int
	CurrentHealth      int
	AttackBonus        int
	BaseArmorClass     int
	BaseProtection     int
	BaseMovementSpeed  int
	BaseDodgeChance    float32
	TotalArmorClass    int
	TotalProtection    int
	TotalDodgeChance   float32
	TotalMovementSpeed int
	TotalAttackSpeed   int
}

func NewBaseAttributes(maxHealth, attackBonus, baseAC, baseProt, baseMovSpeed int, dodge float32) Attributes {
	return Attributes{
		MaxHealth:         maxHealth,
		CurrentHealth:     maxHealth,
		AttackBonus:       attackBonus,
		BaseArmorClass:    baseAC,
		BaseProtection:    baseProt,
		BaseDodgeChance:   dodge,
		BaseMovementSpeed: baseMovSpeed,
	}
}

func (a Attributes) AttributeText() string {
	s := ""
	s += "HP: " + strconv.Itoa(a.CurrentHealth) + "/" + strconv.Itoa(a.MaxHealth) + "\n"
	s += "Armor Class: " + strconv.Itoa(a.TotalArmorClass) + "\n"
	s += "Protection: " + strconv.Itoa(a.TotalProtection) + "\n"
	s += "Dodge: " + strconv.FormatFloat(float64(a.TotalDodgeChance), 'f', 2, 32) + "\n"
	s += "Movement Speed: " + strconv.Itoa(a.TotalMovementSpeed) + "\n"
	s += "Attack Speed: " + strconv.Itoa(a.TotalAttackSpeed) + "\n"

	return s

}

type Name struct {
	NameStr string
}
