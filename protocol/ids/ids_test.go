package ids_test

import (
	"strings"
	"testing"

	"oncode/protocol/ids"
)

func TestRejectNonV7(t *testing.T) {
	err := ids.Validate(ids.Message, "MSG-018f0000-0000-4000-8000-000000000001")
	if err == nil {
		t.Fatal("uuid v4 must be rejected")
	}
}

func TestRejectWrongKind(t *testing.T) {
	id, err := ids.New(ids.Call)
	if err != nil {
		t.Fatal(err)
	}
	if err := ids.Validate(ids.Message, id); err == nil {
		t.Fatal("kind mismatch must fail")
	}
	if !strings.HasPrefix(id, "CALL-") {
		t.Fatalf("prefix: %s", id)
	}
}
