CREATE TABLE jobs (
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    payload TEXT,
    queued_at DATETIME NOT NULL,
    errcode TEXT NULL,

    CONSTRAINT pk_jobs PRIMARY KEY(id)
);