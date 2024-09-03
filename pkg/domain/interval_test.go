package domain_test

import (
	"testing"
	"time"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/domain"
)

func Test_TimeInterval(t *testing.T) {
	t.Run("should fail if the from date is after the to date", func(t *testing.T) {
		_, err := domain.NewTimeInterval(time.Now(), time.Now().Add(-time.Second))

		assert.ErrorIs(t, domain.ErrInvalidTimeInterval, err)
	})

	t.Run("should succeed if the from date is before the to date", func(t *testing.T) {
		from := time.Now()
		to := time.Now().Add(time.Second)
		ti, err := domain.NewTimeInterval(from, to)

		assert.Nil(t, err)
		assert.Equal(t, from, ti.From())
		assert.Equal(t, to, ti.To())
	})
}
