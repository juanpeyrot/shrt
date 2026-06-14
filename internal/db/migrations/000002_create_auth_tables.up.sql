CREATE TABLE users (
    id           UUID        PRIMARY KEY,
    display_name TEXT        NOT NULL,
    email        TEXT        NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth_methods (
    id                   UUID        PRIMARY KEY,
    user_id              UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider             TEXT        NOT NULL,
    provider_user_id     TEXT,
    password_hash        TEXT,
    refresh_token_hash   TEXT,
    refresh_token_jwt_id TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, provider)
);
