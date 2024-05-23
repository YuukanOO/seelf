package create_registry_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_registry"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_CreateRegistry(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	sut := func(existing ...*domain.Registry) bus.RequestHandler[string, create_registry.Command] {
		store := memory.NewRegistriesStore(existing...)
		return create_registry.Handler(store, store)
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := sut()
		id, err := uc(ctx, create_registry.Command{})

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
		testutil.Equals(t, "", id)
	})

	t.Run("should fail if the url is already taken", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))
		uc := sut(&r)

		id, err := uc(ctx, create_registry.Command{
			Name: "registry",
			Url:  "http://example.com",
		})

		testutil.Equals(t, "", id)
		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrUrlAlreadyTaken, validationErr["url"])
	})

	t.Run("should create a new registry if everything is good", func(t *testing.T) {
		uc := sut()

		id, err := uc(ctx, create_registry.Command{
			Name: "registry",
			Url:  "http://example.com",
			Credentials: monad.Value(create_registry.Credentials{
				Username: "user",
				Password: "password",
			}),
		})

		testutil.NotEquals(t, "", id)
		testutil.IsNil(t, err)
	})
}
