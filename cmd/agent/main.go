package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"mysql-exporter/internal/driver"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

type AgentConfig struct {
	MySQLDSN   string
	ReactorURL string
	AgentKey   string
}

type JobCommand struct {
	ID    string `json:"id"`
	Query string `json:"query"`
}

func main() {
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})
	gob.Register([]byte{})
	gob.Register(time.Time{})

	_ = godotenv.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	config := AgentConfig{
		MySQLDSN:   os.Getenv("MYSQL_DSN"),
		ReactorURL: os.Getenv("REACTOR_URL"), // e.g., "ws://localhost:8080"
		AgentKey:   os.Getenv("AGENT_KEY"),
	}

	if config.MySQLDSN == "" || config.ReactorURL == "" {
		slog.Error("Missing configuration (MYSQL_DSN, REACTOR_URL)")
		os.Exit(1)
	}

	slog.Info("Starting FluxQuery Agent", "reactor", config.ReactorURL)

	// Initialize Driver
	dbDriver := driver.NewMySQLDriver(config.MySQLDSN)
	if err := dbDriver.Ping(context.Background()); err != nil {
		slog.Error("Failed to connect to Local DB", "error", err)
		os.Exit(1)
	}
	defer dbDriver.Close()
	slog.Info("Connected to Local DB (MySQL)")

	// Connect to Control Plane
	controlURL := config.ReactorURL + "/agent/control"
	headers := make(map[string][]string)
	headers["X-Agent-Key"] = []string{config.AgentKey}

	conn, _, err := websocket.DefaultDialer.Dial(controlURL, headers)
	if err != nil {
		slog.Error("Failed to connect to Reactor Control Plane", "error", err)
		os.Exit(1) // In prod, rely on restart policy or retry loop
	}
	defer conn.Close()
	slog.Info("Connected to Reactor Control Plane")

	// Main Loop
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				slog.Error("Read error", "error", err)
				return // Reconnect logic would go here
			}

			var job JobCommand
			if err := json.Unmarshal(message, &job); err != nil {
				slog.Error("Invalid command", "error", err)
				continue
			}

			slog.Info("Received Job", "id", job.ID, "query", job.Query)
			go executeJob(dbDriver, config.ReactorURL, config.AgentKey, job)
		}
	}()

	<-interrupt
	slog.Info("Agent shutting down...")
}

func executeJob(d driver.Driver, reactorURL, agentKey string, job JobCommand) {
	slog.Info("Executing Job", "id", job.ID)

	// 1. Run Query
	streamer, err := d.Query(context.Background(), job.Query)
	if err != nil {
		slog.Error("Query execution failed", "id", job.ID, "error", err)
		return
	}
	defer streamer.Close()

	// 2. Connect to Data Stream
	dataURL := reactorURL + "/agent/data?job_id=" + job.ID
	headers := make(map[string][]string)
	headers["X-Agent-Key"] = []string{agentKey}

	conn, _, err := websocket.DefaultDialer.Dial(dataURL, headers)
	if err != nil {
		slog.Error("Failed to connect to Data Stream", "id", job.ID, "error", err)
		return
	}
	defer conn.Close()

	// 3. Stream Data (Gob encoded)
	wsWriter := &WSWriter{Conn: conn}
	enc := gob.NewEncoder(wsWriter)

	// Send Headers
	columns, _ := streamer.Columns()
	if err := enc.Encode(columns); err != nil {
		slog.Error("Failed to encode columns", "id", job.ID, "error", err)
		return
	}

	// Send Rows
	values := make([]interface{}, len(columns))
	pointers := make([]interface{}, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}

	rowCount := 0
	for streamer.Next() {
		if err := streamer.Scan(pointers...); err != nil {
			slog.Error("Scan failed", "id", job.ID, "error", err)
			break
		}

		if err := enc.Encode(values); err != nil {
			slog.Error("Encode failed", "id", job.ID, "error", err)
			break
		}
		rowCount++
	}

	slog.Info("Job Completed", "id", job.ID, "rows", rowCount)
}

type WSWriter struct {
	Conn *websocket.Conn
}

func (w *WSWriter) Write(p []byte) (n int, err error) {
	err = w.Conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
