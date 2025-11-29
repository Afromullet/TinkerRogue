package spawning

import (
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"
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

	// 1. Generate effects using existing loot tables (procedural generation logic)
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

	// 2. Generate AOE shape and quality
	shapeType, shapeOK := ThrowableAOEProbTable.GetRandomEntry(false)
	qual, qualOK := LootQualityTable.GetRandomEntry(false)

	// 3. Select a random visual effect based on the effects to apply
	var vx graphics.VisualEffect
	if len(effectsToApply) > 0 {
		if len(effectsToApply) == 1 {
			vx = gear.GetVisualEffect(effectsToApply[0])
		} else {
			vx = gear.GetVisualEffect(effectsToApply[common.RandomInt(len(effectsToApply))])
		}
	}

	pos := coords.LogicalPosition{X: xPos, Y: yPos}

	// 4. Delegate entity creation to entitytemplates
	if shapeOK && qualOK {
		ThrowableEffectStatTable.RestoreWeights()
		return entitytemplates.CreateThrowable(
			common.EntityManager{World: manager},
			itemName,
			pos,
			effectsToApply,
			shapeType,
			qual,
			vx,
		)
	} else {
		// Fallback with basic throwable
		ThrowableEffectStatTable.RestoreWeights()
		return entitytemplates.CreateThrowable(
			common.EntityManager{World: manager},
			itemName,
			pos,
			effectsToApply,
			graphics.Circular,
			common.NormalQuality,
			vx,
		)
	}

}
