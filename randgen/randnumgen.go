package randgen

import (
	"math/big"

	cryptorand "crypto/rand"
)

// Todo remove later once you change teh random number generation. The same function is in another aprt of the code
func GetDiceRoll(num int) int {
	x, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(num)))
	return int(x.Int64()) + 1

}

// Todo remove later once you change teh random number generation. The same function is in another aprt of the code
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
