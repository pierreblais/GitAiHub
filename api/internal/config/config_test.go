package config

import (
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestFromEnvRequired(t *testing.T) {
	_, err := FromEnv(env(map[string]string{}))
	if err == nil || !strings.Contains(err.Error(), "GITAIHUB_DATABASE_URL, GITAI_ADDRESS, GITAI_TOKEN") {
		t.Fatalf("want missing vars error, got %v", err)
	}
}

func TestFromEnvDefaultsAndParsing(t *testing.T) {
	c, err := FromEnv(env(map[string]string{
		"GITAIHUB_DATABASE_URL":      "postgres://x",
		"GITAI_ADDRESS":              "tcp://gitai:8075",
		"GITAI_TOKEN":                "t",
		"GITAI_REPOSITORIES":         "a.git, b.git ,",
		"GITAIHUB_USER_GITAI_TOKENS": "pierre=abc, alice=def",
		"GITAIHUB_SESSION_TTL":       "1h",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTPAddr != ":8080" || c.GitaiStorage != "default" {
		t.Fatalf("defaults not applied: %+v", c)
	}
	if len(c.Repositories) != 2 || c.Repositories[1] != "b.git" {
		t.Fatalf("repositories: %v", c.Repositories)
	}
	if c.UserGitaiTokens["alice"] != "def" {
		t.Fatalf("tokens: %v", c.UserGitaiTokens)
	}
	if c.SessionTTL != time.Hour {
		t.Fatalf("ttl %v", c.SessionTTL)
	}
}

func TestFromEnvAdminPair(t *testing.T) {
	_, err := FromEnv(env(map[string]string{
		"GITAIHUB_DATABASE_URL": "x", "GITAI_ADDRESS": "x", "GITAI_TOKEN": "x",
		"GITAIHUB_ADMIN_USERNAME": "pierre",
	}))
	if err == nil {
		t.Fatal("want error for admin username without password")
	}
}

func TestFromEnvBadTokens(t *testing.T) {
	_, err := FromEnv(env(map[string]string{
		"GITAIHUB_DATABASE_URL": "x", "GITAI_ADDRESS": "x", "GITAI_TOKEN": "x",
		"GITAIHUB_USER_GITAI_TOKENS": "nope",
	}))
	if err == nil {
		t.Fatal("want error")
	}
}
