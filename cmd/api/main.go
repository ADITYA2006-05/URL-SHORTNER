package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/urlshortener/internal/config"
	"github.com/urlshortener/internal/handler"
	"github.com/urlshortener/internal/repository"
	"github.com/urlshortener/internal/service"
)

func main() {
	// 1. Load environment variables from .env if present
	loadEnv(".env")

	// 2. Load config
	cfg := config.Load()

	// 3. Connect to PostgreSQL
	log.Printf("Connecting to PostgreSQL at %s:%s...", cfg.DBHost, cfg.DBPort)
	pgRepo, err := repository.NewPostgresRepo(cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgRepo.Close()
	log.Println("PostgreSQL connection established.")

	// Run migrations
	migrationPath := filepath.Join("migrations", "001_init.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Printf("Warning: failed to read migration file %s: %v", migrationPath, err)
	} else {
		log.Println("Running migrations...")
		if err := pgRepo.RunMigrations(string(migrationSQL)); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations executed successfully.")
	}

	// 4. Connect to Redis
	log.Printf("Connecting to Redis at %s...", cfg.RedisAddr)
	redisRepo, err := repository.NewRedisRepo(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisRepo.Close()
	log.Println("Redis connection established.")

	// 5. Initialize Services
	svc := service.NewURLService(pgRepo, redisRepo, cfg.BaseURL)

	// Start background expiry cleanup every 5 minutes
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartExpiryCleanup(ctx, 5*time.Minute)

	// 6. Initialize Handlers and Middleware
	urlHandler := handler.NewURLHandler(svc)
	authHandler := handler.NewAuthHandler(pgRepo, cfg)
	redirectHandler := handler.NewRedirectHandler(svc)
	mw := handler.NewMiddleware(redisRepo, cfg.RateLimitRPM, cfg.JWTSecret)

	// 7. Setup Router
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(mw.Logger)
	r.Use(mw.CORS)

	// Static Files serving (avoid conflict with redirect router)
	// We explicitly serve static routes for CSS and JS
	r.Handle("/css/*", http.StripPrefix("/css/", http.FileServer(http.Dir("web/css"))))
	r.Handle("/js/*", http.StripPrefix("/js/", http.FileServer(http.Dir("web/js"))))

	// Root route serves index.html
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/index.html")
	})

	// Favicon
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	// API Routes (with Rate Limiting)
	r.Route("/api", func(r chi.Router) {
		r.Use(mw.RateLimit)

		// Public Auth Endpoints
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Get("/auth/google/login", authHandler.GoogleLogin)
		r.Get("/auth/google/callback", authHandler.GoogleCallback)
		r.Get("/health", urlHandler.HealthCheck)

		// Protected endpoints (Require Auth token)
		r.Group(func(r chi.Router) {
			r.Use(mw.Auth)

			r.Post("/shorten", urlHandler.ShortenURL)
			r.Get("/urls", urlHandler.ListURLs)
			r.Get("/urls/{code}/stats", urlHandler.GetURLStats)
			r.Delete("/urls/{code}", urlHandler.DeleteURL)
		})
	})

	// Redirect Route (captures anything else at the root that doesn't match above)
	r.Get("/{code}", redirectHandler.HandleRedirect)

	// 8. Start HTTP Server
	serverAddr := ":" + cfg.ServerPort
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Server shutdown handling
	go func() {
		log.Printf("Starting Server on port %s...", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server listen/serve error: %v", err)
		}
	}()

	// Wait for terminate signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped.")
}

// loadEnv reads .env and loads properties into environment variables
func loadEnv(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return // ignore if no .env file
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// Strip optional quotes
			val = strings.Trim(val, `"'`)
			os.Setenv(key, val)
		}
	}
}
