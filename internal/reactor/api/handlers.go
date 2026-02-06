package api

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"mysql-exporter/internal/reactor/hub"
	"mysql-exporter/internal/reactor/store"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all for now
	},
}

type Handler struct {
	Store *store.Store
	Hub   *hub.Hub
}

func NewHandler(s *store.Store, h *hub.Hub) *Handler {
	return &Handler{
		Store: s,
		Hub:   h,
	}
}

// --- Auth Handlers ---

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.Store.CreateUser(req.Email, req.Password); err != nil {
		slog.Error("Register failed", "error", err)
		http.Error(w, "Email already exists or DB error", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User created"})
}

func (h *Handler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := h.Store.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(user)
}

// --- API Key Handlers ---

type CreateKeyRequest struct {
	UserID int    `json:"user_id"`
	Type   string `json:"type"` // "live" or "test"
}

func (h *Handler) HandleCreateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	key, err := h.Store.CreateAPIKey(req.UserID, req.Type)
	if err != nil {
		slog.Error("Create Key failed", "error", err)
		http.Error(w, "Failed to create key", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"key": key, "type": req.Type})
}

func (h *Handler) HandleListKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	// In a real app, we'd verify the user_id matches the authenticated session token
	// For MVP, we trust the query param or (better) would parse from JWT header if implemented.
	// Since we don't have JWT middleware *yet* in Reactor for these endpoints (dashboard handles auth),
	// this is a temporary simplification.

	// userID, _ := strconv.Atoi(userIDStr)
	// To avoid import issues, let's just assume simple int conversion or pass string if store changed
	// Store expects int.
	var userID int
	// simple atoi logic or use fmt.Sscan
	_, err := fmt.Sscan(userIDStr, &userID)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	keys, err := h.Store.ListAPIKeys(userID)
	if err != nil {
		slog.Error("List keys failed", "error", err)
		http.Error(w, "Failed to list keys", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(keys)
}

// --- Dashboard Handler ---

func (h *Handler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Dashboard upgrade failed", "error", err)
		return
	}

	h.Hub.Register(conn)

	// Keep connection open
	for {
		if _, _, err := conn.NextReader(); err != nil {
			h.Hub.Unregister(conn)
			break
		}
	}
}

// --- Agent Handlers ---

type JobCommand struct {
	ID    string `json:"id"`
	Query string `json:"query"`
}

func (h *Handler) HandleControl(w http.ResponseWriter, r *http.Request) {
	agentKeyRaw := r.Header.Get("X-Agent-Key")
	if agentKeyRaw == "" {
		http.Error(w, "Missing Agent Key", http.StatusUnauthorized)
		return
	}

	apiKey, err := h.Store.VerifyAPIKey(agentKeyRaw)
	if err != nil {
		slog.Warn("Invalid Agent Key", "key", agentKeyRaw, "error", err)
		http.Error(w, "Invalid Agent Key", http.StatusUnauthorized)
		return
	}

	slog.Info("Agent Connected (Control)", "key_id", apiKey.ID, "type", apiKey.Type)
	h.Hub.UpdateAgentCount(1)
	defer h.Hub.UpdateAgentCount(-1)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	// Detect Sandbox Mode
	isSandbox := (apiKey.Type == "test")

	// Simulate "Dispatch Job"
	go func() {
		delay := 5 * time.Second
		if isSandbox {
			delay = 2 * time.Second // Faster in sandbox
		}
		time.Sleep(delay)

		jobID := "job_" + apiKey.Type + "_" + time.Now().Format("150405")
		query := "SELECT * FROM users LIMIT 100"
		if isSandbox {
			query = "SELECT * FROM users_test_data LIMIT 10" // Sandbox specific query
		}

		job := JobCommand{
			ID:    jobID,
			Query: query,
		}

		payload, _ := json.Marshal(job)
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			slog.Error("Failed to send job", "error", err)
			return
		}
		slog.Info("Dispatched Job", "id", job.ID, "sandbox", isSandbox)

		h.Hub.Broadcast(hub.DashboardUpdate{
			Type:   "job_start",
			JobID:  job.ID,
			Status: "dispatched",
		})
	}()

	for {
		if _, _, err := conn.NextReader(); err != nil {
			slog.Info("Agent Disconnected (Control)")
			break
		}
	}
}

func (h *Handler) HandleData(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	slog.Info("Agent Connected (Data Stream)", "job_id", jobID)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	dec := gob.NewDecoder(&WSReader{Conn: conn})

	// 1. Read Columns
	var columns []string
	if err := dec.Decode(&columns); err != nil {
		slog.Error("Failed to decode columns", "error", err)
		return
	}
	slog.Info("Received Schema", "columns", columns)

	// 2. Read Rows
	rowCount := 0
	for {
		var values []interface{}
		if err := dec.Decode(&values); err != nil {
			if err.Error() == "EOF" {
				break
			}
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				break
			}
			slog.Info("Stream ended", "reason", err)
			break
		}
		rowCount++

		if rowCount%10 == 0 {
			h.Hub.Broadcast(hub.DashboardUpdate{
				Type:  "progress",
				JobID: jobID,
				Rows:  rowCount,
			})
		}
	}

	slog.Info("Data Stream Complete", "job_id", jobID, "total_rows", rowCount)
	h.Hub.Broadcast(hub.DashboardUpdate{
		Type:  "job_complete",
		JobID: jobID,
		Rows:  rowCount,
	})
}

// WSReader Helper (could be moved to util if shared)
type WSReader struct {
	Conn   *websocket.Conn
	reader io.Reader
}

func (r *WSReader) Read(p []byte) (n int, err error) {
	if r.reader == nil {
		_, reader, err := r.Conn.NextReader() // messageType ignored
		if err != nil {
			return 0, err
		}
		r.reader = reader
	}

	n, err = r.reader.Read(p)
	if err == io.EOF {
		r.reader = nil
		return r.Read(p) // Try next message
	}
	return n, err
}
