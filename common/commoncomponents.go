package common

import (
	"game_main/graphics"
	"math"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// This file contains

var (
	PositionComponent   *ecs.Component
	NameComponent       *ecs.Component
	AttributeComponent  *ecs.Component
	RenderableComponent *ecs.Component //Putting this here for now rather than in graphics
	UsrMsg              *ecs.Component //I can probably remove this later
)

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

// The functions which are a GetComponentType wrapper get called frequency
func GetPosition(e *ecs.Entity) *Position {
	return GetComponentType[*Position](e, PositionComponent)
}

func PixelsFromPosition(pos *Position, tileWidth, tileHeight int) (int, int) {

	return pos.X * tileWidth, pos.Y * tileHeight
}

func PositionFromIndex(i, screenWidth int) Position {

	return Position{
		X: i % screenWidth,
		Y: i / screenWidth,
	}

}

func GridPositionFromPixels(x, y int) Position {
	gd := graphics.NewScreenData()
	return Position{
		X: x / gd.TileWidth,
		Y: y / gd.TileHeight,
	}
}

// A TileBasedShape returns indices that correspond to the elements in the gamemaps Tiles slice
// This returns the X,Y positions since we handle player and creature location through Position
func GetTilePositions(indices []int) []Position {

	gd := graphics.NewScreenData()

	pos := make([]Position, len(indices))

	for i, inds := range indices {

		pos[i] = PositionFromIndex(inds, gd.ScreenWidth)

	}

	return pos

}

type Attributes struct {
	MaxHealth          int
	CurrentHealth      int
	AttackBonus        int
	BaseArmorClass     int
	BaseProteciton     int
	BaseDodgeChange    float32
	TotalArmorClass    int
	TotalProtection    int
	TotalDodgeChance   float32
	TotalMovementSpeed int
	TotalAttackSpeed   int
}

func UpdateAttributes(e *ecs.Entity, armorClass, protection int, dodgechance float32) {

	attr := GetComponentType[*Attributes](e, AttributeComponent)

	attr.TotalArmorClass = attr.BaseArmorClass + armorClass
	attr.TotalProtection = attr.BaseProteciton + protection
	attr.TotalDodgeChance = attr.BaseDodgeChange + dodgechance

}

type Name struct {
	NameStr string
}

type Renderable struct {
	Image   *ebiten.Image
	Visible bool
}

type UserMessage struct {
	AttackMessage    string
	GameStateMessage string
}
