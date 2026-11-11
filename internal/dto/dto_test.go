package dto

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/model"
)

func TestRegisterRequestValidate(t *testing.T) {
	valid := RegisterRequest{Username: "alice", Email: "alice@example.com", Password: "test-password-123"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	cases := []struct {
		name string
		req  RegisterRequest
	}{
		{"short username", RegisterRequest{Username: "ab", Email: valid.Email, Password: valid.Password}},
		{"bad email", RegisterRequest{Username: valid.Username, Email: "not-an-email", Password: valid.Password}},
		{"short password", RegisterRequest{Username: valid.Username, Email: valid.Email, Password: "123"}},
		{"long password", RegisterRequest{Username: valid.Username, Email: valid.Email, Password: string(make([]byte, 80))}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.req.Validate(); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestCreateRequestsValidate(t *testing.T) {
	if err := (CreateJobRequest{Title: "job"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (CreateJobRequest{}).Validate(); err == nil {
		t.Error("empty title should fail")
	}
	if err := (CreateTaskRequest{Title: "task"}).Validate(); err == nil {
		t.Error("missing job_id should fail")
	}
	if err := (CreateTaskRequest{JobID: 1, Title: "task"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (UpdateTaskStatusRequest{Status: model.StatusRunning}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (UpdateTaskStatusRequest{Status: model.Status(99)}).Validate(); err == nil {
		t.Error("invalid status should fail")
	}
}

func TestParsePageQueryClamps(t *testing.T) {
	a := fiber.New()
	a.Get("/", func(c *fiber.Ctx) error { return c.JSON(ParsePageQuery(c)) })

	resp, err := a.Test(httptest.NewRequest("GET", "/?page=3&page_size=500&status=2&job_id=7", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var q PageQuery
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		t.Fatal(err)
	}
	if q.Page != 3 || q.PageSize != 100 || q.JobID != 7 {
		t.Fatalf("unexpected page query: %+v", q)
	}
	if q.Status == nil || *q.Status != model.StatusRunning {
		t.Fatalf("unexpected status filter: %+v", q.Status)
	}
	if q.Offset() != 200 {
		t.Errorf("Offset() = %d, want 200", q.Offset())
	}

	resp2, _ := a.Test(httptest.NewRequest("GET", "/", nil), -1)
	defer resp2.Body.Close()
	var def PageQuery
	_ = json.NewDecoder(resp2.Body).Decode(&def)
	if def.Page != 1 || def.PageSize != 20 || def.Offset() != 0 {
		t.Fatalf("unexpected defaults: %+v", def)
	}
}
