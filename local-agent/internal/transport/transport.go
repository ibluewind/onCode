package transport

import (
	"context"

	"oncode/protocol/messages"
)

// Transport is the Server ↔ Local Agent communication abstraction.
// Phase 1 may use an in-memory harness; gRPC will implement this later (ADR-003).
// Tool handlers must not depend on transport-specific types.
type Transport interface {
	Connect(ctx context.Context) error
	Register(ctx context.Context, agentID string) error
	SendHeartbeat(ctx context.Context, hb messages.HeartbeatPayload) error
	ReceiveRequest(ctx context.Context) (*messages.Envelope, error)
	SendResponse(ctx context.Context, env *messages.Envelope) error
	Close() error
}

// Handler processes a tool-request envelope and returns a response envelope.
type Handler func(ctx context.Context, req *messages.Envelope) (*messages.Envelope, error)

// Loop receives requests until ctx is cancelled or ReceiveRequest returns an error.
func Loop(ctx context.Context, t Transport, handle Handler) error {
	for {
		req, err := t.ReceiveRequest(ctx)
		if err != nil {
			return err
		}
		resp, err := handle(ctx, req)
		if err != nil {
			return err
		}
		if resp != nil {
			if err := t.SendResponse(ctx, resp); err != nil {
				return err
			}
		}
	}
}
