package spawning

import (
	"fmt"
	"game_main/common"
	"game_main/entitytemplates"
	"game_main/rendering"
	"game_main/worldmap"
	"math/rand/v2"
)

// Basic spawning to start off with. Has a 30% chance to spawn a random item near the center
func SpawnStartingLoot(em common.EntityManager, gm *worldmap.GameMap) {
	spawnChance := 30

	for _, room := range gm.Rooms {

		roll := rand.IntN(spawnChance + 1)

		if roll < spawnChance {

			fmt.Println("Spawning")
			x, y := room.Center()

			wep := entitytemplates.CreateMeleeWepFromTemplate(em, entitytemplates.MeleeWeaponTemplates[0])

			common.GetComponentType[*rendering.Renderable](wep, rendering.RenderableComponent).Visible = true

			pos := common.GetPosition(wep)

			pos.X = x + 1
			pos.Y = y + 1

			gm.AddEntityToTile(wep, pos)

		}
	}
}
