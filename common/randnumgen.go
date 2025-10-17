package common

import (
	"math/big"

	cryptorand "crypto/rand"
)

// GetDiceRoll returns a random number between 1 and num (inclusive)
func GetDiceRoll(num int) int {
	x, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(num)))
	return int(x.Int64()) + 1

}

// GetRandomBetween returns a random number between low and high (inclusive)
func GetRandomBetween(low int, high int) int {
	var randy int = -1
	for {
		randy = GetDiceRoll(high)
		if randy >= low {
			break
		}
	}
	return randy
}
