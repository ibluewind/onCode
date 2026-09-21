package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	protoerr "oncode/protocol/errors"
	"oncode/protocol/ids"
)

// Workspace is a registered local workspace root.
type Workspace struct {
	ID              string
	ProjectID       string
	RootPath        string // absolute, canonical
	RegisteredAt    time.Time
	CurrentRevision int64
}

// Registry tracks registered workspaces in memory (Phase 1 persistence can follow).
type Registry struct {
	mu   sync.RWMutex
	byID map[string]*Workspace
}

func NewRegistry() *Registry {
	return &Registry{byID: make(map[string]*Workspace)}
}

// Register adds a workspace rooted at rootPath. Returns the workspace record.
func (r *Registry) Register(projectID, rootPath string) (*Workspace, error) {
	abs, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, wrap(protoerr.InvalidPath, "resolve workspace root", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, wrap(protoerr.InvalidPath, "workspace root not found", err)
	}
	if !info.IsDir() {
		return nil, mustErr(protoerr.InvalidPath, "workspace root must be a directory")
	}
	canon, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, wrap(protoerr.InvalidPath, "canonicalize workspace root", err)
	}
	wid, err := ids.New(ids.Workspace)
	if err != nil {
		return nil, err
	}
	ws := &Workspace{
		ID:              wid,
		ProjectID:       projectID,
		RootPath:        canon,
		RegisteredAt:    time.Now().UTC(),
		CurrentRevision: 0,
	}
	r.mu.Lock()
	r.byID[wid] = ws
	r.mu.Unlock()
	return ws, nil
}

// Sole은 등록된 워크스페이스가 하나일 때만 그 복사본을 돌려준다.
// 없거나 둘 이상이면 WORKSPACE_NOT_FOUND다. 서버가 ws-local을 보내도 등록 루트를 고른다.
func (r *Registry) Sole() (*Workspace, error) {
	if r == nil {
		return nil, mustErr(protoerr.WorkspaceNotFound, "workspace registry is nil")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.byID) != 1 {
		return nil, mustErr(protoerr.WorkspaceNotFound, "workspace_id required unless exactly one workspace is registered")
	}
	for _, ws := range r.byID {
		cp := *ws
		return &cp, nil
	}
	return nil, mustErr(protoerr.WorkspaceNotFound, "workspace not registered")
}

// Get returns a registered workspace copy or WORKSPACE_NOT_FOUND.
func (r *Registry) Get(workspaceID string) (*Workspace, error) {
	if err := ids.Validate(ids.Workspace, workspaceID); err != nil {
		return nil, mustErr(protoerr.InvalidArgument, "invalid workspace_id")
	}
	r.mu.RLock()
	ws, ok := r.byID[workspaceID]
	r.mu.RUnlock()
	if !ok {
		return nil, mustErr(protoerr.WorkspaceNotFound, "workspace not registered")
	}
	cp := *ws
	return &cp, nil
}

// Mutate runs fn against the live workspace record under a write lock.
func (r *Registry) Mutate(workspaceID string, fn func(*Workspace) error) error {
	if err := ids.Validate(ids.Workspace, workspaceID); err != nil {
		return mustErr(protoerr.InvalidArgument, "invalid workspace_id")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	ws, ok := r.byID[workspaceID]
	if !ok {
		return mustErr(protoerr.WorkspaceNotFound, "workspace not registered")
	}
	return fn(ws)
}

func wrap(code protoerr.Code, msg string, err error) error {
	e, nerr := protoerr.New(code, fmt.Sprintf("%s: %v", msg, err))
	if nerr != nil {
		return nerr
	}
	return e
}

func mustErr(code protoerr.Code, msg string) error {
	e, err := protoerr.New(code, msg)
	if err != nil {
		return err
	}
	return e
}
