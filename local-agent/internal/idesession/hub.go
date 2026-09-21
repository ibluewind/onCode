// Package idesession은 IDE WebSocket 메시지와 서버 gRPC 이벤트를 중계한다.
// 워크스페이스 읽기·apply는 등록된 Local Agent workspace만 쓰고, 승인 결정만으로 apply하지 않는다.
package idesession

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync"

	"oncode/local-agent/internal/actualdiff"
	"oncode/local-agent/internal/chatctx"
	"oncode/local-agent/internal/idebridge"
	"oncode/local-agent/internal/workspace"
	"oncode/protocol/messages"
)

// Stream is the subset of gRPC transport the IDE hub needs.
type Stream interface {
	SendEvent(ctx context.Context, event string, data json.RawMessage) error
	Events() <-chan *messages.Envelope
}

// Hub는 IDE WebSocket 메시지와 서버 gRPC 이벤트를 중계한다.
// 서버 연결이 없으면 chat.submit은 오류다. change.proposed는 actual diff로 바꾼다.
// 등록된 workspace가 있으면 원문을 읽고, 서버가 APPLYING_CHANGE를 보낸 뒤에만 apply한다.
type Hub struct {
	stream      Stream
	push        func(idebridge.Message)
	workspaceID string
	oldContent  func(path string) string
	workspaces  *workspace.Registry
	proposals   *workspace.ProposalStore
	maxBytes    int64

	mu             sync.Mutex
	lastWorkItemID string
	lastBind       map[string]any
	pending        pendingApply
}

// Options는 Hub 구성이다. Workspaces가 nil이면 actual diff는 OldContent(없으면 빈 원본)로 만들고 apply는 없다.
type Options struct {
	Stream       Stream
	Push         func(idebridge.Message)
	WorkspaceID  string
	OldContent   func(path string) string
	Workspaces   *workspace.Registry
	Proposals    *workspace.ProposalStore
	MaxFileBytes int64
}

// New는 gRPC 스트림과 IDE broadcast 함수를 묶는다. stream이 nil이면 채팅을 거절한다.
func New(stream Stream, push func(idebridge.Message)) *Hub {
	return NewWithOptions(Options{Stream: stream, Push: push, WorkspaceID: "ws-local"})
}

// NewWithOptions는 등록 워크스페이스와 제안 저장소를 주입한다. MaxFileBytes가 0이면 읽기 크기 제한이 없다.
func NewWithOptions(opts Options) *Hub {
	ws := opts.WorkspaceID
	if ws == "" {
		ws = "ws-local"
	}
	max := opts.MaxFileBytes
	if max < 0 {
		max = 0
	}
	return &Hub{
		stream:      liveStream(opts.Stream),
		push:        opts.Push,
		workspaceID: ws,
		oldContent:  opts.OldContent,
		workspaces:  opts.Workspaces,
		proposals:   opts.Proposals,
		maxBytes:    max,
	}
}

// liveStream은 인터페이스에 담긴 typed-nil 포인터를 진짜 nil로 만든다.
// nil *transport.Stream을 Stream에 넣으면 h.stream == nil이 false가 되어 Events()에서 패닉 난다.
func liveStream(s Stream) Stream {
	if s == nil {
		return nil
	}
	v := reflect.ValueOf(s)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		if v.IsNil() {
			return nil
		}
	}
	return s
}

// Handle은 등록 이후 IDE 프레임을 처리한다. 동기 응답이 있으면 반환한다.
func (h *Hub) Handle(_ string, msg idebridge.Message) *idebridge.Message {
	switch msg.Type {
	case "chat.submit":
		return h.submitChat(msg)
	case "question.answer":
		return h.forwardAnswer(msg)
	case "approval.decide":
		return h.forwardDecide(msg)
	case "session.restore":
		return h.restore(msg)
	default:
		return &idebridge.Message{
			Type:      "error",
			MessageID: msg.MessageID,
			Data:      map[string]any{"code": "UNSUPPORTED", "message": "unsupported type " + msg.Type},
		}
	}
}

