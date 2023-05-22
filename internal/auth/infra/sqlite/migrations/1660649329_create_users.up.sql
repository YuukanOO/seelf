CREATE TABLE users (
    id TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    api_key TEXT NOT NULL,
    registered_at DATETIME NOT NULL,

    CONSTRAINT pk_users PRIMARY KEY(id),
    CONSTRAINT unique_users_email UNIQUE(email),
    CONSTRAINT unique_users_api_key UNIQUE(api_key)
);
