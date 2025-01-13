package sqlite

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type (
	TargetsStore interface {
		domain.TargetsReader
		domain.TargetsWriter
	}

	targetsStore struct {
		db *sqlite.Database
	}
)

func NewTargetsStore(db *sqlite.Database) TargetsStore {
	return &targetsStore{db}
}

func (s *targetsStore) CheckUrlAvailability(ctx context.Context, url domain.Url, excluded ...domain.TargetID) (domain.TargetUrlRequirement, error) {
	unique, err := builder.
		Query[bool]("SELECT NOT EXISTS(SELECT 1 FROM [deployment.targets] WHERE url = ?", url).
		S(builder.Array("AND id NOT IN", excluded)).
		F(")").
		Extract(s.db, ctx)

	return domain.NewTargetUrlRequirement(url, unique), err
}

func (s *targetsStore) CheckConfigAvailability(ctx context.Context, config domain.ProviderConfig, excluded ...domain.TargetID) (domain.ProviderConfigRequirement, error) {
	unique, err := builder.
		Query[bool]("SELECT NOT EXISTS(SELECT 1 FROM [deployment.targets] WHERE provider_fingerprint = ?", config.Fingerprint()).
		S(builder.Array("AND id NOT IN", excluded)).
		F(")").
		Extract(s.db, ctx)

	return domain.NewProviderConfigRequirement(config, unique), err
}

func (s *targetsStore) GetLocalTarget(ctx context.Context) (domain.Target, error) {
	return builder.
		Query[domain.Target](`
		SELECT
			id
			,name
			,url
			,provider_kind
			,provider
			,state_status
			,state_version
			,state_errcode
			,state_last_ready_version
			,entrypoints
			,cleanup_requested_at
			,cleanup_requested_by
			,created_at
			,created_by
			,version
		FROM [deployment.targets]
		WHERE provider_fingerprint = ''
		LIMIT 1`).
		One(s.db, ctx, domain.TargetFrom)
}

func (s *targetsStore) GetByID(ctx context.Context, id domain.TargetID) (domain.Target, error) {
	return builder.
		Query[domain.Target](`
		SELECT
			id
			,name
			,url
			,provider_kind
			,provider
			,state_status
			,state_version
			,state_errcode
			,state_last_ready_version
			,entrypoints
			,cleanup_requested_at
			,cleanup_requested_by
			,created_at
			,created_by
			,version
		FROM [deployment.targets]
		WHERE id = ?`, id).
		One(s.db, ctx, domain.TargetFrom)
}

func (s *targetsStore) Write(c context.Context, targets ...*domain.Target) error {
	return sqlite.WriteEvents(s.db, c, targets,
		"[deployment.targets]",
		func(t *domain.Target) sqlite.Key {
			return sqlite.Key{
				"id": t.ID(),
			}
		},
		func(e event.Event, v builder.Values) sqlite.WriteMode {
			switch evt := e.(type) {
			case domain.TargetCreated:
				v["id"] = evt.ID
				v["name"] = evt.Name
				v["provider_kind"] = evt.Provider.Kind()
				v["provider_fingerprint"] = evt.Provider.Fingerprint()
				v["provider"] = evt.Provider
				v["state_status"] = evt.State.Status()
				v["state_version"] = evt.State.Version()
				v["state_errcode"] = evt.State.ErrCode()
				v["state_last_ready_version"] = evt.State.LastReadyVersion()
				v["entrypoints"] = evt.Entrypoints
				v["created_at"] = evt.Created.At()
				v["created_by"] = evt.Created.By()
			case domain.TargetStateChanged:
				v["state_status"] = evt.State.Status()
				v["state_version"] = evt.State.Version()
				v["state_errcode"] = evt.State.ErrCode()
				v["state_last_ready_version"] = evt.State.LastReadyVersion()
			case domain.TargetRenamed:
				v["name"] = evt.Name
			case domain.TargetUrlChanged:
				v["url"] = evt.Url
			case domain.TargetUrlRemoved:
				v["url"] = nil
			case domain.TargetProviderChanged:
				v["provider_kind"] = evt.Provider.Kind()
				v["provider_fingerprint"] = evt.Provider.Fingerprint()
				v["provider"] = evt.Provider
			case domain.TargetEntrypointsChanged:
				v["entrypoints"] = evt.Entrypoints
			case domain.TargetCleanupRequested:
				v["cleanup_requested_at"] = evt.Requested.At()
				v["cleanup_requested_by"] = evt.Requested.By()
			case domain.TargetDeleted:
				return sqlite.WriteModeDelete
			}

			return sqlite.WriteModeUpsert
		})
}
