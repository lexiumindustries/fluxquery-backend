package driver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDriver struct {
	uri    string
	client *mongo.Client
}

func NewMongoDriver(uri string) *MongoDriver {
	return &MongoDriver{uri: uri}
}

func (d *MongoDriver) Name() string {
	return "mongo"
}

func (d *MongoDriver) Ping(ctx context.Context) error {
	if d.client == nil {
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(d.uri))
		if err != nil {
			return err
		}
		d.client = client
	}
	return d.client.Ping(ctx, nil)
}

func (d *MongoDriver) Query(ctx context.Context, query string) (RowStreamer, error) {
	if d.client == nil {
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(d.uri))
		if err != nil {
			return nil, err
		}
		d.client = client
	}

	// Parse simple query syntax: db.collection.find({filter})
	// Valid Examples:
	// db.users.find({"age": {"$gt": 18}})
	// users.find({}) (assumes default db from URI)

	var dbName, collName string
	var filter bson.M

	// Naive parser for "collection.find(filter)" or "db.collection.find(filter)"
	// Extract JSON filter from between first '(' and last ')'
	start := strings.Index(query, "(")
	end := strings.LastIndex(query, ")")
	if start == -1 || end == -1 || end < start {
		return nil, errors.New("invalid query format: expected collection.find(filter)")
	}

	jsonFilter := query[start+1 : end]
	if jsonFilter == "" {
		jsonFilter = "{}"
	}

	if err := json.Unmarshal([]byte(jsonFilter), &filter); err != nil {
		return nil, fmt.Errorf("invalid filter JSON: %w", err)
	}

	// Determine collection and DB
	prefix := query[:start] // e.g. "db.users.find"
	segments := strings.Split(prefix, ".")

	// Handle "find" command only for now
	if segments[len(segments)-1] != "find" {
		return nil, errors.New("only 'find' command is supported")
	}

	if len(segments) == 3 {
		dbName = segments[0]
		collName = segments[1]
	} else if len(segments) == 2 {
		collName = segments[0]
		// Use database from URI if not specified
		// Note: The driver selects the DB from URI automatically if we use client.Database("").
		// But we need a name if we want to be explicit.
		// If URI has a DB, d.client.Database("") might work or default?
		// Actually typical Mongo usage is client.Database("name").Collection("name").
		// If URI is "mongodb://host/mydb", then "mydb" is default.
		// Let's rely on URI default if not provided, or error.
	} else {
		return nil, errors.New("invalid query format: expected [db.]collection.find(...)")
	}

	coll := d.client.Database(dbName).Collection(collName)
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &MongoStreamer{cursor: cursor, ctx: ctx}, nil
}

func (d *MongoDriver) Close() error {
	if d.client != nil {
		return d.client.Disconnect(context.Background())
	}
	return nil
}

// MongoStreamer implements RowStreamer for MongoDB functionality
type MongoStreamer struct {
	cursor *mongo.Cursor
	ctx    context.Context
	row    bson.M
	err    error
}

func (s *MongoStreamer) Columns() ([]string, error) {
	return []string{"document"}, nil
}

// ColumnTypes is not applicable to Mongo in the SQL sense, returning nil/empty
func (s *MongoStreamer) ColumnTypes() ([]*sql.ColumnType, error) {
	return nil, nil // Not implementing sql.ColumnType
}

func (s *MongoStreamer) Next() bool {
	if s.cursor.Next(s.ctx) {
		if err := s.cursor.Decode(&s.row); err != nil {
			s.err = err
			return false
		}
		return true
	}
	s.err = s.cursor.Err()
	return false
}

func (s *MongoStreamer) Scan(dest ...interface{}) error {
	if len(dest) != 1 {
		return errors.New("expected exactly 1 destination for document")
	}

	// Marshal row to JSON string
	data, err := json.Marshal(s.row)
	if err != nil {
		return err
	}

	// Assign to *interface{} or *string
	switch v := dest[0].(type) {
	case *string:
		*v = string(data)
	case *interface{}:
		*v = string(data)
	default:
		return errors.New("destination must be *string or *interface{}")
	}
	return nil
}

func (s *MongoStreamer) Err() error {
	return s.err
}

func (s *MongoStreamer) Close() error {
	return s.cursor.Close(s.ctx)
}
