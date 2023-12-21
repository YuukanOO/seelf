-- Since this migration remove the old jobs table, fail all pending deployments because the job will be lost
UPDATE deployments
SET
  state_status = 2,
  state_errcode = 'seelf_incompatible_upgrade',
  state_started_at = datetime('now'),
  state_finished_at = datetime('now')
WHERE state_status = 0; -- Running jobs will be failed with the server_reset error so no need to handle them here

-- Delete unused worker stuff since I now use a scheduler adapter
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS worker_schema_migrations;