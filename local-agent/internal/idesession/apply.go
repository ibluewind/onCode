package idesession

import (
	"context"
	"encoding/json"
	"strings"

	"oncode/local-agent/internal/actualdiff"
	"oncode/local-agent/internal/idebridge"
	"oncode/local-agent/internal/workspace"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/ids"
	"oncode/protocol/messages"
)

// pendingApply는 서버 APPLYING_CHANGE를 받을 때까지 보관하는 승인된 제안이다.
// 거절이거나 apply 시도 후에는 비운다. 재시도하지 않는다.
type pendingApply struct {
	ChangeSetID string
	WorkspaceID string
	ApprovalID  string
	DiffHash    string
	WorkItemID  string
	Unchanged   bool
}

// boundApproval은 서버가 이미 소비한 승인과 로컬 제안이 같은지 검사한다.
// UUID v7이 아닌 서버 APR-/CHG- 형식도 받고, 불일치면 fail-closed다.
type boundApproval struct {
	approvalID  string
	changeSetID string
	diffHash    string
}

// Validate는 approval_id·change_set_id·diff_hash가 비어 있지 않고 바인딩과 같은지 본다.
func (b boundApproval) Validate(approvalID, changeSetID, approvedDiffHash string) error {
	if approvalID == "" || changeSetID == "" || approvedDiffHash == "" {
		return mustProto(protoerr.ApprovalRequired, "approval bind fields are required")
	}
	if approvalID != b.approvalID || changeSetID != b.changeSetID || approvedDiffHash != b.diffHash {
		return mustProto(protoerr.ApprovalInvalid, "approval bind does not match stored proposal")
	}
	return nil
}

// readOld는 actual diff용 현재 파일 본문이다. 없거나 루트 밖이면 빈 문자열이고 내용을 밖으로 새기지 않는다.
func (h *Hub) readOld(path string) string {
	if h == nil {
		return ""
	}
	if h.oldContent != nil {
		return h.oldContent(path)
	}
	if h.workspaces == nil || h.workspaceID == "" {
		return ""
	}
	ws, err := h.workspaces.Get(h.workspaceID)
	if err != nil {
		return ""
	}
	res, err := ws.ReadTextFile(path, h.maxBytes)
	if err != nil {
		return ""
	}
	return res.Content
}

// prepareStoredProposal은 등록 워크스페이스에서 제안을 만들고 저장한다.
// 서버 change_set_id가 프로토콜 ID가 아니면 새 CHG ID를 발급한다. apply는 하지 않는다.
func (h *Hub) prepareStoredProposal(csID string, changes []actualdiff.Change) (actualdiff.Result, error) {
	if h == nil || h.workspaces == nil || h.proposals == nil {
		return actualdiff.Result{}, mustProto(protoerr.InternalError, "workspace registry is required")
	}
	ws, err := h.workspaces.Get(h.workspaceID)
	if err != nil {
		return actualdiff.Result{}, err
	}
	if ids.Validate(ids.ChangeSet, csID) != nil {
		csID = ids.MustNew(ids.ChangeSet)
	}
	cs, err := changeSetFromProposed(ws, csID, changes, h.maxBytes)
	if err != nil {
		return actualdiff.Result{}, err
	}
	prop, err := ws.Propose(cs, h.maxBytes)
	if err != nil {
		return actualdiff.Result{}, err
	}
	h.proposals.Put(prop)
	return actualdiff.Result{
		ChangeSetID: prop.ChangeSet.ChangeSetID,
		WorkspaceID: prop.ChangeSet.WorkspaceID,
		Diff:        prop.ActualDiff.Diff,
		DiffHash:    prop.ActualDiff.DiffHash,
	}, nil
}

// changeSetFromProposed는 서버 제안 JSON을 디스크 해시가 채워진 ProposedChangeSet으로 바꾼다.
// MODIFY/DELETE/RENAME은 현재 파일이 있어야 하고, 경로 탈출은 ResolvePath가 거부한다.
func changeSetFromProposed(ws *workspace.Workspace, csID string, changes []actualdiff.Change, maxBytes int64) (messages.ProposedChangeSet, error) {
	out := messages.ProposedChangeSet{
		ChangeSetID: csID,
		WorkspaceID: ws.ID,
	}
	for _, ch := range changes {
		op := messages.ChangeOperation(strings.ToUpper(ch.Operation))
		if op == "" {
			op = messages.OpModify
		}
		fc := messages.FileChange{
			Path:       strings.TrimPrefix(ch.Path, "/"),
			Operation:  op,
			Content:    ch.Content,
			TargetPath: strings.TrimPrefix(ch.TargetPath, "/"),
		}
		if op == messages.OpModify || op == messages.OpDelete || op == messages.OpRename {
			res, err := ws.ReadTextFile(fc.Path, maxBytes)
			if err != nil {
				return messages.ProposedChangeSet{}, err
			}
			fc.BaseHash = res.Hash
		}
		out.Changes = append(out.Changes, fc)
	}
	return out, nil
}