// Listen은 서버 EVENT를 IDE JSON으로 바꿔 push한다. ctx가 끝나면 반환한다.
func (h *Hub) Listen(ctx context.Context) {
	if h == nil || h.stream == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case env, ok := <-h.stream.Events():
			if !ok {
				return
			}
			h.forwardServerEvent(env)
		}
	}
}

func (h *Hub) submitChat(msg idebridge.Message) *idebridge.Message {
	if err := chatctx.ValidateSubmit(msg.Data); err != nil {
		return errorMsg(msg.MessageID, "INVALID_REQUEST", err.Error())
	}
	if h.stream == nil {
		return errorMsg(msg.MessageID, "UNAVAILABLE", "central server is not connected")
	}
	data := cloneMap(msg.Data)
	if _, ok := data["workspace_id"]; !ok {
		data["workspace_id"] = h.workspaceID
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return errorMsg(msg.MessageID, "INVALID_REQUEST", err.Error())
	}
	if err := h.stream.SendEvent(context.Background(), messages.EventChatSubmit, raw); err != nil {
		return errorMsg(msg.MessageID, "UNAVAILABLE", err.Error())
	}
	return &idebridge.Message{
		Type:      "chat.accepted",
		MessageID: msg.MessageID,
		Data:      map[string]any{"ok": true},
	}
}

func (h *Hub) forwardAnswer(msg idebridge.Message) *idebridge.Message {
	if h.stream == nil {
		return errorMsg(msg.MessageID, "UNAVAILABLE", "central server is not connected")
	}
	raw, err := json.Marshal(msg.Data)
	if err != nil {
		return errorMsg(msg.MessageID, "INVALID_REQUEST", err.Error())
	}
	if err := h.stream.SendEvent(context.Background(), messages.EventQuestionAnswer, raw); err != nil {
		return errorMsg(msg.MessageID, "UNAVAILABLE", err.Error())
	}
	return nil
}

func (h *Hub) forwardDecide(msg idebridge.Message) *idebridge.Message {
	if h.stream == nil {
		return errorMsg(msg.MessageID, "UNAVAILABLE", "central server is not connected")
	}
	data := cloneMap(msg.Data)
	h.mu.Lock()
	for k, v := range h.lastBind {
		if _, ok := data[k]; !ok {
			data[k] = v
		}
	}
	h.mu.Unlock()
	if _, ok := data["workspace_id"]; !ok {
		data["workspace_id"] = h.workspaceID
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return errorMsg(msg.MessageID, "INVALID_REQUEST", err.Error())
	}
	if err := h.stream.SendEvent(context.Background(), messages.EventApprovalDecide, raw); err != nil {
		return errorMsg(msg.MessageID, "UNAVAILABLE", err.Error())
	}
	return nil
}

func (h *Hub) restore(msg idebridge.Message) *idebridge.Message {
	if h.stream == nil {
		return errorMsg(msg.MessageID, "UNAVAILABLE", "central server is not connected")
	}
	data := cloneMap(msg.Data)
	h.mu.Lock()
	if text(data["work_item_id"]) == "" && h.lastWorkItemID != "" {
		data["work_item_id"] = h.lastWorkItemID
	}
	h.mu.Unlock()
	if text(data["work_item_id"]) == "" {
		return nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return errorMsg(msg.MessageID, "INVALID_REQUEST", err.Error())
	}
	if err := h.stream.SendEvent(context.Background(), messages.EventSessionRestore, raw); err != nil {
		return errorMsg(msg.MessageID, "UNAVAILABLE", err.Error())
	}
	return nil
}

