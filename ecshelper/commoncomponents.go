package ecshelper

import (
	"game_main/graphics"
	"math"

	"github.com/bytearena/ecs"
)

// This file contains

var (
	PositionComponent  *ecs.Component
	NameComponent      *ecs.Component
	AttributeComponent *ecs.Component
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

func PositionFromIndex(i, screenWidth, screenHeight int) Position {

	return Position{
		X: i % screenWidth,
		Y: i / screenHeight,
	}

}

func GridPositionFromPixels(x, y int) Position {
	gd := graphics.NewScreenData()
	return Position{
		X: x / gd.TileWidth,
		Y: y / gd.TileHeight,
	}
}

type Attributes struct {
	MaxHealth        int
	CurrentHealth    int
	AttackBonus      int
	BaseArmorClass   int
	BaseProteciton   int
	BaseDodgeChange  float32
	TotalArmorClass  int
	TotalProtection  int
	TotalDodgeChance float32
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
