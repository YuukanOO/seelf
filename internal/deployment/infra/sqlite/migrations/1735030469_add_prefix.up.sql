DROP TRIGGER IF EXISTS on_deployment_failed_cleanup_jobs;
DROP TRIGGER IF EXISTS on_target_configure_remove_outdated_jobs;

-- REGISTRIES

CREATE TABLE [deployment.registries]
(
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    credentials_username TEXT NULL,
    credentials_password TEXT NULL,
    created_at DATETIME NOT NULL,
    created_by TEXT NOT NULL,
    version DATETIME NOT NULL,

    CONSTRAINT [deployment.pk_registries] PRIMARY KEY(id),
    CONSTRAINT [deployment.fk_registries_created_by] FOREIGN KEY(created_by) REFERENCES [auth.users](id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX [deployment.idx_registries_url] ON [deployment.registries](url);
CREATE INDEX [deployment.idx_registries_created_by] ON [deployment.registries](created_by);
CREATE INDEX [deployment.idx_registries_version] ON [deployment.registries](version);

INSERT INTO [deployment.registries]
(
    id
    ,name
    ,url
    ,credentials_username
    ,credentials_password
    ,created_at
    ,created_by
    ,version
)
SELECT 
    id
    ,name
    ,url
    ,credentials_username
    ,credentials_password
    ,created_at
    ,created_by
    ,version
FROM registries;

-- TARGETS

CREATE TABLE [deployment.targets]
(
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    url TEXT NULL,
    provider_kind TEXT NOT NULL,
    provider_fingerprint TEXT NOT NULL,
    provider JSON NOT NULL,
    state_status INTEGER NOT NULL,
    state_version DATETIME NOT NULL,
    state_errcode TEXT NULL,
    state_last_ready_version DATETIME NULL,
    cleanup_requested_at DATETIME NULL,
    cleanup_requested_by TEXT NULL,
    created_at DATETIME NOT NULL,
    created_by TEXT NOT NULL,
    entrypoints JSON NOT NULL DEFAULT '{}',
    version DATETIME NOT NULL,

    CONSTRAINT [deployment.pk_targets] PRIMARY KEY(id),
    CONSTRAINT [deployment.fk_targets_created_by] FOREIGN KEY(created_by) REFERENCES [auth.users](id) ON DELETE CASCADE,
    CONSTRAINT [deployment.fk_targets_cleanup_requested_by] FOREIGN KEY(cleanup_requested_by) REFERENCES [auth.users](id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX [deployment.idx_targets_url] ON [deployment.targets](url); -- unique url among all targets
CREATE UNIQUE INDEX [deployment.idx_targets_provider_fingerprint] ON [deployment.targets](provider_fingerprint); -- unique provider fingerprint
CREATE INDEX [deployment.idx_targets_created_by] ON [deployment.targets](created_by);
CREATE INDEX [deployment.idx_targets_cleanup_requested_by] ON [deployment.targets](cleanup_requested_by);
CREATE INDEX [deployment.idx_targets_version] ON [deployment.targets](version);

INSERT INTO [deployment.targets]
(
    id
    ,name
    ,url
    ,provider_kind
    ,provider_fingerprint
    ,provider
    ,state_status
    ,state_version
    ,state_errcode
    ,state_last_ready_version
    ,cleanup_requested_at
    ,cleanup_requested_by
    ,created_at
    ,created_by
    ,entrypoints
    ,version
)
SELECT
    id
    ,name
    ,url
    ,provider_kind
    ,provider_fingerprint
    ,provider
    ,state_status
    ,state_version
    ,state_errcode
    ,state_last_ready_version
    ,cleanup_requested_at
    ,cleanup_requested_by
    ,created_at
    ,created_by
    ,entrypoints
    ,version
FROM targets;

-- APPS

CREATE TABLE [deployment.apps]
(
    id TEXT NOT NULL,
    name TEXT NOT NULL,

    version_control_url TEXT NULL,
    version_control_token TEXT NULL,

    production_migration_target TEXT NULL,
    production_migration_from DATETIME NULL,
    production_migration_to DATETIME NULL,
    production_since DATETIME NOT NULL,
    production_cleaned BOOLEAN NOT NULL,
    production_config_target TEXT NOT NULL,
    production_config_vars JSON NULL,

    staging_migration_target TEXT NULL,
    staging_migration_from DATETIME NULL,
    staging_migration_to DATETIME NULL,
    staging_since DATETIME NOT NULL,
    staging_cleaned BOOLEAN NOT NULL,
    staging_config_target TEXT NOT NULL,
    staging_config_vars JSON NULL,

    created_at DATETIME NOT NULL,
    created_by TEXT NOT NULL,
    cleanup_requested_at DATETIME NULL,
    cleanup_requested_by TEXT NULL,
    version DATETIME NOT NULL,

    CONSTRAINT [deployment.pk_apps] PRIMARY KEY(id),
    CONSTRAINT [deployment.fk_apps_production_migration_target] FOREIGN KEY(production_migration_target) REFERENCES [deployment.targets](id) ON DELETE RESTRICT,
    CONSTRAINT [deployment.fk_apps_production_config_target] FOREIGN KEY(production_config_target) REFERENCES [deployment.targets](id) ON DELETE RESTRICT,
    CONSTRAINT [deployment.fk_apps_staging_migration_target] FOREIGN KEY(staging_migration_target) REFERENCES [deployment.targets](id) ON DELETE RESTRICT,
    CONSTRAINT [deployment.fk_apps_staging_config_target] FOREIGN KEY(staging_config_target) REFERENCES [deployment.targets](id) ON DELETE RESTRICT,
    CONSTRAINT [deployment.fk_apps_created_by] FOREIGN KEY(created_by) REFERENCES [auth.users](id) ON DELETE CASCADE,
    CONSTRAINT [deployment.fk_apps_cleanup_requested_by] FOREIGN KEY (cleanup_requested_by) REFERENCES [auth.users](id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX [deployment.idx_apps_name_production_config_target] ON [deployment.apps](name, production_config_target);
CREATE INDEX [deployment.idx_apps_name_production_migration_target] ON [deployment.apps](name, production_migration_target);
CREATE UNIQUE INDEX [deployment.idx_apps_name_staging_config_target] ON [deployment.apps](name, staging_config_target);
CREATE INDEX [deployment.idx_apps_name_staging_migration_target] ON [deployment.apps](name, staging_migration_target);
CREATE INDEX [deployment.idx_apps_production_config_target] ON [deployment.apps](production_config_target);
CREATE INDEX [deployment.idx_apps_production_migration_target] ON [deployment.apps](production_migration_target);
CREATE INDEX [deployment.idx_apps_staging_config_target] ON [deployment.apps](staging_config_target);
CREATE INDEX [deployment.idx_apps_staging_migration_target] ON [deployment.apps](staging_migration_target);
CREATE INDEX [deployment.idx_apps_created_by] ON [deployment.apps](created_by);
CREATE INDEX [deployment.idx_apps_cleanup_requested_by] ON [deployment.apps](cleanup_requested_by);
CREATE INDEX [deployment.idx_apps_version] ON [deployment.apps](version);

INSERT INTO [deployment.apps]
(
    id
    ,name
    ,version_control_url
    ,version_control_token
    ,production_since
    ,production_cleaned
    ,production_config_target
    ,production_config_vars
    ,staging_since
    ,staging_cleaned
    ,staging_config_target
    ,staging_config_vars
    ,created_at
    ,created_by
    ,version
)
SELECT
    id
    ,name
    ,version_control_url
    ,version_control_token
    ,production_version
    ,false
    ,production_target
    ,production_vars
    ,staging_version
    ,false
    ,staging_target
    ,staging_vars
    ,created_at
    ,created_by
    ,version
FROM apps;

-- DEPLOYMENTS

CREATE TABLE [deployment.deployments]
(
    app_id TEXT NOT NULL,
    deployment_number INTEGER NOT NULL,
    config_appid TEXT NOT NULL,
    config_appname TEXT NOT NULL,
    config_environment TEXT NOT NULL,
    config_target TEXT NOT NULL, -- No FK on config_target because we don't want to deal with a target deletion.
    config_vars JSON NULL,
    state_status INTEGER NOT NULL,
    state_errcode TEXT NULL,
    state_services JSON NULL,
    state_started_at DATETIME NULL,
    state_finished_at DATETIME NULL,
    source_discriminator TEXT NOT NULL,
    source JSON NOT NULL,
    requested_at DATETIME NOT NULL,
    requested_by TEXT NOT NULL,
    version DATETIME NOT NULL,

    CONSTRAINT [deployment.pk_deployments] PRIMARY KEY(app_id, deployment_number),
    CONSTRAINT [deployment.fk_deployments_app_id] FOREIGN KEY(app_id) REFERENCES [deployment.apps](id) ON DELETE CASCADE,
    CONSTRAINT [deployment.fk_deployments_requested_by] FOREIGN KEY(requested_by) REFERENCES [auth.users](id) ON DELETE CASCADE
);

CREATE INDEX [deployment.idx_deployments_app_id] ON [deployment.deployments](app_id);
CREATE INDEX [deployment.idx_deployments_target] ON [deployment.deployments](config_target);
CREATE INDEX [deployment.idx_deployments_state_status] ON [deployment.deployments](state_status);
CREATE INDEX [deployment.idx_deployments_requested_by] ON [deployment.deployments](requested_by);
CREATE INDEX [deployment.idx_deployments_version] ON [deployment.deployments](version);

INSERT INTO [deployment.deployments]
(
    app_id
    ,deployment_number
    ,config_appid
    ,config_appname
    ,config_environment
    ,config_target
    ,config_vars
    ,state_status
    ,state_errcode
    ,state_services
    ,state_started_at
    ,state_finished_at
    ,source_discriminator
    ,source
    ,requested_at
    ,requested_by
    ,version
)
SELECT
    app_id
    ,deployment_number
    ,config_appid
    ,config_appname
    ,config_environment
    ,config_target
    ,config_vars
    ,state_status
    ,state_errcode
    ,state_services
    ,state_started_at
    ,state_finished_at
    ,source_discriminator
    ,source
    ,requested_at
    ,requested_by
    ,version
FROM deployments;

-- TRIGGERS

-- When a deployment is marked as failed, remove all pending jobs for that resource.
-- This is to avoid running jobs that are no longer needed.
CREATE TRIGGER IF NOT EXISTS [deployment.on_deployment_failed_cleanup_jobs]
AFTER UPDATE ON [deployment.deployments]
    WHEN OLD.state_status != NEW.state_status AND NEW.state_status = 2 -- Only when the deployment goes to the failed state
BEGIN
    DELETE FROM [scheduler.scheduled_jobs]
    WHERE
        message_name = 'deployment.command.deploy'
        AND message_data ->> '$.app_id' = NEW.app_id
        AND message_data ->> '$.deployment_number' = NEW.deployment_number
        AND retrieved = 0;
END;

-- When a target is configured, no need to keep old configure jobs since they are outdated.
CREATE TRIGGER IF NOT EXISTS [deployment.on_target_configure_remove_outdated_jobs]
BEFORE INSERT ON [scheduler.scheduled_jobs]
    WHEN NEW.message_name = 'deployment.command.configure_target'
BEGIN
  DELETE FROM [scheduler.scheduled_jobs]
  WHERE
      message_name = 'deployment.command.configure_target'
      AND (message_data ->> '$.target_id') = (NEW.message_data ->> '$.target_id')
      AND retrieved = 0;
END;

-- CLEANUP OLD TABLES

DROP TABLE deployments;
DROP TABLE apps;
DROP TABLE targets;
DROP TABLE registries;
DROP TABLE users; -- Drop the users table here because data has been migrated (this is because I messed up a bit :/)
