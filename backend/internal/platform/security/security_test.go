package security

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("Password123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(hash, "Password123") {
		t.Fatal("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("wrong password accepted")
	}
}

func TestJWTRoundTrip(t *testing.T) {
	m := NewJWTManager("a-test-secret-of-sufficient-length!!", time.Minute, time.Hour)
	uid := uuid.New()

	access, err := m.IssueAccess(uid, "investor")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := m.Verify(access, AccessToken)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.UserID != uid || claims.Role != "investor" {
		t.Fatalf("claims mismatch: %+v", claims)
	}

	// An access token must NOT verify as a refresh token (kind separation).
	if _, err := m.Verify(access, RefreshToken); err == nil {
		t.Fatal("access token wrongly accepted as refresh")
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	m := NewJWTManager("a-test-secret-of-sufficient-length!!", -time.Minute, time.Hour)
	tok, _ := m.IssueAccess(uuid.New(), "admin")
	if _, err := m.Verify(tok, AccessToken); err == nil {
		t.Fatal("expired token accepted")
	}
}
