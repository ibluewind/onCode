package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"time"

	"oncode/local-agent/internal/audit"
	"oncode/local-agent/internal/build"
	"oncode/local-agent/internal/config"
	"oncode/local-agent/internal/maven"
	"oncode/local-agent/internal/policy"
	"oncode/local-agent/internal/testrun"
	"oncode/local-agent/internal/tools"
	buildtool "oncode/local-agent/internal/tools/build"
	gittool "oncode/local-agent/internal/tools/git"
	"oncode/local-agent/internal/tools/system"
	testtool "oncode/local-agent/internal/tools/test"
	workspacetool "oncode/local-agent/internal/tools/workspace"
	"oncode/local-agent/internal/transport"
	"oncode/local-agent/internal/workspace"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/messages"
	"oncode/protocol/versioning"
)

// Agent is the Local Agent process core.
type Agent struct {
	Cfg        config.Config
	Workspaces *workspace.Registry
	Proposals  *workspace.ProposalStore
	Tools      *tools.Registry
	Dispatcher *tools.Dispatcher
	Audit      *audit.Logger
}

func New(cfg config.Config, logOut io.Writer) (*Agent, error) {
	if cfg.AgentID == "" {
		id, err := newAgentID()
		if err != nil {
			return nil, err
		}
		cfg.AgentID = id
	}
	if logOut == nil {
		logOut = os.Stderr
	}
	logger := audit.New(logOut)
	reg := tools.NewRegistry()
	ws := workspace.NewRegistry()
	proposals := workspace.NewProposalStore()
	a := &Agent{
		Cfg:        cfg,
		Workspaces: ws,
		Proposals:  proposals,
		Tools:      reg,
		Audit:      logger,
	}
	a.Dispatcher = tools.NewDispatcher(reg, logger)

	system.Register(reg, system.Deps{
		AgentID:      cfg.AgentID,
		AgentVersion: cfg.AgentVersion,
		ToolNames:    reg.Names,
	})
	workspacetool.Register(reg, workspacetool.Deps{
		Workspaces:     ws,
		Proposals:      proposals,
		Approval:       policy.LocalApproval{},
		MaxFileBytes:   cfg.MaxFileBytes,
		SearchExcludes: config.DefaultSearchExcludeDirs(),
		MaxSearchHits:  config.DefaultMaxSearchResults,
	})
	buildtool.Register(reg, buildtool.Deps{
		Workspaces: ws,
		Adapters:   []build.Adapter{maven.BuildAdapter{}},
	})
	testtool.Register(reg, testtool.Deps{
		Workspaces: ws,
		Adapters:   []testrun.Adapter{maven.TestAdapter{}},
	})
	gittool.Register(reg, gittool.Deps{Workspaces: ws})
	return a, nil
}

func (a *Agent) HandleEnvelope(ctx context.Context, env *messages.Envelope) (*messages.Envelope, error) {
	if env == nil {
		return a.errorEnvelope("", protoerr.InvalidRequest, "nil envelope")
	}
	if err := env.Validate(); err != nil {
		return a.errorEnvelope("", protoerr.InvalidRequest, err.Error())
	}
	if _, err := versioning.ParseCompatible(env.Protocol); err != nil {
		return a.errorEnvelope("", protoerr.ProtocolVersionUnsupported, err.Error())
	}
	if env.MessageType != messages.TypeRequest {
		return a.errorEnvelope("", protoerr.InvalidRequest, "expected REQUEST")
	}
	var req messages.ToolRequestPayload
	if err := json.Unmarshal(env.Payload, &req); err != nil {
		return a.errorEnvelope("", protoerr.InvalidRequest, "invalid tool request payload")
	}
	resp := a.Dispatcher.Dispatch(ctx, &req)
	out, err := messages.NewEnvelope(messages.TypeResponse, resp)
	if err != nil {
		return nil, err
	}
	out.SessionID = env.SessionID
	out.ProjectID = env.ProjectID
	out.WorkspaceID = env.WorkspaceID
	out.WorkflowID = env.WorkflowID
	out.WorkItemID = env.WorkItemID
	out.TaskID = env.TaskID
	out.Actor = &messages.Actor{Type: messages.ActorLocalAgent, ID: a.Cfg.AgentID}
	return out, nil
}

func (a *Agent) Serve(ctx context.Context, t transport.Transport) error {
	if err := t.Connect(ctx); err != nil {
		return err
	}
	if err := t.Register(ctx, a.Cfg.AgentID); err != nil {
		return err
	}
	hb := messages.HeartbeatPayload{
		AgentID:         a.Cfg.AgentID,
		ProtocolVersion: versioning.Current,
		Timestamp:       time.Now().UTC().Format(time.RFC3339Nano),
	}
	_ = t.SendHeartbeat(ctx, hb)
	return transport.Loop(ctx, t, a.HandleEnvelope)
}

func (a *Agent) errorEnvelope(callID string, code protoerr.Code, msg string) (*messages.Envelope, error) {
	e, err := protoerr.New(code, msg)
	if err != nil {
		return nil, err
	}
	payload := messages.ToolResponsePayload{
		CallID: callID,
		Status: messages.StatusFailed,
		Error:  e,
	}
	return messages.NewEnvelope(messages.TypeResponse, payload)
}

func newAgentID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "agent-" + hex.EncodeToString(b[:]), nil
}
