package create_target_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_CreateTarget(t *testing.T) {
	var (
		uid    auth.UserID = "uid"
		ctx                = auth.WithUserID(context.Background(), uid)
		config dummyConfig
	)

	sut := func(existingTargets ...*domain.Target) bus.RequestHandler[string, create_target.Command] {
		store := memory.NewTargetsStore(existingTargets...)

		return create_target.Handler(store, store, &dummyProvider{})
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := sut()

		_, err := uc(ctx, create_target.Command{})

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
	})

	t.Run("should require a unique url", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true),
			domain.NewProviderConfigRequirement(config, true), uid))

		uc := sut(&target)

		_, err := uc(ctx, create_target.Command{
			Name:     "target",
			Url:      "http://example.com",
			Provider: config,
		})

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
		validateError, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrUrlAlreadyTaken, validateError["url"])
		testutil.ErrorIs(t, domain.ErrConfigAlreadyTaken, validateError[config.Kind()])
	})

	t.Run("should require a valid provider config", func(t *testing.T) {
		uc := sut()

		_, err := uc(ctx, create_target.Command{
			Name: "target",
			Url:  "http://example.com",
		})

		testutil.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should require a unique provider config", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true),
			domain.NewProviderConfigRequirement(config, true), uid))

		uc := sut(&target)

		_, err := uc(ctx, create_target.Command{
			Name:     "target",
			Url:      "http://another.example.com",
			Provider: config,
		})

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
		validateError, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrConfigAlreadyTaken, validateError[config.Kind()])
	})

	t.Run("should create a new target", func(t *testing.T) {
		uc := sut()

		id, err := uc(ctx, create_target.Command{
			Name:     "target",
			Url:      "http://example.com",
			Provider: config,
		})

		testutil.IsNil(t, err)
		testutil.NotEquals(t, "", id)
	})
}

type (
	dummyProvider struct {
		domain.Provider
	}

	dummyConfig struct{}
)

func (*dummyProvider) Prepare(ctx context.Context, payload any, existing ...domain.ProviderConfig) (domain.ProviderConfig, error) {
	if payload == nil {
		return nil, domain.ErrNoValidProviderFound
	}

	return dummyConfig{}, nil
}

func (dummyConfig) Fingerprint() string                   { return "dummy" }
func (c dummyConfig) Equals(o domain.ProviderConfig) bool { return c == o }
func (dummyConfig) Kind() string                          { return "dummy" }
func (dummyConfig) String() string                        { return "dummy" }
