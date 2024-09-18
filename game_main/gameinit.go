package main

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/gear"
	"game_main/rendering"
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
	attr.MaxHealth = 50
	attr.CurrentHealth = 50
	attr.AttackBonus = 5
	attr.TotalMovementSpeed = 1
	attr.BaseMovementSpeed = 1

	armor := gear.Armor{
		ArmorClass:  1,
		Protection:  5,
		DodgeChance: 1}

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
		AddComponent(common.UsrMsg, &common.UserMessage{
			AttackMessage:    "",
			GameStateMessage: "",
		}).AddComponent(gear.ArmorComponent, &armor).
		AddComponent(timesystem.ActionQueueComponent, &timesystem.ActionQueue{TotalActionPoints: 100})

	players := ecs.BuildTag(avatar.PlayerComponent, common.PositionComponent, gear.InventoryComponent)
	ecsmanager.WorldTags["players"] = players

	//g.playerData = PlayerData{}

	pl.PlayerEntity = playerEntity

	//Don't want to Query for the player position every time, so we're storing it

	startPos := common.GetComponentType[*common.Position](pl.PlayerEntity, common.PositionComponent)
	startPos.X = gm.StartingPosition().X
	startPos.Y = gm.StartingPosition().Y

	inventory := common.GetComponentType[*gear.Inventory](pl.PlayerEntity, gear.InventoryComponent)

	pl.Pos = startPos
	pl.Inv = inventory

}
