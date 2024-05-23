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
		FROM registries`).
		All(s.db, ctx, domain.RegistryFrom)
}

func (s *registriesStore) Write(ctx context.Context, registries ...*domain.Registry) error {
	return sqlite.WriteAndDispatch(s.db, ctx, registries, func(ctx context.Context, e event.Event) error {
		switch evt := e.(type) {
		case domain.RegistryCreated:
			return builder.
				Insert("registries", builder.Values{
					"id":         evt.ID,
					"name":       evt.Name,
					"url":        evt.Url,
					"created_at": evt.Created.At(),
					"created_by": evt.Created.By(),
				}).
				Exec(s.db, ctx)
		case domain.RegistryRenamed:
			return builder.
				Update("registries", builder.Values{
					"name": evt.Name,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.RegistryUrlChanged:
			return builder.
				Update("registries", builder.Values{
					"url": evt.Url,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.RegistryCredentialsChanged:
			return builder.
				Update("registries", builder.Values{
					"credentials_username": evt.Credentials.Username(),
					"credentials_password": evt.Credentials.Password(),
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.RegistryCredentialsRemoved:
			return builder.
				Update("registries", builder.Values{
					"credentials_username": nil,
					"credentials_password": nil,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.RegistryDeleted:
			return builder.
				Command("DELETE FROM registries WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		default:
			return nil
		}
	})
}
