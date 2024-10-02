ALTER TABLE scheduled_jobs DROP COLUMN policy;

-- Make the policy column nullable since it's not used anymore.
-- Since migrations are executed by domain orders (scheduler, auth, deployment) and
-- I have failed by making deployment rely on the scheduled_jobs table, I must keep
-- it or else I have to update the migration (which I think is worse).
ALTER TABLE scheduled_jobs ADD policy INTEGER NULL;
