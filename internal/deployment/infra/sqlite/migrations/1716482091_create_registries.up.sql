CREATE TABLE registries (
    id TEXT NOT NULL
    ,name TEXT NOT NULL
    ,url TEXT NOT NULL
    ,credentials_username TEXT NULL
    ,credentials_password TEXT NULL
    ,created_at DATETIME NOT NULL
    ,created_by TEXT NOT NULL
    ,CONSTRAINT pk_registries PRIMARY KEY(id)
    ,CONSTRAINT unique_registries_url UNIQUE(url)
    ,CONSTRAINT fk_registries_created_by FOREIGN KEY(created_by) REFERENCES [auth.users](id) ON DELETE CASCADE
);