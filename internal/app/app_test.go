package app_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/app"
	"github.com/lrypcy/aha_webserver/internal/config"
	"github.com/lrypcy/aha_webserver/internal/database"
	"github.com/lrypcy/aha_webserver/internal/middleware/jwt"
	"github.com/lrypcy/aha_webserver/internal/model"
	"github.com/lrypcy/aha_webserver/internal/response"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	testSecret = "integration-test-jwt-secret-0123456789-abcdefgh"
	testIssuer = "aha_webserver-test"
)

func newTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.Migrate(gdb); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{
		Server: config.Server{
			Env: "test", Port: 8080,
			ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
			ShutdownTimeout: time.Second, BodyLimit: 1 << 20,
		},
		Database:  config.Database{Type: "sqlite", DBName: ":memory:"},
		JWT:       config.JWT{Secret: testSecret, Expiration: time.Hour, Issuer: testIssuer},
		Log:       config.Log{Level: "error"},
		CORS:      config.CORS{Origins: "*"},
		RateLimit: config.RateLimit{Max: 100000, Expiration: time.Minute},
	}
	tokens, err := jwt.New(testSecret, time.Hour, testIssuer)
	if err != nil {
		t.Fatal(err)
	}
	return app.New(app.Deps{
		Config: cfg, DB: gdb, Tokens: tokens, Version: "test", StartedAt: time.Now(),
	}), gdb
}

func call(t *testing.T, a *fiber.App, method, path, body, token string) (int, response.Body) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := a.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out response.Body
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s %s response: %v", method, path, err)
	}
	return resp.StatusCode, out
}

func doJSON(t *testing.T, a *fiber.App, method, path, token string, payload any) (int, response.Body) {
	t.Helper()
	var body string
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		body = string(raw)
	}
	return call(t, a, method, path, body, token)
}

func registerAndLogin(t *testing.T, a *fiber.App) string {
	t.Helper()
	status, b := doJSON(t, a, "POST", "/api/v1/register", "", map[string]any{
		"username": "alice", "email": "alice@example.com", "password": "password123",
	})
	if status != fiber.StatusCreated {
		t.Fatalf("register: status=%d body=%+v", status, b)
	}
	status, b = doJSON(t, a, "POST", "/api/v1/login", "", map[string]any{
		"username": "alice", "password": "password123",
	})
	if status != fiber.StatusOK {
		t.Fatalf("login: status=%d body=%+v", status, b)
	}
	data, ok := b.Data.(map[string]any)
	if !ok {
		t.Fatalf("login data: %#v", b.Data)
	}
	token, ok := data["token"].(string)
	if !ok || token == "" {
		t.Fatal("login did not return a token")
	}
	return token
}

func asMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %#v", v)
	}
	return m
}

func TestHealthAndUnified404(t *testing.T) {
	a, _ := newTestApp(t)

	for _, path := range []string{"/health", "/ready"} {
		status, b := call(t, a, "GET", path, "", "")
		if status != fiber.StatusOK || b.Code != response.CodeOK {
			t.Fatalf("GET %s: status=%d body=%+v", path, status, b)
		}
	}

	status, b := call(t, a, "GET", "/no-such-route", "", "")
	if status != fiber.StatusNotFound || b.Code != response.CodeNotFound {
		t.Fatalf("404 envelope: status=%d body=%+v", status, b)
	}
	if b.RequestID == "" {
		t.Error("response should carry a request_id")
	}
}

func TestRegisterValidationAndLogin(t *testing.T) {
	a, _ := newTestApp(t)

	status, _ := doJSON(t, a, "POST", "/api/v1/register", "", map[string]any{
		"username": "alice", "email": "alice@example.com", "password": "123",
	})
	if status != fiber.StatusBadRequest {
		t.Errorf("weak password should be rejected, got %d", status)
	}

	status, _ = doJSON(t, a, "POST", "/api/v1/register", "", map[string]any{
		"username": "alice", "email": "bad-email", "password": "password123",
	})
	if status != fiber.StatusBadRequest {
		t.Errorf("invalid email should be rejected, got %d", status)
	}

	token := registerAndLogin(t, a)

	status, _ = doJSON(t, a, "POST", "/api/v1/register", "", map[string]any{
		"username": "alice", "email": "alice@example.com", "password": "password123",
	})
	if status != fiber.StatusConflict {
		t.Errorf("duplicate username should be 409, got %d", status)
	}

	status, _ = doJSON(t, a, "POST", "/api/v1/login", "", map[string]any{
		"username": "alice", "password": "wrong-password",
	})
	if status != fiber.StatusUnauthorized {
		t.Errorf("wrong password should be 401, got %d", status)
	}

	status, _ = call(t, a, "GET", "/api/v1/me", "", "")
	if status != fiber.StatusUnauthorized {
		t.Errorf("missing token should be 401, got %d", status)
	}

	status, b := call(t, a, "GET", "/api/v1/me", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("GET /me: status=%d body=%+v", status, b)
	}
	user := asMap(t, b.Data)
	if _, leaked := user["password"]; leaked {
		t.Error("password must never appear in a user response")
	}
	if user["role"] != model.RoleUser {
		t.Errorf("default role = %v, want user", user["role"])
	}
}

