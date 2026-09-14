package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"oncode/local-agent/internal/policy"
	"oncode/local-agent/internal/workspace"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/ids"
	"oncode/protocol/messages"
)

func TestApplyChanges_SuccessAndRejectReapply(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), "one\n")
	reg := workspace.NewRegistry()
	store := workspace.NewProposalStore()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	hash := workspace.HashBytes([]byte("one\n"))
	chg := ids.MustNew(ids.ChangeSet)
	prop, err := ws.Propose(messages.ProposedChangeSet{
		ChangeSetID: chg,
		WorkspaceID: ws.ID,
		Changes: []messages.FileChange{{
			Path: "a.txt", Operation: messages.OpModify, BaseHash: hash, Content: "one\ntwo\n",
		}},
	}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	store.Put(prop)

	res, err := workspace.ApplyChanges(reg, store, policy.LocalApproval{}, workspace.ApplyRequest{
		ChangeSetID: chg, WorkspaceID: ws.ID,
		ApprovalID: ids.MustNew(ids.Approval), ApprovedDiffHash: prop.ActualDiff.DiffHash,
	}, workspace.ApplyOptions{MaxBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	if res.WorkspaceRevision != 1 {
		t.Fatalf("revision=%d", res.WorkspaceRevision)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(raw) != "one\ntwo\n" {
		t.Fatalf("%q", raw)
	}

	_, err = workspace.ApplyChanges(reg, store, policy.LocalApproval{}, workspace.ApplyRequest{
		ChangeSetID: chg, WorkspaceID: ws.ID,
		ApprovalID: ids.MustNew(ids.Approval), ApprovedDiffHash: prop.ActualDiff.DiffHash,
	}, workspace.ApplyOptions{MaxBytes: 1024})
	if err == nil || codeOf(err) != protoerr.InvalidChangeSet {
		t.Fatalf("reapply err=%v", err)
	}
}

func TestApplyChanges_StaleFileAndDiffMismatch(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), "one\n")
	reg := workspace.NewRegistry()
	store := workspace.NewProposalStore()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	hash := workspace.HashBytes([]byte("one\n"))
	chg := ids.MustNew(ids.ChangeSet)
	prop, err := ws.Propose(messages.ProposedChangeSet{
		ChangeSetID: chg, WorkspaceID: ws.ID,
		Changes: []messages.FileChange{{
			Path: "a.txt", Operation: messages.OpModify, BaseHash: hash, Content: "two\n",
		}},
	}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	store.Put(prop)

	mustWrite(t, filepath.Join(root, "a.txt"), "changed\n")
	_, err = workspace.ApplyChanges(reg, store, policy.LocalApproval{}, workspace.ApplyRequest{
		ChangeSetID: chg, WorkspaceID: ws.ID,
		ApprovalID: ids.MustNew(ids.Approval), ApprovedDiffHash: prop.ActualDiff.DiffHash,
	}, workspace.ApplyOptions{MaxBytes: 1024})
	if err == nil || (codeOf(err) != protoerr.WorkspaceFileChanged && codeOf(err) != protoerr.FileHashMismatch) {
		t.Fatalf("stale err=%v code=%v", err, codeOf(err))
	}

	// restore for diff mismatch path using a fresh proposal
	mustWrite(t, filepath.Join(root, "a.txt"), "one\n")
	chg2 := ids.MustNew(ids.ChangeSet)
	prop2, err := ws.Propose(messages.ProposedChangeSet{
		ChangeSetID: chg2, WorkspaceID: ws.ID,
		Changes: []messages.FileChange{{
			Path: "a.txt", Operation: messages.OpModify, BaseHash: hash, Content: "two\n",
		}},
	}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	store.Put(prop2)
	_, err = workspace.ApplyChanges(reg, store, policy.LocalApproval{}, workspace.ApplyRequest{
		ChangeSetID: chg2, WorkspaceID: ws.ID,
		ApprovalID: ids.MustNew(ids.Approval), ApprovedDiffHash: "sha256:" + "00",
	}, workspace.ApplyOptions{MaxBytes: 1024})
	if err == nil || codeOf(err) != protoerr.DiffMismatch {
		// LocalApproval may fail first on empty-looking hash - "sha256:00" is non-empty
		if codeOf(err) != protoerr.DiffMismatch {
			t.Fatalf("diff mismatch err=%v code=%v", err, codeOf(err))
		}
	}
}

func TestApplyChanges_InvalidApproval(t *testing.T) {
	root := t.TempDir()
	reg := workspace.NewRegistry()
	store := workspace.NewProposalStore()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	chg := ids.MustNew(ids.ChangeSet)
	prop, err := ws.Propose(messages.ProposedChangeSet{
		ChangeSetID: chg, WorkspaceID: ws.ID,
		Changes: []messages.FileChange{{Path: "n.txt", Operation: messages.OpCreate, Content: "x"}},
	}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	store.Put(prop)
	_, err = workspace.ApplyChanges(reg, store, policy.LocalApproval{}, workspace.ApplyRequest{
		ChangeSetID: chg, WorkspaceID: ws.ID,
		ApprovalID: "bad", ApprovedDiffHash: prop.ActualDiff.DiffHash,
	}, workspace.ApplyOptions{MaxBytes: 1024})
	if err == nil || codeOf(err) != protoerr.ApprovalInvalid {
		t.Fatalf("err=%v", err)
	}
}

func TestApplyChanges_RollbackOnForcedFailure(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), "A\n")
	mustWrite(t, filepath.Join(root, "b.txt"), "B\n")
	reg := workspace.NewRegistry()
	store := workspace.NewProposalStore()
	ws, err := reg.Register("", root)
	if err != nil {
		t.Fatal(err)
	}
	ha := workspace.HashBytes([]byte("A\n"))
	hb := workspace.HashBytes([]byte("B\n"))
	chg := ids.MustNew(ids.ChangeSet)
	prop, err := ws.Propose(messages.ProposedChangeSet{
		ChangeSetID: chg, WorkspaceID: ws.ID,
		Changes: []messages.FileChange{
			{Path: "a.txt", Operation: messages.OpModify, BaseHash: ha, Content: "A2\n"},
			{Path: "b.txt", Operation: messages.OpModify, BaseHash: hb, Content: "B2\n"},
		},
	}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	store.Put(prop)

	_, err = workspace.ApplyChanges(reg, store, policy.LocalApproval{}, workspace.ApplyRequest{
		ChangeSetID: chg, WorkspaceID: ws.ID,
		ApprovalID: ids.MustNew(ids.Approval), ApprovedDiffHash: prop.ActualDiff.DiffHash,
	}, workspace.ApplyOptions{MaxBytes: 1024, FailAfter: 1})
	if err == nil || codeOf(err) != protoerr.ApplyFailed {
		t.Fatalf("err=%v", err)
	}
	ra, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	rb, _ := os.ReadFile(filepath.Join(root, "b.txt"))
	if string(ra) != "A\n" || string(rb) != "B\n" {
		t.Fatalf("rollback failed a=%q b=%q", ra, rb)
	}
	// claim aborted; can apply again
	res, err := workspace.ApplyChanges(reg, store, policy.LocalApproval{}, workspace.ApplyRequest{
		ChangeSetID: chg, WorkspaceID: ws.ID,
		ApprovalID: ids.MustNew(ids.Approval), ApprovedDiffHash: prop.ActualDiff.DiffHash,
	}, workspace.ApplyOptions{MaxBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	if res.WorkspaceRevision != 1 {
		t.Fatalf("rev=%d", res.WorkspaceRevision)
	}
}
