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

DELETE FROM deployments;
DELETE FROM apps;
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
    CONSTRAINT fk_targets_created_by FOREIGN KEY(created_by) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT unique_targets_url UNIQUE(url), -- unique url among all targets
    CONSTRAINT unique_targets_provider_fingerprint UNIQUE(provider_fingerprint) -- unique provider fingerprint
);

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