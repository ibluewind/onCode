package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const DefaultMaxLogBytes = 256 * 1024

// Options for a safe local process run (no shell string concatenation).
type Options struct {
	Dir         string // absolute working directory (workspace root or approved subdir)
	Path        string // executable path or name (LookPath applied if not absolute)
	Args        []string
	Timeout     time.Duration
	MaxLogBytes int
}

// Result is a structured process outcome.
type Result struct {
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	TimedOut   bool   `json:"timed_out,omitempty"`
	Cancelled  bool   `json:"cancelled,omitempty"`
}

// Run starts a process with an argument array. Dir must exist and be absolute.
func Run(ctx context.Context, opts Options) (*Result, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("executable path required")
	}
	if opts.Dir == "" {
		return nil, fmt.Errorf("working directory required")
	}
	absDir, err := filepath.Abs(opts.Dir)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(absDir)
	if err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("working directory invalid: %s", absDir)
	}
	maxLog := opts.MaxLogBytes
	if maxLog <= 0 {
		maxLog = DefaultMaxLogBytes
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, opts.Path, opts.Args...)
	cmd.Dir = absDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitedBuffer{buf: &stdout, max: maxLog}
	cmd.Stderr = &limitedBuffer{buf: &stderr, max: maxLog}

	start := time.Now()
	err = cmd.Run()
	dur := time.Since(start).Milliseconds()

	res := &Result{
		DurationMS: dur,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}
	if runCtx.Err() != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			res.TimedOut = true
			res.ExitCode = -1
			if cmd.ProcessState != nil {
				res.ExitCode = cmd.ProcessState.ExitCode()
			}
			return res, nil
		}
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(runCtx.Err(), context.Canceled) {
			res.Cancelled = true
			res.ExitCode = -1
			return res, nil
		}
	}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		return nil, err
	}
	res.ExitCode = 0
	return res, nil
}

type limitedBuffer struct {
	buf *bytes.Buffer
	max int
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	remain := l.max - l.buf.Len()
	if remain <= 0 {
		return len(p), nil
	}
	if len(p) > remain {
		_, _ = l.buf.Write(p[:remain])
		return len(p), nil
	}
	return l.buf.Write(p)
}

// Summary truncates large logs for tool responses.
func Summary(s string, max int) string {
	if max <= 0 {
		max = 4000
	}
	if len(s) <= max {
		return s
	}
	half := max / 2
	return s[:half] + "\n...[truncated]...\n" + s[len(s)-half:]
}
