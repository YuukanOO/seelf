package monad_test

import (
	"testing"
	"time"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/monad"
	"gopkg.in/yaml.v3"
)

func Test_Maybe(t *testing.T) {
	t.Run("should have a default state without value", func(t *testing.T) {
		var m monad.Maybe[time.Time]

		assert.False(t, m.HasValue())
	})

	t.Run("could be created empty", func(t *testing.T) {
		m := monad.None[time.Time]()
		assert.False(t, m.HasValue())
	})

	t.Run("could be created with a defined value", func(t *testing.T) {
		m := monad.Value("ok")

		assert.Equal(t, "ok", m.MustGet())
		assert.True(t, m.HasValue())
	})

	t.Run("could returns its internal value and a boolean indicating if it has been set", func(t *testing.T) {
		var m monad.Maybe[string]

		value, hasValue := m.TryGet()

		assert.False(t, hasValue)
		assert.Equal(t, "", value)

		m.Set("ok")

		value, hasValue = m.TryGet()

		assert.True(t, hasValue)
		assert.Equal(t, "ok", value)
	})

	t.Run("could be assigned a value", func(t *testing.T) {
		var (
			m   monad.Maybe[time.Time]
			now = time.Now().UTC()
		)

		m.Set(now)
		assert.Equal(t, now, m.MustGet())
		assert.True(t, m.HasValue())
	})

	t.Run("could unset its value", func(t *testing.T) {
		m := monad.Value(time.Now().UTC())

		m.Unset()

		assert.False(t, m.HasValue())
	})

	t.Run("should panic if trying to access a value with MustGet", func(t *testing.T) {
		defer func() {
			err := recover()
			assert.NotNil(t, err)
			assert.Equal(t, "trying to access a monad's value but none is set", err.(string))
		}()

		var m monad.Maybe[time.Time]

		m.MustGet()
	})

	t.Run("could returns its value if its set", func(t *testing.T) {
		var now = time.Now().UTC()

		m := monad.Value(now)

		assert.Equal(t, now, m.MustGet())
	})

	t.Run("could returns its value or fallback if not set", func(t *testing.T) {
		var (
			woValue monad.Maybe[string]
			wValue  = monad.Value("got a value")
		)

		assert.Equal(t, "got a value", wValue.Get("default"))
		assert.Equal(t, "default", woValue.Get("default"))
	})

	t.Run("should implements the valuer interface", func(t *testing.T) {
		var m monad.Maybe[time.Time]

		driverValue, err := m.Value()

		assert.Nil(t, err)
		assert.Nil(t, driverValue)

		now := time.Now().UTC()
		m.Set(now)
		driverValue, err = m.Value()

		assert.Nil(t, err)
		assert.True(t, driverValue == now)
	})

	t.Run("should implements the Scanner interface", func(t *testing.T) {
		var m monad.Maybe[string]

		err := m.Scan(nil)

		assert.Nil(t, err)
		assert.False(t, m.HasValue())

		err = m.Scan("data")

		assert.Nil(t, err)
		assert.True(t, m.HasValue())
		assert.Equal(t, "data", m.MustGet())
	})

	t.Run("should correctly marshal to json", func(t *testing.T) {
		var m monad.Maybe[string]

		data, err := m.MarshalJSON()

		assert.Nil(t, err)
		assert.Equal(t, "null", string(data))

		m.Set("ok")

		data, err = m.MarshalJSON()

		assert.Nil(t, err)
		assert.Equal(t, `"ok"`, string(data))
	})

	t.Run("should correctly marshal to yaml", func(t *testing.T) {
		var m monad.Maybe[string]

		data, err := m.MarshalYAML()

		assert.Nil(t, err)
		assert.True(t, m.IsZero())
		assert.Nil(t, data)

		m.Set("ok")

		data, err = m.MarshalYAML()

		assert.Nil(t, err)
		assert.False(t, m.IsZero())
		assert.Equal(t, "ok", data)
	})

	t.Run("should correctly unmarshal from yaml", func(t *testing.T) {
		var m monad.Maybe[string]

		err := m.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "ok"})

		assert.Nil(t, err)
		assert.True(t, m.HasValue())
		assert.Equal(t, "ok", m.MustGet())
	})

	t.Run("should correctly unmarshal from env variables", func(t *testing.T) {
		var m monad.Maybe[string]

		err := m.UnmarshalEnvironmentValue("")

		assert.Nil(t, err)
		assert.False(t, m.HasValue())

		err = m.UnmarshalEnvironmentValue("ok")
		assert.Nil(t, err)
		assert.True(t, m.HasValue())
		assert.Equal(t, "ok", m.MustGet())
	})
}
