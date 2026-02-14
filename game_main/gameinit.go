package main

import (
	"game_main/common"
	"game_main/config"
	"game_main/gear"
	"game_main/tactical/commander"
	_ "game_main/tactical/squadcommands" // Blank import to trigger init() for command queue components
	"game_main/tactical/squads"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"game_main/world/worldmap"

	"log"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// InitializePlayerData creates and configures the player entity with all necessary components.
// It sets up the player's position, attributes, inventory, equipment, and adds them to the ECS world.
func InitializePlayerData(ecsmanager *common.EntityManager, pl *common.PlayerData, gm *worldmap.GameMap) {

	// PlayerComponent already registered in componentinit.go - no need to recreate

	playerImg, _, err := ebitenutil.NewImageFromFile(config.PlayerImagePath)
	if err != nil {
		log.Fatal(err)
	}

	// Create player attributes using default configuration values (see config.go)
	attr := common.NewAttributes(
		config.DefaultPlayerStrength,
		config.DefaultPlayerDexterity,
		config.DefaultPlayerMagic,
		config.DefaultPlayerLeadership,
		config.DefaultPlayerArmor,
		config.DefaultPlayerWeapon,
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
		AddComponent(common.ResourceStockpileComponent, common.NewResourceStockpile(
			config.DefaultPlayerStartingGold,
			config.DefaultPlayerStartingIron,
			config.DefaultPlayerStartingWood,
			config.DefaultPlayerStartingStone,
		)).
		AddComponent(squads.UnitRosterComponent, squads.NewUnitRoster(config.DefaultPlayerMaxUnits)).
		AddComponent(commander.CommanderRosterComponent, &commander.CommanderRosterData{
			CommanderIDs:  make([]ecs.EntityID, 0),
			MaxCommanders: config.DefaultMaxCommanders,
		})

	players := ecs.BuildTag(common.PlayerComponent, common.PositionComponent, gear.InventoryComponent)
	ecsmanager.WorldTags["players"] = players

	//g.playerData = PlayerData{}

	pl.PlayerEntityID = playerEntity.GetID()

	//Don't want to Query for the player position every time, so we're storing it

	startPos := common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)
	startPos.X = gm.StartingPosition().X
	startPos.Y = gm.StartingPosition().Y

	// Test weapon/armor initialization removed - squad system handles combat equipment
	// See CLAUDE.md Section 7 (Squad System Infrastructure) for replacement system

	pl.Pos = startPos

	// Add player to PositionSystem for tracking
	if common.GlobalPositionSystem != nil {
		common.GlobalPositionSystem.AddEntity(playerEntity.GetID(), *startPos)
	}

}

