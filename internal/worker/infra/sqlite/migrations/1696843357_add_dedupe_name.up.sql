ALTER TABLE jobs ADD dedupe_name TEXT NOT NULL DEFAULT '';
ALTER TABLE jobs ADD retrieved BOOLEAN NOT NULL DEFAULT false;
CREATE INDEX idx_jobs_dedupe_name ON jobs(dedupe_name);