package id_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type someDomainID string

func Test_ID_GeneratesANonEmptyUniqueIdentifier(t *testing.T) {
	id1 := id.New[someDomainID]()
	id2 := id.New[someDomainID]()

	testutil.HasNChars(t, 27, id1)
	testutil.HasNChars(t, 27, id2)
	testutil.NotEquals(t, id1, id2)
}
