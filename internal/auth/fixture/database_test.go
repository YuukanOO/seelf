package fixture_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_PrepareDatabase(t *testing.T) {
	t.Run("should be able to prepare a database without seeding it", func(t *testing.T) {
		ctx := fixture.PrepareDatabase(t)

		assert.NotNil(t, ctx)
		assert.NotNil(t, ctx.Dispatcher)
		assert.NotNil(t, ctx.UsersStore)
		assert.HasLength(t, 0, ctx.Dispatcher.Signals())
		assert.HasLength(t, 0, ctx.Dispatcher.Requests())
	})

	t.Run("should seed users and attach the first user id to the created context", func(t *testing.T) {
		user1 := fixture.User()
		user2 := fixture.User()

		ctx := fixture.PrepareDatabase(t, fixture.WithUsers(&user1, &user2))

		assert.Equal(t, user1.ID(), domain.CurrentUser(ctx.Context).Get(""))
	})
}
