package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"oncode/local-agent/internal/diff"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/messages"
)

// Proposal is a validated, non-applied change set with Local Agent actual diff.
type Proposal struct {
	ChangeSet    messages.ProposedChangeSet
	ActualDiff   messages.ActualDiff
	BaseRevision int64
	CreatedAt    time.Time
	Planned      map[string]PlannedFile
	Applied      bool
	Applying     bool
}

// PlannedFile is the intended end state of a path.
type PlannedFile struct {
	Delete  bool
	Content string
}

// ProposalStore keeps proposals until apply (Phase 1 in-memory).
type ProposalStore struct {
	mu   sync.RWMutex
	byID map[string]*Proposal
}

func NewProposalStore() *ProposalStore {
	return &ProposalStore{byID: make(map[string]*Proposal)}
}

func (s *ProposalStore) Put(p *Proposal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *p
	planned := make(map[string]PlannedFile, len(p.Planned))
	for k, v := range p.Planned {
		planned[k] = v
	}
	cp.Planned = planned
	s.byID[p.ChangeSet.ChangeSetID] = &cp
}

func (s *ProposalStore) Get(changeSetID string) (*Proposal, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byID[changeSetID]
	if !ok {
		return nil, false
	}
	cp := *p
	planned := make(map[string]PlannedFile, len(p.Planned))
	for k, v := range p.Planned {
		planned[k] = v
	}
	cp.Planned = planned
	return &cp, true
}

func (s *ProposalStore) MarkApplied(changeSetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.byID[changeSetID]
	if !ok {
		return mustErr(protoerr.ChangeSetNotFound, "change set not found")
	}
	if p.Applied {
		return mustErr(protoerr.InvalidChangeSet, "change set already applied")
	}
	p.Applied = true
	p.Applying = false
	return nil
}

// ClaimForApply marks a proposal as in-flight apply. Caller must FinishApply or AbortApply.
func (s *ProposalStore) ClaimForApply(changeSetID string) (*Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.byID[changeSetID]
	if !ok {
		return nil, mustErr(protoerr.ChangeSetNotFound, "change set not found")
	}
	if p.Applied {
		return nil, mustErr(protoerr.InvalidChangeSet, "change set already applied")
	}
	if p.Applying {
		return nil, mustErr(protoerr.InvalidChangeSet, "change set apply already in progress")
	}
	p.Applying = true
	cp := *p
	planned := make(map[string]PlannedFile, len(p.Planned))
	for k, v := range p.Planned {
		planned[k] = v
	}
	cp.Planned = planned
	return &cp, nil
}

func (s *ProposalStore) AbortApply(changeSetID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.byID[changeSetID]; ok && !p.Applied {
		p.Applying = false
	}
}

func (s *ProposalStore) FinishApply(changeSetID string) error {
	return s.MarkApplied(changeSetID)
}

// ProposeResult is returned to the tool caller (files are not mutated).
type ProposeResult struct {
	ChangeSetID           string                `json:"change_set_id"`
	WorkspaceID           string                `json:"workspace_id"`
	BaseWorkspaceRevision string                `json:"base_workspace_revision,omitempty"`
	Diff                  string                `json:"diff"`
	DiffHash              string                `json:"diff_hash"`
	Changes               []messages.FileChange `json:"changes"`
}

