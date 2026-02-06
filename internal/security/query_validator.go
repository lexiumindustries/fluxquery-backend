package security

import (
	"errors"
	"strings"
)

var (
	ErrUnsafeQuery     = errors.New("unsafe query detected")
	ErrMultipleQueries = errors.New("multi-statement queries are not allowed")
	ErrNotSelect       = errors.New("only SELECT queries are allowed")
	ErrInvalidEmail    = errors.New("invalid email address format")
)

// ValidateEmail checks if the provided email is a valid format to prevent header injection.
func ValidateEmail(email string) error {
	// Simple but effective check for most cases.
	// Prevents \r and \n which are used for header injection.
	if strings.ContainsAny(email, "\r\n") {
		return ErrInvalidEmail
	}

	// Basic check for @ and .
	atIdx := strings.Index(email, "@")
	dotIdx := strings.LastIndex(email, ".")
	if atIdx < 1 || dotIdx < atIdx+2 || dotIdx == len(email)-1 {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateQuery adheres to the Principle of Least Privilege.
// It enforces strict rules to prevent SQL injection and unauthorized access:
//  1. Must be a SELECT statement.
//  2. Must not contain multiple statements (semicolons).
//  3. Must not contain destructive keywords (DELETE, DROP, UPDATE, etc.).
//  4. Must not access restricted system tables (information_schema, mysql, etc.).
func ValidateQuery(query string) error {
	q := strings.TrimSpace(query)
	qUpper := strings.ToUpper(q)

	// Rule 1: Must start with SELECT
	if !strings.HasPrefix(qUpper, "SELECT") {
		return ErrNotSelect
	}

	// Rule 2: No semicolons (prevent stacking)
	if strings.Contains(q, ";") {
		return ErrMultipleQueries
	}

	// Rule 3: Deny list of DML/DDL keywords and leakage vectors
	forbidden := []string{
		"DELETE", "DROP", "INSERT", "UPDATE", "ALTER", "TRUNCATE", "GRANT", "REVOKE",
		"CREATE", "REPLACE", "CALL", "DO", "HANDLER", "LOAD", "UNION",
		"USER(", "VERSION(", "DATABASE(", "LOAD_FILE(", "@@VERSION", "@@HOSTNAME",
	}

	for _, word := range forbidden {
		if containsWord(qUpper, word) {
			return errors.New("forbidden keyword detected: " + word)
		}
	}

	// Rule 4: Prevent access to system tables
	systemTables := []string{
		"INFORMATION_SCHEMA", "MYSQL", "PERFORMANCE_SCHEMA", "SYS",
	}
	for _, table := range systemTables {
		if containsWord(qUpper, table) {
			return errors.New("access to system table blocked: " + table)
		}
	}

	return nil
}

// containsWord checks if the word exists in s as a standalone word.
// It assumes s is already uppercase.
func containsWord(s, word string) bool {
	// Quick check if present at all
	if !strings.Contains(s, word) {
		return false
	}
	// Check strict boundaries if found
	// We want to match "DELETE" but not "IS_DELETED"
	// A word boundary in SQL is usually whitespace, or maybe `(`, `)`, `,`.
	// For simplicity and performance in this strict validator:
	// check if ` word ` exists, or starts with `word ` or ends with ` word`.
	// This covers most standard SQL injection attempts.

	// However, edge cases: "DELETE/**/FROM"
	// So simply banning "DELETE" might be too aggressive if a column is "deleted_at".
	// But the prompt asked for "Reject queries containing: ; DROP DELETE UPDATE INSERT".
	// It didn't specify "standalone words".
	// Given "Safe Query Execution" requirement, false positives are better than false negatives.
	// But "deleted_at" is super common.
	// Let's implement a smarter check: word boundary.

	// Helper to detecting word boundaries.
	// We'll iterate through the string and find the word.
	idx := 0
	for {
		i := strings.Index(s[idx:], word)
		if i == -1 {
			return false
		}
		start := idx + i
		end := start + len(word)

		// Check previous char
		isStartValid := start == 0 || isBoundary(s[start-1])
		// Check next char
		isEndValid := end == len(s) || isBoundary(s[end])

		if isStartValid && isEndValid {
			return true
		}

		idx = start + 1
	}
}

func isBoundary(b byte) bool {
	// Standard SQL delimiters
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' ||
		b == '(' || b == ')' || b == ',' || b == '=' ||
		b == '<' || b == '>' || b == '`' || b == '.' ||
		b == '"' || b == '[' || b == ']'
}
