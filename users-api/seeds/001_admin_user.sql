-- Seed file for creating admin user
-- Password: admin1234
-- Email: admin@cryptosim.com
-- Username: admin

-- Check if admin user already exists
SET @admin_exists = (SELECT COUNT(*) FROM users WHERE email = 'admin@cryptosim.com');

-- Insert admin user only if it doesn't exist
INSERT INTO users (
    username,
    email,
    password_hash,
    role,
    initial_balance,
    is_active,
    created_at,
    updated_at,
    preferences
)
SELECT
    'admin',
    'admin@cryptosim.com',
    -- Bcrypt hash for 'admin1234' (cost=12)
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIAl.Mu1rG',
    'admin',
    100000.00,
    true,
    NOW(),
    NOW(),
    '{"theme":"light","notifications":true,"language":"en"}'
FROM DUAL
WHERE @admin_exists = 0;

-- Verify admin user was created
SELECT id, username, email, role, is_active, created_at
FROM users
WHERE email = 'admin@cryptosim.com';
