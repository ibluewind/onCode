package idebridge

import "testing"

func TestOriginAllowed(t *testing.T) {
	t.Parallel()
	allow := []string{"", "null", "vscode-file://vscode-app", "http://127.0.0.1", "http://localhost:1234"}
	for _, o := range allow {
		if !originAllowed(o) {
			t.Fatalf("expected allow origin %q", o)
		}
	}
	deny := []string{"https://evil.example", "http://192.168.1.5", "file://tmp"}
	for _, o := range deny {
		if originAllowed(o) {
			t.Fatalf("expected deny origin %q", o)
		}
	}
}
