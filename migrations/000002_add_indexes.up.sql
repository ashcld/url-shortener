-- migrate: no_transaction

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_urls_created_at
    ON urls (created_at DESC);
