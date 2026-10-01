package config

import (
	"os"
	"time"
)

type Config struct {
	HTTPPort      string
	ListenIP      string
	DBPath        string
	AdminToken    string
	LetEncryptEmail string
	PublicDomain  string
	PollInterval  time.Duration
}

func Load() *Config {
	return &Config{
		HTTPPort:      getEnv("HTTP_PORT", "8080"),
		ListenIP:      getEnv("LISTEN_IP", "0.0.0.0"),
		DBPath:        getEnv("DB_PATH", "./data/microdashboard.db"),
		AdminToken:    getEnv("ADMIN_TOKEN", "admin-change-me"),
		LetEncryptEmail: getEnv("LETENCRYPT_EMAIL", ""),
		PublicDomain:  getEnv("PUBLIC_DOMAIN", "localhost"),
		PollInterval:  parseDurationEnv("TARGET_POLL_INTERVAL", 30*time.Second),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseDurationEnv(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}