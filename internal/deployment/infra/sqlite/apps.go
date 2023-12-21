package sqlite

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type (
	AppsStore interface {
		domain.AppsReader
		domain.AppsWriter
	}

	appsStore struct {
		db *sqlite.Database
	}
)

func NewAppsStore(db *sqlite.Database) AppsStore {
	return &appsStore{db}
}

func (s *appsStore) IsNameUnique(ctx context.Context, name domain.AppName) (domain.UniqueAppName, error) {
	count, err := builder.
		Query[uint]("SELECT COUNT(name) FROM apps WHERE name = ?", name).
		Extract(s.db, ctx)

	if err != nil {
		return "", err
	}

	if count > 0 {
		return "", domain.ErrAppNameAlreadyTaken
	}

	return domain.UniqueAppName(name), nil
}

func (s *appsStore) GetByID(ctx context.Context, id domain.AppID) (domain.App, error) {
	return builder.
		Query[domain.App](`
			SELECT
				id
				,name
				,vcs_url
				,vcs_token
				,env
				,cleanup_requested_at
				,cleanup_requested_by
				,created_at
				,created_by
			FROM apps
			WHERE id = ?`, id).
		One(s.db, ctx, domain.AppFrom)
}

func (s *appsStore) Write(c context.Context, apps ...*domain.App) error {
	return sqlite.WriteAndDispatch(s.db, c, apps, func(ctx context.Context, e event.Event) error {
		switch evt := e.(type) {
		case domain.AppCreated:
			return builder.
				Insert("apps", builder.Values{
					"id":         evt.ID,
					"name":       evt.Name,
					"created_at": evt.Created.At(),
					"created_by": evt.Created.By(),
				}).
				Exec(s.db, ctx)
		case domain.AppEnvChanged:
			return builder.
				Update("apps", builder.Values{
					"env": evt.Env,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.AppEnvRemoved:
			return builder.
				Update("apps", builder.Values{
					"env": nil,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.AppVCSConfigured:
			return builder.
				Update("apps", builder.Values{
					"vcs_url":   evt.Config.Url(),
					"vcs_token": evt.Config.Token(),
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.AppVCSRemoved:
			return builder.
				Update("apps", builder.Values{
					"vcs_url":   nil,
					"vcs_token": nil,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.AppCleanupRequested:
			return builder.
				Update("apps", builder.Values{
					"cleanup_requested_at": evt.Requested.At(),
					"cleanup_requested_by": evt.Requested.By(),
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.AppDeleted:
			return builder.
				Command("DELETE FROM apps WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		default:
			return nil
		}
	})
}
