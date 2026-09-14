package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oncode/local-agent/internal/workspace"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/ids"
	"oncode/protocol/messages"
)

func TestPropose_ModifyDoesNotMutateAndHashes(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "greet.txt"), "hello\n")
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	before, err := ws.ReadTextFile("greet.txt", 1024)
	if err != nil {
		t.Fatal(err)
	}

	cs := messages.ProposedChangeSet{
		ChangeSetID: ids.MustNew(ids.ChangeSet),
		WorkspaceID: ws.ID,
		Changes: []messages.FileChange{{
			Path:      "greet.txt",
			Operation: messages.OpModify,
			BaseHash:  before.Hash,
			Content:   "hello\nworld\n",
		}},
	}
	prop, err := ws.Propose(cs, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if prop.ActualDiff.DiffHash == "" || !strings.HasPrefix(prop.ActualDiff.DiffHash, "sha256:") {
		t.Fatalf("hash=%s", prop.ActualDiff.DiffHash)
	}
	if !strings.Contains(prop.ActualDiff.Diff, "+world") {
		t.Fatalf("diff=%q", prop.ActualDiff.Diff)
	}

	raw, err := os.ReadFile(filepath.Join(root, "greet.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "hello\n" {
		t.Fatalf("file mutated: %q", raw)
	}

	// wrong hash
	cs.ChangeSetID = ids.MustNew(ids.ChangeSet)
	cs.Changes[0].BaseHash = "sha256:" + strings.Repeat("0", 64)
	if _, err := ws.Propose(cs, 1024); err == nil {
		t.Fatal("expected hash mismatch")
	} else if codeOf(err) != protoerr.FileHashMismatch {
		t.Fatalf("code=%v", codeOf(err))
	}
}

func TestPropose_CreateDeleteRename(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "old.txt"), "same\n")
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	oldHash := workspace.HashBytes([]byte("same\n"))

	createID := ids.MustNew(ids.ChangeSet)
	prop, err := ws.Propose(messages.ProposedChangeSet{
		ChangeSetID: createID,
		WorkspaceID: ws.ID,
		Changes: []messages.FileChange{{
			Path:      "new.txt",
			Operation: messages.OpCreate,
			Content:   "fresh\n",
		}},
	}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prop.ActualDiff.Diff, "+++ b/new.txt") {
		t.Fatalf("%q", prop.ActualDiff.Diff)
	}

	delID := ids.MustNew(ids.ChangeSet)
	if _, err := ws.Propose(messages.ProposedChangeSet{
		ChangeSetID: delID,
		WorkspaceID: ws.ID,
		Changes: []messages.FileChange{{
			Path:      "old.txt",
			Operation: messages.OpDelete,
			BaseHash:  oldHash,
		}},
	}, 1024); err != nil {
		t.Fatal(err)
	}

	renID := ids.MustNew(ids.ChangeSet)
	prop, err = ws.Propose(messages.ProposedChangeSet{
		ChangeSetID: renID,
		WorkspaceID: ws.ID,
		Changes: []messages.FileChange{{
			Path:       "old.txt",
			Operation:  messages.OpRename,
			BaseHash:   oldHash,
			TargetPath: "renamed.txt",
		}},
	}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prop.ActualDiff.Diff, "renamed.txt") {
		t.Fatalf("%q", prop.ActualDiff.Diff)
	}
}

func TestPropose_DuplicatePathRejected(t *testing.T) {
	root := t.TempDir()
	reg := workspace.NewRegistry()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ws.Propose(messages.ProposedChangeSet{
		ChangeSetID: ids.MustNew(ids.ChangeSet),
		WorkspaceID: ws.ID,
		Changes: []messages.FileChange{
			{Path: "a.txt", Operation: messages.OpCreate, Content: "1"},
			{Path: "a.txt", Operation: messages.OpCreate, Content: "2"},
		},
	}, 1024)
	if err == nil || codeOf(err) != protoerr.InvalidChangeSet {
		t.Fatalf("err=%v", err)
	}
}
