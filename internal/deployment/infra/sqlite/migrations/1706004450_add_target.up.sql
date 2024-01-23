/**
 * target fields are mandatory but since SQLite doesn't support altering that NULL
 * constraint, keep it nullable...
 */

CREATE TABLE targets (
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    provider_kind TEXT NOT NULL,
    provider_fingerprint TEXT NOT NULL,
    provider TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    created_by TEXT NOT NULL,

    CONSTRAINT pk_targets PRIMARY KEY(id),
    CONSTRAINT fk_targets_created_by FOREIGN KEY(created_by) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT unique_targets_url UNIQUE(url), -- unique domain among all targets
    CONSTRAINT unique_targets_provider_fingerprint UNIQUE(provider_fingerprint) -- and one unique provider fingerpint (same host and so on)
);

ALTER TABLE deployments RENAME COLUMN config_env TO config_vars;
ALTER TABLE deployments ADD config_target TEXT NULL; -- No FK on config_target because we don't want to deal with a target deletion.

ALTER TABLE apps DROP COLUMN env;

-- Add FK on targets because we don't want to delete a target if it's used by an app.
ALTER TABLE apps ADD production_target TEXT NULL CONSTRAINT fk_apps_production_target REFERENCES targets(id) ON DELETE RESTRICT;
ALTER TABLE apps ADD production_vars TEXT NULL;
ALTER TABLE apps ADD staging_target TEXT NULL CONSTRAINT fk_apps_production_target REFERENCES targets(id) ON DELETE RESTRICT;
ALTER TABLE apps ADD staging_vars TEXT NULL;

-- Creates a default target if at least one app exists, else do nothing
INSERT INTO targets (id, name, url, provider_kind, provider_fingerprint, provider, created_at, created_by)
SELECT '2bRUdQnyRELMqyh9gFLQV1s0cqv', 'local', 'http://docker.localhost', 'docker', '', '{}', DATETIME() , (SELECT id FROM users LIMIT 1) FROM apps LIMIT 1;

-- Update all deployments to point to the default target
UPDATE deployments SET config_target = (SELECT id FROM targets LIMIT 1);

-- Try to populate new vars fields with the latest deployment queued and update apps targets
UPDATE apps
SET
    production_target = (SELECT id FROM targets LIMIT 1)
    ,staging_target = (SELECT id FROM targets LIMIT 1)
	,production_vars = (
        SELECT config_vars
        FROM deployments d
        WHERE d.app_id = id AND d.config_environment = 'production'
        ORDER BY d.requested_at DESC LIMIT 1
    )
	,staging_vars = (
        SELECT config_vars
        FROM deployments d
        WHERE d.app_id = id AND d.config_environment = 'staging'
        ORDER BY d.requested_at DESC LIMIT 1
    );

-- Create unique indexes on the couple (name, target) for both environments
CREATE UNIQUE INDEX unique_apps_name_production_target ON apps(name, production_target);
CREATE UNIQUE INDEX unique_apps_name_staging_target ON apps(name, staging_target);

DROP INDEX IF EXISTS unique_apps_name;