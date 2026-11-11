package model

import "testing"

func TestSetPasswordAndVerify(t *testing.T) {
	u := &User{Username: "alice", Email: "alice@example.com", Password: "test-password-123"}
	if err := u.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if u.Password == "test-password-123" {
		t.Fatal("password was not hashed")
	}
	if u.Role != RoleUser {
		t.Errorf("default role = %q, want %q", u.Role, RoleUser)
	}
	if !u.VerifyPassword("test-password-123") {
		t.Error("correct password rejected")
	}
	if u.VerifyPassword("wrong-password") {
		t.Error("wrong password accepted")
	}
	if err := u.SetPassword("another-password"); err != nil {
		t.Fatal(err)
	}
	if !u.VerifyPassword("another-password") {
		t.Error("password rotation failed")
	}
}

func TestStatusHelpers(t *testing.T) {
	for _, s := range AllStatuses() {
		if !s.Valid() || s.String() == "unknown" {
			t.Errorf("status %d should be valid and named", s)
		}
	}
	if Status(99).Valid() || Status(99).String() != "unknown" {
		t.Error("invalid status should be rejected")
	}
}

func TestUserInfoExcludesPassword(t *testing.T) {
	u := &User{Username: "alice", Email: "a@b.com", Password: "hashed", Role: RoleAdmin}
	info := u.Info()
	if info.Role != RoleAdmin || info.Username != "alice" {
		t.Fatalf("unexpected info: %+v", info)
	}
}
