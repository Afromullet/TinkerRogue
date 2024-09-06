package main

import (
	"fmt"
	"math"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

/*
 */

var PositionComponent *ecs.Component
var RenderableComponent *ecs.Component
var nameComponent *ecs.Component

var AttributeComponent *ecs.Component
var CreatureComponent *ecs.Component

var WeaponComponent *ecs.Component
var RangedWeaponComponent *ecs.Component
var ArmorComponent *ecs.Component
var InventoryComponent *ecs.Component
var userMessage *ecs.Component

// The ECS library returns pointers to the struct when querying it for components, so the Position methods take a pointer as input
// Other than that, there's no reason for using pointers for the functions below.

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

// Creates a slice of Positions from p to other. Uses AStar to build the path
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

// TargetArea is the area the weapon covers
// I.E, a pistol is just a 1 by 1 rectangle, a shotgun uses a cone, and so on
type RangedWeapon struct {
	MinDamage     int
	MaxDamage     int
	ShootingRange int
	TargetArea    TileBasedShape
}

// todo add ammo to this
func (r RangedWeapon) CalculateDamage() int {

	return GetRandomBetween(r.MinDamage, r.MaxDamage)

}

// Gets all of the targets in the weapons AOE
func (r RangedWeapon) GetTargets(g *Game) []*ecs.Entity {

	pos := GetTilePositions(r.TargetArea)
	targets := make([]*ecs.Entity, 0)

	//TODO, this will be slow in case there are a lot of creatures
	for _, c := range g.World.Query(g.WorldTags["monsters"]) {

		curPos := c.Components[PositionComponent].(*Position)

		for _, p := range pos {
			if curPos.IsEqual(&p) {
				targets = append(targets, c.Entity)

			}
		}

	}

	return targets
}

type Armor struct {
	ArmorClass  int
	Protection  int
	DodgeChance float32
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

	attr := GetComponentType[*Attributes](e, AttributeComponent)

	armor := GetComponentType[*Armor](e, ArmorComponent)

	if armor != nil {
		attr.TotalArmorClass = attr.BaseArmorClass + armor.ArmorClass
		attr.TotalProtection = attr.BaseProteciton + armor.Protection
		attr.TotalDodgeChance = attr.BaseDodgeChange + armor.DodgeChance

	}

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

// The functions which are a GetComponentType wrapper get called frequency
func GetPosition(e *ecs.Entity) *Position {
	return GetComponentType[*Position](e, PositionComponent)
}

// This gets called so often that it might as well be a function
func GetItem(e *ecs.Entity) *Item {
	return GetComponentType[*Item](e, ItemComponent)
}

// This gets called so often that it might as well be a function
func GetAttributes(e *ecs.Entity) *Attributes {
	return GetComponentType[*Attributes](e, AttributeComponent)
}

// This gets called so often that it might as well be a function
func GetCreature(e *ecs.Entity) *Creature {
	return GetComponentType[*Creature](e, CreatureComponent)
}

// todo Will be refactored. Don't get distracted by this at the moment.
// ALl of the initialziation will have to be handled differently - since
func InitializeECS(g *Game) {
	tags := make(map[string]ecs.Tag)
	manager := ecs.NewManager()
	PositionComponent = manager.NewComponent()
	RenderableComponent = manager.NewComponent()

	nameComponent = manager.NewComponent()

	InventoryComponent = manager.NewComponent()

	AttributeComponent = manager.NewComponent()
	userMessage = manager.NewComponent()

	WeaponComponent = manager.NewComponent()
	RangedWeaponComponent = manager.NewComponent()
	ArmorComponent = manager.NewComponent()

	renderables := ecs.BuildTag(RenderableComponent, PositionComponent)
	tags["renderables"] = renderables

	messengers := ecs.BuildTag(userMessage)
	tags["messengers"] = messengers

	InitializeMovementComponents(manager, tags)
	InitializeItemComponents(manager, tags)
	InitializeCreatureComponents(manager, tags)

	g.WorldTags = tags
	g.World = manager
}

func InitializeCreatureComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	CreatureComponent = manager.NewComponent()

	approachAndAttack = manager.NewComponent()
	distanceRangeAttack = manager.NewComponent()

	creatures := ecs.BuildTag(CreatureComponent, PositionComponent, AttributeComponent)
	tags["monsters"] = creatures

}
