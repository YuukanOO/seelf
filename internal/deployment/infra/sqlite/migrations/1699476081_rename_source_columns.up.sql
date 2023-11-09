ALTER TABLE deployments RENAME COLUMN source_kind TO source_discriminator;
ALTER TABLE deployments RENAME COLUMN source_data TO source;