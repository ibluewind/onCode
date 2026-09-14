package agent_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oncode/local-agent/internal/agent"
	"oncode/local-agent/internal/config"
	"oncode/protocol/ids"
	"oncode/protocol/messages"
	"oncode/protocol/versioning"
)

func TestSystemPingAndCapabilities(t *testing.T) {
	a := newTestAgent(t)
	callID := ids.MustNew(ids.Call)

	pingEnv := mustRequest(t, callID, "system.ping", map[string]any{})
	pingResp, err := a.HandleEnvelope(context.Background(), pingEnv)
	if err != nil {
		t.Fatal(err)
	}
	payload := decodeResponse(t, pingResp)
	if payload.Status != messages.StatusSuccess {
		t.Fatalf("status=%s err=%v", payload.Status, payload.Error)
	}
	var ping struct {
		AgentID         string `json:"agent_id"`
		Version         string `json:"version"`
		ProtocolVersion string `json:"protocol_version"`
	}
	if err := json.Unmarshal(payload.Result, &ping); err != nil {
		t.Fatal(err)
	}
	if ping.AgentID != a.Cfg.AgentID || ping.ProtocolVersion != versioning.Current {
		t.Fatalf("%+v", ping)
	}

	capEnv := mustRequest(t, ids.MustNew(ids.Call), "system.get_capabilities", map[string]any{})
	capResp, err := a.HandleEnvelope(context.Background(), capEnv)
	if err != nil {
		t.Fatal(err)
	}
	capPayload := decodeResponse(t, capResp)
	if capPayload.Status != messages.StatusSuccess {
		t.Fatalf("%v", capPayload.Error)
	}
	var caps struct {
		SupportedTools []string `json:"supported_tools"`
		Capabilities   struct {
			Workspace bool     `json:"workspace"`
			Git       bool     `json:"git"`
			Build     []string `json:"build"`
			Test      []string `json:"test"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(capPayload.Result, &caps); err != nil {
		t.Fatal(err)
	}
	if !caps.Capabilities.Workspace || !caps.Capabilities.Git {
		t.Fatalf("%+v", caps)
	}
	if len(caps.Capabilities.Build) == 0 || caps.Capabilities.Build[0] != "maven" {
		t.Fatalf("build caps=%v", caps.Capabilities.Build)
	}
	if !contains(caps.SupportedTools, "system.ping") ||
		!contains(caps.SupportedTools, "workspace.read_file") ||
		!contains(caps.SupportedTools, "workspace.read_files") ||
		!contains(caps.SupportedTools, "workspace.search") ||
		!contains(caps.SupportedTools, "workspace.propose_changes") ||
		!contains(caps.SupportedTools, "workspace.apply_changes") ||
		!contains(caps.SupportedTools, "build.run") ||
		!contains(caps.SupportedTools, "test.run") ||
		!contains(caps.SupportedTools, "git.status") ||
		!contains(caps.SupportedTools, "git.diff") {
		t.Fatalf("tools=%v", caps.SupportedTools)
	}
}

func TestWorkspaceReadFileTool(t *testing.T) {
	a := newTestAgent(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := a.Workspaces.Register("", root)
	if err != nil {
		t.Fatal(err)
	}

	env := mustRequest(t, ids.MustNew(ids.Call), "workspace.read_file", map[string]any{
		"workspace_id": ws.ID,
		"path":         "f.txt",
	})
	resp, err := a.HandleEnvelope(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	payload := decodeResponse(t, resp)
	if payload.Status != messages.StatusSuccess {
		t.Fatalf("%v", payload.Error)
	}
	var result struct {
		Content string `json:"content"`
		Hash    string `json:"hash"`
		Size    int64  `json:"size"`
	}
	if err := json.Unmarshal(payload.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result.Content != "abc" || result.Size != 3 || result.Hash == "" {
		t.Fatalf("%+v", result)
	}
}

func TestWorkspaceReadFilesAndSearchTools(t *testing.T) {
	a := newTestAgent(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ok.txt"), []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := a.Workspaces.Register("", root)
	if err != nil {
		t.Fatal(err)
	}

	readEnv := mustRequest(t, ids.MustNew(ids.Call), "workspace.read_files", map[string]any{
		"workspace_id": ws.ID,
		"paths":        []any{"ok.txt", "missing.txt"},
	})
	readResp, err := a.HandleEnvelope(context.Background(), readEnv)
	if err != nil {
		t.Fatal(err)
	}
	readPayload := decodeResponse(t, readResp)
	if readPayload.Status != messages.StatusPartial {
		t.Fatalf("status=%s err=%v", readPayload.Status, readPayload.Error)
	}

	searchEnv := mustRequest(t, ids.MustNew(ids.Call), "workspace.search", map[string]any{
		"workspace_id": ws.ID,
		"query":        "hello",
		"mode":         "content",
	})
	searchResp, err := a.HandleEnvelope(context.Background(), searchEnv)
	if err != nil {
		t.Fatal(err)
	}
	searchPayload := decodeResponse(t, searchResp)
	if searchPayload.Status != messages.StatusSuccess {
		t.Fatalf("%v", searchPayload.Error)
	}
	var searchResult struct {
		Matches []struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
		} `json:"matches"`
	}
	if err := json.Unmarshal(searchPayload.Result, &searchResult); err != nil {
		t.Fatal(err)
	}
	if len(searchResult.Matches) < 1 || searchResult.Matches[0].Kind != "content" {
		t.Fatalf("%+v", searchResult)
	}
}

func TestWorkspaceProposeChangesTool(t *testing.T) {
	a := newTestAgent(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := a.Workspaces.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	hash := workspaceHash(t, a, ws.ID, "f.txt")
	chgID := ids.MustNew(ids.ChangeSet)

	env := mustRequest(t, ids.MustNew(ids.Call), "workspace.propose_changes", map[string]any{
		"change_set_id": chgID,
		"workspace_id":  ws.ID,
		"changes": []any{
			map[string]any{
				"path":      "f.txt",
				"operation": "MODIFY",
				"base_hash": hash,
				"content":   "a\nb\n",
			},
		},
	})
	resp, err := a.HandleEnvelope(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	payload := decodeResponse(t, resp)
	if payload.Status != messages.StatusSuccess {
		t.Fatalf("%v", payload.Error)
	}
	var result struct {
		Diff     string `json:"diff"`
		DiffHash string `json:"diff_hash"`
	}
	if err := json.Unmarshal(payload.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result.DiffHash == "" || !strings.Contains(result.Diff, "+b") {
		t.Fatalf("%+v", result)
	}
	if _, ok := a.Proposals.Get(chgID); !ok {
		t.Fatal("proposal not stored")
	}
	raw, _ := os.ReadFile(filepath.Join(root, "f.txt"))
	if string(raw) != "a\n" {
		t.Fatalf("mutated: %q", raw)
	}
}

func TestWorkspaceProposeAndApplyTools(t *testing.T) {
	a := newTestAgent(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := a.Workspaces.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	hash := workspaceHash(t, a, ws.ID, "f.txt")
	chgID := ids.MustNew(ids.ChangeSet)

	proposeEnv := mustRequest(t, ids.MustNew(ids.Call), "workspace.propose_changes", map[string]any{
		"change_set_id": chgID,
		"workspace_id":  ws.ID,
		"changes": []any{
			map[string]any{
				"path":      "f.txt",
				"operation": "MODIFY",
				"base_hash": hash,
				"content":   "a\nb\n",
			},
		},
	})
	proposeResp, err := a.HandleEnvelope(context.Background(), proposeEnv)
	if err != nil {
		t.Fatal(err)
	}
	proposePayload := decodeResponse(t, proposeResp)
	if proposePayload.Status != messages.StatusSuccess {
		t.Fatalf("%v", proposePayload.Error)
	}
	var proposed struct {
		DiffHash string `json:"diff_hash"`
	}
	if err := json.Unmarshal(proposePayload.Result, &proposed); err != nil {
		t.Fatal(err)
	}

	applyEnv := mustRequest(t, ids.MustNew(ids.Call), "workspace.apply_changes", map[string]any{
		"change_set_id":       chgID,
		"workspace_id":        ws.ID,
		"approval_id":         ids.MustNew(ids.Approval),
		"approved_diff_hash":  proposed.DiffHash,
	})
	applyResp, err := a.HandleEnvelope(context.Background(), applyEnv)
	if err != nil {
		t.Fatal(err)
	}
	applyPayload := decodeResponse(t, applyResp)
	if applyPayload.Status != messages.StatusSuccess {
		t.Fatalf("%v", applyPayload.Error)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "f.txt"))
	if string(raw) != "a\nb\n" {
		t.Fatalf("%q", raw)
	}
}

func workspaceHash(t *testing.T, a *agent.Agent, workspaceID, path string) string {
	t.Helper()
	env := mustRequest(t, ids.MustNew(ids.Call), "workspace.read_file", map[string]any{
		"workspace_id": workspaceID,
		"path":         path,
	})
	resp, err := a.HandleEnvelope(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	payload := decodeResponse(t, resp)
	var result struct {
		Hash string `json:"hash"`
	}
	if err := json.Unmarshal(payload.Result, &result); err != nil {
		t.Fatal(err)
	}
	return result.Hash
}

func TestUnknownTool(t *testing.T) {
	a := newTestAgent(t)
	env := mustRequest(t, ids.MustNew(ids.Call), "system.nope", map[string]any{})
	resp, err := a.HandleEnvelope(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	payload := decodeResponse(t, resp)
	if payload.Status != messages.StatusFailed || payload.Error == nil {
		t.Fatalf("%+v", payload)
	}
	if string(payload.Error.Code) != "TOOL_NOT_SUPPORTED" {
		t.Fatalf("code=%s", payload.Error.Code)
	}
}

func newTestAgent(t *testing.T) *agent.Agent {
	t.Helper()
	cfg := config.Config{
		AgentID:      "agent-test",
		AgentVersion: "0.1.0-test",
		MaxFileBytes: 1024,
		StateDir:     t.TempDir(),
	}
	a, err := agent.New(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func mustRequest(t *testing.T, callID, tool string, args map[string]any) *messages.Envelope {
	t.Helper()
	payload := messages.ToolRequestPayload{
		CallID:    callID,
		Tool:      tool,
		Arguments: args,
	}
	env, err := messages.NewEnvelope(messages.TypeRequest, payload)
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func decodeResponse(t *testing.T, env *messages.Envelope) messages.ToolResponsePayload {
	t.Helper()
	var p messages.ToolResponsePayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
