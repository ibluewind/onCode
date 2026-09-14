package diff_test

import (
	"strings"
	"testing"

	"oncode/local-agent/internal/diff"
)

func TestUnifiedFile_Modify(t *testing.T) {
	got := diff.UnifiedFile("a.txt", "one\ntwo\n", "one\nthree\n")
	if !strings.Contains(got, "--- a/a.txt\n") || !strings.Contains(got, "+++ b/a.txt\n") {
		t.Fatalf("headers: %q", got)
	}
	if !strings.Contains(got, "-two") || !strings.Contains(got, "+three") {
		t.Fatalf("body: %q", got)
	}
}

func TestUnifiedFile_CreateDelete(t *testing.T) {
	create := diff.UnifiedFile("n.txt", "", "hi\n")
	if !strings.Contains(create, "--- /dev/null\n") || !strings.Contains(create, "+hi") {
		t.Fatalf("%q", create)
	}
	del := diff.UnifiedFile("n.txt", "hi\n", "")
	if !strings.Contains(del, "+++ /dev/null\n") || !strings.Contains(del, "-hi") {
		t.Fatalf("%q", del)
	}
}
