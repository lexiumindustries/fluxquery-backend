package driver

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
)

type PostgresDriver struct {
	dsn string
	db  *sql.DB
}

func NewPostgresDriver(dsn string) *PostgresDriver {
	return &PostgresDriver{dsn: dsn}
}

func (d *PostgresDriver) Name() string {
	return "postgres"
}

func (d *PostgresDriver) Ping(ctx context.Context) error {
	if d.db == nil {
		var err error
		d.db, err = sql.Open("postgres", d.dsn)
		if err != nil {
			return err
		}
	}
	return d.db.PingContext(ctx)
}

func (d *PostgresDriver) Query(ctx context.Context, query string) (RowStreamer, error) {
	if d.db == nil {
		// Lazy connect
		var err error
		d.db, err = sql.Open("postgres", d.dsn)
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

func (d *PostgresDriver) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
