package main

import (
	"log"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var player *ecs.Component

type Player struct {
}

// Used to keep track of frequently accessed player information.
// Throwing items is an important part of the game, so we store additional information related
// TO throwing
type PlayerData struct {
	PlayerEntity       *ecs.Entity
	PlayerWeapon       *ecs.Entity
	position           *Position
	inventory          *Inventory
	SelectedThrowable  *ecs.Entity
	Shape              TileBasedShape
	ThrowableItemIndex int
	ThrowableItem      *Item
}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerInventory() *Inventory {

	playerInventory := GetComponentType[*Inventory](pl.PlayerEntity, InventoryComponent)

	return playerInventory
}

// Handles all conversions necessary for updating item throwing information
// The index lets us remove an item one it's thrown
// The shape lets us draw it on the screen
func (pl *PlayerData) PrepareThrowable(itemEntity *ecs.Entity, index int) {

	pl.SelectedThrowable = itemEntity
	item := GetComponentType[*Item](pl.SelectedThrowable, ItemComponent)
	pl.ThrowableItem = item

	t := item.ItemEffect(THROWABLE_NAME).(*Throwable)
	pl.ThrowableItemIndex = index

	pl.Shape = t.Shape

}

func (pl *PlayerData) ThrowPreparedItem() {

	pl.inventory.RemoveItem(pl.ThrowableItemIndex)

}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerWeapon() *Weapon {

	weapon := GetComponentType[*Weapon](pl.PlayerWeapon, WeaponComponent)

	return weapon
}

func InitializePlayerData(g *Game) {

	player = g.World.NewComponent()

	playerImg, _, err := ebitenutil.NewImageFromFile("assets/creatures/player1.png")
	if err != nil {
		log.Fatal(err)
	}

	attr := Attributes{}

	attr.MaxHealth = 5
	attr.CurrentHealth = 5
	attr.AttackBonus = 5

	armor := NewArmor(1, 5, 50)
	playerEntity := g.World.NewEntity().
		AddComponent(player, &Player{}).
		AddComponent(renderable, &Renderable{
			Image:   playerImg,
			Visible: true,
		}).
		AddComponent(position, &Position{
			X: 40,
			Y: 45,
		}).
		AddComponent(InventoryComponent, &Inventory{
			InventoryContent: make([]*ecs.Entity, 0),
		}).
		AddComponent(attributeComponent, &attr).
		AddComponent(userMessage, &UserMessage{
			AttackMessage:    "",
			GameStateMessage: "",
		}).AddComponent(ArmorComponent, &armor)

	players := ecs.BuildTag(player, position, InventoryComponent)
	g.WorldTags["players"] = players

	g.playerData = PlayerData{}

	g.playerData.PlayerEntity = playerEntity

	//Don't want to Query for the player position every time, so we're storing it

	startPos := GetComponentType[*Position](g.playerData.PlayerEntity, position)
	startPos.X = g.gameMap.StartingPosition().X
	startPos.Y = g.gameMap.StartingPosition().Y

	inventory := GetComponentType[*Inventory](g.playerData.PlayerEntity, InventoryComponent)

	g.playerData.position = startPos
	g.playerData.inventory = inventory

}
