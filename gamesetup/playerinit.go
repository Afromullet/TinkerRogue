package gamesetup

import (
	"game_main/common"
	"game_main/config"
	"game_main/gear"
	"game_main/templates"

	"game_main/tactical/commander"
	"game_main/tactical/roster"
	_ "game_main/tactical/squadcommands" // Blank import to trigger init() for command queue components
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

	cfg := templates.GameConfig

	// Create player attributes using default configuration values (see gameconfig.json)
	attr := common.NewAttributes(
		cfg.Player.Attributes.Strength,
		cfg.Player.Attributes.Dexterity,
		cfg.Player.Attributes.Magic,
		cfg.Player.Attributes.Leadership,
		cfg.Player.Attributes.Armor,
		cfg.Player.Attributes.Weapon,
	)

	playerEntity := ecsmanager.World.NewEntity().
		AddComponent(common.PlayerComponent, &common.Player{}).
		AddComponent(common.RenderableComponent, &common.Renderable{
			Image:   playerImg,
			Visible: true,
		}).
		AddComponent(common.AttributeComponent, &attr).
		AddComponent(common.ResourceStockpileComponent, common.NewResourceStockpile(
			cfg.Player.Resources.Gold,
			cfg.Player.Resources.Iron,
			cfg.Player.Resources.Wood,
			cfg.Player.Resources.Stone,
		)).
		AddComponent(roster.UnitRosterComponent, roster.NewUnitRoster(cfg.Player.Limits.MaxUnits)).
		AddComponent(commander.CommanderRosterComponent, &commander.CommanderRosterData{
			CommanderIDs:  make([]ecs.EntityID, 0),
			MaxCommanders: cfg.Commander.MaxCommanders,
		}).
		AddComponent(gear.ArtifactInventoryComponent, gear.NewArtifactInventory(cfg.Player.Limits.MaxArtifacts))

	// Atomically add position component and register with position system
	ecsmanager.RegisterEntityPosition(playerEntity, gm.StartingPosition())

	players := ecs.BuildTag(common.PlayerComponent, common.PositionComponent)
	ecsmanager.WorldTags["players"] = players

	pl.PlayerEntityID = playerEntity.GetID()

	// Store pointer to position component for direct access (avoids querying every frame)
	pl.Pos = common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)

}
