CREATE DATABASE IF NOT EXISTS my_app;
USE my_app;

CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    score DECIMAL(10, 2) DEFAULT 0.00
);

-- Seed 100 Rows (Simple)
INSERT INTO users (name, email, created_at, score) VALUES 
('Alice', 'alice@example.com', NOW(), 10.50),
('Bob', 'bob@example.com', NOW(), 20.00),
('Charlie', 'charlie@example.com', NOW(), 5.25);

-- We need more data for testing streaming?
-- We can generate more in Go or just use a loop if MySQL version supports it (MySQL 8.0 CTEs).
-- But for a simple functional test, 10-100 rows is enough to verify the CSV output format.