// Propose validates changes against the current workspace and builds an actual diff.
// It does not mutate workspace files.
func (w *Workspace) Propose(cs messages.ProposedChangeSet, maxBytes int64) (*Proposal, error) {
	if w == nil {
		return nil, mustErr(protoerr.WorkspaceNotFound, "workspace is nil")
	}
	if err := cs.Validate(); err != nil {
		return nil, mustErr(protoerr.InvalidChangeSet, err.Error())
	}
	if cs.WorkspaceID != "" && cs.WorkspaceID != w.ID {
		return nil, mustErr(protoerr.InvalidArgument, "workspace_id mismatch")
	}
	cs.WorkspaceID = w.ID

	if cs.BaseWorkspaceRevision != "" {
		want, err := strconv.ParseInt(cs.BaseWorkspaceRevision, 10, 64)
		if err != nil {
			return nil, mustErr(protoerr.InvalidArgument, "invalid base_workspace_revision")
		}
		if want != w.CurrentRevision {
			return nil, mustErr(protoerr.WorkspaceStale, fmt.Sprintf("workspace revision %d != base %d", w.CurrentRevision, want))
		}
	}

	if err := validateChangeConflicts(cs.Changes); err != nil {
		return nil, err
	}

	planned := map[string]PlannedFile{}
	type fileDiff struct {
		path string
		text string
	}
	var parts []fileDiff

	for _, ch := range cs.Changes {
		path := filepath.ToSlash(filepath.Clean(ch.Path))
		switch ch.Operation {
		case messages.OpCreate:
			if _, err := w.ResolvePath(path); err != nil {
				return nil, err
			}
			if existsFile(w, path) {
				return nil, mustErr(protoerr.InvalidChangeSet, "CREATE path already exists: "+path)
			}
			parts = append(parts, fileDiff{path: path, text: diff.UnifiedFile(path, "", ch.Content)})
			planned[path] = PlannedFile{Content: ch.Content}

		case messages.OpModify:
			old, err := w.readExistingForChange(path, ch.BaseHash, maxBytes)
			if err != nil {
				return nil, err
			}
			parts = append(parts, fileDiff{path: path, text: diff.UnifiedFile(path, old, ch.Content)})
			planned[path] = PlannedFile{Content: ch.Content}

		case messages.OpDelete:
			old, err := w.readExistingForChange(path, ch.BaseHash, maxBytes)
			if err != nil {
				return nil, err
			}
			parts = append(parts, fileDiff{path: path, text: diff.UnifiedFile(path, old, "")})
			planned[path] = PlannedFile{Delete: true}

		case messages.OpRename:
			target := filepath.ToSlash(filepath.Clean(ch.TargetPath))
			if _, err := w.ResolvePath(target); err != nil {
				return nil, err
			}
			if existsFile(w, target) {
				return nil, mustErr(protoerr.InvalidChangeSet, "RENAME target already exists: "+target)
			}
			old, err := w.readExistingForChange(path, ch.BaseHash, maxBytes)
			if err != nil {
				return nil, err
			}
			newContent := ch.Content
			if newContent == "" {
				newContent = old
			}
			parts = append(parts, fileDiff{path: path, text: diff.UnifiedFile(path, old, "")})
			parts = append(parts, fileDiff{path: target, text: diff.UnifiedFile(target, "", newContent)})
			planned[path] = PlannedFile{Delete: true}
			planned[target] = PlannedFile{Content: newContent}

		default:
			return nil, mustErr(protoerr.InvalidChangeSet, "invalid operation")
		}
	}

	sort.SliceStable(parts, func(i, j int) bool { return parts[i].path < parts[j].path })
	diffParts := make([]string, 0, len(parts))
	for _, p := range parts {
		diffParts = append(diffParts, p.text)
	}
	diffText := diff.UnifiedJoin(diffParts)
	diffHash := HashBytes([]byte(diffText))

	return &Proposal{
		ChangeSet:    cs,
		BaseRevision: w.CurrentRevision,
		CreatedAt:    time.Now().UTC(),
		Planned:      planned,
		ActualDiff: messages.ActualDiff{
			ChangeSetID: cs.ChangeSetID,
			WorkspaceID: w.ID,
			Diff:        diffText,
			DiffHash:    diffHash,
		},
	}, nil
}

func (w *Workspace) readExistingForChange(relPath, baseHash string, maxBytes int64) (string, error) {
	res, err := w.ReadTextFile(relPath, maxBytes)
	if err != nil {
		return "", err
	}
	if baseHash != "" && res.Hash != baseHash {
		return "", mustErr(protoerr.FileHashMismatch, fmt.Sprintf("base_hash mismatch for %s", relPath))
	}
	return res.Content, nil
}

func existsFile(w *Workspace, relPath string) bool {
	abs, err := w.ResolvePath(relPath)
	if err != nil {
		return false
	}
	fi, err := os.Lstat(abs)
	if err != nil {
		return false
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(abs)
		if err != nil || !withinRoot(w.RootPath, target) {
			return false
		}
		fi, err = os.Stat(target)
		if err != nil {
			return false
		}
	}
	return !fi.IsDir()
}

func validateChangeConflicts(changes []messages.FileChange) error {
	seenPath := map[string]messages.ChangeOperation{}
	seenTarget := map[string]struct{}{}
	for _, ch := range changes {
		p := filepath.ToSlash(filepath.Clean(ch.Path))
		if prev, ok := seenPath[p]; ok {
			return mustErr(protoerr.InvalidChangeSet, fmt.Sprintf("duplicate path %s (%s and %s)", p, prev, ch.Operation))
		}
		seenPath[p] = ch.Operation
		if ch.Operation == messages.OpRename {
			t := filepath.ToSlash(filepath.Clean(ch.TargetPath))
			if t == p {
				return mustErr(protoerr.InvalidChangeSet, "RENAME target_path must differ from path")
			}
			if _, ok := seenTarget[t]; ok {
				return mustErr(protoerr.InvalidChangeSet, "duplicate target_path "+t)
			}
			if _, ok := seenPath[t]; ok {
				return mustErr(protoerr.InvalidChangeSet, "target_path conflicts with another change path "+t)
			}
			seenTarget[t] = struct{}{}
		}
	}
	for _, ch := range changes {
		if ch.Operation != messages.OpCreate {
			continue
		}
		p := filepath.ToSlash(filepath.Clean(ch.Path))
		if _, ok := seenTarget[p]; ok {
			return mustErr(protoerr.InvalidChangeSet, "CREATE path conflicts with RENAME target "+p)
		}
	}
	return nil
}
