package spawning

import (
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"
)

// Type and variables for overall spawn probability of the kinds of items.
// Spawned every N number of turns

var ConsumableSpawnProb = 30
var ThrowableSpawnProb = 30
var RangedWeaponSpawnProb = 10

// Used for helping select n number of effects that throwables will apply to targets. Not tied to quality.
var RandThrowableOptions = []gear.StatusEffects{gear.NewBurning(1, 1), gear.NewFreezing(1, 1), gear.NewSticky(1, 1)}

var LootQualityTable = NewProbabilityTable[common.QualityType]()           //Determining the quality of the item to be generated
var ThrowableEffectStatTable = NewProbabilityTable[gear.StatusEffects]()   //Determines the effects that throwables apply to targets
var ThrowableAOEProbTable = NewProbabilityTable[graphics.BasicShapeType]() //Determines the AOE of throwables
var ConsumableSpawnTable = NewProbabilityTable[gear.ConsumableType]()

func InitLootSpawnTables() {

	//Todo don't think I have to use the constructor
	ThrowableEffectStatTable.AddEntry(gear.NewBurning(1, 1), 30)
	ThrowableEffectStatTable.AddEntry(gear.NewFreezing(1, 1), 20)
	ThrowableEffectStatTable.AddEntry(gear.NewSticky(1, 1), 10)

	LootQualityTable.AddEntry(common.LowQuality, 50)
	LootQualityTable.AddEntry(common.NormalQuality, 40)
	LootQualityTable.AddEntry(common.HighQuality, 10)

	ThrowableAOEProbTable.AddEntry(graphics.Linear, 30)
	ThrowableAOEProbTable.AddEntry(graphics.Rectangular, 20)
	ThrowableAOEProbTable.AddEntry(graphics.Circular, 10)
	ThrowableAOEProbTable.AddEntry(graphics.Linear, 15)
	ThrowableAOEProbTable.AddEntry(graphics.Rectangular, 5)

	ConsumableSpawnTable.AddEntry(gear.HealingPotion, 50)
	ConsumableSpawnTable.AddEntry(gear.ProtectionPotion, 30)
	ConsumableSpawnTable.AddEntry(gear.SpeedPotion, 20)

}
