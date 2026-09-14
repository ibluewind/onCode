package workspace

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	protoerr "oncode/protocol/errors"
)

// SearchMode selects which match kinds to run.
type SearchMode string

const (
	SearchModeAll     SearchMode = "all"
	SearchModeName    SearchMode = "name"
	SearchModePath    SearchMode = "path"
	SearchModeContent SearchMode = "content"
)

// SearchOptions configures MVP workspace.search.
type SearchOptions struct {
	Query          string
	Mode           SearchMode
	ExcludeDirs    []string
	MaxResults     int
	MaxFileBytes   int64
	MaxContentScan int64
}

// SearchMatch is one hit.
type SearchMatch struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"` // name | path | content
	Line    int    `json:"line,omitempty"`
	Preview string `json:"preview,omitempty"`
}

// SearchResult is the tool result for workspace.search.
type SearchResult struct {
	Query     string        `json:"query"`
	Mode      string        `json:"mode"`
	Matches   []SearchMatch `json:"matches"`
	Truncated bool          `json:"truncated"`
}

var defaultExcludeDirs = []string{".git", "target", "node_modules", "dist", "build"}

// DefaultSearchOptions returns PHASE_01 defaults.
func DefaultSearchOptions(query string, mode SearchMode) SearchOptions {
	if mode == "" {
		mode = SearchModeAll
	}
	return SearchOptions{
		Query:          query,
		Mode:           mode,
		ExcludeDirs:    append([]string(nil), defaultExcludeDirs...),
		MaxResults:     200,
		MaxFileBytes:   2 * 1024 * 1024,
		MaxContentScan: 512 * 1024,
	}
}

// Search walks the workspace for name, path substring, and/or literal text matches.
func (w *Workspace) Search(opts SearchOptions) (*SearchResult, error) {
	if w == nil {
		return nil, mustErr(protoerr.WorkspaceNotFound, "workspace is nil")
	}
	q := strings.TrimSpace(opts.Query)
	if q == "" {
		return nil, mustErr(protoerr.InvalidArgument, "query required")
	}
	mode := opts.Mode
	if mode == "" {
		mode = SearchModeAll
	}
	switch mode {
	case SearchModeAll, SearchModeName, SearchModePath, SearchModeContent:
	default:
		return nil, mustErr(protoerr.InvalidArgument, "invalid search mode")
	}
	if opts.MaxResults <= 0 {
		opts.MaxResults = 200
	}
	if opts.MaxContentScan <= 0 {
		opts.MaxContentScan = 512 * 1024
	}
	exclude := map[string]struct{}{}
	dirs := opts.ExcludeDirs
	if dirs == nil {
		dirs = defaultExcludeDirs
	}
	for _, d := range dirs {
		exclude[strings.ToLower(d)] = struct{}{}
	}

	out := &SearchResult{
		Query:   q,
		Mode:    string(mode),
		Matches: []SearchMatch{},
	}
	qLower := strings.ToLower(q)
	wantName := mode == SearchModeAll || mode == SearchModeName
	wantPath := mode == SearchModeAll || mode == SearchModePath
	wantContent := mode == SearchModeAll || mode == SearchModeContent

	err := filepath.WalkDir(w.RootPath, func(abs string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := exclude[strings.ToLower(d.Name())]; skip {
				return filepath.SkipDir
			}
			if abs != w.RootPath && !withinRoot(w.RootPath, abs) {
				return filepath.SkipDir
			}
			return nil
		}
		if out.Truncated {
			return filepath.SkipAll
		}

		rel, err := filepath.Rel(w.RootPath, abs)
		if err != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		if _, err := w.ResolvePath(relSlash); err != nil {
			return nil
		}

		baseLower := strings.ToLower(filepath.Base(rel))
		relLower := strings.ToLower(relSlash)

		if wantName && strings.Contains(baseLower, qLower) {
			if !out.add(SearchMatch{Path: relSlash, Kind: "name"}, opts.MaxResults) {
				return filepath.SkipAll
			}
		}
		if wantPath && strings.Contains(relLower, qLower) {
			if !out.add(SearchMatch{Path: relSlash, Kind: "path"}, opts.MaxResults) {
				return filepath.SkipAll
			}
		}
		if wantContent {
			if stop := searchFileContent(w, abs, relSlash, q, opts, out); stop {
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil {
		return nil, wrap(protoerr.InternalError, "search walk", err)
	}
	return out, nil
}

func (r *SearchResult) add(m SearchMatch, max int) bool {
	for _, existing := range r.Matches {
		if existing.Path == m.Path && existing.Kind == m.Kind && existing.Line == m.Line {
			return true
		}
	}
	if len(r.Matches) >= max {
		r.Truncated = true
		return false
	}
	r.Matches = append(r.Matches, m)
	return true
}

func searchFileContent(w *Workspace, abs, relSlash, query string, opts SearchOptions, out *SearchResult) (stop bool) {
	fi, err := os.Lstat(abs)
	if err != nil || fi.IsDir() {
		return false
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(abs)
		if err != nil || !withinRoot(w.RootPath, target) {
			return false
		}
		abs = target
		fi, err = os.Stat(abs)
		if err != nil || fi.IsDir() {
			return false
		}
	}
	if opts.MaxFileBytes > 0 && fi.Size() > opts.MaxFileBytes {
		return false
	}

	f, err := os.Open(abs)
	if err != nil {
		return false
	}
	defer f.Close()

	limited := io.LimitReader(f, opts.MaxContentScan)
	head := make([]byte, 4096)
	n, _ := io.ReadFull(limited, head)
	head = head[:n]
	if n == 0 {
		return false
	}
	if !utf8.Valid(head) || bytes.IndexByte(head, 0) >= 0 {
		return false
	}

	reader := io.MultiReader(bytes.NewReader(head), limited)
	sc := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if !strings.Contains(line, query) {
			continue
		}
		preview := line
		if len(preview) > 200 {
			preview = preview[:200]
		}
		if !out.add(SearchMatch{
			Path:    relSlash,
			Kind:    "content",
			Line:    lineNo,
			Preview: preview,
		}, opts.MaxResults) {
			return true
		}
	}
	return false
}
