package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Registry(t *testing.T) {
	t.Run("should returns an error if the url is not unique", func(t *testing.T) {
		_, err := domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), false), "uid")

		testutil.ErrorIs(t, domain.ErrUrlAlreadyTaken, err)
	})

	t.Run("could be created from a valid url", func(t *testing.T) {
		r, err := domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid")

		testutil.IsNil(t, err)
		created := testutil.EventIs[domain.RegistryCreated](t, &r, 0)
		testutil.Equals(t, "http://example.com", created.Url.String())
		testutil.NotEquals(t, "", created.ID)
		testutil.Equals(t, "uid", created.Created.By())
		testutil.IsFalse(t, created.Created.At().IsZero())
	})

	t.Run("could be renamed and raise the event only if different", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))

		r.Rename("new registry")
		r.Rename("new registry")

		testutil.HasNEvents(t, &r, 2)

		renamed := testutil.EventIs[domain.RegistryRenamed](t, &r, 1)
		testutil.Equals(t, r.ID(), renamed.ID)
		testutil.Equals(t, "new registry", renamed.Name)
	})

	t.Run("should require a valid url when updating it", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))

		err := r.HasUrl(domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://localhost:5000")), false))

		testutil.ErrorIs(t, domain.ErrUrlAlreadyTaken, err)
	})

	t.Run("could have its url changed and raise the event only if different", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))

		r.HasUrl(domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true))
		r.HasUrl(domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://localhost:5000")), true))

		testutil.HasNEvents(t, &r, 2)

		changed := testutil.EventIs[domain.RegistryUrlChanged](t, &r, 1)
		testutil.Equals(t, r.ID(), changed.ID)
		testutil.Equals(t, "http://localhost:5000", changed.Url.String())
	})

	t.Run("could have credentials attached and raise the event only if different", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))

		r.UseAuthentication(domain.NewCredentials("user", "password"))
		r.UseAuthentication(domain.NewCredentials("user", "password"))

		testutil.HasNEvents(t, &r, 2)

		changed := testutil.EventIs[domain.RegistryCredentialsChanged](t, &r, 1)
		testutil.Equals(t, r.ID(), changed.ID)
		testutil.Equals(t, "user", changed.Credentials.Username())
		testutil.Equals(t, "password", changed.Credentials.Password())
	})

	t.Run("could have credentials removed and raise the event once", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))
		r.UseAuthentication(domain.NewCredentials("user", "password"))

		r.RemoveAuthentication()
		r.RemoveAuthentication()

		removed := testutil.EventIs[domain.RegistryCredentialsRemoved](t, &r, 2)
		testutil.Equals(t, r.ID(), removed.ID)
	})

	t.Run("could be deleted", func(t *testing.T) {
		r := must.Panic(domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true), "uid"))

		r.Delete()

		deleted := testutil.EventIs[domain.RegistryDeleted](t, &r, 1)
		testutil.Equals(t, r.ID(), deleted.ID)
	})
}
