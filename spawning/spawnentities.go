package spawning

import (
	"game_main/common"
	"game_main/entitytemplates"
	"game_main/graphics"

	//"game_main/entitytemplates"

	"game_main/randgen"
	"game_main/worldmap"
	"math/rand"
	"time"
)

// todo add comments on how to setup and use the probability table
type ProbabilityEntry[T any] struct {
	entry  T
	weight int
}

// totalWeight is the sum of all probabilities. Used for discrete random selection.
// AddEntry updates the totalWeight when a new entry is added.
type ProbabilityTable[T any] struct {
	table       []ProbabilityEntry[T]
	totalWeight int
}

func NewProbabilityTable[T any]() ProbabilityTable[T] {

	return ProbabilityTable[T]{
		table:       make([]ProbabilityEntry[T], 0),
		totalWeight: 0,
	}

}

// Keeps a running sum of the probability
func (lootTable *ProbabilityTable[T]) AddEntry(entry T, chance int) {

	lootEntry := ProbabilityEntry[T]{
		entry:  entry,
		weight: chance,
	}
	lootTable.table = append(lootTable.table, lootEntry)
	lootTable.totalWeight += chance

}

// Todo, this algorithm is from the internet. I don't really understand how works
// I need to understand how it works to see if it does what it claims to do
// Returns a false for the boolean parameter if an entry cannot be found for wahtever reason
func (lootTable *ProbabilityTable[T]) GetRandomEntry() (T, bool) {

	var zerovalue T
	randVal := rand.Intn(lootTable.totalWeight)

	cursor := 0
	for _, e := range lootTable.table {
		cursor += e.weight

		if cursor >= randVal {
			return e.entry, true

		}

	}

	return zerovalue, false

}

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
