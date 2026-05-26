package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/urlshortener/internal/domain"
	"github.com/urlshortener/internal/service"
)

// RedirectHandler handles URL redirects
type RedirectHandler struct {
	service *service.URLService
}

// NewRedirectHandler creates a new redirect handler
func NewRedirectHandler(svc *service.URLService) *RedirectHandler {
	return &RedirectHandler{service: svc}
}

// HandleRedirect handles GET /{code} - redirects to original URL
func (h *RedirectHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		http.NotFound(w, r)
		return
	}

	// Resolve the URL
	urlRecord, err := h.service.ResolveURL(r.Context(), code)
	if err != nil {
		log.Printf("URL not found for code %s: %v", code, err)
		http.NotFound(w, r)
		return
	}

	// Record click asynchronously (never blocks redirect)
	browser, os, deviceType := service.ParseUserAgent(r.UserAgent())
	click := &domain.Click{
		URLID:      urlRecord.ID,
		IPAddress:  extractIP(r),
		UserAgent:  r.UserAgent(),
		Referer:    r.Referer(),
		Browser:    browser,
		OS:         os,
		DeviceType: deviceType,
	}
	h.service.RecordClick(click)

	// 301 permanent redirect
	http.Redirect(w, r, urlRecord.OriginalURL, http.StatusMovedPermanently)
}

// extractIP extracts the client IP from the request
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
