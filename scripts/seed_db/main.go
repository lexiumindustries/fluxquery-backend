package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "root:root@tcp(localhost:3306)/my_app?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Wait for DB to be ready
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		slog.Info("Waiting for database...", "attempt", i+1)
		time.Sleep(1 * time.Second)
	}

	slog.Info("Connected to MySQL. Creating tables...")

	// 1. Create Users Table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name TEXT,
			email TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			score DOUBLE
		)
	`)
	if err != nil {
		panic(err)
	}

	// 2. Create Transactions Table (Complex Relation)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS transactions (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT,
			amount DECIMAL(15, 2),
			currency VARCHAR(3),
			status VARCHAR(20),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_user_id (user_id)
		)
	`)
	if err != nil {
		panic(err)
	}

	// 3. Seed Users (1,000,000)
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if userCount < 1000000 {
		slog.Info("Seeding 1,000,000 users...")
		start := time.Now()
		batchSize := 1000
		total := 1000000

		for i := 0; i < total; i += batchSize {
			vals := []interface{}{}
			stmt := "INSERT INTO users (name, email, created_at, score) VALUES "
			placeholders := []string{}

			for j := 0; j < batchSize; j++ {
				idx := i + j + 1
				placeholders = append(placeholders, "(?, ?, ?, ?)")
				vals = append(vals,
					fmt.Sprintf("User%d", idx),
					fmt.Sprintf("user%d@example.com", idx),
					time.Now(),
					float64(idx)*0.1,
				)
			}

			stmt += strings.Join(placeholders, ",")
			_, err := db.Exec(stmt, vals...)
			if err != nil {
				panic(err)
			}

			if (i+batchSize)%100000 == 0 {
				fmt.Printf("\rSeeding Users: %d/%d", i+batchSize, total)
			}
		}
		fmt.Println()
		slog.Info("User seeding complete", "duration", time.Since(start))
	} else {
		slog.Info("Users already seeded", "count", userCount)
	}

	// 4. Seed Transactions (5,000,000)
	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&txCount)
	if txCount < 5000000 {
		slog.Info("Seeding 5,000,000 transactions...")
		start := time.Now()
		batchSize := 2000
		total := 5000000

		for i := 0; i < total; i += batchSize {
			vals := []interface{}{}
			stmt := "INSERT INTO transactions (user_id, amount, currency, status, created_at) VALUES "
			placeholders := []string{}

			for j := 0; j < batchSize; j++ {
				uid := (i+j)%1000000 + 1 // Cycle through users
				placeholders = append(placeholders, "(?, ?, ?, ?, ?)")
				vals = append(vals,
					uid,
					float64(uid)*0.25,
					"USD",
					"COMPLETED",
					time.Now(),
				)
			}

			stmt += strings.Join(placeholders, ",")
			_, err := db.Exec(stmt, vals...)
			if err != nil {
				panic(err)
			}

			if (i+batchSize)%100000 == 0 {
				fmt.Printf("\rSeeding Transactions: %d/%d", i+batchSize, total)
			}
		}
		fmt.Println()
		slog.Info("Transaction seeding complete", "duration", time.Since(start))
	} else {
		slog.Info("Transactions already seeded", "count", txCount)
	}

	slog.Info("Database schema and data prep complete.")
}
