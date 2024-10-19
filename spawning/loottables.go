package spawning

import (
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"
)

var LootQualityTable = NewProbabilityTable[common.QualityType]()

var ThrowableEffectStatTable = NewProbabilityTable[gear.StatusEffects]()

// Used for helping is select n number of properties for the status effect. Not tied to quality.
var RandThrowableOptions = []gear.StatusEffects{gear.NewBurning(1, 1), gear.NewFreezing(1, 1), gear.NewSticky(1, 1)}

var ThrowableAOEProbTable = NewProbabilityTable[graphics.TileBasedShape]()

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

}
