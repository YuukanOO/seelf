DROP TRIGGER IF EXISTS on_deployment_failed_cleanup_jobs;

-- When a deployment is marked as failed, remove all pending jobs for that resource.
-- This is to avoid running jobs that are no longer needed.
CREATE TRIGGER IF NOT EXISTS on_deployment_failed_cleanup_jobs AFTER UPDATE ON deployments
    WHEN OLD.state_status != NEW.state_status AND NEW.state_status = 2 -- Only when the deployment goes to the failed state
BEGIN
    DELETE FROM scheduled_jobs
    WHERE
        message_name = 'deployment.command.deploy'
        AND message_data ->> '$.app_id' = NEW.app_id
        AND message_data ->> '$.deployment_number' = NEW.deployment_number
        AND retrieved = 0;
END;

-- Since deployment jobs group has changed, just mark all pending deployments as failed
UPDATE deployments
SET
  state_status = 2,
  state_errcode = 'seelf_incompatible_upgrade',
  state_started_at = datetime('now'),
  state_finished_at = datetime('now')
WHERE state_status = 0; -- Running jobs will be failed with the server_reset error so no need to handle them here

ALTER TABLE deployments ADD version DATETIME NOT NULL DEFAULT '2006-01-02 15:04:05.999999999+00:00';
CREATE INDEX idx_deployments_version ON deployments(version);
ALTER TABLE apps ADD version DATETIME NOT NULL DEFAULT '2006-01-02 15:04:05.999999999+00:00';
CREATE INDEX idx_apps_version ON apps(version);
ALTER TABLE registries ADD version DATETIME NOT NULL DEFAULT '2006-01-02 15:04:05.999999999+00:00';
CREATE INDEX idx_registries_version ON registries(version);
ALTER TABLE targets ADD version DATETIME NOT NULL DEFAULT '2006-01-02 15:04:05.999999999+00:00';
CREATE INDEX idx_targets_version ON targets(version);

ALTER TABLE apps ADD history JSON NOT NULL DEFAULT '{}';
UPDATE apps SET history = '{"production": ["' || production_target || '"], "staging": ["'|| staging_target ||'"]}';

-- Since the group has changed for configure and cleanup, just update those jobs.
UPDATE scheduled_jobs
SET
    [group] = (message_data ->> '$.id')
    ,message_data = '{"target_id": "' || (message_data ->> '$.id') || '"}'
WHERE message_name = 'deployment.command.cleanup_target';

UPDATE scheduled_jobs
SET
    [group] = (message_data ->> '$.id')
    ,message_data = '{"target_id": "' || (message_data ->> '$.id') || '", "version": "' || (message_data ->> '$.version') || '"}'
WHERE message_name = 'deployment.command.configure_target';

-- Since those messages no longer exists.
DELETE FROM scheduled_jobs WHERE message_name IN ('deployment.command.delete_target', 'deployment.command.delete_app');

-- When a target is configured, no need to keep old configure jobs since they are outdated.
CREATE TRIGGER IF NOT EXISTS on_target_configure_remove_outdated_jobs
BEFORE INSERT ON scheduled_jobs
WHEN NEW.message_name = 'deployment.command.configure_target'
BEGIN
  DELETE FROM scheduled_jobs
  WHERE
      message_name = 'deployment.command.configure_target'
      AND (message_data ->> '$.target_id') = (NEW.message_data ->> '$.target_id')
      AND retrieved = 0;
END;