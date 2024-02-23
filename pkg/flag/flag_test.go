package flag_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/flag"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type flagType uint

const (
	flagA flagType = 1 << iota
	flagB
	flagC
)

func Test_IsSet(t *testing.T) {
	testutil.IsTrue(t, flag.IsSet(flagA, flagA))
	testutil.IsFalse(t, flag.IsSet(flagA, flagB))
	testutil.IsTrue(t, flag.IsSet(flagA|flagB, flagA))
	testutil.IsTrue(t, flag.IsSet(flagA|flagB, flagB|flagA))
	testutil.IsTrue(t, flag.IsSet(flagA|flagB|flagC, flagB|flagA))
	testutil.IsFalse(t, flag.IsSet(flagA, flagB|flagA))
	testutil.IsFalse(t, flag.IsSet(flagA|flagC, flagB|flagA))
}
