package main

import (
	"game_main/common"
	"game_main/coords"
	"game_main/gear"
	"game_main/rendering"
	"game_main/worldmap"

	"log"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// InitializePlayerData creates and configures the player entity with all necessary components.
// It sets up the player's position, attributes, inventory, equipment, and adds them to the ECS world.
func InitializePlayerData(ecsmanager *common.EntityManager, pl *common.PlayerData, gm *worldmap.GameMap) {

	common.PlayerComponent = ecsmanager.World.NewComponent()

	playerImg, _, err := ebitenutil.NewImageFromFile(PlayerImagePath)
	if err != nil {
		log.Fatal(err)
	}

	// Create player attributes using default configuration values (see config.go)
	attr := common.NewAttributes(
		DefaultPlayerStrength,
		DefaultPlayerDexterity,
		DefaultPlayerMagic,
		DefaultPlayerLeadership,
		DefaultPlayerArmor,
		DefaultPlayerWeapon,
	)

	playerEntity := ecsmanager.World.NewEntity().
		AddComponent(common.PlayerComponent, &common.Player{}).
		AddComponent(rendering.RenderableComponent, &rendering.Renderable{
			Image:   playerImg,
			Visible: true,
		}).
		AddComponent(common.PositionComponent, &coords.LogicalPosition{
			X: 40,
			Y: 45,
		}).
		AddComponent(gear.InventoryComponent, &gear.Inventory{
			ItemEntityIDs: make([]ecs.EntityID, 0),
		}).
		AddComponent(common.AttributeComponent, &attr).
		AddComponent(common.UserMsgComponent, &common.UserMessage{
			AttackMessage:    "",
			GameStateMessage: "",
		})

	playerEntity.AddComponent(common.UserMsgComponent, &common.UserMessage{})
	players := ecs.BuildTag(common.PlayerComponent, common.PositionComponent, gear.InventoryComponent)
	ecsmanager.Tags["players"] = players

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

	// Add player to PositionSystem for tracking
	if common.GlobalPositionSystem != nil {
		common.GlobalPositionSystem.AddEntity(playerEntity.GetID(), *startPos)
	}

}

// AddCreaturesToTracker registers all existing monster entities with the creature tracking system.
// It queries for all monsters in the ECS world and adds them to both the global CreatureTracker
// and the new PositionSystem for O(1) position lookups.
func AddCreaturesToTracker(ecsmanger *common.EntityManager) {

	for _, c := range ecsmanger.World.Query(ecsmanger.Tags["monsters"]) {

		// Also add to new PositionSystem for O(1) lookups
		if common.GlobalPositionSystem != nil {
			pos := common.GetPosition(c.Entity)
			if pos != nil {
				common.GlobalPositionSystem.AddEntity(c.Entity.GetID(), *pos)
			}
		}

	}

}
