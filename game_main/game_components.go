package main

import (
	"log"
	"math"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

/*
 */

var position *ecs.Component
var renderable *ecs.Component
var nameComponent *ecs.Component

var healthComponent *ecs.Component
var creature *ecs.Component
var simpleWander *ecs.Component
var noMove *ecs.Component
var goToPlayer *ecs.Component

var WeaponComponent *ecs.Component
var InventoryComponent *ecs.Component

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

type SimpleWander struct {
}

type NoMovement struct {
}

type GoToPlayerMovement struct {
}

type Health struct {
	MaxHealth     int
	CurrentHealth int
}

// todo Will be refactored. Don't get distracted by this at the moment.
// ALl of the initialziation will have to be handled differently - since
func InitializeECS(g *Game) {
	tags := make(map[string]ecs.Tag)
	manager := ecs.NewManager()
	position = manager.NewComponent()
	renderable = manager.NewComponent()

	nameComponent = manager.NewComponent()
	userMessage := manager.NewComponent()
	InventoryComponent = manager.NewComponent()

	healthComponent = manager.NewComponent()

	renderables := ecs.BuildTag(renderable, position)
	tags["renderables"] = renderables

	messengers := ecs.BuildTag(userMessage)
	tags["messengers"] = messengers

	InitializeItemComponents(manager, tags)
	InitializeCreatureComponents(manager, tags)

	g.WorldTags = tags
	g.World = manager
}

// Don't need tags for the movement types because we're not searching by tag.
func InitializeCreatureComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	creature = manager.NewComponent()
	simpleWander = manager.NewComponent()
	noMove = manager.NewComponent()
	goToPlayer = manager.NewComponent()

	creatures := ecs.BuildTag(creature, position)
	tags["monsters"] = creatures

}

func InitializeItemComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	ItemComponent = manager.NewComponent()
	StickyComponent = manager.NewComponent()
	BurningComponent = manager.NewComponent()
	FreezingComponent = manager.NewComponent()
	WeaponComponent = manager.NewComponent()

	AllItemProperties = append(AllItemProperties, StickyComponent)
	AllItemProperties = append(AllItemProperties, BurningComponent)
	AllItemProperties = append(AllItemProperties, FreezingComponent)

	items := ecs.BuildTag(ItemComponent, position) //todo add all the tags
	tags["items"] = items

	sticking := ecs.BuildTag(StickyComponent)
	tags["sticking"] = sticking

	burning := ecs.BuildTag(BurningComponent)
	tags["burning"] = burning

	freezing := ecs.BuildTag(FreezingComponent)
	tags["freezing"] = freezing

}

func InitializePlayerData(g *Game) {

	player = g.World.NewComponent()

	playerImg, _, err := ebitenutil.NewImageFromFile("assets/creatures/player1.png")
	if err != nil {
		log.Fatal(err)
	}

	playerEntity := g.World.NewEntity().
		AddComponent(player, &Player{}).
		AddComponent(renderable, &Renderable{
			Image:   playerImg,
			visible: true,
		}).
		AddComponent(position, &Position{
			X: 40,
			Y: 45,
		}).
		AddComponent(InventoryComponent, &Inventory{
			InventoryContent: new([]ecs.Entity),
		}).
		AddComponent(healthComponent, &Health{
			MaxHealth:     5,
			CurrentHealth: 5,
		})

	players := ecs.BuildTag(player, position, InventoryComponent)
	g.WorldTags["players"] = players

	g.playerData = PlayerData{}

	g.playerData.playerEntity = playerEntity

	//Don't want to Query for the player position every time, so we're storing it

	pos, _ := g.playerData.playerEntity.GetComponentData(position)
	startPos := pos.(*Position)
	startPos.X = g.gameMap.GetStartingPosition().X
	startPos.Y = g.gameMap.GetStartingPosition().Y

	inv, _ := g.playerData.playerEntity.GetComponentData(InventoryComponent)
	inventory := inv.(*Inventory)

	g.playerData.position = startPos
	g.playerData.inventory = inventory

}

// Create an item with any number of Properties. ItemProperty is a wrapper around an ecs.Component to make
// Manipulating it easier
func CreateItem(manager *ecs.Manager, name string, pos Position, imagePath string, properties ...ItemProperty) *ecs.Entity {

	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	item := &Item{count: 1, properties: manager.NewEntity()}

	for _, prop := range properties {
		item.properties.AddComponent(prop.GetPropertyComponent(), &prop)

	}

	itemEntity := manager.NewEntity().
		AddComponent(renderable, &Renderable{
			Image:   img,
			visible: true,
		}).
		AddComponent(position, &Position{
			X: pos.X,
			Y: pos.Y,
		}).
		AddComponent(nameComponent, &Name{
			NameStr: name,
		}).
		AddComponent(ItemComponent, item)

		//TODO where shoudl I add the tags?

	return itemEntity

}

// A weapon is an Item with a weapon component
func CreateWeapon(manager *ecs.Manager, name string, pos Position, imagePath string, dam int, properties ...ItemProperty) *ecs.Entity {

	weapon := CreateItem(manager, name, pos, imagePath, properties...)

	weapon.AddComponent(WeaponComponent, &Weapon{
		damage: dam,
	})

	return weapon

}
