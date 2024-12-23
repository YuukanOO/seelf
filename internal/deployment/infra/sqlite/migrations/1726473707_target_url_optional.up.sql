-- since we cannot change the url nullable property easily, we have to do this steps
CREATE TEMPORARY TABLE tmp_targets AS
SELECT *
FROM targets;

CREATE TEMPORARY TABLE tmp_apps AS
SELECT *
FROM apps;

CREATE TEMPORARY TABLE tmp_deployments AS
SELECT *
FROM deployments;

DROP TABLE deployments;
DROP TABLE apps;
DROP TABLE targets;

CREATE TABLE targets (
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    url TEXT NULL,
    provider_kind TEXT NOT NULL,
    provider_fingerprint TEXT NOT NULL,
    provider TEXT NOT NULL,
    state_status INTEGER NOT NULL,
    state_version DATETIME NOT NULL,
    state_errcode TEXT NULL,
    state_last_ready_version DATETIME NULL,
    cleanup_requested_at DATETIME NULL,
    cleanup_requested_by TEXT NULL,
    created_at DATETIME NOT NULL,
    created_by TEXT NOT NULL,
    entrypoints TEXT NOT NULL DEFAULT '{}',

    CONSTRAINT pk_targets PRIMARY KEY(id),
    CONSTRAINT fk_targets_created_by FOREIGN KEY(created_by) REFERENCES [auth.users](id) ON DELETE CASCADE,
    CONSTRAINT unique_targets_url UNIQUE(url), -- unique url among all targets
    CONSTRAINT unique_targets_provider_fingerprint UNIQUE(provider_fingerprint) -- unique provider fingerprint
);

CREATE TABLE apps (
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    version_control_url TEXT NULL,
    version_control_token TEXT NULL,
    production_target TEXT NOT NULL,
    production_version DATETIME NOT NULL,
    production_vars TEXT NULL,
    staging_target TEXT NOT NULL,
    staging_version DATETIME NOT NULL,
    staging_vars TEXT NULL,
    created_at DATETIME NOT NULL,
    created_by TEXT NOT NULL,
    cleanup_requested_at DATETIME NULL,
    cleanup_requested_by TEXT NULL,

    CONSTRAINT pk_apps PRIMARY KEY(id),
    CONSTRAINT unique_apps_name_production_target UNIQUE(name, production_target),
    CONSTRAINT unique_apps_name_staging_target UNIQUE(name, staging_target),    
    CONSTRAINT fk_apps_production_target FOREIGN KEY(production_target) REFERENCES targets(id) ON DELETE RESTRICT,
    CONSTRAINT fk_apps_staging_target FOREIGN KEY(staging_target) REFERENCES targets(id) ON DELETE RESTRICT,
    CONSTRAINT fk_apps_created_by FOREIGN KEY(created_by) REFERENCES [auth.users](id) ON DELETE CASCADE,
    CONSTRAINT fk_apps_cleanup_requested_by FOREIGN KEY (cleanup_requested_by) REFERENCES [auth.users](id) ON DELETE CASCADE
);

CREATE TABLE deployments
(
    app_id TEXT NOT NULL,
    deployment_number INTEGER NOT NULL,
    config_appid TEXT NOT NULL,
    config_appname TEXT NOT NULL,
    config_environment TEXT NOT NULL,
    config_target TEXT NOT NULL,
    -- No FK on config_target because we don't want to deal with a target deletion.
    config_vars TEXT NULL,
    state_status INTEGER NOT NULL,
    state_errcode TEXT NULL,
    state_services TEXT NULL,
    state_started_at DATETIME NULL,
    state_finished_at DATETIME NULL,
    source_discriminator TEXT NOT NULL,
    source TEXT NOT NULL,
    requested_at DATETIME NOT NULL,
    requested_by TEXT NOT NULL,

    CONSTRAINT pk_deployments PRIMARY KEY(app_id, deployment_number),
    CONSTRAINT fk_deployments_app_id FOREIGN KEY(app_id) REFERENCES apps(id) ON DELETE CASCADE,
    CONSTRAINT fk_deployments_requested_by FOREIGN KEY(requested_by) REFERENCES [auth.users](id) ON DELETE CASCADE
);

CREATE INDEX idx_deployments_target ON deployments(config_target);
CREATE INDEX idx_deployments_state_status ON deployments(state_status);

INSERT INTO targets
SELECT *
FROM tmp_targets;

INSERT INTO apps
SELECT *
FROM tmp_apps;

INSERT INTO deployments
SELECT *
FROM tmp_deployments;

DROP TABLE tmp_targets;
DROP TABLE tmp_apps;
DROP TABLE tmp_deployments;
