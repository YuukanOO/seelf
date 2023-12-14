-- Delete unused worker stuff since I now use a scheduler adapter
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS worker_schema_migrations;