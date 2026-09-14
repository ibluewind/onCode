package workspace

import (
	protoerr "oncode/protocol/errors"
)

// ReadFileEntry is one path outcome for workspace.read_files.
// Paths are processed in request order. One failure does not stop other reads.
type ReadFileEntry struct {
	Path    string          `json:"path"`
	Status  string          `json:"status"` // SUCCESS | FAILED
	Content string          `json:"content,omitempty"`
	Hash    string          `json:"hash,omitempty"`
	Size    int64           `json:"size,omitempty"`
	Error   *protoerr.Error `json:"error,omitempty"`
}

// ReadFilesResult is the tool result for workspace.read_files.
type ReadFilesResult struct {
	Files   []ReadFileEntry `json:"files"`
	Total   int             `json:"total"`
	Success int             `json:"success"`
	Failed  int             `json:"failed"`
}

// ReadTextFiles reads each path independently in order.
func (w *Workspace) ReadTextFiles(paths []string, maxBytes int64) (*ReadFilesResult, error) {
	if w == nil {
		return nil, mustErr(protoerr.WorkspaceNotFound, "workspace is nil")
	}
	if len(paths) == 0 {
		return nil, mustErr(protoerr.InvalidArgument, "paths required")
	}
	out := &ReadFilesResult{
		Files: make([]ReadFileEntry, 0, len(paths)),
		Total: len(paths),
	}
	for _, p := range paths {
		entry := ReadFileEntry{Path: p, Status: "FAILED"}
		if p == "" {
			entry.Error = mustProtoErr(protoerr.InvalidArgument, "path is empty")
			out.Failed++
			out.Files = append(out.Files, entry)
			continue
		}
		res, err := w.ReadTextFile(p, maxBytes)
		if err != nil {
			entry.Error = asProtoErr(err)
			out.Failed++
			out.Files = append(out.Files, entry)
			continue
		}
		entry.Status = "SUCCESS"
		entry.Path = res.Path
		entry.Content = res.Content
		entry.Hash = res.Hash
		entry.Size = res.Size
		out.Success++
		out.Files = append(out.Files, entry)
	}
	return out, nil
}

func asProtoErr(err error) *protoerr.Error {
	if pe, ok := err.(*protoerr.Error); ok {
		return pe
	}
	e, nerr := protoerr.New(protoerr.InternalError, err.Error())
	if nerr != nil {
		return &protoerr.Error{Code: protoerr.InternalError, Message: err.Error()}
	}
	return e
}

func mustProtoErr(code protoerr.Code, msg string) *protoerr.Error {
	e, err := protoerr.New(code, msg)
	if err != nil {
		return &protoerr.Error{Code: code, Message: msg}
	}
	return e
}

// OverallStatus maps aggregate read_files outcome to tool response status.
func (r *ReadFilesResult) OverallStatus() string {
	if r == nil || r.Total == 0 {
		return "FAILED"
	}
	switch {
	case r.Failed == 0:
		return "SUCCESS"
	case r.Success == 0:
		return "FAILED"
	default:
		return "PARTIAL"
	}
}
