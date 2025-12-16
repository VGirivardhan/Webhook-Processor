-- Drop webhook_queue table schema
-- This migration removes the webhook_queue table and related objects
-- Drop indexes first
DROP INDEX IF EXISTS idx_webhook_queue_event_id;
DROP INDEX IF EXISTS idx_webhook_queue_queue_id;
DROP INDEX IF EXISTS idx_webhook_queue_config_id;
DROP INDEX IF EXISTS idx_webhook_queue_created_at;
DROP INDEX IF EXISTS idx_webhook_queue_event_type;
DROP INDEX IF EXISTS idx_webhook_queue_status_next_retry;
-- Drop the table
DROP TABLE IF EXISTS webhook_queue;
-- Drop the webhook_status enum
DROP TYPE IF EXISTS webhook_status;