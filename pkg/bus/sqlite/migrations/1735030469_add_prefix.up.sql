CREATE TABLE [scheduler.scheduled_jobs]
(
    id TEXT NOT NULL,
    [group] TEXT NOT NULL,
    message_name TEXT NOT NULL,
    message_data JSON NOT NULL,
    queued_at DATETIME NOT NULL,
    not_before DATETIME NOT NULL,
    errcode TEXT NULL,
    retrieved BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT [scheduler.pk_scheduled_jobs] PRIMARY KEY(id)
);

CREATE INDEX [scheduler.idx_scheduled_jobs_group] ON [scheduler.scheduled_jobs]([group]);
CREATE INDEX [scheduler.idx_scheduled_jobs_message_name] ON [scheduler.scheduled_jobs](message_name);

INSERT INTO [scheduler.scheduled_jobs]
(
    id
    ,[group]
    ,message_name
    ,message_data
    ,queued_at
    ,not_before
    ,errcode
    ,retrieved
)
SELECT
    id
    ,[group]
    ,message_name
    ,message_data
    ,queued_at
    ,not_before
    ,errcode
    ,retrieved
FROM scheduled_jobs;

DROP TABLE scheduled_jobs;
