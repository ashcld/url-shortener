-- migrate: no_transaction

DROP INDEX CONCURRENTLY IF EXISTS idx_urls_created_at;
