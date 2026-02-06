package driver

import (
	"context"
	"database/sql"
)

// Driver abstracts the database connection and query execution.
type Driver interface {
	// Name returns the driver name (e.g., "mysql", "postgres").
	Name() string

	// Ping verifies the connection to the database.
	Ping(ctx context.Context) error

	// Query executes a query and returns a RowStreamer to iterate over results.
	Query(ctx context.Context, query string) (RowStreamer, error)

	// Close closes the database connection.
	Close() error
}

// RowStreamer iterates over query results.
// It is designed to be memory-efficient and stream-oriented.
type RowStreamer interface {
	// Columns returns the column names. Safe to call after Query returns.
	Columns() ([]string, error)

	// ColumnTypes returns column information such as database type name.
	ColumnTypes() ([]*sql.ColumnType, error)

	// Next advances to the next row. Returns false when there are no more rows or an error occurs.
	Next() bool

	// Scan copies the columns in the current row into the values pointed at by dest.
	// The number of values must be the same as the number of columns.
	Scan(dest ...interface{}) error

	// Err returns the error, if any, that was encountered during iteration.
	Err() error

	// Close closes the streamer and frees resources.
	Close() error
}
