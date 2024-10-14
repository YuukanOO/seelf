package id_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/id"
)

type someDomainID string

func Test_ID_GeneratesANonEmptyUniqueIdentifier(t *testing.T) {
	id1 := id.New[someDomainID]()
	id2 := id.New[someDomainID]()

	assert.HasNRunes(t, 27, id1)
	assert.HasNRunes(t, 27, id2)
	assert.NotEqual(t, id1, id2)
}
