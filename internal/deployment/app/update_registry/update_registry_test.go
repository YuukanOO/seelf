package update_registry_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_registry"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_UpdateRegistry(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	sut := func(existing ...*domain.Registry) bus.RequestHandler[string, update_registry.Command] {
		store := memory.NewRegistriesStore(existing...)
		return update_registry.Handler(store, store)
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := sut()

		id, err := uc(ctx, update_registry.Command{
			Url: monad.Value("not an url"),
		})

		testutil.Equals(t, "", id)
		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrInvalidUrl, validationErr["url"])
	})

	t.Run("should require an existing registry", func(t *testing.T) {
		uc := sut()

		_, err := uc(ctx, update_registry.Command{
			Url: monad.Value("http://example.com"),
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should rename a registry", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))
		uc := sut(&r)

		id, err := uc(ctx, update_registry.Command{
			ID:   string(r.ID()),
			Name: monad.Value("new-name"),
		})

		testutil.NotEquals(t, "", id)
		testutil.IsNil(t, err)
		evt := testutil.EventIs[domain.RegistryRenamed](t, &r, 1)
		testutil.Equals(t, r.ID(), evt.ID)
		testutil.Equals(t, "new-name", evt.Name)
	})

	t.Run("should require a unique url when updating it", func(t *testing.T) {
		r1 := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))
		r2 := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://localhost:5000")), true), "uid"))
		uc := sut(&r1, &r2)

		id, err := uc(ctx, update_registry.Command{
			ID:  string(r2.ID()),
			Url: monad.Value("http://example.com"),
		})

		testutil.Equals(t, "", id)
		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrUrlAlreadyTaken, validationErr["url"])
	})

	t.Run("should update the url if its good", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))
		uc := sut(&r)

		id, err := uc(ctx, update_registry.Command{
			ID:  string(r.ID()),
			Url: monad.Value("http://localhost:5000"),
		})

		testutil.NotEquals(t, "", id)
		testutil.IsNil(t, err)
		evt := testutil.EventIs[domain.RegistryUrlChanged](t, &r, 1)
		testutil.Equals(t, r.ID(), evt.ID)
		testutil.Equals(t, "http://localhost:5000", evt.Url.String())
	})

	t.Run("should be able to add credentials", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))
		uc := sut(&r)

		id, err := uc(ctx, update_registry.Command{
			ID: string(r.ID()),
			Credentials: monad.PatchValue(update_registry.Credentials{
				Username: "user",
				Password: monad.Value("password"),
			}),
		})

		testutil.NotEquals(t, "", id)
		testutil.IsNil(t, err)
		evt := testutil.EventIs[domain.RegistryCredentialsChanged](t, &r, 1)
		testutil.Equals(t, r.ID(), evt.ID)
		testutil.Equals(t, "user", evt.Credentials.Username())
		testutil.Equals(t, "password", evt.Credentials.Password())
	})

	t.Run("should be able to update only the credentials username", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))
		r.UseAuthentication(domain.NewCredentials("user", "password"))
		uc := sut(&r)

		id, err := uc(ctx, update_registry.Command{
			ID: string(r.ID()),
			Credentials: monad.PatchValue(update_registry.Credentials{
				Username: "new-user",
			}),
		})

		testutil.NotEquals(t, "", id)
		testutil.IsNil(t, err)
		evt := testutil.EventIs[domain.RegistryCredentialsChanged](t, &r, 2)
		testutil.Equals(t, r.ID(), evt.ID)
		testutil.Equals(t, "new-user", evt.Credentials.Username())
		testutil.Equals(t, "password", evt.Credentials.Password())
	})

	t.Run("should be able to remove authentication", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))
		r.UseAuthentication(domain.NewCredentials("user", "password"))
		uc := sut(&r)

		id, err := uc(ctx, update_registry.Command{
			ID:          string(r.ID()),
			Credentials: monad.Nil[update_registry.Credentials](),
		})

		testutil.NotEquals(t, "", id)
		testutil.IsNil(t, err)
		evt := testutil.EventIs[domain.RegistryCredentialsRemoved](t, &r, 2)
		testutil.Equals(t, r.ID(), evt.ID)
	})
}
