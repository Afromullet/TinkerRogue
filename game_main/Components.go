package main

import (
	"fmt"
	"math"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

/*
 */

var position *ecs.Component
var renderable *ecs.Component
var nameComponent *ecs.Component

var attributeComponent *ecs.Component
var creature *ecs.Component

var WeaponComponent *ecs.Component
var ArmorComponent *ecs.Component
var InventoryComponent *ecs.Component
var userMessage *ecs.Component

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

// Creates a slice of Positions which represent a path build with A-Star
func (p *Position) BuildPath(g *Game, other *Position) []Position {

	astar := AStar{}
	return astar.GetPath(g.gameMap, p, other, false)

}

type Renderable struct {
	Image   *ebiten.Image
	Visible bool
}

type Name struct {
	NameStr string
}

type UserMessage struct {
	AttackMessage    string
	GameStateMessage string
}

type Weapon struct {
	MinDamage int
	MaxDamage int
}

func (w Weapon) CalculateDamage() int {

	return GetRandomBetween(w.MinDamage, w.MaxDamage)

}

type Armor struct {
	ArmorClass  int
	Protection  int
	DodgeChance float32
}

func NewArmor(ac, prot int, dodge float32) Armor {

	return Armor{
		ArmorClass:  ac,
		Protection:  prot,
		DodgeChance: dodge,
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

func UpdateAttributes(e *ecs.Entity) {

	attr := GetComponentType[*Attributes](e, attributeComponent)

	armor := GetComponentType[*Armor](e, ArmorComponent)

	if armor != nil {
		attr.TotalArmorClass = attr.BaseArmorClass + armor.ArmorClass
		attr.TotalProtection = attr.BaseProteciton + armor.Protection
		attr.TotalDodgeChance = attr.BaseDodgeChange + armor.DodgeChance

	}

	fmt.Println("Printing attr", attr.TotalArmorClass, attr.TotalProtection, attr.TotalDodgeChance)

}

// A wrapper around the ECS libraries GetComponentData.
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {

	defer func() {
		if r := recover(); r != nil {

			fmt.Println("Error in passing the component type. Component type must match struct.")

		}
	}()

	if c, ok := entity.GetComponentData(component); ok {
		return c.(T)

	} else {
		var nilValue T
		return nilValue
	}

}

// todo Will be refactored. Don't get distracted by this at the moment.
// ALl of the initialziation will have to be handled differently - since
func InitializeECS(g *Game) {
	tags := make(map[string]ecs.Tag)
	manager := ecs.NewManager()
	position = manager.NewComponent()
	renderable = manager.NewComponent()

	nameComponent = manager.NewComponent()

	InventoryComponent = manager.NewComponent()

	attributeComponent = manager.NewComponent()
	userMessage = manager.NewComponent()

	WeaponComponent = manager.NewComponent()
	ArmorComponent = manager.NewComponent()

	renderables := ecs.BuildTag(renderable, position)
	tags["renderables"] = renderables

	messengers := ecs.BuildTag(userMessage)
	tags["messengers"] = messengers

	InitializeItemComponents(manager, tags)
	InitializeCreatureComponents(manager, tags)

	g.WorldTags = tags
	g.World = manager
}

func InitializeCreatureComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	creature = manager.NewComponent()
	simpleWander = manager.NewComponent()
	noMove = manager.NewComponent()
	goToPlayer = manager.NewComponent()

	creatures := ecs.BuildTag(creature, position, attributeComponent)
	tags["monsters"] = creatures

}
