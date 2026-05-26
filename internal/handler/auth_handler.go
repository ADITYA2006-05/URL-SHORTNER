package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/urlshortener/internal/config"
	"github.com/urlshortener/internal/domain"
	"github.com/urlshortener/internal/repository"
	"github.com/urlshortener/internal/service"
)

// AuthHandler handles user registration and login requests
type AuthHandler struct {
	pgRepo             *repository.PostgresRepo
	jwtSecret          string
	googleClientID     string
	googleClientSecret string
	googleRedirectURL  string
	baseURL            string
}

// NewAuthHandler creates a new auth handler instance
func NewAuthHandler(pgRepo *repository.PostgresRepo, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		pgRepo:             pgRepo,
		jwtSecret:          cfg.JWTSecret,
		googleClientID:     cfg.GoogleClientID,
		googleClientSecret: cfg.GoogleClientSecret,
		googleRedirectURL:  cfg.GoogleRedirectURL,
		baseURL:            cfg.BaseURL,
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

// GoogleLogin redirects the user to Google's OAuth consent screen
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if h.googleClientID == "" || h.googleClientSecret == "" {
		respondError(w, http.StatusNotImplemented, "Google Auth is not configured on this server")
		return
	}

	// Generate secure random state string
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to initialize secure login session")
		return
	}
	state := hex.EncodeToString(b)

	// Set state in HttpOnly secure cookie (5 minutes expiration)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode,
	})

	// Construct Google Auth URL
	googleURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid+email+profile&state=%s&access_type=offline&prompt=select_account",
		url.QueryEscape(h.googleClientID),
		url.QueryEscape(h.googleRedirectURL),
		url.QueryEscape(state),
	)

	http.Redirect(w, r, googleURL, http.StatusTemporaryRedirect)
}

// googleTokenResponse represents Google token exchange response
type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IdToken     string `json:"id_token"`
}

// googleProfileResponse represents Google userinfo profile response
type googleProfileResponse struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// GoogleCallback handles Google OAuth callback and processes login/registration
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if h.googleClientID == "" || h.googleClientSecret == "" {
		respondError(w, http.StatusNotImplemented, "Google Auth is not configured on this server")
		return
	}

	// Verify state parameter
	cookie, err := r.Cookie("oauth_state")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Session expired or state cookie is missing")
		return
	}

	stateParam := r.URL.Query().Get("state")
	if stateParam == "" || stateParam != cookie.Value {
		respondError(w, http.StatusBadRequest, "State parameter mismatch (CSRF threat prevented)")
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	codeParam := r.URL.Query().Get("code")
	if codeParam == "" {
		respondError(w, http.StatusBadRequest, "Authorization code is missing from callback")
		return
	}

	// Exchange authorization code for token
	tokenVals := url.Values{}
	tokenVals.Set("code", codeParam)
	tokenVals.Set("client_id", h.googleClientID)
	tokenVals.Set("client_secret", h.googleClientSecret)
	tokenVals.Set("redirect_uri", h.googleRedirectURL)
	tokenVals.Set("grant_type", "authorization_code")

	tokenResp, err := http.PostForm("https://oauth2.googleapis.com/token", tokenVals)
	if err != nil {
		log.Printf("Google Token API Error: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to authenticate with Google Identity Server")
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		var errData map[string]interface{}
		json.NewDecoder(tokenResp.Body).Decode(&errData)
		log.Printf("Google Token Exchange Failed (Status %d): %v", tokenResp.StatusCode, errData)
		respondError(w, http.StatusUnauthorized, "Failed to exchange auth token with Google")
		return
	}

	var tokenInfo googleTokenResponse
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenInfo); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to parse Google auth response")
		return
	}

	// Fetch user profile info
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(r.Context(), "GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create userinfo request")
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenInfo.AccessToken))

	profileResp, err := client.Do(req)
	if err != nil {
		log.Printf("Google UserInfo API Error: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch profile details from Google")
		return
	}
	defer profileResp.Body.Close()

	if profileResp.StatusCode != http.StatusOK {
		log.Printf("Google UserInfo API Failed (Status %d)", profileResp.StatusCode)
		respondError(w, http.StatusUnauthorized, "Google rejected profile info retrieval")
		return
	}

	var profile googleProfileResponse
	if err := json.NewDecoder(profileResp.Body).Decode(&profile); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decode user profile from Google")
		return
	}

	if profile.Sub == "" || profile.Email == "" {
		respondError(w, http.StatusBadRequest, "Invalid profile data received from Google")
		return
	}

	var user *domain.User
	user, err = h.pgRepo.GetUserByGoogleID(r.Context(), profile.Sub)
	if err != nil {
		// Not found by Google ID, let's search by email
		user, err = h.pgRepo.GetUserByEmail(r.Context(), profile.Email)
		if err == nil {
			// User exists by email, link Google ID
			if err := h.pgRepo.LinkGoogleAccount(r.Context(), user.ID, profile.Sub); err != nil {
				log.Printf("Failed to link Google account for user ID %d: %v", user.ID, err)
				respondError(w, http.StatusInternalServerError, "Failed to link Google identity with existing account")
				return
			}
			user.GoogleID = profile.Sub
		} else {
			// User does not exist, create one
			// Generate username from email prefix (e.g. aditya.shah from aditya.shah@gmail.com)
			emailParts := strings.Split(profile.Email, "@")
			usernameBase := emailParts[0]
			usernameBase = strings.ReplaceAll(usernameBase, ".", "_") // replace dots with underscores to keep it clean

			// Validate length bounds (3 to 30)
			if len(usernameBase) < 3 {
				usernameBase = "google_user"
			}
			if len(usernameBase) > 25 {
				usernameBase = usernameBase[:25]
			}

			username := usernameBase
			// Ensure uniqueness
			for i := 1; i <= 20; i++ {
				existingUser, err := h.pgRepo.GetUserByUsername(r.Context(), username)
				if err != nil {
					// Username is unique! Let's proceed
					break
				}
				if existingUser != nil {
					// Username taken, append random-ish suffix
					suffix := fmt.Sprintf("_%d", time.Now().UnixNano()%1000)
					username = usernameBase + suffix
					if len(username) > 30 {
						username = username[:30]
					}
				}
			}

			user = &domain.User{
				Username: username,
				Email:    profile.Email,
				GoogleID: profile.Sub,
			}

			if err := h.pgRepo.CreateOAuthUser(r.Context(), user); err != nil {
				log.Printf("Failed to create OAuth user in database: %v", err)
				respondError(w, http.StatusInternalServerError, "Failed to register Google user account")
				return
			}
		}
	}

	// Generate JWT session token
	token, err := service.GenerateToken(user.ID, user.Username, h.jwtSecret)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to compile session token")
		return
	}

	// Redirect to frontend dashboard with token & username in URL params
	frontendRedirectURL := fmt.Sprintf(
		"%s/?token=%s&username=%s",
		h.baseURL,
		url.QueryEscape(token),
		url.QueryEscape(user.Username),
	)

	http.Redirect(w, r, frontendRedirectURL, http.StatusTemporaryRedirect)
}
