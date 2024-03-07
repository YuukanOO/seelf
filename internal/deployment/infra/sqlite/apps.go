package sqlite

import (
	"context"
	"fmt"
	"strings"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/monad"
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
	r, err := builder.
		Query[appNamingResult](`
		SELECT
			(SELECT COUNT(id) FROM apps WHERE name = ? AND production_target = ?) AS production_count
			,(SELECT COUNT(id) FROM targets WHERE id = ? AND delete_requested_at IS NULL) AS production_target_exists
			,(SELECT COUNT(id) FROM apps WHERE name = ? AND staging_target = ?) AS staging_count
			,(SELECT COUNT(id) FROM targets WHERE id = ? AND delete_requested_at IS NULL) AS staging_target_exists
		`, name, production, production, name, staging, staging).
		One(s.db, ctx, appNameUniquenessResultMapper)

	if err != nil {
		return 0, err
	}

	return r.availability(true, true), nil
}

func (s *appsStore) GetAppNamingAvailabilityOnID(
	ctx context.Context,
	id domain.AppID,
	production monad.Maybe[domain.TargetID],
	staging monad.Maybe[domain.TargetID],
) (domain.AppNamingAvailability, error) {
	productionTarget, hasProductionTarget := production.TryGet()
	stagingTarget, hasStagingTarget := staging.TryGet()

	if !hasProductionTarget && !hasStagingTarget {
		return 0, nil
	}

	var (
		sql  strings.Builder
		args = make([]any, 0, 5)
	)

	// This one is a bit tricky because the request depends on how many target we should check.
	sql.WriteString("SELECT ")

	if hasProductionTarget {
		sql.WriteString(`
		(SELECT COUNT(id) FROM apps WHERE apps.id != src.id AND apps.name = src.name AND apps.production_target = ?) AS production_count
		,(SELECT COUNT(id) FROM targets WHERE id = ? AND delete_requested_at IS NULL) AS production_target_exists`)
		args = append(args, productionTarget, productionTarget)
	} else {
		sql.WriteString("0 AS production_count, 0 AS production_target_exists")
	}

	if hasStagingTarget {
		sql.WriteString(`
		,(SELECT COUNT(id) FROM apps WHERE apps.id != src.id AND apps.name = src.name AND apps.staging_target = ?) AS staging_count
		,(SELECT COUNT(id) FROM targets WHERE id = ? AND delete_requested_at IS NULL) AS staging_target_exists`)
		args = append(args, stagingTarget, stagingTarget)
	} else {
		sql.WriteString(", 0 AS staging_count, 0 AS staging_target_exists")
	}

	sql.WriteString(" FROM apps src WHERE src.id = ?")
	args = append(args, id)

	r, err := builder.
		Query[appNamingResult](sql.String(), args...).
		One(s.db, ctx, appNameUniquenessResultMapper)

	if err != nil {
		return 0, err
	}

	return r.availability(hasProductionTarget, hasStagingTarget), nil
}

func (s *appsStore) GetAppsOnTargetCount(ctx context.Context, target domain.TargetID) (domain.AppsOnTargetCount, error) {
	return builder.
		Query[domain.AppsOnTargetCount](`
		SELECT COUNT(id)
		FROM apps
		WHERE production_target = ? OR staging_target = ?`, target, target).
		Extract(s.db, ctx)
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

type appNamingResult struct {
	productionCount  uint
	productionTarget bool
	stagingCount     uint
	stagingTarget    bool
}

// Returns the appropriate flags set based on result fields.
func (r appNamingResult) availability(includeProduction, includeStaging bool) domain.AppNamingAvailability {
	var availability domain.AppNamingAvailability

	if includeProduction {
		if !r.productionTarget {
			availability |= domain.AppNamingProductionTargetNotFound
		} else if r.productionCount > 0 {
			availability |= domain.AppNamingTakenInProduction
		} else {
			availability |= domain.AppNamingProductionAvailable
		}
	}

	if includeStaging {
		if !r.stagingTarget {
			availability |= domain.AppNamingStagingTargetNotFound
		} else if r.stagingCount > 0 {
			availability |= domain.AppNamingTakenInStaging
		} else {
			availability |= domain.AppNamingStagingAvailable
		}
	}

	return availability
}

func appNameUniquenessResultMapper(s storage.Scanner) (r appNamingResult, err error) {
	err = s.Scan(
		&r.productionCount,
		&r.productionTarget,
		&r.stagingCount,
		&r.stagingTarget,
	)

	return r, err
}
