package audit

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"time"
)

// Logger is a structured local audit logger (SPEC-13 P1 minimum).
// It must not log full file contents, credentials, or tokens.
type Logger struct {
	s *slog.Logger
}

func New(w io.Writer) *Logger {
	if w == nil {
		w = os.Stderr
	}
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	return &Logger{s: slog.New(h)}
}

type ToolEvent struct {
	CallID      string `json:"call_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Tool        string `json:"tool"`
	Status      string `json:"status"`
	DurationMS  int64  `json:"duration_ms"`
	ErrorCode   string `json:"error_code,omitempty"`
}

func (l *Logger) Tool(ev ToolEvent) {
	if l == nil || l.s == nil {
		return
	}
	attrs := []any{
		"event", "tool",
		"tool", ev.Tool,
		"status", ev.Status,
		"duration_ms", ev.DurationMS,
		"ts", time.Now().UTC().Format(time.RFC3339Nano),
	}
	if ev.CallID != "" {
		attrs = append(attrs, "call_id", ev.CallID)
	}
	if ev.WorkspaceID != "" {
		attrs = append(attrs, "workspace_id", ev.WorkspaceID)
	}
	if ev.ErrorCode != "" {
		attrs = append(attrs, "error_code", ev.ErrorCode)
	}
	l.s.Info("tool", attrs...)
}

func (l *Logger) Info(msg string, kv ...any) {
	if l == nil || l.s == nil {
		return
	}
	l.s.Info(msg, kv...)
}

// EncodeJSON is a helper for tests that need deterministic JSON lines.
func EncodeJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
