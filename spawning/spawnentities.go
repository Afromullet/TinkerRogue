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

type ProbabilityEntry[T any] struct {
	entry  T
	weight float32
}

// totalWeight is the sum of all probabilities.
// Only add to this through addEntry, since that keeps a running
// sum of the weights
type ProbabilityTable[T any] struct {
	table       []ProbabilityEntry[T]
	totalWeight float32
}

// Function to return a ProbabilityTable
func NewProbabilityTable[T any]() ProbabilityTable[T] {

	return ProbabilityTable[T]{
		table:       make([]ProbabilityEntry[T], 0),
		totalWeight: 0,
	}

}

// Keeps a running sum of the probability
func (lootTable *ProbabilityTable[T]) AddEntry(entry T, chance float32) {

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
func (looktTable *ProbabilityTable[T]) GetRandomEntry() (T, bool) {

	var zerovalue T
	randVal := rand.Float32() * looktTable.totalWeight

	for _, e := range looktTable.table {

		if randVal < e.weight {
			return e.entry, false
		}
		randVal -= e.weight
	}

	return zerovalue, true

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
