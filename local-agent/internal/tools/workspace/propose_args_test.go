package workspacetool

import (
	"os"
	"path/filepath"
	"testing"

	"oncode/local-agent/internal/workspace"
	"oncode/protocol/ids"
	"oncode/protocol/messages"
)

func TestParseProposedChangeSet_ShorthandFillsHashAndIDs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "AuthService.java"), []byte("class AuthService {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	cs, err := parseProposedChangeSet(map[string]any{
		"path":      "src/AuthService.java",
		"operation": "MODIFY",
		"content":   "class AuthService { /* lock */ }",
	}, reg, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if cs.WorkspaceID != ws.ID {
		t.Fatalf("workspace_id=%s want %s", cs.WorkspaceID, ws.ID)
	}
	if err := ids.Validate(ids.ChangeSet, cs.ChangeSetID); err != nil {
		t.Fatal(err)
	}
	if len(cs.Changes) != 1 || cs.Changes[0].BaseHash == "" {
		t.Fatalf("%+v", cs.Changes)
	}
	if cs.Changes[0].Operation != messages.OpModify {
		t.Fatalf("op=%s", cs.Changes[0].Operation)
	}
}

func TestParseProposedChangeSet_EscapeRejected(t *testing.T) {
	root := t.TempDir()
	reg := workspace.NewRegistry()
	if _, err := reg.Register("", root); err != nil {
		t.Fatal(err)
	}
	_, err := parseProposedChangeSet(map[string]any{
		"path":      "../secret.txt",
		"operation": "MODIFY",
		"content":   "x",
	}, reg, 4096)
	if err == nil {
		t.Fatal("expected path escape to fail")
	}
}
