package spawning

import (
	"game_main/common"
	entitytemplates "game_main/datareader"
	"game_main/graphics"

	//"game_main/entitytemplates"

	"game_main/randgen"
	"game_main/worldmap"
	"math/rand"
	"time"
)

var TurnsPerMonsterSpawn = 10

// Basic monster spawning function that spawns a monster on a random tile
func SpawnMonster(ecsmanager common.EntityManager, gm *worldmap.GameMap) {
	gd := graphics.NewScreenData()
	rand.Seed(time.Now().UnixNano())
	if rand.Intn(100) < 30 { // 30% chance to spawn something

		//Try 3 times to spawn something. Only spawn it if the tile is not blocked
		for i := 0; i <= 2; i++ {

			index := randgen.GetRandomBetween(0, len(worldmap.ValidPos.Pos)-1)

			if !gm.Tiles[index].Blocked {
				pos := common.PositionFromIndex(index, gd.ScreenWidth)

				entitytemplates.CreateCreatureFromTemplate(ecsmanager, entitytemplates.MonsterTemplates[0], gm, pos.X, pos.Y)
				gm.Tiles[index].Blocked = true

				break

			}

		}

	}

}
