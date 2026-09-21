package chatctx

import "testing"

func TestValidateSubmit_RejectsSource(t *testing.T) {
	err := ValidateSubmit(map[string]any{
		"message": "fix this",
		"ide_context": map[string]any{
			"current_file": "A.java",
			"content":      "class A {}",
		},
	})
	if err == nil {
		t.Fatal("expected source body rejected")
	}
}

func TestValidateSubmit_AllowsPathAndRange(t *testing.T) {
	err := ValidateSubmit(map[string]any{
		"message": "add null check",
		"ide_context": map[string]any{
			"current_file": "src/UserService.java",
			"selection": map[string]any{
				"start_line": 42.0,
				"end_line":   67.0,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateSubmit_RequiresMessage(t *testing.T) {
	if err := ValidateSubmit(map[string]any{}); err == nil {
		t.Fatal("expected empty message rejected")
	}
}
