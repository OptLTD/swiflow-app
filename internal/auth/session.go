// Package auth provides tenant login sessions (bearer tokens).
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessionTTL = 7 * 24 * time.Hour

// Session is an authenticated tenant session.
type Session struct {
	Token     string
	Tid       string
	TenantName string
	ExpiresAt time.Time
}

// Store holds opaque bearer sessions in memory.
type Store struct {
	mu   sync.RWMutex
	byTok map[string]*Session
}

// NewStore creates an empty session store.
func NewStore() *Store {
	return &Store{byTok: map[string]*Session{}}
}

// Create issues a new session for tid.
func (s *Store) Create(tid, name string) (*Session, error) {
	tok, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	sess := &Session{
		Token:      tok,
		Tid:        tid,
		TenantName: name,
		ExpiresAt:  time.Now().Add(sessionTTL),
	}
	s.mu.Lock()
	s.byTok[tok] = sess
	s.mu.Unlock()
	return sess, nil
}

// Get returns a valid session for token, or nil.
func (s *Store) Get(token string) *Session {
	if token == "" {
		return nil
	}
	s.mu.RLock()
	sess := s.byTok[token]
	s.mu.RUnlock()
	if sess == nil {
		return nil
	}
	if time.Now().After(sess.ExpiresAt) {
		s.Delete(token)
		return nil
	}
	return sess
}

// Delete removes a session.
func (s *Store) Delete(token string) {
	s.mu.Lock()
	delete(s.byTok, token)
	s.mu.Unlock()
}

// HashPassword returns a bcrypt hash.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword verifies password against bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return hex.EncodeToString(b), nil
}
