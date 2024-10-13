package spawning

import "game_main/gear"

var LootQualityTable = NewProbabilityTable[int]()

var ThrowableProbTable = NewProbabilityTable[gear.StatusEffects]()
var RandThrowableProps = []gear.StatusEffects{gear.NewBurning(1, 1), gear.NewFreezing(1, 1), gear.NewSticky(1, 1)}

func InitThrowableSpawnTable() {
	ThrowableProbTable.AddEntry(gear.NewBurning(1, 1), 30)
	ThrowableProbTable.AddEntry(gear.NewFreezing(1, 1), 30)
	ThrowableProbTable.AddEntry(gear.NewSticky(1, 1), 30)
}

func InitLootSpawnTables() {

	LootQualityTable.AddEntry(gear.LowQuality, 50)
	LootQualityTable.AddEntry(gear.NormalQuality, 40)
	LootQualityTable.AddEntry(gear.HighQuality, 10)

	InitThrowableSpawnTable()

}
