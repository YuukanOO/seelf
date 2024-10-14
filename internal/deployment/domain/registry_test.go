package domain_test

import (
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/must"
)

func Test_Registry(t *testing.T) {
	t.Run("should returns an error if the url is not unique", func(t *testing.T) {
		_, err := domain.NewRegistry("registry", domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), false), "uid")

		assert.ErrorIs(t, domain.ErrUrlAlreadyTaken, err)
	})

	t.Run("could be created from a valid url", func(t *testing.T) {
		var (
			url              = must.Panic(domain.UrlFrom("http://example.com"))
			name             = "registry"
			uid  auth.UserID = "uid"
		)

		r, err := domain.NewRegistry(name, domain.NewRegistryUrlRequirement(url, true), uid)

		assert.Nil(t, err)
		assert.NotZero(t, r.ID())
		assert.Equal(t, url, r.Url())
		assert.Equal(t, name, r.Name())

		created := assert.EventIs[domain.RegistryCreated](t, &r, 0)

		assert.Equal(t, domain.RegistryCreated{
			ID:      r.ID(),
			Name:    name,
			Url:     url,
			Created: shared.ActionFrom(uid, assert.NotZero(t, created.Created.At())),
		}, created)
	})

	t.Run("could be renamed and raise the event only if different", func(t *testing.T) {
		r := fixture.Registry(fixture.WithRegistryName("registry"))

		r.Rename("new registry")
		r.Rename("new registry")

		assert.HasNEvents(t, 2, &r, "should raise the event once per different name")

		renamed := assert.EventIs[domain.RegistryRenamed](t, &r, 1)

		assert.Equal(t, domain.RegistryRenamed{
			ID:   r.ID(),
			Name: "new registry",
		}, renamed)
	})

	t.Run("should require a valid url when updating it", func(t *testing.T) {
		r := fixture.Registry()

		err := r.HasUrl(domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://localhost:5000")), false))

		assert.ErrorIs(t, domain.ErrUrlAlreadyTaken, err)
	})

	t.Run("could have its url changed and raise the event only if different", func(t *testing.T) {
		r := fixture.Registry(fixture.WithUrl(must.Panic(domain.UrlFrom("http://example.com"))))

		assert.Nil(t, r.HasUrl(domain.NewRegistryUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true)))

		differentUrl := must.Panic(domain.UrlFrom("http://localhost:5000"))
		assert.Nil(t, r.HasUrl(domain.NewRegistryUrlRequirement(differentUrl, true)))

		assert.HasNEvents(t, 2, &r, "should raise the event only if given url is different")

		changed := assert.EventIs[domain.RegistryUrlChanged](t, &r, 1)

		assert.Equal(t, domain.RegistryUrlChanged{
			ID:  r.ID(),
			Url: differentUrl,
		}, changed)
	})

	t.Run("could have credentials attached and raise the event only if different", func(t *testing.T) {
		r := fixture.Registry()
		credentials := domain.NewCredentials("user", "password")

		r.UseAuthentication(credentials)
		r.UseAuthentication(credentials)

		assert.HasNEvents(t, 2, &r, "should raise the event once per different credentials")

		changed := assert.EventIs[domain.RegistryCredentialsChanged](t, &r, 1)

		assert.Equal(t, domain.RegistryCredentialsChanged{
			ID:          r.ID(),
			Credentials: credentials,
		}, changed)
	})

	t.Run("could have credentials removed and raise the event once", func(t *testing.T) {
		r := fixture.Registry()
		r.UseAuthentication(domain.NewCredentials("user", "password"))

		r.RemoveAuthentication()
		r.RemoveAuthentication()

		removed := assert.EventIs[domain.RegistryCredentialsRemoved](t, &r, 2)

		assert.Equal(t, domain.RegistryCredentialsRemoved{
			ID: r.ID(),
		}, removed)
	})

	t.Run("could be deleted", func(t *testing.T) {
		r := fixture.Registry()

		r.Delete()

		deleted := assert.EventIs[domain.RegistryDeleted](t, &r, 1)

		assert.Equal(t, domain.RegistryDeleted{
			ID: r.ID(),
		}, deleted)
	})
}
