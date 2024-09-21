package spawning

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/entitytemplates"
	"game_main/graphics"
	"game_main/monsters"

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
				pos := common.PositionFromIndex(index, gd.DungeonWidth)

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
		monsters.BehaviorSelector(ent, pl)

	}

	for range MaxNumCreatures {

		randCreature = rand.Intn(len(entitytemplates.MonsterTemplates))
		indices := gm.Rooms[rand.Intn(len(gm.Rooms))].GetCoordinatesWithoutCenter()
		randomPos := indices[rand.Intn(len(indices))]

		ent := entitytemplates.CreateCreatureFromTemplate(*em, entitytemplates.MonsterTemplates[randCreature], gm, randomPos.X, randomPos.Y)
		monsters.BehaviorSelector(ent, pl)
	}

}
