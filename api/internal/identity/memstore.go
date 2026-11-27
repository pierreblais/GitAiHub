package identity

import (
	"context"
	"sync"
)

// MemStore is an in-memory Store for tests and local experiments.
type MemStore struct {
	mu       sync.Mutex
	users    []*User
	sessions map[string]*Session
}

func (m *MemStore) UserByUsername(_ context.Context, username string) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Username == username {
			c := *u
			return &c, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *MemStore) UserByID(_ context.Context, id int64) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			c := *u
			return &c, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *MemStore) CreateUser(_ context.Context, u *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u.ID = int64(len(m.users) + 1)
	c := *u
	m.users = append(m.users, &c)
	return nil
}

func (m *MemStore) CreateSession(_ context.Context, s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sessions == nil {
		m.sessions = map[string]*Session{}
	}
	c := *s
	m.sessions[s.TokenHash] = &c
	return nil
}

func (m *MemStore) SessionByHash(_ context.Context, hash string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[hash]; ok {
		c := *s
		return &c, nil
	}
	return nil, ErrUnauthenticated
}

func (m *MemStore) DeleteSession(_ context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, hash)
	return nil
}
