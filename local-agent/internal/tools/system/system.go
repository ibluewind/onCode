package system

import (
	"context"
	"time"

	"oncode/local-agent/internal/config"
	"oncode/local-agent/internal/tools"
	"oncode/protocol/messages"
	"oncode/protocol/versioning"
)

const (
	ToolPing           = "system.ping"
	ToolGetCapabilities = "system.get_capabilities"
)

// PingResult matches PHASE_01 §10.1.
type PingResult struct {
	AgentID         string `json:"agent_id"`
	Version         string `json:"version"`
	ProtocolVersion string `json:"protocol_version"`
	Timestamp       string `json:"timestamp"`
}

type Deps struct {
	AgentID      string
	AgentVersion string
	ToolNames    func() []string
}

func Register(reg *tools.Registry, deps Deps) {
	reg.Register(ToolPing, func(ctx context.Context, req *messages.ToolRequestPayload) (*messages.ToolResponsePayload, error) {
		return tools.Success(req.CallID, PingResult{
			AgentID:         deps.AgentID,
			Version:         deps.AgentVersion,
			ProtocolVersion: versioning.Current,
			Timestamp:       time.Now().UTC().Format(time.RFC3339Nano),
		}), nil
	})

	reg.Register(ToolGetCapabilities, func(ctx context.Context, req *messages.ToolRequestPayload) (*messages.ToolResponsePayload, error) {
		names := []string{}
		if deps.ToolNames != nil {
			names = deps.ToolNames()
		}
		out := messages.CapabilitiesResult{
			AgentVersion:    deps.AgentVersion,
			ProtocolVersion: versioning.Current,
			Platform: messages.Platform{
				OS:   config.PlatformOS(),
				Arch: config.PlatformArch(),
			},
			Capabilities: messages.Capabilities{
				Workspace:    true,
				ProjectIndex: false,
				Build:        []string{"maven"},
				Test:         []string{"maven"},
				Git:          true,
				Languages:    []string{"java"},
			},
			SupportedTools: names,
		}
		return tools.Success(req.CallID, out), nil
	})
}
