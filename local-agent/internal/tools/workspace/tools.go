package workspacetool

import (
	"context"
	"encoding/json"
	"strings"

	"oncode/local-agent/internal/policy"
	"oncode/local-agent/internal/tools"
	"oncode/local-agent/internal/workspace"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/ids"
	"oncode/protocol/messages"
)

const (
	ToolReadFile       = "workspace.read_file"
	ToolReadFiles      = "workspace.read_files"
	ToolSearch         = "workspace.search"
	ToolProposeChanges = "workspace.propose_changes"
	ToolApplyChanges   = "workspace.apply_changes"
)

type Deps struct {
	Workspaces     *workspace.Registry
	Proposals      *workspace.ProposalStore
	Approval       policy.ApprovalValidator
	MaxFileBytes   int64
	SearchExcludes []string
	MaxSearchHits  int
}

func Register(reg *tools.Registry, deps Deps) {
	reg.Register(ToolReadFile, func(ctx context.Context, req *messages.ToolRequestPayload) (*messages.ToolResponsePayload, error) {
		workspaceID, _ := req.Arguments["workspace_id"].(string)
		path, _ := req.Arguments["path"].(string)
		if workspaceID == "" {
			return fail(req.CallID, protoerr.InvalidArgument, "workspace_id required"), nil
		}
		if path == "" {
			return fail(req.CallID, protoerr.InvalidArgument, "path required"), nil
		}
		ws, err := deps.Workspaces.Get(workspaceID)
		if err != nil {
			return nil, err
		}
		result, err := ws.ReadTextFile(path, deps.MaxFileBytes)
		if err != nil {
			return nil, err
		}
		return tools.Success(req.CallID, result), nil
	})

	reg.Register(ToolReadFiles, func(ctx context.Context, req *messages.ToolRequestPayload) (*messages.ToolResponsePayload, error) {
		workspaceID, _ := req.Arguments["workspace_id"].(string)
		if workspaceID == "" {
			return fail(req.CallID, protoerr.InvalidArgument, "workspace_id required"), nil
		}
		paths, err := stringSliceArg(req.Arguments["paths"])
		if err != nil {
			return fail(req.CallID, protoerr.InvalidArgument, err.Error()), nil
		}
		ws, err := deps.Workspaces.Get(workspaceID)
		if err != nil {
			return nil, err
		}
		result, err := ws.ReadTextFiles(paths, deps.MaxFileBytes)
		if err != nil {
			return nil, err
		}
		raw, mErr := json.Marshal(result)
		if mErr != nil {
			return fail(req.CallID, protoerr.InternalError, "marshal result"), nil
		}
		status := result.OverallStatus()
		resp := &messages.ToolResponsePayload{
			CallID: req.CallID,
			Status: status,
			Result: raw,
		}
		if status == messages.StatusFailed && result.Failed == result.Total && result.Total > 0 {
			// Aggregate failure: surface first entry error when every path failed.
			if result.Files[0].Error != nil {
				resp.Error = result.Files[0].Error
			} else {
				e, _ := protoerr.New(protoerr.WorkspaceFileNotFound, "all paths failed")
				resp.Error = e
			}
		}
		return resp, nil
	})

	reg.Register(ToolSearch, func(ctx context.Context, req *messages.ToolRequestPayload) (*messages.ToolResponsePayload, error) {
		workspaceID, _ := req.Arguments["workspace_id"].(string)
		query, _ := req.Arguments["query"].(string)
		modeStr, _ := req.Arguments["mode"].(string)
		if workspaceID == "" {
			return fail(req.CallID, protoerr.InvalidArgument, "workspace_id required"), nil
		}
		ws, err := deps.Workspaces.Get(workspaceID)
		if err != nil {
			return nil, err
		}
		opts := workspace.DefaultSearchOptions(query, workspace.SearchMode(modeStr))
		opts.MaxFileBytes = deps.MaxFileBytes
		if len(deps.SearchExcludes) > 0 {
			opts.ExcludeDirs = deps.SearchExcludes
		}
		if deps.MaxSearchHits > 0 {
			opts.MaxResults = deps.MaxSearchHits
		}
		result, err := ws.Search(opts)
		if err != nil {
			return nil, err
		}
		return tools.Success(req.CallID, result), nil
	})

	reg.Register(ToolProposeChanges, func(ctx context.Context, req *messages.ToolRequestPayload) (*messages.ToolResponsePayload, error) {
		cs, err := parseProposedChangeSet(req.Arguments, deps.Workspaces, deps.MaxFileBytes)
		if err != nil {
			if _, ok := err.(*protoerr.Error); ok {
				return nil, err
			}
			return fail(req.CallID, protoerr.InvalidArgument, err.Error()), nil
		}
		ws, err := deps.Workspaces.Get(cs.WorkspaceID)
		if err != nil {
			return nil, err
		}
		prop, err := ws.Propose(cs, deps.MaxFileBytes)
		if err != nil {
			return nil, err
		}
		if deps.Proposals != nil {
			deps.Proposals.Put(prop)
		}
		out := workspace.ProposeResult{
			ChangeSetID:           prop.ChangeSet.ChangeSetID,
			WorkspaceID:           prop.ChangeSet.WorkspaceID,
			BaseWorkspaceRevision: prop.ChangeSet.BaseWorkspaceRevision,
			Diff:                  prop.ActualDiff.Diff,
			DiffHash:              prop.ActualDiff.DiffHash,
			Changes:               prop.ChangeSet.Changes,
		}
		return tools.Success(req.CallID, out), nil
	})

	reg.Register(ToolApplyChanges, func(ctx context.Context, req *messages.ToolRequestPayload) (*messages.ToolResponsePayload, error) {
		changeSetID, _ := req.Arguments["change_set_id"].(string)
		workspaceID, _ := req.Arguments["workspace_id"].(string)
		approvalID, _ := req.Arguments["approval_id"].(string)
		diffHash, _ := req.Arguments["approved_diff_hash"].(string)
		if changeSetID == "" || workspaceID == "" {
			return fail(req.CallID, protoerr.InvalidArgument, "change_set_id and workspace_id required"), nil
		}
		approval := deps.Approval
		if approval == nil {
			approval = policy.LocalApproval{}
		}
		result, err := workspace.ApplyChanges(deps.Workspaces, deps.Proposals, approval, workspace.ApplyRequest{
			ChangeSetID:      changeSetID,
			WorkspaceID:      workspaceID,
			ApprovalID:       approvalID,
			ApprovedDiffHash: diffHash,
		}, workspace.ApplyOptions{MaxBytes: deps.MaxFileBytes})
		if err != nil {
			return nil, err
		}
		return tools.Success(req.CallID, result), nil
	})
}

