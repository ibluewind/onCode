package messages

import (
	"encoding/json"
	"fmt"
	"time"

	protoerr "oncode/protocol/errors"
	"oncode/protocol/ids"
	"oncode/protocol/versioning"
)

const (
	TypeRequest  = "REQUEST"
	TypeResponse = "RESPONSE"
	TypeEvent    = "EVENT"
	TypeCancel   = "CANCEL"
	TypeAck      = "ACK"
)

type ActorType string

const (
	ActorUser         ActorType = "USER"
	ActorAgent        ActorType = "AGENT"
	ActorOrchestrator ActorType = "ORCHESTRATOR"
	ActorLocalAgent   ActorType = "LOCAL_AGENT"
	ActorIDE          ActorType = "IDE"
	ActorSystem       ActorType = "SYSTEM"
)

type TargetType string

const (
	TargetTool TargetType = "TOOL"
)

type Actor struct {
	Type ActorType `json:"type"`
	ID   string    `json:"id"`
}

type Target struct {
	Type TargetType `json:"type"`
	ID   string     `json:"id"`
}

type Envelope struct {
	Protocol    string          `json:"protocol"`
	MessageType string          `json:"message_type"`
	MessageID   string          `json:"message_id"`
	Timestamp   string          `json:"timestamp"`
	SessionID   string          `json:"session_id,omitempty"`
	ProjectID   string          `json:"project_id,omitempty"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	WorkflowID  string          `json:"workflow_id,omitempty"`
	WorkItemID  string          `json:"work_item_id,omitempty"`
	TaskID      string          `json:"task_id,omitempty"`
	Actor       *Actor          `json:"actor,omitempty"`
	Target      *Target         `json:"target,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
}

func NewEnvelope(messageType string, payload any) (*Envelope, error) {
	mid, err := ids.New(ids.Message)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Envelope{
		Protocol:    versioning.Current,
		MessageType: messageType,
		MessageID:   mid,
		Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
		Payload:     raw,
	}, nil
}

func (e *Envelope) Validate() error {
	if e == nil {
		return fmt.Errorf("nil envelope")
	}
	if _, err := versioning.ParseCompatible(e.Protocol); err != nil {
		return fmt.Errorf("%s: %w", protoerr.ProtocolVersionUnsupported, err)
	}
	switch e.MessageType {
	case TypeRequest, TypeResponse, TypeEvent, TypeCancel, TypeAck:
	default:
		return fmt.Errorf("invalid message_type %q", e.MessageType)
	}
	if err := ids.Validate(ids.Message, e.MessageID); err != nil {
		return err
	}
	if e.Timestamp == "" {
		return fmt.Errorf("timestamp required")
	}
	if e.SessionID != "" {
		if err := ids.Validate(ids.Session, e.SessionID); err != nil {
			return err
		}
	}
	if e.ProjectID != "" {
		if err := ids.Validate(ids.Project, e.ProjectID); err != nil {
			return err
		}
	}
	if e.WorkspaceID != "" {
		if err := ids.Validate(ids.Workspace, e.WorkspaceID); err != nil {
			return err
		}
	}
	if e.WorkflowID != "" {
		if err := ids.Validate(ids.Workflow, e.WorkflowID); err != nil {
			return err
		}
	}
	if e.WorkItemID != "" {
		if err := ids.Validate(ids.WorkItem, e.WorkItemID); err != nil {
			return err
		}
	}
	if e.TaskID != "" {
		if err := ids.Validate(ids.Task, e.TaskID); err != nil {
			return err
		}
	}
	return nil
}

const (
	StatusSuccess   = "SUCCESS"
	StatusFailed    = "FAILED"
	StatusPartial   = "PARTIAL"
	StatusAccepted  = "ACCEPTED"
	StatusRunning   = "RUNNING"
	StatusCancelled = "CANCELLED"
	StatusRejected  = "REJECTED"
	StatusTimeout   = "TIMEOUT"
)

type ToolOptions struct {
	TimeoutMS int `json:"timeout_ms,omitempty"`
}

type ToolRequestPayload struct {
	CallID    string         `json:"call_id"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
	Options   *ToolOptions   `json:"options,omitempty"`
}

func (p *ToolRequestPayload) Validate() error {
	if err := ids.Validate(ids.Call, p.CallID); err != nil {
		return err
	}
	if p.Tool == "" {
		return fmt.Errorf("tool required")
	}
	if p.Arguments == nil {
		p.Arguments = map[string]any{}
	}
	return nil
}

type ToolResponsePayload struct {
	CallID  string          `json:"call_id"`
	Status  string          `json:"status"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *protoerr.Error `json:"error,omitempty"`
	Metrics *Metrics        `json:"metrics,omitempty"`
}

type Metrics struct {
	DurationMS int64 `json:"duration_ms,omitempty"`
}

