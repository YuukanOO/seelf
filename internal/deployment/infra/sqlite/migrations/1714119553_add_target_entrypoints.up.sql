ALTER TABLE targets ADD entrypoints TEXT NOT NULL DEFAULT '{}';

-- Update all targets with a new version to force the reconfiguration since the default http entrypoint name have changed.
UPDATE targets
SET
    state_version = '2024-05-19T00:00:00Z'
    , state_status = 0;

-- And schedule it
INSERT INTO [scheduler.scheduled_jobs] (
    id
    ,[group]
    ,message_name
    ,message_data
    ,queued_at
    ,not_before
    ,retrieved
)
SELECT
    id
    , 'deployment.target.configure.' || id
    , 'deployment.command.configure_target'
    , '{"id":"' || id || '","version":"2024-05-19T00:00:00Z"}'
    , DATETIME()
    , DATETIME()
    , false
FROM targets;

-- When a deployment is marked as failed, remove all pending jobs for that resource.
-- This is to avoid running jobs that are no longer needed.
CREATE TRIGGER IF NOT EXISTS on_deployment_failed_cleanup_jobs AFTER UPDATE ON deployments
    WHEN OLD.state_status != NEW.state_status AND NEW.state_status = 2 -- Only when the deployment goes to the failed state
BEGIN
    DELETE FROM [scheduler.scheduled_jobs]
    WHERE
        message_data ->> '$.app_id' = NEW.app_id
        AND message_data ->> '$.deployment_number' = NEW.deployment_number
        AND retrieved = 0;
END