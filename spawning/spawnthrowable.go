package spawning

import (
	"fmt"
	"math/rand"
)

// TODO define some of these parameters in a config file

// Todo, this is only here. Need to find a better way to initialize this,
// so that we do not have to change this variable whenver we add a new status effect

// The number of item properties generated for a throwable using the loot tables
func RandomNumProperties() int {
	return rand.Intn(len(RandThrowableProps))
}

func SpawnThrowableItem() {

	tempSpawner := ThrowableProbTable
	for _ = range RandomNumProperties() {

		if entry, ok := tempSpawner.GetRandomEntry(true); ok {

			qual, _ := LootQualityTable.GetRandomEntry(false)

			fmt.Println(qual)

			fmt.Println(entry)

		} else {
			fmt.Println("Error spawning throwable")
		}

	}

}
