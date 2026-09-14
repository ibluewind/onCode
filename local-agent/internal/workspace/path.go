package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	protoerr "oncode/protocol/errors"
)

// ResolvePath maps a workspace-relative path to an absolute path inside the workspace.
// It rejects empty paths, absolute escape attempts, and symlink escapes.
func (w *Workspace) ResolvePath(relPath string) (string, error) {
	if w == nil {
		return "", mustErr(protoerr.WorkspaceNotFound, "workspace is nil")
	}
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "", mustErr(protoerr.InvalidPath, "path is empty")
	}
	if filepath.IsAbs(relPath) {
		return "", mustErr(protoerr.PathOutsideWorkspace, "absolute paths are not allowed")
	}
	// Reject volume-relative and Windows drive-relative forms early.
	if vol := filepath.VolumeName(relPath); vol != "" {
		return "", mustErr(protoerr.PathOutsideWorkspace, "path contains a volume name")
	}
	cleaned := filepath.Clean(relPath)
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return "", mustErr(protoerr.InvalidPath, "path must refer to a file or subdirectory")
	}
	if isEscapeRel(cleaned) {
		return "", mustErr(protoerr.PathOutsideWorkspace, "path escapes workspace via ..")
	}

	joined := filepath.Join(w.RootPath, cleaned)
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", wrap(protoerr.InvalidPath, "abs path", err)
	}
	if !withinRoot(w.RootPath, absJoined) {
		return "", mustErr(protoerr.PathOutsideWorkspace, "path outside workspace before symlink eval")
	}

	resolved, hadSymlink, err := resolveExisting(absJoined)
	if err != nil {
		return "", wrap(protoerr.InvalidPath, "resolve path", err)
	}
	if !withinRoot(w.RootPath, resolved) {
		if hadSymlink {
			return "", mustErr(protoerr.SymlinkEscape, "symlink target outside workspace")
		}
		return "", mustErr(protoerr.PathOutsideWorkspace, "path outside workspace")
	}
	return resolved, nil
}

func isEscapeRel(cleaned string) bool {
	if cleaned == ".." {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(cleaned, ".."+sep)
}

func withinRoot(root, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	if rel == ".." {
		return false
	}
	sep := string(filepath.Separator)
	if strings.HasPrefix(rel, ".."+sep) {
		return false
	}
	return true
}

// resolveExisting evaluates symlinks for the deepest existing prefix, then appends
// the remaining non-existing segments without following them.
func resolveExisting(path string) (resolved string, hadSymlink bool, err error) {
	path = filepath.Clean(path)
	cur := path
	var missing []string
	for {
		fi, lerr := os.Lstat(cur)
		if lerr == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				hadSymlink = true
			}
			break
		}
		if !os.IsNotExist(lerr) {
			return "", false, lerr
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", false, fmt.Errorf("no existing ancestor for %s", path)
		}
		missing = append([]string{filepath.Base(cur)}, missing...)
		cur = parent
	}

	eval, err := filepath.EvalSymlinks(cur)
	if err != nil {
		return "", hadSymlink, err
	}
	// Detect symlink involvement on the existing prefix.
	if !hadSymlink {
		hadSymlink = pathHadSymlink(cur)
	}
	out := eval
	for _, seg := range missing {
		out = filepath.Join(out, seg)
	}
	return filepath.Clean(out), hadSymlink, nil
}

func pathHadSymlink(path string) bool {
	path = filepath.Clean(path)
	for {
		fi, err := os.Lstat(path)
		if err != nil {
			return false
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return true
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		path = parent
	}
}
