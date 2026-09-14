package gittool

import (
	"context"

	"oncode/local-agent/internal/gitops"
	"oncode/local-agent/internal/tools"
	"oncode/local-agent/internal/workspace"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/messages"
)

const (
	ToolStatus = "git.status"
	ToolDiff   = "git.diff"
)

type Deps struct {
	Workspaces *workspace.Registry
}

func Register(reg *tools.Registry, deps Deps) {
	reg.Register(ToolStatus, func(ctx context.Context, req *messages.ToolRequestPayload) (*messages.ToolResponsePayload, error) {
		workspaceID, _ := req.Arguments["workspace_id"].(string)
		if workspaceID == "" {
			return fail(req.CallID, protoerr.InvalidArgument, "workspace_id required"), nil
		}
		ws, err := deps.Workspaces.Get(workspaceID)
		if err != nil {
			return nil, err
		}
		result, err := gitops.Status(ctx, ws.RootPath)
		if err != nil {
			return nil, err
		}
		return tools.Success(req.CallID, result), nil
	})

	reg.Register(ToolDiff, func(ctx context.Context, req *messages.ToolRequestPayload) (*messages.ToolResponsePayload, error) {
		workspaceID, _ := req.Arguments["workspace_id"].(string)
		if workspaceID == "" {
			return fail(req.CallID, protoerr.InvalidArgument, "workspace_id required"), nil
		}
		path, _ := req.Arguments["path"].(string)
		staged := false
		switch v := req.Arguments["staged"].(type) {
		case bool:
			staged = v
		}
		ws, err := deps.Workspaces.Get(workspaceID)
		if err != nil {
			return nil, err
		}
		if path != "" {
			if _, err := ws.ResolvePath(path); err != nil {
				return nil, err
			}
		}
		result, err := gitops.Diff(ctx, ws.RootPath, path, staged)
		if err != nil {
			return nil, err
		}
		return tools.Success(req.CallID, result), nil
	})
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
