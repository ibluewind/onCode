package process_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"oncode/local-agent/internal/process"
)

func TestRun_EchoSuccess(t *testing.T) {
	dir := t.TempDir()
	var path string
	var args []string
	if runtime.GOOS == "windows" {
		path = "cmd"
		args = []string{"/C", "echo", "hi"}
	} else {
		path = "echo"
		args = []string{"hi"}
	}
	res, err := process.Run(context.Background(), process.Options{
		Dir:     dir,
		Path:    path,
		Args:    args,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || res.TimedOut {
		t.Fatalf("%+v", res)
	}
}

func TestRun_Timeout(t *testing.T) {
	dir := t.TempDir()
	var path string
	var args []string
	if runtime.GOOS == "windows" {
		path = "ping"
		args = []string{"127.0.0.1", "-n", "30"}
	} else {
		path = "sleep"
		args = []string{"30"}
	}
	if _, err := exec.LookPath(path); err != nil {
		t.Skip(err)
	}
	res, err := process.Run(context.Background(), process.Options{
		Dir:     dir,
		Path:    path,
		Args:    args,
		Timeout: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.TimedOut {
		t.Fatalf("expected timeout: %+v", res)
	}
	_ = filepath.Base(dir)
}
