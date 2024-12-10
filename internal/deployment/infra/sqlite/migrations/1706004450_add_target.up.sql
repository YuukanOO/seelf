-- Since sqlite does not support adding constraints afterward, use temp tables to migrate data

CREATE TABLE targets (
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
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

    CONSTRAINT pk_targets PRIMARY KEY(id),
    CONSTRAINT fk_targets_created_by FOREIGN KEY(created_by) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT unique_targets_domain UNIQUE(url), -- unique url among all targets
    CONSTRAINT unique_targets_provider_fingerprint UNIQUE(provider_fingerprint) -- unique provider fingerprint
);

-- Creates a default target if at least one app exists, else do nothing
INSERT INTO targets (
    id
    ,name
    ,url
    ,provider_kind
    ,provider_fingerprint
    ,provider
    ,state_status
    ,state_version
    ,created_at
    ,created_by
)
SELECT 
    '2bRUdQnyRELMqyh9gFLQV1s0cqv'
    ,'local'
    ,'http://docker.localhost'
    ,'docker'
    ,''
    ,'{}'
    ,0
    ,'2024-01-23T10:07:30Z'
    ,DATETIME()
    ,(SELECT id FROM users LIMIT 1)
FROM apps LIMIT 1;

-- Schedule a target configuration if a target was created above
INSERT INTO scheduled_jobs (
    id
    ,[group]
    ,message_name
    ,message_data
    ,queued_at
    ,not_before
    ,retrieved
)
SELECT
    id
    ,'deployment.target.configure.' || id
    ,'deployment.command.configure_target'
    ,'{"id":"' || id || '","version":"2024-01-23T10:07:30Z"}'
    ,DATETIME()
    ,DATETIME()
    ,false
FROM targets;

CREATE TEMPORARY TABLE tmp_deployments
AS SELECT * FROM deployments;

CREATE TEMPORARY TABLE tmp_apps AS
SELECT * FROM apps;

DROP TABLE deployments;
DROP TABLE apps;

-- Create the new apps table with proper columns
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
    CONSTRAINT fk_apps_created_by FOREIGN KEY(created_by) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_apps_cleanup_requested_by FOREIGN KEY (cleanup_requested_by) REFERENCES users (id) ON DELETE CASCADE
);

-- Transfer old apps data into the new structure, trying to retrieve old configuration in the meantime
INSERT INTO apps (
    id
    ,name
    ,version_control_url
    ,version_control_token
    ,production_target
    ,production_version
    ,production_vars
    ,staging_target
    ,staging_version
    ,staging_vars
    ,created_at
    ,created_by
    ,cleanup_requested_at
    ,cleanup_requested_by
)
SELECT
    id
	,name
	,vcs_url
	,vcs_token
	,'2bRUdQnyRELMqyh9gFLQV1s0cqv'
    ,created_at
	,(
        SELECT env ->> "$.production"
        FROM tmp_apps a
        WHERE a.id = tmp_apps.id
    )
	,'2bRUdQnyRELMqyh9gFLQV1s0cqv'
    ,created_at
	,(
        SELECT env ->> "$.staging"
        FROM tmp_apps a
        WHERE a.id = tmp_apps.id
    )
	,created_at
    ,created_by 
	,cleanup_requested_at
	,cleanup_requested_by
FROM tmp_apps;

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
    CONSTRAINT fk_deployments_requested_by FOREIGN KEY(requested_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_deployments_target ON deployments(config_target);
CREATE INDEX idx_deployments_state_status ON deployments(state_status);

INSERT INTO deployments (
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
)
SELECT
    app_id
	,deployment_number
    ,app_id
	,config_appname
	,config_environment
	,'2bRUdQnyRELMqyh9gFLQV1s0cqv'
	,config_env
	,state_status
	,state_errcode
	,state_services
	,state_started_at
	,state_finished_at
	,source_discriminator
	,source
	,requested_at
	,requested_by
FROM tmp_deployments;

-- Remove old tables
DROP TABLE tmp_apps;
DROP TABLE tmp_deployments;