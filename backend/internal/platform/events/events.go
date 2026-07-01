package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
)

// Publisher emits domain events. Implementations: NATS JetStream (prod) and Noop (fallback/tests).
type Publisher interface {
	Publish(ctx context.Context, subject string, payload any) error
	Close()
}

// Noop discards events; used when NATS is unavailable so the app still runs.
type Noop struct{}

func (Noop) Publish(context.Context, string, any) error { return nil }
func (Noop) Close()                                     {}

// NATS publishes to a JetStream stream ("HK", subjects "hk.>").
type NATS struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func ConnectNATS(url string) (*NATS, error) {
	nc, err := nats.Connect(url, nats.Timeout(5*time.Second), nats.RetryOnFailedConnect(true), nats.MaxReconnects(-1))
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}
	// Idempotent stream creation.
	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "HK",
		Subjects: []string{"hk.>"},
		Storage:  nats.FileStorage,
	})
	return &NATS{nc: nc, js: js}, nil
}

func (n *NATS) Publish(_ context.Context, subject string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = n.js.Publish(subject, b)
	return err
}

func (n *NATS) Close() { n.nc.Close() }