// rememberApply는 바인딩된 코드 승인을 apply 대기열에 넣는다. 호출 시점에 디스크는 쓰지 않는다.
func (h *Hub) rememberApply(changeSetID, workspaceID, approvalID, workItemID, diffHash string, unchanged bool) {
	if h == nil || changeSetID == "" || workspaceID == "" || approvalID == "" || diffHash == "" {
		return
	}
	h.mu.Lock()
	h.pending = pendingApply{
		ChangeSetID: changeSetID,
		WorkspaceID: workspaceID,
		ApprovalID:  approvalID,
		DiffHash:    diffHash,
		WorkItemID:  workItemID,
		Unchanged:   unchanged,
	}
	h.mu.Unlock()
}

// applyIfAuthorized는 서버 progress stage를 본다. REJECTED면 대기를 지우고, APPLYING_CHANGE면 한 번만 apply한다.
func (h *Hub) applyIfAuthorized(data map[string]any) {
	if h == nil {
		return
	}
	switch text(data["stage"]) {
	case "REJECTED":
		h.clearPending()
		return
	case "APPLYING_CHANGE":
	default:
		return
	}
	pending, ok := h.takePending()
	if !ok {
		return
	}
	workItemID := pending.WorkItemID
	if workItemID == "" {
		workItemID = text(data["work_item_id"])
	}
	if pending.Unchanged {
		h.push(idebridge.Message{Type: "apply.result", Data: map[string]any{
			"status":        "SUCCESS",
			"change_set_id": pending.ChangeSetID,
			"workspace_id":  pending.WorkspaceID,
			"unchanged":     true,
			"message":       "no content change; workspace left as-is",
		}})
		h.notifyApplyCompleted(workItemID, pending.WorkspaceID, pending.ChangeSetID, true)
		return
	}
	if h.workspaces == nil || h.proposals == nil {
		h.push(idebridge.Message{Type: "error", Data: map[string]any{
			"code":    "UNAVAILABLE",
			"message": "workspace is not registered; apply skipped",
		}})
		return
	}
	result, err := workspace.ApplyChanges(
		h.workspaces,
		h.proposals,
		boundApproval{approvalID: pending.ApprovalID, changeSetID: pending.ChangeSetID, diffHash: pending.DiffHash},
		workspace.ApplyRequest{
			ChangeSetID:      pending.ChangeSetID,
			WorkspaceID:      pending.WorkspaceID,
			ApprovalID:       pending.ApprovalID,
			ApprovedDiffHash: pending.DiffHash,
		},
		workspace.ApplyOptions{MaxBytes: h.maxBytes},
	)
	if err != nil {
		h.push(idebridge.Message{Type: "error", Data: map[string]any{
			"code":    applyErrorCode(err),
			"message": err.Error(),
		}})
		return
	}
	h.push(idebridge.Message{Type: "apply.result", Data: map[string]any{
		"status":             "SUCCESS",
		"change_set_id":      result.ChangeSetID,
		"workspace_id":       result.WorkspaceID,
		"workspace_revision": result.WorkspaceRevision,
		"message":            "approved changes applied",
	}})
	h.notifyApplyCompleted(workItemID, result.WorkspaceID, result.ChangeSetID, false)
}

// notifyApplyCompleted는 서버에 apply.completed를 보내 COMPLETED 전이를 요청한다.
func (h *Hub) notifyApplyCompleted(workItemID, workspaceID, changeSetID string, unchanged bool) {
	if h == nil || h.stream == nil || workItemID == "" || workspaceID == "" {
		return
	}
	payload := map[string]any{
		"work_item_id":  workItemID,
		"workspace_id":  workspaceID,
		"change_set_id": changeSetID,
		"unchanged":     unchanged,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = h.stream.SendEvent(context.Background(), messages.EventApplyCompleted, raw)
}

// takePending은 대기 apply를 꺼내고 비운다. 없으면 ok=false. 부작용 재시도를 막는다.
func (h *Hub) takePending() (pendingApply, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pending.ChangeSetID == "" {
		return pendingApply{}, false
	}
	p := h.pending
	h.pending = pendingApply{}
	return p, true
}

// clearPending은 거절 등으로 apply 자격을 없앤다.
func (h *Hub) clearPending() {
	h.mu.Lock()
	h.pending = pendingApply{}
	h.mu.Unlock()
}

// applyErrorCode는 proto 오류면 그 코드를, 아니면 APPLY_FAILED를 쓴다.
func applyErrorCode(err error) string {
	if pe, ok := err.(*protoerr.Error); ok && pe.Code != "" {
		return string(pe.Code)
	}
	return string(protoerr.ApplyFailed)
}

func mustProto(code protoerr.Code, msg string) error {
	e, err := protoerr.New(code, msg)
	if err != nil {
		return err
	}
	return e
}
