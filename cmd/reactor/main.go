package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"mysql-exporter/internal/config"
	"mysql-exporter/internal/reactor/api"
	"mysql-exporter/internal/reactor/hub"
	middleware "mysql-exporter/internal/reactor/middleware"
	"mysql-exporter/internal/reactor/store"
)

func main() {
	_ = godotenv.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 0. Load Config
	cfg := config.Load()

	slog.Info("Starting FluxQuery Reactor", "env", cfg.AppEnv)

	// 1. Initialize Store (Database)
	if cfg.MySQLDSN == "" {
		slog.Error("MYSQL_DSN not set")
		os.Exit(1)
	}

	st, err := store.NewStore(cfg.MySQLDSN)
	if err != nil {
		slog.Error("Failed to initialize store", "error", err)
		os.Exit(1)
	}

	// 2. Run Migration
	if err := st.InitSchema(); err != nil {
		slog.Error("Migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Database Connected & Schema Initialized")

	// 3. Initialize Hub (WebSocket Manager)
	h := hub.NewHub()

	// 4. Initialize Handlers
	handler := api.NewHandler(st, h)

	// 5. Setup Routes & Middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/agent/control", handler.HandleControl)
	mux.HandleFunc("/agent/data", handler.HandleData)
	mux.HandleFunc("/dashboard/stream", handler.HandleDashboard)
	mux.HandleFunc("/auth/register", handler.HandleRegister)
	mux.HandleFunc("/auth/verify", handler.HandleVerify)
	mux.HandleFunc("/auth/keys/create", handler.HandleCreateKey)
	mux.HandleFunc("/auth/keys/list", handler.HandleListKeys) // New Endpoint

	// Wrap with Middleware
	finalHandler := middleware.CORS(cfg.AllowedOrigins, cfg.AppEnv)(mux)

	slog.Info("Reactor listening", "port", cfg.ServerPort)
	if err := http.ListenAndServe(":"+cfg.ServerPort, finalHandler); err != nil {
		slog.Error("Server failed", "error", err)
	}
}
