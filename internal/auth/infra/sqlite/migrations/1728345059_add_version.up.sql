ALTER TABLE users ADD version DATETIME NOT NULL DEFAULT '2006-01-02 15:04:05.999999999+00:00';
CREATE INDEX idx_users_version ON users(version);