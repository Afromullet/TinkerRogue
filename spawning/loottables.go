package spawning

import (
	"game_main/gear"
	"game_main/graphics"
)

var LootQualityTable = NewProbabilityTable[gear.Quality]()

var ThrowableProbTable = NewProbabilityTable[gear.StatusEffects]()
var RandThrowableOptions = []gear.StatusEffects{gear.NewBurning(1, 1), gear.NewFreezing(1, 1), gear.NewSticky(1, 1)}

var ThrowableAOE = NewProbabilityTable[graphics.TileBasedShape]()

func InitThrowableSpawnTable() {
	ThrowableProbTable.AddEntry(gear.NewBurning(1, 1), 30)
	ThrowableProbTable.AddEntry(gear.NewFreezing(1, 1), 20)
	ThrowableProbTable.AddEntry(gear.NewSticky(1, 1), 10)
}

func InitLootSpawnTables() {

	LootQualityTable.AddEntry(gear.LowQuality, 50)
	LootQualityTable.AddEntry(gear.NormalQuality, 40)
	LootQualityTable.AddEntry(gear.HighQuality, 10)

	InitThrowableSpawnTable()

}
