-- Create balance_transactions table for idempotency
CREATE TABLE IF NOT EXISTS balance_transactions (
    id INT PRIMARY KEY AUTO_INCREMENT,
    order_id VARCHAR(100) UNIQUE NOT NULL,
    user_id INT NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    transaction_type VARCHAR(10) NOT NULL,
    crypto_symbol VARCHAR(10),
    previous_balance DECIMAL(15,2),
    new_balance DECIMAL(15,2),
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_order_id (order_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
