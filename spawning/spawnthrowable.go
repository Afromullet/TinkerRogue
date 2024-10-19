package spawning

import (
	"fmt"
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"
	"math/rand"

	"github.com/bytearena/ecs"
)

// TODO define some of these parameters in a config file

// Todo, this is only here. Need to find a better way to initialize this,
// so that we do not have to change this variable whenver we add a new status effect

// The number of item properties generated for a throwable using the loot tables
func RandomNumProperties() int {

	len := rand.Intn(len(RandThrowableOptions))
	if len == 0 {
		len = 1
	}
	return len
}

func SpawnThrowableItem(manager *ecs.Manager, xPos, yPos int) *ecs.Entity {

	effects := make([]gear.StatusEffects, 0)

	itemName := ""

	for _ = range RandomNumProperties() {

		if entry, ok := ThrowableEffectStatTable.GetRandomEntry(true); ok {

			qual, qualOK := LootQualityTable.GetRandomEntry(false)
			if qualOK {
				entry.CreateWithQuality(qual)
				effects = append(effects, entry)
				itemName += entry.StatusEffectName() //Todo need better way to create a name
			}

		} else {
			fmt.Println("Error spawning throwable")
		}

	}

	aoeShape, shapeOK := ThrowableAOEProbTable.GetRandomEntry(false)
	qual, qualOK := LootQualityTable.GetRandomEntry(false)

	//Select a random visual effect

	randInd := len(effects)

	var vx graphics.VisualEffect
	if randInd == 0 {

		vx = gear.GetVisualEffect(effects[0])

	} else {
		vx = gear.GetVisualEffect(effects[rand.Intn(randInd)])

	}

	throwable := gear.NewThrowable(1, 1, 1, aoeShape)
	if shapeOK && qualOK {

		throwable.CreateWithQuality(qual)

		//Select a random

		//Need to set this again due to how the CreateWithQuality is implemented.
		//It's a problem that I have to take the extra step, but it's not something worth worrying about for now.
		//The problem being that CreateWithQuality takes a reference rather than creating a new type.
		aoeShape.CreateWithQuality(qual)
		throwable.Shape = aoeShape
		throwable.VX = vx

		effects = append(effects, throwable)

	} else {
		fmt.Println("Problem generating AOE shape")
	}
	ThrowableEffectStatTable.RestoreWeights()
	return gear.CreateItem(manager, itemName, common.Position{X: xPos, Y: yPos}, "../assets/items/grenade.png", effects...)

}
