package delete_registry_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/app/delete_registry"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_DeleteRegistry(t *testing.T) {
	sut := func(existing ...*domain.Registry) bus.RequestHandler[bus.UnitType, delete_registry.Command] {
		store := memory.NewRegistriesStore(existing...)
		return delete_registry.Handler(store, store)
	}

	t.Run("should require an existing registry", func(t *testing.T) {
		uc := sut()

		_, err := uc(context.Background(), delete_registry.Command{
			ID: "non-existing-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should delete the registry", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))
		uc := sut(&r)

		_, err := uc(context.Background(), delete_registry.Command{
			ID: string(r.ID()),
		})

		testutil.IsNil(t, err)
		evt := testutil.EventIs[domain.RegistryDeleted](t, &r, 1)
		testutil.Equals(t, r.ID(), evt.ID)
	})
}
