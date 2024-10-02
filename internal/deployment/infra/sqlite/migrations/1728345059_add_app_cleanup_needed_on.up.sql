-- Since deployment jobs group has changed, just mark all pending deployments as failed
UPDATE deployments
SET
  state_status = 2,
  state_errcode = 'seelf_incompatible_upgrade',
  state_started_at = datetime('now'),
  state_finished_at = datetime('now')
WHERE state_status = 0; -- Running jobs will be failed with the server_reset error so no need to handle them here

ALTER TABLE apps ADD history TEXT NOT NULL DEFAULT '{}';
UPDATE apps SET history = '{"production": ["' || production_target || '"], "staging": ["'|| staging_target ||'"]}';

-- Since the group has changed for configure and cleanup, just update those jobs.
UPDATE scheduled_jobs
SET
    [group] = resource_id
WHERE message_name IN ('deployment.command.configure_target', 'deployment.command.cleanup_target');

-- Since those messages no longer exists.
DELETE FROM scheduled_jobs WHERE message_name IN ('deployment.command.delete_target', 'deployment.command.delete_app');

-- When a target is configured, no need to keep old configure jobs since they are outdated.
CREATE TRIGGER IF NOT EXISTS on_target_configure_remove_outdated_jobs
BEFORE INSERT ON scheduled_jobs
WHEN NEW.message_name = 'deployment.command.configure_target'
BEGIN
  DELETE FROM scheduled_jobs
  WHERE
      resource_id = NEW.resource_id
      AND message_name = 'deployment.command.configure_target'
      AND retrieved = 0;
END