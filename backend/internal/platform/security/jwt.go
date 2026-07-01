package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenKind string

const (
	AccessToken  TokenKind = "access"
	RefreshToken TokenKind = "refresh"
)

type Claims struct {
	UserID uuid.UUID `json:"uid"`
	Role   string    `json:"role"`
	Kind   TokenKind `json:"knd"`
	jwt.RegisteredClaims
}

// JWTManager issues and verifies access/refresh tokens (HS256).
type JWTManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (m *JWTManager) issue(userID uuid.UUID, role string, kind TokenKind, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		Kind:   kind,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *JWTManager) IssueAccess(userID uuid.UUID, role string) (string, error) {
	return m.issue(userID, role, AccessToken, m.accessTTL)
}

func (m *JWTManager) IssueRefresh(userID uuid.UUID, role string) (string, error) {
	return m.issue(userID, role, RefreshToken, m.refreshTTL)
}

var ErrInvalidToken = errors.New("invalid token")

func (m *JWTManager) Verify(token string, want TokenKind) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid || claims.Kind != want {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
