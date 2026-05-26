package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/urlshortener/internal/repository"
	"github.com/urlshortener/internal/service"
)

// Middleware holds shared middleware dependencies
type Middleware struct {
	redisRepo    *repository.RedisRepo
	rateLimitRPM int
	jwtSecret    string
}

// NewMiddleware creates a new middleware instance
func NewMiddleware(redisRepo *repository.RedisRepo, rateLimitRPM int, jwtSecret string) *Middleware {
	return &Middleware{
		redisRepo:    redisRepo,
		rateLimitRPM: rateLimitRPM,
		jwtSecret:    jwtSecret,
	}
}

// CORS middleware handles Cross-Origin Resource Sharing
func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimit middleware limits API requests per IP
func (m *Middleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		window := time.Minute

		allowed, err := m.redisRepo.RateLimitCheck(r.Context(), ip, m.rateLimitRPM, window)
		if err != nil {
			log.Printf("Rate limit check error: %v", err)
			// Allow on error
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit headers
		remaining, _ := m.redisRepo.GetRateLimitRemaining(r.Context(), ip, m.rateLimitRPM)
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(m.rateLimitRPM))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			w.Header().Set("Retry-After", "60")
			respondError(w, http.StatusTooManyRequests, "Rate limit exceeded. Try again in 60 seconds.")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Logger middleware logs HTTP requests
func (m *Middleware) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %s",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			time.Since(start).Round(time.Millisecond),
		)
	})
}

// statusWriter wraps http.ResponseWriter to capture the status code
type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.statusCode = code
	sw.ResponseWriter.WriteHeader(code)
}

// Auth middleware validates JWT tokens and sets the userID context
func (m *Middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondError(w, http.StatusUnauthorized, "Authorization token required")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			respondError(w, http.StatusUnauthorized, "Invalid authorization format. Must be 'Bearer <token>'")
			return
		}

		tokenStr := parts[1]
		userID, _, err := service.ValidateToken(tokenStr, m.jwtSecret)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Invalid or expired authorization token")
			return
		}

		// Set user ID in request context
		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
