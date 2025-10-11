// Package spawning handles the procedural generation of game entities including monsters,
// loot, equipment, and consumables. It manages loot tables, quality distributions,
// and placement algorithms for populating the game world with interactive content.
package spawning

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"
	"game_main/gear"
	"game_main/rendering"
	"game_main/worldmap"
	"math/rand"

	"github.com/bytearena/ecs"
)

// SpawnConsumable creates a consumable item entity at the specified position.
// It randomly selects item quality and consumable type from loot tables.
func SpawnConsumable(manager *ecs.Manager, xPos, yPos int) *ecs.Entity {

	qual, qualOK := LootQualityTable.GetRandomEntry(false)

	if qualOK {
		c := gear.Consumable{}
		consType, _ := ConsumableSpawnTable.GetRandomEntry(false)
		c.CreateConsumable(consType, qual)

		item := gear.CreateItem(manager, c.Name, coords.LogicalPosition{X: xPos, Y: yPos}, "../assets/items/bubbly.png")
		item.AddComponent(gear.ConsumableComponent, &c)
		return item

	}
	return nil

}

// REMOVED: SpawnRangedWeapon - weapon spawning removed as part of squad system transition

// Basic spawning to start off with. Has a 30% chance to spawn a consumable in the center
func SpawnStartingConsumables(em common.EntityManager, gm *worldmap.GameMap) {
	spawnChance := 30

	for _, room := range gm.Rooms {

		roll := rand.Intn(spawnChance + 1)

		if roll < spawnChance {

			x, y := room.Center()

			randInd := rand.Intn(len(entitytemplates.ConsumableTemplates))

			wep := entitytemplates.CreateEntityFromTemplate(em, entitytemplates.EntityConfig{
				Type:      entitytemplates.EntityConsumable,
				Name:      entitytemplates.ConsumableTemplates[randInd].Name,
				ImagePath: entitytemplates.ConsumableTemplates[randInd].ImgName,
				AssetDir:  "../assets/items/",
				Visible:   false,
				Position:  nil,
			}, entitytemplates.ConsumableTemplates[randInd])

			common.GetComponentType[*rendering.Renderable](wep, rendering.RenderableComponent).Visible = true

			pos := common.GetPosition(wep)

			pos.X = x + 1
			pos.Y = y + 1

			gm.AddEntityToTile(wep, pos)

		}
	}
}

// REMOVED: SpawnStartingEquipment - equipment spawning removed as part of squad system transition
// Squad system will handle combat equipment through squad templates
func SpawnStartingEquipment(em *common.EntityManager, gm *worldmap.GameMap, pl *avatar.PlayerData) {
	// No-op - weapon/armor spawning removed
}

// Spawns loot in a square of size "size" centered at the position
// Right now it's used to spawn loot in a square centered around the player
// Todo, this is just some basic spawning. There's a chance this can spawn an
// item in the same room as the player. That will look weird.
// Add better spawning mechanics later
//

func SpawnLootAroundPlayer(currentTurnNumber int, playerData avatar.PlayerData, manager *ecs.Manager, gm *worldmap.GameMap) {

	if currentTurnNumber%10 != 0 {
		return
	}

	//To get a random position for spawning the item
	getRandomEntry := func(posSlice []coords.LogicalPosition) coords.LogicalPosition {

		return posSlice[rand.Intn(len(posSlice))]

	}

	logicalPos := coords.LogicalPosition{X: playerData.Pos.X, Y: playerData.Pos.Y}
	pixelPos := coords.CoordManager.LogicalToPixel(logicalPos)
	playerX, playerY := pixelPos.X, pixelPos.Y
	spawnPositions := gm.UnblockedLogicalCoords(playerX, playerY, 10)
	consChance, throwableChance := rand.Intn(100), rand.Intn(100)

	if consChance < ConsumableSpawnProb {

		pos := getRandomEntry(spawnPositions)
		e := SpawnConsumable(manager, pos.X, pos.Y)
		gm.AddEntityToTile(e, &coords.LogicalPosition{X: pos.X, Y: pos.Y})

	}
	if throwableChance < ThrowableSpawnProb {

		pos := getRandomEntry(spawnPositions)
		e := SpawnThrowableItem(manager, pos.X, pos.Y)
		gm.AddEntityToTile(e, &coords.LogicalPosition{X: pos.X, Y: pos.Y})

	}

	// Ranged weapon spawning removed - squad system handles combat equipment

}
