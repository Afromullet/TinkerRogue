package spawning

import "math/rand"

// todo add comments on how to setup and use the probability table
type ProbabilityEntry[T any] struct {
	entry  T
	weight int
}

// totalWeight is the sum of all probabilities. Used for discrete random selection.
// AddEntry updates the totalWeight when a new entry is added.
type ProbabilityTable[T any] struct {
	table       []ProbabilityEntry[T]
	totalWeight int
}

func NewProbabilityTable[T any]() ProbabilityTable[T] {

	return ProbabilityTable[T]{
		table:       make([]ProbabilityEntry[T], 0),
		totalWeight: 0,
	}

}

// Keeps a running sum of the probability
func (lootTable *ProbabilityTable[T]) AddEntry(entry T, chance int) {

	lootEntry := ProbabilityEntry[T]{
		entry:  entry,
		weight: chance,
	}
	lootTable.table = append(lootTable.table, lootEntry)
	lootTable.totalWeight += chance

}

// Todo, this algorithm is from the internet. I don't really understand how works
// I need to understand how it works to see if it does what it claims to do
// Returns a false for the boolean parameter if an entry cannot be found for wahtever reason
// zeroizeWeight is used for when we want to set the weight of the selected item to zero.
// This is currently used for selecting random status effect properties on an item so
// That we don't select the same property more than once
func (lootTable *ProbabilityTable[T]) GetRandomEntry(zeroizeWeight bool) (T, bool) {

	var zerovalue T
	randVal := rand.Intn(lootTable.totalWeight)

	cursor := 0
	for ind, e := range lootTable.table {
		cursor += e.weight

		if cursor >= randVal {

			if zeroizeWeight {
				lootTable.table[ind].weight = 0
				lootTable.totalWeight -= e.weight

			}
			return e.entry, true

		}

	}

	return zerovalue, false

}
