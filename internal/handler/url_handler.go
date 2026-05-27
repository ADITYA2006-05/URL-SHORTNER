package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/urlshortener/internal/domain"
	"github.com/urlshortener/internal/service"
)

// URLHandler handles HTTP requests for URL operations
type URLHandler struct {
	service *service.URLService
}

// NewURLHandler creates a new URL handler
func NewURLHandler(svc *service.URLService) *URLHandler {
	return &URLHandler{service: svc}
}

// ShortenURL handles POST /api/shorten
func (h *URLHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req domain.ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "URL is required")
		return
	}

	// Add scheme if missing
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		req.URL = "https://" + req.URL
	}

	resp, err := h.service.ShortenURL(r.Context(), req, userID)
	if err != nil {
		if strings.Contains(err.Error(), "already taken") {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid") {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("Error shortening URL: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to shorten URL")
		return
	}

	respondJSON(w, http.StatusCreated, resp)
}

// ListURLs handles GET /api/urls
func (h *URLHandler) ListURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	urls, err := h.service.GetRecentURLs(r.Context(), 50, userID)
	if err != nil {
		log.Printf("Error listing URLs: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to list URLs")
		return
	}

	if urls == nil {
		urls = []domain.URL{}
	}

	respondJSON(w, http.StatusOK, urls)
}

// DeleteURL handles DELETE /api/urls/{code}
func (h *URLHandler) DeleteURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "Short code is required")
		return
	}

	if err := h.service.DeleteURL(r.Context(), code, userID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "URL not found")
			return
		}
		log.Printf("Error deleting URL: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to delete URL")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "URL deleted successfully"})
}

// GetURLStats handles GET /api/urls/{code}/stats
func (h *URLHandler) GetURLStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "Short code is required")
		return
	}

	stats, err := h.service.GetURLStats(r.Context(), code, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "expired") {
			respondError(w, http.StatusNotFound, "URL not found")
			return
		}
		log.Printf("Error getting stats: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// HealthCheck handles GET /api/health
func (h *URLHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "url-shortener",
	})
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error JSON response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
