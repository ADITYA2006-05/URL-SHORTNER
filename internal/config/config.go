package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ServerPort         string
	BaseURL            string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	RateLimitRPM       int
	JWTSecret          string
}

func Load() *Config {
	return &Config{
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", "postgres"),
		DBName:             getEnv("DB_NAME", "urlshortener"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvInt("REDIS_DB", 0),
		RateLimitRPM:       getEnvInt("RATE_LIMIT_RPM", 100),
		JWTSecret:          getEnv("JWT_SECRET", "super-secret-development-key-change-in-production"),
	}
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}
