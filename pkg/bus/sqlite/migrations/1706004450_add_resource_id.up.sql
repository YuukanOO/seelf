ALTER TABLE scheduled_jobs RENAME TO tmp_scheduled_jobs;
DROP INDEX IF EXISTS idx_scheduled_jobs_dedupe_name;

CREATE TABLE scheduled_jobs
(
    id TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    [group] TEXT NOT NULL,
    message_name TEXT NOT NULL,
    message_data TEXT NOT NULL,
    queued_at DATETIME NOT NULL,
    not_before DATETIME NOT NULL,
    policy INTEGER NOT NULL,
    errcode TEXT NULL,
    retrieved BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT pk_scheduled_jobs PRIMARY KEY(id)
);

INSERT INTO scheduled_jobs (
    id,
    resource_id,
    [group],
    message_name,
    message_data,
    queued_at,
    not_before,
    policy,
    errcode,
    retrieved
)
SELECT 
    id,
    id,
    dedupe_name,
    message_name,
    message_data,
    queued_at,
    queued_at,
    0,
    errcode,
    retrieved
FROM tmp_scheduled_jobs;

CREATE INDEX idx_scheduled_jobs_resource_id ON scheduled_jobs(resource_id);
CREATE INDEX idx_scheduled_jobs_group ON scheduled_jobs([group]);

DROP TABLE tmp_scheduled_jobs;