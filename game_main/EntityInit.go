package main

import (
	"log"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var position *ecs.Component
var renderable *ecs.Component
var nameComponent *ecs.Component
var inventoryComponent *ecs.Component

// todo Will be refactored. Don't get distracted by this at the moment.
// ALl of the initialziation will have to be handled differently - since
func InitializeECS(g *Game) {
	tags := make(map[string]ecs.Tag)
	manager := ecs.NewManager()
	position = manager.NewComponent()
	renderable = manager.NewComponent()

	nameComponent = manager.NewComponent()
	userMessage := manager.NewComponent()
	inventoryComponent = manager.NewComponent()

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
		AddComponent(inventoryComponent, &Inventory{
			InventoryContent: new([]ecs.Entity),
		})

	players := ecs.BuildTag(player, position, inventoryComponent)
	g.WorldTags["players"] = players

	g.playerData = PlayerData{}

	g.playerData.playerEntity = playerEntity

	//Don't want to Query for the player position every time, so we're storing it

	pos, _ := g.playerData.playerEntity.GetComponentData(position)
	startPos := pos.(*Position)
	startPos.X = g.gameMap.GetStartingPosition().X
	startPos.Y = g.gameMap.GetStartingPosition().Y

	inv, _ := g.playerData.playerEntity.GetComponentData(inventoryComponent)
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
