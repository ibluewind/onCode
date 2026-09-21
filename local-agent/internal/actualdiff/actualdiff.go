package actualdiff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"oncode/local-agent/internal/diff"
)

const hashPrefix = "sha256:"

// Change는 ProposedChanges의 한 파일이다. 모델 unified diff가 아니다.
type Change struct {
	Path       string
	Operation  string
	Content    string
	TargetPath string
}

// Result는 Local Agent가 만든 actual unified diff와 해시이다.
type Result struct {
	ChangeSetID string
	WorkspaceID string
	Diff        string
	DiffHash    string
}

// FromProposed는 현재 파일 내용(없으면 빈 문자열)과 제안 내용을 unified diff로 만든다.
// oldContent는 워크스페이스 읽기. nil이면 모든 경로를 빈 파일로 본다. apply는 하지 않는다.
func FromProposed(changeSetID, workspaceID string, changes []Change, oldContent func(path string) string) (Result, error) {
	if changeSetID == "" {
		return Result{}, fmt.Errorf("change_set_id is required")
	}
	if oldContent == nil {
		oldContent = func(string) string { return "" }
	}
	type part struct {
		path string
		text string
	}
	parts := make([]part, 0, len(changes)+1)
	for _, ch := range changes {
		path := strings.TrimPrefix(ch.Path, "/")
		if path == "" {
			continue
		}
		op := strings.ToUpper(ch.Operation)
		old := oldContent(path)
		switch op {
		case "CREATE":
			parts = append(parts, part{path: path, text: diff.UnifiedFile(path, "", ch.Content)})
		case "DELETE":
			parts = append(parts, part{path: path, text: diff.UnifiedFile(path, old, "")})
		case "RENAME":
			target := strings.TrimPrefix(ch.TargetPath, "/")
			newContent := ch.Content
			if newContent == "" {
				newContent = old
			}
			parts = append(parts, part{path: path, text: diff.UnifiedFile(path, old, "")})
			if target != "" {
				parts = append(parts, part{path: target, text: diff.UnifiedFile(target, "", newContent)})
			}
		default: // MODIFY and unknown treated as modify
			parts = append(parts, part{path: path, text: diff.UnifiedFile(path, old, ch.Content)})
		}
	}
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].path < parts[j].path })
	texts := make([]string, 0, len(parts))
	for _, p := range parts {
		texts = append(texts, p.text)
	}
	diffText := diff.UnifiedJoin(texts)
	return Result{
		ChangeSetID: changeSetID,
		WorkspaceID: workspaceID,
		Diff:        diffText,
		DiffHash:    hashBytes([]byte(diffText)),
	}, nil
}

// ParseProposedJSON은 서버 하니스 ProposedChanges JSON을 읽는다. unified diff 필드는 무시한다.
func ParseProposedJSON(raw string) (changeSetID string, changes []Change, err error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil, fmt.Errorf("proposed changes json is empty")
	}
	var payload struct {
		ChangeSetID string `json:"change_set_id"`
		Changes     []struct {
			Path       string `json:"path"`
			Operation  string `json:"operation"`
			Content    string `json:"content"`
			TargetPath string `json:"target_path"`
		} `json:"changes"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("proposed changes json: %w", err)
	}
	out := make([]Change, 0, len(payload.Changes))
	for _, c := range payload.Changes {
		out = append(out, Change{
			Path:       c.Path,
			Operation:  c.Operation,
			Content:    c.Content,
			TargetPath: c.TargetPath,
		})
	}
	return payload.ChangeSetID, out, nil
}

func hashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hashPrefix + hex.EncodeToString(sum[:])
}
