CREATE TABLE scheduled_jobs (
    id TEXT NOT NULL,
    dedupe_name TEXT NOT NULL,
    message_name TEXT NOT NULL,
    message_data TEXT NOT NULL,
    policy INTEGER NOT NULL,
    queued_at DATETIME NOT NULL,
    errcode TEXT NULL,
    retrieved BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT pk_scheduled_jobs PRIMARY KEY(id)
);

CREATE INDEX idx_scheduled_jobs_dedupe_name ON scheduled_jobs(dedupe_name);
