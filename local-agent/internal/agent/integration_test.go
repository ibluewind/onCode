package agent_test

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"oncode/local-agent/internal/agent"
	"oncode/local-agent/internal/maven"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/ids"
	"oncode/protocol/messages"
)

const greeterRel = "src/main/java/com/oncode/sample/Greeter.java"

func TestPhase1Integration_SampleJavaMavenFlow(t *testing.T) {
	root := copySampleMaven(t)
	requireMavenForIntegration(t, root)

	a := newTestAgent(t)
	ws, err := a.Workspaces.Register("", root)
	if err != nil {
		t.Fatal(err)
	}

	// 1. read Greeter.java
	readPayload := callTool(t, a, "workspace.read_file", map[string]any{
		"workspace_id": ws.ID,
		"path":         greeterRel,
	})
	if readPayload.Status != messages.StatusSuccess {
		t.Fatalf("read: %v", readPayload.Error)
	}
	var readRes struct {
		Content string `json:"content"`
		Hash    string `json:"hash"`
	}
	mustUnmarshal(t, readPayload.Result, &readRes)
	if !strings.Contains(readRes.Content, "greet") || readRes.Hash == "" {
		t.Fatalf("%+v", readRes)
	}

	// 2–3. propose modified Greeter + verify actual diff
	// Keep behavior so Maven tests still pass; only mark the source as changed.
	modified := strings.Replace(readRes.Content, "public final class Greeter {", "public final class Greeter { // phase1-integration", 1)
	if modified == readRes.Content {
		t.Fatal("failed to prepare modified content")
	}
	chgID := ids.MustNew(ids.ChangeSet)
	proposePayload := callTool(t, a, "workspace.propose_changes", map[string]any{
		"change_set_id": chgID,
		"workspace_id":  ws.ID,
		"changes": []any{
			map[string]any{
				"path":      greeterRel,
				"operation": "MODIFY",
				"base_hash": readRes.Hash,
				"content":   modified,
			},
		},
	})
	if proposePayload.Status != messages.StatusSuccess {
		t.Fatalf("propose: %v", proposePayload.Error)
	}
	var proposeRes struct {
		Diff     string `json:"diff"`
		DiffHash string `json:"diff_hash"`
	}
	mustUnmarshal(t, proposePayload.Result, &proposeRes)
	if proposeRes.DiffHash == "" || !strings.Contains(proposeRes.Diff, "phase1-integration") {
		t.Fatalf("diff=%q hash=%s", proposeRes.Diff, proposeRes.DiffHash)
	}
	// propose must not mutate
	raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(greeterRel)))
	if string(raw) != readRes.Content {
		t.Fatal("propose mutated file")
	}

	// 4–5. approve + apply
	applyPayload := callTool(t, a, "workspace.apply_changes", map[string]any{
		"change_set_id":      chgID,
		"workspace_id":       ws.ID,
		"approval_id":        ids.MustNew(ids.Approval),
		"approved_diff_hash": proposeRes.DiffHash,
	})
	if applyPayload.Status != messages.StatusSuccess {
		t.Fatalf("apply: %v", applyPayload.Error)
	}
	var applyRes struct {
		WorkspaceRevision int64 `json:"workspace_revision"`
		Files             []struct {
			Path string `json:"path"`
			Hash string `json:"hash"`
		} `json:"files"`
	}
	mustUnmarshal(t, applyPayload.Result, &applyRes)
	if applyRes.WorkspaceRevision < 1 {
		t.Fatalf("revision=%d", applyRes.WorkspaceRevision)
	}
	after, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(greeterRel)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "phase1-integration") {
		t.Fatalf("apply content: %s", after)
	}

	// 7. verify hash via read
	read2 := callTool(t, a, "workspace.read_file", map[string]any{
		"workspace_id": ws.ID,
		"path":         greeterRel,
	})
	var readRes2 struct {
		Hash    string `json:"hash"`
		Content string `json:"content"`
	}
	mustUnmarshal(t, read2.Result, &readRes2)
	if readRes2.Hash == readRes.Hash {
		t.Fatal("hash should change after apply")
	}
	if len(applyRes.Files) > 0 && applyRes.Files[0].Hash != "" && applyRes.Files[0].Hash != readRes2.Hash {
		t.Fatalf("apply hash %s != read hash %s", applyRes.Files[0].Hash, readRes2.Hash)
	}

	// 6. maven compile + test
	buildPayload := callTool(t, a, "build.run", map[string]any{"workspace_id": ws.ID})
	if buildPayload.Status != messages.StatusSuccess {
		t.Fatalf("build: %v result=%s", buildPayload.Error, string(buildPayload.Result))
	}
	testPayload := callTool(t, a, "test.run", map[string]any{"workspace_id": ws.ID})
	if testPayload.Status != messages.StatusSuccess {
		t.Fatalf("test: %v result=%s", testPayload.Error, string(testPayload.Result))
	}
}

