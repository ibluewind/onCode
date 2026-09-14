package testrun

import (
	"context"
)

// Request is a test tool request.
type Request struct {
	WorkspaceRoot string
	TimeoutMS     int
}

// Failure is one parsed test failure (MVP may be empty).
type Failure struct {
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
}

// Result is a structured test outcome (PHASE_01 §16).
type Result struct {
	Status     string    `json:"status"`
	ExitCode   int       `json:"exit_code"`
	DurationMS int64     `json:"duration_ms"`
	Total      int       `json:"total"`
	Passed     int       `json:"passed"`
	Failed     int       `json:"failed"`
	Skipped    int       `json:"skipped"`
	Failures   []Failure `json:"failures,omitempty"`
	StdoutSummary string `json:"stdout_summary,omitempty"`
	StderrSummary string `json:"stderr_summary,omitempty"`
	Tool       string    `json:"tool,omitempty"`
	Command    []string  `json:"command,omitempty"`
}

// Adapter detects and runs tests.
type Adapter interface {
	Name() string
	Detect(workspaceRoot string) bool
	Run(ctx context.Context, req Request) (*Result, error)
}
