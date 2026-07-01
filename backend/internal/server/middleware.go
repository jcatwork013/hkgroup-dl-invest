package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hkgroup/backend/internal/platform/security"
)

type ctxKey string

const (
	ctxUserID ctxKey = "uid"
	ctxRole   ctxKey = "role"
)

// authMiddleware verifies the access token and injects user id + role into context.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" || token == header {
			writeJSON(w, http.StatusUnauthorized, errBody{"missing bearer token", "unauthorized"})
			return
		}
		claims, err := s.jwt.Verify(token, security.AccessToken)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, errBody{"invalid token", "unauthorized"})
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxRole, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireRole enforces RBAC. INVARIANT: an investor token can never reach an admin route.
func (s *Server) requireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userRole(r) != role {
				writeJSON(w, http.StatusForbidden, errBody{"forbidden", "forbidden"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requireAnyRole allows a route for any of the listed roles (e.g. admin OR saler).
func (s *Server) requireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := userRole(r)
			for _, ok := range roles {
				if role == ok {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSON(w, http.StatusForbidden, errBody{"forbidden", "forbidden"})
		})
	}
}

func userID(r *http.Request) uuid.UUID {
	if v, ok := r.Context().Value(ctxUserID).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}

func userRole(r *http.Request) string {
	if v, ok := r.Context().Value(ctxRole).(string); ok {
		return v
	}
	return ""
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ----- simple in-memory rate limiter (per-IP, fixed window) for auth endpoints -----

type rateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{hits: make(map[string][]time.Time), limit: limit, window: window}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-rl.window)
	kept := rl.hits[key][:0]
	for _, t := range rl.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rl.limit {
		rl.hits[key] = kept
		return false
	}
	rl.hits[key] = append(kept, now)
	return true
}

func (s *Server) rateLimit(rl *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.allow(clientIP(r)) {
				writeJSON(w, http.StatusTooManyRequests, errBody{"rate limit exceeded", "rate_limited"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
