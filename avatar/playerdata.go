package avatar

import (
	"game_main/common"
	"game_main/equipment"
	"game_main/graphics"
	"game_main/worldmap"
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
	wep := common.GetComponentType[*equipment.RangedWeapon](pl.PlayerRangedWeapon, equipment.RangedWeaponComponent)
	pl.RangedWeaponAOEShape = wep.TargetArea
	pl.RangedWeaponMaxDistance = wep.ShootingRange

}

type PlayerThrowable struct {
	SelectedThrowable  *ecs.Entity
	ThrowingAOEShape   graphics.TileBasedShape
	ThrowableItemIndex int
	ThrowableItem      *equipment.Item
}

// Handles all conversions necessary for updating item throwing information
// The index lets us remove an item one it's thrown
// The shape lets us draw it on the screen
func (pl *PlayerThrowable) PrepareThrowable(itemEntity *ecs.Entity, index int) {

	pl.SelectedThrowable = itemEntity

	item := equipment.GetItem(pl.SelectedThrowable)
	pl.ThrowableItem = item

	t := item.ItemEffect(equipment.THROWABLE_NAME).(*equipment.Throwable)
	pl.ThrowableItemIndex = index

	pl.ThrowingAOEShape = t.Shape

}

func (pl *PlayerThrowable) ThrowPreparedItem(inv *equipment.Inventory) {

	inv.RemoveItem(pl.ThrowableItemIndex)

}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerEquipment) GetPlayerWeapon() *equipment.MeleeWeapon {

	weapon := common.GetComponentType[*equipment.MeleeWeapon](pl.PlayerWeapon, equipment.WeaponComponent)

	return weapon
}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerRangedWeapon() *equipment.RangedWeapon {

	weapon := common.GetComponentType[*equipment.RangedWeapon](pl.PlayerRangedWeapon, equipment.RangedWeaponComponent)

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

	Pos *common.Position
	Inv *equipment.Inventory

	Targeting bool
}

func NewPlayerData() PlayerData {
	return PlayerData{}
}

// Helper function to make it less tedious to get the inventory
func (pl *PlayerData) GetPlayerInventory() *equipment.Inventory {

	playerInventory := common.GetComponentType[*equipment.Inventory](pl.PlayerEntity, equipment.InventoryComponent)

	return playerInventory
}

// todo remove game after handling player data init
func InitializePlayerData(ecsmanager *common.EntityManager, pl *PlayerData, gm *worldmap.GameMap) {

	player = ecsmanager.World.NewComponent()

	playerImg, _, err := ebitenutil.NewImageFromFile("assets/creatures/player1.png")
	if err != nil {
		log.Fatal(err)
	}

	attr := common.Attributes{}
	attr.MaxHealth = 5
	attr.CurrentHealth = 5
	attr.AttackBonus = 5

	armor := equipment.Armor{1, 5, 50}

	playerEntity := ecsmanager.World.NewEntity().
		AddComponent(player, &Player{}).
		AddComponent(common.RenderableComponent, &common.Renderable{
			Image:   playerImg,
			Visible: true,
		}).
		AddComponent(common.PositionComponent, &common.Position{
			X: 40,
			Y: 45,
		}).
		AddComponent(equipment.InventoryComponent, &equipment.Inventory{
			InventoryContent: make([]*ecs.Entity, 0),
		}).
		AddComponent(common.AttributeComponent, &attr).
		AddComponent(common.UsrMsg, &common.UserMessage{
			AttackMessage:    "",
			GameStateMessage: "",
		}).AddComponent(equipment.ArmorComponent, &armor)

	players := ecs.BuildTag(player, common.PositionComponent, equipment.InventoryComponent)
	ecsmanager.WorldTags["players"] = players

	//g.playerData = PlayerData{}

	pl.PlayerEntity = playerEntity

	//Don't want to Query for the player position every time, so we're storing it

	startPos := common.GetComponentType[*common.Position](pl.PlayerEntity, common.PositionComponent)
	startPos.X = gm.StartingPosition().X
	startPos.Y = gm.StartingPosition().Y

	inventory := common.GetComponentType[*equipment.Inventory](pl.PlayerEntity, equipment.InventoryComponent)

	pl.Pos = startPos
	pl.Inv = inventory

}
