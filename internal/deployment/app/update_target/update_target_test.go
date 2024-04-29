package update_target_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/app/update_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_UpdateTarget(t *testing.T) {
	sut := func(existingTargets ...*domain.Target) bus.RequestHandler[string, update_target.Command] {
		store := memory.NewTargetsStore(existingTargets...)
		provider := &dummyProvider{}
		return update_target.Handler(store, store, provider)
	}

	t.Run("should fail if the target does not exist", func(t *testing.T) {
		uc := sut()

		_, err := uc(context.Background(), update_target.Command{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should fail if url or config are already taken", func(t *testing.T) {
		t1 := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(dummyConfig{"1"}, true), "uid"))
		t2 := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://docker.localhost")), true),
			domain.NewProviderConfigRequirement(dummyConfig{"2"}, true), "uid"))
		uc := sut(&t1, &t2)

		_, err := uc(context.Background(), update_target.Command{
			ID:       string(t1.ID()),
			Provider: "2",
			Url:      monad.Value("http://docker.localhost"),
		})

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrConfigAlreadyTaken, validationErr["dummy"])
		testutil.ErrorIs(t, domain.ErrUrlAlreadyTaken, validationErr["url"])
	})

	t.Run("should update the target if everything is good", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(dummyConfig{"1"}, true), "uid"))
		uc := sut(&target)

		id, err := uc(context.Background(), update_target.Command{
			ID:       string(target.ID()),
			Name:     monad.Value("new name"),
			Provider: "1",
			Url:      monad.Value("http://docker.localhost"),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(target.ID()), id)
		testutil.HasNEvents(t, &target, 6)

		renamed := testutil.EventIs[domain.TargetRenamed](t, &target, 1)
		testutil.Equals(t, "new name", renamed.Name)
		urlChanged := testutil.EventIs[domain.TargetUrlChanged](t, &target, 2)
		testutil.Equals(t, "http://docker.localhost", urlChanged.Url.String())
		providerChanged := testutil.EventIs[domain.TargetProviderChanged](t, &target, 4)
		testutil.Equals(t, domain.ProviderConfig(dummyConfig{"1"}), providerChanged.Provider)
		testutil.EventIs[domain.TargetStateChanged](t, &target, 3)
		testutil.EventIs[domain.TargetStateChanged](t, &target, 5)
	})
}

type (
	dummyProvider struct {
		domain.Provider
	}

	dummyConfig struct {
		data string
	}
)

func (*dummyProvider) Prepare(ctx context.Context, payload any, existing ...domain.ProviderConfig) (domain.ProviderConfig, error) {
	return dummyConfig{payload.(string)}, nil
}

func (dummyConfig) Kind() string                              { return "dummy" }
func (c dummyConfig) Fingerprint() string                     { return c.data }
func (c dummyConfig) Equals(other domain.ProviderConfig) bool { return false }
func (c dummyConfig) String() string                          { return c.data }
