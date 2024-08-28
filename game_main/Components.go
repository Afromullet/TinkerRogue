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
var userMessage *ecs.Component

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
	return astar.GetPath(g.gameMap, p, other, false)

}

type Renderable struct {
	Image   *ebiten.Image
	visible bool
}

type Name struct {
	NameStr string
}

type UserMessage struct {
	AttackMessage    string
	GameStateMessage string
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

// I don't want to keep on calling GetComponentData due to it being annoying syntax
func GetComponentStruct[T any](entity *ecs.Entity, component *ecs.Component) T {

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

	healthComponent = manager.NewComponent()
	userMessage = manager.NewComponent()

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
		}).AddComponent(userMessage, &UserMessage{
		AttackMessage:    "",
		GameStateMessage: "",
	})

	players := ecs.BuildTag(player, position, InventoryComponent)
	g.WorldTags["players"] = players

	g.playerData = PlayerData{}

	g.playerData.playerEntity = playerEntity

	//Don't want to Query for the player position every time, so we're storing it

	startPos := GetComponentStruct[*Position](g.playerData.playerEntity, position)
	startPos.X = g.gameMap.GetStartingPosition().X
	startPos.Y = g.gameMap.GetStartingPosition().Y

	inventory := GetComponentStruct[*Inventory](g.playerData.playerEntity, InventoryComponent)

	g.playerData.position = startPos
	g.playerData.inventory = inventory

}
