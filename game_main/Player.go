package main

import (
	"game_main/graphics"
	"log"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var player *ecs.Component

type Player struct {
}

type PlayerEquipment struct {
	PlayerWeapon            *ecs.Entity
	PlayerRangedWeapon      *ecs.Entity
	RangedWeaponMaxDistance int
	RangedWeaponAOEShape    graphics.TileBasedShape
}

func (pl *PlayerEquipment) PrepareRangedAttack() {
	wep := GetComponentType[*RangedWeapon](pl.PlayerRangedWeapon, RangedWeaponComponent)
	pl.RangedWeaponAOEShape = wep.TargetArea
	pl.RangedWeaponMaxDistance = wep.ShootingRange

}

type PlayerThrowable struct {
	SelectedThrowable  *ecs.Entity
	ThrowingAOEShape   graphics.TileBasedShape
	ThrowableItemIndex int
	ThrowableItem      *Item
}

// Handles all conversions necessary for updating item throwing information
// The index lets us remove an item one it's thrown
// The shape lets us draw it on the screen
func (pl *PlayerThrowable) PrepareThrowable(itemEntity *ecs.Entity, index int) {

	pl.SelectedThrowable = itemEntity

	item := GetItem(pl.SelectedThrowable)
	pl.ThrowableItem = item

	t := item.ItemEffect(THROWABLE_NAME).(*Throwable)
	pl.ThrowableItemIndex = index

	pl.ThrowingAOEShape = t.Shape

}

func (pl *PlayerThrowable) ThrowPreparedItem(inv *Inventory) {

	inv.RemoveItem(pl.ThrowableItemIndex)

}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerEquipment) GetPlayerWeapon() *Weapon {

	weapon := GetComponentType[*Weapon](pl.PlayerWeapon, WeaponComponent)

	return weapon
}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerRangedWeapon() *RangedWeapon {

	weapon := GetComponentType[*RangedWeapon](pl.PlayerRangedWeapon, RangedWeaponComponent)

	return weapon
}

// Used to keep track of frequently accessed player information.
// Throwing items is an important part of the game, so we store additional information related
// ThrowingAOEShape is the shape that highlights the AOE of the thrown item
// isTargeting is a bool that indicates whether the player is currently selecting a ranged target
type PlayerData struct {
	PlayerEquipment
	PlayerThrowable

	PlayerEntity *ecs.Entity

	position  *Position
	inventory *Inventory

	isTargeting bool
}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerInventory() *Inventory {

	playerInventory := GetComponentType[*Inventory](pl.PlayerEntity, InventoryComponent)

	return playerInventory
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

	armor := Armor{1, 5, 50}

	playerEntity := g.World.NewEntity().
		AddComponent(player, &Player{}).
		AddComponent(RenderableComponent, &Renderable{
			Image:   playerImg,
			Visible: true,
		}).
		AddComponent(PositionComponent, &Position{
			X: 40,
			Y: 45,
		}).
		AddComponent(InventoryComponent, &Inventory{
			InventoryContent: make([]*ecs.Entity, 0),
		}).
		AddComponent(AttributeComponent, &attr).
		AddComponent(userMessage, &UserMessage{
			AttackMessage:    "",
			GameStateMessage: "",
		}).AddComponent(ArmorComponent, &armor)

	players := ecs.BuildTag(player, PositionComponent, InventoryComponent)
	g.WorldTags["players"] = players

	g.playerData = PlayerData{}

	g.playerData.PlayerEntity = playerEntity

	//Don't want to Query for the player position every time, so we're storing it

	startPos := GetComponentType[*Position](g.playerData.PlayerEntity, PositionComponent)
	startPos.X = g.gameMap.StartingPosition().X
	startPos.Y = g.gameMap.StartingPosition().Y

	inventory := GetComponentType[*Inventory](g.playerData.PlayerEntity, InventoryComponent)

	g.playerData.position = startPos
	g.playerData.inventory = inventory

}
