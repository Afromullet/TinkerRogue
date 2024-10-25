package common

import (
	"fmt"
	"math"
)

type Name struct {
	NameStr string
}

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
	CanAct             bool
	DamageBonus        int
}

func NewBaseAttributes(maxHealth, attackBonus, baseAC, baseProt, baseMovSpeed int, dodge float32, damageBonus int) Attributes {
	return Attributes{
		MaxHealth:         maxHealth,
		CurrentHealth:     maxHealth,
		AttackBonus:       attackBonus,
		BaseArmorClass:    baseAC,
		BaseProtection:    baseProt,
		BaseDodgeChance:   dodge,
		BaseMovementSpeed: baseMovSpeed,
		DamageBonus:       damageBonus,
	}
}

// For Displaying to the player
func (a Attributes) DisplayString() string {

	res := ""
	res += fmt.Sprintln("HP ", a.CurrentHealth, "/", a.MaxHealth)
	res += fmt.Sprintln("AC", a.TotalArmorClass)
	res += fmt.Sprintln("Prot", a.TotalProtection)
	res += fmt.Sprintln("Dodge", a.TotalDodgeChance)
	res += fmt.Sprintln("Move Speed", a.TotalMovementSpeed)
	res += fmt.Sprintln("Attack Speed", a.TotalAttackSpeed)

	return res

}
