package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// 避免开发者机器上的 AHA_* 环境变量干扰默认值断言。
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "AHA_") {
			os.Unsetenv(strings.SplitN(kv, "=", 2)[0])
		}
	}
	os.Exit(m.Run())
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDefaultsAndEphemeralSecret(t *testing.T) {
	c, err := Load(writeConfig(t, "server:\n  port: 9000\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Server.Port != 9000 {
		t.Errorf("port = %d, want 9000", c.Server.Port)
	}
	if c.Database.Type != "sqlite" {
		t.Errorf("db type = %q, want sqlite", c.Database.Type)
	}
	if c.JWT.Secret == "" || !c.JWT.EphemeralSecret {
		t.Error("dev mode should generate an ephemeral jwt secret")
	}
	if c.JWT.Expiration != 24*time.Hour {
		t.Errorf("expiration = %v, want 24h", c.JWT.Expiration)
	}
	if c.Addr() != ":9000" {
		t.Errorf("Addr() = %q", c.Addr())
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("AHA_SERVER_PORT", "9999")
	t.Setenv("AHA_DB_TYPE", "postgresql")
	t.Setenv("AHA_JWT_SECRET", "test-jwt-secret-0123456789abcdef0123456789")
	t.Setenv("AHA_JWT_EXPIRATION", "2h")

	c, err := Load(writeConfig(t, "server:\n  port: 9000\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Server.Port != 9999 {
		t.Errorf("env override failed: port = %d", c.Server.Port)
	}
	if c.Database.Type != "postgresql" {
		t.Errorf("env override failed: db type = %q", c.Database.Type)
	}
	if c.JWT.Secret != "test-jwt-secret-0123456789abcdef0123456789" || c.JWT.EphemeralSecret {
		t.Errorf("env override failed: secret = %q, ephemeral = %v", c.JWT.Secret, c.JWT.EphemeralSecret)
	}
	if c.JWT.Expiration != 2*time.Hour {
		t.Errorf("env override failed: expiration = %v", c.JWT.Expiration)
	}
}

func TestProductionRequiresStrongSecretAndExplicitCORS(t *testing.T) {
	t.Setenv("AHA_SERVER_ENV", "production")
	t.Setenv("AHA_JWT_SECRET", "")
	t.Setenv("AHA_CORS_ORIGINS", "*")
	if _, err := Load(writeConfig(t, "server:\n  env: dev\n")); err == nil {
		t.Fatal("expected error for empty production jwt secret")
	}

	t.Setenv("AHA_JWT_SECRET", "test-jwt-secret-0123456789abcdef0123456789")
	if _, err := Load(writeConfig(t, "")); err == nil {
		t.Fatal("expected error for wildcard CORS in production")
	}

	t.Setenv("AHA_CORS_ORIGINS", "https://example.com")
	c, err := Load(writeConfig(t, ""))
	if err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
	if !c.IsProduction() {
		t.Error("IsProduction() should be true")
	}
}

func TestValidateRejectsBadValues(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Config)
	}{
		{"bad port", func(c *Config) { c.Server.Port = 0 }},
		{"bad db type", func(c *Config) { c.Database.Type = "oracle" }},
		{"empty dbname", func(c *Config) { c.Database.DBName = "" }},
		{"no expiration", func(c *Config) { c.JWT.Expiration = 0 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := Load(writeConfig(t, ""))
			if err != nil {
				t.Fatal(err)
			}
			tc.mut(c)
			if err := c.Validate(); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}
