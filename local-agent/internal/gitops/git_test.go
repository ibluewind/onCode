package gitops_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"oncode/local-agent/internal/gitops"
	protoerr "oncode/protocol/errors"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=oncode-test",
		"GIT_AUTHOR_EMAIL=test@oncode.local",
		"GIT_COMMITTER_NAME=oncode-test",
		"GIT_COMMITTER_EMAIL=test@oncode.local",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	mustWrite(t, filepath.Join(dir, "tracked.txt"), "v1\n")
	git(t, dir, "add", "tracked.txt")
	git(t, dir, "commit", "-m", "init")
	return dir
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStatusAndDiff(t *testing.T) {
	dir := initRepo(t)
	mustWrite(t, filepath.Join(dir, "tracked.txt"), "v2\n")
	mustWrite(t, filepath.Join(dir, "new.txt"), "new\n")
	git(t, dir, "add", "tracked.txt") // staged modify; working tree clean for that file after add if no further edit
	mustWrite(t, filepath.Join(dir, "tracked.txt"), "v3\n") // unstaged further change

	st, err := gitops.Status(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Branch != "main" || st.Head == "" {
		t.Fatalf("%+v", st)
	}
	if !contains(st.Staged, "tracked.txt") {
		t.Fatalf("staged=%v", st.Staged)
	}
	if !contains(st.Changed, "tracked.txt") {
		t.Fatalf("changed=%v", st.Changed)
	}
	if !contains(st.Untracked, "new.txt") {
		t.Fatalf("untracked=%v", st.Untracked)
	}

	diff, err := gitops.Diff(context.Background(), dir, "tracked.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff.Diff, "v3") && !strings.Contains(diff.Diff, "+v3") {
		// unstaged diff should show working tree vs index
		t.Fatalf("unstaged diff=%q", diff.Diff)
	}
	stagedDiff, err := gitops.Diff(context.Background(), dir, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if stagedDiff.Diff == "" {
		t.Fatal("expected staged diff")
	}
}

func TestNotARepo(t *testing.T) {
	requireGit(t)
	_, err := gitops.Status(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
	if pe, ok := err.(*protoerr.Error); !ok || pe.Code != protoerr.GitNotAvailable {
		t.Fatalf("err=%v", err)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
