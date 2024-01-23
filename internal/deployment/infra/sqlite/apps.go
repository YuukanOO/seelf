package sqlite

import (
	"context"
	"fmt"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/storage"
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

	appNameUniquenessResult struct {
		productionCount  uint
		productionTarget bool
		stagingCount     uint
		stagingTarget    bool
	}

	appNameUniquenessOnTargetEnvironmentResult struct {
		count  uint
		target bool
	}
)

func NewAppsStore(db *sqlite.Database) AppsStore {
	return &appsStore{db}
}

func (s *appsStore) GetAppNamingAvailability(
	ctx context.Context,
	name domain.AppName,
	production domain.TargetID,
	staging domain.TargetID,
) (domain.AppNamingAvailability, error) {
	var availability domain.AppNamingAvailability

	r, err := builder.
		Query[appNameUniquenessResult](`
		SELECT
			COUNT(CASE WHEN apps.production_target = ? THEN 1 END) AS production_count
			,MAX(CASE WHEN targets.id = ? THEN true ELSE false END) AS production_target_exists
			,COUNT(CASE WHEN apps.staging_target = ? THEN 1 END) AS staging_count
			,MAX(CASE WHEN targets.id = ? THEN true ELSE false END) AS staging_target_exists
		FROM
			targets
		LEFT JOIN apps ON apps.production_target = targets.id OR apps.staging_target = targets.id
		WHERE 
			targets.id IN (?, ?)
			AND (apps.name = ? OR apps.name IS NULL)
		`, production, production, staging, staging, production, staging, name).
		One(s.db, ctx, appNameUniquenessResultMapper)

	if err != nil {
		return availability, err
	}

	if !r.productionTarget {
		availability = availability | domain.AppNamingProductionTargetNotFound
	}

	if !r.stagingTarget {
		availability = availability | domain.AppNamingStagingTargetNotFound
	}

	if r.productionCount > 0 {
		availability = availability | domain.AppNamingTakenInProduction
	}

	if r.stagingCount > 0 {
		availability = availability | domain.AppNamingTakenInStaging
	}

	if availability != 0 {
		return availability, nil
	}

	return domain.AppNamingAvailable, nil
}

func (s *appsStore) GetTargetAppNamingAvailability(
	ctx context.Context,
	id domain.AppID,
	env domain.Environment,
	target domain.TargetID,
) (domain.TargetAppNamingAvailability, error) {
	var availability domain.TargetAppNamingAvailability

	r, err := builder.
		Query[appNameUniquenessOnTargetEnvironmentResult](fmt.Sprintf(`
		SELECT
			COUNT(apps.id) AS count
			,MAX(CASE WHEN targets.id = ? THEN true ELSE false END) AS exists
		FROM
			targets
		LEFT JOIN apps ON apps.%s_target = targets.id
		WHERE
			targets.id = ?
			AND apps.name = (SELECT name FROM apps WHERE apps.id = ?)
			AND apps.id != ?`, env), target, target, id, id).
		One(s.db, ctx, appNameUniquenessOnTargetEnvironmentResultMapper)

	if err != nil {
		return availability, err
	}

	if !r.target {
		availability = availability | domain.TargetAppNamingTargetNotFound
	}

	if r.count > 0 {
		availability = availability | domain.TargetAppNamingTaken
	}

	if availability != 0 {
		return availability, nil
	}

	return domain.TargetAppNamingAvailable, nil
}

func (s *appsStore) GetByID(ctx context.Context, id domain.AppID) (domain.App, error) {
	return builder.
		Query[domain.App](`
			SELECT
				id
				,name
				,vcs_url
				,vcs_token
				,production_target
				,production_vars
				,staging_target
				,staging_vars
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
					"id":                evt.ID,
					"name":              evt.Name,
					"production_target": evt.Production.Target(),
					"production_vars":   evt.Production.Vars(),
					"staging_target":    evt.Staging.Target(),
					"staging_vars":      evt.Staging.Vars(),
					"created_at":        evt.Created.At(),
					"created_by":        evt.Created.By(),
				}).
				Exec(s.db, ctx)
		case domain.AppEnvChanged:
			// This is safe to interpolate the column name here since events are raised by our
			// own code.
			return builder.
				Update("apps", builder.Values{
					fmt.Sprintf("%s_target", evt.Environment): evt.Config.Target(),
					fmt.Sprintf("%s_vars", evt.Environment):   evt.Config.Vars(),
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

func appNameUniquenessResultMapper(s storage.Scanner) (r appNameUniquenessResult, err error) {
	err = s.Scan(
		&r.productionCount,
		&r.productionTarget,
		&r.stagingCount,
		&r.stagingTarget,
	)

	return r, err
}

func appNameUniquenessOnTargetEnvironmentResultMapper(s storage.Scanner) (r appNameUniquenessOnTargetEnvironmentResult, err error) {
	err = s.Scan(&r.count, &r.target)

	return r, err
}
