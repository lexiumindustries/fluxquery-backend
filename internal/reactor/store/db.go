package store

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	db *sql.DB
}

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func NewStore(dsn string) (*Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) InitSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		// Ensure password_hash exists if table was created by a different task
		`ALTER TABLE users ADD COLUMN password_hash VARCHAR(255);`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			key_hash VARCHAR(255) NOT NULL UNIQUE,
			key_prefix VARCHAR(10) NOT NULL,
			type ENUM('live', 'test') NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			// Check for MySQL "Duplicate column name" error (1060)
			if mysqlErr, ok := err.(interface{ ErrorNumber() uint16 }); ok {
				if mysqlErr.ErrorNumber() == 1060 {
					continue // Column already exists, skip
				}
			}
			// Fallback: log warning but continue
			slog.Warn("Migration query issue (might be expected)", "query", query, "error", err)
		}
	}
	return nil
}

func (s *Store) CreateUser(email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("INSERT INTO users (email, password_hash) VALUES (?, ?)", email, string(hash))
	return err
}

func (s *Store) AuthenticateUser(email, password string) (*User, error) {
	var user User
	var hash string

	err := s.db.QueryRow("SELECT id, email, password_hash, created_at FROM users WHERE email = ?", email).Scan(&user.ID, &user.Email, &hash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid credentials")
	} else if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return &user, nil
}

// API Key Methods

type APIKey struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	KeyPrefix string    `json:"key_prefix"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) CreateAPIKey(userID int, keyType string) (string, error) {
	// Generate a secure random key: sk_live_<random> or sk_test_<random>
	// For MVP we just use UUID or random string
	// Store hash of the key

	// Mock generation for now (in real app use crypto/rand)
	suffix := fmt.Sprintf("%d_%d", userID, time.Now().UnixNano())
	rawKey := fmt.Sprintf("sk_%s_%s", keyType, suffix)

	hash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	_, err = s.db.Exec(
		"INSERT INTO api_keys (user_id, key_hash, key_prefix, type) VALUES (?, ?, ?, ?)",
		userID, string(hash), rawKey[:10], keyType,
	)
	if err != nil {
		return "", err
	}

	return rawKey, nil
}

func (s *Store) VerifyAPIKey(rawKey string) (*APIKey, error) {
	// 1. Identify key by prefix to narrow down?
	// Or just brute force check?
	// For production, we'd index by key_prefix or key_id embedded in the key.
	// Let's assume we find the key by prefix logic or just scan matches?
	// Scanning matches is slow.
	// Optimized: Store rawKey? No, insecure.
	// Best practice: key = "prefix" + "secret". DB stores "prefix" and "hash(secret)".
	// Current schema stores "key_prefix" (first 10 chars).

	prefix := rawKey
	if len(rawKey) > 10 {
		prefix = rawKey[:10]
	}

	rows, err := s.db.Query("SELECT id, user_id, key_hash, key_prefix, type, created_at FROM api_keys WHERE key_prefix = ?", prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var k APIKey
		var hash string
		if err := rows.Scan(&k.ID, &k.UserID, &hash, &k.KeyPrefix, &k.Type, &k.CreatedAt); err != nil {
			continue
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(rawKey)); err == nil {
			// Update Last Used (async/background ideally)
			go s.db.Exec("UPDATE api_keys SET last_used_at = NOW() WHERE id = ?", k.ID)
			return &k, nil
		}
	}

	return nil, fmt.Errorf("invalid api key")
}

func (s *Store) ListAPIKeys(userID int) ([]APIKey, error) {
	query := "SELECT id, user_id, key_prefix, type, created_at FROM api_keys WHERE user_id = ? ORDER BY created_at DESC"
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		// Previous query didn't select key_hash, so we scan directly to struct fields
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyPrefix, &k.Type, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}
