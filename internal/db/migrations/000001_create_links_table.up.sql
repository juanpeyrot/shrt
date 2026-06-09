CREATE TABLE links (
    id          UUID        PRIMARY KEY,
    short_code  TEXT        NOT NULL UNIQUE,
    original_url TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ,
    deleted_at  TIMESTAMPTZ,
    click_count BIGINT      NOT NULL DEFAULT 0
);
