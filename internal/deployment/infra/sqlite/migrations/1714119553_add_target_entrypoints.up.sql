ALTER TABLE targets ADD entrypoints TEXT NOT NULL DEFAULT '{}';

-- When a deployment is marked as failed, remove all pending jobs for that resource.
-- This is to avoid running jobs that are no longer needed.
CREATE TRIGGER IF NOT EXISTS on_deployment_failed_cleanup_jobs AFTER UPDATE ON deployments
    WHEN OLD.state_status != NEW.state_status AND NEW.state_status = 2 -- Only when the deployment goes to the failed state
BEGIN
    DELETE FROM scheduled_jobs
    WHERE
        resource_id = NEW.app_id || '-' || NEW.deployment_number
        AND retrieved = 0;
END