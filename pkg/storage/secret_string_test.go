package storage_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_SecretString(t *testing.T) {
	t.Run("should correctly scan a string", func(t *testing.T) {
		var s storage.SecretString

		err := s.Scan("test")

		testutil.IsNil(t, err)
		testutil.Equals(t, "test", s)
	})

	t.Run("should marshal to a json string with the same length as the original string and custom characters", func(t *testing.T) {
		s := storage.SecretString("some random string")

		data, err := s.MarshalJSON()
		dataStr := string(data)

		testutil.IsNil(t, err)
		testutil.Equals(t, `"******************"`, dataStr)
	})

	t.Run("should keep newlines", func(t *testing.T) {
		s := storage.SecretString(`some random string
with a newline
and another one`)

		data, err := s.MarshalJSON()
		dataStr := string(data)

		testutil.IsNil(t, err)
		testutil.Equals(t, `"******************\n**************\n***************"`, dataStr)
	})
}
