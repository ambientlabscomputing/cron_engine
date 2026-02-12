package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ambientlabscomputing/cron_engine/internal/syscall"
)

type Handler struct {
	syscallClient *syscall.Client
	logger        *slog.Logger
}

func NewHandler(syscallClient *syscall.Client, logger *slog.Logger) *Handler {
	return &Handler{
		syscallClient: syscallClient,
		logger:        logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/health":
		h.handleHealth(w, r)
	case "/crons":
		h.handleCrons(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

func (h *Handler) handleCrons(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreateCron(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleCreateCron(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID         string                 `json:"id"`
		Expression string                 `json:"expression"`
		Command    string                 `json:"command"`
		Timeout    string                 `json:"timeout"`
		Metadata   map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload := map[string]interface{}{
		"expression": req.Expression,
		"command":    req.Command,
		"timeout":    req.Timeout,
	}
	if len(req.Metadata) > 0 {
		payload["metadata"] = req.Metadata
	}

	if err := h.syscallClient.EmitCronEvent(
		context.Background(),
		"cron.scheduled",
		req.ID,
		payload,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id": req.ID,
	})
}
