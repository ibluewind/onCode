package gitops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"oncode/local-agent/internal/process"
	protoerr "oncode/protocol/errors"
)

// StatusResult matches PHASE_01 §13 git.status.
type StatusResult struct {
	Branch    string   `json:"branch"`
	Head      string   `json:"head"`
	Changed   []string `json:"changed_files"`
	Staged    []string `json:"staged_files"`
	Untracked []string `json:"untracked_files"`
}

// DiffResult matches PHASE_01 §13 git.diff.
type DiffResult struct {
	Diff   string `json:"diff"`
	Staged bool   `json:"staged"`
	Path   string `json:"path,omitempty"`
}

// EnsureAvailable checks git binary and that root is a git work tree.
func EnsureAvailable(root string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return mustErr(protoerr.GitNotAvailable, "git executable not found")
	}
	gitDir := filepath.Join(root, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return mustErr(protoerr.GitNotAvailable, "workspace is not a git repository")
	}
	return nil
}

// Status runs git status --porcelain=v1 and resolves branch/HEAD.
func Status(ctx context.Context, root string) (*StatusResult, error) {
	if err := EnsureAvailable(root); err != nil {
		return nil, err
	}
	branch, err := gitOut(ctx, root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	head, err := gitOut(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		// empty repo with no commits
		head = ""
	}
	porcelain, err := gitOut(ctx, root, "status", "--porcelain=v1", "-uall")
	if err != nil {
		return nil, err
	}
	changed, staged, untracked := parsePorcelain(porcelain)
	return &StatusResult{
		Branch:    strings.TrimSpace(branch),
		Head:      strings.TrimSpace(head),
		Changed:   changed,
		Staged:    staged,
		Untracked: untracked,
	}, nil
}

// Diff runs git diff (or --cached). Optional path is workspace-relative.
func Diff(ctx context.Context, root, relPath string, staged bool) (*DiffResult, error) {
	if err := EnsureAvailable(root); err != nil {
		return nil, err
	}
	args := []string{"diff", "--no-color"}
	if staged {
		args = append(args, "--cached")
	}
	if relPath != "" {
		relPath = filepath.ToSlash(filepath.Clean(relPath))
		args = append(args, "--", relPath)
	}
	out, err := gitOut(ctx, root, args...)
	if err != nil {
		return nil, err
	}
	return &DiffResult{
		Diff:   out,
		Staged: staged,
		Path:   relPath,
	}, nil
}

func gitOut(ctx context.Context, root string, args ...string) (string, error) {
	res, err := process.Run(ctx, process.Options{
		Dir:     root,
		Path:    "git",
		Args:    args,
		Timeout: 60 * time.Second,
	})
	if err != nil {
		return "", wrap(err)
	}
	if res.TimedOut {
		return "", mustErr(protoerr.ProcessTimeout, "git timed out")
	}
	if res.Cancelled {
		return "", mustErr(protoerr.ProcessCancelled, "git cancelled")
	}
	if res.ExitCode != 0 {
		msg := strings.TrimSpace(res.Stderr)
		if msg == "" {
			msg = strings.TrimSpace(res.Stdout)
		}
		if msg == "" {
			msg = fmt.Sprintf("git exit %d", res.ExitCode)
		}
		return "", mustErr(protoerr.GitNotAvailable, msg)
	}
	return res.Stdout, nil
}

func parsePorcelain(out string) (changed, staged, untracked []string) {
	changed, staged, untracked = []string{}, []string{}, []string{}
	seenC, seenS, seenU := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if len(line) < 3 {
			continue
		}
		x, y := line[0], line[1]
		pathPart := line[2:]
		if strings.HasPrefix(pathPart, " ") {
			pathPart = pathPart[1:]
		}
		pathPart = strings.TrimSpace(pathPart)
		// rename/copy: "old -> new"
		path := pathPart
		if i := strings.Index(pathPart, " -> "); i >= 0 {
			path = pathPart[i+4:]
		}
		path = filepath.ToSlash(path)

		if x == '?' && y == '?' {
			if _, ok := seenU[path]; !ok {
				untracked = append(untracked, path)
				seenU[path] = struct{}{}
			}
			continue
		}
		if x != ' ' && x != '?' {
			if _, ok := seenS[path]; !ok {
				staged = append(staged, path)
				seenS[path] = struct{}{}
			}
		}
		if y != ' ' && y != '?' {
			if _, ok := seenC[path]; !ok {
				changed = append(changed, path)
				seenC[path] = struct{}{}
			}
		}
	}
	return changed, staged, untracked
}

func mustErr(code protoerr.Code, msg string) error {
	e, err := protoerr.New(code, msg)
	if err != nil {
		return err
	}
	return e
}

func wrap(err error) error {
	e, nerr := protoerr.New(protoerr.InternalError, err.Error())
	if nerr != nil {
		return err
	}
	return e
}
