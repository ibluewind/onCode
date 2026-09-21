package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	toolv1 "oncode/protocol/gen/toolv1"
	"oncode/protocol/grpcadapt"
	"oncode/protocol/messages"
	"oncode/protocol/versioning"
)

// Stream은 ADR-003 gRPC bidirectional Envelope 스트림이다.
// 도구 핸들러는 proto 타입 대신 messages.Envelope만 본다. WebSocket 어댑터를 같은 채널 모델에 붙일 수 있다.
type Stream struct {
	target   string
	dial     func(ctx context.Context) (*grpc.ClientConn, error)
	agentID  string
	mu       sync.Mutex
	conn     *grpc.ClientConn
	stream   toolv1.LocalAgentSession_OpenClient
	requests chan *messages.Envelope
	events   chan *messages.Envelope
	cancel   context.CancelFunc
}

// DialOption은 테스트에서 in-process 연결을 주입할 때 쓴다.
type DialOption func(*Stream)

// WithDialer는 기본 TCP 대신 커스텀 ClientConn을 쓰게 한다. bufconn 테스트용.
func WithDialer(d func(ctx context.Context) (*grpc.ClientConn, error)) DialOption {
	return func(s *Stream) { s.dial = d }
}

// NewStream은 아직 연결하지 않은 gRPC Transport를 만든다. target은 host:port다.
func NewStream(target, agentID string, opts ...DialOption) *Stream {
	s := &Stream{
		target:   target,
		agentID:  agentID,
		requests: make(chan *messages.Envelope, 16),
		events:   make(chan *messages.Envelope, 16),
		dial: func(ctx context.Context) (*grpc.ClientConn, error) {
			return grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Connect는 outbound gRPC 스트림을 연다. Local Agent가 클라이언트다.
func (s *Stream) Connect(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("nil grpc stream")
	}
	conn, err := s.dial(ctx)
	if err != nil {
		return fmt.Errorf("grpc dial: %w", err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	client := toolv1.NewLocalAgentSessionClient(conn)
	stream, err := client.Open(runCtx)
	if err != nil {
		cancel()
		_ = conn.Close()
		return fmt.Errorf("grpc open: %w", err)
	}
	s.mu.Lock()
	s.conn = conn
	s.stream = stream
	s.cancel = cancel
	s.mu.Unlock()
	go s.recvLoop()
	return nil
}

// Register는 agent.register 이벤트를 보낸다. Connect 이후에 호출한다.
func (s *Stream) Register(ctx context.Context, agentID string) error {
	if agentID != "" {
		s.agentID = agentID
	}
	data, _ := json.Marshal(map[string]any{"agent_id": s.agentID})
	return s.SendEvent(ctx, messages.EventAgentRegister, data)
}

// SendHeartbeat는 agent.heartbeat 이벤트를 보낸다.
func (s *Stream) SendHeartbeat(ctx context.Context, hb messages.HeartbeatPayload) error {
	data, _ := json.Marshal(hb)
	return s.SendEvent(ctx, messages.EventAgentHeartbeat, data)
}

// ReceiveRequest는 서버가 보낸 TOOL REQUEST가 올 때까지 막는다.
func (s *Stream) ReceiveRequest(ctx context.Context) (*messages.Envelope, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case env, ok := <-s.requests:
		if !ok {
			return nil, fmt.Errorf("grpc stream closed")
		}
		return env, nil
	}
}

// SendResponse는 도구 응답 Envelope를 스트림에 쓴다.
func (s *Stream) SendResponse(ctx context.Context, env *messages.Envelope) error {
	return s.send(ctx, env)
}

// Events는 서버가 보낸 EVENT Envelope를 받는다. 진행/질문 전달용이다.
func (s *Stream) Events() <-chan *messages.Envelope {
	return s.events
}

// SendEvent는 논리 이벤트 이름을 Envelope EVENT로 감싸 보낸다.
func (s *Stream) SendEvent(ctx context.Context, event string, data json.RawMessage) error {
	env, err := messages.NewEnvelope(messages.TypeEvent, messages.EventPayload{
		Event: event,
		Data:  data,
	})
	if err != nil {
		return err
	}
	env.Actor = &messages.Actor{Type: messages.ActorLocalAgent, ID: s.agentID}
	env.Protocol = versioning.Current
	return s.send(ctx, env)
}

// Close는 스트림과 연결을 끊는다.
func (s *Stream) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	cancel := s.cancel
	conn := s.conn
	s.stream = nil
	s.conn = nil
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (s *Stream) send(ctx context.Context, env *messages.Envelope) error {
	pb, err := grpcadapt.ToProto(env)
	if err != nil {
		return err
	}
	s.mu.Lock()
	stream := s.stream
	s.mu.Unlock()
	if stream == nil {
		return fmt.Errorf("grpc stream not connected")
	}
	done := make(chan error, 1)
	go func() { done <- stream.Send(pb) }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (s *Stream) recvLoop() {
	for {
		s.mu.Lock()
		stream := s.stream
		s.mu.Unlock()
		if stream == nil {
			return
		}
		pb, err := stream.Recv()
		if err != nil {
			return
		}
		env, err := grpcadapt.FromProto(pb)
		if err != nil {
			continue
		}
		switch env.MessageType {
		case messages.TypeRequest:
			select {
			case s.requests <- env:
			case <-time.After(5 * time.Second):
			}
		default:
			select {
			case s.events <- env:
			case <-time.After(5 * time.Second):
			}
		}
	}
}
