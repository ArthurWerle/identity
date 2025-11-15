INSERT INTO users (
    name,
    email
) VALUES (
    'Arthur Werle',
    'arthur.werle@gmail.com'
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO roles (
    description
) VALUES (
    'admin'
)
ON CONFLICT (description) DO NOTHING;

INSERT INTO roles (
    description
) VALUES (
    'viewer'
)
ON CONFLICT (description) DO NOTHING;

INSERT INTO roles (
    description
) VALUES (
    'editor'
)
ON CONFLICT (description) DO NOTHING;

INSERT INTO user_roles (
    user_id,
    role_id
) VALUES (
    (SELECT id from users where email = 'arthur.werle@gmail.com'),
    (SELECT id from roles where description = 'admin')
)
ON CONFLICT (user_id, role_id) DO NOTHING;

INSERT INTO feature_flags (
    key,
    description
) VALUES (
    'use-transactions-v2',
    'Use new transactions service'
)
ON CONFLICT (key) DO NOTHING;

INSERT INTO user_feature_flags (
    user_id,
    feature_flag_id
) VALUES (
    (SELECT id from users where email = 'arthur.werle@gmail.com'),
    (SELECT id from feature_flags where key = 'use-transactions-v2')
)
ON CONFLICT (user_id, feature_flag_id) DO NOTHING;