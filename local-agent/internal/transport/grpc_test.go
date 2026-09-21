package transport_test

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"oncode/local-agent/internal/agent"
	"oncode/local-agent/internal/config"
	"oncode/local-agent/internal/transport"
	toolv1 "oncode/protocol/gen/toolv1"
	"oncode/protocol/grpcadapt"
	"oncode/protocol/ids"
	"oncode/protocol/messages"
)

const bufSize = 1 << 20

type chatStub struct {
	toolv1.UnimplementedLocalAgentSessionServer
}

func (s *chatStub) Open(stream grpc.BidiStreamingServer[toolv1.Envelope, toolv1.Envelope]) error {
	for {
		pb, err := stream.Recv()
		if err != nil {
			return nil
		}
		env, err := grpcadapt.FromProto(pb)
		if err != nil || env.MessageType != messages.TypeEvent {
			continue
		}
		var ev messages.EventPayload
		if err := json.Unmarshal(env.Payload, &ev); err != nil {
			continue
		}
		if ev.Event != messages.EventChatSubmit {
			continue
		}
		out, err := messages.NewEnvelope(messages.TypeEvent, messages.EventPayload{
			Event: messages.EventWorkflowProgress,
			Data:  json.RawMessage(`{"stage":"WAITING_DESIGN_APPROVAL","status":"WAITING"}`),
		})
		if err != nil {
			return err
		}
		out.WorkItemID = "WI-018f0000-0000-7000-8000-000000000014"
		pbOut, err := grpcadapt.ToProto(out)
		if err != nil {
			return err
		}
		if err := stream.Send(pbOut); err != nil {
			return err
		}
	}
}

type toolStub struct {
	toolv1.UnimplementedLocalAgentSessionServer
	got chan *messages.Envelope
}

func (s *toolStub) Open(stream grpc.BidiStreamingServer[toolv1.Envelope, toolv1.Envelope]) error {
	req, err := messages.NewEnvelope(messages.TypeRequest, messages.ToolRequestPayload{
		CallID:    mustCallID(),
		Tool:      "system.ping",
		Arguments: map[string]any{},
	})
	if err != nil {
		return err
	}
	pb, err := grpcadapt.ToProto(req)
	if err != nil {
		return err
	}
	if err := stream.Send(pb); err != nil {
		return err
	}
	for {
		in, err := stream.Recv()
		if err != nil {
			return nil
		}
		env, err := grpcadapt.FromProto(in)
		if err != nil || env.MessageType != messages.TypeResponse {
			continue
		}
		s.got <- env
		return nil
	}
}

func TestStream_ToolRequestRoundTrip(t *testing.T) {
	lis := bufconn.Listen(bufSize)
	t.Cleanup(func() { _ = lis.Close() })
	stub := &toolStub{got: make(chan *messages.Envelope, 1)}
	srv := grpc.NewServer()
	toolv1.RegisterLocalAgentSessionServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st := transport.NewStream("bufnet", "agent-test", transport.WithDialer(func(ctx context.Context) (*grpc.ClientConn, error) {
		return grpc.NewClient("passthrough:///bufnet",
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return lis.DialContext(ctx)
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}))
	if err := st.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	a, err := newPingAgent(t)
	if err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- transport.Loop(ctx, st, a.HandleEnvelope) }()

	select {
	case env := <-stub.got:
		var resp messages.ToolResponsePayload
		if err := json.Unmarshal(env.Payload, &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Status != messages.StatusSuccess {
			t.Fatalf("status=%s err=%v", resp.Status, resp.Error)
		}
	case err := <-errCh:
		t.Fatalf("loop ended: %v", err)
	case <-ctx.Done():
		t.Fatal("timeout waiting for tool response")
	}
}

func newPingAgent(t *testing.T) (*agent.Agent, error) {
	t.Helper()
	return agent.New(config.Config{AgentVersion: "test", MaxFileBytes: 1024, StateDir: t.TempDir()}, io.Discard)
}

func mustCallID() string {
	return ids.MustNew(ids.Call)
}

func TestStream_ChatSubmitReceivesProgress(t *testing.T) {
	lis := bufconn.Listen(bufSize)
	t.Cleanup(func() { _ = lis.Close() })
	srv := grpc.NewServer()
	toolv1.RegisterLocalAgentSessionServer(srv, &chatStub{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st := transport.NewStream("bufnet", "agent-test", transport.WithDialer(func(ctx context.Context) (*grpc.ClientConn, error) {
		return grpc.NewClient("passthrough:///bufnet",
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return lis.DialContext(ctx)
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}))
	if err := st.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Register(ctx, "agent-test"); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]any{
		"message": "add null check",
		"ide_context": map[string]any{
			"current_file": "UserService.java",
			"selection":    map[string]any{"start_line": 10, "end_line": 20},
		},
	})
	if err := st.SendEvent(ctx, messages.EventChatSubmit, data); err != nil {
		t.Fatal(err)
	}
	select {
	case env := <-st.Events():
		var ev messages.EventPayload
		if err := json.Unmarshal(env.Payload, &ev); err != nil {
			t.Fatal(err)
		}
		if ev.Event != messages.EventWorkflowProgress {
			t.Fatalf("event=%s", ev.Event)
		}
		if env.WorkItemID == "" {
			t.Fatal("missing work_item_id")
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for progress")
	}
}
