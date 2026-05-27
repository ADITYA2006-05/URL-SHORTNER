package service

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/urlshortener/internal/domain"
	"github.com/urlshortener/internal/repository"
	"github.com/urlshortener/internal/shortcode"
)

// URLService handles URL shortening business logic
type URLService struct {
	pgRepo    *repository.PostgresRepo
	redisRepo *repository.RedisRepo
	baseURL   string
	clickChan chan *domain.Click
}

// NewURLService creates a new URL service
func NewURLService(pgRepo *repository.PostgresRepo, redisRepo *repository.RedisRepo, baseURL string) *URLService {
	svc := &URLService{
		pgRepo:    pgRepo,
		redisRepo: redisRepo,
		baseURL:   strings.TrimRight(baseURL, "/"),
		clickChan: make(chan *domain.Click, 1000), // Buffered channel for async click recording
	}

	// Start background click processor
	go svc.processClicks()

	return svc
}

// ShortenURL creates a new shortened URL
func (s *URLService) ShortenURL(ctx context.Context, req domain.ShortenRequest, userID int64) (*domain.ShortenResponse, error) {
	// Validate URL
	if err := validateURL(req.URL); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	urlRecord := &domain.URL{
		OriginalURL: req.URL,
		UserID:      userID,
	}

	// Handle custom alias
	if req.CustomAlias != "" {
		if !shortcode.IsValidAlias(req.CustomAlias) {
			return nil, fmt.Errorf("invalid custom alias: must be 3-20 alphanumeric characters")
		}

		// Check if alias already exists
		exists, err := s.pgRepo.ShortCodeExists(ctx, req.CustomAlias)
		if err != nil {
			return nil, fmt.Errorf("error checking alias: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("custom alias '%s' is already taken", req.CustomAlias)
		}

		urlRecord.ShortCode = req.CustomAlias
		urlRecord.CustomAlias = true
	}

	// Handle expiration
	if req.ExpiresIn != "" {
		duration, err := parseExpiration(req.ExpiresIn)
		if err != nil {
			return nil, fmt.Errorf("invalid expiration: %w", err)
		}
		if duration > 0 {
			expiresAt := time.Now().Add(duration)
			urlRecord.ExpiresAt = &expiresAt
		}
	}

	// Create URL in database (generates ID)
	if err := s.pgRepo.CreateURL(ctx, urlRecord); err != nil {
		return nil, fmt.Errorf("error creating URL: %w", err)
	}

	// Generate short code from ID if not custom
	if !urlRecord.CustomAlias {
		urlRecord.ShortCode = shortcode.Encode(urlRecord.ID)
		// Update the short code in the database
		// We create with a temp code then update - but simpler to just use the ID-based code
		// Actually, let's regenerate: delete and re-create would be complex
		// Better approach: create with the encoded short code directly
	}

	// For non-custom aliases, we need to update the short code
	if !urlRecord.CustomAlias {
		urlRecord.ShortCode = shortcode.Encode(urlRecord.ID)
		// Update in DB
		s.updateShortCode(ctx, urlRecord.ID, urlRecord.ShortCode)
	}

	// Cache in Redis
	if err := s.redisRepo.CacheURL(ctx, urlRecord.ShortCode, urlRecord.OriginalURL, urlRecord.ExpiresAt); err != nil {
		log.Printf("Warning: failed to cache URL in Redis: %v", err)
	}

	return &domain.ShortenResponse{
		ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, urlRecord.ShortCode),
		ShortCode:   urlRecord.ShortCode,
		OriginalURL: urlRecord.OriginalURL,
		ExpiresAt:   urlRecord.ExpiresAt,
		CreatedAt:   urlRecord.CreatedAt,
	}, nil
}

// updateShortCode updates the short code for a URL
func (s *URLService) updateShortCode(ctx context.Context, id int64, code string) {
	// Use the postgres repo's pool directly through a method
	s.pgRepo.UpdateShortCode(ctx, id, code)
}

// ResolveURL resolves a short code to the original URL (cache-aside pattern)
func (s *URLService) ResolveURL(ctx context.Context, shortCode string) (*domain.URL, error) {
	// Try Redis cache first
	cachedURL, err := s.redisRepo.GetCachedURL(ctx, shortCode)
	if err == nil && cachedURL != "" {
		// Cache hit - still need full URL record for analytics
		urlRecord, err := s.pgRepo.GetURLByShortCode(ctx, shortCode)
		if err != nil {
			return nil, fmt.Errorf("URL not found")
		}
		return urlRecord, nil
	}

	// Cache miss - query PostgreSQL
	urlRecord, err := s.pgRepo.GetURLByShortCode(ctx, shortCode)
	if err != nil {
		return nil, fmt.Errorf("URL not found or expired")
	}

	// Re-populate Redis cache
	if cacheErr := s.redisRepo.CacheURL(ctx, shortCode, urlRecord.OriginalURL, urlRecord.ExpiresAt); cacheErr != nil {
		log.Printf("Warning: failed to cache URL: %v", cacheErr)
	}

	return urlRecord, nil
}