func TestBearerPrefixIsRequired(t *testing.T) {
	a, _ := newTestApp(t)
	token := registerAndLogin(t, a)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.Header.Set("Authorization", token) // 缺少 Bearer 前缀
	resp, err := a.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("raw token should be rejected, got %d", resp.StatusCode)
	}

	req2 := httptest.NewRequest("GET", "/api/v1/me", nil)
	req2.Header.Set("Authorization", "Bearer "+token+"-tampered")
	resp2, err := a.Test(req2, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("tampered token should be rejected, got %d", resp2.StatusCode)
	}
}

func TestJobsAndTasksFlow(t *testing.T) {
	a, _ := newTestApp(t)
	token := registerAndLogin(t, a)

	status, _ := doJSON(t, a, "POST", "/api/v1/jobs", "", map[string]any{"title": "job-1"})
	if status != fiber.StatusUnauthorized {
		t.Errorf("unauthenticated create should be 401, got %d", status)
	}

	status, b := doJSON(t, a, "POST", "/api/v1/jobs", token, map[string]any{
		"title": "job-1", "labels": []string{"a", "b"}, "config": map[string]any{"k": "v"},
	})
	if status != fiber.StatusCreated {
		t.Fatalf("create job: status=%d body=%+v", status, b)
	}
	first := asMap(t, b.Data)
	if first["id"].(float64) != 1 {
		t.Errorf("unexpected first job id: %v", first["id"])
	}

	// DTO 不含 ID 字段：尝试 mass assignment 不能覆盖主键。
	status, b = doJSON(t, a, "POST", "/api/v1/jobs", token, map[string]any{
		"id": 999, "title": "job-2",
	})
	if status != fiber.StatusCreated {
		t.Fatalf("create job 2: status=%d body=%+v", status, b)
	}
	second := asMap(t, b.Data)
	if second["id"].(float64) == 999 {
		t.Error("client-supplied id overwrote the primary key")
	}

	status, b = call(t, a, "GET", "/api/v1/jobs?page=1&page_size=10", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("list jobs: status=%d body=%+v", status, b)
	}
	page := asMap(t, b.Data)
	if page["total"].(float64) != 2 || page["page_size"].(float64) != 10 {
		t.Errorf("unexpected page: %+v", page)
	}

	status, _ = call(t, a, "GET", "/api/v1/jobs/1", "", token)
	if status != fiber.StatusOK {
		t.Errorf("get job: got %d", status)
	}
	status, _ = call(t, a, "GET", "/api/v1/jobs/99999", "", token)
	if status != fiber.StatusNotFound {
		t.Errorf("missing job should be 404, got %d", status)
	}

	// 普通用户不能删除 Job。
	status, _ = call(t, a, "DELETE", "/api/v1/jobs/1", "", token)
	if status != fiber.StatusForbidden {
		t.Errorf("non-admin delete should be 403, got %d", status)
	}

	status, b = doJSON(t, a, "POST", "/api/v1/tasks", token, map[string]any{
		"job_id": 1, "title": "task-1", "labels": []string{"x"},
	})
	if status != fiber.StatusCreated {
		t.Fatalf("create task: status=%d body=%+v", status, b)
	}
	status, _ = doJSON(t, a, "POST", "/api/v1/tasks", token, map[string]any{
		"job_id": 99999, "title": "orphan",
	})
	if status != fiber.StatusBadRequest {
		t.Errorf("unknown job_id should be 400, got %d", status)
	}

	status, b = call(t, a, "GET", "/api/v1/tasks?job_id=1", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("list tasks: status=%d body=%+v", status, b)
	}
	if asMap(t, b.Data)["total"].(float64) != 1 {
		t.Errorf("unexpected task list: %+v", b.Data)
	}

	status, b = doJSON(t, a, "PATCH", "/api/v1/tasks/1/status", token, map[string]any{"status": 2})
	if status != fiber.StatusOK {
		t.Fatalf("update status: status=%d body=%+v", status, b)
	}
	if asMap(t, b.Data)["status"].(float64) != 2 {
		t.Errorf("status not updated: %+v", b.Data)
	}
	status, _ = doJSON(t, a, "PATCH", "/api/v1/tasks/1/status", token, map[string]any{"status": 99})
	if status != fiber.StatusBadRequest {
		t.Errorf("invalid status should be 400, got %d", status)
	}
}

func TestAdminCanDeleteJob(t *testing.T) {
	a, db := newTestApp(t)
	token := registerAndLogin(t, a)
	_, _ = doJSON(t, a, "POST", "/api/v1/jobs", token, map[string]any{"title": "to-delete"})

	if err := db.Model(&model.User{}).Where("username = ?", "alice").Update("role", model.RoleAdmin).Error; err != nil {
		t.Fatal(err)
	}
	adminToken := registerAndLogin(t, a)

	status, b := call(t, a, "DELETE", "/api/v1/jobs/1", "", adminToken)
	if status != fiber.StatusOK {
		t.Fatalf("admin delete: status=%d body=%+v", status, b)
	}
	status, _ = call(t, a, "GET", "/api/v1/jobs/1", "", adminToken)
	if status != fiber.StatusNotFound {
		t.Errorf("soft-deleted job should be 404, got %d", status)
	}
}

func TestSwaggerDocumentIsServed(t *testing.T) {
	a, _ := newTestApp(t)
	req := httptest.NewRequest("GET", "/swagger/doc.json", nil)
	resp, err := a.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("swagger doc: status=%d", resp.StatusCode)
	}
	var doc map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc["basePath"] != "/api/v1" {
		t.Errorf("swagger basePath = %v, want /api/v1", doc["basePath"])
	}
}
