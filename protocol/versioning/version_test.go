package versioning_test

import (
	"testing"

	"oncode/protocol/versioning"
)

func TestCurrent(t *testing.T) {
	if versioning.Current != "oncode-tool/1.0" {
		t.Fatalf("got %s", versioning.Current)
	}
}
