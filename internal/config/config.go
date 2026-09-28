package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort         string
	GinMode         string
	DatabaseURL     string
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTTL      time.Duration
	BcryptCost      int
	CacheTTLList    time.Duration
	CacheTTLItem    time.Duration
	ConnectAttempts int
	ConnectBackoff  time.Duration
	RateLimitPrefix string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		AppPort:         env("APP_PORT", "8080"),
		GinMode:         env("GIN_MODE", "release"),
		DatabaseURL:     env("DATABASE_URL", "postgres://restapi:restapi@localhost:5432/restapi?sslmode=disable"),
		RedisAddr:       env("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   env("REDIS_PASSWORD", ""),
		RedisDB:         envInt("REDIS_DB", 0),
		JWTSecret:       env("JWT_SECRET", ""),
		AccessTokenTTL:  envDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTTL:      envDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		BcryptCost:      envInt("BCRYPT_COST", 10),
		CacheTTLList:    envDuration("CACHE_TTL_LIST", 60*time.Second),
		CacheTTLItem:    envDuration("CACHE_TTL_ITEM", 300*time.Second),
		ConnectAttempts: envInt("CONNECT_ATTEMPTS", 30),
		ConnectBackoff:  envDuration("CONNECT_BACKOFF", time.Second),
		RateLimitPrefix: env("RATE_LIMIT_PREFIX", "rl"),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("config: JWT_SECRET wajib diisi")
	}
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("config: JWT_SECRET minimal 32 karakter")
	}
	return cfg
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return parsed
		}
		log.Printf("config: %s bukan angka, pakai default %d", key, fallback)
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if parsed, err := time.ParseDuration(strings.TrimSpace(v)); err == nil {
			return parsed
		}
		log.Printf("config: %s bukan duration valid, pakai default %s", key, fallback)
	}
	return fallback
}

func (c *Config) Addr() string {
	return ":" + c.AppPort
}

func (c *Config) Validate() error {
	if c.RedisDB < 0 {
		return fmt.Errorf("config: REDIS_DB tidak boleh negatif")
	}
	if c.BcryptCost < 4 || c.BcryptCost > 31 {
		return fmt.Errorf("config: BCRYPT_COST harus di rentang 4-31")
	}
	return nil
}
