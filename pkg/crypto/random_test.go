package crypto_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/crypto"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_RandomKey(t *testing.T) {
	key, err := crypto.RandomKey[string](32)
	testutil.IsNil(t, err)
	testutil.HasNChars(t, 32, key)
}
