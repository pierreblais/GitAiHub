package identity

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLoginFlow(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC)
	svc := &Service{Store: &MemStore{}, TTL: time.Hour, Now: func() time.Time { return now }}
	if _, created, err := svc.EnsureUser(ctx, "Pierre", "correct-horse", "Pierre"); err != nil || !created {
		t.Fatalf("ensure: %v %v", created, err)
	}
	if _, created, _ := svc.EnsureUser(ctx, "pierre", "other-password", ""); created {
		t.Fatal("second ensure must not create")
	}
	if _, _, _, err := svc.Login(ctx, "pierre", "wrong-pass"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: %v", err)
	}
	if _, _, _, err := svc.Login(ctx, "nobody", "whatever1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown user: %v", err)
	}
	tok, u, _, err := svc.Login(ctx, "PIERRE", "correct-horse")
	if err != nil || u.Username != "pierre" {
		t.Fatalf("login: %v", err)
	}
	if got, err := svc.Authenticate(ctx, tok); err != nil || got.Username != "pierre" {
		t.Fatalf("authenticate: %v", err)
	}
	now = now.Add(2 * time.Hour)
	if _, err := svc.Authenticate(ctx, tok); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expired session must fail: %v", err)
	}
}

func TestLogout(t *testing.T) {
	ctx := context.Background()
	svc := &Service{Store: &MemStore{}}
	_, _, _ = svc.EnsureUser(ctx, "a", "password1", "")
	tok, _, _, _ := svc.Login(ctx, "a", "password1")
	_ = svc.Logout(ctx, tok)
	if _, err := svc.Authenticate(ctx, tok); err == nil {
		t.Fatal("logged out session must fail")
	}
}

func TestShortPassword(t *testing.T) {
	svc := &Service{Store: &MemStore{}}
	if _, _, err := svc.EnsureUser(context.Background(), "a", "short", ""); err == nil {
		t.Fatal("want error")
	}
}
