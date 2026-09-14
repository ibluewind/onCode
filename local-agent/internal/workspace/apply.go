package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"oncode/local-agent/internal/policy"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/ids"
)

// ApplyRequest is the input for workspace.apply_changes.
type ApplyRequest struct {
	ChangeSetID      string
	WorkspaceID      string
	ApprovalID       string
	ApprovedDiffHash string
}

// ApplyFileResult describes one path after a successful apply.
type ApplyFileResult struct {
	Path    string `json:"path"`
	Hash    string `json:"hash,omitempty"`
	Deleted bool   `json:"deleted,omitempty"`
}

// ApplyResult is returned after a successful atomic apply.
type ApplyResult struct {
	ChangeSetID       string           `json:"change_set_id"`
	WorkspaceID       string           `json:"workspace_id"`
	WorkspaceRevision int64            `json:"workspace_revision"`
	Files             []ApplyFileResult `json:"files"`
}

// ApplyOptions controls apply behavior. FailAfter is a test hook (0 = disabled).
type ApplyOptions struct {
	MaxBytes  int64
	FailAfter int
}

type backupEntry struct {
	path    string
	abs     string
	existed bool
	data    []byte
	mode    os.FileMode
}

// ApplyChanges loads a stored proposal, revalidates, and atomically applies it.
func ApplyChanges(
	reg *Registry,
	store *ProposalStore,
	approval policy.ApprovalValidator,
	req ApplyRequest,
	opts ApplyOptions,
) (*ApplyResult, error) {
	if reg == nil || store == nil {
		return nil, mustErr(protoerr.InternalError, "registry/store required")
	}
	if approval == nil {
		approval = policy.LocalApproval{}
	}
	if err := ids.Validate(ids.ChangeSet, req.ChangeSetID); err != nil {
		return nil, mustErr(protoerr.InvalidArgument, "invalid change_set_id")
	}
	if err := approval.Validate(req.ApprovalID, req.ChangeSetID, req.ApprovedDiffHash); err != nil {
		return nil, err
	}

	prop, err := store.ClaimForApply(req.ChangeSetID)
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			store.AbortApply(req.ChangeSetID)
		}
	}()

	if prop.ChangeSet.WorkspaceID != req.WorkspaceID {
		return nil, mustErr(protoerr.InvalidArgument, "workspace_id mismatch")
	}
	if req.ApprovedDiffHash != prop.ActualDiff.DiffHash {
		return nil, mustErr(protoerr.DiffMismatch, "approved_diff_hash does not match proposal")
	}

	var result *ApplyResult
	err = reg.Mutate(req.WorkspaceID, func(ws *Workspace) error {
		// Revalidate against current disk by rebuilding the proposal.
		fresh, err := ws.Propose(prop.ChangeSet, opts.MaxBytes)
		if err != nil {
			if pe, ok := err.(*protoerr.Error); ok && pe.Code == protoerr.FileHashMismatch {
				return mustErr(protoerr.WorkspaceFileChanged, pe.Message)
			}
			return err
		}
		if fresh.ActualDiff.DiffHash != prop.ActualDiff.DiffHash {
			return mustErr(protoerr.DiffMismatch, "workspace changed; diff identity mismatch")
		}

		paths := make([]string, 0, len(prop.Planned))
		for p := range prop.Planned {
			paths = append(paths, p)
		}
		sort.Strings(paths)

		backups := make([]backupEntry, 0, len(paths))
		applied := 0
		rollback := func() {
			for i := len(backups) - 1; i >= 0; i-- {
				b := backups[i]
				if b.existed {
					_ = os.MkdirAll(filepath.Dir(b.abs), 0o755)
					_ = os.WriteFile(b.abs, b.data, b.mode)
				} else {
					_ = os.Remove(b.abs)
				}
			}
		}

		for _, rel := range paths {
			plan := prop.Planned[rel]
			abs, err := ws.ResolvePath(rel)
			if err != nil {
				rollback()
				return err
			}
			b := backupEntry{path: rel, abs: abs}
			if fi, err := os.Lstat(abs); err == nil && fi.Mode()&os.ModeSymlink == 0 && !fi.IsDir() {
				data, rerr := os.ReadFile(abs)
				if rerr != nil {
					rollback()
					return wrap(protoerr.ApplyFailed, "backup read", rerr)
				}
				b.existed = true
				b.data = data
				b.mode = fi.Mode().Perm()
			} else if err == nil && fi.Mode()&os.ModeSymlink != 0 {
				rollback()
				return mustErr(protoerr.ApplyFailed, "refusing to apply over symlink: "+rel)
			}
			backups = append(backups, b)

			if plan.Delete {
				if b.existed {
					if err := os.Remove(abs); err != nil {
						rollback()
						return wrap(protoerr.ApplyFailed, "delete", err)
					}
				}
			} else {
				if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
					rollback()
					return wrap(protoerr.ApplyFailed, "mkdir", err)
				}
				tmp := abs + ".oncode-tmp"
				if err := os.WriteFile(tmp, []byte(plan.Content), 0o644); err != nil {
					rollback()
					_ = os.Remove(tmp)
					return wrap(protoerr.ApplyFailed, "stage write", err)
				}
				if err := replaceFile(tmp, abs); err != nil {
					_ = os.Remove(tmp)
					rollback()
					return wrap(protoerr.ApplyFailed, "replace into place", err)
				}
			}

			applied++
			if opts.FailAfter > 0 && applied >= opts.FailAfter {
				rollback()
				return mustErr(protoerr.ApplyFailed, fmt.Sprintf("forced failure after %d ops", opts.FailAfter))
			}
		}

		files := make([]ApplyFileResult, 0, len(paths))
		for _, rel := range paths {
			plan := prop.Planned[rel]
			if plan.Delete {
				files = append(files, ApplyFileResult{Path: rel, Deleted: true})
				continue
			}
			files = append(files, ApplyFileResult{Path: rel, Hash: HashBytes([]byte(plan.Content))})
		}

		ws.CurrentRevision++
		result = &ApplyResult{
			ChangeSetID:       req.ChangeSetID,
			WorkspaceID:       ws.ID,
			WorkspaceRevision: ws.CurrentRevision,
			Files:             files,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := store.FinishApply(req.ChangeSetID); err != nil {
		return nil, err
	}
	success = true
	return result, nil
}

// replaceFile moves tmp onto dest. On Windows the destination is removed first
// because os.Rename cannot replace an existing file.
func replaceFile(tmp, dest string) error {
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmp, dest)
}
