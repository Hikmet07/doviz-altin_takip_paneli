package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/altinkaynak-bot/backend/internal/db"
	"github.com/altinkaynak-bot/backend/internal/websocket"
	"github.com/gorilla/mux"
)

// Handler - HTTP ve WebSocket isteklerini yöneten yapı
type Handler struct {
	repo *db.Repository
	hub  *websocket.Hub
}

// NewHandler - Yeni bir Handler örneği oluşturur
func NewHandler(repo *db.Repository, hub *websocket.Hub) *Handler {
	return &Handler{
		repo: repo,
		hub:  hub,
	}
}

// RegisterRoutes - Tüm API rotalarını router'a kaydeder
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// API endpoint'leri
	r.HandleFunc("/api/health", h.handleHealthCheck).Methods(http.MethodGet)
	r.HandleFunc("/api/prices/latest", h.handleGetLatestAll).Methods(http.MethodGet)
	r.HandleFunc("/api/prices/latest/currency", h.handleGetLatestCurrency).Methods(http.MethodGet)
	r.HandleFunc("/api/prices/latest/gold", h.handleGetLatestGold).Methods(http.MethodGet)
	r.HandleFunc("/api/prices/history", h.handleGetHistoryAll).Methods(http.MethodGet)
	r.HandleFunc("/api/prices/history/{kod}", h.handleGetHistory).Methods(http.MethodGet)

	// WebSocket endpoint'i
	r.HandleFunc("/ws", h.handleWebSocket)
}

// handleHealthCheck - GET /api/health - Sunucu sağlık kontrolü
func (h *Handler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleGetLatestAll - GET /api/prices/latest - Tüm güncel fiyatları getirir
func (h *Handler) handleGetLatestAll(w http.ResponseWriter, r *http.Request) {
	records, err := h.repo.GetLatestAll(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Veriler alınamadı")
		return
	}
	respondJSON(w, http.StatusOK, records)
}

// handleGetLatestCurrency - GET /api/prices/latest/currency - Güncel döviz kurlarını getirir
func (h *Handler) handleGetLatestCurrency(w http.ResponseWriter, r *http.Request) {
	records, err := h.repo.GetLatestByCategory(r.Context(), "currency")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Döviz verileri alınamadı")
		return
	}
	respondJSON(w, http.StatusOK, records)
}

// handleGetLatestGold - GET /api/prices/latest/gold - Güncel altın fiyatlarını getirir
func (h *Handler) handleGetLatestGold(w http.ResponseWriter, r *http.Request) {
	records, err := h.repo.GetLatestByCategory(r.Context(), "gold")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Altın verileri alınamadı")
		return
	}
	respondJSON(w, http.StatusOK, records)
}

// handleGetHistory - GET /api/prices/history/{kod} - Belirli enstrümanın geçmişini getirir
func (h *Handler) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	// Yol değişkenlerinden kod'u al
	vars := mux.Vars(r)
	kod := vars["kod"]

	// Query parametrelerinden gün sayısını al (varsayılan: 30)
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	records, err := h.repo.GetHistory(r.Context(), kod, days)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Geçmiş veriler alınamadı")
		return
	}
	respondJSON(w, http.StatusOK, records)
}

// handleGetHistoryAll - GET /api/prices/history - Tüm enstrümanların geçmişini getirir
func (h *Handler) handleGetHistoryAll(w http.ResponseWriter, r *http.Request) {
	// Query parametrelerinden gün sayısını al (varsayılan: 30)
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	records, err := h.repo.GetHistoryAll(r.Context(), days)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Tüm geçmiş veriler alınamadı")
		return
	}
	respondJSON(w, http.StatusOK, records)
}

// handleWebSocket - GET /ws - WebSocket bağlantı handler'ı
func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// HTTP isteğini WebSocket'e dönüştür
	h.hub.HandleWebSocket(w, r)
}

// Helper: respondJSON - Veriyi JSON formatında HTTP yanıtı olarak gönderir
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("JSON yanıt oluşturulamadı", "error", err)
	}
}

// Helper: respondError - Hata mesajını JSON formatında gönderir
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
