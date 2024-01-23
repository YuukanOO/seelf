package domain_test

import (
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Target(t *testing.T) {
	var (
		name                            = "my-target"
		targetUrl                       = must.Panic(domain.UrlFrom("http://my-url.com"))
		config    domain.ProviderConfig = dummyProviderConfig{}
		uid       auth.UserID           = "uid"
	)

	t.Run("should fail if the url is not unique", func(t *testing.T) {
		_, err := domain.NewTarget(name, targetUrl, config, uid, false)
		testutil.Equals(t, domain.ErrUrlAlreadyTaken, err)
	})

	t.Run("should be instantiable", func(t *testing.T) {
		target, err := domain.NewTarget(name, targetUrl, config, uid, true)

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &target, 1)
		evt := testutil.EventIs[domain.TargetCreated](t, &target, 0)

		testutil.NotEquals(t, "", evt.ID)
		testutil.Equals(t, name, evt.Name)
		testutil.Equals(t, targetUrl, evt.Url)
		testutil.Equals(t, config, evt.Provider)
		testutil.Equals(t, uid, evt.Created.By())
	})
}

type dummyProviderConfig struct {
	domain.ProviderConfig
}
