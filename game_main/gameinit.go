package main

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/gear"
	"game_main/rendering"
	"game_main/testing"
	tracker "game_main/trackers"
	"game_main/worldmap"
	"log"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// InitializePlayerData creates and configures the player entity with all necessary components.
// It sets up the player's position, attributes, inventory, equipment, and adds them to the ECS world.
func InitializePlayerData(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap) {

	avatar.PlayerComponent = ecsmanager.World.NewComponent()

	playerImg, _, err := ebitenutil.NewImageFromFile("../assets/creatures/player1.png")
	if err != nil {
		log.Fatal(err)
	}

	attr := common.Attributes{}
	attr.MaxHealth = 50
	attr.CurrentHealth = 50
	attr.AttackBonus = 5
	attr.TotalMovementSpeed = 5
	attr.BaseMovementSpeed = 5

	attr.TotalAttackSpeed = 1

	playerEntity := ecsmanager.World.NewEntity().
		AddComponent(avatar.PlayerComponent, &avatar.Player{}).
		AddComponent(rendering.RenderableComponent, &rendering.Renderable{
			Image:   playerImg,
			Visible: true,
		}).
		AddComponent(common.PositionComponent, &common.Position{
			X: 40,
			Y: 45,
		}).
		AddComponent(gear.InventoryComponent, &gear.Inventory{
			InventoryContent: make([]*ecs.Entity, 0),
		}).
		AddComponent(common.AttributeComponent, &attr).
		AddComponent(common.UserMsgComponent, &common.UserMessage{
			AttackMessage:    "",
			GameStateMessage: "",
		})

	playerEntity.AddComponent(common.UserMsgComponent, &common.UserMessage{})
	players := ecs.BuildTag(avatar.PlayerComponent, common.PositionComponent, gear.InventoryComponent)
	ecsmanager.WorldTags["players"] = players

	//g.playerData = PlayerData{}

	pl.PlayerEntity = playerEntity

	//Don't want to Query for the player position every time, so we're storing it

	startPos := common.GetComponentType[*common.Position](pl.PlayerEntity, common.PositionComponent)
	startPos.X = gm.StartingPosition().X
	startPos.Y = gm.StartingPosition().Y

	inventory := common.GetComponentType[*gear.Inventory](pl.PlayerEntity, gear.InventoryComponent)

	a := testing.CreateArmor(ecsmanager.World, "A1", *startPos, "../assets/items/sword.png", 10, 5, 1)
	w := testing.CreateWeapon(ecsmanager.World, "W1", *startPos, "../assets/items/sword.png", 5, 10)
	r := testing.CreatedRangedWeapon(ecsmanager.World, "R1", "../assets/items/sword.png", *startPos, 5, 10, 100, testing.TestRect)

	pl.Equipment.EqMeleeWeapon = w
	pl.Equipment.EqRangedWeapon = r
	pl.Equipment.EqArmor = a

	armor := common.GetComponentType[*gear.Armor](pl.Equipment.EqArmor, gear.ArmorComponent)

	pl.PlayerEntity.AddComponent(gear.ArmorComponent, armor)

	pl.Pos = startPos
	pl.Inventory = inventory

}

// AddCreaturesToTracker registers all existing monster entities with the creature tracking system.
// It queries for all monsters in the ECS world and adds them to the global CreatureTracker.
func AddCreaturesToTracker(ecsmanger *common.EntityManager) {

	for _, c := range ecsmanger.World.Query(ecsmanger.WorldTags["monsters"]) {

		tracker.CreatureTracker.Add(c.Entity)

	}

}
