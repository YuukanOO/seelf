package collections_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/collections"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_ToPointers(t *testing.T) {
	t.Run("should convert an array of items to an array of pointers of items", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5}
		pointers := collections.ToPointers(items)

		for idx, item := range items {
			testutil.Equals(t, item, *pointers[idx])
		}
	})
}
