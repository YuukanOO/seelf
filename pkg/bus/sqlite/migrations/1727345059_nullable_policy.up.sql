CREATE TEMPORARY TABLE tmp_scheduled_jobs AS
SELECT * FROM scheduled_jobs;

DROP TABLE scheduled_jobs;

-- Make the policy and resource_id columns nullable since it's not used anymore.
-- Since migrations are executed by domain orders (scheduler, auth, deployment) and
-- I have failed by making deployment rely on the scheduled_jobs table, I must keep
-- it or else I have to update the migration (which I think is worse).
CREATE TABLE scheduled_jobs
(
    id TEXT NOT NULL,
    [group] TEXT NOT NULL,
    message_name TEXT NOT NULL,
    message_data JSON NOT NULL,
    queued_at DATETIME NOT NULL,
    not_before DATETIME NOT NULL,
    errcode TEXT NULL,
    retrieved BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT pk_scheduled_jobs PRIMARY KEY(id)
);

INSERT INTO scheduled_jobs
SELECT
    id 
    ,[group]
    ,message_name 
    ,message_data 
    ,queued_at 
    ,not_before 
    ,errcode 
    ,retrieved
FROM tmp_scheduled_jobs;

CREATE INDEX idx_scheduled_jobs_group ON scheduled_jobs([group]);
CREATE INDEX idx_scheduled_jobs_message_name ON scheduled_jobs(message_name);

DROP TABLE tmp_scheduled_jobs;