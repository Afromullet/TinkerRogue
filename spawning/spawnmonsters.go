package spawning

import (
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"
	"game_main/worldmap"
)

var TurnsPerMonsterSpawn = 10

// Basic monster spawning function that spawns a monster on a random tile
func SpawnMonster(ecsmanager common.EntityManager, gm *worldmap.GameMap) {

	// 30% chance to spawn something
	if common.RandomInt(100) < 30 {

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

	//Spawn 1 random creature in every room except the starting room (room-based generators only)
	if len(gm.Rooms) >= 2 {
		randCreature := 0

		for _, room := range gm.Rooms[1:] {

			x, y := room.Center()
			randCreature = common.RandomInt(len(entitytemplates.MonsterTemplates))
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
	}

	// Spawn additional creatures in random locations
	for range MaxNumCreatures {

		randCreature := common.RandomInt(len(entitytemplates.MonsterTemplates))
		var randomPos coords.LogicalPosition

		// Spawn in rooms if they exist, otherwise use ValidPositions
		if len(gm.Rooms) > 0 {
			indices := gm.Rooms[common.RandomInt(len(gm.Rooms))].GetCoordinatesWithoutCenter()
			if len(indices) > 0 {
				randomPos = indices[common.RandomInt(len(indices))]
			} else {
				// Room has no valid positions, fall back to ValidPositions
				if len(worldmap.ValidPos.Pos) > 0 {
					idx := common.GetRandomBetween(0, len(worldmap.ValidPos.Pos)-1)
					randomPos = worldmap.ValidPos.Pos[idx]
				} else {
					continue // Skip if no valid positions
				}
			}
		} else {
			// Non-room generators: spawn at random valid position
			if len(worldmap.ValidPos.Pos) > 0 {
				idx := common.GetRandomBetween(0, len(worldmap.ValidPos.Pos)-1)
				randomPos = worldmap.ValidPos.Pos[idx]
			} else {
				continue // Skip if no valid positions
			}
		}

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
