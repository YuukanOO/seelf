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
		Query[bool]("SELECT NOT EXISTS(SELECT 1 FROM targets WHERE url = ?", url).
		S(builder.Array("AND id NOT IN", excluded)).
		F(")").
		Extract(s.db, ctx)

	return domain.NewTargetUrlRequirement(url, unique), err
}

func (s *targetsStore) CheckConfigAvailability(ctx context.Context, config domain.ProviderConfig, excluded ...domain.TargetID) (domain.ProviderConfigRequirement, error) {
	unique, err := builder.
		Query[bool]("SELECT NOT EXISTS(SELECT 1 FROM targets WHERE provider_fingerprint = ?", config.Fingerprint()).
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
		FROM targets
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
		FROM targets
		WHERE id = ?`, id).
		One(s.db, ctx, domain.TargetFrom)
}

func (s *targetsStore) Write(c context.Context, targets ...*domain.Target) error {
	return sqlite.WriteAndDispatch(s.db, c, targets, func(ctx context.Context, e event.Event) error {
		switch evt := e.(type) {
		case domain.TargetCreated:
			return builder.
				Insert("targets", builder.Values{
					"id":                       evt.ID,
					"name":                     evt.Name,
					"url":                      evt.Url,
					"provider_kind":            evt.Provider.Kind(),
					"provider_fingerprint":     evt.Provider.Fingerprint(),
					"provider":                 evt.Provider,
					"state_status":             evt.State.Status(),
					"state_version":            evt.State.Version(),
					"state_errcode":            evt.State.ErrCode(),
					"state_last_ready_version": evt.State.LastReadyVersion(),
					"entrypoints":              evt.Entrypoints,
					"created_at":               evt.Created.At(),
					"created_by":               evt.Created.By(),
				}).
				Exec(s.db, ctx)
		case domain.TargetStateChanged:
			return builder.
				Update("targets", builder.Values{
					"state_status":             evt.State.Status(),
					"state_version":            evt.State.Version(),
					"state_errcode":            evt.State.ErrCode(),
					"state_last_ready_version": evt.State.LastReadyVersion(),
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.TargetRenamed:
			return builder.
				Update("targets", builder.Values{
					"name": evt.Name,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.TargetUrlChanged:
			return builder.
				Update("targets", builder.Values{
					"url": evt.Url,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.TargetProviderChanged:
			return builder.
				Update("targets", builder.Values{
					"provider_kind":        evt.Provider.Kind(),
					"provider_fingerprint": evt.Provider.Fingerprint(),
					"provider":             evt.Provider,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.TargetEntrypointsChanged:
			return builder.
				Update("targets", builder.Values{
					"entrypoints": evt.Entrypoints,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.TargetCleanupRequested:
			return builder.
				Update("targets", builder.Values{
					"cleanup_requested_at": evt.Requested.At(),
					"cleanup_requested_by": evt.Requested.By(),
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.TargetDeleted:
			return builder.
				Command("DELETE FROM targets WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		default:
			return nil
		}
	})
}
