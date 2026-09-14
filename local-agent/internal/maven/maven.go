package maven

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"oncode/local-agent/internal/build"
	"oncode/local-agent/internal/process"
	"oncode/local-agent/internal/testrun"
	protoerr "oncode/protocol/errors"
)

// Tool name advertised in capabilities.
const ToolName = "maven"

// Detect returns true when pom.xml exists under root.
func Detect(root string) bool {
	_, err := os.Stat(filepath.Join(root, "pom.xml"))
	return err == nil
}

// ResolveExecutable prefers the Maven wrapper, then mvn on PATH.
func ResolveExecutable(root string) (string, error) {
	if runtime.GOOS == "windows" {
		cmd := filepath.Join(root, "mvnw.cmd")
		if _, err := os.Stat(cmd); err == nil {
			return cmd, nil
		}
	}
	unix := filepath.Join(root, "mvnw")
	if _, err := os.Stat(unix); err == nil {
		return unix, nil
	}
	path, err := exec.LookPath("mvn")
	if err != nil {
		return "", mustErr(protoerr.BuildToolNotFound, "maven wrapper and mvn not found")
	}
	return path, nil
}

// BuildAdapter implements build.Adapter for Maven.
type BuildAdapter struct{}

func (BuildAdapter) Name() string { return ToolName }

func (BuildAdapter) Detect(root string) bool { return Detect(root) }

func (BuildAdapter) Run(ctx context.Context, req build.Request) (*build.Result, error) {
	exe, err := ResolveExecutable(req.WorkspaceRoot)
	if err != nil {
		return nil, err
	}
	args := []string{"-B", "-DskipTests", "compile"}
	timeout := durationMS(req.TimeoutMS, 10*time.Minute)
	pr, err := process.Run(ctx, process.Options{
		Dir:     req.WorkspaceRoot,
		Path:    exe,
		Args:    args,
		Timeout: timeout,
	})
	if err != nil {
		return nil, wrapInternal(err)
	}
	out := &build.Result{
		ExitCode:      pr.ExitCode,
		DurationMS:    pr.DurationMS,
		StdoutSummary: process.Summary(pr.Stdout, 4000),
		StderrSummary: process.Summary(pr.Stderr, 4000),
		Tool:          ToolName,
		Command:       append([]string{exe}, args...),
		Status:        statusOf(pr),
	}
	if out.Status == "FAILED" {
		out.Diagnostics = []string{"maven compile failed"}
	}
	return out, nil
}

// TestAdapter implements testrun.Adapter for Maven.
type TestAdapter struct{}

func (TestAdapter) Name() string { return ToolName }

func (TestAdapter) Detect(root string) bool { return Detect(root) }

func (TestAdapter) Run(ctx context.Context, req testrun.Request) (*testrun.Result, error) {
	exe, err := ResolveExecutable(req.WorkspaceRoot)
	if err != nil {
		return nil, err
	}
	args := []string{"-B", "test"}
	timeout := durationMS(req.TimeoutMS, 15*time.Minute)
	pr, err := process.Run(ctx, process.Options{
		Dir:     req.WorkspaceRoot,
		Path:    exe,
		Args:    args,
		Timeout: timeout,
	})
	if err != nil {
		return nil, wrapInternal(err)
	}
	total, failed, errors, skipped := parseSurefire(pr.Stdout + "\n" + pr.Stderr)
	passed := total - failed - errors - skipped
	if passed < 0 {
		passed = 0
	}
	out := &testrun.Result{
		ExitCode:      pr.ExitCode,
		DurationMS:    pr.DurationMS,
		Total:         total,
		Passed:        passed,
		Failed:        failed + errors,
		Skipped:       skipped,
		StdoutSummary: process.Summary(pr.Stdout, 4000),
		StderrSummary: process.Summary(pr.Stderr, 4000),
		Tool:          ToolName,
		Command:       append([]string{exe}, args...),
		Status:        statusOf(pr),
	}
	if out.Failed > 0 && out.Status == "SUCCESS" {
		out.Status = "FAILED"
	}
	return out, nil
}

var surefireRe = regexp.MustCompile(`Tests run:\s*(\d+),\s*Failures:\s*(\d+),\s*Errors:\s*(\d+),\s*Skipped:\s*(\d+)`)

func parseSurefire(log string) (total, failures, errors, skipped int) {
	m := surefireRe.FindStringSubmatch(log)
	if m == nil {
		return 0, 0, 0, 0
	}
	total, _ = strconv.Atoi(m[1])
	failures, _ = strconv.Atoi(m[2])
	errors, _ = strconv.Atoi(m[3])
	skipped, _ = strconv.Atoi(m[4])
	return
}

func statusOf(pr *process.Result) string {
	if pr.TimedOut {
		return "TIMEOUT"
	}
	if pr.Cancelled {
		return "CANCELLED"
	}
	if pr.ExitCode == 0 {
		return "SUCCESS"
	}
	return "FAILED"
}

func durationMS(ms int, def time.Duration) time.Duration {
	if ms <= 0 {
		return def
	}
	return time.Duration(ms) * time.Millisecond
}

func mustErr(code protoerr.Code, msg string) error {
	e, err := protoerr.New(code, msg)
	if err != nil {
		return err
	}
	return e
}

func wrapInternal(err error) error {
	e, nerr := protoerr.New(protoerr.InternalError, err.Error())
	if nerr != nil {
		return err
	}
	return e
}
