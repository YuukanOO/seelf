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
		_, err := domain.NewTarget(name, targetUrl, false, config, true, uid)
		testutil.Equals(t, domain.ErrDomainAlreadyTaken, err)
	})

	t.Run("should fail if the config is not unique", func(t *testing.T) {
		_, err := domain.NewTarget(name, targetUrl, true, config, false, uid)
		testutil.Equals(t, domain.ErrConfigAlreadyTaken, err)
	})

	t.Run("should be instantiable", func(t *testing.T) {
		target, err := domain.NewTarget(name, targetUrl, true, config, true, uid)

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &target, 1)
		evt := testutil.EventIs[domain.TargetCreated](t, &target, 0)

		testutil.NotEquals(t, "", evt.ID)
		testutil.Equals(t, name, evt.Name)
		testutil.Equals(t, targetUrl, evt.Domain)
		testutil.Equals(t, config, evt.Provider)
		testutil.Equals(t, uid, evt.Created.By())
	})

	t.Run("could be renamed and raise the event only if different", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		err := target.Rename("new-name")

		testutil.IsNil(t, err)
		evt := testutil.EventIs[domain.TargetRenamed](t, &target, 1)
		testutil.Equals(t, "new-name", evt.Name)

		testutil.IsNil(t, target.Rename("new-name"))
		testutil.HasNEvents(t, &target, 2)
	})

	t.Run("could not be renamed if delete requested", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))
		testutil.IsNil(t, target.RequestDelete(0, uid))

		testutil.ErrorIs(t, domain.ErrTargetDeleteRequested, target.Rename("new-name"))
	})

	t.Run("could have its domain changed if available and raise the event only if different", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))
		newUrl := must.Panic(domain.UrlFrom("http://new-url.com"))

		err := target.HasDomain(newUrl, false)

		testutil.ErrorIs(t, domain.ErrDomainAlreadyTaken, err)

		err = target.HasDomain(newUrl, true)

		testutil.IsNil(t, err)
		evt := testutil.EventIs[domain.TargetDomainChanged](t, &target, 1)
		testutil.Equals(t, newUrl, evt.Domain)

		testutil.IsNil(t, target.HasDomain(newUrl, true))
		testutil.HasNEvents(t, &target, 2)
	})

	t.Run("could not have its domain changed if delete requested", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		testutil.IsNil(t, target.RequestDelete(0, uid))
		testutil.ErrorIs(t, domain.ErrTargetDeleteRequested, target.HasDomain(must.Panic(domain.UrlFrom("http://new-url.com")), true))
	})

	t.Run("should not be removed if still used by an app", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		testutil.ErrorIs(t, domain.ErrTargetInUse, target.RequestDelete(1, uid))
	})

	t.Run("could be removed if no app is using it", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		err := target.RequestDelete(0, uid)
		testutil.IsNil(t, err)

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &target, 2)
		evt := testutil.EventIs[domain.TargetDeleteRequested](t, &target, 1)
		testutil.Equals(t, target.ID(), evt.ID)
	})

	t.Run("should not be removed if no delete request has been set", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		testutil.ErrorIs(t, domain.ErrTargetDeleteRequestNeeded, target.Delete(0))
	})

	t.Run("should not be removed if at least one deployment is using it", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		testutil.IsNil(t, target.RequestDelete(0, uid))              // No application is using it
		testutil.ErrorIs(t, domain.ErrTargetInUse, target.Delete(1)) // But one deployment is still running on it
	})

	t.Run("could be removed if a delete request has been set", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		testutil.IsNil(t, target.RequestDelete(0, uid))
		err := target.Delete(0)

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &target, 3)
		evt := testutil.EventIs[domain.TargetDeleted](t, &target, 2)
		testutil.Equals(t, target.ID(), evt.ID)
	})
}

type dummyProviderConfig struct {
	domain.ProviderConfig
}
