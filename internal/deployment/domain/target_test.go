package domain_test

import (
	"errors"
	"testing"
	"time"

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
		testutil.Equals(t, domain.ErrUrlAlreadyTaken, err)
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
		testutil.Equals(t, targetUrl, evt.Url)
		testutil.Equals(t, config, evt.Provider)
		testutil.Equals(t, domain.TargetStatusConfiguring, evt.State.Status())
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

		err := target.HasUrl(newUrl, false)

		testutil.ErrorIs(t, domain.ErrUrlAlreadyTaken, err)

		err = target.HasUrl(newUrl, true)

		testutil.IsNil(t, err)
		evt := testutil.EventIs[domain.TargetUrlChanged](t, &target, 1)
		testutil.Equals(t, newUrl, evt.Url)

		evtTargetChanged := testutil.EventIs[domain.TargetStateChanged](t, &target, 2)
		testutil.Equals(t, domain.TargetStatusConfiguring, evtTargetChanged.State.Status())

		testutil.IsNil(t, target.HasUrl(newUrl, true))
		testutil.HasNEvents(t, &target, 3)
	})

	t.Run("could not have its domain changed if delete requested", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		testutil.IsNil(t, target.RequestDelete(0, uid))
		testutil.ErrorIs(t, domain.ErrTargetDeleteRequested, target.HasUrl(must.Panic(domain.UrlFrom("http://new-url.com")), true))
	})

	t.Run("should forbid a provider change if the fingerpint has changed", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name,
			must.Panic(domain.UrlFrom("http://docker.localhost")), true, config, true, uid))

		err := target.HasProvider(dummyProviderConfig{data: "new-config", fingerprint: "new-fingerprint"}, true)

		testutil.ErrorIs(t, domain.ErrTargetProviderUpdateNotPermitted, err)
	})

	t.Run("could have its provider changed if available and raise the event only if different", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name,
			must.Panic(domain.UrlFrom("http://docker.localhost")), true, config, true, uid))
		newConfig := dummyProviderConfig{data: "new-config"}

		err := target.HasProvider(newConfig, false)

		testutil.ErrorIs(t, domain.ErrConfigAlreadyTaken, err)

		err = target.HasProvider(newConfig, true)

		testutil.IsNil(t, err)
		evt := testutil.EventIs[domain.TargetProviderChanged](t, &target, 1)
		testutil.IsTrue(t, newConfig == evt.Provider)

		evtTargetChanged := testutil.EventIs[domain.TargetStateChanged](t, &target, 2)
		testutil.Equals(t, domain.TargetStatusConfiguring, evtTargetChanged.State.Status())

		testutil.IsNil(t, target.HasProvider(newConfig, true))
		testutil.HasNEvents(t, &target, 3)
	})

	t.Run("could not have its provider changed if delete requested", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		testutil.IsNil(t, target.RequestDelete(0, uid))
		testutil.ErrorIs(t, domain.ErrTargetDeleteRequested, target.HasProvider(config, true))
	})

	t.Run("should raise the TargetStateChanged only once when updating both the domain and the config", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))
		newUrl := must.Panic(domain.UrlFrom("http://new-url.com"))
		newConfig := dummyProviderConfig{data: "new-config"}

		testutil.IsNil(t, target.HasUrl(newUrl, true))
		testutil.IsNil(t, target.HasProvider(newConfig, true))

		testutil.HasNEvents(t, &target, 4)
		evt := testutil.EventIs[domain.TargetStateChanged](t, &target, 3)
		testutil.Equals(t, domain.TargetStatusConfiguring, evt.State.Status())
	})

	t.Run("could be marked as configured and raise the appropriate event", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)

		target.Configured(created.State.Version().Add(-1*time.Hour), nil)

		testutil.HasNEvents(t, &target, 1)

		target.Configured(created.State.Version(), nil)
		target.Configured(created.State.Version(), nil) // Should not raise a new event

		testutil.HasNEvents(t, &target, 2)
		changed := testutil.EventIs[domain.TargetStateChanged](t, &target, 1)
		testutil.Equals(t, domain.TargetStatusReady, changed.State.Status())
	})

	t.Run("could expose its availability based on its internal state", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		// Configuring
		beenReadyAtLeastOnce, err := target.CheckAvailability()

		testutil.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
		testutil.IsFalse(t, beenReadyAtLeastOnce)

		// Configuration failed
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), errors.New("configuration failed"))

		beenReadyAtLeastOnce, err = target.CheckAvailability()

		testutil.ErrorIs(t, domain.ErrTargetConfigurationFailed, err)
		testutil.IsFalse(t, beenReadyAtLeastOnce)

		// Configuration success
		target.Reconfigure()
		changed := testutil.EventIs[domain.TargetStateChanged](t, &target, 1)

		target.Configured(changed.State.Version(), nil)

		beenReadyAtLeastOnce, err = target.CheckAvailability()

		testutil.IsNil(t, err)
		testutil.IsTrue(t, beenReadyAtLeastOnce)

		// Delete requested
		target.RequestDelete(0, uid)

		beenReadyAtLeastOnce, err = target.CheckAvailability()

		testutil.ErrorIs(t, domain.ErrTargetDeleteRequested, err)
		testutil.IsTrue(t, beenReadyAtLeastOnce)
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

	t.Run("should not raise an event is the target is already marked has deleting", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		testutil.IsNil(t, target.RequestDelete(0, uid))
		testutil.IsNil(t, target.RequestDelete(0, uid))

		testutil.HasNEvents(t, &target, 2)
	})

	t.Run("should not be removed if no delete request has been set", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))

		_, err := target.Delete(0)

		testutil.ErrorIs(t, domain.ErrTargetDeleteRequestNeeded, err)
	})

	t.Run("should not be removed if at least one deployment is using it", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))
		testutil.IsNil(t, target.RequestDelete(0, uid)) // No application is using it

		_, err := target.Delete(1)

		testutil.ErrorIs(t, domain.ErrTargetInUse, err) // But one deployment is still running on it
	})

	t.Run("should not be removed while configuring", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))
		target.RequestDelete(0, uid)

		_, err := target.Delete(0)

		testutil.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
	})

	t.Run("should returns the skip cleanup strategy if the configuration has failed and has never been reachable", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), errors.New("configuration failed"))
		target.RequestDelete(0, uid)

		s, err := target.Delete(0)

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.TargetCleanupStrategySkip, s)
		testutil.EventIs[domain.TargetDeleted](t, &target, 3)
	})

	t.Run("should returns the force cleanup strategy if the configuration has failed and has been reachable in the past", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), nil)
		target.Reconfigure()
		changed := testutil.EventIs[domain.TargetStateChanged](t, &target, 1)
		target.Configured(changed.State.Version(), errors.New("configuration failed"))
		target.RequestDelete(0, uid)

		s, err := target.Delete(0)

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.TargetCleanupStrategyForce, s)
		testutil.EventIs[domain.TargetDeleted](t, &target, 3)
	})

	t.Run("could be removed if a delete request has been set and the target is correctly configured", func(t *testing.T) {
		target := must.Panic(domain.NewTarget(name, targetUrl, true, config, true, uid))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), nil)

		testutil.IsNil(t, target.RequestDelete(0, uid))
		s, err := target.Delete(0)

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.TargetCleanupStrategyDefault, s)
		testutil.HasNEvents(t, &target, 4)
		evt := testutil.EventIs[domain.TargetDeleted](t, &target, 3)
		testutil.Equals(t, target.ID(), evt.ID)
	})
}

type dummyProviderConfig struct {
	data        string
	fingerprint string
}

func (d dummyProviderConfig) Kind() string        { return "dummy" }
func (d dummyProviderConfig) Fingerprint() string { return d.fingerprint }
func (d dummyProviderConfig) String() string      { return d.fingerprint }

func (d dummyProviderConfig) Equals(other domain.ProviderConfig) bool {
	return d == other
}
