package storage_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type (
	discriminatedType interface {
		Discriminator() string
	}

	type1 struct {
		data string
	}

	type2 struct {
		data string
	}
)

func (t type1) Discriminator() string { return "type1" }
func (t type2) Discriminator() string { return "type2" }

var mapper = storage.NewDiscriminatedMapper(func(dt discriminatedType) string {
	return dt.Discriminator()
})

func Test_Discriminated(t *testing.T) {
	mapper.Register(type1{}, func(data string) (discriminatedType, error) { return type1{data}, nil })
	mapper.Register(type2{}, func(data string) (discriminatedType, error) { return type2{data}, nil })

	t.Run("should panic if a type is already registered with the same discriminator", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got none")
			}
		}()

		mapper.Register(type1{}, func(data string) (discriminatedType, error) { return type1{data}, nil })
	})

	t.Run("should error if the discriminator is not known", func(t *testing.T) {
		_, err := mapper.From("unknown", "")

		assert.ErrorIs(t, err, storage.ErrCouldNotUnmarshalGivenType)
	})

	t.Run("should return registered keys", func(t *testing.T) {
		assert.ArrayEqual(t, []string{"type1", "type2"}, mapper.Keys())
	})

	t.Run("should return the correct type", func(t *testing.T) {
		t1, err := mapper.From("type1", "data1")

		assert.Nil(t, err)
		assert.Equal(t, type1{"data1"}, t1.(type1))

		t2, err := mapper.From("type2", "data2")

		assert.Nil(t, err)
		assert.Equal(t, type2{"data2"}, t2.(type2))
	})
}
