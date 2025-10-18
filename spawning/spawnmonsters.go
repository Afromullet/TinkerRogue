package spawning

import (
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"
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

			index := common.GetRandomBetween(0, len(worldmap.ValidPos.Pos)-1)

			if !gm.Tiles[index].Blocked {

				logicalPos := coords.CoordManager.IndexToLogical(index)

				entitytemplates.CreateEntityFromTemplate(ecsmanager, entitytemplates.EntityConfig{
					Type:      entitytemplates.EntityCreature,
					Name:      entitytemplates.MonsterTemplates[0].Name,
					ImagePath: entitytemplates.MonsterTemplates[0].ImageName,
					AssetDir:  "../assets/creatures/",
					Visible:   true,
					Position:  &coords.LogicalPosition{X: logicalPos.X, Y: logicalPos.Y},
					GameMap:   gm,
				}, entitytemplates.MonsterTemplates[0])
				gm.Tiles[index].Blocked = true

				break

			}

		}

	}

}

// Spawns one creature in every room.
// Then it spawns MaxNumCreatures in random rooms
func SpawnStartingCreatures(MaxNumCreatures int, em *common.EntityManager, gm *worldmap.GameMap, pl *common.PlayerData) {

	//Spawn 1 random creature in every room except the starting room

	randCreature := 0

	for _, room := range gm.Rooms[1:] {

		x, y := room.Center()
		randCreature = rand.Intn(len(entitytemplates.MonsterTemplates))
		entitytemplates.CreateEntityFromTemplate(*em, entitytemplates.EntityConfig{
			Type:      entitytemplates.EntityCreature,
			Name:      entitytemplates.MonsterTemplates[randCreature].Name,
			ImagePath: entitytemplates.MonsterTemplates[randCreature].ImageName,
			AssetDir:  "../assets/creatures/",
			Visible:   true,
			Position:  &coords.LogicalPosition{X: x, Y: y},
			GameMap:   gm,
		}, entitytemplates.MonsterTemplates[randCreature])

	}

	for range MaxNumCreatures {

		randCreature = rand.Intn(len(entitytemplates.MonsterTemplates))
		indices := gm.Rooms[rand.Intn(len(gm.Rooms))].GetCoordinatesWithoutCenter()
		randomPos := indices[rand.Intn(len(indices))]

		entitytemplates.CreateEntityFromTemplate(*em, entitytemplates.EntityConfig{
			Type:      entitytemplates.EntityCreature,
			Name:      entitytemplates.MonsterTemplates[randCreature].Name,
			ImagePath: entitytemplates.MonsterTemplates[randCreature].ImageName,
			AssetDir:  "../assets/creatures/",
			Visible:   true,
			Position:  &coords.LogicalPosition{X: randomPos.X, Y: randomPos.Y},
			GameMap:   gm,
		}, entitytemplates.MonsterTemplates[randCreature])

	}

}
