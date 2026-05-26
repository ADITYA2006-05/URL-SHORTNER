-- URL Shortener Database Schema

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS urls (
    id            BIGSERIAL PRIMARY KEY,
    short_code    VARCHAR(20) UNIQUE NOT NULL,
    original_url  TEXT NOT NULL,
    custom_alias  BOOLEAN DEFAULT FALSE,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    expires_at    TIMESTAMPTZ,
    click_count   BIGINT DEFAULT 0,
    is_active     BOOLEAN DEFAULT TRUE,
    user_id       BIGINT REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS clicks (
    id          BIGSERIAL PRIMARY KEY,
    url_id      BIGINT REFERENCES urls(id) ON DELETE CASCADE,
    clicked_at  TIMESTAMPTZ DEFAULT NOW(),
    ip_address  VARCHAR(45),
    user_agent  TEXT,
    referer     TEXT,
    device_type VARCHAR(20),
    browser     VARCHAR(50),
    os          VARCHAR(50)
);

CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);
CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_urls_is_active ON urls(is_active);
CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls(user_id);
CREATE INDEX IF NOT EXISTS idx_clicks_url_id ON clicks(url_id);
CREATE INDEX IF NOT EXISTS idx_clicks_clicked_at ON clicks(clicked_at);
