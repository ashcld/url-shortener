CREATE TABLE IF NOT EXISTS urls (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    short_code   TEXT        NOT NULL,
    original_url TEXT        NOT NULL,
    user_id      BIGINT,
    expires_at   TIMESTAMPTZ,
    click_count  BIGINT      NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_short_code UNIQUE (short_code),
    CONSTRAINT chk_short_code_len CHECK (length(short_code) BETWEEN 4 AND 32),
    CONSTRAINT chk_original_url CHECK (
        original_url ~ '^https?://.+'
    )
);

CREATE INDEX idx_urls_short_code ON urls (short_code);
CREATE INDEX idx_urls_user_id    ON urls (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_urls_expires_at ON urls (expires_at) WHERE expires_at IS NOT NULL;