func (p *ToolResponsePayload) Validate() error {
	if err := ids.Validate(ids.Call, p.CallID); err != nil {
		return err
	}
	switch p.Status {
	case StatusSuccess, StatusFailed, StatusPartial, StatusAccepted,
		StatusRunning, StatusCancelled, StatusRejected, StatusTimeout:
	default:
		return fmt.Errorf("invalid status %q", p.Status)
	}
	if p.Status == StatusFailed {
		if p.Error == nil {
			return fmt.Errorf("error required when status is FAILED")
		}
		if !protoerr.Known(p.Error.Code) {
			return fmt.Errorf("unknown error code %q", p.Error.Code)
		}
	}
	return nil
}

type EventPayload struct {
	Event       string          `json:"event"`
	ExecutionID string          `json:"execution_id,omitempty"`
	Data        json.RawMessage `json:"data,omitempty"`
}

type HeartbeatPayload struct {
	AgentID         string `json:"agent_id"`
	ProtocolVersion string `json:"protocol_version"`
	Timestamp       string `json:"timestamp"`
}

func (h *HeartbeatPayload) Validate() error {
	if h.AgentID == "" {
		return fmt.Errorf("agent_id required")
	}
	if _, err := versioning.ParseCompatible(h.ProtocolVersion); err != nil {
		return err
	}
	if h.Timestamp == "" {
		return fmt.Errorf("timestamp required")
	}
	return nil
}

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type Capabilities struct {
	Workspace    bool     `json:"workspace"`
	ProjectIndex bool     `json:"project_index"`
	Build        []string `json:"build,omitempty"`
	Test         []string `json:"test,omitempty"`
	Git          bool     `json:"git"`
	Languages    []string `json:"languages,omitempty"`
}

type CapabilitiesResult struct {
	AgentVersion string       `json:"agent_version"`
	Platform     Platform     `json:"platform"`
	Capabilities Capabilities `json:"capabilities"`
}

type ApprovalDecision string

const (
	DecisionApprove ApprovalDecision = "APPROVE"
	DecisionReject  ApprovalDecision = "REJECT"
)

type Approval struct {
	ApprovalID string           `json:"approval_id"`
	Type       string           `json:"type,omitempty"`
	Decision   ApprovalDecision `json:"decision"`
	Reason     string           `json:"reason,omitempty"`
	DiffHash   string           `json:"diff_hash,omitempty"`
	ResourceID string           `json:"resource_id,omitempty"`
}

func (a *Approval) Validate() error {
	if err := ids.Validate(ids.Approval, a.ApprovalID); err != nil {
		return err
	}
	if a.Decision != DecisionApprove && a.Decision != DecisionReject {
		return fmt.Errorf("invalid decision %q", a.Decision)
	}
	if a.Decision == DecisionReject && a.Reason == "" {
		return fmt.Errorf("reject reason required")
	}
	return nil
}

type ChangeOperation string

const (
	OpCreate ChangeOperation = "CREATE"
	OpModify ChangeOperation = "MODIFY"
	OpDelete ChangeOperation = "DELETE"
	OpRename ChangeOperation = "RENAME"
)

type FileChange struct {
	Path       string          `json:"path"`
	Operation  ChangeOperation `json:"operation"`
	BaseHash   string          `json:"base_hash,omitempty"`
	Content    string          `json:"content,omitempty"`
	TargetPath string          `json:"target_path,omitempty"`
}

type ProposedChangeSet struct {
	ChangeSetID           string       `json:"change_set_id"`
	WorkspaceID           string       `json:"workspace_id"`
	BaseWorkspaceRevision string       `json:"base_workspace_revision,omitempty"`
	Changes               []FileChange `json:"changes"`
}

func (p *ProposedChangeSet) Validate() error {
	if err := ids.Validate(ids.ChangeSet, p.ChangeSetID); err != nil {
		return err
	}
	if p.WorkspaceID != "" {
		if err := ids.Validate(ids.Workspace, p.WorkspaceID); err != nil {
			return err
		}
	}
	if len(p.Changes) == 0 {
		return fmt.Errorf("changes required")
	}
	for _, c := range p.Changes {
		if c.Path == "" {
			return fmt.Errorf("change path required")
		}
		switch c.Operation {
		case OpCreate, OpModify, OpDelete, OpRename:
		default:
			return fmt.Errorf("invalid operation %q", c.Operation)
		}
		if (c.Operation == OpModify || c.Operation == OpDelete || c.Operation == OpRename) && c.BaseHash == "" {
			return fmt.Errorf("base_hash required for %s", c.Operation)
		}
		if c.Operation == OpRename && c.TargetPath == "" {
			return fmt.Errorf("target_path required for RENAME")
		}
	}
	return nil
}

type ActualDiff struct {
	ChangeSetID string `json:"change_set_id"`
	Diff        string `json:"diff"`
	DiffHash    string `json:"diff_hash"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

func (d *ActualDiff) Validate() error {
	if err := ids.Validate(ids.ChangeSet, d.ChangeSetID); err != nil {
		return err
	}
	if d.DiffHash == "" {
		return fmt.Errorf("diff_hash required")
	}
	return nil
}
