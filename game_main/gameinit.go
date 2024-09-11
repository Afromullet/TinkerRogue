package main

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/equipment"
	"game_main/timesystem"
	"game_main/worldmap"
	"log"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// todo remove game after handling player data init
func InitializePlayerData(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap) {

	avatar.PlayerComponent = ecsmanager.World.NewComponent()

	playerImg, _, err := ebitenutil.NewImageFromFile("../assets/creatures/player1.png")
	if err != nil {
		log.Fatal(err)
	}

	attr := common.Attributes{}
	attr.MaxHealth = 5
	attr.CurrentHealth = 5
	attr.AttackBonus = 5
	attr.TotalMovementSpeed = 5

	armor := equipment.Armor{
		ArmorClass:  1,
		Protection:  5,
		DodgeChance: 50}

	playerEntity := ecsmanager.World.NewEntity().
		AddComponent(avatar.PlayerComponent, &avatar.Player{}).
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
		}).AddComponent(equipment.ArmorComponent, &armor).
		AddComponent(timesystem.ActionQueueComponent, &timesystem.ActionQueue{TotalActionPoints: 100})

	players := ecs.BuildTag(avatar.PlayerComponent, common.PositionComponent, equipment.InventoryComponent)
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
