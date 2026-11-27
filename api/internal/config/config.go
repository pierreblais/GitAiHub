// Package config reads the whole process configuration from environment
// variables (12-factor, factor III). Nothing is read from files.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the configuration of the GitaiHub API process.
type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	GitaiAddress  string
	GitaiToken    string
	GitaiStorage  string
	Repositories  []string // explicit relative paths; empty means discover
	AdminUsername string
	AdminPassword string
	// UserGitaiTokens maps a GitaiHub username to the gitai token that review
	// comments and verdicts are written with, so gitai records the right actor.
	UserGitaiTokens   map[string]string
	SessionTTL        time.Duration
	CookieSecure      bool
	DiscoveryInterval time.Duration
	LogLevel          string
}

// FromEnv builds a Config from the environment, using getenv (os.Getenv in
// production, a map in tests).
func FromEnv(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	get := func(key, def string) string {
		if v := strings.TrimSpace(getenv(key)); v != "" {
			return v
		}
		return def
	}
	c := Config{
		HTTPAddr:      get("GITAIHUB_HTTP_ADDR", ":8080"),
		DatabaseURL:   get("GITAIHUB_DATABASE_URL", ""),
		GitaiAddress:  get("GITAI_ADDRESS", ""),
		GitaiToken:    get("GITAI_TOKEN", ""),
		GitaiStorage:  get("GITAI_STORAGE", "default"),
		AdminUsername: get("GITAIHUB_ADMIN_USERNAME", ""),
		AdminPassword: get("GITAIHUB_ADMIN_PASSWORD", ""),
		LogLevel:      get("LOG_LEVEL", "info"),
	}
	var missing []string
	for k, v := range map[string]string{
		"GITAIHUB_DATABASE_URL": c.DatabaseURL,
		"GITAI_ADDRESS":         c.GitaiAddress,
		"GITAI_TOKEN":           c.GitaiToken,
	} {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return c, fmt.Errorf("missing required environment variables: %s", strings.Join(sortStrings(missing), ", "))
	}
	if (c.AdminUsername == "") != (c.AdminPassword == "") {
		return c, fmt.Errorf("GITAIHUB_ADMIN_USERNAME and GITAIHUB_ADMIN_PASSWORD must be set together")
	}
	for _, p := range strings.Split(get("GITAI_REPOSITORIES", ""), ",") {
		if p = strings.TrimSpace(p); p != "" {
			c.Repositories = append(c.Repositories, p)
		}
	}
	tokens, err := parsePairs(get("GITAIHUB_USER_GITAI_TOKENS", ""))
	if err != nil {
		return c, fmt.Errorf("GITAIHUB_USER_GITAI_TOKENS: %w", err)
	}
	c.UserGitaiTokens = tokens
	if c.SessionTTL, err = time.ParseDuration(get("GITAIHUB_SESSION_TTL", "720h")); err != nil {
		return c, fmt.Errorf("GITAIHUB_SESSION_TTL: %w", err)
	}
	if c.DiscoveryInterval, err = time.ParseDuration(get("GITAIHUB_DISCOVERY_INTERVAL", "30s")); err != nil {
		return c, fmt.Errorf("GITAIHUB_DISCOVERY_INTERVAL: %w", err)
	}
	if c.CookieSecure, err = strconv.ParseBool(get("GITAIHUB_COOKIE_SECURE", "false")); err != nil {
		return c, fmt.Errorf("GITAIHUB_COOKIE_SECURE: %w", err)
	}
	return c, nil
}

func parsePairs(s string) (map[string]string, error) {
	out := map[string]string{}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		k, v, ok := strings.Cut(pair, "=")
		if !ok || strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			return nil, fmt.Errorf("expected user=token, got %q", pair)
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out, nil
}

func sortStrings(s []string) []string {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
	return s
}
