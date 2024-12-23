CREATE TABLE [auth.users]
(
    id TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    api_key TEXT NOT NULL,
    registered_at DATETIME NOT NULL,
    version DATETIME NOT NULL,

    CONSTRAINT [auth.pk_users] PRIMARY KEY(id)
);

CREATE UNIQUE INDEX [auth.idx_users_email] ON [auth.users](email);
CREATE UNIQUE INDEX [auth.idx_users_api_key] ON [auth.users](api_key);
CREATE INDEX [auth.idx_users_version] ON [auth.users](version);

INSERT INTO [auth.users]
(
    id
    ,email
    ,password_hash
    ,api_key
    ,registered_at
    ,version
)
SELECT 
    id
    ,email
    ,password_hash
    ,api_key
    ,registered_at
    ,version
FROM users;

-- Do not drop the users table right now because it will empty the deployments data.