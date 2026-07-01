package otp

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/hkgroup/backend/internal/platform/idgen"
)

// Service issues and verifies OTP challenges, stored in Redis with a TTL. Used to sign contracts.
type Service struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(rdb *redis.Client) *Service {
	return &Service{rdb: rdb, ttl: 5 * time.Minute}
}

type Challenge struct {
	Ref  string // opaque reference stored on the contract
	Code string // the 6-digit code delivered to the user (via SMS in prod)
}

// Issue creates a challenge bound to a purpose+subject (e.g. "contract", contractID).
func (s *Service) Issue(ctx context.Context, purpose, subject string) (Challenge, error) {
	ref := uuid.NewString()
	code := idgen.OTP()
	key := s.key(purpose, subject, ref)
	if err := s.rdb.Set(ctx, key, code, s.ttl).Err(); err != nil {
		return Challenge{}, err
	}
	return Challenge{Ref: ref, Code: code}, nil
}

// Verify checks the code for a challenge and consumes it (single use).
func (s *Service) Verify(ctx context.Context, purpose, subject, ref, code string) (bool, error) {
	key := s.key(purpose, subject, ref)
	stored, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if stored != code {
		return false, nil
	}
	s.rdb.Del(ctx, key)
	return true, nil
}

func (s *Service) key(purpose, subject, ref string) string {
	return fmt.Sprintf("otp:%s:%s:%s", purpose, subject, ref)
}
