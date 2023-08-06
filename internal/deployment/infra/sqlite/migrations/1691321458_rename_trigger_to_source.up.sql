ALTER TABLE deployments RENAME COLUMN trigger_kind TO source_kind;
ALTER TABLE deployments RENAME COLUMN trigger_data TO source_data;