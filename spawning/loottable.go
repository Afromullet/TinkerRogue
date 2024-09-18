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
func (lootTable *ProbabilityTable[T]) GetRandomEntry() (T, bool) {

	var zerovalue T
	randVal := rand.Intn(lootTable.totalWeight)

	cursor := 0
	for _, e := range lootTable.table {
		cursor += e.weight

		if cursor >= randVal {
			return e.entry, true

		}

	}

	return zerovalue, false

}
