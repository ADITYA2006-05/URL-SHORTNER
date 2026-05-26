package domain

import "time"

// User represents a registered user account
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Email        string    `json:"email,omitempty"`
	GoogleID     string    `json:"google_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// URL represents a shortened URL
type URL struct {
	ID          int64      `json:"id"`
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	CustomAlias bool       `json:"custom_alias"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	ClickCount  int64      `json:"click_count"`
	IsActive    bool       `json:"is_active"`
	ShortURL    string     `json:"short_url,omitempty"`
	UserID      int64      `json:"user_id,omitempty"`
}

// Click represents a click analytics event
type Click struct {
	ID         int64     `json:"id"`
	URLID      int64     `json:"url_id"`
	ClickedAt  time.Time `json:"clicked_at"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	Referer    string    `json:"referer,omitempty"`
	DeviceType string    `json:"device_type,omitempty"`
	Browser    string    `json:"browser,omitempty"`
	OS         string    `json:"os,omitempty"`
}

// URLStats represents analytics for a URL
type URLStats struct {
	URL              URL              `json:"url"`
	TotalClicks      int64            `json:"total_clicks"`
	ClicksPerDay     []DailyClicks    `json:"clicks_per_day"`
	TopReferers      []RefererCount   `json:"top_referers"`
	DeviceBreakdown  []DeviceCount    `json:"device_breakdown"`
	BrowserBreakdown []BrowserCount   `json:"browser_breakdown"`
}

// DailyClicks represents clicks per day
type DailyClicks struct {
	Date   string `json:"date"`
	Clicks int64  `json:"clicks"`
}

// RefererCount represents a referer with click count
type RefererCount struct {
	Referer string `json:"referer"`
	Count   int64  `json:"count"`
}

// DeviceCount represents device type with count
type DeviceCount struct {
	Device string `json:"device"`
	Count  int64  `json:"count"`
}

// BrowserCount represents browser with count
type BrowserCount struct {
	Browser string `json:"browser"`
	Count   int64  `json:"count"`
}

// ShortenRequest represents the API request to create a short URL
type ShortenRequest struct {
	URL         string `json:"url"`
	CustomAlias string `json:"custom_alias,omitempty"`
	ExpiresIn   string `json:"expires_in,omitempty"`
}

// ShortenResponse represents the API response after creating a short URL
type ShortenResponse struct {
	ShortURL    string     `json:"short_url"`
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// AuthRequest represents a login or register request
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse represents the response containing the JWT token
type AuthResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}
