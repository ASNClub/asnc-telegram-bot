package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken     string
	DBPath       string
	Cooldown     time.Duration
	MaxDelta     int
	EphemeralTTL time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		return nil, errors.New("BOT_TOKEN is required")
	}

	dbPath := envDefault("DB_PATH", "./data/rep.db")

	cooldownMin, err := envInt("COOLDOWN_MINUTES", 30)
	if err != nil {
		return nil, err
	}
	if cooldownMin < 0 {
		return nil, fmt.Errorf("COOLDOWN_MINUTES must be >= 0, got %d", cooldownMin)
	}

	maxDelta, err := envInt("MAX_DELTA", 10)
	if err != nil {
		return nil, err
	}
	if maxDelta <= 0 {
		return nil, fmt.Errorf("MAX_DELTA must be > 0, got %d", maxDelta)
	}

	ttl, err := envDuration("EPHEMERAL_TTL", 10*time.Second)
	if err != nil {
		return nil, err
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("EPHEMERAL_TTL must be > 0, got %s", ttl)
	}

	return &Config{
		BotToken:     token,
		DBPath:       dbPath,
		Cooldown:     time.Duration(cooldownMin) * time.Minute,
		MaxDelta:     maxDelta,
		EphemeralTTL: ttl,
	}, nil
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid integer %q: %w", key, v, err)
	}
	return n, nil
}

func envDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid duration %q: %w", key, v, err)
	}
	return d, nil
}
