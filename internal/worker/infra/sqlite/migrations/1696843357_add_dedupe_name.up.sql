-- Now use period to namespace job names
UPDATE jobs
SET name = replace(name, ':', '.');

ALTER TABLE jobs ADD dedupe_name TEXT NOT NULL DEFAULT '';
ALTER TABLE jobs ADD retrieved BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE jobs RENAME COLUMN name TO data_discriminator;
ALTER TABLE jobs RENAME COLUMN payload TO data;
CREATE INDEX idx_jobs_dedupe_name ON jobs(dedupe_name);
CREATE INDEX idx_jobs_data_discriminator ON jobs(data_discriminator);
