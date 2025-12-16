-- Remove sample webhook configuration data
-- This migration removes the sample webhook configurations
DELETE FROM webhook_configs
WHERE name IN (
        'Credit Postback',
        'Debit Chargeback'
    )
    AND webhook_url LIKE 'https://httpbin.org%';