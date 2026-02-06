package hub

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
)

type DashboardUpdate struct {
	Type       string `json:"type"` // "job_start", "progress", "job_complete", "agent_update"
	JobID      string `json:"job_id,omitempty"`
	Rows       int    `json:"rows,omitempty"`
	Status     string `json:"status,omitempty"`
	AgentCount int    `json:"agent_count,omitempty"`
}

type Hub struct {
	dashboards map[*websocket.Conn]bool
	agentCount int
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		dashboards: make(map[*websocket.Conn]bool),
	}
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.dashboards[conn] = true
	slog.Info("Dashboard Connected", "total_connections", len(h.dashboards))
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.dashboards[conn]; ok {
		delete(h.dashboards, conn)
		conn.Close()
		slog.Info("Dashboard Disconnected", "total_connections", len(h.dashboards))
	}
}

func (h *Hub) Broadcast(update DashboardUpdate) {
	h.mu.Lock()
	defer h.mu.Unlock()

	payload, _ := json.Marshal(update)
	for conn := range h.dashboards {
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			slog.Error("Create broadcast failed", "error", err)
			conn.Close()
			delete(h.dashboards, conn)
		}
	}
}

func (h *Hub) UpdateAgentCount(delta int) {
	h.mu.Lock()
	h.agentCount += delta
	count := h.agentCount
	h.mu.Unlock()

	h.Broadcast(DashboardUpdate{
		Type:       "agent_update",
		AgentCount: count,
	})
}
