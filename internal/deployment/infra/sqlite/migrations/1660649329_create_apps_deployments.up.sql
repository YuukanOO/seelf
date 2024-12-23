
CREATE TABLE apps (
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    vcs_url TEXT NULL,
    vcs_token TEXT NULL,
    env TEXT NULL,
    created_at DATETIME NOT NULL,
    created_by TEXT NOT NULL,
    cleanup_requested_at DATETIME NULL,
    cleanup_requested_by TEXT NULL,

    CONSTRAINT pk_apps PRIMARY KEY(id),
    CONSTRAINT unique_apps_name UNIQUE(name),
    CONSTRAINT fk_apps_created_by FOREIGN KEY(created_by) REFERENCES [auth.users](id) ON DELETE CASCADE,
    CONSTRAINT fk_apps_cleanup_requested_by FOREIGN KEY(cleanup_requested_by) REFERENCES [auth.users](id) ON DELETE CASCADE
);

CREATE TABLE deployments (
    app_id TEXT NOT NULL,
    deployment_number INTEGER NOT NULL,
    path TEXT NOT NULL,
    config_appname TEXT NOT NULL,
    config_environment TEXT NOT NULL,
    config_env TEXT NULL,
    state_status INTEGER NOT NULL,
    state_errcode TEXT NULL,
    state_services TEXT NULL,
    state_logfile TEXT NOT NULL,
    state_started_at DATETIME NULL,
    state_finished_at DATETIME NULL,
    trigger_kind TEXT NOT NULL,
    trigger_data TEXT NOT NULL,
    requested_at DATETIME NOT NULL,
    requested_by TEXT NOT NULL,

    CONSTRAINT pk_deployments PRIMARY KEY(app_id, deployment_number),
    CONSTRAINT fk_deployments_app_id FOREIGN KEY(app_id) REFERENCES apps(id) ON DELETE CASCADE,
    CONSTRAINT fk_deployments_requested_by FOREIGN KEY(requested_by) REFERENCES [auth.users](id) ON DELETE CASCADE
);