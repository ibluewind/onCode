package transport

import (
	"context"
	"errors"
	"sync"

	"oncode/protocol/messages"
)

// Memory is an in-process transport for integration tests before a real server exists.
type Memory struct {
	mu       sync.Mutex
	requests chan *messages.Envelope
	responses chan *messages.Envelope
	closed   bool
	agentID  string
}

func NewMemory(buffer int) *Memory {
	if buffer < 1 {
		buffer = 8
	}
	return &Memory{
		requests:  make(chan *messages.Envelope, buffer),
		responses: make(chan *messages.Envelope, buffer),
	}
}

func (m *Memory) Connect(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (m *Memory) Register(ctx context.Context, agentID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		m.mu.Lock()
		m.agentID = agentID
		m.mu.Unlock()
		return nil
	}
}

func (m *Memory) SendHeartbeat(ctx context.Context, _ messages.HeartbeatPayload) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (m *Memory) ReceiveRequest(ctx context.Context) (*messages.Envelope, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case env, ok := <-m.requests:
		if !ok {
			return nil, errors.New("transport closed")
		}
		return env, nil
	}
}

func (m *Memory) SendResponse(ctx context.Context, env *messages.Envelope) error {
	m.mu.Lock()
	closed := m.closed
	m.mu.Unlock()
	if closed {
		return errors.New("transport closed")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m.responses <- env:
		return nil
	}
}

func (m *Memory) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.requests)
	close(m.responses)
	return nil
}

// InjectRequest enqueues a request for the agent loop (test harness).
func (m *Memory) InjectRequest(ctx context.Context, env *messages.Envelope) error {
	m.mu.Lock()
	closed := m.closed
	m.mu.Unlock()
	if closed {
		return errors.New("transport closed")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m.requests <- env:
		return nil
	}
}

// NextResponse waits for the next agent response (test harness).
func (m *Memory) NextResponse(ctx context.Context) (*messages.Envelope, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case env, ok := <-m.responses:
		if !ok {
			return nil, errors.New("transport closed")
		}
		return env, nil
	}
}

func (m *Memory) AgentID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.agentID
}
