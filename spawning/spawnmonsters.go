package spawning

import (
	"game_main/avatar"
	"game_main/behavior"
	"game_main/common"
	"game_main/entitytemplates"
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

	rand.Seed(time.Now().UnixNano())
	//Half-open interval - so it includes 0 but not 100. Since it includes 0, we still have 100 possible vlaues
	if rand.Intn(100) < 30 { // 30% chance to spawn something

		//Try 3 times to spawn something. Only spawn it if the tile is not blocked
		for i := 0; i <= 2; i++ {

			index := randgen.GetRandomBetween(0, len(worldmap.ValidPos.Pos)-1)

			if !gm.Tiles[index].Blocked {

				x, y := graphics.CoordTransformer.LogicalXYFromIndex(index)
				pos := common.Position{X: x, Y: y}

				entitytemplates.CreateCreatureFromTemplate(ecsmanager, entitytemplates.MonsterTemplates[0], gm, pos.X, pos.Y)
				gm.Tiles[index].Blocked = true

				break

			}

		}

	}

}

// Spawns one creature in every room.
// Then it spawns MaxNumCreatures in random rooms
func SpawnStartingCreatures(MaxNumCreatures int, em *common.EntityManager, gm *worldmap.GameMap, pl *avatar.PlayerData) {

	//Spawn 1 random creature in every room except the starting room

	randCreature := 0

	for _, room := range gm.Rooms[1:] {

		x, y := room.Center()
		randCreature = rand.Intn(len(entitytemplates.MonsterTemplates))
		ent := entitytemplates.CreateCreatureFromTemplate(*em, entitytemplates.MonsterTemplates[randCreature], gm, x, y)
		behavior.BehaviorSelector(ent, pl)

	}

	for range MaxNumCreatures {

		randCreature = rand.Intn(len(entitytemplates.MonsterTemplates))
		indices := gm.Rooms[rand.Intn(len(gm.Rooms))].GetCoordinatesWithoutCenter()
		randomPos := indices[rand.Intn(len(indices))]

		ent := entitytemplates.CreateCreatureFromTemplate(*em, entitytemplates.MonsterTemplates[randCreature], gm, randomPos.X, randomPos.Y)
		behavior.BehaviorSelector(ent, pl)
	}

}
