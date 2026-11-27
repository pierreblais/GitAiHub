// Package identity is the Identity bounded context: GitaiHub users and their
// browser sessions. Users are separate from gitai agents; a user may map to a
// gitai identity (see review.Service.Tokens) to comment and give verdicts.
package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials is returned for a wrong username or password.
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrUnauthenticated is returned for a missing or expired session.
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrUserNotFound is returned by stores for unknown users.
	ErrUserNotFound = errors.New("user not found")
)

// User is a person who signs in to GitaiHub.
type User struct {
	ID           int64
	Username     string
	DisplayName  string
	PasswordHash string
	CreateTime   time.Time
}

// Session is a signed-in browser session. Only the hash of its token is stored.
type Session struct {
	TokenHash  string
	UserID     int64
	CreateTime time.Time
	ExpireTime time.Time
}

// Store persists users and sessions.
type Store interface {
	UserByUsername(ctx context.Context, username string) (*User, error)
	UserByID(ctx context.Context, id int64) (*User, error)
	CreateUser(ctx context.Context, u *User) error
	CreateSession(ctx context.Context, s *Session) error
	SessionByHash(ctx context.Context, hash string) (*Session, error)
	DeleteSession(ctx context.Context, hash string) error
}

// Service signs users in and out and authenticates requests.
type Service struct {
	Store Store
	TTL   time.Duration
	Now   func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// EnsureUser creates the user if it does not exist yet (used to bootstrap
// the first user from environment variables). An existing user is kept as is.
func (s *Service) EnsureUser(ctx context.Context, username, password, displayName string) (*User, bool, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if u, err := s.Store.UserByUsername(ctx, username); err == nil {
		return u, false, nil
	} else if !errors.Is(err, ErrUserNotFound) {
		return nil, false, err
	}
	if len(password) < 8 {
		return nil, false, fmt.Errorf("password for %q must have at least 8 characters", username)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, false, err
	}
	if displayName == "" {
		displayName = username
	}
	u := &User{Username: username, DisplayName: displayName, PasswordHash: string(hash), CreateTime: s.now()}
	if err := s.Store.CreateUser(ctx, u); err != nil {
		return nil, false, err
	}
	return u, true, nil
}

// Login checks the password and opens a session. It returns the raw session
// token to set as a cookie.
func (s *Service) Login(ctx context.Context, username, password string) (string, *User, *Session, error) {
	u, err := s.Store.UserByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if errors.Is(err, ErrUserNotFound) {
		// Spend the same time as a real check to not reveal which users exist.
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoO5Uq1ZYQyF2pX7mQ6b4H6sY8i4l0G1rG"), []byte(password))
		return "", nil, nil, ErrInvalidCredentials
	}
	if err != nil {
		return "", nil, nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", nil, nil, ErrInvalidCredentials
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	ttl := s.TTL
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	sess := &Session{TokenHash: HashToken(token), UserID: u.ID, CreateTime: s.now(), ExpireTime: s.now().Add(ttl)}
	if err := s.Store.CreateSession(ctx, sess); err != nil {
		return "", nil, nil, err
	}
	return token, u, sess, nil
}

// Authenticate returns the user of a session token.
func (s *Service) Authenticate(ctx context.Context, token string) (*User, error) {
	if token == "" {
		return nil, ErrUnauthenticated
	}
	sess, err := s.Store.SessionByHash(ctx, HashToken(token))
	if err != nil {
		return nil, ErrUnauthenticated
	}
	if !s.now().Before(sess.ExpireTime) {
		_ = s.Store.DeleteSession(ctx, sess.TokenHash)
		return nil, ErrUnauthenticated
	}
	u, err := s.Store.UserByID(ctx, sess.UserID)
	if err != nil {
		return nil, ErrUnauthenticated
	}
	return u, nil
}

// Logout deletes the session of a token.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.Store.DeleteSession(ctx, HashToken(token))
}

// HashToken returns the stored form of a session token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