func TestPhase1Integration_StaleApplyRejected(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "a.txt"), "one\n")
	a := newTestAgent(t)
	ws, err := a.Workspaces.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	readPayload := callTool(t, a, "workspace.read_file", map[string]any{
		"workspace_id": ws.ID,
		"path":         "a.txt",
	})
	var readRes struct {
		Hash    string `json:"hash"`
		Content string `json:"content"`
	}
	mustUnmarshal(t, readPayload.Result, &readRes)

	chgID := ids.MustNew(ids.ChangeSet)
	proposePayload := callTool(t, a, "workspace.propose_changes", map[string]any{
		"change_set_id": chgID,
		"workspace_id":  ws.ID,
		"changes": []any{
			map[string]any{
				"path":      "a.txt",
				"operation": "MODIFY",
				"base_hash": readRes.Hash,
				"content":   "two\n",
			},
		},
	})
	var proposeRes struct {
		DiffHash string `json:"diff_hash"`
	}
	mustUnmarshal(t, proposePayload.Result, &proposeRes)

	// external edit after propose
	mustWriteFile(t, filepath.Join(root, "a.txt"), "external\n")

	applyPayload := callTool(t, a, "workspace.apply_changes", map[string]any{
		"change_set_id":      chgID,
		"workspace_id":       ws.ID,
		"approval_id":        ids.MustNew(ids.Approval),
		"approved_diff_hash": proposeRes.DiffHash,
	})
	if applyPayload.Status != messages.StatusFailed || applyPayload.Error == nil {
		t.Fatalf("expected failure: %+v", applyPayload)
	}
	code := applyPayload.Error.Code
	if code != protoerr.WorkspaceFileChanged && code != protoerr.FileHashMismatch && code != protoerr.DiffMismatch {
		t.Fatalf("code=%s", code)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(raw) != "external\n" {
		t.Fatalf("file should remain external edit: %q", raw)
	}
}

func TestPhase1DoD_NoShellExecuteTool(t *testing.T) {
	a := newTestAgent(t)
	for _, name := range a.Tools.Names() {
		if strings.Contains(name, "shell") {
			t.Fatalf("unexpected shell tool: %s", name)
		}
	}
}

func callTool(t *testing.T, a *agent.Agent, tool string, args map[string]any) messages.ToolResponsePayload {
	t.Helper()
	env := mustRequest(t, ids.MustNew(ids.Call), tool, args)
	resp, err := a.HandleEnvelope(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	return decodeResponse(t, resp)
}

func mustUnmarshal(t *testing.T, raw json.RawMessage, v any) {
	t.Helper()
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sampleMavenSource(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "sample-java-maven"))
	if _, err := os.Stat(filepath.Join(root, "pom.xml")); err != nil {
		t.Fatal(err)
	}
	return root
}

func copySampleMaven(t *testing.T) string {
	t.Helper()
	src := sampleMavenSource(t)
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		// skip build outputs
		if rel == "target" || strings.HasPrefix(rel, "target"+string(os.PathSeparator)) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		out := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(f, in)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func requireMavenForIntegration(t *testing.T, root string) {
	t.Helper()
	if _, err := maven.ResolveExecutable(root); err != nil {
		t.Skip("maven not available: ", err)
	}
	if _, err := exec.LookPath("java"); err != nil {
		t.Log("java not on PATH; maven may still work")
	}
}