// RecordClick asynchronously records a click event
func (s *URLService) RecordClick(click *domain.Click) {
	// Non-blocking send to channel
	select {
	case s.clickChan <- click:
	default:
		log.Printf("Warning: click channel full, dropping click event for URL ID %d", click.URLID)
	}
}

// processClicks processes click events from the channel
func (s *URLService) processClicks() {
	for click := range s.clickChan {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		// Record in PostgreSQL
		if err := s.pgRepo.RecordClick(ctx, click); err != nil {
			log.Printf("Error recording click: %v", err)
		}

		// Increment click count
		if err := s.pgRepo.IncrementClickCount(ctx, click.URLID); err != nil {
			log.Printf("Error incrementing click count: %v", err)
		}

		cancel()
	}
}

// GetRecentURLs retrieves recent URLs with short URLs populated for a user
func (s *URLService) GetRecentURLs(ctx context.Context, limit int, userID int64) ([]domain.URL, error) {
	urls, err := s.pgRepo.GetRecentURLs(ctx, limit, userID)
	if err != nil {
		return nil, err
	}

	// Populate short URLs
	for i := range urls {
		urls[i].ShortURL = fmt.Sprintf("%s/%s", s.baseURL, urls[i].ShortCode)
	}

	return urls, nil
}

// GetURLStats retrieves analytics for a URL owned by a specific user
func (s *URLService) GetURLStats(ctx context.Context, shortCode string, userID int64) (*domain.URLStats, error) {
	stats, err := s.pgRepo.GetURLStats(ctx, shortCode, userID)
	if err != nil {
		return nil, err
	}
	stats.URL.ShortURL = fmt.Sprintf("%s/%s", s.baseURL, stats.URL.ShortCode)
	return stats, nil
}

// DeleteURL deletes a URL by short code and checks ownership
func (s *URLService) DeleteURL(ctx context.Context, shortCode string, userID int64) error {
	if err := s.pgRepo.DeleteURL(ctx, shortCode, userID); err != nil {
		return err
	}

	// Remove from cache
	if err := s.redisRepo.DeleteCachedURL(ctx, shortCode); err != nil {
		log.Printf("Warning: failed to delete URL from cache: %v", err)
	}

	return nil
}

// CleanupExpiredURLs removes expired URLs
func (s *URLService) CleanupExpiredURLs(ctx context.Context) {
	count, err := s.pgRepo.CleanupExpiredURLs(ctx)
	if err != nil {
		log.Printf("Error cleaning up expired URLs: %v", err)
		return
	}
	if count > 0 {
		log.Printf("Cleaned up %d expired URLs", count)
	}
}

// StartExpiryCleanup starts a background goroutine to clean up expired URLs
func (s *URLService) StartExpiryCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.CleanupExpiredURLs(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// ParseUserAgent extracts browser, OS, and device from user agent string
func ParseUserAgent(ua string) (browser, os, deviceType string) {
	ua = strings.ToLower(ua)

	// Detect browser
	switch {
	case strings.Contains(ua, "firefox"):
		browser = "Firefox"
	case strings.Contains(ua, "edg"):
		browser = "Edge"
	case strings.Contains(ua, "chrome") && !strings.Contains(ua, "edg"):
		browser = "Chrome"
	case strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome"):
		browser = "Safari"
	case strings.Contains(ua, "opera") || strings.Contains(ua, "opr"):
		browser = "Opera"
	default:
		browser = "Other"
	}

	// Detect OS
	switch {
	case strings.Contains(ua, "windows"):
		os = "Windows"
	case strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os"):
		os = "macOS"
	case strings.Contains(ua, "linux") && !strings.Contains(ua, "android"):
		os = "Linux"
	case strings.Contains(ua, "android"):
		os = "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		os = "iOS"
	default:
		os = "Other"
	}

	// Detect device type
	switch {
	case strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "android"):
		if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
			deviceType = "Tablet"
		} else {
			deviceType = "Mobile"
		}
	default:
		deviceType = "Desktop"
	}

	return
}

// validateURL validates that a URL is well-formed
func validateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Add scheme if missing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	return nil
}

// parseExpiration parses an expiration string into a duration
func parseExpiration(expiresIn string) (time.Duration, error) {
	switch expiresIn {
	case "1h", "1 hour":
		return 1 * time.Hour, nil
	case "24h", "1 day":
		return 24 * time.Hour, nil
	case "7d", "7 days":
		return 7 * 24 * time.Hour, nil
	case "30d", "30 days":
		return 30 * 24 * time.Hour, nil
	case "never", "":
		return 0, nil
	default:
		return time.ParseDuration(expiresIn)
	}
}