func (h *Hub) forwardServerEvent(env *messages.Envelope) {
	if env == nil || h.push == nil {
		return
	}
	var ev messages.EventPayload
	if err := json.Unmarshal(env.Payload, &ev); err != nil {
		return
	}
	data := map[string]any{}
	if len(ev.Data) > 0 {
		_ = json.Unmarshal(ev.Data, &data)
	}
	if env.WorkItemID != "" {
		data["work_item_id"] = env.WorkItemID
		h.mu.Lock()
		h.lastWorkItemID = env.WorkItemID
		h.mu.Unlock()
	}
	if env.WorkflowID != "" {
		data["workflow_id"] = env.WorkflowID
	}
	typ := ev.Event
	switch ev.Event {
	case messages.EventWorkflowProgress:
		typ = "workflow.progress"
	case messages.EventQuestionAsk:
		typ = "question.ask"
	case messages.EventError:
		typ = "error"
	case messages.EventDesignReview:
		typ = "design.review"
		h.rememberBind(data, false)
		data["can_apply"] = false
	case messages.EventChangeProposed:
		h.pushDiffReview(data)
		return
	case messages.EventBuildResult:
		typ = "build.result"
	case messages.EventTestResult:
		typ = "test.result"
	}
	h.push(idebridge.Message{Type: typ, Data: data})
	if ev.Event == messages.EventWorkflowProgress {
		h.applyIfAuthorized(data)
		if text(data["stage"]) == "COMPLETED" {
			h.mu.Lock()
			h.lastWorkItemID = ""
			h.lastBind = nil
			h.mu.Unlock()
			h.clearPending()
		}
	}
}

func (h *Hub) pushDiffReview(data map[string]any) {
	raw := text(data["proposed_changes_json"])
	csID, changes, err := actualdiff.ParseProposedJSON(raw)
	if err != nil {
		h.push(idebridge.Message{Type: "error", Data: map[string]any{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	if csID == "" {
		csID = "CS-unknown"
	}
	ws := text(data["workspace_id"])
	if ws == "" {
		ws = h.workspaceID
	}
	var result actualdiff.Result
	if h.workspaces != nil && h.proposals != nil {
		result, err = h.prepareStoredProposal(csID, changes)
	} else {
		result, err = actualdiff.FromProposed(csID, ws, changes, h.readOld)
	}
	if err != nil {
		h.push(idebridge.Message{Type: "error", Data: map[string]any{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	view := cloneMap(data)
	delete(view, "proposed_changes_json")
	view["change_set_id"] = result.ChangeSetID
	view["actual_diff"] = result.Diff
	view["diff_hash"] = result.DiffHash
	view["workspace_id"] = result.WorkspaceID
	view["can_apply"] = false
	unchanged := !diffHasContentChange(result.Diff)
	view["unchanged"] = unchanged
	if unchanged {
		view["message"] = "제안 내용이 현재 파일과 같습니다. 승인해도 디스크는 바뀌지 않습니다."
	}
	h.rememberBind(view, true)
	if h.stream != nil {
		bind := map[string]any{
			"approval_id":  view["approval_id"],
			"work_item_id": view["work_item_id"],
			"workspace_id": view["workspace_id"],
			"resource_id":  view["resource_id"],
			"diff_hash":    result.DiffHash,
		}
		rawBind, _ := json.Marshal(bind)
		_ = h.stream.SendEvent(context.Background(), messages.EventApprovalBind, rawBind)
	}
	if h.workspaces != nil && h.proposals != nil {
		h.rememberApply(
			result.ChangeSetID,
			result.WorkspaceID,
			text(view["approval_id"]),
			text(view["work_item_id"]),
			result.DiffHash,
			unchanged,
		)
	}
	h.push(idebridge.Message{Type: "diff.review", Data: view})
}

// diffHasContentChange는 unified diff에 추가/삭제 줄이 있는지 본다. 컨텍스트만 있으면 false다.
func diffHasContentChange(diffText string) bool {
	for _, line := range strings.Split(diffText, "\n") {
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			return true
		}
	}
	return false
}

func (h *Hub) rememberBind(data map[string]any, withDiff bool) {
	bind := map[string]any{
		"approval_id":     data["approval_id"],
		"work_item_id":    data["work_item_id"],
		"workspace_id":    data["workspace_id"],
		"resource_id":     data["resource_id"],
		"design_identity": data["design_identity"],
	}
	if withDiff {
		bind["diff_hash"] = data["diff_hash"]
	}
	h.mu.Lock()
	h.lastBind = bind
	if id := text(data["work_item_id"]); id != "" {
		h.lastWorkItemID = id
	}
	h.mu.Unlock()
}

func cloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func text(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func errorMsg(id, code, message string) *idebridge.Message {
	return &idebridge.Message{
		Type:      "error",
		MessageID: id,
		Data:      map[string]any{"code": code, "message": message},
	}
}
