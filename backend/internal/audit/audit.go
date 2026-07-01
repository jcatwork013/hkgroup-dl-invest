package audit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/hkgroup/backend/internal/db"
)

// Write appends an immutable audit entry. MUST be called with the same *db.Queries (tx-bound) as
// the mutation it records — this is how INVARIANT 8 ("approve/issue/payout logs in the same
// transaction") is guaranteed. before/after are JSON-marshalled snapshots (nil allowed).
func Write(ctx context.Context, q *db.Queries, actor uuid.NullUUID, action, entity, entityID string, before, after any) error {
	_, err := q.InsertAuditLog(ctx, db.InsertAuditLogParams{
		ActorID:  actor,
		Action:   action,
		Entity:   entity,
		EntityID: entityID,
		Before:   marshal(before),
		After:    marshal(after),
	})
	return err
}

func marshal(v any) []byte {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

// Actor wraps a user id as the nullable actor column.
func Actor(id uuid.UUID) uuid.NullUUID { return uuid.NullUUID{UUID: id, Valid: true} }

// System is the actor for non-user-initiated entries.
var System = uuid.NullUUID{}
