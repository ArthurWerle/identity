-- Seed some example data for development/testing

-- Insert example users
INSERT INTO users (name, email, enabled) VALUES
    ('Alice Johnson', 'alice@example.com', true),
    ('Bob Smith', 'bob@example.com', true),
    ('Charlie Brown', 'charlie@example.com', false)
ON CONFLICT (email) DO NOTHING;

-- Insert example feature flags
INSERT INTO feature_flags (key, description, enabled) VALUES
    ('dark_mode', 'Enable dark mode interface', true),
    ('beta_features', 'Access to beta features', false),
    ('premium_content', 'Access to premium content', true),
    ('analytics_tracking', 'Enable analytics tracking', true)
ON CONFLICT (key) DO NOTHING;

-- Assign some feature flags to users
INSERT INTO user_feature_flags (user_id, feature_flag_id)
SELECT u.id, f.id
FROM users u, feature_flags f
WHERE u.email = 'alice@example.com' AND f.key IN ('dark_mode', 'premium_content')
ON CONFLICT DO NOTHING;

INSERT INTO user_feature_flags (user_id, feature_flag_id)
SELECT u.id, f.id
FROM users u, feature_flags f
WHERE u.email = 'bob@example.com' AND f.key IN ('beta_features')
ON CONFLICT DO NOTHING;
