package common

import (
	"math/rand/v2"
)

// Global RNG source for deterministic, seedable randomness
var rng = rand.New(rand.NewPCG(1, 2))

// SetRNGSeed allows tests to seed the RNG for reproducible results
func SetRNGSeed(seed1, seed2 uint64) {
	rng = rand.New(rand.NewPCG(seed1, seed2))
}

// GetDiceRoll returns a random number between 1 and num (inclusive)
// If num <= 0, returns 1
func GetDiceRoll(num int) int {
	if num <= 0 {
		return 1
	}
	return rng.IntN(num) + 1
}

// GetRandomBetween returns a random number between low and high (inclusive)
// If low > high, they are swapped
// If low == high, returns that value
func GetRandomBetween(low int, high int) int {
	// Ensure low <= high
	if low > high {
		low, high = high, low
	}
	if low == high {
		return low
	}
	// Return low + random(0 to high-low)
	return low + rng.IntN(high-low+1)
}

// RandomInt returns a random integer in the range [0, n)
func RandomInt(n int) int {
	if n <= 0 {
		return 0
	}
	return rng.IntN(n)
}

// RandomFloat returns a random float64 in the range [0.0, 1.0)
func RandomFloat() float64 {
	return rng.Float64()
}
