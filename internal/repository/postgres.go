package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/urlshortener/internal/domain"
)

// PostgresRepo handles all PostgreSQL operations
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgreSQL repository
func NewPostgresRepo(databaseURL string) (*PostgresRepo, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &PostgresRepo{pool: pool}, nil
}

// Close closes the database connection pool
func (r *PostgresRepo) Close() {
	r.pool.Close()
}

// RunMigrations runs the SQL migration file
func (r *PostgresRepo) RunMigrations(sql string) error {
	_, err := r.pool.Exec(context.Background(), sql)
	return err
}

// CreateUser inserts a new user into the database
func (r *PostgresRepo) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query, user.Username, user.PasswordHash).Scan(&user.ID, &user.CreatedAt)
}

// GetUserByUsername retrieves a user by their username
func (r *PostgresRepo) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	user := &domain.User{Username: username}
	query := `
		SELECT id, COALESCE(password_hash, ''), created_at, COALESCE(email, ''), COALESCE(google_id, '')
		FROM users
		WHERE username = $1`

	err := r.pool.QueryRow(ctx, query, username).Scan(&user.ID, &user.PasswordHash, &user.CreatedAt, &user.Email, &user.GoogleID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByGoogleID retrieves a user by their Google ID
func (r *PostgresRepo) GetUserByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	user := &domain.User{GoogleID: googleID}
	query := `
		SELECT id, username, COALESCE(password_hash, ''), COALESCE(email, ''), created_at
		FROM users
		WHERE google_id = $1`

	err := r.pool.QueryRow(ctx, query, googleID).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Email, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByEmail retrieves a user by their email address
func (r *PostgresRepo) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	user := &domain.User{Email: email}
	query := `
		SELECT id, username, COALESCE(password_hash, ''), COALESCE(google_id, ''), created_at
		FROM users
		WHERE email = $1`

	err := r.pool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.GoogleID, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// LinkGoogleAccount links a Google ID to an existing user account
func (r *PostgresRepo) LinkGoogleAccount(ctx context.Context, userID int64, googleID string) error {
	query := `UPDATE users SET google_id = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, googleID, userID)
	return err
}

// CreateOAuthUser creates a new passwordless user from Google OAuth
func (r *PostgresRepo) CreateOAuthUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (username, password_hash, email, google_id)
		VALUES ($1, NULL, $2, $3)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query, user.Username, user.Email, user.GoogleID).Scan(&user.ID, &user.CreatedAt)
}

// CreateURL inserts a new URL record and returns it with the generated ID
func (r *PostgresRepo) CreateURL(ctx context.Context, url *domain.URL) error {
	query := `
		INSERT INTO urls (short_code, original_url, custom_alias, expires_at, user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, click_count, is_active`

	return r.pool.QueryRow(ctx, query,
		url.ShortCode,
		url.OriginalURL,
		url.CustomAlias,
		url.ExpiresAt,
		url.UserID,
	).Scan(&url.ID, &url.CreatedAt, &url.ClickCount, &url.IsActive)
}

// GetURLByShortCode retrieves a URL by its short code
func (r *PostgresRepo) GetURLByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	url := &domain.URL{}
	query := `
		SELECT id, short_code, original_url, custom_alias, created_at, 
		       expires_at, click_count, is_active, user_id
		FROM urls
		WHERE short_code = $1 AND is_active = true`

	err := r.pool.QueryRow(ctx, query, shortCode).Scan(
		&url.ID, &url.ShortCode, &url.OriginalURL, &url.CustomAlias,
		&url.CreatedAt, &url.ExpiresAt, &url.ClickCount, &url.IsActive, &url.UserID,
	)
	if err != nil {
		return nil, err
	}

	// Check if expired
	if url.ExpiresAt != nil && url.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("url has expired")
	}

	return url, nil
}

// GetURLByShortCodeAndUser retrieves a URL by its short code and checks ownership
func (r *PostgresRepo) GetURLByShortCodeAndUser(ctx context.Context, shortCode string, userID int64) (*domain.URL, error) {
	url := &domain.URL{}
	query := `
		SELECT id, short_code, original_url, custom_alias, created_at, 
		       expires_at, click_count, is_active, user_id
		FROM urls
		WHERE short_code = $1 AND user_id = $2 AND is_active = true`

	err := r.pool.QueryRow(ctx, query, shortCode, userID).Scan(
		&url.ID, &url.ShortCode, &url.OriginalURL, &url.CustomAlias,
		&url.CreatedAt, &url.ExpiresAt, &url.ClickCount, &url.IsActive, &url.UserID,
	)
	if err != nil {
		return nil, err
	}

	// Check if expired
	if url.ExpiresAt != nil && url.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("url has expired")
	}

	return url, nil
}

// GetRecentURLs retrieves the most recent URLs for a specific user
func (r *PostgresRepo) GetRecentURLs(ctx context.Context, limit int, userID int64) ([]domain.URL, error) {
	query := `
		SELECT id, short_code, original_url, custom_alias, created_at,
		       expires_at, click_count, is_active, user_id
		FROM urls
		WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []domain.URL
	for rows.Next() {
		var u domain.URL
		err := rows.Scan(
			&u.ID, &u.ShortCode, &u.OriginalURL, &u.CustomAlias,
			&u.CreatedAt, &u.ExpiresAt, &u.ClickCount, &u.IsActive, &u.UserID,
		)
		if err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, nil
}

// IncrementClickCount atomically increments the click count for a URL
func (r *PostgresRepo) IncrementClickCount(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "UPDATE urls SET click_count = click_count + 1 WHERE id = $1", id)
	return err
}

// RecordClick inserts a click analytics event
func (r *PostgresRepo) RecordClick(ctx context.Context, click *domain.Click) error {
	query := `
		INSERT INTO clicks (url_id, ip_address, user_agent, referer, device_type, browser, os)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		click.URLID, click.IPAddress, click.UserAgent,
		click.Referer, click.DeviceType, click.Browser, click.OS,
	)
	return err
}

// DeleteURL soft-deletes a URL by short code and checks ownership
func (r *PostgresRepo) DeleteURL(ctx context.Context, shortCode string, userID int64) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE urls SET is_active = false WHERE short_code = $1 AND user_id = $2 AND is_active = true",
		shortCode, userID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("url not found")
	}
	return nil
}

// GetURLStats retrieves analytics for a URL owned by a specific user
func (r *PostgresRepo) GetURLStats(ctx context.Context, shortCode string, userID int64) (*domain.URLStats, error) {
	// Get the URL first, ensuring it belongs to the user
	url, err := r.GetURLByShortCodeAndUser(ctx, shortCode, userID)
	if err != nil {
		return nil, err
	}

	stats := &domain.URLStats{
		URL:         *url,
		TotalClicks: url.ClickCount,
	}

	// Clicks per day (last 30 days)
	clicksQuery := `
		SELECT DATE(clicked_at) as date, COUNT(*) as clicks
		FROM clicks
		WHERE url_id = $1 AND clicked_at >= NOW() - INTERVAL '30 days'
		GROUP BY DATE(clicked_at)
		ORDER BY date DESC`

	rows, err := r.pool.Query(ctx, clicksQuery, url.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dc domain.DailyClicks
			var date time.Time
			if err := rows.Scan(&date, &dc.Clicks); err == nil {
				dc.Date = date.Format("2006-01-02")
				stats.ClicksPerDay = append(stats.ClicksPerDay, dc)
			}
		}
	}

	// Top referers
	referersQuery := `
		SELECT COALESCE(NULLIF(referer, ''), 'Direct') as referer, COUNT(*) as count
		FROM clicks
		WHERE url_id = $1
		GROUP BY referer
		ORDER BY count DESC
		LIMIT 10`

	rows2, err := r.pool.Query(ctx, referersQuery, url.ID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var rc domain.RefererCount
			if err := rows2.Scan(&rc.Referer, &rc.Count); err == nil {
				stats.TopReferers = append(stats.TopReferers, rc)
			}
		}
	}

	// Device breakdown
	devicesQuery := `
		SELECT COALESCE(NULLIF(device_type, ''), 'Unknown') as device, COUNT(*) as count
		FROM clicks
		WHERE url_id = $1
		GROUP BY device_type
		ORDER BY count DESC`

	rows3, err := r.pool.Query(ctx, devicesQuery, url.ID)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var dc domain.DeviceCount
			if err := rows3.Scan(&dc.Device, &dc.Count); err == nil {
				stats.DeviceBreakdown = append(stats.DeviceBreakdown, dc)
			}
		}
	}

	// Browser breakdown
	browsersQuery := `
		SELECT COALESCE(NULLIF(browser, ''), 'Unknown') as browser, COUNT(*) as count
		FROM clicks
		WHERE url_id = $1
		GROUP BY browser
		ORDER BY count DESC`

	rows4, err := r.pool.Query(ctx, browsersQuery, url.ID)
	if err == nil {
		defer rows4.Close()
		for rows4.Next() {
			var bc domain.BrowserCount
			if err := rows4.Scan(&bc.Browser, &bc.Count); err == nil {
				stats.BrowserBreakdown = append(stats.BrowserBreakdown, bc)
			}
		}
	}

	return stats, nil
}

// CleanupExpiredURLs marks expired URLs as inactive
func (r *PostgresRepo) CleanupExpiredURLs(ctx context.Context) (int64, error) {
	result, err := r.pool.Exec(ctx,
		"UPDATE urls SET is_active = false WHERE expires_at < NOW() AND is_active = true",
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// ShortCodeExists checks if a short code already exists
func (r *PostgresRepo) ShortCodeExists(ctx context.Context, shortCode string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)",
		shortCode,
	).Scan(&exists)
	return exists, err
}

// UpdateShortCode updates the short code for a URL
func (r *PostgresRepo) UpdateShortCode(ctx context.Context, id int64, shortCode string) error {
	_, err := r.pool.Exec(ctx, "UPDATE urls SET short_code = $1 WHERE id = $2", shortCode, id)
	return err
}
