package sqlite

import (
	"context"
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

func (s *appsStore) CheckAppNamingAvailability(
	ctx context.Context,
	name domain.AppName,
	production domain.EnvironmentConfig,
	staging domain.EnvironmentConfig,
) (domain.EnvironmentConfigRequirement, domain.EnvironmentConfigRequirement, error) {
	r, err := builder.
		Query[appNamingResult](`
		SELECT
			NOT EXISTS(SELECT 1 FROM [deployment.apps] WHERE name = ? AND (production_config_target = ? OR production_migration_target = ?)) AS production_available
			,EXISTS(SELECT 1 FROM [deployment.targets] WHERE id = ? AND cleanup_requested_at IS NULL) AS production_target_exists
			,NOT EXISTS(SELECT 1 FROM [deployment.apps] WHERE name = ? AND (staging_config_target = ? OR staging_migration_target = ?)) AS staging_available
			,EXISTS(SELECT 1 FROM [deployment.targets] WHERE id = ? AND cleanup_requested_at IS NULL) AS staging_target_exists
		`,
		name, production.Target(), production.Target(),
		production.Target(),
		name, staging.Target(), staging.Target(),
		staging.Target()).
		One(s.db, ctx, appNameUniquenessResultMapper)

	if err != nil {
		return domain.EnvironmentConfigRequirement{}, domain.EnvironmentConfigRequirement{}, err
	}

	return domain.NewEnvironmentConfigRequirement(production, r.productionTargetFound, r.productionAvailable),
		domain.NewEnvironmentConfigRequirement(staging, r.stagingTargetFound, r.stagingAvailable),
		nil
}

func (s *appsStore) CheckAppNamingAvailabilityByID(
	ctx context.Context,
	id domain.AppID,
	production monad.Maybe[domain.EnvironmentConfig],
	staging monad.Maybe[domain.EnvironmentConfig],
) (
	productionRequirement domain.EnvironmentConfigRequirement,
	stagingRequirement domain.EnvironmentConfigRequirement,
	err error,
) {
	productionValue, hasProductionTarget := production.TryGet()
	stagingValue, hasStagingTarget := staging.TryGet()

	if !hasProductionTarget && !hasStagingTarget {
		return productionRequirement, stagingRequirement, err
	}

	var (
		sql  strings.Builder
		args = make([]any, 0, 5)
	)

	// This one is a bit tricky because the request depends on how many target we should check.
	sql.WriteString("SELECT ")

	if hasProductionTarget {
		sql.WriteString(`
		NOT EXISTS(SELECT 1 FROM [deployment.apps] WHERE [deployment.apps].id != src.id AND [deployment.apps].name = src.name AND (production_config_target = ? OR production_migration_target = ?)) AS production_available
		,EXISTS(SELECT 1 FROM [deployment.targets] WHERE id = ? AND cleanup_requested_at IS NULL) AS production_target_exists`)
		args = append(args, productionValue.Target(), productionValue.Target(), productionValue.Target())
	} else {
		sql.WriteString("0 AS production_available, 0 AS production_target_exists")
	}

	if hasStagingTarget {
		sql.WriteString(`
		,NOT EXISTS(SELECT 1 FROM [deployment.apps] WHERE [deployment.apps].id != src.id AND [deployment.apps].name = src.name AND (staging_config_target = ? OR staging_migration_target = ?)) AS staging_available
		,EXISTS(SELECT 1 FROM [deployment.targets] WHERE id = ? AND cleanup_requested_at IS NULL) AS staging_target_exists`)
		args = append(args, stagingValue.Target(), stagingValue.Target(), stagingValue.Target())
	} else {
		sql.WriteString(", 0 AS staging_available, 0 AS staging_target_exists")
	}

	sql.WriteString(" FROM [deployment.apps] src WHERE src.id = ?")
	args = append(args, id)

	r, err := builder.
		Query[appNamingResult](sql.String(), args...).
		One(s.db, ctx, appNameUniquenessResultMapper)

	if err != nil {
		return productionRequirement, stagingRequirement, err
	}

	if hasProductionTarget {
		productionRequirement = domain.NewEnvironmentConfigRequirement(productionValue, r.productionTargetFound, r.productionAvailable)
	}

	if hasStagingTarget {
		stagingRequirement = domain.NewEnvironmentConfigRequirement(stagingValue, r.stagingTargetFound, r.stagingAvailable)
	}

	return productionRequirement, stagingRequirement, err
}

func (s *appsStore) HasAppsOnTarget(ctx context.Context, target domain.TargetID) (domain.HasAppsOnTarget, error) {
	r, err := builder.
		Query[bool](`
		SELECT EXISTS(SELECT 1 FROM [deployment.apps] WHERE production_config_target = ? OR production_migration_target = ? OR staging_config_target = ? OR staging_migration_target = ?)`,
		target, target, target, target).
		Extract(s.db, ctx)

	return domain.HasAppsOnTarget(r), err
}

func (s *appsStore) GetByID(ctx context.Context, id domain.AppID) (domain.App, error) {
	return builder.
		Query[domain.App](`
		SELECT
			id
			,name
			,version_control_url
			,version_control_token
			,production_migration_target
			,production_migration_from
			,production_migration_to
			,production_since
			,production_cleaned
			,production_config_target
			,production_config_vars
			,staging_migration_target
			,staging_migration_from
			,staging_migration_to
			,staging_since
			,staging_cleaned
			,staging_config_target
			,staging_config_vars
			,created_at
			,created_by
			,cleanup_requested_at
			,cleanup_requested_by
			,version
		FROM [deployment.apps]
		WHERE id = ?`, id).
		One(s.db, ctx, domain.AppFrom)
}

func (s *appsStore) Write(c context.Context, apps ...*domain.App) error {
	return sqlite.WriteEvents(s.db, c, apps,
		"[deployment.apps]",
		func(app *domain.App) sqlite.Key {
			return sqlite.Key{
				"id": app.ID(),
			}
		},
		func(e event.Event, v builder.Values) sqlite.WriteMode {
			switch evt := e.(type) {
			case domain.AppCreated:
				v["id"] = evt.ID
				v["name"] = evt.Name
				v["production_since"] = evt.Production.Since()
				v["production_cleaned"] = evt.Production.IsCleanedUp()
				v["production_config_target"] = evt.Production.Config().Target()
				v["production_config_vars"] = evt.Production.Config().Vars()
				v["staging_since"] = evt.Staging.Since()
				v["staging_cleaned"] = evt.Staging.IsCleanedUp()
				v["staging_config_target"] = evt.Staging.Config().Target()
				v["staging_config_vars"] = evt.Staging.Config().Vars()
				v["created_at"] = evt.Created.At()
				v["created_by"] = evt.Created.By()
			case domain.AppEnvChanged:
				writeEnvironmentConfig(evt.Environment, evt.Config, v)
			case domain.AppEnvCleanedUp:
				writeEnvironmentConfig(evt.Environment, evt.Config, v)
			case domain.AppVersionControlChanged:
				if vcs, hasVcs := evt.Config.TryGet(); hasVcs {
					v["version_control_url"] = vcs.Url()
					v["version_control_token"] = vcs.Token()
				} else {
					v["version_control_url"] = nil
					v["version_control_token"] = nil
				}
			case domain.AppCleanupRequested:
				v["cleanup_requested_at"] = evt.Requested.At()
				v["cleanup_requested_by"] = evt.Requested.By()
			case domain.AppDeleted:
				return sqlite.WriteModeDelete
			}

			return sqlite.WriteModeUpsert
		})
}

func writeEnvironmentConfig(env domain.EnvironmentName, config domain.Environment, v builder.Values) {
	envStr := string(env)
	// This is safe to interpolate the column name here since events are raised by our
	// own code.
	if migration, hasMigration := config.Migration().TryGet(); hasMigration {
		v[envStr+"_migration_target"] = migration.Target()
		v[envStr+"_migration_from"] = migration.Interval().From()
		v[envStr+"_migration_to"] = migration.Interval().To()
	} else {
		v[envStr+"_migration_target"] = nil
		v[envStr+"_migration_from"] = nil
		v[envStr+"_migration_to"] = nil
	}

	v[envStr+"_since"] = config.Since()
	v[envStr+"_cleaned"] = config.IsCleanedUp()
	v[envStr+"_config_target"] = config.Config().Target()
	v[envStr+"_config_vars"] = config.Config().Vars()
}

type appNamingResult struct {
	productionAvailable   bool
	productionTargetFound bool
	stagingAvailable      bool
	stagingTargetFound    bool
}

func appNameUniquenessResultMapper(s storage.Scanner) (r appNamingResult, err error) {
	err = s.Scan(
		&r.productionAvailable,
		&r.productionTargetFound,
		&r.stagingAvailable,
		&r.stagingTargetFound,
	)

	return r, err
}
