package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	protoerr "oncode/protocol/errors"
)

const HashPrefix = "sha256:"

// HashBytes returns sha256:<hex> for content.
func HashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return HashPrefix + hex.EncodeToString(sum[:])
}

// ReadFileResult is the successful output of workspace.read_file.
type ReadFileResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Hash    string `json:"hash"`
	Size    int64  `json:"size"`
}

// ReadTextFile reads a UTF-8 text file inside the workspace.
func (w *Workspace) ReadTextFile(relPath string, maxBytes int64) (*ReadFileResult, error) {
	abs, err := w.ResolvePath(relPath)
	if err != nil {
		return nil, err
	}
	fi, err := os.Lstat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, mustErr(protoerr.WorkspaceFileNotFound, "file not found")
		}
		return nil, wrap(protoerr.InternalError, "stat file", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return nil, wrap(protoerr.InvalidPath, "eval symlink", err)
		}
		if !withinRoot(w.RootPath, target) {
			return nil, mustErr(protoerr.SymlinkEscape, "symlink target outside workspace")
		}
		abs = target
		fi, err = os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, mustErr(protoerr.WorkspaceFileNotFound, "file not found")
			}
			return nil, wrap(protoerr.InternalError, "stat symlink target", err)
		}
	}
	if fi.IsDir() {
		return nil, mustErr(protoerr.InvalidPath, "path is a directory")
	}
	if maxBytes > 0 && fi.Size() > maxBytes {
		return nil, mustErr(protoerr.FileTooLarge, fmt.Sprintf("file size %d exceeds limit %d", fi.Size(), maxBytes))
	}

	f, err := os.Open(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, mustErr(protoerr.WorkspaceFileNotFound, "file not found")
		}
		return nil, wrap(protoerr.InternalError, "open file", err)
	}
	defer f.Close()

	var reader io.Reader = f
	if maxBytes > 0 {
		reader = io.LimitReader(f, maxBytes+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, wrap(protoerr.InternalError, "read file", err)
	}
	if maxBytes > 0 && int64(len(data)) > maxBytes {
		return nil, mustErr(protoerr.FileTooLarge, fmt.Sprintf("file exceeds limit %d", maxBytes))
	}
	if !utf8.Valid(data) {
		return nil, mustErr(protoerr.InvalidArgument, "file is not valid UTF-8 text")
	}

	return &ReadFileResult{
		Path:    filepath.ToSlash(filepath.Clean(relPath)),
		Content: string(data),
		Hash:    HashBytes(data),
		Size:    int64(len(data)),
	}, nil
}
