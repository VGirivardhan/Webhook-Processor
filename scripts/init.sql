-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- Create PostgreSQL enums for type safety
CREATE TYPE event_type AS ENUM ('CREDIT', 'DEBIT');
CREATE TYPE webhook_status AS ENUM ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED');
-- Create webhook_configs table
CREATE TABLE IF NOT EXISTS webhook_configs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    event_type event_type NOT NULL,
    webhook_url TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    max_retries INTEGER DEFAULT 6,
    timeout_ms INTEGER DEFAULT 30000,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
-- Create webhook_queue table
CREATE TABLE IF NOT EXISTS webhook_queue (
    id BIGSERIAL PRIMARY KEY,
    queue_id UUID DEFAULT uuid_generate_v4() UNIQUE NOT NULL,
    -- Event information
    event_type event_type NOT NULL,
    event_id VARCHAR(255) NOT NULL,
    -- Webhook details  
    config_id BIGINT NOT NULL REFERENCES webhook_configs(id),
    webhook_url TEXT NOT NULL,
    -- Processing status
    status webhook_status NOT NULL DEFAULT 'PENDING',
    -- Retry tracking (max 6 retries as per documentation)
    retry_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMP NOT NULL DEFAULT NOW(),
    -- Retry 0 (initial attempt)
    retry_0_started_at TIMESTAMP,
    retry_0_completed_at TIMESTAMP,
    retry_0_duration_ms BIGINT,
    retry_0_http_status INTEGER,
    retry_0_response_body TEXT,
    retry_0_error TEXT,
    -- Retry 1
    retry_1_started_at TIMESTAMP,
    retry_1_completed_at TIMESTAMP,
    retry_1_duration_ms BIGINT,
    retry_1_http_status INTEGER,
    retry_1_response_body TEXT,
    retry_1_error TEXT,
    -- Retry 2
    retry_2_started_at TIMESTAMP,
    retry_2_completed_at TIMESTAMP,
    retry_2_duration_ms BIGINT,
    retry_2_http_status INTEGER,
    retry_2_response_body TEXT,
    retry_2_error TEXT,
    -- Retry 3
    retry_3_started_at TIMESTAMP,
    retry_3_completed_at TIMESTAMP,
    retry_3_duration_ms BIGINT,
    retry_3_http_status INTEGER,
    retry_3_response_body TEXT,
    retry_3_error TEXT,
    -- Retry 4
    retry_4_started_at TIMESTAMP,
    retry_4_completed_at TIMESTAMP,
    retry_4_duration_ms BIGINT,
    retry_4_http_status INTEGER,
    retry_4_response_body TEXT,
    retry_4_error TEXT,
    -- Retry 5
    retry_5_started_at TIMESTAMP,
    retry_5_completed_at TIMESTAMP,
    retry_5_duration_ms BIGINT,
    retry_5_http_status INTEGER,
    retry_5_response_body TEXT,
    retry_5_error TEXT,
    -- Retry 6 (final retry)
    retry_6_started_at TIMESTAMP,
    retry_6_completed_at TIMESTAMP,
    retry_6_duration_ms BIGINT,
    retry_6_http_status INTEGER,
    retry_6_response_body TEXT,
    retry_6_error TEXT,
    -- General tracking
    last_error TEXT,
    last_http_status INTEGER,
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    processing_started_at TIMESTAMP,
    completed_at TIMESTAMP,
    deleted_at TIMESTAMP
);
-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_webhook_queue_status_next_retry ON webhook_queue(status, next_retry_at)
WHERE status = 'PENDING';
CREATE INDEX IF NOT EXISTS idx_webhook_queue_event_type ON webhook_queue(event_type);
CREATE INDEX IF NOT EXISTS idx_webhook_queue_created_at ON webhook_queue(created_at);
CREATE INDEX IF NOT EXISTS idx_webhook_configs_event_type_active ON webhook_configs(event_type, is_active)
WHERE is_active = true;
-- Insert sample webhook configurations
INSERT INTO webhook_configs (
        name,
        event_type,
        webhook_url,
        url_params,
        is_active,
        max_retries
    )
VALUES (
        'Credit Postback',
        'CREDIT',
        'https://httpbin.org/get',
        '{"source": "webhook-processor", "type": "postback"}',
        true,
        6
    ),
    (
        'Debit Chargeback',
        'DEBIT',
        'https://httpbin.org/get',
        '{"source": "webhook-processor", "type": "chargeback"}',
        true,
        6
    ) ON CONFLICT DO NOTHING;