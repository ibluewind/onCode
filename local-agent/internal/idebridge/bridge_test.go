package idebridge

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestValidateBind_RejectsNonLoopback(t *testing.T) {
	t.Parallel()
	rejects := []string{"", "0.0.0.0", "::", "192.168.1.10", "10.0.0.1", "localhost", "example.com"}
	for _, host := range rejects {
		if err := ValidateBind(host); err == nil {
			t.Fatalf("expected reject for bind %q", host)
		}
	}
	for _, host := range []string{"127.0.0.1", "::1"} {
		if err := ValidateBind(host); err != nil {
			t.Fatalf("loopback %q: %v", host, err)
		}
	}
}

func TestNew_RefusesNonLoopbackBind(t *testing.T) {
	t.Parallel()
	_, err := New(Options{Bind: "0.0.0.0", Port: 0, StateDir: t.TempDir(), AgentID: "agent-test"})
	if err == nil {
		t.Fatal("expected error for 0.0.0.0 bind")
	}
}

func TestBridge_MissingAndInvalidTokenRejected(t *testing.T) {
	b := startTestBridge(t)
	ep, err := b.Endpoint()
	if err != nil {
		t.Fatal(err)
	}
	base := "ws://127.0.0.1:" + strconv.Itoa(ep.Port) + ipcPath

	if _, resp, err := websocket.DefaultDialer.Dial(base, nil); err == nil {
		t.Fatal("missing token should fail")
	} else if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing token status=%v err=%v", statusOf(resp), err)
	}

	if _, resp, err := websocket.DefaultDialer.Dial(base+"?token=deadbeef", nil); err == nil {
		t.Fatal("invalid token should fail")
	} else if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid token status=%v err=%v", statusOf(resp), err)
	}

	conn := mustDial(t, base+"?token="+ep.Token, nil)
	defer conn.Close()
}

func TestBridge_ForbiddenOriginRejected(t *testing.T) {
	b := startTestBridge(t)
	ep, err := b.Endpoint()
	if err != nil {
		t.Fatal(err)
	}
	url := "ws://127.0.0.1:" + strconv.Itoa(ep.Port) + ipcPath + "?token=" + ep.Token
	hdr := http.Header{}
	hdr.Set("Origin", "https://evil.example")
	if _, resp, err := websocket.DefaultDialer.Dial(url, hdr); err == nil {
		t.Fatal("foreign origin should fail")
	} else if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("origin status=%v err=%v", statusOf(resp), err)
	}
}

func TestBridge_RegisterAndReconnect(t *testing.T) {
	b := startTestBridge(t)
	ep, err := b.Endpoint()
	if err != nil {
		t.Fatal(err)
	}
	url := "ws://127.0.0.1:" + strconv.Itoa(ep.Port) + ipcPath + "?token=" + ep.Token

	firstID := register(t, url, "VSCODE")
	secondID := register(t, url, "VSCODE")
	if firstID == "" || secondID == "" {
		t.Fatal("empty ide_client_id")
	}
	if firstID == secondID {
		t.Fatalf("reconnect should allocate a new ide_client_id, got %s", firstID)
	}
}

func TestBridge_TokenRotatesOnRestart(t *testing.T) {
	stateDir := t.TempDir()
	ctx1, cancel1 := context.WithCancel(context.Background())
	t.Cleanup(cancel1)
	b1, err := New(Options{Bind: "127.0.0.1", Port: 0, StateDir: stateDir, AgentID: "agent-a"})
	if err != nil {
		t.Fatal(err)
	}
	if err := b1.Start(ctx1); err != nil {
		t.Fatal(err)
	}
	old, err := b1.Endpoint()
	if err != nil {
		t.Fatal(err)
	}
	if err := b1.Close(); err != nil {
		t.Fatal(err)
	}

	ctx2, cancel2 := context.WithCancel(context.Background())
	t.Cleanup(cancel2)
	b2, err := New(Options{Bind: "127.0.0.1", Port: 0, StateDir: stateDir, AgentID: "agent-a"})
	if err != nil {
		t.Fatal(err)
	}
	if err := b2.Start(ctx2); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b2.Close() })
	fresh, err := b2.Endpoint()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Token == old.Token {
		t.Fatal("token must rotate on restart")
	}

	oldURL := "ws://127.0.0.1:" + strconv.Itoa(fresh.Port) + ipcPath + "?token=" + old.Token
	if _, resp, err := websocket.DefaultDialer.Dial(oldURL, nil); err == nil {
		t.Fatal("rotated-away token should fail")
	} else if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old token status=%v err=%v", statusOf(resp), err)
	}
	register(t, "ws://127.0.0.1:"+strconv.Itoa(fresh.Port)+ipcPath+"?token="+fresh.Token, "VSCODE")
}

func TestReadEndpoint_WrittenAtStartAndRemovedOnClose(t *testing.T) {
	b := startTestBridge(t)
	ep, err := ReadEndpoint(b.stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if ep.Bind != "127.0.0.1" || ep.Port <= 0 || ep.Token == "" || ep.Path != ipcPath {
		t.Fatalf("%+v", ep)
	}
	info, err := os.Stat(filepath.Join(b.stateDir, EndpointFileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 && runtime.GOOS != "windows" {
		t.Fatalf("endpoint file should not be group/world readable, mode=%v", info.Mode())
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadEndpoint(b.stateDir); err == nil {
		t.Fatal("endpoint file should be removed on close")
	}
}

func startTestBridge(t *testing.T) *Bridge {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	b, err := New(Options{Bind: "127.0.0.1", Port: 0, StateDir: t.TempDir(), AgentID: "agent-test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })
	return b
}

func register(t *testing.T, url, ide string) string {
	t.Helper()
	conn := mustDial(t, url, nil)
	defer conn.Close()
	if err := conn.WriteJSON(Message{
		Type:      typeRegister,
		MessageID: "msg-1",
		Data:      map[string]any{"ide": ide, "extension_version": "0.1.0"},
	}); err != nil {
		t.Fatal(err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var reply Message
	if err := conn.ReadJSON(&reply); err != nil {
		t.Fatal(err)
	}
	if reply.Type != typeRegistered {
		raw, _ := json.Marshal(reply)
		t.Fatalf("want ide.registered, got %s", raw)
	}
	id, _ := reply.Data["ide_client_id"].(string)
	agentID, _ := reply.Data["agent_id"].(string)
	if id == "" || agentID == "" {
		t.Fatalf("incomplete registered payload: %+v", reply.Data)
	}
	return id
}

func mustDial(t *testing.T, url string, hdr http.Header) *websocket.Conn {
	t.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial(url, hdr)
	if err != nil {
		t.Fatalf("dial: status=%v err=%v", statusOf(resp), err)
	}
	return conn
}

func statusOf(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}