func parseProposedChangeSet(args map[string]any, workspaces *workspace.Registry, maxBytes int64) (messages.ProposedChangeSet, error) {
	if args == nil {
		return messages.ProposedChangeSet{}, argError("arguments required")
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return messages.ProposedChangeSet{}, argError("invalid arguments")
	}
	var cs messages.ProposedChangeSet
	if err := json.Unmarshal(raw, &cs); err != nil {
		return messages.ProposedChangeSet{}, argError("invalid proposed change set")
	}
	if len(cs.Changes) == 0 {
		path, _ := args["path"].(string)
		if path != "" {
			op, _ := args["operation"].(string)
			content, _ := args["content"].(string)
			cs.Changes = []messages.FileChange{{
				Path:      path,
				Operation: messages.ChangeOperation(op),
				Content:   content,
			}}
		}
	}
	ws, err := resolveProposeWorkspace(workspaces, cs.WorkspaceID)
	if err != nil {
		return messages.ProposedChangeSet{}, err
	}
	cs.WorkspaceID = ws.ID
	if ids.Validate(ids.ChangeSet, cs.ChangeSetID) != nil {
		cs.ChangeSetID = ids.MustNew(ids.ChangeSet)
	}
	if err := fillBaseHashes(ws, cs.Changes, maxBytes); err != nil {
		return messages.ProposedChangeSet{}, err
	}
	return cs, nil
}

// resolveProposeWorkspace는 유효한 WS ID면 Get, 아니면 등록이 하나일 때 그걸 쓴다.
func resolveProposeWorkspace(workspaces *workspace.Registry, workspaceID string) (*workspace.Workspace, error) {
	if workspaces == nil {
		return nil, argError("workspace registry is required")
	}
	if ids.Validate(ids.Workspace, workspaceID) == nil {
		return workspaces.Get(workspaceID)
	}
	return workspaces.Sole()
}

// fillBaseHashes는 MODIFY/DELETE/RENAME에 현재 파일 해시를 넣는다. 없으면 Propose가 거절한다.
func fillBaseHashes(ws *workspace.Workspace, changes []messages.FileChange, maxBytes int64) error {
	for i := range changes {
		op := messages.ChangeOperation(strings.ToUpper(string(changes[i].Operation)))
		if op == "" {
			op = messages.OpModify
		}
		changes[i].Operation = op
		if op != messages.OpModify && op != messages.OpDelete && op != messages.OpRename {
			continue
		}
		if changes[i].BaseHash != "" {
			continue
		}
		res, err := ws.ReadTextFile(changes[i].Path, maxBytes)
		if err != nil {
			return err
		}
		changes[i].BaseHash = res.Hash
	}
	return nil
}

func stringSliceArg(v any) ([]string, error) {
	if v == nil {
		return nil, errPathsRequired()
	}
	switch t := v.(type) {
	case []string:
		if len(t) == 0 {
			return nil, errPathsRequired()
		}
		return t, nil
	case []any:
		if len(t) == 0 {
			return nil, errPathsRequired()
		}
		out := make([]string, 0, len(t))
		for i, item := range t {
			s, ok := item.(string)
			if !ok {
				return nil, invalidPathElem(i)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, errPathsRequired()
	}
}

type argError string

func (e argError) Error() string { return string(e) }

func errPathsRequired() error { return argError("paths required") }
func invalidPathElem(i int) error {
	_ = i
	return argError("paths elements must be strings")
}

func fail(callID string, code protoerr.Code, msg string) *messages.ToolResponsePayload {
	e, err := protoerr.New(code, msg)
	if err != nil {
		e, _ = protoerr.New(protoerr.InternalError, msg)
	}
	return &messages.ToolResponsePayload{
		CallID: callID,
		Status: messages.StatusFailed,
		Error:  e,
	}
}
