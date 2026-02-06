package driver

import (
	"context"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLDriver struct {
	dsn string
	db  *sql.DB
}

func NewMySQLDriver(dsn string) *MySQLDriver {
	return &MySQLDriver{dsn: dsn}
}

func (d *MySQLDriver) Name() string {
	return "mysql"
}

func (d *MySQLDriver) Ping(ctx context.Context) error {
	if d.db == nil {
		var err error
		d.db, err = sql.Open("mysql", d.dsn)
		if err != nil {
			return err
		}
	}
	return d.db.PingContext(ctx)
}

func (d *MySQLDriver) Query(ctx context.Context, query string) (RowStreamer, error) {
	if d.db == nil {
		// Lazy connect
		var err error
		d.db, err = sql.Open("mysql", d.dsn)
		if err != nil {
			return nil, err
		}
	}

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (d *MySQLDriver) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Enforce interface compliance using sql.Rows directly?
// sql.Rows implements Scan, Next, Close, Err, Columns.
// So we can return *sql.Rows directly as RowStreamer if the signatures match exactly.
// Columns() ([]string, error) - Match
// ColumnTypes() ([]*sql.ColumnType, error) - Match
// Next() bool - Match
// Scan(dest ...interface{}) error - Match
// Err() error - Match
// Close() error - Match
