package build

import (
	"context"
)

// Request is a build tool request.
type Request struct {
	WorkspaceRoot string
	TimeoutMS     int
}

// Result is a structured build outcome (PHASE_01 §15).
type Result struct {
	Status        string   `json:"status"` // SUCCESS | FAILED | TIMEOUT | CANCELLED
	ExitCode      int      `json:"exit_code"`
	DurationMS    int64    `json:"duration_ms"`
	StdoutSummary string   `json:"stdout_summary"`
	StderrSummary string   `json:"stderr_summary"`
	Diagnostics   []string `json:"diagnostics,omitempty"`
	Tool          string   `json:"tool,omitempty"` // e.g. maven
	Command       []string `json:"command,omitempty"`
}

// Adapter detects and runs a build system.
type Adapter interface {
	Name() string
	Detect(workspaceRoot string) bool
	Run(ctx context.Context, req Request) (*Result, error)
}
