CREATE TEMPORARY TABLE tmp_scheduled_jobs AS
SELECT * FROM [scheduler.scheduled_jobs];

DROP TABLE [scheduler.scheduled_jobs];

CREATE TABLE [scheduler.scheduled_jobs]
(
    key TEXT NOT NULL,
    topic TEXT NOT NULL,
    message_name TEXT NOT NULL,
    message_data JSON NOT NULL,
    queued_at DATETIME NOT NULL,
    state_status INTEGER NOT NULL,
    state_not_before DATETIME NOT NULL,
    state_errcode TEXT NULL,
    version DATETIME NOT NULL,

    CONSTRAINT [scheduler.pk_scheduled_jobs] PRIMARY KEY(key)
);

CREATE INDEX [scheduler.idx_scheduled_jobs_topic] ON [scheduler.scheduled_jobs](topic);
CREATE INDEX [scheduler.idx_scheduled_jobs_message_name] ON [scheduler.scheduled_jobs](message_name);
CREATE INDEX [scheduler.idx_scheduled_jobs_version] ON [scheduler.scheduled_jobs](version);

INSERT INTO [scheduler.scheduled_jobs]
(
    key
    ,topic
    ,message_name
    ,message_data
    ,queued_at
    ,state_status
    ,state_not_before
    ,state_errcode
    ,version
)
SELECT
    id
    ,[group]
    ,message_name
    ,message_data
    ,queued_at
    ,0
    ,not_before
    ,errcode
    ,'2006-01-02 15:04:05.999999999+00:00'
FROM tmp_scheduled_jobs;

-- When a job is delayed, delay other jobs in the same topic
CREATE TRIGGER IF NOT EXISTS [scheduler.on_job_delayed_keep_topic_order]
AFTER UPDATE ON [scheduler.scheduled_jobs]
    WHEN NEW.state_not_before != OLD.state_not_before
BEGIN
    UPDATE [scheduler.scheduled_jobs]
    SET
        state_not_before = v.updated_date
        ,state_status = 0
    FROM (
        SELECT
            key
            , DATETIME(NEW.state_not_before, '+' || CAST(1 * ROW_NUMBER() OVER (ORDER BY queued_at) AS TEXT) || ' seconds') AS updated_date
        FROM [scheduler.scheduled_jobs]
        WHERE key != NEW.key AND topic = NEW.topic
    ) v
    WHERE [scheduler.scheduled_jobs].key = v.key;
END;

DROP TABLE tmp_scheduled_jobs;