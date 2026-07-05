INSERT INTO feature_flags (key, description, enabled, created_at, updated_at)
VALUES ('use-transactions-v2', 'Use the v2 transactions service', FALSE, NOW(), NOW())
ON CONFLICT (key) DO NOTHING;
