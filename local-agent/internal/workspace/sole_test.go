package workspace_test

import (
	"testing"

	"oncode/local-agent/internal/workspace"
)

func TestRegistry_SoleRequiresExactlyOne(t *testing.T) {
	reg := workspace.NewRegistry()
	if _, err := reg.Sole(); err == nil {
		t.Fatal("empty registry must fail")
	}
	root := t.TempDir()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reg.Sole()
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != ws.ID {
		t.Fatalf("id=%s want %s", got.ID, ws.ID)
	}
	if _, err := reg.Register("", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Sole(); err == nil {
		t.Fatal("two workspaces must fail")
	}
}
