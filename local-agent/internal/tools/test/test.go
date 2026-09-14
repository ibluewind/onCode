package testtool

import (
	"context"

	"oncode/local-agent/internal/maven"
	"oncode/local-agent/internal/testrun"
	"oncode/local-agent/internal/tools"
	"oncode/local-agent/internal/workspace"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/messages"
)

const ToolRun = "test.run"

type Deps struct {
	Workspaces *workspace.Registry
	Adapters   []testrun.Adapter
}

func Register(reg *tools.Registry, deps Deps) {
	adapters := deps.Adapters
	if len(adapters) == 0 {
		adapters = []testrun.Adapter{maven.TestAdapter{}}
	}
	reg.Register(ToolRun, func(ctx context.Context, req *messages.ToolRequestPayload) (*messages.ToolResponsePayload, error) {
		workspaceID, _ := req.Arguments["workspace_id"].(string)
		if workspaceID == "" {
			return fail(req.CallID, protoerr.InvalidArgument, "workspace_id required"), nil
		}
		ws, err := deps.Workspaces.Get(workspaceID)
		if err != nil {
			return nil, err
		}
		timeoutMS := 0
		if opts := req.Options; opts != nil {
			timeoutMS = opts.TimeoutMS
		}
		var adapter testrun.Adapter
		for _, a := range adapters {
			if a.Detect(ws.RootPath) {
				adapter = a
				break
			}
		}
		if adapter == nil {
			return fail(req.CallID, protoerr.BuildToolNotFound, "no test adapter detected for workspace"), nil
		}
		result, err := adapter.Run(ctx, testrun.Request{
			WorkspaceRoot: ws.RootPath,
			TimeoutMS:     timeoutMS,
		})
		if err != nil {
			return nil, err
		}
		resp := tools.Success(req.CallID, result)
		if result.Status != "SUCCESS" {
			resp.Status = mapStatus(result.Status)
			switch result.Status {
			case "TIMEOUT":
				e, _ := protoerr.New(protoerr.ProcessTimeout, "test timed out")
				resp.Error = e
			case "CANCELLED":
				e, _ := protoerr.New(protoerr.ProcessCancelled, "test cancelled")
				resp.Error = e
			default:
				e, _ := protoerr.New(protoerr.TestFailed, "tests failed")
				resp.Error = e
			}
		}
		if resp.Metrics == nil {
			resp.Metrics = &messages.Metrics{DurationMS: result.DurationMS}
		}
		return resp, nil
	})
}

func mapStatus(s string) string {
	switch s {
	case "TIMEOUT":
		return messages.StatusTimeout
	case "CANCELLED":
		return messages.StatusCancelled
	default:
		return messages.StatusFailed
	}
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
