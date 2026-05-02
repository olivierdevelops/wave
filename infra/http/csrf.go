package http

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const CSRFTokenDuration = 1 * time.Hour

// CSRFStore is a thread-safe in-memory store of CSRF tokens.
type CSRFStore struct {
	tokens map[string]time.Time
	mutex  sync.RWMutex
}

// NewCSRFStore creates a ready-to-use CSRF store.
func NewCSRFStore() *CSRFStore {
	return &CSRFStore{tokens: make(map[string]time.Time)}
}

// GenerateToken creates a new CSRF token, stores it, sets the cookie, and
// returns the token string.
func (s *CSRFStore) GenerateToken(w http.ResponseWriter, r *http.Request) string {
	// Reuse existing valid token if present.
	if cookie, err := r.Cookie("csrf_token"); err == nil {
		s.mutex.RLock()
		_, found := s.tokens[cookie.Value]
		s.mutex.RUnlock()
		if found {
			return cookie.Value
		}
	}

	token, err := generateCSRFToken()
	if err != nil {
		return ""
	}

	s.mutex.Lock()
	s.tokens[token] = time.Now()
	s.mutex.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(CSRFTokenDuration.Seconds()),
	})

	return token
}

// Validate checks that the request carries a valid CSRF token.
// Returns nil for safe methods (GET, HEAD, OPTIONS).
func (s *CSRFStore) Validate(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
		return nil
	}

	csrfCookie, err := r.Cookie("csrf_token")
	if err != nil {
		r.Header.Write(os.Stdout)
		http.Error(w, "CSRF token missing from cookie", http.StatusForbidden)
		return fmt.Errorf("CSRF token missing from cookie")
	}

	requestToken := csrfCookie.Value
	if requestToken == "" {
		http.Error(w, "CSRF token missing from request", http.StatusForbidden)
		return fmt.Errorf("CSRF token missing from request")
	}

	s.mutex.RLock()
	tokenTime, found := s.tokens[requestToken]
	s.mutex.RUnlock()

	if !found {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return fmt.Errorf("invalid CSRF token")
	}

	if time.Since(tokenTime) >= CSRFTokenDuration {
		s.mutex.Lock()
		delete(s.tokens, requestToken)
		s.mutex.Unlock()
		http.Error(w, "CSRF token expired", http.StatusForbidden)
		return fmt.Errorf("CSRF token expired")
	}

	// One-time use.
	s.mutex.Lock()
	delete(s.tokens, requestToken)
	s.mutex.Unlock()

	return nil
}

// Middleware wraps next with CSRF validation using this store.
func (s *CSRFStore) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := s.Validate(w, r); err != nil {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CleanupExpired removes tokens that have passed their TTL.
func (s *CSRFStore) CleanupExpired() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	now := time.Now()
	for token, created := range s.tokens {
		if now.Sub(created) >= CSRFTokenDuration {
			delete(s.tokens, token)
		}
	}
}

// StartCleanup launches a background goroutine that purges expired tokens
// every 10 minutes.
func (s *CSRFStore) StartCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			s.CleanupExpired()
		}
	}()
}

func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
