package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	os.Setenv("HTTP_PORT", "9090")
	os.Setenv("LISTEN_IP", "127.0.0.1")
	os.Setenv("DB_PATH", "/tmp/test.db")
	os.Setenv("ADMIN_TOKEN", "test-admin-token")
	os.Setenv("LETENCRYPT_EMAIL", "test@example.com")
	os.Setenv("PUBLIC_DOMAIN", "test.example.com")
	os.Setenv("TARGET_POLL_INTERVAL", "15s")
	defer os.Unsetenv("HTTP_PORT")
	defer os.Unsetenv("LISTEN_IP")
	defer os.Unsetenv("DB_PATH")
	defer os.Unsetenv("ADMIN_TOKEN")
	defer os.Unsetenv("LETENCRYPT_EMAIL")
	defer os.Unsetenv("PUBLIC_DOMAIN")
	defer os.Unsetenv("TARGET_POLL_INTERVAL")

	cfg := Load()

	if cfg.HTTPPort != "9090" {
		t.Errorf("HTTPPort = %q, want %q", cfg.HTTPPort, "9090")
	}
	if cfg.ListenIP != "127.0.0.1" {
		t.Errorf("ListenIP = %q, want %q", cfg.ListenIP, "127.0.0.1")
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/tmp/test.db")
	}
	if cfg.AdminToken != "test-admin-token" {
		t.Errorf("AdminToken = %q, want %q", cfg.AdminToken, "test-admin-token")
	}
	if cfg.LetEncryptEmail != "test@example.com" {
		t.Errorf("LetEncryptEmail = %q, want %q", cfg.LetEncryptEmail, "test@example.com")
	}
	if cfg.PublicDomain != "test.example.com" {
		t.Errorf("PublicDomain = %q, want %q", cfg.PublicDomain, "test.example.com")
	}
	if cfg.PollInterval != 15*time.Second {
		t.Errorf("PollInterval = %v, want %v", cfg.PollInterval, 15*time.Second)
	}
}

func TestLoadDefaults(t *testing.T) {
	os.Unsetenv("HTTP_PORT")
	os.Unsetenv("LISTEN_IP")
	os.Unsetenv("DB_PATH")
	os.Unsetenv("ADMIN_TOKEN")
	os.Unsetenv("LETENCRYPT_EMAIL")
	os.Unsetenv("PUBLIC_DOMAIN")
	os.Unsetenv("TARGET_POLL_INTERVAL")

	cfg := Load()

	if cfg.HTTPPort != "8080" {
		t.Errorf("default HTTPPort = %q, want %q", cfg.HTTPPort, "8080")
	}
	if cfg.ListenIP != "0.0.0.0" {
		t.Errorf("default ListenIP = %q, want %q", cfg.ListenIP, "0.0.0.0")
	}
	if cfg.DBPath != "./data/microdashboard.db" {
		t.Errorf("default DBPath = %q, want %q", cfg.DBPath, "./data/microdashboard.db")
	}
	if cfg.AdminToken != "admin-change-me" {
		t.Errorf("default AdminToken = %q, want %q", cfg.AdminToken, "admin-change-me")
	}
	if cfg.PollInterval != 30*time.Second {
		t.Errorf("default PollInterval = %v, want %v", cfg.PollInterval, 30*time.Second)
	}
}