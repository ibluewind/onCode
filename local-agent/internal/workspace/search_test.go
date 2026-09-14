package workspace_test

import (
	"path/filepath"
	"strings"
	"testing"

	"oncode/local-agent/internal/workspace"
	protoerr "oncode/protocol/errors"
)

func TestReadTextFiles_PartialSuccess(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), "A")
	mustWrite(t, filepath.Join(root, "b.txt"), "B")
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}

	res, err := ws.ReadTextFiles([]string{"a.txt", "missing.txt", "b.txt"}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 3 || res.Success != 2 || res.Failed != 1 {
		t.Fatalf("%+v", res)
	}
	if res.OverallStatus() != "PARTIAL" {
		t.Fatalf("status=%s", res.OverallStatus())
	}
	if res.Files[0].Status != "SUCCESS" || res.Files[0].Content != "A" {
		t.Fatalf("%+v", res.Files[0])
	}
	if res.Files[1].Status != "FAILED" || codeOf(res.Files[1].Error) != protoerr.WorkspaceFileNotFound {
		t.Fatalf("%+v", res.Files[1])
	}
	if res.Files[2].Status != "SUCCESS" || res.Files[2].Content != "B" {
		t.Fatalf("%+v", res.Files[2])
	}
}

func TestReadTextFiles_EmptyPaths(t *testing.T) {
	root := t.TempDir()
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ReadTextFiles(nil, 1024); err == nil {
		t.Fatal("expected error")
	} else if codeOf(err) != protoerr.InvalidArgument {
		t.Fatalf("code=%v", codeOf(err))
	}
}

func TestSearch_NamePathContentAndExclude(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "src", "Greeter.java"), "class Greeter { Hello }\n")
	mustWrite(t, filepath.Join(root, "src", "other.txt"), "nope\n")
	mustWrite(t, filepath.Join(root, "target", "Greeter.class"), "binary-ish Greeter\n")
	mustWrite(t, filepath.Join(root, "node_modules", "pkg", "index.js"), "Greeter\n")

	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}

	opts := workspace.DefaultSearchOptions("Greeter", workspace.SearchModeAll)
	opts.MaxFileBytes = 1024
	res, err := ws.Search(opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Truncated {
		t.Fatal("unexpected truncate")
	}
	kinds := map[string]int{}
	for _, m := range res.Matches {
		p := filepath.ToSlash(m.Path)
		if strings.Contains(p, "target/") || strings.HasPrefix(p, "target/") {
			t.Fatalf("excluded target hit: %+v", m)
		}
		if strings.Contains(p, "node_modules/") {
			t.Fatalf("excluded node_modules hit: %+v", m)
		}
		kinds[m.Kind]++
	}
	if kinds["name"] < 1 || kinds["path"] < 1 || kinds["content"] < 1 {
		t.Fatalf("kinds=%v matches=%+v", kinds, res.Matches)
	}

	nameOnly, err := ws.Search(workspace.DefaultSearchOptions("Greeter", workspace.SearchModeName))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range nameOnly.Matches {
		if m.Kind != "name" {
			t.Fatalf("expected name only: %+v", m)
		}
	}
}

func TestSearch_RequiresQuery(t *testing.T) {
	root := t.TempDir()
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Search(workspace.DefaultSearchOptions("  ", workspace.SearchModeAll)); err == nil {
		t.Fatal("expected query required")
	} else if codeOf(err) != protoerr.InvalidArgument {
		t.Fatalf("code=%v", codeOf(err))
	}
}
