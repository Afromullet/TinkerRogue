package main

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/coords"
	"game_main/gear"
	"game_main/rendering"
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

	// Create player attributes using new core attribute system
	// Strength: 15 → 50 HP (20 + 15*2)
	// Dexterity: 20 → 100% hit, 10% crit, 6% dodge
	// Magic: 0 → Player starts without magic abilities
	// Leadership: 0 → Player doesn't start with squad leadership
	// Armor: 2 → 4 physical resistance (2*2)
	// Weapon: 3 → 6 bonus damage (3*2)
	attr := common.NewAttributes(15, 20, 0, 0, 2, 3)

	playerEntity := ecsmanager.World.NewEntity().
		AddComponent(avatar.PlayerComponent, &avatar.Player{}).
		AddComponent(rendering.RenderableComponent, &rendering.Renderable{
			Image:   playerImg,
			Visible: true,
		}).
		AddComponent(common.PositionComponent, &coords.LogicalPosition{
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

	startPos := common.GetComponentType[*coords.LogicalPosition](pl.PlayerEntity, common.PositionComponent)
	startPos.X = gm.StartingPosition().X
	startPos.Y = gm.StartingPosition().Y

	inventory := common.GetComponentType[*gear.Inventory](pl.PlayerEntity, gear.InventoryComponent)

	// Test weapon/armor initialization removed - squad system handles combat equipment
	// See CLAUDE.md Section 7 (Squad System Infrastructure) for replacement system

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
