package chatctx

import (
	"encoding/json"
	"fmt"
	"strings"
)

var forbiddenKeys = map[string]struct{}{
	"content":         {},
	"source":          {},
	"selected_text":   {},
	"selection_text":  {},
	"file_content":    {},
	"source_text":     {},
}

// ValidateSubmit는 chat.submit data에서 소스 본문 첨부를 거부한다 (SPEC-07 §14).
// 허용: message, work_item_id, ide_context.current_file, selection.start_line/end_line, cursor.
func ValidateSubmit(data map[string]any) error {
	if data == nil {
		return fmt.Errorf("chat.submit data is required")
	}
	msg, _ := data["message"].(string)
	if strings.TrimSpace(msg) == "" {
		return fmt.Errorf("message is required")
	}
	if err := rejectForbidden(data, ""); err != nil {
		return err
	}
	if ctx, ok := data["ide_context"].(map[string]any); ok {
		if file, ok := ctx["current_file"].(string); ok && strings.Contains(file, "\n") {
			return fmt.Errorf("current_file must be a path, not file contents")
		}
	}
	return nil
}

// ParseData는 chat.submit JSON object를 map으로 읽는다.
func ParseData(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty chat data")
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, ValidateSubmit(data)
}

func rejectForbidden(v any, path string) error {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			key := strings.ToLower(k)
			if _, bad := forbiddenKeys[key]; bad && key != "message" {
				return fmt.Errorf("source body field %q is not allowed", k)
			}
			if err := rejectForbidden(child, path+"."+k); err != nil {
				return err
			}
		}
	case []any:
		for i, child := range t {
			if err := rejectForbidden(child, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}
