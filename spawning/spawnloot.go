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
	"game_main/graphics"
	"game_main/rendering"
	"game_main/worldmap"
	"math/rand"
	"strconv"

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

		item := gear.CreateItem(manager, c.Name, common.Position{X: xPos, Y: yPos}, "../assets/items/bubbly.png")
		item.AddComponent(gear.ConsumableComponent, &c)
		return item

	}
	return nil

}

// SpawnRangedWeapon creates a ranged weapon entity at the specified position.
// It randomly generates quality and AOE shape properties from loot tables.
// TODO: Improve image selection logic.
func SpawnRangedWeapon(manager *ecs.Manager, xPos, yPos int) *ecs.Entity {

	//TODO better name generation
	name := "Ranged " + strconv.Itoa(rand.Intn(1000))
	weapon := gear.CreateItem(manager, name, common.Position{X: xPos, Y: yPos}, "../assets/items/longbow1.png")

	qual, qualOK := LootQualityTable.GetRandomEntry(false)
	aoeShape, shapeOK := ThrowableAOEProbTable.GetRandomEntry(false)

	if qualOK && shapeOK {

		r := gear.RangedWeapon{}
		r.CreateWithQuality(qual)

		// Convert BasicShapeType to actual shape instance
		var targetShape graphics.TileBasedShape
		switch aoeShape {
		case graphics.Circular:
			targetShape = graphics.NewCircle(0, 0, qual)
		case graphics.Rectangular:
			targetShape = graphics.NewSquare(0, 0, qual)
		case graphics.Linear:
			targetShape = graphics.NewLine(0, 0, graphics.LineRight, qual)
		}

		r.TargetArea = targetShape
		weapon.AddComponent(gear.RangedWeaponComponent, &r)
		return weapon
	}

	return nil

}

// Basic spawning to start off with. Has a 30% chance to spawn a consumable in the center
func SpawnStartingConsumables(em common.EntityManager, gm *worldmap.GameMap) {
	spawnChance := 30

	for _, room := range gm.Rooms {

		roll := rand.Intn(spawnChance + 1)

		if roll < spawnChance {

			x, y := room.Center()

			randInd := rand.Intn(len(entitytemplates.ConsumableTemplates))

			wep := entitytemplates.CreateConsumableFromTemplate(em, entitytemplates.ConsumableTemplates[randInd])

			common.GetComponentType[*rendering.Renderable](wep, rendering.RenderableComponent).Visible = true

			pos := common.GetPosition(wep)

			pos.X = x + 1
			pos.Y = y + 1

			gm.AddEntityToTile(wep, pos)

		}
	}
}

// 30 percent chance to spawn either a melee or ranged weapon in a room
// Todo add these items to the players inventory instead and also add consumables
func SpawnStartingEquipment(em *common.EntityManager, gm *worldmap.GameMap, pl *avatar.PlayerData) {

	weaponChoice := -1
	weaponInd := -1

	for _, room := range gm.Rooms {

		if rand.Intn(100) < 50 {
			weaponChoice = rand.Intn(2)

			x, y := room.Center()

			if weaponChoice == 0 {

				weaponInd = rand.Intn(len(entitytemplates.MeleeWeaponTemplates))
				wep := entitytemplates.CreateMeleeWepFromTemplate(*em, entitytemplates.MeleeWeaponTemplates[weaponInd])
				common.GetComponentType[*rendering.Renderable](wep, rendering.RenderableComponent).Visible = true
				pos := common.GetPosition(wep)
				pos.X = x
				pos.Y = y
				gm.AddEntityToTile(wep, pos)

			} else if weaponChoice == 1 {

				weaponInd = rand.Intn(len(entitytemplates.RangedWeaponTemplates))
				wep := entitytemplates.CreateMeleeWepFromTemplate(*em, entitytemplates.MeleeWeaponTemplates[weaponInd])
				common.GetComponentType[*rendering.Renderable](wep, rendering.RenderableComponent).Visible = true
				pos := common.GetPosition(wep)
				pos.X = x
				pos.Y = y
				gm.AddEntityToTile(wep, pos)

			} else {
				// TODO: Handle starting equipment spawn error
			}

		}

	}
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
	getRandomEntry := func(posSlice []common.Position) common.Position {

		return posSlice[rand.Intn(len(posSlice))]

	}

	logicalPos := coords.LogicalPosition{X: playerData.Pos.X, Y: playerData.Pos.Y}
	pixelPos := coords.CoordManager.LogicalToPixel(logicalPos)
	playerX, playerY := pixelPos.X, pixelPos.Y
	spawnPositions := gm.UnblockedLogicalCoords(playerX, playerY, 10)
	consChance, throwableChance, rangedWepChance := rand.Intn(100), rand.Intn(100), rand.Intn(100)

	if consChance < ConsumableSpawnProb {

		pos := getRandomEntry(spawnPositions)
		e := SpawnConsumable(manager, pos.X, pos.Y)
		gm.AddEntityToTile(e, &common.Position{X: pos.X, Y: pos.Y})

	}
	if throwableChance < ThrowableSpawnProb {

		pos := getRandomEntry(spawnPositions)
		e := SpawnThrowableItem(manager, pos.X, pos.Y)
		gm.AddEntityToTile(e, &common.Position{X: pos.X, Y: pos.Y})

	}

	if rangedWepChance < RangedWeaponSpawnProb {

		pos := getRandomEntry(spawnPositions)
		e := SpawnRangedWeapon(manager, pos.X, pos.Y)
		gm.AddEntityToTile(e, &common.Position{X: pos.X, Y: pos.Y})

	}

}
