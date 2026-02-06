package exporter

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// MySQLStreamer handles the execution of queries and streaming of results.
type MySQLStreamer struct {
	db *sql.DB
}

// NewMySQLStreamer creates a new streamer instance.
func NewMySQLStreamer(db *sql.DB) *MySQLStreamer {
	return &MySQLStreamer{db: db}
}

// ExportResult contains stats about the export
type ExportResult struct {
	RowsProcessed int64
	Duration      time.Duration
}

// StreamQuery executes the query and streams rows to the encoder.
// It ensures constant memory usage by using rows.Next() and scanning into reused buffers.
func (ms *MySQLStreamer) StreamQuery(ctx context.Context, query string, encoder RowEncoder) (*ExportResult, error) {
	start := time.Now()

	// Use QueryContext for cancellation and timeout support.
	// IMPORTANT: We use a read-only transaction or ensure session consistency if possible.
	// The prompt requests "Repeatable Read or Consistent Snapshot".
	// We can set this session variable or start a transaction.
	// For production safety with 10M+ rows, a single transaction is safest to ensure
	// data doesn't shift, but long running transactions have rollback segment costs.
	// "Consistent snapshot" is good for InnoDB.

	tx, err := ms.db.BeginTx(ctx, &sql.TxOptions{
		ReadOnly:  true,
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Safety cleanup

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Write Header
	if err := encoder.WriteHeader(columns); err != nil {
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	// Prepare scanners
	// To support ANY query, we need dynamic scanning.
	// sql.RawBytes is perfect here - it points into the driver's buffer.
	// It is valid only until next Next() call, which is exactly what we need for loose-coupling CSV writing.
	colCount := len(columns)
	values := make([]interface{}, colCount)
	scanArgs := make([]interface{}, colCount)
	for i := range values {
		scanArgs[i] = &values[i]
	}

	var rowCount int64

	for rows.Next() {
		// Stop if context cancelled
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}

		// Re-map sql.RawBytes to []byte/string if needed, or let CSV encoder handle it.
		// The driver might return []byte for strings/numbers.
		// Our CSV encoder handles []byte.
		// But values[i] might be nil.
		if err := encoder.WriteRow(values); err != nil {
			return nil, fmt.Errorf("csv write failed: %w", err)
		}

		rowCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	encoder.Flush()
	if err := encoder.Error(); err != nil {
		return nil, fmt.Errorf("csv flush error: %w", err)
	}

	// We don't need to Commit a read-only transaction, but it's good practice to close it cleanly.
	_ = tx.Commit()

	return &ExportResult{
		RowsProcessed: rowCount,
		Duration:      time.Since(start),
	}, nil
}
