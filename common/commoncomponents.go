package common

import (
	"game_main/graphics"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
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

// Gets the Pixel X,Y, not the tile XY
func PixelsFromPosition(pos *Position, tileWidth, tileHeight int) (int, int) {

	return pos.X * tileWidth, pos.Y * tileHeight

}

// Get the Tile X,Y from the Pixels
func PositionFromIndex(i, dungeonWidth int) Position {

	return Position{
		X: i % dungeonWidth,
		Y: i / dungeonWidth,
	}

}

func PositionFromPixels(x, y int) Position {

	return Position{
		X: x / graphics.ScreenInfo.TileWidth,
		Y: y / graphics.ScreenInfo.TileHeight,
	}

}

// Gets the index in the tilemap from the cursor
func GetTileIndexFromCursor() int {

	pos := PositionFromPixels(ebiten.CursorPosition())
	return graphics.IndexFromXY(pos.X, pos.Y)

}

// A TileBasedShape returns indices that correspond to the tiles on the GameMap
// The caller of this function has to decide what to do with the positions.
func GetTilePositions(indices []int) []Position {

	pos := make([]Position, len(indices))

	for i, inds := range indices {
		pos[i] = PositionFromIndex(inds, graphics.ScreenInfo.DungeonWidth)
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

/*
func (a *Attributes) UpdateEntityAttributes(e *ecs.Entity) {

	armor := GetComponentType[*gear.Armor](e, gear.ArmorComponent)

	ac := 0
	prot := 0
	dodge := float32(0.0)

	if armor != nil {

		ac = armor.ArmorClass
		prot = armor.Protection
		dodge = float32(armor.DodgeChance)
	}

	a.TotalArmorClass = a.BaseArmorClass + ac
	a.TotalProtection = a.BaseProtection + prot
	a.TotalDodgeChance = a.BaseDodgeChance + dodge

}
*/

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
