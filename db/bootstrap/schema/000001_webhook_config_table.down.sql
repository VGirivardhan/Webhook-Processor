-- Drop webhook_config table schema
-- This migration removes the webhook_configs table and related objects
-- Drop indexes first
DROP INDEX IF EXISTS idx_webhook_configs_created_at;
DROP INDEX IF EXISTS idx_webhook_configs_name;
DROP INDEX IF EXISTS idx_webhook_configs_event_type_active;
-- Drop the table
DROP TABLE IF EXISTS webhook_configs;
