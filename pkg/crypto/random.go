package crypto

import (
	"crypto/rand"
	"math/big"
)

const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Generates a random key (from: https://gist.github.com/dopey/c69559607800d2f2f90b1b1ed4e550fb)
func RandomKey[T ~string](lengthInBytes int) (T, error) {
	ret := make([]byte, lengthInBytes)
	for i := 0; i < lengthInBytes; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return T(ret), nil
}
