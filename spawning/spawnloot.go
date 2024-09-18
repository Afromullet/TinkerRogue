package spawning

import (
	"fmt"
	"game_main/avatar"
	"game_main/common"
	"game_main/entitytemplates"
	"game_main/rendering"
	"game_main/worldmap"
	"math/rand"
)

// Basic spawning to start off with. Has a 30% chance to spawn a consumable in the center
func SpawnStartingConsumables(em common.EntityManager, gm *worldmap.GameMap) {
	spawnChance := 30

	for _, room := range gm.Rooms {

		roll := rand.Intn(spawnChance + 1)

		if roll < spawnChance {

			fmt.Println("Spawning")
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
				fmt.Println("Error spawning starting equipmen")
			}

		}

	}
}
