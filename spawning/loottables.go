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

// Used for helping is select n number of properties for the status effect. Not tied to quality.
var RandThrowableOptions = []gear.StatusEffects{gear.NewBurning(1, 1), gear.NewFreezing(1, 1), gear.NewSticky(1, 1)}

var LootQualityTable = NewProbabilityTable[common.QualityType]()           //Determining the quality of the item to be generated
var ThrowableEffectStatTable = NewProbabilityTable[gear.StatusEffects]()   //Determines the status effects of throwables
var ThrowableAOEProbTable = NewProbabilityTable[graphics.TileBasedShape]() //Determines the AOE of throwables
var ConsumableSpawnTable = NewProbabilityTable[gear.ConsumableType]()

func InitLootSpawnTables() {

	//Todo don't think I have to use the constructor
	ThrowableEffectStatTable.AddEntry(gear.NewBurning(1, 1), 30)
	ThrowableEffectStatTable.AddEntry(gear.NewFreezing(1, 1), 20)
	ThrowableEffectStatTable.AddEntry(gear.NewSticky(1, 1), 10)

	LootQualityTable.AddEntry(common.LowQuality, 50)
	LootQualityTable.AddEntry(common.NormalQuality, 40)
	LootQualityTable.AddEntry(common.HighQuality, 10)

	ThrowableAOEProbTable.AddEntry(&graphics.TileLine{}, 30)
	ThrowableAOEProbTable.AddEntry(&graphics.TileSquare{}, 20)
	ThrowableAOEProbTable.AddEntry(&graphics.TileCircle{}, 10)
	ThrowableAOEProbTable.AddEntry(&graphics.TileCone{}, 15)
	ThrowableAOEProbTable.AddEntry(&graphics.TileRectangle{}, 5)

	ConsumableSpawnTable.AddEntry(gear.HealingPotion, 50)
	ConsumableSpawnTable.AddEntry(gear.ProtectionPotion, 30)
	ConsumableSpawnTable.AddEntry(gear.SpeedPotion, 20)

}
