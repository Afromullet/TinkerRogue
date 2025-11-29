package spawning

import (
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"
	"game_main/worldmap"
)

var TurnsPerMonsterSpawn = 10

// selectValidSpawnPosition attempts to find an unblocked tile for spawning.
// Returns nil if no valid position found after maxAttempts.
//
// This is pure spawn logic - separated from entity creation.
func selectValidSpawnPosition(gm *worldmap.GameMap, maxAttempts int) *coords.LogicalPosition {
	for i := 0; i < maxAttempts; i++ {
		index := common.GetRandomBetween(0, len(gm.ValidPositions)-1)
		if !gm.Tiles[index].Blocked {
			pos := coords.CoordManager.IndexToLogical(index)
			return &pos
		}
	}
	return nil
}

// Basic monster spawning function that spawns a monster on a random tile
func SpawnMonster(ecsmanager common.EntityManager, gm *worldmap.GameMap) {
	// 30% chance to spawn something
	if common.RandomInt(100) >= 30 {
		return
	}

	// Try 3 times to find valid spawn position
	pos := selectValidSpawnPosition(gm, 3)
	if pos == nil {
		return
	}

	// Use type-specific factory - handles ALL entity construction
	template := entitytemplates.MonsterTemplates[0]
	entitytemplates.CreateMonster(ecsmanager, gm, *pos, template)
}

// Spawns one creature in every room.
// Then it spawns MaxNumCreatures in random rooms
func SpawnStartingCreatures(MaxNumCreatures int, em *common.EntityManager, gm *worldmap.GameMap, pl *common.PlayerData) {

	// Spawn 1 random creature in every room except the starting room (room-based generators only)
	if len(gm.Rooms) >= 2 {
		for _, room := range gm.Rooms[1:] {
			x, y := room.Center()
			pos := coords.LogicalPosition{X: x, Y: y}

			randCreature := common.RandomInt(len(entitytemplates.MonsterTemplates))
			template := entitytemplates.MonsterTemplates[randCreature]

			entitytemplates.CreateMonster(*em, gm, pos, template)
		}
	}

	// Spawn additional creatures in random locations
	for range MaxNumCreatures {
		randCreature := common.RandomInt(len(entitytemplates.MonsterTemplates))
		template := entitytemplates.MonsterTemplates[randCreature]
		var randomPos coords.LogicalPosition

		// Spawn in rooms if they exist, otherwise use ValidPositions
		if len(gm.Rooms) > 0 {
			indices := gm.Rooms[common.RandomInt(len(gm.Rooms))].GetCoordinatesWithoutCenter()
			if len(indices) > 0 {
				randomPos = indices[common.RandomInt(len(indices))]
			} else {
				// Room has no valid positions, fall back to ValidPositions
				if len(gm.ValidPositions) > 0 {
					idx := common.GetRandomBetween(0, len(gm.ValidPositions)-1)
					randomPos = gm.ValidPositions[idx]
				} else {
					continue // Skip if no valid positions
				}
			}
		} else {
			// Non-room generators: spawn at random valid position
			if len(gm.ValidPositions) > 0 {
				idx := common.GetRandomBetween(0, len(gm.ValidPositions)-1)
				randomPos = gm.ValidPositions[idx]
			} else {
				continue // Skip if no valid positions
			}
		}

		entitytemplates.CreateMonster(*em, gm, randomPos, template)
	}

}
