package workspace_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"oncode/local-agent/internal/workspace"
	protoerr "oncode/protocol/errors"
)

func TestResolvePath_NormalAndNested(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "src", "a.txt"), "hello")
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}

	abs, err := ws.ResolvePath("src/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(abs) != "a.txt" {
		t.Fatalf("got %s", abs)
	}
	nested, err := ws.ResolvePath(filepath.Join("src", "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if nested != abs {
		t.Fatalf("nested mismatch %s vs %s", nested, abs)
	}
}

func TestResolvePath_RejectDotDotAndAbsolute(t *testing.T) {
	root := t.TempDir()
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ws.ResolvePath("../secret.txt"); err == nil {
		t.Fatal("expected ../ rejection")
	} else if codeOf(err) != protoerr.PathOutsideWorkspace {
		t.Fatalf("code=%v err=%v", codeOf(err), err)
	}

	absOutside := filepath.Join(root, "..", "outside.txt")
	if _, err := ws.ResolvePath(absOutside); err == nil {
		t.Fatal("expected absolute rejection")
	} else if codeOf(err) != protoerr.PathOutsideWorkspace {
		t.Fatalf("code=%v err=%v", codeOf(err), err)
	}
}

func TestResolvePath_SymlinkInsideAndEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Creating symlinks may require admin/Developer Mode; skip if create fails.
	}
	root := t.TempDir()
	inside := filepath.Join(root, "inside")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(inside, "ok.txt"), "ok")

	outsideDir := t.TempDir()
	mustWrite(t, filepath.Join(outsideDir, "secret.txt"), "secret")

	linkInside := filepath.Join(root, "link-in")
	if err := os.Symlink(inside, linkInside); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	linkEscape := filepath.Join(root, "link-out")
	if err := os.Symlink(outsideDir, linkEscape); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ws.ResolvePath("link-in/ok.txt"); err != nil {
		t.Fatalf("inside symlink: %v", err)
	}
	_, err = ws.ResolvePath("link-out/secret.txt")
	if err == nil {
		t.Fatal("expected symlink escape")
	}
	if codeOf(err) != protoerr.SymlinkEscape && codeOf(err) != protoerr.PathOutsideWorkspace {
		t.Fatalf("unexpected code %v for %v", codeOf(err), err)
	}
}

func TestReadTextFile_HashAndMissing(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "greet.txt"), "hi")
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ws.ReadTextFile("greet.txt", 1024)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "hi" || got.Size != 2 {
		t.Fatalf("%+v", got)
	}
	wantHash := workspace.HashBytes([]byte("hi"))
	if got.Hash != wantHash {
		t.Fatalf("hash %s want %s", got.Hash, wantHash)
	}

	if _, err := ws.ReadTextFile("missing.txt", 1024); err == nil {
		t.Fatal("expected missing")
	} else if codeOf(err) != protoerr.WorkspaceFileNotFound {
		t.Fatalf("code=%v", codeOf(err))
	}
}

func TestReadTextFile_TooLarge(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "big.txt"), "abcdef")
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ReadTextFile("big.txt", 3); err == nil {
		t.Fatal("expected too large")
	} else if codeOf(err) != protoerr.FileTooLarge {
		t.Fatalf("code=%v", codeOf(err))
	}
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

func codeOf(err error) protoerr.Code {
	if pe, ok := err.(*protoerr.Error); ok {
		return pe.Code
	}
	return ""
}
