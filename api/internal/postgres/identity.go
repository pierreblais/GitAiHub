package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pierreblais/gitaihub/api/internal/identity"
)

// IdentityStore implements identity.Store.
type IdentityStore struct{ DB *pgxpool.Pool }

func scanUser(row pgx.Row) (*identity.User, error) {
	var u identity.User
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.CreateTime)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identity.ErrUserNotFound
	}
	return &u, err
}

func (s *IdentityStore) UserByUsername(ctx context.Context, username string) (*identity.User, error) {
	return scanUser(s.DB.QueryRow(ctx, `SELECT id, username, display_name, password_hash, create_time FROM users WHERE username = $1`, username))
}

func (s *IdentityStore) UserByID(ctx context.Context, id int64) (*identity.User, error) {
	return scanUser(s.DB.QueryRow(ctx, `SELECT id, username, display_name, password_hash, create_time FROM users WHERE id = $1`, id))
}

func (s *IdentityStore) CreateUser(ctx context.Context, u *identity.User) error {
	return s.DB.QueryRow(ctx, `INSERT INTO users (username, display_name, password_hash, create_time) VALUES ($1, $2, $3, $4) RETURNING id`,
		u.Username, u.DisplayName, u.PasswordHash, u.CreateTime).Scan(&u.ID)
}

func (s *IdentityStore) CreateSession(ctx context.Context, sess *identity.Session) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, create_time, expire_time) VALUES ($1, $2, $3, $4)`,
		sess.TokenHash, sess.UserID, sess.CreateTime, sess.ExpireTime)
	return err
}

func (s *IdentityStore) SessionByHash(ctx context.Context, hash string) (*identity.Session, error) {
	var sess identity.Session
	err := s.DB.QueryRow(ctx, `SELECT token_hash, user_id, create_time, expire_time FROM sessions WHERE token_hash = $1`, hash).
		Scan(&sess.TokenHash, &sess.UserID, &sess.CreateTime, &sess.ExpireTime)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identity.ErrUnauthenticated
	}
	return &sess, err
}

func (s *IdentityStore) DeleteSession(ctx context.Context, hash string) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hash)
	return err
}
