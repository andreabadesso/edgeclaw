package timescale

import (
	"testing"
)

func TestConfigDefaults_ZeroValues(t *testing.T) {
	cfg := Config{}
	cfg.Defaults()

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 5432 {
		t.Errorf("Port = %d, want %d", cfg.Port, 5432)
	}
	if cfg.SSLMode != "disable" {
		t.Errorf("SSLMode = %q, want %q", cfg.SSLMode, "disable")
	}
	if cfg.MaxConns != 5 {
		t.Errorf("MaxConns = %d, want %d", cfg.MaxConns, 5)
	}
}

func TestConfigDefaults_PreservesExplicitValues(t *testing.T) {
	cfg := Config{
		Host:     "db.example.com",
		Port:     9999,
		User:     "admin",
		Password: "secret",
		DBName:   "mydb",
		SSLMode:  "require",
		MaxConns: 20,
	}
	cfg.Defaults()

	if cfg.Host != "db.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "db.example.com")
	}
	if cfg.Port != 9999 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9999)
	}
	if cfg.SSLMode != "require" {
		t.Errorf("SSLMode = %q, want %q", cfg.SSLMode, "require")
	}
	if cfg.MaxConns != 20 {
		t.Errorf("MaxConns = %d, want %d", cfg.MaxConns, 20)
	}
	if cfg.User != "admin" {
		t.Errorf("User = %q, want %q", cfg.User, "admin")
	}
	if cfg.Password != "secret" {
		t.Errorf("Password = %q, want %q", cfg.Password, "secret")
	}
	if cfg.DBName != "mydb" {
		t.Errorf("DBName = %q, want %q", cfg.DBName, "mydb")
	}
}

func TestConfigDefaults_PartialOverride(t *testing.T) {
	cfg := Config{
		Host: "custom-host",
		Port: 0, // zero => should get default
	}
	cfg.Defaults()

	if cfg.Host != "custom-host" {
		t.Errorf("Host = %q, want %q", cfg.Host, "custom-host")
	}
	if cfg.Port != 5432 {
		t.Errorf("Port = %d, want default %d", cfg.Port, 5432)
	}
}

func TestQueryJSON_NilPoolPanicsOrErrors(t *testing.T) {
	// A Client with a nil pool should not be usable. We verify that calling
	// QueryJSON on a zero-value Client causes a panic (since pool is nil).
	c := &Client{}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when calling QueryJSON with nil pool, but none occurred")
		}
	}()
	_, _ = c.QueryJSON(nil, "SELECT 1") //nolint: staticcheck
}

func TestEnsureSchema_NilPoolPanics(t *testing.T) {
	// Similar to QueryJSON, calling EnsureSchema with a nil pool should panic.
	c := &Client{}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when calling EnsureSchema with nil pool, but none occurred")
		}
	}()
	_ = c.EnsureSchema(nil) //nolint: staticcheck
}
