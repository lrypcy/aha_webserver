package response

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func get(t *testing.T, a *fiber.App, path string) (int, Body) {
	t.Helper()
	resp, err := a.Test(httptest.NewRequest("GET", path, nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b Body
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, b
}

func TestOKAndCreatedEnvelope(t *testing.T) {
	a := fiber.New()
	a.Get("/", func(c *fiber.Ctx) error { return OK(c, fiber.Map{"n": 1}) })
	a.Post("/", func(c *fiber.Ctx) error { return Created(c, fiber.Map{"n": 2}) })

	status, b := get(t, a, "/")
	if status != fiber.StatusOK || b.Code != CodeOK || b.Message != "ok" {
		t.Fatalf("unexpected OK response: %d %+v", status, b)
	}

	resp, err := a.Test(httptest.NewRequest("POST", "/", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var created Body
	_ = json.NewDecoder(resp.Body).Decode(&created)
	if resp.StatusCode != fiber.StatusCreated || created.Message != "created" {
		t.Fatalf("unexpected Created response: %d %+v", resp.StatusCode, created)
	}
}

func TestErrorHandlerMasksInternalDetails(t *testing.T) {
	a := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	a.Get("/bad", func(c *fiber.Ctx) error { return BadRequest("invalid field") })
	a.Get("/boom", func(c *fiber.Ctx) error {
		return Internal("internal error", errors.New("super-secret-stack-trace"))
	})

	status, b := get(t, a, "/bad")
	if status != fiber.StatusBadRequest || b.Code != CodeBadRequest || b.Message != "invalid field" {
		t.Fatalf("unexpected 400: %d %+v", status, b)
	}

	resp, err := a.Test(httptest.NewRequest("GET", "/boom", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "super-secret-stack-trace") {
		t.Fatal("internal error detail leaked to client")
	}
	var b2 Body
	_ = json.Unmarshal(payload, &b2)
	if resp.StatusCode != fiber.StatusInternalServerError || b2.Code != CodeInternal {
		t.Fatalf("unexpected 500: %d %+v", resp.StatusCode, b2)
	}
}

func TestAppErrorUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	e := Wrap(fiber.StatusInternalServerError, CodeInternal, "wrapped", cause)
	if !errors.Is(e, cause) {
		t.Fatal("AppError should unwrap to its cause")
	}
}
