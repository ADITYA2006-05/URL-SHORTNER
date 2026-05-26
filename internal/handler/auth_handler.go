package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/urlshortener/internal/config"
	"github.com/urlshortener/internal/domain"
	"github.com/urlshortener/internal/repository"
	"github.com/urlshortener/internal/service"
)

// AuthHandler handles user registration and login requests
type AuthHandler struct {
	pgRepo    *repository.PostgresRepo
	jwtSecret string
}

// NewAuthHandler creates a new auth handler instance
func NewAuthHandler(pgRepo *repository.PostgresRepo, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		pgRepo:    pgRepo,
		jwtSecret: cfg.JWTSecret,
	}
}

// Register handles user registration POST /api/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password

	if len(username) < 3 || len(username) > 30 {
		respondError(w, http.StatusBadRequest, "Username must be between 3 and 30 characters")
		return
	}
	if len(password) < 6 {
		respondError(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	// Hash password
	hashedPassword, err := service.HashPassword(password)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to register user")
		return
	}

	user := &domain.User{
		Username:     username,
		PasswordHash: hashedPassword,
	}

	// Store user in Postgres
	err = h.pgRepo.CreateUser(r.Context(), user)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, "Username is already taken")
			return
		}
		log.Printf("Failed to create user in database: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

	// Generate JWT token
	token, err := service.GenerateToken(user.ID, user.Username, h.jwtSecret)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		respondError(w, http.StatusInternalServerError, "Account created, but failed to log in")
		return
	}

	respondJSON(w, http.StatusCreated, domain.AuthResponse{
		Token:    token,
		Username: user.Username,
	})
}

// Login handles user login POST /api/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password

	if username == "" || password == "" {
		respondError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Retrieve user from database
	user, err := h.pgRepo.GetUserByUsername(r.Context(), username)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	// Verify password hash
	if !service.CheckPasswordHash(password, user.PasswordHash) {
		respondError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	// Generate JWT token
	token, err := service.GenerateToken(user.ID, user.Username, h.jwtSecret)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to compile token")
		return
	}

	respondJSON(w, http.StatusOK, domain.AuthResponse{
		Token:    token,
		Username: user.Username,
	})
}

