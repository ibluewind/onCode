package actualdiff

import (
	"strings"
	"testing"
)

func TestFromProposed_IsUnifiedNotModelBlob(t *testing.T) {
	proposed := `{"change_set_id":"CS-1","changes":[{"path":"src/AuthService.java","operation":"MODIFY","content":"class AuthService { /* lock */ }"}]}`
	id, changes, err := ParseProposedJSON(proposed)
	if err != nil {
		t.Fatal(err)
	}
	got, err := FromProposed(id, "ws-local", changes, func(path string) string {
		if path == "src/AuthService.java" {
			return "class AuthService {}"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Diff, "--- a/src/AuthService.java") {
		t.Fatalf("expected unified diff, got %q", got.Diff)
	}
	if !strings.Contains(got.Diff, "+class AuthService { /* lock */ }") {
		t.Fatalf("missing added line: %q", got.Diff)
	}
	if got.DiffHash == "" || !strings.HasPrefix(got.DiffHash, "sha256:") {
		t.Fatalf("hash=%s", got.DiffHash)
	}
	if got.Diff == proposed || strings.Contains(got.Diff, "change_set_id") {
		t.Fatal("must not display model/proposal JSON as the diff")
	}
}

func TestFromProposed_HashChangesWithWorkspace(t *testing.T) {
	changes := []Change{{Path: "A.java", Operation: "MODIFY", Content: "new"}}
	empty, err := FromProposed("CS-1", "ws", changes, nil)
	if err != nil {
		t.Fatal(err)
	}
	withOld, err := FromProposed("CS-1", "ws", changes, func(string) string { return "old" })
	if err != nil {
		t.Fatal(err)
	}
	if empty.DiffHash == withOld.DiffHash {
		t.Fatal("hash must depend on current workspace content")
	}
}
