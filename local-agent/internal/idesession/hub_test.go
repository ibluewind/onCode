package idesession

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oncode/local-agent/internal/idebridge"
	"oncode/local-agent/internal/workspace"
	"oncode/protocol/messages"
)

type fakeStream struct {
	sent []string
	ch   chan *messages.Envelope
}

func (f *fakeStream) SendEvent(_ context.Context, event string, _ json.RawMessage) error {
	f.sent = append(f.sent, event)
	return nil
}

func (f *fakeStream) Events() <-chan *messages.Envelope { return f.ch }

func TestHub_ListenTypedNilStreamDoesNotPanic(t *testing.T) {
	var typedNil *fakeStream
	h := New(typedNil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h.Listen(ctx)
	if h.stream != nil {
		t.Fatal("typed-nil stream must be stored as nil")
	}
}

func TestHub_RejectsSourceBody(t *testing.T) {
	h := New(nil, nil)
	reply := h.Handle("ide-1", idebridge.Message{
		Type: "chat.submit",
		Data: map[string]any{
			"message": "x",
			"content": "class A {}",
		},
	})
	if reply == nil || reply.Type != "error" {
		t.Fatalf("%+v", reply)
	}
}

func TestHub_ChatSubmitRequiresServer(t *testing.T) {
	h := New(nil, nil)
	reply := h.Handle("ide-1", idebridge.Message{
		Type: "chat.submit",
		Data: map[string]any{"message": "hello", "ide_context": map[string]any{"current_file": "A.java"}},
	})
	if reply == nil || reply.Data["code"] != "UNAVAILABLE" {
		t.Fatalf("%+v", reply)
	}
}

func TestHub_ChatSubmitForwardsEvent(t *testing.T) {
	fs := &fakeStream{ch: make(chan *messages.Envelope)}
	h := New(fs, nil)
	reply := h.Handle("ide-1", idebridge.Message{
		Type: "chat.submit",
		Data: map[string]any{"message": "hello", "ide_context": map[string]any{"current_file": "A.java"}},
	})
	if reply == nil || reply.Type != "chat.accepted" {
		t.Fatalf("%+v", reply)
	}
	if len(fs.sent) != 1 || fs.sent[0] != messages.EventChatSubmit {
		t.Fatalf("%v", fs.sent)
	}
}

func TestHub_ChangeProposedBecomesActualDiff(t *testing.T) {
	fs := &fakeStream{ch: make(chan *messages.Envelope, 1)}
	var pushed []idebridge.Message
	h := NewWithOptions(Options{
		Stream:      fs,
		Push:        func(m idebridge.Message) { pushed = append(pushed, m) },
		WorkspaceID: "ws-local",
		OldContent:  func(string) string { return "class AuthService {}" },
	})
	proposed := `{"change_set_id":"CS-1","changes":[{"path":"src/AuthService.java","operation":"MODIFY","content":"class AuthService { /* lock */ }"}]}`
	env, err := messages.NewEnvelope(messages.TypeEvent, messages.EventPayload{
		Event: messages.EventChangeProposed,
		Data: mustJSON(map[string]any{
			"approval_id":           "APR-1",
			"work_item_id":          "WI-1",
			"workspace_id":          "ws-local",
			"resource_id":           "cs://work-items/WI-1/latest",
			"proposed_changes_json": proposed,
			"can_apply":             false,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	h.forwardServerEvent(env)
	if len(pushed) != 1 || pushed[0].Type != "diff.review" {
		t.Fatalf("%+v", pushed)
	}
	diffText, _ := pushed[0].Data["actual_diff"].(string)
	if !strings.Contains(diffText, "--- a/src/AuthService.java") {
		t.Fatalf("actual diff missing: %v", pushed[0].Data)
	}
	if _, ok := pushed[0].Data["proposed_changes_json"]; ok {
		t.Fatal("must not forward proposal JSON to IDE")
	}
	if pushed[0].Data["can_apply"] != false {
		t.Fatal("approve must not apply")
	}
	if len(fs.sent) != 1 || fs.sent[0] != messages.EventApprovalBind {
		t.Fatalf("bind not sent: %v", fs.sent)
	}
}

func TestHub_SessionRestoreUsesLastWorkItem(t *testing.T) {
	fs := &fakeStream{ch: make(chan *messages.Envelope)}
	h := New(fs, func(idebridge.Message) {})
	h.lastWorkItemID = "WI-restore"
	reply := h.Handle("ide-1", idebridge.Message{Type: "session.restore", Data: map[string]any{}})
	if reply != nil {
		t.Fatalf("%+v", reply)
	}
	if len(fs.sent) != 1 || fs.sent[0] != messages.EventSessionRestore {
		t.Fatalf("%v", fs.sent)
	}
}

func TestHub_DesignReviewHasNoApply(t *testing.T) {
	var pushed []idebridge.Message
	h := New(&fakeStream{ch: make(chan *messages.Envelope)}, func(m idebridge.Message) { pushed = append(pushed, m) })
	env, err := messages.NewEnvelope(messages.TypeEvent, messages.EventPayload{
		Event: messages.EventDesignReview,
		Data: mustJSON(map[string]any{
			"approval_id":     "APR-d",
			"design_summary":  "lock account",
			"design_identity": "sha256:abc",
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	h.forwardServerEvent(env)
	if len(pushed) != 1 || pushed[0].Type != "design.review" {
		t.Fatalf("%+v", pushed)
	}
	if pushed[0].Data["can_apply"] != false {
		t.Fatal("design review must not offer apply")
	}
	if _, ok := pushed[0].Data["actual_diff"]; ok {
		t.Fatal("design review must not include diff")
	}
}

func TestHub_RegisteredWorkspaceFeedsActualDiff(t *testing.T) {
	fs := &fakeStream{ch: make(chan *messages.Envelope, 1)}
	var pushed []idebridge.Message
	h, root := newWorkspaceHub(t, fs, func(m idebridge.Message) { pushed = append(pushed, m) })
	mustWrite(t, filepath.Join(root, "src", "AuthService.java"), "class AuthService {}")
	h.forwardServerEvent(mustChangeProposed("APR-bind-1"))
	if len(pushed) != 1 || pushed[0].Type != "diff.review" {
		t.Fatalf("%+v", pushed)
	}
	diffText, _ := pushed[0].Data["actual_diff"].(string)
	if !strings.Contains(diffText, "class AuthService {}") {
		t.Fatalf("disk original missing from actual diff: %s", diffText)
	}
	if !strings.Contains(diffText, "/* lock */") {
		t.Fatalf("proposed content missing: %s", diffText)
	}
	if pushed[0].Data["can_apply"] != false {
		t.Fatal("approve must not apply by itself")
	}
	if len(fs.sent) != 1 || fs.sent[0] != messages.EventApprovalBind {
		t.Fatalf("bind not sent: %v", fs.sent)
	}
}

func TestHub_PathEscapeRejected(t *testing.T) {
	fs := &fakeStream{ch: make(chan *messages.Envelope, 1)}
	var pushed []idebridge.Message
	h, _ := newWorkspaceHub(t, fs, func(m idebridge.Message) { pushed = append(pushed, m) })
	env, err := messages.NewEnvelope(messages.TypeEvent, messages.EventPayload{
		Event: messages.EventChangeProposed,
		Data: mustJSON(map[string]any{
			"approval_id":           "APR-esc",
			"work_item_id":          "WI-1",
			"resource_id":           "cs://work-items/WI-1/latest",
			"proposed_changes_json": `{"change_set_id":"CS-1","changes":[{"path":"../secret.txt","operation":"MODIFY","content":"x"}]}`,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	h.forwardServerEvent(env)
	if len(pushed) != 1 || pushed[0].Type != "error" {
		t.Fatalf("want error, got %+v", pushed)
	}
	if len(fs.sent) != 0 {
		t.Fatalf("must not bind escaped path: %v", fs.sent)
	}
}

func TestHub_ApplyingChangeAppliesOnce(t *testing.T) {
	fs := &fakeStream{ch: make(chan *messages.Envelope, 1)}
	var pushed []idebridge.Message
	h, root := newWorkspaceHub(t, fs, func(m idebridge.Message) { pushed = append(pushed, m) })
	target := filepath.Join(root, "src", "AuthService.java")
	mustWrite(t, target, "class AuthService {}")
	h.forwardServerEvent(mustChangeProposed("APR-apply-1"))
	h.forwardServerEvent(mustProgress("APPLYING_CHANGE"))
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "/* lock */") {
		t.Fatalf("apply did not write: %q", raw)
	}
	if !hasType(pushed, "apply.result") {
		t.Fatalf("missing apply.result: %+v", pushed)
	}
	if !hasSent(fs, messages.EventApplyCompleted) {
		t.Fatalf("missing apply.completed: %v", fs.sent)
	}
	pushed = pushed[:0]
	h.forwardServerEvent(mustProgress("APPLYING_CHANGE"))
	if hasType(pushed, "apply.result") {
		t.Fatal("second APPLYING_CHANGE must not apply again")
	}
}

func TestHub_UnchangedSkipsRewriteAndCompletes(t *testing.T) {
	fs := &fakeStream{ch: make(chan *messages.Envelope, 1)}
	var pushed []idebridge.Message
	h, root := newWorkspaceHub(t, fs, func(m idebridge.Message) { pushed = append(pushed, m) })
	target := filepath.Join(root, "src", "AuthService.java")
	same := "class AuthService { /* lock */ }"
	mustWrite(t, target, same)
	env, err := messages.NewEnvelope(messages.TypeEvent, messages.EventPayload{
		Event: messages.EventChangeProposed,
		Data: mustJSON(map[string]any{
			"approval_id":           "APR-same-1",
			"work_item_id":          "WI-1",
			"resource_id":           "cs://work-items/WI-1/latest",
			"proposed_changes_json": `{"change_set_id":"CS-same","changes":[{"path":"src/AuthService.java","operation":"MODIFY","content":"class AuthService { /* lock */ }"}]}`,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	h.forwardServerEvent(env)
	var review map[string]any
	for _, m := range pushed {
		if m.Type == "diff.review" {
			review = m.Data
		}
	}
	if review == nil || review["unchanged"] != true {
		t.Fatalf("want unchanged diff.review, got %+v", pushed)
	}
	pushed = pushed[:0]
	fs.sent = nil
	h.forwardServerEvent(mustProgress("APPLYING_CHANGE"))
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != same {
		t.Fatalf("unchanged apply must not rewrite: %q", raw)
	}
	if !hasType(pushed, "apply.result") {
		t.Fatalf("missing apply.result: %+v", pushed)
	}
	if !hasSent(fs, messages.EventApplyCompleted) {
		t.Fatalf("missing apply.completed: %v", fs.sent)
	}
}

func TestDiffHasContentChange(t *testing.T) {
	if diffHasContentChange("--- a/x\n+++ b/x\n@@\n context\n") {
		t.Fatal("context-only must be unchanged")
	}
	if !diffHasContentChange("--- a/x\n+++ b/x\n@@\n-old\n+new\n") {
		t.Fatal("want content change")
	}
}

func TestHub_RejectDoesNotApply(t *testing.T) {
	fs := &fakeStream{ch: make(chan *messages.Envelope, 1)}
	h, root := newWorkspaceHub(t, fs, func(idebridge.Message) {})
	target := filepath.Join(root, "src", "AuthService.java")
	mustWrite(t, target, "class AuthService {}")
	h.forwardServerEvent(mustChangeProposed("APR-rej-1"))
	h.forwardServerEvent(mustProgress("REJECTED"))
	h.forwardServerEvent(mustProgress("APPLYING_CHANGE"))
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "class AuthService {}" {
		t.Fatalf("reject must leave workspace: %q", raw)
	}
}

func TestHub_StaleWorkspaceDoesNotApply(t *testing.T) {
	fs := &fakeStream{ch: make(chan *messages.Envelope, 1)}
	var pushed []idebridge.Message
	h, root := newWorkspaceHub(t, fs, func(m idebridge.Message) { pushed = append(pushed, m) })
	target := filepath.Join(root, "src", "AuthService.java")
	mustWrite(t, target, "class AuthService {}")
	h.forwardServerEvent(mustChangeProposed("APR-stale-1"))
	mustWrite(t, target, "class AuthService { /* edited */ }")
	h.forwardServerEvent(mustProgress("APPLYING_CHANGE"))
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "class AuthService { /* edited */ }" {
		t.Fatalf("stale apply must not overwrite: %q", raw)
	}
	if !hasType(pushed, "error") {
		t.Fatalf("want stale error, got %+v", pushed)
	}
}

// newWorkspaceHub는 임시 루트를 등록한 Hub를 만든다. 테스트 전용이며 서버 프로세스는 없다.
func newWorkspaceHub(t *testing.T, fs *fakeStream, push func(idebridge.Message)) (*Hub, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	return NewWithOptions(Options{
		Stream:       fs,
		Push:         push,
		WorkspaceID:  ws.ID,
		Workspaces:   reg,
		Proposals:    workspace.NewProposalStore(),
		MaxFileBytes: 4096,
	}), root
}

// mustChangeProposed는 스크립트 하니스와 같은 AuthService MODIFY 제안을 만든다.
func mustChangeProposed(approvalID string) *messages.Envelope {
	env, err := messages.NewEnvelope(messages.TypeEvent, messages.EventPayload{
		Event: messages.EventChangeProposed,
		Data: mustJSON(map[string]any{
			"approval_id":           approvalID,
			"work_item_id":          "WI-1",
			"resource_id":           "cs://work-items/WI-1/latest",
			"proposed_changes_json": `{"change_set_id":"CS-1","changes":[{"path":"src/AuthService.java","operation":"MODIFY","content":"class AuthService { /* lock */ }"}]}`,
		}),
	})
	if err != nil {
		panic(err)
	}
	return env
}

// mustProgress는 workflow.progress Envelope를 만든다. stage는 공식 상태 이름이다.
func mustProgress(stage string) *messages.Envelope {
	env, err := messages.NewEnvelope(messages.TypeEvent, messages.EventPayload{
		Event: messages.EventWorkflowProgress,
		Data:  mustJSON(map[string]any{"stage": stage, "status": "RUNNING", "message": stage}),
	})
	if err != nil {
		panic(err)
	}
	return env
}

// hasType은 푸시된 IDE 메시지에 type이 있는지 본다.
func hasType(msgs []idebridge.Message, typ string) bool {
	for _, m := range msgs {
		if m.Type == typ {
			return true
		}
	}
	return false
}

// hasSent는 서버로 보낸 EVENT 이름이 있는지 본다.
func hasSent(fs *fakeStream, event string) bool {
	for _, e := range fs.sent {
		if e == event {
			return true
		}
	}
	return false
}

// mustWrite는 테스트 워크스페이스 파일을 만든다. 실패하면 테스트를 멈춘다.
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}
