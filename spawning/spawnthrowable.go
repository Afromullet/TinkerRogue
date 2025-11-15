package spawning

import (
	"game_main/common"
	"game_main/coords"
	"game_main/gear"
	"game_main/graphics"

	"github.com/bytearena/ecs"
)

// TODO define some of these parameters in a config file

// Todo, this is only here. Need to find a better way to initialize this,
// so that we do not have to change this variable whenver we add a new status effect

// The number of item properties generated for a throwable using the loot tables
func RandomNumProperties() int {

	len := common.RandomInt(len(RandThrowableOptions))
	if len == 0 {
		len = 1
	}
	return len
}

func SpawnThrowableItem(manager *ecs.Manager, xPos, yPos int) *ecs.Entity {

	// Effects that will be applied when throwable is used (not item properties)
	effectsToApply := make([]gear.StatusEffects, 0)

	itemName := ""

	for _ = range RandomNumProperties() {

		if entry, ok := ThrowableEffectStatTable.GetRandomEntry(true); ok {

			qual, qualOK := LootQualityTable.GetRandomEntry(false)
			if qualOK {
				entry.CreateWithQuality(qual)
				effectsToApply = append(effectsToApply, entry)
				itemName += entry.StatusEffectName() // Todo need better way to create a name
			}

		} else {
			// TODO: Handle throwable spawn error
		}

	}

	shapeType, shapeOK := ThrowableAOEProbTable.GetRandomEntry(false)
	qual, qualOK := LootQualityTable.GetRandomEntry(false)

	// Select a random visual effect based on the effects to apply
	var vx graphics.VisualEffect
	if len(effectsToApply) > 0 {
		if len(effectsToApply) == 1 {
			vx = gear.GetVisualEffect(effectsToApply[0])
		} else {
			vx = gear.GetVisualEffect(effectsToApply[common.RandomInt(len(effectsToApply))])
		}
	}

	if shapeOK && qualOK {
		// Create throwable action with the effects it will apply to targets
		throwableAction := gear.NewShapeThrowableAction(1, 1, 1, shapeType, qual, nil, effectsToApply...)
		throwableAction.VX = vx

		// Create item with throwable action - no status effects as item properties
		actions := []gear.ItemAction{throwableAction}

		ThrowableEffectStatTable.RestoreWeights()
		return gear.CreateItemWithActions(manager, itemName, coords.LogicalPosition{X: xPos, Y: yPos}, "../assets/items/grenade.png", actions)

	} else {
		// TODO: Handle AOE shape generation error
		ThrowableEffectStatTable.RestoreWeights()
		// Even in error case, don't create old-style items with effects as properties
		// Create a basic throwable action instead
		basicThrowable := gear.NewShapeThrowableAction(1, 3, 1, graphics.Circular, common.NormalQuality, nil, effectsToApply...)
		actions := []gear.ItemAction{basicThrowable}
		return gear.CreateItemWithActions(manager, itemName, coords.LogicalPosition{X: xPos, Y: yPos}, "../assets/items/grenade.png", actions)
	}

}
