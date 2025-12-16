-- Insert sample webhook configuration data
-- This migration adds sample webhook configurations for testing and development
INSERT INTO webhook_configs (
        name,
        event_type,
        webhook_url,
        is_active,
        timeout_ms
    )
VALUES (
        'Credit Postback',
        'CREDIT',
        'https://httpbin.org/get?source=webhook-processor&type=postback',
        true,
        30000
    ),
    (
        'Debit Chargeback',
        'DEBIT',
        'https://httpbin.org/get?source=webhook-processor&type=chargeback',
        true,
        30000
    ) ON CONFLICT DO NOTHING;