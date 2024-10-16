package spawning

import (
	"fmt"
	"game_main/common"
	"game_main/gear"
	"math/rand"

	"github.com/bytearena/ecs"
)

// TODO define some of these parameters in a config file

// Todo, this is only here. Need to find a better way to initialize this,
// so that we do not have to change this variable whenver we add a new status effect

// The number of item properties generated for a throwable using the loot tables
func RandomNumProperties() int {
	return rand.Intn(len(RandThrowableOptions))
}

// T
func SpawnThrowableItem(manager *ecs.Manager, xPos, yPos int) *ecs.Entity {

	effects := make([]gear.StatusEffects, 0)

	//numbers := []gear.StatusEffects{1, 2, 3, 4, 5}

	for _ = range RandomNumProperties() {

		//	var effect gear.StatusEffects // This is just a nil interface for now

		//eff := gear.StatusEffects{}
		if entry, ok := ThrowableProbTable.GetRandomEntry(true); ok {

			qual, qualOK := LootQualityTable.GetRandomEntry(false)

			if qualOK {
				entry.CreateWithQuality(qual)
				effects = append(effects, entry)
			}

		} else {
			fmt.Println("Error spawning throwable")
		}

	}

	ThrowableProbTable.RestoreWeights()

	return gear.CreateItem(manager, "po", common.Position{X: xPos, Y: yPos}, "../assets/items/sword.png", effects...)

}
