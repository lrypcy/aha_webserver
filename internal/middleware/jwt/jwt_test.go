package jwt

import (
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

func newManager(t *testing.T) *Manager {
	t.Helper()
	m, err := New("test-jwt-secret-0123456789abcdef0123456789", time.Hour, "aha_webserver-test")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestGenerateAndParse(t *testing.T) {
	m := newManager(t)
	raw, err := m.Generate(42, "alice", "user")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := m.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 || claims.Username != "alice" || claims.Role != "user" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.Issuer != "aha_webserver-test" {
		t.Errorf("issuer = %q", claims.Issuer)
	}
	if m.Expiration() != time.Hour {
		t.Errorf("Expiration() = %v", m.Expiration())
	}
}

func TestParseRejectsBadTokens(t *testing.T) {
	m := newManager(t)
	good, _ := m.Generate(1, "alice", "user")

	cases := []struct {
		name string
		raw  string
	}{
		{"garbage", "not-a-token"},
		{"wrong secret", signWith(t, "another-test-secret-0123456789abcdef", m.issuer, time.Now().Add(time.Hour))},
		{"expired", signWith(t, string(m.secret), m.issuer, time.Now().Add(-time.Hour))},
		{"wrong issuer", signWith(t, string(m.secret), "other-issuer", time.Now().Add(time.Hour))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := m.Parse(tc.raw); err == nil {
				t.Error("expected parse error")
			}
		})
	}
	if _, err := m.Parse(good); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
}

func TestExtractBearer(t *testing.T) {
	if got, err := ExtractBearer("Bearer abc.def.ghi"); err != nil || got != "abc.def.ghi" {
		t.Fatalf("got %q, err %v", got, err)
	}
	if _, err := ExtractBearer("bearer abc"); err != nil {
		t.Error("scheme should be case-insensitive")
	}
	for _, bad := range []string{"", "abc", "Basic abc", "Bearer", "Bearer  "} {
		if _, err := ExtractBearer(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestNewRejectsWeakConfig(t *testing.T) {
	if _, err := New("short", time.Hour, ""); err == nil {
		t.Error("short secret should be rejected")
	}
	if _, err := New("test-jwt-secret-0123456789abcdef0123456789", 0, ""); err == nil {
		t.Error("non-positive expiration should be rejected")
	}
}

func signWith(t *testing.T, secret, issuer string, expires time.Time) string {
	t.Helper()
	claims := Claims{
		UserID: 1, Role: "user", Username: "alice",
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    issuer,
			ExpiresAt: gojwt.NewNumericDate(expires),
			IssuedAt:  gojwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	}
	raw, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
