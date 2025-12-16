-- Create webhook_config table schema
-- This migration creates the webhook_configs table with all necessary constraints and indexes
-- Ensure UUID extension is enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- Create PostgreSQL enum for event types
CREATE TYPE event_type AS ENUM ('CREDIT', 'DEBIT');
-- Create webhook_configs table
CREATE TABLE IF NOT EXISTS webhook_configs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    event_type event_type NOT NULL,
    webhook_url TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    timeout_ms INTEGER DEFAULT 30000,
    created_at TIMESTAMP DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW() NOT NULL,
    deleted_at TIMESTAMP
);
-- Create indexes for webhook_configs table
CREATE INDEX IF NOT EXISTS idx_webhook_configs_event_type_active ON webhook_configs(event_type, is_active)
WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_webhook_configs_name ON webhook_configs(name);
CREATE INDEX IF NOT EXISTS idx_webhook_configs_created_at ON webhook_configs(created_at);