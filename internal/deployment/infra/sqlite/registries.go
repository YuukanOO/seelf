package sqlite

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type (
	RegistriesStore interface {
		domain.RegistriesReader
		domain.RegistriesWriter
	}

	registriesStore struct {
		db *sqlite.Database
	}
)

func NewRegistriesStore(db *sqlite.Database) RegistriesStore {
	return &registriesStore{db}
}

func (s *registriesStore) CheckUrlAvailability(ctx context.Context, url domain.Url, excluded ...domain.RegistryID) (domain.RegistryUrlRequirement, error) {
	unique, err := builder.
		Query[bool]("SELECT NOT EXISTS(SELECT 1 FROM registries WHERE url = ?", url).
		S(builder.Array("AND id NOT IN", excluded)).
		F(")").
		Extract(s.db, ctx)

	return domain.NewRegistryUrlRequirement(url, unique), err
}

func (s *registriesStore) GetByID(ctx context.Context, id domain.RegistryID) (domain.Registry, error) {
	return builder.
		Query[domain.Registry](`
		SELECT
			id
			,name
			,url
			,credentials_username
			,credentials_password
			,created_at
			,created_by
			,version
		FROM registries
		WHERE id = ?`, id).
		One(s.db, ctx, domain.RegistryFrom)
}

func (s *registriesStore) GetAll(ctx context.Context) ([]domain.Registry, error) {
	return builder.
		Query[domain.Registry](`
		SELECT
			id
			,name
			,url
			,credentials_username
			,credentials_password
			,created_at
			,created_by
			,version
		FROM registries`).
		All(s.db, ctx, domain.RegistryFrom)
}

func (s *registriesStore) Write(ctx context.Context, registries ...*domain.Registry) error {
	return sqlite.WriteEvents(s.db, ctx, registries,
		"registries",
		func(r *domain.Registry) sqlite.Key {
			return sqlite.Key{
				"id": r.ID(),
			}
		},
		func(e event.Event, v builder.Values) sqlite.WriteMode {
			switch evt := e.(type) {
			case domain.RegistryCreated:
				v["id"] = evt.ID
				v["name"] = evt.Name
				v["url"] = evt.Url
				v["created_at"] = evt.Created.At()
				v["created_by"] = evt.Created.By()
			case domain.RegistryRenamed:
				v["name"] = evt.Name
			case domain.RegistryUrlChanged:
				v["url"] = evt.Url
			case domain.RegistryCredentialsChanged:
				v["credentials_username"] = evt.Credentials.Username()
				v["credentials_password"] = evt.Credentials.Password()
			case domain.RegistryCredentialsRemoved:
				v["credentials_username"] = nil
				v["credentials_password"] = nil
			case domain.RegistryDeleted:
				return sqlite.WriteModeDelete
			}

			return sqlite.WriteModeUpsert
		})
}
